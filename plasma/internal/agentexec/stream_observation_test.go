package agentexec

import (
	"strconv"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

func TestCodexObservationNormalizesSafeEvents(t *testing.T) {
	var events []AgentObservation
	observer := func(event AgentObservation) { events = append(events, event) }

	observeCodexJSONLine(`{"type":"turn.started"}`, observer)
	observeCodexJSONLine(`{"type":"item.started","item":{"type":"mcp_tool_call","tool":"plasma.sources.candidates.propose","arguments":{"path":"/path/to/file.md"}}}`, observer)
	observeCodexJSONLine(`{"type":"item.completed","item":{"type":"agent_message","text":"초안입니다. https://example.com 과 /path/to/file.md 참고"}}`, observer)

	if len(events) != 3 {
		t.Fatalf("expected 3 safe observations, got %#v", events)
	}
	if events[0].Type != AgentObservationPhase || events[0].Phase != AgentPhaseThinking {
		t.Fatalf("unexpected phase event: %#v", events[0])
	}
	if events[1].Type != AgentObservationTool || events[1].ToolCategory != AgentToolCategorySourcePropose {
		t.Fatalf("unexpected tool event: %#v", events[1])
	}
	if events[2].Type != AgentObservationAnswer {
		t.Fatalf("unexpected answer event: %#v", events[2])
	}
	for _, forbidden := range []string{"https://example.com", "/path/to", "plasma.sources.candidates.propose"} {
		if strings.Contains(events[2].Text, forbidden) {
			t.Fatalf("answer preview leaked %q: %q", forbidden, events[2].Text)
		}
	}
}

func TestClaudeObservationUsesPartialTextAndIgnoresUnknown(t *testing.T) {
	var events []AgentObservation
	stream := &claudeStreamObserver{observer: func(event AgentObservation) { events = append(events, event) }}

	stream.observe(`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"raw thinking"}}}`)
	stream.observe(`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","name":"mcp__plasma__plasma_research_read"}}}`)
	stream.observe(`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"첫 문장 "}}}`)
	stream.observe(`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"https://example.com "}}}`)
	stream.observe(`{"type":"stream_event","event":{"type":"unknown_event","raw":"ignored"}}`)

	if len(events) != 4 {
		t.Fatalf("expected 4 safe observations, got %#v", events)
	}
	if events[0].Type != AgentObservationPhase || events[0].Phase != AgentPhaseThinking {
		t.Fatalf("unexpected thinking event: %#v", events[0])
	}
	if events[1].Type != AgentObservationTool || events[1].ToolCategory != AgentToolCategoryMissionRead {
		t.Fatalf("unexpected tool event: %#v", events[1])
	}
	if events[2].Type != AgentObservationAnswer || events[2].Text != "첫 문장" {
		t.Fatalf("unexpected first preview: %#v", events[2])
	}
	if events[3].Type != AgentObservationAnswer || strings.Contains(events[3].Text, "https://example.com") {
		t.Fatalf("unexpected sanitized preview: %#v", events[3])
	}
}

func TestProviderObservationPreservesAnswerToolAnswerOrder(t *testing.T) {
	var codex []AgentObservation
	observeCodexJSONLine(`{"type":"item.completed","item":{"type":"agent_message","text":"첫 답변 "}}`, func(event AgentObservation) {
		codex = append(codex, event)
	})
	observeCodexJSONLine(`{"type":"item.started","item":{"type":"mcp_tool_call","tool":"WebSearch"}}`, func(event AgentObservation) {
		codex = append(codex, event)
	})
	observeCodexJSONLine(`{"type":"item.completed","item":{"type":"agent_message","text":"둘째 답변 "}}`, func(event AgentObservation) {
		codex = append(codex, event)
	})
	assertObservationOrder(t, codex, []AgentObservationType{AgentObservationAnswer, AgentObservationTool, AgentObservationAnswer})

	var claude []AgentObservation
	stream := &claudeStreamObserver{observer: func(event AgentObservation) { claude = append(claude, event) }}
	observeClaudeTextDelta(t, stream, "첫답변")
	stream.observe(`{"type":"stream_event","event":{"type":"content_block_stop"}}`)
	stream.observe(`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","name":"WebFetch"}}}`)
	observeClaudeTextDelta(t, stream, " 둘째 답변 ")
	assertObservationOrder(t, claude, []AgentObservationType{AgentObservationAnswer, AgentObservationTool, AgentObservationAnswer})
}

func TestSafeAnswerPreviewRedactsSensitiveTokens(t *testing.T) {
	for _, tc := range []struct {
		name      string
		raw       string
		forbidden []string
	}{
		{name: "file uri", raw: `열기 file:///path/to/private.md`, forbidden: []string{"file://", "/path/to"}},
		{name: "opaque uri", raw: `토큰 custom+v1:opaque-token`, forbidden: []string{"custom+v1:", "opaque-token"}},
		{name: "quoted absolute path", raw: `{"path":"/path/to/private.md"}`, forbidden: []string{`"/path/to`, "/private.md"}},
		{name: "quoted paths with spaces", raw: `{"abs":"/path/to/My Documents/private.md","rel":"../My Documents/private.md"}`, forbidden: []string{"/path/to/My Documents", "../My Documents", "private.md"}},
		{name: "punctuated paths", raw: "(`./private/a.md`, `/tmp/b.md`)", forbidden: []string{"./private", "/tmp/b.md"}},
		{name: "path punctuation prefixes", raw: `;/path/to/a.md !/tmp/b.md ?Documents/private.md |C:\path\to\private.md`, forbidden: []string{"/path/to", "/tmp/b.md", "Documents/private.md", `C:\path`}},
		{name: "single segment absolute path", raw: `열기 /tmp`, forbidden: []string{"/tmp"}},
		{name: "slash and backslash tokens", raw: `Documents/private.md Documents\private.md C:\path\to\private.md C:/path/to/private.md`, forbidden: []string{"Documents/private.md", `Documents\private.md`, `C:\path`, "C:/path"}},
		{name: "markdown relative path", raw: `참고 [문서](../private/source.md)`, forbidden: []string{"../private", "source.md"}},
		{name: "uuid v7", raw: `id 01890f3a-7cc2-7bbd-98c4-2f56a5f1d222`, forbidden: []string{"01890f3a-7cc2-7bbd-98c4-2f56a5f1d222"}},
		{name: "prefixed ids", raw: `evt_abcdef123456 run_abcdef123456 art_abcdef123456 source_abcdef123456`, forbidden: []string{"evt_abcdef", "run_abcdef", "art_abcdef", "source_abcdef"}},
		{name: "labeled ids", raw: `session_id: ses_abcdef123456 request=abcdef1234567890`, forbidden: []string{"ses_abcdef", "abcdef1234567890"}},
	} {
		got := safeAnswerPreview(tc.raw)
		for _, forbidden := range tc.forbidden {
			if strings.Contains(got, forbidden) {
				t.Fatalf("%s leaked %q in %q", tc.name, forbidden, got)
			}
		}
	}
	for _, prefix := range []string{"toolu", "tool", "thread", "msg", "message", "resp", "response", "job", "token"} {
		raw := prefix + "_abcdef123456"
		if got := safeAnswerPreview(raw); strings.Contains(got, raw) {
			t.Fatalf("opaque prefix %q leaked in %q", prefix, got)
		}
	}
	for _, label := range []string{"tool", "tool_use", "toolu", "thread", "message", "msg", "response", "resp", "job", "token"} {
		for _, suffix := range []string{"", "_id", "-id", " id"} {
			for _, sep := range []string{":", "="} {
				rawValue := "rawvalue123456"
				got := safeAnswerPreview(label + suffix + sep + rawValue)
				if strings.Contains(got, rawValue) {
					t.Fatalf("labeled id %q leaked in %q", label+suffix+sep, got)
				}
			}
		}
	}
}

func TestClaudeObservationBuffersSplitSensitiveDeltas(t *testing.T) {
	var answers []string
	stream := &claudeStreamObserver{observer: func(event AgentObservation) {
		if event.Type == AgentObservationAnswer {
			answers = append(answers, event.Text)
		}
	}}
	observeClaudeTextDelta(t, stream, "참고 file://")
	observeClaudeTextDelta(t, stream, "/tmp/private ")
	observeClaudeTextDelta(t, stream, "session_id: ses_abc")
	observeClaudeTextDelta(t, stream, "def123 ")
	if len(answers) < 3 {
		t.Fatalf("expected stable whitespace-boundary previews, got %#v", answers)
	}
	for _, answer := range answers {
		for _, forbidden := range []string{"file://", "/tmp/private", "ses_abc", "ses_abcdef123"} {
			if strings.Contains(answer, forbidden) {
				t.Fatalf("split delta leaked %q in %#v", forbidden, answers)
			}
		}
	}
	if !strings.Contains(answers[len(answers)-1], "[ID]") {
		t.Fatalf("expected completed opaque id to be masked in final preview: %#v", answers)
	}
}

func TestClaudeObservationBuffersEverySplitOfSensitiveToken(t *testing.T) {
	for _, tc := range []struct {
		name      string
		raw       string
		forbidden []string
	}{
		{name: "file uri", raw: "see file:///path/to/private.md done", forbidden: []string{"file://", "/path/to", "private.md"}},
		{name: "path token", raw: "see Documents/private.md done", forbidden: []string{"Documents/private.md"}},
		{name: "opaque id", raw: "see toolu_abcdef123456 done", forbidden: []string{"toolu_abcdef123456"}},
	} {
		for split := 1; split < len(tc.raw); split++ {
			var answers []string
			stream := &claudeStreamObserver{observer: func(event AgentObservation) {
				if event.Type == AgentObservationAnswer {
					answers = append(answers, event.Text)
				}
			}}
			observeClaudeTextDelta(t, stream, tc.raw[:split])
			observeClaudeTextDelta(t, stream, tc.raw[split:])
			for _, answer := range answers {
				for _, forbidden := range tc.forbidden {
					if strings.Contains(answer, forbidden) {
						t.Fatalf("%s split %d leaked %q in %#v", tc.name, split, forbidden, answers)
					}
				}
			}
		}
	}
}

func TestClaudeObservationFlushesSafePreviewOnContentBlockStop(t *testing.T) {
	var answers []string
	stream := &claudeStreamObserver{observer: func(event AgentObservation) {
		if event.Type == AgentObservationAnswer {
			answers = append(answers, event.Text)
		}
	}}
	observeClaudeTextDelta(t, stream, "확인했습니다.")
	if len(answers) != 0 {
		t.Fatalf("text_delta without whitespace must be withheld, got %#v", answers)
	}
	stream.observe(`{"type":"stream_event","event":{"type":"content_block_stop"}}`)
	if len(answers) != 1 || answers[0] != "확인했습니다." {
		t.Fatalf("content_block_stop did not flush no-whitespace answer: %#v", answers)
	}

	answers = nil
	stream = &claudeStreamObserver{observer: func(event AgentObservation) {
		if event.Type == AgentObservationAnswer {
			answers = append(answers, event.Text)
		}
	}}
	observeClaudeTextDelta(t, stream, "답변 ")
	if len(answers) != 1 || answers[0] != "답변" {
		t.Fatalf("expected whitespace-boundary preview before trailing word, got %#v", answers)
	}
	observeClaudeTextDelta(t, stream, "완료")
	if len(answers) != 1 {
		t.Fatalf("trailing word must stay withheld before stop, got %#v", answers)
	}
	stream.observe(`{"type":"stream_event","event":{"type":"content_block_stop"}}`)
	if len(answers) != 2 || answers[1] != "답변 완료" {
		t.Fatalf("content_block_stop did not flush trailing word: %#v", answers)
	}
}

func observeClaudeTextDelta(t *testing.T, stream *claudeStreamObserver, text string) {
	t.Helper()
	stream.observe(`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":` + strconv.Quote(text) + `}}}`)
}

func TestToolCategoryFromNameUsesClosedProviderGroups(t *testing.T) {
	for _, tc := range []struct {
		name string
		want AgentToolCategory
	}{
		{name: "WebSearch", want: AgentToolCategoryWebSearch},
		{name: "web_fetch", want: AgentToolCategoryWebRead},
		{name: "mcp__plasma__plasma_sources_read", want: AgentToolCategoryMissionRead},
		{name: "plasma.research.grep", want: AgentToolCategoryMissionRead},
		{name: "plasma.sources.candidates.propose", want: AgentToolCategorySourcePropose},
		{name: "plasma.evidence.propose", want: AgentToolCategoryOrganize},
		{name: "plasma.questions.propose", want: AgentToolCategoryOrganize},
		{name: "plasma.claims.confidence.update", want: AgentToolCategoryOrganize},
		{name: "plasma.proposals.submit", want: AgentToolCategoryOrganize},
		{name: "plasma.mermaid.validate", want: AgentToolCategoryValidate},
		{name: "unknown.future.tool", want: AgentToolCategoryUnknown},
	} {
		if got := toolCategoryFromName(tc.name); got != tc.want {
			t.Fatalf("toolCategoryFromName(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestParseClaudeJSONOutputAcceptsJSONLStream(t *testing.T) {
	raw := []byte(`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"partial"}}}
{"type":"result","session_id":"55555555-5555-4555-8555-555555555555","result":"final answer","usage":{"input_tokens":3,"output_tokens":2}}`)
	usage := agentusage.New("claude", "claude", "haiku", "", "prompt")

	result, err := parseClaudeJSONOutput(raw, usage)
	if err != nil {
		t.Fatalf("parseClaudeJSONOutput returned error: %v", err)
	}
	if result.Text != "final answer" {
		t.Fatalf("expected final result text, got %q", result.Text)
	}
	if result.SessionID != "55555555-5555-4555-8555-555555555555" {
		t.Fatalf("unexpected session id %q", result.SessionID)
	}
	if result.Usage.ProviderUsage == nil || result.Usage.ProviderUsage.OutputTokens != 2 {
		t.Fatalf("expected usage from final result, got %#v", result.Usage.ProviderUsage)
	}
}

func TestClaudeStreamArgsUsePartialVerboseStreamOnly(t *testing.T) {
	base := (ClaudeExecutor{}).baseArgsForRequest(AgentRequest{Model: "haiku"})
	if strings.Contains(strings.Join(base, " "), "stream-json") || indexOfArg(base, "--include-partial-messages") >= 0 || indexOfArg(base, "--verbose") >= 0 {
		t.Fatalf("base args must not use streaming flags: %#v", base)
	}
	stream := claudeStreamArgs(base)
	if argValueAfter(stream, "--output-format") != "stream-json" {
		t.Fatalf("expected stream-json output format, got %#v", stream)
	}
	if indexOfArg(stream, "--include-partial-messages") < 0 || indexOfArg(stream, "--verbose") < 0 {
		t.Fatalf("expected Claude partial verbose stream flags, got %#v", stream)
	}
}

func TestClaudeEphemeralSessionIsOptIn(t *testing.T) {
	args := (ClaudeExecutor{}).baseArgsForRequest(AgentRequest{Model: "haiku", EphemeralSession: true})
	if indexOfArg(args, "--no-session-persistence") < 0 {
		t.Fatalf("ephemeral args missing --no-session-persistence: %#v", args)
	}
	defaultArgs := (ClaudeExecutor{}).baseArgsForRequest(AgentRequest{Model: "haiku"})
	if indexOfArg(defaultArgs, "--no-session-persistence") >= 0 {
		t.Fatalf("default args unexpectedly disabled session persistence: %#v", defaultArgs)
	}
}

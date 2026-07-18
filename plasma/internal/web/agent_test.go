package web

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestCodexEnvironmentUsesAllowlist(t *testing.T) {
	t.Setenv("PATH", "/bin")
	t.Setenv("PLASMA_RUNTIME_MODE", "dev")
	t.Setenv("OPENAI_API_KEY", "should-not-be-inherited")

	env := codexEnvironment(nil)
	if !containsEnv(env, "PATH=/opt/homebrew/bin:/usr/local/bin:/bin:/usr/bin:/usr/sbin:/sbin") {
		t.Fatalf("expected PATH to be retained in %#v", env)
	}
	if !containsEnv(env, "PLASMA_RUNTIME_MODE=dev") {
		t.Fatalf("expected PLASMA_RUNTIME_MODE to be retained in %#v", env)
	}
	for _, value := range env {
		if strings.HasPrefix(value, "OPENAI_API_KEY=") {
			t.Fatalf("expected OPENAI_API_KEY to be scrubbed from %#v", env)
		}
	}
}

func TestAgentPromptAutoUsesC1ReadLoopWithoutLegacyMutations(t *testing.T) {
	recall := recallPreview{
		Mission: recallMission{
			MissionID: "mis_1",
			Title:     "조사 미션",
			Objective: "근거 기반 조사",
		},
	}
	prompt := agentPrompt("조사해줘", recall, "auto", false, "ses_1", selectControllerStrategy("", "조사해줘", recall, false))
	for _, expected := range []string{
		"plasma.research.outline",
		"plasma.research.list",
		"plasma.research.grep",
		"plasma.research.read",
		"plasma.research.references",
		"plasma.sources.read",
		"plasma.sources.search",
		"plasma.sources.candidates.propose",
		"copy source_uri into url and title into title",
		"Confluence pages",
		"live_reference local_path",
		"source.observed",
		"observation_event_id",
		"do not paste local file content into prompts",
		"continue with next_offset",
		"do not treat the first chunk of a long source as the whole source",
		"Your answer is a result, not a source",
		"소스 후보:",
		"채택 의견:",
		"Source candidates are not sources",
		"C1 boundary",
		"Controller strategy",
		"v2",
		"Stay close to the user's latest request",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected prompt to contain %q:\n%s", expected, prompt)
		}
	}
	for _, forbidden := range []string{
		"plasma.evidence.propose",
		"plasma.claims.propose",
		"plasma.claims.confidence.update",
		"plasma.proposals.submit",
		"Source candidate URL discipline",
		"Create review proposals",
		"evidence must cite a source snapshot/artifact",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("default C1 prompt contains legacy instruction %q:\n%s", forbidden, prompt)
		}
	}
}

func TestControllerStrategySelection(t *testing.T) {
	recall := recallPreview{
		Mission: recallMission{
			MissionID: "mis_1",
			Title:     "조사 미션",
			Objective: "근거 기반 조사",
		},
	}
	narrow := selectControllerStrategy("", "핵심만 정리해줘", recall, false)
	if narrow.ID != controllerStrategyV2 {
		t.Fatalf("expected conservative default, got %#v", narrow)
	}
	broad := selectControllerStrategy("", "반대 관점과 대안을 넓게 비교해줘", recall, false)
	if broad.ID != controllerStrategyV3 {
		t.Fatalf("expected broadening strategy, got %#v", broad)
	}
	override := selectControllerStrategy("v3", "핵심만 정리해줘", recall, false)
	if override.ID != controllerStrategyV3 {
		t.Fatalf("expected explicit override, got %#v", override)
	}
}

func TestCodexExecutorCreatesMissingWorkDir(t *testing.T) {
	dir := t.TempDir()
	workDir := filepath.Join(dir, "missing-workdir")
	command := filepath.Join(dir, "fake-codex")
	script := `#!/bin/sh
out=""
want_out=0
for arg in "$@"; do
  if [ "$want_out" = "1" ]; then
    out="$arg"
    want_out=0
  elif [ "$arg" = "--output-last-message" ]; then
    want_out=1
  fi
done
cat >/dev/null
printf 'session id: created-workdir-session\n'
printf 'done' > "$out"
`
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	result, err := (CodexExecutor{
		Command: command,
		WorkDir: workDir,
		Timeout: 2 * time.Second,
		Env:     []string{"PATH=/usr/bin:/bin"},
	}).Run(context.Background(), AgentRequest{
		Prompt:        "test prompt",
		MissionID:     "mis_1",
		ToolSessionID: "ses_1",
		AgentExecutor: "codex",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionID != "created-workdir-session" {
		t.Fatalf("unexpected session id %q", result.SessionID)
	}
	if info, err := os.Stat(workDir); err != nil || !info.IsDir() {
		t.Fatalf("expected workdir to be created, info=%#v err=%v", info, err)
	}
}

func TestAgentProposalPromptAsksForMissingEvidenceSlate(t *testing.T) {
	recall := recallPreview{
		Mission: recallMission{
			MissionID: "mis_1",
			Title:     "조사 미션",
			Objective: "풍부한 근거 추출",
		},
	}
	prompt := agentProposalPrompt(recall, "source-backed answer", "ses_1")
	for _, expected := range []string{
		"add missing non-duplicate proposals",
		"plasma.research.outline",
		"plasma.research.read",
		"If a read is truncated, continue with next_offset",
		"Build a useful evidence slate, not only a minimal proof set.",
		"distinct source-backed facts",
		"reactions, rumors or unconfirmed circulating claims, controversies, market signals",
		"Do not invent evidence and do not split duplicates just to increase count.",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected proposal prompt to contain %q:\n%s", expected, prompt)
		}
	}
}

func TestAgentReportPromptUsesResearchToolsWithoutRecallPayload(t *testing.T) {
	planPrompt := agentReportPlanPrompt("Report", "mis_1", "ses_1", "evt_pending", "key_1", reportRigorProfiles["strict"])
	plan := agentReportPlan{
		Summary: "Use source-backed material.",
		Sections: []agentReportSection{{
			Title:   "Section",
			Purpose: "Cover evidence.",
			TargetRefs: app.ReportBlockSourceRefs{
				EvidenceIDs: []string{"evd_1"},
			},
		}},
	}
	prompt := agentReportPrompt("Report", "mis_1", "ses_1", reportRigorProfiles["strict"], plan)
	for _, expected := range []string{
		"plasma.research.outline",
		"plasma.research.list",
		"plasma.research.grep",
		"plasma.research.read",
		"plasma.research.references",
		"mission_id mis_1",
		`producer {"type":"agent_session","id":"ses_1"}`,
		"live local_path observations",
		"observation_event_id",
		"User-visible generation plan",
		"visible footnotes",
		"final AST refs must only contain approved claim_ids and approved evidence_ids",
		"proposed, pending, or rejected material",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected report prompt to contain %q:\n%s", expected, prompt)
		}
	}
	if strings.Contains(planPrompt, "supplied by the tool context") || strings.Contains(planPrompt, "exactly once") {
		t.Fatalf("planned prompt retained false binding or retry wording:\n%s", planPrompt)
	}
	for _, expected := range []string{
		"Create a user-visible Korean report generation plan",
		"Do not write the article yet",
		"plasma.research.outline",
		"plasma.research.read",
		"mission_id mis_1",
		"source.observed",
		"observation_event_id",
		"target_refs should name only approved records",
		"pending_event_id evt_pending",
		"report_mode planned",
		"idempotency_key key_1",
		`producer {"type":"agent_session","id":"ses_1"}`,
		"at most three parsed submission calls total",
		"including a success or replay",
	} {
		if !strings.Contains(planPrompt, expected) {
			t.Fatalf("expected report plan prompt to contain %q:\n%s", expected, planPrompt)
		}
	}
	for _, forbidden := range []string{
		"Mission recall:",
		"plasma.agent_recall_preview",
		`"Sources"`,
		`"Evidence"`,
		`"Claims"`,
	} {
		if strings.Contains(prompt, forbidden) || strings.Contains(planPrompt, forbidden) {
			t.Fatalf("report prompts contain forbidden payload marker %q:\nplan=%s\narticle=%s", forbidden, planPrompt, prompt)
		}
	}
}

func TestAgentReportRepairPromptExplainsApprovedRefBoundary(t *testing.T) {
	prompt := agentReportRepairPrompt(
		"Report",
		"mis_1",
		"ses_1",
		reportRigorProfiles["balanced"],
		agentReportPlan{Summary: "Use approved material."},
		agentReportAST{
			Title: "Report",
			Blocks: []agentReportBlock{{
				Type: "paragraph",
				Text: "Draft text.",
				Refs: app.ReportBlockSourceRefs{ClaimIDs: []string{"clm_proposed"}},
			}},
		},
		[]string{"clm_approved"},
		[]string{"evd_approved"},
		[]reportRefViolation{{
			ObjectKind: "claim_record",
			ObjectID:   "clm_proposed",
			State:      "proposed",
			Reason:     "claim is not approved for this report scope",
			BlockIndex: 2,
			BlockType:  "paragraph",
		}},
	)
	for _, expected := range []string{
		"correctable reference mistake",
		"Final AST refs/source_refs must only contain approved claim_ids and approved evidence_ids",
		"clm_approved",
		"evd_approved",
		"clm_proposed",
		"proposed",
		"Original AST to repair",
		"mission_id mis_1",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected repair prompt to contain %q:\n%s", expected, prompt)
		}
	}
}

func TestAgentPATHDedupesAndKeepsHomebrewFirst(t *testing.T) {
	got := agentPATH("/bin:/opt/homebrew/bin:/custom/bin")
	want := "/opt/homebrew/bin:/usr/local/bin:/bin:/custom/bin:/usr/bin:/usr/sbin:/sbin"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveAgentCommandKeepsAbsoluteCommand(t *testing.T) {
	if got := resolveAgentCommand("/tmp/codex"); got != "/tmp/codex" {
		t.Fatalf("expected absolute command to be kept, got %q", got)
	}
}

func TestCodexExecutorCheckForkSessionUsesHomeCodexDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", "")
	sessionID := "session-home-default"
	sessionDir := filepath.Join(home, ".codex", "sessions", "2026", "07", "02")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "rollout-"+sessionID+".jsonl"), []byte(`{"id":"`+sessionID+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	executor := CodexExecutor{Env: []string{"HOME=" + home}}
	if err := executor.CheckForkSession(context.Background(), sessionID); err != nil {
		t.Fatalf("expected HOME/.codex session to be ready, got %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(sessionDir, ".plasma-fork-check-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("readiness check should clean temporary files, got %#v", matches)
	}
}

func TestAgentSessionForkReadyRequiresReadinessInterface(t *testing.T) {
	if AgentSessionForkReady(context.Background(), &fakeForkOnlyExecutor{}, "session-1") {
		t.Fatal("fork readiness must not be optimistic when executor cannot verify readiness")
	}
}

type fakeForkOnlyExecutor struct{}

func (executor *fakeForkOnlyExecutor) Run(context.Context, AgentRequest) (AgentResult, error) {
	return AgentResult{Text: "ok", SessionID: "session-1"}, nil
}

func (executor *fakeForkOnlyExecutor) ForkSession(context.Context, string) (AgentSessionForkResult, error) {
	return AgentSessionForkResult{SessionID: "forked-session"}, nil
}

func TestCodexExecutorUsesSlashCompactForCompactionResume(t *testing.T) {
	dir := t.TempDir()
	capturePath := filepath.Join(dir, "stdin.txt")
	scriptPath := filepath.Join(dir, "fake-codex")
	script := `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    out="$1"
  fi
  shift
done
/bin/cat > "$CAPTURE"
printf 'session id: prior-session\n'
printf 'done' > "$out"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	result, err := (CodexExecutor{
		Command: scriptPath,
		WorkDir: dir,
		Timeout: 2 * time.Second,
		Env: []string{
			"CAPTURE=" + capturePath,
			"PATH=/usr/bin:/bin",
		},
	}).Run(context.Background(), AgentRequest{
		Prompt:            "custom compaction prompt must not be sent",
		MissionID:         "mis_1",
		ToolSessionID:     "ses_1",
		PreviousSessionID: "prior-session",
		Compaction:        true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionID != "prior-session" {
		t.Fatalf("expected prior session id, got %q", result.SessionID)
	}
	captured, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(captured) != "/compact" {
		t.Fatalf("expected slash compact stdin, got %q", string(captured))
	}
}

func TestCodexExecutorInjectsPlasmaMCPConfig(t *testing.T) {
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args.txt")
	scriptPath := filepath.Join(dir, "fake-codex")
	script := `#!/bin/sh
out=""
for arg in "$@"; do
  printf '%s\n' "$arg" >> "$ARGS_CAPTURE"
  if [ "$arg" = "--output-last-message" ]; then
    want_out=1
  elif [ "$want_out" = "1" ]; then
    out="$arg"
    want_out=0
  fi
done
printf 'session id: new-session\n'
printf 'done' > "$out"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := (CodexExecutor{
		Command: scriptPath,
		WorkDir: dir,
		Timeout: 2 * time.Second,
		Env: []string{
			"ARGS_CAPTURE=" + argsPath,
			"PATH=/usr/bin:/bin",
		},
		MCPServer: CodexMCPServer{
			Name:              "plasma",
			Command:           "/tmp/plasma-browser-server",
			Args:              []string{"mcp", "-db", "/tmp/plasma.db"},
			Required:          true,
			StartupTimeoutSec: 10,
			ToolTimeoutSec:    60,
			EnabledTools:      []string{"plasma.sources.list", "plasma.sources.read"},
		},
	}).Run(context.Background(), AgentRequest{
		Prompt:        "hi",
		MissionID:     "mis_1",
		ToolSessionID: "ses_1",
		UserEventID:   "evt_user_1",
		AgentExecutor: "codex",
	})
	if err != nil {
		t.Fatal(err)
	}
	captured, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	args := string(captured)
	for _, expected := range []string{
		`mcp_servers.plasma.command="/tmp/plasma-browser-server"`,
		`mcp_servers.plasma.args=["mcp","-db","/tmp/plasma.db","-mission-id","mis_1","-agent-session-id","ses_1","-current-user-event-id","evt_user_1","-agent-executor","codex"]`,
		`mcp_servers.plasma.required=true`,
		`mcp_servers.plasma.enabled_tools=["plasma.sources.list","plasma.sources.read"]`,
	} {
		if !strings.Contains(args, expected) {
			t.Fatalf("expected %q in args:\n%s", expected, args)
		}
	}
}

func TestCodexExecutorCanDisableMCPConfig(t *testing.T) {
	args := codexMCPConfigArgs(CodexMCPServer{
		Name:     "plasma",
		Command:  "/tmp/plasma-browser-server",
		Args:     []string{"mcp", "-db", "/tmp/plasma.db"},
		Required: true,
	}, AgentRequest{
		MissionID:     "mis_1",
		ToolSessionID: "ses_1",
		DisableTools:  true,
	})
	if len(args) != 0 {
		t.Fatalf("expected disabled tools to omit MCP config args, got %#v", args)
	}
}

func TestCodexExecutorCanReplaceMCPToolsForPatchOnlyRequest(t *testing.T) {
	args := codexMCPConfigArgs(CodexMCPServer{
		Name:         "plasma",
		Command:      "/tmp/plasma-browser-server",
		Args:         []string{"mcp", "-db", "/tmp/plasma.db", "-enabled-tool", "plasma.sources.list"},
		EnabledTools: []string{"plasma.sources.list", "plasma.sources.read"},
	}, AgentRequest{
		MissionID:       "mis_1",
		ToolSessionID:   "ses_1",
		ReplaceMCPTools: true,
		ExtraMCPTools:   []string{"plasma.report.patch.start", "plasma.report.patch.read"},
	})
	joined := strings.Join(args, "\n")
	for _, forbidden := range []string{"plasma.sources.list", "plasma.sources.read"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("expected replaced MCP tools to omit %s, got %#v", forbidden, args)
		}
	}
	for _, expected := range []string{"plasma.report.patch.start", "plasma.report.patch.read"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected replaced MCP tools to include %s, got %#v", expected, args)
		}
	}
}

func TestCodexExecutorAddsBoundReportPlanToolWithoutReplacingResearchTools(t *testing.T) {
	args := codexMCPConfigArgs(CodexMCPServer{
		Name: "plasma", Command: "/tmp/plasma", Args: []string{"mcp", "-db", "/tmp/plasma.db", "-enabled-tool", "plasma.sources.read"}, EnabledTools: []string{"plasma.sources.read"},
	}, AgentRequest{
		MissionID: "mis_1", ToolSessionID: "ses_tool", AgentExecutor: "codex", ExtraMCPTools: []string{"plasma.report.plan.submit"},
		ReportPlan: &AgentReportPlanContext{PendingEventID: "evt_pending", ReportMode: "planned", IdempotencyKey: "key_1", PreviousProviderSessionID: "ses_previous", AgentModel: "gpt-test", AgentReasoningEffort: "high"},
	})
	joined := strings.Join(args, "\n")
	for _, expected := range []string{"plasma.sources.read", "plasma.report.plan.submit", "-report-plan-pending-event-id", "evt_pending", "-report-plan-mode", "planned", "-report-plan-idempotency-key", "key_1", "-report-plan-tool-session-id", "ses_tool", "-report-plan-previous-provider-session-id", "ses_previous", "-report-plan-agent-model", "gpt-test", "-report-plan-agent-reasoning-effort", "high"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %q in %#v", expected, args)
		}
	}
	if strings.Contains(joined, "-report-patch") {
		t.Fatalf("report plan binding enabled patch mode: %#v", args)
	}
}

func TestAgentSectionalReportPlanPromptContainsConcreteBinding(t *testing.T) {
	prompt := agentSectionalReportPlanPrompt("Long", "mis_long", "ses_tool", "evt_pending_long", "key_long", reportRigorProfiles["strict"], "")
	for _, expected := range []string{"mission_id mis_long", "session_id ses_tool", "pending_event_id evt_pending_long", "report_mode long_form", "idempotency_key key_long", `producer {"type":"agent_session","id":"ses_tool"}`, "at most three parsed submission calls total"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("sectional prompt missing %q:\n%s", expected, prompt)
		}
	}
	if strings.Contains(prompt, "supplied by the tool context") || strings.Contains(prompt, "exactly once") {
		t.Fatalf("sectional prompt retained false binding or retry wording:\n%s", prompt)
	}
}

func TestLongFormGenerationGuidanceAcceptsSectionBriefOptions(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		profile string
		marker  string
	}{
		{
			name:    "section brief",
			input:   "section_brief",
			profile: reportGenerationGuidanceProfileSectionBrief,
			marker:  "Long-form section-brief guidance:",
		},
		{
			name:    "section brief cluster memory",
			input:   "section_brief_cluster_memory",
			profile: reportGenerationGuidanceProfileSectionBriefCluster,
			marker:  "Long-form section-brief cluster-memory guidance:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, sha, err := SelectReportGenerationGuidanceForMode(reportModeLongForm, tt.input)
			if err != nil {
				t.Fatalf("expected %s to be accepted for long-form reports: %v", tt.input, err)
			}
			if profile != tt.profile || strings.TrimSpace(sha) == "" {
				t.Fatalf("unexpected profile selection: profile=%q sha=%q", profile, sha)
			}
			guidance := LongFormReportGenerationGuidance(profile)
			if !strings.Contains(guidance, tt.marker) || !strings.Contains(guidance, "Long-form human-writer guidance:") {
				t.Fatalf("long-form guidance for %s missing expected markers:\n%s", profile, guidance)
			}
		})
	}
	if _, _, err := SelectReportGenerationGuidanceForMode(reportModePlanned, "section_brief"); err == nil {
		t.Fatalf("section_brief must remain long-form-only")
	}
}

func TestCodexExecutorPassesModelOverride(t *testing.T) {
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args.txt")
	scriptPath := filepath.Join(dir, "fake-codex")
	script := `#!/bin/sh
out=""
for arg in "$@"; do
  printf '%s\n' "$arg" >> "$ARGS_CAPTURE"
  if [ "$arg" = "--output-last-message" ]; then
    want_out=1
  elif [ "$want_out" = "1" ]; then
    out="$arg"
    want_out=0
  fi
done
printf 'session id: new-session\n'
printf 'done' > "$out"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := (CodexExecutor{
		Command: scriptPath,
		WorkDir: dir,
		Timeout: 2 * time.Second,
		Env: []string{
			"ARGS_CAPTURE=" + argsPath,
			"PATH=/usr/bin:/bin",
		},
	}).Run(context.Background(), AgentRequest{
		Prompt:          "hi",
		Model:           "gpt-5.5",
		ReasoningEffort: "low",
	})
	if err != nil {
		t.Fatal(err)
	}
	captured, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Split(strings.TrimSpace(string(captured)), "\n")
	if len(args) < 5 || args[0] != "exec" || args[1] != "--model" || args[2] != "gpt-5.5" || args[3] != "-c" || args[4] != `model_reasoning_effort="low"` {
		t.Fatalf("expected model and reasoning override immediately after exec, got %#v", args)
	}
}

func TestCodexExecutorResolvesNewSessionDefaultsButPreservesLegacyResume(t *testing.T) {
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args.txt")
	scriptPath := filepath.Join(dir, "fake-codex")
	script := `#!/bin/sh
out=""
for arg in "$@"; do
  printf '%s\n' "$arg" >> "$ARGS_CAPTURE"
  if [ "$arg" = "--output-last-message" ]; then want_out=1
  elif [ "$want_out" = "1" ]; then out="$arg"; want_out=0; fi
done
printf 'session id: session\n'
printf 'done' > "$out"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	executor := CodexExecutor{Command: scriptPath, WorkDir: dir, Timeout: 2 * time.Second, Env: []string{"ARGS_CAPTURE=" + argsPath, "PATH=/usr/bin:/bin"}}
	if _, err := executor.Run(context.Background(), AgentRequest{Prompt: "new"}); err != nil {
		t.Fatal(err)
	}
	captured, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(captured), "--model\ngpt-5.5\n") || !strings.Contains(string(captured), "model_reasoning_effort=\"medium\"") {
		t.Fatalf("expected GPT-5.5/medium defaults, got %q", captured)
	}
	if err := os.WriteFile(argsPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := executor.Run(context.Background(), AgentRequest{Prompt: "resume", PreviousSessionID: "legacy-session"}); err != nil {
		t.Fatal(err)
	}
	captured, err = os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(captured), "--model") || strings.Contains(string(captured), "model_reasoning_effort") {
		t.Fatalf("legacy resume must preserve empty settings, got %q", captured)
	}
}

func TestCodexExecutorCapturesJSONLUsage(t *testing.T) {
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args.txt")
	scriptPath := filepath.Join(dir, "fake-codex")
	script := `#!/bin/sh
out=""
for arg in "$@"; do
  printf '%s\n' "$arg" >> "$ARGS_CAPTURE"
  if [ "$arg" = "--output-last-message" ]; then
    want_out=1
  elif [ "$want_out" = "1" ]; then
    out="$arg"
    want_out=0
  fi
done
printf '{"type":"thread.started","thread_id":"json-session"}\n'
printf '{"type":"turn.completed","usage":{"input_tokens":120,"cached_input_tokens":80,"output_tokens":30,"reasoning_output_tokens":7}}\n'
printf 'done' > "$out"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	result, err := (CodexExecutor{
		Command: scriptPath,
		WorkDir: dir,
		Timeout: 2 * time.Second,
		Env: []string{
			"ARGS_CAPTURE=" + argsPath,
			"PATH=/usr/bin:/bin",
		},
	}).Run(context.Background(), AgentRequest{
		Prompt:          "hi",
		Model:           "gpt-5.5",
		ReasoningEffort: "low",
		AgentExecutor:   "codex",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionID != "json-session" {
		t.Fatalf("expected JSONL session id, got %q", result.SessionID)
	}
	if result.Usage.ProviderUsage == nil {
		t.Fatalf("expected provider usage: %#v", result.Usage)
	}
	if result.Usage.ProviderUsage.InputTokens != 120 || result.Usage.ProviderUsage.CachedInputTokens != 80 || result.Usage.ProviderUsage.UncachedInputTokens != 40 {
		t.Fatalf("unexpected usage: %#v", result.Usage.ProviderUsage)
	}
	if result.Usage.Prompt.Bytes != 2 || result.Usage.Prompt.EstimatedTokens != 1 || result.Usage.UsageSource != "codex_jsonl_turn_completed" {
		t.Fatalf("unexpected usage envelope: %#v", result.Usage)
	}
	captured, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(captured), "--json\n") {
		t.Fatalf("expected --json in args:\n%s", string(captured))
	}
}

func TestAddAgentUsagePayloadSkipsEmptyAndAddsUsage(t *testing.T) {
	payload := map[string]any{}
	addAgentUsagePayload(payload, agentusage.AgentUsage{}, "turn", 12, "", "", false, false)
	if _, ok := payload["agent_usage"]; ok {
		t.Fatalf("empty usage should not be added: %#v", payload)
	}
	usage := agentusage.New("codex", "codex", "gpt-5.5", "low", "hi").
		WithProviderUsage(agentusage.ProviderUsage{InputTokens: 12, CachedInputTokens: 8, OutputTokens: 3}, "test")
	addAgentUsagePayload(payload, usage, "turn", 34, "prev-session", "session-1", true, false)
	eventUsage, ok := payload["agent_usage"].(agentusage.AgentUsage)
	if !ok {
		t.Fatalf("expected agent_usage payload, got %#v", payload)
	}
	if eventUsage.Surface != "turn" || eventUsage.DurationMS != 34 || eventUsage.Session.PreviousAgentSessionID != "prev-session" || eventUsage.Session.AgentSessionID != "session-1" || !eventUsage.Session.Resumed {
		t.Fatalf("unexpected event usage: %#v", eventUsage)
	}
}

func TestCodexEnvironmentAllowsExplicitOverride(t *testing.T) {
	env := codexEnvironment([]string{"A=B"})
	if len(env) != 1 || env[0] != "A=B" {
		t.Fatalf("expected explicit env to be preserved, got %#v", env)
	}
}

func containsEnv(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

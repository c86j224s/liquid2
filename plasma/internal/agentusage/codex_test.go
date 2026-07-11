package agentusage

import "testing"

func TestParseCodexJSONLUsageAndSession(t *testing.T) {
	log := `{"type":"thread.started","thread_id":"019f-session"}
{"type":"turn.completed","usage":{"input_tokens":120,"cached_input_tokens":80,"output_tokens":30,"reasoning_output_tokens":7}}
`
	sessionID := ParseCodexSessionID(log)
	if sessionID != "019f-session" {
		t.Fatalf("expected JSONL thread id, got %q", sessionID)
	}
	usage, ok := ParseCodexProviderUsage(log)
	if !ok {
		t.Fatalf("expected usage")
	}
	if usage.InputTokens != 120 || usage.CachedInputTokens != 80 || usage.UncachedInputTokens != 40 || usage.OutputTokens != 30 || usage.ReasoningOutputTokens != 7 || usage.TotalTokens != 150 {
		t.Fatalf("unexpected usage: %#v", usage)
	}
}

func TestParseCodexSessionIDFallsBackToStatusLine(t *testing.T) {
	if got := ParseCodexSessionID("session id: prior-session\n"); got != "prior-session" {
		t.Fatalf("expected status session id, got %q", got)
	}
}

func TestForEventPreservesUsageUnavailableReason(t *testing.T) {
	usage := New("codex", "codex", "gpt-5.5", "low", "hello").
		WithUnavailable("codex JSONL did not include turn.completed usage")
	eventUsage, ok := usage.ForEvent("turn", 12, "prev", "next", true, false)
	if !ok {
		t.Fatal("expected event usage")
	}
	if !eventUsage.UsageUnavailable || eventUsage.UsageUnavailableReason != "codex JSONL did not include turn.completed usage" {
		t.Fatalf("expected specific unavailable reason to be preserved, got %#v", eventUsage)
	}
}

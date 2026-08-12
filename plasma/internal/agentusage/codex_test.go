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
	if usage.Scope != UsageScopeSessionCumulative {
		t.Fatalf("Codex turn.completed usage must be marked cumulative, got %#v", usage)
	}
}

func TestParseCodexSessionIDFallsBackToStatusLine(t *testing.T) {
	if got := ParseCodexSessionID("session id: prior-session\n"); got != "prior-session" {
		t.Fatalf("expected status session id, got %q", got)
	}
}

func TestParseCodexContextWindowMetricsUsesLatestValidTokenCount(t *testing.T) {
	log := `not-json
{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"total_tokens":110},"model_context_window":200}}}
{"type":"event_msg","payload":{"type":"token_count","info":null}}
{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"total_tokens":120},"model_context_window":200}}}
`
	metrics, ok := ParseCodexContextWindowMetrics(log)
	if !ok {
		t.Fatal("expected context window metrics")
	}
	if metrics.UsedTokens != 120 || metrics.WindowTokens != 200 || metrics.Source != "codex_session_token_count" {
		t.Fatalf("unexpected context metrics: %#v", metrics)
	}
	if !metrics.AtOrAbovePercent(55) || metrics.AtOrAbovePercent(61) {
		t.Fatalf("unexpected threshold result: %#v", metrics)
	}
}

func TestParseCodexContextWindowMetricsRejectsIncompleteObservation(t *testing.T) {
	log := `{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"total_tokens":120},"model_context_window":0}}}`
	if metrics, ok := ParseCodexContextWindowMetrics(log); ok || metrics.Valid() {
		t.Fatalf("expected incomplete metrics to be rejected, got %#v", metrics)
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

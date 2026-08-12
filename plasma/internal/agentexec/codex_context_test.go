package agentexec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexContextWindowMetricsReadsBoundedSessionTail(t *testing.T) {
	codexHome := t.TempDir()
	sessionID := "019f-context-session"
	sessionDir := filepath.Join(codexHome, "sessions", "2026", "08", "11")
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := strings.Repeat("x", 128) + "\n" +
		`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"total_tokens":143000},"model_context_window":258400}}}` + "\n"
	if err := os.WriteFile(filepath.Join(sessionDir, "rollout-"+sessionID+".jsonl"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	metrics, ok := (CodexExecutor{Env: []string{"CODEX_HOME=" + codexHome}}).codexContextWindowMetrics(sessionID)
	if !ok {
		t.Fatal("expected Codex context metrics")
	}
	if metrics.UsedTokens != 143000 || metrics.WindowTokens != 258400 || !metrics.AtOrAbovePercent(55) {
		t.Fatalf("unexpected context metrics: %#v", metrics)
	}
}

func TestCodexContextWindowMetricsIsUnavailableWithoutSessionFile(t *testing.T) {
	if metrics, ok := (CodexExecutor{Env: []string{"CODEX_HOME=" + t.TempDir()}}).codexContextWindowMetrics("missing"); ok || metrics.Valid() {
		t.Fatalf("expected unavailable metrics, got %#v", metrics)
	}
}

func TestCodexTelemetrySessionIDFallsBackForResumedRun(t *testing.T) {
	if got := codexTelemetrySessionID("", " 019f-resumed-session "); got != "019f-resumed-session" {
		t.Fatalf("expected previous session ID, got %q", got)
	}
	if got := codexTelemetrySessionID(" 019f-returned-session ", "019f-resumed-session"); got != "019f-returned-session" {
		t.Fatalf("expected returned session ID, got %q", got)
	}
}

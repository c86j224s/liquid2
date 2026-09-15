package longformutil

import (
	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"testing"
)

func TestSharedHelpersPreserveSerializationAndFilename(t *testing.T) {
	if got := AnyJSON(map[string]string{"a": "b"}); got != "{\n  \"a\": \"b\"\n}" {
		t.Fatalf("JSON=%q", got)
	}
	if got := AnyJSON(make(chan int)); got != "{}" {
		t.Fatalf("fallback=%q", got)
	}
	if got := SafeFilename(" A.B / 문서 ", ".md"); got != "a-b.md" {
		t.Fatalf("filename=%q", got)
	}
}

func TestSessionMismatchClearsReturnedSession(t *testing.T) {
	got, err := ValidateSameSessionResult(agentexec.AgentResult{SessionID: "other"}, "original")
	if err == nil || got.SessionID != "" {
		t.Fatalf("session=%q err=%v", got.SessionID, err)
	}
	got, err = ValidateSameSessionResult(agentexec.AgentResult{SessionID: " same "}, "same")
	if err != nil || got.SessionID != "same" {
		t.Fatalf("session=%q err=%v", got.SessionID, err)
	}
}

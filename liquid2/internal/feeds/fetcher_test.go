package feeds

import (
	"strings"
	"testing"
)

func TestReadLimitedUsesConfiguredLimitInError(t *testing.T) {
	_, err := readLimited(strings.NewReader("abcd"), 3)
	if err == nil {
		t.Fatal("expected limit error")
	}
	if !strings.Contains(err.Error(), "response exceeds 3 bytes") {
		t.Fatalf("unexpected limit error %q", err.Error())
	}
}

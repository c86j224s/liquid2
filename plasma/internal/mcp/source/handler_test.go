package source

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"testing"
)

func TestSourceListBindingFailurePrecedesServiceAccess(t *testing.T) {
	calls := 0
	h := Handler{Decode: func(raw json.RawMessage, target any) error { return json.Unmarshal(raw, target) }, ValidateID: func(string, string) error { return nil }, EnforceMission: func(string) error { calls++; return errors.New("outside mission") }, ErrorResult: func(tool, mission, kind, message string, retry bool, related []string) wire.ToolResult {
		return wire.ToolResult{}
	}}
	h.CallSourcesList(context.Background(), wire.ToolCall{Arguments: []byte(`{"mission_id":"mis_one"}`)})
	if calls != 1 {
		t.Fatalf("binding calls=%d", calls)
	}
}

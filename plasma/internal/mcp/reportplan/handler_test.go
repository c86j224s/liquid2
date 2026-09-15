package reportplan

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"testing"
)

func TestInvalidInputDoesNotResolveSubmissionCapability(t *testing.T) {
	h := Handler{Binding: func() Binding { return Binding{} }, MissionID: func() string { return "mis_one" }, Available: func() bool { return true }, Capability: func() (Service, bool) { t.Fatal("capability resolved before validation"); return nil, false }, ErrorResult: func(tool, mission, kind, message string, retry bool, ids []string) wire.ToolResult {
		return wire.ToolResult{ToolName: tool, MissionID: mission}
	}}
	h.Submit(context.Background(), wire.ToolCall{Name: "test", Arguments: []byte(`invalid`)})
	if h.AttemptCount() != 1 {
		t.Fatalf("attempts=%d", h.AttemptCount())
	}
}

func TestBindingReadPerCallBeforeAvailability(t *testing.T) {
	count := 0
	h := Handler{Binding: func() Binding { count++; return Binding{} }, MissionID: func() string { return "mis_one" }, Available: func() bool { return false }, ErrorResult: func(tool, mission, kind, message string, retry bool, ids []string) wire.ToolResult {
		return wire.ToolResult{}
	}}
	h.Submit(context.Background(), wire.ToolCall{})
	h.Submit(context.Background(), wire.ToolCall{})
	if count != 2 || h.AttemptCount() != 0 {
		t.Fatalf("binding reads=%d attempts=%d", count, h.AttemptCount())
	}
}

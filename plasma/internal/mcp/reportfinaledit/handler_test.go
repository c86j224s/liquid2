package reportfinaledit

import (
	"context"
	"encoding/json"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"testing"
)

func TestUnavailableFinalizationStopsBeforeDecode(t *testing.T) {
	h := Handler{FinalizeBinding: func() reporting.LongFormFinalizeBinding { return reporting.LongFormFinalizeBinding{} }, FinalizeAvailable: func(reporting.LongFormFinalizeBinding) bool { return false }, MissionID: func() string { return "mis_one" }, Decode: func(json.RawMessage, any) error { t.Fatal("decode before unavailable gate"); return nil }, ErrorResult: func(tool, mission, kind, message string, retry bool, related []string) wire.ToolResult {
		return wire.ToolResult{}
	}}
	h.CallReportLongFormFinalize(context.Background(), wire.ToolCall{})
}

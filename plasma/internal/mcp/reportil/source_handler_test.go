package reportil

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"sync"
	"testing"
)

func TestSourceBindingFailurePrecedesReadDecode(t *testing.T) {
	h := SourceHandler{Mu: &sync.Mutex{}, State: &NewState().Source, MissionID: func() string { return "mis_one" }, AccessBinding: func() (reportilcontract.SourceAccessBinding, error) {
		return reportilcontract.SourceAccessBinding{}, errors.New("invalid binding")
	}, Decode: func(json.RawMessage, any) error { t.Fatal("decoded before binding rejection"); return nil }, ErrorResult: func(tool, mission, kind, message string, retry bool, ids []string) wire.ToolResult {
		return wire.ToolResult{ToolName: tool, MissionID: mission}
	}}
	h.CallReportILSourcesRead(context.Background(), wire.ToolCall{Name: "test", Arguments: []byte(`{}`)})
	if h.State.ReadBytes != 0 {
		t.Fatal("binding rejection mutated read budget")
	}
}

func TestQuoteWireOmitsTraceMetadata(t *testing.T) {
	data, err := json.Marshal(ReportILSourceQuoteOutput{SourceReceipt: "opaque", SourceKey: "private", Offset: 42, ByteSize: 9, Sha256: "hash", CatalogSHA256: "catalog", Stage: "il_narrative"})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"source_receipt":"opaque"}` {
		t.Fatalf("trace metadata leaked: %s", data)
	}
}

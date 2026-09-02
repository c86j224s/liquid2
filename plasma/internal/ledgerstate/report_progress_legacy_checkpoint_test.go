package ledgerstate

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

func TestProjectReportProgressEnablesRecoverableLegacyILCheckpoint(t *testing.T) {
	pendingID := "evt_legacy_checkpoint_pending"
	finalizePayload := func(tool, stage, artifact string, accounts int) map[string]any {
		return map[string]any{
			"tool_name": tool, "success": true,
			"io_metrics": map[string]any{
				"report_il_stage": stage, "artifact_id": artifact,
				"sha256": strings.Repeat("a", 64), "byte_size": 100,
				"revision": 1, "accounts": accounts, "finalized": true,
			},
		}
	}
	progress := ProjectReportProgress([]Event{
		reportEvent(pendingID, "report.draft.pending", map[string]any{"origin_pending_event_id": pendingID, "report_mode": "long_form", "pipeline_family": "report_il_experimental"}),
		reportEvent("evt_memory", "mcp.tool.called", finalizePayload(reportilcontract.EditorialMemoryFinalizeTool, "il_editorial_memory", "art_memory", 1)),
		reportEvent("evt_final_tool", "mcp.tool.called", finalizePayload(reportilcontract.LongFormDocumentFinalizeTool, "il_long_form_final", "art_final", 0)),
		reportEvent("evt_final_done", "report.il_long_form_final.completed", map[string]any{"pending_event_id": pendingID}),
		reportEvent("evt_failed", "report.draft.failed", map[string]any{"pending_event_id": pendingID, "failed_stage_kind": "il_reader", "failed_stage_id": "il_reader"}),
	})
	if !progress.Retry.ResumeFailed || !progress.Retry.Restart || progress.Retry.ReasonCode != "legacy_checkpoint_recoverable" {
		t.Fatalf("legacy retry capability = %#v", progress.Retry)
	}
}

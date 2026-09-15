package reportexecution

import (
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
	"strings"
)

func ReportPendingEvent(events []ledger.Event, pendingID string) (ledger.Event, bool) {
	for _, event := range events {
		if event.EventID != pendingID {
			continue
		}
		switch event.EventType {
		case "report.draft.pending", "report.design.pending", "report.humanize.pending", "report.patch.pending":
			return event, true
		default:
			return ledger.Event{}, false
		}
	}
	return ledger.Event{}, false
}

func reportPendingPipelineFamily(event ledger.Event) string {
	if event.EventType != "report.draft.pending" {
		return ""
	}
	var payload struct {
		PipelineFamily string `json:"pipeline_family"`
	}
	if json.Unmarshal(event.Payload, &payload) != nil {
		return ""
	}
	return strings.TrimSpace(payload.PipelineFamily)
}

func IsIndependentReportILPending(event ledger.Event) bool {
	return reportPendingPipelineFamily(event) == reportpipeline.ExperimentalIL
}

func ValidateReportTerminalAppend(pending, terminal ledger.Event) (bool, error) {
	var payload struct {
		PendingID      string `json:"pending_event_id"`
		StageKind      string `json:"stage_kind"`
		StageID        string `json:"stage_id"`
		TerminalID     string `json:"terminal_event_id"`
		PipelineFamily string `json:"pipeline_family"`
		Generation     struct {
			PendingID string `json:"pending_event_id"`
		} `json:"generation"`
	}
	if err := json.Unmarshal(terminal.Payload, &payload); err != nil {
		return false, fmt.Errorf("%w: invalid report event payload", producterror.ErrInvalidInput)
	}
	if terminal.EventType == "report.drafted" && payload.PendingID == "" {
		payload.PendingID = payload.Generation.PendingID
	}
	if strings.TrimSpace(payload.PendingID) != pending.EventID {
		return false, fmt.Errorf("%w: report event must correlate to pending event %q", producterror.ErrInvalidInput, pending.EventID)
	}
	family := strings.TrimSpace(payload.PipelineFamily)
	pendingFamily := reportPendingPipelineFamily(pending)
	if pendingFamily != "" && !reportpipeline.Independent(pendingFamily) {
		return false, fmt.Errorf("%w: unsupported report pending pipeline family", producterror.ErrInvalidInput)
	}
	if family != "" && (!reportpipeline.Independent(family) || family != pendingFamily) {
		return false, fmt.Errorf("%w: report terminal family does not match pending event", producterror.ErrInvalidInput)
	}
	if family == "" && reportpipeline.Independent(pendingFamily) && terminal.EventType == "report.artifact.created" {
		return false, fmt.Errorf("%w: independent report terminal family is required", producterror.ErrInvalidInput)
	}
	if strings.HasPrefix(terminal.EventType, "report.") && strings.HasSuffix(terminal.EventType, ".failed") && terminal.EventType != "report.draft.failed" && terminal.EventType != "report.patch.failed" && terminal.EventType != "report.design.failed" && terminal.EventType != "report.humanize.failed" {
		kind := strings.TrimPrefix(strings.TrimSuffix(terminal.EventType, ".failed"), "report.")
		validKind := map[string]bool{"plan": true, "requirements": true, "part_plan": true, "section": true, "part": true, "part_edit": true, "final": true, "artifact": true, "il_source_selection": true, "il_editorial_memory": true, "il_narrative": true, "il_long_form_plan": true, "il_long_form_sections": true, "il_long_form_parts": true, "il_long_form_final": true, "il_continuity": true, "il_reader": true, "il_images": true, "il_document": true, "il_flow": true, "il_render": true, "il_store": true, "source_packet": true}[kind]
		independentKind := map[string]bool{"il_source_selection": true, "il_editorial_memory": true, "il_narrative": true, "il_long_form_plan": true, "il_long_form_sections": true, "il_long_form_parts": true, "il_long_form_final": true, "il_continuity": true, "il_reader": true, "il_images": true, "il_document": true, "il_flow": true, "il_render": true, "il_store": true, "source_packet": true}[kind]
		if pending.EventType != "report.draft.pending" || !validKind || payload.StageKind != kind || payload.StageID == "" || independentKind && !IsIndependentReportILPending(pending) {
			return false, fmt.Errorf("%w: invalid report stage companion", producterror.ErrInvalidInput)
		}
		return false, nil
	}
	allowed := map[string]map[string]bool{
		"report.draft.pending":    {"report.draft.failed": true, "report.drafted": true, "report.artifact.created": true},
		"report.design.pending":   {"report.design.failed": true, "report.artifact.exported": true},
		"report.humanize.pending": {"report.humanize.failed": true, "report.humanize.skipped": true, "report.artifact.exported": true},
		"report.patch.pending":    {"report.patch.failed": true, "report.artifact.created": true},
	}
	if !allowed[pending.EventType][terminal.EventType] {
		return false, fmt.Errorf("%w: terminal event %q does not match %q", producterror.ErrInvalidInput, terminal.EventType, pending.EventType)
	}
	return true, nil
}

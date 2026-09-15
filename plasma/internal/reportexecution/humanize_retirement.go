package reportexecution

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"strings"
	"time"
)

type humanizeRetiredError struct{}

func (humanizeRetiredError) Error() string {
	return "deprecated post-canonical H5 has been retired; original artifact preserved"
}

// RetireHumanizePending preserves historical lineage while conditionally closing
// the old operation without provider work or artifact promotion.
func (runner Runner) RetireHumanizePending(ctx context.Context, missionID string, pending ledger.Event) error {
	payload := humanizePendingPayloadFromEvent(pending)
	executor := firstNonEmpty(payload.AgentExecutor, "plasma")
	_, _, err := runner.Service.AppendReportTerminalIfOpen(ctx, missionID, pending.EventID, []ledger.AppendRequest{{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.humanize.failed",
		Producer:  ledger.Producer{Type: "system", ID: "plasma"},
		Payload: mustJSON(map[string]any{
			"kind":                        "humanized_markdown_report_retired",
			"target":                      firstNonEmpty(payload.Target, ExportTargetHumanizedMarkdown),
			"profile":                     firstNonEmpty(payload.Profile, HumanizeProfileH5),
			"pending_event_id":            pending.EventID,
			"report_pending_event_id":     strings.TrimSpace(payload.ReportPendingEventID),
			"title":                       strings.TrimSpace(payload.Title),
			"source_artifact_id":          strings.TrimSpace(payload.SourceArtifactID),
			"source_artifact_sha256":      strings.TrimSpace(payload.SourceArtifactSHA256),
			"agent_executor":              executor,
			"agent_model":                 strings.TrimSpace(payload.AgentModel),
			"agent_reasoning_effort":      strings.TrimSpace(payload.AgentReasoningEffort),
			"previous_agent_session_id":   strings.TrimSpace(payload.PreviousSessionID),
			"tool_session_id":             strings.TrimSpace(payload.ToolSessionID),
			"mcp_mode":                    strings.TrimSpace(payload.MCPMode),
			"report_mode":                 strings.TrimSpace(payload.ReportMode),
			"report_mode_label":           strings.TrimSpace(payload.ReportModeLabel),
			"humanize_transport":          firstNonEmpty(payload.HumanizeTransport, HumanizeTransportPatch),
			"text":                        "폐기된 H5 작업을 종료하고 기존 원본 Markdown artifact를 유지했습니다.",
			"error":                       humanizeRetiredError{}.Error(),
			"internal_failure_detail":     "feature_retired",
			"retired":                     true,
			"retired_at":                  time.Now().UTC().Format(time.RFC3339Nano),
			"relationship":                "retired_post_report_tone_pass_of_source_artifact",
			"preserved_original_markdown": true,
		}),
	}})
	return err
}

package mcp

import (
	"context"
	"errors"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type reportLongFormFinalizeInput struct {
	MissionID      string       `json:"mission_id"`
	SessionID      string       `json:"session_id"`
	PendingEventID string       `json:"pending_event_id"`
	PlanEventID    string       `json:"plan_event_id"`
	IdempotencyKey string       `json:"idempotency_key"`
	Producer       app.Producer `json:"producer"`
	Opening        string       `json:"opening_markdown"`
	Closing        string       `json:"closing_markdown"`
}

func (server *Server) callReportLongFormFinalize(ctx context.Context, call ToolCall) ToolResult {
	binding := server.longFormFinalizeBinding
	if ValidateLongFormFinalizeBinding(server.binding, binding) != nil || !server.toolEnabled(ToolReportLongFormFinalize) {
		return errorResult(call.Name, server.binding.MissionID, "binding", "long-form finalization binding is incomplete", false, nil)
	}
	var input reportLongFormFinalizeInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, server.binding.MissionID, "validation", "long-form finalization arguments are invalid", false, nil)
	}
	if input.MissionID != binding.MissionID || input.SessionID != binding.ToolSessionID || input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID || input.IdempotencyKey != binding.IdempotencyKey || strings.TrimSpace(input.Producer.Type) != "agent_session" || strings.TrimSpace(input.Producer.ID) != binding.ToolSessionID {
		return errorResult(call.Name, input.MissionID, "binding", "long-form finalization call does not match the runner binding", false, nil)
	}
	result, err := reporting.FinalizeLongForm(ctx, server.service, reporting.LongFormFinalizeRequest{Binding: binding, EventID: newMCPID("evt"), OpeningMarkdown: input.Opening, ClosingMarkdown: input.Closing})
	if err != nil {
		kind := "storage"
		if errors.Is(err, app.ErrInvalidInput) {
			kind = "validation"
		}
		if errors.Is(err, app.ErrConflict) {
			kind = "conflict"
		}
		return errorResult(call.Name, input.MissionID, kind, "long-form finalization was rejected", false, nil)
	}
	return ToolResult{ToolName: call.Name, MissionID: input.MissionID, CreatedEventIDs: []string{result.Event.EventID}, Content: map[string]any{
		"event_id": result.Event.EventID, "artifact_id": result.Artifact.ArtifactID, "artifact_sha256": result.Artifact.SHA256, "replay": result.Replay,
	}}
}

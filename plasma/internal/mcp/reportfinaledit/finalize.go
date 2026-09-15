package reportfinaledit

import (
	"context"
	"errors"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type ReportLongFormFinalizeInput struct {
	MissionID      string          `json:"mission_id"`
	SessionID      string          `json:"session_id"`
	PendingEventID string          `json:"pending_event_id"`
	PlanEventID    string          `json:"plan_event_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	Producer       ledger.Producer `json:"producer"`
	Opening        string          `json:"opening_markdown"`
	Closing        string          `json:"closing_markdown"`
}

func (server *Handler) CallReportLongFormFinalize(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	binding := server.FinalizeBinding()
	if !server.FinalizeAvailable(binding) {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", "long-form finalization binding is incomplete", false, nil)
	}
	var input ReportLongFormFinalizeInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "long-form finalization arguments are invalid", false, nil)
	}
	if input.MissionID != binding.MissionID || input.SessionID != binding.ToolSessionID || input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID || input.IdempotencyKey != binding.IdempotencyKey || strings.TrimSpace(input.Producer.Type) != "agent_session" || strings.TrimSpace(input.Producer.ID) != binding.ToolSessionID {
		return server.ErrorResult(call.Name, input.MissionID, "binding", "long-form finalization call does not match the runner binding", false, nil)
	}
	result, err := reporting.FinalizeLongForm(ctx, server.Service, reporting.LongFormFinalizeRequest{Binding: binding, EventID: server.NewID("evt"), OpeningMarkdown: input.Opening, ClosingMarkdown: input.Closing})
	if err != nil {
		kind := "storage"
		if errors.Is(err, producterror.ErrInvalidInput) {
			kind = "validation"
		}
		if errors.Is(err, producterror.ErrConflict) {
			kind = "conflict"
		}
		return server.ErrorResult(call.Name, input.MissionID, kind, "long-form finalization was rejected", false, nil)
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: input.MissionID, CreatedEventIDs: []string{result.Event.EventID}, Content: map[string]any{
		"event_id": result.Event.EventID, "artifact_id": result.Artifact.ArtifactID, "artifact_sha256": result.Artifact.SHA256, "replay": result.Replay,
	}}
}

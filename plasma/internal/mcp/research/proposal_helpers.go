package research

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
)

func (handler *Handler) mutationReadyResult(toolName, missionID string) (wire.ToolResult, bool) {
	if !handler.legacyResearchLoop {
		return errorResult(toolName, missionID, "validation", "legacy research mutation tool is disabled in the default C1 loop", false, nil), false
	}
	if err := handler.enforceBoundMission(missionID); err != nil {
		return errorResult(toolName, missionID, "validation", err.Error(), false, nil), false
	}
	if handler.proposalWriter == nil {
		return errorResult(toolName, missionID, "validation", "research proposal writer is not available", false, nil), false
	}
	return wire.ToolResult{}, true
}

func normalizeMutatingInput(input CommonMutatingInput) (CommonMutatingInput, app.Producer, error) {
	input.MissionID = strings.TrimSpace(input.MissionID)
	input.SessionID = strings.TrimSpace(input.SessionID)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if err := validateID("mis_", input.MissionID); err != nil {
		return input, app.Producer{}, err
	}
	if err := validateID("ses_", input.SessionID); err != nil {
		return input, app.Producer{}, err
	}
	if input.IdempotencyKey == "" {
		return input, app.Producer{}, fmt.Errorf("%w: idempotency_key is required", app.ErrInvalidInput)
	}
	producer := app.Producer{Type: strings.TrimSpace(input.Producer.Type), ID: strings.TrimSpace(input.Producer.ID)}
	if producer.Type == "" || producer.ID == "" {
		return input, app.Producer{}, fmt.Errorf("%w: producer type and id are required", app.ErrInvalidInput)
	}
	if producer.Type != "agent_session" || producer.ID != input.SessionID {
		return input, app.Producer{}, fmt.Errorf("%w: tool producer must be agent_session matching session_id", app.ErrInvalidInput)
	}
	return input, producer, nil
}

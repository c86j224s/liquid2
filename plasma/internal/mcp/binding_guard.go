package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func (server *Server) enforceBoundMission(missionID string) error {
	boundMissionID := strings.TrimSpace(server.binding.MissionID)
	if boundMissionID == "" {
		return nil
	}
	if missionID != boundMissionID {
		return fmt.Errorf("%w: tool call mission_id is outside this MCP session", producterror.ErrInvalidInput)
	}
	return nil
}

func (server *Server) enforceBoundMutation(args json.RawMessage) error {
	var input commonMutatingInput
	if err := decodeArgs(args, &input); err != nil {
		return err
	}
	missionID := strings.TrimSpace(input.MissionID)
	sessionID := strings.TrimSpace(input.SessionID)
	if err := server.enforceBoundMission(missionID); err != nil {
		return err
	}
	boundSessionID := strings.TrimSpace(server.binding.AgentSessionID)
	if boundSessionID != "" && sessionID != boundSessionID {
		return fmt.Errorf("%w: tool call session_id is outside this MCP session", producterror.ErrInvalidInput)
	}
	producer := ledger.Producer{
		Type: strings.TrimSpace(input.Producer.Type),
		ID:   strings.TrimSpace(input.Producer.ID),
	}
	if boundSessionID != "" && (producer.Type != "agent_session" || producer.ID != boundSessionID) {
		return fmt.Errorf("%w: tool producer is outside this MCP session", producterror.ErrInvalidInput)
	}
	return nil
}

func (server *Server) requireBoundWriteSession(input commonMutatingInput) error {
	boundMissionID := strings.TrimSpace(server.binding.MissionID)
	boundSessionID := strings.TrimSpace(server.binding.AgentSessionID)
	if boundMissionID == "" || boundSessionID == "" {
		return fmt.Errorf("%w: MCP write tools require a mission-bound MCP agent session", producterror.ErrInvalidInput)
	}
	if input.MissionID != boundMissionID || input.SessionID != boundSessionID {
		return fmt.Errorf("%w: tool call is outside this MCP session", producterror.ErrInvalidInput)
	}
	return nil
}

func (server *Server) boundObservationProducer() (ledger.Producer, string, error) {
	boundSessionID := strings.TrimSpace(server.binding.AgentSessionID)
	if boundSessionID == "" {
		return ledger.Producer{}, "", fmt.Errorf("%w: live source reads require a mission-bound MCP agent session", producterror.ErrInvalidInput)
	}
	if err := validateID("ses_", boundSessionID); err != nil {
		return ledger.Producer{}, "", err
	}
	return ledger.Producer{Type: "agent_session", ID: boundSessionID}, boundSessionID, nil
}

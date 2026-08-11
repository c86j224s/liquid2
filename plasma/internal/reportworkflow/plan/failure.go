package plan

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
)

type agentFailure struct {
	cause   error
	payload map[string]any
}

func (err agentFailure) Error() string { return err.cause.Error() }

func (err agentFailure) Unwrap() error { return err.cause }

func (err agentFailure) FailurePayload() map[string]any { return err.payload }

func reportAgentFailure(cause error, result agentexec.AgentResult, surface string, durationMS int64, previousSessionID string) error {
	if cause == nil {
		return nil
	}
	payload := map[string]any{"failed_surface": surface}
	if strings.TrimSpace(result.SessionID) != "" {
		payload["agent_session_id"] = result.SessionID
	}
	payload["resumed"] = result.Resumed
	if eventUsage, ok := result.Usage.ForEvent(surface, durationMS, previousSessionID, result.SessionID, result.Resumed, false); ok {
		payload["agent_usage"] = eventUsage
	}
	return agentFailure{cause: cause, payload: payload}
}

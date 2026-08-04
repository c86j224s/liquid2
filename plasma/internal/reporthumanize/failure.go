package reporthumanize

import "github.com/c86j224s/liquid2/plasma/internal/agentexec"

type failureWithPayload struct {
	cause   error
	payload map[string]any
}

func (err failureWithPayload) Error() string {
	return err.cause.Error()
}

func (err failureWithPayload) Unwrap() error {
	return err.cause
}

func (err failureWithPayload) FailurePayload() map[string]any {
	return err.payload
}

func agentFailure(cause error, result agentexec.AgentResult, surface string, durationMS int64, previousSessionID string) error {
	if cause == nil {
		return nil
	}
	payload := map[string]any{
		"failed_surface": surface,
	}
	if result.SessionID != "" {
		payload["agent_session_id"] = result.SessionID
	}
	payload["resumed"] = result.Resumed
	if eventUsage, ok := result.Usage.ForEvent(surface, durationMS, previousSessionID, result.SessionID, result.Resumed, false); ok {
		payload["agent_usage"] = eventUsage
	}
	return failureWithPayload{cause: cause, payload: payload}
}

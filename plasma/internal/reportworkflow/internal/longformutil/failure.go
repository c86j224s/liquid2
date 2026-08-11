package longformutil

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

type agentFailure struct {
	cause   error
	payload map[string]any
}

func (err agentFailure) Error() string { return err.cause.Error() }

func (err agentFailure) Unwrap() error { return err.cause }

func (err agentFailure) FailurePayload() map[string]any { return err.payload }

// AgentFailure는 provider 실패를 reportexecution terminal payload가 이해하는 usage metadata로 감싼다.
func AgentFailure(cause error, result agentexec.AgentResult, surface string, durationMS int64, previousSessionID string) error {
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

// StageFailure는 장문 prefix stage 좌표를 reportexecution 실패 분류로 보존한다.
func StageFailure(kind, planID string, part, section int, cause error) error {
	return reportexecution.NewStageFailure(kind, planID, part, section, cause)
}

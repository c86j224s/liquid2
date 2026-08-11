package finaledit

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

type failureWithPayload struct {
	cause   error
	payload map[string]any
}

func (err failureWithPayload) Error() string { return err.cause.Error() }

func (err failureWithPayload) Unwrap() error { return err.cause }

func (err failureWithPayload) FailurePayload() map[string]any { return err.payload }

// StageFailure는 기존 reportexecution.StageFailureError-compatible final stage 오류를 만든다.
func StageFailure(stage, planID string, cause error) error {
	if stage != "" {
		cause = stageError{stage: stage, cause: cause}
	}
	return reportexecution.NewStageFailure("final", planID, 0, 0, cause)
}

type stageError struct {
	stage string
	cause error
}

func (err stageError) Error() string { return err.stage + ": " + err.cause.Error() }

func (err stageError) Unwrap() error { return err.cause }

// AgentFailure는 agent 실행 실패 payload의 failed_surface, usage, session metadata를 보존한다.
func AgentFailure(cause error, result agentexec.AgentResult, surface string, durationMS int64, previousSessionID string) error {
	if cause == nil {
		return nil
	}
	payload := map[string]any{"failed_surface": surface}
	if strings.TrimSpace(result.SessionID) != "" {
		payload["agent_session_id"] = result.SessionID
	}
	payload["resumed"] = result.Resumed
	addAgentUsagePayload(payload, result.Usage, surface, durationMS, previousSessionID, result.SessionID, result.Resumed, false)
	return failureWithPayload{cause: cause, payload: payload}
}

func addAgentUsagePayload(payload map[string]any, usage agentusage.AgentUsage, surface string, durationMS int64, previousSessionID string, sessionID string, resumed bool, compaction bool) {
	if eventUsage, ok := usage.ForEvent(surface, durationMS, previousSessionID, sessionID, resumed, compaction); ok {
		payload["agent_usage"] = eventUsage
	}
}

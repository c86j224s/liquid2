package plan

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func (runner Runner) id(prefix string) string {
	if runner.NewID != nil {
		return runner.NewID(prefix)
	}
	return prefix + "_missing"
}

func validateSameSessionResult(result agentexec.AgentResult, previousSessionID string) (agentexec.AgentResult, error) {
	previousSessionID = strings.TrimSpace(previousSessionID)
	result.SessionID = strings.TrimSpace(result.SessionID)
	if previousSessionID == "" {
		if result.SessionID == "" {
			return result, fmt.Errorf("%w: agent did not return a session id", producterror.ErrInvalidInput)
		}
		return result, nil
	}
	if result.SessionID == "" {
		return result, fmt.Errorf("%w: agent did not return a session id for resumed session", producterror.ErrInvalidInput)
	}
	if result.SessionID != previousSessionID {
		result.SessionID = ""
		return result, fmt.Errorf("%w: agent returned a different session id", producterror.ErrInvalidInput)
	}
	return result, nil
}

func fallbackSessionID(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

package directdraft

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

func (runner Runner) id(prefix string) string {
	if runner.NewID != nil {
		return runner.NewID(prefix)
	}
	return prefix + "_missing"
}

func latestSession(ctx context.Context, runner Runner, missionID string, executor string) string {
	if runner.LatestSessionID == nil {
		return ""
	}
	return strings.TrimSpace(runner.LatestSessionID(ctx, missionID, executor))
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

func safeFilename(title string, ext string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "source"
	}
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '.':
			b.WriteRune('-')
		}
	}
	name := strings.Trim(b.String(), "-_")
	if name == "" {
		name = "source"
	}
	if len(name) > 80 {
		name = name[:80]
	}
	return name + ext
}

func modeLabel(mode string) string {
	return reportexecution.ModeLabel(mode)
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

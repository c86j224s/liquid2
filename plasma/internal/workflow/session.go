package workflow

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

// LatestAgentSessionID exposes the conversation session restoration contract
// to workflow callers.
func LatestAgentSessionID(events []ledger.Event, executorName string) string {
	return conversation.LatestAgentSessionID(events, executorName)
}

func validatedSameSessionResult(result AgentResult, previousSessionID string) (AgentResult, error) {
	previousSessionID = strings.TrimSpace(previousSessionID)
	result.SessionID = strings.TrimSpace(result.SessionID)
	if previousSessionID == "" {
		if result.SessionID == "" {
			return result, fmt.Errorf("%w: agent did not return a session id", producterror.ErrInvalidInput)
		}
		return result, nil
	}
	if result.SessionID == "" {
		result.SessionID = previousSessionID
		return result, nil
	}
	if result.SessionID != previousSessionID {
		result.SessionID = ""
		return result, fmt.Errorf("%w: agent returned a different session id", producterror.ErrInvalidInput)
	}
	return result, nil
}

func shouldAutoCompactAfterAgentError(previousSessionID string, err error, result AgentResult) bool {
	if strings.TrimSpace(previousSessionID) == "" || err == nil {
		return false
	}
	text := strings.ToLower(err.Error() + "\n" + result.Log)
	return strings.Contains(text, "ran out of room in the model's context window")
}

func workflowCompactPrompt() string {
	return strings.TrimSpace(`Compact the useful session context for future Plasma workflow steps. Do not answer the user's research question in this turn.

Preserve:
- the mission objective and any steering that changed it
- important sources, source candidates, and why they matter
- useful findings, unresolved questions, and next investigation paths
- constraints about using Plasma MCP tools and not treating agent results as sources`)
}

func hasOpenAgentPending(events []ledger.Event) bool {
	return conversation.HasOpenAgentPending(events)
}

func hasAgentTerminalEventForUser(events []ledger.Event, userEventID string) bool {
	if strings.TrimSpace(userEventID) == "" {
		return true
	}
	return conversation.HasAgentTerminalEventForUser(events, userEventID)
}

func nextInstruction(view workflowstate.WorkflowRunView) string {
	for i := len(view.Steps) - 1; i >= 0; i-- {
		if strings.TrimSpace(view.Steps[i].NextInstruction) != "" {
			return strings.TrimSpace(view.Steps[i].NextInstruction)
		}
	}
	return strings.TrimSpace(view.Instruction)
}

func latestContinuationInstruction(view workflowstate.WorkflowRunView) (string, bool) {
	for i := len(view.Steps) - 1; i >= 0; i-- {
		step := view.Steps[i]
		if strings.TrimSpace(step.Decision) != "continue" {
			return "", false
		}
		return strings.TrimSpace(firstNonEmpty(step.NextInstruction, step.Reason, view.Instruction)), true
	}
	return "", false
}

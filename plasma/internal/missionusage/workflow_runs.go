package missionusage

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

// workflowRunAccumulator owns the private session set needed to produce one
// public autonomous-research usage row without exposing provider session IDs.
type workflowRunAccumulator struct {
	value    WorkflowRunUsage
	sessions map[string]struct{}
	perCall  []int64
}

func newWorkflowRunAccumulator(workflowRunID string) *workflowRunAccumulator {
	return &workflowRunAccumulator{
		value:    WorkflowRunUsage{WorkflowRunID: workflowRunID},
		sessions: map[string]struct{}{},
	}
}

func (run *workflowRunAccumulator) add(usage agentusage.AgentUsage, increment agentusage.ProviderUsage) {
	run.value.CallCount++
	run.value.InputTokens += int64(increment.InputTokens)
	run.value.CachedInputTokens += int64(increment.CachedInputTokens)
	run.value.UncachedInputTokens += int64(increment.UncachedInputTokens)
	run.value.OutputTokens += int64(increment.OutputTokens)
	run.value.ReasoningOutputTokens += int64(increment.ReasoningOutputTokens)
	run.value.TotalTokens += int64(increment.TotalTokens)
	run.value.AgentModel = mergedDescriptor(run.value.AgentModel, usage.Model)
	run.value.ReasoningEffort = mergedDescriptor(run.value.ReasoningEffort, usage.ReasoningEffort)
	if sessionID := strings.TrimSpace(usage.Session.AgentSessionID); sessionID != "" {
		run.sessions[sessionID] = struct{}{}
	}
	if usage.Session.Resumed {
		run.value.ResumedCallCount++
	}
	run.perCall = append(run.perCall, int64(increment.TotalTokens))
}

func (run *workflowRunAccumulator) summary() WorkflowRunUsage {
	run.value.SessionCount = int64(len(run.sessions))
	run.value.PerCall = percentiles(run.perCall)
	return run.value
}

func mergedDescriptor(current string, next string) string {
	next = strings.TrimSpace(next)
	if next == "" || current == "mixed" {
		return current
	}
	if current == "" || current == next {
		return next
	}
	return "mixed"
}

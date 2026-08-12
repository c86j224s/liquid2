package missionusage

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

// Project converts agent usage records into corrected call increments in
// ledger order. It never preserves prompts, provider responses, or source text.
func Project(events []ledger.Event) Summary {
	summary := Summary{AggregationVersion: AggregationVersion}
	baselines := map[string]agentusage.AgentUsage{}
	sessions := map[string]bool{}
	seen := map[string]bool{}
	categoryValues := map[string]Category{}
	workflowRuns := map[string]*workflowRunAccumulator{}
	workflowRunOrder := make([]string, 0)
	perCall := make([]int64, 0)
	missingUsage := map[string]string{}
	missingUsageBySession := map[string]int{}

	for _, event := range orderedEvents(events) {
		if event.EventID == "" || seen[event.EventID] {
			continue
		}
		seen[event.EventID] = true
		record, ok := usagePayload(event.Payload)
		if sessionID, required := requiredUsageSession(event); required && !ok {
			missingUsage[event.EventID] = sessionID
			missingUsageBySession[sessionID]++
			if sessionID != "" {
				sessions[sessionID] = true
			}
		}
		if !ok {
			continue
		}
		if missingSession, found := missingUsage[record.CorrelationEventID]; found {
			delete(missingUsage, record.CorrelationEventID)
			missingUsageBySession[missingSession]--
		}
		summary.UsageRecordCount++

		var usage agentusage.AgentUsage
		if json.Unmarshal(record.AgentUsage, &usage) != nil {
			markUnavailable(&summary)
			continue
		}
		sessionID := strings.TrimSpace(usage.Session.AgentSessionID)
		if sessionID != "" {
			sessions[sessionID] = true
		}
		scope := agentusage.ProviderUsageScope(usage)
		var previous *agentusage.AgentUsage
		forkBaseline := false
		if scope == agentusage.UsageScopeSessionCumulative && sessionID != "" {
			if prior, found := baselines[sessionID]; found {
				copy := prior
				previous = &copy
			} else if record.ForkSourceAgentSessionID != "" {
				sourceID := record.ForkSourceAgentSessionID
				if missingUsageBySession[sourceID] > 0 {
					rememberCumulativeBaseline(baselines, sessionID, usage)
					markUnavailable(&summary)
					continue
				}
				prior, found := baselines[sourceID]
				if !found {
					rememberCumulativeBaseline(baselines, sessionID, usage)
					markUnavailable(&summary)
					continue
				}
				copy := prior
				copy.Session.AgentSessionID = sessionID
				previous = &copy
				forkBaseline = true
			}
		}
		increment, metadata, ok := agentusage.IncrementalProviderUsage(usage, previous)
		if !ok {
			markUnavailable(&summary)
			continue
		}
		if forkBaseline && metadata.CounterReset {
			baselines[sessionID] = usage
			markUnavailable(&summary)
			continue
		}
		if scope == agentusage.UsageScopeSessionCumulative {
			baselines[sessionID] = usage
		}
		if metadata.CounterReset {
			summary.CounterResetCount++
			summary.UsagePartial = true
		}
		if !addUsage(&summary, increment) {
			markUnavailable(&summary)
			continue
		}

		summary.UsageAvailableCount++
		perCall = append(perCall, int64(increment.TotalTokens))
		category := categoryForSurface(usage.Surface)
		value := categoryValues[category]
		value.Key = category
		value.Label = categoryLabels[category]
		value.CallCount++
		value.TotalTokens += int64(increment.TotalTokens)
		categoryValues[category] = value
		if strings.TrimSpace(usage.Surface) == "workflow_step" && record.WorkflowRunID != "" {
			run := workflowRuns[record.WorkflowRunID]
			if run == nil {
				run = newWorkflowRunAccumulator(record.WorkflowRunID)
				workflowRuns[record.WorkflowRunID] = run
				workflowRunOrder = append(workflowRunOrder, record.WorkflowRunID)
			}
			run.add(usage, increment)
		}
		if strings.HasSuffix(strings.TrimSpace(event.EventType), ".failed") {
			summary.FailedCallCount++
			summary.FailedTotalTokens += int64(increment.TotalTokens)
		}
	}
	if len(missingUsage) > 0 {
		summary.UsageUnavailableCount += int64(len(missingUsage))
		summary.UsagePartial = true
	}

	summary.SessionCount = int64(len(sessions))
	summary.PerCall = percentiles(perCall)
	for _, key := range categoryOrder {
		if value := categoryValues[key]; value.CallCount > 0 {
			summary.Categories = append(summary.Categories, value)
		}
	}
	for _, runID := range workflowRunOrder {
		summary.WorkflowRuns = append(summary.WorkflowRuns, workflowRuns[runID].summary())
	}
	return summary
}

func orderedEvents(events []ledger.Event) []ledger.Event {
	ordered := append([]ledger.Event(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Sequence != ordered[j].Sequence {
			return ordered[i].Sequence < ordered[j].Sequence
		}
		if !ordered[i].CreatedAt.Equal(ordered[j].CreatedAt) {
			return ordered[i].CreatedAt.Before(ordered[j].CreatedAt)
		}
		return ordered[i].EventID < ordered[j].EventID
	})
	return ordered
}

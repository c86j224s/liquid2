package reportrun

import (
	"encoding/json"
	"math"
	"sort"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

// AggregateUsage builds the retained post-purge token aggregate from member
// events. It reads only top-level agent_usage and never copies prompts or raw
// provider responses. Session-cumulative snapshots are converted to increments
// in ledger order before they are added.
func AggregateUsage(events []MemberEvent) UsageAggregate {
	aggregate := UsageAggregate{AggregationVersion: UsageAggregationVersion}
	seen := map[string]bool{}
	baselines := map[string]agentusage.AgentUsage{}

	for _, member := range orderedUsageEvents(events) {
		eventID := member.Event.EventID
		if eventID == "" || seen[eventID] {
			continue
		}
		seen[eventID] = true
		raw, ok := agentUsagePayload(member.Event.Payload)
		if !ok {
			continue
		}
		aggregate.UsageRecordCount++

		var usage agentusage.AgentUsage
		if json.Unmarshal(raw, &usage) != nil || usage.ProviderUsage == nil || usage.UsageUnavailable {
			aggregate = markUsageUnavailable(aggregate)
			continue
		}

		scope := agentusage.ProviderUsageScope(usage)
		var previous *agentusage.AgentUsage
		sessionID := strings.TrimSpace(usage.Session.AgentSessionID)
		if scope == agentusage.UsageScopeSessionCumulative && sessionID != "" {
			if prior, found := baselines[sessionID]; found {
				priorCopy := prior
				previous = &priorCopy
			}
		}
		increment, metadata, ok := agentusage.IncrementalProviderUsage(usage, previous)
		if !ok {
			aggregate = markUsageUnavailable(aggregate)
			continue
		}
		if scope == agentusage.UsageScopeSessionCumulative {
			baselines[sessionID] = usage
		}
		if metadata.CounterReset {
			aggregate.UsagePartial = true
		}
		next, ok := aggregateWithProviderUsage(aggregate, increment)
		if !ok {
			aggregate = markUsageUnavailable(aggregate)
			continue
		}
		aggregate = next
		aggregate.UsageAvailableCount++
	}
	return aggregate
}

func orderedUsageEvents(events []MemberEvent) []MemberEvent {
	ordered := append([]MemberEvent(nil), events...)
	sort.SliceStable(ordered, func(i int, j int) bool {
		left := ordered[i].Event
		right := ordered[j].Event
		if left.Sequence != right.Sequence {
			return left.Sequence < right.Sequence
		}
		if !left.CreatedAt.Equal(right.CreatedAt) {
			return left.CreatedAt.Before(right.CreatedAt)
		}
		return left.EventID < right.EventID
	})
	return ordered
}

func agentUsagePayload(raw []byte) (json.RawMessage, bool) {
	var payload struct {
		AgentUsage json.RawMessage `json:"agent_usage"`
	}
	if json.Unmarshal(raw, &payload) != nil || len(payload.AgentUsage) == 0 || string(payload.AgentUsage) == "null" {
		return nil, false
	}
	return payload.AgentUsage, true
}

func markUsageUnavailable(aggregate UsageAggregate) UsageAggregate {
	aggregate.UsageUnavailableCount++
	aggregate.UsagePartial = true
	return aggregate
}

func aggregateWithProviderUsage(aggregate UsageAggregate, provider agentusage.ProviderUsage) (UsageAggregate, bool) {
	values := []int{
		provider.InputTokens,
		provider.CachedInputTokens,
		provider.UncachedInputTokens,
		provider.OutputTokens,
		provider.ReasoningOutputTokens,
		provider.TotalTokens,
	}
	for _, value := range values {
		if value < 0 {
			return UsageAggregate{}, false
		}
	}
	next := aggregate
	fields := []struct {
		target *int64
		value  int
	}{
		{&next.InputTokens, provider.InputTokens},
		{&next.CachedInputTokens, provider.CachedInputTokens},
		{&next.UncachedInputTokens, provider.UncachedInputTokens},
		{&next.OutputTokens, provider.OutputTokens},
		{&next.ReasoningOutputTokens, provider.ReasoningOutputTokens},
		{&next.TotalTokens, provider.TotalTokens},
	}
	for _, field := range fields {
		value, ok := checkedAddInt64(*field.target, int64(field.value))
		if !ok {
			return UsageAggregate{}, false
		}
		*field.target = value
	}
	return next, true
}

func checkedAddInt64(left int64, right int64) (int64, bool) {
	if right > math.MaxInt64-left {
		return 0, false
	}
	return left + right, true
}

package reportrun

import (
	"encoding/json"
	"math"
)

// AggregateUsage builds the retained post-purge token aggregate from member
// events. It reads only top-level agent_usage and never copies prompts or raw
// provider responses.
func AggregateUsage(events []MemberEvent) UsageAggregate {
	aggregate := UsageAggregate{AggregationVersion: UsageAggregationVersion}
	seen := map[string]bool{}
	for _, member := range events {
		eventID := member.Event.EventID
		if eventID == "" || seen[eventID] {
			continue
		}
		seen[eventID] = true
		var payload struct {
			AgentUsage json.RawMessage `json:"agent_usage"`
		}
		if json.Unmarshal(member.Event.Payload, &payload) != nil || len(payload.AgentUsage) == 0 || string(payload.AgentUsage) == "null" {
			continue
		}
		aggregate.UsageRecordCount++
		usage, ok := decodeAgentUsage(payload.AgentUsage)
		if !ok || usage.ProviderUsage == nil || usage.UsageUnavailable {
			aggregate.UsageUnavailableCount++
			aggregate.UsagePartial = true
			continue
		}
		next, ok := aggregateWithProviderUsage(aggregate, *usage.ProviderUsage)
		if !ok {
			aggregate.UsageUnavailableCount++
			aggregate.UsagePartial = true
			continue
		}
		aggregate = next
		aggregate.UsageAvailableCount++
	}
	return aggregate
}

type agentUsagePayload struct {
	UsageUnavailable bool                  `json:"usage_unavailable"`
	ProviderUsage    *providerUsagePayload `json:"provider_usage"`
}

type providerUsagePayload struct {
	InputTokens           int64 `json:"input_tokens"`
	CachedInputTokens     int64 `json:"cached_input_tokens"`
	UncachedInputTokens   int64 `json:"uncached_input_tokens"`
	OutputTokens          int64 `json:"output_tokens"`
	ReasoningOutputTokens int64 `json:"reasoning_output_tokens"`
	TotalTokens           int64 `json:"total_tokens"`
}

func decodeAgentUsage(raw json.RawMessage) (agentUsagePayload, bool) {
	var payload struct {
		UsageUnavailable bool            `json:"usage_unavailable"`
		ProviderUsage    json.RawMessage `json:"provider_usage"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return agentUsagePayload{}, false
	}
	if len(payload.ProviderUsage) == 0 || string(payload.ProviderUsage) == "null" {
		return agentUsagePayload{UsageUnavailable: payload.UsageUnavailable}, true
	}
	var provider providerUsagePayload
	if err := json.Unmarshal(payload.ProviderUsage, &provider); err != nil {
		return agentUsagePayload{}, false
	}
	return agentUsagePayload{UsageUnavailable: payload.UsageUnavailable, ProviderUsage: &provider}, true
}

func aggregateWithProviderUsage(aggregate UsageAggregate, provider providerUsagePayload) (UsageAggregate, bool) {
	values := []int64{
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
	var ok bool
	if next.InputTokens, ok = checkedAddInt64(next.InputTokens, provider.InputTokens); !ok {
		return UsageAggregate{}, false
	}
	if next.CachedInputTokens, ok = checkedAddInt64(next.CachedInputTokens, provider.CachedInputTokens); !ok {
		return UsageAggregate{}, false
	}
	if next.UncachedInputTokens, ok = checkedAddInt64(next.UncachedInputTokens, provider.UncachedInputTokens); !ok {
		return UsageAggregate{}, false
	}
	if next.OutputTokens, ok = checkedAddInt64(next.OutputTokens, provider.OutputTokens); !ok {
		return UsageAggregate{}, false
	}
	if next.ReasoningOutputTokens, ok = checkedAddInt64(next.ReasoningOutputTokens, provider.ReasoningOutputTokens); !ok {
		return UsageAggregate{}, false
	}
	if next.TotalTokens, ok = checkedAddInt64(next.TotalTokens, provider.TotalTokens); !ok {
		return UsageAggregate{}, false
	}
	return next, true
}

func checkedAddInt64(left int64, right int64) (int64, bool) {
	if right > math.MaxInt64-left {
		return 0, false
	}
	return left + right, true
}

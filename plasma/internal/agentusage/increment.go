package agentusage

import "strings"

const legacyCodexCumulativeSource = "codex_jsonl_turn_completed"

// IncrementMetadata describes how a provider snapshot became a call increment.
// A counter reset keeps the current snapshot usable but means continuity is not
// proven, so aggregate callers should mark their result partial.
type IncrementMetadata struct {
	InitialSnapshot bool
	CounterReset    bool
}

// ProviderUsageScope returns the recorded usage meaning. Schema v1 Codex JSONL
// records are the only legacy values inferred as cumulative; all other missing
// scopes retain the historical call-scoped meaning.
func ProviderUsageScope(usage AgentUsage) string {
	if usage.ProviderUsage != nil {
		if scope := strings.TrimSpace(usage.ProviderUsage.Scope); scope != "" {
			return scope
		}
	}
	if usage.SchemaVersion == 1 && strings.TrimSpace(usage.UsageSource) == legacyCodexCumulativeSource {
		return UsageScopeSessionCumulative
	}
	return UsageScopeCall
}

// IncrementalProviderUsage converts one observed provider value into the amount
// attributable to the current call. Cumulative values require a stable session
// ID and use the previous valid snapshot for that session when available.
func IncrementalProviderUsage(current AgentUsage, previous *AgentUsage) (ProviderUsage, IncrementMetadata, bool) {
	if current.ProviderUsage == nil || current.UsageUnavailable {
		return ProviderUsage{}, IncrementMetadata{}, false
	}
	scope := ProviderUsageScope(current)
	provider := *current.ProviderUsage
	provider.Normalize()
	if !validProviderUsage(provider) {
		return ProviderUsage{}, IncrementMetadata{}, false
	}

	switch scope {
	case UsageScopeCall:
		provider.Scope = UsageScopeCall
		return provider, IncrementMetadata{}, true
	case UsageScopeSessionCumulative:
		sessionID := strings.TrimSpace(current.Session.AgentSessionID)
		if sessionID == "" {
			return ProviderUsage{}, IncrementMetadata{}, false
		}
		if previous == nil {
			provider.Scope = UsageScopeCall
			return provider, IncrementMetadata{InitialSnapshot: true}, true
		}
		if strings.TrimSpace(previous.Session.AgentSessionID) != sessionID ||
			ProviderUsageScope(*previous) != UsageScopeSessionCumulative || previous.ProviderUsage == nil {
			return ProviderUsage{}, IncrementMetadata{}, false
		}
		prior := *previous.ProviderUsage
		prior.Normalize()
		if !validProviderUsage(prior) {
			return ProviderUsage{}, IncrementMetadata{}, false
		}
		if countersDecreased(provider, prior) {
			provider.Scope = UsageScopeCall
			return provider, IncrementMetadata{CounterReset: true}, true
		}
		return subtractProviderUsage(provider, prior), IncrementMetadata{}, true
	default:
		return ProviderUsage{}, IncrementMetadata{}, false
	}
}

func validProviderUsage(usage ProviderUsage) bool {
	values := []int{
		usage.InputTokens,
		usage.CachedInputTokens,
		usage.UncachedInputTokens,
		usage.OutputTokens,
		usage.ReasoningOutputTokens,
		usage.TotalTokens,
	}
	for _, value := range values {
		if value < 0 {
			return false
		}
	}
	return usage.CachedInputTokens <= usage.InputTokens &&
		usage.UncachedInputTokens <= usage.InputTokens &&
		usage.ReasoningOutputTokens <= usage.OutputTokens
}

func countersDecreased(current ProviderUsage, previous ProviderUsage) bool {
	return current.InputTokens < previous.InputTokens ||
		current.CachedInputTokens < previous.CachedInputTokens ||
		current.UncachedInputTokens < previous.UncachedInputTokens ||
		current.OutputTokens < previous.OutputTokens ||
		current.ReasoningOutputTokens < previous.ReasoningOutputTokens ||
		current.TotalTokens < previous.TotalTokens
}

func subtractProviderUsage(current ProviderUsage, previous ProviderUsage) ProviderUsage {
	return ProviderUsage{
		Scope:                 UsageScopeCall,
		InputTokens:           current.InputTokens - previous.InputTokens,
		CachedInputTokens:     current.CachedInputTokens - previous.CachedInputTokens,
		UncachedInputTokens:   current.UncachedInputTokens - previous.UncachedInputTokens,
		OutputTokens:          current.OutputTokens - previous.OutputTokens,
		ReasoningOutputTokens: current.ReasoningOutputTokens - previous.ReasoningOutputTokens,
		TotalTokens:           current.TotalTokens - previous.TotalTokens,
	}
}

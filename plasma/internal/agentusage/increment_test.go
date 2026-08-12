package agentusage

import "testing"

func TestNewUsageRecordsCurrentSchemaAndCallScope(t *testing.T) {
	usage := New("claude", "claude", "model", "low", "prompt").
		WithProviderUsage(ProviderUsage{InputTokens: 10, OutputTokens: 4}, "claude_json")
	if SchemaVersion != 2 {
		t.Fatalf("schema contract changed without updating this test: %d", SchemaVersion)
	}
	if usage.SchemaVersion != SchemaVersion {
		t.Fatalf("unexpected schema version: %#v", usage)
	}
	if usage.ProviderUsage == nil || usage.ProviderUsage.Scope != UsageScopeCall || usage.ProviderUsage.TotalTokens != 14 {
		t.Fatalf("new call usage must record explicit call scope: %#v", usage)
	}
}

func TestIncrementalProviderUsageKeepsCallScopedValue(t *testing.T) {
	usage := AgentUsage{
		SchemaVersion: 1,
		ProviderUsage: &ProviderUsage{InputTokens: 10, OutputTokens: 4, TotalTokens: 14},
	}
	got, metadata, ok := IncrementalProviderUsage(usage, nil)
	if !ok || metadata != (IncrementMetadata{}) {
		t.Fatalf("expected direct call usage, got usage=%#v metadata=%#v ok=%v", got, metadata, ok)
	}
	if got.Scope != UsageScopeCall || got.InputTokens != 10 || got.OutputTokens != 4 || got.TotalTokens != 14 {
		t.Fatalf("unexpected call usage: %#v", got)
	}
}

func TestIncrementalProviderUsageCalculatesCumulativeDelta(t *testing.T) {
	previous := cumulativeUsage(2, "session-1", ProviderUsage{
		InputTokens: 100, CachedInputTokens: 60, UncachedInputTokens: 40,
		OutputTokens: 10, ReasoningOutputTokens: 4, TotalTokens: 110,
	})
	current := cumulativeUsage(2, "session-1", ProviderUsage{
		InputTokens: 160, CachedInputTokens: 100, UncachedInputTokens: 60,
		OutputTokens: 18, ReasoningOutputTokens: 7, TotalTokens: 178,
	})

	got, metadata, ok := IncrementalProviderUsage(current, &previous)
	if !ok || metadata != (IncrementMetadata{}) {
		t.Fatalf("expected cumulative delta, got usage=%#v metadata=%#v ok=%v", got, metadata, ok)
	}
	if got.Scope != UsageScopeCall || got.InputTokens != 60 || got.CachedInputTokens != 40 ||
		got.UncachedInputTokens != 20 || got.OutputTokens != 8 ||
		got.ReasoningOutputTokens != 3 || got.TotalTokens != 68 {
		t.Fatalf("unexpected cumulative delta: %#v", got)
	}
}

func TestIncrementalProviderUsageUsesFirstSnapshotAndDetectsReset(t *testing.T) {
	first := cumulativeUsage(2, "session-1", ProviderUsage{InputTokens: 100, OutputTokens: 10, TotalTokens: 110})
	got, metadata, ok := IncrementalProviderUsage(first, nil)
	if !ok || !metadata.InitialSnapshot || metadata.CounterReset || got.TotalTokens != 110 {
		t.Fatalf("expected initial cumulative snapshot, got usage=%#v metadata=%#v ok=%v", got, metadata, ok)
	}

	reset := cumulativeUsage(2, "session-1", ProviderUsage{InputTokens: 20, OutputTokens: 3, TotalTokens: 23})
	got, metadata, ok = IncrementalProviderUsage(reset, &first)
	if !ok || metadata.InitialSnapshot || !metadata.CounterReset || got.TotalTokens != 23 {
		t.Fatalf("expected reset snapshot, got usage=%#v metadata=%#v ok=%v", got, metadata, ok)
	}
}

func TestProviderUsageScopeInfersOnlyLegacyCodexCumulativeRecords(t *testing.T) {
	legacy := AgentUsage{
		SchemaVersion: 1,
		UsageSource:   legacyCodexCumulativeSource,
		ProviderUsage: &ProviderUsage{},
	}
	if got := ProviderUsageScope(legacy); got != UsageScopeSessionCumulative {
		t.Fatalf("expected legacy Codex cumulative scope, got %q", got)
	}
	legacy.UsageSource = "claude_json"
	if got := ProviderUsageScope(legacy); got != UsageScopeCall {
		t.Fatalf("non-Codex legacy usage must remain call scoped, got %q", got)
	}
	legacy.SchemaVersion = 0
	legacy.UsageSource = legacyCodexCumulativeSource
	if got := ProviderUsageScope(legacy); got != UsageScopeCall {
		t.Fatalf("schema-less usage must not be reclassified, got %q", got)
	}
}

func TestIncrementalProviderUsageRejectsUnusableCumulativeRecords(t *testing.T) {
	tests := []struct {
		name  string
		usage AgentUsage
	}{
		{
			name:  "missing session",
			usage: cumulativeUsage(2, "", ProviderUsage{InputTokens: 10, TotalTokens: 10}),
		},
		{
			name: "unknown scope",
			usage: AgentUsage{SchemaVersion: 2, ProviderUsage: &ProviderUsage{
				Scope: "unknown", InputTokens: 10, TotalTokens: 10,
			}},
		},
		{
			name: "invalid counters",
			usage: cumulativeUsage(2, "session-1", ProviderUsage{
				InputTokens: 5, CachedInputTokens: 6, TotalTokens: 5,
			}),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, metadata, ok := IncrementalProviderUsage(tc.usage, nil); ok {
				t.Fatalf("expected rejection, got usage=%#v metadata=%#v", got, metadata)
			}
		})
	}
}

func cumulativeUsage(schemaVersion int, sessionID string, provider ProviderUsage) AgentUsage {
	provider.Scope = UsageScopeSessionCumulative
	return AgentUsage{
		SchemaVersion: schemaVersion,
		Session:       SessionMetrics{AgentSessionID: sessionID},
		ProviderUsage: &provider,
	}
}

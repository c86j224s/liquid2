package reportrun

import (
	"math"
	"testing"
)

func TestAggregateUsageCountsEachMemberEventOnce(t *testing.T) {
	events := []MemberEvent{
		{Event: Event{EventID: "evt_a", Payload: jsonPayload(t, map[string]any{
			"agent_usage": map[string]any{"provider_usage": map[string]any{
				"input_tokens": 10, "cached_input_tokens": 3, "uncached_input_tokens": 7,
				"output_tokens": 4, "reasoning_output_tokens": 2, "total_tokens": 16,
			}},
		})}},
		{Event: Event{EventID: "evt_a", Payload: jsonPayload(t, map[string]any{
			"agent_usage": map[string]any{"provider_usage": map[string]any{"total_tokens": 999}},
		})}},
		{Event: Event{EventID: "evt_b", Payload: jsonPayload(t, map[string]any{
			"agent_usage": map[string]any{"usage_unavailable": true},
		})}},
		{Event: Event{EventID: "evt_c", Payload: jsonPayload(t, map[string]any{"note": "ignored"})}},
	}

	got := AggregateUsage(events)
	if got.UsageRecordCount != 2 || got.UsageAvailableCount != 1 || got.UsageUnavailableCount != 1 {
		t.Fatalf("unexpected usage counts: %#v", got)
	}
	if got.InputTokens != 10 || got.CachedInputTokens != 3 || got.UncachedInputTokens != 7 ||
		got.OutputTokens != 4 || got.ReasoningOutputTokens != 2 || got.TotalTokens != 16 {
		t.Fatalf("unexpected usage totals: %#v", got)
	}
	if !got.UsagePartial || got.AggregationVersion != UsageAggregationVersion {
		t.Fatalf("unexpected usage metadata: %#v", got)
	}
}

func TestAggregateUsageConvertsCumulativeSnapshotsInLedgerOrder(t *testing.T) {
	events := []MemberEvent{
		cumulativeUsageEvent(t, "evt_second", 2, 2, "session-1", map[string]any{
			"input_tokens": 160, "cached_input_tokens": 100, "uncached_input_tokens": 60,
			"output_tokens": 18, "reasoning_output_tokens": 7, "total_tokens": 178,
		}),
		cumulativeUsageEvent(t, "evt_first", 1, 2, "session-1", map[string]any{
			"input_tokens": 100, "cached_input_tokens": 60, "uncached_input_tokens": 40,
			"output_tokens": 10, "reasoning_output_tokens": 4, "total_tokens": 110,
		}),
	}

	got := AggregateUsage(events)
	if got.UsageRecordCount != 2 || got.UsageAvailableCount != 2 || got.UsageUnavailableCount != 0 || got.UsagePartial {
		t.Fatalf("unexpected cumulative usage metadata: %#v", got)
	}
	if got.InputTokens != 160 || got.CachedInputTokens != 100 || got.UncachedInputTokens != 60 ||
		got.OutputTokens != 18 || got.ReasoningOutputTokens != 7 || got.TotalTokens != 178 {
		t.Fatalf("cumulative snapshots must total to the latest observed value, got %#v", got)
	}
}

func TestAggregateUsageMarksCounterResetPartial(t *testing.T) {
	events := []MemberEvent{
		cumulativeUsageEvent(t, "evt_first", 1, 2, "session-1", map[string]any{
			"input_tokens": 100, "output_tokens": 10, "total_tokens": 110,
		}),
		cumulativeUsageEvent(t, "evt_reset", 2, 2, "session-1", map[string]any{
			"input_tokens": 20, "output_tokens": 3, "total_tokens": 23,
		}),
	}

	got := AggregateUsage(events)
	if got.UsageAvailableCount != 2 || got.UsageUnavailableCount != 0 || !got.UsagePartial {
		t.Fatalf("reset snapshot should remain available but partial: %#v", got)
	}
	if got.InputTokens != 120 || got.OutputTokens != 13 || got.TotalTokens != 133 {
		t.Fatalf("reset snapshot should establish a new baseline: %#v", got)
	}
}

func TestAggregateUsageInfersLegacyCodexCumulativeScope(t *testing.T) {
	events := []MemberEvent{
		cumulativeUsageEvent(t, "evt_first", 1, 1, "session-1", map[string]any{
			"input_tokens": 10, "output_tokens": 2, "total_tokens": 12,
		}),
		cumulativeUsageEvent(t, "evt_second", 2, 1, "session-1", map[string]any{
			"input_tokens": 15, "output_tokens": 3, "total_tokens": 18,
		}),
	}

	got := AggregateUsage(events)
	if got.UsageAvailableCount != 2 || got.UsagePartial || got.TotalTokens != 18 {
		t.Fatalf("legacy Codex records should use cumulative semantics: %#v", got)
	}
}

func TestAggregateUsageRejectsCumulativeSnapshotWithoutSession(t *testing.T) {
	event := cumulativeUsageEvent(t, "evt_missing_session", 1, 2, "", map[string]any{
		"input_tokens": 10, "total_tokens": 10,
	})
	got := AggregateUsage([]MemberEvent{event})
	if got.UsageRecordCount != 1 || got.UsageAvailableCount != 0 || got.UsageUnavailableCount != 1 || !got.UsagePartial {
		t.Fatalf("sessionless cumulative usage must be unavailable: %#v", got)
	}
}

func TestAggregateUsageMalformedProviderUsageIsUnavailable(t *testing.T) {
	got := AggregateUsage([]MemberEvent{{
		Event: Event{EventID: "evt_bad_usage", Payload: []byte(`{"agent_usage":{"provider_usage":{"input_tokens":"10","total_tokens":15}}}`)},
	}})
	if got.UsageRecordCount != 1 || got.UsageAvailableCount != 0 || got.UsageUnavailableCount != 1 || !got.UsagePartial {
		t.Fatalf("malformed usage should be retained as unavailable partial aggregate: %#v", got)
	}
	if got.TotalTokens != 0 || got.AggregationVersion != UsageAggregationVersion {
		t.Fatalf("malformed usage should not create complete-looking totals: %#v", got)
	}
}

func TestAggregateUsageInvalidTokenRecordIsUnavailable(t *testing.T) {
	tests := []struct {
		name    string
		events  []MemberEvent
		wantSum int64
	}{
		{
			name: "negative token",
			events: []MemberEvent{
				{Event: Event{EventID: "evt_good", Payload: jsonPayload(t, map[string]any{
					"agent_usage": map[string]any{"provider_usage": map[string]any{"input_tokens": 4, "total_tokens": 4}},
				})}},
				{Event: Event{EventID: "evt_negative", Payload: jsonPayload(t, map[string]any{
					"agent_usage": map[string]any{"provider_usage": map[string]any{"input_tokens": -1, "total_tokens": 10}},
				})}},
			},
			wantSum: 4,
		},
		{
			name: "overflow token",
			events: []MemberEvent{
				{Event: Event{EventID: "evt_max", Payload: jsonPayload(t, map[string]any{
					"agent_usage": map[string]any{"provider_usage": map[string]any{"input_tokens": math.MaxInt64, "total_tokens": math.MaxInt64}},
				})}},
				{Event: Event{EventID: "evt_overflow", Payload: jsonPayload(t, map[string]any{
					"agent_usage": map[string]any{"provider_usage": map[string]any{"input_tokens": 1, "total_tokens": 1}},
				})}},
			},
			wantSum: math.MaxInt64,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AggregateUsage(tc.events)
			if got.UsageRecordCount != 2 || got.UsageAvailableCount != 1 || got.UsageUnavailableCount != 1 || !got.UsagePartial {
				t.Fatalf("invalid record should be unavailable partial aggregate: %#v", got)
			}
			if got.InputTokens != tc.wantSum || got.TotalTokens != tc.wantSum {
				t.Fatalf("invalid record should not change prior totals: %#v", got)
			}
		})
	}
}

func cumulativeUsageEvent(t *testing.T, eventID string, sequence int64, schemaVersion int, sessionID string, provider map[string]any) MemberEvent {
	if schemaVersion >= 2 {
		provider["scope"] = "session_cumulative"
	}
	return MemberEvent{Event: Event{
		EventID:  eventID,
		Sequence: sequence,
		Payload: jsonPayload(t, map[string]any{"agent_usage": map[string]any{
			"schema_version": schemaVersion,
			"usage_source":   "codex_jsonl_turn_completed",
			"session":        map[string]any{"agent_session_id": sessionID},
			"provider_usage": provider,
		}}),
	}}
}

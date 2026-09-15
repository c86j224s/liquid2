package reportrun

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reportusage"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
)

type recoveryTestStore struct {
	completionTestStore
	byMission  map[string][]ledger.Event
	listErrors map[string]error
	missions   []mission.Mission
	listErr    error
}

func (s *recoveryTestStore) ListEvents(ctx context.Context, missionID string) ([]ledger.Event, error) {
	if err := s.listErrors[missionID]; err != nil {
		return nil, err
	}
	if s.byMission != nil {
		return append([]ledger.Event(nil), s.byMission[missionID]...), nil
	}
	return s.completionTestStore.ListEvents(ctx, missionID)
}

func (s *recoveryTestStore) AppendEventsConditionally(ctx context.Context, missionID string, build func([]ledger.Event) ([]ledger.AppendRequest, error)) ([]ledger.Event, error) {
	if s.byMission == nil {
		return s.completionTestStore.AppendEventsConditionally(ctx, missionID, build)
	}
	local := completionTestStore{events: s.byMission[missionID]}
	out, err := local.AppendEventsConditionally(ctx, missionID, build)
	s.byMission[missionID] = local.events
	return out, err
}

func (s *recoveryTestStore) ListMissionsWithState(context.Context, mission.ListRequest) ([]mission.Mission, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.missions, nil
}
func recoveryEvents() []ledger.Event {
	return []ledger.Event{{EventID: "evt_root", MissionID: "mis_recovery", EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "u"}, Payload: []byte(`{"retry_strategy":"initial"}`)}, {EventID: "evt_target", MissionID: "mis_recovery", EventType: "report.requirements.mapped", Producer: ledger.Producer{Type: "agent_session", ID: "ses"}, Payload: []byte(`{"pending_event_id":"evt_root","previous_provider_session_id":"ses"}`)}, {EventID: "evt_final", MissionID: "mis_recovery", EventType: "report.artifact.created", Producer: ledger.Producer{Type: "agent", ID: "a"}, Payload: []byte(`{"pending_event_id":"evt_root","artifact_id":"art"}`)}}
}
func TestRecoverMissionSyntheticOutcomeHasExactEnvelopeAndReplay(t *testing.T) {
	s := &recoveryTestStore{completionTestStore: completionTestStore{events: recoveryEvents()}}
	changed, err := RecoverMission(context.Background(), s, "mis_recovery")
	if err != nil || changed != 1 {
		t.Fatalf("changed=%d err=%v", changed, err)
	}
	usage, ok := eventByID(s.completionTestStore.events, "evt_report_usage_target")
	if !ok {
		t.Fatal("deterministic usage event missing")
	}
	if usage.EventType != reportusage.ReportAgentUsageRecordedEventType || usage.Producer != (ledger.Producer{Type: "agent_session", ID: "ses"}) || usage.CausationEventID != "evt_target" || usage.CorrelationID != "evt_root" {
		t.Fatalf("usage envelope=%#v", usage)
	}
	var up struct {
		AgentUsage struct {
			SchemaVersion    int             `json:"schema_version"`
			ProviderUsage    json.RawMessage `json:"provider_usage"`
			UsageUnavailable bool            `json:"usage_unavailable"`
			Reason           string          `json:"usage_unavailable_reason"`
		} `json:"agent_usage"`
	}
	if err := json.Unmarshal(usage.Payload, &up); err != nil || up.AgentUsage.SchemaVersion != 2 || len(up.AgentUsage.ProviderUsage) != 0 || !up.AgentUsage.UsageUnavailable || up.AgentUsage.Reason != ReportRunCompletionReason {
		t.Fatalf("usage payload=%s", usage.Payload)
	}
	completion, ok := eventByID(s.completionTestStore.events, "evt_report_run_completed_root")
	if !ok || completion.Producer != (ledger.Producer{Type: "system", ID: "report-completion"}) || completion.CausationEventID != "evt_final" || completion.CorrelationID != "evt_root" {
		t.Fatalf("completion=%#v", completion)
	}
	var cp struct {
		Schema      string `json:"schema_version"`
		Targets     int    `json:"delayed_usage_target_count"`
		Recorded    int    `json:"usage_recorded_count"`
		Unavailable int    `json:"usage_unavailable_count"`
	}
	if err := json.Unmarshal(completion.Payload, &cp); err != nil || cp.Schema != ReportRunCompletionSchema || cp.Targets != 1 || cp.Recorded != 0 || cp.Unavailable != 1 {
		t.Fatalf("completion payload=%s", completion.Payload)
	}
	before := len(s.completionTestStore.events)
	changed, err = RecoverMission(context.Background(), s, "mis_recovery")
	if err != nil || changed != 0 || len(s.completionTestStore.events) != before {
		t.Fatalf("replay changed=%d err=%v events=%d->%d", changed, err, before, len(s.completionTestStore.events))
	}
}

func TestRecoverMissionIsIdempotentAndCountsNewCompletionOnly(t *testing.T) {
	s := &recoveryTestStore{completionTestStore: completionTestStore{events: recoveryEvents()}}
	changed, err := RecoverMission(context.Background(), s, "mis_recovery")
	if err != nil || changed != 1 {
		t.Fatalf("changed=%d err=%v", changed, err)
	}
	changed, err = RecoverMission(context.Background(), s, "mis_recovery")
	if err != nil || changed != 0 {
		t.Fatalf("replay changed=%d err=%v", changed, err)
	}
}
func TestRecoverMissionContinuesAfterMalformedRunInSameMission(t *testing.T) {
	missionID := "mis_same_mission"
	events := []ledger.Event{
		{EventID: "evt_first_root", MissionID: missionID, EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "u"}, Payload: []byte(`{"retry_strategy":"initial"}`)},
		{EventID: "evt_first_final", MissionID: missionID, EventType: "report.artifact.created", Producer: ledger.Producer{Type: "agent", ID: "a"}, Payload: []byte(`{"pending_event_id":"evt_first_root","artifact_id":"art_first"}`)},
		{EventID: "evt_report_run_completed_first_root", MissionID: missionID, EventType: ReportRunCompletedEventType, Producer: ledger.Producer{Type: "system", ID: "report-completion"}, Payload: []byte(`{"run_id":"evt_first_root"}`)},
		{EventID: "evt_second_root", MissionID: missionID, EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "u"}, Payload: []byte(`{"retry_strategy":"initial"}`)},
		{EventID: "evt_second_final", MissionID: missionID, EventType: "report.artifact.created", Producer: ledger.Producer{Type: "agent", ID: "a"}, Payload: []byte(`{"pending_event_id":"evt_second_root","artifact_id":"art_second"}`)},
	}
	s := &recoveryTestStore{completionTestStore: completionTestStore{events: events}}
	changed, err := RecoverMission(context.Background(), s, missionID)
	if err == nil || !strings.Contains(err.Error(), "evt_first_root") || changed != 1 {
		t.Fatalf("changed=%d err=%v", changed, err)
	}
	if _, ok := eventByID(s.events, "evt_report_run_completed_second_root"); !ok {
		t.Fatal("later run was not recovered")
	}
}

func TestRecoverMissionChoosesLatestFinalByLedgerSequence(t *testing.T) {
	events := recoveryEvents()
	events = append(events,
		ledger.Event{EventID: "evt_later_final", MissionID: "mis_recovery", Sequence: 20, EventType: "report.artifact.created", Producer: ledger.Producer{Type: "agent", ID: "a"}, Payload: []byte(`{"pending_event_id":"evt_root","artifact_id":"art_later"}`)},
	)
	for i := range events {
		if events[i].Sequence == 0 {
			events[i].Sequence = int64(i + 1)
		}
	}
	s := &recoveryTestStore{completionTestStore: completionTestStore{events: events}}
	if changed, err := RecoverMission(context.Background(), s, "mis_recovery"); err != nil || changed != 1 {
		t.Fatalf("changed=%d err=%v", changed, err)
	}
	completion, ok := eventByID(s.events, "evt_report_run_completed_root")
	if !ok {
		t.Fatal("completion missing")
	}
	var payload struct {
		CanonicalEventID string `json:"canonical_event_id"`
	}
	if err := json.Unmarshal(completion.Payload, &payload); err != nil || payload.CanonicalEventID != "evt_later_final" {
		t.Fatalf("unexpected canonical event: %#v err=%v", payload, err)
	}
}

func TestRecoverAllIncludesArchivedAndContinuesAfterMissionError(t *testing.T) {
	s := &recoveryTestStore{
		byMission:  map[string][]ledger.Event{"mis_recovery": recoveryEvents(), "mis_after_error": recoveryEventsForMission("mis_after_error")},
		listErrors: map[string]error{"mis_bad": context.Canceled},
		missions:   []mission.Mission{{MissionID: "mis_bad", LifecycleState: "active"}, {MissionID: "mis_recovery", LifecycleState: "archived"}, {MissionID: "mis_after_error", LifecycleState: "active"}},
	}
	changed, err := RecoverAll(context.Background(), s)
	if err == nil || !strings.Contains(err.Error(), "mis_bad") || changed != 2 {
		t.Fatalf("changed=%d err=%v", changed, err)
	}
	if _, ok := eventByID(s.byMission["mis_after_error"], "evt_report_run_completed_after_error_root"); !ok {
		t.Fatal("later mission was not recovered")
	}
}

func recoveryEventsForMission(missionID string) []ledger.Event {
	return []ledger.Event{{EventID: "evt_" + strings.TrimPrefix(missionID, "mis_") + "_root", MissionID: missionID, EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "u"}, Payload: []byte(`{"retry_strategy":"initial"}`)}, {EventID: "evt_" + strings.TrimPrefix(missionID, "mis_") + "_final", MissionID: missionID, EventType: "report.artifact.created", Producer: ledger.Producer{Type: "agent", ID: "a"}, Payload: []byte(`{"pending_event_id":"evt_` + strings.TrimPrefix(missionID, "mis_") + `_root","artifact_id":"art"}`)}}
}

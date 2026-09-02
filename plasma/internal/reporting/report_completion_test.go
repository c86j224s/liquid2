package reporting

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

type completionTestStore struct {
	events []ledger.Event
	fail   bool
}

func (s *completionTestStore) ListEvents(context.Context, string) ([]ledger.Event, error) {
	return append([]ledger.Event(nil), s.events...), nil
}
func (s *completionTestStore) AppendEventsConditionally(_ context.Context, mission string, build func([]ledger.Event) ([]ledger.AppendRequest, error)) ([]ledger.Event, error) {
	before := len(s.events)
	reqs, err := build(append([]ledger.Event(nil), s.events...))
	if err != nil || s.fail {
		if err == nil {
			err = errors.New("forced rollback")
		}
		return nil, err
	}
	out := make([]ledger.Event, 0, len(reqs))
	for _, r := range reqs {
		if r.MissionID != mission {
			return nil, errors.New("mission mismatch")
		}
		e := ledger.Event{EventID: r.EventID, MissionID: r.MissionID, EventType: r.EventType, Producer: r.Producer, CausationEventID: r.CausationEventID, CorrelationID: r.CorrelationID, Payload: r.Payload}
		s.events = append(s.events, e)
		out = append(out, e)
	}
	if len(s.events) != before+len(reqs) {
		panic("append")
	}
	return out, nil
}
func baseCompletionEvents() []ledger.Event {
	return []ledger.Event{{EventID: "evt_root", MissionID: "mis_1", EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "u"}, Payload: json.RawMessage(`{"origin_pending_event_id":"evt_root","retry_strategy":"initial"}`)}, {EventID: "evt_target", MissionID: "mis_1", EventType: "report.requirements.mapped", Producer: ledger.Producer{Type: "agent_session", ID: "ses_1"}, Payload: json.RawMessage(`{"pending_event_id":"evt_root","previous_provider_session_id":"ses_1","agent_executor":"codex","agent_model":"m","agent_reasoning_effort":"r"}`)}, {EventID: "evt_art", MissionID: "mis_1", EventType: "report.artifact.created", Producer: ledger.Producer{Type: "agent", ID: "a"}, Payload: json.RawMessage(`{"pending_event_id":"evt_root","artifact_id":"art_1"}`)}}
}
func TestCompleteReportRunDynamicUnavailableAndReplay(t *testing.T) {
	s := completionTestStore{events: baseCompletionEvents()}
	got, err := CompleteReportRun(context.Background(), &s, ReportCompletionRequest{MissionID: "mis_1", CanonicalEventID: "evt_art"})
	if err != nil {
		t.Fatal(err)
	}
	if got.EventID != "evt_report_run_completed_root" || len(s.events) != 5 {
		t.Fatalf("got=%#v events=%d", got, len(s.events))
	}
	replay, err := CompleteReportRun(context.Background(), &s, ReportCompletionRequest{MissionID: "mis_1", CanonicalEventID: "evt_art"})
	if err != nil || replay.EventID != got.EventID {
		t.Fatalf("replay=%#v err=%v", replay, err)
	}
}

func TestCompleteReportRunReplaysOwnCanonicalAfterLaterPatchFinal(t *testing.T) {
	s := completionTestStore{events: baseCompletionEvents()}
	got, err := CompleteReportRun(context.Background(), &s, ReportCompletionRequest{MissionID: "mis_1", CanonicalEventID: "evt_art"})
	if err != nil {
		t.Fatal(err)
	}
	s.events = append(s.events, ledger.Event{EventID: "evt_patch_final", MissionID: "mis_1", EventType: "report.artifact.created", Producer: ledger.Producer{Type: "agent", ID: "a"}, Payload: json.RawMessage(`{"pending_event_id":"evt_root","artifact_id":"art_patch"}`)})
	replay, err := CompleteReportRun(context.Background(), &s, ReportCompletionRequest{MissionID: "mis_1", CanonicalEventID: "evt_patch_final"})
	if err != nil || replay.EventID != got.EventID {
		t.Fatalf("replay=%#v err=%v", replay, err)
	}
}
func TestCompleteReportRunRejectsUnknownActualAndRollsBack(t *testing.T) {
	s := completionTestStore{events: baseCompletionEvents()}
	_, err := CompleteReportRun(context.Background(), &s, ReportCompletionRequest{MissionID: "mis_1", CanonicalEventID: "evt_art", ActualUsage: &ReportAgentUsageRequest{MissionID: "mis_1", CanonicalEventID: "evt_unknown", AgentSessionID: "ses_1", Usage: agentusage.New("p", "codex", "m", "r", "x").WithProviderUsage(agentusage.ProviderUsage{InputTokens: 1}, "p")}})
	if err == nil || len(s.events) != 3 {
		t.Fatalf("unknown actual err=%v events=%d", err, len(s.events))
	}
}
func TestCompleteReportRunConditionalRollback(t *testing.T) {
	s := completionTestStore{events: baseCompletionEvents(), fail: true}
	_, err := CompleteReportRun(context.Background(), &s, ReportCompletionRequest{MissionID: "mis_1", CanonicalEventID: "evt_art"})
	if err == nil || len(s.events) != 3 {
		t.Fatalf("rollback err=%v events=%d", err, len(s.events))
	}
}
func TestCompleteReportRunCountsUnavailableActualAsUnavailable(t *testing.T) {
	s := completionTestStore{events: baseCompletionEvents()}
	usage := agentusage.New("", "codex", "m", "r", "x").WithUnavailable("lost")
	got, err := CompleteReportRun(context.Background(), &s, ReportCompletionRequest{MissionID: "mis_1", CanonicalEventID: "evt_art", ActualUsage: &ReportAgentUsageRequest{MissionID: "mis_1", CanonicalEventID: "evt_target", AgentSessionID: "ses_1", PreviousAgentSessionID: "ses_1", Surface: "report_requirements", Usage: usage}})
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Unavailable int `json:"usage_unavailable_count"`
	}
	_ = json.Unmarshal(got.Payload, &p)
	if p.Unavailable != 1 {
		t.Fatalf("payload=%s", got.Payload)
	}
}

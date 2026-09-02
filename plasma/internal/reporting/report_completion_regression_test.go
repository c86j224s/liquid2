package reporting

import (
	"context"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func TestCompleteReportRunRejectsActualUsageMetadataConflict(t *testing.T) {
	s := completionTestStore{events: baseCompletionEvents()}
	_, err := CompleteReportRun(context.Background(), &s, ReportCompletionRequest{MissionID: "mis_1", CanonicalEventID: "evt_art", ActualUsage: &ReportAgentUsageRequest{MissionID: "mis_1", CanonicalEventID: "evt_target", AgentSessionID: "ses_1", Usage: agentusage.New("", "other-executor", "", "", "").WithProviderUsage(agentusage.ProviderUsage{InputTokens: 1}, "provider")}})
	if err == nil || len(s.events) != 3 {
		t.Fatalf("expected metadata conflict and rollback, err=%v events=%d", err, len(s.events))
	}
}

func TestCompleteReportRunRejectsActualLineageMismatch(t *testing.T) {
	s := completionTestStore{events: baseCompletionEvents()}
	_, err := CompleteReportRun(context.Background(), &s, ReportCompletionRequest{MissionID: "mis_1", CanonicalEventID: "evt_art", ActualUsage: &ReportAgentUsageRequest{MissionID: "mis_1", CanonicalEventID: "evt_target", AgentSessionID: "ses-other", Surface: "report_wrong", Usage: agentusage.New("", "", "", "", "").WithUnavailable("lost")}})
	if err == nil || len(s.events) != 3 {
		t.Fatalf("expected lineage mismatch and rollback, err=%v events=%d", err, len(s.events))
	}
}

func TestCompleteReportRunRejectsEmptyCanonicalArtifactID(t *testing.T) {
	events := baseCompletionEvents()
	events[2].Payload = []byte(`{"pending_event_id":"evt_root"}`)
	s := completionTestStore{events: events}
	if _, err := CompleteReportRun(context.Background(), &s, ReportCompletionRequest{MissionID: "mis_1", CanonicalEventID: "evt_art"}); err == nil {
		t.Fatal("empty canonical artifact ID accepted")
	}
}

func ledgerEvent(id, eventType string, payload []byte) ledger.Event {
	return ledger.Event{EventID: id, MissionID: "mis_recovery", EventType: eventType, Producer: ledger.Producer{Type: "system", ID: "report-completion"}, CausationEventID: "evt_final", CorrelationID: "evt_root", Payload: payload}
}

func TestRecoverMissionRejectsMalformedExistingCompletion(t *testing.T) {
	events := recoveryEvents()
	events = append(events, ledgerEvent("evt_report_run_completed_root", "report.run.completed", []byte(`{"kind":"report_run_completed"}`)))
	s := &recoveryTestStore{completionTestStore: completionTestStore{events: events}}
	if _, err := RecoverMission(context.Background(), s, "mis_recovery"); err == nil {
		t.Fatal("malformed existing completion accepted")
	}
}

package reportexecution

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func retryPending(id, origin, parent, strategy string) ledger.Event {
	payload, _ := json.Marshal(map[string]any{"origin_pending_event_id": origin, "retry_of_pending_event_id": parent, "retry_strategy": strategy})
	return ledger.Event{EventID: id, MissionID: "mis_1", EventType: "report.draft.pending", Payload: payload}
}

func TestReportRecoveryLineageIncludesAllAncestors(t *testing.T) {
	events := []ledger.Event{retryPending("evt_root", "evt_root", "", "initial"), retryPending("evt_one", "evt_root", "evt_root", "resume_failed"), retryPending("evt_two", "evt_root", "evt_one", "resume_failed")}
	lineage, err := ReportRecoveryLineage(events, "evt_two")
	if err != nil {
		t.Fatal(err)
	}
	if len(lineage) != 3 || lineage[0] != "evt_root" || lineage[2] != "evt_two" {
		t.Fatalf("unexpected lineage: %#v", lineage)
	}
}

func TestReportRecoveryLineageRejectsCycle(t *testing.T) {
	events := []ledger.Event{retryPending("evt_one", "evt_one", "evt_two", "resume_failed"), retryPending("evt_two", "evt_one", "evt_one", "resume_failed")}
	if _, err := ReportRecoveryLineage(events, "evt_one"); err == nil {
		t.Fatal("expected cycle rejection")
	}
}

func TestReportRecoveryLineageRejectsMissingAncestorAndOriginMismatch(t *testing.T) {
	if _, err := ReportRecoveryLineage([]ledger.Event{retryPending("evt_retry", "evt_root", "evt_missing", "resume_failed")}, "evt_retry"); err == nil {
		t.Fatal("expected missing ancestor")
	}
	events := []ledger.Event{retryPending("evt_root", "evt_root", "", "initial"), retryPending("evt_retry", "evt_other", "evt_root", "resume_failed")}
	if _, err := ReportRecoveryLineage(events, "evt_retry"); err == nil {
		t.Fatal("expected origin mismatch")
	}
}

func TestReportRecoveryLineageRestartIsIsolated(t *testing.T) {
	events := []ledger.Event{retryPending("evt_root", "evt_root", "", "initial"), retryPending("evt_restart", "evt_root", "evt_root", "restart")}
	lineage, err := ReportRecoveryLineage(events, "evt_restart")
	if err != nil {
		t.Fatal(err)
	}
	if len(lineage) != 1 || lineage[0] != "evt_restart" {
		t.Fatalf("restart reused ancestor: %#v", lineage)
	}
}

func TestReportRecoveryLineageRestartBoundsDescendantResume(t *testing.T) {
	events := []ledger.Event{retryPending("evt_a", "evt_a", "", "initial"), retryPending("evt_b", "evt_a", "evt_a", "restart"), retryPending("evt_c", "evt_a", "evt_b", "resume_failed")}
	lineage, err := ReportRecoveryLineage(events, "evt_c")
	if err != nil {
		t.Fatal(err)
	}
	if len(lineage) != 2 || lineage[0] != "evt_b" || lineage[1] != "evt_c" {
		t.Fatalf("restart boundary failed: %#v", lineage)
	}
}

func TestReportRecoveryLineageRejectsMalformedPendingBeforeTargetLookup(t *testing.T) {
	_, err := ReportRecoveryLineage([]ledger.Event{{EventID: "evt_malformed", EventType: "report.draft.pending", Payload: []byte("{")}}, "evt_missing")
	if !errors.Is(err, producterror.ErrInvalidInput) || err.Error() != "invalid input: invalid report attempt" {
		t.Fatalf("error=%v, want stable invalid report attempt error", err)
	}
}

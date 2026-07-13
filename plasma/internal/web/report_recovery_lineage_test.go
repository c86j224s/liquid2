package web

import (
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func retryPending(id, origin, parent, strategy string) app.LedgerEvent {
	payload, _ := json.Marshal(map[string]any{"origin_pending_event_id": origin, "retry_of_pending_event_id": parent, "retry_strategy": strategy})
	return app.LedgerEvent{EventID: id, MissionID: "mis_1", EventType: "report.draft.pending", Payload: payload}
}

func TestReportRecoveryLineageIncludesAllAncestors(t *testing.T) {
	events := []app.LedgerEvent{retryPending("evt_root", "evt_root", "", "initial"), retryPending("evt_one", "evt_root", "evt_root", "resume_failed"), retryPending("evt_two", "evt_root", "evt_one", "resume_failed")}
	lineage, err := reportRecoveryLineage(events, "evt_two")
	if err != nil {
		t.Fatal(err)
	}
	if len(lineage) != 3 || lineage[0] != "evt_root" || lineage[2] != "evt_two" {
		t.Fatalf("unexpected lineage: %#v", lineage)
	}
}

func TestReportRecoveryLineageRejectsCycle(t *testing.T) {
	events := []app.LedgerEvent{retryPending("evt_one", "evt_one", "evt_two", "resume_failed"), retryPending("evt_two", "evt_one", "evt_one", "resume_failed")}
	if _, err := reportRecoveryLineage(events, "evt_one"); err == nil {
		t.Fatal("expected cycle rejection")
	}
}

func TestReportRecoveryLineageRejectsMissingAncestorAndOriginMismatch(t *testing.T) {
	if _, err := reportRecoveryLineage([]app.LedgerEvent{retryPending("evt_retry", "evt_root", "evt_missing", "resume_failed")}, "evt_retry"); err == nil {
		t.Fatal("expected missing ancestor")
	}
	events := []app.LedgerEvent{retryPending("evt_root", "evt_root", "", "initial"), retryPending("evt_retry", "evt_other", "evt_root", "resume_failed")}
	if _, err := reportRecoveryLineage(events, "evt_retry"); err == nil {
		t.Fatal("expected origin mismatch")
	}
}

func TestReportRecoveryLineageRestartIsIsolated(t *testing.T) {
	events := []app.LedgerEvent{retryPending("evt_root", "evt_root", "", "initial"), retryPending("evt_restart", "evt_root", "evt_root", "restart")}
	lineage, err := reportRecoveryLineage(events, "evt_restart")
	if err != nil {
		t.Fatal(err)
	}
	if len(lineage) != 1 || lineage[0] != "evt_restart" {
		t.Fatalf("restart reused ancestor: %#v", lineage)
	}
}

func TestReportRecoveryLineageRestartBoundsDescendantResume(t *testing.T) {
	events := []app.LedgerEvent{retryPending("evt_a", "evt_a", "", "initial"), retryPending("evt_b", "evt_a", "evt_a", "restart"), retryPending("evt_c", "evt_a", "evt_b", "resume_failed")}
	lineage, err := reportRecoveryLineage(events, "evt_c")
	if err != nil {
		t.Fatal(err)
	}
	if len(lineage) != 2 || lineage[0] != "evt_b" || lineage[1] != "evt_c" {
		t.Fatalf("restart boundary failed: %#v", lineage)
	}
}

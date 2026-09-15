package reportexecution

import (
	"github.com/c86j224s/liquid2/plasma/internal/ledger"

	"testing"
)

func retryAttempt(id, origin, parent string) ReportAttempt {
	return ReportAttempt{Event: ledger.Event{EventID: id, MissionID: "mis_1"}, ReportAttemptPayload: ReportAttemptPayload{OriginID: origin, RetryOf: parent, Attempt: 1}}
}

func TestValidateRetryLeafAndLineage(t *testing.T) {
	root := retryAttempt("evt_root", "evt_root", "")
	leaf := retryAttempt("evt_leaf", "evt_root", "evt_root")
	if err := ValidateRetryLeafAndLineage(map[string]ReportAttempt{root.EventID: root, leaf.EventID: leaf}, leaf); err != nil {
		t.Fatal(err)
	}
	child := retryAttempt("evt_child", "evt_root", "evt_leaf")
	if err := ValidateRetryLeafAndLineage(map[string]ReportAttempt{root.EventID: root, leaf.EventID: leaf, child.EventID: child}, leaf); err == nil {
		t.Fatal("expected superseded leaf rejection")
	}
}

func TestValidateRetryLineageRejectsCycleAndOriginMismatch(t *testing.T) {
	first := retryAttempt("evt_one", "evt_one", "evt_two")
	second := retryAttempt("evt_two", "evt_one", "evt_one")
	if err := ValidateRetryLeafAndLineage(map[string]ReportAttempt{first.EventID: first, second.EventID: second}, first); err == nil {
		t.Fatal("expected cycle rejection")
	}
	bad := retryAttempt("evt_bad", "evt_other", "evt_one")
	if err := ValidateRetryLeafAndLineage(map[string]ReportAttempt{first.EventID: retryAttempt("evt_one", "evt_one", ""), bad.EventID: bad}, bad); err == nil {
		t.Fatal("expected origin mismatch rejection")
	}
}

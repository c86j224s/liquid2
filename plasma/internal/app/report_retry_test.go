package app

import "testing"

func retryAttempt(id, origin, parent string) reportAttempt {
	return reportAttempt{LedgerEvent: LedgerEvent{EventID: id, MissionID: "mis_1"}, reportAttemptPayload: reportAttemptPayload{OriginID: origin, RetryOf: parent, Attempt: 1}}
}

func TestValidateRetryLeafAndLineage(t *testing.T) {
	root := retryAttempt("evt_root", "evt_root", "")
	leaf := retryAttempt("evt_leaf", "evt_root", "evt_root")
	if err := validateRetryLeafAndLineage(map[string]reportAttempt{root.EventID: root, leaf.EventID: leaf}, leaf); err != nil {
		t.Fatal(err)
	}
	child := retryAttempt("evt_child", "evt_root", "evt_leaf")
	if err := validateRetryLeafAndLineage(map[string]reportAttempt{root.EventID: root, leaf.EventID: leaf, child.EventID: child}, leaf); err == nil {
		t.Fatal("expected superseded leaf rejection")
	}
}

func TestValidateRetryLineageRejectsCycleAndOriginMismatch(t *testing.T) {
	first := retryAttempt("evt_one", "evt_one", "evt_two")
	second := retryAttempt("evt_two", "evt_one", "evt_one")
	if err := validateRetryLeafAndLineage(map[string]reportAttempt{first.EventID: first, second.EventID: second}, first); err == nil {
		t.Fatal("expected cycle rejection")
	}
	bad := retryAttempt("evt_bad", "evt_other", "evt_one")
	if err := validateRetryLeafAndLineage(map[string]reportAttempt{first.EventID: retryAttempt("evt_one", "evt_one", ""), bad.EventID: bad}, bad); err == nil {
		t.Fatal("expected origin mismatch rejection")
	}
}

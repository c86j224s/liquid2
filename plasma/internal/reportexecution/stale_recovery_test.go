package reportexecution

import (
	"context"
	"encoding/json"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"testing"
)

type staleRecoveryStore struct {
	Service
	events []ledger.Event
	err    error
}

func (s staleRecoveryStore) ListEvents(context.Context, string) ([]ledger.Event, error) {
	return s.events, s.err
}

func TestStaleRecoveryStopsAtNonrecoverableNewestDraft(t *testing.T) {
	runner := Runner{Service: staleRecoveryStore{events: []ledger.Event{
		{EventID: "evt_older", EventType: "report.humanize.pending"},
		{EventID: "evt_newer", EventType: "report.draft.pending", Payload: []byte(`{}`)},
	}}, InFlight: &InFlight{}}
	err := runner.ResumeStaleReportOperations(context.Background(), "mis_one", RecoveryHooks{RecoverHumanizeFinalized: func(context.Context, string, ledger.Event) (bool, error) {
		t.Fatal("continued past newer nonrecoverable draft")
		return false, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
}

type retiredRecoveryStore struct {
	staleRecoveryStore
	terminal []ledger.AppendRequest
}

func (s *retiredRecoveryStore) AppendReportTerminalIfOpen(_ context.Context, _, _ string, requests []ledger.AppendRequest) ([]ledger.Event, bool, error) {
	if len(s.terminal) != 0 {
		return nil, false, nil
	}
	s.terminal = requests
	return []ledger.Event{{EventID: requests[0].EventID}}, true, nil
}

func TestStaleHumanizeRetiresWithoutProviderOrPatchRecovery(t *testing.T) {
	store := &retiredRecoveryStore{staleRecoveryStore: staleRecoveryStore{events: []ledger.Event{{EventID: "evt_pending", EventType: "report.humanize.pending"}}}}
	runner := Runner{Service: store, InFlight: &InFlight{}, GenerateHumanize: func(context.Context, string, HumanizeRequest, string) error {
		t.Fatal("retired provider invoked")
		return nil
	}}
	hooks := RecoveryHooks{RecoverHumanizeFinalized: func(context.Context, string, ledger.Event) (bool, error) {
		t.Fatal("retired patch recovery invoked")
		return false, nil
	}}
	for i := 0; i < 2; i++ {
		if err := runner.ResumeStaleReportOperations(context.Background(), "mis_one", hooks); err != nil {
			t.Fatal(err)
		}
	}
	if len(store.terminal) != 1 || store.terminal[0].EventType != "report.humanize.failed" {
		t.Fatalf("terminal=%+v", store.terminal)
	}
	var payload map[string]any
	if err := json.Unmarshal(store.terminal[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["internal_failure_detail"] != "feature_retired" || payload["preserved_original_markdown"] != true {
		t.Fatalf("payload=%v", payload)
	}
}

package researchproposal

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"strings"
	"testing"
)

func TestApprovalProducerSkipsPayloadAndLedgerReads(t *testing.T) {
	for _, kind := range []string{"user", "steering_chat"} {
		readers := ApprovalReaders{RequireMissionEvent: func(context.Context, string, string) (ledger.Event, error) {
			return ledger.Event{Producer: ledger.Producer{Type: kind}, Payload: []byte(`invalid`)}, nil
		}, ListMissionEvents: func(context.Context, string) ([]ledger.Event, error) {
			t.Fatal("unexpected ledger scan")
			return nil, nil
		}}
		if err := RequireApprovedObject(context.Background(), readers, "mis_one", "evt_created", "clm_one"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMalformedApprovalPrecedesLaterMatchingDecision(t *testing.T) {
	readers := ApprovalReaders{RequireMissionEvent: func(context.Context, string, string) (ledger.Event, error) {
		return ledger.Event{Producer: ledger.Producer{Type: "agent"}, Payload: []byte(`{"proposal_id":"prp_one","claim_id":"clm_one"}`)}, nil
	}, ListMissionEvents: func(context.Context, string) ([]ledger.Event, error) {
		return []ledger.Event{
			{EventType: "proposal.approved", Producer: ledger.Producer{Type: "user"}, Payload: []byte(`invalid`)},
			{EventType: "proposal.approved", Producer: ledger.Producer{Type: "user"}, Payload: []byte(`{"proposal_id":"prp_one","approved_object_ids":["clm_one"]}`)},
		}, nil
	}}
	err := RequireApprovedObject(context.Background(), readers, "mis_one", "evt_created", "clm_one")
	if err == nil || !strings.Contains(err.Error(), "invalid ledger event payload") {
		t.Fatalf("unexpected error: %v", err)
	}
}

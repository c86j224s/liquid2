package researchproposal

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/researchcatalog"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
)

func TestBuildProposalBundleValidatesAndNormalizesRefs(t *testing.T) {
	var calls int
	bundle, err := BuildProposalBundle(context.Background(), func(context.Context, string, string, string) error {
		calls++
		return nil
	}, CreateProposalBundleRequest{
		ProposalID:        " prp_1 ",
		MissionID:         " mis_1 ",
		Title:             "  Save this  ",
		ObjectRefs:        []researchcatalog.ObjectRef{{ObjectKind: researchrecords.EvidenceRecordObjectKind, ObjectID: " evd_1 "}},
		RequestedDecision: " approve ",
		CreatedEventID:    " evt_submit ",
	}, ledger.Event{
		EventID:   "evt_submit",
		MissionID: "mis_1",
		EventType: "proposal.submitted",
		Payload:   json.RawMessage(`{"proposal_id":"prp_1"}`),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || bundle.State != "pending_review" || bundle.Title != "Save this" || bundle.ObjectRefs[0].ObjectID != "evd_1" {
		t.Fatalf("unexpected bundle or lookup count: %#v calls=%d", bundle, calls)
	}
	if bundle.CreatedAt.IsZero() || bundle.UpdatedAt.IsZero() || !bundle.CreatedAt.Equal(bundle.UpdatedAt) {
		t.Fatalf("timestamps not initialized together: %#v", bundle)
	}
}

func TestBuildProposalBundlePendingRefsBypassLookupButNotValidation(t *testing.T) {
	lookup := func(context.Context, string, string, string) error {
		t.Fatal("pending ref unexpectedly looked up")
		return nil
	}
	pending := []researchcatalog.ObjectRef{{ObjectKind: researchrecords.EvidenceRecordObjectKind, ObjectID: "evd_1"}}
	base := CreateProposalBundleRequest{
		ProposalID:        "prp_1",
		MissionID:         "mis_1",
		Title:             "Proposal",
		RequestedDecision: "approve",
		CreatedEventID:    "evt_submit",
	}
	event := ledger.Event{EventID: "evt_submit", MissionID: "mis_1", EventType: "proposal.submitted", Payload: json.RawMessage(`{"proposal_id":"prp_1"}`)}

	if _, err := BuildProposalBundle(context.Background(), lookup, withRefs(base, pending), event, pending); err != nil {
		t.Fatalf("pending ref should build: %v", err)
	}
	cases := []struct {
		name string
		refs []researchcatalog.ObjectRef
	}{
		{"empty id", []researchcatalog.ObjectRef{{ObjectKind: researchrecords.EvidenceRecordObjectKind}}},
		{"unsupported kind", []researchcatalog.ObjectRef{{ObjectKind: "unknown", ObjectID: "obj_1"}}},
		{"duplicate", []researchcatalog.ObjectRef{{ObjectKind: researchrecords.EvidenceRecordObjectKind, ObjectID: "evd_1"}, {ObjectKind: researchrecords.EvidenceRecordObjectKind, ObjectID: " evd_1 "}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildProposalBundle(context.Background(), lookup, withRefs(base, tc.refs), event, pending)
			if !errors.Is(err, producterror.ErrInvalidInput) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func withRefs(req CreateProposalBundleRequest, refs []researchcatalog.ObjectRef) CreateProposalBundleRequest {
	req.ObjectRefs = refs
	return req
}

func TestProposalDecisionValidationAndTimestamps(t *testing.T) {
	bundle := ProposalBundle{
		ProposalID: "prp_1",
		State:      "pending_review",
		ObjectRefs: []researchcatalog.ObjectRef{
			{ObjectKind: researchrecords.EvidenceRecordObjectKind, ObjectID: "evd_1"},
			{ObjectKind: researchrecords.ClaimRecordObjectKind, ObjectID: "clm_1"},
		},
	}
	tests := []struct {
		name      string
		target    string
		eventType string
		payload   string
		wantErr   bool
	}{
		{"approved", "approved", "proposal.approved", `{"proposal_id":"prp_1","approved_object_ids":["evd_1","clm_1"],"rejected_object_ids":[]}`, false},
		{"rejected", "rejected", "proposal.rejected", `{"proposal_id":"prp_1","approved_object_ids":[],"rejected_object_ids":["evd_1","clm_1"]}`, false},
		{"partial", "partially_approved", "proposal.partially_approved", `{"proposal_id":"prp_1","approved_object_ids":["evd_1"],"rejected_object_ids":["clm_1"]}`, false},
		{"withdrawn", "withdrawn", "proposal.withdrawn", `{"proposal_id":"prp_1"}`, false},
		{"overlap", "partially_approved", "proposal.partially_approved", `{"proposal_id":"prp_1","approved_object_ids":["evd_1"],"rejected_object_ids":["evd_1","clm_1"]}`, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := ledger.Event{EventType: tc.eventType, Producer: ledger.Producer{Type: "user", ID: "u_1"}, Payload: json.RawMessage(tc.payload)}
			update, err := BuildStateUpdate(UpdateProposalBundleStateRequest{ProposalID: " prp_1 ", State: tc.target, DecisionEventID: " evt_decision "}, bundle, event, tc.target)
			if tc.wantErr {
				if !errors.Is(err, producterror.ErrInvalidInput) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil || update.FromState != "pending_review" || update.ToState != tc.target || update.DecisionEventID != "evt_decision" || update.UpdatedAt.IsZero() {
				t.Fatalf("unexpected update: %#v err=%v", update, err)
			}
		})
	}
	zeroEvent := ledger.Event{EventType: "proposal.approved", Producer: ledger.Producer{Type: "user", ID: "u_1"}, Payload: json.RawMessage(`{"proposal_id":"prp_1","approved_object_ids":["evd_1","clm_1"],"rejected_object_ids":[]}`)}
	update, err := BuildStateUpdate(UpdateProposalBundleStateRequest{ProposalID: "prp_1", State: "approved", DecisionEventID: "evt_1"}, bundle, zeroEvent, "approved")
	if err != nil || update.DecidedAt.IsZero() || time.Since(update.DecidedAt) > time.Second {
		t.Fatalf("zero event timestamp fallback failed: %#v err=%v", update, err)
	}
}

func TestValidateTransitionOnlyAllowsPendingReview(t *testing.T) {
	for _, state := range []string{"approved", "partially_approved", "rejected", "withdrawn"} {
		if err := ValidateTransition(ProposalBundle{State: "pending_review"}, state); err != nil {
			t.Errorf("pending_review -> %s: %v", state, err)
		}
	}
	if err := ValidateTransition(ProposalBundle{State: "approved"}, "rejected"); !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("terminal transition error = %v", err)
	}
}

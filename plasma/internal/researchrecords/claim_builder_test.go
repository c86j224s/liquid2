package researchrecords

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func TestBuildClaimRecordRejectsEarlyValidationWithoutCallbacks(t *testing.T) {
	calls := 0
	requirements := ClaimRequirements{
		RequireEvidenceRecords: func(context.Context, string, []string) error { calls++; return nil },
		RequireQuestionRecords: func(context.Context, string, []string) error { calls++; return nil },
		RequireMissionEvent:    func(context.Context, string, string) (ledger.Event, error) { calls++; return ledger.Event{}, nil },
	}
	_, err := BuildClaimRecord(context.Background(), requirements, CreateClaimRecordRequest{
		ClaimID: "bad", MissionID: "mis_1", Text: "claim", CreatedEventID: "evt_create",
	}, ledger.Event{MissionID: "mis_1", EventID: "evt_create", EventType: "claim.proposed"})
	if err == nil || calls != 0 {
		t.Fatalf("expected early validation with no callbacks, err=%v calls=%d", err, calls)
	}
}

func TestBuildClaimRecordValidUserAssertionWithEmptyRefsUsesCallbacks(t *testing.T) {
	var calls []string
	sentinel := errors.New("empty evidence callback sentinel")
	requirements := ClaimRequirements{
		RequireEvidenceRecords: func(_ context.Context, missionID string, ids []string) error {
			calls = append(calls, "evidence:"+missionID+":"+strings.Join(ids, ","))
			if len(ids) == 0 {
				return sentinel
			}
			return nil
		},
		RequireQuestionRecords: func(_ context.Context, missionID string, ids []string) error {
			calls = append(calls, "questions:"+missionID+":"+strings.Join(ids, ","))
			return nil
		},
		RequireMissionEvent: func(_ context.Context, missionID, eventID string) (ledger.Event, error) {
			calls = append(calls, "event:"+missionID+":"+eventID)
			return ledger.Event{MissionID: missionID, EventID: eventID, Producer: ledger.Producer{Type: "user", ID: "u_1"}}, nil
		},
	}
	_, err := BuildClaimRecord(context.Background(), requirements, CreateClaimRecordRequest{
		ClaimID: "clm_1", MissionID: "mis_1", Text: "claim", CreatedEventID: "evt_create", UserAssertionEventID: "evt_assertion",
	}, ledger.Event{MissionID: "mis_1", EventID: "evt_create", EventType: "claim.proposed", Payload: json.RawMessage(`{"claim_id":"clm_1","proposal_id":"prp_1"}`)})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected empty evidence callback sentinel, got %v", err)
	}
	wantCalls := []string{"evidence:mis_1:"}
	if strings.Join(calls, "|") != strings.Join(wantCalls, "|") {
		t.Fatalf("callback order mismatch: got=%v want=%v", calls, wantCalls)
	}

	calls = nil
	requirements.RequireEvidenceRecords = func(_ context.Context, missionID string, ids []string) error {
		calls = append(calls, "evidence:"+missionID+":"+strings.Join(ids, ","))
		return nil
	}
	record, err := BuildClaimRecord(context.Background(), requirements, CreateClaimRecordRequest{
		ClaimID: "clm_1", MissionID: "mis_1", Text: "claim", CreatedEventID: "evt_create", UserAssertionEventID: "evt_assertion",
	}, ledger.Event{MissionID: "mis_1", EventID: "evt_create", EventType: "claim.proposed", Payload: json.RawMessage(`{"claim_id":"clm_1","proposal_id":"prp_1"}`)})
	if err != nil {
		t.Fatalf("BuildClaimRecord returned error: %v", err)
	}
	wantCalls = []string{"evidence:mis_1:", "questions:mis_1:", "event:mis_1:evt_assertion"}
	if strings.Join(calls, "|") != strings.Join(wantCalls, "|") {
		t.Fatalf("callback order mismatch: got=%v want=%v", calls, wantCalls)
	}
	if record.ClaimID != "clm_1" || record.MissionID != "mis_1" || record.UserAssertionEventID != "evt_assertion" || record.State != "proposed" {
		t.Fatalf("unexpected claim record: %#v", record)
	}
}

func TestBuildClaimRecordOrdersEvidenceQuestionsAssertionAndApproval(t *testing.T) {
	var calls []string
	approvalAt := time.Date(2026, 9, 7, 12, 30, 0, 0, time.FixedZone("PDT", -7*60*60))
	requirements := ClaimRequirements{
		RequireEvidenceRecords: func(_ context.Context, missionID string, ids []string) error {
			calls = append(calls, "evidence:"+missionID+":"+strings.Join(ids, ","))
			return nil
		},
		RequireQuestionRecords: func(_ context.Context, missionID string, ids []string) error {
			calls = append(calls, "questions:"+missionID+":"+strings.Join(ids, ","))
			return nil
		},
		RequireMissionEvent: func(_ context.Context, missionID, eventID string) (ledger.Event, error) {
			calls = append(calls, "event:"+missionID+":"+eventID)
			switch eventID {
			case "evt_assertion":
				return ledger.Event{MissionID: missionID, EventID: eventID, Producer: ledger.Producer{Type: "user", ID: "u_1"}}, nil
			case "evt_approval":
				return ledger.Event{MissionID: missionID, EventID: eventID, EventType: "claim.approved", Producer: ledger.Producer{Type: "user", ID: "u_1"}, CreatedAt: approvalAt, Payload: json.RawMessage(`{"claim_id":"clm_1"}`)}, nil
			default:
				return ledger.Event{}, nil
			}
		},
	}
	createdEvent := ledger.Event{MissionID: "mis_1", EventID: "evt_create", EventType: "claim.proposed", Payload: json.RawMessage(`{"claim_id":"clm_1","proposal_id":"prp_1"}`)}
	record, err := BuildClaimRecord(context.Background(), requirements, CreateClaimRecordRequest{
		ClaimID: "clm_1", MissionID: "mis_1", Text: "  claim text  ", CreatedEventID: "evt_create",
		SupportingEvidenceIDs: []string{" evd_1 "}, DependsOnQuestionIDs: []string{" qst_1 "},
		UserAssertionEventID: "evt_assertion", Approval: ClaimApproval{State: "approved", ApprovalEventID: "evt_approval"},
	}, createdEvent)
	if err != nil {
		t.Fatalf("BuildClaimRecord returned error: %v", err)
	}
	wantCalls := []string{"evidence:mis_1:evd_1", "questions:mis_1:qst_1", "event:mis_1:evt_assertion", "event:mis_1:evt_approval"}
	if strings.Join(calls, "|") != strings.Join(wantCalls, "|") {
		t.Fatalf("callback order mismatch: got=%v want=%v", calls, wantCalls)
	}
	if record.State != "approved" || record.Approval.ApprovedAt != approvalAt || record.Confidence.Level != "unknown" || record.ClaimType != "descriptive" {
		t.Fatalf("unexpected normalized claim: %#v", record)
	}
	if record.CreatedAt.IsZero() || record.CreatedAt.Location() != time.UTC || record.Text != "claim text" {
		t.Fatalf("unexpected creation metadata: %#v", record)
	}
}

func TestBuildClaimRecordRejectsDuplicateRefsAndApprovalMembershipErrors(t *testing.T) {
	base := CreateClaimRecordRequest{ClaimID: "clm_1", MissionID: "mis_1", Text: "claim", CreatedEventID: "evt_create", SupportingEvidenceIDs: []string{"evd_1", " evd_1"}}
	requirements := noOpClaimRequirements()
	created := claimProposedEvent("clm_1", "prp_1")
	if _, err := BuildClaimRecord(context.Background(), requirements, base, created); err == nil {
		t.Fatal("expected duplicate reference error")
	}

	for _, tc := range []struct {
		name    string
		payload string
	}{
		{name: "wrong proposal", payload: `{"proposal_id":"prp_other","approved_object_ids":["clm_1"]}`},
		{name: "rejected membership", payload: `{"proposal_id":"prp_1","approved_object_ids":["clm_1"],"rejected_object_ids":["clm_1"]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reqs := noOpClaimRequirements()
			reqs.RequireMissionEvent = func(_ context.Context, _, eventID string) (ledger.Event, error) {
				if eventID == "evt_approval" {
					return ledger.Event{EventID: eventID, MissionID: "mis_1", EventType: "proposal.approved", Producer: ledger.Producer{Type: "user", ID: "u_1"}, Payload: json.RawMessage(tc.payload)}, nil
				}
				return ledger.Event{EventID: eventID, MissionID: "mis_1"}, nil
			}
			req := CreateClaimRecordRequest{ClaimID: "clm_1", MissionID: "mis_1", Text: "claim", CreatedEventID: "evt_create", SupportingEvidenceIDs: []string{"evd_1"}, Approval: ClaimApproval{State: "approved", ApprovalEventID: "evt_approval"}}
			if _, err := BuildClaimRecord(context.Background(), reqs, req, created); err == nil {
				t.Fatal("expected proposal approval membership error")
			}
		})
	}
}

func noOpClaimRequirements() ClaimRequirements {
	return ClaimRequirements{
		RequireEvidenceRecords: func(context.Context, string, []string) error { return nil },
		RequireQuestionRecords: func(context.Context, string, []string) error { return nil },
		RequireMissionEvent: func(_ context.Context, missionID, eventID string) (ledger.Event, error) {
			return ledger.Event{MissionID: missionID, EventID: eventID}, nil
		},
	}
}

func claimProposedEvent(claimID, proposalID string) ledger.Event {
	payload, _ := json.Marshal(map[string]string{"claim_id": claimID, "proposal_id": proposalID})
	return ledger.Event{MissionID: "mis_1", EventID: "evt_create", EventType: "claim.proposed", Payload: payload}
}

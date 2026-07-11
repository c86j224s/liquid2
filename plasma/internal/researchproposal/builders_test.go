package researchproposal

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestBuildProposalProposedAppendRequestsPreservePayloadContracts(t *testing.T) {
	evidence := BuildEvidenceProposedAppendRequest(EvidenceProposedEventRequest{
		EventID:    "evt_evidence",
		MissionID:  "mis_1",
		EvidenceID: "evd_1",
		ProposalID: "prp_1",
		Producer:   Producer{Type: "agent_session", ID: "ses_agent"},
	})
	if evidence.EventType != "evidence.proposed" || evidence.Producer.ID != "ses_agent" {
		t.Fatalf("unexpected evidence event: %#v", evidence)
	}
	assertPayloadMap(t, evidence.Payload, map[string]any{"evidence_id": "evd_1", "proposal_id": "prp_1"})

	manual := BuildEvidenceProposedAppendRequest(EvidenceProposedEventRequest{
		EventID:    "evt_manual",
		MissionID:  "mis_1",
		EvidenceID: "evd_manual",
		ProposalID: "prp_manual",
		Source:     "manual_candidate",
		Producer:   Producer{Type: "user", ID: "plasma-ui"},
	})
	assertPayloadMap(t, manual.Payload, map[string]any{"evidence_id": "evd_manual", "proposal_id": "prp_manual", "source": "manual_candidate"})

	question := BuildQuestionProposedAppendRequest(QuestionProposedEventRequest{
		EventID:    "evt_question",
		MissionID:  "mis_1",
		QuestionID: "qst_1",
		ProposalID: "prp_2",
		Producer:   Producer{Type: "agent_session", ID: "ses_agent"},
	})
	if question.EventType != "question.proposed" {
		t.Fatalf("unexpected question event: %#v", question)
	}
	assertPayloadMap(t, question.Payload, map[string]any{"question_id": "qst_1", "proposal_id": "prp_2"})

	claim := BuildClaimProposedAppendRequest(ClaimProposedEventRequest{
		EventID:    "evt_claim",
		MissionID:  "mis_1",
		ClaimID:    "clm_1",
		ProposalID: "prp_3",
		Producer:   Producer{Type: "agent_session", ID: "ses_agent"},
	})
	if claim.EventType != "claim.proposed" {
		t.Fatalf("unexpected claim event: %#v", claim)
	}
	assertPayloadMap(t, claim.Payload, map[string]any{"claim_id": "clm_1", "proposal_id": "prp_3"})
}

func TestBuildProposalSubmittedPreservesMCPAndManualPayloadContracts(t *testing.T) {
	mcp := BuildProposalSubmitted(ProposalSubmittedRequest{
		EventID:    "evt_proposal",
		MissionID:  "mis_1",
		ProposalID: "prp_1",
		ObjectRefs: []ObjectRef{
			{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: "evd_1"},
		},
		Producer:                   Producer{Type: "agent_session", ID: "ses_agent"},
		IncludeObjectRefsInPayload: true,
	})
	if mcp.Event.EventType != "proposal.submitted" || mcp.Bundle.State != "pending_review" ||
		mcp.Bundle.Title != "Investigation proposal" || mcp.Bundle.RequestedDecision != "approve" {
		t.Fatalf("unexpected MCP proposal build: %#v", mcp)
	}
	var mcpPayload struct {
		ProposalID string          `json:"proposal_id"`
		ObjectRefs []app.ObjectRef `json:"object_refs"`
	}
	if err := json.Unmarshal(mcp.Event.Payload, &mcpPayload); err != nil {
		t.Fatalf("unmarshal MCP proposal payload: %v", err)
	}
	if mcpPayload.ProposalID != "prp_1" || len(mcpPayload.ObjectRefs) != 1 || mcpPayload.ObjectRefs[0].ObjectID != "evd_1" {
		t.Fatalf("unexpected MCP proposal payload: %#v", mcpPayload)
	}

	manual := BuildProposalSubmitted(ProposalSubmittedRequest{
		EventID:           "evt_manual_proposal",
		MissionID:         "mis_1",
		ProposalID:        "prp_manual",
		Title:             "Save evidence candidate",
		ObjectRefs:        []ObjectRef{{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: "evd_manual"}},
		RequestedDecision: "approve",
		Producer:          Producer{Type: "user", ID: "plasma-ui"},
	})
	if manual.Bundle.Title != "Save evidence candidate" {
		t.Fatalf("unexpected manual proposal title: %#v", manual.Bundle)
	}
	assertPayloadMap(t, manual.Event.Payload, map[string]any{"proposal_id": "prp_manual"})
}

func TestBuildProposalDecisionAppendRequestPreservesPayloadContract(t *testing.T) {
	proposal := ProposalBundle{
		ProposalID: "prp_1",
		MissionID:  "mis_1",
		ObjectRefs: []ObjectRef{
			{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: "evd_1"},
			{ObjectKind: app.ClaimRecordObjectKind, ObjectID: "clm_1"},
		},
	}
	approved, nextState := BuildProposalDecisionAppendRequest(ProposalDecisionAppendRequest{
		EventID:  "evt_approved",
		Proposal: proposal,
		Action:   "approve",
		Producer: Producer{Type: "user", ID: "plasma-ui"},
	})
	if nextState != "approved" || approved.EventID != "evt_approved" || approved.MissionID != "mis_1" ||
		approved.EventType != "proposal.approved" || approved.Producer.Type != "user" || approved.Producer.ID != "plasma-ui" {
		t.Fatalf("unexpected approval event request: next=%q req=%#v", nextState, approved)
	}
	assertPayloadMap(t, approved.Payload, map[string]any{
		"proposal_id":         "prp_1",
		"approved_object_ids": []any{"evd_1", "clm_1"},
		"rejected_object_ids": []any{},
	})

	rejected, nextState := BuildProposalDecisionAppendRequest(ProposalDecisionAppendRequest{
		EventID:  "evt_rejected",
		Proposal: proposal,
		Action:   "reject",
		Producer: Producer{Type: "user", ID: "plasma-ui"},
	})
	if nextState != "rejected" || rejected.EventID != "evt_rejected" || rejected.MissionID != "mis_1" ||
		rejected.EventType != "proposal.rejected" || rejected.Producer.Type != "user" || rejected.Producer.ID != "plasma-ui" {
		t.Fatalf("unexpected rejection event request: next=%q req=%#v", nextState, rejected)
	}
	assertPayloadMap(t, rejected.Payload, map[string]any{
		"proposal_id":         "prp_1",
		"approved_object_ids": []any{},
		"rejected_object_ids": []any{"evd_1", "clm_1"},
	})
}

func TestBuildManualEvidenceCandidateProposalRequestPreservesContract(t *testing.T) {
	req := BuildManualEvidenceCandidateProposalRequest(ManualEvidenceCandidateProposalRequest{
		MissionID:       "mis_1",
		EvidenceID:      "evd_manual",
		ProposalID:      "prp_manual",
		EvidenceEventID: "evt_evidence",
		ProposalEventID: "evt_proposal",
		Summary:         "사용자가 선택한 후보 근거입니다.",
		EvidenceType:    "observation",
		SnapshotID:      "src_1",
		ArtifactID:      "art_1",
		Producer:        Producer{Type: "user", ID: "plasma-ui"},
	})
	if req.EvidenceEvent.EventType != "evidence.proposed" || req.ProposalEvent.EventType != "proposal.submitted" {
		t.Fatalf("unexpected manual events: %#v", req)
	}
	assertPayloadMap(t, req.EvidenceEvent.Payload, map[string]any{"evidence_id": "evd_manual", "proposal_id": "prp_manual", "source": "manual_candidate"})
	assertPayloadMap(t, req.ProposalEvent.Payload, map[string]any{"proposal_id": "prp_manual"})
	if req.Evidence.State != "proposed" || req.Evidence.Summary != "사용자가 선택한 후보 근거입니다." ||
		req.Evidence.EvidenceType != "observation" || len(req.Evidence.SnapshotRefs) != 1 ||
		req.Evidence.Confidence.Level != "unknown" || !req.Evidence.Confidence.NeedsVerification {
		t.Fatalf("unexpected manual evidence request: %#v", req.Evidence)
	}
	if string(req.Evidence.SnapshotRefs[0].Locator) != `{"kind":"source_backed_candidate"}` {
		t.Fatalf("unexpected locator: %s", req.Evidence.SnapshotRefs[0].Locator)
	}
	if req.Proposal.Title != "Save evidence candidate" || req.Proposal.RequestedDecision != "approve" ||
		len(req.Proposal.ObjectRefs) != 1 || req.Proposal.ObjectRefs[0].ObjectID != "evd_manual" {
		t.Fatalf("unexpected manual proposal request: %#v", req.Proposal)
	}
}

func assertPayloadMap(t *testing.T, raw []byte, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("payload mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

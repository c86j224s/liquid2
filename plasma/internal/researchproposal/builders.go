package researchproposal

import (
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type Producer = app.Producer
type AppendEventRequest = app.AppendEventRequest
type ObjectRef = app.ObjectRef
type CreateProposalBundleRequest = app.CreateProposalBundleRequest
type CreateEvidenceProposalRequest = app.CreateEvidenceProposalRequest
type CreateEvidenceRecordRequest = app.CreateEvidenceRecordRequest
type SnapshotRef = app.SnapshotRef
type Confidence = app.Confidence
type ProposalBundle = app.ProposalBundle

const EvidenceRecordObjectKind = app.EvidenceRecordObjectKind

type EvidenceProposedEventRequest struct {
	EventID    string
	MissionID  string
	EvidenceID string
	ProposalID string
	Source     string
	Producer   Producer
}

type QuestionProposedEventRequest struct {
	EventID    string
	MissionID  string
	QuestionID string
	ProposalID string
	Producer   Producer
}

type ClaimProposedEventRequest struct {
	EventID    string
	MissionID  string
	ClaimID    string
	ProposalID string
	Producer   Producer
}

type ProposalSubmittedRequest struct {
	EventID                    string
	MissionID                  string
	ProposalID                 string
	Title                      string
	ObjectRefs                 []ObjectRef
	RequestedDecision          string
	Producer                   Producer
	IncludeObjectRefsInPayload bool
}

type ProposalSubmittedBuildResult struct {
	Event  AppendEventRequest
	Bundle CreateProposalBundleRequest
}

type ProposalDecisionAppendRequest struct {
	EventID  string
	Proposal ProposalBundle
	Action   string
	Producer Producer
}

type ManualEvidenceCandidateProposalRequest struct {
	MissionID       string
	EvidenceID      string
	ProposalID      string
	EvidenceEventID string
	ProposalEventID string
	Summary         string
	EvidenceType    string
	SnapshotID      string
	ArtifactID      string
	Producer        Producer
}

func BuildEvidenceProposedAppendRequest(req EvidenceProposedEventRequest) AppendEventRequest {
	payload := map[string]any{
		"evidence_id": req.EvidenceID,
		"proposal_id": req.ProposalID,
	}
	if source := strings.TrimSpace(req.Source); source != "" {
		payload["source"] = source
	}
	return AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "evidence.proposed",
		Producer:  req.Producer,
		Payload:   mustMarshalJSON(payload),
	}
}

func BuildQuestionProposedAppendRequest(req QuestionProposedEventRequest) AppendEventRequest {
	return AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "question.proposed",
		Producer:  req.Producer,
		Payload: mustMarshalJSON(map[string]any{
			"question_id": req.QuestionID,
			"proposal_id": req.ProposalID,
		}),
	}
}

func BuildClaimProposedAppendRequest(req ClaimProposedEventRequest) AppendEventRequest {
	return AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "claim.proposed",
		Producer:  req.Producer,
		Payload: mustMarshalJSON(map[string]any{
			"claim_id":    req.ClaimID,
			"proposal_id": req.ProposalID,
		}),
	}
}

func BuildProposalSubmitted(req ProposalSubmittedRequest) ProposalSubmittedBuildResult {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Investigation proposal"
	}
	requestedDecision := strings.TrimSpace(req.RequestedDecision)
	if requestedDecision == "" {
		requestedDecision = "approve"
	}
	payload := map[string]any{"proposal_id": req.ProposalID}
	if req.IncludeObjectRefsInPayload {
		payload["object_refs"] = req.ObjectRefs
	}
	return ProposalSubmittedBuildResult{
		Event: AppendEventRequest{
			EventID:   req.EventID,
			MissionID: req.MissionID,
			EventType: "proposal.submitted",
			Producer:  req.Producer,
			Payload:   mustMarshalJSON(payload),
		},
		Bundle: CreateProposalBundleRequest{
			ProposalID:        req.ProposalID,
			MissionID:         req.MissionID,
			State:             "pending_review",
			Title:             title,
			ObjectRefs:        req.ObjectRefs,
			RequestedDecision: requestedDecision,
			CreatedEventID:    req.EventID,
		},
	}
}

func BuildProposalDecisionAppendRequest(req ProposalDecisionAppendRequest) (AppendEventRequest, string) {
	proposal := req.Proposal
	objectIDs := objectRefIDs(proposal.ObjectRefs)
	payload := map[string]any{"proposal_id": proposal.ProposalID}
	nextState := "approved"
	eventType := "proposal.approved"
	if req.Action == "reject" {
		nextState = "rejected"
		eventType = "proposal.rejected"
		payload["approved_object_ids"] = []string{}
		payload["rejected_object_ids"] = objectIDs
	} else {
		payload["approved_object_ids"] = objectIDs
		payload["rejected_object_ids"] = []string{}
	}
	return AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(proposal.MissionID),
		EventType: eventType,
		Producer:  req.Producer,
		Payload:   mustMarshalJSON(payload),
	}, nextState
}

func BuildManualEvidenceCandidateProposalRequest(req ManualEvidenceCandidateProposalRequest) CreateEvidenceProposalRequest {
	proposal := BuildProposalSubmitted(ProposalSubmittedRequest{
		EventID:           req.ProposalEventID,
		MissionID:         req.MissionID,
		ProposalID:        req.ProposalID,
		Title:             "Save evidence candidate",
		ObjectRefs:        []ObjectRef{{ObjectKind: EvidenceRecordObjectKind, ObjectID: req.EvidenceID}},
		RequestedDecision: "approve",
		Producer:          req.Producer,
	})
	return CreateEvidenceProposalRequest{
		EvidenceEvent: BuildEvidenceProposedAppendRequest(EvidenceProposedEventRequest{
			EventID:    req.EvidenceEventID,
			MissionID:  req.MissionID,
			EvidenceID: req.EvidenceID,
			ProposalID: req.ProposalID,
			Source:     "manual_candidate",
			Producer:   req.Producer,
		}),
		Evidence: CreateEvidenceRecordRequest{
			EvidenceID:   req.EvidenceID,
			MissionID:    req.MissionID,
			State:        "proposed",
			Summary:      req.Summary,
			EvidenceType: req.EvidenceType,
			SnapshotRefs: []SnapshotRef{{
				SnapshotID: req.SnapshotID,
				ArtifactID: req.ArtifactID,
				Locator:    json.RawMessage(`{"kind":"source_backed_candidate"}`),
			}},
			Confidence: Confidence{
				Level:             "unknown",
				Rationale:         "Manual candidate created from the conversation workspace and linked to a selected source snapshot.",
				NeedsVerification: true,
			},
			Producer:       req.Producer,
			CreatedEventID: req.EvidenceEventID,
		},
		ProposalEvent: proposal.Event,
		Proposal:      proposal.Bundle,
	}
}

func mustMarshalJSON(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

func objectRefIDs(refs []ObjectRef) []string {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		id := strings.TrimSpace(ref.ObjectID)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

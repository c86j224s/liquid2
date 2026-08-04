package researchproposal

import (
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// Producer는 app.Producer의 proposal builder용 alias다.
type Producer = app.Producer

// AppendEventRequest는 proposal builder가 app service에 넘길 event append 요청 alias다.
type AppendEventRequest = app.AppendEventRequest

// ObjectRef는 proposal이 가리키는 후보 객체 참조 alias다.
type ObjectRef = app.ObjectRef

// CreateProposalBundleRequest는 proposal bundle 생성 요청 alias다.
type CreateProposalBundleRequest = app.CreateProposalBundleRequest

// CreateEvidenceProposalRequest는 evidence proposal 생성을 한 번에 넘기는 요청 alias다.
type CreateEvidenceProposalRequest = app.CreateEvidenceProposalRequest

// CreateEvidenceRecordRequest는 evidence record 생성 요청 alias다.
type CreateEvidenceRecordRequest = app.CreateEvidenceRecordRequest

// SnapshotRef는 evidence 후보가 참조하는 source snapshot alias다.
type SnapshotRef = app.SnapshotRef

// Confidence는 proposal 단계 evidence의 신뢰도 설명 alias다.
type Confidence = app.Confidence

// ProposalBundle은 승인/기각 대기 중인 후보 묶음 alias다.
type ProposalBundle = app.ProposalBundle

// EvidenceRecordObjectKind는 evidence record proposal의 object kind wire 값이다.
const EvidenceRecordObjectKind = app.EvidenceRecordObjectKind

// EvidenceProposedEventRequest는 evidence.proposed event builder 입력이다.
type EvidenceProposedEventRequest struct {
	EventID    string
	MissionID  string
	EvidenceID string
	ProposalID string
	Source     string
	Producer   Producer
}

// QuestionProposedEventRequest는 question.proposed event builder 입력이다.
type QuestionProposedEventRequest struct {
	EventID    string
	MissionID  string
	QuestionID string
	ProposalID string
	Producer   Producer
}

// ClaimProposedEventRequest는 claim.proposed event builder 입력이다.
type ClaimProposedEventRequest struct {
	EventID    string
	MissionID  string
	ClaimID    string
	ProposalID string
	Producer   Producer
}

// ProposalSubmittedRequest는 proposal.submitted event와 proposal bundle을 함께 만들기
// 위한 입력이다.
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

// ProposalSubmittedBuildResult는 proposal 제출 시 함께 만들어야 하는 event와 bundle이다.
type ProposalSubmittedBuildResult struct {
	Event  AppendEventRequest
	Bundle CreateProposalBundleRequest
}

// ProposalDecisionAppendRequest는 proposal 승인/기각 event builder 입력이다.
type ProposalDecisionAppendRequest struct {
	EventID  string
	Proposal ProposalBundle
	Action   string
	Producer Producer
}

// ManualEvidenceCandidateProposalRequest는 대화에서 고른 source-backed candidate를
// evidence proposal로 묶을 때의 입력이다.
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

// BuildEvidenceProposedAppendRequest는 evidence.proposed append request를 만든다.
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

// BuildQuestionProposedAppendRequest는 question.proposed append request를 만든다.
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

// BuildClaimProposedAppendRequest는 claim.proposed append request를 만든다.
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

// BuildProposalSubmitted는 proposal.submitted event와 pending proposal bundle을 함께
// 만든다.
//
// 이 결과는 아직 승인된 saved knowledge가 아니다. decision event가 별도로 기록되어야
// proposal state가 바뀐다.
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

// BuildProposalDecisionAppendRequest는 proposal 승인 또는 기각 event와 다음 bundle
// state를 계산한다.
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

// BuildManualEvidenceCandidateProposalRequest는 source snapshot에 근거한 수동 evidence
// 후보를 proposal 흐름에 넣기 위한 복합 요청을 만든다.
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

package app

import (
	"encoding/json"
	"time"
)

const (
	// Evidence/Claim/Question/Option/Proposal schema 상수는 legacy research object의
	// stable JSON identity다.
	EvidenceRecordSchemaVersion = "plasma.evidence_record.v1"
	EvidenceRecordObjectKind    = "evidence_record"
	ClaimRecordSchemaVersion    = "plasma.claim_record.v1"
	ClaimRecordObjectKind       = "claim_record"
	ClaimConfidenceUpdatedEvent = "claim.confidence.updated"
	QuestionRecordSchemaVersion = "plasma.question_record.v1"
	QuestionRecordObjectKind    = "question_record"
	OptionRecordSchemaVersion   = "plasma.option_record.v1"
	OptionRecordObjectKind      = "option_record"
	ProposalBundleSchemaVersion = "plasma.proposal_bundle.v1"
	ProposalBundleObjectKind    = "proposal_bundle"
)

// Confidence는 agent 또는 사용자가 남긴 claim/evidence 신뢰도 설명이다.
//
// 보고서 생성의 gate가 아니라 참고자료와 추적성을 위한 내부 metadata로 취급한다.
type Confidence struct {
	Level             string   `json:"level"`
	Rationale         string   `json:"rationale"`
	OpenRisks         []string `json:"open_risks,omitempty"`
	NeedsVerification bool     `json:"needs_verification"`
}

// Approval은 record나 block이 승인됐는지 여부와 승인 event를 기록한다.
type Approval struct {
	State           string    `json:"state"`
	Required        bool      `json:"required"`
	ApprovalEventID string    `json:"approval_event_id,omitempty"`
	ApprovedAt      time.Time `json:"approved_at,omitempty"`
}

// SnapshotRef는 evidence가 참조하는 source snapshot artifact 위치다.
type SnapshotRef struct {
	SnapshotID string          `json:"snapshot_id"`
	ArtifactID string          `json:"artifact_id"`
	Locator    json.RawMessage `json:"locator"`
}

// ObjectRef는 proposal bundle이 포함하는 후보 객체의 stable reference다.
type ObjectRef struct {
	ObjectKind string `json:"object_kind"`
	ObjectID   string `json:"object_id"`
}

// EvidenceRecord는 source snapshot의 특정 내용에서 뽑힌 evidence 후보/기록이다.
type EvidenceRecord struct {
	SchemaVersion  string        `json:"schema_version"`
	ObjectKind     string        `json:"object_kind"`
	EvidenceID     string        `json:"evidence_id"`
	MissionID      string        `json:"mission_id"`
	State          string        `json:"state"`
	Summary        string        `json:"summary"`
	EvidenceType   string        `json:"evidence_type"`
	SnapshotRefs   []SnapshotRef `json:"snapshot_refs"`
	Confidence     Confidence    `json:"confidence"`
	Producer       Producer      `json:"producer"`
	CreatedEventID string        `json:"created_event_id"`
	CreatedAt      time.Time     `json:"created_at"`
}

// ClaimRecord는 mission 안에서 다루는 주장 또는 중간 결론이다.
type ClaimRecord struct {
	SchemaVersion         string     `json:"schema_version"`
	ObjectKind            string     `json:"object_kind"`
	ClaimID               string     `json:"claim_id"`
	MissionID             string     `json:"mission_id"`
	State                 string     `json:"state"`
	Text                  string     `json:"text"`
	ClaimType             string     `json:"claim_type"`
	SupportingEvidenceIDs []string   `json:"supporting_evidence_ids"`
	OpposingEvidenceIDs   []string   `json:"opposing_evidence_ids"`
	DependsOnQuestionIDs  []string   `json:"depends_on_question_ids"`
	UserAssertionEventID  string     `json:"user_assertion_event_id,omitempty"`
	Confidence            Confidence `json:"confidence"`
	Approval              Approval   `json:"approval"`
	CreatedEventID        string     `json:"created_event_id"`
	CreatedAt             time.Time  `json:"created_at"`
}

// ClaimConfidenceUpdatePayload는 claim confidence 변경 event payload다.
type ClaimConfidenceUpdatePayload struct {
	ClaimID          string     `json:"claim_id"`
	Confidence       Confidence `json:"confidence"`
	BasisEvidenceIDs []string   `json:"basis_evidence_ids,omitempty"`
	Origin           string     `json:"origin"`
}

// ClaimConfidenceUpdate는 장부 event에서 읽은 claim confidence 변경 projection이다.
type ClaimConfidenceUpdate struct {
	EventID          string     `json:"event_id"`
	MissionID        string     `json:"mission_id"`
	Sequence         int64      `json:"sequence"`
	ClaimID          string     `json:"claim_id"`
	Confidence       Confidence `json:"confidence"`
	BasisEvidenceIDs []string   `json:"basis_evidence_ids,omitempty"`
	Origin           string     `json:"origin"`
	Producer         Producer   `json:"producer"`
	CreatedAt        time.Time  `json:"created_at"`
}

// QuestionRecord는 조사 중 남겨 둔 질문 또는 미해결 쟁점이다.
type QuestionRecord struct {
	SchemaVersion      string    `json:"schema_version"`
	ObjectKind         string    `json:"object_kind"`
	QuestionID         string    `json:"question_id"`
	MissionID          string    `json:"mission_id"`
	State              string    `json:"state"`
	Text               string    `json:"text"`
	Priority           string    `json:"priority"`
	Blocking           bool      `json:"blocking"`
	RelatedEvidenceIDs []string  `json:"related_evidence_ids"`
	RelatedClaimIDs    []string  `json:"related_claim_ids"`
	Resolution         string    `json:"resolution,omitempty"`
	CreatedEventID     string    `json:"created_event_id"`
	CreatedAt          time.Time `json:"created_at"`
}

// OptionRecord는 비교/의사결정형 조사에서 검토하는 선택지다.
type OptionRecord struct {
	SchemaVersion      string    `json:"schema_version"`
	ObjectKind         string    `json:"object_kind"`
	OptionID           string    `json:"option_id"`
	MissionID          string    `json:"mission_id"`
	State              string    `json:"state"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Pros               []string  `json:"pros"`
	Cons               []string  `json:"cons"`
	SupportingClaimIDs []string  `json:"supporting_claim_ids"`
	RiskLevel          string    `json:"risk_level"`
	CreatedEventID     string    `json:"created_event_id"`
	CreatedAt          time.Time `json:"created_at"`
}

// ProposalBundle은 evidence/claim/question 후보를 사용자 승인 대상으로 묶은 record다.
type ProposalBundle struct {
	SchemaVersion     string      `json:"schema_version"`
	ObjectKind        string      `json:"object_kind"`
	ProposalID        string      `json:"proposal_id"`
	MissionID         string      `json:"mission_id"`
	State             string      `json:"state"`
	Title             string      `json:"title"`
	ObjectRefs        []ObjectRef `json:"object_refs"`
	RequestedDecision string      `json:"requested_decision"`
	CreatedEventID    string      `json:"created_event_id"`
	DecisionEventID   string      `json:"decision_event_id,omitempty"`
	CreatedAt         time.Time   `json:"created_at"`
	DecidedAt         time.Time   `json:"decided_at,omitempty"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

// CreateEvidenceRecordRequest는 evidence record 생성 입력이다.
type CreateEvidenceRecordRequest struct {
	EvidenceID     string
	MissionID      string
	State          string
	Summary        string
	EvidenceType   string
	SnapshotRefs   []SnapshotRef
	Confidence     Confidence
	Producer       Producer
	CreatedEventID string
}

// CreateClaimRecordRequest는 claim record 생성 입력이다.
type CreateClaimRecordRequest struct {
	ClaimID               string
	MissionID             string
	State                 string
	Text                  string
	ClaimType             string
	SupportingEvidenceIDs []string
	OpposingEvidenceIDs   []string
	DependsOnQuestionIDs  []string
	UserAssertionEventID  string
	Confidence            Confidence
	Approval              Approval
	CreatedEventID        string
}

// UpdateClaimConfidenceRequest는 claim confidence 변경 요청이다.
type UpdateClaimConfidenceRequest struct {
	EventID          string
	MissionID        string
	ClaimID          string
	Confidence       Confidence
	BasisEvidenceIDs []string
	Origin           string
	Producer         Producer
	CausationEventID string
	CorrelationID    string
}

// CreateQuestionRecordRequest는 question record 생성 입력이다.
type CreateQuestionRecordRequest struct {
	QuestionID         string
	MissionID          string
	State              string
	Text               string
	Priority           string
	Blocking           bool
	RelatedEvidenceIDs []string
	RelatedClaimIDs    []string
	Resolution         string
	CreatedEventID     string
}

// CreateOptionRecordRequest는 option record 생성 입력이다.
type CreateOptionRecordRequest struct {
	OptionID           string
	MissionID          string
	State              string
	Title              string
	Description        string
	Pros               []string
	Cons               []string
	SupportingClaimIDs []string
	RiskLevel          string
	CreatedEventID     string
}

// CreateProposalBundleRequest는 proposal bundle 생성 입력이다.
type CreateProposalBundleRequest struct {
	ProposalID        string
	MissionID         string
	State             string
	Title             string
	ObjectRefs        []ObjectRef
	RequestedDecision string
	CreatedEventID    string
}

// UpdateProposalBundleStateRequest는 proposal bundle의 승인/기각 상태 변경 입력이다.
type UpdateProposalBundleStateRequest struct {
	ProposalID      string
	State           string
	DecisionEventID string
}

// ProposalBundleStateUpdate는 proposal state transition 결과다.
type ProposalBundleStateUpdate struct {
	ProposalID      string
	FromState       string
	ToState         string
	DecisionEventID string
	DecidedAt       time.Time
	UpdatedAt       time.Time
}

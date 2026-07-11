package app

import (
	"encoding/json"
	"time"
)

const (
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

type Confidence struct {
	Level             string   `json:"level"`
	Rationale         string   `json:"rationale"`
	OpenRisks         []string `json:"open_risks,omitempty"`
	NeedsVerification bool     `json:"needs_verification"`
}

type Approval struct {
	State           string    `json:"state"`
	Required        bool      `json:"required"`
	ApprovalEventID string    `json:"approval_event_id,omitempty"`
	ApprovedAt      time.Time `json:"approved_at,omitempty"`
}

type SnapshotRef struct {
	SnapshotID string          `json:"snapshot_id"`
	ArtifactID string          `json:"artifact_id"`
	Locator    json.RawMessage `json:"locator"`
}

type ObjectRef struct {
	ObjectKind string `json:"object_kind"`
	ObjectID   string `json:"object_id"`
}

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

type ClaimConfidenceUpdatePayload struct {
	ClaimID          string     `json:"claim_id"`
	Confidence       Confidence `json:"confidence"`
	BasisEvidenceIDs []string   `json:"basis_evidence_ids,omitempty"`
	Origin           string     `json:"origin"`
}

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

type CreateProposalBundleRequest struct {
	ProposalID        string
	MissionID         string
	State             string
	Title             string
	ObjectRefs        []ObjectRef
	RequestedDecision string
	CreatedEventID    string
}

type UpdateProposalBundleStateRequest struct {
	ProposalID      string
	State           string
	DecisionEventID string
}

type ProposalBundleStateUpdate struct {
	ProposalID      string
	FromState       string
	ToState         string
	DecisionEventID string
	DecidedAt       time.Time
	UpdatedAt       time.Time
}

package research

import "github.com/c86j224s/liquid2/plasma/internal/app"

type CommonMutatingInput struct {
	MissionID      string       `json:"mission_id"`
	SessionID      string       `json:"session_id"`
	IdempotencyKey string       `json:"idempotency_key"`
	Producer       app.Producer `json:"producer"`
}

type researchOutlineInput struct {
	MissionID string `json:"mission_id"`
	Legacy    bool   `json:"legacy"`
}

type researchChangesInput struct {
	MissionID     string `json:"mission_id"`
	AfterSequence int64  `json:"after_sequence"`
	Limit         int    `json:"limit"`
}

type researchListInput struct {
	MissionID  string `json:"mission_id"`
	ObjectKind string `json:"object_kind"`
	Limit      int    `json:"limit"`
	Cursor     string `json:"cursor"`
	Legacy     bool   `json:"legacy"`
}

// ReadInput is exported so root tracing can derive request metrics without
// duplicating the research read request shape.
type ReadInput = researchReadInput

type researchReadInput struct {
	MissionID  string `json:"mission_id"`
	ObjectKind string `json:"object_kind"`
	ObjectID   string `json:"object_id"`
	Offset     int    `json:"offset"`
	MaxBytes   int    `json:"max_bytes"`
	Limit      int    `json:"limit"`
	Cursor     string `json:"cursor"`
	Legacy     bool   `json:"legacy"`
}

type researchGrepInput struct {
	MissionID string `json:"mission_id"`
	Query     string `json:"query"`
	Limit     int    `json:"limit"`
	Cursor    string `json:"cursor"`
	Legacy    bool   `json:"legacy"`
}

type researchReferencesInput struct {
	MissionID  string `json:"mission_id"`
	ObjectKind string `json:"object_kind"`
	ObjectID   string `json:"object_id"`
	Limit      int    `json:"limit"`
	Cursor     string `json:"cursor"`
	Legacy     bool   `json:"legacy"`
}

type evidenceProposeInput struct {
	CommonMutatingInput
	EvidenceID      string            `json:"evidence_id"`
	EventID         string            `json:"event_id"`
	ProposalID      string            `json:"proposal_id"`
	ProposalEventID string            `json:"proposal_event_id"`
	ProposalTitle   string            `json:"proposal_title"`
	Summary         string            `json:"summary"`
	EvidenceType    string            `json:"evidence_type"`
	SnapshotRefs    []app.SnapshotRef `json:"snapshot_refs"`
	Confidence      app.Confidence    `json:"confidence"`
}

type questionsProposeInput struct {
	CommonMutatingInput
	QuestionID         string   `json:"question_id"`
	EventID            string   `json:"event_id"`
	ProposalID         string   `json:"proposal_id"`
	ProposalEventID    string   `json:"proposal_event_id"`
	ProposalTitle      string   `json:"proposal_title"`
	Text               string   `json:"text"`
	Priority           string   `json:"priority"`
	Blocking           bool     `json:"blocking"`
	RelatedEvidenceIDs []string `json:"related_evidence_ids"`
	RelatedClaimIDs    []string `json:"related_claim_ids"`
}

type claimsProposeInput struct {
	CommonMutatingInput
	ClaimID               string         `json:"claim_id"`
	EventID               string         `json:"event_id"`
	ProposalID            string         `json:"proposal_id"`
	ProposalEventID       string         `json:"proposal_event_id"`
	ProposalTitle         string         `json:"proposal_title"`
	Text                  string         `json:"text"`
	ClaimType             string         `json:"claim_type"`
	SupportingEvidenceIDs []string       `json:"supporting_evidence_ids"`
	OpposingEvidenceIDs   []string       `json:"opposing_evidence_ids"`
	DependsOnQuestionIDs  []string       `json:"depends_on_question_ids"`
	UserAssertionEventID  string         `json:"user_assertion_event_id"`
	Confidence            app.Confidence `json:"confidence"`
}

type claimConfidenceInput struct {
	CommonMutatingInput
	ClaimID          string         `json:"claim_id"`
	EventID          string         `json:"event_id"`
	Confidence       app.Confidence `json:"confidence"`
	BasisEvidenceIDs []string       `json:"basis_evidence_ids"`
	CausationEventID string         `json:"causation_event_id"`
	CorrelationID    string         `json:"correlation_id"`
}

type proposalsSubmitInput struct {
	CommonMutatingInput
	ProposalID string          `json:"proposal_id"`
	EventID    string          `json:"event_id"`
	Title      string          `json:"title"`
	ObjectRefs []app.ObjectRef `json:"object_refs"`
}

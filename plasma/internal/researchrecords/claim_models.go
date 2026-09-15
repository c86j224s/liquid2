package researchrecords

import (
	"context"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

const (
	ClaimRecordSchemaVersion = "plasma.claim_record.v1"
	ClaimRecordObjectKind    = "claim_record"
)

// ClaimApproval records the approval state attached to a claim record.
type ClaimApproval struct {
	State           string    `json:"state"`
	Required        bool      `json:"required"`
	ApprovalEventID string    `json:"approval_event_id,omitempty"`
	ApprovedAt      time.Time `json:"approved_at,omitempty"`
}

// ClaimRecord is a claim or intermediate conclusion within a mission.
type ClaimRecord struct {
	SchemaVersion         string        `json:"schema_version"`
	ObjectKind            string        `json:"object_kind"`
	ClaimID               string        `json:"claim_id"`
	MissionID             string        `json:"mission_id"`
	State                 string        `json:"state"`
	Text                  string        `json:"text"`
	ClaimType             string        `json:"claim_type"`
	SupportingEvidenceIDs []string      `json:"supporting_evidence_ids"`
	OpposingEvidenceIDs   []string      `json:"opposing_evidence_ids"`
	DependsOnQuestionIDs  []string      `json:"depends_on_question_ids"`
	UserAssertionEventID  string        `json:"user_assertion_event_id,omitempty"`
	Confidence            Confidence    `json:"confidence"`
	Approval              ClaimApproval `json:"approval"`
	CreatedEventID        string        `json:"created_event_id"`
	CreatedAt             time.Time     `json:"created_at"`
}

// CreateClaimRecordRequest contains caller-owned claim values.
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
	Approval              ClaimApproval
	CreatedEventID        string
}

// ClaimRequirements supplies app-owned record and event lookups to claim creation.
// Evidence and question callbacks are required when their lookup phases are reached, even for empty ID slices;
// the mission-event callback is required on assertion and approval paths. Missing callbacks are programmer misuse.
type ClaimRequirements struct {
	RequireEvidenceRecords func(context.Context, string, []string) error
	RequireQuestionRecords func(context.Context, string, []string) error
	RequireMissionEvent    func(context.Context, string, string) (ledger.Event, error)
}

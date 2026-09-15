package researchproposal

import (
	"context"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/researchcatalog"
)

const (
	ProposalBundleSchemaVersion = "plasma.proposal_bundle.v1"
	ProposalBundleObjectKind    = "proposal_bundle"
)

// ProposalBundle은 evidence/claim/question 후보를 사용자 승인 대상으로 묶은 record다.
type ProposalBundle struct {
	SchemaVersion     string                      `json:"schema_version"`
	ObjectKind        string                      `json:"object_kind"`
	ProposalID        string                      `json:"proposal_id"`
	MissionID         string                      `json:"mission_id"`
	State             string                      `json:"state"`
	Title             string                      `json:"title"`
	ObjectRefs        []researchcatalog.ObjectRef `json:"object_refs"`
	RequestedDecision string                      `json:"requested_decision"`
	CreatedEventID    string                      `json:"created_event_id"`
	DecisionEventID   string                      `json:"decision_event_id,omitempty"`
	CreatedAt         time.Time                   `json:"created_at"`
	DecidedAt         time.Time                   `json:"decided_at,omitempty"`
	UpdatedAt         time.Time                   `json:"updated_at"`
}

// CreateProposalBundleRequest는 proposal bundle 생성 입력이다.
type CreateProposalBundleRequest struct {
	ProposalID        string
	MissionID         string
	State             string
	Title             string
	ObjectRefs        []researchcatalog.ObjectRef
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

// Store persists and reads proposal bundles.
type Store interface {
	CreateProposalBundle(context.Context, ProposalBundle) error
	GetProposalBundle(context.Context, string) (ProposalBundle, error)
	UpdateProposalBundleState(context.Context, ProposalBundleStateUpdate) error
}

// ListStore lists proposal bundles for one mission.
type ListStore interface {
	ListProposalBundles(context.Context, string) ([]ProposalBundle, error)
}

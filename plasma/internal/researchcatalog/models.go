package researchcatalog

import (
	"github.com/c86j224s/liquid2/plasma/internal/mission"
)

const (
	ObjectSourceSnapshot = "source_snapshot"
	ObjectRawArtifact    = "raw_artifact"
	ObjectEvidenceRecord = "evidence_record"
	ObjectClaimRecord    = "claim_record"
	ObjectQuestionRecord = "question_record"
	ObjectOptionRecord   = "option_record"
	ObjectProposalBundle = "proposal_bundle"
	ObjectReport         = "report"
	ObjectReportVersion  = "report_version"
	ObjectReportBlock    = "report_block"
	ObjectLedgerEvent    = "ledger_event"
)

type ObjectRef struct {
	ObjectKind string `json:"object_kind"`
	ObjectID   string `json:"object_id"`
}
type ObjectSummary struct {
	ObjectKind string         `json:"object_kind"`
	ObjectID   string         `json:"object_id"`
	MissionID  string         `json:"mission_id"`
	Summary    string         `json:"summary"`
	Refs       []ObjectRef    `json:"refs,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}
type Page struct {
	MissionID  string          `json:"mission_id"`
	ObjectKind string          `json:"object_kind"`
	Items      []ObjectSummary `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
	Limit      int             `json:"limit"`
	Truncated  bool            `json:"truncated"`
}
type References struct {
	MissionID  string      `json:"mission_id"`
	ObjectKind string      `json:"object_kind"`
	ObjectID   string      `json:"object_id"`
	Forward    []ObjectRef `json:"forward"`
	Backward   []ObjectRef `json:"backward"`
	NextCursor string      `json:"next_cursor,omitempty"`
	Limit      int         `json:"limit"`
	Truncated  bool        `json:"truncated"`
}
type Outline struct {
	MissionID               string
	LastSequence            int64
	Title                   string
	Objective               string
	Scope                   mission.Scope
	Counts                  map[string]int
	ActiveReportVersionID   string
	RecentLedgerEvents      []ObjectSummary
	NextSuggestedObjectRefs []ObjectRef
}

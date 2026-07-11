package app

const (
	ResearchIDEObjectSourceSnapshot = "source_snapshot"
	ResearchIDEObjectRawArtifact    = "raw_artifact"
	ResearchIDEObjectEvidenceRecord = "evidence_record"
	ResearchIDEObjectClaimRecord    = "claim_record"
	ResearchIDEObjectQuestionRecord = "question_record"
	ResearchIDEObjectOptionRecord   = "option_record"
	ResearchIDEObjectProposalBundle = "proposal_bundle"
	ResearchIDEObjectReport         = "report"
	ResearchIDEObjectReportVersion  = "report_version"
	ResearchIDEObjectReportBlock    = "report_block"
	ResearchIDEObjectLedgerEvent    = "ledger_event"
)

type ResearchIDEObjectRef struct {
	ObjectKind string `json:"object_kind"`
	ObjectID   string `json:"object_id"`
}

type ResearchIDEObjectSummary struct {
	ObjectKind string                 `json:"object_kind"`
	ObjectID   string                 `json:"object_id"`
	MissionID  string                 `json:"mission_id"`
	Summary    string                 `json:"summary"`
	Refs       []ResearchIDEObjectRef `json:"refs,omitempty"`
	Metadata   map[string]any         `json:"metadata,omitempty"`
}

type ResearchIDEPage struct {
	MissionID  string                     `json:"mission_id"`
	ObjectKind string                     `json:"object_kind"`
	Items      []ResearchIDEObjectSummary `json:"items"`
	NextCursor string                     `json:"next_cursor,omitempty"`
	Limit      int                        `json:"limit"`
	Truncated  bool                       `json:"truncated"`
}

type ResearchIDEOutline struct {
	MissionID               string                     `json:"mission_id"`
	Title                   string                     `json:"title"`
	Objective               string                     `json:"objective,omitempty"`
	Scope                   MissionScope               `json:"scope"`
	Counts                  map[string]int             `json:"counts"`
	ActiveReportVersionID   string                     `json:"active_report_version_id,omitempty"`
	RecentLedgerEvents      []ResearchIDEObjectSummary `json:"recent_ledger_events,omitempty"`
	NextSuggestedObjectRefs []ResearchIDEObjectRef     `json:"next_suggested_object_refs,omitempty"`
}

type ResearchIDEReadRequest struct {
	MissionID  string
	ObjectKind string
	ObjectID   string
	Offset     int
	MaxBytes   int
	Cursor     string
	Limit      int
	Legacy     bool
}

type ResearchIDEObjectRead struct {
	ObjectKind string                 `json:"object_kind"`
	ObjectID   string                 `json:"object_id"`
	MissionID  string                 `json:"mission_id"`
	Summary    string                 `json:"summary"`
	Refs       []ResearchIDEObjectRef `json:"refs,omitempty"`
	Data       string                 `json:"data"`
	Truncated  bool                   `json:"truncated"`
	NextOffset int                    `json:"next_offset,omitempty"`
	Children   *ResearchIDEPage       `json:"children,omitempty"`
}

type ResearchIDEGrepMatch struct {
	ObjectKind string                 `json:"object_kind"`
	ObjectID   string                 `json:"object_id"`
	MissionID  string                 `json:"mission_id"`
	Snippet    string                 `json:"snippet"`
	Position   int                    `json:"position"`
	Refs       []ResearchIDEObjectRef `json:"refs,omitempty"`
}

type ResearchIDEGrepResult struct {
	MissionID  string                 `json:"mission_id"`
	Query      string                 `json:"query"`
	Matches    []ResearchIDEGrepMatch `json:"matches"`
	NextCursor string                 `json:"next_cursor,omitempty"`
	Limit      int                    `json:"limit"`
	Truncated  bool                   `json:"truncated"`
}

type ResearchIDEReferences struct {
	MissionID  string                 `json:"mission_id"`
	ObjectKind string                 `json:"object_kind"`
	ObjectID   string                 `json:"object_id"`
	Forward    []ResearchIDEObjectRef `json:"forward"`
	Backward   []ResearchIDEObjectRef `json:"backward"`
	NextCursor string                 `json:"next_cursor,omitempty"`
	Limit      int                    `json:"limit"`
	Truncated  bool                   `json:"truncated"`
}

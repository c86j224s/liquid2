package app

const (
	// ResearchIDEObject* 값은 research read/list MCP tool이 탐색할 수 있는 object kind다.
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

// ResearchIDEObjectRef는 research object 사이의 얕은 참조다.
type ResearchIDEObjectRef struct {
	ObjectKind string `json:"object_kind"`
	ObjectID   string `json:"object_id"`
}

// ResearchIDEObjectSummary는 목록/outline에서 큰 본문 없이 object를 보여 주는 값이다.
type ResearchIDEObjectSummary struct {
	ObjectKind string                 `json:"object_kind"`
	ObjectID   string                 `json:"object_id"`
	MissionID  string                 `json:"mission_id"`
	Summary    string                 `json:"summary"`
	Refs       []ResearchIDEObjectRef `json:"refs,omitempty"`
	Metadata   map[string]any         `json:"metadata,omitempty"`
}

// ResearchIDEPage는 research object 목록의 cursor 기반 page다.
type ResearchIDEPage struct {
	MissionID  string                     `json:"mission_id"`
	ObjectKind string                     `json:"object_kind"`
	Items      []ResearchIDEObjectSummary `json:"items"`
	NextCursor string                     `json:"next_cursor,omitempty"`
	Limit      int                        `json:"limit"`
	Truncated  bool                       `json:"truncated"`
}

// ResearchIDEOutline은 agent가 미션 전체 구조를 가볍게 파악하기 위한 summary view다.
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

// ResearchIDEReadRequest는 research object 본문 또는 children을 bounded하게 읽기 위한
// 입력이다.
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

// ResearchIDEObjectRead는 object read 결과와 다음 offset/children page를 담는다.
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

// ResearchIDEGrepMatch는 mission object grep 결과의 한 snippet이다.
type ResearchIDEGrepMatch struct {
	ObjectKind string                 `json:"object_kind"`
	ObjectID   string                 `json:"object_id"`
	MissionID  string                 `json:"mission_id"`
	Snippet    string                 `json:"snippet"`
	Position   int                    `json:"position"`
	Refs       []ResearchIDEObjectRef `json:"refs,omitempty"`
}

// ResearchIDEGrepResult는 mission object grep 결과 page다.
type ResearchIDEGrepResult struct {
	MissionID  string                 `json:"mission_id"`
	Query      string                 `json:"query"`
	Matches    []ResearchIDEGrepMatch `json:"matches"`
	NextCursor string                 `json:"next_cursor,omitempty"`
	Limit      int                    `json:"limit"`
	Truncated  bool                   `json:"truncated"`
}

// ResearchIDEReferences는 object 간 forward/backward reference page다.
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

package app

import (
	"encoding/json"
	"time"
)

const (
	// Report*SchemaVersion/ObjectKind 값은 report object와 block JSON의 stable identity다.
	ReportSchemaVersion        = "plasma.report.v1"
	ReportObjectKind           = "report"
	ReportVersionSchemaVersion = "plasma.report_version.v1"
	ReportVersionObjectKind    = "report_version"
	ReportBlockSchemaVersion   = "plasma.report_block.v1"
	ReportBlockObjectKind      = "report_block"

	// ReportExportTarget* 값은 report version을 외부 artifact로 내보낼 때의 target format이다.
	ReportExportTargetMarkdown = "markdown"
	ReportExportTargetJSONAST  = "json_ast"
	ReportExportTargetHTML     = "html"
)

// Report는 사용자에게 보이는 report의 지속 aggregate root다.
//
// 실제 내용은 active version과 block에 있으며, Report는 현재 active version과 상태를
// 가리킨다.
type Report struct {
	SchemaVersion   string    `json:"schema_version"`
	ObjectKind      string    `json:"object_kind"`
	ReportID        string    `json:"report_id"`
	MissionID       string    `json:"mission_id"`
	Title           string    `json:"title"`
	ActiveVersionID string    `json:"active_version_id"`
	State           string    `json:"state"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ReportEvidenceScope는 report version이 어떤 evidence/claim/question 범위를 참고했는지
// 기록한다.
//
// user-facing report가 source inventory처럼 읽히지 않더라도 내부 traceability를 위해
// 포함 범위는 보존한다.
type ReportEvidenceScope struct {
	AcceptedOnly    bool     `json:"accepted_only"`
	IncludeProposed bool     `json:"include_proposed,omitempty"`
	EvidenceIDs     []string `json:"evidence_ids,omitempty"`
	ClaimIDs        []string `json:"claim_ids,omitempty"`
	QuestionIDs     []string `json:"question_ids,omitempty"`
	OptionIDs       []string `json:"option_ids,omitempty"`
}

// ReportVersion은 특정 시점 report 내용의 immutable version metadata다.
type ReportVersion struct {
	SchemaVersion         string              `json:"schema_version"`
	ObjectKind            string              `json:"object_kind"`
	ReportVersionID       string              `json:"report_version_id"`
	ReportID              string              `json:"report_id"`
	MissionID             string              `json:"mission_id"`
	BaseVersionID         string              `json:"base_version_id,omitempty"`
	State                 string              `json:"state"`
	RootBlockID           string              `json:"root_block_id"`
	BlockIDs              []string            `json:"block_ids"`
	IncludedEvidenceScope ReportEvidenceScope `json:"included_evidence_scope"`
	CreatedEventID        string              `json:"created_event_id"`
	CreatedAt             time.Time           `json:"created_at"`
}

// ReportBlockSourceRefs는 block 내용이 참조한 내부 객체 ID들을 담는다.
type ReportBlockSourceRefs struct {
	ClaimIDs    []string `json:"claim_ids,omitempty"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
	SnapshotIDs []string `json:"snapshot_ids,omitempty"`
	QuestionIDs []string `json:"question_ids,omitempty"`
	OptionIDs   []string `json:"option_ids,omitempty"`
}

// ReportBlockAuthorship은 block을 만든 주체와 작성 방식을 기록한다.
type ReportBlockAuthorship struct {
	Mode     string   `json:"mode"`
	Producer Producer `json:"producer"`
}

// ReportBlock은 report version 안의 구조화된 content 단위다.
type ReportBlock struct {
	SchemaVersion   string                `json:"schema_version"`
	ObjectKind      string                `json:"object_kind"`
	BlockID         string                `json:"block_id"`
	ReportVersionID string                `json:"report_version_id"`
	MissionID       string                `json:"mission_id"`
	BlockType       string                `json:"block_type"`
	ParentBlockID   string                `json:"parent_block_id,omitempty"`
	Order           int                   `json:"order"`
	Content         json.RawMessage       `json:"content"`
	SourceRefs      ReportBlockSourceRefs `json:"source_refs"`
	Authorship      ReportBlockAuthorship `json:"authorship"`
	Approval        Approval              `json:"approval"`
}

// CreateReportDraftRequest는 draft report version과 block을 한 번에 생성하는 요청이다.
type CreateReportDraftRequest struct {
	ReportID        string
	ReportVersionID string
	MissionID       string
	BaseVersionID   string
	Title           string
	FormatIntent    string
	Scope           ReportEvidenceScope
	Producer        Producer
	CreatedEventID  string
	Generation      map[string]any
	Blocks          []ReportBlockDraftInput
}

// ReportBlockDraftInput은 새 report block을 만들기 위한 content와 source ref 입력이다.
type ReportBlockDraftInput struct {
	BlockType  string
	Content    json.RawMessage
	SourceRefs ReportBlockSourceRefs
}

// ReportDraftResult는 draft report 생성 후 함께 반환되는 aggregate와 생성 event다.
type ReportDraftResult struct {
	Report  Report
	Version ReportVersion
	Blocks  []ReportBlock
	Event   LedgerEvent
}

// PromoteReportVersionRequest는 draft report version을 active version으로 승격하는 요청이다.
type PromoteReportVersionRequest struct {
	ReportVersionID string
	ApprovalEventID string
}

// ReportPromotionAppendRequest는 report promotion event builder 입력이다.
type ReportPromotionAppendRequest struct {
	EventID  string
	Version  ReportVersion
	Producer Producer
}

// ReportVersionPromotion은 report version 승격으로 바뀐 상태를 나타낸다.
type ReportVersionPromotion struct {
	ReportID        string
	ReportVersionID string
	FromState       string
	ToState         string
	ReportState     string
	ApprovalEventID string
	UpdatedAt       time.Time
}

// ExportReportVersionRequest는 report version export artifact와 event를 만들기 위한
// 요청이다.
type ExportReportVersionRequest struct {
	ExportID        string
	ReportVersionID string
	Target          string
	ArtifactID      string
	EventID         string
	ApprovalEventID string
	Producer        Producer
}

// ReportExportResult는 report export artifact와 그 기록 event를 함께 반환한다.
type ReportExportResult struct {
	Artifact RawArtifact
	Event    LedgerEvent
}

// ReportASTExport는 report version을 JSON AST artifact로 내보낼 때의 payload다.
type ReportASTExport struct {
	SchemaVersion string        `json:"schema_version"`
	ObjectKind    string        `json:"object_kind"`
	Version       ReportVersion `json:"report_version"`
	Blocks        []ReportBlock `json:"blocks"`
}

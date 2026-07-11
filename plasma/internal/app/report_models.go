package app

import (
	"encoding/json"
	"time"
)

const (
	ReportSchemaVersion        = "plasma.report.v1"
	ReportObjectKind           = "report"
	ReportVersionSchemaVersion = "plasma.report_version.v1"
	ReportVersionObjectKind    = "report_version"
	ReportBlockSchemaVersion   = "plasma.report_block.v1"
	ReportBlockObjectKind      = "report_block"

	ReportExportTargetMarkdown = "markdown"
	ReportExportTargetJSONAST  = "json_ast"
	ReportExportTargetHTML     = "html"
)

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

type ReportEvidenceScope struct {
	AcceptedOnly    bool     `json:"accepted_only"`
	IncludeProposed bool     `json:"include_proposed,omitempty"`
	EvidenceIDs     []string `json:"evidence_ids,omitempty"`
	ClaimIDs        []string `json:"claim_ids,omitempty"`
	QuestionIDs     []string `json:"question_ids,omitempty"`
	OptionIDs       []string `json:"option_ids,omitempty"`
}

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

type ReportBlockSourceRefs struct {
	ClaimIDs    []string `json:"claim_ids,omitempty"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
	SnapshotIDs []string `json:"snapshot_ids,omitempty"`
	QuestionIDs []string `json:"question_ids,omitempty"`
	OptionIDs   []string `json:"option_ids,omitempty"`
}

type ReportBlockAuthorship struct {
	Mode     string   `json:"mode"`
	Producer Producer `json:"producer"`
}

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

type ReportBlockDraftInput struct {
	BlockType  string
	Content    json.RawMessage
	SourceRefs ReportBlockSourceRefs
}

type ReportDraftResult struct {
	Report  Report
	Version ReportVersion
	Blocks  []ReportBlock
	Event   LedgerEvent
}

type PromoteReportVersionRequest struct {
	ReportVersionID string
	ApprovalEventID string
}

type ReportPromotionAppendRequest struct {
	EventID  string
	Version  ReportVersion
	Producer Producer
}

type ReportVersionPromotion struct {
	ReportID        string
	ReportVersionID string
	FromState       string
	ToState         string
	ReportState     string
	ApprovalEventID string
	UpdatedAt       time.Time
}

type ExportReportVersionRequest struct {
	ExportID        string
	ReportVersionID string
	Target          string
	ArtifactID      string
	EventID         string
	ApprovalEventID string
	Producer        Producer
}

type ReportExportResult struct {
	Artifact RawArtifact
	Event    LedgerEvent
}

type ReportASTExport struct {
	SchemaVersion string        `json:"schema_version"`
	ObjectKind    string        `json:"object_kind"`
	Version       ReportVersion `json:"report_version"`
	Blocks        []ReportBlock `json:"blocks"`
}

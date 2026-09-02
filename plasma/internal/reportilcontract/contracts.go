package reportilcontract

import (
	"context"
	"encoding/json"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

const PipelineFamily = "report_il_experimental"

type SourceSnapshot struct {
	SnapshotID      string
	MissionID       string
	Title           string
	ArtifactIDs     []string
	ContentHash     string
	RetrievalPolicy string
	ConnectorType   string
	ExternalURI     string
	Locators        json.RawMessage
	Active          bool
}

type Artifact struct {
	ArtifactID string
	MissionID  string
	MediaType  string
	Filename   string
	ByteSize   int64
	SHA256     string
	Content    []byte
}

type LocalRead struct {
	Content            string
	Binary             bool
	Truncated          bool
	Size               int64
	MTime              string
	SHA256             string
	ObservationReceipt string
	Extraction         string
}

type SourceReader interface {
	ListSourceSnapshots(context.Context, string) ([]SourceSnapshot, error)
	GetArtifact(context.Context, string) (Artifact, error)
	ReadLive(context.Context, string, string, int64) (LocalRead, error)
}

type ArtifactInput struct {
	ArtifactID     string
	MissionID      string
	MediaType      string
	Filename       string
	Producer       ledger.Producer
	Content        []byte
	ExpectedSHA256 string
}

type EventInput struct {
	EventID          string
	MissionID        string
	EventType        string
	CausationEventID string
	CorrelationID    string
	Producer         ledger.Producer
	Payload          []byte
}

type BundleRequest struct {
	MissionID      string
	PendingID      string
	Artifacts      []ArtifactInput
	StoreCompleted EventInput
	Terminal       EventInput
}

type BundleResult struct {
	Artifacts []Artifact
	Terminal  ledger.Event
	Created   bool
}

// TerminalArtifactEntry is the closed durable artifact lineage entry.
type TerminalArtifactEntry struct {
	ArtifactID string `json:"artifact_id"`
	Kind       string `json:"kind"`
	MediaType  string `json:"media_type"`
	SHA256     string `json:"sha256"`
	ByteSize   int    `json:"byte_size"`
	Role       string `json:"role"`
	Filename   string `json:"filename"`
	AssetID    string `json:"asset_id,omitempty"`
}

type TerminalSourceSelectionSummary struct {
	Applied                 bool `json:"applied"`
	AcceptedSources         int  `json:"accepted_sources"`
	UsableSources           int  `json:"usable_sources"`
	SelectedSources         int  `json:"selected_sources"`
	SupplementalSources     int  `json:"supplemental_sources"`
	ExcludedUnusableSources int  `json:"excluded_unusable_sources"`
	ExcludedBudgetSources   int  `json:"excluded_budget_sources"`
}

type TerminalArtifactLineage struct {
	PipelineFamily     string                          `json:"pipeline_family"`
	Artifacts          []TerminalArtifactEntry         `json:"artifacts"`
	MarkdownArtifactID string                          `json:"markdown_artifact_id"`
	SourceSelection    *TerminalSourceSelectionSummary `json:"source_selection,omitempty"`
}

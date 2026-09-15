package reportexperiment

import "github.com/c86j224s/liquid2/plasma/internal/mcp/wire"

type ExperimentReportCreateInput struct {
	wire.CommonMutatingInput
	DraftID string `json:"draft_id"`
	Title   string `json:"title"`
}

type ExperimentReportAppendInput struct {
	wire.CommonMutatingInput
	DraftID string `json:"draft_id"`
	Content string `json:"content"`
}

type ExperimentReportReadInput struct {
	MissionID string `json:"mission_id"`
	SessionID string `json:"session_id"`
	DraftID   string `json:"draft_id"`
	Offset    int    `json:"offset"`
	MaxBytes  int    `json:"max_bytes"`
}

type ExperimentReportFinalizeInput struct {
	wire.CommonMutatingInput
	DraftID        string `json:"draft_id"`
	ArtifactID     string `json:"artifact_id"`
	Filename       string `json:"filename"`
	Title          string `json:"title"`
	ExpectedSHA256 string `json:"expected_sha256"`
}

type ExperimentReportDraftOutput struct {
	DraftID       string `json:"draft_id"`
	MissionID     string `json:"mission_id"`
	SessionID     string `json:"session_id"`
	Title         string `json:"title,omitempty"`
	State         string `json:"state"`
	ContentLength int    `json:"content_length"`
	ChunkCount    int    `json:"chunk_count"`
	Finalized     bool   `json:"finalized"`
	ArtifactID    string `json:"artifact_id,omitempty"`
}

type ExperimentReportReadOutput struct {
	DraftID       string `json:"draft_id"`
	MissionID     string `json:"mission_id"`
	SessionID     string `json:"session_id"`
	Content       string `json:"content"`
	Offset        int    `json:"offset"`
	NextOffset    int    `json:"next_offset,omitempty"`
	ContentLength int    `json:"content_length"`
	Truncated     bool   `json:"truncated"`
	Finalized     bool   `json:"finalized"`
	ArtifactID    string `json:"artifact_id,omitempty"`
}

type ExperimentReportFinalizeOutput struct {
	DraftID       string                               `json:"draft_id"`
	MissionID     string                               `json:"mission_id"`
	SessionID     string                               `json:"session_id"`
	ContentLength int                                  `json:"content_length"`
	Artifact      wire.RawArtifactOutput               `json:"artifact"`
	EventID       string                               `json:"event_id"`
	HumanizeReady *ExperimentReportHumanizeReadyOutput `json:"humanize_ready,omitempty"`
}

type ExperimentReportHumanizeReadyOutput struct {
	EventID                   string `json:"event_id,omitempty"`
	Profile                   string `json:"profile"`
	Target                    string `json:"target"`
	SourceArtifactID          string `json:"source_artifact_id"`
	SourceArtifactSHA256      string `json:"source_artifact_sha256"`
	PreservedOriginalMarkdown bool   `json:"preserved_original_markdown"`
	Reason                    string `json:"reason"`
}

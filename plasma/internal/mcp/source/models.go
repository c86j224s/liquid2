package source

import (
	"encoding/json"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"github.com/c86j224s/liquid2/plasma/internal/source/liquid2source"
	"strings"
)

type SourcesListInput struct {
	MissionID         string `json:"mission_id"`
	IncludeRemoved    bool   `json:"include_removed"`
	IncludeSuperseded bool   `json:"include_superseded"`
}

type SourcesListOutput struct {
	Sources []SourceSnapshotOutput `json:"sources"`
}

type SourcesReadInput struct {
	MissionID  string `json:"mission_id"`
	SnapshotID string `json:"snapshot_id"`
	ArtifactID string `json:"artifact_id"`
	Subpath    string `json:"subpath"`
	Offset     int    `json:"offset"`
	MaxBytes   int    `json:"max_bytes"`
}

type SourcesReadOutput struct {
	Snapshot            SourceSnapshotOutput              `json:"snapshot"`
	Artifact            wire.RawArtifactOutput            `json:"artifact"`
	Content             string                            `json:"content"`
	Offset              int                               `json:"offset"`
	NextOffset          int                               `json:"next_offset,omitempty"`
	ContentLength       int                               `json:"content_length"`
	ContentLengthKnown  bool                              `json:"content_length_known"`
	Truncated           bool                              `json:"truncated"`
	MetadataOnly        bool                              `json:"metadata_only,omitempty"`
	Extraction          *wire.SourceExtractionOutput      `json:"extraction,omitempty"`
	ObservationMetadata *sourcecontract.LocalPathMetadata `json:"observation_metadata,omitempty"`
	ObservationEventID  string                            `json:"observation_event_id,omitempty"`
}

type SourcesTreeInput struct {
	MissionID  string `json:"mission_id"`
	SnapshotID string `json:"snapshot_id"`
	Subpath    string `json:"subpath"`
	Depth      int    `json:"depth"`
	Limit      int    `json:"limit"`
}

type SourcesTreeOutput struct {
	Snapshot            SourceSnapshotOutput               `json:"snapshot"`
	Tree                sourcecontract.LocalPathTreeResult `json:"tree"`
	ObservationMetadata *sourcecontract.LocalPathMetadata  `json:"observation_metadata,omitempty"`
	ObservationEventID  string                             `json:"observation_event_id,omitempty"`
}

type SourcesGrepInput struct {
	MissionID   string `json:"mission_id"`
	SnapshotID  string `json:"snapshot_id"`
	Subpath     string `json:"subpath"`
	Query       string `json:"query"`
	MaxSnippets int    `json:"max_snippets"`
}

type SourcesGrepOutput struct {
	Snapshot            SourceSnapshotOutput               `json:"snapshot"`
	Grep                sourcecontract.LocalPathGrepResult `json:"grep"`
	ObservationMetadata *sourcecontract.LocalPathMetadata  `json:"observation_metadata,omitempty"`
	ObservationEventID  string                             `json:"observation_event_id,omitempty"`
}

type MediaSourceReadOutput struct {
	Snapshot       SourceSnapshotOutput        `json:"snapshot"`
	Artifact       wire.RawArtifactOutput      `json:"artifact,omitempty"`
	Media          sourcecontract.MediaLocator `json:"media"`
	InspectionNote string                      `json:"inspection_note"`
}

type SourcesSearchInput struct {
	MissionID    string   `json:"mission_id"`
	Query        string   `json:"query"`
	Connectors   []string `json:"connectors"`
	ConnectionID string   `json:"connection_id"`
	CloudID      string   `json:"cloud_id"`
	SpaceKey     string   `json:"space_key"`
	Limit        int      `json:"limit"`
	Cursor       string   `json:"cursor"`
}

type SourcesSearchOutput struct {
	Candidates  []SourceCandidateOutput `json:"candidates"`
	NextCursors map[string]string       `json:"next_cursors,omitempty"`
}

type SourcesSnapshotInput struct {
	wire.CommonMutatingInput
	Connector  ConnectorRefInput   `json:"connector"`
	ArtifactID string              `json:"artifact_id"`
	SnapshotID string              `json:"snapshot_id"`
	EventID    string              `json:"event_id"`
	Ranges     []ContentRangeInput `json:"ranges"`
	Reason     string              `json:"reason"`
}

type SourcesSnapshotOutput struct {
	SnapshotID  string   `json:"snapshot_id"`
	ArtifactIDs []string `json:"artifact_ids"`
}

type LocalPathRootsInput struct {
	MissionID string `json:"mission_id"`
}

type LocalPathRootsOutput struct {
	Roots []sourcecontract.LocalPathRoot `json:"roots"`
}

type LocalPathTreeInput struct {
	MissionID    string `json:"mission_id"`
	RootID       string `json:"root_id"`
	RelativePath string `json:"relative_path"`
	Depth        int    `json:"depth"`
	Limit        int    `json:"limit"`
}

type LocalPathTreeOutput struct {
	Tree sourcecontract.LocalPathTreeResult `json:"tree"`
}

type LocalPathAttachInput struct {
	wire.CommonMutatingInput
	SnapshotID   string `json:"snapshot_id"`
	RootID       string `json:"root_id"`
	RelativePath string `json:"relative_path"`
	Title        string `json:"title"`
	Restore      bool   `json:"restore"`
}

type LocalPathAttachOutput struct {
	Snapshot        SourceSnapshotOutput `json:"snapshot"`
	EventID         string               `json:"event_id,omitempty"`
	Existing        bool                 `json:"existing"`
	Restored        bool                 `json:"restored"`
	RestoreRequired bool                 `json:"restore_required,omitempty"`
}

type SourceRemoveInput struct {
	wire.CommonMutatingInput
	SnapshotID string `json:"snapshot_id"`
	Reason     string `json:"reason"`
}

type SourceRestoreInput struct {
	wire.CommonMutatingInput
	SnapshotID string `json:"snapshot_id"`
}

type SourceStateChangeOutput struct {
	Snapshot   SourceSnapshotOutput `json:"snapshot"`
	EventID    string               `json:"event_id,omitempty"`
	Idempotent bool                 `json:"idempotent"`
}

type ConnectorRefInput struct {
	ConnectorID      string `json:"connector_id"`
	ConnectorType    string `json:"connector_type"`
	ExternalSourceID string `json:"external_source_id"`
	ExternalURI      string `json:"external_uri"`
	ExternalVersion  string `json:"external_version"`
	ConnectorVersion string `json:"connector_version"`
}

type ContentRangeInput struct {
	ContentID string `json:"content_id"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
}

type SourceCandidateOutput struct {
	Connector     ConnectorRefInput    `json:"connector"`
	Title         string               `json:"title"`
	SourceURI     string               `json:"source_uri"`
	Summary       string               `json:"summary"`
	MatchedRanges []MatchedRangeOutput `json:"matched_ranges,omitempty"`
	UpdatedAt     string               `json:"updated_at,omitempty"`
	CanSnapshot   bool                 `json:"can_snapshot"`
}

type MatchedRangeOutput struct {
	ContentID string `json:"content_id,omitempty"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
}

type SourceSnapshotOutput struct {
	SnapshotID        string                     `json:"snapshot_id"`
	MissionID         string                     `json:"mission_id"`
	Connector         ConnectorRefInput          `json:"connector"`
	Title             string                     `json:"title"`
	CapturedAt        string                     `json:"captured_at,omitempty"`
	ExternalUpdatedAt string                     `json:"external_updated_at,omitempty"`
	ArtifactIDs       []string                   `json:"artifact_ids"`
	ContentHash       sourcecontract.ContentHash `json:"content_hash"`
	Locators          json.RawMessage            `json:"locators"`
	Access            sourcecontract.Access      `json:"access"`
	RetrievalPolicy   string                     `json:"retrieval_policy"`
	State             sourcecontract.State       `json:"state"`
}

func (input ConnectorRefInput) toApp() sourcecontract.ConnectorRef {
	return sourcecontract.ConnectorRef{
		ConnectorID:      input.ConnectorID,
		ConnectorType:    input.ConnectorType,
		ExternalSourceID: input.ExternalSourceID,
		ExternalURI:      input.ExternalURI,
		ExternalVersion:  input.ExternalVersion,
		ConnectorVersion: input.ConnectorVersion,
	}
}

func (input ContentRangeInput) toApp() liquid2source.Liquid2ContentRange {
	return liquid2source.Liquid2ContentRange{
		ContentID: input.ContentID,
		Start:     input.Start,
		End:       input.End,
	}
}

func SourceCandidateFromApp(candidate liquid2source.Liquid2SourceCandidate) SourceCandidateOutput {
	ranges := make([]MatchedRangeOutput, 0, len(candidate.MatchedRanges))
	for _, matchedRange := range candidate.MatchedRanges {
		ranges = append(ranges, MatchedRangeOutput{
			ContentID: matchedRange.ContentID,
			Start:     matchedRange.Start,
			End:       matchedRange.End,
		})
	}
	output := SourceCandidateOutput{
		Connector:     ConnectorRefFromApp(candidate.Connector),
		Title:         candidate.Title,
		SourceURI:     candidate.SourceURI,
		Summary:       candidate.Summary,
		CanSnapshot:   candidate.CanSnapshot,
		MatchedRanges: ranges,
	}
	if !candidate.UpdatedAt.IsZero() {
		output.UpdatedAt = candidate.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return output
}

func SourceCandidateFromConfluence(candidate confluencesource.ConfluenceSourceCandidate) SourceCandidateOutput {
	output := SourceCandidateOutput{
		Connector:   ConnectorRefFromApp(candidate.Connector),
		Title:       candidate.Title,
		SourceURI:   candidate.SourceURI,
		CanSnapshot: candidate.CanSnapshot,
	}
	if !candidate.UpdatedAt.IsZero() {
		output.UpdatedAt = candidate.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return output
}

func SourceSnapshotFromApp(snapshot sourcecontract.Snapshot) SourceSnapshotOutput {
	output := SourceSnapshotOutput{
		SnapshotID:      snapshot.SnapshotID,
		MissionID:       snapshot.MissionID,
		Connector:       ConnectorRefFromApp(snapshot.Connector),
		Title:           snapshot.Title,
		ArtifactIDs:     append([]string(nil), snapshot.ArtifactIDs...),
		ContentHash:     snapshot.ContentHash,
		Locators:        append(json.RawMessage(nil), snapshot.Locators...),
		Access:          snapshot.Access,
		RetrievalPolicy: sourceRetrievalPolicy(snapshot),
		State:           sourceState(snapshot),
	}
	if strings.TrimSpace(output.Access.RetrievalPolicy) == "" {
		output.Access.RetrievalPolicy = output.RetrievalPolicy
	}
	if !snapshot.CapturedAt.IsZero() {
		output.CapturedAt = snapshot.CapturedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	if !snapshot.ExternalUpdatedAt.IsZero() {
		output.ExternalUpdatedAt = snapshot.ExternalUpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return output
}

func SourceSnapshotsFromApp(snapshots []sourcecontract.Snapshot) []SourceSnapshotOutput {
	output := make([]SourceSnapshotOutput, 0, len(snapshots))
	for _, snapshot := range snapshots {
		output = append(output, SourceSnapshotFromApp(snapshot))
	}
	return output
}

func ConnectorRefFromApp(connector sourcecontract.ConnectorRef) ConnectorRefInput {
	return ConnectorRefInput{
		ConnectorID:      connector.ConnectorID,
		ConnectorType:    connector.ConnectorType,
		ExternalSourceID: connector.ExternalSourceID,
		ExternalURI:      connector.ExternalURI,
		ExternalVersion:  connector.ExternalVersion,
		ConnectorVersion: connector.ConnectorVersion,
	}
}

func ContentRangesToApp(inputs []ContentRangeInput) []liquid2source.Liquid2ContentRange {
	ranges := make([]liquid2source.Liquid2ContentRange, 0, len(inputs))
	for _, input := range inputs {
		ranges = append(ranges, input.toApp())
	}
	return ranges
}

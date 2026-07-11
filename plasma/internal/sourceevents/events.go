package sourceevents

import (
	"encoding/json"
	"strings"
)

const SourceSnapshottedEventType = "source.snapshotted"

type ConnectorRef struct {
	ConnectorID      string
	ConnectorType    string
	ExternalSourceID string
	ExternalURI      string
	ExternalVersion  string
	ConnectorVersion string
}

type SourceSnapshottedPayloadRequest struct {
	SnapshotID         string
	ArtifactIDs        []string
	Connector          ConnectorRef
	IncludeArtifactIDs bool
}

type ConnectorSourceSnapshottedPayloadRequest struct {
	SnapshotID  string
	ArtifactIDs []string
	Connector   ConnectorRef
	Reason      string
}

type ConfluenceUpdateSourceSnapshottedPayloadRequest struct {
	SnapshotID         string
	ArtifactIDs        []string
	Connector          ConnectorRef
	Reason             string
	PreviousSnapshotID string
	PreviousVersion    int
	CloudID            string
	PageID             string
}

type UploadedFileSourceSnapshottedPayloadRequest struct {
	SnapshotID        string
	ArtifactIDs       []string
	Title             string
	OriginalFilename  string
	SanitizedFilename string
	MediaType         string
	ContentKind       string
	SHA256            string
	Deduplicated      bool
}

func BuildSourceSnapshottedPayload(req SourceSnapshottedPayloadRequest) []byte {
	payload := map[string]any{
		"snapshot_id": strings.TrimSpace(req.SnapshotID),
		"connector":   connectorRefPayloadFromSourceEvents(req.Connector),
	}
	if req.IncludeArtifactIDs {
		payload["artifact_ids"] = req.ArtifactIDs
	}
	return mustMarshalJSON(payload)
}

func BuildConnectorSourceSnapshottedPayload(req ConnectorSourceSnapshottedPayloadRequest) []byte {
	return mustMarshalJSON(map[string]any{
		"snapshot_id":  strings.TrimSpace(req.SnapshotID),
		"artifact_ids": req.ArtifactIDs,
		"connector":    connectorRefPayloadFromSourceEvents(req.Connector),
		"reason":       strings.TrimSpace(req.Reason),
	})
}

func BuildConfluenceUpdateSourceSnapshottedPayload(req ConfluenceUpdateSourceSnapshottedPayloadRequest) []byte {
	return mustMarshalJSON(map[string]any{
		"snapshot_id":          strings.TrimSpace(req.SnapshotID),
		"artifact_ids":         req.ArtifactIDs,
		"connector":            connectorRefPayloadFromSourceEvents(req.Connector),
		"reason":               strings.TrimSpace(req.Reason),
		"previous_snapshot_id": strings.TrimSpace(req.PreviousSnapshotID),
		"previous_version":     req.PreviousVersion,
		"confluence_cloud_id":  strings.TrimSpace(req.CloudID),
		"confluence_page_id":   strings.TrimSpace(req.PageID),
	})
}

func BuildUploadedFileSourceSnapshottedPayload(req UploadedFileSourceSnapshottedPayloadRequest) []byte {
	return mustMarshalJSON(map[string]any{
		"snapshot_id":        strings.TrimSpace(req.SnapshotID),
		"artifact_ids":       req.ArtifactIDs,
		"source_kind":        "file_upload",
		"title":              strings.TrimSpace(req.Title),
		"original_filename":  strings.TrimSpace(req.OriginalFilename),
		"sanitized_filename": strings.TrimSpace(req.SanitizedFilename),
		"media_type":         strings.TrimSpace(req.MediaType),
		"content_kind":       strings.TrimSpace(req.ContentKind),
		"sha256":             strings.TrimSpace(req.SHA256),
		"deduplicated":       req.Deduplicated,
	})
}

type connectorRefPayload struct {
	ConnectorID      string `json:"connector_id"`
	ConnectorType    string `json:"connector_type"`
	ExternalSourceID string `json:"external_source_id"`
	ExternalURI      string `json:"external_uri"`
	ExternalVersion  string `json:"external_version"`
	ConnectorVersion string `json:"connector_version"`
}

func connectorRefPayloadFromSourceEvents(connector ConnectorRef) connectorRefPayload {
	return connectorRefPayload{
		ConnectorID:      connector.ConnectorID,
		ConnectorType:    connector.ConnectorType,
		ExternalSourceID: connector.ExternalSourceID,
		ExternalURI:      connector.ExternalURI,
		ExternalVersion:  connector.ExternalVersion,
		ConnectorVersion: connector.ConnectorVersion,
	}
}

func mustMarshalJSON(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

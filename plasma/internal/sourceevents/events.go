package sourceevents

import (
	"encoding/json"
	"strings"
)

// SourceSnapshottedEventType은 source snapshot이 mission source set에 추가됐음을
// 나타내는 장부 event type이다.
const SourceSnapshottedEventType = "source.snapshotted"

// ConnectorRef는 외부 source와 Plasma snapshot을 연결하는 connector identity다.
//
// ExternalSourceID와 ExternalVersion은 connector 내부에서 stable해야 하며, 같은 외부
// 문서를 다른 snapshot version으로 추적하는 데 쓰인다.
type ConnectorRef struct {
	ConnectorID      string
	ConnectorType    string
	ExternalSourceID string
	ExternalURI      string
	ExternalVersion  string
	ConnectorVersion string
}

// SourceSnapshottedPayloadRequest는 가장 일반적인 source snapshot payload 입력이다.
type SourceSnapshottedPayloadRequest struct {
	SnapshotID         string
	ArtifactIDs        []string
	Connector          ConnectorRef
	IncludeArtifactIDs bool
}

// ConnectorSourceSnapshottedPayloadRequest는 connector-backed source snapshot payload
// 입력이다.
type ConnectorSourceSnapshottedPayloadRequest struct {
	SnapshotID  string
	ArtifactIDs []string
	Connector   ConnectorRef
	Reason      string
}

// ConfluenceUpdateSourceSnapshottedPayloadRequest는 기존 Confluence source update가
// 새 snapshot으로 기록될 때의 payload 입력이다.
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

// UploadedFileSourceSnapshottedPayloadRequest는 업로드 파일 source snapshot payload
// 입력이다.
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

// BuildSourceSnapshottedPayload는 일반 source.snapshotted payload JSON을 만든다.
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

// BuildConnectorSourceSnapshottedPayload는 connector-backed snapshot payload JSON을 만든다.
func BuildConnectorSourceSnapshottedPayload(req ConnectorSourceSnapshottedPayloadRequest) []byte {
	return mustMarshalJSON(map[string]any{
		"snapshot_id":  strings.TrimSpace(req.SnapshotID),
		"artifact_ids": req.ArtifactIDs,
		"connector":    connectorRefPayloadFromSourceEvents(req.Connector),
		"reason":       strings.TrimSpace(req.Reason),
	})
}

// BuildConfluenceUpdateSourceSnapshottedPayload는 Confluence update snapshot payload
// JSON을 만든다.
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

// BuildUploadedFileSourceSnapshottedPayload는 file upload source snapshot payload JSON을
// 만든다.
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

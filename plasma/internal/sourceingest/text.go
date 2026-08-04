package sourceingest

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
)

// CreateTextSourceWithEvent는 사용자가 붙여넣은 텍스트를 artifact와 manual
// source snapshot으로 저장한다. 외부 URI가 없으면 snapshot ID 기반 manual 식별자로
// 같은 미션 안의 저장 식별자를 만든다.
func CreateTextSourceWithEvent(ctx context.Context, store Store, req CreateTextSourceWithEventRequest) (SourceSnapshotWithEventResult, error) {
	title := strings.TrimSpace(req.Source.Title)
	if title == "" {
		title = "Pasted text source"
	}
	content := strings.TrimSpace(req.Source.Content)
	externalURI := strings.TrimSpace(req.Source.ExternalURI)
	externalID := externalURI
	if externalID == "" {
		externalID = "manual:" + req.SnapshotID
	}
	return store.CreateSourceSnapshotWithEvent(ctx, CreateSourceSnapshotWithEventRequest{
		Artifact: CreateRawArtifactRequest{
			ArtifactID: req.ArtifactID,
			MissionID:  req.MissionID,
			MediaType:  "text/plain; charset=utf-8",
			Filename:   sourceIngestFilename(title, ".txt"),
			Producer:   req.Producer,
			Content:    []byte(content),
		},
		Snapshot: CreateSourceSnapshotRequest{
			SnapshotID: req.SnapshotID,
			MissionID:  req.MissionID,
			Connector: ConnectorRef{
				ConnectorID:      "manual",
				ConnectorType:    "text",
				ExternalSourceID: externalID,
				ExternalURI:      externalURI,
				ConnectorVersion: "plasma-ui.v1",
			},
			Title:       title,
			ArtifactIDs: []string{req.ArtifactID},
			Locators:    json.RawMessage(`[{"locator_type":"full_text"}]`),
		},
		Event: AppendEventRequest{
			EventID:   req.EventID,
			MissionID: req.MissionID,
			EventType: sourceevents.SourceSnapshottedEventType,
			Producer:  req.Producer,
			Payload: mustMarshalJSON(map[string]any{
				"snapshot_id":  req.SnapshotID,
				"artifact_ids": []string{req.ArtifactID},
				"source_kind":  "text",
				"title":        title,
			}),
		},
	})
}

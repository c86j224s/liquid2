package sourceingest

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
)

func CreateFetchedURLSourceWithEvent(ctx context.Context, store Store, req CreateFetchedURLSourceRequest) (URLSourceSnapshotResult, error) {
	title := firstNonEmptyString(req.Title, req.Fetched.Title, req.URL)
	fetchedAt := req.FetchedAt.UTC()
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	locators, err := json.Marshal([]map[string]string{{
		"locator_type": SourceLocatorTypeFullDocument,
		"url":          req.URL,
		"fetched_at":   fetchedAt.Format(time.RFC3339Nano),
	}})
	if err != nil {
		return URLSourceSnapshotResult{}, err
	}
	result, err := store.CreateSourceSnapshotWithEvent(ctx, CreateSourceSnapshotWithEventRequest{
		Artifact: CreateRawArtifactRequest{
			ArtifactID: req.ArtifactID,
			MissionID:  req.MissionID,
			MediaType:  req.Fetched.MediaType,
			Filename:   sourceDocumentFilename(title, req.Fetched.MediaType),
			Producer:   req.Producer,
			Content:    req.Fetched.Content,
		},
		Snapshot: CreateSourceSnapshotRequest{
			SnapshotID: req.SnapshotID,
			MissionID:  req.MissionID,
			Connector: ConnectorRef{
				ConnectorID:      "url",
				ConnectorType:    "url",
				ExternalSourceID: req.URL,
				ExternalURI:      req.URL,
				ExternalVersion:  req.Fetched.ExternalVersion,
				ConnectorVersion: "plasma-ui.url.v1",
			},
			Title:             title,
			ExternalUpdatedAt: req.Fetched.ExternalUpdatedAt,
			ArtifactIDs:       []string{req.ArtifactID},
			Locators:          json.RawMessage(locators),
		},
		Event: AppendEventRequest{
			EventID:   req.EventID,
			MissionID: req.MissionID,
			EventType: sourceevents.SourceSnapshottedEventType,
			Producer:  req.Producer,
			Payload: mustMarshalJSON(map[string]any{
				"snapshot_id":  req.SnapshotID,
				"artifact_ids": []string{req.ArtifactID},
				"source_kind":  "url",
				"title":        title,
				"url":          req.URL,
			}),
		},
	})
	if err != nil {
		return URLSourceSnapshotResult{}, err
	}
	return URLSourceSnapshotResult{Artifact: result.Artifact, Snapshot: result.Snapshot, Event: result.Event}, nil
}

func BuildSourceSnapshotFailureAppendRequest(req SourceSnapshotFailureAppendRequest) AppendEventRequest {
	return AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(req.MissionID),
		EventType: "source.snapshot_failed",
		Producer:  req.Producer,
		Payload: mustMarshalJSON(map[string]any{
			"kind":        "source_snapshot_failed",
			"source_kind": strings.TrimSpace(req.SourceKind),
			"url":         req.URL,
			"message":     req.Message,
		}),
	}
}

func CreateStagedURLSourceWithEvent(ctx context.Context, store Store, req CreateStagedURLSourceRequest) (URLSourceSnapshotResult, error) {
	title := firstNonEmptyString(req.Title, req.Staged.Title, req.URL)
	locators, err := json.Marshal([]map[string]string{{
		"locator_type": SourceLocatorTypeFullDocument,
		"url":          req.URL,
		"fetched_at":   req.Staged.Artifact.CreatedAt.Format(time.RFC3339Nano),
		"staged_from":  req.Staged.ProposalEventID,
	}})
	if err != nil {
		return URLSourceSnapshotResult{}, err
	}
	result, err := store.CreateExistingArtifactSourceSnapshotWithEvent(ctx, CreateExistingArtifactSourceSnapshotWithEventRequest{
		Snapshot: CreateSourceSnapshotRequest{
			SnapshotID: req.SnapshotID,
			MissionID:  req.MissionID,
			Connector: ConnectorRef{
				ConnectorID:      "url",
				ConnectorType:    "url",
				ExternalSourceID: req.URL,
				ExternalURI:      req.URL,
				ExternalVersion:  req.Staged.ExternalVersion,
				ConnectorVersion: "plasma-ui.url.v1",
			},
			Title:             title,
			ExternalUpdatedAt: req.Staged.ExternalUpdatedAt,
			ArtifactIDs:       []string{req.Staged.Artifact.ArtifactID},
			Locators:          json.RawMessage(locators),
		},
		Event: AppendEventRequest{
			EventID:   req.EventID,
			MissionID: req.MissionID,
			EventType: sourceevents.SourceSnapshottedEventType,
			Producer:  req.Producer,
			Payload: mustMarshalJSON(map[string]any{
				"snapshot_id":                        req.SnapshotID,
				"artifact_ids":                       []string{req.Staged.Artifact.ArtifactID},
				"source_kind":                        "url",
				"title":                              title,
				"url":                                req.URL,
				"source_candidate_proposal_event_id": req.Staged.ProposalEventID,
				"source_candidate_artifact_reused":   true,
			}),
		},
	})
	if err != nil {
		return URLSourceSnapshotResult{}, err
	}
	return URLSourceSnapshotResult{Artifact: req.Staged.Artifact, Snapshot: result.Snapshot, Event: result.Event, ReusedSourceCandidate: true}, nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

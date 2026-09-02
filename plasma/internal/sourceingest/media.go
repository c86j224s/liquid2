package sourceingest

import (
	"context"
	"encoding/json"
	"mime"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
)

// CreateFetchedMediaURLSourceWithEvent는 미디어 URL fetch 결과를 미디어 source
// snapshot으로 저장한다. 이미지는 artifact를 저장하고, 오디오/비디오는 현재
// 직접 검사하지 않는 live reference로 남긴다.
func CreateFetchedMediaURLSourceWithEvent(ctx context.Context, store Store, req CreateFetchedMediaURLSourceRequest) (MediaSourceSnapshotResult, error) {
	title := firstNonEmptyString(req.Title, req.Fetched.Title, req.URL)
	license := strings.TrimSpace(req.License)
	if license == "" {
		license = "unknown"
	}
	attribution := strings.TrimSpace(req.Attribution)
	locator := MediaLocator{
		LocatorType:       SourceLocatorTypeMedia,
		MediaKind:         req.Fetched.MediaKind,
		Provider:          SourceConnectorTypeMediaURL,
		CanonicalURL:      req.URL,
		SourcePageURL:     req.URL,
		DirectMediaURL:    req.URL,
		MIMEType:          req.Fetched.MediaType,
		ByteSize:          req.Fetched.ByteSize,
		Width:             req.Fetched.Width,
		Height:            req.Fetched.Height,
		Title:             title,
		Attribution:       attribution,
		License:           license,
		InspectionSupport: sourceMediaInspectionSupport(req.Fetched.MediaKind),
	}
	if req.Fetched.MediaKind == MediaKindImage {
		return createFetchedImageMediaURLSource(ctx, store, req, title, license, locator)
	}
	return createFetchedLiveMediaURLSource(ctx, store, req, title, license, locator)
}

func CreateStagedImageMediaURLSourceWithEvent(ctx context.Context, store Store, req CreateStagedImageMediaURLSourceRequest) (MediaSourceSnapshotResult, error) {
	artifact := req.Staged.Artifact
	if artifact.MissionID != req.MissionID || artifact.ArtifactID == "" ||
		artifact.SHA256 == "" || artifact.ByteSize != int64(len(artifact.Content)) ||
		!strings.EqualFold(artifact.SHA256, sha256HexBytes(artifact.Content)) ||
		!stagedImageMediaType(artifact.MediaType) {
		return MediaSourceSnapshotResult{}, ErrInvalidInput
	}
	title := firstNonEmptyString(req.Title, req.Staged.Title, req.URL)
	license := strings.TrimSpace(req.License)
	if license == "" {
		license = "unknown"
	}
	locator := MediaLocator{
		LocatorType:       SourceLocatorTypeMedia,
		MediaKind:         MediaKindImage,
		Provider:          SourceConnectorTypeMediaURL,
		CanonicalURL:      req.URL,
		SourcePageURL:     req.URL,
		DirectMediaURL:    req.URL,
		MIMEType:          req.Staged.Artifact.MediaType,
		ByteSize:          req.Staged.Artifact.ByteSize,
		Width:             req.Staged.Width,
		Height:            req.Staged.Height,
		Title:             title,
		Attribution:       strings.TrimSpace(req.Attribution),
		License:           license,
		SHA256:            req.Staged.Artifact.SHA256,
		InspectionSupport: sourceMediaInspectionSupport(MediaKindImage),
	}
	locators, err := json.Marshal([]MediaLocator{locator})
	if err != nil {
		return MediaSourceSnapshotResult{}, err
	}
	result, err := store.CreateExistingArtifactSourceSnapshotWithEvent(ctx, CreateExistingArtifactSourceSnapshotWithEventRequest{
		Snapshot: CreateSourceSnapshotRequest{
			SnapshotID: req.SnapshotID,
			MissionID:  req.MissionID,
			Connector: ConnectorRef{
				ConnectorID:      SourceConnectorTypeMediaURL,
				ConnectorType:    SourceConnectorTypeMediaURL,
				ExternalSourceID: req.URL,
				ExternalURI:      req.URL,
				ExternalVersion:  req.Staged.ExternalVersion,
				ConnectorVersion: "plasma-ui.media-url.v1",
			},
			Title:             title,
			ExternalUpdatedAt: req.Staged.ExternalUpdatedAt,
			ArtifactIDs:       []string{req.Staged.Artifact.ArtifactID},
			Locators:          json.RawMessage(locators),
			Access:            SourceAccess{License: license},
		},
		Event: AppendEventRequest{
			EventID:   req.EventID,
			MissionID: req.MissionID,
			EventType: sourceevents.SourceSnapshottedEventType,
			Producer:  req.Producer,
			Payload: mustMarshalJSON(map[string]any{
				"snapshot_id":                        req.SnapshotID,
				"artifact_ids":                       []string{req.Staged.Artifact.ArtifactID},
				"source_kind":                        SourceConnectorTypeMediaURL,
				"media_kind":                         MediaKindImage,
				"title":                              title,
				"url":                                req.URL,
				"source_candidate_proposal_event_id": req.Staged.ProposalEventID,
				"source_candidate_artifact_reused":   true,
			}),
		},
	})
	if err != nil {
		return MediaSourceSnapshotResult{}, err
	}
	return MediaSourceSnapshotResult{Artifact: req.Staged.Artifact, HasArtifact: true, Snapshot: result.Snapshot, Event: result.Event}, nil
}

func createFetchedImageMediaURLSource(ctx context.Context, store Store, req CreateFetchedMediaURLSourceRequest, title string, license string, locator MediaLocator) (MediaSourceSnapshotResult, error) {
	contentSHA := sha256HexBytes(req.Fetched.Content)
	locator.SHA256 = contentSHA
	locators, err := json.Marshal([]MediaLocator{locator})
	if err != nil {
		return MediaSourceSnapshotResult{}, err
	}
	result, err := store.CreateSourceSnapshotWithEvent(ctx, CreateSourceSnapshotWithEventRequest{
		Artifact: CreateRawArtifactRequest{
			ArtifactID:     req.ArtifactID,
			MissionID:      req.MissionID,
			MediaType:      req.Fetched.MediaType,
			Filename:       sourceMediaFilename(title, req.Fetched.MediaType),
			Producer:       req.Producer,
			Content:        req.Fetched.Content,
			ExpectedSHA256: contentSHA,
		},
		Snapshot: CreateSourceSnapshotRequest{
			SnapshotID: req.SnapshotID,
			MissionID:  req.MissionID,
			Connector: ConnectorRef{
				ConnectorID:      SourceConnectorTypeMediaURL,
				ConnectorType:    SourceConnectorTypeMediaURL,
				ExternalSourceID: req.URL,
				ExternalURI:      req.URL,
				ExternalVersion:  req.Fetched.ExternalVersion,
				ConnectorVersion: "plasma-ui.media-url.v1",
			},
			Title:             title,
			ExternalUpdatedAt: req.Fetched.ExternalUpdatedAt,
			ArtifactIDs:       []string{req.ArtifactID},
			Locators:          json.RawMessage(locators),
			Access: SourceAccess{
				License: license,
			},
		},
		Event: AppendEventRequest{
			EventID:   req.EventID,
			MissionID: req.MissionID,
			EventType: sourceevents.SourceSnapshottedEventType,
			Producer:  req.Producer,
			Payload: mustMarshalJSON(map[string]any{
				"snapshot_id":  req.SnapshotID,
				"artifact_ids": []string{req.ArtifactID},
				"source_kind":  SourceConnectorTypeMediaURL,
				"media_kind":   req.Fetched.MediaKind,
				"title":        title,
				"url":          req.URL,
			}),
		},
	})
	if err != nil {
		return MediaSourceSnapshotResult{}, err
	}
	return MediaSourceSnapshotResult{Artifact: result.Artifact, HasArtifact: true, Snapshot: result.Snapshot, Event: result.Event}, nil
}

func createFetchedLiveMediaURLSource(ctx context.Context, store Store, req CreateFetchedMediaURLSourceRequest, title string, license string, locator MediaLocator) (MediaSourceSnapshotResult, error) {
	locators, err := json.Marshal([]MediaLocator{locator})
	if err != nil {
		return MediaSourceSnapshotResult{}, err
	}
	result, err := store.CreateLiveSourceSnapshotWithEvent(ctx, CreateLiveSourceSnapshotWithEventRequest{
		Snapshot: CreateSourceSnapshotRequest{
			SnapshotID: req.SnapshotID,
			MissionID:  req.MissionID,
			Connector: ConnectorRef{
				ConnectorID:      SourceConnectorTypeMediaURL,
				ConnectorType:    SourceConnectorTypeMediaURL,
				ExternalSourceID: req.URL,
				ExternalURI:      req.URL,
				ExternalVersion:  req.Fetched.ExternalVersion,
				ConnectorVersion: "plasma-ui.media-url.v1",
			},
			Title:             title,
			ExternalUpdatedAt: req.Fetched.ExternalUpdatedAt,
			Locators:          json.RawMessage(locators),
			Access: SourceAccess{
				License:         license,
				RetrievalPolicy: SourceRetrievalPolicyLiveReference,
			},
		},
		Event: AppendEventRequest{
			EventID:   req.EventID,
			MissionID: req.MissionID,
			EventType: sourceevents.SourceSnapshottedEventType,
			Producer:  req.Producer,
			Payload: mustMarshalJSON(map[string]any{
				"snapshot_id": req.SnapshotID,
				"source_kind": SourceConnectorTypeMediaURL,
				"media_kind":  req.Fetched.MediaKind,
				"title":       title,
				"url":         req.URL,
			}),
		},
	})
	if err != nil {
		return MediaSourceSnapshotResult{}, err
	}
	return MediaSourceSnapshotResult{Snapshot: result.Snapshot, Event: result.Event}, nil
}

func stagedImageMediaType(mediaType string) bool {
	base, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		base = mediaType
	}
	switch strings.ToLower(strings.TrimSpace(base)) {
	case "image/png", "image/jpeg", "image/gif":
		return true
	default:
		return false
	}
}

func sourceMediaInspectionSupport(mediaKind string) string {
	switch strings.TrimSpace(mediaKind) {
	case MediaKindImage:
		return "metadata_only_until_vision_engine_configured"
	case MediaKindAudio, MediaKindVideo:
		return "inspect_unsupported"
	default:
		return "unsupported"
	}
}

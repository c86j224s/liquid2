package sourceingest

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
	"github.com/c86j224s/liquid2/plasma/internal/sources/pdftext"
)

func CreateFetchedPDFURLSourceWithEvent(ctx context.Context, store Store, req CreateFetchedPDFURLSourceRequest) (URLSourceSnapshotResult, error) {
	title := firstNonEmptyString(req.Title, req.Fetched.Title, req.URL)
	fetchedAt := req.FetchedAt.UTC()
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	contentSHA := sha256HexBytes(req.Fetched.Content)
	locators, err := json.Marshal([]map[string]any{{
		"locator_type":       SourceLocatorTypePDFDocument,
		"url":                req.URL,
		"fetched_at":         fetchedAt.Format(time.RFC3339Nano),
		"mime_type":          req.Fetched.MediaType,
		"byte_size":          req.Fetched.ByteSize,
		"sha256":             contentSHA,
		"page_count":         req.Fetched.PageCount,
		"text_length":        req.Fetched.TextLength,
		"text_length_known":  req.Fetched.TextLengthKnown,
		"extraction_support": "pdf_text",
	}})
	if err != nil {
		return URLSourceSnapshotResult{}, err
	}
	result, err := store.CreateSourceSnapshotWithEvent(ctx, CreateSourceSnapshotWithEventRequest{
		Artifact: CreateRawArtifactRequest{
			ArtifactID:     req.ArtifactID,
			MissionID:      req.MissionID,
			MediaType:      req.Fetched.MediaType,
			Filename:       sourceIngestFilename(title, ".pdf"),
			Producer:       req.Producer,
			Content:        req.Fetched.Content,
			ExpectedSHA256: contentSHA,
		},
		Snapshot: CreateSourceSnapshotRequest{
			SnapshotID: req.SnapshotID,
			MissionID:  req.MissionID,
			Connector: ConnectorRef{
				ConnectorID:      SourceConnectorTypePDFURL,
				ConnectorType:    SourceConnectorTypePDFURL,
				ExternalSourceID: req.URL,
				ExternalURI:      req.URL,
				ExternalVersion:  req.Fetched.ExternalVersion,
				ConnectorVersion: "plasma-ui.pdf-url.v1",
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
				"snapshot_id":       req.SnapshotID,
				"artifact_ids":      []string{req.ArtifactID},
				"source_kind":       SourceConnectorTypePDFURL,
				"title":             title,
				"url":               req.URL,
				"page_count":        req.Fetched.PageCount,
				"text_length":       req.Fetched.TextLength,
				"text_length_known": req.Fetched.TextLengthKnown,
			}),
		},
	})
	if err != nil {
		return URLSourceSnapshotResult{}, err
	}
	return URLSourceSnapshotResult{Artifact: result.Artifact, Snapshot: result.Snapshot, Event: result.Event}, nil
}

func CreateStagedPDFURLSourceWithEvent(ctx context.Context, store Store, req CreateStagedPDFURLSourceRequest) (URLSourceSnapshotResult, error) {
	title := firstNonEmptyString(req.Title, req.Staged.Title, req.URL)
	contentSHA := req.Staged.Artifact.SHA256
	if contentSHA == "" {
		contentSHA = sha256HexBytes(req.Staged.Artifact.Content)
	}
	info, err := pdftext.Inspect(req.Staged.Artifact.Content)
	if err != nil {
		return URLSourceSnapshotResult{}, fmt.Errorf("%w: PDF inspection failed: %v", ErrInvalidInput, err)
	}
	byteSize := req.Staged.Artifact.ByteSize
	if byteSize == 0 {
		byteSize = int64(len(req.Staged.Artifact.Content))
	}
	locators, err := json.Marshal([]map[string]any{{
		"locator_type":       SourceLocatorTypePDFDocument,
		"url":                req.URL,
		"fetched_at":         req.Staged.Artifact.CreatedAt.Format(time.RFC3339Nano),
		"staged_from":        req.Staged.ProposalEventID,
		"mime_type":          pdftext.MediaType,
		"byte_size":          byteSize,
		"sha256":             contentSHA,
		"page_count":         info.PageCount,
		"text_length":        0,
		"text_length_known":  false,
		"extraction_support": "pdf_text",
	}})
	if err != nil {
		return URLSourceSnapshotResult{}, err
	}
	result, err := store.CreateExistingArtifactSourceSnapshotWithEvent(ctx, CreateExistingArtifactSourceSnapshotWithEventRequest{
		Snapshot: CreateSourceSnapshotRequest{
			SnapshotID: req.SnapshotID,
			MissionID:  req.MissionID,
			Connector: ConnectorRef{
				ConnectorID:      SourceConnectorTypePDFURL,
				ConnectorType:    SourceConnectorTypePDFURL,
				ExternalSourceID: req.URL,
				ExternalURI:      req.URL,
				ExternalVersion:  req.Staged.ExternalVersion,
				ConnectorVersion: "plasma-ui.pdf-url.v1",
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
				"source_kind":                        SourceConnectorTypePDFURL,
				"title":                              title,
				"url":                                req.URL,
				"page_count":                         info.PageCount,
				"text_length":                        0,
				"text_length_known":                  false,
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

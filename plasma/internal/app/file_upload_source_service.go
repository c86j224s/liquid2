package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
)

const (
	UploadedFileMaxBytes = int64(100 * 1024 * 1024)

	UploadedContentKindText  = "text"
	UploadedContentKindPDF   = "pdf"
	UploadedContentKindImage = "image"
)

type CreateUploadedFileSourceRequest struct {
	MissionID        string
	ArtifactID       string
	SnapshotID       string
	EventID          string
	Title            string
	OriginalFilename string
	Content          []byte
	Producer         Producer
	UploadedAt       time.Time
}

type UploadedFileSourceResult struct {
	Artifact RawArtifact
	Snapshot SourceSnapshot
	Event    LedgerEvent
	Existing bool
}

func (s *Service) CreateUploadedFileSourceWithEvent(ctx context.Context, req CreateUploadedFileSourceRequest) (UploadedFileSourceResult, error) {
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return UploadedFileSourceResult{}, err
	}
	if len(req.Content) == 0 {
		return UploadedFileSourceResult{}, fmt.Errorf("%w: uploaded source content is required", ErrInvalidInput)
	}
	if int64(len(req.Content)) > UploadedFileMaxBytes {
		return UploadedFileSourceResult{}, fmt.Errorf("%w: uploaded source exceeds 100 MiB limit", ErrInvalidInput)
	}
	if err := validateProducer(req.Producer); err != nil {
		return UploadedFileSourceResult{}, err
	}
	mediaType, contentKind, err := classifyUploadedFile(req.OriginalFilename, req.Content)
	if err != nil {
		return UploadedFileSourceResult{}, err
	}
	uploadedAt := req.UploadedAt.UTC()
	if uploadedAt.IsZero() {
		uploadedAt = time.Now().UTC()
	}
	sum := sha256.Sum256(req.Content)
	sha := hex.EncodeToString(sum[:])
	filename := sanitizeUploadedFilename(req.OriginalFilename, mediaType)
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = filename
	}
	existing, found, err := s.findUploadedFileRawArtifactByMissionSHA(ctx, missionID, sha)
	if err != nil {
		return UploadedFileSourceResult{}, err
	}
	artifact := existing
	artifactID := strings.TrimSpace(req.ArtifactID)
	if !found {
		if err := validateID("art_", artifactID); err != nil {
			return UploadedFileSourceResult{}, err
		}
		artifact, err = buildRawArtifact(CreateRawArtifactRequest{
			ArtifactID:     artifactID,
			MissionID:      missionID,
			MediaType:      mediaType,
			Filename:       filename,
			Producer:       req.Producer,
			Content:        req.Content,
			ExpectedSHA256: sha,
		})
		if err != nil {
			return UploadedFileSourceResult{}, err
		}
	}
	locator, err := json.Marshal([]UploadedFileLocator{{
		LocatorType:       uploadedFileLocatorType(contentKind),
		MediaKind:         uploadedFileMediaKind(contentKind),
		OriginalFilename:  strings.TrimSpace(req.OriginalFilename),
		SanitizedFilename: filename,
		MIMEType:          mediaType,
		ByteSize:          int64(len(req.Content)),
		SHA256:            sha,
		UploadedAt:        uploadedAt,
		ContentKind:       contentKind,
	}})
	if err != nil {
		return UploadedFileSourceResult{}, err
	}
	snapshot, err := s.buildSourceSnapshot(ctx, CreateSourceSnapshotRequest{
		SnapshotID: strings.TrimSpace(req.SnapshotID),
		MissionID:  missionID,
		Connector: ConnectorRef{
			ConnectorID:      "file_upload",
			ConnectorType:    SourceConnectorTypeFileUpload,
			ExternalSourceID: "file_upload:" + sha,
			ExternalURI:      "file-upload://" + sha,
			ExternalVersion:  sha,
			ConnectorVersion: "plasma.file_upload.v1",
		},
		Title:             title,
		ExternalUpdatedAt: uploadedAt,
		ArtifactIDs:       []string{artifact.ArtifactID},
		ContentHash:       ContentHash{Algorithm: "sha256", Value: sha},
		Locators:          locator,
		Access:            SourceAccess{RetrievalPolicy: SourceRetrievalPolicySnapshotOnly},
	}, []RawArtifact{artifact})
	if err != nil {
		return UploadedFileSourceResult{}, err
	}
	event, err := buildLedgerEvent(AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: missionID,
		EventType: sourceevents.SourceSnapshottedEventType,
		Producer:  req.Producer,
		Payload: sourceevents.BuildUploadedFileSourceSnapshottedPayload(sourceevents.UploadedFileSourceSnapshottedPayloadRequest{
			SnapshotID:        snapshot.SnapshotID,
			ArtifactIDs:       snapshot.ArtifactIDs,
			Title:             title,
			OriginalFilename:  req.OriginalFilename,
			SanitizedFilename: filename,
			MediaType:         mediaType,
			ContentKind:       contentKind,
			SHA256:            sha,
			Deduplicated:      found,
		}),
	})
	if err != nil {
		return UploadedFileSourceResult{}, err
	}
	write := AtomicWrite{
		Events:          []LedgerEvent{event},
		SourceSnapshots: []SourceSnapshot{snapshot},
	}
	if !found {
		write.RawArtifacts = []RawArtifact{artifact}
	}
	committed, err := s.commitAtomicWrite(ctx, write)
	if err != nil {
		return UploadedFileSourceResult{}, err
	}
	return UploadedFileSourceResult{Artifact: artifact, Snapshot: snapshot, Event: committed.Events[0], Existing: found}, nil
}

func uploadedFileLocatorType(contentKind string) string {
	switch contentKind {
	case UploadedContentKindPDF:
		return SourceLocatorTypePDFDocument
	case UploadedContentKindImage:
		return SourceLocatorTypeMedia
	default:
		return SourceLocatorTypeFullDocument
	}
}

func uploadedFileMediaKind(contentKind string) string {
	if contentKind == UploadedContentKindImage {
		return MediaKindImage
	}
	return ""
}

func (s *Service) findUploadedFileRawArtifactByMissionSHA(ctx context.Context, missionID string, sha string) (RawArtifact, bool, error) {
	store, ok := s.store.(RawArtifactListStore)
	if !ok {
		return RawArtifact{}, false, fmt.Errorf("%w: raw artifact list store is required", ErrInvalidInput)
	}
	snapshotStore, ok := s.store.(SourceSnapshotListStore)
	if !ok {
		return RawArtifact{}, false, fmt.Errorf("%w: source snapshot list store is required", ErrInvalidInput)
	}
	snapshots, err := snapshotStore.ListSourceSnapshots(ctx, missionID)
	if err != nil {
		return RawArtifact{}, false, err
	}
	uploadArtifactIDs := make(map[string]struct{})
	for _, snapshot := range snapshots {
		if snapshot.Connector.ConnectorType != SourceConnectorTypeFileUpload {
			continue
		}
		if snapshot.ContentHash.Algorithm != "" &&
			!strings.EqualFold(snapshot.ContentHash.Algorithm, "sha256") {
			continue
		}
		if snapshot.ContentHash.Value != "" &&
			!strings.EqualFold(snapshot.ContentHash.Value, sha) {
			continue
		}
		for _, artifactID := range snapshot.ArtifactIDs {
			uploadArtifactIDs[artifactID] = struct{}{}
		}
	}
	artifacts, err := store.ListRawArtifacts(ctx, missionID)
	if err != nil {
		return RawArtifact{}, false, err
	}
	var sameSHAOutsideUpload bool
	for _, artifact := range artifacts {
		if strings.EqualFold(artifact.SHA256, sha) {
			if _, ok := uploadArtifactIDs[artifact.ArtifactID]; ok {
				return artifact, true, nil
			}
			sameSHAOutsideUpload = true
		}
	}
	if sameSHAOutsideUpload {
		return RawArtifact{}, false, fmt.Errorf("%w: uploaded source content matches an existing non-upload artifact; refusing to reuse result/report material as a source artifact", ErrConflict)
	}
	return RawArtifact{}, false, nil
}

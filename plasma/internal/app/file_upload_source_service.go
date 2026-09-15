package app

import uploadsource "github.com/c86j224s/liquid2/plasma/internal/source"

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/source"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
)

const (
	UploadedFileMaxBytes = int64(100 * 1024 * 1024)
)

// CreateUploadedFileSourceRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type CreateUploadedFileSourceRequest struct {
	MissionID        string
	ArtifactID       string
	SnapshotID       string
	EventID          string
	Title            string
	OriginalFilename string
	Content          []byte
	Producer         ledger.Producer
	UploadedAt       time.Time
}

// UploadedFileSourceResult는 업로드 source snapshot과 기록된 장부 이벤트를 함께 반환한다.
type UploadedFileSourceResult struct {
	Artifact artifactcontract.Raw
	Snapshot sourcecontract.Snapshot
	Event    ledger.Event
	Existing bool
}

// CreateUploadedFileSourceWithEvent는 업로드 artifact, source snapshot, 이벤트를 함께 기록한다.
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
	mediaType, contentKind, err := uploadsource.ClassifyUploadedFile(req.OriginalFilename, req.Content)
	if err != nil {
		return UploadedFileSourceResult{}, err
	}
	uploadedAt := req.UploadedAt.UTC()
	if uploadedAt.IsZero() {
		uploadedAt = time.Now().UTC()
	}
	sum := sha256.Sum256(req.Content)
	sha := hex.EncodeToString(sum[:])
	filename := uploadsource.SanitizeUploadedFilename(req.OriginalFilename, mediaType)
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
		artifact, err = artifactcontract.Build(artifactcontract.CreateRequest{
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
	locator, err := json.Marshal([]source.UploadedFileLocator{{
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
	snapshot, err := source.BuildSnapshot(ctx, s.store, source.CreateRequest{
		SnapshotID: strings.TrimSpace(req.SnapshotID),
		MissionID:  missionID,
		Connector: sourcecontract.ConnectorRef{
			ConnectorID:      "file_upload",
			ConnectorType:    sourcecontract.ConnectorTypeFileUpload,
			ExternalSourceID: "file_upload:" + sha,
			ExternalURI:      "file-upload://" + sha,
			ExternalVersion:  sha,
			ConnectorVersion: "plasma.file_upload.v1",
		},
		Title:             title,
		ExternalUpdatedAt: uploadedAt,
		ArtifactIDs:       []string{artifact.ArtifactID},
		ContentHash:       sourcecontract.ContentHash{Algorithm: "sha256", Value: sha},
		Locators:          locator,
		Access:            sourcecontract.Access{RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly},
	}, []artifactcontract.Raw{artifact})
	if err != nil {
		return UploadedFileSourceResult{}, err
	}
	event, err := buildLedgerEvent(ledger.AppendRequest{
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
		Events:          []ledger.Event{event},
		SourceSnapshots: []sourcecontract.Snapshot{snapshot},
	}
	if !found {
		write.RawArtifacts = []artifactcontract.Raw{artifact}
	}
	committed, err := s.commitAtomicWrite(ctx, write)
	if err != nil {
		return UploadedFileSourceResult{}, err
	}
	return UploadedFileSourceResult{Artifact: artifact, Snapshot: snapshot, Event: committed.Events[0], Existing: found}, nil
}

func uploadedFileLocatorType(contentKind string) string {
	switch contentKind {
	case uploadsource.UploadedContentKindPDF:
		return sourcecontract.LocatorTypePDFDocument
	case uploadsource.UploadedContentKindImage:
		return sourcecontract.LocatorTypeMedia
	default:
		return sourcecontract.LocatorTypeFullDocument
	}
}

func uploadedFileMediaKind(contentKind string) string {
	if contentKind == uploadsource.UploadedContentKindImage {
		return sourcecontract.MediaKindImage
	}
	return ""
}

func (s *Service) findUploadedFileRawArtifactByMissionSHA(ctx context.Context, missionID string, sha string) (artifactcontract.Raw, bool, error) {
	store, ok := s.store.(RawArtifactListStore)
	if !ok {
		return artifactcontract.Raw{}, false, fmt.Errorf("%w: raw artifact list store is required", ErrInvalidInput)
	}
	snapshotStore, ok := s.store.(sourcecontract.ListStore)
	if !ok {
		return artifactcontract.Raw{}, false, fmt.Errorf("%w: source snapshot list store is required", ErrInvalidInput)
	}
	snapshots, err := snapshotStore.ListSourceSnapshots(ctx, missionID)
	if err != nil {
		return artifactcontract.Raw{}, false, err
	}
	uploadArtifactIDs := make(map[string]struct{})
	for _, snapshot := range snapshots {
		if snapshot.Connector.ConnectorType != sourcecontract.ConnectorTypeFileUpload {
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
		return artifactcontract.Raw{}, false, err
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
		return artifactcontract.Raw{}, false, fmt.Errorf("%w: uploaded source content matches an existing non-upload artifact; refusing to reuse result/report material as a source artifact", ErrConflict)
	}
	return artifactcontract.Raw{}, false, nil
}

package app

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidateevents"
)

// ErrSourceDownloadNotFound is returned for every source-download eligibility
// failure. The transport deliberately maps it to one indistinguishable 404.
var ErrSourceDownloadNotFound = errors.New("source download not found")

// SourceCandidateDownloadRequest identifies one exact staged terminal event and
// artifact. The event ID prevents a URL from selecting an older or newer stage.
type SourceCandidateDownloadRequest struct {
	MissionID     string
	StagedEventID string
	ArtifactID    string
}

// SourceSnapshotDownloadRequest identifies an approved source snapshot and its
// exact stored artifact. A blank artifact ID is allowed only for one-artifact
// snapshots.
type SourceSnapshotDownloadRequest struct {
	MissionID  string
	SnapshotID string
	ArtifactID string
}

// ResolveSourceCandidateDownload applies candidate lifecycle and provenance
// policy before returning durable artifact bytes. It never calls a connector,
// fetcher, renderer, or filesystem adapter.
func (s *Service) ResolveSourceCandidateDownload(ctx context.Context, req SourceCandidateDownloadRequest) (RawArtifact, error) {
	missionID := strings.TrimSpace(req.MissionID)
	stagedEventID := strings.TrimSpace(req.StagedEventID)
	artifactID := strings.TrimSpace(req.ArtifactID)
	if err := validateID("mis_", missionID); err != nil || stagedEventID == "" || artifactID == "" {
		return RawArtifact{}, ErrSourceDownloadNotFound
	}

	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return RawArtifact{}, err
	}
	_, selectedPayload, ok := latestCandidateStagedEvent(events, stagedEventID)
	if !ok || strings.TrimSpace(selectedPayload.ArtifactID) != artifactID {
		return RawArtifact{}, ErrSourceDownloadNotFound
	}
	normalizedURL, err := normalizeDownloadCandidateURL(selectedPayload.URL)
	if err != nil || sourceCandidateDownloadExcluded(normalizedURL) {
		return RawArtifact{}, ErrSourceDownloadNotFound
	}
	latestEvent, latestPayload, latestState, found := sourcecandidateevents.LatestStagingTerminalEventForURL(sourceDownloadEvents(events), normalizedURL, normalizeDownloadCandidateURL)
	if !found || latestState != sourcecandidateevents.StagedEventType || latestEvent.EventID != stagedEventID || strings.TrimSpace(latestPayload.ArtifactID) != artifactID {
		return RawArtifact{}, ErrSourceDownloadNotFound
	}
	if candidateLatestDecisionRejected(events, normalizedURL) {
		return RawArtifact{}, ErrSourceDownloadNotFound
	}

	snapshots, err := s.ListSourceSnapshotsWithState(ctx, ListSourceSnapshotsRequest{
		MissionID: missionID, IncludeRemoved: true, IncludeSuperseded: true,
	})
	if err != nil {
		return RawArtifact{}, err
	}
	if candidateConsumed(events, snapshots, artifactID, normalizedURL) {
		return RawArtifact{}, ErrSourceDownloadNotFound
	}
	return s.downloadArtifactForMission(ctx, missionID, artifactID)
}

// ResolveSourceCandidateDownloadByIDs is the transport-neutral primitive boundary
// for the candidate download route.
func (s *Service) ResolveSourceCandidateDownloadByIDs(ctx context.Context, missionID, stagedEventID, artifactID string) (RawArtifact, error) {
	return s.ResolveSourceCandidateDownload(ctx, SourceCandidateDownloadRequest{MissionID: missionID, StagedEventID: stagedEventID, ArtifactID: artifactID})
}

// ResolveSourceSnapshotDownloadByIDs is the transport-neutral primitive boundary
// for the approved-source download route.
func (s *Service) ResolveSourceSnapshotDownloadByIDs(ctx context.Context, missionID, snapshotID, artifactID string) (RawArtifact, error) {
	return s.ResolveSourceSnapshotDownload(ctx, SourceSnapshotDownloadRequest{MissionID: missionID, SnapshotID: snapshotID, ArtifactID: artifactID})
}

// ResolveSourceSnapshotDownload applies approved-source lifecycle policy before
// returning the exact attached stored artifact. Live references never resolve.
func (s *Service) ResolveSourceSnapshotDownload(ctx context.Context, req SourceSnapshotDownloadRequest) (RawArtifact, error) {
	missionID := strings.TrimSpace(req.MissionID)
	snapshotID := strings.TrimSpace(req.SnapshotID)
	requestedArtifactID := strings.TrimSpace(req.ArtifactID)
	if err := validateID("mis_", missionID); err != nil || snapshotID == "" {
		return RawArtifact{}, ErrSourceDownloadNotFound
	}
	snapshot, err := s.GetSourceSnapshot(ctx, snapshotID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrInvalidInput) {
			return RawArtifact{}, ErrSourceDownloadNotFound
		}
		return RawArtifact{}, err
	}
	if snapshot.MissionID != missionID || snapshot.State.Removed || snapshot.State.Superseded ||
		snapshot.State.State == SourceStateRemoved || snapshot.Access.RetrievalPolicy != SourceRetrievalPolicySnapshotOnly ||
		connectorSnapshotDownloadExcluded(snapshot.Connector) {
		return RawArtifact{}, ErrSourceDownloadNotFound
	}
	artifactID := requestedArtifactID
	if artifactID == "" && len(snapshot.ArtifactIDs) == 1 {
		artifactID = strings.TrimSpace(snapshot.ArtifactIDs[0])
	}
	if len(snapshot.ArtifactIDs) != 1 || artifactID == "" || !snapshotHasDownloadArtifact(snapshot.ArtifactIDs, artifactID) {
		return RawArtifact{}, ErrSourceDownloadNotFound
	}
	return s.downloadArtifactForMission(ctx, missionID, artifactID)
}

func (s *Service) downloadArtifactForMission(ctx context.Context, missionID, artifactID string) (RawArtifact, error) {
	artifact, err := s.GetRawArtifact(ctx, artifactID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrInvalidInput) {
			return RawArtifact{}, ErrSourceDownloadNotFound
		}
		return RawArtifact{}, err
	}
	if artifact.MissionID != missionID || strings.TrimSpace(artifact.ArtifactID) != artifactID {
		return RawArtifact{}, ErrSourceDownloadNotFound
	}
	return artifact, nil
}

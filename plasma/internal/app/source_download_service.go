package app

import (
	"context"
	"database/sql"
	"errors"
	"github.com/c86j224s/liquid2/plasma/internal/source"
	"strings"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
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
func (s *Service) ResolveSourceCandidateDownload(ctx context.Context, req SourceCandidateDownloadRequest) (artifactcontract.Raw, error) {
	missionID := strings.TrimSpace(req.MissionID)
	stagedEventID := strings.TrimSpace(req.StagedEventID)
	artifactID := strings.TrimSpace(req.ArtifactID)
	if err := validateID("mis_", missionID); err != nil || stagedEventID == "" || artifactID == "" {
		return artifactcontract.Raw{}, ErrSourceDownloadNotFound
	}

	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return artifactcontract.Raw{}, err
	}
	_, selectedPayload, ok := latestCandidateStagedEvent(events, stagedEventID)
	if !ok || strings.TrimSpace(selectedPayload.ArtifactID) != artifactID {
		return artifactcontract.Raw{}, ErrSourceDownloadNotFound
	}
	normalizedURL, err := sourcecandidateevents.NormalizeDownloadURL(selectedPayload.URL)
	if err != nil || sourcecandidateevents.DownloadCandidateExcluded(normalizedURL) {
		return artifactcontract.Raw{}, ErrSourceDownloadNotFound
	}
	latestEvent, latestPayload, latestState, found := sourcecandidateevents.LatestStagingTerminalEventForURL(sourceDownloadEvents(events), normalizedURL, sourcecandidateevents.NormalizeDownloadURL)
	if !found || latestState != sourcecandidateevents.StagedEventType || latestEvent.EventID != stagedEventID || strings.TrimSpace(latestPayload.ArtifactID) != artifactID {
		return artifactcontract.Raw{}, ErrSourceDownloadNotFound
	}
	if sourcecandidateevents.LatestDownloadDecisionRejected(sourceDownloadEvents(events), normalizedURL) {
		return artifactcontract.Raw{}, ErrSourceDownloadNotFound
	}

	snapshots, err := s.ListSourceSnapshotsWithState(ctx, source.ListRequest{
		MissionID: missionID, IncludeRemoved: true, IncludeSuperseded: true,
	})
	if err != nil {
		return artifactcontract.Raw{}, err
	}
	if sourcecandidateevents.DownloadCandidateConsumed(sourceDownloadEvents(events), snapshots, artifactID, normalizedURL) {
		return artifactcontract.Raw{}, ErrSourceDownloadNotFound
	}
	return s.downloadArtifactForMission(ctx, missionID, artifactID)
}

// ResolveSourceCandidateDownloadByIDs is the transport-neutral primitive boundary
// for the candidate download route.
func (s *Service) ResolveSourceCandidateDownloadByIDs(ctx context.Context, missionID, stagedEventID, artifactID string) (artifactcontract.Raw, error) {
	return s.ResolveSourceCandidateDownload(ctx, SourceCandidateDownloadRequest{MissionID: missionID, StagedEventID: stagedEventID, ArtifactID: artifactID})
}

// ResolveSourceSnapshotDownloadByIDs is the transport-neutral primitive boundary
// for the approved-source download route.
func (s *Service) ResolveSourceSnapshotDownloadByIDs(ctx context.Context, missionID, snapshotID, artifactID string) (artifactcontract.Raw, error) {
	return s.ResolveSourceSnapshotDownload(ctx, SourceSnapshotDownloadRequest{MissionID: missionID, SnapshotID: snapshotID, ArtifactID: artifactID})
}

// ResolveSourceSnapshotDownload applies approved-source lifecycle policy before
// returning the exact attached stored artifact. Live references never resolve.
func (s *Service) ResolveSourceSnapshotDownload(ctx context.Context, req SourceSnapshotDownloadRequest) (artifactcontract.Raw, error) {
	missionID := strings.TrimSpace(req.MissionID)
	snapshotID := strings.TrimSpace(req.SnapshotID)
	requestedArtifactID := strings.TrimSpace(req.ArtifactID)
	if err := validateID("mis_", missionID); err != nil || snapshotID == "" {
		return artifactcontract.Raw{}, ErrSourceDownloadNotFound
	}
	snapshot, err := s.GetSourceSnapshot(ctx, snapshotID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrInvalidInput) {
			return artifactcontract.Raw{}, ErrSourceDownloadNotFound
		}
		return artifactcontract.Raw{}, err
	}
	if snapshot.MissionID != missionID {
		return artifactcontract.Raw{}, ErrSourceDownloadNotFound
	}
	artifactID, ok := source.SnapshotDownloadArtifact(snapshot, requestedArtifactID, connectorSnapshotDownloadExcluded)
	if !ok {
		return artifactcontract.Raw{}, ErrSourceDownloadNotFound
	}
	return s.downloadArtifactForMission(ctx, missionID, artifactID)
}

func (s *Service) downloadArtifactForMission(ctx context.Context, missionID, artifactID string) (artifactcontract.Raw, error) {
	artifact, err := s.GetRawArtifact(ctx, artifactID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrInvalidInput) {
			return artifactcontract.Raw{}, ErrSourceDownloadNotFound
		}
		return artifactcontract.Raw{}, err
	}
	if artifact.MissionID != missionID || strings.TrimSpace(artifact.ArtifactID) != artifactID {
		return artifactcontract.Raw{}, ErrSourceDownloadNotFound
	}
	return artifact, nil
}

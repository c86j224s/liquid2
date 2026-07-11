package sourcecandidates

import (
	"context"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidateevents"
)

func reusableArtifactBySHA(ctx context.Context, store Store, missionID string, sha string) (app.RawArtifact, bool, error) {
	artifacts, err := store.ListRawArtifacts(ctx, missionID)
	if err != nil {
		return app.RawArtifact{}, false, err
	}
	for _, artifact := range artifacts {
		if strings.EqualFold(strings.TrimSpace(artifact.SHA256), strings.TrimSpace(sha)) {
			reusable, err := artifactReusable(ctx, store, missionID, artifact.ArtifactID)
			if err != nil {
				return app.RawArtifact{}, false, err
			}
			if !reusable {
				continue
			}
			return artifact, true, nil
		}
	}
	return app.RawArtifact{}, false, nil
}

func artifactReusable(ctx context.Context, store Store, missionID string, artifactID string) (bool, error) {
	artifactID = strings.TrimSpace(artifactID)
	events, err := store.ListEvents(ctx, missionID)
	if err != nil {
		return false, err
	}
	snapshots, err := store.ListSourceSnapshotsWithState(ctx, app.ListSourceSnapshotsRequest{
		MissionID:         missionID,
		IncludeRemoved:    true,
		IncludeSuperseded: true,
	})
	if err != nil {
		return false, err
	}
	return sourcecandidateevents.IsOpenStagedArtifact(
		sourceCandidateEventsFromApp(events),
		sourceCandidateSnapshotsFromApp(snapshots),
		artifactID,
	) || artifactIsAttachedToAnySnapshot(snapshots, artifactID), nil
}

func artifactIsAttachedToAnySnapshot(snapshots []app.SourceSnapshot, artifactID string) bool {
	for _, snapshot := range snapshots {
		for _, snapshotArtifactID := range snapshot.ArtifactIDs {
			if strings.TrimSpace(snapshotArtifactID) == artifactID {
				return true
			}
		}
	}
	return false
}

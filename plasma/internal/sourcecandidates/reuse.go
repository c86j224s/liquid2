package sourcecandidates

import (
	"context"
	"strings"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidateevents"
)

func reusableArtifactBySHA(ctx context.Context, store Store, missionID string, sha string) (artifactcontract.Raw, bool, error) {
	artifacts, err := store.ListRawArtifacts(ctx, missionID)
	if err != nil {
		return artifactcontract.Raw{}, false, err
	}
	for _, artifact := range artifacts {
		if strings.EqualFold(strings.TrimSpace(artifact.SHA256), strings.TrimSpace(sha)) {
			reusable, err := artifactReusable(ctx, store, missionID, artifact.ArtifactID)
			if err != nil {
				return artifactcontract.Raw{}, false, err
			}
			if !reusable {
				continue
			}
			return artifact, true, nil
		}
	}
	return artifactcontract.Raw{}, false, nil
}

func artifactReusable(ctx context.Context, store Store, missionID string, artifactID string) (bool, error) {
	artifactID = strings.TrimSpace(artifactID)
	events, err := store.ListEvents(ctx, missionID)
	if err != nil {
		return false, err
	}
	snapshots, err := store.ListSourceSnapshotsWithState(ctx, sourcecontract.ListRequest{
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

func artifactIsAttachedToAnySnapshot(snapshots []sourcecontract.Snapshot, artifactID string) bool {
	for _, snapshot := range snapshots {
		for _, snapshotArtifactID := range snapshot.ArtifactIDs {
			if strings.TrimSpace(snapshotArtifactID) == artifactID {
				return true
			}
		}
	}
	return false
}

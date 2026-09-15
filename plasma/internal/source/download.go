package source

import "strings"

// SnapshotDownloadArtifact preserves the single-artifact download contract.
// Mission binding and artifact IO remain the caller's responsibility; even an
// explicit selector cannot enable a snapshot containing multiple artifacts.
func SnapshotDownloadArtifact(snapshot Snapshot, requested string, excluded func(ConnectorRef) bool) (string, bool) {
	if snapshot.State.Removed || snapshot.State.Superseded || snapshot.State.State == StateRemoved || snapshot.Access.RetrievalPolicy != RetrievalPolicySnapshotOnly || excluded(snapshot.Connector) {
		return "", false
	}
	artifactID := requested
	if artifactID == "" && len(snapshot.ArtifactIDs) == 1 {
		artifactID = strings.TrimSpace(snapshot.ArtifactIDs[0])
	}
	if len(snapshot.ArtifactIDs) != 1 || artifactID == "" {
		return "", false
	}
	for _, attached := range snapshot.ArtifactIDs {
		if strings.TrimSpace(attached) == artifactID {
			return artifactID, true
		}
	}
	return "", false
}

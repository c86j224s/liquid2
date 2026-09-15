package source

import "testing"

func TestSnapshotDownloadPreservesSingleArtifactRestriction(t *testing.T) {
	excluded := func(ConnectorRef) bool { return false }
	snapshot := Snapshot{ArtifactIDs: []string{"art_one"}, Access: Access{RetrievalPolicy: RetrievalPolicySnapshotOnly}}
	if id, ok := SnapshotDownloadArtifact(snapshot, "", excluded); !ok || id != "art_one" {
		t.Fatalf("id=%q ok=%v", id, ok)
	}
	snapshot.ArtifactIDs = append(snapshot.ArtifactIDs, "art_two")
	for _, selector := range []string{"", "art_one", "art_two", "art_missing"} {
		if _, ok := SnapshotDownloadArtifact(snapshot, selector, excluded); ok {
			t.Fatalf("accepted multi-artifact selector %q", selector)
		}
	}
}

package reportilsource

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/source"
	"testing"
)

func TestFrozenRemovedStateStopsBeforeArtifactRead(t *testing.T) {
	readers := FrozenSourceReaders{
		GetSourceSnapshot: func(context.Context, string) (source.Snapshot, error) {
			return source.Snapshot{MissionID: "mis_one", State: source.State{State: " removed "}}, nil
		},
		GetRawArtifact: func(context.Context, string) (artifact.Raw, error) {
			t.Fatal("read artifact after removed state")
			return artifact.Raw{}, nil
		},
	}
	_, err := ResolveFrozenSource(context.Background(), readers, reportilcontract.SourceAccessBinding{Catalog: reportilcontract.SourceCatalog{MissionID: "mis_one"}}, reportilcontract.SourceCatalogEntry{SnapshotID: "src_one"})
	if err == nil || err.Error() != "frozen report IL source is no longer active" {
		t.Fatalf("err=%v", err)
	}
}

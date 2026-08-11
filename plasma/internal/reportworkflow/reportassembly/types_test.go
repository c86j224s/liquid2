package reportassembly

import (
	"context"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestPrepareReaderSourceReturnsDeterministicV1ID(t *testing.T) {
	input := ReaderSourceInput{PlanEventID: "evt_plan", PartArtifactIDs: []string{"art_part_1", "art_part_2"}}
	got, err := (Runner{}).PrepareReaderSource(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	want := reporting.FinalEditReaderSourceArtifactID(input.PlanEventID, input.PartArtifactIDs)
	if got.ArtifactID != want {
		t.Fatalf("reader source artifact id = %q, want %q", got.ArtifactID, want)
	}
}

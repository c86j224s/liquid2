package reporting

import (
	"context"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestFinalEditReaderStartRejectsPartArtifactSHAMismatch(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	readerBinding := finalEditStageStoreStageBinding(
		binding,
		FinalEditStageReader,
		FinalEditReaderSourceArtifactID(binding.PlanEventID, []string{"art_part"}),
		"art_reader_tampered_part",
	)
	store := finalEditTamperedArtifactStore{FinalEditStageStore: svc, artifactID: "art_part"}

	if _, _, err := StartFinalEditStage(ctx, store, "evt_reader_tampered_part_start", readerBinding); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("tampered Part artifact error=%v, want conflict", err)
	}
	if _, err := svc.GetRawArtifact(ctx, readerBinding.SourceArtifactID); err == nil {
		t.Fatal("tampered Part artifact materialized a reader source")
	}
}

type finalEditTamperedArtifactStore struct {
	FinalEditStageStore
	artifactID string
}

func (s finalEditTamperedArtifactStore) GetRawArtifact(ctx context.Context, artifactID string) (app.RawArtifact, error) {
	artifact, err := s.FinalEditStageStore.GetRawArtifact(ctx, artifactID)
	if err == nil && artifactID == s.artifactID {
		artifact.Content = append([]byte(nil), artifact.Content...)
		artifact.Content = append(artifact.Content, []byte("\nTampered.\n")...)
	}
	return artifact, err
}

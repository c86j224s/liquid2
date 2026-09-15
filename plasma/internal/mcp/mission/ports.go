package mission

import (
	"context"

	canonicalmission "github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

// Reader is the narrow application port required by mission.get.
type Reader interface {
	GetProjection(context.Context, string) (canonicalmission.Projection, error)
	ListSourceSnapshotsWithState(context.Context, source.ListRequest) ([]source.Snapshot, error)
	ListEvidenceRecords(context.Context, string) ([]researchrecords.EvidenceRecord, error)
	ListClaimRecords(context.Context, string) ([]researchrecords.ClaimRecord, error)
	ListQuestionRecords(context.Context, string) ([]researchrecords.QuestionRecord, error)
}

// MetadataUpdater is the optional application port required by mission.update.
type MetadataUpdater interface {
	UpdateMissionMetadata(context.Context, canonicalmission.UpdateMissionMetadataRequest) (canonicalmission.UpdateMissionMetadataResult, error)
}

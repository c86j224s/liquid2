package reportdocument

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
)

// PlanReaders retains facade getter validation; scope construction uses raw store
// readers instead. Do not merge these paths or reorder their approval checks.
type PlanReaders struct {
	ScopeReaders
	GetSourceSnapshot func(context.Context, string) (sourcecontract.Snapshot, error)
}

// ValidatePlanRefs preserves per-kind error redaction and state-before-approval checks.
func ValidatePlanRefs(ctx context.Context, readers PlanReaders, missionID string, refs []ReportBlockSourceRefs) error {
	for _, group := range refs {
		for _, id := range group.ClaimIDs {
			record, err := readers.GetClaimRecord(ctx, id)
			if err != nil || record.MissionID != missionID || record.State == "rejected" || record.State == "superseded" || record.State == "archived" {
				return invalidReportPlanRef("claim")
			}
			if record.State != "approved" {
				if err := readers.RequireApprovedObject(ctx, missionID, record.CreatedEventID, record.ClaimID); err != nil {
					return invalidReportPlanRef("claim")
				}
			}
		}
		for _, id := range group.EvidenceIDs {
			record, err := readers.GetEvidenceRecord(ctx, id)
			if err != nil || record.MissionID != missionID || record.State == "rejected" || record.State == "superseded" || record.State == "archived" {
				return invalidReportPlanRef("evidence")
			}
			if err := requireApprovedEvidence(ctx, readers.ScopeReaders, record); err != nil {
				return invalidReportPlanRef("evidence")
			}
		}
		for _, id := range group.SnapshotIDs {
			record, err := readers.GetSourceSnapshot(ctx, id)
			if err != nil || record.MissionID != missionID || record.State.Removed || record.State.Superseded {
				return invalidReportPlanRef("source snapshot")
			}
		}
		for _, id := range group.QuestionIDs {
			record, err := readers.GetQuestionRecord(ctx, id)
			if err != nil || record.MissionID != missionID || record.State == "rejected" || record.State == "superseded" || record.State == "archived" {
				return invalidReportPlanRef("question")
			}
		}
		for _, id := range group.OptionIDs {
			record, err := readers.GetOptionRecord(ctx, id)
			if err != nil || record.MissionID != missionID || record.State == "rejected" || record.State == "superseded" || record.State == "archived" {
				return invalidReportPlanRef("option")
			}
		}
	}
	return nil
}

func invalidReportPlanRef(kind string) error {
	return fmt.Errorf("%w: report plan contains an invalid %s reference", producterror.ErrInvalidInput, kind)
}

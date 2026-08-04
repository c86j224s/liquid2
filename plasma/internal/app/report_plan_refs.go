package app

import (
	"context"
	"fmt"
)

// ValidateReportPlanRefs는 애플리케이션 서비스 계층 계약을 검사한다. 제품 상태를 변경하지 않는 순수 검증 경계다.
func (s *Service) ValidateReportPlanRefs(ctx context.Context, missionID string, refs []ReportBlockSourceRefs) error {
	for _, group := range refs {
		for _, id := range group.ClaimIDs {
			record, err := s.GetClaimRecord(ctx, id)
			if err != nil || record.MissionID != missionID || record.State == "rejected" || record.State == "superseded" || record.State == "archived" {
				return invalidReportPlanRef("claim")
			}
			if record.State != "approved" {
				if err := s.requireApprovedProposalObject(ctx, missionID, record.CreatedEventID, record.ClaimID); err != nil {
					return invalidReportPlanRef("claim")
				}
			}
		}
		for _, id := range group.EvidenceIDs {
			record, err := s.GetEvidenceRecord(ctx, id)
			if err != nil || record.MissionID != missionID || record.State == "rejected" || record.State == "superseded" || record.State == "archived" {
				return invalidReportPlanRef("evidence")
			}
			if err := s.requireReportApprovedEvidence(ctx, record); err != nil {
				return invalidReportPlanRef("evidence")
			}
		}
		for _, id := range group.SnapshotIDs {
			record, err := s.GetSourceSnapshot(ctx, id)
			if err != nil || record.MissionID != missionID || record.State.Removed || record.State.Superseded {
				return invalidReportPlanRef("source snapshot")
			}
		}
		for _, id := range group.QuestionIDs {
			record, err := s.GetQuestionRecord(ctx, id)
			if err != nil || record.MissionID != missionID || record.State == "rejected" || record.State == "superseded" || record.State == "archived" {
				return invalidReportPlanRef("question")
			}
		}
		for _, id := range group.OptionIDs {
			record, err := s.GetOptionRecord(ctx, id)
			if err != nil || record.MissionID != missionID || record.State == "rejected" || record.State == "superseded" || record.State == "archived" {
				return invalidReportPlanRef("option")
			}
		}
	}
	return nil
}

func invalidReportPlanRef(kind string) error {
	return fmt.Errorf("%w: report plan contains an invalid %s reference", ErrInvalidInput, kind)
}

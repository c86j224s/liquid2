package app

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/reporting/reportdocument"
)

// ValidateReportPlanRefs validates plan references through the document owner.
func (s *Service) ValidateReportPlanRefs(ctx context.Context, missionID string, refs []reportdocument.ReportBlockSourceRefs) error {
	return reportdocument.ValidatePlanRefs(ctx, reportdocument.PlanReaders{ScopeReaders: reportdocument.ScopeReaders{GetEvidenceRecord: s.GetEvidenceRecord, GetClaimRecord: s.GetClaimRecord, GetQuestionRecord: s.GetQuestionRecord, GetOptionRecord: s.GetOptionRecord, RequireApprovedObject: s.requireApprovedProposalObject}, GetSourceSnapshot: s.GetSourceSnapshot}, missionID, refs)
}

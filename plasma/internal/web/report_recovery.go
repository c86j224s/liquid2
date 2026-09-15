package web

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

func (server *Server) reportRecoveryHooks() reportexecution.RecoveryHooks {
	return reportexecution.RecoveryHooks{Defaults: reportexecution.RecoveryDefaults{RigorLevel: legacyPendingReportRigorLevel, ReportMode: defaultReportMode, SessionPolicy: reportSessionPolicySameSession, GuidanceProfile: reportprompt.ProfileVisualPlan}, PendingExecutor: reportDraftPendingExecutor, PendingMode: reportDraftPendingMode}
}
func (server *Server) resumeReportDraftWorker(ctx context.Context, missionID string, pending ledger.Event) error {
	return server.reportRunner().ResumeLegacyDraft(ctx, missionID, pending, server.reportRecoveryHooks())
}

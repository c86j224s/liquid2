package web

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/missionrecovery"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const missionActivityCursorSchema = "mission-activity/v1"

type missionActivityCursor struct {
	Schema   string `json:"schema"`
	Sequence int64  `json:"sequence"`
	ServerID string `json:"server_id"`
}

type missionActivityResponse struct {
	Activity app.MissionActivitySummary `json:"activity"`
	Cursor   missionActivityCursor      `json:"cursor"`
}

func (server *Server) missionActivityCursor(sequence int64) missionActivityCursor {
	return missionActivityCursor{
		Schema:   missionActivityCursorSchema,
		Sequence: sequence,
		ServerID: server.activityServerID,
	}
}

// reconcileMissionRecovery는 이전 서버 프로세스가 열어 둔 지속 작업의 idempotent
// 복구를 담당한다. 이 경로는 의도적으로 기존 full-detail GET 호환 경로에서만 호출한다.
func (server *Server) reconcileMissionRecovery(ctx context.Context, missionID string) error {
	_, err := missionrecovery.Run(ctx, missionrecovery.Plan{
		PreLock: [2]missionrecovery.Step{
			{
				Name:   missionrecovery.StepReportCompletion,
				Policy: missionrecovery.FailFast,
				Run: func(ctx context.Context) error {
					if _, pending := server.runningReports.PendingEventID(missionID); pending {
						return nil
					}
					_, err := reporting.RecoverMission(ctx, server.service, missionID)
					return err
				},
			},
			{
				Name:   missionrecovery.StepWorkflowState,
				Policy: missionrecovery.BestEffort,
				Run: func(ctx context.Context) error {
					return server.reconcileWorkflowState(ctx, missionID)
				},
			},
		},
		AcquireReportLock: func() func() {
			return server.reports.lock(missionID)
		},
		Report: [2]missionrecovery.Step{
			{
				Name:   missionrecovery.StepReportDraft,
				Policy: missionrecovery.FailFast,
				Run: func(ctx context.Context) error {
					return server.reconcileStaleReportDrafts(ctx, missionID)
				},
			},
			{
				Name:   missionrecovery.StepDesignedReportExport,
				Policy: missionrecovery.FailFast,
				Run: func(ctx context.Context) error {
					return server.reconcileStaleDesignedReportExports(ctx, missionID)
				},
			},
		},
	})
	return err
}

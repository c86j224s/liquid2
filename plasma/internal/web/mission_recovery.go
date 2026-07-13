package web

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
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

// reconcileMissionRecovery owns idempotent recovery of durable work left open
// by a previous server process. It is intentionally called only by the
// established full-detail GET compatibility path.
func (server *Server) reconcileMissionRecovery(ctx context.Context, missionID string) error {
	server.reconcileWorkflowState(ctx, missionID)
	unlock := server.reports.lock(missionID)
	defer unlock()
	if err := server.reconcileStaleReportDrafts(ctx, missionID); err != nil {
		return err
	}
	return server.reconcileStaleDesignedReportExports(ctx, missionID)
}

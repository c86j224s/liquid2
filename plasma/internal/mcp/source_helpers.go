package mcp

import (
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func newReportILLiveSourceReadRequest(missionID, snapshotID, sessionID string, maxBytes int) app.ReadLocalPathSourceRequest {
	return app.ReadLocalPathSourceRequest{
		MissionID:     missionID,
		SnapshotID:    snapshotID,
		MaxBytes:      int64(maxBytes),
		Producer:      ledger.Producer{Type: "agent_session", ID: sessionID},
		ToolSessionID: sessionID,
	}
}

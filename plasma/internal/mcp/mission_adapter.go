package mcp

import (
	"github.com/c86j224s/liquid2/plasma/internal/mcp/mission"
	sourcehandler "github.com/c86j224s/liquid2/plasma/internal/mcp/source"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

func newMissionHandler(server *Server) *mission.Handler {
	reader, _ := any(server.service).(mission.Reader)
	updater, _ := any(server.service).(mission.MetadataUpdater)
	return mission.NewHandler(reader, updater, server.enforceBoundMission, newMCPID, func(snapshot source.Snapshot) any {
		return sourcehandler.SourceSnapshotFromApp(snapshot)
	}, errorResult, errorFromErr)
}

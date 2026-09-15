package mcp

import "github.com/c86j224s/liquid2/plasma/internal/mcp/workflow"

func newWorkflowHandler(server *Server) *workflow.Handler {
	return workflow.NewHandler(server.service, workflow.Binding{
		MissionID:          server.binding.MissionID,
		AgentSessionID:     server.binding.AgentSessionID,
		CurrentUserEventID: server.binding.CurrentUserEventID,
		AgentExecutor:      server.binding.AgentExecutor,
	}, server.enforceBoundMission, errorResult, errorFromErr)
}

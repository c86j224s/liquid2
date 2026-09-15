package mcp

import (
	"context"
	sourcehandler "github.com/c86j224s/liquid2/plasma/internal/mcp/source"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"github.com/c86j224s/liquid2/plasma/internal/source/liquid2source"
)

func (server *Server) sourceHandler() sourcehandler.Handler {
	return sourcehandler.Handler{ApprovalRequired: approvalRequiredResult, Service: server.service, MissionID: func() string { return server.binding.MissionID }, EnforceMission: server.enforceBoundMission, RequireSession: server.requireBoundWriteSession, ObservationProducer: server.boundObservationProducer, Liquid2Connector: func() (liquid2source.Liquid2SourceConnector, bool) {
		c, ok := server.connectors[liquid2source.Liquid2ConnectorID]
		return c, ok
	}, ConfluenceFactory: func() sourcehandler.Factory {
		if server.confluenceConnectorFactory == nil {
			return nil
		}
		return func(ctx context.Context, req sourcehandler.ConfluenceConnectorRequest) (confluencesource.ConfluenceSourceConnector, error) {
			return server.confluenceConnectorFactory(ctx, ConfluenceConnectorRequest(req))
		}
	}, Decode: decodeArgs, NormalizeInput: normalizeMutatingInput, ErrorResult: errorResult, ErrorFromErr: errorFromErr, NewID: newMCPID, ValidateID: validateID, ArtifactOutput: rawArtifactFromApp}
}
func (server *Server) callSourcesList(ctx context.Context, call ToolCall) ToolResult {
	h := server.sourceHandler()
	return h.CallSourcesList(ctx, call)
}
func (server *Server) callSourcesRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.sourceHandler()
	return h.CallSourcesRead(ctx, call)
}
func (server *Server) callSourcesTree(ctx context.Context, call ToolCall) ToolResult {
	h := server.sourceHandler()
	return h.CallSourcesTree(ctx, call)
}
func (server *Server) callSourcesGrep(ctx context.Context, call ToolCall) ToolResult {
	h := server.sourceHandler()
	return h.CallSourcesGrep(ctx, call)
}
func (server *Server) callLocalPathRoots(ctx context.Context, call ToolCall) ToolResult {
	h := server.sourceHandler()
	return h.CallLocalPathRoots(ctx, call)
}
func (server *Server) callLocalPathTree(ctx context.Context, call ToolCall) ToolResult {
	h := server.sourceHandler()
	return h.CallLocalPathTree(ctx, call)
}
func (server *Server) callLocalPathAttach(ctx context.Context, call ToolCall) ToolResult {
	h := server.sourceHandler()
	return h.CallLocalPathAttach(ctx, call)
}
func (server *Server) callSourcesRemove(ctx context.Context, call ToolCall) ToolResult {
	h := server.sourceHandler()
	return h.CallSourcesRemove(ctx, call)
}
func (server *Server) callSourcesRestore(ctx context.Context, call ToolCall) ToolResult {
	h := server.sourceHandler()
	return h.CallSourcesRestore(ctx, call)
}
func (server *Server) callSourcesSearch(ctx context.Context, call ToolCall) ToolResult {
	h := server.sourceHandler()
	return h.CallSourcesSearch(ctx, call)
}
func (server *Server) callSourcesSnapshot(ctx context.Context, call ToolCall) ToolResult {
	h := server.sourceHandler()
	return h.CallSourcesSnapshot(ctx, call)
}

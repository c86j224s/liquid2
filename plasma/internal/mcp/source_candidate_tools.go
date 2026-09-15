package mcp

import (
	"context"
	sourcehandler "github.com/c86j224s/liquid2/plasma/internal/mcp/source"
	candidatehandler "github.com/c86j224s/liquid2/plasma/internal/mcp/sourcecandidate"
	"github.com/c86j224s/liquid2/plasma/internal/sourceretrieval"
)

func (server *Server) candidateHandler() candidatehandler.Handler {
	return candidatehandler.Handler{Service: server.service, CurrentUserEventID: func() string { return server.binding.CurrentUserEventID }, AgentExecutor: func() string { return server.binding.AgentExecutor }, RequireSession: server.requireBoundWriteSession, EnforceMission: server.enforceBoundMission, Fetcher: func() func(context.Context, string) (sourceretrieval.Fetched, error) {
		return server.sourceCandidateFetcher
	}, Decode: decodeArgs, NormalizeInput: normalizeMutatingInput, ErrorResult: errorResult, ErrorFromErr: errorFromErr, NewID: newMCPID, ValidateID: validateID, ArtifactOutput: rawArtifactFromApp, BoundedRead: sourcehandler.BoundedArtifactContent}
}
func (server *Server) callSourceCandidatesPropose(ctx context.Context, call ToolCall) ToolResult {
	h := server.candidateHandler()
	return h.Propose(ctx, call)
}
func (server *Server) callSourceCandidatesRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.candidateHandler()
	return h.Read(ctx, call)
}

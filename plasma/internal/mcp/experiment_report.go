package mcp

import (
	"context"
	experimenthandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportexperiment"
)

func (server *Server) experimentHandler() experimenthandler.Handler {
	return experimenthandler.Handler{State: server.reportExperimentState, Mu: &server.mu, Service: server.service, MissionID: func() string { return server.binding.MissionID }, SessionID: func() string { return server.binding.AgentSessionID }, Decode: decodeArgs, NormalizeInput: normalizeMutatingInput, ErrorResult: errorResult, ErrorFromErr: errorFromErr, NewID: newMCPID, ArtifactOutput: rawArtifactFromApp}
}
func (server *Server) callExperimentReportCreate(ctx context.Context, call ToolCall) ToolResult {
	h := server.experimentHandler()
	return h.Create(ctx, call)
}
func (server *Server) callExperimentReportAppend(ctx context.Context, call ToolCall) ToolResult {
	h := server.experimentHandler()
	return h.Append(ctx, call)
}
func (server *Server) callExperimentReportRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.experimentHandler()
	return h.Read(ctx, call)
}
func (server *Server) callExperimentReportFinalize(ctx context.Context, call ToolCall) ToolResult {
	h := server.experimentHandler()
	return h.Finalize(ctx, call)
}

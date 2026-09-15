package mcp

import (
	"context"
	patchhandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportpatch"
)

func (server *Server) patchHandler() patchhandler.Handler {
	return patchhandler.Handler{State: server.reportPatchState, Mu: &server.mu, Service: server.service, Binding: func() patchhandler.ReportPatchBinding {
		return patchhandler.ReportPatchBinding(server.reportPatchBinding)
	}, MissionID: func() string { return server.binding.MissionID }, SessionID: func() string { return server.binding.AgentSessionID }, Decode: decodeArgs, NormalizeInput: normalizeMutatingInput, ErrorResult: errorResult, ErrorFromErr: errorFromErr, NewID: newMCPID, ArtifactOutput: rawArtifactFromApp}
}
func (server *Server) callReportPatchStart(ctx context.Context, call ToolCall) ToolResult {
	h := server.patchHandler()
	return h.Start(ctx, call)
}
func (server *Server) callReportPatchRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.patchHandler()
	return h.Read(ctx, call)
}
func (server *Server) callReportPatchApply(ctx context.Context, call ToolCall) ToolResult {
	h := server.patchHandler()
	return h.Apply(ctx, call)
}
func (server *Server) callReportPatchFinalize(ctx context.Context, call ToolCall) ToolResult {
	h := server.patchHandler()
	return h.Finalize(ctx, call)
}

type reportPatchOperation = patchhandler.Operation

func normalizeReportPatchBinding(b ReportPatchBinding) ReportPatchBinding {
	return ReportPatchBinding(patchhandler.NormalizeBinding(patchhandler.ReportPatchBinding(b)))
}

const (
	reportPatchMaxBytes        = patchhandler.ReportPatchMaxBytes
	reportPatchMaxApplyBytes   = patchhandler.ReportPatchMaxApplyBytes
	reportPatchMaxOperations   = patchhandler.ReportPatchMaxOperations
	reportPatchDefaultReadSize = patchhandler.ReportPatchDefaultReadSize
	reportPatchMaxReadSize     = patchhandler.ReportPatchMaxReadSize
	reportPatchMaxDrafts       = patchhandler.ReportPatchMaxDrafts
)

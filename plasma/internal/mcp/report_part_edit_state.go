package mcp

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportparts"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) partsHandler() reportparts.Handler {
	return reportparts.Handler{AssemblyBinding: server.requirePartAssemblyBinding, CurrentAssemblyBinding: func() reporting.PartAssemblyBinding { return server.partAssemblyBinding }, ValidateAssemblyBinding: func(b reporting.PartAssemblyBinding) error { return ValidatePartAssemblyBinding(server.binding, b) }, RequireSession: server.requireBoundWriteSession, State: server.reportPartsState, Mu: &server.mu, Service: server.service, EditBinding: server.requirePartEditBinding, Decode: decodeReportPlanJSON, NormalizeInput: normalizeMutatingInput, ErrorResult: errorResult, ErrorFromErr: errorFromErr, ValidateID: validateID, NewID: newMCPID}
}
func (server *Server) requirePartEditBinding(common commonMutatingInput) (reporting.PartEditBinding, error) {
	if err := server.requireBoundWriteSession(common); err != nil {
		return reporting.PartEditBinding{}, err
	}
	if err := ValidatePartEditBinding(server.binding, server.partEditBinding); err != nil {
		return reporting.PartEditBinding{}, err
	}
	return server.partEditBinding, nil
}

func (server *Server) partEditToolEnabled(name string) bool {
	return server.toolEnabled(name) && ValidatePartEditBinding(server.binding, server.partEditBinding) == nil
}

func partEditDisabledResult(call ToolCall) ToolResult {
	return errorResult(call.Name, missionIDFromArguments(call.Arguments), "binding", "Part editor tools are only enabled for one bound assembled Part", false, nil)
}

func (server *Server) callReportPartEditStart(ctx context.Context, call ToolCall) ToolResult {
	h := server.partsHandler()
	return h.CallReportPartEditStart(ctx, call)
}
func (server *Server) callReportPartEditRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.partsHandler()
	return h.CallReportPartEditRead(ctx, call)
}
func (server *Server) callReportPartEditPatch(ctx context.Context, call ToolCall) ToolResult {
	h := server.partsHandler()
	return h.CallReportPartEditPatch(ctx, call)
}
func (server *Server) callReportPartEditSubmit(ctx context.Context, call ToolCall) ToolResult {
	h := server.partsHandler()
	return h.CallReportPartEditSubmit(ctx, call)
}

type partEditDraft = reportparts.PartEditDraft
type reportPartEditStartInput = reportparts.ReportPartEditStartInput
type reportPartEditReadInput = reportparts.ReportPartEditReadInput
type reportPartEditPatchInput = reportparts.ReportPartEditPatchInput
type reportPartEditSubmitInput = reportparts.ReportPartEditSubmitInput

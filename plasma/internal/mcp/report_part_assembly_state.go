package mcp

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportparts"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"time"
)

func (server *Server) requirePartAssemblyBinding(common commonMutatingInput) (reporting.PartAssemblyBinding, error) {
	if err := server.requireBoundWriteSession(common); err != nil {
		return reporting.PartAssemblyBinding{}, err
	}
	binding := server.partAssemblyBinding
	if err := ValidatePartAssemblyBinding(server.binding, binding); err != nil {
		return reporting.PartAssemblyBinding{}, err
	}
	return binding, nil
}

func (server *Server) anyPartAssemblyToolEnabled() bool {
	return server.toolEnabled(ToolReportPartAssemblyStart) ||
		server.toolEnabled(ToolReportPartAssemblyRead) ||
		server.toolEnabled(ToolReportPartSectionRead) ||
		server.toolEnabled(ToolReportPartAssemblyPatch) ||
		server.toolEnabled(ToolReportPartAssemblySubmit)
}

func (server *Server) partAssemblySectionReadToolEnabled() bool {
	return server.toolEnabled(ToolReportPartSectionRead) && reporting.ValidatePartAssemblySectionReadBinding(server.partAssemblyBinding) == nil && ValidatePartAssemblyBinding(server.binding, server.partAssemblyBinding) == nil
}

func (server *Server) partAssemblyToolEnabled(name string) bool {
	return server.toolEnabled(name) && ValidatePartAssemblyBinding(server.binding, server.partAssemblyBinding) == nil
}

func partAssemblyDisabledResult(call ToolCall) ToolResult {
	return errorResult(
		call.Name,
		missionIDFromArguments(call.Arguments),
		"binding",
		"part assembly tools are only enabled for bound long-form part assembly sessions",
		false,
		nil,
	)
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
func (server *Server) callReportPartAssemblyStart(ctx context.Context, call ToolCall) ToolResult {
	h := server.partsHandler()
	return h.CallReportPartAssemblyStart(ctx, call)
}
func (server *Server) callReportPartAssemblyRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.partsHandler()
	return h.CallReportPartAssemblyRead(ctx, call)
}
func (server *Server) callReportPartAssemblyPatch(ctx context.Context, call ToolCall) ToolResult {
	h := server.partsHandler()
	return h.CallReportPartAssemblyPatch(ctx, call)
}
func (server *Server) callReportPartAssemblySubmit(ctx context.Context, call ToolCall) ToolResult {
	h := server.partsHandler()
	return h.CallReportPartAssemblySubmit(ctx, call)
}
func (server *Server) callReportPartSectionRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.partsHandler()
	return h.CallReportPartSectionRead(ctx, call)
}

type partAssemblyDraft = reportparts.PartAssemblyDraft
type partAssemblyOperation = reportparts.PartAssemblyOperation
type reportPartAssemblyStartInput = reportparts.ReportPartAssemblyStartInput
type reportPartAssemblyReadInput = reportparts.ReportPartAssemblyReadInput
type reportPartSectionReadInput = reportparts.ReportPartSectionReadInput
type reportPartAssemblyPatchInput = reportparts.ReportPartAssemblyPatchInput
type reportPartAssemblySubmitInput = reportparts.ReportPartAssemblySubmitInput

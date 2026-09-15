package mcp

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportil"
)

func (server *Server) callReportILLongFormPlanSubmit(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILLongFormPlanSubmit(ctx, call)
}
func (server *Server) callReportILLongFormPlanRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILLongFormPlanRead(ctx, call)
}

type reportILLongFormPlanSubmitInput = reportil.ReportILLongFormPlanSubmitInput
type reportILLongFormPlanPartInput = reportil.ReportILLongFormPlanPartInput
type reportILLongFormPlanSectionInput = reportil.ReportILLongFormPlanSectionInput
type reportILLongFormPlanReadInput = reportil.ReportILLongFormPlanReadInput
type reportILLongFormPlanStateOutput = reportil.ReportILLongFormPlanStateOutput
type reportILLongFormPlanReadOutput = reportil.ReportILLongFormPlanReadOutput

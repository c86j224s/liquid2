package mcp

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportil"
)

func (server *Server) callReportILDocumentStart(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILDocumentStart(ctx, call)
}
func (server *Server) callReportILDocumentOpen(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILDocumentOpen(ctx, call)
}
func (server *Server) callReportILDocumentAppend(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILDocumentAppend(ctx, call)
}
func (server *Server) callReportILDocumentAppendSource(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILDocumentAppendSource(ctx, call)
}
func (server *Server) callReportILDocumentRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILDocumentRead(ctx, call)
}
func (server *Server) callReportILDocumentReplace(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILDocumentReplace(ctx, call)
}
func (server *Server) callReportILDocumentEditText(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILDocumentEditText(ctx, call)
}
func (server *Server) callReportILDocumentReviseBlock(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILDocumentReviseBlock(ctx, call)
}
func (server *Server) callReportILDocumentFinalize(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILDocumentFinalize(ctx, call)
}

type reportILDocumentWorkspace = reportil.DocumentWorkspace

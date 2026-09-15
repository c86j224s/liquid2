package mcp

import (
	"context"
)

func (server *Server) callReportILLongFormDocumentStart(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILLongFormDocumentStart(ctx, call)
}
func (server *Server) callReportILLongFormDocumentAppend(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILLongFormDocumentAppend(ctx, call)
}
func (server *Server) callReportILLongFormDocumentRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILLongFormDocumentRead(ctx, call)
}
func (server *Server) callReportILLongFormDocumentReplace(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILLongFormDocumentReplace(ctx, call)
}
func (server *Server) callReportILLongFormDocumentCorrectBlock(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILLongFormDocumentCorrectBlock(ctx, call)
}
func (server *Server) callReportILLongFormDocumentFinalize(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILLongFormDocumentFinalize(ctx, call)
}

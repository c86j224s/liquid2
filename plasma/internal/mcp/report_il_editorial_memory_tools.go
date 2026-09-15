package mcp

import (
	"context"
)

func (server *Server) callReportILEditorialMemoryStart(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILEditorialMemoryStart(ctx, call)
}
func (server *Server) callReportILEditorialMemoryAppend(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILEditorialMemoryAppend(ctx, call)
}
func (server *Server) callReportILEditorialMemoryRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILEditorialMemoryRead(ctx, call)
}
func (server *Server) callReportILEditorialMemoryFinalize(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILEditorialMemoryFinalize(ctx, call)
}

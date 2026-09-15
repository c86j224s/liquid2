package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportplan"
	"io"
)

func (server *Server) planHandler() *reportplan.Handler {
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.reportPlanHandler == nil {
		server.reportPlanHandler = &reportplan.Handler{Capability: func() (reportplan.Service, bool) { svc, ok := server.service.(reportplan.Service); return svc, ok }, Binding: func() reportplan.Binding { return reportplan.Binding(server.reportPlanBinding) }, MissionID: func() string { return server.binding.MissionID }, Available: func() bool { return server.reportPlanBinding.complete() && server.toolEnabled(ToolReportPlanSubmit) }, ArgumentsHash: canonicalArgumentsHash, NewID: newMCPID, ErrorResult: errorResult}
	}
	return server.reportPlanHandler
}
func (server *Server) callReportPlanSubmit(ctx context.Context, call ToolCall) ToolResult {
	return server.planHandler().Submit(ctx, call)
}
func (server *Server) reportPlanAttemptCount() int { return server.planHandler().AttemptCount() }
func decodeReportPlanJSON(payload json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

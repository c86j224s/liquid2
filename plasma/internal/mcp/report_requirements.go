package mcp

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportrequirements"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) callReportRequirementsSubmit(ctx context.Context, call ToolCall) ToolResult {
	server.mu.Lock()
	if server.reportRequirementsHandler == nil {
		server.reportRequirementsHandler = &reportrequirements.Handler{ListEvents: func(ctx context.Context, missionID string) ([]ledger.Event, error) {
			return server.service.ListEvents(ctx, missionID)
		}, Capability: func() (reportrequirements.Service, bool) {
			svc, ok := server.service.(reportrequirements.Service)
			return svc, ok
		}, Binding: func() reporting.ReportRequirementMapBinding { return server.reportRequirementMapBinding }, MissionID: func() string { return server.binding.MissionID }, Available: server.reportRequirementToolAvailable, Decode: decodeReportPlanJSON, ArgumentsHash: canonicalArgumentsHash, NewID: newMCPID, ErrorResult: errorResult}
	}
	handler := server.reportRequirementsHandler
	server.mu.Unlock()
	return handler.Submit(ctx, call)
}
func (server *Server) reportRequirementToolAvailable() bool {
	if !server.toolEnabled(ToolReportRequirementsSubmit) {
		return false
	}
	return ValidateReportRequirementMapBinding(server.binding, server.reportRequirementMapBinding) == nil
}

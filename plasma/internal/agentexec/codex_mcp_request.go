package agentexec

import (
	"context"
	"encoding/json"
	"strings"
)

func codexMCPArgsForRequest(base []string, req AgentRequest) []string {
	args := append([]string(nil), base...)
	if req.ReplaceMCPTools {
		args = stripMCPEnabledToolArgs(args)
	}
	if hasReportPatchTool(req.ExtraMCPTools) || req.ReportPatch != nil {
		args = append(args, "-report-patch")
	}
	if req.ReportPlan != nil {
		args = appendReportPlanMCPArgs(args, req.ToolSessionID, *req.ReportPlan)
	}
	if req.ReportRequirements != nil {
		if encoded, err := json.Marshal(req.ReportRequirements); err == nil {
			args = append(args, "-report-requirements-binding-json", string(encoded))
		}
	}
	if req.PartAssembly != nil {
		if encoded, err := json.Marshal(req.PartAssembly); err == nil {
			args = append(args, "-report-part-assembly-binding-json", string(encoded))
		}
	}
	if req.PartEdit != nil {
		if encoded, err := json.Marshal(req.PartEdit); err == nil {
			args = append(args, "-report-part-edit-binding-json", string(encoded))
		}
	}
	if req.LongFormFinalize != nil {
		if encoded, err := json.Marshal(req.LongFormFinalize); err == nil {
			args = append(args, "-report-long-form-finalize-binding-json", string(encoded))
		}
	}
	if req.FinalEditStage != nil {
		if encoded, err := json.Marshal(req.FinalEditStage); err == nil {
			args = append(args, "-report-final-edit-stage-binding-json", string(encoded))
		}
	}
	if req.ReportPatch != nil {
		args = appendReportPatchMCPArgs(args, *req.ReportPatch)
	}
	for _, tool := range req.ExtraMCPTools {
		if tool = strings.TrimSpace(tool); tool != "" {
			args = append(args, "-enabled-tool", tool)
		}
	}
	if missionID := strings.TrimSpace(req.MissionID); missionID != "" {
		args = append(args, "-mission-id", missionID)
	}
	if toolSessionID := strings.TrimSpace(req.ToolSessionID); toolSessionID != "" {
		args = append(args, "-agent-session-id", toolSessionID)
	}
	if userEventID := strings.TrimSpace(req.UserEventID); userEventID != "" {
		args = append(args, "-current-user-event-id", userEventID)
	}
	if agentExecutor := strings.TrimSpace(strings.ToLower(req.AgentExecutor)); agentExecutor != "" {
		args = append(args, "-agent-executor", agentExecutor)
	}
	return args
}

// CodexMCPArgsForRequest returns the bound MCP argv for compatibility tests and transport wiring checks.
func CodexMCPArgsForRequest(base []string, req AgentRequest) []string {
	return codexMCPArgsForRequest(base, req)
}

func stripMCPEnabledToolArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		if args[index] == "-enabled-tool" {
			index++
			continue
		}
		out = append(out, args[index])
	}
	return out
}

func appendReportPatchMCPArgs(args []string, patch AgentReportPatchContext) []string {
	values := []struct {
		flag  string
		value string
	}{
		{"-report-patch-base-artifact-id", patch.BaseArtifactID},
		{"-report-patch-pending-event-id", patch.PendingEventID},
		{"-report-patch-agent-executor", patch.AgentExecutor},
		{"-report-patch-agent-model", patch.AgentModel},
		{"-report-patch-agent-reasoning-effort", patch.AgentReasoningEffort},
		{"-report-patch-mcp-mode", patch.MCPMode},
		{"-report-patch-agent-session-id", patch.AgentSessionID},
		{"-report-patch-previous-agent-session-id", patch.PreviousAgentSessionID},
		{"-report-patch-returned-agent-session-id", patch.ReturnedAgentSessionID},
		{"-report-patch-report-session-id", patch.ReportSessionID},
		{"-report-patch-fork-source-agent-session-id", patch.ForkSourceAgentSessionID},
		{"-report-patch-report-session-policy", patch.ReportSessionPolicy},
		{"-report-patch-report-session-policy-selection", patch.ReportSessionPolicySelection},
		{"-report-patch-session-chain-kind", patch.SessionChainKind},
	}
	for _, item := range values {
		if value := strings.TrimSpace(item.value); value != "" {
			args = append(args, item.flag, value)
		}
	}
	return args
}

func appendReportPlanMCPArgs(args []string, toolSessionID string, plan AgentReportPlanContext) []string {
	values := []struct{ flag, value string }{
		{"-report-plan-pending-event-id", plan.PendingEventID}, {"-report-plan-mode", plan.ReportMode},
		{"-report-plan-idempotency-key", plan.IdempotencyKey}, {"-report-plan-tool-session-id", toolSessionID},
		{"-report-plan-previous-provider-session-id", plan.PreviousProviderSessionID},
		{"-report-plan-agent-model", plan.AgentModel}, {"-report-plan-agent-reasoning-effort", plan.AgentReasoningEffort},
	}
	for _, item := range values {
		if value := strings.TrimSpace(item.value); value != "" {
			args = append(args, item.flag, value)
		}
	}
	if plan.RequireWritingContract {
		args = append(args, "-report-plan-require-writing-contract")
	}
	return args
}

// AppendReportPlanMCPArgs appends report-plan binding argv for compatibility tests.
func AppendReportPlanMCPArgs(args []string, toolSessionID string, plan AgentReportPlanContext) []string {
	return appendReportPlanMCPArgs(args, toolSessionID, plan)
}

// AgentSessionForkReady reports whether an executor can read and prepare the source session for fork.
func AgentSessionForkReady(ctx context.Context, executor AgentExecutor, sourceSessionID string) bool {
	if strings.TrimSpace(sourceSessionID) == "" {
		return false
	}
	readiness, ok := executor.(AgentSessionForkReadiness)
	return ok && readiness.CheckForkSession(ctx, sourceSessionID) == nil
}

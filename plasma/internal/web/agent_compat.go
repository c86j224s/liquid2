package web

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
)

type AgentExecutor = agentexec.AgentExecutor
type StreamingAgentExecutor = agentexec.StreamingAgentExecutor
type AgentRequest = agentexec.AgentRequest
type AgentResult = agentexec.AgentResult
type AgentObserver = agentexec.AgentObserver
type AgentObservation = agentexec.AgentObservation
type AgentObservationType = agentexec.AgentObservationType
type AgentPhase = agentexec.AgentPhase
type AgentToolCategory = agentexec.AgentToolCategory
type AgentReportPlanContext = agentexec.AgentReportPlanContext
type AgentReportPatchContext = agentexec.AgentReportPatchContext
type AgentSessionForkResult = agentexec.AgentSessionForkResult
type AgentSessionForker = agentexec.AgentSessionForker
type AgentSessionForkReadiness = agentexec.AgentSessionForkReadiness
type CodexExecutor = agentexec.CodexExecutor
type CodexMCPServer = agentexec.CodexMCPServer
type ClaudeExecutor = agentexec.ClaudeExecutor
type ClaudeMCPServer = agentexec.ClaudeMCPServer

const (
	AgentObservationPhase          = agentexec.AgentObservationPhase
	AgentObservationTool           = agentexec.AgentObservationTool
	AgentObservationAnswer         = agentexec.AgentObservationAnswer
	AgentPhaseThinking             = agentexec.AgentPhaseThinking
	AgentToolCategoryWebSearch     = agentexec.AgentToolCategoryWebSearch
	AgentToolCategoryWebRead       = agentexec.AgentToolCategoryWebRead
	AgentToolCategoryMissionRead   = agentexec.AgentToolCategoryMissionRead
	AgentToolCategorySourcePropose = agentexec.AgentToolCategorySourcePropose
	AgentToolCategoryOrganize      = agentexec.AgentToolCategoryOrganize
	AgentToolCategoryValidate      = agentexec.AgentToolCategoryValidate
	AgentToolCategoryUnknown       = agentexec.AgentToolCategoryUnknown
)

func codexMCPArgsForRequest(base []string, req AgentRequest) []string {
	return agentexec.CodexMCPArgsForRequest(base, req)
}

func appendReportPlanMCPArgs(args []string, toolSessionID string, plan AgentReportPlanContext) []string {
	return agentexec.AppendReportPlanMCPArgs(args, toolSessionID, plan)
}

func codexMCPConfigArgs(server CodexMCPServer, req AgentRequest) []string {
	return agentexec.CodexMCPConfigArgs(server, req)
}

func codexEnvironment(explicit []string) []string {
	return agentexec.CodexEnvironment(explicit)
}

func agentPATH(current string) string {
	return agentexec.AgentPATH(current)
}

func resolveAgentCommand(command string) string {
	return agentexec.ResolveAgentCommand(command)
}

// AgentSessionForkReady preserves the Web compatibility surface while delegating provider readiness to agentexec.
func AgentSessionForkReady(ctx context.Context, executor AgentExecutor, sourceSessionID string) bool {
	return agentexec.AgentSessionForkReady(ctx, executor, sourceSessionID)
}

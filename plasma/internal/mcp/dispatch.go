package mcp

import (
	"context"
	"strings"
	"time"
)

func (server *Server) Call(ctx context.Context, call ToolCall) ToolResult {
	if server.service == nil {
		return errorResult(call.Name, "", "internal", "service is required", false, nil)
	}
	started := time.Now().UTC()
	result := server.dispatchCall(ctx, call)
	if eventID, err := server.recordToolCall(ctx, call, result, started); err != nil {
		result.TraceError = err.Error()
	} else if eventID != "" {
		result.TraceEventID = eventID
	}
	return result
}

func (server *Server) dispatchCall(ctx context.Context, call ToolCall) ToolResult {
	if len(server.enabledTools) > 0 && !server.toolEnabled(call.Name) {
		return errorResult(call.Name, server.binding.MissionID, "validation", "tool is not enabled for this MCP server", false, nil)
	}
	switch call.Name {
	case ToolMissionGet:
		return server.callMissionGet(ctx, call)
	case ToolMissionUpdate:
		return server.withUserMutationIdempotency(ctx, call, server.callMissionUpdate)
	case ToolSourcesList:
		return server.callSourcesList(ctx, call)
	case ToolSourcesRead:
		return server.callSourcesRead(ctx, call)
	case ToolSourcesTree:
		return server.callSourcesTree(ctx, call)
	case ToolSourcesGrep:
		return server.callSourcesGrep(ctx, call)
	case ToolSourcesSearch:
		return server.callSourcesSearch(ctx, call)
	case ToolSourceCandidatesPropose:
		return server.withIdempotency(ctx, call, server.callSourceCandidatesPropose)
	case ToolSourceCandidatesRead:
		return server.callSourceCandidatesRead(ctx, call)
	case ToolLocalPathRoots:
		return server.callLocalPathRoots(ctx, call)
	case ToolLocalPathTree:
		return server.callLocalPathTree(ctx, call)
	case ToolLocalPathAttach:
		if !server.operatorSourceMutation {
			return sourceMutationDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callLocalPathAttach)
	case ToolSourcesRemove:
		if !server.operatorSourceMutation {
			return sourceMutationDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callSourcesRemove)
	case ToolSourcesRestore:
		if !server.operatorSourceMutation {
			return sourceMutationDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callSourcesRestore)
	case ToolResearchOutline:
		return server.callResearchOutline(ctx, call)
	case ToolResearchList:
		return server.callResearchList(ctx, call)
	case ToolResearchRead:
		return server.callResearchRead(ctx, call)
	case ToolResearchGrep:
		return server.callResearchGrep(ctx, call)
	case ToolResearchRefs:
		return server.callResearchReferences(ctx, call)
	case ToolMermaidValidate:
		return server.callMermaidValidate(ctx, call)
	case ToolWorkflowStart:
		return server.callWorkflowStart(ctx, call)
	case ToolWorkflowStatus:
		return server.callWorkflowStatus(ctx, call)
	case ToolWorkflowStop:
		return server.callWorkflowStop(ctx, call)
	case ToolReportPatchStart:
		if !server.reportPatch {
			return reportPatchDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callReportPatchStart)
	case ToolReportPatchRead:
		if !server.reportPatch {
			return reportPatchDisabledResult(call)
		}
		return server.callReportPatchRead(ctx, call)
	case ToolReportPatchApply:
		if !server.reportPatch {
			return reportPatchDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callReportPatchApply)
	case ToolReportPatchFinalize:
		if !server.reportPatch {
			return reportPatchDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callReportPatchFinalize)
	case ToolReportPlanSubmit:
		return server.callReportPlanSubmit(ctx, call)
	case ToolReportPartAssemblyStart:
		if !server.partAssemblyToolEnabled(call.Name) {
			return partAssemblyDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callReportPartAssemblyStart)
	case ToolReportPartAssemblyRead:
		if !server.partAssemblyToolEnabled(call.Name) {
			return partAssemblyDisabledResult(call)
		}
		return server.callReportPartAssemblyRead(ctx, call)
	case ToolReportPartAssemblyPatch:
		if !server.partAssemblyToolEnabled(call.Name) {
			return partAssemblyDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callReportPartAssemblyPatch)
	case ToolReportPartAssemblySubmit:
		if !server.partAssemblyToolEnabled(call.Name) {
			return partAssemblyDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callReportPartAssemblySubmit)
	case ToolReportLongFormFinalize:
		return server.callReportLongFormFinalize(ctx, call)
	case ToolExperimentReportCreate:
		if !server.experimentalReportComposition {
			return experimentReportDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callExperimentReportCreate)
	case ToolExperimentReportAppend:
		if !server.experimentalReportComposition {
			return experimentReportDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callExperimentReportAppend)
	case ToolExperimentReportRead:
		if !server.experimentalReportComposition {
			return experimentReportDisabledResult(call)
		}
		return server.callExperimentReportRead(ctx, call)
	case ToolExperimentReportFinalize:
		if !server.experimentalReportComposition {
			return experimentReportDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callExperimentReportFinalize)
	case ToolSourcesSnapshot:
		return server.withIdempotency(ctx, call, server.callSourcesSnapshot)
	case ToolEvidencePropose:
		if !server.legacyResearchLoop {
			return legacyMutationDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callEvidencePropose)
	case ToolQuestionsPropose:
		if !server.legacyResearchLoop {
			return legacyMutationDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callQuestionsPropose)
	case ToolClaimsPropose:
		if !server.legacyResearchLoop {
			return legacyMutationDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callClaimsPropose)
	case ToolClaimConfidence:
		if !server.legacyResearchLoop {
			return legacyMutationDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callClaimConfidence)
	case ToolProposalsSubmit:
		if !server.legacyResearchLoop {
			return legacyMutationDisabledResult(call)
		}
		return server.withIdempotency(ctx, call, server.callProposalsSubmit)
	default:
		return errorResult(call.Name, "", "validation", "unknown tool", false, nil)
	}
}

func (server *Server) withUserMutationIdempotency(ctx context.Context, call ToolCall, fn func(context.Context, ToolCall) ToolResult) ToolResult {
	key, missionID, err := idempotencyKey(call)
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	var input missionUpdateInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if bound := strings.TrimSpace(server.binding.AgentSessionID); bound != "" && strings.TrimSpace(input.SessionID) != bound {
		return errorResult(call.Name, missionID, "validation", "tool call session_id is outside this MCP session", false, nil)
	}
	if strings.TrimSpace(input.Producer.Type) != "user" || strings.TrimSpace(input.Producer.ID) == "" {
		return errorResult(call.Name, missionID, "validation", "mission metadata updates require a user producer", false, nil)
	}
	hash, err := canonicalArgumentsHash(call.Arguments)
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	server.mu.Lock()
	if cached, ok := server.idempotency[key]; ok {
		server.mu.Unlock()
		if cached.ArgumentsHash != hash {
			return errorResult(call.Name, missionID, "conflict", "idempotency_key reused with different arguments", false, nil)
		}
		return cached.Result
	}
	server.mu.Unlock()
	result := fn(ctx, call)
	if result.Error == nil {
		server.mu.Lock()
		server.idempotency[key] = idempotencyEntry{ArgumentsHash: hash, Result: result}
		server.mu.Unlock()
	}
	return result
}

func reportPatchDisabledResult(call ToolCall) ToolResult {
	return errorResult(
		call.Name,
		missionIDFromArguments(call.Arguments),
		"validation",
		"report patch tools are only enabled for report patch sessions",
		false,
		nil,
	)
}

func (server *Server) toolEnabled(name string) bool {
	_, ok := server.enabledTools[strings.TrimSpace(name)]
	return ok
}

func experimentReportDisabledResult(call ToolCall) ToolResult {
	return errorResult(
		call.Name,
		missionIDFromArguments(call.Arguments),
		"validation",
		"experimental report composition tool is disabled",
		false,
		nil,
	)
}

func sourceMutationDisabledResult(call ToolCall) ToolResult {
	return errorResult(
		call.Name,
		missionIDFromArguments(call.Arguments),
		"validation",
		"source mutation tools require an explicit operator surface; agents may inspect source candidates but cannot promote or remove accepted sources by default",
		false,
		nil,
	)
}

func legacyMutationDisabledResult(call ToolCall) ToolResult {
	return errorResult(
		call.Name,
		missionIDFromArguments(call.Arguments),
		"validation",
		"legacy research mutation tool is disabled in the default C1 loop",
		false,
		nil,
	)
}

func (server *Server) withIdempotency(
	ctx context.Context,
	call ToolCall,
	fn func(context.Context, ToolCall) ToolResult,
) ToolResult {
	key, missionID, err := idempotencyKey(call)
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMutation(call.Arguments); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	argumentsHash, err := canonicalArgumentsHash(call.Arguments)
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	server.mu.Lock()
	if cached, ok := server.idempotency[key]; ok {
		server.mu.Unlock()
		if cached.ArgumentsHash != argumentsHash {
			return errorResult(call.Name, missionID, "conflict", "idempotency_key reused with different arguments", false, nil)
		}
		return cached.Result
	}
	server.mu.Unlock()

	result := fn(ctx, call)
	if result.Error == nil {
		server.mu.Lock()
		server.idempotency[key] = idempotencyEntry{ArgumentsHash: argumentsHash, Result: result}
		server.mu.Unlock()
	}
	return result
}

package mcp

import (
	"context"
	"strings"
	"time"
)

func (server *Server) callReportPartAssemblyStart(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	var input reportPartAssemblyStartInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "part assembly start arguments are invalid", false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.requirePartAssemblyBinding(common)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID || input.PartIndex != binding.PartIndex || input.SectionCount != binding.SectionCount {
		return errorResult(call.Name, common.MissionID, "binding", "part assembly start does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if draftID == "" {
		draftID = newMCPID("rpa")
	}
	if err := validateID("rpa_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	now := time.Now().UTC()
	draft := &partAssemblyDraft{
		DraftID:      draftID,
		MissionID:    common.MissionID,
		SessionID:    common.SessionID,
		PendingID:    binding.PendingEventID,
		PlanEventID:  binding.PlanEventID,
		PartIndex:    binding.PartIndex,
		SectionCount: binding.SectionCount,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.partAssemblyDrafts) >= reportPartAssemblyMaxDrafts {
		return errorResult(call.Name, common.MissionID, "validation", "too many in-process part assembly drafts", false, nil)
	}
	if _, exists := server.partAssemblyDrafts[draftID]; exists {
		return errorResult(call.Name, common.MissionID, "conflict", "part assembly draft already exists", false, []string{draftID})
	}
	server.partAssemblyDrafts[draftID] = draft
	return ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: partAssemblyFromState(*draft)}
}

func (server *Server) callReportPartAssemblyRead(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	var input reportPartAssemblyReadInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "part assembly read arguments are invalid", false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	sessionID := strings.TrimSpace(input.SessionID)
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("ses_", sessionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("rpa_", draftID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	if err := server.requireBoundWriteSession(commonMutatingInput{MissionID: missionID, SessionID: sessionID}); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	server.mu.Lock()
	draft, ok := server.partAssemblyDrafts[draftID]
	if ok {
		copyDraft := *draft
		server.mu.Unlock()
		if err := validatePartAssemblyAccess(copyDraft, missionID, sessionID); err != nil {
			return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
		}
		return ToolResult{ToolName: call.Name, MissionID: missionID, Content: partAssemblyFromState(copyDraft)}
	}
	server.mu.Unlock()
	return errorResult(call.Name, missionID, "validation", "part assembly draft was not found in this MCP process", false, []string{draftID})
}

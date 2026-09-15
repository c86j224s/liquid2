package reportparts

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"
	"time"
)

func (server *Handler) CallReportPartAssemblyStart(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	var input ReportPartAssemblyStartInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "part assembly start arguments are invalid", false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.AssemblyBinding(common)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID || input.PartIndex != binding.PartIndex || input.SectionCount != binding.SectionCount {
		return server.ErrorResult(call.Name, common.MissionID, "binding", "part assembly start does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if draftID == "" {
		draftID = server.NewID("rpa")
	}
	if err := server.ValidateID("rpa_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	now := time.Now().UTC()
	draft := &PartAssemblyDraft{
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
	server.Mu.Lock()
	defer server.Mu.Unlock()
	if len(server.State.AssemblyDrafts) >= reportPartAssemblyMaxDrafts {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "too many in-process part assembly drafts", false, nil)
	}
	if _, exists := server.State.AssemblyDrafts[draftID]; exists {
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "part assembly draft already exists", false, []string{draftID})
	}
	server.State.AssemblyDrafts[draftID] = draft
	return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: partAssemblyFromState(*draft)}
}

func (server *Handler) CallReportPartAssemblyRead(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	var input ReportPartAssemblyReadInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "part assembly read arguments are invalid", false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	sessionID := strings.TrimSpace(input.SessionID)
	draftID := strings.TrimSpace(input.DraftID)
	if err := server.ValidateID("mis_", missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.ValidateID("ses_", sessionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.ValidateID("rpa_", draftID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	if err := server.RequireSession(wire.CommonMutatingInput{MissionID: missionID, SessionID: sessionID}); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	server.Mu.Lock()
	draft, ok := server.State.AssemblyDrafts[draftID]
	if ok {
		copyDraft := *draft
		server.Mu.Unlock()
		if err := validatePartAssemblyAccess(copyDraft, missionID, sessionID); err != nil {
			return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
		}
		return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: partAssemblyFromState(copyDraft)}
	}
	server.Mu.Unlock()
	return server.ErrorResult(call.Name, missionID, "validation", "part assembly draft was not found in this MCP process", false, []string{draftID})
}

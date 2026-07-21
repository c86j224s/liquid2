package mcp

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) callReportPartAssemblyPatch(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	var input reportPartAssemblyPatchInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "part assembly patch arguments are invalid", false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if _, err := server.requirePartAssemblyBinding(common); err != nil {
		return errorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rpa_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	field := strings.TrimSpace(input.Field)
	markdown := strings.TrimSpace(input.Markdown)
	if !utf8.ValidString(markdown) {
		return errorResult(call.Name, common.MissionID, "validation", "part assembly markdown must be UTF-8 text", false, []string{draftID})
	}
	if len([]byte(markdown)) > reportPartAssemblyMaxPatchBytes {
		return errorResult(call.Name, common.MissionID, "validation", "part assembly markdown is too large", false, []string{draftID})
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	draft, ok := server.partAssemblyDrafts[draftID]
	if !ok {
		return errorResult(call.Name, common.MissionID, "validation", "part assembly draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validatePartAssemblyAccess(*draft, common.MissionID, common.SessionID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted {
		return errorResult(call.Name, common.MissionID, "conflict", "part assembly draft is already submitted", false, []string{draftID, draft.EventID})
	}
	if len(draft.Operations) >= reportPartAssemblyMaxOperations {
		return errorResult(call.Name, common.MissionID, "validation", "part assembly draft has too many operations", false, []string{draftID})
	}
	if err := applyPartAssemblyPatch(draft, field, input.AfterSectionIndex, markdown); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	draft.Operations = append(draft.Operations, partAssemblyOperation{
		Field:             field,
		AfterSectionIndex: input.AfterSectionIndex,
		Summary:           strings.TrimSpace(input.Summary),
		Bytes:             len([]byte(markdown)),
	})
	draft.UpdatedAt = nowUTC()
	return ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: partAssemblyFromState(*draft)}
}

func (server *Server) callReportPartAssemblySubmit(ctx context.Context, call ToolCall) ToolResult {
	var input reportPartAssemblySubmitInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "part assembly submit arguments are invalid", false, nil)
	}
	common, producer, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.requirePartAssemblyBinding(common)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID {
		return errorResult(call.Name, common.MissionID, "binding", "part assembly submit does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rpa_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	server.mu.Lock()
	draft, ok := server.partAssemblyDrafts[draftID]
	if !ok {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "validation", "part assembly draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validatePartAssemblyAccess(*draft, common.MissionID, common.SessionID); err != nil {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted {
		copyDraft := *draft
		server.mu.Unlock()
		return ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{copyDraft.EventID}, Content: partAssemblyFromState(copyDraft)}
	}
	assembly := reporting.PartAssembly{
		Intro:       draft.Assembly.Intro,
		Transitions: append([]reporting.PartTransition(nil), draft.Assembly.Transitions...),
		Closing:     draft.Assembly.Closing,
	}
	server.mu.Unlock()
	if strings.TrimSpace(assembly.Intro) == "" && strings.TrimSpace(assembly.Closing) == "" && len(assembly.Transitions) == 0 {
		return errorResult(call.Name, common.MissionID, "validation", "part assembly connective markdown is empty", false, []string{draftID})
	}
	binding.Producer = producer
	event, err := server.service.AppendEvent(ctx, reporting.BuildPartAssemblySubmittedAppendRequest(reporting.PartAssemblySubmittedEventRequest{
		EventID:  newMCPID("evt"),
		Binding:  binding,
		Assembly: assembly,
	}))
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	server.mu.Lock()
	if current, ok := server.partAssemblyDrafts[draftID]; ok {
		current.Submitted = true
		current.EventID = event.EventID
		current.UpdatedAt = nowUTC()
		copyDraft := *current
		server.mu.Unlock()
		return ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{event.EventID}, Content: partAssemblyFromState(copyDraft)}
	}
	server.mu.Unlock()
	return ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{event.EventID}, Content: map[string]any{
		"draft_id":        draftID,
		"mission_id":      common.MissionID,
		"submitted":       true,
		"event_id":        event.EventID,
		"submission_only": true,
	}}
}

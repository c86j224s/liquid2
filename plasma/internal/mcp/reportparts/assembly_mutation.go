package reportparts

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Handler) CallReportPartAssemblyPatch(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	var input ReportPartAssemblyPatchInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "part assembly patch arguments are invalid", false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if _, err := server.AssemblyBinding(common); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := server.ValidateID("rpa_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	field := strings.TrimSpace(input.Field)
	markdown := strings.TrimSpace(input.Markdown)
	if !utf8.ValidString(markdown) {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "part assembly markdown must be UTF-8 text", false, []string{draftID})
	}
	if len([]byte(markdown)) > reportPartAssemblyMaxPatchBytes {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "part assembly markdown is too large", false, []string{draftID})
	}

	server.Mu.Lock()
	defer server.Mu.Unlock()
	draft, ok := server.State.AssemblyDrafts[draftID]
	if !ok {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "part assembly draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validatePartAssemblyAccess(*draft, common.MissionID, common.SessionID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted {
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "part assembly draft is already submitted", false, []string{draftID, draft.EventID})
	}
	if len(draft.Operations) >= reportPartAssemblyMaxOperations {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "part assembly draft has too many operations", false, []string{draftID})
	}
	if err := applyPartAssemblyPatch(draft, field, input.AfterSectionIndex, markdown); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	draft.Operations = append(draft.Operations, PartAssemblyOperation{
		Field:             field,
		AfterSectionIndex: input.AfterSectionIndex,
		Summary:           strings.TrimSpace(input.Summary),
		Bytes:             len([]byte(markdown)),
	})
	draft.UpdatedAt = nowUTC()
	return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: partAssemblyFromState(*draft)}
}

func (server *Handler) CallReportPartAssemblySubmit(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input ReportPartAssemblySubmitInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "part assembly submit arguments are invalid", false, nil)
	}
	common, producer, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.AssemblyBinding(common)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID {
		return server.ErrorResult(call.Name, common.MissionID, "binding", "part assembly submit does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := server.ValidateID("rpa_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	server.Mu.Lock()
	draft, ok := server.State.AssemblyDrafts[draftID]
	if !ok {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "validation", "part assembly draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validatePartAssemblyAccess(*draft, common.MissionID, common.SessionID); err != nil {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted {
		copyDraft := *draft
		server.Mu.Unlock()
		return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{copyDraft.EventID}, Content: partAssemblyFromState(copyDraft)}
	}
	assembly := reporting.PartAssembly{
		Intro:       draft.Assembly.Intro,
		Transitions: append([]reporting.PartTransition(nil), draft.Assembly.Transitions...),
		Closing:     draft.Assembly.Closing,
	}
	server.Mu.Unlock()
	if strings.TrimSpace(assembly.Intro) == "" && strings.TrimSpace(assembly.Closing) == "" && len(assembly.Transitions) == 0 {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "part assembly connective markdown is empty", false, []string{draftID})
	}
	binding.Producer = producer
	event, err := server.Service.AppendEvent(ctx, reporting.BuildPartAssemblySubmittedAppendRequest(reporting.PartAssemblySubmittedEventRequest{
		EventID:  server.NewID("evt"),
		Binding:  binding,
		Assembly: assembly,
	}))
	if err != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	server.Mu.Lock()
	if current, ok := server.State.AssemblyDrafts[draftID]; ok {
		current.Submitted = true
		current.EventID = event.EventID
		current.UpdatedAt = nowUTC()
		copyDraft := *current
		server.Mu.Unlock()
		return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{event.EventID}, Content: partAssemblyFromState(copyDraft)}
	}
	server.Mu.Unlock()
	return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{event.EventID}, Content: map[string]any{
		"draft_id":        draftID,
		"mission_id":      common.MissionID,
		"submitted":       true,
		"event_id":        event.EventID,
		"submission_only": true,
	}}
}

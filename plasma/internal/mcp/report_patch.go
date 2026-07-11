package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const (
	reportPatchMaxDrafts       = 4
	reportPatchMaxBytes        = 2 * 1024 * 1024
	reportPatchMaxApplyBytes   = 256 * 1024
	reportPatchMaxOperations   = 64
	reportPatchDefaultReadSize = 32 * 1024
	reportPatchMaxReadSize     = 64 * 1024
)

type reportPatchDraft struct {
	PatchID          string
	MissionID        string
	SessionID        string
	BaseArtifactID   string
	BaseContent      string
	Title            string
	Instruction      string
	Content          string
	SessionChainKind string
	Operations       []reportPatchOperation
	Finalizing       bool
	Finalized        bool
	ArtifactID       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type reportPatchOperation struct {
	Operation string `json:"operation"`
	Summary   string `json:"summary,omitempty"`
	Bytes     int    `json:"bytes"`
}

func (server *Server) callReportPatchStart(ctx context.Context, call ToolCall) ToolResult {
	var input reportPatchStartInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundReportPatchSession(common); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.requireReportPatchBinding()
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	baseArtifactID := strings.TrimSpace(input.BaseArtifactID)
	if err := validateID("art_", baseArtifactID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{baseArtifactID})
	}
	if binding.BaseArtifactID != baseArtifactID {
		return errorResult(call.Name, common.MissionID, "validation", "report patch base artifact does not match this request", false, []string{baseArtifactID, binding.BaseArtifactID})
	}
	instruction := strings.TrimSpace(input.Instruction)
	if instruction == "" {
		return errorResult(call.Name, common.MissionID, "validation", "report patch instruction is required", false, []string{baseArtifactID})
	}
	artifact, err := server.service.GetRawArtifact(ctx, baseArtifactID)
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{baseArtifactID})
	}
	baseContent, err := reportPatchBaseContent(common.MissionID, artifact)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{baseArtifactID})
	}
	patchID := strings.TrimSpace(input.PatchID)
	if patchID == "" {
		patchID = newMCPID("rptp")
	}
	if err := validateID("rptp_", patchID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{patchID})
	}
	now := time.Now().UTC()
	patch := &reportPatchDraft{
		PatchID:          patchID,
		MissionID:        common.MissionID,
		SessionID:        common.SessionID,
		BaseArtifactID:   baseArtifactID,
		BaseContent:      baseContent,
		Title:            strings.TrimSpace(input.Title),
		Instruction:      instruction,
		Content:          baseContent,
		SessionChainKind: binding.SessionChainKind,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.reportPatches) >= reportPatchMaxDrafts {
		return errorResult(call.Name, common.MissionID, "validation", "too many in-process report patches", false, nil)
	}
	if _, exists := server.reportPatches[patchID]; exists {
		return errorResult(call.Name, common.MissionID, "conflict", "report patch already exists", false, []string{patchID})
	}
	server.reportPatches[patchID] = patch
	return ToolResult{
		ToolName:  call.Name,
		MissionID: common.MissionID,
		Content:   reportPatchFromState(*patch),
	}
}

func (server *Server) callReportPatchRead(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	var input reportPatchReadInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	sessionID := strings.TrimSpace(input.SessionID)
	patchID := strings.TrimSpace(input.PatchID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("ses_", sessionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("rptp_", patchID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{patchID})
	}
	if err := server.requireBoundReportPatchSession(commonMutatingInput{MissionID: missionID, SessionID: sessionID}); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}

	server.mu.Lock()
	patch, ok := server.reportPatches[patchID]
	if !ok {
		server.mu.Unlock()
		return errorResult(call.Name, missionID, "validation", "report patch was not found in this MCP process", false, []string{patchID})
	}
	copyPatch := *patch
	server.mu.Unlock()
	if err := validateReportPatchAccess(&copyPatch, missionID, sessionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{patchID})
	}
	content, offset, nextOffset, truncated, err := boundedReportPatchContent(copyPatch.Content, input.Offset, input.MaxBytes)
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{patchID})
	}
	return ToolResult{
		ToolName:  call.Name,
		MissionID: missionID,
		Content: reportPatchReadOutput{
			PatchID:        copyPatch.PatchID,
			MissionID:      copyPatch.MissionID,
			SessionID:      copyPatch.SessionID,
			BaseArtifactID: copyPatch.BaseArtifactID,
			Content:        content,
			Offset:         offset,
			NextOffset:     nextOffset,
			ContentLength:  len([]byte(copyPatch.Content)),
			Truncated:      truncated,
			Finalized:      copyPatch.Finalized,
			ArtifactID:     copyPatch.ArtifactID,
		},
	}
}

func (server *Server) callReportPatchApply(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	var input reportPatchApplyInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundReportPatchSession(common); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	patchID := strings.TrimSpace(input.PatchID)
	if err := validateID("rptp_", patchID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{patchID})
	}
	operation := strings.TrimSpace(input.Operation)
	replacement := input.Replacement
	if !utf8.ValidString(replacement) {
		return errorResult(call.Name, common.MissionID, "validation", "report patch replacement must be UTF-8 text", false, []string{patchID})
	}
	if len([]byte(replacement)) > reportPatchMaxApplyBytes {
		return errorResult(call.Name, common.MissionID, "validation", "report patch replacement is too large", false, []string{patchID})
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	patch, ok := server.reportPatches[patchID]
	if !ok {
		return errorResult(call.Name, common.MissionID, "validation", "report patch was not found in this MCP process", false, []string{patchID})
	}
	if err := validateReportPatchAccess(patch, common.MissionID, common.SessionID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{patchID})
	}
	if patch.Finalized {
		return errorResult(call.Name, common.MissionID, "conflict", "report patch is already finalized", false, []string{patchID, patch.ArtifactID})
	}
	if patch.Finalizing {
		return errorResult(call.Name, common.MissionID, "conflict", "report patch is already finalizing", true, []string{patchID})
	}
	if len(patch.Operations) >= reportPatchMaxOperations {
		return errorResult(call.Name, common.MissionID, "validation", "report patch has too many operations", false, []string{patchID})
	}
	if reportPatchRequiresHumanizeFidelity(patch) {
		if err := validateHumanizePatchOperation(input); err != nil {
			return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{patchID})
		}
	}
	nextContent, err := applyReportPatchOperation(patch.Content, input)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{patchID})
	}
	if len([]byte(nextContent)) > reportPatchMaxBytes {
		return errorResult(call.Name, common.MissionID, "validation", "patched report content is too large", false, []string{patchID})
	}
	if reportPatchRequiresHumanizeFidelity(patch) {
		if err := reporting.ValidateHumanizedMarkdown(patch.Content, nextContent); err != nil {
			return errorResult(call.Name, common.MissionID, "validation", reportPatchHumanizeFidelityMessage(err), false, []string{patchID})
		}
	}
	patch.Content = nextContent
	patch.Operations = append(patch.Operations, reportPatchOperation{
		Operation: operation,
		Summary:   strings.TrimSpace(input.Summary),
		Bytes:     len([]byte(replacement)),
	})
	patch.UpdatedAt = time.Now().UTC()
	return ToolResult{
		ToolName:  call.Name,
		MissionID: common.MissionID,
		Content:   reportPatchFromState(*patch),
	}
}

func (server *Server) callReportPatchFinalize(ctx context.Context, call ToolCall) ToolResult {
	var input reportPatchFinalizeInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundReportPatchSession(common); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	metadata, err := server.reportPatchFinalizeMetadata(input)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	patchID := strings.TrimSpace(input.PatchID)
	if err := validateID("rptp_", patchID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{patchID})
	}
	artifactID := strings.TrimSpace(input.ArtifactID)
	if artifactID == "" {
		artifactID = newMCPID("art")
	}
	if err := validateID("art_", artifactID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{artifactID})
	}

	server.mu.Lock()
	patch, ok := server.reportPatches[patchID]
	if !ok {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "validation", "report patch was not found in this MCP process", false, []string{patchID})
	}
	if err := validateReportPatchAccess(patch, common.MissionID, common.SessionID); err != nil {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{patchID})
	}
	if patch.Finalized {
		copyPatch := *patch
		server.mu.Unlock()
		return server.reportPatchFinalizedResult(ctx, call.Name, copyPatch)
	}
	if patch.Finalizing {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "conflict", "report patch is already finalizing", true, []string{patchID})
	}
	patch.Finalizing = true
	patch.UpdatedAt = time.Now().UTC()
	copyPatch := *patch
	server.mu.Unlock()
	finalized := false
	defer func() {
		if finalized {
			return
		}
		server.mu.Lock()
		if current, ok := server.reportPatches[patchID]; ok && !current.Finalized {
			current.Finalizing = false
			current.UpdatedAt = time.Now().UTC()
		}
		server.mu.Unlock()
	}()
	content := strings.TrimSpace(copyPatch.Content)
	if content == "" {
		return errorResult(call.Name, common.MissionID, "validation", "patched report content is required before finalization", false, []string{patchID})
	}
	if reportPatchRequiresHumanizeFidelity(&copyPatch) {
		if strings.TrimSpace(copyPatch.Content) == strings.TrimSpace(copyPatch.BaseContent) {
			return errorResult(call.Name, common.MissionID, "validation", "H5 tone pass did not make any safe Markdown changes; return NO_H5_CHANGES instead of finalizing", false, []string{patchID})
		}
		if err := reporting.ValidateHumanizedMarkdown(copyPatch.BaseContent, copyPatch.Content); err != nil {
			return errorResult(call.Name, common.MissionID, "validation", reportPatchHumanizeFidelityMessage(err), false, []string{patchID})
		}
	}
	title := firstNonEmpty(input.Title, copyPatch.Title, "Patched report")
	filename := safeReportPatchFilename(firstNonEmpty(input.Filename, title))
	eventID := newMCPID("evt")
	patchSummary := strings.TrimSpace(input.PatchSummary)
	if patchSummary == "" {
		patchSummary = reportPatchOperationSummary(copyPatch.Operations)
	}
	artifact, event, err := server.service.CreateRawArtifactWithEvent(ctx, app.CreateRawArtifactRequest{
		ArtifactID:     artifactID,
		MissionID:      common.MissionID,
		MediaType:      "text/markdown; charset=utf-8",
		Filename:       filename,
		Producer:       app.Producer{Type: "mcp_tool", ID: ToolReportPatchFinalize},
		Content:        []byte(copyPatch.Content),
		ExpectedSHA256: strings.TrimSpace(input.ExpectedSHA256),
	}, func(artifact app.RawArtifact) app.AppendEventRequest {
		return reporting.BuildPatchFinalizedAppendRequest(reporting.PatchFinalizedEventRequest{
			EventID:                      eventID,
			MissionID:                    common.MissionID,
			CorrelationID:                common.SessionID,
			PendingEventID:               metadata.PendingEventID,
			Title:                        title,
			Artifact:                     artifact,
			BaseArtifactID:               copyPatch.BaseArtifactID,
			PatchID:                      copyPatch.PatchID,
			PatchInstruction:             copyPatch.Instruction,
			PatchSummary:                 patchSummary,
			OperationCount:               len(copyPatch.Operations),
			Operations:                   copyPatch.Operations,
			AgentExecutor:                metadata.AgentExecutor,
			AgentModel:                   metadata.AgentModel,
			AgentReasoningEffort:         metadata.AgentReasoningEffort,
			AgentSessionID:               metadata.AgentSessionID,
			PreviousAgentSessionID:       metadata.PreviousAgentSessionID,
			ReturnedAgentSessionID:       metadata.ReturnedAgentSessionID,
			ReportSessionID:              metadata.ReportSessionID,
			ForkSourceAgentSessionID:     metadata.ForkSourceAgentSessionID,
			ReportSessionPolicy:          metadata.ReportSessionPolicy,
			ReportSessionPolicySelection: metadata.ReportSessionPolicySelection,
			ToolSessionID:                common.SessionID,
			MCPMode:                      metadata.MCPMode,
			ProducerToolName:             ToolReportPatchFinalize,
			SessionChainKind:             metadata.SessionChainKind,
			Producer:                     app.Producer{Type: "mcp_tool", ID: ToolReportPatchFinalize},
		})
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{patchID, artifactID})
	}
	server.mu.Lock()
	if current, ok := server.reportPatches[patchID]; ok {
		current.Finalized = true
		current.Finalizing = false
		current.ArtifactID = artifact.ArtifactID
		current.UpdatedAt = time.Now().UTC()
	}
	server.mu.Unlock()
	finalized = true
	return ToolResult{
		ToolName:        call.Name,
		MissionID:       common.MissionID,
		CreatedEventIDs: []string{event.EventID},
		Content: reportPatchFinalizeOutput{
			PatchID:        patchID,
			MissionID:      common.MissionID,
			SessionID:      common.SessionID,
			BaseArtifactID: copyPatch.BaseArtifactID,
			ContentLength:  len([]byte(copyPatch.Content)),
			Artifact:       rawArtifactFromApp(artifact),
			EventID:        event.EventID,
		},
	}
}

func (server *Server) reportPatchFinalizedResult(ctx context.Context, toolName string, patch reportPatchDraft) ToolResult {
	artifact, err := server.service.GetRawArtifact(ctx, patch.ArtifactID)
	if err != nil {
		return errorFromErr(toolName, patch.MissionID, err, []string{patch.PatchID, patch.ArtifactID})
	}
	return ToolResult{
		ToolName:  toolName,
		MissionID: patch.MissionID,
		Content: reportPatchFinalizeOutput{
			PatchID:        patch.PatchID,
			MissionID:      patch.MissionID,
			SessionID:      patch.SessionID,
			BaseArtifactID: patch.BaseArtifactID,
			ContentLength:  len([]byte(patch.Content)),
			Artifact:       rawArtifactFromApp(artifact),
		},
	}
}

func (server *Server) requireBoundReportPatchSession(input commonMutatingInput) error {
	boundMissionID := strings.TrimSpace(server.binding.MissionID)
	boundSessionID := strings.TrimSpace(server.binding.AgentSessionID)
	if boundMissionID == "" || boundSessionID == "" {
		return fmt.Errorf("%w: report patch tools require a mission-bound MCP agent session", app.ErrInvalidInput)
	}
	if input.MissionID != boundMissionID || input.SessionID != boundSessionID {
		return fmt.Errorf("%w: tool call is outside this MCP session", app.ErrInvalidInput)
	}
	return nil
}

type reportPatchFinalizeMetadata struct {
	PendingEventID               string
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	MCPMode                      string
	AgentSessionID               string
	PreviousAgentSessionID       string
	ReturnedAgentSessionID       string
	ReportSessionID              string
	ForkSourceAgentSessionID     string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	SessionChainKind             string
}

func normalizeReportPatchBinding(binding ReportPatchBinding) ReportPatchBinding {
	return ReportPatchBinding{
		BaseArtifactID:               strings.TrimSpace(binding.BaseArtifactID),
		PendingEventID:               strings.TrimSpace(binding.PendingEventID),
		AgentExecutor:                strings.TrimSpace(binding.AgentExecutor),
		AgentModel:                   strings.TrimSpace(binding.AgentModel),
		AgentReasoningEffort:         strings.TrimSpace(binding.AgentReasoningEffort),
		MCPMode:                      strings.TrimSpace(binding.MCPMode),
		AgentSessionID:               strings.TrimSpace(binding.AgentSessionID),
		PreviousAgentSessionID:       strings.TrimSpace(binding.PreviousAgentSessionID),
		ReturnedAgentSessionID:       strings.TrimSpace(binding.ReturnedAgentSessionID),
		ReportSessionID:              strings.TrimSpace(binding.ReportSessionID),
		ForkSourceAgentSessionID:     strings.TrimSpace(binding.ForkSourceAgentSessionID),
		ReportSessionPolicy:          strings.TrimSpace(binding.ReportSessionPolicy),
		ReportSessionPolicySelection: strings.TrimSpace(binding.ReportSessionPolicySelection),
		SessionChainKind:             strings.TrimSpace(binding.SessionChainKind),
	}
}

func (server *Server) requireReportPatchBinding() (ReportPatchBinding, error) {
	binding := normalizeReportPatchBinding(server.reportPatchBinding)
	if binding.BaseArtifactID == "" {
		return ReportPatchBinding{}, fmt.Errorf("%w: report patch tools require a bound base artifact", app.ErrInvalidInput)
	}
	if binding.PendingEventID == "" {
		return ReportPatchBinding{}, fmt.Errorf("%w: report patch tools require a bound pending event", app.ErrInvalidInput)
	}
	if binding.ReportSessionID == "" {
		return ReportPatchBinding{}, fmt.Errorf("%w: report patch tools require a bound report session", app.ErrInvalidInput)
	}
	if binding.AgentExecutor == "" {
		return ReportPatchBinding{}, fmt.Errorf("%w: report patch tools require a bound agent executor", app.ErrInvalidInput)
	}
	return binding, nil
}

func (server *Server) reportPatchFinalizeMetadata(input reportPatchFinalizeInput) (reportPatchFinalizeMetadata, error) {
	binding, err := server.requireReportPatchBinding()
	if err != nil {
		return reportPatchFinalizeMetadata{}, err
	}
	pendingEventID, err := reportPatchBoundValue("pending_event_id", input.PendingEventID, binding.PendingEventID, true)
	if err != nil {
		return reportPatchFinalizeMetadata{}, err
	}
	agentExecutor, err := reportPatchBoundValue("agent_executor", input.AgentExecutor, binding.AgentExecutor, true)
	if err != nil {
		return reportPatchFinalizeMetadata{}, err
	}
	reportSessionID, err := reportPatchBoundValue("report_session_id", input.ReportSessionID, binding.ReportSessionID, true)
	if err != nil {
		return reportPatchFinalizeMetadata{}, err
	}
	reportSessionPolicy, err := reportPatchBoundValue("report_session_policy", input.ReportSessionPolicy, binding.ReportSessionPolicy, true)
	if err != nil {
		return reportPatchFinalizeMetadata{}, err
	}
	reportSessionPolicySelection, err := reportPatchBoundValue("report_session_policy_selection", input.ReportSessionPolicySelection, binding.ReportSessionPolicySelection, true)
	if err != nil {
		return reportPatchFinalizeMetadata{}, err
	}
	agentModel, err := reportPatchBoundValue("agent_model", input.AgentModel, binding.AgentModel, false)
	if err != nil {
		return reportPatchFinalizeMetadata{}, err
	}
	agentReasoningEffort, err := reportPatchBoundValue("agent_reasoning_effort", input.AgentReasoningEffort, binding.AgentReasoningEffort, false)
	if err != nil {
		return reportPatchFinalizeMetadata{}, err
	}
	mcpMode, err := reportPatchBoundValue("mcp_mode", input.MCPMode, binding.MCPMode, false)
	if err != nil {
		return reportPatchFinalizeMetadata{}, err
	}
	agentSessionID, err := reportPatchBoundValue("agent_session_id", input.AgentSessionID, binding.AgentSessionID, false)
	if err != nil {
		return reportPatchFinalizeMetadata{}, err
	}
	previousAgentSessionID, err := reportPatchBoundValue("previous_agent_session_id", input.PreviousAgentSessionID, binding.PreviousAgentSessionID, false)
	if err != nil {
		return reportPatchFinalizeMetadata{}, err
	}
	returnedAgentSessionID, err := reportPatchBoundValue("returned_agent_session_id", input.ReturnedAgentSessionID, binding.ReturnedAgentSessionID, false)
	if err != nil {
		return reportPatchFinalizeMetadata{}, err
	}
	forkSourceAgentSessionID, err := reportPatchBoundValue("fork_source_agent_session_id", input.ForkSourceAgentSessionID, binding.ForkSourceAgentSessionID, false)
	if err != nil {
		return reportPatchFinalizeMetadata{}, err
	}
	sessionChainKind, err := reportPatchBoundValue("session_chain_kind", input.SessionChainKind, binding.SessionChainKind, false)
	if err != nil {
		return reportPatchFinalizeMetadata{}, err
	}
	agentSessionID = firstNonEmpty(agentSessionID, reportSessionID)
	previousAgentSessionID = firstNonEmpty(previousAgentSessionID, reportSessionID)
	returnedAgentSessionID = firstNonEmpty(returnedAgentSessionID, agentSessionID, reportSessionID)
	sessionChainKind = firstNonEmpty(sessionChainKind, "report_patch_session")
	return reportPatchFinalizeMetadata{
		PendingEventID:               pendingEventID,
		AgentExecutor:                agentExecutor,
		AgentModel:                   agentModel,
		AgentReasoningEffort:         agentReasoningEffort,
		MCPMode:                      mcpMode,
		AgentSessionID:               agentSessionID,
		PreviousAgentSessionID:       previousAgentSessionID,
		ReturnedAgentSessionID:       returnedAgentSessionID,
		ReportSessionID:              reportSessionID,
		ForkSourceAgentSessionID:     forkSourceAgentSessionID,
		ReportSessionPolicy:          reportSessionPolicy,
		ReportSessionPolicySelection: reportSessionPolicySelection,
		SessionChainKind:             sessionChainKind,
	}, nil
}

func reportPatchRequiresHumanizeFidelity(patch *reportPatchDraft) bool {
	if patch == nil {
		return false
	}
	kind := strings.ToLower(strings.TrimSpace(patch.SessionChainKind))
	return strings.Contains(kind, "h5") || strings.Contains(kind, "humanize")
}

func reportPatchHumanizeFidelityMessage(err error) string {
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		msg = "humanized Markdown failed fidelity guard"
	}
	return msg + "; this H5 tone pass may only make small Korean prose tone edits. Keep quoted text, source/citation lines, numbers, headings, links, code, lists, and block structure unchanged, then retry with a smaller replacement."
}

func validateHumanizePatchOperation(input reportPatchApplyInput) error {
	operation := strings.TrimSpace(input.Operation)
	if operation != "replace" {
		return fmt.Errorf("%w: H5 tone pass only supports small replace operations", app.ErrInvalidInput)
	}
	if input.ReplaceAll {
		return fmt.Errorf("%w: H5 tone pass cannot use replace_all", app.ErrInvalidInput)
	}
	if strings.TrimSpace(input.MatchText) == "" {
		return fmt.Errorf("%w: H5 tone pass requires exact match_text", app.ErrInvalidInput)
	}
	replacementRunes := len([]rune(input.Replacement))
	matchRunes := len([]rune(input.MatchText))
	if replacementRunes > 500 || matchRunes > 500 {
		return fmt.Errorf("%w: H5 tone pass replacement is too broad; patch one small sentence or phrase at a time", app.ErrInvalidInput)
	}
	if replacementRunes > matchRunes+180 {
		return fmt.Errorf("%w: H5 tone pass replacement expands the report too much", app.ErrInvalidInput)
	}
	return nil
}

func reportPatchBoundValue(name string, provided string, bound string, required bool) (string, error) {
	provided = strings.TrimSpace(provided)
	bound = strings.TrimSpace(bound)
	if bound != "" {
		// The MCP process is already bound to one report patch request. Use the
		// server-side lineage metadata even when the model echoes a stale value.
		return bound, nil
	}
	if required && provided == "" {
		return "", fmt.Errorf("%w: %s is required", app.ErrInvalidInput, name)
	}
	return provided, nil
}

func validateReportPatchAccess(patch *reportPatchDraft, missionID string, sessionID string) error {
	if patch == nil {
		return fmt.Errorf("%w: report patch is required", app.ErrInvalidInput)
	}
	if patch.MissionID != missionID || patch.SessionID != sessionID {
		return fmt.Errorf("%w: report patch belongs to another MCP session", app.ErrInvalidInput)
	}
	return nil
}

func reportPatchBaseContent(missionID string, artifact app.RawArtifact) (string, error) {
	if artifact.MissionID != missionID {
		return "", fmt.Errorf("%w: base report artifact belongs to another mission", app.ErrInvalidInput)
	}
	mediaType := strings.ToLower(strings.TrimSpace(artifact.MediaType))
	if !strings.HasPrefix(mediaType, "text/markdown") {
		return "", fmt.Errorf("%w: base report artifact must be readable Markdown text", app.ErrInvalidInput)
	}
	if len(artifact.Content) == 0 {
		return "", fmt.Errorf("%w: base report artifact is empty", app.ErrInvalidInput)
	}
	if len(artifact.Content) > reportPatchMaxBytes {
		return "", fmt.Errorf("%w: base report artifact is too large for MCP patching", app.ErrInvalidInput)
	}
	if !utf8.Valid(artifact.Content) {
		return "", fmt.Errorf("%w: base report artifact must be UTF-8 text", app.ErrInvalidInput)
	}
	return string(artifact.Content), nil
}

func applyReportPatchOperation(content string, input reportPatchApplyInput) (string, error) {
	operation := strings.TrimSpace(input.Operation)
	matchText := input.MatchText
	switch operation {
	case "append":
		return content + input.Replacement, nil
	case "insert_after":
		if matchText == "" {
			return "", fmt.Errorf("%w: insert_after requires match_text", app.ErrInvalidInput)
		}
		index := strings.Index(content, matchText)
		if index < 0 {
			return "", fmt.Errorf("%w: match_text was not found for insert_after", app.ErrInvalidInput)
		}
		insertAt := index + len(matchText)
		return content[:insertAt] + input.Replacement + content[insertAt:], nil
	case "replace":
		if matchText == "" {
			return "", fmt.Errorf("%w: replace requires match_text", app.ErrInvalidInput)
		}
		if input.ReplaceAll {
			if !strings.Contains(content, matchText) {
				return "", fmt.Errorf("%w: match_text was not found for replace", app.ErrInvalidInput)
			}
			return strings.ReplaceAll(content, matchText, input.Replacement), nil
		}
		occurrence := input.Occurrence
		if occurrence <= 0 {
			occurrence = 1
		}
		return replaceNth(content, matchText, input.Replacement, occurrence)
	default:
		return "", fmt.Errorf("%w: unsupported report patch operation %q", app.ErrInvalidInput, operation)
	}
}

func replaceNth(content string, old string, replacement string, occurrence int) (string, error) {
	searchStart := 0
	for current := 1; ; current++ {
		index := strings.Index(content[searchStart:], old)
		if index < 0 {
			return "", fmt.Errorf("%w: match_text occurrence was not found for replace", app.ErrInvalidInput)
		}
		absolute := searchStart + index
		if current == occurrence {
			return content[:absolute] + replacement + content[absolute+len(old):], nil
		}
		searchStart = absolute + len(old)
	}
}

func reportPatchFromState(patch reportPatchDraft) reportPatchOutput {
	state := "open"
	if patch.Finalized {
		state = "finalized"
	} else if patch.Finalizing {
		state = "finalizing"
	}
	return reportPatchOutput{
		PatchID:        patch.PatchID,
		MissionID:      patch.MissionID,
		SessionID:      patch.SessionID,
		BaseArtifactID: patch.BaseArtifactID,
		Title:          patch.Title,
		State:          state,
		ContentLength:  len([]byte(patch.Content)),
		OperationCount: len(patch.Operations),
		Finalized:      patch.Finalized,
		ArtifactID:     patch.ArtifactID,
	}
}

func boundedReportPatchContent(content string, offset int, maxBytes int) (string, int, int, bool, error) {
	raw := []byte(content)
	if offset < 0 {
		return "", 0, 0, false, fmt.Errorf("%w: report patch offset must be non-negative", app.ErrInvalidInput)
	}
	if offset > len(raw) {
		return "", 0, 0, false, fmt.Errorf("%w: report patch offset is beyond content length", app.ErrInvalidInput)
	}
	if offset < len(raw) && !utf8.RuneStart(raw[offset]) {
		return "", 0, 0, false, fmt.Errorf("%w: report patch offset must align to UTF-8 boundary", app.ErrInvalidInput)
	}
	limit := maxBytes
	if limit <= 0 {
		limit = reportPatchDefaultReadSize
	} else if limit > reportPatchMaxReadSize {
		limit = reportPatchMaxReadSize
	}
	remaining := raw[offset:]
	if len(remaining) <= limit {
		return string(remaining), offset, 0, false, nil
	}
	cut := offset + limit
	for cut > offset && !utf8.Valid(raw[offset:cut]) {
		cut--
	}
	if cut == offset {
		return "", 0, 0, false, fmt.Errorf("%w: report patch could not be sliced as UTF-8", app.ErrInvalidInput)
	}
	return string(raw[offset:cut]), offset, cut, true, nil
}

func reportPatchOperationSummary(operations []reportPatchOperation) string {
	if len(operations) == 0 {
		return "No explicit patch operations were recorded."
	}
	parts := make([]string, 0, len(operations))
	for _, operation := range operations {
		if strings.TrimSpace(operation.Summary) != "" {
			parts = append(parts, strings.TrimSpace(operation.Summary))
			continue
		}
		parts = append(parts, operation.Operation)
	}
	return strings.Join(parts, "; ")
}

func safeReportPatchFilename(value string) string {
	base := strings.TrimSpace(value)
	if base == "" {
		base = "patched-report"
	}
	base = filepath.Base(base)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	var builder strings.Builder
	for _, r := range strings.ToLower(base) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
		case unicode.IsLetter(r), unicode.IsNumber(r):
			builder.WriteRune(r)
		case r == '-' || r == '_':
			builder.WriteRune(r)
		case unicode.IsSpace(r) || r == '.':
			builder.WriteRune('-')
		}
		if builder.Len() >= 80 {
			break
		}
	}
	name := strings.Trim(builder.String(), "-_")
	if name == "" {
		name = "patched-report"
	}
	return name + ".md"
}

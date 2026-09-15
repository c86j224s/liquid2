package reportpatch

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type State struct{ Drafts map[string]*Draft }
type Service interface {
	GetRawArtifact(context.Context, string) (artifactcontract.Raw, error)
	CreateRawArtifactWithEvent(context.Context, artifactcontract.CreateRequest, func(artifactcontract.Raw) ledger.AppendRequest) (artifactcontract.Raw, ledger.Event, error)
}
type Handler struct {
	State                *State
	Mu                   *sync.Mutex
	Service              Service
	Binding              func() ReportPatchBinding
	MissionID, SessionID func() string
	Decode               func(json.RawMessage, any) error
	NormalizeInput       func(wire.CommonMutatingInput) (wire.CommonMutatingInput, ledger.Producer, error)
	ErrorResult          func(string, string, string, string, bool, []string) wire.ToolResult
	ErrorFromErr         func(string, string, error, []string) wire.ToolResult
	NewID                func(string) string
	ArtifactOutput       func(artifactcontract.Raw) wire.RawArtifactOutput
}

const (
	ReportPatchMaxDrafts       = 4
	ReportPatchMaxBytes        = 2 * 1024 * 1024
	ReportPatchMaxApplyBytes   = 256 * 1024
	ReportPatchMaxOperations   = 64
	ReportPatchDefaultReadSize = 32 * 1024
	ReportPatchMaxReadSize     = 64 * 1024
)

type Draft struct {
	PatchID          string
	MissionID        string
	SessionID        string
	BaseArtifactID   string
	BaseContent      string
	Title            string
	Instruction      string
	Content          string
	SessionChainKind string
	Operations       []Operation
	Finalizing       bool
	Finalized        bool
	ArtifactID       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Operation struct {
	Operation   string `json:"operation"`
	Summary     string `json:"summary,omitempty"`
	Bytes       int    `json:"bytes"`
	Category    string `json:"category,omitempty"`
	Reason      string `json:"-"`
	MatchText   string `json:"-"`
	Replacement string `json:"-"`
	Occurrence  int    `json:"-"`
}

func (server *Handler) Start(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input ReportPatchStartInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundReportPatchSession(common); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.requireReportPatchBinding()
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	baseArtifactID := strings.TrimSpace(input.BaseArtifactID)
	if err := validateID("art_", baseArtifactID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{baseArtifactID})
	}
	if binding.BaseArtifactID != baseArtifactID {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "report patch base artifact does not match this request", false, []string{baseArtifactID, binding.BaseArtifactID})
	}
	instruction := strings.TrimSpace(input.Instruction)
	if instruction == "" {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "report patch instruction is required", false, []string{baseArtifactID})
	}
	artifact, err := server.Service.GetRawArtifact(ctx, baseArtifactID)
	if err != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{baseArtifactID})
	}
	baseContent, err := reportPatchBaseContent(common.MissionID, artifact)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{baseArtifactID})
	}
	patchID := strings.TrimSpace(input.PatchID)
	if patchID == "" {
		patchID = server.NewID("rptp")
	}
	if err := validateID("rptp_", patchID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{patchID})
	}
	now := time.Now().UTC()
	patch := &Draft{
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

	server.Mu.Lock()
	defer server.Mu.Unlock()
	if len(server.State.Drafts) >= ReportPatchMaxDrafts {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "too many in-process report patches", false, nil)
	}
	if _, exists := server.State.Drafts[patchID]; exists {
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "report patch already exists", false, []string{patchID})
	}
	server.State.Drafts[patchID] = patch
	return wire.ToolResult{
		ToolName:  call.Name,
		MissionID: common.MissionID,
		Content:   reportPatchFromState(*patch),
	}
}

func (server *Handler) Read(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	var input ReportPatchReadInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	sessionID := strings.TrimSpace(input.SessionID)
	patchID := strings.TrimSpace(input.PatchID)
	if err := validateID("mis_", missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("ses_", sessionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("rptp_", patchID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{patchID})
	}
	if err := server.requireBoundReportPatchSession(wire.CommonMutatingInput{MissionID: missionID, SessionID: sessionID}); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}

	server.Mu.Lock()
	patch, ok := server.State.Drafts[patchID]
	if !ok {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, missionID, "validation", "report patch was not found in this MCP process", false, []string{patchID})
	}
	copyPatch := *patch
	server.Mu.Unlock()
	if err := validateReportPatchAccess(&copyPatch, missionID, sessionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{patchID})
	}
	content, offset, nextOffset, truncated, err := BoundedContent(copyPatch.Content, input.Offset, input.MaxBytes)
	if err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{patchID})
	}
	return wire.ToolResult{
		ToolName:  call.Name,
		MissionID: missionID,
		Content: ReportPatchReadOutput{
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

func (server *Handler) Apply(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	var input ReportPatchApplyInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundReportPatchSession(common); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	patchID := strings.TrimSpace(input.PatchID)
	if err := validateID("rptp_", patchID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{patchID})
	}
	operation := strings.TrimSpace(input.Operation)
	replacement := input.Replacement
	if !utf8.ValidString(replacement) {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "report patch replacement must be UTF-8 text", false, []string{patchID})
	}
	if len([]byte(replacement)) > ReportPatchMaxApplyBytes {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "report patch replacement is too large", false, []string{patchID})
	}

	server.Mu.Lock()
	defer server.Mu.Unlock()
	patch, ok := server.State.Drafts[patchID]
	if !ok {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "report patch was not found in this MCP process", false, []string{patchID})
	}
	if err := validateReportPatchAccess(patch, common.MissionID, common.SessionID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{patchID})
	}
	if patch.Finalized {
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "report patch is already finalized", false, []string{patchID, patch.ArtifactID})
	}
	if patch.Finalizing {
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "report patch is already finalizing", true, []string{patchID})
	}
	if len(patch.Operations) >= ReportPatchMaxOperations {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "report patch has too many operations", false, []string{patchID})
	}
	if reportPatchRequiresHumanizeFidelity(patch) {
		if err := validateHumanizePatchOperation(input); err != nil {
			return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{patchID})
		}
	}
	nextContent, err := ApplyOperation(patch.Content, input)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{patchID})
	}
	if len([]byte(nextContent)) > ReportPatchMaxBytes {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "patched report content is too large", false, []string{patchID})
	}
	if reportPatchRequiresHumanizeFidelity(patch) {
		if err := reporting.ValidateHumanizedMarkdown(patch.Content, nextContent); err != nil {
			return server.ErrorResult(call.Name, common.MissionID, "validation", reportPatchHumanizeFidelityMessage(err), false, []string{patchID})
		}
	}
	patch.Content = nextContent
	patch.Operations = append(patch.Operations, Operation{
		Operation: operation,
		Summary:   strings.TrimSpace(input.Summary),
		Bytes:     len([]byte(replacement)),
	})
	patch.UpdatedAt = time.Now().UTC()
	return wire.ToolResult{
		ToolName:  call.Name,
		MissionID: common.MissionID,
		Content:   reportPatchFromState(*patch),
	}
}

func (server *Handler) Finalize(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input ReportPatchFinalizeInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundReportPatchSession(common); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	metadata, err := server.reportPatchFinalizeMetadata(input)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	patchID := strings.TrimSpace(input.PatchID)
	if err := validateID("rptp_", patchID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{patchID})
	}
	artifactID := strings.TrimSpace(input.ArtifactID)
	if artifactID == "" {
		artifactID = server.NewID("art")
	}
	if err := validateID("art_", artifactID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{artifactID})
	}

	server.Mu.Lock()
	patch, ok := server.State.Drafts[patchID]
	if !ok {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "validation", "report patch was not found in this MCP process", false, []string{patchID})
	}
	if err := validateReportPatchAccess(patch, common.MissionID, common.SessionID); err != nil {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{patchID})
	}
	if patch.Finalized {
		copyPatch := *patch
		server.Mu.Unlock()
		return server.reportPatchFinalizedResult(ctx, call.Name, copyPatch)
	}
	if patch.Finalizing {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "report patch is already finalizing", true, []string{patchID})
	}
	patch.Finalizing = true
	patch.UpdatedAt = time.Now().UTC()
	copyPatch := *patch
	server.Mu.Unlock()
	finalized := false
	defer func() {
		if finalized {
			return
		}
		server.Mu.Lock()
		if current, ok := server.State.Drafts[patchID]; ok && !current.Finalized {
			current.Finalizing = false
			current.UpdatedAt = time.Now().UTC()
		}
		server.Mu.Unlock()
	}()
	content := strings.TrimSpace(copyPatch.Content)
	if content == "" {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "patched report content is required before finalization", false, []string{patchID})
	}
	if reportPatchRequiresHumanizeFidelity(&copyPatch) {
		if strings.TrimSpace(copyPatch.Content) == strings.TrimSpace(copyPatch.BaseContent) {
			return server.ErrorResult(call.Name, common.MissionID, "validation", "H5 tone pass did not make any safe Markdown changes; return NO_H5_CHANGES instead of finalizing", false, []string{patchID})
		}
		if err := reporting.ValidateHumanizedMarkdown(copyPatch.BaseContent, copyPatch.Content); err != nil {
			return server.ErrorResult(call.Name, common.MissionID, "validation", reportPatchHumanizeFidelityMessage(err), false, []string{patchID})
		}
	}
	title := firstNonEmpty(input.Title, copyPatch.Title, "Patched report")
	filename := safeReportPatchFilename(firstNonEmpty(input.Filename, title))
	eventID := server.NewID("evt")
	patchSummary := strings.TrimSpace(input.PatchSummary)
	if patchSummary == "" {
		patchSummary = reportPatchOperationSummary(copyPatch.Operations)
	}
	artifact, event, err := server.Service.CreateRawArtifactWithEvent(ctx, artifactcontract.CreateRequest{
		ArtifactID:     artifactID,
		MissionID:      common.MissionID,
		MediaType:      "text/markdown; charset=utf-8",
		Filename:       filename,
		Producer:       ledger.Producer{Type: "mcp_tool", ID: mcptools.ToolReportPatchFinalize},
		Content:        []byte(copyPatch.Content),
		ExpectedSHA256: strings.TrimSpace(input.ExpectedSHA256),
	}, func(artifact artifactcontract.Raw) ledger.AppendRequest {
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
			ProducerToolName:             mcptools.ToolReportPatchFinalize,
			SessionChainKind:             metadata.SessionChainKind,
			Producer:                     ledger.Producer{Type: "mcp_tool", ID: mcptools.ToolReportPatchFinalize},
		})
	})
	if err != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{patchID, artifactID})
	}
	server.Mu.Lock()
	if current, ok := server.State.Drafts[patchID]; ok {
		current.Finalized = true
		current.Finalizing = false
		current.ArtifactID = artifact.ArtifactID
		current.UpdatedAt = time.Now().UTC()
	}
	server.Mu.Unlock()
	finalized = true
	return wire.ToolResult{
		ToolName:        call.Name,
		MissionID:       common.MissionID,
		CreatedEventIDs: []string{event.EventID},
		Content: ReportPatchFinalizeOutput{
			PatchID:        patchID,
			MissionID:      common.MissionID,
			SessionID:      common.SessionID,
			BaseArtifactID: copyPatch.BaseArtifactID,
			ContentLength:  len([]byte(copyPatch.Content)),
			Artifact:       server.ArtifactOutput(artifact),
			EventID:        event.EventID,
		},
	}
}

func (server *Handler) reportPatchFinalizedResult(ctx context.Context, toolName string, patch Draft) wire.ToolResult {
	artifact, err := server.Service.GetRawArtifact(ctx, patch.ArtifactID)
	if err != nil {
		return server.ErrorFromErr(toolName, patch.MissionID, err, []string{patch.PatchID, patch.ArtifactID})
	}
	return wire.ToolResult{
		ToolName:  toolName,
		MissionID: patch.MissionID,
		Content: ReportPatchFinalizeOutput{
			PatchID:        patch.PatchID,
			MissionID:      patch.MissionID,
			SessionID:      patch.SessionID,
			BaseArtifactID: patch.BaseArtifactID,
			ContentLength:  len([]byte(patch.Content)),
			Artifact:       server.ArtifactOutput(artifact),
		},
	}
}

func (server *Handler) requireBoundReportPatchSession(input wire.CommonMutatingInput) error {
	boundMissionID := strings.TrimSpace(server.MissionID())
	boundSessionID := strings.TrimSpace(server.SessionID())
	if boundMissionID == "" || boundSessionID == "" {
		return fmt.Errorf("%w: report patch tools require a mission-bound MCP agent session", producterror.ErrInvalidInput)
	}
	if input.MissionID != boundMissionID || input.SessionID != boundSessionID {
		return fmt.Errorf("%w: tool call is outside this MCP session", producterror.ErrInvalidInput)
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

func NormalizeBinding(binding ReportPatchBinding) ReportPatchBinding {
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

func (server *Handler) requireReportPatchBinding() (ReportPatchBinding, error) {
	binding := NormalizeBinding(server.Binding())
	if binding.BaseArtifactID == "" {
		return ReportPatchBinding{}, fmt.Errorf("%w: report patch tools require a bound base artifact", producterror.ErrInvalidInput)
	}
	if binding.PendingEventID == "" {
		return ReportPatchBinding{}, fmt.Errorf("%w: report patch tools require a bound pending event", producterror.ErrInvalidInput)
	}
	if binding.ReportSessionID == "" {
		return ReportPatchBinding{}, fmt.Errorf("%w: report patch tools require a bound report session", producterror.ErrInvalidInput)
	}
	if binding.AgentExecutor == "" {
		return ReportPatchBinding{}, fmt.Errorf("%w: report patch tools require a bound agent executor", producterror.ErrInvalidInput)
	}
	return binding, nil
}

func (server *Handler) reportPatchFinalizeMetadata(input ReportPatchFinalizeInput) (reportPatchFinalizeMetadata, error) {
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

func reportPatchRequiresHumanizeFidelity(patch *Draft) bool {
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

func validateHumanizePatchOperation(input ReportPatchApplyInput) error {
	operation := strings.TrimSpace(input.Operation)
	if operation != "replace" {
		return fmt.Errorf("%w: H5 tone pass only supports small replace operations", producterror.ErrInvalidInput)
	}
	if input.ReplaceAll {
		return fmt.Errorf("%w: H5 tone pass cannot use replace_all", producterror.ErrInvalidInput)
	}
	if strings.TrimSpace(input.MatchText) == "" {
		return fmt.Errorf("%w: H5 tone pass requires exact match_text", producterror.ErrInvalidInput)
	}
	replacementRunes := len([]rune(input.Replacement))
	matchRunes := len([]rune(input.MatchText))
	if replacementRunes > 500 || matchRunes > 500 {
		return fmt.Errorf("%w: H5 tone pass replacement is too broad; patch one small sentence or phrase at a time", producterror.ErrInvalidInput)
	}
	if replacementRunes > matchRunes+180 {
		return fmt.Errorf("%w: H5 tone pass replacement expands the report too much", producterror.ErrInvalidInput)
	}
	return nil
}

func reportPatchBoundValue(name string, provided string, bound string, required bool) (string, error) {
	provided = strings.TrimSpace(provided)
	bound = strings.TrimSpace(bound)
	if bound != "" {
		// MCP 프로세스는 이미 하나의 report patch 요청에 묶여 있다. 모델이 stale 값을
		// 되풀이하더라도 서버 쪽 lineage metadata를 사용한다.
		return bound, nil
	}
	if required && provided == "" {
		return "", fmt.Errorf("%w: %s is required", producterror.ErrInvalidInput, name)
	}
	return provided, nil
}

func validateReportPatchAccess(patch *Draft, missionID string, sessionID string) error {
	if patch == nil {
		return fmt.Errorf("%w: report patch is required", producterror.ErrInvalidInput)
	}
	if patch.MissionID != missionID || patch.SessionID != sessionID {
		return fmt.Errorf("%w: report patch belongs to another MCP session", producterror.ErrInvalidInput)
	}
	return nil
}

func reportPatchBaseContent(missionID string, artifact artifactcontract.Raw) (string, error) {
	if artifact.MissionID != missionID {
		return "", fmt.Errorf("%w: base report artifact belongs to another mission", producterror.ErrInvalidInput)
	}
	mediaType := strings.ToLower(strings.TrimSpace(artifact.MediaType))
	if !strings.HasPrefix(mediaType, "text/markdown") {
		return "", fmt.Errorf("%w: base report artifact must be readable Markdown text", producterror.ErrInvalidInput)
	}
	if len(artifact.Content) == 0 {
		return "", fmt.Errorf("%w: base report artifact is empty", producterror.ErrInvalidInput)
	}
	if len(artifact.Content) > ReportPatchMaxBytes {
		return "", fmt.Errorf("%w: base report artifact is too large for MCP patching", producterror.ErrInvalidInput)
	}
	if !utf8.Valid(artifact.Content) {
		return "", fmt.Errorf("%w: base report artifact must be UTF-8 text", producterror.ErrInvalidInput)
	}
	return string(artifact.Content), nil
}

func ApplyOperation(content string, input ReportPatchApplyInput) (string, error) {
	operation := strings.TrimSpace(input.Operation)
	matchText := input.MatchText
	switch operation {
	case "append":
		return content + input.Replacement, nil
	case "insert_after":
		if matchText == "" {
			return "", fmt.Errorf("%w: insert_after requires match_text", producterror.ErrInvalidInput)
		}
		index := strings.Index(content, matchText)
		if index < 0 {
			return "", fmt.Errorf("%w: match_text was not found for insert_after", producterror.ErrInvalidInput)
		}
		insertAt := index + len(matchText)
		return content[:insertAt] + input.Replacement + content[insertAt:], nil
	case "replace":
		if matchText == "" {
			return "", fmt.Errorf("%w: replace requires match_text", producterror.ErrInvalidInput)
		}
		if input.ReplaceAll {
			if !strings.Contains(content, matchText) {
				return "", fmt.Errorf("%w: match_text was not found for replace", producterror.ErrInvalidInput)
			}
			return strings.ReplaceAll(content, matchText, input.Replacement), nil
		}
		occurrence := input.Occurrence
		if occurrence <= 0 {
			occurrence = 1
		}
		return replaceNth(content, matchText, input.Replacement, occurrence)
	default:
		return "", fmt.Errorf("%w: unsupported report patch operation %q", producterror.ErrInvalidInput, operation)
	}
}

func replaceNth(content string, old string, replacement string, occurrence int) (string, error) {
	searchStart := 0
	for current := 1; ; current++ {
		index := strings.Index(content[searchStart:], old)
		if index < 0 {
			return "", fmt.Errorf("%w: match_text occurrence was not found for replace", producterror.ErrInvalidInput)
		}
		absolute := searchStart + index
		if current == occurrence {
			return content[:absolute] + replacement + content[absolute+len(old):], nil
		}
		searchStart = absolute + len(old)
	}
}

func reportPatchFromState(patch Draft) ReportPatchOutput {
	state := "open"
	if patch.Finalized {
		state = "finalized"
	} else if patch.Finalizing {
		state = "finalizing"
	}
	return ReportPatchOutput{
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

func BoundedContent(content string, offset int, maxBytes int) (string, int, int, bool, error) {
	raw := []byte(content)
	if offset < 0 {
		return "", 0, 0, false, fmt.Errorf("%w: report patch offset must be non-negative", producterror.ErrInvalidInput)
	}
	if offset > len(raw) {
		return "", 0, 0, false, fmt.Errorf("%w: report patch offset is beyond content length", producterror.ErrInvalidInput)
	}
	if offset < len(raw) && !utf8.RuneStart(raw[offset]) {
		return "", 0, 0, false, fmt.Errorf("%w: report patch offset must align to UTF-8 boundary", producterror.ErrInvalidInput)
	}
	limit := maxBytes
	if limit <= 0 {
		limit = ReportPatchDefaultReadSize
	} else if limit > ReportPatchMaxReadSize {
		limit = ReportPatchMaxReadSize
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
		return "", 0, 0, false, fmt.Errorf("%w: report patch could not be sliced as UTF-8", producterror.ErrInvalidInput)
	}
	return string(raw[offset:cut]), offset, cut, true, nil
}

func reportPatchOperationSummary(operations []Operation) string {
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

func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", producterror.ErrInvalidInput, prefix)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

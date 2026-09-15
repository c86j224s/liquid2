package web

import (
	"context"
	"fmt"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"strings"
)

func (server *Server) startDesignedReportHTMLExport(ctx context.Context, missionID string, sourceArtifact artifactcontract.Raw, req reportDesignRequest) (map[string]any, bool, error) {
	if !isMarkdownMediaType(sourceArtifact.MediaType) {
		return nil, false, fmt.Errorf("%w: designed HTML export requires a Markdown report artifact", producterror.ErrInvalidInput)
	}
	images, notes, err := server.inlineReportImages(ctx, missionID)
	if err != nil {
		return nil, false, err
	}
	imageSetFingerprint := designedReportImageSetFingerprint(images, notes)
	if cached, ok, err := server.existingDesignedReportHTMLExport(ctx, missionID, sourceArtifact.ArtifactID, imageSetFingerprint); err != nil {
		return nil, false, err
	} else if ok {
		return map[string]any{
			"status":          "completed",
			"artifact":        rawArtifactMetadata(cached.Artifact),
			"source_artifact": rawArtifactMetadata(sourceArtifact),
			"event":           cached.Event,
			"content":         string(cached.Artifact.Content),
		}, false, nil
	}
	executorName, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		return nil, false, err
	}
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return nil, false, fmt.Errorf("%w: designed HTML export requires an agent executor", producterror.ErrInvalidInput)
	}
	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	unlockTurns := server.turns.lock(missionID)
	defer unlockTurns()
	sourceArtifact, err = server.reportArtifact(ctx, missionID, sourceArtifact.ArtifactID)
	if err != nil {
		return nil, false, err
	}
	if !isMarkdownMediaType(sourceArtifact.MediaType) {
		return nil, false, fmt.Errorf("%w: designed HTML export requires a Markdown report artifact", producterror.ErrInvalidInput)
	}
	if cached, ok, err := server.existingDesignedReportHTMLExport(ctx, missionID, sourceArtifact.ArtifactID, imageSetFingerprint); err != nil {
		return nil, false, err
	} else if ok {
		return map[string]any{
			"status":          "completed",
			"artifact":        rawArtifactMetadata(cached.Artifact),
			"source_artifact": rawArtifactMetadata(sourceArtifact),
			"event":           cached.Event,
			"content":         string(cached.Artifact.Content),
		}, false, nil
	}
	if err := server.validateMissionAgentExecutor(ctx, missionID, executorName); err != nil {
		return nil, false, err
	}
	if err := server.reconcileStaleAgentTurn(ctx, missionID); err != nil {
		return nil, false, err
	}
	if err := server.reconcileStaleReportDrafts(ctx, missionID); err != nil {
		return nil, false, err
	}
	if err := server.reconcileStaleDesignedReportExports(ctx, missionID); err != nil {
		return nil, false, err
	}
	if server.hasOpenReportDraft(ctx, missionID) {
		return nil, false, errReportDraftRunning
	}
	if server.hasOpenAgentTurn(ctx, missionID) {
		return nil, false, fmt.Errorf("%w: agent turn is already running for this mission", producterror.ErrInvalidInput)
	}
	if active := server.activeWorkflowRun(ctx, missionID); active != nil {
		return nil, false, fmt.Errorf("%w: workflow %s is %s for this mission", producterror.ErrInvalidInput, active.WorkflowRunID, active.Status)
	}
	agentModel := server.latestAgentSessionModel(ctx, missionID, executorName)
	agentReasoningEffort := server.latestAgentReasoningEffort(ctx, missionID, executorName)
	pendingEvent, err := server.reportRunner().StartDesign(ctx, missionID, reportexecution.DesignRequest{
		SourceArtifactID:     sourceArtifact.ArtifactID,
		SourceMediaType:      sourceArtifact.MediaType,
		Title:                reportArtifactTitle(sourceArtifact),
		AgentExecutor:        executorName,
		AgentModel:           agentModel,
		AgentReasoningEffort: agentReasoningEffort,
		RendererVersion:      designedReportRendererVersion,
	}, ledger.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		return nil, false, err
	}
	return map[string]any{
		"pending_event":   pendingEvent,
		"source_artifact": rawArtifactMetadata(sourceArtifact),
		"status":          "pending",
	}, true, nil
}

func (server *Server) startReportDraft(ctx context.Context, missionID string, req reportDraftRequest) (map[string]any, error) {
	pipelineFamily, err := reportexecution.NormalizePipelineFamily(req.PipelineFamily)
	if err != nil {
		return nil, err
	}
	req.PipelineFamily = pipelineFamily
	experimental := pipelineFamily == reportilcontract.PipelineFamily
	unverified := pipelineFamily == reportpipeline.Unverified
	independent := experimental || unverified
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Mission report"
	}
	var executorName, mcpMode, reportMode, executionStrategy string
	var rigor reportRigorProfile
	var guidanceProfile, guidanceSHA, postReportHumanize string
	if req.OutputKind == reportexecution.OutputKindArticle {
		if req.PipelineFamily == reportilcontract.PipelineFamily && req.ReportMode == reportModeLongForm {
			req.ExecutionStrategy, err = normalizeReportExecutionStrategy(req.ExecutionStrategy, reportModeLongForm)
			if err != nil {
				return nil, err
			}
		} else {
			req.ReportMode = reportModeOneTake
			req.ExecutionStrategy = reportExecutionStrategySerial
		}
		req.PostReportHumanize = "disabled"
	}
	if independent {
		// Independent families do not use classic model, session, capability, or
		// authoring resolution. Experimental IL preserves planned versus long-form
		// authoring while keeping classic section execution unavailable.
		executorName = "codex"
		mcpMode = "source_read_only"
		if experimental && req.ReportMode == reportModeLongForm {
			reportMode = reportModeLongForm
		} else {
			reportMode = reportModePlanned
		}
		executionStrategy = ""
		postReportHumanize = "disabled"
		if unverified {
			rigor = reportRigorProfile{level: "unverified", label: "무검증형"}
		} else {
			normalized := reportexecution.NormalizeDraftRequest(reportexecution.DraftRequest{
				PipelineFamily: req.PipelineFamily,
				RigorLevel:     req.RigorLevel,
			})
			rigor = reportRigorProfile{level: normalized.RigorLevel, label: normalized.RigorLabel}
		}
	} else {
		executorName, err = normalizeAgentExecutorName(req.AgentExecutor)
		if err != nil {
			return nil, err
		}
		mcpMode, err = normalizeMCPMode(req.MCPMode)
		if err != nil {
			return nil, err
		}
		rigor, err = normalizeReportRigorProfile(req.RigorLevel)
		if err != nil {
			return nil, err
		}
		reportMode, err = normalizeReportMode(req.ReportMode)
		if err != nil {
			return nil, err
		}
		executionStrategy, err = normalizeReportExecutionStrategy(req.ExecutionStrategy, reportMode)
		if err != nil {
			return nil, err
		}
		guidanceProfile, guidanceSHA, err = reportprompt.SelectReportGenerationGuidanceForMode(reportMode, req.GenerationGuidanceProfile)
		if err != nil {
			return nil, err
		}
		postReportHumanize = reportprompt.NormalizePostReportHumanize(req.PostReportHumanize)
	}
	req.Title = title
	req.AgentExecutor = executorName
	req.MCPMode = mcpMode
	req.RigorLevel = rigor.level
	req.ReportMode = reportMode
	req.ExecutionStrategy = executionStrategy
	if independent {
		req.AgentModel = strings.TrimSpace(req.AgentModel)
		req.AgentReasoningEffort = strings.TrimSpace(req.AgentReasoningEffort)
		req.ReportSessionPolicy = reportSessionPolicyFreshSession
		req.GenerationGuidanceProfile = ""
		req.GenerationGuidanceSHA256 = ""
		if unverified {
			req.AgentSelectionSource = "unverified_fixed"
			req.ReportSessionPolicySelection = "unverified_fixed"
		} else {
			req.AgentSelectionSource = "experimental_fixed"
			req.ReportSessionPolicySelection = "experimental_fixed"
		}
	} else {
		req.AgentModel = strings.TrimSpace(req.AgentModel)
		req.AgentReasoningEffort = strings.TrimSpace(req.AgentReasoningEffort)
	}
	req.PostReportHumanize = postReportHumanize

	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	unlockTurns := server.turns.lock(missionID)
	defer unlockTurns()
	if !independent {
		if err := server.validateMissionAgentExecutor(ctx, missionID, executorName); err != nil {
			return nil, err
		}
	}
	if err := server.reconcileStaleAgentTurn(ctx, missionID); err != nil {
		return nil, err
	}
	if server.hasOpenReportDraft(ctx, missionID) {
		return nil, errReportDraftRunning
	}
	if server.hasOpenAgentTurn(ctx, missionID) {
		return nil, fmt.Errorf("%w: agent turn is already running for this mission", producterror.ErrInvalidInput)
	}
	if active := server.activeWorkflowRun(ctx, missionID); active != nil {
		return nil, fmt.Errorf("%w: workflow %s is %s for this mission", producterror.ErrInvalidInput, active.WorkflowRunID, active.Status)
	}
	if !unverified {
		selection, err := server.resolveReportModelSelection(ctx, missionID, req)
		if err != nil {
			return nil, err
		}
		req.AgentModel = selection.Model
		req.AgentReasoningEffort = selection.ReasoningEffort
		req.AgentSelectionSource = selection.Source
	}
	executor := server.agentExecutor(executorName)
	if independent {
		req.ReportSessionPolicy = reportSessionPolicyFreshSession
	} else {
		reportSessionPolicy, reportSessionPolicySelection, err := server.selectReportSessionPolicy(ctx, missionID, executorName, reportMode, strings.TrimSpace(req.ReportSessionPolicy), executor)
		if err != nil {
			return nil, err
		}
		req.ReportSessionPolicy = reportSessionPolicy
		req.ReportSessionPolicySelection = reportSessionPolicySelection
	}
	pendingEvent, err := server.reportRunner().StartDraft(ctx, missionID, reportexecution.DraftRequest{
		Title:                        title,
		DirectionHint:                req.DirectionHint,
		ExecutionStrategy:            storedReportExecutionStrategy(req.ExecutionStrategy),
		AgentExecutor:                executorName,
		AgentModel:                   req.AgentModel,
		AgentReasoningEffort:         req.AgentReasoningEffort,
		AgentSelectionSource:         req.AgentSelectionSource,
		MCPMode:                      mcpMode,
		RigorLevel:                   rigor.level,
		RigorLabel:                   rigor.label,
		ReportMode:                   reportMode,
		PipelineFamily:               req.PipelineFamily,
		ReportSessionPolicy:          req.ReportSessionPolicy,
		ReportSessionPolicySelection: req.ReportSessionPolicySelection,
		PostReportHumanize:           postReportHumanize,
		GenerationGuidanceProfile:    guidanceProfile,
		GenerationGuidanceSHA256:     guidanceSHA,
		OutputKind:                   req.OutputKind,
		ArticleIntent:                req.ArticleIntent,
	}, ledger.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"pending_event": pendingEvent,
		"status":        "pending",
	}, nil
}

func (server *Server) startReportPatch(ctx context.Context, missionID string, req reportPatchRequest) (map[string]any, error) {
	baseArtifactID := strings.TrimSpace(req.BaseArtifactID)
	if baseArtifactID == "" {
		return nil, fmt.Errorf("%w: base report artifact is required", producterror.ErrInvalidInput)
	}
	instruction := strings.TrimSpace(req.Instruction)
	if instruction == "" {
		return nil, fmt.Errorf("%w: report patch instruction is required", producterror.ErrInvalidInput)
	}
	baseArtifact, err := server.reportArtifact(ctx, missionID, baseArtifactID)
	if err != nil {
		return nil, err
	}
	if !isMarkdownMediaType(baseArtifact.MediaType) {
		return nil, fmt.Errorf("%w: report patch requires a Markdown report artifact", producterror.ErrInvalidInput)
	}
	info, err := server.reportArtifactSessionInfo(ctx, missionID, baseArtifactID)
	if err != nil {
		return nil, err
	}
	executorName := strings.TrimSpace(req.AgentExecutor)
	if executorName == "" {
		executorName = info.AgentExecutor
	}
	executorName, err = normalizeAgentExecutorName(executorName)
	if err != nil {
		return nil, err
	}
	if baseExecutor := strings.TrimSpace(info.AgentExecutor); baseExecutor != "" && baseExecutor != executorName {
		return nil, fmt.Errorf("%w: report patch must use the original report executor %q", producterror.ErrInvalidInput, baseExecutor)
	}
	mcpMode, err := normalizeMCPMode(req.MCPMode)
	if err != nil {
		return nil, err
	}
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return nil, fmt.Errorf("%w: report patch requires an agent executor", producterror.ErrInvalidInput)
	}
	agentModel := strings.TrimSpace(req.AgentModel)
	if agentModel == "" {
		agentModel = strings.TrimSpace(info.AgentModel)
	}
	agentReasoningEffort := strings.TrimSpace(req.AgentReasoningEffort)
	if agentReasoningEffort == "" {
		agentReasoningEffort = strings.TrimSpace(info.AgentReasoningEffort)
	}
	agentModel, agentReasoningEffort, err = resolveAgentSettings(executorName, agentModel, agentReasoningEffort, strings.TrimSpace(info.ReportSessionID))
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = firstNonEmpty(info.Title+" 수정본", reportArtifactTitle(baseArtifact)+" 수정본", "Patched report")
	}

	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	unlockTurns := server.turns.lock(missionID)
	defer unlockTurns()
	baseArtifact, err = server.reportArtifact(ctx, missionID, baseArtifactID)
	if err != nil {
		return nil, err
	}
	if !isMarkdownMediaType(baseArtifact.MediaType) {
		return nil, fmt.Errorf("%w: report patch requires a Markdown report artifact", producterror.ErrInvalidInput)
	}
	info, err = server.reportArtifactSessionInfo(ctx, missionID, baseArtifactID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.AgentExecutor) == "" {
		executorName = info.AgentExecutor
		executorName, err = normalizeAgentExecutorName(executorName)
		if err != nil {
			return nil, err
		}
	}
	if baseExecutor := strings.TrimSpace(info.AgentExecutor); baseExecutor != "" && baseExecutor != executorName {
		return nil, fmt.Errorf("%w: report patch must use the original report executor %q", producterror.ErrInvalidInput, baseExecutor)
	}
	executor = server.agentExecutor(executorName)
	if executor == nil {
		return nil, fmt.Errorf("%w: report patch requires an agent executor", producterror.ErrInvalidInput)
	}
	agentModel = strings.TrimSpace(req.AgentModel)
	if agentModel == "" {
		agentModel = strings.TrimSpace(info.AgentModel)
	}
	agentReasoningEffort = strings.TrimSpace(req.AgentReasoningEffort)
	if agentReasoningEffort == "" {
		agentReasoningEffort = strings.TrimSpace(info.AgentReasoningEffort)
	}
	agentModel, agentReasoningEffort, err = resolveAgentSettings(executorName, agentModel, agentReasoningEffort, strings.TrimSpace(info.ReportSessionID))
	if err != nil {
		return nil, err
	}
	title = strings.TrimSpace(req.Title)
	if title == "" {
		title = firstNonEmpty(info.Title+" 수정본", reportArtifactTitle(baseArtifact)+" 수정본", "Patched report")
	}
	if err := server.validateMissionAgentExecutor(ctx, missionID, executorName); err != nil {
		return nil, err
	}
	if err := server.reconcileStaleAgentTurn(ctx, missionID); err != nil {
		return nil, err
	}
	if server.hasOpenReportDraft(ctx, missionID) {
		return nil, errReportDraftRunning
	}
	if server.hasOpenAgentTurn(ctx, missionID) {
		return nil, fmt.Errorf("%w: agent turn is already running for this mission", producterror.ErrInvalidInput)
	}
	if active := server.activeWorkflowRun(ctx, missionID); active != nil {
		return nil, fmt.Errorf("%w: workflow %s is %s for this mission", producterror.ErrInvalidInput, active.WorkflowRunID, active.Status)
	}
	selection, err := selectReportPatchSession(ctx, executor, info.ReportSessionID, req.ReportSessionPolicy)
	if err != nil {
		return nil, err
	}
	pendingEvent, err := server.reportRunner().StartPatch(ctx, missionID, reportexecution.PatchRequest{
		BaseArtifactID:               baseArtifact.ArtifactID,
		Instruction:                  instruction,
		Title:                        title,
		AgentExecutor:                executorName,
		AgentModel:                   agentModel,
		AgentReasoningEffort:         agentReasoningEffort,
		MCPMode:                      mcpMode,
		ReportSessionID:              selection.SessionID,
		PreviousAgentSessionID:       selection.PreviousAgentSessionID,
		ForkSourceAgentSessionID:     selection.ForkSourceAgentSessionID,
		ReportSessionPolicy:          selection.ReportSessionPolicy,
		ReportSessionPolicySelection: selection.ReportSessionPolicySelection,
		SessionChainKind:             selection.SessionChainKind,
	}, ledger.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"pending_event": pendingEvent,
		"status":        "pending",
	}, nil
}

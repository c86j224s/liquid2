package web

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"strings"
	"time"
)

func (server *Server) reportRunner() reportexecution.Runner {
	return reportexecution.Runner{
		Service:  server.service,
		InFlight: &server.runningReports,
		NewID:    newID,
		GenerateExperimental: func(ctx context.Context, missionID string, req reportexecution.DraftRequest, pendingEventID string) error {
			return server.createExperimentalReportDraft(ctx, missionID, req, pendingEventID)
		},
		GenerateUnverified: func(ctx context.Context, missionID string, req reportexecution.DraftRequest, pendingEventID string) error {
			return server.createUnverifiedReportDraft(ctx, missionID, req, pendingEventID)
		},
		GenerateDraft: func(ctx context.Context, missionID string, req reportexecution.DraftRequest, pendingEventID string) error {
			_, err := server.createReportDraft(ctx, missionID, reportDraftRequest{
				Title:                        req.Title,
				DirectionHint:                req.DirectionHint,
				ExecutionStrategy:            req.ExecutionStrategy,
				AgentExecutor:                req.AgentExecutor,
				AgentModel:                   req.AgentModel,
				AgentReasoningEffort:         req.AgentReasoningEffort,
				AgentSelectionSource:         req.AgentSelectionSource,
				MCPMode:                      req.MCPMode,
				RigorLevel:                   req.RigorLevel,
				ReportMode:                   req.ReportMode,
				PipelineFamily:               req.PipelineFamily,
				ReportSessionPolicy:          req.ReportSessionPolicy,
				ReportSessionPolicySelection: req.ReportSessionPolicySelection,
				PostReportHumanize:           req.PostReportHumanize,
				GenerationGuidanceProfile:    req.GenerationGuidanceProfile,
				GenerationGuidanceSHA256:     req.GenerationGuidanceSHA256,
				OutputKind:                   req.OutputKind,
				ArticleIntent:                req.ArticleIntent,
			}, pendingEventID)
			return err
		},
		GenerateDesign: func(ctx context.Context, missionID string, req reportexecution.DesignRequest, pendingEventID string) error {
			_, err := server.createDesignedReportHTMLExport(ctx, missionID, req.SourceArtifactID, reportDesignRequest{
				AgentExecutor:        req.AgentExecutor,
				AgentModel:           req.AgentModel,
				AgentReasoningEffort: req.AgentReasoningEffort,
			}, pendingEventID)
			return err
		},
		GeneratePatch: func(ctx context.Context, missionID string, req reportexecution.PatchRequest, pendingEventID string) error {
			_, err := server.createReportPatch(ctx, missionID, reportPatchRequest{
				BaseArtifactID:       req.BaseArtifactID,
				Instruction:          req.Instruction,
				Title:                req.Title,
				AgentExecutor:        req.AgentExecutor,
				AgentModel:           req.AgentModel,
				AgentReasoningEffort: req.AgentReasoningEffort,
				MCPMode:              req.MCPMode,
				ReportSessionPolicy:  req.ReportSessionPolicy,
			}, pendingEventID, req)
			return err
		},
	}
}

func (server *Server) reconcileStaleReportDrafts(ctx context.Context, missionID string) error {
	return server.reportRunner().ResumeStaleReportOperations(ctx, missionID, server.reportRecoveryHooks())
}

func (server *Server) createReportDraft(ctx context.Context, missionID string, req reportDraftRequest, pendingEventID string) (map[string]any, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Mission report"
	}
	executorName, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		return nil, err
	}
	mcpMode, err := normalizeMCPMode(req.MCPMode)
	if err != nil {
		return nil, err
	}
	rigor, err := normalizeReportRigorProfile(req.RigorLevel)
	if err != nil {
		return nil, err
	}
	reportMode, err := normalizeReportMode(req.ReportMode)
	if err != nil {
		return nil, err
	}
	pipelineFamily, err := reportexecution.NormalizePipelineFamily(req.PipelineFamily)
	if err != nil {
		return nil, err
	}
	req.PipelineFamily = pipelineFamily
	executionStrategy, err := normalizeReportExecutionStrategy(req.ExecutionStrategy, reportMode)
	if err != nil {
		return nil, err
	}
	reportSessionPolicy, err := normalizeReportSessionPolicy(req.ReportSessionPolicy)
	if err != nil {
		return nil, err
	}
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return nil, fmt.Errorf("%w: report generation requires an agent executor", producterror.ErrInvalidInput)
	}
	agentModel := strings.TrimSpace(req.AgentModel)
	agentReasoningEffort := strings.TrimSpace(req.AgentReasoningEffort)
	if strings.TrimSpace(req.AgentSelectionSource) == "" {
		agentModel, agentReasoningEffort, err = resolveAgentSettings(executorName, agentModel, agentReasoningEffort, server.latestAgentSessionID(ctx, missionID, executorName))
		if err != nil {
			return nil, err
		}
	}
	if err := server.validateReportSessionPolicy(ctx, missionID, executorName, reportMode, reportSessionPolicy, executor, false); err != nil {
		return nil, err
	}
	latestSession := server.latestAgentSession(ctx, missionID, executorName)
	profile, err := server.agentCapabilityProfileForSession(ctx, missionID, executorName, latestSession.SessionID)
	if err != nil {
		return nil, err
	}
	executor = agentExecutorWithCapabilityProfile(executor, profile)
	postReportHumanize := reportprompt.NormalizePostReportHumanize(req.PostReportHumanize)
	guidanceProfile, guidanceSHA, err := reportprompt.SelectReportGenerationGuidanceForMode(reportMode, req.GenerationGuidanceProfile)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GenerationGuidanceSHA256) != "" {
		guidanceSHA = strings.TrimSpace(req.GenerationGuidanceSHA256)
	}
	if req.OutputKind == reportexecution.OutputKindArticle {
		return server.createReportWorkflowDraft(ctx, missionID, title, req.DirectionHint, executorName, agentModel, agentReasoningEffort, req.AgentSelectionSource, mcpMode, rigor, reportSessionPolicy, req.ReportSessionPolicySelection, "disabled", guidanceProfile, guidanceSHA, pendingEventID, reportModeOneTake, reportExecutionStrategySerial, executor, req.OutputKind, req.ArticleIntent)
	}
	switch reportMode {
	case reportModeLongForm:
		if executionStrategy == reportExecutionStrategySectionFanout {
			return server.createSectionFanoutLongFormReportDraft(ctx, missionID, title, req.DirectionHint, executorName, agentModel, agentReasoningEffort, req.AgentSelectionSource, mcpMode, rigor, reportSessionPolicy, req.ReportSessionPolicySelection, postReportHumanize, guidanceProfile, guidanceSHA, pendingEventID, executor)
		}
		return server.createSectionalLongFormReportDraft(ctx, missionID, title, req.DirectionHint, executorName, agentModel, agentReasoningEffort, req.AgentSelectionSource, mcpMode, rigor, reportSessionPolicy, req.ReportSessionPolicySelection, postReportHumanize, guidanceProfile, guidanceSHA, pendingEventID, executor)
	}
	classicMode := reportModeOneTake
	if reportMode == reportModePlanned {
		classicMode = reportModePlanned
	}
	return server.createReportWorkflowDraft(ctx, missionID, title, req.DirectionHint, executorName, agentModel, agentReasoningEffort, req.AgentSelectionSource, mcpMode, rigor, reportSessionPolicy, req.ReportSessionPolicySelection, postReportHumanize, guidanceProfile, guidanceSHA, pendingEventID, classicMode, reportExecutionStrategySerial, executor, "", reportexecution.ArticleIntent{})
}

func (server *Server) createReportPatch(ctx context.Context, missionID string, req reportPatchRequest, pendingEventID string, patchReq reportexecution.PatchRequest) (map[string]any, error) {
	baseArtifact, err := server.reportArtifact(ctx, missionID, req.BaseArtifactID)
	if err != nil {
		return nil, err
	}
	if !isMarkdownMediaType(baseArtifact.MediaType) {
		return nil, fmt.Errorf("%w: report patch requires a Markdown report artifact", producterror.ErrInvalidInput)
	}
	instruction := strings.TrimSpace(req.Instruction)
	if instruction == "" {
		return nil, fmt.Errorf("%w: report patch instruction is required", producterror.ErrInvalidInput)
	}
	executorName, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		return nil, err
	}
	mcpMode, err := normalizeMCPMode(req.MCPMode)
	if err != nil {
		return nil, err
	}
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return nil, fmt.Errorf("%w: report patch requires an agent executor", producterror.ErrInvalidInput)
	}
	reportSessionID := strings.TrimSpace(patchReq.ReportSessionID)
	if reportSessionID == "" {
		return nil, fmt.Errorf("%w: report patch requires a report session", producterror.ErrInvalidInput)
	}
	profile, err := server.agentCapabilityProfileForSession(ctx, missionID, executorName, reportSessionID)
	if err != nil {
		return nil, err
	}
	executor = agentExecutorWithCapabilityProfile(executor, profile)
	title := firstNonEmpty(req.Title, patchReq.Title, reportArtifactTitle(baseArtifact)+" 수정본")
	agentModel := strings.TrimSpace(req.AgentModel)
	if agentModel == "" {
		agentModel = strings.TrimSpace(patchReq.AgentModel)
	}
	agentReasoningEffort := strings.TrimSpace(req.AgentReasoningEffort)
	if agentReasoningEffort == "" {
		agentReasoningEffort = strings.TrimSpace(patchReq.AgentReasoningEffort)
	}
	toolSessionID := newID("ses")
	started := time.Now()
	result, err := executor.Run(ctx, AgentRequest{
		UserText:          "patch markdown report artifact with MCP",
		Prompt:            agentReportPatchPrompt(title, missionID, toolSessionID, pendingEventID, baseArtifact.ArtifactID, instruction, patchReq),
		Model:             agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		PreviousSessionID: reportSessionID,
		AgentExecutor:     executorName,
		MCPMode:           mcpMode,
		ExtraMCPTools:     reportPatchMCPTools(),
		ReplaceMCPTools:   true,
		ReportPatch: &AgentReportPatchContext{
			BaseArtifactID:               baseArtifact.ArtifactID,
			PendingEventID:               pendingEventID,
			AgentExecutor:                executorName,
			AgentModel:                   agentModel,
			AgentReasoningEffort:         agentReasoningEffort,
			MCPMode:                      mcpMode,
			AgentSessionID:               reportSessionID,
			PreviousAgentSessionID:       patchReq.PreviousAgentSessionID,
			ReturnedAgentSessionID:       reportSessionID,
			ReportSessionID:              reportSessionID,
			ForkSourceAgentSessionID:     patchReq.ForkSourceAgentSessionID,
			ReportSessionPolicy:          patchReq.ReportSessionPolicy,
			ReportSessionPolicySelection: patchReq.ReportSessionPolicySelection,
			SessionChainKind:             patchReq.SessionChainKind,
		},
	})
	durationMS := time.Since(started).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("report patch agent failed: %w", reportAgentFailure(err, result, "report_patch", durationMS, reportSessionID))
	}
	validated, err := conversation.ValidateSameSessionResult(result, reportSessionID)
	if err != nil {
		return nil, reportAgentFailure(err, result, "report_patch", durationMS, reportSessionID)
	}
	if _, ok, err := server.reportArtifactEventForPending(ctx, missionID, pendingEventID); err != nil {
		return nil, err
	} else if ok {
		return map[string]any{
			"status":           "completed",
			"agent_session_id": validated.SessionID,
		}, nil
	}
	finalizedEvent, ok, err := server.reportPatchFinalizedEventForPending(ctx, missionID, pendingEventID)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, reportAgentFailure(fmt.Errorf("%w: report patch agent did not finalize through MCP", producterror.ErrInvalidInput), result, "report_patch", durationMS, reportSessionID)
	}
	if _, err := server.promoteReportPatchFinalizedArtifact(ctx, missionID, finalizedEvent); err != nil {
		return nil, err
	}
	return map[string]any{
		"status":           "completed",
		"agent_session_id": validated.SessionID,
	}, nil
}

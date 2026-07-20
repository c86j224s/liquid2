package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/web"
	workflowruntime "github.com/c86j224s/liquid2/plasma/internal/workflow"
)

type cliReportDraftRunResult struct {
	Artifact  app.RawArtifact
	Event     app.LedgerEvent
	Humanized web.ReportHumanizeResult
	SessionID string
	Err       error
}

type cliReportPatchRunResult struct {
	Artifact  app.RawArtifact
	Event     app.LedgerEvent
	SessionID string
	Err       error
}

type cliReportArtifactInfo struct {
	Title                string
	AgentExecutor        string
	AgentSessionID       string
	PreviousSessionID    string
	ReportSessionID      string
	AgentModel           string
	AgentReasoningEffort string
}

func cliReportArtifactSessionInfo(ctx context.Context, svc *app.Service, missionID string, artifactID string) (cliReportArtifactInfo, error) {
	artifact, err := svc.GetRawArtifact(ctx, artifactID)
	if err != nil {
		return cliReportArtifactInfo{}, err
	}
	if artifact.MissionID != missionID {
		return cliReportArtifactInfo{}, fmt.Errorf("%w: artifact belongs to another mission", app.ErrInvalidInput)
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(artifact.MediaType)), "text/markdown") {
		return cliReportArtifactInfo{}, fmt.Errorf("%w: report patch requires a Markdown report artifact", app.ErrInvalidInput)
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		return cliReportArtifactInfo{}, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "report.artifact.created" && event.EventType != "report.artifact.exported" {
			continue
		}
		var payload struct {
			ArtifactID           string `json:"artifact_id"`
			Title                string `json:"title"`
			AgentExecutor        string `json:"agent_executor"`
			AgentSessionID       string `json:"agent_session_id"`
			PreviousSessionID    string `json:"previous_agent_session_id"`
			ReportSessionID      string `json:"report_session_id"`
			AgentModel           string `json:"agent_model"`
			AgentReasoningEffort string `json:"agent_reasoning_effort"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.ArtifactID) != artifactID {
			continue
		}
		info := cliReportArtifactInfo{
			Title:                strings.TrimSpace(payload.Title),
			AgentExecutor:        firstNonEmptyString(strings.TrimSpace(payload.AgentExecutor), "codex"),
			AgentSessionID:       strings.TrimSpace(payload.AgentSessionID),
			PreviousSessionID:    strings.TrimSpace(payload.PreviousSessionID),
			ReportSessionID:      strings.TrimSpace(payload.ReportSessionID),
			AgentModel:           strings.TrimSpace(payload.AgentModel),
			AgentReasoningEffort: strings.TrimSpace(payload.AgentReasoningEffort),
		}
		info.ReportSessionID = firstNonEmptyString(info.ReportSessionID, info.AgentSessionID, info.PreviousSessionID)
		return info, nil
	}
	return cliReportArtifactInfo{}, fmt.Errorf("%w: report artifact event not found", app.ErrInvalidInput)
}

func createCLIReportPatchArtifact(ctx context.Context, svc *app.Service, executor web.AgentExecutor, missionID string, pendingEventID string, req reporting.PatchRequest) cliReportPatchRunResult {
	toolSessionID := cliNewID("ses")
	result, err := executor.Run(ctx, web.AgentRequest{
		UserText:          "patch markdown report artifact with MCP",
		Prompt:            web.AgentReportPatchPrompt(req.Title, missionID, toolSessionID, pendingEventID, req.BaseArtifactID, req.Instruction, req),
		Model:             req.AgentModel,
		ReasoningEffort:   req.AgentReasoningEffort,
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		PreviousSessionID: req.ReportSessionID,
		AgentExecutor:     req.AgentExecutor,
		MCPMode:           req.MCPMode,
		ExtraMCPTools: []string{
			mcp.ToolReportPatchStart,
			mcp.ToolReportPatchRead,
			mcp.ToolReportPatchApply,
			mcp.ToolReportPatchFinalize,
		},
		ReplaceMCPTools: true,
		ReportPatch: &web.AgentReportPatchContext{
			BaseArtifactID:               req.BaseArtifactID,
			PendingEventID:               pendingEventID,
			AgentExecutor:                req.AgentExecutor,
			AgentModel:                   req.AgentModel,
			AgentReasoningEffort:         req.AgentReasoningEffort,
			MCPMode:                      req.MCPMode,
			AgentSessionID:               req.ReportSessionID,
			PreviousAgentSessionID:       req.PreviousAgentSessionID,
			ReturnedAgentSessionID:       req.ReportSessionID,
			ReportSessionID:              req.ReportSessionID,
			ForkSourceAgentSessionID:     req.ForkSourceAgentSessionID,
			ReportSessionPolicy:          req.ReportSessionPolicy,
			ReportSessionPolicySelection: req.ReportSessionPolicySelection,
			SessionChainKind:             req.SessionChainKind,
		},
	})
	if err != nil {
		return cliReportPatchRunResult{Err: fmt.Errorf("report patch agent failed: %w", err)}
	}
	sessionID, err := cliValidatedSessionID(result.SessionID, req.ReportSessionID)
	if err != nil {
		return cliReportPatchRunResult{Err: err}
	}
	if event, artifact, err := cliReportArtifactForPending(ctx, svc, missionID, pendingEventID); err != nil {
		return cliReportPatchRunResult{Err: err}
	} else if event.EventID != "" {
		return cliReportPatchRunResult{Artifact: artifact, Event: event, SessionID: sessionID}
	}
	finalizedEvent, _, err := cliReportPatchFinalizedForPending(ctx, svc, missionID, pendingEventID)
	if err != nil {
		return cliReportPatchRunResult{Err: err}
	}
	event, artifact, err := cliPromoteReportPatchFinalizedArtifact(ctx, svc, missionID, finalizedEvent)
	if err != nil {
		return cliReportPatchRunResult{Err: err}
	}
	return cliReportPatchRunResult{Artifact: artifact, Event: event, SessionID: sessionID}
}

func cliReportArtifactForPending(ctx context.Context, svc *app.Service, missionID string, pendingEventID string) (app.LedgerEvent, app.RawArtifact, error) {
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		return app.LedgerEvent{}, app.RawArtifact{}, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "report.artifact.created" {
			continue
		}
		var payload struct {
			PendingEventID string `json:"pending_event_id"`
			ArtifactID     string `json:"artifact_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.PendingEventID) != strings.TrimSpace(pendingEventID) {
			continue
		}
		artifact, err := svc.GetRawArtifact(ctx, strings.TrimSpace(payload.ArtifactID))
		if err != nil {
			return app.LedgerEvent{}, app.RawArtifact{}, err
		}
		return event, artifact, nil
	}
	return app.LedgerEvent{}, app.RawArtifact{}, nil
}

func cliReportPatchFinalizedForPending(ctx context.Context, svc *app.Service, missionID string, pendingEventID string) (app.LedgerEvent, app.RawArtifact, error) {
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		return app.LedgerEvent{}, app.RawArtifact{}, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "report.patch.finalized" {
			continue
		}
		var payload struct {
			PendingEventID string `json:"pending_event_id"`
			ArtifactID     string `json:"artifact_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.PendingEventID) != strings.TrimSpace(pendingEventID) {
			continue
		}
		artifactID := strings.TrimSpace(payload.ArtifactID)
		if artifactID == "" {
			continue
		}
		artifact, err := svc.GetRawArtifact(ctx, artifactID)
		if err != nil {
			return app.LedgerEvent{}, app.RawArtifact{}, err
		}
		if artifact.MissionID != missionID {
			return app.LedgerEvent{}, app.RawArtifact{}, fmt.Errorf("%w: finalized report artifact belongs to another mission", app.ErrInvalidInput)
		}
		return event, artifact, nil
	}
	return app.LedgerEvent{}, app.RawArtifact{}, fmt.Errorf("%w: report patch agent did not finalize through MCP", app.ErrInvalidInput)
}

func cliPromoteReportPatchFinalizedArtifact(ctx context.Context, svc *app.Service, missionID string, finalized app.LedgerEvent) (app.LedgerEvent, app.RawArtifact, error) {
	var payload map[string]any
	if err := json.Unmarshal(finalized.Payload, &payload); err != nil {
		return app.LedgerEvent{}, app.RawArtifact{}, fmt.Errorf("%w: invalid report patch finalized payload", app.ErrInvalidInput)
	}
	pendingEventID, _ := payload["pending_event_id"].(string)
	pendingEventID = strings.TrimSpace(pendingEventID)
	if pendingEventID == "" {
		return app.LedgerEvent{}, app.RawArtifact{}, fmt.Errorf("%w: report patch finalized payload is missing pending_event_id", app.ErrInvalidInput)
	}
	if event, artifact, err := cliReportArtifactForPending(ctx, svc, missionID, pendingEventID); err != nil {
		return app.LedgerEvent{}, app.RawArtifact{}, err
	} else if event.EventID != "" {
		return event, artifact, nil
	}
	artifactID, _ := payload["artifact_id"].(string)
	artifact, err := svc.GetRawArtifact(ctx, strings.TrimSpace(artifactID))
	if err != nil {
		return app.LedgerEvent{}, app.RawArtifact{}, err
	}
	if artifact.MissionID != missionID {
		return app.LedgerEvent{}, app.RawArtifact{}, fmt.Errorf("%w: finalized report artifact belongs to another mission", app.ErrInvalidInput)
	}
	payload["kind"] = "markdown_report_artifact"
	payload["promoted_from_event_id"] = finalized.EventID
	producerID, _ := payload["report_session_id"].(string)
	producerID = firstNonEmptyString(strings.TrimSpace(producerID), strings.TrimSpace(finalized.CorrelationID))
	event, err := svc.AppendEvent(ctx, reporting.BuildPromotedMarkdownReportArtifactAppendRequest(reporting.PromotedMarkdownReportArtifactEventRequest{
		EventID:             cliNewID("evt"),
		MissionID:           missionID,
		PromotedFromEventID: finalized.EventID,
		Payload:             payload,
		Producer:            app.Producer{Type: "agent_session", ID: producerID},
	}))
	if err != nil {
		return app.LedgerEvent{}, app.RawArtifact{}, err
	}
	return event, artifact, nil
}

func createCLIReportDraftArtifact(ctx context.Context, svc *app.Service, executor web.AgentExecutor, missionID string, pendingEventID string, req reporting.DraftRequest) cliReportDraftRunResult {
	reportTitle, directionHint, agentName := req.Title, req.DirectionHint, req.AgentExecutor
	mcpMode, reportMode := req.MCPMode, req.ReportMode
	reportSessionPolicy, reportSessionPolicySelection := req.ReportSessionPolicy, req.ReportSessionPolicySelection
	postReportHumanize := req.PostReportHumanize
	generationGuidanceProfile, generationGuidanceSHA256 := req.GenerationGuidanceProfile, req.GenerationGuidanceSHA256
	postReportHumanize = cliNormalizePostReportHumanize(postReportHumanize)
	generationGuidanceProfile = strings.TrimSpace(generationGuidanceProfile)
	generationGuidanceSHA256 = strings.TrimSpace(generationGuidanceSHA256)
	events, _ := svc.ListEvents(ctx, missionID)
	preReportResearchSessionID := workflowruntime.LatestAgentSessionID(events, strings.TrimSpace(agentName))
	previousSessionID := preReportResearchSessionID
	forkSourceSessionID := ""
	sessionChainKind := "same_session_report"
	if reportSessionPolicy == reporting.SessionPolicyIsolatedFork {
		forker, ok := executor.(web.AgentSessionForker)
		if !ok {
			return cliReportDraftRunResult{Err: reporting.ValidateSessionPolicy(reportSessionPolicy, reportMode, false, strings.TrimSpace(preReportResearchSessionID) != "", false)}
		}
		if strings.TrimSpace(preReportResearchSessionID) == "" {
			return cliReportDraftRunResult{Err: reporting.ValidateSessionPolicy(reportSessionPolicy, reportMode, true, false, false)}
		}
		fork, err := forker.ForkSession(ctx, preReportResearchSessionID)
		if err != nil {
			return cliReportDraftRunResult{Err: fmt.Errorf("report session fork: %w", err)}
		}
		previousSessionID = fork.SessionID
		forkSourceSessionID = fork.SourceSessionID
		if strings.TrimSpace(forkSourceSessionID) == "" {
			forkSourceSessionID = preReportResearchSessionID
		}
		sessionChainKind = "isolated_fork_report"
	}
	toolSessionID := cliNewID("ses")
	started := time.Now()
	planEventID := ""
	planToolSessionID := ""
	reportPlanSessionID := ""
	if reportMode != reporting.ModeOneTake {
		planToolSessionID = toolSessionID
		planPreviousSessionID := previousSessionID
		planResult, err := executor.Run(ctx, web.AgentRequest{
			UserText:          "plan markdown report artifact",
			Prompt:            cliPromptWithDirection(cliReportPlanPrompt(reportTitle, missionID, planToolSessionID, generationGuidanceProfile), directionHint),
			MissionID:         missionID,
			ToolSessionID:     planToolSessionID,
			PreviousSessionID: planPreviousSessionID,
			AgentExecutor:     strings.TrimSpace(agentName),
			Model:             req.AgentModel,
			ReasoningEffort:   req.AgentReasoningEffort,
			MCPMode:           strings.TrimSpace(mcpMode),
		})
		if err != nil {
			return cliReportDraftRunResult{Err: fmt.Errorf("report plan agent: %w", err)}
		}
		sessionID, err := cliValidatedSessionID(planResult.SessionID, planPreviousSessionID)
		if err != nil {
			return cliReportDraftRunResult{Err: fmt.Errorf("report plan session: %w", err)}
		}
		previousSessionID = sessionID
		reportPlanSessionID = sessionID
		planEvent, err := svc.AppendEvent(ctx, reporting.BuildCLIMarkdownReportPlanCreatedAppendRequest(reporting.CLIMarkdownReportPlanCreatedEventRequest{
			EventID:                      cliNewID("evt"),
			MissionID:                    missionID,
			PendingEventID:               pendingEventID,
			Title:                        reportTitle,
			AgentExecutor:                strings.TrimSpace(agentName),
			AgentModel:                   req.AgentModel,
			AgentReasoningEffort:         req.AgentReasoningEffort,
			AgentSelectionSource:         req.AgentSelectionSource,
			AgentSessionID:               sessionID,
			PreviousAgentSessionID:       planPreviousSessionID,
			ToolSessionID:                planToolSessionID,
			MCPMode:                      strings.TrimSpace(mcpMode),
			ReportMode:                   reportMode,
			ReportSessionPolicy:          reportSessionPolicy,
			ReportSessionPolicySelection: reportSessionPolicySelection,
			PostReportHumanize:           postReportHumanize,
			HumanizeEnabled:              postReportHumanize != "disabled",
			GenerationGuidanceProfile:    generationGuidanceProfile,
			GenerationGuidanceSHA256:     generationGuidanceSHA256,
			SessionChainKind:             sessionChainKind,
			PreReportResearchSessionID:   preReportResearchSessionID,
			ReportPlanSessionID:          sessionID,
			ForkSourceAgentSessionID:     forkSourceSessionID,
			CompositionStrategy:          cliReportCompositionStrategy(reportMode),
			PlanText:                     planResult.Text,
			Producer:                     app.Producer{Type: "agent_session", ID: firstNonEmptyString(sessionID, planToolSessionID)},
		}))
		if err != nil {
			return cliReportDraftRunResult{Err: fmt.Errorf("append report.plan.created: %w", err)}
		}
		planEventID = planEvent.EventID
		toolSessionID = cliNewID("ses")
	}
	reportPreviousSessionID := previousSessionID
	result, err := executor.Run(ctx, web.AgentRequest{
		UserText:          "generate markdown report artifact",
		Prompt:            cliPromptWithDirection(cliReportPrompt(reportTitle, missionID, toolSessionID, reportMode, planEventID, generationGuidanceProfile), directionHint),
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		PreviousSessionID: reportPreviousSessionID,
		AgentExecutor:     strings.TrimSpace(agentName),
		Model:             req.AgentModel,
		ReasoningEffort:   req.AgentReasoningEffort,
		MCPMode:           strings.TrimSpace(mcpMode),
	})
	if err != nil {
		return cliReportDraftRunResult{Err: fmt.Errorf("report agent: %w", err)}
	}
	sessionID, err := cliValidatedSessionID(result.SessionID, reportPreviousSessionID)
	if err != nil {
		return cliReportDraftRunResult{Err: fmt.Errorf("report session: %w", err)}
	}
	markdown := strings.TrimSpace(result.Text)
	if markdown == "" {
		return cliReportDraftRunResult{Err: fmt.Errorf("report agent returned empty Markdown")}
	}
	artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: cliNewID("art"),
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   safeCLIReportFilename(reportTitle),
		Producer:   app.Producer{Type: "agent_session", ID: firstNonEmptyString(sessionID, toolSessionID)},
		Content:    []byte(markdown),
	})
	if err != nil {
		return cliReportDraftRunResult{Err: fmt.Errorf("create report artifact: %w", err)}
	}
	event, err := svc.AppendEvent(ctx, reporting.BuildCLIMarkdownReportArtifactCreatedAppendRequest(reporting.CLIMarkdownReportArtifactCreatedEventRequest{
		EventID:                      cliNewID("evt"),
		MissionID:                    missionID,
		PendingEventID:               pendingEventID,
		Title:                        reportTitle,
		Artifact:                     artifact,
		AgentExecutor:                strings.TrimSpace(agentName),
		AgentModel:                   req.AgentModel,
		AgentReasoningEffort:         req.AgentReasoningEffort,
		AgentSelectionSource:         req.AgentSelectionSource,
		AgentSessionID:               sessionID,
		PreviousAgentSessionID:       reportPreviousSessionID,
		ToolSessionID:                toolSessionID,
		MCPMode:                      strings.TrimSpace(mcpMode),
		ReportMode:                   reportMode,
		ReportSessionPolicy:          reportSessionPolicy,
		ReportSessionPolicySelection: reportSessionPolicySelection,
		PostReportHumanize:           postReportHumanize,
		HumanizeEnabled:              postReportHumanize != "disabled",
		GenerationGuidanceProfile:    generationGuidanceProfile,
		GenerationGuidanceSHA256:     generationGuidanceSHA256,
		SessionChainKind:             sessionChainKind,
		PreReportResearchSessionID:   preReportResearchSessionID,
		ReportPlanSessionID:          reportPlanSessionID,
		ReportSessionID:              sessionID,
		ForkSourceAgentSessionID:     forkSourceSessionID,
		CompositionStrategy:          cliReportCompositionStrategy(reportMode),
		PlanEventID:                  planEventID,
		PlanToolSessionID:            planToolSessionID,
		DurationMS:                   time.Since(started).Milliseconds(),
		Producer:                     app.Producer{Type: "agent_session", ID: firstNonEmptyString(sessionID, toolSessionID)},
	}))
	if err != nil {
		return cliReportDraftRunResult{Err: fmt.Errorf("append report.artifact.created: %w", err)}
	}
	if postReportHumanize == "disabled" {
		return cliReportDraftRunResult{Artifact: artifact, Event: event, SessionID: sessionID}
	}
	humanized, err := web.HumanizeMarkdownReport(ctx, svc, cliNewID, missionID, web.ReportHumanizeInput{
		Title:             reportTitle,
		Markdown:          markdown,
		SourceArtifact:    artifact,
		ExecutorName:      strings.TrimSpace(agentName),
		MCPMode:           strings.TrimSpace(mcpMode),
		PreviousSessionID: sessionID,
		ReportMode:        reportMode,
		PendingEventID:    pendingEventID,
	}, executor)
	if err != nil {
		return cliReportDraftRunResult{Err: fmt.Errorf("report humanize: %w", err)}
	}
	return cliReportDraftRunResult{Artifact: artifact, Event: event, Humanized: humanized, SessionID: sessionID}
}

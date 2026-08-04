package web

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

type sectionalReportProgress struct {
	hasPlan                      bool
	planEvent                    app.LedgerEvent
	plan                         agentSectionalReportPlan
	artifactID                   string
	currentSessionID             string
	reportSessionPolicy          string
	reportSessionPolicySelection string
	agentExecutor                string
	agentModel                   string
	agentReasoningEffort         string
	agentSelectionSource         string
	mcpMode                      string
	sessionChainKind             string
	preReportResearchSessionID   string
	reportPlanSessionID          string
	forkSourceSessionID          string
	generationGuidanceProfile    string
	generationGuidanceSHA256     string
	partEditEnabled              bool
	partPlanningEnabled          bool
	hasRequirementMap            bool
	hasRequirementStage          bool
	hasPostPlanSectionStarted    bool
	requirementMapEvent          app.LedgerEvent
	requirementMap               reporting.ReportRequirementMap
	partPlans                    map[int]sectionFanoutPartPlan
	sections                     map[sectionalReportIndex]sectionalReportDraft
	parts                        map[int]sectionalReportPartDraft
	editedParts                  map[int]sectionalReportPartDraft
}

type sectionalReportIndex struct {
	part    int
	section int
}

func (server *Server) resumeReportDraftWorker(ctx context.Context, missionID string, pending app.LedgerEvent) error {
	if !reportDraftPendingHasRecoveryContract(pending) {
		return nil
	}
	req, err := reportDraftRequestFromPendingEvent(pending)
	if err != nil {
		_, failErr := server.reportRunner().AppendDraftFailed(ctx, missionID, pending.EventID, reportDraftPendingExecutor(pending), reportDraftPendingMode(pending), err)
		return failErr
	}
	return server.reportRunner().RunDraft(context.Background(), missionID, reportexecution.DraftRequest{
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
		ReportSessionPolicy:          req.ReportSessionPolicy,
		ReportSessionPolicySelection: req.ReportSessionPolicySelection,
		PostReportHumanize:           req.PostReportHumanize,
		GenerationGuidanceProfile:    req.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     req.GenerationGuidanceSHA256,
	}, pending.EventID)
}

func reportDraftPendingHasRecoveryContract(event app.LedgerEvent) bool {
	if event.EventType != "report.draft.pending" {
		return false
	}
	var payload struct {
		Kind              string `json:"kind"`
		AgentExecutor     string `json:"agent_executor"`
		ReportMode        string `json:"report_mode"`
		MCPMode           string `json:"mcp_mode"`
		ExecutionStrategy string `json:"execution_strategy"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return false
	}
	switch strings.TrimSpace(payload.Kind) {
	case "markdown_report_artifact_pending", "report_draft_pending":
		return true
	}
	return strings.TrimSpace(payload.AgentExecutor) != "" &&
		strings.TrimSpace(payload.ReportMode) != ""
}

func reportDraftRequestFromPendingEvent(event app.LedgerEvent) (reportDraftRequest, error) {
	var payload struct {
		Title                        string `json:"title"`
		DirectionHint                string `json:"direction_hint"`
		ExecutionStrategy            string `json:"execution_strategy"`
		AgentExecutor                string `json:"agent_executor"`
		AgentModel                   string `json:"agent_model"`
		AgentReasoningEffort         string `json:"agent_reasoning_effort"`
		AgentSelectionSource         string `json:"agent_selection_source"`
		MCPMode                      string `json:"mcp_mode"`
		RigorLevel                   string `json:"rigor_level"`
		ReportMode                   string `json:"report_mode"`
		ReportSessionPolicy          string `json:"report_session_policy"`
		ReportSessionPolicySelection string `json:"report_session_policy_selection"`
		PostReportHumanize           string `json:"post_report_humanize"`
		GenerationGuidanceProfile    string `json:"generation_guidance_profile"`
		GenerationGuidanceSHA256     string `json:"generation_guidance_sha256"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return reportDraftRequest{}, fmt.Errorf("%w: invalid report pending payload", app.ErrInvalidInput)
	}
	return reportDraftRequest{
		Title:                firstNonEmpty(payload.Title, "Mission report"),
		DirectionHint:        reportexecution.NormalizeDirectionHint(payload.DirectionHint),
		ExecutionStrategy:    strings.TrimSpace(strings.ToLower(payload.ExecutionStrategy)),
		AgentExecutor:        firstNonEmpty(payload.AgentExecutor, "codex"),
		AgentModel:           strings.TrimSpace(payload.AgentModel),
		AgentReasoningEffort: strings.TrimSpace(payload.AgentReasoningEffort),
		AgentSelectionSource: strings.TrimSpace(payload.AgentSelectionSource),
		MCPMode:              firstNonEmpty(payload.MCPMode, "auto"),
		// rigor가 지속 상태로 저장되기 전의 pending event는 과거 balanced 동작을 사용했다.
		// recovery는 새 요청 기본값이 아니라 그 frozen behavior를 재개해야 한다.
		RigorLevel:                   firstNonEmpty(payload.RigorLevel, legacyPendingReportRigorLevel),
		ReportMode:                   firstNonEmpty(payload.ReportMode, defaultReportMode),
		ReportSessionPolicy:          firstNonEmpty(payload.ReportSessionPolicy, reportSessionPolicySameSession),
		ReportSessionPolicySelection: strings.TrimSpace(payload.ReportSessionPolicySelection),
		PostReportHumanize:           strings.TrimSpace(payload.PostReportHumanize),
		// guidance profile이 지속 상태로 저장되기 전의 pending event는 legacy
		// preserve-markdown 경로에 속한다. 재시작 recovery에서 중단된 과거 리포트를
		// 새 기본 profile로 재해석하면 안 된다.
		GenerationGuidanceProfile: firstNonEmpty(payload.GenerationGuidanceProfile, reportprompt.ProfileVisualPlan),
		GenerationGuidanceSHA256:  strings.TrimSpace(payload.GenerationGuidanceSHA256),
	}, nil
}

func (server *Server) loadSectionalReportProgress(ctx context.Context, missionID string, pendingEventID string) (sectionalReportProgress, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return sectionalReportProgress{}, err
	}
	progress := sectionalReportProgress{
		partPlans:   map[int]sectionFanoutPartPlan{},
		sections:    map[sectionalReportIndex]sectionalReportDraft{},
		parts:       map[int]sectionalReportPartDraft{},
		editedParts: map[int]sectionalReportPartDraft{},
	}
	lineage, err := reportRecoveryLineage(events, pendingEventID)
	if err != nil {
		return sectionalReportProgress{}, err
	}
	for _, attemptID := range lineage {
		for _, event := range events {
			if event.EventType == "report.plan.created" {
				if err := server.applySectionalPlanProgress(ctx, attemptID, event, &progress); err != nil {
					return sectionalReportProgress{}, err
				}
			}
		}
	}
	if !progress.hasPlan {
		return progress, nil
	}
	for _, attemptID := range lineage {
		for _, event := range events {
			switch event.EventType {
			case reporting.ReportRequirementsStartedEventType:
				if reportRequirementStageMatches(attemptID, progress.planEvent.EventID, event) {
					progress.hasRequirementStage = true
				}
			case reporting.ReportRequirementsMappedEventType:
				if err := applyReportRequirementMapProgress(attemptID, progress.planEvent.EventID, event, &progress); err != nil {
					return sectionalReportProgress{}, err
				}
			case "report.section.started":
				if reportSectionStartedMatches(attemptID, progress.planEvent.EventID, event) {
					progress.hasPostPlanSectionStarted = true
				}
			}
		}
	}
	for _, attemptID := range lineage {
		for _, event := range events {
			switch event.EventType {
			case reporting.PartPlanCreatedEventType:
				if progress.partPlanningEnabled {
					if err := applyPartPlanProgress(missionID, attemptID, progress.planEvent.EventID, len(progress.plan.Parts), event, &progress); err != nil {
						return sectionalReportProgress{}, err
					}
				}
			case "report.section.created":
				if err := server.applySectionProgress(ctx, attemptID, progress.planEvent.EventID, event, &progress); err != nil {
					return sectionalReportProgress{}, err
				}
			case "report.part.created":
				if err := server.applyPartProgress(ctx, attemptID, progress.planEvent.EventID, event, &progress); err != nil {
					return sectionalReportProgress{}, err
				}
			case reporting.PartEditedEventType:
				if err := server.applyPartEditProgress(ctx, attemptID, progress.planEvent.EventID, event, &progress); err != nil {
					return sectionalReportProgress{}, err
				}
			}
		}
	}
	return progress, nil
}

func reportRecoveryLineage(events []app.LedgerEvent, pendingID string) ([]string, error) {
	type pending struct{ Origin, Parent, Strategy string }
	pendingByID := map[string]pending{}
	for _, event := range events {
		if event.EventType != "report.draft.pending" {
			continue
		}
		var p struct {
			Origin   string `json:"origin_pending_event_id"`
			Parent   string `json:"retry_of_pending_event_id"`
			Strategy string `json:"retry_strategy"`
		}
		if json.Unmarshal(event.Payload, &p) != nil {
			return nil, fmt.Errorf("%w: invalid report attempt", app.ErrInvalidInput)
		}
		if p.Origin == "" {
			p.Origin = event.EventID
		}
		pendingByID[event.EventID] = pending{p.Origin, p.Parent, p.Strategy}
	}
	current, ok := pendingByID[pendingID]
	if !ok {
		return nil, fmt.Errorf("%w: report attempt missing", app.ErrInvalidInput)
	}
	if current.Strategy == "restart" {
		parent, ok := pendingByID[current.Parent]
		if current.Parent == "" || !ok || parent.Origin != current.Origin {
			return nil, fmt.Errorf("%w: invalid report restart lineage", app.ErrInvalidInput)
		}
		return []string{pendingID}, nil
	}
	if current.Parent == "" {
		if current.Origin != pendingID {
			return nil, fmt.Errorf("%w: invalid report root lineage", app.ErrInvalidInput)
		}
		return []string{pendingID}, nil
	}
	chain := []string{}
	seen := map[string]bool{}
	for depth := 0; depth < 64; depth++ {
		if seen[pendingID] {
			return nil, fmt.Errorf("%w: report lineage cycle", app.ErrInvalidInput)
		}
		seen[pendingID] = true
		item, ok := pendingByID[pendingID]
		if !ok {
			return nil, fmt.Errorf("%w: report lineage ancestor missing", app.ErrInvalidInput)
		}
		if item.Origin != current.Origin {
			return nil, fmt.Errorf("%w: report lineage origin mismatch", app.ErrInvalidInput)
		}
		chain = append([]string{pendingID}, chain...)
		if item.Strategy == "restart" {
			return chain, nil
		}
		if item.Parent == "" {
			return chain, nil
		}
		pendingID = item.Parent
	}
	return nil, fmt.Errorf("%w: report lineage too deep", app.ErrInvalidInput)
}

func (server *Server) applySectionalPlanProgress(ctx context.Context, pendingEventID string, event app.LedgerEvent, progress *sectionalReportProgress) error {
	var payload struct {
		PendingEventID               string                   `json:"pending_event_id"`
		ReportMode                   string                   `json:"report_mode"`
		ArtifactID                   string                   `json:"artifact_id"`
		AgentSessionID               string                   `json:"agent_session_id"`
		PreviousAgentSessionID       string                   `json:"previous_agent_session_id"`
		AgentExecutor                string                   `json:"agent_executor"`
		AgentModel                   string                   `json:"agent_model"`
		AgentReasoningEffort         string                   `json:"agent_reasoning_effort"`
		AgentSelectionSource         string                   `json:"agent_selection_source"`
		MCPMode                      string                   `json:"mcp_mode"`
		ReportSessionPolicy          string                   `json:"report_session_policy"`
		ReportSessionPolicySelection string                   `json:"report_session_policy_selection"`
		SessionChainKind             string                   `json:"session_chain_kind"`
		PreReportResearchSessionID   string                   `json:"pre_report_research_session_id"`
		ReportPlanSessionID          string                   `json:"report_plan_session_id"`
		ForkSourceSessionID          string                   `json:"fork_source_agent_session_id"`
		GenerationGuidanceProfile    string                   `json:"generation_guidance_profile"`
		GenerationGuidanceSHA256     string                   `json:"generation_guidance_sha256"`
		PartEditEnabled              bool                     `json:"part_edit_enabled"`
		PartPlanningEnabled          bool                     `json:"part_planning_enabled"`
		Plan                         agentSectionalReportPlan `json:"plan"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return nil
	}
	if strings.TrimSpace(payload.PendingEventID) != pendingEventID {
		return nil
	}
	if payload.PartPlanningEnabled {
		parent, err := sectionFanoutPartPlanningParent(event, pendingEventID)
		if err != nil {
			return err
		}
		if parent.ReportMode != reportModeLongForm {
			return fmt.Errorf("%w: Part planning parent report mode is invalid", app.ErrConflict)
		}
		normalized, err := normalizeRecoveredSectionalPlan(payload.Plan)
		if err != nil {
			return err
		}
		progress.hasPlan = true
		progress.planEvent = event
		progress.plan = normalized
		progress.artifactID = strings.TrimSpace(payload.ArtifactID)
		progress.currentSessionID = strings.TrimSpace(payload.AgentSessionID)
		progress.reportSessionPolicy = parent.ReportSessionPolicy
		progress.reportSessionPolicySelection = parent.ReportSessionPolicySelection
		progress.agentExecutor = parent.AgentExecutor
		progress.agentModel = parent.AgentModel
		progress.agentReasoningEffort = parent.AgentReasoningEffort
		progress.agentSelectionSource = parent.AgentSelectionSource
		progress.mcpMode = strings.TrimSpace(payload.MCPMode)
		progress.sessionChainKind = parent.SessionChainKind
		progress.preReportResearchSessionID = strings.TrimSpace(payload.PreReportResearchSessionID)
		progress.reportPlanSessionID = parent.ReportPlanSessionID
		progress.forkSourceSessionID = strings.TrimSpace(payload.ForkSourceSessionID)
		progress.generationGuidanceProfile = parent.GenerationGuidanceProfile
		progress.generationGuidanceSHA256 = parent.GenerationGuidanceSHA256
		progress.partEditEnabled = parent.PartEditEnabled
		progress.partPlanningEnabled = parent.PartPlanningEnabled
		_ = ctx
		return nil
	}
	if strings.TrimSpace(payload.ReportMode) != reportModeLongForm {
		return nil
	}
	normalized, err := normalizeRecoveredSectionalPlan(payload.Plan)
	if err != nil {
		return err
	}
	progress.hasPlan = true
	progress.planEvent = event
	progress.plan = normalized
	progress.artifactID = strings.TrimSpace(payload.ArtifactID)
	if sessionID := strings.TrimSpace(payload.AgentSessionID); sessionID != "" {
		progress.currentSessionID = sessionID
	}
	progress.reportSessionPolicy = firstNonEmpty(payload.ReportSessionPolicy, reportSessionPolicySameSession)
	progress.reportSessionPolicySelection = strings.TrimSpace(payload.ReportSessionPolicySelection)
	progress.agentExecutor = strings.TrimSpace(strings.ToLower(payload.AgentExecutor))
	progress.agentModel = strings.TrimSpace(payload.AgentModel)
	progress.agentReasoningEffort = strings.TrimSpace(payload.AgentReasoningEffort)
	progress.agentSelectionSource = strings.TrimSpace(payload.AgentSelectionSource)
	progress.mcpMode = strings.TrimSpace(payload.MCPMode)
	progress.sessionChainKind = firstNonEmpty(payload.SessionChainKind, "same_session_report")
	progress.preReportResearchSessionID = firstNonEmpty(payload.PreReportResearchSessionID, payload.PreviousAgentSessionID)
	progress.reportPlanSessionID = firstNonEmpty(payload.ReportPlanSessionID, payload.AgentSessionID)
	progress.forkSourceSessionID = strings.TrimSpace(payload.ForkSourceSessionID)
	progress.generationGuidanceProfile = strings.TrimSpace(payload.GenerationGuidanceProfile)
	progress.generationGuidanceSHA256 = strings.TrimSpace(payload.GenerationGuidanceSHA256)
	progress.partEditEnabled = payload.PartEditEnabled
	progress.partPlanningEnabled = payload.PartPlanningEnabled
	_ = ctx
	return nil
}

func reportSectionStartedMatches(pendingEventID, planEventID string, event app.LedgerEvent) bool {
	var payload struct {
		PendingEventID string `json:"pending_event_id"`
		PlanEventID    string `json:"plan_event_id"`
		PartIndex      int    `json:"part_index"`
		SectionIndex   int    `json:"section_index"`
	}
	return json.Unmarshal(event.Payload, &payload) == nil &&
		strings.TrimSpace(payload.PendingEventID) == pendingEventID &&
		strings.TrimSpace(payload.PlanEventID) == planEventID &&
		payload.PartIndex > 0 &&
		payload.SectionIndex > 0
}

func applyPartPlanProgress(missionID string, pendingEventID string, planEventID string, partCount int, event app.LedgerEvent, progress *sectionalReportProgress) error {
	plan, ok, err := reporting.DecodeStoredPartPlan(event, reporting.StoredPartPlanExpectation{
		MissionID: missionID, PendingEventID: pendingEventID, PlanEventID: planEventID,
		PartCount: partCount, AgentExecutor: progress.agentExecutor, AgentModel: progress.agentModel,
		AgentReasoningEffort: progress.agentReasoningEffort, AgentSelectionSource: progress.agentSelectionSource,
		ReportMode: reportModeLongForm, ReportSessionPolicy: progress.reportSessionPolicy,
		ReportSessionPolicySelection: progress.reportSessionPolicySelection,
		GenerationGuidanceProfile:    progress.generationGuidanceProfile,
		GenerationGuidanceSHA256:     progress.generationGuidanceSHA256,
		SessionChainKind:             progress.sessionChainKind,
		ReportPlanSessionID:          progress.reportPlanSessionID,
	})
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	index := plan.PartIndex - 1
	if _, exists := progress.partPlans[index]; exists {
		return fmt.Errorf("%w: multiple recovered Part plans match one Part", app.ErrConflict)
	}
	progress.partPlans[index] = sectionFanoutPartPlan{
		brief: plan.Brief, providerSessionID: plan.ProviderSessionID, event: plan.Event,
	}
	return nil
}

func (server *Server) applySectionProgress(ctx context.Context, pendingEventID string, planEventID string, event app.LedgerEvent, progress *sectionalReportProgress) error {
	var payload struct {
		PendingEventID string `json:"pending_event_id"`
		PlanEventID    string `json:"plan_event_id"`
		ArtifactID     string `json:"artifact_id"`
		Title          string `json:"title"`
		AgentSessionID string `json:"agent_session_id"`
		PartIndex      int    `json:"part_index"`
		SectionIndex   int    `json:"section_index"`
		WordCount      int    `json:"word_count"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return nil
	}
	if strings.TrimSpace(payload.PendingEventID) != pendingEventID || payload.PartIndex <= 0 || payload.SectionIndex <= 0 {
		return nil
	}
	if strings.TrimSpace(payload.PlanEventID) != planEventID {
		return nil
	}
	markdown, ok := server.recoveredMarkdownArtifact(ctx, payload.ArtifactID, event.MissionID)
	if !ok {
		return nil
	}
	progress.sections[sectionalReportIndex{part: payload.PartIndex - 1, section: payload.SectionIndex - 1}] = sectionalReportDraft{
		Title:      strings.TrimSpace(payload.Title),
		Markdown:   markdown,
		ArtifactID: strings.TrimSpace(payload.ArtifactID),
		WordCount:  fallbackWordCount(payload.WordCount, markdown),
	}
	if sessionID := strings.TrimSpace(payload.AgentSessionID); sessionID != "" {
		progress.currentSessionID = sessionID
	}
	return nil
}

func (server *Server) applyPartProgress(ctx context.Context, pendingEventID string, planEventID string, event app.LedgerEvent, progress *sectionalReportProgress) error {
	var payload struct {
		PendingEventID string `json:"pending_event_id"`
		PlanEventID    string `json:"plan_event_id"`
		ArtifactID     string `json:"artifact_id"`
		Title          string `json:"title"`
		AgentSessionID string `json:"agent_session_id"`
		PartIndex      int    `json:"part_index"`
		WordCount      int    `json:"word_count"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return nil
	}
	if strings.TrimSpace(payload.PendingEventID) != pendingEventID || payload.PartIndex <= 0 {
		return nil
	}
	if strings.TrimSpace(payload.PlanEventID) != planEventID {
		return nil
	}
	markdown, ok := server.recoveredMarkdownArtifact(ctx, payload.ArtifactID, event.MissionID)
	if !ok {
		return nil
	}
	progress.parts[payload.PartIndex-1] = sectionalReportPartDraft{
		Title:      strings.TrimSpace(payload.Title),
		Markdown:   markdown,
		ArtifactID: strings.TrimSpace(payload.ArtifactID),
		WordCount:  fallbackWordCount(payload.WordCount, markdown),
	}
	if sessionID := strings.TrimSpace(payload.AgentSessionID); sessionID != "" {
		progress.currentSessionID = sessionID
	}
	return nil
}

func (server *Server) applyPartEditProgress(ctx context.Context, pendingEventID string, planEventID string, event app.LedgerEvent, progress *sectionalReportProgress) error {
	var payload struct {
		PendingEventID string `json:"pending_event_id"`
		PlanEventID    string `json:"plan_event_id"`
		PartIndex      int    `json:"part_index"`
		WordCount      int    `json:"edited_word_count"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return nil
	}
	if !progress.partEditEnabled || strings.TrimSpace(payload.PendingEventID) != pendingEventID || payload.PartIndex <= 0 {
		return nil
	}
	if strings.TrimSpace(payload.PlanEventID) != planEventID {
		return nil
	}
	source, exists := progress.parts[payload.PartIndex-1]
	if !exists || strings.TrimSpace(source.ArtifactID) == "" {
		return nil
	}
	sourcePartEventID, err := server.reportPartCreatedEventID(ctx, event.MissionID, planEventID, payload.PartIndex, source.ArtifactID)
	if err != nil {
		return err
	}
	outcome, ok, err := reporting.LoadPartEditOutcome(ctx, server.service, reporting.PartEditOutcomeContract{
		MissionID: event.MissionID, CurrentPendingEventID: pendingEventID, PlanEventID: planEventID,
		SourcePartEventID: sourcePartEventID, SourceArtifactID: source.ArtifactID, PartIndex: payload.PartIndex,
		AgentExecutor: progress.agentExecutor, AgentModel: progress.agentModel, AgentReasoningEffort: progress.agentReasoningEffort,
		AgentSelectionSource: progress.agentSelectionSource, MCPMode: progress.mcpMode,
		ReportSessionPolicy: progress.reportSessionPolicy, ReportSessionPolicySelection: progress.reportSessionPolicySelection,
		GenerationGuidanceProfile: progress.generationGuidanceProfile, GenerationGuidanceSHA256: progress.generationGuidanceSHA256,
		SessionChainKind: progress.sessionChainKind, ReportPlanSessionID: progress.reportPlanSessionID,
	})
	if err != nil || !ok || outcome.Event.EventID != event.EventID {
		return err
	}
	markdown := strings.TrimSpace(string(outcome.Artifact.Content))
	if markdown == "" {
		return nil
	}
	title := ""
	if source, exists := progress.parts[payload.PartIndex-1]; exists {
		title = source.Title
	}
	progress.editedParts[payload.PartIndex-1] = sectionalReportPartDraft{
		Title:      title,
		Markdown:   markdown,
		ArtifactID: outcome.Artifact.ArtifactID,
		WordCount:  fallbackWordCount(payload.WordCount, markdown),
	}
	return nil
}

func (server *Server) recoveredMarkdownArtifact(ctx context.Context, artifactID string, missionID string) (string, bool) {
	artifactID = strings.TrimSpace(artifactID)
	if artifactID == "" {
		return "", false
	}
	artifact, err := server.service.GetRawArtifact(ctx, artifactID)
	if err != nil || artifact.MissionID != missionID {
		return "", false
	}
	if !strings.HasPrefix(strings.ToLower(artifact.MediaType), "text/markdown") {
		return "", false
	}
	markdown := strings.TrimSpace(string(artifact.Content))
	if markdown == "" {
		return "", false
	}
	return markdown, true
}

func normalizeRecoveredSectionalPlan(plan agentSectionalReportPlan) (agentSectionalReportPlan, error) {
	return reporting.NormalizeSectionalReportPlan(plan)
}

func fallbackWordCount(count int, markdown string) int {
	if count > 0 {
		return count
	}
	return reportWordCount(markdown)
}

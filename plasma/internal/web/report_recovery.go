package web

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type sectionalReportProgress struct {
	hasPlan                      bool
	planEvent                    app.LedgerEvent
	plan                         agentSectionalReportPlan
	artifactID                   string
	currentSessionID             string
	reportSessionPolicy          string
	reportSessionPolicySelection string
	sessionChainKind             string
	preReportResearchSessionID   string
	reportPlanSessionID          string
	forkSourceSessionID          string
	sections                     map[sectionalReportIndex]sectionalReportDraft
	parts                        map[int]sectionalReportPartDraft
}

type sectionalReportIndex struct {
	part    int
	section int
}

func (server *Server) resumeReportDraftWorker(ctx context.Context, missionID string, pending app.LedgerEvent) error {
	req, err := reportDraftRequestFromPendingEvent(pending)
	if err != nil {
		_, failErr := server.reportRunner().AppendDraftFailed(ctx, missionID, pending.EventID, reportDraftPendingExecutor(pending), reportDraftPendingMode(pending), err)
		return failErr
	}
	return server.reportRunner().RunDraft(context.Background(), missionID, reporting.DraftRequest{
		Title:                        req.Title,
		DirectionHint:                req.DirectionHint,
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

func reportDraftRequestFromPendingEvent(event app.LedgerEvent) (reportDraftRequest, error) {
	var payload struct {
		Title                        string `json:"title"`
		DirectionHint                string `json:"direction_hint"`
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
		Title:                        firstNonEmpty(payload.Title, "Mission report"),
		DirectionHint:                reporting.NormalizeDirectionHint(payload.DirectionHint),
		AgentExecutor:                firstNonEmpty(payload.AgentExecutor, "codex"),
		AgentModel:                   strings.TrimSpace(payload.AgentModel),
		AgentReasoningEffort:         strings.TrimSpace(payload.AgentReasoningEffort),
		AgentSelectionSource:         strings.TrimSpace(payload.AgentSelectionSource),
		MCPMode:                      firstNonEmpty(payload.MCPMode, "auto"),
		RigorLevel:                   firstNonEmpty(payload.RigorLevel, defaultReportRigorLevel),
		ReportMode:                   firstNonEmpty(payload.ReportMode, defaultReportMode),
		ReportSessionPolicy:          firstNonEmpty(payload.ReportSessionPolicy, reportSessionPolicySameSession),
		ReportSessionPolicySelection: strings.TrimSpace(payload.ReportSessionPolicySelection),
		PostReportHumanize:           strings.TrimSpace(payload.PostReportHumanize),
		GenerationGuidanceProfile:    strings.TrimSpace(payload.GenerationGuidanceProfile),
		GenerationGuidanceSHA256:     strings.TrimSpace(payload.GenerationGuidanceSHA256),
	}, nil
}

func (server *Server) loadSectionalReportProgress(ctx context.Context, missionID string, pendingEventID string) (sectionalReportProgress, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return sectionalReportProgress{}, err
	}
	progress := sectionalReportProgress{
		sections: map[sectionalReportIndex]sectionalReportDraft{},
		parts:    map[int]sectionalReportPartDraft{},
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
			case "report.section.created":
				if err := server.applySectionProgress(ctx, attemptID, progress.planEvent.EventID, event, &progress); err != nil {
					return sectionalReportProgress{}, err
				}
			case "report.part.created":
				if err := server.applyPartProgress(ctx, attemptID, progress.planEvent.EventID, event, &progress); err != nil {
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
		ReportSessionPolicy          string                   `json:"report_session_policy"`
		ReportSessionPolicySelection string                   `json:"report_session_policy_selection"`
		SessionChainKind             string                   `json:"session_chain_kind"`
		PreReportResearchSessionID   string                   `json:"pre_report_research_session_id"`
		ReportPlanSessionID          string                   `json:"report_plan_session_id"`
		ForkSourceSessionID          string                   `json:"fork_source_agent_session_id"`
		Plan                         agentSectionalReportPlan `json:"plan"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return nil
	}
	if strings.TrimSpace(payload.PendingEventID) != pendingEventID || strings.TrimSpace(payload.ReportMode) != reportModeLongForm {
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
	progress.sessionChainKind = firstNonEmpty(payload.SessionChainKind, "same_session_report")
	progress.preReportResearchSessionID = firstNonEmpty(payload.PreReportResearchSessionID, payload.PreviousAgentSessionID)
	progress.reportPlanSessionID = firstNonEmpty(payload.ReportPlanSessionID, payload.AgentSessionID)
	progress.forkSourceSessionID = strings.TrimSpace(payload.ForkSourceSessionID)
	_ = ctx
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
	encoded, err := json.Marshal(plan)
	if err != nil {
		return agentSectionalReportPlan{}, err
	}
	return parseAgentSectionalReportPlan(string(encoded))
}

func fallbackWordCount(count int, markdown string) int {
	if count > 0 {
		return count
	}
	return reportWordCount(markdown)
}

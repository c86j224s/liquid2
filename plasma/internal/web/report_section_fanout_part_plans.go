package web

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

type sectionFanoutPartPlanTask struct {
	partIndex         int
	part              agentReportPart
	providerSession   string
	forkSourceSession string
}

type sectionFanoutPartPlanOutcome struct {
	partIndex int
	plan      sectionFanoutPartPlan
	err       error
}

func longFormPartPlanningEnabled(profile string) bool {
	return reportprompt.IsPartConnectiveEconomyVoice(profile) ||
		reportprompt.IsPartConnectiveSubjectDirectSynthesis(profile)
}

func (server *Server) ensureSectionFanoutPartPlans(ctx context.Context, req sectionFanoutLongFormRequest, state sectionFanoutPlanState, progress sectionalReportProgress, forker AgentSessionForker, executor AgentExecutor) (map[int]sectionFanoutPartPlan, error) {
	plans := make(map[int]sectionFanoutPartPlan, len(state.plan.Parts))
	for index, plan := range progress.partPlans {
		plans[index] = plan
	}
	tasks := make([]sectionFanoutPartPlanTask, 0, len(state.plan.Parts)-len(plans))
	for partIndex, part := range state.plan.Parts {
		if _, ok := plans[partIndex]; ok {
			continue
		}
		if _, hasPart := progress.parts[partIndex]; hasPart || hasRecoveredSection(progress, partIndex) {
			return nil, fmt.Errorf("%w: Part output exists without its planning state", app.ErrConflict)
		}
		providerSession, forkSource, err := forkSectionFanoutSession(ctx, forker, state.reportPlanSessionID)
		if err != nil {
			return nil, longFormStageFailure("part_plan", state.planEvent.EventID, partIndex+1, 0, err)
		}
		tasks = append(tasks, sectionFanoutPartPlanTask{
			partIndex: partIndex, part: part, providerSession: providerSession,
			forkSourceSession: firstNonEmpty(forkSource, state.reportPlanSessionID),
		})
	}
	if len(tasks) == 0 {
		return plans, nil
	}

	limit := sectionFanoutWorkerLimit
	if limit > len(tasks) {
		limit = len(tasks)
	}
	sem := make(chan struct{}, limit)
	outcomes := make(chan sectionFanoutPartPlanOutcome, len(tasks))
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		sem <- struct{}{}
		go func(task sectionFanoutPartPlanTask) {
			defer wg.Done()
			defer func() { <-sem }()
			outcomes <- server.runSectionFanoutPartPlan(ctx, req, state, task, executor)
		}(task)
	}
	go func() {
		wg.Wait()
		close(outcomes)
	}()
	var firstErr error
	for outcome := range outcomes {
		if outcome.err != nil {
			if firstErr == nil {
				firstErr = outcome.err
			}
			continue
		}
		plans[outcome.partIndex] = outcome.plan
	}
	if firstErr != nil {
		return nil, firstErr
	}
	if len(plans) != len(state.plan.Parts) {
		return nil, fmt.Errorf("%w: Part planning left a Part incomplete", app.ErrConflict)
	}
	return plans, nil
}

func (server *Server) runSectionFanoutPartPlan(ctx context.Context, req sectionFanoutLongFormRequest, state sectionFanoutPlanState, task sectionFanoutPartPlanTask, executor AgentExecutor) sectionFanoutPartPlanOutcome {
	toolSessionID := newID("ses")
	started := time.Now()
	agentExecutor := firstNonEmpty(state.agentExecutor, req.executorName)
	agentModel := state.agentModel
	agentReasoningEffort := state.agentReasoningEffort
	agentSelectionSource := state.agentSelectionSource
	generationGuidanceProfile := state.generationGuidanceProfile
	generationGuidanceSHA256 := state.generationGuidanceSHA256
	result, err := executor.Run(ctx, AgentRequest{
		UserText:          fmt.Sprintf("plan the reading flow for Part %d of the long-form report", task.partIndex+1),
		Prompt:            agentPartPlanningPrompt(req, state, task),
		Model:             agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MissionID:         req.missionID,
		ToolSessionID:     toolSessionID,
		PreviousSessionID: task.providerSession,
		AgentExecutor:     agentExecutor,
		MCPMode:           req.mcpMode,
		ReplaceMCPTools:   true,
	})
	durationMS := time.Since(started).Milliseconds()
	if err == nil {
		result, err = validatedSameSessionResult(result, task.providerSession)
	}
	if err != nil {
		return sectionFanoutPartPlanOutcome{partIndex: task.partIndex, err: longFormStageFailure("part_plan", state.planEvent.EventID, task.partIndex+1, 0, reportAgentFailure(err, result, "report_part_plan", durationMS, task.providerSession))}
	}
	brief := strings.TrimSpace(result.Text)
	if brief == "" {
		return sectionFanoutPartPlanOutcome{partIndex: task.partIndex, err: longFormStageFailure("part_plan", state.planEvent.EventID, task.partIndex+1, 0, fmt.Errorf("%w: Part planner returned an empty brief", app.ErrInvalidInput))}
	}
	finalized, err := reporting.FinalizePartPlan(context.WithoutCancel(ctx), server.service, reporting.PartPlanCreatedEventRequest{
		MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
			EventID: newID("evt"), MissionID: req.missionID, PendingEventID: req.pendingEventID, PlanEventID: state.planEvent.EventID,
			Title: task.part.Title, AgentExecutor: agentExecutor, AgentModel: agentModel,
			AgentReasoningEffort: agentReasoningEffort, AgentSelectionSource: agentSelectionSource,
			AgentSessionID: result.SessionID, PreviousAgentSessionID: task.providerSession,
			ReturnedAgentSessionID: result.SessionID, ToolSessionID: toolSessionID,
			ReportMode: reportModeLongForm, ReportModeLabel: reportModeLabel(reportModeLongForm),
			ReportSessionPolicy: state.reportSessionPolicy, ReportSessionPolicySelection: state.reportSessionPolicySelection,
			PostReportHumanize: req.postReportHumanize, HumanizeEnabled: req.postReportHumanize != "disabled",
			GenerationGuidanceProfile: generationGuidanceProfile, GenerationGuidanceSHA256: generationGuidanceSHA256,
			SessionChainKind: state.sessionChainKind, PreReportResearchSessionID: state.preReportResearchSessionID,
			ReportPlanSessionID: state.reportPlanSessionID, ReportSessionID: result.SessionID,
			ForkSourceAgentSessionID: task.forkSourceSession, CompositionStrategy: "sectional_preserve_markdown",
			AssemblyStrategy: "c4_normalized_section_headings", DurationMS: durationMS,
			AgentUsage: result.Usage, AgentUsageSurface: "report_part_plan", AgentUsageDurationMS: durationMS,
			AgentResumed: result.Resumed, Producer: app.Producer{Type: "agent_session", ID: result.SessionID},
		},
		PartIndex: task.partIndex + 1,
		Brief:     brief,
	})
	if err != nil {
		return sectionFanoutPartPlanOutcome{partIndex: task.partIndex, err: longFormStageFailure("part_plan", state.planEvent.EventID, task.partIndex+1, 0, err)}
	}
	return sectionFanoutPartPlanOutcome{partIndex: task.partIndex, plan: sectionFanoutPartPlan{
		brief: finalized.Brief, providerSessionID: finalized.ProviderSessionID, event: finalized.Event,
	}}
}

func agentPartPlanningPrompt(req sectionFanoutLongFormRequest, state sectionFanoutPlanState, task sectionFanoutPartPlanTask) string {
	requirements := reporting.ReportRequirementsForPart(state.requirementMap, task.partIndex+1)
	prompt := fmt.Sprintf(`Take responsibility for the reading flow of one Part before its Sections are drafted.

Report title: %s
Part %d: %s

Overall report plan:
%s

Requirements assigned to this Part:
%s

Write a short private editorial brief that you can use again when the assembled Part returns for final authorship. State the Part's central reader question, the intended flow across its Sections, and one natural-sentence job for each Section. When one explanation clearly belongs in a single Section, name that home once; otherwise leave that point out.

Teach the later Section writers what a reader should understand and how the explanation should move. Keep this as useful working memory rather than a compliance checklist. Do not draft report paragraphs or add new researched facts. Return only the brief.`,
		req.title, task.partIndex+1, task.part.Title, agentReportAnyJSON(state.plan), agentReportAnyJSON(requirements))
	return withLongFormDownstreamDirection(prompt, req.directionHint)
}

func hasRecoveredSection(progress sectionalReportProgress, partIndex int) bool {
	for index := range progress.sections {
		if index.part == partIndex {
			return true
		}
	}
	return false
}

func sectionFanoutPartSourceSession(state sectionFanoutPlanState, partIndex int) (string, error) {
	if !state.partPlanningEnabled {
		return state.reportPlanSessionID, nil
	}
	plan, ok := state.partPlans[partIndex]
	if !ok || strings.TrimSpace(plan.providerSessionID) == "" {
		return "", fmt.Errorf("%w: Part planning session is missing", app.ErrConflict)
	}
	return strings.TrimSpace(plan.providerSessionID), nil
}

package web

import (
	"context"
	"fmt"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const sectionFanoutWorkerLimit = 8

type sectionFanoutLongFormRequest struct {
	missionID                    string
	title                        string
	directionHint                string
	executorName                 string
	agentModel                   string
	agentReasoningEffort         string
	agentSelectionSource         string
	mcpMode                      string
	rigor                        reportRigorProfile
	reportSessionPolicy          string
	reportSessionPolicySelection string
	postReportHumanize           string
	generationGuidanceProfile    string
	generationGuidanceSHA256     string
	pendingEventID               string
}

type sectionFanoutPlanState struct {
	artifactID                   string
	plan                         agentSectionalReportPlan
	planEvent                    app.LedgerEvent
	reportPlanSessionID          string
	agentExecutor                string
	agentModel                   string
	agentReasoningEffort         string
	agentSelectionSource         string
	reportSessionPolicy          string
	reportSessionPolicySelection string
	sessionChainKind             string
	preReportResearchSessionID   string
	forkSourceSessionID          string
	generationGuidanceProfile    string
	generationGuidanceSHA256     string
	requirementMap               reporting.ReportRequirementMap
	requirementMapEvent          app.LedgerEvent
	partEditEnabled              bool
	partPlanningEnabled          bool
	partPlans                    map[int]sectionFanoutPartPlan
}

type sectionFanoutPartPlan struct {
	brief             string
	providerSessionID string
	event             app.LedgerEvent
}

type sectionFanoutTask struct {
	partIndex       int
	sectionIndex    int
	part            agentReportPart
	section         agentReportSection
	previousSession string
	toolSessionID   string
	sourceSessionID string
}

type sectionFanoutResult struct {
	task              sectionFanoutTask
	result            AgentResult
	returnedSessionID string
	durationMS        int64
	markdown          string
	draft             sectionalReportDraft
}

func (server *Server) createSectionFanoutLongFormReportDraft(ctx context.Context, missionID string, title string, directionHint string, executorName string, agentModel string, agentReasoningEffort string, agentSelectionSource string, mcpMode string, rigor reportRigorProfile, reportSessionPolicy string, reportSessionPolicySelection string, postReportHumanize string, generationGuidanceProfile string, generationGuidanceSHA256 string, pendingEventID string, executor AgentExecutor) (map[string]any, error) {
	req := sectionFanoutLongFormRequest{
		missionID:                    missionID,
		title:                        title,
		directionHint:                directionHint,
		executorName:                 executorName,
		agentModel:                   agentModel,
		agentReasoningEffort:         agentReasoningEffort,
		agentSelectionSource:         agentSelectionSource,
		mcpMode:                      mcpMode,
		rigor:                        rigor,
		reportSessionPolicy:          reportSessionPolicy,
		reportSessionPolicySelection: reportSessionPolicySelection,
		postReportHumanize:           postReportHumanize,
		generationGuidanceProfile:    generationGuidanceProfile,
		generationGuidanceSHA256:     generationGuidanceSHA256,
		pendingEventID:               pendingEventID,
	}
	return server.runSectionFanoutLongFormReport(ctx, req, executor)
}

func (server *Server) runSectionFanoutLongFormReport(ctx context.Context, req sectionFanoutLongFormRequest, executor AgentExecutor) (map[string]any, error) {
	started := time.Now()
	forker, ok := executor.(AgentSessionForker)
	if !ok {
		return nil, fmt.Errorf("%w: section fanout requires an agent session forker", app.ErrInvalidInput)
	}
	progress, err := server.loadSectionalReportProgress(ctx, req.missionID, req.pendingEventID)
	if err != nil {
		return nil, err
	}
	state, err := server.ensureSectionFanoutPlan(ctx, req, progress, executor)
	if err != nil {
		return nil, err
	}
	state.requirementMap, state.requirementMapEvent, err = server.ensureReportRequirementMap(ctx, reportRequirementStageRequest{
		missionID:       req.missionID,
		title:           req.title,
		directionHint:   req.directionHint,
		executorName:    req.executorName,
		agentModel:      req.agentModel,
		reasoningEffort: req.agentReasoningEffort,
		mcpMode:         req.mcpMode,
		pendingEventID:  req.pendingEventID,
		planEventID:     state.planEvent.EventID,
		planSessionID:   state.reportPlanSessionID,
		plan:            state.plan,
	}, progress, executor)
	if err != nil {
		return nil, err
	}
	if state.partPlanningEnabled {
		state.partPlans, err = server.ensureSectionFanoutPartPlans(ctx, req, state, progress, forker, executor)
		if err != nil {
			return nil, err
		}
	}
	sections, sectionArtifactIDs, sectionWordTotal, err := server.draftSectionFanoutSections(ctx, req, state, progress, forker, executor)
	if err != nil {
		return nil, err
	}
	parts, partArtifactIDs, err := server.assembleSectionFanoutParts(ctx, req, state, progress, sections, forker, executor)
	if err != nil {
		return nil, err
	}
	if longFormReaderStyleGatePlanEventEnabled(state.planEvent) {
		req.agentReasoningEffort = longFormFinalEditContractReasoningEffort(req.agentReasoningEffort)
		state.agentReasoningEffort = longFormFinalEditContractReasoningEffort(firstNonEmpty(state.agentReasoningEffort, req.agentReasoningEffort))
	}
	if state.partEditEnabled {
		if state.partPlanningEnabled {
			parts, partArtifactIDs, err = server.authorSectionFanoutParts(ctx, req, state, progress, parts, executor)
		} else {
			parts, partArtifactIDs, err = server.editSectionFanoutParts(ctx, req, state, progress, parts, forker, executor)
		}
		if err != nil {
			return nil, err
		}
	}
	if longFormReaderStyleGatePlanEventEnabled(state.planEvent) {
		return server.runLongFormReaderStyleGatePipeline(ctx, longFormReaderStyleGatePipelineRequest{
			missionID: req.missionID, title: req.title, executorName: req.executorName,
			agentModel: req.agentModel, agentReasoningEffort: req.agentReasoningEffort,
			agentSelectionSource: req.agentSelectionSource, mcpMode: req.mcpMode, rigor: req.rigor,
			reportSessionPolicy:          firstNonEmpty(state.reportSessionPolicy, req.reportSessionPolicy),
			reportSessionPolicySelection: firstNonEmpty(state.reportSessionPolicySelection, req.reportSessionPolicySelection),
			postReportHumanize:           req.postReportHumanize,
			generationGuidanceProfile:    firstNonEmpty(state.generationGuidanceProfile, req.generationGuidanceProfile),
			generationGuidanceSHA256:     firstNonEmpty(state.generationGuidanceSHA256, req.generationGuidanceSHA256),
			pendingEventID:               req.pendingEventID, artifactID: state.artifactID, planEvent: state.planEvent,
			plan: state.plan, requirementMap: state.requirementMap, parts: parts, partArtifactIDs: partArtifactIDs,
			sectionArtifactIDs: sectionArtifactIDs, sectionWordTotal: sectionWordTotal,
			sessionChainKind:           firstNonEmpty(state.sessionChainKind, "section_fanout_report"),
			preReportResearchSessionID: state.preReportResearchSessionID,
			reportPlanSessionID:        state.reportPlanSessionID,
			forkSourceAgentSessionID:   state.forkSourceSessionID,
			started:                    started,
		}, executor)
	}
	finalSessionID, finalForkSourceID, err := forkSectionFanoutSession(ctx, forker, state.reportPlanSessionID)
	if err != nil {
		return nil, longFormStageFailure("final", state.planEvent.EventID, 0, 0, err)
	}
	if finalForkSourceID == "" {
		finalForkSourceID = state.reportPlanSessionID
	}
	artifact, event, finalResult, err := server.finalizeSectionFanoutLongForm(ctx, req, state, parts, sectionArtifactIDs, partArtifactIDs, sectionWordTotal, finalSessionID, finalForkSourceID, started, executor)
	if err != nil {
		return nil, err
	}
	markdown := string(artifact.Content)
	if req.postReportHumanize == "disabled" {
		return map[string]any{"artifact": artifact, "event": event, "markdown": markdown}, nil
	}
	humanized, err := server.humanizeMarkdownReport(ctx, req.missionID, reportHumanizeInput{
		Title:             req.title,
		Markdown:          markdown,
		SourceArtifact:    artifact,
		ExecutorName:      req.executorName,
		AgentModel:        req.agentModel,
		ReasoningEffort:   req.agentReasoningEffort,
		MCPMode:           req.mcpMode,
		PreviousSessionID: fallbackSessionID(finalResult.SessionID, finalSessionID),
		ReportMode:        reportModeLongForm,
		PendingEventID:    req.pendingEventID,
	}, executor)
	if err != nil {
		return nil, err
	}
	return map[string]any{"artifact": artifact, "event": event, "markdown": markdown, "humanized": humanized}, nil
}

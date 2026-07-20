package web

import (
	"context"
	"fmt"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
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
	reportSessionPolicy          string
	reportSessionPolicySelection string
	sessionChainKind             string
	preReportResearchSessionID   string
	forkSourceSessionID          string
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
	sections, sectionArtifactIDs, sectionWordTotal, err := server.draftSectionFanoutSections(ctx, req, state, progress, forker, executor)
	if err != nil {
		return nil, err
	}
	parts, partArtifactIDs, err := server.assembleSectionFanoutParts(ctx, req, state, progress, sections, forker, executor)
	if err != nil {
		return nil, err
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

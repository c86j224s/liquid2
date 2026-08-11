package web

import (
	"context"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow"
)

const (
	finalEditStageSubmittedText = "FINAL_EDIT_STAGE_SUBMITTED"
	finalEditGateSubmittedText  = "REPORT_FINALIZED"
)

// finalizationPrefixFixture는 Web test의 기존 seed 값을 reportworkflow handoff로 옮기는
// 순수 데이터 fixture다. topology, prompt/tool, retry, session 정책을 계산하지 않는다.
type finalizationPrefixFixture struct {
	missionID                    string
	title                        string
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
	directionHint                string
	artifactID                   string
	planEvent                    app.LedgerEvent
	plan                         agentSectionalReportPlan
	requirementMap               reporting.ReportRequirementMap
	parts                        []sectionalReportPartDraft
	partArtifactIDs              []string
	sectionArtifactIDs           []string
	sectionWordTotal             int
	sessionChainKind             string
	preReportResearchSessionID   string
	reportPlanSessionID          string
	forkSourceAgentSessionID     string
	finalTail                    reportworkflow.FinalTail
	started                      time.Time
}

// finalizePrefixForWebTest는 test fixture가 만든 PrefixOutput을 실제 reportworkflow graph에 넘긴다.
func finalizePrefixForWebTest(ctx context.Context, svc *app.Service, id func(string) string, req finalizationPrefixFixture, executor AgentExecutor) (reportworkflow.DraftOutput, error) {
	server := NewServer(svc, Options{}).(*Server)
	return reportworkflow.NewRunner(reportworkflow.RunnerConfig{
		Service: svc, Lifecycle: reporting.Runner(server.reportRunner()),
		Executor: executor, NewID: id,
		LatestSessionID: server.latestAgentSessionID,
	}).FinalizeLongFormPrefix(ctx, req.toPrefixOutput())
}

// toPrefixOutput은 fixture field를 같은 의미의 typed handoff field로 복사한다.
func (req finalizationPrefixFixture) toPrefixOutput() reportworkflow.PrefixOutput {
	parts := make([]reportworkflow.PrefixPart, len(req.parts))
	for index, part := range req.parts {
		parts[index] = reportworkflow.PrefixPart{Title: part.Title, Markdown: part.Markdown, ArtifactID: part.ArtifactID, WordCount: part.WordCount}
	}
	return reportworkflow.PrefixOutput{
		MissionID: req.missionID, PendingEventID: req.pendingEventID, Title: req.title,
		DirectionHint: req.directionHint, AgentExecutor: req.executorName,
		AgentModel: req.agentModel, AgentReasoningEffort: req.agentReasoningEffort,
		AgentSelectionSource: req.agentSelectionSource, MCPMode: req.mcpMode, Rigor: reportWorkflowRigor(req.rigor),
		ReportSessionPolicy: req.reportSessionPolicy, ReportSessionPolicySelection: req.reportSessionPolicySelection,
		PostReportHumanize: req.postReportHumanize, GenerationGuidanceProfile: req.generationGuidanceProfile,
		GenerationGuidanceSHA256: req.generationGuidanceSHA256, ArtifactID: req.artifactID,
		PlanEvent: req.planEvent, Plan: req.plan, RequirementMap: req.requirementMap, Parts: parts,
		PartArtifactIDs: req.partArtifactIDs, SectionArtifactIDs: req.sectionArtifactIDs,
		SectionWordTotal: req.sectionWordTotal, SessionChainKind: req.sessionChainKind,
		PreReportResearchSessionID: req.preReportResearchSessionID, ReportPlanSessionID: req.reportPlanSessionID,
		ForkSourceAgentSessionID: req.forkSourceAgentSessionID, FinalEditPipeline: req.planEventPipeline(),
		FinalTail: req.finalTail, StartedAt: req.started, ExecutionStrategy: reportExecutionStrategySerial,
	}
}

func (req finalizationPrefixFixture) planEventPipeline() string {
	state, ok, err := reporting.FinalEditPipelineFromPlanEvent(req.planEvent)
	if err != nil || !ok {
		return ""
	}
	return state.Pipeline
}

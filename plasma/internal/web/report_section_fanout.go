package web

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

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
	finalUserText                string
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

func (server *Server) createSectionFanoutLongFormReportDraft(ctx context.Context, missionID string, title string, directionHint string, executorName string, agentModel string, agentReasoningEffort string, agentSelectionSource string, mcpMode string, rigor reportRigorProfile, reportSessionPolicy string, reportSessionPolicySelection string, postReportHumanize string, generationGuidanceProfile string, generationGuidanceSHA256 string, pendingEventID string, executor AgentExecutor) (map[string]any, error) {
	return server.createLongFormPrefixWorkflowDraft(ctx, missionID, title, directionHint, executorName, agentModel, agentReasoningEffort, agentSelectionSource, mcpMode, rigor, reportSessionPolicy, reportSessionPolicySelection, postReportHumanize, generationGuidanceProfile, generationGuidanceSHA256, pendingEventID, reportExecutionStrategySectionFanout, executor)
}

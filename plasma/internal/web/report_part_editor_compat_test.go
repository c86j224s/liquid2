package web

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partedit"
	workflowplan "github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
)

type reportPartEditorRequest struct {
	title                        string
	missionID                    string
	pendingEventID               string
	planEventID                  string
	toolSessionID                string
	previousSessionID            string
	editedArtifactID             string
	filename                     string
	executorName                 string
	agentModel                   string
	agentReasoningEffort         string
	agentSelectionSource         string
	mcpMode                      string
	rigor                        reportRigorProfile
	plan                         agentSectionalReportPlan
	part                         agentReportPart
	partIndex                    int
	source                       sectionalReportPartDraft
	directionHint                string
	requirements                 []reporting.ReportRequirement
	requirementMapEvent          app.LedgerEvent
	requirementMap               reporting.ReportRequirementMap
	reportSessionPolicy          string
	reportSessionPolicySelection string
	generationGuidanceProfile    string
	generationGuidanceSHA256     string
	sessionChainKind             string
	reportPlanSessionID          string
	forkSourceAgentSessionID     string
}

type reportPartAuthorRequest struct {
	editor            reportPartEditorRequest
	partPlanningBrief string
}

func longFormPartEditEnabled(profile string) bool {
	return workflowplan.LongFormPartEditEnabled(profile)
}

func (server *Server) runPartEditorAgent(ctx context.Context, req reportPartEditorRequest, executor AgentExecutor) (sectionalReportPartDraft, AgentResult, error) {
	out, err := (partedit.Runner{Service: server.service, Executor: executor, NewID: newID}).Run(ctx, partEditInput(req, false, ""))
	return partEditDraft(out), out.Result, err
}

func (server *Server) partEditBinding(ctx context.Context, req reportPartEditorRequest) (reporting.PartEditBinding, error) {
	return (partedit.Runner{Service: server.service}).Binding(ctx, partEditInput(req, false, ""))
}

func agentPartEditorPrompt(req reportPartEditorRequest, binding reporting.PartEditBinding, draftID string) string {
	return partedit.Prompt(partEditInput(req, false, ""), binding, draftID)
}

func agentPartAuthorPrompt(req reportPartAuthorRequest, binding reporting.PartEditBinding, draftID string) string {
	return partedit.Prompt(partEditInput(req.editor, true, req.partPlanningBrief), binding, draftID)
}

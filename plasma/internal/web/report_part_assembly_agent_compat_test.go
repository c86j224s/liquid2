package web

import (
	"context"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partassembly"
)

type reportPartAssemblyAgentRequest struct {
	title                        string
	missionID                    string
	toolSessionID                string
	previousSessionID            string
	pendingEventID               string
	planEventID                  string
	executorName                 string
	agentModel                   string
	agentReasoningEffort         string
	agentSelectionSource         string
	mcpMode                      string
	rigor                        reportRigorProfile
	plan                         agentSectionalReportPlan
	part                         agentReportPart
	directionHint                string
	drafts                       []sectionalReportDraft
	partIndex                    int
	reportSessionPolicy          string
	reportSessionPolicySelection string
	postReportHumanize           string
	generationGuidanceProfile    string
	generationGuidanceSHA256     string
	sessionChainKind             string
	preReportResearchSessionID   string
	reportPlanSessionID          string
	forkSourceAgentSessionID     string
}

func (server *Server) runPartAssemblyAgent(ctx context.Context, req reportPartAssemblyAgentRequest, executor AgentExecutor) (agentPartAssembly, AgentResult, string, error) {
	return (partassembly.Runner{Service: server.service, Executor: executor, NewID: newID}).RunAgent(ctx, partAssemblyInput(req))
}

func (req reportPartAssemblyAgentRequest) partAssemblyBinding() reporting.PartAssemblyBinding {
	return partassembly.Binding(partAssemblyInput(req))
}

func partSectionArtifactIDs(drafts []sectionalReportDraft) []string {
	ids := make([]string, len(drafts))
	for index, draft := range drafts {
		ids[index] = strings.TrimSpace(draft.ArtifactID)
	}
	return ids
}

func usePartAssemblyEditTools(profile string) bool {
	return partassembly.UseEditTools(profile)
}

func agentPartAssemblyEditToolsPrompt(req reportPartAssemblyAgentRequest, binding reporting.PartAssemblyBinding, draftID string) string {
	return partassembly.EditToolsPrompt(partAssemblyInput(req), binding, draftID)
}

func parseAgentPartAssembly(text string) (agentPartAssembly, error) {
	return partassembly.ParseAgentPartAssembly(text)
}

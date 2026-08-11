package web

import (
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partassembly"
	workflowplan "github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/sectiondraft"
)

func agentSectionalReportPlanPrompt(title string, missionID string, toolSessionID string, pendingEventID string, idempotencyKey string, rigor reportRigorProfile, generationGuidanceProfile string) string {
	return workflowplan.LongFormPrompt(title, missionID, toolSessionID, pendingEventID, idempotencyKey, reportWorkflowRigor(rigor), generationGuidanceProfile)
}

func agentSectionDraftPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile, plan agentSectionalReportPlan, part agentReportPart, section agentReportSection, partIndex int, sectionIndex int, generationGuidanceProfile string) string {
	return sectiondraft.PromptWithRequirements(sectionDraftPromptInput(title, missionID, toolSessionID, rigor, plan, part, section, partIndex, sectionIndex, generationGuidanceProfile), nil)
}

func agentPartAssemblyPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile, plan agentSectionalReportPlan, part agentReportPart, drafts []sectionalReportDraft, partIndex int, generationGuidanceProfile string) string {
	return partassembly.Prompt(partAssemblyInput(reportPartAssemblyAgentRequest{
		title: title, missionID: missionID, toolSessionID: toolSessionID, rigor: rigor,
		plan: plan, part: part, drafts: drafts, partIndex: partIndex, generationGuidanceProfile: generationGuidanceProfile,
	}))
}

package web

import (
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	workflowrequirements "github.com/c86j224s/liquid2/plasma/internal/reportworkflow/requirements"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/sectiondraft"
)

func agentReportRequirementMapPrompt(title, directionHint string, plan agentSectionalReportPlan, reviewEventIDs []string, binding reporting.ReportRequirementMapBinding) string {
	return workflowrequirements.Prompt(title, directionHint, plan, reviewEventIDs, binding)
}

func agentSectionDraftPromptWithRequirements(title string, missionID string, toolSessionID string, rigor reportRigorProfile, plan agentSectionalReportPlan, part agentReportPart, section agentReportSection, partIndex int, sectionIndex int, generationGuidanceProfile string, requirements []reporting.ReportRequirement) string {
	return sectiondraft.PromptWithRequirements(sectionDraftPromptInput(title, missionID, toolSessionID, rigor, plan, part, section, partIndex, sectionIndex, generationGuidanceProfile), requirements)
}

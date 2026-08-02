package web

import (
	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// Report sessions may read accepted mission material, but source discovery and
// source-candidate tools stay in conversation/research sessions.
func reportReadMCPTools() []string {
	return []string{
		plasmamcp.ToolResearchOutline,
		plasmamcp.ToolResearchList,
		plasmamcp.ToolResearchGrep,
		plasmamcp.ToolResearchRead,
		plasmamcp.ToolResearchRefs,
		plasmamcp.ToolMermaidValidate,
		plasmamcp.ToolSourcesList,
		plasmamcp.ToolSourcesRead,
		plasmamcp.ToolSourcesTree,
		plasmamcp.ToolSourcesGrep,
	}
}

func reportPlanMCPTools() []string {
	return append(reportReadMCPTools(), plasmamcp.ToolReportPlanSubmit)
}

func reportRequirementMCPTools() []string {
	return []string{
		plasmamcp.ToolResearchRead,
		plasmamcp.ToolReportRequirementsSubmit,
	}
}

func reportPartAssemblyMCPTools(profile string) []string {
	tools := []string{
		plasmamcp.ToolReportPartAssemblyStart,
		plasmamcp.ToolReportPartAssemblyRead,
		plasmamcp.ToolReportPartAssemblyPatch,
		plasmamcp.ToolReportPartAssemblySubmit,
	}
	if isReportGenerationGuidanceProfileNarrativeContract(profile) {
		tools = append(tools, plasmamcp.ToolReportPartSectionRead, plasmamcp.ToolMermaidValidate)
	}
	return tools
}

func reportPartEditMCPTools() []string {
	return []string{
		plasmamcp.ToolReportPartEditStart,
		plasmamcp.ToolReportPartEditRead,
		plasmamcp.ToolReportPartEditPatch,
		plasmamcp.ToolReportPartEditSubmit,
	}
}

func reportFinalEditReaderMCPTools() []string {
	return []string{
		plasmamcp.ToolReportLongFormReaderEditStart,
		plasmamcp.ToolReportLongFormReaderEditRead,
		plasmamcp.ToolReportLongFormReaderEditPatch,
		plasmamcp.ToolReportLongFormReaderEditSubmit,
	}
}

func reportFinalEditWriterMCPTools() []string {
	return []string{
		plasmamcp.ToolReportLongFormFinalWriteStart,
		plasmamcp.ToolReportLongFormFinalWriteRead,
		plasmamcp.ToolReportLongFormFinalWritePatch,
		plasmamcp.ToolReportLongFormFinalWriteSubmit,
	}
}

func reportFinalEditStyleMCPTools() []string {
	return []string{
		plasmamcp.ToolReportLongFormStyleEditStart,
		plasmamcp.ToolReportLongFormStyleEditRead,
		plasmamcp.ToolReportLongFormStyleEditPatch,
		plasmamcp.ToolReportLongFormStyleEditSubmit,
	}
}

func reportFinalEditGateMCPTools() []string {
	return reportFinalEditDisabledGateMCPTools()
}

func reportFinalEditDisabledGateMCPTools() []string {
	return append(reportReadMCPTools(),
		plasmamcp.ToolReportLongFormEditStart,
		plasmamcp.ToolReportLongFormEditRead,
		plasmamcp.ToolReportLongFormEditPatch,
		plasmamcp.ToolReportLongFormEditSubmit,
	)
}

func reportFinalEditSemanticGateMCPTools() []string {
	return append(reportReadMCPTools(),
		plasmamcp.ToolReportLongFormEditStart,
		plasmamcp.ToolReportLongFormEditRead,
		plasmamcp.ToolReportLongFormStyleReviewRead,
		plasmamcp.ToolReportLongFormEditPatch,
		plasmamcp.ToolReportLongFormEditSubmit,
	)
}

func reportFinalEditGateMCPToolsForHumanize(humanize string) []string {
	if humanize == reporting.FinalEditHumanizeEnabled {
		return reportFinalEditSemanticGateMCPTools()
	}
	return reportFinalEditDisabledGateMCPTools()
}

func reportFinalEditStyleSemanticValidationMCPTools() []string {
	return []string{
		plasmamcp.ToolReportLongFormStyleSemanticValidationRead,
		plasmamcp.ToolReportLongFormStyleSemanticValidationSubmit,
	}
}

func reportFinalEditEvidenceGateMCPTools() []string {
	return append(reportReadMCPTools(),
		plasmamcp.ToolReportLongFormEvidenceGateRead,
		plasmamcp.ToolReportLongFormEvidenceGateSubmit,
	)
}

func reportFinalizeMCPTools(profile string) []string {
	if isReportGenerationGuidanceProfileNarrativeContract(profile) {
		return []string{
			plasmamcp.ToolReportLongFormEditStart,
			plasmamcp.ToolReportLongFormEditRead,
			plasmamcp.ToolReportLongFormEditPatch,
			plasmamcp.ToolReportLongFormEditSubmit,
			plasmamcp.ToolMermaidValidate,
		}
	}
	return []string{plasmamcp.ToolReportLongFormFinalize}
}

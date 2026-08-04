package web

import (
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// Report 세션은 승인된 미션 자료만 읽을 수 있다. source discovery와 source-candidate
// 도구는 대화/조사 세션 경계에 남아야 한다.
func reportReadMCPTools() []string {
	return []string{
		mcptools.ToolResearchOutline,
		mcptools.ToolResearchList,
		mcptools.ToolResearchGrep,
		mcptools.ToolResearchRead,
		mcptools.ToolResearchRefs,
		mcptools.ToolMermaidValidate,
		mcptools.ToolSourcesList,
		mcptools.ToolSourcesRead,
		mcptools.ToolSourcesTree,
		mcptools.ToolSourcesGrep,
	}
}

func reportPlanMCPTools() []string {
	return append(reportReadMCPTools(), mcptools.ToolReportPlanSubmit)
}

func reportRequirementMCPTools() []string {
	return []string{
		mcptools.ToolResearchRead,
		mcptools.ToolReportRequirementsSubmit,
	}
}

func reportPartAssemblyMCPTools(profile string) []string {
	tools := []string{
		mcptools.ToolReportPartAssemblyStart,
		mcptools.ToolReportPartAssemblyRead,
		mcptools.ToolReportPartAssemblyPatch,
		mcptools.ToolReportPartAssemblySubmit,
	}
	if reportprompt.IsNarrativeContract(profile) {
		tools = append(tools, mcptools.ToolReportPartSectionRead, mcptools.ToolMermaidValidate)
	}
	return tools
}

func reportPartEditMCPTools() []string {
	return []string{
		mcptools.ToolReportPartEditStart,
		mcptools.ToolReportPartEditRead,
		mcptools.ToolReportPartEditPatch,
		mcptools.ToolReportPartEditSubmit,
	}
}

func reportFinalEditReaderMCPTools() []string {
	return []string{
		mcptools.ToolReportLongFormReaderEditStart,
		mcptools.ToolReportLongFormReaderEditRead,
		mcptools.ToolReportLongFormReaderEditPatch,
		mcptools.ToolReportLongFormReaderEditSubmit,
	}
}

func reportFinalEditWriterMCPTools() []string {
	return []string{
		mcptools.ToolReportLongFormFinalWriteStart,
		mcptools.ToolReportLongFormFinalWriteRead,
		mcptools.ToolReportLongFormFinalWritePatch,
		mcptools.ToolReportLongFormFinalWriteSubmit,
	}
}

func reportFinalEditStyleMCPTools() []string {
	return []string{
		mcptools.ToolReportLongFormStyleEditStart,
		mcptools.ToolReportLongFormStyleEditRead,
		mcptools.ToolReportLongFormStyleEditPatch,
		mcptools.ToolReportLongFormStyleEditSubmit,
	}
}

func reportFinalEditGateMCPTools() []string {
	return reportFinalEditDisabledGateMCPTools()
}

func reportFinalEditDisabledGateMCPTools() []string {
	return append(reportReadMCPTools(),
		mcptools.ToolReportLongFormEditStart,
		mcptools.ToolReportLongFormEditRead,
		mcptools.ToolReportLongFormEditPatch,
		mcptools.ToolReportLongFormEditSubmit,
	)
}

func reportFinalEditSemanticGateMCPTools() []string {
	return append(reportReadMCPTools(),
		mcptools.ToolReportLongFormEditStart,
		mcptools.ToolReportLongFormEditRead,
		mcptools.ToolReportLongFormStyleReviewRead,
		mcptools.ToolReportLongFormEditPatch,
		mcptools.ToolReportLongFormEditSubmit,
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
		mcptools.ToolReportLongFormStyleSemanticValidationRead,
		mcptools.ToolReportLongFormStyleSemanticValidationSubmit,
	}
}

func reportFinalEditEvidenceGateMCPTools() []string {
	return append(reportReadMCPTools(),
		mcptools.ToolReportLongFormEvidenceGateRead,
		mcptools.ToolReportLongFormEvidenceGateSubmit,
	)
}

func reportFinalizeMCPTools(profile string) []string {
	if reportprompt.IsNarrativeContract(profile) {
		return []string{
			mcptools.ToolReportLongFormEditStart,
			mcptools.ToolReportLongFormEditRead,
			mcptools.ToolReportLongFormEditPatch,
			mcptools.ToolReportLongFormEditSubmit,
			mcptools.ToolMermaidValidate,
		}
	}
	return []string{mcptools.ToolReportLongFormFinalize}
}

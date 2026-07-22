package web

import plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"

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

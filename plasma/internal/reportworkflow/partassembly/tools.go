package partassembly

import (
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// ReadMCPTools는 JSON connective assembly 경로의 기존 read-only allowlist다.
func ReadMCPTools() []string {
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

// MCPTools는 Part assembly edit-tool 경로의 allowlist를 profile에 따라 기존 순서로 반환한다.
func MCPTools(profile string) []string {
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

// UseEditTools는 profile이 MCP assembly edit tools를 요구하는지 결정한다.
func UseEditTools(profile string) bool {
	return reportprompt.IsPartAssemblyEditTools(profile) ||
		reportprompt.IsVisualPlan(profile)
}

package plan

import "github.com/c86j224s/liquid2/plasma/internal/mcptools"

// MCPTools는 planned plan 제출 단계의 tool allowlist를 기준선 순서대로 반환한다.
func MCPTools() []string {
	return append(ResearchMCPTools(), mcptools.ToolReportPlanSubmit)
}

// ResearchMCPTools returns the read-only mission tools used when an existing
// canonical plan may be amended but never resubmitted or replaced.
func ResearchMCPTools() []string {
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

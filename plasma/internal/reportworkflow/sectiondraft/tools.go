package sectiondraft

import "github.com/c86j224s/liquid2/plasma/internal/mcptools"

// MCPTools는 Section writer가 읽기 전용 research/source 도구만 쓰도록 하는 기존 allowlist다.
func MCPTools() []string {
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

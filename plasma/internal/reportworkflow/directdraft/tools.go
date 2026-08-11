package directdraft

import "github.com/c86j224s/liquid2/plasma/internal/mcptools"

// ReadMCPTools는 직접 본문 작성 단계에서 허용하는 report-read 도구 순서다.
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

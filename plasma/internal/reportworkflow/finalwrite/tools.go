package finalwrite

import "github.com/c86j224s/liquid2/plasma/internal/mcptools"

// MCPTools는 final writer stage에서 허용되는 도구 순서를 고정한다.
func MCPTools() []string {
	return []string{
		mcptools.ToolReportLongFormFinalWriteStart,
		mcptools.ToolReportLongFormFinalWriteRead,
		mcptools.ToolReportLongFormFinalWritePatch,
		mcptools.ToolReportLongFormFinalWriteSubmit,
	}
}

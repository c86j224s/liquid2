package partedit

import "github.com/c86j224s/liquid2/plasma/internal/mcptools"

// MCPTools는 Part edit/author agent가 사용할 수 있는 dedicated edit tool 순서다.
func MCPTools() []string {
	return []string{
		mcptools.ToolReportPartEditStart,
		mcptools.ToolReportPartEditRead,
		mcptools.ToolReportPartEditPatch,
		mcptools.ToolReportPartEditSubmit,
	}
}

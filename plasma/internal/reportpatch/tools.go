package reportpatch

import "github.com/c86j224s/liquid2/plasma/internal/mcptools"

// MCPTools returns the only MCP tools exposed to the ordinary report patch
// agent, in the order expected by the existing executor bindings.
func MCPTools() []string {
	return []string{
		mcptools.ToolReportPatchStart,
		mcptools.ToolReportPatchRead,
		mcptools.ToolReportPatchApply,
		mcptools.ToolReportPatchFinalize,
	}
}

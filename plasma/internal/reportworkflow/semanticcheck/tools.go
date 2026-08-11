package semanticcheck

import "github.com/c86j224s/liquid2/plasma/internal/mcptools"

// MCPTools는 semantic validation stage의 허용 tool 순서를 고정한다.
func MCPTools() []string {
	return []string{
		mcptools.ToolReportLongFormStyleSemanticValidationRead,
		mcptools.ToolReportLongFormStyleSemanticValidationSubmit,
	}
}

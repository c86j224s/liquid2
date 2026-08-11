package requirements

import "github.com/c86j224s/liquid2/plasma/internal/mcptools"

// MCPTools는 요구사항 mapping agent의 allowlist를 기존 순서 그대로 반환한다.
func MCPTools() []string {
	return []string{
		mcptools.ToolResearchRead,
		mcptools.ToolReportRequirementsSubmit,
	}
}

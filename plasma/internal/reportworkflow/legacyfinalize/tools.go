package legacyfinalize

import (
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// MCPTools는 legacy finalizer의 generation profile별 도구 allowlist를 기존 순서로 반환한다.
func MCPTools(profile string) []string {
	if reportprompt.IsNarrativeContract(profile) {
		return []string{
			mcptools.ToolReportLongFormEditStart,
			mcptools.ToolReportLongFormEditRead,
			mcptools.ToolReportLongFormEditPatch,
			mcptools.ToolReportLongFormEditSubmit,
			mcptools.ToolMermaidValidate,
		}
	}
	return []string{mcptools.ToolReportLongFormFinalize}
}

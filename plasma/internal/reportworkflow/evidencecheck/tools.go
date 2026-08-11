package evidencecheck

import (
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// MCPToolsForKind는 terminal gate kind와 humanize flag의 허용 tool 순서를 반환한다.
func MCPToolsForKind(kind Kind, humanize string) []string {
	if kind == KindEvidence {
		return evidenceMCPTools()
	}
	return gateMCPToolsForHumanize(humanize)
}

func reportReadMCPTools() []string {
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

func disabledGateMCPTools() []string {
	return append(reportReadMCPTools(),
		mcptools.ToolReportLongFormEditStart,
		mcptools.ToolReportLongFormEditRead,
		mcptools.ToolReportLongFormEditPatch,
		mcptools.ToolReportLongFormEditSubmit,
	)
}

func semanticGateMCPTools() []string {
	return append(reportReadMCPTools(),
		mcptools.ToolReportLongFormEditStart,
		mcptools.ToolReportLongFormEditRead,
		mcptools.ToolReportLongFormStyleReviewRead,
		mcptools.ToolReportLongFormEditPatch,
		mcptools.ToolReportLongFormEditSubmit,
	)
}

func gateMCPToolsForHumanize(humanize string) []string {
	if humanize == reporting.FinalEditHumanizeEnabled {
		return semanticGateMCPTools()
	}
	return disabledGateMCPTools()
}

func evidenceMCPTools() []string {
	return append(reportReadMCPTools(),
		mcptools.ToolReportLongFormEvidenceGateRead,
		mcptools.ToolReportLongFormEvidenceGateSubmit,
	)
}

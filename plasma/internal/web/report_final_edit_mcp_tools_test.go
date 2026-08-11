package web

import (
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/evidencecheck"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/finalwrite"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/readeredit"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/semanticcheck"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/styleedit"
)

func reportFinalEditReaderMCPTools() []string {
	return mustFinalEditTools(reporting.FinalEditStageReader, "")
}

func reportFinalEditWriterMCPTools() []string {
	return mustFinalEditTools(reporting.FinalEditStageWriter, "")
}

func reportFinalEditStyleMCPTools() []string {
	return mustFinalEditTools(reporting.FinalEditStageStyle, "")
}

func reportFinalEditGateMCPTools() []string {
	return reportFinalEditDisabledGateMCPTools()
}

func reportFinalEditDisabledGateMCPTools() []string {
	return mustFinalEditTools(reporting.FinalEditStageGate, reporting.FinalEditHumanizeDisabled)
}

func reportFinalEditSemanticGateMCPTools() []string {
	return mustFinalEditTools(reporting.FinalEditStageGate, reporting.FinalEditHumanizeEnabled)
}

func reportFinalEditGateMCPToolsForHumanize(humanize string) []string {
	return mustFinalEditTools(reporting.FinalEditStageGate, humanize)
}

func reportFinalEditStyleSemanticValidationMCPTools() []string {
	return mustFinalEditTools(reporting.FinalEditStageStyleSemanticValidation, "")
}

func reportFinalEditEvidenceGateMCPTools() []string {
	return mustFinalEditTools(reporting.FinalEditStageEvidenceGate, "")
}

func mustFinalEditTools(stage string, humanize string) []string {
	switch stage {
	case reporting.FinalEditStageWriter:
		return finalwrite.MCPTools()
	case reporting.FinalEditStageReader:
		return readeredit.MCPTools()
	case reporting.FinalEditStageStyle:
		return styleedit.MCPTools()
	case reporting.FinalEditStageStyleSemanticValidation:
		return semanticcheck.MCPTools()
	case reporting.FinalEditStageEvidenceGate:
		return evidencecheck.MCPToolsForKind(evidencecheck.KindEvidence, humanize)
	default:
		return evidencecheck.MCPToolsForKind(evidencecheck.KindCorrective, humanize)
	}
}

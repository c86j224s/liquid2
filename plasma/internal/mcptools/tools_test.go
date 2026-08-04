package mcptools_test

import (
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
)

func TestMCPToolNameCompatibilityAliases(t *testing.T) {
	tests := map[string][2]string{
		"ToolMissionGet":                                  {mcp.ToolMissionGet, mcptools.ToolMissionGet},
		"ToolMissionUpdate":                               {mcp.ToolMissionUpdate, mcptools.ToolMissionUpdate},
		"ToolSourcesList":                                 {mcp.ToolSourcesList, mcptools.ToolSourcesList},
		"ToolSourcesRead":                                 {mcp.ToolSourcesRead, mcptools.ToolSourcesRead},
		"ToolSourcesTree":                                 {mcp.ToolSourcesTree, mcptools.ToolSourcesTree},
		"ToolSourcesGrep":                                 {mcp.ToolSourcesGrep, mcptools.ToolSourcesGrep},
		"ToolSourcesSearch":                               {mcp.ToolSourcesSearch, mcptools.ToolSourcesSearch},
		"ToolSourceCandidatesPropose":                     {mcp.ToolSourceCandidatesPropose, mcptools.ToolSourceCandidatesPropose},
		"ToolSourceCandidatesRead":                        {mcp.ToolSourceCandidatesRead, mcptools.ToolSourceCandidatesRead},
		"ToolLocalPathRoots":                              {mcp.ToolLocalPathRoots, mcptools.ToolLocalPathRoots},
		"ToolLocalPathTree":                               {mcp.ToolLocalPathTree, mcptools.ToolLocalPathTree},
		"ToolLocalPathAttach":                             {mcp.ToolLocalPathAttach, mcptools.ToolLocalPathAttach},
		"ToolSourcesRemove":                               {mcp.ToolSourcesRemove, mcptools.ToolSourcesRemove},
		"ToolSourcesRestore":                              {mcp.ToolSourcesRestore, mcptools.ToolSourcesRestore},
		"ToolResearchOutline":                             {mcp.ToolResearchOutline, mcptools.ToolResearchOutline},
		"ToolResearchList":                                {mcp.ToolResearchList, mcptools.ToolResearchList},
		"ToolResearchRead":                                {mcp.ToolResearchRead, mcptools.ToolResearchRead},
		"ToolResearchGrep":                                {mcp.ToolResearchGrep, mcptools.ToolResearchGrep},
		"ToolResearchRefs":                                {mcp.ToolResearchRefs, mcptools.ToolResearchRefs},
		"ToolMermaidValidate":                             {mcp.ToolMermaidValidate, mcptools.ToolMermaidValidate},
		"ToolWorkflowStart":                               {mcp.ToolWorkflowStart, mcptools.ToolWorkflowStart},
		"ToolWorkflowStatus":                              {mcp.ToolWorkflowStatus, mcptools.ToolWorkflowStatus},
		"ToolWorkflowStop":                                {mcp.ToolWorkflowStop, mcptools.ToolWorkflowStop},
		"ToolReportPatchStart":                            {mcp.ToolReportPatchStart, mcptools.ToolReportPatchStart},
		"ToolReportPatchRead":                             {mcp.ToolReportPatchRead, mcptools.ToolReportPatchRead},
		"ToolReportPatchApply":                            {mcp.ToolReportPatchApply, mcptools.ToolReportPatchApply},
		"ToolReportPatchFinalize":                         {mcp.ToolReportPatchFinalize, mcptools.ToolReportPatchFinalize},
		"ToolReportPlanSubmit":                            {mcp.ToolReportPlanSubmit, mcptools.ToolReportPlanSubmit},
		"ToolReportRequirementsSubmit":                    {mcp.ToolReportRequirementsSubmit, mcptools.ToolReportRequirementsSubmit},
		"ToolReportPartAssemblyStart":                     {mcp.ToolReportPartAssemblyStart, mcptools.ToolReportPartAssemblyStart},
		"ToolReportPartAssemblyRead":                      {mcp.ToolReportPartAssemblyRead, mcptools.ToolReportPartAssemblyRead},
		"ToolReportPartSectionRead":                       {mcp.ToolReportPartSectionRead, mcptools.ToolReportPartSectionRead},
		"ToolReportPartAssemblyPatch":                     {mcp.ToolReportPartAssemblyPatch, mcptools.ToolReportPartAssemblyPatch},
		"ToolReportPartAssemblySubmit":                    {mcp.ToolReportPartAssemblySubmit, mcptools.ToolReportPartAssemblySubmit},
		"ToolReportPartEditStart":                         {mcp.ToolReportPartEditStart, mcptools.ToolReportPartEditStart},
		"ToolReportPartEditRead":                          {mcp.ToolReportPartEditRead, mcptools.ToolReportPartEditRead},
		"ToolReportPartEditPatch":                         {mcp.ToolReportPartEditPatch, mcptools.ToolReportPartEditPatch},
		"ToolReportPartEditSubmit":                        {mcp.ToolReportPartEditSubmit, mcptools.ToolReportPartEditSubmit},
		"ToolReportLongFormFinalize":                      {mcp.ToolReportLongFormFinalize, mcptools.ToolReportLongFormFinalize},
		"ToolReportLongFormFinalWriteStart":               {mcp.ToolReportLongFormFinalWriteStart, mcptools.ToolReportLongFormFinalWriteStart},
		"ToolReportLongFormFinalWriteRead":                {mcp.ToolReportLongFormFinalWriteRead, mcptools.ToolReportLongFormFinalWriteRead},
		"ToolReportLongFormFinalWritePatch":               {mcp.ToolReportLongFormFinalWritePatch, mcptools.ToolReportLongFormFinalWritePatch},
		"ToolReportLongFormFinalWriteSubmit":              {mcp.ToolReportLongFormFinalWriteSubmit, mcptools.ToolReportLongFormFinalWriteSubmit},
		"ToolReportLongFormReaderEditStart":               {mcp.ToolReportLongFormReaderEditStart, mcptools.ToolReportLongFormReaderEditStart},
		"ToolReportLongFormReaderEditRead":                {mcp.ToolReportLongFormReaderEditRead, mcptools.ToolReportLongFormReaderEditRead},
		"ToolReportLongFormReaderEditPatch":               {mcp.ToolReportLongFormReaderEditPatch, mcptools.ToolReportLongFormReaderEditPatch},
		"ToolReportLongFormReaderEditSubmit":              {mcp.ToolReportLongFormReaderEditSubmit, mcptools.ToolReportLongFormReaderEditSubmit},
		"ToolReportLongFormStyleEditStart":                {mcp.ToolReportLongFormStyleEditStart, mcptools.ToolReportLongFormStyleEditStart},
		"ToolReportLongFormStyleEditRead":                 {mcp.ToolReportLongFormStyleEditRead, mcptools.ToolReportLongFormStyleEditRead},
		"ToolReportLongFormStyleEditPatch":                {mcp.ToolReportLongFormStyleEditPatch, mcptools.ToolReportLongFormStyleEditPatch},
		"ToolReportLongFormStyleEditSubmit":               {mcp.ToolReportLongFormStyleEditSubmit, mcptools.ToolReportLongFormStyleEditSubmit},
		"ToolReportLongFormEditStart":                     {mcp.ToolReportLongFormEditStart, mcptools.ToolReportLongFormEditStart},
		"ToolReportLongFormEditRead":                      {mcp.ToolReportLongFormEditRead, mcptools.ToolReportLongFormEditRead},
		"ToolReportLongFormStyleReviewRead":               {mcp.ToolReportLongFormStyleReviewRead, mcptools.ToolReportLongFormStyleReviewRead},
		"ToolReportLongFormEditPatch":                     {mcp.ToolReportLongFormEditPatch, mcptools.ToolReportLongFormEditPatch},
		"ToolReportLongFormEditSubmit":                    {mcp.ToolReportLongFormEditSubmit, mcptools.ToolReportLongFormEditSubmit},
		"ToolReportLongFormStyleSemanticValidationRead":   {mcp.ToolReportLongFormStyleSemanticValidationRead, mcptools.ToolReportLongFormStyleSemanticValidationRead},
		"ToolReportLongFormStyleSemanticValidationSubmit": {mcp.ToolReportLongFormStyleSemanticValidationSubmit, mcptools.ToolReportLongFormStyleSemanticValidationSubmit},
		"ToolReportLongFormEvidenceGateRead":              {mcp.ToolReportLongFormEvidenceGateRead, mcptools.ToolReportLongFormEvidenceGateRead},
		"ToolReportLongFormEvidenceGateSubmit":            {mcp.ToolReportLongFormEvidenceGateSubmit, mcptools.ToolReportLongFormEvidenceGateSubmit},
		"ToolExperimentReportCreate":                      {mcp.ToolExperimentReportCreate, mcptools.ToolExperimentReportCreate},
		"ToolExperimentReportAppend":                      {mcp.ToolExperimentReportAppend, mcptools.ToolExperimentReportAppend},
		"ToolExperimentReportRead":                        {mcp.ToolExperimentReportRead, mcptools.ToolExperimentReportRead},
		"ToolExperimentReportFinalize":                    {mcp.ToolExperimentReportFinalize, mcptools.ToolExperimentReportFinalize},
		"ToolSourcesSnapshot":                             {mcp.ToolSourcesSnapshot, mcptools.ToolSourcesSnapshot},
		"ToolEvidencePropose":                             {mcp.ToolEvidencePropose, mcptools.ToolEvidencePropose},
		"ToolQuestionsPropose":                            {mcp.ToolQuestionsPropose, mcptools.ToolQuestionsPropose},
		"ToolClaimsPropose":                               {mcp.ToolClaimsPropose, mcptools.ToolClaimsPropose},
		"ToolClaimConfidence":                             {mcp.ToolClaimConfidence, mcptools.ToolClaimConfidence},
		"ToolProposalsSubmit":                             {mcp.ToolProposalsSubmit, mcptools.ToolProposalsSubmit},
	}
	for name, pair := range tests {
		if pair[0] != pair[1] {
			t.Fatalf("%s compatibility alias changed: mcp=%q mcptools=%q", name, pair[0], pair[1])
		}
	}
}

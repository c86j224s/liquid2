package mcp

import (
	"github.com/c86j224s/liquid2/plasma/internal/mcp/research"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const (
	// ToolMissionGet부터 ToolProposalsSubmit까지는 agent에게 노출되는 MCP tool 이름의
	// 안정적인 wire contract다. 이름 변경은 기존 agent prompt, 실험 기록, 장부 trace
	// 해석을 깨뜨릴 수 있으므로 새 tool을 추가하는 방식으로 확장한다.
	ToolMissionGet                                  = mcptools.ToolMissionGet
	ToolMissionUpdate                               = mcptools.ToolMissionUpdate
	ToolSourcesList                                 = mcptools.ToolSourcesList
	ToolSourcesRead                                 = mcptools.ToolSourcesRead
	ToolSourcesTree                                 = mcptools.ToolSourcesTree
	ToolSourcesGrep                                 = mcptools.ToolSourcesGrep
	ToolSourcesSearch                               = mcptools.ToolSourcesSearch
	ToolSourceCandidatesPropose                     = mcptools.ToolSourceCandidatesPropose
	ToolSourceCandidatesRead                        = mcptools.ToolSourceCandidatesRead
	ToolLocalPathRoots                              = mcptools.ToolLocalPathRoots
	ToolLocalPathTree                               = mcptools.ToolLocalPathTree
	ToolLocalPathAttach                             = mcptools.ToolLocalPathAttach
	ToolSourcesRemove                               = mcptools.ToolSourcesRemove
	ToolSourcesRestore                              = mcptools.ToolSourcesRestore
	ToolResearchOutline                             = mcptools.ToolResearchOutline
	ToolResearchList                                = mcptools.ToolResearchList
	ToolResearchRead                                = mcptools.ToolResearchRead
	ToolResearchGrep                                = mcptools.ToolResearchGrep
	ToolResearchRefs                                = mcptools.ToolResearchRefs
	ToolMermaidValidate                             = mcptools.ToolMermaidValidate
	ToolWorkflowStart                               = mcptools.ToolWorkflowStart
	ToolWorkflowStatus                              = mcptools.ToolWorkflowStatus
	ToolWorkflowStop                                = mcptools.ToolWorkflowStop
	ToolReportPatchStart                            = mcptools.ToolReportPatchStart
	ToolReportPatchRead                             = mcptools.ToolReportPatchRead
	ToolReportPatchApply                            = mcptools.ToolReportPatchApply
	ToolReportPatchFinalize                         = mcptools.ToolReportPatchFinalize
	ToolReportPlanSubmit                            = mcptools.ToolReportPlanSubmit
	ToolReportRequirementsSubmit                    = mcptools.ToolReportRequirementsSubmit
	ToolReportPartAssemblyStart                     = mcptools.ToolReportPartAssemblyStart
	ToolReportPartAssemblyRead                      = mcptools.ToolReportPartAssemblyRead
	ToolReportPartSectionRead                       = mcptools.ToolReportPartSectionRead
	ToolReportPartAssemblyPatch                     = mcptools.ToolReportPartAssemblyPatch
	ToolReportPartAssemblySubmit                    = mcptools.ToolReportPartAssemblySubmit
	ToolReportPartEditStart                         = mcptools.ToolReportPartEditStart
	ToolReportPartEditRead                          = mcptools.ToolReportPartEditRead
	ToolReportPartEditPatch                         = mcptools.ToolReportPartEditPatch
	ToolReportPartEditSubmit                        = mcptools.ToolReportPartEditSubmit
	ToolReportLongFormFinalize                      = mcptools.ToolReportLongFormFinalize
	ToolReportLongFormFinalWriteStart               = mcptools.ToolReportLongFormFinalWriteStart
	ToolReportLongFormFinalWriteRead                = mcptools.ToolReportLongFormFinalWriteRead
	ToolReportLongFormFinalWritePatch               = mcptools.ToolReportLongFormFinalWritePatch
	ToolReportLongFormFinalWriteSubmit              = mcptools.ToolReportLongFormFinalWriteSubmit
	ToolReportLongFormReaderEditStart               = mcptools.ToolReportLongFormReaderEditStart
	ToolReportLongFormReaderEditRead                = mcptools.ToolReportLongFormReaderEditRead
	ToolReportLongFormReaderEditPatch               = mcptools.ToolReportLongFormReaderEditPatch
	ToolReportLongFormReaderEditSubmit              = mcptools.ToolReportLongFormReaderEditSubmit
	ToolReportLongFormStyleEditStart                = mcptools.ToolReportLongFormStyleEditStart
	ToolReportLongFormStyleEditRead                 = mcptools.ToolReportLongFormStyleEditRead
	ToolReportLongFormStyleEditPatch                = mcptools.ToolReportLongFormStyleEditPatch
	ToolReportLongFormStyleEditSubmit               = mcptools.ToolReportLongFormStyleEditSubmit
	ToolReportLongFormEditStart                     = mcptools.ToolReportLongFormEditStart
	ToolReportLongFormEditRead                      = mcptools.ToolReportLongFormEditRead
	ToolReportLongFormStyleReviewRead               = mcptools.ToolReportLongFormStyleReviewRead
	ToolReportLongFormEditPatch                     = mcptools.ToolReportLongFormEditPatch
	ToolReportLongFormEditSubmit                    = mcptools.ToolReportLongFormEditSubmit
	ToolReportLongFormStyleSemanticValidationRead   = mcptools.ToolReportLongFormStyleSemanticValidationRead
	ToolReportLongFormStyleSemanticValidationSubmit = mcptools.ToolReportLongFormStyleSemanticValidationSubmit
	ToolReportLongFormEvidenceGateRead              = mcptools.ToolReportLongFormEvidenceGateRead
	ToolReportLongFormEvidenceGateSubmit            = mcptools.ToolReportLongFormEvidenceGateSubmit
	ToolExperimentReportCreate                      = mcptools.ToolExperimentReportCreate
	ToolExperimentReportAppend                      = mcptools.ToolExperimentReportAppend
	ToolExperimentReportRead                        = mcptools.ToolExperimentReportRead
	ToolExperimentReportFinalize                    = mcptools.ToolExperimentReportFinalize
	ToolSourcesSnapshot                             = mcptools.ToolSourcesSnapshot
	ToolEvidencePropose                             = mcptools.ToolEvidencePropose
	ToolQuestionsPropose                            = mcptools.ToolQuestionsPropose
	ToolClaimsPropose                               = mcptools.ToolClaimsPropose
	ToolClaimConfidence                             = mcptools.ToolClaimConfidence
	ToolProposalsSubmit                             = mcptools.ToolProposalsSubmit
)

// ToolDefinition은 MCP list_tools 응답에 실리는 tool metadata다.
//
// InputSchema는 이미 정규화된 JSON schema여야 하며, tool handler가 기대하는 request
// shape와 같은 버전으로 유지되어야 한다.
type ToolDefinition = wire.ToolDefinition

// ToolCall은 MCP call_tool request의 최소 envelope이다.
//
// Arguments는 handler별 schema로만 해석한다. 공통 dispatch 단계에서 임의로
// normalize하지 않으면 tool별 validation 오류가 agent에게 정확히 전달된다.
type ToolCall = wire.ToolCall

// ToolResult는 Plasma MCP tool 호출의 안정적인 response envelope이다.
//
// 성공과 실패 모두 같은 envelope을 사용한다. Error가 채워진 경우 Content는
// 보조 정보일 수 있지만 제품 상태 변경 성공을 뜻하지 않는다.
type ToolResult = wire.ToolResult

// ToolError는 agent가 재시도 가능성과 관련 객체를 판단할 수 있게 하는 안전한
// 오류 표현이다.
//
// Message는 사용자/agent에게 노출 가능한 문구여야 하며 credentials, 원문 source
// 본문, provider raw response를 포함하면 안 된다.
type ToolError = wire.ToolError

// ListTools는 현재 Server binding과 feature option에 따라 노출 가능한 tool 목록을
// 계산한다.
//
// tool 노출 여부는 보수적으로 계산한다. binding이 불완전하거나 mode가 맞지 않으면
// tool을 숨겨 agent가 잘못된 state transition을 시도하지 않게 한다.
func (server *Server) ListTools() []ToolDefinition {
	tools := []ToolDefinition{
		{Name: ToolMissionGet, Description: "Read a Plasma mission projection.", InputSchema: schemaMissionGet},
		{Name: ToolMissionUpdate, Description: "Update supplied current mission metadata fields through the shared application service only when the user explicitly requests the edit.", InputSchema: schemaMissionUpdate},
		{Name: ToolSourcesList, Description: "List active Plasma source snapshots for a mission, optionally including soft-removed sources.", InputSchema: schemaSourcesList},
		{Name: ToolSourcesRead, Description: "Read bounded UTF-8 text from a snapshot_only source artifact, extracted text from uploaded/PDF sources, metadata-only output for binary media such as images, or observe a live local_path reference. For live directory local_path sources, pass subpath to read a child file inside the accepted source boundary. Use offset and next_offset to inspect long readable sources in multiple chunks.", InputSchema: schemaSourcesRead},
		{Name: ToolSourcesTree, Description: "Observe a bounded directory tree for an accepted live local_path source snapshot. Optional subpath is scoped inside that source; root_id and absolute filesystem paths are not accepted.", InputSchema: schemaSourcesTree},
		{Name: ToolSourcesGrep, Description: "Search bounded snippets inside an accepted live local_path source snapshot. Optional subpath is scoped inside that source; matches are observations, not source promotion.", InputSchema: schemaSourcesGrep},
		{Name: ToolSourcesSearch, Description: "Search mounted read-only source connectors for possible original materials. Connector failure is a route failure, not a reason to abandon investigation. Search results are candidates for agent judgment; source snapshot creation remains user-reviewed.", InputSchema: schemaSourcesSearch},
		{Name: ToolSourceCandidatesPropose, Description: "Propose one or more original-material URLs as source candidates for user review. This records review candidates and starts background staging so agents can later read staged unapproved candidates. It does not create source snapshots or saved knowledge. When proposing a plasma.sources.search result, copy source_uri into url and title into title so connector names such as Confluence page titles are preserved.", InputSchema: schemaSourceCandidatesPropose},
		{Name: ToolSourceCandidatesRead, Description: "Read a staged unapproved source candidate by URL, proposal event, or artifact id. This is for conversation/research only; staged candidates are not approved source snapshots and are excluded from default report generation.", InputSchema: schemaSourceCandidatesRead},
		{Name: ToolLocalPathRoots, Description: "List configured allowlisted local path roots. Output never includes absolute filesystem paths.", InputSchema: schemaLocalPathRoots},
		{Name: ToolLocalPathTree, Description: "Browse an allowlisted local path root by root_id and relative_path with bounded depth and entry count.", InputSchema: schemaLocalPathTree},
	}
	tools = append(tools, research.Definitions(server.legacyResearchLoop)...)
	tools = append(tools,
		ToolDefinition{Name: ToolMermaidValidate, Description: "Validate Mermaid source with Plasma's server-side preflight rules before showing it to the user. This catches known Mermaid 11.16.0 parse-breaking patterns and compatibility risks; it does not execute a browser render.", InputSchema: schemaMermaidValidate},
		ToolDefinition{Name: ToolWorkflowStart, Description: "Request a bounded Plasma workflow run for the bound mission. This queues work and does not call the provider inside the MCP tool.", InputSchema: schemaWorkflowStart},
		ToolDefinition{Name: ToolWorkflowStatus, Description: "Read shared workflow run status from the mission ledger projection.", InputSchema: schemaWorkflowStatus},
		ToolDefinition{Name: ToolWorkflowStop, Description: "Request that a bounded workflow run stop before the next step.", InputSchema: schemaWorkflowStop},
	)
	if server.operatorSourceMutation {
		tools = append(tools,
			ToolDefinition{Name: ToolLocalPathAttach, Description: "Operator-only: attach an allowlisted local path as a live_reference source for the bound mission without snapshotting file content.", InputSchema: schemaLocalPathAttach},
			ToolDefinition{Name: ToolSourcesRemove, Description: "Operator-only: soft-remove a source snapshot from the active mission source set without deleting stored artifacts.", InputSchema: schemaSourcesRemove},
			ToolDefinition{Name: ToolSourcesRestore, Description: "Operator-only: restore a soft-removed source snapshot to the active mission source set.", InputSchema: schemaSourcesRestore},
		)
	}
	if server.reportPatch {
		tools = append(tools,
			ToolDefinition{Name: ToolReportPatchStart, Description: "Report-session only: open an existing Markdown report artifact for bounded MCP patching without pasting the whole report into the prompt.", InputSchema: schemaReportPatchStart},
			ToolDefinition{Name: ToolReportPatchRead, Description: "Report-session only: read a bounded slice of the in-process patched report draft.", InputSchema: schemaReportPatchRead},
			ToolDefinition{Name: ToolReportPatchApply, Description: "Report-session only: apply a small replace, insert_after, or append operation to the in-process report patch draft.", InputSchema: schemaReportPatchApply},
			ToolDefinition{Name: ToolReportPatchFinalize, Description: "Report-session only: finalize the patched Markdown report draft into a new report artifact version linked to the base artifact.", InputSchema: schemaReportPatchFinalize},
		)
	}
	if server.reportPlanBinding.complete() && server.toolEnabled(ToolReportPlanSubmit) {
		tools = append(tools, ToolDefinition{Name: ToolReportPlanSubmit, Description: "Report-planning session only: validate and durably submit one planned or long-form report plan for runner promotion.", InputSchema: schemaReportPlanSubmit})
	}
	if server.reportRequirementToolAvailable() {
		tools = append(tools, ToolDefinition{Name: ToolReportRequirementsSubmit, Description: "Long-form requirement mapping session only: attach explicit user output requirements to the fixed report outline without changing it.", InputSchema: schemaReportRequirementsSubmit})
	}
	if ValidatePartAssemblyBinding(server.binding, server.partAssemblyBinding) == nil && server.anyPartAssemblyToolEnabled() {
		tools = append(tools,
			ToolDefinition{Name: ToolReportPartAssemblyStart, Description: "Long-form part assembly session only: start a bounded draft for connective Markdown around immutable Section bodies.", InputSchema: schemaReportPartAssemblyStart},
			ToolDefinition{Name: ToolReportPartAssemblyRead, Description: "Long-form part assembly session only: read the current connective draft state.", InputSchema: schemaReportPartAssemblyRead},
			ToolDefinition{Name: ToolReportPartAssemblyPatch, Description: "Long-form part assembly session only: set intro, transition, or closing connective Markdown without editing Section bodies.", InputSchema: schemaReportPartAssemblyPatch},
			ToolDefinition{Name: ToolReportPartAssemblySubmit, Description: "Long-form part assembly session only: durably submit the connective Markdown for server-side part assembly.", InputSchema: schemaReportPartAssemblySubmit},
		)
		if server.partAssemblySectionReadToolEnabled() {
			tools = append(tools, ToolDefinition{Name: ToolReportPartSectionRead, Description: "Long-form part assembly session only: read a bounded slice of one runner-bound immutable Section artifact by its Part-local index.", InputSchema: schemaReportPartSectionRead})
		}
	}
	if ValidatePartEditBinding(server.binding, server.partEditBinding) == nil {
		for _, tool := range []ToolDefinition{
			{Name: ToolReportPartEditStart, Description: "Long-form Part editor only: open one runner-bound assembled Part as an isolated editable draft.", InputSchema: schemaReportPartEditStart},
			{Name: ToolReportPartEditRead, Description: "Long-form Part editor only: read a bounded slice of the isolated Part draft.", InputSchema: schemaReportPartEditRead},
			{Name: ToolReportPartEditPatch, Description: "Long-form Part editor only: apply an exact bounded edit inside one Part without research access or source Part mutation.", InputSchema: schemaReportPartEditPatch},
			{Name: ToolReportPartEditSubmit, Description: "Long-form Part editor only: atomically record the reviewed Part; changed content creates a separate artifact and unchanged content reuses the bound source artifact.", InputSchema: schemaReportPartEditSubmit},
		} {
			if server.partEditToolEnabled(tool.Name) {
				tools = append(tools, tool)
			}
		}
	}
	if !server.finalEditStageBindingSet && server.finalEditConfigErr == nil && ValidateLongFormFinalizeBinding(server.binding, server.longFormFinalizeBinding) == nil && server.toolEnabled(ToolReportLongFormFinalize) {
		tools = append(tools, ToolDefinition{Name: ToolReportLongFormFinalize, Description: "Long-form final session only: atomically assemble and finalize the bound durable report parts.", InputSchema: schemaReportLongFormFinalize})
	}
	switch server.finalEditStageMode() {
	case reporting.FinalEditStageWriter:
		for _, tool := range []ToolDefinition{
			{Name: ToolReportLongFormFinalWriteStart, Description: "Long-form final writer only: open the deterministic final assembly for bounded writing of the report opening, conclusion, Part transitions, and whole-report logic; no research, external facts, or complete Part/Section redesign.", InputSchema: schemaReportLongFormStageEditStart},
			{Name: ToolReportLongFormFinalWriteRead, Description: "Long-form final writer only: read a bounded slice of the in-process final writer draft for report-level connective logic only; no research, external facts, or Part/Section redesign.", InputSchema: schemaReportLongFormStageEditRead},
			{Name: ToolReportLongFormFinalWritePatch, Description: "Long-form final writer only: apply an exact bounded writer edit to the opening, conclusion, Part transitions, or whole-report logic; no research, external facts, or complete Part/Section redesign.", InputSchema: schemaReportLongFormStageEditPatch},
			{Name: ToolReportLongFormFinalWriteSubmit, Description: "Long-form final writer only: durably submit the written final artifact for downstream reader review without canonical finalization; no research, external facts, or complete Part/Section redesign.", InputSchema: schemaReportLongFormStageEditSubmit},
		} {
			if server.finalEditStageToolEnabled(tool.Name) {
				tools = append(tools, tool)
			}
		}
	case reporting.FinalEditStageReader:
		for _, tool := range []ToolDefinition{
			{Name: ToolReportLongFormReaderEditStart, Description: "Long-form reader editor only: open the durable reader-source manuscript for bounded MCP editing.", InputSchema: schemaReportLongFormStageEditStart},
			{Name: ToolReportLongFormReaderEditRead, Description: "Long-form reader editor only: read a bounded slice of the in-process reader edit draft.", InputSchema: schemaReportLongFormStageEditRead},
			{Name: ToolReportLongFormReaderEditPatch, Description: "Long-form reader editor only: apply an exact bounded edit without mutating Part or Section artifacts.", InputSchema: schemaReportLongFormStageEditPatch},
			{Name: ToolReportLongFormReaderEditSubmit, Description: "Long-form reader editor only: durably submit the reviewed reader edit artifact without canonical finalization.", InputSchema: schemaReportLongFormStageEditSubmit},
		} {
			if server.finalEditStageToolEnabled(tool.Name) {
				tools = append(tools, tool)
			}
		}
	case reporting.FinalEditStageStyle:
		for _, tool := range []ToolDefinition{
			{Name: ToolReportLongFormStyleEditStart, Description: "Long-form style editor only: open the reader-reviewed manuscript for bounded style editing.", InputSchema: schemaReportLongFormStageEditStart},
			{Name: ToolReportLongFormStyleEditRead, Description: "Long-form style editor only: read a bounded slice of the in-process style edit draft.", InputSchema: schemaReportLongFormStageEditRead},
			{Name: ToolReportLongFormStyleEditPatch, Description: "Long-form style editor only: apply an exact bounded style edit without canonical finalization.", InputSchema: schemaReportLongFormStageEditPatch},
			{Name: ToolReportLongFormStyleEditSubmit, Description: "Long-form style editor only: durably submit the reviewed style edit artifact without canonical finalization.", InputSchema: schemaReportLongFormStageEditSubmit},
		} {
			if server.finalEditStageToolEnabled(tool.Name) {
				tools = append(tools, tool)
			}
		}
	case reporting.FinalEditStageGate:
		for _, tool := range []ToolDefinition{
			{Name: ToolReportLongFormEditStart, Description: "Long-form corrective gate only: open the reviewed final edit manuscript for bounded MCP correction.", InputSchema: schemaReportLongFormStageEditStart},
			{Name: ToolReportLongFormEditRead, Description: "Long-form corrective gate only: read a bounded slice of the in-process corrective gate draft.", InputSchema: schemaReportLongFormStageEditRead},
			{Name: ToolReportLongFormStyleReviewRead, Description: "Long-form corrective gate only: read bounded changed style paragraphs for semantic acceptance; transient text is not persisted in trace summaries.", InputSchema: schemaReportLongFormStageEditRead},
			{Name: ToolReportLongFormEditPatch, Description: "Long-form corrective gate only: apply an exact bounded correction before canonical finalization.", InputSchema: schemaReportLongFormStageEditPatch},
			{Name: ToolReportLongFormEditSubmit, Description: "Long-form corrective gate only: submit corrective gate findings and canonicalize the final artifact.", InputSchema: schemaReportLongFormGateEditSubmit},
		} {
			if server.finalEditStageToolEnabled(tool.Name) {
				tools = append(tools, tool)
			}
		}
	case reporting.FinalEditStageStyleSemanticValidation:
		for _, tool := range []ToolDefinition{
			{Name: ToolReportLongFormStyleSemanticValidationRead, Description: "Long-form style semantic validation only: read bounded changed reader/style paragraph comparisons; this tool cannot mutate manuscript text.", InputSchema: schemaReportLongFormStageEditRead},
			{Name: ToolReportLongFormStyleSemanticValidationSubmit, Description: "Long-form style semantic validation only: submit per-paragraph accepted_equivalent or rejected_revert_to_reader verdicts; the server constructs the resolved manuscript.", InputSchema: schemaReportLongFormStyleSemanticValidationSubmit},
		} {
			if server.finalEditStageToolEnabled(tool.Name) {
				tools = append(tools, tool)
			}
		}
	case reporting.FinalEditStageEvidenceGate:
		for _, tool := range []ToolDefinition{
			{Name: ToolReportLongFormEvidenceGateRead, Description: "Long-form evidence gate only: read the final report draft for report-to-evidence connection judgment; this tool cannot mutate manuscript text.", InputSchema: schemaReportLongFormStageEditRead},
			{Name: ToolReportLongFormEvidenceGateSubmit, Description: "Long-form evidence gate only: submit connection judgments and canonicalize the exact bound source artifact with zero operations.", InputSchema: schemaReportLongFormEvidenceGateSubmit},
		} {
			if server.finalEditStageToolEnabled(tool.Name) {
				tools = append(tools, tool)
			}
		}
	case "":
		for _, tool := range []ToolDefinition{
			ToolDefinition{Name: ToolReportLongFormEditStart, Description: "Long-form final editor only: create an in-process manuscript from the runner-bound durable Part artifacts.", InputSchema: schemaReportLongFormEditStart},
			ToolDefinition{Name: ToolReportLongFormEditRead, Description: "Long-form final editor only: read a bounded slice of the in-process manuscript.", InputSchema: schemaReportLongFormEditRead},
			ToolDefinition{Name: ToolReportLongFormEditPatch, Description: "Long-form final editor only: apply an exact bounded edit to the in-process manuscript without mutating Part or Section artifacts.", InputSchema: schemaReportLongFormEditPatch},
			ToolDefinition{Name: ToolReportLongFormEditSubmit, Description: "Long-form final editor only: atomically submit the edited manuscript through the canonical long-form finalization boundary.", InputSchema: schemaReportLongFormEditSubmit},
		} {
			if server.longFormEditToolEnabled(tool.Name) {
				tools = append(tools, tool)
			}
		}
	}
	if server.experimentalReportComposition {
		tools = append(tools,
			ToolDefinition{Name: ToolExperimentReportCreate, Description: "EXPERIMENTAL - report composition harness only; not part of the default C1 product flow. Create an in-process Markdown report draft.", InputSchema: schemaExperimentReportCreate},
			ToolDefinition{Name: ToolExperimentReportAppend, Description: "EXPERIMENTAL - report composition harness only; not part of the default C1 product flow. Append Markdown text to an experiment report draft.", InputSchema: schemaExperimentReportAppend},
			ToolDefinition{Name: ToolExperimentReportRead, Description: "EXPERIMENTAL - report composition harness only; not part of the default C1 product flow. Read a bounded slice from an experiment report draft.", InputSchema: schemaExperimentReportRead},
			ToolDefinition{Name: ToolExperimentReportFinalize, Description: "EXPERIMENTAL - report composition harness only; not part of the default C1 product flow. Finalize a Markdown draft into a raw report artifact.", InputSchema: schemaExperimentReportFinalize},
		)
	}
	if server.legacyResearchLoop {
		tools = append(tools, research.LegacyMutationDefinitions()...)
	}
	if len(server.enabledTools) > 0 {
		filtered := tools[:0]
		for _, tool := range tools {
			if server.toolEnabled(tool.Name) {
				filtered = append(filtered, tool)
			}
		}
		tools = filtered
	}
	return tools
}

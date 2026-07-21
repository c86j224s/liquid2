package mcp

import (
	"encoding/json"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	ToolMissionGet               = "plasma.mission.get"
	ToolMissionUpdate            = "plasma.mission.update"
	ToolSourcesList              = "plasma.sources.list"
	ToolSourcesRead              = "plasma.sources.read"
	ToolSourcesTree              = "plasma.sources.tree"
	ToolSourcesGrep              = "plasma.sources.grep"
	ToolSourcesSearch            = "plasma.sources.search"
	ToolSourceCandidatesPropose  = "plasma.sources.candidates.propose"
	ToolSourceCandidatesRead     = "plasma.sources.candidates.read"
	ToolLocalPathRoots           = "plasma.local_path.roots"
	ToolLocalPathTree            = "plasma.local_path.tree"
	ToolLocalPathAttach          = "plasma.local_path.attach"
	ToolSourcesRemove            = "plasma.sources.remove"
	ToolSourcesRestore           = "plasma.sources.restore"
	ToolResearchOutline          = "plasma.research.outline"
	ToolResearchList             = "plasma.research.list"
	ToolResearchRead             = "plasma.research.read"
	ToolResearchGrep             = "plasma.research.grep"
	ToolResearchRefs             = "plasma.research.references"
	ToolMermaidValidate          = "plasma.mermaid.validate"
	ToolWorkflowStart            = "plasma.workflow.start"
	ToolWorkflowStatus           = "plasma.workflow.status"
	ToolWorkflowStop             = "plasma.workflow.stop"
	ToolReportPatchStart         = "plasma.report.patch.start"
	ToolReportPatchRead          = "plasma.report.patch.read"
	ToolReportPatchApply         = "plasma.report.patch.apply"
	ToolReportPatchFinalize      = "plasma.report.patch.finalize"
	ToolReportPlanSubmit         = "plasma.report.plan.submit"
	ToolReportPartAssemblyStart  = "plasma.report.part_assembly.start"
	ToolReportPartAssemblyRead   = "plasma.report.part_assembly.read"
	ToolReportPartAssemblyPatch  = "plasma.report.part_assembly.patch"
	ToolReportPartAssemblySubmit = "plasma.report.part_assembly.submit"
	ToolReportLongFormFinalize   = "plasma.report.long_form.finalize"
	ToolExperimentReportCreate   = "plasma.experiment.report.create"
	ToolExperimentReportAppend   = "plasma.experiment.report.append"
	ToolExperimentReportRead     = "plasma.experiment.report.read"
	ToolExperimentReportFinalize = "plasma.experiment.report.finalize"
	ToolSourcesSnapshot          = "plasma.sources.snapshot"
	ToolEvidencePropose          = "plasma.evidence.propose"
	ToolQuestionsPropose         = "plasma.questions.propose"
	ToolClaimsPropose            = "plasma.claims.propose"
	ToolClaimConfidence          = "plasma.claims.confidence.update"
	ToolProposalsSubmit          = "plasma.proposals.submit"
)

type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type ToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolResult struct {
	ToolName             string          `json:"tool_name"`
	MissionID            string          `json:"mission_id,omitempty"`
	CreatedEventIDs      []string        `json:"created_event_ids,omitempty"`
	ProposalID           string          `json:"proposal_id,omitempty"`
	CreatedRecords       []app.ObjectRef `json:"created_records,omitempty"`
	RequiresUserApproval bool            `json:"requires_user_approval,omitempty"`
	Content              any             `json:"content,omitempty"`
	Error                *ToolError      `json:"error,omitempty"`
	TraceEventID         string          `json:"trace_event_id,omitempty"`
	TraceError           string          `json:"trace_error,omitempty"`
}

type ToolError struct {
	ErrorKind        string   `json:"error_kind"`
	Message          string   `json:"message"`
	Retryable        bool     `json:"retryable"`
	RelatedObjectIDs []string `json:"related_object_ids,omitempty"`
}

func (server *Server) ListTools() []ToolDefinition {
	researchListSchema := schemaResearchList
	researchReadSchema := schemaResearchRead
	researchRefsSchema := schemaResearchRefs
	researchRefsDescription := "Follow forward and backward references between pinned sources, raw artifacts, and ledger events."
	if server.legacyResearchLoop {
		researchListSchema = schemaResearchListLegacy
		researchReadSchema = schemaResearchReadLegacy
		researchRefsSchema = schemaResearchRefsLegacy
		researchRefsDescription = "Follow forward and backward references between sources, raw artifacts, ledger events, and legacy evidence, claims, questions, proposals, and report records."
	}
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
		{Name: ToolResearchOutline, Description: "Outline a mission ledger without returning source bodies or large record arrays.", InputSchema: schemaResearchOutline},
		{Name: ToolResearchList, Description: "List mission ledger objects by kind with enforced cursor and limit paging.", InputSchema: researchListSchema},
		{Name: ToolResearchRead, Description: "Read one mission ledger object or source artifact with bounded bytes and next_offset for long payloads.", InputSchema: researchReadSchema},
		{Name: ToolResearchGrep, Description: "Find candidate snippets across mission ledger objects. Matches are candidates, not evidence or sources until read and referenced.", InputSchema: schemaResearchGrep},
		{Name: ToolResearchRefs, Description: researchRefsDescription, InputSchema: researchRefsSchema},
		{Name: ToolMermaidValidate, Description: "Validate Mermaid source with Plasma's server-side preflight rules before showing it to the user. This catches known Mermaid 11.16.0 parse-breaking patterns and compatibility risks; it does not execute a browser render.", InputSchema: schemaMermaidValidate},
		{Name: ToolWorkflowStart, Description: "Request a bounded Plasma workflow run for the bound mission. This queues work and does not call the provider inside the MCP tool.", InputSchema: schemaWorkflowStart},
		{Name: ToolWorkflowStatus, Description: "Read shared workflow run status from the mission ledger projection.", InputSchema: schemaWorkflowStatus},
		{Name: ToolWorkflowStop, Description: "Request that a bounded workflow run stop before the next step.", InputSchema: schemaWorkflowStop},
	}
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
	if ValidatePartAssemblyBinding(server.binding, server.partAssemblyBinding) == nil && server.anyPartAssemblyToolEnabled() {
		tools = append(tools,
			ToolDefinition{Name: ToolReportPartAssemblyStart, Description: "Long-form part assembly session only: start a bounded draft for connective Markdown around immutable Section bodies.", InputSchema: schemaReportPartAssemblyStart},
			ToolDefinition{Name: ToolReportPartAssemblyRead, Description: "Long-form part assembly session only: read the current connective draft state.", InputSchema: schemaReportPartAssemblyRead},
			ToolDefinition{Name: ToolReportPartAssemblyPatch, Description: "Long-form part assembly session only: set intro, transition, or closing connective Markdown without editing Section bodies.", InputSchema: schemaReportPartAssemblyPatch},
			ToolDefinition{Name: ToolReportPartAssemblySubmit, Description: "Long-form part assembly session only: durably submit the connective Markdown for server-side part assembly.", InputSchema: schemaReportPartAssemblySubmit},
		)
	}
	if ValidateLongFormFinalizeBinding(server.binding, server.longFormFinalizeBinding) == nil && server.toolEnabled(ToolReportLongFormFinalize) {
		tools = append(tools, ToolDefinition{Name: ToolReportLongFormFinalize, Description: "Long-form final session only: atomically assemble and finalize the bound durable report parts.", InputSchema: schemaReportLongFormFinalize})
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
		tools = append(tools,
			ToolDefinition{Name: ToolEvidencePropose, Description: "Propose focused evidence grounded in source snapshots, including facts and useful research signals for later review or reporting.", InputSchema: schemaEvidencePropose},
			ToolDefinition{Name: ToolQuestionsPropose, Description: "Propose a follow-up research question.", InputSchema: schemaQuestionsPropose},
			ToolDefinition{Name: ToolClaimsPropose, Description: "Propose a claim backed by evidence or user assertion.", InputSchema: schemaClaimsPropose},
			ToolDefinition{Name: ToolClaimConfidence, Description: "Record an advisory confidence update for an existing claim when new evidence changes the assessment. This does not approve or reject the claim.", InputSchema: schemaClaimConfidence},
			ToolDefinition{Name: ToolProposalsSubmit, Description: "Submit existing proposed records for user review.", InputSchema: schemaProposalsSubmit},
		)
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

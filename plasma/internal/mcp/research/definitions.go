package research

import (
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
)

// Definitions returns the research tool definitions in the root MCP list order.
func Definitions(legacy bool) []wire.ToolDefinition {
	listSchema := schemaList
	readSchema := schemaRead
	outlineDescription := "Outline the non-report mission research discovery surface without returning source bodies or large record arrays."
	listDescription := "List non-report mission research discovery objects by kind with enforced cursor and limit paging."
	grepDescription := "Find candidate snippets using case-insensitive literal substring search across non-report mission research discovery objects. The entire query must occur contiguously; use one short exact word or phrase per call and split separate concepts into separate searches. Matches are candidates, not evidence or sources until read and referenced."
	refsSchema := schemaRefs
	refsDescription := "Follow forward and backward references between pinned sources, non-report raw artifacts, and non-report ledger events."
	if legacy {
		listSchema = schemaListLegacy
		readSchema = schemaReadLegacy
		outlineDescription = "Outline a mission ledger without returning source bodies or large record arrays."
		listDescription = "List mission ledger objects by kind with enforced cursor and limit paging."
		grepDescription = "Find candidate snippets using case-insensitive literal substring search across mission ledger objects. The entire query must occur contiguously; use one short exact word or phrase per call and split separate concepts into separate searches. Matches are candidates, not evidence or sources until read and referenced."
		refsSchema = schemaRefsLegacy
		refsDescription = "Follow forward and backward references between sources, raw artifacts, ledger events, and legacy evidence, claims, questions, proposals, and report records."
	}
	return []wire.ToolDefinition{
		{Name: mcptools.ToolResearchOutline, Description: outlineDescription, InputSchema: schemaOutline},
		{Name: mcptools.ToolResearchChanges, Description: "List bounded meaningful mission changes after a previously observed ledger sequence. Internal execution telemetry and report workflow events are omitted; use the returned current_sequence as the next cursor, and re-read the outline when resync_required is true.", InputSchema: schemaChanges},
		{Name: mcptools.ToolResearchList, Description: listDescription, InputSchema: listSchema},
		{Name: mcptools.ToolResearchRead, Description: "Read one mission ledger object or source artifact with bounded bytes and next_offset for long payloads.", InputSchema: readSchema},
		{Name: mcptools.ToolResearchGrep, Description: grepDescription, InputSchema: schemaGrep},
		{Name: mcptools.ToolResearchRefs, Description: refsDescription, InputSchema: refsSchema},
	}
}

// LegacyMutationDefinitions returns legacy proposal mutation definitions in the
// existing root MCP list order.
func LegacyMutationDefinitions() []wire.ToolDefinition {
	return []wire.ToolDefinition{
		{Name: mcptools.ToolEvidencePropose, Description: "Propose focused evidence grounded in source snapshots, including facts and useful research signals for later review or reporting.", InputSchema: schemaEvidencePropose},
		{Name: mcptools.ToolQuestionsPropose, Description: "Propose a follow-up research question.", InputSchema: schemaQuestionsPropose},
		{Name: mcptools.ToolClaimsPropose, Description: "Propose a claim backed by evidence or user assertion.", InputSchema: schemaClaimsPropose},
		{Name: mcptools.ToolClaimConfidence, Description: "Record an advisory confidence update for an existing claim when new evidence changes the assessment. This does not approve or reject the claim.", InputSchema: schemaClaimConfidence},
		{Name: mcptools.ToolProposalsSubmit, Description: "Submit existing proposed records for user review.", InputSchema: schemaProposalsSubmit},
	}
}

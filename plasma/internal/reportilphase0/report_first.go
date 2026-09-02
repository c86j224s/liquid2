package reportilphase0

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

// reportFirstAuthorDraft is the complete publishable manuscript. The provider
// chooses the reader-facing structure and wording through MCP; the server adds
// only stable IL identity, citation metadata, and target syntax.
type reportFirstAuthorDraft struct {
	Title    string                    `json:"title"`
	Language string                    `json:"language"`
	Sections []reportFirstSectionDraft `json:"sections"`
}

type reportFirstSectionDraft struct {
	Title  string                  `json:"title"`
	Blocks []reportFirstBlockDraft `json:"blocks"`
}

type reportFirstBlockDraft struct {
	Kind                 string              `json:"kind"`
	Prose                string              `json:"prose,omitempty"`
	Items                []string            `json:"items,omitempty"`
	Code                 string              `json:"code,omitempty"`
	Language             *string             `json:"language,omitempty"`
	Table                *documentTableDraft `json:"table,omitempty"`
	EditorialAccountKeys []string            `json:"editorial_account_keys,omitempty"`
	EvidenceSourceKeys   []string            `json:"evidence_source_keys"`
}

func (draft reportFirstBlockDraft) documentDraft() documentBlockDraft {
	return documentBlockDraft{
		Kind:               draft.Kind,
		Prose:              draft.Prose,
		Items:              append([]string(nil), draft.Items...),
		Code:               draft.Code,
		Language:           draft.Language,
		Table:              draft.Table,
		EvidenceSourceKeys: append([]string(nil), draft.EvidenceSourceKeys...),
	}
}

func runDirectSourceAuthorStage(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	citations map[int]SourceCitation,
	selectionApplied bool,
) (Narrative, Document, reportilcontract.AuthorWorkspaceReceipt, []agentexec.AgentResult, error) {
	contractID, documentID := config.NewID("narrative"), config.NewID("doc")
	request := sourceIsolatedRequest(
		config,
		catalog,
		"il_narrative",
		1,
		directSourceAuthorPrompt(config, catalog, selectionApplied),
		nil,
		0,
	)
	request.ExtraMCPTools = []string{
		reportilcontract.SourceListTool,
		reportilcontract.SourceReadTool,
		reportilcontract.AuthorDocumentStartTool,
		reportilcontract.AuthorDocumentAppendSourceTool,
		reportilcontract.AuthorDocumentReadTool,
		reportilcontract.AuthorDocumentReplaceTool,
		reportilcontract.AuthorDocumentFinalizeTool,
	}
	result, err := config.Provider.Run(ctx, request)
	results := []agentexec.AgentResult{result}
	if err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonTransport,
			cause:  err,
		}
	}
	sourceReceipt, err := config.VerifySourceRead.VerifyReportILSourceRead(
		ctx, config.MissionID, request.ToolSessionID, "il_narrative", catalog,
	)
	if err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  withValidationCode(reportexecution.ProviderValidationCodeSourceReadContract, err),
		}
	}
	if err := validateCompleteSourceRead(catalog, sourceReceipt); err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  withValidationCode(reportexecution.ProviderValidationCodeSourceReadContract, err),
		}
	}
	authored, workspaceReceipt, err := config.AuthorDocuments.ReadReportILAuthorDocument(
		ctx, config.MissionID, request.ToolSessionID, catalog,
	)
	if err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  withValidationCode(reportexecution.ProviderValidationCodeDocumentContract, err),
		}
	}
	if err := reportilcontract.ValidateAuthorDocument(authored, catalog); err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  withValidationCode(reportexecution.ProviderValidationCodeDocumentContract, err),
		}
	}
	narrative, document, err := compileReportFirstAuthorDraftForMode(
		reportFirstDraftFromAuthorDocument(authored), config.AuthoringMode,
		contractID, documentID, config.MissionObjective, catalog, sourceReceipt, citations,
	)
	if err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  err,
		}
	}
	return narrative, document, workspaceReceipt, results, nil
}

func runReportFirstAuthorStage(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	memory reportilcontract.EditorialMemory,
	memoryReceipt reportilcontract.EditorialMemoryReceipt,
	citations map[int]SourceCitation,
	selectionApplied bool,
) (Narrative, Document, reportilcontract.AuthorWorkspaceReceipt, []agentexec.AgentResult, error) {
	contractID, documentID := config.NewID("narrative"), config.NewID("doc")
	request := sourceIsolatedRequest(
		config,
		catalog,
		"il_narrative",
		1,
		reportFirstAuthorPrompt(config, catalog, selectionApplied),
		nil,
		0,
	)
	request.ReportILSources.EditorialMemoryArtifactID = memoryReceipt.ArtifactID
	request.ReportILSources.EditorialMemorySHA256 = memoryReceipt.SHA256
	request.ExtraMCPTools = []string{
		reportilcontract.EditorialMemoryReadTool,
		reportilcontract.AuthorDocumentStartTool,
		reportilcontract.AuthorDocumentAppendTool,
		reportilcontract.AuthorDocumentReadTool,
		reportilcontract.AuthorDocumentReplaceTool,
		reportilcontract.AuthorDocumentFinalizeTool,
	}
	result, err := config.Provider.Run(ctx, request)
	results := []agentexec.AgentResult{result}
	if err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonTransport,
			cause:  err,
		}
	}
	authored, workspaceReceipt, err := config.AuthorDocuments.ReadReportILAuthorDocument(
		ctx,
		config.MissionID,
		request.ToolSessionID,
		catalog,
	)
	if err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause: withValidationCode(
				reportexecution.ProviderValidationCodeDocumentContract,
				err,
			),
		}
	}
	draft := reportFirstDraftFromAuthorDocument(authored)
	if err := reportilcontract.ValidateAuthorDocumentWithMemory(authored, memory, catalog); err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  withValidationCode(reportexecution.ProviderValidationCodeDocumentContract, err),
		}
	}
	narrative, document, err := compileReportFirstAuthorDraftForMode(
		draft,
		config.AuthoringMode,
		contractID,
		documentID,
		config.MissionObjective,
		catalog,
		editorialMemorySourceReadReceipt(catalog),
		citations,
	)
	if err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  err,
		}
	}
	return narrative, document, workspaceReceipt, results, nil
}

func reportFirstDraftFromAuthorDocument(document reportilcontract.AuthorDocument) reportFirstAuthorDraft {
	draft := reportFirstAuthorDraft{Title: document.Title, Language: document.Language}
	sections := append([]reportilcontract.AuthorSection(nil), document.Sections...)
	for _, part := range document.Parts {
		sections = append(sections, part.Sections...)
	}
	for _, section := range sections {
		sectionDraft := reportFirstSectionDraft{Title: section.Title}
		for _, block := range section.Blocks {
			blockDraft := reportFirstBlockDraft{
				Kind: block.Kind, Prose: block.Prose, Items: append([]string(nil), block.Items...),
				Code: block.Code, Language: block.Language,
				EditorialAccountKeys: append([]string(nil), block.EditorialAccountKeys...),
				EvidenceSourceKeys:   append([]string(nil), block.EvidenceSourceKeys...),
			}
			if block.Table != nil {
				table := &documentTableDraft{Caption: block.Table.Caption}
				if len(block.Table.Columns) > 0 {
					table.Column1 = block.Table.Columns[0]
				}
				if len(block.Table.Columns) > 1 {
					table.Column2 = block.Table.Columns[1]
				}
				if len(block.Table.Columns) > 2 {
					table.Column3 = authorStringPointer(block.Table.Columns[2])
				}
				if len(block.Table.Columns) > 3 {
					table.Column4 = authorStringPointer(block.Table.Columns[3])
				}
				for _, row := range block.Table.Rows {
					rowDraft := documentTableRowDraft{}
					if len(row.Cells) > 0 {
						rowDraft.Cell1 = row.Cells[0]
					}
					if len(row.Cells) > 1 {
						rowDraft.Cell2 = row.Cells[1]
					}
					if len(row.Cells) > 2 {
						rowDraft.Cell3 = authorStringPointer(row.Cells[2])
					}
					if len(row.Cells) > 3 {
						rowDraft.Cell4 = authorStringPointer(row.Cells[3])
					}
					table.Rows = append(table.Rows, rowDraft)
				}
				blockDraft.Table = table
			}
			sectionDraft.Blocks = append(sectionDraft.Blocks, blockDraft)
		}
		draft.Sections = append(draft.Sections, sectionDraft)
	}
	return draft
}

func authorStringPointer(value string) *string { return &value }

func compileReportFirstAuthorDraft(
	draft reportFirstAuthorDraft,
	contractID,
	documentID,
	missionObjective string,
	catalog reportilcontract.SourceCatalog,
	receipt reportilcontract.SourceReadReceipt,
	citations map[int]SourceCitation,
) (Narrative, Document, error) {
	return compileReportFirstAuthorDraftForMode(
		draft, AuthoringModeStandard, contractID, documentID,
		missionObjective, catalog, receipt, citations,
	)
}

func compileReportFirstAuthorDraftForMode(
	draft reportFirstAuthorDraft,
	authoringMode,
	contractID,
	documentID,
	missionObjective string,
	catalog reportilcontract.SourceCatalog,
	receipt reportilcontract.SourceReadReceipt,
	citations map[int]SourceCitation,
) (Narrative, Document, error) {
	if !providerDocumentLanguagePattern.MatchString(draft.Language) {
		return Narrative{}, Document{}, withValidationCode(
			reportexecution.ProviderValidationCodeDocumentContract,
			fmt.Errorf("author language is invalid"),
		)
	}
	minimumSections := 2
	if normalizeAuthoringMode(authoringMode) == AuthoringModeLongForm {
		minimumSections = 4
	}
	if len(draft.Sections) < minimumSections || len(draft.Sections) > authorSectionLimit(authoringMode) {
		return Narrative{}, Document{}, withValidationCode(
			reportexecution.ProviderValidationCodeDocumentContract,
			fmt.Errorf("author section inventory is invalid"),
		)
	}
	if err := validateCompleteSourceRead(catalog, receipt); err != nil {
		return Narrative{}, Document{}, withValidationCode(
			reportexecution.ProviderValidationCodeSourceReadContract,
			err,
		)
	}
	if authoredTextContainsSourceKey(draft.Title, catalog) {
		return Narrative{}, Document{}, withValidationCode(
			reportexecution.ProviderValidationCodeReaderInternalMachinery,
			fmt.Errorf("author title exposes an internal source key"),
		)
	}
	document := Document{
		SchemaVersion:       DocumentSchemaVersion,
		PipelineFamily:      PipelineFamily,
		DocumentID:          documentID,
		RevisionID:          "draft",
		NarrativeContractID: contractID,
		Title:               strings.TrimSpace(draft.Title),
		Language:            draft.Language,
		Provenance:          map[string]string{"source": "accepted mission sources"},
	}
	used := map[string]bool{}
	sectionIDs := make([]string, 0, len(draft.Sections))
	for sectionIndex, section := range draft.Sections {
		if strings.TrimSpace(section.Title) == "" || len(section.Blocks) == 0 {
			return Narrative{}, Document{}, withValidationCode(
				reportexecution.ProviderValidationCodeDocumentContract,
				fmt.Errorf("author section is incomplete"),
			)
		}
		if authoredTextContainsSourceKey(section.Title, catalog) {
			return Narrative{}, Document{}, withValidationCode(
				reportexecution.ProviderValidationCodeReaderInternalMachinery,
				fmt.Errorf("author section title exposes an internal source key"),
			)
		}
		if sectionIndex == 0 && section.Blocks[0].Kind != "prose" {
			return Narrative{}, Document{}, withValidationCode(
				reportexecution.ProviderValidationCodeDocumentContract,
				fmt.Errorf("author manuscript must begin with prose"),
			)
		}
		sectionID := serverAuthorSectionID(documentID, sectionIndex, section.Title, used)
		sectionIDs = append(sectionIDs, sectionID)
		document.Blocks = append(document.Blocks, Block{
			NodeID: sectionID,
			Kind:   "section",
			Level:  2,
			Title:  section.Title,
		})
		for blockIndex, source := range section.Blocks {
			if len(source.EvidenceSourceKeys) == 0 {
				return Narrative{}, Document{}, withValidationCode(
					reportexecution.ProviderValidationCodeSourceReadContract,
					fmt.Errorf("author content block requires at least one source citation"),
				)
			}
			block, err := compileDocumentBlockDraft(
				documentID,
				sectionID,
				sectionIndex,
				blockIndex,
				source.documentDraft(),
				used,
			)
			if err != nil {
				return Narrative{}, Document{}, withValidationCode(
					reportexecution.ProviderValidationCodeDocumentContract,
					err,
				)
			}
			if err := compileBlockEvidence(
				&document,
				&block,
				catalog,
				receipt,
				source.EvidenceSourceKeys,
				citations,
			); err != nil {
				return Narrative{}, Document{}, withValidationCode(
					reportexecution.ProviderValidationCodeSourceReadContract,
					err,
				)
			}
			document.Blocks = append(document.Blocks, block)
		}
	}
	sections := readerSections(document)
	opening := readerSectionBoundaryText(sections[0], false)
	conclusion := readerSectionBoundaryText(sections[len(sections)-1], true)
	if opening == "" || conclusion == "" {
		return Narrative{}, Document{}, withValidationCode(
			reportexecution.ProviderValidationCodeDocumentContract,
			fmt.Errorf("author manuscript requires opening and conclusion text"),
		)
	}
	centralQuestion := strings.TrimSpace(missionObjective)
	if centralQuestion == "" {
		centralQuestion = document.Title
	}
	narrative := Narrative{
		SchemaVersion:         NarrativeSchemaVersion,
		ContractID:            contractID,
		DocumentID:            documentID,
		CentralQuestion:       centralQuestion,
		ReaderTakeaway:        conclusion,
		Throughline:           opening,
		ConclusionObligations: []string{conclusion},
		VoiceAndTone:          "Reader-facing authored report",
	}
	for index, section := range draft.Sections {
		narrative.ReaderJourney = append(narrative.ReaderJourney, section.Title)
		narrative.ArgumentArc = append(narrative.ArgumentArc, section.Title)
		narrative.SectionRoles = append(narrative.SectionRoles, SectionRole{
			SectionID:        sectionIDs[index],
			Role:             authorSectionRole(index, len(draft.Sections)),
			QuestionAnswered: section.Title,
		})
	}
	if err := validateAuthorFacingDocument(document, missionObjective); err != nil {
		return Narrative{}, Document{}, err
	}
	if err := validateProductDocument(narrative, document); err != nil {
		return Narrative{}, Document{}, withValidationCode(
			reportexecution.ProviderValidationCodeDocumentContract,
			err,
		)
	}
	return narrative, document, nil
}

func authoredTextContainsSourceKey(value string, catalog reportilcontract.SourceCatalog) bool {
	for _, entry := range catalog.Sources {
		if strings.Contains(value, entry.SourceKey) {
			return true
		}
	}
	return false
}

func editorialMemorySourceReadReceipt(catalog reportilcontract.SourceCatalog) reportilcontract.SourceReadReceipt {
	receipt := reportilcontract.SourceReadReceipt{
		ReadBytesBySource: make(map[string]int, len(catalog.Sources)),
	}
	for _, entry := range catalog.Sources {
		receipt.SourceKeys = append(receipt.SourceKeys, entry.SourceKey)
		receipt.FullyReadSourceKeys = append(receipt.FullyReadSourceKeys, entry.SourceKey)
		receipt.ReadBytesBySource[entry.SourceKey] = entry.ReadableBytes
		receipt.ReturnedContentBytes += entry.ReadableBytes
	}
	return receipt
}

func directSourceAuthorPrompt(
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	selectionApplied bool,
) string {
	objective := strings.TrimSpace(config.MissionObjective)
	if objective == "" {
		objective = strings.TrimSpace(config.Title)
	}
	direction := strings.TrimSpace(config.Direction)
	if direction == "" {
		direction = "No additional direction. Answer the mission objective directly."
	}
	requestedTitle := strings.TrimSpace(config.Title)
	if !reportMachinerySubjectPattern.MatchString(objective) &&
		readerFacingInternalTermPattern.MatchString(requestedTitle) {
		requestedTitle = "Choose a natural reader-facing title from the mission objective and evidence."
	}
	sourcePriority := ""
	if selectionApplied {
		sourcePriority = "- Source selection was applied, so the selected catalog is ordered strongest-first. Use that order to find the best material without organizing the report source by source.\n"
	}
	return fmt.Sprintf(`Write the complete %s report that the reader asked for. This is the unverified IL profile: make one complete authoring pass from the selected frozen sources, then compile the same manuscript to every product format. The report itself must live in the server-owned MCP document workspace; do not return it in your final response.

MISSION OBJECTIVE:
%s

REQUESTED TITLE:
%s

ADDITIONAL DIRECTION:
%s

WORKFLOW:
1. Call plasma.report_il.sources.list with {} to obtain every selected source_key and readable size.
2. Read every selected source completely. For each source_key, call plasma.report_il.sources.read at offset 0 with max_bytes up to 65536, then continue that same source at the exact returned next_offset until truncated is false. Do not skip a listed source, restart one after EOF, or invent keys, offsets, or sizes.
3. Decide the report's natural structure and strongest factual throughline from the material you read.
4. Call plasma.report_il.document.start once with the reader-facing title and language.
5. Build the entire report by calling plasma.report_il.document.append_source once per content block, in final reading order. Bind each block only to the source_keys that directly support it.
6. After assembly, call plasma.report_il.document.read from offset 0 and continue at exact next_offset until truncated is false. Read the complete manuscript once as its intended reader.
7. Use plasma.report_il.document.replace only for an obvious local writing error you notice during that required read. Do not begin a separate fact-check or editorial audit pass.
8. Call plasma.report_il.document.finalize exactly once. Your terminal response may only briefly confirm finalization.

AUTHORING STANDARD:
- Give the subject and central answer in the first body paragraph. Do not begin with a source tour, roadmap, preview, duplicated introduction, or research disclaimer.
- Follow the reader's next question. Use substantial sections and connected paragraphs rather than a heading per detail.
- Preserve useful dates, people, roles, structures, relationships, mechanisms, cases, context, and uncertainty from the sources you read.
%s- Do not organize the report source by source, expose coverage accounting, or mention tools, source keys, prompts, schemas, validators, pipelines, or writing stages.
- Keep caveats once beside the affected claim. End with a sharper subject-level judgment, not a section recap.
- Use prose by default. The first block must be prose.
%s
- This profile intentionally omits the separate editorial-memory, publication-reader, and source-continuity passes. Do not simulate those stages inside the terminal response.

Frozen source catalog SHA-256: %s`, authoringModeLabel(config.AuthoringMode), objective, requestedTitle, direction, sourcePriority, authoringLengthGuidance(config.AuthoringMode), catalog.SHA256)
}

func reportFirstAuthorPrompt(
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	selectionApplied bool,
) string {
	objective := strings.TrimSpace(config.MissionObjective)
	if objective == "" {
		objective = strings.TrimSpace(config.Title)
	}
	direction := strings.TrimSpace(config.Direction)
	if direction == "" {
		direction = "No additional direction. Answer the mission objective directly."
	}
	requestedTitle := strings.TrimSpace(config.Title)
	if !reportMachinerySubjectPattern.MatchString(objective) &&
		readerFacingInternalTermPattern.MatchString(requestedTitle) {
		requestedTitle = "Choose a natural reader-facing title from the mission objective and evidence."
	}
	sourcePriority := ""
	if selectionApplied {
		sourcePriority = "- Source selection was applied, so the selected catalog is ordered strongest-first for this report. Use that order to find the best source for each coverage obligation, not to allocate more prose to one obligation or omit lower-ranked evidence that uniquely serves another.\n"
	}
	return fmt.Sprintf(`Write the complete %s report that the reader asked for. The report itself must live in the server-owned MCP document workspace; do not return the manuscript in your final response.

MISSION OBJECTIVE:
%s

REQUESTED TITLE:
%s

ADDITIONAL DIRECTION:
%s

WORKFLOW:
1. Call plasma.report_il.memory.read with offset 0 and continue at exact next_offset until truncated is false. Read the complete server-owned editorial memory. Raw source bodies are intentionally unavailable in this stage.
2. Decide the report's natural structure and strongest factual throughline from that memory.
3. Call plasma.report_il.document.start once with the reader-facing title and language.
4. Build the entire report by calling plasma.report_il.document.append once per content block, in final reading order. Reuse the exact same section_title for adjacent blocks in one section. Bind every block to the editorial_account_keys it actually realizes; the server derives source citations from those accounts.
5. After the complete report is assembled, call plasma.report_il.document.read from offset 0 and continue at exact next_offset until truncated is false. Read the entire assembled report as its intended reader.
6. Use plasma.report_il.document.replace for any needed exact local correction. Every replacement invalidates the prior full read, so reread from offset 0 through EOF after the final replacement.
7. Call plasma.report_il.document.finalize exactly once. Do not print or reconstruct the manuscript afterward. Your terminal response may only briefly confirm that the workspace was finalized.

AUTHORING STANDARD:
- Give the subject and central answer in the first body paragraph. Do not begin with a source tour, roadmap, preview, duplicated introduction, or research disclaimer.
- Follow the reader's next question. Use substantial sections and connected paragraphs rather than fragmenting the report into a heading per detail.
- Preserve the concrete dates, durations, people, roles, structures, relationships, mechanisms, cases, context, and uncertainty that materially improve the answer. When one editorial account presents connected details, keep them connected in the same passage.
- If editorial accounts materially disagree, state the different accounts together near the affected claim without silently averaging or erasing either account.
%s- Use the smallest set of editorial accounts that directly supports each block. Do not split one account merely to shorten a sentence.
- Do not force every account into the report and do not organize the report account by account.
- Use natural reader language. Translate descriptive terms and institutional functions; avoid mechanical transliteration, mixed-script words, unexplained source-language labels, and false precision.
- Caveats belong once beside the affected claim. Do not turn limitations into the report's throughline.
- End with a sharper subject-level judgment, not a section recap, visitor guidance, or preservation metadata.
- Never mention tools, source keys, prompts, schemas, validators, pipelines, or writing stages in reader-facing text.
- Use prose by default. Use lists, quotations, callouts, code, or compact tables only when they improve understanding. The first block must be prose.
%s
- The assembled MCP workspace is the only authoritative manuscript. Do not rely on a final JSON response to carry, summarize, or repair it.

Frozen source catalog SHA-256: %s`, authoringModeLabel(config.AuthoringMode), objective, requestedTitle, direction, sourcePriority, authoringLengthGuidance(config.AuthoringMode), catalog.SHA256)
}

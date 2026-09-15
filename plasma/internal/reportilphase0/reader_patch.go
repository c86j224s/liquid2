package reportilphase0

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

// ReaderPatchRejectionReason is retained in the manifest wire contract for
// compatibility with earlier experimental runs. MCP workspace finalization is
// now all-or-nothing, so a successful MCP workspace run emits no rejection entries.
type ReaderPatchRejectionReason string

const (
	ReaderPatchRejectionBaseBinding           ReaderPatchRejectionReason = "reader_base_binding"
	ReaderPatchRejectionInventory             ReaderPatchRejectionReason = "reader_patch_inventory"
	ReaderPatchRejectionTargetBinding         ReaderPatchRejectionReason = "reader_target_binding"
	ReaderPatchRejectionLocality              ReaderPatchRejectionReason = "reader_patch_locality"
	ReaderPatchRejectionOrder                 ReaderPatchRejectionReason = "reader_patch_order"
	ReaderPatchRejectionEvidenceBinding       ReaderPatchRejectionReason = "reader_evidence_binding"
	ReaderPatchRejectionStructurePreservation ReaderPatchRejectionReason = "reader_structure_preservation"
	ReaderPatchRejectionDocumentValidation    ReaderPatchRejectionReason = "reader_document_validation"
	ReaderPatchRejectionReaderQuality         ReaderPatchRejectionReason = "reader_quality"
)

// ReaderFinalizationReceipt is content-free product telemetry for the
// publication workspace. ProposedPatches and AcceptedPatches both count exact
// successful MCP replacements; a failed replacement or finalization is a stage
// failure and never becomes partial product output.
type ReaderFinalizationReceipt struct {
	ContinuityPatches  int                           `json:"continuity_patches"`
	PublicationPatches int                           `json:"publication_patches"`
	ProposedPatches    int                           `json:"proposed_patches"`
	AcceptedPatches    int                           `json:"accepted_patches"`
	RejectedPatches    int                           `json:"rejected_patches"`
	Applied            bool                          `json:"applied"`
	Rejections         []ReaderPatchRejectionReceipt `json:"rejections,omitempty"`
}

type ReaderPatchRejectionReceipt struct {
	Reason ReaderPatchRejectionReason `json:"reason"`
	Count  int                        `json:"count"`
}

func (reason ReaderPatchRejectionReason) valid() bool {
	switch reason {
	case ReaderPatchRejectionBaseBinding,
		ReaderPatchRejectionInventory,
		ReaderPatchRejectionTargetBinding,
		ReaderPatchRejectionLocality,
		ReaderPatchRejectionOrder,
		ReaderPatchRejectionEvidenceBinding,
		ReaderPatchRejectionStructurePreservation,
		ReaderPatchRejectionDocumentValidation,
		ReaderPatchRejectionReaderQuality:
		return true
	default:
		return false
	}
}

func validateReaderFinalizationReceipt(receipt ReaderFinalizationReceipt) error {
	if receipt.ContinuityPatches < 0 || receipt.PublicationPatches < 0 ||
		receipt.ProposedPatches < 0 || receipt.AcceptedPatches < 0 ||
		receipt.RejectedPatches < 0 ||
		receipt.AcceptedPatches+receipt.RejectedPatches != receipt.ProposedPatches ||
		receipt.AcceptedPatches != receipt.ContinuityPatches+receipt.PublicationPatches ||
		receipt.Applied != (receipt.AcceptedPatches > 0) {
		return fmt.Errorf("reader finalization counts are invalid")
	}
	rejected := 0
	seen := map[ReaderPatchRejectionReason]bool{}
	for _, rejection := range receipt.Rejections {
		if !rejection.Reason.valid() || rejection.Count <= 0 || seen[rejection.Reason] {
			return fmt.Errorf("reader finalization rejection inventory is invalid")
		}
		seen[rejection.Reason] = true
		rejected += rejection.Count
	}
	if rejected != receipt.RejectedPatches ||
		(receipt.RejectedPatches == 0 && len(receipt.Rejections) != 0) {
		return fmt.Errorf("reader finalization rejection count is invalid")
	}
	return nil
}

func validatePublicationDocumentStructure(original, edited Document) error {
	if original.SchemaVersion != edited.SchemaVersion || original.PipelineFamily != edited.PipelineFamily ||
		original.DocumentID != edited.DocumentID || original.NarrativeContractID != edited.NarrativeContractID ||
		original.Language != edited.Language || len(original.Blocks) != len(edited.Blocks) ||
		!reflect.DeepEqual(original.Assets, edited.Assets) ||
		!reflect.DeepEqual(original.Coverage, edited.Coverage) ||
		!reflect.DeepEqual(original.Provenance, edited.Provenance) ||
		!reflect.DeepEqual(original.Extensions, edited.Extensions) {
		return fmt.Errorf("publication reader changed the document envelope")
	}
	for index := range original.Blocks {
		left, right := original.Blocks[index], edited.Blocks[index]
		if !publicationBlockStructureEqual(left, right) {
			return fmt.Errorf("publication reader changed block %d structure", index)
		}
		if left.Kind == "list" && len(left.Items) != len(right.Items) ||
			left.Kind == "table" && !publicationTableShapeEqual(left.Table, right.Table) {
			return fmt.Errorf("publication reader changed block %d cardinality", index)
		}
	}
	return nil
}

func publicationBlockStructureEqual(left, right Block) bool {
	return left.NodeID == right.NodeID && left.Kind == right.Kind && left.ParentNodeID == right.ParentNodeID &&
		left.Level == right.Level && left.Language == right.Language && left.SemanticRole == right.SemanticRole &&
		equalJSONValue(left.Supports, right.Supports) && equalJSONValue(left.Qualifies, right.Qualifies) &&
		equalJSONValue(left.ContrastsWith, right.ContrastsWith) && equalJSONValue(left.Elaborates, right.Elaborates) &&
		equalJSONValue(left.RefersTo, right.RefersTo) &&
		equalJSONValue(left.RequirementRefs, right.RequirementRefs) && left.PresentationIntent == right.PresentationIntent &&
		equalJSONValue(left.Extension, right.Extension) && equalJSONValue(left.Figure, right.Figure)
}

func publicationTableShapeEqual(left, right *Table) bool {
	if left == nil || right == nil || len(left.Columns) != len(right.Columns) || len(left.Rows) != len(right.Rows) {
		return left == nil && right == nil
	}
	for index := range left.Rows {
		if len(left.Rows[index]) != len(right.Rows[index]) {
			return false
		}
	}
	return true
}

func validatePublicationStructuredPayloads(original, edited Document) error {
	return validateStructuredPayloads(original, edited, false)
}

func validateStructuredPayloads(original, edited Document, continuity bool) error {
	if len(original.Blocks) != len(edited.Blocks) {
		return fmt.Errorf("publication reader changed the document block inventory")
	}
	for index := range original.Blocks {
		left, right := original.Blocks[index], edited.Blocks[index]
		switch left.Kind {
		case "code":
			if left.Code != right.Code || left.Language != right.Language {
				return fmt.Errorf("publication reader changed code block %d", index)
			}
		case "equation":
			if !reflect.DeepEqual(left.Equation, right.Equation) {
				return fmt.Errorf("publication reader changed equation block %d", index)
			}
		case "table":
			if !reflect.DeepEqual(left.Table, right.Table) {
				return fmt.Errorf("publication reader changed table block %d", index)
			}
		case "list":
			if !continuity && !reflect.DeepEqual(left.Items, right.Items) {
				return fmt.Errorf("publication reader changed list block %d", index)
			}
		}
	}
	return nil
}

// runContinuityPatchStage performs a source-faithfulness pass over the document
// it receives. The product runs this after publication editing so the last
// mutable boundary is an exact-anchor-aware account comparison.
func runContinuityPatchStage(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	memory reportilcontract.EditorialMemory,
	memoryReceipt reportilcontract.EditorialMemoryReceipt,
	document Document,
	citations map[int]SourceCitation,
	baseWorkspace reportilcontract.AuthorWorkspaceReceipt,
) (Document, ReaderFinalizationReceipt, reportilcontract.AuthorWorkspaceReceipt, []agentexec.AgentResult, error) {
	request := sourceIsolatedRequest(
		config, catalog, "il_continuity", 1,
		continuityEditorPrompt(config, catalog), nil, 0,
	)
	request.ReportILSources.EditorialMemoryArtifactID = memoryReceipt.ArtifactID
	request.ReportILSources.EditorialMemorySHA256 = memoryReceipt.SHA256
	request.ReportILSources.BaseAuthorArtifactID = baseWorkspace.ArtifactID
	request.ReportILSources.BaseAuthorSHA256 = baseWorkspace.SHA256
	request.ExtraMCPTools = []string{
		reportilcontract.EditorialMemoryReadTool,
		reportilcontract.AuthorDocumentOpenTool,
		reportilcontract.AuthorDocumentReadTool,
		reportilcontract.AuthorDocumentReviseBlockTool,
		reportilcontract.AuthorDocumentFinalizeTool,
	}
	result, err := config.Provider.Run(ctx, request)
	results := []agentexec.AgentResult{result}
	if err != nil {
		return Document{}, ReaderFinalizationReceipt{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{reason: reportexecution.ProviderFailureReasonTransport, cause: err}
	}
	authored, workspaceReceipt, err := config.AuthorDocuments.ReadReportILContinuityDocument(
		ctx, config.MissionID, request.ToolSessionID, catalog,
	)
	if err != nil {
		return Document{}, ReaderFinalizationReceipt{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  withValidationCode(reportexecution.ProviderValidationCodeDocumentContract, err),
		}
	}
	if err := reportilcontract.ValidateAuthorDocumentWithMemory(authored, memory, catalog); err != nil {
		return Document{}, ReaderFinalizationReceipt{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  withValidationCode(reportexecution.ProviderValidationCodeDocumentContract, err),
		}
	}
	edited, err := compileAuthorDocumentCandidate(
		authored, document, config.AuthoringMode, config.MissionObjective, catalog, citations, true,
	)
	if err != nil {
		return Document{}, ReaderFinalizationReceipt{}, reportilcontract.AuthorWorkspaceReceipt{}, results, err
	}
	if err := validateReaderFacingDocument(edited, config.MissionObjective); err != nil {
		return Document{}, ReaderFinalizationReceipt{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{reason: reportexecution.ProviderFailureReasonSemanticValidation, cause: err}
	}
	finalization := readerWorkspaceFinalization("il_continuity", workspaceReceipt)
	return edited, finalization, workspaceReceipt, results, nil
}

// runReaderPatchStage performs one reader-facing publication pass and at most one
// bounded repair pass. Repairable reader-quality failures reopen the finalized
// candidate; document, source, and artifact contract failures remain terminal.
func runReaderPatchStage(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	memory reportilcontract.EditorialMemory,
	memoryReceipt reportilcontract.EditorialMemoryReceipt,
	document Document,
	citations map[int]SourceCitation,
	authorWorkspace reportilcontract.AuthorWorkspaceReceipt,
) (Document, ReaderFinalizationReceipt, reportilcontract.AuthorWorkspaceReceipt, []agentexec.AgentResult, error) {
	baseDocument := document
	baseWorkspace := authorWorkspace
	prompt := publicationReaderPrompt(config, catalog)
	results := make([]agentexec.AgentResult, 0, 2)
	totalReplacements := 0

	for attempt := 1; attempt <= 2; attempt++ {
		request := sourceIsolatedRequest(config, catalog, "il_reader", attempt, prompt, nil, 0)
		request.ReportILSources.EditorialMemoryArtifactID = memoryReceipt.ArtifactID
		request.ReportILSources.EditorialMemorySHA256 = memoryReceipt.SHA256
		request.ReportILSources.BaseAuthorArtifactID = baseWorkspace.ArtifactID
		request.ReportILSources.BaseAuthorSHA256 = baseWorkspace.SHA256
		request.ExtraMCPTools = []string{
			reportilcontract.EditorialMemoryReadTool,
			reportilcontract.AuthorDocumentOpenTool,
			reportilcontract.AuthorDocumentReadTool,
			reportilcontract.AuthorDocumentEditTextTool,
			reportilcontract.AuthorDocumentFinalizeTool,
		}
		result, runErr := config.Provider.Run(ctx, request)
		results = append(results, result)
		if runErr != nil {
			return Document{}, ReaderFinalizationReceipt{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{reason: reportexecution.ProviderFailureReasonTransport, cause: runErr}
		}
		authored, workspaceReceipt, readErr := config.AuthorDocuments.ReadReportILPublicationDocument(
			ctx, config.MissionID, request.ToolSessionID, catalog,
		)
		if readErr != nil {
			return Document{}, ReaderFinalizationReceipt{}, reportilcontract.AuthorWorkspaceReceipt{}, results, readerDocumentContractFailure(readErr)
		}
		if validationErr := reportilcontract.ValidateAuthorDocumentWithMemory(authored, memory, catalog); validationErr != nil {
			return Document{}, ReaderFinalizationReceipt{}, reportilcontract.AuthorWorkspaceReceipt{}, results, readerDocumentContractFailure(validationErr)
		}
		edited, compileErr := compilePublicationAuthorDocumentCandidate(
			authored, baseDocument, config.AuthoringMode, config.MissionObjective, catalog, citations,
		)
		if compileErr != nil {
			code := providerValidationCode(compileErr)
			if attempt == 2 || !repairableReaderValidationCode(code) {
				return Document{}, ReaderFinalizationReceipt{}, reportilcontract.AuthorWorkspaceReceipt{}, results, compileErr
			}
			totalReplacements += workspaceReceipt.Replacements
			baseWorkspace = workspaceReceipt
			prompt = publicationReaderRepairPrompt(config, catalog, code)
			continue
		}
		qualityErr := validateReaderFacingDocument(edited, config.MissionObjective)
		if qualityErr == nil {
			totalReplacements += workspaceReceipt.Replacements
			finalization := readerWorkspaceFinalization("il_reader", workspaceReceipt)
			finalization.PublicationPatches = totalReplacements
			finalization.ProposedPatches = totalReplacements
			finalization.AcceptedPatches = totalReplacements
			finalization.Applied = totalReplacements > 0
			return edited, finalization, workspaceReceipt, results, nil
		}
		code := providerValidationCode(qualityErr)
		if attempt == 2 || !repairableReaderValidationCode(code) {
			return Document{}, ReaderFinalizationReceipt{}, reportilcontract.AuthorWorkspaceReceipt{}, results, &providerStageError{
				reason: reportexecution.ProviderFailureReasonSemanticValidation,
				cause:  qualityErr,
			}
		}
		totalReplacements += workspaceReceipt.Replacements
		baseDocument = edited
		baseWorkspace = workspaceReceipt
		prompt = publicationReaderRepairPrompt(config, catalog, code)
	}
	return Document{}, ReaderFinalizationReceipt{}, reportilcontract.AuthorWorkspaceReceipt{}, results, fmt.Errorf("publication reader attempts exhausted")
}

func readerDocumentContractFailure(err error) error {
	return &providerStageError{
		reason: reportexecution.ProviderFailureReasonSemanticValidation,
		cause:  withValidationCode(reportexecution.ProviderValidationCodeDocumentContract, err),
	}
}

func repairableReaderValidationCode(code reportexecution.ProviderValidationCode) bool {
	switch code {
	case reportexecution.ProviderValidationCodeReaderFacingContent,
		reportexecution.ProviderValidationCodeReaderOpening,
		reportexecution.ProviderValidationCodeReaderAuditVoice,
		reportexecution.ProviderValidationCodeReaderOrdinarySI,
		reportexecution.ProviderValidationCodeReaderProcess,
		reportexecution.ProviderValidationCodeReaderMetadata,
		reportexecution.ProviderValidationCodeReaderInternalMachinery:
		return true
	default:
		return false
	}
}

func compilePublicationAuthorDocument(
	authored reportilcontract.AuthorDocument,
	original Document,
	authoringMode,
	missionObjective string,
	catalog reportilcontract.SourceCatalog,
	citations map[int]SourceCitation,
) (Document, error) {
	edited, err := compilePublicationAuthorDocumentCandidate(
		authored, original, authoringMode, missionObjective, catalog, citations,
	)
	if err != nil {
		return Document{}, err
	}
	if err := validateReaderFacingDocument(edited, missionObjective); err != nil {
		return Document{}, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  err,
		}
	}
	return edited, nil
}

func compilePublicationAuthorDocumentCandidate(
	authored reportilcontract.AuthorDocument,
	original Document,
	authoringMode,
	missionObjective string,
	catalog reportilcontract.SourceCatalog,
	citations map[int]SourceCitation,
) (Document, error) {
	return compileAuthorDocumentCandidate(authored, original, authoringMode, missionObjective, catalog, citations, false)
}

func compileAuthorDocumentCandidate(
	authored reportilcontract.AuthorDocument,
	original Document,
	authoringMode,
	missionObjective string,
	catalog reportilcontract.SourceCatalog,
	citations map[int]SourceCitation,
	continuity bool,
) (Document, error) {
	var edited Document
	var err error
	if authored.SchemaVersion == reportilcontract.LongFormAuthorDocumentSchemaVersion {
		_, edited, err = compileLongFormAuthorDocument(
			authored, original.NarrativeContractID, original.DocumentID,
			missionObjective, catalog, editorialMemorySourceReadReceipt(catalog), citations,
		)
	} else {
		draft := reportFirstDraftFromAuthorDocument(authored)
		_, edited, err = compileReportFirstAuthorDraftForMode(
			draft, authoringMode, original.NarrativeContractID, original.DocumentID,
			missionObjective, catalog, editorialMemorySourceReadReceipt(catalog), citations,
		)
	}
	if err != nil {
		code := providerValidationCode(err)
		if !repairableReaderValidationCode(code) {
			code = reportexecution.ProviderValidationCodeDocumentContract
		}
		return Document{}, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  withValidationCode(code, err),
		}
	}
	if err := validatePublicationDocumentStructure(original, edited); err != nil {
		return Document{}, readerDocumentContractFailure(err)
	}
	if err := validateStructuredPayloads(original, edited, continuity); err != nil {
		return Document{}, readerDocumentContractFailure(err)
	}
	return edited, nil
}

func readerWorkspaceFinalization(stage string, workspace reportilcontract.AuthorWorkspaceReceipt) ReaderFinalizationReceipt {
	receipt := ReaderFinalizationReceipt{
		ProposedPatches: workspace.Replacements,
		AcceptedPatches: workspace.Replacements,
		Applied:         workspace.Replacements > 0,
	}
	switch stage {
	case "il_continuity":
		receipt.ContinuityPatches = workspace.Replacements
	case "il_reader":
		receipt.PublicationPatches = workspace.Replacements
	}
	return receipt
}

func mergeReaderFinalization(left, right ReaderFinalizationReceipt) ReaderFinalizationReceipt {
	return ReaderFinalizationReceipt{
		ContinuityPatches:  left.ContinuityPatches + right.ContinuityPatches,
		PublicationPatches: left.PublicationPatches + right.PublicationPatches,
		ProposedPatches:    left.ProposedPatches + right.ProposedPatches,
		AcceptedPatches:    left.AcceptedPatches + right.AcceptedPatches,
		RejectedPatches:    left.RejectedPatches + right.RejectedPatches,
		Applied:            left.Applied || right.Applied,
		Rejections:         append(append([]ReaderPatchRejectionReceipt(nil), left.Rejections...), right.Rejections...),
	}
}

func continuityEditorPrompt(config ProductConfig, catalog reportilcontract.SourceCatalog) string {
	objective := strings.TrimSpace(config.MissionObjective)
	if objective == "" {
		objective = strings.TrimSpace(config.Title)
	}
	return fmt.Sprintf(`Act only as the final source-account continuity editor for the complete publication-edited report. This is the last mutable product boundary: restore any historical or factual meaning that authoring or publication editing lost, using both each natural-language account and its server-verified exact frozen-source anchors. The manuscript lives only in the server-owned MCP document workspace; do not return prose, JSON, an inventory, or a review in your terminal response.

MISSION OBJECTIVE:
%s

ADDITIONAL DIRECTION:
%s

WORKFLOW:
1. Call plasma.report_il.memory.read with offset 0 and continue at exact next_offset until truncated is false. Read the complete server-owned editorial memory. Raw source bodies are intentionally unavailable in this stage.
2. Call plasma.report_il.document.open with {} exactly once.
3. Call plasma.report_il.document.read from offset 0 and continue at exact next_offset until truncated is false. Read the whole assembled manuscript.
4. Fact-check the manuscript sentence by sentence against every essential memory account and its exact source anchors. A connected account includes the actor, role, action, object or relationship, date, duration, case, and uncertainty preserved together in memory. The exact anchors decide fine-grained relations: who acted directly, who ordered or caused an action, what was attributed as tradition, and how certain the source is. Nearby fragments or a citation alone do not preserve it. Distinct or conflicting accounts must remain separately attributed rather than borrowing one account's actor or role for another account's date or duration.
5. When a factual sentence subtly changes an anchored relationship, call plasma.report_il.document.revise_block with the exact block_key shown by document.read and replace the shortest exact once-only substring inside that sentence that can correct the expression. Preserve the rest of the sentence and block whenever possible. Supply the complete editorial_account_keys realized by the corrected block; the server derives its complete source binding from those accounts.
6. When a material account is missing, passive, fragmented, wrongly attached, silently merged, or contradicts an exact anchor, make only the smallest natural reader-facing repair needed to restore it. Do not rewrite a whole paragraph or section, reorder the report, or replace a sound sentence merely to improve style. A broader local replacement is allowed only when no smaller sentence-level correction can restore the anchored fact. Never restore an incomplete, mixed-script, placeholder-like, or malformed personal name merely because it appears in an account or exact anchor; when no account establishes the complete identity, preserve the supported role, action, and relationship without the name.
7. Do not perform general style polishing, terminology cleanup, transition edits, opening edits, conclusion edits, or unsupported expansion. Do not reintroduce a reader-facing language defect that the publication pass removed. If every factual sentence already preserves the material accounts and anchored relations, make no edit.
8. Every block revision invalidates the prior full read. After the final revision, reread from offset 0 through EOF, recheck each changed sentence against its exact anchor, and confirm that its surrounding passage still reads naturally.
9. Call plasma.report_il.document.finalize exactly once. Your terminal response may only briefly confirm finalization.

An account repair is editorial judgment, not a coverage exercise: do not force every memory account into the report, do not create a source tour, and do not expose tools, account keys, source keys, prompts, schemas, validators, pipelines, or writing stages in reader-facing text.

Frozen source catalog SHA-256: %s`, objective, readerDirection(config), catalog.SHA256) + longFormArticleGuidance(config)
}

func publicationReaderRepairPrompt(config ProductConfig, catalog reportilcontract.SourceCatalog, code reportexecution.ProviderValidationCode) string {
	return publicationReaderPrompt(config, catalog) + fmt.Sprintf(`

BOUNDED REPAIR ATTEMPT:
The previous finalized publication candidate passed its document, source-binding, and artifact contracts but failed the reader-quality check %q. The document opened in this attempt is that exact finalized candidate, not the original author artifact. Apply only the smallest text edit needed for this check, reread the complete result, and finalize once. Do not revisit unrelated prose. %s`, code, readerRepairGuidance(code))
}

func publicationReaderPrompt(config ProductConfig, catalog reportilcontract.SourceCatalog) string {
	objective := strings.TrimSpace(config.MissionObjective)
	if objective == "" {
		objective = strings.TrimSpace(config.Title)
	}
	return fmt.Sprintf(`Read the complete authored report as its publication reader. Your job is the experience of the actual reader, not source auditing. A final continuity editor will compare the result with connected accounts and their exact source anchors after this pass. The manuscript lives only in the server-owned MCP document workspace; do not return prose or JSON in your terminal response.

MISSION OBJECTIVE:
%s

ADDITIONAL DIRECTION:
%s

WORKFLOW:
1. Call plasma.report_il.memory.read with offset 0 and continue at exact next_offset until truncated is false. Read the complete editorial memory so wording corrections remain meaning-bounded. Raw source bodies are intentionally unavailable in this stage.
2. Call plasma.report_il.document.open with {} exactly once.
3. Call plasma.report_il.document.read from offset 0 and continue at exact next_offset until truncated is false. Read the whole assembled manuscript from beginning to end as its intended reader.
4. Use plasma.report_il.document.edit_text only for a concrete reader-facing defect that does not change account meaning or source support: malformed or incomplete names, mixed scripts, inaccurate wording, duplicated framing, weak local transitions, misplaced caveats, source-tour language, roadmap or recap language, or a conclusion that merely repeats sections. Use the exact target_kind and target_key shown by document.read; inside that named value, replace the shortest exact once-only substring that corrects the defect. In a Korean report, remove every Hiragana or Katakana span unless it is indispensable quoted subject matter; when a source gives only an incomplete or malformed personal name and the memory does not establish the full identity, omit the name and retain the supported role, action, and relationship without guessing. Compare adjacent paragraphs inside every prose-like block: if the later paragraph only shortens or restates the preceding paragraph's thesis, delete that recap by replacing the smallest once-only span that leaves the passage grammatical, or recast it only when it adds a genuinely new inference. Remove construction narration such as “이 보고서는…”, “이 Part는…”, “이 Section은…”, and “다음 절에서는…” rather than replacing it with another roadmap. Code, equations, tables, lists, source bindings, and block structure are immutable in this pass. Do not rewrite the report merely to produce an edit.
5. Every replacement invalidates the prior full read. After the final mutation, reread from offset 0 through EOF as a continuous report, checking Korean-script integrity, remaining roadmap language, and adjacent-paragraph repetition explicitly.
6. Call plasma.report_il.document.finalize exactly once. Your terminal response may only briefly confirm finalization.

PUBLICATION STANDARD:
- Preserve the manuscript's source-backed factual meaning, source distinctions, supported detail, structure, block kinds, and citation bindings. Do not add a source-backed account or change source support in this pass.
- The opening must give the subject and central answer immediately, without a source tour, roadmap, preview, duplicated introduction, or research disclaimer.
- Follow the reader's questions with natural transitions and substantial paragraphs. Explain unfamiliar terms without turning the report into an audit record.
- Keep caveats once beside the affected claim. Do not make limitations the report's throughline.
- The conclusion must sharpen the subject-level judgment rather than provide visitor guidance or a section recap.
- Do not mention tools, source keys, prompts, schemas, validators, pipelines, or writing stages in reader-facing text.
- If the report already reads naturally and accurately from beginning to end, make no replacement. Finalize the unchanged workspace after the complete read.

Frozen source catalog SHA-256: %s`, objective, readerDirection(config), catalog.SHA256) + longFormArticleGuidance(config)
}

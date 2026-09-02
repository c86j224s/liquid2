package reportilphase0

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

// readerDraft contains a complete source-grounded edit of the authored manuscript.
// The reader may improve titles and all leaf content, but cannot add, remove, or
// reorder sections and blocks.
type readerDraft struct {
	Title                   string                                        `json:"title"`
	SectionEdits            map[string]readerSectionDraft                 `json:"section_edits"`
	TerminologyEdits        map[string]readerTermDraft                    `json:"terminology_edits"`
	EvidenceSupportCoverage map[string]readerEvidenceSupportCoverageDraft `json:"evidence_support_coverage"`
	PreservationCoverage    map[string]readerPreservationCoverageDraft    `json:"preservation_coverage"`
	Reviewer                string                                        `json:"reviewer"`
	LanguageReview          languageReviewDraft                           `json:"language_review"`
	ReaderReview            readerReviewDraft                             `json:"reader_review"`
}

type readerEvidenceSupportCoverageDraft struct {
	EvidenceReceipt string  `json:"evidence_receipt"`
	SectionKey      string  `json:"section_key"`
	BlockKey        string  `json:"block_key"`
	SupportLevel    string  `json:"support_level"`
	CoverageQuote   *string `json:"coverage_quote"`
}

type readerPreservationCoverageDraft struct {
	EvidenceReceipt string `json:"evidence_receipt"`
	SectionKey      string `json:"section_key"`
	BlockKey        string `json:"block_key"`
}

type languageReviewDraft struct {
	TargetLanguageNatural       bool `json:"target_language_natural"`
	NamesAndTermsPreserved      bool `json:"names_and_terms_preserved"`
	MechanicalTranslationAbsent bool `json:"mechanical_translation_absent"`
	OpeningNotDuplicated        bool `json:"opening_not_duplicated"`
}

type readerReviewDraft struct {
	UnsupportedPronunciationAbsent bool `json:"unsupported_pronunciation_absent"`
	LowValueSourceAliasesAbsent    bool `json:"low_value_source_aliases_absent"`
	ConclusionNotRecap             bool `json:"conclusion_not_recap"`
	ClaimStrengthBounded           bool `json:"claim_strength_bounded"`
	SubjectFirstAnswerPresent      bool `json:"subject_first_answer_present"`
	AuditRecordProseAbsent         bool `json:"audit_record_prose_absent"`
	LowValueSIExplanationsAbsent   bool `json:"low_value_si_explanations_absent"`
	UnfamiliarTermsExplained       bool `json:"unfamiliar_terms_explained"`
}

type readerSectionDraft struct {
	SectionReceipt string                      `json:"section_receipt"`
	Title          string                      `json:"title"`
	Blocks         map[string]readerBlockDraft `json:"blocks"`
}

type readerTermDraft struct {
	TermReceipt          string  `json:"term_receipt"`
	ReaderForm           *string `json:"reader_form"`
	RetainAsTerminology  bool    `json:"retain_as_terminology"`
	PresentSourceAliases bool    `json:"present_source_aliases"`
}

func readerSectionKey(index int) string {
	return fmt.Sprintf("section_%02d", index+1)
}

func readerBlockKey(index int) string {
	return fmt.Sprintf("block_%02d", index+1)
}

func readerTermKey(index int) string {
	return fmt.Sprintf("term_%02d", index+1)
}

func readerTermReceipt(term terminologyDecision) string {
	return SHA256(mustMarshal(term))
}

type readerBlockDraft struct {
	OriginalSHA256 string              `json:"original_sha256"`
	Prose          *string             `json:"prose"`
	Items          []string            `json:"items"`
	Code           *string             `json:"code"`
	Table          *documentTableDraft `json:"table"`
}

func languageReviewSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required": []any{
			"target_language_natural", "names_and_terms_preserved",
			"mechanical_translation_absent", "opening_not_duplicated",
		},
		"properties": map[string]any{
			"target_language_natural":       map[string]any{"type": "boolean", "const": true},
			"names_and_terms_preserved":     map[string]any{"type": "boolean", "const": true},
			"mechanical_translation_absent": map[string]any{"type": "boolean", "const": true},
			"opening_not_duplicated":        map[string]any{"type": "boolean", "const": true},
		},
	}
}

func readerReviewSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required": []any{
			"unsupported_pronunciation_absent", "low_value_source_aliases_absent",
			"conclusion_not_recap", "claim_strength_bounded",
			"subject_first_answer_present", "audit_record_prose_absent",
			"low_value_si_explanations_absent", "unfamiliar_terms_explained",
		},
		"properties": map[string]any{
			"unsupported_pronunciation_absent": map[string]any{"type": "boolean", "const": true},
			"low_value_source_aliases_absent":  map[string]any{"type": "boolean", "const": true},
			"conclusion_not_recap":             map[string]any{"type": "boolean", "const": true},
			"claim_strength_bounded":           map[string]any{"type": "boolean", "const": true},
			"subject_first_answer_present":     map[string]any{"type": "boolean", "const": true},
			"audit_record_prose_absent":        map[string]any{"type": "boolean", "const": true},
			"low_value_si_explanations_absent": map[string]any{"type": "boolean", "const": true},
			"unfamiliar_terms_explained":       map[string]any{"type": "boolean"},
		},
	}
}

func validateLanguageReview(review languageReviewDraft) error {
	if !review.TargetLanguageNatural ||
		!review.NamesAndTermsPreserved ||
		!review.MechanicalTranslationAbsent ||
		!review.OpeningNotDuplicated {
		return fmt.Errorf("language review was not accepted")
	}
	return nil
}

func validateReaderReview(review readerReviewDraft) error {
	if !review.ClaimStrengthBounded {
		return withValidationCode(
			reportexecution.ProviderValidationCodeClaimStrength,
			fmt.Errorf("reader claim-strength review was not accepted"),
		)
	}
	if !review.UnfamiliarTermsExplained {
		return withValidationCode(
			reportexecution.ProviderValidationCodeReaderUnexplainedTerm,
			fmt.Errorf("reader unfamiliar-term review was not accepted"),
		)
	}
	if !review.UnsupportedPronunciationAbsent ||
		!review.LowValueSourceAliasesAbsent ||
		!review.ConclusionNotRecap ||
		!review.SubjectFirstAnswerPresent ||
		!review.AuditRecordProseAbsent ||
		!review.LowValueSIExplanationsAbsent {
		return fmt.Errorf("reader review was not accepted")
	}
	return nil
}

func providerReaderSchemaBytes(document Document, arguments ...any) []byte {
	terms := []terminologyDecision{}
	evidencePackets := []EvidencePacket{}
	for _, argument := range arguments {
		switch value := argument.(type) {
		case []terminologyDecision:
			terms = value
		case []EvidencePacket:
			evidencePackets = value
		}
	}
	sectionProperties := map[string]any{}
	sectionRequired := []any{}
	blockAliases := map[string][2]string{}
	for sectionIndex, section := range readerSections(document) {
		sectionKey := readerSectionKey(sectionIndex)
		blockProperties := map[string]any{}
		blockRequired := []any{}
		for blockIndex, block := range section.Blocks {
			blockKey := readerBlockKey(blockIndex)
			blockProperties[blockKey] = readerBlockDraftSchema(block)
			blockRequired = append(blockRequired, blockKey)
			blockAliases[block.NodeID] = [2]string{sectionKey, blockKey}
		}
		sectionProperties[sectionKey] = map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []any{"section_receipt", "title", "blocks"},
			"properties": map[string]any{
				"section_receipt": map[string]any{"type": "string", "const": readerSectionReceipt(section)},
				"title":           nonblankStringSchema(),
				"blocks": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             blockRequired,
					"properties":           blockProperties,
				},
			},
		}
		sectionRequired = append(sectionRequired, sectionKey)
	}
	termProperties := map[string]any{}
	termRequired := make([]any, 0, len(terms))
	for index, term := range terms {
		key := readerTermKey(index)
		readerFormSchema := nullableSchema(nonblankStringSchema())
		if requiresSourceFormRemoval(document.Language, term) {
			readerFormSchema = map[string]any{"type": "null"}
		} else if term.PreserveSourceForm {
			readerFormSchema = map[string]any{
				"type":  "string",
				"const": term.SourceForm,
			}
		} else if requiresSourceFormReaderForm(term) {
			readerFormSchema = nullableSchema(map[string]any{
				"type":  "string",
				"const": term.SourceForm,
			})
		}
		termProperties[key] = map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required": []any{
				"term_receipt", "reader_form", "retain_as_terminology",
				"present_source_aliases",
			},
			"properties": map[string]any{
				"term_receipt":           map[string]any{"type": "string", "const": readerTermReceipt(term)},
				"reader_form":            readerFormSchema,
				"retain_as_terminology":  map[string]any{"type": "boolean"},
				"present_source_aliases": map[string]any{"type": "boolean"},
			},
		}
		termRequired = append(termRequired, key)
	}
	preservationProperties := map[string]any{}
	preservationRequired := []any{}
	evidenceSupportProperties := map[string]any{}
	evidenceSupportRequired := []any{}
	preservedIndex := 0
	for packetIndex, packet := range evidencePackets {
		evidenceKey := fmt.Sprintf("evidence_%03d", packetIndex+1)
		aliases := blockAliases[packet.AuthorNodeID]
		evidenceSupportProperties[evidenceKey] = map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []any{"evidence_receipt", "section_key", "block_key", "support_level", "coverage_quote"},
			"properties": map[string]any{
				"evidence_receipt": map[string]any{"type": "string", "const": evidenceReceipt(packet)},
				"section_key":      map[string]any{"type": "string", "const": aliases[0]},
				"block_key":        map[string]any{"type": "string", "const": aliases[1]},
				"coverage_quote":   nullableSchema(nonblankStringSchema()),
				"support_level": map[string]any{
					"type": "string", "enum": []any{
						"direct_source_statement", "bounded_inference", "unsupported_removed",
					},
				},
			},
		}
		evidenceSupportRequired = append(evidenceSupportRequired, evidenceKey)
		if !packet.Preserve {
			continue
		}
		preservedIndex++
		key := fmt.Sprintf("preserve_%02d", preservedIndex)
		preservationProperties[key] = map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []any{"evidence_receipt", "section_key", "block_key"},
			"properties": map[string]any{
				"evidence_receipt": map[string]any{"type": "string", "const": evidenceReceipt(packet)},
				"section_key":      map[string]any{"type": "string", "const": aliases[0]},
				"block_key":        map[string]any{"type": "string", "const": aliases[1]},
			},
		}
		preservationRequired = append(preservationRequired, key)
	}
	component := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"title", "section_edits", "terminology_edits", "evidence_support_coverage", "preservation_coverage", "reviewer", "language_review", "reader_review"},
		"properties": map[string]any{
			"title": nonblankStringSchema(),
			"section_edits": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             sectionRequired,
				"properties":           sectionProperties,
			},
			"terminology_edits": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             termRequired,
				"properties":           termProperties,
			},
			"evidence_support_coverage": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             evidenceSupportRequired,
				"properties":           evidenceSupportProperties,
			},
			"preservation_coverage": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             preservationRequired,
				"properties":           preservationProperties,
			},
			"reviewer":        nonblankStringSchema(),
			"language_review": languageReviewSchema(),
			"reader_review":   readerReviewSchema(),
		},
	}
	return marshalProviderSchema(providerSchemaFlow, providerClosedRoot("reader_draft", component))
}

func readerSectionReceipt(section readerSectionContext) string {
	blocks := make([]string, 0, len(section.Blocks))
	for _, block := range section.Blocks {
		blocks = append(blocks, readerBlockSHA256(block))
	}
	return SHA256(mustMarshal(struct {
		Title  string   `json:"title"`
		Blocks []string `json:"blocks"`
	}{Title: section.Title, Blocks: blocks}))
}

func readerBlockSHA256(block Block) string {
	return SHA256(mustMarshal(struct {
		Kind     string    `json:"kind"`
		Prose    string    `json:"prose,omitempty"`
		Items    []string  `json:"items,omitempty"`
		Code     string    `json:"code,omitempty"`
		Table    *Table    `json:"table,omitempty"`
		Equation *Equation `json:"equation,omitempty"`
	}{
		Kind:     block.Kind,
		Prose:    block.Prose,
		Items:    block.Items,
		Code:     block.Code,
		Table:    block.Table,
		Equation: block.Equation,
	}))
}

func readerBlockDraftSchema(block Block) map[string]any {
	null := map[string]any{"type": "null"}
	properties := map[string]any{
		"original_sha256": map[string]any{"type": "string", "const": readerBlockSHA256(block)},
		"prose":           null,
		"items":           null,
		"code":            null,
		"table":           null,
	}
	switch block.Kind {
	case "prose", "quote", "callout":
		properties["prose"] = nonblankStringSchema()
	case "list":
		properties["items"] = map[string]any{"type": "array", "minItems": len(block.Items), "maxItems": len(block.Items), "items": nonblankStringSchema()}
	case "code":
		properties["code"] = nonemptyStringSchema()
	case "table":
		properties["table"] = readerTableDraftSchema(block.Table)
	case "equation":
		// Equations are preserved unchanged by the publication reader. Their
		// notation and source binding are authored in the long-form MCP path.
	}

	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"original_sha256", "prose", "items", "code", "table"},
		"properties":           properties,
	}
}

func readerTableDraftSchema(table *Table) map[string]any {
	if table == nil || len(table.Columns) < 2 || len(table.Columns) > 4 {
		return documentTableDraftSchema()
	}
	width := len(table.Columns)
	tableProperties := map[string]any{"caption": nullableSchema(nonblankStringSchema())}
	rowProperties := map[string]any{}
	tableRequired := []any{"caption", "rows"}
	rowRequired := make([]any, 0, width)
	for column := 1; column <= width; column++ {
		columnName := fmt.Sprintf("column_%d", column)
		cellName := fmt.Sprintf("cell_%d", column)
		tableProperties[columnName] = nonblankStringSchema()
		rowProperties[cellName] = nonblankStringSchema()
		tableRequired = append(tableRequired, columnName)
		rowRequired = append(rowRequired, cellName)
	}
	tableProperties["rows"] = map[string]any{
		"type":     "array",
		"minItems": len(table.Rows),
		"maxItems": len(table.Rows),
		"items": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             rowRequired,
			"properties":           rowProperties,
		},
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             tableRequired,
		"properties":           tableProperties,
	}
}

type readerSectionContext struct {
	SectionID string
	Title     string
	Blocks    []Block
}

func readerSections(document Document) []readerSectionContext {
	sections := []readerSectionContext{}
	byID := map[string]int{}
	for _, block := range document.Blocks {
		if block.Kind == "section" {
			// Level-2 sections are Parts when they own nested level-3 Sections.
			// Only leaf Sections participate in reader edit inventories.
			if block.Level == 2 && documentSectionHasChild(document, block.NodeID) {
				continue
			}
			byID[block.NodeID] = len(sections)
			sections = append(sections, readerSectionContext{SectionID: block.NodeID, Title: block.Title})
			continue
		}
		if index, ok := byID[block.ParentNodeID]; ok {
			sections[index].Blocks = append(sections[index].Blocks, block)
		}
	}
	return sections
}

func documentSectionHasChild(document Document, parentID string) bool {
	for _, block := range document.Blocks {
		if block.Kind == "section" && block.ParentNodeID == parentID {
			return true
		}
	}
	return false
}

func compileReaderDraft(document Document, draft readerDraft, missionObjective string, arguments ...any) (Document, FlowAttestation, error) {
	edited, attestation, _, err := compileReaderDraftWithTerminology(document, draft, missionObjective, arguments...)
	return edited, attestation, err
}

func compileReaderDraftWithTerminology(document Document, draft readerDraft, missionObjective string, arguments ...any) (Document, FlowAttestation, []terminologyDecision, error) {
	terms := []terminologyDecision{}
	evidencePackets := []EvidencePacket{}
	for _, argument := range arguments {
		switch value := argument.(type) {
		case []terminologyDecision:
			terms = value
		case []EvidencePacket:
			evidencePackets = value
		}
	}
	sections := readerSections(document)
	if strings.TrimSpace(draft.Title) == "" || strings.TrimSpace(draft.Reviewer) == "" || len(draft.SectionEdits) != len(sections) {
		return Document{}, FlowAttestation{}, nil, withValidationCode(
			reportexecution.ProviderValidationCodeDocumentContract,
			fmt.Errorf("reader edit inventory changed"),
		)
	}
	if err := validateLanguageReview(draft.LanguageReview); err != nil {
		return Document{}, FlowAttestation{}, nil, withValidationCode(
			reportexecution.ProviderValidationCodeLanguageReview,
			fmt.Errorf("reader %w", err),
		)
	}
	if err := validateReaderReview(draft.ReaderReview); err != nil {
		switch providerValidationCode(err) {
		case reportexecution.ProviderValidationCodeClaimStrength,
			reportexecution.ProviderValidationCodeReaderUnexplainedTerm,
			reportexecution.ProviderValidationCodeSupportedDetail,
			reportexecution.ProviderValidationCodeReportDepth:
			return Document{}, FlowAttestation{}, nil, err
		}
		return Document{}, FlowAttestation{}, nil, withValidationCode(
			reportexecution.ProviderValidationCodeLanguageReview,
			fmt.Errorf("reader %w", err),
		)
	}
	edited := document
	edited.Title = draft.Title
	edited.Blocks = append([]Block(nil), document.Blocks...)
	blockIndex := map[string]int{}
	for index, block := range edited.Blocks {
		blockIndex[block.NodeID] = index
	}
	for sectionIndex, section := range sections {
		sectionEdit, ok := draft.SectionEdits[readerSectionKey(sectionIndex)]
		if !ok || sectionEdit.SectionReceipt != readerSectionReceipt(section) || strings.TrimSpace(sectionEdit.Title) == "" || len(sectionEdit.Blocks) != len(section.Blocks) {
			return Document{}, FlowAttestation{}, nil, fmt.Errorf("reader edit inventory changed")
		}
		edited.Blocks[blockIndex[section.SectionID]].Title = sectionEdit.Title
		for blockEditIndex, original := range section.Blocks {
			candidate, ok := sectionEdit.Blocks[readerBlockKey(blockEditIndex)]
			if !ok || candidate.OriginalSHA256 != readerBlockSHA256(original) {
				return Document{}, FlowAttestation{}, nil, fmt.Errorf("reader edit inventory changed")
			}
			block, err := applyReaderBlockEdit(original, candidate)
			if err != nil {
				return Document{}, FlowAttestation{}, nil, err
			}
			edited.Blocks[blockIndex[original.NodeID]] = block
		}
	}
	if err := validateCanonicalDocumentSchema(edited); err != nil {
		return Document{}, FlowAttestation{}, nil, err
	}
	if err := validateDocument(edited); err != nil {
		return Document{}, FlowAttestation{}, nil, err
	}
	if err := validateReaderDocumentStructure(document, edited); err != nil {
		return Document{}, FlowAttestation{}, nil, err
	}
	if err := validateReaderFacingDocument(edited, missionObjective); err != nil {
		return Document{}, FlowAttestation{}, nil, err
	}
	finalTerms, err := compileReaderTerminology(edited, draft.TerminologyEdits, terms)
	if err != nil {
		return Document{}, FlowAttestation{}, nil, withValidationCode(
			reportexecution.ProviderValidationCodeTerminologyPresentation,
			err,
		)
	}
	evidenceSupportCoverage, err := compileEvidenceSupportCoverage(
		edited,
		draft.EvidenceSupportCoverage,
		evidencePackets,
	)
	if err != nil {
		return Document{}, FlowAttestation{}, nil, err
	}
	preservationCoverage, err := compilePreservationCoverage(
		edited,
		draft.PreservationCoverage,
		evidencePackets,
	)
	if err != nil {
		return Document{}, FlowAttestation{}, nil, err
	}
	return edited, FlowAttestation{
		Reviewer:                draft.Reviewer,
		ReviewedFullManuscript:  true,
		CentralThreadPreserved:  true,
		SectionHandoffsResolved: true,
		OpenLoopsResolved:       true,
		TerminologyContinuous:   true,
		EvidenceSupportCoverage: evidenceSupportCoverage,
		PreservationCoverage:    preservationCoverage,
		Verdict:                 "accept",
	}, finalTerms, nil
}

func applyReaderBlockEdit(original Block, edit readerBlockDraft) (Block, error) {
	edited := original
	switch original.Kind {
	case "prose", "quote", "callout":
		if edit.Prose == nil || edit.Items != nil || edit.Code != nil || edit.Table != nil {
			return Block{}, fmt.Errorf("reader prose edit is invalid")
		}
		edited.Prose = *edit.Prose
	case "list":
		if edit.Prose != nil || len(edit.Items) != len(original.Items) || edit.Code != nil || edit.Table != nil {
			return Block{}, fmt.Errorf("reader list edit is invalid")
		}
		edited.Items = append([]string(nil), edit.Items...)
	case "code":
		if edit.Prose != nil || edit.Items != nil || edit.Code == nil || edit.Table != nil {
			return Block{}, fmt.Errorf("reader code edit is invalid")
		}
		edited.Code = *edit.Code
	case "table":
		if edit.Prose != nil || edit.Items != nil || edit.Code != nil || edit.Table == nil {
			return Block{}, fmt.Errorf("reader table edit is invalid")
		}
		table := edit.Table
		columns := []string{table.Column1, table.Column2}
		if table.Column3 != nil {
			columns = append(columns, *table.Column3)
		}
		if table.Column4 != nil {
			columns = append(columns, *table.Column4)
		}
		if original.Table == nil || len(columns) != len(original.Table.Columns) || len(table.Rows) != len(original.Table.Rows) {
			return Block{}, fmt.Errorf("reader table edit is invalid")
		}
		edited.Table = &Table{Caption: optionalProviderString(table.Caption), Columns: columns}
		for _, row := range table.Rows {
			cells := []string{row.Cell1, row.Cell2}
			if row.Cell3 != nil {
				cells = append(cells, *row.Cell3)
			}
			if row.Cell4 != nil {
				cells = append(cells, *row.Cell4)
			}
			if len(cells) != len(columns) {
				return Block{}, fmt.Errorf("reader table edit is invalid")
			}
			edited.Table.Rows = append(edited.Table.Rows, cells)
		}
	case "equation":
		if edit.Prose != nil || edit.Items != nil || edit.Code != nil || edit.Table != nil || original.Equation == nil {
			return Block{}, fmt.Errorf("reader equation edit is invalid")
		}
	default:
		return Block{}, fmt.Errorf("reader edit contains unsupported block kind")
	}
	return edited, nil
}

func validateReaderDocumentStructure(original, edited Document) error {
	if len(original.Blocks) != len(edited.Blocks) {
		return fmt.Errorf("reader edit changed document structure or evidence semantics")
	}
	left, right := original, edited
	left.Blocks = append([]Block(nil), original.Blocks...)
	right.Blocks = append([]Block(nil), edited.Blocks...)
	left.Title, right.Title = "", ""
	left.RevisionID = right.RevisionID
	for index := range left.Blocks {
		if left.Blocks[index].Kind == "section" {
			left.Blocks[index].Title, right.Blocks[index].Title = "", ""
		}
		left.Blocks[index].Prose, right.Blocks[index].Prose = "", ""
		left.Blocks[index].Items, right.Blocks[index].Items = nil, nil
		left.Blocks[index].Code, right.Blocks[index].Code = "", ""
		left.Blocks[index].Table, right.Blocks[index].Table = nil, nil
	}
	if !equalJSONValue(left, right) {
		return fmt.Errorf("reader edit changed document structure or evidence semantics")
	}
	return nil
}

func equalJSONValue(left, right any) bool {
	return string(mustMarshal(left)) == string(mustMarshal(right))
}

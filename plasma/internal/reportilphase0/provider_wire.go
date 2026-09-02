package reportilphase0

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

var providerDocumentLanguagePattern = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

type documentDraft struct {
	Language string                          `json:"language"`
	Sections map[string]documentSectionDraft `json:"sections"`
}

type documentSectionDraft struct {
	Title  string               `json:"title"`
	Blocks []documentBlockDraft `json:"blocks"`
}

type documentBlockDraft struct {
	Kind               string              `json:"kind"`
	Prose              string              `json:"prose,omitempty"`
	Items              []string            `json:"items,omitempty"`
	Code               string              `json:"code,omitempty"`
	Language           *string             `json:"language,omitempty"`
	Table              *documentTableDraft `json:"table,omitempty"`
	EvidenceSourceKeys []string            `json:"evidence_source_keys"`
	SemanticRole       *string             `json:"semantic_role"`
	PresentationIntent *string             `json:"presentation_intent"`
}

type documentTableDraft struct {
	Caption *string                 `json:"caption"`
	Column1 string                  `json:"column_1"`
	Column2 string                  `json:"column_2"`
	Column3 *string                 `json:"column_3,omitempty"`
	Column4 *string                 `json:"column_4,omitempty"`
	Rows    []documentTableRowDraft `json:"rows"`
}

type documentTableRowDraft struct {
	Cell1 string  `json:"cell_1"`
	Cell2 string  `json:"cell_2"`
	Cell3 *string `json:"cell_3,omitempty"`
	Cell4 *string `json:"cell_4,omitempty"`
}

type flowDraft struct {
	ProseEdits  map[string]string `json:"prose_edits"`
	Attestation flowReviewDraft   `json:"flow_attestation"`
}

type flowReviewDraft struct {
	Reviewer                string `json:"reviewer"`
	ReviewedFullManuscript  bool   `json:"reviewed_full_manuscript"`
	CentralThreadPreserved  bool   `json:"central_thread_preserved"`
	SectionHandoffsResolved bool   `json:"section_handoffs_resolved"`
	OpenLoopsResolved       bool   `json:"open_loops_resolved"`
	TerminologyContinuous   bool   `json:"terminology_continuous"`
	Verdict                 string `json:"verdict"`
}

func providerDocumentDraftSchema(narrative Narrative, catalog reportilcontract.SourceCatalog) map[string]any {
	sectionProperties := make(map[string]any, len(narrative.SectionRoles))
	sectionRequired := make([]any, 0, len(narrative.SectionRoles))
	for _, role := range narrative.SectionRoles {
		sectionProperties[role.SectionID] = map[string]any{"$ref": "#/$defs/document_section_draft"}
		sectionRequired = append(sectionRequired, role.SectionID)
	}
	component := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"language", "sections"},
		"properties": map[string]any{
			"language": map[string]any{"type": "string", "pattern": providerDocumentLanguagePattern.String()},
			"sections": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             sectionRequired,
				"properties":           sectionProperties,
			},
		},
	}
	return map[string]any{
		"$ref": "#/$defs/document_draft",
		"$defs": map[string]any{
			"document_draft":         component,
			"document_section_draft": documentSectionDraftSchema(),
			"document_block_draft":   documentBlockDraftSchema(catalog),
		},
	}
}

func documentSectionDraftSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"title", "blocks"},
		"properties": map[string]any{
			"title":  nonblankStringSchema(),
			"blocks": map[string]any{"type": "array", "minItems": 1, "items": map[string]any{"$ref": "#/$defs/document_block_draft"}},
		},
	}
}

func documentBlockDraftSchema(catalog reportilcontract.SourceCatalog) map[string]any {
	sourceKeys := make([]any, 0, len(catalog.Sources))
	for _, entry := range catalog.Sources {
		sourceKeys = append(sourceKeys, entry.SourceKey)
	}
	evidenceSourceKeys := map[string]any{
		"type":     "array",
		"minItems": 1,
		"items":    map[string]any{"type": "string", "enum": sourceKeys},
	}
	nullableSemanticRole := nullableSchema(map[string]any{"type": "string", "enum": []any{"opening", "context", "claim", "mechanism", "evidence", "example", "contrast", "caveat", "implication", "synthesis", "transition", "conclusion"}})
	nullablePresentation := nullableSchema(map[string]any{"type": "string", "enum": []any{"prose", "supporting", "formal_display", "keep_with_next"}})
	common := func(kind string) map[string]any {
		return map[string]any{
			"kind":                 map[string]any{"type": "string", "const": kind},
			"evidence_source_keys": cloneAny(evidenceSourceKeys),
			"semantic_role":        cloneAny(nullableSemanticRole),
			"presentation_intent":  cloneAny(nullablePresentation),
		}
	}
	variant := func(kind string, payload map[string]any) map[string]any {
		properties := common(kind)
		required := []any{"kind", "evidence_source_keys", "semantic_role", "presentation_intent"}
		for name, schema := range payload {
			properties[name] = schema
			required = append(required, name)
		}
		sort.Slice(required, func(i, j int) bool { return required[i].(string) < required[j].(string) })
		return map[string]any{"type": "object", "additionalProperties": false, "required": required, "properties": properties}
	}
	prose := func(kind string) map[string]any {
		return variant(kind, map[string]any{"prose": nonblankStringSchema()})
	}
	return map[string]any{"anyOf": []any{
		prose("prose"),
		prose("quote"),
		prose("callout"),
		variant("list", map[string]any{"items": map[string]any{"type": "array", "minItems": 1, "items": nonblankStringSchema()}}),
		variant("code", map[string]any{"code": nonemptyStringSchema(), "language": nullableSchema(nonblankStringSchema())}),
		variant("table", map[string]any{"table": documentTableDraftSchema()}),
	}}
}

func documentTableDraftSchema() map[string]any {
	variants := make([]any, 0, 3)
	for width := 2; width <= 4; width++ {
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
		row := map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             rowRequired,
			"properties":           rowProperties,
		}
		tableProperties["rows"] = map[string]any{"type": "array", "minItems": 1, "items": row}
		variants = append(variants, map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             tableRequired,
			"properties":           tableProperties,
		})
	}
	return map[string]any{"anyOf": variants}
}

func providerFlowDraftSchema(document Document) map[string]any {
	proseProperties := map[string]any{}
	proseRequired := []any{}
	for _, block := range document.Blocks {
		if !authoredProseKind(block.Kind) {
			continue
		}
		proseProperties[block.NodeID] = nonblankStringSchema()
		proseRequired = append(proseRequired, block.NodeID)
	}
	component := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"prose_edits", "flow_attestation"},
		"properties": map[string]any{
			"prose_edits": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             proseRequired,
				"properties":           proseProperties,
			},
			"flow_attestation": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required": []any{
					"reviewer", "reviewed_full_manuscript", "central_thread_preserved",
					"section_handoffs_resolved", "open_loops_resolved", "terminology_continuous", "verdict",
				},
				"properties": map[string]any{
					"reviewer":                  nonblankStringSchema(),
					"reviewed_full_manuscript":  map[string]any{"type": "boolean", "const": true},
					"central_thread_preserved":  map[string]any{"type": "boolean", "const": true},
					"section_handoffs_resolved": map[string]any{"type": "boolean", "const": true},
					"open_loops_resolved":       map[string]any{"type": "boolean", "const": true},
					"terminology_continuous":    map[string]any{"type": "boolean", "const": true},
					"verdict":                   map[string]any{"type": "string", "const": "accept"},
				},
			},
		},
	}
	return providerClosedRoot("flow_draft", component)
}

func providerClosedRoot(name string, component map[string]any) map[string]any {
	return map[string]any{
		"$ref":  "#/$defs/" + name,
		"$defs": map[string]any{name: component},
	}
}

func compileDocumentDraft(title string, narrative Narrative, catalog reportilcontract.SourceCatalog, receipt reportilcontract.SourceReadReceipt, draft documentDraft) (Document, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Document{}, fmt.Errorf("document title is blank")
	}
	if !providerDocumentLanguagePattern.MatchString(draft.Language) {
		return Document{}, fmt.Errorf("document language is invalid")
	}
	if err := receipt.Validate(catalog); err != nil {
		return Document{}, err
	}
	if len(draft.Sections) != len(narrative.SectionRoles) {
		return Document{}, fmt.Errorf("document section inventory does not match Narrative")
	}
	document := Document{
		SchemaVersion:       DocumentSchemaVersion,
		PipelineFamily:      PipelineFamily,
		DocumentID:          narrative.DocumentID,
		RevisionID:          "draft",
		NarrativeContractID: narrative.ContractID,
		Title:               title,
		Language:            draft.Language,
		Provenance:          map[string]string{"source": "accepted mission sources"},
	}
	used := make(map[string]bool, len(narrative.SectionRoles))
	for _, role := range narrative.SectionRoles {
		if used[role.SectionID] {
			return Document{}, fmt.Errorf("Narrative section inventory is not unique")
		}
		used[role.SectionID] = true
	}
	for sectionIndex, role := range narrative.SectionRoles {
		section, ok := draft.Sections[role.SectionID]
		if !ok {
			return Document{}, fmt.Errorf("document section inventory does not match Narrative")
		}
		if strings.TrimSpace(section.Title) == "" || len(section.Blocks) == 0 {
			return Document{}, fmt.Errorf("document section is incomplete")
		}
		document.Blocks = append(document.Blocks, Block{NodeID: role.SectionID, Kind: "section", Level: 2, Title: section.Title})
		for blockIndex, source := range section.Blocks {
			block, err := compileDocumentBlockDraft(narrative.DocumentID, role.SectionID, sectionIndex, blockIndex, source, used)
			if err != nil {
				return Document{}, err
			}
			if err := compileBlockEvidence(&document, &block, catalog, receipt, source.EvidenceSourceKeys); err != nil {
				return Document{}, err
			}
			document.Blocks = append(document.Blocks, block)
		}
	}
	if err := validateProductDocument(narrative, document); err != nil {
		return Document{}, err
	}
	return document, nil
}

func compileDocumentBlockDraft(documentID, parent string, sectionIndex, blockIndex int, source documentBlockDraft, used map[string]bool) (Block, error) {
	block := Block{
		NodeID:             serverDocumentNodeID(documentID, parent, sectionIndex, blockIndex, used),
		Kind:               source.Kind,
		ParentNodeID:       parent,
		SemanticRole:       optionalProviderString(source.SemanticRole),
		PresentationIntent: optionalProviderString(source.PresentationIntent),
	}
	switch source.Kind {
	case "prose", "quote", "callout":
		if strings.TrimSpace(source.Prose) == "" || len(source.Items) != 0 || source.Code != "" || source.Language != nil || source.Table != nil {
			return Block{}, fmt.Errorf("authored prose block is invalid")
		}
		block.Prose = source.Prose
	case "list":
		if len(source.Items) == 0 || source.Prose != "" || source.Code != "" || source.Language != nil || source.Table != nil {
			return Block{}, fmt.Errorf("authored list block is invalid")
		}
		for _, item := range source.Items {
			if strings.TrimSpace(item) == "" {
				return Block{}, fmt.Errorf("authored list block is invalid")
			}
		}
		block.Items = append([]string(nil), source.Items...)
	case "code":
		if source.Code == "" || source.Prose != "" || len(source.Items) != 0 || source.Table != nil {
			return Block{}, fmt.Errorf("authored code block is invalid")
		}
		block.Code = source.Code
		block.Language = optionalProviderString(source.Language)
	case "table":
		if source.Table == nil || source.Prose != "" || len(source.Items) != 0 || source.Code != "" || source.Language != nil {
			return Block{}, fmt.Errorf("authored table block is invalid")
		}
		table := source.Table
		columns := []string{table.Column1, table.Column2}
		if table.Column3 != nil {
			columns = append(columns, *table.Column3)
		}
		if table.Column4 != nil {
			columns = append(columns, *table.Column4)
		}
		if len(table.Rows) == 0 || len(columns) < 2 || len(columns) > 4 || table.Column4 != nil && table.Column3 == nil {
			return Block{}, fmt.Errorf("authored table block is invalid")
		}
		for _, column := range columns {
			if strings.TrimSpace(column) == "" {
				return Block{}, fmt.Errorf("authored table block is invalid")
			}
		}
		block.Table = &Table{Caption: optionalProviderString(table.Caption), Columns: columns}
		for _, row := range table.Rows {
			cells := []string{row.Cell1, row.Cell2}
			if row.Cell3 != nil {
				cells = append(cells, *row.Cell3)
			}
			if row.Cell4 != nil {
				cells = append(cells, *row.Cell4)
			}
			if len(cells) != len(columns) || row.Cell4 != nil && row.Cell3 == nil {
				return Block{}, fmt.Errorf("authored table block is invalid")
			}
			for _, cell := range cells {
				if strings.TrimSpace(cell) == "" {
					return Block{}, fmt.Errorf("authored table block is invalid")
				}
			}
			block.Table.Rows = append(block.Table.Rows, cells)
		}
	default:
		return Block{}, fmt.Errorf("authored block kind is unsupported")
	}
	return block, nil
}

func compileBlockEvidence(document *Document, block *Block, catalog reportilcontract.SourceCatalog, receipt reportilcontract.SourceReadReceipt, sourceKeys []string, citations ...map[int]SourceCitation) error {
	if authoredBlockContainsSourceKey(*block, catalog) {
		return fmt.Errorf("authored content leaf exposes an internal source key")
	}
	seen := make(map[string]bool, len(sourceKeys))
	for _, sourceKey := range sourceKeys {
		if seen[sourceKey] {
			continue
		}
		seen[sourceKey] = true
		_, available := catalog.Entry(sourceKey)
		if !available || !receipt.Includes(sourceKey) {
			return fmt.Errorf("authored content leaf cites a source not read in this attempt")
		}
		refID := serverAcceptedSourceReferenceID(catalog, sourceKey)
		block.EvidenceRefs = append(block.EvidenceRefs, refID)
		if !documentHasReference(document.References, refID) {
			ordinal := acceptedSourceOrdinal(catalog, sourceKey)
			visibleLabel := acceptedSourceVisibleLabel(document.Language, ordinal)
			locator := "Frozen source catalog"
			if len(citations) > 0 {
				if citation, ok := citations[0][ordinal]; ok {
					if strings.TrimSpace(citation.VisibleLabel) != "" {
						visibleLabel = strings.TrimSpace(citation.VisibleLabel)
					}
					if citation.URL != "" {
						locator = citation.URL
					}
				}
			}
			document.References = append(document.References, Reference{
				RefID:        refID,
				Kind:         "footnote",
				Target:       fmt.Sprintf("accepted-source:%03d", ordinal),
				VisibleLabel: visibleLabel,
				Locator:      locator,
			})
		}
	}
	return nil
}

func authoredBlockContainsSourceKey(block Block, catalog reportilcontract.SourceCatalog) bool {
	values := []string{block.Prose, block.Code}
	values = append(values, block.Items...)
	if block.Table != nil {
		values = append(values, block.Table.Caption)
		values = append(values, block.Table.Columns...)
		for _, row := range block.Table.Rows {
			values = append(values, row...)
		}
	}
	for _, entry := range catalog.Sources {
		for _, value := range values {
			if strings.Contains(value, entry.SourceKey) {
				return true
			}
		}
	}
	return false
}

func serverAcceptedSourceReferenceID(catalog reportilcontract.SourceCatalog, sourceKey string) string {
	sum := sha256.Sum256([]byte(catalog.SHA256 + "\x00" + sourceKey))
	return "ref.accepted_source." + hex.EncodeToString(sum[:8])
}

func acceptedSourceVisibleLabel(language string, ordinal int) string {
	if strings.EqualFold(language, "ko") || strings.HasPrefix(strings.ToLower(language), "ko-") {
		return fmt.Sprintf("승인 소스 %d", ordinal)
	}
	return fmt.Sprintf("Accepted source %d", ordinal)
}

func acceptedSourceOrdinal(catalog reportilcontract.SourceCatalog, sourceKey string) int {
	for _, entry := range catalog.Sources {
		if entry.SourceKey == sourceKey {
			return entry.AcceptedOrdinal
		}
	}
	return 0
}

func documentHasReference(references []Reference, refID string) bool {
	for _, ref := range references {
		if ref.RefID == refID {
			return true
		}
	}
	return false
}

func serverDocumentNodeID(documentID, sectionID string, sectionIndex, blockIndex int, used map[string]bool) string {
	for salt := 0; ; salt++ {
		preimage := fmt.Sprintf("%s\x00%s\x00%d\x00%d\x00%d", documentID, sectionID, sectionIndex, blockIndex, salt)
		sum := sha256.Sum256([]byte(preimage))
		candidate := "node." + hex.EncodeToString(sum[:])
		if !used[candidate] {
			used[candidate] = true
			return candidate
		}
	}
}

func optionalProviderString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func compileFlowDraft(document Document, draft flowDraft) (flowResponse, error) {
	proseNodes := map[string]bool{}
	for _, block := range document.Blocks {
		if authoredProseKind(block.Kind) {
			proseNodes[block.NodeID] = true
		}
	}
	if len(draft.ProseEdits) != len(proseNodes) {
		return flowResponse{}, fmt.Errorf("flow prose inventory changed")
	}
	if strings.TrimSpace(draft.Attestation.Reviewer) == "" ||
		!draft.Attestation.ReviewedFullManuscript ||
		!draft.Attestation.CentralThreadPreserved ||
		!draft.Attestation.SectionHandoffsResolved ||
		!draft.Attestation.OpenLoopsResolved ||
		!draft.Attestation.TerminologyContinuous ||
		draft.Attestation.Verdict != "accept" {
		return flowResponse{}, fmt.Errorf("whole-manuscript flow review was not accepted")
	}
	edited := document
	edited.Blocks = append([]Block(nil), document.Blocks...)
	for index := range edited.Blocks {
		if !authoredProseKind(edited.Blocks[index].Kind) {
			continue
		}
		text, ok := draft.ProseEdits[edited.Blocks[index].NodeID]
		if !ok || strings.TrimSpace(text) == "" {
			return flowResponse{}, fmt.Errorf("flow prose inventory changed")
		}
		edited.Blocks[index].Prose = text
	}
	if err := validateCanonicalDocumentSchema(edited); err != nil {
		return flowResponse{}, err
	}
	if err := validateDocument(edited); err != nil {
		return flowResponse{}, err
	}
	if err := preserveDocumentStructure(document, edited); err != nil {
		return flowResponse{}, err
	}
	return flowResponse{
		Document: edited,
		Attestation: FlowAttestation{
			Reviewer:                draft.Attestation.Reviewer,
			ReviewedFullManuscript:  true,
			CentralThreadPreserved:  true,
			SectionHandoffsResolved: true,
			OpenLoopsResolved:       true,
			TerminologyContinuous:   true,
			Verdict:                 "accept",
		},
	}, nil
}

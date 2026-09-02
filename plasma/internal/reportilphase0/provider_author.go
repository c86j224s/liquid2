package reportilphase0

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

// authorDraft is one complete reader-facing manuscript. The provider owns the
// title, section order, and authored leaves; the server adds IL identity,
// topology, provenance, and target-specific syntax only after writing ends.
type authorDraft struct {
	Title           string                `json:"title"`
	Language        string                `json:"language"`
	ReaderTakeaway  string                `json:"reader_takeaway"`
	Throughline     string                `json:"throughline"`
	VoiceAndTone    string                `json:"voice_and_tone"`
	Sections        []authorSectionDraft  `json:"sections"`
	EvidencePackets []authorEvidenceDraft `json:"evidence_packets"`
	Terminology     []authorTermDraft     `json:"terminology"`
	LanguageReview  languageReviewDraft   `json:"language_review"`
}

type authorSectionDraft struct {
	Title  string               `json:"title"`
	Blocks []documentBlockDraft `json:"blocks"`
}

type authorEvidenceRepairDraft struct {
	Bindings map[string][]authorEvidenceRepairPacketDraft `json:"bindings"`
}

type authorEvidenceRepairPacketDraft struct {
	Claim         string `json:"claim"`
	SourceReceipt string `json:"source_receipt"`
	SourceExcerpt string `json:"-"`
	Preserve      bool   `json:"preserve"`
}

type authorEvidenceBindingSlot struct {
	Alias      string
	SectionKey string
	BlockKey   string
	SourceKey  string
}

func sourceReceiptSchema() map[string]any {
	return map[string]any{"type": "string", "pattern": `^quote_[a-f0-9]{64}$`}
}

func authorEvidenceDraftSchema(catalog reportilcontract.SourceCatalog) map[string]any {
	sourceKeys := make([]any, 0, len(catalog.Sources))
	for _, entry := range catalog.Sources {
		sourceKeys = append(sourceKeys, entry.SourceKey)
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required": []any{
			"section_key", "block_key", "claim", "source_key", "source_receipt",
			"preserve",
		},
		"properties": map[string]any{
			"section_key":    map[string]any{"type": "string", "pattern": `^section_[0-9]{2}$`},
			"block_key":      map[string]any{"type": "string", "pattern": `^block_[0-9]{2}$`},
			"claim":          nonblankStringSchema(),
			"source_key":     map[string]any{"type": "string", "enum": sourceKeys},
			"source_receipt": sourceReceiptSchema(),
			"preserve":       map[string]any{"type": "boolean"},
		},
	}
}

func providerAuthorEvidenceRepairSchemaBytes(frozen authorDraft) []byte {
	slots := authorEvidenceBindingSlots(frozen)
	bindingProperties := make(map[string]any, len(slots))
	bindingRequired := make([]any, 0, len(slots))
	for _, slot := range slots {
		bindingProperties[slot.Alias] = map[string]any{
			"type":     "array",
			"minItems": 1,
			"maxItems": reportilcontract.MaxSourceReadSpans,
			"items":    map[string]any{"$ref": "#/$defs/author_evidence_repair_packet_draft"},
		}
		bindingRequired = append(bindingRequired, slot.Alias)
	}
	return marshalProviderSchema(providerSchemaAuthorEvidenceRepair, map[string]any{
		"$ref": "#/$defs/author_evidence_repair_draft",
		"$defs": map[string]any{
			"author_evidence_repair_draft": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"bindings"},
				"properties": map[string]any{
					"bindings": map[string]any{
						"type":                 "object",
						"additionalProperties": false,
						"required":             bindingRequired,
						"properties":           bindingProperties,
					},
				},
			},
			"author_evidence_repair_packet_draft": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"claim", "source_receipt", "preserve"},
				"properties": map[string]any{
					"claim":          nonblankStringSchema(),
					"source_receipt": sourceReceiptSchema(),
					"preserve":       map[string]any{"type": "boolean"},
				},
			},
		},
	})
}

func authorEvidenceBindingSlots(frozen authorDraft) []authorEvidenceBindingSlot {
	slots := make([]authorEvidenceBindingSlot, 0)
	for sectionIndex, section := range frozen.Sections {
		for blockIndex, block := range section.Blocks {
			seenSources := map[string]bool{}
			for _, sourceKey := range block.EvidenceSourceKeys {
				if seenSources[sourceKey] {
					continue
				}
				seenSources[sourceKey] = true
				slots = append(slots, authorEvidenceBindingSlot{
					Alias:      fmt.Sprintf("binding_%03d", len(slots)+1),
					SectionKey: fmt.Sprintf("section_%02d", sectionIndex+1),
					BlockKey:   fmt.Sprintf("block_%02d", blockIndex+1),
					SourceKey:  sourceKey,
				})
			}
		}
	}
	return slots
}

func providerAuthorSchemaBytes(catalog reportilcontract.SourceCatalog) []byte {
	section := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"title", "blocks"},
		"properties": map[string]any{
			"title": nonblankStringSchema(),
			"blocks": map[string]any{
				"type":     "array",
				"minItems": 1,
				"items":    map[string]any{"$ref": "#/$defs/document_block_draft"},
			},
		},
	}
	component := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required": []any{
			"title", "language", "reader_takeaway", "throughline",
			"voice_and_tone", "sections", "evidence_packets", "terminology", "language_review",
		},
		"properties": map[string]any{
			"title":           nonblankStringSchema(),
			"language":        map[string]any{"type": "string", "pattern": providerDocumentLanguagePattern.String()},
			"reader_takeaway": nonblankStringSchema(),
			"throughline":     nonblankStringSchema(),
			"voice_and_tone":  nonblankStringSchema(),
			"sections": map[string]any{
				"type":     "array",
				"minItems": 2,
				"maxItems": 12,
				"items":    map[string]any{"$ref": "#/$defs/author_section_draft"},
			},
			"evidence_packets": map[string]any{
				"type":     "array",
				"minItems": 0,
				"maxItems": reportilcontract.MaxSourceReadSpans,
				"items":    map[string]any{"$ref": "#/$defs/author_evidence_draft"},
			},
			"terminology": map[string]any{
				"type":     "array",
				"minItems": 0,
				"maxItems": maxAuthorTerminologyTerms,
				"items":    map[string]any{"$ref": "#/$defs/author_term_draft"},
			},
			"language_review": languageReviewSchema(),
		},
	}
	return marshalProviderSchema(providerSchemaDocument, map[string]any{
		"$ref": "#/$defs/author_draft",
		"$defs": map[string]any{
			"author_draft":          component,
			"author_section_draft":  section,
			"author_evidence_draft": authorEvidenceDraftSchema(catalog),
			"author_term_draft":     authorTerminologySchema(),
			"document_block_draft":  documentBlockDraftSchema(catalog),
		},
	})
}

func compileAuthorDraft(
	draft authorDraft,
	contractID,
	documentID,
	missionObjective string,
	catalog reportilcontract.SourceCatalog,
	receipt reportilcontract.SourceReadReceipt,
	citations ...map[int]SourceCitation,
) (Narrative, Document, error) {
	citationCatalog := map[int]SourceCitation{}
	if len(citations) > 0 && citations[0] != nil {
		citationCatalog = citations[0]
	}
	narrative, document, _, err := compileAuthorDraftWithTerminology(
		draft, contractID, documentID, missionObjective, catalog, receipt,
		citationCatalog, nil,
	)
	return narrative, document, err
}

func compileAuthorDraftWithTerminology(
	draft authorDraft,
	contractID,
	documentID,
	missionObjective string,
	catalog reportilcontract.SourceCatalog,
	receipt reportilcontract.SourceReadReceipt,
	citations map[int]SourceCitation,
	readableByAcceptedOrdinal map[int]string,
) (Narrative, Document, []terminologyDecision, error) {
	if !providerDocumentLanguagePattern.MatchString(draft.Language) {
		return Narrative{}, Document{}, nil, withValidationCode(
			reportexecution.ProviderValidationCodeDocumentContract,
			fmt.Errorf("author language is invalid"),
		)
	}
	if len(draft.Sections) < 2 || len(draft.Sections) > 12 {
		return Narrative{}, Document{}, nil, withValidationCode(
			reportexecution.ProviderValidationCodeDocumentContract,
			fmt.Errorf("author section inventory is invalid"),
		)
	}
	if err := validateLanguageReview(draft.LanguageReview); err != nil {
		return Narrative{}, Document{}, nil, withValidationCode(
			reportexecution.ProviderValidationCodeLanguageReview,
			fmt.Errorf("author %w", err),
		)
	}
	if err := receipt.Validate(catalog); err != nil {
		return Narrative{}, Document{}, nil, withValidationCode(
			reportexecution.ProviderValidationCodeSourceReadContract,
			err,
		)
	}

	narrative := Narrative{
		SchemaVersion:         NarrativeSchemaVersion,
		ContractID:            contractID,
		DocumentID:            documentID,
		CentralQuestion:       strings.TrimSpace(draft.Throughline),
		ReaderTakeaway:        strings.TrimSpace(draft.ReaderTakeaway),
		Throughline:           strings.TrimSpace(draft.Throughline),
		VoiceAndTone:          strings.TrimSpace(draft.VoiceAndTone),
		ConclusionObligations: []string{strings.TrimSpace(draft.ReaderTakeaway)},
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
	if citations == nil {
		citations = map[int]SourceCitation{}
	}
	used := map[string]bool{}
	for sectionIndex, section := range draft.Sections {
		if strings.TrimSpace(section.Title) == "" || len(section.Blocks) == 0 {
			return Narrative{}, Document{}, nil, withValidationCode(
				reportexecution.ProviderValidationCodeDocumentContract,
				fmt.Errorf("author section is incomplete"),
			)
		}
		sectionID := serverAuthorSectionID(documentID, sectionIndex, section.Title, used)
		narrative.ReaderJourney = append(narrative.ReaderJourney, section.Title)
		narrative.ArgumentArc = append(narrative.ArgumentArc, section.Title)
		narrative.SectionRoles = append(narrative.SectionRoles, SectionRole{
			SectionID:        sectionID,
			Role:             authorSectionRole(sectionIndex, len(draft.Sections)),
			QuestionAnswered: section.Title,
		})
		document.Blocks = append(document.Blocks, Block{
			NodeID: sectionID,
			Kind:   "section",
			Level:  2,
			Title:  section.Title,
		})
		for blockIndex, source := range section.Blocks {
			block, err := compileDocumentBlockDraft(documentID, sectionID, sectionIndex, blockIndex, source, used)
			if err != nil {
				return Narrative{}, Document{}, nil, withValidationCode(
					reportexecution.ProviderValidationCodeDocumentContract,
					err,
				)
			}
			if err := compileBlockEvidence(&document, &block, catalog, receipt, source.EvidenceSourceKeys, citations); err != nil {
				return Narrative{}, Document{}, nil, withValidationCode(
					reportexecution.ProviderValidationCodeSourceReadContract,
					err,
				)
			}
			document.Blocks = append(document.Blocks, block)
		}
	}
	evidencePackets, err := compileAuthorEvidence(
		draft.EvidencePackets,
		document,
		catalog,
		receipt,
		readableByAcceptedOrdinal,
	)
	if err != nil {
		return Narrative{}, Document{}, nil, err
	}
	narrative.EvidencePackets = evidencePackets
	terms, err := compileAuthorTerminology(draft.Terminology, document, catalog, receipt, readableByAcceptedOrdinal)
	if err != nil {
		return Narrative{}, Document{}, nil, err
	}
	narrative.ContinuityTerms = continuityTerms(terms)
	if err := validateNarrative(&narrative, document); err != nil {
		return Narrative{}, Document{}, nil, withValidationCode(
			reportexecution.ProviderValidationCodeSemanticContract,
			err,
		)
	}
	if err := validateProductDocument(narrative, document); err != nil {
		return Narrative{}, Document{}, nil, withValidationCode(
			reportexecution.ProviderValidationCodeDocumentContract,
			err,
		)
	}
	return narrative, document, terms, nil
}

func serverAuthorSectionID(documentID string, index int, _ string, used map[string]bool) string {
	for salt := 0; ; salt++ {
		preimage := fmt.Sprintf("%s\x00section\x00%d\x00%d", documentID, index, salt)
		sum := sha256.Sum256([]byte(preimage))
		candidate := "section." + hex.EncodeToString(sum[:])
		if !used[candidate] {
			used[candidate] = true
			return candidate
		}
	}
}

func authorSectionRole(index, total int) string {
	switch {
	case index == 0:
		return "opening"
	case index == total-1:
		return "conclusion"
	default:
		return "evidence"
	}
}

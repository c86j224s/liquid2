package reportilphase0

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

func evidencePacketFixture(t *testing.T) (
	reportilcontract.SourceCatalog,
	map[int]string,
	authorDraft,
) {
	t.Helper()
	readable := "ARCHIVAL RECORD: fortress was rebuilt in 1580; stone walls followed in 1600."
	content := []byte(readable)
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: "mis_evidence_packet",
		Sources: []reportilcontract.SourceCatalogEntry{{
			SourceKey: "source_001", SnapshotID: "src_evidence",
			SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_evidence", sha256Hex(content)),
			ContentHash:     sha256Hex(content), RetrievalPolicy: "snapshot_only",
			Artifacts: []reportilcontract.SourceCatalogArtifact{{
				ArtifactID: "art_evidence", SHA256: sha256Hex(content),
				ByteSize: int64(len(content)), MediaType: "text/plain",
			}},
			ReadableSHA256: sha256Hex(content), ReadableBytes: len(content),
			Extraction: "stored_text",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ordinal := catalog.Sources[0].AcceptedOrdinal
	author := authorDraft{
		LanguageReview: acceptedLanguageReview(),
		Title:          "Fortress history",
		Language:       "en",
		ReaderTakeaway: "The rebuild and stone walls mark distinct phases.",
		Throughline:    "The site changed in two phases.",
		VoiceAndTone:   "Direct",
		Sections: []authorSectionDraft{
			{
				Title: "Two phases",
				Blocks: []documentBlockDraft{{
					Kind:               "prose",
					Prose:              "The fortress was rebuilt in 1580. The stone walls followed in 1600.",
					EvidenceSourceKeys: []string{"source_001"},
				}},
			},
			{
				Title: "Meaning",
				Blocks: []documentBlockDraft{{
					Kind:  "prose",
					Prose: "The phases should not be collapsed into one event.",
				}},
			},
		},
		EvidencePackets: []authorEvidenceDraft{
			{
				SectionKey: "section_01", BlockKey: "block_01",
				Claim: "The fortress was rebuilt in 1580.", SourceKey: "source_001",
				SourceExcerpt: "ARCHIVAL RECORD: fortress was rebuilt in 1580", Preserve: true,
			},
			{
				SectionKey: "section_01", BlockKey: "block_01",
				Claim: "The stone walls followed in 1600.", SourceKey: "source_001",
				SourceExcerpt: "rebuilt in 1580; stone walls followed", Preserve: false,
			},
		},
	}
	return catalog, map[int]string{ordinal: readable}, author
}

func authorEvidenceRepairFromPackets(
	t *testing.T,
	frozen authorDraft,
	packets []authorEvidenceDraft,
) authorEvidenceRepairDraft {
	t.Helper()
	result := authorEvidenceRepairDraft{Bindings: map[string][]authorEvidenceRepairPacketDraft{}}
	slots := authorEvidenceBindingSlots(frozen)
	aliasByBinding := map[string]string{}
	for _, slot := range slots {
		result.Bindings[slot.Alias] = []authorEvidenceRepairPacketDraft{}
		aliasByBinding[slot.SectionKey+"\x00"+slot.BlockKey+"\x00"+slot.SourceKey] = slot.Alias
	}
	for _, packet := range packets {
		alias, ok := aliasByBinding[packet.SectionKey+"\x00"+packet.BlockKey+"\x00"+packet.SourceKey]
		if !ok {
			t.Fatalf("packet has no frozen binding: %#v", packet)
		}
		result.Bindings[alias] = append(result.Bindings[alias], authorEvidenceRepairPacketDraft{
			Claim: packet.Claim, SourceReceipt: testAuthorSourceReceipt(packet), Preserve: packet.Preserve,
		})
	}
	for alias, bindingPackets := range result.Bindings {
		if len(bindingPackets) == 0 {
			t.Fatalf("frozen binding %s has no repair packet", alias)
		}
	}
	return result
}

func testAuthorSourceReceipt(packet authorEvidenceDraft) string {
	return "quote_" + SHA256([]byte(packet.SectionKey+"\x00"+packet.BlockKey+"\x00"+packet.SourceKey+"\x00"+packet.SourceExcerpt))
}

func authorWithSourceReceipts(author authorDraft) authorDraft {
	result := author
	result.EvidencePackets = append([]authorEvidenceDraft(nil), author.EvidencePackets...)
	for index := range result.EvidencePackets {
		result.EvidencePackets[index].SourceReceipt = testAuthorSourceReceipt(result.EvidencePackets[index])
		result.EvidencePackets[index].SourceExcerpt = ""
	}
	return result
}

func sourceQuoteReceiptFixture(
	author authorDraft,
	catalog reportilcontract.SourceCatalog,
	readableByAcceptedOrdinal map[int]string,
) map[string]reportilcontract.SourceQuoteReceipt {
	quotes := map[string]reportilcontract.SourceQuoteReceipt{}
	for _, packet := range author.EvidencePackets {
		entry, ok := catalog.Entry(packet.SourceKey)
		if !ok {
			continue
		}
		excerpt := strings.TrimSpace(packet.SourceExcerpt)
		offset := strings.Index(readableByAcceptedOrdinal[entry.AcceptedOrdinal], excerpt)
		if excerpt == "" || offset < 0 {
			continue
		}
		receipt := testAuthorSourceReceipt(packet)
		quotes[receipt] = reportilcontract.SourceQuoteReceipt{
			Receipt: receipt, SourceKey: packet.SourceKey, Offset: offset,
			ByteSize: len([]byte(excerpt)), SHA256: SHA256([]byte(excerpt)),
		}
	}
	return quotes
}

func compileEvidencePacketFixture(
	t *testing.T,
	author authorDraft,
	catalog reportilcontract.SourceCatalog,
	readable map[int]string,
) (Narrative, Document) {
	t.Helper()
	narrative, document, _, err := compileAuthorDraftWithTerminology(
		author,
		"narrative_evidence",
		"doc_evidence",
		"Explain the phases.",
		catalog,
		testSourceReadReceipt(catalog),
		nil,
		readable,
	)
	if err != nil {
		t.Fatal(err)
	}
	return narrative, document
}

func TestAuthorEvidencePacketRejectsMissingSourceBinding(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	author.EvidencePackets = author.EvidencePackets[:0]
	_, document, _, err := compileAuthorDraftWithTerminology(
		author,
		"narrative_evidence",
		"doc_evidence",
		"Explain the phases.",
		catalog,
		testSourceReadReceipt(catalog),
		nil,
		readable,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(documentNodeEvidenceOrdinals(document, readerSections(document)[0].Blocks[0].NodeID)) == 0 {
		t.Fatal("fixture lost its manuscript source binding")
	}
	author.EvidencePackets = []authorEvidenceDraft{
		{
			SectionKey: "section_02", BlockKey: "block_01",
			Claim:     "The phases should not be collapsed into one event.",
			SourceKey: "source_001", SourceExcerpt: "fortress was rebuilt in 1580",
		},
	}
	if _, _, _, err := compileAuthorDraftWithTerminology(
		author,
		"narrative_evidence",
		"doc_evidence",
		"Explain the phases.",
		catalog,
		testSourceReadReceipt(catalog),
		nil,
		readable,
	); err == nil || !strings.Contains(err.Error(), "manuscript evidence") ||
		providerValidationCode(err) != reportexecution.ProviderValidationCodeEvidencePacketBinding {
		t.Fatalf("missing source binding was accepted: %v", err)
	}
}

func TestAuthorEvidencePacketUsesGranularInventoryAndTargetCodes(t *testing.T) {
	compile := func(t *testing.T, author authorDraft) error {
		t.Helper()
		catalog, readable, _ := evidencePacketFixture(t)
		_, _, _, err := compileAuthorDraftWithTerminology(
			author,
			"narrative_evidence",
			"doc_evidence",
			"Explain the phases.",
			catalog,
			testSourceReadReceipt(catalog),
			nil,
			readable,
		)
		return err
	}

	_, _, target := evidencePacketFixture(t)
	target.EvidencePackets[0].SectionKey = "section_99"
	if err := compile(t, target); err == nil ||
		providerValidationCode(err) != reportexecution.ProviderValidationCodeEvidencePacketTarget {
		t.Fatalf("invalid target code = %q, err=%v", providerValidationCode(err), err)
	}

	catalog, readable, redundant := evidencePacketFixture(t)
	duplicate := redundant.EvidencePackets[1]
	duplicate.SourceExcerpt = "stone walls followed in 1600."
	duplicate.Preserve = true
	redundant.EvidencePackets = append(redundant.EvidencePackets, duplicate)
	providerAuthor := authorWithSourceReceipts(redundant)
	if providerAuthor.EvidencePackets[1].SourceReceipt == providerAuthor.EvidencePackets[2].SourceReceipt {
		t.Fatal("duplicate packet fixture reused one source receipt")
	}
	receipt := testSourceReadReceipt(catalog)
	receipt.SourceQuotes = sourceQuoteReceiptFixture(redundant, catalog, readable)
	narrative, _, _, err := compileAuthorDraftWithTerminology(
		providerAuthor,
		"narrative_evidence",
		"doc_evidence",
		"Explain the phases.",
		catalog,
		receipt,
		nil,
		readable,
	)
	if err != nil {
		t.Fatalf("redundant packet meaning was not canonicalized: %v", err)
	}
	if len(narrative.EvidencePackets) != 2 || !narrative.EvidencePackets[1].Preserve {
		t.Fatalf("canonical evidence packet inventory = %#v", narrative.EvidencePackets)
	}
}

func TestProductEvidencePacketsRejectSourceFreeFactualLeaf(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, document := compileEvidencePacketFixture(t, author, catalog, readable)
	if err := validateProductEvidencePackets(narrative, document); err == nil ||
		!strings.Contains(err.Error(), "reader-facing content leaf") {
		t.Fatalf("source-free factual leaf was accepted: %v", err)
	}

	conclusion := &author.Sections[1].Blocks[0]
	conclusion.EvidenceSourceKeys = []string{"source_001"}
	conclusionText := conclusion.Prose
	author.EvidencePackets = append(author.EvidencePackets, authorEvidenceDraft{
		SectionKey: "section_02", BlockKey: "block_01", Claim: conclusionText,
		SourceKey: "source_001", SourceExcerpt: "stone walls followed in 1600",
	})
	narrative, document = compileEvidencePacketFixture(t, author, catalog, readable)
	if err := validateProductEvidencePackets(narrative, document); err != nil {
		t.Fatal(err)
	}
}

func TestAuthorEvidencePacketResolvesOnlyVerifiedSingleUseSourceReceipts(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	providerAuthor := authorWithSourceReceipts(author)
	quotes := sourceQuoteReceiptFixture(author, catalog, readable)
	receipt := testSourceReadReceipt(catalog)
	receipt.SourceQuotes = quotes
	if _, _, _, err := compileAuthorDraftWithTerminology(
		providerAuthor, "narrative_evidence", "doc_evidence", "Explain the phases.",
		catalog, receipt, nil, readable,
	); err != nil {
		t.Fatal(err)
	}

	cases := map[string]func(*authorDraft, *reportilcontract.SourceReadReceipt){
		"unknown": func(draft *authorDraft, _ *reportilcontract.SourceReadReceipt) {
			draft.EvidencePackets[0].SourceReceipt = "quote_" + strings.Repeat("f", 64)
		},
		"wrong source": func(draft *authorDraft, sourceReceipt *reportilcontract.SourceReadReceipt) {
			key := draft.EvidencePackets[0].SourceReceipt
			quote := sourceReceipt.SourceQuotes[key]
			quote.SourceKey = "source_999"
			sourceReceipt.SourceQuotes[key] = quote
		},
		"duplicate": func(draft *authorDraft, _ *reportilcontract.SourceReadReceipt) {
			draft.EvidencePackets[1].SourceReceipt = draft.EvidencePackets[0].SourceReceipt
		},
		"missing": func(draft *authorDraft, _ *reportilcontract.SourceReadReceipt) {
			draft.EvidencePackets[0].SourceReceipt = ""
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			draft := cloneAuthorDraft(t, providerAuthor)
			attemptReceipt := testSourceReadReceipt(catalog)
			attemptReceipt.SourceQuotes = map[string]reportilcontract.SourceQuoteReceipt{}
			for key, quote := range quotes {
				attemptReceipt.SourceQuotes[key] = quote
			}
			mutate(&draft, &attemptReceipt)
			if _, _, _, err := compileAuthorDraftWithTerminology(
				draft, "narrative_evidence", "doc_evidence", "Explain the phases.",
				catalog, attemptReceipt, nil, readable,
			); err == nil {
				t.Fatal("invalid source receipt inventory was accepted")
			}
		})
	}

	extra := "quote_" + strings.Repeat("e", 64)
	receipt.SourceQuotes[extra] = reportilcontract.SourceQuoteReceipt{
		Receipt: extra, SourceKey: "source_001", Offset: 0, ByteSize: 1,
		SHA256: SHA256([]byte("A")),
	}
	if _, _, _, err := compileAuthorDraftWithTerminology(
		providerAuthor, "narrative_evidence", "doc_evidence", "Explain the phases.",
		catalog, receipt, nil, readable,
	); err != nil {
		t.Fatalf("unused request-local quote candidate was not discarded: %v", err)
	}
}

func TestAuthorEvidencePacketRejectsUnfrozenExcerpt(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	author.EvidencePackets[0].SourceExcerpt = "not present in the frozen source"
	if _, _, _, err := compileAuthorDraftWithTerminology(
		author,
		"narrative_evidence",
		"doc_evidence",
		"Explain the phases.",
		catalog,
		testSourceReadReceipt(catalog),
		nil,
		readable,
	); err == nil || !strings.Contains(err.Error(), "not source-grounded") ||
		providerValidationCode(err) != reportexecution.ProviderValidationCodeEvidencePacketSource ||
		!reflect.DeepEqual(providerValidationPacketAliases(err), []string{"evidence_001"}) {
		t.Fatalf("unfrozen excerpt was accepted: %v", err)
	}
}

func TestAuthorEvidencePacketCollectsEveryInvalidSourceAlias(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	author.EvidencePackets[0].SourceExcerpt = "PRIVATE INVALID FIRST"
	author.EvidencePackets[1].SourceExcerpt = "PRIVATE INVALID SECOND"
	_, _, _, err := compileAuthorDraftWithTerminology(
		author,
		"narrative_evidence",
		"doc_evidence",
		"Explain the phases.",
		catalog,
		testSourceReadReceipt(catalog),
		nil,
		readable,
	)
	if err == nil ||
		providerValidationCode(err) != reportexecution.ProviderValidationCodeEvidencePacketSource ||
		!reflect.DeepEqual(
			providerValidationPacketAliases(err),
			[]string{"evidence_001", "evidence_002"},
		) {
		t.Fatalf("invalid source aliases = %#v, err=%v", providerValidationPacketAliases(err), err)
	}
}

func TestAuthorEvidencePacketRejectsUncoveredFactualLeafText(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	author.EvidencePackets = author.EvidencePackets[:1]
	if _, _, _, err := compileAuthorDraftWithTerminology(
		author,
		"narrative_evidence",
		"doc_evidence",
		"Explain the phases.",
		catalog,
		testSourceReadReceipt(catalog),
		nil,
		readable,
	); err == nil || !strings.Contains(err.Error(), "complete factual leaf") ||
		providerValidationCode(err) != reportexecution.ProviderValidationCodeEvidencePacketCoverage {
		t.Fatalf("uncovered factual sentence was accepted: %v", err)
	}
}

func TestAuthorEvidencePacketFailuresUseServerFixedPacketOnlyRepair(t *testing.T) {
	catalog, readable, valid := evidencePacketFixture(t)
	conclusion := &valid.Sections[1].Blocks[0]
	conclusion.EvidenceSourceKeys = []string{"source_001"}
	valid.EvidencePackets = append(valid.EvidencePackets, authorEvidenceDraft{
		SectionKey: "section_02", BlockKey: "block_01", Claim: conclusion.Prose,
		SourceKey: "source_001", SourceExcerpt: "stone walls followed in 1600",
	})
	valid.Terminology = []authorTermDraft{{
		Category: "proper_name", SourceForm: "fortress", ReaderForm: "fortress",
		Handling: "preserve",
	}}

	cases := map[string]struct {
		code   reportexecution.ProviderValidationCode
		mutate func(*authorDraft)
	}{
		"inventory": {
			code: reportexecution.ProviderValidationCodeEvidencePacketInventory,
			mutate: func(draft *authorDraft) {
				for len(draft.EvidencePackets) <= reportilcontract.MaxSourceReadSpans {
					draft.EvidencePackets = append(draft.EvidencePackets, draft.EvidencePackets[0])
				}
			},
		},
		"target": {
			code: reportexecution.ProviderValidationCodeEvidencePacketTarget,
			mutate: func(draft *authorDraft) {
				draft.EvidencePackets[0].SectionKey = "section_99"
			},
		},
		"source": {
			code: reportexecution.ProviderValidationCodeEvidencePacketSource,
			mutate: func(draft *authorDraft) {
				draft.EvidencePackets[0].SourceExcerpt = "PRIVATE INVALID SOURCE EXCERPT"
			},
		},
		"binding": {
			code: reportexecution.ProviderValidationCodeEvidencePacketBinding,
			mutate: func(draft *authorDraft) {
				draft.EvidencePackets = draft.EvidencePackets[:2]
			},
		},
		"coverage": {
			code: reportexecution.ProviderValidationCodeEvidencePacketCoverage,
			mutate: func(draft *authorDraft) {
				draft.EvidencePackets = append(
					append([]authorEvidenceDraft(nil), draft.EvidencePackets[:1]...),
					draft.EvidencePackets[2],
				)
			},
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			invalid := cloneAuthorDraft(t, valid)
			test.mutate(&invalid)
			_, _, _, validationErr := compileAuthorStageDraft(
				invalid, "narrative_1", "doc_1", "Explain the phases.", catalog,
				testSourceReadReceipt(catalog), nil, readable,
			)
			if validationErr == nil || providerValidationCode(validationErr) != test.code {
				t.Fatalf(
					"first draft validation code = %q, want %q, err=%v",
					providerValidationCode(validationErr), test.code, validationErr,
				)
			}

			providerInvalid := cloneAuthorDraft(t, invalid)
			for index := range providerInvalid.EvidencePackets {
				providerInvalid.EvidencePackets[index].SourceReceipt =
					testAuthorSourceReceipt(providerInvalid.EvidencePackets[index])
			}
			provider := &recordingProvider{outputs: []string{
				string(mustMarshal(providerInvalid)),
				string(mustMarshal(authorEvidenceRepairFromPackets(t, invalid, valid.EvidencePackets))),
			}}
			verifier := &acceptingSourceReadVerifier{
				sourceQuotes: sourceQuoteReceiptFixture(valid, catalog, readable),
			}
			config := testSourceProductConfig(provider, verifier)
			config.MissionID = catalog.MissionID
			config.MissionObjective = "Explain the phases."
			narrative, document, terminology, attempts, err := runAuthorStageWithTerminology(
				context.Background(), config, catalog, nil, readable,
			)
			if err != nil || len(attempts) != 2 || len(provider.requests) != 2 ||
				len(verifier.calls) != 2 {
				t.Fatalf(
					"packet repair attempts=%d requests=%d reads=%d err=%v",
					len(attempts), len(provider.requests), len(verifier.calls), err,
				)
			}
			if verifier.calls[0] == verifier.calls[1] ||
				provider.requests[0].ReportILSources.Attempt != 1 ||
				provider.requests[1].ReportILSources.Attempt != 2 {
				t.Fatalf("packet repair did not use a fresh source session: %#v", provider.requests)
			}

			expectedNarrative, expectedDocument, expectedTerminology, err :=
				compileAuthorDraftWithTerminology(
					valid, "narrative_1", "doc_1", config.MissionObjective, catalog,
					testSourceReadReceipt(catalog), nil, readable,
				)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(narrative, expectedNarrative) ||
				!reflect.DeepEqual(document, expectedDocument) ||
				!reflect.DeepEqual(terminology, expectedTerminology) {
				t.Fatal("packet repair changed frozen manuscript, terminology, or source bindings")
			}

			if string(provider.requests[0].OutputJSONSchema) !=
				string(providerAuthorSchemaBytes(catalog)) ||
				string(provider.requests[1].OutputJSONSchema) !=
					string(providerAuthorEvidenceRepairSchemaBytes(invalid)) {
				t.Fatal("packet repair did not switch to the dedicated response schema")
			}
			var repairSchema map[string]any
			if err := json.Unmarshal(provider.requests[1].OutputJSONSchema, &repairSchema); err != nil {
				t.Fatal(err)
			}
			definitions := repairSchema["$defs"].(map[string]any)
			root := definitions["author_evidence_repair_draft"].(map[string]any)
			properties := root["properties"].(map[string]any)
			bindings := properties["bindings"].(map[string]any)
			bindingProperties := bindings["properties"].(map[string]any)
			packetSchema := definitions["author_evidence_repair_packet_draft"].(map[string]any)
			packetProperties := packetSchema["properties"].(map[string]any)
			if len(properties) != 1 || len(bindingProperties) != len(authorEvidenceBindingSlots(invalid)) ||
				bindings["additionalProperties"] != false || len(packetProperties) != 3 ||
				packetProperties["claim"] == nil || packetProperties["source_receipt"] == nil ||
				packetProperties["preserve"] == nil {
				t.Fatalf("packet repair schema does not fix binding ownership: %#v", repairSchema)
			}
			for _, forbidden := range []string{"section_key", "block_key", "source_key", "evidence_packets"} {
				if strings.Contains(string(provider.requests[1].OutputJSONSchema), forbidden) {
					t.Fatalf("packet repair schema exposes provider-owned %q", forbidden)
				}
			}

			repairPrompt := provider.requests[1].Prompt
			for _, required := range []string{
				"Rebuild only the complete evidence packet content",
				"server-frozen manuscript", "FROZEN READER-FACING LEAVES AND SERVER-FIXED BINDINGS",
				"section_01/block_01", "kind=prose", "bindings=binding_001",
				"binding_001 is server-fixed to source_001", valid.Sections[0].Blocks[0].Prose,
				"plasma.report_il.sources.quote", "opaque source_receipt",
				"512 UTF-8 bytes", reportilcontract.SourceListTool,
				reportilcontract.SourceReadTool, reportilcontract.SourceQuoteRegisterTool,
			} {
				if !strings.Contains(repairPrompt, required) {
					t.Fatalf("packet repair prompt lacks %q: %s", required, repairPrompt)
				}
			}
			for _, forbidden := range []string{
				"PRIVATE INVALID SOURCE EXCERPT",
				"ARCHIVAL RECORD: fortress was rebuilt in 1580",
				string(mustMarshal(invalid.EvidencePackets)),
				`"source_form"`, `"language_review"`,
				"INVALID EVIDENCE PACKETS TO REPAIR", "PRIVATE VALIDATOR DETAIL",
			} {
				if strings.Contains(repairPrompt, forbidden) {
					t.Fatalf("packet repair prompt leaked %q: %s", forbidden, repairPrompt)
				}
			}
		})
	}
}

func TestAuthorEvidenceRepairSchemaRequiresOnlyFrozenBindingSlots(t *testing.T) {
	_, _, frozen := evidencePacketFixture(t)
	frozen.Sections[1].Blocks[0].EvidenceSourceKeys = []string{"source_001"}
	schemaRaw := providerAuthorEvidenceRepairSchemaBytes(frozen)
	compiler := jsonschema.NewCompiler()
	var providerSchema any
	if err := json.Unmarshal(schemaRaw, &providerSchema); err != nil {
		t.Fatal(err)
	}
	const schemaURL = "urn:plasma:test:request-local-author-binding-slots"
	if err := compiler.AddResource(schemaURL, providerSchema); err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.Compile(schemaURL)
	if err != nil {
		t.Fatal(err)
	}
	packet := authorEvidenceRepairPacketDraft{
		Claim: "Claim", SourceReceipt: "quote_" + strings.Repeat("a", 64), Preserve: true,
	}
	valid := authorEvidenceRepairDraft{Bindings: map[string][]authorEvidenceRepairPacketDraft{
		"binding_001": {packet},
		"binding_002": {packet},
	}}
	assertSchemaResult := func(name string, value authorEvidenceRepairDraft, wantValid bool) {
		t.Helper()
		generic, err := schemaValue(value)
		if err != nil {
			t.Fatal(err)
		}
		err = compiled.Validate(generic)
		if (err == nil) != wantValid {
			t.Fatalf("%s schema validity = %v, want %v: %v", name, err == nil, wantValid, err)
		}
	}
	assertSchemaResult("complete frozen inventory", valid, true)

	missing := authorEvidenceRepairDraft{Bindings: map[string][]authorEvidenceRepairPacketDraft{
		"binding_001": {packet},
	}}
	assertSchemaResult("missing frozen slot", missing, false)

	extra := authorEvidenceRepairDraft{Bindings: map[string][]authorEvidenceRepairPacketDraft{
		"binding_001": {packet},
		"binding_002": {packet},
		"binding_999": {packet},
	}}
	assertSchemaResult("provider-selected slot", extra, false)

	empty := authorEvidenceRepairDraft{Bindings: map[string][]authorEvidenceRepairPacketDraft{
		"binding_001": {},
		"binding_002": {packet},
	}}
	assertSchemaResult("empty frozen slot", empty, false)
}

func TestAuthorEvidenceRepairRuntimeRejectsChangedBindingInventory(t *testing.T) {
	catalog, readable, frozen := evidencePacketFixture(t)
	conclusion := &frozen.Sections[1].Blocks[0]
	conclusion.EvidenceSourceKeys = []string{"source_001"}
	validRepairPackets := append(
		append([]authorEvidenceDraft(nil), frozen.EvidencePackets...),
		authorEvidenceDraft{
			SectionKey: "section_02", BlockKey: "block_01", Claim: conclusion.Prose,
			SourceKey: "source_001", SourceExcerpt: "stone walls followed in 1600",
		},
	)

	cases := map[string]func(*authorEvidenceRepairDraft){
		"extra slot": func(repair *authorEvidenceRepairDraft) {
			repair.Bindings["binding_999"] = repair.Bindings["binding_001"]
		},
		"missing slot": func(repair *authorEvidenceRepairDraft) {
			delete(repair.Bindings, "binding_002")
		},
		"empty slot": func(repair *authorEvidenceRepairDraft) {
			repair.Bindings["binding_002"] = nil
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			invalidRepair := authorEvidenceRepairFromPackets(t, frozen, validRepairPackets)
			mutate(&invalidRepair)
			provider := &recordingProvider{outputs: []string{
				string(mustMarshal(authorDraft{
					Title: frozen.Title, Language: frozen.Language,
					ReaderTakeaway: frozen.ReaderTakeaway, Throughline: frozen.Throughline,
					VoiceAndTone: frozen.VoiceAndTone, Sections: frozen.Sections,
					EvidencePackets: nil, Terminology: frozen.Terminology,
					LanguageReview: frozen.LanguageReview,
				})),
				string(mustMarshal(invalidRepair)),
			}}
			verifier := &acceptingSourceReadVerifier{}
			config := testSourceProductConfig(provider, verifier)
			config.MissionID = catalog.MissionID
			_, _, _, attempts, err := runAuthorStageWithTerminology(
				context.Background(), config, catalog, nil, readable,
			)
			var stageErr *providerStageError
			code := reportexecution.ProviderValidationCode("")
			if errors.As(err, &stageErr) {
				code = providerValidationCode(stageErr.cause)
			}
			if stageErr == nil ||
				stageErr.reason != reportexecution.ProviderFailureReasonSemanticValidation ||
				code != reportexecution.ProviderValidationCodeEvidencePacketInventory ||
				len(attempts) != 2 || len(provider.requests) != 2 || len(verifier.calls) != 2 {
				t.Fatalf("runtime changed inventory attempts=%d requests=%d reads=%d code=%q err=%v", len(attempts), len(provider.requests), len(verifier.calls), code, err)
			}
		})
	}
}

func TestAuthorEvidenceBindingSlotsDeduplicateAndApplyServerOwnership(t *testing.T) {
	_, _, frozen := evidencePacketFixture(t)
	frozen.Sections[0].Blocks[0].EvidenceSourceKeys = []string{"source_001", "source_001"}
	slots := authorEvidenceBindingSlots(frozen)
	if len(slots) != 1 || slots[0].Alias != "binding_001" ||
		slots[0].SectionKey != "section_01" || slots[0].BlockKey != "block_01" ||
		slots[0].SourceKey != "source_001" {
		t.Fatalf("server-fixed binding slots = %#v", slots)
	}

	original := cloneAuthorDraft(t, frozen)
	repair := authorEvidenceRepairDraft{Bindings: map[string][]authorEvidenceRepairPacketDraft{
		"binding_001": {
			{Claim: "The fortress was rebuilt in 1580.", SourceExcerpt: "fortress was rebuilt in 1580", Preserve: true},
			{Claim: "The stone walls followed in 1600.", SourceExcerpt: "stone walls followed in 1600"},
		},
	}}
	repaired := applyAuthorEvidenceRepair(frozen, repair)
	if len(repaired.EvidencePackets) != 2 {
		t.Fatalf("flattened packet count = %d", len(repaired.EvidencePackets))
	}
	for _, packet := range repaired.EvidencePackets {
		if packet.SectionKey != "section_01" || packet.BlockKey != "block_01" ||
			packet.SourceKey != "source_001" {
			t.Fatalf("provider changed server-fixed binding: %#v", packet)
		}
	}
	repaired.EvidencePackets = original.EvidencePackets
	if !reflect.DeepEqual(repaired, original) {
		t.Fatal("server-fixed repair changed manuscript, terminology, or source bindings")
	}
}

func TestAuthorEvidenceRepairFlattenedInventoryKeepsGlobalCeiling(t *testing.T) {
	catalog, readable, frozen := evidencePacketFixture(t)
	conclusion := &frozen.Sections[1].Blocks[0]
	conclusion.EvidenceSourceKeys = []string{"source_001"}
	packet := authorEvidenceRepairPacketDraft{
		Claim:         "The fortress was rebuilt in 1580.",
		SourceExcerpt: "fortress was rebuilt in 1580",
	}
	firstBinding := make([]authorEvidenceRepairPacketDraft, reportilcontract.MaxSourceReadSpans)
	for index := range firstBinding {
		firstBinding[index] = packet
	}
	repair := authorEvidenceRepairDraft{Bindings: map[string][]authorEvidenceRepairPacketDraft{
		"binding_001": firstBinding,
		"binding_002": {{
			Claim: conclusion.Prose, SourceExcerpt: "stone walls followed in 1600",
		}},
	}}
	if err := validateAuthorEvidenceRepair(frozen, repair); err == nil ||
		providerValidationCode(err) != reportexecution.ProviderValidationCodeEvidencePacketInventory {
		t.Fatalf("server repair ceiling code=%q err=%v", providerValidationCode(err), err)
	}
	repaired := applyAuthorEvidenceRepair(frozen, repair)
	if len(repaired.EvidencePackets) != reportilcontract.MaxSourceReadSpans+1 {
		t.Fatalf("flattened packet count = %d", len(repaired.EvidencePackets))
	}
	_, _, _, err := compileAuthorDraftWithTerminology(
		repaired, "narrative_1", "doc_1", "Explain the phases.", catalog,
		testSourceReadReceipt(catalog), nil, readable,
	)
	if err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeEvidencePacketInventory {
		t.Fatalf("global packet ceiling code=%q err=%v", providerValidationCode(err), err)
	}
}

func TestAuthorEvidenceRepairUsesFullSchemaAboveBindingCeiling(t *testing.T) {
	catalog, readable, oversized := evidencePacketFixture(t)
	oversized.Sections = nil
	for sectionIndex := 0; sectionIndex < 2; sectionIndex++ {
		section := authorSectionDraft{Title: fmt.Sprintf("Section %d", sectionIndex+1)}
		for blockIndex := 0; blockIndex < 65; blockIndex++ {
			section.Blocks = append(section.Blocks, documentBlockDraft{
				Kind: "prose", Prose: fmt.Sprintf("Bounded fact %d.%d.", sectionIndex+1, blockIndex+1),
				EvidenceSourceKeys: []string{"source_001"},
			})
		}
		oversized.Sections = append(oversized.Sections, section)
	}
	oversized.EvidencePackets = nil
	if slots := authorEvidenceBindingSlots(oversized); len(slots) != 130 {
		t.Fatalf("oversized frozen slots = %d", len(slots))
	}
	provider := &recordingProvider{outputs: []string{
		string(mustMarshal(oversized)), string(mustMarshal(oversized)),
	}}
	verifier := &acceptingSourceReadVerifier{}
	config := testSourceProductConfig(provider, verifier)
	config.MissionID = catalog.MissionID
	_, _, _, attempts, err := runAuthorStageWithTerminology(
		context.Background(), config, catalog, nil, readable,
	)
	if err == nil || len(attempts) != 2 || len(provider.requests) != 2 || len(verifier.calls) != 2 {
		t.Fatalf("oversized binding fallback attempts=%d requests=%d reads=%d err=%v", len(attempts), len(provider.requests), len(verifier.calls), err)
	}
	if string(provider.requests[1].OutputJSONSchema) != string(providerAuthorSchemaBytes(catalog)) ||
		strings.Contains(provider.requests[1].Prompt, "schema-required binding_NNN slot") {
		t.Fatalf("oversized binding inventory used packet-only repair: %s", provider.requests[1].Prompt)
	}
}

func TestAuthorEvidencePacketRepairStopsAfterSecondFailure(t *testing.T) {
	catalog, readable, valid := evidencePacketFixture(t)
	conclusion := &valid.Sections[1].Blocks[0]
	conclusion.EvidenceSourceKeys = []string{"source_001"}
	validRepairPackets := append(
		append([]authorEvidenceDraft(nil), valid.EvidencePackets...),
		authorEvidenceDraft{
			SectionKey: "section_02", BlockKey: "block_01", Claim: conclusion.Prose,
			SourceKey: "source_001", SourceExcerpt: "stone walls followed in 1600",
		},
	)
	invalidRepair := authorEvidenceRepairFromPackets(t, valid, validRepairPackets)
	invalidRepair.Bindings["binding_002"][0].Claim = "PRIVATE INVALID REPAIR CLAIM"
	provider := &recordingProvider{outputs: []string{
		string(mustMarshal(valid)),
		string(mustMarshal(invalidRepair)),
	}}
	verifier := &acceptingSourceReadVerifier{}
	config := testSourceProductConfig(provider, verifier)
	config.MissionID = catalog.MissionID
	_, _, _, attempts, err := runAuthorStageWithTerminology(
		context.Background(), config, catalog, nil, readable,
	)
	var stageErr *providerStageError
	if !errors.As(err, &stageErr) ||
		stageErr.reason != reportexecution.ProviderFailureReasonSemanticValidation ||
		providerValidationCode(stageErr.cause) != reportexecution.ProviderValidationCodeEvidencePacketBinding ||
		len(attempts) != 2 || len(provider.requests) != 2 || len(verifier.calls) != 2 {
		t.Fatalf(
			"second packet failure attempts=%d requests=%d reads=%d err=%v",
			len(attempts), len(provider.requests), len(verifier.calls), err,
		)
	}
	usage := TerminalUsageReceipt{}
	usage.add("il_narrative", attempts...)
	failure := providerStageFailure("il_narrative", err, usage)
	encoded := string(failure.AppendRequest(
		config.MissionID, config.PendingEventID, "evt_terminal",
		ledger.Producer{Type: "agent", ID: "codex"},
	).Payload)
	if failure.SafeValidationCode != reportexecution.ProviderValidationCodeEvidencePacketBinding ||
		strings.Contains(encoded, valid.Sections[0].Blocks[0].Prose) ||
		strings.Contains(encoded, "source_001") ||
		strings.Contains(encoded, "section_01") ||
		strings.Contains(encoded, "evidence_001") {
		t.Fatalf("durable packet failure leaked request-local content: %s", encoded)
	}
}

func TestAuthorEvidenceRepairPromptRejectsSensitiveFrozenLeaf(t *testing.T) {
	for name, sensitive := range map[string]string{
		"password":    "password=secret",
		"private key": strings.Join([]string{"-----BEGIN", "PRIVATE KEY-----\nsecret\n-----END PRIVATE KEY-----"}, " "),
	} {
		t.Run(name, func(t *testing.T) {
			catalog, readable, valid := evidencePacketFixture(t)
			conclusion := &valid.Sections[1].Blocks[0]
			conclusion.EvidenceSourceKeys = []string{"source_001"}
			valid.EvidencePackets = append(valid.EvidencePackets, authorEvidenceDraft{
				SectionKey: "section_02", BlockKey: "block_01", Claim: conclusion.Prose,
				SourceKey: "source_001", SourceExcerpt: "stone walls followed in 1600",
			})
			invalid := cloneAuthorDraft(t, valid)
			invalid.Sections[0].Blocks[0].Prose = sensitive
			providerInvalid := authorWithSourceReceipts(invalid)
			providerValid := authorWithSourceReceipts(valid)
			provider := &recordingProvider{outputs: []string{
				string(mustMarshal(providerInvalid)), string(mustMarshal(providerValid)),
			}}
			verifier := &acceptingSourceReadVerifier{
				sourceQuotes: sourceQuoteReceiptFixture(valid, catalog, readable),
			}
			config := testSourceProductConfig(provider, verifier)
			config.MissionID = catalog.MissionID
			_, _, _, attempts, err := runAuthorStageWithTerminology(
				context.Background(), config, catalog, nil, readable,
			)
			if err != nil || len(attempts) != 2 || len(provider.requests) != 2 ||
				len(verifier.calls) != 2 {
				t.Fatalf("sensitive frozen leaf repair attempts=%d err=%v", len(attempts), err)
			}
			if string(provider.requests[1].OutputJSONSchema) != string(providerAuthorSchemaBytes(catalog)) ||
				strings.Contains(provider.requests[1].Prompt, sensitive) ||
				strings.Contains(provider.requests[1].Prompt, string(mustMarshal(invalid))) {
				t.Fatalf("sensitive frozen leaf entered packet repair context: %s", provider.requests[1].Prompt)
			}
		})
	}
}

func TestAuthorEvidenceBindingRepairUsesFullSchemaWhenFrozenLeafHasNoSource(t *testing.T) {
	catalog, readable, valid := evidencePacketFixture(t)
	invalid := cloneAuthorDraft(t, valid)
	conclusion := &valid.Sections[1].Blocks[0]
	conclusion.EvidenceSourceKeys = []string{"source_001"}
	valid.EvidencePackets = append(valid.EvidencePackets, authorEvidenceDraft{
		SectionKey: "section_02", BlockKey: "block_01", Claim: conclusion.Prose,
		SourceKey: "source_001", SourceExcerpt: "stone walls followed in 1600",
	})
	provider := &recordingProvider{outputs: []string{
		string(mustMarshal(authorWithSourceReceipts(invalid))),
		string(mustMarshal(authorWithSourceReceipts(valid))),
	}}
	verifier := &acceptingSourceReadVerifier{
		sourceQuotes: sourceQuoteReceiptFixture(valid, catalog, readable),
	}
	config := testSourceProductConfig(provider, verifier)
	config.MissionID = catalog.MissionID
	_, _, _, attempts, err := runAuthorStageWithTerminology(
		context.Background(), config, catalog, nil, readable,
	)
	if err != nil || len(attempts) != 2 || len(provider.requests) != 2 ||
		len(verifier.calls) != 2 {
		t.Fatalf("missing frozen binding repair attempts=%d err=%v", len(attempts), err)
	}
	if string(provider.requests[1].OutputJSONSchema) != string(providerAuthorSchemaBytes(catalog)) ||
		strings.Contains(provider.requests[1].Prompt, "Rebuild only the complete evidence_packets inventory") {
		t.Fatalf("unrepairable frozen binding used packet-only schema: %s", provider.requests[1].Prompt)
	}
}

func cloneAuthorDraft(t *testing.T, source authorDraft) authorDraft {
	t.Helper()
	clone := source
	clone.Sections = append([]authorSectionDraft(nil), source.Sections...)
	for sectionIndex := range clone.Sections {
		clone.Sections[sectionIndex].Blocks = append(
			[]documentBlockDraft(nil), source.Sections[sectionIndex].Blocks...,
		)
	}
	clone.EvidencePackets = append([]authorEvidenceDraft(nil), source.EvidencePackets...)
	clone.Terminology = append([]authorTermDraft(nil), source.Terminology...)
	return clone
}

func TestAuthorEvidencePacketAllowsNoPreservationQuota(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	for index := range author.EvidencePackets {
		author.EvidencePackets[index].Preserve = false
	}
	narrative, _ := compileEvidencePacketFixture(t, author, catalog, readable)
	for _, packet := range narrative.EvidencePackets {
		if packet.Preserve {
			t.Fatal("fixture unexpectedly preserved a claim")
		}
	}
}

func TestEvidencePacketMergesOverlappingSourceSpans(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, _ := compileEvidencePacketFixture(t, author, catalog, readable)
	spans, err := evidenceSourceReadSpans(catalog, narrative.EvidencePackets)
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 1 || spans[0].ByteSize <= narrative.EvidencePackets[0].SourceByteSize ||
		spans[0].ByteSize <= narrative.EvidencePackets[1].SourceByteSize {
		t.Fatalf("overlapping evidence spans were not merged: %#v", spans)
	}
}

func TestEvidencePacketMergesContainedSourceSpan(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, _ := compileEvidencePacketFixture(t, author, catalog, readable)
	outer := narrative.EvidencePackets[0]
	inner := outer
	inner.EvidenceID = "evidence.contained"
	inner.SourceOffset = outer.SourceOffset + len("ARCHIVAL RECORD: ")
	inner.SourceExcerpt = "fortress was rebuilt in 1580"
	inner.SourceByteSize = len([]byte(inner.SourceExcerpt))
	inner.SourceExcerptSHA = SHA256([]byte(inner.SourceExcerpt))
	spans, err := evidenceSourceReadSpans(catalog, []EvidencePacket{outer, inner})
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 1 || spans[0].Offset != outer.SourceOffset ||
		spans[0].ByteSize != outer.SourceByteSize || spans[0].SHA256 != outer.SourceExcerptSHA {
		t.Fatalf("contained evidence span changed the outer span: %#v", spans)
	}
}

func TestEvidencePacketSplitsLargeAdjacentUnionAtUTF8Boundaries(t *testing.T) {
	content := strings.Repeat("가", 850)
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: "mis_evidence_split",
		Sources: []reportilcontract.SourceCatalogEntry{{
			SourceKey: "source_001", SnapshotID: "src_evidence_split",
			SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_evidence_split", SHA256([]byte(content))),
			ContentHash:     SHA256([]byte(content)), RetrievalPolicy: "snapshot_only",
			Artifacts: []reportilcontract.SourceCatalogArtifact{{
				ArtifactID: "art_evidence_split", SHA256: SHA256([]byte(content)),
				ByteSize: int64(len([]byte(content))), MediaType: "text/plain",
			}},
			ReadableSHA256: SHA256([]byte(content)), ReadableBytes: len([]byte(content)),
			Extraction: "stored_text",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	contentBytes := []byte(content)
	packets := make([]EvidencePacket, 0, 5)
	for index, offset := 0, 0; index < 5; index++ {
		excerpt := string(contentBytes[offset : offset+510])
		packets = append(packets, EvidencePacket{
			AcceptedOrdinal: catalog.Sources[0].AcceptedOrdinal,
			SourceExcerpt:   excerpt, SourceExcerptSHA: SHA256([]byte(excerpt)),
			SourceOffset: offset, SourceByteSize: len([]byte(excerpt)),
		})
		offset += 510
	}
	spans, err := evidenceSourceReadSpans(catalog, packets)
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 2 || spans[0].Offset != 0 || spans[0].ByteSize != 2046 ||
		spans[1].Offset != 2046 || spans[1].ByteSize != 504 {
		t.Fatalf("large adjacent evidence union was not split deterministically: %#v", spans)
	}
	for _, span := range spans {
		if span.ByteSize > reportilcontract.MaxSourceReadSpanBytes ||
			!utf8.Valid(contentBytes[span.Offset:span.Offset+span.ByteSize]) {
			t.Fatalf("split span violates the UTF-8 read ceiling: %#v", span)
		}
	}
}

func TestEvidencePacketRejectsMismatchedReadRange(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, _ := compileEvidencePacketFixture(t, author, catalog, readable)
	spans, err := evidenceSourceReadSpans(catalog, narrative.EvidencePackets)
	if err != nil {
		t.Fatal(err)
	}
	receipt := reportilcontract.SourceReadReceipt{
		SourceKeys:           []string{spans[0].SourceKey},
		ReturnedContentBytes: spans[0].ByteSize,
		ReadBytesBySource:    map[string]int{spans[0].SourceKey: spans[0].ByteSize},
		ReadRanges: []reportilcontract.SourceReadRange{{
			SourceKey: spans[0].SourceKey,
			Offset:    spans[0].Offset + 1,
			ByteSize:  spans[0].ByteSize,
		}},
	}
	if err := validateEvidenceSourceReads(catalog, narrative.EvidencePackets, receipt); err == nil ||
		!strings.Contains(err.Error(), "does not match") {
		t.Fatalf("mismatched source read range was accepted: %v", err)
	}
}

func TestEvidencePacketDurableNarrativeOmitsSourceExcerpt(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, _ := compileEvidencePacketFixture(t, author, catalog, readable)
	serialized := string(mustMarshal(narrative))
	for _, sourceExcerpt := range []string{
		"ARCHIVAL RECORD: fortress was rebuilt in 1580",
		"rebuilt in 1580; stone walls followed",
	} {
		if strings.Contains(serialized, sourceExcerpt) {
			t.Fatalf("durable narrative contains source excerpt: %s", serialized)
		}
	}
	if !strings.Contains(serialized, "source_excerpt_sha256") {
		t.Fatalf("durable narrative lacks content-free source receipt: %s", serialized)
	}
}

func TestReaderEvidencePromptBindsEveryClaimWithoutSourceExcerpt(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, document := compileEvidencePacketFixture(t, author, catalog, readable)
	prompt := readerPrompt(
		ProductConfig{MissionObjective: "Explain the phases."},
		document,
		catalog,
		narrative.EvidencePackets,
	)
	for _, required := range []string{
		"[evidence_001]", "[evidence_002]", "source=source_001",
		"manuscript=section_01/block_01",
		fmt.Sprintf("source_offset=%d source_byte_size=%d", narrative.EvidencePackets[0].SourceOffset, narrative.EvidencePackets[0].SourceByteSize),
		fmt.Sprintf("source_offset=%d source_byte_size=%d", narrative.EvidencePackets[1].SourceOffset, narrative.EvidencePackets[1].SourceByteSize),
		"exact authored claim to check: The fortress was rebuilt in 1580.",
		"exact authored claim to check: The stone walls followed in 1600.",
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("reader evidence prompt lacks %q: %s", required, prompt)
		}
	}
	for _, sourceExcerpt := range []string{
		"ARCHIVAL RECORD: fortress was rebuilt in 1580",
		"rebuilt in 1580; stone walls followed",
	} {
		if strings.Contains(prompt, sourceExcerpt) {
			t.Fatalf("reader evidence prompt contains raw source excerpt: %s", prompt)
		}
	}
}

func TestEvidenceSupportCoverageUsesGranularCodesAndPacketAliases(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, document := compileEvidencePacketFixture(t, author, catalog, readable)
	packets := narrative.EvidencePackets
	first, second := packets[0].Claim, packets[1].Claim
	validCoverage := func() map[string]readerEvidenceSupportCoverageDraft {
		return map[string]readerEvidenceSupportCoverageDraft{
			"evidence_001": {
				EvidenceReceipt: evidenceReceipt(packets[0]),
				SectionKey:      "section_01", BlockKey: "block_01",
				SupportLevel: "direct_source_statement", CoverageQuote: &first,
			},
			"evidence_002": {
				EvidenceReceipt: evidenceReceipt(packets[1]),
				SectionKey:      "section_01", BlockKey: "block_01",
				SupportLevel: "direct_source_statement", CoverageQuote: &second,
			},
		}
	}
	assertFailure := func(
		t *testing.T,
		drafts map[string]readerEvidenceSupportCoverageDraft,
		edited Document,
		code reportexecution.ProviderValidationCode,
		aliases ...string,
	) {
		t.Helper()
		_, err := compileEvidenceSupportCoverage(edited, drafts, packets)
		if err == nil || providerValidationCode(err) != code ||
			!reflect.DeepEqual(providerValidationPacketAliases(err), aliases) {
			t.Fatalf(
				"support failure code/aliases = %q/%#v, want %q/%#v, err=%v",
				providerValidationCode(err), providerValidationPacketAliases(err),
				code, aliases, err,
			)
		}
	}

	inventory := validCoverage()
	delete(inventory, "evidence_002")
	assertFailure(t, inventory, document, reportexecution.ProviderValidationCodeEvidenceSupportInventory, "evidence_002")

	receipt := validCoverage()
	entry := receipt["evidence_001"]
	entry.EvidenceReceipt = "invalid"
	receipt["evidence_001"] = entry
	assertFailure(t, receipt, document, reportexecution.ProviderValidationCodeEvidenceSupportReceipt, "evidence_001")

	target := validCoverage()
	entry = target["evidence_001"]
	entry.SectionKey = "section_99"
	target["evidence_001"] = entry
	assertFailure(t, target, document, reportexecution.ProviderValidationCodeEvidenceSupportTarget, "evidence_001")

	binding := validCoverage()
	entry = binding["evidence_001"]
	entry.SectionKey, entry.BlockKey = "section_02", "block_01"
	binding["evidence_001"] = entry
	assertFailure(t, binding, document, reportexecution.ProviderValidationCodeEvidenceSupportBinding, "evidence_001")

	quote := validCoverage()
	entry = quote["evidence_002"]
	entry.CoverageQuote = nil
	quote["evidence_002"] = entry
	assertFailure(t, quote, document, reportexecution.ProviderValidationCodeEvidenceSupportQuote, "evidence_002")

	unsupported := validCoverage()
	entry = unsupported["evidence_001"]
	entry.SupportLevel, entry.CoverageQuote = "unsupported_removed", nil
	unsupported["evidence_001"] = entry
	assertFailure(t, unsupported, document, reportexecution.ProviderValidationCodeEvidenceSupportUnsupported, "evidence_001")

	level := validCoverage()
	entry = level["evidence_002"]
	entry.SupportLevel = "unsupported"
	level["evidence_002"] = entry
	assertFailure(t, level, document, reportexecution.ProviderValidationCodeEvidenceSupportLevel, "evidence_002")

	incomplete := document
	incomplete.Blocks = append([]Block(nil), document.Blocks...)
	for index := range incomplete.Blocks {
		if incomplete.Blocks[index].NodeID == packets[0].AuthorNodeID {
			incomplete.Blocks[index].Prose += " An uncovered factual addition remains."
		}
	}
	assertFailure(
		t, validCoverage(), incomplete,
		reportexecution.ProviderValidationCodeEvidenceSupportCoverage,
		"evidence_001", "evidence_002",
	)
}

func TestEvidenceSupportCoverageRejectsRetainedUnsupportedClaim(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, document := compileEvidencePacketFixture(t, author, catalog, readable)
	packets := narrative.EvidencePackets
	base := unchangedReaderDraft(document)
	secondQuote := packets[1].Claim
	base.EvidenceSupportCoverage = map[string]readerEvidenceSupportCoverageDraft{
		"evidence_001": {
			EvidenceReceipt: evidenceReceipt(packets[0]),
			SectionKey:      "section_01", BlockKey: "block_01",
			SupportLevel: "unsupported_removed", CoverageQuote: nil,
		},
		"evidence_002": {
			EvidenceReceipt: evidenceReceipt(packets[1]),
			SectionKey:      "section_01", BlockKey: "block_01",
			SupportLevel: "direct_source_statement", CoverageQuote: &secondQuote,
		},
	}
	if _, _, err := compileReaderDraft(document, base, "Explain the phases.", packets); err == nil {
		t.Fatal("retained claim marked unsupported_removed was accepted")
	}
}

func TestEvidenceSupportCoverageRejectsReassignedClaimToSourceBoundBlock(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, document := compileEvidencePacketFixture(t, author, catalog, readable)
	packets := narrative.EvidencePackets
	sections := readerSections(document)
	original := sections[0].Blocks[0]
	reassigned := sections[1].Blocks[0]
	for index := range document.Blocks {
		if document.Blocks[index].NodeID == reassigned.NodeID {
			document.Blocks[index].Prose = packets[0].Claim
			document.Blocks[index].EvidenceRefs = append([]string(nil), original.EvidenceRefs...)
		}
	}
	base := unchangedReaderDraft(document)
	base.EvidenceSupportCoverage = map[string]readerEvidenceSupportCoverageDraft{
		"evidence_001": {
			EvidenceReceipt: evidenceReceipt(packets[0]),
			SectionKey:      "section_02", BlockKey: "block_01",
			SupportLevel: "direct_source_statement",
		},
		"evidence_002": {
			EvidenceReceipt: evidenceReceipt(packets[1]),
			SectionKey:      "section_01", BlockKey: "block_01",
			SupportLevel: "direct_source_statement",
		},
	}
	base.PreservationCoverage = map[string]readerPreservationCoverageDraft{
		"preserve_01": {
			EvidenceReceipt: evidenceReceipt(packets[0]),
			SectionKey:      "section_02", BlockKey: "block_01",
		},
	}
	if _, _, err := compileReaderDraft(document, base, "Explain the phases.", packets); err == nil ||
		!strings.Contains(err.Error(), "changed source binding") {
		t.Fatalf("claim reassigned to another source-bound block was accepted: %v", err)
	}
}

func TestEvidenceSupportCoverageAcceptsRemovedUnsupportedNonPreservedClaim(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, document := compileEvidencePacketFixture(t, author, catalog, readable)
	packets := narrative.EvidencePackets
	base := unchangedReaderDraft(document)
	base.SectionEdits = cloneReaderSectionEdits(base.SectionEdits)
	value := "The fortress was rebuilt in 1580."
	coverageQuote := value
	base.SectionEdits["section_01"].Blocks["block_01"] = readerBlockDraft{
		OriginalSHA256: readerBlockSHA256(readerSections(document)[0].Blocks[0]),
		Prose:          &value,
	}
	base.EvidenceSupportCoverage = map[string]readerEvidenceSupportCoverageDraft{
		"evidence_001": {
			EvidenceReceipt: evidenceReceipt(packets[0]),
			SectionKey:      "section_01", BlockKey: "block_01",
			SupportLevel: "direct_source_statement", CoverageQuote: &coverageQuote,
		},
		"evidence_002": {
			EvidenceReceipt: evidenceReceipt(packets[1]),
			SectionKey:      "section_01", BlockKey: "block_01",
			SupportLevel: "unsupported_removed", CoverageQuote: nil,
		},
	}
	base.PreservationCoverage = map[string]readerPreservationCoverageDraft{
		"preserve_01": {
			EvidenceReceipt: evidenceReceipt(packets[0]),
			SectionKey:      "section_01", BlockKey: "block_01",
		},
	}
	if _, _, err := compileReaderDraft(document, base, "Explain the phases.", packets); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceSupportCoverageBindsSupportedParaphraseToFinalQuote(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, document := compileEvidencePacketFixture(t, author, catalog, readable)
	packets := narrative.EvidencePackets
	base := unchangedReaderDraft(document)
	base.SectionEdits = cloneReaderSectionEdits(base.SectionEdits)
	preserved := packets[0].Claim
	paraphrase := "Stone fortifications were added in 1600."
	finalProse := preserved + " " + paraphrase
	base.SectionEdits["section_01"].Blocks["block_01"] = readerBlockDraft{
		OriginalSHA256: readerBlockSHA256(readerSections(document)[0].Blocks[0]),
		Prose:          &finalProse,
	}
	base.EvidenceSupportCoverage = map[string]readerEvidenceSupportCoverageDraft{
		"evidence_001": {
			EvidenceReceipt: evidenceReceipt(packets[0]),
			SectionKey:      "section_01", BlockKey: "block_01",
			SupportLevel: "direct_source_statement", CoverageQuote: &preserved,
		},
		"evidence_002": {
			EvidenceReceipt: evidenceReceipt(packets[1]),
			SectionKey:      "section_01", BlockKey: "block_01",
			SupportLevel: "direct_source_statement", CoverageQuote: &paraphrase,
		},
	}
	base.PreservationCoverage = map[string]readerPreservationCoverageDraft{
		"preserve_01": {
			EvidenceReceipt: evidenceReceipt(packets[0]),
			SectionKey:      "section_01", BlockKey: "block_01",
		},
	}
	edited, attestation, err := compileReaderDraft(document, base, "Explain the phases.", packets)
	if err != nil {
		t.Fatal(err)
	}
	coverage := attestation.EvidenceSupportCoverage[1]
	if coverage.ClaimSHA != packets[1].ClaimSHA ||
		coverage.CoverageQuoteSHA != SHA256([]byte(paraphrase)) ||
		coverage.CoverageValueIndex != 0 ||
		coverage.CoverageByteOffset != strings.Index(finalProse, paraphrase) ||
		coverage.CoverageByteSize != len([]byte(paraphrase)) ||
		!blockQuoteRangeValid(
			documentBlockByID(edited, coverage.CoveredNodeID),
			coverage.CoverageValueIndex,
			coverage.CoverageByteOffset,
			coverage.CoverageByteSize,
			coverage.CoverageQuoteSHA,
		) {
		t.Fatalf("final quote lineage = %#v", coverage)
	}
}

func TestEvidenceSupportCoverageRejectsMissingOrStaleFinalQuote(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, document := compileEvidencePacketFixture(t, author, catalog, readable)
	packets := narrative.EvidencePackets
	base := unchangedReaderDraft(document)
	first := packets[0].Claim
	second := packets[1].Claim
	base.EvidenceSupportCoverage = map[string]readerEvidenceSupportCoverageDraft{
		"evidence_001": {
			EvidenceReceipt: evidenceReceipt(packets[0]),
			SectionKey:      "section_01", BlockKey: "block_01",
			SupportLevel: "direct_source_statement", CoverageQuote: &first,
		},
		"evidence_002": {
			EvidenceReceipt: evidenceReceipt(packets[1]),
			SectionKey:      "section_01", BlockKey: "block_01",
			SupportLevel: "direct_source_statement", CoverageQuote: &second,
		},
	}
	base.PreservationCoverage = map[string]readerPreservationCoverageDraft{
		"preserve_01": {
			EvidenceReceipt: evidenceReceipt(packets[0]),
			SectionKey:      "section_01", BlockKey: "block_01",
		},
	}
	missing := base
	missing.EvidenceSupportCoverage = map[string]readerEvidenceSupportCoverageDraft{}
	for key, value := range base.EvidenceSupportCoverage {
		missing.EvidenceSupportCoverage[key] = value
	}
	withoutQuote := missing.EvidenceSupportCoverage["evidence_002"]
	withoutQuote.CoverageQuote = nil
	missing.EvidenceSupportCoverage["evidence_002"] = withoutQuote
	if _, _, err := compileReaderDraft(document, missing, "Explain the phases.", packets); err == nil ||
		providerValidationCode(err) != reportexecution.ProviderValidationCodeEvidenceSupportQuote ||
		!reflect.DeepEqual(providerValidationPacketAliases(err), []string{"evidence_002"}) {
		t.Fatalf("missing final quote accepted: %v", err)
	}

	edited, attestation, err := compileReaderDraft(document, base, "Explain the phases.", packets)
	if err != nil {
		t.Fatal(err)
	}
	attestation.SchemaVersion = AttestationSchemaVersion
	attestation.DocumentID = edited.DocumentID
	attestation.RevisionID = edited.RevisionID
	projection, _, err := RenderMarkdown(edited)
	if err != nil {
		t.Fatal(err)
	}
	attestation.LinearProjectionSHA256 = SHA256(projection)
	attestation.EvidenceSupportCoverage[1].CoverageByteOffset++
	if err := validateEvidenceLineage(&narrative, &attestation, edited); err == nil {
		t.Fatal("stale final quote range was accepted")
	}
}

func TestBlockQuoteRangeRejectsSplitUTF8Character(t *testing.T) {
	block := Block{NodeID: "prose.utf8", Kind: "prose", Prose: "éé"}
	value := []byte(block.Prose)
	if !blockQuoteRangeValid(block, 0, 0, 2, SHA256(value[:2])) {
		t.Fatal("complete UTF-8 quote was rejected")
	}
	if blockQuoteRangeValid(block, 0, 1, 1, SHA256(value[1:2])) ||
		blockQuoteRangeValid(block, 0, 0, 3, SHA256(value[:3])) {
		t.Fatal("quote splitting a UTF-8 character was accepted")
	}
}

func TestBlockSupportCoverageCoversRepeatedExactQuoteContent(t *testing.T) {
	block := Block{NodeID: "prose.repeated", Kind: "prose", Prose: "Fact Fact"}
	coverage := []EvidenceSupportCoverageReceipt{{
		CoveredNodeID: block.NodeID, SupportLevel: "direct_source_statement",
		CoverageQuoteSHA: SHA256([]byte("Fact")), CoverageValueIndex: 0,
		CoverageByteOffset: 0, CoverageByteSize: len([]byte("Fact")),
	}}
	if !blockSupportCoverageComplete(block, coverage) {
		t.Fatal("identical repeated quote content was not covered by its support receipt")
	}
}

func TestPreservationCoverageRejectsDeletedParaphrasedOrReassignedClaim(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, document := compileEvidencePacketFixture(t, author, catalog, readable)
	packet := narrative.EvidencePackets[0]
	base := unchangedReaderDraft(document)
	base.PreservationCoverage = map[string]readerPreservationCoverageDraft{
		"preserve_01": {
			EvidenceReceipt: evidenceReceipt(packet),
			SectionKey:      "section_01", BlockKey: "block_01",
		},
	}

	deleted := base
	deleted.SectionEdits = cloneReaderSectionEdits(base.SectionEdits)
	value := "The stone walls followed in 1600."
	deleted.SectionEdits["section_01"].Blocks["block_01"] = readerBlockDraft{
		OriginalSHA256: readerBlockSHA256(readerSections(document)[0].Blocks[0]),
		Prose:          &value,
	}
	if _, _, err := compileReaderDraft(document, deleted, "Explain the phases.", narrative.EvidencePackets); err == nil {
		t.Fatal("deleted preserved claim was accepted")
	}

	paraphrased := base
	paraphrased.SectionEdits = cloneReaderSectionEdits(base.SectionEdits)
	value = "The fortress underwent reconstruction during 1580. The stone walls followed in 1600."
	paraphrased.SectionEdits["section_01"].Blocks["block_01"] = readerBlockDraft{
		OriginalSHA256: readerBlockSHA256(readerSections(document)[0].Blocks[0]),
		Prose:          &value,
	}
	if _, _, err := compileReaderDraft(document, paraphrased, "Explain the phases.", narrative.EvidencePackets); err == nil {
		t.Fatal("paraphrased preserved claim was accepted")
	}

	reassigned := base
	reassigned.PreservationCoverage = map[string]readerPreservationCoverageDraft{
		"preserve_01": {
			EvidenceReceipt: evidenceReceipt(packet),
			SectionKey:      "section_02", BlockKey: "block_01",
		},
	}
	if _, _, err := compileReaderDraft(document, reassigned, "Explain the phases.", narrative.EvidencePackets); err == nil {
		t.Fatal("preserved claim reassigned to a different evidence block was accepted")
	}
}

func TestBundleRejectsPreservationLineageMismatch(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, document := compileEvidencePacketFixture(t, author, catalog, readable)
	packet := narrative.EvidencePackets[0]
	attestation := FlowAttestation{
		EvidenceSupportCoverage: []EvidenceSupportCoverageReceipt{
			{
				EvidenceID:       narrative.EvidencePackets[0].EvidenceID,
				EvidenceReceipt:  evidenceReceipt(narrative.EvidencePackets[0]),
				SourceExcerptSHA: narrative.EvidencePackets[0].SourceExcerptSHA,
				ClaimSHA:         narrative.EvidencePackets[0].ClaimSHA,
				CoveredNodeID:    narrative.EvidencePackets[0].AuthorNodeID,
				SupportLevel:     "direct_source_statement",
			},
			{
				EvidenceID:       narrative.EvidencePackets[1].EvidenceID,
				EvidenceReceipt:  evidenceReceipt(narrative.EvidencePackets[1]),
				SourceExcerptSHA: narrative.EvidencePackets[1].SourceExcerptSHA,
				ClaimSHA:         narrative.EvidencePackets[1].ClaimSHA,
				CoveredNodeID:    narrative.EvidencePackets[1].AuthorNodeID,
				SupportLevel:     "direct_source_statement",
			},
		},
		PreservationCoverage: []PreservationCoverageReceipt{{
			EvidenceID: packet.EvidenceID, EvidenceReceipt: strings.Repeat("0", 64),
			SourceExcerptSHA: packet.SourceExcerptSHA, CoveredNodeID: packet.AuthorNodeID,
			CoverageQuoteSHA: packet.ClaimSHA,
		}},
	}
	if err := validateEvidenceLineage(&narrative, &attestation, document); err == nil ||
		!strings.Contains(err.Error(), "stale or invalid") {
		t.Fatalf("mismatched evidence lineage was accepted: %v", err)
	}
}

func TestBundleRejectsEvidenceCoverageReassignedToAnotherSourceBoundBlock(t *testing.T) {
	catalog, readable, author := evidencePacketFixture(t)
	narrative, document := compileEvidencePacketFixture(t, author, catalog, readable)
	packets := narrative.EvidencePackets
	sections := readerSections(document)
	original := sections[0].Blocks[0]
	reassigned := sections[1].Blocks[0]
	for index := range document.Blocks {
		if document.Blocks[index].NodeID == reassigned.NodeID {
			document.Blocks[index].Prose = packets[0].Claim
			document.Blocks[index].EvidenceRefs = append([]string(nil), original.EvidenceRefs...)
		}
	}
	attestation := FlowAttestation{
		EvidenceSupportCoverage: []EvidenceSupportCoverageReceipt{
			{
				EvidenceID: packets[0].EvidenceID, EvidenceReceipt: evidenceReceipt(packets[0]),
				SourceExcerptSHA: packets[0].SourceExcerptSHA, ClaimSHA: packets[0].ClaimSHA,
				CoveredNodeID: reassigned.NodeID, SupportLevel: "direct_source_statement",
			},
			{
				EvidenceID: packets[1].EvidenceID, EvidenceReceipt: evidenceReceipt(packets[1]),
				SourceExcerptSHA: packets[1].SourceExcerptSHA, ClaimSHA: packets[1].ClaimSHA,
				CoveredNodeID: original.NodeID, SupportLevel: "direct_source_statement",
			},
		},
		PreservationCoverage: []PreservationCoverageReceipt{{
			EvidenceID: packets[0].EvidenceID, EvidenceReceipt: evidenceReceipt(packets[0]),
			SourceExcerptSHA: packets[0].SourceExcerptSHA, CoveredNodeID: reassigned.NodeID,
			CoverageQuoteSHA: packets[0].ClaimSHA,
		}},
	}
	if err := validateEvidenceLineage(&narrative, &attestation, document); err == nil ||
		!strings.Contains(err.Error(), "stale or invalid") {
		t.Fatalf("reassigned bundle evidence coverage was accepted: %v", err)
	}
}

func cloneReaderSectionEdits(source map[string]readerSectionDraft) map[string]readerSectionDraft {
	result := make(map[string]readerSectionDraft, len(source))
	for sectionKey, section := range source {
		cloned := section
		cloned.Blocks = make(map[string]readerBlockDraft, len(section.Blocks))
		for blockKey, block := range section.Blocks {
			cloned.Blocks[blockKey] = block
		}
		result[sectionKey] = cloned
	}
	return result
}

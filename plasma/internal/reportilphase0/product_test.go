package reportilphase0

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentcapability"
	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

type productSourceReader struct {
	sources   []reportilcontract.SourceSnapshot
	artifacts map[string]reportilcontract.Artifact
	reads     map[string]reportilcontract.LocalRead
}

func (r productSourceReader) ListSourceSnapshots(context.Context, string) ([]reportilcontract.SourceSnapshot, error) {
	return append([]reportilcontract.SourceSnapshot(nil), r.sources...), nil
}
func (r productSourceReader) GetArtifact(_ context.Context, id string) (reportilcontract.Artifact, error) {
	return r.artifacts[id], nil
}
func (r productSourceReader) ReadLive(_ context.Context, _, snapshotID string, _ int64) (reportilcontract.LocalRead, error) {
	return r.reads[snapshotID], nil
}

type recordingProvider struct {
	requests []agentexec.AgentRequest
	outputs  []string
	results  []agentexec.AgentResult
	errs     []error
}

func (p *recordingProvider) Run(_ context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
	p.requests = append(p.requests, req)
	index := len(p.requests) - 1
	if index < len(p.errs) && p.errs[index] != nil {
		return agentexec.AgentResult{}, p.errs[index]
	}
	if index < len(p.results) {
		result := p.results[index]
		if index < len(p.outputs) {
			result.Text = p.outputs[index]
		}
		return result, nil
	}
	if index >= len(p.outputs) {
		return agentexec.AgentResult{}, nil
	}
	return agentexec.AgentResult{Text: p.outputs[index]}, nil
}

type acceptingSourceReadVerifier struct {
	calls        []string
	sourceQuotes map[string]reportilcontract.SourceQuoteReceipt
}

type fixedSourceReadVerifier struct {
	receipt reportilcontract.SourceReadReceipt
	calls   int
}

type fixedAuthorDocumentReader struct {
	memory                       reportilcontract.EditorialMemory
	document                     reportilcontract.AuthorDocument
	continuity                   reportilcontract.AuthorDocument
	publication                  reportilcontract.AuthorDocument
	publications                 []reportilcontract.AuthorDocument
	continuityReplacements       int
	publicationReplacements      int
	publicationReplacementCounts []int
	err                          error
	continuityErr                error
	publicationErr               error
	calls                        int
	continuityCalls              int
	publicationCalls             int
}

func (r *fixedAuthorDocumentReader) ReadReportILEditorialMemory(_ context.Context, _, _ string, catalog reportilcontract.SourceCatalog) (reportilcontract.EditorialMemory, reportilcontract.EditorialMemoryReceipt, error) {
	memory := r.memory
	if memory.SchemaVersion == "" {
		memory = testEditorialMemory(catalog)
	}
	return memory, reportilcontract.EditorialMemoryReceipt{WorkspaceID: "ilm_fixture", ArtifactID: "art_memory", SHA256: strings.Repeat("d", 64), ByteSize: 1, Revision: 1, Accounts: len(memory.Accounts)}, nil
}

func (r *fixedAuthorDocumentReader) ReadReportILAuthorDocument(_ context.Context, _, _ string, _ reportilcontract.SourceCatalog) (reportilcontract.AuthorDocument, reportilcontract.AuthorWorkspaceReceipt, error) {
	r.calls++
	if r.err != nil {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, r.err
	}
	return r.document, reportilcontract.AuthorWorkspaceReceipt{WorkspaceID: "ilw_fixture", ArtifactID: "art_fixture", SHA256: strings.Repeat("a", 64), ByteSize: 1, Revision: 1, Stage: "il_narrative"}, nil
}

func (r *fixedAuthorDocumentReader) ReadReportILLongFormPlan(_ context.Context, _, _ string, _ reportilcontract.SourceCatalog) (reportilcontract.LongFormPlan, reportilcontract.LongFormPlanReceipt, error) {
	return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, r.err
}

func (r *fixedAuthorDocumentReader) ReadReportILLongFormStageDocument(_ context.Context, _, _, stage string, _ reportilcontract.SourceCatalog) (reportilcontract.AuthorDocument, reportilcontract.AuthorWorkspaceReceipt, error) {
	return r.document, reportilcontract.AuthorWorkspaceReceipt{WorkspaceID: "ilw_long_form", ArtifactID: "art_long_form", SHA256: strings.Repeat("e", 64), ByteSize: 1, Revision: 1, Stage: stage}, r.err
}

func (r *fixedAuthorDocumentReader) ReadReportILContinuityDocument(_ context.Context, _, _ string, _ reportilcontract.SourceCatalog) (reportilcontract.AuthorDocument, reportilcontract.AuthorWorkspaceReceipt, error) {
	r.continuityCalls++
	if r.continuityErr != nil {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, r.continuityErr
	}
	document := r.continuity
	if document.SchemaVersion == "" {
		document = r.document
	}
	return document, reportilcontract.AuthorWorkspaceReceipt{WorkspaceID: "ilw_continuity", ArtifactID: "art_continuity", SHA256: strings.Repeat("b", 64), ByteSize: 1, Revision: 1, Stage: "il_continuity", Replacements: r.continuityReplacements}, nil
}

func (r *fixedAuthorDocumentReader) ReadReportILPublicationDocument(_ context.Context, _, _ string, _ reportilcontract.SourceCatalog) (reportilcontract.AuthorDocument, reportilcontract.AuthorWorkspaceReceipt, error) {
	r.publicationCalls++
	if r.publicationErr != nil {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, r.publicationErr
	}
	document := r.publication
	if r.publicationCalls <= len(r.publications) {
		document = r.publications[r.publicationCalls-1]
	}
	if document.SchemaVersion == "" {
		document = r.continuity
	}
	if document.SchemaVersion == "" {
		document = r.document
	}
	replacements := r.publicationReplacements
	if r.publicationCalls <= len(r.publicationReplacementCounts) {
		replacements = r.publicationReplacementCounts[r.publicationCalls-1]
	}
	return document, reportilcontract.AuthorWorkspaceReceipt{
		WorkspaceID: fmt.Sprintf("ilw_publication_%d", r.publicationCalls),
		ArtifactID:  fmt.Sprintf("art_publication_%d", r.publicationCalls),
		SHA256:      strings.Repeat(string(rune('b'+r.publicationCalls)), 64), ByteSize: 1,
		Revision: 1, Stage: "il_reader", Replacements: replacements,
	}, nil
}

func testEditorialMemory(catalog reportilcontract.SourceCatalog) reportilcontract.EditorialMemory {
	memory := reportilcontract.EditorialMemory{
		SchemaVersion: reportilcontract.EditorialMemorySchemaVersion,
		Language:      "en",
	}
	for index, source := range catalog.Sources {
		memory.Accounts = append(memory.Accounts, reportilcontract.EditorialAccount{
			AccountKey: fmt.Sprintf("account_%03d", index+1), Importance: "essential",
			Account:    fmt.Sprintf("Bounded account %d supports the report.", index+1),
			SourceKeys: []string{source.SourceKey},
		})
	}
	return memory
}

func testAuthorDocument() reportilcontract.AuthorDocument {
	return reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.AuthorDocumentSchemaVersion,
		Title:         "Bounded result", Language: "en",
		Sections: []reportilcontract.AuthorSection{
			{SectionKey: "section_001", Title: "Answer", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "section_001.block_001", Kind: "prose", Prose: "One bounded fact defines the result.", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}}}},
			{SectionKey: "section_002", Title: "Judgment", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "section_002.block_001", Kind: "prose", Prose: "That fact supports one bounded conclusion.", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
}

func canonicalReaderTestDocument(t *testing.T, missionID string) (Document, reportilcontract.SourceCatalog, reportilcontract.AuthorDocument) {
	t.Helper()
	catalog := testSourceCatalog(t, missionID)
	authored := testAuthorDocument()
	_, document, err := compileReportFirstAuthorDraft(
		reportFirstDraftFromAuthorDocument(authored), "narrative_reader", "doc_reader",
		"Explain the answer.", catalog, testSourceReadReceipt(catalog), nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return document, catalog, authored
}

func authorDocumentFromIL(document Document, catalog reportilcontract.SourceCatalog) reportilcontract.AuthorDocument {
	authored := reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.AuthorDocumentSchemaVersion,
		Title:         document.Title, Language: document.Language,
	}
	for _, section := range readerSections(document) {
		authorSection := reportilcontract.AuthorSection{
			SectionKey: fmt.Sprintf("section_%03d", len(authored.Sections)+1),
			Title:      section.Title,
		}
		for _, block := range section.Blocks {
			authorBlock := reportilcontract.AuthorBlock{
				BlockKey: fmt.Sprintf("%s.block_%03d", authorSection.SectionKey, len(authorSection.Blocks)+1),
				Kind:     block.Kind, Prose: block.Prose, Items: append([]string(nil), block.Items...),
				Code: block.Code, EvidenceSourceKeys: []string{"source_001"},
			}
			if block.Language != "" {
				authorBlock.Language = authorStringPointer(block.Language)
			}
			if block.Table != nil {
				table := &reportilcontract.AuthorTable{Columns: append([]string(nil), block.Table.Columns...)}
				if block.Table.Caption != "" {
					table.Caption = authorStringPointer(block.Table.Caption)
				}
				for _, row := range block.Table.Rows {
					table.Rows = append(table.Rows, reportilcontract.AuthorTableRow{Cells: append([]string(nil), row...)})
				}
				authorBlock.Table = table
			}
			for _, refID := range block.EvidenceRefs {
				for _, ref := range document.References {
					if ref.RefID != refID || !strings.HasPrefix(ref.Target, "accepted-source:") {
						continue
					}
					ordinal, _ := strconv.Atoi(strings.TrimPrefix(ref.Target, "accepted-source:"))
					for _, entry := range catalog.Sources {
						if entry.AcceptedOrdinal == ordinal {
							authorBlock.EvidenceSourceKeys = append(authorBlock.EvidenceSourceKeys[:0], entry.SourceKey)
						}
					}
				}
			}
			authorSection.Blocks = append(authorSection.Blocks, authorBlock)
		}
		authored.Sections = append(authored.Sections, authorSection)
	}
	return authored
}

func (v *fixedSourceReadVerifier) VerifyReportILSourceRead(_ context.Context, _, _, _ string, _ reportilcontract.SourceCatalog) (reportilcontract.SourceReadReceipt, error) {
	v.calls++
	return v.receipt, nil
}

func (v *acceptingSourceReadVerifier) VerifyReportILSourceRead(_ context.Context, missionID, toolSessionID, stage string, catalog reportilcontract.SourceCatalog) (reportilcontract.SourceReadReceipt, error) {
	if missionID != catalog.MissionID || !strings.HasPrefix(toolSessionID, "ses_") ||
		(stage != "il_source_selection" && stage != "il_editorial_memory" && stage != "il_narrative" && stage != "il_continuity" && stage != "il_reader" && stage != "il_document" && stage != "il_flow") {
		return reportilcontract.SourceReadReceipt{}, errors.New("invalid source-read verification binding")
	}
	if err := reportilcontract.ValidateSourceCatalog(catalog); err != nil {
		return reportilcontract.SourceReadReceipt{}, err
	}
	v.calls = append(v.calls, toolSessionID)
	receipt := reportilcontract.SourceReadReceipt{
		ReadBytesBySource: map[string]int{},
		SourceQuotes:      map[string]reportilcontract.SourceQuoteReceipt{},
	}
	for key, quote := range v.sourceQuotes {
		receipt.SourceQuotes[key] = quote
	}
	for _, entry := range catalog.Sources {
		receipt.SourceKeys = append(receipt.SourceKeys, entry.SourceKey)
		receipt.FullyReadSourceKeys = append(receipt.FullyReadSourceKeys, entry.SourceKey)
		receipt.ReturnedContentBytes += entry.ReadableBytes
		receipt.ReadBytesBySource[entry.SourceKey] = entry.ReadableBytes
		receipt.ReadRanges = append(receipt.ReadRanges, reportilcontract.SourceReadRange{
			SourceKey: entry.SourceKey,
			Offset:    0,
			ByteSize:  entry.ReadableBytes,
		})
	}
	return receipt, nil
}

func testSourceCatalog(t *testing.T, missionID string) reportilcontract.SourceCatalog {
	t.Helper()
	content := []byte("fixture source content that must never enter a provider prompt")
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: missionID,
		Sources: []reportilcontract.SourceCatalogEntry{{
			SourceKey:       "source_001",
			SnapshotID:      "src_fixture",
			SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_fixture", sha256Hex(content)),
			ContentHash:     sha256Hex(content),
			RetrievalPolicy: "snapshot_only",
			Artifacts: []reportilcontract.SourceCatalogArtifact{{
				ArtifactID: "art_fixture",
				SHA256:     sha256Hex(content),
				ByteSize:   int64(len(content)),
				MediaType:  "text/plain",
			}},
			ReadableSHA256: sha256Hex(content),
			ReadableBytes:  len(content),
			Extraction:     "stored_text",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func testSourceReadReceipt(catalog reportilcontract.SourceCatalog) reportilcontract.SourceReadReceipt {
	receipt := reportilcontract.SourceReadReceipt{ReadBytesBySource: map[string]int{}}
	for _, entry := range catalog.Sources {
		receipt.SourceKeys = append(receipt.SourceKeys, entry.SourceKey)
		receipt.FullyReadSourceKeys = append(receipt.FullyReadSourceKeys, entry.SourceKey)
		receipt.ReturnedContentBytes += entry.ReadableBytes
		receipt.ReadBytesBySource[entry.SourceKey] = entry.ReadableBytes
		receipt.ReadRanges = append(receipt.ReadRanges, reportilcontract.SourceReadRange{
			SourceKey: entry.SourceKey,
			Offset:    0,
			ByteSize:  entry.ReadableBytes,
		})
	}
	return receipt
}

func testSourceCatalogWithCount(t *testing.T, missionID string, count int) reportilcontract.SourceCatalog {
	t.Helper()
	entries := make([]reportilcontract.SourceCatalogEntry, 0, count)
	for index := 0; index < count; index++ {
		content := []byte(fmt.Sprintf("fixture source %d content", index+1))
		entries = append(entries, reportilcontract.SourceCatalogEntry{
			SourceKey:       fmt.Sprintf("source_%03d", index+1),
			SnapshotID:      fmt.Sprintf("src_fixture_%03d", index+1),
			SnapshotReceipt: reportilcontract.SourceSnapshotReceipt(fmt.Sprintf("src_fixture_%03d", index+1), sha256Hex(content)),
			ContentHash:     sha256Hex(content),
			RetrievalPolicy: "snapshot_only",
			Artifacts: []reportilcontract.SourceCatalogArtifact{{
				ArtifactID: fmt.Sprintf("art_fixture_%03d", index+1), SHA256: sha256Hex(content), ByteSize: int64(len(content)), MediaType: "text/plain",
			}},
			ReadableSHA256: sha256Hex(content), ReadableBytes: len(content), Extraction: "stored_text",
		})
	}
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{MissionID: missionID, Sources: entries})
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestExperimentalCompilerVersionV68(t *testing.T) {
	if CompilerVersion != "plasma.report_il.phase0_compiler.v68" {
		t.Fatalf("compiler version = %q", CompilerVersion)
	}
	if ProductManifestSchemaVersion != "plasma.report_il.product_manifest.experimental.v36" {
		t.Fatalf("product manifest schema version = %q", ProductManifestSchemaVersion)
	}
}

func TestAuthorSchemasRequireOpaqueSourceReceipts(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_source_receipt")
	_, _, repairFrozen := evidencePacketFixture(t)
	repairFrozen.Sections[1].Blocks[0].EvidenceSourceKeys = []string{"source_001"}
	cases := map[string]struct {
		schemaRaw        []byte
		packetDefinition string
	}{
		"complete author": {
			schemaRaw:        providerAuthorSchemaBytes(catalog),
			packetDefinition: "author_evidence_draft",
		},
		"packet repair": {
			schemaRaw:        providerAuthorEvidenceRepairSchemaBytes(repairFrozen),
			packetDefinition: "author_evidence_repair_packet_draft",
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			var schema map[string]any
			if err := json.Unmarshal(test.schemaRaw, &schema); err != nil {
				t.Fatal(err)
			}
			definitions := schema["$defs"].(map[string]any)
			packet := definitions[test.packetDefinition].(map[string]any)
			properties := packet["properties"].(map[string]any)
			if properties["source_excerpt"] != nil {
				t.Fatal("provider schema still exposes source_excerpt")
			}
			receipt := properties["source_receipt"].(map[string]any)
			if receipt["pattern"] != `^quote_[a-f0-9]{64}$` {
				t.Fatalf("source receipt pattern = %#v", receipt["pattern"])
			}
		})
	}
}

func TestValidationProfilesSelectTruthfulAuthoringStages(t *testing.T) {
	for _, tc := range []struct {
		profile        string
		wantMemory     bool
		wantReader     bool
		wantContinuity bool
	}{
		{profile: ValidationProfileUnverified},
		{profile: ValidationProfileExploratory, wantMemory: true, wantReader: true},
		{profile: ValidationProfileStrict, wantMemory: true, wantReader: true, wantContinuity: true},
	} {
		t.Run(tc.profile, func(t *testing.T) {
			reader := &fixedAuthorDocumentReader{document: testAuthorDocument()}
			provider := &recordingProvider{}
			config := testSourceProductConfig(provider, &acceptingSourceReadVerifier{})
			catalog := testSourceCatalog(t, config.MissionID)
			config.ValidationProfile = tc.profile
			config.AuthorDocuments = reader
			if tc.profile == ValidationProfileUnverified {
				_, _, _, results, err := runDirectSourceAuthorStage(context.Background(), config, catalog, nil, false)
				if err != nil || len(results) != 1 {
					t.Fatalf("direct author stage err=%v results=%d", err, len(results))
				}
				request := provider.requests[len(provider.requests)-1]
				if request.ReportILSources == nil || request.ReportILSources.MaxReadBytes == 0 ||
					!slices.Contains(request.ExtraMCPTools, reportilcontract.AuthorDocumentAppendSourceTool) ||
					slices.Contains(request.ExtraMCPTools, reportilcontract.EditorialMemoryReadTool) {
					t.Fatalf("direct author request = %#v", request)
				}
			}
			plan := validationPlan(tc.profile)
			if plan.EditorialMemory != tc.wantMemory || plan.PublicationRead != tc.wantReader || plan.Continuity != tc.wantContinuity {
				t.Fatalf("validation plan = %#v", plan)
			}
		})
	}
}

func TestDirectSourceAuthorPromptMatchesKeyedReadContract(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_direct_prompt")
	config := testSourceProductConfig(&recordingProvider{}, &acceptingSourceReadVerifier{})
	prompt := directSourceAuthorPrompt(config, catalog, false)
	for _, required := range []string{
		reportilcontract.SourceListTool,
		reportilcontract.SourceReadTool,
		"every selected source_key",
		"offset 0",
		"exact returned next_offset",
		"until truncated is false",
		"max_bytes up to 65536",
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("direct author prompt lacks %q: %s", required, prompt)
		}
	}
	for _, forbidden := range []string{"remaining_sources", "read with {}", "server-ordered batch"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("direct author prompt contains batch instruction %q: %s", forbidden, prompt)
		}
	}
}

func TestProductManifestRecordsAuthoringAndValidationProfiles(t *testing.T) {
	manifest := ProductManifest{
		SchemaVersion:     ProductManifestSchemaVersion,
		CompilerVersion:   CompilerVersion,
		PipelineFamily:    PipelineFamily,
		AuthoringMode:     AuthoringModeLongForm,
		ValidationProfile: ValidationProfileUnverified,
	}
	encoded := mustMarshal(manifest)
	var decoded ProductManifest
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.AuthoringMode != AuthoringModeLongForm || decoded.ValidationProfile != ValidationProfileUnverified || decoded.EditorialMemory != nil ||
		!strings.Contains(string(encoded), `"authoring_mode":"long_form"`) ||
		!strings.Contains(string(encoded), `"validation_profile":"unverified"`) ||
		strings.Contains(string(encoded), `"editorial_memory"`) {
		t.Fatalf("manifest authoring/validation profiles = %s", encoded)
	}
}

func TestCompleteSourceCatalogBudgetFailsBeforeProviderWhenNoSourceCanFit(t *testing.T) {
	within := testSourceCatalogWithCount(t, "mis_budget_within", 5)
	remaining := reportilcontract.DefaultSourceAttemptReadBytes
	for index := range within.Sources {
		bytes := 1
		if index == len(within.Sources)-1 {
			bytes = remaining
		}
		within.Sources[index].ReadableBytes = bytes
		remaining -= bytes
	}
	within, err := reportilcontract.SealSourceCatalog(within)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateCompleteSourceCatalogBudget(within); err != nil {
		t.Fatal(err)
	}

	over := within
	over.Sources = append([]reportilcontract.SourceCatalogEntry(nil), within.Sources...)
	over.Sources[len(over.Sources)-1].ReadableBytes++
	over, err = reportilcontract.SealSourceCatalog(over)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateCompleteSourceCatalogBudget(over); err == nil {
		t.Fatal("source catalog over complete-read attempt ceiling was accepted")
	}

	content := []byte(strings.Repeat("x", reportilcontract.DefaultSourceAttemptReadBytes+1))
	hash := sha256Hex(content)
	sources := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{{
			SnapshotID: "src_budget", MissionID: "mis_budget", Active: true,
			ArtifactIDs: []string{"art_budget"}, ContentHash: hash,
		}},
		artifacts: map[string]reportilcontract.Artifact{
			"art_budget": {
				ArtifactID: "art_budget", MissionID: "mis_budget", MediaType: "text/plain",
				SHA256: hash, ByteSize: int64(len(content)), Content: content,
			},
		},
	}
	provider := &recordingProvider{}
	_, err = RunProduct(context.Background(), ProductConfig{
		MissionID: "mis_budget", PendingEventID: "evt_budget", Sources: sources,
		Provider: provider, VerifySourceRead: &acceptingSourceReadVerifier{},
		AuthorDocuments: &fixedAuthorDocumentReader{document: testAuthorDocument()},
		NewID:           func(prefix string) string { return prefix + "_budget" },
	})
	if err == nil || len(provider.requests) != 0 {
		t.Fatalf("unselectable oversized catalog = err=%v requests=%d", err, len(provider.requests))
	}
	var failure *reportexecution.StageFailureError
	if !errors.As(err, &failure) || failure.Kind != "il_source_selection" {
		t.Fatalf("oversized catalog failure = %T %#v", err, failure)
	}
}

func TestCompleteSourceReadAndDocumentCoverageRejectPartialFiveSourceWork(t *testing.T) {
	catalog := testSourceCatalogWithCount(t, "mis_coverage", 5)
	complete := testSourceReadReceipt(catalog)
	if err := validateCompleteSourceRead(catalog, complete); err != nil {
		t.Fatal(err)
	}
	partial := complete
	partial.FullyReadSourceKeys = append([]string(nil), complete.FullyReadSourceKeys[:4]...)
	if err := validateCompleteSourceRead(catalog, partial); err == nil {
		t.Fatal("partial frozen source byte coverage was accepted")
	}

	document := Document{}
	document.References = append(document.References, Reference{
		RefID: "ref.accepted_source.1", Kind: "footnote", Target: "accepted-source:001",
	})
	if err := validateDocumentSourceCoverage(document, catalog); err != nil {
		t.Fatal(err)
	}
	document.References = nil
	if err := validateDocumentSourceCoverage(document, catalog); err == nil {
		t.Fatal("document without any selected-source evidence was accepted")
	}
}

func testSourceProductConfig(provider Provider, verifier agentexec.ReportILSourceReadVerifier) ProductConfig {
	counts := map[string]int{}
	return ProductConfig{
		MissionID:        "mis_fixture",
		MissionObjective: "Explain the bounded result.",
		Title:            "Bounded result",
		PendingEventID:   "evt_pending",
		Provider:         provider,
		VerifySourceRead: verifier,
		AuthorDocuments:  &fixedAuthorDocumentReader{document: testAuthorDocument()},
		NewID: func(prefix string) string {
			counts[prefix]++
			return prefix + "_" + strconv.Itoa(counts[prefix])
		},
	}
}

func TestRunReaderPatchStageRequiresFinalizedPublicationWorkspace(t *testing.T) {
	document, catalog, _ := canonicalReaderTestDocument(t, "mis_reader_workspace_required")
	provider := &recordingProvider{outputs: []string{"workspace finalized"}}
	config := testSourceProductConfig(provider, &acceptingSourceReadVerifier{})
	config.MissionID = catalog.MissionID
	config.MissionObjective = "Explain the answer."
	config.AuthorDocuments = &fixedAuthorDocumentReader{publicationErr: errors.New("publication workspace was not finalized")}
	_, _, _, results, err := runReaderPatchStage(
		context.Background(), config, catalog, testEditorialMemory(catalog),
		reportilcontract.EditorialMemoryReceipt{ArtifactID: "art_memory", SHA256: strings.Repeat("d", 64)}, document, nil,
		reportilcontract.AuthorWorkspaceReceipt{ArtifactID: "art_author", SHA256: strings.Repeat("a", 64)},
	)
	var providerFailure *providerStageError
	if err == nil || len(results) != 1 || len(provider.requests) != 1 ||
		!errors.As(err, &providerFailure) ||
		providerFailure.reason != reportexecution.ProviderFailureReasonSemanticValidation ||
		providerValidationCode(err) != reportexecution.ProviderValidationCodeDocumentContract {
		t.Fatalf("publication workspace result err=%v", err)
	}
}

func TestRunReaderPatchStageRepairsReaderQualityFromFinalizedCandidate(t *testing.T) {
	document, catalog, authored := canonicalReaderTestDocument(t, "mis_reader_quality_repair")
	broken := authored
	broken.Sections = append([]reportilcontract.AuthorSection(nil), authored.Sections...)
	broken.Sections[0].Blocks = append([]reportilcontract.AuthorBlock(nil), authored.Sections[0].Blocks...)
	broken.Sections[0].Blocks[0].Prose = "이 보고서에서는 사실을 설명한다."
	repaired := broken
	repaired.Sections = append([]reportilcontract.AuthorSection(nil), broken.Sections...)
	repaired.Sections[0].Blocks = append([]reportilcontract.AuthorBlock(nil), broken.Sections[0].Blocks...)
	repaired.Sections[0].Blocks[0].Prose = authored.Sections[0].Blocks[0].Prose
	provider := &recordingProvider{outputs: []string{"candidate finalized", "repair finalized"}}
	reader := &fixedAuthorDocumentReader{
		publications:                 []reportilcontract.AuthorDocument{broken, repaired},
		publicationReplacementCounts: []int{2, 1},
	}
	config := testSourceProductConfig(provider, &acceptingSourceReadVerifier{})
	config.MissionID = catalog.MissionID
	config.MissionObjective = "Explain the answer."
	config.AuthorDocuments = reader
	edited, receipt, workspace, results, err := runReaderPatchStage(
		context.Background(), config, catalog, testEditorialMemory(catalog),
		reportilcontract.EditorialMemoryReceipt{ArtifactID: "art_memory", SHA256: strings.Repeat("d", 64)}, document, nil,
		reportilcontract.AuthorWorkspaceReceipt{ArtifactID: "art_author", SHA256: strings.Repeat("a", 64)},
	)
	if err != nil || len(results) != 2 || len(provider.requests) != 2 || reader.publicationCalls != 2 {
		t.Fatalf("publication repair: calls=%d reads=%d results=%d err=%#v code=%q", len(provider.requests), reader.publicationCalls, len(results), err, providerValidationCode(err))
	}
	if provider.requests[1].ReportILSources.Attempt != 2 ||
		provider.requests[1].ReportILSources.BaseAuthorArtifactID != "art_publication_1" ||
		provider.requests[1].ReportILSources.BaseAuthorSHA256 != strings.Repeat("c", 64) ||
		!strings.Contains(provider.requests[1].Prompt, "BOUNDED REPAIR ATTEMPT") ||
		!strings.Contains(provider.requests[1].Prompt, string(reportexecution.ProviderValidationCodeReaderProcess)) {
		t.Fatalf("publication repair request = %#v", provider.requests[1])
	}
	if !equalJSONValue(edited, document) || receipt.PublicationPatches != 3 || workspace.Replacements != 1 || workspace.ArtifactID != "art_publication_2" {
		t.Fatalf("publication repair result = %#v / %#v / %#v", edited, receipt, workspace)
	}
}

func TestRunReaderPatchStageDoesNotRepairDocumentContractFailure(t *testing.T) {
	document, catalog, _ := canonicalReaderTestDocument(t, "mis_reader_contract_terminal")
	provider := &recordingProvider{outputs: []string{"workspace finalized", "must not run"}}
	config := testSourceProductConfig(provider, &acceptingSourceReadVerifier{})
	config.MissionID = catalog.MissionID
	config.MissionObjective = "Explain the answer."
	config.AuthorDocuments = &fixedAuthorDocumentReader{publicationErr: errors.New("publication workspace was not finalized")}
	_, _, _, results, err := runReaderPatchStage(
		context.Background(), config, catalog, testEditorialMemory(catalog),
		reportilcontract.EditorialMemoryReceipt{ArtifactID: "art_memory", SHA256: strings.Repeat("d", 64)}, document, nil,
		reportilcontract.AuthorWorkspaceReceipt{ArtifactID: "art_author", SHA256: strings.Repeat("a", 64)},
	)
	if err == nil || len(results) != 1 || len(provider.requests) != 1 || providerValidationCode(err) != reportexecution.ProviderValidationCodeDocumentContract {
		t.Fatalf("document contract repair boundary: calls=%d results=%d err=%#v code=%q", len(provider.requests), len(results), err, providerValidationCode(err))
	}
}

func TestRunReaderPatchStageAcceptsUnchangedFinalizedWorkspace(t *testing.T) {
	document, catalog, authored := canonicalReaderTestDocument(t, "mis_reader_workspace_unchanged")
	provider := &recordingProvider{outputs: []string{"workspace finalized"}}
	config := testSourceProductConfig(provider, &acceptingSourceReadVerifier{})
	config.MissionID = catalog.MissionID
	config.MissionObjective = "Explain the answer."
	config.AuthorDocuments = &fixedAuthorDocumentReader{publication: authored}
	edited, receipt, _, results, err := runReaderPatchStage(
		context.Background(), config, catalog, testEditorialMemory(catalog),
		reportilcontract.EditorialMemoryReceipt{ArtifactID: "art_memory", SHA256: strings.Repeat("d", 64)}, document, nil,
		reportilcontract.AuthorWorkspaceReceipt{ArtifactID: "art_author", SHA256: strings.Repeat("a", 64)},
	)
	if err != nil || len(results) != 1 || len(provider.requests) != 1 {
		t.Fatalf("publication reader: calls=%d results=%d err=%#v cause=%v code=%q", len(provider.requests), len(results), err, errors.Unwrap(err), providerValidationCode(err))
	}
	if !equalJSONValue(edited, document) || receipt.ProposedPatches != 0 || receipt.Applied {
		t.Fatalf("unchanged publication result = %#v / %#v", edited, receipt)
	}
}

func TestRunReaderPatchStagePreservesDistinctTakedaAccountAndAddsItsSource(t *testing.T) {
	catalog := testSourceCatalogWithCount(t, "mis_reader_takeda_account", 2)
	originalAuthor := testAuthorDocument()
	originalAuthor.Sections[0].Blocks[0].Prose = "아사고시는 1431년 착수와 1443년 완공을 야마나 모치토요의 명령으로 전한다."
	publication := originalAuthor
	publication.Sections = append([]reportilcontract.AuthorSection(nil), originalAuthor.Sections...)
	publication.Sections[0].Blocks = append([]reportilcontract.AuthorBlock(nil), originalAuthor.Sections[0].Blocks...)
	publication.Sections[0].Blocks[0].Prose = "아사고시는 1431년 착수와 1443년 완공을 야마나 모치토요의 명령으로 전한다. 한편 효고현립역사박물관은 1433년 다지마국 수호 야마나 소젠이 축성을 명해 13년에 걸쳐 세웠다고 설명한다. 두 연대기는 하나로 합칠 수 없는 별개의 전승이다."
	publication.Sections[0].Blocks[0].EditorialAccountKeys = []string{"account_001", "account_002"}
	publication.Sections[0].Blocks[0].EvidenceSourceKeys = []string{"source_001", "source_002"}
	_, document, err := compileReportFirstAuthorDraft(
		reportFirstDraftFromAuthorDocument(originalAuthor), "narrative_takeda", "doc_takeda",
		"다케다성의 역사적 역할", catalog, testSourceReadReceipt(catalog), nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	provider := &recordingProvider{outputs: []string{"workspace finalized"}}
	config := testSourceProductConfig(provider, &acceptingSourceReadVerifier{})
	config.MissionID = catalog.MissionID
	config.MissionObjective = "다케다성의 역사적 역할"
	config.AuthorDocuments = &fixedAuthorDocumentReader{
		publication: publication, publicationReplacements: 1,
	}
	edited, receipt, _, _, err := runReaderPatchStage(
		context.Background(), config, catalog, testEditorialMemory(catalog),
		reportilcontract.EditorialMemoryReceipt{ArtifactID: "art_memory", SHA256: strings.Repeat("d", 64)}, document, nil,
		reportilcontract.AuthorWorkspaceReceipt{ArtifactID: "art_author", SHA256: strings.Repeat("a", 64)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Applied || receipt.AcceptedPatches != 1 || receipt.ProposedPatches != 1 {
		t.Fatalf("reader receipt = %#v", receipt)
	}
	originalBlock, editedBlock := document.Blocks[1], edited.Blocks[1]
	if originalBlock.NodeID != editedBlock.NodeID || originalBlock.Kind != editedBlock.Kind ||
		!strings.Contains(editedBlock.Prose, "1433년") || !strings.Contains(editedBlock.Prose, "야마나 소젠") ||
		!strings.Contains(editedBlock.Prose, "13년에 걸쳐") || len(editedBlock.EvidenceRefs) != 2 {
		t.Fatalf("edited chronology block = %#v", editedBlock)
	}
	if len(edited.References) != 2 || edited.References[1].Target != "accepted-source:002" {
		t.Fatalf("edited references = %#v", edited.References)
	}
}

func TestValidatePublicationDocumentStructureStillRejectsNonEvidenceChanges(t *testing.T) {
	document, _, _ := canonicalReaderTestDocument(t, "mis_reader_structure")
	edited := document
	edited.References = append([]Reference(nil), document.References...)
	edited.Blocks = append([]Block(nil), document.Blocks...)
	edited.Blocks[1].EvidenceRefs = append([]string(nil), document.Blocks[1].EvidenceRefs...)
	edited.References = append(edited.References, Reference{
		RefID: "ref.accepted_source.extra", Kind: "footnote", Target: "accepted-source:002", VisibleLabel: "Accepted source 2",
	})
	edited.Blocks[1].EvidenceRefs = append(edited.Blocks[1].EvidenceRefs, "ref.accepted_source.extra")
	if err := validatePublicationDocumentStructure(document, edited); err != nil {
		t.Fatalf("evidence-only change rejected: %v", err)
	}
	edited.Blocks[1].ParentNodeID = "section.other"
	if err := validatePublicationDocumentStructure(document, edited); err == nil {
		t.Fatal("publication topology change was accepted")
	}
}

func TestReaderFinalizationReceiptRejectsUnknownOrInconsistentValues(t *testing.T) {
	valid := ReaderFinalizationReceipt{
		ContinuityPatches: 1,
		ProposedPatches:   2,
		AcceptedPatches:   1,
		RejectedPatches:   1,
		Applied:           true,
		Rejections: []ReaderPatchRejectionReceipt{{
			Reason: ReaderPatchRejectionLocality,
			Count:  1,
		}},
	}
	if err := validateReaderFinalizationReceipt(valid); err != nil {
		t.Fatalf("valid reader finalization receipt rejected: %v", err)
	}
	cases := []ReaderFinalizationReceipt{
		{ProposedPatches: 1, RejectedPatches: 1, Rejections: []ReaderPatchRejectionReceipt{{Reason: "private detail", Count: 1}}},
		{ContinuityPatches: 1, ProposedPatches: 1, AcceptedPatches: 1, Applied: false},
		{ContinuityPatches: 1, PublicationPatches: 1, ProposedPatches: 1, AcceptedPatches: 1, Applied: true},
		{ProposedPatches: 2, RejectedPatches: 1, Rejections: []ReaderPatchRejectionReceipt{{Reason: ReaderPatchRejectionLocality, Count: 1}}},
	}
	for index, receipt := range cases {
		if err := validateReaderFinalizationReceipt(receipt); err == nil {
			t.Fatalf("invalid reader finalization receipt %d was accepted: %#v", index, receipt)
		}
	}
}

func TestRunContinuityPatchStageMakesExactlyOneSourceAwareWorkspaceCall(t *testing.T) {
	document, catalog, authored := canonicalReaderTestDocument(t, "mis_continuity_patch_stage")
	provider := &recordingProvider{outputs: []string{"workspace finalized"}}
	config := testSourceProductConfig(provider, &acceptingSourceReadVerifier{})
	config.MissionID = catalog.MissionID
	config.MissionObjective = "Explain the answer."
	config.AuthorDocuments = &fixedAuthorDocumentReader{continuity: authored, continuityReplacements: 1}
	edited, receipt, workspace, results, err := runContinuityPatchStage(
		context.Background(), config, catalog, testEditorialMemory(catalog),
		reportilcontract.EditorialMemoryReceipt{ArtifactID: "art_memory", SHA256: strings.Repeat("d", 64)}, document, nil,
		reportilcontract.AuthorWorkspaceReceipt{ArtifactID: "art_author", SHA256: strings.Repeat("a", 64)},
	)
	if err != nil || len(results) != 1 || len(provider.requests) != 1 {
		t.Fatalf("continuity workspace boundary: calls=%d results=%d err=%#v cause=%v code=%q", len(provider.requests), len(results), err, errors.Unwrap(err), providerValidationCode(err))
	}
	if edited.Title != document.Title || receipt.ContinuityPatches != 1 || receipt.PublicationPatches != 0 || !receipt.Applied || workspace.Stage != "il_continuity" {
		t.Fatalf("continuity workspace output = %#v / %#v / %#v", edited, receipt, workspace)
	}
	request := provider.requests[0]
	if request.UserText != "report IL il_continuity" || request.ReportILSources == nil || request.ReportILSources.Stage != "il_continuity" ||
		request.ReportILSources.BaseAuthorArtifactID != "art_author" || request.ReportILSources.BaseAuthorSHA256 != strings.Repeat("a", 64) ||
		!reflect.DeepEqual(request.ExtraMCPTools, []string{
			reportilcontract.EditorialMemoryReadTool,
			reportilcontract.AuthorDocumentOpenTool, reportilcontract.AuthorDocumentReadTool,
			reportilcontract.AuthorDocumentReviseBlockTool, reportilcontract.AuthorDocumentFinalizeTool,
		}) || len(request.OutputJSONSchema) != 0 {
		t.Fatalf("continuity source isolation request = %#v", request)
	}
	for _, required := range []string{
		"Fact-check the manuscript sentence by sentence",
		"replace the shortest exact once-only substring inside that sentence",
		"Preserve the rest of the sentence and block whenever possible",
		"Do not rewrite a whole paragraph or section",
		"recheck each changed sentence against its exact anchor",
	} {
		if !strings.Contains(request.Prompt, required) {
			t.Fatalf("continuity prompt lacks %q: %s", required, request.Prompt)
		}
	}
	if strings.Contains(request.Prompt, "Generate only strict JSON") {
		t.Fatalf("MCP continuity prompt retained terminal-output instruction: %s", request.Prompt)
	}
}

func TestValidatePublicationStructuredPayloadsRejectsEquationMutation(t *testing.T) {
	original := Document{Blocks: []Block{
		{NodeID: "prose", Kind: "prose", Prose: "Reader-facing explanation."},
		{NodeID: "equation", Kind: "equation", Equation: &Equation{Expression: `C_{final} = O`, Notation: "latex"}},
		{NodeID: "code", Kind: "code", Code: "print('x')", Language: "python"},
		{NodeID: "table", Kind: "table", Table: &Table{Columns: []string{"A", "B"}, Rows: [][]string{{"x", "y"}}}},
		{NodeID: "list", Kind: "list", Items: []string{"one", "two"}},
	}}
	for name, mutate := range map[string]func(*Document){
		"equation": func(document *Document) { document.Blocks[1].Equation.Expression = `C_{task} \\ne O` },
		"code":     func(document *Document) { document.Blocks[2].Code = "print('changed')" },
		"table":    func(document *Document) { document.Blocks[3].Table.Rows[0][0] = "changed" },
		"list":     func(document *Document) { document.Blocks[4].Items[0] = "changed" },
	} {
		t.Run(name, func(t *testing.T) {
			edited := original
			edited.Blocks = append([]Block(nil), original.Blocks...)
			for index := range edited.Blocks {
				if original.Blocks[index].Equation != nil {
					value := *original.Blocks[index].Equation
					edited.Blocks[index].Equation = &value
				}
				if original.Blocks[index].Table != nil {
					table := *original.Blocks[index].Table
					table.Columns = append([]string(nil), table.Columns...)
					table.Rows = make([][]string, len(original.Blocks[index].Table.Rows))
					for rowIndex := range table.Rows {
						table.Rows[rowIndex] = append([]string(nil), original.Blocks[index].Table.Rows[rowIndex]...)
					}
					edited.Blocks[index].Table = &table
				}
				edited.Blocks[index].Items = append([]string(nil), original.Blocks[index].Items...)
			}
			mutate(&edited)
			if err := validatePublicationStructuredPayloads(original, edited); err == nil {
				t.Fatalf("publication %s mutation was accepted", name)
			}
		})
	}
}

func TestRunReaderPatchStageMakesExactlyOneMCPWorkspaceCall(t *testing.T) {
	document, catalog, authored := canonicalReaderTestDocument(t, "mis_reader_patch_stage")
	provider := &recordingProvider{outputs: []string{"workspace finalized"}}
	config := testSourceProductConfig(provider, &acceptingSourceReadVerifier{})
	config.MissionID = catalog.MissionID
	config.MissionObjective = "Explain the answer."
	config.AuthorDocuments = &fixedAuthorDocumentReader{publication: authored}
	_, _, _, results, err := runReaderPatchStage(
		context.Background(), config, catalog, testEditorialMemory(catalog),
		reportilcontract.EditorialMemoryReceipt{ArtifactID: "art_memory", SHA256: strings.Repeat("d", 64)}, document, nil,
		reportilcontract.AuthorWorkspaceReceipt{ArtifactID: "art_author", SHA256: strings.Repeat("a", 64)},
	)
	if err != nil || len(results) != 1 || len(provider.requests) != 1 {
		t.Fatalf("reader workspace boundary: calls=%d results=%d err=%#v cause=%v code=%q", len(provider.requests), len(results), err, errors.Unwrap(err), providerValidationCode(err))
	}
	request := provider.requests[0]
	if request.UserText != "report IL il_reader" || request.ReportILSources == nil || request.ReportILSources.Stage != "il_reader" ||
		request.ReportILSources.BaseAuthorArtifactID != "art_author" || request.ReportILSources.BaseAuthorSHA256 != strings.Repeat("a", 64) ||
		!reflect.DeepEqual(request.ExtraMCPTools, []string{
			reportilcontract.EditorialMemoryReadTool,
			reportilcontract.AuthorDocumentOpenTool, reportilcontract.AuthorDocumentReadTool,
			reportilcontract.AuthorDocumentEditTextTool, reportilcontract.AuthorDocumentFinalizeTool,
		}) || len(request.OutputJSONSchema) != 0 {
		t.Fatalf("reader source isolation request = %#v", request)
	}
	for _, required := range []string{
		"Use plasma.report_il.document.edit_text",
		"Use the exact target_kind and target_key shown by document.read",
		"Remove construction narration",
		"Code, equations, tables, lists, source bindings, and block structure are immutable",
	} {
		if !strings.Contains(request.Prompt, required) {
			t.Fatalf("publication prompt lacks %q: %s", required, request.Prompt)
		}
	}
	if strings.Contains(request.Prompt, "Generate only strict JSON") {
		t.Fatalf("MCP publication prompt retained terminal-output instruction: %s", request.Prompt)
	}
}

func TestPublicationDamageIsRepairedByFinalContinuityStage(t *testing.T) {
	document, catalog, authored := canonicalReaderTestDocument(t, "mis_final_continuity_repair")
	damaged := authored
	damaged.Sections = append([]reportilcontract.AuthorSection(nil), authored.Sections...)
	damaged.Sections[0].Blocks = append([]reportilcontract.AuthorBlock(nil), authored.Sections[0].Blocks...)
	damaged.Sections[0].Blocks[0].Prose = "The Tajima governor Yamana Sōzen built the castle over thirteen years."
	corrected := damaged
	corrected.Sections = append([]reportilcontract.AuthorSection(nil), damaged.Sections...)
	corrected.Sections[0].Blocks = append([]reportilcontract.AuthorBlock(nil), damaged.Sections[0].Blocks...)
	corrected.Sections[0].Blocks[0].Prose = "The Tajima governor Yamana Sōzen had the castle built over thirteen years."

	provider := &recordingProvider{outputs: []string{"publication finalized", "continuity finalized"}}
	reader := &fixedAuthorDocumentReader{
		publication:             damaged,
		continuity:              corrected,
		publicationReplacements: 1,
		continuityReplacements:  1,
	}
	config := testSourceProductConfig(provider, &acceptingSourceReadVerifier{})
	config.MissionID = catalog.MissionID
	config.MissionObjective = "Explain the castle chronology."
	config.AuthorDocuments = reader
	memory := testEditorialMemory(catalog)
	memory.Accounts[0].Account = "The Tajima governor Yamana Sōzen commissioned the castle and had it built over thirteen years."
	memoryReceipt := reportilcontract.EditorialMemoryReceipt{ArtifactID: "art_memory", SHA256: strings.Repeat("d", 64)}

	published, publicationReceipt, publicationWorkspace, publicationResults, err := runReaderPatchStage(
		context.Background(), config, catalog, memory, memoryReceipt, document, nil,
		reportilcontract.AuthorWorkspaceReceipt{ArtifactID: "art_author", SHA256: strings.Repeat("a", 64)},
	)
	if err != nil || len(publicationResults) != 1 || publicationWorkspace.Stage != "il_reader" ||
		publicationReceipt.PublicationPatches != 1 {
		t.Fatalf("publication stage = %#v / %#v / %#v / %v", published, publicationReceipt, publicationWorkspace, err)
	}
	if got := published.Blocks[1].Prose; got != damaged.Sections[0].Blocks[0].Prose {
		t.Fatalf("publication damage was not represented: %q", got)
	}

	finalDocument, continuityReceipt, _, continuityResults, err := runContinuityPatchStage(
		context.Background(), config, catalog, memory, memoryReceipt, published, nil, publicationWorkspace,
	)
	if err != nil || len(continuityResults) != 1 || continuityReceipt.ContinuityPatches != 1 {
		t.Fatalf("continuity stage = %#v / %#v / %v", finalDocument, continuityReceipt, err)
	}
	if got := finalDocument.Blocks[1].Prose; got != corrected.Sections[0].Blocks[0].Prose {
		t.Fatalf("final continuity did not restore causative meaning: %q", got)
	}
	if len(provider.requests) != 2 || provider.requests[1].ReportILSources.BaseAuthorArtifactID != "art_publication_1" ||
		provider.requests[1].ReportILSources.BaseAuthorSHA256 != strings.Repeat("c", 64) {
		t.Fatalf("continuity did not open publication artifact: %#v", provider.requests)
	}
}

func TestBuildSourcePacketRejectsUnsupportedAssetsAndIsDeterministic(t *testing.T) {
	content := []byte("bounded source")
	reader := productSourceReader{sources: []reportilcontract.SourceSnapshot{{SnapshotID: "src_1", MissionID: "mis_1", Active: true, ArtifactIDs: []string{"art_1"}, ContentHash: sha256Hex(content)}}, artifacts: map[string]reportilcontract.Artifact{"art_1": {ArtifactID: "art_1", MissionID: "mis_1", MediaType: "text/plain", SHA256: sha256Hex(content), ByteSize: int64(len(content)), Content: content}}}
	first, err := BuildSourcePacket(context.Background(), reader, "mis_1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildSourcePacket(context.Background(), reader, "mis_1")
	if err != nil {
		t.Fatal(err)
	}
	if first.SHA256 != second.SHA256 {
		t.Fatalf("packet hash changed")
	}
	reader.artifacts["art_1"] = reportilcontract.Artifact{ArtifactID: "art_1", MissionID: "mis_1", MediaType: "image/png", SHA256: strings.Repeat("a", 64), Content: []byte("png")}
	if _, err := BuildSourcePacket(context.Background(), reader, "mis_1"); err == nil {
		t.Fatal("unsupported asset accepted")
	}
}

func TestBuildSourcePacketRejectsBinaryLiveSource(t *testing.T) {
	reader := productSourceReader{sources: []reportilcontract.SourceSnapshot{{SnapshotID: "src_1", MissionID: "mis_1", Active: true, RetrievalPolicy: "live_reference"}}, reads: map[string]reportilcontract.LocalRead{"src_1": {Binary: true, SHA256: strings.Repeat("b", 64)}}}
	if _, err := BuildSourcePacket(context.Background(), reader, "mis_1"); err == nil || !strings.Contains(err.Error(), "binary") {
		t.Fatalf("binary source error = %v", err)
	}
}

func TestBuildSourcePacketCanonicalizesSnapshotsAndPreservesStoredArtifactOrder(t *testing.T) {
	first := []byte("first artifact")
	second := []byte("second artifact")
	firstHash, secondHash := sha256Hex(first), sha256Hex(second)
	firstSnapshotHash := snapshotHashValue([]SourceArtifact{{ArtifactID: "art_b", SHA256: secondHash}, {ArtifactID: "art_a", SHA256: firstHash}})
	secondSnapshotHash := sha256Hex([]byte("single snapshot"))
	reader := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{
			{SnapshotID: "src_z", MissionID: "mis_packet", Active: true, ArtifactIDs: []string{"art_z"}, ContentHash: secondSnapshotHash},
			{SnapshotID: "src_a", MissionID: "mis_packet", Active: true, ArtifactIDs: []string{"art_b", "art_a"}, ContentHash: firstSnapshotHash},
		},
		artifacts: map[string]reportilcontract.Artifact{
			"art_a": {ArtifactID: "art_a", MissionID: "mis_packet", MediaType: "text/plain", SHA256: firstHash, ByteSize: int64(len(first)), Content: first},
			"art_b": {ArtifactID: "art_b", MissionID: "mis_packet", MediaType: "text/plain", SHA256: secondHash, ByteSize: int64(len(second)), Content: second},
			"art_z": {ArtifactID: "art_z", MissionID: "mis_packet", MediaType: "text/plain", SHA256: secondSnapshotHash, ByteSize: int64(len("single snapshot")), Content: []byte("single snapshot")},
		},
	}
	packet, err := BuildSourcePacket(context.Background(), reader, " mis_packet ")
	if err != nil {
		t.Fatal(err)
	}
	if len(packet.Sources) != 2 || packet.Sources[0].SnapshotReceipt != snapshotReceipt(reader.sources[1]) {
		t.Fatalf("snapshot ordering = %#v", packet.Sources)
	}
	if got := []string{packet.Sources[0].Artifacts[0].ArtifactID, packet.Sources[0].Artifacts[1].ArtifactID}; !reflect.DeepEqual(got, []string{"art_b", "art_a"}) {
		t.Fatalf("artifact order = %#v", got)
	}
	if packet.Sources[0].Artifacts[0].SHA256 != secondHash || packet.Sources[0].Artifacts[1].ByteSize != int64(len(first)) || packet.Sources[0].Artifacts[0].MediaType != "text/plain" {
		t.Fatalf("artifact metadata was not retained: %#v", packet.Sources[0].Artifacts)
	}
	preimage := struct {
		SchemaVersion string         `json:"schema_version"`
		MissionID     string         `json:"mission_id"`
		Sources       []PacketSource `json:"sources"`
	}{packet.SchemaVersion, packet.MissionID, packet.Sources}
	raw, err := json.Marshal(preimage)
	if err != nil {
		t.Fatal(err)
	}
	if packet.SHA256 != sha256Hex(raw) {
		t.Fatalf("packet hash = %q, want %q", packet.SHA256, sha256Hex(raw))
	}
	changed := reader
	changed.sources = append([]reportilcontract.SourceSnapshot(nil), reader.sources...)
	changed.sources[1].ArtifactIDs = []string{"art_a", "art_b"}
	changed.sources[1].ContentHash = snapshotHashValue([]SourceArtifact{{ArtifactID: "art_a", SHA256: firstHash}, {ArtifactID: "art_b", SHA256: secondHash}})
	changedPacket, err := BuildSourcePacket(context.Background(), changed, "mis_packet")
	if err != nil {
		t.Fatal(err)
	}
	if changedPacket.SHA256 == packet.SHA256 {
		t.Fatal("changing persisted artifact ordinal did not change packet hash")
	}
}

func TestBuildSourcePacketValidatesMissionBeforeReading(t *testing.T) {
	reader := productSourceReader{sources: []reportilcontract.SourceSnapshot{{SnapshotID: "src_bad", MissionID: "mis_other", Active: true, ArtifactIDs: []string{"art_bad"}}}, artifacts: map[string]reportilcontract.Artifact{"art_bad": {Content: []byte("should not read")}}}
	if _, err := BuildSourcePacket(context.Background(), reader, "mis_packet"); err == nil || strings.Contains(err.Error(), "mis_other") {
		t.Fatalf("cross-mission result = %v", err)
	}
	if _, err := BuildSourcePacket(context.Background(), reader, "   "); err == nil {
		t.Fatal("empty mission accepted")
	}
}

func TestBuildSourcePacketLiveObservationCanonicalization(t *testing.T) {
	reader := productSourceReader{sources: []reportilcontract.SourceSnapshot{{SnapshotID: "src_live", MissionID: "mis_live", Active: true, RetrievalPolicy: "live_reference"}}, reads: map[string]reportilcontract.LocalRead{"src_live": {Content: "extracted PDF text", Size: 18, Extraction: "pdf_text", ObservationReceipt: "evt_observed"}}}
	packet, err := BuildSourcePacket(context.Background(), reader, "mis_live")
	if err != nil {
		t.Fatal(err)
	}
	observation := packet.Sources[0].Observation
	if observation["size"] != "18" || observation["extraction"] != "pdf_text" || observation["sha256"] != "" {
		t.Fatalf("live observation = %#v", observation)
	}
	if _, ok := observation["sha256"]; ok || packet.Sources[0].ObservationReceipt != "evt_observed" {
		t.Fatalf("live observation privacy/receipt = %#v/%q", observation, packet.Sources[0].ObservationReceipt)
	}
	encoded, _ := json.Marshal(packet)
	if strings.Contains(string(encoded), "path") || strings.Contains(string(encoded), "locator") {
		t.Fatalf("packet contains path/locator fields: %s", encoded)
	}
}

func TestValidateProviderContentCredentialAndRestrictedURLCases(t *testing.T) {
	restricted := []string{
		"Authorization: secret-value", "Bearer secret-value", "api_key=secret-value", "api-key: secret-value", "access_token=secret-value", "refresh-token: secret-value", "client_secret=secret-value", "Cookie: secret-value", "session_token=secret-value",
		"file:///private/secret.txt", "FILE:///private/secret.txt", "https://user:password@example.com/a", "https://node.ts.net/a", "http://localhost:8080/a", "http://service.localhost/a", "http://127.0.0.1:8080/a", "http://[::1]/a", "http://10.0.0.1/a", "http://172.16.0.1/a", "http://192.168.1.2/a", "http://169.254.1.2/a", "http://100.64.0.1/a", "http://[::ffff:127.0.0.1]/a",
	}
	for _, content := range restricted {
		if err := validateProviderContent(content); err == nil || strings.Contains(err.Error(), content) {
			t.Fatalf("restricted content result for %q: %v", content, err)
		}
	}
	for _, content := range []string{"The authorization process is documented.", "Bearer nouns are common.", "The API key concept is discussed.", "A cookie recipe is harmless.", "Password policies are discussed."} {
		if err := validateProviderContent(content); err != nil {
			t.Fatalf("benign content rejected %q: %v", content, err)
		}
	}
}

func TestSourceIsolatedRequestContract(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_fixture")
	config := testSourceProductConfig(&recordingProvider{}, &acceptingSourceReadVerifier{})
	req := sourceIsolatedRequest(config, catalog, "il_narrative", 1, "prompt", []byte(`{"type":"object"}`), 0)
	if req.Model != "gpt-5.6-luna" || req.ReasoningEffort != "xhigh" || req.AgentExecutor != "codex" || req.MCPMode != "source_read_only" || req.CapabilityProfile != agentcapability.ProfileReportILSourceV1 || req.ProfileRevision != agentcapability.RevisionV1 || req.DisableTools || !req.IgnoreUserConfig || !req.EphemeralSession || !req.ReplaceMCPTools || !reflect.DeepEqual(req.ExtraMCPTools, []string{reportilcontract.SourceListTool, reportilcontract.SourceReadTool, reportilcontract.SourceQuoteRegisterTool}) {
		t.Fatalf("request contract = %#v", req)
	}
	if req.MissionID != catalog.MissionID || req.ToolSessionID != "ses_1" || req.ReportILSources == nil {
		t.Fatalf("request source binding = %#v", req)
	}
	binding := req.ReportILSources
	if binding.PendingEventID != "evt_pending" || binding.Stage != "il_narrative" || binding.Attempt != 1 || binding.Catalog.SHA256 != catalog.SHA256 || binding.MaxCallBytes != 0 || binding.MaxReadBytes != 0 {
		t.Fatalf("source binding = %#v", binding)
	}

	selection := sourceIsolatedRequest(config, catalog, "il_source_selection", 1, "prompt", []byte(`{"type":"object"}`), 1024)
	if selection.ReportILSources == nil || selection.ReportILSources.MaxReadBytes != reportilcontract.DefaultSourceReadMaxBytes || selection.ReportILSources.MaxSourceReadBytes != 1024 {
		t.Fatalf("selection source binding = %#v", selection.ReportILSources)
	}
}

func TestDocumentCompatibilityUsesCompleteSingleSourceContract(t *testing.T) {
	contract := completeSingleSourceToolContract()
	for _, required := range []string{
		"Read every byte of every frozen accepted source exactly once",
		"source_key",
		"offset 0",
		"exact returned next_offset",
		fmt.Sprintf("up to %d max_bytes", reportilcontract.DefaultSourceReadMaxBytes),
	} {
		if !strings.Contains(contract, required) {
			t.Fatalf("document source contract lacks %q: %s", required, contract)
		}
	}
	for _, forbidden := range []string{"remaining_sources", "catalog-ordered batches", "read with {}"} {
		if strings.Contains(contract, forbidden) {
			t.Fatalf("document source contract contains batch instruction %q: %s", forbidden, contract)
		}
	}
}

func TestStrictDecodeRejectsTrailingJSON(t *testing.T) {
	if _, err := strictDecode[map[string]any]([]byte(`{} {}`)); err == nil {
		t.Fatal("strict decode accepted trailing JSON")
	}
}

func TestPreserveDocumentStructureAllowsOnlyProse(t *testing.T) {
	original := Document{SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily, DocumentID: "doc_1", RevisionID: "rev_1", NarrativeContractID: "narr_1", Title: "Title", Language: "en", Blocks: []Block{{NodeID: "n_1", Kind: "prose", Prose: "old"}}}
	edited := original
	edited.RevisionID = "rev_2"
	edited.Blocks = []Block{{NodeID: "n_1", Kind: "prose", Prose: "new"}}
	if err := preserveDocumentStructure(original, edited); err != nil {
		t.Fatal(err)
	}
	edited.Blocks[0].Kind = "section"
	if err := preserveDocumentStructure(original, edited); err == nil {
		t.Fatal("structure edit accepted")
	}
}

func TestProviderStructuredOutputSchemasAreClosedAndSupported(t *testing.T) {
	narrative := Narrative{ContractID: "narrative_1", DocumentID: "document_1", SectionRoles: []SectionRole{{SectionID: "section.a"}}}
	document := Document{Blocks: []Block{{NodeID: "section.a", Kind: "section", Title: "A", Level: 2}, {NodeID: "node.a", Kind: "prose", ParentNodeID: "section.a", Prose: "Body"}}}
	catalog := testSourceCatalog(t, "mis_fixture")
	_, _, frozenAuthor := evidencePacketFixture(t)
	schemas := map[string][]byte{
		providerSchemaSourceSelection:      sourceSelectionSchemaBytes(catalog),
		providerSchemaNarrative:            providerNarrativeSchemaBytes(narrative.ContractID, narrative.DocumentID),
		providerSchemaDocument:             providerDocumentSchemaBytes(narrative, catalog),
		providerSchemaAuthorEvidenceRepair: providerAuthorEvidenceRepairSchemaBytes(frozenAuthor),
		providerSchemaFlow:                 providerFlowSchemaBytes(document),
	}
	for stage, raw := range schemas {
		t.Run(stage, func(t *testing.T) {
			if !json.Valid(raw) {
				t.Fatalf("provider schema is invalid JSON: %s", raw)
			}
			if err := lintProviderSchema(raw); err != nil {
				t.Fatal(err)
			}
			var root map[string]any
			if err := json.Unmarshal(raw, &root); err != nil {
				t.Fatal(err)
			}
			if _, ok := root["$defs"].(map[string]any); !ok {
				t.Fatalf("provider schema lacks root $defs: %#v", root)
			}
		})
	}
}

func TestProviderSchemaTransformPreservesPropertyNamedTitle(t *testing.T) {
	source := map[string]any{
		"type":  "object",
		"title": "schema metadata",
		"properties": map[string]any{
			"title": map[string]any{"type": "string", "title": "field metadata"},
		},
		"required": []any{"title"},
	}
	transformed := transformSchema(source).(map[string]any)
	if _, ok := transformed["title"]; ok {
		t.Fatalf("schema metadata title survived: %#v", transformed)
	}
	properties := transformed["properties"].(map[string]any)
	field, ok := properties["title"].(map[string]any)
	if !ok || field["type"] != "string" {
		t.Fatalf("property named title was removed: %#v", transformed)
	}
	if _, ok := field["title"]; ok {
		t.Fatalf("nested schema metadata title survived: %#v", field)
	}
}

func TestProviderDocumentSchemaRequiresExactNarrativeSectionsAndAuthoredLeaves(t *testing.T) {
	narrative := Narrative{SectionRoles: []SectionRole{{SectionID: "section.evidence"}, {SectionID: "section.synthesis"}}}
	raw := providerDocumentSchemaBytes(narrative, testSourceCatalog(t, "mis_fixture"))
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	draft := root["$defs"].(map[string]any)["document_draft"].(map[string]any)
	properties := draft["properties"].(map[string]any)
	sections := properties["sections"].(map[string]any)
	sectionProperties := sections["properties"].(map[string]any)
	if sections["additionalProperties"] != false || len(sectionProperties) != 2 {
		t.Fatalf("provider section inventory is open: %#v", sections)
	}
	for _, sectionID := range []string{"section.evidence", "section.synthesis"} {
		section := sectionProperties[sectionID].(map[string]any)
		if section["$ref"] != "#/$defs/document_section_draft" || len(section) != 1 {
			t.Fatalf("provider section %s does not share the closed section schema: %#v", sectionID, section)
		}
	}
	defs := root["$defs"].(map[string]any)
	sectionFields := defs["document_section_draft"].(map[string]any)["properties"].(map[string]any)
	if _, ok := sectionFields["title"]; !ok {
		t.Fatal("shared provider section omitted title")
	}
	if sectionFields["blocks"].(map[string]any)["items"].(map[string]any)["$ref"] != "#/$defs/document_block_draft" {
		t.Fatalf("provider blocks do not share the leaf schema: %#v", sectionFields["blocks"])
	}
	variants := defs["document_block_draft"].(map[string]any)["anyOf"].([]any)
	if len(variants) != 6 {
		t.Fatalf("authored leaf variants = %d, want 6", len(variants))
	}
	for _, variant := range variants {
		fields := variant.(map[string]any)["properties"].(map[string]any)
		evidence := fields["evidence_source_keys"].(map[string]any)
		if evidence["minItems"] != float64(1) || !reflect.DeepEqual(evidence["items"].(map[string]any)["enum"], []any{"source_001"}) {
			t.Fatalf("provider evidence source enum = %#v", evidence)
		}
		for _, serverOwned := range []string{"node_id", "parent_node_id", "supports", "qualifies", "contrasts_with", "elaborates", "refers_to", "evidence_refs", "requirement_refs"} {
			if _, ok := fields[serverOwned]; ok {
				t.Fatalf("provider leaf exposes server-owned %s: %#v", serverOwned, fields)
			}
		}
	}
	for _, serverOwned := range []string{"schema_version", "pipeline_family", "document_id", "revision_id", "narrative_contract_id", "title", "blocks", "provenance", "references", "assets", "coverage", "extensions"} {
		if _, ok := properties[serverOwned]; ok {
			t.Fatalf("provider Document draft exposes server-owned %s", serverOwned)
		}
	}
}

func TestProviderDocumentDraftCompilesToCanonicalDocument(t *testing.T) {
	narrative := Narrative{
		SchemaVersion: NarrativeSchemaVersion, ContractID: "narrative_intersection", DocumentID: "document_intersection",
		CentralQuestion: "What does the source establish?", ReaderTakeaway: "A bounded conclusion.", Throughline: "Evidence to synthesis.",
		ReaderJourney: []string{"Read", "Conclude"}, ArgumentArc: []string{"Evidence", "Synthesis"},
		SectionRoles:          []SectionRole{{SectionID: "section.evidence", Role: "evidence", QuestionAnswered: "What?"}, {SectionID: "section.synthesis", Role: "synthesis", QuestionAnswered: "What follows?"}},
		ConclusionObligations: []string{"Conclude"}, VoiceAndTone: "Direct",
	}
	roleEvidence, roleSynthesis := "evidence", "synthesis"
	draft := documentDraft{Language: "en", Sections: map[string]documentSectionDraft{
		"section.evidence":  {Title: "Evidence", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "The source establishes one bounded fact.", EvidenceSourceKeys: []string{"source_001"}, SemanticRole: &roleEvidence}}},
		"section.synthesis": {Title: "Synthesis", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "That fact supports one bounded conclusion.", EvidenceSourceKeys: []string{"source_001"}, SemanticRole: &roleSynthesis}, {Kind: "table", Table: &documentTableDraft{Column1: "Boundary", Column2: "Value", Rows: []documentTableRowDraft{{Cell1: "Scope", Cell2: "Bounded"}}}, EvidenceSourceKeys: []string{"source_001"}}}},
	}}

	catalog := testSourceCatalog(t, "mis_fixture")
	schemaRaw := providerDocumentSchemaBytes(narrative, catalog)
	compiler := jsonschema.NewCompiler()
	var providerSchema any
	if err := json.Unmarshal(schemaRaw, &providerSchema); err != nil {
		t.Fatal(err)
	}
	const schemaURL = "urn:plasma:test:request-local-document"
	if err := compiler.AddResource(schemaURL, providerSchema); err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.Compile(schemaURL)
	if err != nil {
		t.Fatal(err)
	}
	generic, err := schemaValue(draft)
	if err != nil {
		t.Fatal(err)
	}
	if err := compiled.Validate(generic); err != nil {
		t.Fatalf("valid authored draft rejected by provider schema: %v", err)
	}
	document, err := compileDocumentDraft("Bounded Source Conclusion", narrative, catalog, testSourceReadReceipt(catalog), draft)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateProductDocument(narrative, document); err != nil {
		t.Fatalf("server-compiled Document rejected: %v", err)
	}
	if len(document.Blocks) != 5 || document.Blocks[0].NodeID != "section.evidence" || document.Blocks[2].NodeID != "section.synthesis" {
		t.Fatalf("server-owned topology = %#v", document.Blocks)
	}
	seen := map[string]bool{}
	for _, block := range document.Blocks {
		if seen[block.NodeID] {
			t.Fatalf("server generated duplicate node ID %q", block.NodeID)
		}
		seen[block.NodeID] = true
		if block.Kind != "section" && !seen[block.ParentNodeID] {
			t.Fatalf("server generated invalid parent %q", block.ParentNodeID)
		}
	}
	if table := document.Blocks[4].Table; table == nil || len(table.Columns) != 2 || len(table.Rows) != 1 || len(table.Rows[0]) != 2 {
		t.Fatalf("server table shape = %#v", table)
	}
	if len(document.References) != 1 || document.References[0].Kind != "footnote" || document.References[0].Target != "accepted-source:001" || document.References[0].VisibleLabel != "Accepted source 1" || document.References[0].Locator != "Frozen source catalog" {
		t.Fatalf("server-owned accepted-source references = %#v", document.References)
	}
	for _, index := range []int{1, 3, 4} {
		if !reflect.DeepEqual(document.Blocks[index].EvidenceRefs, []string{document.References[0].RefID}) {
			t.Fatalf("block %d evidence refs = %#v", index, document.Blocks[index].EvidenceRefs)
		}
	}
	markdown, _, err := RenderMarkdown(document)
	if err != nil {
		t.Fatal(err)
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	for _, rendered := range []string{string(mustMarshal(document)), string(markdown), string(html)} {
		for _, forbidden := range []string{"source_001", "src_fixture", "art_fixture", "fixture source content", "https://"} {
			if strings.Contains(rendered, forbidden) {
				t.Fatalf("compiled citation artifact leaked %q: %s", forbidden, rendered)
			}
		}
	}
	if !strings.Contains(string(markdown), "<sup>[1]</sup>") || !strings.Contains(string(markdown), "## References\n\n1. Accepted source 1") || strings.Contains(string(markdown), "Sources:") || strings.Contains(string(markdown), "Frozen source catalog") {
		t.Fatalf("Markdown claim provenance is not reader-facing:\n%s", markdown)
	}
	if !strings.Contains(string(html), `class="evidence-refs" aria-label="References"`) || !strings.Contains(string(html), `href="#`+document.References[0].RefID+`"`) || !strings.Contains(string(html), `>1</a>`) || !strings.Contains(string(html), `<h2 id="references">References</h2>`) || !strings.Contains(string(html), `<li id="`+document.References[0].RefID+`">Accepted source 1`) || strings.Contains(string(html), "Frozen source catalog") {
		t.Fatalf("HTML claim provenance is not reader-facing:\n%s", html)
	}
	second, err := compileDocumentDraft("Bounded Source Conclusion", narrative, catalog, testSourceReadReceipt(catalog), draft)
	if err != nil || !reflect.DeepEqual(document, second) {
		t.Fatalf("server compiler is not deterministic: err=%v\nfirst=%#v\nsecond=%#v", err, document, second)
	}
	if document.Blocks[1].NodeID == document.Blocks[3].NodeID || !strings.HasPrefix(document.Blocks[1].NodeID, "node.") {
		t.Fatalf("server node namespace = %#v", document.Blocks)
	}
}

func TestReaderProjectionCompactsEvidenceBySection(t *testing.T) {
	document := Document{
		Title: "Readable evidence", Language: "en",
		Blocks: []Block{
			{NodeID: "section.one", Kind: "section", Level: 2, Title: "One"},
			{NodeID: "prose.one", ParentNodeID: "section.one", Kind: "prose", Prose: "First claim.", EvidenceRefs: []string{"ref.one"}},
			{NodeID: "prose.two", ParentNodeID: "section.one", Kind: "prose", Prose: "Second claim.", EvidenceRefs: []string{"ref.two", "ref.one"}},
			{NodeID: "prose.three", ParentNodeID: "section.one", Kind: "prose", Prose: "Section synthesis.", EvidenceRefs: []string{"ref.one"}},
			{NodeID: "section.two", Kind: "section", Level: 2, Title: "Two"},
			{NodeID: "prose.four", ParentNodeID: "section.two", Kind: "prose", Prose: "Final claim.", EvidenceRefs: []string{"ref.two"}},
		},
		References: []Reference{
			{RefID: "ref.one", Kind: "footnote", Target: "accepted-source:001", VisibleLabel: "Source one"},
			{RefID: "ref.two", Kind: "footnote", Target: "accepted-source:002", VisibleLabel: "Source two"},
		},
	}
	original := append([]Block(nil), document.Blocks...)
	markdown, _, err := RenderMarkdown(document)
	if err != nil {
		t.Fatal(err)
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(document.Blocks, original) {
		t.Fatal("reader projection mutated block-level IL provenance")
	}
	if strings.Count(string(markdown), "<sup>") != 2 || !strings.Contains(string(markdown), "Section synthesis.<sup>[1], [2]</sup>") || !strings.Contains(string(markdown), "Final claim.<sup>[2]</sup>") || strings.Contains(string(markdown), "First claim.<sup>") {
		t.Fatalf("Markdown did not compact evidence into one inline group per section:\n%s", markdown)
	}
	if strings.Count(string(html), `class="evidence-refs"`) != 2 || !strings.Contains(string(html), `Section synthesis.<sup class="evidence-refs"`) || !strings.Contains(string(html), `Final claim.<sup class="evidence-refs"`) || strings.Contains(string(html), `First claim.<sup`) {
		t.Fatalf("HTML did not compact evidence into one inline group per section:\n%s", html)
	}
}

func TestRenderReferencesUsesDocumentLanguage(t *testing.T) {
	document := Document{
		Title: "한국어 보고서", Language: "ko",
		Blocks:     []Block{{NodeID: "section.ko", Kind: "section", Level: 2, Title: "본문"}},
		References: []Reference{{RefID: "ref.ko", Kind: "footnote", Target: "accepted-source:001", VisibleLabel: "공식 자료", Locator: "Frozen source catalog"}},
	}
	markdown, _, err := RenderMarkdown(document)
	if err != nil {
		t.Fatal(err)
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "## 근거") || !strings.Contains(string(html), `aria-labelledby="references"><h2 id="references">근거</h2>`) {
		t.Fatalf("Korean reference heading was not localized:\n%s\n%s", markdown, html)
	}
}

func TestCompileAuthorDraftRequiresAcceptedLanguageReview(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_author_language_review")
	author := authorDraft{
		Title: "Readable report", Language: "en", ReaderTakeaway: "One grounded answer.",
		Throughline: "One grounded answer.", VoiceAndTone: "Direct.",
		Sections: []authorSectionDraft{
			{Title: "Answer", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "A grounded claim.", EvidenceSourceKeys: []string{"source_001"}}}},
			{Title: "Conclusion", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "The answer holds.", EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
	if _, _, err := compileAuthorDraft(author, "narrative_language", "doc_language", "Explain the answer.", catalog, testSourceReadReceipt(catalog)); err == nil || !strings.Contains(err.Error(), "language review") {
		t.Fatalf("author accepted an unreviewed manuscript: %v", err)
	}
	author.LanguageReview = acceptedLanguageReview()
	if _, _, err := compileAuthorDraft(author, "narrative_language", "doc_language", "Explain the answer.", catalog, testSourceReadReceipt(catalog)); err != nil {
		t.Fatalf("author rejected an accepted language review: %v", err)
	}
}

func TestSourceGroundedTerminologyRejectsMechanicalNamesAndUnexplainedTerms(t *testing.T) {
	content := "法樹寺（ほうじゅじ）は此隅山城と山陰道の近くにある。建物は50間と記録された。"
	catalog := testSourceCatalog(t, "mis_terminology")
	readable := map[int]string{catalog.Sources[0].AcceptedOrdinal: content}
	receipt := testSourceReadReceipt(catalog)
	base := authorDraft{
		LanguageReview: acceptedLanguageReview(),
		Title:          "다케다성", Language: "ko", ReaderTakeaway: "지역사를 이해한다.",
		Throughline: "성곽과 교통의 관계를 설명한다.", VoiceAndTone: "자연스럽고 직접적",
		Sections: []authorSectionDraft{
			{Title: "지역의 성곽", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "호주지(法樹寺, 일본어 독음 ほうじゅじ)는 고노스미산성(此隅山城)과 관계가 있다.", EvidenceSourceKeys: []string{"source_001"}}}},
			{Title: "길과 단위", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "산인도(山陰道)는 고대 일본의 행정·교통로였고, 건물 길이는 50켄(間, 약 90.9m)으로 기록됐다.", EvidenceSourceKeys: []string{"source_001"}}}},
		},
		Terminology: []authorTermDraft{
			{Category: "proper_name", SourceForm: "法樹寺", SourceReading: stringPointer("ほうじゅじ"), ReaderForm: "호주지", Handling: "preserve_and_explain"},
			{Category: "proper_name", SourceForm: "此隅山城", ReaderForm: "고노스미산성", Handling: "preserve_and_explain"},
			{Category: "official_designation", SourceForm: "山陰道", ReaderForm: "산인도", Handling: "preserve_and_explain"},
			{Category: "foreign_unit", SourceForm: "間", ReaderForm: "켄", Handling: "convert_and_explain"},
		},
	}
	narrative, document, terms, err := compileAuthorDraftWithTerminology(
		base, "narrative_terms", "doc_terms", "다케다성을 설명한다.", catalog, receipt, nil, readable,
	)
	if err != nil {
		t.Fatalf("source-grounded terminology was rejected: %v", err)
	}
	if len(terms) != 4 || len(narrative.ContinuityTerms) != 4 || narrative.ContinuityTerms[0].Canonical != "호주지" || !slices.Contains(narrative.ContinuityTerms[0].Aliases, "法樹寺") || !slices.Contains(narrative.ContinuityTerms[0].Aliases, "ほうじゅじ") {
		t.Fatalf("compiled terminology = %#v / %#v", terms, narrative.ContinuityTerms)
	}

	omitted := base
	omitted.Terminology = nil
	if _, _, _, err := compileAuthorDraftWithTerminology(omitted, "narrative_omitted", "doc_omitted", "다케다성을 설명한다.", catalog, receipt, nil, readable); err == nil || !strings.Contains(err.Error(), "missing from the terminology inventory") {
		t.Fatalf("source-script terminology omission was accepted: %v", err)
	}

	mechanical := base
	mechanical.Terminology = append([]authorTermDraft(nil), base.Terminology...)
	mechanical.Terminology[0].ReaderForm = "법수사"
	mechanical.Terminology[0].Handling = "preserve"
	mechanical.Sections = append([]authorSectionDraft(nil), base.Sections...)
	mechanical.Sections[0].Blocks = append([]documentBlockDraft(nil), base.Sections[0].Blocks...)
	mechanical.Sections[0].Blocks[0].Prose = strings.ReplaceAll(mechanical.Sections[0].Blocks[0].Prose, "호주지(法樹寺, 일본어 독음 ほうじゅじ)", "법수사")
	if _, _, _, err := compileAuthorDraftWithTerminology(mechanical, "narrative_mechanical", "doc_mechanical", "다케다성을 설명한다.", catalog, receipt, nil, readable); err == nil || !strings.Contains(err.Error(), "requires an explanation") {
		t.Fatalf("mechanical 法樹寺 form was accepted: %v", err)
	}

	unsupported := base
	unsupported.Terminology = append([]authorTermDraft(nil), base.Terminology...)
	unsupported.Terminology[1].SourceReading = stringPointer("こすみやま")
	if _, _, _, err := compileAuthorDraftWithTerminology(unsupported, "narrative_unsupported", "doc_unsupported", "다케다성을 설명한다.", catalog, receipt, nil, readable); err == nil || !strings.Contains(err.Error(), "source reading") {
		t.Fatalf("unsupported 此隅山城 reading was accepted: %v", err)
	}

	unexplainedRoad := base
	unexplainedRoad.Sections = append([]authorSectionDraft(nil), base.Sections...)
	unexplainedRoad.Sections[1].Blocks = append([]documentBlockDraft(nil), base.Sections[1].Blocks...)
	unexplainedRoad.Sections[1].Blocks[0].Prose = "산인도가 이어지고, 건물 길이는 50켄(間, 약 90.9m)으로 기록됐다."
	unexplainedRoad.Terminology = append([]authorTermDraft(nil), base.Terminology...)
	_, unexplainedRoadDocument, unexplainedRoadTerms, err := compileAuthorDraftWithTerminology(unexplainedRoad, "narrative_road", "doc_road", "다케다성을 설명한다.", catalog, receipt, nil, readable)
	if err != nil {
		t.Fatalf("author should defer reader-facing first-use placement: %v", err)
	}
	unexplainedRoadReader := unchangedReaderDraft(unexplainedRoadDocument)
	unexplainedRoadReader.TerminologyEdits = keptTerminologyEdits(unexplainedRoadTerms)
	if _, _, err := compileReaderDraft(unexplainedRoadDocument, unexplainedRoadReader, "다케다성을 설명한다.", unexplainedRoadTerms); err == nil || !strings.Contains(err.Error(), "exact source form") {
		t.Fatalf("final reader accepted unexplained 山陰道: %v", err)
	}

	unexplainedUnit := base
	unexplainedUnit.Sections = append([]authorSectionDraft(nil), base.Sections...)
	unexplainedUnit.Sections[1].Blocks = append([]documentBlockDraft(nil), base.Sections[1].Blocks...)
	unexplainedUnit.Sections[1].Blocks[0].Prose = "산인도(山陰道)는 고대 일본의 행정·교통로였고, 건물 길이는 50켄으로 기록됐다."
	unexplainedUnit.Terminology = append([]authorTermDraft(nil), base.Terminology...)
	_, unexplainedUnitDocument, unexplainedUnitTerms, err := compileAuthorDraftWithTerminology(unexplainedUnit, "narrative_unit", "doc_unit", "다케다성을 설명한다.", catalog, receipt, nil, readable)
	if err != nil {
		t.Fatalf("author should defer reader-facing unit placement: %v", err)
	}
	unexplainedUnitReader := unchangedReaderDraft(unexplainedUnitDocument)
	unexplainedUnitReader.TerminologyEdits = keptTerminologyEdits(unexplainedUnitTerms)
	if _, _, err := compileReaderDraft(unexplainedUnitDocument, unexplainedUnitReader, "다케다성을 설명한다.", unexplainedUnitTerms); err == nil || !strings.Contains(err.Error(), "exact source form") {
		t.Fatalf("final reader accepted unexplained 間: %v", err)
	}

	lateExplanation := base
	lateExplanation.Sections = append([]authorSectionDraft(nil), base.Sections...)
	lateExplanation.Sections[0].Blocks = append([]documentBlockDraft(nil), base.Sections[0].Blocks...)
	lateExplanation.Sections[0].Blocks[0].Prose = "호주지는 고노스미산성(此隅山城)과 관계가 있다. 호주지는 法樹寺이며 일본어 독음은 ほうじゅじ다."
	lateExplanation.Terminology = append([]authorTermDraft(nil), base.Terminology...)
	_, lateDocument, lateTerms, err := compileAuthorDraftWithTerminology(lateExplanation, "narrative_late_explanation", "doc_late_explanation", "다케다성을 설명한다.", catalog, receipt, nil, readable)
	if err != nil {
		t.Fatalf("author should defer final sentence placement: %v", err)
	}
	lateReader := unchangedReaderDraft(lateDocument)
	lateReader.TerminologyEdits = keptTerminologyEdits(lateTerms)
	lateReader.TerminologyEdits["term_01"] = readerTermDraft{
		TermReceipt:          readerTermReceipt(lateTerms[0]),
		ReaderForm:           stringPointer(lateTerms[0].ReaderForm),
		RetainAsTerminology:  true,
		PresentSourceAliases: true,
	}
	if _, _, err := compileReaderDraft(lateDocument, lateReader, "다케다성을 설명한다.", lateTerms); err == nil ||
		(!strings.Contains(err.Error(), "exact source form") && !strings.Contains(err.Error(), "mixed-script lexical form") && !strings.Contains(err.Error(), "mixed Japanese-script lexical form") && !strings.Contains(err.Error(), "malformed Japanese-script name")) {
		t.Fatalf("final reader accepted later-sentence first-use explanation: %v", err)
	}

	lateAlias := base
	lateAlias.Sections = append([]authorSectionDraft(nil), base.Sections...)
	lateAlias.Sections[0].Blocks = append([]documentBlockDraft(nil), base.Sections[0].Blocks...)
	lateAlias.Sections[0].Blocks[0].Prose = base.Sections[0].Blocks[0].Prose + " 뒤 문장에 法樹寺를 다시 적었다."
	_, aliasDocument, aliasTerms, err := compileAuthorDraftWithTerminology(lateAlias, "narrative_late_alias", "doc_late_alias", "다케다성을 설명한다.", catalog, receipt, nil, readable)
	if err != nil {
		t.Fatalf("author should defer final alias placement: %v", err)
	}
	aliasReader := unchangedReaderDraft(aliasDocument)
	aliasReader.TerminologyEdits = keptTerminologyEdits(aliasTerms)
	if _, _, err := compileReaderDraft(aliasDocument, aliasReader, "다케다성을 설명한다.", aliasTerms); err == nil || !strings.Contains(err.Error(), "first body sentence") {
		t.Fatalf("final reader accepted source alias outside the first body sentence: %v", err)
	}

	missingSourceForm := base
	missingSourceForm.Terminology = append([]authorTermDraft(nil), base.Terminology...)
	missingSourceForm.Sections = append([]authorSectionDraft(nil), base.Sections...)
	missingSourceForm.Sections[0].Blocks = append([]documentBlockDraft(nil), base.Sections[0].Blocks...)
	missingSourceForm.Sections[0].Blocks[0].Prose = strings.ReplaceAll(missingSourceForm.Sections[0].Blocks[0].Prose, "(此隅山城)", "(일본의 산성)")
	_, missingSourceDocument, missingSourceTerms, err := compileAuthorDraftWithTerminology(missingSourceForm, "narrative_source_form", "doc_source_form", "다케다성을 설명한다.", catalog, receipt, nil, readable)
	if err != nil {
		t.Fatalf("author should defer final source-form presentation: %v", err)
	}
	missingSourceReader := unchangedReaderDraft(missingSourceDocument)
	missingSourceReader.TerminologyEdits = keptTerminologyEdits(missingSourceTerms)
	if _, _, err := compileReaderDraft(missingSourceDocument, missingSourceReader, "다케다성을 설명한다.", missingSourceTerms); err == nil || !strings.Contains(err.Error(), "exact source form") {
		t.Fatalf("final reader accepted a transformed name without its source form: %v", err)
	}

	missingReading := base
	missingReading.Terminology = append([]authorTermDraft(nil), base.Terminology...)
	missingReading.Sections = append([]authorSectionDraft(nil), base.Sections...)
	missingReading.Sections[0].Blocks = append([]documentBlockDraft(nil), base.Sections[0].Blocks...)
	missingReading.Sections[0].Blocks[0].Prose = strings.ReplaceAll(missingReading.Sections[0].Blocks[0].Prose, "法樹寺, 일본어 독음 ほうじゅじ", "法樹寺")
	_, missingReadingDocument, missingReadingTerms, err := compileAuthorDraftWithTerminology(missingReading, "narrative_reading", "doc_reading", "다케다성을 설명한다.", catalog, receipt, nil, readable)
	if err != nil {
		t.Fatalf("author should defer final reading presentation: %v", err)
	}
	missingReadingReader := unchangedReaderDraft(missingReadingDocument)
	missingReadingReader.TerminologyEdits = keptTerminologyEdits(missingReadingTerms)
	if _, _, err := compileReaderDraft(missingReadingDocument, missingReadingReader, "다케다성을 설명한다.", missingReadingTerms); err == nil || !strings.Contains(err.Error(), "source-supplied reading") {
		t.Fatalf("final reader accepted a transformed name without its source reading: %v", err)
	}

	finalProse := "호주지(法樹寺, 일본어 독음 ほうじゅじ)는 고노스미산성(此隅山城)과 관계가 있다."
	finalConclusion := "두 장소의 관계는 지역의 성곽사를 보여준다."
	reader := unchangedReaderDraft(document)
	reader.SectionEdits["section_01"].Blocks["block_01"] = readerBlockDraft{OriginalSHA256: readerBlockSHA256(document.Blocks[1]), Prose: &finalProse}
	reader.SectionEdits["section_02"].Blocks["block_01"] = readerBlockDraft{OriginalSHA256: readerBlockSHA256(document.Blocks[3]), Prose: &finalConclusion}
	reader.TerminologyEdits = keptTerminologyEdits(terms)
	reader.TerminologyEdits["term_03"] = readerTermDraft{TermReceipt: readerTermReceipt(terms[2])}
	reader.TerminologyEdits["term_04"] = readerTermDraft{TermReceipt: readerTermReceipt(terms[3])}
	_, _, finalTerms, err := compileReaderDraftWithTerminology(document, reader, "다케다성을 설명한다.", terms)
	if err != nil {
		t.Fatalf("reader could not remove unnecessary terms cleanly: %v", err)
	}
	expectedFirstUseSentence := strings.TrimSuffix(finalProse, ".")
	if len(finalTerms) != 2 || finalTerms[0].FirstUseExplanation != expectedFirstUseSentence || finalTerms[1].FirstUseExplanation != expectedFirstUseSentence {
		t.Fatalf("reader first-use sentences were not derived from final prose: %#v", finalTerms)
	}
	badReader := reader
	badReader.TerminologyEdits = keptTerminologyEdits(terms)
	badText := strings.ReplaceAll(document.Blocks[1].Prose, "호주지", "법수사")
	badReader.SectionEdits["section_01"].Blocks["block_01"] = readerBlockDraft{OriginalSHA256: readerBlockSHA256(document.Blocks[1]), Prose: &badText}
	if _, _, err := compileReaderDraft(document, badReader, "다케다성을 설명한다.", terms); err == nil {
		t.Fatal("reader preserved a disallowed mechanical alias")
	}

	for name, leakedText := range map[string]string{
		"source key":       "호주지는 source_001이 뒷받침한다.",
		"source URL":       "호주지의 출처 URL: https://private.example/source",
		"source home path": "호주지의 근거 경로: ~/.private/source.txt",
	} {
		leakedReader := unchangedReaderDraft(document)
		leakedReader.TerminologyEdits = keptTerminologyEdits(terms)
		leakedReader.SectionEdits["section_01"].Blocks["block_01"] = readerBlockDraft{OriginalSHA256: readerBlockSHA256(document.Blocks[1]), Prose: &leakedText}
		if _, _, err := compileReaderDraft(document, leakedReader, "다케다성을 설명한다.", terms); err == nil || !strings.Contains(err.Error(), "source metadata") {
			t.Fatalf("reader-facing %s was accepted: %v", name, err)
		}
	}

	renamed := unchangedReaderDraft(document)
	renamedTerms := keptTerminologyEdits(terms)
	newForm := "호주사"
	renamedTerms["term_01"] = readerTermDraft{
		TermReceipt:          readerTermReceipt(terms[0]),
		ReaderForm:           &newForm,
		RetainAsTerminology:  true,
		PresentSourceAliases: true,
	}
	renamed.TerminologyEdits = renamedTerms
	renamedText := strings.ReplaceAll(document.Blocks[1].Prose, "호주지", newForm)
	renamed.SectionEdits["section_01"].Blocks["block_01"] = readerBlockDraft{OriginalSHA256: readerBlockSHA256(document.Blocks[1]), Prose: &renamedText}
	if _, _, err := compileReaderDraft(document, renamed, "다케다성을 설명한다.", terms); err != nil {
		t.Fatalf("reader could not replace a canonical term: %v", err)
	}
	oldFormSurvives := renamed
	oldFormSurvivesText := renamedText + " 과거 초안은 호주지라고 썼다."
	oldFormSurvives.SectionEdits["section_01"].Blocks["block_01"] = readerBlockDraft{OriginalSHA256: readerBlockSHA256(document.Blocks[1]), Prose: &oldFormSurvivesText}
	if _, _, err := compileReaderDraft(document, oldFormSurvives, "다케다성을 설명한다.", terms); err == nil {
		t.Fatal("reader kept the superseded canonical form")
	}

	aliasCollision := unchangedReaderDraft(document)
	aliasCollisionTerms := keptTerminologyEdits(terms)
	collidingForm := terms[1].SourceForm
	aliasCollisionTerms["term_01"] = readerTermDraft{
		TermReceipt:          readerTermReceipt(terms[0]),
		ReaderForm:           &collidingForm,
		RetainAsTerminology:  true,
		PresentSourceAliases: true,
	}
	aliasCollision.TerminologyEdits = aliasCollisionTerms
	aliasCollisionText := strings.ReplaceAll(document.Blocks[1].Prose, terms[0].ReaderForm, collidingForm)
	aliasCollision.SectionEdits["section_01"].Blocks["block_01"] = readerBlockDraft{OriginalSHA256: readerBlockSHA256(document.Blocks[1]), Prose: &aliasCollisionText}
	if _, _, err := compileReaderDraft(document, aliasCollision, "다케다성을 설명한다.", terms); err == nil || !strings.Contains(err.Error(), "aliases overlap") {
		t.Fatalf("reader created an alias collision across terms: %v", err)
	}

	containedAliases := []terminologyDecision{
		{Category: "proper_name", SourceForm: "竹田", ReaderForm: "다케다", Handling: "preserve_and_explain"},
		{Category: "proper_name", SourceForm: "竹田城", ReaderForm: "다케다성", Handling: "preserve_and_explain"},
	}
	if err := validateOriginalTerminologyAliases(containedAliases); err == nil || !strings.Contains(err.Error(), "aliases overlap") {
		t.Fatalf("contained aliases were accepted across terms: %v", err)
	}
}

func testPrivateUserPath(user string, parts ...string) string {
	return "/" + strings.Join(append([]string{"Users", user}, parts...), "/")
}

func TestReaderMetadataAllowsTechnicalLocatorsWithoutSourceNarration(t *testing.T) {
	for name, value := range map[string]string{
		"public API endpoint": "공식 구성 예시는 https://api.example.com/v1/responses를 호출한다.",
		"WebSocket endpoint":  `구성 파일에는 "url": "wss://example.com/mcp"를 넣는다.`,
		"home config path":    "Codex는 ~/.codex/config.toml에서 프로젝트 설정을 읽는다.",
	} {
		document := Document{
			Title: "Technical report", Language: "ko",
			Blocks: []Block{
				{NodeID: "section_01", Kind: "section", Level: 2, Title: "설정"},
				{NodeID: "block_01", ParentNodeID: "section_01", Kind: "prose", Prose: value},
			},
		}
		if err := validateReaderFacingDocumentExceptMixedScript(document); err != nil {
			t.Fatalf("reader-facing technical %s was rejected: %v", name, err)
		}
	}
}

func TestReaderMetadataRejectsSourceLocatorNarration(t *testing.T) {
	for name, value := range map[string]string{
		"source key":       "이 주장은 source_001이 뒷받침한다.",
		"source URL":       "출처 URL: https://private.example/source",
		"source home path": "근거 경로: ~/.private/source.txt",
		"private path":     "자료는 " + testPrivateUserPath("researcher", "private", "source.txt") + "에 있다.",
	} {
		document := Document{
			Title: "Reader report", Language: "ko",
			Blocks: []Block{
				{NodeID: "section_01", Kind: "section", Level: 2, Title: "설명"},
				{NodeID: "block_01", ParentNodeID: "section_01", Kind: "prose", Prose: value},
			},
		}
		err := validateReaderFacingDocumentExceptMixedScript(document)
		if err == nil || !strings.Contains(err.Error(), "source metadata") {
			t.Fatalf("reader-facing %s was accepted: %v", name, err)
		}
	}
}

func TestReaderCanKeepNaturalWordingWithoutLowValueSourceAliases(t *testing.T) {
	terms := []terminologyDecision{
		{Category: "foreign_unit", SourceForm: "標高353m", ReaderForm: "해발 약 353미터", Handling: "convert_and_explain"},
		{Category: "specialist_term", SourceForm: "unkai", ReaderForm: "운해", Handling: "preserve_and_explain"},
		{Category: "specialist_term", SourceForm: "廃城", ReaderForm: "廃城", Handling: "preserve_and_explain"},
		{Category: "proper_name", SourceForm: "日本", ReaderForm: "일본", Handling: "preserve_and_explain"},
	}
	document := Document{
		Language: "ko",
		Blocks: []Block{
			{Kind: "section", Title: "다케다성"},
			{Kind: "prose", Prose: "정상 높이는 약 353미터이며, 운해는 산 사이에 구름이 바다처럼 펼쳐지는 현상이다. 일본의 이 성은 1600년 뒤 군사 기능을 잃었다."},
		},
	}
	cloud := terms[1].ReaderForm
	country := terms[3].ReaderForm
	edits := map[string]readerTermDraft{
		"term_01": {
			TermReceipt: readerTermReceipt(terms[0]), ReaderForm: nil,
			RetainAsTerminology: false, PresentSourceAliases: false,
		},
		"term_02": {
			TermReceipt: readerTermReceipt(terms[1]), ReaderForm: &cloud,
			RetainAsTerminology: true, PresentSourceAliases: false,
		},
		"term_03": {
			TermReceipt: readerTermReceipt(terms[2]), ReaderForm: nil,
			RetainAsTerminology: false, PresentSourceAliases: false,
		},
		"term_04": {
			TermReceipt: readerTermReceipt(terms[3]), ReaderForm: &country,
			RetainAsTerminology: false, PresentSourceAliases: false,
		},
	}
	compiled, err := compileReaderTerminology(document, edits, terms)
	if err != nil {
		t.Fatalf("natural reader wording was rejected: %v", err)
	}
	if len(compiled) != 1 || compiled[0].ReaderForm != "운해" || !compiled[0].HideSourceAliases || compiled[0].FirstUseExplanation == "" {
		t.Fatalf("final reader terminology = %#v", compiled)
	}
	continuity := continuityTerms(compiled)
	if len(continuity) != 1 || continuity[0].Canonical != "운해" || len(continuity[0].Aliases) != 0 {
		t.Fatalf("low-value source aliases survived continuity terms: %#v", continuity)
	}
	for _, forbidden := range []string{"標高353m", "unkai", "廃城", "日本"} {
		if readerValuesContain(documentReaderValues(document), forbidden) {
			t.Fatalf("low-value source alias %q survived the reader manuscript", forbidden)
		}
	}

	aliasesRemain := document
	aliasesRemain.Blocks = append([]Block(nil), document.Blocks...)
	aliasesRemain.Blocks[1].Prose += " 원문은 unkai라고 적는다."
	if _, err := compileReaderTerminology(aliasesRemain, edits, terms); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeTerminologyAliasPlacement {
		t.Fatalf("hidden source alias was accepted: code=%q err=%v", providerValidationCode(err), err)
	}

	cloneEdits := func() map[string]readerTermDraft {
		result := make(map[string]readerTermDraft, len(edits))
		for key, edit := range edits {
			result[key] = edit
		}
		return result
	}
	unretainedSpecialist := cloneEdits()
	unretainedSpecialist["term_02"] = readerTermDraft{
		TermReceipt: readerTermReceipt(terms[1]), ReaderForm: &cloud,
		RetainAsTerminology: false, PresentSourceAliases: false,
	}
	if _, err := compileReaderTerminology(document, unretainedSpecialist, terms); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeTerminologyInventory {
		t.Fatalf("unretained specialist bypass was accepted: code=%q err=%v", providerValidationCode(err), err)
	}

	height := terms[0].ReaderForm
	unretainedUnit := cloneEdits()
	unretainedUnit["term_01"] = readerTermDraft{
		TermReceipt: readerTermReceipt(terms[0]), ReaderForm: &height,
		RetainAsTerminology: false, PresentSourceAliases: false,
	}
	if _, err := compileReaderTerminology(document, unretainedUnit, terms); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeTerminologyInventory {
		t.Fatalf("unretained foreign-unit bypass was accepted: code=%q err=%v", providerValidationCode(err), err)
	}
}

func TestReaderSourceScriptSpecialistWithoutReadingCannotInventPronunciation(t *testing.T) {
	term := terminologyDecision{
		Category: "specialist_term", SourceForm: "穴太積み",
		ReaderForm: "아노즈쿠리", Handling: "preserve_and_explain",
	}
	document := Document{
		Language: "ko",
		Blocks: []Block{
			{Kind: "section", Title: "석벽"},
			{Kind: "prose", Prose: "아노즈쿠리(穴太積み)는 다케다성에 사용된 돌쌓기 방식이다."},
		},
	}
	invented := map[string]readerTermDraft{"term_01": {
		TermReceipt: readerTermReceipt(term), ReaderForm: stringPointer("아노즈쿠리"),
		RetainAsTerminology: true, PresentSourceAliases: true,
	}}
	if _, err := compileReaderTerminology(document, invented, []terminologyDecision{term}); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeTerminologyReaderForm {
		t.Fatalf("invented specialist pronunciation code=%q err=%v", providerValidationCode(err), err)
	}

	exactDocument := document
	exactDocument.Blocks = append([]Block(nil), document.Blocks...)
	exactDocument.Blocks[1].Prose = "穴太積み는 다케다성에 사용된 돌쌓기 방식이다."
	exact := map[string]readerTermDraft{"term_01": {
		TermReceipt: readerTermReceipt(term), ReaderForm: stringPointer(term.SourceForm),
		RetainAsTerminology: true, PresentSourceAliases: true,
	}}
	if _, err := compileReaderTerminology(exactDocument, exact, []terminologyDecision{term}); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeTerminologyReaderForm {
		t.Fatalf("Korean reader kept a source-script specialist term without a source reading: code=%q err=%v", providerValidationCode(err), err)
	}

	removedDocument := document
	removedDocument.Blocks = []Block{{Kind: "section", Title: "석벽"}, {Kind: "prose", Prose: "이 돌쌓기 방식은 다케다성의 석벽에 사용되었다."}}
	removed := map[string]readerTermDraft{"term_01": {TermReceipt: readerTermReceipt(term)}}
	compiled, err := compileReaderTerminology(removedDocument, removed, []terminologyDecision{term})
	if err != nil || len(compiled) != 0 {
		t.Fatalf("naturally rewritten specialist fact was rejected: %#v err=%v", compiled, err)
	}

	schema := string(providerReaderSchemaBytes(exactDocument, []terminologyDecision{term}))
	if !strings.Contains(schema, `"reader_form":{"type":"null"}`) || strings.Contains(schema, `"reader_form":{"const":"穴太積み"`) || !strings.Contains(schema, `"retain_as_terminology"`) || !strings.Contains(schema, `"present_source_aliases"`) || strings.Contains(schema, `"const":"아노즈쿠리"`) {
		t.Fatalf("Korean reader schema does not require source-form removal: %s", schema)
	}

	requiredTerms := []terminologyDecision{term}
	markRequiredSourceForms(requiredTerms, "穴太積み를 설명한다.", "")
	requiredSchema := string(providerReaderSchemaBytes(exactDocument, requiredTerms))
	if !requiredTerms[0].PreserveSourceForm || !strings.Contains(requiredSchema, `"reader_form":{"const":"穴太積み","type":"string"}`) || strings.Contains(requiredSchema, `"reader_form":{"type":"null"}`) {
		t.Fatalf("explicitly requested source form was not preserved exactly: terms=%#v schema=%s", requiredTerms, requiredSchema)
	}
	requiredExact := map[string]readerTermDraft{"term_01": {
		TermReceipt: readerTermReceipt(requiredTerms[0]), ReaderForm: stringPointer(term.SourceForm),
		RetainAsTerminology: true, PresentSourceAliases: true,
	}}
	compiled, err = compileReaderTerminology(exactDocument, requiredExact, requiredTerms)
	if err != nil || len(compiled) != 1 || compiled[0].ReaderForm != term.SourceForm {
		t.Fatalf("explicitly requested exact source form was rejected: %#v err=%v", compiled, err)
	}

	prompt := readerPromptTerminology([]terminologyDecision{term})
	if !strings.Contains(prompt, "source-supplied reading: none; do not invent a pronunciation") {
		t.Fatalf("reader terminology prompt does not expose the missing source reading: %s", prompt)
	}
}

func TestReaderTerminologyFailuresKeepGranularClosedCodes(t *testing.T) {
	base := terminologyDecision{
		Category: "proper_name", SourceForm: "法樹寺", SourceReading: "ほうじゅじ",
		ReaderForm: "호주지", Handling: "preserve_and_explain",
	}
	validDocument := Document{
		Language: "ko",
		Blocks: []Block{
			{Kind: "section", Title: "사찰"},
			{Kind: "prose", Prose: "호주지(法樹寺, 일본어 독음 ほうじゅじ)는 지역 사찰이다."},
		},
	}
	keep := func(term terminologyDecision) map[string]readerTermDraft {
		return map[string]readerTermDraft{"term_01": {
			TermReceipt: readerTermReceipt(term), ReaderForm: stringPointer(term.ReaderForm),
			RetainAsTerminology: true, PresentSourceAliases: true,
		}}
	}
	assertCode := func(name string, document Document, edits map[string]readerTermDraft, terms []terminologyDecision, want reportexecution.ProviderValidationCode) {
		t.Helper()
		_, err := compileReaderTerminology(document, edits, terms)
		if err == nil || providerValidationCode(err) != want {
			t.Fatalf("%s code=%q err=%v, want %q", name, providerValidationCode(err), err, want)
		}
	}

	assertCode("removal", validDocument, map[string]readerTermDraft{"term_01": {TermReceipt: readerTermReceipt(base)}}, []terminologyDecision{base}, reportexecution.ProviderValidationCodeTerminologyRemoval)

	renamed := base
	renamedForm := "호주사"
	renamedDocument := validDocument
	renamedDocument.Blocks = append([]Block(nil), validDocument.Blocks...)
	renamedDocument.Blocks[1].Prose = "호주사(法樹寺, 일본어 독음 ほうじゅじ)는 지역 사찰이다. 이전 표기는 호주지였다."
	assertCode("rename", renamedDocument, map[string]readerTermDraft{"term_01": {TermReceipt: readerTermReceipt(renamed), ReaderForm: &renamedForm, RetainAsTerminology: true, PresentSourceAliases: true}}, []terminologyDecision{renamed}, reportexecution.ProviderValidationCodeTerminologyRename)

	missingReader := validDocument
	missingReader.Blocks = append([]Block(nil), validDocument.Blocks...)
	missingReader.Blocks[1].Prose = "法樹寺는 지역 사찰이다."
	assertCode("reader form", missingReader, keep(base), []terminologyDecision{base}, reportexecution.ProviderValidationCodeTerminologyReaderForm)

	headingOnly := validDocument
	headingOnly.Blocks = []Block{{Kind: "section", Title: "호주지"}, {Kind: "prose", Prose: "이 사찰은 지역에 남아 있다."}}
	assertCode("first use", headingOnly, keep(base), []terminologyDecision{base}, reportexecution.ProviderValidationCodeTerminologyFirstUse)

	missingSource := validDocument
	missingSource.Blocks = append([]Block(nil), validDocument.Blocks...)
	missingSource.Blocks[1].Prose = "호주지는 일본의 지역 사찰이다."
	assertCode("source form", missingSource, keep(base), []terminologyDecision{base}, reportexecution.ProviderValidationCodeTerminologySourceForm)

	missingReading := validDocument
	missingReading.Blocks = append([]Block(nil), validDocument.Blocks...)
	missingReading.Blocks[1].Prose = "호주지(法樹寺)는 일본의 지역 사찰이다."
	assertCode("source reading", missingReading, keep(base), []terminologyDecision{base}, reportexecution.ProviderValidationCodeTerminologySourceReading)

	aliasOutside := validDocument
	aliasOutside.Blocks = append([]Block(nil), validDocument.Blocks...)
	aliasOutside.Blocks[1].Prose += " 뒤 기록은 法樹寺라고 다시 적는다."
	assertCode("alias placement", aliasOutside, keep(base), []terminologyDecision{base}, reportexecution.ProviderValidationCodeTerminologyAliasPlacement)

	other := terminologyDecision{Category: "proper_name", SourceForm: "別寺", ReaderForm: "호주지", Handling: "preserve_and_explain"}
	assertCode("alias collision", validDocument, map[string]readerTermDraft{}, []terminologyDecision{base, other}, reportexecution.ProviderValidationCodeTerminologyAliasCollision)

	uncoveredScript := validDocument
	uncoveredScript.Title = "竹田城"
	assertCode("script coverage", uncoveredScript, keep(base), []terminologyDecision{base}, reportexecution.ProviderValidationCodeTerminologyScriptCoverage)
}

func TestReaderTerminologyRepairGuidanceIsSpecificAndContentFree(t *testing.T) {
	cases := map[reportexecution.ProviderValidationCode]string{
		reportexecution.ProviderValidationCodeTerminologyInventory:      "original alias and receipt",
		reportexecution.ProviderValidationCodeTerminologyRemoval:        "every known form",
		reportexecution.ProviderValidationCodeTerminologyRename:         "superseded author reader form",
		reportexecution.ProviderValidationCodeTerminologyReaderForm:     "first body occurrence",
		reportexecution.ProviderValidationCodeTerminologyFirstUse:       "title or heading alone",
		reportexecution.ProviderValidationCodeTerminologySourceForm:     "exact source form",
		reportexecution.ProviderValidationCodeTerminologySourceReading:  "exact reading",
		reportexecution.ProviderValidationCodeTerminologyAliasPlacement: "remove every source form",
		reportexecution.ProviderValidationCodeTerminologyAliasCollision: "contains, or is contained",
		reportexecution.ProviderValidationCodeTerminologyScriptCoverage: "Hiragana",
		reportexecution.ProviderValidationCodeReaderOpening:             "first paragraph",
		reportexecution.ProviderValidationCodeReaderAuditVoice:          "distinct claim",
		reportexecution.ProviderValidationCodeReaderOrdinarySI:          "kilometers",
		reportexecution.ProviderValidationCodeReaderProcess:             "construction narration",
		reportexecution.ProviderValidationCodeReaderMetadata:            "source key",
		reportexecution.ProviderValidationCodeReaderInternalMachinery:   "report-system terminology",
		reportexecution.ProviderValidationCodeSupportedDetail:           "preservation_coverage",
		reportexecution.ProviderValidationCodeReportDepth:               "self-contained report",
	}
	for code, required := range cases {
		prompt := sourceFreeRepairPrompt("SOURCE-FREE MANUSCRIPT", "il_flow", "semantic_validation", code)
		if !strings.Contains(prompt, required) {
			t.Fatalf("repair %q lacks %q: %s", code, required, prompt)
		}
		for _, forbidden := range []string{"法樹寺", "ほうじゅじ", "호주지", "source_001", "PRIVATE VALIDATOR DETAIL"} {
			if strings.Contains(prompt, forbidden) {
				t.Fatalf("repair %q leaked %q: %s", code, forbidden, prompt)
			}
		}
	}
}

func TestAuthorEvidencePacketRepairGuidanceIsGranularAndContentFree(t *testing.T) {
	cases := map[reportexecution.ProviderValidationCode][]string{
		reportexecution.ProviderValidationCodeEvidencePacketInventory: {"complete evidence_packets inventory", "omit unused candidate receipts", "canonicalized to one packet"},
		reportexecution.ProviderValidationCodeEvidencePacketTarget:    {"exact section_NN/block_NN aliases"},
		reportexecution.ProviderValidationCodeEvidencePacketSource:    {"INVALID EVIDENCE PACKETS TO REPAIR", reportilcontract.SourceQuoteRegisterTool, "successful opaque receipt", "512 UTF-8 bytes"},
		reportexecution.ProviderValidationCodeEvidencePacketBinding:   {"every evidence_source_keys entry"},
		reportexecution.ProviderValidationCodeEvidencePacketCoverage:  {"union of its exact packet claims"},
	}
	for code, required := range cases {
		repairContext := "SAFE BODY"
		if code == reportexecution.ProviderValidationCodeEvidencePacketSource {
			repairContext += "\nINVALID EVIDENCE PACKETS TO REPAIR:\n- evidence_001: section_01/block_01 from source_001"
		}
		prompt := sourceRepairPrompt(
			"ORIGINAL", "il_narrative", "semantic_validation", repairContext, code, "",
		)
		for _, value := range append(required, "SAFE BODY", "Start a fresh source-tool session") {
			if !strings.Contains(prompt, value) {
				t.Fatalf("repair %q lacks %q: %s", code, value, prompt)
			}
		}
		forbiddenValues := []string{
			"PRIVATE VALIDATOR DETAIL", "accepted-source:001",
			"法樹寺", "https://private.example", testPrivateUserPath("alice"),
		}
		if code != reportexecution.ProviderValidationCodeEvidencePacketSource {
			forbiddenValues = append(forbiddenValues, "source_001")
		}
		for _, forbidden := range forbiddenValues {
			if strings.Contains(prompt, forbidden) {
				t.Fatalf("repair %q leaked %q: %s", code, forbidden, prompt)
			}
		}
		failure := providerStageFailure("il_narrative", &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  withValidationCode(code, errors.New("PRIVATE VALIDATOR DETAIL")),
		}, TerminalUsageReceipt{})
		request := failure.AppendRequest(
			"mis_safe", "evt_pending", "evt_terminal", ledger.Producer{Type: "agent", ID: "codex"},
		)
		encoded := string(request.Payload)
		if !strings.Contains(encoded, string(code)) || strings.Contains(encoded, "PRIVATE VALIDATOR DETAIL") {
			t.Fatalf("durable evidence failure leaked detail: %s", encoded)
		}
	}
}

func TestAuthorTerminologySchemaOmitsProviderSelectedSourceKeys(t *testing.T) {
	schema := authorTerminologySchema()
	required := schema["required"].([]any)
	properties := schema["properties"].(map[string]any)
	if slices.Contains(required, any("source_keys")) {
		t.Fatalf("terminology schema still requires provider-selected source keys: %#v", required)
	}
	if _, exists := properties["source_keys"]; exists {
		t.Fatalf("terminology schema still exposes provider-selected source keys: %#v", properties)
	}
}

func TestAuthorTerminologyGroundingIsDerivedAcrossSelectedSources(t *testing.T) {
	catalog := testSourceCatalogWithCount(t, "mis_derived_terminology_sources", 2)
	content := map[int]string{
		catalog.Sources[0].AcceptedOrdinal: "unrelated selected source",
		catalog.Sources[1].AcceptedOrdinal: "法樹寺（ほうじゅじ）は地域の寺院である。",
	}
	draft := authorTermDraft{
		Category: "proper_name", SourceForm: "法樹寺", SourceReading: stringPointer("ほうじゅじ"),
		ReaderForm: "호주지", Handling: "preserve_and_explain",
	}
	if _, err := validateAuthorTermDraft(draft, catalog, testSourceReadReceipt(catalog), content); err != nil {
		t.Fatalf("server-derived grounding rejected a term present in one selected source: %v", err)
	}
}

func TestAuthorTerminologyGroundingUsesGranularClosedCodesAndTermAlias(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_terminology_grounding_codes")
	receipt := testSourceReadReceipt(catalog)
	content := map[int]string{catalog.Sources[0].AcceptedOrdinal: strings.Repeat("x", 200) + "ほうじゅじ" + strings.Repeat("y", 200) + "法樹寺"}
	document := Document{Language: "ko", Blocks: []Block{{Kind: "prose", Prose: "호주지는 지역 사찰이다."}}}

	missingForm := authorTermDraft{Category: "proper_name", SourceForm: "不存在", ReaderForm: "호주지", Handling: "preserve_and_explain"}
	_, err := compileAuthorTerminology([]authorTermDraft{missingForm}, document, catalog, receipt, content)
	if err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeTerminologySourceFormGrounding || providerValidationTermAlias(err) != "term_01" {
		t.Fatalf("missing form code/alias = %q/%q, err=%v", providerValidationCode(err), providerValidationTermAlias(err), err)
	}

	distantReading := authorTermDraft{Category: "proper_name", SourceForm: "法樹寺", SourceReading: stringPointer("ほうじゅじ"), ReaderForm: "호주지", Handling: "preserve_and_explain"}
	_, err = compileAuthorTerminology([]authorTermDraft{distantReading}, document, catalog, receipt, content)
	if err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeTerminologySourceReadingGrounding || providerValidationTermAlias(err) != "term_01" {
		t.Fatalf("distant reading code/alias = %q/%q, err=%v", providerValidationCode(err), providerValidationTermAlias(err), err)
	}
}

func TestAuthorTerminologyInventoryFailuresCarryTermAlias(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_terminology_inventory_aliases")
	receipt := testSourceReadReceipt(catalog)
	content := map[int]string{catalog.Sources[0].AcceptedOrdinal: "法樹寺と別寺"}
	base := authorTermDraft{Category: "proper_name", SourceForm: "法樹寺", ReaderForm: "호주지", Handling: "preserve_and_explain"}
	document := Document{Language: "ko", Blocks: []Block{{Kind: "prose", Prose: "호주지는 지역 사찰이다."}}}

	duplicate := authorTermDraft{Category: "proper_name", SourceForm: "別寺", ReaderForm: "호주지", Handling: "preserve_and_explain"}
	_, err := compileAuthorTerminology([]authorTermDraft{base, duplicate}, document, catalog, receipt, content)
	if err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeTerminologyInventory || providerValidationTermAlias(err) != "term_02" {
		t.Fatalf("duplicate code/alias = %q/%q, err=%v", providerValidationCode(err), providerValidationTermAlias(err), err)
	}

	missingReader := authorTermDraft{Category: "proper_name", SourceForm: "別寺", ReaderForm: "별사", Handling: "preserve_and_explain"}
	_, err = compileAuthorTerminology([]authorTermDraft{base, missingReader}, document, catalog, receipt, content)
	if err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeTerminologyInventory || providerValidationTermAlias(err) != "term_02" {
		t.Fatalf("missing reader code/alias = %q/%q, err=%v", providerValidationCode(err), providerValidationTermAlias(err), err)
	}
	prompt := sourceRepairPrompt("ORIGINAL", "il_narrative", "semantic_validation", "SAFE BODY", providerValidationCode(err), providerValidationTermAlias(err))
	if !strings.Contains(prompt, "term_02") || !strings.Contains(prompt, "complete terminology array") {
		t.Fatalf("inventory repair lacks targeted alias: %s", prompt)
	}
}

func TestAuthorTerminologyGroundingRepairGuidanceIsGranularAndContentFree(t *testing.T) {
	cases := []struct {
		code     reportexecution.ProviderValidationCode
		required string
	}{
		{reportexecution.ProviderValidationCodeTerminologySourceFormGrounding, "exact source form that appears in a selected source"},
		{reportexecution.ProviderValidationCodeTerminologySourceReadingGrounding, "source_reading as null"},
	}
	for _, test := range cases {
		prompt := sourceRepairPrompt("ORIGINAL", "il_narrative", "semantic_validation", "SAFE BODY", test.code, "term_03")
		for _, required := range []string{"term_03", test.required, "SAFE BODY"} {
			if !strings.Contains(prompt, required) {
				t.Fatalf("repair %q lacks %q: %s", test.code, required, prompt)
			}
		}
		for _, forbidden := range []string{"法樹寺", "ほうじゅじ", "호주지", "source_001", "PRIVATE VALIDATOR DETAIL"} {
			if strings.Contains(prompt, forbidden) {
				t.Fatalf("repair %q leaked %q: %s", test.code, forbidden, prompt)
			}
		}
		failure := providerStageFailure("il_narrative", &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  withValidationCodeAndTerm(test.code, "term_03", errors.New("PRIVATE VALIDATOR DETAIL")),
		}, TerminalUsageReceipt{})
		request := failure.AppendRequest("mis_safe", "evt_pending", "evt_terminal", ledger.Producer{Type: "agent", ID: "codex"})
		encoded := string(request.Payload)
		if !strings.Contains(encoded, string(test.code)) || strings.Contains(encoded, "term_03") || strings.Contains(encoded, "PRIVATE VALIDATOR DETAIL") {
			t.Fatalf("durable failure payload leaked request-local detail: %s", encoded)
		}
	}
}

func TestSourceReadingMustBeGroundedNearItsSourceForm(t *testing.T) {
	content := strings.Repeat("x", 200) + "ほうじゅじ" + strings.Repeat("y", 200) + "法樹寺"
	catalog := testSourceCatalog(t, "mis_reading_distance")
	draft := authorTermDraft{
		Category: "proper_name", SourceForm: "法樹寺", SourceReading: stringPointer("ほうじゅじ"),
		ReaderForm: "호주지", Handling: "preserve_and_explain",
	}
	if _, err := validateAuthorTermDraft(draft, catalog, testSourceReadReceipt(catalog), map[int]string{catalog.Sources[0].AcceptedOrdinal: content}); err == nil || !strings.Contains(err.Error(), "source reading") {
		t.Fatalf("distant unrelated reading was accepted: %v", err)
	}
	adjacent := "法樹寺（ほうじゅじ）"
	if _, err := validateAuthorTermDraft(draft, catalog, testSourceReadReceipt(catalog), map[int]string{catalog.Sources[0].AcceptedOrdinal: adjacent}); err != nil {
		t.Fatalf("adjacent source reading was rejected: %v", err)
	}
}

func stringPointer(value string) *string { return &value }

func unchangedReaderDraft(document Document) readerDraft {
	draft := readerDraft{Title: document.Title, Reviewer: "whole-reader", LanguageReview: acceptedLanguageReview(), ReaderReview: acceptedReaderReview(), SectionEdits: map[string]readerSectionDraft{}, TerminologyEdits: map[string]readerTermDraft{}}
	for sectionIndex, section := range readerSections(document) {
		edit := readerSectionDraft{SectionReceipt: readerSectionReceipt(section), Title: section.Title, Blocks: map[string]readerBlockDraft{}}
		for blockIndex, block := range section.Blocks {
			candidate := readerBlockDraft{OriginalSHA256: readerBlockSHA256(block)}
			switch block.Kind {
			case "prose", "quote", "callout":
				value := block.Prose
				candidate.Prose = &value
			case "list":
				candidate.Items = append([]string(nil), block.Items...)
			case "code":
				value := block.Code
				candidate.Code = &value
			}
			edit.Blocks[readerBlockKey(blockIndex)] = candidate
		}
		draft.SectionEdits[readerSectionKey(sectionIndex)] = edit
	}
	return draft
}

func keptTerminologyEdits(terms []terminologyDecision) map[string]readerTermDraft {
	edits := make(map[string]readerTermDraft, len(terms))
	for index, term := range terms {
		readerForm := term.ReaderForm
		edits[readerTermKey(index)] = readerTermDraft{
			TermReceipt:          readerTermReceipt(term),
			ReaderForm:           &readerForm,
			RetainAsTerminology:  true,
			PresentSourceAliases: true,
		}
	}
	return edits
}

func TestAuthorDefersInternalMachineryCleanupToFinalReader(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_internal_machinery")
	author := authorDraft{
		LanguageReview: acceptedLanguageReview(),
		Title:          "Experimental IL report", Language: "en", ReaderTakeaway: "One grounded answer.",
		Throughline: "One grounded answer.", VoiceAndTone: "Direct.",
		Sections: []authorSectionDraft{
			{Title: "Answer", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "A grounded claim.", EvidenceSourceKeys: []string{"source_001"}}}},
			{Title: "Conclusion", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "The answer holds.", EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
	_, document, err := compileAuthorDraft(author, "narrative_internal", "doc_internal", "Explain the answer.", catalog, testSourceReadReceipt(catalog))
	if err != nil {
		t.Fatalf("author rejected final-reader-editable machinery: %v", err)
	}
	reader := unchangedReaderDraft(document)
	if _, _, err := compileReaderDraft(document, reader, "Explain the answer."); err == nil || !strings.Contains(err.Error(), "internal report machinery") {
		t.Fatalf("final reader accepted unedited machinery: %v", err)
	}
	reader.Title = "Direct answer"
	if _, _, err := compileReaderDraft(document, reader, "Explain the answer."); err != nil {
		t.Fatalf("final reader could not remove internal machinery: %v", err)
	}
}

func TestCompileAuthorDraftRendersPublicSourceTitleAndURLWithoutProviderExposure(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_public_citation")
	author := authorDraft{
		LanguageReview: acceptedLanguageReview(),
		Title:          "Readable report", Language: "en", ReaderTakeaway: "One grounded answer.",
		Throughline: "One grounded answer.", VoiceAndTone: "Direct.",
		Sections: []authorSectionDraft{
			{Title: "Answer", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "A grounded claim.", EvidenceSourceKeys: []string{"source_001"}}}},
			{Title: "Conclusion", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "The answer holds.", EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
	_, document, err := compileAuthorDraft(author, "narrative_public", "doc_public", "Explain the answer.", catalog, testSourceReadReceipt(catalog), map[int]SourceCitation{
		1: {VisibleLabel: "Official public source", URL: "https://example.com/public"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(document.References) != 1 || document.References[0].Kind != "footnote" || document.References[0].Target != "accepted-source:001" || document.References[0].VisibleLabel != "Official public source" || document.References[0].Locator != "https://example.com/public" {
		t.Fatalf("public citation reference = %#v", document.References)
	}
	markdown, _, err := RenderMarkdown(document)
	if err != nil {
		t.Fatal(err)
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "1. [Official public source](https://example.com/public)") || strings.Contains(string(markdown), " — https://example.com/public") {
		t.Fatalf("Markdown omitted compact public citation:\n%s", markdown)
	}
	if !strings.Contains(string(html), `<li id="`+document.References[0].RefID+`"><a href="https://example.com/public">Official public source</a></li>`) || strings.Contains(string(html), ` — <a href="https://example.com/public">https://example.com/public</a>`) {
		t.Fatalf("HTML omitted compact public citation:\n%s", html)
	}
}

func TestCompileAuthorDraftNormalizesDuplicateEvidenceKeys(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_duplicate_evidence")
	author := authorDraft{
		LanguageReview: acceptedLanguageReview(),
		Title:          "Readable report", Language: "en", ReaderTakeaway: "One grounded answer.",
		Throughline: "One grounded answer.", VoiceAndTone: "Direct.",
		Sections: []authorSectionDraft{
			{Title: "Answer", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "A grounded claim.", EvidenceSourceKeys: []string{"source_001", "source_001"}}}},
			{Title: "Conclusion", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "The answer holds.", EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
	_, document, err := compileAuthorDraft(author, "narrative_duplicate", "doc_duplicate", "Explain the answer.", catalog, testSourceReadReceipt(catalog))
	if err != nil {
		t.Fatal(err)
	}
	if len(document.References) != 1 || len(document.Blocks[1].EvidenceRefs) != 1 {
		t.Fatalf("duplicate evidence was not normalized: references=%#v block=%#v", document.References, document.Blocks[1])
	}
}

func TestCompileDocumentDraftRejectsUnreadOrExposedSourceKeys(t *testing.T) {
	narrative := Narrative{
		SchemaVersion: NarrativeSchemaVersion, ContractID: "narrative_citation", DocumentID: "document_citation",
		CentralQuestion: "What?", ReaderTakeaway: "One fact.", Throughline: "Evidence.", ReaderJourney: []string{"Read"}, ArgumentArc: []string{"Conclude"},
		SectionRoles: []SectionRole{{SectionID: "section.citation", Role: "evidence", QuestionAnswered: "What?"}}, ConclusionObligations: []string{"Conclude"}, VoiceAndTone: "Direct",
	}
	catalog := testSourceCatalog(t, "mis_fixture")
	draft := documentDraft{Language: "en", Sections: map[string]documentSectionDraft{
		"section.citation": {Title: "Citation", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "One fact.", EvidenceSourceKeys: []string{"source_001"}}}},
	}}

	unread := reportilcontract.SourceReadReceipt{SourceKeys: []string{"source_999"}, ReturnedContentBytes: 1}
	if _, err := compileDocumentDraft("Citation", narrative, catalog, unread, draft); err == nil {
		t.Fatal("unread evidence source was accepted")
	}
	draft.Sections["section.citation"] = documentSectionDraft{Title: "Citation", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "Internal source_001 leaked.", EvidenceSourceKeys: []string{"source_001"}}}}
	if _, err := compileDocumentDraft("Citation", narrative, catalog, testSourceReadReceipt(catalog), draft); err == nil || !strings.Contains(err.Error(), "internal source key") {
		t.Fatalf("authored source-key leak result = %v", err)
	}
}

func TestValidateFlowEvidenceReadsRequiresEveryProseCitation(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_fixture")
	document := Document{
		References: []Reference{{RefID: "ref.accepted_source.test", Kind: "footnote", Target: "accepted-source:001", VisibleLabel: "Accepted source 1"}},
		Blocks:     []Block{{NodeID: "prose.1", Kind: "prose", Prose: "Fact.", EvidenceRefs: []string{"ref.accepted_source.test"}}},
	}
	if err := validateFlowEvidenceReads(document, catalog, testSourceReadReceipt(catalog)); err != nil {
		t.Fatal(err)
	}
	partialReceipt := reportilcontract.SourceReadReceipt{
		SourceKeys:           []string{"source_001"},
		ReturnedContentBytes: 1,
	}
	if err := validateFlowEvidenceReads(document, catalog, partialReceipt); err == nil {
		t.Fatal("flow accepted a partial read of a prose citation source")
	}
	if err := validateFlowEvidenceReads(document, catalog, reportilcontract.SourceReadReceipt{SourceKeys: []string{"source_999"}, ReturnedContentBytes: 1}); err == nil {
		t.Fatal("flow accepted a prose citation whose source was not read")
	}
	document.References[0].Target = "accepted-source:999"
	if err := validateFlowEvidenceReads(document, catalog, testSourceReadReceipt(catalog)); err == nil {
		t.Fatal("flow accepted an out-of-catalog evidence ordinal")
	}
}

func TestReaderStageRejectsIncompleteSelectedSourceRead(t *testing.T) {
	catalog := testSourceCatalogWithCount(t, "mis_reader_complete", 2)
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: "doc_reader_complete", RevisionID: "draft", NarrativeContractID: "narr_reader_complete",
		Title: "Bounded report", Language: "en",
		Blocks: []Block{
			{NodeID: "section.opening", Kind: "section", Level: 2, Title: "Opening"},
			{NodeID: "node.opening", ParentNodeID: "section.opening", Kind: "prose", Prose: "One bounded fact."},
			{NodeID: "section.conclusion", Kind: "section", Level: 2, Title: "Conclusion"},
			{NodeID: "node.conclusion", ParentNodeID: "section.conclusion", Kind: "prose", Prose: "One bounded conclusion."},
		},
	}
	draft := unchangedReaderDraft(document)
	partial := testSourceReadReceipt(catalog)
	partial.FullyReadSourceKeys = partial.FullyReadSourceKeys[:1]
	partial.ReadBytesBySource[catalog.Sources[1].SourceKey]--
	partial.ReturnedContentBytes--
	provider := &recordingProvider{
		outputs: []string{string(mustMarshal(draft)), string(mustMarshal(draft))},
	}
	verifier := &fixedSourceReadVerifier{receipt: partial}
	config := testSourceProductConfig(provider, verifier)
	config.MissionID = catalog.MissionID
	_, attempts, err := runReaderStage(context.Background(), config, catalog, document)
	var stageErr *providerStageError
	if !errors.As(err, &stageErr) {
		t.Fatalf("partial reader source-read failure = %T %v", err, err)
	}
	if stageErr.reason != reportexecution.ProviderFailureReasonSemanticValidation || providerValidationCode(stageErr.cause) != reportexecution.ProviderValidationCodeSourceReadContract {
		t.Fatalf("partial reader source-read reason=%q code=%q err=%v", stageErr.reason, providerValidationCode(stageErr.cause), err)
	}
	if len(attempts) != 2 || len(provider.requests) != 2 || verifier.calls != 2 || provider.requests[0].ToolSessionID == provider.requests[1].ToolSessionID {
		t.Fatalf("partial reader source-read attempts=%d requests=%d verifier=%d", len(attempts), len(provider.requests), verifier.calls)
	}
	if !strings.Contains(provider.requests[1].Prompt, "remaining_sources is zero") || strings.Contains(provider.requests[1].Prompt, "fixture source") {
		t.Fatalf("reader source repair prompt is unsafe or incomplete: %s", provider.requests[1].Prompt)
	}
}

func TestProviderTableDraftSupportsTwoToFourFixedWidthColumns(t *testing.T) {
	role := "evidence"
	narrative := Narrative{
		SchemaVersion: NarrativeSchemaVersion, ContractID: "narrative_table", DocumentID: "document_table",
		CentralQuestion: "What?", ReaderTakeaway: "A table.", Throughline: "Table.", ReaderJourney: []string{"Read"}, ArgumentArc: []string{"Compare"},
		SectionRoles: []SectionRole{{SectionID: "section.table", Role: "evidence", QuestionAnswered: "What?"}}, ConclusionObligations: []string{"Conclude"}, VoiceAndTone: "Direct",
	}
	for width := 2; width <= 4; width++ {
		t.Run(strconv.Itoa(width), func(t *testing.T) {
			third, fourth := "Third", "Fourth"
			cell3, cell4 := "C", "D"
			table := &documentTableDraft{Column1: "First", Column2: "Second", Rows: []documentTableRowDraft{{Cell1: "A", Cell2: "B"}}}
			if width >= 3 {
				table.Column3, table.Rows[0].Cell3 = &third, &cell3
			}
			if width == 4 {
				table.Column4, table.Rows[0].Cell4 = &fourth, &cell4
			}
			draft := documentDraft{Language: "en", Sections: map[string]documentSectionDraft{
				"section.table": {Title: "Table", Blocks: []documentBlockDraft{{Kind: "table", Table: table, EvidenceSourceKeys: []string{"source_001"}, SemanticRole: &role}}},
			}}
			catalog := testSourceCatalog(t, "mis_fixture")
			document, err := compileDocumentDraft("Table", narrative, catalog, testSourceReadReceipt(catalog), draft)
			if err != nil {
				t.Fatal(err)
			}
			compiled := document.Blocks[1].Table
			if compiled == nil || len(compiled.Columns) != width || len(compiled.Rows[0]) != width {
				t.Fatalf("compiled %d-column table = %#v", width, compiled)
			}
		})
	}
	fourth, cell4 := "Fourth", "D"
	invalid := documentDraft{Language: "en", Sections: map[string]documentSectionDraft{
		"section.table": {Title: "Table", Blocks: []documentBlockDraft{{Kind: "table", Table: &documentTableDraft{Column1: "First", Column2: "Second", Column4: &fourth, Rows: []documentTableRowDraft{{Cell1: "A", Cell2: "B", Cell4: &cell4}}}, EvidenceSourceKeys: []string{"source_001"}}}},
	}}
	catalog := testSourceCatalog(t, "mis_fixture")
	if _, err := compileDocumentDraft("Table", narrative, catalog, testSourceReadReceipt(catalog), invalid); err == nil {
		t.Fatal("gapped fixed-width table accepted")
	}
}

func schemaAllowsNull(value any) bool {
	node, ok := value.(map[string]any)
	if !ok {
		return false
	}
	choices, ok := node["anyOf"].([]any)
	if !ok {
		return false
	}
	for _, choice := range choices {
		entry, ok := choice.(map[string]any)
		if ok && entry["type"] == "null" {
			return true
		}
	}
	return false
}

func TestProductStagesAttachExactStructuredOutputSchemas(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_fixture")
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: "doc_1", RevisionID: "draft", NarrativeContractID: "narr_1",
		Title: "Title", Language: "en",
		Blocks: []Block{
			{NodeID: "section_1", Kind: "section", Level: 2, Title: "Opening"},
			{NodeID: "node_1", ParentNodeID: "section_1", Kind: "prose", Prose: "Complete manuscript.", EvidenceRefs: []string{"ref.accepted_source.test"}},
			{NodeID: "section_2", Kind: "section", Level: 2, Title: "Conclusion"},
			{NodeID: "node_2", ParentNodeID: "section_2", Kind: "prose", Prose: "The conclusion."},
		},
		References: []Reference{{RefID: "ref.accepted_source.test", Kind: "footnote", Target: "accepted-source:001", VisibleLabel: "Accepted source 1", Locator: "Frozen source catalog"}},
		Provenance: map[string]string{"source": "test"},
	}
	t.Run("source-aware author", func(t *testing.T) {
		provider := &recordingProvider{errs: []error{errors.New("stop after request capture")}}
		config := testSourceProductConfig(provider, &acceptingSourceReadVerifier{})
		config.MissionObjective = "Explain the subject directly."
		if _, _, _, err := runAuthorStage(context.Background(), config, catalog); err == nil || len(provider.requests) != 1 {
			t.Fatalf("author request capture: requests=%d err=%v", len(provider.requests), err)
		}
		request := provider.requests[0]
		if got, want := request.OutputJSONSchema, providerAuthorSchemaBytes(catalog); string(got) != string(want) {
			t.Fatalf("author schema mismatch\ngot:  %s\nwant: %s", got, want)
		}
		if request.ReportILSources == nil || request.ReportILSources.Catalog.SHA256 != catalog.SHA256 || !reflect.DeepEqual(request.ExtraMCPTools, []string{reportilcontract.SourceListTool, reportilcontract.SourceReadTool, reportilcontract.SourceQuoteRegisterTool}) || request.DisableTools {
			t.Fatalf("author source-tool request = %#v", request)
		}
		for _, required := range []string{config.MissionObjective, catalog.SHA256, reportilcontract.SourceListTool, reportilcontract.SourceReadTool, reportilcontract.SourceQuoteRegisterTool, "complete report a real reader asked for", "SERVER-CHECKED MANUSCRIPT BOUNDARIES", "repeated keys in one content leaf as one citation", "do not mechanically pronounce source-script characters", "official designation", "terminology inventory only", "source_reading only when", "foreign unit", "server validates that sentence directly", "explicitly named subject", "remaining_sources is zero, then stop", "language_review"} {
			if !strings.Contains(request.Prompt, required) {
				t.Fatalf("author prompt lacks %q: %s", required, request.Prompt)
			}
		}
		for _, forbidden := range []string{"fixture source content", "src_fixture", "art_fixture", catalog.Sources[0].ReadableSHA256} {
			if strings.Contains(request.Prompt, forbidden) {
				t.Fatalf("author prompt leaked %q: %s", forbidden, request.Prompt)
			}
		}
		if strings.Contains(string(request.OutputJSONSchema), `"first_use_explanation"`) {
			t.Fatalf("author schema retains redundant first-use explanation metadata: %s", request.OutputJSONSchema)
		}
	})
	t.Run("internal working title is not reader title guidance", func(t *testing.T) {
		provider := &recordingProvider{errs: []error{errors.New("stop after request capture")}}
		config := testSourceProductConfig(provider, &acceptingSourceReadVerifier{})
		config.Title = "다케다성 Experimental IL 리포트"
		config.MissionObjective = "다케다성이 어떤 곳이고 왜 중요한지 설명한다."
		if _, _, _, err := runAuthorStage(context.Background(), config, catalog); err == nil || len(provider.requests) != 1 {
			t.Fatalf("author request capture: requests=%d err=%v", len(provider.requests), err)
		}
		prompt := provider.requests[0].Prompt
		if strings.Contains(prompt, config.Title) || !strings.Contains(prompt, "Choose a natural reader-facing title") {
			t.Fatalf("internal working title reached reader title guidance: %s", prompt)
		}
	})
	t.Run("source-aware reader", func(t *testing.T) {
		provider := &recordingProvider{errs: []error{errors.New("stop after request capture")}}
		config := testSourceProductConfig(provider, &acceptingSourceReadVerifier{})
		config.MissionObjective = "Explain the subject directly."
		config.Direction = "Name Ikuno Silver Mine and explain its relationship to the castle."
		if _, _, err := runReaderStage(context.Background(), config, catalog, document); err == nil || len(provider.requests) != 1 {
			t.Fatalf("reader request capture: requests=%d err=%v", len(provider.requests), err)
		}
		request := provider.requests[0]
		if got, want := request.OutputJSONSchema, providerReaderSchemaBytes(document); string(got) != string(want) {
			t.Fatalf("reader schema mismatch\ngot:  %s\nwant: %s", got, want)
		}
		if request.ReportILSources == nil || request.ReportILSources.Catalog.SHA256 != catalog.SHA256 || request.ReportILSources.Stage != "il_flow" || request.ReportILSources.MaxReadBytes != reportilcontract.DefaultSourceAttemptReadBytes || request.ReportILSources.MaxCallBytes != reportilcontract.DefaultSourceReadMaxBytes || request.ReportILSources.MaxSourceReadBytes != 0 || !reflect.DeepEqual(request.ExtraMCPTools, []string{reportilcontract.SourceListTool, reportilcontract.SourceReadTool}) || request.DisableTools || request.MCPMode != "source_read_only" || request.CapabilityProfile != agentcapability.ProfileReportILSourceV1 {
			t.Fatalf("reader request is not source-aware = %#v", request)
		}
		for _, required := range []string{
			config.MissionObjective, config.Direction, catalog.SHA256,
			"[section_01] Opening", "[block_01 kind=prose]\nComplete manuscript.",
			"[section_02] Conclusion", "Treat section_02 as the final conclusion section",
			"conclusion_not_recap means section_02", "section_01 for section_01",
			"mechanical translation", "SOURCE-SAFE TERMINOLOGY DECISIONS",
			"retain_as_terminology", "present_source_aliases", "low_value_source_aliases_absent",
			"server derives that first-use sentence", "ordinary SI measurements",
			"explicitly named subject", "equals, contains, or is contained",
			"opening must not be followed by a second paragraph", "language_review",
			"direct source statement", "bounded inference", "unsupported operational or causal claim",
			"A request to explain a relationship does not prove direct operation",
			"Visibility alone does not support saying that movement was watched",
			"source review is private work", "ordinary units",
			"Style editing must never increase claim strength", "claim_strength_bounded",
			"subject_first_answer_present", "audit_record_prose_absent",
			"low_value_si_explanations_absent", "unfamiliar_terms_explained",
			"preservation_coverage",
			"self-contained general report",
			"historical route", "naturalized or transliterated",
			reportilcontract.SourceListTool, reportilcontract.SourceReadTool,
			"remaining_sources is zero", "Read every bound span",
		} {
			if !strings.Contains(request.Prompt, required) {
				t.Fatalf("reader prompt lacks %q: %s", required, request.Prompt)
			}
		}
		for _, forbidden := range []string{"fixture source content", "src_fixture", "art_fixture", catalog.Sources[0].ReadableSHA256, "Accepted source 1", "<sup>", "section_1", "node_1"} {
			if strings.Contains(request.Prompt, forbidden) {
				t.Fatalf("reader prompt leaked %q: %s", forbidden, request.Prompt)
			}
		}
		for _, field := range []string{"terminology_edits", "language_review", "reader_review", "target_language_natural", "names_and_terms_preserved", "mechanical_translation_absent", "opening_not_duplicated", "unsupported_pronunciation_absent", "low_value_source_aliases_absent", "conclusion_not_recap", "claim_strength_bounded", "subject_first_answer_present", "audit_record_prose_absent", "low_value_si_explanations_absent", "unfamiliar_terms_explained"} {
			if !strings.Contains(string(request.OutputJSONSchema), `"`+field+`"`) {
				t.Fatalf("reader schema lacks %q: %s", field, request.OutputJSONSchema)
			}
		}
		if strings.Contains(string(request.OutputJSONSchema), `"unfamiliar_terms_explained":{"type":"boolean","const":true}`) {
			t.Fatalf("unfamiliar-term review must allow false so typed repair can run: %s", request.OutputJSONSchema)
		}
		for _, removedField := range []string{"disposition", "first_use_explanation"} {
			if strings.Contains(string(request.OutputJSONSchema), `"`+removedField+`"`) {
				t.Fatalf("reader schema retains redundant %q: %s", removedField, request.OutputJSONSchema)
			}
		}
	})
}

func TestServerAuthorSectionIDIsStableAcrossReaderFacingTitleEdits(t *testing.T) {
	first := serverAuthorSectionID("doc_stable", 2, "Original title", map[string]bool{})
	second := serverAuthorSectionID("doc_stable", 2, "Edited title", map[string]bool{})
	if first != second {
		t.Fatalf("reader-facing title changed server topology: %q != %q", first, second)
	}
}

func TestReaderSchemaUsesOrderedAliasesInsteadOfOpaqueTopologyIDs(t *testing.T) {
	document := Document{
		Blocks: []Block{
			{NodeID: "section.z", Kind: "section", Level: 2, Title: "Opening"},
			{NodeID: "node.z", ParentNodeID: "section.z", Kind: "prose", Prose: "Opening prose."},
			{NodeID: "section.a", Kind: "section", Level: 2, Title: "Conclusion"},
			{NodeID: "node.a", ParentNodeID: "section.a", Kind: "prose", Prose: "Conclusion prose."},
		},
	}
	schema := string(providerReaderSchemaBytes(document))
	for _, required := range []string{`"section_01"`, `"section_02"`, `"block_01"`} {
		if !strings.Contains(schema, required) {
			t.Fatalf("reader schema lacks ordered alias %s: %s", required, schema)
		}
	}
	for _, forbidden := range []string{`"section.z"`, `"section.a"`, `"node.z"`, `"node.a"`} {
		if strings.Contains(schema, forbidden) {
			t.Fatalf("reader schema exposes opaque topology key %s: %s", forbidden, schema)
		}
	}
	prompt := readerPrompt(ProductConfig{MissionObjective: "Explain the subject."}, document)
	if strings.Index(prompt, "[section_01] Opening") >= strings.Index(prompt, "[section_02] Conclusion") {
		t.Fatalf("ordered manuscript aliases do not follow document order: %s", prompt)
	}
	for _, required := range []string{
		"Treat section_02 as the final conclusion section",
		"Do not recap the report by listing, naming, or walking back through the subjects of earlier sections",
		"conclusion_not_recap means section_02",
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("reader prompt lacks final-section obligation %q: %s", required, prompt)
		}
	}
}

func acceptedLanguageReview() languageReviewDraft {
	return languageReviewDraft{
		TargetLanguageNatural:       true,
		NamesAndTermsPreserved:      true,
		MechanicalTranslationAbsent: true,
		OpeningNotDuplicated:        true,
	}
}

func acceptedReaderReview() readerReviewDraft {
	return readerReviewDraft{
		UnsupportedPronunciationAbsent: true,
		LowValueSourceAliasesAbsent:    true,
		ConclusionNotRecap:             true,
		ClaimStrengthBounded:           true,
		SubjectFirstAnswerPresent:      true,
		AuditRecordProseAbsent:         true,
		LowValueSIExplanationsAbsent:   true,
		UnfamiliarTermsExplained:       true,
	}
}

func TestReaderEvidenceSupportRepairGuidanceIsGranularTargetedAndContentFree(t *testing.T) {
	cases := map[reportexecution.ProviderValidationCode][]string{
		reportexecution.ProviderValidationCodeEvidenceSupportInventory:   {"complete inventory", "include evidence_002"},
		reportexecution.ProviderValidationCodeEvidenceSupportReceipt:     {"evidence_002", "exact evidence_receipt"},
		reportexecution.ProviderValidationCodeEvidenceSupportTarget:      {"evidence_002", "exact schema-bound section_key and block_key"},
		reportexecution.ProviderValidationCodeEvidenceSupportBinding:     {"evidence_002", "original packet node"},
		reportexecution.ProviderValidationCodeEvidenceSupportQuote:       {"evidence_002", "nonempty coverage_quote", "stale pre-edit quote"},
		reportexecution.ProviderValidationCodeEvidenceSupportUnsupported: {"evidence_002", "coverage_quote null", "preserve=true"},
		reportexecution.ProviderValidationCodeEvidenceSupportLevel:       {"evidence_002", "direct_source_statement", "bounded_inference", "unsupported_removed"},
		reportexecution.ProviderValidationCodeEvidenceSupportCoverage:    {"evidence_002", "union of all supported exact coverage_quote values", "table data cell"},
	}
	for code, required := range cases {
		prompt := sourceRepairPrompt(
			"ORIGINAL", "il_flow", "semantic_validation", "", code, "",
			"evidence_002", "claim from source_001", "evidence_002", "PRIVATE VALIDATOR DETAIL",
		)
		for _, value := range append(required, "Start a fresh source-tool session") {
			if !strings.Contains(prompt, value) {
				t.Fatalf("reader repair %q lacks %q: %s", code, value, prompt)
			}
		}
		if strings.Count(prompt, "evidence_002") != 1 {
			t.Fatalf("reader repair %q did not deduplicate alias: %s", code, prompt)
		}
		for _, forbidden := range []string{
			"claim from source_001", "source_001", "PRIVATE VALIDATOR DETAIL",
			"exact authored claim to check", "source_excerpt", "https://private.example", testPrivateUserPath("alice"),
		} {
			if strings.Contains(prompt, forbidden) {
				t.Fatalf("reader repair %q leaked %q: %s", code, forbidden, prompt)
			}
		}
		cause := withValidationCodeAndPackets(code, []string{"evidence_002"}, errors.New("PRIVATE VALIDATOR DETAIL"))
		if !reflect.DeepEqual(providerValidationPacketAliases(cause), []string{"evidence_002"}) {
			t.Fatalf("reader repair %q lost request-local alias", code)
		}
		failure := providerStageFailure("il_flow", &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  cause,
		}, TerminalUsageReceipt{})
		request := failure.AppendRequest(
			"mis_safe", "evt_pending", "evt_terminal", ledger.Producer{Type: "agent", ID: "codex"},
		)
		encoded := string(request.Payload)
		if !strings.Contains(encoded, string(code)) || strings.Contains(encoded, "evidence_002") || strings.Contains(encoded, "PRIVATE VALIDATOR DETAIL") {
			t.Fatalf("durable reader failure leaked request-local detail: %s", encoded)
		}
	}
}

func TestReaderUnexplainedTermRepairGuidanceIsSpecificAndContentFree(t *testing.T) {
	guidance := readerRepairGuidance(reportexecution.ProviderValidationCodeReaderUnexplainedTerm)
	for _, required := range []string{"historical route", "naturalized or transliterated", "first body sentence", "unfamiliar_terms_explained"} {
		if !strings.Contains(guidance, required) {
			t.Fatalf("unexplained-term repair lacks %q: %s", required, guidance)
		}
	}
	for _, forbidden := range []string{"산인도", "山陰道", "source_001"} {
		if strings.Contains(guidance, forbidden) {
			t.Fatalf("unexplained-term repair leaked %q: %s", forbidden, guidance)
		}
	}
}

func TestReaderEditPreservesTopologyAndEvidence(t *testing.T) {
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: "doc_reader", RevisionID: "draft", NarrativeContractID: "narr_reader",
		Title: "Original title", Language: "en",
		Blocks: []Block{
			{NodeID: "section.z_opening", Kind: "section", Level: 2, Title: "Original opening"},
			{NodeID: "node.z_opening", ParentNodeID: "section.z_opening", Kind: "prose", Prose: "Original fact.", EvidenceRefs: []string{"ref.accepted_source.reader"}},
			{NodeID: "section.a_conclusion", Kind: "section", Level: 2, Title: "Original conclusion"},
			{NodeID: "node.a_conclusion", ParentNodeID: "section.a_conclusion", Kind: "prose", Prose: "Original conclusion."},
		},
		References: []Reference{{RefID: "ref.accepted_source.reader", Kind: "footnote", Target: "accepted-source:001", VisibleLabel: "Accepted source 1", Locator: "Frozen source catalog"}},
		Provenance: map[string]string{"source": "accepted mission sources"},
	}
	opening, conclusion := "Clear fact.", "Sharper conclusion."
	draft := readerDraft{
		Title: "Clear title", Reviewer: "reader", LanguageReview: acceptedLanguageReview(), ReaderReview: acceptedReaderReview(),
		SectionEdits: map[string]readerSectionDraft{}, TerminologyEdits: map[string]readerTermDraft{},
	}
	sections := readerSections(document)
	draft.SectionEdits["section_01"] = readerSectionDraft{
		SectionReceipt: readerSectionReceipt(sections[0]),
		Title:          "Clear opening",
		Blocks: map[string]readerBlockDraft{
			"block_01": {OriginalSHA256: readerBlockSHA256(document.Blocks[1]), Prose: &opening},
		},
	}
	draft.SectionEdits["section_02"] = readerSectionDraft{
		SectionReceipt: readerSectionReceipt(sections[1]),
		Title:          "Clear conclusion",
		Blocks: map[string]readerBlockDraft{
			"block_01": {OriginalSHA256: readerBlockSHA256(document.Blocks[3]), Prose: &conclusion},
		},
	}
	edited, attestation, err := compileReaderDraft(document, draft, "Explain the subject directly.")
	if err != nil {
		t.Fatal(err)
	}
	if edited.Title != draft.Title || edited.Blocks[1].Prose != opening || edited.Blocks[3].Prose != conclusion || attestation.Verdict != "accept" {
		t.Fatalf("reader edit did not apply: %#v/%#v", edited, attestation)
	}
	if !reflect.DeepEqual(edited.Blocks[1].EvidenceRefs, document.Blocks[1].EvidenceRefs) || !reflect.DeepEqual(edited.References, document.References) || edited.Blocks[1].NodeID != document.Blocks[1].NodeID || edited.Blocks[1].ParentNodeID != document.Blocks[1].ParentNodeID {
		t.Fatalf("reader edit changed topology or evidence: %#v", edited)
	}
	if edited.Blocks[0].Title != "Clear opening" || edited.Blocks[1].Prose != opening || edited.Blocks[2].Title != "Clear conclusion" || edited.Blocks[3].Prose != conclusion {
		t.Fatalf("ordered aliases reassigned reader edits: %#v", edited.Blocks)
	}
	swapped := draft
	swapped.SectionEdits = map[string]readerSectionDraft{
		"section_01": draft.SectionEdits["section_02"],
		"section_02": draft.SectionEdits["section_01"],
	}
	if _, _, err := compileReaderDraft(document, swapped, "Explain the subject directly."); err == nil {
		t.Fatal("reader section reassignment passed original-title binding")
	}
	unchecked := draft
	unchecked.LanguageReview.OpeningNotDuplicated = false
	if _, _, err := compileReaderDraft(document, unchecked, "Explain the subject directly."); err == nil || !strings.Contains(err.Error(), "language review") {
		t.Fatalf("reader accepted a failed language review: %v", err)
	}
	unsupportedPronunciation := draft
	unsupportedPronunciation.ReaderReview.UnsupportedPronunciationAbsent = false
	if _, _, err := compileReaderDraft(document, unsupportedPronunciation, "Explain the subject directly."); err == nil || !strings.Contains(err.Error(), "reader review") {
		t.Fatalf("reader accepted unsupported pronunciation attestation: %v", err)
	}
	lowValueAliases := draft
	lowValueAliases.ReaderReview.LowValueSourceAliasesAbsent = false
	if _, _, err := compileReaderDraft(document, lowValueAliases, "Explain the subject directly."); err == nil || !strings.Contains(err.Error(), "reader review") {
		t.Fatalf("reader accepted low-value source aliases attestation: %v", err)
	}
	conclusionRecap := draft
	conclusionRecap.ReaderReview.ConclusionNotRecap = false
	if _, _, err := compileReaderDraft(document, conclusionRecap, "Explain the subject directly."); err == nil || !strings.Contains(err.Error(), "reader review") {
		t.Fatalf("reader accepted conclusion recap attestation: %v", err)
	}
	unboundedClaims := draft
	unboundedClaims.ReaderReview.ClaimStrengthBounded = false
	if _, _, err := compileReaderDraft(document, unboundedClaims, "Explain the subject directly."); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeClaimStrength {
		t.Fatalf("reader accepted unbounded claim-strength attestation: code=%q err=%v", providerValidationCode(err), err)
	}
	auditVoice := draft
	auditVoice.ReaderReview.AuditRecordProseAbsent = false
	if _, _, err := compileReaderDraft(document, auditVoice, "Explain the subject directly."); err == nil || !strings.Contains(err.Error(), "reader review") {
		t.Fatalf("reader accepted audit-record prose attestation: %v", err)
	}
	unexplainedTerm := draft
	unexplainedTerm.ReaderReview.UnfamiliarTermsExplained = false
	if _, _, err := compileReaderDraft(document, unexplainedTerm, "Explain the subject directly."); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeReaderUnexplainedTerm {
		t.Fatalf("reader accepted unexplained-term attestation: code=%q err=%v", providerValidationCode(err), err)
	}
	blockSwapped := draft
	blockSwapped.SectionEdits = map[string]readerSectionDraft{}
	for key, section := range draft.SectionEdits {
		blockSwapped.SectionEdits[key] = section
	}
	openingEdit := blockSwapped.SectionEdits["section_01"]
	openingEdit.Blocks = map[string]readerBlockDraft{"block_01": draft.SectionEdits["section_02"].Blocks["block_01"]}
	blockSwapped.SectionEdits["section_01"] = openingEdit
	if _, _, err := compileReaderDraft(document, blockSwapped, "Explain the subject directly."); err == nil {
		t.Fatal("reader block reassignment passed original-content binding")
	}
	mutated := edited
	mutated.Blocks = append([]Block(nil), edited.Blocks...)
	mutated.Blocks[1].EvidenceRefs = nil
	if err := validateReaderDocumentStructure(document, mutated); err == nil {
		t.Fatal("reader evidence mutation was accepted")
	}
}

func TestFinalizeNarrativeUsesAcceptedReaderDocument(t *testing.T) {
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: "doc_final_narrative", RevisionID: "draft", NarrativeContractID: "narr_final_narrative",
		Title: "Takeda Castle", Language: "en",
		Blocks: []Block{
			{NodeID: "section.opening", Kind: "section", Level: 2, Title: "Why the castle matters"},
			{NodeID: "node.opening", ParentNodeID: "section.opening", Kind: "prose", Prose: "Takeda Castle concentrated regional power in a mountain stronghold."},
			{NodeID: "section.evidence", Kind: "section", Level: 2, Title: "What survives"},
			{NodeID: "node.evidence", ParentNodeID: "section.evidence", Kind: "prose", Prose: "Stone walls preserve its defensive form."},
			{NodeID: "section.conclusion", Kind: "section", Level: 2, Title: "Historical judgment"},
			{NodeID: "node.conclusion", ParentNodeID: "section.conclusion", Kind: "prose", Prose: "Its importance lies in how terrain and rule became one historical landscape."},
		},
		Provenance: map[string]string{"source": "accepted mission sources"},
	}
	narrative := Narrative{
		SchemaVersion: NarrativeSchemaVersion, ContractID: document.NarrativeContractID, DocumentID: document.DocumentID,
		CentralQuestion: "Stale question", ReaderTakeaway: "The castle monitored mine traffic.", Throughline: "The castle monitored mine traffic.",
		ReaderJourney: []string{"Old opening", "Old evidence", "Old conclusion"}, ArgumentArc: []string{"Old opening", "Old evidence", "Old conclusion"},
		SectionRoles: []SectionRole{
			{SectionID: "section.opening", Role: "opening", QuestionAnswered: "Old opening"},
			{SectionID: "section.evidence", Role: "evidence", QuestionAnswered: "Old evidence"},
			{SectionID: "section.conclusion", Role: "conclusion", QuestionAnswered: "Old conclusion"},
		},
		ConclusionObligations: []string{"The castle monitored mine traffic."}, VoiceAndTone: "Direct historical explanation",
	}
	finalized, err := finalizeNarrativeFromDocument(narrative, document, "Explain Takeda Castle's historical importance.", nil)
	if err != nil {
		t.Fatal(err)
	}
	if finalized.CentralQuestion != "Explain Takeda Castle's historical importance." ||
		finalized.Throughline != document.Blocks[1].Prose ||
		finalized.ReaderTakeaway != document.Blocks[5].Prose ||
		!reflect.DeepEqual(finalized.ReaderJourney, []string{"Why the castle matters", "What survives", "Historical judgment"}) ||
		len(finalized.ConclusionObligations) != 1 || finalized.ConclusionObligations[0] != document.Blocks[5].Prose ||
		strings.Contains(string(mustMarshal(finalized)), "monitored mine traffic") {
		t.Fatalf("final Narrative retained stale author claims: %#v", finalized)
	}
	if len(finalized.SectionRoles) != 3 || finalized.SectionRoles[0].Role != "opening" || finalized.SectionRoles[2].Role != "conclusion" || finalized.SectionRoles[1].QuestionAnswered != "What survives" {
		t.Fatalf("final Narrative section binding = %#v", finalized.SectionRoles)
	}
}

func TestReaderEditPreservesListAndTableInventory(t *testing.T) {
	list := Block{Kind: "list", Items: []string{"First fact", "Second fact"}}
	if _, err := applyReaderBlockEdit(list, readerBlockDraft{Items: []string{"Only one fact"}}); err == nil {
		t.Fatal("reader removed a list item")
	}
	table := Block{Kind: "table", Table: &Table{Columns: []string{"Claim", "Value"}, Rows: [][]string{{"A", "1"}, {"B", "2"}}}}
	caption := "Edited table"
	edit := readerBlockDraft{Table: &documentTableDraft{
		Caption: &caption, Column1: "Claim", Column2: "Value",
		Rows: []documentTableRowDraft{{Cell1: "A", Cell2: "1"}},
	}}
	if _, err := applyReaderBlockEdit(table, edit); err == nil {
		t.Fatal("reader removed a table row")
	}
}

func TestReaderFacingValidationAllowsILOnlyWhenItIsTheMissionSubject(t *testing.T) {
	document := Document{Title: "Semantic IL architecture", Blocks: []Block{{Kind: "prose", Prose: "Semantic IL compiles one manuscript into several formats."}}}
	if err := validateReaderFacingDocument(document, "Explain Takeda Castle history."); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeReaderInternalMachinery {
		t.Fatalf("unrelated mission accepted internal report machinery: code=%q err=%v", providerValidationCode(err), err)
	}
	if err := validateReaderFacingDocument(document, "Evaluate semantic IL as an intermediate representation for reports."); err != nil {
		t.Fatalf("IL mission subject was rejected: %v", err)
	}
	subjectCaveat := Document{Title: "다케다성", Blocks: []Block{{Kind: "prose", Prose: "축성 시기와 폐성 과정에는 기록이 엇갈려, 판정 기준에 따라 해석이 달라질 수 있다."}}}
	if err := validateReaderFacingDocument(subjectCaveat, "일본 다케다성"); err != nil {
		t.Fatalf("reader-facing subject caveat was rejected: %v", err)
	}
	process := Document{Title: "Direct answer", Blocks: []Block{{Kind: "prose", Prose: "이 보고서에서는 사실을 설명한다."}, {Kind: "prose", Prose: "다음 섹션에서는 결론을 살펴본다."}}}
	if err := validateReaderFacingDocument(process, "Explain the subject directly."); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeReaderProcess {
		t.Fatalf("repeated report-process narration was accepted: code=%q err=%v", providerValidationCode(err), err)
	}
	for _, prose := range []string{
		"중요한 것은 무엇이 먼저 검토되었고, 그 검토가 다음 판단을 어떻게 열었는지를 시간의 흐름 속에서 드러내는 일이다.",
		"문제는 그 흐름이 한 사례를 대표 사례처럼 느끼게 만들 때, 독자가 따로 확인해야 할 질문까지 대신 답해 버린다는 데 있다.",
	} {
		subjectProse := Document{Title: "Direct answer", Blocks: []Block{{Kind: "prose", Prose: prose}}}
		if err := validateReaderFacingDocument(subjectProse, "Explain the subject directly."); err != nil {
			t.Fatalf("subject prose was mistaken for report-process narration: %q: %v", prose, err)
		}
	}
	auditOpening := Document{Title: "다케다성", Blocks: []Block{{Kind: "prose", Prose: "자료가 직접 보여 주는 답은 석축이 남아 있다는 점이다."}, {Kind: "prose", Prose: "다케다성은 산악 지역의 성곽이었다."}}}
	if err := validateReaderFacingDocument(auditOpening, "일본 다케다성"); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeReaderOpening {
		t.Fatalf("source-audit opening was accepted: code=%q err=%v", providerValidationCode(err), err)
	}
	repeatedAudit := Document{Title: "다케다성", Blocks: []Block{{Kind: "prose", Prose: "다케다성은 산 정상의 군사 거점이었다."}, {Kind: "prose", Prose: "자료는 병력 배치를 설명하지 않는다."}, {Kind: "prose", Prose: "기록으로 관리 체계를 확인할 수 없다."}, {Kind: "prose", Prose: "근거만으로 통행 통제를 단정할 수 없다."}}}
	if err := validateReaderFacingDocument(repeatedAudit, "일본 다케다성"); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeReaderAuditVoice {
		t.Fatalf("repeated source-audit voice was accepted: code=%q err=%v", providerValidationCode(err), err)
	}
	twoBoundaries := Document{Title: "다케다성", Blocks: []Block{
		{Kind: "prose", Prose: "다케다성은 산 정상의 군사 거점이었다."},
		{Kind: "prose", Prose: "축성 시기는 기록이 엇갈려 하나의 연대로 단정할 수 없다."},
		{Kind: "prose", Prose: "현재 기록만으로 성이 은광 운송을 직접 통제했다고 단정할 수는 없다."},
		{Kind: "prose", Prose: "두 한계는 서로 다른 역사적 질문을 제한한다."},
	}}
	if err := validateReaderFacingDocument(twoBoundaries, "일본 다케다성"); err != nil {
		t.Fatalf("two distinct nearby evidence boundaries were rejected: %v", err)
	}
	lowValueSI := Document{Title: "다케다성", Blocks: []Block{{Kind: "prose", Prose: "다케다성은 해발 353미터, 곧 바닷물 높이를 기준으로 353미터 높은 곳에 자리한다."}}}
	if err := validateReaderFacingDocument(lowValueSI, "일본 다케다성"); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeReaderOrdinarySI {
		t.Fatalf("low-value SI explanation was accepted: code=%q err=%v", providerValidationCode(err), err)
	}
	for _, prose := range []string{
		"석벽은 노づ라즈미 방식으로 쌓았다.",
		"석벽은 노ヅ라즈미 방식으로 쌓았다.",
	} {
		mixedScript := Document{Title: "다케다성", Language: "ko", Blocks: []Block{{Kind: "prose", Prose: prose}}}
		if err := validateReaderFacingDocument(mixedScript, "일본 다케다성"); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeReaderFacingContent {
			t.Fatalf("mixed-script lexical form was accepted: code=%q err=%v", providerValidationCode(err), err)
		}
	}
	brokenName := Document{Title: "다케다성", Language: "ko", Blocks: []Block{{Kind: "prose", Prose: "히로히데는 조선 유학자 姜こう에게 가르침을 청했다."}}}
	if err := validateReaderFacingDocument(brokenName, "일본 다케다성"); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeReaderFacingContent {
		t.Fatalf("broken mixed-script name was accepted: code=%q err=%v", providerValidationCode(err), err)
	}
	explainedSourceForm := Document{Title: "다케다성", Language: "ko", Blocks: []Block{{Kind: "prose", Prose: "히로히데는 돗토리의 사찰인 진쿄지(真教寺)에서 자결했다."}}}
	if err := validateReaderFacingDocument(explainedSourceForm, "일본 다케다성"); err != nil {
		t.Fatalf("explained proper-noun source form was rejected: %v", err)
	}
	for _, prose := range []string{
		"성 안 곳곳에 마스가타 호구(枡形虎口)와 어긋난 호구(食い違い虎口)가 놓였다.",
		"석벽에는 노면쌓기(野面積み)가 사용되었고 천수대 모서리에는 산기쌓기(算木積み)가 남아 있다.",
		"산기슭의 사찰마을길(寺町通り)은 폐성 뒤의 기억을 이어 간다.",
	} {
		explainedTerm := Document{Title: "다케다성", Language: "ko", Blocks: []Block{{Kind: "prose", Prose: prose}}}
		if err := validateReaderFacingDocument(explainedTerm, "일본 다케다성"); err != nil {
			t.Fatalf("explained Japanese source term was rejected: %q: %v", prose, err)
		}
	}
}

func TestPublicationCompilerRejectsBrokenKoreanMixedScriptName(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_publication_korean_script")
	baseAuthor := reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.AuthorDocumentSchemaVersion,
		Title:         "다케다성", Language: "ko",
		Sections: []reportilcontract.AuthorSection{
			{SectionKey: "section_001", Title: "통치의 성격", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "section_001.block_001", Kind: "prose", Prose: "히로히데는 조선 유학자에게 가르침을 청했다.", EvidenceSourceKeys: []string{"source_001"}}}},
			{SectionKey: "section_002", Title: "역사적 판단", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "section_002.block_001", Kind: "prose", Prose: "학문 후원은 통치의 범위를 넓혔다.", EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
	draft := reportFirstDraftFromAuthorDocument(baseAuthor)
	_, original, err := compileReportFirstAuthorDraftForMode(
		draft, AuthoringModeStandard, "narrative_publication_korean_script", "doc_publication_korean_script",
		"일본 다케다성", catalog, editorialMemorySourceReadReceipt(catalog), nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	broken := baseAuthor
	broken.Sections = append([]reportilcontract.AuthorSection(nil), baseAuthor.Sections...)
	broken.Sections[0].Blocks = append([]reportilcontract.AuthorBlock(nil), baseAuthor.Sections[0].Blocks...)
	broken.Sections[0].Blocks[0].Prose = "히로히데는 조선 유학자 姜こう에게 가르침을 청했다."
	_, err = compilePublicationAuthorDocument(broken, original, AuthoringModeStandard, "일본 다케다성", catalog, nil)
	if err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeReaderFacingContent {
		t.Fatalf("publication compiler accepted broken mixed-script name: code=%q err=%v", providerValidationCode(err), err)
	}
}

func TestReaderStageRepairUsesSpecificReaderFacingCode(t *testing.T) {
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: "doc_reader_code", RevisionID: "draft", NarrativeContractID: "narrative_reader_code",
		Title: "다케다성", Language: "ko",
		Blocks: []Block{
			{NodeID: "section.opening", Kind: "section", Level: 2, Title: "역사적 성격"},
			{NodeID: "node.opening", ParentNodeID: "section.opening", Kind: "prose", Prose: "다케다성은 산악 지역의 성곽이었다."},
			{NodeID: "section.conclusion", Kind: "section", Level: 2, Title: "역사적 판단"},
			{NodeID: "node.conclusion", ParentNodeID: "section.conclusion", Kind: "prose", Prose: "입지가 성의 역사적 성격을 만들었다."},
		},
		Provenance: map[string]string{"source": "accepted mission sources"},
	}
	makeDraft := func(opening string) readerDraft {
		draft := readerDraft{Title: document.Title, Reviewer: "whole-reader", LanguageReview: acceptedLanguageReview(), ReaderReview: acceptedReaderReview(), SectionEdits: map[string]readerSectionDraft{}, TerminologyEdits: map[string]readerTermDraft{}}
		for sectionIndex, section := range readerSections(document) {
			sectionEdit := readerSectionDraft{SectionReceipt: readerSectionReceipt(section), Title: section.Title, Blocks: map[string]readerBlockDraft{}}
			for blockIndex, block := range section.Blocks {
				prose := block.Prose
				if sectionIndex == 0 && blockIndex == 0 {
					prose = opening
				}
				sectionEdit.Blocks[readerBlockKey(blockIndex)] = readerBlockDraft{OriginalSHA256: readerBlockSHA256(block), Prose: &prose}
			}
			draft.SectionEdits[readerSectionKey(sectionIndex)] = sectionEdit
		}
		return draft
	}
	invalid := makeDraft("자료가 직접 보여 주는 답은 산성이라는 점이다.")
	valid := makeDraft("다케다성은 산정의 군사 거점으로 지역 권력을 집중시킨 산성이었다.")
	provider := &recordingProvider{outputs: []string{string(mustMarshal(invalid)), string(mustMarshal(valid))}}
	verifier := &acceptingSourceReadVerifier{}
	catalog := testSourceCatalog(t, "mis_reader_code")
	config := testSourceProductConfig(provider, verifier)
	config.MissionID = catalog.MissionID
	config.MissionObjective = "일본 다케다성"
	_, _, attempts, err := runReaderStageWithTerminology(context.Background(), config, catalog, document, nil)
	if err != nil || len(attempts) != 2 || len(provider.requests) != 2 {
		t.Fatalf("reader repair attempts/error = %d/%v", len(attempts), err)
	}
	repair := provider.requests[1].Prompt
	if !strings.Contains(repair, "first paragraph") || !strings.Contains(repair, "distinct claim") {
		t.Fatalf("reader opening repair lacks specific content-free guidance: %s", repair)
	}
	if strings.Contains(repair, "자료가 직접 보여") {
		t.Fatalf("reader repair leaked prior rejected output: %s", repair)
	}
}

func TestAuthorDefersProcessNarrationCleanupToFinalReader(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_author_reader_boundary")
	author := authorDraft{
		LanguageReview: acceptedLanguageReview(),
		Title:          "다케다성", Language: "ko", ReaderTakeaway: "다케다성의 성격을 이해한다.",
		Throughline: "다케다성이 어떻게 기능했는가.", VoiceAndTone: "직접적",
		Sections: []authorSectionDraft{
			{Title: "입구", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "이 보고서에서는 다케다성의 위치와 기능을 살펴본다.", EvidenceSourceKeys: []string{"source_001"}}}},
			{Title: "결론", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "다음 섹션에서는 결론을 알아본다."}}},
		},
	}
	_, document, err := compileAuthorDraft(author, "narrative_boundary", "doc_boundary", "일본 다케다성", catalog, testSourceReadReceipt(catalog))
	if err != nil {
		t.Fatalf("author draft was rejected before the final reader could edit process narration: %v", err)
	}
	reader := readerDraft{Title: document.Title, Reviewer: "whole-reader", LanguageReview: acceptedLanguageReview(), ReaderReview: acceptedReaderReview(), SectionEdits: map[string]readerSectionDraft{}, TerminologyEdits: map[string]readerTermDraft{}}
	sections := readerSections(document)
	for sectionIndex, section := range sections {
		sectionEdit := readerSectionDraft{SectionReceipt: readerSectionReceipt(section), Title: section.Title, Blocks: map[string]readerBlockDraft{}}
		for blockIndex, block := range section.Blocks {
			prose := block.Prose
			sectionEdit.Blocks[readerBlockKey(blockIndex)] = readerBlockDraft{OriginalSHA256: readerBlockSHA256(block), Prose: &prose}
		}
		reader.SectionEdits[readerSectionKey(sectionIndex)] = sectionEdit
	}
	if _, _, err := compileReaderDraft(document, reader, "일본 다케다성"); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeReaderProcess {
		t.Fatalf("final reader accepted unedited process narration: code=%q err=%v", providerValidationCode(err), err)
	}
	opening := reader.SectionEdits["section_01"]
	directOpening := "다케다성은 험준한 산세를 방어와 통제에 활용한 산성이다."
	opening.Blocks["block_01"] = readerBlockDraft{OriginalSHA256: readerBlockSHA256(document.Blocks[1]), Prose: &directOpening}
	reader.SectionEdits["section_01"] = opening
	conclusion := reader.SectionEdits["section_02"]
	directConclusion := "결국 다케다성의 가치는 입지와 기능을 함께 볼 때 선명해진다."
	conclusion.Blocks["block_01"] = readerBlockDraft{OriginalSHA256: readerBlockSHA256(document.Blocks[3]), Prose: &directConclusion}
	reader.SectionEdits["section_02"] = conclusion
	if _, _, err := compileReaderDraft(document, reader, "일본 다케다성"); err != nil {
		t.Fatalf("final reader could not remove process narration: %v", err)
	}
}

func TestProviderPromptAndSchemaCostGuards(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_fixture")
	schema := providerAuthorSchemaBytes(catalog)
	t.Logf("author_schema_bytes=%d", len(schema))
	if len(schema) >= 12*1024 {
		t.Fatalf("author schema grew to %d bytes", len(schema))
	}
	if got := strings.Count(string(schema), `"document_block_draft"`); got != 1 {
		t.Fatalf("author block schema definition count = %d, want 1", got)
	}
	if got := strings.Count(string(schema), `#/$defs/document_block_draft`); got != 1 {
		t.Fatalf("author block schema reference count = %d, want 1", got)
	}

	provider := &recordingProvider{errs: []error{errors.New("capture")}}
	config := testSourceProductConfig(provider, &acceptingSourceReadVerifier{})
	config.Title = "Cost guard"
	config.MissionObjective = "Explain the result readers need."
	_, _, _, _ = runAuthorStage(context.Background(), config, catalog)
	if len(provider.requests) != 1 {
		t.Fatalf("author request capture = %d", len(provider.requests))
	}
	request := provider.requests[0]
	if strings.Contains(request.Prompt, "SCHEMA:\n") || strings.Contains(request.Prompt, string(request.OutputJSONSchema)) {
		t.Fatal("author prompt duplicated the request-local Structured Output schema")
	}
	for _, required := range []string{"request-local Structured Output schema", "server returns catalog-ordered batches", "until remaining_sources is zero, then stop", "Do not force every selected source", config.MissionObjective} {
		if !strings.Contains(request.Prompt, required) {
			t.Fatalf("author prompt lost %q: %s", required, request.Prompt)
		}
	}
	if strings.Contains(request.Prompt, "cite every frozen accepted source at least once") {
		t.Fatalf("author prompt retained forced source coverage: %s", request.Prompt)
	}
}

func TestFlowPromptUsesCompactOrderedReviewContext(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_fixture")
	narrative := Narrative{
		SchemaVersion: NarrativeSchemaVersion, ContractID: "narrative_flow_cost", DocumentID: "document_flow_cost",
		CentralQuestion: "Question?", ReaderTakeaway: "Takeaway.", Throughline: "Thread.", VoiceAndTone: "Natural.",
		TransitionObligations: []Transition{{FromSectionID: "section.a", ToSectionID: "section.b", Obligation: "Connect evidence to synthesis."}},
		ConclusionObligations: []string{"Close the open question."},
	}
	refID := serverAcceptedSourceReferenceID(catalog, "source_001")
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily, DocumentID: narrative.DocumentID, RevisionID: "draft", NarrativeContractID: narrative.ContractID,
		Title: "Compact flow", Language: "en", Provenance: map[string]string{"source": "accepted mission sources"},
		References: []Reference{{RefID: refID, Kind: "footnote", Target: "accepted-source:001", VisibleLabel: "Accepted source 1", Locator: "Frozen source catalog"}},
	}
	for index := 0; index < 9; index++ {
		sectionID := fmt.Sprintf("section.%c", 'a'+index)
		document.Blocks = append(document.Blocks, Block{NodeID: sectionID, Kind: "section", Level: 2, Title: fmt.Sprintf("Section %d", index+1)})
		for proseIndex := 0; proseIndex < 4; proseIndex++ {
			nodeID := fmt.Sprintf("prose.%d.%d", index+1, proseIndex+1)
			prose := strings.Repeat(fmt.Sprintf("Grounded sentence %d.%d keeps the manuscript coherent. ", index+1, proseIndex+1), 6)
			document.Blocks = append(document.Blocks, Block{NodeID: nodeID, Kind: "prose", ParentNodeID: sectionID, Prose: prose, EvidenceRefs: []string{refID}})
		}
	}
	provider := &recordingProvider{errs: []error{errors.New("capture")}}
	config := testSourceProductConfig(provider, &acceptingSourceReadVerifier{})
	_, _, _ = runFlowStage(context.Background(), config, catalog, narrative, document)
	if len(provider.requests) != 1 {
		t.Fatalf("Flow request capture = %d", len(provider.requests))
	}
	prompt := provider.requests[0].Prompt
	for _, required := range []string{"FLOW OBLIGATIONS:", "ORDERED REVIEW MANUSCRIPT:", "[PROSE prose.1.1]", "Accepted source 1", "Connect evidence to synthesis.", "Close the open question.", "Re-read every selected source completely", "remaining_sources is zero"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("compact Flow prompt lacks %q: %s", required, prompt)
		}
	}
	for _, forbidden := range []string{"SCHEMA:\n", `"schema_version"`, `"narrative_contract_id"`, `"references"`, "source_001", "src_fixture", "art_fixture"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("compact Flow prompt retained duplicated/private %q: %s", forbidden, prompt)
		}
	}
	reviewContext, err := flowReviewContext(narrative, document, catalog)
	if err != nil {
		t.Fatal(err)
	}
	fullDuplicateBytes := len(mustMarshal(narrative)) + len(mustMarshal(document))
	t.Logf("flow_review_context_bytes=%d duplicated_json_bytes=%d", len(reviewContext), fullDuplicateBytes)
	if len(reviewContext) >= fullDuplicateBytes*3/4 {
		t.Fatalf("compact Flow context = %d bytes, duplicated JSON inputs = %d bytes", len(reviewContext), fullDuplicateBytes)
	}
}

func TestRequestLocalSchemasBindIDsAndMoveStructureServerSide(t *testing.T) {
	narrative := Narrative{
		ContractID:   "narrative_bound",
		DocumentID:   "document_bound",
		SectionRoles: []SectionRole{{SectionID: "section.alpha"}, {SectionID: "section.beta"}},
	}
	narrativeSchema := providerNarrativeSchemaBytes(narrative.ContractID, narrative.DocumentID)
	if err := lintProviderSchema(narrativeSchema); err != nil {
		t.Fatal(err)
	}
	var narrativeRoot map[string]any
	if err := json.Unmarshal(narrativeSchema, &narrativeRoot); err != nil {
		t.Fatal(err)
	}
	narrativeProperties := narrativeRoot["$defs"].(map[string]any)["narrative"].(map[string]any)["properties"].(map[string]any)
	if narrativeProperties["contract_id"].(map[string]any)["const"] != narrative.ContractID || narrativeProperties["document_id"].(map[string]any)["const"] != narrative.DocumentID {
		t.Fatalf("request-local Narrative bindings = %#v", narrativeProperties)
	}

	documentSchema := providerDocumentSchemaBytes(narrative, testSourceCatalog(t, "mis_fixture"))
	if err := lintProviderSchema(documentSchema); err != nil {
		t.Fatal(err)
	}
	var documentRoot map[string]any
	if err := json.Unmarshal(documentSchema, &documentRoot); err != nil {
		t.Fatal(err)
	}
	draft := documentRoot["$defs"].(map[string]any)["document_draft"].(map[string]any)
	properties := draft["properties"].(map[string]any)
	sections := properties["sections"].(map[string]any)
	if got := sections["required"]; !reflect.DeepEqual(got, []any{"section.alpha", "section.beta"}) {
		t.Fatalf("required Narrative section inventory = %#v", got)
	}
	for _, serverOwned := range []string{"document_id", "narrative_contract_id", "revision_id", "node_id", "parent_node_id"} {
		if strings.Contains(string(documentSchema), `"`+serverOwned+`"`) {
			t.Fatalf("provider Document schema exposes server-owned %s: %s", serverOwned, documentSchema)
		}
	}
}

func TestValidateNarrativeContractRejectsInternalSemanticDefectsBeforeDocument(t *testing.T) {
	narrative := Narrative{
		SchemaVersion: NarrativeSchemaVersion,
		ContractID:    "narrative_1", DocumentID: "document_1",
		CentralQuestion: "Question?", ReaderTakeaway: "Takeaway", Throughline: "Thread",
		ReaderJourney: []string{"Journey"}, ArgumentArc: []string{"Arc"}, ConclusionObligations: []string{"Conclude"}, VoiceAndTone: "Direct",
		SectionRoles: []SectionRole{
			{SectionID: "section.a", Role: "setup", QuestionAnswered: "A?", PrerequisiteSections: []string{"section.b"}},
			{SectionID: "section.b", Role: "synthesis", QuestionAnswered: "B?", PrerequisiteSections: []string{"section.a"}},
		},
		DependencyEdges: []DependencyEdge{{FromSectionID: "section.a", ToSectionID: "section.b", Reason: "forward"}, {FromSectionID: "section.b", ToSectionID: "section.a", Reason: "back"}},
	}
	if err := validateNarrativeContract(narrative); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("Narrative semantic validation = %v", err)
	}
}

func TestProductDocumentIssuesIncludesCanonicalSchemaDefects(t *testing.T) {
	narrative := Narrative{
		ContractID: "narrative_1", DocumentID: "document_1",
		SectionRoles: []SectionRole{{SectionID: "section.a", Role: "synthesis", QuestionAnswered: "What?"}},
	}
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: narrative.DocumentID, RevisionID: "draft", NarrativeContractID: narrative.ContractID,
		Title: "Title", Language: "en",
		Blocks: []Block{
			{NodeID: "section.a", Kind: "section", Level: 2, Title: "A"},
			{NodeID: "paragraph.a", Kind: "prose", ParentNodeID: "section.a", Prose: "Body", SemanticRole: "provider-invented-role"},
		},
		Provenance: map[string]string{"source": "test"},
	}
	issues := strings.Join(productDocumentIssues(narrative, document), "\n")
	if !strings.Contains(issues, "canonical document schema") || !strings.Contains(issues, "semantic_role") {
		t.Fatalf("canonical schema defect was deferred past Document stage: %s", issues)
	}
}

func TestProductDocumentIssuesReportsAllCrossObjectDefects(t *testing.T) {
	narrative := Narrative{
		ContractID: "narrative_1", DocumentID: "document_1",
		SectionRoles: []SectionRole{{SectionID: "section.a", Role: "setup", QuestionAnswered: "A?"}, {SectionID: "section.b", Role: "synthesis", QuestionAnswered: "B?"}},
	}
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: "wrong", RevisionID: "wrong", NarrativeContractID: "wrong",
		Title: "Title", Language: "en",
		Blocks: []Block{
			{NodeID: "section.a", Kind: "section", Level: 2, Title: "A"},
			{NodeID: "section.a", Kind: "prose", ParentNodeID: "missing", Prose: "Prose", Supports: []string{"missing"}},
			{NodeID: "table.1", Kind: "table", ParentNodeID: "section.a", Table: &Table{Columns: []string{"A", "B"}, Rows: [][]string{{"one"}}}},
		},
		References: []Reference{{RefID: "ref.1", Kind: "citation", Target: "https://example.com"}},
		Coverage:   []Coverage{{RequirementID: "req.1", NodeIDs: []string{"missing"}, Status: "covered"}},
		Provenance: map[string]string{"source": "test"},
	}
	issues := productDocumentIssues(narrative, document)
	joined := strings.Join(issues, "\n")
	for _, want := range []string{"document shape", "Narrative binding", "revision binding", "initial profile", "document semantics", "Narrative realization"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("issues lack %q: %v", want, issues)
		}
	}
}

func TestAuthorRepairPromptRetainsReaderContractWithSafeManuscriptContext(t *testing.T) {
	invalid := authorDraft{
		LanguageReview: acceptedLanguageReview(), Title: "Invalid", Language: "en",
		ReaderTakeaway: "password=secret " + testPrivateUserPath("alice", "private.txt") + " Accepted source 1",
		Sections: []authorSectionDraft{
			{Title: "Unsafe", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "Private source_001 and https://private.example must not enter repair context.", EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
	provider := &recordingProvider{outputs: []string{string(mustMarshal(invalid)), string(mustMarshal(invalid))}}
	verifier := &acceptingSourceReadVerifier{}
	config := testSourceProductConfig(provider, verifier)
	config.MissionObjective = "Explain the subject itself."
	_, _, _, err := runAuthorStage(context.Background(), config, testSourceCatalog(t, "mis_fixture"))
	if err == nil || len(provider.requests) != 2 {
		t.Fatalf("repair calls/error = %d/%v", len(provider.requests), err)
	}
	if len(verifier.calls) != 2 || verifier.calls[0] == verifier.calls[1] || provider.requests[0].ReportILSources.Attempt != 1 || provider.requests[1].ReportILSources.Attempt != 2 {
		t.Fatalf("repair source sessions/bindings = %#v/%#v", verifier.calls, provider.requests)
	}
	repair := provider.requests[1].Prompt
	for _, want := range []string{"READER CONTRACT", "complete report a real reader asked for", "Explain the subject itself", "two-to-four-column table", "Do not force every selected source", config.MissionObjective} {
		if !strings.Contains(repair, want) {
			t.Fatalf("repair prompt lacks %q: %s", want, repair)
		}
	}
	for _, forbidden := range []string{"node_id must", "Every node_id", "parent_node_id", "cite every frozen accepted source"} {
		if strings.Contains(repair, forbidden) {
			t.Fatalf("repair retains old structural contract %q: %s", forbidden, repair)
		}
	}
	if string(provider.requests[0].OutputJSONSchema) != string(provider.requests[1].OutputJSONSchema) {
		t.Fatal("author repair changed request-local schema")
	}
	priorOutput := string(mustMarshal(invalid))
	safeContext := strings.SplitN(repair, "SAFE REQUEST-LOCAL MANUSCRIPT TO REPAIR:", 2)
	if strings.Contains(repair, priorOutput) || strings.Contains(repair, "PRIOR INVALID OUTPUT") || len(safeContext) != 2 || strings.Contains(safeContext[1], "evidence_source_keys") || strings.Contains(safeContext[1], "source_001") || strings.Contains(safeContext[1], "private.example") || strings.Contains(safeContext[1], "password=secret") || strings.Contains(safeContext[1], testPrivateUserPath("alice")) || strings.Contains(safeContext[1], "Accepted source 1") {
		t.Fatalf("source-aware repair prompt reused raw output or source metadata: %s", repair)
	}
	for _, want := range []string{
		"Raw provider output and validator prose are intentionally omitted",
		"Start a fresh source-tool session",
		"SAFE REQUEST-LOCAL MANUSCRIPT TO REPAIR",
		"TITLE: Invalid",
		"two to twelve nonempty sections",
		"Cite only source keys you actually read in this attempt",
		"Never reproduce source keys, generic source labels, or report-system terminology",
	} {
		if !strings.Contains(repair, want) {
			t.Fatalf("source-aware repair prompt lacks %q: %s", want, repair)
		}
	}
}

func TestAuthorSemanticRepairUsesClosedCodeAndSourceSafeContext(t *testing.T) {
	content := "法樹寺（ほうじゅじ）は地域の寺院である。"
	catalog := testSourceCatalog(t, "mis_safe_repair")
	readable := map[int]string{catalog.Sources[0].AcceptedOrdinal: content}
	invalid := authorDraft{
		LanguageReview: acceptedLanguageReview(),
		Title:          "다케다성", Language: "ko", ReaderTakeaway: "지역사를 이해한다.",
		Throughline: "성곽과 지역의 관계를 설명한다.", VoiceAndTone: "자연스럽고 직접적",
		Sections: []authorSectionDraft{
			{Title: "지역", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "법수사는 지역 사찰이다.", EvidenceSourceKeys: []string{"source_001"}}}},
			{Title: "의미", Blocks: []documentBlockDraft{{Kind: "prose", Prose: "성곽과 사찰의 관계가 지역사를 보여준다.", EvidenceSourceKeys: []string{"source_001"}}}},
		},
		Terminology: []authorTermDraft{{
			Category: "proper_name", SourceForm: "존재하지않는원문", ReaderForm: "법수사",
			Handling: "preserve_and_explain",
		}},
	}
	valid := invalid
	valid.Terminology = []authorTermDraft{{
		Category: "proper_name", SourceForm: "法樹寺", SourceReading: stringPointer("ほうじゅじ"), ReaderForm: "호주지",
		Handling: "preserve_and_explain",
	}}
	valid.Sections = append([]authorSectionDraft(nil), invalid.Sections...)
	valid.Sections[0].Blocks = append([]documentBlockDraft(nil), invalid.Sections[0].Blocks...)
	valid.Sections[0].Blocks[0].Prose = "호주지는 지역 사찰이다."
	provider := &recordingProvider{outputs: []string{string(mustMarshal(invalid)), string(mustMarshal(valid))}}
	verifier := &acceptingSourceReadVerifier{}
	config := testSourceProductConfig(provider, verifier)
	config.MissionID = catalog.MissionID
	contractID, documentID := "narrative_safe_repair", "doc_safe_repair"
	var accepted bool
	_, attempts, err := runJSONStageWithSourceSchema(context.Background(), config, catalog, "il_narrative", "ORIGINAL", providerAuthorSchemaBytes(catalog), func(value authorDraft, receipt reportilcontract.SourceReadReceipt) error {
		_, _, _, compileErr := compileAuthorDraftWithTerminology(value, contractID, documentID, "다케다성을 설명한다.", catalog, receipt, nil, readable)
		accepted = compileErr == nil
		return compileErr
	}, authorRepairContextWithValidation)
	if err != nil || !accepted || len(attempts) != 2 || len(provider.requests) != 2 {
		t.Fatalf("semantic repair result attempts=%d requests=%d accepted=%v err=%v", len(attempts), len(provider.requests), accepted, err)
	}
	repair := provider.requests[1].Prompt
	for _, required := range []string{"term_01", "exact source form", "selected source", "SAFE REQUEST-LOCAL MANUSCRIPT TO REPAIR"} {
		if !strings.Contains(repair, required) {
			t.Fatalf("semantic repair prompt lacks %q: %s", required, repair)
		}
	}
	for _, forbidden := range []string{"존재하지않는원문", "법수사", "source_001", "PRIVATE VALIDATOR DETAIL", string(mustMarshal(invalid))} {
		if strings.Contains(repair, forbidden) {
			t.Fatalf("semantic repair prompt leaked %q: %s", forbidden, repair)
		}
	}
	failure := providerStageFailure("il_narrative", &providerStageError{
		reason: reportexecution.ProviderFailureReasonSemanticValidation,
		cause:  withValidationCode(reportexecution.ProviderValidationCodeTerminologySourceGrounding, errors.New("private detail")),
	}, TerminalUsageReceipt{})
	if failure.SafeValidationCode != reportexecution.ProviderValidationCodeTerminologySourceGrounding {
		t.Fatalf("safe validation code = %q", failure.SafeValidationCode)
	}
}

func TestSafeRepairTextDropsEverySensitiveValueClass(t *testing.T) {
	for name, value := range map[string]string{
		"source key":             "claim from source_001",
		"accepted source id":     "claim from accepted-source:001",
		"generic source label":   "claim from Accepted source 1",
		"English source label":   "claim from Source 1",
		"Korean source label":    "claim from 승인 소스 1",
		"public URL locator":     "claim at https://example.com/source",
		"file URL locator":       "claim at file:///private/source.txt",
		"local user path":        "claim at " + testPrivateUserPath("alice", "private.txt"),
		"local home path":        "claim at /home/alice/private.txt",
		"password credential":    "password=secret-value",
		"authorization material": "Authorization: secret-value",
	} {
		if got := safeRepairText(value, nil); got != "" {
			t.Fatalf("%s survived repair sanitizer: %q", name, got)
		}
	}
	if got := safeRepairText("법수사는 지역 사찰이다.", []string{"법수사"}); got != "" {
		t.Fatalf("unvalidated terminology survived repair sanitizer: %q", got)
	}
	if got := safeRepairText("다케다성은 산정에 자리한다.", nil); got != "다케다성은 산정에 자리한다." {
		t.Fatalf("safe reader text was changed: %q", got)
	}
}

func TestReaderPromptOmitsUngroundedAuthorExplanation(t *testing.T) {
	term := terminologyDecision{
		Category: "proper_name", SourceForm: "法樹寺", SourceReading: "ほうじゅじ",
		ReaderForm: "호주지", FirstUseExplanation: "See source_001 accepted-source:001 PRIVATE VALIDATOR DETAIL",
		Handling: "preserve_and_explain",
	}
	prompt := readerPromptTerminology([]terminologyDecision{term})
	retry := sourceFreeRepairPrompt(prompt, "il_flow", "semantic_validation", reportexecution.ProviderValidationCodeTerminologyPresentation)
	for _, candidate := range []string{prompt, retry} {
		for _, forbidden := range []string{"source_001", "accepted-source:001", "PRIVATE VALIDATOR DETAIL", "author first-use explanation"} {
			if strings.Contains(candidate, forbidden) {
				t.Fatalf("reader terminology prompt leaked %q: %s", forbidden, candidate)
			}
		}
	}
	for _, required := range []string{"source form: 法樹寺", "source-supplied reading: ほうじゅじ", "author reader form: 호주지"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("reader terminology prompt lacks %q: %s", required, prompt)
		}
	}
}

func TestSourceAwareReaderRepairCarriesOnlyPacketAliasesIntoFreshSecondAttempt(t *testing.T) {
	provider := &recordingProvider{outputs: []string{`{"ok":true}`, `{"ok":true}`}}
	verifier := &acceptingSourceReadVerifier{}
	catalog := testSourceCatalog(t, "mis_reader_alias_repair")
	config := testSourceProductConfig(provider, verifier)
	config.MissionID = catalog.MissionID
	validationCalls := 0
	value, attempts, err := runJSONStageWithSourceSchema(
		context.Background(), config, catalog, "il_flow", "ORIGINAL SOURCE-FREE READER CONTEXT",
		[]byte(`{"type":"object","additionalProperties":false,"required":["ok"],"properties":{"ok":{"type":"boolean"}}}`),
		func(value map[string]any, _ reportilcontract.SourceReadReceipt) error {
			validationCalls++
			if validationCalls == 1 {
				return withValidationCodeAndPackets(
					reportexecution.ProviderValidationCodeEvidenceSupportQuote,
					[]string{"evidence_002", "claim from source_001", "evidence_002"},
					errors.New("PRIVATE VALIDATOR DETAIL"),
				)
			}
			return nil
		},
	)
	if err != nil || value["ok"] != true || len(attempts) != 2 || len(provider.requests) != 2 || len(verifier.calls) != 2 {
		t.Fatalf("reader alias repair result=%#v attempts=%d requests=%d verifier=%d err=%v", value, len(attempts), len(provider.requests), len(verifier.calls), err)
	}
	if provider.requests[0].ToolSessionID == provider.requests[1].ToolSessionID || !provider.requests[0].EphemeralSession || !provider.requests[1].EphemeralSession {
		t.Fatalf("reader repair did not use fresh ephemeral source sessions: %#v", provider.requests)
	}
	repair := provider.requests[1].Prompt
	for _, required := range []string{"evidence_002", "nonempty coverage_quote", "Start a fresh source-tool session"} {
		if !strings.Contains(repair, required) {
			t.Fatalf("reader alias repair lacks %q: %s", required, repair)
		}
	}
	if strings.Count(repair, "evidence_002") != 1 {
		t.Fatalf("reader alias repair did not deduplicate alias: %s", repair)
	}
	for _, forbidden := range []string{"claim from source_001", "source_001", "PRIVATE VALIDATOR DETAIL", "PRIOR INVALID OUTPUT"} {
		if strings.Contains(repair, forbidden) {
			t.Fatalf("reader alias repair leaked %q: %s", forbidden, repair)
		}
	}
}

func TestSourceAwareDecodeRepairOmitsPriorOutputAndValidatorDetail(t *testing.T) {
	const priorOutput = `SOURCE BODY SENTINEL {not-json}`
	provider := &recordingProvider{outputs: []string{priorOutput, `{"ok":true}`}}
	verifier := &acceptingSourceReadVerifier{}
	config := testSourceProductConfig(provider, verifier)
	value, attempts, err := runJSONStageWithSourceSchema(context.Background(), config, testSourceCatalog(t, "mis_fixture"), "il_narrative", "ORIGINAL SOURCE-FREE CONTEXT", []byte(`{"type":"object"}`), func(value map[string]any, _ reportilcontract.SourceReadReceipt) error {
		if value["ok"] != true {
			return errors.New("PRIVATE VALIDATOR DETAIL")
		}
		return nil
	})
	if err != nil || value["ok"] != true || len(attempts) != 2 || len(provider.requests) != 2 || len(verifier.calls) != 2 {
		t.Fatalf("source repair result=%#v attempts=%d requests=%d verifier=%#v err=%v", value, len(attempts), len(provider.requests), verifier.calls, err)
	}
	repair := provider.requests[1].Prompt
	for _, forbidden := range []string{priorOutput, "SOURCE BODY SENTINEL", "PRIVATE VALIDATOR DETAIL", "PRIOR INVALID OUTPUT"} {
		if strings.Contains(repair, forbidden) {
			t.Fatalf("source repair prompt leaked %q: %s", forbidden, repair)
		}
	}
	if !strings.Contains(repair, "(decode)") || !strings.Contains(repair, "ORIGINAL SOURCE-FREE CONTEXT") || provider.requests[0].ToolSessionID == provider.requests[1].ToolSessionID {
		t.Fatalf("source repair contract changed: %#v", provider.requests)
	}
}

func TestRunJSONStageRepairPreservesExactOutputSchema(t *testing.T) {
	provider := &recordingProvider{outputs: []string{"{}", `{"ok":true}`}}
	schema := []byte(`{"type":"object","additionalProperties":false,"required":["ok"],"properties":{"ok":{"type":"boolean"}}}`)
	value, attempts, err := runJSONStageWithSchema(context.Background(), ProductConfig{Provider: provider}, "il_document", "ORIGINAL CONTEXT", schema, func(value map[string]any) error {
		if value["ok"] != true {
			return errors.New("repair required")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if value["ok"] != true || len(attempts) != 2 || len(provider.requests) != 2 {
		t.Fatalf("repair result = %#v attempts=%d requests=%d", value, len(attempts), len(provider.requests))
	}
	for index, request := range provider.requests {
		if string(request.OutputJSONSchema) != string(schema) {
			t.Fatalf("request %d schema changed: %q", index, request.OutputJSONSchema)
		}
	}
	provider.requests[0].OutputJSONSchema[0] = 'X'
	if provider.requests[1].OutputJSONSchema[0] == 'X' {
		t.Fatal("repair requests share mutable schema storage")
	}
}

func TestRunJSONStageRepairPreservesContextSchemaAndUsage(t *testing.T) {
	const priorOutput = `{"source_key":"source_001","locator":"https://private.example/source"}`
	provider := &recordingProvider{
		outputs: []string{priorOutput, `{"ok":true}`},
		results: []agentexec.AgentResult{
			{Usage: testUsage(11, 3, "first-session")},
			{Usage: testUsage(17, 5, "second-session")},
		},
	}
	original := "ORIGINAL CONTEXT\nSCHEMA: strict-schema"
	value, attempts, err := runJSONStage(context.Background(), ProductConfig{Provider: provider}, "il_document", original, func(value map[string]any) error {
		if value["ok"] != true {
			return errors.New("repair required")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("repair failed: %v", err)
	}
	if value["ok"] != true || len(provider.requests) != 2 || len(attempts) != 2 {
		t.Fatalf("result/attempts/calls = %#v/%d/%d", value, len(attempts), len(provider.requests))
	}
	repair := provider.requests[1].Prompt
	if !strings.Contains(repair, original) || !strings.Contains(repair, "prior output and validator details are intentionally omitted") {
		t.Fatalf("repair prompt did not retain safe original context: %q", repair)
	}
	for _, forbidden := range []string{priorOutput, "source_001", "private.example", "PRIOR INVALID OUTPUT", "repair required"} {
		if strings.Contains(repair, forbidden) {
			t.Fatalf("source-free repair prompt leaked %q: %q", forbidden, repair)
		}
	}
	var receipt TerminalUsageReceipt
	receipt.add("il_document", attempts...)
	if receipt.DurationMS != 36 || len(receipt.Stages) != 2 || receipt.Stages[0].Usage.DurationMS != 14 || receipt.Stages[1].Usage.DurationMS != 22 || receipt.Stages[0].Usage.ProviderUsage.InputTokens != 11 || receipt.Stages[0].Usage.ProviderUsage.OutputTokens != 3 || receipt.Stages[1].Usage.ProviderUsage.InputTokens != 17 || receipt.Stages[1].Usage.ProviderUsage.OutputTokens != 5 {
		t.Fatalf("usage receipt = %#v", receipt)
	}
	for _, stage := range receipt.Stages {
		if stage.Usage.Prompt != (agentusage.PromptMetrics{}) || stage.Usage.Session != (agentusage.SessionMetrics{}) {
			t.Fatalf("durable receipt leaked prompt/session: %#v", stage.Usage)
		}
	}
}

func TestRunJSONStageTransportErrorMakesOneCall(t *testing.T) {
	provider := &recordingProvider{errs: []error{errors.New("transport")}}
	_, results, err := runJSONStage[map[string]any](context.Background(), ProductConfig{Provider: provider}, "il_document", "prompt", func(map[string]any) error { return nil })
	if err == nil || len(results) != 1 || len(provider.requests) != 1 {
		t.Fatalf("transport result/calls = %#v/%d err=%v", results, len(provider.requests), err)
	}
}

func TestRunJSONStageAllowsOneRepairOnly(t *testing.T) {
	provider := &recordingProvider{outputs: []string{"not json", "still not json", "third"}}
	config := ProductConfig{Provider: provider}
	if _, _, err := runJSONStage[map[string]any](context.Background(), config, "il_document", "prompt", func(map[string]any) error { return nil }); err == nil {
		t.Fatal("invalid provider JSON unexpectedly accepted")
	}
	if len(provider.requests) != 2 {
		t.Fatalf("provider calls = %d, want one repair", len(provider.requests))
	}
}

func TestProviderStageFailureRetainsPriorAndFailedAttemptUsage(t *testing.T) {
	var usage TerminalUsageReceipt
	narrativeUsage := testUsage(11, 3, "narrative-session")
	narrativeUsage.UsageSource = "private usage source"
	narrativeUsage.ContextWindow = &agentusage.ContextWindowMetrics{UsedTokens: 29, WindowTokens: 100, Source: "private context source"}
	usage.addResult("il_narrative", agentexec.AgentResult{
		Text:      "private narrative response",
		SessionID: "narrative-session",
		Log:       "private narrative log",
		Usage:     narrativeUsage,
	})
	provider := &recordingProvider{
		outputs: []string{`{"ok":false}`, `{"ok":false}`},
		results: []agentexec.AgentResult{
			{SessionID: "document-session-1", Log: "private document log 1", Usage: testUsage(17, 5, "document-session-1")},
			{SessionID: "document-session-2", Log: "private document log 2", Usage: testUsage(19, 7, "document-session-2")},
		},
	}
	schema := []byte(`{"type":"object","additionalProperties":false,"required":["ok"],"properties":{"ok":{"type":"boolean"}}}`)
	_, attempts, err := runJSONStageWithSchema(context.Background(), ProductConfig{Provider: provider}, "il_document", "private document prompt", schema, func(map[string]any) error {
		return errors.New("private semantic validator detail")
	})
	usage.add("il_document", attempts...)
	failure := providerStageFailure("il_document", err, usage)
	if failure.SafeFailureReason != reportexecution.ProviderFailureReasonSemanticValidation || failure.SafeValidationCode != reportexecution.ProviderValidationCodeSemanticContract || failure.ProviderUsage == nil || failure.Retryable {
		t.Fatalf("failure = %#v", failure)
	}
	if len(provider.requests) != 2 || string(provider.requests[0].OutputJSONSchema) != string(provider.requests[1].OutputJSONSchema) {
		t.Fatalf("repair requests = %#v", provider.requests)
	}
	receipt := failure.ProviderUsage
	if len(receipt.Attempts) != 3 || receipt.Attempts[0].Stage != "il_narrative" || receipt.Attempts[0].Attempt != 1 || receipt.Attempts[1].Stage != "il_document" || receipt.Attempts[1].Attempt != 1 || receipt.Attempts[2].Attempt != 2 || receipt.DurationMS != 62 {
		t.Fatalf("failure usage receipt = %#v", receipt)
	}
	if receipt.Attempts[0].Usage.ProviderUsage.TotalTokens != 14 || receipt.Attempts[1].Usage.ProviderUsage.TotalTokens != 22 || receipt.Attempts[2].Usage.ProviderUsage.TotalTokens != 26 {
		t.Fatalf("failure token receipt = %#v", receipt)
	}
	if receipt.Attempts[0].Usage.ContextWindow == nil || receipt.Attempts[0].Usage.ContextWindow.UsedTokens != 29 || receipt.Attempts[0].Usage.ContextWindow.WindowTokens != 100 {
		t.Fatalf("failure context receipt = %#v", receipt)
	}
	encoded, marshalErr := json.Marshal(map[string]any{"safe_failure_reason": failure.SafeFailureReason, "provider_usage_receipt": failure.ProviderUsage})
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	for _, forbidden := range []string{"private narrative response", "private narrative log", "private document prompt", "private document log", "private semantic validator detail", "private usage source", "private context source", "narrative-session", "document-session", `"prompt"`, `"session"`, "usage_source"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("failure payload leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestSemanticProviderStageFailureRetainsCompletedAttemptUsage(t *testing.T) {
	var usage TerminalUsageReceipt
	usage.add("il_narrative", agentexec.AgentResult{Usage: testUsage(11, 3, "narrative")})
	usage.add("il_document", agentexec.AgentResult{Usage: testUsage(17, 5, "document")})
	usage.add("il_flow", agentexec.AgentResult{Usage: testUsage(19, 7, "flow")})

	failure := semanticProviderStageFailure("il_flow", errors.New("private final bundle validation detail"), usage)
	if failure.SafeFailureReason != reportexecution.ProviderFailureReasonSemanticValidation || failure.ProviderUsage == nil || failure.Retryable {
		t.Fatalf("post-provider flow failure = %#v", failure)
	}
	if len(failure.ProviderUsage.Attempts) != 3 || failure.ProviderUsage.Attempts[2].Stage != reportexecution.ProviderFailureStageFlow || failure.ProviderUsage.Attempts[2].Attempt != 1 {
		t.Fatalf("post-provider flow usage = %#v", failure.ProviderUsage)
	}
	encoded, err := json.Marshal(map[string]any{"safe_failure_reason": failure.SafeFailureReason, "provider_usage_receipt": failure.ProviderUsage})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private final bundle validation detail") {
		t.Fatalf("post-provider failure leaked validator detail: %s", encoded)
	}
}

func TestRunProductClassifiesPostProviderDocumentValidationBeforeCompletion(t *testing.T) {
	content := []byte("bounded source")
	sourceHash := sha256Hex(content)
	sources := productSourceReader{
		sources:   []reportilcontract.SourceSnapshot{{SnapshotID: "src_1", MissionID: "mis_1", Active: true, ArtifactIDs: []string{"art_1"}, ContentHash: sourceHash}},
		artifacts: map[string]reportilcontract.Artifact{"art_1": {ArtifactID: "art_1", MissionID: "mis_1", MediaType: "text/plain", SHA256: sourceHash, ByteSize: int64(len(content)), Content: content}},
	}
	preserveStatement := "One bounded fact defines the result."
	author := reportFirstAuthorDraft{
		Title: "Bounded result", Language: "en",
		Sections: []reportFirstSectionDraft{
			{Title: "What is established", Blocks: []reportFirstBlockDraft{{Kind: "prose", Prose: preserveStatement, EvidenceSourceKeys: []string{"source_001"}}}},
			{Title: "What follows", Blocks: []reportFirstBlockDraft{{Kind: "prose", Prose: "That fact supports one bounded conclusion.", EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
	provider := &recordingProvider{
		outputs: []string{"workspace finalized", string(mustMarshal(author)), "workspace finalized", "workspace finalized"},
		results: []agentexec.AgentResult{
			{Usage: testUsage(13, 4, "editorial-memory")},
			{Usage: testUsage(11, 3, "author")},
			{Usage: testUsage(5, 1, "reader")},
			{Usage: testUsage(7, 2, "continuity")},
		},
	}
	progress := make([]string, 0)
	idCounts := map[string]int{}
	_, err := RunProduct(context.Background(), ProductConfig{
		MissionID: "mis_1", MissionObjective: "Explain the bounded conclusion.", Title: "Title", Direction: "Bounded", Sources: sources, Provider: provider,
		NewID: func(prefix string) string {
			idCounts[prefix]++
			if prefix == "rev" {
				return "invalid revision"
			}
			switch prefix {
			case "narrative":
				return "narrative_1"
			case "doc":
				return "doc_1"
			}
			return prefix + "_" + string(rune('0'+idCounts[prefix]))
		},
		Progress: func(stage, status string) error {
			progress = append(progress, stage+":"+status)
			return nil
		},
		PendingEventID: "evt_pending",
		VerifySourceRead: &acceptingSourceReadVerifier{sourceQuotes: map[string]reportilcontract.SourceQuoteReceipt{
			"quote_" + strings.Repeat("a", 64): {Receipt: "quote_" + strings.Repeat("a", 64), SourceKey: "source_001", Offset: 0, ByteSize: 1, SHA256: strings.Repeat("a", 64)},
		}},
		AuthorDocuments: &fixedAuthorDocumentReader{document: testAuthorDocument()},
	})
	var failure *reportexecution.StageFailureError
	if !errors.As(err, &failure) {
		t.Fatalf("post-provider validation error = %T %v", err, err)
	}
	if failure.Kind != "il_document" || failure.SafeFailureReason != reportexecution.ProviderFailureReasonSemanticValidation || failure.ProviderUsage == nil || len(failure.ProviderUsage.Attempts) != 4 || failure.ProviderUsage.Attempts[2].Stage != reportexecution.ProviderFailureStageReader || failure.ProviderUsage.Attempts[3].Stage != reportexecution.ProviderFailureStageContinuity {
		t.Fatalf("post-provider document failure = %#v", failure)
	}
	if slices.Contains(progress, "il_document:completed") {
		t.Fatalf("document completed before final validation: %v", progress)
	}
	if len(provider.requests) != 4 || provider.requests[0].UserText != "report IL il_editorial_memory" || provider.requests[1].UserText != "report IL il_narrative" || provider.requests[2].UserText != "report IL il_reader" || provider.requests[3].UserText != "report IL il_continuity" {
		t.Fatalf("provider calls = %#v", provider.requests)
	}
}

func TestProviderStageFailureClassifiesDecodeAndTransport(t *testing.T) {
	decodeProvider := &recordingProvider{outputs: []string{"not json", "still not json"}}
	_, decodeAttempts, decodeErr := runJSONStage[map[string]any](context.Background(), ProductConfig{Provider: decodeProvider}, "il_document", "prompt", func(map[string]any) error { return nil })
	var decodeUsage TerminalUsageReceipt
	decodeUsage.add("il_document", decodeAttempts...)
	decodeFailure := providerStageFailure("il_document", decodeErr, decodeUsage)
	if decodeFailure.SafeFailureReason != reportexecution.ProviderFailureReasonDecode || decodeFailure.ProviderUsage == nil || len(decodeFailure.ProviderUsage.Attempts) != 2 {
		t.Fatalf("decode failure = %#v", decodeFailure)
	}

	transportProvider := &recordingProvider{errs: []error{errors.New("private transport detail")}}
	_, transportAttempts, transportErr := runJSONStage[map[string]any](context.Background(), ProductConfig{Provider: transportProvider}, "il_document", "prompt", func(map[string]any) error { return nil })
	var transportUsage TerminalUsageReceipt
	transportUsage.add("il_document", transportAttempts...)
	transportFailure := providerStageFailure("il_document", transportErr, transportUsage)
	if transportFailure.SafeFailureReason != reportexecution.ProviderFailureReasonTransport || transportFailure.ProviderUsage == nil || len(transportFailure.ProviderUsage.Attempts) != 1 || !transportFailure.ProviderUsage.Attempts[0].UsageUnavailable || len(transportProvider.requests) != 1 {
		t.Fatalf("transport failure = %#v", transportFailure)
	}
}

func testUsage(input, output int, session string) agentusage.AgentUsage {
	return agentusage.New("codex", "codex", "model", "high", "prompt").WithProviderUsage(agentusage.ProviderUsage{InputTokens: input, OutputTokens: output}, "test").WithDuration(int64(input+output)).WithSession("previous", session, true, false)
}

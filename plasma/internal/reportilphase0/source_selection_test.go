package reportilphase0

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

func selectionCatalogWithSizes(t *testing.T, missionID string, sizes ...int) reportilcontract.SourceCatalog {
	t.Helper()
	entries := make([]reportilcontract.SourceCatalogEntry, 0, len(sizes))
	for index, size := range sizes {
		contentHash := sha256Hex([]byte(fmt.Sprintf("source-%d", index+1)))
		entries = append(entries, reportilcontract.SourceCatalogEntry{
			SourceKey:       fmt.Sprintf("source_%03d", index+1),
			AcceptedOrdinal: index + 1,
			SnapshotID:      fmt.Sprintf("src_selection_%03d", index+1),
			SnapshotReceipt: reportilcontract.SourceSnapshotReceipt(fmt.Sprintf("src_selection_%03d", index+1), contentHash),
			ContentHash:     contentHash,
			RetrievalPolicy: "snapshot_only",
			Artifacts: []reportilcontract.SourceCatalogArtifact{{
				ArtifactID: fmt.Sprintf("art_selection_%03d", index+1), SHA256: contentHash, ByteSize: int64(size), MediaType: "text/plain",
			}},
			ReadableSHA256: strings.Repeat(fmt.Sprintf("%x", (index+1)%16), 64),
			ReadableBytes:  size,
			Extraction:     "stored_text",
		})
	}
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{MissionID: missionID, Sources: entries})
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestSourceSelectionSchemaIsProviderCompatible(t *testing.T) {
	catalog := selectionCatalogWithSizes(t, "mis_selection_schema", 8, 16)
	standard := string(sourceSelectionSchemaBytes(catalog))
	longForm := string(sourceSelectionSchemaBytes(catalog, true))
	for _, schema := range []string{standard, longForm} {
		if err := lintProviderSchema([]byte(schema)); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Contains(standard, "long_form_supplement_source_keys") ||
		!strings.Contains(longForm, "long_form_supplement_source_keys") {
		t.Fatalf("authoring-mode source selection schemas:\nstandard=%s\nlong-form=%s", standard, longForm)
	}
}

func TestSourceSelectionSampleLimitFitsOneCallBudget(t *testing.T) {
	sizes := make([]int, 65)
	for index := range sizes {
		sizes[index] = 16 * 1024
	}
	catalog := selectionCatalogWithSizes(t, "mis_selection_sample", sizes...)
	limit, err := sourceSelectionSampleLimit(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if want := reportilcontract.DefaultSourceReadMaxBytes / len(catalog.Sources); limit != want || limit*len(catalog.Sources) > reportilcontract.DefaultSourceReadMaxBytes {
		t.Fatalf("sample limit = %d, want %d within one %d-byte call", limit, want, reportilcontract.DefaultSourceReadMaxBytes)
	}
}

func TestSourceSelectionSampleLimitFailsBeforeProviderWhenUTF8SampleCannotFit(t *testing.T) {
	catalog := reportilcontract.SourceCatalog{Sources: make([]reportilcontract.SourceCatalogEntry, reportilcontract.DefaultSourceReadMaxBytes/4+1)}
	if _, err := sourceSelectionSampleLimit(catalog); err == nil {
		t.Fatal("source selection accepted a per-source sample smaller than one UTF-8 rune")
	}
}

func TestValidateSourceSelectionReadsRequiresEveryExactSample(t *testing.T) {
	catalog := selectionCatalogWithSizes(t, "mis_selection_reads", 5, 10_000)
	receipt := reportilcontract.SourceReadReceipt{
		SourceKeys:           []string{"source_001", "source_002"},
		FullyReadSourceKeys:  []string{"source_001"},
		ReturnedContentBytes: 8197,
		ReadBytesBySource:    map[string]int{"source_001": 5, "source_002": 8192},
	}
	if err := validateSourceSelectionReads(catalog, receipt, 8192); err != nil {
		t.Fatal(err)
	}
	receipt.ReadBytesBySource["source_002"] = 8188
	receipt.ReturnedContentBytes = 8193
	if err := validateSourceSelectionReads(catalog, receipt, 8192); err == nil {
		t.Fatal("short source-selection sample was accepted")
	}
}

func TestSourceSelectionDraftStandardJSONOmitsLongFormSupplement(t *testing.T) {
	encoded := mustMarshal(sourceSelectionDraft{
		RankedSourceKeys: []string{"source_001"},
	})
	if strings.Contains(string(encoded), "long_form_supplement_source_keys") {
		t.Fatalf("standard source selection draft exposes long-form supplement: %s", encoded)
	}
}

func TestSourceSelectionPromptPreservesDistinctRequestedDimensions(t *testing.T) {
	catalog := selectionCatalogWithSizes(t, "mis_selection_coverage_prompt", 70*1024, 70*1024)
	ranking := sourceSelectionDraft{
		RankedSourceKeys:             []string{"source_001", "source_002"},
		LongFormSupplementSourceKeys: []string{},
	}
	provider := &recordingProvider{outputs: []string{string(mustMarshal(ranking))}}
	verifier := &fixedSourceReadVerifier{receipt: reportilcontract.SourceReadReceipt{
		SourceKeys: []string{"source_001", "source_002"}, ReturnedContentBytes: 16 * 1024,
		ReadBytesBySource: map[string]int{"source_001": 8 * 1024, "source_002": 8 * 1024},
	}}
	config := testSourceProductConfig(provider, verifier)
	config.MissionID = catalog.MissionID
	config.Direction = "Explain chronology, structures, relationships, and excavation examples."
	build := SourceCatalogBuild{
		Catalog: catalog,
		Dispositions: []SourceCatalogDisposition{
			{AcceptedOrdinal: 1, SourceKey: "source_001", Status: "available"},
			{AcceptedOrdinal: 2, SourceKey: "source_002", Status: "available"},
		},
	}
	if _, _, _, err := selectSourceCatalog(context.Background(), config, build); err != nil {
		t.Fatal(err)
	}
	if len(provider.requests) != 1 {
		t.Fatalf("source selection calls = %d", len(provider.requests))
	}
	prompt := provider.requests[0].Prompt
	for _, required := range []string{
		"every distinct positively requested subject, relationship, time period, structure, example, and context",
		"Rank at least one direct source for each supported requirement ahead of redundant sources",
		"does not displace a source that uniquely supplies a different requested dimension",
		"This stage chooses a compact reading set",
		"it does not write, summarize, or validate the report's facts",
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("source selection prompt lacks %q: %s", required, prompt)
		}
	}
	if strings.Contains(prompt, "material_source_accounts") || strings.Contains(string(provider.requests[0].OutputJSONSchema), "material_source_accounts") {
		t.Fatal("source selection still exposes material source accounts")
	}
}

func TestSourceSelectionReadFailureUsesClosedSourceReadCode(t *testing.T) {
	catalog := selectionCatalogWithSizes(t, "mis_selection_read_code", 70*1024, 70*1024)
	ranking := sourceSelectionDraft{
		RankedSourceKeys:             []string{"source_001", "source_002"},
		LongFormSupplementSourceKeys: []string{},
	}
	provider := &recordingProvider{outputs: []string{string(mustMarshal(ranking)), string(mustMarshal(ranking))}}
	verifier := &fixedSourceReadVerifier{receipt: reportilcontract.SourceReadReceipt{
		SourceKeys: []string{"source_001", "source_002"}, ReturnedContentBytes: 2,
		ReadBytesBySource: map[string]int{"source_001": 1, "source_002": 1},
	}}
	config := testSourceProductConfig(provider, verifier)
	config.MissionID = catalog.MissionID
	build := SourceCatalogBuild{
		Catalog: catalog,
		Dispositions: []SourceCatalogDisposition{
			{AcceptedOrdinal: 1, SourceKey: "source_001", Status: "available"},
			{AcceptedOrdinal: 2, SourceKey: "source_002", Status: "available"},
		},
	}
	_, _, results, err := selectSourceCatalog(context.Background(), config, build)
	if err == nil || len(results) != 2 || verifier.calls != 2 {
		t.Fatalf("source selection result attempts=%d verifier_calls=%d err=%v", len(results), verifier.calls, err)
	}
	failure := providerStageFailure("il_source_selection", err, TerminalUsageReceipt{})
	if failure.SafeFailureReason != reportexecution.ProviderFailureReasonSemanticValidation || failure.SafeValidationCode != reportexecution.ProviderValidationCodeSourceReadContract {
		t.Fatalf("source selection failure classification = %#v", failure)
	}
}

func TestCompileLongFormSupplementCatalogPreservesUniqueBoundedSources(t *testing.T) {
	catalog := selectionCatalogWithSizes(t, "mis_selection_long_form_supplement",
		70*1024, 70*1024, 40*1024, 300*1024, 220*1024,
	)
	selectedReadableBytes := 70 * 1024
	supplement, err := compileLongFormSupplementCatalog(
		catalog,
		[]string{"source_001", "source_002", "source_003", "source_004", "source_005"},
		[]int{1},
		selectedReadableBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(supplement.Sources) != 3 || sourceCatalogReadableBytes(supplement) != 410*1024 {
		t.Fatalf("supplement = %#v, bytes = %d", supplement.Sources, sourceCatalogReadableBytes(supplement))
	}
	if selectedReadableBytes+sourceCatalogReadableBytes(supplement) > reportilcontract.DefaultSourceAttemptReadBytes {
		t.Fatal("compact and supplemental catalogs exceed the complete-read attempt ceiling")
	}
	for index, wantOrdinal := range []int{2, 3, 4} {
		if supplement.Sources[index].AcceptedOrdinal != wantOrdinal ||
			supplement.Sources[index].SourceKey != fmt.Sprintf("source_%03d", index+1) {
			t.Fatalf("supplement source %d = %#v", index, supplement.Sources[index])
		}
	}
	merged, err := mergeSourceCatalogs(
		selectionCatalogWithSizes(t, "mis_selection_long_form_supplement", 70*1024),
		supplement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Sources) != 4 {
		t.Fatalf("merged source count = %d", len(merged.Sources))
	}
	for index, entry := range merged.Sources {
		if entry.SourceKey != fmt.Sprintf("source_%03d", index+1) {
			t.Fatalf("merged source key %d = %q", index, entry.SourceKey)
		}
	}
	if err := validateCompleteSourceCatalogBudget(merged); err != nil {
		t.Fatal(err)
	}
}

func TestCompileLongFormSupplementCatalogSharesCompleteReadBudget(t *testing.T) {
	catalog := selectionCatalogWithSizes(
		t,
		"mis_selection_long_form_shared_budget",
		120*1024,
		400*1024,
		16*1024,
	)
	supplement, err := compileLongFormSupplementCatalog(
		catalog,
		[]string{"source_002", "source_003"},
		[]int{1},
		120*1024,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(supplement.Sources) != 1 ||
		supplement.Sources[0].AcceptedOrdinal != 3 ||
		sourceCatalogReadableBytes(supplement) != 16*1024 {
		t.Fatalf("shared-budget supplement = %#v", supplement.Sources)
	}
	if _, err := compileLongFormSupplementCatalog(
		catalog,
		[]string{"source_002"},
		nil,
		reportilcontract.DefaultSourceAttemptReadBytes+1,
	); err == nil {
		t.Fatal("invalid compact source budget was accepted")
	}
}

func TestMergeSourceCatalogsPreservesCitationOrdinals(t *testing.T) {
	candidate := selectionCatalogWithSizes(t, "mis_selection_citation_ordinals", 32*1024, 32*1024, 32*1024)
	primary, _, err := compileSelectedSourceCatalog(
		candidate,
		[]string{"source_003", "source_001", "source_002"},
	)
	if err != nil {
		t.Fatal(err)
	}
	primary.Sources = append([]reportilcontract.SourceCatalogEntry(nil), primary.Sources[:2]...)
	primary, err = reportilcontract.SealSourceCatalog(primary)
	if err != nil {
		t.Fatal(err)
	}
	supplement, err := compileLongFormSupplementCatalog(
		candidate,
		[]string{"source_002"},
		[]int{3, 1},
		sourceCatalogReadableBytes(primary),
	)
	if err != nil {
		t.Fatal(err)
	}
	merged, err := mergeSourceCatalogs(primary, supplement)
	if err != nil {
		t.Fatal(err)
	}
	for index, wantOrdinal := range []int{3, 1, 2} {
		entry := merged.Sources[index]
		if entry.SourceKey != fmt.Sprintf("source_%03d", index+1) ||
			entry.AcceptedOrdinal != wantOrdinal {
			t.Fatalf("merged source %d = %#v", index, entry)
		}
	}
	build := SourceCatalogBuild{
		Catalog: candidate,
		CitationByKey: map[string]SourceCitation{
			"source_001": {VisibleLabel: "one"},
			"source_002": {VisibleLabel: "two"},
			"source_003": {VisibleLabel: "three"},
		},
	}
	citations := build.citationCatalog(merged)
	for ordinal, wantLabel := range map[int]string{1: "one", 2: "two", 3: "three"} {
		if citations[ordinal].VisibleLabel != wantLabel {
			t.Fatalf("citation ordinal %d = %#v", ordinal, citations[ordinal])
		}
	}
}

func TestSourceSelectionLongFormPromptRequestsOnlyUniqueSupplements(t *testing.T) {
	catalog := selectionCatalogWithSizes(t, "mis_selection_long_form_prompt", 70*1024, 70*1024)
	ranking := sourceSelectionDraft{
		RankedSourceKeys:             []string{"source_001", "source_002"},
		LongFormSupplementSourceKeys: []string{"source_002"},
	}
	provider := &recordingProvider{outputs: []string{string(mustMarshal(ranking))}}
	verifier := &fixedSourceReadVerifier{receipt: reportilcontract.SourceReadReceipt{
		SourceKeys: []string{"source_001", "source_002"}, ReturnedContentBytes: 16 * 1024,
		ReadBytesBySource: map[string]int{"source_001": 8 * 1024, "source_002": 8 * 1024},
	}}
	config := testSourceProductConfig(provider, verifier)
	config.MissionID = catalog.MissionID
	config.AuthoringMode = AuthoringModeLongForm
	build := SourceCatalogBuild{
		Catalog: catalog,
		Dispositions: []SourceCatalogDisposition{
			{AcceptedOrdinal: 1, SourceKey: "source_001", Status: "available"},
			{AcceptedOrdinal: 2, SourceKey: "source_002", Status: "available"},
		},
	}
	selection, receipt, _, err := selectSourceCatalog(context.Background(), config, build)
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Catalog.Sources) != 1 || len(selection.Supplement.Sources) != 1 ||
		selection.Supplement.Sources[0].AcceptedOrdinal != 2 ||
		len(selection.AuthorCatalog.Sources) != 2 ||
		selection.AuthorCatalog.Sources[1].SourceKey != "source_002" {
		t.Fatalf("long-form selection = %#v", selection)
	}
	if receipt.SelectedSources != 1 || receipt.SupplementalSources != 1 ||
		receipt.ExcludedBudgetSources != 0 ||
		receipt.SupplementalReadableBytes != 70*1024 ||
		receipt.AuthorCatalogSHA256 != selection.AuthorCatalog.SHA256 ||
		receipt.Dispositions[0].Status != "selected" ||
		receipt.Dispositions[1].Status != "supplemental" {
		t.Fatalf("long-form selection receipt = %#v", receipt)
	}
	prompt := provider.requests[0].Prompt
	for _, required := range []string{
		"long_form_supplement_source_keys",
		"uniquely supplies a materially different dimension",
		"Do not repeat a source already likely to fit the compact packet",
		"not permission to expand the standard author packet",
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("long-form selection prompt lacks %q: %s", required, prompt)
		}
	}
}

func TestSourceCatalogHasSelectableEntryRejectsOnlyOversizedCandidates(t *testing.T) {
	catalog := selectionCatalogWithSizes(t, "mis_selection_unfit", reportilcontract.DefaultSourceAttemptReadBytes+1)
	if sourceCatalogHasSelectableEntry(catalog) {
		t.Fatal("source selection would invoke provider when no source can fit")
	}
	catalog = selectionCatalogWithSizes(t, "mis_selection_fit", reportilcontract.DefaultSourceAttemptReadBytes+1, 1)
	if !sourceCatalogHasSelectableEntry(catalog) {
		t.Fatal("source selection rejected a catalog with one fitting source")
	}
}

func TestSourceCatalogFitsSelectionTargetRequiresByteAndCountBounds(t *testing.T) {
	if !sourceCatalogFitsSelectionTarget(selectionCatalogWithSizes(t, "mis_selection_target", 64*1024, 64*1024)) {
		t.Fatal("compact catalog did not bypass source selection")
	}
	if sourceCatalogFitsSelectionTarget(selectionCatalogWithSizes(t, "mis_selection_target_bytes", 64*1024, 64*1024+1)) {
		t.Fatal("catalog above the author byte target bypassed source selection")
	}
	sizes := make([]int, sourceSelectionMaxSources+1)
	for index := range sizes {
		sizes[index] = 1
	}
	if sourceCatalogFitsSelectionTarget(selectionCatalogWithSizes(t, "mis_selection_target_count", sizes...)) {
		t.Fatal("catalog above the author source-count target bypassed source selection")
	}
}

func TestCompileSelectedSourceCatalogUsesStrongestSourcesWithinBudget(t *testing.T) {
	catalog := selectionCatalogWithSizes(
		t,
		"mis_selection_ranked_budget",
		70*1024,
		70*1024,
		8*1024,
	)
	selected, ordinals, err := compileSelectedSourceCatalog(
		catalog,
		[]string{"source_001", "source_002", "source_003"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ordinals, []int{1, 3}) {
		t.Fatalf("ranked budget order = %#v / %#v", ordinals, selected.Sources)
	}
}

func TestCompileSelectedSourceCatalogTargetsCompactHighRankedPacket(t *testing.T) {
	catalog := selectionCatalogWithSizes(t, "mis_selection_compile", 443759, 60*1024, 50*1024, 30*1024)
	selected, ordinals, err := compileSelectedSourceCatalog(catalog, []string{"source_002", "source_003", "source_001", "source_004"})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Sources) != 2 || len(ordinals) != 2 || ordinals[0] != 2 || ordinals[1] != 3 {
		t.Fatalf("selected ordinals = %#v, sources = %#v", ordinals, selected.Sources)
	}
	if selected.Sources[0].SourceKey != "source_001" || selected.Sources[0].AcceptedOrdinal != 2 || selected.Sources[1].SourceKey != "source_002" || selected.Sources[1].AcceptedOrdinal != 3 {
		t.Fatalf("selected catalog did not preserve accepted ordinals: %#v", selected.Sources)
	}
	if got := sourceCatalogReadableBytes(selected); got > sourceSelectionTargetReadBytes {
		t.Fatalf("selected readable bytes = %d, target = %d", got, sourceSelectionTargetReadBytes)
	}
}

func TestCompileSelectedSourceCatalogPreservesStrongestFirstOrder(t *testing.T) {
	catalog := selectionCatalogWithSizes(t, "mis_selection_rank_order", 1024, 1024, 1024)
	selected, ordinals, err := compileSelectedSourceCatalog(
		catalog,
		[]string{"source_003", "source_001", "source_002"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ordinals, []int{3, 1, 2}) {
		t.Fatalf("selected ordinals = %#v, want strongest-first [3 1 2]", ordinals)
	}
	for index, wantOrdinal := range []int{3, 1, 2} {
		entry := selected.Sources[index]
		wantKey := fmt.Sprintf("source_%03d", index+1)
		if entry.SourceKey != wantKey || entry.AcceptedOrdinal != wantOrdinal {
			t.Fatalf("selected source %d = %#v, want key %s and ordinal %d", index, entry, wantKey, wantOrdinal)
		}
	}
	if err := reportilcontract.ValidateSourceCatalog(selected); err != nil {
		t.Fatal(err)
	}
}

func TestCompileSelectedSourceCatalogPreservesSmallSourcesWithinByteTarget(t *testing.T) {
	catalog := selectionCatalogWithSizes(t, "mis_selection_takeda_distribution",
		4076, 12150, 7751, 3637, 2725, 3670, 3366, 3719, 2397,
		4710, 3230, 7045, 6022, 3866, 45036, 443759, 4008, 2660,
	)
	ranked := make([]string, len(catalog.Sources))
	for index := range ranked {
		ranked[index] = fmt.Sprintf("source_%03d", index+1)
	}
	selected, ordinals, err := compileSelectedSourceCatalog(catalog, ranked)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Sources) != 17 || len(ordinals) != 17 {
		t.Fatalf("selected source count = %d, ordinals = %#v", len(selected.Sources), ordinals)
	}
	if slices.Contains(ordinals, 16) {
		t.Fatalf("oversized source displaced small sources: %#v", ordinals)
	}
	if got := sourceCatalogReadableBytes(selected); got != 120068 {
		t.Fatalf("selected readable bytes = %d, want 120068", got)
	}
}

func TestCompileSelectedSourceCatalogCapsCountAndFallsBackForOneLargeEssentialSource(t *testing.T) {
	sizes := make([]int, sourceSelectionMaxSources+3)
	for index := range sizes {
		sizes[index] = 1
	}
	catalog := selectionCatalogWithSizes(t, "mis_selection_count", sizes...)
	ranked := make([]string, len(sizes))
	for index := range ranked {
		ranked[index] = fmt.Sprintf("source_%03d", index+1)
	}
	selected, _, err := compileSelectedSourceCatalog(catalog, ranked)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Sources) != sourceSelectionMaxSources {
		t.Fatalf("selected source count = %d, want %d", len(selected.Sources), sourceSelectionMaxSources)
	}

	large := selectionCatalogWithSizes(t, "mis_selection_large_fallback", 200*1024, 300*1024)
	selected, ordinals, err := compileSelectedSourceCatalog(large, []string{"source_002", "source_001"})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Sources) != 1 || len(ordinals) != 1 || ordinals[0] != 2 || selected.Sources[0].ReadableBytes != 300*1024 {
		t.Fatalf("large-source fallback = %#v / %#v", ordinals, selected.Sources)
	}
	if err := validateCompleteSourceCatalogBudget(selected); err != nil {
		t.Fatal(err)
	}
}

func TestSourceSelectionReceiptIsContentFree(t *testing.T) {
	catalog := selectionCatalogWithSizes(t, "mis_selection_receipt", 8)
	build := SourceCatalogBuild{
		Catalog: catalog,
		Dispositions: []SourceCatalogDisposition{{
			AcceptedOrdinal: 1,
			SourceKey:       "source_001",
			SnapshotReceipt: "private-snapshot-receipt",
			Status:          "available",
		}},
	}
	receipt := sourceSelectionReceipt(build, catalog)
	receipt.Dispositions[0].Status = "selected"
	receipt.DispositionSHA256 = sourceSelectionDispositionSHA(receipt.Dispositions)
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"source_001", "private-snapshot-receipt", "src_selection", "art_selection"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("selection receipt leaked %q: %s", forbidden, encoded)
		}
	}
}

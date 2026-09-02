package reportilcontract

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/source"
)

func TestSourceCatalogRejectsInvalidRetrievalReceiptShapes(t *testing.T) {
	valid := SourceCatalog{
		MissionID: "mis_contract",
		Sources: []SourceCatalogEntry{{
			SourceKey: "source_001", SnapshotID: "src_contract",
			SnapshotReceipt: SourceSnapshotReceipt("src_contract", strings.Repeat("a", 64)),
			ContentHash:     strings.Repeat("a", 64),
			RetrievalPolicy: source.RetrievalPolicySnapshotOnly,
			Artifacts: []SourceCatalogArtifact{{
				ArtifactID: "art_contract", SHA256: strings.Repeat("a", 64), ByteSize: 5, MediaType: "text/plain",
			}},
			ReadableSHA256: strings.Repeat("b", 64), ReadableBytes: 5, Extraction: "stored_text",
		}},
	}
	if _, err := SealSourceCatalog(valid); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		mutate func(*SourceCatalogEntry)
	}{
		{name: "unknown policy", mutate: func(entry *SourceCatalogEntry) { entry.RetrievalPolicy = "ambient" }},
		{name: "snapshot without artifact", mutate: func(entry *SourceCatalogEntry) { entry.Artifacts = nil }},
		{name: "snapshot with observation", mutate: func(entry *SourceCatalogEntry) { entry.ObservationReceipt = "evt_observed" }},
		{name: "live with artifact", mutate: func(entry *SourceCatalogEntry) {
			entry.RetrievalPolicy = source.RetrievalPolicyLiveReference
			entry.ObservationReceipt = "evt_observed"
		}},
		{name: "live without observation", mutate: func(entry *SourceCatalogEntry) {
			entry.RetrievalPolicy = source.RetrievalPolicyLiveReference
			entry.Artifacts = nil
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			catalog := valid
			catalog.Sources = append([]SourceCatalogEntry(nil), valid.Sources...)
			catalog.Sources[0].Artifacts = append([]SourceCatalogArtifact(nil), valid.Sources[0].Artifacts...)
			tc.mutate(&catalog.Sources[0])
			if _, err := SealSourceCatalog(catalog); err == nil {
				t.Fatal("invalid source catalog receipt was accepted")
			}
		})
	}
}

func TestSourceReadReceiptValidatesContentFreeReadInventory(t *testing.T) {
	catalog, err := SealSourceCatalog(SourceCatalog{
		MissionID: "mis_receipt",
		Sources: []SourceCatalogEntry{{
			SourceKey: "source_001", SnapshotID: "src_receipt",
			SnapshotReceipt: SourceSnapshotReceipt("src_receipt", strings.Repeat("a", 64)),
			ContentHash:     strings.Repeat("a", 64), RetrievalPolicy: source.RetrievalPolicySnapshotOnly,
			Artifacts:      []SourceCatalogArtifact{{ArtifactID: "art_receipt", SHA256: strings.Repeat("a", 64), ByteSize: 5, MediaType: "text/plain"}},
			ReadableSHA256: strings.Repeat("b", 64), ReadableBytes: 5, Extraction: "stored_text",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	valid := SourceReadReceipt{
		SourceKeys:           []string{"source_001"},
		FullyReadSourceKeys:  []string{"source_001"},
		ReturnedContentBytes: 5,
		ReadBytesBySource:    map[string]int{"source_001": 5},
	}
	if err := valid.Validate(catalog); err != nil ||
		!valid.Includes("source_001") ||
		valid.Includes("source_999") ||
		!valid.IncludesEntireSource("source_001") ||
		valid.IncludesEntireSource("source_999") {
		t.Fatalf("valid source read receipt = %#v / %v", valid, err)
	}
	for _, invalid := range []SourceReadReceipt{
		{},
		{SourceKeys: []string{"source_001"}},
		{SourceKeys: []string{"source_999"}, ReturnedContentBytes: 1},
		{SourceKeys: []string{"source_001", "source_001"}, ReturnedContentBytes: 2},
		{SourceKeys: []string{"source_001"}, FullyReadSourceKeys: []string{"source_999"}, ReturnedContentBytes: 1},
		{SourceKeys: []string{"source_001"}, FullyReadSourceKeys: []string{"source_001", "source_001"}, ReturnedContentBytes: 2},
		{SourceKeys: []string{"source_001"}, ReturnedContentBytes: 5, ReadBytesBySource: map[string]int{"source_001": 4}},
		{SourceKeys: []string{"source_001"}, ReturnedContentBytes: 6, ReadBytesBySource: map[string]int{"source_001": 6}},
		{SourceKeys: []string{"source_001"}, ReturnedContentBytes: 5, ReadBytesBySource: map[string]int{"source_999": 5}},
	} {
		if err := invalid.Validate(catalog); err == nil {
			t.Fatalf("invalid source read receipt accepted: %#v", invalid)
		}
	}
}

func TestReaderSourceAccessBindingAllowsBoundedRepairAttempt(t *testing.T) {
	catalog, err := SealSourceCatalog(SourceCatalog{
		MissionID: "mis_reader_binding",
		Sources: []SourceCatalogEntry{{
			SourceKey: "source_001", SnapshotID: "src_reader_binding",
			SnapshotReceipt: SourceSnapshotReceipt("src_reader_binding", strings.Repeat("a", 64)),
			ContentHash:     strings.Repeat("a", 64), RetrievalPolicy: source.RetrievalPolicySnapshotOnly,
			Artifacts:      []SourceCatalogArtifact{{ArtifactID: "art_reader_binding", SHA256: strings.Repeat("a", 64), ByteSize: 5, MediaType: "text/plain"}},
			ReadableSHA256: strings.Repeat("b", 64), ReadableBytes: 5, Extraction: "stored_text",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	binding := SourceAccessBinding{
		PendingEventID: "evt_reader_binding", Stage: "il_reader", Attempt: 1,
		MaxReadBytes: DefaultSourceAttemptReadBytes, MaxCallBytes: DefaultSourceReadMaxBytes,
		EditorialMemoryArtifactID: "art_editorial_memory", EditorialMemorySHA256: strings.Repeat("d", 64),
		BaseAuthorArtifactID: "art_reader_document", BaseAuthorSHA256: strings.Repeat("c", 64),
		Catalog: catalog,
	}
	if err := ValidateSourceAccessBinding(binding); err != nil {
		t.Fatal(err)
	}
	binding.Attempt = 2
	if err := ValidateSourceAccessBinding(binding); err != nil {
		t.Fatalf("reader repair attempt source binding = %v", err)
	}
	binding.Attempt = 3
	if err := ValidateSourceAccessBinding(binding); err == nil {
		t.Fatal("third reader attempt source binding was accepted")
	}
}

func TestLongFormPartSourceBindingAllowsOptionalEditorialMemory(t *testing.T) {
	catalog, err := SealSourceCatalog(SourceCatalog{
		MissionID: "mis_long_form_part",
		Sources: []SourceCatalogEntry{{
			SourceKey: "source_001", SnapshotID: "src_long_form_part",
			SnapshotReceipt: SourceSnapshotReceipt("src_long_form_part", strings.Repeat("a", 64)),
			ContentHash:     strings.Repeat("a", 64), RetrievalPolicy: source.RetrievalPolicySnapshotOnly,
			Artifacts: []SourceCatalogArtifact{{
				ArtifactID: "art_long_form_source", SHA256: strings.Repeat("a", 64), ByteSize: 5, MediaType: "text/plain",
			}},
			ReadableSHA256: strings.Repeat("b", 64), ReadableBytes: 5, Extraction: "stored_text",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	binding := SourceAccessBinding{
		PendingEventID: "evt_long_form_part", Stage: "il_long_form_part", Attempt: 1,
		LongFormInputArtifactIDs: []string{"art_section_001"},
		LongFormInputSHA256s:     []string{strings.Repeat("c", 64)},
		LongFormPlanArtifactID:   "art_long_form_plan",
		LongFormPlanSHA256:       strings.Repeat("d", 64),
		LongFormPartKey:          "part_001",
		LongFormTargetLanguage:   "ko",
		Catalog:                  catalog,
	}
	if err := ValidateSourceAccessBinding(binding); err != nil {
		t.Fatalf("unverified Part binding = %v", err)
	}
	binding.EditorialMemoryArtifactID = "art_editorial_memory"
	binding.EditorialMemorySHA256 = strings.Repeat("e", 64)
	if err := ValidateSourceAccessBinding(binding); err != nil {
		t.Fatalf("memory-backed Part binding = %v", err)
	}
	binding.EditorialMemorySHA256 = ""
	if err := ValidateSourceAccessBinding(binding); err == nil {
		t.Fatal("partial optional editorial-memory binding was accepted")
	}
}

func TestLongFormSectionMemoryBindingRequiresBoundAccountInventory(t *testing.T) {
	catalog, err := SealSourceCatalog(SourceCatalog{
		MissionID: "mis_long_form_section",
		Sources: []SourceCatalogEntry{{
			SourceKey: "source_001", SnapshotID: "src_long_form_section",
			SnapshotReceipt: SourceSnapshotReceipt("src_long_form_section", strings.Repeat("a", 64)),
			ContentHash:     strings.Repeat("a", 64), RetrievalPolicy: source.RetrievalPolicySnapshotOnly,
			Artifacts:      []SourceCatalogArtifact{{ArtifactID: "art_long_form_source", SHA256: strings.Repeat("a", 64), ByteSize: 5, MediaType: "text/plain"}},
			ReadableSHA256: strings.Repeat("b", 64), ReadableBytes: 5, Extraction: "stored_text",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	binding := SourceAccessBinding{
		PendingEventID: "evt_long_form_section", Stage: "il_long_form_section", Attempt: 1,
		EditorialMemoryArtifactID: "art_editorial_memory", EditorialMemorySHA256: strings.Repeat("c", 64),
		LongFormPlanArtifactID: "art_long_form_plan", LongFormPlanSHA256: strings.Repeat("d", 64),
		LongFormPartKey: "part_001", LongFormSectionKey: "part_001.section_001", LongFormTargetLanguage: "ko",
		Catalog: catalog,
	}
	if err := ValidateSourceAccessBinding(binding); err == nil {
		t.Fatal("memory-backed long-form Section without account inventory was accepted")
	}
	binding.LongFormSectionAccountKeys = []string{"account_001"}
	if err := ValidateSourceAccessBinding(binding); err != nil {
		t.Fatalf("memory-backed long-form Section binding = %v", err)
	}
	binding.LongFormSectionAccountKeys = []string{"account_001", "account_001"}
	if err := ValidateSourceAccessBinding(binding); err == nil {
		t.Fatal("duplicate long-form Section account inventory was accepted")
	}
}

func TestFlowSourceAccessBindingRequiresOrderedNonOverlappingSpans(t *testing.T) {
	catalog, err := SealSourceCatalog(SourceCatalog{
		MissionID: "mis_flow_spans",
		Sources: []SourceCatalogEntry{{
			SourceKey: "source_001", SnapshotID: "src_flow_spans",
			SnapshotReceipt: SourceSnapshotReceipt("src_flow_spans", strings.Repeat("a", 64)),
			ContentHash:     strings.Repeat("a", 64), RetrievalPolicy: source.RetrievalPolicySnapshotOnly,
			Artifacts: []SourceCatalogArtifact{{
				ArtifactID: "art_flow_spans", SHA256: strings.Repeat("a", 64),
				ByteSize: 20, MediaType: "text/plain",
			}},
			ReadableSHA256: strings.Repeat("b", 64), ReadableBytes: 20,
			Extraction: "stored_text",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	valid := SourceAccessBinding{
		PendingEventID: "evt_flow_spans", Stage: "il_flow", Attempt: 1,
		MaxReadBytes: 8, MaxCallBytes: 8, Catalog: catalog,
		ReadSpans: []SourceReadSpan{
			{SourceKey: "source_001", Offset: 1, ByteSize: 3, SHA256: strings.Repeat("c", 64)},
			{SourceKey: "source_001", Offset: 10, ByteSize: 5, SHA256: strings.Repeat("d", 64)},
		},
	}
	if err := ValidateSourceAccessBinding(valid); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		mutate func(*SourceAccessBinding)
	}{
		{name: "overlap", mutate: func(value *SourceAccessBinding) {
			value.ReadSpans[1].Offset = 3
		}},
		{name: "unordered", mutate: func(value *SourceAccessBinding) {
			value.ReadSpans[0], value.ReadSpans[1] = value.ReadSpans[1], value.ReadSpans[0]
		}},
		{name: "budget mismatch", mutate: func(value *SourceAccessBinding) {
			value.MaxReadBytes++
		}},
		{name: "wrong stage", mutate: func(value *SourceAccessBinding) {
			value.Stage = "il_narrative"
		}},
		{name: "outside source", mutate: func(value *SourceAccessBinding) {
			value.ReadSpans[1].Offset = 18
		}},
		{name: "non-hex hash", mutate: func(value *SourceAccessBinding) {
			value.ReadSpans[0].SHA256 = strings.Repeat("g", 64)
		}},
		{name: "uppercase hash", mutate: func(value *SourceAccessBinding) {
			value.ReadSpans[0].SHA256 = strings.Repeat("A", 64)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := valid
			candidate.ReadSpans = append([]SourceReadSpan(nil), valid.ReadSpans...)
			tc.mutate(&candidate)
			if err := ValidateSourceAccessBinding(candidate); err == nil {
				t.Fatalf("invalid flow span binding was accepted: %#v", candidate)
			}
		})
	}
}

func TestSourceCatalogAcceptsLiveReceiptWithoutStoredArtifacts(t *testing.T) {
	catalog, err := SealSourceCatalog(SourceCatalog{
		MissionID: "mis_live_contract",
		Sources: []SourceCatalogEntry{{
			SourceKey: "source_001", SnapshotID: "src_live_contract",
			SnapshotReceipt: SourceSnapshotReceipt("src_live_contract", ""),
			RetrievalPolicy: source.RetrievalPolicyLiveReference,
			ReadableSHA256:  strings.Repeat("b", 64), ReadableBytes: 5, Extraction: "live_text",
			ObservationReceipt: "evt_observed",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSourceCatalog(catalog); err != nil {
		t.Fatal(err)
	}
}

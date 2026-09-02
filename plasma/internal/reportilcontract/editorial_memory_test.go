package reportilcontract

import (
	"reflect"
	"strings"
	"testing"
)

func TestEditorialAccountSourceKeysPreservesAccountOrderAndDeduplicatesSources(t *testing.T) {
	catalog := testEditorialMemoryCatalog(t)
	memory := EditorialMemory{
		SchemaVersion: EditorialMemorySchemaVersion,
		Language:      "en",
		Accounts: []EditorialAccount{
			{AccountKey: "account_001", Importance: "essential", Account: "The first connected account.", SourceKeys: []string{"source_002", "source_001"}},
			{AccountKey: "account_002", Importance: "supporting", Account: "The second connected account.", SourceKeys: []string{"source_001", "source_003"}},
		},
	}
	if err := ValidateEditorialMemory(memory, catalog); err != nil {
		t.Fatal(err)
	}
	got, err := EditorialAccountSourceKeys(memory, []string{"account_001", "account_002"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"source_002", "source_001", "source_003"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("source keys = %#v, want %#v", got, want)
	}
	for _, keys := range [][]string{nil, {}, {"account_001", "account_001"}, {"account_999"}} {
		if _, err := EditorialAccountSourceKeys(memory, keys); err == nil {
			t.Fatalf("invalid account binding accepted: %#v", keys)
		}
	}
}

func TestValidateAuthorDocumentWithMemoryRejectsDivergentSourceBindings(t *testing.T) {
	catalog := testEditorialMemoryCatalog(t)
	memory := EditorialMemory{
		SchemaVersion: EditorialMemorySchemaVersion,
		Language:      "en",
		Accounts: []EditorialAccount{
			{AccountKey: "account_001", Importance: "essential", Account: "The connected account.", SourceKeys: []string{"source_001", "source_002"}},
		},
	}
	document := AuthorDocument{
		SchemaVersion: AuthorDocumentSchemaVersion,
		Title:         "Bound report",
		Language:      "en",
		Sections: []AuthorSection{
			{SectionKey: "section_001", Title: "Answer", Blocks: []AuthorBlock{{
				BlockKey: "section_001.block_001", Kind: "prose", Prose: "The connected account answers the question.",
				EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001", "source_002"},
			}}},
			{SectionKey: "section_002", Title: "Judgment", Blocks: []AuthorBlock{{
				BlockKey: "section_002.block_001", Kind: "prose", Prose: "The evidence supports the judgment.",
				EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001", "source_002"},
			}}},
		},
	}
	if err := ValidateAuthorDocumentWithMemory(document, memory, catalog); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*AuthorDocument){
		"missing account": func(document *AuthorDocument) {
			document.Sections[0].Blocks[0].EditorialAccountKeys = nil
		},
		"missing source": func(document *AuthorDocument) {
			document.Sections[0].Blocks[0].EvidenceSourceKeys = []string{"source_001"}
		},
		"reordered source": func(document *AuthorDocument) {
			document.Sections[0].Blocks[0].EvidenceSourceKeys = []string{"source_002", "source_001"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := document
			candidate.Sections = append([]AuthorSection(nil), document.Sections...)
			candidate.Sections[0].Blocks = append([]AuthorBlock(nil), document.Sections[0].Blocks...)
			mutate(&candidate)
			if err := ValidateAuthorDocumentWithMemory(candidate, memory, catalog); err == nil {
				t.Fatal("divergent document binding was accepted")
			}
		})
	}
}

func TestValidateEditorialMemoryArtifactPreservesExactCausativeAnchor(t *testing.T) {
	catalog := testEditorialMemoryCatalogWithReadableBytes(t, len([]byte("築かせた")))
	anchorText := "築かせた"
	artifact := EditorialMemoryArtifact{
		SchemaVersion: EditorialMemorySchemaVersion,
		Language:      "ko",
		Accounts: []EditorialAccount{{
			AccountKey: "account_001", Importance: "essential",
			Account:    "1433년 다지마 수호 야마나 소젠이 성을 직접 쌓은 것이 아니라 13년에 걸쳐 쌓게 했다고 전한다.",
			SourceKeys: []string{"source_001"},
		}},
		Anchors: []EditorialAnchor{{
			AccountKey: "account_001", SourceKey: "source_001", Excerpt: anchorText,
			Offset: 0, ByteSize: len([]byte(anchorText)), SHA256: sourceAccessSHA256([]byte(anchorText)),
		}},
	}
	if err := ValidateEditorialMemoryArtifact(artifact, catalog); err != nil {
		t.Fatal(err)
	}
	if artifact.Anchors[0].Excerpt != "築かせた" {
		t.Fatalf("causative anchor = %q", artifact.Anchors[0].Excerpt)
	}
}

func TestValidateEditorialMemoryArtifactRejectsInvalidAnchors(t *testing.T) {
	catalog := testEditorialMemoryCatalogWithReadableBytes(t, 32)
	valid := EditorialMemoryArtifact{
		SchemaVersion: EditorialMemorySchemaVersion,
		Language:      "en",
		Accounts: []EditorialAccount{{
			AccountKey: "account_001", Importance: "essential", Account: "The source says the governor had the castle built.",
			SourceKeys: []string{"source_001"},
		}},
		Anchors: []EditorialAnchor{{
			AccountKey: "account_001", SourceKey: "source_001", Excerpt: "had it built",
			Offset: 0, ByteSize: len("had it built"), SHA256: sourceAccessSHA256([]byte("had it built")),
		}},
	}
	if err := ValidateEditorialMemoryArtifact(valid, catalog); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*EditorialMemoryArtifact){
		"missing anchors": func(artifact *EditorialMemoryArtifact) {
			artifact.Anchors = nil
		},
		"unknown account": func(artifact *EditorialMemoryArtifact) {
			artifact.Anchors[0].AccountKey = "account_999"
		},
		"unknown source": func(artifact *EditorialMemoryArtifact) {
			artifact.Anchors[0].SourceKey = "source_999"
		},
		"source not declared": func(artifact *EditorialMemoryArtifact) {
			artifact.Accounts[0].SourceKeys = []string{"source_002"}
		},
		"duplicate anchor": func(artifact *EditorialMemoryArtifact) {
			artifact.Anchors = append(artifact.Anchors, artifact.Anchors[0])
		},
		"out of range": func(artifact *EditorialMemoryArtifact) {
			artifact.Anchors[0].Offset = 32
		},
		"incorrect byte size": func(artifact *EditorialMemoryArtifact) {
			artifact.Anchors[0].ByteSize++
		},
		"malformed sha": func(artifact *EditorialMemoryArtifact) {
			artifact.Anchors[0].SHA256 = "invalid"
		},
		"excerpt sha mismatch": func(artifact *EditorialMemoryArtifact) {
			artifact.Anchors[0].SHA256 = strings.Repeat("a", 64)
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			candidate.Accounts = append([]EditorialAccount(nil), valid.Accounts...)
			candidate.Accounts[0].SourceKeys = append([]string(nil), valid.Accounts[0].SourceKeys...)
			candidate.Anchors = append([]EditorialAnchor(nil), valid.Anchors...)
			mutate(&candidate)
			if err := ValidateEditorialMemoryArtifact(candidate, catalog); err == nil {
				t.Fatal("invalid editorial anchor was accepted")
			}
		})
	}
}

func testEditorialMemoryCatalogWithReadableBytes(t *testing.T, readableBytes int) SourceCatalog {
	t.Helper()
	catalog := testEditorialMemoryCatalog(t)
	catalog.Sources = catalog.Sources[:2]
	for index := range catalog.Sources {
		catalog.Sources[index].ReadableBytes = readableBytes
	}
	sealed, err := SealSourceCatalog(catalog)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}

func testEditorialMemoryCatalog(t *testing.T) SourceCatalog {
	t.Helper()
	sources := make([]SourceCatalogEntry, 3)
	for index := range sources {
		ordinal := index + 1
		key := "source_00" + string(rune('0'+ordinal))
		hash := strings.Repeat(string(rune('a'+index)), 64)
		snapshotID := "src_editorial_00" + string(rune('0'+ordinal))
		sources[index] = SourceCatalogEntry{
			SourceKey: key, AcceptedOrdinal: ordinal, SnapshotID: snapshotID,
			SnapshotReceipt: SourceSnapshotReceipt(snapshotID, hash), ContentHash: hash,
			RetrievalPolicy: "snapshot_only",
			Artifacts: []SourceCatalogArtifact{{
				ArtifactID: "art_editorial_00" + string(rune('0'+ordinal)), SHA256: hash, ByteSize: 1, MediaType: "text/plain",
			}},
			ReadableSHA256: hash, ReadableBytes: 1, Extraction: "stored_text",
		}
	}
	catalog, err := SealSourceCatalog(SourceCatalog{MissionID: "mis_editorial_memory", Sources: sources})
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

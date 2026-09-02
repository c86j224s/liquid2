package reportilcontract

import "testing"

func TestLongFormFinalizedFragmentRejectsEmptySection(t *testing.T) {
	catalog := longFormAuthorTestCatalog(t)
	document := AuthorDocument{
		SchemaVersion: LongFormAuthorDocumentSchemaVersion,
		Title:         "Long report",
		Language:      "en",
		Parts: []AuthorPart{{
			PartKey: "part_002",
			Title:   "Part 2",
			Sections: []AuthorSection{{
				SectionKey: "part_002.section_003",
				Title:      "Section 3",
			}},
		}},
	}
	if err := ValidateAuthorDocumentPartial(document, catalog); err != nil {
		t.Fatalf("in-progress fragment = %v", err)
	}
	if err := ValidateLongFormAuthorFragment(document, catalog); err == nil {
		t.Fatal("empty finalized long-form fragment was accepted")
	}
}

func TestLongFormAuthorEquationRequiresLatexPayload(t *testing.T) {
	catalog := longFormAuthorTestCatalog(t)
	section := AuthorSection{
		SectionKey: "part_001.section_001",
		Title:      "Cost equation",
		Blocks: []AuthorBlock{
			{
				BlockKey: "part_001.section_001.block_001", Kind: "prose",
				Prose:              "The total cost separates repeated inputs from generated output.",
				EvidenceSourceKeys: []string{"source_001"},
			},
			{
				BlockKey: "part_001.section_001.block_002", Kind: "equation",
				Equation:           &AuthorEquation{Expression: `C = n_i p_i + n_o p_o`, Notation: "latex"},
				EvidenceSourceKeys: []string{"source_001"},
			},
		},
	}
	document := AuthorDocument{
		SchemaVersion: LongFormAuthorDocumentSchemaVersion,
		Title:         "Long report", Language: "en",
		Parts: []AuthorPart{{PartKey: "part_001", Title: "Part 1", Sections: []AuthorSection{section}}},
	}
	if err := ValidateLongFormAuthorFragment(document, catalog); err != nil {
		t.Fatal(err)
	}

	badNotation := document
	badNotation.Parts = append([]AuthorPart(nil), document.Parts...)
	badNotation.Parts[0].Sections = append([]AuthorSection(nil), document.Parts[0].Sections...)
	badNotation.Parts[0].Sections[0].Blocks = append([]AuthorBlock(nil), section.Blocks...)
	badNotation.Parts[0].Sections[0].Blocks[1].Equation = &AuthorEquation{Expression: "C = x", Notation: "mathml"}
	if err := ValidateLongFormAuthorFragment(badNotation, catalog); err == nil {
		t.Fatal("non-LaTeX equation was accepted")
	}
}

func TestAuthorProseRejectsProjectionMarkup(t *testing.T) {
	catalog := longFormAuthorTestCatalog(t)
	for _, prose := range []string{
		"**Worked example**\n\nReader-facing explanation.",
		"```python\nprint('x')\n```",
		"## Hidden heading",
		"- Hidden list item",
		`\[C = x\]`,
	} {
		document := AuthorDocument{
			SchemaVersion: LongFormAuthorDocumentSchemaVersion,
			Title:         "Long report",
			Language:      "en",
			Parts: []AuthorPart{{
				PartKey: "part_001", Title: "Part 1",
				Sections: []AuthorSection{{
					SectionKey: "part_001.section_001", Title: "Section 1",
					Blocks: []AuthorBlock{{
						BlockKey: "part_001.section_001.block_001", Kind: "prose",
						Prose: prose, EvidenceSourceKeys: []string{"source_001"},
					}},
				}},
			}},
		}
		if err := ValidateLongFormAuthorFragment(document, catalog); err == nil || err.Error() != "author document prose block contains presentation markup" {
			t.Fatalf("prose %q validation = %v", prose, err)
		}
	}
}

func TestLongFormRepresentationObligationsRequireFirstClassBlocks(t *testing.T) {
	planned := LongFormSection{
		Representations: []string{
			LongFormRepresentationTable,
			LongFormRepresentationCode,
			LongFormRepresentationEquation,
			LongFormRepresentationWorkedExample,
			LongFormRepresentationBenchmark,
			LongFormRepresentationChecklist,
			LongFormRepresentationDiagram,
		},
	}
	authored := AuthorSection{Blocks: []AuthorBlock{
		{Kind: "prose"},
		{Kind: "table"},
		{Kind: "code"},
		{Kind: "equation"},
	}}
	if err := ValidateLongFormRepresentationObligations(planned, authored); err != nil {
		t.Fatal(err)
	}
	for index, missing := range []string{
		LongFormRepresentationTable,
		LongFormRepresentationCode,
		LongFormRepresentationEquation,
	} {
		candidate := authored
		candidate.Blocks = append([]AuthorBlock(nil), authored.Blocks...)
		candidate.Blocks = append(candidate.Blocks[:index+1], candidate.Blocks[index+2:]...)
		if err := ValidateLongFormRepresentationObligations(planned, candidate); err == nil || err.Error() != "planned long-form "+missing+" representation is missing" {
			t.Fatalf("missing %s = %v", missing, err)
		}
	}
}

func TestLongFormValidatorsRejectStandardDocument(t *testing.T) {
	catalog := longFormAuthorTestCatalog(t)
	document := AuthorDocument{
		SchemaVersion: AuthorDocumentSchemaVersion,
		Title:         "Standard report",
		Language:      "en",
		Sections: []AuthorSection{
			longFormAuthorStandardSection("section_001"),
			longFormAuthorStandardSection("section_002"),
		},
	}
	if err := ValidateAuthorDocument(document, catalog); err != nil {
		t.Fatalf("standard document = %v", err)
	}
	if err := ValidateLongFormAuthorFragment(document, catalog); err == nil {
		t.Fatal("standard document was accepted as a long-form fragment")
	}
	if err := ValidateLongFormAuthorDocument(document, catalog); err == nil {
		t.Fatal("standard document was accepted as a publishable long-form document")
	}
}

func longFormAuthorStandardSection(key string) AuthorSection {
	return AuthorSection{
		SectionKey: key,
		Title:      "Section",
		Blocks: []AuthorBlock{{
			BlockKey:           key + ".block_001",
			Kind:               "prose",
			Prose:              "A complete reader-facing passage.",
			EvidenceSourceKeys: []string{"source_001"},
		}},
	}
}

func longFormAuthorTestCatalog(t *testing.T) SourceCatalog {
	t.Helper()
	catalog, err := SealSourceCatalog(SourceCatalog{
		MissionID: "mis_long_form_author",
		Sources: []SourceCatalogEntry{{
			SourceKey:       "source_001",
			SnapshotID:      "src_long_form_author",
			SnapshotReceipt: SourceSnapshotReceipt("src_long_form_author", testSHA256("a")),
			ContentHash:     testSHA256("a"),
			RetrievalPolicy: "snapshot_only",
			Artifacts: []SourceCatalogArtifact{{
				ArtifactID: "art_long_form_author", SHA256: testSHA256("a"), ByteSize: 5, MediaType: "text/plain",
			}},
			ReadableSHA256: testSHA256("b"), ReadableBytes: 5, Extraction: "stored_text",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func testSHA256(character string) string {
	value := ""
	for len(value) < 64 {
		value += character
	}
	return value
}

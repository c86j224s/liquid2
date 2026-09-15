package reportilcontract

import (
	"fmt"
	"strings"
	"testing"
)

func TestLongFormAllowsMoreThan128BlocksWithinByteBudget(t *testing.T) {
	catalog := longFormAuthorTestCatalog(t)
	document := AuthorDocument{SchemaVersion: LongFormAuthorDocumentSchemaVersion, Title: "Long report", Language: "en"}
	for p := 1; p <= 2; p++ {
		part := AuthorPart{PartKey: fmt.Sprintf("part_%03d", p), Title: fmt.Sprintf("Part %d", p)}
		for s := 1; s <= 3; s++ {
			key := fmt.Sprintf("%s.section_%03d", part.PartKey, s)
			section := AuthorSection{SectionKey: key, Title: fmt.Sprintf("Section %d", s)}
			for b := 1; b <= 27; b++ {
				section.Blocks = append(section.Blocks, AuthorBlock{BlockKey: fmt.Sprintf("%s.block_%03d", key, b), Kind: "prose", Prose: "A supported explanation.", EvidenceSourceKeys: []string{"source_001"}})
			}
			part.Sections = append(part.Sections, section)
		}
		document.Parts = append(document.Parts, part)
	}
	if err := ValidateAuthorDocumentPartial(document, catalog); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLongFormAuthorDocument(document, catalog); err != nil {
		t.Fatal(err)
	}
	document.Title = strings.Repeat("x", MaxAuthorDocumentBytes)
	if err := ValidateAuthorDocumentPartial(document, catalog); err == nil {
		t.Fatal("oversized document accepted")
	}
}

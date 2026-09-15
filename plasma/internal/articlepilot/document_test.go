package articlepilot

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/articleexperiment"
)

func TestParseDocumentAndRenderMarkdown(t *testing.T) {
	long := strings.Repeat("가", 7000)
	document := Document{SchemaVersion: DocumentSchemaVersion, Language: "ko", Title: Leaf{NodeID: "title", Text: "제목", NoFactualClaims: true}, Sections: []Section{{SectionID: "section_1", Heading: Leaf{NodeID: "heading_1", Text: "첫째", NoFactualClaims: true}, Nodes: []Leaf{{NodeID: "node_1", Text: long, ClaimIDs: []string{"m1-c01"}}}}, {SectionID: "section_2", Heading: Leaf{NodeID: "heading_2", Text: "둘째", NoFactualClaims: true}, Nodes: []Leaf{{NodeID: "node_2", Text: "마무리", NoFactualClaims: true}}}}}
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseDocument(raw, articleexperiment.RealFixture{MinimumBodyCharacters: 7000, MaximumBodyCharacters: 11000, MaterialTruthClaims: 1}, []string{"m1-c01"})
	if err != nil {
		t.Fatal(err)
	}
	markdown := string(RenderMarkdown(parsed))
	if !strings.HasPrefix(markdown, "# 제목\n\n## 첫째") || !strings.Contains(markdown, "## 둘째") {
		t.Fatalf("markdown=%q", markdown[:80])
	}
}

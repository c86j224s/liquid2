package articlepilot

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/articleexperiment"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const DocumentSchemaVersion = "plasma.article_pilot.document.v1"

var documentIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,79}$`)

// Document is the small target-neutral IL shared by E and A. Narrative planning
// changes author input, not this output schema or the renderer.
type Document struct {
	SchemaVersion string    `json:"schema_version"`
	Language      string    `json:"language"`
	Title         Leaf      `json:"title"`
	Standfirst    *Leaf     `json:"standfirst,omitempty"`
	Sections      []Section `json:"sections"`
}

type Section struct {
	SectionID string `json:"section_id"`
	Heading   Leaf   `json:"heading"`
	Nodes     []Leaf `json:"nodes"`
}

// Leaf is one authored, independently auditable text node.
type Leaf struct {
	NodeID          string   `json:"node_id"`
	Text            string   `json:"text"`
	ClaimIDs        []string `json:"claim_ids"`
	NoFactualClaims bool     `json:"no_factual_claims"`
}

func ParseDocument(raw []byte, fixture articleexperiment.RealFixture, allowedClaimIDs []string) (Document, error) {
	if len(raw) == 0 || len(raw) > 16<<20 || !utf8.Valid(raw) {
		return Document{}, fmt.Errorf("%w: Article document bytes are invalid", producterror.ErrInvalidInput)
	}
	allowedClaims := map[string]bool{}
	for _, claimID := range allowedClaimIDs {
		if !documentIDPattern.MatchString(claimID) || allowedClaims[claimID] {
			return Document{}, fmt.Errorf("%w: allowed Article claim inventory is invalid", producterror.ErrInvalidInput)
		}
		allowedClaims[claimID] = true
	}
	if len(allowedClaims) != fixture.MaterialTruthClaims {
		return Document{}, fmt.Errorf("%w: allowed Article claim inventory is incomplete", producterror.ErrInvalidInput)
	}
	var document Document
	if err := decodeStrict(raw, &document); err != nil {
		return Document{}, err
	}
	if document.SchemaVersion != DocumentSchemaVersion || document.Language != "ko" || len(document.Sections) < 2 || len(document.Sections) > 12 {
		return Document{}, fmt.Errorf("%w: Article document envelope is invalid", producterror.ErrInvalidInput)
	}
	seen := map[string]bool{}
	if err := validateLeaf(document.Title, seen, allowedClaims); err != nil {
		return Document{}, err
	}
	if document.Standfirst != nil {
		if err := validateLeaf(*document.Standfirst, seen, allowedClaims); err != nil {
			return Document{}, err
		}
	}
	for _, section := range document.Sections {
		if !documentIDPattern.MatchString(section.SectionID) || seen[section.SectionID] || len(section.Nodes) == 0 {
			return Document{}, fmt.Errorf("%w: Article section is invalid", producterror.ErrInvalidInput)
		}
		seen[section.SectionID] = true
		if err := validateLeaf(section.Heading, seen, allowedClaims); err != nil {
			return Document{}, err
		}
		for _, node := range section.Nodes {
			if err := validateLeaf(node, seen, allowedClaims); err != nil {
				return Document{}, err
			}
		}
	}
	markdown := RenderMarkdown(document)
	characters := utf8.RuneCount(markdown)
	if characters < fixture.MinimumBodyCharacters || characters > fixture.MaximumBodyCharacters {
		return Document{}, fmt.Errorf("%w: Article document length is outside the frozen band", producterror.ErrInvalidInput)
	}
	return document, nil
}

func validateLeaf(leaf Leaf, seen, allowedClaims map[string]bool) error {
	text := strings.TrimSpace(leaf.Text)
	if !documentIDPattern.MatchString(leaf.NodeID) || seen[leaf.NodeID] || text == "" || !utf8.ValidString(text) || len([]byte(text)) > 32<<10 || strings.Contains(text, "\n\n") || strings.HasPrefix(text, "#") || strings.HasPrefix(text, "- ") || strings.HasPrefix(text, "* ") || strings.HasPrefix(text, "> ") || strings.HasPrefix(text, "```") {
		return fmt.Errorf("%w: Article leaf is invalid", producterror.ErrInvalidInput)
	}
	seen[leaf.NodeID] = true
	if len(leaf.ClaimIDs) > 12 || leaf.NoFactualClaims == (len(leaf.ClaimIDs) > 0) {
		return fmt.Errorf("%w: Article leaf factual coverage is invalid", producterror.ErrInvalidInput)
	}
	claimSeen := map[string]bool{}
	for _, claimID := range leaf.ClaimIDs {
		if !allowedClaims[claimID] || claimSeen[claimID] {
			return fmt.Errorf("%w: Article leaf claim binding is invalid", producterror.ErrInvalidInput)
		}
		claimSeen[claimID] = true
	}
	return nil
}

func RenderMarkdown(document Document) []byte {
	var output strings.Builder
	output.WriteString("# ")
	output.WriteString(strings.TrimSpace(document.Title.Text))
	output.WriteString("\n\n")
	if document.Standfirst != nil {
		output.WriteString(strings.TrimSpace(document.Standfirst.Text))
		output.WriteString("\n\n")
	}
	for _, section := range document.Sections {
		output.WriteString("## ")
		output.WriteString(strings.TrimSpace(section.Heading.Text))
		output.WriteString("\n\n")
		for _, node := range section.Nodes {
			output.WriteString(strings.TrimSpace(node.Text))
			output.WriteString("\n\n")
		}
	}
	return []byte(strings.TrimSpace(output.String()) + "\n")
}

func documentLeaf(document *Document, nodeID string) (*Leaf, bool) {
	if document.Title.NodeID == nodeID {
		return &document.Title, true
	}
	if document.Standfirst != nil && document.Standfirst.NodeID == nodeID {
		return document.Standfirst, true
	}
	for sectionIndex := range document.Sections {
		section := &document.Sections[sectionIndex]
		if section.Heading.NodeID == nodeID {
			return &section.Heading, true
		}
		for nodeIndex := range section.Nodes {
			if section.Nodes[nodeIndex].NodeID == nodeID {
				return &section.Nodes[nodeIndex], true
			}
		}
	}
	return nil, false
}

package reportilcontract

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	AuthorDocumentSchemaVersion         = "plasma.report_il.author_document.experimental.v1"
	LongFormAuthorDocumentSchemaVersion = "plasma.report_il.author_document.experimental.v3"
	AuthorDocumentMediaType             = "application/vnd.plasma.report-il-author+json"
	MaxAuthorDocumentBytes              = 512 * 1024
	MaxAuthorSections                   = 16
	MinLongFormParts                    = 2
	MaxLongFormParts                    = 5
	MinLongFormSections                 = 6
	MaxLongFormSections                 = 14
)

var (
	authorDocumentLanguagePattern = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
	authorPartKeyPattern          = regexp.MustCompile(`^part_[0-9]{3}$`)
	authorLongFormSectionPattern  = regexp.MustCompile(`^part_[0-9]{3}\.section_[0-9]{3}$`)
)

// AuthorDocument is the complete manuscript finalized by the request-local MCP
// authoring workspace. Standard reports use Sections. Long-form reports use the
// explicit Parts hierarchy so the authoritative manuscript retains its Part and
// Section structure through every edit and projection.
type AuthorDocument struct {
	SchemaVersion string          `json:"schema_version"`
	Title         string          `json:"title"`
	Language      string          `json:"language"`
	Sections      []AuthorSection `json:"sections,omitempty"`
	Parts         []AuthorPart    `json:"parts,omitempty"`
}

type AuthorPart struct {
	PartKey  string          `json:"part_key"`
	Title    string          `json:"title"`
	Sections []AuthorSection `json:"sections"`
}

type AuthorSection struct {
	SectionKey string        `json:"section_key"`
	Title      string        `json:"title"`
	Blocks     []AuthorBlock `json:"blocks"`
}

type AuthorBlock struct {
	BlockKey             string          `json:"block_key"`
	Kind                 string          `json:"kind"`
	Prose                string          `json:"prose,omitempty"`
	Items                []string        `json:"items,omitempty"`
	Code                 string          `json:"code,omitempty"`
	Language             *string         `json:"language,omitempty"`
	Table                *AuthorTable    `json:"table,omitempty"`
	Equation             *AuthorEquation `json:"equation,omitempty"`
	EditorialAccountKeys []string        `json:"editorial_account_keys,omitempty"`
	EvidenceSourceKeys   []string        `json:"evidence_source_keys"`
}

type AuthorTable struct {
	Caption *string          `json:"caption"`
	Columns []string         `json:"columns"`
	Rows    []AuthorTableRow `json:"rows"`
}

type AuthorTableRow struct {
	Cells []string `json:"cells"`
}

type AuthorEquation struct {
	Expression string `json:"expression"`
	Notation   string `json:"notation"`
}

// AuthorWorkspaceReceipt is content-free proof that one provider session
// finalized a server-owned manuscript artifact through the MCP workspace.
type AuthorWorkspaceReceipt struct {
	WorkspaceID  string
	ArtifactID   string
	SHA256       string
	ByteSize     int
	Revision     int
	Stage        string
	Replacements int
}

func ValidateAuthorDocument(document AuthorDocument, catalog SourceCatalog) error {
	return validateAuthorDocument(document, catalog, true, false)
}

// ValidateLongFormAuthorFragment validates one finalized v3 Section or Part
// artifact whose stable keys retain their coordinates in the complete plan.
func ValidateLongFormAuthorFragment(document AuthorDocument, catalog SourceCatalog) error {
	if document.SchemaVersion != LongFormAuthorDocumentSchemaVersion {
		return fmt.Errorf("long-form author fragment schema version is invalid")
	}
	if err := validateAuthorDocument(document, catalog, false, false); err != nil {
		return err
	}
	return visitAuthorSections(document, func(section AuthorSection) error {
		if len(section.Blocks) == 0 {
			return fmt.Errorf("finalized long-form author section is empty")
		}
		return nil
	})
}

// ValidateLongFormAuthorDocument enforces the publishable long-form inventory.
func ValidateLongFormAuthorDocument(document AuthorDocument, catalog SourceCatalog) error {
	if document.SchemaVersion != LongFormAuthorDocumentSchemaVersion {
		return fmt.Errorf("long-form author document schema version is invalid")
	}
	return validateAuthorDocument(document, catalog, true, true)
}

func ValidateAuthorDocumentWithMemory(document AuthorDocument, memory EditorialMemory, catalog SourceCatalog) error {
	if err := ValidateAuthorDocument(document, catalog); err != nil {
		return err
	}
	return validateAuthorDocumentMemoryBindings(document, memory, catalog)
}

func ValidateLongFormAuthorFragmentWithMemory(document AuthorDocument, memory EditorialMemory, catalog SourceCatalog) error {
	if err := ValidateLongFormAuthorFragment(document, catalog); err != nil {
		return err
	}
	return validateAuthorDocumentMemoryBindings(document, memory, catalog)
}

func validateAuthorDocumentMemoryBindings(document AuthorDocument, memory EditorialMemory, catalog SourceCatalog) error {
	if err := ValidateEditorialMemory(memory, catalog); err != nil {
		return err
	}
	return visitAuthorBlocks(document, func(block AuthorBlock) error {
		sourceKeys, err := EditorialAccountSourceKeys(memory, block.EditorialAccountKeys)
		if err != nil {
			return err
		}
		if len(sourceKeys) != len(block.EvidenceSourceKeys) {
			return fmt.Errorf("author document block source binding differs from editorial accounts")
		}
		for index := range sourceKeys {
			if sourceKeys[index] != block.EvidenceSourceKeys[index] {
				return fmt.Errorf("author document block source binding differs from editorial accounts")
			}
		}
		return nil
	})
}

// ValidateAuthorDocumentPartial validates an in-progress MCP manuscript after
// each append or edit without requiring the final section or block minimum yet.
func ValidateAuthorDocumentPartial(document AuthorDocument, catalog SourceCatalog) error {
	return validateAuthorDocument(document, catalog, false, false)
}

func validateAuthorDocument(document AuthorDocument, catalog SourceCatalog, complete, longFormProduct bool) error {
	if err := ValidateSourceCatalog(catalog); err != nil {
		return err
	}
	if document.SchemaVersion != AuthorDocumentSchemaVersion &&
		document.SchemaVersion != LongFormAuthorDocumentSchemaVersion {
		return fmt.Errorf("author document schema version is invalid")
	}
	if strings.TrimSpace(document.Title) == "" || !utf8.ValidString(document.Title) {
		return fmt.Errorf("author document title is invalid")
	}
	if !authorDocumentLanguagePattern.MatchString(document.Language) {
		return fmt.Errorf("author document language is invalid")
	}

	seenSections := map[string]bool{}
	seenBlocks := map[string]bool{}
	sectionCount := 0
	firstBlock := true
	validateSection := func(section AuthorSection, wantSectionKey string) error {
		if section.SectionKey != wantSectionKey || seenSections[section.SectionKey] ||
			strings.TrimSpace(section.Title) == "" || !utf8.ValidString(section.Title) ||
			complete && len(section.Blocks) == 0 {
			return fmt.Errorf("author document section is invalid")
		}
		seenSections[section.SectionKey] = true
		sectionCount++
		for blockIndex, block := range section.Blocks {
			wantBlockKey := fmt.Sprintf("%s.block_%03d", section.SectionKey, blockIndex+1)
			if block.BlockKey != wantBlockKey || seenBlocks[block.BlockKey] {
				return fmt.Errorf("author document block identity is invalid")
			}
			seenBlocks[block.BlockKey] = true
			if firstBlock && block.Kind != "prose" {
				return fmt.Errorf("author document must begin with prose")
			}
			firstBlock = false
			if err := validateAuthorBlock(block, catalog); err != nil {
				return err
			}
		}
		return nil
	}

	switch document.SchemaVersion {
	case AuthorDocumentSchemaVersion:
		if len(document.Parts) != 0 {
			return fmt.Errorf("standard author document contains long-form parts")
		}
		minimumSections := 1
		if complete {
			minimumSections = 2
		}
		if len(document.Sections) < minimumSections || len(document.Sections) > MaxAuthorSections {
			return fmt.Errorf("author document section inventory is invalid")
		}
		for sectionIndex, section := range document.Sections {
			if err := validateSection(section, fmt.Sprintf("section_%03d", sectionIndex+1)); err != nil {
				return err
			}
		}
	case LongFormAuthorDocumentSchemaVersion:
		if len(document.Sections) != 0 || len(document.Parts) == 0 || len(document.Parts) > MaxLongFormParts {
			return fmt.Errorf("long-form author document part inventory is invalid")
		}
		if longFormProduct && (len(document.Parts) < MinLongFormParts || len(document.Parts) > MaxLongFormParts) {
			return fmt.Errorf("long-form author document part inventory is invalid")
		}
		seenParts := map[string]bool{}
		for partIndex, part := range document.Parts {
			wantPartKey := fmt.Sprintf("part_%03d", partIndex+1)
			if !complete && len(document.Parts) == 1 {
				wantPartKey = part.PartKey
			}
			if part.PartKey != wantPartKey || !authorPartKeyPattern.MatchString(part.PartKey) || seenParts[part.PartKey] ||
				strings.TrimSpace(part.Title) == "" || !utf8.ValidString(part.Title) || len(part.Sections) == 0 {
				return fmt.Errorf("long-form author document part is invalid")
			}
			seenParts[part.PartKey] = true
			for sectionIndex, section := range part.Sections {
				wantSectionKey := fmt.Sprintf("%s.section_%03d", part.PartKey, sectionIndex+1)
				if !complete && len(part.Sections) == 1 {
					wantSectionKey = section.SectionKey
				}
				if !strings.HasPrefix(wantSectionKey, part.PartKey+".") ||
					!authorLongFormSectionPattern.MatchString(wantSectionKey) {
					return fmt.Errorf("author document section is invalid")
				}
				if err := validateSection(section, wantSectionKey); err != nil {
					return err
				}
			}
		}
		if sectionCount > MaxLongFormSections || longFormProduct && sectionCount < MinLongFormSections {
			return fmt.Errorf("long-form author document section inventory is invalid")
		}
	}
	encoded, err := json.Marshal(document)
	if err != nil || len(encoded) > MaxAuthorDocumentBytes {
		return fmt.Errorf("author document exceeds the byte ceiling")
	}
	return nil
}

func visitAuthorSections(document AuthorDocument, visit func(AuthorSection) error) error {
	for _, section := range document.Sections {
		if err := visit(section); err != nil {
			return err
		}
	}
	for _, part := range document.Parts {
		for _, section := range part.Sections {
			if err := visit(section); err != nil {
				return err
			}
		}
	}
	return nil
}

func visitAuthorBlocks(document AuthorDocument, visit func(AuthorBlock) error) error {
	return visitAuthorSections(document, func(section AuthorSection) error {
		for _, block := range section.Blocks {
			if err := visit(block); err != nil {
				return err
			}
		}
		return nil
	})
}

func validateAuthorBlock(block AuthorBlock, catalog SourceCatalog) error {
	if len(block.EvidenceSourceKeys) == 0 {
		return fmt.Errorf("author document block requires source citations")
	}
	seenSources := make(map[string]bool, len(block.EvidenceSourceKeys))
	for _, sourceKey := range block.EvidenceSourceKeys {
		if seenSources[sourceKey] {
			return fmt.Errorf("author document block contains duplicate source citations")
		}
		if _, ok := catalog.Entry(sourceKey); !ok {
			return fmt.Errorf("author document block contains an unavailable source citation")
		}
		seenSources[sourceKey] = true
	}
	validText := func(value string) bool {
		return strings.TrimSpace(value) != "" && utf8.ValidString(value)
	}
	switch block.Kind {
	case "prose", "quote", "callout":
		if !validText(block.Prose) || len(block.Items) != 0 || block.Code != "" || block.Language != nil || block.Table != nil || block.Equation != nil {
			return fmt.Errorf("author document prose block is invalid")
		}
		if containsAuthorProseMarkup(block.Prose) {
			return fmt.Errorf("author document prose block contains presentation markup")
		}
	case "list":
		if block.Prose != "" || len(block.Items) == 0 || block.Code != "" || block.Language != nil || block.Table != nil || block.Equation != nil {
			return fmt.Errorf("author document list block is invalid")
		}
		for _, item := range block.Items {
			if !validText(item) {
				return fmt.Errorf("author document list item is invalid")
			}
		}
	case "code":
		if block.Prose != "" || len(block.Items) != 0 || block.Code == "" || !utf8.ValidString(block.Code) || block.Table != nil || block.Equation != nil {
			return fmt.Errorf("author document code block is invalid")
		}
		if block.Language != nil && !validText(*block.Language) {
			return fmt.Errorf("author document code language is invalid")
		}
	case "table":
		if block.Prose != "" || len(block.Items) != 0 || block.Code != "" || block.Language != nil || block.Table == nil || block.Equation != nil {
			return fmt.Errorf("author document table block is invalid")
		}
		if err := validateAuthorTable(*block.Table); err != nil {
			return err
		}
	case "equation":
		if block.Prose != "" || len(block.Items) != 0 || block.Code != "" || block.Language != nil || block.Table != nil || block.Equation == nil ||
			!validText(block.Equation.Expression) || block.Equation.Notation != "latex" {
			return fmt.Errorf("author document equation block is invalid")
		}
	default:
		return fmt.Errorf("author document block kind is invalid")
	}
	return nil
}

func containsAuthorProseMarkup(value string) bool {
	if strings.Contains(value, "**") ||
		strings.Contains(value, "```") ||
		strings.Contains(value, `\[`) ||
		strings.Contains(value, `\]`) {
		return true
	}
	for _, line := range strings.Split(value, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") ||
			strings.HasPrefix(trimmed, "## ") ||
			strings.HasPrefix(trimmed, "### ") ||
			strings.HasPrefix(trimmed, "- ") ||
			strings.HasPrefix(trimmed, "* ") {
			return true
		}
	}
	return false
}

func validateAuthorTable(table AuthorTable) error {
	if table.Caption != nil && (strings.TrimSpace(*table.Caption) == "" || !utf8.ValidString(*table.Caption)) {
		return fmt.Errorf("author document table caption is invalid")
	}
	if len(table.Columns) < 2 || len(table.Columns) > 4 || len(table.Rows) == 0 {
		return fmt.Errorf("author document table shape is invalid")
	}
	for _, column := range table.Columns {
		if strings.TrimSpace(column) == "" || !utf8.ValidString(column) {
			return fmt.Errorf("author document table column is invalid")
		}
	}
	for _, row := range table.Rows {
		if len(row.Cells) != len(table.Columns) {
			return fmt.Errorf("author document table row width is invalid")
		}
		for _, cell := range row.Cells {
			if strings.TrimSpace(cell) == "" || !utf8.ValidString(cell) {
				return fmt.Errorf("author document table cell is invalid")
			}
		}
	}
	return nil
}

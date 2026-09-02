package reportilcontract

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	LongFormPlanSchemaVersion = "plasma.report_il.long_form_plan.experimental.v3"
	LongFormPlanMediaType     = "application/vnd.plasma.report-il-long-form-plan+json"
	MaxLongFormPlanBytes      = 128 * 1024

	LongFormSectionRoleBody       = "body"
	LongFormSectionRoleConclusion = "conclusion"

	LongFormRepresentationTable         = "table"
	LongFormRepresentationCode          = "code"
	LongFormRepresentationEquation      = "equation"
	LongFormRepresentationWorkedExample = "worked_example"
	LongFormRepresentationBenchmark     = "benchmark"
	LongFormRepresentationChecklist     = "checklist"
	LongFormRepresentationDiagram       = "diagram"
)

type LongFormPlan struct {
	SchemaVersion string         `json:"schema_version"`
	Title         string         `json:"title"`
	Language      string         `json:"language"`
	Summary       string         `json:"summary"`
	Parts         []LongFormPart `json:"parts"`
}

type LongFormPart struct {
	PartKey  string            `json:"part_key"`
	Title    string            `json:"title"`
	Purpose  string            `json:"purpose"`
	Sections []LongFormSection `json:"sections"`
}

type LongFormSection struct {
	SectionKey           string   `json:"section_key"`
	Title                string   `json:"title"`
	Purpose              string   `json:"purpose"`
	Role                 string   `json:"role"`
	Representations      []string `json:"representations"`
	EditorialAccountKeys []string `json:"editorial_account_keys,omitempty"`
	EvidenceSourceKeys   []string `json:"evidence_source_keys"`
}

type LongFormPlanReceipt struct {
	ArtifactID string
	SHA256     string
	ByteSize   int
	Parts      int
	Sections   int
}

func ValidateLongFormPlan(plan LongFormPlan, memory *EditorialMemory, catalog SourceCatalog) error {
	if err := ValidateSourceCatalog(catalog); err != nil {
		return err
	}
	if plan.SchemaVersion != LongFormPlanSchemaVersion ||
		strings.TrimSpace(plan.Title) == "" || !utf8.ValidString(plan.Title) ||
		!authorDocumentLanguagePattern.MatchString(plan.Language) ||
		strings.TrimSpace(plan.Summary) == "" || !utf8.ValidString(plan.Summary) ||
		len(plan.Parts) < MinLongFormParts || len(plan.Parts) > MaxLongFormParts {
		return fmt.Errorf("long-form plan envelope is invalid")
	}
	if memory != nil {
		if err := ValidateEditorialMemory(*memory, catalog); err != nil {
			return err
		}
	}
	sectionCount := 0
	conclusionCount := 0
	seenSources := map[string]bool{}
	seenAccounts := map[string]bool{}
	for partIndex, part := range plan.Parts {
		if part.PartKey != fmt.Sprintf("part_%03d", partIndex+1) ||
			strings.TrimSpace(part.Title) == "" || !utf8.ValidString(part.Title) ||
			strings.TrimSpace(part.Purpose) == "" || !utf8.ValidString(part.Purpose) || len(part.Sections) == 0 {
			return fmt.Errorf("long-form plan part is invalid")
		}
		for sectionIndex, section := range part.Sections {
			sectionCount++
			lastSection := partIndex == len(plan.Parts)-1 && sectionIndex == len(part.Sections)-1
			if section.SectionKey != fmt.Sprintf("%s.section_%03d", part.PartKey, sectionIndex+1) ||
				strings.TrimSpace(section.Title) == "" || !utf8.ValidString(section.Title) ||
				strings.TrimSpace(section.Purpose) == "" || !utf8.ValidString(section.Purpose) ||
				section.Role != LongFormSectionRoleBody && section.Role != LongFormSectionRoleConclusion ||
				(section.Role == LongFormSectionRoleConclusion) != lastSection ||
				len(section.EvidenceSourceKeys) == 0 {
				return fmt.Errorf("long-form plan section is invalid")
			}
			if section.Role == LongFormSectionRoleConclusion {
				conclusionCount++
			}
			if section.Representations == nil {
				return fmt.Errorf("long-form plan section representations must be an array")
			}
			seenRepresentations := map[string]bool{}
			for _, representation := range section.Representations {
				if !validLongFormRepresentation(representation) || seenRepresentations[representation] {
					return fmt.Errorf("long-form plan section representation is invalid")
				}
				seenRepresentations[representation] = true
			}
			sectionSources := map[string]bool{}
			for _, sourceKey := range section.EvidenceSourceKeys {
				if sectionSources[sourceKey] {
					return fmt.Errorf("long-form plan section contains duplicate sources")
				}
				if _, ok := catalog.Entry(sourceKey); !ok {
					return fmt.Errorf("long-form plan section contains an unavailable source")
				}
				sectionSources[sourceKey] = true
				seenSources[sourceKey] = true
			}
			if memory == nil {
				if len(section.EditorialAccountKeys) != 0 {
					return fmt.Errorf("direct long-form plan contains editorial accounts")
				}
				continue
			}
			sourceKeys, err := EditorialAccountSourceKeys(*memory, section.EditorialAccountKeys)
			if err != nil || !equalLongFormStrings(sourceKeys, section.EvidenceSourceKeys) {
				return fmt.Errorf("long-form plan section source binding differs from editorial accounts")
			}
			for _, accountKey := range section.EditorialAccountKeys {
				seenAccounts[accountKey] = true
			}
		}
	}
	if sectionCount < MinLongFormSections || sectionCount > MaxLongFormSections || conclusionCount != 1 || len(seenSources) == 0 {
		return fmt.Errorf("long-form plan section inventory is invalid")
	}
	if memory != nil {
		for _, account := range memory.Accounts {
			if account.Importance == "essential" && !seenAccounts[account.AccountKey] {
				return fmt.Errorf("long-form plan omits an essential editorial account")
			}
		}
	}
	encoded, err := json.Marshal(plan)
	if err != nil || len(encoded) > MaxLongFormPlanBytes {
		return fmt.Errorf("long-form plan exceeds the byte ceiling")
	}
	return nil
}

func (plan LongFormPlan) Find(partKey, sectionKey string) (LongFormPart, LongFormSection, bool) {
	for _, part := range plan.Parts {
		if part.PartKey != partKey {
			continue
		}
		for _, section := range part.Sections {
			if section.SectionKey == sectionKey {
				return part, section, true
			}
		}
	}
	return LongFormPart{}, LongFormSection{}, false
}

func LongFormPlanSectionCount(plan LongFormPlan) int {
	count := 0
	for _, part := range plan.Parts {
		count += len(part.Sections)
	}
	return count
}

// ValidateLongFormRepresentationObligations verifies the planned forms that
// have an unambiguous first-class author block. Other representation labels
// remain reader-facing authoring obligations because they may legitimately be
// realized across prose and more than one structured block.
func ValidateLongFormRepresentationObligations(planned LongFormSection, authored AuthorSection) error {
	realized := make(map[string]bool, len(authored.Blocks))
	for _, block := range authored.Blocks {
		realized[block.Kind] = true
	}
	for _, representation := range planned.Representations {
		switch representation {
		case LongFormRepresentationTable,
			LongFormRepresentationCode,
			LongFormRepresentationEquation:
			if !realized[representation] {
				return fmt.Errorf("planned long-form %s representation is missing", representation)
			}
		}
	}
	return nil
}

func validLongFormRepresentation(value string) bool {
	switch value {
	case LongFormRepresentationTable,
		LongFormRepresentationCode,
		LongFormRepresentationEquation,
		LongFormRepresentationWorkedExample,
		LongFormRepresentationBenchmark,
		LongFormRepresentationChecklist,
		LongFormRepresentationDiagram:
		return true
	default:
		return false
	}
}

func equalLongFormStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

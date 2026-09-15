package reportdocument

import (
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"strings"
)

type ScopeRecords struct {
	Claims    []researchrecords.ClaimRecord
	Evidence  []researchrecords.EvidenceRecord
	Questions []researchrecords.QuestionRecord
	Options   []researchrecords.OptionRecord
}

func addUnique(values *[]string, value string) {
	if value == "" {
		return
	}
	for _, existing := range *values {
		if existing == value {
			return
		}
	}
	*values = append(*values, value)
}

func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", producterror.ErrInvalidInput, prefix)
	}
	return nil
}

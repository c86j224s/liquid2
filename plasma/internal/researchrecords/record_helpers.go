package researchrecords

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func normalizeIDList(prefix string, ids []string) ([]string, error) {
	normalized := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if err := validateID(prefix, trimmed); err != nil {
			return nil, err
		}
		if _, ok := seen[trimmed]; ok {
			return nil, fmt.Errorf("%w: duplicate id", producterror.ErrInvalidInput)
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized, nil
}

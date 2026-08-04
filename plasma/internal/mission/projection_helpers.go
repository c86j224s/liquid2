package mission

import (
	"fmt"
	"slices"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func validateMissionID(id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, "mis_") || len(trimmed) <= len("mis_") {
		return fmt.Errorf("%w: id must start with mis_", producterror.ErrInvalidInput)
	}
	return nil
}

func approvalProducer(producer ledger.Producer) bool {
	return producer.Type == "user" || producer.Type == "steering_chat"
}

func normalizeScope(scope Scope) Scope {
	return Scope{Included: trimEntries(scope.Included), Excluded: trimEntries(scope.Excluded)}
}

func trimEntries(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func emptyScope(scope Scope) bool {
	return len(scope.Included) == 0 && len(scope.Excluded) == 0
}

func equalScopes(left, right Scope) bool {
	return slices.Equal(left.Included, right.Included) && slices.Equal(left.Excluded, right.Excluded)
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

func removeValue(values *[]string, value string) {
	if value == "" {
		return
	}
	next := (*values)[:0]
	for _, existing := range *values {
		if existing != value {
			next = append(next, existing)
		}
	}
	*values = next
}

package web

import (
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func longFormStageFailure(kind, planID string, part, section int, cause error) error {
	return reporting.NewStageFailure(kind, planID, part, section, cause)
}

package web

import (
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	workflowplan "github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
)

func sectionFanoutPlanActivationFlags(event ledger.Event) (bool, bool, error) {
	return workflowplan.LongFormActivationFlags(event)
}

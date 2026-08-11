package web

import (
	"github.com/c86j224s/liquid2/plasma/internal/app"
	workflowplan "github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
)

func sectionFanoutPlanActivationFlags(event app.LedgerEvent) (bool, bool, error) {
	return workflowplan.LongFormActivationFlags(event)
}

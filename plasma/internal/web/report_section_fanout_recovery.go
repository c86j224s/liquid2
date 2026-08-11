package web

import (
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// sectionFanoutPartPlanningParent는 Web recovery projection이 기존 canonical
// plan event의 part-planning parent만 읽는 adapter다. Prefix stage 실행, fork,
// provider 호출, 병렬 정책은 reportworkflow가 소유하며 이 함수는 장부를 쓰지 않는다.
func sectionFanoutPartPlanningParent(event app.LedgerEvent, pendingEventID string) (reporting.PartPlanParentState, error) {
	parent, ok, err := reporting.DecodePartPlanParent(event, pendingEventID, event.EventID)
	if err != nil {
		return reporting.PartPlanParentState{}, err
	}
	if !ok {
		return reporting.PartPlanParentState{}, fmt.Errorf("%w: Part planning parent is missing", app.ErrConflict)
	}
	return parent, nil
}

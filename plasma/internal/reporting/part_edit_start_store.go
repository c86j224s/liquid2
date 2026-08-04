package reporting

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// StartPartEdit는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func StartPartEdit(ctx context.Context, store PartEditStartStore, eventID string, binding PartEditBinding) (app.LedgerEvent, bool, error) {
	binding = normalizePartEditBinding(binding)
	if err := ValidatePartEditBinding(binding); err != nil {
		return app.LedgerEvent{}, false, err
	}
	return store.AppendEventConditionally(ctx, binding.MissionID, func(events []app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		if err := validatePartEditLineage(events, binding); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if err := validatePartEditRequirementMap(events, binding); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if existing, ok, err := canonicalPartEditStartEvent(events, binding); ok || err != nil {
			return app.AppendEventRequest{}, existing, false, err
		}
		return BuildPartEditStartedAppendRequest(eventID, binding), app.LedgerEvent{}, true, nil
	})
}

func canonicalPartEditStartEvent(events []app.LedgerEvent, binding PartEditBinding) (app.LedgerEvent, bool, error) {
	var found app.LedgerEvent
	count := 0
	for _, candidate := range events {
		if candidate.EventType != PartEditStartedEventType || candidate.CorrelationID != binding.IdempotencyKey {
			continue
		}
		if !partEditStartEventMatches(candidate, binding) {
			return app.LedgerEvent{}, false, fmt.Errorf("%w: part edit start binding differs", app.ErrConflict)
		}
		found, count = candidate, count+1
	}
	if count > 1 {
		return app.LedgerEvent{}, false, fmt.Errorf("%w: multiple part edit starts match binding", app.ErrConflict)
	}
	return found, count == 1, nil
}

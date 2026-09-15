package reporting

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// StartPartEdit는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func StartPartEdit(ctx context.Context, store PartEditStartStore, eventID string, binding PartEditBinding) (ledger.Event, bool, error) {
	binding = normalizePartEditBinding(binding)
	if err := ValidatePartEditBinding(binding); err != nil {
		return ledger.Event{}, false, err
	}
	return store.AppendEventConditionally(ctx, binding.MissionID, func(events []ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error) {
		if err := validatePartEditLineage(events, binding); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if err := validatePartEditRequirementMap(events, binding); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if existing, ok, err := canonicalPartEditStartEvent(events, binding); ok || err != nil {
			return ledger.AppendRequest{}, existing, false, err
		}
		return BuildPartEditStartedAppendRequest(eventID, binding), ledger.Event{}, true, nil
	})
}

func canonicalPartEditStartEvent(events []ledger.Event, binding PartEditBinding) (ledger.Event, bool, error) {
	var found ledger.Event
	count := 0
	for _, candidate := range events {
		if candidate.EventType != PartEditStartedEventType || candidate.CorrelationID != binding.IdempotencyKey {
			continue
		}
		if !partEditStartEventMatches(candidate, binding) {
			return ledger.Event{}, false, fmt.Errorf("%w: part edit start binding differs", producterror.ErrConflict)
		}
		found, count = candidate, count+1
	}
	if count > 1 {
		return ledger.Event{}, false, fmt.Errorf("%w: multiple part edit starts match binding", producterror.ErrConflict)
	}
	return found, count == 1, nil
}

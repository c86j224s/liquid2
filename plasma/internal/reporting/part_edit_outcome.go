package reporting

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func LoadPartEditOutcome(ctx context.Context, store PartEditOutcomeStore, contract PartEditOutcomeContract) (PartEditResult, bool, error) {
	contract = normalizePartEditOutcomeContract(contract)
	if err := validatePartEditOutcomeContract(contract); err != nil {
		return PartEditResult{}, false, err
	}
	events, err := store.ListEvents(ctx, contract.MissionID)
	if err != nil {
		return PartEditResult{}, false, err
	}
	acceptedPending, err := longFormPendingLineage(events, contract.CurrentPendingEventID)
	if err != nil {
		return PartEditResult{}, false, err
	}
	outcomes, err := validPartEditOutcomes(ctx, store, events, acceptedPending, contract)
	if err != nil {
		return PartEditResult{}, false, err
	}
	if len(outcomes) == 0 {
		return PartEditResult{}, false, nil
	}
	if len(outcomes) != 1 {
		return PartEditResult{}, false, fmt.Errorf("%w: multiple valid part edit outcomes match binding", app.ErrConflict)
	}
	return outcomes[0], true, nil
}

func validPartEditOutcomes(ctx context.Context, store PartEditOutcomeStore, events []app.LedgerEvent, acceptedPending map[string]bool, contract PartEditOutcomeContract) ([]PartEditResult, error) {
	contract = normalizePartEditOutcomeContract(contract)
	if err := validatePartEditOutcomeContract(contract); err != nil {
		return nil, err
	}
	results := []PartEditResult{}
	for _, event := range events {
		if event.EventType != PartEditedEventType {
			continue
		}
		result, valid, err := validatePartEditOutcome(ctx, store, events, acceptedPending, contract, event)
		if err != nil {
			return nil, err
		}
		if valid {
			results = append(results, result)
		}
	}
	return results, nil
}

func validatePartEditOutcome(ctx context.Context, store PartEditOutcomeStore, events []app.LedgerEvent, acceptedPending map[string]bool, contract PartEditOutcomeContract, event app.LedgerEvent) (PartEditResult, bool, error) {
	binding, ok := partEditBindingFromEditedEvent(event)
	if !ok {
		return PartEditResult{}, false, nil
	}
	if !acceptedPending[binding.PendingEventID] {
		return PartEditResult{}, false, nil
	}
	startBinding, ok, err := partEditStartedBindingForOutcome(events, acceptedPending, event)
	if err != nil {
		return PartEditResult{}, false, err
	}
	if !ok || !partEditOutcomeMatchesContract(startBinding, contract) {
		return PartEditResult{}, false, nil
	}
	if err := validatePartEditLineage(events, startBinding); err != nil {
		return PartEditResult{}, false, nil
	}
	if !partEditEventMatches(event, startBinding) || !partEditRequirementMapMatches(events, acceptedPending, startBinding) {
		return PartEditResult{}, false, nil
	}
	result, err := partEditResultFromEvent(ctx, store, startBinding, event, true)
	if err != nil {
		return PartEditResult{}, false, nil
	}
	return result, true, nil
}

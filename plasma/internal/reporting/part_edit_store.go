package reporting

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func LoadPartEdit(ctx context.Context, store PartEditStore, binding PartEditBinding) (PartEditResult, bool, error) {
	binding = normalizePartEditBinding(binding)
	if err := ValidatePartEditBinding(binding); err != nil {
		return PartEditResult{}, false, err
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return PartEditResult{}, false, err
	}
	if err := validatePartEditLineage(events, binding); err != nil {
		return PartEditResult{}, false, err
	}
	var found app.LedgerEvent
	count := 0
	for _, event := range events {
		if event.EventType != PartEditedEventType || event.CorrelationID != binding.IdempotencyKey {
			continue
		}
		if !partEditEventMatches(event, binding) {
			return PartEditResult{}, false, fmt.Errorf("%w: part edit replay binding differs", app.ErrConflict)
		}
		found, count = event, count+1
	}
	if count == 0 {
		return PartEditResult{}, false, nil
	}
	if count != 1 {
		return PartEditResult{}, false, fmt.Errorf("%w: multiple part edits match binding", app.ErrConflict)
	}
	if _, ok, err := canonicalPartEditStartEvent(events, binding); err != nil {
		return PartEditResult{}, false, err
	} else if !ok {
		return PartEditResult{}, false, fmt.Errorf("%w: matching Part edit start is missing", app.ErrConflict)
	}
	if err := validatePartEditRequirementMap(events, binding); err != nil {
		return PartEditResult{}, false, err
	}
	result, err := partEditResultFromEvent(ctx, store, binding, found, true)
	return result, err == nil, err
}

func FinalizePartEdit(ctx context.Context, store PartEditStore, binding PartEditBinding, eventID string, markdown string, operationCount int) (PartEditResult, error) {
	binding = normalizePartEditBinding(binding)
	if err := ValidatePartEditBinding(binding); err != nil {
		return PartEditResult{}, err
	}
	if existing, ok, err := LoadPartEdit(ctx, store, binding); err != nil {
		return PartEditResult{}, err
	} else if ok {
		return existing, nil
	}
	if strings.TrimSpace(markdown) == "" || operationCount < 0 {
		return PartEditResult{}, fmt.Errorf("%w: edited part Markdown is invalid", app.ErrInvalidInput)
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return PartEditResult{}, err
	}
	if err := validatePartEditLineage(events, binding); err != nil {
		return PartEditResult{}, err
	}
	if _, ok, err := canonicalPartEditStartEvent(events, binding); err != nil {
		return PartEditResult{}, err
	} else if !ok {
		return PartEditResult{}, fmt.Errorf("%w: matching Part edit start is missing", app.ErrConflict)
	}
	if err := validatePartEditRequirementMap(events, binding); err != nil {
		return PartEditResult{}, err
	}
	source, err := store.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return PartEditResult{}, err
	}
	if source.MissionID != binding.MissionID || source.MediaType != "text/markdown; charset=utf-8" {
		return PartEditResult{}, fmt.Errorf("%w: source part artifact is foreign or not Markdown", app.ErrConflict)
	}
	if string(source.Content) == markdown {
		event, created, err := store.AppendEventConditionally(ctx, binding.MissionID, func(events []app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
			if err := validatePartEditLineage(events, binding); err != nil {
				return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
			}
			if _, ok, err := canonicalPartEditStartEvent(events, binding); err != nil {
				return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
			} else if !ok {
				return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: matching Part edit start is missing", app.ErrConflict)
			}
			if err := validatePartEditRequirementMap(events, binding); err != nil {
				return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
			}
			if existing, ok, err := canonicalPartEditEvent(events, binding); ok || err != nil {
				return app.AppendEventRequest{}, existing, false, err
			}
			return buildPartEditedAppendRequest(eventID, binding, source, source, operationCount, false), app.LedgerEvent{}, true, nil
		})
		if err != nil {
			return PartEditResult{}, err
		}
		return partEditResultFromEvent(ctx, store, binding, event, !created)
	}
	producer := app.Producer{Type: "agent_session", ID: binding.ProviderSessionID}
	artifactReq := app.CreateRawArtifactRequest{
		ArtifactID: binding.EditedArtifactID, MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename,
		Producer: producer, Content: []byte(markdown),
	}
	artifact, event, created, err := store.CreateRawArtifactWithEventConditionally(ctx, artifactReq, func(events []app.LedgerEvent, artifact app.RawArtifact) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		if err := validatePartEditLineage(events, binding); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if _, ok, err := canonicalPartEditStartEvent(events, binding); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		} else if !ok {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: matching Part edit start is missing", app.ErrConflict)
		}
		if err := validatePartEditRequirementMap(events, binding); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if existing, ok, err := canonicalPartEditEvent(events, binding); ok || err != nil {
			return app.AppendEventRequest{}, existing, false, err
		}
		return buildPartEditedAppendRequest(eventID, binding, source, artifact, operationCount, true), app.LedgerEvent{}, true, nil
	})
	if err != nil {
		if existing, ok, loadErr := LoadPartEdit(ctx, store, binding); ok && loadErr == nil {
			return existing, nil
		}
		return PartEditResult{}, err
	}
	if created {
		return PartEditResult{Artifact: artifact, Event: event}, nil
	}
	return partEditResultFromEvent(ctx, store, binding, event, true)
}

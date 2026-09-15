package reporting

import (
	"context"
	"fmt"
	"strings"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// LoadPartEdit는 part edit 제출 이벤트와 artifact를 장부에서 복원한다.
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
	var found ledger.Event
	count := 0
	for _, event := range events {
		if event.EventType != PartEditedEventType || event.CorrelationID != binding.IdempotencyKey {
			continue
		}
		if !partEditEventMatches(event, binding) {
			return PartEditResult{}, false, fmt.Errorf("%w: part edit replay binding differs", producterror.ErrConflict)
		}
		found, count = event, count+1
	}
	if count == 0 {
		return PartEditResult{}, false, nil
	}
	if count != 1 {
		return PartEditResult{}, false, fmt.Errorf("%w: multiple part edits match binding", producterror.ErrConflict)
	}
	if _, ok, err := canonicalPartEditStartEvent(events, binding); err != nil {
		return PartEditResult{}, false, err
	} else if !ok {
		return PartEditResult{}, false, fmt.Errorf("%w: matching Part edit start is missing", producterror.ErrConflict)
	}
	if err := validatePartEditRequirementMap(events, binding); err != nil {
		return PartEditResult{}, false, err
	}
	result, err := partEditResultFromEvent(ctx, store, binding, found, true)
	return result, err == nil, err
}

// FinalizePartEdit는 part edit 결과를 검증하고 저장용 이벤트 요청으로 만든다.
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
		return PartEditResult{}, fmt.Errorf("%w: edited part Markdown is invalid", producterror.ErrInvalidInput)
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
		return PartEditResult{}, fmt.Errorf("%w: matching Part edit start is missing", producterror.ErrConflict)
	}
	if err := validatePartEditRequirementMap(events, binding); err != nil {
		return PartEditResult{}, err
	}
	source, err := store.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return PartEditResult{}, err
	}
	if source.MissionID != binding.MissionID || source.MediaType != "text/markdown; charset=utf-8" {
		return PartEditResult{}, fmt.Errorf("%w: source part artifact is foreign or not Markdown", producterror.ErrConflict)
	}
	if string(source.Content) == markdown {
		event, created, err := store.AppendEventConditionally(ctx, binding.MissionID, func(events []ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error) {
			if err := validatePartEditLineage(events, binding); err != nil {
				return ledger.AppendRequest{}, ledger.Event{}, false, err
			}
			if _, ok, err := canonicalPartEditStartEvent(events, binding); err != nil {
				return ledger.AppendRequest{}, ledger.Event{}, false, err
			} else if !ok {
				return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("%w: matching Part edit start is missing", producterror.ErrConflict)
			}
			if err := validatePartEditRequirementMap(events, binding); err != nil {
				return ledger.AppendRequest{}, ledger.Event{}, false, err
			}
			if existing, ok, err := canonicalPartEditEvent(events, binding); ok || err != nil {
				return ledger.AppendRequest{}, existing, false, err
			}
			return buildPartEditedAppendRequest(eventID, binding, source, source, operationCount, false), ledger.Event{}, true, nil
		})
		if err != nil {
			return PartEditResult{}, err
		}
		return partEditResultFromEvent(ctx, store, binding, event, !created)
	}
	producer := ledger.Producer{Type: "agent_session", ID: binding.ProviderSessionID}
	artifactReq := artifactcontract.CreateRequest{
		ArtifactID: binding.EditedArtifactID, MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename,
		Producer: producer, Content: []byte(markdown),
	}
	artifact, event, created, err := store.CreateRawArtifactWithEventConditionally(ctx, artifactReq, func(events []ledger.Event, artifact artifactcontract.Raw) (ledger.AppendRequest, ledger.Event, bool, error) {
		if err := validatePartEditLineage(events, binding); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if _, ok, err := canonicalPartEditStartEvent(events, binding); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		} else if !ok {
			return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("%w: matching Part edit start is missing", producterror.ErrConflict)
		}
		if err := validatePartEditRequirementMap(events, binding); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if existing, ok, err := canonicalPartEditEvent(events, binding); ok || err != nil {
			return ledger.AppendRequest{}, existing, false, err
		}
		return buildPartEditedAppendRequest(eventID, binding, source, artifact, operationCount, true), ledger.Event{}, true, nil
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

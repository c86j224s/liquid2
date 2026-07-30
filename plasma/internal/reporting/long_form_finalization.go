package reporting

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func LoadLongFormFinalization(ctx context.Context, store LongFormFinalizationStore, binding LongFormFinalizeBinding) (LongFormFinalizeResult, bool, error) {
	binding = normalizeLongFormFinalizeBinding(binding)
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return LongFormFinalizeResult{}, false, err
	}
	event, count := longFormCanonical(events, binding.PendingEventID)
	if count == 0 {
		return LongFormFinalizeResult{}, false, nil
	}
	if count != 1 {
		return LongFormFinalizeResult{}, false, fmt.Errorf("%w: multiple canonical long-form finalizations", app.ErrConflict)
	}
	result, err := loadLongFormCanonicalResult(ctx, store, binding, event)
	if err != nil {
		return LongFormFinalizeResult{}, false, err
	}
	return result, true, nil
}

func FinalizeLongForm(ctx context.Context, store LongFormFinalizationStore, req LongFormFinalizeRequest) (LongFormFinalizeResult, error) {
	return finalizeLongForm(ctx, store, req, false)
}

func finalizeLongForm(ctx context.Context, store LongFormFinalizationStore, req LongFormFinalizeRequest, allowReaderStyleGate bool) (LongFormFinalizeResult, error) {
	binding := normalizeLongFormFinalizeBinding(req.Binding)
	if err := validateLongFormFinalizeBinding(binding); err != nil {
		return LongFormFinalizeResult{}, err
	}
	if strings.TrimSpace(req.FinalEditPipeline) != "" && (!allowReaderStyleGate || !isSupportedFinalEditPipeline(strings.TrimSpace(req.FinalEditPipeline))) {
		return LongFormFinalizeResult{}, fmt.Errorf("%w: final edit pipeline can only be set by corrective gate finalization", app.ErrInvalidInput)
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	if err := validateLongFormFinalPipeline(events, binding, allowReaderStyleGate); err != nil {
		return LongFormFinalizeResult{}, err
	}
	if existing, ok, err := LoadLongFormFinalization(ctx, store, binding); err != nil {
		return LongFormFinalizeResult{}, err
	} else if ok {
		return replayLongFormFinalizeForRequest(ctx, store, binding, existing.Event, req)
	}
	markdown, err := longFormMarkdownForRequest(ctx, store, binding, req)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	if strings.TrimSpace(markdown) == "" {
		return LongFormFinalizeResult{}, fmt.Errorf("%w: assembled long-form report is empty", app.ErrInvalidInput)
	}
	actualArtifactID := canonicalArtifactIDForFinalizeRequest(binding, req)
	if actualArtifactID != binding.ArtifactID {
		existingArtifact, err := store.GetRawArtifact(ctx, actualArtifactID)
		if err != nil {
			return LongFormFinalizeResult{}, err
		}
		return appendLongFormCanonicalForExistingArtifact(ctx, store, req, binding, existingArtifact, markdown, allowReaderStyleGate)
	}
	if existingArtifact, err := store.GetRawArtifact(ctx, binding.ArtifactID); err == nil {
		return appendLongFormCanonicalForExistingArtifact(ctx, store, req, binding, existingArtifact, markdown, allowReaderStyleGate)
	}
	artifactReq := app.CreateRawArtifactRequest{
		ArtifactID: binding.ArtifactID, MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename,
		Producer: binding.Producer, Content: []byte(markdown),
	}
	artifact, event, created, err := store.CreateRawArtifactWithEventConditionally(ctx, artifactReq, func(events []app.LedgerEvent, artifact app.RawArtifact) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		existing, count := longFormCanonical(events, binding.PendingEventID)
		if count > 1 {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: multiple canonical long-form finalizations", app.ErrConflict)
		}
		if count == 1 {
			if existing.CorrelationID != binding.IdempotencyKey || !canonicalMatchesBinding(existing, eventPayload(existing), binding) {
				return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: canonical long-form finalization binding differs", app.ErrConflict)
			}
			if err := validateLongFormCanonicalReplayRequest(ctx, store, events, binding, existing, req); err != nil {
				return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
			}
			return app.AppendEventRequest{}, existing, false, nil
		}
		if err := validateLongFormLineage(ctx, store, events, binding); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if err := validateLongFormFinalPipeline(events, binding, allowReaderStyleGate); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		return longFormCanonicalRequestForFinalEdit(strings.TrimSpace(req.EventID), binding, artifact, len(strings.Fields(markdown)), req), app.LedgerEvent{}, true, nil
	})
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	if created {
		return LongFormFinalizeResult{Artifact: artifact, Event: event}, nil
	}
	return replayLongFormFinalizeForRequest(ctx, store, binding, event, req)
}

func PrepareLongFormEditingDraft(ctx context.Context, store LongFormFinalizationStore, binding LongFormFinalizeBinding) (string, error) {
	binding = normalizeLongFormFinalizeBinding(binding)
	if err := validateLongFormFinalizeBinding(binding); err != nil {
		return "", err
	}
	if binding.CompositionStrategy != LongFormCompositionNarrativeEdit {
		return "", fmt.Errorf("%w: long-form editing draft requires narrative edit strategy", app.ErrInvalidInput)
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return "", err
	}
	if err := validateLongFormLineage(ctx, store, events, binding); err != nil {
		return "", err
	}
	parts, err := loadLongFormParts(ctx, store, binding)
	if err != nil {
		return "", err
	}
	return AssembleLongFormFinalMarkdown(binding.Title, "", "", parts), nil
}

func longFormMarkdownForRequest(ctx context.Context, store LongFormFinalizationStore, binding LongFormFinalizeBinding, req LongFormFinalizeRequest) (string, error) {
	if binding.CompositionStrategy == LongFormCompositionNarrativeEdit {
		if strings.TrimSpace(req.OpeningMarkdown) != "" || strings.TrimSpace(req.ClosingMarkdown) != "" {
			return "", fmt.Errorf("%w: narrative edit finalization accepts only manuscript Markdown", app.ErrInvalidInput)
		}
		if strings.TrimSpace(req.ManuscriptMarkdown) == "" {
			return "", fmt.Errorf("%w: edited long-form manuscript is empty", app.ErrInvalidInput)
		}
		return req.ManuscriptMarkdown, nil
	}
	if strings.TrimSpace(req.ManuscriptMarkdown) != "" {
		return "", fmt.Errorf("%w: preserved long-form finalization cannot accept edited manuscript", app.ErrInvalidInput)
	}
	parts, err := loadLongFormParts(ctx, store, binding)
	if err != nil {
		return "", err
	}
	return AssembleLongFormFinalMarkdown(binding.Title, req.OpeningMarkdown, req.ClosingMarkdown, parts), nil
}

func normalizeLongFormFinalizeBinding(value LongFormFinalizeBinding) LongFormFinalizeBinding {
	value.MissionID = strings.TrimSpace(value.MissionID)
	value.PendingEventID = strings.TrimSpace(value.PendingEventID)
	value.PlanEventID = strings.TrimSpace(value.PlanEventID)
	value.ArtifactID = strings.TrimSpace(value.ArtifactID)
	value.Filename = strings.TrimSpace(value.Filename)
	value.Title = strings.TrimSpace(value.Title)
	value.ToolSessionID = strings.TrimSpace(value.ToolSessionID)
	value.IdempotencyKey = strings.TrimSpace(value.IdempotencyKey)
	value.ProviderSessionID = strings.TrimSpace(value.ProviderSessionID)
	value.PreviousProviderSessionID = strings.TrimSpace(value.PreviousProviderSessionID)
	value.CompositionStrategy = strings.TrimSpace(value.CompositionStrategy)
	if value.CompositionStrategy == "" {
		value.CompositionStrategy = LongFormCompositionPreserveMarkdown
	}
	value.Producer.Type = strings.TrimSpace(value.Producer.Type)
	value.Producer.ID = strings.TrimSpace(value.Producer.ID)
	return value
}

func validateLongFormFinalizeBinding(value LongFormFinalizeBinding) error {
	if value.MissionID == "" || value.PendingEventID == "" || value.PlanEventID == "" || value.ArtifactID == "" || value.Filename == "" || value.Title == "" || value.ToolSessionID == "" || value.IdempotencyKey == "" || value.ProviderSessionID == "" || value.AgentExecutor == "" || len(value.PartArtifactIDs) == 0 {
		return fmt.Errorf("%w: long-form finalization binding is incomplete", app.ErrInvalidInput)
	}
	if value.Producer.Type != "agent_session" || value.Producer.ID != value.ProviderSessionID {
		return fmt.Errorf("%w: final artifact producer must be the bound provider session", app.ErrInvalidInput)
	}
	if duplicateStrings(value.PartArtifactIDs) || duplicateStrings(value.SectionArtifactIDs) {
		return fmt.Errorf("%w: finalization artifact order contains duplicates", app.ErrConflict)
	}
	if value.CompositionStrategy != LongFormCompositionPreserveMarkdown && value.CompositionStrategy != LongFormCompositionNarrativeEdit {
		return fmt.Errorf("%w: unsupported long-form composition strategy", app.ErrInvalidInput)
	}
	return nil
}

func ValidateLongFormFinalizeBinding(value LongFormFinalizeBinding) error {
	return validateLongFormFinalizeBinding(normalizeLongFormFinalizeBinding(value))
}

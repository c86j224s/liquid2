package reporting

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	payload := eventPayload(event)
	if event.CorrelationID != binding.IdempotencyKey || !canonicalMatchesBinding(event, payload, binding) {
		return LongFormFinalizeResult{}, false, fmt.Errorf("%w: canonical long-form finalization binding differs", app.ErrConflict)
	}
	artifact, err := store.GetRawArtifact(ctx, binding.ArtifactID)
	if err != nil {
		return LongFormFinalizeResult{}, false, err
	}
	if artifact.MissionID != binding.MissionID || artifact.MediaType != "text/markdown; charset=utf-8" || artifact.Filename != binding.Filename || artifact.Producer != binding.Producer {
		return LongFormFinalizeResult{}, false, fmt.Errorf("%w: canonical long-form final artifact differs", app.ErrConflict)
	}
	return LongFormFinalizeResult{Artifact: artifact, Event: event, Replay: true}, true, nil
}

func FinalizeLongForm(ctx context.Context, store LongFormFinalizationStore, req LongFormFinalizeRequest) (LongFormFinalizeResult, error) {
	binding := normalizeLongFormFinalizeBinding(req.Binding)
	if err := validateLongFormFinalizeBinding(binding); err != nil {
		return LongFormFinalizeResult{}, err
	}
	if existing, ok, err := LoadLongFormFinalization(ctx, store, binding); err != nil {
		return LongFormFinalizeResult{}, err
	} else if ok {
		return replayLongFormFinalize(ctx, store, binding, existing.Event, req)
	}
	markdown, err := longFormMarkdownForRequest(ctx, store, binding, req)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	if strings.TrimSpace(markdown) == "" {
		return LongFormFinalizeResult{}, fmt.Errorf("%w: assembled long-form report is empty", app.ErrInvalidInput)
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
			return app.AppendEventRequest{}, existing, false, nil
		}
		if err := validateLongFormLineage(events, binding); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		return longFormCanonicalRequest(strings.TrimSpace(req.EventID), binding, artifact, len(strings.Fields(markdown))), app.LedgerEvent{}, true, nil
	})
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	if created {
		return LongFormFinalizeResult{Artifact: artifact, Event: event}, nil
	}
	return replayLongFormFinalize(ctx, store, binding, event, req)
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
	if err := validateLongFormLineage(events, binding); err != nil {
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

func longFormCanonicalRequest(eventID string, binding LongFormFinalizeBinding, artifact app.RawArtifact, finalWords int) app.AppendEventRequest {
	duration := time.Since(binding.StartedAt).Milliseconds()
	if binding.StartedAt.IsZero() || duration < 0 {
		duration = 0
	}
	assemblyStrategy := "c4_normalized_section_headings"
	text := "섹션별 보존 조립 방식으로 장문 Markdown 리포트 artifact를 생성했습니다."
	if binding.CompositionStrategy == LongFormCompositionNarrativeEdit {
		assemblyStrategy = "narrative_contract_final_edit"
		text = "바인딩된 파트 원고를 최종 편집해 장문 Markdown 리포트 artifact를 생성했습니다."
	}
	request := BuildMarkdownReportArtifactCreatedAppendRequest(MarkdownReportArtifactCreatedEventRequest{
		MarkdownReportEventBase: MarkdownReportEventBase{
			EventID: eventID, MissionID: binding.MissionID, PendingEventID: binding.PendingEventID, Title: binding.Title,
			AgentExecutor: binding.AgentExecutor, AgentModel: binding.AgentModel, AgentReasoningEffort: binding.AgentReasoningEffort, AgentSelectionSource: binding.AgentSelectionSource,
			AgentSessionID: binding.ProviderSessionID, PreviousAgentSessionID: binding.PreviousProviderSessionID,
			ToolSessionID: binding.ToolSessionID, MCPMode: binding.MCPMode, RigorLevel: binding.RigorLevel, RigorLabel: binding.RigorLabel,
			ReportMode: ModeLongForm, ReportModeLabel: ModeLabel(ModeLongForm), ReportSessionPolicy: binding.ReportSessionPolicy, ReportSessionPolicySelection: binding.ReportSessionPolicySelection,
			PostReportHumanize: binding.PostReportHumanize, HumanizeEnabled: binding.PostReportHumanize != "disabled",
			GenerationGuidanceProfile: binding.GenerationGuidanceProfile, GenerationGuidanceSHA256: binding.GenerationGuidanceSHA256,
			SessionChainKind: binding.SessionChainKind, PreReportResearchSessionID: binding.PreReportResearchSessionID, ReportPlanSessionID: binding.ReportPlanSessionID,
			ReportSessionID: binding.ProviderSessionID, ForkSourceAgentSessionID: binding.ForkSourceAgentSessionID,
			CompositionStrategy: binding.CompositionStrategy, DurationMS: duration,
			Text: text, Producer: binding.Producer,
		},
		Artifact: artifact, PlanEventID: binding.PlanEventID, PlanToolSessionID: binding.PlanToolSessionID,
		IncludePlanReview: true, PlanReviewState: "auto_accepted", AssemblyStrategy: assemblyStrategy,
		SectionCount: len(binding.SectionArtifactIDs), PartCount: len(binding.PartArtifactIDs), SectionArtifactIDs: binding.SectionArtifactIDs,
		PartArtifactIDs: binding.PartArtifactIDs, SectionWordCount: binding.SectionWordCount, FinalWordCount: finalWords,
		PreservationRatio:     float64(finalWords) / float64(maxReportingInt(1, binding.SectionWordCount)),
		OmitPreservationRatio: binding.CompositionStrategy == LongFormCompositionNarrativeEdit, IncludeLongFormFields: true,
	})
	request.CorrelationID = binding.IdempotencyKey
	return request
}

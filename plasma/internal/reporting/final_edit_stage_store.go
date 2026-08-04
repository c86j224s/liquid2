package reporting

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// FinalEditStageStore는 final edit stage 제출과 조회 계약을 모은 저장소 포트다.
type FinalEditStageStore interface {
	LongFormFinalizationStore
	GetEvidenceRecord(context.Context, string) (app.EvidenceRecord, error)
}

// StartFinalEditStage는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func StartFinalEditStage(ctx context.Context, store FinalEditStageStore, eventID string, binding FinalEditStageBinding) (app.LedgerEvent, bool, error) {
	binding = normalizeFinalEditStageBinding(binding)
	if err := validateFinalEditStageBinding(binding); err != nil {
		return app.LedgerEvent{}, false, err
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	if binding.Stage == FinalEditStageWriter {
		return startFinalEditWriterStage(ctx, store, eventID, binding)
	}
	if binding.Stage == FinalEditStageReader && plan.Pipeline == FinalEditPipelineReaderStyleGateV1 {
		return startFinalEditReaderStage(ctx, store, eventID, binding)
	}
	if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
		return app.LedgerEvent{}, false, err
	}
	return store.AppendEventConditionally(ctx, binding.MissionID, func(events []app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if existing, ok, err := finalEditStageStartedEvent(events, binding); ok || err != nil {
			return app.AppendEventRequest{}, existing, false, err
		}
		return BuildFinalEditStageStartedAppendRequest(eventID, binding), app.LedgerEvent{}, true, nil
	})
}

func startFinalEditWriterStage(ctx context.Context, store FinalEditStageStore, eventID string, binding FinalEditStageBinding) (app.LedgerEvent, bool, error) {
	if _, _, err := EnsureFinalEditAssembly(ctx, store, newFinalEditAssemblyEventID(eventID), binding); err != nil {
		return app.LedgerEvent{}, false, err
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
		return app.LedgerEvent{}, false, err
	}
	return store.AppendEventConditionally(ctx, binding.MissionID, func(events []app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if existing, ok, err := finalEditStageStartedEvent(events, binding); ok || err != nil {
			return app.AppendEventRequest{}, existing, false, err
		}
		return BuildFinalEditStageStartedAppendRequest(eventID, binding), app.LedgerEvent{}, true, nil
	})
}

func newFinalEditAssemblyEventID(stageEventID string) string {
	stageEventID = strings.TrimSpace(stageEventID)
	if strings.HasPrefix(stageEventID, "evt_") {
		return "evt_final_assembly_" + strings.TrimPrefix(stageEventID, "evt_")
	}
	return stageEventID
}

func startFinalEditReaderStage(ctx context.Context, store FinalEditStageStore, eventID string, binding FinalEditStageBinding) (app.LedgerEvent, bool, error) {
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
		return app.LedgerEvent{}, false, err
	}
	artifactReq, err := finalEditReaderSourceRequest(ctx, store, events, binding)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	if existing, err := store.GetRawArtifact(ctx, artifactReq.ArtifactID); err == nil {
		if err := validateFinalEditReaderSourceArtifact(existing, artifactReq); err != nil {
			return app.LedgerEvent{}, false, err
		}
	}
	artifact, event, created, err := store.CreateRawArtifactWithEventConditionally(ctx, artifactReq, func(events []app.LedgerEvent, artifact app.RawArtifact) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if err := validateFinalEditReaderSourceArtifact(artifact, artifactReq); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if existing, ok, err := finalEditStageStartedEvent(events, binding); ok || err != nil {
			return app.AppendEventRequest{}, existing, false, err
		}
		return BuildFinalEditStageStartedAppendRequest(eventID, binding), app.LedgerEvent{}, true, nil
	})
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	if err := validateFinalEditReaderSourceArtifact(artifact, artifactReq); err != nil {
		return app.LedgerEvent{}, false, err
	}
	return event, created, nil
}

// SubmitFinalEditStage는 최종 편집 stage 제출 artifact와 이벤트를 기록한다.
func SubmitFinalEditStage(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, markdown string, operationCount int) (FinalEditStageResult, error) {
	switch normalizeFinalEditStageBinding(binding).Stage {
	case FinalEditStageGate:
		return FinalEditStageResult{}, fmt.Errorf("%w: corrective gate completion must use SubmitFinalEditGate", app.ErrInvalidInput)
	case FinalEditStageStyleSemanticValidation:
		return FinalEditStageResult{}, fmt.Errorf("%w: style semantic validation completion must use SubmitFinalEditStyleSemanticValidation", app.ErrInvalidInput)
	case FinalEditStageEvidenceGate:
		return FinalEditStageResult{}, fmt.Errorf("%w: evidence gate completion must use SubmitFinalEditEvidenceGate", app.ErrInvalidInput)
	}
	return submitFinalEditStage(ctx, store, binding, eventID, markdown, operationCount, nil, FinalEditSemanticAttestation{})
}

// SubmitFinalEditStyleStage는 문체 편집 stage 제출 artifact와 이벤트를 기록한다.
func SubmitFinalEditStyleStage(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, markdown string, operationCount int, diagnoses []FinalEditStyleOperationDiagnosis) (FinalEditStageResult, error) {
	if normalizeFinalEditStageBinding(binding).Stage != FinalEditStageStyle {
		return FinalEditStageResult{}, fmt.Errorf("%w: style operation diagnoses are only valid for style edit", app.ErrInvalidInput)
	}
	return submitFinalEditStageWithStyleDiagnoses(ctx, store, binding, eventID, markdown, operationCount, diagnoses, nil, FinalEditSemanticAttestation{})
}

// SubmitFinalEditStyleSemanticValidation는 문체 편집 의미 검증 결과를 기록한다.
func SubmitFinalEditStyleSemanticValidation(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, reviews []FinalEditSemanticAcceptance) (FinalEditStageResult, error) {
	binding = normalizeFinalEditStageBinding(binding)
	markdown, semanticReview, err := BuildFinalEditStyleSemanticValidation(ctx, store, binding, reviews)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	style, ok, err := finalEditStyleSubmissionForGate(ctx, store, events, binding)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if ok && style.Artifact.ArtifactID != style.SourceArtifact.ArtifactID && markdown == string(style.SourceArtifact.Content) {
		return submitFinalEditStageExistingArtifact(ctx, store, binding, eventID, style.Artifact, style.SourceArtifact, markdown, 0, true, nil, nil, semanticReview)
	}
	return submitFinalEditStage(ctx, store, binding, eventID, markdown, 0, nil, semanticReview)
}

// LoadFinalEditStageSubmission는 stage 제출 이벤트와 artifact를 장부에서 복원한다.
func LoadFinalEditStageSubmission(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding) (FinalEditStageResult, bool, error) {
	binding = normalizeFinalEditStageBinding(binding)
	if err := validateFinalEditStageBinding(binding); err != nil {
		return FinalEditStageResult{}, false, err
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return FinalEditStageResult{}, false, err
	}
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return FinalEditStageResult{}, false, err
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	if err := validateFinalEditStageLineage(ctx, store, events, binding, finalEditStageAllowsCanonicalLoad(binding.Stage)); err != nil {
		return FinalEditStageResult{}, false, err
	}
	event, ok, err := finalEditStageSubmittedEvent(events, binding)
	if err != nil || !ok {
		return FinalEditStageResult{}, ok, err
	}
	result, err := finalEditStageResultFromEvent(ctx, store, binding, event, true)
	return result, err == nil, err
}

func submitFinalEditStage(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, markdown string, operationCount int, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) (FinalEditStageResult, error) {
	return submitFinalEditStageWithStyleDiagnoses(ctx, store, binding, eventID, markdown, operationCount, nil, findings, semanticReview)
}

func submitFinalEditStageWithStyleDiagnoses(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, markdown string, operationCount int, diagnoses []FinalEditStyleOperationDiagnosis, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) (FinalEditStageResult, error) {
	binding = normalizeFinalEditStageBinding(binding)
	if err := validateFinalEditStageBinding(binding); err != nil {
		return FinalEditStageResult{}, err
	}
	if strings.TrimSpace(markdown) == "" || operationCount < 0 {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage Markdown is invalid", app.ErrInvalidInput)
	}
	if binding.Stage != FinalEditStageStyle {
		if existing, ok, err := LoadFinalEditStageSubmission(ctx, store, binding); err != nil {
			return FinalEditStageResult{}, err
		} else if ok {
			if err := validateFinalEditStageReplayInput(existing, markdown, operationCount, diagnoses, findings, semanticReview); err != nil {
				return FinalEditStageResult{}, err
			}
			return existing, nil
		}
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	if err := validateFinalEditStageLineage(ctx, store, events, binding, finalEditStageAllowsCanonicalLoad(binding.Stage)); err != nil {
		return FinalEditStageResult{}, err
	}
	if _, ok, err := finalEditStageStartedEvent(events, binding); err != nil {
		return FinalEditStageResult{}, err
	} else if !ok {
		return FinalEditStageResult{}, fmt.Errorf("%w: matching final edit stage start is missing", app.ErrConflict)
	}
	source, err := store.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if source.MissionID != binding.MissionID || source.MediaType != "text/markdown; charset=utf-8" || source.SHA256 != contentSHA256(source.Content) {
		return FinalEditStageResult{}, fmt.Errorf("%w: source final edit artifact is foreign or not Markdown", app.ErrConflict)
	}
	if binding.Stage == FinalEditStageStyle {
		if err := ValidateFinalEditStyleMarkdown(string(source.Content), markdown); err != nil {
			markdown = string(source.Content)
			operationCount = 0
			diagnoses = nil
		}
		if string(source.Content) == markdown {
			operationCount = 0
			diagnoses = nil
		}
		if string(source.Content) != markdown && operationCount <= 0 {
			return FinalEditStageResult{}, fmt.Errorf("%w: changed style edit requires at least one operation diagnosis", app.ErrInvalidInput)
		}
		if err := ValidateFinalEditStyleOperationDiagnoses(operationCount, diagnoses, true); err != nil {
			return FinalEditStageResult{}, err
		}
		if existing, ok, err := LoadFinalEditStageSubmission(ctx, store, binding); err != nil {
			return FinalEditStageResult{}, err
		} else if ok {
			if err := validateFinalEditStageReplayInput(existing, markdown, operationCount, diagnoses, findings, semanticReview); err != nil {
				return FinalEditStageResult{}, err
			}
			return existing, nil
		}
	}
	if string(source.Content) == markdown {
		if binding.Stage == FinalEditStageStyle {
			operationCount = 0
			diagnoses = nil
		}
		return submitFinalEditStageNoOp(ctx, store, binding, eventID, source, operationCount, diagnoses, findings, semanticReview)
	}
	return submitFinalEditStageChanged(ctx, store, binding, eventID, source, markdown, operationCount, diagnoses, findings, semanticReview)
}

func submitFinalEditStageNoOp(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, source app.RawArtifact, operationCount int, diagnoses []FinalEditStyleOperationDiagnosis, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) (FinalEditStageResult, error) {
	event, created, err := store.AppendEventConditionally(ctx, binding.MissionID, func(events []app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		if err := validateFinalEditStageLineage(ctx, store, events, binding, finalEditStageAllowsCanonicalLoad(binding.Stage)); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if _, ok, err := finalEditStageStartedEvent(events, binding); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		} else if !ok {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: matching final edit stage start is missing", app.ErrConflict)
		}
		if existing, ok, err := finalEditStageSubmittedEvent(events, binding); ok || err != nil {
			return app.AppendEventRequest{}, existing, false, err
		}
		return buildFinalEditSubmittedAppendRequestWithStyleDiagnoses(eventID, binding, source, source, operationCount, false, diagnoses, findings, semanticReview), app.LedgerEvent{}, true, nil
	})
	if err != nil {
		return FinalEditStageResult{}, err
	}
	result, err := finalEditStageResultFromEvent(ctx, store, binding, event, !created)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if err := validateFinalEditStageReplayInput(result, string(source.Content), operationCount, diagnoses, findings, semanticReview); err != nil {
		return FinalEditStageResult{}, err
	}
	return result, nil
}

func submitFinalEditStageExistingArtifact(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, source app.RawArtifact, artifact app.RawArtifact, markdown string, operationCount int, changed bool, diagnoses []FinalEditStyleOperationDiagnosis, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) (FinalEditStageResult, error) {
	event, created, err := store.AppendEventConditionally(ctx, binding.MissionID, func(events []app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		if err := validateFinalEditStageLineage(ctx, store, events, binding, finalEditStageAllowsCanonicalLoad(binding.Stage)); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if _, ok, err := finalEditStageStartedEvent(events, binding); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		} else if !ok {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: matching final edit stage start is missing", app.ErrConflict)
		}
		if existing, ok, err := finalEditStageSubmittedEvent(events, binding); ok || err != nil {
			return app.AppendEventRequest{}, existing, false, err
		}
		return buildFinalEditSubmittedAppendRequestWithStyleDiagnoses(eventID, binding, source, artifact, operationCount, changed, diagnoses, findings, semanticReview), app.LedgerEvent{}, true, nil
	})
	if err != nil {
		return FinalEditStageResult{}, err
	}
	result, err := finalEditStageResultFromEvent(ctx, store, binding, event, !created)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if err := validateFinalEditStageReplayInput(result, markdown, operationCount, diagnoses, findings, semanticReview); err != nil {
		return FinalEditStageResult{}, err
	}
	return result, nil
}

func finalEditStageAllowsCanonicalLoad(stage string) bool {
	return stage == FinalEditStageGate || stage == FinalEditStageEvidenceGate
}

func validateFinalEditStageReplayInput(result FinalEditStageResult, markdown string, operationCount int, diagnoses []FinalEditStyleOperationDiagnosis, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) error {
	if string(result.Artifact.Content) != markdown || result.OperationCount != operationCount || !equalStoredFinalEditGateFindings(result.GateFindings, findings) || !equalStoredFinalEditSemanticAcceptance(result.SemanticReview.Records, semanticReview.Records) || result.SemanticReview.Digest != semanticReview.Digest || result.SemanticReview.Count != semanticReview.Count {
		return fmt.Errorf("%w: final edit stage replay input differs", app.ErrConflict)
	}
	if result.StyleOperationDiagnosesPresent && !equalFinalEditStyleOperationDiagnosesForReplay(result.StyleOperationDiagnoses, diagnoses) {
		return fmt.Errorf("%w: final edit stage replay input differs", app.ErrConflict)
	}
	return nil
}

package reporting

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

// SubmitFinalEditStage는 최종 편집 stage 제출 artifact와 이벤트를 기록한다.
func SubmitFinalEditStage(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, markdown string, operationCount int) (FinalEditStageResult, error) {
	switch normalizeFinalEditStageBinding(binding).Stage {
	case FinalEditStageGate:
		return FinalEditStageResult{}, fmt.Errorf("%w: corrective gate completion must use SubmitFinalEditGate", producterror.ErrInvalidInput)
	case FinalEditStageStyleSemanticValidation:
		return FinalEditStageResult{}, fmt.Errorf("%w: style semantic validation completion must use SubmitFinalEditStyleSemanticValidation", producterror.ErrInvalidInput)
	case FinalEditStageEvidenceGate:
		return FinalEditStageResult{}, fmt.Errorf("%w: evidence gate completion must use SubmitFinalEditEvidenceGate", producterror.ErrInvalidInput)
	}
	return submitFinalEditStage(ctx, store, binding, eventID, markdown, operationCount, nil, FinalEditSemanticAttestation{})
}

// SubmitFinalEditStyleStage는 문체 편집 stage 제출 artifact와 이벤트를 기록한다.
func SubmitFinalEditStyleStage(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, markdown string, operationCount int, diagnoses []FinalEditStyleOperationDiagnosis) (FinalEditStageResult, error) {
	if normalizeFinalEditStageBinding(binding).Stage != FinalEditStageStyle {
		return FinalEditStageResult{}, fmt.Errorf("%w: style operation diagnoses are only valid for style edit", producterror.ErrInvalidInput)
	}
	return submitFinalEditStageWithStyleDiagnoses(ctx, store, binding, eventID, markdown, operationCount, diagnoses, nil, FinalEditSemanticAttestation{})
}

// SubmitFinalEditStyleSemanticValidation는 문체 편집 의미 검증 결과를 기록한다.
func SubmitFinalEditStyleSemanticValidation(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, reviews []FinalEditSemanticAcceptance) (FinalEditStageResult, error) {
	binding = normalizeFinalEditStageBinding(binding)
	markdown, semanticReview, err := buildFinalEditStyleSemanticValidation(ctx, store, binding, reviews)
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

func submitFinalEditStage(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, markdown string, operationCount int, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) (FinalEditStageResult, error) {
	return submitFinalEditStageWithStyleDiagnoses(ctx, store, binding, eventID, markdown, operationCount, nil, findings, semanticReview)
}

func submitFinalEditStageWithStyleDiagnoses(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, markdown string, operationCount int, diagnoses []FinalEditStyleOperationDiagnosis, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) (FinalEditStageResult, error) {
	binding = normalizeFinalEditStageBinding(binding)
	if err := validateFinalEditStageBinding(binding); err != nil {
		return FinalEditStageResult{}, err
	}
	if strings.TrimSpace(markdown) == "" || operationCount < 0 {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage Markdown is invalid", producterror.ErrInvalidInput)
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
		return FinalEditStageResult{}, fmt.Errorf("%w: matching final edit stage start is missing", producterror.ErrConflict)
	}
	source, err := store.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if source.MissionID != binding.MissionID || source.MediaType != "text/markdown; charset=utf-8" || source.SHA256 != contentSHA256(source.Content) {
		return FinalEditStageResult{}, fmt.Errorf("%w: source final edit artifact is foreign or not Markdown", producterror.ErrConflict)
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
			return FinalEditStageResult{}, fmt.Errorf("%w: changed style edit requires at least one operation diagnosis", producterror.ErrInvalidInput)
		}
		if err := validateFinalEditStyleOperationDiagnoses(operationCount, diagnoses, true); err != nil {
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

package reporting

import (
	"context"
	"fmt"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// FinalEditStageStartContract는 재실행과 검증에 쓰는 binding 계약이다.
type FinalEditStageStartContract struct {
	FinalBinding LongFormFinalizeBinding
	Stage        string
}

// FinalEditStageStartResult는 final edit stage 시작 이벤트와 binding을 함께 반환한다.
type FinalEditStageStartResult struct {
	Binding        FinalEditStageBinding
	SourceArtifact artifactcontract.Raw
	Event          ledger.Event
}

// FinalEditStageProgress는 final edit stage별 제출 artifact와 실패 상태를 투영한 값이다.
type FinalEditStageProgress struct {
	Binding        FinalEditStageBinding
	SourceArtifact artifactcontract.Raw
	StartEvent     ledger.Event
	Submission     *FinalEditStageResult
}

// LoadFinalEditStageProgress는 final edit stage들의 제출·실패 진행 상태를 장부에서 복원한다.
func LoadFinalEditStageProgress(ctx context.Context, store FinalEditStageStore, contract FinalEditStageStartContract) (FinalEditStageProgress, bool, error) {
	finalBinding := normalizeLongFormFinalizeBinding(contract.FinalBinding)
	if err := validateLongFormFinalizeBinding(finalBinding); err != nil {
		return FinalEditStageProgress{}, false, err
	}
	stage := normalizeFinalEditStageBinding(FinalEditStageBinding{Stage: contract.Stage}).Stage
	if finalEditStartedEventType(stage) == "" {
		return FinalEditStageProgress{}, false, fmt.Errorf("%w: unsupported final edit stage", producterror.ErrInvalidInput)
	}
	events, err := store.ListEvents(ctx, finalBinding.MissionID)
	if err != nil {
		return FinalEditStageProgress{}, false, err
	}
	acceptedPending, err := longFormPendingLineage(events, finalBinding.PendingEventID)
	if err != nil {
		return FinalEditStageProgress{}, false, err
	}
	plan, ok, err := longFormPlanPipeline(events, finalBinding.PendingEventID, finalBinding.PlanEventID)
	if err != nil {
		return FinalEditStageProgress{}, false, err
	}
	if !ok || !isSupportedFinalEditPipeline(plan.Pipeline) {
		return FinalEditStageProgress{}, false, fmt.Errorf("%w: final edit plan is not active", producterror.ErrConflict)
	}
	finalBinding.PostReportHumanize = plan.PostReportHumanize
	var progress FinalEditStageProgress
	count := 0
	for _, event := range events {
		if event.EventType != finalEditStartedEventType(stage) ||
			event.MissionID != finalBinding.MissionID ||
			!acceptedPending[payloadString(eventPayload(event), "pending_event_id")] {
			continue
		}
		binding, ok := finalEditStageBindingFromStartEventForPipeline(event, plan.Pipeline)
		if !ok {
			return FinalEditStageProgress{}, false, fmt.Errorf("%w: stored final edit start is invalid", producterror.ErrConflict)
		}
		if binding.PendingEventID != finalBinding.PendingEventID ||
			binding.PlanEventID != finalBinding.PlanEventID ||
			binding.Stage != stage {
			continue
		}
		submitted, submittedOK, err := finalEditStageSubmittedEvent(events, binding)
		if err != nil {
			return FinalEditStageProgress{}, false, err
		}
		if err := validateFinalEditStageLineage(ctx, store, events, binding, submittedOK); err != nil {
			return FinalEditStageProgress{}, false, err
		}
		if err := finalEditStageStartMatchesFinalBinding(binding, finalBinding, plan); err != nil {
			return FinalEditStageProgress{}, false, err
		}
		source, err := store.GetRawArtifact(ctx, binding.SourceArtifactID)
		if err != nil {
			return FinalEditStageProgress{}, false, err
		}
		if source.MissionID != binding.MissionID || source.MediaType != "text/markdown; charset=utf-8" || source.Filename != binding.Filename || source.SHA256 != contentSHA256(source.Content) {
			return FinalEditStageProgress{}, false, fmt.Errorf("%w: final edit start source artifact is invalid", producterror.ErrConflict)
		}
		progress = FinalEditStageProgress{Binding: binding, SourceArtifact: source, StartEvent: event}
		if submittedOK {
			result, err := finalEditStageResultFromEvent(ctx, store, progress.Binding, submitted, true)
			if err != nil {
				return FinalEditStageProgress{}, false, err
			}
			progress.Submission = &result
		}
		count++
	}
	if count > 1 {
		return FinalEditStageProgress{}, false, fmt.Errorf("%w: multiple final edit starts match current pending", producterror.ErrConflict)
	}
	if count == 0 {
		return FinalEditStageProgress{}, false, nil
	}
	return progress, true, nil
}

func finalEditStageStartMatchesFinalBinding(binding FinalEditStageBinding, finalBinding LongFormFinalizeBinding, plan FinalEditPipelinePlanState) error {
	if binding.MissionID != finalBinding.MissionID ||
		binding.PendingEventID != finalBinding.PendingEventID ||
		binding.PlanEventID != finalBinding.PlanEventID ||
		binding.Filename != finalBinding.Filename ||
		binding.Title != finalBinding.Title ||
		binding.AgentExecutor != finalBinding.AgentExecutor ||
		binding.AgentModel != finalBinding.AgentModel ||
		binding.AgentReasoningEffort != finalBinding.AgentReasoningEffort ||
		binding.AgentSelectionSource != finalBinding.AgentSelectionSource ||
		binding.MCPMode != finalBinding.MCPMode ||
		binding.RigorLevel != finalBinding.RigorLevel ||
		binding.RigorLabel != finalBinding.RigorLabel ||
		binding.ReportSessionPolicy != finalBinding.ReportSessionPolicy ||
		binding.ReportSessionPolicySelection != finalBinding.ReportSessionPolicySelection ||
		binding.PostReportHumanize != plan.PostReportHumanize ||
		binding.GenerationGuidanceProfile != finalBinding.GenerationGuidanceProfile ||
		binding.GenerationGuidanceSHA256 != finalBinding.GenerationGuidanceSHA256 ||
		binding.SessionChainKind != finalBinding.SessionChainKind ||
		binding.PreReportResearchSessionID != finalBinding.PreReportResearchSessionID ||
		binding.ReportPlanSessionID != finalBinding.ReportPlanSessionID {
		return fmt.Errorf("%w: final edit start binding differs from final binding", producterror.ErrConflict)
	}
	return nil
}

func loadLongFormCanonicalResult(ctx context.Context, store LongFormFinalizationStore, binding LongFormFinalizeBinding, event ledger.Event) (LongFormFinalizeResult, error) {
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	artifact, err := loadCanonicalArtifactForBinding(ctx, store, events, binding, event)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	return LongFormFinalizeResult{Artifact: artifact, Event: event, Replay: true}, nil
}

func replayLongFormFinalizeForRequest(ctx context.Context, store LongFormFinalizationStore, binding LongFormFinalizeBinding, event ledger.Event, req LongFormFinalizeRequest) (LongFormFinalizeResult, error) {
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	if err := validateLongFormCanonicalReplayRequest(ctx, store, events, binding, event, req); err != nil {
		return LongFormFinalizeResult{}, err
	}
	artifact, err := loadCanonicalArtifactForBinding(ctx, store, events, binding, event)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	expected, err := longFormMarkdownForRequest(ctx, store, binding, req)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	if artifact.SHA256 != contentSHA256([]byte(expected)) || string(artifact.Content) != expected {
		return LongFormFinalizeResult{}, fmt.Errorf("%w: canonical long-form finalization content differs", producterror.ErrConflict)
	}
	return LongFormFinalizeResult{Artifact: artifact, Event: event, Replay: true}, nil
}

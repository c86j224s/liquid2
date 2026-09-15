package reporting

import (
	"context"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"strings"
)

// FinalEditStageStore는 final edit stage 제출과 조회 계약을 모은 저장소 포트다.
type FinalEditStageStore interface {
	LongFormFinalizationStore
	GetEvidenceRecord(context.Context, string) (researchrecords.EvidenceRecord, error)
}

// StartFinalEditStage는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func StartFinalEditStage(ctx context.Context, store FinalEditStageStore, eventID string, binding FinalEditStageBinding) (ledger.Event, bool, error) {
	binding = normalizeFinalEditStageBinding(binding)
	if err := validateFinalEditStageBinding(binding); err != nil {
		return ledger.Event{}, false, err
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return ledger.Event{}, false, err
	}
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return ledger.Event{}, false, err
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	if binding.Stage == FinalEditStageWriter {
		return startFinalEditWriterStage(ctx, store, eventID, binding)
	}
	if binding.Stage == FinalEditStageReader && plan.Pipeline == FinalEditPipelineReaderStyleGateV1 {
		return startFinalEditReaderStage(ctx, store, eventID, binding)
	}
	if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
		return ledger.Event{}, false, err
	}
	return store.AppendEventConditionally(ctx, binding.MissionID, func(events []ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error) {
		if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if existing, ok, err := finalEditStageStartedEvent(events, binding); ok || err != nil {
			return ledger.AppendRequest{}, existing, false, err
		}
		return BuildFinalEditStageStartedAppendRequest(eventID, binding), ledger.Event{}, true, nil
	})
}

func startFinalEditWriterStage(ctx context.Context, store FinalEditStageStore, eventID string, binding FinalEditStageBinding) (ledger.Event, bool, error) {
	if _, _, err := EnsureFinalEditAssembly(ctx, store, newFinalEditAssemblyEventID(eventID), binding); err != nil {
		return ledger.Event{}, false, err
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return ledger.Event{}, false, err
	}
	if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
		return ledger.Event{}, false, err
	}
	return store.AppendEventConditionally(ctx, binding.MissionID, func(events []ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error) {
		if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if existing, ok, err := finalEditStageStartedEvent(events, binding); ok || err != nil {
			return ledger.AppendRequest{}, existing, false, err
		}
		return BuildFinalEditStageStartedAppendRequest(eventID, binding), ledger.Event{}, true, nil
	})
}

func newFinalEditAssemblyEventID(stageEventID string) string {
	stageEventID = strings.TrimSpace(stageEventID)
	if strings.HasPrefix(stageEventID, "evt_") {
		return "evt_final_assembly_" + strings.TrimPrefix(stageEventID, "evt_")
	}
	return stageEventID
}

func startFinalEditReaderStage(ctx context.Context, store FinalEditStageStore, eventID string, binding FinalEditStageBinding) (ledger.Event, bool, error) {
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return ledger.Event{}, false, err
	}
	if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
		return ledger.Event{}, false, err
	}
	artifactReq, err := finalEditReaderSourceRequest(ctx, store, events, binding)
	if err != nil {
		return ledger.Event{}, false, err
	}
	if existing, err := store.GetRawArtifact(ctx, artifactReq.ArtifactID); err == nil {
		if err := validateFinalEditReaderSourceArtifact(existing, artifactReq); err != nil {
			return ledger.Event{}, false, err
		}
	}
	artifact, event, created, err := store.CreateRawArtifactWithEventConditionally(ctx, artifactReq, func(events []ledger.Event, artifact artifactcontract.Raw) (ledger.AppendRequest, ledger.Event, bool, error) {
		if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if err := validateFinalEditReaderSourceArtifact(artifact, artifactReq); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if existing, ok, err := finalEditStageStartedEvent(events, binding); ok || err != nil {
			return ledger.AppendRequest{}, existing, false, err
		}
		return BuildFinalEditStageStartedAppendRequest(eventID, binding), ledger.Event{}, true, nil
	})
	if err != nil {
		return ledger.Event{}, false, err
	}
	if err := validateFinalEditReaderSourceArtifact(artifact, artifactReq); err != nil {
		return ledger.Event{}, false, err
	}
	return event, created, nil
}

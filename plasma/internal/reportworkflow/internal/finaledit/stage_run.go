package finaledit

import (
	"context"
	"fmt"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// StagePrompt는 stage package가 소유한 prompt builder 계약이다.
type StagePrompt func(Input, reporting.FinalEditStageBinding, string, int) string

// RunStage는 final edit stage의 기존 두 번 시도, same-session 검증, durable submission
// replay, gate resume 순서를 보존한다.
func (runner Runner) RunStage(ctx context.Context, input Input, progress reporting.FinalEditStageProgress, finalBinding *reporting.LongFormFinalizeBinding, tools []string, prompt StagePrompt, executor agentexec.AgentExecutor) (StageRun, error) {
	binding := progress.Binding
	if progress.Submission != nil {
		return runner.ReplayStage(ctx, input, binding, *progress.Submission, finalBinding)
	}
	for attempt := 1; attempt <= 2; attempt++ {
		started := time.Now()
		result, runErr := executor.Run(ctx, agentexec.AgentRequest{
			UserText:          "run long-form " + binding.Stage,
			Prompt:            prompt(input, binding, runner.ID("rfe"), attempt),
			Model:             input.AgentModel,
			ReasoningEffort:   input.AgentReasoningEffort,
			MissionID:         input.MissionID,
			ToolSessionID:     binding.ToolSessionID,
			PreviousSessionID: binding.ProviderSessionID,
			AgentExecutor:     input.ExecutorName,
			MCPMode:           input.MCPMode,
			ExtraMCPTools:     tools,
			ReplaceMCPTools:   true,
			FinalEditStage:    &binding,
			LongFormFinalize:  finalBinding,
		})
		durationMS := time.Since(started).Milliseconds()
		if runErr == nil {
			result, runErr = ValidatedSameSessionResult(result, binding.ProviderSessionID)
		}
		stage, stageOK, stageErr := reporting.LoadFinalEditStageSubmission(context.WithoutCancel(ctx), runner.Store, binding)
		if stageErr != nil {
			return StageRun{}, StageFailure(binding.Stage, input.PlanEvent.EventID, stageErr)
		}
		if finalBinding != nil {
			final, finalOK, finalErr := reporting.LoadLongFormFinalization(context.WithoutCancel(ctx), runner.Store, *finalBinding)
			if finalErr != nil {
				return StageRun{}, StageFailure(binding.Stage, input.PlanEvent.EventID, finalErr)
			}
			if !finalOK && stageOK {
				final, finalErr = reporting.ResumeFinalEditGate(context.WithoutCancel(ctx), runner.Store, reporting.FinalEditGateResumeRequest{
					StageBinding: binding, FinalBinding: *finalBinding, CanonicalEventID: runner.ID("evt"),
				})
				finalOK = finalErr == nil
				if finalErr != nil && runErr == nil {
					runErr = finalErr
				}
			}
			if finalOK && stageOK {
				recordStageUsage(ctx, runner.Store, input, binding, stage, result, durationMS)
				return StageRun{Binding: binding, Stage: stage, Final: final}, nil
			}
		} else if stageOK {
			recordStageUsage(ctx, runner.Store, input, binding, stage, result, durationMS)
			return StageRun{Binding: binding, Stage: stage}, nil
		}
		if attempt == 1 {
			continue
		}
		cause := runErr
		if cause == nil {
			cause = fmt.Errorf("%w: final edit stage acknowledgement was not exact", producterror.ErrConflict)
		}
		return StageRun{}, StageFailure(binding.Stage, input.PlanEvent.EventID, AgentFailure(cause, result, "report_"+binding.Stage, durationMS, binding.ProviderSessionID))
	}
	return StageRun{Binding: binding}, nil
}

// ReplayStage는 이미 제출된 stage가 terminal gate event를 아직 만들지 못한 경우에만
// reporting.ResumeFinalEditGate로 canonical event를 복구한다.
func (runner Runner) ReplayStage(ctx context.Context, input Input, binding reporting.FinalEditStageBinding, stage reporting.FinalEditStageResult, finalBinding *reporting.LongFormFinalizeBinding) (StageRun, error) {
	if finalBinding == nil {
		return StageRun{Binding: binding, Stage: stage}, nil
	}
	final, finalOK, finalErr := reporting.LoadLongFormFinalization(context.WithoutCancel(ctx), runner.Store, *finalBinding)
	if finalErr != nil {
		return StageRun{}, StageFailure(binding.Stage, input.PlanEvent.EventID, finalErr)
	}
	if !finalOK {
		resumed, err := reporting.ResumeFinalEditGate(context.WithoutCancel(ctx), runner.Store, reporting.FinalEditGateResumeRequest{
			StageBinding: binding, FinalBinding: *finalBinding, CanonicalEventID: runner.ID("evt"),
		})
		if err != nil {
			return StageRun{}, StageFailure(binding.Stage, input.PlanEvent.EventID, err)
		}
		final = resumed
	}
	return StageRun{Binding: binding, Stage: stage, Final: final}, nil
}

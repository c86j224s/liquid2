package readeredit

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

const Stage = reporting.FinalEditStageReader

// Runner는 reader edit stage의 prompt, tool allowlist, durable replay를 실행한다.
type Runner struct {
	Store reporting.FinalEditStageStore
	NewID finaledit.IDGenerator
}

// Input은 reader edit source와 provider session lineage를 고정한다.
type Input struct {
	Base                      finaledit.Input
	FinalBase                 reporting.LongFormFinalizeBinding
	SourceArtifactID          string
	ForkFromSessionID         string
	PreviousProviderSessionID string
}

// Output은 reader가 제출한 artifact와 stage binding이다.
type Output struct {
	Run finaledit.StageRun
}

// Run은 기존 reader edit 두-attempt/replay 계약으로 stage를 실행한다.
func (runner Runner) Run(ctx context.Context, input Input, executor agentexec.AgentExecutor, forker agentexec.AgentSessionForker) (Output, error) {
	core := finaledit.Runner{Store: runner.Store, NewID: runner.NewID}
	progress, err := core.StageProgress(ctx, input.FinalBase, input.Base, Stage, input.SourceArtifactID, core.ID("art"), input.ForkFromSessionID, forker, input.PreviousProviderSessionID)
	if err != nil {
		return Output{}, err
	}
	run, err := core.RunStage(ctx, input.Base, progress, nil, MCPTools(), prompt, executor)
	if err != nil {
		return Output{}, err
	}
	return Output{Run: run}, nil
}

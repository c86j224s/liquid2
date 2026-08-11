package styleedit

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

const Stage = reporting.FinalEditStageStyle

// Runner는 style edit stage의 prompt, tools, durable replay를 실행한다.
type Runner struct {
	Store reporting.FinalEditStageStore
	NewID finaledit.IDGenerator
}

// Input은 style edit source와 reader session fork lineage를 고정한다.
type Input struct {
	Base             finaledit.Input
	FinalBase        reporting.LongFormFinalizeBinding
	SourceArtifactID string
	ReaderSessionID  string
}

// Output은 style edit submission 결과다.
type Output struct {
	Run finaledit.StageRun
}

// Run은 기존 style edit 두-attempt/replay 계약으로 stage를 실행한다.
func (runner Runner) Run(ctx context.Context, input Input, executor agentexec.AgentExecutor, forker agentexec.AgentSessionForker) (Output, error) {
	core := finaledit.Runner{Store: runner.Store, NewID: runner.NewID}
	progress, err := core.StageProgress(ctx, input.FinalBase, input.Base, Stage, input.SourceArtifactID, core.ID("art"), input.ReaderSessionID, forker, input.ReaderSessionID)
	if err != nil {
		return Output{}, err
	}
	run, err := core.RunStage(ctx, input.Base, progress, nil, MCPTools(), prompt, executor)
	if err != nil {
		return Output{}, err
	}
	return Output{Run: run}, nil
}

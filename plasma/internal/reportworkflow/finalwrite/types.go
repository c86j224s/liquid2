package finalwrite

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

const Stage = reporting.FinalEditStageWriter

// Runner는 final writer stage의 durable replay와 provider 호출을 실행한다.
type Runner struct {
	Store reporting.FinalEditStageStore
	NewID finaledit.IDGenerator
}

// Input은 writer stage binding을 준비하는 데 필요한 typed 값이다.
type Input struct {
	Base           finaledit.Input
	FinalBase      reporting.LongFormFinalizeBinding
	SourceArtifact string
	PlanSessionID  string
}

// Prepared는 reportassembly가 먼저 검증·materialize할 writer binding이다.
type Prepared struct {
	Base     finaledit.Input
	Progress reporting.FinalEditStageProgress
}

// Output은 writer submission과 다음 reader source artifact를 담는다.
type Output struct {
	Run finaledit.StageRun
}

// Prepare는 기존 final writer recovery/fork 규칙으로 durable binding을 만든다.
func (runner Runner) Prepare(ctx context.Context, input Input, forker agentexec.AgentSessionForker) (Prepared, error) {
	core := finaledit.Runner{Store: runner.Store, NewID: runner.NewID}
	progress, err := core.StageProgress(ctx, input.FinalBase, input.Base, Stage, input.SourceArtifact, core.ID("art"), input.PlanSessionID, forker, input.PlanSessionID)
	if err != nil {
		return Prepared{}, err
	}
	return Prepared{Base: input.Base, Progress: progress}, nil
}

// Run은 writer prompt와 MCP allowlist로 provider를 한 번의 stage 계약 안에서 실행한다.
func (runner Runner) Run(ctx context.Context, input Prepared, executor agentexec.AgentExecutor) (Output, error) {
	core := finaledit.Runner{Store: runner.Store, NewID: runner.NewID}
	run, err := core.RunStage(ctx, input.Base, input.Progress, nil, MCPTools(), prompt, executor)
	if err != nil {
		return Output{}, err
	}
	return Output{Run: run}, nil
}

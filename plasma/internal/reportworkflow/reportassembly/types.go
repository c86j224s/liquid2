package reportassembly

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

const Stage = "reportassembly"

// Runner는 final assembly materialization의 durable replay 경계를 실행한다.
type Runner struct {
	Store reporting.FinalEditStageStore
	NewID finaledit.IDGenerator
}

// Input은 writer binding에서 assembly identity를 검증하기 위한 값이다.
type Input struct {
	Binding         reporting.FinalEditStageBinding
	SkipIfStageOpen bool
	SkipIfSubmitted bool
	PlanEventID     string
}

// Output은 materialized 또는 replayed final assembly 결과다.
type Output struct {
	Result  reporting.FinalEditAssemblyResult
	Replay  bool
	Skipped bool
}

// ReaderSourceInput은 V1 reader가 소비하는 deterministic Part manuscript source를 지정한다.
type ReaderSourceInput struct {
	PlanEventID     string
	PartArtifactIDs []string
}

// ReaderSourceOutput은 V1 reader stage에 전달할 source artifact identity다.
type ReaderSourceOutput struct {
	ArtifactID string
}

// Run은 기존 writer 전 단계의 deterministic assembly 생성 순서와 replay 규칙을 보존한다.
func (runner Runner) Run(ctx context.Context, input Input) (Output, error) {
	if input.SkipIfStageOpen || input.SkipIfSubmitted {
		return Output{Skipped: true}, nil
	}
	core := finaledit.Runner{Store: runner.Store, NewID: runner.NewID}
	result, _, err := reporting.EnsureFinalEditAssembly(ctx, runner.Store, core.ID("evt"), input.Binding)
	if err != nil {
		return Output{}, finaledit.StageFailure(input.Binding.Stage, input.PlanEventID, err)
	}
	return Output{Result: result, Replay: result.Replay}, nil
}

// PrepareReaderSource는 V1 reader source materialization의 기존 deterministic ID를
// side effect 없이 typed stage output으로 만든다.
func (runner Runner) PrepareReaderSource(_ context.Context, input ReaderSourceInput) (ReaderSourceOutput, error) {
	return ReaderSourceOutput{ArtifactID: ReaderSourceArtifactID(input.PlanEventID, input.PartArtifactIDs)}, nil
}

// AssemblyArtifactID는 canonical plan과 ordered Part artifact IDs에서 deterministic ID를 만든다.
func AssemblyArtifactID(planEventID string, partArtifactIDs []string) string {
	return reporting.FinalEditAssemblyArtifactID(planEventID, partArtifactIDs)
}

// ReaderSourceArtifactID는 V1 reader source materialization이 쓰는 deterministic ID다.
func ReaderSourceArtifactID(planEventID string, partArtifactIDs []string) string {
	return reporting.FinalEditReaderSourceArtifactID(planEventID, partArtifactIDs)
}

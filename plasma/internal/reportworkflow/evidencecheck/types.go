package evidencecheck

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

type Kind string

const (
	KindCorrective Kind = "corrective"
	KindEvidence   Kind = "evidence"
)

// Runner는 terminal gate stage의 prompt, tools, durable replay/resume을 실행한다.
type Runner struct {
	Store reporting.FinalEditStageStore
	NewID finaledit.IDGenerator
}

// Input은 canonical finalization으로 승격할 source와 gate session lineage를 고정한다.
type Input struct {
	Base                      finaledit.Input
	FinalBase                 reporting.LongFormFinalizeBinding
	Kind                      Kind
	SourceArtifactID          string
	PreviousProviderSessionID string
	PlanSessionID             string
}

// Output은 terminal gate submission과 canonical finalization binding/result다.
type Output struct {
	Run          finaledit.StageRun
	FinalBinding reporting.LongFormFinalizeBinding
}

// Run은 기존 terminal gate 두-attempt/replay/resume 계약으로 canonical finalization을 만든다.
func (runner Runner) Run(ctx context.Context, input Input, executor agentexec.AgentExecutor, forker agentexec.AgentSessionForker) (Output, error) {
	core := finaledit.Runner{Store: runner.Store, NewID: runner.NewID}
	var progress reporting.FinalEditStageProgress
	var finalBinding reporting.LongFormFinalizeBinding
	var err error
	tools := gateMCPToolsForHumanize(input.Base.PostReportHumanize)
	prompt := promptForHumanize(input.Base.PostReportHumanize)
	if input.Kind == KindEvidence {
		progress, finalBinding, err = core.EvidenceGateProgress(ctx, input.Base, input.FinalBase, input.SourceArtifactID, input.PreviousProviderSessionID, input.PlanSessionID, forker)
		tools = evidenceMCPTools()
		prompt = evidencePrompt
	} else {
		progress, finalBinding, err = core.GateProgress(ctx, input.Base, input.FinalBase, input.SourceArtifactID, input.PreviousProviderSessionID, input.PlanSessionID, forker)
	}
	if err != nil {
		return Output{}, err
	}
	run, err := core.RunStage(ctx, input.Base, progress, &finalBinding, tools, prompt, executor)
	if err != nil {
		return Output{}, err
	}
	return Output{Run: run, FinalBinding: finalBinding}, nil
}

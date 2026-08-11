package reportworkflow

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/evidencecheck"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/finalstore"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/finalwrite"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/readeredit"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/reportassembly"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/semanticcheck"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/styleedit"
)

// runWriterPath는 V2/V3 writer source materialization과 finalwrite stage 순서를 고정한다.
func (runner Runner) runWriterPath(ctx context.Context, base finaledit.Input, recoveryFinal reporting.LongFormFinalizeBinding, planSessionID string, forker agentexec.AgentSessionForker) (finalwrite.Output, error) {
	source := reportassembly.AssemblyArtifactID(base.PlanEvent.EventID, base.PartArtifactIDs)
	prepared, err := runner.finalWriteRunner.Prepare(ctx, finalwrite.Input{Base: base, FinalBase: recoveryFinal, SourceArtifact: source, PlanSessionID: planSessionID}, forker)
	if err != nil {
		return finalwrite.Output{}, err
	}
	doneAssembly := runner.observeStart(NodeReportAssembly)
	_, err = runner.reportAssemblyRunner.Run(ctx, reportassembly.Input{
		Binding: prepared.Progress.Binding, PlanEventID: base.PlanEvent.EventID,
		SkipIfStageOpen: prepared.Progress.StartEvent.EventID != "", SkipIfSubmitted: prepared.Progress.Submission != nil,
	})
	doneAssembly(err, false)
	if err != nil {
		return finalwrite.Output{}, err
	}
	doneWriter := runner.observeStart(NodeFinalWrite)
	writer, err := runner.finalWriteRunner.Run(ctx, prepared, runner.executor)
	doneWriter(err, prepared.Progress.Submission != nil)
	return writer, err
}

// runReaderSource는 V1 reader가 읽는 deterministic source ID를 reportassembly stage 경계에서 만든다.
func (runner Runner) runReaderSource(ctx context.Context, base finaledit.Input) (reportassembly.ReaderSourceOutput, error) {
	done := runner.observeStart(NodeReportAssembly)
	out, err := runner.reportAssemblyRunner.PrepareReaderSource(ctx, reportassembly.ReaderSourceInput{
		PlanEventID: base.PlanEvent.EventID, PartArtifactIDs: base.PartArtifactIDs,
	})
	done(err, false)
	return out, err
}

// runReader는 readeredit stage의 provider call과 durable replay를 stage package에 위임한다.
func (runner Runner) runReader(ctx context.Context, base finaledit.Input, recoveryFinal reporting.LongFormFinalizeBinding, sourceArtifactID string, forkFromSessionID string, previousSessionID string, forker agentexec.AgentSessionForker) (readeredit.Output, error) {
	done := runner.observeStart(NodeReaderEdit)
	out, err := runner.readerEditRunner.Run(ctx, readeredit.Input{
		Base: base, FinalBase: recoveryFinal, SourceArtifactID: sourceArtifactID,
		ForkFromSessionID: forkFromSessionID, PreviousProviderSessionID: previousSessionID,
	}, runner.executor, forker)
	done(err, false)
	return out, err
}

// runStyle은 styleedit가 reader provider session에서 fork된다는 계보 규칙을 유지한다.
func (runner Runner) runStyle(ctx context.Context, base finaledit.Input, recoveryFinal reporting.LongFormFinalizeBinding, prior finaledit.StageRun, forker agentexec.AgentSessionForker) (styleedit.Output, error) {
	done := runner.observeStart(NodeStyleEdit)
	out, err := runner.styleEditRunner.Run(ctx, styleedit.Input{
		Base: base, FinalBase: recoveryFinal, SourceArtifactID: prior.Stage.Artifact.ArtifactID,
		ReaderSessionID: prior.Binding.ProviderSessionID,
	}, runner.executor, forker)
	done(err, false)
	return out, err
}

// runSemantic은 V3 semantic stage의 plan-session fork 원천을 root graph에서 고정한다.
func (runner Runner) runSemantic(ctx context.Context, base finaledit.Input, recoveryFinal reporting.LongFormFinalizeBinding, prior finaledit.StageRun, planSessionID string, forker agentexec.AgentSessionForker) (semanticcheck.Output, error) {
	done := runner.observeStart(NodeSemanticCheck)
	out, err := runner.semanticCheckRunner.Run(ctx, semanticcheck.Input{
		Base: base, FinalBase: recoveryFinal, SourceArtifactID: prior.Stage.Artifact.ArtifactID,
		PlanSessionID: planSessionID,
	}, runner.executor, forker)
	done(err, false)
	return out, err
}

// runGateAndAdopt는 terminal gate replay 뒤 finalstore가 이미 저장된 canonical 결과만 채택하게 한다.
func (runner Runner) runGateAndAdopt(ctx context.Context, base finaledit.Input, recoveryFinal reporting.LongFormFinalizeBinding, kind evidencecheck.Kind, sourceArtifactID string, previousProviderSessionID string, planSessionID string, forker agentexec.AgentSessionForker) (DraftOutput, error) {
	doneGate := runner.observeStart(NodeEvidenceCheck)
	gate, err := runner.evidenceCheckRunner.Run(ctx, evidencecheck.Input{
		Base: base, FinalBase: recoveryFinal, Kind: kind, SourceArtifactID: sourceArtifactID,
		PreviousProviderSessionID: previousProviderSessionID, PlanSessionID: planSessionID,
	}, runner.executor, forker)
	doneGate(err, false)
	if err != nil {
		return DraftOutput{}, err
	}
	doneStore := runner.observeStart(NodeFinalStore)
	stored, err := runner.finalStoreRunner.AdoptGate(context.WithoutCancel(ctx), finalstore.GateInput{GateReadRequest: finalstore.GateReadRequest{
		MissionID: base.MissionID, PendingEventID: base.PendingEventID, PlanEventID: base.PlanEvent.EventID,
		ArtifactID: base.ArtifactID, Binding: gate.FinalBinding,
	}})
	doneStore(err, false)
	if err != nil {
		return DraftOutput{}, err
	}
	return draftOutput(stored), nil
}

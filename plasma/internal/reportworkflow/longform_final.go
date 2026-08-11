package reportworkflow

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/evidencecheck"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

// RunLongForm은 prefix graph와 frozen finalization tail을 하나의 typed report graph로 실행한다.
func (runner Runner) RunLongForm(ctx context.Context, input DraftInput) (DraftOutput, error) {
	prefix, err := runner.RunLongFormPrefix(ctx, input)
	if err != nil {
		return DraftOutput{}, err
	}
	return runner.FinalizeLongFormPrefix(ctx, prefix)
}

// FinalizeLongFormPrefix는 이미 typed prefix handoff로 고정된 long-form final tail만 실행한다.
//
// FinalTail 외의 caller 선택 정책은 받지 않는다. 알 수 없는 tail은 legacy로
// 암묵 변환하지 않고 conflict로 거부한다.
func (runner Runner) FinalizeLongFormPrefix(ctx context.Context, prefix PrefixOutput) (DraftOutput, error) {
	switch prefix.FinalTail {
	case FinalTailLegacy:
		return runner.runLegacyFinalTail(ctx, prefix)
	case FinalTailV3:
		return runner.runFinalEditV3(ctx, prefix)
	case FinalTailV2:
		return runner.runFinalEditV2(ctx, prefix)
	case FinalTailV1:
		return runner.runFinalEditV1(ctx, prefix)
	default:
		return DraftOutput{}, fmt.Errorf("%w: unsupported final tail %q", producterror.ErrConflict, prefix.FinalTail)
	}
}

func (runner Runner) finalEditBase(prefix PrefixOutput) finaledit.Input {
	return finaledit.Input{
		MissionID: prefix.MissionID, Title: prefix.Title, ExecutorName: prefix.AgentExecutor,
		AgentModel: prefix.AgentModel, AgentReasoningEffort: prefix.AgentReasoningEffort,
		AgentSelectionSource: prefix.AgentSelectionSource, MCPMode: prefix.MCPMode,
		Rigor:               finaledit.Rigor{Level: prefix.Rigor.Level, Label: prefix.Rigor.Label},
		ReportSessionPolicy: prefix.ReportSessionPolicy, ReportSessionPolicySelection: prefix.ReportSessionPolicySelection,
		PostReportHumanize: prefix.PostReportHumanize, GenerationGuidanceProfile: prefix.GenerationGuidanceProfile,
		GenerationGuidanceSHA256: prefix.GenerationGuidanceSHA256, PendingEventID: prefix.PendingEventID,
		DirectionHint: prefix.DirectionHint, ArtifactID: prefix.ArtifactID, PlanEvent: prefix.PlanEvent,
		Plan: prefix.Plan, RequirementMap: prefix.RequirementMap, PartArtifactIDs: prefix.PartArtifactIDs,
		SectionArtifactIDs: prefix.SectionArtifactIDs, SectionWordTotal: prefix.SectionWordTotal,
		SessionChainKind: prefix.SessionChainKind, PreReportResearchSessionID: prefix.PreReportResearchSessionID,
		ReportPlanSessionID: prefix.ReportPlanSessionID, ForkSourceAgentSessionID: prefix.ForkSourceAgentSessionID,
		Started: prefix.StartedAt,
	}
}

func (runner Runner) runFinalEditV3(ctx context.Context, prefix PrefixOutput) (DraftOutput, error) {
	base, recoveryFinal, planSessionID, forker, err := runner.finalEditStart(prefix)
	if err != nil {
		return DraftOutput{}, err
	}
	writer, err := runner.runWriterPath(ctx, base, recoveryFinal, planSessionID, forker)
	if err != nil {
		return DraftOutput{}, err
	}
	reader, err := runner.runReader(ctx, base, recoveryFinal, writer.Run.Stage.Artifact.ArtifactID, planSessionID, planSessionID, forker)
	if err != nil {
		return DraftOutput{}, err
	}
	prior := reader.Run
	if prefix.PostReportHumanize == reporting.FinalEditHumanizeEnabled {
		style, err := runner.runStyle(ctx, base, recoveryFinal, prior, forker)
		if err != nil {
			return DraftOutput{}, err
		}
		semantic, err := runner.runSemantic(ctx, base, recoveryFinal, style.Run, planSessionID, forker)
		if err != nil {
			return DraftOutput{}, err
		}
		prior = semantic.Run
	}
	return runner.runGateAndAdopt(ctx, base, recoveryFinal, evidencecheck.KindEvidence, prior.Stage.Artifact.ArtifactID, planSessionID, planSessionID, forker)
}

func (runner Runner) runFinalEditV2(ctx context.Context, prefix PrefixOutput) (DraftOutput, error) {
	base, recoveryFinal, planSessionID, forker, err := runner.finalEditStart(prefix)
	if err != nil {
		return DraftOutput{}, err
	}
	writer, err := runner.runWriterPath(ctx, base, recoveryFinal, planSessionID, forker)
	if err != nil {
		return DraftOutput{}, err
	}
	reader, err := runner.runReader(ctx, base, recoveryFinal, writer.Run.Stage.Artifact.ArtifactID, planSessionID, planSessionID, forker)
	if err != nil {
		return DraftOutput{}, err
	}
	prior := reader.Run
	if prefix.PostReportHumanize == reporting.FinalEditHumanizeEnabled {
		style, err := runner.runStyle(ctx, base, recoveryFinal, prior, forker)
		if err != nil {
			return DraftOutput{}, err
		}
		prior = style.Run
	}
	return runner.runGateAndAdopt(ctx, base, recoveryFinal, evidencecheck.KindCorrective, prior.Stage.Artifact.ArtifactID, planSessionID, planSessionID, forker)
}

func (runner Runner) runFinalEditV1(ctx context.Context, prefix PrefixOutput) (DraftOutput, error) {
	base, recoveryFinal, planSessionID, forker, err := runner.finalEditStart(prefix)
	if err != nil {
		return DraftOutput{}, err
	}
	source, err := runner.runReaderSource(ctx, base)
	if err != nil {
		return DraftOutput{}, err
	}
	reader, err := runner.runReader(ctx, base, recoveryFinal, source.ArtifactID, planSessionID, "", forker)
	if err != nil {
		return DraftOutput{}, err
	}
	prior := reader.Run
	if prefix.PostReportHumanize == reporting.FinalEditHumanizeEnabled {
		style, err := runner.runStyle(ctx, base, recoveryFinal, prior, forker)
		if err != nil {
			return DraftOutput{}, err
		}
		prior = style.Run
	}
	return runner.runGateAndAdopt(ctx, base, recoveryFinal, evidencecheck.KindCorrective, prior.Stage.Artifact.ArtifactID, prior.Binding.ProviderSessionID, planSessionID, forker)
}

func (runner Runner) finalEditStart(prefix PrefixOutput) (finaledit.Input, reporting.LongFormFinalizeBinding, string, agentexec.AgentSessionForker, error) {
	base := runner.finalEditBase(prefix)
	forker, ok := runner.executor.(agentexec.AgentSessionForker)
	if !ok {
		return base, reporting.LongFormFinalizeBinding{}, "", nil, finaledit.StageFailure("final_edit", prefix.PlanEvent.EventID, fmt.Errorf("%w: final edit pipeline requires an agent session forker", producterror.ErrInvalidInput))
	}
	planSessionID := finaledit.FirstNonEmpty(prefix.ReportPlanSessionID, prefix.ForkSourceAgentSessionID)
	if planSessionID == "" {
		return base, reporting.LongFormFinalizeBinding{}, "", nil, finaledit.StageFailure("final_edit", prefix.PlanEvent.EventID, fmt.Errorf("%w: final edit pipeline requires a report plan session", producterror.ErrConflict))
	}
	core := finaledit.Runner{Store: runner.finalEditStore, NewID: runner.newID}
	recoveryFinal := base.LongFormFinalBinding(core.ID("ses"), planSessionID, planSessionID, finaledit.FirstNonEmpty(prefix.ForkSourceAgentSessionID, planSessionID))
	return base, recoveryFinal, planSessionID, forker, nil
}

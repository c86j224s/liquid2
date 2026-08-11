package finaledit

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// StageProgress는 한 final edit stage의 durable start/submission replay를 먼저 찾고,
// 없으면 기존 fork 규칙으로 새 provider session binding을 만든다.
func (runner Runner) StageProgress(ctx context.Context, final reporting.LongFormFinalizeBinding, input Input, stage string, sourceArtifactID string, editedArtifactID string, forkFromSessionID string, forker agentexec.AgentSessionForker, previousProviderSessionID string) (reporting.FinalEditStageProgress, error) {
	if recovered, ok, err := reporting.LoadFinalEditStageProgress(ctx, runner.Store, reporting.FinalEditStageStartContract{FinalBinding: final, Stage: stage}); err != nil {
		return reporting.FinalEditStageProgress{}, StageFailure(stage, input.PlanEvent.EventID, err)
	} else if ok {
		return recovered, nil
	}
	providerSessionID, forkSourceID, err := ForkSession(ctx, forker, forkFromSessionID)
	if err != nil {
		return reporting.FinalEditStageProgress{}, StageFailure(stage, input.PlanEvent.EventID, err)
	}
	if strings.TrimSpace(previousProviderSessionID) == "" {
		previousProviderSessionID = providerSessionID
	}
	return reporting.FinalEditStageProgress{Binding: input.FinalEditStageBinding(stage, sourceArtifactID, editedArtifactID, runner.ID("ses"), providerSessionID, previousProviderSessionID, forkSourceID)}, nil
}

// GateProgress는 V1/V2 corrective gate의 durable binding을 복구하거나 만든다.
func (runner Runner) GateProgress(ctx context.Context, input Input, finalBase reporting.LongFormFinalizeBinding, sourceArtifactID string, previousProviderSessionID string, planSessionID string, forker agentexec.AgentSessionForker) (reporting.FinalEditStageProgress, reporting.LongFormFinalizeBinding, error) {
	return runner.finalGateProgress(ctx, input, finalBase, reporting.FinalEditStageGate, sourceArtifactID, previousProviderSessionID, planSessionID, forker)
}

// EvidenceGateProgress는 V3 evidence gate의 durable binding을 복구하거나 만든다.
func (runner Runner) EvidenceGateProgress(ctx context.Context, input Input, finalBase reporting.LongFormFinalizeBinding, sourceArtifactID string, previousProviderSessionID string, planSessionID string, forker agentexec.AgentSessionForker) (reporting.FinalEditStageProgress, reporting.LongFormFinalizeBinding, error) {
	return runner.finalGateProgress(ctx, input, finalBase, reporting.FinalEditStageEvidenceGate, sourceArtifactID, previousProviderSessionID, planSessionID, forker)
}

func (runner Runner) finalGateProgress(ctx context.Context, input Input, finalBase reporting.LongFormFinalizeBinding, stage string, sourceArtifactID string, previousProviderSessionID string, planSessionID string, forker agentexec.AgentSessionForker) (reporting.FinalEditStageProgress, reporting.LongFormFinalizeBinding, error) {
	if recovered, ok, err := reporting.LoadFinalEditStageProgress(ctx, runner.Store, reporting.FinalEditStageStartContract{FinalBinding: finalBase, Stage: stage}); err != nil {
		return reporting.FinalEditStageProgress{}, reporting.LongFormFinalizeBinding{}, StageFailure(stage, input.PlanEvent.EventID, err)
	} else if ok {
		final := input.LongFormFinalBinding(recovered.Binding.ToolSessionID, recovered.Binding.ProviderSessionID, recovered.Binding.PreviousProviderSessionID, recovered.Binding.ForkSourceAgentSessionID)
		final.Producer = recovered.Binding.Producer
		if err := reporting.ValidateFinalEditGateBindingsCompatible(recovered.Binding, final); err != nil {
			return reporting.FinalEditStageProgress{}, reporting.LongFormFinalizeBinding{}, err
		}
		return recovered, final, nil
	}
	gateSessionID, gateForkSourceID, err := ForkSession(ctx, forker, planSessionID)
	if err != nil {
		return reporting.FinalEditStageProgress{}, reporting.LongFormFinalizeBinding{}, StageFailure(stage, input.PlanEvent.EventID, err)
	}
	final := input.LongFormFinalBinding(runner.ID("ses"), gateSessionID, previousProviderSessionID, gateForkSourceID)
	gate := input.FinalEditStageBinding(stage, sourceArtifactID, input.ArtifactID, final.ToolSessionID, final.ProviderSessionID, final.PreviousProviderSessionID, final.ForkSourceAgentSessionID)
	if err := reporting.ValidateFinalEditGateBindingsCompatible(gate, final); err != nil {
		return reporting.FinalEditStageProgress{}, reporting.LongFormFinalizeBinding{}, err
	}
	return reporting.FinalEditStageProgress{Binding: gate}, final, nil
}

// ForkSession은 final edit V1/V2/V3의 기존 session fork 오류와 fallback 규칙을 보존한다.
func ForkSession(ctx context.Context, forker agentexec.AgentSessionForker, sourceSessionID string) (string, string, error) {
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	if sourceSessionID == "" {
		return "", "", fmt.Errorf("%w: final edit requires a source provider session", producterror.ErrConflict)
	}
	fork, err := forker.ForkSession(ctx, sourceSessionID)
	if err != nil {
		return "", "", fmt.Errorf("final edit session fork failed: %w", err)
	}
	if strings.TrimSpace(fork.SessionID) == "" {
		return "", "", fmt.Errorf("%w: final edit session fork returned an empty session", producterror.ErrConflict)
	}
	return strings.TrimSpace(fork.SessionID), FirstNonEmpty(strings.TrimSpace(fork.SourceSessionID), sourceSessionID), nil
}

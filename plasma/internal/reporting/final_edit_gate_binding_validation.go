package reporting

import (
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// ValidateFinalEditGateBindingsCompatible checks the plan-independent
// contract shared by the corrective gate and canonical finalization.
func ValidateFinalEditGateBindingsCompatible(stage FinalEditStageBinding, final LongFormFinalizeBinding) error {
	stage = normalizeFinalEditStageBinding(stage)
	final = normalizeLongFormFinalizeBinding(final)
	if err := validateFinalEditStageBinding(stage); err != nil {
		return err
	}
	if err := validateLongFormFinalizeBinding(final); err != nil {
		return err
	}
	if stage.Stage != FinalEditStageGate || final.CompositionStrategy != LongFormCompositionNarrativeEdit {
		return fmt.Errorf("%w: corrective gate requires narrative final edit bindings", app.ErrInvalidInput)
	}
	if stage.MissionID != final.MissionID ||
		stage.PendingEventID != final.PendingEventID ||
		stage.PlanEventID != final.PlanEventID ||
		stage.Title != final.Title ||
		stage.Filename != final.Filename ||
		stage.EditedArtifactID != final.ArtifactID ||
		stage.ToolSessionID != final.ToolSessionID ||
		stage.ProviderSessionID != final.ProviderSessionID ||
		stage.PreviousProviderSessionID != final.PreviousProviderSessionID ||
		stage.ForkSourceAgentSessionID != final.ForkSourceAgentSessionID ||
		stage.Producer != final.Producer ||
		stage.AgentExecutor != final.AgentExecutor ||
		stage.AgentModel != final.AgentModel ||
		stage.AgentReasoningEffort != final.AgentReasoningEffort ||
		stage.AgentSelectionSource != final.AgentSelectionSource ||
		stage.MCPMode != final.MCPMode ||
		stage.RigorLevel != final.RigorLevel ||
		stage.RigorLabel != final.RigorLabel ||
		stage.ReportSessionPolicy != final.ReportSessionPolicy ||
		stage.ReportSessionPolicySelection != final.ReportSessionPolicySelection ||
		stage.PostReportHumanize != final.PostReportHumanize ||
		stage.GenerationGuidanceProfile != final.GenerationGuidanceProfile ||
		stage.GenerationGuidanceSHA256 != final.GenerationGuidanceSHA256 ||
		stage.SessionChainKind != final.SessionChainKind ||
		stage.PreReportResearchSessionID != final.PreReportResearchSessionID ||
		stage.ReportPlanSessionID != final.ReportPlanSessionID {
		return fmt.Errorf("%w: corrective gate binding differs from final binding", app.ErrConflict)
	}
	return nil
}

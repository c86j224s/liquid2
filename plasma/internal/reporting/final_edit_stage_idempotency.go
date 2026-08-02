package reporting

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func FinalEditStageIdempotencyKey(stage string, pendingEventID string, planEventID string) string {
	switch strings.TrimSpace(stage) {
	case FinalEditStageWriter:
		return "report-final-edit-writer:" + strings.TrimSpace(pendingEventID) + ":" + strings.TrimSpace(planEventID)
	case FinalEditStageReader:
		return "report-final-edit-reader:" + strings.TrimSpace(pendingEventID) + ":" + strings.TrimSpace(planEventID)
	case FinalEditStageStyle:
		return "report-final-edit-style:" + strings.TrimSpace(pendingEventID) + ":" + strings.TrimSpace(planEventID)
	case FinalEditStageGate:
		return "report-final-edit-gate:" + strings.TrimSpace(pendingEventID) + ":" + strings.TrimSpace(planEventID)
	case FinalEditStageStyleSemanticValidation:
		return "report-final-edit-style-semantic-validation:" + strings.TrimSpace(pendingEventID) + ":" + strings.TrimSpace(planEventID)
	case FinalEditStageEvidenceGate:
		return "report-final-edit-evidence-gate:" + strings.TrimSpace(pendingEventID) + ":" + strings.TrimSpace(planEventID)
	default:
		return ""
	}
}

func normalizeFinalEditStageBinding(value FinalEditStageBinding) FinalEditStageBinding {
	value.MissionID = strings.TrimSpace(value.MissionID)
	value.PendingEventID = strings.TrimSpace(value.PendingEventID)
	value.PlanEventID = strings.TrimSpace(value.PlanEventID)
	value.FinalEditPipeline = strings.TrimSpace(value.FinalEditPipeline)
	value.Title = strings.TrimSpace(value.Title)
	value.Stage = strings.TrimSpace(value.Stage)
	value.SourceArtifactID = strings.TrimSpace(value.SourceArtifactID)
	value.EditedArtifactID = strings.TrimSpace(value.EditedArtifactID)
	value.Filename = strings.TrimSpace(value.Filename)
	value.ToolSessionID = strings.TrimSpace(value.ToolSessionID)
	value.ProviderSessionID = strings.TrimSpace(value.ProviderSessionID)
	value.PreviousProviderSessionID = strings.TrimSpace(value.PreviousProviderSessionID)
	value.IdempotencyKey = strings.TrimSpace(value.IdempotencyKey)
	value.AgentExecutor = strings.TrimSpace(strings.ToLower(value.AgentExecutor))
	value.AgentModel = strings.TrimSpace(value.AgentModel)
	value.AgentReasoningEffort = strings.TrimSpace(value.AgentReasoningEffort)
	value.AgentSelectionSource = strings.TrimSpace(value.AgentSelectionSource)
	value.MCPMode = strings.TrimSpace(value.MCPMode)
	value.RigorLevel = strings.TrimSpace(value.RigorLevel)
	value.RigorLabel = strings.TrimSpace(value.RigorLabel)
	value.ReportSessionPolicy = strings.TrimSpace(value.ReportSessionPolicy)
	value.ReportSessionPolicySelection = strings.TrimSpace(value.ReportSessionPolicySelection)
	value.PostReportHumanize = strings.TrimSpace(value.PostReportHumanize)
	value.GenerationGuidanceProfile = strings.TrimSpace(value.GenerationGuidanceProfile)
	value.GenerationGuidanceSHA256 = strings.TrimSpace(value.GenerationGuidanceSHA256)
	value.SessionChainKind = strings.TrimSpace(value.SessionChainKind)
	value.PreReportResearchSessionID = strings.TrimSpace(value.PreReportResearchSessionID)
	value.ReportPlanSessionID = strings.TrimSpace(value.ReportPlanSessionID)
	value.ForkSourceAgentSessionID = strings.TrimSpace(value.ForkSourceAgentSessionID)
	value.Producer.Type = strings.TrimSpace(value.Producer.Type)
	value.Producer.ID = strings.TrimSpace(value.Producer.ID)
	return value
}

func validateFinalEditStageBinding(value FinalEditStageBinding) error {
	if value.MissionID == "" ||
		value.PendingEventID == "" ||
		value.PlanEventID == "" ||
		value.Title == "" ||
		value.Stage == "" ||
		value.SourceArtifactID == "" ||
		value.EditedArtifactID == "" ||
		value.Filename == "" ||
		value.ToolSessionID == "" ||
		value.ProviderSessionID == "" ||
		value.PreviousProviderSessionID == "" ||
		value.IdempotencyKey == "" ||
		value.AgentExecutor == "" ||
		value.AgentModel == "" ||
		value.AgentReasoningEffort == "" ||
		value.AgentSelectionSource == "" ||
		value.MCPMode == "" ||
		value.ReportSessionPolicy == "" ||
		value.ReportSessionPolicySelection == "" ||
		value.PostReportHumanize == "" ||
		value.GenerationGuidanceProfile == "" ||
		value.GenerationGuidanceSHA256 == "" ||
		value.SessionChainKind == "" ||
		value.ReportPlanSessionID == "" {
		return fmt.Errorf("%w: final edit stage binding is incomplete", app.ErrInvalidInput)
	}
	if finalEditStartedEventType(value.Stage) == "" {
		return fmt.Errorf("%w: unsupported final edit stage", app.ErrInvalidInput)
	}
	if value.FinalEditPipeline != "" && !isSupportedFinalEditPipeline(value.FinalEditPipeline) {
		return fmt.Errorf("%w: unsupported final edit pipeline", app.ErrInvalidInput)
	}
	if value.Stage == FinalEditStageWriter && value.FinalEditPipeline != "" &&
		value.FinalEditPipeline != FinalEditPipelineAssemblyWriterReaderStyleGateV2 &&
		value.FinalEditPipeline != FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		return fmt.Errorf("%w: final writer stage requires an assembly final edit pipeline", app.ErrInvalidInput)
	}
	if value.IdempotencyKey != FinalEditStageIdempotencyKey(value.Stage, value.PendingEventID, value.PlanEventID) {
		return fmt.Errorf("%w: final edit stage idempotency key differs from contract", app.ErrInvalidInput)
	}
	if value.Producer.Type != "agent_session" || value.Producer.ID != value.ProviderSessionID {
		return fmt.Errorf("%w: final edit stage producer must be the bound provider session", app.ErrInvalidInput)
	}
	return nil
}

func ValidateFinalEditStageBinding(value FinalEditStageBinding) error {
	return validateFinalEditStageBinding(normalizeFinalEditStageBinding(value))
}

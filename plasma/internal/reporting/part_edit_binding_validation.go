package reporting

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func ValidatePartEditBinding(value PartEditBinding) error {
	value = normalizePartEditBinding(value)
	if value.MissionID == "" ||
		value.PendingEventID == "" ||
		value.PlanEventID == "" ||
		value.SourcePartEventID == "" ||
		value.SourceArtifactID == "" ||
		value.EditedArtifactID == "" ||
		value.Filename == "" ||
		value.ToolSessionID == "" ||
		value.ProviderSessionID == "" ||
		value.PreviousProviderSessionID == "" ||
		value.IdempotencyKey == "" ||
		value.PartIndex < 1 ||
		value.AgentExecutor == "" ||
		value.MCPMode == "" ||
		value.ReportSessionPolicy == "" ||
		value.GenerationGuidanceProfile == "" ||
		value.SessionChainKind == "" ||
		value.ReportPlanSessionID == "" ||
		value.ForkSourceAgentSessionID == "" {
		return fmt.Errorf("%w: part edit binding is incomplete", app.ErrInvalidInput)
	}
	if (value.RequirementMapEventID == "") != (value.RequirementMapHash == "") {
		return fmt.Errorf("%w: part edit requirement map binding is incomplete", app.ErrInvalidInput)
	}
	if value.PreviousProviderSessionID != value.ProviderSessionID {
		return fmt.Errorf("%w: part edit provider session chain differs", app.ErrInvalidInput)
	}
	if value.ProviderSessionID == value.ReportPlanSessionID {
		return fmt.Errorf("%w: part edit provider session must differ from report plan session", app.ErrInvalidInput)
	}
	if value.ForkSourceAgentSessionID != value.ReportPlanSessionID {
		return fmt.Errorf("%w: part edit fork source must be the report plan session", app.ErrInvalidInput)
	}
	if value.EditedArtifactID == value.SourceArtifactID {
		return fmt.Errorf("%w: part edit target artifact must differ from source", app.ErrInvalidInput)
	}
	expectedKey := fmt.Sprintf("report-part-edit:%s:%s:%d", value.PendingEventID, value.PlanEventID, value.PartIndex)
	if value.IdempotencyKey != expectedKey {
		return fmt.Errorf("%w: part edit idempotency key differs from binding", app.ErrInvalidInput)
	}
	return nil
}

func normalizePartEditBinding(value PartEditBinding) PartEditBinding {
	value.MissionID = strings.TrimSpace(value.MissionID)
	value.PendingEventID = strings.TrimSpace(value.PendingEventID)
	value.PlanEventID = strings.TrimSpace(value.PlanEventID)
	value.SourcePartEventID = strings.TrimSpace(value.SourcePartEventID)
	value.SourceArtifactID = strings.TrimSpace(value.SourceArtifactID)
	value.EditedArtifactID = strings.TrimSpace(value.EditedArtifactID)
	value.Filename = strings.TrimSpace(value.Filename)
	value.ToolSessionID = strings.TrimSpace(value.ToolSessionID)
	value.ProviderSessionID = strings.TrimSpace(value.ProviderSessionID)
	value.PreviousProviderSessionID = strings.TrimSpace(value.PreviousProviderSessionID)
	value.IdempotencyKey = strings.TrimSpace(value.IdempotencyKey)
	value.RequirementMapEventID = strings.TrimSpace(value.RequirementMapEventID)
	value.RequirementMapHash = strings.TrimSpace(value.RequirementMapHash)
	value.AgentExecutor = strings.TrimSpace(strings.ToLower(value.AgentExecutor))
	value.AgentModel = strings.TrimSpace(value.AgentModel)
	value.AgentReasoningEffort = strings.TrimSpace(value.AgentReasoningEffort)
	value.AgentSelectionSource = strings.TrimSpace(value.AgentSelectionSource)
	value.MCPMode = strings.TrimSpace(value.MCPMode)
	value.ReportSessionPolicy = strings.TrimSpace(value.ReportSessionPolicy)
	value.ReportSessionPolicySelection = strings.TrimSpace(value.ReportSessionPolicySelection)
	value.GenerationGuidanceProfile = strings.TrimSpace(value.GenerationGuidanceProfile)
	value.GenerationGuidanceSHA256 = strings.TrimSpace(value.GenerationGuidanceSHA256)
	value.SessionChainKind = strings.TrimSpace(value.SessionChainKind)
	value.ReportPlanSessionID = strings.TrimSpace(value.ReportPlanSessionID)
	value.ForkSourceAgentSessionID = strings.TrimSpace(value.ForkSourceAgentSessionID)
	return value
}

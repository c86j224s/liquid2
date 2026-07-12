package reporting

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentmodels"
)

const (
	AgentSelectionSourceExplicitRequest = "explicit_request"
	AgentSelectionSourceMissionSession  = "mission_session"
	AgentSelectionSourceProviderDefault = "provider_default"
)

type ModelSelectionInput struct {
	Executor                 string
	RequestModel             string
	RequestReasoningEffort   string
	SessionModel             string
	SessionReasoningEffort   string
	ProviderModel            string
	ProviderReasoningEffort  string
	ReasoningEffortSupported bool
}

type ModelSelection struct {
	Model           string
	ReasoningEffort string
	Source          string
}

func ResolveModelSelection(input ModelSelectionInput) (ModelSelection, error) {
	requestModel := strings.TrimSpace(input.RequestModel)
	requestEffort := strings.ToLower(strings.TrimSpace(input.RequestReasoningEffort))
	selection := ModelSelection{}

	if requestModel != "" || requestEffort != "" {
		selection.Source = AgentSelectionSourceExplicitRequest
	} else if strings.TrimSpace(input.SessionModel) != "" || strings.TrimSpace(input.SessionReasoningEffort) != "" {
		selection.Source = AgentSelectionSourceMissionSession
	} else {
		selection.Source = AgentSelectionSourceProviderDefault
	}

	selection.Model = requestModel
	if selection.Model == "" {
		selection.Model = strings.TrimSpace(input.SessionModel)
	}
	if selection.Model == "" {
		selection.Model = strings.TrimSpace(input.ProviderModel)
	}

	if !input.ReasoningEffortSupported {
		if requestEffort != "" {
			return ModelSelection{}, fmt.Errorf("executor %q does not support reasoning effort", strings.TrimSpace(input.Executor))
		}
		return selection, nil
	}

	selection.ReasoningEffort = requestEffort
	if selection.ReasoningEffort == "" && requestModel == "" {
		selection.ReasoningEffort = strings.ToLower(strings.TrimSpace(input.SessionReasoningEffort))
		if selection.ReasoningEffort == "" {
			selection.ReasoningEffort = strings.ToLower(strings.TrimSpace(input.ProviderReasoningEffort))
		}
	}

	if strings.EqualFold(strings.TrimSpace(input.Executor), "codex") {
		model, effort, err := agentmodels.Resolve(selection.Model, selection.ReasoningEffort)
		if err != nil {
			return ModelSelection{}, err
		}
		selection.Model, selection.ReasoningEffort = model, effort
	}
	return selection, nil
}

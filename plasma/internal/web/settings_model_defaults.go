package web

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentmodels"
	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type modelDefaultsRequest struct {
	WorkflowGoalModel           string `json:"workflow_goal_model"`
	WorkflowGoalReasoningEffort string `json:"workflow_goal_reasoning_effort"`
}

type modelDefaultsResponse struct {
	ModelDefaults  app.ModelDefaults     `json:"model_defaults"`
	Effective      app.ModelDefaults     `json:"effective"`
	AgentExecutors []agentExecutorStatus `json:"agent_executors"`
	Display        map[string]string     `json:"display,omitempty"`
}

func (server *Server) handleSettingsModelDefaults(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		response, err := server.modelDefaultsResponse(r.Context())
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, response)
	case http.MethodPatch:
		var req modelDefaultsRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		defaults := app.ModelDefaults{
			WorkflowGoalModel:           req.WorkflowGoalModel,
			WorkflowGoalReasoningEffort: req.WorkflowGoalReasoningEffort,
		}
		if err := validateWorkflowGoalDefaults(defaults); err != nil {
			writeAppError(w, err)
			return
		}
		if _, err := server.service.SaveModelDefaults(r.Context(), defaults); err != nil {
			writeAppError(w, err)
			return
		}
		response, err := server.modelDefaultsResponse(r.Context())
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, response)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (server *Server) modelDefaultsResponse(ctx context.Context) (modelDefaultsResponse, error) {
	defaults, err := server.service.GetModelDefaults(ctx)
	if err != nil {
		return modelDefaultsResponse{}, err
	}
	effective := server.effectiveWorkflowGoalDefaults(defaults)
	return modelDefaultsResponse{
		ModelDefaults:  defaults,
		Effective:      effective,
		AgentExecutors: server.agentStatuses(),
		Display: map[string]string{
			"workflow_goal_source": workflowGoalDefaultSource(defaults, server.workflowGoalModel, server.workflowGoalReasoningEffort),
		},
	}, nil
}

func (server *Server) workflowGoalDefaults(ctx context.Context) (app.ModelDefaults, error) {
	defaults, err := server.service.GetModelDefaults(ctx)
	if err != nil {
		return app.ModelDefaults{}, err
	}
	return server.effectiveWorkflowGoalDefaults(defaults), nil
}

func (server *Server) effectiveWorkflowGoalDefaults(defaults app.ModelDefaults) app.ModelDefaults {
	model := strings.TrimSpace(defaults.WorkflowGoalModel)
	effort := strings.TrimSpace(defaults.WorkflowGoalReasoningEffort)
	if model == "" {
		model = strings.TrimSpace(server.workflowGoalModel)
	}
	if effort == "" {
		effort = strings.TrimSpace(server.workflowGoalReasoningEffort)
	}
	return app.ModelDefaults{
		WorkflowGoalModel:           model,
		WorkflowGoalReasoningEffort: effort,
	}
}

func validateWorkflowGoalDefaults(defaults app.ModelDefaults) error {
	model := strings.TrimSpace(defaults.WorkflowGoalModel)
	effort := strings.TrimSpace(defaults.WorkflowGoalReasoningEffort)
	if model == "" && effort == "" {
		return nil
	}
	if _, _, err := agentmodels.Resolve(model, effort); err != nil {
		return fmt.Errorf("%w: %v", app.ErrInvalidInput, err)
	}
	return nil
}

func workflowGoalDefaultSource(defaults app.ModelDefaults, configModel string, configEffort string) string {
	if strings.TrimSpace(defaults.WorkflowGoalModel) != "" || strings.TrimSpace(defaults.WorkflowGoalReasoningEffort) != "" {
		return "settings"
	}
	if strings.TrimSpace(configModel) != "" || strings.TrimSpace(configEffort) != "" {
		return "server_config"
	}
	return "provider_default"
}

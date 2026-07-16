package app

import (
	"context"
	"strings"
)

type ModelDefaults struct {
	WorkflowGoalModel           string `json:"workflow_goal_model"`
	WorkflowGoalReasoningEffort string `json:"workflow_goal_reasoning_effort"`
}

type ModelDefaultsStore interface {
	GetModelDefaults(context.Context) (ModelDefaults, error)
	SaveModelDefaults(context.Context, ModelDefaults) error
}

func (s *Service) GetModelDefaults(ctx context.Context) (ModelDefaults, error) {
	store, ok := s.store.(ModelDefaultsStore)
	if !ok {
		return ModelDefaults{}, nil
	}
	return store.GetModelDefaults(ctx)
}

func (s *Service) SaveModelDefaults(ctx context.Context, defaults ModelDefaults) (ModelDefaults, error) {
	normalized := ModelDefaults{
		WorkflowGoalModel:           strings.TrimSpace(defaults.WorkflowGoalModel),
		WorkflowGoalReasoningEffort: strings.ToLower(strings.TrimSpace(defaults.WorkflowGoalReasoningEffort)),
	}
	store, ok := s.store.(ModelDefaultsStore)
	if !ok {
		return ModelDefaults{}, ErrInvalidInput
	}
	if err := store.SaveModelDefaults(ctx, normalized); err != nil {
		return ModelDefaults{}, err
	}
	return normalized, nil
}

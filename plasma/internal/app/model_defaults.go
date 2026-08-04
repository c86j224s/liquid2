package app

import (
	"context"
	"strings"
)

// ModelDefaults는 설정 화면에서 저장하는 agent 모델 기본값이다.
//
// 현재는 workflow directing model 기본값만 포함하며 report별 선택값과는 별개다.
type ModelDefaults struct {
	WorkflowGoalModel           string `json:"workflow_goal_model"`
	WorkflowGoalReasoningEffort string `json:"workflow_goal_reasoning_effort"`
}

// ModelDefaultsStore는 모델 기본값을 읽고 저장하는 persistence port다.
type ModelDefaultsStore interface {
	GetModelDefaults(context.Context) (ModelDefaults, error)
	SaveModelDefaults(context.Context, ModelDefaults) error
}

// GetModelDefaults는 저장소가 지원할 때 모델 기본값을 반환한다.
//
// store가 아직 이 port를 구현하지 않으면 zero value를 반환해 설정 화면이 안전하게
// 비어 있는 상태로 동작하게 한다.
func (s *Service) GetModelDefaults(ctx context.Context) (ModelDefaults, error) {
	store, ok := s.store.(ModelDefaultsStore)
	if !ok {
		return ModelDefaults{}, nil
	}
	return store.GetModelDefaults(ctx)
}

// SaveModelDefaults는 모델 기본값을 정규화해 저장한다.
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

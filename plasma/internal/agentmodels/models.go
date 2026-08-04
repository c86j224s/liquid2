// Package agentmodels는 Plasma에서 선택 가능한 Codex 모델과 reasoning effort의
// 제품 기본값을 정의한다.
package agentmodels

import (
	"fmt"
	"strings"
)

const (
	DefaultModel           = "gpt-5.5"
	DefaultReasoningEffort = "medium"
)

// Model은 UI와 API에 노출되는 선택 가능한 Codex 모델의 안정적인 메타데이터다.
//
// ReasoningEfforts는 해당 모델에서 허용하는 값의 닫힌 목록이고,
// DefaultReasoningEffort는 사용자가 별도 선택을 하지 않았을 때 적용되는 제품
// 기본값이다.
type Model struct {
	Name                   string   `json:"name"`
	Label                  string   `json:"label"`
	ReasoningEfforts       []string `json:"reasoning_efforts"`
	DefaultReasoningEffort string   `json:"default_reasoning_effort"`
}

var catalog = []Model{
	{Name: "gpt-5.6-sol", Label: "GPT-5.6 Sol", ReasoningEfforts: []string{"low", "medium", "high", "xhigh", "max", "ultra"}, DefaultReasoningEffort: "medium"},
	{Name: "gpt-5.6-terra", Label: "GPT-5.6 Terra", ReasoningEfforts: []string{"low", "medium", "high", "xhigh", "max", "ultra"}, DefaultReasoningEffort: "medium"},
	{Name: "gpt-5.6-luna", Label: "GPT-5.6 Luna", ReasoningEfforts: []string{"low", "medium", "high", "xhigh", "max"}, DefaultReasoningEffort: "medium"},
	{Name: "gpt-5.5", Label: "GPT-5.5", ReasoningEfforts: []string{"low", "medium", "high", "xhigh"}, DefaultReasoningEffort: "medium"},
	{Name: "gpt-5.4", Label: "GPT-5.4", ReasoningEfforts: []string{"low", "medium", "high", "xhigh"}, DefaultReasoningEffort: "medium"},
	{Name: "gpt-5.4-mini", Label: "GPT-5.4 mini", ReasoningEfforts: []string{"low", "medium", "high", "xhigh"}, DefaultReasoningEffort: "medium"},
	{Name: "gpt-5.3-codex-spark", Label: "GPT-5.3 Codex Spark", ReasoningEfforts: []string{"low", "medium", "high", "xhigh"}, DefaultReasoningEffort: "medium"},
}

var genericReasoningEfforts = []string{"low", "medium", "high", "xhigh"}

// Catalog는 transport 계층이 그대로 내보낼 수 있는 모델 catalog 사본을 반환한다.
//
// 호출자가 반환값의 slice를 수정해도 package-level catalog가 바뀌지 않아야 한다.
func Catalog() []Model {
	result := make([]Model, len(catalog))
	for i, model := range catalog {
		result[i] = clone(model)
	}
	return result
}

// Default는 현재 제품 기본 Codex 모델의 metadata 사본을 반환한다.
func Default() Model {
	model, _ := lookup(DefaultModel)
	return clone(model)
}

// Resolve는 모델/effort 입력에 제품 기본값을 적용하고 알려진 모델 capability를
// 검증한다.
//
// 아직 catalog에 없는 모델명은 향후 Codex 릴리스를 막지 않기 위해 generic effort
// 목록으로만 검증한다. 이 함수는 모델을 실행하지 않으며, 실행 provider 선택도
// 하지 않는다.
func Resolve(model, effort string) (string, string, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		model = DefaultModel
	}
	effort = strings.ToLower(strings.TrimSpace(effort))
	if effort == "" {
		effort = DefaultReasoningEffort
		if known, ok := lookup(model); ok {
			effort = known.DefaultReasoningEffort
		}
	}
	allowed := genericReasoningEfforts
	if known, ok := lookup(model); ok {
		allowed = known.ReasoningEfforts
	}
	if !contains(allowed, effort) {
		return "", "", fmt.Errorf("unsupported reasoning effort %q for model %q", effort, model)
	}
	return model, effort, nil
}

// ResolveForSession은 기존 provider session을 이어갈 때 기록되지 않은 모델 설정을
// 그대로 비워 둔다.
//
// 새 session을 시작할 때만 Resolve의 기본값 적용 계약을 따른다. 이 불변 조건이
// 깨지면 과거 session resume이 의도치 않게 다른 모델로 해석될 수 있다.
func ResolveForSession(model, effort, previousSessionID string) (string, string, error) {
	if strings.TrimSpace(previousSessionID) != "" && strings.TrimSpace(model) == "" && strings.TrimSpace(effort) == "" {
		return "", "", nil
	}
	return Resolve(model, effort)
}

func lookup(name string) (Model, bool) {
	for _, model := range catalog {
		if model.Name == name {
			return model, true
		}
	}
	return Model{}, false
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func clone(model Model) Model {
	model.ReasoningEfforts = append([]string(nil), model.ReasoningEfforts...)
	return model
}

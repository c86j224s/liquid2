package reporting

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentmodels"
)

const (
	// AgentSelectionSourceExplicitRequest는 보고서 생성 요청이 모델/추론 강도를 직접 지정한 경우다.
	AgentSelectionSourceExplicitRequest = "explicit_request"
	// AgentSelectionSourceMissionSession은 미션의 최근 에이전트 세션 설정을 따른 경우다.
	AgentSelectionSourceMissionSession = "mission_session"
	// AgentSelectionSourceProviderDefault는 provider 기본 설정으로 fallback한 경우다.
	AgentSelectionSourceProviderDefault = "provider_default"
)

// ModelSelectionInput은 보고서 생성 시 모델 선택 우선순위에 필요한 입력이다.
// 요청값, 미션 세션값, provider 기본값을 모두 담되, executor가 reasoning effort를
// 지원하지 않는 경우는 별도로 표시한다.
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

// ModelSelection은 보고서 생성에 실제로 사용할 모델 설정과 그 출처다.
// Source는 UI와 원장에서 사용자가 어떤 설정이 적용됐는지 확인하기 위한 값이다.
type ModelSelection struct {
	Model           string
	ReasoningEffort string
	Source          string
}

// ResolveModelSelection은 요청값, 미션 세션값, provider 기본값 순서로 보고서
// 모델을 선택한다. Codex executor는 agentmodels.Resolve를 통과해 모델/effort
// 조합을 제품이 허용하는 값으로 보정한다.
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

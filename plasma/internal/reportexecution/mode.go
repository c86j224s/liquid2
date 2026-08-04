package reportexecution

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func NormalizeMode(mode string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(mode))
	if normalized == "" {
		return DefaultMode, nil
	}
	switch normalized {
	case "planned", "standard", "default":
		return ModePlanned, nil
	case "quick", "fast", "one-take", "one_take":
		return ModeOneTake, nil
	case "long", "long-form", "long_form":
		return ModeLongForm, nil
	default:
		return "", fmt.Errorf("%w: unsupported report mode", producterror.ErrInvalidInput)
	}
}

// NormalizeSessionPolicy는 보고서 생성 파이프라인 입력을 표준 형태로 정규화하고 허용되지 않는 값은 안정 오류로 거부한다.
func NormalizeSessionPolicy(policy string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(policy))
	if normalized == "" {
		return DefaultSessionPolicy, nil
	}
	switch normalized {
	case "same", "same-session", "same_session", "default":
		return SessionPolicySameSession, nil
	case "isolated-fork", "isolated_fork", "fork":
		return SessionPolicyIsolatedFork, nil
	default:
		return "", fmt.Errorf("%w: unsupported report session policy", producterror.ErrInvalidInput)
	}
}

// SelectSessionPolicy는 요청 값과 기본값에서 report session 정책을 결정한다.
func SelectSessionPolicy(input SessionPolicySelectionInput) (string, string, error) {
	if strings.TrimSpace(input.RequestedPolicy) != "" {
		policy, err := NormalizeSessionPolicy(input.RequestedPolicy)
		if err != nil {
			return "", "", err
		}
		if err := ValidateSessionPolicy(policy, input.ReportMode, input.CanForkSession, input.HasPreReportResearchSession, input.ForkReady); err != nil {
			return "", "", err
		}
		if policy == SessionPolicyIsolatedFork {
			return policy, SessionPolicySelectionExplicitIsolatedFork, nil
		}
		return policy, SessionPolicySelectionExplicitSameSession, nil
	}
	mode, err := NormalizeMode(input.ReportMode)
	if err != nil {
		return "", "", err
	}
	if mode == ModeOneTake {
		return SessionPolicySameSession, SessionPolicySelectionAutoSameSessionOneTake, nil
	}
	if !input.CanForkSession {
		return SessionPolicySameSession, SessionPolicySelectionAutoSameSessionNoForker, nil
	}
	if !input.HasPreReportResearchSession {
		return SessionPolicySameSession, SessionPolicySelectionAutoSameSessionNoSession, nil
	}
	if !input.ForkReady {
		return SessionPolicySameSession, SessionPolicySelectionAutoSameSessionForkFailed, nil
	}
	return SessionPolicyIsolatedFork, SessionPolicySelectionAutoIsolatedFork, nil
}

// ValidateSessionPolicy는 보고서 생성 파이프라인 계약을 검사한다. 제품 상태를 변경하지 않는 순수 검증 경계다.
func ValidateSessionPolicy(policy string, reportMode string, canForkSession bool, hasPreReportResearchSession bool, forkReady bool) error {
	policy, err := NormalizeSessionPolicy(policy)
	if err != nil {
		return err
	}
	if policy == SessionPolicySameSession {
		return nil
	}
	mode, err := NormalizeMode(reportMode)
	if err != nil {
		return err
	}
	if mode == ModeOneTake {
		return fmt.Errorf("%w: report session policy %q is not supported for one-take reports", producterror.ErrInvalidInput, policy)
	}
	if !canForkSession {
		return fmt.Errorf("%w: report session policy %q is unavailable because this executor cannot fork provider sessions", producterror.ErrInvalidInput, policy)
	}
	if !hasPreReportResearchSession {
		return fmt.Errorf("%w: report session policy %q requires a pre-report research session", producterror.ErrInvalidInput, policy)
	}
	if !forkReady {
		return fmt.Errorf("%w: report session policy %q is unavailable because the provider session cannot be prepared for fork", producterror.ErrInvalidInput, policy)
	}
	return nil
}

// ModeLabel는 report mode를 사용자 표시용 label로 변환한다.
func ModeLabel(mode string) string {
	switch mode {
	case ModeLongForm:
		return labelLongForm
	case ModeOneTake:
		return labelOneTake
	default:
		return labelPlanned
	}
}

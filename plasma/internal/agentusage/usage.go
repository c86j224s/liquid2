package agentusage

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

// SchemaVersion은 agent usage payload의 장부 저장 형식 버전이다.
const SchemaVersion = 2

const (
	// UsageScopeCall은 provider usage가 이번 agent 호출에서 발생한 양임을 뜻한다.
	UsageScopeCall = "call"
	// UsageScopeSessionCumulative은 provider usage가 같은 provider session의 누적
	// snapshot임을 뜻한다. 호출별 사용량은 이전 snapshot과의 차이로 계산해야 한다.
	UsageScopeSessionCumulative = "session_cumulative"
)

// PromptMetrics는 prompt 본문을 저장하지 않고 크기와 hash만 남기는 계측값이다.
//
// SHA256은 동일 prompt 재현 여부를 비교하기 위한 값이며, prompt 내용을 역으로
// 복원하는 용도로 쓰면 안 된다.
type PromptMetrics struct {
	Bytes           int    `json:"bytes"`
	Chars           int    `json:"chars"`
	EstimatedTokens int    `json:"estimated_tokens"`
	SHA256          string `json:"sha256"`
}

// SessionMetrics는 provider session 재사용과 compaction 시도 여부를 추적한다.
//
// 값이 비어 있으면 해당 provider가 session ID를 제공하지 않았거나 이번 실행에서
// 세션을 새로 기록하지 않았다는 뜻이다.
type SessionMetrics struct {
	PreviousAgentSessionID string `json:"previous_agent_session_id,omitempty"`
	AgentSessionID         string `json:"agent_session_id,omitempty"`
	Resumed                bool   `json:"resumed"`
	CompactionAttempted    bool   `json:"compaction_attempted,omitempty"`
}

// ProviderUsage는 provider가 노출한 token 사용량을 Plasma 공통 필드로 축소한
// 값이다.
//
// Scope는 값이 한 호출의 양인지 provider session 누적 snapshot인지 구분한다.
// provider마다 일부 필드는 비어 있을 수 있으며 Normalize가 누락 가능한 합계만
// 보강한다. 이 값은 과금의 최종 원장이 아니라 report/workflow 관측치다.
type ProviderUsage struct {
	Scope                 string `json:"scope,omitempty"`
	InputTokens           int    `json:"input_tokens,omitempty"`
	CachedInputTokens     int    `json:"cached_input_tokens,omitempty"`
	UncachedInputTokens   int    `json:"uncached_input_tokens,omitempty"`
	OutputTokens          int    `json:"output_tokens,omitempty"`
	ReasoningOutputTokens int    `json:"reasoning_output_tokens,omitempty"`
	TotalTokens           int    `json:"total_tokens,omitempty"`
}

// ContextWindowMetrics는 provider가 보고한 현재 컨텍스트 점유량이다.
//
// UsedTokens와 WindowTokens가 모두 양수일 때만 유효하다. Plasma는 비율을
// 저장하지 않고 원시 숫자를 보관해 임계값 정책과 계측 해석을 분리한다.
type ContextWindowMetrics struct {
	UsedTokens   int    `json:"used_tokens"`
	WindowTokens int    `json:"window_tokens"`
	Source       string `json:"source,omitempty"`
}

// AgentUsage는 한 번의 agent 호출에서 Plasma가 장부에 남기는 usage envelope이다.
//
// Prompt는 원문 대신 metrics만 보관하고, provider usage가 없을 때는
// UsageUnavailableReason으로 원인을 남긴다.
type AgentUsage struct {
	SchemaVersion          int                   `json:"schema_version"`
	Surface                string                `json:"surface,omitempty"`
	Provider               string                `json:"provider,omitempty"`
	Executor               string                `json:"executor,omitempty"`
	Model                  string                `json:"model,omitempty"`
	ReasoningEffort        string                `json:"reasoning_effort,omitempty"`
	Prompt                 PromptMetrics         `json:"prompt"`
	Session                SessionMetrics        `json:"session"`
	ProviderUsage          *ProviderUsage        `json:"provider_usage,omitempty"`
	ContextWindow          *ContextWindowMetrics `json:"context_window,omitempty"`
	DurationMS             int64                 `json:"duration_ms,omitempty"`
	UsageSource            string                `json:"usage_source,omitempty"`
	UsageUnavailable       bool                  `json:"usage_unavailable"`
	UsageUnavailableReason string                `json:"usage_unavailable_reason,omitempty"`
}

// WithContextWindow는 provider가 보고한 유효한 현재 컨텍스트 점유량을 붙인다.
// 유효하지 않은 값은 추정하거나 보정하지 않고 무시한다.
func (usage AgentUsage) WithContextWindow(metrics ContextWindowMetrics) AgentUsage {
	if !metrics.Valid() {
		return usage
	}
	metrics.Source = strings.TrimSpace(metrics.Source)
	usage.ContextWindow = &metrics
	return usage
}

// New는 provider 실행 전후에 공통으로 채울 수 있는 usage envelope을 만든다.
//
// 실행 결과에서만 알 수 있는 session, duration, provider token 값은 With*
// 메서드로 뒤에 붙인다.
func New(provider string, executor string, model string, reasoningEffort string, prompt string) AgentUsage {
	return AgentUsage{
		SchemaVersion:   SchemaVersion,
		Provider:        strings.TrimSpace(provider),
		Executor:        strings.TrimSpace(executor),
		Model:           strings.TrimSpace(model),
		ReasoningEffort: strings.TrimSpace(reasoningEffort),
		Prompt:          Prompt(prompt),
	}
}

// Prompt는 prompt 원문을 저장하지 않는 계측값을 계산한다.
func Prompt(prompt string) PromptMetrics {
	sum := sha256.Sum256([]byte(prompt))
	return PromptMetrics{
		Bytes:           len([]byte(prompt)),
		Chars:           utf8.RuneCountInString(prompt),
		EstimatedTokens: estimateTokens(prompt),
		SHA256:          hex.EncodeToString(sum[:]),
	}
}

// WithProviderUsage는 provider가 낸 token 사용량을 붙이고 unavailable 상태를 해제한다.
func (usage AgentUsage) WithProviderUsage(providerUsage ProviderUsage, source string) AgentUsage {
	providerUsage.Normalize()
	usage.ProviderUsage = &providerUsage
	usage.UsageSource = strings.TrimSpace(source)
	usage.UsageUnavailable = false
	usage.UsageUnavailableReason = ""
	return usage
}

// WithUnavailable은 provider usage를 얻지 못한 실행을 명시적으로 표시한다.
//
// 이미 provider usage가 채워진 값은 성공 계측을 우선하기 위해 그대로 반환한다.
func (usage AgentUsage) WithUnavailable(reason string) AgentUsage {
	if usage.ProviderUsage != nil {
		return usage
	}
	usage.UsageUnavailable = true
	usage.UsageUnavailableReason = strings.TrimSpace(reason)
	return usage
}

// WithSurface는 이 usage가 발생한 제품 표면을 붙인다.
func (usage AgentUsage) WithSurface(surface string) AgentUsage {
	usage.Surface = strings.TrimSpace(surface)
	return usage
}

// WithDuration은 agent 실행의 wall-clock duration을 밀리초 단위로 붙인다.
func (usage AgentUsage) WithDuration(durationMS int64) AgentUsage {
	usage.DurationMS = durationMS
	return usage
}

// WithSession은 provider session lineage를 붙인다.
func (usage AgentUsage) WithSession(previousSessionID string, sessionID string, resumed bool, compactionAttempted bool) AgentUsage {
	usage.Session = SessionMetrics{
		PreviousAgentSessionID: strings.TrimSpace(previousSessionID),
		AgentSessionID:         strings.TrimSpace(sessionID),
		Resumed:                resumed,
		CompactionAttempted:    compactionAttempted,
	}
	return usage
}

// Empty는 장부에 남길 usage envelope이 실제로 구성되지 않았는지 판정한다.
func (usage AgentUsage) Empty() bool {
	return usage.SchemaVersion == 0
}

// ForEvent는 agent 실행 직후 장부 event payload에 넣을 최종 usage 값을 만든다.
//
// provider usage가 비어 있으면 누락 이유를 안정적인 문구로 채워, 호출자가 별도
// 방어 로직 없이 관측 실패를 표현할 수 있게 한다.
func (usage AgentUsage) ForEvent(surface string, durationMS int64, previousSessionID string, sessionID string, resumed bool, compactionAttempted bool) (AgentUsage, bool) {
	if usage.Empty() {
		return AgentUsage{}, false
	}
	usage = usage.WithSurface(surface)
	usage = usage.WithDuration(durationMS)
	usage = usage.WithSession(previousSessionID, sessionID, resumed, compactionAttempted)
	if usage.ProviderUsage == nil && !usage.UsageUnavailable {
		usage = usage.WithUnavailable("provider usage was not emitted")
	}
	return usage, true
}

// Normalize는 provider가 일부 token breakdown만 준 경우 파생 가능한 합계를 채운다.
//
// 이 함수는 없는 값을 추정하지 않고 산술적으로 확정 가능한 값만 보강한다.
func (usage *ProviderUsage) Normalize() {
	if usage == nil {
		return
	}
	usage.Scope = strings.TrimSpace(usage.Scope)
	if usage.Scope == "" {
		usage.Scope = UsageScopeCall
	}
	if usage.InputTokens > 0 && usage.CachedInputTokens > 0 && usage.UncachedInputTokens == 0 {
		uncached := usage.InputTokens - usage.CachedInputTokens
		if uncached > 0 {
			usage.UncachedInputTokens = uncached
		}
	}
	if usage.TotalTokens == 0 {
		total := usage.InputTokens + usage.OutputTokens
		if total > 0 {
			usage.TotalTokens = total
		}
	}
}

// Valid는 점유율을 계산할 수 있는 완전한 provider 관측치인지 판정한다.
func (metrics ContextWindowMetrics) Valid() bool {
	return metrics.UsedTokens > 0 && metrics.WindowTokens > 0
}

// AtOrAbovePercent는 부동소수점 반올림 없이 정수 백분율 임계값을 비교한다.
func (metrics ContextWindowMetrics) AtOrAbovePercent(percent int) bool {
	if !metrics.Valid() || percent <= 0 || percent > 100 {
		return false
	}
	return int64(metrics.UsedTokens)*100 >= int64(metrics.WindowTokens)*int64(percent)
}

func estimateTokens(prompt string) int {
	chars := utf8.RuneCountInString(prompt)
	if chars == 0 {
		return 0
	}
	return (chars + 3) / 4
}

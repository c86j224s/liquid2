package agentusage

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

const SchemaVersion = 1

type PromptMetrics struct {
	Bytes           int    `json:"bytes"`
	Chars           int    `json:"chars"`
	EstimatedTokens int    `json:"estimated_tokens"`
	SHA256          string `json:"sha256"`
}

type SessionMetrics struct {
	PreviousAgentSessionID string `json:"previous_agent_session_id,omitempty"`
	AgentSessionID         string `json:"agent_session_id,omitempty"`
	Resumed                bool   `json:"resumed"`
	CompactionAttempted    bool   `json:"compaction_attempted,omitempty"`
}

type ProviderUsage struct {
	InputTokens           int `json:"input_tokens,omitempty"`
	CachedInputTokens     int `json:"cached_input_tokens,omitempty"`
	UncachedInputTokens   int `json:"uncached_input_tokens,omitempty"`
	OutputTokens          int `json:"output_tokens,omitempty"`
	ReasoningOutputTokens int `json:"reasoning_output_tokens,omitempty"`
	TotalTokens           int `json:"total_tokens,omitempty"`
}

type AgentUsage struct {
	SchemaVersion          int            `json:"schema_version"`
	Surface                string         `json:"surface,omitempty"`
	Provider               string         `json:"provider,omitempty"`
	Executor               string         `json:"executor,omitempty"`
	Model                  string         `json:"model,omitempty"`
	ReasoningEffort        string         `json:"reasoning_effort,omitempty"`
	Prompt                 PromptMetrics  `json:"prompt"`
	Session                SessionMetrics `json:"session"`
	ProviderUsage          *ProviderUsage `json:"provider_usage,omitempty"`
	DurationMS             int64          `json:"duration_ms,omitempty"`
	UsageSource            string         `json:"usage_source,omitempty"`
	UsageUnavailable       bool           `json:"usage_unavailable"`
	UsageUnavailableReason string         `json:"usage_unavailable_reason,omitempty"`
}

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

func Prompt(prompt string) PromptMetrics {
	sum := sha256.Sum256([]byte(prompt))
	return PromptMetrics{
		Bytes:           len([]byte(prompt)),
		Chars:           utf8.RuneCountInString(prompt),
		EstimatedTokens: estimateTokens(prompt),
		SHA256:          hex.EncodeToString(sum[:]),
	}
}

func (usage AgentUsage) WithProviderUsage(providerUsage ProviderUsage, source string) AgentUsage {
	providerUsage.Normalize()
	usage.ProviderUsage = &providerUsage
	usage.UsageSource = strings.TrimSpace(source)
	usage.UsageUnavailable = false
	usage.UsageUnavailableReason = ""
	return usage
}

func (usage AgentUsage) WithUnavailable(reason string) AgentUsage {
	if usage.ProviderUsage != nil {
		return usage
	}
	usage.UsageUnavailable = true
	usage.UsageUnavailableReason = strings.TrimSpace(reason)
	return usage
}

func (usage AgentUsage) WithSurface(surface string) AgentUsage {
	usage.Surface = strings.TrimSpace(surface)
	return usage
}

func (usage AgentUsage) WithDuration(durationMS int64) AgentUsage {
	usage.DurationMS = durationMS
	return usage
}

func (usage AgentUsage) WithSession(previousSessionID string, sessionID string, resumed bool, compactionAttempted bool) AgentUsage {
	usage.Session = SessionMetrics{
		PreviousAgentSessionID: strings.TrimSpace(previousSessionID),
		AgentSessionID:         strings.TrimSpace(sessionID),
		Resumed:                resumed,
		CompactionAttempted:    compactionAttempted,
	}
	return usage
}

func (usage AgentUsage) Empty() bool {
	return usage.SchemaVersion == 0
}

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

func (usage *ProviderUsage) Normalize() {
	if usage == nil {
		return
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

func estimateTokens(prompt string) int {
	chars := utf8.RuneCountInString(prompt)
	if chars == 0 {
		return 0
	}
	return (chars + 3) / 4
}

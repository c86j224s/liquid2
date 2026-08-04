package agentexec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

type claudeEvent struct {
	Type         string                      `json:"type"`
	Subtype      string                      `json:"subtype"`
	SessionID    string                      `json:"session_id"`
	Result       *string                     `json:"result"`
	Message      claudeMessage               `json:"message"`
	Usage        claudeUsage                 `json:"usage"`
	ModelUsage   map[string]claudeModelUsage `json:"modelUsage"`
	TotalCostUSD float64                     `json:"total_cost_usd"`
	Errors       []string                    `json:"errors"`
}

type claudeMessage struct {
	Content []claudeContent `json:"content"`
	Usage   claudeUsage     `json:"usage"`
}

type claudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type claudeUsage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokens             int `json:"output_tokens"`
}

type claudeModelUsage struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
	CostUSD                  float64 `json:"costUSD"`
}

func parseClaudeJSONOutput(raw []byte, usage agentusage.AgentUsage) (AgentResult, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return AgentResult{Usage: usage.WithUnavailable("claude emitted no JSON output")}, fmt.Errorf("agent emitted no JSON output")
	}
	events, err := parseClaudeEvents(raw)
	if err != nil {
		return AgentResult{Usage: usage.WithUnavailable("claude JSON output could not be parsed")}, err
	}
	result := AgentResult{Usage: usage}
	var lastText string
	var providerUsage agentusage.ProviderUsage
	var sawProviderUsage bool
	for _, event := range events {
		if strings.TrimSpace(event.SessionID) != "" {
			result.SessionID = strings.TrimSpace(event.SessionID)
		}
		if event.Result != nil && strings.TrimSpace(*event.Result) != "" {
			lastText = strings.TrimSpace(*event.Result)
		}
		for _, content := range event.Message.Content {
			if content.Type == "text" && strings.TrimSpace(content.Text) != "" {
				lastText = strings.TrimSpace(content.Text)
			}
		}
		if usage, ok := claudeProviderUsage(event); ok {
			providerUsage = usage
			sawProviderUsage = true
		}
	}
	result.Text = lastText
	result.Resumed = strings.TrimSpace(usage.Session.PreviousAgentSessionID) != ""
	if sawProviderUsage {
		result.Usage = result.Usage.WithProviderUsage(providerUsage, "claude_json")
	} else {
		result.Usage = result.Usage.WithUnavailable("claude JSON did not include provider usage")
	}
	result.Usage = result.Usage.WithSession(usage.Session.PreviousAgentSessionID, result.SessionID, result.Resumed, usage.Session.CompactionAttempted)
	return result, nil
}

func parseClaudeEvents(raw []byte) ([]claudeEvent, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, fmt.Errorf("agent emitted no JSON output")
	}
	if raw[0] == '[' {
		var events []claudeEvent
		if err := json.Unmarshal(raw, &events); err != nil {
			return nil, err
		}
		return events, nil
	}
	var event claudeEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return nil, err
	}
	return []claudeEvent{event}, nil
}

func claudeProviderUsage(event claudeEvent) (agentusage.ProviderUsage, bool) {
	for _, modelUsage := range event.ModelUsage {
		return agentusage.ProviderUsage{
			InputTokens:         modelUsage.InputTokens + modelUsage.CacheReadInputTokens + modelUsage.CacheCreationInputTokens,
			CachedInputTokens:   modelUsage.CacheReadInputTokens,
			UncachedInputTokens: modelUsage.InputTokens + modelUsage.CacheCreationInputTokens,
			OutputTokens:        modelUsage.OutputTokens,
		}, true
	}
	usage := event.Usage
	totalInput := usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
	if totalInput == 0 && usage.OutputTokens == 0 {
		return agentusage.ProviderUsage{}, false
	}
	return agentusage.ProviderUsage{
		InputTokens:         totalInput,
		CachedInputTokens:   usage.CacheReadInputTokens,
		UncachedInputTokens: usage.InputTokens + usage.CacheCreationInputTokens,
		OutputTokens:        usage.OutputTokens,
	}, true
}

func claudeModel(requestModel string, executorModel string) string {
	if model := strings.TrimSpace(requestModel); model != "" {
		return model
	}
	if model := strings.TrimSpace(executorModel); model != "" {
		return model
	}
	return "haiku"
}

func claudeUsageExecutorName(executorName string) string {
	executorName = strings.TrimSpace(executorName)
	if executorName == "" {
		return "claude"
	}
	return executorName
}

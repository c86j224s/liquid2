package agentusage

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
)

var codexStatusSessionPattern = regexp.MustCompile(`(?m)^session id:\s+([^\s]+)`)

// ParseCodexProviderUsage는 Codex JSONL 로그에서 가장 최근 turn.completed usage
// 누적 snapshot을 추출한다.
//
// 로그에 사람이 읽는 status line이나 다른 JSON event가 섞여 있어도 무시한다. 원시
// 로그는 장부에 저장하지 않고 ProviderUsage만 반환한다.
func ParseCodexProviderUsage(log string) (ProviderUsage, bool) {
	var latest ProviderUsage
	found := false
	reader := bufio.NewReader(strings.NewReader(log))
	for {
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}
		var event struct {
			Type  string        `json:"type"`
			Usage ProviderUsage `json:"usage"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Type != "turn.completed" {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}
		event.Usage.Scope = UsageScopeSessionCumulative
		event.Usage.Normalize()
		latest = event.Usage
		found = true
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return latest, found
}

// ParseCodexContextWindowMetrics는 Codex session JSONL에서 가장 최근의 현재
// 컨텍스트 점유량을 추출한다. 세션 본문과 도구 결과는 해석하거나 반환하지 않는다.
func ParseCodexContextWindowMetrics(log string) (ContextWindowMetrics, bool) {
	var latest ContextWindowMetrics
	found := false
	reader := bufio.NewReader(strings.NewReader(log))
	for {
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			break
		}
		line = strings.TrimSpace(line)
		if line != "" && strings.HasPrefix(line, "{") {
			var event struct {
				Type    string `json:"type"`
				Payload struct {
					Type string `json:"type"`
					Info struct {
						LastTokenUsage struct {
							TotalTokens int `json:"total_tokens"`
						} `json:"last_token_usage"`
						ModelContextWindow int `json:"model_context_window"`
					} `json:"info"`
				} `json:"payload"`
			}
			if json.Unmarshal([]byte(line), &event) == nil && event.Type == "event_msg" && event.Payload.Type == "token_count" {
				metrics := ContextWindowMetrics{
					UsedTokens:   event.Payload.Info.LastTokenUsage.TotalTokens,
					WindowTokens: event.Payload.Info.ModelContextWindow,
					Source:       "codex_session_token_count",
				}
				if metrics.Valid() {
					latest = metrics
					found = true
				}
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return latest, found
}

// ParseCodexSessionID는 Codex 로그에서 provider session/thread ID를 찾는다.
//
// status 출력의 session id와 JSON event의 thread.started를 모두 지원하지만, 찾지
// 못하면 빈 문자열을 반환해 caller가 sessionless 실행으로 처리하게 한다.
func ParseCodexSessionID(log string) string {
	if match := codexStatusSessionPattern.FindStringSubmatch(log); len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	reader := bufio.NewReader(strings.NewReader(log))
	for {
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}
		var event struct {
			Type     string `json:"type"`
			ThreadID string `json:"thread_id"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Type == "thread.started" && strings.TrimSpace(event.ThreadID) != "" {
			return strings.TrimSpace(event.ThreadID)
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return ""
}

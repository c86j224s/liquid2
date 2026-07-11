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
		event.Usage.Normalize()
		latest = event.Usage
		found = true
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return latest, found
}

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

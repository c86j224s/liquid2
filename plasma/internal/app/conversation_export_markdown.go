package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var conversationExportLocalPathRE = regexp.MustCompile(`(?:/Users|/home)/[^\s"'<>)]*`)

type conversationExportEntry struct {
	Role      string
	Text      string
	CreatedAt time.Time
}

type conversationEventPayload struct {
	Kind           string `json:"kind"`
	Text           string `json:"text"`
	WorkflowStepID string `json:"workflow_step_id"`
}

func buildConversationExportMarkdown(title string, events []LedgerEvent) ([]byte, int, error) {
	entries := visibleConversationEntries(events)
	if len(entries) == 0 {
		return nil, 0, fmt.Errorf("%w: conversation export requires visible conversation entries", ErrInvalidInput)
	}
	var body bytes.Buffer
	body.WriteString("# ")
	body.WriteString(title)
	body.WriteString("\n\n")
	body.WriteString("이 문서는 보고서로 재작성하지 않고, 미션 대화에서 사용자가 읽을 수 있는 요청과 응답만 순서대로 모은 export입니다.\n\n")
	body.WriteString("- 포함: 사용자 요청, 에이전트 응답, 워크플로우 단계 결과\n")
	body.WriteString("- 제외: 대기 상태, MCP trace, provider raw response, 내부 session id, 도구 호출 원문\n\n")
	for index, entry := range entries {
		body.WriteString("## ")
		body.WriteString(fmt.Sprintf("%d. %s", index+1, entry.Role))
		body.WriteString("\n\n")
		if !entry.CreatedAt.IsZero() {
			body.WriteString("_")
			body.WriteString(entry.CreatedAt.Format(time.RFC3339))
			body.WriteString("_\n\n")
		}
		body.WriteString(sanitizeConversationExportText(entry.Text))
		body.WriteString("\n\n")
	}
	return body.Bytes(), len(entries), nil
}

func visibleConversationEntries(events []LedgerEvent) []conversationExportEntry {
	entries := make([]conversationExportEntry, 0)
	for _, event := range events {
		var payload conversationEventPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		text := strings.TrimSpace(payload.Text)
		if text == "" {
			continue
		}
		switch event.EventType {
		case "turn.user":
			if strings.TrimSpace(payload.Kind) != "user_turn" {
				continue
			}
			entries = append(entries, conversationExportEntry{
				Role:      "사용자 요청",
				Text:      text,
				CreatedAt: event.CreatedAt,
			})
		case "turn.agent.response":
			if strings.TrimSpace(payload.Kind) != "agent_response" {
				continue
			}
			role := "에이전트 응답"
			if strings.TrimSpace(payload.WorkflowStepID) != "" {
				role = "워크플로우 단계 결과"
			}
			entries = append(entries, conversationExportEntry{
				Role:      role,
				Text:      text,
				CreatedAt: event.CreatedAt,
			})
		}
	}
	return entries
}

func conversationExportTitle(title string) string {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return "대화내역 export"
	}
	return trimmed
}

func conversationExportFilename(title string) string {
	name := strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '.':
			b.WriteRune('-')
		}
	}
	filename := strings.Trim(b.String(), "-_")
	if filename == "" {
		filename = "conversation-export"
	}
	if len(filename) > 80 {
		filename = filename[:80]
	}
	return filename + ".md"
}

func sanitizeConversationExportText(text string) string {
	text = strings.TrimSpace(text)
	text = conversationExportLocalPathRE.ReplaceAllString(text, "/path/to/...")
	return text
}

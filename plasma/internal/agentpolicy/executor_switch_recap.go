package agentpolicy

import (
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

const (
	switchRecapMaxTurns    = 12
	switchRecapMaxFieldLen = 600
)

// SwitchRecap builds a compact recap of a mission's completed turns for the
// prompt of an executor that has no provider-native memory of them. It
// returns "" when the mission has no completed turn from an executor other
// than targetExecutor, since a genuinely fresh mission needs no recap.
func SwitchRecap(events []ledger.Event, targetExecutor string) string {
	target, err := NormalizeExecutorName(targetExecutor)
	if err != nil {
		target = strings.TrimSpace(strings.ToLower(targetExecutor))
	}

	type turn struct {
		userText     string
		executor     string
		responseText string
		hasResponse  bool
	}
	turns := map[string]*turn{}
	order := make([]string, 0, len(events))
	sawOtherExecutor := false

	for _, event := range events {
		switch event.EventType {
		case "turn.user":
			var payload struct {
				Text string `json:"text"`
			}
			if json.Unmarshal(event.Payload, &payload) != nil {
				continue
			}
			turns[event.EventID] = &turn{userText: payload.Text}
			order = append(order, event.EventID)
		case "turn.agent.response":
			var payload struct {
				Kind          string `json:"kind"`
				Text          string `json:"text"`
				UserEventID   string `json:"user_event_id"`
				AgentExecutor string `json:"agent_executor"`
			}
			if json.Unmarshal(event.Payload, &payload) != nil || payload.Kind != "agent_response" {
				continue
			}
			record, ok := turns[payload.UserEventID]
			if !ok {
				continue
			}
			record.responseText = payload.Text
			record.hasResponse = true
			record.executor = strings.TrimSpace(strings.ToLower(payload.AgentExecutor))
			if record.executor != "" && record.executor != target {
				sawOtherExecutor = true
			}
		}
	}

	if !sawOtherExecutor {
		return ""
	}

	start := 0
	if len(order) > switchRecapMaxTurns {
		start = len(order) - switchRecapMaxTurns
	}

	var builder strings.Builder
	for _, id := range order[start:] {
		record := turns[id]
		if record == nil || !record.hasResponse {
			continue
		}
		builder.WriteString("- user: ")
		builder.WriteString(truncateRecapText(record.userText))
		builder.WriteString("\n  ")
		builder.WriteString(record.executor)
		builder.WriteString(": ")
		builder.WriteString(truncateRecapText(record.responseText))
		builder.WriteString("\n")
	}
	return strings.TrimRight(builder.String(), "\n")
}

// PriorExecutor returns the most recent executor that completed a turn in
// this mission, when it differs from targetExecutor. It returns "" when the
// mission has no completed turn from a different executor, so callers can use
// it to decide whether a switch actually happened and who it switched from.
func PriorExecutor(events []ledger.Event, targetExecutor string) string {
	target, err := NormalizeExecutorName(targetExecutor)
	if err != nil {
		target = strings.TrimSpace(strings.ToLower(targetExecutor))
	}
	last := ""
	for _, event := range events {
		if event.EventType != "turn.agent.response" {
			continue
		}
		var payload struct {
			Kind          string `json:"kind"`
			AgentExecutor string `json:"agent_executor"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil || payload.Kind != "agent_response" {
			continue
		}
		if executor := strings.TrimSpace(strings.ToLower(payload.AgentExecutor)); executor != "" {
			last = executor
		}
	}
	if last == "" || last == target {
		return ""
	}
	return last
}

// ExecutorSwitchedEvent is the durable audit event type recorded when a
// mission's next operation uses a different executor than its most recently
// completed one.
const ExecutorSwitchedEvent = "agent.executor.switched"

// BuildExecutorSwitchedAppendRequest builds the audit event append request
// for an executor switch. from and to should already be normalized executor
// names.
func BuildExecutorSwitchedAppendRequest(eventID, missionID, from, to string, producer ledger.Producer) ledger.AppendRequest {
	payload, _ := json.Marshal(map[string]any{
		"from": from,
		"to":   to,
	})
	return ledger.AppendRequest{
		EventID:   eventID,
		MissionID: missionID,
		EventType: ExecutorSwitchedEvent,
		Producer:  producer,
		Payload:   payload,
	}
}

func truncateRecapText(text string) string {
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) <= switchRecapMaxFieldLen {
		return text
	}
	return string(runes[:switchRecapMaxFieldLen]) + "…"
}

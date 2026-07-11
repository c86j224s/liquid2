package mcptrace

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type ToolCalledAppendRequest struct {
	EventID        string
	MissionID      string
	ToolName       string
	AgentSessionID string
	StartedAt      time.Time
	FinishedAt     time.Time
	Success        bool
	Arguments      map[string]any
	Result         map[string]any
	IOMetrics      map[string]any
	Producer       app.Producer
}

func BuildToolCalledAppendRequest(req ToolCalledAppendRequest) app.AppendEventRequest {
	started := req.StartedAt
	finished := req.FinishedAt
	return app.AppendEventRequest{
		EventID:       strings.TrimSpace(req.EventID),
		MissionID:     strings.TrimSpace(req.MissionID),
		EventType:     "mcp.tool.called",
		Producer:      req.Producer,
		CorrelationID: strings.TrimSpace(req.AgentSessionID),
		Payload: mustJSON(map[string]any{
			"tool_name":        strings.TrimSpace(req.ToolName),
			"tool_session_id":  strings.TrimSpace(req.AgentSessionID),
			"agent_session_id": strings.TrimSpace(req.AgentSessionID),
			"mission_id":       strings.TrimSpace(req.MissionID),
			"started_at":       started.Format(time.RFC3339Nano),
			"finished_at":      finished.Format(time.RFC3339Nano),
			"duration_ms":      finished.Sub(started).Milliseconds(),
			"success":          req.Success,
			"arguments":        req.Arguments,
			"result":           req.Result,
			"io_metrics":       req.IOMetrics,
		}),
	}
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

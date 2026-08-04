package mcptrace

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// ToolCalledAppendRequest는 MCP tool 호출 한 번을 장부 trace event로 남기기 위한
// 입력이다.
//
// Arguments와 Result는 이미 redaction이 끝난 작은 map이어야 한다. 원문 source,
// credential, provider raw response를 넣으면 안 된다.
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

// BuildToolCalledAppendRequest는 mcp.tool.called append request를 만든다.
//
// 이 builder는 추적 payload만 만들고 tool 성공 여부나 제품 상태 transition을
// 판단하지 않는다.
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

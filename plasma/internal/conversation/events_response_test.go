package conversation

import (
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestBuildTurnAgentResponseAppendRequestPreservesPayloadOptions(t *testing.T) {
	usage := agentusage.New("openai", "codex", "gpt-5.5", "medium", "prompt")
	req := BuildTurnAgentResponseAppendRequest(TurnAgentResponseEventRequest{
		EventID:               "evt_response",
		MissionID:             "mis_1",
		Kind:                  "agent_response",
		AgentExecutor:         "codex",
		MCPMode:               "auto",
		IncludeMCPMode:        true,
		Text:                  "done",
		AgentSessionID:        "ses_next",
		IncludeAgentSessionID: true,
		Resumed:               true,
		IncludeResumed:        true,
		DurationMS:            12,
		IncludeDuration:       true,
		UserEventID:           "evt_user",
		Extra: map[string]any{
			"previous_agent_session_id": "ses_prev",
			"strategy_id":               "",
		},
		Usage:                  usage,
		UsageSurface:           "turn",
		UsagePreviousSessionID: "ses_prev",
		Producer:               app.Producer{Type: "agent", ID: "codex"},
	})
	payload := appPayload(t, req)
	if payload["kind"] != "agent_response" ||
		payload["mcp_mode"] != "auto" ||
		payload["agent_session_id"] != "ses_next" ||
		payload["resumed"] != true ||
		payload["duration_ms"] != float64(12) ||
		payload["previous_agent_session_id"] != "ses_prev" ||
		payload["strategy_id"] != "" {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
	if _, ok := payload["agent_usage"]; !ok {
		t.Fatalf("expected agent_usage payload: %#v", payload)
	}
}

func TestBuildTurnAgentResponseAppendRequestCanOmitMCPModeAndResumed(t *testing.T) {
	req := BuildTurnAgentResponseAppendRequest(TurnAgentResponseEventRequest{
		EventID:               "evt_error",
		MissionID:             "mis_1",
		Kind:                  "agent_error",
		AgentExecutor:         "codex",
		Text:                  "Agent failed: boom",
		Error:                 "",
		IncludeError:          true,
		LogExcerpt:            "",
		IncludeLogExcerpt:     true,
		AgentSessionID:        "",
		IncludeAgentSessionID: false,
		DurationMS:            34,
		IncludeDuration:       true,
		UserEventID:           "evt_user",
		Producer:              app.Producer{Type: "agent", ID: "codex"},
	})
	payload := appPayload(t, req)
	if _, ok := payload["mcp_mode"]; ok {
		t.Fatalf("mcp_mode must be omitted when not requested: %#v", payload)
	}
	if _, ok := payload["resumed"]; ok {
		t.Fatalf("resumed must be omitted when not requested: %#v", payload)
	}
	if _, ok := payload["agent_session_id"]; ok {
		t.Fatalf("agent_session_id must be omitted when not requested: %#v", payload)
	}
	if errorText, ok := payload["error"]; !ok || errorText != "" {
		t.Fatalf("empty error key must be preserved: %#v", payload)
	}
	if logExcerpt, ok := payload["log_excerpt"]; !ok || logExcerpt != "" {
		t.Fatalf("empty log_excerpt key must be preserved: %#v", payload)
	}
}

func TestBuildTurnAgentResponseAppendRequestPreservesCanceledPayloadShape(t *testing.T) {
	req := BuildTurnAgentResponseAppendRequest(TurnAgentResponseEventRequest{
		EventID:       "evt_canceled",
		MissionID:     "mis_1",
		Kind:          "agent_canceled",
		AgentExecutor: "codex",
		Text:          "취소했습니다.",
		UserEventID:   "evt_user",
		Extra: map[string]any{
			"canceled_at":      "2026-07-09T01:02:03Z",
			"workflow_run_id":  "wfr_1",
			"workflow_step_id": "wfs_1",
		},
		Producer: app.Producer{Type: "agent", ID: "codex"},
	})
	payload := appPayload(t, req)
	if payload["kind"] != "agent_canceled" ||
		payload["canceled_at"] != "2026-07-09T01:02:03Z" ||
		payload["workflow_run_id"] != "wfr_1" ||
		payload["workflow_step_id"] != "wfs_1" {
		t.Fatalf("unexpected canceled payload: %#v", payload)
	}
	if _, ok := payload["mcp_mode"]; ok {
		t.Fatalf("canceled payload must not gain mcp_mode: %#v", payload)
	}
	if _, ok := payload["duration_ms"]; ok {
		t.Fatalf("canceled payload must not gain duration_ms: %#v", payload)
	}
}

func TestBuildTurnAgentCompactedAppendRequestPreservesPayloadContract(t *testing.T) {
	req := BuildTurnAgentCompactedAppendRequest(TurnAgentCompactedEventRequest{
		EventID:                "evt_compacted",
		MissionID:              "mis_1",
		AgentExecutor:          "codex",
		AgentModel:             "",
		AgentReasoningEffort:   "",
		MCPMode:                "auto",
		AgentSessionID:         "ses_next",
		PreviousAgentSessionID: "ses_prev",
		WorkflowRunID:          "wfr_1",
		WorkflowStepID:         "wfs_1",
		ToolSessionID:          "ses_tool",
		Summary:                "summary",
		DurationMS:             45,
		UserEventID:            "evt_user",
		Manual:                 false,
		Reason:                 "context_window_exhausted",
		Producer:               app.Producer{Type: "agent", ID: "codex"},
	})
	payload := appPayload(t, req)
	if payload["kind"] != "agent_session_compacted" ||
		payload["agent_model"] != "" ||
		payload["agent_reasoning_effort"] != "" ||
		payload["workflow_run_id"] != "wfr_1" ||
		payload["workflow_step_id"] != "wfs_1" ||
		payload["manual"] != false ||
		payload["reason"] != "context_window_exhausted" {
		t.Fatalf("unexpected compacted payload: %#v", payload)
	}
}

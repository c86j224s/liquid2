package conversation

import (
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestBuildTurnStartAppendRequestsPreserveWebPayloadContracts(t *testing.T) {
	userReq := BuildTurnUserAppendRequest(TurnUserEventRequest{
		EventID:       "evt_user",
		MissionID:     "mis_1",
		Kind:          "user_turn",
		Text:          "조사해줘",
		AgentExecutor: "codex",
		MCPMode:       "auto",
		ToolSessionID: "ses_tool",
		Producer:      app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if userReq.EventType != "turn.user" || userReq.Producer.Type != "user" || userReq.Producer.ID != "plasma-ui" {
		t.Fatalf("unexpected user turn shell: %#v", userReq)
	}
	userPayload := appPayload(t, userReq)
	if userPayload["kind"] != "user_turn" ||
		userPayload["text"] != "조사해줘" ||
		userPayload["agent_executor"] != "codex" ||
		userPayload["mcp_mode"] != "auto" ||
		userPayload["tool_session_id"] != "ses_tool" {
		t.Fatalf("unexpected user turn payload: %#v", userPayload)
	}
	if _, ok := userPayload["agent_model"]; ok {
		t.Fatalf("web user turn must not gain agent_model: %#v", userPayload)
	}

	controllerReq := BuildControllerStrategySelectedAppendRequest(ControllerStrategySelectedEventRequest{
		EventID:           "evt_controller",
		MissionID:         "mis_1",
		StrategyID:        "v2_conservative",
		StrategyLabel:     "V2 conservative",
		Reason:            "stay close",
		Guidance:          "recover direction",
		RequestedStrategy: "auto",
		AgentExecutor:     "codex",
		MCPMode:           "auto",
		UserEventID:       "evt_user",
		ToolSessionID:     "ses_tool",
		PreviousSessionID: "ses_prev",
		Producer:          app.Producer{Type: "steering_chat", ID: "plasma-controller"},
	})
	if controllerReq.EventType != "controller.strategy.selected" {
		t.Fatalf("unexpected controller event shell: %#v", controllerReq)
	}
	controllerPayload := appPayload(t, controllerReq)
	if controllerPayload["kind"] != "controller_strategy_selected" ||
		controllerPayload["strategy_id"] != "v2_conservative" ||
		controllerPayload["strategy_label"] != "V2 conservative" ||
		controllerPayload["reason"] != "stay close" ||
		controllerPayload["guidance"] != "recover direction" ||
		controllerPayload["requested_strategy"] != "auto" ||
		controllerPayload["user_event_id"] != "evt_user" ||
		controllerPayload["previous_session_id"] != "ses_prev" {
		t.Fatalf("unexpected controller payload: %#v", controllerPayload)
	}

	pendingReq := BuildTurnAgentPendingAppendRequest(TurnAgentPendingEventRequest{
		EventID:           "evt_pending",
		MissionID:         "mis_1",
		AgentExecutor:     "codex",
		MCPMode:           "auto",
		StrategyID:        "v2_conservative",
		IncludeStrategyID: true,
		Text:              "에이전트 응답을 기다리는 중입니다.",
		UserEventID:       "evt_user",
		ToolSessionID:     "ses_tool",
		StartedAt:         "2026-07-09T01:02:03Z",
		Producer:          app.Producer{Type: "agent", ID: "codex"},
	})
	if pendingReq.EventType != "turn.agent.pending" {
		t.Fatalf("unexpected pending event shell: %#v", pendingReq)
	}
	pendingPayload := appPayload(t, pendingReq)
	if pendingPayload["kind"] != "agent_pending" ||
		pendingPayload["strategy_id"] != "v2_conservative" ||
		pendingPayload["text"] != "에이전트 응답을 기다리는 중입니다." ||
		pendingPayload["started_at"] != "2026-07-09T01:02:03Z" {
		t.Fatalf("unexpected pending payload: %#v", pendingPayload)
	}
	if _, ok := pendingPayload["agent_model"]; ok {
		t.Fatalf("web pending turn must not gain agent_model: %#v", pendingPayload)
	}

	manualCompactPendingReq := BuildTurnAgentPendingAppendRequest(TurnAgentPendingEventRequest{
		EventID:           "evt_pending_compact",
		MissionID:         "mis_1",
		AgentExecutor:     "codex",
		MCPMode:           "auto",
		IncludeStrategyID: true,
		Text:              "에이전트 응답을 기다리는 중입니다.",
		UserEventID:       "evt_user",
		ToolSessionID:     "ses_tool",
		StartedAt:         "2026-07-09T01:02:03Z",
		Producer:          app.Producer{Type: "agent", ID: "codex"},
	})
	manualCompactPendingPayload := appPayload(t, manualCompactPendingReq)
	if strategyID, ok := manualCompactPendingPayload["strategy_id"]; !ok || strategyID != "" {
		t.Fatalf("web pending turn must preserve empty strategy_id key: %#v", manualCompactPendingPayload)
	}
}

func TestBuildTurnStartAppendRequestsPreserveWorkflowPayloadContracts(t *testing.T) {
	userReq := BuildTurnUserAppendRequest(TurnUserEventRequest{
		EventID:              "evt_user",
		MissionID:            "mis_1",
		Kind:                 "workflow_steering",
		Text:                 "다음 단계 조사",
		AgentExecutor:        "codex",
		AgentModel:           "",
		AgentReasoningEffort: "",
		IncludeAgentConfig:   true,
		MCPMode:              "auto",
		ToolSessionID:        "ses_tool",
		WorkflowRunID:        "wfr_1",
		WorkflowStepID:       "wfs_1",
		StepInstructionMode:  "layered",
		Producer:             app.Producer{Type: "workflow", ID: "wfr_1"},
	})
	userPayload := appPayload(t, userReq)
	if userPayload["kind"] != "workflow_steering" ||
		userPayload["workflow_run_id"] != "wfr_1" ||
		userPayload["workflow_step_id"] != "wfs_1" ||
		userPayload["step_instruction_mode"] != "layered" ||
		userPayload["agent_model"] != "" ||
		userPayload["agent_reasoning_effort"] != "" {
		t.Fatalf("unexpected workflow user payload: %#v", userPayload)
	}

	pendingReq := BuildTurnAgentPendingAppendRequest(TurnAgentPendingEventRequest{
		EventID:              "evt_pending",
		MissionID:            "mis_1",
		AgentExecutor:        "codex",
		AgentModel:           "",
		AgentReasoningEffort: "",
		IncludeAgentConfig:   true,
		MCPMode:              "auto",
		Text:                 "워크플로우 단계의 에이전트 응답을 기다리는 중입니다.",
		UserEventID:          "evt_user",
		ToolSessionID:        "ses_tool",
		StartedAt:            "2026-07-09T01:02:03Z",
		WorkflowRunID:        "wfr_1",
		WorkflowStepID:       "wfs_1",
		StepInstructionMode:  "layered",
		Producer:             app.Producer{Type: "agent", ID: "codex"},
	})
	pendingPayload := appPayload(t, pendingReq)
	if pendingPayload["kind"] != "agent_pending" ||
		pendingPayload["workflow_run_id"] != "wfr_1" ||
		pendingPayload["workflow_step_id"] != "wfs_1" ||
		pendingPayload["step_instruction_mode"] != "layered" ||
		pendingPayload["agent_model"] != "" ||
		pendingPayload["agent_reasoning_effort"] != "" {
		t.Fatalf("unexpected workflow pending payload: %#v", pendingPayload)
	}
	if _, ok := pendingPayload["strategy_id"]; ok {
		t.Fatalf("workflow pending turn must not gain strategy_id: %#v", pendingPayload)
	}
}

func TestBuildAgentSessionResetAppendRequestPreservesPayloadContract(t *testing.T) {
	req := BuildAgentSessionResetAppendRequest(AgentSessionResetEventRequest{
		EventID:                "evt_reset",
		MissionID:              "mis_1",
		AgentExecutor:          "codex",
		AgentModel:             "gpt-5.5",
		AgentReasoningEffort:   "medium",
		PreviousAgentSessionID: "ses_prev",
		Producer:               app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if req.EventType != "agent.session.reset" || req.Producer.Type != "user" || req.Producer.ID != "plasma-ui" {
		t.Fatalf("unexpected reset event shell: %#v", req)
	}
	payload := appPayload(t, req)
	if payload["kind"] != "agent_session_reset" ||
		payload["agent_executor"] != "codex" ||
		payload["agent_model"] != "gpt-5.5" ||
		payload["agent_reasoning_effort"] != "medium" ||
		payload["previous_agent_session_id"] != "ses_prev" ||
		payload["text"] != "사용자가 에이전트 세션을 새로 시작하도록 요청했습니다." {
		t.Fatalf("unexpected reset payload: %#v", payload)
	}
}

func appPayload(t *testing.T, req app.AppendEventRequest) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	return payload
}

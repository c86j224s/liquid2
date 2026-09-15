package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

type fakeService struct {
	start              workflowstate.RequestWorkflowRunRequest
	stop               workflowstate.RequestWorkflowStopRequest
	getMission, getRun string
	listMission        string
	starts, stops      int
	gets, lists        int
}

func (service *fakeService) RequestWorkflowRun(_ context.Context, request workflowstate.RequestWorkflowRunRequest) (workflowstate.WorkflowRunView, error) {
	service.start = request
	service.starts++
	return workflowstate.WorkflowRunView{WorkflowRunID: request.WorkflowRunID, MissionID: request.MissionID, Status: workflowstate.WorkflowStatusQueued}, nil
}
func (service *fakeService) GetWorkflowRun(_ context.Context, missionID, runID string) (workflowstate.WorkflowRunView, error) {
	service.getMission, service.getRun = missionID, runID
	service.gets++
	return workflowstate.WorkflowRunView{WorkflowRunID: runID, MissionID: missionID}, nil
}
func (service *fakeService) ListWorkflowRuns(_ context.Context, missionID string) ([]workflowstate.WorkflowRunView, error) {
	service.listMission, service.lists = missionID, service.lists+1
	return []workflowstate.WorkflowRunView{}, nil
}
func (service *fakeService) RequestWorkflowStop(_ context.Context, request workflowstate.RequestWorkflowStopRequest) (workflowstate.WorkflowRunView, error) {
	service.stop = request
	service.stops++
	return workflowstate.WorkflowRunView{WorkflowRunID: request.WorkflowRunID, MissionID: request.MissionID}, nil
}

type callbackCapture struct{ validation, mapped int }

func newTestHandler(service Service, binding Binding, bound func(string) error, capture *callbackCapture) *Handler {
	return NewHandler(service, binding, bound,
		func(name, mission, kind, message string, retryable bool, related []string) wire.ToolResult {
			capture.validation++
			return wire.ToolResult{ToolName: name, MissionID: mission, Error: &wire.ToolError{ErrorKind: kind, Message: message}}
		},
		func(name, mission string, err error, related []string) wire.ToolResult {
			capture.mapped++
			return wire.ToolResult{ToolName: name, MissionID: mission, Error: &wire.ToolError{ErrorKind: "mapped", Message: err.Error()}}
		},
	)
}

func call(t *testing.T, name string, args any) wire.ToolCall {
	t.Helper()
	encoded, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	return wire.ToolCall{Name: name, Arguments: encoded}
}

func TestStartValidationOrderAndFallback(t *testing.T) {
	service := &fakeService{}
	capture := &callbackCapture{}
	boundCalls := 0
	handler := newTestHandler(service, Binding{MissionID: "mis_1", AgentSessionID: "ses_bound", CurrentUserEventID: "evt_bound", AgentExecutor: "codex"}, func(string) error {
		boundCalls++
		return nil
	}, capture)

	result := handler.CallStart(context.Background(), call(t, "plasma.workflow.start", map[string]any{
		"mission_id": "mis_1", "instruction": "run", "agent_executor": "codex",
	}))
	if result.Error != nil || service.starts != 1 {
		t.Fatalf("start failed: %#v", result)
	}
	if boundCalls != 1 || service.start.RequestedByToolSessionID != "ses_bound" || service.start.StartAfterEventID != "evt_bound" {
		t.Fatalf("binding fallback not forwarded: calls=%d request=%#v", boundCalls, service.start)
	}
	if service.start.RequestedBySurface != workflowstate.WorkflowSurfaceMCP || result.Content.(map[string]any)["provider_invoked"] != false {
		t.Fatalf("unexpected start mapping: %#v", service.start)
	}

	mismatch := handler.CallStart(context.Background(), call(t, "plasma.workflow.start", map[string]any{
		"mission_id": "mis_1", "instruction": "run", "agent_executor": "claude",
	}))
	if mismatch.Error == nil || service.starts != 1 {
		t.Fatalf("executor mismatch should stop before service: %#v", mismatch)
	}
}

func TestStartBoundMissionPrecedesLaterValidation(t *testing.T) {
	service := &fakeService{}
	capture := &callbackCapture{}
	handler := newTestHandler(service, Binding{MissionID: "mis_bound", AgentExecutor: "codex"}, func(string) error { return errors.New("bound mismatch") }, capture)
	result := handler.CallStart(context.Background(), call(t, "plasma.workflow.start", map[string]any{"mission_id": "mis_other", "instruction": "run"}))
	if result.Error == nil || capture.validation != 1 || service.starts != 0 {
		t.Fatalf("expected bound mission validation: %#v capture=%#v", result, capture)
	}
}

func TestStartDecodeErrorPrecedesBinding(t *testing.T) {
	service := &fakeService{}
	capture := &callbackCapture{}
	boundCalls := 0
	handler := newTestHandler(service, Binding{MissionID: "mis_1"}, func(string) error { boundCalls++; return nil }, capture)
	result := handler.CallStart(context.Background(), wire.ToolCall{Name: "plasma.workflow.start", Arguments: json.RawMessage(`{"mission_id": 4}`)})
	if result.Error == nil || boundCalls != 0 || service.starts != 0 {
		t.Fatalf("decode should precede binding: %#v bound=%d", result, boundCalls)
	}
}

func TestStatusForwardsRawNonblankIDAndListsOnlyWhenBlank(t *testing.T) {
	service := &fakeService{}
	capture := &callbackCapture{}
	handler := newTestHandler(service, Binding{MissionID: "mis_1"}, func(string) error { return nil }, capture)
	if result := handler.CallStatus(context.Background(), call(t, "plasma.workflow.status", map[string]any{"mission_id": "mis_1", "workflow_run_id": " wfr_raw "})); result.Error != nil {
		t.Fatal(result.Error)
	}
	if service.gets != 1 || service.getRun != " wfr_raw " || service.lists != 0 {
		t.Fatalf("raw status ID not preserved: %#v", service)
	}
	if result := handler.CallStatus(context.Background(), call(t, "plasma.workflow.status", map[string]any{"mission_id": "mis_1", "workflow_run_id": "   "})); result.Error != nil {
		t.Fatal(result.Error)
	}
	if service.lists != 1 || service.listMission != "mis_1" {
		t.Fatalf("blank status ID should list: %#v", service)
	}
}

func TestStopMapsRawReasonAndSession(t *testing.T) {
	service := &fakeService{}
	capture := &callbackCapture{}
	handler := newTestHandler(service, Binding{MissionID: "mis_1", AgentSessionID: "ses_1"}, func(string) error { return nil }, capture)
	if result := handler.CallStop(context.Background(), call(t, "plasma.workflow.stop", map[string]any{"mission_id": "mis_1", "workflow_run_id": " wfr_1 ", "reason": "  stop now  "})); result.Error != nil {
		t.Fatal(result.Error)
	}
	if service.stop.WorkflowRunID != "wfr_1" || service.stop.Reason != "stop now" || service.stop.RequestedByToolSessionID != "ses_1" || service.stop.RequestedBySurface != workflowstate.WorkflowSurfaceMCP {
		t.Fatalf("unexpected stop request: %#v", service.stop)
	}
}

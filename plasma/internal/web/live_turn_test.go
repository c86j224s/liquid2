package web

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestLiveTurnStoreDeterministicLabelsAndInterleavedAnswerTransition(t *testing.T) {
	for _, tc := range []struct {
		name     string
		event    AgentObservation
		expected string
	}{
		{name: "thinking", event: AgentObservation{Type: AgentObservationPhase, Phase: AgentPhaseThinking}, expected: "생각 중..."},
		{name: "web search", event: AgentObservation{Type: AgentObservationTool, ToolCategory: AgentToolCategoryWebSearch}, expected: "웹에서 조사하는 중..."},
		{name: "web read", event: AgentObservation{Type: AgentObservationTool, ToolCategory: AgentToolCategoryWebRead}, expected: "웹 자료를 읽는 중..."},
		{name: "mission read", event: AgentObservation{Type: AgentObservationTool, ToolCategory: AgentToolCategoryMissionRead}, expected: "미션 자료를 살펴보는 중..."},
		{name: "source propose", event: AgentObservation{Type: AgentObservationTool, ToolCategory: AgentToolCategorySourcePropose}, expected: "자료를 제안하는 중..."},
		{name: "organize", event: AgentObservation{Type: AgentObservationTool, ToolCategory: AgentToolCategoryOrganize}, expected: "조사 내용을 정리하는 중..."},
		{name: "validate", event: AgentObservation{Type: AgentObservationTool, ToolCategory: AgentToolCategoryValidate}, expected: "답변을 확인하는 중..."},
		{name: "unknown", event: AgentObservation{Type: AgentObservationTool, ToolCategory: AgentToolCategoryUnknown}, expected: "작업 중..."},
	} {
		var store liveTurnStore
		store.start("mis_1", tc.name)
		store.applyObservation("mis_1", tc.name, tc.event)
		snapshot, _, unsubscribe, ok := store.subscribe("mis_1", tc.name)
		unsubscribe()
		if !ok || snapshot.Status != tc.expected {
			t.Fatalf("%s status = ok=%v %#v, want %q", tc.name, ok, snapshot, tc.expected)
		}
	}

	var store liveTurnStore
	store.start("mis_1", "evt_user")
	store.applyObservation("mis_1", "evt_user", AgentObservation{Type: AgentObservationTool, ToolCategory: AgentToolCategoryMissionRead})
	store.applyObservation("mis_1", "evt_user", AgentObservation{Type: AgentObservationAnswer, Text: "첫 답변"})
	store.applyObservation("mis_1", "evt_user", AgentObservation{Type: AgentObservationTool, ToolCategory: AgentToolCategoryWebSearch})
	snapshot, _, unsubscribe, ok := store.subscribe("mis_1", "evt_user")
	defer unsubscribe()
	if !ok || snapshot.State != "answer" || snapshot.Preview != "첫 답변" || snapshot.Status != "웹에서 조사하는 중..." {
		t.Fatalf("expected later activity to preserve answer preview and update status, got ok=%v snapshot=%#v", ok, snapshot)
	}
	store.applyObservation("mis_1", "evt_user", AgentObservation{Type: AgentObservationAnswer, Text: "둘째 답변"})
	snapshot, _, unsubscribe, ok = store.subscribe("mis_1", "evt_user")
	defer unsubscribe()
	if !ok || snapshot.State != "answer" || snapshot.Preview != "둘째 답변" || snapshot.Status != "" {
		t.Fatalf("expected later answer to update preview and return to fallback status, got ok=%v snapshot=%#v", ok, snapshot)
	}

	if _, _, unsubscribe, ok = store.subscribe("mis_other", "evt_user"); ok {
		unsubscribe()
		t.Fatal("other mission must not see live state")
	}
}

func TestWorkflowLiveTerminalState(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{name: "completed", want: "completed"},
		{name: "canceled", err: context.Canceled, want: "canceled"},
		{name: "error", err: errors.New("failed"), want: "error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := workflowLiveTerminalState(tc.err); got != tc.want {
				t.Fatalf("workflowLiveTerminalState(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

func TestLiveTurnSSEReplaysSnapshotStreamsPreviewAndCloses(t *testing.T) {
	executor := &liveTurnStreamingExecutor{release: make(chan struct{}), started: make(chan struct{})}
	server, cleanup, missionID := newLiveTurnHTTPTest(t, executor)
	defer cleanup()

	turn := postJSON(t, server+"/api/missions/"+missionID+"/turns", map[string]any{"text": "조사해줘"})
	userEventID := turn["user_event"].(map[string]any)["EventID"].(string)
	select {
	case <-executor.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for streaming executor to start")
	}

	liveCtx, cancelLive := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelLive()
	liveReq, err := http.NewRequestWithContext(liveCtx, http.MethodGet, server+"/api/missions/"+missionID+"/turns/"+userEventID+"/live", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(liveReq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected SSE 200, got %d", resp.StatusCode)
	}
	reader := bufio.NewReader(resp.Body)
	initial := readLiveTurnSSESnapshot(t, reader)
	if initial.Schema != liveTurnSchema || initial.MissionID != missionID || initial.UserEventID != userEventID || initial.State != "activity" || initial.Status != "미션 자료를 살펴보는 중..." {
		t.Fatalf("unexpected initial snapshot: %#v", initial)
	}

	close(executor.release)
	answer := readLiveTurnSSESnapshotUntil(t, reader, func(snapshot liveTurnSnapshot) bool { return snapshot.State == "answer" })
	if answer.State != "answer" || !strings.Contains(answer.Preview, "미리보기") || strings.Contains(answer.Preview, "https://example.com") {
		t.Fatalf("unexpected answer snapshot: %#v", answer)
	}
	terminal := readLiveTurnSSESnapshotUntil(t, reader, func(snapshot liveTurnSnapshot) bool { return snapshot.Terminal })
	if !terminal.Terminal || terminal.State != "completed" {
		t.Fatalf("expected completed terminal snapshot, got %#v", terminal)
	}

	detail := waitForEventType(t, server, missionID, "turn.agent.response")
	events := detail["events"].([]any)
	var payload map[string]any
	for _, value := range events {
		event := value.(map[string]any)
		if event["EventType"] == "turn.agent.response" {
			payload = event["Payload"].(map[string]any)
			break
		}
	}
	if payload["text"] != "최종 답변 https://example.com" {
		t.Fatalf("durable ledger response changed: %#v", payload)
	}
	if calls := executor.runCalls.Load(); calls != 0 {
		t.Fatalf("caption helper must not invoke AgentExecutor.Run, got %d calls", calls)
	}
}

func TestWorkflowLiveTurnSSEStreamsActivityAndAnswerAcrossWholeStep(t *testing.T) {
	executor := &workflowLiveTurnStreamingExecutor{release: make(chan struct{}), started: make(chan struct{})}
	server, cleanup, missionID := newLiveTurnHTTPTest(t, executor)
	defer cleanup()
	defer func() {
		select {
		case <-executor.release:
		default:
			close(executor.release)
		}
	}()

	startedRun := postJSON(t, server+"/api/missions/"+missionID+"/workflows", map[string]any{
		"instruction":    "워크플로로 조사해줘",
		"agent_executor": "codex",
		"max_steps":      1,
	})
	workflowRunID := startedRun["workflow_run"].(map[string]any)["workflow_run_id"].(string)
	select {
	case <-executor.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for workflow streaming executor to start")
	}

	detail := waitForEventType(t, server, missionID, "turn.agent.pending")
	var userEventID string
	for _, value := range detail["events"].([]any) {
		event := value.(map[string]any)
		if event["EventType"] != "turn.agent.pending" {
			continue
		}
		payload := event["Payload"].(map[string]any)
		if payload["workflow_run_id"] == workflowRunID {
			userEventID, _ = payload["user_event_id"].(string)
			break
		}
	}
	if userEventID == "" {
		t.Fatalf("workflow pending turn missing for run %q", workflowRunID)
	}

	liveCtx, cancelLive := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelLive()
	liveReq, err := http.NewRequestWithContext(liveCtx, http.MethodGet, server+"/api/missions/"+missionID+"/turns/"+userEventID+"/live", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(liveReq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected workflow SSE 200, got %d", resp.StatusCode)
	}
	reader := bufio.NewReader(resp.Body)
	initial := readLiveTurnSSESnapshot(t, reader)
	if initial.MissionID != missionID || initial.UserEventID != userEventID || initial.State != "activity" || initial.Status != "웹에서 조사하는 중..." {
		t.Fatalf("unexpected workflow activity snapshot: %#v", initial)
	}

	close(executor.release)
	answer := readLiveTurnSSESnapshotUntil(t, reader, func(snapshot liveTurnSnapshot) bool { return snapshot.State == "answer" })
	if answer.Preview != "워크플로 미리보기" {
		t.Fatalf("unexpected workflow answer snapshot: %#v", answer)
	}
	terminal := readLiveTurnSSESnapshotUntil(t, reader, func(snapshot liveTurnSnapshot) bool { return snapshot.Terminal })
	if terminal.State != "completed" {
		t.Fatalf("expected completed workflow live turn, got %#v", terminal)
	}
	if calls := executor.runCalls.Load(); calls != 0 {
		t.Fatalf("workflow observation must not invoke fallback Run, got %d calls", calls)
	}

	detail = waitForEventType(t, server, missionID, app.WorkflowRunCompletedEvent)
	foundResponse := false
	for _, value := range detail["events"].([]any) {
		event := value.(map[string]any)
		if event["EventType"] != "turn.agent.response" {
			continue
		}
		payload := event["Payload"].(map[string]any)
		if payload["user_event_id"] == userEventID {
			foundResponse = true
			if payload["text"] != "워크플로 최종 답변" {
				t.Fatalf("durable workflow response changed: %#v", payload)
			}
		}
	}
	if !foundResponse {
		t.Fatalf("durable workflow response missing for user event %q", userEventID)
	}
}

type liveTurnStreamingExecutor struct {
	startedOnce sync.Once
	started     chan struct{}
	release     chan struct{}
	runCalls    atomic.Int32
}

type workflowLiveTurnStreamingExecutor struct {
	startedOnce sync.Once
	started     chan struct{}
	release     chan struct{}
	runCalls    atomic.Int32
}

func (executor *workflowLiveTurnStreamingExecutor) Run(context.Context, AgentRequest) (AgentResult, error) {
	executor.runCalls.Add(1)
	return AgentResult{}, errors.New("workflow executor must use observation")
}

func (executor *workflowLiveTurnStreamingExecutor) RunWithObserver(ctx context.Context, _ AgentRequest, observer AgentObserver) (AgentResult, error) {
	observer(AgentObservation{Type: AgentObservationTool, ToolCategory: AgentToolCategoryWebSearch})
	executor.startedOnce.Do(func() { close(executor.started) })
	select {
	case <-executor.release:
	case <-ctx.Done():
		return AgentResult{}, ctx.Err()
	}
	observer(AgentObservation{Type: AgentObservationAnswer, Text: "워크플로 미리보기\nPLASMA_WORKFLOW_CONTROL:"})
	return AgentResult{
		Text:      "워크플로 최종 답변\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"stop\",\"reason\":\"done\"}",
		SessionID: "workflow-live-session",
	}, nil
}

func (executor *liveTurnStreamingExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	executor.runCalls.Add(1)
	return AgentResult{Text: "최종 답변 https://example.com", SessionID: "live-session"}, nil
}

func (executor *liveTurnStreamingExecutor) RunWithObserver(ctx context.Context, _ AgentRequest, observer AgentObserver) (AgentResult, error) {
	observer(AgentObservation{Type: AgentObservationTool, ToolCategory: AgentToolCategoryMissionRead})
	executor.startedOnce.Do(func() { close(executor.started) })
	select {
	case <-executor.release:
	case <-ctx.Done():
		return AgentResult{}, ctx.Err()
	}
	observer(AgentObservation{Type: AgentObservationAnswer, Text: "미리보기 [링크]"})
	return AgentResult{Text: "최종 답변 https://example.com", SessionID: "live-session"}, nil
}

func newLiveTurnHTTPTest(t *testing.T, executor AgentExecutor) (string, func(), string) {
	t.Helper()
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	svc := app.NewService(store)
	handler := NewServer(svc, Options{AgentExecutor: executor})
	httpServer := httptest.NewServer(handler)
	mission := postJSON(t, httpServer.URL+"/api/missions", map[string]any{"title": "Live turn"})
	return httpServer.URL, func() {
		httpServer.Close()
		_ = store.Close()
	}, mission["projection"].(map[string]any)["mission_id"].(string)
}

func readLiveTurnSSESnapshot(t *testing.T, reader *bufio.Reader) liveTurnSnapshot {
	t.Helper()
	var data string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read SSE: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		}
		if line == "" && data != "" {
			var snapshot liveTurnSnapshot
			if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
				t.Fatalf("decode SSE snapshot %q: %v", data, err)
			}
			return snapshot
		}
	}
}

func readLiveTurnSSESnapshotUntil(t *testing.T, reader *bufio.Reader, accept func(liveTurnSnapshot) bool) liveTurnSnapshot {
	t.Helper()
	for i := 0; i < 10; i++ {
		snapshot := readLiveTurnSSESnapshot(t, reader)
		if accept(snapshot) {
			return snapshot
		}
	}
	t.Fatal("timed out waiting for matching live turn snapshot")
	return liveTurnSnapshot{}
}

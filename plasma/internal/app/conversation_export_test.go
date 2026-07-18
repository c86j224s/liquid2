package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type conversationExportStore struct {
	fakeStore
	events    []LedgerEvent
	artifacts map[string]RawArtifact
}

func (s *conversationExportStore) ListLedgerEvents(_ context.Context, missionID string) ([]LedgerEvent, error) {
	var events []LedgerEvent
	for _, event := range s.events {
		if event.MissionID == missionID {
			events = append(events, event)
		}
	}
	return events, nil
}

func (s *conversationExportStore) CommitAtomicWrite(_ context.Context, write AtomicWrite) (AtomicWriteResult, error) {
	if s.artifacts == nil {
		s.artifacts = map[string]RawArtifact{}
	}
	committed := make([]LedgerEvent, 0, len(write.Events))
	for _, event := range write.Events {
		event.Sequence = int64(len(s.events) + 1)
		s.events = append(s.events, event)
		committed = append(committed, event)
	}
	for _, artifact := range write.RawArtifacts {
		s.artifacts[artifact.ArtifactID] = artifact
	}
	return AtomicWriteResult{Events: committed}, nil
}

func (s *conversationExportStore) GetRawArtifact(_ context.Context, artifactID string) (RawArtifact, error) {
	if artifact, ok := s.artifacts[artifactID]; ok {
		return artifact, nil
	}
	return s.fakeStore.GetRawArtifact(context.Background(), artifactID)
}

func TestExportConversationCreatesMarkdownArtifactFromVisibleTurns(t *testing.T) {
	store := &conversationExportStore{events: []LedgerEvent{
		testConversationEvent("evt_user", "mis_1", 1, "turn.user", map[string]any{
			"kind":            "user_turn",
			"text":            "기술면접 질문 100개를 뽑아줘",
			"tool_session_id": "ses_private",
		}),
		testConversationEvent("evt_pending", "mis_1", 2, "turn.agent.pending", map[string]any{
			"kind":            "agent_pending",
			"text":            "대기 중",
			"tool_session_id": "ses_private",
		}),
		testConversationEvent("evt_response", "mis_1", 3, "turn.agent.response", map[string]any{
			"kind":             "agent_response",
			"text":             "1. HTTP 캐시는 무엇인가?\n\n답: 응답 재사용을 제어하는 메커니즘입니다.\n\n/path note: /home/example-user/private/file.txt",
			"agent_session_id": "ses_private",
			"user_event_id":    "evt_user",
		}),
		testConversationEvent("evt_workflow", "mis_1", 4, "turn.agent.response", map[string]any{
			"kind":             "agent_response",
			"text":             "워크플로우가 확인한 결과입니다.",
			"workflow_step_id": "wfs_1",
		}),
	}}
	result, err := NewService(store).ExportConversation(context.Background(), ConversationExportRequest{
		EventID:    "evt_export",
		ArtifactID: "art_export",
		MissionID:  "mis_1",
		Title:      "면접 Q&A 원문",
		Producer:   Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		t.Fatalf("ExportConversation returned error: %v", err)
	}
	if result.Event.EventType != ConversationExportedEvent || result.EntryCount != 3 {
		t.Fatalf("unexpected result: %#v", result)
	}
	content := string(result.Artifact.Content)
	for _, expected := range []string{
		"# 면접 Q&A 원문",
		"## 1. 사용자 요청",
		"기술면접 질문 100개를 뽑아줘",
		"## 2. 에이전트 응답",
		"1. HTTP 캐시는 무엇인가?",
		"## 3. 워크플로우 단계 결과",
		"워크플로우가 확인한 결과입니다.",
		"/path/to/...",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("export missing %q:\n%s", expected, content)
		}
	}
	for _, forbidden := range []string{"대기 중", "turn.agent.pending", "ses_private", "/home/example-user"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("export leaked %q:\n%s", forbidden, content)
		}
	}
	var payload struct {
		Kind       string `json:"kind"`
		ArtifactID string `json:"artifact_id"`
		EntryCount int    `json:"entry_count"`
	}
	if err := json.Unmarshal(result.Event.Payload, &payload); err != nil {
		t.Fatalf("unmarshal export event payload: %v", err)
	}
	if payload.Kind != ConversationExportKindMarkdown || payload.ArtifactID != result.Artifact.ArtifactID || payload.EntryCount != 3 {
		t.Fatalf("unexpected event payload: %#v", payload)
	}
}

func TestExportConversationRejectsEmptyVisibleConversation(t *testing.T) {
	store := &conversationExportStore{events: []LedgerEvent{
		testConversationEvent("evt_pending", "mis_1", 1, "turn.agent.pending", map[string]any{
			"kind": "agent_pending",
			"text": "대기 중",
		}),
	}}
	_, err := NewService(store).ExportConversation(context.Background(), ConversationExportRequest{
		EventID:    "evt_export",
		ArtifactID: "art_export",
		MissionID:  "mis_1",
		Producer:   Producer{Type: "user", ID: "plasma-ui"},
	})
	if err == nil || !strings.Contains(err.Error(), "visible conversation entries") {
		t.Fatalf("expected visible-entry validation error, got %v", err)
	}
}

func testConversationEvent(eventID string, missionID string, sequence int64, eventType string, payload map[string]any) LedgerEvent {
	encoded, _ := json.Marshal(payload)
	return LedgerEvent{
		EventID:   eventID,
		MissionID: missionID,
		Sequence:  sequence,
		EventType: eventType,
		Producer:  Producer{Type: "test", ID: "test"},
		Payload:   encoded,
		CreatedAt: time.Date(2026, 7, 16, 1, 2, int(sequence), 0, time.UTC),
	}
}

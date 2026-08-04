package app

import (
	"context"
	"strings"
)

const (
	// ConversationExportedEvent는 대화내역 export artifact가 생성됐음을 기록한다.
	ConversationExportedEvent = "conversation.exported"

	// ConversationExportKindMarkdown은 현재 지원하는 대화내역 export artifact 종류다.
	ConversationExportKindMarkdown = "conversation_export_markdown"
)

// ConversationExportRequest는 미션 장부의 대화 event를 Markdown artifact로 내보내는
// 요청이다.
type ConversationExportRequest struct {
	EventID    string
	ArtifactID string
	MissionID  string
	Title      string
	Producer   Producer
}

// ConversationExportResult는 생성된 대화내역 artifact와 기록 event를 함께 반환한다.
type ConversationExportResult struct {
	Artifact   RawArtifact
	Event      LedgerEvent
	EntryCount int
}

// ExportConversation은 미션 장부에서 사용자/agent 대화에 해당하는 event만 읽어
// Markdown artifact로 저장한다.
//
// report처럼 내용을 재작성하지 않고, 읽기 쉬운 transcript artifact를 만드는 것이
// 계약이다.
func (s *Service) ExportConversation(ctx context.Context, req ConversationExportRequest) (ConversationExportResult, error) {
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return ConversationExportResult{}, err
	}
	if err := validateProducer(req.Producer); err != nil {
		return ConversationExportResult{}, err
	}
	events, err := s.ListEvents(ctx, missionID)
	if err != nil {
		return ConversationExportResult{}, err
	}
	title := conversationExportTitle(req.Title)
	content, entryCount, err := buildConversationExportMarkdown(title, events)
	if err != nil {
		return ConversationExportResult{}, err
	}
	artifact, err := buildRawArtifact(CreateRawArtifactRequest{
		ArtifactID: strings.TrimSpace(req.ArtifactID),
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   conversationExportFilename(title),
		Producer:   req.Producer,
		Content:    content,
	})
	if err != nil {
		return ConversationExportResult{}, err
	}
	event, err := buildLedgerEvent(AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: missionID,
		EventType: ConversationExportedEvent,
		Producer:  req.Producer,
		Payload: mustJSON(map[string]any{
			"kind":        ConversationExportKindMarkdown,
			"title":       title,
			"artifact_id": artifact.ArtifactID,
			"media_type":  artifact.MediaType,
			"entry_count": entryCount,
			"text":        "대화내역 export artifact를 생성했습니다.",
		}),
	})
	if err != nil {
		return ConversationExportResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:       []LedgerEvent{event},
		RawArtifacts: []RawArtifact{artifact},
	})
	if err != nil {
		return ConversationExportResult{}, err
	}
	return ConversationExportResult{
		Artifact:   artifact,
		Event:      committed.Events[0],
		EntryCount: entryCount,
	}, nil
}

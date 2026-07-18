package app

import (
	"context"
	"strings"
)

const (
	ConversationExportedEvent = "conversation.exported"

	ConversationExportKindMarkdown = "conversation_export_markdown"
)

type ConversationExportRequest struct {
	EventID    string
	ArtifactID string
	MissionID  string
	Title      string
	Producer   Producer
}

type ConversationExportResult struct {
	Artifact   RawArtifact
	Event      LedgerEvent
	EntryCount int
}

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

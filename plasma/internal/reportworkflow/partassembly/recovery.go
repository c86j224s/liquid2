package partassembly

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

// CreatedEventType은 Part assembly가 완료 시 남기는 durable event type이다.
const CreatedEventType = "report.part.created"

// RecoverService는 Part recovery가 artifact bytes를 검증할 때 필요한 조회 포트다.
type RecoverService interface {
	GetRawArtifact(context.Context, string) (artifact.Raw, error)
}

// RecoverInput은 저장된 Part event와 artifact가 현재 canonical plan 범위에
// 들어오는지 검증하는 typed 계약이다.
type RecoverInput struct {
	Service        RecoverService
	Event          ledger.Event
	MissionID      string
	PendingEventID string
	PlanEventID    string
	Plan           reporting.SectionalReportPlan
}

// RecoverOutput은 root가 zero-based Part 순서로 집계할 수 있는 복구 결과다.
type RecoverOutput struct {
	PartIndex int
	Draft     PartDraft
}

// Recover는 report.part.created payload와 Markdown artifact를 함께 검증한다.
// plan 밖 event는 안전하게 무시되어 requirements mapping skip 근거가 되지 않는다.
func Recover(ctx context.Context, input RecoverInput) (RecoverOutput, bool, error) {
	if input.Event.EventType != CreatedEventType {
		return RecoverOutput{}, false, nil
	}
	var payload struct {
		PendingEventID string `json:"pending_event_id"`
		PlanEventID    string `json:"plan_event_id"`
		ArtifactID     string `json:"artifact_id"`
		Title          string `json:"title"`
		AgentSessionID string `json:"agent_session_id"`
		PartIndex      int    `json:"part_index"`
		WordCount      int    `json:"word_count"`
	}
	if json.Unmarshal(input.Event.Payload, &payload) != nil ||
		strings.TrimSpace(input.Event.MissionID) != strings.TrimSpace(input.MissionID) ||
		strings.TrimSpace(payload.PendingEventID) != strings.TrimSpace(input.PendingEventID) ||
		strings.TrimSpace(payload.PlanEventID) != strings.TrimSpace(input.PlanEventID) ||
		payload.PartIndex < 1 || payload.PartIndex > len(input.Plan.Parts) {
		return RecoverOutput{}, false, nil
	}
	markdown, ok := recoverMarkdown(ctx, input.Service, strings.TrimSpace(payload.ArtifactID), input.MissionID)
	if !ok {
		return RecoverOutput{}, false, nil
	}
	return RecoverOutput{
		PartIndex: payload.PartIndex - 1,
		Draft: PartDraft{
			Title: strings.TrimSpace(payload.Title), Markdown: markdown,
			ArtifactID: strings.TrimSpace(payload.ArtifactID),
			WordCount:  fallbackWordCount(payload.WordCount, markdown),
			SessionID:  strings.TrimSpace(payload.AgentSessionID),
		},
	}, true, nil
}

func recoverMarkdown(ctx context.Context, service RecoverService, artifactID, missionID string) (string, bool) {
	if service == nil || strings.TrimSpace(artifactID) == "" {
		return "", false
	}
	raw, err := service.GetRawArtifact(ctx, artifactID)
	if err != nil || raw.MissionID != strings.TrimSpace(missionID) {
		return "", false
	}
	if !strings.HasPrefix(strings.ToLower(raw.MediaType), "text/markdown") {
		return "", false
	}
	markdown := strings.TrimSpace(string(raw.Content))
	return markdown, markdown != ""
}

func fallbackWordCount(count int, markdown string) int {
	if count > 0 {
		return count
	}
	return longformutil.WordCount(markdown)
}

package sectiondraft

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

// CreatedEventType은 Section writer가 완료 시 남기는 durable event type이다.
const CreatedEventType = "report.section.created"

// EvidenceGapEventType은 Section writer가 Markdown 대신 evidence gap을 선언할 때 남기는 durable event type이다.
const EvidenceGapEventType = "report.section.evidence_gap"

// RecoverService는 Section recovery가 artifact bytes를 검증할 때 필요한 조회 포트다.
type RecoverService interface {
	GetRawArtifact(context.Context, string) (artifact.Raw, error)
}

// RecoverInput은 저장된 Section event와 artifact가 현재 canonical plan 범위에
// 들어오는지 검증하는 typed 계약이다.
type RecoverInput struct {
	Service        RecoverService
	Event          ledger.Event
	MissionID      string
	PendingEventID string
	PlanEventID    string
	Plan           reporting.SectionalReportPlan
}

// RecoverOutput은 root가 zero-based 좌표로 집계할 수 있는 복구 Section 결과다.
type RecoverOutput struct {
	PartIndex    int
	SectionIndex int
	Draft        Draft
}

// RecoverEvidenceGapInput is the typed contract for adopting a durable
// evidence-gap attempt into the current Section writer retry budget.
type RecoverEvidenceGapInput struct {
	Event               ledger.Event
	MissionID           string
	PendingEventID      string
	PlanEventID         string
	Plan                reporting.SectionalReportPlan
	AgentExecutor       string
	SessionChainKind    string
	ReportPlanSessionID string
}

// Recover는 report.section.created payload와 Markdown artifact를 함께 검증한다.
// 범위 밖이거나 과거 호환상 무시 가능한 event는 ok=false로 돌려 요구사항 skip 신호가 되지 않게 한다.
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
		SectionIndex   int    `json:"section_index"`
		WordCount      int    `json:"word_count"`
	}
	if json.Unmarshal(input.Event.Payload, &payload) != nil ||
		strings.TrimSpace(input.Event.MissionID) != strings.TrimSpace(input.MissionID) ||
		strings.TrimSpace(payload.PendingEventID) != strings.TrimSpace(input.PendingEventID) ||
		strings.TrimSpace(payload.PlanEventID) != strings.TrimSpace(input.PlanEventID) ||
		!sectionInPlan(payload.PartIndex, payload.SectionIndex, input.Plan) {
		return RecoverOutput{}, false, nil
	}
	markdown, ok := recoverMarkdown(ctx, input.Service, strings.TrimSpace(payload.ArtifactID), input.MissionID)
	if !ok {
		return RecoverOutput{}, false, nil
	}
	return RecoverOutput{
		PartIndex: payload.PartIndex - 1, SectionIndex: payload.SectionIndex - 1,
		Draft: Draft{
			Title: strings.TrimSpace(payload.Title), Markdown: markdown,
			ArtifactID: strings.TrimSpace(payload.ArtifactID),
			WordCount:  fallbackWordCount(payload.WordCount, markdown),
			SessionID:  strings.TrimSpace(payload.AgentSessionID),
		},
	}, true, nil
}

// RecoverEvidenceGap validates a durable evidence-gap attempt for the current
// pending event. It deliberately ignores gap events from retry ancestors so a
// user-created report retry receives a fresh two-attempt Section budget.
func RecoverEvidenceGap(input RecoverEvidenceGapInput) (EvidenceGap, bool, error) {
	event := input.Event
	if event.EventType != EvidenceGapEventType {
		return EvidenceGap{}, false, nil
	}
	var payload struct {
		PendingEventID           string `json:"pending_event_id"`
		PlanEventID              string `json:"plan_event_id"`
		PartIndex                int    `json:"part_index"`
		SectionIndex             int    `json:"section_index"`
		Attempt                  int    `json:"attempt_number"`
		ReasonCode               string `json:"reason_code"`
		AgentExecutor            string `json:"agent_executor"`
		AgentSessionID           string `json:"agent_session_id"`
		PreviousAgentSessionID   string `json:"previous_agent_session_id"`
		ReturnedAgentSessionID   string `json:"returned_agent_session_id"`
		ToolSessionID            string `json:"tool_session_id"`
		SessionChainKind         string `json:"session_chain_kind"`
		ReportPlanSessionID      string `json:"report_plan_session_id"`
		ReportSessionID          string `json:"report_session_id"`
		ForkSourceAgentSessionID string `json:"fork_source_agent_session_id"`
		DurationMS               int64  `json:"duration_ms"`
	}
	if json.Unmarshal(event.Payload, &payload) != nil {
		return EvidenceGap{}, false, nil
	}
	agentSessionID := strings.TrimSpace(payload.AgentSessionID)
	if strings.TrimSpace(event.MissionID) != strings.TrimSpace(input.MissionID) ||
		event.Producer.Type != "agent_session" ||
		strings.TrimSpace(event.Producer.ID) != agentSessionID ||
		strings.TrimSpace(payload.PendingEventID) != strings.TrimSpace(input.PendingEventID) ||
		strings.TrimSpace(payload.PlanEventID) != strings.TrimSpace(input.PlanEventID) ||
		strings.TrimSpace(payload.ReasonCode) != EvidenceGapReasonCode ||
		agentSessionID == "" ||
		strings.TrimSpace(payload.PreviousAgentSessionID) != agentSessionID ||
		strings.TrimSpace(payload.ReturnedAgentSessionID) != agentSessionID ||
		strings.TrimSpace(payload.ReportSessionID) != agentSessionID ||
		strings.TrimSpace(payload.AgentExecutor) != strings.TrimSpace(input.AgentExecutor) ||
		strings.TrimSpace(payload.SessionChainKind) != strings.TrimSpace(input.SessionChainKind) ||
		strings.TrimSpace(payload.ReportPlanSessionID) != strings.TrimSpace(input.ReportPlanSessionID) ||
		strings.TrimSpace(payload.ToolSessionID) == "" ||
		payload.Attempt < 1 || payload.Attempt > MaxEvidenceGapAttempts ||
		!sectionInPlan(payload.PartIndex, payload.SectionIndex, input.Plan) {
		return EvidenceGap{}, false, nil
	}
	return EvidenceGap{
		PartIndex: payload.PartIndex, SectionIndex: payload.SectionIndex, Attempt: payload.Attempt,
		ReasonCode: strings.TrimSpace(payload.ReasonCode), SessionID: strings.TrimSpace(payload.AgentSessionID),
		ReturnedSessionID: strings.TrimSpace(payload.ReturnedAgentSessionID),
		PreviousSessionID: strings.TrimSpace(payload.PreviousAgentSessionID),
		ToolSessionID:     strings.TrimSpace(payload.ToolSessionID),
		SourceSessionID:   strings.TrimSpace(payload.ForkSourceAgentSessionID),
		DurationMS:        payload.DurationMS,
	}, true, nil
}

func sectionInPlan(partIndex, sectionIndex int, plan reporting.SectionalReportPlan) bool {
	if partIndex < 1 || partIndex > len(plan.Parts) {
		return false
	}
	sections := plan.Parts[partIndex-1].Sections
	return sectionIndex >= 1 && sectionIndex <= len(sections)
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

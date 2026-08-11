package partedit

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

// RecoverInput은 저장된 Part-edit outcome이 현재 canonical plan과 source Part에
// 속하는지 검증하기 위한 typed replay 계약이다.
type RecoverInput struct {
	Service                      Service
	Event                        ledger.Event
	Events                       []ledger.Event
	MissionID                    string
	PendingEventID               string
	PlanEventID                  string
	Plan                         reporting.SectionalReportPlan
	Sources                      map[int]PartDraft
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSelectionSource         string
	MCPMode                      string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
	SessionChainKind             string
	ReportPlanSessionID          string
}

// RecoverOutput은 root가 zero-based Part 순서로 집계할 수 있는 복구 편집 결과다.
type RecoverOutput struct {
	PartIndex int
	Draft     PartDraft
}

// Recover는 report.part.edited outcome을 durable replay 계약으로 다시 읽는다.
// source Part event와 outcome artifact 검증은 stage 경계 안에 머물고 root는 결과만 집계한다.
func Recover(ctx context.Context, input RecoverInput) (RecoverOutput, bool, error) {
	if input.Event.EventType != reporting.PartEditedEventType {
		return RecoverOutput{}, false, nil
	}
	var payload struct {
		PendingEventID string `json:"pending_event_id"`
		PlanEventID    string `json:"plan_event_id"`
		PartIndex      int    `json:"part_index"`
		WordCount      int    `json:"edited_word_count"`
	}
	if json.Unmarshal(input.Event.Payload, &payload) != nil ||
		strings.TrimSpace(input.Event.MissionID) != strings.TrimSpace(input.MissionID) ||
		strings.TrimSpace(payload.PendingEventID) != strings.TrimSpace(input.PendingEventID) ||
		strings.TrimSpace(payload.PlanEventID) != strings.TrimSpace(input.PlanEventID) ||
		payload.PartIndex < 1 || payload.PartIndex > len(input.Plan.Parts) {
		return RecoverOutput{}, false, nil
	}
	source, exists := input.Sources[payload.PartIndex-1]
	if !exists || strings.TrimSpace(source.ArtifactID) == "" {
		return RecoverOutput{}, false, nil
	}
	sourcePartEventID, err := sourcePartEventID(input.Events, input.PlanEventID, payload.PartIndex, source.ArtifactID)
	if err != nil || sourcePartEventID == "" {
		return RecoverOutput{}, false, err
	}
	outcome, ok, err := reporting.LoadPartEditOutcome(ctx, input.Service, reporting.PartEditOutcomeContract{
		MissionID: input.MissionID, CurrentPendingEventID: input.PendingEventID, PlanEventID: input.PlanEventID,
		SourcePartEventID: sourcePartEventID, SourceArtifactID: source.ArtifactID, PartIndex: payload.PartIndex,
		AgentExecutor: input.AgentExecutor, AgentModel: input.AgentModel, AgentReasoningEffort: input.AgentReasoningEffort,
		AgentSelectionSource: input.AgentSelectionSource, MCPMode: input.MCPMode,
		ReportSessionPolicy: input.ReportSessionPolicy, ReportSessionPolicySelection: input.ReportSessionPolicySelection,
		GenerationGuidanceProfile: input.GenerationGuidanceProfile, GenerationGuidanceSHA256: input.GenerationGuidanceSHA256,
		SessionChainKind: input.SessionChainKind, ReportPlanSessionID: input.ReportPlanSessionID,
	})
	if err != nil || !ok || outcome.Event.EventID != input.Event.EventID {
		return RecoverOutput{}, ok && outcome.Event.EventID == input.Event.EventID, err
	}
	markdown := strings.TrimSpace(string(outcome.Artifact.Content))
	if markdown == "" {
		return RecoverOutput{}, false, nil
	}
	return RecoverOutput{
		PartIndex: payload.PartIndex - 1,
		Draft: PartDraft{
			Title: source.Title, Markdown: markdown, ArtifactID: outcome.Artifact.ArtifactID,
			WordCount: fallbackWordCount(payload.WordCount, markdown),
		},
	}, true, nil
}

func fallbackWordCount(count int, markdown string) int {
	if count > 0 {
		return count
	}
	return longformutil.WordCount(markdown)
}

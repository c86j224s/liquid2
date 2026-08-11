package partedit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// CurrentStart는 같은 Part edit start가 이미 durable하게 시작됐는지 검증 후 복원한다.
func (runner Runner) CurrentStart(ctx context.Context, input Input, expectedProviderSessionID string) (reporting.PartEditBinding, bool, error) {
	contract, err := runner.startContract(ctx, input, expectedProviderSessionID)
	if err != nil {
		return reporting.PartEditBinding{}, false, err
	}
	return reporting.LoadCurrentPartEditStart(ctx, runner.Service, contract)
}

func (runner Runner) binding(ctx context.Context, input Input) (reporting.PartEditBinding, error) {
	sourcePartEventID, err := runner.reportPartCreatedEventID(ctx, input.Base.MissionID, input.Base.PlanEvent.EventID, input.PartIndex+1, input.Source.ArtifactID)
	if err != nil {
		return reporting.PartEditBinding{}, err
	}
	mapHash := ""
	if strings.TrimSpace(input.Base.RequirementMapEvent.EventID) != "" {
		mapHash, _, err = reporting.ReportRequirementMapHash(input.Base.RequirementMap)
		if err != nil {
			return reporting.PartEditBinding{}, err
		}
	}
	return reporting.PartEditBinding{
		MissionID: input.Base.MissionID, PendingEventID: input.Base.PendingEventID, PlanEventID: input.Base.PlanEvent.EventID,
		SourcePartEventID: sourcePartEventID, SourceArtifactID: input.Source.ArtifactID,
		EditedArtifactID: input.EditedArtifactID, Filename: input.Filename,
		ToolSessionID: input.ToolSessionID, ProviderSessionID: input.PreviousSessionID,
		PreviousProviderSessionID: input.PreviousSessionID,
		IdempotencyKey:            fmt.Sprintf("report-part-edit:%s:%s:%d", input.Base.PendingEventID, input.Base.PlanEvent.EventID, input.PartIndex+1),
		PartIndex:                 input.PartIndex + 1,
		RequirementMapEventID:     strings.TrimSpace(input.Base.RequirementMapEvent.EventID),
		RequirementMapHash:        mapHash,
		AgentExecutor:             input.Base.AgentExecutor, AgentModel: input.Base.AgentModel,
		AgentReasoningEffort: input.Base.AgentReasoningEffort, AgentSelectionSource: input.Base.AgentSelectionSource,
		MCPMode: input.Base.MCPMode, ReportSessionPolicy: input.Base.ReportSessionPolicy,
		ReportSessionPolicySelection: input.Base.ReportSessionPolicySelection,
		GenerationGuidanceProfile:    input.Base.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     input.Base.GenerationGuidanceSHA256,
		SessionChainKind:             input.Base.SessionChainKind,
		ReportPlanSessionID:          input.Base.ReportPlanSessionID,
		ForkSourceAgentSessionID:     input.ForkSourceAgentSessionID,
	}, nil
}

// Binding은 Part edit/author MCP 도구가 사용할 durable identity 계약을 만든다.
// source Part event 조회와 requirement map hash 검증은 이 stage 경계에 머문다.
func (runner Runner) Binding(ctx context.Context, input Input) (reporting.PartEditBinding, error) {
	return runner.binding(ctx, input)
}

func (runner Runner) startContract(ctx context.Context, input Input, expectedProviderSessionID string) (reporting.PartEditStartContract, error) {
	binding, err := runner.binding(ctx, input)
	if err != nil {
		return reporting.PartEditStartContract{}, err
	}
	return reporting.PartEditStartContract{
		MissionID: input.Base.MissionID, CurrentPendingEventID: input.Base.PendingEventID, PlanEventID: input.Base.PlanEvent.EventID,
		SourcePartEventID: binding.SourcePartEventID, SourceArtifactID: input.Source.ArtifactID, PartIndex: input.PartIndex + 1,
		IdempotencyKey: binding.IdempotencyKey, RequirementMapEventID: binding.RequirementMapEventID,
		RequirementMapHash: binding.RequirementMapHash,
		AgentExecutor:      binding.AgentExecutor, AgentModel: binding.AgentModel,
		AgentReasoningEffort: binding.AgentReasoningEffort, AgentSelectionSource: binding.AgentSelectionSource,
		MCPMode: binding.MCPMode, ReportSessionPolicy: binding.ReportSessionPolicy,
		ReportSessionPolicySelection: binding.ReportSessionPolicySelection,
		GenerationGuidanceProfile:    binding.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     binding.GenerationGuidanceSHA256,
		SessionChainKind:             binding.SessionChainKind,
		ReportPlanSessionID:          binding.ReportPlanSessionID,
		ForkSourceAgentSessionID:     binding.ForkSourceAgentSessionID,
		ExpectedProviderSessionID:    expectedProviderSessionID,
		ExcludedProviderSessionIDs:   []string{input.Base.ReportPlanSessionID},
	}, nil
}

// StartContract는 durable start replay가 비교할 canonical Part edit 계약을 반환한다.
// Web 호환 wrapper는 이 값을 읽기만 하며 lineage 정책을 다시 구현하지 않는다.
func (runner Runner) StartContract(ctx context.Context, input Input, expectedProviderSessionID string) (reporting.PartEditStartContract, error) {
	return runner.startContract(ctx, input, expectedProviderSessionID)
}

func (runner Runner) reportPartCreatedEventID(ctx context.Context, missionID string, planEventID string, partIndex int, artifactID string) (string, error) {
	events, err := runner.Service.ListEvents(ctx, missionID)
	if err != nil {
		return "", err
	}
	return sourcePartEventID(events, planEventID, partIndex, artifactID)
}

func sourcePartEventID(events []ledger.Event, planEventID string, partIndex int, artifactID string) (string, error) {
	found := ""
	for _, event := range events {
		if event.EventType != "report.part.created" {
			continue
		}
		var payload struct {
			PlanEventID string `json:"plan_event_id"`
			ArtifactID  string `json:"artifact_id"`
			PartIndex   int    `json:"part_index"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.PlanEventID) != planEventID || strings.TrimSpace(payload.ArtifactID) != artifactID || payload.PartIndex != partIndex {
			continue
		}
		if found != "" {
			return "", fmt.Errorf("%w: source Part event is duplicated", producterror.ErrConflict)
		}
		found = event.EventID
	}
	if found == "" {
		return "", fmt.Errorf("%w: source Part event is missing", producterror.ErrConflict)
	}
	return found, nil
}

// SourcePartEventID는 edited Part가 어떤 durable source Part event를 대상으로 하는지 검증한다.
func (runner Runner) SourcePartEventID(ctx context.Context, missionID string, planEventID string, partIndex int, artifactID string) (string, error) {
	return runner.reportPartCreatedEventID(ctx, missionID, planEventID, partIndex, artifactID)
}

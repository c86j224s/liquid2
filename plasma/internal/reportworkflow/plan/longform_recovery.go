package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type longFormPlanPayload struct {
	PendingEventID               string                        `json:"pending_event_id"`
	ReportMode                   string                        `json:"report_mode"`
	ArtifactID                   string                        `json:"artifact_id"`
	AgentSessionID               string                        `json:"agent_session_id"`
	PreviousAgentSessionID       string                        `json:"previous_agent_session_id"`
	AgentExecutor                string                        `json:"agent_executor"`
	AgentModel                   string                        `json:"agent_model"`
	AgentReasoningEffort         string                        `json:"agent_reasoning_effort"`
	AgentSelectionSource         string                        `json:"agent_selection_source"`
	MCPMode                      string                        `json:"mcp_mode"`
	ReportSessionPolicy          string                        `json:"report_session_policy"`
	ReportSessionPolicySelection string                        `json:"report_session_policy_selection"`
	SessionChainKind             string                        `json:"session_chain_kind"`
	PreReportResearchSessionID   string                        `json:"pre_report_research_session_id"`
	ReportPlanSessionID          string                        `json:"report_plan_session_id"`
	ForkSourceSessionID          string                        `json:"fork_source_agent_session_id"`
	GenerationGuidanceProfile    string                        `json:"generation_guidance_profile"`
	GenerationGuidanceSHA256     string                        `json:"generation_guidance_sha256"`
	PartEditEnabled              bool                          `json:"part_edit_enabled"`
	PartPlanningEnabled          bool                          `json:"part_planning_enabled"`
	FinalEditPipeline            string                        `json:"final_edit_pipeline"`
	Plan                         reporting.SectionalReportPlan `json:"plan"`
}

// RecoverLongForm는 같은 retry lineage에 이미 기록된 canonical long-form plan을 복원한다.
func (runner Runner) RecoverLongForm(ctx context.Context, input LongFormInput) (LongFormOutput, bool, error) {
	events, err := runner.Service.ListEvents(ctx, input.MissionID)
	if err != nil {
		return LongFormOutput{}, false, err
	}
	lineage, err := RecoveryLineage(events, input.PendingEventID)
	if err != nil {
		return LongFormOutput{}, false, err
	}
	var recovered LongFormOutput
	for _, attemptID := range lineage {
		for _, event := range events {
			if event.EventType != "report.plan.created" {
				continue
			}
			applied, err := applyRecoveredLongFormPlan(attemptID, event, &recovered)
			if err != nil {
				return LongFormOutput{}, false, err
			}
			if applied {
				recovered.Event = event
			}
		}
	}
	return recovered, recovered.ReportPlanSessionID != "", nil
}

func applyRecoveredLongFormPlan(pendingEventID string, event ledger.Event, recovered *LongFormOutput) (bool, error) {
	var payload longFormPlanPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return false, nil
	}
	if strings.TrimSpace(payload.PendingEventID) != pendingEventID {
		return false, nil
	}
	if payload.PartPlanningEnabled {
		parent, err := partPlanningParent(event, pendingEventID)
		if err != nil {
			return false, err
		}
		if parent.ReportMode != reportexecution.ModeLongForm {
			return false, fmt.Errorf("%w: Part planning parent report mode is invalid", producterror.ErrConflict)
		}
		return applyRecoveredLongFormPlanPayload(event, payload, parent, recovered)
	}
	if strings.TrimSpace(payload.ReportMode) != reportexecution.ModeLongForm {
		return false, nil
	}
	parent := reporting.PartPlanParentState{
		AgentExecutor: payload.AgentExecutor, AgentModel: payload.AgentModel,
		AgentReasoningEffort: payload.AgentReasoningEffort, AgentSelectionSource: payload.AgentSelectionSource,
		ReportMode: reportexecution.ModeLongForm, ReportSessionPolicy: firstNonEmpty(payload.ReportSessionPolicy, reportexecution.SessionPolicySameSession),
		ReportSessionPolicySelection: strings.TrimSpace(payload.ReportSessionPolicySelection),
		SessionChainKind:             firstNonEmpty(payload.SessionChainKind, "same_session_report"),
		ReportPlanSessionID:          firstNonEmpty(payload.ReportPlanSessionID, payload.AgentSessionID),
		GenerationGuidanceProfile:    payload.GenerationGuidanceProfile, GenerationGuidanceSHA256: payload.GenerationGuidanceSHA256,
		PartEditEnabled: payload.PartEditEnabled, PartPlanningEnabled: payload.PartPlanningEnabled,
	}
	return applyRecoveredLongFormPlanPayload(event, payload, parent, recovered)
}

func applyRecoveredLongFormPlanPayload(event ledger.Event, payload longFormPlanPayload, parent reporting.PartPlanParentState, recovered *LongFormOutput) (bool, error) {
	if recovered.ReportPlanSessionID != "" {
		return false, fmt.Errorf("%w: multiple recovered long-form report plans match one pending event", producterror.ErrConflict)
	}
	normalized, err := reporting.NormalizeSectionalReportPlan(payload.Plan)
	if err != nil {
		return false, err
	}
	pipeline, err := recoveredFinalEditPipeline(event)
	if err != nil {
		return false, err
	}
	recovered.Plan = normalized
	recovered.Event = event
	recovered.ArtifactID = strings.TrimSpace(payload.ArtifactID)
	recovered.ReportPlanSessionID = parent.ReportPlanSessionID
	recovered.ReportSessionPolicy = parent.ReportSessionPolicy
	recovered.ReportSessionPolicySelection = parent.ReportSessionPolicySelection
	recovered.AgentExecutor = strings.TrimSpace(strings.ToLower(parent.AgentExecutor))
	recovered.AgentModel = strings.TrimSpace(parent.AgentModel)
	recovered.AgentReasoningEffort = strings.TrimSpace(parent.AgentReasoningEffort)
	recovered.AgentSelectionSource = strings.TrimSpace(parent.AgentSelectionSource)
	recovered.MCPMode = strings.TrimSpace(payload.MCPMode)
	recovered.SessionChainKind = parent.SessionChainKind
	recovered.PreReportResearchSessionID = firstNonEmpty(payload.PreReportResearchSessionID, payload.PreviousAgentSessionID)
	recovered.ForkSourceSessionID = strings.TrimSpace(payload.ForkSourceSessionID)
	recovered.GenerationGuidanceProfile = parent.GenerationGuidanceProfile
	recovered.GenerationGuidanceSHA256 = parent.GenerationGuidanceSHA256
	recovered.PartEditEnabled = parent.PartEditEnabled
	recovered.PartPlanningEnabled = parent.PartPlanningEnabled
	recovered.FinalEditPipeline = pipeline
	recovered.Recovered = true
	return true, nil
}

// recoveredFinalEditPipeline은 durable canonical plan event만 final tail 선택의
// 원천으로 인정한다. 미지원 literal과 humanize 불일치는 reporting의 기존 conflict
// 계약을 그대로 반환한다.
func recoveredFinalEditPipeline(event ledger.Event) (string, error) {
	state, ok, err := reporting.FinalEditPipelineFromPlanEvent(event)
	if err != nil || !ok {
		return "", err
	}
	return state.Pipeline, nil
}

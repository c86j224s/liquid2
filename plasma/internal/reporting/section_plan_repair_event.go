package reporting

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

type sectionPlanRepairParentState struct {
	pendingEventID string
	plan           SectionalReportPlan
	metadata       PartPlanParentState
}

type sectionPlanRepairEventPayload struct {
	Kind                         string                         `json:"kind"`
	PendingEventID               string                         `json:"pending_event_id"`
	PlanEventID                  string                         `json:"plan_event_id"`
	RepairRound                  int                            `json:"repair_round"`
	AgentExecutor                string                         `json:"agent_executor"`
	AgentModel                   string                         `json:"agent_model"`
	AgentReasoningEffort         string                         `json:"agent_reasoning_effort"`
	AgentSelectionSource         string                         `json:"agent_selection_source"`
	AgentSessionID               string                         `json:"agent_session_id"`
	PreviousAgentSessionID       string                         `json:"previous_agent_session_id"`
	ReturnedAgentSessionID       string                         `json:"returned_agent_session_id"`
	ToolSessionID                string                         `json:"tool_session_id"`
	ReportMode                   string                         `json:"report_mode"`
	ReportSessionPolicy          string                         `json:"report_session_policy"`
	ReportSessionPolicySelection string                         `json:"report_session_policy_selection"`
	GenerationGuidanceProfile    string                         `json:"generation_guidance_profile"`
	GenerationGuidanceSHA256     string                         `json:"generation_guidance_sha256"`
	SessionChainKind             string                         `json:"session_chain_kind"`
	ReportPlanSessionID          string                         `json:"report_plan_session_id"`
	ReportSessionID              string                         `json:"report_session_id"`
	Outcome                      string                         `json:"outcome"`
	Coordinates                  []ReportSectionCoordinate      `json:"coordinates"`
	Replacements                 []ReportSectionPlanReplacement `json:"replacements"`
}

// BuildLongFormSectionPlanRepairCompletedAppendRequest builds the immutable
// outcome event for the single bounded repair round.
func BuildLongFormSectionPlanRepairCompletedAppendRequest(req LongFormSectionPlanRepairEventRequest) ledger.AppendRequest {
	base := req.MarkdownReportEventBase
	payload := markdownReportBasePayload(base)
	payload["kind"] = longFormSectionPlanRepairKind
	payload["plan_event_id"] = strings.TrimSpace(req.PlanEventID)
	payload["repair_round"] = 1
	payload["coordinates"] = req.Coordinates
	payload["outcome"] = sectionPlanRepairOutcomeApplied
	payload["replacements"] = req.Replacements
	payload["duration_ms"] = base.DurationMS
	payload["text"] = "근거가 부족한 Section의 계획을 한 번 조정했습니다."
	if req.Unrepairable {
		payload["outcome"] = sectionPlanRepairOutcomeUnrepairable
		delete(payload, "replacements")
		payload["text"] = "근거가 부족한 Section을 대체할 수 없어 계획 보정을 종료했습니다."
	}
	addReportAgentUsage(payload, base)
	planEventID := strings.TrimSpace(req.PlanEventID)
	return ledger.AppendRequest{
		EventID:          strings.TrimSpace(base.EventID),
		MissionID:        strings.TrimSpace(base.MissionID),
		EventType:        LongFormSectionPlanRepairCompletedEventType,
		Producer:         base.Producer,
		CausationEventID: planEventID,
		CorrelationID:    "report-section-plan-repair:" + planEventID,
		Payload:          mustJSON(payload),
	}
}

func sectionPlanRepairParent(events []ledger.Event, missionID, planEventID string, original SectionalReportPlan) (sectionPlanRepairParentState, error) {
	for _, event := range events {
		if event.EventID != strings.TrimSpace(planEventID) || event.EventType != "report.plan.created" {
			continue
		}
		if strings.TrimSpace(event.MissionID) != strings.TrimSpace(missionID) {
			return sectionPlanRepairParentState{}, fmt.Errorf("%w: Section plan repair mission differs", producterror.ErrConflict)
		}
		var payload struct {
			PendingEventID string              `json:"pending_event_id"`
			Plan           SectionalReportPlan `json:"plan"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil {
			return sectionPlanRepairParentState{}, fmt.Errorf("%w: Section plan repair parent is invalid", producterror.ErrConflict)
		}
		plan, err := NormalizeSectionalReportPlan(payload.Plan)
		if err != nil {
			return sectionPlanRepairParentState{}, err
		}
		expected, err := NormalizeSectionalReportPlan(original)
		if err != nil || !reflect.DeepEqual(plan, expected) {
			return sectionPlanRepairParentState{}, fmt.Errorf("%w: Section plan repair parent plan differs", producterror.ErrConflict)
		}
		metadata, ok, err := DecodePartPlanParent(event, payload.PendingEventID, event.EventID)
		if err != nil || !ok {
			return sectionPlanRepairParentState{}, fmt.Errorf("%w: Section plan repair parent metadata is invalid", producterror.ErrConflict)
		}
		return sectionPlanRepairParentState{pendingEventID: strings.TrimSpace(payload.PendingEventID), plan: plan, metadata: metadata}, nil
	}
	return sectionPlanRepairParentState{}, fmt.Errorf("%w: Section plan repair parent is missing", producterror.ErrConflict)
}

func validateSectionPlanRepairRequest(req LongFormSectionPlanRepairEventRequest, parent sectionPlanRepairParentState) error {
	base := req.MarkdownReportEventBase
	metadata := parent.metadata
	planSessionID := strings.TrimSpace(metadata.ReportPlanSessionID)
	if strings.TrimSpace(base.MissionID) == "" || strings.TrimSpace(base.PendingEventID) == "" || strings.TrimSpace(req.PlanEventID) == "" ||
		strings.TrimSpace(base.AgentExecutor) != metadata.AgentExecutor || strings.TrimSpace(base.AgentModel) != metadata.AgentModel ||
		strings.TrimSpace(base.AgentReasoningEffort) != metadata.AgentReasoningEffort || strings.TrimSpace(base.AgentSelectionSource) != metadata.AgentSelectionSource ||
		strings.TrimSpace(base.ReportMode) != ModeLongForm || strings.TrimSpace(base.ReportSessionPolicy) != metadata.ReportSessionPolicy ||
		strings.TrimSpace(base.ReportSessionPolicySelection) != metadata.ReportSessionPolicySelection ||
		strings.TrimSpace(base.GenerationGuidanceProfile) != metadata.GenerationGuidanceProfile ||
		strings.TrimSpace(base.GenerationGuidanceSHA256) != metadata.GenerationGuidanceSHA256 ||
		strings.TrimSpace(base.SessionChainKind) != metadata.SessionChainKind ||
		strings.TrimSpace(base.ReportPlanSessionID) != planSessionID || strings.TrimSpace(base.AgentSessionID) != planSessionID ||
		strings.TrimSpace(base.PreviousAgentSessionID) != planSessionID || strings.TrimSpace(base.ReturnedAgentSessionID) != planSessionID ||
		strings.TrimSpace(base.ReportSessionID) != planSessionID || strings.TrimSpace(base.ToolSessionID) == "" ||
		base.Producer.Type != "agent_session" || strings.TrimSpace(base.Producer.ID) != planSessionID {
		return fmt.Errorf("%w: Section plan repair provenance differs from its parent", producterror.ErrConflict)
	}
	return nil
}

func matchingSectionPlanRepairs(events []ledger.Event, planEventID string, lineage map[string]bool) ([]ledger.Event, error) {
	matches := []ledger.Event{}
	for _, event := range events {
		if event.EventType != LongFormSectionPlanRepairCompletedEventType {
			continue
		}
		payload, err := sectionPlanRepairPayload(event)
		if err != nil {
			if strings.TrimSpace(event.CausationEventID) == strings.TrimSpace(planEventID) {
				return nil, err
			}
			continue
		}
		if strings.TrimSpace(payload.PlanEventID) != strings.TrimSpace(planEventID) {
			continue
		}
		if !lineage[strings.TrimSpace(payload.PendingEventID)] {
			return nil, fmt.Errorf("%w: Section plan repair is outside the report retry lineage", producterror.ErrConflict)
		}
		matches = append(matches, event)
	}
	return matches, nil
}

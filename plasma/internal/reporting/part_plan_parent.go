package reporting

import (
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

func partPlanExpectationForParent(req PartPlanCreatedEventRequest, parent PartPlanParentState) (StoredPartPlanExpectation, error) {
	expected := normalizeStoredPartPlanExpectation(StoredPartPlanExpectation{
		MissionID:                    strings.TrimSpace(req.MissionID),
		PendingEventID:               strings.TrimSpace(req.PendingEventID),
		PlanEventID:                  strings.TrimSpace(req.PlanEventID),
		PartIndex:                    req.PartIndex,
		PartCount:                    parent.PartCount,
		AgentExecutor:                parent.AgentExecutor,
		AgentModel:                   parent.AgentModel,
		AgentReasoningEffort:         parent.AgentReasoningEffort,
		AgentSelectionSource:         parent.AgentSelectionSource,
		ReportMode:                   parent.ReportMode,
		ReportSessionPolicy:          parent.ReportSessionPolicy,
		ReportSessionPolicySelection: parent.ReportSessionPolicySelection,
		GenerationGuidanceProfile:    parent.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     parent.GenerationGuidanceSHA256,
		SessionChainKind:             parent.SessionChainKind,
		ReportPlanSessionID:          parent.ReportPlanSessionID,
	})
	request := partPlanExpectationForRequest(req, parent.PartCount)
	if request.AgentExecutor != expected.AgentExecutor ||
		request.AgentModel != expected.AgentModel ||
		request.AgentReasoningEffort != expected.AgentReasoningEffort ||
		request.AgentSelectionSource != expected.AgentSelectionSource ||
		request.ReportMode != expected.ReportMode ||
		request.ReportSessionPolicy != expected.ReportSessionPolicy ||
		request.ReportSessionPolicySelection != expected.ReportSessionPolicySelection ||
		request.GenerationGuidanceProfile != expected.GenerationGuidanceProfile ||
		request.GenerationGuidanceSHA256 != expected.GenerationGuidanceSHA256 ||
		request.SessionChainKind != expected.SessionChainKind ||
		request.ReportPlanSessionID != expected.ReportPlanSessionID {
		return StoredPartPlanExpectation{}, fmt.Errorf("%w: Part plan request provenance differs from its parent", producterror.ErrConflict)
	}
	return expected, nil
}

func partPlanParent(events []ledger.Event, pendingEventID string, planEventID string) (PartPlanParentState, bool, error) {
	for _, event := range events {
		parent, ok, err := DecodePartPlanParent(event, pendingEventID, planEventID)
		if err != nil || ok {
			return parent, ok, err
		}
	}
	return PartPlanParentState{}, false, nil
}

// DecodePartPlanParent는 part plan parent payload를 후속 stage 입력으로 복원한다.
func DecodePartPlanParent(event ledger.Event, pendingEventID string, planEventID string) (PartPlanParentState, bool, error) {
	if event.EventID != strings.TrimSpace(planEventID) || event.EventType != "report.plan.created" {
		return PartPlanParentState{}, false, nil
	}
	var payload struct {
		Kind                         string `json:"kind"`
		PendingEventID               string `json:"pending_event_id"`
		PartEditEnabled              bool   `json:"part_edit_enabled"`
		PartPlanningEnabled          bool   `json:"part_planning_enabled"`
		AgentExecutor                string `json:"agent_executor"`
		AgentModel                   string `json:"agent_model"`
		AgentReasoningEffort         string `json:"agent_reasoning_effort"`
		AgentSelectionSource         string `json:"agent_selection_source"`
		ReportMode                   string `json:"report_mode"`
		ReportSessionPolicy          string `json:"report_session_policy"`
		ReportSessionPolicySelection string `json:"report_session_policy_selection"`
		GenerationGuidanceProfile    string `json:"generation_guidance_profile"`
		GenerationGuidanceSHA256     string `json:"generation_guidance_sha256"`
		SessionChainKind             string `json:"session_chain_kind"`
		ReportPlanSessionID          string `json:"report_plan_session_id"`
		Plan                         struct {
			Parts []json.RawMessage `json:"parts"`
		} `json:"plan"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part plan parent payload is invalid", producterror.ErrConflict)
	}
	if strings.TrimSpace(payload.PendingEventID) != strings.TrimSpace(pendingEventID) {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part plan parent pending id does not match", producterror.ErrConflict)
	}
	if strings.TrimSpace(payload.Kind) != reportPlanKind(ModeLongForm) {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part plan parent kind is invalid", producterror.ErrConflict)
	}
	if strings.TrimSpace(payload.ReportPlanSessionID) == "" {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part plan parent report session is missing", producterror.ErrConflict)
	}
	if len(payload.Plan.Parts) == 0 {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part plan parent has no Parts", producterror.ErrConflict)
	}
	if payload.PartPlanningEnabled && !payload.PartEditEnabled {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part planning requires Part edit on the parent plan", producterror.ErrConflict)
	}
	if payload.PartPlanningEnabled && strings.TrimSpace(payload.ReportMode) != ModeLongForm {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part planning requires long-form report mode", producterror.ErrConflict)
	}
	if payload.PartPlanningEnabled &&
		(strings.TrimSpace(payload.AgentExecutor) == "" ||
			strings.TrimSpace(payload.ReportSessionPolicy) == "") {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part plan parent provenance is incomplete", producterror.ErrConflict)
	}
	if payload.PartPlanningEnabled && strings.TrimSpace(payload.SessionChainKind) != "section_fanout_report" {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part planning requires section_fanout lineage", producterror.ErrConflict)
	}
	return PartPlanParentState{
		PartEditEnabled:              payload.PartEditEnabled,
		PartPlanningEnabled:          payload.PartPlanningEnabled,
		PartCount:                    len(payload.Plan.Parts),
		AgentExecutor:                strings.TrimSpace(strings.ToLower(payload.AgentExecutor)),
		AgentModel:                   strings.TrimSpace(payload.AgentModel),
		AgentReasoningEffort:         strings.TrimSpace(payload.AgentReasoningEffort),
		AgentSelectionSource:         strings.TrimSpace(payload.AgentSelectionSource),
		ReportMode:                   strings.TrimSpace(payload.ReportMode),
		ReportSessionPolicy:          strings.TrimSpace(payload.ReportSessionPolicy),
		ReportSessionPolicySelection: strings.TrimSpace(payload.ReportSessionPolicySelection),
		GenerationGuidanceProfile:    strings.TrimSpace(payload.GenerationGuidanceProfile),
		GenerationGuidanceSHA256:     strings.TrimSpace(payload.GenerationGuidanceSHA256),
		SessionChainKind:             strings.TrimSpace(payload.SessionChainKind),
		ReportPlanSessionID:          strings.TrimSpace(payload.ReportPlanSessionID),
	}, true, nil
}

package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	PartPlanCreatedEventType = "report.part_plan.created"
	PartPlanCreatedKind      = "sectional_markdown_report_part_plan"
	maxPartPlanBriefBytes    = 16 * 1024
)

// PartPlanCreatedEventRequest는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type PartPlanCreatedEventRequest struct {
	MarkdownReportStageEventBase
	PartIndex int
	Brief     string
}

// PartPlanResult는 part plan 제출 이벤트와 artifact를 함께 반환한다.
type PartPlanResult struct {
	Event             app.LedgerEvent
	Brief             string
	ProviderSessionID string
	PartIndex         int
}

// PartPlanStore는 part plan conditional append에 필요한 저장소 계약이다.
type PartPlanStore interface {
	AppendEventConditionally(context.Context, string, func([]app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.LedgerEvent, bool, error)
}

// FinalizePartPlan는 part plan 제출을 검증하고 저장용 이벤트 요청으로 만든다.
func FinalizePartPlan(ctx context.Context, service PartPlanStore, req PartPlanCreatedEventRequest) (PartPlanResult, error) {
	req.Brief = strings.TrimSpace(req.Brief)
	base := req.MarkdownReportStageEventBase
	if service == nil || strings.TrimSpace(base.MissionID) == "" || strings.TrimSpace(base.PendingEventID) == "" || strings.TrimSpace(base.PlanEventID) == "" || req.PartIndex < 1 || req.Brief == "" {
		return PartPlanResult{}, fmt.Errorf("%w: Part plan is incomplete", app.ErrInvalidInput)
	}
	if len([]byte(req.Brief)) > maxPartPlanBriefBytes {
		return PartPlanResult{}, fmt.Errorf("%w: Part plan brief is too large", app.ErrInvalidInput)
	}
	if strings.TrimSpace(base.ReportPlanSessionID) == "" ||
		strings.TrimSpace(base.AgentSessionID) == "" ||
		strings.TrimSpace(base.ToolSessionID) == "" ||
		strings.TrimSpace(base.AgentSessionID) == strings.TrimSpace(base.ReportPlanSessionID) ||
		strings.TrimSpace(base.PreviousAgentSessionID) != strings.TrimSpace(base.AgentSessionID) ||
		strings.TrimSpace(base.ReturnedAgentSessionID) != strings.TrimSpace(base.AgentSessionID) ||
		strings.TrimSpace(base.ReportSessionID) != strings.TrimSpace(base.AgentSessionID) ||
		strings.TrimSpace(base.ForkSourceAgentSessionID) != strings.TrimSpace(base.ReportPlanSessionID) ||
		base.Producer.Type != "agent_session" ||
		strings.TrimSpace(base.Producer.ID) != strings.TrimSpace(base.AgentSessionID) {
		return PartPlanResult{}, fmt.Errorf("%w: Part plan provider session is invalid", app.ErrInvalidInput)
	}

	storedExpectation := StoredPartPlanExpectation{}
	event, _, err := service.AppendEventConditionally(ctx, base.MissionID, func(events []app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		parent, ok, err := partPlanParent(events, base.PendingEventID, base.PlanEventID)
		if err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if !ok {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: Part plan parent is missing", app.ErrConflict)
		}
		if !parent.PartPlanningEnabled {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: Part planning is not enabled for this report plan", app.ErrConflict)
		}
		if req.PartIndex > parent.PartCount {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: Part plan index is outside the report plan", app.ErrConflict)
		}
		if parent.ReportPlanSessionID != strings.TrimSpace(base.ReportPlanSessionID) {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: Part plan report session does not match its parent", app.ErrConflict)
		}
		expected, err := partPlanExpectationForParent(req, parent)
		if err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		storedExpectation = expected
		matches := matchingPartPlanEvents(events, base.PendingEventID, base.PlanEventID, req.PartIndex)
		if len(matches) > 1 {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: multiple Part plans match one Part", app.ErrConflict)
		}
		if len(matches) == 1 {
			if err := validatePartPlanCreatedEvent(matches[0], expected); err != nil {
				return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
			}
			return app.AppendEventRequest{}, matches[0], false, nil
		}
		request := BuildPartPlanCreatedAppendRequest(req)
		if err := validatePartPlanCreatedRequest(request, expected); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		return request, app.LedgerEvent{}, true, nil
	})
	if err != nil {
		return PartPlanResult{}, err
	}
	result, ok, err := DecodeStoredPartPlan(event, storedExpectation)
	if err != nil {
		return PartPlanResult{}, err
	}
	if !ok {
		return PartPlanResult{}, fmt.Errorf("%w: finalized Part plan is invalid", app.ErrConflict)
	}
	return result, nil
}

// BuildPartPlanCreatedAppendRequest는 보고서 생성 파이프라인에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildPartPlanCreatedAppendRequest(req PartPlanCreatedEventRequest) app.AppendEventRequest {
	base := req.MarkdownReportStageEventBase
	payload := markdownReportStagePayload(base)
	delete(payload, "artifact_id")
	delete(payload, "media_type")
	payload["kind"] = PartPlanCreatedKind
	payload["part_index"] = req.PartIndex
	payload["stage_kind"] = "part_plan"
	payload["stage_id"] = fmt.Sprintf("part-plan-%d", req.PartIndex)
	payload["brief"] = strings.TrimSpace(req.Brief)
	payload["duration_ms"] = base.DurationMS
	payload["text"] = "장문 리포트 Part의 읽기 흐름을 계획했습니다."
	addReportStageAgentUsage(payload, base)
	return app.AppendEventRequest{
		EventID:          strings.TrimSpace(base.EventID),
		MissionID:        strings.TrimSpace(base.MissionID),
		EventType:        PartPlanCreatedEventType,
		Producer:         base.Producer,
		CausationEventID: strings.TrimSpace(base.PlanEventID),
		CorrelationID:    fmt.Sprintf("report-part-plan:%s:%s:%d", strings.TrimSpace(base.PendingEventID), strings.TrimSpace(base.PlanEventID), req.PartIndex),
		Payload:          mustJSON(payload),
	}
}

// PartPlanParentState는 계산한 읽기 모델이다. 원천 상태는 장부와 저장소에 남아 있다.
type PartPlanParentState struct {
	PartEditEnabled              bool
	PartPlanningEnabled          bool
	PartCount                    int
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSelectionSource         string
	ReportMode                   string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
	SessionChainKind             string
	ReportPlanSessionID          string
}

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
		return StoredPartPlanExpectation{}, fmt.Errorf("%w: Part plan request provenance differs from its parent", app.ErrConflict)
	}
	return expected, nil
}

func partPlanParent(events []app.LedgerEvent, pendingEventID string, planEventID string) (PartPlanParentState, bool, error) {
	for _, event := range events {
		parent, ok, err := DecodePartPlanParent(event, pendingEventID, planEventID)
		if err != nil || ok {
			return parent, ok, err
		}
	}
	return PartPlanParentState{}, false, nil
}

// DecodePartPlanParent는 part plan parent payload를 후속 stage 입력으로 복원한다.
func DecodePartPlanParent(event app.LedgerEvent, pendingEventID string, planEventID string) (PartPlanParentState, bool, error) {
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
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part plan parent payload is invalid", app.ErrConflict)
	}
	if strings.TrimSpace(payload.PendingEventID) != strings.TrimSpace(pendingEventID) {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part plan parent pending id does not match", app.ErrConflict)
	}
	if strings.TrimSpace(payload.Kind) != reportPlanKind(ModeLongForm) {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part plan parent kind is invalid", app.ErrConflict)
	}
	if strings.TrimSpace(payload.ReportPlanSessionID) == "" {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part plan parent report session is missing", app.ErrConflict)
	}
	if len(payload.Plan.Parts) == 0 {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part plan parent has no Parts", app.ErrConflict)
	}
	if payload.PartPlanningEnabled && !payload.PartEditEnabled {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part planning requires Part edit on the parent plan", app.ErrConflict)
	}
	if payload.PartPlanningEnabled && strings.TrimSpace(payload.ReportMode) != ModeLongForm {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part planning requires long-form report mode", app.ErrConflict)
	}
	if payload.PartPlanningEnabled &&
		(strings.TrimSpace(payload.AgentExecutor) == "" ||
			strings.TrimSpace(payload.ReportSessionPolicy) == "") {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part plan parent provenance is incomplete", app.ErrConflict)
	}
	if payload.PartPlanningEnabled && strings.TrimSpace(payload.SessionChainKind) != "section_fanout_report" {
		return PartPlanParentState{}, false, fmt.Errorf("%w: Part planning requires section_fanout lineage", app.ErrConflict)
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

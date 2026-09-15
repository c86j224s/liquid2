package reporting

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
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
	Event             ledger.Event
	Brief             string
	ProviderSessionID string
	PartIndex         int
}

// PartPlanStore는 part plan conditional append에 필요한 저장소 계약이다.
type PartPlanStore interface {
	AppendEventConditionally(context.Context, string, func([]ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error)) (ledger.Event, bool, error)
}

// FinalizePartPlan는 part plan 제출을 검증하고 저장용 이벤트 요청으로 만든다.
func FinalizePartPlan(ctx context.Context, service PartPlanStore, req PartPlanCreatedEventRequest) (PartPlanResult, error) {
	req.Brief = strings.TrimSpace(req.Brief)
	base := req.MarkdownReportStageEventBase
	if service == nil || strings.TrimSpace(base.MissionID) == "" || strings.TrimSpace(base.PendingEventID) == "" || strings.TrimSpace(base.PlanEventID) == "" || req.PartIndex < 1 || req.Brief == "" {
		return PartPlanResult{}, fmt.Errorf("%w: Part plan is incomplete", producterror.ErrInvalidInput)
	}
	if len([]byte(req.Brief)) > maxPartPlanBriefBytes {
		return PartPlanResult{}, fmt.Errorf("%w: Part plan brief is too large", producterror.ErrInvalidInput)
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
		return PartPlanResult{}, fmt.Errorf("%w: Part plan provider session is invalid", producterror.ErrInvalidInput)
	}

	storedExpectation := StoredPartPlanExpectation{}
	event, _, err := service.AppendEventConditionally(ctx, base.MissionID, func(events []ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error) {
		parent, ok, err := partPlanParent(events, base.PendingEventID, base.PlanEventID)
		if err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if !ok {
			return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("%w: Part plan parent is missing", producterror.ErrConflict)
		}
		if !parent.PartPlanningEnabled {
			return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("%w: Part planning is not enabled for this report plan", producterror.ErrConflict)
		}
		if req.PartIndex > parent.PartCount {
			return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("%w: Part plan index is outside the report plan", producterror.ErrConflict)
		}
		if parent.ReportPlanSessionID != strings.TrimSpace(base.ReportPlanSessionID) {
			return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("%w: Part plan report session does not match its parent", producterror.ErrConflict)
		}
		expected, err := partPlanExpectationForParent(req, parent)
		if err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		storedExpectation = expected
		matches := matchingPartPlanEvents(events, base.PendingEventID, base.PlanEventID, req.PartIndex)
		if len(matches) > 1 {
			return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("%w: multiple Part plans match one Part", producterror.ErrConflict)
		}
		if len(matches) == 1 {
			if err := validatePartPlanCreatedEvent(matches[0], expected); err != nil {
				return ledger.AppendRequest{}, ledger.Event{}, false, err
			}
			return ledger.AppendRequest{}, matches[0], false, nil
		}
		request := BuildPartPlanCreatedAppendRequest(req)
		if err := validatePartPlanCreatedRequest(request, expected); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		return request, ledger.Event{}, true, nil
	})
	if err != nil {
		return PartPlanResult{}, err
	}
	result, ok, err := DecodeStoredPartPlan(event, storedExpectation)
	if err != nil {
		return PartPlanResult{}, err
	}
	if !ok {
		return PartPlanResult{}, fmt.Errorf("%w: finalized Part plan is invalid", producterror.ErrConflict)
	}
	return result, nil
}

// BuildPartPlanCreatedAppendRequest는 보고서 생성 파이프라인에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildPartPlanCreatedAppendRequest(req PartPlanCreatedEventRequest) ledger.AppendRequest {
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
	return ledger.AppendRequest{
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

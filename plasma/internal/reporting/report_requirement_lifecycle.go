package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// ReportRequirementMapLifecycleService는 요구사항 map 제출과 조회에 필요한 service 포트다.
type ReportRequirementMapLifecycleService interface {
	SelectReportRequirementMap(context.Context, app.ReportRequirementMapQuery) (app.ReportRequirementMapSelection, error)
}

// ReportRequirementMapAgentResult는 agent가 제출한 요구사항 map artifact와 실행 metadata다.
type ReportRequirementMapAgentResult struct {
	Text       string
	SessionID  string
	Resumed    bool
	DurationMS int64
	Usage      agentusage.AgentUsage
}

// ReportRequirementMapLifecycleRequest는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type ReportRequirementMapLifecycleRequest struct {
	MissionID, PendingEventID, PlanEventID, AgentExecutor, AgentModel, AgentReasoningEffort, PreviousProviderSessionID string
	Plan                                                                                                               SectionalReportPlan
	Invoke                                                                                                             func(context.Context, ReportRequirementMapBinding) (ReportRequirementMapAgentResult, error)
}

// ReportRequirementMapLifecycleResult는 요구사항 map 제출 이벤트와 agent 결과를 함께 반환한다.
type ReportRequirementMapLifecycleResult struct {
	RequirementMap ReportRequirementMap
	Event          app.LedgerEvent
	Binding        ReportRequirementMapBinding
	Agent          ReportRequirementMapAgentResult
}

// RunReportRequirementMapLifecycle는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func (runner Runner) RunReportRequirementMapLifecycle(ctx context.Context, req ReportRequirementMapLifecycleRequest) (ReportRequirementMapLifecycleResult, error) {
	service, ok := runner.Service.(ReportRequirementMapLifecycleService)
	if !ok {
		return ReportRequirementMapLifecycleResult{}, fmt.Errorf("%w: durable report requirement lifecycle service is required", app.ErrInvalidInput)
	}
	if req.Invoke == nil {
		return ReportRequirementMapLifecycleResult{}, fmt.Errorf("%w: report requirement lifecycle callback is required", app.ErrInvalidInput)
	}
	binding := ReportRequirementMapBinding{
		MissionID: req.MissionID, PendingEventID: req.PendingEventID, PlanEventID: req.PlanEventID,
		ToolSessionID: runner.id("ses"), PreviousProviderSessionID: strings.TrimSpace(req.PreviousProviderSessionID),
		IdempotencyKey: runner.id("rrk"), AgentExecutor: req.AgentExecutor, AgentModel: req.AgentModel,
		AgentReasoningEffort: req.AgentReasoningEffort,
	}
	binding.Producer = app.Producer{Type: "agent_session", ID: binding.ToolSessionID}
	if _, err := runner.Service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: runner.id("evt"), MissionID: req.MissionID, EventType: ReportRequirementsStartedEventType,
		Producer: binding.Producer, CausationEventID: req.PlanEventID, CorrelationID: req.PendingEventID,
		Payload: mustJSON(map[string]any{
			"kind": "sectional_markdown_report_requirements_started", "pending_event_id": req.PendingEventID,
			"plan_event_id": req.PlanEventID, "stage_kind": "requirements", "stage_id": "requirements",
			"tool_session_id": binding.ToolSessionID, "previous_agent_session_id": binding.PreviousProviderSessionID,
			"agent_executor": req.AgentExecutor, "agent_model": req.AgentModel,
			"agent_reasoning_effort": req.AgentReasoningEffort, "text": "사용자 출력 요구 연결을 시작했습니다.",
		}),
	}); err != nil {
		return ReportRequirementMapLifecycleResult{}, err
	}
	agent, err := req.Invoke(ctx, binding)
	if err != nil {
		return ReportRequirementMapLifecycleResult{}, err
	}
	if agent.Text != ReportRequirementsMappedSentinel {
		return ReportRequirementMapLifecycleResult{}, fmt.Errorf("%w: report requirement agent did not confirm mapping submission", app.ErrInvalidInput)
	}
	selection, err := service.SelectReportRequirementMap(ctx, app.ReportRequirementMapQuery{
		MissionID: req.MissionID, PendingEventID: req.PendingEventID, PlanEventID: req.PlanEventID,
		ToolSessionID: binding.ToolSessionID, PreviousProviderSessionID: binding.PreviousProviderSessionID,
		AgentExecutor: req.AgentExecutor, AgentModel: req.AgentModel, AgentReasoningEffort: req.AgentReasoningEffort,
		IdempotencyKey: binding.IdempotencyKey,
	})
	if err != nil {
		return ReportRequirementMapLifecycleResult{}, err
	}
	var requirementMap ReportRequirementMap
	if json.Unmarshal(selection.RequirementMap, &requirementMap) != nil {
		return ReportRequirementMapLifecycleResult{}, fmt.Errorf("%w: submitted report requirement map is invalid", app.ErrInvalidInput)
	}
	requirementMap, err = NormalizeReportRequirementMap(requirementMap, req.Plan)
	if err != nil {
		return ReportRequirementMapLifecycleResult{}, err
	}
	hash, _, err := ReportRequirementMapHash(requirementMap)
	if err != nil || hash != selection.RequirementMapHash {
		return ReportRequirementMapLifecycleResult{}, fmt.Errorf("%w: report requirement map hash mismatch", app.ErrConflict)
	}
	if usageStore, ok := runner.Service.(ReportAgentUsageStore); ok {
		if _, _, usageErr := RecordReportAgentUsage(context.WithoutCancel(ctx), usageStore, ReportAgentUsageRequest{
			MissionID: req.MissionID, PendingEventID: req.PendingEventID, CanonicalEventID: selection.Event.EventID,
			Surface: "report_requirements", PreviousAgentSessionID: binding.PreviousProviderSessionID,
			AgentSessionID: agent.SessionID, DurationMS: agent.DurationMS, Resumed: agent.Resumed, Usage: agent.Usage,
		}); usageErr != nil {
			log.Printf("report_agent_usage_write_failed mission_id=%q canonical_event_id=%q surface=%q err=%q", req.MissionID, selection.Event.EventID, "report_requirements", usageErr)
		}
	} else {
		log.Printf("report_agent_usage_store_unavailable mission_id=%q canonical_event_id=%q surface=%q", req.MissionID, selection.Event.EventID, "report_requirements")
	}
	return ReportRequirementMapLifecycleResult{RequirementMap: requirementMap, Event: selection.Event, Binding: binding, Agent: agent}, nil
}

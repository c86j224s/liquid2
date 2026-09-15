package requirements

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportusage"
)

const ReportRequirementsMappedSentinel = "REQUIREMENTS_MAPPED"

// ReportRequirementMapLifecycleService is the durable requirement-map selection port.
type ReportRequirementMapLifecycleService interface {
	SelectReportRequirementMap(context.Context, reporting.ReportRequirementMapQuery) (reporting.ReportRequirementMapSelection, error)
}

// ReportRequirementMapAgentResult carries the provider mapping result and usage metadata.
type ReportRequirementMapAgentResult struct {
	Text       string
	SessionID  string
	Resumed    bool
	DurationMS int64
	Usage      agentusage.AgentUsage
}

// ReportRequirementMapLifecycleRequest is the stage-owned requirement mapping input.
type ReportRequirementMapLifecycleRequest struct {
	MissionID, PendingEventID, PlanEventID, AgentExecutor, AgentModel, AgentReasoningEffort, PreviousProviderSessionID string
	Plan                                                                                                               reporting.SectionalReportPlan
	Invoke                                                                                                             func(context.Context, reporting.ReportRequirementMapBinding) (ReportRequirementMapAgentResult, error)
}

// ReportRequirementMapLifecycleResult returns the normalized map and durable event.
type ReportRequirementMapLifecycleResult struct {
	RequirementMap reporting.ReportRequirementMap
	Event          ledger.Event
	Binding        reporting.ReportRequirementMapBinding
	Agent          ReportRequirementMapAgentResult
}

// RunReportRequirementMapLifecycle starts, validates, selects, and records a requirement map.
func (runner Runner) RunReportRequirementMapLifecycle(ctx context.Context, req ReportRequirementMapLifecycleRequest) (ReportRequirementMapLifecycleResult, error) {
	service, ok := runner.Lifecycle.Service.(ReportRequirementMapLifecycleService)
	if !ok {
		return ReportRequirementMapLifecycleResult{}, fmt.Errorf("%w: durable report requirement lifecycle service is required", producterror.ErrInvalidInput)
	}
	if req.Invoke == nil {
		return ReportRequirementMapLifecycleResult{}, fmt.Errorf("%w: report requirement lifecycle callback is required", producterror.ErrInvalidInput)
	}
	binding := reporting.ReportRequirementMapBinding{
		MissionID: req.MissionID, PendingEventID: req.PendingEventID, PlanEventID: req.PlanEventID,
		ToolSessionID: lifecycleID(runner, "ses"), PreviousProviderSessionID: strings.TrimSpace(req.PreviousProviderSessionID),
		IdempotencyKey: lifecycleID(runner, "rrk"), AgentExecutor: req.AgentExecutor, AgentModel: req.AgentModel,
		AgentReasoningEffort: req.AgentReasoningEffort,
	}
	binding.Producer = ledger.Producer{Type: "agent_session", ID: binding.ToolSessionID}
	if _, err := runner.Lifecycle.Service.AppendEvent(ctx, ledger.AppendRequest{
		EventID: lifecycleID(runner, "evt"), MissionID: req.MissionID, EventType: reporting.ReportRequirementsStartedEventType,
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
		return ReportRequirementMapLifecycleResult{}, fmt.Errorf("%w: report requirement agent did not confirm mapping submission", producterror.ErrInvalidInput)
	}
	selection, err := service.SelectReportRequirementMap(ctx, reporting.ReportRequirementMapQuery{
		MissionID: req.MissionID, PendingEventID: req.PendingEventID, PlanEventID: req.PlanEventID,
		ToolSessionID: binding.ToolSessionID, PreviousProviderSessionID: binding.PreviousProviderSessionID,
		AgentExecutor: req.AgentExecutor, AgentModel: req.AgentModel, AgentReasoningEffort: req.AgentReasoningEffort,
		IdempotencyKey: binding.IdempotencyKey,
	})
	if err != nil {
		return ReportRequirementMapLifecycleResult{}, err
	}
	var requirementMap reporting.ReportRequirementMap
	if json.Unmarshal(selection.RequirementMap, &requirementMap) != nil {
		return ReportRequirementMapLifecycleResult{}, fmt.Errorf("%w: submitted report requirement map is invalid", producterror.ErrInvalidInput)
	}
	requirementMap, err = reporting.NormalizeReportRequirementMap(requirementMap, req.Plan)
	if err != nil {
		return ReportRequirementMapLifecycleResult{}, err
	}
	hash, _, err := reporting.ReportRequirementMapHash(requirementMap)
	if err != nil || hash != selection.RequirementMapHash {
		return ReportRequirementMapLifecycleResult{}, fmt.Errorf("%w: report requirement map hash mismatch", producterror.ErrConflict)
	}
	if usageStore, ok := runner.Lifecycle.Service.(reportusage.ReportAgentUsageStore); ok {
		if _, _, usageErr := reportusage.RecordReportAgentUsage(context.WithoutCancel(ctx), usageStore, reportusage.ReportAgentUsageRequest{
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

func lifecycleID(runner Runner, prefix string) string {
	if runner.Lifecycle.NewID == nil {
		return prefix + "_report"
	}
	return runner.Lifecycle.NewID(prefix)
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

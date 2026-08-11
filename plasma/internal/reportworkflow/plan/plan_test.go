package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

func TestRunMarkdownFreshPlanUsesEmptyPreviousSession(t *testing.T) {
	service := &fakePlanService{
		events: []ledger.Event{{
			EventID: "evt_pending", MissionID: "mis_1", EventType: "report.draft.pending",
			Payload: mustJSON(map[string]any{"origin_pending_event_id": "evt_pending", "retry_strategy": "initial"}),
		}},
		selection: reporting.ReportPlanSubmissionSelection{
			EventID: "evt_submitted", ArgumentsHash: "args", PlanHash: "plan_hash",
			Plan: mustJSON(reporting.ReportPlan{Summary: "Plan", Sections: []reporting.ReportPlanSection{{Title: "Section"}}}),
		},
	}
	executor := &fakePlanExecutor{results: []agentexec.AgentResult{{Text: reporting.ReportPlanSubmittedSentinel, SessionID: "plan-session-1"}}}
	runner := Runner{
		Service: service, Executor: executor, NewID: sequenceID(),
		Lifecycle:       reporting.Runner(reportexecution.Runner{Service: service, NewID: sequenceID()}),
		LatestSessionID: func(context.Context, string, string) string { return "research-session-1" },
	}
	out, err := runner.RunMarkdown(context.Background(), Input{
		MissionID: "mis_1", PendingEventID: "evt_pending", Title: "Report",
		AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "medium",
		MCPMode: "auto", Rigor: reportprompt.RigorProfile{Level: "balanced", Label: "균형형"},
		ReportSessionPolicy:          reportexecution.SessionPolicyFreshSession,
		ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoFreshSession,
		PostReportHumanize:           "disabled", GenerationGuidanceProfile: reportprompt.ProfileNarrativeContract,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := executor.requests[0]
	if req.PreviousSessionID != "" || req.ReportPlan.PreviousProviderSessionID != "" {
		t.Fatalf("fresh plan must start without previous provider session: %#v", req)
	}
	if !reflect.DeepEqual(req.ExtraMCPTools, MCPTools()) || !req.ReplaceMCPTools {
		t.Fatalf("unexpected plan tools: %#v replace=%t", req.ExtraMCPTools, req.ReplaceMCPTools)
	}
	if service.query.PreviousProviderSessionID != "" {
		t.Fatalf("fresh plan selection query must bind empty previous session: %#v", service.query)
	}
	payload := eventPayload(t, out.Event)
	if payload["previous_agent_session_id"] != "" || payload["report_plan_session_id"] != "plan-session-1" {
		t.Fatalf("canonical plan payload lost fresh lineage: %#v", payload)
	}
	if payload["session_chain_kind"] != "fresh_session_report" || payload["pre_report_research_session_id"] != "research-session-1" {
		t.Fatalf("canonical plan payload lost policy metadata: %#v", payload)
	}
}

type fakePlanExecutor struct {
	requests []agentexec.AgentRequest
	results  []agentexec.AgentResult
}

func (fake *fakePlanExecutor) Run(_ context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
	fake.requests = append(fake.requests, req)
	result := fake.results[0]
	fake.results = fake.results[1:]
	return result, nil
}

type fakePlanService struct {
	events    []ledger.Event
	selection reporting.ReportPlanSubmissionSelection
	query     reporting.ReportPlanSubmissionQuery
	promote   reporting.PromoteReportPlanRequest
}

func (fake *fakePlanService) ListEvents(_ context.Context, _ string) ([]ledger.Event, error) {
	return fake.events, nil
}

func (fake *fakePlanService) AppendEvent(_ context.Context, req ledger.AppendRequest) (ledger.Event, error) {
	return ledger.Event{EventID: req.EventID, MissionID: req.MissionID, EventType: req.EventType, Producer: req.Producer, Payload: req.Payload}, nil
}

func (fake *fakePlanService) AppendEvents(_ context.Context, _ string, reqs []ledger.AppendRequest) ([]ledger.Event, error) {
	events := make([]ledger.Event, 0, len(reqs))
	for _, req := range reqs {
		events = append(events, ledger.Event{EventID: req.EventID, MissionID: req.MissionID, EventType: req.EventType, Producer: req.Producer, Payload: req.Payload})
	}
	return events, nil
}

func (fake *fakePlanService) AppendReportTerminalIfOpen(_ context.Context, _ string, _ string, reqs []ledger.AppendRequest) ([]ledger.Event, bool, error) {
	events, err := fake.AppendEvents(context.Background(), "", reqs)
	return events, true, err
}

func (fake *fakePlanService) AppendEventsIfNoActiveAgentWork(_ context.Context, _ string, reqs []ledger.AppendRequest) ([]ledger.Event, error) {
	return fake.AppendEvents(context.Background(), "", reqs)
}

func (fake *fakePlanService) ListSourceSnapshotsWithState(context.Context, source.ListRequest) ([]source.Snapshot, error) {
	return nil, nil
}

func (fake *fakePlanService) SelectReportPlanSubmission(_ context.Context, query reporting.ReportPlanSubmissionQuery) (reporting.ReportPlanSubmissionSelection, error) {
	fake.query = query
	return fake.selection, nil
}

func (fake *fakePlanService) PromoteReportPlan(_ context.Context, req reporting.PromoteReportPlanRequest) (ledger.Event, error) {
	fake.promote = req
	return ledger.Event{
		EventID: req.Canonical.EventID, MissionID: req.Canonical.MissionID,
		EventType: req.Canonical.EventType, Producer: req.Canonical.Producer,
		Payload: req.Canonical.Payload, CreatedAt: time.Now(),
	}, nil
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func eventPayload(t *testing.T, event ledger.Event) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func sequenceID() func(string) string {
	counts := map[string]int{}
	return func(prefix string) string {
		counts[prefix]++
		return fmt.Sprintf("%s_%d", prefix, counts[prefix])
	}
}

package reportworkflow

import (
	"context"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestRunDraftObservesFinalStoreAfterOneTakeDraft(t *testing.T) {
	service := &workflowService{}
	executor := &workflowExecutor{results: []agentexec.AgentResult{{Text: "# Quick\n\nBody.", SessionID: "research-session-1"}}}
	observer := &workflowObserver{}
	runner := workflowRunner(service, executor, observer)

	out, err := runner.RunDraft(context.Background(), draftInput(reportexecution.ModeOneTake))
	if err != nil {
		t.Fatal(err)
	}
	if out.Artifact.ArtifactID == "" || out.Event.EventID == "" || out.Markdown != "# Quick\n\nBody." {
		t.Fatalf("unexpected one_take output: %#v", out)
	}
	assertObservedNodes(t, observer, []string{NodeDirectDraft, NodeDirectDraft, NodeFinalStore, NodeFinalStore})
}

func TestRunDraftObservesFinalStoreAfterPlannedDraft(t *testing.T) {
	service := &workflowService{
		events: []ledger.Event{{
			EventID: "evt_pending", MissionID: "mis_1", EventType: "report.draft.pending",
			Payload: mustWorkflowJSON(map[string]any{"origin_pending_event_id": "evt_pending", "retry_strategy": "initial"}),
		}},
		selection: reporting.ReportPlanSubmissionSelection{
			EventID: "evt_submitted", ArgumentsHash: "args", PlanHash: "plan_hash",
			Plan: mustWorkflowJSON(reporting.ReportPlan{Summary: "Plan", Sections: []reporting.ReportPlanSection{{Title: "Section"}}}),
		},
	}
	executor := &workflowExecutor{results: []agentexec.AgentResult{
		{Text: reporting.ReportPlanSubmittedSentinel, SessionID: "plan-session-1"},
		{Text: "# Planned\n\nBody.", SessionID: "plan-session-1"},
	}}
	observer := &workflowObserver{}
	runner := workflowRunner(service, executor, observer)

	out, err := runner.RunDraft(context.Background(), draftInput(reportexecution.ModePlanned))
	if err != nil {
		t.Fatal(err)
	}
	if out.Artifact.ArtifactID == "" || out.Event.EventID == "" || out.ReportSessionID != "plan-session-1" {
		t.Fatalf("unexpected planned output: %#v", out)
	}
	assertObservedNodes(t, observer, []string{NodePlan, NodePlan, NodeDirectDraft, NodeDirectDraft, NodeFinalStore, NodeFinalStore})
	if service.atomicCalls != 1 || len(service.appended) != 0 {
		t.Fatalf("planned runtime must keep atomic finalstore write: atomic=%d appended=%d", service.atomicCalls, len(service.appended))
	}
}

func TestRunDraftSkipsFinalStoreWhenProviderFails(t *testing.T) {
	service := &workflowService{}
	executor := &workflowExecutor{
		results: []agentexec.AgentResult{{SessionID: "research-session-1"}},
		errs:    []error{workflowErr("provider failed")},
	}
	observer := &workflowObserver{}
	runner := workflowRunner(service, executor, observer)

	_, err := runner.RunDraft(context.Background(), draftInput(reportexecution.ModeOneTake))
	if err == nil {
		t.Fatal("expected provider failure")
	}
	assertObservedNodes(t, observer, []string{NodeDirectDraft, NodeDirectDraft})
	if !observer.observations[1].Failed {
		t.Fatalf("directdraft failure was not observed: %#v", observer.observations)
	}
}

func TestRunDraftObservesFinalStoreFailureAfterDirectDraftSuccess(t *testing.T) {
	service := &workflowService{createErr: workflowErr("create failed")}
	executor := &workflowExecutor{results: []agentexec.AgentResult{{Text: "# Quick\n\nBody.", SessionID: "research-session-1"}}}
	observer := &workflowObserver{}
	runner := workflowRunner(service, executor, observer)

	_, err := runner.RunDraft(context.Background(), draftInput(reportexecution.ModeOneTake))
	if err == nil {
		t.Fatal("expected storage failure")
	}
	assertObservedNodes(t, observer, []string{NodeDirectDraft, NodeDirectDraft, NodeFinalStore, NodeFinalStore})
	if observer.observations[1].Failed || !observer.observations[3].Failed {
		t.Fatalf("storage failure must be attributed to finalstore: %#v", observer.observations)
	}
}

func TestNewRunnerRejectsNilService(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewRunner must reject nil Service immediately")
		}
	}()
	_ = NewRunner(RunnerConfig{})
}

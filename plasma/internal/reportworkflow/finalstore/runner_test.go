package finalstore

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

func TestCommitOneTakeUsesAtomicTraceAndPayload(t *testing.T) {
	service := &fakeService{}
	ids := &idRecorder{}
	out, err := (Runner{Service: service, NewID: ids.next}).CommitOneTake(context.Background(), OneTakeInput{Base: baseInput(), Candidate: oneTakeCandidate()})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(service.calls, []string{"atomic"}) || service.atomicCalls != 1 ||
		len(service.created) != 0 || len(service.appended) != 0 {
		t.Fatalf("one_take storage trace changed: calls=%v atomic=%d", service.calls, service.atomicCalls)
	}
	if !reflect.DeepEqual(ids.order, []string{"evt"}) || out.Event.EventID != "evt_1" {
		t.Fatalf("one_take event id allocation changed: ids=%v event=%#v", ids.order, out.Event)
	}
	if out.Artifact.Filename != "report.md" || out.Artifact.SHA256 != sha("# Quick\n\nBody.") ||
		out.Artifact.Producer != (ledger.Producer{Type: "agent_session", ID: "research-session-1"}) {
		t.Fatalf("one_take artifact parity changed: %#v", out.Artifact)
	}
	payload := eventPayload(t, out.Event)
	if payload["composition_strategy"] != "one_take_markdown" ||
		payload["previous_agent_session_id"] != "research-session-1" ||
		payload["plan_review_state"] != "not_applicable" {
		t.Fatalf("one_take payload parity changed: %#v", payload)
	}
}

func TestCommitOneTakeAtomicFailureLeavesNoStandaloneWrites(t *testing.T) {
	service := &fakeService{atomicErr: errors.New("atomic failed")}
	ids := &idRecorder{}
	_, err := (Runner{Service: service, NewID: ids.next}).CommitOneTake(context.Background(), OneTakeInput{Base: baseInput(), Candidate: oneTakeCandidate()})
	if err == nil || !reflect.DeepEqual(service.calls, []string{"atomic"}) ||
		len(service.created) != 0 || len(service.appended) != 0 || !reflect.DeepEqual(ids.order, []string{"evt"}) {
		t.Fatalf("atomic failure semantics changed: err=%v calls=%v created=%d appended=%d ids=%v",
			err, service.calls, len(service.created), len(service.appended), ids.order)
	}
}

func TestCommitPlannedPreservesAtomicTraceAndPayload(t *testing.T) {
	service := &fakeService{}
	ids := &idRecorder{}
	out, err := (Runner{Service: service, NewID: ids.next}).CommitPlanned(context.Background(), PlannedInput{Base: baseInput(), Candidate: plannedCandidate()})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(service.calls, []string{"atomic"}) || service.atomicCalls != 1 ||
		len(service.created) != 0 || len(service.appended) != 0 {
		t.Fatalf("planned storage trace changed: calls=%v atomic=%d created=%d appended=%d", service.calls, service.atomicCalls, len(service.created), len(service.appended))
	}
	if out.Artifact.ArtifactID != "art_plan" || !reflect.DeepEqual(ids.order, []string{"evt"}) {
		t.Fatalf("planned artifact id or event id allocation changed: out=%#v ids=%v", out, ids.order)
	}
	payload := eventPayload(t, out.Event)
	if payload["plan_event_id"] != "evt_plan" || payload["plan_tool_session_id"] != "ses_plan" ||
		payload["previous_agent_session_id"] != "plan-session-1" || payload["session_chain_kind"] != "fresh_session_report" {
		t.Fatalf("planned payload parity changed: %#v", payload)
	}
}

func TestAdoptGateReReadsDurableResultAndRejectsBadLineageWithoutWrites(t *testing.T) {
	okRecord := gateRecord()
	service := &fakeService{}
	out, err := (Runner{Service: service, GateReader: &fakeGateReader{record: okRecord}}).AdoptGate(context.Background(), gateInput())
	if err != nil {
		t.Fatal(err)
	}
	if out.Artifact.ArtifactID != "art_gate" || len(service.calls) != 0 {
		t.Fatalf("AdoptGate must adopt durable result without writes: out=%#v calls=%v", out, service.calls)
	}
	for _, tc := range []struct {
		name   string
		record GateRecord
	}{
		{name: "foreign", record: func() GateRecord { record := gateRecord(); record.Artifact.MissionID = "mis_other"; return record }()},
		{name: "tampered", record: func() GateRecord { record := gateRecord(); record.Artifact.SHA256 = sha("tampered"); return record }()},
		{name: "duplicate", record: func() GateRecord { record := gateRecord(); record.CanonicalLineageCount = 2; return record }()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := (Runner{GateReader: &fakeGateReader{record: tc.record}}).AdoptGate(context.Background(), gateInput())
			if !errors.Is(err, producterror.ErrConflict) {
				t.Fatalf("expected conflict for %s gate record, got %v", tc.name, err)
			}
		})
	}
}

func baseInput() BaseInput {
	return BaseInput{
		MissionID: "mis_1", PendingEventID: "evt_pending", Title: "Report",
		AgentExecutor: "codex", AgentModel: "gpt-test", ReasoningEffort: "medium",
		MCPMode: "auto", Rigor: reportprompt.RigorProfile{Level: "balanced", Label: "균형형"},
		SessionPolicy: reportexecution.SessionPolicyFreshSession, PolicySelection: reportexecution.SessionPolicySelectionAutoFreshSession,
		PostHumanize: "disabled", GuidanceProfile: reportprompt.ProfileNarrativeContract,
	}
}

func oneTakeCandidate() OneTakeCandidate {
	return OneTakeCandidate{
		ArtifactID: "art_1", ToolSessionID: "ses_1", PreviousSessionID: "research-session-1",
		ReturnedSessionID: "research-session-1", ReportSessionID: "research-session-1",
		ReportSessionPolicy: reportexecution.SessionPolicyFreshSession,
		Markdown:            "# Quick\n\nBody.", StartedAt: time.Now().Add(-time.Second),
	}
}

func plannedCandidate() PlannedCandidate {
	return PlannedCandidate{
		ArtifactID: "art_plan", ToolSessionID: "ses_1", PlanEventID: "evt_plan",
		PlanToolSessionID: "ses_plan", ReportPlanSessionID: "plan-session-1",
		SessionChainKind: "fresh_session_report", PreReportResearchSessionID: "research-session-1",
		ReturnedSessionID: "plan-session-1", ReportSessionID: "plan-session-1",
		Markdown: "# Planned\n\nBody.", WorkflowStartedAt: time.Now().Add(-time.Second),
	}
}

func gateInput() GateInput {
	return GateInput{GateReadRequest: GateReadRequest{MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ArtifactID: "art_gate"}}
}

func gateRecord() GateRecord {
	markdown := "# Gate\n\nBody."
	return GateRecord{
		Artifact: artifact.Raw{ArtifactID: "art_gate", MissionID: "mis_1", MediaType: markdownMediaType, SHA256: sha(markdown), Content: []byte(markdown)},
		Event:    ledger.Event{EventID: "evt_gate", MissionID: "mis_1", EventType: "report.artifact.created", Payload: mustJSON(map[string]any{"pending_event_id": "evt_pending", "plan_event_id": "evt_plan", "artifact_id": "art_gate"})},
		Markdown: markdown, ReportSessionID: "gate-session", CanonicalLineageCount: 1,
	}
}

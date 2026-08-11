package sectiondraft

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

func TestPromptIncludesEvidenceGapRulesAndExactControlToken(t *testing.T) {
	prompt := Prompt(sectionDraftTestInput(2))
	for _, want := range []string{
		EvidenceGapControlToken,
		"Section evidence attempt: 2 of 2",
		"The evidence threshold is unchanged",
		"metadata-only",
		"main explanatory job promised by this Section's title and purpose",
		"supporting jobs unless source criticism, bibliography, transmission, or holdings history is explicitly and primarily this Section's subject",
		"Evidence that supports only supporting jobs is not enough",
		"A Section title that names a document does not by itself make source criticism the subject",
		"do not replace that job with supporting source commentary",
		"Source, document, report, and material must not become recurring grammatical subjects",
	} {
		if !contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestRunMarkdownCreatesOneArtifactAndSectionEvent(t *testing.T) {
	service := &sectionDraftRunService{}
	runner := Runner{Service: service, Executor: sectionDraftExecutor{result: agentexec.AgentResult{
		Text: "# Section\n\nSubstantive body.", SessionID: "provider-plan",
	}}, NewID: sectionDraftID}

	out, err := runner.Run(context.Background(), sectionDraftTestInput(1))
	if err != nil {
		t.Fatal(err)
	}
	if out.EvidenceGap != nil || out.Draft.ArtifactID == "" {
		t.Fatalf("unexpected output: %#v", out)
	}
	if len(service.artifacts) != 1 {
		t.Fatalf("artifacts = %d, want 1", len(service.artifacts))
	}
	if countSectionDraftEvents(service.events, CreatedEventType) != 1 || countSectionDraftEvents(service.events, EvidenceGapEventType) != 0 {
		t.Fatalf("unexpected events: %#v", service.events)
	}
}

func TestRunEvidenceGapCreatesNoArtifactOrCreatedEventAndRecordsAttempt(t *testing.T) {
	usage := agentusage.New("codex", "codex", "gpt-test", "medium", "prompt").WithProviderUsage(agentusage.ProviderUsage{InputTokens: 3, OutputTokens: 2}, "test")
	service := &sectionDraftRunService{}
	runner := Runner{Service: service, Executor: sectionDraftExecutor{result: agentexec.AgentResult{
		Text: EvidenceGapControlToken, SessionID: "provider-plan", Usage: usage, Resumed: true,
	}}, NewID: sectionDraftID}
	input := sectionDraftTestInput(1)

	out, err := runner.Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if out.EvidenceGap == nil || out.EvidenceGap.Attempt != 1 || out.EvidenceGap.ReasonCode != EvidenceGapReasonCode {
		t.Fatalf("unexpected gap output: %#v", out)
	}
	if len(service.artifacts) != 0 {
		t.Fatalf("artifacts = %d, want 0", len(service.artifacts))
	}
	if countSectionDraftEvents(service.events, CreatedEventType) != 0 || countSectionDraftEvents(service.events, EvidenceGapEventType) != 1 {
		t.Fatalf("unexpected events: %#v", service.events)
	}
	payload := sectionDraftPayload(t, service.events[0])
	for _, unexpected := range []string{"artifact_id", "media_type", "title", "text"} {
		if _, ok := payload[unexpected]; ok {
			t.Fatalf("gap payload included %q: %#v", unexpected, payload)
		}
	}
	expected := map[string]any{
		"pending_event_id":          "evt_pending",
		"plan_event_id":             "evt_plan",
		"part_index":                float64(1),
		"section_index":             float64(1),
		"attempt_number":            float64(1),
		"reason_code":               EvidenceGapReasonCode,
		"agent_executor":            "codex",
		"agent_session_id":          "provider-plan",
		"previous_agent_session_id": "provider-plan",
		"returned_agent_session_id": "provider-plan",
		"tool_session_id":           "tool-section",
		"session_chain_kind":        "fresh_session_report",
		"report_plan_session_id":    "provider-plan",
		"report_session_id":         "provider-plan",
	}
	for key, want := range expected {
		if got := payload[key]; got != want {
			t.Fatalf("payload[%s] = %#v, want %#v in %#v", key, got, want, payload)
		}
	}
	if _, ok := payload["agent_usage"]; !ok {
		t.Fatalf("gap payload missing agent_usage: %#v", payload)
	}
}

func TestRunRejectsAttemptAboveMaximumBeforeProviderEventOrArtifact(t *testing.T) {
	service := &sectionDraftRunService{}
	executor := &countingSectionDraftExecutor{}
	runner := Runner{Service: service, Executor: executor, NewID: sectionDraftID}
	input := sectionDraftTestInput(MaxEvidenceGapAttempts + 1)
	input.StartedEvent = true

	_, err := runner.Run(context.Background(), input)
	var stage *reportexecution.StageFailureError
	if err == nil || !errors.As(err, &stage) || stage.Kind != "section" || !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("err = %v, stage = %#v, want section invalid input", err, stage)
	}
	if executor.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", executor.calls)
	}
	if len(service.events) != 0 || len(service.artifacts) != 0 {
		t.Fatalf("events=%d artifacts=%d, want none", len(service.events), len(service.artifacts))
	}
}

type sectionDraftExecutor struct {
	result agentexec.AgentResult
}

func (executor sectionDraftExecutor) Run(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
	return executor.result, nil
}

type countingSectionDraftExecutor struct {
	calls int
}

func (executor *countingSectionDraftExecutor) Run(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
	executor.calls++
	return agentexec.AgentResult{Text: "# Should not run", SessionID: "provider-plan"}, nil
}

type sectionDraftRunService struct {
	events    []ledger.Event
	artifacts []artifact.Raw
}

func (service *sectionDraftRunService) AppendEvent(_ context.Context, req ledger.AppendRequest) (ledger.Event, error) {
	event := ledger.Event{EventID: req.EventID, MissionID: req.MissionID, EventType: req.EventType, Producer: req.Producer, Payload: req.Payload}
	service.events = append(service.events, event)
	return event, nil
}

func (service *sectionDraftRunService) CreateRawArtifact(_ context.Context, req artifact.CreateRequest) (artifact.Raw, error) {
	raw := artifact.Raw{ArtifactID: req.ArtifactID, MissionID: req.MissionID, MediaType: req.MediaType, Filename: req.Filename, Producer: req.Producer, Content: append([]byte(nil), req.Content...)}
	service.artifacts = append(service.artifacts, raw)
	return raw, nil
}

func sectionDraftTestInput(attempt int) Input {
	plan := reporting.SectionalReportPlan{Parts: []reporting.ReportPlanPart{{Title: "Part", Sections: []reporting.ReportPlanSection{{Title: "Section", Purpose: "Explain the subject directly."}}}}}
	return Input{
		Base: BaseInput{
			MissionID: "mis_1", PendingEventID: "evt_pending", Title: "Report",
			AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "medium",
			MCPMode: "auto", Rigor: reportprompt.RigorProfile{Level: "balanced", Label: "균형형"},
			ReportSessionPolicy:          reportexecution.SessionPolicyFreshSession,
			ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoFreshSession,
			PostReportHumanize:           "disabled",
			Plan:                         plan,
			PlanEvent:                    ledger.Event{EventID: "evt_plan"},
			ReportPlanSessionID:          "provider-plan",
			SessionChainKind:             "fresh_session_report",
		},
		Part: plan.Parts[0], Section: plan.Parts[0].Sections[0], PartIndex: 0, SectionIndex: 0,
		Attempt: attempt, ToolSessionID: "tool-section", PreviousSessionID: "provider-plan",
		UserText: "draft section 1.1", CreatedText: "created",
	}
}

func countSectionDraftEvents(events []ledger.Event, eventType string) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func sectionDraftPayload(t *testing.T, event ledger.Event) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func sectionDraftID(prefix string) string { return prefix + "_test" }

func contains(value, needle string) bool {
	return strings.Contains(value, needle)
}

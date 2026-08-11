package reportworkflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

func workflowRunner(service *workflowService, executor agentexec.AgentExecutor, observer *workflowObserver) Runner {
	ids := workflowSequenceID()
	return NewRunner(RunnerConfig{
		Service: service,
		Lifecycle: reporting.Runner(reportexecution.Runner{
			Service: service,
			NewID:   ids,
		}),
		Executor: executor,
		NewID:    ids,
		LatestSessionID: func(context.Context, string, string) string {
			return "research-session-1"
		},
	}).WithObserver(observer)
}

func draftInput(mode string) DraftInput {
	return DraftInput{
		MissionID: "mis_1", PendingEventID: "evt_pending", Title: "Report",
		AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "medium",
		MCPMode: "auto", Rigor: reportprompt.RigorProfile{Level: "balanced", Label: "균형형"},
		ReportMode: mode, ReportSessionPolicy: reportexecution.SessionPolicyFreshSession,
		ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoFreshSession,
		PostReportHumanize:           "disabled", GenerationGuidanceProfile: reportprompt.ProfileNarrativeContract,
	}
}

type workflowObserver struct {
	observations []NodeObservation
}

func (observer *workflowObserver) Observe(observation NodeObservation) {
	observer.observations = append(observer.observations, observation)
}

func assertObservedNodes(t *testing.T, observer *workflowObserver, want []string) {
	t.Helper()
	got := make([]string, 0, len(observer.observations))
	for _, observation := range observer.observations {
		got = append(got, observation.NodeID)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("unexpected observed node order: got %v want %v", got, want)
	}
}

type workflowExecutor struct {
	requests []agentexec.AgentRequest
	results  []agentexec.AgentResult
	errs     []error
}

func (fake *workflowExecutor) Run(_ context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
	fake.requests = append(fake.requests, req)
	var result agentexec.AgentResult
	if len(fake.results) > 0 {
		result = fake.results[0]
		fake.results = fake.results[1:]
	}
	if len(fake.errs) > 0 {
		err := fake.errs[0]
		fake.errs = fake.errs[1:]
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

type workflowService struct {
	mu                   sync.Mutex
	events               []ledger.Event
	artifacts            []artifact.Raw
	selection            reporting.ReportPlanSubmissionSelection
	reqSelect            app.ReportRequirementMapSelection
	appended             []ledger.AppendRequest
	atomicCalls          int
	createErr            error
	appendErr            error
	validateRefsErr      error
	validatedRefsMission string
	validatedRefs        []reporting.ReportPlanSourceRefs
}

func (fake *workflowService) ListEvents(context.Context, string) ([]ledger.Event, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return append([]ledger.Event(nil), fake.events...), nil
}

func (fake *workflowService) ValidateReportPlanRefs(_ context.Context, missionID string, refs []reporting.ReportPlanSourceRefs) error {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.validatedRefsMission = missionID
	fake.validatedRefs = append([]reporting.ReportPlanSourceRefs(nil), refs...)
	return fake.validateRefsErr
}

func (fake *workflowService) SelectReportPlanSubmission(context.Context, reporting.ReportPlanSubmissionQuery) (reporting.ReportPlanSubmissionSelection, error) {
	return fake.selection, nil
}

func (fake *workflowService) SelectReportRequirementMap(context.Context, app.ReportRequirementMapQuery) (app.ReportRequirementMapSelection, error) {
	return fake.reqSelect, nil
}

func (fake *workflowService) PromoteReportPlan(_ context.Context, req reporting.PromoteReportPlanRequest) (ledger.Event, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	event := ledger.Event{
		EventID: req.Canonical.EventID, MissionID: req.Canonical.MissionID,
		EventType: req.Canonical.EventType, Producer: req.Canonical.Producer,
		Payload: req.Canonical.Payload, CreatedAt: time.Now(),
	}
	fake.events = append(fake.events, event)
	return event, nil
}

func (fake *workflowService) AppendEvent(_ context.Context, req ledger.AppendRequest) (ledger.Event, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.appended = append(fake.appended, req)
	if fake.appendErr != nil {
		return ledger.Event{}, fake.appendErr
	}
	event := ledger.Event{EventID: req.EventID, MissionID: req.MissionID, EventType: req.EventType, Producer: req.Producer, Payload: req.Payload, CreatedAt: time.Now()}
	fake.events = append(fake.events, event)
	return event, nil
}

func (fake *workflowService) AppendEvents(_ context.Context, _ string, reqs []ledger.AppendRequest) ([]ledger.Event, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	events := make([]ledger.Event, 0, len(reqs))
	for _, req := range reqs {
		events = append(events, ledger.Event{EventID: req.EventID, MissionID: req.MissionID, EventType: req.EventType, Producer: req.Producer, Payload: req.Payload, CreatedAt: time.Now()})
	}
	return events, nil
}

func (fake *workflowService) AppendReportTerminalIfOpen(_ context.Context, missionID string, _ string, reqs []ledger.AppendRequest) ([]ledger.Event, bool, error) {
	events, err := fake.AppendEvents(context.Background(), missionID, reqs)
	return events, true, err
}

func (fake *workflowService) AppendEventsIfNoActiveAgentWork(_ context.Context, missionID string, reqs []ledger.AppendRequest) ([]ledger.Event, error) {
	return fake.AppendEvents(context.Background(), missionID, reqs)
}

func (fake *workflowService) AppendEventConditionally(_ context.Context, missionID string, build func([]ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error)) (ledger.Event, bool, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	req, existing, create, err := build(fake.events)
	if err != nil || !create {
		return existing, false, err
	}
	event := ledger.Event{
		EventID: req.EventID, MissionID: missionID, EventType: req.EventType,
		Producer: req.Producer, CausationEventID: req.CausationEventID,
		CorrelationID: req.CorrelationID, Payload: req.Payload, CreatedAt: time.Now(),
	}
	fake.events = append(fake.events, event)
	return event, true, nil
}

func (fake *workflowService) ListSourceSnapshotsWithState(context.Context, source.ListRequest) ([]source.Snapshot, error) {
	return nil, nil
}

func (fake *workflowService) GetRawArtifact(_ context.Context, artifactID string) (artifact.Raw, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	for _, raw := range fake.artifacts {
		if raw.ArtifactID == artifactID {
			return raw, nil
		}
	}
	return artifact.Raw{}, workflowErr("artifact missing")
}

func (fake *workflowService) GetEvidenceRecord(context.Context, string) (app.EvidenceRecord, error) {
	return app.EvidenceRecord{}, workflowErr("evidence missing")
}

func (fake *workflowService) CreateRawArtifact(_ context.Context, req artifact.CreateRequest) (artifact.Raw, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.createErr != nil {
		return artifact.Raw{}, fake.createErr
	}
	raw := workflowRawArtifact(req)
	fake.artifacts = append(fake.artifacts, raw)
	return raw, nil
}

func (fake *workflowService) CreateRawArtifactWithEvent(_ context.Context, req artifact.CreateRequest, build func(artifact.Raw) ledger.AppendRequest) (artifact.Raw, ledger.Event, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.atomicCalls++
	raw := workflowRawArtifact(req)
	eventReq := build(raw)
	if fake.createErr != nil {
		return artifact.Raw{}, ledger.Event{}, fake.createErr
	}
	if fake.appendErr != nil {
		return artifact.Raw{}, ledger.Event{}, fake.appendErr
	}
	event := ledger.Event{EventID: eventReq.EventID, MissionID: eventReq.MissionID, EventType: eventReq.EventType, Producer: eventReq.Producer, Payload: eventReq.Payload, CreatedAt: time.Now()}
	fake.artifacts = append(fake.artifacts, raw)
	fake.events = append(fake.events, event)
	return raw, event, nil
}

func (fake *workflowService) CreateRawArtifactWithEventConditionally(_ context.Context, req artifact.CreateRequest, build func([]ledger.Event, artifact.Raw) (ledger.AppendRequest, ledger.Event, bool, error)) (artifact.Raw, ledger.Event, bool, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	raw := workflowRawArtifact(req)
	eventReq, existing, create, err := build(fake.events, raw)
	if err != nil || !create {
		return raw, existing, false, err
	}
	event := ledger.Event{EventID: eventReq.EventID, MissionID: eventReq.MissionID, EventType: eventReq.EventType, Producer: eventReq.Producer, Payload: eventReq.Payload, CreatedAt: time.Now()}
	fake.artifacts = append(fake.artifacts, raw)
	fake.events = append(fake.events, event)
	return raw, event, true, nil
}

func workflowRawArtifact(req artifact.CreateRequest) artifact.Raw {
	return artifact.Raw{
		ArtifactID: req.ArtifactID, MissionID: req.MissionID, MediaType: req.MediaType,
		ByteSize: int64(len(req.Content)), SHA256: workflowSHA(string(req.Content)),
		Filename: req.Filename, Producer: req.Producer, Content: append([]byte(nil), req.Content...),
	}
}

func mustWorkflowJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func workflowSequenceID() func(string) string {
	var mu sync.Mutex
	counts := map[string]int{}
	return func(prefix string) string {
		mu.Lock()
		defer mu.Unlock()
		counts[prefix]++
		return fmt.Sprintf("%s_%d", prefix, counts[prefix])
	}
}

func workflowSHA(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func workflowErr(message string) error { return errors.New(message) }

package reporting_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestAssembleLongFormFinalMarkdownMatchesLegacyFixture(t *testing.T) {
	got := reporting.AssembleLongFormFinalMarkdown("보고서", "---\n# 보고서\n\n### 안내\n본문\n---", "---\n# 결론\n끝\n---", []string{" # Part 1\n\n본문 1\n", "# Part 2\n\n본문 2"})
	want := "# 보고서\n\n## 안내\n본문\n\n---\n\n# Part 1\n\n본문 1\n\n# Part 2\n\n본문 2\n\n---\n\n## 결론\n끝\n"
	if got != want {
		t.Fatalf("assembled Markdown differs\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestFinalizeLongFormAtomicReplayAndConflict(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixture(t, ctx)
	defer closeStore()
	binding := longFormFinalizeBinding()
	req := reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_final", OpeningMarkdown: "# Report\n\nOpening", ClosingMarkdown: "## Closing\n\nDone"}

	const callers = 12
	results := make(chan reporting.LongFormFinalizeResult, callers)
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	for index := 0; index < callers; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := reporting.FinalizeLongForm(ctx, svc, req)
			results <- result
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent finalize: %v", err)
		}
	}
	for result := range results {
		if result.Artifact.ArtifactID != binding.ArtifactID || result.Event.EventID != "evt_final" {
			t.Fatalf("unexpected canonical result: %#v", result)
		}
	}
	events, err := svc.ListEvents(ctx, binding.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, event := range events {
		if event.EventType == "report.artifact.created" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("canonical event count=%d, want 1", count)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(resultsPayload(events, "report.artifact.created"), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["returned_agent_session_id"] != "" || payload["agent_usage"] != nil {
		t.Fatalf("canonical must not guess returned session or usage: %#v", payload)
	}

	_, err = reporting.FinalizeLongForm(ctx, svc, reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_other", OpeningMarkdown: "# Different", ClosingMarkdown: req.ClosingMarkdown})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("different content error=%v, want conflict", err)
	}
}

func TestFinalizeLongFormReplayRejectsEveryStoredSemanticBinding(t *testing.T) {
	mutations := map[string]func(*reporting.LongFormFinalizeBinding){
		"pending":            func(value *reporting.LongFormFinalizeBinding) { value.PendingEventID = "evt_other_pending" },
		"plan":               func(value *reporting.LongFormFinalizeBinding) { value.PlanEventID = "evt_other_plan" },
		"artifact":           func(value *reporting.LongFormFinalizeBinding) { value.ArtifactID = "art_other" },
		"filename":           func(value *reporting.LongFormFinalizeBinding) { value.Filename = "other.md" },
		"title":              func(value *reporting.LongFormFinalizeBinding) { value.Title = "Other" },
		"tool session":       func(value *reporting.LongFormFinalizeBinding) { value.ToolSessionID = "ses_other" },
		"plan tool session":  func(value *reporting.LongFormFinalizeBinding) { value.PlanToolSessionID = "ses_plan_other" },
		"idempotency":        func(value *reporting.LongFormFinalizeBinding) { value.IdempotencyKey = "other-key" },
		"executor":           func(value *reporting.LongFormFinalizeBinding) { value.AgentExecutor = "claude" },
		"model":              func(value *reporting.LongFormFinalizeBinding) { value.AgentModel = "other-model" },
		"effort":             func(value *reporting.LongFormFinalizeBinding) { value.AgentReasoningEffort = "low" },
		"selection":          func(value *reporting.LongFormFinalizeBinding) { value.AgentSelectionSource = "other" },
		"mcp":                func(value *reporting.LongFormFinalizeBinding) { value.MCPMode = "strict" },
		"rigor level":        func(value *reporting.LongFormFinalizeBinding) { value.RigorLevel = "other" },
		"rigor label":        func(value *reporting.LongFormFinalizeBinding) { value.RigorLabel = "other" },
		"session policy":     func(value *reporting.LongFormFinalizeBinding) { value.ReportSessionPolicy = "other" },
		"policy selection":   func(value *reporting.LongFormFinalizeBinding) { value.ReportSessionPolicySelection = "other" },
		"post humanize":      func(value *reporting.LongFormFinalizeBinding) { value.PostReportHumanize = "disabled" },
		"guidance profile":   func(value *reporting.LongFormFinalizeBinding) { value.GenerationGuidanceProfile = "other" },
		"guidance hash":      func(value *reporting.LongFormFinalizeBinding) { value.GenerationGuidanceSHA256 = "other" },
		"chain kind":         func(value *reporting.LongFormFinalizeBinding) { value.SessionChainKind = "other" },
		"pre-report session": func(value *reporting.LongFormFinalizeBinding) { value.PreReportResearchSessionID = "provider-other" },
		"plan session":       func(value *reporting.LongFormFinalizeBinding) { value.ReportPlanSessionID = "provider-other" },
		"fork session":       func(value *reporting.LongFormFinalizeBinding) { value.ForkSourceAgentSessionID = "provider-other" },
		"section word count": func(value *reporting.LongFormFinalizeBinding) { value.SectionWordCount++ },
		"part order":         func(value *reporting.LongFormFinalizeBinding) { value.PartArtifactIDs = []string{"art_other_part"} },
		"section order": func(value *reporting.LongFormFinalizeBinding) {
			value.SectionArtifactIDs = []string{"art_other_section"}
		},
		"previous provider session": func(value *reporting.LongFormFinalizeBinding) { value.PreviousProviderSessionID = "provider-before" },
		"provider session": func(value *reporting.LongFormFinalizeBinding) {
			value.ProviderSessionID = "provider-other"
			value.Producer.ID = "provider-other"
		},
	}
	for name, mutate := range mutations {
		mutate := mutate
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			svc, closeStore := newLongFormFinalizeFixture(t, ctx)
			defer closeStore()
			binding := longFormFinalizeBinding()
			request := reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_final", OpeningMarkdown: "# Open", ClosingMarkdown: "## Close"}
			if _, err := reporting.FinalizeLongForm(ctx, svc, request); err != nil {
				t.Fatal(err)
			}
			mutate(&binding)
			request.Binding = binding
			if _, err := reporting.FinalizeLongForm(ctx, svc, request); !errors.Is(err, app.ErrConflict) {
				t.Fatalf("replay error=%v, want conflict", err)
			}
		})
	}
}

func TestFinalizeLongFormReplayAfterSQLiteRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "plasma.db")
	store, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	svc := app.NewService(store)
	seedLongFormFinalizeFixture(t, ctx, svc)
	binding := longFormFinalizeBinding()
	req := reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_final", OpeningMarkdown: "# Open", ClosingMarkdown: "## Close"}
	if _, err := reporting.FinalizeLongForm(ctx, svc, req); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = sqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	replayed, err := reporting.FinalizeLongForm(ctx, app.NewService(store), req)
	if err != nil || !replayed.Replay || replayed.Event.EventID != "evt_final" {
		t.Fatalf("restart replay=%#v err=%v", replayed, err)
	}
}

func TestFinalizeLongFormRejectsDuplicateAndOutOfRangeStageLineage(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request app.AppendEventRequest
	}{
		{name: "duplicate part", request: app.AppendEventRequest{EventID: "evt_part_duplicate", EventType: "report.part.created", Payload: testJSON(map[string]any{"pending_event_id": "evt_pending", "plan_event_id": "evt_plan", "artifact_id": "art_part", "part_index": 1})}},
		{name: "out of range section", request: app.AppendEventRequest{EventID: "evt_section_out_of_range", EventType: "report.section.created", Payload: testJSON(map[string]any{"pending_event_id": "evt_pending", "plan_event_id": "evt_plan", "artifact_id": "art_section", "part_index": 2, "section_index": 1})}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			svc, closeStore := newLongFormFinalizeFixture(t, ctx)
			defer closeStore()
			binding := longFormFinalizeBinding()
			tc.request.MissionID = binding.MissionID
			tc.request.Producer = binding.Producer
			if _, err := svc.AppendEvent(ctx, tc.request); err != nil {
				t.Fatal(err)
			}
			_, err := reporting.FinalizeLongForm(ctx, svc, reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_final", OpeningMarkdown: "# Open", ClosingMarkdown: "## Close"})
			if !errors.Is(err, app.ErrConflict) {
				t.Fatalf("lineage error=%v, want conflict", err)
			}
		})
	}
}

func TestFinalizeLongFormAndFailureClosureAreMutuallyExclusive(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixture(t, ctx)
	defer closeStore()
	binding := longFormFinalizeBinding()
	start := make(chan struct{})
	finalErr := make(chan error, 1)
	failureResult := make(chan struct {
		closed bool
		err    error
	}, 1)
	go func() {
		<-start
		_, err := reporting.FinalizeLongForm(ctx, svc, reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_final", OpeningMarkdown: "# Open", ClosingMarkdown: "## Close"})
		finalErr <- err
	}()
	go func() {
		<-start
		_, closed, err := svc.AppendReportTerminalIfOpen(ctx, binding.MissionID, binding.PendingEventID, []app.AppendEventRequest{{
			EventID: "evt_failed", MissionID: binding.MissionID, EventType: "report.draft.failed", Producer: app.Producer{Type: "agent", ID: "codex"},
			Payload: testJSON(map[string]any{"kind": "worker_failed", "pending_event_id": binding.PendingEventID}),
		}})
		failureResult <- struct {
			closed bool
			err    error
		}{closed: closed, err: err}
	}()
	close(start)
	finalizeErr, failed := <-finalErr, <-failureResult
	if failed.err != nil {
		t.Fatal(failed.err)
	}
	if (finalizeErr == nil) == failed.closed {
		t.Fatalf("exactly one terminal must win: finalize_err=%v failure_closed=%t", finalizeErr, failed.closed)
	}
	events, err := svc.ListEvents(ctx, binding.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	if countEventType(events, "report.artifact.created")+countEventType(events, "report.draft.failed") != 1 {
		t.Fatalf("contradictory terminal ledger: %#v", events)
	}
}

func TestFinalizeLongFormRollsBackArtifactWhenCanonicalEventIsInvalid(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixture(t, ctx)
	defer closeStore()
	binding := longFormFinalizeBinding()
	_, err := reporting.FinalizeLongForm(ctx, svc, reporting.LongFormFinalizeRequest{Binding: binding, OpeningMarkdown: "# Open", ClosingMarkdown: "## Close"})
	if !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("invalid canonical event error=%v, want invalid input", err)
	}
	if _, err := svc.GetRawArtifact(ctx, binding.ArtifactID); err == nil {
		t.Fatal("final artifact survived rolled-back canonical event")
	}
	events, err := svc.ListEvents(ctx, binding.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	if countEventType(events, "report.artifact.created") != 0 {
		t.Fatalf("invalid canonical event was persisted: %#v", events)
	}
}

func newLongFormFinalizeFixture(t *testing.T, ctx context.Context) (*app.Service, func()) {
	t.Helper()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	svc := app.NewService(store)
	seedLongFormFinalizeFixture(t, ctx, svc)
	return svc, func() { _ = store.Close() }
}

func seedLongFormFinalizeFixture(t *testing.T, ctx context.Context, svc *app.Service) {
	t.Helper()
	binding := longFormFinalizeBinding()
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: binding.MissionID, Title: "finalize"}); err != nil {
		t.Fatal(err)
	}
	part, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{ArtifactID: binding.PartArtifactIDs[0], MissionID: binding.MissionID, MediaType: "text/markdown; charset=utf-8", Filename: "part.md", Producer: binding.Producer, Content: []byte("# Part 1\n\nPreserved body.\n")})
	if err != nil {
		t.Fatal(err)
	}
	requests := []app.AppendEventRequest{
		{EventID: binding.PendingEventID, MissionID: binding.MissionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: testJSON(map[string]any{"report_mode": "long_form"})},
		{EventID: binding.PlanEventID, MissionID: binding.MissionID, EventType: "report.plan.created", Producer: binding.Producer, Payload: testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "report_mode": "long_form", "artifact_id": binding.ArtifactID})},
		{EventID: "evt_part", MissionID: binding.MissionID, EventType: "report.part.created", Producer: binding.Producer, Payload: testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": part.ArtifactID, "part_index": 1})},
		{EventID: "evt_section", MissionID: binding.MissionID, EventType: "report.section.created", Producer: binding.Producer, Payload: testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": binding.SectionArtifactIDs[0], "part_index": 1, "section_index": 1})},
	}
	for _, request := range requests {
		if _, err := svc.AppendEvent(ctx, request); err != nil {
			t.Fatal(err)
		}
	}
}

func longFormFinalizeBinding() reporting.LongFormFinalizeBinding {
	return reporting.LongFormFinalizeBinding{
		MissionID: "mis_finalize", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ArtifactID: "art_final", Filename: "report.md", Title: "Report",
		ToolSessionID: "ses_tool", IdempotencyKey: "final-key", ProviderSessionID: "provider-session", PreviousProviderSessionID: "provider-session",
		PartArtifactIDs: []string{"art_part"}, SectionArtifactIDs: []string{"art_section"}, SectionWordCount: 3,
		AgentExecutor: "codex", AgentModel: "model", AgentReasoningEffort: "high", AgentSelectionSource: "request", MCPMode: "auto",
		RigorLevel: "standard", RigorLabel: "Standard", ReportSessionPolicy: "same_session", ReportSessionPolicySelection: "default",
		PostReportHumanize: "h5", GenerationGuidanceProfile: "default", GenerationGuidanceSHA256: "guidance-sha",
		SessionChainKind: "same_session_report", PreReportResearchSessionID: "provider-research", ReportPlanSessionID: "provider-session",
		ForkSourceAgentSessionID: "", PlanToolSessionID: "ses_plan", Producer: app.Producer{Type: "agent_session", ID: "provider-session"},
	}
}

func resultsPayload(events []app.LedgerEvent, eventType string) json.RawMessage {
	for _, event := range events {
		if event.EventType == eventType {
			return event.Payload
		}
	}
	return nil
}

func countEventType(events []app.LedgerEvent, eventType string) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func testJSON(value any) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}

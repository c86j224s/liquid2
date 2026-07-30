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

const partEditSourceMarkdown = "# Part 1\n\nSource body.\n"

func TestFinalizePartEditRecordsNoOpAndChangedOutcomes(t *testing.T) {
	t.Run("no-op reuses source artifact", func(t *testing.T) {
		ctx := context.Background()
		svc, closeStore, binding := newPartEditFixture(t, ctx)
		defer closeStore()
		startPartEdit(t, ctx, svc, binding)

		result, err := reporting.FinalizePartEdit(ctx, svc, binding, "evt_part_edit", partEditSourceMarkdown, 0)
		if err != nil {
			t.Fatal(err)
		}
		if result.Artifact.ArtifactID != binding.SourceArtifactID || result.Replay {
			t.Fatalf("no-op result should reuse source without replay: %#v", result)
		}
		if _, err := svc.GetRawArtifact(ctx, binding.EditedArtifactID); err == nil {
			t.Fatal("no-op Part edit created a separate edited artifact")
		}
		payload := partEditedPayload(t, result.Event)
		if payload["artifact_id"] != binding.SourceArtifactID || payload["changed"] != false {
			t.Fatalf("no-op payload differs: %#v", payload)
		}
		replayed, err := reporting.FinalizePartEdit(ctx, svc, binding, "evt_part_edit_other", partEditSourceMarkdown, 0)
		if err != nil || !replayed.Replay || replayed.Event.EventID != result.Event.EventID {
			t.Fatalf("no-op replay=%#v err=%v", replayed, err)
		}
	})

	t.Run("changed edit creates separate artifact", func(t *testing.T) {
		ctx := context.Background()
		svc, closeStore, binding := newPartEditFixture(t, ctx)
		defer closeStore()
		startPartEdit(t, ctx, svc, binding)

		editedMarkdown := "# Part 1\n\nEdited body.\n"
		result, err := reporting.FinalizePartEdit(ctx, svc, binding, "evt_part_edit", editedMarkdown, 1)
		if err != nil {
			t.Fatal(err)
		}
		if result.Artifact.ArtifactID != binding.EditedArtifactID || string(result.Artifact.Content) != editedMarkdown {
			t.Fatalf("changed result did not use edited artifact: %#v", result)
		}
		payload := partEditedPayload(t, result.Event)
		if payload["artifact_id"] != binding.EditedArtifactID || payload["changed"] != true {
			t.Fatalf("changed payload differs: %#v", payload)
		}
		source, err := svc.GetRawArtifact(ctx, binding.SourceArtifactID)
		if err != nil {
			t.Fatal(err)
		}
		if string(source.Content) != partEditSourceMarkdown {
			t.Fatalf("source artifact was mutated: %q", source.Content)
		}
	})
}

func TestStartPartEditIsIdempotentAndRejectsConflictingBinding(t *testing.T) {
	ctx := context.Background()
	svc, closeStore, binding := newPartEditFixture(t, ctx)
	defer closeStore()

	started, created, err := reporting.StartPartEdit(ctx, svc, "evt_part_edit_started", binding)
	if err != nil || !created {
		t.Fatalf("start created=%t event=%#v err=%v", created, started, err)
	}
	replayed, created, err := reporting.StartPartEdit(ctx, svc, "evt_part_edit_started_replay", binding)
	if err != nil || created || replayed.EventID != started.EventID {
		t.Fatalf("start replay created=%t event=%#v err=%v", created, replayed, err)
	}

	conflict := binding
	conflict.ToolSessionID = "ses_conflicting_part_edit"
	_, _, err = reporting.StartPartEdit(ctx, svc, "evt_part_edit_started_conflict", conflict)
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("conflicting same-key start error=%v, want conflict", err)
	}
	events, err := svc.ListEvents(ctx, binding.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	if countEventType(events, reporting.PartEditStartedEventType) != 1 {
		t.Fatalf("idempotent start appended duplicates: %#v", events)
	}
}

func TestFinalizePartEditRequiresMatchingStart(t *testing.T) {
	ctx := context.Background()
	svc, closeStore, binding := newPartEditFixture(t, ctx)
	defer closeStore()

	if _, err := reporting.FinalizePartEdit(ctx, svc, binding, "evt_part_edit_without_start", "# Part 1\n\nEdited body.\n", 1); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("FinalizePartEdit without start error=%v, want conflict", err)
	}
	startPartEdit(t, ctx, svc, binding)
	conflict := binding
	conflict.ToolSessionID = "ses_conflicting_part_edit"
	if _, err := reporting.FinalizePartEdit(ctx, svc, conflict, "evt_part_edit_wrong_start", "# Part 1\n\nEdited body.\n", 1); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("FinalizePartEdit with mismatched start error=%v, want conflict", err)
	}
}

func TestFinalizePartEditReplaysCanonicalWinnerAcrossChangedNoOpRace(t *testing.T) {
	ctx := context.Background()
	svc, closeStore, binding := newPartEditFixture(t, ctx)
	defer closeStore()
	startPartEdit(t, ctx, svc, binding)

	start := make(chan struct{})
	results := make(chan reporting.PartEditResult, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, call := range []struct {
		binding   reporting.PartEditBinding
		eventID   string
		markdown  string
		operation int
	}{
		{binding: binding, eventID: "evt_part_edit_changed", markdown: "# Part 1\n\nEdited body.\n", operation: 1},
		{binding: binding, eventID: "evt_part_edit_noop", markdown: partEditSourceMarkdown, operation: 0},
	} {
		call := call
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := reporting.FinalizePartEdit(ctx, svc, call.binding, call.eventID, call.markdown, call.operation)
			results <- result
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Part edit failed: %v", err)
		}
	}
	events, err := svc.ListEvents(ctx, binding.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	if countEventType(events, reporting.PartEditedEventType) != 1 {
		t.Fatalf("canonical Part edit event count differs: %#v", events)
	}
	var canonical app.LedgerEvent
	for _, event := range events {
		if event.EventType == reporting.PartEditedEventType {
			canonical = event
		}
	}
	canonicalArtifactID := partEditedPayload(t, canonical)["artifact_id"]
	for result := range results {
		if result.Event.EventID != canonical.EventID || result.Artifact.ArtifactID != canonicalArtifactID {
			t.Fatalf("losing race did not replay canonical winner: result=%#v canonical=%#v", result, canonical)
		}
	}
}

func TestFinalizePartEditReusesAncestorPartOnlyForResumeFailed(t *testing.T) {
	ctx := context.Background()
	svc, closeStore, binding := newPartEditFixture(t, ctx)
	defer closeStore()

	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_root_failed", MissionID: binding.MissionID, EventType: "report.draft.failed",
		Producer: app.Producer{Type: "agent", ID: "codex"},
		Payload:  testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "kind": "report_draft_failed"}),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_pending_resume", MissionID: binding.MissionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"},
		Payload: testJSON(map[string]any{
			"report_mode": "long_form", "origin_pending_event_id": binding.PendingEventID,
			"retry_of_pending_event_id": binding.PendingEventID, "retry_strategy": "resume_failed", "attempt_number": 2,
		}),
	}); err != nil {
		t.Fatal(err)
	}
	resume := binding
	resume.PendingEventID = "evt_pending_resume"
	resume.EditedArtifactID = "art_part_edit_resume"
	resume.IdempotencyKey = "report-part-edit:evt_pending_resume:evt_plan:1"
	startPartEdit(t, ctx, svc, resume)
	if _, err := reporting.FinalizePartEdit(ctx, svc, resume, "evt_part_edit_resume", "# Part 1\n\nResume edit.\n", 1); err != nil {
		t.Fatalf("resume_failed should reuse ancestor Part: %v", err)
	}

	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_pending_restart", MissionID: binding.MissionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"},
		Payload: testJSON(map[string]any{
			"report_mode": "long_form", "origin_pending_event_id": binding.PendingEventID,
			"retry_of_pending_event_id": binding.PendingEventID, "retry_strategy": "restart", "attempt_number": 2,
		}),
	}); err != nil {
		t.Fatal(err)
	}
	restart := binding
	restart.PendingEventID = "evt_pending_restart"
	restart.EditedArtifactID = "art_part_edit_restart"
	restart.IdempotencyKey = "report-part-edit:evt_pending_restart:evt_plan:1"
	_, err := reporting.FinalizePartEdit(ctx, svc, restart, "evt_part_edit_restart", "# Part 1\n\nRestart edit.\n", 1)
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("restart should not reuse ancestor Part: %v", err)
	}
}

func newPartEditFixture(t *testing.T, ctx context.Context) (*app.Service, func(), reporting.PartEditBinding) {
	t.Helper()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	svc := app.NewService(store)
	binding := partEditBinding()
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: binding.MissionID, Title: "part edit"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: binding.SourceArtifactID, MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: "part-1.md",
		Producer: app.Producer{Type: "agent_session", ID: "provider-part"}, Content: []byte(partEditSourceMarkdown),
	}); err != nil {
		t.Fatal(err)
	}
	for _, request := range []app.AppendEventRequest{
		{EventID: binding.PendingEventID, MissionID: binding.MissionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: testJSON(map[string]any{"report_mode": "long_form"})},
		{EventID: binding.PlanEventID, MissionID: binding.MissionID, EventType: "report.plan.created", Producer: app.Producer{Type: "agent_session", ID: "provider-plan"}, Payload: testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "report_mode": "long_form", "artifact_id": "art_final"})},
		{EventID: binding.SourcePartEventID, MissionID: binding.MissionID, EventType: "report.part.created", Producer: app.Producer{Type: "agent_session", ID: "provider-part"}, Payload: testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": binding.SourceArtifactID, "part_index": binding.PartIndex})},
	} {
		if _, err := svc.AppendEvent(ctx, request); err != nil {
			t.Fatal(err)
		}
	}
	return svc, func() { _ = store.Close() }, binding
}

func startPartEdit(t *testing.T, ctx context.Context, svc *app.Service, binding reporting.PartEditBinding) {
	t.Helper()
	if _, _, err := reporting.StartPartEdit(ctx, svc, "evt_"+binding.IdempotencyKey+"_started", binding); err != nil {
		t.Fatal(err)
	}
}

func partEditBinding() reporting.PartEditBinding {
	return reporting.PartEditBinding{
		MissionID: "mis_part_edit", PendingEventID: "evt_pending", PlanEventID: "evt_plan",
		SourcePartEventID: "evt_part", SourceArtifactID: "art_part", EditedArtifactID: "art_part_edit",
		Filename: "part-1-edited.md", ToolSessionID: "ses_part_edit", ProviderSessionID: "provider-editor",
		PreviousProviderSessionID: "provider-editor", IdempotencyKey: "report-part-edit:evt_pending:evt_plan:1", PartIndex: 1,
		AgentExecutor: "codex", AgentModel: "model", AgentReasoningEffort: "medium", AgentSelectionSource: "request",
		MCPMode: "auto", ReportSessionPolicy: "isolated_fork", ReportSessionPolicySelection: "default",
		GenerationGuidanceProfile: "narrative-contract", GenerationGuidanceSHA256: "guidance-sha",
		SessionChainKind: "section_fanout_report", ReportPlanSessionID: "provider-plan", ForkSourceAgentSessionID: "provider-plan",
	}
}

func partEditedPayload(t *testing.T, event app.LedgerEvent) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

package reporting_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
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

func TestFinalizeLongFormNarrativeEditUsesExactManuscriptAndReplaysIt(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixture(t, ctx)
	defer closeStore()
	binding := longFormFinalizeBinding()
	binding.CompositionStrategy = reporting.LongFormCompositionNarrativeEdit

	draft, err := reporting.PrepareLongFormEditingDraft(ctx, svc, binding)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(draft, "Preserved body.") {
		t.Fatalf("editing draft lost bound Part content: %q", draft)
	}
	manuscript := "# Report\n\n직접 설명하는 도입입니다.\n\n## Part 1\n\nEdited body.\n\n## Conclusion\n\n정리합니다.\n"
	req := reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_final", ManuscriptMarkdown: manuscript}
	created, err := reporting.FinalizeLongForm(ctx, svc, req)
	if err != nil {
		t.Fatal(err)
	}
	if string(created.Artifact.Content) != manuscript {
		t.Fatalf("canonical artifact differs from edited manuscript:\n%s", created.Artifact.Content)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(created.Event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["composition_strategy"] != reporting.LongFormCompositionNarrativeEdit || payload["assembly_strategy"] != "narrative_contract_final_edit" {
		t.Fatalf("narrative-edit metadata missing: %#v", payload)
	}
	if _, exists := payload["preservation_ratio"]; exists {
		t.Fatalf("edited manuscript must not claim a word-count preservation ratio: %#v", payload)
	}
	replayed, err := reporting.FinalizeLongForm(ctx, svc, req)
	if err != nil || !replayed.Replay {
		t.Fatalf("exact narrative replay=%#v err=%v", replayed, err)
	}
	req.ManuscriptMarkdown = strings.Replace(manuscript, "Edited body.", "Different body.", 1)
	if _, err := reporting.FinalizeLongForm(ctx, svc, req); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("different edited manuscript error=%v, want conflict", err)
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

func TestFinalizeNarrativeEditedLongFormReplayAfterSQLiteRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "plasma.db")
	store, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	svc := app.NewService(store)
	seedLongFormFinalizeFixture(t, ctx, svc)
	binding := longFormFinalizeBinding()
	binding.CompositionStrategy = reporting.LongFormCompositionNarrativeEdit
	req := reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_final", ManuscriptMarkdown: "# Report\n\nEdited after reading every Part.\n"}
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
	if err != nil || !replayed.Replay || string(replayed.Artifact.Content) != req.ManuscriptMarkdown {
		t.Fatalf("narrative restart replay=%#v err=%v", replayed, err)
	}
	changed := req
	changed.ManuscriptMarkdown = "# Report\n\nDifferent edit.\n"
	if _, err := reporting.FinalizeLongForm(ctx, app.NewService(store), changed); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("changed narrative restart replay error=%v, want conflict", err)
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

func TestFinalizeLongFormRequiresEditedPartCompletionWhenPartEditEnabled(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixtureWithPartEditFlag(t, ctx, true)
	defer closeStore()
	binding := longFormFinalizeBinding()

	_, err := reporting.FinalizeLongForm(ctx, svc, reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_final", OpeningMarkdown: "# Open", ClosingMarkdown: "## Close"})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("enabled Part edit without reviewed Part error=%v, want conflict", err)
	}

	editBinding := longFormPartEditBinding(binding, "art_part_edit_noop", "part-edit-noop")
	edit := finalizeLongFormPartEdit(t, ctx, svc, editBinding, "evt_part_edit_noop", "# Part 1\n\nPreserved body.\n", 0)
	if edit.Artifact.ArtifactID != binding.PartArtifactIDs[0] {
		t.Fatalf("no-op Part edit should preserve source artifact: %#v", edit)
	}
	if _, err := reporting.FinalizeLongForm(ctx, svc, reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_final", OpeningMarkdown: "# Open", ClosingMarkdown: "## Close"}); err != nil {
		t.Fatalf("enabled Part edit should accept reviewed no-op source artifact: %v", err)
	}
}

func TestFinalizeLongFormRequiresEditedPartArtifactWhenPartEditChangesContent(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixtureWithPartEditFlag(t, ctx, true)
	defer closeStore()
	binding := longFormFinalizeBinding()

	editBinding := longFormPartEditBinding(binding, "art_part_edit_changed", "part-edit-changed")
	edit := finalizeLongFormPartEdit(t, ctx, svc, editBinding, "evt_part_edit_changed", "# Part 1\n\nEdited body.\n", 1)
	if edit.Artifact.ArtifactID == binding.PartArtifactIDs[0] {
		t.Fatalf("changed Part edit reused source artifact: %#v", edit)
	}
	_, err := reporting.FinalizeLongForm(ctx, svc, reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_final", OpeningMarkdown: "# Open", ClosingMarkdown: "## Close"})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("enabled Part edit accepted stale original artifact error=%v, want conflict", err)
	}

	binding.PartArtifactIDs = []string{edit.Artifact.ArtifactID}
	if _, err := reporting.FinalizeLongForm(ctx, svc, reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_final", OpeningMarkdown: "# Open", ClosingMarkdown: "## Close"}); err != nil {
		t.Fatalf("enabled Part edit rejected edited artifact lineage: %v", err)
	}
}

func TestFinalizeLongFormRejectsEditedPartArtifactWhenPartEditDisabled(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixtureWithPartEditFlag(t, ctx, false)
	defer closeStore()
	binding := longFormFinalizeBinding()

	editBinding := longFormPartEditBinding(binding, "art_part_edit_disabled", "part-edit-disabled")
	edit := finalizeLongFormPartEdit(t, ctx, svc, editBinding, "evt_part_edit_disabled", "# Part 1\n\nEdited body.\n", 1)

	editedBinding := binding
	editedBinding.PartArtifactIDs = []string{edit.Artifact.ArtifactID}
	_, err := reporting.FinalizeLongForm(ctx, svc, reporting.LongFormFinalizeRequest{Binding: editedBinding, EventID: "evt_final", OpeningMarkdown: "# Open", ClosingMarkdown: "## Close"})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("disabled Part edit accepted edited artifact error=%v, want conflict", err)
	}
	if _, err := reporting.FinalizeLongForm(ctx, svc, reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_final", OpeningMarkdown: "# Open", ClosingMarkdown: "## Close"}); err != nil {
		t.Fatalf("disabled Part edit should keep original Part lineage: %v", err)
	}
}

func TestReaderStyleGatePipelineStagesAreDurableAndOnlyGateCreatesCanonical(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixtureWithFinalEditPipeline(t, ctx, reporting.FinalEditPipelineReaderStyleGateV1, reporting.FinalEditHumanizeEnabled)
	defer closeStore()
	binding := longFormFinalizeBindingForFinalEditPipeline(reporting.FinalEditHumanizeEnabled)

	_, err := reporting.FinalizeLongForm(ctx, svc, reporting.LongFormFinalizeRequest{
		Binding:            binding,
		EventID:            "evt_final_direct",
		ManuscriptMarkdown: "# Part 1\n\nPreserved body.\n",
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("direct reader_style_gate_v1 finalization error=%v, want conflict", err)
	}

	readerSourceID := reporting.FinalEditReaderSourceArtifactID(binding.PlanEventID, []string{"art_part"})
	readerMarkdown := reporting.AssembleLongFormFinalMarkdown(binding.Title, "", "", []string{"# Part 1\n\nPreserved body.\n"})
	readerBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageReader, readerSourceID, "art_reader", "")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_reader_start", readerBinding); err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	source, err := svc.GetRawArtifact(ctx, readerSourceID)
	if err != nil {
		t.Fatal(err)
	}
	if source.Filename != binding.Filename || source.Producer != (app.Producer{Type: "system", ID: "reporting_reader_assembly"}) || string(source.Content) != readerMarkdown {
		t.Fatalf("reader source artifact differs: %#v", source)
	}
	reader, err := reporting.SubmitFinalEditStage(ctx, svc, readerBinding, "evt_reader_submit", readerMarkdown, 0)
	if err != nil {
		t.Fatal(err)
	}
	if reader.Artifact.ArtifactID != readerSourceID {
		t.Fatalf("reader no-op must reuse source artifact: %#v", reader.Artifact)
	}
	if loaded, ok, err := reporting.LoadFinalEditStageSubmission(ctx, svc, readerBinding); err != nil || !ok || !loaded.Replay {
		t.Fatalf("reader stage replay loaded=%#v ok=%t err=%v", loaded, ok, err)
	}

	styleMarkdown := strings.Replace(readerMarkdown, "Preserved body.", "Preserved body!", 1)
	styleBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageStyle, reader.Artifact.ArtifactID, "art_style", "")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_style_start", styleBinding); err != nil || !created {
		t.Fatalf("style start created=%t err=%v", created, err)
	}
	style, err := reporting.SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_style_submit", styleMarkdown, 1, []reporting.FinalEditStyleOperationDiagnosis{{
		OperationOrdinal: 1, Category: "unnatural_collocation", Reason: "awkward local phrasing",
		MatchText: "Preserved body.", Replacement: "Preserved body!", Occurrence: 1,
	}})
	if err != nil {
		t.Fatal(err)
	}
	events, err := svc.ListEvents(ctx, binding.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	if countEventType(events, "report.artifact.created") != 0 {
		t.Fatalf("reader/style stage created canonical artifact before gate: %#v", events)
	}

	gateBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageGate, style.Artifact.ArtifactID, binding.ArtifactID, "")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_gate_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	manuscript := strings.Replace(styleMarkdown, "Preserved body!", "Gate-corrected body.", 1)
	comparison, err := reporting.FinalEditSemanticComparison(ctx, svc, gateBinding, manuscript)
	if err != nil || len(comparison) != 1 {
		t.Fatalf("semantic comparison=%#v err=%v", comparison, err)
	}
	statement := "This unsupported external fact is removed."
	finalized, err := reporting.SubmitFinalEditGate(ctx, svc, reporting.FinalEditGateSubmitRequest{
		StageBinding:       gateBinding,
		FinalBinding:       binding,
		StageEventID:       "evt_gate_submit",
		CanonicalEventID:   "evt_final",
		ManuscriptMarkdown: manuscript,
		OperationCount:     1,
		Findings: []reporting.FinalEditGateFinding{{
			Statement:      statement,
			Classification: reporting.FinalEditGateClassUnverifiedExternalFact,
			RepairAction:   reporting.FinalEditRepairRemove,
		}},
		SemanticAcceptance: []reporting.FinalEditSemanticAcceptance{{
			ParagraphOrdinal:      comparison[0].ParagraphOrdinal,
			FinalParagraphOrdinal: comparison[0].ParagraphOrdinal,
			Verdict:               reporting.FinalEditSemanticRepairedByGate,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if finalized.Artifact.ArtifactID != binding.ArtifactID || string(finalized.Artifact.Content) != manuscript {
		t.Fatalf("gate final artifact differs: %#v", finalized.Artifact)
	}
	events, err = svc.ListEvents(ctx, binding.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	if countEventType(events, reporting.FinalEditReaderSubmittedEventType) != 1 ||
		countEventType(events, reporting.FinalEditStyleSubmittedEventType) != 1 ||
		countEventType(events, reporting.FinalEditGateSubmittedEventType) != 1 ||
		countEventType(events, "report.artifact.created") != 1 {
		t.Fatalf("unexpected final edit event counts: %#v", events)
	}
	for _, event := range events {
		if event.EventType == reporting.FinalEditGateSubmittedEventType || event.EventType == "report.artifact.created" {
			if strings.Contains(string(event.Payload), statement) {
				t.Fatalf("gate event leaked raw statement text: %s", string(event.Payload))
			}
		}
	}
	payload := map[string]any{}
	if err := json.Unmarshal(resultsPayload(events, "report.artifact.created"), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["final_edit_pipeline"] != reporting.FinalEditPipelineReaderStyleGateV1 {
		t.Fatalf("canonical payload missed pipeline: %#v", payload)
	}
	if payload["artifact_id"] != binding.ArtifactID ||
		payload["planned_final_artifact_id"] != binding.ArtifactID ||
		payload["final_edit_gate_event_id"] != "evt_gate_submit" ||
		payload["final_edit_gate_changed"] != true ||
		payload["artifact_sha256"] != finalized.Artifact.SHA256 {
		t.Fatalf("canonical final edit artifact fields differ: %#v", payload)
	}
	findings, ok := payload["final_edit_gate_findings"].([]any)
	if !ok || len(findings) != 1 {
		t.Fatalf("canonical payload missed gate findings: %#v", payload)
	}
	finding, ok := findings[0].(map[string]any)
	if !ok || finding["statement_sha256"] != sha256Hex(statement) || finding["repair_action"] != reporting.FinalEditRepairRemove {
		t.Fatalf("canonical finding was not redacted and normalized: %#v", finding)
	}
}

func TestSubmitFinalEditGateNoOpAdoptsPriorArtifactAsCanonical(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixtureWithFinalEditPipeline(t, ctx, reporting.FinalEditPipelineReaderStyleGateV1, reporting.FinalEditHumanizeDisabled)
	defer closeStore()
	binding := longFormFinalizeBindingForFinalEditPipeline(reporting.FinalEditHumanizeDisabled)
	readerSourceID := reporting.FinalEditReaderSourceArtifactID(binding.PlanEventID, []string{"art_part"})
	readerMarkdown := reporting.AssembleLongFormFinalMarkdown(binding.Title, "", "", []string{"# Part 1\n\nPreserved body.\n"})
	readerBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageReader, readerSourceID, "art_reader_noop_gate", "")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_reader_noop_gate_start", readerBinding); err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	reader, err := reporting.SubmitFinalEditStage(ctx, svc, readerBinding, "evt_reader_noop_gate_submit", readerMarkdown, 0)
	if err != nil {
		t.Fatal(err)
	}
	gateBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageGate, reader.Artifact.ArtifactID, binding.ArtifactID, "")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_gate_noop_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	finalized, err := reporting.SubmitFinalEditGate(ctx, svc, reporting.FinalEditGateSubmitRequest{
		StageBinding:       gateBinding,
		FinalBinding:       binding,
		StageEventID:       "evt_gate_noop_submit",
		CanonicalEventID:   "evt_gate_noop_final",
		ManuscriptMarkdown: string(reader.Artifact.Content),
		OperationCount:     0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if finalized.Artifact.ArtifactID != reader.Artifact.ArtifactID {
		t.Fatalf("no-op canonical should adopt prior artifact: %#v", finalized.Artifact)
	}
	if _, err := svc.GetRawArtifact(ctx, binding.ArtifactID); err == nil {
		t.Fatal("no-op gate created duplicate planned final raw artifact")
	}
	payload := map[string]any{}
	if err := json.Unmarshal(finalized.Event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["artifact_id"] != reader.Artifact.ArtifactID ||
		payload["planned_final_artifact_id"] != binding.ArtifactID ||
		payload["final_edit_gate_event_id"] != "evt_gate_noop_submit" ||
		payload["final_edit_gate_changed"] != false ||
		payload["artifact_sha256"] != reader.Artifact.SHA256 {
		t.Fatalf("no-op canonical payload differs: %#v", payload)
	}
	loaded, ok, err := reporting.LoadLongFormFinalization(ctx, svc, binding)
	if err != nil || !ok || loaded.Artifact.ArtifactID != reader.Artifact.ArtifactID {
		t.Fatalf("load no-op canonical loaded=%#v ok=%t err=%v", loaded, ok, err)
	}
}

func TestAssemblyWriterReaderStyleGateV2GateSubmitPreservesPipeline(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixtureWithFinalEditPipeline(t, ctx, reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2, reporting.FinalEditHumanizeDisabled)
	defer closeStore()
	binding := longFormFinalizeBindingForFinalEditPipeline(reporting.FinalEditHumanizeDisabled)
	reader := startAndSubmitV2WriterReaderStage(t, ctx, svc, binding, "v2_submit", true)
	gateBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageGate, reader.Artifact.ArtifactID, binding.ArtifactID, "")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_gate_v2_submit_start", gateBinding); err != nil || !created {
		t.Fatalf("v2 gate start created=%t err=%v", created, err)
	}
	manuscript := strings.Replace(string(reader.Artifact.Content), "Reader-smoothed", "Gate-corrected", 1)
	finalized, err := reporting.SubmitFinalEditGate(ctx, svc, reporting.FinalEditGateSubmitRequest{
		StageBinding:       gateBinding,
		FinalBinding:       binding,
		StageEventID:       "evt_gate_v2_submit",
		CanonicalEventID:   "evt_final_v2_submit",
		ManuscriptMarkdown: manuscript,
		OperationCount:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if finalized.Artifact.ArtifactID != binding.ArtifactID || string(finalized.Artifact.Content) != manuscript {
		t.Fatalf("v2 changed gate final artifact differs: %#v", finalized.Artifact)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(finalized.Event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["final_edit_pipeline"] != reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2 ||
		payload["artifact_id"] != binding.ArtifactID ||
		payload["planned_final_artifact_id"] != binding.ArtifactID ||
		payload["final_edit_gate_event_id"] != "evt_gate_v2_submit" ||
		payload["final_edit_gate_changed"] != true ||
		payload["artifact_sha256"] != finalized.Artifact.SHA256 {
		t.Fatalf("v2 changed canonical payload differs: %#v", payload)
	}
	loaded, ok, err := reporting.LoadLongFormFinalization(ctx, svc, binding)
	if err != nil || !ok || loaded.Artifact.ArtifactID != finalized.Artifact.ArtifactID {
		t.Fatalf("v2 changed canonical load loaded=%#v ok=%t err=%v", loaded, ok, err)
	}
	replayed, err := reporting.SubmitFinalEditGate(ctx, svc, reporting.FinalEditGateSubmitRequest{
		StageBinding:       gateBinding,
		FinalBinding:       binding,
		StageEventID:       "evt_gate_v2_submit_again",
		CanonicalEventID:   "evt_final_v2_submit_again",
		ManuscriptMarkdown: manuscript,
		OperationCount:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replay || replayed.Event.EventID != finalized.Event.EventID || replayed.Artifact.ArtifactID != finalized.Artifact.ArtifactID {
		t.Fatalf("v2 changed canonical replay differs: %#v", replayed)
	}
}

func TestAssemblyWriterReaderStyleGateV2GateNoOpAdoptsPriorArtifactAsCanonical(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixtureWithFinalEditPipeline(t, ctx, reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2, reporting.FinalEditHumanizeDisabled)
	defer closeStore()
	binding := longFormFinalizeBindingForFinalEditPipeline(reporting.FinalEditHumanizeDisabled)
	reader := startAndSubmitV2WriterReaderStage(t, ctx, svc, binding, "v2_noop", false)
	gateBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageGate, reader.Artifact.ArtifactID, binding.ArtifactID, "")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_gate_v2_noop_start", gateBinding); err != nil || !created {
		t.Fatalf("v2 noop gate start created=%t err=%v", created, err)
	}
	finalized, err := reporting.SubmitFinalEditGate(ctx, svc, reporting.FinalEditGateSubmitRequest{
		StageBinding:       gateBinding,
		FinalBinding:       binding,
		StageEventID:       "evt_gate_v2_noop_submit",
		CanonicalEventID:   "evt_final_v2_noop",
		ManuscriptMarkdown: string(reader.Artifact.Content),
		OperationCount:     0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if finalized.Artifact.ArtifactID != reader.Artifact.ArtifactID {
		t.Fatalf("v2 no-op canonical should adopt prior artifact: %#v", finalized.Artifact)
	}
	if _, err := svc.GetRawArtifact(ctx, binding.ArtifactID); err == nil {
		t.Fatal("v2 no-op gate created duplicate planned final raw artifact")
	}
	payload := map[string]any{}
	if err := json.Unmarshal(finalized.Event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["final_edit_pipeline"] != reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2 ||
		payload["artifact_id"] != reader.Artifact.ArtifactID ||
		payload["planned_final_artifact_id"] != binding.ArtifactID ||
		payload["final_edit_gate_event_id"] != "evt_gate_v2_noop_submit" ||
		payload["final_edit_gate_changed"] != false ||
		payload["artifact_sha256"] != reader.Artifact.SHA256 {
		t.Fatalf("v2 no-op canonical payload differs: %#v", payload)
	}
	loaded, ok, err := reporting.LoadLongFormFinalization(ctx, svc, binding)
	if err != nil || !ok || loaded.Artifact.ArtifactID != reader.Artifact.ArtifactID {
		t.Fatalf("v2 no-op canonical loaded=%#v ok=%t err=%v", loaded, ok, err)
	}
}

func TestFinalizeLongFormRejectsMalformedPartEditOutcomeWithMatchingSourceTuple(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixtureWithPartEditFlag(t, ctx, true)
	defer closeStore()
	binding := longFormFinalizeBinding()
	fakeArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_part_edit_fake", MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: "fake-edited.md",
		Producer: app.Producer{Type: "agent_session", ID: "provider-fake"}, Content: []byte("# Part 1\n\nFake same-mission Markdown.\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_part_edit_fake", MissionID: binding.MissionID, EventType: reporting.PartEditedEventType,
		Producer:         app.Producer{Type: "agent_session", ID: "provider-fake"},
		CausationEventID: "evt_part", CorrelationID: "wrong-part-edit-key",
		Payload: testJSON(map[string]any{
			"kind":                            reporting.PartEditedKind,
			"pending_event_id":                binding.PendingEventID,
			"plan_event_id":                   binding.PlanEventID,
			"source_part_event_id":            "evt_part",
			"source_artifact_id":              "art_part",
			"artifact_id":                     fakeArtifact.ArtifactID,
			"tool_session_id":                 "ses_wrong_part_edit",
			"provider_session_id":             "provider-fake",
			"previous_provider_session_id":    "provider-fake",
			"idempotency_key":                 "wrong-part-edit-key",
			"part_index":                      1,
			"agent_executor":                  binding.AgentExecutor,
			"agent_model":                     binding.AgentModel,
			"agent_reasoning_effort":          binding.AgentReasoningEffort,
			"agent_selection_source":          binding.AgentSelectionSource,
			"mcp_mode":                        binding.MCPMode,
			"report_session_policy":           binding.ReportSessionPolicy,
			"report_session_policy_selection": binding.ReportSessionPolicySelection,
			"generation_guidance_profile":     binding.GenerationGuidanceProfile,
			"generation_guidance_sha256":      binding.GenerationGuidanceSHA256,
			"session_chain_kind":              binding.SessionChainKind,
			"report_plan_session_id":          binding.ReportPlanSessionID,
			"fork_source_agent_session_id":    binding.ReportPlanSessionID,
			"operation_count":                 1,
			"changed":                         true,
		}),
	}); err != nil {
		t.Fatal(err)
	}
	binding.PartArtifactIDs = []string{fakeArtifact.ArtifactID}
	_, err = reporting.FinalizeLongForm(ctx, svc, reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_final", OpeningMarkdown: "# Open", ClosingMarkdown: "## Close"})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("malformed Part edit outcome finalized error=%v, want conflict", err)
	}
}

func TestFinalizeLongFormAcceptsIdempotentPartEditStartReplay(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixtureWithPartEditFlag(t, ctx, true)
	defer closeStore()
	binding := longFormFinalizeBinding()
	editBinding := longFormPartEditBinding(binding, "art_part_edit_replay", "part-edit-replay")
	editBinding.IdempotencyKey = fmt.Sprintf("report-part-edit:%s:%s:%d", editBinding.PendingEventID, editBinding.PlanEventID, editBinding.PartIndex)

	started, created, err := reporting.StartPartEdit(ctx, svc, "evt_part_edit_replay_started", editBinding)
	if err != nil || !created {
		t.Fatalf("Part edit start created=%t event=%#v err=%v", created, started, err)
	}
	replayed, created, err := reporting.StartPartEdit(ctx, svc, "evt_part_edit_replay_started_again", editBinding)
	if err != nil || created || replayed.EventID != started.EventID {
		t.Fatalf("Part edit start replay created=%t event=%#v err=%v", created, replayed, err)
	}
	edit, err := reporting.FinalizePartEdit(ctx, svc, editBinding, "evt_part_edit_replay", "# Part 1\n\nEdited after replayed start.\n", 1)
	if err != nil {
		t.Fatal(err)
	}
	binding.PartArtifactIDs = []string{edit.Artifact.ArtifactID}
	if _, err := reporting.FinalizeLongForm(ctx, svc, reporting.LongFormFinalizeRequest{Binding: binding, EventID: "evt_final", OpeningMarkdown: "# Open", ClosingMarkdown: "## Close"}); err != nil {
		t.Fatalf("replayed Part edit start made finalization invalid: %v", err)
	}
	events, err := svc.ListEvents(ctx, binding.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	if countEventType(events, reporting.PartEditStartedEventType) != 1 {
		t.Fatalf("exact Part edit start replay appended duplicates: %#v", events)
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
	return newLongFormFinalizeFixtureWithPartEditFlag(t, ctx, false)
}

func newLongFormFinalizeFixtureWithPartEditFlag(t *testing.T, ctx context.Context, partEditEnabled bool) (*app.Service, func()) {
	return newLongFormFinalizeFixtureWithOptions(t, ctx, longFormFinalizeFixtureOptions{PartEditEnabled: partEditEnabled})
}

func newLongFormFinalizeFixtureWithFinalEditPipeline(t *testing.T, ctx context.Context, pipeline string, postReportHumanize string) (*app.Service, func()) {
	return newLongFormFinalizeFixtureWithOptions(t, ctx, longFormFinalizeFixtureOptions{FinalEditPipeline: pipeline, PostReportHumanize: postReportHumanize})
}

type longFormFinalizeFixtureOptions struct {
	PartEditEnabled    bool
	FinalEditPipeline  string
	PostReportHumanize string
}

func newLongFormFinalizeFixtureWithOptions(t *testing.T, ctx context.Context, options longFormFinalizeFixtureOptions) (*app.Service, func()) {
	t.Helper()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	svc := app.NewService(store)
	seedLongFormFinalizeFixtureWithOptions(t, ctx, svc, options)
	return svc, func() { _ = store.Close() }
}

func seedLongFormFinalizeFixture(t *testing.T, ctx context.Context, svc *app.Service) {
	seedLongFormFinalizeFixtureWithPartEditFlag(t, ctx, svc, false)
}

func seedLongFormFinalizeFixtureWithPartEditFlag(t *testing.T, ctx context.Context, svc *app.Service, partEditEnabled bool) {
	seedLongFormFinalizeFixtureWithOptions(t, ctx, svc, longFormFinalizeFixtureOptions{PartEditEnabled: partEditEnabled})
}

func seedLongFormFinalizeFixtureWithOptions(t *testing.T, ctx context.Context, svc *app.Service, options longFormFinalizeFixtureOptions) {
	t.Helper()
	binding := longFormFinalizeBinding()
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: binding.MissionID, Title: "finalize"}); err != nil {
		t.Fatal(err)
	}
	part, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{ArtifactID: binding.PartArtifactIDs[0], MissionID: binding.MissionID, MediaType: "text/markdown; charset=utf-8", Filename: "part.md", Producer: binding.Producer, Content: []byte("# Part 1\n\nPreserved body.\n")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{ArtifactID: binding.SectionArtifactIDs[0], MissionID: binding.MissionID, MediaType: "text/markdown; charset=utf-8", Filename: "section.md", Producer: binding.Producer, Content: []byte("# Section 1\n\nPreserved body.\n")}); err != nil {
		t.Fatal(err)
	}
	planPayload := map[string]any{
		"pending_event_id": binding.PendingEventID, "report_mode": "long_form", "artifact_id": binding.ArtifactID,
		"plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"section 1"}}}},
	}
	if options.PartEditEnabled {
		planPayload["part_edit_enabled"] = true
	}
	if strings.TrimSpace(options.FinalEditPipeline) != "" {
		planPayload["final_edit_pipeline"] = strings.TrimSpace(options.FinalEditPipeline)
	}
	if strings.TrimSpace(options.PostReportHumanize) != "" {
		planPayload["post_report_humanize"] = strings.TrimSpace(options.PostReportHumanize)
	}
	requests := []app.AppendEventRequest{
		{EventID: binding.PendingEventID, MissionID: binding.MissionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: testJSON(map[string]any{"report_mode": "long_form"})},
		{EventID: binding.PlanEventID, MissionID: binding.MissionID, EventType: "report.plan.created", Producer: binding.Producer, Payload: testJSON(planPayload)},
		{EventID: "evt_part", MissionID: binding.MissionID, EventType: "report.part.created", Producer: binding.Producer, Payload: testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": part.ArtifactID, "part_index": 1})},
		{EventID: "evt_section", MissionID: binding.MissionID, EventType: "report.section.created", Producer: binding.Producer, Payload: testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": binding.SectionArtifactIDs[0], "part_index": 1, "section_index": 1})},
	}
	for _, request := range requests {
		if _, err := svc.AppendEvent(ctx, request); err != nil {
			t.Fatal(err)
		}
	}
}

func longFormFinalEditStageBinding(binding reporting.LongFormFinalizeBinding, stage string, sourceArtifactID string, editedArtifactID string, key string) reporting.FinalEditStageBinding {
	providerSessionID := "provider-" + strings.ReplaceAll(stage, "_", "-")
	filename := binding.Filename
	if stage == reporting.FinalEditStageGate {
		providerSessionID = binding.ProviderSessionID
	}
	if strings.TrimSpace(key) == "" {
		key = reporting.FinalEditStageIdempotencyKey(stage, binding.PendingEventID, binding.PlanEventID)
	}
	return reporting.FinalEditStageBinding{
		MissionID: binding.MissionID, PendingEventID: binding.PendingEventID, PlanEventID: binding.PlanEventID,
		Title: binding.Title, Stage: stage, SourceArtifactID: sourceArtifactID, EditedArtifactID: editedArtifactID,
		Filename: filename, ToolSessionID: "ses_" + stage, ProviderSessionID: providerSessionID,
		PreviousProviderSessionID: providerSessionID, IdempotencyKey: key,
		AgentExecutor: binding.AgentExecutor, AgentModel: binding.AgentModel, AgentReasoningEffort: binding.AgentReasoningEffort,
		AgentSelectionSource: binding.AgentSelectionSource, MCPMode: binding.MCPMode, RigorLevel: binding.RigorLevel, RigorLabel: binding.RigorLabel,
		ReportSessionPolicy: binding.ReportSessionPolicy, ReportSessionPolicySelection: binding.ReportSessionPolicySelection,
		PostReportHumanize: binding.PostReportHumanize, GenerationGuidanceProfile: binding.GenerationGuidanceProfile, GenerationGuidanceSHA256: binding.GenerationGuidanceSHA256,
		SessionChainKind: binding.SessionChainKind, PreReportResearchSessionID: binding.PreReportResearchSessionID, ReportPlanSessionID: binding.ReportPlanSessionID,
		ForkSourceAgentSessionID: binding.ReportPlanSessionID, Producer: app.Producer{Type: "agent_session", ID: providerSessionID},
	}
}

func longFormPartEditBinding(binding reporting.LongFormFinalizeBinding, editedArtifactID string, key string) reporting.PartEditBinding {
	return reporting.PartEditBinding{
		MissionID: binding.MissionID, PendingEventID: binding.PendingEventID, PlanEventID: binding.PlanEventID,
		SourcePartEventID: "evt_part", SourceArtifactID: "art_part", EditedArtifactID: editedArtifactID,
		Filename: "part-1-edited.md", ToolSessionID: "ses_part_edit", ProviderSessionID: "provider-part-edit",
		PreviousProviderSessionID: "provider-part-edit", IdempotencyKey: key, PartIndex: 1,
		AgentExecutor: binding.AgentExecutor, AgentModel: binding.AgentModel, AgentReasoningEffort: binding.AgentReasoningEffort,
		AgentSelectionSource: binding.AgentSelectionSource, MCPMode: binding.MCPMode,
		ReportSessionPolicy: binding.ReportSessionPolicy, ReportSessionPolicySelection: binding.ReportSessionPolicySelection,
		GenerationGuidanceProfile: binding.GenerationGuidanceProfile, GenerationGuidanceSHA256: binding.GenerationGuidanceSHA256,
		SessionChainKind: binding.SessionChainKind, ReportPlanSessionID: binding.ReportPlanSessionID,
		ForkSourceAgentSessionID: binding.ReportPlanSessionID,
	}
}

func finalizeLongFormPartEdit(t *testing.T, ctx context.Context, svc *app.Service, binding reporting.PartEditBinding, eventID string, markdown string, operationCount int) reporting.PartEditResult {
	t.Helper()
	binding.IdempotencyKey = fmt.Sprintf("report-part-edit:%s:%s:%d", binding.PendingEventID, binding.PlanEventID, binding.PartIndex)
	if _, _, err := reporting.StartPartEdit(ctx, svc, eventID+"_started", binding); err != nil {
		t.Fatal(err)
	}
	result, err := reporting.FinalizePartEdit(ctx, svc, binding, eventID, markdown, operationCount)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func startAndSubmitV2WriterReaderStage(t *testing.T, ctx context.Context, svc *app.Service, binding reporting.LongFormFinalizeBinding, suffix string, readerChanged bool) reporting.FinalEditStageResult {
	t.Helper()
	assemblyID := reporting.FinalEditAssemblyArtifactID(binding.PlanEventID, binding.PartArtifactIDs)
	writerBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageWriter, assemblyID, "art_writer_"+suffix, "")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_writer_"+suffix+"_start", writerBinding); err != nil || !created {
		t.Fatalf("writer start created=%t err=%v", created, err)
	}
	writerMarkdown := "# Report\n\nWriter opening.\n\n# Part 1\n\nPreserved body.\n\n## Conclusion\n\nWriter conclusion.\n"
	writer, err := reporting.SubmitFinalEditStage(ctx, svc, writerBinding, "evt_writer_"+suffix+"_submit", writerMarkdown, 2)
	if err != nil {
		t.Fatal(err)
	}
	readerBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageReader, writer.Artifact.ArtifactID, "art_reader_"+suffix, "")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_reader_"+suffix+"_start", readerBinding); err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	readerMarkdown := string(writer.Artifact.Content)
	operationCount := 0
	if readerChanged {
		readerMarkdown = strings.Replace(readerMarkdown, "Writer opening.", "Reader-smoothed opening.", 1)
		operationCount = 1
	}
	reader, err := reporting.SubmitFinalEditStage(ctx, svc, readerBinding, "evt_reader_"+suffix+"_submit", readerMarkdown, operationCount)
	if err != nil {
		t.Fatal(err)
	}
	return reader
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

func longFormFinalizeBindingForFinalEditPipeline(postReportHumanize string) reporting.LongFormFinalizeBinding {
	binding := longFormFinalizeBinding()
	binding.CompositionStrategy = reporting.LongFormCompositionNarrativeEdit
	binding.ToolSessionID = "ses_corrective_gate"
	binding.ProviderSessionID = "provider-corrective-gate"
	binding.PreviousProviderSessionID = "provider-corrective-gate"
	binding.PostReportHumanize = postReportHumanize
	binding.ForkSourceAgentSessionID = binding.ReportPlanSessionID
	binding.Producer = app.Producer{Type: "agent_session", ID: binding.ProviderSessionID}
	return binding
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

package reportworkflow

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporthumanize"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestFinalStoreAdoptionIgnoresCanceledRequestAfterGatePersistence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := &cancelAwareFinalStore{Service: app.NewService(store)}
	prefix := seedFinalTailPrefix(t, ctx, svc.Service, FinalTailV3, reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3, reporting.FinalEditHumanizeDisabled)
	executor := &cancelAfterGateExecutor{store: svc, cancel: func() {
		cancel()
		svc.markCanceled()
	}}
	out, err := NewRunner(RunnerConfig{Service: svc, Executor: executor, NewID: workflowSequenceID()}).FinalizeLongFormPrefix(ctx, prefix)
	if err != nil {
		t.Fatalf("%v cause=%v", err, errors.Unwrap(err))
	}
	if ctx.Err() == nil {
		t.Fatal("test did not cancel original context")
	}
	if out.Event.EventType != "report.artifact.created" || strings.TrimSpace(out.Markdown) == "" || out.Artifact.SHA256 != workflowSHA(out.Markdown) {
		t.Fatalf("canonical output not adopted after cancellation: artifact=%#v event=%#v markdown=%q", out.Artifact, out.Event, out.Markdown)
	}
	if svc.withoutCancelReads() == 0 {
		t.Fatal("expected finalstore adoption to read through a non-canceled durable context")
	}
}

func TestLegacyFinalTailH5BoundaryPreservesCanonicalOutput(t *testing.T) {
	tests := []struct {
		name          string
		humanize      string
		h5Result      agentexec.AgentResult
		h5Err         error
		wantHumanized bool
	}{
		{name: "disabled", humanize: reporting.FinalEditHumanizeDisabled},
		{name: "enabled skipped", humanize: reporting.FinalEditHumanizeEnabled, h5Result: agentexec.AgentResult{Text: "NO_H5_CHANGES", SessionID: "provider-final"}, wantHumanized: true},
		{name: "enabled failed", humanize: reporting.FinalEditHumanizeEnabled, h5Err: errors.New("h5 failed"), wantHumanized: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			svc := app.NewService(store)
			prefix := seedFinalTailPrefix(t, ctx, svc, FinalTailLegacy, "", tc.humanize)
			finalMarkdown := "# Final\n\n이 작업은 수행해야 한다.\n"
			executor := &legacyH5BoundaryExecutor{service: svc, finalMarkdown: finalMarkdown, h5Result: tc.h5Result, h5Err: tc.h5Err}
			out, err := NewRunner(RunnerConfig{Service: svc, Executor: executor, NewID: workflowSequenceID()}).FinalizeLongFormPrefix(ctx, prefix)
			if err != nil {
				t.Fatal(err)
			}
			if out.Artifact.ArtifactID != prefix.ArtifactID || out.Event.EventType != "report.artifact.created" || out.Markdown != finalMarkdown || string(out.Artifact.Content) != finalMarkdown {
				t.Fatalf("legacy canonical output changed: artifact=%#v event=%#v markdown=%q", out.Artifact, out.Event, out.Markdown)
			}
			if (out.Humanized != nil) != tc.wantHumanized {
				t.Fatalf("humanized presence=%t, want %t", out.Humanized != nil, tc.wantHumanized)
			}
			if out.Humanized != nil && (out.Humanized.Applied || out.Humanized.Artifact.ArtifactID != "" || out.Humanized.Markdown != "") {
				t.Fatalf("safe H5 zero-result must not replace canonical artifact: %#v", *out.Humanized)
			}
		})
	}
}

type cancelAwareFinalStore struct {
	*app.Service
	mu              sync.Mutex
	canceled        bool
	withoutCancelCt int
}

func (store *cancelAwareFinalStore) markCanceled() {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.canceled = true
}

func (store *cancelAwareFinalStore) withoutCancelReads() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.withoutCancelCt
}

func (store *cancelAwareFinalStore) ListEvents(ctx context.Context, missionID string) ([]ledger.Event, error) {
	store.mu.Lock()
	canceled := store.canceled
	if canceled && ctx.Err() == nil {
		store.withoutCancelCt++
	}
	store.mu.Unlock()
	if canceled {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	return store.Service.ListEvents(ctx, missionID)
}

func (store *cancelAwareFinalStore) GetRawArtifact(ctx context.Context, artifactID string) (artifact.Raw, error) {
	store.mu.Lock()
	canceled := store.canceled
	if canceled && ctx.Err() == nil {
		store.withoutCancelCt++
	}
	store.mu.Unlock()
	if canceled {
		if err := ctx.Err(); err != nil {
			return artifact.Raw{}, err
		}
	}
	return store.Service.GetRawArtifact(ctx, artifactID)
}

type cancelAfterGateExecutor struct {
	store    reporting.FinalEditStageStore
	cancel   func()
	requests int
	forks    int
}

func (executor *cancelAfterGateExecutor) Run(ctx context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
	executor.requests++
	binding := *req.FinalEditStage
	if _, _, err := reporting.StartFinalEditStage(ctx, executor.store, fmt.Sprintf("evt_cancel_start_%d", executor.requests), binding); err != nil {
		return agentexec.AgentResult{}, err
	}
	if binding.Stage == reporting.FinalEditStageEvidenceGate {
		if req.LongFormFinalize == nil {
			return agentexec.AgentResult{}, errors.New("evidence gate missing final binding")
		}
		if _, err := reporting.SubmitFinalEditEvidenceGate(ctx, executor.store, reporting.FinalEditEvidenceGateSubmitRequest{
			StageBinding: binding, FinalBinding: *req.LongFormFinalize,
			StageEventID: fmt.Sprintf("evt_cancel_submit_%d", executor.requests), CanonicalEventID: "evt_cancel_final",
		}); err != nil {
			return agentexec.AgentResult{}, err
		}
		executor.cancel()
		return agentexec.AgentResult{Text: "REPORT_FINALIZED", SessionID: binding.ProviderSessionID}, nil
	}
	source, err := executor.store.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return agentexec.AgentResult{}, err
	}
	if _, err := reporting.SubmitFinalEditStage(ctx, executor.store, binding, fmt.Sprintf("evt_cancel_submit_%d", executor.requests), string(source.Content), 0); err != nil {
		return agentexec.AgentResult{}, err
	}
	return agentexec.AgentResult{Text: "FINAL_EDIT_STAGE_SUBMITTED", SessionID: binding.ProviderSessionID}, nil
}

func (executor *cancelAfterGateExecutor) ForkSession(context.Context, string) (agentexec.AgentSessionForkResult, error) {
	executor.forks++
	sessionID := fmt.Sprintf("provider-cancel-fork-%d", executor.forks)
	return agentexec.AgentSessionForkResult{SessionID: sessionID, SourceSessionID: "provider-plan"}, nil
}

type legacyH5BoundaryExecutor struct {
	service       *app.Service
	finalMarkdown string
	h5Result      agentexec.AgentResult
	h5Err         error
}

func (executor *legacyH5BoundaryExecutor) Run(ctx context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
	if req.LongFormFinalize != nil {
		_, err := reporting.FinalizeLongForm(ctx, executor.service, reporting.LongFormFinalizeRequest{
			Binding: *req.LongFormFinalize, EventID: "evt_legacy_final", ManuscriptMarkdown: executor.finalMarkdown,
		})
		return agentexec.AgentResult{Text: "REPORT_FINALIZED", SessionID: req.PreviousSessionID}, err
	}
	if req.ReportPatch != nil {
		return executor.h5Result, executor.h5Err
	}
	return agentexec.AgentResult{}, errors.New("unexpected legacy boundary request")
}

func seedFinalTailPrefix(t *testing.T, ctx context.Context, svc *app.Service, tail FinalTail, pipeline string, humanize string) PrefixOutput {
	t.Helper()
	missionID, pendingID, planID := "mis_final_tail", "evt_final_tail_pending", "evt_final_tail_plan"
	finalID, partID, sectionID := "art_final_tail", "art_final_tail_part", "art_final_tail_section"
	producer := app.Producer{Type: "agent_session", ID: "provider-plan"}
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "Final Tail"}); err != nil {
		t.Fatal(err)
	}
	for _, req := range []app.CreateRawArtifactRequest{
		{ArtifactID: partID, MissionID: missionID, MediaType: "text/markdown; charset=utf-8", Filename: "part.md", Producer: producer, Content: []byte("# Part\n\n이 작업은 수행되어야 한다.\n")},
		{ArtifactID: sectionID, MissionID: missionID, MediaType: "text/markdown; charset=utf-8", Filename: "section.md", Producer: producer, Content: []byte("# Section\n\n이 작업은 수행되어야 한다.\n")},
	} {
		if _, err := svc.CreateRawArtifact(ctx, req); err != nil {
			t.Fatal(err)
		}
	}
	planPayload := map[string]any{"pending_event_id": pendingID, "report_mode": reportexecution.ModeLongForm, "artifact_id": finalID, "post_report_humanize": humanize, "tool_session_id": "ses_plan", "plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"section"}}}}}
	if strings.TrimSpace(pipeline) != "" {
		planPayload["final_edit_pipeline"] = pipeline
	}
	var planEvent ledger.Event
	for _, req := range []app.AppendEventRequest{
		{EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustWorkflowJSON(map[string]any{"report_mode": reportexecution.ModeLongForm})},
		{EventID: planID, MissionID: missionID, EventType: "report.plan.created", Producer: producer, Payload: mustWorkflowJSON(planPayload)},
		{EventID: "evt_final_tail_part", MissionID: missionID, EventType: "report.part.created", Producer: producer, Payload: mustWorkflowJSON(map[string]any{"pending_event_id": pendingID, "plan_event_id": planID, "artifact_id": partID, "part_index": 1})},
		{EventID: "evt_final_tail_section", MissionID: missionID, EventType: "report.section.created", Producer: producer, Payload: mustWorkflowJSON(map[string]any{"pending_event_id": pendingID, "plan_event_id": planID, "artifact_id": sectionID, "part_index": 1, "section_index": 1})},
	} {
		event, err := svc.AppendEvent(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		if event.EventID == planID {
			planEvent = event
		}
	}
	return PrefixOutput{
		MissionID: missionID, PendingEventID: pendingID, Title: "Final Tail", ExecutionStrategy: "serial",
		AgentExecutor: "codex", AgentModel: "model", AgentReasoningEffort: "high", AgentSelectionSource: "request",
		MCPMode: "auto", Rigor: reportprompt.RigorProfile{Level: "balanced", Label: "균형형"},
		ReportSessionPolicy: reportexecution.SessionPolicySameSession, ReportSessionPolicySelection: "default",
		PostReportHumanize: humanize, GenerationGuidanceProfile: reportprompt.ProfileNarrativeContract,
		GenerationGuidanceSHA256: "guidance-sha", ArtifactID: finalID,
		PlanEvent: planEvent, Plan: reporting.SectionalReportPlan{Summary: "plan"},
		Parts:           []PrefixPart{{Title: "Part", Markdown: "# Part\n\n이 작업은 수행되어야 한다.\n", ArtifactID: partID, WordCount: 4}},
		PartArtifactIDs: []string{partID}, SectionArtifactIDs: []string{sectionID}, SectionWordTotal: 4,
		SessionChainKind: "same_session_report", PreReportResearchSessionID: "provider-research",
		ReportPlanSessionID: "provider-plan", ForkSourceAgentSessionID: "provider-plan",
		ReportSessionID: "provider-final", FinalTail: tail, FinalEditPipeline: pipeline, StartedAt: time.Unix(0, 0).UTC(),
	}
}

var _ reporthumanize.Service = (*cancelAwareFinalStore)(nil)

package sourcecandidates_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidates"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestFailInterruptedStagingClosesOnlyOpenCandidatesAfterRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	store, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	svc := app.NewService(store)

	active := startRecoveryCandidate(t, ctx, svc, "active")
	archived := startRecoveryCandidate(t, ctx, svc, "archived")
	failed := startRecoveryCandidate(t, ctx, svc, "failed")
	staged := startRecoveryCandidate(t, ctx, svc, "staged")
	if _, err := svc.ArchiveMission(ctx, app.MissionLifecycleChangeRequest{
		EventID: "evt_archived_archive", MissionID: archived.MissionID,
		Producer: app.Producer{Type: "user", ID: "test"}, Reason: "test archived recovery",
	}); err != nil {
		t.Fatal(err)
	}
	if err := sourcecandidates.AppendStagingFailed(ctx, svc, failed, fixedRecoveryID("evt_failed_terminal"), fmt.Errorf("existing failure")); err != nil {
		t.Fatal(err)
	}
	if err := sourcecandidates.Stage(ctx, svc, sourcecandidates.SourceCandidateStageRequest{
		Job: staged,
		Fetcher: func(context.Context, string) (sourcecandidates.SourceCandidateFetched, error) {
			return sourcecandidates.SourceCandidateFetched{
				Content: []byte("completed body"), MediaType: "text/plain; charset=utf-8", Title: "Completed",
			}, nil
		},
		NewArtifactID: fixedRecoveryID("art_staged"),
		NewEventID:    fixedRecoveryID("evt_staged_terminal"),
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store, err = sqlite.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc = app.NewService(store)

	var sequence atomic.Int64
	newEventID := func(string) string {
		return fmt.Sprintf("evt_recovery_%d", sequence.Add(1))
	}
	type recoveryResult struct {
		closed int
		err    error
	}
	results := make([]recoveryResult, 2)
	var wait sync.WaitGroup
	for index := range results {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			results[index].closed, results[index].err = sourcecandidates.FailInterruptedStaging(ctx, svc, newEventID)
		}(index)
	}
	wait.Wait()
	closed := 0
	for _, result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		closed += result.closed
	}
	if closed != 2 {
		t.Fatalf("expected two interrupted candidates to close once, got %d", closed)
	}

	assertRecoveredFailure(t, ctx, svc, active, true)
	assertRecoveredFailure(t, ctx, svc, archived, true)
	assertRecoveredFailure(t, ctx, svc, failed, false)
	assertStagedCandidatePreserved(t, ctx, svc, staged)
	if closed, err := sourcecandidates.FailInterruptedStaging(ctx, svc, newEventID); err != nil || closed != 0 {
		t.Fatalf("repeated recovery closed=%d err=%v", closed, err)
	}
}

func startRecoveryCandidate(t *testing.T, ctx context.Context, svc *app.Service, name string) sourcecandidates.SourceCandidateStagingJob {
	t.Helper()
	missionID := "mis_" + name
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: name}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.BuildMissionCreatedAppendRequest(app.MissionCreatedEventRequest{
		EventID: "evt_" + name + "_created", MissionID: missionID, Title: name,
		Objective: "source candidate recovery test", Producer: app.Producer{Type: "user", ID: "test"},
	})); err != nil {
		t.Fatal(err)
	}
	start, err := sourcecandidates.StartStaging(ctx, svc, sourcecandidates.SourceCandidateStagingStartRequest{
		EventID: "evt_" + name + "_started", MissionID: missionID, SessionID: "ses_" + name,
		ProposalEventID: "evt_" + name + "_proposal", CandidateKind: "url",
		Candidate: sourcecandidates.SourceCandidateProposal{URL: "https://example.com/" + name, Title: name},
		Producer:  app.Producer{Type: "agent_session", ID: "agent"}, AgentExecutor: "codex",
	})
	if err != nil {
		t.Fatal(err)
	}
	return sourcecandidates.SourceCandidateStagingJob{
		MissionID: missionID, SessionID: "ses_" + name, ProposalEventID: "evt_" + name + "_proposal",
		CandidateKind: "url", Candidate: sourcecandidates.SourceCandidateProposal{URL: "https://example.com/" + name, Title: name},
		Producer: app.Producer{Type: "agent_session", ID: "agent"}, StartedEventID: start.Event.EventID,
		AgentExecutor: "codex", EmitAgentExecutorInTerminalEvents: true,
	}
}

func assertRecoveredFailure(t *testing.T, ctx context.Context, svc *app.Service, job sourcecandidates.SourceCandidateStagingJob, recovered bool) {
	t.Helper()
	events, err := svc.ListEvents(ctx, job.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	matching := 0
	for _, event := range events {
		if event.EventType == "source.candidate.rejected" {
			t.Fatalf("recovery must not reject candidate %s", job.StartedEventID)
		}
		if event.EventType != "source.candidate.staging_failed" {
			continue
		}
		var payload struct {
			StagingEventID string `json:"staging_event_id"`
			Message        string `json:"message"`
			ApprovalState  string `json:"approval_state"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.StagingEventID != job.StartedEventID {
			continue
		}
		matching++
		if recovered {
			if event.Producer != (app.Producer{Type: "system", ID: "plasma-startup"}) ||
				event.CausationEventID != job.StartedEventID || payload.ApprovalState != "unapproved_candidate" ||
				payload.Message != "Plasma가 다시 시작되어 후보 원문 가져오기를 완료하지 못했습니다. 후보를 다시 제안해 주세요." {
				t.Fatalf("unexpected recovered failure: event=%#v payload=%#v", event, payload)
			}
		}
	}
	if matching != 1 {
		t.Fatalf("expected one failure for %s, got %d", job.StartedEventID, matching)
	}
}

func assertStagedCandidatePreserved(t *testing.T, ctx context.Context, svc *app.Service, job sourcecandidates.SourceCandidateStagingJob) {
	t.Helper()
	events, err := svc.ListEvents(ctx, job.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.EventType == "source.candidate.staging_failed" && event.CausationEventID == job.StartedEventID {
			t.Fatalf("completed candidate was failed: %#v", event)
		}
	}
	artifacts, err := svc.ListRawArtifacts(ctx, job.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 || artifacts[0].ArtifactID != "art_staged" {
		t.Fatalf("completed artifact changed: %#v", artifacts)
	}
}

func fixedRecoveryID(id string) sourcecandidates.SourceCandidateIDFunc {
	return func(string) string { return id }
}

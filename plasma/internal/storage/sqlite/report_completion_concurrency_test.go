package sqlite

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
	"sync"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportusage"
)

func TestCompleteReportRunConcurrentCallsAreIdempotent(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	service := app.NewService(store)
	missionID := "mis_completion_concurrent"
	if _, err := service.CreateMission(ctx, mission.CreateRequest{MissionID: missionID, Title: "Concurrent completion"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateRawArtifact(ctx, artifactcontract.CreateRequest{ArtifactID: "art_completion_concurrent", MissionID: missionID, MediaType: "text/markdown", Filename: "report.md", Producer: ledger.Producer{Type: "agent", ID: "a"}, Content: []byte("# report")}); err != nil {
		t.Fatal(err)
	}
	for _, event := range []ledger.AppendRequest{
		{EventID: "evt_completion_concurrent_root", MissionID: missionID, EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "u"}, Payload: []byte(`{"retry_strategy":"initial"}`)},
		{EventID: "evt_completion_concurrent_target", MissionID: missionID, EventType: "report.requirements.mapped", Producer: ledger.Producer{Type: "agent_session", ID: "ses"}, Payload: []byte(`{"pending_event_id":"evt_completion_concurrent_root","previous_provider_session_id":"ses"}`)},
		{EventID: "evt_completion_concurrent_final", MissionID: missionID, EventType: "report.artifact.created", Producer: ledger.Producer{Type: "agent", ID: "a"}, Payload: []byte(`{"pending_event_id":"evt_completion_concurrent_root","artifact_id":"art_completion_concurrent"}`)},
	} {
		if _, err := service.AppendEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	const callers = 8
	results := make(chan string, callers)
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			event, err := reportrun.CompleteReportRun(ctx, service, reportrun.ReportCompletionRequest{MissionID: missionID, CanonicalEventID: "evt_completion_concurrent_final"})
			if err != nil {
				errs <- err
				return
			}
			results <- event.EventID
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	for eventID := range results {
		if eventID != "evt_report_run_completed_completion_concurrent_root" {
			t.Fatalf("unexpected completion ID %q", eventID)
		}
	}
	events, err := service.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	completionCount, usageCount := 0, 0
	for _, event := range events {
		if event.EventType == reportrun.ReportRunCompletedEventType {
			completionCount++
		}
		if event.EventType == reportusage.ReportAgentUsageRecordedEventType {
			usageCount++
		}
	}
	if completionCount != 1 || usageCount != 1 {
		t.Fatalf("completion=%d usage=%d events=%#v", completionCount, usageCount, events)
	}
}

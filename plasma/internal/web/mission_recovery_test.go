package web

import (
	"context"
	"errors"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestMissionRecoveryWorkflowFailureIsBestEffortForDetail(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	if _, err := service.CreateMission(ctx, mission.CreateRequest{MissionID: "mis_workflow_best_effort", Title: "Workflow best effort"}); err != nil {
		t.Fatal(err)
	}
	server := NewServer(service, Options{}).(*Server)
	workflowErr := errors.New("workflow reconciliation failed")
	var workflowCalls int
	server.workflowReconcile = func(context.Context, string) error {
		workflowCalls++
		return workflowErr
	}
	request := httptest.NewRequest(http.MethodGet, "/api/missions/mis_workflow_best_effort", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want %d", response.Code, http.StatusOK)
	}
	if workflowCalls != 1 {
		t.Fatalf("workflow reconciliation calls = %d, want one", workflowCalls)
	}
}

func TestMissionRecoverySkipsCompletionRecoveryWhileReportInFlight(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	if _, err := service.CreateMission(ctx, mission.CreateRequest{MissionID: "mis_inflight", Title: "In flight"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateRawArtifact(ctx, artifactcontract.CreateRequest{ArtifactID: "art_inflight", MissionID: "mis_inflight", MediaType: "text/markdown", Filename: "report.md", Producer: ledger.Producer{Type: "agent", ID: "a"}, Content: []byte("# report")}); err != nil {
		t.Fatal(err)
	}
	for _, event := range []ledger.AppendRequest{
		{EventID: "evt_inflight_root", MissionID: "mis_inflight", EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "u"}, Payload: []byte(`{"retry_strategy":"initial"}`)},
		{EventID: "evt_inflight_final", MissionID: "mis_inflight", EventType: "report.artifact.created", Producer: ledger.Producer{Type: "agent", ID: "a"}, Payload: []byte(`{"pending_event_id":"evt_inflight_root","artifact_id":"art_inflight"}`)},
	} {
		if _, err := service.AppendEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	server := NewServer(service, Options{}).(*Server)
	if _, ok := server.runningReports.Start("mis_inflight", "evt_inflight_root", func() {}); !ok {
		t.Fatal("failed to acquire in-flight ownership")
	}
	if err := server.reconcileMissionRecovery(ctx, "mis_inflight"); err != nil {
		t.Fatal(err)
	}
	events, err := service.ListEvents(ctx, "mis_inflight")
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.EventType == reportrun.ReportRunCompletedEventType {
			t.Fatal("completion was recovered while report was in flight")
		}
	}
	if !server.runningReports.Cancel("mis_inflight", "evt_inflight_root") {
		t.Fatal("failed to release in-flight ownership")
	}
	if err := server.reconcileMissionRecovery(ctx, "mis_inflight"); err != nil {
		t.Fatal(err)
	}
	events, err = service.ListEvents(ctx, "mis_inflight")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, event := range events {
		if event.EventType == reportrun.ReportRunCompletedEventType {
			found = true
		}
	}
	if !found {
		t.Fatal("completion was not recovered after in-flight release")
	}
}

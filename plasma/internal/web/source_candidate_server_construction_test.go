package web

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidates"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestNewServerDoesNotRecoverInterruptedSourceCandidate(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	if _, err := service.CreateMission(ctx, mission.CreateRequest{MissionID: "mis_server", Title: "Server"}); err != nil {
		t.Fatal(err)
	}
	if _, err := sourcecandidates.StartStaging(ctx, service, sourcecandidates.SourceCandidateStagingStartRequest{
		EventID: "evt_server_started", MissionID: "mis_server", SessionID: "ses_server",
		Candidate: sourcecandidates.SourceCandidateProposal{URL: "https://example.com/server", Title: "Server"},
		Producer:  ledger.Producer{Type: "agent_session", ID: "agent"},
	}); err != nil {
		t.Fatal(err)
	}
	before, err := service.ListEvents(ctx, "mis_server")
	if err != nil {
		t.Fatal(err)
	}

	handler := NewServer(service, Options{})
	if handler == nil {
		t.Fatal("NewServer returned a nil handler")
	}

	after, err := service.ListEvents(ctx, "mis_server")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("events changed after NewServer: before=%#v after=%#v", before, after)
	}
	for _, event := range after {
		if event.EventType == "source.candidate.staging_failed" {
			t.Fatal("NewServer created a staging_failed event")
		}
	}
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidates"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestRunServeRecoversInterruptedSourceCandidateBeforeListening(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	store, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	svc := app.NewService(store)
	if _, err := svc.CreateMission(ctx, mission.CreateRequest{MissionID: "mis_recovery", Title: "Recovery"}); err != nil {
		t.Fatal(err)
	}
	started, err := sourcecandidates.StartStaging(ctx, svc, sourcecandidates.SourceCandidateStagingStartRequest{
		EventID: "evt_recovery_started", MissionID: "mis_recovery", SessionID: "ses_recovery",
		Candidate: sourcecandidates.SourceCandidateProposal{URL: "https://example.com/recovery", Title: "Recovery"},
		Producer:  ledger.Producer{Type: "agent_session", ID: "agent"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	listen := func(server *http.Server) error {
		listenStore, err := sqlite.Open(ctx, dbPath)
		if err != nil {
			t.Fatal(err)
		}
		defer listenStore.Close()
		events, err := app.NewService(listenStore).ListEvents(ctx, "mis_recovery")
		if err != nil {
			t.Fatal(err)
		}
		foundFailure := false
		for _, event := range events {
			if event.EventType == "source.candidate.staging_failed" {
				foundFailure = true
				break
			}
		}
		if !foundFailure {
			t.Fatal("recovery was not durable before listen")
		}
		return http.ErrServerClosed
	}
	if code := runServeWithListen(ctx, []string{"-db", dbPath, "-addr", "invalid", "-agent", "none"}, &stdout, &stderr, listen); code != 0 {
		t.Fatalf("runServe returned %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "source candidate recovery: marked 1 interrupted fetches as failed") {
		t.Fatalf("missing recovery result in stderr: %q", stderr.String())
	}

	store, err = sqlite.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	events, err := app.NewService(store).ListEvents(ctx, "mis_recovery")
	if err != nil {
		t.Fatal(err)
	}
	failures := 0
	for _, event := range events {
		if event.EventType != "source.candidate.staging_failed" {
			continue
		}
		var payload struct {
			StagingEventID string `json:"staging_event_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.StagingEventID == started.Event.EventID {
			failures++
		}
	}
	if failures != 1 {
		t.Fatalf("expected one recovered failure, got %d", failures)
	}
}

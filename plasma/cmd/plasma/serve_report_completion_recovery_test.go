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
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestRunServeRecoversReportCompletionBeforeListening(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	store, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	svc := app.NewService(store)
	if _, err := svc.CreateMission(ctx, mission.CreateRequest{MissionID: "mis_report_recovery", Title: "Recovery"}); err != nil {
		t.Fatal(err)
	}
	for _, event := range []ledger.AppendRequest{
		{EventID: "evt_report_recovery_root", MissionID: "mis_report_recovery", EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "test"}, Payload: []byte(`{"retry_strategy":"initial"}`)},
		{EventID: "evt_report_recovery_target", MissionID: "mis_report_recovery", EventType: "report.requirements.mapped", Producer: ledger.Producer{Type: "agent_session", ID: "ses_recovery"}, Payload: []byte(`{"pending_event_id":"evt_report_recovery_root","previous_provider_session_id":"ses_recovery","agent_executor":"codex"}`)},
	} {
		if _, err := svc.AppendEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	artifact, _, err := svc.CreateRawArtifactWithEvent(ctx, artifactcontract.CreateRequest{ArtifactID: "art_report_recovery", MissionID: "mis_report_recovery", MediaType: "text/markdown", Filename: "report.md", Producer: ledger.Producer{Type: "agent", ID: "test"}, Content: []byte("# recovered")}, func(a artifactcontract.Raw) ledger.AppendRequest {
		return ledger.AppendRequest{EventID: "evt_report_recovery_final", MissionID: "mis_report_recovery", EventType: "report.artifact.created", Producer: ledger.Producer{Type: "agent", ID: "test"}, Payload: []byte(`{"pending_event_id":"evt_report_recovery_root","artifact_id":"` + a.ArtifactID + `"}`)}
	})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.ArtifactID != "art_report_recovery" {
		t.Fatalf("unexpected artifact id %q", artifact.ArtifactID)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	listen := func(*http.Server) error {
		opened, err := sqlite.Open(ctx, dbPath)
		if err != nil {
			t.Fatal(err)
		}
		defer opened.Close()
		events, err := app.NewService(opened).ListEvents(ctx, "mis_report_recovery")
		if err != nil {
			t.Fatal(err)
		}
		var usageEvent, completionEvent ledger.Event
		usageCount, completionCount := 0, 0
		for _, event := range events {
			switch event.EventType {
			case "report.agent_usage.recorded":
				usageCount++
				usageEvent = event
			case "report.run.completed":
				completionCount++
				completionEvent = event
			}
		}
		if usageCount != 1 || completionCount != 1 {
			t.Fatalf("recovery event counts usage=%d completion=%d", usageCount, completionCount)
		}
		if usageEvent.EventID != "evt_report_usage_report_recovery_target" || usageEvent.Producer != (ledger.Producer{Type: "agent_session", ID: "ses_recovery"}) || usageEvent.CausationEventID != "evt_report_recovery_target" || usageEvent.CorrelationID != "evt_report_recovery_root" {
			t.Fatalf("usage envelope=%#v", usageEvent)
		}
		var usagePayload struct {
			AgentUsage struct {
				SchemaVersion          int             `json:"schema_version"`
				ProviderUsage          json.RawMessage `json:"provider_usage"`
				UsageUnavailable       bool            `json:"usage_unavailable"`
				UsageUnavailableReason string          `json:"usage_unavailable_reason"`
			} `json:"agent_usage"`
		}
		if err := json.Unmarshal(usageEvent.Payload, &usagePayload); err != nil || usagePayload.AgentUsage.SchemaVersion != 2 || len(usagePayload.AgentUsage.ProviderUsage) != 0 || !usagePayload.AgentUsage.UsageUnavailable || usagePayload.AgentUsage.UsageUnavailableReason != "provider usage was lost before durable report completion" {
			t.Fatalf("usage payload=%s err=%v", usageEvent.Payload, err)
		}
		if completionEvent.EventID != "evt_report_run_completed_report_recovery_root" || completionEvent.Producer != (ledger.Producer{Type: "system", ID: "report-completion"}) || completionEvent.CausationEventID != "evt_report_recovery_final" || completionEvent.CorrelationID != "evt_report_recovery_root" {
			t.Fatalf("completion envelope=%#v", completionEvent)
		}
		var completionPayload struct {
			Kind             string `json:"kind"`
			SchemaVersion    string `json:"schema_version"`
			RunID            string `json:"run_id"`
			PendingEventID   string `json:"pending_event_id"`
			CanonicalEventID string `json:"canonical_event_id"`
			ArtifactID       string `json:"artifact_id"`
			TargetCount      int    `json:"delayed_usage_target_count"`
			Recorded         int    `json:"usage_recorded_count"`
			Unavailable      int    `json:"usage_unavailable_count"`
		}
		if err := json.Unmarshal(completionEvent.Payload, &completionPayload); err != nil || completionPayload.Kind != "report_run_completed" || completionPayload.SchemaVersion != "plasma.report_run_completion.v1" || completionPayload.RunID != "evt_report_recovery_root" || completionPayload.PendingEventID != "evt_report_recovery_root" || completionPayload.CanonicalEventID != "evt_report_recovery_final" || completionPayload.ArtifactID != "art_report_recovery" || completionPayload.TargetCount != 1 || completionPayload.Recorded != 0 || completionPayload.Unavailable != 1 {
			t.Fatalf("completion payload=%s err=%v", completionEvent.Payload, err)
		}
		return http.ErrServerClosed
	}
	if code := runServeWithListen(ctx, []string{"-db", dbPath, "-addr", "invalid", "-agent", "none"}, &stdout, &stderr, listen); code != 0 {
		t.Fatalf("runServe returned %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "startup recovery report_completion: changed 1") {
		t.Fatalf("missing report recovery result: %q", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	replayedListenCalls := 0
	replayedListen := func(server *http.Server) error {
		replayedListenCalls++
		return listen(server)
	}
	if code := runServeWithListen(ctx, []string{"-db", dbPath, "-addr", "invalid", "-agent", "none"}, &stdout, &stderr, replayedListen); code != 0 {
		t.Fatalf("replayed runServe returned %d stderr=%q", code, stderr.String())
	}
	if replayedListenCalls != 1 {
		t.Fatalf("idempotent startup did not reach listen: calls=%d stderr=%q", replayedListenCalls, stderr.String())
	}
}

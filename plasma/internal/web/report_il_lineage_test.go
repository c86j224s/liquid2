package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestIsReportArtifactAuthorizesOnlyClosedILCompanions(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	server := NewServer(svc, Options{}).(*Server)
	mission := createMissionForTest(t, ctx, svc)
	if _, err := svc.AppendEvent(ctx, ledger.AppendRequest{EventID: "evt_lineage_pending", MissionID: mission, EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{"pipeline_family": reportilcontract.PipelineFamily})}); err != nil {
		t.Fatal(err)
	}
	valid := closedILBundlePayloadWeb()
	artifactRequests := make([]artifactcontract.CreateRequest, 0, 7)
	for _, item := range []struct{ id, media, filename string }{{"art_narrative", "application/json", "narrative.json"}, {"art_document", "application/json", "semantic-il.json"}, {"art_flow", "application/json", "flow-attestation.json"}, {"art_markdown", "text/markdown; charset=utf-8", "report.md"}, {"art_html", "text/html; charset=utf-8", "report.html"}, {"art_pdf", "application/pdf", "report.pdf"}, {"art_manifest", "application/json", "manifest.json"}} {
		content := []byte(item.id)
		sum := sha256.Sum256(content)
		artifactRequests = append(artifactRequests, artifactcontract.CreateRequest{ArtifactID: item.id, MissionID: mission, MediaType: item.media, Filename: item.filename, Producer: ledger.Producer{Type: "agent", ID: "codex"}, Content: content, ExpectedSHA256: hex.EncodeToString(sum[:])})
	}
	_, terminal, created, err := svc.CreateReportILBundleIfOpen(ctx, app.ReportILBundleRequest{
		MissionID: mission, PendingID: "evt_lineage_pending", Artifacts: artifactRequests,
		StoreCompleted: ledger.AppendRequest{EventID: "evt_lineage_store", MissionID: mission, EventType: "report.il_store.completed", CausationEventID: "evt_lineage_pending", CorrelationID: "evt_lineage_pending", Producer: ledger.Producer{Type: "system", ID: "report-il"}, Payload: mustJSON(map[string]any{"kind": "report_il_stage_progress", "pending_event_id": "evt_lineage_pending", "pipeline_family": reportilcontract.PipelineFamily, "stage": "il_store", "status": "completed"})},
		Terminal:       ledger.AppendRequest{EventID: "evt_lineage_final", MissionID: mission, EventType: "report.artifact.created", CausationEventID: "evt_lineage_pending", CorrelationID: "evt_lineage_pending", Producer: ledger.Producer{Type: "agent", ID: "codex"}, Payload: mustJSON(valid)},
	})
	if err != nil || !created || terminal.EventID != "evt_lineage_final" {
		t.Fatalf("atomic lineage fixture: created=%v terminal=%q err=%v", created, terminal.EventID, err)
	}
	for _, id := range []string{"art_html", "art_pdf", "art_manifest"} {
		ok, err := server.isReportArtifact(ctx, mission, id)
		if err != nil || !ok {
			t.Fatalf("valid %s ok=%v err=%v", id, ok, err)
		}
	}
	repair := closedILBundlePayloadWeb()
	repair["source_event_id"] = "evt_lineage_final"
	for _, raw := range repair["artifact_bundle"].(map[string]any)["artifacts"].([]any) {
		entry := raw.(map[string]any)
		switch entry["kind"] {
		case "html":
			entry["artifact_id"] = "art_html_reprojected"
		case "pdf":
			entry["artifact_id"] = "art_pdf_reprojected"
		}
	}
	if _, err := svc.AppendEvent(ctx, ledger.AppendRequest{EventID: "evt_lineage_reprojected", MissionID: mission, EventType: "report.artifact.reprojected", Producer: ledger.Producer{Type: "system", ID: "report-il-reproject"}, CausationEventID: "evt_lineage_final", CorrelationID: "evt_lineage_pending", Payload: mustJSON(repair)}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"art_html_reprojected", "art_pdf_reprojected"} {
		ok, err := server.isReportArtifact(ctx, mission, id)
		if err != nil || !ok {
			t.Fatalf("valid deterministic reprojection %s ok=%v err=%v", id, ok, err)
		}
	}
	unboundRepair := closedILBundlePayloadWeb()
	unboundRepair["source_event_id"] = "evt_missing_source"
	unboundRepair["artifact_bundle"].(map[string]any)["artifacts"].([]any)[4].(map[string]any)["artifact_id"] = "art_html_unbound_reprojection"
	if _, err := svc.AppendEvent(ctx, ledger.AppendRequest{EventID: "evt_lineage_unbound_reprojection", MissionID: mission, EventType: "report.artifact.reprojected", Producer: ledger.Producer{Type: "system", ID: "report-il-reproject"}, Payload: mustJSON(unboundRepair)}); err != nil {
		t.Fatal(err)
	}
	if ok, _ := server.isReportArtifact(ctx, mission, "art_html_unbound_reprojection"); ok {
		t.Fatal("unbound deterministic reprojection authorized")
	}
	badMission := createMissionForTest(t, ctx, svc)
	if _, err := svc.AppendEvent(ctx, ledger.AppendRequest{EventID: "evt_bad_pending", MissionID: badMission, EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{"pipeline_family": reportilcontract.PipelineFamily})}); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ id, media, filename string }{{"art_narrative", "application/json", "narrative.json"}, {"art_document", "application/json", "semantic-il.json"}, {"art_flow", "application/json", "flow-attestation.json"}, {"art_markdown", "text/markdown; charset=utf-8", "report.md"}, {"art_html", "text/html; charset=utf-8", "report.html"}, {"art_pdf", "application/pdf", "report.pdf"}, {"art_manifest", "application/json", "manifest.json"}} {
		if _, err := svc.CreateRawArtifact(ctx, artifactcontract.CreateRequest{ArtifactID: item.id + "_bad", MissionID: badMission, MediaType: item.media, Filename: item.filename, Producer: ledger.Producer{Type: "agent", ID: "codex"}, Content: []byte(item.id + "_bad")}); err != nil {
			t.Fatal(err)
		}
	}
	bad := closedILBundlePayloadWeb()
	bad["pending_event_id"] = "evt_bad_pending"
	bad["artifact_id"] = "art_markdown_bad"
	bad["artifact_bundle"].(map[string]any)["markdown_artifact_id"] = "art_markdown_bad"
	for _, raw := range bad["artifact_bundle"].(map[string]any)["artifacts"].([]any) {
		entry := raw.(map[string]any)
		entry["artifact_id"] = entry["artifact_id"].(string) + "_bad"
	}
	bad["artifact_bundle"].(map[string]any)["artifacts"].([]any)[3].(map[string]any)["artifact_id"] = "art_markdown_bad"
	bad["artifact_bundle"].(map[string]any)["artifacts"].([]any)[0].(map[string]any)["sha256"] = "BAD"
	if _, err := svc.AppendEvent(ctx, ledger.AppendRequest{EventID: "evt_lineage_bad", MissionID: badMission, EventType: "report.artifact.created", Producer: ledger.Producer{Type: "agent", ID: "codex"}, Payload: mustJSON(bad)}); err == nil || !strings.Contains(err.Error(), "atomic bundle") {
		t.Fatalf("malformed experimental success was not rejected by the atomic-only boundary: %v", err)
	}
	events, err := svc.ListEvents(ctx, badMission)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventID != "evt_bad_pending" {
		t.Fatalf("rejected experimental success changed ledger: %#v", events)
	}
	ok, _ := server.isReportArtifact(ctx, badMission, "art_html_bad")
	if ok {
		t.Fatal("malformed companion authorized")
	}

	nonILMission := createMissionForTest(t, ctx, svc)
	if _, err := svc.AppendEvent(ctx, ledger.AppendRequest{EventID: "evt_non_il_pending", MissionID: nonILMission, EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{})}); err != nil {
		t.Fatal(err)
	}
	nonIL := closedILBundlePayloadWeb()
	nonIL["pending_event_id"] = "evt_non_il_pending"
	nonIL["artifact_id"] = "art_markdown_non_il"
	nonIL["artifact_bundle"].(map[string]any)["markdown_artifact_id"] = "art_markdown_non_il"
	for _, raw := range nonIL["artifact_bundle"].(map[string]any)["artifacts"].([]any) {
		entry := raw.(map[string]any)
		entry["artifact_id"] = entry["artifact_id"].(string) + "_non_il"
	}
	nonIL["artifact_bundle"].(map[string]any)["artifacts"].([]any)[3].(map[string]any)["artifact_id"] = "art_markdown_non_il"
	if _, err := svc.CreateRawArtifact(ctx, artifactcontract.CreateRequest{ArtifactID: "art_markdown_non_il", MissionID: nonILMission, MediaType: "text/markdown; charset=utf-8", Filename: "report.md", Producer: ledger.Producer{Type: "agent", ID: "codex"}, Content: []byte("non-IL")}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, ledger.AppendRequest{EventID: "evt_non_il_final", MissionID: nonILMission, EventType: "report.artifact.created", Producer: ledger.Producer{Type: "agent", ID: "codex"}, Payload: mustJSON(nonIL)}); err == nil || !strings.Contains(err.Error(), "family") {
		t.Fatalf("non-IL pending accepted an experimental terminal family: %v", err)
	}
	ok, _ = server.isReportArtifact(ctx, nonILMission, "art_html_non_il")
	if ok {
		t.Fatal("non-IL pending authorized a companion")
	}

	if _, err := svc.CreateRawArtifact(ctx, artifactcontract.CreateRequest{ArtifactID: "art_classic", MissionID: mission, MediaType: "text/markdown; charset=utf-8", Filename: "classic.md", Producer: ledger.Producer{Type: "agent", ID: "codex"}, Content: []byte("classic")}); err != nil {
		t.Fatal(err)
	}
	classic := map[string]any{"kind": "markdown_report_artifact", "artifact_id": "art_classic"}
	if _, err := svc.AppendEvent(ctx, ledger.AppendRequest{EventID: "evt_classic", MissionID: mission, EventType: "report.artifact.created", Producer: ledger.Producer{Type: "agent", ID: "codex"}, Payload: mustJSON(classic)}); err != nil {
		t.Fatal(err)
	}
	ok, _ = server.isReportArtifact(ctx, mission, "art_classic")
	if !ok {
		t.Fatal("classic Markdown denied")
	}

	if _, err := svc.CreateRawArtifact(ctx, artifactcontract.CreateRequest{ArtifactID: "art_unknown_family", MissionID: mission, MediaType: "text/markdown; charset=utf-8", Filename: "unknown.md", Producer: ledger.Producer{Type: "agent", ID: "codex"}, Content: []byte("unknown")}); err != nil {
		t.Fatal(err)
	}
	unknown := map[string]any{"kind": "markdown_report_artifact", "artifact_id": "art_unknown_family", "pipeline_family": "report_unknown"}
	if _, err := svc.AppendEvent(ctx, ledger.AppendRequest{EventID: "evt_unknown_family", MissionID: mission, EventType: "report.artifact.created", Producer: ledger.Producer{Type: "agent", ID: "codex"}, Payload: mustJSON(unknown)}); err != nil {
		t.Fatal(err)
	}
	ok, _ = server.isReportArtifact(ctx, mission, "art_unknown_family")
	if ok {
		t.Fatal("unknown report family authorized as ordinary Markdown")
	}
}

func createMissionForTest(t *testing.T, ctx context.Context, svc *app.Service) string {
	t.Helper()
	missionID := "mis_lineage_test_" + newID("x")
	if _, err := svc.CreateMission(ctx, mission.CreateRequest{MissionID: missionID, Title: "lineage"}); err != nil {
		t.Fatal(err)
	}
	return missionID
}
func closedILBundlePayloadWeb() map[string]any {
	entries := []any{}
	for _, item := range []struct{ id, kind, media, file, role string }{{"art_narrative", "narrative", "application/json", "narrative.json", "intermediate"}, {"art_document", "semantic_il", "application/json", "semantic-il.json", "intermediate"}, {"art_flow", "flow_attestation", "application/json", "flow-attestation.json", "intermediate"}, {"art_markdown", "markdown", "text/markdown; charset=utf-8", "report.md", "final"}, {"art_html", "html", "text/html; charset=utf-8", "report.html", "derivative"}, {"art_pdf", "pdf", "application/pdf", "report.pdf", "derivative"}, {"art_manifest", "manifest", "application/json", "manifest.json", "intermediate"}} {
		content := []byte(item.id)
		sum := sha256.Sum256(content)
		entries = append(entries, map[string]any{"artifact_id": item.id, "kind": item.kind, "media_type": item.media, "filename": item.file, "role": item.role, "sha256": hex.EncodeToString(sum[:]), "byte_size": len(content)})
	}
	return map[string]any{"kind": "markdown_report_artifact", "pending_event_id": "evt_lineage_pending", "pipeline_family": reportilcontract.PipelineFamily, "artifact_id": "art_markdown", "artifact_bundle": map[string]any{"pipeline_family": reportilcontract.PipelineFamily, "markdown_artifact_id": "art_markdown", "artifacts": entries}, "created_at": time.Now().UTC()}
}

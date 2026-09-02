package web

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
	_ "modernc.org/sqlite"
)

type ilHTTPGateArtifact struct {
	ID        string
	Kind      string
	MediaType string
	Filename  string
	Role      string
	Content   []byte
	AssetID   string
}

type ilHTTPGateBundle struct {
	MissionID   string
	PendingID   string
	StoreID     string
	TerminalID  string
	Completion  string
	Artifacts   []ilHTTPGateArtifact
	MarkdownID  string
	TerminalRef string
}

func TestReportILArtifactDeleteReadDownloadHTTPGate(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	store, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	server := httptest.NewServer(NewServer(service, Options{}))
	defer server.Close()

	missionResponse := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "IL artifact route gate"})
	missionID := nestedString(t, missionResponse, "projection", "mission_id")
	bundle := seedILHTTPGateBundle(t, ctx, service, missionID)
	controlArtifacts := seedILHTTPGateControlCases(t, ctx, service, missionID)

	evidenceDB := openILHTTPGateDB(t, dbPath)
	assertILHTTPGateRawArtifacts(t, ctx, evidenceDB, missionID, bundle.Artifacts)
	assertILHTTPGateMemberships(t, ctx, evidenceDB, bundle)
	assertILHTTPGateCompletion(t, ctx, evidenceDB, bundle)
	assertILHTTPGateControlMemberships(t, ctx, evidenceDB, missionID, controlArtifacts)

	preview := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+bundle.MarkdownID+"/report_delete_preview")
	assertILHTTPGateControlDeleteFacts(t, server.URL, missionID, controlArtifacts)
	wantBytes := int64(0)
	for _, artifact := range bundle.Artifacts {
		wantBytes += int64(len(artifact.Content))
	}
	if preview["eligible"] != true || preview["deletable_artifact_count"] != float64(len(bundle.Artifacts)) ||
		preview["deletable_artifact_bytes"] != float64(wantBytes) || preview["shared_artifact_count"] != float64(0) ||
		strings.TrimSpace(preview["delete_facts_hash"].(string)) == "" {
		t.Fatalf("IL delete preview = %#v, want %d artifacts and %d bytes", preview, len(bundle.Artifacts), wantBytes)
	}
	for _, artifact := range bundle.Artifacts {
		if artifact.ID == bundle.MarkdownID {
			continue
		}
		status, _ := getJSONFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifact.ID+"/report_delete_preview")
		if status != http.StatusNotFound {
			t.Fatalf("IL companion %s was accepted as a delete root: %d", artifact.ID, status)
		}
	}

	for _, artifact := range bundle.Artifacts {
		assertILHTTPGateDownload(t, server.URL, missionID, artifact, "attachment")
	}
	for _, kind := range []string{"narrative", "semantic_il", "flow_attestation", "markdown", "manifest"} {
		artifact := ilHTTPGateArtifactByKind(t, bundle.Artifacts, kind)
		read := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifact.ID)
		if nestedString(t, read, "artifact", "artifact_id") != artifact.ID || read["content"] != string(artifact.Content) {
			t.Fatalf("normal artifact read for %s = %#v", artifact.ID, read)
		}
	}
	html := ilHTTPGateArtifactByKind(t, bundle.Artifacts, "html")
	assertILHTTPGateDownload(t, server.URL, missionID, html, "attachment")
	assertILHTTPGateHTMLPreview(t, server.URL, missionID, html)
	pdf := ilHTTPGateArtifactByKind(t, bundle.Artifacts, "pdf")
	assertILHTTPGateDownload(t, server.URL, missionID, pdf, "attachment")

	assertILHTTPGateClassicBehavior(t, server.URL, missionID)
	assertILHTTPGateRejectedCompanions(t, server.URL, missionID)
	evidenceDB.Close()

	deleted := deleteJSONBody(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+bundle.MarkdownID+"/report", map[string]any{
		"confirm_artifact_id": bundle.MarkdownID,
		"expected_revision":   int(preview["revision"].(float64)),
		"delete_facts_hash":   preview["delete_facts_hash"],
	})
	if deleted["deleted"] != true || deleted["run_id"] != bundle.PendingID {
		t.Fatalf("IL delete result = %#v", deleted)
	}

	evidenceDB = openILHTTPGateDB(t, dbPath)
	defer evidenceDB.Close()
	for _, artifact := range bundle.Artifacts {
		var count int
		if err := evidenceDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE mission_id = ? AND artifact_id = ?`, missionID, artifact.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("raw artifact %s remains after purge", artifact.ID)
		}
	}
	var membershipCount int
	if err := evidenceDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM plasma_report_run_artifacts WHERE run_id = ?`, bundle.PendingID).Scan(&membershipCount); err != nil {
		t.Fatal(err)
	}
	if membershipCount != 0 {
		t.Fatalf("IL artifact memberships remain after purge: %d", membershipCount)
	}
	var lifecycle, finalArtifactID, purgedAt string
	if err := evidenceDB.QueryRowContext(ctx, `SELECT lifecycle_state, final_artifact_id, purged_at FROM plasma_report_runs WHERE run_id = ?`, bundle.PendingID).Scan(&lifecycle, &finalArtifactID, &purgedAt); err != nil {
		t.Fatal(err)
	}
	if lifecycle != "purged" || finalArtifactID != "" || strings.TrimSpace(purgedAt) == "" {
		t.Fatalf("IL tombstone = lifecycle %q final %q purged_at %q", lifecycle, finalArtifactID, purgedAt)
	}
	for _, artifact := range bundle.Artifacts {
		status, _ := getJSONFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifact.ID+"/download")
		if status != http.StatusNotFound {
			t.Fatalf("purged artifact %s download returned %d", artifact.ID, status)
		}
	}
}

func seedILHTTPGateBundle(t *testing.T, ctx context.Context, service *app.Service, missionID string) ilHTTPGateBundle {
	t.Helper()
	bundle := ilHTTPGateBundle{
		MissionID:  missionID,
		PendingID:  "evt_il_http_gate_pending",
		StoreID:    "evt_il_http_gate_store",
		TerminalID: "evt_il_http_gate_terminal",
		Completion: "evt_report_run_completed_il_http_gate_pending",
		MarkdownID: "art_il_http_gate_markdown",
	}
	bundle.Artifacts = []ilHTTPGateArtifact{
		{ID: "art_il_http_gate_narrative", Kind: "narrative", MediaType: "application/json", Filename: "narrative.json", Role: "intermediate", Content: []byte(`{"title":"Narrative","claims":["claim-1"]}`)},
		{ID: "art_il_http_gate_semantic", Kind: "semantic_il", MediaType: "application/json", Filename: "semantic-il.json", Role: "intermediate", Content: []byte(`{"document":"semantic-il","sections":2}`)},
		{ID: "art_il_http_gate_flow", Kind: "flow_attestation", MediaType: "application/json", Filename: "flow-attestation.json", Role: "intermediate", Content: []byte(`{"stages":["narrative","document","flow","render","store"]}`)},
		{ID: bundle.MarkdownID, Kind: "markdown", MediaType: "text/markdown; charset=utf-8", Filename: "report.md", Role: "final", Content: []byte("# IL route gate\n\nThis is the Markdown report.\n")},
		{ID: "art_il_http_gate_html", Kind: "html", MediaType: "text/html; charset=utf-8", Filename: "report.html", Role: "derivative", Content: []byte(`<!doctype html><html><head><meta charset="utf-8"><style>body{font-family: sans-serif}</style></head><body><h1>IL route gate</h1></body></html>`)},
		{ID: "art_il_http_gate_pdf", Kind: "pdf", MediaType: "application/pdf", Filename: "report.pdf", Role: "derivative", Content: []byte("%PDF-1.7\nIL route gate PDF bytes\n%%EOF\n")},
		{ID: "art_il_http_gate_manifest", Kind: "manifest", MediaType: "application/json", Filename: "manifest.json", Role: "intermediate", Content: []byte(`{"bundle":"report_il_experimental","artifact_count":8}`)},
		{ID: "art_il_http_gate_image", Kind: "image", MediaType: "image/jpeg", Filename: "report-image.jpg", Role: "asset", Content: []byte("pinned image bytes"), AssetID: "asset_il_http_gate_image"},
	}
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: bundle.PendingID, MissionID: missionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"},
		Payload:  mustJSON(map[string]any{"title": "IL route gate", "pipeline_family": reportilcontract.PipelineFamily}),
	}); err != nil {
		t.Fatal(err)
	}
	artifactRequests := make([]app.CreateRawArtifactRequest, 0, len(bundle.Artifacts))
	entries := make([]map[string]any, 0, len(bundle.Artifacts))
	for _, artifact := range bundle.Artifacts {
		hash := sha256Hex(artifact.Content)
		artifactRequests = append(artifactRequests, app.CreateRawArtifactRequest{
			ArtifactID: artifact.ID, MissionID: missionID, MediaType: artifact.MediaType,
			Filename: artifact.Filename, Producer: app.Producer{Type: "agent", ID: "codex"},
			Content: artifact.Content, ExpectedSHA256: hash,
		})
		entries = append(entries, map[string]any{
			"artifact_id": artifact.ID, "kind": artifact.Kind, "media_type": artifact.MediaType,
			"filename": artifact.Filename, "role": artifact.Role, "sha256": hash, "byte_size": len(artifact.Content),
			"asset_id": artifact.AssetID,
		})
	}
	terminalPayload := map[string]any{
		"kind": "markdown_report_artifact", "pending_event_id": bundle.PendingID,
		"pipeline_family": reportilcontract.PipelineFamily, "artifact_id": bundle.MarkdownID,
		"artifact_bundle": map[string]any{
			"pipeline_family": reportilcontract.PipelineFamily, "markdown_artifact_id": bundle.MarkdownID, "artifacts": entries,
		},
	}
	result, terminal, created, err := service.CreateReportILBundleIfOpen(ctx, app.ReportILBundleRequest{
		MissionID: missionID, PendingID: bundle.PendingID, Artifacts: artifactRequests,
		StoreCompleted: app.AppendEventRequest{
			EventID: bundle.StoreID, MissionID: missionID, EventType: "report.il_store.completed",
			CausationEventID: bundle.PendingID, CorrelationID: bundle.PendingID,
			Producer: app.Producer{Type: "system", ID: "report-il"},
			Payload:  mustJSON(map[string]any{"kind": "report_il_stage_progress", "pending_event_id": bundle.PendingID, "pipeline_family": reportilcontract.PipelineFamily, "stage": "il_store", "status": "completed"}),
		},
		Terminal: app.AppendEventRequest{
			EventID: bundle.TerminalID, MissionID: missionID, EventType: "report.artifact.created",
			CausationEventID: bundle.PendingID, CorrelationID: bundle.PendingID,
			Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: mustJSON(terminalPayload),
		},
	})
	if err != nil || !created || len(result) != len(bundle.Artifacts) || terminal.EventID != bundle.TerminalID {
		t.Fatalf("atomic IL bundle = artifacts %d terminal %q created %v err %v", len(result), terminal.EventID, created, err)
	}
	bundle.TerminalRef = terminal.EventID
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: bundle.Completion, MissionID: missionID, EventType: "report.run.completed",
		CausationEventID: bundle.TerminalID, CorrelationID: bundle.PendingID,
		Producer: app.Producer{Type: "system", ID: "report-completion"},
		Payload: mustJSON(map[string]any{
			"kind": "report_run_completed", "schema_version": "plasma.report_run_completion.v1",
			"run_id": bundle.PendingID, "pending_event_id": bundle.PendingID,
			"canonical_event_id": bundle.TerminalID, "artifact_id": bundle.MarkdownID,
			"delayed_usage_target_count": 0, "usage_recorded_count": 0, "usage_unavailable_count": 0,
		}),
	}); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func seedILHTTPGateControlCases(t *testing.T, ctx context.Context, service *app.Service, missionID string) []ilHTTPGateArtifact {
	t.Helper()
	seedClassicILHTTPGateArtifact(t, ctx, service, missionID)
	var controlArtifacts []ilHTTPGateArtifact
	for _, malformed := range []struct {
		pending string
		suffix  string
		badHash bool
	}{
		{pending: "evt_il_http_gate_malformed_pending", suffix: "malformed", badHash: true},
		{pending: "evt_il_http_gate_nonil_pending", suffix: "nonil", badHash: false},
	} {
		if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
			EventID: malformed.pending, MissionID: missionID, EventType: "report.draft.pending",
			Producer: app.Producer{Type: "user", ID: "test"},
			Payload: func() []byte {
				if malformed.badHash {
					return mustJSON(map[string]any{"title": "malformed IL", "pipeline_family": reportilcontract.PipelineFamily})
				}
				return mustJSON(map[string]any{"title": "non-IL pending"})
			}(),
		}); err != nil {
			t.Fatal(err)
		}
		artifacts := make([]ilHTTPGateArtifact, 0, 7)
		for _, source := range []struct {
			kind, media, filename, role string
		}{
			{"narrative", "application/json", "narrative.json", "intermediate"},
			{"semantic_il", "application/json", "semantic-il.json", "intermediate"},
			{"flow_attestation", "application/json", "flow-attestation.json", "intermediate"},
			{"markdown", "text/markdown; charset=utf-8", "report.md", "final"},
			{"html", "text/html; charset=utf-8", "report.html", "derivative"},
			{"pdf", "application/pdf", "report.pdf", "derivative"},
			{"manifest", "application/json", "manifest.json", "intermediate"},
		} {
			id := "art_il_http_gate_" + malformed.suffix + "_" + source.kind
			content := []byte(malformed.suffix + "-" + source.kind)
			if source.kind == "html" {
				content = []byte("<!doctype html><html><body>" + malformed.suffix + "</body></html>")
			}
			artifact := ilHTTPGateArtifact{ID: id, Kind: source.kind, MediaType: source.media, Filename: source.filename, Role: source.role, Content: content}
			artifacts = append(artifacts, artifact)
			controlArtifacts = append(controlArtifacts, artifact)
			if _, err := service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
				ArtifactID: id, MissionID: missionID, MediaType: source.media, Filename: source.filename,
				Producer: app.Producer{Type: "agent", ID: "codex"}, Content: content,
			}); err != nil {
				t.Fatal(err)
			}
		}
		markdownID := "art_il_http_gate_" + malformed.suffix + "_markdown"
		entries := make([]map[string]any, 0, len(artifacts))
		for _, artifact := range artifacts {
			hash := sha256Hex(artifact.Content)
			if malformed.badHash && artifact.Kind == "narrative" {
				hash = "BAD"
			}
			entries = append(entries, map[string]any{
				"artifact_id": artifact.ID, "kind": artifact.Kind, "media_type": artifact.MediaType,
				"filename": artifact.Filename, "role": artifact.Role, "sha256": hash, "byte_size": len(artifact.Content),
			})
		}
		terminalID := "evt_il_http_gate_" + malformed.suffix + "_terminal"
		_, appendErr := service.AppendEvent(ctx, app.AppendEventRequest{
			EventID: terminalID, MissionID: missionID, EventType: "report.artifact.created",
			Producer: app.Producer{Type: "agent", ID: "codex"}, CausationEventID: malformed.pending, CorrelationID: malformed.pending,
			Payload: mustJSON(map[string]any{
				"kind": "markdown_report_artifact", "pending_event_id": malformed.pending,
				"pipeline_family": reportilcontract.PipelineFamily, "artifact_id": markdownID,
				"artifact_bundle": map[string]any{"pipeline_family": reportilcontract.PipelineFamily, "markdown_artifact_id": markdownID, "artifacts": entries},
			}),
		})
		if appendErr == nil {
			t.Fatalf("%s experimental terminal was accepted outside its pending family", malformed.suffix)
		}
		if malformed.badHash && !strings.Contains(appendErr.Error(), "atomic bundle") {
			t.Fatalf("malformed experimental success was not rejected by the atomic-only boundary: %v", appendErr)
		}
		if !malformed.badHash && !strings.Contains(appendErr.Error(), "family") {
			t.Fatalf("non-IL pending accepted an experimental terminal family: %v", appendErr)
		}
	}
	return controlArtifacts
}

func seedClassicILHTTPGateArtifact(t *testing.T, ctx context.Context, service *app.Service, missionID string) {
	t.Helper()
	pendingID := "evt_il_http_gate_classic_pending"
	finalID := "art_il_http_gate_classic"
	terminalID := "evt_il_http_gate_classic_terminal"
	completionID := "evt_report_run_completed_il_http_gate_classic_pending"
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{"title": "Classic report"}),
	}); err != nil {
		t.Fatal(err)
	}
	content := []byte("# Classic report\n")
	if _, err := service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{ArtifactID: finalID, MissionID: missionID, MediaType: "text/markdown; charset=utf-8", Filename: "classic.md", Producer: app.Producer{Type: "agent", ID: "test"}, Content: content}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: terminalID, MissionID: missionID, EventType: "report.artifact.created", Producer: app.Producer{Type: "agent", ID: "test"}, CausationEventID: pendingID, CorrelationID: pendingID,
		Payload: mustJSON(map[string]any{"kind": "markdown_report_artifact", "pending_event_id": pendingID, "artifact_id": finalID}),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: completionID, MissionID: missionID, EventType: "report.run.completed", Producer: app.Producer{Type: "system", ID: "report-completion"}, CausationEventID: terminalID, CorrelationID: pendingID,
		Payload: mustJSON(map[string]any{"kind": "report_run_completed", "schema_version": "plasma.report_run_completion.v1", "run_id": pendingID, "pending_event_id": pendingID, "canonical_event_id": terminalID, "artifact_id": finalID, "delayed_usage_target_count": 0, "usage_recorded_count": 0, "usage_unavailable_count": 0}),
	}); err != nil {
		t.Fatal(err)
	}
}

func assertILHTTPGateRawArtifacts(t *testing.T, ctx context.Context, db *sql.DB, missionID string, artifacts []ilHTTPGateArtifact) {
	t.Helper()
	for _, artifact := range artifacts {
		var mediaType, filename, hash string
		var byteSize int64
		if err := db.QueryRowContext(ctx, `SELECT media_type, filename, byte_size, sha256 FROM plasma_raw_artifacts WHERE mission_id = ? AND artifact_id = ?`, missionID, artifact.ID).Scan(&mediaType, &filename, &byteSize, &hash); err != nil {
			t.Fatalf("raw artifact %s: %v", artifact.ID, err)
		}
		if mediaType != artifact.MediaType || filename != artifact.Filename || byteSize != int64(len(artifact.Content)) || hash != sha256Hex(artifact.Content) {
			t.Fatalf("raw artifact %s = media %q filename %q bytes %d hash %q", artifact.ID, mediaType, filename, byteSize, hash)
		}
	}
}

func assertILHTTPGateMemberships(t *testing.T, ctx context.Context, db *sql.DB, bundle ilHTTPGateBundle) {
	t.Helper()
	rows, err := db.QueryContext(ctx, `SELECT artifact_id, artifact_role, ownership, source_event_id FROM plasma_report_run_artifacts WHERE run_id = ? AND mission_id = ? ORDER BY artifact_id`, bundle.PendingID, bundle.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := map[string][3]string{}
	for rows.Next() {
		var artifactID, role, ownership, sourceEventID string
		if err := rows.Scan(&artifactID, &role, &ownership, &sourceEventID); err != nil {
			t.Fatal(err)
		}
		got[artifactID] = [3]string{role, ownership, sourceEventID}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(bundle.Artifacts) {
		t.Fatalf("IL membership count = %d, want %d: %#v", len(got), len(bundle.Artifacts), got)
	}
	for _, artifact := range bundle.Artifacts {
		membership, ok := got[artifact.ID]
		membershipRole := artifact.Role
		if membershipRole == "asset" {
			membershipRole = "intermediate"
		}
		if !ok || membership != [3]string{membershipRole, "created", bundle.TerminalID} {
			t.Fatalf("IL membership %s = %#v, want role %q created source %q", artifact.ID, membership, membershipRole, bundle.TerminalID)
		}
	}
}

func assertILHTTPGateControlMemberships(t *testing.T, ctx context.Context, db *sql.DB, missionID string, artifacts []ilHTTPGateArtifact) {
	t.Helper()
	for _, artifact := range artifacts {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM plasma_report_run_artifacts WHERE mission_id = ? AND artifact_id = ?`, missionID, artifact.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("control companion %s unexpectedly has report-run membership", artifact.ID)
		}
	}
}

func assertILHTTPGateControlDeleteFacts(t *testing.T, serverURL, missionID string, artifacts []ilHTTPGateArtifact) {
	t.Helper()
	for _, artifact := range artifacts {
		status, _ := getJSONFailure(t, serverURL+"/api/missions/"+missionID+"/artifacts/"+artifact.ID+"/report_delete_preview")
		if status != http.StatusNotFound {
			t.Fatalf("control artifact %s expanded delete facts with status %d", artifact.ID, status)
		}
		for _, suffix := range []string{"", "/download"} {
			status, _ = getJSONFailure(t, serverURL+"/api/missions/"+missionID+"/artifacts/"+artifact.ID+suffix)
			if status != http.StatusNotFound {
				t.Fatalf("control artifact %s route %s returned %d", artifact.ID, suffix, status)
			}
		}
	}
}

func assertILHTTPGateCompletion(t *testing.T, ctx context.Context, db *sql.DB, bundle ilHTTPGateBundle) {
	t.Helper()
	var eventType, causation, correlation, producerType, producerID string
	var payload string
	if err := db.QueryRowContext(ctx, `SELECT event_type, causation_event_id, correlation_id, producer_type, producer_id, payload_json FROM plasma_ledger_events WHERE event_id = ? AND mission_id = ?`, bundle.Completion, bundle.MissionID).Scan(&eventType, &causation, &correlation, &producerType, &producerID, &payload); err != nil {
		t.Fatal(err)
	}
	if eventType != "report.run.completed" || causation != bundle.TerminalID || correlation != bundle.PendingID || producerType != "system" || producerID != "report-completion" {
		t.Fatalf("completion envelope = type %q causation %q correlation %q producer %s/%s", eventType, causation, correlation, producerType, producerID)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["kind"] != "report_run_completed" || decoded["schema_version"] != "plasma.report_run_completion.v1" || decoded["run_id"] != bundle.PendingID || decoded["canonical_event_id"] != bundle.TerminalID || decoded["artifact_id"] != bundle.MarkdownID {
		t.Fatalf("completion payload = %#v", decoded)
	}
	var lifecycle, finalArtifactID, registration string
	if err := db.QueryRowContext(ctx, `SELECT lifecycle_state, final_artifact_id, registration_status FROM plasma_report_runs WHERE run_id = ?`, bundle.PendingID).Scan(&lifecycle, &finalArtifactID, &registration); err != nil {
		t.Fatal(err)
	}
	if lifecycle != "completed" || finalArtifactID != bundle.MarkdownID || registration != "native" {
		t.Fatalf("IL run = lifecycle %q final %q registration %q", lifecycle, finalArtifactID, registration)
	}
}

type ilHTTPGateResponse struct {
	Status int
	Header http.Header
	Body   []byte
}

func assertILHTTPGateDownload(t *testing.T, serverURL, missionID string, artifact ilHTTPGateArtifact, disposition string) {
	t.Helper()
	response := fetchILHTTPGate(t, serverURL+"/api/missions/"+missionID+"/artifacts/"+artifact.ID+"/download")
	if response.Status != http.StatusOK || !bytes.Equal(response.Body, artifact.Content) {
		t.Fatalf("download %s = status %d body %q", artifact.ID, response.Status, response.Body)
	}
	if response.Header.Get("Content-Type") != artifact.MediaType || response.Header.Get("Content-Length") != strconv.Itoa(len(artifact.Content)) {
		t.Fatalf("download %s headers content-type %q length %q", artifact.ID, response.Header.Get("Content-Type"), response.Header.Get("Content-Length"))
	}
	wantDisposition := mime.FormatMediaType(disposition, map[string]string{"filename": artifact.Filename})
	if response.Header.Get("Content-Disposition") != wantDisposition {
		t.Fatalf("download %s disposition %q, want %q", artifact.ID, response.Header.Get("Content-Disposition"), wantDisposition)
	}
}

func assertILHTTPGateHTMLPreview(t *testing.T, serverURL, missionID string, artifact ilHTTPGateArtifact) {
	t.Helper()
	response := fetchILHTTPGate(t, serverURL+"/api/missions/"+missionID+"/artifacts/"+artifact.ID+"/preview")
	if response.Status != http.StatusOK || !bytes.Equal(response.Body, artifact.Content) {
		t.Fatalf("HTML preview = status %d body %q", response.Status, response.Body)
	}
	if response.Header.Get("Content-Type") != artifact.MediaType || response.Header.Get("Content-Length") != strconv.Itoa(len(artifact.Content)) {
		t.Fatalf("HTML preview headers content-type %q length %q", response.Header.Get("Content-Type"), response.Header.Get("Content-Length"))
	}
	wantDisposition := mime.FormatMediaType("inline", map[string]string{"filename": artifact.Filename})
	csp := response.Header.Get("Content-Security-Policy")
	if response.Header.Get("Content-Disposition") != wantDisposition || !strings.Contains(csp, "sandbox allow-scripts") || strings.Contains(csp, "allow-same-origin") || bytes.Contains(response.Body, []byte("http://")) || bytes.Contains(response.Body, []byte("https://")) {
		t.Fatalf("HTML preview is not self-contained: disposition %q CSP %q body %q", response.Header.Get("Content-Disposition"), csp, response.Body)
	}
}

func assertILHTTPGateClassicBehavior(t *testing.T, serverURL, missionID string) {
	t.Helper()
	finalID := "art_il_http_gate_classic"
	read := getJSON(t, serverURL+"/api/missions/"+missionID+"/artifacts/"+finalID)
	if read["content"] != "# Classic report\n" {
		t.Fatalf("classic report read = %#v", read)
	}
	classic := ilHTTPGateArtifact{ID: finalID, MediaType: "text/markdown; charset=utf-8", Filename: "classic.md", Content: []byte("# Classic report\n")}
	assertILHTTPGateDownload(t, serverURL, missionID, classic, "attachment")
	preview := getJSON(t, serverURL+"/api/missions/"+missionID+"/artifacts/"+finalID+"/report_delete_preview")
	if preview["eligible"] != true || preview["deletable_artifact_count"] != float64(1) || preview["deletable_artifact_bytes"] != float64(len(classic.Content)) {
		t.Fatalf("classic delete preview = %#v", preview)
	}
	deleted := deleteJSONBody(t, serverURL+"/api/missions/"+missionID+"/artifacts/"+finalID+"/report", map[string]any{
		"confirm_artifact_id": finalID,
		"expected_revision":   int(preview["revision"].(float64)),
		"delete_facts_hash":   preview["delete_facts_hash"],
	})
	if deleted["deleted"] != true {
		t.Fatalf("classic delete result = %#v", deleted)
	}
	if status, _ := getJSONFailure(t, serverURL+"/api/missions/"+missionID+"/artifacts/"+finalID); status != http.StatusNotFound {
		t.Fatalf("classic artifact after delete = %d", status)
	}
}

func assertILHTTPGateRejectedCompanions(t *testing.T, serverURL, missionID string) {
	t.Helper()
	for _, suffix := range []string{"malformed", "nonil"} {
		for _, kind := range []string{"narrative", "semantic_il", "flow_attestation", "markdown", "html", "pdf", "manifest"} {
			artifactID := "art_il_http_gate_" + suffix + "_" + kind
			if status, _ := getJSONFailure(t, serverURL+"/api/missions/"+missionID+"/artifacts/"+artifactID); status != http.StatusNotFound {
				t.Fatalf("%s %s normal read returned %d", suffix, kind, status)
			}
			if status, _ := getJSONFailure(t, serverURL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/download"); status != http.StatusNotFound {
				t.Fatalf("%s %s download returned %d", suffix, kind, status)
			}
		}
	}
}

func fetchILHTTPGate(t *testing.T, endpoint string) ilHTTPGateResponse {
	t.Helper()
	response, err := http.Get(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return ilHTTPGateResponse{Status: response.StatusCode, Header: response.Header, Body: body}
}

func openILHTTPGateDB(t *testing.T, dbPath string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db
}

func ilHTTPGateArtifactByKind(t *testing.T, artifacts []ilHTTPGateArtifact, kind string) ilHTTPGateArtifact {
	t.Helper()
	for _, artifact := range artifacts {
		if artifact.Kind == kind {
			return artifact
		}
	}
	t.Fatalf("missing IL artifact kind %q", kind)
	return ilHTTPGateArtifact{}
}

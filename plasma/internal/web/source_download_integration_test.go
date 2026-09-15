package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestSourceDownloadRoutesServeExactBytesAndUniformFailures(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	store, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	service := app.NewService(store)
	if err := store.CreateMission(ctx, mission.Mission{MissionID: "mis_1", Title: "Downloads"}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateMission(ctx, mission.Mission{MissionID: "mis_2", Title: "Other"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateRawArtifact(ctx, artifactcontract.CreateRequest{ArtifactID: "art_html", MissionID: "mis_1", MediaType: "text/html; charset=utf-8", Filename: "page.html", Producer: ledger.Producer{Type: "test", ID: "test"}, Content: []byte("<html>stored</html>")}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateRawArtifact(ctx, artifactcontract.CreateRequest{ArtifactID: "art_pdf", MissionID: "mis_1", MediaType: "application/pdf", Filename: "paper.pdf", Producer: ledger.Producer{Type: "test", ID: "test"}, Content: []byte("%PDF-stored")}); err != nil {
		t.Fatal(err)
	}
	appendEvent(t, store, ledger.Event{EventID: "evt_staged", MissionID: "mis_1", EventType: "source.candidate.staged", Payload: json.RawMessage(`{"url":"https://example.com/doc","artifact_id":"art_html","proposal_event_id":"evt_proposed"}`)})
	if _, err := service.CreateSourceSnapshot(ctx, sourcecontract.CreateRequest{SnapshotID: "src_1", MissionID: "mis_1", Connector: sourcecontract.ConnectorRef{ConnectorID: "pdf", ConnectorType: "pdf_url", ExternalURI: "https://example.com/paper.pdf"}, Title: "Paper", ArtifactIDs: []string{"art_pdf"}}); err != nil {
		t.Fatal(err)
	}
	store.Close()
	store, err = sqlite.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service = app.NewService(store)
	handler := NewServer(service, Options{urlFetcher: func(context.Context, string) (fetchedURLSource, error) {
		t.Fatal("refetch")
		return fetchedURLSource{}, nil
	}, browserRenderer: func(context.Context, string) (fetchedURLSource, error) {
		t.Fatal("render")
		return fetchedURLSource{}, nil
	}})
	server := httptest.NewServer(handler)
	defer server.Close()
	for _, tc := range []struct{ path, body, contentType string }{{"/api/missions/mis_1/candidates/sources/evt_staged/download?artifact_id=art_html", "<html>stored</html>", "text/html; charset=utf-8"}, {"/api/missions/mis_1/sources/src_1/download", "%PDF-stored", "application/pdf"}} {
		resp, err := http.Get(server.URL + tc.path)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(data) != tc.body || resp.Header.Get("Content-Type") != tc.contentType || resp.Header.Get("Content-Length") != stringLength(len(data)) || resp.Header.Get("X-Content-Type-Options") != "nosniff" || resp.Header.Get("Cache-Control") != "no-store" {
			t.Fatalf("response %#v body=%q", resp.Header, data)
		}
		media, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
		if err != nil || media != "attachment" || strings.ContainsAny(params["filename"], "/\\\"\r\n") {
			t.Fatalf("bad disposition %q %v", resp.Header.Get("Content-Disposition"), err)
		}
	}
	for _, path := range []string{"/api/missions/mis_1/candidates/sources/evt_missing/download?artifact_id=art_html", "/api/missions/mis_2/candidates/sources/evt_staged/download?artifact_id=art_html", "/api/missions/mis_1/sources/src_missing/download", "/api/missions/mis_1/sources/src_1/download?artifact_id=art_missing", "/api/missions/mis_1/artifacts/art_html/download"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound || !bytes.Contains(body, []byte("source artifact not found")) && strings.Contains(path, "/candidates/") {
			t.Fatalf("path %s status=%d body=%s", path, resp.StatusCode, body)
		}
	}
	for _, method := range []string{"POST", "PUT"} {
		req, _ := http.NewRequest(method, server.URL+"/api/missions/mis_1/sources/src_1/download", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("method %s status %d", method, resp.StatusCode)
		}
	}
}

func appendEvent(t *testing.T, store *sqlite.Store, event ledger.Event) {
	t.Helper()
	if _, err := store.AppendLedgerEvent(context.Background(), event); err != nil {
		t.Fatal(err)
	}
}
func stringLength(n int) string { return fmt.Sprintf("%d", n) }

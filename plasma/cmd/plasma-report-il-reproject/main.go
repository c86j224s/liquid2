package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/pdfdocument"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportilphase0"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func main() {
	var databasePath string
	var missionID string
	var documentArtifactID string
	var chromePath string
	var write bool
	flag.StringVar(&databasePath, "db", "", "Plasma SQLite database")
	flag.StringVar(&missionID, "mission", "", "mission ID")
	flag.StringVar(&documentArtifactID, "document", "", "semantic IL artifact ID")
	flag.StringVar(&chromePath, "chrome", "", "Chrome executable path")
	flag.BoolVar(&write, "write", false, "store the corrected projections")
	flag.Parse()
	if strings.TrimSpace(databasePath) == "" || !strings.HasPrefix(missionID, "mis_") || !strings.HasPrefix(documentArtifactID, "art_") || strings.TrimSpace(chromePath) == "" {
		fmt.Fprintln(os.Stderr, "usage: plasma-report-il-reproject -db PATH -mission mis_... -document art_... -chrome PATH [-write]")
		os.Exit(2)
	}
	ctx := context.Background()
	store, err := sqlite.Open(ctx, databasePath)
	if err != nil {
		fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	artifact, err := service.GetRawArtifact(ctx, documentArtifactID)
	if err != nil {
		fatal(err)
	}
	if artifact.MissionID != missionID || artifact.MediaType != "application/json" {
		fatal(fmt.Errorf("semantic IL artifact binding differs"))
	}
	var document reportilphase0.Document
	decoder := json.NewDecoder(strings.NewReader(string(artifact.Content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		fatal(err)
	}
	htmlContent, _, err := reportilphase0.RenderHTML(document)
	if err != nil {
		fatal(err)
	}
	pdf, err := reportilphase0.RenderPDF(ctx, htmlContent, chromePath)
	if err != nil {
		fatal(err)
	}
	if !write {
		fmt.Printf("verified %s html=%d pdf=%d renderer=%s revision=%s\n", documentArtifactID, len(htmlContent), len(pdf.Content), pdf.RendererProduct, pdf.RendererRevision)
		return
	}
	htmlArtifact, err := service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: newID("art"), MissionID: missionID, MediaType: "text/html; charset=utf-8", Filename: "report.html",
		Producer: app.Producer{Type: "system", ID: "report-il-reproject"}, Content: htmlContent,
	})
	if err != nil {
		fatal(err)
	}
	pdfArtifact, err := service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: newID("art"), MissionID: missionID, MediaType: pdfdocument.MediaType, Filename: "report.pdf",
		Producer: app.Producer{Type: "system", ID: "report-il-reproject"}, Content: pdf.Content,
	})
	if err != nil {
		fatal(err)
	}
	events, err := service.ListEvents(ctx, missionID)
	if err != nil {
		fatal(err)
	}
	var original app.LedgerEvent
	var payload map[string]any
	for index := len(events) - 1; index >= 0; index-- {
		event := events[index]
		if event.EventType != "report.artifact.created" {
			continue
		}
		var candidate map[string]any
		if json.Unmarshal(event.Payload, &candidate) != nil {
			continue
		}
		bundle, _ := candidate["artifact_bundle"].(map[string]any)
		entries, _ := bundle["artifacts"].([]any)
		matched := false
		for _, value := range entries {
			entry, _ := value.(map[string]any)
			if entry["artifact_id"] == documentArtifactID && entry["kind"] == "semantic_il" {
				matched = true
				break
			}
		}
		if matched {
			original = event
			payload = candidate
			break
		}
	}
	if original.EventID == "" {
		fatal(fmt.Errorf("semantic IL terminal lineage not found"))
	}
	bundle, _ := payload["artifact_bundle"].(map[string]any)
	entries, _ := bundle["artifacts"].([]any)
	for _, value := range entries {
		entry, _ := value.(map[string]any)
		switch entry["kind"] {
		case "html":
			entry["artifact_id"], entry["sha256"], entry["byte_size"] = htmlArtifact.ArtifactID, htmlArtifact.SHA256, htmlArtifact.ByteSize
		case "pdf":
			entry["artifact_id"], entry["sha256"], entry["byte_size"] = pdfArtifact.ArtifactID, pdfArtifact.SHA256, pdfArtifact.ByteSize
		}
	}
	payload["text"] = "IL 보고서 HTML과 PDF를 Mermaid SVG 렌더링으로 갱신했습니다."
	payload["source_event_id"] = original.EventID
	payload["projection_repair"] = map[string]any{
		"source_semantic_il_artifact_id": documentArtifactID,
		"renderer_product":               pdf.RendererProduct, "renderer_revision": pdf.RendererRevision,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fatal(err)
	}
	terminal, err := reportilcontract.DecodeTerminalPayload(payloadBytes)
	if err != nil {
		fatal(err)
	}
	if terminal.Bundle.MarkdownArtifactID == "" {
		fatal(fmt.Errorf("corrected terminal lineage lost Markdown binding"))
	}
	eventID := newID("evt")
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: eventID, MissionID: missionID, EventType: "report.artifact.reprojected", Producer: app.Producer{Type: "system", ID: "report-il-reproject"},
		CausationEventID: original.EventID, CorrelationID: original.CorrelationID, Payload: payloadBytes,
	}); err != nil {
		fatal(err)
	}
	fmt.Printf("stored html=%s sha=%s bytes=%d pdf=%s sha=%s bytes=%d event=%s renderer=%s revision=%s\n", htmlArtifact.ArtifactID, htmlArtifact.SHA256, htmlArtifact.ByteSize, pdfArtifact.ArtifactID, pdfArtifact.SHA256, pdfArtifact.ByteSize, eventID, pdf.RendererProduct, pdf.RendererRevision)
}

func newID(prefix string) string {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%s_%s", prefix, time.Now().UTC().Format("20060102150405"), hex.EncodeToString(random[:]))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

type reportILDocumentStore struct {
	fakeStore
	events   []ledger.Event
	artifact artifactcontract.Raw
}

func (store reportILDocumentStore) ListLedgerEvents(context.Context, string) ([]ledger.Event, error) {
	return append([]ledger.Event(nil), store.events...), nil
}

func (store reportILDocumentStore) GetRawArtifact(context.Context, string) (artifactcontract.Raw, error) {
	return store.artifact, nil
}

func TestReadReportILDocumentBindsStageTraceAndStrictArtifact(t *testing.T) {
	catalog := reportILSourceVerifierCatalog(t, 5)
	document := reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.AuthorDocumentSchemaVersion,
		Title:         "Bounded report", Language: "en",
		Sections: []reportilcontract.AuthorSection{
			{SectionKey: "section_001", Title: "Answer", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "section_001.block_001", Kind: "prose", Prose: "The source gives a bounded answer.", EvidenceSourceKeys: []string{"source_001"}}}},
			{SectionKey: "section_002", Title: "Judgment", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "section_002.block_001", Kind: "prose", Prose: "The evidence supports one bounded judgment.", EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
	content, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, '\n')
	sum := sha256.Sum256(content)
	hash := hex.EncodeToString(sum[:])
	artifact := artifactcontract.Raw{
		ArtifactID: "art_publication", MissionID: catalog.MissionID,
		MediaType: reportilcontract.AuthorDocumentMediaType,
		Producer:  ledger.Producer{Type: "mcp_tool", ID: reportilcontract.AuthorDocumentFinalizeTool},
		SHA256:    hash, ByteSize: int64(len(content)), Content: content,
	}
	finalize := reportILDocumentFinalizeEvent(
		"evt_finalize", "ses_publication", "il_reader", artifact,
		"ilw_publication", 3, 1,
	)
	service := NewService(reportILDocumentStore{events: []ledger.Event{finalize}, artifact: artifact})

	got, receipt, err := service.ReadReportILPublicationDocument(
		context.Background(), catalog.MissionID, "ses_publication", catalog,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != document.Title || receipt.ArtifactID != artifact.ArtifactID ||
		receipt.Stage != "il_reader" || receipt.Revision != 3 || receipt.Replacements != 1 {
		t.Fatalf("publication document = %#v / %#v", got, receipt)
	}

	authorOnly := NewService(reportILDocumentStore{events: []ledger.Event{finalize}, artifact: artifact})
	if _, _, err := authorOnly.ReadReportILAuthorDocument(context.Background(), catalog.MissionID, "ses_publication", catalog); err == nil {
		t.Fatal("author reader accepted a publication-stage finalize trace")
	}
	if _, _, err := authorOnly.ReadReportILContinuityDocument(context.Background(), catalog.MissionID, "ses_publication", catalog); err == nil {
		t.Fatal("continuity reader accepted a publication-stage finalize trace")
	}

	reusedLongForm := artifact
	reusedLongForm.ArtifactID = "art_long_form_final"
	reusedLongForm.Producer.ID = reportilcontract.LongFormDocumentFinalizeTool
	reusedLongFormService := NewService(reportILDocumentStore{
		events: []ledger.Event{reportILDocumentFinalizeEvent(
			"evt_reader_noop", "ses_reader_noop", "il_reader", reusedLongForm,
			"ilw_reader_noop", 1, 0,
		)},
		artifact: reusedLongForm,
	})
	if _, receipt, err := reusedLongFormService.ReadReportILPublicationDocument(
		context.Background(), catalog.MissionID, "ses_reader_noop", catalog,
	); err != nil || receipt.ArtifactID != reusedLongForm.ArtifactID || receipt.Replacements != 0 {
		t.Fatalf("reused long-form publication document = %#v / %v", receipt, err)
	}
	replacedLongFormService := NewService(reportILDocumentStore{
		events: []ledger.Event{reportILDocumentFinalizeEvent(
			"evt_reader_replaced", "ses_reader_replaced", "il_reader", reusedLongForm,
			"ilw_reader_replaced", 2, 1,
		)},
		artifact: reusedLongForm,
	})
	if _, _, err := replacedLongFormService.ReadReportILPublicationDocument(
		context.Background(), catalog.MissionID, "ses_reader_replaced", catalog,
	); err == nil {
		t.Fatal("edited publication accepted a long-form producer artifact")
	}

	continuityFinalize := reportILDocumentFinalizeEvent(
		"evt_continuity", "ses_continuity", "il_continuity", artifact,
		"ilw_continuity", 2, 1,
	)
	continuityService := NewService(reportILDocumentStore{events: []ledger.Event{continuityFinalize}, artifact: artifact})
	if _, receipt, err := continuityService.ReadReportILContinuityDocument(context.Background(), catalog.MissionID, "ses_continuity", catalog); err != nil || receipt.Stage != "il_continuity" || receipt.Replacements != 1 {
		t.Fatalf("continuity document = %#v / %v", receipt, err)
	}
	if _, _, err := continuityService.ReadReportILPublicationDocument(context.Background(), catalog.MissionID, "ses_continuity", catalog); err == nil {
		t.Fatal("publication reader accepted a continuity-stage finalize trace")
	}

	duplicate := NewService(reportILDocumentStore{events: []ledger.Event{finalize, reportILDocumentFinalizeEvent(
		"evt_finalize_2", "ses_publication", "il_reader", artifact, "ilw_publication_2", 3, 0,
	)}, artifact: artifact})
	if _, _, err := duplicate.ReadReportILPublicationDocument(context.Background(), catalog.MissionID, "ses_publication", catalog); err == nil {
		t.Fatal("multiple publication finalizations were accepted")
	}

	tampered := artifact
	tampered.Content = append([]byte(nil), artifact.Content...)
	tampered.Content[len(tampered.Content)-2] ^= 1
	tamperedService := NewService(reportILDocumentStore{events: []ledger.Event{finalize}, artifact: tampered})
	if _, _, err := tamperedService.ReadReportILPublicationDocument(context.Background(), catalog.MissionID, "ses_publication", catalog); err == nil || !strings.Contains(err.Error(), "artifact binding is invalid") {
		t.Fatalf("tampered publication document error = %v", err)
	}
}

func TestReadReportILDocumentRejectsUnknownArtifactFields(t *testing.T) {
	catalog := reportILSourceVerifierCatalog(t, 5)
	content := []byte(`{"schema_version":"plasma.report_il.author_document.experimental.v1","title":"Report","language":"en","sections":[],"private":"must-not-persist"}`)
	sum := sha256.Sum256(content)
	artifact := artifactcontract.Raw{
		ArtifactID: "art_author", MissionID: catalog.MissionID,
		MediaType: reportilcontract.AuthorDocumentMediaType,
		Producer:  ledger.Producer{Type: "mcp_tool", ID: reportilcontract.AuthorDocumentFinalizeTool},
		SHA256:    hex.EncodeToString(sum[:]), ByteSize: int64(len(content)), Content: content,
	}
	service := NewService(reportILDocumentStore{
		events:   []ledger.Event{reportILDocumentFinalizeEvent("evt_finalize", "ses_author", "il_narrative", artifact, "ilw_author", 1, 0)},
		artifact: artifact,
	})
	if _, _, err := service.ReadReportILAuthorDocument(context.Background(), catalog.MissionID, "ses_author", catalog); err == nil || !strings.Contains(err.Error(), "decode failed") {
		t.Fatalf("unknown author artifact field error = %v", err)
	}
}

func reportILDocumentFinalizeEvent(eventID, sessionID, stage string, artifact artifactcontract.Raw, workspaceID string, revision, replacements int) ledger.Event {
	payload, _ := json.Marshal(map[string]any{
		"tool_name":       reportilcontract.AuthorDocumentFinalizeTool,
		"tool_session_id": sessionID,
		"success":         true,
		"io_metrics": map[string]any{
			"report_il_stage": stage, "workspace_id": workspaceID,
			"artifact_id": artifact.ArtifactID, "sha256": artifact.SHA256,
			"byte_size": artifact.ByteSize, "revision": revision,
			"replacements": replacements, "finalized": true,
		},
	})
	return ledger.Event{
		EventID: eventID, MissionID: artifact.MissionID, EventType: "mcp.tool.called",
		CorrelationID: sessionID, Payload: payload,
	}
}

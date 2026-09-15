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

func TestReadReportILEditorialMemoryBindsFinalizeTraceAndArtifactBytes(t *testing.T) {
	catalog := reportILSourceVerifierCatalog(t, 5)
	memory := reportilcontract.EditorialMemory{
		SchemaVersion: reportilcontract.EditorialMemorySchemaVersion,
		Language:      "en",
		Accounts: []reportilcontract.EditorialAccount{{
			AccountKey: "account_001", Importance: "essential",
			Account: "The source gives one connected account.", SourceKeys: []string{"source_001"},
		}},
	}
	memoryArtifact := reportilcontract.EditorialMemoryArtifact{
		SchemaVersion: memory.SchemaVersion,
		Language:      memory.Language,
		Accounts:      memory.Accounts,
		Anchors: []reportilcontract.EditorialAnchor{{
			AccountKey: "account_001", SourceKey: "source_001", Excerpt: "x",
			Offset: 0, ByteSize: 1, SHA256: sha256Hex("x"),
		}},
	}
	content, err := json.Marshal(memoryArtifact)
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, '\n')
	sum := sha256.Sum256(content)
	artifact := artifactcontract.Raw{
		ArtifactID: "art_editorial_memory", MissionID: catalog.MissionID,
		MediaType: reportilcontract.EditorialMemoryMediaType,
		SHA256:    hex.EncodeToString(sum[:]), ByteSize: int64(len(content)), Content: content,
	}
	finalize := reportILEditorialMemoryFinalizeEvent("evt_memory_finalize", "ses_memory", artifact, "ilm_memory", 2, 1)
	service := NewService(reportILDocumentStore{events: []ledger.Event{finalize}, artifact: artifact})
	got, receipt, err := service.ReadReportILEditorialMemory(context.Background(), catalog.MissionID, "ses_memory", catalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Accounts) != 1 || receipt.ArtifactID != artifact.ArtifactID || receipt.Revision != 2 || receipt.Accounts != 1 {
		t.Fatalf("editorial memory = %#v / %#v", got, receipt)
	}

	tampered := artifact
	tampered.Content = append([]byte(nil), artifact.Content...)
	accountOffset := strings.Index(string(tampered.Content), "connected")
	if accountOffset < 0 {
		t.Fatal("test memory account not found")
	}
	copy(tampered.Content[accountOffset:], []byte("corrupted"))
	tamperedService := NewService(reportILDocumentStore{events: []ledger.Event{finalize}, artifact: tampered})
	if _, _, err := tamperedService.ReadReportILEditorialMemory(context.Background(), catalog.MissionID, "ses_memory", catalog); err == nil || !strings.Contains(err.Error(), "artifact binding is invalid") {
		t.Fatalf("tampered editorial memory error = %v", err)
	}

	duplicate := NewService(reportILDocumentStore{events: []ledger.Event{finalize, reportILEditorialMemoryFinalizeEvent(
		"evt_memory_finalize_2", "ses_memory", artifact, "ilm_memory_2", 2, 1,
	)}, artifact: artifact})
	if _, _, err := duplicate.ReadReportILEditorialMemory(context.Background(), catalog.MissionID, "ses_memory", catalog); err == nil {
		t.Fatal("multiple editorial memory finalizations were accepted")
	}
}

func reportILEditorialMemoryFinalizeEvent(eventID, sessionID string, artifact artifactcontract.Raw, workspaceID string, revision, accounts int) ledger.Event {
	payload, _ := json.Marshal(map[string]any{
		"tool_name":       reportilcontract.EditorialMemoryFinalizeTool,
		"tool_session_id": sessionID,
		"success":         true,
		"io_metrics": map[string]any{
			"report_il_stage": "il_editorial_memory", "workspace_id": workspaceID,
			"artifact_id": artifact.ArtifactID, "sha256": artifact.SHA256,
			"byte_size": artifact.ByteSize, "revision": revision,
			"accounts": accounts, "finalized": true,
		},
	})
	return ledger.Event{
		EventID: eventID, MissionID: artifact.MissionID, EventType: "mcp.tool.called",
		CorrelationID: sessionID, Payload: payload,
	}
}

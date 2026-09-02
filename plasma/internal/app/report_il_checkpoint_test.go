package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

type reportILCheckpointStore struct {
	reportILDocumentStore
	artifacts map[string]RawArtifact
}

func (store *reportILCheckpointStore) GetRawArtifact(_ context.Context, artifactID string) (RawArtifact, error) {
	return store.artifacts[artifactID], nil
}

func TestLoadReportILResumeCheckpointRejectsTamperedArtifact(t *testing.T) {
	catalog := reportILSourceVerifierCatalog(t, 5)
	memoryContent, memory := checkpointMemory(t, catalog)
	documentContent, document := checkpointDocument(t, catalog)
	memoryArtifact := checkpointRawArtifact("art_memory", catalog.MissionID, reportilcontract.EditorialMemoryMediaType, reportilcontract.EditorialMemoryFinalizeTool, memoryContent)
	documentArtifact := checkpointRawArtifact("art_final", catalog.MissionID, reportilcontract.AuthorDocumentMediaType, reportilcontract.LongFormDocumentFinalizeTool, documentContent)
	checkpoint := checkpointAppValue(catalog, memoryArtifact, documentArtifact, len(memory.Accounts))
	payload, _ := json.Marshal(map[string]any{"pending_event_id": "evt_failed", "checkpoint": checkpoint})
	store := &reportILCheckpointStore{
		reportILDocumentStore: reportILDocumentStore{
			events: []LedgerEvent{{EventID: "evt_checkpoint", MissionID: catalog.MissionID, EventType: reportILCheckpointEventType, CausationEventID: "evt_failed", Payload: payload}},
		},
		artifacts: map[string]RawArtifact{memoryArtifact.ArtifactID: memoryArtifact, documentArtifact.ArtifactID: documentArtifact},
	}
	service := NewService(store)
	resume, err := service.LoadReportILResumeCheckpoint(context.Background(), catalog.MissionID, "evt_failed")
	if err != nil || resume.AuthorDocument.Title != document.Title {
		t.Fatalf("load checkpoint: resume=%#v err=%v", resume, err)
	}
	tampered := documentArtifact
	tampered.Content = append([]byte(nil), tampered.Content...)
	tampered.Content[0] ^= 0xff
	store.artifacts[documentArtifact.ArtifactID] = tampered
	if _, err := service.LoadReportILResumeCheckpoint(context.Background(), catalog.MissionID, "evt_failed"); err == nil {
		t.Fatal("checkpoint accepted tampered author artifact")
	}
}

func checkpointMemory(t *testing.T, catalog reportilcontract.SourceCatalog) ([]byte, reportilcontract.EditorialMemory) {
	t.Helper()
	memory := reportilcontract.EditorialMemory{
		SchemaVersion: reportilcontract.EditorialMemorySchemaVersion, Language: "ko",
		Accounts: []reportilcontract.EditorialAccount{{AccountKey: "account_001", Importance: "essential", Account: "근거 계정", SourceKeys: []string{"source_001"}}},
	}
	entry := catalog.Sources[0]
	excerpt := "12345"
	anchorSum := sha256.Sum256([]byte(excerpt))
	stored := reportilcontract.EditorialMemoryArtifact{
		SchemaVersion: memory.SchemaVersion, Language: memory.Language, Accounts: memory.Accounts,
		Anchors: []reportilcontract.EditorialAnchor{{AccountKey: "account_001", SourceKey: "source_001", Excerpt: excerpt, Offset: 0, ByteSize: len(excerpt), SHA256: hex.EncodeToString(anchorSum[:])}},
	}
	content, err := json.Marshal(stored)
	if err != nil || entry.ReadableBytes < len(excerpt) {
		t.Fatal("memory fixture is invalid")
	}
	return content, memory
}

func checkpointDocument(t *testing.T, catalog reportilcontract.SourceCatalog) ([]byte, reportilcontract.AuthorDocument) {
	t.Helper()
	document := reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.LongFormAuthorDocumentSchemaVersion, Title: "재시도 보고서", Language: "ko",
		Parts: []reportilcontract.AuthorPart{
			{PartKey: "part_001", Title: "1부", Sections: []reportilcontract.AuthorSection{
				{SectionKey: "part_001.section_001", Title: "1", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "part_001.section_001.block_001", Kind: "prose", Prose: "첫 문단", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}}}},
				{SectionKey: "part_001.section_002", Title: "2", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "part_001.section_002.block_001", Kind: "prose", Prose: "둘째 문단", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}}}},
				{SectionKey: "part_001.section_003", Title: "3", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "part_001.section_003.block_001", Kind: "prose", Prose: "셋째 문단", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}}}},
			}},
			{PartKey: "part_002", Title: "2부", Sections: []reportilcontract.AuthorSection{
				{SectionKey: "part_002.section_001", Title: "4", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "part_002.section_001.block_001", Kind: "prose", Prose: "넷째 문단", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}}}},
				{SectionKey: "part_002.section_002", Title: "5", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "part_002.section_002.block_001", Kind: "prose", Prose: "다섯째 문단", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}}}},
				{SectionKey: "part_002.section_003", Title: "6", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "part_002.section_003.block_001", Kind: "prose", Prose: "결론 문단", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}}}},
			}},
		},
	}
	content, err := json.Marshal(document)
	if err != nil || reportilcontract.ValidateLongFormAuthorDocument(document, catalog) != nil {
		t.Fatal("document fixture is invalid")
	}
	return content, document
}

func checkpointRawArtifact(id, missionID, mediaType, producerID string, content []byte) RawArtifact {
	sum := sha256.Sum256(content)
	return RawArtifact{ArtifactID: id, MissionID: missionID, MediaType: mediaType, ByteSize: int64(len(content)), SHA256: hex.EncodeToString(sum[:]), Producer: Producer{Type: "mcp_tool", ID: producerID}, Content: content}
}

func checkpointAppValue(catalog reportilcontract.SourceCatalog, memory, document RawArtifact, accounts int) reportilcontract.ProductCheckpoint {
	workspace := reportilcontract.AuthorWorkspaceReceipt{ArtifactID: document.ArtifactID, SHA256: document.SHA256, ByteSize: int(document.ByteSize), Revision: 1, Stage: "il_long_form_final"}
	artifact := reportilcontract.CheckpointArtifact{ArtifactID: workspace.ArtifactID, SHA256: workspace.SHA256, ByteSize: workspace.ByteSize, Stage: workspace.Stage}
	return reportilcontract.ProductCheckpoint{
		SchemaVersion: reportilcontract.ProductCheckpointSchemaVersion, PendingEventID: "evt_failed", Stage: workspace.Stage, ArtifactID: workspace.ArtifactID,
		CandidateCatalogSHA256: catalog.SHA256, AuthorCatalog: catalog,
		SourceSelection:   reportilcontract.CheckpointSourceSelection{CandidateCatalogSHA256: catalog.SHA256, SelectedCatalogSHA256: catalog.SHA256, AuthorCatalogSHA256: catalog.SHA256},
		EditorialMemory:   reportilcontract.CheckpointEditorialMemory{ArtifactID: memory.ArtifactID, SHA256: memory.SHA256, ByteSize: int(memory.ByteSize), Revision: 1, Accounts: accounts},
		AuthorWorkspace:   workspace,
		LongFormAuthoring: reportilcontract.CheckpointLongFormAuthoring{Final: artifact, Finalizations: []reportilcontract.CheckpointFinalization{{Stage: workspace.Stage, Artifact: artifact}}, Parts: 2, Sections: 6},
	}
}

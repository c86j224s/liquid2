package reportilcontract

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/source"
)

func TestValidateProductCheckpointRejectsUnboundFinalization(t *testing.T) {
	catalog := checkpointTestCatalog(t)
	checkpoint := checkpointTestValue(catalog)
	checkpoint.LongFormAuthoring.Finalizations = nil
	if err := ValidateProductCheckpoint(checkpoint, catalog.MissionID); err == nil {
		t.Fatal("checkpoint accepted a final artifact without stage finalization lineage")
	}
}

func TestValidateProductCheckpointAcceptsBoundLongFormFinal(t *testing.T) {
	catalog := checkpointTestCatalog(t)
	checkpoint := checkpointTestValue(catalog)
	if err := ValidateProductCheckpoint(checkpoint, catalog.MissionID); err != nil {
		t.Fatalf("valid checkpoint = %v", err)
	}
}

func TestValidateProductCheckpointAcceptsCompleteParts(t *testing.T) {
	catalog := checkpointTestCatalog(t)
	checkpoint := checkpointTestValue(catalog)
	checkpoint.Stage, checkpoint.ArtifactID, checkpoint.AuthorWorkspace = "il_long_form_parts", "", AuthorWorkspaceReceipt{}
	checkpoint.LongFormAuthoring.Final = CheckpointArtifact{}
	checkpoint.LongFormAuthoring.Finalizations = nil
	checkpoint.LongFormAuthoring.Plan = CheckpointArtifact{ArtifactID: "art_plan", SHA256: strings.Repeat("d", 64), ByteSize: 50, Stage: "il_long_form_plan"}
	for index := 0; index < MinLongFormSections; index++ {
		checkpoint.LongFormAuthoring.SectionArtifacts = append(checkpoint.LongFormAuthoring.SectionArtifacts, CheckpointArtifact{ArtifactID: "art_section_" + string(rune('a'+index)), SHA256: strings.Repeat("e", 64), ByteSize: 50, Stage: "il_long_form_section"})
	}
	for index := 0; index < MinLongFormParts; index++ {
		checkpoint.LongFormAuthoring.PartArtifacts = append(checkpoint.LongFormAuthoring.PartArtifacts, CheckpointArtifact{ArtifactID: "art_part_" + string(rune('a'+index)), SHA256: strings.Repeat("f", 64), ByteSize: 50, Stage: "il_long_form_part"})
	}
	if err := ValidateProductCheckpoint(checkpoint, catalog.MissionID); err != nil {
		t.Fatalf("valid Part checkpoint = %v", err)
	}
	checkpoint.LongFormAuthoring.PartArtifacts = checkpoint.LongFormAuthoring.PartArtifacts[:1]
	if err := ValidateProductCheckpoint(checkpoint, catalog.MissionID); err == nil {
		t.Fatal("incomplete Part checkpoint was accepted")
	}
}

func checkpointTestCatalog(t *testing.T) SourceCatalog {
	t.Helper()
	catalog, err := SealSourceCatalog(SourceCatalog{MissionID: "mis_checkpoint", Sources: []SourceCatalogEntry{{
		SourceKey: "source_001", AcceptedOrdinal: 1, SnapshotID: "src_1",
		SnapshotReceipt: SourceSnapshotReceipt("src_1", strings.Repeat("a", 64)),
		ContentHash:     strings.Repeat("a", 64), RetrievalPolicy: source.RetrievalPolicySnapshotOnly,
		Artifacts:      []SourceCatalogArtifact{{ArtifactID: "art_source", SHA256: strings.Repeat("a", 64), ByteSize: 10, MediaType: "text/plain"}},
		ReadableSHA256: strings.Repeat("a", 64), ReadableBytes: 10, Extraction: "stored_text",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func checkpointTestValue(catalog SourceCatalog) ProductCheckpoint {
	workspace := AuthorWorkspaceReceipt{
		ArtifactID: "art_final", SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ByteSize: 100, Revision: 1, Stage: "il_long_form_final",
	}
	artifact := CheckpointArtifact{
		ArtifactID: workspace.ArtifactID, SHA256: workspace.SHA256,
		ByteSize: workspace.ByteSize, Stage: workspace.Stage,
	}
	return ProductCheckpoint{
		SchemaVersion:  ProductCheckpointSchemaVersion,
		PendingEventID: "evt_pending", Stage: "il_long_form_final", ArtifactID: workspace.ArtifactID,
		CandidateCatalogSHA256: catalog.SHA256, AuthorCatalog: catalog,
		SourceSelection: CheckpointSourceSelection{
			CandidateCatalogSHA256: catalog.SHA256, SelectedCatalogSHA256: catalog.SHA256, AuthorCatalogSHA256: catalog.SHA256,
		},
		EditorialMemory: CheckpointEditorialMemory{
			ArtifactID: "art_memory", SHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			ByteSize: 50, Revision: 1, Accounts: 1,
		},
		AuthorWorkspace: workspace,
		LongFormAuthoring: CheckpointLongFormAuthoring{
			Final: artifact, Finalizations: []CheckpointFinalization{{Stage: workspace.Stage, Artifact: artifact}},
			Parts: MinLongFormParts, Sections: MinLongFormSections,
		},
	}
}

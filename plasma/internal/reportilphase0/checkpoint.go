package reportilphase0

import "github.com/c86j224s/liquid2/plasma/internal/reportilcontract"

func checkpointSourceSelection(receipt SourceSelectionReceipt) reportilcontract.CheckpointSourceSelection {
	dispositions := make([]reportilcontract.CheckpointSourceDisposition, len(receipt.Dispositions))
	for index, disposition := range receipt.Dispositions {
		dispositions[index] = reportilcontract.CheckpointSourceDisposition{
			AcceptedOrdinal: disposition.AcceptedOrdinal,
			Status:          disposition.Status,
			Reason:          disposition.Reason,
		}
	}
	return reportilcontract.CheckpointSourceSelection{
		Applied: receipt.Applied, AcceptedSources: receipt.AcceptedSources,
		UsableSources: receipt.UsableSources, SelectedSources: receipt.SelectedSources,
		SupplementalSources:       receipt.SupplementalSources,
		ExcludedUnusableSources:   receipt.ExcludedUnusableSources,
		ExcludedBudgetSources:     receipt.ExcludedBudgetSources,
		SelectedReadableBytes:     receipt.SelectedReadableBytes,
		SupplementalReadableBytes: receipt.SupplementalReadableBytes,
		CandidateCatalogSHA256:    receipt.CandidateCatalogSHA256,
		SelectedCatalogSHA256:     receipt.SelectedCatalogSHA256,
		AuthorCatalogSHA256:       receipt.AuthorCatalogSHA256,
		DispositionSHA256:         receipt.DispositionSHA256, Dispositions: dispositions,
	}
}

func sourceSelectionFromCheckpoint(receipt reportilcontract.CheckpointSourceSelection) SourceSelectionReceipt {
	dispositions := make([]SourceSelectionDisposition, len(receipt.Dispositions))
	for index, disposition := range receipt.Dispositions {
		dispositions[index] = SourceSelectionDisposition{
			AcceptedOrdinal: disposition.AcceptedOrdinal,
			Status:          disposition.Status,
			Reason:          disposition.Reason,
		}
	}
	return SourceSelectionReceipt{
		Applied: receipt.Applied, AcceptedSources: receipt.AcceptedSources,
		UsableSources: receipt.UsableSources, SelectedSources: receipt.SelectedSources,
		SupplementalSources:       receipt.SupplementalSources,
		ExcludedUnusableSources:   receipt.ExcludedUnusableSources,
		ExcludedBudgetSources:     receipt.ExcludedBudgetSources,
		SelectedReadableBytes:     receipt.SelectedReadableBytes,
		SupplementalReadableBytes: receipt.SupplementalReadableBytes,
		CandidateCatalogSHA256:    receipt.CandidateCatalogSHA256,
		SelectedCatalogSHA256:     receipt.SelectedCatalogSHA256,
		AuthorCatalogSHA256:       receipt.AuthorCatalogSHA256,
		DispositionSHA256:         receipt.DispositionSHA256, Dispositions: dispositions,
	}
}

func checkpointArtifact(receipt LongFormArtifactReceipt) reportilcontract.CheckpointArtifact {
	return reportilcontract.CheckpointArtifact{
		ArtifactID: receipt.ArtifactID, SHA256: receipt.SHA256,
		ByteSize: receipt.ByteSize, Stage: receipt.Stage,
	}
}

func artifactFromCheckpoint(receipt reportilcontract.CheckpointArtifact) LongFormArtifactReceipt {
	return LongFormArtifactReceipt{
		ArtifactID: receipt.ArtifactID, SHA256: receipt.SHA256,
		ByteSize: receipt.ByteSize, Stage: receipt.Stage,
	}
}

func checkpointLongForm(receipt LongFormAuthoringReceipt) reportilcontract.CheckpointLongFormAuthoring {
	out := reportilcontract.CheckpointLongFormAuthoring{
		Plan: checkpointArtifact(receipt.Plan), Final: checkpointArtifact(receipt.Final),
		Parts: receipt.Parts, Sections: receipt.Sections,
		SectionAuthors: receipt.SectionAuthors, PartEditors: receipt.PartEditors,
		FinalEdits: receipt.FinalEdits,
	}
	for _, artifact := range receipt.SectionArtifacts {
		out.SectionArtifacts = append(out.SectionArtifacts, checkpointArtifact(artifact))
	}
	for _, artifact := range receipt.PartArtifacts {
		out.PartArtifacts = append(out.PartArtifacts, checkpointArtifact(artifact))
	}
	for _, finalization := range receipt.Finalizations {
		out.Finalizations = append(out.Finalizations, reportilcontract.CheckpointFinalization{
			Stage: finalization.Stage, Artifact: checkpointArtifact(finalization.Artifact), Reused: finalization.Reused,
		})
	}
	return out
}

func longFormFromCheckpoint(receipt reportilcontract.CheckpointLongFormAuthoring) LongFormAuthoringReceipt {
	out := LongFormAuthoringReceipt{
		Plan: artifactFromCheckpoint(receipt.Plan), Final: artifactFromCheckpoint(receipt.Final),
		Parts: receipt.Parts, Sections: receipt.Sections,
		SectionAuthors: receipt.SectionAuthors, PartEditors: receipt.PartEditors,
		FinalEdits: receipt.FinalEdits,
	}
	for _, artifact := range receipt.SectionArtifacts {
		out.SectionArtifacts = append(out.SectionArtifacts, artifactFromCheckpoint(artifact))
	}
	for _, artifact := range receipt.PartArtifacts {
		out.PartArtifacts = append(out.PartArtifacts, artifactFromCheckpoint(artifact))
	}
	for _, finalization := range receipt.Finalizations {
		out.Finalizations = append(out.Finalizations, LongFormFinalizationReceipt{
			Stage: finalization.Stage, Artifact: artifactFromCheckpoint(finalization.Artifact), Reused: finalization.Reused,
		})
	}
	return out
}

func checkpointReader(receipt ReaderFinalizationReceipt) reportilcontract.CheckpointReaderFinalization {
	return reportilcontract.CheckpointReaderFinalization{
		ContinuityPatches: receipt.ContinuityPatches, PublicationPatches: receipt.PublicationPatches,
		ProposedPatches: receipt.ProposedPatches, AcceptedPatches: receipt.AcceptedPatches,
		RejectedPatches: receipt.RejectedPatches, Applied: receipt.Applied,
	}
}

func readerFromCheckpoint(receipt reportilcontract.CheckpointReaderFinalization) ReaderFinalizationReceipt {
	return ReaderFinalizationReceipt{
		ContinuityPatches: receipt.ContinuityPatches, PublicationPatches: receipt.PublicationPatches,
		ProposedPatches: receipt.ProposedPatches, AcceptedPatches: receipt.AcceptedPatches,
		RejectedPatches: receipt.RejectedPatches, Applied: receipt.Applied,
	}
}

// NewPartsCheckpoint freezes every validated Section and Part before the final
// manuscript pass so a failed final editor can resume without re-authoring them.
func NewPartsCheckpoint(
	config ProductConfig,
	candidateCatalogSHA string,
	authorCatalog reportilcontract.SourceCatalog,
	imageCatalog reportilcontract.SourceCatalog,
	selection SourceSelectionReceipt,
	memoryReceipt reportilcontract.EditorialMemoryReceipt,
	planReceipt reportilcontract.LongFormPlanReceipt,
	sectionArtifacts,
	partArtifacts []reportilcontract.AuthorWorkspaceReceipt,
	parts,
	sections int,
) reportilcontract.ProductCheckpoint {
	longForm := reportilcontract.CheckpointLongFormAuthoring{
		Plan: checkpointArtifact(longFormPlanArtifactReceipt(planReceipt)), Parts: parts, Sections: sections,
		SectionAuthors: sections, PartEditors: parts,
	}
	for _, receipt := range sectionArtifacts {
		longForm.SectionArtifacts = append(longForm.SectionArtifacts, checkpointArtifact(longFormArtifactReceipt(receipt)))
	}
	for _, receipt := range partArtifacts {
		longForm.PartArtifacts = append(longForm.PartArtifacts, checkpointArtifact(longFormArtifactReceipt(receipt)))
	}
	return reportilcontract.ProductCheckpoint{
		SchemaVersion:  reportilcontract.ProductCheckpointSchemaVersion,
		PendingEventID: config.PendingEventID, Stage: "il_long_form_parts",
		CandidateCatalogSHA256: candidateCatalogSHA, AuthorCatalog: authorCatalog,
		ImageCatalog: imageCatalog, SourceSelection: checkpointSourceSelection(selection),
		EditorialMemory: reportilcontract.CheckpointEditorialMemory{
			ArtifactID: memoryReceipt.ArtifactID, SHA256: memoryReceipt.SHA256,
			ByteSize: memoryReceipt.ByteSize, Revision: memoryReceipt.Revision, Accounts: memoryReceipt.Accounts,
		},
		LongFormAuthoring: longForm,
	}
}

func newProductCheckpoint(
	config ProductConfig,
	stage string,
	candidateCatalogSHA string,
	authorCatalog reportilcontract.SourceCatalog,
	imageCatalog reportilcontract.SourceCatalog,
	selection SourceSelectionReceipt,
	memoryReceipt reportilcontract.EditorialMemoryReceipt,
	authorWorkspace reportilcontract.AuthorWorkspaceReceipt,
	longForm LongFormAuthoringReceipt,
	reader ReaderFinalizationReceipt,
) reportilcontract.ProductCheckpoint {
	return reportilcontract.ProductCheckpoint{
		SchemaVersion:  reportilcontract.ProductCheckpointSchemaVersion,
		PendingEventID: config.PendingEventID, Stage: stage, ArtifactID: authorWorkspace.ArtifactID,
		CandidateCatalogSHA256: candidateCatalogSHA, AuthorCatalog: authorCatalog,
		ImageCatalog: imageCatalog, SourceSelection: checkpointSourceSelection(selection),
		EditorialMemory: reportilcontract.CheckpointEditorialMemory{
			ArtifactID: memoryReceipt.ArtifactID, SHA256: memoryReceipt.SHA256,
			ByteSize: memoryReceipt.ByteSize, Revision: memoryReceipt.Revision, Accounts: memoryReceipt.Accounts,
		},
		AuthorWorkspace: authorWorkspace, LongFormAuthoring: checkpointLongForm(longForm),
		ReaderFinalization: checkpointReader(reader),
	}
}

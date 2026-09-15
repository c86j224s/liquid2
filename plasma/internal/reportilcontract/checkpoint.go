package reportilcontract

import (
	"fmt"
	"strings"
)

const ProductCheckpointSchemaVersion = "plasma.report_il.product_checkpoint.experimental.v1"

type ProductCheckpoint struct {
	SchemaVersion          string                       `json:"schema_version"`
	PendingEventID         string                       `json:"pending_event_id"`
	Stage                  string                       `json:"stage"`
	ArtifactID             string                       `json:"artifact_id"`
	CandidateCatalogSHA256 string                       `json:"candidate_catalog_sha256"`
	AuthorCatalog          SourceCatalog                `json:"author_catalog"`
	ImageCatalog           SourceCatalog                `json:"image_catalog"`
	SourceSelection        CheckpointSourceSelection    `json:"source_selection"`
	EditorialMemory        CheckpointEditorialMemory    `json:"editorial_memory"`
	AuthorWorkspace        AuthorWorkspaceReceipt       `json:"author_workspace"`
	LongFormAuthoring      CheckpointLongFormAuthoring  `json:"long_form_authoring"`
	ReaderFinalization     CheckpointReaderFinalization `json:"reader_finalization"`
}

type CheckpointSourceSelection struct {
	Applied                   bool                          `json:"applied"`
	AcceptedSources           int                           `json:"accepted_sources"`
	UsableSources             int                           `json:"usable_sources"`
	SelectedSources           int                           `json:"selected_sources"`
	SupplementalSources       int                           `json:"supplemental_sources"`
	ExcludedUnusableSources   int                           `json:"excluded_unusable_sources"`
	ExcludedBudgetSources     int                           `json:"excluded_budget_sources"`
	SelectedReadableBytes     int                           `json:"selected_readable_bytes"`
	SupplementalReadableBytes int                           `json:"supplemental_readable_bytes"`
	CandidateCatalogSHA256    string                        `json:"candidate_catalog_sha256"`
	SelectedCatalogSHA256     string                        `json:"selected_catalog_sha256"`
	AuthorCatalogSHA256       string                        `json:"author_catalog_sha256"`
	DispositionSHA256         string                        `json:"disposition_sha256"`
	Dispositions              []CheckpointSourceDisposition `json:"dispositions"`
}

type CheckpointSourceDisposition struct {
	AcceptedOrdinal int    `json:"accepted_ordinal"`
	Status          string `json:"status"`
	Reason          string `json:"reason,omitempty"`
}

type CheckpointEditorialMemory struct {
	ArtifactID string `json:"artifact_id"`
	SHA256     string `json:"sha256"`
	ByteSize   int    `json:"byte_size"`
	Revision   int    `json:"revision"`
	Accounts   int    `json:"accounts"`
}

type CheckpointArtifact struct {
	ArtifactID string `json:"artifact_id"`
	SHA256     string `json:"sha256"`
	ByteSize   int    `json:"byte_size"`
	Stage      string `json:"stage"`
}

type CheckpointFinalization struct {
	Stage    string             `json:"stage"`
	Artifact CheckpointArtifact `json:"artifact"`
	Reused   bool               `json:"reused"`
}

type CheckpointLongFormAuthoring struct {
	Plan             CheckpointArtifact       `json:"plan"`
	SectionArtifacts []CheckpointArtifact     `json:"section_artifacts"`
	PartArtifacts    []CheckpointArtifact     `json:"part_artifacts"`
	Finalizations    []CheckpointFinalization `json:"finalizations"`
	Final            CheckpointArtifact       `json:"final"`
	Parts            int                      `json:"parts"`
	Sections         int                      `json:"sections"`
	SectionAuthors   int                      `json:"section_authors"`
	PartEditors      int                      `json:"part_editors"`
	FinalEdits       int                      `json:"final_edits"`
}

type CheckpointReaderFinalization struct {
	ContinuityPatches  int  `json:"continuity_patches"`
	PublicationPatches int  `json:"publication_patches"`
	ProposedPatches    int  `json:"proposed_patches"`
	AcceptedPatches    int  `json:"accepted_patches"`
	RejectedPatches    int  `json:"rejected_patches"`
	Applied            bool `json:"applied"`
}

type ResumeCheckpoint struct {
	ProductCheckpoint
	AuthorDocument  AuthorDocument  `json:"-"`
	EditorialMemory EditorialMemory `json:"-"`
	LongFormPlan    LongFormPlan    `json:"-"`
}

func (checkpoint CheckpointEditorialMemory) Receipt() EditorialMemoryReceipt {
	return EditorialMemoryReceipt{
		ArtifactID: checkpoint.ArtifactID,
		SHA256:     checkpoint.SHA256,
		ByteSize:   checkpoint.ByteSize,
		Revision:   checkpoint.Revision,
		Accounts:   checkpoint.Accounts,
	}
}

func ValidateProductCheckpoint(checkpoint ProductCheckpoint, missionID string) error {
	if checkpoint.SchemaVersion != ProductCheckpointSchemaVersion ||
		!strings.HasPrefix(strings.TrimSpace(checkpoint.PendingEventID), "evt_") ||
		checkpoint.AuthorCatalog.MissionID != missionID || len(checkpoint.CandidateCatalogSHA256) != 64 ||
		checkpoint.ArtifactID != checkpoint.AuthorWorkspace.ArtifactID {
		return fmt.Errorf("report IL checkpoint envelope is invalid")
	}
	switch checkpoint.Stage {
	case "il_source_selection", "il_long_form_parts", "il_long_form_final", "il_reader", "il_continuity":
	default:
		return fmt.Errorf("report IL checkpoint stage is invalid")
	}
	if err := ValidateSourceCatalog(checkpoint.AuthorCatalog); err != nil {
		return err
	}
	if checkpoint.SourceSelection.Applied {
		if checkpoint.SourceSelection.SelectedCatalogSHA256 == "" {
			return fmt.Errorf("report IL checkpoint selected-source receipt is incomplete")
		}
	} else if checkpoint.SourceSelection.SelectedCatalogSHA256 != checkpoint.AuthorCatalog.SHA256 {
		return fmt.Errorf("report IL checkpoint unselected catalog binding is invalid")
	}
	if len(checkpoint.ImageCatalog.Sources) > 0 {
		if checkpoint.ImageCatalog.MissionID != missionID {
			return fmt.Errorf("report IL checkpoint image catalog belongs to another mission")
		}
		if err := ValidateSourceCatalog(checkpoint.ImageCatalog); err != nil {
			return err
		}
	}
	selection := checkpoint.SourceSelection
	if selection.AuthorCatalogSHA256 != checkpoint.AuthorCatalog.SHA256 ||
		selection.CandidateCatalogSHA256 != checkpoint.CandidateCatalogSHA256 {
		return fmt.Errorf("report IL checkpoint source selection binding is invalid")
	}
	if selection.AcceptedSources != selection.UsableSources+selection.ExcludedUnusableSources ||
		selection.UsableSources != selection.SelectedSources+selection.SupplementalSources+selection.ExcludedBudgetSources ||
		selection.Applied != (selection.SupplementalSources > 0 || selection.ExcludedUnusableSources+selection.ExcludedBudgetSources > 0) {
		return fmt.Errorf("report IL checkpoint source selection counts are inconsistent")
	}
	memory := checkpoint.EditorialMemory
	if checkpoint.Stage == "il_source_selection" {
		if selection.AcceptedSources < 1 || len(selection.Dispositions) != selection.AcceptedSources || len(selection.DispositionSHA256) != 64 {
			return fmt.Errorf("report IL source-selection checkpoint receipt is incomplete")
		}
		workspace := checkpoint.AuthorWorkspace
		longForm := checkpoint.LongFormAuthoring
		reader := checkpoint.ReaderFinalization
		if checkpoint.ArtifactID != "" || memory.ArtifactID != "" || memory.SHA256 != "" || memory.ByteSize != 0 || memory.Revision != 0 || memory.Accounts != 0 ||
			workspace.ArtifactID != "" || workspace.SHA256 != "" || workspace.ByteSize != 0 || workspace.Revision != 0 || workspace.Stage != "" ||
			longForm.Plan.ArtifactID != "" || len(longForm.SectionArtifacts) != 0 || len(longForm.PartArtifacts) != 0 || len(longForm.Finalizations) != 0 || longForm.Final.ArtifactID != "" || longForm.Parts != 0 || longForm.Sections != 0 || longForm.SectionAuthors != 0 || longForm.PartEditors != 0 || longForm.FinalEdits != 0 ||
			reader.ContinuityPatches != 0 || reader.PublicationPatches != 0 || reader.ProposedPatches != 0 || reader.AcceptedPatches != 0 || reader.RejectedPatches != 0 || reader.Applied {
			return fmt.Errorf("report IL source-selection checkpoint lineage is invalid")
		}
		return nil
	}
	if !strings.HasPrefix(memory.ArtifactID, "art_") || len(memory.SHA256) != 64 ||
		memory.ByteSize < 1 || memory.Revision < 1 || memory.Accounts < 1 {
		return fmt.Errorf("report IL checkpoint editorial memory is invalid")
	}
	workspace := checkpoint.AuthorWorkspace
	parts := checkpoint.LongFormAuthoring.Parts
	sections := checkpoint.LongFormAuthoring.Sections
	if parts < MinLongFormParts || parts > MaxLongFormParts || sections < MinLongFormSections || sections > MaxLongFormSections {
		return fmt.Errorf("report IL checkpoint long-form inventory is invalid")
	}
	if checkpoint.Stage == "il_long_form_parts" {
		if checkpoint.ArtifactID != "" || workspace.ArtifactID != "" || workspace.SHA256 != "" || workspace.ByteSize != 0 || workspace.Revision != 0 || workspace.Stage != "" ||
			checkpoint.LongFormAuthoring.Final.ArtifactID != "" || len(checkpoint.LongFormAuthoring.Finalizations) != 0 ||
			len(checkpoint.LongFormAuthoring.PartArtifacts) != parts || len(checkpoint.LongFormAuthoring.SectionArtifacts) != sections {
			return fmt.Errorf("report IL Part checkpoint lineage is invalid")
		}
		if err := validateCheckpointArtifact(checkpoint.LongFormAuthoring.Plan, "il_long_form_plan"); err != nil {
			return err
		}
		if err := validateCheckpointArtifacts(checkpoint.LongFormAuthoring.SectionArtifacts, "il_long_form_section", sections); err != nil {
			return err
		}
		if err := validateCheckpointArtifacts(checkpoint.LongFormAuthoring.PartArtifacts, "il_long_form_part", parts); err != nil {
			return err
		}
	} else {
		if !strings.HasPrefix(workspace.ArtifactID, "art_") || len(workspace.SHA256) != 64 || workspace.ByteSize < 1 || workspace.Revision < 1 || workspace.Stage != checkpoint.Stage {
			return fmt.Errorf("report IL checkpoint author workspace is invalid: stage=%s workspace_stage=%s", checkpoint.Stage, workspace.Stage)
		}
		finalizationBound := false
		for _, finalization := range checkpoint.LongFormAuthoring.Finalizations {
			if finalization.Stage == checkpoint.Stage && finalization.Artifact.ArtifactID == workspace.ArtifactID && finalization.Artifact.SHA256 == workspace.SHA256 && finalization.Artifact.ByteSize == workspace.ByteSize {
				finalizationBound = true
				break
			}
		}
		if checkpoint.LongFormAuthoring.Final.ArtifactID != workspace.ArtifactID || checkpoint.LongFormAuthoring.Final.SHA256 != workspace.SHA256 || checkpoint.LongFormAuthoring.Final.ByteSize != workspace.ByteSize || !finalizationBound {
			return fmt.Errorf("report IL checkpoint long-form lineage is invalid: stage=%s final_stage=%s finalization_bound=%t", checkpoint.Stage, checkpoint.LongFormAuthoring.Final.Stage, finalizationBound)
		}
	}
	reader := checkpoint.ReaderFinalization
	if reader.ContinuityPatches < 0 || reader.PublicationPatches < 0 || reader.ProposedPatches < 0 ||
		reader.AcceptedPatches < 0 || reader.RejectedPatches < 0 ||
		reader.AcceptedPatches+reader.RejectedPatches != reader.ProposedPatches ||
		reader.AcceptedPatches != reader.ContinuityPatches+reader.PublicationPatches ||
		reader.Applied != (reader.AcceptedPatches > 0) {
		return fmt.Errorf("report IL checkpoint reader finalization is invalid")
	}
	return nil
}

func validateCheckpointArtifacts(artifacts []CheckpointArtifact, stage string, expected int) error {
	if len(artifacts) != expected {
		return fmt.Errorf("report IL checkpoint %s artifact count differs", stage)
	}
	seen := map[string]bool{}
	for _, artifact := range artifacts {
		if err := validateCheckpointArtifact(artifact, stage); err != nil {
			return err
		}
		if seen[artifact.ArtifactID] {
			return fmt.Errorf("report IL checkpoint %s artifact is duplicated", stage)
		}
		seen[artifact.ArtifactID] = true
	}
	return nil
}

func validateCheckpointArtifact(artifact CheckpointArtifact, stage string) error {
	if !strings.HasPrefix(artifact.ArtifactID, "art_") || len(artifact.SHA256) != 64 || artifact.ByteSize < 1 || artifact.Stage != stage {
		return fmt.Errorf("report IL checkpoint %s artifact is invalid", stage)
	}
	return nil
}

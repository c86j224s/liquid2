package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

const reportILCheckpointEventType = "report.il.checkpoint.created"

func (s *Service) AppendReportILCheckpoint(
	ctx context.Context,
	missionID string,
	checkpoint reportilcontract.ProductCheckpoint,
) error {
	if err := reportilcontract.ValidateProductCheckpoint(checkpoint, missionID); err != nil {
		return fmt.Errorf("validate report IL checkpoint: %w", err)
	}
	_, err := s.AppendEvent(ctx, AppendEventRequest{
		EventID: newAppID("evt"), MissionID: missionID,
		EventType:        reportILCheckpointEventType,
		Producer:         Producer{Type: "system", ID: "report-il"},
		CausationEventID: checkpoint.PendingEventID,
		CorrelationID:    checkpoint.PendingEventID,
		Payload: mustMarshalJSON(map[string]any{
			"kind":             "report_il_checkpoint",
			"pending_event_id": checkpoint.PendingEventID,
			"pipeline_family":  reportilcontract.PipelineFamily,
			"stage":            checkpoint.Stage,
			"checkpoint":       checkpoint,
		}),
	})
	return err
}

func (s *Service) LoadReportILResumeCheckpoint(
	ctx context.Context,
	missionID,
	failedPendingID string,
) (*reportilcontract.ResumeCheckpoint, error) {
	events, err := s.ListEvents(ctx, missionID)
	if err != nil {
		return nil, err
	}
	var selected *reportilcontract.ProductCheckpoint
	for _, event := range events {
		if event.EventType != reportILCheckpointEventType {
			continue
		}
		var header struct {
			PendingID string `json:"pending_event_id"`
		}
		if json.Unmarshal(event.Payload, &header) != nil {
			return nil, fmt.Errorf("report IL checkpoint event is malformed")
		}
		if header.PendingID != failedPendingID {
			continue
		}
		var payload struct {
			PendingID  string                             `json:"pending_event_id"`
			Checkpoint reportilcontract.ProductCheckpoint `json:"checkpoint"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil || payload.PendingID != failedPendingID {
			return nil, fmt.Errorf("report IL checkpoint event is malformed")
		}
		checkpoint := payload.Checkpoint
		if err := reportilcontract.ValidateProductCheckpoint(checkpoint, missionID); err != nil {
			continue
		}
		if selected == nil || reportILResumeStageOrder(checkpoint.Stage) >= reportILResumeStageOrder(selected.Stage) {
			selected = &checkpoint
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("report IL retry has no durable checkpoint")
	}
	memory, err := s.readCheckpointEditorialMemory(ctx, missionID, selected.EditorialMemory, selected.AuthorCatalog)
	if err != nil {
		return nil, err
	}
	resume := &reportilcontract.ResumeCheckpoint{ProductCheckpoint: *selected, EditorialMemory: memory}
	if selected.Stage == "il_long_form_parts" {
		plan, err := s.readCheckpointLongFormPlan(ctx, missionID, selected.LongFormAuthoring.Plan, selected.AuthorCatalog, memory, selected.LongFormAuthoring.Parts, selected.LongFormAuthoring.Sections)
		if err != nil {
			return nil, err
		}
		for _, artifact := range selected.LongFormAuthoring.SectionArtifacts {
			if _, err := s.readCheckpointAuthorArtifact(ctx, missionID, artifact, selected.AuthorCatalog, false); err != nil {
				return nil, err
			}
		}
		for index, artifact := range selected.LongFormAuthoring.PartArtifacts {
			document, err := s.readCheckpointAuthorArtifact(ctx, missionID, artifact, selected.AuthorCatalog, false)
			if err != nil {
				return nil, err
			}
			if len(document.Parts) != 1 || document.Parts[0].PartKey != plan.Parts[index].PartKey {
				return nil, fmt.Errorf("report IL Part checkpoint artifact inventory differs")
			}
		}
		resume.LongFormPlan = plan
		return resume, nil
	}
	document, err := s.readCheckpointAuthorDocument(ctx, missionID, selected.AuthorWorkspace, selected.AuthorCatalog)
	if err != nil {
		return nil, err
	}
	resume.AuthorDocument = document
	return resume, nil
}

func reportILResumeStageOrder(stage string) int {
	switch stage {
	case "il_long_form_parts":
		return 1
	case "il_long_form_final":
		return 2
	case "il_reader":
		return 3
	case "il_continuity":
		return 4
	default:
		return 0
	}
}

func (s *Service) readCheckpointEditorialMemory(
	ctx context.Context,
	missionID string,
	checkpoint reportilcontract.CheckpointEditorialMemory,
	catalog reportilcontract.SourceCatalog,
) (reportilcontract.EditorialMemory, error) {
	artifact, err := s.GetRawArtifact(ctx, checkpoint.ArtifactID)
	if err != nil {
		return reportilcontract.EditorialMemory{}, err
	}
	if err := validateCheckpointArtifact(artifact, missionID, reportilcontract.EditorialMemoryMediaType, checkpoint.SHA256, checkpoint.ByteSize); err != nil {
		return reportilcontract.EditorialMemory{}, err
	}
	var stored reportilcontract.EditorialMemoryArtifact
	if err := decodeCheckpointJSON(artifact.Content, &stored); err != nil {
		return reportilcontract.EditorialMemory{}, err
	}
	if len(stored.Accounts) != checkpoint.Accounts {
		return reportilcontract.EditorialMemory{}, fmt.Errorf("report IL checkpoint editorial memory count differs")
	}
	if err := reportilcontract.ValidateEditorialMemoryArtifact(stored, catalog); err != nil {
		return reportilcontract.EditorialMemory{}, err
	}
	return reportilcontract.EditorialMemoryFromArtifact(stored), nil
}

func (s *Service) readCheckpointLongFormPlan(
	ctx context.Context,
	missionID string,
	checkpoint reportilcontract.CheckpointArtifact,
	catalog reportilcontract.SourceCatalog,
	memory reportilcontract.EditorialMemory,
	parts,
	sections int,
) (reportilcontract.LongFormPlan, error) {
	artifact, err := s.GetRawArtifact(ctx, checkpoint.ArtifactID)
	if err != nil {
		return reportilcontract.LongFormPlan{}, err
	}
	if err := validateCheckpointArtifact(artifact, missionID, reportilcontract.LongFormPlanMediaType, checkpoint.SHA256, checkpoint.ByteSize); err != nil {
		return reportilcontract.LongFormPlan{}, err
	}
	var plan reportilcontract.LongFormPlan
	if err := decodeCheckpointJSON(artifact.Content, &plan); err != nil {
		return reportilcontract.LongFormPlan{}, err
	}
	if err := reportilcontract.ValidateLongFormPlan(plan, &memory, catalog); err != nil {
		return reportilcontract.LongFormPlan{}, err
	}
	if len(plan.Parts) != parts || reportilcontract.LongFormPlanSectionCount(plan) != sections {
		return reportilcontract.LongFormPlan{}, fmt.Errorf("report IL checkpoint plan inventory differs")
	}
	return plan, nil
}

func (s *Service) readCheckpointAuthorArtifact(
	ctx context.Context,
	missionID string,
	checkpoint reportilcontract.CheckpointArtifact,
	catalog reportilcontract.SourceCatalog,
	complete bool,
) (reportilcontract.AuthorDocument, error) {
	artifact, err := s.GetRawArtifact(ctx, checkpoint.ArtifactID)
	if err != nil {
		return reportilcontract.AuthorDocument{}, err
	}
	if err := validateCheckpointArtifact(artifact, missionID, reportilcontract.AuthorDocumentMediaType, checkpoint.SHA256, checkpoint.ByteSize); err != nil {
		return reportilcontract.AuthorDocument{}, err
	}
	var document reportilcontract.AuthorDocument
	if err := decodeCheckpointJSON(artifact.Content, &document); err != nil {
		return reportilcontract.AuthorDocument{}, err
	}
	if complete {
		err = reportilcontract.ValidateLongFormAuthorDocument(document, catalog)
	} else {
		err = reportilcontract.ValidateLongFormAuthorFragment(document, catalog)
	}
	if err != nil {
		return reportilcontract.AuthorDocument{}, err
	}
	return document, nil
}

func (s *Service) readCheckpointAuthorDocument(
	ctx context.Context,
	missionID string,
	checkpoint reportilcontract.AuthorWorkspaceReceipt,
	catalog reportilcontract.SourceCatalog,
) (reportilcontract.AuthorDocument, error) {
	return s.readCheckpointAuthorArtifact(ctx, missionID, reportilcontract.CheckpointArtifact{
		ArtifactID: checkpoint.ArtifactID, SHA256: checkpoint.SHA256, ByteSize: checkpoint.ByteSize, Stage: checkpoint.Stage,
	}, catalog, true)
}

func validateCheckpointArtifact(
	artifact RawArtifact,
	missionID,
	mediaType,
	expectedSHA string,
	expectedBytes int,
) error {
	sum := sha256.Sum256(artifact.Content)
	if artifact.MissionID != missionID || artifact.MediaType != mediaType ||
		artifact.SHA256 != expectedSHA || hex.EncodeToString(sum[:]) != expectedSHA ||
		artifact.ByteSize != int64(expectedBytes) || len(artifact.Content) != expectedBytes ||
		artifact.Producer.Type != "mcp_tool" {
		return fmt.Errorf("report IL checkpoint artifact binding is invalid")
	}
	return nil
}

func decodeCheckpointJSON(content []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("report IL checkpoint artifact decode failed: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("report IL checkpoint artifact contains multiple JSON values")
	}
	return nil
}

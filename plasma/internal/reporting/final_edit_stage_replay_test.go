package reporting

import (
	"context"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestFinalEditStageReplayRejectsArtifactFilenameMismatch(t *testing.T) {
	ctx := context.Background()
	binding := finalEditStageStoreStageBinding(finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled), FinalEditStageReader, "art_source", "art_reader")
	source := finalEditStageReplayArtifact(binding.SourceArtifactID, binding.MissionID, binding.Filename, app.Producer{Type: "system", ID: "source"}, "source")
	artifact := finalEditStageReplayArtifact(binding.EditedArtifactID, binding.MissionID, binding.Filename, binding.Producer, "edited")

	for name, mutate := range map[string]func(*app.RawArtifact){
		"source": func(value *app.RawArtifact) { value.Filename = "wrong-source.md" },
		"result": func(value *app.RawArtifact) { value.Filename = "wrong-result.md" },
	} {
		t.Run(name, func(t *testing.T) {
			source, artifact := source, artifact
			if name == "source" {
				mutate(&source)
			} else {
				mutate(&artifact)
			}
			event := finalEditStageReplaySubmittedEvent(binding, source, artifact, true, nil)
			_, err := finalEditStageResultFromEvent(ctx, finalEditStageReplayBaseStore{artifacts: map[string]app.RawArtifact{
				source.ArtifactID:   source,
				artifact.ArtifactID: artifact,
			}}, binding, event, true)
			if !errors.Is(err, app.ErrConflict) {
				t.Fatalf("err=%v, want conflict", err)
			}
		})
	}
}

func TestFinalEditStageReplayRejectsChangedArtifactWithSameContentAndSHA(t *testing.T) {
	ctx := context.Background()
	binding := finalEditStageStoreStageBinding(finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled), FinalEditStageReader, "art_source", "art_reader")
	source := finalEditStageReplayArtifact(binding.SourceArtifactID, binding.MissionID, binding.Filename, app.Producer{Type: "system", ID: "source"}, "same")
	artifact := finalEditStageReplayArtifact(binding.EditedArtifactID, binding.MissionID, binding.Filename, binding.Producer, "same")
	event := finalEditStageReplaySubmittedEvent(binding, source, artifact, true, nil)

	_, err := finalEditStageResultFromEvent(ctx, finalEditStageReplayBaseStore{artifacts: map[string]app.RawArtifact{
		source.ArtifactID:   source,
		artifact.ArtifactID: artifact,
	}}, binding, event, true)
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("err=%v, want conflict", err)
	}
}

func TestFinalEditStageReplayRequiresStoredApprovedEvidence(t *testing.T) {
	ctx := context.Background()
	binding := finalEditStageStoreStageBinding(finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled), FinalEditStageGate, "art_source", "art_final")
	source := finalEditStageReplayArtifact(binding.SourceArtifactID, binding.MissionID, binding.Filename, app.Producer{Type: "system", ID: "source"}, "source")
	finding := StoredFinalEditGateFinding{
		StatementSHA256: contentSHA256([]byte("Stored unsupported claim.")),
		Classification:  FinalEditGateClassUnverifiedExternalFact,
		RepairAction:    FinalEditRepairAttachApprovedEvidence,
		EvidenceIDs:     []string{"evd_gate"},
	}
	event := finalEditStageReplaySubmittedEvent(binding, source, source, false, []StoredFinalEditGateFinding{finding})

	for name, store := range map[string]LongFormFinalizationStore{
		"missing_validator": finalEditStageReplayBaseStore{artifacts: map[string]app.RawArtifact{source.ArtifactID: source}},
		"lookup_error": finalEditStageReplayEvidenceStore{
			finalEditStageReplayBaseStore: finalEditStageReplayBaseStore{artifacts: map[string]app.RawArtifact{source.ArtifactID: source}},
			err:                           errors.New("lookup failed"),
		},
		"foreign_mission": finalEditStageReplayEvidenceStore{
			finalEditStageReplayBaseStore: finalEditStageReplayBaseStore{artifacts: map[string]app.RawArtifact{source.ArtifactID: source}},
			evidence: map[string]app.EvidenceRecord{
				"evd_gate": {EvidenceID: "evd_gate", MissionID: "mis_other", State: "approved"},
			},
		},
		"proposed": finalEditStageReplayEvidenceStore{
			finalEditStageReplayBaseStore: finalEditStageReplayBaseStore{artifacts: map[string]app.RawArtifact{source.ArtifactID: source}},
			evidence: map[string]app.EvidenceRecord{
				"evd_gate": {EvidenceID: "evd_gate", MissionID: binding.MissionID, State: "proposed"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := finalEditStageResultFromEvent(ctx, store, binding, event, true)
			if !errors.Is(err, app.ErrConflict) {
				t.Fatalf("err=%v, want conflict", err)
			}
		})
	}

	okStore := finalEditStageReplayEvidenceStore{
		finalEditStageReplayBaseStore: finalEditStageReplayBaseStore{artifacts: map[string]app.RawArtifact{source.ArtifactID: source}},
		evidence: map[string]app.EvidenceRecord{
			"evd_gate": {EvidenceID: "evd_gate", MissionID: binding.MissionID, State: "approved"},
		},
	}
	if _, err := finalEditStageResultFromEvent(ctx, okStore, binding, event, true); err != nil {
		t.Fatalf("approved evidence rejected: %v", err)
	}
}

func TestFinalEditStageEventDecodePreservesStoredPipelineOwnership(t *testing.T) {
	legacy := finalEditStageStoreStageBinding(finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled), FinalEditStageReader, "art_source", "art_reader")
	legacySource := finalEditStageReplayArtifact(legacy.SourceArtifactID, legacy.MissionID, legacy.Filename, app.Producer{Type: "system", ID: "source"}, "source")
	legacyEvent := finalEditStageReplaySubmittedEvent(legacy, legacySource, legacySource, false, nil)
	decodedLegacy, ok := finalEditStageBindingFromSubmittedEventForPipeline(legacyEvent, FinalEditPipelineReaderStyleGateV1)
	if !ok || decodedLegacy.FinalEditPipeline != "" || decodedLegacy.Stage != FinalEditStageReader {
		t.Fatalf("legacy v1 decode changed: decoded=%#v ok=%t", decodedLegacy, ok)
	}

	writer := finalEditStageStoreStageBinding(finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled), FinalEditStageWriter, "art_assembly", "art_writer")
	writer.FinalEditPipeline = FinalEditPipelineAssemblyWriterReaderStyleGateV2
	writer.PreviousProviderSessionID = writer.ReportPlanSessionID
	writer.ForkSourceAgentSessionID = writer.ReportPlanSessionID
	writerSource := finalEditStageReplayArtifact(writer.SourceArtifactID, writer.MissionID, writer.Filename, app.Producer{Type: "system", ID: FinalEditAssemblyProducerID}, "assembled")
	writerArtifact := finalEditStageReplayArtifact(writer.EditedArtifactID, writer.MissionID, writer.Filename, writer.Producer, "written")
	writerEvent := finalEditStageReplaySubmittedEvent(writer, writerSource, writerArtifact, true, nil)
	decodedWriter, ok := finalEditStageBindingFromSubmittedEventForPipeline(writerEvent, FinalEditPipelineAssemblyWriterReaderStyleGateV2)
	if !ok || decodedWriter.FinalEditPipeline != FinalEditPipelineAssemblyWriterReaderStyleGateV2 || decodedWriter.Stage != FinalEditStageWriter {
		t.Fatalf("v2 writer decode changed: decoded=%#v ok=%t", decodedWriter, ok)
	}
	if _, ok := finalEditStageBindingFromSubmittedEventForPipeline(writerEvent, FinalEditPipelineReaderStyleGateV1); ok {
		t.Fatal("v2 writer event decoded through v1 replay path")
	}
}

type finalEditStageReplayBaseStore struct {
	artifacts map[string]app.RawArtifact
}

func (s finalEditStageReplayBaseStore) ListEvents(context.Context, string) ([]app.LedgerEvent, error) {
	return nil, nil
}

func (s finalEditStageReplayBaseStore) GetRawArtifact(_ context.Context, artifactID string) (app.RawArtifact, error) {
	artifact, ok := s.artifacts[artifactID]
	if !ok {
		return app.RawArtifact{}, errors.New("missing artifact")
	}
	return artifact, nil
}

func (s finalEditStageReplayBaseStore) AppendEventConditionally(context.Context, string, func([]app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.LedgerEvent, bool, error) {
	return app.LedgerEvent{}, false, errors.New("unused")
}

func (s finalEditStageReplayBaseStore) CreateRawArtifactWithEventConditionally(context.Context, app.CreateRawArtifactRequest, func([]app.LedgerEvent, app.RawArtifact) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.RawArtifact, app.LedgerEvent, bool, error) {
	return app.RawArtifact{}, app.LedgerEvent{}, false, errors.New("unused")
}

type finalEditStageReplayEvidenceStore struct {
	finalEditStageReplayBaseStore
	evidence map[string]app.EvidenceRecord
	err      error
}

func (s finalEditStageReplayEvidenceStore) GetEvidenceRecord(_ context.Context, evidenceID string) (app.EvidenceRecord, error) {
	if s.err != nil {
		return app.EvidenceRecord{}, s.err
	}
	record, ok := s.evidence[evidenceID]
	if !ok {
		return app.EvidenceRecord{}, errors.New("missing evidence")
	}
	return record, nil
}

func finalEditStageReplayArtifact(artifactID, missionID, filename string, producer app.Producer, content string) app.RawArtifact {
	return app.RawArtifact{
		ArtifactID: artifactID,
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   filename,
		Producer:   producer,
		SHA256:     contentSHA256([]byte(content)),
		Content:    []byte(content),
	}
}

func finalEditStageReplaySubmittedEvent(binding FinalEditStageBinding, source, artifact app.RawArtifact, changed bool, findings []StoredFinalEditGateFinding) app.LedgerEvent {
	request := buildFinalEditSubmittedAppendRequest("evt_stage_replay_submit", binding, source, artifact, 1, changed, findings)
	return app.LedgerEvent{
		EventID:          request.EventID,
		MissionID:        request.MissionID,
		EventType:        request.EventType,
		Producer:         request.Producer,
		CausationEventID: request.CausationEventID,
		CorrelationID:    request.CorrelationID,
		Payload:          request.Payload,
	}
}

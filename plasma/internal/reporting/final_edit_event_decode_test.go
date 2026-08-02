package reporting

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestFinalEditSubmittedEventDecodeFailsClosedOnEnvelopeTamper(t *testing.T) {
	binding := finalEditStageStoreStageBinding(finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled), FinalEditStageReader, "art_source", "art_reader")
	source := app.RawArtifact{ArtifactID: binding.SourceArtifactID, MissionID: binding.MissionID, MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename, Producer: app.Producer{Type: "system", ID: "source"}, SHA256: contentSHA256([]byte("source")), Content: []byte("source")}
	artifact := app.RawArtifact{ArtifactID: binding.EditedArtifactID, MissionID: binding.MissionID, MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename, Producer: binding.Producer, SHA256: contentSHA256([]byte("edited")), Content: []byte("edited")}
	request := buildFinalEditSubmittedAppendRequest("evt_reader_submitted", binding, source, artifact, 1, true, nil, FinalEditSemanticAttestation{})
	base := app.LedgerEvent{EventID: request.EventID, MissionID: request.MissionID, EventType: request.EventType, Producer: request.Producer, CausationEventID: request.CausationEventID, CorrelationID: request.CorrelationID, Payload: request.Payload}

	for name, mutate := range map[string]func(*app.LedgerEvent){
		"correlation": func(event *app.LedgerEvent) { event.CorrelationID = "wrong-key" },
		"causation":   func(event *app.LedgerEvent) { event.CausationEventID = "evt_wrong_plan" },
		"type":        func(event *app.LedgerEvent) { event.EventType = FinalEditStyleSubmittedEventType },
		"producer": func(event *app.LedgerEvent) {
			event.Producer = app.Producer{Type: "agent_session", ID: "provider-other"}
		},
		"stage": func(event *app.LedgerEvent) {
			finalEditDecodeMutatePayload(t, event, func(payload map[string]any) { payload["stage"] = FinalEditStageStyle })
		},
		"stage_id": func(event *app.LedgerEvent) {
			finalEditDecodeMutatePayload(t, event, func(payload map[string]any) { payload["stage_id"] = "style-edit" })
		},
		"negative_operation": func(event *app.LedgerEvent) {
			finalEditDecodeMutatePayload(t, event, func(payload map[string]any) { payload["operation_count"] = -1 })
		},
	} {
		t.Run(name, func(t *testing.T) {
			event := base
			mutate(&event)
			if _, ok := finalEditStageBindingFromSubmittedEvent(event); ok {
				t.Fatalf("tampered event decoded: %#v", event)
			}
		})
	}
}

func TestStoredFinalEditGateFindingsDecodeFailsClosedOnMalformedPayload(t *testing.T) {
	validHash := contentSHA256([]byte("Unsupported claim."))
	for name, payload := range map[string]any{
		"raw_statement": []any{map[string]any{"statement": "leak", "statement_sha256": validHash, "classification": FinalEditGateClassMissionSourceGrounded}},
		"bad_class":     []any{map[string]any{"statement_sha256": validHash, "classification": "external"}},
		"bad_action":    []any{map[string]any{"statement_sha256": validHash, "classification": FinalEditGateClassMissionSourceGrounded, "repair_action": FinalEditRepairRemove}},
		"bad_evidence":  []any{map[string]any{"statement_sha256": validHash, "classification": FinalEditGateClassUnverifiedExternalFact, "repair_action": FinalEditRepairAttachApprovedEvidence, "evidence_ids": []any{1}}},
		"duplicate": []any{
			map[string]any{"statement_sha256": validHash, "classification": FinalEditGateClassMissionSourceGrounded},
			map[string]any{"statement_sha256": validHash, "classification": FinalEditGateClassMissionSourceGrounded},
		},
		"short_hash": []any{
			map[string]any{"statement_sha256": "abc123", "classification": FinalEditGateClassMissionSourceGrounded},
		},
		"uppercase_hash": []any{
			map[string]any{"statement_sha256": "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789", "classification": FinalEditGateClassMissionSourceGrounded},
		},
		"nonhex_hash": []any{
			map[string]any{"statement_sha256": "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", "classification": FinalEditGateClassMissionSourceGrounded},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeStoredFinalEditGateFindingsPayload(payload); !errors.Is(err, app.ErrConflict) {
				t.Fatalf("err=%v, want conflict", err)
			}
		})
	}
}

func TestStoredFinalEditGateFindingsDecodeRejectsMalformedStatementSHA256(t *testing.T) {
	for _, hash := range []string{
		"abc123",
		"ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789",
		"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
	} {
		payload := []any{map[string]any{
			"statement_sha256": hash,
			"classification":   FinalEditGateClassUnverifiedExternalFact,
			"repair_action":    FinalEditRepairRemove,
		}}
		if _, err := decodeStoredFinalEditGateFindingsPayload(payload); !errors.Is(err, app.ErrConflict) {
			t.Fatalf("hash=%q err=%v, want conflict", hash, err)
		}
	}
}

func TestStoredFinalEditGateFindingsDecodeSeparatesLegacyAndEvidenceGateRules(t *testing.T) {
	validHash := contentSHA256([]byte("Unsupported claim."))
	readOnly := []any{map[string]any{
		"statement_sha256": validHash,
		"classification":   FinalEditGateClassUnverifiedExternalFact,
		"evidence_ids":     []any{"evd_ok"},
	}}
	if _, err := decodeStoredFinalEditGateFindingsPayloadForStage(readOnly, FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3, FinalEditStageEvidenceGate); err != nil {
		t.Fatalf("v3 evidence gate read-only finding rejected: %v", err)
	}
	if _, err := decodeStoredFinalEditGateFindingsPayloadForStage(readOnly, FinalEditPipelineReaderStyleGateV1, FinalEditStageGate); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("legacy unverified finding without repair_action err=%v, want conflict", err)
	}

	legacyRepaired := []any{map[string]any{
		"statement_sha256": validHash,
		"classification":   FinalEditGateClassUnverifiedExternalFact,
		"repair_action":    FinalEditRepairAttachApprovedEvidence,
		"evidence_ids":     []any{"evd_ok"},
	}}
	if _, err := decodeStoredFinalEditGateFindingsPayloadForStage(legacyRepaired, FinalEditPipelineReaderStyleGateV1, FinalEditStageGate); err != nil {
		t.Fatalf("legacy repaired finding rejected: %v", err)
	}
	if _, err := decodeStoredFinalEditGateFindingsPayloadForStage(legacyRepaired, FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3, FinalEditStageEvidenceGate); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("v3 repair_action finding err=%v, want conflict", err)
	}
}

func finalEditDecodeMutatePayload(t *testing.T, event *app.LedgerEvent, mutate func(map[string]any)) {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	mutate(payload)
	event.Payload = mustJSON(payload)
}

package reporting

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestSubmitFinalEditStyleStageStoresOperationDiagnoses(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeEnabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeEnabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "style_diagnosis")
	styleBinding := finalEditStageStoreStageBinding(binding, FinalEditStageStyle, reader.Artifact.ArtifactID, "art_style_diagnosis")
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_style_diagnosis_start", styleBinding); err != nil || !created {
		t.Fatalf("style start created=%t err=%v", created, err)
	}
	markdown := strings.Replace(string(reader.Artifact.Content), "Preserved body.", "Preserved body!", 1)
	diagnoses := finalEditStyleDiagnosesForTest(1)
	result, err := SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_style_diagnosis_submit", markdown, 1, diagnoses)
	if err != nil {
		t.Fatal(err)
	}
	if !result.StyleOperationDiagnosesPresent || !equalFinalEditStyleOperationDiagnoses(result.StyleOperationDiagnoses, diagnoses) {
		t.Fatalf("style diagnoses were not returned: %#v", result)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(result.Event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	records, ok := payload[FinalEditStyleOperationDiagnosesField].([]any)
	if !ok || len(records) != 1 {
		t.Fatalf("style diagnoses payload missing: %#v", payload)
	}
	record, ok := records[0].(map[string]any)
	if !ok || len(record) != 6 || record["operation_ordinal"] != float64(1) || record["category"] != "unnatural_collocation" ||
		record["reason"] != "awkward local phrasing" || record["match_text"] != "Preserved body." ||
		record["replacement"] != "Preserved body!" || record["occurrence"] != float64(1) {
		t.Fatalf("style diagnosis payload record differs: %#v", records[0])
	}
	if payload[FinalEditStyleOperationDiagnosesVersionField] != float64(FinalEditStyleOperationDiagnosesVersion) {
		t.Fatalf("style diagnosis version missing: %#v", payload)
	}
}

func TestSubmitFinalEditStyleStageRejectsChangedZeroOperations(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeEnabled)
	defer closeStore()
	_, reader, styleBinding := startStyleDiagnosisTestStage(t, ctx, svc, "changed_zero")
	markdown := strings.Replace(string(reader.Artifact.Content), "Preserved body.", "Preserved body!", 1)
	_, err := SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_style_changed_zero_submit", markdown, 0, nil)
	if !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("changed zero-operation style submit error=%v, want invalid input", err)
	}
}

func TestSubmitFinalEditStyleStageNoOpStoresExplicitEmptyDiagnoses(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeEnabled)
	defer closeStore()
	_, reader, styleBinding := startStyleDiagnosisTestStage(t, ctx, svc, "noop")
	result, err := SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_style_noop_diagnosis_submit", string(reader.Artifact.Content), 1, finalEditStyleDiagnosesForTest(1))
	if err != nil {
		t.Fatal(err)
	}
	if result.OperationCount != 0 || !result.StyleOperationDiagnosesPresent || len(result.StyleOperationDiagnoses) != 0 {
		t.Fatalf("no-op style diagnosis result mismatch: %#v", result)
	}
	payload := eventPayload(result.Event)
	records, ok := payload[FinalEditStyleOperationDiagnosesField].([]any)
	if !ok || len(records) != 0 || jsonInt(payload["operation_count"]) != 0 || payload[FinalEditStyleOperationDiagnosesVersionField] != float64(FinalEditStyleOperationDiagnosesVersion) {
		t.Fatalf("no-op style payload did not store explicit empty diagnoses: %#v", payload)
	}
}

func TestFinalEditStyleDiagnosisReplayAllowsHistoricalMissingField(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeEnabled)
	defer closeStore()
	_, reader, styleBinding := startStyleDiagnosisTestStage(t, ctx, svc, "legacy_missing")
	source, artifact := styleDiagnosisChangedArtifacts(t, reader.Artifact, styleBinding)
	request := buildFinalEditSubmittedAppendRequestWithStyleDiagnoses("evt_style_legacy_missing_submit", styleBinding, source, artifact, 1, true, finalEditStyleDiagnosesForTest(1), nil, FinalEditSemanticAttestation{})
	payload := eventPayload(app.LedgerEvent{Payload: request.Payload})
	delete(payload, FinalEditStyleOperationDiagnosesField)
	delete(payload, FinalEditStyleOperationDiagnosesVersionField)
	request.Payload = finalEditStageStoreJSON(payload)
	appendStyleDiagnosisArtifactAndEvent(t, ctx, svc, artifact, request)

	result, ok, err := LoadFinalEditStageSubmission(ctx, svc, styleBinding)
	if err != nil || !ok || !result.Replay || result.StyleOperationDiagnosesPresent || len(result.StyleOperationDiagnoses) != 0 {
		t.Fatalf("legacy missing-field style replay mismatch: result=%#v ok=%t err=%v", result, ok, err)
	}
}

func TestFinalEditStyleDiagnosisReplayAllowsLegacyTwoFieldRecords(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeEnabled)
	defer closeStore()
	_, reader, styleBinding := startStyleDiagnosisTestStage(t, ctx, svc, "legacy_two_field")
	source, artifact := styleDiagnosisChangedArtifacts(t, reader.Artifact, styleBinding)
	request := buildFinalEditSubmittedAppendRequestWithStyleDiagnoses("evt_style_legacy_two_field_submit", styleBinding, source, artifact, 1, true, finalEditStyleDiagnosesForTest(1), nil, FinalEditSemanticAttestation{})
	payload := eventPayload(app.LedgerEvent{Payload: request.Payload})
	delete(payload, FinalEditStyleOperationDiagnosesVersionField)
	payload[FinalEditStyleOperationDiagnosesField] = []any{map[string]any{
		"operation_ordinal": float64(1),
		"category":          "unnatural_collocation",
	}}
	request.Payload = finalEditStageStoreJSON(payload)
	appendStyleDiagnosisArtifactAndEvent(t, ctx, svc, artifact, request)

	result, ok, err := LoadFinalEditStageSubmission(ctx, svc, styleBinding)
	if err != nil || !ok || !result.Replay || !result.StyleOperationDiagnosesPresent || len(result.StyleOperationDiagnoses) != 1 {
		t.Fatalf("legacy two-field style replay mismatch: result=%#v ok=%t err=%v", result, ok, err)
	}
	if result.StyleOperationDiagnoses[0].OperationOrdinal != 1 || result.StyleOperationDiagnoses[0].Category != "unnatural_collocation" ||
		result.StyleOperationDiagnoses[0].Reason != "" || result.StyleOperationDiagnoses[0].MatchText != "" ||
		result.StyleOperationDiagnoses[0].Replacement != "" || result.StyleOperationDiagnoses[0].Occurrence != 0 {
		t.Fatalf("legacy two-field diagnosis mismatch: %#v", result.StyleOperationDiagnoses[0])
	}
}

func TestFinalEditStyleDiagnosisReplayComparesLegacyRetryByOrdinalAndCategory(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeEnabled)
	defer closeStore()
	_, reader, styleBinding := startStyleDiagnosisTestStage(t, ctx, svc, "legacy_retry")
	source, artifact := styleDiagnosisChangedArtifacts(t, reader.Artifact, styleBinding)
	markdown := string(artifact.Content)
	request := buildFinalEditSubmittedAppendRequestWithStyleDiagnoses("evt_style_legacy_retry_submit", styleBinding, source, artifact, 1, true, finalEditStyleDiagnosesForTest(1), nil, FinalEditSemanticAttestation{})
	payload := eventPayload(app.LedgerEvent{Payload: request.Payload})
	delete(payload, FinalEditStyleOperationDiagnosesVersionField)
	payload[FinalEditStyleOperationDiagnosesField] = []any{map[string]any{
		"operation_ordinal": float64(1),
		"category":          "unnatural_collocation",
	}}
	request.Payload = finalEditStageStoreJSON(payload)
	appendStyleDiagnosisArtifactAndEvent(t, ctx, svc, artifact, request)

	detailed := finalEditStyleDiagnosesForTest(1)
	result, err := SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_style_legacy_retry_again", markdown, 1, detailed)
	if err != nil {
		t.Fatalf("legacy two-field retry with detailed records conflicted: %v", err)
	}
	if !result.Replay || !result.StyleOperationDiagnosesPresent || len(result.StyleOperationDiagnoses) != 1 {
		t.Fatalf("legacy retry did not replay stored event: %#v", result)
	}

	mismatched := finalEditStyleDiagnosesForTest(1)
	mismatched[0].Category = "vague_reference"
	_, err = SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_style_legacy_retry_mismatch", markdown, 1, mismatched)
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("legacy two-field retry mismatch error=%v, want conflict", err)
	}
}

func TestFinalEditStyleDiagnosisReplayCategoryMismatchConflicts(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeEnabled)
	defer closeStore()
	_, reader, styleBinding := startStyleDiagnosisTestStage(t, ctx, svc, "replay_mismatch")
	markdown := strings.Replace(string(reader.Artifact.Content), "Preserved body.", "Preserved body!", 1)
	if _, err := SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_style_replay_mismatch_submit", markdown, 1, finalEditStyleDiagnosesForTest(1)); err != nil {
		t.Fatal(err)
	}
	mismatched := finalEditStyleDiagnosesForTest(1)
	mismatched[0].Category = "vague_reference"
	_, err := SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_style_replay_mismatch_again", markdown, 1, mismatched)
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("style replay category mismatch error=%v, want conflict", err)
	}
}

func TestFinalEditNonStyleDiagnosisFieldConflicts(t *testing.T) {
	ctx := context.Background()
	for name, mutate := range map[string]func(map[string]any){
		"field": func(payload map[string]any) { payload[FinalEditStyleOperationDiagnosesField] = []any{} },
		"version": func(payload map[string]any) {
			payload[FinalEditStyleOperationDiagnosesVersionField] = FinalEditStyleOperationDiagnosesVersion
		},
	} {
		t.Run(name, func(t *testing.T) {
			svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
			defer closeStore()
			binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
			sourceID := FinalEditReaderSourceArtifactID(binding.PlanEventID, []string{"art_part"})
			readerBinding := finalEditStageStoreStageBinding(binding, FinalEditStageReader, sourceID, "art_reader_diagnosis_"+name)
			if _, created, err := StartFinalEditStage(ctx, svc, "evt_reader_diagnosis_"+name+"_start", readerBinding); err != nil || !created {
				t.Fatalf("reader start created=%t err=%v", created, err)
			}
			source, err := svc.GetRawArtifact(ctx, sourceID)
			if err != nil {
				t.Fatal(err)
			}
			request := buildFinalEditSubmittedAppendRequest("evt_reader_diagnosis_"+name+"_submit", readerBinding, source, source, 0, false, nil, FinalEditSemanticAttestation{})
			payload := eventPayload(app.LedgerEvent{Payload: request.Payload})
			mutate(payload)
			request.Payload = finalEditStageStoreJSON(payload)
			if _, err := svc.AppendEvent(ctx, request); err != nil {
				t.Fatal(err)
			}
			_, _, err = LoadFinalEditStageSubmission(ctx, svc, readerBinding)
			if !errors.Is(err, app.ErrConflict) {
				t.Fatalf("non-style diagnosis %s error=%v, want conflict", name, err)
			}
		})
	}
}

func TestFinalEditStyleDiagnosisReplayStrictWhenPresent(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		operationCount int
		diagnoses      []FinalEditStyleOperationDiagnosis
		mutate         func(map[string]any)
	}{
		"null":             {operationCount: 1, diagnoses: finalEditStyleDiagnosesForTest(1), mutate: func(payload map[string]any) { payload[FinalEditStyleOperationDiagnosesField] = nil }},
		"non_array":        {operationCount: 1, diagnoses: finalEditStyleDiagnosesForTest(1), mutate: func(payload map[string]any) { payload[FinalEditStyleOperationDiagnosesField] = "invalid" }},
		"count_mismatch":   {operationCount: 2, diagnoses: finalEditStyleDiagnosesForTest(1), mutate: func(payload map[string]any) {}},
		"unknown_field":    {operationCount: 1, diagnoses: finalEditStyleDiagnosesForTest(1), mutate: func(payload map[string]any) { diagnosisRecords(payload)[0]["summary"] = "do not store" }},
		"unknown_category": {operationCount: 1, diagnoses: finalEditStyleDiagnosesForTest(1), mutate: func(payload map[string]any) { diagnosisRecords(payload)[0]["category"] = "unknown_category" }},
		"empty_reason":     {operationCount: 1, diagnoses: finalEditStyleDiagnosesForTest(1), mutate: func(payload map[string]any) { diagnosisRecords(payload)[0]["reason"] = "" }},
		"empty_match_text": {operationCount: 1, diagnoses: finalEditStyleDiagnosesForTest(1), mutate: func(payload map[string]any) { diagnosisRecords(payload)[0]["match_text"] = "" }},
		"bad_occurrence":   {operationCount: 1, diagnoses: finalEditStyleDiagnosesForTest(1), mutate: func(payload map[string]any) { diagnosisRecords(payload)[0]["occurrence"] = float64(0) }},
		"partial_detail":   {operationCount: 1, diagnoses: finalEditStyleDiagnosesForTest(1), mutate: func(payload map[string]any) { delete(diagnosisRecords(payload)[0], "reason") }},
		"missing_replacement": {operationCount: 1, diagnoses: finalEditStyleDiagnosesForTest(1), mutate: func(payload map[string]any) {
			delete(diagnosisRecords(payload)[0], "replacement")
		}},
		"bad_version":     {operationCount: 1, diagnoses: finalEditStyleDiagnosesForTest(1), mutate: func(payload map[string]any) { payload[FinalEditStyleOperationDiagnosesVersionField] = float64(2) }},
		"missing_version": {operationCount: 1, diagnoses: finalEditStyleDiagnosesForTest(1), mutate: func(payload map[string]any) { delete(payload, FinalEditStyleOperationDiagnosesVersionField) }},
		"duplicate":       {operationCount: 2, diagnoses: finalEditStyleDiagnosesForTest(2), mutate: func(payload map[string]any) { diagnosisRecords(payload)[1]["operation_ordinal"] = float64(1) }},
		"out_of_sequence": {operationCount: 2, diagnoses: finalEditStyleDiagnosesForTest(2), mutate: func(payload map[string]any) { diagnosisRecords(payload)[0]["operation_ordinal"] = float64(2) }},
		"changed_zero":    {operationCount: 0, diagnoses: nil, mutate: func(payload map[string]any) {}},
	} {
		t.Run(name, func(t *testing.T) {
			svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeEnabled)
			defer closeStore()
			_, reader, styleBinding := startStyleDiagnosisTestStage(t, ctx, svc, "style_strict_"+name)
			source, artifact := styleDiagnosisChangedArtifacts(t, reader.Artifact, styleBinding)
			request := buildFinalEditSubmittedAppendRequestWithStyleDiagnoses("evt_style_strict_"+name+"_submit", styleBinding, source, artifact, tc.operationCount, true, tc.diagnoses, nil, FinalEditSemanticAttestation{})
			payload := eventPayload(app.LedgerEvent{Payload: request.Payload})
			tc.mutate(payload)
			request.Payload = finalEditStageStoreJSON(payload)
			appendStyleDiagnosisArtifactAndEvent(t, ctx, svc, artifact, request)
			_, _, err := LoadFinalEditStageSubmission(ctx, svc, styleBinding)
			if !errors.Is(err, app.ErrConflict) {
				t.Fatalf("strict replay error=%v, want conflict", err)
			}
		})
	}
}

func startStyleDiagnosisTestStage(t *testing.T, ctx context.Context, svc *app.Service, suffix string) (LongFormFinalizeBinding, finalEditStageStoreReaderResult, FinalEditStageBinding) {
	t.Helper()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeEnabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, suffix)
	styleBinding := finalEditStageStoreStageBinding(binding, FinalEditStageStyle, reader.Artifact.ArtifactID, "art_style_"+suffix)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_style_"+suffix+"_start", styleBinding); err != nil || !created {
		t.Fatalf("style start created=%t err=%v", created, err)
	}
	return binding, reader, styleBinding
}

func styleDiagnosisChangedArtifacts(t *testing.T, source app.RawArtifact, styleBinding FinalEditStageBinding) (app.RawArtifact, app.RawArtifact) {
	t.Helper()
	artifact := app.RawArtifact{
		ArtifactID: styleBinding.EditedArtifactID, MissionID: styleBinding.MissionID, MediaType: "text/markdown; charset=utf-8",
		Filename: styleBinding.Filename, Producer: app.Producer{Type: "agent_session", ID: styleBinding.ProviderSessionID},
		Content: []byte(strings.Replace(string(source.Content), "Preserved body.", "Preserved body!", 1)),
	}
	artifact.SHA256 = contentSHA256(artifact.Content)
	return source, artifact
}

func appendStyleDiagnosisArtifactAndEvent(t *testing.T, ctx context.Context, svc *app.Service, artifact app.RawArtifact, request app.AppendEventRequest) {
	t.Helper()
	if _, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: artifact.ArtifactID, MissionID: artifact.MissionID, MediaType: artifact.MediaType,
		Filename: artifact.Filename, Producer: artifact.Producer, Content: artifact.Content,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, request); err != nil {
		t.Fatal(err)
	}
}

func diagnosisRecords(payload map[string]any) []map[string]any {
	values := payload[FinalEditStyleOperationDiagnosesField].([]any)
	records := make([]map[string]any, 0, len(values))
	for _, value := range values {
		records = append(records, value.(map[string]any))
	}
	return records
}

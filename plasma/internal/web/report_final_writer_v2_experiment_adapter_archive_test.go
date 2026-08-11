package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type finalWriterV2StoredMachineCheck struct {
	ExperimentID                 string           `json:"experiment_id"`
	EvidenceVersion              string           `json:"evidence_version"`
	PairID                       string           `json:"pair_id"`
	Arm                          string           `json:"arm"`
	Pipeline                     string           `json:"pipeline"`
	MissionID                    string           `json:"mission_id"`
	PendingEventID               string           `json:"pending_event_id"`
	PlanEventID                  string           `json:"plan_event_id"`
	InputManifestSHA256          string           `json:"input_manifest_sha256"`
	ReportSHA256                 string           `json:"report_sha256"`
	DBPath                       string           `json:"db_path"`
	LedgerEventsPath             string           `json:"ledger_events_path"`
	LedgerEventsSHA256           string           `json:"ledger_events_sha256"`
	StageTrace                   []map[string]any `json:"stage_trace"`
	StageTraceErrors             []string         `json:"stage_trace_errors"`
	InformationLossCount         int              `json:"information_loss_count"`
	CitationLossCount            int              `json:"citation_loss_count"`
	RequirementLossCount         int              `json:"requirement_loss_count"`
	UnsupportedExternalFactCount int              `json:"unsupported_external_fact_count"`
	ProductPromptStageParity     string           `json:"product_prompt_stage_parity"`
	PairIdentity                 string           `json:"pair_identity"`
	Blinding                     string           `json:"blinding"`
	ArchiveAdoption              string           `json:"archive_adoption"`
	StagePayloadContract         string           `json:"stage_payload_contract"`
}

type finalWriterV2ManualAdjudication struct {
	ExperimentID       string                          `json:"experiment_id"`
	EvidenceVersion    string                          `json:"evidence_version"`
	AdjudicatedAt      string                          `json:"adjudicated_at"`
	BlindMappingSHA256 string                          `json:"blind_mapping_sha256"`
	BlindPackSHA256    map[string]string               `json:"blind_pack_sha256"`
	Pairs              []finalWriterV2PairAdjudication `json:"pairs"`
}

type finalWriterV2PairAdjudication struct {
	PairID                    string         `json:"pair_id"`
	DirectReadingWinner       string         `json:"direct_reading_winner"`
	UnsupportedExternalFacts  map[string]int `json:"unsupported_external_facts"`
	V2StructuralRegression    bool           `json:"v2_structural_regression"`
	StructuralRegressionNotes string         `json:"structural_regression_notes"`
	InferenceBoundary         string         `json:"inference_boundary"`
	ReadingNotes              string         `json:"reading_notes"`
}

func TestFinalWriterV2ExperimentAdapterRejectsMissingManualAdjudicationDefaults(t *testing.T) {
	_, err := loadFinalWriterV2ManualAdjudication(t.TempDir())
	if err == nil {
		t.Fatal("missing manual adjudication was accepted")
	}
}

func TestFinalWriterV2ExperimentAdapterRejectsArchiveAdoptionWithoutReportHashAndDBLineage(t *testing.T) {
	ctx := context.Background()
	pair := finalWriterV2ExperimentPairs()[0]
	archive := t.TempDir()
	manifest, digest := writeFinalWriterV2TestFrozenManifest(t, archive, pair)
	runDir := finalWriterV2RunDir(archive, pair, "A")
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(runDir, "report.md")
	if err := os.WriteFile(reportPath, []byte("# 리포트\n\n본문\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	check := map[string]any{
		"experiment_id": finalWriterV2ExperimentID, "evidence_version": finalWriterV2ExperimentRunNamespace,
		"pair_id": pair.PairID, "arm": "A", "pipeline": finalWriterV2PipelineForArm("A"),
		"mission_id": "mis_missing", "pending_event_id": "evt_missing_pending", "plan_event_id": "evt_missing_plan",
		"input_manifest_sha256": digest, "report_sha256": sha256Hex([]byte("# 리포트\n\n본문\n")),
		"db_path": filepath.Join(runDir, "plasma.db"), "ledger_events_path": filepath.Join(runDir, "ledger", "events.json"),
		"ledger_events_sha256": strings.Repeat("0", 64), "stage_trace": []map[string]any{}, "stage_trace_errors": []string{},
		"archive_adoption": "pass", "stage_payload_contract": "pass",
	}
	writeFinalWriterV2JSON(t, finalWriterV2CheckPath(archive, pair, "A"), check)
	cfg := finalWriterV2AdapterConfig{ArchiveRoot: archive, PostHumanize: reporting.FinalEditHumanizeDisabled}
	if _, ok := loadFinalWriterV2ExistingExperimentRun(t, ctx, cfg, pair, "A", manifest, digest); ok {
		t.Fatal("archive adoption accepted a run without a readable product DB and ledger lineage")
	}
}

func TestFinalWriterV2ExperimentAdapterRejectsStoredMachineCheckWithInvalidHardFailCounts(t *testing.T) {
	base := map[string]any{
		"information_loss_count": 0, "citation_loss_count": 0, "requirement_loss_count": 0, "unsupported_external_fact_count": 0,
	}
	valid, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	if reasons := finalWriterV2StoredHardFailCountReasons(valid); len(reasons) > 0 {
		t.Fatalf("rejected valid hard-fail counts: %v", reasons)
	}
	tests := []struct {
		name  string
		key   string
		value any
	}{
		{name: "missing", key: "information_loss_count", value: nil},
		{name: "non_integer", key: "citation_loss_count", value: 1.25},
		{name: "negative", key: "requirement_loss_count", value: -1},
		{name: "unsupported_missing", key: "unsupported_external_fact_count", value: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			check := map[string]any{}
			for key, value := range base {
				check[key] = value
			}
			if tc.value == nil {
				delete(check, tc.key)
			} else {
				check[tc.key] = tc.value
			}
			content, err := json.Marshal(check)
			if err != nil {
				t.Fatal(err)
			}
			if reasons := finalWriterV2StoredHardFailCountReasons(content); len(reasons) == 0 {
				t.Fatalf("accepted invalid hard-fail count %s", tc.key)
			}
		})
	}
}

func loadFinalWriterV2ExistingExperimentRun(t *testing.T, ctx context.Context, cfg finalWriterV2AdapterConfig, pair finalWriterV2ExperimentPair, arm string, manifest finalWriterV2FrozenManifest, manifestDigest string) (finalWriterV2ExperimentRun, bool) {
	t.Helper()
	reportPath := filepath.Join(finalWriterV2RunDir(cfg.ArchiveRoot, pair, arm), "report.md")
	report, err := os.ReadFile(reportPath)
	if err != nil || strings.TrimSpace(string(report)) == "" {
		return finalWriterV2ExperimentRun{}, false
	}
	checkBytes, err := os.ReadFile(finalWriterV2CheckPath(cfg.ArchiveRoot, pair, arm))
	if err != nil {
		return finalWriterV2ExperimentRun{}, false
	}
	var stored finalWriterV2StoredMachineCheck
	if err := json.Unmarshal(checkBytes, &stored); err != nil {
		return finalWriterV2ExperimentRun{}, false
	}
	if reasons := finalWriterV2StoredHardFailCountReasons(checkBytes); len(reasons) > 0 {
		return finalWriterV2ExperimentRun{}, false
	}
	run := finalWriterV2ExperimentRun{
		PairID: pair.PairID, Arm: arm, Pipeline: finalWriterV2PipelineForArm(arm), MissionID: stored.MissionID,
		PendingEventID: stored.PendingEventID, PlanEventID: stored.PlanEventID, ReportPath: reportPath,
		CheckPath: finalWriterV2CheckPath(cfg.ArchiveRoot, pair, arm), InputManifestSHA256: stored.InputManifestSHA256,
		ReportSHA256: stored.ReportSHA256, DBPath: stored.DBPath, LedgerEventsPath: stored.LedgerEventsPath,
		LedgerEventsSHA256: stored.LedgerEventsSHA256, StageTrace: stored.StageTrace,
	}
	if stored.ExperimentID != finalWriterV2ExperimentID || stored.EvidenceVersion != finalWriterV2ExperimentRunNamespace ||
		stored.PairID != pair.PairID || stored.Arm != arm || stored.Pipeline != finalWriterV2PipelineForArm(arm) ||
		stored.InputManifestSHA256 != manifestDigest || stored.ReportSHA256 != sha256Hex(report) ||
		stored.ArchiveAdoption != "pass" || stored.StagePayloadContract != "pass" || len(stored.StageTraceErrors) != 0 {
		return finalWriterV2ExperimentRun{}, false
	}
	if reasons := finalWriterV2PreReadingHardFailReasons(finalWriterV2MachineCheckFromStored(stored)); len(reasons) > 0 {
		return finalWriterV2ExperimentRun{}, false
	}
	if err := validateFinalWriterV2ArchiveRun(ctx, run, pair, arm, manifest, manifestDigest); err != nil {
		return finalWriterV2ExperimentRun{}, false
	}
	return run, true
}

func validateFinalWriterV2ArchiveRun(ctx context.Context, run finalWriterV2ExperimentRun, pair finalWriterV2ExperimentPair, arm string, manifest finalWriterV2FrozenManifest, manifestDigest string) error {
	if run.InputManifestSHA256 != manifestDigest || run.PairID != pair.PairID || run.Arm != arm || run.Pipeline != finalWriterV2PipelineForArm(arm) {
		return fmt.Errorf("archive identity mismatch for %s %s", pair.PairID, arm)
	}
	archive := finalWriterV2ArchiveRootFromRunPath(run.ReportPath)
	for _, path := range []string{run.ReportPath, run.DBPath, run.LedgerEventsPath, run.CheckPath} {
		if err := finalWriterV2PathInsideArchive(archive, path); err != nil {
			return err
		}
	}
	reportBytes, err := os.ReadFile(run.ReportPath)
	if err != nil {
		return err
	}
	if sha256Hex(reportBytes) != run.ReportSHA256 {
		return fmt.Errorf("report hash mismatch for %s %s", pair.PairID, arm)
	}
	if finalWriterV2SHA256FileNoErr(run.LedgerEventsPath) != run.LedgerEventsSHA256 {
		return fmt.Errorf("ledger events hash mismatch for %s %s", pair.PairID, arm)
	}
	svc, closeStore, err := openFinalWriterV2ExperimentServicePath(ctx, run.DBPath)
	if err != nil {
		return err
	}
	defer closeStore()
	events, err := svc.ListEvents(ctx, run.MissionID)
	if err != nil {
		return err
	}
	if err := finalWriterV2ValidateLedgerLineage(events, run, manifest); err != nil {
		return err
	}
	if err := finalWriterV2ValidateFrozenPartArtifacts(ctx, svc, run, manifest); err != nil {
		return err
	}
	if err := finalWriterV2ValidateFinalArtifact(ctx, svc, events, run, string(reportBytes)); err != nil {
		return err
	}
	if errors := finalWriterV2ValidateStoredStageTrace(run.StageTrace, arm, finalWriterV2TraceStyleEnabled(run.StageTrace)); len(errors) > 0 {
		return fmt.Errorf("stored stage trace mismatch: %s", strings.Join(errors, "; "))
	}
	return nil
}

func finalWriterV2ArchiveRootFromRunPath(reportPath string) string {
	root := filepath.Clean(reportPath)
	for i := 0; i < 5; i++ {
		root = filepath.Dir(root)
	}
	return root
}

func finalWriterV2ValidateLedgerLineage(events []app.LedgerEvent, run finalWriterV2ExperimentRun, manifest finalWriterV2FrozenManifest) error {
	foundPending, foundPlan := false, false
	for _, event := range events {
		switch event.EventID {
		case run.PendingEventID:
			foundPending = event.EventType == "report.draft.pending"
		case run.PlanEventID:
			foundPlan = event.EventType == "report.plan.created" && finalWriterV2EventString(event, "pending_event_id") == run.PendingEventID &&
				finalWriterV2EventString(event, "final_edit_pipeline") == run.Pipeline
		}
	}
	if !foundPending || !foundPlan {
		return fmt.Errorf("pending/plan lineage missing")
	}
	for _, part := range manifest.Parts {
		found := false
		for _, event := range events {
			if event.EventType == "report.part.created" &&
				finalWriterV2EventInt(event, "part_index") == part.PartIndex &&
				finalWriterV2EventString(event, "pending_event_id") == run.PendingEventID &&
				finalWriterV2EventString(event, "plan_event_id") == run.PlanEventID {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("part lineage missing for part %d", part.PartIndex)
		}
	}
	return nil
}

func finalWriterV2ValidateFrozenPartArtifacts(ctx context.Context, svc *app.Service, run finalWriterV2ExperimentRun, manifest finalWriterV2FrozenManifest) error {
	fragment := finalWriterV2IDFragment(run.PairID + "_" + run.Arm + "_" + finalWriterV2ExperimentRunNamespace)
	for _, part := range manifest.Parts {
		artifact, err := svc.GetRawArtifact(ctx, fmt.Sprintf("art_exp55_%s_part_%02d", fragment, part.PartIndex))
		if err != nil {
			return err
		}
		if artifact.SHA256 != part.SHA256 || string(artifact.Content) != part.Markdown {
			return fmt.Errorf("frozen Part artifact mismatch for part %d", part.PartIndex)
		}
	}
	return nil
}

func finalWriterV2ValidateFinalArtifact(ctx context.Context, svc *app.Service, events []app.LedgerEvent, run finalWriterV2ExperimentRun, report string) error {
	for _, event := range events {
		if event.EventType != "report.artifact.created" || finalWriterV2EventString(event, "pending_event_id") != run.PendingEventID {
			continue
		}
		artifactID := finalWriterV2EventString(event, "artifact_id")
		if artifactID == "" {
			return fmt.Errorf("final report event is missing artifact_id")
		}
		artifact, err := svc.GetRawArtifact(ctx, artifactID)
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(artifact.Content)) != strings.TrimSpace(report) {
			return fmt.Errorf("final artifact content differs from report markdown")
		}
		return nil
	}
	return fmt.Errorf("final report artifact event missing")
}

func finalWriterV2MachineCheck(pair finalWriterV2ExperimentPair, arm string, req finalizationPrefixFixture, manifest finalWriterV2FrozenManifest, manifestDigest string, run finalWriterV2ExperimentRun, markdown string, trace []map[string]any, traceErrors []string) map[string]any {
	citationLoss := 0
	for _, part := range manifest.Parts {
		for _, token := range finalWriterV2ProtectedCitationTokens(part.Markdown) {
			if !strings.Contains(markdown, token) {
				citationLoss++
			}
		}
	}
	informationLoss := finalWriterV2InformationLossCount(pair, markdown)
	requirementLoss := finalWriterV2RequirementLossCount(pair, markdown)
	payloadContract := finalWriterV2PassFail(len(traceErrors) == 0 && finalWriterV2StagePayloadContractOK(trace, arm))
	return map[string]any{
		"experiment_id": finalWriterV2ExperimentID, "evidence_version": finalWriterV2ExperimentRunNamespace,
		"pair_id": pair.PairID, "arm": arm, "pipeline": finalWriterV2PipelineForArm(arm),
		"mission_id": req.missionID, "pending_event_id": req.pendingEventID, "plan_event_id": req.planEvent.EventID,
		"input_manifest_sha256": manifestDigest, "report_sha256": run.ReportSHA256, "db_path": run.DBPath,
		"ledger_events_path": run.LedgerEventsPath, "ledger_events_sha256": run.LedgerEventsSHA256,
		"information_loss_count": informationLoss, "citation_loss_count": citationLoss, "requirement_loss_count": requirementLoss,
		"unsupported_external_fact_count": nil, "product_prompt_stage_parity": finalWriterV2PassFail(len(traceErrors) == 0),
		"pair_identity": finalWriterV2PassFail(manifest.PairID == pair.PairID), "blinding": "pass",
		"stage_trace_errors": traceErrors, "stage_trace": trace, "archive_adoption": "pass", "stage_payload_contract": payloadContract,
		"hard_fail": informationLoss > 0 || citationLoss > 0 || requirementLoss > 0 || len(traceErrors) > 0 || payloadContract != "pass",
	}
}

func finalWriterV2PreReadingHardFailReasons(check map[string]any) []string {
	reasons := []string{}
	for _, key := range []string{"information_loss_count", "citation_loss_count", "requirement_loss_count"} {
		if value, ok := finalWriterV2IntFromAny(check[key]); ok && value > 0 {
			reasons = append(reasons, fmt.Sprintf("%s=%d", key, value))
		}
	}
	for _, key := range []string{"product_prompt_stage_parity", "pair_identity", "blinding", "archive_adoption", "stage_payload_contract"} {
		if value, _ := check[key].(string); value != "pass" {
			reasons = append(reasons, fmt.Sprintf("%s=%s", key, value))
		}
	}
	if value, ok := check["stage_trace_errors"].([]string); ok && len(value) > 0 {
		reasons = append(reasons, "stage_trace_errors")
	}
	return reasons
}

func finalWriterV2StoredHardFailCountReasons(content []byte) []string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(content, &raw); err != nil {
		return []string{"hard_fail_counts=invalid_json"}
	}
	reasons := []string{}
	for _, key := range []string{"information_loss_count", "citation_loss_count", "requirement_loss_count", "unsupported_external_fact_count"} {
		count, ok := raw[key]
		if !ok {
			reasons = append(reasons, key+"=missing")
			continue
		}
		var value int
		if err := json.Unmarshal(count, &value); err != nil {
			reasons = append(reasons, key+"=non_integer")
			continue
		}
		if value < 0 {
			reasons = append(reasons, fmt.Sprintf("%s=%d", key, value))
		}
	}
	return reasons
}

func finalWriterV2HardFailReasons(check map[string]any) []string {
	reasons := finalWriterV2PreReadingHardFailReasons(check)
	value, ok := finalWriterV2IntFromAny(check["unsupported_external_fact_count"])
	if !ok {
		return append(reasons, "unsupported_external_fact_count=missing_manual_adjudication")
	}
	if value > 0 {
		reasons = append(reasons, fmt.Sprintf("unsupported_external_fact_count=%d", value))
	}
	return reasons
}

func loadFinalWriterV2ManualAdjudication(archive string) (finalWriterV2ManualAdjudication, error) {
	path := filepath.Join(archive, "control", "manual-adjudication.json")
	content, err := os.ReadFile(path)
	if err != nil {
		return finalWriterV2ManualAdjudication{}, err
	}
	var raw struct {
		ExperimentID       string                       `json:"experiment_id"`
		EvidenceVersion    string                       `json:"evidence_version"`
		AdjudicatedAt      string                       `json:"adjudicated_at"`
		BlindMappingSHA256 string                       `json:"blind_mapping_sha256"`
		BlindPackSHA256    map[string]string            `json:"blind_pack_sha256"`
		Pairs              []map[string]json.RawMessage `json:"pairs"`
	}
	if err := json.Unmarshal(content, &raw); err != nil {
		return finalWriterV2ManualAdjudication{}, err
	}
	if raw.ExperimentID != finalWriterV2ExperimentID || raw.EvidenceVersion != finalWriterV2ExperimentRunNamespace || strings.TrimSpace(raw.AdjudicatedAt) == "" {
		return finalWriterV2ManualAdjudication{}, fmt.Errorf("manual adjudication identity is invalid")
	}
	expected := map[string]bool{}
	for _, pair := range finalWriterV2ExperimentPairs() {
		expected[pair.PairID] = true
	}
	result := finalWriterV2ManualAdjudication{
		ExperimentID: raw.ExperimentID, EvidenceVersion: raw.EvidenceVersion, AdjudicatedAt: raw.AdjudicatedAt,
		BlindMappingSHA256: raw.BlindMappingSHA256, BlindPackSHA256: raw.BlindPackSHA256,
	}
	seen := map[string]bool{}
	for _, row := range raw.Pairs {
		required := []string{"pair_id", "direct_reading_winner", "unsupported_external_facts", "v2_structural_regression", "inference_boundary", "reading_notes"}
		for _, key := range required {
			if _, ok := row[key]; !ok {
				return finalWriterV2ManualAdjudication{}, fmt.Errorf("manual adjudication missing %s", key)
			}
		}
		if _, err := finalWriterV2RawExplicitBool(row["v2_structural_regression"], "manual adjudication v2_structural_regression"); err != nil {
			return finalWriterV2ManualAdjudication{}, err
		}
		var item finalWriterV2PairAdjudication
		encoded, _ := json.Marshal(row)
		if err := json.Unmarshal(encoded, &item); err != nil {
			return finalWriterV2ManualAdjudication{}, err
		}
		if !expected[item.PairID] || seen[item.PairID] {
			return finalWriterV2ManualAdjudication{}, fmt.Errorf("manual adjudication pair set is invalid")
		}
		if item.DirectReadingWinner != "A" && item.DirectReadingWinner != "B" && item.DirectReadingWinner != "tie" {
			return finalWriterV2ManualAdjudication{}, fmt.Errorf("manual adjudication winner is invalid for %s", item.PairID)
		}
		if len(item.UnsupportedExternalFacts) != 2 {
			return finalWriterV2ManualAdjudication{}, fmt.Errorf("manual adjudication unsupported fact map is invalid for %s", item.PairID)
		}
		for _, arm := range []string{"A", "B"} {
			value, ok := item.UnsupportedExternalFacts[arm]
			if !ok || value < 0 {
				return finalWriterV2ManualAdjudication{}, fmt.Errorf("manual adjudication unsupported fact count is invalid for %s %s", item.PairID, arm)
			}
		}
		if strings.TrimSpace(item.InferenceBoundary) == "" || strings.TrimSpace(item.ReadingNotes) == "" {
			return finalWriterV2ManualAdjudication{}, fmt.Errorf("manual adjudication notes are required for %s", item.PairID)
		}
		seen[item.PairID] = true
		result.Pairs = append(result.Pairs, item)
	}
	if len(seen) != len(expected) {
		return finalWriterV2ManualAdjudication{}, fmt.Errorf("manual adjudication must cover exactly the four pairs")
	}
	mappingSHA, packSHA, err := finalWriterV2BlindSeal(archive)
	if err != nil {
		return finalWriterV2ManualAdjudication{}, err
	}
	if raw.BlindMappingSHA256 != mappingSHA || !finalWriterV2StringMapEqual(raw.BlindPackSHA256, packSHA) {
		return finalWriterV2ManualAdjudication{}, fmt.Errorf("manual adjudication blind seal mismatch")
	}
	return result, nil
}

func writeFinalWriterV2ReadingResults(t *testing.T, archive string, runs []finalWriterV2ExperimentRun, adjudication finalWriterV2ManualAdjudication) (bool, error) {
	t.Helper()
	adjudicationSHA := finalWriterV2SHA256FileNoErr(filepath.Join(archive, "control", "manual-adjudication.json"))
	mappingSHA, packSHA, err := finalWriterV2BlindSeal(archive)
	if err != nil {
		return false, err
	}
	byPairArm := map[string]map[string]finalWriterV2ExperimentRun{}
	for _, run := range runs {
		if byPairArm[run.PairID] == nil {
			byPairArm[run.PairID] = map[string]finalWriterV2ExperimentRun{}
		}
		byPairArm[run.PairID][run.Arm] = run
	}
	adjudicationByPair := map[string]finalWriterV2PairAdjudication{}
	for _, row := range adjudication.Pairs {
		adjudicationByPair[row.PairID] = row
	}
	pairResults := []map[string]any{}
	for _, pair := range finalWriterV2ExperimentPairs() {
		row := adjudicationByPair[pair.PairID]
		hard := false
		counts := map[string]int{"information_loss_count": 0, "citation_loss_count": 0, "requirement_loss_count": 0, "unsupported_external_fact_count": 0}
		status := map[string]string{"product_prompt_stage_parity": "pass", "pair_identity": "pass", "blinding": "pass", "archive_adoption": "pass", "stage_payload_contract": "pass"}
		stageErrors := []string{}
		for _, arm := range []string{"A", "B"} {
			run, ok := byPairArm[pair.PairID][arm]
			if !ok {
				return false, fmt.Errorf("missing run for %s %s", pair.PairID, arm)
			}
			checkBytes, err := os.ReadFile(run.CheckPath)
			if err != nil {
				return false, err
			}
			var check map[string]any
			if err := json.Unmarshal(checkBytes, &check); err != nil {
				return false, err
			}
			check["unsupported_external_fact_count"] = row.UnsupportedExternalFacts[arm]
			check["manual_adjudication_sha256"] = adjudicationSHA
			check["inference_boundary"] = row.InferenceBoundary
			check["hard_fail"] = len(finalWriterV2HardFailReasons(check)) > 0
			writeFinalWriterV2JSON(t, run.CheckPath, check)
			for key := range counts {
				if value, ok := finalWriterV2IntFromAny(check[key]); ok {
					counts[key] += value
				}
			}
			for key := range status {
				if value, _ := check[key].(string); value != "pass" {
					status[key] = value
				}
			}
			stageErrors = append(stageErrors, finalWriterV2StringsFromAny(check["stage_trace_errors"])...)
			hard = hard || len(finalWriterV2HardFailReasons(check)) > 0
		}
		pairResult := map[string]any{
			"pair_id": pair.PairID, "winner": row.DirectReadingWinner, "hard_fail": hard,
			"v2_structural_regression": row.V2StructuralRegression, "inference_boundary": row.InferenceBoundary,
			"reading_notes": row.ReadingNotes, "stage_trace_errors": stageErrors,
		}
		for key, value := range counts {
			pairResult[key] = value
		}
		for key, value := range status {
			pairResult[key] = value
		}
		pairResults = append(pairResults, pairResult)
	}
	accepted, summary := finalWriterV2Acceptance(pairResults)
	writeFinalWriterV2JSON(t, filepath.Join(archive, "control", "reading-results.json"), map[string]any{
		"experiment_id": finalWriterV2ExperimentID, "evidence_version": finalWriterV2ExperimentRunNamespace,
		"adjudication_sha256": adjudicationSHA, "blind_mapping_sha256": mappingSHA, "blind_pack_sha256": packSHA,
		"accepted": accepted, "summary": summary, "pairs": pairResults,
	})
	return accepted, nil
}

func finalWriterV2ValidateReadingResultsSeal(archive string) error {
	content, err := os.ReadFile(filepath.Join(archive, "control", "reading-results.json"))
	if err != nil {
		return err
	}
	var result struct {
		ExperimentID       string            `json:"experiment_id"`
		EvidenceVersion    string            `json:"evidence_version"`
		BlindMappingSHA256 string            `json:"blind_mapping_sha256"`
		BlindPackSHA256    map[string]string `json:"blind_pack_sha256"`
	}
	if err := json.Unmarshal(content, &result); err != nil {
		return err
	}
	if result.ExperimentID != finalWriterV2ExperimentID || result.EvidenceVersion != finalWriterV2ExperimentRunNamespace {
		return fmt.Errorf("reading result identity mismatch")
	}
	mappingSHA, packSHA, err := finalWriterV2BlindSeal(archive)
	if err != nil {
		return err
	}
	if result.BlindMappingSHA256 != mappingSHA || !finalWriterV2StringMapEqual(result.BlindPackSHA256, packSHA) {
		return fmt.Errorf("reading result blind seal mismatch")
	}
	return nil
}

func finalWriterV2RawExplicitBool(raw json.RawMessage, label string) (bool, error) {
	trimmed := bytes.TrimSpace(raw)
	if !bytes.Equal(trimmed, []byte("true")) && !bytes.Equal(trimmed, []byte("false")) {
		return false, fmt.Errorf("%s must be an explicit boolean", label)
	}
	var value bool
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return false, err
	}
	return value, nil
}

func finalWriterV2StringMapEqual(a map[string]string, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func finalWriterV2Acceptance(results []map[string]any) (bool, map[string]any) {
	equalOrBetter := 0
	hardFailures := []string{}
	regressions := map[string]map[string]bool{}
	for _, row := range results {
		pairID, _ := row["pair_id"].(string)
		winner, _ := row["winner"].(string)
		if winner == "B" || winner == "tie" {
			equalOrBetter++
		}
		if hard, _ := row["hard_fail"].(bool); hard {
			hardFailures = append(hardFailures, pairID)
		}
		if regression, _ := row["v2_structural_regression"].(bool); regression {
			for _, pair := range finalWriterV2ExperimentPairs() {
				if pair.PairID == pairID {
					if regressions[pair.TopicID] == nil {
						regressions[pair.TopicID] = map[string]bool{}
					}
					regressions[pair.TopicID][pair.Rigor] = true
				}
			}
		}
	}
	repeated := []string{}
	for topic, rigors := range regressions {
		if rigors["exploratory"] && rigors["strict"] {
			repeated = append(repeated, topic)
		}
	}
	slices.Sort(repeated)
	accepted := equalOrBetter >= 3 && len(hardFailures) == 0 && len(repeated) == 0
	return accepted, map[string]any{
		"candidate_arm": "B", "equal_or_better_pairs": equalOrBetter,
		"hard_failure_pairs": hardFailures, "repeated_structural_regression_topics": repeated,
	}
}

func finalWriterV2MachineCheckFromStored(stored finalWriterV2StoredMachineCheck) map[string]any {
	return map[string]any{
		"information_loss_count": stored.InformationLossCount, "citation_loss_count": stored.CitationLossCount, "requirement_loss_count": stored.RequirementLossCount,
		"unsupported_external_fact_count": stored.UnsupportedExternalFactCount,
		"product_prompt_stage_parity":     stored.ProductPromptStageParity,
		"pair_identity":                   stored.PairIdentity, "blinding": stored.Blinding, "archive_adoption": stored.ArchiveAdoption,
		"stage_payload_contract": stored.StagePayloadContract, "stage_trace_errors": stored.StageTraceErrors,
	}
}

func finalWriterV2StagePayloadContractOK(trace []map[string]any, arm string) bool {
	expected := finalWriterV2ExpectedTraceRows(arm, finalWriterV2TraceStyleEnabled(trace))
	if len(trace) != len(expected) {
		return false
	}
	for index, want := range expected {
		row := trace[index]
		if want.providerStage == "" {
			if len(finalWriterV2StringsFromAny(row["tools"])) != 0 {
				return false
			}
			continue
		}
		if strings.TrimSpace(fmt.Sprint(row["source_artifact_id"])) == "" ||
			strings.TrimSpace(fmt.Sprint(row["provider_session_id"])) == "" ||
			strings.TrimSpace(fmt.Sprint(row["fork_source_agent_session_id"])) == "" ||
			!slices.Equal(finalWriterV2StringsFromAny(row["tools"]), want.tools) {
			return false
		}
	}
	return true
}

func finalWriterV2EventString(event app.LedgerEvent, key string) string {
	var payload map[string]any
	if json.Unmarshal(event.Payload, &payload) != nil {
		return ""
	}
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}

func finalWriterV2EventInt(event app.LedgerEvent, key string) int {
	var payload map[string]any
	if json.Unmarshal(event.Payload, &payload) != nil {
		return 0
	}
	switch value := payload[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		return 0
	}
}

func writeFinalWriterV2InvalidW6ANote(t *testing.T, archive string, now time.Time) {
	t.Helper()
	writeFinalWriterV2JSON(t, filepath.Join(archive, "control", "w6-a-invalid-directional.json"), map[string]any{
		"experiment_id": finalWriterV2ExperimentID, "invalidated_at": now.Format(time.RFC3339Nano),
		"status": "rejected_directional_only",
		"reasons": []string{
			"reviewed Part inputs were hand-authored synthetic fixtures rather than product-reviewed Korean Parts",
			"manual semantic fields were defaulted instead of loaded from explicit adjudication",
			"archive adoption did not revalidate DB, ledger, report hash, manifest digest, and stage payload contracts",
		},
	})
}

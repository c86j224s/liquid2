package web

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
)

func TestFinalWriterV2ProtectedCitationTokensCoversEveryExperimentTag(t *testing.T) {
	markdown := "[WANG-1] [WANG-5] [WANG-6] [RAFT-1] [RAFT-5] [RAFT-6] [T-1] [T-2] [WANG-5]"
	want := []string{"[WANG-1]", "[WANG-5]", "[WANG-6]", "[RAFT-1]", "[RAFT-5]", "[RAFT-6]", "[T-1]", "[T-2]"}
	if got := finalWriterV2ProtectedCitationTokens(markdown); !slices.Equal(got, want) {
		t.Fatalf("protected citation tokens=%#v, want %#v", got, want)
	}
}

func TestFinalWriterV2RaftInformationLossTracksTermVoteAndLogSeparately(t *testing.T) {
	pair := finalWriterV2ExperimentPair{TopicID: "go-raft-implementation-roadmap"}
	markdown := "term log randomized AppendEntries snapshot membership simulator storage transport observability"
	if got := finalWriterV2InformationLossCount(pair, markdown); got != 1 {
		t.Fatalf("information loss count=%d, want 1 missing vote", got)
	}

	markdown = "term randomized AppendEntries snapshot membership simulator storage transport observability"
	if got := finalWriterV2InformationLossCount(pair, markdown); got != 2 {
		t.Fatalf("information loss count=%d, want 2 missing vote and log", got)
	}
}

func TestFinalWriterV2BlindMappingIsReusedAfterFirstCreation(t *testing.T) {
	archive := t.TempDir()
	mapping := map[string]map[string]string{}
	for index, pair := range finalWriterV2ExperimentPairs() {
		if index%2 == 0 {
			mapping[pair.PairID] = map[string]string{"report_1": "A", "report_2": "B"}
		} else {
			mapping[pair.PairID] = map[string]string{"report_1": "B", "report_2": "A"}
		}
	}
	path := filepath.Join(archive, "control", "blind_mapping."+finalWriterV2ExperimentRunNamespace+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(mapping)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}

	got := loadOrCreateFinalWriterV2BlindMapping(t, archive)
	for pairID, labels := range mapping {
		if !slices.Equal([]string{got[pairID]["report_1"], got[pairID]["report_2"]}, []string{labels["report_1"], labels["report_2"]}) {
			t.Fatalf("blind mapping changed for %s: got=%#v want=%#v", pairID, got[pairID], labels)
		}
	}
}

func TestFinalWriterV2ExperimentAdapterPrepProvenanceReplaysDBLedgerAndArtifacts(t *testing.T) {
	ctx := context.Background()
	archive := t.TempDir()
	pair := finalWriterV2ExperimentPairs()[0]
	manifest, _ := writeFinalWriterV2ReplayableTestFrozenManifest(t, ctx, archive, pair)
	if err := finalWriterV2PrepProvenanceValid(ctx, archive, manifest); err != nil {
		t.Fatalf("valid prep provenance was rejected: %v", err)
	}

	tampered := manifest
	tampered.Prep.DBPath = archive + "-evil/plasma.db"
	if err := finalWriterV2PrepProvenanceValid(ctx, archive, tampered); err == nil {
		t.Fatal("archive prefix sibling prep DB path was accepted")
	}

	tampered = manifest
	tampered.Prep.SourceEventIDs = append([]string(nil), manifest.Prep.SourceEventIDs...)
	tampered.Prep.SourceEventIDs[0] = "evt_missing_source"
	if err := finalWriterV2PrepProvenanceValid(ctx, archive, tampered); err == nil {
		t.Fatal("forged source event provenance was accepted")
	}

	sourcePath := filepath.Join(archive, "source-corpora", pair.TopicID, "source-01.md")
	if err := os.WriteFile(sourcePath, []byte("변조된 source corpus [T-1]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := finalWriterV2PrepProvenanceValid(ctx, archive, manifest); err == nil {
		t.Fatal("source corpus byte tampering was accepted")
	}
}

func TestFinalWriterV2ExperimentAdapterManualAdjudicationRequiresLiteralStructuralRegressionBool(t *testing.T) {
	archive := t.TempDir()
	writeFinalWriterV2TestBlindSeal(t, archive)
	for _, value := range []any{nil, "false", 0} {
		with := finalWriterV2TestManualAdjudication(value)
		writeFinalWriterV2JSON(t, filepath.Join(archive, "control", "manual-adjudication.json"), with)
		if _, err := loadFinalWriterV2ManualAdjudication(archive); err == nil {
			t.Fatalf("manual adjudication accepted malformed v2_structural_regression=%#v", value)
		}
	}

	valid := finalWriterV2TestManualAdjudication(false)
	mappingSHA, packSHA, err := finalWriterV2BlindSeal(archive)
	if err != nil {
		t.Fatal(err)
	}
	valid["blind_mapping_sha256"] = mappingSHA
	valid["blind_pack_sha256"] = packSHA
	writeFinalWriterV2JSON(t, filepath.Join(archive, "control", "manual-adjudication.json"), valid)
	if _, err := loadFinalWriterV2ManualAdjudication(archive); err != nil {
		t.Fatalf("literal boolean manual adjudication was rejected: %v", err)
	}
}

func TestFinalWriterV2ExperimentAdapterReadingResultsAreSealedToMappingAndPacks(t *testing.T) {
	archive := t.TempDir()
	writeFinalWriterV2TestBlindSeal(t, archive)
	mappingSHA, packSHA, err := finalWriterV2BlindSeal(archive)
	if err != nil {
		t.Fatal(err)
	}
	writeFinalWriterV2JSON(t, filepath.Join(archive, "control", "reading-results.json"), map[string]any{
		"experiment_id": finalWriterV2ExperimentID, "evidence_version": finalWriterV2ExperimentRunNamespace,
		"blind_mapping_sha256": mappingSHA, "blind_pack_sha256": packSHA,
	})
	if err := finalWriterV2ValidateReadingResultsSeal(archive); err != nil {
		t.Fatalf("fresh reading result seal was rejected: %v", err)
	}

	path := filepath.Join(archive, "reading-packs", finalWriterV2ExperimentRunNamespace, "blind", finalWriterV2ExperimentPairs()[0].PairID+".md")
	if err := os.WriteFile(path, []byte("# Blind Pair\n\nchanged body\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := finalWriterV2ValidateReadingResultsSeal(archive); err == nil {
		t.Fatal("stale reading result survived blind-pack mutation")
	}
}

func writeFinalWriterV2ReplayableTestFrozenManifest(t *testing.T, ctx context.Context, archive string, pair finalWriterV2ExperimentPair) (finalWriterV2FrozenManifest, string) {
	t.Helper()
	runDir := filepath.Join(archive, "prep-reviewed-parts", finalWriterV2ExperimentRunNamespace, pair.PairID)
	dbPath := filepath.Join(runDir, "plasma.db")
	svc, closeStore := openFinalWriterV2ExperimentService(t, ctx, dbPath)
	defer closeStore()
	fragment := finalWriterV2IDFragment("w6c_" + pair.PairID)
	missionID := "mis_" + fragment
	pendingID := "evt_" + fragment + "_pending"
	planID := "evt_" + fragment + "_plan"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: pair.TopicTitle}); err != nil {
		t.Fatal(err)
	}
	sourceDir := filepath.Join(archive, "source-corpora", pair.TopicID)
	if err := os.MkdirAll(sourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	sourceBytes := []byte("제품 상류 경로 검증용 source corpus입니다. [T-1]\n")
	sourcePath := filepath.Join(sourceDir, "source-01.md")
	if err := os.WriteFile(sourcePath, sourceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	sourceArtifactID := "art_" + fragment + "_source_01"
	sourceSnapshotID := "src_" + fragment + "_source_01"
	sourceEventID := "evt_" + fragment + "_source_01"
	if _, err := svc.CreateSourceSnapshotWithEvent(ctx, app.CreateSourceSnapshotWithEventRequest{
		Artifact: app.CreateRawArtifactRequest{
			ArtifactID: sourceArtifactID, MissionID: missionID, MediaType: "text/markdown; charset=utf-8",
			Filename: "source-01.md", Producer: app.Producer{Type: "user", ID: "experiment"}, Content: sourceBytes,
		},
		Snapshot: app.CreateSourceSnapshotRequest{
			SnapshotID: sourceSnapshotID, MissionID: missionID,
			Connector: app.ConnectorRef{ConnectorID: "experiment-archive", ConnectorType: app.SourceConnectorTypeFileUpload, ExternalSourceID: pair.TopicID + "/source-01.md", ConnectorVersion: finalWriterV2ExperimentRunNamespace},
			Title:     "source-01", Locators: json.RawMessage(`[{"locator_type":"full_text"}]`),
			Access: app.SourceAccess{Visibility: "private", License: "experiment-corpus", RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly},
		},
		Event: app.AppendEventRequest{EventID: sourceEventID, MissionID: missionID, EventType: sourceevents.SourceSnapshottedEventType, Producer: app.Producer{Type: "user", ID: "experiment"}},
	}); err != nil {
		t.Fatal(err)
	}
	appendFinalWriterV2PrepEvent(t, ctx, svc, missionID, pendingID, "report.draft.pending", map[string]any{"kind": "markdown_report_artifact_pending"})
	appendFinalWriterV2PrepEvent(t, ctx, svc, missionID, planID, "report.plan.created", map[string]any{"pending_event_id": pendingID})
	for index, eventType := range []string{"report.requirements.started", "report.requirements.mapped", "report.section.created", "report.part_plan.created", "report.part_assembly.submitted", "report.part.created", "report.part_edit.started"} {
		appendFinalWriterV2PrepEvent(t, ctx, svc, missionID, fmt.Sprintf("evt_%s_step_%02d", fragment, index), eventType, map[string]any{"pending_event_id": pendingID, "plan_event_id": planID})
	}
	partMarkdown := "# 검증 Part\n\n제품 상류 경로에서 검토된 한국어 Part 바이트입니다. [T-1]\n"
	partArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_" + fragment + "_part_01", MissionID: missionID, MediaType: "text/markdown; charset=utf-8",
		Filename: "part-01-edited.md", Producer: app.Producer{Type: "agent", ID: "experiment"}, Content: []byte(partMarkdown),
	})
	if err != nil {
		t.Fatal(err)
	}
	appendFinalWriterV2PrepEvent(t, ctx, svc, missionID, "evt_"+fragment+"_part_edited", "report.part.edited", map[string]any{"pending_event_id": pendingID, "plan_event_id": planID, "artifact_id": partArtifact.ArtifactID})
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(runDir, "ledger", "events.json")
	if err := writeFinalWriterV2JSONFilePath(ledgerPath, events); err != nil {
		t.Fatal(err)
	}
	return writeFinalWriterV2ManifestForTest(t, archive, pair, finalWriterV2PrepProvenance{
		ProductPath: "section_fanout_plan_requirement_sections_part_assembly_part_author", MissionID: missionID,
		PendingEventID: pendingID, PlanEventID: planID, DBPath: dbPath, LedgerEventsPath: ledgerPath,
		LedgerEventsSHA256: finalWriterV2SHA256FileNoErr(ledgerPath), SourceSnapshotIDs: []string{sourceSnapshotID},
		SourceArtifactIDs: []string{sourceArtifactID}, SourceEventIDs: []string{sourceEventID}, DiscardedFinalReport: true,
	}, []sectionalReportPartDraft{{Title: "검증 Part", Markdown: partMarkdown, ArtifactID: partArtifact.ArtifactID}})
}

func appendFinalWriterV2PrepEvent(t *testing.T, ctx context.Context, svc *app.Service, missionID string, eventID string, eventType string, payload map[string]any) {
	t.Helper()
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: eventID, MissionID: missionID, EventType: eventType, Producer: app.Producer{Type: "agent", ID: "experiment"},
		Payload: finalWriterV2MustJSON(payload),
	}); err != nil {
		t.Fatal(err)
	}
}

func writeFinalWriterV2ManifestForTest(t *testing.T, archive string, pair finalWriterV2ExperimentPair, provenance finalWriterV2PrepProvenance, parts []sectionalReportPartDraft) (finalWriterV2FrozenManifest, string) {
	t.Helper()
	manifest, digest, err := writeFinalWriterV2FrozenManifestFromParts(archive, pair, provenance, parts, []string{"art_section_test"})
	if err != nil {
		t.Fatal(err)
	}
	return manifest, digest
}

func writeFinalWriterV2TestBlindSeal(t *testing.T, archive string) {
	t.Helper()
	mapping := map[string]map[string]string{}
	for _, pair := range finalWriterV2ExperimentPairs() {
		mapping[pair.PairID] = map[string]string{"report_1": "A", "report_2": "B"}
		packPath := filepath.Join(archive, "reading-packs", finalWriterV2ExperimentRunNamespace, "blind", pair.PairID+".md")
		if err := os.MkdirAll(filepath.Dir(packPath), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(packPath, []byte("# Blind Pair\n\n## Report 1\n\nClean report.\n\n## Report 2\n\nClean report.\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeFinalWriterV2JSON(t, filepath.Join(archive, "control", "blind_mapping."+finalWriterV2ExperimentRunNamespace+".json"), mapping)
}

func finalWriterV2TestManualAdjudication(regression any) map[string]any {
	pairs := []map[string]any{}
	for _, pair := range finalWriterV2ExperimentPairs() {
		pairs = append(pairs, map[string]any{
			"pair_id": pair.PairID, "direct_reading_winner": "tie", "unsupported_external_facts": map[string]int{"A": 0, "B": 0},
			"v2_structural_regression": regression, "inference_boundary": "explicit boundary", "reading_notes": "explicit notes",
		})
	}
	return map[string]any{
		"experiment_id": finalWriterV2ExperimentID, "evidence_version": finalWriterV2ExperimentRunNamespace,
		"adjudicated_at": "2026-07-30T00:00:00Z", "pairs": pairs,
	}
}

package reportexperiment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func writeTestFixture(t *testing.T, archive, id string) string {
	t.Helper()
	dir := filepath.Join(archive, "fixtures", id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	partPath := filepath.Join(dir, "part-01.md")
	part := []byte("# 검증 Part\n\n제품 경로에서 이미 검토된 Part 바이트를 대신하는 테스트 고정 입력입니다. [T-1]\n")
	if err := os.WriteFile(partPath, part, 0o600); err != nil {
		t.Fatal(err)
	}
	fixture := Fixture{
		SchemaVersion: FixtureSchemaVersion,
		FixtureID:     "fixture-" + id,
		SourceProvenance: SourceProvenance{
			ProvenanceID: "prov-" + id, ProductCommit: "abc123", SourceID: "source-" + id,
		},
		ReportTitle:   "테스트 finalization 보고서",
		Rigor:         FixtureRigor{Level: "balanced", Label: "균형"},
		DirectionHint: "한국어 장문 보고서로 작성하고 bracket citation tag를 보존한다.",
		WritingContract: &reporting.ReportWritingContract{
			CentralQuestion: "검증 Part를 어떻게 하나의 보고서로 읽어야 하는가?",
			ReaderTakeaway:  "검토된 Part의 사실과 caveat를 보존한 최종 보고서를 만든다.",
			ReadingPath:     []string{"검토된 Part를 순서대로 읽는다.", "citation tag를 보존한다."},
			MustKeep:        []string{"bracket citation tag", "Part 순서"},
			VisualRole:      "시각 자료는 필요하지 않다.",
			ToneAndShape:    "명확한 한국어 장문 보고서 문체.",
		},
		GenerationGuidanceProfile: "",
		PostReportHumanize:        reporting.FinalEditHumanizeEnabled,
		Parts: []FixturePart{{
			Index: 1, Title: "검증 Part", Path: "part-01.md", SHA256: bytesSHA256(part),
		}},
	}
	encoded, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	fixturePath := filepath.Join(dir, "fixture.json")
	if err := os.WriteFile(fixturePath, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return fixturePath
}

func rewriteFixture(t *testing.T, fixturePath string, mutate func(*Fixture)) {
	t.Helper()
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var fixture Fixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	mutate(&fixture)
	encoded, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixturePath, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func nodeWasObserved(observations []NodeObservation, nodeID string) bool {
	for _, observation := range observations {
		if observation.NodeID == nodeID && observation.Terminal && !observation.Failed {
			return true
		}
	}
	return false
}

func terminalNodeIndex(observations []NodeObservation, nodeID string) int {
	for _, observation := range observations {
		if observation.NodeID == nodeID && observation.Terminal && !observation.Failed {
			return observation.Index
		}
	}
	return 0
}

func fileSHA256ForTest(t *testing.T, path string) string {
	t.Helper()
	sha, err := fileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha
}

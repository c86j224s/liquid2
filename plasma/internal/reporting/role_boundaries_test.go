package reporting

import (
	artifact "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"reflect"
	"testing"
)

func TestEvidenceReplayRequiresExactUnchangedSource(t *testing.T) {
	source := artifact.Raw{ArtifactID: "art_one", SHA256: "hash", Content: []byte("source")}
	if err := validateV3EvidenceGateReplay(source, source, 0, false, nil, FinalEditSemanticAttestation{}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		raw        artifact.Raw
		operations int
		changed    bool
		review     FinalEditSemanticAttestation
	}{
		{source, 1, false, FinalEditSemanticAttestation{}},
		{source, 0, true, FinalEditSemanticAttestation{}},
		{artifact.Raw{ArtifactID: "art_other", SHA256: "hash", Content: []byte("source")}, 0, false, FinalEditSemanticAttestation{}},
		{source, 0, false, FinalEditSemanticAttestation{Count: 1}},
	} {
		if err := validateV3EvidenceGateReplay(source, tc.raw, tc.operations, tc.changed, nil, tc.review); err == nil {
			t.Fatalf("accepted invalid replay: %+v", tc)
		}
	}
}

func TestMarkdownStructureIgnoresFencedSourceLines(t *testing.T) {
	text := "# Heading\n```text\n# Hidden\nSource: hidden\n```\nSource: visible\n## References\nreference item\n"
	if got := markdownHeadings(text); !reflect.DeepEqual(got, []string{"# Heading", "## References"}) {
		t.Fatalf("headings=%v", got)
	}
	if got := markdownSourceBearingLines(text); !reflect.DeepEqual(got, []string{"Source: visible", "reference item"}) {
		t.Fatalf("sources=%v", got)
	}
}

func TestHumanizationConservativeAndMeaningDrift(t *testing.T) {
	original := "# 안내\n\n이 작업은 승인 후 실행합니다.\n"
	if err := ValidateHumanizedMarkdown(original, original); err != nil {
		t.Fatal(err)
	}
	if err := ValidateHumanizedMarkdown(original, "# 안내\n\n이 작업은 승인 없이 실행합니다!\n"); err == nil {
		t.Fatal("accepted meaning and terminator drift")
	}
}

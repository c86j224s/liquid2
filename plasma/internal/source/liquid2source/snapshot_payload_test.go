package liquid2source

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestBuildSnapshotPayloadPreservesRuneRanges(t *testing.T) {
	document := Liquid2SourceDocument{Contents: []Liquid2SourceContent{{ContentID: "part", Content: "가🙂끝", Role: "body", Format: "text"}}}
	payload, locators, err := BuildSnapshotPayload(document, "art_test", "selected", []Liquid2ContentRange{{ContentID: "part", Start: 1, End: 2}})
	if err != nil {
		t.Fatal(err)
	}
	var artifact liquid2SnapshotArtifact
	if err := json.Unmarshal(payload, &artifact); err != nil {
		t.Fatal(err)
	}
	if len(artifact.Contents) != 1 || artifact.Contents[0].Content != "🙂" || artifact.Contents[0].Start != 1 || artifact.Contents[0].End != 2 || artifact.Reason != "selected" {
		t.Fatalf("unexpected payload: %s", payload)
	}
	var refs []liquid2SnapshotLocator
	if err := json.Unmarshal(locators, &refs); err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].ArtifactID != "art_test" || refs[0].Start != 1 || refs[0].End != 2 {
		t.Fatalf("unexpected locators: %s", locators)
	}
}

func TestBuildSnapshotPayloadRejectsDuplicateContentBeforeSelection(t *testing.T) {
	document := Liquid2SourceDocument{Contents: []Liquid2SourceContent{{ContentID: "part", Content: "first"}, {ContentID: "part", Content: "second"}}}
	_, _, err := BuildSnapshotPayload(document, "art_test", "", []Liquid2ContentRange{{ContentID: "missing", Start: 0, End: 1}})
	if !errors.Is(err, producterror.ErrInvalidInput) || err.Error() != producterror.ErrInvalidInput.Error()+": duplicate liquid2 content id" {
		t.Fatalf("unexpected error: %v", err)
	}
}

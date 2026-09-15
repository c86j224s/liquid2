package confluencesource

import (
	"encoding/json"
	"testing"
)

func TestSnapshotPayloadFullAndRuneRange(t *testing.T) {
	page := ConfluenceSourcePage{CloudID: "cloud", PageID: "123", BodyStorage: "<p>가🙂끝</p>", PlainText: "가🙂끝"}
	for _, tc := range []struct {
		selection ConfluenceRangeSelection
		count     int
		partial   bool
	}{{ConfluenceRangeSelection{}, 2, false}, {ConfluenceRangeSelection{ContentID: "plain_text", Start: 1, End: 2}, 1, true}} {
		data, refs, err := BuildSnapshotPayload(page, "art_one", "reason", tc.selection)
		if err != nil {
			t.Fatal(err)
		}
		var payload confluenceSnapshotArtifact
		var locators []confluenceSnapshotLocator
		if err := json.Unmarshal(data, &payload); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(refs, &locators); err != nil {
			t.Fatal(err)
		}
		if len(payload.Contents) != tc.count || len(locators) != tc.count || payload.Page.Partial != tc.partial || len(payload.Page.StorageSHA256) != 64 {
			t.Fatalf("payload=%s refs=%s", data, refs)
		}
		if tc.partial && (payload.Contents[0].Content != "🙂" || !locators[0].Partial || locators[0].LocatorType != "confluence_page_range") {
			t.Fatalf("range changed: %s %s", data, refs)
		}
	}
}

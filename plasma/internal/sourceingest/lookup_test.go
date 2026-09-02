package sourceingest

import (
	"context"
	"encoding/json"
	"testing"
)

func TestConfluencePageURLKeySupportsAllStableURLForms(t *testing.T) {
	for _, raw := range []string{
		"https://example.atlassian.net/wiki/pages/123/title",
		"https://example.atlassian.net/wiki/edit-v2/123",
		"https://example.atlassian.net/wiki/%70ages/123/title",
		"https://example.atlassian.net/wiki/%65dit-v2/123",
		"https://example.atlassian.net/wiki?spaceKey=ENG&pageId=123",
	} {
		if key, ok := confluencePageURLKey(raw); !ok || key != "example.atlassian.net\x00123" {
			t.Fatalf("%s => %q %v", raw, key, ok)
		}
	}
}

func TestExistingSourceSnapshotForURLMatchesConfluenceLocator(t *testing.T) {
	locators, err := json.Marshal([]map[string]string{{
		"locator_type": "confluence_page",
		"site_url":     "https://docs.atlassian.net/wiki",
		"page_id":      "123",
	}})
	if err != nil {
		t.Fatal(err)
	}
	store := &sourceCandidateServiceStore{
		sources: []SourceSnapshot{{
			SnapshotID: "src_confluence",
			MissionID:  "mis_1",
			Connector: ConnectorRef{
				ConnectorID:      "confluence",
				ConnectorType:    "confluence_cloud",
				ExternalSourceID: "site_docs.atlassian.net:123",
				ExternalURI:      "confluence://cloud/site_docs.atlassian.net/pages/123",
			},
			Locators: locators,
		}},
	}

	snapshot, ok, err := ExistingSourceSnapshotForURL(context.Background(), store, "mis_1", "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || snapshot.SnapshotID != "src_confluence" {
		t.Fatalf("expected Confluence source snapshot match, ok=%v snapshot=%#v", ok, snapshot)
	}
}

func TestExistingSourceSnapshotForURLDoesNotMatchDifferentConfluencePage(t *testing.T) {
	locators, err := json.Marshal([]map[string]string{{
		"locator_type": "confluence_page",
		"site_url":     "https://docs.atlassian.net/wiki",
		"page_id":      "123",
	}})
	if err != nil {
		t.Fatal(err)
	}
	store := &sourceCandidateServiceStore{
		sources: []SourceSnapshot{{
			SnapshotID: "src_confluence",
			MissionID:  "mis_1",
			Connector: ConnectorRef{
				ConnectorID:      "confluence",
				ConnectorType:    "confluence_cloud",
				ExternalSourceID: "site_docs.atlassian.net:123",
				ExternalURI:      "confluence://cloud/site_docs.atlassian.net/pages/123",
			},
			Locators: locators,
		}},
	}

	_, ok, err := ExistingSourceSnapshotForURL(context.Background(), store, "mis_1", "https://docs.atlassian.net/wiki/spaces/ENG/pages/456/Roadmap")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("different Confluence page must not match existing snapshot")
	}
}

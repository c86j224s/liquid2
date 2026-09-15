package confluencesource

import (
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
)

func TestNormalizeSearchAndBrowseLimitsKeepParityAndBounds(t *testing.T) {
	for _, test := range []struct {
		name  string
		input int
		want  int
	}{
		{name: "zero default", input: 0, want: 10},
		{name: "negative default", input: -1, want: 10},
		{name: "maximum", input: 100, want: 100},
		{name: "search capped", input: 101, want: 100},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := NormalizeSearchLimit(test.input); got != test.want {
				t.Fatalf("NormalizeSearchLimit(%d) = %d, want %d", test.input, got, test.want)
			}
			if got := NormalizeBrowseLimit(test.input); got != test.want {
				t.Fatalf("NormalizeBrowseLimit(%d) = %d, want %d", test.input, got, test.want)
			}
		})
	}

	search, err := NormalizeSearchRequest(ConfluenceSourceSearchRequest{
		MissionID: " mis_1 ", CloudID: " cloud_1 ", SiteURL: " https://cloud.example ",
		Query: " query ", Cursor: " cursor ", SpaceID: " space ", SpaceKey: " key ", Limit: 101,
	})
	if err != nil {
		t.Fatalf("NormalizeSearchRequest() error = %v", err)
	}
	if search.MissionID != "mis_1" || search.CloudID != "cloud_1" || search.SiteURL != "https://cloud.example" ||
		search.Query != "query" || search.Cursor != "cursor" || search.SpaceID != "space" || search.SpaceKey != "key" || search.Limit != 100 {
		t.Fatalf("NormalizeSearchRequest() = %#v", search)
	}

	spaceList, err := NormalizeSpaceListRequest(ConfluenceSpaceListRequest{MissionID: " mis_1 ", CloudID: " cloud_1 ", Cursor: " cursor ", Limit: 0})
	if err != nil {
		t.Fatalf("NormalizeSpaceListRequest() error = %v", err)
	}
	if spaceList.MissionID != "mis_1" || spaceList.CloudID != "cloud_1" || spaceList.Cursor != "cursor" || spaceList.Limit != 10 {
		t.Fatalf("NormalizeSpaceListRequest() = %#v", spaceList)
	}

	spacePages, err := NormalizeSpacePagesRequest(ConfluenceSpacePagesRequest{MissionID: "mis_1", CloudID: "cloud_1", SpaceID: " space ", Limit: 101})
	if err != nil {
		t.Fatalf("NormalizeSpacePagesRequest() error = %v", err)
	}
	if spacePages.SpaceID != "space" || spacePages.Limit != 100 {
		t.Fatalf("NormalizeSpacePagesRequest() = %#v", spacePages)
	}

	children, err := NormalizePageChildrenRequest(ConfluencePageChildrenRequest{MissionID: "mis_1", CloudID: "cloud_1", PageID: " page ", Limit: -1})
	if err != nil {
		t.Fatalf("NormalizePageChildrenRequest() error = %v", err)
	}
	if children.PageID != "page" || children.Limit != 10 {
		t.Fatalf("NormalizePageChildrenRequest() = %#v", children)
	}
}

func TestNormalizeCandidateAndPagePreserveIdentityAndVersionSemantics(t *testing.T) {
	candidate, err := NormalizeCandidate(ConfluenceSourceCandidate{
		Connector: sourcecontract.ConnectorRef{ExternalSourceID: " cloud_1:123 "},
		CloudID:   "ignored",
		SiteURL:   " https://cloud.example ",
		Title:     " title ",
		SourceURI: "https://cloud.example/wiki/123",
		Summary:   "discarded",
	}, "cloud_1")
	if err != nil {
		t.Fatalf("NormalizeCandidate() error = %v", err)
	}
	if candidate.CloudID != "cloud_1" || candidate.SiteURL != "https://cloud.example" || candidate.Title != "title" ||
		candidate.SourceURI != "https://cloud.example/wiki/123" || candidate.Summary != "" || !candidate.CanSnapshot {
		t.Fatalf("NormalizeCandidate() = %#v", candidate)
	}
	if candidate.Connector.ExternalSourceID != "cloud_1:123" || candidate.Connector.ExternalURI != "" {
		t.Fatalf("candidate connector = %#v", candidate.Connector)
	}

	page, err := NormalizePage(ConfluenceSourcePage{
		CloudID:     " ",
		PageID:      " ",
		SiteURL:     " https://cloud.example ",
		WebURL:      " https://cloud.example/wiki/123 ",
		Version:     7,
		BodyStorage: " body ",
		PlainText:   " text ",
		Connector:   sourcecontract.ConnectorRef{},
		Metadata:    nil,
	}, "cloud_1", "123", 7)
	if err != nil {
		t.Fatalf("NormalizePage() error = %v", err)
	}
	if page.CloudID != "cloud_1" || page.PageID != "123" || page.Title != "123" || page.Version != 7 ||
		page.BodyStorage != "body" || page.PlainText != "text" || string(page.Metadata) != `{}` {
		t.Fatalf("NormalizePage() = %#v", page)
	}
	if page.Connector.ExternalSourceID != "cloud_1:123" || page.Connector.ExternalURI != "confluence://cloud/cloud_1/pages/123" {
		t.Fatalf("page connector = %#v", page.Connector)
	}

	for _, test := range []struct {
		name string
		page ConfluenceSourcePage
		want string
	}{
		{
			name: "cloud mismatch",
			page: ConfluenceSourcePage{CloudID: "other", PageID: "123", Version: 7, BodyStorage: "body", PlainText: "text"},
			want: ConfluenceErrorCodeCloudMismatch,
		},
		{
			name: "page mismatch",
			page: ConfluenceSourcePage{CloudID: "cloud_1", PageID: "other", Version: 7, BodyStorage: "body", PlainText: "text"},
			want: ConfluenceErrorCodePageMismatch,
		},
		{
			name: "version drift",
			page: ConfluenceSourcePage{CloudID: "cloud_1", PageID: "123", Version: 8, BodyStorage: "body", PlainText: "text"},
			want: ConfluenceErrorCodeVersionDrift,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := NormalizePage(test.page, "cloud_1", "123", 7)
			if err == nil {
				t.Fatal("NormalizePage() error = nil")
			}
			var confluenceErr *ConfluenceError
			if !errors.As(err, &confluenceErr) || confluenceErr == nil || confluenceErr.Code != test.want {
				t.Fatalf("NormalizePage() error = %v, want structured code %q", err, test.want)
			}
		})
	}
}

func TestSafeWebURLRejectsCredentialsNonHTTPAndOtherHosts(t *testing.T) {
	if got := SafeWebURL(" https://cloud.example/wiki/123 ", "https://cloud.example"); got != "https://cloud.example/wiki/123" {
		t.Fatalf("safe URL = %q", got)
	}
	for _, test := range []struct {
		name string
		raw  string
		site string
	}{
		{name: "raw userinfo", raw: "https://user:secret@cloud.example/wiki/123", site: "https://cloud.example"},
		{name: "site userinfo", raw: "https://cloud.example/wiki/123", site: "https://user:secret@cloud.example"},
		{name: "ftp scheme", raw: "ftp://cloud.example/wiki/123", site: "https://cloud.example"},
		{name: "javascript scheme", raw: "javascript:alert(1)", site: "https://cloud.example"},
		{name: "other host", raw: "https://evil.example/wiki/123", site: "https://cloud.example"},
		{name: "non-http site", raw: "https://cloud.example/wiki/123", site: "ftp://cloud.example"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := SafeWebURL(test.raw, test.site); got != "" {
				t.Fatalf("SafeWebURL(%q, %q) = %q, want empty", test.raw, test.site, got)
			}
		})
	}
}

func TestConfluenceExternalIdentityStringsTrimWithoutEscaping(t *testing.T) {
	for _, test := range []struct {
		cloudID  string
		pageID   string
		external string
		uri      string
	}{
		{cloudID: " cloud_1 ", pageID: " 123 ", external: "cloud_1:123", uri: "confluence://cloud/cloud_1/pages/123"},
		{cloudID: "site/path", pageID: "page 1", external: "site/path:page 1", uri: "confluence://cloud/site/path/pages/page 1"},
		{cloudID: "", pageID: "123", external: "", uri: ""},
	} {
		if got := ConfluenceExternalSourceID(test.cloudID, test.pageID); got != test.external {
			t.Fatalf("ConfluenceExternalSourceID(%q, %q) = %q, want %q", test.cloudID, test.pageID, got, test.external)
		}
		if got := ConfluenceExternalURI(test.cloudID, test.pageID); got != test.uri {
			t.Fatalf("ConfluenceExternalURI(%q, %q) = %q, want %q", test.cloudID, test.pageID, got, test.uri)
		}
	}

	if _, err := NormalizeSearchRequest(ConfluenceSourceSearchRequest{MissionID: "bad", CloudID: "cloud_1"}); !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("invalid mission id error = %v, want producterror.ErrInvalidInput", err)
	}
}

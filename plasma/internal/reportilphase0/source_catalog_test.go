package reportilphase0

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

func TestBuildSourceCatalogIsDeterministicContentFreeAndOpaqueToProvider(t *testing.T) {
	htmlContent := []byte(`<!doctype html><html><head><script>hidden script</script></head><body><h1 data-path="/private/source">Visible title</h1><p>Repeated fact.</p><p>Repeated fact.</p><a href="https://example.com/locator">Visible label</a></body></html>`)
	textContent := []byte("plain accepted source")
	htmlHash := sha256Hex(htmlContent)
	textHash := sha256Hex(textContent)
	reader := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{
			{SnapshotID: "src_z", MissionID: "mis_catalog", Active: true, ArtifactIDs: []string{"art_z"}, ContentHash: textHash},
			{SnapshotID: "src_a", MissionID: "mis_catalog", Active: true, ArtifactIDs: []string{"art_a"}, ContentHash: htmlHash},
		},
		artifacts: map[string]reportilcontract.Artifact{
			"art_a": {ArtifactID: "art_a", MissionID: "mis_catalog", MediaType: "text/html; charset=utf-8", SHA256: htmlHash, ByteSize: int64(len(htmlContent)), Content: htmlContent},
			"art_z": {ArtifactID: "art_z", MissionID: "mis_catalog", MediaType: "text/plain", SHA256: textHash, ByteSize: int64(len(textContent)), Content: textContent},
		},
	}

	first, err := BuildSourceCatalog(context.Background(), reader, " mis_catalog ")
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildSourceCatalog(context.Background(), reader, "mis_catalog")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || first.SHA256 == "" {
		t.Fatalf("source catalog is not deterministic: %#v %#v", first, second)
	}
	if err := reportilcontract.ValidateSourceCatalog(first); err != nil {
		t.Fatal(err)
	}
	if len(first.Sources) != 2 || first.Sources[0].SourceKey != "source_001" || first.Sources[0].SnapshotID != "src_a" || first.Sources[1].SourceKey != "source_002" || first.Sources[1].SnapshotID != "src_z" {
		t.Fatalf("source catalog ordering = %#v", first.Sources)
	}
	if first.Sources[0].Extraction != "html_visible_text" || first.Sources[0].ReadableBytes == len(htmlContent) || first.Sources[0].ReadableSHA256 == htmlHash {
		t.Fatalf("HTML readable receipt was not canonicalized: %#v", first.Sources[0])
	}
	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"Visible title", "Repeated fact", "hidden script", "/private/source", "example.com/locator", "locator", "filename", "url", "content\""} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("source catalog leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestBuildSourceCatalogIncludesImageSourceAsMetadataOnly(t *testing.T) {
	content := []byte("pinned-image-bytes")
	hash := sha256Hex(content)
	textContent := []byte("reader evidence")
	textHash := sha256Hex(textContent)
	reader := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{{
			SnapshotID: "src_image", MissionID: "mis_image_catalog", Active: true,
			ConnectorType: "media_url", ExternalURI: "https://example.com/image.jpg",
			ArtifactIDs: []string{"art_image"}, ContentHash: hash,
		}, {
			SnapshotID: "src_text", MissionID: "mis_image_catalog", Active: true,
			ConnectorType: "url", ExternalURI: "https://example.com/article",
			ArtifactIDs: []string{"art_text"}, ContentHash: textHash,
		}},
		artifacts: map[string]reportilcontract.Artifact{
			"art_image": {ArtifactID: "art_image", MissionID: "mis_image_catalog", MediaType: "image/jpeg", SHA256: hash, ByteSize: int64(len(content)), Content: content},
			"art_text":  {ArtifactID: "art_text", MissionID: "mis_image_catalog", MediaType: "text/plain", SHA256: textHash, ByteSize: int64(len(textContent)), Content: textContent},
		},
	}
	build, err := BuildSourceCatalogForSelection(context.Background(), reader, "mis_image_catalog")
	if err != nil {
		t.Fatal(err)
	}
	if len(build.Catalog.Sources) != 1 || build.Catalog.Sources[0].Extraction == "binary_metadata" || build.Catalog.Sources[0].Artifacts[0].MediaType != "text/plain" {
		t.Fatalf("text source catalog = %#v", build.Catalog.Sources)
	}
	if len(build.ImageCatalog.Sources) != 1 || build.ImageCatalog.Sources[0].Extraction != "binary_metadata" || build.ImageCatalog.Sources[0].Artifacts[0].MediaType != "image/jpeg" {
		t.Fatalf("image source catalog = %#v", build.ImageCatalog.Sources)
	}
	if _, ok := build.ReadableByAcceptedOrdinal[1]; ok || build.ReadableByAcceptedOrdinal[2] != string(textContent) {
		t.Fatalf("reader evidence cache = %#v", build.ReadableByAcceptedOrdinal)
	}
	if len(build.Dispositions) != 2 || build.Dispositions[0].Status != "image_only" || build.Dispositions[0].Reason != "illustration_only" || build.Dispositions[1].Status != "available" {
		t.Fatalf("image source dispositions = %#v", build.Dispositions)
	}
}

func TestBuildSourceCatalogImageCatalogIncludesUnselectedHTMLSources(t *testing.T) {
	firstHTML := []byte(`<html><body><p>` + strings.Repeat("reader evidence ", 5000) + `</p><img src="/first.jpg"></body></html>`)
	secondHTML := []byte(`<html><body><p>` + strings.Repeat("supporting evidence ", 5000) + `</p><img src="/second.jpg"></body></html>`)
	firstHash, secondHash := sha256Hex(firstHTML), sha256Hex(secondHTML)
	reader := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{{
			SnapshotID: "src_first", MissionID: "mis_html_images", Active: true,
			ConnectorType: "url", ExternalURI: "https://first.example.com/page",
			ArtifactIDs: []string{"art_first"}, ContentHash: firstHash,
		}, {
			SnapshotID: "src_second", MissionID: "mis_html_images", Active: true,
			ConnectorType: "url", ExternalURI: "https://second.example.com/page",
			ArtifactIDs: []string{"art_second"}, ContentHash: secondHash,
		}},
		artifacts: map[string]reportilcontract.Artifact{
			"art_first":  {ArtifactID: "art_first", MissionID: "mis_html_images", MediaType: "text/html", SHA256: firstHash, ByteSize: int64(len(firstHTML)), Content: firstHTML},
			"art_second": {ArtifactID: "art_second", MissionID: "mis_html_images", MediaType: "text/html", SHA256: secondHash, ByteSize: int64(len(secondHTML)), Content: secondHTML},
		},
	}
	build, err := BuildSourceCatalogForSelection(context.Background(), reader, "mis_html_images")
	if err != nil {
		t.Fatal(err)
	}
	selected, _, err := compileSelectedSourceCatalog(build.Catalog, []string{"source_001", "source_002"})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Sources) != 1 {
		t.Fatalf("expected one compact author source, got %#v", selected.Sources)
	}
	if len(build.ImageCatalog.Sources) != 2 || build.ImageCatalog.Sources[0].AcceptedOrdinal != 1 || build.ImageCatalog.Sources[1].AcceptedOrdinal != 2 {
		t.Fatalf("full HTML image catalog = %#v", build.ImageCatalog.Sources)
	}
}

func TestBuildSourceCatalogRejectsImageOnlyMissionWithoutReadableEvidence(t *testing.T) {
	content := []byte("pinned-image-bytes")
	hash := sha256Hex(content)
	reader := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{{
			SnapshotID: "src_image", MissionID: "mis_image_only", Active: true,
			ConnectorType: "media_url", ExternalURI: "https://example.com/image.jpg",
			ArtifactIDs: []string{"art_image"}, ContentHash: hash,
		}},
		artifacts: map[string]reportilcontract.Artifact{
			"art_image": {ArtifactID: "art_image", MissionID: "mis_image_only", MediaType: "image/jpeg", SHA256: hash, ByteSize: int64(len(content)), Content: content},
		},
	}
	if _, err := BuildSourceCatalogForSelection(context.Background(), reader, "mis_image_only"); err == nil || !strings.Contains(err.Error(), "no usable accepted text sources") {
		t.Fatalf("image-only catalog error = %v", err)
	}
}

func TestBuildSourceCatalogKeepsReadableTextServerOnlyByAcceptedOrdinal(t *testing.T) {
	content := []byte("法樹寺（ほうじゅじ）は山陰道に近い。")
	hash := sha256Hex(content)
	reader := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{{
			SnapshotID: "src_term", MissionID: "mis_term_cache", Active: true,
			ArtifactIDs: []string{"art_term"}, ContentHash: hash,
		}},
		artifacts: map[string]reportilcontract.Artifact{
			"art_term": {ArtifactID: "art_term", MissionID: "mis_term_cache", MediaType: "text/plain", SHA256: hash, ByteSize: int64(len(content)), Content: content},
		},
	}
	build, err := BuildSourceCatalogForSelection(context.Background(), reader, "mis_term_cache")
	if err != nil {
		t.Fatal(err)
	}
	ordinal := build.Catalog.Sources[0].AcceptedOrdinal
	if build.ReadableByAcceptedOrdinal[ordinal] != string(content) {
		t.Fatalf("server-only readable cache = %#v", build.ReadableByAcceptedOrdinal)
	}
	encoded, err := json.Marshal(build)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), string(content)) || strings.Contains(string(encoded), "ほうじゅじ") {
		t.Fatalf("source text cache was serialized: %s", encoded)
	}
}

func TestBuildSourceCatalogKeepsReaderCitationMetadataOutsideProviderCatalog(t *testing.T) {
	content := []byte("public source")
	hash := sha256Hex(content)
	reader := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{{
			SnapshotID: "src_public", MissionID: "mis_citation", Active: true,
			Title: "Public source title", ConnectorType: "url", ExternalURI: "https://Example.com/report#section",
			ArtifactIDs: []string{"art_public"}, ContentHash: hash,
		}},
		artifacts: map[string]reportilcontract.Artifact{
			"art_public": {ArtifactID: "art_public", MissionID: "mis_citation", MediaType: "text/plain", SHA256: hash, ByteSize: int64(len(content)), Content: content},
		},
	}
	build, err := BuildSourceCatalogForSelection(context.Background(), reader, "mis_citation")
	if err != nil {
		t.Fatal(err)
	}
	citation := build.CitationByKey["source_001"]
	if citation.VisibleLabel != "Public source title" || citation.URL != "https://example.com/report" {
		t.Fatalf("citation metadata = %#v", citation)
	}
	encoded, err := json.Marshal(build.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"Public source title", "example.com", "external_uri", "citation_metadata"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("provider catalog leaked citation metadata %q: %s", forbidden, encoded)
		}
	}
}

func TestSourceCitationReplacesHostnameTitleWithReadableDocumentTitle(t *testing.T) {
	snapshot := reportilcontract.SourceSnapshot{
		Title: "www.city.asago.hyogo.jp", ConnectorType: "url",
		ExternalURI: "https://www.city.asago.hyogo.jp/site/takeda/3094.html",
	}
	citation := sourceCitation(snapshot, sourceCatalogReadable{Text: "生野銀山と竹田城の関係\n本文"})
	if citation.VisibleLabel != "生野銀山と竹田城の関係" || citation.URL != "https://www.city.asago.hyogo.jp/site/takeda/3094.html" {
		t.Fatalf("readable citation = %#v", citation)
	}
}

func TestSourceCitationPrefersHTMLTitleAndSkipsBoilerplate(t *testing.T) {
	snapshot := reportilcontract.SourceSnapshot{
		Title: "www.city.asago.hyogo.jp", ConnectorType: "url",
		ExternalURI: "https://www.city.asago.hyogo.jp/site/takeda/3092.html",
	}
	citation := sourceCitation(snapshot, sourceCatalogReadable{
		Text:       "ページの先頭です。 メニューを飛ばして本文へ\n本文",
		Extraction: "html_visible_text", DocumentTitle: "竹田城跡の歴史",
		PrimaryHeading: "竹田城跡",
	})
	if citation.VisibleLabel != "竹田城跡の歴史" {
		t.Fatalf("HTML title citation = %#v", citation)
	}
}

func TestSourceCitationSkipsHTMLSearchControlsForPublicPath(t *testing.T) {
	snapshot := reportilcontract.SourceSnapshot{
		Title: "api.online.bunka.go.jp", ConnectorType: "url",
		ExternalURI: "https://api.online.bunka.go.jp/heritages/detail/161758",
	}
	citation := sourceCitation(snapshot, sourceCatalogReadable{
		Text: "検索\n本文", Extraction: "html_visible_text",
		DocumentTitle: "検索", PrimaryHeading: "検索",
	})
	if citation.VisibleLabel != "heritages › detail › 161758" {
		t.Fatalf("boilerplate citation = %#v", citation)
	}
}

func TestSourceCitationRejectsRestrictedHTMLMetadata(t *testing.T) {
	snapshot := reportilcontract.SourceSnapshot{
		Title: "example.com", ConnectorType: "url",
		ExternalURI: "https://example.com/history/report",
	}
	citation := sourceCitation(snapshot, sourceCatalogReadable{
		Text: "본문", Extraction: "html_visible_text",
		DocumentTitle:  "Authorization: secret-value",
		PrimaryHeading: "See http://10.0.0.1/private",
	})
	if citation.VisibleLabel != "history › report" {
		t.Fatalf("restricted HTML metadata citation = %#v", citation)
	}
}

func TestSourceCitationFallsBackToDistinguishablePublicPath(t *testing.T) {
	snapshot := reportilcontract.SourceSnapshot{
		ConnectorType: "url", ExternalURI: "https://example.com/history/takeda-castle.html",
	}
	citation := sourceCitation(snapshot, sourceCatalogReadable{})
	if citation.VisibleLabel != "history › takeda castle.html" {
		t.Fatalf("path citation = %#v", citation)
	}
}

func TestPublicCitationURLRejectsPrivateAndCredentialBearingLocators(t *testing.T) {
	for _, raw := range []string{
		"file:///private/source", "https://user:secret@example.com/a", "http://localhost/a",
		"https://node.ts.net/a", "https://intranet/report", "https://example.invalid/report",
		"http://127.0.0.1/a", "http://10.0.0.1/a", "http://100.64.0.1/a",
	} {
		if got := publicCitationURL(raw); got != "" {
			t.Fatalf("publicCitationURL(%q) = %q", raw, got)
		}
	}
	if got := publicCitationURL("HTTPS://Example.com/A?q=1#fragment"); got != "https://example.com/A?q=1" {
		t.Fatalf("public citation URL = %q", got)
	}
}

func TestSourceCitationKeepsNonPublicConnectorsGeneric(t *testing.T) {
	for _, snapshot := range []reportilcontract.SourceSnapshot{
		{Title: "Private local file", ConnectorType: "local_path", ExternalURI: "https://example.com/private"},
		{Title: "Uploaded secret.pdf", ConnectorType: "file_upload", ExternalURI: "https://example.com/upload"},
		{Title: "Private Confluence", ConnectorType: "confluence", ExternalURI: "https://example.atlassian.net/wiki/x"},
	} {
		if citation := sourceCitation(snapshot); citation != (SourceCitation{}) {
			t.Fatalf("non-public citation = %#v", citation)
		}
	}
}

func TestBuildSourceCatalogHasNoLegacyAggregatePacketCeiling(t *testing.T) {
	reader := productSourceReader{artifacts: map[string]reportilcontract.Artifact{}}
	for index := 1; index <= 3; index++ {
		content := []byte(strings.Repeat(string(rune('a'+index-1)), 200*1024))
		artifactID := "art_" + string(rune('a'+index-1))
		snapshotID := "src_" + string(rune('a'+index-1))
		hash := sha256Hex(content)
		reader.sources = append(reader.sources, reportilcontract.SourceSnapshot{
			SnapshotID: snapshotID, MissionID: "mis_large_catalog", Active: true,
			ArtifactIDs: []string{artifactID}, ContentHash: hash,
		})
		reader.artifacts[artifactID] = reportilcontract.Artifact{
			ArtifactID: artifactID, MissionID: "mis_large_catalog", MediaType: "text/plain",
			SHA256: hash, ByteSize: int64(len(content)), Content: content,
		}
	}
	catalog, err := BuildSourceCatalog(context.Background(), reader, "mis_large_catalog")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Sources) != 3 {
		t.Fatalf("large catalog sources = %d", len(catalog.Sources))
	}
	if encoded, err := json.Marshal(catalog); err != nil || len(encoded) >= MaxPacketBytes {
		t.Fatalf("content-free catalog unexpectedly scales with source bytes: bytes=%d err=%v", len(encoded), err)
	}
}

func TestBuildSourceCatalogRejectsChangedUnsupportedAndCrossMissionSources(t *testing.T) {
	content := []byte("accepted")
	hash := sha256Hex(content)
	base := productSourceReader{
		sources:   []reportilcontract.SourceSnapshot{{SnapshotID: "src_1", MissionID: "mis_catalog", Active: true, ArtifactIDs: []string{"art_1"}, ContentHash: hash}},
		artifacts: map[string]reportilcontract.Artifact{"art_1": {ArtifactID: "art_1", MissionID: "mis_catalog", MediaType: "text/plain", SHA256: hash, ByteSize: int64(len(content)), Content: content}},
	}
	cases := []struct {
		name   string
		mutate func(*productSourceReader)
	}{
		{name: "cross mission", mutate: func(reader *productSourceReader) { reader.sources[0].MissionID = "mis_other" }},
		{name: "changed artifact", mutate: func(reader *productSourceReader) {
			reader.artifacts["art_1"] = reportilcontract.Artifact{ArtifactID: "art_1", MissionID: "mis_catalog", MediaType: "text/plain", SHA256: hash, ByteSize: 7, Content: []byte("changed")}
		}},
		{name: "unsupported media", mutate: func(reader *productSourceReader) {
			reader.artifacts["art_1"] = reportilcontract.Artifact{ArtifactID: "art_1", MissionID: "mis_catalog", MediaType: "application/octet-stream", SHA256: hash, ByteSize: int64(len(content)), Content: content}
		}},
		{name: "changed snapshot hash", mutate: func(reader *productSourceReader) { reader.sources[0].ContentHash = strings.Repeat("f", 64) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := productSourceReader{
				sources:   append([]reportilcontract.SourceSnapshot(nil), base.sources...),
				artifacts: map[string]reportilcontract.Artifact{"art_1": base.artifacts["art_1"]},
			}
			tc.mutate(&reader)
			if _, err := BuildSourceCatalog(context.Background(), reader, "mis_catalog"); err == nil {
				t.Fatal("invalid frozen source was accepted")
			}
		})
	}
}

func TestBuildSourceCatalogForSelectionExcludesOnlyUnusableTextAndPreservesOrdinal(t *testing.T) {
	usable := []byte("usable source")
	damaged := []byte("damaged � source")
	usableHash := sha256Hex(usable)
	damagedHash := sha256Hex(damaged)
	reader := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{
			{SnapshotID: "src_a", MissionID: "mis_selection_catalog", Active: true, ArtifactIDs: []string{"art_a"}, ContentHash: damagedHash},
			{SnapshotID: "src_b", MissionID: "mis_selection_catalog", Active: true, ArtifactIDs: []string{"art_b"}, ContentHash: usableHash},
		},
		artifacts: map[string]reportilcontract.Artifact{
			"art_a": {ArtifactID: "art_a", MissionID: "mis_selection_catalog", MediaType: "text/plain", SHA256: damagedHash, ByteSize: int64(len(damaged)), Content: damaged},
			"art_b": {ArtifactID: "art_b", MissionID: "mis_selection_catalog", MediaType: "text/plain", SHA256: usableHash, ByteSize: int64(len(usable)), Content: usable},
		},
	}
	build, err := BuildSourceCatalogForSelection(context.Background(), reader, "mis_selection_catalog")
	if err != nil {
		t.Fatal(err)
	}
	if len(build.Catalog.Sources) != 1 || build.Catalog.Sources[0].SourceKey != "source_001" || build.Catalog.Sources[0].AcceptedOrdinal != 2 {
		t.Fatalf("selection catalog = %#v", build.Catalog.Sources)
	}
	if len(build.ReadableByAcceptedOrdinal) != 1 || build.ReadableByAcceptedOrdinal[2] != string(usable) {
		t.Fatalf("readable cache did not preserve the accepted ordinal: %#v", build.ReadableByAcceptedOrdinal)
	}
	if len(build.Dispositions) != 2 || build.Dispositions[0].Status != "excluded" || build.Dispositions[0].Reason != "unusable_readable_text" || build.Dispositions[1].Status != "available" {
		t.Fatalf("selection dispositions = %#v", build.Dispositions)
	}
	if _, err := BuildSourceCatalog(context.Background(), reader, "mis_selection_catalog"); err == nil {
		t.Fatal("strict source catalog accepted an unusable source")
	}
}

func TestBuildSourceCatalogForSelectionRedactsRestrictedSpansWithoutExcludingSource(t *testing.T) {
	content := []byte("HTTP headers\nAuthorization: secret-value\nPublic explanation")
	hash := sha256Hex(content)
	reader := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{{SnapshotID: "src_restricted", MissionID: "mis_restricted", Active: true, ArtifactIDs: []string{"art_restricted"}, ContentHash: hash}},
		artifacts: map[string]reportilcontract.Artifact{
			"art_restricted": {ArtifactID: "art_restricted", MissionID: "mis_restricted", MediaType: "text/plain", SHA256: hash, ByteSize: int64(len(content)), Content: content},
		},
	}
	build, err := BuildSourceCatalogForSelection(context.Background(), reader, "mis_restricted")
	if err != nil {
		t.Fatal(err)
	}
	readable := build.ReadableByAcceptedOrdinal[1]
	if len(build.Catalog.Sources) != 1 || len(build.Dispositions) != 1 || build.Dispositions[0].Status != "available" || !strings.Contains(readable, "[redacted credential]") || strings.Contains(readable, "secret-value") {
		t.Fatalf("redacted catalog = %#v / %#v / %q", build.Catalog.Sources, build.Dispositions, readable)
	}
}

func TestBuildSourceCatalogIgnoresInactiveSourcesButRequiresOneActive(t *testing.T) {
	reader := productSourceReader{sources: []reportilcontract.SourceSnapshot{{SnapshotID: "src_inactive", MissionID: "mis_catalog", Active: false}}}
	if _, err := BuildSourceCatalog(context.Background(), reader, "mis_catalog"); err == nil || !strings.Contains(err.Error(), "no active") {
		t.Fatalf("inactive-only catalog = %v", err)
	}
}

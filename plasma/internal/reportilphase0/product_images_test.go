package reportilphase0

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

func TestProductImageFilenameMatchesPinnedMediaType(t *testing.T) {
	for _, test := range []struct {
		mediaType string
		ordinal   int
		want      string
	}{
		{"image/jpeg", 1, "report-image-1.jpg"},
		{"image/png", 2, "report-image-2.png"},
		{"image/gif", 3, "report-image-3.gif"},
	} {
		if got := productImageFilename(test.mediaType, test.ordinal); got != test.want {
			t.Fatalf("productImageFilename(%q, %d) = %q, want %q", test.mediaType, test.ordinal, got, test.want)
		}
	}
}

func TestResolveProductImageURLAllowsSiblingCDNOnSameRegistrableDomain(t *testing.T) {
	page, err := url.Parse("https://gall.dcinside.com/mgallery/board/view?id=1")
	if err != nil {
		t.Fatal(err)
	}
	if got := resolveProductImageURL(page, "https://dcimg4.dcinside.co.kr/viewimage.php?id=2"); got != "https://dcimg4.dcinside.co.kr/viewimage.php?id=2" {
		t.Fatalf("same-site image URL = %q", got)
	}
	if got := resolveProductImageURL(page, "https://other.example.net/image.jpg"); got != "" {
		t.Fatalf("cross-site image URL = %q", got)
	}
}

func TestCollectProductImageCandidatesUsesOnlySelectedAcceptedSourceLinks(t *testing.T) {
	html := []byte(`<html><body>
<img src="/large.jpg" alt="성곽 경관">
<img src="https://other.example.net/ambient.jpg" alt="다른 호스트">
<img src="data:image/png;base64,AAAA" alt="인라인">
</body></html>`)
	hash := SHA256(html)
	reader := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{{
			SnapshotID: "src_selected", MissionID: "mis_images", Title: "공식 성곽 페이지",
			ArtifactIDs: []string{"art_html"}, ContentHash: hash, ConnectorType: "url",
			ExternalURI: "https://example.com/castle/page.html", Active: true,
		}, {
			SnapshotID: "src_unselected", MissionID: "mis_images", Title: "선택되지 않은 페이지",
			ArtifactIDs: []string{"art_other"}, ContentHash: SHA256([]byte("other")), ConnectorType: "url",
			ExternalURI: "https://example.com/other.html", Active: true,
		}},
		artifacts: map[string]reportilcontract.Artifact{
			"art_html":  {ArtifactID: "art_html", MissionID: "mis_images", MediaType: "text/html", ByteSize: int64(len(html)), SHA256: hash, Content: html},
			"art_other": {ArtifactID: "art_other", MissionID: "mis_images", MediaType: "text/plain", ByteSize: 5, SHA256: SHA256([]byte("other")), Content: []byte("other")},
		},
	}
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{MissionID: "mis_images", Sources: []reportilcontract.SourceCatalogEntry{{
		SourceKey: "source_001", AcceptedOrdinal: 1, SnapshotID: "src_selected", SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_selected", hash),
		ContentHash: hash, RetrievalPolicy: "snapshot_only", Artifacts: []reportilcontract.SourceCatalogArtifact{{ArtifactID: "art_html", SHA256: hash, ByteSize: int64(len(html)), MediaType: "text/html"}},
		ReadableSHA256: strings.Repeat("a", 64), ReadableBytes: 10, Extraction: "html_visible_text",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	var fetched []string
	candidates, err := collectProductImageCandidates(context.Background(), reader, catalog, func(_ context.Context, rawURL string) (ProductImage, error) {
		fetched = append(fetched, rawURL)
		return ProductImage{MediaType: "image/jpeg", Content: []byte("jpeg"), Width: 1200, Height: 800}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fetched) != 1 || fetched[0] != "https://example.com/large.jpg" || len(candidates) != 1 || candidates[0].SnapshotID != "src_selected" || candidates[0].SourceAlt != "성곽 경관" {
		t.Fatalf("accepted-source image candidates = fetched %#v candidates %#v", fetched, candidates)
	}
}

func TestCollectProductImageCandidatesSkipsBrokenLinksWithinAttemptCap(t *testing.T) {
	links := make([]string, 0, maxProductImageCandidates+2)
	for index := 0; index < maxProductImageCandidates+2; index++ {
		links = append(links, `<img src="/image-`+fmt.Sprint(index)+`.jpg" alt="candidate">`)
	}
	html := []byte(strings.Join(links, "\n"))
	hash := SHA256(html)
	reader := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{{
			SnapshotID: "src_selected", MissionID: "mis_images", Title: "공식 성곽 페이지",
			ArtifactIDs: []string{"art_html"}, ContentHash: hash, ConnectorType: "url",
			ExternalURI: "https://example.com/castle/page.html", Active: true,
		}},
		artifacts: map[string]reportilcontract.Artifact{
			"art_html": {ArtifactID: "art_html", MissionID: "mis_images", MediaType: "text/html", ByteSize: int64(len(html)), SHA256: hash, Content: html},
		},
	}
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{MissionID: "mis_images", Sources: []reportilcontract.SourceCatalogEntry{{
		SourceKey: "source_001", AcceptedOrdinal: 1, SnapshotID: "src_selected", SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_selected", hash),
		ContentHash: hash, RetrievalPolicy: "snapshot_only", Artifacts: []reportilcontract.SourceCatalogArtifact{{ArtifactID: "art_html", SHA256: hash, ByteSize: int64(len(html)), MediaType: "text/html"}},
		ReadableSHA256: strings.Repeat("a", 64), ReadableBytes: 10, Extraction: "html_visible_text",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	attempts := 0
	candidates, err := collectProductImageCandidates(context.Background(), reader, catalog, func(_ context.Context, _ string) (ProductImage, error) {
		attempts++
		return ProductImage{}, fmt.Errorf("unavailable")
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 || attempts != maxProductImageCandidates {
		t.Fatalf("broken optional images = %d candidates from %d attempts", len(candidates), attempts)
	}
}

func TestCollectProductImageCandidatesCapsDirectSourceArtifacts(t *testing.T) {
	reader := productSourceReader{artifacts: map[string]reportilcontract.Artifact{}}
	catalog := reportilcontract.SourceCatalog{MissionID: "mis_images"}
	for index := 0; index < maxProductImageCandidates+2; index++ {
		snapshotID := fmt.Sprintf("src_%02d", index)
		artifactID := fmt.Sprintf("art_%02d", index)
		content := []byte(fmt.Sprintf("image-%02d", index))
		hash := SHA256(content)
		reader.sources = append(reader.sources, reportilcontract.SourceSnapshot{
			SnapshotID: snapshotID, MissionID: "mis_images", Title: "직접 이미지",
			ArtifactIDs: []string{artifactID}, ContentHash: hash, ConnectorType: "media_url",
			ExternalURI: fmt.Sprintf("https://example.com/image-%02d.jpg", index), Active: true,
		})
		reader.artifacts[artifactID] = reportilcontract.Artifact{
			ArtifactID: artifactID, MissionID: "mis_images", MediaType: "image/jpeg",
			ByteSize: int64(len(content)), SHA256: hash, Content: content,
		}
		catalog.Sources = append(catalog.Sources, reportilcontract.SourceCatalogEntry{
			SourceKey: fmt.Sprintf("source_%03d", index+1), AcceptedOrdinal: index + 1,
			SnapshotID: snapshotID, SnapshotReceipt: reportilcontract.SourceSnapshotReceipt(snapshotID, hash),
			ContentHash: hash, RetrievalPolicy: "snapshot_only",
			Artifacts:      []reportilcontract.SourceCatalogArtifact{{ArtifactID: artifactID, SHA256: hash, ByteSize: int64(len(content)), MediaType: "image/jpeg"}},
			ReadableSHA256: strings.Repeat("a", 64), ReadableBytes: 1, Extraction: "binary_metadata",
		})
	}
	sealed, err := reportilcontract.SealSourceCatalog(catalog)
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := collectProductImageCandidates(context.Background(), reader, sealed, func(context.Context, string) (ProductImage, error) {
		t.Fatal("direct source artifacts must not be fetched again")
		return ProductImage{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != maxProductImageCandidates {
		t.Fatalf("direct source candidates = %d, want %d", len(candidates), maxProductImageCandidates)
	}
}

func TestCollectProductImageCandidatesGivesEverySelectedSourceAnEarlyAttempt(t *testing.T) {
	firstLinks := make([]string, 0, maxProductImageCandidates+2)
	for index := 0; index < maxProductImageCandidates+2; index++ {
		firstLinks = append(firstLinks, `<img src="/broken-`+fmt.Sprint(index)+`.jpg" alt="broken">`)
	}
	firstHTML := []byte(strings.Join(firstLinks, "\n"))
	secondHTML := []byte(`<img src="/useful.jpg" alt="성곽 배치도">`)
	firstHash, secondHash := SHA256(firstHTML), SHA256(secondHTML)
	reader := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{{
			SnapshotID: "src_first", MissionID: "mis_images", Title: "앞선 페이지",
			ArtifactIDs: []string{"art_first"}, ContentHash: firstHash, ConnectorType: "url",
			ExternalURI: "https://first.example.com/page.html", Active: true,
		}, {
			SnapshotID: "src_second", MissionID: "mis_images", Title: "공식 배치 페이지",
			ArtifactIDs: []string{"art_second"}, ContentHash: secondHash, ConnectorType: "url",
			ExternalURI: "https://second.example.com/page.html", Active: true,
		}},
		artifacts: map[string]reportilcontract.Artifact{
			"art_first":  {ArtifactID: "art_first", MissionID: "mis_images", MediaType: "text/html", ByteSize: int64(len(firstHTML)), SHA256: firstHash, Content: firstHTML},
			"art_second": {ArtifactID: "art_second", MissionID: "mis_images", MediaType: "text/html", ByteSize: int64(len(secondHTML)), SHA256: secondHash, Content: secondHTML},
		},
	}
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{MissionID: "mis_images", Sources: []reportilcontract.SourceCatalogEntry{{
		SourceKey: "source_001", AcceptedOrdinal: 1, SnapshotID: "src_first", SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_first", firstHash),
		ContentHash: firstHash, RetrievalPolicy: "snapshot_only", Artifacts: []reportilcontract.SourceCatalogArtifact{{ArtifactID: "art_first", SHA256: firstHash, ByteSize: int64(len(firstHTML)), MediaType: "text/html"}},
		ReadableSHA256: strings.Repeat("a", 64), ReadableBytes: 10, Extraction: "html_visible_text",
	}, {
		SourceKey: "source_002", AcceptedOrdinal: 2, SnapshotID: "src_second", SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_second", secondHash),
		ContentHash: secondHash, RetrievalPolicy: "snapshot_only", Artifacts: []reportilcontract.SourceCatalogArtifact{{ArtifactID: "art_second", SHA256: secondHash, ByteSize: int64(len(secondHTML)), MediaType: "text/html"}},
		ReadableSHA256: strings.Repeat("b", 64), ReadableBytes: 10, Extraction: "html_visible_text",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	var fetched []string
	candidates, err := collectProductImageCandidates(context.Background(), reader, catalog, func(_ context.Context, rawURL string) (ProductImage, error) {
		fetched = append(fetched, rawURL)
		if rawURL == "https://second.example.com/useful.jpg" {
			return ProductImage{MediaType: "image/jpeg", Content: []byte("jpeg"), Width: 1200, Height: 800}, nil
		}
		return ProductImage{}, fmt.Errorf("unavailable")
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fetched) != maxProductImageCandidates || len(fetched) < 2 || fetched[1] != "https://second.example.com/useful.jpg" {
		t.Fatalf("round-robin fetch order = %#v", fetched)
	}
	if len(candidates) != 1 || candidates[0].SnapshotID != "src_second" || candidates[0].SourceAlt != "성곽 배치도" {
		t.Fatalf("later selected source was starved: %#v", candidates)
	}
}

func TestCollectProductImageCandidatesContinuesAfterDuplicateOnlyRound(t *testing.T) {
	firstHTML := []byte(`<img src="/shared.jpg"><img src="/other.jpg"><img src="/later.jpg" alt="후속 경관">`)
	secondHTML := []byte(`<img src="/other.jpg"><img src="/shared.jpg">`)
	firstHash, secondHash := SHA256(firstHTML), SHA256(secondHTML)
	reader := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{{
			SnapshotID: "src_first", MissionID: "mis_images", Title: "첫 페이지",
			ArtifactIDs: []string{"art_first"}, ContentHash: firstHash, ConnectorType: "url",
			ExternalURI: "https://example.com/first.html", Active: true,
		}, {
			SnapshotID: "src_second", MissionID: "mis_images", Title: "둘째 페이지",
			ArtifactIDs: []string{"art_second"}, ContentHash: secondHash, ConnectorType: "url",
			ExternalURI: "https://example.com/second.html", Active: true,
		}},
		artifacts: map[string]reportilcontract.Artifact{
			"art_first":  {ArtifactID: "art_first", MissionID: "mis_images", MediaType: "text/html", ByteSize: int64(len(firstHTML)), SHA256: firstHash, Content: firstHTML},
			"art_second": {ArtifactID: "art_second", MissionID: "mis_images", MediaType: "text/html", ByteSize: int64(len(secondHTML)), SHA256: secondHash, Content: secondHTML},
		},
	}
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{MissionID: "mis_images", Sources: []reportilcontract.SourceCatalogEntry{{
		SourceKey: "source_001", AcceptedOrdinal: 1, SnapshotID: "src_first", SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_first", firstHash),
		ContentHash: firstHash, RetrievalPolicy: "snapshot_only", Artifacts: []reportilcontract.SourceCatalogArtifact{{ArtifactID: "art_first", SHA256: firstHash, ByteSize: int64(len(firstHTML)), MediaType: "text/html"}},
		ReadableSHA256: strings.Repeat("a", 64), ReadableBytes: 10, Extraction: "html_visible_text",
	}, {
		SourceKey: "source_002", AcceptedOrdinal: 2, SnapshotID: "src_second", SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_second", secondHash),
		ContentHash: secondHash, RetrievalPolicy: "snapshot_only", Artifacts: []reportilcontract.SourceCatalogArtifact{{ArtifactID: "art_second", SHA256: secondHash, ByteSize: int64(len(secondHTML)), MediaType: "text/html"}},
		ReadableSHA256: strings.Repeat("b", 64), ReadableBytes: 10, Extraction: "html_visible_text",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	var fetched []string
	candidates, err := collectProductImageCandidates(context.Background(), reader, catalog, func(_ context.Context, rawURL string) (ProductImage, error) {
		fetched = append(fetched, rawURL)
		if rawURL == "https://example.com/later.jpg" {
			return ProductImage{MediaType: "image/jpeg", Content: []byte("jpeg"), Width: 1200, Height: 800}, nil
		}
		return ProductImage{}, fmt.Errorf("unavailable")
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fetched) != 3 || fetched[2] != "https://example.com/later.jpg" || len(candidates) != 1 {
		t.Fatalf("duplicate-only round stopped later unique links: fetched %#v candidates %#v", fetched, candidates)
	}
}

func TestCollectProductImageCandidatesIgnoresArtifactsOutsideFrozenCatalog(t *testing.T) {
	html := []byte(`<html><body><img src="/large.jpg" alt="성곽 경관"></body></html>`)
	hash := SHA256(html)
	unfrozen := []byte(`<html><body><img src="/ambient.jpg" alt="추가 경관"></body></html>`)
	reader := productSourceReader{
		sources: []reportilcontract.SourceSnapshot{{
			SnapshotID: "src_selected", MissionID: "mis_images", Title: "공식 성곽 페이지",
			ArtifactIDs: []string{"art_html", "art_unfrozen"}, ContentHash: hash, ConnectorType: "url",
			ExternalURI: "https://example.com/castle/page.html", Active: true,
		}},
		artifacts: map[string]reportilcontract.Artifact{
			"art_html":     {ArtifactID: "art_html", MissionID: "mis_images", MediaType: "text/html", ByteSize: int64(len(html)), SHA256: hash, Content: html},
			"art_unfrozen": {ArtifactID: "art_unfrozen", MissionID: "mis_images", MediaType: "text/html", ByteSize: int64(len(unfrozen)), SHA256: SHA256(unfrozen), Content: unfrozen},
		},
	}
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{MissionID: "mis_images", Sources: []reportilcontract.SourceCatalogEntry{{
		SourceKey: "source_001", AcceptedOrdinal: 1, SnapshotID: "src_selected", SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_selected", hash),
		ContentHash: hash, RetrievalPolicy: "snapshot_only", Artifacts: []reportilcontract.SourceCatalogArtifact{{ArtifactID: "art_html", SHA256: hash, ByteSize: int64(len(html)), MediaType: "text/html"}},
		ReadableSHA256: strings.Repeat("a", 64), ReadableBytes: 10, Extraction: "html_visible_text",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	var fetched []string
	candidates, err := collectProductImageCandidates(context.Background(), reader, catalog, func(_ context.Context, rawURL string) (ProductImage, error) {
		fetched = append(fetched, rawURL)
		return ProductImage{MediaType: "image/jpeg", Content: []byte("jpeg"), Width: 1200, Height: 800}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || len(fetched) != 1 || fetched[0] != "https://example.com/large.jpg" {
		t.Fatalf("non-frozen snapshot artifact affected candidates: fetched %#v candidates %#v", fetched, candidates)
	}
}

func TestJoinProductImagesPreservesEveryFinalizedBlockAndMakesCompactMarkdown(t *testing.T) {
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily, DocumentID: "doc_images", RevisionID: "draft",
		NarrativeContractID: "narr_images", Title: "성곽 보고서", Language: "ko",
		Blocks: []Block{
			{NodeID: "section.opening", Kind: "section", Level: 2, Title: "핵심"},
			{NodeID: "prose.opening", Kind: "prose", ParentNodeID: "section.opening", Prose: "이 문장은 바뀌면 안 됩니다."},
		},
		Provenance: map[string]string{"source": "accepted"},
	}
	imageSourceHash := strings.Repeat("c", 64)
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{MissionID: "mis_images", Sources: []reportilcontract.SourceCatalogEntry{{
		SourceKey: "source_001", AcceptedOrdinal: 1, SnapshotID: "src_image", SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_image", imageSourceHash),
		ContentHash: imageSourceHash, RetrievalPolicy: "snapshot_only", Artifacts: []reportilcontract.SourceCatalogArtifact{{ArtifactID: "art_source", SHA256: imageSourceHash, ByteSize: 10, MediaType: "text/plain"}},
		ReadableSHA256: strings.Repeat("a", 64), ReadableBytes: 10, Extraction: "plain_text",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	candidates := []productImageCandidate{{
		CandidateID: "image_01", SnapshotID: "src_image", SourcePageURL: "https://example.com/castle",
		ImageURL: "https://example.com/castle.jpg", SourceArtifactID: "art_source", MediaType: "image/jpeg", Content: []byte(strings.Repeat("x", 2048)),
	}}
	count := 0
	joined, artifacts, _, err := joinProductImages(document, catalog, candidates, []imagePlacement{{
		CandidateID: "image_01", AfterNodeID: "prose.opening", Caption: "산지에 놓인 성곽", Alt: "산 정상의 성곽 경관",
	}}, func(prefix string) string { count++; return prefix + "_" + string(rune('0'+count)) })
	if err != nil {
		t.Fatal(err)
	}
	if len(joined.Blocks) != 3 || joined.Blocks[0].NodeID != document.Blocks[0].NodeID || joined.Blocks[0].Title != document.Blocks[0].Title || joined.Blocks[1].NodeID != document.Blocks[1].NodeID || joined.Blocks[1].Prose != document.Blocks[1].Prose || len(artifacts) != 1 || len(joined.Assets) != 1 {
		t.Fatalf("joined document changed finalized blocks: %#v", joined.Blocks)
	}
	if artifacts[0].ID == "art_source" || joined.Assets[0].ArtifactID == "art_source" || artifacts[0].ID != joined.Assets[0].ArtifactID {
		t.Fatalf("report image reused or lost its bundle-local artifact identity: artifact=%q asset=%q", artifacts[0].ID, joined.Assets[0].ArtifactID)
	}
	markdown, _, err := RenderMarkdown(joined)
	if err != nil {
		t.Fatal(err)
	}
	if len(markdown) > 1024 || !strings.Contains(string(markdown), "](artifact:") || strings.Contains(string(markdown), "data:image") {
		t.Fatalf("Markdown did not use compact image reference: bytes=%d content=%s", len(markdown), markdown)
	}
	html, _, err := RenderHTML(joined)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "data:image/jpeg;base64,") {
		t.Fatal("self-contained HTML did not embed image bytes")
	}
	if err := validateProductDocumentShape(joined); err != nil {
		t.Fatalf("source-backed figure document rejected: %v", err)
	}
}

func TestRunImagePlacementStageUsesClosedToolFreeRequest(t *testing.T) {
	document := Document{Blocks: []Block{{NodeID: "p1", Kind: "prose", Prose: "final prose"}}}
	candidates := []productImageCandidate{{
		CandidateID: "image_01", SourceTitle: "공식 성곽 페이지", SourceAlt: "성곽 경관",
		SourcePageURL: "https://example.com/castle", ImageURL: "https://example.com/castle.jpg",
	}}
	provider := &recordingProvider{outputs: []string{`{"placements":[{"candidate_id":"image_01","after_node_id":"p1","caption":"산 위의 성곽","alt":"산 정상 성곽"}]}`}}
	placements, results, err := runImagePlacementStage(context.Background(), ProductConfig{Provider: provider}, document, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || len(placements.Placements) != 1 || len(provider.requests) != 1 {
		t.Fatalf("image placement result=%#v results=%d requests=%d", placements, len(results), len(provider.requests))
	}
	request := provider.requests[0]
	if request.UserText != "report IL il_images" || request.MCPMode != "disabled" || !request.DisableTools || !request.IgnoreUserConfig || !request.EphemeralSession || !request.ReplaceMCPTools || len(request.ExtraMCPTools) != 0 || request.ReportILSources != nil || len(request.OutputJSONSchema) == 0 {
		t.Fatalf("image placement request escaped closed profile: %#v", request)
	}
	if !strings.Contains(request.Prompt, "final prose") || strings.Contains(request.Prompt, "jpeg") || strings.Contains(request.Prompt, "data:image") {
		t.Fatalf("image placement prompt lost prose or included image bytes: %s", request.Prompt)
	}
}

func TestValidateImagePlacementsRejectsMutationShapedSelections(t *testing.T) {
	document := Document{Blocks: []Block{{NodeID: "p1", Kind: "prose", Prose: "one"}}}
	candidates := []productImageCandidate{{CandidateID: "image_01"}}
	for name, placements := range map[string][]imagePlacement{
		"missing node": {{CandidateID: "image_01", AfterNodeID: "missing", Caption: "caption", Alt: "alt"}},
		"duplicate candidate": {
			{CandidateID: "image_01", AfterNodeID: "p1", Caption: "caption", Alt: "alt"},
			{CandidateID: "image_01", AfterNodeID: "p1", Caption: "caption", Alt: "alt"},
		},
		"empty caption":    {{CandidateID: "image_01", AfterNodeID: "p1", Alt: "alt"}},
		"credential alt":   {{CandidateID: "image_01", AfterNodeID: "p1", Caption: "caption", Alt: "Authorization: Bearer abcdefgh"}},
		"oversize caption": {{CandidateID: "image_01", AfterNodeID: "p1", Caption: strings.Repeat("가", 321), Alt: "alt"}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateImagePlacements(document, candidates, placements); err == nil {
				t.Fatal("invalid image placement accepted")
			}
		})
	}
}

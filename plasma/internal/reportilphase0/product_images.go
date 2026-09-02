package reportilphase0

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"net/url"
	"reflect"
	"sort"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

const (
	maxProductFigures         = 3
	maxProductImageCandidates = 24
	maxProductImageBytes      = 10 << 20
)

type ProductImageFetcher func(context.Context, string) (ProductImage, error)

type ProductImage struct {
	MediaType string
	Content   []byte
	Width     int
	Height    int
}

func (image ProductImage) Validate() error {
	if !supportedProductImageMediaType(image.MediaType) || len(image.Content) == 0 || len(image.Content) > maxProductImageBytes {
		return fmt.Errorf("fetched product image is invalid")
	}
	return nil
}

type productImageCandidate struct {
	CandidateID      string `json:"candidate_id"`
	AcceptedOrdinal  int    `json:"accepted_ordinal"`
	SnapshotID       string `json:"-"`
	SourcePageURL    string `json:"source_page_url"`
	ImageURL         string `json:"image_url"`
	SourceAlt        string `json:"source_alt,omitempty"`
	SourceTitle      string `json:"source_title,omitempty"`
	SourceArtifactID string `json:"-"`
	MediaType        string `json:"-"`
	Content          []byte `json:"-"`
}

type imagePlacementDraft struct {
	Placements []imagePlacement `json:"placements"`
}

type imagePlacement struct {
	CandidateID string `json:"candidate_id"`
	AfterNodeID string `json:"after_node_id"`
	Caption     string `json:"caption"`
	Alt         string `json:"alt"`
}

type ManifestImage struct {
	ArtifactID     string `json:"artifact_id"`
	AssetID        string `json:"asset_id"`
	SHA256         string `json:"sha256"`
	ByteSize       int    `json:"byte_size"`
	MediaType      string `json:"media_type"`
	SourceSnapshot string `json:"source_snapshot_receipt"`
	SourceArtifact string `json:"source_artifact_id,omitempty"`
	SourcePageURL  string `json:"source_page_url,omitempty"`
	SourceImageURL string `json:"source_image_url,omitempty"`
}

func collectProductImageCandidates(ctx context.Context, reader SourceReader, catalog reportilcontract.SourceCatalog, fetch ProductImageFetcher) ([]productImageCandidate, error) {
	if fetch == nil {
		return nil, nil
	}
	snapshots, err := reader.ListSourceSnapshots(ctx, catalog.MissionID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]reportilcontract.SourceSnapshot, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.Active && snapshot.MissionID == catalog.MissionID {
			byID[snapshot.SnapshotID] = snapshot
		}
	}
	var candidates []productImageCandidate
	var sourceLinks []productImageSourceLinks
	seenURLs := map[string]bool{}
	for _, entry := range catalog.Sources {
		snapshot, ok := byID[entry.SnapshotID]
		if !ok {
			return nil, fmt.Errorf("selected image source snapshot is unavailable")
		}
		pageURL := publicCitationURL(snapshot.ExternalURI)
		catalogArtifacts := make(map[string]reportilcontract.SourceCatalogArtifact, len(entry.Artifacts))
		for _, artifact := range entry.Artifacts {
			catalogArtifacts[artifact.ArtifactID] = artifact
		}
		var entryLinks []productImageLink
		for _, artifactID := range snapshot.ArtifactIDs {
			frozenArtifact, ok := catalogArtifacts[artifactID]
			if !ok {
				continue
			}
			artifact, err := reader.GetArtifact(ctx, artifactID)
			if err != nil {
				return nil, err
			}
			if artifact.MissionID != catalog.MissionID ||
				artifact.ByteSize != int64(len(artifact.Content)) ||
				artifact.ByteSize != frozenArtifact.ByteSize ||
				artifact.SHA256 == "" ||
				!strings.EqualFold(artifact.SHA256, SHA256(artifact.Content)) ||
				!strings.EqualFold(artifact.SHA256, frozenArtifact.SHA256) ||
				baseMediaType(artifact.MediaType) != baseMediaType(frozenArtifact.MediaType) {
				return nil, fmt.Errorf("image candidate source artifact is outside the frozen catalog")
			}
			baseType := baseMediaType(artifact.MediaType)
			switch {
			case supportedProductImageMediaType(baseType):
				if len(artifact.Content) > maxProductImageBytes ||
					len(candidates) >= maxProductImageCandidates {
					continue
				}
				candidateURL := publicCitationURL(snapshot.ExternalURI)
				if candidateURL == "" || seenURLs[candidateURL] {
					continue
				}
				seenURLs[candidateURL] = true
				candidates = append(candidates, productImageCandidate{
					AcceptedOrdinal: entry.AcceptedOrdinal, SnapshotID: snapshot.SnapshotID,
					SourcePageURL: pageURL, ImageURL: candidateURL, SourceAlt: snapshot.Title,
					SourceTitle: snapshot.Title, SourceArtifactID: artifact.ArtifactID, MediaType: baseType,
					Content: append([]byte(nil), artifact.Content...),
				})
			case baseType == "text/html" && pageURL != "":
				entryLinks = append(entryLinks, htmlImageLinks(artifact.Content, pageURL)...)
			}
		}
		if len(entryLinks) > 0 {
			sourceLinks = append(sourceLinks, productImageSourceLinks{
				AcceptedOrdinal: entry.AcceptedOrdinal,
				SnapshotID:      snapshot.SnapshotID,
				SourcePageURL:   pageURL,
				SourceTitle:     snapshot.Title,
				Links:           uniqueProductImageLinks(entryLinks),
			})
		}
	}
	fetchAttempts := 0
	for round := 0; fetchAttempts < maxProductImageCandidates && len(candidates) < maxProductImageCandidates; round++ {
		hasRoundLinks := false
		for _, source := range sourceLinks {
			if round >= len(source.Links) {
				continue
			}
			hasRoundLinks = true
			link := source.Links[round]
			if seenURLs[link.URL] {
				continue
			}
			seenURLs[link.URL] = true
			fetchAttempts++
			image, err := fetch(ctx, link.URL)
			if err == nil && supportedProductImageMediaType(image.MediaType) && len(image.Content) > 0 && len(image.Content) <= maxProductImageBytes && usefulProductImageDimensions(image.Width, image.Height) {
				candidates = append(candidates, productImageCandidate{
					AcceptedOrdinal: source.AcceptedOrdinal, SnapshotID: source.SnapshotID,
					SourcePageURL: source.SourcePageURL, ImageURL: link.URL, SourceAlt: link.Alt,
					SourceTitle: source.SourceTitle, MediaType: baseMediaType(image.MediaType),
					Content: append([]byte(nil), image.Content...),
				})
			}
			if fetchAttempts >= maxProductImageCandidates || len(candidates) >= maxProductImageCandidates {
				break
			}
		}
		if !hasRoundLinks {
			break
		}
	}
	for index := range candidates {
		candidates[index].CandidateID = fmt.Sprintf("image_%02d", index+1)
	}
	return candidates, nil
}

type productImageSourceLinks struct {
	AcceptedOrdinal int
	SnapshotID      string
	SourcePageURL   string
	SourceTitle     string
	Links           []productImageLink
}

type productImageLink struct {
	URL string
	Alt string
}

func uniqueProductImageLinks(links []productImageLink) []productImageLink {
	result := make([]productImageLink, 0, len(links))
	seen := make(map[string]bool, len(links))
	for _, link := range links {
		if seen[link.URL] {
			continue
		}
		seen[link.URL] = true
		result = append(result, link)
	}
	return result
}

func htmlImageLinks(content []byte, pageURL string) []productImageLink {
	root, err := html.Parse(bytes.NewReader(content))
	if err != nil {
		return nil
	}
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil
	}
	var links []productImageLink
	seen := map[string]bool{}
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && strings.EqualFold(node.Data, "img") {
			var raw, alt string
			for _, attr := range node.Attr {
				switch strings.ToLower(attr.Key) {
				case "src":
					raw = attr.Val
				case "alt":
					alt = strings.Join(strings.Fields(attr.Val), " ")
				}
			}
			if resolved := resolveProductImageURL(base, raw); resolved != "" && !seen[resolved] {
				seen[resolved] = true
				links = append(links, productImageLink{URL: resolved, Alt: alt})
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return links
}

func resolveProductImageURL(base *url.URL, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(strings.ToLower(raw), "data:") {
		return ""
	}
	ref, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(ref)
	if !samePublicImageHost(base, resolved) {
		return ""
	}
	return publicCitationURL(resolved.String())
}

func samePublicImageHost(page, image *url.URL) bool {
	if page == nil || image == nil ||
		(!strings.EqualFold(image.Scheme, "https") && !strings.EqualFold(image.Scheme, "http")) {
		return false
	}
	pageHost := strings.TrimSuffix(strings.ToLower(page.Hostname()), ".")
	imageHost := strings.TrimSuffix(strings.ToLower(image.Hostname()), ".")
	if pageHost == "" || imageHost == "" {
		return false
	}
	pageDomain, pageErr := publicsuffix.EffectiveTLDPlusOne(pageHost)
	imageDomain, imageErr := publicsuffix.EffectiveTLDPlusOne(imageHost)
	if pageErr == nil && imageErr == nil && strings.EqualFold(pageDomain, imageDomain) {
		return true
	}
	return publicImageHostFamily(pageHost) != "" && publicImageHostFamily(pageHost) == publicImageHostFamily(imageHost)
}

func publicImageHostFamily(host string) string {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	switch {
	case host == "dcinside.com",
		strings.HasSuffix(host, ".dcinside.com"),
		host == "dcinside.co.kr",
		strings.HasSuffix(host, ".dcinside.co.kr"):
		return "dcinside"
	default:
		return ""
	}
}

func usefulProductImageDimensions(width, height int) bool {
	if width <= 0 || height <= 0 {
		return true
	}
	return width >= 480 && height >= 240
}

func supportedProductImageMediaType(mediaType string) bool {
	switch baseMediaType(mediaType) {
	case "image/png", "image/jpeg", "image/gif":
		return true
	default:
		return false
	}
}

func baseMediaType(value string) string {
	base, _, err := mime.ParseMediaType(value)
	if err != nil {
		base = value
	}
	return strings.ToLower(strings.TrimSpace(base))
}

func productImageFilename(mediaType string, ordinal int) string {
	extension := ".jpg"
	switch baseMediaType(mediaType) {
	case "image/png":
		extension = ".png"
	case "image/gif":
		extension = ".gif"
	}
	return fmt.Sprintf("report-image-%d%s", ordinal, extension)
}

func providerImagePlacementSchema(document Document, candidates []productImageCandidate) []byte {
	candidateIDs := make([]any, 0, len(candidates))
	for _, candidate := range candidates {
		candidateIDs = append(candidateIDs, candidate.CandidateID)
	}
	nodeIDs := make([]any, 0, len(document.Blocks))
	for _, block := range document.Blocks {
		if block.Kind != "section" {
			nodeIDs = append(nodeIDs, block.NodeID)
		}
	}
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "additionalProperties": false,
		"required": []string{"placements"},
		"properties": map[string]any{"placements": map[string]any{
			"type": "array", "minItems": 0, "maxItems": maxProductFigures,
			"items": map[string]any{"type": "object", "additionalProperties": false,
				"required": []string{"candidate_id", "after_node_id", "caption", "alt"},
				"properties": map[string]any{
					"candidate_id": map[string]any{"enum": candidateIDs}, "after_node_id": map[string]any{"enum": nodeIDs},
					"caption": map[string]any{"type": "string", "minLength": 1, "maxLength": 320},
					"alt":     map[string]any{"type": "string", "minLength": 1, "maxLength": 320},
				},
			},
		}},
	}
	return mustMarshal(schema)
}

func runImagePlacementStage(ctx context.Context, config ProductConfig, document Document, candidates []productImageCandidate) (imagePlacementDraft, []agentexec.AgentResult, error) {
	if len(candidates) == 0 {
		return imagePlacementDraft{}, nil, nil
	}
	metadata := make([]map[string]any, 0, len(candidates))
	for _, candidate := range candidates {
		metadata = append(metadata, map[string]any{
			"candidate_id": candidate.CandidateID, "source_title": candidate.SourceTitle,
			"source_alt": candidate.SourceAlt, "source_page_url": candidate.SourcePageURL,
		})
	}
	blocks := make([]map[string]string, 0, len(document.Blocks))
	for _, block := range document.Blocks {
		if block.Kind == "section" {
			blocks = append(blocks, map[string]string{"node_id": block.NodeID, "kind": block.Kind, "text": block.Title})
			continue
		}
		blocks = append(blocks, map[string]string{"node_id": block.NodeID, "kind": block.Kind, "text": blockReaderText(block)})
	}
	prompt := stagePrompt("il_images", fmt.Sprintf(`Select zero to %d source-backed images that materially help a reader understand this finalized report. Choose only supplied candidate_id values and place each after one existing non-section node. Do not rewrite, summarize, or modify prose. Avoid logos, navigation icons, duplicate views, and images whose meaning is unclear from source metadata. Write concise natural captions and useful alt text in the report language. Return only strict JSON matching the schema.
FINALIZED REPORT BLOCKS:
%s
ACCEPTED-SOURCE IMAGE CANDIDATES:
%s`, maxProductFigures, mustMarshal(blocks), mustMarshal(metadata)))
	return runJSONStageWithSchema(ctx, config, "il_images", prompt, providerImagePlacementSchema(document, candidates), func(value imagePlacementDraft) error {
		return validateImagePlacements(document, candidates, value.Placements)
	})
}

func validateImagePlacements(document Document, candidates []productImageCandidate, placements []imagePlacement) error {
	if len(placements) > maxProductFigures {
		return fmt.Errorf("image placement exceeds product figure ceiling")
	}
	candidateByID := make(map[string]productImageCandidate, len(candidates))
	for _, candidate := range candidates {
		candidateByID[candidate.CandidateID] = candidate
	}
	nodes := map[string]bool{}
	for _, block := range document.Blocks {
		if block.Kind != "section" {
			nodes[block.NodeID] = true
		}
	}
	seenCandidate, seenNode := map[string]bool{}, map[string]bool{}
	for _, placement := range placements {
		caption, alt := strings.TrimSpace(placement.Caption), strings.TrimSpace(placement.Alt)
		if _, ok := candidateByID[placement.CandidateID]; !ok || !nodes[placement.AfterNodeID] || seenCandidate[placement.CandidateID] || seenNode[placement.AfterNodeID] || caption == "" || alt == "" || len([]rune(caption)) > 320 || len([]rune(alt)) > 320 || validateProviderContent(caption) != nil || validateProviderContent(alt) != nil {
			return fmt.Errorf("image placement is invalid")
		}
		seenCandidate[placement.CandidateID], seenNode[placement.AfterNodeID] = true, true
	}
	return nil
}

func joinProductImages(document Document, catalog reportilcontract.SourceCatalog, candidates []productImageCandidate, placements []imagePlacement, newID func(string) string) (Document, []ProductArtifact, []ManifestImage, error) {
	if err := validateImagePlacements(document, candidates, placements); err != nil {
		return Document{}, nil, nil, err
	}
	original := append([]Block(nil), document.Blocks...)
	candidateByID := make(map[string]productImageCandidate, len(candidates))
	for _, candidate := range candidates {
		candidateByID[candidate.CandidateID] = candidate
	}
	placementByNode := make(map[string]imagePlacement, len(placements))
	for _, placement := range placements {
		placementByNode[placement.AfterNodeID] = placement
	}
	joined := make([]Block, 0, len(document.Blocks)+len(placements))
	var artifacts []ProductArtifact
	var manifest []ManifestImage
	for _, block := range document.Blocks {
		joined = append(joined, block)
		placement, ok := placementByNode[block.NodeID]
		if !ok {
			continue
		}
		candidate := candidateByID[placement.CandidateID]
		assetID := newID("asset")
		artifactID := newID("art")
		filename := productImageFilename(candidate.MediaType, len(artifacts)+1)
		asset := Asset{
			AssetID: assetID, MediaType: candidate.MediaType, SHA256: SHA256(candidate.Content),
			DataBase64: base64.StdEncoding.EncodeToString(candidate.Content), Alt: strings.TrimSpace(placement.Alt),
			LicenseStatus: "private_source",
			ArtifactID:    artifactID, SourceSnapshotReceipt: sourceSnapshotReceiptForID(catalog, candidate.SnapshotID),
			SourcePageURL: candidate.SourcePageURL, SourceImageURL: candidate.ImageURL,
		}
		document.Assets = append(document.Assets, asset)
		joined = append(joined, Block{
			NodeID: newID("figure"), Kind: "figure", ParentNodeID: block.ParentNodeID,
			Figure:             &Figure{AssetID: assetID, Caption: strings.TrimSpace(placement.Caption), Alt: asset.Alt},
			PresentationIntent: "supporting",
		})
		artifacts = append(artifacts, ProductArtifact{ID: artifactID, Kind: "image", MediaType: candidate.MediaType, Content: append([]byte(nil), candidate.Content...), Role: "asset", Filename: filename, AssetID: assetID})
		manifest = append(manifest, ManifestImage{
			ArtifactID: artifactID, AssetID: assetID, SHA256: asset.SHA256, ByteSize: len(candidate.Content), MediaType: candidate.MediaType,
			SourceSnapshot: asset.SourceSnapshotReceipt, SourceArtifact: candidate.SourceArtifactID,
			SourcePageURL: asset.SourcePageURL, SourceImageURL: asset.SourceImageURL,
		})
	}
	document.Blocks = joined
	if err := preserveProseBlocks(original, document.Blocks); err != nil {
		return Document{}, nil, nil, err
	}
	for i := range artifacts {
		artifacts[i].SHA256 = SHA256(artifacts[i].Content)
		artifacts[i].ByteSize = len(artifacts[i].Content)
	}
	return document, artifacts, manifest, nil
}

func preserveProseBlocks(original, joined []Block) error {
	var filtered []Block
	for _, block := range joined {
		if block.Kind != "figure" {
			filtered = append(filtered, block)
		}
	}
	if len(filtered) != len(original) {
		return fmt.Errorf("image joining changed finalized document structure")
	}
	for index := range original {
		if !reflect.DeepEqual(original[index], filtered[index]) {
			return fmt.Errorf("image joining changed finalized report prose or structure")
		}
	}
	return nil
}

func sourceSnapshotReceiptForID(catalog reportilcontract.SourceCatalog, snapshotID string) string {
	for _, entry := range catalog.Sources {
		if entry.SnapshotID == snapshotID {
			return entry.SnapshotReceipt
		}
	}
	return ""
}

func blockReaderText(block Block) string {
	values := []string{block.Prose, block.Code}
	values = append(values, block.Items...)
	if block.Equation != nil {
		values = append(values, block.Equation.Expression)
	}
	if block.Table != nil {
		values = append(values, block.Table.Caption)
		for _, row := range block.Table.Rows {
			values = append(values, row...)
		}
	}
	return strings.Join(strings.Fields(strings.Join(values, " ")), " ")
}

func sortManifestImages(values []ManifestImage) []ManifestImage {
	result := append([]ManifestImage(nil), values...)
	sort.Slice(result, func(i, j int) bool { return result[i].ArtifactID < result[j].ArtifactID })
	return result
}

package reportilphase0

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"sort"
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportilsource"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

type SourceCatalogDisposition struct {
	AcceptedOrdinal int    `json:"accepted_ordinal"`
	SourceKey       string `json:"source_key,omitempty"`
	SnapshotReceipt string `json:"snapshot_receipt"`
	Status          string `json:"status"`
	Reason          string `json:"reason,omitempty"`
}

type SourceCatalogBuild struct {
	Catalog                   reportilcontract.SourceCatalog
	ImageCatalog              reportilcontract.SourceCatalog
	Dispositions              []SourceCatalogDisposition
	CitationByKey             map[string]SourceCitation
	ReadableByAcceptedOrdinal map[int]string `json:"-"`
}

type SourceCitation struct {
	VisibleLabel string
	URL          string
}

type sourceCatalogReadable struct {
	Text           string
	Extraction     string
	DocumentTitle  string
	PrimaryHeading string
}

func BuildSourceCatalog(ctx context.Context, reader SourceReader, missionID string) (reportilcontract.SourceCatalog, error) {
	build, err := BuildSourceCatalogForSelection(ctx, reader, missionID)
	if err != nil {
		return reportilcontract.SourceCatalog{}, err
	}
	for _, disposition := range build.Dispositions {
		if disposition.Status != "available" && disposition.Status != "image_only" {
			return reportilcontract.SourceCatalog{}, fmt.Errorf("source catalog contains an unusable accepted source")
		}
	}
	return build.Catalog, nil
}

// BuildSourceCatalogForSelection exposes the frozen server-owned catalog build
// for verified legacy checkpoint recovery as well as current product runs.
func BuildSourceCatalogForSelection(ctx context.Context, reader SourceReader, missionID string) (SourceCatalogBuild, error) {
	missionID = strings.TrimSpace(missionID)
	if missionID == "" {
		return SourceCatalogBuild{}, fmt.Errorf("source catalog requires a mission")
	}
	snapshots, err := reader.ListSourceSnapshots(ctx, missionID)
	if err != nil {
		return SourceCatalogBuild{}, err
	}
	if len(snapshots) == 0 {
		return SourceCatalogBuild{}, fmt.Errorf("source catalog requires at least one accepted source")
	}
	for _, snapshot := range snapshots {
		if snapshot.Active && strings.TrimSpace(snapshot.MissionID) != missionID {
			return SourceCatalogBuild{}, fmt.Errorf("source catalog contains a cross-mission source")
		}
	}
	sort.Slice(snapshots, func(i, j int) bool { return snapshots[i].SnapshotID < snapshots[j].SnapshotID })
	catalog := reportilcontract.SourceCatalog{MissionID: missionID}
	build := SourceCatalogBuild{
		ImageCatalog:              reportilcontract.SourceCatalog{MissionID: missionID},
		CitationByKey:             map[string]SourceCitation{},
		ReadableByAcceptedOrdinal: map[int]string{},
	}
	activeOrdinal := 0
	for _, snapshot := range snapshots {
		if !snapshot.Active {
			continue
		}
		activeOrdinal++
		disposition := SourceCatalogDisposition{AcceptedOrdinal: activeOrdinal, SnapshotReceipt: snapshotReceipt(snapshot), Status: "available"}
		entry, readable, err := buildSourceCatalogEntry(ctx, reader, snapshot)
		if err != nil {
			if errors.Is(err, reportilsource.ErrUnusableReadableText) {
				disposition.Status = "excluded"
				disposition.Reason = "unusable_readable_text"
				build.Dispositions = append(build.Dispositions, disposition)
				continue
			}
			return SourceCatalogBuild{}, err
		}
		entry.AcceptedOrdinal = activeOrdinal
		if sourceCatalogEntryCanYieldImage(entry) {
			imageEntry := entry
			imageEntry.SourceKey = fmt.Sprintf("source_%03d", len(build.ImageCatalog.Sources)+1)
			build.ImageCatalog.Sources = append(build.ImageCatalog.Sources, imageEntry)
		}
		if sourceCatalogEntryHasReadableEvidence(entry) {
			entry.SourceKey = fmt.Sprintf("source_%03d", len(catalog.Sources)+1)
			disposition.SourceKey = entry.SourceKey
			catalog.Sources = append(catalog.Sources, entry)
			build.CitationByKey[entry.SourceKey] = sourceCitation(snapshot, readable)
			build.ReadableByAcceptedOrdinal[activeOrdinal] = readable.Text
		} else {
			disposition.Status = "image_only"
			disposition.Reason = "illustration_only"
		}
		build.Dispositions = append(build.Dispositions, disposition)
	}
	if activeOrdinal == 0 {
		return SourceCatalogBuild{}, fmt.Errorf("source catalog has no active accepted sources")
	}
	if len(catalog.Sources) == 0 {
		return SourceCatalogBuild{}, fmt.Errorf("source catalog has no usable accepted text sources")
	}
	catalog, err = reportilcontract.SealSourceCatalog(catalog)
	if err != nil {
		return SourceCatalogBuild{}, err
	}
	build.Catalog = catalog
	if len(build.ImageCatalog.Sources) > 0 {
		build.ImageCatalog, err = reportilcontract.SealSourceCatalog(build.ImageCatalog)
		if err != nil {
			return SourceCatalogBuild{}, err
		}
	}
	return build, nil
}

func sourceCatalogEntryCanYieldImage(entry reportilcontract.SourceCatalogEntry) bool {
	for _, artifact := range entry.Artifacts {
		mediaType := baseMediaType(artifact.MediaType)
		if supportedProductImageMediaType(mediaType) || mediaType == "text/html" {
			return true
		}
	}
	return false
}

func sourceCatalogEntryHasReadableEvidence(entry reportilcontract.SourceCatalogEntry) bool {
	return entry.Extraction != "binary_metadata"
}

func (build SourceCatalogBuild) HasExclusions() bool {
	for _, disposition := range build.Dispositions {
		if disposition.Status != "available" && disposition.Status != "image_only" {
			return true
		}
	}
	return false
}

func (build SourceCatalogBuild) citationCatalog(catalog reportilcontract.SourceCatalog) map[int]SourceCitation {
	citations := make(map[int]SourceCitation, len(catalog.Sources))
	for _, entry := range catalog.Sources {
		citation, ok := build.CitationByKey[sourceKeyByAcceptedOrdinal(build.Catalog, entry.AcceptedOrdinal)]
		if ok {
			citations[entry.AcceptedOrdinal] = citation
		}
	}
	return citations
}

func sourceKeyByAcceptedOrdinal(catalog reportilcontract.SourceCatalog, ordinal int) string {
	for _, entry := range catalog.Sources {
		if entry.AcceptedOrdinal == ordinal {
			return entry.SourceKey
		}
	}
	return ""
}

func sourceCitation(snapshot reportilcontract.SourceSnapshot, readable ...sourceCatalogReadable) SourceCitation {
	switch strings.ToLower(strings.TrimSpace(snapshot.ConnectorType)) {
	case "url", source.ConnectorTypePDFURL, source.ConnectorTypeMediaURL:
	default:
		return SourceCitation{}
	}
	publicURL := publicCitationURL(snapshot.ExternalURI)
	if publicURL == "" {
		return SourceCitation{}
	}
	candidates := []string{snapshot.Title}
	if len(readable) > 0 {
		candidates = append(candidates, readable[0].DocumentTitle, readable[0].PrimaryHeading)
	}
	label := ""
	for _, candidate := range candidates {
		label = citationLabel(candidate)
		if label != "" && !citationLabelIsHostname(label, publicURL) {
			break
		}
		label = ""
	}
	if label == "" && len(readable) > 0 && readable[0].Extraction != "html_visible_text" {
		label = firstReadableCitationLabel(readable[0].Text)
	}
	if label == "" {
		label = publicCitationFallbackLabel(publicURL)
	}
	return SourceCitation{VisibleLabel: label, URL: publicURL}
}

func citationLabel(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" || len(value) > 240 || validateProviderContent(value) != nil || citationLabelIsBoilerplate(value) {
		return ""
	}
	return value
}

func citationLabelIsBoilerplate(value string) bool {
	normalized := strings.ToLower(strings.Join(strings.Fields(value), " "))
	for _, exact := range []string{
		"検索", "search", "menu", "メニュー", "ページの先頭です。", "本文へ", "トップへ戻る",
	} {
		if normalized == strings.ToLower(exact) {
			return true
		}
	}
	for _, fragment := range []string{
		"メニューを飛ばして本文へ", "skip to main content", "skip to content", "back to top",
	} {
		if strings.Contains(normalized, strings.ToLower(fragment)) {
			return true
		}
	}
	return false
}

func citationLabelIsHostname(label, publicURL string) bool {
	if label == "" {
		return false
	}
	parsed, err := url.Parse(publicURL)
	return err == nil && strings.EqualFold(strings.TrimSuffix(label, "."), parsed.Hostname())
}

func firstReadableCitationLabel(readable string) string {
	for _, line := range strings.Split(readable, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "#*-• "))
		if label := citationLabel(line); label != "" {
			return label
		}
	}
	return ""
}

func publicCitationFallbackLabel(publicURL string) string {
	parsed, err := url.Parse(publicURL)
	if err != nil {
		return ""
	}
	label := strings.Trim(parsed.EscapedPath(), "/")
	if unescaped, unescapeErr := url.PathUnescape(label); unescapeErr == nil {
		label = unescaped
	}
	label = strings.NewReplacer("/", " › ", "-", " ", "_", " ").Replace(label)
	label = citationLabel(label)
	if label != "" {
		return label
	}
	return parsed.Hostname()
}

func publicCitationURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.User != nil || parsed.Hostname() == "" {
		return ""
	}
	if !strings.EqualFold(parsed.Scheme, "https") && !strings.EqualFold(parsed.Scheme, "http") {
		return ""
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".ts.net") {
		return ""
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		ip = ip.Unmap()
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() || netip.MustParsePrefix("100.64.0.0/10").Contains(ip) {
			return ""
		}
	} else {
		if _, err := publicsuffix.EffectiveTLDPlusOne(host); err != nil {
			return ""
		}
		if _, icann := publicsuffix.PublicSuffix(host); !icann {
			return ""
		}
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""
	return parsed.String()
}

func buildSourceCatalogEntry(ctx context.Context, reader SourceReader, snapshot reportilcontract.SourceSnapshot) (reportilcontract.SourceCatalogEntry, sourceCatalogReadable, error) {
	retrievalPolicy := strings.TrimSpace(snapshot.RetrievalPolicy)
	if retrievalPolicy == "" {
		retrievalPolicy = source.RetrievalPolicySnapshotOnly
	}
	entry := reportilcontract.SourceCatalogEntry{
		SnapshotID:      snapshot.SnapshotID,
		SnapshotReceipt: snapshotReceipt(snapshot),
		ContentHash:     snapshot.ContentHash,
		RetrievalPolicy: retrievalPolicy,
	}
	var readable reportilsource.Readable
	var err error
	if retrievalPolicy == source.RetrievalPolicyLiveReference {
		read, err := reader.ReadLive(ctx, snapshot.MissionID, snapshot.SnapshotID, reportilsource.MaxReadableBytes)
		if err != nil {
			return reportilcontract.SourceCatalogEntry{}, sourceCatalogReadable{}, err
		}
		if read.Binary || read.Truncated || strings.TrimSpace(read.Content) == "" {
			return reportilcontract.SourceCatalogEntry{}, sourceCatalogReadable{}, fmt.Errorf("source catalog rejects binary, empty, or truncated live source")
		}
		mediaType := "text/plain"
		if strings.TrimSpace(read.Extraction) == "pdf_text" {
			mediaType = "text/plain"
		}
		readable, err = reportilsource.Extract([]byte(read.Content), mediaType)
		if err != nil {
			return reportilcontract.SourceCatalogEntry{}, sourceCatalogReadable{}, err
		}
		if read.SHA256 != "" && sha256Hex([]byte(read.Content)) != read.SHA256 {
			return reportilcontract.SourceCatalogEntry{}, sourceCatalogReadable{}, fmt.Errorf("source catalog rejects changed live-source continuity")
		}
		if read.ObservationReceipt == "" {
			return reportilcontract.SourceCatalogEntry{}, sourceCatalogReadable{}, fmt.Errorf("source catalog requires a live observation receipt")
		}
		entry.ObservationReceipt = read.ObservationReceipt
		if strings.TrimSpace(read.Extraction) != "" {
			readable.Extraction = read.Extraction
		} else {
			readable.Extraction = "live_text"
		}
	} else {
		if len(snapshot.ArtifactIDs) == 0 {
			return reportilcontract.SourceCatalogEntry{}, sourceCatalogReadable{}, fmt.Errorf("source catalog rejects source without artifact")
		}
		var combined strings.Builder
		var imageArtifacts int
		for _, artifactID := range snapshot.ArtifactIDs {
			artifact, err := reader.GetArtifact(ctx, artifactID)
			if err != nil {
				return reportilcontract.SourceCatalogEntry{}, sourceCatalogReadable{}, err
			}
			if artifact.MissionID != snapshot.MissionID || artifact.SHA256 == "" || sha256Hex(artifact.Content) != artifact.SHA256 || artifact.ByteSize != int64(len(artifact.Content)) {
				return reportilcontract.SourceCatalogEntry{}, sourceCatalogReadable{}, fmt.Errorf("source catalog artifact hash, size, or mission mismatch")
			}
			if supportedProductImageMediaType(artifact.MediaType) {
				imageArtifacts++
			} else {
				part, err := reportilsource.Extract(artifact.Content, artifact.MediaType)
				if err != nil {
					return reportilcontract.SourceCatalogEntry{}, sourceCatalogReadable{}, err
				}
				if combined.Len() > 0 {
					combined.WriteString("\n\n")
				}
				combined.WriteString(part.Text)
			}
			entry.Artifacts = append(entry.Artifacts, reportilcontract.SourceCatalogArtifact{
				ArtifactID: artifact.ArtifactID,
				SHA256:     artifact.SHA256,
				ByteSize:   artifact.ByteSize,
				MediaType:  artifact.MediaType,
			})
		}
		if snapshot.ContentHash != "" && !strings.EqualFold(snapshot.ContentHash, catalogSnapshotHash(entry.Artifacts)) {
			return reportilcontract.SourceCatalogEntry{}, sourceCatalogReadable{}, fmt.Errorf("source catalog snapshot hash mismatch")
		}
		if combined.Len() == 0 && imageArtifacts > 0 {
			combined.WriteString("Image source available for report illustration. Content inspection is metadata-only.")
		}
		readable, err = reportilsource.Extract([]byte(combined.String()), "text/plain")
		if err != nil {
			return reportilcontract.SourceCatalogEntry{}, sourceCatalogReadable{}, err
		}
		if imageArtifacts == len(entry.Artifacts) {
			readable.Extraction = "binary_metadata"
		} else if len(entry.Artifacts) == 1 {
			part, partErr := reader.GetArtifact(ctx, entry.Artifacts[0].ArtifactID)
			if partErr != nil {
				return reportilcontract.SourceCatalogEntry{}, sourceCatalogReadable{}, partErr
			}
			partReadable, partErr := reportilsource.Extract(part.Content, part.MediaType)
			if partErr != nil {
				return reportilcontract.SourceCatalogEntry{}, sourceCatalogReadable{}, partErr
			}
			readable.Extraction = partReadable.Extraction
			readable.DocumentTitle = partReadable.DocumentTitle
			readable.PrimaryHeading = partReadable.PrimaryHeading
		} else {
			readable.Extraction = "joined_artifact_text"
		}
	}
	entry.ReadableSHA256 = readable.SHA256
	entry.ReadableBytes = readable.ByteSize
	entry.Extraction = readable.Extraction
	return entry, sourceCatalogReadable{
		Text: readable.Text, Extraction: readable.Extraction,
		DocumentTitle: readable.DocumentTitle, PrimaryHeading: readable.PrimaryHeading,
	}, nil
}

func catalogSnapshotHash(artifacts []reportilcontract.SourceCatalogArtifact) string {
	converted := make([]SourceArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		converted = append(converted, SourceArtifact{ArtifactID: artifact.ArtifactID, SHA256: artifact.SHA256})
	}
	return snapshotHashValue(converted)
}

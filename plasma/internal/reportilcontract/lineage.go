package reportilcontract

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

var sha256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

var closedKinds = map[string]struct{}{"narrative": {}, "semantic_il": {}, "markdown": {}, "html": {}, "pdf": {}, "manifest": {}}

var legacyKinds = map[string]struct{}{"narrative": {}, "semantic_il": {}, "flow_attestation": {}, "markdown": {}, "html": {}, "pdf": {}, "manifest": {}}

var imageMediaTypes = map[string]struct{}{"image/png": {}, "image/jpeg": {}, "image/gif": {}}

// TerminalPayload is the strict, provider-independent terminal lineage shape.
type TerminalPayload struct {
	Kind           string
	PendingEventID string
	ArtifactID     string
	Bundle         TerminalArtifactLineage
}

func DecodeTerminalPayload(raw []byte) (TerminalPayload, error) {
	var envelope struct {
		Kind           string          `json:"kind"`
		PendingEventID string          `json:"pending_event_id"`
		ArtifactID     string          `json:"artifact_id"`
		PipelineFamily string          `json:"pipeline_family"`
		ArtifactBundle json.RawMessage `json:"artifact_bundle"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&envelope); err != nil {
		return TerminalPayload{}, fmt.Errorf("decode terminal lineage: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return TerminalPayload{}, fmt.Errorf("terminal lineage has trailing JSON")
		}
		return TerminalPayload{}, fmt.Errorf("malformed trailing JSON: %w", err)
	}
	if len(envelope.ArtifactBundle) == 0 {
		return TerminalPayload{}, fmt.Errorf("terminal lineage requires artifact_bundle")
	}
	var bundle struct {
		PipelineFamily     string                  `json:"pipeline_family"`
		Artifacts          []TerminalArtifactEntry `json:"artifacts"`
		MarkdownArtifactID string                  `json:"markdown_artifact_id"`
		SourceSelection    json.RawMessage         `json:"source_selection"`
		Usage              json.RawMessage         `json:"usage"`
	}
	bd := json.NewDecoder(bytes.NewReader(envelope.ArtifactBundle))
	bd.DisallowUnknownFields()
	if err := bd.Decode(&bundle); err != nil {
		return TerminalPayload{}, fmt.Errorf("decode artifact lineage: %w", err)
	}
	if envelope.PipelineFamily != PipelineFamily || bundle.PipelineFamily != envelope.PipelineFamily {
		return TerminalPayload{}, fmt.Errorf("invalid report IL terminal family")
	}
	var sourceSelection *TerminalSourceSelectionSummary
	if len(bundle.SourceSelection) > 0 {
		if bytes.Equal(bytes.TrimSpace(bundle.SourceSelection), []byte("null")) {
			return TerminalPayload{}, fmt.Errorf("report IL source selection cannot be null")
		}
		var summary TerminalSourceSelectionSummary
		sd := json.NewDecoder(bytes.NewReader(bundle.SourceSelection))
		sd.DisallowUnknownFields()
		if err := sd.Decode(&summary); err != nil {
			return TerminalPayload{}, fmt.Errorf("decode source selection: %w", err)
		}
		sourceSelection = &summary
	}
	lineage := TerminalArtifactLineage{
		PipelineFamily:     bundle.PipelineFamily,
		Artifacts:          bundle.Artifacts,
		MarkdownArtifactID: bundle.MarkdownArtifactID,
		SourceSelection:    sourceSelection,
	}
	if err := ValidateLineage(envelope.Kind, envelope.PendingEventID, envelope.ArtifactID, lineage); err != nil {
		return TerminalPayload{}, err
	}
	return TerminalPayload{Kind: envelope.Kind, PendingEventID: envelope.PendingEventID, ArtifactID: envelope.ArtifactID, Bundle: lineage}, nil
}

func ValidateLineage(kind, pendingID, artifactID string, lineage TerminalArtifactLineage) error {
	if kind != "markdown_report_artifact" || strings.TrimSpace(pendingID) == "" {
		return fmt.Errorf("invalid report IL terminal envelope")
	}
	if lineage.PipelineFamily != PipelineFamily || strings.TrimSpace(artifactID) == "" || artifactID != lineage.MarkdownArtifactID {
		return fmt.Errorf("invalid report IL family or Markdown binding")
	}
	expectedKinds := closedKinds
	for _, entry := range lineage.Artifacts {
		if entry.Kind == "flow_attestation" {
			expectedKinds = legacyKinds
			break
		}
	}
	seenIDs, seenKinds, seenAssetIDs := map[string]bool{}, map[string]bool{}, map[string]bool{}
	imageCount := 0
	for _, entry := range lineage.Artifacts {
		if entry.ArtifactID == "" || seenIDs[entry.ArtifactID] {
			return fmt.Errorf("report IL lineage has duplicate or empty identity")
		}
		seenIDs[entry.ArtifactID] = true
		if entry.Kind == "image" {
			imageCount++
			_, supportedMediaType := imageMediaTypes[entry.MediaType]
			if imageCount > 3 || entry.Role != "asset" || !supportedMediaType || !validImageFilename(entry.Filename, entry.MediaType) || strings.TrimSpace(entry.AssetID) == "" || seenAssetIDs[entry.AssetID] {
				return fmt.Errorf("report IL lineage has invalid image metadata")
			}
			seenAssetIDs[entry.AssetID] = true
		} else {
			if seenKinds[entry.Kind] {
				return fmt.Errorf("report IL lineage has duplicate kind")
			}
			seenKinds[entry.Kind] = true
			if _, ok := expectedKinds[entry.Kind]; !ok || entry.AssetID != "" {
				return fmt.Errorf("report IL lineage has unsupported kind")
			}
			role, media, filename := expectedArtifact(entry.Kind)
			if entry.Role != role || entry.MediaType != media || entry.Filename != filename {
				return fmt.Errorf("report IL lineage has invalid %s metadata", entry.Kind)
			}
		}
		if !sha256Pattern.MatchString(entry.SHA256) || entry.ByteSize <= 0 {
			return fmt.Errorf("report IL lineage has invalid %s bytes", entry.Kind)
		}
	}
	if len(lineage.Artifacts) != len(expectedKinds)+imageCount {
		return fmt.Errorf("report IL lineage has an invalid artifact count")
	}
	for kind := range expectedKinds {
		if !seenKinds[kind] {
			return fmt.Errorf("report IL lineage missing %s", kind)
		}
	}
	if lineage.SourceSelection != nil {
		if err := validateTerminalSourceSelection(*lineage.SourceSelection); err != nil {
			return err
		}
	}
	return nil
}

func validateTerminalSourceSelection(summary TerminalSourceSelectionSummary) error {
	if summary.AcceptedSources < 1 ||
		summary.UsableSources < 1 ||
		summary.SelectedSources < 1 ||
		summary.SupplementalSources < 0 ||
		summary.ExcludedUnusableSources < 0 ||
		summary.ExcludedBudgetSources < 0 {
		return fmt.Errorf("report IL source selection has invalid counts")
	}
	if summary.AcceptedSources != summary.UsableSources+summary.ExcludedUnusableSources ||
		summary.UsableSources != summary.SelectedSources+summary.SupplementalSources+summary.ExcludedBudgetSources {
		return fmt.Errorf("report IL source selection counts are inconsistent")
	}
	selectionChangedAuthoringSet := summary.SupplementalSources > 0 ||
		summary.ExcludedUnusableSources+summary.ExcludedBudgetSources > 0
	if summary.Applied != selectionChangedAuthoringSet {
		return fmt.Errorf("report IL source selection application state is inconsistent")
	}
	return nil
}

func validImageFilename(filename, mediaType string) bool {
	filename = strings.TrimSpace(filename)
	if filename == "" || filename != filepath.Base(filename) || filename == "." || filename == ".." {
		return false
	}
	extension := strings.ToLower(filepath.Ext(filename))
	switch mediaType {
	case "image/png":
		return extension == ".png"
	case "image/jpeg":
		return extension == ".jpg" || extension == ".jpeg"
	case "image/gif":
		return extension == ".gif"
	default:
		return false
	}
}

func expectedArtifact(kind string) (string, string, string) {
	switch kind {
	case "markdown":
		return "final", "text/markdown; charset=utf-8", "report.md"
	case "html":
		return "derivative", "text/html; charset=utf-8", "report.html"
	case "pdf":
		return "derivative", "application/pdf", "report.pdf"
	case "narrative":
		return "intermediate", "application/json", "narrative.json"
	case "semantic_il":
		return "intermediate", "application/json", "semantic-il.json"
	case "flow_attestation":
		return "intermediate", "application/json", "flow-attestation.json"
	case "manifest":
		return "intermediate", "application/json", "manifest.json"
	default:
		return "", "", ""
	}
}

func ValidateActual(lineage TerminalArtifactLineage, actual []Artifact) error {
	if err := ValidateLineage("markdown_report_artifact", "pending", lineage.MarkdownArtifactID, lineage); err != nil {
		return err
	}
	byID := make(map[string]Artifact, len(actual))
	for _, item := range actual {
		if item.ArtifactID == "" {
			return fmt.Errorf("actual report IL artifact has empty ID")
		}
		if _, exists := byID[item.ArtifactID]; exists {
			return fmt.Errorf("duplicate actual report IL artifact %s", item.ArtifactID)
		}
		byID[item.ArtifactID] = item
	}
	for _, entry := range lineage.Artifacts {
		item, ok := byID[entry.ArtifactID]
		if !ok || item.SHA256 != entry.SHA256 || item.ByteSize != int64(entry.ByteSize) || item.MediaType != entry.MediaType || item.Filename != entry.Filename {
			return fmt.Errorf("lineage does not match artifact %s", entry.ArtifactID)
		}
		sum := sha256.Sum256(item.Content)
		if hex.EncodeToString(sum[:]) != entry.SHA256 {
			return fmt.Errorf("artifact %s hash does not match bytes", entry.ArtifactID)
		}
	}
	if len(byID) != len(lineage.Artifacts) {
		return fmt.Errorf("actual report IL artifact count does not match lineage")
	}

	return nil
}

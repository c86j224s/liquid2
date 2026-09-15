package researchcatalog

import (
	"encoding/json"
	"fmt"
	"strings"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

const (
	legacyUploadedContentKindPDF   = "pdf"
	legacyUploadedContentKindImage = "image"
)

// SummarizeSourceSnapshot returns the transport-neutral summary for a source snapshot.
// Locator parse failures are intentionally ignored so malformed locator data retains
// the same base summary and metadata as the application boundary did before extraction.
func SummarizeSourceSnapshot(snapshot source.Snapshot) ObjectSummary {
	refs := make([]ObjectRef, 0, len(snapshot.ArtifactIDs))
	for _, id := range snapshot.ArtifactIDs {
		refs = append(refs, ObjectRef{ObjectKind: ObjectRawArtifact, ObjectID: id})
	}
	metadata := map[string]any{
		"connector_type":   snapshot.Connector.ConnectorType,
		"retrieval_policy": snapshot.Access.RetrievalPolicy,
		"state":            firstNonEmpty(snapshot.State.State, source.StateActive),
		"removed":          snapshot.State.Removed,
	}
	if snapshot.Connector.ConnectorType == source.ConnectorTypeLocalPath {
		if locator, err := source.ParseLocalPathLocator(snapshot.Locators); err == nil {
			metadata["root_id"] = locator.RootID
			metadata["relative_path"] = locator.RelativePath
			metadata["path_kind"] = locator.PathKind
		}
	}
	if snapshot.Connector.ConnectorType == source.ConnectorTypeMediaURL {
		if locator, err := source.ParseMediaLocator(snapshot.Locators); err == nil {
			metadata["media_kind"] = locator.MediaKind
			metadata["mime_type"] = locator.MIMEType
			metadata["byte_size"] = locator.ByteSize
			metadata["width"] = locator.Width
			metadata["height"] = locator.Height
			metadata["canonical_url"] = locator.CanonicalURL
			metadata["source_page_url"] = locator.SourcePageURL
			metadata["direct_media_url"] = locator.DirectMediaURL
			metadata["license"] = locator.License
			metadata["attribution"] = locator.Attribution
			metadata["inspection_support"] = locator.InspectionSupport
		}
	}
	if snapshot.Connector.ConnectorType == source.ConnectorTypeFileUpload {
		for key, value := range uploadedFileLocatorMetadata(snapshot.Locators) {
			metadata[key] = value
		}
	}
	if snapshot.Connector.ConnectorType == source.ConnectorTypePDFURL || snapshot.Connector.ConnectorType == source.ConnectorTypeFileUpload {
		for key, value := range pdfLocatorMetadata(snapshot.Locators) {
			metadata[key] = value
		}
	}
	return ObjectSummary{ObjectKind: ObjectSourceSnapshot, ObjectID: snapshot.SnapshotID, MissionID: snapshot.MissionID, Summary: firstNonEmpty(snapshot.Title, snapshot.Connector.ExternalURI, snapshot.SnapshotID), Refs: refs, Metadata: metadata}
}

// SummarizeRawArtifact returns the artifact summary using the caller's already
// computed read kind; upload policy remains owned by the application boundary.
func SummarizeRawArtifact(artifact artifactcontract.Raw, readKind string) ObjectSummary {
	return ObjectSummary{ObjectKind: ObjectRawArtifact, ObjectID: artifact.ArtifactID, MissionID: artifact.MissionID, Summary: firstNonEmpty(artifact.Filename, artifact.MediaType, artifact.ArtifactID), Metadata: map[string]any{"byte_size": artifact.ByteSize, "media_type": artifact.MediaType, "read_kind": readKind}}
}

// SummarizeLedgerEvent returns the ledger event summary and ordered, deduplicated
// references extracted from its payload. Malformed payloads retain empty refs.
func SummarizeLedgerEvent(event ledger.Event) ObjectSummary {
	refs := []ObjectRef{}
	var payload struct {
		SnapshotID  string   `json:"snapshot_id"`
		ArtifactID  string   `json:"artifact_id"`
		ArtifactIDs []string `json:"artifact_ids"`
	}
	if json.Unmarshal(event.Payload, &payload) == nil {
		if strings.TrimSpace(payload.SnapshotID) != "" {
			refs = append(refs, ObjectRef{ObjectKind: ObjectSourceSnapshot, ObjectID: strings.TrimSpace(payload.SnapshotID)})
		}
		if strings.TrimSpace(payload.ArtifactID) != "" {
			refs = append(refs, ObjectRef{ObjectKind: ObjectRawArtifact, ObjectID: strings.TrimSpace(payload.ArtifactID)})
		}
		for _, artifactID := range payload.ArtifactIDs {
			artifactID = strings.TrimSpace(artifactID)
			if artifactID != "" {
				refs = append(refs, ObjectRef{ObjectKind: ObjectRawArtifact, ObjectID: artifactID})
			}
		}
	}
	return ObjectSummary{ObjectKind: ObjectLedgerEvent, ObjectID: event.EventID, MissionID: event.MissionID, Summary: fmt.Sprintf("#%d %s", event.Sequence, event.EventType), Refs: DedupeRefs(refs), Metadata: map[string]any{"event_type": event.EventType, "sequence": event.Sequence}}
}

func pdfLocatorMetadata(raw json.RawMessage) map[string]any {
	metadata := map[string]any{}
	if len(raw) == 0 {
		return metadata
	}
	var locators []map[string]any
	if err := json.Unmarshal(raw, &locators); err != nil {
		var locator map[string]any
		if err := json.Unmarshal(raw, &locator); err != nil {
			return metadata
		}
		locators = []map[string]any{locator}
	}
	for _, locator := range locators {
		if !isPDFLocatorMap(locator) {
			continue
		}
		for _, key := range []string{"url", "filename", "original_filename", "sanitized_filename", "mime_type", "media_type", "byte_size", "sha256", "page_count", "text_length", "text_length_known", "extraction_support"} {
			if value, ok := locator[key]; ok {
				metadata[key] = value
			}
		}
		return metadata
	}
	return metadata
}

func uploadedFileLocatorMetadata(raw json.RawMessage) map[string]any {
	metadata := map[string]any{}
	if len(raw) == 0 {
		return metadata
	}
	var locators []map[string]any
	if err := json.Unmarshal(raw, &locators); err != nil {
		var locator map[string]any
		if err := json.Unmarshal(raw, &locator); err != nil {
			return metadata
		}
		locators = []map[string]any{locator}
	}
	for _, locator := range locators {
		locatorType := uploadedFileLocatorMapType(locator)
		if locatorType == "" {
			continue
		}
		metadata["locator_type"] = locatorType
		for _, key := range []string{"original_filename", "sanitized_filename", "filename", "media_kind", "content_kind", "byte_size", "sha256", "uploaded_at"} {
			if value, ok := locator[key]; ok {
				metadata[key] = value
			}
		}
		if filename := firstLocatorMapString(locator, "sanitized_filename", "filename", "original_filename"); filename != "" {
			metadata["filename"] = filename
		}
		if mimeType := firstLocatorMapString(locator, "mime_type", "media_type"); mimeType != "" {
			metadata["mime_type"] = mimeType
		}
		return metadata
	}
	return metadata
}

func uploadedFileLocatorMapType(locator map[string]any) string {
	discriminator := locatorMapDiscriminator(locator)
	switch discriminator {
	case source.LocatorTypeFullDocument, source.LocatorTypePDFDocument, source.LocatorTypeMedia:
		return discriminator
	case source.ConnectorTypeFileUpload:
	default:
		return ""
	}
	contentKind := strings.TrimSpace(fmt.Sprint(locator["content_kind"]))
	mediaType := firstLocatorMapString(locator, "mime_type", "media_type")
	switch {
	case contentKind == legacyUploadedContentKindPDF || mediaType == "application/pdf":
		return source.LocatorTypePDFDocument
	case contentKind == legacyUploadedContentKindImage || strings.HasPrefix(mediaType, "image/"):
		return source.LocatorTypeMedia
	default:
		return source.LocatorTypeFullDocument
	}
}

func isPDFLocatorMap(locator map[string]any) bool {
	discriminator := locatorMapDiscriminator(locator)
	if discriminator == source.LocatorTypePDFDocument {
		return true
	}
	if discriminator != source.ConnectorTypeFileUpload {
		return false
	}
	contentKind := strings.TrimSpace(fmt.Sprint(locator["content_kind"]))
	mediaType := firstLocatorMapString(locator, "mime_type", "media_type")
	return contentKind == legacyUploadedContentKindPDF || mediaType == "application/pdf"
}

func firstLocatorMapString(locator map[string]any, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(fmt.Sprint(locator[key]))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func locatorMapDiscriminator(locator map[string]any) string {
	if value := strings.TrimSpace(fmt.Sprint(locator["locator_type"])); value != "" && value != "<nil>" {
		return value
	}
	value := strings.TrimSpace(fmt.Sprint(locator["kind"]))
	if value == "<nil>" {
		return ""
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// SummarizeEvidence returns the app's canonical evidence projection.
func SummarizeEvidence(record researchrecords.EvidenceRecord) ObjectSummary {
	refs := make([]ObjectRef, 0, len(record.SnapshotRefs)*2)
	for _, ref := range record.SnapshotRefs {
		refs = append(refs, ObjectRef{ObjectKind: ObjectSourceSnapshot, ObjectID: ref.SnapshotID}, ObjectRef{ObjectKind: ObjectRawArtifact, ObjectID: ref.ArtifactID})
	}
	return ObjectSummary{ObjectKind: ObjectEvidenceRecord, ObjectID: record.EvidenceID, MissionID: record.MissionID, Summary: record.Summary, Refs: DedupeRefs(refs), Metadata: map[string]any{"state": record.State, "evidence_type": record.EvidenceType}}
}

func SummarizeClaim(record researchrecords.ClaimRecord) ObjectSummary {
	var refs []ObjectRef
	for _, id := range record.SupportingEvidenceIDs {
		refs = append(refs, ObjectRef{ObjectKind: ObjectEvidenceRecord, ObjectID: id})
	}
	for _, id := range record.OpposingEvidenceIDs {
		refs = append(refs, ObjectRef{ObjectKind: ObjectEvidenceRecord, ObjectID: id})
	}
	for _, id := range record.DependsOnQuestionIDs {
		refs = append(refs, ObjectRef{ObjectKind: ObjectQuestionRecord, ObjectID: id})
	}
	return ObjectSummary{ObjectKind: ObjectClaimRecord, ObjectID: record.ClaimID, MissionID: record.MissionID, Summary: record.Text, Refs: DedupeRefs(refs), Metadata: map[string]any{"state": record.State, "claim_type": record.ClaimType}}
}

func SummarizeQuestion(record researchrecords.QuestionRecord) ObjectSummary {
	var refs []ObjectRef
	for _, id := range record.RelatedEvidenceIDs {
		refs = append(refs, ObjectRef{ObjectKind: ObjectEvidenceRecord, ObjectID: id})
	}
	for _, id := range record.RelatedClaimIDs {
		refs = append(refs, ObjectRef{ObjectKind: ObjectClaimRecord, ObjectID: id})
	}
	return ObjectSummary{ObjectKind: ObjectQuestionRecord, ObjectID: record.QuestionID, MissionID: record.MissionID, Summary: record.Text, Refs: DedupeRefs(refs), Metadata: map[string]any{"state": record.State, "priority": record.Priority}}
}

func SummarizeOption(record researchrecords.OptionRecord) ObjectSummary {
	var refs []ObjectRef
	for _, id := range record.SupportingClaimIDs {
		refs = append(refs, ObjectRef{ObjectKind: ObjectClaimRecord, ObjectID: id})
	}
	return ObjectSummary{ObjectKind: ObjectOptionRecord, ObjectID: record.OptionID, MissionID: record.MissionID, Summary: firstNonEmpty(record.Title, record.Description, record.OptionID), Refs: refs, Metadata: map[string]any{"state": record.State, "risk_level": record.RiskLevel}}
}

// DedupeRefs filters empty references, removes duplicates, and sorts by kind then
// ID. A nil input produces a non-nil empty slice, matching the prior app contract.
func DedupeRefs(refs []ObjectRef) []ObjectRef {
	seen := map[ObjectRef]bool{}
	out := make([]ObjectRef, 0, len(refs))
	for _, ref := range refs {
		if ref.ObjectKind == "" || ref.ObjectID == "" || seen[ref] {
			continue
		}
		seen[ref] = true
		out = append(out, ref)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && (out[j].ObjectKind < out[j-1].ObjectKind || (out[j].ObjectKind == out[j-1].ObjectKind && out[j].ObjectID < out[j-1].ObjectID)); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

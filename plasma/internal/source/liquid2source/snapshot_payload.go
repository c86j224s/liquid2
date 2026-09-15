package liquid2source

import (
	"encoding/json"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"time"
)

type liquid2SnapshotArtifact struct {
	SchemaVersion string                      `json:"schema_version"`
	Connector     sourcecontract.ConnectorRef `json:"connector"`
	Document      liquid2SnapshotDocument     `json:"document"`
	Contents      []liquid2SnapshotContent    `json:"contents"`
	Reason        string                      `json:"reason,omitempty"`
	Metadata      json.RawMessage             `json:"metadata"`
}

type liquid2SnapshotDocument struct {
	ExternalSourceID string `json:"external_source_id"`
	Title            string `json:"title"`
	SourceURI        string `json:"source_uri,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

type liquid2SnapshotContent struct {
	ContentID string `json:"content_id"`
	Role      string `json:"role"`
	Format    string `json:"format"`
	Language  string `json:"language,omitempty"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
	Content   string `json:"content"`
}

type liquid2SnapshotLocator struct {
	LocatorType      string `json:"locator_type"`
	ArtifactID       string `json:"artifact_id"`
	ExternalSourceID string `json:"external_source_id"`
	SourceURI        string `json:"source_uri,omitempty"`
	ContentID        string `json:"content_id"`
	Role             string `json:"role"`
	Format           string `json:"format"`
	Start            int    `json:"start"`
	End              int    `json:"end"`
}

// BuildSnapshotPayload encodes selected document content and its locators without
// performing connector reads or persistence. Range offsets remain rune-based.
func BuildSnapshotPayload(
	document Liquid2SourceDocument,
	artifactID string,
	reason string,
	ranges []Liquid2ContentRange,
) ([]byte, json.RawMessage, error) {
	selected, locators, err := selectLiquid2SnapshotContents(document, artifactID, ranges)
	if err != nil {
		return nil, nil, err
	}
	updatedAt := ""
	if !document.UpdatedAt.IsZero() {
		updatedAt = document.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	payload := liquid2SnapshotArtifact{
		SchemaVersion: Liquid2SnapshotSchemaV1,
		Connector:     document.Connector,
		Document: liquid2SnapshotDocument{
			ExternalSourceID: document.Connector.ExternalSourceID,
			Title:            document.Title,
			SourceURI:        document.SourceURI,
			UpdatedAt:        updatedAt,
		},
		Contents: selected,
		Reason:   reason,
		Metadata: append(json.RawMessage(nil), document.Metadata...),
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	locatorJSON, err := json.Marshal(locators)
	if err != nil {
		return nil, nil, err
	}
	return content, locatorJSON, nil
}

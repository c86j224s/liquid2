package liquid2source

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
)

// NormalizeSearchLimit applies the existing default and maximum search limits.
func NormalizeSearchLimit(limit int) int {
	if limit <= 0 {
		return defaultSearchLimit
	}
	if limit > maxSearchLimit {
		return maxSearchLimit
	}
	return limit
}

// NormalizeFilters trims filter strings without adding validation or defaults.
func NormalizeFilters(filters Liquid2SourceFilters) Liquid2SourceFilters {
	return Liquid2SourceFilters{
		Status:         strings.TrimSpace(filters.Status),
		Tag:            strings.TrimSpace(filters.Tag),
		Kind:           strings.TrimSpace(filters.Kind),
		RatingMin:      filters.RatingMin,
		IncludeDeleted: filters.IncludeDeleted,
		IncludeTrash:   filters.IncludeTrash,
	}
}

// NormalizeCandidate normalizes connector identity and candidate presentation fields.
func NormalizeCandidate(candidate Liquid2SourceCandidate) (Liquid2SourceCandidate, error) {
	connector, err := NormalizeConnector(candidate.Connector)
	if err != nil {
		return Liquid2SourceCandidate{}, err
	}
	candidate.Connector = connector
	candidate.Title = strings.TrimSpace(candidate.Title)
	candidate.SourceURI = strings.TrimSpace(candidate.SourceURI)
	candidate.Summary = strings.TrimSpace(candidate.Summary)
	candidate.CanSnapshot = true
	return candidate, nil
}

// NormalizeDocument validates the requested document identity and metadata shape.
func NormalizeDocument(document Liquid2SourceDocument, requestedExternalSourceID string) (Liquid2SourceDocument, error) {
	requestedExternalSourceID = strings.TrimSpace(requestedExternalSourceID)
	returnedExternalSourceID := strings.TrimSpace(document.Connector.ExternalSourceID)
	if returnedExternalSourceID == "" {
		document.Connector.ExternalSourceID = requestedExternalSourceID
	} else if returnedExternalSourceID != requestedExternalSourceID {
		return Liquid2SourceDocument{}, fmt.Errorf("%w: liquid2 document id mismatch", producterror.ErrInvalidInput)
	}
	connector, err := NormalizeConnector(document.Connector)
	if err != nil {
		return Liquid2SourceDocument{}, err
	}
	document.Connector = connector
	document.Title = strings.TrimSpace(document.Title)
	if document.Title == "" {
		document.Title = connector.ExternalSourceID
	}
	document.SourceURI = strings.TrimSpace(document.SourceURI)
	if len(document.Metadata) > 0 && !json.Valid(document.Metadata) {
		return Liquid2SourceDocument{}, fmt.Errorf("%w: liquid2 metadata must be valid JSON", producterror.ErrInvalidInput)
	}
	if len(document.Metadata) == 0 {
		document.Metadata = json.RawMessage(`{}`)
	}
	return document, nil
}

// NormalizeConnector fills only Liquid2 connector identity defaults and preserves existing values.
func NormalizeConnector(connector sourcecontract.ConnectorRef) (sourcecontract.ConnectorRef, error) {
	connector = sourcecontract.NormalizeConnector(connector)
	if connector.ConnectorID == "" {
		connector.ConnectorID = Liquid2ConnectorID
	}
	if connector.ConnectorType == "" {
		connector.ConnectorType = Liquid2ConnectorType
	}
	if connector.ConnectorVersion == "" {
		connector.ConnectorVersion = Liquid2HTTPConnectorV1
	}
	if connector.ExternalSourceID == "" {
		return sourcecontract.ConnectorRef{}, fmt.Errorf("%w: liquid2 external source id is required", producterror.ErrInvalidInput)
	}
	if connector.ExternalURI == "" {
		connector.ExternalURI = DocumentURI(connector.ExternalSourceID)
	}
	return connector, nil
}

// DocumentURI returns the canonical Liquid2 document URI with the trimmed identifier.
func DocumentURI(externalSourceID string) string {
	return "liquid2://documents/" + strings.TrimSpace(externalSourceID)
}

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
)

func (s *Service) SearchLiquid2Sources(
	ctx context.Context,
	connector Liquid2SourceConnector,
	req Liquid2SourceSearchRequest,
) (Liquid2SourceSearchResult, error) {
	if connector == nil {
		return Liquid2SourceSearchResult{}, fmt.Errorf("%w: liquid2 connector is required", ErrInvalidInput)
	}
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return Liquid2SourceSearchResult{}, err
	}

	normalized := req
	normalized.MissionID = missionID
	normalized.Query = strings.TrimSpace(req.Query)
	normalized.Cursor = strings.TrimSpace(req.Cursor)
	normalized.Limit = normalizeLiquid2SearchLimit(req.Limit)
	normalized.Filters = normalizeLiquid2Filters(req.Filters)

	result, err := connector.SearchLiquid2Sources(ctx, normalized)
	if err != nil {
		return Liquid2SourceSearchResult{}, err
	}
	result.MissionID = missionID
	for i := range result.Candidates {
		candidate, err := normalizeLiquid2Candidate(result.Candidates[i])
		if err != nil {
			return Liquid2SourceSearchResult{}, err
		}
		result.Candidates[i] = candidate
	}
	return result, nil
}

func (s *Service) SnapshotLiquid2Source(
	ctx context.Context,
	connector Liquid2SourceConnector,
	req SnapshotLiquid2SourceRequest,
) (Liquid2SnapshotResult, error) {
	artifact, snapshot, err := s.buildLiquid2Snapshot(ctx, connector, req)
	if err != nil {
		return Liquid2SnapshotResult{}, err
	}
	if err := s.store.CreateRawArtifact(ctx, artifact); err != nil {
		return Liquid2SnapshotResult{}, err
	}
	if err := s.store.CreateSourceSnapshot(ctx, snapshot); err != nil {
		return Liquid2SnapshotResult{}, err
	}
	return Liquid2SnapshotResult{Artifact: artifact, Snapshot: snapshot}, nil
}

func (s *Service) SnapshotLiquid2SourceWithEvent(
	ctx context.Context,
	connector Liquid2SourceConnector,
	req SnapshotLiquid2SourceWithEventRequest,
) (Liquid2SnapshotWithEventResult, error) {
	artifact, snapshot, err := s.buildLiquid2Snapshot(ctx, connector, req.Snapshot)
	if err != nil {
		return Liquid2SnapshotWithEventResult{}, err
	}
	event, err := buildLedgerEvent(AppendEventRequest{
		EventID:   req.EventID,
		MissionID: snapshot.MissionID,
		EventType: sourceevents.SourceSnapshottedEventType,
		Producer:  req.Producer,
		Payload: sourceevents.BuildConnectorSourceSnapshottedPayload(sourceevents.ConnectorSourceSnapshottedPayloadRequest{
			SnapshotID:  snapshot.SnapshotID,
			ArtifactIDs: snapshot.ArtifactIDs,
			Connector:   sourceEventConnectorRef(snapshot.Connector),
			Reason:      req.Snapshot.Reason,
		}),
	})
	if err != nil {
		return Liquid2SnapshotWithEventResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []LedgerEvent{event},
		RawArtifacts:    []RawArtifact{artifact},
		SourceSnapshots: []SourceSnapshot{snapshot},
	})
	if err != nil {
		return Liquid2SnapshotWithEventResult{}, err
	}
	return Liquid2SnapshotWithEventResult{
		Artifact: artifact,
		Snapshot: snapshot,
		Event:    committed.Events[0],
	}, nil
}

func (s *Service) buildLiquid2Snapshot(
	ctx context.Context,
	connector Liquid2SourceConnector,
	req SnapshotLiquid2SourceRequest,
) (RawArtifact, SourceSnapshot, error) {
	if connector == nil {
		return RawArtifact{}, SourceSnapshot{}, fmt.Errorf("%w: liquid2 connector is required", ErrInvalidInput)
	}
	missionID := strings.TrimSpace(req.MissionID)
	artifactID := strings.TrimSpace(req.ArtifactID)
	snapshotID := strings.TrimSpace(req.SnapshotID)
	externalSourceID := strings.TrimSpace(req.ExternalSourceID)
	if err := validateID("mis_", missionID); err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	if err := validateID("art_", artifactID); err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	if err := validateID("src_", snapshotID); err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	if externalSourceID == "" {
		return RawArtifact{}, SourceSnapshot{}, fmt.Errorf("%w: external source id is required", ErrInvalidInput)
	}

	document, err := connector.ReadLiquid2Source(ctx, Liquid2SourceReadRequest{ExternalSourceID: externalSourceID})
	if err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	document, err = normalizeLiquid2Document(document, externalSourceID)
	if err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}

	content, locators, err := buildLiquid2SnapshotPayload(document, artifactID, strings.TrimSpace(req.Reason), req.ContentRanges)
	if err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	producer, err := liquid2SnapshotProducer(req.Producer)
	if err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	artifact, err := buildRawArtifact(CreateRawArtifactRequest{
		ArtifactID: artifactID,
		MissionID:  missionID,
		MediaType:  Liquid2SnapshotMediaType,
		Filename:   liquid2SnapshotFilename(document.Connector.ExternalSourceID),
		Producer:   producer,
		Content:    content,
	})
	if err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}

	snapshot, err := s.buildSourceSnapshot(ctx, CreateSourceSnapshotRequest{
		SnapshotID:        snapshotID,
		MissionID:         missionID,
		Connector:         document.Connector,
		Title:             document.Title,
		ExternalUpdatedAt: document.UpdatedAt,
		ArtifactIDs:       []string{artifact.ArtifactID},
		ContentHash:       req.ExpectedContentHash,
		Locators:          locators,
	}, []RawArtifact{artifact})
	if err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	return artifact, snapshot, nil
}

func normalizeLiquid2SearchLimit(limit int) int {
	if limit <= 0 {
		return defaultLiquid2SearchLimit
	}
	if limit > maxLiquid2SearchLimit {
		return maxLiquid2SearchLimit
	}
	return limit
}

func normalizeLiquid2Filters(filters Liquid2SourceFilters) Liquid2SourceFilters {
	return Liquid2SourceFilters{
		Status:         strings.TrimSpace(filters.Status),
		Tag:            strings.TrimSpace(filters.Tag),
		Kind:           strings.TrimSpace(filters.Kind),
		RatingMin:      filters.RatingMin,
		IncludeDeleted: filters.IncludeDeleted,
		IncludeTrash:   filters.IncludeTrash,
	}
}

func normalizeLiquid2Candidate(candidate Liquid2SourceCandidate) (Liquid2SourceCandidate, error) {
	connector, err := normalizeLiquid2Connector(candidate.Connector)
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

func normalizeLiquid2Document(document Liquid2SourceDocument, requestedExternalSourceID string) (Liquid2SourceDocument, error) {
	requestedExternalSourceID = strings.TrimSpace(requestedExternalSourceID)
	returnedExternalSourceID := strings.TrimSpace(document.Connector.ExternalSourceID)
	if returnedExternalSourceID == "" {
		document.Connector.ExternalSourceID = requestedExternalSourceID
	} else if returnedExternalSourceID != requestedExternalSourceID {
		return Liquid2SourceDocument{}, fmt.Errorf("%w: liquid2 document id mismatch", ErrInvalidInput)
	}
	connector, err := normalizeLiquid2Connector(document.Connector)
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
		return Liquid2SourceDocument{}, fmt.Errorf("%w: liquid2 metadata must be valid JSON", ErrInvalidInput)
	}
	if len(document.Metadata) == 0 {
		document.Metadata = json.RawMessage(`{}`)
	}
	return document, nil
}

func normalizeLiquid2Connector(connector ConnectorRef) (ConnectorRef, error) {
	connector = normalizeConnector(connector)
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
		return ConnectorRef{}, fmt.Errorf("%w: liquid2 external source id is required", ErrInvalidInput)
	}
	if connector.ExternalURI == "" {
		connector.ExternalURI = liquid2DocumentURI(connector.ExternalSourceID)
	}
	return connector, nil
}

func liquid2SnapshotProducer(producer Producer) (Producer, error) {
	producer.Type = strings.TrimSpace(producer.Type)
	producer.ID = strings.TrimSpace(producer.ID)
	if producer.Type == "" && producer.ID == "" {
		return Producer{Type: "connector", ID: Liquid2ConnectorID}, nil
	}
	if producer.Type != "connector" || producer.ID != Liquid2ConnectorID {
		return Producer{}, fmt.Errorf("%w: liquid2 snapshot producer must be connector/liquid2", ErrInvalidInput)
	}
	return producer, nil
}

func liquid2DocumentURI(externalSourceID string) string {
	return "liquid2://documents/" + strings.TrimSpace(externalSourceID)
}

func mustMarshalJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func liquid2SnapshotFilename(externalSourceID string) string {
	externalSourceID = strings.TrimSpace(externalSourceID)
	if externalSourceID == "" {
		return "liquid2-source.json"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_")
	return "liquid2-" + replacer.Replace(externalSourceID) + ".json"
}

type liquid2SnapshotArtifact struct {
	SchemaVersion string                   `json:"schema_version"`
	Connector     ConnectorRef             `json:"connector"`
	Document      liquid2SnapshotDocument  `json:"document"`
	Contents      []liquid2SnapshotContent `json:"contents"`
	Reason        string                   `json:"reason,omitempty"`
	Metadata      json.RawMessage          `json:"metadata"`
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
	ContentID        string `json:"content_id"`
	Role             string `json:"role"`
	Format           string `json:"format"`
	Start            int    `json:"start"`
	End              int    `json:"end"`
}

func buildLiquid2SnapshotPayload(
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

func selectLiquid2SnapshotContents(
	document Liquid2SourceDocument,
	artifactID string,
	ranges []Liquid2ContentRange,
) ([]liquid2SnapshotContent, []liquid2SnapshotLocator, error) {
	if len(document.Contents) == 0 {
		return nil, nil, fmt.Errorf("%w: liquid2 source content is required", ErrInvalidInput)
	}
	byID, err := liquid2ContentsByID(document.Contents)
	if err != nil {
		return nil, nil, err
	}
	if len(ranges) == 0 {
		selected := make([]liquid2SnapshotContent, 0, len(document.Contents))
		locators := make([]liquid2SnapshotLocator, 0, len(document.Contents))
		for _, content := range document.Contents {
			item, err := fullLiquid2SnapshotContent(content)
			if err != nil {
				return nil, nil, err
			}
			selected = append(selected, item)
			locators = append(locators, liquid2Locator(document, artifactID, item))
		}
		return selected, locators, nil
	}

	selected := make([]liquid2SnapshotContent, 0, len(ranges))
	locators := make([]liquid2SnapshotLocator, 0, len(ranges))
	for _, requestedRange := range ranges {
		contentID := strings.TrimSpace(requestedRange.ContentID)
		content, ok := byID[contentID]
		if !ok || contentID == "" {
			return nil, nil, fmt.Errorf("%w: liquid2 content range references unknown content", ErrInvalidInput)
		}
		item, err := rangedLiquid2SnapshotContent(content, requestedRange.Start, requestedRange.End)
		if err != nil {
			return nil, nil, err
		}
		selected = append(selected, item)
		locators = append(locators, liquid2Locator(document, artifactID, item))
	}
	return selected, locators, nil
}

func liquid2ContentsByID(contents []Liquid2SourceContent) (map[string]Liquid2SourceContent, error) {
	byID := make(map[string]Liquid2SourceContent, len(contents))
	for _, content := range contents {
		contentID := strings.TrimSpace(content.ContentID)
		if contentID == "" {
			return nil, fmt.Errorf("%w: liquid2 content id is required", ErrInvalidInput)
		}
		if strings.TrimSpace(content.Content) == "" {
			return nil, fmt.Errorf("%w: liquid2 content body is required", ErrInvalidInput)
		}
		if _, exists := byID[contentID]; exists {
			return nil, fmt.Errorf("%w: duplicate liquid2 content id", ErrInvalidInput)
		}
		byID[contentID] = content
	}
	return byID, nil
}

func fullLiquid2SnapshotContent(content Liquid2SourceContent) (liquid2SnapshotContent, error) {
	if strings.TrimSpace(content.ContentID) == "" || strings.TrimSpace(content.Content) == "" {
		return liquid2SnapshotContent{}, fmt.Errorf("%w: liquid2 content id and body are required", ErrInvalidInput)
	}
	return liquid2SnapshotContent{
		ContentID: strings.TrimSpace(content.ContentID),
		Role:      strings.TrimSpace(content.Role),
		Format:    strings.TrimSpace(content.Format),
		Language:  strings.TrimSpace(content.Language),
		Start:     0,
		End:       len([]rune(content.Content)),
		Content:   content.Content,
	}, nil
}

func rangedLiquid2SnapshotContent(content Liquid2SourceContent, start int, end int) (liquid2SnapshotContent, error) {
	if strings.TrimSpace(content.ContentID) == "" || strings.TrimSpace(content.Content) == "" {
		return liquid2SnapshotContent{}, fmt.Errorf("%w: liquid2 content id and body are required", ErrInvalidInput)
	}
	runes := []rune(content.Content)
	if start < 0 || end <= start || end > len(runes) {
		return liquid2SnapshotContent{}, fmt.Errorf("%w: invalid liquid2 content range", ErrInvalidInput)
	}
	return liquid2SnapshotContent{
		ContentID: strings.TrimSpace(content.ContentID),
		Role:      strings.TrimSpace(content.Role),
		Format:    strings.TrimSpace(content.Format),
		Language:  strings.TrimSpace(content.Language),
		Start:     start,
		End:       end,
		Content:   string(runes[start:end]),
	}, nil
}

func liquid2Locator(document Liquid2SourceDocument, artifactID string, content liquid2SnapshotContent) liquid2SnapshotLocator {
	return liquid2SnapshotLocator{
		LocatorType:      "liquid2_content_range",
		ArtifactID:       artifactID,
		ExternalSourceID: document.Connector.ExternalSourceID,
		ContentID:        content.ContentID,
		Role:             content.Role,
		Format:           content.Format,
		Start:            content.Start,
		End:              content.End,
	}
}

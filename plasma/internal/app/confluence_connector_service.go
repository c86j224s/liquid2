package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
)

func (s *Service) SearchConfluenceSources(
	ctx context.Context,
	connector ConfluenceSourceConnector,
	req ConfluenceSourceSearchRequest,
) (ConfluenceSourceSearchResult, error) {
	if connector == nil {
		return ConfluenceSourceSearchResult{}, fmt.Errorf("%w: confluence connector is required", ErrInvalidInput)
	}
	normalized, err := normalizeConfluenceSearchRequest(req)
	if err != nil {
		return ConfluenceSourceSearchResult{}, err
	}
	result, err := connector.SearchConfluenceSources(ctx, normalized)
	if err != nil {
		return ConfluenceSourceSearchResult{}, err
	}
	result.MissionID = normalized.MissionID
	result.CloudID = normalized.CloudID
	result.NextCursor = strings.TrimSpace(result.NextCursor)
	for i := range result.Candidates {
		candidate, err := normalizeConfluenceCandidate(result.Candidates[i], normalized.CloudID)
		if err != nil {
			return ConfluenceSourceSearchResult{}, err
		}
		result.Candidates[i] = candidate
	}
	return result, nil
}

func (s *Service) ListConfluenceSpaces(
	ctx context.Context,
	connector ConfluenceBrowserConnector,
	req ConfluenceSpaceListRequest,
) (ConfluenceSpaceListResult, error) {
	if connector == nil {
		return ConfluenceSpaceListResult{}, fmt.Errorf("%w: confluence browser connector is required", ErrInvalidInput)
	}
	normalized, err := normalizeConfluenceSpaceListRequest(req)
	if err != nil {
		return ConfluenceSpaceListResult{}, err
	}
	result, err := connector.ListConfluenceSpaces(ctx, normalized)
	if err != nil {
		return ConfluenceSpaceListResult{}, err
	}
	result.MissionID = normalized.MissionID
	result.CloudID = normalized.CloudID
	result.NextCursor = strings.TrimSpace(result.NextCursor)
	for i := range result.Spaces {
		result.Spaces[i] = normalizeConfluenceSpaceSummary(result.Spaces[i], normalized.CloudID)
	}
	return result, nil
}

func (s *Service) ListConfluenceSpacePages(
	ctx context.Context,
	connector ConfluenceBrowserConnector,
	req ConfluenceSpacePagesRequest,
) (ConfluencePageListResult, error) {
	if connector == nil {
		return ConfluencePageListResult{}, fmt.Errorf("%w: confluence browser connector is required", ErrInvalidInput)
	}
	normalized, err := normalizeConfluenceSpacePagesRequest(req)
	if err != nil {
		return ConfluencePageListResult{}, err
	}
	result, err := connector.ListConfluenceSpacePages(ctx, normalized)
	if err != nil {
		return ConfluencePageListResult{}, err
	}
	result.MissionID = normalized.MissionID
	result.CloudID = normalized.CloudID
	result.NextCursor = strings.TrimSpace(result.NextCursor)
	for i := range result.Pages {
		result.Pages[i] = normalizeConfluencePageSummary(result.Pages[i], normalized.CloudID)
	}
	return result, nil
}

func (s *Service) ListConfluencePageChildren(
	ctx context.Context,
	connector ConfluenceBrowserConnector,
	req ConfluencePageChildrenRequest,
) (ConfluencePageListResult, error) {
	if connector == nil {
		return ConfluencePageListResult{}, fmt.Errorf("%w: confluence browser connector is required", ErrInvalidInput)
	}
	normalized, err := normalizeConfluencePageChildrenRequest(req)
	if err != nil {
		return ConfluencePageListResult{}, err
	}
	result, err := connector.ListConfluencePageChildren(ctx, normalized)
	if err != nil {
		return ConfluencePageListResult{}, err
	}
	result.MissionID = normalized.MissionID
	result.CloudID = normalized.CloudID
	result.NextCursor = strings.TrimSpace(result.NextCursor)
	for i := range result.Pages {
		result.Pages[i] = normalizeConfluencePageSummary(result.Pages[i], normalized.CloudID)
	}
	return result, nil
}

func (s *Service) PreviewConfluenceSource(
	ctx context.Context,
	connector ConfluenceSourceConnector,
	req ConfluenceSourcePreviewRequest,
) (ConfluenceSourcePreviewResult, error) {
	if connector == nil {
		return ConfluenceSourcePreviewResult{}, fmt.Errorf("%w: confluence connector is required", ErrInvalidInput)
	}
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return ConfluenceSourcePreviewResult{}, err
	}
	cloudID := strings.TrimSpace(req.CloudID)
	pageID := strings.TrimSpace(req.PageID)
	if cloudID == "" || pageID == "" {
		return ConfluenceSourcePreviewResult{}, fmt.Errorf("%w: confluence cloud id and page id are required", ErrInvalidInput)
	}
	page, err := connector.ReadConfluenceSource(ctx, ConfluenceSourceReadRequest{CloudID: cloudID, PageID: pageID})
	if err != nil {
		return ConfluenceSourcePreviewResult{}, err
	}
	page, err = normalizeConfluencePage(page, cloudID, pageID, req.ExpectedVersion)
	if err != nil {
		return ConfluenceSourcePreviewResult{}, err
	}
	maxBytes := normalizeConfluenceMaxBodyBytes(req.MaxBodyBytes)
	bodyBytes := int64(len([]byte(page.BodyStorage)))
	preview, truncated := confluencePreviewText(page.PlainText, req.PreviewRunes)
	result := ConfluenceSourcePreviewResult{
		MissionID:        missionID,
		CandidateKind:    "confluence_page_preview_result",
		Page:             confluencePreviewPage(page),
		PreviewText:      preview,
		PreviewTruncated: truncated,
		BodyBytes:        bodyBytes,
		MaxBodyBytes:     maxBytes,
		FullBodyTooLarge: bodyBytes > maxBytes,
		RangeOptions:     confluenceRangeOptions(page.PlainText, maxBytes),
	}
	if !result.FullBodyTooLarge && len(result.RangeOptions) > 4 {
		result.RangeOptions = result.RangeOptions[:4]
	}
	return result, nil
}

func (s *Service) SnapshotConfluenceSource(
	ctx context.Context,
	connector ConfluenceSourceConnector,
	req SnapshotConfluenceSourceRequest,
) (ConfluenceSnapshotResult, error) {
	artifact, snapshot, err := s.buildConfluenceSnapshot(ctx, connector, req)
	if err != nil {
		return ConfluenceSnapshotResult{}, err
	}
	if err := s.store.CreateRawArtifact(ctx, artifact); err != nil {
		return ConfluenceSnapshotResult{}, err
	}
	if err := s.store.CreateSourceSnapshot(ctx, snapshot); err != nil {
		return ConfluenceSnapshotResult{}, err
	}
	return ConfluenceSnapshotResult{Artifact: artifact, Snapshot: snapshot}, nil
}

func (s *Service) SnapshotConfluenceSourceWithEvent(
	ctx context.Context,
	connector ConfluenceSourceConnector,
	req SnapshotConfluenceSourceWithEventRequest,
) (ConfluenceSnapshotWithEventResult, error) {
	artifact, snapshot, err := s.buildConfluenceSnapshot(ctx, connector, req.Snapshot)
	if err != nil {
		return ConfluenceSnapshotWithEventResult{}, err
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
		return ConfluenceSnapshotWithEventResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []LedgerEvent{event},
		RawArtifacts:    []RawArtifact{artifact},
		SourceSnapshots: []SourceSnapshot{snapshot},
	})
	if err != nil {
		return ConfluenceSnapshotWithEventResult{}, err
	}
	return ConfluenceSnapshotWithEventResult{Artifact: artifact, Snapshot: snapshot, Event: committed.Events[0]}, nil
}

func (s *Service) buildConfluenceSnapshot(
	ctx context.Context,
	connector ConfluenceSourceConnector,
	req SnapshotConfluenceSourceRequest,
) (RawArtifact, SourceSnapshot, error) {
	if connector == nil {
		return RawArtifact{}, SourceSnapshot{}, fmt.Errorf("%w: confluence connector is required", ErrInvalidInput)
	}
	missionID := strings.TrimSpace(req.MissionID)
	artifactID := strings.TrimSpace(req.ArtifactID)
	snapshotID := strings.TrimSpace(req.SnapshotID)
	cloudID := strings.TrimSpace(req.CloudID)
	pageID := strings.TrimSpace(req.PageID)
	if err := validateID("mis_", missionID); err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	if err := validateID("art_", artifactID); err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	if err := validateID("src_", snapshotID); err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	if cloudID == "" || pageID == "" {
		return RawArtifact{}, SourceSnapshot{}, fmt.Errorf("%w: confluence cloud id and page id are required", ErrInvalidInput)
	}
	if err := validateConfluenceExpectedVersion(req.ExpectedVersion); err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}

	page, err := connector.ReadConfluenceSource(ctx, ConfluenceSourceReadRequest{CloudID: cloudID, PageID: pageID})
	if err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	if page.Title = strings.TrimSpace(page.Title); page.Title == "" {
		page.Title = strings.TrimSpace(req.Title)
	}
	page, err = normalizeConfluencePage(page, cloudID, pageID, req.ExpectedVersion)
	if err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	if confluenceRangeSelected(req.Range) {
		if err := validateConfluenceSelectedRange(page, req.Range, req.MaxBodyBytes); err != nil {
			return RawArtifact{}, SourceSnapshot{}, err
		}
	} else {
		if err := validateConfluenceSnapshotBodySize(page, req.MaxBodyBytes); err != nil {
			return RawArtifact{}, SourceSnapshot{}, err
		}
	}
	content, locators, err := buildConfluenceSnapshotPayload(page, artifactID, strings.TrimSpace(req.Reason), req.Range)
	if err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	producer, err := confluenceSnapshotProducer(req.Producer)
	if err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	artifact, err := buildRawArtifact(CreateRawArtifactRequest{
		ArtifactID: artifactID,
		MissionID:  missionID,
		MediaType:  ConfluenceSnapshotMediaType,
		Filename:   confluenceSnapshotFilename(page.Connector.ExternalSourceID),
		Producer:   producer,
		Content:    content,
	})
	if err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	title := page.Title
	if confluenceRangeSelected(req.Range) {
		title = fmt.Sprintf("%s (Confluence range %d-%d)", page.Title, req.Range.Start, req.Range.End)
	}
	snapshot, err := s.buildSourceSnapshot(ctx, CreateSourceSnapshotRequest{
		SnapshotID:        snapshotID,
		MissionID:         missionID,
		Connector:         page.Connector,
		Title:             title,
		ExternalUpdatedAt: page.UpdatedAt,
		ArtifactIDs:       []string{artifact.ArtifactID},
		ContentHash:       req.ExpectedContentHash,
		Locators:          locators,
	}, []RawArtifact{artifact})
	if err != nil {
		return RawArtifact{}, SourceSnapshot{}, err
	}
	return artifact, snapshot, nil
}

func (s *Service) CheckConfluenceSourceUpdateWithEvent(
	ctx context.Context,
	connector ConfluenceSourceConnector,
	req CheckConfluenceSourceUpdateRequest,
) (ConfluenceUpdateCheckResult, error) {
	result, err := s.CheckConfluenceSourceUpdate(ctx, connector, req)
	if err != nil {
		return ConfluenceUpdateCheckResult{}, err
	}
	eventType := ConfluenceUpdateCurrentEvent
	if result.UpdateAvailable {
		eventType = ConfluenceUpdateAvailableEvent
	}
	event, err := buildLedgerEvent(AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(req.MissionID),
		EventType: eventType,
		Producer:  normalizeConfluenceUpdateProducer(req.Producer),
		Payload: mustMarshalJSON(map[string]any{
			"old_snapshot_id":  result.Snapshot.SnapshotID,
			"cloud_id":         result.SnapshotIdentity().CloudID,
			"page_id":          result.SnapshotIdentity().PageID,
			"old_version":      result.CurrentVersion,
			"new_version":      result.LatestVersion,
			"new_title":        result.LatestTitle,
			"new_updated_at":   optionalTimeString(result.LatestUpdatedAt),
			"update_available": result.UpdateAvailable,
			"checked_at":       time.Now().UTC().Format(time.RFC3339Nano),
		}),
	})
	if err != nil {
		return ConfluenceUpdateCheckResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{Events: []LedgerEvent{event}})
	if err != nil {
		return ConfluenceUpdateCheckResult{}, err
	}
	result.Event = committed.Events[0]
	return result, nil
}

func (s *Service) CheckConfluenceSourceUpdate(
	ctx context.Context,
	connector ConfluenceSourceConnector,
	req CheckConfluenceSourceUpdateRequest,
) (ConfluenceUpdateCheckResult, error) {
	if connector == nil {
		return ConfluenceUpdateCheckResult{}, fmt.Errorf("%w: confluence connector is required", ErrInvalidInput)
	}
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return ConfluenceUpdateCheckResult{}, err
	}
	snapshot, identity, err := s.activeConfluenceSnapshotIdentity(ctx, missionID, req.SnapshotID)
	if err != nil {
		return ConfluenceUpdateCheckResult{}, err
	}
	latest, err := readConfluenceVersion(ctx, connector, identity)
	if err != nil {
		return ConfluenceUpdateCheckResult{}, err
	}
	latest, err = normalizeConfluenceVersion(latest, identity.CloudID, identity.PageID)
	if err != nil {
		return ConfluenceUpdateCheckResult{}, err
	}
	currentVersion := confluenceSnapshotVersion(snapshot)
	return ConfluenceUpdateCheckResult{
		Snapshot:         snapshot,
		CurrentVersion:   currentVersion,
		CurrentTitle:     snapshot.Title,
		CurrentUpdatedAt: snapshot.ExternalUpdatedAt,
		LatestPageID:     latest.PageID,
		LatestSpaceID:    latest.SpaceID,
		LatestSpaceKey:   latest.SpaceKey,
		LatestWebURL:     latest.WebURL,
		LatestVersion:    latest.Version,
		LatestTitle:      latest.Title,
		LatestUpdatedAt:  latest.UpdatedAt,
		UpdateAvailable:  latest.Version > 0 && currentVersion > 0 && latest.Version > currentVersion,
	}, nil
}

func (s *Service) PreviewConfluenceSourceUpdate(
	ctx context.Context,
	connector ConfluenceSourceConnector,
	req ConfluenceUpdatePreviewRequest,
) (ConfluenceUpdatePreviewResult, error) {
	if connector == nil {
		return ConfluenceUpdatePreviewResult{}, fmt.Errorf("%w: confluence connector is required", ErrInvalidInput)
	}
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return ConfluenceUpdatePreviewResult{}, err
	}
	previous, identity, err := s.activeConfluenceSnapshotIdentity(ctx, missionID, req.SnapshotID)
	if err != nil {
		return ConfluenceUpdatePreviewResult{}, err
	}
	page, err := connector.ReadConfluenceSource(ctx, ConfluenceSourceReadRequest{CloudID: identity.CloudID, PageID: identity.PageID})
	if err != nil {
		return ConfluenceUpdatePreviewResult{}, err
	}
	page, err = normalizeConfluencePage(page, identity.CloudID, identity.PageID, req.ExpectedVersion)
	if err != nil {
		return ConfluenceUpdatePreviewResult{}, err
	}
	maxBytes := normalizeConfluenceMaxBodyBytes(req.MaxBodyBytes)
	bodyBytes := int64(len([]byte(page.BodyStorage)))
	preview, truncated := confluencePreviewText(page.PlainText, req.PreviewRunes)
	previousRange, hasPreviousRange := confluenceRangeFromSnapshot(previous)
	return ConfluenceUpdatePreviewResult{
		Snapshot:               previous,
		OldPage:                confluencePreviewPageFromSnapshot(previous, identity),
		NewPage:                confluencePreviewPage(page),
		UpdateAvailable:        page.Version > confluenceSnapshotVersion(previous),
		PreviewText:            preview,
		PreviewTruncated:       truncated,
		BodyBytes:              bodyBytes,
		MaxBodyBytes:           maxBytes,
		FullBodyTooLarge:       bodyBytes > maxBytes,
		RangeOptions:           confluenceRangeOptions(page.PlainText, maxBytes),
		RequiresRangeReselect:  hasPreviousRange,
		PreviousRangeSelection: previousRange,
	}, nil
}

func (s *Service) UpdateConfluenceSourceWithEvent(
	ctx context.Context,
	connector ConfluenceSourceConnector,
	req UpdateConfluenceSourceRequest,
) (ConfluenceUpdateResult, error) {
	if connector == nil {
		return ConfluenceUpdateResult{}, fmt.Errorf("%w: confluence connector is required", ErrInvalidInput)
	}
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return ConfluenceUpdateResult{}, err
	}
	previous, identity, err := s.activeConfluenceSnapshotIdentity(ctx, missionID, req.PreviousSnapshotID)
	if err != nil {
		return ConfluenceUpdateResult{}, err
	}
	if _, partial := confluenceRangeFromSnapshot(previous); partial && !confluenceRangeSelected(req.Range) {
		return ConfluenceUpdateResult{}, NewConfluenceValidationError(
			ConfluenceErrorCodeVersionDrift,
			"부분 Confluence 소스는 업데이트 전에 새 페이지에서 범위를 다시 선택해야 합니다.",
		)
	}
	artifact, snapshot, err := s.buildConfluenceSnapshot(ctx, connector, SnapshotConfluenceSourceRequest{
		MissionID:       missionID,
		ArtifactID:      strings.TrimSpace(req.ArtifactID),
		SnapshotID:      strings.TrimSpace(req.SnapshotID),
		CloudID:         identity.CloudID,
		PageID:          identity.PageID,
		ExpectedVersion: req.ExpectedVersion,
		MaxBodyBytes:    req.MaxBodyBytes,
		Range:           req.Range,
		Reason:          req.Reason,
	})
	if err != nil {
		return ConfluenceUpdateResult{}, err
	}
	oldVersion := confluenceSnapshotVersion(previous)
	newVersion := confluenceSnapshotVersion(snapshot)
	if oldVersion > 0 && newVersion > 0 && newVersion <= oldVersion {
		return ConfluenceUpdateResult{}, fmt.Errorf("%w: confluence source is not newer than the previous snapshot", ErrInvalidInput)
	}
	snapshotEvent, err := buildLedgerEvent(AppendEventRequest{
		EventID:   strings.TrimSpace(req.SnapshotEventID),
		MissionID: missionID,
		EventType: sourceevents.SourceSnapshottedEventType,
		Producer:  normalizeConfluenceUpdateProducer(req.Producer),
		Payload: sourceevents.BuildConfluenceUpdateSourceSnapshottedPayload(sourceevents.ConfluenceUpdateSourceSnapshottedPayloadRequest{
			SnapshotID:         snapshot.SnapshotID,
			ArtifactIDs:        snapshot.ArtifactIDs,
			Connector:          sourceEventConnectorRef(snapshot.Connector),
			Reason:             req.Reason,
			PreviousSnapshotID: previous.SnapshotID,
			PreviousVersion:    oldVersion,
			CloudID:            identity.CloudID,
			PageID:             identity.PageID,
		}),
	})
	if err != nil {
		return ConfluenceUpdateResult{}, err
	}
	updateEvent, err := buildLedgerEvent(AppendEventRequest{
		EventID:   strings.TrimSpace(req.UpdateEventID),
		MissionID: missionID,
		EventType: ConfluenceUpdatedEvent,
		Producer:  normalizeConfluenceUpdateProducer(req.Producer),
		Payload: mustMarshalJSON(map[string]any{
			"old_snapshot_id": previous.SnapshotID,
			"new_snapshot_id": snapshot.SnapshotID,
			"artifact_ids":    snapshot.ArtifactIDs,
			"cloud_id":        identity.CloudID,
			"page_id":         identity.PageID,
			"old_version":     oldVersion,
			"new_version":     newVersion,
			"reason":          strings.TrimSpace(req.Reason),
		}),
	})
	if err != nil {
		return ConfluenceUpdateResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []LedgerEvent{snapshotEvent, updateEvent},
		RawArtifacts:    []RawArtifact{artifact},
		SourceSnapshots: []SourceSnapshot{snapshot},
	})
	if err != nil {
		return ConfluenceUpdateResult{}, err
	}
	return ConfluenceUpdateResult{
		PreviousSnapshot: previous,
		Artifact:         artifact,
		Snapshot:         snapshot,
		SnapshotEvent:    committed.Events[0],
		UpdateEvent:      committed.Events[1],
	}, nil
}

type confluenceSourceIdentity struct {
	CloudID string
	PageID  string
}

func (result ConfluenceUpdateCheckResult) SnapshotIdentity() confluenceSourceIdentity {
	identity, _ := confluenceIdentityFromSnapshot(result.Snapshot)
	return identity
}

func (s *Service) confluenceSnapshotIdentity(ctx context.Context, missionID string, snapshotID string) (SourceSnapshot, confluenceSourceIdentity, error) {
	snapshot, err := s.GetSourceSnapshot(ctx, strings.TrimSpace(snapshotID))
	if err != nil {
		return SourceSnapshot{}, confluenceSourceIdentity{}, err
	}
	if snapshot.MissionID != missionID {
		return SourceSnapshot{}, confluenceSourceIdentity{}, fmt.Errorf("%w: source belongs to another mission", ErrInvalidInput)
	}
	if snapshot.Connector.ConnectorID != ConfluenceConnectorID || snapshot.Connector.ConnectorType != ConfluenceConnectorType {
		return SourceSnapshot{}, confluenceSourceIdentity{}, fmt.Errorf("%w: source is not a confluence snapshot", ErrInvalidInput)
	}
	identity, err := confluenceIdentityFromSnapshot(snapshot)
	if err != nil {
		return SourceSnapshot{}, confluenceSourceIdentity{}, err
	}
	return snapshot, identity, nil
}

func (s *Service) activeConfluenceSnapshotIdentity(ctx context.Context, missionID string, snapshotID string) (SourceSnapshot, confluenceSourceIdentity, error) {
	snapshot, identity, err := s.confluenceSnapshotIdentity(ctx, missionID, snapshotID)
	if err != nil {
		return SourceSnapshot{}, confluenceSourceIdentity{}, err
	}
	if snapshot.State.Removed {
		return SourceSnapshot{}, confluenceSourceIdentity{}, fmt.Errorf("%w: confluence source is removed", ErrInvalidInput)
	}
	if snapshot.State.Superseded {
		return SourceSnapshot{}, confluenceSourceIdentity{}, fmt.Errorf("%w: confluence source has been superseded", ErrInvalidInput)
	}
	current, ok, err := s.currentConfluenceSnapshotForIdentity(ctx, missionID, identity)
	if err != nil {
		return SourceSnapshot{}, confluenceSourceIdentity{}, err
	}
	if ok && current.SnapshotID != snapshot.SnapshotID {
		return SourceSnapshot{}, confluenceSourceIdentity{}, fmt.Errorf("%w: confluence source is not the current active snapshot", ErrInvalidInput)
	}
	return snapshot, identity, nil
}

func (s *Service) currentConfluenceSnapshotForIdentity(ctx context.Context, missionID string, identity confluenceSourceIdentity) (SourceSnapshot, bool, error) {
	sources, err := s.ListSourceSnapshotsWithState(ctx, ListSourceSnapshotsRequest{MissionID: missionID, IncludeRemoved: true})
	if err != nil {
		return SourceSnapshot{}, false, err
	}
	current := SourceSnapshot{}
	for _, source := range sources {
		if source.Connector.ConnectorID != ConfluenceConnectorID || source.Connector.ConnectorType != ConfluenceConnectorType {
			continue
		}
		sourceIdentity, err := confluenceIdentityFromSnapshot(source)
		if err != nil || sourceIdentity != identity {
			continue
		}
		if source.State.Removed || source.State.Superseded {
			continue
		}
		if current.SnapshotID == "" || confluenceSnapshotNewerThan(source, current) {
			current = source
		}
	}
	return current, current.SnapshotID != "", nil
}

func confluenceIdentityFromSnapshot(snapshot SourceSnapshot) (confluenceSourceIdentity, error) {
	if identity, ok := confluenceIdentityFromLocators(snapshot.Locators); ok {
		return identity, nil
	}
	externalID := strings.TrimSpace(snapshot.Connector.ExternalSourceID)
	for _, sep := range []string{":", "/"} {
		parts := strings.Split(externalID, sep)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != "" {
			return confluenceSourceIdentity{CloudID: strings.TrimSpace(parts[0]), PageID: strings.TrimSpace(parts[1])}, nil
		}
	}
	return confluenceSourceIdentity{}, fmt.Errorf("%w: confluence cloud id and page id are required", ErrInvalidInput)
}

func confluenceIdentityFromLocators(raw json.RawMessage) (confluenceSourceIdentity, bool) {
	var locators []struct {
		CloudID string `json:"cloud_id"`
		PageID  string `json:"page_id"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &locators) != nil {
		return confluenceSourceIdentity{}, false
	}
	for _, locator := range locators {
		cloudID := strings.TrimSpace(locator.CloudID)
		pageID := strings.TrimSpace(locator.PageID)
		if cloudID != "" && pageID != "" {
			return confluenceSourceIdentity{CloudID: cloudID, PageID: pageID}, true
		}
	}
	return confluenceSourceIdentity{}, false
}

func readConfluenceVersion(ctx context.Context, connector ConfluenceSourceConnector, identity confluenceSourceIdentity) (ConfluenceSourceVersion, error) {
	req := ConfluenceSourceReadRequest{CloudID: identity.CloudID, PageID: identity.PageID}
	if versionConnector, ok := connector.(ConfluenceSourceVersionConnector); ok {
		return versionConnector.GetConfluenceSourceVersion(ctx, req)
	}
	page, err := connector.ReadConfluenceSource(ctx, req)
	if err != nil {
		return ConfluenceSourceVersion{}, err
	}
	return ConfluenceSourceVersion{
		Connector: page.Connector,
		CloudID:   page.CloudID,
		SiteURL:   page.SiteURL,
		PageID:    page.PageID,
		SpaceID:   page.SpaceID,
		SpaceKey:  page.SpaceKey,
		Title:     page.Title,
		WebURL:    page.WebURL,
		Version:   page.Version,
		UpdatedAt: page.UpdatedAt,
	}, nil
}

func normalizeConfluenceVersion(version ConfluenceSourceVersion, cloudID string, pageID string) (ConfluenceSourceVersion, error) {
	version.CloudID = strings.TrimSpace(version.CloudID)
	if version.CloudID == "" {
		version.CloudID = cloudID
	} else if version.CloudID != cloudID {
		return ConfluenceSourceVersion{}, NewConfluenceValidationError(
			ConfluenceErrorCodeCloudMismatch,
			"Confluence cloud id가 저장된 소스와 일치하지 않습니다. 연결 site를 확인하세요.",
		)
	}
	version.PageID = strings.TrimSpace(version.PageID)
	if version.PageID == "" {
		version.PageID = pageID
	} else if version.PageID != pageID {
		return ConfluenceSourceVersion{}, NewConfluenceValidationError(
			ConfluenceErrorCodePageMismatch,
			"Confluence page id가 저장된 소스와 일치하지 않습니다. 페이지를 다시 확인하세요.",
		)
	}
	version.Title = strings.TrimSpace(version.Title)
	if version.Title == "" {
		version.Title = version.PageID
	}
	return version, nil
}

func confluenceSnapshotVersion(snapshot SourceSnapshot) int {
	if parsed, err := strconv.Atoi(strings.TrimSpace(snapshot.Connector.ExternalVersion)); err == nil {
		return parsed
	}
	return 0
}

func confluenceSnapshotNewerThan(candidate SourceSnapshot, current SourceSnapshot) bool {
	candidateVersion := confluenceSnapshotVersion(candidate)
	currentVersion := confluenceSnapshotVersion(current)
	if candidateVersion != currentVersion {
		return candidateVersion > currentVersion
	}
	if !candidate.ExternalUpdatedAt.Equal(current.ExternalUpdatedAt) {
		return candidate.ExternalUpdatedAt.After(current.ExternalUpdatedAt)
	}
	if !candidate.CapturedAt.Equal(current.CapturedAt) {
		return candidate.CapturedAt.After(current.CapturedAt)
	}
	return candidate.SnapshotID > current.SnapshotID
}

func validateConfluenceSnapshotBodySize(page ConfluenceSourcePage, maxBytes int64) error {
	maxBytes = normalizeConfluenceMaxBodyBytes(maxBytes)
	bodyBytes := int64(len([]byte(page.BodyStorage)))
	if bodyBytes > maxBytes {
		return NewConfluenceValidationError(
			ConfluenceErrorCodeTooLarge,
			fmt.Sprintf("Confluence 페이지가 너무 큽니다. 전체 %d bytes가 한도 %d bytes를 넘었습니다. 미리보기에서 범위를 선택하세요.", bodyBytes, maxBytes),
		)
	}
	return nil
}

func validateConfluenceSelectedRange(page ConfluenceSourcePage, selection ConfluenceRangeSelection, maxBytes int64) error {
	body, err := confluenceRangeBody(page.PlainText, selection)
	if err != nil {
		return err
	}
	maxBytes = normalizeConfluenceMaxBodyBytes(maxBytes)
	bodyBytes := int64(len([]byte(body.Content)))
	if bodyBytes > maxBytes {
		return NewConfluenceValidationError(
			ConfluenceErrorCodeTooLarge,
			fmt.Sprintf("선택한 Confluence 범위가 너무 큽니다. 선택 범위 %d bytes가 한도 %d bytes를 넘었습니다.", bodyBytes, maxBytes),
		)
	}
	return nil
}

func normalizeConfluenceMaxBodyBytes(maxBytes int64) int64 {
	if maxBytes <= 0 {
		return DefaultConfluenceMaxBodyBytes
	}
	return maxBytes
}

func confluencePreviewText(plainText string, limit int) (string, bool) {
	if limit <= 0 {
		limit = 1200
	}
	runes := []rune(plainText)
	if len(runes) <= limit {
		return plainText, false
	}
	return string(runes[:limit]), true
}

func confluenceRangeOptions(plainText string, maxBytes int64) []ConfluenceRangeOption {
	runes := []rune(plainText)
	if len(runes) == 0 {
		return nil
	}
	maxBytes = normalizeConfluenceMaxBodyBytes(maxBytes)
	const maxRunesPerOption = 4000
	options := []ConfluenceRangeOption{}
	for start := 0; start < len(runes) && len(options) < 20; {
		end := start
		bytes := int64(0)
		for end < len(runes) && end-start < maxRunesPerOption {
			nextBytes := int64(len(string(runes[end])))
			if end > start && bytes+nextBytes > maxBytes {
				break
			}
			if end == start && nextBytes > maxBytes {
				start++
				break
			}
			bytes += nextBytes
			end++
		}
		if end <= start {
			continue
		}
		options = append(options, ConfluenceRangeOption{
			ContentID: "plain_text",
			Label:     fmt.Sprintf("문자 %d-%d", start, end),
			Start:     start,
			End:       end,
			RuneCount: end - start,
		})
		start = end
	}
	return options
}

func confluencePreviewPage(page ConfluenceSourcePage) ConfluenceSourcePreviewPage {
	return ConfluenceSourcePreviewPage{
		CloudID:   page.CloudID,
		SiteURL:   page.SiteURL,
		PageID:    page.PageID,
		SpaceID:   page.SpaceID,
		SpaceKey:  page.SpaceKey,
		Title:     page.Title,
		WebURL:    page.WebURL,
		Version:   page.Version,
		UpdatedAt: page.UpdatedAt,
	}
}

func confluencePreviewPageFromSnapshot(snapshot SourceSnapshot, identity confluenceSourceIdentity) ConfluenceSourcePreviewPage {
	return ConfluenceSourcePreviewPage{
		CloudID:   identity.CloudID,
		PageID:    identity.PageID,
		Title:     snapshot.Title,
		Version:   confluenceSnapshotVersion(snapshot),
		UpdatedAt: snapshot.ExternalUpdatedAt,
	}
}

func confluenceRangeFromSnapshot(snapshot SourceSnapshot) (ConfluenceRangeSelection, bool) {
	var locators []struct {
		LocatorType string `json:"locator_type"`
		ContentID   string `json:"content_id"`
		Start       int    `json:"start"`
		End         int    `json:"end"`
		Partial     bool   `json:"partial"`
	}
	if len(snapshot.Locators) == 0 || json.Unmarshal(snapshot.Locators, &locators) != nil {
		return ConfluenceRangeSelection{}, false
	}
	for _, locator := range locators {
		if locator.LocatorType == "confluence_page_range" || locator.Partial {
			return ConfluenceRangeSelection{
				ContentID: strings.TrimSpace(locator.ContentID),
				Start:     locator.Start,
				End:       locator.End,
			}, true
		}
	}
	return ConfluenceRangeSelection{}, false
}

func normalizeConfluenceUpdateProducer(producer Producer) Producer {
	producer.Type = strings.TrimSpace(producer.Type)
	producer.ID = strings.TrimSpace(producer.ID)
	if producer.Type == "" || producer.ID == "" {
		return Producer{Type: "connector", ID: ConfluenceConnectorID}
	}
	return producer
}

func validateConfluenceExpectedVersion(version int) error {
	if version <= 0 {
		return NewConfluenceValidationError(
			ConfluenceErrorCodeVersionDrift,
			"Confluence 소스 승인에는 검토한 page version이 필요합니다. 다시 미리보기한 뒤 승인하세요.",
		)
	}
	return nil
}

func optionalTimeString(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

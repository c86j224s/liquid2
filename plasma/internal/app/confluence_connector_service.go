package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
	"strings"
	"time"
)

// CheckConfluenceSourceUpdateWithEvent는 저장된 Confluence snapshot과 현재 page version 차이를 확인하고 결과 이벤트를 남긴다.
func (s *Service) CheckConfluenceSourceUpdateWithEvent(
	ctx context.Context,
	connector confluencesource.ConfluenceSourceConnector,
	req CheckConfluenceSourceUpdateRequest,
) (ConfluenceUpdateCheckResult, error) {
	result, err := s.CheckConfluenceSourceUpdate(ctx, connector, req)
	if err != nil {
		if recordErr := s.recordConfluenceUpdateCheckFailure(ctx, req, err); recordErr != nil {
			return ConfluenceUpdateCheckResult{}, errors.Join(err, recordErr)
		}
		return ConfluenceUpdateCheckResult{}, err
	}
	eventType := ConfluenceUpdateCurrentEvent
	if result.UpdateAvailable {
		eventType = ConfluenceUpdateAvailableEvent
	}
	event, err := buildLedgerEvent(ledger.AppendRequest{
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
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{Events: []ledger.Event{event}})
	if err != nil {
		return ConfluenceUpdateCheckResult{}, err
	}
	result.Event = committed.Events[0]
	return result, nil
}

// CheckConfluenceSourceUpdate는 이벤트를 쓰지 않고 Confluence source의 갱신 가능성만 계산한다.
func (s *Service) CheckConfluenceSourceUpdate(
	ctx context.Context,
	connector confluencesource.ConfluenceSourceConnector,
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
	latest, err = confluencesource.NormalizeVersion(latest, identity.CloudID, identity.PageID)
	if err != nil {
		return ConfluenceUpdateCheckResult{}, err
	}
	currentVersion := confluencesource.SnapshotVersion(snapshot)
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

// UpdateConfluenceSourceWithEvent는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) UpdateConfluenceSourceWithEvent(
	ctx context.Context,
	connector confluencesource.ConfluenceSourceConnector,
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
	if _, partial := confluenceRangeFromSnapshot(previous); partial && !confluencesource.RangeSelected(req.Range) {
		return ConfluenceUpdateResult{}, confluencesource.NewConfluenceValidationError(
			confluencesource.ConfluenceErrorCodeVersionDrift,
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
	oldVersion := confluencesource.SnapshotVersion(previous)
	newVersion := confluencesource.SnapshotVersion(snapshot)
	if oldVersion > 0 && newVersion > 0 && newVersion <= oldVersion {
		return ConfluenceUpdateResult{}, fmt.Errorf("%w: confluence source is not newer than the previous snapshot", ErrInvalidInput)
	}
	snapshotEvent, err := buildLedgerEvent(ledger.AppendRequest{
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
	updateEvent, err := buildLedgerEvent(ledger.AppendRequest{
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
		Events:          []ledger.Event{snapshotEvent, updateEvent},
		RawArtifacts:    []artifactcontract.Raw{artifact},
		SourceSnapshots: []sourcecontract.Snapshot{snapshot},
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

// SnapshotIdentity는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (result ConfluenceUpdateCheckResult) SnapshotIdentity() confluenceSourceIdentity {
	identity, _ := confluenceIdentityFromSnapshot(result.Snapshot)
	return identity
}

func (s *Service) confluenceSnapshotIdentity(ctx context.Context, missionID string, snapshotID string) (sourcecontract.Snapshot, confluenceSourceIdentity, error) {
	snapshot, err := s.GetSourceSnapshot(ctx, strings.TrimSpace(snapshotID))
	if err != nil {
		return sourcecontract.Snapshot{}, confluenceSourceIdentity{}, err
	}
	if snapshot.MissionID != missionID {
		return sourcecontract.Snapshot{}, confluenceSourceIdentity{}, fmt.Errorf("%w: source belongs to another mission", ErrInvalidInput)
	}
	if snapshot.Connector.ConnectorID != confluencesource.ConfluenceConnectorID || snapshot.Connector.ConnectorType != confluencesource.ConfluenceConnectorType {
		return sourcecontract.Snapshot{}, confluenceSourceIdentity{}, fmt.Errorf("%w: source is not a confluence snapshot", ErrInvalidInput)
	}
	identity, err := confluenceIdentityFromSnapshot(snapshot)
	if err != nil {
		return sourcecontract.Snapshot{}, confluenceSourceIdentity{}, err
	}
	return snapshot, identity, nil
}

func (s *Service) activeConfluenceSnapshotIdentity(ctx context.Context, missionID string, snapshotID string) (sourcecontract.Snapshot, confluenceSourceIdentity, error) {
	snapshot, identity, err := s.confluenceSnapshotIdentity(ctx, missionID, snapshotID)
	if err != nil {
		return sourcecontract.Snapshot{}, confluenceSourceIdentity{}, err
	}
	if snapshot.State.Removed {
		return sourcecontract.Snapshot{}, confluenceSourceIdentity{}, fmt.Errorf("%w: confluence source is removed", ErrInvalidInput)
	}
	if snapshot.State.Superseded {
		return sourcecontract.Snapshot{}, confluenceSourceIdentity{}, fmt.Errorf("%w: confluence source has been superseded", ErrInvalidInput)
	}
	current, ok, err := s.currentConfluenceSnapshotForIdentity(ctx, missionID, identity)
	if err != nil {
		return sourcecontract.Snapshot{}, confluenceSourceIdentity{}, err
	}
	if ok && current.SnapshotID != snapshot.SnapshotID {
		return sourcecontract.Snapshot{}, confluenceSourceIdentity{}, fmt.Errorf("%w: confluence source is not the current active snapshot", ErrInvalidInput)
	}
	return snapshot, identity, nil
}

func (s *Service) currentConfluenceSnapshotForIdentity(ctx context.Context, missionID string, identity confluenceSourceIdentity) (sourcecontract.Snapshot, bool, error) {
	sources, err := s.ListSourceSnapshotsWithState(ctx, sourcecontract.ListRequest{MissionID: missionID, IncludeRemoved: true})
	if err != nil {
		return sourcecontract.Snapshot{}, false, err
	}
	current := sourcecontract.Snapshot{}
	for _, source := range sources {
		if source.Connector.ConnectorID != confluencesource.ConfluenceConnectorID || source.Connector.ConnectorType != confluencesource.ConfluenceConnectorType {
			continue
		}
		sourceIdentity, err := confluenceIdentityFromSnapshot(source)
		if err != nil || sourceIdentity != identity {
			continue
		}
		if source.State.Removed || source.State.Superseded {
			continue
		}
		if current.SnapshotID == "" || confluencesource.SnapshotNewerThan(source, current) {
			current = source
		}
	}
	return current, current.SnapshotID != "", nil
}

func confluenceIdentityFromSnapshot(snapshot sourcecontract.Snapshot) (confluenceSourceIdentity, error) {
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

func readConfluenceVersion(ctx context.Context, connector confluencesource.ConfluenceSourceConnector, identity confluenceSourceIdentity) (confluencesource.ConfluenceSourceVersion, error) {
	req := confluencesource.ConfluenceSourceReadRequest{CloudID: identity.CloudID, PageID: identity.PageID}
	if versionConnector, ok := connector.(confluencesource.ConfluenceSourceVersionConnector); ok {
		return versionConnector.GetConfluenceSourceVersion(ctx, req)
	}
	page, err := connector.ReadConfluenceSource(ctx, req)
	if err != nil {
		return confluencesource.ConfluenceSourceVersion{}, err
	}
	return confluencesource.ConfluenceSourceVersion{
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

func normalizeConfluenceUpdateProducer(producer ledger.Producer) ledger.Producer {
	producer.Type = strings.TrimSpace(producer.Type)
	producer.ID = strings.TrimSpace(producer.ID)
	if producer.Type == "" || producer.ID == "" {
		return ledger.Producer{Type: "connector", ID: confluencesource.ConfluenceConnectorID}
	}
	return producer
}

func validateConfluenceExpectedVersion(version int) error {
	if version <= 0 {
		return confluencesource.NewConfluenceValidationError(
			confluencesource.ConfluenceErrorCodeVersionDrift,
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

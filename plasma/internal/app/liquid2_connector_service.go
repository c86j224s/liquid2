package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/source"
	"strings"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/source/liquid2source"
	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
)

// SearchLiquid2Sources는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) SearchLiquid2Sources(
	ctx context.Context,
	connector liquid2source.Liquid2SourceConnector,
	req liquid2source.Liquid2SourceSearchRequest,
) (liquid2source.Liquid2SourceSearchResult, error) {
	if connector == nil {
		return liquid2source.Liquid2SourceSearchResult{}, fmt.Errorf("%w: liquid2 connector is required", ErrInvalidInput)
	}
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return liquid2source.Liquid2SourceSearchResult{}, err
	}

	normalized := req
	normalized.MissionID = missionID
	normalized.Query = strings.TrimSpace(req.Query)
	normalized.Cursor = strings.TrimSpace(req.Cursor)
	normalized.Limit = liquid2source.NormalizeSearchLimit(req.Limit)
	normalized.Filters = liquid2source.NormalizeFilters(req.Filters)

	result, err := connector.SearchLiquid2Sources(ctx, normalized)
	if err != nil {
		return liquid2source.Liquid2SourceSearchResult{}, err
	}
	result.MissionID = missionID
	for i := range result.Candidates {
		candidate, err := liquid2source.NormalizeCandidate(result.Candidates[i])
		if err != nil {
			return liquid2source.Liquid2SourceSearchResult{}, err
		}
		result.Candidates[i] = candidate
	}
	return result, nil
}

// SnapshotLiquid2Source는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) SnapshotLiquid2Source(
	ctx context.Context,
	connector liquid2source.Liquid2SourceConnector,
	req liquid2source.SnapshotLiquid2SourceRequest,
) (liquid2source.Liquid2SnapshotResult, error) {
	artifact, snapshot, err := s.buildLiquid2Snapshot(ctx, connector, req)
	if err != nil {
		return liquid2source.Liquid2SnapshotResult{}, err
	}
	if err := s.store.CreateRawArtifact(ctx, artifact); err != nil {
		return liquid2source.Liquid2SnapshotResult{}, err
	}
	if err := s.store.CreateSourceSnapshot(ctx, snapshot); err != nil {
		return liquid2source.Liquid2SnapshotResult{}, err
	}
	return liquid2source.Liquid2SnapshotResult{Artifact: artifact, Snapshot: snapshot}, nil
}

// SnapshotLiquid2SourceWithEvent는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) SnapshotLiquid2SourceWithEvent(
	ctx context.Context,
	connector liquid2source.Liquid2SourceConnector,
	req liquid2source.SnapshotLiquid2SourceWithEventRequest,
) (liquid2source.Liquid2SnapshotWithEventResult, error) {
	artifact, snapshot, err := s.buildLiquid2Snapshot(ctx, connector, req.Snapshot)
	if err != nil {
		return liquid2source.Liquid2SnapshotWithEventResult{}, err
	}
	event, err := buildLedgerEvent(ledger.AppendRequest{
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
		return liquid2source.Liquid2SnapshotWithEventResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []ledger.Event{event},
		RawArtifacts:    []artifactcontract.Raw{artifact},
		SourceSnapshots: []sourcecontract.Snapshot{snapshot},
	})
	if err != nil {
		return liquid2source.Liquid2SnapshotWithEventResult{}, err
	}
	return liquid2source.Liquid2SnapshotWithEventResult{
		Artifact: artifact,
		Snapshot: snapshot,
		Event:    committed.Events[0],
	}, nil
}

func (s *Service) buildLiquid2Snapshot(
	ctx context.Context,
	connector liquid2source.Liquid2SourceConnector,
	req liquid2source.SnapshotLiquid2SourceRequest,
) (artifactcontract.Raw, sourcecontract.Snapshot, error) {
	if connector == nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, fmt.Errorf("%w: liquid2 connector is required", ErrInvalidInput)
	}
	missionID := strings.TrimSpace(req.MissionID)
	artifactID := strings.TrimSpace(req.ArtifactID)
	snapshotID := strings.TrimSpace(req.SnapshotID)
	externalSourceID := strings.TrimSpace(req.ExternalSourceID)
	if err := validateID("mis_", missionID); err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	if err := validateID("art_", artifactID); err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	if err := validateID("src_", snapshotID); err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	if externalSourceID == "" {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, fmt.Errorf("%w: external source id is required", ErrInvalidInput)
	}

	document, err := connector.ReadLiquid2Source(ctx, liquid2source.Liquid2SourceReadRequest{ExternalSourceID: externalSourceID})
	if err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	document, err = liquid2source.NormalizeDocument(document, externalSourceID)
	if err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}

	content, locators, err := liquid2source.BuildSnapshotPayload(document, artifactID, strings.TrimSpace(req.Reason), req.ContentRanges)
	if err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	producer, err := liquid2SnapshotProducer(req.Producer)
	if err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	artifact, err := artifactcontract.Build(artifactcontract.CreateRequest{
		ArtifactID: artifactID,
		MissionID:  missionID,
		MediaType:  liquid2source.Liquid2SnapshotMediaType,
		Filename:   liquid2SnapshotFilename(document.Connector.ExternalSourceID),
		Producer:   producer,
		Content:    content,
	})
	if err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}

	snapshot, err := source.BuildSnapshot(ctx, s.store, source.CreateRequest{
		SnapshotID:        snapshotID,
		MissionID:         missionID,
		Connector:         document.Connector,
		Title:             document.Title,
		ExternalUpdatedAt: document.UpdatedAt,
		ArtifactIDs:       []string{artifact.ArtifactID},
		ContentHash:       req.ExpectedContentHash,
		Locators:          locators,
	}, []artifactcontract.Raw{artifact})
	if err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	return artifact, snapshot, nil
}

func liquid2SnapshotProducer(producer ledger.Producer) (ledger.Producer, error) {
	producer.Type = strings.TrimSpace(producer.Type)
	producer.ID = strings.TrimSpace(producer.ID)
	if producer.Type == "" && producer.ID == "" {
		return ledger.Producer{Type: "connector", ID: liquid2source.Liquid2ConnectorID}, nil
	}
	if producer.Type != "connector" || producer.ID != liquid2source.Liquid2ConnectorID {
		return ledger.Producer{}, fmt.Errorf("%w: liquid2 snapshot producer must be connector/liquid2", ErrInvalidInput)
	}
	return producer, nil
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
		return "plasma-liquid2-snapshot.json"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_")
	return "plasma-liquid2-snapshot-" + replacer.Replace(externalSourceID) + ".json"
}

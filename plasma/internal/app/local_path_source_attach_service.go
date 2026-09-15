package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/source"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"strings"
)

// ListLocalPathRoots는 configured allowlist root의 공개 view만 반환한다.
func (s *Service) ListLocalPathRoots(ctx context.Context) ([]sourcecontract.LocalPathRoot, error) {
	engine, err := s.localPathEngine()
	if err != nil {
		return nil, err
	}
	return engine.Roots(), nil
}

// BrowseLocalPathRoot는 source 승인 전 root 탐색을 bounded tree result로 반환한다.
func (s *Service) BrowseLocalPathRoot(ctx context.Context, req BrowseLocalPathRootRequest) (sourcecontract.LocalPathTreeResult, error) {
	engine, err := s.localPathEngine()
	if err != nil {
		return sourcecontract.LocalPathTreeResult{}, err
	}
	return engine.Tree(ctx, sourcecontract.LocalPathTreeRequest{RootID: req.RootID, RelativePath: req.RelativePath, Depth: req.Depth, Limit: req.Limit})
}

// AttachLocalPathSource는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) AttachLocalPathSource(ctx context.Context, req AttachLocalPathSourceRequest) (LocalPathSourceResult, error) {
	engine, err := s.localPathEngine()
	if err != nil {
		return LocalPathSourceResult{}, err
	}
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return LocalPathSourceResult{}, err
	}
	metadata, err := engine.Inspect(ctx, req.RootID, req.RelativePath)
	if err != nil {
		return LocalPathSourceResult{}, localPathErr(err)
	}
	existing, ok, err := s.findLocalPathSource(ctx, missionID, metadata.RootID, metadata.RelativePath)
	if err != nil {
		return LocalPathSourceResult{}, err
	}
	if ok {
		if existing.State.Removed {
			if !req.Restore {
				return LocalPathSourceResult{Snapshot: existing, Existing: true, RestoreRequired: true}, fmt.Errorf("%w: source is removed; restore is required", ErrConflict)
			}
			restored, err := s.RestoreSource(ctx, RestoreSourceRequest{MissionID: missionID, SnapshotID: existing.SnapshotID, Producer: defaultProducer(req.Producer)})
			if err != nil {
				return LocalPathSourceResult{}, err
			}
			return LocalPathSourceResult{Snapshot: restored.Snapshot, Event: restored.Event, Existing: true, Restored: true}, nil
		}
		return LocalPathSourceResult{Snapshot: existing, Existing: true}, nil
	}
	snapshotID := strings.TrimSpace(req.SnapshotID)
	if snapshotID == "" {
		snapshotID = newAppID("src")
	}
	locator := source.LocalPathLocator{LocatorType: sourcecontract.LocatorTypeLocalPath, RootID: metadata.RootID, RelativePath: metadata.RelativePath, PathKind: metadata.PathKind}
	locators, err := json.Marshal([]source.LocalPathLocator{locator})
	if err != nil {
		return LocalPathSourceResult{}, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = metadata.RelativePath
	}
	snapshot, err := source.BuildSnapshot(ctx, s.store, source.CreateRequest{
		SnapshotID: snapshotID,
		MissionID:  missionID,
		Connector: sourcecontract.ConnectorRef{
			ConnectorID:      sourcecontract.ConnectorTypeLocalPath,
			ConnectorType:    sourcecontract.ConnectorTypeLocalPath,
			ExternalSourceID: metadata.RootID + ":" + metadata.RelativePath,
			ConnectorVersion: "plasma.local_path.v1",
		},
		Title:    title,
		Locators: json.RawMessage(locators),
		Access:   sourcecontract.Access{RetrievalPolicy: sourcecontract.RetrievalPolicyLiveReference},
	}, nil)
	if err != nil {
		return LocalPathSourceResult{}, err
	}
	event, err := buildLedgerEvent(ledger.AppendRequest{
		EventID:   newAppID("evt"),
		MissionID: missionID,
		EventType: SourceLocalPathAttachedEvent,
		Producer:  defaultProducer(req.Producer),
		Payload: mustMarshalJSON(map[string]any{
			"snapshot_id":      snapshot.SnapshotID,
			"connector_type":   sourcecontract.ConnectorTypeLocalPath,
			"retrieval_policy": sourcecontract.RetrievalPolicyLiveReference,
			"root_id":          metadata.RootID,
			"relative_path":    metadata.RelativePath,
			"path_kind":        metadata.PathKind,
			"title":            snapshot.Title,
		}),
	})
	if err != nil {
		return LocalPathSourceResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{Events: []ledger.Event{event}, SourceSnapshots: []sourcecontract.Snapshot{snapshot}})
	if err != nil {
		return LocalPathSourceResult{}, err
	}
	snapshot.State = source.State{State: sourcecontract.StateActive}
	return LocalPathSourceResult{Snapshot: snapshot, Event: &committed.Events[0]}, nil
}

func (s *Service) findLocalPathSource(ctx context.Context, missionID string, rootID string, relativePath string) (sourcecontract.Snapshot, bool, error) {
	sources, err := s.ListSourceSnapshotsWithState(ctx, source.ListRequest{MissionID: missionID, IncludeRemoved: true})
	if err != nil {
		return sourcecontract.Snapshot{}, false, err
	}
	for _, snapshot := range sources {
		if snapshot.Connector.ConnectorType != sourcecontract.ConnectorTypeLocalPath || snapshot.Access.RetrievalPolicy != sourcecontract.RetrievalPolicyLiveReference {
			continue
		}
		locator, err := source.ParseLocalPathLocator(snapshot.Locators)
		if err != nil {
			continue
		}
		if locator.RootID == rootID && locator.RelativePath == relativePath {
			return snapshot, true, nil
		}
	}
	return sourcecontract.Snapshot{}, false, nil
}

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/sources/localpath"
)

const (
	SourceLocalPathAttachedEvent = "source.local_path.attached"
	SourceObservedEvent          = "source.observed"
	SourceObserveFailedEvent     = "source.observe.failed"
)

type AttachLocalPathSourceRequest struct {
	MissionID    string
	SnapshotID   string
	RootID       string
	RelativePath string
	Title        string
	Restore      bool
	Producer     Producer
}

type LocalPathSourceResult struct {
	Snapshot        SourceSnapshot
	Event           *LedgerEvent
	Existing        bool
	Restored        bool
	RestoreRequired bool
}

type BrowseLocalPathRootRequest struct {
	RootID       string
	RelativePath string
	Depth        int
	Limit        int
}

type ReadLocalPathSourceRequest struct {
	MissionID     string
	SnapshotID    string
	Subpath       string
	Offset        int64
	MaxBytes      int64
	Producer      Producer
	ToolSessionID string
}

type ReadLocalPathSourceResult struct {
	Snapshot         SourceSnapshot
	Read             localpath.ReadResult
	ObservationEvent *LedgerEvent
}

type TreeLocalPathSourceRequest struct {
	MissionID     string
	SnapshotID    string
	Subpath       string
	Depth         int
	Limit         int
	Producer      Producer
	ToolSessionID string
}

type TreeLocalPathSourceResult struct {
	Snapshot         SourceSnapshot
	Tree             localpath.TreeResult
	ObservationEvent *LedgerEvent
}

type GrepLocalPathSourceRequest struct {
	MissionID     string
	SnapshotID    string
	Subpath       string
	Query         string
	MaxSnippets   int
	Producer      Producer
	ToolSessionID string
}

type GrepLocalPathSourceResult struct {
	Snapshot         SourceSnapshot
	Grep             localpath.GrepResult
	ObservationEvent *LedgerEvent
}

type RemoveSourceRequest struct {
	MissionID  string
	SnapshotID string
	Reason     string
	Producer   Producer
}

type RestoreSourceRequest struct {
	MissionID  string
	SnapshotID string
	Producer   Producer
}

type SourceStateChangeResult struct {
	Snapshot   SourceSnapshot
	Event      *LedgerEvent
	Idempotent bool
}

func (s *Service) ListLocalPathRoots(ctx context.Context) ([]localpath.RootView, error) {
	engine, err := s.localPathEngine()
	if err != nil {
		return nil, err
	}
	return engine.Roots(), nil
}

func (s *Service) BrowseLocalPathRoot(ctx context.Context, req BrowseLocalPathRootRequest) (localpath.TreeResult, error) {
	engine, err := s.localPathEngine()
	if err != nil {
		return localpath.TreeResult{}, err
	}
	return engine.Tree(ctx, localpath.TreeRequest{RootID: req.RootID, RelativePath: req.RelativePath, Depth: req.Depth, Limit: req.Limit})
}

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
	locator := LocalPathLocator{LocatorType: SourceLocatorTypeLocalPath, RootID: metadata.RootID, RelativePath: metadata.RelativePath, PathKind: metadata.PathKind}
	locators, err := json.Marshal([]LocalPathLocator{locator})
	if err != nil {
		return LocalPathSourceResult{}, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = metadata.RelativePath
	}
	snapshot, err := s.buildSourceSnapshot(ctx, CreateSourceSnapshotRequest{
		SnapshotID: snapshotID,
		MissionID:  missionID,
		Connector: ConnectorRef{
			ConnectorID:      SourceConnectorTypeLocalPath,
			ConnectorType:    SourceConnectorTypeLocalPath,
			ExternalSourceID: metadata.RootID + ":" + metadata.RelativePath,
			ConnectorVersion: "plasma.local_path.v1",
		},
		Title:    title,
		Locators: json.RawMessage(locators),
		Access:   SourceAccess{RetrievalPolicy: SourceRetrievalPolicyLiveReference},
	}, nil)
	if err != nil {
		return LocalPathSourceResult{}, err
	}
	event, err := buildLedgerEvent(AppendEventRequest{
		EventID:   newAppID("evt"),
		MissionID: missionID,
		EventType: SourceLocalPathAttachedEvent,
		Producer:  defaultProducer(req.Producer),
		Payload: mustMarshalJSON(map[string]any{
			"snapshot_id":      snapshot.SnapshotID,
			"connector_type":   SourceConnectorTypeLocalPath,
			"retrieval_policy": SourceRetrievalPolicyLiveReference,
			"root_id":          metadata.RootID,
			"relative_path":    metadata.RelativePath,
			"path_kind":        metadata.PathKind,
			"title":            snapshot.Title,
		}),
	})
	if err != nil {
		return LocalPathSourceResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{Events: []LedgerEvent{event}, SourceSnapshots: []SourceSnapshot{snapshot}})
	if err != nil {
		return LocalPathSourceResult{}, err
	}
	snapshot.State = SourceState{State: SourceStateActive}
	return LocalPathSourceResult{Snapshot: snapshot, Event: &committed.Events[0]}, nil
}

func (s *Service) ReadLocalPathSource(ctx context.Context, req ReadLocalPathSourceRequest) (ReadLocalPathSourceResult, error) {
	snapshot, locator, err := s.activeLiveLocalPathSource(ctx, req.MissionID, req.SnapshotID)
	if err != nil {
		return ReadLocalPathSourceResult{}, err
	}
	engine, err := s.localPathEngine()
	if err != nil {
		return ReadLocalPathSourceResult{}, err
	}
	targetRelativePath, subpath, err := localPathSourceTarget(locator, req.Subpath)
	if err != nil {
		_ = s.appendObserveFailure(ctx, snapshot, "read", "", req.ToolSessionID, defaultProducer(req.Producer), err)
		return ReadLocalPathSourceResult{}, err
	}
	readReq := localpath.ReadRequest{RootID: locator.RootID, RelativePath: locator.RelativePath, Subpath: subpath, Offset: req.Offset, MaxBytes: req.MaxBytes}
	read, err := engine.ReadFile(ctx, readReq)
	isPDF := isPDFLocalPath(targetRelativePath)
	if err == nil && !isPDF {
		detected, detectErr := engine.IsPDF(ctx, locator.RootID, targetRelativePath)
		if detectErr != nil {
			err = detectErr
		} else {
			isPDF = detected
		}
	}
	if err == nil && (isPDF || read.Metadata.Cap == "pdf_text") {
		read, err = engine.ReadPDFText(ctx, readReq)
	}
	if err != nil {
		_ = s.appendObserveFailure(ctx, snapshot, "read", subpath, req.ToolSessionID, defaultProducer(req.Producer), localPathErr(err))
		return ReadLocalPathSourceResult{}, localPathErr(err)
	}
	event, err := s.appendObservation(ctx, snapshot, "read", read.Metadata, map[string]any{
		"content_returned": strings.TrimSpace(read.Content) != "",
		"binary":           read.Metadata.Binary,
	}, req.ToolSessionID, defaultProducer(req.Producer))
	if err != nil {
		return ReadLocalPathSourceResult{}, err
	}
	return ReadLocalPathSourceResult{Snapshot: snapshot, Read: read, ObservationEvent: event}, nil
}

func (s *Service) TreeLocalPathSource(ctx context.Context, req TreeLocalPathSourceRequest) (TreeLocalPathSourceResult, error) {
	snapshot, locator, err := s.activeLiveLocalPathSource(ctx, req.MissionID, req.SnapshotID)
	if err != nil {
		return TreeLocalPathSourceResult{}, err
	}
	engine, err := s.localPathEngine()
	if err != nil {
		return TreeLocalPathSourceResult{}, err
	}
	_, subpath, err := localPathSourceTarget(locator, req.Subpath)
	if err != nil {
		_ = s.appendObserveFailure(ctx, snapshot, "tree", "", req.ToolSessionID, defaultProducer(req.Producer), err)
		return TreeLocalPathSourceResult{}, err
	}
	tree, err := engine.Tree(ctx, localpath.TreeRequest{RootID: locator.RootID, RelativePath: locator.RelativePath, Subpath: subpath, Depth: req.Depth, Limit: req.Limit})
	if err != nil {
		_ = s.appendObserveFailure(ctx, snapshot, "tree", subpath, req.ToolSessionID, defaultProducer(req.Producer), localPathErr(err))
		return TreeLocalPathSourceResult{}, localPathErr(err)
	}
	event, err := s.appendObservation(ctx, snapshot, "tree", tree.Metadata, map[string]any{
		"entry_count": len(tree.Entries),
		"truncated":   tree.Truncated,
	}, req.ToolSessionID, defaultProducer(req.Producer))
	if err != nil {
		return TreeLocalPathSourceResult{}, err
	}
	return TreeLocalPathSourceResult{Snapshot: snapshot, Tree: tree, ObservationEvent: event}, nil
}

func (s *Service) GrepLocalPathSource(ctx context.Context, req GrepLocalPathSourceRequest) (GrepLocalPathSourceResult, error) {
	snapshot, locator, err := s.activeLiveLocalPathSource(ctx, req.MissionID, req.SnapshotID)
	if err != nil {
		return GrepLocalPathSourceResult{}, err
	}
	engine, err := s.localPathEngine()
	if err != nil {
		return GrepLocalPathSourceResult{}, err
	}
	_, subpath, err := localPathSourceTarget(locator, req.Subpath)
	if err != nil {
		_ = s.appendObserveFailure(ctx, snapshot, "grep", "", req.ToolSessionID, defaultProducer(req.Producer), err)
		return GrepLocalPathSourceResult{}, err
	}
	grep, err := engine.Grep(ctx, localpath.GrepRequest{RootID: locator.RootID, RelativePath: locator.RelativePath, Subpath: subpath, Query: req.Query, MaxSnippets: req.MaxSnippets})
	if err != nil {
		_ = s.appendObserveFailure(ctx, snapshot, "grep", subpath, req.ToolSessionID, defaultProducer(req.Producer), localPathErr(err))
		return GrepLocalPathSourceResult{}, localPathErr(err)
	}
	event, err := s.appendObservation(ctx, snapshot, "grep", grep.Metadata, map[string]any{
		"query":       strings.TrimSpace(req.Query),
		"match_count": len(grep.Matches),
		"truncated":   grep.Truncated,
	}, req.ToolSessionID, defaultProducer(req.Producer))
	if err != nil {
		return GrepLocalPathSourceResult{}, err
	}
	return GrepLocalPathSourceResult{Snapshot: snapshot, Grep: grep, ObservationEvent: event}, nil
}

func (s *Service) RemoveSource(ctx context.Context, req RemoveSourceRequest) (SourceStateChangeResult, error) {
	snapshot, err := s.GetSourceSnapshot(ctx, strings.TrimSpace(req.SnapshotID))
	if err != nil {
		return SourceStateChangeResult{}, err
	}
	missionID := strings.TrimSpace(req.MissionID)
	if snapshot.MissionID != missionID {
		return SourceStateChangeResult{}, fmt.Errorf("%w: source belongs to another mission", ErrInvalidInput)
	}
	if snapshot.State.Removed {
		return SourceStateChangeResult{Snapshot: snapshot, Idempotent: true}, nil
	}
	event, err := s.AppendEvent(ctx, AppendEventRequest{
		EventID:   newAppID("evt"),
		MissionID: missionID,
		EventType: SourceRemovedEvent,
		Producer:  defaultProducer(req.Producer),
		Payload: mustMarshalJSON(map[string]any{
			"snapshot_id": snapshot.SnapshotID,
			"reason":      strings.TrimSpace(req.Reason),
			"removed_at":  time.Now().UTC().Format(time.RFC3339Nano),
		}),
	})
	if err != nil {
		return SourceStateChangeResult{}, err
	}
	snapshot.State, _ = s.sourceState(ctx, missionID, snapshot.SnapshotID)
	return SourceStateChangeResult{Snapshot: snapshot, Event: &event}, nil
}

func (s *Service) RestoreSource(ctx context.Context, req RestoreSourceRequest) (SourceStateChangeResult, error) {
	snapshot, err := s.GetSourceSnapshot(ctx, strings.TrimSpace(req.SnapshotID))
	if err != nil {
		return SourceStateChangeResult{}, err
	}
	missionID := strings.TrimSpace(req.MissionID)
	if snapshot.MissionID != missionID {
		return SourceStateChangeResult{}, fmt.Errorf("%w: source belongs to another mission", ErrInvalidInput)
	}
	if !snapshot.State.Removed {
		return SourceStateChangeResult{Snapshot: snapshot, Idempotent: true}, nil
	}
	event, err := s.AppendEvent(ctx, AppendEventRequest{
		EventID:   newAppID("evt"),
		MissionID: missionID,
		EventType: SourceRestoredEvent,
		Producer:  defaultProducer(req.Producer),
		Payload: mustMarshalJSON(map[string]any{
			"snapshot_id": snapshot.SnapshotID,
			"restored_at": time.Now().UTC().Format(time.RFC3339Nano),
		}),
	})
	if err != nil {
		return SourceStateChangeResult{}, err
	}
	snapshot.State, _ = s.sourceState(ctx, missionID, snapshot.SnapshotID)
	return SourceStateChangeResult{Snapshot: snapshot, Event: &event}, nil
}

func (s *Service) activeLiveLocalPathSource(ctx context.Context, missionID string, snapshotID string) (SourceSnapshot, LocalPathLocator, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateID("mis_", missionID); err != nil {
		return SourceSnapshot{}, LocalPathLocator{}, err
	}
	snapshot, err := s.GetSourceSnapshot(ctx, strings.TrimSpace(snapshotID))
	if err != nil {
		return SourceSnapshot{}, LocalPathLocator{}, err
	}
	if snapshot.MissionID != missionID {
		return SourceSnapshot{}, LocalPathLocator{}, fmt.Errorf("%w: source belongs to another mission", ErrInvalidInput)
	}
	if snapshot.State.Removed {
		return SourceSnapshot{}, LocalPathLocator{}, fmt.Errorf("%w: source is removed", ErrInvalidInput)
	}
	if snapshot.Access.RetrievalPolicy != SourceRetrievalPolicyLiveReference || snapshot.Connector.ConnectorType != SourceConnectorTypeLocalPath {
		return SourceSnapshot{}, LocalPathLocator{}, fmt.Errorf("%w: source is not a local path live reference", ErrInvalidInput)
	}
	locator, err := parseLocalPathLocator(snapshot.Locators)
	if err != nil {
		return SourceSnapshot{}, LocalPathLocator{}, err
	}
	return snapshot, locator, nil
}

func localPathSourceTarget(locator LocalPathLocator, subpath string) (string, string, error) {
	if strings.TrimSpace(subpath) != "" && locator.PathKind == "file" {
		return "", "", fmt.Errorf("%w: subpath is only valid for directory local_path sources", ErrInvalidInput)
	}
	target, cleanSubpath, err := localpath.TargetRelativePath(locator.RelativePath, subpath)
	if err != nil {
		return "", "", localPathErr(err)
	}
	return target, cleanSubpath, nil
}

func (s *Service) findLocalPathSource(ctx context.Context, missionID string, rootID string, relativePath string) (SourceSnapshot, bool, error) {
	sources, err := s.ListSourceSnapshotsWithState(ctx, ListSourceSnapshotsRequest{MissionID: missionID, IncludeRemoved: true})
	if err != nil {
		return SourceSnapshot{}, false, err
	}
	for _, source := range sources {
		if source.Connector.ConnectorType != SourceConnectorTypeLocalPath || source.Access.RetrievalPolicy != SourceRetrievalPolicyLiveReference {
			continue
		}
		locator, err := parseLocalPathLocator(source.Locators)
		if err != nil {
			continue
		}
		if locator.RootID == rootID && locator.RelativePath == relativePath {
			return source, true, nil
		}
	}
	return SourceSnapshot{}, false, nil
}

func (s *Service) appendObservation(ctx context.Context, snapshot SourceSnapshot, operation string, metadata localpath.PathMetadata, extra map[string]any, toolSessionID string, producer Producer) (*LedgerEvent, error) {
	payload := map[string]any{
		"snapshot_id":       snapshot.SnapshotID,
		"connector_type":    SourceConnectorTypeLocalPath,
		"retrieval_policy":  SourceRetrievalPolicyLiveReference,
		"operation":         operation,
		"observed_at":       metadata.ObservedAt,
		"root_id":           metadata.RootID,
		"root_alias":        metadata.RootAlias,
		"relative_path":     metadata.RelativePath,
		"subpath":           metadata.Subpath,
		"path_kind":         metadata.PathKind,
		"size":              metadata.Size,
		"mtime":             metadata.MTime,
		"sha256":            metadata.SHA256,
		"offset":            metadata.Offset,
		"max_bytes":         metadata.MaxBytes,
		"next_offset":       metadata.NextOffset,
		"truncated":         metadata.Truncated,
		"binary":            metadata.Binary,
		"extraction":        metadata.Extraction,
		"page_count":        metadata.PageCount,
		"text_length":       metadata.TextLength,
		"text_length_known": metadata.TextLengthKnown,
		"git":               metadata.Git,
	}
	if strings.TrimSpace(toolSessionID) != "" {
		payload["tool_session_id"] = strings.TrimSpace(toolSessionID)
	}
	for key, value := range extra {
		payload[key] = value
	}
	event, err := s.AppendEvent(ctx, AppendEventRequest{
		EventID:   newAppID("evt"),
		MissionID: snapshot.MissionID,
		EventType: SourceObservedEvent,
		Producer:  producer,
		Payload:   mustMarshalJSON(payload),
	})
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func isPDFLocalPath(relativePath string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(relativePath)), ".pdf")
}

func (s *Service) appendObserveFailure(ctx context.Context, snapshot SourceSnapshot, operation string, subpath string, toolSessionID string, producer Producer, cause error) error {
	payload := map[string]any{
		"snapshot_id":      snapshot.SnapshotID,
		"connector_type":   SourceConnectorTypeLocalPath,
		"retrieval_policy": SourceRetrievalPolicyLiveReference,
		"operation":        operation,
		"error":            strings.TrimSpace(cause.Error()),
	}
	if strings.TrimSpace(subpath) != "" {
		payload["subpath"] = strings.TrimSpace(subpath)
	}
	if strings.TrimSpace(toolSessionID) != "" {
		payload["tool_session_id"] = strings.TrimSpace(toolSessionID)
	}
	_, err := s.AppendEvent(ctx, AppendEventRequest{
		EventID:   newAppID("evt"),
		MissionID: snapshot.MissionID,
		EventType: SourceObserveFailedEvent,
		Producer:  producer,
		Payload:   mustMarshalJSON(payload),
	})
	return err
}

func (s *Service) localPathEngine() (*localpath.Engine, error) {
	if s.localPaths == nil {
		return nil, fmt.Errorf("%w: local path roots are not configured", ErrInvalidInput)
	}
	return s.localPaths, nil
}

func defaultProducer(producer Producer) Producer {
	if strings.TrimSpace(producer.Type) == "" {
		producer.Type = "user"
	}
	if strings.TrimSpace(producer.ID) == "" {
		producer.ID = "plasma"
	}
	producer.Type = strings.TrimSpace(producer.Type)
	producer.ID = strings.TrimSpace(producer.ID)
	return producer
}

func localPathErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", ErrInvalidInput, err)
}

func observationEventID(event *LedgerEvent) string {
	if event == nil {
		return ""
	}
	return event.EventID
}

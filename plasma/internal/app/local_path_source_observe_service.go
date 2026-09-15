package app

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"strings"
)

// ReadLocalPathSource는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
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
	readReq := sourcecontract.LocalPathReadRequest{RootID: locator.RootID, RelativePath: locator.RelativePath, Subpath: subpath, Offset: req.Offset, MaxBytes: req.MaxBytes}
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

// TreeLocalPathSource는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
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
	tree, err := engine.Tree(ctx, sourcecontract.LocalPathTreeRequest{RootID: locator.RootID, RelativePath: locator.RelativePath, Subpath: subpath, Depth: req.Depth, Limit: req.Limit})
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

// GrepLocalPathSource는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
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
	grep, err := engine.Grep(ctx, sourcecontract.LocalPathGrepRequest{RootID: locator.RootID, RelativePath: locator.RelativePath, Subpath: subpath, Query: req.Query, MaxSnippets: req.MaxSnippets})
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

func (s *Service) appendObservation(ctx context.Context, snapshot sourcecontract.Snapshot, operation string, metadata sourcecontract.LocalPathMetadata, extra map[string]any, toolSessionID string, producer ledger.Producer) (*ledger.Event, error) {
	payload := map[string]any{
		"snapshot_id":       snapshot.SnapshotID,
		"connector_type":    sourcecontract.ConnectorTypeLocalPath,
		"retrieval_policy":  sourcecontract.RetrievalPolicyLiveReference,
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
	event, err := s.AppendEvent(ctx, ledger.AppendRequest{
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

func (s *Service) appendObserveFailure(ctx context.Context, snapshot sourcecontract.Snapshot, operation string, subpath string, toolSessionID string, producer ledger.Producer, cause error) error {
	payload := map[string]any{
		"snapshot_id":      snapshot.SnapshotID,
		"connector_type":   sourcecontract.ConnectorTypeLocalPath,
		"retrieval_policy": sourcecontract.RetrievalPolicyLiveReference,
		"operation":        operation,
		"error":            strings.TrimSpace(cause.Error()),
	}
	if strings.TrimSpace(subpath) != "" {
		payload["subpath"] = strings.TrimSpace(subpath)
	}
	if strings.TrimSpace(toolSessionID) != "" {
		payload["tool_session_id"] = strings.TrimSpace(toolSessionID)
	}
	_, err := s.AppendEvent(ctx, ledger.AppendRequest{
		EventID:   newAppID("evt"),
		MissionID: snapshot.MissionID,
		EventType: SourceObserveFailedEvent,
		Producer:  producer,
		Payload:   mustMarshalJSON(payload),
	})
	return err
}

func observationEventID(event *ledger.Event) string {
	if event == nil {
		return ""
	}
	return event.EventID
}

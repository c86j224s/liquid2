package app

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/source"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"reflect"
	"strings"
)

const (
	// SourceLocalPathAttachedEvent와 SourceObservedEvent 계열은 live local path source의
	// 등록과 관찰을 장부에 남기는 event type이다.
	SourceLocalPathAttachedEvent = "source.local_path.attached"
	SourceObservedEvent          = "source.observed"
	SourceObserveFailedEvent     = "source.observe.failed"
)

// AttachLocalPathSourceRequest는 allowlisted local path를 mission source로 연결하는
// 입력이다.
type AttachLocalPathSourceRequest struct {
	MissionID    string
	SnapshotID   string
	RootID       string
	RelativePath string
	Title        string
	Restore      bool
	Producer     ledger.Producer
}

// LocalPathSourceResult는 local path source attach/restore 결과다.
type LocalPathSourceResult struct {
	Snapshot        sourcecontract.Snapshot
	Event           *ledger.Event
	Existing        bool
	Restored        bool
	RestoreRequired bool
}

// BrowseLocalPathRootRequest는 아직 source로 승인되지 않은 allowlist root를 탐색하는
// 입력이다.
type BrowseLocalPathRootRequest struct {
	RootID       string
	RelativePath string
	Depth        int
	Limit        int
}

// ReadLocalPathSourceRequest는 승인된 live local path source의 file content를 관찰하는
// 입력이다.
type ReadLocalPathSourceRequest struct {
	MissionID     string
	SnapshotID    string
	Subpath       string
	Offset        int64
	MaxBytes      int64
	Producer      ledger.Producer
	ToolSessionID string
}

// ReadLocalPathSourceResult는 local path read 결과와 선택적으로 기록된 observation event다.
type ReadLocalPathSourceResult struct {
	Snapshot         sourcecontract.Snapshot
	Read             sourcecontract.LocalPathReadResult
	ObservationEvent *ledger.Event
}

// TreeLocalPathSourceRequest는 승인된 live local path source의 directory tree를 관찰하는
// 입력이다.
type TreeLocalPathSourceRequest struct {
	MissionID     string
	SnapshotID    string
	Subpath       string
	Depth         int
	Limit         int
	Producer      ledger.Producer
	ToolSessionID string
}

// TreeLocalPathSourceResult는 local path tree 결과와 observation event다.
type TreeLocalPathSourceResult struct {
	Snapshot         sourcecontract.Snapshot
	Tree             sourcecontract.LocalPathTreeResult
	ObservationEvent *ledger.Event
}

// GrepLocalPathSourceRequest는 승인된 live local path source 안에서 bounded grep을
// 수행하는 입력이다.
type GrepLocalPathSourceRequest struct {
	MissionID     string
	SnapshotID    string
	Subpath       string
	Query         string
	MaxSnippets   int
	Producer      ledger.Producer
	ToolSessionID string
}

// GrepLocalPathSourceResult는 local path grep 결과와 observation event다.
type GrepLocalPathSourceResult struct {
	Snapshot         sourcecontract.Snapshot
	Grep             sourcecontract.LocalPathGrepResult
	ObservationEvent *ledger.Event
}

// RemoveSourceRequest는 source snapshot을 active set에서 soft-remove하는 입력이다.
type RemoveSourceRequest struct {
	MissionID  string
	SnapshotID string
	Reason     string
	Producer   ledger.Producer
}

// RestoreSourceRequest는 soft-removed source snapshot을 active set으로 되돌리는 입력이다.
type RestoreSourceRequest struct {
	MissionID  string
	SnapshotID string
	Producer   ledger.Producer
}

// SourceStateChangeResult는 source 상태 변경 결과와 idempotency 여부다.
type SourceStateChangeResult struct {
	Snapshot   sourcecontract.Snapshot
	Event      *ledger.Event
	Idempotent bool
}

func (s *Service) activeLiveLocalPathSource(ctx context.Context, missionID string, snapshotID string) (sourcecontract.Snapshot, source.LocalPathLocator, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateID("mis_", missionID); err != nil {
		return sourcecontract.Snapshot{}, source.LocalPathLocator{}, err
	}
	snapshot, err := s.GetSourceSnapshot(ctx, strings.TrimSpace(snapshotID))
	if err != nil {
		return sourcecontract.Snapshot{}, source.LocalPathLocator{}, err
	}
	if snapshot.MissionID != missionID {
		return sourcecontract.Snapshot{}, source.LocalPathLocator{}, fmt.Errorf("%w: source belongs to another mission", ErrInvalidInput)
	}
	if snapshot.State.Removed {
		return sourcecontract.Snapshot{}, source.LocalPathLocator{}, fmt.Errorf("%w: source is removed", ErrInvalidInput)
	}
	if snapshot.Access.RetrievalPolicy != sourcecontract.RetrievalPolicyLiveReference || snapshot.Connector.ConnectorType != sourcecontract.ConnectorTypeLocalPath {
		return sourcecontract.Snapshot{}, source.LocalPathLocator{}, fmt.Errorf("%w: source is not a local path live reference", ErrInvalidInput)
	}
	locator, err := source.ParseLocalPathLocator(snapshot.Locators)
	if err != nil {
		return sourcecontract.Snapshot{}, source.LocalPathLocator{}, err
	}
	return snapshot, locator, nil
}

func localPathSourceTarget(locator source.LocalPathLocator, subpath string) (string, string, error) {
	if strings.TrimSpace(subpath) != "" && locator.PathKind == "file" {
		return "", "", fmt.Errorf("%w: subpath is only valid for directory local_path sources", ErrInvalidInput)
	}
	target, cleanSubpath, err := sourcecontract.LocalPathTargetRelativePath(locator.RelativePath, subpath)
	if err != nil {
		return "", "", localPathErr(err)
	}
	return target, cleanSubpath, nil
}

func isNilLocalPathReader(reader sourcecontract.LocalPathReader) bool {
	if reader == nil {
		return true
	}
	value := reflect.ValueOf(reader)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func (s *Service) localPathEngine() (sourcecontract.LocalPathReader, error) {
	if isNilLocalPathReader(s.localPaths) {
		return nil, fmt.Errorf("%w: local path roots are not configured", ErrInvalidInput)
	}
	return s.localPaths, nil
}

func defaultProducer(producer ledger.Producer) ledger.Producer {
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

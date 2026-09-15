package app

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"strings"
	"time"
)

// RemoveSource는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
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
	event, err := s.AppendEvent(ctx, ledger.AppendRequest{
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

// RestoreSource는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
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
	event, err := s.AppendEvent(ctx, ledger.AppendRequest{
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

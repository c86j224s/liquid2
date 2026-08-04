package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/mission"
)

const (
	MissionLifecycleActive   = mission.LifecycleActive
	MissionLifecycleArchived = mission.LifecycleArchived

	MissionArchivedEvent = mission.ArchivedEvent
	MissionRestoredEvent = mission.RestoredEvent
)

// MissionLifecycleChangeRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type MissionLifecycleChangeRequest struct {
	EventID   string
	MissionID string
	Producer  Producer
	Reason    string
}

// MissionLifecycleChangeResult는 archive/restore 이벤트와 갱신된 미션을 함께 반환한다.
type MissionLifecycleChangeResult struct {
	Event      *LedgerEvent      `json:"event,omitempty"`
	Projection MissionProjection `json:"projection"`
	Idempotent bool              `json:"idempotent,omitempty"`
}

// ArchiveMission는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) ArchiveMission(ctx context.Context, req MissionLifecycleChangeRequest) (MissionLifecycleChangeResult, error) {
	return s.changeMissionLifecycle(ctx, req, MissionLifecycleArchived, MissionArchivedEvent)
}

// RestoreMission는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) RestoreMission(ctx context.Context, req MissionLifecycleChangeRequest) (MissionLifecycleChangeResult, error) {
	return s.changeMissionLifecycle(ctx, req, MissionLifecycleActive, MissionRestoredEvent)
}

func (s *Service) changeMissionLifecycle(ctx context.Context, req MissionLifecycleChangeRequest, targetState, eventType string) (MissionLifecycleChangeResult, error) {
	if err := validateID("evt_", req.EventID); err != nil {
		return MissionLifecycleChangeResult{}, err
	}
	if err := validateID("mis_", req.MissionID); err != nil {
		return MissionLifecycleChangeResult{}, err
	}
	if req.Producer.Type != "user" {
		return MissionLifecycleChangeResult{}, fmt.Errorf("%w: mission lifecycle updates require a user producer", ErrInvalidInput)
	}
	payload, err := json.Marshal(map[string]any{
		"lifecycle_state": targetState,
		"reason":          strings.TrimSpace(req.Reason),
	})
	if err != nil {
		return MissionLifecycleChangeResult{}, err
	}
	var idempotent bool
	appended, err := s.appendLedgerEventsConditionally(ctx, req.MissionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
		if len(events) == 0 {
			return nil, fmt.Errorf("%w: mission does not exist", ErrInvalidInput)
		}
		projection, err := BuildProjection(req.MissionID, events)
		if err != nil {
			return nil, err
		}
		if normalizeMissionLifecycleState(projection.LifecycleState) == targetState {
			idempotent = true
			return nil, nil
		}
		if err := validateNoActiveAgentWork(events); err != nil {
			return nil, err
		}
		event, err := buildLedgerEvent(AppendEventRequest{
			EventID:   req.EventID,
			MissionID: req.MissionID,
			EventType: eventType,
			Producer:  req.Producer,
			Payload:   payload,
		})
		if err != nil {
			return nil, err
		}
		return []LedgerEvent{event}, nil
	})
	if err != nil {
		return MissionLifecycleChangeResult{}, err
	}
	projection, err := s.RebuildProjection(ctx, req.MissionID)
	if err != nil {
		return MissionLifecycleChangeResult{}, err
	}
	result := MissionLifecycleChangeResult{Projection: projection, Idempotent: idempotent}
	if len(appended) > 0 {
		result.Event = &appended[0]
	}
	return result, nil
}

func normalizeMissionLifecycleState(value string) string {
	switch strings.TrimSpace(value) {
	case MissionLifecycleArchived:
		return MissionLifecycleArchived
	default:
		return MissionLifecycleActive
	}
}

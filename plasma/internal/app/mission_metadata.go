package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// UpdateMissionMetadataRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type UpdateMissionMetadataRequest struct {
	EventID   string
	MissionID string
	Producer  Producer
	Title     *string
	Objective *string
	Scope     *MissionScope
}

// UpdateMissionMetadataResult는 미션 metadata 변경 이벤트와 갱신된 미션을 함께 반환한다.
type UpdateMissionMetadataResult struct {
	Event      LedgerEvent       `json:"event"`
	Projection MissionProjection `json:"projection"`
}

// UpdateMissionMetadata는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) UpdateMissionMetadata(ctx context.Context, req UpdateMissionMetadataRequest) (UpdateMissionMetadataResult, error) {
	if req.Title == nil && req.Objective == nil && req.Scope == nil {
		return UpdateMissionMetadataResult{}, fmt.Errorf("%w: at least one metadata field is required", ErrInvalidInput)
	}
	if req.Producer.Type != "user" {
		return UpdateMissionMetadataResult{}, fmt.Errorf("%w: metadata updates require a user producer", ErrInvalidInput)
	}
	payload := make(map[string]any, 3)
	if req.Title != nil {
		value := strings.TrimSpace(*req.Title)
		if value == "" {
			return UpdateMissionMetadataResult{}, fmt.Errorf("%w: title must not be blank", ErrInvalidInput)
		}
		payload["title"] = value
	}
	if req.Objective != nil {
		payload["objective"] = strings.TrimSpace(*req.Objective)
	}
	if req.Scope != nil {
		payload["scope"] = normalizeMissionScope(*req.Scope)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return UpdateMissionMetadataResult{}, err
	}
	event, err := s.AppendEvent(ctx, AppendEventRequest{
		EventID: req.EventID, MissionID: req.MissionID, EventType: "mission.metadata.updated",
		Producer: req.Producer, Payload: body,
	})
	if err != nil {
		return UpdateMissionMetadataResult{}, err
	}
	projection, err := s.RebuildProjection(ctx, req.MissionID)
	if err != nil {
		return UpdateMissionMetadataResult{}, err
	}
	return UpdateMissionMetadataResult{Event: event, Projection: projection}, nil
}

func normalizeMissionScope(scope MissionScope) MissionScope {
	return MissionScope{Included: trimEntries(scope.Included), Excluded: trimEntries(scope.Excluded)}
}

func trimEntries(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

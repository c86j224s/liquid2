package app

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/mission"
)

// UpdateMissionMetadata applies the mission-owned metadata policy, appends the
// resulting event, and rebuilds the projection in that order.
func (s *Service) UpdateMissionMetadata(ctx context.Context, req mission.UpdateMissionMetadataRequest) (mission.UpdateMissionMetadataResult, error) {
	appendReq, err := mission.BuildMetadataUpdate(req)
	if err != nil {
		return mission.UpdateMissionMetadataResult{}, err
	}
	event, err := s.AppendEvent(ctx, appendReq)
	if err != nil {
		return mission.UpdateMissionMetadataResult{}, err
	}
	projection, err := s.RebuildProjection(ctx, req.MissionID)
	if err != nil {
		return mission.UpdateMissionMetadataResult{}, err
	}
	return mission.UpdateMissionMetadataResult{Event: event, Projection: projection}, nil
}

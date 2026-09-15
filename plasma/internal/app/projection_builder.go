package app

import (
	"context"
	"slices"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
)

func (s *Service) RebuildProjection(ctx context.Context, missionID string) (mission.Projection, error) {
	events, err := s.ListEvents(ctx, missionID)
	if err != nil {
		return mission.Projection{}, err
	}
	projection, err := BuildProjection(missionID, events)
	if err != nil {
		return mission.Projection{}, err
	}
	if err := s.store.SaveMissionProjection(ctx, projection); err != nil {
		return mission.Projection{}, err
	}
	return projection, nil
}

func (s *Service) GetProjection(ctx context.Context, missionID string) (mission.Projection, error) {
	if err := validateID("mis_", missionID); err != nil {
		return mission.Projection{}, err
	}
	return s.store.GetMissionProjection(ctx, missionID)
}

// BuildProjection preserves the app facade while mission owns the projection rule.
func BuildProjection(missionID string, events []ledger.Event) (mission.Projection, error) {
	return mission.BuildProjection(missionID, events)
}

func isApprovalProducer(producer ledger.Producer) bool {
	return producer.Type == "user" || producer.Type == "steering_chat"
}

func isEmptyScope(scope mission.Scope) bool {
	return len(scope.Included) == 0 && len(scope.Excluded) == 0
}

func equalScope(left, right mission.Scope) bool {
	return slices.Equal(left.Included, right.Included) && slices.Equal(left.Excluded, right.Excluded)
}

func addUnique(values *[]string, value string) {
	if value == "" {
		return
	}
	for _, existing := range *values {
		if existing == value {
			return
		}
	}
	*values = append(*values, value)
}

func removeValue(values *[]string, value string) {
	if value == "" {
		return
	}
	next := (*values)[:0]
	for _, existing := range *values {
		if existing != value {
			next = append(next, existing)
		}
	}
	*values = next
}

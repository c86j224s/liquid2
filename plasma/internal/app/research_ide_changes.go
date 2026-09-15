package app

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/researchcatalog"
	"strings"
)

// ListMissionChanges returns meaningful mission changes after a caller-owned
// ledger sequence. It never returns event payload bodies; callers can inspect a
// selected ledger_event through the existing bounded research read path.
func (s *Service) ListMissionChanges(ctx context.Context, req ResearchIDEChangesRequest) (ResearchIDEChanges, error) {
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return ResearchIDEChanges{}, err
	}
	if req.AfterSequence < 0 {
		return ResearchIDEChanges{}, fmt.Errorf("%w: after sequence must be non-negative", ErrInvalidInput)
	}
	limit := researchcatalog.ClampLimit(req.Limit)
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return ResearchIDEChanges{}, err
	}
	currentSequence := researchIDELastSequence(events)
	result := ResearchIDEChanges{
		MissionID:         missionID,
		AfterSequence:     req.AfterSequence,
		CurrentSequence:   currentSequence,
		Items:             []researchcatalog.ObjectSummary{},
		NextAfterSequence: currentSequence,
		Limit:             limit,
	}
	if req.AfterSequence > currentSequence {
		result.NextAfterSequence = 0
		result.ResyncRequired = true
		return result, nil
	}

	changes := make([]researchcatalog.ObjectSummary, 0, limit)
	var lastIncludedSequence int64
	for _, event := range researchcatalog.VisibleLedgerEvents(events, false) {
		if event.Sequence <= req.AfterSequence || !researchIDEMeaningfulChange(event) {
			continue
		}
		if len(changes) == limit {
			result.Items = changes
			result.NextAfterSequence = lastIncludedSequence
			result.Truncated = true
			return result, nil
		}
		changes = append(changes, researchcatalog.SummarizeLedgerEvent(event))
		lastIncludedSequence = event.Sequence
	}
	result.Items = changes
	return result, nil
}

func researchIDELastSequence(events []ledger.Event) int64 {
	var last int64
	for _, event := range events {
		if event.Sequence > last {
			last = event.Sequence
		}
	}
	return last
}

// researchIDEMeaningfulChange excludes execution mechanics that a resumed
// provider session already knows. All other non-report domain events remain
// visible so new capabilities enter the feed without another allowlist change.
func researchIDEMeaningfulChange(event ledger.Event) bool {
	eventType := strings.TrimSpace(event.EventType)
	if eventType == "turn.user" {
		return strings.TrimSpace(event.Producer.Type) == "user"
	}
	for _, prefix := range []string{"mcp.", "workflow.", "turn.agent.", "controller.", "agent.session."} {
		if strings.HasPrefix(eventType, prefix) {
			return false
		}
	}
	return eventType != ""
}

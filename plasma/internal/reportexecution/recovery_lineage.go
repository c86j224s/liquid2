package reportexecution

import (
	"encoding/json"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// ReportRecoveryLineage resolves the pending-event chain used to replay a report attempt.
func ReportRecoveryLineage(events []ledger.Event, pendingID string) ([]string, error) {
	type pending struct{ Origin, Parent, Strategy string }
	pendingByID := map[string]pending{}
	for _, event := range events {
		if event.EventType != "report.draft.pending" {
			continue
		}
		var p struct {
			Origin   string `json:"origin_pending_event_id"`
			Parent   string `json:"retry_of_pending_event_id"`
			Strategy string `json:"retry_strategy"`
		}
		if json.Unmarshal(event.Payload, &p) != nil {
			return nil, fmt.Errorf("%w: invalid report attempt", producterror.ErrInvalidInput)
		}
		if p.Origin == "" {
			p.Origin = event.EventID
		}
		pendingByID[event.EventID] = pending{p.Origin, p.Parent, p.Strategy}
	}
	current, ok := pendingByID[pendingID]
	if !ok {
		return nil, fmt.Errorf("%w: report attempt missing", producterror.ErrInvalidInput)
	}
	if current.Strategy == "restart" {
		parent, ok := pendingByID[current.Parent]
		if current.Parent == "" || !ok || parent.Origin != current.Origin {
			return nil, fmt.Errorf("%w: invalid report restart lineage", producterror.ErrInvalidInput)
		}
		return []string{pendingID}, nil
	}
	if current.Parent == "" {
		if current.Origin != pendingID {
			return nil, fmt.Errorf("%w: invalid report root lineage", producterror.ErrInvalidInput)
		}
		return []string{pendingID}, nil
	}
	chain := []string{}
	seen := map[string]bool{}
	for depth := 0; depth < 64; depth++ {
		if seen[pendingID] {
			return nil, fmt.Errorf("%w: report lineage cycle", producterror.ErrInvalidInput)
		}
		seen[pendingID] = true
		item, ok := pendingByID[pendingID]
		if !ok {
			return nil, fmt.Errorf("%w: report lineage ancestor missing", producterror.ErrInvalidInput)
		}
		if item.Origin != current.Origin {
			return nil, fmt.Errorf("%w: report lineage origin mismatch", producterror.ErrInvalidInput)
		}
		chain = append([]string{pendingID}, chain...)
		if item.Strategy == "restart" {
			return chain, nil
		}
		if item.Parent == "" {
			return chain, nil
		}
		pendingID = item.Parent
	}
	return nil, fmt.Errorf("%w: report lineage too deep", producterror.ErrInvalidInput)
}

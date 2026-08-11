package requirements

import (
	"encoding/json"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

type pendingLineage struct{ Origin, Parent, Strategy string }

// recoveryLineage는 requirements stage가 재사용할 수 있는 pending event ID 순서를 만든다.
// retry 정책은 plan stage와 같은 durable attempt 계약을 따른다.
func recoveryLineage(events []ledger.Event, pendingID string) ([]string, error) {
	pendingByID := map[string]pendingLineage{}
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
		pendingByID[event.EventID] = pendingLineage{p.Origin, p.Parent, p.Strategy}
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
	return ancestorLineage(pendingByID, current.Origin, pendingID)
}

func ancestorLineage(pendingByID map[string]pendingLineage, origin string, pendingID string) ([]string, error) {
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
		if item.Origin != origin {
			return nil, fmt.Errorf("%w: report lineage origin mismatch", producterror.ErrInvalidInput)
		}
		chain = append([]string{pendingID}, chain...)
		if item.Strategy == "restart" || item.Parent == "" {
			return chain, nil
		}
		pendingID = item.Parent
	}
	return nil, fmt.Errorf("%w: report lineage too deep", producterror.ErrInvalidInput)
}

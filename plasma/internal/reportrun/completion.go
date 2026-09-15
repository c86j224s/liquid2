package reportrun

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/reportusage"
)

const (
	ReportRunCompletedEventType = "report.run.completed"
	ReportRunCompletionSchema   = "plasma.report_run_completion.v1"
	ReportRunCompletionReason   = "provider usage was lost before durable report completion"
)

type ReportCompletionStore interface {
	AppendEventsConditionally(context.Context, string, func([]ledger.Event) ([]ledger.AppendRequest, error)) ([]ledger.Event, error)
	ListEvents(context.Context, string) ([]ledger.Event, error)
}

type ReportCompletionMissionStore interface {
	ReportCompletionStore
	ListMissionsWithState(context.Context, mission.ListRequest) ([]mission.Mission, error)
}

type ReportCompletionRequest struct {
	MissionID        string
	CanonicalEventID string
	ActualUsage      *reportusage.ReportAgentUsageRequest
}

type completionTarget struct {
	Event   ledger.Event
	Pending ledger.Event
	Meta    reportusage.Target
}

type completionState struct {
	Root      string
	Canonical ledger.Event
	Pending   ledger.Event
	Targets   []completionTarget
}

// CompleteReportRun conditionally records all target outcomes and the completion
// boundary from one latest ledger snapshot.
func CompleteReportRun(ctx context.Context, store ReportCompletionStore, req ReportCompletionRequest) (ledger.Event, error) {
	event, _, err := completeReportRun(ctx, store, req)
	return event, err
}

func completeReportRun(ctx context.Context, store ReportCompletionStore, req ReportCompletionRequest) (ledger.Event, bool, error) {
	missionID := strings.TrimSpace(req.MissionID)
	canonicalID := strings.TrimSpace(req.CanonicalEventID)
	if missionID == "" || canonicalID == "" {
		return ledger.Event{}, false, fmt.Errorf("report completion requires mission and canonical event")
	}
	var replay ledger.Event
	var created bool
	appended, err := store.AppendEventsConditionally(ctx, missionID, func(events []ledger.Event) ([]ledger.AppendRequest, error) {
		state, err := findCompletionState(events, canonicalID)
		if err != nil {
			return nil, err
		}
		completionID := completionEventID(state.Root)
		if existing, ok := eventByID(events, completionID); ok {
			if err := validateExistingCompletion(existing, state.Root, events); err != nil {
				return nil, err
			}
			replay = existing
			return nil, nil
		}
		reqs, err := completionRequests(state, events, req.ActualUsage)
		if err != nil {
			return nil, err
		}
		created = true
		return reqs, nil
	})
	if err != nil {
		return ledger.Event{}, false, err
	}
	if replay.EventID != "" {
		return replay, false, nil
	}
	var completion ledger.Event
	for _, event := range appended {
		if event.EventType != ReportRunCompletedEventType {
			continue
		}
		if completion.EventID != "" {
			return ledger.Event{}, false, fmt.Errorf("report completion batch contains multiple completion events")
		}
		completion = event
	}
	if completion.EventID == "" {
		return ledger.Event{}, false, fmt.Errorf("report completion was not appended")
	}
	return completion, created, nil
}

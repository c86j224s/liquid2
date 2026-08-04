package reportexecution

import (
	"context"
	"strings"
)

func (runner Runner) hasTerminalEvent(ctx context.Context, missionID string, pendingEventID string) bool {
	events, err := runner.Service.ListEvents(ctx, missionID)
	if err != nil {
		return false
	}
	_, ok := CompletedPendingEventIDs(events)[strings.TrimSpace(pendingEventID)]
	return ok
}

func (runner Runner) isSamePendingAlreadyRunning(missionID string, pendingEventID string) bool {
	if runner.InFlight == nil {
		return false
	}
	current, ok := runner.InFlight.PendingEventID(missionID)
	return ok && strings.TrimSpace(current) == strings.TrimSpace(pendingEventID)
}

func (runner Runner) id(prefix string) string {
	if runner.NewID == nil {
		return prefix + "_report"
	}
	return runner.NewID(prefix)
}

package app

import (
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
)

const (
	turnAgentPendingEvent  = "turn.agent.pending"
	turnAgentResponseEvent = "turn.agent.response"
)

func MissionActivityEventTypes() []string {
	return mission.ActivityEventTypes()
}

func MissionActivityFromEvents(events []ledger.Event) mission.ActivitySummary {
	return mission.ActivityFromEvents(events)
}

func MissionActivityFromInput(input mission.ActivityInput) mission.ActivitySummary {
	return mission.ActivityFromInput(input)
}

func terminalActivityFromEvent(event ledger.Event) (mission.TerminalActivityView, bool) {
	return mission.TerminalActivityFromEvent(event)
}

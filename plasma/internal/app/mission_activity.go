package app

import "github.com/c86j224s/liquid2/plasma/internal/mission"

const (
	turnAgentPendingEvent  = "turn.agent.pending"
	turnAgentResponseEvent = "turn.agent.response"
)

func MissionActivityEventTypes() []string {
	return mission.ActivityEventTypes()
}

func MissionActivityFromEvents(events []LedgerEvent) MissionActivitySummary {
	return mission.ActivityFromEvents(events)
}

func MissionActivityFromInput(input MissionActivityInput) MissionActivitySummary {
	return mission.ActivityFromInput(input)
}

func terminalActivityFromEvent(event LedgerEvent) (TerminalActivityView, bool) {
	return mission.TerminalActivityFromEvent(event)
}

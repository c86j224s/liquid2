package app

import "github.com/c86j224s/liquid2/plasma/internal/ledgerstate"

// ReportProgress는 transport 계층이 소비하는 타입화된 애플리케이션 read model이다.
type ReportProgress = ledgerstate.ReportProgress

// ReportProgressFromEvents는 장부 이벤트를 report 진행 상태 view로 투영한다.
func ReportProgressFromEvents(events []LedgerEvent) ReportProgress {
	stateEvents := make([]ledgerstate.Event, 0, len(events))
	for _, event := range events {
		stateEvents = append(stateEvents, ledgerstate.Event{
			EventID: event.EventID, Sequence: event.Sequence, EventType: event.EventType,
			Payload: event.Payload, CreatedAt: event.CreatedAt,
		})
	}
	return ledgerstate.ProjectReportProgress(stateEvents)
}

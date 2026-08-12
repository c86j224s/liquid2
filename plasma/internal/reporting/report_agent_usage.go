package reporting

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const ReportAgentUsageRecordedEventType = "report.agent_usage.recorded"

// ReportAgentUsageStore는 canonical 보고서 결과 뒤에 늦게 확인된 provider usage를
// 중복 없이 기록하는 장부 포트다.
type ReportAgentUsageStore interface {
	AppendEventConditionally(context.Context, string, func([]ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error)) (ledger.Event, bool, error)
}

// ReportAgentUsageRequest는 본문을 보존하지 않는 보고서 agent 호출 계측값이다.
// CanonicalEventID는 이 호출이 만든 durable 결과를 가리키며 이벤트 ID의 기준이 된다.
type ReportAgentUsageRequest struct {
	MissionID                string
	PendingEventID           string
	CanonicalEventID         string
	ForkSourceAgentSessionID string
	Surface                  string
	PreviousAgentSessionID   string
	AgentSessionID           string
	DurationMS               int64
	Resumed                  bool
	Usage                    agentusage.AgentUsage
}

// RecordReportAgentUsage는 canonical 결과 뒤에 확인된 usage를 별도 이벤트로 기록한다.
// 같은 canonical 이벤트에 대한 재호출은 기존 이벤트를 반환하며 보고서 상태를 바꾸지 않는다.
func RecordReportAgentUsage(ctx context.Context, store ReportAgentUsageStore, req ReportAgentUsageRequest) (ledger.Event, bool, error) {
	request, ok, err := buildReportAgentUsageAppendRequest(req)
	if err != nil || !ok {
		return ledger.Event{}, false, err
	}
	event, _, err := store.AppendEventConditionally(ctx, request.MissionID, func(events []ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error) {
		for _, event := range events {
			if event.EventID != request.EventID {
				continue
			}
			if !sameReportAgentUsageEvent(event, request) {
				return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("%w: report agent usage event differs from existing record", producterror.ErrConflict)
			}
			return ledger.AppendRequest{}, event, false, nil
		}
		return request, ledger.Event{}, true, nil
	})
	if err != nil {
		return ledger.Event{}, false, err
	}
	return event, true, nil
}

func buildReportAgentUsageAppendRequest(req ReportAgentUsageRequest) (ledger.AppendRequest, bool, error) {
	missionID := strings.TrimSpace(req.MissionID)
	pendingEventID := strings.TrimSpace(req.PendingEventID)
	canonicalEventID := strings.TrimSpace(req.CanonicalEventID)
	agentSessionID := strings.TrimSpace(req.AgentSessionID)
	if missionID == "" || pendingEventID == "" || canonicalEventID == "" || agentSessionID == "" {
		return ledger.AppendRequest{}, false, fmt.Errorf("%w: report usage lineage is incomplete", producterror.ErrInvalidInput)
	}
	eventUsage, ok := req.Usage.ForEvent(req.Surface, req.DurationMS, req.PreviousAgentSessionID, agentSessionID, req.Resumed, false)
	if !ok {
		return ledger.AppendRequest{}, false, nil
	}
	payload := map[string]any{
		"kind":                 "report_agent_usage",
		"pending_event_id":     pendingEventID,
		"correlation_event_id": canonicalEventID,
		"agent_usage":          eventUsage,
	}
	if forkSourceID := strings.TrimSpace(req.ForkSourceAgentSessionID); forkSourceID != "" {
		payload["fork_source_agent_session_id"] = forkSourceID
	}
	return ledger.AppendRequest{
		EventID:          reportAgentUsageEventID(canonicalEventID),
		MissionID:        missionID,
		EventType:        ReportAgentUsageRecordedEventType,
		Producer:         ledger.Producer{Type: "agent_session", ID: agentSessionID},
		CausationEventID: canonicalEventID,
		CorrelationID:    pendingEventID,
		Payload:          mustJSON(payload),
	}, true, nil
}

func reportAgentUsageEventID(canonicalEventID string) string {
	return "evt_report_usage_" + strings.TrimPrefix(strings.TrimSpace(canonicalEventID), "evt_")
}

func sameReportAgentUsageEvent(event ledger.Event, request ledger.AppendRequest) bool {
	return event.MissionID == request.MissionID &&
		event.EventType == request.EventType &&
		event.Producer == request.Producer &&
		event.CausationEventID == request.CausationEventID &&
		event.CorrelationID == request.CorrelationID &&
		bytes.Equal(event.Payload, request.Payload)
}

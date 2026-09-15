package reportrun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportusage"
)

func completionRequests(state completionState, events []ledger.Event, actual *reportusage.ReportAgentUsageRequest) ([]ledger.AppendRequest, error) {
	usageByTarget, err := validateUsageEvents(state, events)
	if err != nil {
		return nil, err
	}
	requests := make([]ledger.AppendRequest, 0, len(state.Targets)+1)
	actualTarget := ""
	if actual != nil {
		actualTarget = strings.TrimSpace(actual.CanonicalEventID)
		known := false
		for _, target := range state.Targets {
			if target.Event.EventID == actualTarget {
				known = true
				break
			}
		}
		if !known {
			return nil, fmt.Errorf("actual report usage does not match a completion target")
		}
	}
	actualRecorded := false
	actualUnavailable := false
	for _, target := range state.Targets {
		if _, ok := usageByTarget[target.Event.EventID]; ok {
			if actualTarget == target.Event.EventID && actual != nil {
				request, err := actualUsageAppendRequest(state, target, *actual)
				if err != nil {
					return nil, err
				}
				if existing, found := eventByID(events, request.EventID); found && (existing.EventType != request.EventType || existing.Producer != request.Producer || existing.CausationEventID != request.CausationEventID || existing.CorrelationID != request.CorrelationID || !bytes.Equal(existing.Payload, request.Payload)) {
					return nil, fmt.Errorf("actual report usage conflicts with existing record")
				}
			}
			continue
		}
		if actualTarget == target.Event.EventID {
			request, err := actualUsageAppendRequest(state, target, *actual)
			if err != nil {
				return nil, err
			}
			requests = append(requests, request)
			var usagePayload struct {
				AgentUsage agentusage.AgentUsage `json:"agent_usage"`
			}
			if err := json.Unmarshal(request.Payload, &usagePayload); err != nil {
				return nil, err
			}
			actualRecorded = usagePayload.AgentUsage.ProviderUsage != nil && !usagePayload.AgentUsage.UsageUnavailable
			actualUnavailable = !actualRecorded
			continue
		}
		request, err := unavailableUsageRequest(state, target)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	usageRecorded := 0
	usageUnavailable := 0
	for _, target := range state.Targets {
		if usage, ok := usageByTarget[target.Event.EventID]; ok {
			if usage.ProviderUsage != nil && !usage.UsageUnavailable {
				usageRecorded++
			} else {
				usageUnavailable++
			}
			continue
		}
		if actualRecorded && actualTarget == target.Event.EventID {
			usageRecorded++
		} else if actualUnavailable && actualTarget == target.Event.EventID {
			usageUnavailable++
		} else {
			usageUnavailable++
		}
	}
	if usageRecorded+usageUnavailable != len(state.Targets) {
		return nil, fmt.Errorf("report completion has incomplete usage outcomes")
	}
	requests = append(requests, ledger.AppendRequest{
		EventID:          completionEventID(state.Root),
		MissionID:        state.Canonical.MissionID,
		EventType:        ReportRunCompletedEventType,
		Producer:         ledger.Producer{Type: "system", ID: "report-completion"},
		CausationEventID: state.Canonical.EventID,
		CorrelationID:    state.Root,
		Payload:          completionPayload(state, usageRecorded, usageUnavailable),
	})
	return requests, nil
}

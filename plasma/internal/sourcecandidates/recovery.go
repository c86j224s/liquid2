package sourcecandidates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const interruptedStagingMessage = "Plasma가 다시 시작되어 후보 원문 가져오기를 완료하지 못했습니다. 후보를 다시 제안해 주세요."

type interruptedStagingStore interface {
	ListMissionsWithState(context.Context, mission.ListRequest) ([]mission.Mission, error)
	ListEvents(context.Context, string) ([]ledger.Event, error)
	AppendEventConditionally(context.Context, string, func([]ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error)) (ledger.Event, bool, error)
}

// FailInterruptedStaging closes candidate fetches left open by an earlier
// process. Each started event receives at most one technical failure terminal;
// existing staged or failed terminals and user decisions remain unchanged.
func FailInterruptedStaging(ctx context.Context, store interruptedStagingStore, newEventID SourceCandidateIDFunc) (int, error) {
	if newEventID == nil {
		return 0, fmt.Errorf("%w: source candidate event id generator is required", producterror.ErrInvalidInput)
	}
	missions, err := store.ListMissionsWithState(ctx, mission.ListRequest{IncludeArchived: true})
	if err != nil {
		return 0, fmt.Errorf("list missions for source candidate recovery: %w", err)
	}

	closed := 0
	var recoveryErrors []error
	for _, currentMission := range missions {
		events, listErr := store.ListEvents(ctx, currentMission.MissionID)
		if listErr != nil {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("list source candidate events for %s: %w", currentMission.MissionID, listErr))
			continue
		}
		starts, scanErr := openStagingStarts(events)
		if scanErr != nil {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("scan source candidate events for %s: %w", currentMission.MissionID, scanErr))
			continue
		}
		for _, started := range starts {
			_, appended, closeErr := failInterruptedStart(ctx, store, started, newEventID("evt"))
			if closeErr != nil {
				recoveryErrors = append(recoveryErrors, fmt.Errorf("close interrupted source candidate %s: %w", started.EventID, closeErr))
				continue
			}
			if appended {
				closed++
			}
		}
	}
	return closed, errors.Join(recoveryErrors...)
}

func failInterruptedStart(ctx context.Context, store interruptedStagingStore, started ledger.Event, eventID string) (ledger.Event, bool, error) {
	return store.AppendEventConditionally(ctx, started.MissionID, func(events []ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error) {
		current, ok := eventByID(events, started.EventID)
		if !ok || current.EventType != "source.candidate.staging_started" {
			return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("staging start event is missing")
		}
		terminal, found, err := stagingTerminal(events, started.EventID)
		if err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if found {
			return ledger.AppendRequest{}, terminal, false, nil
		}
		job, err := interruptedStagingJob(current)
		if err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		return sourceCandidateStagingFailedEventRequest(job, eventID, errors.New(interruptedStagingMessage)), ledger.Event{}, true, nil
	})
}

func openStagingStarts(events []ledger.Event) ([]ledger.Event, error) {
	terminalByStart := make(map[string]struct{})
	for _, event := range events {
		if event.EventType != "source.candidate.staged" && event.EventType != "source.candidate.staging_failed" {
			continue
		}
		startedID, err := terminalStagingEventID(event)
		if err != nil {
			return nil, err
		}
		if startedID != "" {
			terminalByStart[startedID] = struct{}{}
		}
	}
	starts := make([]ledger.Event, 0)
	for _, event := range events {
		if event.EventType != "source.candidate.staging_started" {
			continue
		}
		if _, closed := terminalByStart[event.EventID]; !closed {
			starts = append(starts, event)
		}
	}
	return starts, nil
}

func stagingTerminal(events []ledger.Event, startedID string) (ledger.Event, bool, error) {
	for _, event := range events {
		if event.EventType != "source.candidate.staged" && event.EventType != "source.candidate.staging_failed" {
			continue
		}
		candidateStartedID, err := terminalStagingEventID(event)
		if err != nil {
			return ledger.Event{}, false, err
		}
		if candidateStartedID == startedID {
			return event, true, nil
		}
	}
	return ledger.Event{}, false, nil
}

func terminalStagingEventID(event ledger.Event) (string, error) {
	var payload struct {
		StagingEventID string `json:"staging_event_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return "", fmt.Errorf("decode terminal event %s: %w", event.EventID, err)
	}
	return strings.TrimSpace(payload.StagingEventID), nil
}

func interruptedStagingJob(event ledger.Event) (SourceCandidateStagingJob, error) {
	var payload struct {
		CandidateKind   string `json:"candidate_kind"`
		URL             string `json:"url"`
		Title           string `json:"title"`
		ProposalEventID string `json:"proposal_event_id"`
		AgentExecutor   string `json:"agent_executor"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return SourceCandidateStagingJob{}, fmt.Errorf("decode staging start event %s: %w", event.EventID, err)
	}
	if strings.TrimSpace(payload.URL) == "" {
		return SourceCandidateStagingJob{}, fmt.Errorf("staging start event %s has no URL", event.EventID)
	}
	return SourceCandidateStagingJob{
		MissionID:                         event.MissionID,
		SessionID:                         event.CorrelationID,
		ProposalEventID:                   strings.TrimSpace(payload.ProposalEventID),
		CandidateKind:                     strings.TrimSpace(payload.CandidateKind),
		Candidate:                         SourceCandidateProposal{URL: strings.TrimSpace(payload.URL), Title: strings.TrimSpace(payload.Title)},
		Producer:                          ledger.Producer{Type: "system", ID: "plasma-startup"},
		StartedEventID:                    event.EventID,
		AgentExecutor:                     strings.TrimSpace(payload.AgentExecutor),
		EmitAgentExecutorInTerminalEvents: strings.TrimSpace(payload.AgentExecutor) != "",
	}, nil
}

func eventByID(events []ledger.Event, eventID string) (ledger.Event, bool) {
	for _, event := range events {
		if event.EventID == eventID {
			return event, true
		}
	}
	return ledger.Event{}, false
}

package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/workflowruns"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

// StopRunNow closes an open workflow agent turn and appends the stopped event
// in one durable append, or appends only the terminal event when no turn is open.
func (supervisor *Supervisor) StopRunNow(ctx context.Context, missionID string, workflowRunID string, reason string) (ledger.Event, error) {
	if supervisor == nil || supervisor.service == nil {
		return ledger.Event{}, nil
	}
	events, err := supervisor.service.ListEvents(ctx, missionID)
	if err != nil {
		return ledger.Event{}, err
	}
	if pending, ok := conversation.LatestOpenAgentPending(events, workflowRunID); ok {
		return supervisor.appendAgentCanceledWithTerminal(ctx, events, missionID, pending, reason, workflowstate.WorkflowRunStoppedEvent)
	}
	req, ok, err := supervisor.terminalRequest(events, workflowstate.WorkflowRunTerminalEventRequest{
		WorkflowRunID: workflowRunID,
		MissionID:     missionID,
		EventType:     workflowstate.WorkflowRunStoppedEvent,
		Reason:        reason,
	})
	if err != nil || !ok {
		return ledger.Event{}, err
	}
	return supervisor.appendFirst(ctx, missionID, []ledger.AppendRequest{req})
}

func (supervisor *Supervisor) closeOpenPending(ctx context.Context, run workflowstate.WorkflowRunView, events []ledger.Event) error {
	pending, ok := conversation.LatestOpenAgentPending(events, run.WorkflowRunID)
	if !ok {
		return nil
	}
	text := fmt.Sprintf("자동조사 실행자가 사라져 열린 대기 상태를 정리했습니다. workflow=%s status=%s", run.WorkflowRunID, run.Status)
	_, err := supervisor.appendAgentCanceledWithTerminal(ctx, events, run.MissionID, pending, text, "")
	return err
}

func (supervisor *Supervisor) appendAgentCanceledWithTerminal(ctx context.Context, events []ledger.Event, missionID string, pending conversation.OpenAgentPending, text string, terminalEventType string) (ledger.Event, error) {
	executor := firstNonEmpty(strings.TrimSpace(pending.AgentExecutor), "codex")
	reqs := make([]ledger.AppendRequest, 0, 2)
	if !conversation.HasAgentTerminalEventForUser(events, pending.UserEventID) {
		extra := map[string]any{"canceled_at": time.Now().UTC().Format(time.RFC3339Nano)}
		if strings.TrimSpace(pending.WorkflowRunID) != "" {
			extra["workflow_run_id"] = pending.WorkflowRunID
		}
		if strings.TrimSpace(pending.WorkflowStepID) != "" {
			extra["workflow_step_id"] = pending.WorkflowStepID
		}
		reqs = append(reqs, conversation.BuildTurnAgentResponseAppendRequest(conversation.TurnAgentResponseEventRequest{
			EventID:       supervisor.nextID("evt"),
			MissionID:     missionID,
			Kind:          "agent_canceled",
			AgentExecutor: executor,
			Text:          text,
			UserEventID:   pending.UserEventID,
			Extra:         extra,
			Producer:      ledger.Producer{Type: "agent", ID: executor},
		}))
	}
	if strings.TrimSpace(terminalEventType) != "" && strings.TrimSpace(pending.WorkflowRunID) != "" {
		req, ok, err := supervisor.terminalRequest(events, workflowstate.WorkflowRunTerminalEventRequest{
			WorkflowRunID: pending.WorkflowRunID,
			MissionID:     missionID,
			EventType:     terminalEventType,
			Reason:        text,
		})
		if err != nil {
			return ledger.Event{}, err
		}
		if ok {
			reqs = append(reqs, req)
		}
	}
	if len(reqs) == 0 {
		return ledger.Event{}, nil
	}
	return supervisor.appendFirst(ctx, missionID, reqs)
}

func (supervisor *Supervisor) terminalRequest(events []ledger.Event, req workflowstate.WorkflowRunTerminalEventRequest) (ledger.AppendRequest, bool, error) {
	terminal, ok, err := workflowruns.BuildTerminalAppendRequest(workflowStateEvents(events), req, supervisor.nextID, time.Now().UTC())
	if err != nil {
		if errors.Is(err, workflowruns.ErrInvalidInput) {
			message := workflowruns.InvalidInputMessage(err)
			if message == "" {
				message = err.Error()
			}
			return ledger.AppendRequest{}, false, fmt.Errorf("%w: %s", producterror.ErrInvalidInput, message)
		}
		return ledger.AppendRequest{}, false, err
	}
	if !ok {
		return ledger.AppendRequest{}, false, nil
	}
	return ledger.AppendRequest{
		EventID:          terminal.EventID,
		MissionID:        terminal.MissionID,
		EventType:        terminal.EventType,
		Producer:         ledger.Producer{Type: terminal.Producer.Type, ID: terminal.Producer.ID},
		CausationEventID: terminal.CausationEventID,
		CorrelationID:    terminal.CorrelationID,
		Payload:          terminal.Payload,
	}, true, nil
}

func (supervisor *Supervisor) appendFirst(ctx context.Context, missionID string, reqs []ledger.AppendRequest) (ledger.Event, error) {
	appended, err := supervisor.service.AppendEvents(ctx, missionID, reqs)
	if err != nil || len(appended) == 0 {
		return ledger.Event{}, err
	}
	return appended[0], nil
}

func (supervisor *Supervisor) nextID(prefix string) string {
	if supervisor != nil && supervisor.newID != nil {
		return supervisor.newID(prefix)
	}
	return ""
}

func workflowStateEvents(events []ledger.Event) []workflowstate.Event {
	converted := make([]workflowstate.Event, 0, len(events))
	for _, event := range events {
		converted = append(converted, workflowstate.Event{
			EventID:   event.EventID,
			MissionID: event.MissionID,
			Sequence:  event.Sequence,
			EventType: event.EventType,
			Payload:   event.Payload,
			CreatedAt: event.CreatedAt,
		})
	}
	return converted
}

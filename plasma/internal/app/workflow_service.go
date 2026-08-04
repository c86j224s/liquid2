package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledgerstate"
	"github.com/c86j224s/liquid2/plasma/internal/workflowruns"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

const (
	DefaultWorkflowMaxSteps  = workflowruns.DefaultMaxSteps
	WorkflowInstructionLimit = workflowruns.InstructionLimit
	workflowStaleAfter       = workflowruns.StaleAfter
)

// RequestWorkflowRun은 워크플로우 실행 요청 이벤트를 장부에 남긴다. 실제 실행은 runner가 별도로 claim한다.
func (s *Service) RequestWorkflowRun(ctx context.Context, req RequestWorkflowRunRequest) (WorkflowRunView, error) {
	s.workflowMu.Lock()
	defer s.workflowMu.Unlock()

	view, err := workflowruns.RequestRun(ctx, workflowRunStore{service: s}, req, newAppID, time.Now().UTC())
	return view, translateWorkflowRunError(err)
}

// RequestWorkflowStop은 실행 중인 워크플로우에 대한 중단 요청 이벤트를 기록한다.
func (s *Service) RequestWorkflowStop(ctx context.Context, req RequestWorkflowStopRequest) (WorkflowRunView, error) {
	s.workflowMu.Lock()
	defer s.workflowMu.Unlock()

	view, err := workflowruns.RequestStop(ctx, workflowRunStore{service: s}, req, newAppID, time.Now().UTC())
	return view, translateWorkflowRunError(err)
}

// BuildWorkflowRunTerminalAppendRequest는 애플리케이션 서비스 계층에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildWorkflowRunTerminalAppendRequest(events []LedgerEvent, req WorkflowRunTerminalEventRequest) (AppendEventRequest, bool, error) {
	eventReq, ok, err := workflowruns.BuildTerminalAppendRequest(workflowEventsFromApp(events), req, newAppID, time.Now().UTC())
	if err != nil || !ok {
		return AppendEventRequest{}, ok, translateWorkflowRunError(err)
	}
	return workflowAppendRequestToApp(eventReq), true, nil
}

// ClaimWorkflowRunStart는 workflow run 시작 claim 결과를 app view로 변환한다.
func (s *Service) ClaimWorkflowRunStart(ctx context.Context, missionID string, workflowRunID string, startedAt time.Time) (WorkflowRunView, bool, error) {
	s.workflowMu.Lock()
	defer s.workflowMu.Unlock()

	view, claimed, err := workflowruns.ClaimStart(ctx, workflowRunStore{service: s}, missionID, workflowRunID, startedAt, newAppID)
	return view, claimed, translateWorkflowRunError(err)
}

// ListWorkflowRuns는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListWorkflowRuns(ctx context.Context, missionID string) ([]WorkflowRunView, error) {
	runs, err := workflowruns.ListRuns(ctx, workflowRunStore{service: s}, missionID)
	return runs, translateWorkflowRunError(err)
}

// GetWorkflowRun는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) GetWorkflowRun(ctx context.Context, missionID string, workflowRunID string) (WorkflowRunView, error) {
	view, err := workflowruns.GetRun(ctx, workflowRunStore{service: s}, missionID, workflowRunID)
	return view, translateWorkflowRunError(err)
}

func projectWorkflowRuns(events []LedgerEvent) []WorkflowRunView {
	return workflowstate.ProjectRuns(workflowEventsFromApp(events))
}

func workflowEventsFromApp(events []LedgerEvent) []workflowstate.Event {
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

func validateWorkflowEventPayload(eventType string, missionID string, payload json.RawMessage) error {
	return translateWorkflowRunError(workflowruns.ValidateEventPayload(strings.TrimSpace(eventType), strings.TrimSpace(missionID), payload))
}

func workflowHasOpenAgentPending(events []LedgerEvent) bool {
	return ledgerstate.HasOpenAgentPending(ledgerStateEventsFromApp(events))
}

func workflowHasAgentTerminalEventForUser(events []LedgerEvent, userEventID string) bool {
	return ledgerstate.HasAgentTerminalEventForUser(ledgerStateEventsFromApp(events), userEventID)
}

func validateWorkflowStartAfterEvent(events []LedgerEvent, startAfterEventID string) error {
	message := ledgerstate.ValidateWorkflowStartAfterEvent(ledgerStateEventsFromApp(events), startAfterEventID)
	if message == "" {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrInvalidInput, message)
}

type workflowRunStore struct {
	service *Service
}

// ListEvents는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (store workflowRunStore) ListEvents(ctx context.Context, missionID string) ([]workflowruns.Event, error) {
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	events, err := store.service.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return nil, err
	}
	return workflowEventsFromApp(events), nil
}

// AppendRequestsConditionally는 조건 함수가 승인한 append 요청만 장부에 반영한다.
func (store workflowRunStore) AppendRequestsConditionally(ctx context.Context, missionID string, build func([]workflowruns.Event) ([]workflowruns.AppendEventRequest, error)) ([]workflowruns.Event, error) {
	appended, err := store.service.appendLedgerEventsConditionally(ctx, missionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
		workflowRequests, err := build(workflowEventsFromApp(events))
		if err != nil {
			return nil, err
		}
		toAppend := make([]LedgerEvent, 0, len(workflowRequests))
		for _, workflowRequest := range workflowRequests {
			event, err := buildLedgerEvent(workflowAppendRequestToApp(workflowRequest))
			if err != nil {
				return nil, err
			}
			toAppend = append(toAppend, event)
		}
		if err := ValidateAgentExecutorAppend(events, toAppend); err != nil {
			return nil, err
		}
		return toAppend, nil
	})
	if err != nil {
		return nil, err
	}
	return workflowEventsFromApp(appended), nil
}

func workflowAppendRequestToApp(req workflowruns.AppendEventRequest) AppendEventRequest {
	return AppendEventRequest{
		EventID:          req.EventID,
		MissionID:        req.MissionID,
		EventType:        req.EventType,
		Producer:         Producer{Type: req.Producer.Type, ID: req.Producer.ID},
		CausationEventID: req.CausationEventID,
		CorrelationID:    req.CorrelationID,
		Payload:          req.Payload,
	}
}

func translateWorkflowRunError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, workflowruns.ErrInvalidInput) {
		message := workflowruns.InvalidInputMessage(err)
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("%w: %s", ErrInvalidInput, message)
	}
	return err
}

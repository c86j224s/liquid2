package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledgerstate"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

var ErrInvalidInput = errors.New("invalid input")
var ErrConflict = errors.New("conflict")

type MissionStore interface {
	CreateMission(context.Context, Mission) error
	AppendLedgerEvent(context.Context, LedgerEvent) (LedgerEvent, error)
	ListLedgerEvents(context.Context, string) ([]LedgerEvent, error)
}

type MissionListStore interface {
	ListMissions(context.Context) ([]Mission, error)
}

// MissionActivityListStore loads list-summary inputs in bulk. It prevents the
// mission list from issuing one full-ledger read per mission.
type MissionActivityListStore interface {
	ListMissionActivityInputs(context.Context, []string) ([]MissionActivityInput, error)
}

type ConditionalLedgerStore interface {
	AppendLedgerEventsConditionally(context.Context, string, func([]LedgerEvent) ([]LedgerEvent, error)) ([]LedgerEvent, error)
}

// AppendReportTerminalIfOpen atomically checks and closes one report pending
// event. A false appended flag means another caller already closed it.
func (s *Service) AppendReportTerminalIfOpen(ctx context.Context, missionID, pendingEventID string, reqs []AppendEventRequest) ([]LedgerEvent, bool, error) {
	if _, ok := s.store.(ConditionalLedgerStore); !ok {
		return nil, false, fmt.Errorf("%w: conditional ledger store is required for report terminal closure", ErrInvalidInput)
	}
	if err := validateID("mis_", missionID); err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(pendingEventID) == "" || len(reqs) == 0 {
		return nil, false, fmt.Errorf("%w: pending event and terminal events are required", ErrInvalidInput)
	}
	appended, err := s.appendLedgerEventsConditionally(ctx, missionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
		pending, found := reportPendingEvent(events, pendingEventID)
		if !found {
			return nil, fmt.Errorf("%w: report pending event %q does not exist", ErrInvalidInput, pendingEventID)
		}
		completed := ledgerstate.CompletedReportPendingEventIDs(ledgerStateEventsFromApp(events))
		if _, closed := completed[strings.TrimSpace(pendingEventID)]; closed {
			return nil, nil
		}
		built := make([]LedgerEvent, 0, len(reqs))
		terminalCount := 0
		terminalID := ""
		for _, req := range reqs {
			if strings.TrimSpace(req.MissionID) != missionID {
				return nil, fmt.Errorf("%w: event mission_id must match %s", ErrInvalidInput, missionID)
			}
			event, err := buildLedgerEvent(req)
			if err != nil {
				return nil, err
			}
			terminal, err := validateReportTerminalAppend(pending, event)
			if err != nil {
				return nil, err
			}
			if terminal {
				terminalCount++
				terminalID = event.EventID
			}
			built = append(built, event)
		}
		if terminalCount != 1 {
			return nil, fmt.Errorf("%w: report closure requires exactly one terminal event", ErrInvalidInput)
		}
		for _, event := range built {
			if event.EventType != "report.plan.failed" && event.EventType != "report.section.failed" && event.EventType != "report.part.failed" && event.EventType != "report.final.failed" && event.EventType != "report.artifact.failed" {
				continue
			}
			var payload struct {
				TerminalID string `json:"terminal_event_id"`
			}
			_ = json.Unmarshal(event.Payload, &payload)
			if event.CorrelationID != terminalID || payload.TerminalID != terminalID {
				return nil, fmt.Errorf("%w: stage companion must correlate to terminal event", ErrInvalidInput)
			}
		}
		if err := ValidateAgentExecutorAppend(events, built); err != nil {
			return nil, err
		}
		return built, nil
	})
	if err != nil {
		return nil, false, err
	}
	if len(appended) == 0 {
		return nil, false, nil
	}
	return appended, true, nil
}

func reportPendingEvent(events []LedgerEvent, pendingID string) (LedgerEvent, bool) {
	for _, event := range events {
		if event.EventID != pendingID {
			continue
		}
		switch event.EventType {
		case "report.draft.pending", "report.design.pending", "report.humanize.pending", "report.patch.pending":
			return event, true
		default:
			return LedgerEvent{}, false
		}
	}
	return LedgerEvent{}, false
}

func validateReportTerminalAppend(pending, terminal LedgerEvent) (bool, error) {
	var payload struct {
		PendingID  string `json:"pending_event_id"`
		StageKind  string `json:"stage_kind"`
		StageID    string `json:"stage_id"`
		TerminalID string `json:"terminal_event_id"`
		Generation struct {
			PendingID string `json:"pending_event_id"`
		} `json:"generation"`
	}
	if err := json.Unmarshal(terminal.Payload, &payload); err != nil {
		return false, fmt.Errorf("%w: invalid report event payload", ErrInvalidInput)
	}
	if terminal.EventType == "report.drafted" && payload.PendingID == "" {
		payload.PendingID = payload.Generation.PendingID
	}
	if strings.TrimSpace(payload.PendingID) != pending.EventID {
		return false, fmt.Errorf("%w: report event must correlate to pending event %q", ErrInvalidInput, pending.EventID)
	}
	if strings.HasPrefix(terminal.EventType, "report.") && strings.HasSuffix(terminal.EventType, ".failed") && terminal.EventType != "report.draft.failed" && terminal.EventType != "report.patch.failed" && terminal.EventType != "report.design.failed" && terminal.EventType != "report.humanize.failed" {
		kind := strings.TrimPrefix(strings.TrimSuffix(terminal.EventType, ".failed"), "report.")
		validKind := map[string]bool{"plan": true, "section": true, "part": true, "final": true, "artifact": true}[kind]
		if pending.EventType != "report.draft.pending" || !validKind || payload.StageKind != kind || payload.StageID == "" {
			return false, fmt.Errorf("%w: invalid report stage companion", ErrInvalidInput)
		}
		return false, nil
	}
	allowed := map[string]map[string]bool{
		"report.draft.pending":    {"report.draft.failed": true, "report.drafted": true, "report.artifact.created": true},
		"report.design.pending":   {"report.design.failed": true, "report.artifact.exported": true},
		"report.humanize.pending": {"report.humanize.failed": true, "report.humanize.skipped": true, "report.artifact.exported": true},
		"report.patch.pending":    {"report.patch.failed": true, "report.artifact.created": true},
	}
	if !allowed[pending.EventType][terminal.EventType] {
		return false, fmt.Errorf("%w: terminal event %q does not match %q", ErrInvalidInput, terminal.EventType, pending.EventType)
	}
	return true, nil
}

func (s *Service) CreateMission(ctx context.Context, req CreateMissionRequest) (Mission, error) {
	if err := validateID("mis_", req.MissionID); err != nil {
		return Mission{}, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return Mission{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	now := time.Now().UTC()
	mission := Mission{
		MissionID:      req.MissionID,
		Title:          strings.TrimSpace(req.Title),
		CreatedAt:      now,
		UpdatedAt:      now,
		LifecycleState: MissionLifecycleActive,
	}
	if err := s.store.CreateMission(ctx, mission); err != nil {
		return Mission{}, err
	}
	return mission, nil
}

func BuildMissionCreatedAppendRequest(req MissionCreatedEventRequest) AppendEventRequest {
	return AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "mission.created",
		Producer:  req.Producer,
		Payload: mustJSON(map[string]any{
			"title":     req.Title,
			"objective": req.Objective,
			"scope":     req.Scope,
		}),
	}
}

func (s *Service) AppendEvent(ctx context.Context, req AppendEventRequest) (LedgerEvent, error) {
	if req.EventType == "report.artifact.created" {
		var payload struct {
			PendingID string `json:"pending_event_id"`
		}
		if json.Unmarshal(req.Payload, &payload) == nil && strings.TrimSpace(payload.PendingID) != "" {
			appended, closed, err := s.AppendReportTerminalIfOpen(ctx, req.MissionID, payload.PendingID, []AppendEventRequest{req})
			if err != nil {
				return LedgerEvent{}, err
			}
			if !closed {
				return LedgerEvent{}, fmt.Errorf("%w: report pending %q is already closed", ErrConflict, payload.PendingID)
			}
			return appended[0], nil
		}
	}
	event, err := buildLedgerEvent(req)
	if err != nil {
		return LedgerEvent{}, err
	}
	if EventLocksAgentExecutor(event.EventType) {
		appended, err := s.appendLedgerEventsConditionally(ctx, event.MissionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
			if err := ValidateAgentExecutorAppend(events, []LedgerEvent{event}); err != nil {
				return nil, err
			}
			return []LedgerEvent{event}, nil
		})
		if err != nil {
			return LedgerEvent{}, err
		}
		if len(appended) != 1 {
			return LedgerEvent{}, fmt.Errorf("%w: expected one appended event", ErrInvalidInput)
		}
		return appended[0], nil
	}
	return s.store.AppendLedgerEvent(ctx, event)
}

func (s *Service) AppendEvents(ctx context.Context, missionID string, reqs []AppendEventRequest) ([]LedgerEvent, error) {
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	if len(reqs) == 0 {
		return nil, fmt.Errorf("%w: at least one event is required", ErrInvalidInput)
	}
	return s.appendLedgerEventsConditionally(ctx, missionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
		built := make([]LedgerEvent, 0, len(reqs))
		for _, req := range reqs {
			if strings.TrimSpace(req.MissionID) != missionID {
				return nil, fmt.Errorf("%w: event mission_id must match %s", ErrInvalidInput, missionID)
			}
			event, err := buildLedgerEvent(req)
			if err != nil {
				return nil, err
			}
			built = append(built, event)
		}
		if err := ValidateAgentExecutorAppend(events, built); err != nil {
			return nil, err
		}
		return built, nil
	})
}

func (s *Service) AppendEventsIfNoActiveAgentWork(ctx context.Context, missionID string, reqs []AppendEventRequest) ([]LedgerEvent, error) {
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	if len(reqs) == 0 {
		return nil, fmt.Errorf("%w: at least one event is required", ErrInvalidInput)
	}
	return s.appendLedgerEventsConditionally(ctx, missionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
		if err := validateNoActiveAgentWork(events); err != nil {
			return nil, err
		}
		built := make([]LedgerEvent, 0, len(reqs))
		for _, req := range reqs {
			if strings.TrimSpace(req.MissionID) != missionID {
				return nil, fmt.Errorf("%w: event mission_id must match %s", ErrInvalidInput, missionID)
			}
			event, err := buildLedgerEvent(req)
			if err != nil {
				return nil, err
			}
			built = append(built, event)
		}
		if err := ValidateAgentExecutorAppend(events, built); err != nil {
			return nil, err
		}
		return built, nil
	})
}

func (s *Service) ListMissions(ctx context.Context) ([]Mission, error) {
	return s.ListMissionsWithState(ctx, ListMissionsRequest{})
}

func (s *Service) ListMissionsWithState(ctx context.Context, req ListMissionsRequest) ([]Mission, error) {
	store, ok := s.store.(MissionListStore)
	if !ok {
		return nil, fmt.Errorf("%w: mission list store is required", ErrInvalidInput)
	}
	missions, err := store.ListMissions(ctx)
	if err != nil {
		return nil, err
	}
	missions = filterMissionsByLifecycle(missions, req)
	if len(missions) == 0 {
		return missions, nil
	}
	missionIDs := make([]string, 0, len(missions))
	for _, mission := range missions {
		missionIDs = append(missionIDs, mission.MissionID)
	}
	if activityStore, ok := s.store.(MissionActivityListStore); ok {
		inputs, err := activityStore.ListMissionActivityInputs(ctx, missionIDs)
		if err != nil {
			return nil, err
		}
		activityByMissionID := make(map[string]MissionActivitySummary, len(inputs))
		for _, input := range inputs {
			activityByMissionID[input.MissionID] = MissionActivityFromInput(input)
		}
		for index := range missions {
			missions[index].Activity = activityByMissionID[missions[index].MissionID]
		}
		return missions, nil
	}

	// Non-SQL adapters retain the existing contract. Production SQLite stores
	// implement MissionActivityListStore and never take this per-mission path.
	for index := range missions {
		events, err := s.store.ListLedgerEvents(ctx, missions[index].MissionID)
		if err != nil {
			return nil, err
		}
		missions[index].Activity = MissionActivityFromEvents(events)
	}
	return missions, nil
}

func filterMissionsByLifecycle(missions []Mission, req ListMissionsRequest) []Mission {
	result := make([]Mission, 0, len(missions))
	for _, mission := range missions {
		mission.LifecycleState = normalizeMissionLifecycleState(mission.LifecycleState)
		if !req.IncludeArchived && mission.LifecycleState == MissionLifecycleArchived {
			continue
		}
		result = append(result, mission)
	}
	return result
}

// MissionActivity returns one mission's list-level activity projection without
// loading its detail projection or changing durable mission state.
func (s *Service) MissionActivity(ctx context.Context, missionID string) (MissionActivitySummary, error) {
	if err := validateID("mis_", missionID); err != nil {
		return MissionActivitySummary{}, err
	}
	if activityStore, ok := s.store.(MissionActivityListStore); ok {
		inputs, err := activityStore.ListMissionActivityInputs(ctx, []string{missionID})
		if err != nil {
			return MissionActivitySummary{}, err
		}
		for _, input := range inputs {
			if input.MissionID == missionID {
				return MissionActivityFromInput(input), nil
			}
		}
		return MissionActivitySummary{}, nil
	}
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return MissionActivitySummary{}, err
	}
	return MissionActivityFromEvents(events), nil
}

func buildLedgerEvent(req AppendEventRequest) (LedgerEvent, error) {
	if err := validateID("evt_", req.EventID); err != nil {
		return LedgerEvent{}, err
	}
	if err := validateID("mis_", req.MissionID); err != nil {
		return LedgerEvent{}, err
	}
	if strings.TrimSpace(req.EventType) == "" {
		return LedgerEvent{}, fmt.Errorf("%w: event type is required", ErrInvalidInput)
	}
	if strings.TrimSpace(req.Producer.Type) == "" || strings.TrimSpace(req.Producer.ID) == "" {
		return LedgerEvent{}, fmt.Errorf("%w: producer type and id are required", ErrInvalidInput)
	}
	payload := req.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	if !json.Valid(payload) {
		return LedgerEvent{}, fmt.Errorf("%w: payload must be valid JSON", ErrInvalidInput)
	}
	if err := validateWorkflowEventPayload(strings.TrimSpace(req.EventType), strings.TrimSpace(req.MissionID), payload); err != nil {
		return LedgerEvent{}, err
	}
	if err := validateSourceStateEventPayload(strings.TrimSpace(req.EventType), payload); err != nil {
		return LedgerEvent{}, err
	}

	event := LedgerEvent{
		EventID:          req.EventID,
		MissionID:        req.MissionID,
		EventType:        strings.TrimSpace(req.EventType),
		Producer:         Producer{Type: strings.TrimSpace(req.Producer.Type), ID: strings.TrimSpace(req.Producer.ID)},
		CausationEventID: strings.TrimSpace(req.CausationEventID),
		CorrelationID:    strings.TrimSpace(req.CorrelationID),
		Payload:          append(json.RawMessage(nil), payload...),
		CreatedAt:        time.Now().UTC(),
	}
	return event, nil
}

func (s *Service) ListEvents(ctx context.Context, missionID string) ([]LedgerEvent, error) {
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	return s.store.ListLedgerEvents(ctx, missionID)
}

func (s *Service) appendLedgerEventsConditionally(ctx context.Context, missionID string, build func([]LedgerEvent) ([]LedgerEvent, error)) ([]LedgerEvent, error) {
	if store, ok := s.store.(ConditionalLedgerStore); ok {
		return store.AppendLedgerEventsConditionally(ctx, missionID, build)
	}
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return nil, err
	}
	toAppend, err := build(events)
	if err != nil {
		return nil, err
	}
	appended := make([]LedgerEvent, 0, len(toAppend))
	for _, event := range toAppend {
		committed, err := s.store.AppendLedgerEvent(ctx, event)
		if err != nil {
			return nil, err
		}
		appended = append(appended, committed)
	}
	return appended, nil
}

func validateNoActiveAgentWork(events []LedgerEvent) error {
	if workflowHasOpenAgentPending(events) {
		return fmt.Errorf("%w: agent turn is already running for this mission", ErrInvalidInput)
	}
	if workflowHasOpenReportDraftPending(events) {
		return fmt.Errorf("%w: report draft is already running for this mission", ErrInvalidInput)
	}
	for _, run := range projectWorkflowRuns(events) {
		if !workflowstate.TerminalStatus(run.Status) {
			return fmt.Errorf("%w: workflow %s is %s for this mission", ErrInvalidInput, run.WorkflowRunID, run.Status)
		}
	}
	return nil
}

func workflowHasOpenReportDraftPending(events []LedgerEvent) bool {
	return ledgerstate.HasOpenReportPending(ledgerStateEventsFromApp(events))
}

func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", ErrInvalidInput, prefix)
	}
	return nil
}

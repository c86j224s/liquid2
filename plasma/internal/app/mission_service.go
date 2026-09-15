package app

import "github.com/c86j224s/liquid2/plasma/internal/reportexecution"

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/ledgerstate"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

var ErrInvalidInput = producterror.ErrInvalidInput
var ErrConflict = producterror.ErrConflict

// MissionStore는 단일 미션과 장부 이벤트의 생성·조회·append 계약을 묶는 저장소 포트다.
type MissionStore interface {
	mission.Store
	ledger.Store
}

// MissionListStore는 미션 목록 조회 전용 저장소 포트다.
type MissionListStore = mission.ListStore

// MissionActivityListStore는 목록 요약 입력을 bulk로 읽는 저장소 포트다. 미션 목록이
// 미션마다 전체 원장을 읽는 경로로 퇴행하지 않게 하는 성능 경계다.
type MissionActivityListStore = mission.ActivityListStore

// ConditionalLedgerStore는 조건부 장부 append를 지원하는 저장소 포트다.
type ConditionalLedgerStore = ledger.ConditionalStore

// AppendReportTerminalIfOpen은 하나의 report pending 이벤트가 아직 열려 있는지
// 원자적으로 확인하고 닫는다. appended가 false이면 다른 호출자가 이미 닫았다는 뜻이다.
func (s *Service) AppendReportTerminalIfOpen(ctx context.Context, missionID, pendingEventID string, reqs []ledger.AppendRequest) ([]ledger.Event, bool, error) {
	if _, ok := s.store.(ConditionalLedgerStore); !ok {
		return nil, false, fmt.Errorf("%w: conditional ledger store is required for report terminal closure", ErrInvalidInput)
	}
	if err := validateID("mis_", missionID); err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(pendingEventID) == "" || len(reqs) == 0 {
		return nil, false, fmt.Errorf("%w: pending event and terminal events are required", ErrInvalidInput)
	}
	appended, err := s.appendLedgerEventsConditionally(ctx, missionID, func(events []ledger.Event) ([]ledger.Event, error) {
		built, open, err := buildReportTerminalEventsIfOpen(events, missionID, pendingEventID, reqs)
		if err != nil || !open {
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

func buildReportTerminalEventsIfOpen(events []ledger.Event, missionID, pendingEventID string, reqs []ledger.AppendRequest) ([]ledger.Event, bool, error) {
	pending, found := reportexecution.ReportPendingEvent(events, pendingEventID)
	if !found {
		return nil, false, fmt.Errorf("%w: report pending event %q does not exist", ErrInvalidInput, pendingEventID)
	}
	completed := ledgerstate.CompletedReportPendingEventIDs(ledgerStateEventsFromApp(events))
	if _, closed := completed[strings.TrimSpace(pendingEventID)]; closed {
		return nil, false, nil
	}
	built := make([]ledger.Event, 0, len(reqs))
	terminalCount := 0
	terminalID := ""
	for _, req := range reqs {
		if strings.TrimSpace(req.MissionID) != missionID {
			return nil, false, fmt.Errorf("%w: event mission_id must match %s", ErrInvalidInput, missionID)
		}
		event, err := buildLedgerEvent(req)
		if err != nil {
			return nil, false, err
		}
		terminal, err := reportexecution.ValidateReportTerminalAppend(pending, event)
		if err != nil {
			return nil, false, err
		}
		if terminal && reportexecution.IsIndependentReportILPending(pending) && event.EventType != "report.draft.failed" {
			return nil, false, fmt.Errorf("%w: report IL success requires atomic bundle storage", ErrInvalidInput)
		}
		if terminal {
			terminalCount++
			terminalID = event.EventID
		}
		built = append(built, event)
	}
	if terminalCount != 1 {
		return nil, false, fmt.Errorf("%w: report closure requires exactly one terminal event", ErrInvalidInput)
	}
	for _, event := range built {
		if event.EventType != "report.plan.failed" && event.EventType != "report.requirements.failed" && event.EventType != "report.part_plan.failed" && event.EventType != "report.section.failed" && event.EventType != "report.part.failed" && event.EventType != "report.part_edit.failed" && event.EventType != "report.final.failed" && event.EventType != "report.artifact.failed" && event.EventType != "report.il_source_selection.failed" && event.EventType != "report.il_editorial_memory.failed" && event.EventType != "report.il_narrative.failed" && event.EventType != "report.il_long_form_plan.failed" && event.EventType != "report.il_long_form_sections.failed" && event.EventType != "report.il_long_form_parts.failed" && event.EventType != "report.il_long_form_final.failed" && event.EventType != "report.il_continuity.failed" && event.EventType != "report.il_reader.failed" && event.EventType != "report.il_images.failed" && event.EventType != "report.il_document.failed" && event.EventType != "report.il_flow.failed" && event.EventType != "report.il_render.failed" && event.EventType != "report.il_store.failed" && event.EventType != "report.source_packet.failed" {
			continue
		}
		var payload struct {
			TerminalID string `json:"terminal_event_id"`
		}
		_ = json.Unmarshal(event.Payload, &payload)
		if event.CorrelationID != terminalID || payload.TerminalID != terminalID {
			return nil, false, fmt.Errorf("%w: stage companion must correlate to terminal event", ErrInvalidInput)
		}
	}
	if err := ValidateAgentExecutorAppend(events, built); err != nil {
		return nil, false, err
	}
	return built, true, nil
}

// CreateMission는 새 미션과 최초 장부 이벤트를 저장한다.
func (s *Service) CreateMission(ctx context.Context, req mission.CreateRequest) (mission.Mission, error) {
	if err := validateID("mis_", req.MissionID); err != nil {
		return mission.Mission{}, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return mission.Mission{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	now := time.Now().UTC()
	createdMission := mission.Mission{
		MissionID:      req.MissionID,
		Title:          strings.TrimSpace(req.Title),
		CreatedAt:      now,
		UpdatedAt:      now,
		LifecycleState: mission.LifecycleActive,
	}
	if err := s.store.CreateMission(ctx, createdMission); err != nil {
		return mission.Mission{}, err
	}
	return createdMission, nil
}

// BuildMissionCreatedAppendRequest는 애플리케이션 서비스 계층에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildMissionCreatedAppendRequest(req mission.CreatedEventRequest) ledger.AppendRequest {
	return ledger.AppendRequest{
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

// AppendEvent는 단일 장부 이벤트를 미션에 추가한다.
func (s *Service) AppendEvent(ctx context.Context, req ledger.AppendRequest) (ledger.Event, error) {
	if req.EventType == "report.artifact.created" {
		var payload struct {
			PendingID string `json:"pending_event_id"`
		}
		if json.Unmarshal(req.Payload, &payload) == nil && strings.TrimSpace(payload.PendingID) != "" {
			appended, closed, err := s.AppendReportTerminalIfOpen(ctx, req.MissionID, payload.PendingID, []ledger.AppendRequest{req})
			if err != nil {
				return ledger.Event{}, err
			}
			if !closed {
				return ledger.Event{}, fmt.Errorf("%w: report pending %q is already closed", ErrConflict, payload.PendingID)
			}
			return appended[0], nil
		}
	}
	event, err := buildLedgerEvent(req)
	if err != nil {
		return ledger.Event{}, err
	}
	if EventLocksAgentExecutor(event.EventType) {
		appended, err := s.appendLedgerEventsConditionally(ctx, event.MissionID, func(events []ledger.Event) ([]ledger.Event, error) {
			if err := ValidateAgentExecutorAppend(events, []ledger.Event{event}); err != nil {
				return nil, err
			}
			return []ledger.Event{event}, nil
		})
		if err != nil {
			return ledger.Event{}, err
		}
		if len(appended) != 1 {
			return ledger.Event{}, fmt.Errorf("%w: expected one appended event", ErrInvalidInput)
		}
		return appended[0], nil
	}
	return s.store.AppendLedgerEvent(ctx, event)
}

// AppendEvents는 여러 장부 이벤트를 한 번의 요청으로 추가한다.
func (s *Service) AppendEvents(ctx context.Context, missionID string, reqs []ledger.AppendRequest) ([]ledger.Event, error) {
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	if len(reqs) == 0 {
		return nil, fmt.Errorf("%w: at least one event is required", ErrInvalidInput)
	}
	return s.appendLedgerEventsConditionally(ctx, missionID, func(events []ledger.Event) ([]ledger.Event, error) {
		built := make([]ledger.Event, 0, len(reqs))
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

// AppendEventsIfNoActiveAgentWork는 활성 agent 작업이 없을 때만 이벤트를 추가한다.
func (s *Service) AppendEventsIfNoActiveAgentWork(ctx context.Context, missionID string, reqs []ledger.AppendRequest) ([]ledger.Event, error) {
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	if len(reqs) == 0 {
		return nil, fmt.Errorf("%w: at least one event is required", ErrInvalidInput)
	}
	return s.appendLedgerEventsConditionally(ctx, missionID, func(events []ledger.Event) ([]ledger.Event, error) {
		if err := validateNoActiveAgentWork(events); err != nil {
			return nil, err
		}
		built := make([]ledger.Event, 0, len(reqs))
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

// ListMissions는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListMissions(ctx context.Context) ([]mission.Mission, error) {
	return s.ListMissionsWithState(ctx, mission.ListRequest{})
}

// ListMissionsWithState는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListMissionsWithState(ctx context.Context, req mission.ListRequest) ([]mission.Mission, error) {
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
		activityByMissionID := make(map[string]mission.ActivitySummary, len(inputs))
		for _, input := range inputs {
			activityByMissionID[input.MissionID] = MissionActivityFromInput(input)
		}
		for index := range missions {
			missions[index].Activity = activityByMissionID[missions[index].MissionID]
		}
		return missions, nil
	}

	// SQL이 아닌 어댑터는 기존 계약을 유지한다. 운영 SQLite 저장소는
	// MissionActivityListStore를 구현하므로 이 per-mission 경로를 타지 않는다.
	for index := range missions {
		events, err := s.store.ListLedgerEvents(ctx, missions[index].MissionID)
		if err != nil {
			return nil, err
		}
		missions[index].Activity = MissionActivityFromEvents(events)
	}
	return missions, nil
}

func filterMissionsByLifecycle(missions []mission.Mission, req mission.ListRequest) []mission.Mission {
	result := make([]mission.Mission, 0, len(missions))
	for _, item := range missions {
		item.LifecycleState = mission.NormalizeLifecycleState(item.LifecycleState)
		if !req.IncludeArchived && item.LifecycleState == mission.LifecycleArchived {
			continue
		}
		result = append(result, item)
	}
	return result
}

// MissionActivity는 detail projection을 읽거나 지속 미션 상태를 바꾸지 않고,
// 미션 하나의 목록 수준 activity projection만 계산한다.
func (s *Service) MissionActivity(ctx context.Context, missionID string) (mission.ActivitySummary, error) {
	if err := validateID("mis_", missionID); err != nil {
		return mission.ActivitySummary{}, err
	}
	if activityStore, ok := s.store.(MissionActivityListStore); ok {
		inputs, err := activityStore.ListMissionActivityInputs(ctx, []string{missionID})
		if err != nil {
			return mission.ActivitySummary{}, err
		}
		for _, input := range inputs {
			if input.MissionID == missionID {
				return MissionActivityFromInput(input), nil
			}
		}
		return mission.ActivitySummary{}, nil
	}
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return mission.ActivitySummary{}, err
	}
	return MissionActivityFromEvents(events), nil
}

func buildLedgerEvent(req ledger.AppendRequest) (ledger.Event, error) {
	if err := validateID("evt_", req.EventID); err != nil {
		return ledger.Event{}, err
	}
	if err := validateID("mis_", req.MissionID); err != nil {
		return ledger.Event{}, err
	}
	if strings.TrimSpace(req.EventType) == "" {
		return ledger.Event{}, fmt.Errorf("%w: event type is required", ErrInvalidInput)
	}
	if strings.TrimSpace(req.Producer.Type) == "" || strings.TrimSpace(req.Producer.ID) == "" {
		return ledger.Event{}, fmt.Errorf("%w: producer type and id are required", ErrInvalidInput)
	}
	payload := req.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	if !json.Valid(payload) {
		return ledger.Event{}, fmt.Errorf("%w: payload must be valid JSON", ErrInvalidInput)
	}
	if err := validateWorkflowEventPayload(strings.TrimSpace(req.EventType), strings.TrimSpace(req.MissionID), payload); err != nil {
		return ledger.Event{}, err
	}
	if err := validateSourceStateEventPayload(strings.TrimSpace(req.EventType), payload); err != nil {
		return ledger.Event{}, err
	}

	event := ledger.Event{
		EventID:          req.EventID,
		MissionID:        req.MissionID,
		EventType:        strings.TrimSpace(req.EventType),
		Producer:         ledger.Producer{Type: strings.TrimSpace(req.Producer.Type), ID: strings.TrimSpace(req.Producer.ID)},
		CausationEventID: strings.TrimSpace(req.CausationEventID),
		CorrelationID:    strings.TrimSpace(req.CorrelationID),
		Payload:          append(json.RawMessage(nil), payload...),
		CreatedAt:        time.Now().UTC(),
	}
	return event, nil
}

// ListEvents는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListEvents(ctx context.Context, missionID string) ([]ledger.Event, error) {
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	return s.store.ListLedgerEvents(ctx, missionID)
}

func (s *Service) appendLedgerEventsConditionally(ctx context.Context, missionID string, build func([]ledger.Event) ([]ledger.Event, error)) ([]ledger.Event, error) {
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
	appended := make([]ledger.Event, 0, len(toAppend))
	for _, event := range toAppend {
		committed, err := s.store.AppendLedgerEvent(ctx, event)
		if err != nil {
			return nil, err
		}
		appended = append(appended, committed)
	}
	return appended, nil
}

func validateNoActiveAgentWork(events []ledger.Event) error {
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

func workflowHasOpenReportDraftPending(events []ledger.Event) bool {
	return ledgerstate.HasOpenReportPending(ledgerStateEventsFromApp(events))
}

func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", ErrInvalidInput, prefix)
	}
	return nil
}

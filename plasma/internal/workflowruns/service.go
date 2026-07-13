package workflowruns

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledgerstate"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

const (
	DefaultMaxSteps      = 20
	DefaultMaxDurationMS = int64(0)
	MaxStepsLimit        = 20
	MaxDurationMSLimit   = int64(86_400_000)
	InstructionLimit     = 4000
	SummaryLimit         = 600
	StaleAfter           = 30 * time.Minute
)

var ErrInvalidInput = errors.New("invalid workflow input")

type invalidInputError struct {
	message string
}

func (err invalidInputError) Error() string {
	return err.message
}

func (err invalidInputError) Is(target error) bool {
	return target == ErrInvalidInput
}

func InvalidInputMessage(err error) string {
	var invalid invalidInputError
	if errors.As(err, &invalid) {
		return invalid.message
	}
	return strings.TrimSpace(strings.TrimPrefix(err.Error(), ErrInvalidInput.Error()+":"))
}

type Event = workflowstate.Event

type Producer struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type AppendEventRequest struct {
	EventID          string
	MissionID        string
	EventType        string
	Producer         Producer
	CausationEventID string
	CorrelationID    string
	Payload          json.RawMessage
}

type Store interface {
	ListEvents(context.Context, string) ([]Event, error)
	AppendRequestsConditionally(context.Context, string, func([]Event) ([]AppendEventRequest, error)) ([]Event, error)
}

type IDGenerator func(string) string

func RequestRun(ctx context.Context, store Store, req workflowstate.RequestWorkflowRunRequest, newID IDGenerator, now time.Time) (workflowstate.WorkflowRunView, error) {
	if store == nil {
		return workflowstate.WorkflowRunView{}, invalidInputf("workflow store is required")
	}
	normalized, err := normalizeRunRequest(req, newID)
	if err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	payload := workflowstate.WorkflowRunRequestedPayload{
		WorkflowRunID:             normalized.WorkflowRunID,
		MissionID:                 normalized.MissionID,
		RequestedBySurface:        normalized.RequestedBySurface,
		RequestedByToolSessionID:  normalized.RequestedByToolSessionID,
		AgentExecutor:             normalized.AgentExecutor,
		MCPMode:                   normalized.MCPMode,
		StepInstructionMode:       normalized.StepInstructionMode,
		UserInstructionRaw:        normalized.UserInstructionRaw,
		RunGoal:                   normalized.RunGoal,
		Instruction:               normalized.Instruction,
		MaxSteps:                  normalized.MaxSteps,
		MaxDurationMS:             normalized.MaxDurationMS,
		StopCondition:             normalized.StopCondition,
		StartAfterEventID:         normalized.StartAfterEventID,
		CreatedAt:                 now.UTC().Format(time.RFC3339Nano),
		ArgumentSummary:           normalized.ArgumentSummary,
		ContinueFromWorkflowRunID: normalized.ContinueFromWorkflowRunID,
	}
	if _, err := store.AppendRequestsConditionally(ctx, normalized.MissionID, func(events []Event) ([]AppendEventRequest, error) {
		runs := workflowstate.ProjectRuns(events)
		continueFromFound := normalized.ContinueFromWorkflowRunID == ""
		for _, run := range runs {
			if run.WorkflowRunID == normalized.WorkflowRunID {
				return nil, invalidInputf("workflow run %s already exists", normalized.WorkflowRunID)
			}
			if run.WorkflowRunID == normalized.ContinueFromWorkflowRunID {
				continueFromFound = true
			}
			if !workflowstate.TerminalStatus(run.Status) {
				return nil, invalidInputf("workflow %s is already %s", run.WorkflowRunID, run.Status)
			}
		}
		if !continueFromFound {
			return nil, invalidInputf("continue_from_workflow_run_id not found")
		}
		if hasOpenReportPending(events) {
			return nil, invalidInputf("report draft is already running for this mission")
		}
		if normalized.StartAfterEventID == "" && hasOpenAgentPending(events) {
			return nil, invalidInputf("agent turn is already running for this mission")
		}
		if err := validateStartAfterEvent(events, normalized.StartAfterEventID); err != nil {
			return nil, err
		}
		return []AppendEventRequest{{
			EventID:   nextID(newID, "evt"),
			MissionID: normalized.MissionID,
			EventType: workflowstate.WorkflowRunRequestedEvent,
			Producer:  workflowProducer(normalized.RequestedBySurface, normalized.RequestedByToolSessionID),
			Payload:   mustJSONRaw(payload),
		}}, nil
	}); err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	return GetRun(ctx, store, normalized.MissionID, normalized.WorkflowRunID)
}

func RequestStop(ctx context.Context, store Store, req workflowstate.RequestWorkflowStopRequest, newID IDGenerator, now time.Time) (workflowstate.WorkflowRunView, error) {
	if store == nil {
		return workflowstate.WorkflowRunView{}, invalidInputf("workflow store is required")
	}
	if err := validateID("mis_", req.MissionID); err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	if err := validateID("wfr_", req.WorkflowRunID); err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	surface, err := normalizeSurface(req.RequestedBySurface)
	if err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	payload := workflowstate.WorkflowRunStopRequestedPayload{
		WorkflowRunID:            strings.TrimSpace(req.WorkflowRunID),
		MissionID:                strings.TrimSpace(req.MissionID),
		RequestedBySurface:       surface,
		RequestedByToolSessionID: strings.TrimSpace(req.RequestedByToolSessionID),
		Reason:                   limitText(req.Reason, SummaryLimit),
		RequestedAt:              now.UTC().Format(time.RFC3339Nano),
	}
	var current workflowstate.WorkflowRunView
	_, err = store.AppendRequestsConditionally(ctx, payload.MissionID, func(events []Event) ([]AppendEventRequest, error) {
		var found bool
		for _, run := range workflowstate.ProjectRuns(events) {
			if run.WorkflowRunID == payload.WorkflowRunID {
				current = run
				found = true
				break
			}
		}
		if !found {
			return nil, invalidInputf("workflow run not found")
		}
		if workflowstate.TerminalStatus(current.Status) {
			return nil, nil
		}
		toAppend := []AppendEventRequest{{
			EventID:   nextID(newID, "evt"),
			MissionID: payload.MissionID,
			EventType: workflowstate.WorkflowRunStopRequestedEvent,
			Producer:  workflowProducer(surface, payload.RequestedByToolSessionID),
			Payload:   mustJSONRaw(payload),
		}}
		if strings.TrimSpace(current.StartedEventID) == "" {
			reason := firstNonEmptyText(payload.Reason, "queued workflow stopped before runner start")
			toAppend = append(toAppend, AppendEventRequest{
				EventID:   nextID(newID, "evt"),
				MissionID: payload.MissionID,
				EventType: workflowstate.WorkflowRunStoppedEvent,
				Producer:  workflowProducer(surface, payload.RequestedByToolSessionID),
				Payload: mustJSONRaw(workflowstate.WorkflowRunTerminalPayload{
					WorkflowRunID:      payload.WorkflowRunID,
					MissionID:          payload.MissionID,
					Reason:             reason,
					StopReason:         reason,
					CompletedStepCount: current.CompletedStepCount,
					TerminalAt:         now.UTC().Format(time.RFC3339Nano),
				}),
			})
		}
		return toAppend, nil
	})
	if err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	if workflowstate.TerminalStatus(current.Status) {
		return current, nil
	}
	return GetRun(ctx, store, payload.MissionID, payload.WorkflowRunID)
}

func BuildTerminalAppendRequest(events []Event, req workflowstate.WorkflowRunTerminalEventRequest, newID IDGenerator, now time.Time) (AppendEventRequest, bool, error) {
	workflowRunID := strings.TrimSpace(req.WorkflowRunID)
	eventReq, ok, err := workflowstate.BuildTerminalAppendRequest(events, nextID(newID, "evt"), req, now)
	if err != nil {
		return AppendEventRequest{}, false, invalidInputf("%v", err)
	}
	if !ok {
		return AppendEventRequest{}, false, nil
	}
	return AppendEventRequest{
		EventID:   eventReq.EventID,
		MissionID: eventReq.MissionID,
		EventType: eventReq.EventType,
		Producer:  Producer{Type: "workflow", ID: workflowRunID},
		Payload:   eventReq.Payload,
	}, true, nil
}

func ClaimStart(ctx context.Context, store Store, missionID string, workflowRunID string, startedAt time.Time, newID IDGenerator) (workflowstate.WorkflowRunView, bool, error) {
	if store == nil {
		return workflowstate.WorkflowRunView{}, false, invalidInputf("workflow store is required")
	}
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	var before workflowstate.WorkflowRunView
	var shouldClaim bool
	_, err := store.AppendRequestsConditionally(ctx, missionID, func(events []Event) ([]AppendEventRequest, error) {
		var found bool
		for _, run := range workflowstate.ProjectRuns(events) {
			if run.WorkflowRunID == strings.TrimSpace(workflowRunID) {
				before = run
				found = true
				break
			}
		}
		if !found {
			return nil, invalidInputf("workflow run not found")
		}
		if workflowstate.TerminalStatus(before.Status) || strings.TrimSpace(before.StartedEventID) != "" {
			return nil, nil
		}
		if before.StartAfterEventID != "" && !hasAgentTerminalEventForUser(events, before.StartAfterEventID) {
			return nil, nil
		}
		if hasOpenAgentPending(events) {
			return nil, invalidInputf("agent turn is already running for this mission")
		}
		shouldClaim = true
		return []AppendEventRequest{{
			EventID:   nextID(newID, "evt"),
			MissionID: missionID,
			EventType: workflowstate.WorkflowRunStartedEvent,
			Producer:  Producer{Type: "workflow", ID: workflowRunID},
			Payload: mustJSONRaw(workflowstate.WorkflowRunStartedPayload{
				WorkflowRunID: workflowRunID,
				MissionID:     missionID,
				StartedAt:     startedAt.UTC().Format(time.RFC3339Nano),
			}),
		}}, nil
	})
	if err != nil {
		return workflowstate.WorkflowRunView{}, false, err
	}
	if !shouldClaim {
		return before, false, nil
	}
	view, err := GetRun(ctx, store, missionID, workflowRunID)
	if err != nil {
		return workflowstate.WorkflowRunView{}, false, err
	}
	return view, true, nil
}

func ListRuns(ctx context.Context, store Store, missionID string) ([]workflowstate.WorkflowRunView, error) {
	if store == nil {
		return nil, invalidInputf("workflow store is required")
	}
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	events, err := store.ListEvents(ctx, missionID)
	if err != nil {
		return nil, err
	}
	return workflowstate.ProjectRuns(events), nil
}

func GetRun(ctx context.Context, store Store, missionID string, workflowRunID string) (workflowstate.WorkflowRunView, error) {
	if err := validateID("mis_", missionID); err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	if err := validateID("wfr_", workflowRunID); err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	runs, err := ListRuns(ctx, store, missionID)
	if err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	for _, run := range runs {
		if run.WorkflowRunID == strings.TrimSpace(workflowRunID) {
			return run, nil
		}
	}
	return workflowstate.WorkflowRunView{}, invalidInputf("workflow run not found")
}

func ValidateEventPayload(eventType string, missionID string, payload json.RawMessage) error {
	if !workflowstate.IsEventType(eventType) {
		return nil
	}
	runID, payloadMissionID := workflowstate.RunAndMissionFromPayload(eventType, payload)
	if err := validateID("wfr_", runID); err != nil {
		return err
	}
	if payloadMissionID != strings.TrimSpace(missionID) {
		return invalidInputf("workflow event mission_id must match event mission")
	}
	switch eventType {
	case workflowstate.WorkflowRunRequestedEvent:
		var typed workflowstate.WorkflowRunRequestedPayload
		if err := json.Unmarshal(payload, &typed); err != nil {
			return invalidInputf("workflow payload must be valid JSON")
		}
		if _, err := normalizeSurface(typed.RequestedBySurface); err != nil {
			return err
		}
		if strings.TrimSpace(typed.AgentExecutor) == "" {
			return invalidInputf("agent_executor is required")
		}
		if _, err := normalizeMCPMode(typed.MCPMode); err != nil {
			return err
		}
		if _, err := normalizeStepInstructionMode(typed.StepInstructionMode); err != nil {
			return err
		}
		if strings.TrimSpace(typed.Instruction) == "" {
			return invalidInputf("instruction is required")
		}
		if err := validateRunBounds(typed.MaxSteps, typed.MaxDurationMS); err != nil {
			return err
		}
	case workflowstate.WorkflowStepStartedEvent:
		var typed workflowstate.WorkflowStepStartedPayload
		if err := json.Unmarshal(payload, &typed); err != nil {
			return invalidInputf("workflow payload must be valid JSON")
		}
		if err := validateID("wfs_", typed.WorkflowStepID); err != nil {
			return err
		}
	case workflowstate.WorkflowSourceSkippedEvent:
		var typed workflowstate.WorkflowSourceSkippedPayload
		if err := json.Unmarshal(payload, &typed); err != nil {
			return invalidInputf("workflow payload must be valid JSON")
		}
		if strings.TrimSpace(typed.WorkflowStepID) != "" {
			if err := validateID("wfs_", typed.WorkflowStepID); err != nil {
				return err
			}
		}
		if err := validateID("src_", typed.SnapshotID); err != nil {
			return err
		}
		if strings.TrimSpace(typed.Reason) == "" {
			return invalidInputf("workflow source skip reason is required")
		}
	case workflowstate.WorkflowStepCompletedEvent:
		var typed workflowstate.WorkflowStepCompletedPayload
		if err := json.Unmarshal(payload, &typed); err != nil {
			return invalidInputf("workflow payload must be valid JSON")
		}
		if err := validateID("wfs_", typed.WorkflowStepID); err != nil {
			return err
		}
		switch strings.TrimSpace(typed.Decision) {
		case "continue", "stop":
		default:
			return invalidInputf("workflow step decision must be continue or stop")
		}
	}
	return nil
}

func normalizeRunRequest(req workflowstate.RequestWorkflowRunRequest, newID IDGenerator) (workflowstate.RequestWorkflowRunRequest, error) {
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return workflowstate.RequestWorkflowRunRequest{}, err
	}
	runID := strings.TrimSpace(req.WorkflowRunID)
	if runID == "" {
		runID = nextID(newID, "wfr")
	}
	if err := validateID("wfr_", runID); err != nil {
		return workflowstate.RequestWorkflowRunRequest{}, err
	}
	surface, err := normalizeSurface(req.RequestedBySurface)
	if err != nil {
		return workflowstate.RequestWorkflowRunRequest{}, err
	}
	if strings.TrimSpace(req.AgentExecutor) == "" {
		return workflowstate.RequestWorkflowRunRequest{}, invalidInputf("agent_executor is required")
	}
	agentExecutor, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		return workflowstate.RequestWorkflowRunRequest{}, err
	}
	mcpMode, err := normalizeMCPMode(req.MCPMode)
	if err != nil {
		return workflowstate.RequestWorkflowRunRequest{}, err
	}
	stepInstructionMode, err := normalizeStepInstructionMode(req.StepInstructionMode)
	if err != nil {
		return workflowstate.RequestWorkflowRunRequest{}, err
	}
	instruction := limitText(req.Instruction, InstructionLimit)
	if strings.TrimSpace(instruction) == "" {
		return workflowstate.RequestWorkflowRunRequest{}, invalidInputf("instruction is required")
	}
	userInstructionRaw := ""
	runGoal := ""
	if stepInstructionMode == workflowstate.WorkflowStepInstructionModeLayered {
		userInstructionRaw = limitText(req.UserInstructionRaw, InstructionLimit)
		if strings.TrimSpace(userInstructionRaw) == "" {
			userInstructionRaw = instruction
		}
		runGoal = limitText(req.RunGoal, InstructionLimit)
		if strings.TrimSpace(runGoal) == "" {
			runGoal = instruction
		}
	}
	maxSteps := req.MaxSteps
	if maxSteps <= 0 {
		maxSteps = DefaultMaxSteps
	}
	maxDurationMS := req.MaxDurationMS
	if err := validateRunBounds(maxSteps, maxDurationMS); err != nil {
		return workflowstate.RequestWorkflowRunRequest{}, err
	}
	stopCondition := strings.TrimSpace(req.StopCondition)
	if stopCondition == "" {
		stopCondition = "Stop when max steps, explicit user stop, provider failure, or agent-declared completion is reached."
	}
	argumentSummary := limitText(req.ArgumentSummary, SummaryLimit)
	if argumentSummary == "" {
		argumentSummary = limitText(instruction, SummaryLimit)
	}
	continueFromWorkflowRunID := strings.TrimSpace(req.ContinueFromWorkflowRunID)
	if continueFromWorkflowRunID != "" {
		if err := validateID("wfr_", continueFromWorkflowRunID); err != nil {
			return workflowstate.RequestWorkflowRunRequest{}, err
		}
	}
	return workflowstate.RequestWorkflowRunRequest{
		WorkflowRunID:             runID,
		MissionID:                 missionID,
		RequestedBySurface:        surface,
		RequestedByToolSessionID:  strings.TrimSpace(req.RequestedByToolSessionID),
		AgentExecutor:             agentExecutor,
		MCPMode:                   mcpMode,
		StepInstructionMode:       stepInstructionMode,
		UserInstructionRaw:        userInstructionRaw,
		RunGoal:                   runGoal,
		Instruction:               instruction,
		MaxSteps:                  maxSteps,
		MaxDurationMS:             maxDurationMS,
		StopCondition:             limitText(stopCondition, SummaryLimit),
		StartAfterEventID:         strings.TrimSpace(req.StartAfterEventID),
		ArgumentSummary:           argumentSummary,
		ContinueFromWorkflowRunID: continueFromWorkflowRunID,
	}, nil
}

func validateRunBounds(maxSteps int, maxDurationMS int64) error {
	if maxSteps < 1 || maxSteps > MaxStepsLimit {
		return invalidInputf("max_steps must be between 1 and %d", MaxStepsLimit)
	}
	if maxDurationMS < 0 || maxDurationMS > MaxDurationMSLimit {
		return invalidInputf("max_duration_ms must be between 0 and %d", MaxDurationMSLimit)
	}
	return nil
}

func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return invalidInputf("id must start with %s", prefix)
	}
	return nil
}

func normalizeAgentExecutorName(value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "codex", nil
	}
	switch value {
	case "codex", "claude":
		return value, nil
	default:
		return "", invalidInputf("unsupported agent executor %q", value)
	}
}

func normalizeStepInstructionMode(mode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "", workflowstate.WorkflowStepInstructionModeCurrent:
		return workflowstate.WorkflowStepInstructionModeLayered, nil
	case workflowstate.WorkflowStepInstructionModeLayered:
		return workflowstate.WorkflowStepInstructionModeLayered, nil
	default:
		return "", invalidInputf("invalid workflow step_instruction_mode %q", mode)
	}
}

func normalizeSurface(surface string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(surface)) {
	case workflowstate.WorkflowSurfaceWeb:
		return workflowstate.WorkflowSurfaceWeb, nil
	case workflowstate.WorkflowSurfaceCLI:
		return workflowstate.WorkflowSurfaceCLI, nil
	case workflowstate.WorkflowSurfaceMCP:
		return workflowstate.WorkflowSurfaceMCP, nil
	case workflowstate.WorkflowSurfaceAgentSession:
		return workflowstate.WorkflowSurfaceAgentSession, nil
	default:
		return "", invalidInputf("unsupported workflow surface %q", surface)
	}
}

func normalizeMCPMode(mode string) (string, error) {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		return "auto", nil
	}
	switch mode {
	case "auto", "explicit":
		return mode, nil
	default:
		return "", invalidInputf("unsupported MCP mode %q", mode)
	}
}

func workflowProducer(surface string, toolSessionID string) Producer {
	toolSessionID = strings.TrimSpace(toolSessionID)
	if surface == workflowstate.WorkflowSurfaceAgentSession && toolSessionID != "" {
		return Producer{Type: "agent_session", ID: toolSessionID}
	}
	if surface == workflowstate.WorkflowSurfaceMCP && toolSessionID != "" {
		return Producer{Type: "agent_session", ID: toolSessionID}
	}
	return Producer{Type: "workflow", ID: surface}
}

func validateStartAfterEvent(events []Event, startAfterEventID string) error {
	message := ledgerstate.ValidateWorkflowStartAfterEvent(ledgerEvents(events), startAfterEventID)
	if message == "" {
		return nil
	}
	return invalidInputError{message: message}
}

func hasOpenAgentPending(events []Event) bool {
	return ledgerstate.HasOpenAgentPending(ledgerEvents(events))
}

func hasAgentTerminalEventForUser(events []Event, userEventID string) bool {
	return ledgerstate.HasAgentTerminalEventForUser(ledgerEvents(events), userEventID)
}

func hasOpenReportPending(events []Event) bool {
	return ledgerstate.HasOpenReportPending(ledgerEvents(events))
}

func ledgerEvents(events []Event) []ledgerstate.Event {
	converted := make([]ledgerstate.Event, 0, len(events))
	for _, event := range events {
		converted = append(converted, ledgerstate.Event{
			EventID:   event.EventID,
			Sequence:  event.Sequence,
			EventType: event.EventType,
			Payload:   event.Payload,
			CreatedAt: event.CreatedAt,
		})
	}
	return converted
}

func firstNonEmptyText(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func limitText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return strings.TrimSpace(value[:limit])
}

func mustJSONRaw(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func nextID(newID IDGenerator, prefix string) string {
	if newID != nil {
		return newID(prefix)
	}
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%s_%s", strings.TrimSuffix(prefix, "_"), time.Now().UTC().Format("20060102150405"), hex.EncodeToString(b[:]))
}

func invalidInputf(format string, args ...any) error {
	return invalidInputError{message: fmt.Sprintf(format, args...)}
}

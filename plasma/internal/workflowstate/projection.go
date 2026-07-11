package workflowstate

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

const staleAfter = 30 * time.Minute

type Event struct {
	EventID   string
	MissionID string
	Sequence  int64
	EventType string
	Payload   json.RawMessage
	CreatedAt time.Time
}

func ProjectRuns(events []Event) []WorkflowRunView {
	byRunID := map[string]*WorkflowRunView{}
	order := []string{}
	for _, event := range events {
		if !IsEventType(event.EventType) {
			continue
		}
		runID, missionID := RunAndMissionFromEvent(event)
		if runID == "" || missionID == "" {
			continue
		}
		run := byRunID[runID]
		if run == nil {
			run = &WorkflowRunView{
				WorkflowRunID: runID,
				MissionID:     missionID,
				Status:        WorkflowStatusQueued,
				StatusText:    "워크플로우 실행이 대기 중입니다.",
			}
			byRunID[runID] = run
			order = append(order, runID)
		}
		applyEvent(run, event)
	}
	for _, run := range byRunID {
		applyDerivedStatus(run, time.Now().UTC())
	}
	sort.SliceStable(order, func(i, j int) bool {
		left := byRunID[order[i]]
		right := byRunID[order[j]]
		if left.RequestedAt.Equal(right.RequestedAt) {
			return left.WorkflowRunID < right.WorkflowRunID
		}
		return left.RequestedAt.Before(right.RequestedAt)
	})
	runs := make([]WorkflowRunView, 0, len(order))
	for _, runID := range order {
		runs = append(runs, *byRunID[runID])
	}
	return runs
}

func applyEvent(run *WorkflowRunView, event Event) {
	run.LatestEventID = event.EventID
	run.LatestSequence = event.Sequence
	run.UpdatedAt = event.CreatedAt
	switch event.EventType {
	case WorkflowRunRequestedEvent:
		var payload WorkflowRunRequestedPayload
		if decodePayload(event, &payload) != nil {
			return
		}
		run.RequestedEventID = event.EventID
		run.RequestedAt = event.CreatedAt
		run.RequestedBySurface = payload.RequestedBySurface
		run.AgentExecutor = payload.AgentExecutor
		run.MCPMode = payload.MCPMode
		run.StepInstructionMode = firstNonEmptyText(payload.StepInstructionMode, WorkflowStepInstructionModeCurrent)
		if run.StepInstructionMode == WorkflowStepInstructionModeLayered {
			run.UserInstructionRaw = firstNonEmptyText(payload.UserInstructionRaw, payload.Instruction)
			run.RunGoal = firstNonEmptyText(payload.RunGoal, payload.Instruction)
		}
		run.Instruction = payload.Instruction
		run.MaxSteps = payload.MaxSteps
		run.MaxDurationMS = payload.MaxDurationMS
		run.StopCondition = payload.StopCondition
		run.StartAfterEventID = payload.StartAfterEventID
		run.ContinueFromWorkflowRunID = payload.ContinueFromWorkflowRunID
		if run.Status == "" {
			run.Status = WorkflowStatusQueued
		}
	case WorkflowRunStartedEvent:
		run.StartedEventID = event.EventID
		if !TerminalStatus(run.Status) {
			run.Status = WorkflowStatusRunning
		}
	case WorkflowRunStopRequestedEvent:
		var payload WorkflowRunStopRequestedPayload
		_ = decodePayload(event, &payload)
		run.StopRequestedEventID = event.EventID
		if run.StopReason == "" {
			run.StopReason = payload.Reason
		}
		if !TerminalStatus(run.Status) {
			run.Status = WorkflowStatusStopping
		}
	case WorkflowSourceSkippedEvent:
		if !TerminalStatus(run.Status) && run.Status != WorkflowStatusStopping {
			run.Status = WorkflowStatusRunning
		}
	case WorkflowStepStartedEvent:
		var payload WorkflowStepStartedPayload
		if decodePayload(event, &payload) != nil {
			return
		}
		step := WorkflowStepView{
			WorkflowStepID: payload.WorkflowStepID,
			StepIndex:      payload.StepIndex,
			Status:         WorkflowStatusRunning,
			Instruction:    payload.Instruction,
			ToolSessionID:  payload.ToolSessionID,
			StartedEventID: event.EventID,
		}
		run.CurrentStep = &step
		run.Steps = append(run.Steps, step)
		if !TerminalStatus(run.Status) && run.Status != WorkflowStatusStopping {
			run.Status = WorkflowStatusRunning
		}
	case WorkflowStepCompletedEvent:
		var payload WorkflowStepCompletedPayload
		if decodePayload(event, &payload) != nil {
			return
		}
		run.CompletedStepCount++
		completed := WorkflowStepView{
			WorkflowStepID:  payload.WorkflowStepID,
			Status:          WorkflowStatusCompleted,
			Decision:        payload.Decision,
			NextInstruction: payload.NextInstruction,
			Reason:          payload.Reason,
			DurationMS:      payload.DurationMS,
			AgentSessionID:  payload.AgentSessionID,
			ToolSessionID:   payload.ToolSessionID,
			ResultEventID:   payload.ResultEventID,
		}
		for i := range run.Steps {
			if run.Steps[i].WorkflowStepID == payload.WorkflowStepID {
				completed.StepIndex = run.Steps[i].StepIndex
				completed.Instruction = run.Steps[i].Instruction
				completed.StartedEventID = run.Steps[i].StartedEventID
				run.Steps[i] = completed
				break
			}
		}
		if run.CurrentStep != nil && run.CurrentStep.WorkflowStepID == payload.WorkflowStepID {
			run.CurrentStep = nil
		}
		if strings.TrimSpace(payload.Decision) == "continue" {
			run.ContinuationInstruction = strings.TrimSpace(payload.NextInstruction)
		} else {
			run.ContinuationInstruction = ""
		}
	case WorkflowRunCompletedEvent, WorkflowRunPausedEvent, WorkflowRunStoppedEvent, WorkflowRunFailedEvent, WorkflowRunInterruptedEvent:
		var payload WorkflowRunTerminalPayload
		_ = decodePayload(event, &payload)
		run.TerminalEventID = event.EventID
		run.Status = StatusForTerminalEvent(event.EventType)
		run.StopReason = firstNonEmptyText(payload.StopReason, payload.Reason, payload.Error, run.StopReason)
		if event.EventType == WorkflowRunPausedEvent {
			run.ContinuationInstruction = firstNonEmptyText(payload.NextInstruction, run.ContinuationInstruction)
		} else {
			run.ContinuationInstruction = ""
		}
		if payload.CompletedStepCount > 0 {
			run.CompletedStepCount = payload.CompletedStepCount
		}
		run.CurrentStep = nil
	}
}

func applyDerivedStatus(run *WorkflowRunView, now time.Time) {
	if run.Status == "" {
		run.Status = WorkflowStatusQueued
	}
	if !TerminalStatus(run.Status) && run.StartedEventID != "" && !run.UpdatedAt.IsZero() && now.Sub(run.UpdatedAt) > staleAfter {
		run.Status = WorkflowStatusInterrupted
		run.StopReason = "workflow runner is no longer active"
		run.CurrentStep = nil
	}
	switch run.Status {
	case WorkflowStatusQueued:
		run.StatusText = "워크플로우 실행이 대기 중입니다."
	case WorkflowStatusRunning:
		run.StatusText = "워크플로우 실행 중입니다."
	case WorkflowStatusStopping:
		run.StatusText = "정지 요청을 받았고 다음 단계 전에 종료됩니다."
	case WorkflowStatusCompleted:
		run.StatusText = "워크플로우가 완료되었습니다."
	case WorkflowStatusPaused:
		run.StatusText = "추가 조사가 남아 자율 진행이 일시 중단되었습니다."
	case WorkflowStatusStopped:
		run.StatusText = "워크플로우가 사용자 요청으로 정지되었습니다."
	case WorkflowStatusFailed:
		run.StatusText = "워크플로우가 실패했습니다."
	case WorkflowStatusInterrupted:
		run.StatusText = "워크플로우 실행자가 사라져 중단된 상태입니다."
	}
}

func IsEventType(eventType string) bool {
	switch eventType {
	case WorkflowRunRequestedEvent,
		WorkflowRunStartedEvent,
		WorkflowRunStopRequestedEvent,
		WorkflowSourceSkippedEvent,
		WorkflowStepStartedEvent,
		WorkflowStepCompletedEvent,
		WorkflowRunCompletedEvent,
		WorkflowRunPausedEvent,
		WorkflowRunStoppedEvent,
		WorkflowRunFailedEvent,
		WorkflowRunInterruptedEvent:
		return true
	default:
		return false
	}
}

func RunAndMissionFromEvent(event Event) (string, string) {
	return RunAndMissionFromPayload(event.EventType, event.Payload)
}

func RunAndMissionFromPayload(eventType string, payload json.RawMessage) (string, string) {
	var base struct {
		WorkflowRunID string `json:"workflow_run_id"`
		MissionID     string `json:"mission_id"`
	}
	if err := json.Unmarshal(payload, &base); err != nil {
		return "", ""
	}
	return strings.TrimSpace(base.WorkflowRunID), strings.TrimSpace(base.MissionID)
}

func StatusForTerminalEvent(eventType string) string {
	switch eventType {
	case WorkflowRunCompletedEvent:
		return WorkflowStatusCompleted
	case WorkflowRunPausedEvent:
		return WorkflowStatusPaused
	case WorkflowRunStoppedEvent:
		return WorkflowStatusStopped
	case WorkflowRunFailedEvent:
		return WorkflowStatusFailed
	case WorkflowRunInterruptedEvent:
		return WorkflowStatusInterrupted
	default:
		return ""
	}
}

func TerminalStatus(status string) bool {
	switch status {
	case WorkflowStatusCompleted, WorkflowStatusPaused, WorkflowStatusStopped, WorkflowStatusFailed, WorkflowStatusInterrupted:
		return true
	default:
		return false
	}
}

func HasTerminalEvent(events []Event, workflowRunID string) bool {
	workflowRunID = strings.TrimSpace(workflowRunID)
	if workflowRunID == "" {
		return false
	}
	for _, event := range events {
		if StatusForTerminalEvent(event.EventType) == "" {
			continue
		}
		var payload struct {
			WorkflowRunID string `json:"workflow_run_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.WorkflowRunID) == workflowRunID {
			return true
		}
	}
	return false
}

func decodePayload(event Event, target any) error {
	if err := json.Unmarshal(event.Payload, target); err != nil {
		return err
	}
	return nil
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

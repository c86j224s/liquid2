package app

import "github.com/c86j224s/liquid2/plasma/internal/workflowstate"

// Workflow*Event 상수는 workflowstate의 stable event type을 app 경계에서 재노출한다.
const WorkflowRunRequestedEvent = workflowstate.WorkflowRunRequestedEvent
const WorkflowRunStartedEvent = workflowstate.WorkflowRunStartedEvent
const WorkflowRunStopRequestedEvent = workflowstate.WorkflowRunStopRequestedEvent
const WorkflowSourceSkippedEvent = workflowstate.WorkflowSourceSkippedEvent
const WorkflowStepStartedEvent = workflowstate.WorkflowStepStartedEvent
const WorkflowStepCompletedEvent = workflowstate.WorkflowStepCompletedEvent
const WorkflowRunCompletedEvent = workflowstate.WorkflowRunCompletedEvent
const WorkflowRunPausedEvent = workflowstate.WorkflowRunPausedEvent
const WorkflowRunStoppedEvent = workflowstate.WorkflowRunStoppedEvent
const WorkflowRunFailedEvent = workflowstate.WorkflowRunFailedEvent
const WorkflowRunInterruptedEvent = workflowstate.WorkflowRunInterruptedEvent

// WorkflowStatus* 상수는 workflowstate의 stable status 값을 app 경계에서 재노출한다.
const WorkflowStatusQueued = workflowstate.WorkflowStatusQueued
const WorkflowStatusRunning = workflowstate.WorkflowStatusRunning
const WorkflowStatusStopping = workflowstate.WorkflowStatusStopping
const WorkflowStatusCompleted = workflowstate.WorkflowStatusCompleted
const WorkflowStatusPaused = workflowstate.WorkflowStatusPaused
const WorkflowStatusStopped = workflowstate.WorkflowStatusStopped
const WorkflowStatusFailed = workflowstate.WorkflowStatusFailed
const WorkflowStatusInterrupted = workflowstate.WorkflowStatusInterrupted

// WorkflowSurface* 상수는 workflow 요청 표면 값을 app 경계에서 재노출한다.
const WorkflowSurfaceWeb = workflowstate.WorkflowSurfaceWeb
const WorkflowSurfaceCLI = workflowstate.WorkflowSurfaceCLI
const WorkflowSurfaceMCP = workflowstate.WorkflowSurfaceMCP
const WorkflowSurfaceAgentSession = workflowstate.WorkflowSurfaceAgentSession

// WorkflowStepInstructionMode* 상수는 workflow step prompt 구성 방식을 app 경계에서
// 재노출한다.
const WorkflowStepInstructionModeCurrent = workflowstate.WorkflowStepInstructionModeCurrent
const WorkflowStepInstructionModeLayered = workflowstate.WorkflowStepInstructionModeLayered

// RequestWorkflowRunRequest부터 WorkflowStepView까지는 workflowstate model alias다.
//
// app package가 외부 transport의 단일 import 경계가 될 수 있도록 재노출하지만, 실제
// projection과 terminal event 규칙은 workflowstate/workflowruns 패키지가 소유한다.
type RequestWorkflowRunRequest = workflowstate.RequestWorkflowRunRequest

// RequestWorkflowStopRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type RequestWorkflowStopRequest = workflowstate.RequestWorkflowStopRequest

// WorkflowRunTerminalEventRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type WorkflowRunTerminalEventRequest = workflowstate.WorkflowRunTerminalEventRequest

// WorkflowRunRequestedPayload는 애플리케이션 서비스 계층에서 장부나 전송에 저장하는 payload다. 민감한 원문과 credential을 포함하지 않는 것이 계약이다.
type WorkflowRunRequestedPayload = workflowstate.WorkflowRunRequestedPayload

// WorkflowRunStopRequestedPayload는 애플리케이션 서비스 계층에서 장부나 전송에 저장하는 payload다. 민감한 원문과 credential을 포함하지 않는 것이 계약이다.
type WorkflowRunStopRequestedPayload = workflowstate.WorkflowRunStopRequestedPayload

// WorkflowRunStartedPayload는 애플리케이션 서비스 계층에서 장부나 전송에 저장하는 payload다. 민감한 원문과 credential을 포함하지 않는 것이 계약이다.
type WorkflowRunStartedPayload = workflowstate.WorkflowRunStartedPayload

// WorkflowStepStartedPayload는 애플리케이션 서비스 계층에서 장부나 전송에 저장하는 payload다. 민감한 원문과 credential을 포함하지 않는 것이 계약이다.
type WorkflowStepStartedPayload = workflowstate.WorkflowStepStartedPayload

// WorkflowSourceSkippedPayload는 애플리케이션 서비스 계층에서 장부나 전송에 저장하는 payload다. 민감한 원문과 credential을 포함하지 않는 것이 계약이다.
type WorkflowSourceSkippedPayload = workflowstate.WorkflowSourceSkippedPayload

// WorkflowStepCompletedPayload는 애플리케이션 서비스 계층에서 장부나 전송에 저장하는 payload다. 민감한 원문과 credential을 포함하지 않는 것이 계약이다.
type WorkflowStepCompletedPayload = workflowstate.WorkflowStepCompletedPayload

// WorkflowRunTerminalPayload는 애플리케이션 서비스 계층에서 장부나 전송에 저장하는 payload다. 민감한 원문과 credential을 포함하지 않는 것이 계약이다.
type WorkflowRunTerminalPayload = workflowstate.WorkflowRunTerminalPayload

// WorkflowRunView는 계산한 읽기 모델이다. 원천 상태는 장부와 저장소에 남아 있다.
type WorkflowRunView = workflowstate.WorkflowRunView

// WorkflowStepView는 계산한 읽기 모델이다. 원천 상태는 장부와 저장소에 남아 있다.
type WorkflowStepView = workflowstate.WorkflowStepView

package conversation

import (
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	isolatedForkReportSessionPolicy = "isolated_fork"
	freshReportSessionPolicy        = "fresh_session"
)

// OpenAgentPending은 원장에서 복원한 미완료 에이전트 턴의 실행 단위다.
// UserEventID는 닫힘 판정의 안정 식별자이고, WorkflowRunID와 WorkflowStepID는
// workflow 소속을 보존하기 위한 선택적 문맥이다.
type OpenAgentPending struct {
	UserEventID    string
	AgentExecutor  string
	WorkflowRunID  string
	WorkflowStepID string
}

// LatestAgentSessionID는 특정 executor가 다음 대화 턴에서 재개할 세션 ID를
// 원장 이벤트에서 복원한다. isolated/fresh 보고서 세션은 보고서 작성용 세션을
// 이어 쓰지 않도록 pre-report 연구 세션으로 되돌린다.
func LatestAgentSessionID(events []app.LedgerEvent, executorName string) string {
	latestOrder := int64(-1)
	latestSessionID := ""
	for i, event := range events {
		if event.EventType != "turn.agent.response" && event.EventType != "report.artifact.created" && event.EventType != "agent.session.reset" {
			continue
		}
		var payload struct {
			AgentSessionID             string `json:"agent_session_id"`
			AgentExecutor              string `json:"agent_executor"`
			Kind                       string `json:"kind"`
			ReportSessionPolicy        string `json:"report_session_policy"`
			PreReportResearchSessionID string `json:"pre_report_research_session_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if !AgentEventMatchesExecutor(payload.AgentExecutor, executorName) {
			continue
		}
		if event.EventType == "turn.agent.response" && strings.TrimSpace(payload.Kind) != "agent_response" {
			continue
		}
		order := event.Sequence
		if order == 0 {
			order = int64(i + 1)
		}
		if order < latestOrder {
			continue
		}
		if event.EventType == "agent.session.reset" {
			latestOrder = order
			latestSessionID = ""
			continue
		}
		if event.EventType == "report.artifact.created" && isReportSessionIsolatedFromResearch(payload.ReportSessionPolicy) {
			preReportSessionID := strings.TrimSpace(payload.PreReportResearchSessionID)
			if preReportSessionID == "" {
				continue
			}
			latestOrder = order
			latestSessionID = preReportSessionID
			continue
		}
		if strings.TrimSpace(payload.AgentSessionID) != "" {
			latestOrder = order
			latestSessionID = strings.TrimSpace(payload.AgentSessionID)
		}
	}
	return latestSessionID
}

// LatestAgentModel은 특정 executor에 대해 마지막으로 확인된 모델명을 반환한다.
// 세션 reset 이벤트가 있으면 reset payload의 모델 설정을 현재값으로 본다.
func LatestAgentModel(events []app.LedgerEvent, executorName string) string {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].EventType != "agent.session.reset" && events[i].EventType != "turn.agent.response" {
			continue
		}
		var payload struct {
			AgentExecutor string `json:"agent_executor"`
			AgentModel    string `json:"agent_model"`
			Kind          string `json:"kind"`
		}
		if err := json.Unmarshal(events[i].Payload, &payload); err != nil {
			continue
		}
		if !AgentEventMatchesExecutor(payload.AgentExecutor, executorName) {
			continue
		}
		if events[i].EventType == "agent.session.reset" {
			return strings.TrimSpace(payload.AgentModel)
		}
		if strings.TrimSpace(payload.Kind) == "agent_response" && strings.TrimSpace(payload.AgentModel) != "" {
			return strings.TrimSpace(payload.AgentModel)
		}
	}
	return ""
}

// LatestAgentReasoningEffort는 특정 executor에 대해 마지막으로 확인된 추론
// 강도를 반환한다. 빈 값은 호출자가 기본 설정을 적용해야 한다는 뜻이다.
func LatestAgentReasoningEffort(events []app.LedgerEvent, executorName string) string {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].EventType != "agent.session.reset" && events[i].EventType != "turn.agent.response" {
			continue
		}
		var payload struct {
			AgentExecutor        string `json:"agent_executor"`
			AgentReasoningEffort string `json:"agent_reasoning_effort"`
			Kind                 string `json:"kind"`
		}
		if err := json.Unmarshal(events[i].Payload, &payload); err != nil {
			continue
		}
		if !AgentEventMatchesExecutor(payload.AgentExecutor, executorName) {
			continue
		}
		if events[i].EventType == "agent.session.reset" {
			return strings.TrimSpace(payload.AgentReasoningEffort)
		}
		if strings.TrimSpace(payload.Kind) == "agent_response" && strings.TrimSpace(payload.AgentReasoningEffort) != "" {
			return strings.TrimSpace(payload.AgentReasoningEffort)
		}
	}
	return ""
}

// LatestOpenAgentPending은 아직 turn.agent.response로 닫히지 않은 최신 pending
// 턴을 반환한다. workflowRunID가 주어지면 해당 workflow에 속한 pending만
// 대상으로 삼는다.
func LatestOpenAgentPending(events []app.LedgerEvent, workflowRunID string) (OpenAgentPending, bool) {
	completed := CompletedUserEventIDs(events)
	workflowRunID = strings.TrimSpace(workflowRunID)
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].EventType != "turn.agent.pending" {
			continue
		}
		var payload struct {
			UserEventID    string `json:"user_event_id"`
			AgentExecutor  string `json:"agent_executor"`
			WorkflowRunID  string `json:"workflow_run_id"`
			WorkflowStepID string `json:"workflow_step_id"`
		}
		if err := json.Unmarshal(events[i].Payload, &payload); err != nil {
			continue
		}
		pendingWorkflowRunID := strings.TrimSpace(payload.WorkflowRunID)
		if workflowRunID != "" && pendingWorkflowRunID != workflowRunID {
			continue
		}
		userEventID := strings.TrimSpace(payload.UserEventID)
		if userEventID == "" {
			continue
		}
		if _, ok := completed[userEventID]; ok {
			continue
		}
		return OpenAgentPending{
			UserEventID:    userEventID,
			AgentExecutor:  defaultAgentExecutor(payload.AgentExecutor),
			WorkflowRunID:  pendingWorkflowRunID,
			WorkflowStepID: strings.TrimSpace(payload.WorkflowStepID),
		}, true
	}
	return OpenAgentPending{}, false
}

// AgentPendingForUserEvent는 특정 사용자 이벤트에 연결된 pending 턴을 찾는다.
// 이 함수는 terminal 여부를 판정하지 않고, 원장에 기록된 pending 문맥만 복원한다.
func AgentPendingForUserEvent(events []app.LedgerEvent, userEventID string) (OpenAgentPending, bool) {
	userEventID = strings.TrimSpace(userEventID)
	if userEventID == "" {
		return OpenAgentPending{}, false
	}
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].EventType != "turn.agent.pending" {
			continue
		}
		var payload struct {
			UserEventID    string `json:"user_event_id"`
			AgentExecutor  string `json:"agent_executor"`
			WorkflowRunID  string `json:"workflow_run_id"`
			WorkflowStepID string `json:"workflow_step_id"`
		}
		if err := json.Unmarshal(events[i].Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.UserEventID) != userEventID {
			continue
		}
		return OpenAgentPending{
			UserEventID:    userEventID,
			AgentExecutor:  defaultAgentExecutor(payload.AgentExecutor),
			WorkflowRunID:  strings.TrimSpace(payload.WorkflowRunID),
			WorkflowStepID: strings.TrimSpace(payload.WorkflowStepID),
		}, true
	}
	return OpenAgentPending{}, false
}

// HasOpenAgentPending은 workflow 구분 없이 열린 에이전트 pending이 남아 있는지
// 판정한다.
func HasOpenAgentPending(events []app.LedgerEvent) bool {
	_, ok := LatestOpenAgentPending(events, "")
	return ok
}

// HasAgentTerminalEventForUser는 특정 사용자 이벤트가 에이전트 terminal 응답으로
// 닫혔는지 판정한다. 빈 ID는 유효한 사용자 이벤트가 아니므로 닫힘으로 보지 않는다.
func HasAgentTerminalEventForUser(events []app.LedgerEvent, userEventID string) bool {
	userEventID = strings.TrimSpace(userEventID)
	if userEventID == "" {
		return false
	}
	_, ok := CompletedUserEventIDs(events)[userEventID]
	return ok
}

// CompletedUserEventIDs는 turn.agent.response가 닫은 사용자 이벤트 ID 집합을
// 만든다. payload 파싱에 실패한 이벤트는 닫힘 근거로 쓰지 않는다.
func CompletedUserEventIDs(events []app.LedgerEvent) map[string]struct{} {
	completed := map[string]struct{}{}
	for _, event := range events {
		if event.EventType != "turn.agent.response" {
			continue
		}
		var payload struct {
			UserEventID string `json:"user_event_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		userEventID := strings.TrimSpace(payload.UserEventID)
		if userEventID != "" {
			completed[userEventID] = struct{}{}
		}
	}
	return completed
}

// AgentEventMatchesExecutor는 과거 이벤트와 현재 executor 이름의 호환 규칙을
// 캡슐화한다. 과거 payload의 빈 executor는 codex 이벤트로만 취급한다.
func AgentEventMatchesExecutor(eventExecutor string, executorName string) bool {
	eventExecutor = strings.TrimSpace(eventExecutor)
	executorName = strings.TrimSpace(executorName)
	if eventExecutor == "" {
		return executorName == "codex"
	}
	return eventExecutor == executorName
}

func defaultAgentExecutor(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "codex"
	}
	return value
}

func isReportSessionIsolatedFromResearch(value string) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case isolatedForkReportSessionPolicy, "isolated-fork", "fork", freshReportSessionPolicy:
		return true
	default:
		return false
	}
}

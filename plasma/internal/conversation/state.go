package conversation

import (
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentcapability"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
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
func LatestAgentSessionID(events []ledger.Event, executorName string) string {
	return LatestAgentSession(events, executorName).SessionID
}

// AgentSession identifies the latest resumable provider session and the immutable
// capability profile that was used to create it.
type AgentSession struct {
	SessionID       string
	ProfileID       agentcapability.ProfileID
	ProfileRevision string
}

// LatestAgentSession restores the latest provider session and profile identity.
// Historical events without profile fields map to legacy.v1. Report sessions
// inherit the profile of their recorded pre-report or same-session ancestor.
func LatestAgentSession(events []ledger.Event, executorName string) AgentSession {
	latest, _ := scanAgentSessions(events, executorName)
	return latest
}

// AgentSessionByID restores the immutable profile associated with a recorded
// provider session. It includes report sessions whose profile is inherited from
// their pre-report research lineage.
func AgentSessionByID(events []ledger.Event, executorName string, sessionID string) (AgentSession, bool) {
	_, sessions := scanAgentSessions(events, executorName)
	session, ok := sessions[strings.TrimSpace(sessionID)]
	return session, ok
}

type agentSessionPayload struct {
	AgentSessionID             string                    `json:"agent_session_id"`
	AgentExecutor              string                    `json:"agent_executor"`
	CapabilityProfile          agentcapability.ProfileID `json:"capability_profile"`
	CapabilityProfileRevision  string                    `json:"capability_profile_revision"`
	Kind                       string                    `json:"kind"`
	ReportSessionPolicy        string                    `json:"report_session_policy"`
	PreReportResearchSessionID string                    `json:"pre_report_research_session_id"`
}

func scanAgentSessions(events []ledger.Event, executorName string) (AgentSession, map[string]AgentSession) {
	latestOrder := int64(-1)
	latest := AgentSession{}
	sessions := map[string]AgentSession{}
	for i, event := range events {
		if event.EventType != "turn.agent.response" && event.EventType != "report.artifact.created" && event.EventType != "agent.session.reset" {
			continue
		}
		var payload agentSessionPayload
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
		if event.EventType == "agent.session.reset" {
			if order >= latestOrder {
				latestOrder = order
				latest = AgentSession{}
			}
			continue
		}

		sessionID := strings.TrimSpace(payload.AgentSessionID)
		preReportSessionID := strings.TrimSpace(payload.PreReportResearchSessionID)
		profile := profileFromSessionPayload(payload, sessions, sessionID, preReportSessionID)
		if sessionID != "" {
			sessions[sessionID] = AgentSession{
				SessionID:       sessionID,
				ProfileID:       profile.ProfileID,
				ProfileRevision: profile.ProfileRevision,
			}
		}

		if event.EventType == "report.artifact.created" && isReportSessionIsolatedFromResearch(payload.ReportSessionPolicy) {
			if preReportSessionID == "" || order < latestOrder {
				continue
			}
			preReportSession := sessions[preReportSessionID]
			if preReportSession.SessionID == "" {
				preReportSession = AgentSession{
					SessionID:       preReportSessionID,
					ProfileID:       profile.ProfileID,
					ProfileRevision: profile.ProfileRevision,
				}
				sessions[preReportSessionID] = preReportSession
			}
			latestOrder = order
			latest = preReportSession
			continue
		}
		if sessionID != "" && order >= latestOrder {
			latestOrder = order
			latest = sessions[sessionID]
		}
	}
	return latest, sessions
}

type sessionProfile struct {
	ProfileID       agentcapability.ProfileID
	ProfileRevision string
}

func profileFromSessionPayload(payload agentSessionPayload, sessions map[string]AgentSession, sessionID string, preReportSessionID string) sessionProfile {
	if payload.CapabilityProfile != "" || strings.TrimSpace(payload.CapabilityProfileRevision) != "" {
		return sessionProfile{
			ProfileID:       firstProfileID(payload.CapabilityProfile),
			ProfileRevision: firstProfileRevision(payload.CapabilityProfileRevision),
		}
	}
	for _, ancestorID := range []string{sessionID, preReportSessionID} {
		if ancestor := sessions[ancestorID]; ancestor.SessionID != "" {
			return sessionProfile{
				ProfileID:       ancestor.ProfileID,
				ProfileRevision: ancestor.ProfileRevision,
			}
		}
	}
	return sessionProfile{
		ProfileID:       agentcapability.ProfileLegacyV1,
		ProfileRevision: agentcapability.RevisionV1,
	}
}

func firstProfileID(value agentcapability.ProfileID) agentcapability.ProfileID {
	if value == "" {
		return agentcapability.ProfileLegacyV1
	}
	return value
}

func firstProfileRevision(value string) string {
	if strings.TrimSpace(value) == "" {
		return agentcapability.RevisionV1
	}
	return strings.TrimSpace(value)
}

// LatestAgentModel은 특정 executor에 대해 마지막으로 확인된 모델명을 반환한다.
// 세션 reset 이벤트가 있으면 reset payload의 모델 설정을 현재값으로 본다.
func LatestAgentModel(events []ledger.Event, executorName string) string {
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
func LatestAgentReasoningEffort(events []ledger.Event, executorName string) string {
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
func LatestOpenAgentPending(events []ledger.Event, workflowRunID string) (OpenAgentPending, bool) {
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
func AgentPendingForUserEvent(events []ledger.Event, userEventID string) (OpenAgentPending, bool) {
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
func HasOpenAgentPending(events []ledger.Event) bool {
	_, ok := LatestOpenAgentPending(events, "")
	return ok
}

// HasAgentTerminalEventForUser는 특정 사용자 이벤트가 에이전트 terminal 응답으로
// 닫혔는지 판정한다. 빈 ID는 유효한 사용자 이벤트가 아니므로 닫힘으로 보지 않는다.
func HasAgentTerminalEventForUser(events []ledger.Event, userEventID string) bool {
	userEventID = strings.TrimSpace(userEventID)
	if userEventID == "" {
		return false
	}
	_, ok := CompletedUserEventIDs(events)[userEventID]
	return ok
}

// CompletedUserEventIDs는 turn.agent.response가 닫은 사용자 이벤트 ID 집합을
// 만든다. payload 파싱에 실패한 이벤트는 닫힘 근거로 쓰지 않는다.
func CompletedUserEventIDs(events []ledger.Event) map[string]struct{} {
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

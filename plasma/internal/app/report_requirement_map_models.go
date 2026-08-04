package app

import "encoding/json"

// ReportRequirementMapSubmissionSchemaVersion은 report requirement map 제출 payload의
// schema version이다.
const ReportRequirementMapSubmissionSchemaVersion = "plasma.report_requirement_map_submission.v1"

// ReportRequirementMapSubmissionRequest는 requirement mapping agent가 제출한 section별
// 요구사항 검토 결과를 장부에 기록하는 입력이다.
type ReportRequirementMapSubmissionRequest struct {
	EventID                   string
	MissionID                 string
	PendingEventID            string
	PlanEventID               string
	ToolSessionID             string
	PreviousProviderSessionID string
	AgentExecutor             string
	AgentModel                string
	AgentReasoningEffort      string
	IdempotencyKey            string
	ArgumentsHash             string
	RequirementMapHash        string
	RequirementMap            json.RawMessage
	ReviewedEventIDs          []string
	Attempt                   int
	ToolProducer              Producer
}

// ReportRequirementMapSubmission은 requirement map 제출 event와 replay 여부를 반환한다.
type ReportRequirementMapSubmission struct {
	Event  LedgerEvent
	Replay bool
}

// ReportRequirementMapQuery는 이미 제출된 requirement map을 찾기 위한 binding query다.
type ReportRequirementMapQuery struct {
	MissionID, PendingEventID, PlanEventID, ToolSessionID, PreviousProviderSessionID string
	AgentExecutor, AgentModel, AgentReasoningEffort, IdempotencyKey                  string
}

// ReportRequirementMapSelection은 선택된 requirement map event와 hash metadata다.
type ReportRequirementMapSelection struct {
	Event              LedgerEvent
	RequirementMapHash string
	RequirementMap     json.RawMessage
}

type reportRequirementMapPayload struct {
	SchemaVersion             string          `json:"schema_version"`
	Kind                      string          `json:"kind"`
	PendingEventID            string          `json:"pending_event_id"`
	PlanEventID               string          `json:"plan_event_id"`
	ToolSessionID             string          `json:"tool_session_id"`
	PreviousProviderSessionID string          `json:"previous_provider_session_id,omitempty"`
	AgentExecutor             string          `json:"agent_executor"`
	AgentModel                string          `json:"agent_model,omitempty"`
	AgentReasoningEffort      string          `json:"agent_reasoning_effort,omitempty"`
	IdempotencyKey            string          `json:"idempotency_key"`
	ArgumentsHash             string          `json:"arguments_hash"`
	RequirementMapHash        string          `json:"requirement_map_hash"`
	RequirementMap            json.RawMessage `json:"requirement_map"`
	Attempt                   int             `json:"attempt"`
	Text                      string          `json:"text"`
}

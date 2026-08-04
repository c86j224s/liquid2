package sourcecandidates

import (
	"context"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// SourceCandidateProposalInput는 소스 후보 스테이징 경계에 전달되는 요청 값이다.
type SourceCandidateProposalInput struct {
	URL    string
	Title  string
	Reason string
}

// SourceCandidateProposal는 대화 agent가 제안한 URL 후보와 제안 이유다.
type SourceCandidateProposal struct {
	URL    string `json:"url"`
	Title  string `json:"title,omitempty"`
	Reason string `json:"reason"`
	State  string `json:"state"`
}

// WorkflowSourceCandidateProposal는 workflow 실행 중 제안된 URL 후보와 제안 이유다.
type WorkflowSourceCandidateProposal struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
	State  string `json:"state"`
}

// SourceCandidateFetched는 후보를 승인 전에 미리 가져왔을 때 저장할 본문과 메타데이터다.
type SourceCandidateFetched struct {
	CandidateKind     string
	Content           []byte
	MediaType         string
	Title             string
	ExternalVersion   string
	ExternalUpdatedAt time.Time
	ByteSize          int64
	PageCount         int
	TextLength        int
	TextLengthKnown   bool
}

// SourceCandidateFetcher는 후보 URL을 승인 전 artifact 후보로 가져오는 fetch 포트다.
type SourceCandidateFetcher func(context.Context, string) (SourceCandidateFetched, error)

// SourceCandidateIDFunc는 소스 후보 스테이징 경계에서 테스트와 실행이 같은 ID 생성 계약을 주입할 수 있게 하는 함수 포트다.
type SourceCandidateIDFunc func(prefix string) string

// SourceCandidateProposalEventRequest는 소스 후보 스테이징 경계에 전달되는 요청 값이다.
type SourceCandidateProposalEventRequest struct {
	EventID       string
	MissionID     string
	UserEventID   string
	AgentEventID  string
	ExecutorName  string
	MCPMode       string
	ToolSessionID string
	StrategyID    string
	Producer      app.Producer
	Candidates    []SourceCandidateProposal
}

// SourceCandidateMCPProposalEventRequest는 소스 후보 스테이징 경계에 전달되는 요청 값이다.
type SourceCandidateMCPProposalEventRequest struct {
	EventID            string
	MissionID          string
	SessionID          string
	CurrentUserEventID string
	AgentExecutor      string
	Producer           app.Producer
	Candidates         []SourceCandidateProposal
}

// WorkflowSourceCandidateProposalEventRequest는 소스 후보 스테이징 경계에 전달되는 요청 값이다.
type WorkflowSourceCandidateProposalEventRequest struct {
	EventID        string
	MissionID      string
	WorkflowRunID  string
	WorkflowStepID string
	UserEventID    string
	AgentEventID   string
	Producer       app.Producer
	Candidates     []WorkflowSourceCandidateProposal
}

// SourceCandidateStagingStartRequest는 소스 후보 스테이징 경계에 전달되는 요청 값이다.
type SourceCandidateStagingStartRequest struct {
	EventID          string
	MissionID        string
	SessionID        string
	ProposalEventID  string
	CausationEventID string
	CandidateKind    string
	Candidate        SourceCandidateProposal
	Producer         app.Producer
	AgentExecutor    string
}

// SourceCandidateStagingOutput는 후보 fetch 시도 후 사용자와 agent에게 돌려줄 staging 상태다.
type SourceCandidateStagingOutput struct {
	URL             string `json:"url"`
	ProposalEventID string `json:"proposal_event_id"`
	StagingEventID  string `json:"staging_event_id,omitempty"`
	StagingState    string `json:"staging_state"`
	Message         string `json:"message"`
}

// SourceCandidateStagingStartResult는 staging 시작 이벤트와 tool 응답 payload를 함께 반환한다.
type SourceCandidateStagingStartResult struct {
	Event  app.LedgerEvent
	Output SourceCandidateStagingOutput
}

// SourceCandidateStagingJob는 백그라운드 fetch가 처리할 후보와 causation 정보를 묶는다.
type SourceCandidateStagingJob struct {
	MissionID                         string
	SessionID                         string
	ProposalEventID                   string
	CandidateKind                     string
	Candidate                         SourceCandidateProposal
	Producer                          app.Producer
	StartedEventID                    string
	AgentExecutor                     string
	EmitAgentExecutorInTerminalEvents bool
}

// SourceCandidateStageRequest는 소스 후보 스테이징 경계에 전달되는 요청 값이다.
type SourceCandidateStageRequest struct {
	Job              SourceCandidateStagingJob
	Fetcher          SourceCandidateFetcher
	NewArtifactID    SourceCandidateIDFunc
	NewEventID       SourceCandidateIDFunc
	FilenameFallback string
}

// SourceCandidateDecisionRequest는 소스 후보 스테이징 경계에 전달되는 요청 값이다.
type SourceCandidateDecisionRequest struct {
	EventID   string
	MissionID string
	URL       string
	Reason    string
	Producer  app.Producer
}

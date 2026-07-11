package sourcecandidates

import (
	"context"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type SourceCandidateProposalInput struct {
	URL    string
	Title  string
	Reason string
}

type SourceCandidateProposal struct {
	URL    string `json:"url"`
	Title  string `json:"title,omitempty"`
	Reason string `json:"reason"`
	State  string `json:"state"`
}

type WorkflowSourceCandidateProposal struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
	State  string `json:"state"`
}

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

type SourceCandidateFetcher func(context.Context, string) (SourceCandidateFetched, error)

type SourceCandidateIDFunc func(prefix string) string

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

type SourceCandidateMCPProposalEventRequest struct {
	EventID            string
	MissionID          string
	SessionID          string
	CurrentUserEventID string
	AgentExecutor      string
	Producer           app.Producer
	Candidates         []SourceCandidateProposal
}

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

type SourceCandidateStagingOutput struct {
	URL             string `json:"url"`
	ProposalEventID string `json:"proposal_event_id"`
	StagingEventID  string `json:"staging_event_id,omitempty"`
	StagingState    string `json:"staging_state"`
	Message         string `json:"message"`
}

type SourceCandidateStagingStartResult struct {
	Event  app.LedgerEvent
	Output SourceCandidateStagingOutput
}

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

type SourceCandidateStageRequest struct {
	Job              SourceCandidateStagingJob
	Fetcher          SourceCandidateFetcher
	NewArtifactID    SourceCandidateIDFunc
	NewEventID       SourceCandidateIDFunc
	FilenameFallback string
}

type SourceCandidateDecisionRequest struct {
	EventID   string
	MissionID string
	URL       string
	Reason    string
	Producer  app.Producer
}

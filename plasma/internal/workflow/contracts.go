package workflow

import (
	"context"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

// Service is the consumer-owned port used by the workflow runner. Storage
// transactions and transport request shapes stay behind the implementation.
type Service interface {
	AppendEvent(context.Context, ledger.AppendRequest) (ledger.Event, error)
	ListEvents(context.Context, string) ([]ledger.Event, error)
	ListSourceSnapshotsWithState(context.Context, source.ListRequest) ([]source.Snapshot, error)
	GetWorkflowRun(context.Context, string, string) (workflowstate.WorkflowRunView, error)
	ClaimWorkflowRunStart(context.Context, string, string, time.Time) (workflowstate.WorkflowRunView, bool, error)
}

// AgentExecutor delegates one workflow step to an agent provider.
type AgentExecutor interface {
	Run(context.Context, AgentRequest) (AgentResult, error)
}

// AgentRequest is the provider-agnostic input for one workflow step.
type AgentRequest struct {
	UserText          string
	Prompt            string
	Model             string
	ReasoningEffort   string
	MissionID         string
	ToolSessionID     string
	UserEventID       string
	PreviousSessionID string
	AgentExecutor     string
	MCPMode           string
	Compaction        bool
}

// AgentResult carries the provider result and usage data recorded in the ledger.
type AgentResult struct {
	Text      string
	SessionID string
	Resumed   bool
	Log       string
	Usage     agentusage.AgentUsage
}

// Runner advances a durable workflow one step at a time. Persistent state is
// owned by Service; the value only carries replaceable execution dependencies.
type Runner struct {
	Service               Service
	Agent                 AgentExecutor
	AgentModel            string
	ReasoningEffort       string
	Now                   func() time.Time
	NewID                 func(string) string
	SourceCandidateStager func(context.Context, ledger.Event)
	StepTimeout           time.Duration
}

// ControlDecision is the bounded continuation marker returned by the agent.
type ControlDecision struct {
	Decision        string `json:"decision"`
	Reason          string `json:"reason"`
	NextInstruction string `json:"next_instruction"`
}

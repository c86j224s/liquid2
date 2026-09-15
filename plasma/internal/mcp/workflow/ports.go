package workflow

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

// Service is the narrow workflow application port consumed by the MCP adapter.
type Service interface {
	RequestWorkflowRun(context.Context, workflowstate.RequestWorkflowRunRequest) (workflowstate.WorkflowRunView, error)
	GetWorkflowRun(context.Context, string, string) (workflowstate.WorkflowRunView, error)
	ListWorkflowRuns(context.Context, string) ([]workflowstate.WorkflowRunView, error)
	RequestWorkflowStop(context.Context, workflowstate.RequestWorkflowStopRequest) (workflowstate.WorkflowRunView, error)
}

// Binding carries only workflow-relevant session values from the root transport.
type Binding struct {
	MissionID          string
	AgentSessionID     string
	CurrentUserEventID string
	AgentExecutor      string
}

// EnforceBoundMission is the root-owned mission binding policy callback.
type EnforceBoundMission func(string) error

// ErrorResult maps validation and policy errors through root MCP error policy.
type ErrorResult func(string, string, string, string, bool, []string) wire.ToolResult

// ErrorFromErr maps application errors through root MCP error policy.
type ErrorFromErr func(string, string, error, []string) wire.ToolResult

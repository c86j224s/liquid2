package wire

import (
	"encoding/json"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// ToolDefinition is the metadata entry returned by MCP list_tools.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ToolCall is the minimal MCP call_tool request envelope used by dispatchers.
type ToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResult is the stable Plasma MCP response envelope for success and error
// outcomes. Transport-level tracing may attach TraceEventID or TraceError.
type ToolResult struct {
	ToolName             string          `json:"tool_name"`
	MissionID            string          `json:"mission_id,omitempty"`
	CreatedEventIDs      []string        `json:"created_event_ids,omitempty"`
	ProposalID           string          `json:"proposal_id,omitempty"`
	CreatedRecords       []app.ObjectRef `json:"created_records,omitempty"`
	RequiresUserApproval bool            `json:"requires_user_approval,omitempty"`
	Content              any             `json:"content,omitempty"`
	Error                *ToolError      `json:"error,omitempty"`
	TraceEventID         string          `json:"trace_event_id,omitempty"`
	TraceError           string          `json:"trace_error,omitempty"`
}

// ToolError is the safe, user-visible error shape returned by MCP tools.
type ToolError struct {
	ErrorKind        string   `json:"error_kind"`
	Message          string   `json:"message"`
	Retryable        bool     `json:"retryable"`
	RelatedObjectIDs []string `json:"related_object_ids,omitempty"`
}

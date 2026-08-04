package reporthumanize

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

// Input is the normalized H5 humanization request handed to the capability by
// Web, CLI, or recovery orchestration. Blank executor means no H5 run is
// attempted; blank Markdown means the original artifact is preserved silently.
type Input struct {
	Title                  string
	Markdown               string
	SourceArtifact         artifact.Raw
	ExecutorName           string
	AgentModel             string
	ReasoningEffort        string
	MCPMode                string
	PreviousSessionID      string
	ReportMode             string
	PendingEventID         string
	HumanizePendingEventID string
	ToolSessionID          string
}

// Result is the applied H5 artifact and terminal event. Applied is false when
// the pass is skipped, fails safely, or has already been terminally closed.
type Result struct {
	Artifact artifact.Raw
	Event    ledger.Event
	Markdown string
	Applied  bool
}

// IDFunc supplies deterministic event and session identifiers for production
// and tests. The prefix is the stable caller intent, such as "evt" or "ses".
type IDFunc func(prefix string) string

// Service is the durable boundary needed by the H5 capability. Storage adapter
// details stay outside this package; callers provide this narrow ledger/artifact
// surface.
type Service interface {
	GetRawArtifact(context.Context, string) (artifact.Raw, error)
	AppendEvent(context.Context, ledger.AppendRequest) (ledger.Event, error)
	AppendReportTerminalIfOpen(context.Context, string, string, []ledger.AppendRequest) ([]ledger.Event, bool, error)
}

type eventLister interface {
	ListEvents(context.Context, string) ([]ledger.Event, error)
}

// PendingPayload is the stable payload stored on report.humanize.pending events
// and used to resume or recover H5 work after restart.
type PendingPayload struct {
	Target               string `json:"target"`
	Profile              string `json:"profile"`
	PendingEventID       string `json:"pending_event_id"`
	ReportPendingEventID string `json:"report_pending_event_id"`
	Title                string `json:"title"`
	SourceArtifactID     string `json:"source_artifact_id"`
	SourceArtifactSHA256 string `json:"source_artifact_sha256"`
	AgentExecutor        string `json:"agent_executor"`
	AgentModel           string `json:"agent_model"`
	AgentReasoningEffort string `json:"agent_reasoning_effort"`
	PreviousSessionID    string `json:"previous_agent_session_id"`
	ToolSessionID        string `json:"tool_session_id"`
	MCPMode              string `json:"mcp_mode"`
	ReportMode           string `json:"report_mode"`
	ReportModeLabel      string `json:"report_mode_label"`
	HumanizeTransport    string `json:"humanize_transport"`
}

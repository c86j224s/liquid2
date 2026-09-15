package reportpatch

import "github.com/c86j224s/liquid2/plasma/internal/mcp/wire"

type ReportPatchStartInput struct {
	wire.CommonMutatingInput
	PatchID        string `json:"patch_id"`
	BaseArtifactID string `json:"base_artifact_id"`
	Title          string `json:"title"`
	Instruction    string `json:"instruction"`
}

type ReportPatchReadInput struct {
	MissionID string `json:"mission_id"`
	SessionID string `json:"session_id"`
	PatchID   string `json:"patch_id"`
	Offset    int    `json:"offset"`
	MaxBytes  int    `json:"max_bytes"`
}

type ReportPatchApplyInput struct {
	wire.CommonMutatingInput
	PatchID     string `json:"patch_id"`
	Operation   string `json:"operation"`
	MatchText   string `json:"match_text"`
	Replacement string `json:"replacement"`
	Occurrence  int    `json:"occurrence"`
	ReplaceAll  bool   `json:"replace_all"`
	Summary     string `json:"summary"`
}

type ReportPatchFinalizeInput struct {
	wire.CommonMutatingInput
	PatchID                      string `json:"patch_id"`
	ArtifactID                   string `json:"artifact_id"`
	Filename                     string `json:"filename"`
	Title                        string `json:"title"`
	PatchSummary                 string `json:"patch_summary"`
	ExpectedSHA256               string `json:"expected_sha256"`
	PendingEventID               string `json:"pending_event_id"`
	AgentExecutor                string `json:"agent_executor"`
	AgentModel                   string `json:"agent_model"`
	AgentReasoningEffort         string `json:"agent_reasoning_effort"`
	MCPMode                      string `json:"mcp_mode"`
	AgentSessionID               string `json:"agent_session_id"`
	PreviousAgentSessionID       string `json:"previous_agent_session_id"`
	ReturnedAgentSessionID       string `json:"returned_agent_session_id"`
	ReportSessionID              string `json:"report_session_id"`
	ForkSourceAgentSessionID     string `json:"fork_source_agent_session_id"`
	ReportSessionPolicy          string `json:"report_session_policy"`
	ReportSessionPolicySelection string `json:"report_session_policy_selection"`
	SessionChainKind             string `json:"session_chain_kind"`
}

type ReportPatchOutput struct {
	PatchID        string `json:"patch_id"`
	MissionID      string `json:"mission_id"`
	SessionID      string `json:"session_id"`
	BaseArtifactID string `json:"base_artifact_id"`
	Title          string `json:"title,omitempty"`
	State          string `json:"state"`
	ContentLength  int    `json:"content_length"`
	OperationCount int    `json:"operation_count"`
	Finalized      bool   `json:"finalized"`
	ArtifactID     string `json:"artifact_id,omitempty"`
}

type ReportPatchReadOutput struct {
	PatchID        string `json:"patch_id"`
	MissionID      string `json:"mission_id"`
	SessionID      string `json:"session_id"`
	BaseArtifactID string `json:"base_artifact_id"`
	Content        string `json:"content"`
	Offset         int    `json:"offset"`
	NextOffset     int    `json:"next_offset,omitempty"`
	ContentLength  int    `json:"content_length"`
	Truncated      bool   `json:"truncated"`
	Finalized      bool   `json:"finalized"`
	ArtifactID     string `json:"artifact_id,omitempty"`
}

type ReportPatchFinalizeOutput struct {
	PatchID        string                 `json:"patch_id"`
	MissionID      string                 `json:"mission_id"`
	SessionID      string                 `json:"session_id"`
	BaseArtifactID string                 `json:"base_artifact_id"`
	ContentLength  int                    `json:"content_length"`
	Artifact       wire.RawArtifactOutput `json:"artifact"`
	EventID        string                 `json:"event_id"`
}

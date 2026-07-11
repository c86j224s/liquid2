package web

import "github.com/c86j224s/liquid2/plasma/internal/app"

type createMissionRequest struct {
	Title     string           `json:"title"`
	Objective string           `json:"objective"`
	Scope     app.MissionScope `json:"scope"`
}

type turnRequest struct {
	Text               string `json:"text"`
	AgentExecutor      string `json:"agent_executor"`
	MCPMode            string `json:"mcp_mode"`
	Controller         bool   `json:"controller"`
	ControllerStrategy string `json:"controller_strategy"`
}

type cancelTurnRequest struct {
	AgentExecutor string `json:"agent_executor"`
}

type resetAgentSessionRequest struct {
	AgentExecutor        string `json:"agent_executor"`
	AgentModel           string `json:"agent_model"`
	AgentReasoningEffort string `json:"agent_reasoning_effort"`
}

type workflowGoalDraftRequest struct {
	UserInstructionRaw string `json:"user_instruction_raw"`
	AgentExecutor      string `json:"agent_executor"`
}

type workflowGoalDraftResponse struct {
	UserInstructionRaw string `json:"user_instruction_raw"`
	RunGoal            string `json:"run_goal"`
	StepInstruction    string `json:"step_instruction"`
	Model              string `json:"model,omitempty"`
	ReasoningEffort    string `json:"reasoning_effort,omitempty"`
}

type workflowStartRequest struct {
	WorkflowRunID             string `json:"workflow_run_id"`
	StepInstructionMode       string `json:"step_instruction_mode"`
	UserInstructionRaw        string `json:"user_instruction_raw"`
	RunGoal                   string `json:"run_goal"`
	Instruction               string `json:"instruction"`
	AgentExecutor             string `json:"agent_executor"`
	MCPMode                   string `json:"mcp_mode"`
	MaxSteps                  int    `json:"max_steps"`
	MaxDurationMS             int64  `json:"max_duration_ms"`
	StopCondition             string `json:"stop_condition"`
	ContinueFromWorkflowRunID string `json:"continue_from_workflow_run_id"`
}

type textSourceRequest struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	ExternalURI string `json:"external_uri"`
}

type urlSourceRequest struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type mediaURLSourceRequest struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	License     string `json:"license"`
	Attribution string `json:"attribution"`
}

type pdfURLSourceRequest struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type liquid2SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type liquid2SnapshotRequest struct {
	ExternalSourceID string `json:"external_source_id"`
	Reason           string `json:"reason"`
}

type confluenceConnectionRequest struct {
	ConnectionID string               `json:"connection_id"`
	DisplayName  string               `json:"display_name"`
	AuthType     string               `json:"auth_type"`
	AccountID    string               `json:"account_id"`
	AccountName  string               `json:"account_name"`
	AccessToken  string               `json:"access_token"`
	APIToken     string               `json:"api_token"`
	RefreshToken string               `json:"refresh_token"`
	ExpiresAt    string               `json:"expires_at"`
	Scopes       []string             `json:"scopes"`
	Sites        []app.ConfluenceSite `json:"sites"`
}

type confluenceConnectionUpdateRequest struct {
	DisplayName string `json:"display_name"`
}

type confluenceOAuthStartRequest struct {
	ConnectionID string   `json:"connection_id"`
	DisplayName  string   `json:"display_name"`
	AccountID    string   `json:"account_id"`
	AccountName  string   `json:"account_name"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURI  string   `json:"redirect_uri"`
	Scopes       []string `json:"scopes"`
}

type confluenceSitesRequest struct {
	ConnectionID string `json:"connection_id"`
	Refresh      bool   `json:"refresh"`
}

type confluenceSpacesRequest struct {
	ConnectionID string `json:"connection_id"`
	CloudID      string `json:"cloud_id"`
	Limit        int    `json:"limit"`
	Cursor       string `json:"cursor"`
}

type confluenceSpacePagesRequest struct {
	ConnectionID string `json:"connection_id"`
	CloudID      string `json:"cloud_id"`
	SpaceID      string `json:"space_id"`
	Limit        int    `json:"limit"`
	Cursor       string `json:"cursor"`
}

type confluencePageChildrenRequest struct {
	ConnectionID string `json:"connection_id"`
	CloudID      string `json:"cloud_id"`
	PageID       string `json:"page_id"`
	Limit        int    `json:"limit"`
	Cursor       string `json:"cursor"`
}

type confluenceSearchRequest struct {
	ConnectionID string `json:"connection_id"`
	CloudID      string `json:"cloud_id"`
	Query        string `json:"query"`
	SpaceKey     string `json:"space_key"`
	Limit        int    `json:"limit"`
	Cursor       string `json:"cursor"`
}

type confluenceURLSnapshotRequest struct {
	URL          string `json:"url"`
	Title        string `json:"title"`
	ConnectionID string `json:"connection_id"`
	CloudID      string `json:"cloud_id"`
}

type confluenceSnapshotRequest struct {
	ConnectionID    string `json:"connection_id"`
	CloudID         string `json:"cloud_id"`
	PageID          string `json:"page_id"`
	ExpectedVersion int    `json:"expected_version"`
	MaxBodyBytes    int64  `json:"max_body_bytes"`
	RangeContentID  string `json:"range_content_id"`
	RangeStart      int    `json:"range_start"`
	RangeEnd        int    `json:"range_end"`
	Reason          string `json:"reason"`
}

type confluenceUpdateRequest struct {
	ConnectionID    string `json:"connection_id"`
	SnapshotID      string `json:"snapshot_id"`
	ExpectedVersion int    `json:"expected_version"`
	MaxBodyBytes    int64  `json:"max_body_bytes"`
	RangeContentID  string `json:"range_content_id"`
	RangeStart      int    `json:"range_start"`
	RangeEnd        int    `json:"range_end"`
	Reason          string `json:"reason"`
}

type connectorAccessRequest struct {
	Enabled      bool   `json:"enabled"`
	ConnectionID string `json:"connection_id"`
	CloudID      string `json:"cloud_id"`
	SpaceKey     string `json:"space_key"`
}

type localPathTreeRequest struct {
	RootID       string `json:"root_id"`
	RelativePath string `json:"relative_path"`
	Depth        int    `json:"depth"`
	Limit        int    `json:"limit"`
}

type localPathAttachRequest struct {
	RootID       string `json:"root_id"`
	RelativePath string `json:"relative_path"`
	Title        string `json:"title"`
	Restore      bool   `json:"restore"`
}

type sourceReadRequest struct {
	ArtifactID string `json:"artifact_id"`
	Offset     int64  `json:"offset"`
	MaxBytes   int64  `json:"max_bytes"`
	Depth      int    `json:"depth"`
	Limit      int    `json:"limit"`
}

type sourceGrepRequest struct {
	Query       string `json:"query"`
	MaxSnippets int    `json:"max_snippets"`
}

type sourceRemoveRequest struct {
	Reason string `json:"reason"`
}

type evidenceCandidateRequest struct {
	Summary      string `json:"summary"`
	EvidenceType string `json:"evidence_type"`
	SnapshotID   string `json:"snapshot_id"`
	ArtifactID   string `json:"artifact_id"`
}

type claimConfidenceRequest struct {
	Level             string   `json:"level"`
	Rationale         string   `json:"rationale"`
	OpenRisks         []string `json:"open_risks"`
	NeedsVerification bool     `json:"needs_verification"`
	BasisEvidenceIDs  []string `json:"basis_evidence_ids"`
}

type rejectSourceCandidateRequest struct {
	URL    string `json:"url"`
	Reason string `json:"reason"`
}

type restoreSourceCandidateRequest struct {
	URL    string `json:"url"`
	Reason string `json:"reason"`
}

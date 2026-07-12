package mcp

import (
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/sources/localpath"
)

type commonMutatingInput = CommonMutatingInput

type CommonMutatingInput struct {
	MissionID      string       `json:"mission_id"`
	SessionID      string       `json:"session_id"`
	IdempotencyKey string       `json:"idempotency_key"`
	Producer       app.Producer `json:"producer"`
}

type missionGetInput struct {
	MissionID string   `json:"mission_id"`
	Include   []string `json:"include"`
}

type missionUpdateInput struct {
	CommonMutatingInput
	Title     *string           `json:"title"`
	Objective *string           `json:"objective"`
	Scope     *app.MissionScope `json:"scope"`
}

type missionGetOutput struct {
	MissionProjection   app.MissionProjection  `json:"mission_projection"`
	Sources             []sourceSnapshotOutput `json:"sources,omitempty"`
	Evidence            []app.EvidenceRecord   `json:"evidence,omitempty"`
	Claims              []app.ClaimRecord      `json:"claims,omitempty"`
	OpenQuestions       []app.QuestionRecord   `json:"open_questions"`
	ActiveReportVersion any                    `json:"active_report_version"`
}

type sourcesListInput struct {
	MissionID         string `json:"mission_id"`
	IncludeRemoved    bool   `json:"include_removed"`
	IncludeSuperseded bool   `json:"include_superseded"`
}

type sourcesListOutput struct {
	Sources []sourceSnapshotOutput `json:"sources"`
}

type sourcesReadInput struct {
	MissionID  string `json:"mission_id"`
	SnapshotID string `json:"snapshot_id"`
	ArtifactID string `json:"artifact_id"`
	Subpath    string `json:"subpath"`
	Offset     int    `json:"offset"`
	MaxBytes   int    `json:"max_bytes"`
}

type sourcesReadOutput struct {
	Snapshot            sourceSnapshotOutput    `json:"snapshot"`
	Artifact            rawArtifactOutput       `json:"artifact"`
	Content             string                  `json:"content"`
	Offset              int                     `json:"offset"`
	NextOffset          int                     `json:"next_offset,omitempty"`
	ContentLength       int                     `json:"content_length"`
	ContentLengthKnown  bool                    `json:"content_length_known"`
	Truncated           bool                    `json:"truncated"`
	MetadataOnly        bool                    `json:"metadata_only,omitempty"`
	Extraction          *sourceExtractionOutput `json:"extraction,omitempty"`
	ObservationMetadata *localpath.PathMetadata `json:"observation_metadata,omitempty"`
	ObservationEventID  string                  `json:"observation_event_id,omitempty"`
}

type sourceExtractionOutput struct {
	Type               string `json:"type"`
	PageCount          int    `json:"page_count,omitempty"`
	TextLength         int    `json:"text_length,omitempty"`
	TextLengthKnown    bool   `json:"text_length_known"`
	SuggestedReadBytes int    `json:"suggested_read_bytes,omitempty"`
	MaxReadBytes       int    `json:"max_read_bytes,omitempty"`
}

type sourcesTreeInput struct {
	MissionID  string `json:"mission_id"`
	SnapshotID string `json:"snapshot_id"`
	Subpath    string `json:"subpath"`
	Depth      int    `json:"depth"`
	Limit      int    `json:"limit"`
}

type sourcesTreeOutput struct {
	Snapshot            sourceSnapshotOutput    `json:"snapshot"`
	Tree                localpath.TreeResult    `json:"tree"`
	ObservationMetadata *localpath.PathMetadata `json:"observation_metadata,omitempty"`
	ObservationEventID  string                  `json:"observation_event_id,omitempty"`
}

type sourcesGrepInput struct {
	MissionID   string `json:"mission_id"`
	SnapshotID  string `json:"snapshot_id"`
	Subpath     string `json:"subpath"`
	Query       string `json:"query"`
	MaxSnippets int    `json:"max_snippets"`
}

type sourcesGrepOutput struct {
	Snapshot            sourceSnapshotOutput    `json:"snapshot"`
	Grep                localpath.GrepResult    `json:"grep"`
	ObservationMetadata *localpath.PathMetadata `json:"observation_metadata,omitempty"`
	ObservationEventID  string                  `json:"observation_event_id,omitempty"`
}

type mediaSourceReadOutput struct {
	Snapshot       sourceSnapshotOutput `json:"snapshot"`
	Artifact       rawArtifactOutput    `json:"artifact,omitempty"`
	Media          app.MediaLocator     `json:"media"`
	InspectionNote string               `json:"inspection_note"`
}

type sourcesSearchInput struct {
	MissionID    string   `json:"mission_id"`
	Query        string   `json:"query"`
	Connectors   []string `json:"connectors"`
	ConnectionID string   `json:"connection_id"`
	CloudID      string   `json:"cloud_id"`
	SpaceKey     string   `json:"space_key"`
	Limit        int      `json:"limit"`
	Cursor       string   `json:"cursor"`
}

type sourcesSearchOutput struct {
	Candidates  []sourceCandidateOutput `json:"candidates"`
	NextCursors map[string]string       `json:"next_cursors,omitempty"`
}

type sourceCandidatesProposeInput struct {
	CommonMutatingInput
	Candidates []sourceCandidateProposalInput `json:"candidates"`
}

type sourceCandidateProposalInput struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

type sourceCandidatesProposeOutput struct {
	EventID    string                         `json:"event_id"`
	Candidates []sourceCandidateProposalEvent `json:"candidates"`
	Staging    []sourceCandidateStagingOutput `json:"staging,omitempty"`
}

type sourceCandidateProposalEvent struct {
	URL    string `json:"url"`
	Title  string `json:"title,omitempty"`
	Reason string `json:"reason"`
	State  string `json:"state"`
}

type sourceCandidateStagingOutput struct {
	URL             string `json:"url"`
	ProposalEventID string `json:"proposal_event_id"`
	StagingEventID  string `json:"staging_event_id,omitempty"`
	StagingState    string `json:"staging_state"`
	Message         string `json:"message"`
}

type sourceCandidatesReadInput struct {
	MissionID       string `json:"mission_id"`
	URL             string `json:"url"`
	ProposalEventID string `json:"proposal_event_id"`
	StagingEventID  string `json:"staging_event_id"`
	ArtifactID      string `json:"artifact_id"`
	Offset          int    `json:"offset"`
	MaxBytes        int    `json:"max_bytes"`
}

type sourceCandidatesReadOutput struct {
	ApprovalState      string                  `json:"approval_state"`
	NotReportDefault   bool                    `json:"not_report_default"`
	CandidateURL       string                  `json:"candidate_url,omitempty"`
	ProposalEventID    string                  `json:"proposal_event_id,omitempty"`
	StagingEventID     string                  `json:"staging_event_id,omitempty"`
	StagingState       string                  `json:"staging_state"`
	Message            string                  `json:"message,omitempty"`
	FailureMessage     string                  `json:"failure_message,omitempty"`
	Artifact           *rawArtifactOutput      `json:"artifact,omitempty"`
	Content            string                  `json:"content,omitempty"`
	Offset             int                     `json:"offset,omitempty"`
	NextOffset         int                     `json:"next_offset,omitempty"`
	ContentLength      int                     `json:"content_length,omitempty"`
	ContentLengthKnown bool                    `json:"content_length_known,omitempty"`
	Truncated          bool                    `json:"truncated,omitempty"`
	Extraction         *sourceExtractionOutput `json:"extraction,omitempty"`
}

type researchOutlineInput struct {
	MissionID string `json:"mission_id"`
	Legacy    bool   `json:"legacy"`
}

type researchListInput struct {
	MissionID  string `json:"mission_id"`
	ObjectKind string `json:"object_kind"`
	Limit      int    `json:"limit"`
	Cursor     string `json:"cursor"`
	Legacy     bool   `json:"legacy"`
}

type researchReadInput struct {
	MissionID  string `json:"mission_id"`
	ObjectKind string `json:"object_kind"`
	ObjectID   string `json:"object_id"`
	Offset     int    `json:"offset"`
	MaxBytes   int    `json:"max_bytes"`
	Limit      int    `json:"limit"`
	Cursor     string `json:"cursor"`
	Legacy     bool   `json:"legacy"`
}

type researchGrepInput struct {
	MissionID string `json:"mission_id"`
	Query     string `json:"query"`
	Limit     int    `json:"limit"`
	Cursor    string `json:"cursor"`
	Legacy    bool   `json:"legacy"`
}

type researchReferencesInput struct {
	MissionID  string `json:"mission_id"`
	ObjectKind string `json:"object_kind"`
	ObjectID   string `json:"object_id"`
	Limit      int    `json:"limit"`
	Cursor     string `json:"cursor"`
	Legacy     bool   `json:"legacy"`
}

type workflowStartInput struct {
	MissionID                string `json:"mission_id"`
	WorkflowRunID            string `json:"workflow_run_id"`
	StepInstructionMode      string `json:"step_instruction_mode"`
	UserInstructionRaw       string `json:"user_instruction_raw"`
	RunGoal                  string `json:"run_goal"`
	Instruction              string `json:"instruction"`
	AgentExecutor            string `json:"agent_executor"`
	MCPMode                  string `json:"mcp_mode"`
	MaxSteps                 int    `json:"max_steps"`
	MaxDurationMS            int64  `json:"max_duration_ms"`
	StopCondition            string `json:"stop_condition"`
	StartAfterEventID        string `json:"start_after_event_id"`
	RequestedByToolSessionID string `json:"requested_by_tool_session_id"`
}

type workflowStatusInput struct {
	MissionID     string `json:"mission_id"`
	WorkflowRunID string `json:"workflow_run_id"`
}

type workflowStopInput struct {
	MissionID     string `json:"mission_id"`
	WorkflowRunID string `json:"workflow_run_id"`
	Reason        string `json:"reason"`
}

type experimentReportCreateInput struct {
	CommonMutatingInput
	DraftID string `json:"draft_id"`
	Title   string `json:"title"`
}

type experimentReportAppendInput struct {
	CommonMutatingInput
	DraftID string `json:"draft_id"`
	Content string `json:"content"`
}

type experimentReportReadInput struct {
	MissionID string `json:"mission_id"`
	SessionID string `json:"session_id"`
	DraftID   string `json:"draft_id"`
	Offset    int    `json:"offset"`
	MaxBytes  int    `json:"max_bytes"`
}

type experimentReportFinalizeInput struct {
	CommonMutatingInput
	DraftID        string `json:"draft_id"`
	ArtifactID     string `json:"artifact_id"`
	Filename       string `json:"filename"`
	Title          string `json:"title"`
	ExpectedSHA256 string `json:"expected_sha256"`
}

type reportPatchStartInput struct {
	CommonMutatingInput
	PatchID        string `json:"patch_id"`
	BaseArtifactID string `json:"base_artifact_id"`
	Title          string `json:"title"`
	Instruction    string `json:"instruction"`
}

type reportPatchReadInput struct {
	MissionID string `json:"mission_id"`
	SessionID string `json:"session_id"`
	PatchID   string `json:"patch_id"`
	Offset    int    `json:"offset"`
	MaxBytes  int    `json:"max_bytes"`
}

type reportPatchApplyInput struct {
	CommonMutatingInput
	PatchID     string `json:"patch_id"`
	Operation   string `json:"operation"`
	MatchText   string `json:"match_text"`
	Replacement string `json:"replacement"`
	Occurrence  int    `json:"occurrence"`
	ReplaceAll  bool   `json:"replace_all"`
	Summary     string `json:"summary"`
}

type reportPatchFinalizeInput struct {
	CommonMutatingInput
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

type experimentReportDraftOutput struct {
	DraftID       string `json:"draft_id"`
	MissionID     string `json:"mission_id"`
	SessionID     string `json:"session_id"`
	Title         string `json:"title,omitempty"`
	State         string `json:"state"`
	ContentLength int    `json:"content_length"`
	ChunkCount    int    `json:"chunk_count"`
	Finalized     bool   `json:"finalized"`
	ArtifactID    string `json:"artifact_id,omitempty"`
}

type experimentReportReadOutput struct {
	DraftID       string `json:"draft_id"`
	MissionID     string `json:"mission_id"`
	SessionID     string `json:"session_id"`
	Content       string `json:"content"`
	Offset        int    `json:"offset"`
	NextOffset    int    `json:"next_offset,omitempty"`
	ContentLength int    `json:"content_length"`
	Truncated     bool   `json:"truncated"`
	Finalized     bool   `json:"finalized"`
	ArtifactID    string `json:"artifact_id,omitempty"`
}

type experimentReportFinalizeOutput struct {
	DraftID       string                               `json:"draft_id"`
	MissionID     string                               `json:"mission_id"`
	SessionID     string                               `json:"session_id"`
	ContentLength int                                  `json:"content_length"`
	Artifact      rawArtifactOutput                    `json:"artifact"`
	EventID       string                               `json:"event_id"`
	HumanizeReady *experimentReportHumanizeReadyOutput `json:"humanize_ready,omitempty"`
}

type reportPatchOutput struct {
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

type reportPatchReadOutput struct {
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

type reportPatchFinalizeOutput struct {
	PatchID        string            `json:"patch_id"`
	MissionID      string            `json:"mission_id"`
	SessionID      string            `json:"session_id"`
	BaseArtifactID string            `json:"base_artifact_id"`
	ContentLength  int               `json:"content_length"`
	Artifact       rawArtifactOutput `json:"artifact"`
	EventID        string            `json:"event_id"`
}

type experimentReportHumanizeReadyOutput struct {
	EventID                   string `json:"event_id,omitempty"`
	Profile                   string `json:"profile"`
	Target                    string `json:"target"`
	SourceArtifactID          string `json:"source_artifact_id"`
	SourceArtifactSHA256      string `json:"source_artifact_sha256"`
	PreservedOriginalMarkdown bool   `json:"preserved_original_markdown"`
	Reason                    string `json:"reason"`
}

type sourcesSnapshotInput struct {
	CommonMutatingInput
	Connector  connectorRefInput   `json:"connector"`
	ArtifactID string              `json:"artifact_id"`
	SnapshotID string              `json:"snapshot_id"`
	EventID    string              `json:"event_id"`
	Ranges     []contentRangeInput `json:"ranges"`
	Reason     string              `json:"reason"`
}

type sourcesSnapshotOutput struct {
	SnapshotID  string   `json:"snapshot_id"`
	ArtifactIDs []string `json:"artifact_ids"`
}

type localPathRootsInput struct {
	MissionID string `json:"mission_id"`
}

type localPathRootsOutput struct {
	Roots []localpath.RootView `json:"roots"`
}

type localPathTreeInput struct {
	MissionID    string `json:"mission_id"`
	RootID       string `json:"root_id"`
	RelativePath string `json:"relative_path"`
	Depth        int    `json:"depth"`
	Limit        int    `json:"limit"`
}

type localPathTreeOutput struct {
	Tree localpath.TreeResult `json:"tree"`
}

type localPathAttachInput struct {
	CommonMutatingInput
	SnapshotID   string `json:"snapshot_id"`
	RootID       string `json:"root_id"`
	RelativePath string `json:"relative_path"`
	Title        string `json:"title"`
	Restore      bool   `json:"restore"`
}

type localPathAttachOutput struct {
	Snapshot        sourceSnapshotOutput `json:"snapshot"`
	EventID         string               `json:"event_id,omitempty"`
	Existing        bool                 `json:"existing"`
	Restored        bool                 `json:"restored"`
	RestoreRequired bool                 `json:"restore_required,omitempty"`
}

type sourceRemoveInput struct {
	CommonMutatingInput
	SnapshotID string `json:"snapshot_id"`
	Reason     string `json:"reason"`
}

type sourceRestoreInput struct {
	CommonMutatingInput
	SnapshotID string `json:"snapshot_id"`
}

type sourceStateChangeOutput struct {
	Snapshot   sourceSnapshotOutput `json:"snapshot"`
	EventID    string               `json:"event_id,omitempty"`
	Idempotent bool                 `json:"idempotent"`
}

type evidenceProposeInput struct {
	CommonMutatingInput
	EvidenceID      string            `json:"evidence_id"`
	EventID         string            `json:"event_id"`
	ProposalID      string            `json:"proposal_id"`
	ProposalEventID string            `json:"proposal_event_id"`
	ProposalTitle   string            `json:"proposal_title"`
	Summary         string            `json:"summary"`
	EvidenceType    string            `json:"evidence_type"`
	SnapshotRefs    []app.SnapshotRef `json:"snapshot_refs"`
	Confidence      app.Confidence    `json:"confidence"`
}

type questionsProposeInput struct {
	CommonMutatingInput
	QuestionID         string   `json:"question_id"`
	EventID            string   `json:"event_id"`
	ProposalID         string   `json:"proposal_id"`
	ProposalEventID    string   `json:"proposal_event_id"`
	ProposalTitle      string   `json:"proposal_title"`
	Text               string   `json:"text"`
	Priority           string   `json:"priority"`
	Blocking           bool     `json:"blocking"`
	RelatedEvidenceIDs []string `json:"related_evidence_ids"`
	RelatedClaimIDs    []string `json:"related_claim_ids"`
}

type claimsProposeInput struct {
	CommonMutatingInput
	ClaimID               string         `json:"claim_id"`
	EventID               string         `json:"event_id"`
	ProposalID            string         `json:"proposal_id"`
	ProposalEventID       string         `json:"proposal_event_id"`
	ProposalTitle         string         `json:"proposal_title"`
	Text                  string         `json:"text"`
	ClaimType             string         `json:"claim_type"`
	SupportingEvidenceIDs []string       `json:"supporting_evidence_ids"`
	OpposingEvidenceIDs   []string       `json:"opposing_evidence_ids"`
	DependsOnQuestionIDs  []string       `json:"depends_on_question_ids"`
	UserAssertionEventID  string         `json:"user_assertion_event_id"`
	Confidence            app.Confidence `json:"confidence"`
}

type claimConfidenceInput struct {
	CommonMutatingInput
	ClaimID          string         `json:"claim_id"`
	EventID          string         `json:"event_id"`
	Confidence       app.Confidence `json:"confidence"`
	BasisEvidenceIDs []string       `json:"basis_evidence_ids"`
	CausationEventID string         `json:"causation_event_id"`
	CorrelationID    string         `json:"correlation_id"`
}

type proposalsSubmitInput struct {
	CommonMutatingInput
	ProposalID string          `json:"proposal_id"`
	EventID    string          `json:"event_id"`
	Title      string          `json:"title"`
	ObjectRefs []app.ObjectRef `json:"object_refs"`
}

type connectorRefInput struct {
	ConnectorID      string `json:"connector_id"`
	ConnectorType    string `json:"connector_type"`
	ExternalSourceID string `json:"external_source_id"`
	ExternalURI      string `json:"external_uri"`
	ExternalVersion  string `json:"external_version"`
	ConnectorVersion string `json:"connector_version"`
}

func (input connectorRefInput) toApp() app.ConnectorRef {
	return app.ConnectorRef{
		ConnectorID:      input.ConnectorID,
		ConnectorType:    input.ConnectorType,
		ExternalSourceID: input.ExternalSourceID,
		ExternalURI:      input.ExternalURI,
		ExternalVersion:  input.ExternalVersion,
		ConnectorVersion: input.ConnectorVersion,
	}
}

type contentRangeInput struct {
	ContentID string `json:"content_id"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
}

func (input contentRangeInput) toApp() app.Liquid2ContentRange {
	return app.Liquid2ContentRange{
		ContentID: input.ContentID,
		Start:     input.Start,
		End:       input.End,
	}
}

type sourceCandidateOutput struct {
	Connector     connectorRefInput    `json:"connector"`
	Title         string               `json:"title"`
	SourceURI     string               `json:"source_uri"`
	Summary       string               `json:"summary"`
	MatchedRanges []matchedRangeOutput `json:"matched_ranges,omitempty"`
	UpdatedAt     string               `json:"updated_at,omitempty"`
	CanSnapshot   bool                 `json:"can_snapshot"`
}

type matchedRangeOutput struct {
	ContentID string `json:"content_id,omitempty"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
}

type sourceSnapshotOutput struct {
	SnapshotID        string            `json:"snapshot_id"`
	MissionID         string            `json:"mission_id"`
	Connector         connectorRefInput `json:"connector"`
	Title             string            `json:"title"`
	CapturedAt        string            `json:"captured_at,omitempty"`
	ExternalUpdatedAt string            `json:"external_updated_at,omitempty"`
	ArtifactIDs       []string          `json:"artifact_ids"`
	ContentHash       app.ContentHash   `json:"content_hash"`
	Locators          json.RawMessage   `json:"locators"`
	Access            app.SourceAccess  `json:"access"`
	RetrievalPolicy   string            `json:"retrieval_policy"`
	State             app.SourceState   `json:"state"`
}

type rawArtifactOutput struct {
	ArtifactID string `json:"artifact_id"`
	MissionID  string `json:"mission_id"`
	MediaType  string `json:"media_type"`
	ByteSize   int64  `json:"byte_size"`
	SHA256     string `json:"sha256"`
	StorageURI string `json:"storage_uri"`
	Filename   string `json:"filename,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	ReadKind   string `json:"read_kind,omitempty"`
}

func sourceCandidateFromApp(candidate app.Liquid2SourceCandidate) sourceCandidateOutput {
	ranges := make([]matchedRangeOutput, 0, len(candidate.MatchedRanges))
	for _, matchedRange := range candidate.MatchedRanges {
		ranges = append(ranges, matchedRangeOutput{
			ContentID: matchedRange.ContentID,
			Start:     matchedRange.Start,
			End:       matchedRange.End,
		})
	}
	output := sourceCandidateOutput{
		Connector:     connectorRefFromApp(candidate.Connector),
		Title:         candidate.Title,
		SourceURI:     candidate.SourceURI,
		Summary:       candidate.Summary,
		CanSnapshot:   candidate.CanSnapshot,
		MatchedRanges: ranges,
	}
	if !candidate.UpdatedAt.IsZero() {
		output.UpdatedAt = candidate.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return output
}

func sourceCandidateFromConfluence(candidate app.ConfluenceSourceCandidate) sourceCandidateOutput {
	output := sourceCandidateOutput{
		Connector:   connectorRefFromApp(candidate.Connector),
		Title:       candidate.Title,
		SourceURI:   candidate.SourceURI,
		CanSnapshot: candidate.CanSnapshot,
	}
	if !candidate.UpdatedAt.IsZero() {
		output.UpdatedAt = candidate.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return output
}

func sourceSnapshotFromApp(snapshot app.SourceSnapshot) sourceSnapshotOutput {
	output := sourceSnapshotOutput{
		SnapshotID:      snapshot.SnapshotID,
		MissionID:       snapshot.MissionID,
		Connector:       connectorRefFromApp(snapshot.Connector),
		Title:           snapshot.Title,
		ArtifactIDs:     append([]string(nil), snapshot.ArtifactIDs...),
		ContentHash:     snapshot.ContentHash,
		Locators:        append(json.RawMessage(nil), snapshot.Locators...),
		Access:          snapshot.Access,
		RetrievalPolicy: sourceRetrievalPolicy(snapshot),
		State:           sourceState(snapshot),
	}
	if strings.TrimSpace(output.Access.RetrievalPolicy) == "" {
		output.Access.RetrievalPolicy = output.RetrievalPolicy
	}
	if !snapshot.CapturedAt.IsZero() {
		output.CapturedAt = snapshot.CapturedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	if !snapshot.ExternalUpdatedAt.IsZero() {
		output.ExternalUpdatedAt = snapshot.ExternalUpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return output
}

func sourceSnapshotsFromApp(snapshots []app.SourceSnapshot) []sourceSnapshotOutput {
	output := make([]sourceSnapshotOutput, 0, len(snapshots))
	for _, snapshot := range snapshots {
		output = append(output, sourceSnapshotFromApp(snapshot))
	}
	return output
}

func rawArtifactFromApp(artifact app.RawArtifact) rawArtifactOutput {
	output := rawArtifactOutput{
		ArtifactID: artifact.ArtifactID,
		MissionID:  artifact.MissionID,
		MediaType:  artifact.MediaType,
		ByteSize:   artifact.ByteSize,
		SHA256:     artifact.SHA256,
		StorageURI: artifact.StorageURI,
		Filename:   artifact.Filename,
		ReadKind:   app.UploadedArtifactReadKind(artifact),
	}
	if !artifact.CreatedAt.IsZero() {
		output.CreatedAt = artifact.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return output
}

func connectorRefFromApp(connector app.ConnectorRef) connectorRefInput {
	return connectorRefInput{
		ConnectorID:      connector.ConnectorID,
		ConnectorType:    connector.ConnectorType,
		ExternalSourceID: connector.ExternalSourceID,
		ExternalURI:      connector.ExternalURI,
		ExternalVersion:  connector.ExternalVersion,
		ConnectorVersion: connector.ConnectorVersion,
	}
}

func contentRangesToApp(inputs []contentRangeInput) []app.Liquid2ContentRange {
	ranges := make([]app.Liquid2ContentRange, 0, len(inputs))
	for _, input := range inputs {
		ranges = append(ranges, input.toApp())
	}
	return ranges
}

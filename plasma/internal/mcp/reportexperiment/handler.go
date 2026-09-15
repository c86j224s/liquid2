package reportexperiment

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

type State struct{ Drafts map[string]*Draft }
type Service interface {
	GetRawArtifact(context.Context, string) (artifactcontract.Raw, error)
	CreateRawArtifact(context.Context, artifactcontract.CreateRequest) (artifactcontract.Raw, error)
	AppendEvent(context.Context, ledger.AppendRequest) (ledger.Event, error)
}
type Handler struct {
	State                *State
	Mu                   *sync.Mutex
	Service              Service
	MissionID, SessionID func() string
	Decode               func(json.RawMessage, any) error
	NormalizeInput       func(wire.CommonMutatingInput) (wire.CommonMutatingInput, ledger.Producer, error)
	ErrorResult          func(string, string, string, string, bool, []string) wire.ToolResult
	ErrorFromErr         func(string, string, error, []string) wire.ToolResult
	NewID                func(string) string
	ArtifactOutput       func(artifactcontract.Raw) wire.RawArtifactOutput
}

const (
	experimentReportMaxDrafts       = 8
	experimentReportMaxAppendBytes  = 256 * 1024
	experimentReportMaxDraftBytes   = 2 * 1024 * 1024
	experimentReportMaxChunks       = 64
	experimentReportDefaultReadSize = 32 * 1024
	experimentReportMaxReadSize     = 64 * 1024
	ExperimentReportHumanizeProfile = "h5-full-report-tone-pass"
	ExperimentReportHumanizeTarget  = "humanized_markdown"
	ExperimentReportHumanizeReason  = "mcp_finalize_does_not_spawn_nested_agent"
)

type Draft struct {
	DraftID              string
	MissionID            string
	SessionID            string
	Title                string
	Content              string
	ChunkCount           int
	Finalized            bool
	ArtifactID           string
	HumanizeReadyEventID string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (server *Handler) Create(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	var input ExperimentReportCreateInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundExperimentReportSession(common); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if draftID == "" {
		draftID = server.NewID("rpd")
	}
	if err := validateID("rpd_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	now := time.Now().UTC()
	draft := &Draft{
		DraftID:   draftID,
		MissionID: common.MissionID,
		SessionID: common.SessionID,
		Title:     strings.TrimSpace(input.Title),
		CreatedAt: now,
		UpdatedAt: now,
	}

	server.Mu.Lock()
	defer server.Mu.Unlock()
	if len(server.State.Drafts) >= experimentReportMaxDrafts {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "too many in-process experiment report drafts", false, nil)
	}
	if _, exists := server.State.Drafts[draftID]; exists {
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "experiment report draft already exists", false, []string{draftID})
	}
	server.State.Drafts[draftID] = draft
	return wire.ToolResult{
		ToolName:  call.Name,
		MissionID: common.MissionID,
		Content:   experimentReportDraftFromState(*draft),
	}
}

func (server *Handler) Append(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	var input ExperimentReportAppendInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundExperimentReportSession(common); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rpd_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	content := input.Content
	if strings.TrimSpace(content) == "" {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "report draft append content is required", false, []string{draftID})
	}
	if !utf8.ValidString(content) {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "report draft append content must be UTF-8 text", false, []string{draftID})
	}
	if len([]byte(content)) > experimentReportMaxAppendBytes {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "report draft append content is too large", false, []string{draftID})
	}

	server.Mu.Lock()
	defer server.Mu.Unlock()
	draft, ok := server.State.Drafts[draftID]
	if !ok {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "experiment report draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validateExperimentReportDraftAccess(draft, common.MissionID, common.SessionID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Finalized {
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "experiment report draft is already finalized", false, []string{draftID, draft.ArtifactID})
	}
	if draft.ChunkCount >= experimentReportMaxChunks {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "report draft has too many append chunks", false, []string{draftID})
	}
	if len([]byte(draft.Content))+len([]byte(content)) > experimentReportMaxDraftBytes {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "report draft content is too large", false, []string{draftID})
	}
	draft.Content += content
	draft.ChunkCount++
	draft.UpdatedAt = time.Now().UTC()
	return wire.ToolResult{
		ToolName:  call.Name,
		MissionID: common.MissionID,
		Content:   experimentReportDraftFromState(*draft),
	}
}

func (server *Handler) Read(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	var input ExperimentReportReadInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	sessionID := strings.TrimSpace(input.SessionID)
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("mis_", missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("ses_", sessionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("rpd_", draftID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	if err := server.requireBoundExperimentReportSession(wire.CommonMutatingInput{MissionID: missionID, SessionID: sessionID}); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}

	server.Mu.Lock()
	draft, ok := server.State.Drafts[draftID]
	if !ok {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, missionID, "validation", "experiment report draft was not found in this MCP process", false, []string{draftID})
	}
	copyDraft := *draft
	server.Mu.Unlock()
	if err := validateExperimentReportDraftAccess(&copyDraft, missionID, sessionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	content, offset, nextOffset, truncated, err := boundedReportDraftContent(copyDraft.Content, input.Offset, input.MaxBytes)
	if err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	return wire.ToolResult{
		ToolName:  call.Name,
		MissionID: missionID,
		Content: ExperimentReportReadOutput{
			DraftID:       copyDraft.DraftID,
			MissionID:     copyDraft.MissionID,
			SessionID:     copyDraft.SessionID,
			Content:       content,
			Offset:        offset,
			NextOffset:    nextOffset,
			ContentLength: len([]byte(copyDraft.Content)),
			Truncated:     truncated,
			Finalized:     copyDraft.Finalized,
			ArtifactID:    copyDraft.ArtifactID,
		},
	}
}

func (server *Handler) Finalize(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input ExperimentReportFinalizeInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundExperimentReportSession(common); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rpd_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	artifactID := strings.TrimSpace(input.ArtifactID)
	if artifactID == "" {
		artifactID = server.NewID("art")
	}
	if err := validateID("art_", artifactID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{artifactID})
	}

	server.Mu.Lock()
	draft, ok := server.State.Drafts[draftID]
	if !ok {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "validation", "experiment report draft was not found in this MCP process", false, []string{draftID})
	}
	copyDraft := *draft
	server.Mu.Unlock()
	if err := validateExperimentReportDraftAccess(&copyDraft, common.MissionID, common.SessionID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if copyDraft.Finalized {
		return server.experimentReportFinalizedResult(ctx, call.Name, copyDraft)
	}
	content := strings.TrimSpace(copyDraft.Content)
	if content == "" {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "report draft content is required before finalization", false, []string{draftID})
	}
	title := firstNonEmpty(input.Title, copyDraft.Title, "Experiment report")
	filename := safeExperimentReportFilename(firstNonEmpty(input.Filename, title))
	artifact, err := server.Service.CreateRawArtifact(ctx, artifactcontract.CreateRequest{
		ArtifactID:     artifactID,
		MissionID:      common.MissionID,
		MediaType:      "text/markdown; charset=utf-8",
		Filename:       filename,
		Producer:       ledger.Producer{Type: "mcp_tool", ID: mcptools.ToolExperimentReportFinalize},
		Content:        []byte(copyDraft.Content),
		ExpectedSHA256: strings.TrimSpace(input.ExpectedSHA256),
	})
	if err != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{draftID, artifactID})
	}
	eventID := server.NewID("evt")
	event, err := server.Service.AppendEvent(ctx, ledger.AppendRequest{
		EventID:       eventID,
		MissionID:     common.MissionID,
		EventType:     "experiment.report.artifact.created",
		Producer:      ledger.Producer{Type: "mcp_tool", ID: mcptools.ToolExperimentReportFinalize},
		CorrelationID: common.SessionID,
		Payload: mustJSON(map[string]any{
			"kind":               "experimental_mcp_markdown_report_artifact",
			"draft_id":           draftID,
			"title":              title,
			"artifact_id":        artifact.ArtifactID,
			"media_type":         artifact.MediaType,
			"byte_size":          artifact.ByteSize,
			"sha256":             artifact.SHA256,
			"filename":           artifact.Filename,
			"tool_session_id":    common.SessionID,
			"agent_session_id":   common.SessionID,
			"producer_tool_name": mcptools.ToolExperimentReportFinalize,
			"experiment_feature": "report_composition_mcp_artifact",
		}),
	})
	if err != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{draftID, artifact.ArtifactID})
	}
	createdEventIDs := []string{event.EventID}
	var readyOutput *ExperimentReportHumanizeReadyOutput
	readyEventID := server.NewID("evt")
	readyCandidate := ExperimentReportHumanizeReadyOutput{
		EventID:                   readyEventID,
		Profile:                   ExperimentReportHumanizeProfile,
		Target:                    ExperimentReportHumanizeTarget,
		SourceArtifactID:          artifact.ArtifactID,
		SourceArtifactSHA256:      artifact.SHA256,
		PreservedOriginalMarkdown: true,
		Reason:                    ExperimentReportHumanizeReason,
	}
	readyEvent, err := server.Service.AppendEvent(ctx, ledger.AppendRequest{
		EventID:       readyEventID,
		MissionID:     common.MissionID,
		EventType:     "experiment.report.humanize.ready",
		Producer:      ledger.Producer{Type: "mcp_tool", ID: mcptools.ToolExperimentReportFinalize},
		CorrelationID: common.SessionID,
		Payload: mustJSON(map[string]any{
			"kind":                        "experimental_mcp_report_humanize_ready",
			"profile":                     readyCandidate.Profile,
			"target":                      readyCandidate.Target,
			"source_artifact_id":          readyCandidate.SourceArtifactID,
			"source_artifact_sha256":      readyCandidate.SourceArtifactSHA256,
			"preserved_original_markdown": readyCandidate.PreservedOriginalMarkdown,
			"reason":                      readyCandidate.Reason,
			"producer_tool_name":          mcptools.ToolExperimentReportFinalize,
			"experiment_feature":          "report_composition_mcp_artifact",
		}),
	})
	if err == nil {
		readyCandidate.EventID = readyEvent.EventID
		readyOutput = &readyCandidate
		createdEventIDs = append(createdEventIDs, readyEvent.EventID)
	}

	server.Mu.Lock()
	if current, ok := server.State.Drafts[draftID]; ok {
		current.Finalized = true
		current.ArtifactID = artifact.ArtifactID
		if readyOutput != nil {
			current.HumanizeReadyEventID = readyOutput.EventID
		}
		current.UpdatedAt = time.Now().UTC()
	}
	server.Mu.Unlock()
	return wire.ToolResult{
		ToolName:        call.Name,
		MissionID:       common.MissionID,
		CreatedEventIDs: createdEventIDs,
		Content: ExperimentReportFinalizeOutput{
			DraftID:       draftID,
			MissionID:     common.MissionID,
			SessionID:     common.SessionID,
			ContentLength: len([]byte(copyDraft.Content)),
			Artifact:      server.ArtifactOutput(artifact),
			EventID:       event.EventID,
			HumanizeReady: readyOutput,
		},
	}
}

func (server *Handler) experimentReportFinalizedResult(ctx context.Context, toolName string, draft Draft) wire.ToolResult {
	artifact, err := server.Service.GetRawArtifact(ctx, draft.ArtifactID)
	if err != nil {
		return server.ErrorFromErr(toolName, draft.MissionID, err, []string{draft.DraftID, draft.ArtifactID})
	}
	return wire.ToolResult{
		ToolName:  toolName,
		MissionID: draft.MissionID,
		Content: ExperimentReportFinalizeOutput{
			DraftID:       draft.DraftID,
			MissionID:     draft.MissionID,
			SessionID:     draft.SessionID,
			ContentLength: len([]byte(draft.Content)),
			Artifact:      server.ArtifactOutput(artifact),
			HumanizeReady: experimentReportHumanizeReadyFromDraft(draft, artifact),
		},
	}
}

func experimentReportHumanizeReadyFromDraft(draft Draft, artifact artifactcontract.Raw) *ExperimentReportHumanizeReadyOutput {
	if strings.TrimSpace(draft.HumanizeReadyEventID) == "" {
		return nil
	}
	return &ExperimentReportHumanizeReadyOutput{
		EventID:                   draft.HumanizeReadyEventID,
		Profile:                   ExperimentReportHumanizeProfile,
		Target:                    ExperimentReportHumanizeTarget,
		SourceArtifactID:          artifact.ArtifactID,
		SourceArtifactSHA256:      artifact.SHA256,
		PreservedOriginalMarkdown: true,
		Reason:                    ExperimentReportHumanizeReason,
	}
}

func (server *Handler) requireBoundExperimentReportSession(input wire.CommonMutatingInput) error {
	boundMissionID := strings.TrimSpace(server.MissionID())
	boundSessionID := strings.TrimSpace(server.SessionID())
	if boundMissionID == "" || boundSessionID == "" {
		return fmt.Errorf("%w: experimental report composition tools require a mission-bound MCP agent session", producterror.ErrInvalidInput)
	}
	if input.MissionID != boundMissionID || input.SessionID != boundSessionID {
		return fmt.Errorf("%w: tool call is outside this MCP session", producterror.ErrInvalidInput)
	}
	return nil
}

func validateExperimentReportDraftAccess(draft *Draft, missionID string, sessionID string) error {
	if draft == nil {
		return fmt.Errorf("%w: experiment report draft is required", producterror.ErrInvalidInput)
	}
	if draft.MissionID != missionID || draft.SessionID != sessionID {
		return fmt.Errorf("%w: experiment report draft belongs to another MCP session", producterror.ErrInvalidInput)
	}
	return nil
}

func experimentReportDraftFromState(draft Draft) ExperimentReportDraftOutput {
	state := "open"
	if draft.Finalized {
		state = "finalized"
	}
	return ExperimentReportDraftOutput{
		DraftID:       draft.DraftID,
		MissionID:     draft.MissionID,
		SessionID:     draft.SessionID,
		Title:         draft.Title,
		State:         state,
		ContentLength: len([]byte(draft.Content)),
		ChunkCount:    draft.ChunkCount,
		Finalized:     draft.Finalized,
		ArtifactID:    draft.ArtifactID,
	}
}

func boundedReportDraftContent(content string, offset int, maxBytes int) (string, int, int, bool, error) {
	raw := []byte(content)
	if offset < 0 {
		return "", 0, 0, false, fmt.Errorf("%w: report draft offset must be non-negative", producterror.ErrInvalidInput)
	}
	if offset > len(raw) {
		return "", 0, 0, false, fmt.Errorf("%w: report draft offset is beyond content length", producterror.ErrInvalidInput)
	}
	if offset < len(raw) && !utf8.RuneStart(raw[offset]) {
		return "", 0, 0, false, fmt.Errorf("%w: report draft offset must align to UTF-8 boundary", producterror.ErrInvalidInput)
	}
	limit := maxBytes
	if limit <= 0 {
		limit = experimentReportDefaultReadSize
	} else if limit > experimentReportMaxReadSize {
		limit = experimentReportMaxReadSize
	}
	remaining := raw[offset:]
	if len(remaining) <= limit {
		return string(remaining), offset, 0, false, nil
	}
	cut := offset + limit
	for cut > offset && !utf8.Valid(raw[offset:cut]) {
		cut--
	}
	if cut == offset {
		return "", 0, 0, false, fmt.Errorf("%w: report draft could not be sliced as UTF-8", producterror.ErrInvalidInput)
	}
	return string(raw[offset:cut]), offset, cut, true, nil
}

func safeExperimentReportFilename(value string) string {
	base := strings.TrimSpace(value)
	if base == "" {
		base = "experiment-report"
	}
	base = filepath.Base(base)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	var builder strings.Builder
	for _, r := range strings.ToLower(base) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
		case unicode.IsLetter(r), unicode.IsNumber(r):
			builder.WriteRune(r)
		case r == '-' || r == '_':
			builder.WriteRune(r)
		case unicode.IsSpace(r) || r == '.':
			builder.WriteRune('-')
		}
		if builder.Len() >= 80 {
			break
		}
	}
	name := strings.Trim(builder.String(), "-_")
	if name == "" {
		name = "experiment-report"
	}
	return name + ".md"
}

func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", producterror.ErrInvalidInput, prefix)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

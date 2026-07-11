package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	experimentReportMaxDrafts       = 8
	experimentReportMaxAppendBytes  = 256 * 1024
	experimentReportMaxDraftBytes   = 2 * 1024 * 1024
	experimentReportMaxChunks       = 64
	experimentReportDefaultReadSize = 32 * 1024
	experimentReportMaxReadSize     = 64 * 1024
	experimentReportHumanizeProfile = "h5-full-report-tone-pass"
	experimentReportHumanizeTarget  = "humanized_markdown"
	experimentReportHumanizeReason  = "mcp_finalize_does_not_spawn_nested_agent"
)

type experimentReportDraft struct {
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

func (server *Server) callExperimentReportCreate(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	var input experimentReportCreateInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundExperimentReportSession(common); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if draftID == "" {
		draftID = newMCPID("rpd")
	}
	if err := validateID("rpd_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	now := time.Now().UTC()
	draft := &experimentReportDraft{
		DraftID:   draftID,
		MissionID: common.MissionID,
		SessionID: common.SessionID,
		Title:     strings.TrimSpace(input.Title),
		CreatedAt: now,
		UpdatedAt: now,
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.reportDrafts) >= experimentReportMaxDrafts {
		return errorResult(call.Name, common.MissionID, "validation", "too many in-process experiment report drafts", false, nil)
	}
	if _, exists := server.reportDrafts[draftID]; exists {
		return errorResult(call.Name, common.MissionID, "conflict", "experiment report draft already exists", false, []string{draftID})
	}
	server.reportDrafts[draftID] = draft
	return ToolResult{
		ToolName:  call.Name,
		MissionID: common.MissionID,
		Content:   experimentReportDraftFromState(*draft),
	}
}

func (server *Server) callExperimentReportAppend(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	var input experimentReportAppendInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundExperimentReportSession(common); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rpd_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	content := input.Content
	if strings.TrimSpace(content) == "" {
		return errorResult(call.Name, common.MissionID, "validation", "report draft append content is required", false, []string{draftID})
	}
	if !utf8.ValidString(content) {
		return errorResult(call.Name, common.MissionID, "validation", "report draft append content must be UTF-8 text", false, []string{draftID})
	}
	if len([]byte(content)) > experimentReportMaxAppendBytes {
		return errorResult(call.Name, common.MissionID, "validation", "report draft append content is too large", false, []string{draftID})
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	draft, ok := server.reportDrafts[draftID]
	if !ok {
		return errorResult(call.Name, common.MissionID, "validation", "experiment report draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validateExperimentReportDraftAccess(draft, common.MissionID, common.SessionID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Finalized {
		return errorResult(call.Name, common.MissionID, "conflict", "experiment report draft is already finalized", false, []string{draftID, draft.ArtifactID})
	}
	if draft.ChunkCount >= experimentReportMaxChunks {
		return errorResult(call.Name, common.MissionID, "validation", "report draft has too many append chunks", false, []string{draftID})
	}
	if len([]byte(draft.Content))+len([]byte(content)) > experimentReportMaxDraftBytes {
		return errorResult(call.Name, common.MissionID, "validation", "report draft content is too large", false, []string{draftID})
	}
	draft.Content += content
	draft.ChunkCount++
	draft.UpdatedAt = time.Now().UTC()
	return ToolResult{
		ToolName:  call.Name,
		MissionID: common.MissionID,
		Content:   experimentReportDraftFromState(*draft),
	}
}

func (server *Server) callExperimentReportRead(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	var input experimentReportReadInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	sessionID := strings.TrimSpace(input.SessionID)
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("ses_", sessionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("rpd_", draftID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	if err := server.requireBoundExperimentReportSession(commonMutatingInput{MissionID: missionID, SessionID: sessionID}); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}

	server.mu.Lock()
	draft, ok := server.reportDrafts[draftID]
	if !ok {
		server.mu.Unlock()
		return errorResult(call.Name, missionID, "validation", "experiment report draft was not found in this MCP process", false, []string{draftID})
	}
	copyDraft := *draft
	server.mu.Unlock()
	if err := validateExperimentReportDraftAccess(&copyDraft, missionID, sessionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	content, offset, nextOffset, truncated, err := boundedReportDraftContent(copyDraft.Content, input.Offset, input.MaxBytes)
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	return ToolResult{
		ToolName:  call.Name,
		MissionID: missionID,
		Content: experimentReportReadOutput{
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

func (server *Server) callExperimentReportFinalize(ctx context.Context, call ToolCall) ToolResult {
	var input experimentReportFinalizeInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundExperimentReportSession(common); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rpd_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	artifactID := strings.TrimSpace(input.ArtifactID)
	if artifactID == "" {
		artifactID = newMCPID("art")
	}
	if err := validateID("art_", artifactID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{artifactID})
	}

	server.mu.Lock()
	draft, ok := server.reportDrafts[draftID]
	if !ok {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "validation", "experiment report draft was not found in this MCP process", false, []string{draftID})
	}
	copyDraft := *draft
	server.mu.Unlock()
	if err := validateExperimentReportDraftAccess(&copyDraft, common.MissionID, common.SessionID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if copyDraft.Finalized {
		return server.experimentReportFinalizedResult(ctx, call.Name, copyDraft)
	}
	content := strings.TrimSpace(copyDraft.Content)
	if content == "" {
		return errorResult(call.Name, common.MissionID, "validation", "report draft content is required before finalization", false, []string{draftID})
	}
	title := firstNonEmpty(input.Title, copyDraft.Title, "Experiment report")
	filename := safeExperimentReportFilename(firstNonEmpty(input.Filename, title))
	artifact, err := server.service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID:     artifactID,
		MissionID:      common.MissionID,
		MediaType:      "text/markdown; charset=utf-8",
		Filename:       filename,
		Producer:       app.Producer{Type: "mcp_tool", ID: ToolExperimentReportFinalize},
		Content:        []byte(copyDraft.Content),
		ExpectedSHA256: strings.TrimSpace(input.ExpectedSHA256),
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{draftID, artifactID})
	}
	eventID := newMCPID("evt")
	event, err := server.service.AppendEvent(ctx, app.AppendEventRequest{
		EventID:       eventID,
		MissionID:     common.MissionID,
		EventType:     "experiment.report.artifact.created",
		Producer:      app.Producer{Type: "mcp_tool", ID: ToolExperimentReportFinalize},
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
			"producer_tool_name": ToolExperimentReportFinalize,
			"experiment_feature": "report_composition_mcp_artifact",
		}),
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{draftID, artifact.ArtifactID})
	}
	createdEventIDs := []string{event.EventID}
	var readyOutput *experimentReportHumanizeReadyOutput
	readyEventID := newMCPID("evt")
	readyCandidate := experimentReportHumanizeReadyOutput{
		EventID:                   readyEventID,
		Profile:                   experimentReportHumanizeProfile,
		Target:                    experimentReportHumanizeTarget,
		SourceArtifactID:          artifact.ArtifactID,
		SourceArtifactSHA256:      artifact.SHA256,
		PreservedOriginalMarkdown: true,
		Reason:                    experimentReportHumanizeReason,
	}
	readyEvent, err := server.service.AppendEvent(ctx, app.AppendEventRequest{
		EventID:       readyEventID,
		MissionID:     common.MissionID,
		EventType:     "experiment.report.humanize.ready",
		Producer:      app.Producer{Type: "mcp_tool", ID: ToolExperimentReportFinalize},
		CorrelationID: common.SessionID,
		Payload: mustJSON(map[string]any{
			"kind":                        "experimental_mcp_report_humanize_ready",
			"profile":                     readyCandidate.Profile,
			"target":                      readyCandidate.Target,
			"source_artifact_id":          readyCandidate.SourceArtifactID,
			"source_artifact_sha256":      readyCandidate.SourceArtifactSHA256,
			"preserved_original_markdown": readyCandidate.PreservedOriginalMarkdown,
			"reason":                      readyCandidate.Reason,
			"producer_tool_name":          ToolExperimentReportFinalize,
			"experiment_feature":          "report_composition_mcp_artifact",
		}),
	})
	if err == nil {
		readyCandidate.EventID = readyEvent.EventID
		readyOutput = &readyCandidate
		createdEventIDs = append(createdEventIDs, readyEvent.EventID)
	}

	server.mu.Lock()
	if current, ok := server.reportDrafts[draftID]; ok {
		current.Finalized = true
		current.ArtifactID = artifact.ArtifactID
		if readyOutput != nil {
			current.HumanizeReadyEventID = readyOutput.EventID
		}
		current.UpdatedAt = time.Now().UTC()
	}
	server.mu.Unlock()
	return ToolResult{
		ToolName:        call.Name,
		MissionID:       common.MissionID,
		CreatedEventIDs: createdEventIDs,
		Content: experimentReportFinalizeOutput{
			DraftID:       draftID,
			MissionID:     common.MissionID,
			SessionID:     common.SessionID,
			ContentLength: len([]byte(copyDraft.Content)),
			Artifact:      rawArtifactFromApp(artifact),
			EventID:       event.EventID,
			HumanizeReady: readyOutput,
		},
	}
}

func (server *Server) experimentReportFinalizedResult(ctx context.Context, toolName string, draft experimentReportDraft) ToolResult {
	artifact, err := server.service.GetRawArtifact(ctx, draft.ArtifactID)
	if err != nil {
		return errorFromErr(toolName, draft.MissionID, err, []string{draft.DraftID, draft.ArtifactID})
	}
	return ToolResult{
		ToolName:  toolName,
		MissionID: draft.MissionID,
		Content: experimentReportFinalizeOutput{
			DraftID:       draft.DraftID,
			MissionID:     draft.MissionID,
			SessionID:     draft.SessionID,
			ContentLength: len([]byte(draft.Content)),
			Artifact:      rawArtifactFromApp(artifact),
			HumanizeReady: experimentReportHumanizeReadyFromDraft(draft, artifact),
		},
	}
}

func experimentReportHumanizeReadyFromDraft(draft experimentReportDraft, artifact app.RawArtifact) *experimentReportHumanizeReadyOutput {
	if strings.TrimSpace(draft.HumanizeReadyEventID) == "" {
		return nil
	}
	return &experimentReportHumanizeReadyOutput{
		EventID:                   draft.HumanizeReadyEventID,
		Profile:                   experimentReportHumanizeProfile,
		Target:                    experimentReportHumanizeTarget,
		SourceArtifactID:          artifact.ArtifactID,
		SourceArtifactSHA256:      artifact.SHA256,
		PreservedOriginalMarkdown: true,
		Reason:                    experimentReportHumanizeReason,
	}
}

func (server *Server) requireBoundExperimentReportSession(input commonMutatingInput) error {
	boundMissionID := strings.TrimSpace(server.binding.MissionID)
	boundSessionID := strings.TrimSpace(server.binding.AgentSessionID)
	if boundMissionID == "" || boundSessionID == "" {
		return fmt.Errorf("%w: experimental report composition tools require a mission-bound MCP agent session", app.ErrInvalidInput)
	}
	if input.MissionID != boundMissionID || input.SessionID != boundSessionID {
		return fmt.Errorf("%w: tool call is outside this MCP session", app.ErrInvalidInput)
	}
	return nil
}

func validateExperimentReportDraftAccess(draft *experimentReportDraft, missionID string, sessionID string) error {
	if draft == nil {
		return fmt.Errorf("%w: experiment report draft is required", app.ErrInvalidInput)
	}
	if draft.MissionID != missionID || draft.SessionID != sessionID {
		return fmt.Errorf("%w: experiment report draft belongs to another MCP session", app.ErrInvalidInput)
	}
	return nil
}

func experimentReportDraftFromState(draft experimentReportDraft) experimentReportDraftOutput {
	state := "open"
	if draft.Finalized {
		state = "finalized"
	}
	return experimentReportDraftOutput{
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
		return "", 0, 0, false, fmt.Errorf("%w: report draft offset must be non-negative", app.ErrInvalidInput)
	}
	if offset > len(raw) {
		return "", 0, 0, false, fmt.Errorf("%w: report draft offset is beyond content length", app.ErrInvalidInput)
	}
	if offset < len(raw) && !utf8.RuneStart(raw[offset]) {
		return "", 0, 0, false, fmt.Errorf("%w: report draft offset must align to UTF-8 boundary", app.ErrInvalidInput)
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
		return "", 0, 0, false, fmt.Errorf("%w: report draft could not be sliced as UTF-8", app.ErrInvalidInput)
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

package web

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	htmlpkg "html"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

func (server *Server) handleMissionReports(w http.ResponseWriter, r *http.Request, missionID string, rest []string) {
	if len(rest) == 1 && rest[0] == "cancel" {
		server.handleCancelMissionReport(w, r, missionID)
		return
	}
	if len(rest) == 1 && rest[0] == "patch" {
		server.handlePatchMissionReport(w, r, missionID)
		return
	}
	if len(rest) != 0 {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		reports, err := server.service.ListReports(r.Context(), missionID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		versions, err := server.service.ListReportVersions(r.Context(), missionID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reports": reports, "versions": versions})
	case http.MethodPost:
		var req reportDraftRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		result, err := server.startReportDraft(r.Context(), missionID, req)
		if err != nil {
			if errors.Is(err, errReportDraftRunning) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, result)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (server *Server) handlePatchMissionReport(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req reportPatchRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := server.startReportPatch(r.Context(), missionID, req)
	if err != nil {
		if errors.Is(err, errReportDraftRunning) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (server *Server) handleCancelMissionReport(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	event, canceledInFlight, err := server.cancelReportDraft(r.Context(), missionID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"canceled": true, "in_flight": canceledInFlight, "event": event})
}

func (server *Server) cancelReportDraft(ctx context.Context, missionID string) (app.LedgerEvent, bool, error) {
	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	pending, ok := latestOpenReportDraftPendingEvent(events)
	if !ok {
		return app.LedgerEvent{}, false, fmt.Errorf("%w: no report draft is running for this mission", app.ErrInvalidInput)
	}
	cancelInFlightPendingEventID := server.reportCancelInFlightPendingEventID(missionID, pending)
	canceledInFlight := false
	if pending.EventType != "report.humanize.pending" {
		canceledInFlight = server.runningReports.Cancel(missionID, cancelInFlightPendingEventID)
	} else {
		canceledInFlight = server.runningReports.Owns(missionID, cancelInFlightPendingEventID)
	}
	event, err := server.reportRunner().AppendCanceled(ctx, missionID, pending, canceledInFlight, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		return app.LedgerEvent{}, canceledInFlight, err
	}
	if pending.EventType == "report.humanize.pending" && canceledInFlight {
		server.runningReports.Cancel(missionID, cancelInFlightPendingEventID)
	}
	return event, canceledInFlight, nil
}

func (server *Server) reportCancelInFlightPendingEventID(missionID string, pending app.LedgerEvent) string {
	if pending.EventType == "report.humanize.pending" {
		if reportPendingEventID := reportHumanizeInFlightPendingEventID(pending); reportPendingEventID != "" && server.runningReports.Owns(missionID, reportPendingEventID) {
			return reportPendingEventID
		}
	}
	return pending.EventID
}

func (server *Server) handleMissionArtifacts(w http.ResponseWriter, r *http.Request, missionID string, rest []string) {
	if len(rest) != 1 && !(len(rest) == 2 && (rest[1] == "download" || rest[1] == "html_export" || rest[1] == "designed_html_export" || rest[1] == "humanized_markdown_export")) {
		http.NotFound(w, r)
		return
	}
	if len(rest) == 2 && rest[1] == "humanized_markdown_export" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		server.handleReportArtifactHumanizedMarkdownExport(w, r, missionID, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "designed_html_export" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		server.handleReportArtifactDesignedHTMLExport(w, r, missionID, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "html_export" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		server.handleReportArtifactHTMLExport(w, r, missionID, rest[0])
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	artifact, err := server.service.GetRawArtifact(r.Context(), rest[0])
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		writeAppError(w, err)
		return
	}
	if artifact.MissionID != missionID {
		writeError(w, http.StatusNotFound, "artifact not found")
		return
	}
	if ok, err := server.isReportArtifact(r.Context(), missionID, artifact.ArtifactID); err != nil {
		writeAppError(w, err)
		return
	} else if !ok {
		writeError(w, http.StatusNotFound, "artifact not found")
		return
	}
	if len(rest) == 2 {
		writeRawArtifactDownload(w, artifact)
		return
	}
	writeRawArtifactFullPreview(w, artifact)
}

func (server *Server) handleReportArtifactHTMLExport(w http.ResponseWriter, r *http.Request, missionID string, artifactID string) {
	sourceArtifact, err := server.reportArtifact(r.Context(), missionID, artifactID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result, err := server.exportMarkdownArtifactAsHTML(r.Context(), missionID, sourceArtifact)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"artifact":        rawArtifactMetadata(result.Artifact),
		"source_artifact": rawArtifactMetadata(sourceArtifact),
		"event":           result.Event,
		"content":         string(result.Artifact.Content),
	})
}

func (server *Server) handleReportArtifactDesignedHTMLExport(w http.ResponseWriter, r *http.Request, missionID string, artifactID string) {
	sourceArtifact, err := server.reportArtifact(r.Context(), missionID, artifactID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	var req reportDesignRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, started, err := server.startDesignedReportHTMLExport(r.Context(), missionID, sourceArtifact, req)
	if err != nil {
		if errors.Is(err, errReportDraftRunning) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeAppError(w, err)
		return
	}
	status := http.StatusOK
	if started {
		status = http.StatusAccepted
	}
	writeJSON(w, status, result)
}

func (server *Server) handleReportArtifactHumanizedMarkdownExport(w http.ResponseWriter, r *http.Request, missionID string, artifactID string) {
	sourceArtifact, err := server.reportArtifact(r.Context(), missionID, artifactID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	var req reportHumanizeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := server.startReportHumanize(r.Context(), missionID, sourceArtifact, req)
	if err != nil {
		if errors.Is(err, errReportDraftRunning) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (server *Server) reportArtifact(ctx context.Context, missionID string, artifactID string) (app.RawArtifact, error) {
	artifact, err := server.service.GetRawArtifact(ctx, artifactID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return app.RawArtifact{}, fmt.Errorf("%w: artifact not found", app.ErrInvalidInput)
		}
		return app.RawArtifact{}, err
	}
	if artifact.MissionID != missionID {
		return app.RawArtifact{}, fmt.Errorf("%w: artifact not found", app.ErrInvalidInput)
	}
	if ok, err := server.isReportArtifact(ctx, missionID, artifact.ArtifactID); err != nil {
		return app.RawArtifact{}, err
	} else if !ok {
		return app.RawArtifact{}, fmt.Errorf("%w: artifact not found", app.ErrInvalidInput)
	}
	return artifact, nil
}

type reportArtifactSessionInfo struct {
	EventID                      string
	Kind                         string
	Title                        string
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSessionID               string
	PreviousAgentSessionID       string
	ReportSessionID              string
	ForkSourceAgentSessionID     string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	SessionChainKind             string
	ReportMode                   string
	ReportPendingEventID         string
}

type ReportPatchSessionSelection struct {
	SessionID                    string
	PreviousAgentSessionID       string
	ForkSourceAgentSessionID     string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	SessionChainKind             string
}

type reportPatchSessionSelection = ReportPatchSessionSelection

func (server *Server) reportArtifactSessionInfo(ctx context.Context, missionID string, artifactID string) (reportArtifactSessionInfo, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return reportArtifactSessionInfo{}, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "report.artifact.created" && event.EventType != "report.artifact.exported" {
			continue
		}
		var payload struct {
			Kind                         string `json:"kind"`
			ArtifactID                   string `json:"artifact_id"`
			Title                        string `json:"title"`
			AgentExecutor                string `json:"agent_executor"`
			AgentModel                   string `json:"agent_model"`
			AgentReasoningEffort         string `json:"agent_reasoning_effort"`
			AgentSessionID               string `json:"agent_session_id"`
			PreviousAgentSessionID       string `json:"previous_agent_session_id"`
			ReportSessionID              string `json:"report_session_id"`
			ForkSourceAgentSessionID     string `json:"fork_source_agent_session_id"`
			ReportSessionPolicy          string `json:"report_session_policy"`
			ReportSessionPolicySelection string `json:"report_session_policy_selection"`
			SessionChainKind             string `json:"session_chain_kind"`
			ReportMode                   string `json:"report_mode"`
			PendingEventID               string `json:"pending_event_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.ArtifactID) != artifactID {
			continue
		}
		info := reportArtifactSessionInfo{
			EventID:                      event.EventID,
			Kind:                         strings.TrimSpace(payload.Kind),
			Title:                        strings.TrimSpace(payload.Title),
			AgentExecutor:                strings.TrimSpace(payload.AgentExecutor),
			AgentModel:                   strings.TrimSpace(payload.AgentModel),
			AgentReasoningEffort:         strings.TrimSpace(payload.AgentReasoningEffort),
			AgentSessionID:               strings.TrimSpace(payload.AgentSessionID),
			PreviousAgentSessionID:       strings.TrimSpace(payload.PreviousAgentSessionID),
			ReportSessionID:              strings.TrimSpace(payload.ReportSessionID),
			ForkSourceAgentSessionID:     strings.TrimSpace(payload.ForkSourceAgentSessionID),
			ReportSessionPolicy:          strings.TrimSpace(payload.ReportSessionPolicy),
			ReportSessionPolicySelection: strings.TrimSpace(payload.ReportSessionPolicySelection),
			SessionChainKind:             strings.TrimSpace(payload.SessionChainKind),
			ReportMode:                   strings.TrimSpace(payload.ReportMode),
			ReportPendingEventID:         strings.TrimSpace(payload.PendingEventID),
		}
		info.ReportSessionID = firstNonEmpty(info.ReportSessionID, info.AgentSessionID, info.PreviousAgentSessionID)
		return info, nil
	}
	return reportArtifactSessionInfo{}, fmt.Errorf("%w: report artifact event not found", app.ErrInvalidInput)
}

func selectReportPatchSession(ctx context.Context, executor AgentExecutor, sourceSessionID string, requestedPolicy string) (reportPatchSessionSelection, error) {
	return SelectReportPatchSession(ctx, executor, sourceSessionID, requestedPolicy)
}

func SelectReportPatchSession(ctx context.Context, executor AgentExecutor, sourceSessionID string, requestedPolicy string) (ReportPatchSessionSelection, error) {
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	if sourceSessionID == "" {
		return ReportPatchSessionSelection{}, fmt.Errorf("%w: report patch requires a previous report session", app.ErrInvalidInput)
	}
	requestedPolicy = strings.TrimSpace(requestedPolicy)
	if requestedPolicy != "" {
		policy, err := normalizeReportSessionPolicy(requestedPolicy)
		if err != nil {
			return ReportPatchSessionSelection{}, err
		}
		if policy == reportSessionPolicySameSession {
			return ReportPatchSessionSelection{
				SessionID:                    sourceSessionID,
				PreviousAgentSessionID:       sourceSessionID,
				ReportSessionPolicy:          reportSessionPolicySameSession,
				ReportSessionPolicySelection: reporting.SessionPolicySelectionExplicitSameSession,
				SessionChainKind:             "same_report_session_patch",
			}, nil
		}
		forker, ok := executor.(AgentSessionForker)
		if !ok {
			return ReportPatchSessionSelection{}, fmt.Errorf("%w: isolated report patch session requires a forkable executor", app.ErrInvalidInput)
		}
		if !AgentSessionForkReady(ctx, executor, sourceSessionID) {
			return ReportPatchSessionSelection{}, fmt.Errorf("%w: isolated report patch session is not ready for fork", app.ErrInvalidInput)
		}
		fork, err := forker.ForkSession(ctx, sourceSessionID)
		if err != nil {
			return ReportPatchSessionSelection{}, fmt.Errorf("report patch session fork failed: %w", err)
		}
		forkSource := firstNonEmpty(fork.SourceSessionID, sourceSessionID)
		return ReportPatchSessionSelection{
			SessionID:                    fork.SessionID,
			PreviousAgentSessionID:       fork.SessionID,
			ForkSourceAgentSessionID:     forkSource,
			ReportSessionPolicy:          reportSessionPolicyIsolatedFork,
			ReportSessionPolicySelection: reporting.SessionPolicySelectionExplicitIsolatedFork,
			SessionChainKind:             "isolated_fork_report_patch",
		}, nil
	}

	forker, canFork := executor.(AgentSessionForker)
	if !canFork {
		return ReportPatchSessionSelection{
			SessionID:                    sourceSessionID,
			PreviousAgentSessionID:       sourceSessionID,
			ReportSessionPolicy:          reportSessionPolicySameSession,
			ReportSessionPolicySelection: reporting.SessionPolicySelectionAutoSameSessionNoForker,
			SessionChainKind:             "same_report_session_patch",
		}, nil
	}
	if !AgentSessionForkReady(ctx, executor, sourceSessionID) {
		return ReportPatchSessionSelection{
			SessionID:                    sourceSessionID,
			PreviousAgentSessionID:       sourceSessionID,
			ReportSessionPolicy:          reportSessionPolicySameSession,
			ReportSessionPolicySelection: reporting.SessionPolicySelectionAutoSameSessionForkFailed,
			SessionChainKind:             "same_report_session_patch",
		}, nil
	}
	fork, err := forker.ForkSession(ctx, sourceSessionID)
	if err != nil {
		return ReportPatchSessionSelection{
			SessionID:                    sourceSessionID,
			PreviousAgentSessionID:       sourceSessionID,
			ReportSessionPolicy:          reportSessionPolicySameSession,
			ReportSessionPolicySelection: reporting.SessionPolicySelectionAutoSameSessionForkFailed,
			SessionChainKind:             "same_report_session_patch",
		}, nil
	}
	forkSource := firstNonEmpty(fork.SourceSessionID, sourceSessionID)
	return ReportPatchSessionSelection{
		SessionID:                    fork.SessionID,
		PreviousAgentSessionID:       fork.SessionID,
		ForkSourceAgentSessionID:     forkSource,
		ReportSessionPolicy:          reportSessionPolicyIsolatedFork,
		ReportSessionPolicySelection: reporting.SessionPolicySelectionAutoIsolatedFork,
		SessionChainKind:             "isolated_fork_report_patch",
	}, nil
}

func (server *Server) exportMarkdownArtifactAsHTML(ctx context.Context, missionID string, sourceArtifact app.RawArtifact) (app.ReportExportResult, error) {
	if !isMarkdownMediaType(sourceArtifact.MediaType) {
		return app.ReportExportResult{}, fmt.Errorf("%w: HTML export requires a Markdown report artifact", app.ErrInvalidInput)
	}
	if cached, ok, err := server.existingMarkdownArtifactHTMLExport(ctx, missionID, sourceArtifact.ArtifactID); err != nil {
		return app.ReportExportResult{}, err
	} else if ok {
		return cached, nil
	}
	content, err := server.renderSelfContainedReportHTML(ctx, missionID, sourceArtifact)
	if err != nil {
		return app.ReportExportResult{}, err
	}
	artifact, err := server.service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: newID("art"),
		MissionID:  missionID,
		MediaType:  "text/html; charset=utf-8",
		Filename:   markdownReportHTMLFilename(sourceArtifact),
		Producer:   app.Producer{Type: "plasma", ID: "html-export"},
		Content:    content,
	})
	if err != nil {
		return app.ReportExportResult{}, err
	}
	event, err := server.service.AppendEvent(ctx, reporting.BuildSelfContainedHTMLExportAppendRequest(reporting.SelfContainedHTMLExportEventRequest{
		EventID:          newID("evt"),
		MissionID:        missionID,
		SourceArtifactID: sourceArtifact.ArtifactID,
		Artifact:         artifact,
		Producer:         app.Producer{Type: "plasma", ID: "html-export"},
	}))
	if err != nil {
		return app.ReportExportResult{}, err
	}
	return app.ReportExportResult{Artifact: artifact, Event: event}, nil
}

func (server *Server) existingMarkdownArtifactHTMLExport(ctx context.Context, missionID string, sourceArtifactID string) (app.ReportExportResult, bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return app.ReportExportResult{}, false, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "report.artifact.exported" {
			continue
		}
		var payload struct {
			Kind             string `json:"kind"`
			SourceArtifactID string `json:"source_artifact_id"`
			ArtifactID       string `json:"artifact_id"`
			Target           string `json:"target"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.Kind) != reporting.ExportKindSelfContainedHTML ||
			strings.TrimSpace(payload.SourceArtifactID) != sourceArtifactID ||
			strings.TrimSpace(payload.Target) != reporting.ExportTargetSelfContainedHTML {
			continue
		}
		artifactID := strings.TrimSpace(payload.ArtifactID)
		if artifactID == "" {
			continue
		}
		artifact, err := server.service.GetRawArtifact(ctx, artifactID)
		if err != nil {
			return app.ReportExportResult{}, false, err
		}
		return app.ReportExportResult{Artifact: artifact, Event: event}, true, nil
	}
	return app.ReportExportResult{}, false, nil
}

func (server *Server) startDesignedReportHTMLExport(ctx context.Context, missionID string, sourceArtifact app.RawArtifact, req reportDesignRequest) (map[string]any, bool, error) {
	if !isMarkdownMediaType(sourceArtifact.MediaType) {
		return nil, false, fmt.Errorf("%w: designed HTML export requires a Markdown report artifact", app.ErrInvalidInput)
	}
	images, notes, err := server.inlineReportImages(ctx, missionID)
	if err != nil {
		return nil, false, err
	}
	imageSetFingerprint := designedReportImageSetFingerprint(images, notes)
	if cached, ok, err := server.existingDesignedReportHTMLExport(ctx, missionID, sourceArtifact.ArtifactID, imageSetFingerprint); err != nil {
		return nil, false, err
	} else if ok {
		return map[string]any{
			"status":          "completed",
			"artifact":        rawArtifactMetadata(cached.Artifact),
			"source_artifact": rawArtifactMetadata(sourceArtifact),
			"event":           cached.Event,
			"content":         string(cached.Artifact.Content),
		}, false, nil
	}
	executorName, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		return nil, false, err
	}
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return nil, false, fmt.Errorf("%w: designed HTML export requires an agent executor", app.ErrInvalidInput)
	}
	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	unlockTurns := server.turns.lock(missionID)
	defer unlockTurns()
	if err := server.validateMissionAgentExecutor(ctx, missionID, executorName); err != nil {
		return nil, false, err
	}
	if err := server.reconcileStaleAgentTurn(ctx, missionID); err != nil {
		return nil, false, err
	}
	if err := server.reconcileStaleReportDrafts(ctx, missionID); err != nil {
		return nil, false, err
	}
	if err := server.reconcileStaleDesignedReportExports(ctx, missionID); err != nil {
		return nil, false, err
	}
	if server.hasOpenReportDraft(ctx, missionID) {
		return nil, false, errReportDraftRunning
	}
	if server.hasOpenAgentTurn(ctx, missionID) {
		return nil, false, fmt.Errorf("%w: agent turn is already running for this mission", app.ErrInvalidInput)
	}
	if active := server.activeWorkflowRun(ctx, missionID); active != nil {
		return nil, false, fmt.Errorf("%w: workflow %s is %s for this mission", app.ErrInvalidInput, active.WorkflowRunID, active.Status)
	}
	agentModel := server.latestAgentSessionModel(ctx, missionID, executorName)
	agentReasoningEffort := server.latestAgentReasoningEffort(ctx, missionID, executorName)
	pendingEvent, err := server.reportRunner().StartDesign(ctx, missionID, reporting.DesignRequest{
		SourceArtifactID:     sourceArtifact.ArtifactID,
		SourceMediaType:      sourceArtifact.MediaType,
		Title:                reportArtifactTitle(sourceArtifact),
		AgentExecutor:        executorName,
		AgentModel:           agentModel,
		AgentReasoningEffort: agentReasoningEffort,
		RendererVersion:      designedReportRendererVersion,
	}, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		return nil, false, err
	}
	return map[string]any{
		"pending_event":   pendingEvent,
		"source_artifact": rawArtifactMetadata(sourceArtifact),
		"status":          "pending",
	}, true, nil
}

func (server *Server) createDesignedReportHTMLExport(ctx context.Context, missionID string, sourceArtifactID string, req reportDesignRequest, pendingEventID string) (app.ReportExportResult, error) {
	sourceArtifact, err := server.reportArtifact(ctx, missionID, sourceArtifactID)
	if err != nil {
		return app.ReportExportResult{}, err
	}
	if !isMarkdownMediaType(sourceArtifact.MediaType) {
		return app.ReportExportResult{}, fmt.Errorf("%w: designed HTML export requires a Markdown report artifact", app.ErrInvalidInput)
	}
	images, notes, err := server.inlineReportImages(ctx, missionID)
	if err != nil {
		return app.ReportExportResult{}, err
	}
	imageSetFingerprint := designedReportImageSetFingerprint(images, notes)
	if cached, ok, err := server.existingDesignedReportHTMLExport(ctx, missionID, sourceArtifact.ArtifactID, imageSetFingerprint); err != nil {
		return app.ReportExportResult{}, err
	} else if ok {
		return cached, nil
	}
	executorName, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		return app.ReportExportResult{}, err
	}
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return app.ReportExportResult{}, fmt.Errorf("%w: designed HTML export requires an agent executor", app.ErrInvalidInput)
	}
	agentModel := strings.TrimSpace(req.AgentModel)
	agentReasoningEffort := strings.TrimSpace(req.AgentReasoningEffort)
	agentModel, agentReasoningEffort, err = resolveAgentSettings(executorName, agentModel, agentReasoningEffort, "")
	if err != nil {
		return app.ReportExportResult{}, err
	}
	title := reportArtifactTitle(sourceArtifact)
	toolSessionID := newID("ses")
	started := time.Now()
	result, err := executor.Run(ctx, AgentRequest{
		UserText:        "generate designed HTML content model",
		Prompt:          agentDesignedHTMLContentModelPrompt(title, string(sourceArtifact.Content), images),
		Model:           agentModel,
		ReasoningEffort: agentReasoningEffort,
		MissionID:       missionID,
		ToolSessionID:   toolSessionID,
		AgentExecutor:   executorName,
		MCPMode:         "auto",
	})
	agentDurationMS := time.Since(started).Milliseconds()
	if err != nil {
		return app.ReportExportResult{}, fmt.Errorf("designed HTML content model agent failed: %w", reportAgentFailure(err, result, "report_design", agentDurationMS, ""))
	}
	model, modelJSON, err := parseDesignedReportContentModel(result.Text)
	if err != nil {
		return app.ReportExportResult{}, reportAgentFailure(err, result, "report_design", agentDurationMS, "")
	}
	modelArtifact, err := server.service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: newID("art"),
		MissionID:  missionID,
		MediaType:  "application/json; charset=utf-8",
		Filename:   safeFilename(title+" content model", ".json"),
		Producer:   app.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)},
		Content:    modelJSON,
	})
	if err != nil {
		return app.ReportExportResult{}, err
	}
	content, err := server.renderDesignedReportHTML(sourceArtifact, model, images, notes)
	if err != nil {
		return app.ReportExportResult{}, err
	}
	artifact, err := server.service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: newID("art"),
		MissionID:  missionID,
		MediaType:  "text/html; charset=utf-8",
		Filename:   safeFilename(title+" designed", ".html"),
		Producer:   app.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)},
		Content:    content,
	})
	if err != nil {
		return app.ReportExportResult{}, err
	}
	event, err := server.service.AppendEvent(ctx, reporting.BuildDesignedHTMLExportAppendRequest(reporting.DesignedHTMLExportEventRequest{
		EventID:                newID("evt"),
		MissionID:              missionID,
		PendingEventID:         pendingEventID,
		SourceArtifactID:       sourceArtifact.ArtifactID,
		ContentModelArtifactID: modelArtifact.ArtifactID,
		Artifact:               artifact,
		RendererVersion:        designedReportRendererVersion,
		ImageSetFingerprint:    imageSetFingerprint,
		AgentExecutor:          executorName,
		AgentModel:             agentModel,
		AgentReasoningEffort:   agentReasoningEffort,
		AgentSessionID:         result.SessionID,
		ToolSessionID:          toolSessionID,
		DurationMS:             time.Since(started).Milliseconds(),
		AgentDurationMS:        agentDurationMS,
		AgentUsage:             result.Usage,
		AgentResumed:           result.Resumed,
		Producer:               app.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)},
	}))
	if err != nil {
		return app.ReportExportResult{}, err
	}
	return app.ReportExportResult{Artifact: artifact, Event: event}, nil
}

func (server *Server) reconcileStaleDesignedReportExports(ctx context.Context, missionID string) error {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return err
	}
	completed := reporting.CompletedPendingEventIDs(events)
	for _, event := range events {
		if event.EventType != "report.design.pending" {
			continue
		}
		if _, ok := completed[event.EventID]; ok {
			continue
		}
		if server.runningReports.Owns(missionID, event.EventID) {
			continue
		}
		if err := server.reportRunner().ResumeDesign(ctx, missionID, event); err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (server *Server) existingDesignedReportHTMLExport(ctx context.Context, missionID string, sourceArtifactID string, imageSetFingerprint string) (app.ReportExportResult, bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return app.ReportExportResult{}, false, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "report.artifact.exported" {
			continue
		}
		var payload struct {
			Kind             string `json:"kind"`
			SourceArtifactID string `json:"source_artifact_id"`
			ArtifactID       string `json:"artifact_id"`
			Target           string `json:"target"`
			RendererVersion  string `json:"renderer_version"`
			ImageSet         string `json:"image_set_fingerprint"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.Kind) != reporting.ExportKindDesignedHTML ||
			strings.TrimSpace(payload.SourceArtifactID) != sourceArtifactID ||
			strings.TrimSpace(payload.Target) != reporting.ExportTargetDesignedHTML ||
			strings.TrimSpace(payload.RendererVersion) != designedReportRendererVersion ||
			strings.TrimSpace(payload.ImageSet) != imageSetFingerprint {
			continue
		}
		artifactID := strings.TrimSpace(payload.ArtifactID)
		if artifactID == "" {
			continue
		}
		artifact, err := server.service.GetRawArtifact(ctx, artifactID)
		if err != nil {
			return app.ReportExportResult{}, false, err
		}
		return app.ReportExportResult{Artifact: artifact, Event: event}, true, nil
	}
	return app.ReportExportResult{}, false, nil
}

func (server *Server) renderSelfContainedReportHTML(ctx context.Context, missionID string, sourceArtifact app.RawArtifact) ([]byte, error) {
	var rendered bytes.Buffer
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	if err := md.Convert(sourceArtifact.Content, &rendered); err != nil {
		return nil, err
	}
	images, notes, err := server.inlineReportImages(ctx, missionID)
	if err != nil {
		return nil, err
	}
	title := reportArtifactTitle(sourceArtifact)
	wordCount := len(strings.Fields(string(sourceArtifact.Content)))
	var out bytes.Buffer
	out.WriteString("<!doctype html>\n<html lang=\"ko\">\n<head>\n<meta charset=\"utf-8\">\n")
	out.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	out.WriteString("<title>" + htmlpkg.EscapeString(title) + "</title>\n")
	out.WriteString(selfContainedReportCSS())
	out.WriteString("</head>\n<body>\n")
	out.WriteString("<header class=\"hero\"><div><p class=\"eyebrow\">Plasma Report</p><h1>" + htmlpkg.EscapeString(title) + "</h1><p class=\"sub\">Markdown report artifact에서 파생한 self-contained interactive HTML입니다. 이미지는 가능한 경우 원본 source artifact를 data URI로 포함했습니다.</p></div><button id=\"themeToggle\" type=\"button\">테마 전환</button></header>\n")
	out.WriteString("<main class=\"layout\">\n")
	out.WriteString("<aside class=\"rail\"><div class=\"metric\"><span>본문 단어</span><strong>" + strconv.Itoa(wordCount) + "</strong></div><div class=\"metric\"><span>포함 이미지</span><strong>" + strconv.Itoa(len(images)) + "</strong></div><div class=\"metric\"><span>원본 artifact</span><code>" + htmlpkg.EscapeString(sourceArtifact.ArtifactID) + "</code></div><nav><a href=\"#report-body\">본문</a><a href=\"#media-gallery\">미디어</a><a href=\"#export-notes\">생성 노트</a></nav></aside>\n")
	out.WriteString("<article id=\"report-body\" class=\"report-body\">\n")
	out.Write(rendered.Bytes())
	out.WriteString("</article>\n")
	out.WriteString("<section id=\"media-gallery\" class=\"media-panel\"><div class=\"section-head\"><h2>미디어</h2><span>" + strconv.Itoa(len(images)) + "개 이미지 포함</span></div>")
	if len(images) == 0 {
		out.WriteString("<p class=\"muted\">이 미션의 active image source 중 self-contained HTML에 포함할 수 있는 이미지가 없습니다.</p>")
	} else {
		out.WriteString("<div class=\"gallery\">")
		for _, image := range images {
			out.WriteString("<figure><img loading=\"lazy\" src=\"" + image.DataURI + "\" alt=\"" + htmlpkg.EscapeString(image.Title) + "\"><figcaption><strong>" + htmlpkg.EscapeString(image.Title) + "</strong><span>" + htmlpkg.EscapeString(image.Caption()) + "</span></figcaption></figure>")
		}
		out.WriteString("</div>")
	}
	out.WriteString("</section>\n")
	out.WriteString("<section id=\"export-notes\" class=\"notes\"><h2>생성 노트</h2><ul>")
	out.WriteString("<li>이 HTML은 보고서 내용을 다시 생성하지 않고 저장된 Markdown artifact를 렌더링했습니다.</li>")
	out.WriteString("<li>오디오와 영상은 self-contained로 포함하지 않습니다.</li>")
	for _, note := range notes {
		out.WriteString("<li>" + htmlpkg.EscapeString(note) + "</li>")
	}
	out.WriteString("</ul></section>\n")
	out.WriteString("</main>\n")
	out.WriteString("<script>const b=document.body,t=document.getElementById('themeToggle');t?.addEventListener('click',()=>b.classList.toggle('light'));document.querySelectorAll('.report-body h2,.report-body h3').forEach(h=>{h.tabIndex=0;h.addEventListener('click',()=>h.classList.toggle('marked'))});</script>\n")
	out.WriteString("</body>\n</html>\n")
	return out.Bytes(), nil
}

func isMarkdownMediaType(mediaType string) bool {
	base, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		base = mediaType
	}
	base = strings.ToLower(strings.TrimSpace(base))
	return base == "text/markdown" || base == "text/x-markdown"
}

func isImageMediaType(mediaType string) bool {
	return mediaKindForType(mediaType) == app.MediaKindImage
}

func reportArtifactTitle(artifact app.RawArtifact) string {
	filename := strings.TrimSpace(artifact.Filename)
	if filename != "" {
		base := strings.TrimSuffix(filename, filepath.Ext(filename))
		if strings.TrimSpace(base) != "" {
			return base
		}
		return filename
	}
	return artifact.ArtifactID
}

func markdownReportHTMLFilename(artifact app.RawArtifact) string {
	title := reportArtifactTitle(artifact)
	return safeFilename(title, ".html")
}

func selfContainedReportCSS() string {
	return `<style>
:root{color-scheme:dark;--bg:#101214;--panel:#191d20;--text:#f2efe8;--muted:#a9b0aa;--line:#333a3f;--accent:#e3b04b;--accent2:#7cc6b2}
body{margin:0;background:var(--bg);color:var(--text);font:15px/1.68 Georgia,"Noto Serif KR",serif;letter-spacing:0}
body.light{--bg:#f6f3ec;--panel:#fffdf8;--text:#202326;--muted:#5d665f;--line:#ddd5c6;--accent:#9a5b16;--accent2:#276f61}
.hero{display:flex;justify-content:space-between;gap:24px;align-items:flex-end;padding:42px clamp(20px,5vw,72px) 28px;border-bottom:1px solid var(--line);background:linear-gradient(120deg,rgba(227,176,75,.16),transparent 45%),var(--panel)}
.eyebrow{margin:0 0 8px;color:var(--accent);font:700 12px/1.2 ui-monospace,monospace;text-transform:uppercase}
h1{margin:0;font-size:clamp(28px,5vw,58px);line-height:1.08;max-width:980px}
.sub{max-width:780px;color:var(--muted);margin:14px 0 0}
button{border:1px solid var(--line);background:transparent;color:var(--text);border-radius:6px;padding:9px 12px;cursor:pointer}
.layout{display:grid;grid-template-columns:minmax(180px,260px) minmax(0,1fr);gap:28px;max-width:1320px;margin:0 auto;padding:30px clamp(16px,4vw,56px) 80px}
.rail{position:sticky;top:0;align-self:start;display:grid;gap:12px}
.metric,.media-panel,.notes,.report-body{background:var(--panel);border:1px solid var(--line);border-radius:8px}
.metric{padding:14px}.metric span{display:block;color:var(--muted);font-size:12px}.metric strong{font-size:30px;color:var(--accent)}.metric code{word-break:break-all}
nav{display:grid;gap:8px;margin-top:8px}nav a{color:var(--accent2);text-decoration:none}
.report-body{padding:clamp(22px,4vw,54px);min-width:0}.report-body h1:first-child{font-size:clamp(30px,5vw,54px)}
.report-body h2{margin-top:38px;border-top:1px solid var(--line);padding-top:22px}.report-body h2.marked,.report-body h3.marked{color:var(--accent)}
.report-body p,.report-body li{font-size:17px}.report-body a{color:var(--accent2)}.report-body img{max-width:100%;height:auto;border-radius:8px}
pre,code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}pre{overflow:auto;padding:14px;background:rgba(0,0,0,.22);border-radius:8px}
.media-panel,.notes{grid-column:2;padding:24px}.section-head{display:flex;justify-content:space-between;gap:12px;align-items:center}.muted,.notes{color:var(--muted)}
.gallery{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:16px}.gallery figure{margin:0;border:1px solid var(--line);border-radius:8px;overflow:hidden;background:rgba(255,255,255,.03)}.gallery img{display:block;width:100%;height:auto}.gallery figcaption{display:grid;gap:4px;padding:10px}.gallery figcaption span{font-size:12px;color:var(--muted);word-break:break-word}
@media(max-width:820px){.hero{display:block}.layout{display:block}.rail{position:static;margin-bottom:18px}.media-panel,.notes{margin-top:18px}.report-body{padding:20px}.report-body p,.report-body li{font-size:15px}}
</style>
`
}

func (server *Server) isReportArtifact(ctx context.Context, missionID string, artifactID string) (bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return false, err
	}
	for _, event := range events {
		if event.EventType != "report.artifact.created" && event.EventType != "report.artifact.exported" {
			continue
		}
		var payload struct {
			ArtifactID string `json:"artifact_id"`
			Kind       string `json:"kind"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		kind := strings.TrimSpace(payload.Kind)
		if strings.TrimSpace(payload.ArtifactID) == artifactID && (kind == "markdown_report_artifact" || kind == reporting.ExportKindSelfContainedHTML || kind == reporting.ExportKindDesignedHTML || kind == reporting.ExportKindHumanizedMarkdown) {
			return true, nil
		}
	}
	return false, nil
}

func (server *Server) handleReportVersionRoute(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/report_versions/"), "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	switch parts[1] {
	case "ast":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		ast, err := server.service.ReportAST(r.Context(), parts[0])
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, ast)
	case "export":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		server.exportReportVersion(w, r, parts[0])
	default:
		http.NotFound(w, r)
	}
}

func (server *Server) hasOpenReportDraft(ctx context.Context, missionID string) bool {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return false
	}
	return hasOpenReportDraftPending(events)
}

func (server *Server) hasReportDraftTerminalEvent(ctx context.Context, missionID string, pendingEventID string) bool {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return false
	}
	_, ok := reporting.CompletedPendingEventIDs(events)[strings.TrimSpace(pendingEventID)]
	return ok
}

type openAgentPending = conversation.OpenAgentPending

func (server *Server) startReportDraft(ctx context.Context, missionID string, req reportDraftRequest) (map[string]any, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Mission report"
	}
	executorName, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		return nil, err
	}
	mcpMode, err := normalizeMCPMode(req.MCPMode)
	if err != nil {
		return nil, err
	}
	rigor, err := normalizeReportRigorProfile(req.RigorLevel)
	if err != nil {
		return nil, err
	}
	reportMode, err := normalizeReportMode(req.ReportMode)
	if err != nil {
		return nil, err
	}
	req.Title = title
	req.AgentExecutor = executorName
	req.MCPMode = mcpMode
	req.RigorLevel = rigor.level
	req.ReportMode = reportMode
	guidanceProfile, guidanceSHA, err := SelectReportGenerationGuidance(req.GenerationGuidanceProfile)
	if err != nil {
		return nil, err
	}
	postReportHumanize := normalizePostReportHumanize(req.PostReportHumanize)

	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	unlockTurns := server.turns.lock(missionID)
	defer unlockTurns()
	if err := server.validateMissionAgentExecutor(ctx, missionID, executorName); err != nil {
		return nil, err
	}
	if err := server.reconcileStaleAgentTurn(ctx, missionID); err != nil {
		return nil, err
	}
	if err := server.reconcileStaleReportDrafts(ctx, missionID); err != nil {
		return nil, err
	}
	if server.hasOpenReportDraft(ctx, missionID) {
		return nil, errReportDraftRunning
	}
	if server.hasOpenAgentTurn(ctx, missionID) {
		return nil, fmt.Errorf("%w: agent turn is already running for this mission", app.ErrInvalidInput)
	}
	if active := server.activeWorkflowRun(ctx, missionID); active != nil {
		return nil, fmt.Errorf("%w: workflow %s is %s for this mission", app.ErrInvalidInput, active.WorkflowRunID, active.Status)
	}
	// Pending state keeps the mission's raw selection. The worker resolves it
	// when constructing agent requests, after the report start is durable.
	req.AgentModel = server.latestAgentSessionModel(ctx, missionID, executorName)
	req.AgentReasoningEffort = server.latestAgentReasoningEffort(ctx, missionID, executorName)
	executor := server.agentExecutor(executorName)
	reportSessionPolicy, reportSessionPolicySelection, err := server.selectReportSessionPolicy(ctx, missionID, executorName, reportMode, strings.TrimSpace(req.ReportSessionPolicy), executor)
	if err != nil {
		return nil, err
	}
	req.ReportSessionPolicy = reportSessionPolicy
	req.ReportSessionPolicySelection = reportSessionPolicySelection
	pendingEvent, err := server.reportRunner().StartDraft(ctx, missionID, reporting.DraftRequest{
		Title:                        title,
		AgentExecutor:                executorName,
		AgentModel:                   req.AgentModel,
		AgentReasoningEffort:         req.AgentReasoningEffort,
		MCPMode:                      mcpMode,
		RigorLevel:                   rigor.level,
		RigorLabel:                   rigor.label,
		ReportMode:                   reportMode,
		ReportSessionPolicy:          reportSessionPolicy,
		ReportSessionPolicySelection: reportSessionPolicySelection,
		PostReportHumanize:           postReportHumanize,
		GenerationGuidanceProfile:    guidanceProfile,
		GenerationGuidanceSHA256:     guidanceSHA,
	}, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"pending_event": pendingEvent,
		"status":        "pending",
	}, nil
}

func (server *Server) startReportPatch(ctx context.Context, missionID string, req reportPatchRequest) (map[string]any, error) {
	baseArtifactID := strings.TrimSpace(req.BaseArtifactID)
	if baseArtifactID == "" {
		return nil, fmt.Errorf("%w: base report artifact is required", app.ErrInvalidInput)
	}
	instruction := strings.TrimSpace(req.Instruction)
	if instruction == "" {
		return nil, fmt.Errorf("%w: report patch instruction is required", app.ErrInvalidInput)
	}
	baseArtifact, err := server.reportArtifact(ctx, missionID, baseArtifactID)
	if err != nil {
		return nil, err
	}
	if !isMarkdownMediaType(baseArtifact.MediaType) {
		return nil, fmt.Errorf("%w: report patch requires a Markdown report artifact", app.ErrInvalidInput)
	}
	info, err := server.reportArtifactSessionInfo(ctx, missionID, baseArtifactID)
	if err != nil {
		return nil, err
	}
	executorName := strings.TrimSpace(req.AgentExecutor)
	if executorName == "" {
		executorName = info.AgentExecutor
	}
	executorName, err = normalizeAgentExecutorName(executorName)
	if err != nil {
		return nil, err
	}
	if baseExecutor := strings.TrimSpace(info.AgentExecutor); baseExecutor != "" && baseExecutor != executorName {
		return nil, fmt.Errorf("%w: report patch must use the original report executor %q", app.ErrInvalidInput, baseExecutor)
	}
	mcpMode, err := normalizeMCPMode(req.MCPMode)
	if err != nil {
		return nil, err
	}
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return nil, fmt.Errorf("%w: report patch requires an agent executor", app.ErrInvalidInput)
	}
	agentModel := strings.TrimSpace(req.AgentModel)
	if agentModel == "" {
		agentModel = strings.TrimSpace(info.AgentModel)
	}
	agentReasoningEffort := strings.TrimSpace(req.AgentReasoningEffort)
	if agentReasoningEffort == "" {
		agentReasoningEffort = strings.TrimSpace(info.AgentReasoningEffort)
	}
	agentModel, agentReasoningEffort, err = resolveAgentSettings(executorName, agentModel, agentReasoningEffort, strings.TrimSpace(info.ReportSessionID))
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = firstNonEmpty(info.Title+" 수정본", reportArtifactTitle(baseArtifact)+" 수정본", "Patched report")
	}

	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	unlockTurns := server.turns.lock(missionID)
	defer unlockTurns()
	if err := server.validateMissionAgentExecutor(ctx, missionID, executorName); err != nil {
		return nil, err
	}
	if err := server.reconcileStaleAgentTurn(ctx, missionID); err != nil {
		return nil, err
	}
	if err := server.reconcileStaleReportDrafts(ctx, missionID); err != nil {
		return nil, err
	}
	if server.hasOpenReportDraft(ctx, missionID) {
		return nil, errReportDraftRunning
	}
	if server.hasOpenAgentTurn(ctx, missionID) {
		return nil, fmt.Errorf("%w: agent turn is already running for this mission", app.ErrInvalidInput)
	}
	if active := server.activeWorkflowRun(ctx, missionID); active != nil {
		return nil, fmt.Errorf("%w: workflow %s is %s for this mission", app.ErrInvalidInput, active.WorkflowRunID, active.Status)
	}
	selection, err := selectReportPatchSession(ctx, executor, info.ReportSessionID, req.ReportSessionPolicy)
	if err != nil {
		return nil, err
	}
	pendingEvent, err := server.reportRunner().StartPatch(ctx, missionID, reporting.PatchRequest{
		BaseArtifactID:               baseArtifact.ArtifactID,
		Instruction:                  instruction,
		Title:                        title,
		AgentExecutor:                executorName,
		AgentModel:                   agentModel,
		AgentReasoningEffort:         agentReasoningEffort,
		MCPMode:                      mcpMode,
		ReportSessionID:              selection.SessionID,
		PreviousAgentSessionID:       selection.PreviousAgentSessionID,
		ForkSourceAgentSessionID:     selection.ForkSourceAgentSessionID,
		ReportSessionPolicy:          selection.ReportSessionPolicy,
		ReportSessionPolicySelection: selection.ReportSessionPolicySelection,
		SessionChainKind:             selection.SessionChainKind,
	}, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"pending_event": pendingEvent,
		"status":        "pending",
	}, nil
}

func (server *Server) startReportHumanize(ctx context.Context, missionID string, sourceArtifact app.RawArtifact, req reportHumanizeRequest) (map[string]any, error) {
	if !isMarkdownMediaType(sourceArtifact.MediaType) {
		return nil, fmt.Errorf("%w: H5 humanize requires a Markdown report artifact", app.ErrInvalidInput)
	}
	info, err := server.reportArtifactSessionInfo(ctx, missionID, sourceArtifact.ArtifactID)
	if err != nil {
		return nil, err
	}
	executorName := strings.TrimSpace(req.AgentExecutor)
	if executorName == "" {
		executorName = info.AgentExecutor
	}
	executorName, err = normalizeAgentExecutorName(executorName)
	if err != nil {
		return nil, err
	}
	if baseExecutor := strings.TrimSpace(info.AgentExecutor); baseExecutor != "" && baseExecutor != executorName {
		return nil, fmt.Errorf("%w: H5 humanize must use the original report executor %q", app.ErrInvalidInput, baseExecutor)
	}
	mcpMode, err := normalizeMCPMode(req.MCPMode)
	if err != nil {
		return nil, err
	}
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return nil, fmt.Errorf("%w: H5 humanize requires an agent executor", app.ErrInvalidInput)
	}
	reportSessionID := strings.TrimSpace(info.ReportSessionID)
	if reportSessionID == "" {
		return nil, fmt.Errorf("%w: H5 humanize requires a report session", app.ErrInvalidInput)
	}
	agentModel := strings.TrimSpace(req.AgentModel)
	if agentModel == "" {
		agentModel = firstNonEmpty(info.AgentModel, server.latestAgentSessionModel(ctx, missionID, executorName))
	}
	agentReasoningEffort := strings.TrimSpace(req.AgentReasoningEffort)
	if agentReasoningEffort == "" {
		agentReasoningEffort = firstNonEmpty(info.AgentReasoningEffort, server.latestAgentReasoningEffort(ctx, missionID, executorName))
	}
	agentModel, agentReasoningEffort, err = resolveAgentSettings(executorName, agentModel, agentReasoningEffort, reportSessionID)
	if err != nil {
		return nil, err
	}
	title := firstNonEmpty(req.Title, info.Title, reportArtifactTitle(sourceArtifact))
	reportMode := firstNonEmpty(info.ReportMode, defaultReportMode)

	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	unlockTurns := server.turns.lock(missionID)
	defer unlockTurns()
	if err := server.validateMissionAgentExecutor(ctx, missionID, executorName); err != nil {
		return nil, err
	}
	if err := server.reconcileStaleAgentTurn(ctx, missionID); err != nil {
		return nil, err
	}
	if err := server.reconcileStaleReportDrafts(ctx, missionID); err != nil {
		return nil, err
	}
	if server.hasOpenReportDraft(ctx, missionID) {
		return nil, errReportDraftRunning
	}
	if server.hasOpenAgentTurn(ctx, missionID) {
		return nil, fmt.Errorf("%w: agent turn is already running for this mission", app.ErrInvalidInput)
	}
	if active := server.activeWorkflowRun(ctx, missionID); active != nil {
		return nil, fmt.Errorf("%w: workflow %s is %s for this mission", app.ErrInvalidInput, active.WorkflowRunID, active.Status)
	}
	pendingEvent, err := server.reportRunner().StartHumanize(ctx, missionID, reporting.HumanizeRequest{
		SourceArtifactID:       sourceArtifact.ArtifactID,
		SourceArtifactSHA256:   sourceArtifact.SHA256,
		SourceMediaType:        sourceArtifact.MediaType,
		Title:                  title,
		AgentExecutor:          executorName,
		AgentModel:             agentModel,
		AgentReasoningEffort:   agentReasoningEffort,
		MCPMode:                mcpMode,
		PreviousAgentSessionID: reportSessionID,
		ReportMode:             reportMode,
		ReportPendingEventID:   info.ReportPendingEventID,
	}, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"pending_event":   pendingEvent,
		"source_artifact": rawArtifactMetadata(sourceArtifact),
		"status":          "pending",
	}, nil
}

func (server *Server) reportRunner() reporting.Runner {
	return reporting.Runner{
		Service:  server.service,
		InFlight: &server.runningReports,
		NewID:    newID,
		GenerateDraft: func(ctx context.Context, missionID string, req reporting.DraftRequest, pendingEventID string) error {
			_, err := server.createReportDraft(ctx, missionID, reportDraftRequest{
				Title:                        req.Title,
				AgentExecutor:                req.AgentExecutor,
				AgentModel:                   req.AgentModel,
				AgentReasoningEffort:         req.AgentReasoningEffort,
				MCPMode:                      req.MCPMode,
				RigorLevel:                   req.RigorLevel,
				ReportMode:                   req.ReportMode,
				ReportSessionPolicy:          req.ReportSessionPolicy,
				ReportSessionPolicySelection: req.ReportSessionPolicySelection,
				PostReportHumanize:           req.PostReportHumanize,
				GenerationGuidanceProfile:    req.GenerationGuidanceProfile,
				GenerationGuidanceSHA256:     req.GenerationGuidanceSHA256,
			}, pendingEventID)
			return err
		},
		GenerateDesign: func(ctx context.Context, missionID string, req reporting.DesignRequest, pendingEventID string) error {
			_, err := server.createDesignedReportHTMLExport(ctx, missionID, req.SourceArtifactID, reportDesignRequest{
				AgentExecutor:        req.AgentExecutor,
				AgentModel:           req.AgentModel,
				AgentReasoningEffort: req.AgentReasoningEffort,
			}, pendingEventID)
			return err
		},
		GenerateHumanize: func(ctx context.Context, missionID string, req reporting.HumanizeRequest, pendingEventID string) error {
			_, err := server.createReportHumanize(ctx, missionID, reportHumanizeRequest{
				Title:                req.Title,
				AgentExecutor:        req.AgentExecutor,
				AgentModel:           req.AgentModel,
				AgentReasoningEffort: req.AgentReasoningEffort,
				MCPMode:              req.MCPMode,
			}, pendingEventID, req)
			return err
		},
		GeneratePatch: func(ctx context.Context, missionID string, req reporting.PatchRequest, pendingEventID string) error {
			_, err := server.createReportPatch(ctx, missionID, reportPatchRequest{
				BaseArtifactID:       req.BaseArtifactID,
				Instruction:          req.Instruction,
				Title:                req.Title,
				AgentExecutor:        req.AgentExecutor,
				AgentModel:           req.AgentModel,
				AgentReasoningEffort: req.AgentReasoningEffort,
				MCPMode:              req.MCPMode,
				ReportSessionPolicy:  req.ReportSessionPolicy,
			}, pendingEventID, req)
			return err
		},
	}
}

func (server *Server) reconcileStaleReportDrafts(ctx context.Context, missionID string) error {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return err
	}
	completed := reporting.CompletedPendingEventIDs(events)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if _, ok := completed[event.EventID]; ok {
			continue
		}
		switch event.EventType {
		case "report.draft.pending":
			if server.runningReports.Owns(missionID, event.EventID) {
				continue
			}
			if err := server.resumeReportDraftWorker(ctx, missionID, event); err != nil {
				return err
			}
			return nil
		case "report.humanize.pending":
			if server.runningReports.Owns(missionID, event.EventID) {
				continue
			}
			if reportPendingEventID := reportHumanizeInFlightPendingEventID(event); reportPendingEventID != "" && server.runningReports.Owns(missionID, reportPendingEventID) {
				continue
			}
			if recovered, err := server.recoverStaleReportHumanizeFinalizedPatch(ctx, missionID, event); err != nil {
				return err
			} else if recovered {
				return nil
			}
			if err := server.reportRunner().ResumeHumanize(ctx, missionID, event); err != nil {
				return err
			}
			return nil
		case "report.patch.pending":
			if server.runningReports.Owns(missionID, event.EventID) {
				continue
			}
			if err := server.reportRunner().ResumePatch(ctx, missionID, event); err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

func (server *Server) createReportDraft(ctx context.Context, missionID string, req reportDraftRequest, pendingEventID string) (map[string]any, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Mission report"
	}
	executorName, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		return nil, err
	}
	mcpMode, err := normalizeMCPMode(req.MCPMode)
	if err != nil {
		return nil, err
	}
	rigor, err := normalizeReportRigorProfile(req.RigorLevel)
	if err != nil {
		return nil, err
	}
	reportMode, err := normalizeReportMode(req.ReportMode)
	if err != nil {
		return nil, err
	}
	reportSessionPolicy, err := normalizeReportSessionPolicy(req.ReportSessionPolicy)
	if err != nil {
		return nil, err
	}
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return nil, fmt.Errorf("%w: report generation requires an agent executor", app.ErrInvalidInput)
	}
	agentModel := strings.TrimSpace(req.AgentModel)
	agentReasoningEffort := strings.TrimSpace(req.AgentReasoningEffort)
	agentModel, agentReasoningEffort, err = resolveAgentSettings(executorName, agentModel, agentReasoningEffort, server.latestAgentSessionID(ctx, missionID, executorName))
	if err != nil {
		return nil, err
	}
	if err := server.validateReportSessionPolicy(ctx, missionID, executorName, reportMode, reportSessionPolicy, executor, false); err != nil {
		return nil, err
	}
	postReportHumanize := normalizePostReportHumanize(req.PostReportHumanize)
	guidanceProfile, guidanceSHA, err := SelectReportGenerationGuidance(req.GenerationGuidanceProfile)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GenerationGuidanceSHA256) != "" {
		guidanceSHA = strings.TrimSpace(req.GenerationGuidanceSHA256)
	}
	switch reportMode {
	case reportModeLongForm:
		return server.createSectionalLongFormReportDraft(ctx, missionID, title, executorName, agentModel, agentReasoningEffort, mcpMode, rigor, reportSessionPolicy, req.ReportSessionPolicySelection, postReportHumanize, guidanceProfile, guidanceSHA, pendingEventID, executor)
	case reportModePlanned:
		return server.createPlannedReportDraft(ctx, missionID, title, executorName, agentModel, agentReasoningEffort, mcpMode, rigor, reportSessionPolicy, req.ReportSessionPolicySelection, postReportHumanize, guidanceProfile, guidanceSHA, pendingEventID, executor)
	default:
		return server.createOneTakeReportDraft(ctx, missionID, title, executorName, agentModel, agentReasoningEffort, mcpMode, rigor, reportSessionPolicy, req.ReportSessionPolicySelection, postReportHumanize, guidanceProfile, guidanceSHA, pendingEventID, executor)
	}
}

func (server *Server) createReportHumanize(ctx context.Context, missionID string, req reportHumanizeRequest, pendingEventID string, humanizeReq reporting.HumanizeRequest) (map[string]any, error) {
	sourceArtifact, err := server.reportArtifact(ctx, missionID, humanizeReq.SourceArtifactID)
	if err != nil {
		return nil, err
	}
	if !isMarkdownMediaType(sourceArtifact.MediaType) {
		return nil, fmt.Errorf("%w: H5 humanize requires a Markdown report artifact", app.ErrInvalidInput)
	}
	executorName, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		return nil, err
	}
	mcpMode, err := normalizeMCPMode(req.MCPMode)
	if err != nil {
		return nil, err
	}
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return nil, fmt.Errorf("%w: H5 humanize requires an agent executor", app.ErrInvalidInput)
	}
	reportSessionID := strings.TrimSpace(humanizeReq.PreviousAgentSessionID)
	if reportSessionID == "" {
		info, err := server.reportArtifactSessionInfo(ctx, missionID, sourceArtifact.ArtifactID)
		if err != nil {
			return nil, err
		}
		reportSessionID = info.ReportSessionID
	}
	if reportSessionID == "" {
		return nil, fmt.Errorf("%w: H5 humanize requires a report session", app.ErrInvalidInput)
	}
	agentModel := firstNonEmpty(strings.TrimSpace(req.AgentModel), strings.TrimSpace(humanizeReq.AgentModel))
	agentReasoningEffort := firstNonEmpty(strings.TrimSpace(req.AgentReasoningEffort), strings.TrimSpace(humanizeReq.AgentReasoningEffort))
	reportMode := firstNonEmpty(strings.TrimSpace(humanizeReq.ReportMode), defaultReportMode)
	humanized, err := server.humanizeMarkdownReport(ctx, missionID, reportHumanizeInput{
		Title:                  firstNonEmpty(req.Title, humanizeReq.Title, reportArtifactTitle(sourceArtifact)),
		Markdown:               strings.TrimSpace(string(sourceArtifact.Content)),
		SourceArtifact:         sourceArtifact,
		ExecutorName:           executorName,
		AgentModel:             agentModel,
		ReasoningEffort:        agentReasoningEffort,
		MCPMode:                mcpMode,
		PreviousSessionID:      reportSessionID,
		ReportMode:             reportMode,
		PendingEventID:         strings.TrimSpace(humanizeReq.ReportPendingEventID),
		HumanizePendingEventID: pendingEventID,
		ToolSessionID:          strings.TrimSpace(humanizeReq.ToolSessionID),
	}, executor)
	if err != nil {
		return nil, err
	}
	return map[string]any{"source_artifact": sourceArtifact, "humanized": humanized}, nil
}

func (server *Server) createReportPatch(ctx context.Context, missionID string, req reportPatchRequest, pendingEventID string, patchReq reporting.PatchRequest) (map[string]any, error) {
	baseArtifact, err := server.reportArtifact(ctx, missionID, req.BaseArtifactID)
	if err != nil {
		return nil, err
	}
	if !isMarkdownMediaType(baseArtifact.MediaType) {
		return nil, fmt.Errorf("%w: report patch requires a Markdown report artifact", app.ErrInvalidInput)
	}
	instruction := strings.TrimSpace(req.Instruction)
	if instruction == "" {
		return nil, fmt.Errorf("%w: report patch instruction is required", app.ErrInvalidInput)
	}
	executorName, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		return nil, err
	}
	mcpMode, err := normalizeMCPMode(req.MCPMode)
	if err != nil {
		return nil, err
	}
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return nil, fmt.Errorf("%w: report patch requires an agent executor", app.ErrInvalidInput)
	}
	reportSessionID := strings.TrimSpace(patchReq.ReportSessionID)
	if reportSessionID == "" {
		return nil, fmt.Errorf("%w: report patch requires a report session", app.ErrInvalidInput)
	}
	title := firstNonEmpty(req.Title, patchReq.Title, reportArtifactTitle(baseArtifact)+" 수정본")
	agentModel := strings.TrimSpace(req.AgentModel)
	if agentModel == "" {
		agentModel = strings.TrimSpace(patchReq.AgentModel)
	}
	agentReasoningEffort := strings.TrimSpace(req.AgentReasoningEffort)
	if agentReasoningEffort == "" {
		agentReasoningEffort = strings.TrimSpace(patchReq.AgentReasoningEffort)
	}
	toolSessionID := newID("ses")
	started := time.Now()
	result, err := executor.Run(ctx, AgentRequest{
		UserText:          "patch markdown report artifact with MCP",
		Prompt:            agentReportPatchPrompt(title, missionID, toolSessionID, pendingEventID, baseArtifact.ArtifactID, instruction, patchReq),
		Model:             agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		PreviousSessionID: reportSessionID,
		AgentExecutor:     executorName,
		MCPMode:           mcpMode,
		ExtraMCPTools:     reportPatchMCPTools(),
		ReplaceMCPTools:   true,
		ReportPatch: &AgentReportPatchContext{
			BaseArtifactID:               baseArtifact.ArtifactID,
			PendingEventID:               pendingEventID,
			AgentExecutor:                executorName,
			AgentModel:                   agentModel,
			AgentReasoningEffort:         agentReasoningEffort,
			MCPMode:                      mcpMode,
			AgentSessionID:               reportSessionID,
			PreviousAgentSessionID:       patchReq.PreviousAgentSessionID,
			ReturnedAgentSessionID:       reportSessionID,
			ReportSessionID:              reportSessionID,
			ForkSourceAgentSessionID:     patchReq.ForkSourceAgentSessionID,
			ReportSessionPolicy:          patchReq.ReportSessionPolicy,
			ReportSessionPolicySelection: patchReq.ReportSessionPolicySelection,
			SessionChainKind:             patchReq.SessionChainKind,
		},
	})
	durationMS := time.Since(started).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("report patch agent failed: %w", reportAgentFailure(err, result, "report_patch", durationMS, reportSessionID))
	}
	validated, err := validatedSameSessionResult(result, reportSessionID)
	if err != nil {
		return nil, reportAgentFailure(err, result, "report_patch", durationMS, reportSessionID)
	}
	if _, ok, err := server.reportArtifactEventForPending(ctx, missionID, pendingEventID); err != nil {
		return nil, err
	} else if ok {
		return map[string]any{
			"status":           "completed",
			"agent_session_id": validated.SessionID,
		}, nil
	}
	finalizedEvent, ok, err := server.reportPatchFinalizedEventForPending(ctx, missionID, pendingEventID)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, reportAgentFailure(fmt.Errorf("%w: report patch agent did not finalize through MCP", app.ErrInvalidInput), result, "report_patch", durationMS, reportSessionID)
	}
	if _, err := server.promoteReportPatchFinalizedArtifact(ctx, missionID, finalizedEvent); err != nil {
		return nil, err
	}
	return map[string]any{
		"status":           "completed",
		"agent_session_id": validated.SessionID,
	}, nil
}

func (server *Server) reportArtifactEventForPending(ctx context.Context, missionID string, pendingEventID string) (app.LedgerEvent, bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	pendingEventID = strings.TrimSpace(pendingEventID)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "report.artifact.created" {
			continue
		}
		if reportDraftPendingEventID(event) == pendingEventID {
			return event, true, nil
		}
	}
	return app.LedgerEvent{}, false, nil
}

func (server *Server) reportPatchFinalizedEventForPending(ctx context.Context, missionID string, pendingEventID string) (app.LedgerEvent, bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	pendingEventID = strings.TrimSpace(pendingEventID)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "report.patch.finalized" {
			continue
		}
		var payload struct {
			PendingEventID string `json:"pending_event_id"`
			ArtifactID     string `json:"artifact_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.PendingEventID) == pendingEventID && strings.TrimSpace(payload.ArtifactID) != "" {
			return event, true, nil
		}
	}
	return app.LedgerEvent{}, false, nil
}

func (server *Server) promoteReportPatchFinalizedArtifact(ctx context.Context, missionID string, finalized app.LedgerEvent) (app.LedgerEvent, error) {
	var payload map[string]any
	if err := json.Unmarshal(finalized.Payload, &payload); err != nil {
		return app.LedgerEvent{}, fmt.Errorf("%w: invalid report patch finalized payload", app.ErrInvalidInput)
	}
	pendingEventID, _ := payload["pending_event_id"].(string)
	pendingEventID = strings.TrimSpace(pendingEventID)
	if pendingEventID == "" {
		return app.LedgerEvent{}, fmt.Errorf("%w: report patch finalized payload is missing pending_event_id", app.ErrInvalidInput)
	}
	if event, ok, err := server.reportArtifactEventForPending(ctx, missionID, pendingEventID); err != nil {
		return app.LedgerEvent{}, err
	} else if ok {
		return event, nil
	}
	artifactID, _ := payload["artifact_id"].(string)
	artifact, err := server.service.GetRawArtifact(ctx, strings.TrimSpace(artifactID))
	if err != nil {
		return app.LedgerEvent{}, err
	}
	if artifact.MissionID != missionID {
		return app.LedgerEvent{}, fmt.Errorf("%w: finalized report artifact belongs to another mission", app.ErrInvalidInput)
	}
	producerID, _ := payload["report_session_id"].(string)
	producerID = firstNonEmpty(strings.TrimSpace(producerID), strings.TrimSpace(finalized.CorrelationID))
	return server.service.AppendEvent(ctx, reporting.BuildPromotedMarkdownReportArtifactAppendRequest(reporting.PromotedMarkdownReportArtifactEventRequest{
		EventID:             newID("evt"),
		MissionID:           missionID,
		PromotedFromEventID: finalized.EventID,
		Payload:             payload,
		Producer:            app.Producer{Type: "agent_session", ID: producerID},
	}))
}

func reportPatchMCPTools() []string {
	return []string{
		plasmamcp.ToolReportPatchStart,
		plasmamcp.ToolReportPatchRead,
		plasmamcp.ToolReportPatchApply,
		plasmamcp.ToolReportPatchFinalize,
	}
}

func (server *Server) createOneTakeReportDraft(ctx context.Context, missionID string, title string, executorName string, agentModel string, agentReasoningEffort string, mcpMode string, rigor reportRigorProfile, reportSessionPolicy string, reportSessionPolicySelection string, postReportHumanize string, generationGuidanceProfile string, generationGuidanceSHA256 string, pendingEventID string, executor AgentExecutor) (map[string]any, error) {
	artifactID := newID("art")
	toolSessionID := newID("ses")
	reportSessionPolicy = firstNonEmpty(reportSessionPolicy, reportSessionPolicySameSession)
	reportSessionPolicySelection = strings.TrimSpace(reportSessionPolicySelection)
	previousSessionID := server.latestAgentSessionID(ctx, missionID, executorName)
	started := time.Now()
	result, err := executor.Run(ctx, AgentRequest{
		UserText:          "generate quick markdown report artifact",
		Prompt:            agentOneTakeMarkdownReportPrompt(title, missionID, toolSessionID, rigor, generationGuidanceProfile),
		Model:             agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		PreviousSessionID: previousSessionID,
		AgentExecutor:     executorName,
		MCPMode:           mcpMode,
	})
	agentDurationMS := time.Since(started).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("quick report agent failed: %w", reportAgentFailure(err, result, "report_one_take", agentDurationMS, previousSessionID))
	}
	returnedSessionID := strings.TrimSpace(result.SessionID)
	result, err = validatedSameSessionResult(result, previousSessionID)
	if err != nil {
		return nil, reportAgentFailure(err, result, "report_one_take", agentDurationMS, previousSessionID)
	}
	markdown := strings.TrimSpace(result.Text)
	if markdown == "" {
		return nil, reportAgentFailure(fmt.Errorf("%w: report agent returned empty Markdown", app.ErrInvalidInput), result, "report_one_take", agentDurationMS, previousSessionID)
	}
	artifact, err := server.service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: artifactID,
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   safeFilename(title, ".md"),
		Producer:   app.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)},
		Content:    []byte(markdown),
	})
	if err != nil {
		return nil, err
	}
	event, err := server.service.AppendEvent(ctx, reporting.BuildMarkdownReportArtifactCreatedAppendRequest(reporting.MarkdownReportArtifactCreatedEventRequest{
		MarkdownReportEventBase: reporting.MarkdownReportEventBase{
			EventID:                      newID("evt"),
			MissionID:                    missionID,
			PendingEventID:               pendingEventID,
			Title:                        title,
			AgentExecutor:                executorName,
			AgentModel:                   agentModel,
			AgentReasoningEffort:         agentReasoningEffort,
			AgentSessionID:               result.SessionID,
			PreviousAgentSessionID:       previousSessionID,
			ReturnedAgentSessionID:       returnedSessionID,
			ToolSessionID:                toolSessionID,
			MCPMode:                      mcpMode,
			RigorLevel:                   rigor.level,
			RigorLabel:                   rigor.label,
			ReportMode:                   reportModeOneTake,
			ReportModeLabel:              reportModeLabel(reportModeOneTake),
			ReportSessionPolicy:          reportSessionPolicy,
			ReportSessionPolicySelection: reportSessionPolicySelection,
			PostReportHumanize:           postReportHumanize,
			HumanizeEnabled:              postReportHumanize != "disabled",
			GenerationGuidanceProfile:    generationGuidanceProfile,
			GenerationGuidanceSHA256:     generationGuidanceSHA256,
			SessionChainKind:             "same_session_report",
			PreReportResearchSessionID:   previousSessionID,
			ReportPlanSessionID:          "",
			ReportSessionID:              result.SessionID,
			ForkSourceAgentSessionID:     "",
			PostReportResearchSessionID:  "",
			CompositionStrategy:          "one_take_markdown",
			DurationMS:                   time.Since(started).Milliseconds(),
			Text:                         "빠른 Markdown 리포트 artifact를 생성했습니다.",
			AgentUsage:                   result.Usage,
			AgentUsageSurface:            "report_one_take",
			AgentUsageDurationMS:         agentDurationMS,
			AgentResumed:                 result.Resumed,
			Producer:                     app.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)},
		},
		Artifact:          artifact,
		PlanReviewState:   "not_applicable",
		IncludePlanReview: false,
	}))
	if err != nil {
		return nil, err
	}
	if postReportHumanize == "disabled" {
		return map[string]any{"artifact": artifact, "event": event, "markdown": markdown}, nil
	}
	humanized, err := server.humanizeMarkdownReport(ctx, missionID, reportHumanizeInput{
		Title:             title,
		Markdown:          markdown,
		SourceArtifact:    artifact,
		ExecutorName:      executorName,
		AgentModel:        agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MCPMode:           mcpMode,
		PreviousSessionID: result.SessionID,
		ReportMode:        reportModeOneTake,
		PendingEventID:    pendingEventID,
	}, executor)
	if err != nil {
		return nil, err
	}
	return map[string]any{"artifact": artifact, "event": event, "markdown": markdown, "humanized": humanized}, nil
}

func (server *Server) createPlannedReportDraft(ctx context.Context, missionID string, title string, executorName string, agentModel string, agentReasoningEffort string, mcpMode string, rigor reportRigorProfile, reportSessionPolicy string, reportSessionPolicySelection string, postReportHumanize string, generationGuidanceProfile string, generationGuidanceSHA256 string, pendingEventID string, executor AgentExecutor) (map[string]any, error) {
	artifactID := newID("art")
	planToolSessionID := newID("ses")
	if strings.TrimSpace(reportSessionPolicy) == "" {
		reportSessionPolicy = reportSessionPolicySameSession
	}
	reportSessionPolicySelection = strings.TrimSpace(reportSessionPolicySelection)
	previousSessionID := server.latestAgentSessionID(ctx, missionID, executorName)
	reportStartSessionID := previousSessionID
	forkSourceSessionID := ""
	sessionChainKind := "same_session_report"
	if reportSessionPolicy == reportSessionPolicyIsolatedFork {
		if strings.TrimSpace(previousSessionID) == "" {
			return nil, fmt.Errorf("%w: isolated report session requires a pre-report research session", app.ErrInvalidInput)
		}
		forker, ok := executor.(AgentSessionForker)
		if !ok {
			return nil, reporting.ValidateSessionPolicy(reportSessionPolicy, reportModePlanned, false, strings.TrimSpace(previousSessionID) != "", false)
		}
		fork, err := forker.ForkSession(ctx, previousSessionID)
		if err != nil {
			return nil, fmt.Errorf("report session fork failed: %w", err)
		}
		reportStartSessionID = fork.SessionID
		forkSourceSessionID = fork.SourceSessionID
		if strings.TrimSpace(forkSourceSessionID) == "" {
			forkSourceSessionID = previousSessionID
		}
		sessionChainKind = "isolated_fork_report"
	}
	started := time.Now()
	planStarted := time.Now()
	planResult, err := executor.Run(ctx, AgentRequest{
		UserText:          "plan markdown report artifact",
		Prompt:            agentReportPlanPrompt(title, missionID, planToolSessionID, rigor),
		Model:             agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MissionID:         missionID,
		ToolSessionID:     planToolSessionID,
		PreviousSessionID: reportStartSessionID,
		AgentExecutor:     executorName,
		MCPMode:           mcpMode,
	})
	planDurationMS := time.Since(planStarted).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("report planning agent failed: %w", reportAgentFailure(err, planResult, "report_plan", planDurationMS, reportStartSessionID))
	}
	returnedPlanSessionID := strings.TrimSpace(planResult.SessionID)
	planResult, err = validatedSameSessionResult(planResult, reportStartSessionID)
	if err != nil {
		return nil, reportAgentFailure(err, planResult, "report_plan", planDurationMS, reportStartSessionID)
	}
	plan, err := parseAgentReportPlan(planResult.Text)
	if err != nil {
		return nil, reportAgentFailure(err, planResult, "report_plan", planDurationMS, reportStartSessionID)
	}
	planEvent, err := server.service.AppendEvent(ctx, reporting.BuildMarkdownReportPlanCreatedAppendRequest(reporting.MarkdownReportPlanCreatedEventRequest{
		MarkdownReportEventBase: reporting.MarkdownReportEventBase{
			EventID:                      newID("evt"),
			MissionID:                    missionID,
			PendingEventID:               pendingEventID,
			Title:                        title,
			AgentExecutor:                executorName,
			AgentModel:                   agentModel,
			AgentReasoningEffort:         agentReasoningEffort,
			AgentSessionID:               planResult.SessionID,
			PreviousAgentSessionID:       reportStartSessionID,
			ReturnedAgentSessionID:       returnedPlanSessionID,
			ToolSessionID:                planToolSessionID,
			MCPMode:                      mcpMode,
			RigorLevel:                   rigor.level,
			RigorLabel:                   rigor.label,
			ReportMode:                   reportModePlanned,
			ReportModeLabel:              reportModeLabel(reportModePlanned),
			ReportSessionPolicy:          reportSessionPolicy,
			ReportSessionPolicySelection: reportSessionPolicySelection,
			PostReportHumanize:           postReportHumanize,
			HumanizeEnabled:              postReportHumanize != "disabled",
			GenerationGuidanceProfile:    generationGuidanceProfile,
			GenerationGuidanceSHA256:     generationGuidanceSHA256,
			SessionChainKind:             sessionChainKind,
			PreReportResearchSessionID:   previousSessionID,
			ReportPlanSessionID:          planResult.SessionID,
			ReportSessionID:              "",
			ForkSourceAgentSessionID:     forkSourceSessionID,
			PostReportResearchSessionID:  "",
			CompositionStrategy:          "planned_markdown",
			DurationMS:                   planDurationMS,
			Text:                         "Markdown 리포트 생성 계획을 만들었습니다.",
			AgentUsage:                   planResult.Usage,
			AgentUsageSurface:            "report_plan",
			AgentUsageDurationMS:         planDurationMS,
			AgentResumed:                 planResult.Resumed,
			Producer:                     app.Producer{Type: "agent_session", ID: fallbackSessionID(planResult.SessionID, planToolSessionID)},
		},
		ArtifactID:         artifactID,
		Plan:               plan,
		PlanReviewRequired: false,
		PlanReviewState:    "auto_accepted",
	}))
	if err != nil {
		return nil, err
	}

	toolSessionID := newID("ses")
	reportStarted := time.Now()
	result, err := executor.Run(ctx, AgentRequest{
		UserText:          "generate markdown report artifact",
		Prompt:            agentMarkdownReportPrompt(title, missionID, toolSessionID, rigor, plan, generationGuidanceProfile),
		Model:             agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		PreviousSessionID: planResult.SessionID,
		AgentExecutor:     executorName,
		MCPMode:           mcpMode,
	})
	reportDurationMS := time.Since(reportStarted).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("report agent failed: %w", reportAgentFailure(err, result, "report_markdown", reportDurationMS, planResult.SessionID))
	}
	returnedSessionID := strings.TrimSpace(result.SessionID)
	result, err = validatedSameSessionResult(result, planResult.SessionID)
	if err != nil {
		return nil, reportAgentFailure(err, result, "report_markdown", reportDurationMS, planResult.SessionID)
	}
	markdown := strings.TrimSpace(result.Text)
	if markdown == "" {
		return nil, reportAgentFailure(fmt.Errorf("%w: report agent returned empty Markdown", app.ErrInvalidInput), result, "report_markdown", reportDurationMS, planResult.SessionID)
	}
	artifact, err := server.service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: artifactID,
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   safeFilename(title, ".md"),
		Producer:   app.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)},
		Content:    []byte(markdown),
	})
	if err != nil {
		return nil, err
	}
	event, err := server.service.AppendEvent(ctx, reporting.BuildMarkdownReportArtifactCreatedAppendRequest(reporting.MarkdownReportArtifactCreatedEventRequest{
		MarkdownReportEventBase: reporting.MarkdownReportEventBase{
			EventID:                      newID("evt"),
			MissionID:                    missionID,
			PendingEventID:               pendingEventID,
			Title:                        title,
			AgentExecutor:                executorName,
			AgentModel:                   agentModel,
			AgentReasoningEffort:         agentReasoningEffort,
			AgentSessionID:               result.SessionID,
			PreviousAgentSessionID:       planResult.SessionID,
			ReturnedAgentSessionID:       returnedSessionID,
			ToolSessionID:                toolSessionID,
			MCPMode:                      mcpMode,
			RigorLevel:                   rigor.level,
			RigorLabel:                   rigor.label,
			ReportMode:                   reportModePlanned,
			ReportModeLabel:              reportModeLabel(reportModePlanned),
			ReportSessionPolicy:          reportSessionPolicy,
			ReportSessionPolicySelection: reportSessionPolicySelection,
			PostReportHumanize:           postReportHumanize,
			HumanizeEnabled:              postReportHumanize != "disabled",
			GenerationGuidanceProfile:    generationGuidanceProfile,
			GenerationGuidanceSHA256:     generationGuidanceSHA256,
			SessionChainKind:             sessionChainKind,
			PreReportResearchSessionID:   previousSessionID,
			ReportPlanSessionID:          planResult.SessionID,
			ReportSessionID:              result.SessionID,
			ForkSourceAgentSessionID:     forkSourceSessionID,
			PostReportResearchSessionID:  "",
			CompositionStrategy:          "planned_markdown",
			DurationMS:                   time.Since(started).Milliseconds(),
			Text:                         "계획 기반 Markdown 리포트 artifact를 생성했습니다.",
			AgentUsage:                   result.Usage,
			AgentUsageSurface:            "report_markdown",
			AgentUsageDurationMS:         reportDurationMS,
			AgentResumed:                 result.Resumed,
			Producer:                     app.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)},
		},
		Artifact:           artifact,
		PlanEventID:        planEvent.EventID,
		PlanToolSessionID:  planToolSessionID,
		IncludePlanReview:  true,
		PlanReviewRequired: false,
		PlanReviewState:    "auto_accepted",
	}))
	if err != nil {
		return nil, err
	}
	if postReportHumanize == "disabled" {
		return map[string]any{"artifact": artifact, "event": event, "markdown": markdown}, nil
	}
	humanized, err := server.humanizeMarkdownReport(ctx, missionID, reportHumanizeInput{
		Title:             title,
		Markdown:          markdown,
		SourceArtifact:    artifact,
		ExecutorName:      executorName,
		AgentModel:        agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MCPMode:           mcpMode,
		PreviousSessionID: result.SessionID,
		ReportMode:        reportModePlanned,
		PendingEventID:    pendingEventID,
	}, executor)
	if err != nil {
		return nil, err
	}
	return map[string]any{"artifact": artifact, "event": event, "markdown": markdown, "humanized": humanized}, nil
}

func (server *Server) createSectionalLongFormReportDraft(ctx context.Context, missionID string, title string, executorName string, agentModel string, agentReasoningEffort string, mcpMode string, rigor reportRigorProfile, reportSessionPolicy string, reportSessionPolicySelection string, postReportHumanize string, generationGuidanceProfile string, generationGuidanceSHA256 string, pendingEventID string, executor AgentExecutor) (map[string]any, error) {
	started := time.Now()
	progress, err := server.loadSectionalReportProgress(ctx, missionID, pendingEventID)
	if err != nil {
		return nil, err
	}
	artifactID := strings.TrimSpace(progress.artifactID)
	if artifactID == "" {
		artifactID = newID("art")
	}
	planEvent := progress.planEvent
	plan := progress.plan
	currentSessionID := strings.TrimSpace(progress.currentSessionID)
	reportSessionPolicy = firstNonEmpty(reportSessionPolicy, reportSessionPolicySameSession)
	reportSessionPolicySelection = strings.TrimSpace(reportSessionPolicySelection)
	if progress.hasPlan {
		reportSessionPolicy = firstNonEmpty(progress.reportSessionPolicy, reportSessionPolicy)
		reportSessionPolicySelection = firstNonEmpty(progress.reportSessionPolicySelection, reportSessionPolicySelection)
	}
	sessionChainKind := firstNonEmpty(progress.sessionChainKind, "same_session_report")
	preReportResearchSessionID := strings.TrimSpace(progress.preReportResearchSessionID)
	forkSourceSessionID := strings.TrimSpace(progress.forkSourceSessionID)
	reportPlanSessionID := strings.TrimSpace(progress.reportPlanSessionID)
	if !progress.hasPlan {
		planToolSessionID := newID("ses")
		previousSessionID := server.latestAgentSessionID(ctx, missionID, executorName)
		preReportResearchSessionID = previousSessionID
		reportStartSessionID := previousSessionID
		if reportSessionPolicy == reportSessionPolicyIsolatedFork {
			if strings.TrimSpace(previousSessionID) == "" {
				return nil, fmt.Errorf("%w: isolated report session requires a pre-report research session", app.ErrInvalidInput)
			}
			forker, ok := executor.(AgentSessionForker)
			if !ok {
				return nil, reporting.ValidateSessionPolicy(reportSessionPolicy, reportModeLongForm, false, strings.TrimSpace(previousSessionID) != "", false)
			}
			fork, err := forker.ForkSession(ctx, previousSessionID)
			if err != nil {
				return nil, fmt.Errorf("report session fork failed: %w", err)
			}
			reportStartSessionID = fork.SessionID
			forkSourceSessionID = fork.SourceSessionID
			if strings.TrimSpace(forkSourceSessionID) == "" {
				forkSourceSessionID = previousSessionID
			}
			sessionChainKind = "isolated_fork_report"
		}
		planStarted := time.Now()
		planResult, err := executor.Run(ctx, AgentRequest{
			UserText:          "plan sectional long-form markdown report",
			Prompt:            agentSectionalReportPlanPrompt(title, missionID, planToolSessionID, rigor),
			Model:             agentModel,
			ReasoningEffort:   agentReasoningEffort,
			MissionID:         missionID,
			ToolSessionID:     planToolSessionID,
			PreviousSessionID: reportStartSessionID,
			AgentExecutor:     executorName,
			MCPMode:           mcpMode,
		})
		planDurationMS := time.Since(planStarted).Milliseconds()
		if err != nil {
			return nil, fmt.Errorf("sectional report planning agent failed: %w", reportAgentFailure(err, planResult, "report_plan", planDurationMS, reportStartSessionID))
		}
		returnedPlanSessionID := strings.TrimSpace(planResult.SessionID)
		planResult, err = validatedSameSessionResult(planResult, reportStartSessionID)
		if err != nil {
			return nil, reportAgentFailure(err, planResult, "report_plan", planDurationMS, reportStartSessionID)
		}
		plan, err = parseAgentSectionalReportPlan(planResult.Text)
		if err != nil {
			return nil, reportAgentFailure(err, planResult, "report_plan", planDurationMS, reportStartSessionID)
		}
		reportPlanSessionID = planResult.SessionID
		planEvent, err = server.service.AppendEvent(ctx, reporting.BuildMarkdownReportPlanCreatedAppendRequest(reporting.MarkdownReportPlanCreatedEventRequest{
			MarkdownReportEventBase: reporting.MarkdownReportEventBase{
				EventID:                      newID("evt"),
				MissionID:                    missionID,
				PendingEventID:               pendingEventID,
				Title:                        title,
				AgentExecutor:                executorName,
				AgentModel:                   agentModel,
				AgentReasoningEffort:         agentReasoningEffort,
				AgentSessionID:               planResult.SessionID,
				PreviousAgentSessionID:       reportStartSessionID,
				ReturnedAgentSessionID:       returnedPlanSessionID,
				ToolSessionID:                planToolSessionID,
				MCPMode:                      mcpMode,
				RigorLevel:                   rigor.level,
				RigorLabel:                   rigor.label,
				ReportMode:                   reportModeLongForm,
				ReportModeLabel:              reportModeLabel(reportModeLongForm),
				ReportSessionPolicy:          reportSessionPolicy,
				ReportSessionPolicySelection: reportSessionPolicySelection,
				PostReportHumanize:           postReportHumanize,
				HumanizeEnabled:              postReportHumanize != "disabled",
				GenerationGuidanceProfile:    generationGuidanceProfile,
				GenerationGuidanceSHA256:     generationGuidanceSHA256,
				SessionChainKind:             sessionChainKind,
				PreReportResearchSessionID:   preReportResearchSessionID,
				ReportPlanSessionID:          planResult.SessionID,
				ReportSessionID:              "",
				ForkSourceAgentSessionID:     forkSourceSessionID,
				PostReportResearchSessionID:  "",
				CompositionStrategy:          "sectional_preserve_markdown",
				DurationMS:                   planDurationMS,
				Text:                         "섹션별 장문 Markdown 리포트 생성 계획을 만들었습니다.",
				AgentUsage:                   planResult.Usage,
				AgentUsageSurface:            "report_plan",
				AgentUsageDurationMS:         planDurationMS,
				AgentResumed:                 planResult.Resumed,
				Producer:                     app.Producer{Type: "agent_session", ID: fallbackSessionID(planResult.SessionID, planToolSessionID)},
			},
			ArtifactID:         artifactID,
			Plan:               plan,
			AssemblyStrategy:   "c4_normalized_section_headings",
			PlanReviewRequired: false,
			PlanReviewState:    "auto_accepted",
		}))
		if err != nil {
			return nil, err
		}
		currentSessionID = strings.TrimSpace(planResult.SessionID)
	}
	if currentSessionID == "" {
		currentSessionID = server.latestAgentSessionID(ctx, missionID, executorName)
	}

	sectionDraftsByPart := make([][]sectionalReportDraft, 0, len(plan.Parts))
	sectionArtifactIDs := []string{}
	sectionWordTotal := 0
	for partIndex, part := range plan.Parts {
		if draft, ok := progress.parts[partIndex]; ok {
			sectionDraftsByPart = append(sectionDraftsByPart, nil)
			sectionWordTotal += draft.WordCount
			continue
		}
		partDrafts := make([]sectionalReportDraft, 0, len(part.Sections))
		for sectionIndex, section := range part.Sections {
			if draft, ok := progress.sections[sectionalReportIndex{part: partIndex, section: sectionIndex}]; ok {
				partDrafts = append(partDrafts, draft)
				sectionArtifactIDs = append(sectionArtifactIDs, draft.ArtifactID)
				sectionWordTotal += draft.WordCount
				continue
			}
			toolSessionID := newID("ses")
			previousStageSessionID := currentSessionID
			sectionStarted := time.Now()
			result, err := executor.Run(ctx, AgentRequest{
				UserText:          fmt.Sprintf("draft section %d.%d for sectional long-form markdown report", partIndex+1, sectionIndex+1),
				Prompt:            agentSectionDraftPrompt(title, missionID, toolSessionID, rigor, plan, part, section, partIndex, sectionIndex, generationGuidanceProfile),
				Model:             agentModel,
				ReasoningEffort:   agentReasoningEffort,
				MissionID:         missionID,
				ToolSessionID:     toolSessionID,
				PreviousSessionID: previousStageSessionID,
				AgentExecutor:     executorName,
				MCPMode:           mcpMode,
			})
			sectionDurationMS := time.Since(sectionStarted).Milliseconds()
			if err != nil {
				return nil, fmt.Errorf("sectional report section agent failed: %w", reportAgentFailure(err, result, "report_section", sectionDurationMS, previousStageSessionID))
			}
			returnedSessionID := strings.TrimSpace(result.SessionID)
			result, err = validatedSameSessionResult(result, previousStageSessionID)
			if err != nil {
				return nil, reportAgentFailure(err, result, "report_section", sectionDurationMS, previousStageSessionID)
			}
			currentSessionID = strings.TrimSpace(result.SessionID)
			markdown := strings.TrimSpace(result.Text)
			if markdown == "" {
				return nil, reportAgentFailure(fmt.Errorf("%w: section report agent returned empty Markdown", app.ErrInvalidInput), result, "report_section", sectionDurationMS, previousStageSessionID)
			}
			artifact, err := server.service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
				ArtifactID: newID("art"),
				MissionID:  missionID,
				MediaType:  "text/markdown; charset=utf-8",
				Filename:   safeFilename(fmt.Sprintf("%s part %02d section %02d", title, partIndex+1, sectionIndex+1), ".md"),
				Producer:   app.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)},
				Content:    []byte(markdown),
			})
			if err != nil {
				return nil, err
			}
			wordCount := reportWordCount(markdown)
			sectionWordTotal += wordCount
			sectionArtifactIDs = append(sectionArtifactIDs, artifact.ArtifactID)
			_, err = server.service.AppendEvent(ctx, reporting.BuildMarkdownReportSectionCreatedAppendRequest(reporting.MarkdownReportSectionCreatedEventRequest{
				MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
					EventID:                      newID("evt"),
					MissionID:                    missionID,
					PendingEventID:               pendingEventID,
					PlanEventID:                  planEvent.EventID,
					Title:                        section.Title,
					Artifact:                     artifact,
					AgentExecutor:                executorName,
					AgentModel:                   agentModel,
					AgentReasoningEffort:         agentReasoningEffort,
					AgentSessionID:               result.SessionID,
					PreviousAgentSessionID:       previousStageSessionID,
					ReturnedAgentSessionID:       returnedSessionID,
					ToolSessionID:                toolSessionID,
					ReportMode:                   reportModeLongForm,
					ReportModeLabel:              reportModeLabel(reportModeLongForm),
					ReportSessionPolicy:          reportSessionPolicy,
					ReportSessionPolicySelection: reportSessionPolicySelection,
					PostReportHumanize:           postReportHumanize,
					HumanizeEnabled:              postReportHumanize != "disabled",
					GenerationGuidanceProfile:    generationGuidanceProfile,
					GenerationGuidanceSHA256:     generationGuidanceSHA256,
					SessionChainKind:             sessionChainKind,
					PreReportResearchSessionID:   preReportResearchSessionID,
					ReportPlanSessionID:          reportPlanSessionID,
					ReportSessionID:              result.SessionID,
					ForkSourceAgentSessionID:     forkSourceSessionID,
					PostReportResearchSessionID:  "",
					CompositionStrategy:          "sectional_preserve_markdown",
					AssemblyStrategy:             "c4_normalized_section_headings",
					DurationMS:                   sectionDurationMS,
					Text:                         "장문 리포트 섹션 Markdown을 생성했습니다.",
					AgentUsage:                   result.Usage,
					AgentUsageSurface:            "report_section",
					AgentUsageDurationMS:         sectionDurationMS,
					AgentResumed:                 result.Resumed,
					Producer:                     app.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)},
				},
				PartIndex:    partIndex + 1,
				SectionIndex: sectionIndex + 1,
				WordCount:    wordCount,
			}))
			if err != nil {
				return nil, err
			}
			partDrafts = append(partDrafts, sectionalReportDraft{Title: section.Title, Markdown: markdown, ArtifactID: artifact.ArtifactID, WordCount: wordCount})
		}
		sectionDraftsByPart = append(sectionDraftsByPart, partDrafts)
	}

	partDrafts := make([]sectionalReportPartDraft, 0, len(plan.Parts))
	partArtifactIDs := []string{}
	for partIndex, part := range plan.Parts {
		if draft, ok := progress.parts[partIndex]; ok {
			partDrafts = append(partDrafts, draft)
			partArtifactIDs = append(partArtifactIDs, draft.ArtifactID)
			continue
		}
		toolSessionID := newID("ses")
		previousStageSessionID := currentSessionID
		partStarted := time.Now()
		result, err := executor.Run(ctx, AgentRequest{
			UserText:          fmt.Sprintf("assemble part %d for sectional long-form markdown report", partIndex+1),
			Prompt:            agentPartAssemblyPrompt(title, missionID, toolSessionID, rigor, plan, part, sectionDraftsByPart[partIndex], partIndex, generationGuidanceProfile),
			Model:             agentModel,
			ReasoningEffort:   agentReasoningEffort,
			MissionID:         missionID,
			ToolSessionID:     toolSessionID,
			PreviousSessionID: previousStageSessionID,
			AgentExecutor:     executorName,
			MCPMode:           mcpMode,
		})
		partDurationMS := time.Since(partStarted).Milliseconds()
		if err != nil {
			return nil, fmt.Errorf("sectional report part assembly agent failed: %w", reportAgentFailure(err, result, "report_part", partDurationMS, previousStageSessionID))
		}
		returnedSessionID := strings.TrimSpace(result.SessionID)
		result, err = validatedSameSessionResult(result, previousStageSessionID)
		if err != nil {
			return nil, reportAgentFailure(err, result, "report_part", partDurationMS, previousStageSessionID)
		}
		currentSessionID = strings.TrimSpace(result.SessionID)
		assembly, err := parseAgentPartAssembly(result.Text)
		if err != nil {
			return nil, reportAgentFailure(err, result, "report_part", partDurationMS, previousStageSessionID)
		}
		partMarkdown := assembleSectionalPartMarkdown(part, sectionDraftsByPart[partIndex], assembly, partIndex)
		artifact, err := server.service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
			ArtifactID: newID("art"),
			MissionID:  missionID,
			MediaType:  "text/markdown; charset=utf-8",
			Filename:   safeFilename(fmt.Sprintf("%s part %02d", title, partIndex+1), ".md"),
			Producer:   app.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)},
			Content:    []byte(partMarkdown),
		})
		if err != nil {
			return nil, err
		}
		partWordCount := reportWordCount(partMarkdown)
		partArtifactIDs = append(partArtifactIDs, artifact.ArtifactID)
		_, err = server.service.AppendEvent(ctx, reporting.BuildMarkdownReportPartCreatedAppendRequest(reporting.MarkdownReportPartCreatedEventRequest{
			MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
				EventID:                      newID("evt"),
				MissionID:                    missionID,
				PendingEventID:               pendingEventID,
				PlanEventID:                  planEvent.EventID,
				Title:                        part.Title,
				Artifact:                     artifact,
				AgentExecutor:                executorName,
				AgentModel:                   agentModel,
				AgentReasoningEffort:         agentReasoningEffort,
				AgentSessionID:               result.SessionID,
				PreviousAgentSessionID:       previousStageSessionID,
				ReturnedAgentSessionID:       returnedSessionID,
				ToolSessionID:                toolSessionID,
				ReportMode:                   reportModeLongForm,
				ReportModeLabel:              reportModeLabel(reportModeLongForm),
				ReportSessionPolicy:          reportSessionPolicy,
				ReportSessionPolicySelection: reportSessionPolicySelection,
				PostReportHumanize:           postReportHumanize,
				HumanizeEnabled:              postReportHumanize != "disabled",
				GenerationGuidanceProfile:    generationGuidanceProfile,
				GenerationGuidanceSHA256:     generationGuidanceSHA256,
				SessionChainKind:             sessionChainKind,
				PreReportResearchSessionID:   preReportResearchSessionID,
				ReportPlanSessionID:          reportPlanSessionID,
				ReportSessionID:              result.SessionID,
				ForkSourceAgentSessionID:     forkSourceSessionID,
				PostReportResearchSessionID:  "",
				CompositionStrategy:          "sectional_preserve_markdown",
				AssemblyStrategy:             "c4_normalized_section_headings",
				DurationMS:                   partDurationMS,
				Text:                         "장문 리포트 파트 Markdown을 보존 조립했습니다.",
				AgentUsage:                   result.Usage,
				AgentUsageSurface:            "report_part",
				AgentUsageDurationMS:         partDurationMS,
				AgentResumed:                 result.Resumed,
				Producer:                     app.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)},
			},
			PartIndex:    partIndex + 1,
			SectionCount: len(sectionDraftsByPart[partIndex]),
			WordCount:    partWordCount,
		}))
		if err != nil {
			return nil, err
		}
		partDrafts = append(partDrafts, sectionalReportPartDraft{Title: part.Title, Markdown: partMarkdown, ArtifactID: artifact.ArtifactID, WordCount: partWordCount})
	}

	toolSessionID := newID("ses")
	previousStageSessionID := currentSessionID
	frameStarted := time.Now()
	frameResult, err := executor.Run(ctx, AgentRequest{
		UserText:          "write front matter and closing for sectional long-form markdown report",
		Prompt:            agentSectionalFramePrompt(title, missionID, toolSessionID, rigor, plan, partDrafts, generationGuidanceProfile),
		Model:             agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		PreviousSessionID: previousStageSessionID,
		AgentExecutor:     executorName,
		MCPMode:           mcpMode,
	})
	frameDurationMS := time.Since(frameStarted).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("sectional report frame agent failed: %w", reportAgentFailure(err, frameResult, "report_frame", frameDurationMS, previousStageSessionID))
	}
	returnedSessionID := strings.TrimSpace(frameResult.SessionID)
	frameResult, err = validatedSameSessionResult(frameResult, previousStageSessionID)
	if err != nil {
		return nil, reportAgentFailure(err, frameResult, "report_frame", frameDurationMS, previousStageSessionID)
	}
	frame, err := parseAgentSectionalFrame(frameResult.Text)
	if err != nil {
		return nil, reportAgentFailure(err, frameResult, "report_frame", frameDurationMS, previousStageSessionID)
	}
	markdown := assembleSectionalFinalMarkdown(title, frame, partDrafts)
	if strings.TrimSpace(markdown) == "" {
		return nil, fmt.Errorf("%w: sectional report assembled empty Markdown", app.ErrInvalidInput)
	}
	finalWordCount := reportWordCount(markdown)
	preservationRatio := float64(finalWordCount) / float64(maxInt(1, sectionWordTotal))
	artifact, err := server.service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: artifactID,
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   safeFilename(title, ".md"),
		Producer:   app.Producer{Type: "agent_session", ID: fallbackSessionID(frameResult.SessionID, toolSessionID)},
		Content:    []byte(markdown),
	})
	if err != nil {
		return nil, err
	}
	event, err := server.service.AppendEvent(ctx, reporting.BuildMarkdownReportArtifactCreatedAppendRequest(reporting.MarkdownReportArtifactCreatedEventRequest{
		MarkdownReportEventBase: reporting.MarkdownReportEventBase{
			EventID:                      newID("evt"),
			MissionID:                    missionID,
			PendingEventID:               pendingEventID,
			Title:                        title,
			AgentExecutor:                executorName,
			AgentModel:                   agentModel,
			AgentReasoningEffort:         agentReasoningEffort,
			AgentSessionID:               frameResult.SessionID,
			PreviousAgentSessionID:       previousStageSessionID,
			ReturnedAgentSessionID:       returnedSessionID,
			ToolSessionID:                toolSessionID,
			MCPMode:                      mcpMode,
			RigorLevel:                   rigor.level,
			RigorLabel:                   rigor.label,
			ReportMode:                   reportModeLongForm,
			ReportModeLabel:              reportModeLabel(reportModeLongForm),
			ReportSessionPolicy:          reportSessionPolicy,
			ReportSessionPolicySelection: reportSessionPolicySelection,
			PostReportHumanize:           postReportHumanize,
			HumanizeEnabled:              postReportHumanize != "disabled",
			GenerationGuidanceProfile:    generationGuidanceProfile,
			GenerationGuidanceSHA256:     generationGuidanceSHA256,
			SessionChainKind:             sessionChainKind,
			PreReportResearchSessionID:   preReportResearchSessionID,
			ReportPlanSessionID:          reportPlanSessionID,
			ReportSessionID:              frameResult.SessionID,
			ForkSourceAgentSessionID:     forkSourceSessionID,
			PostReportResearchSessionID:  "",
			CompositionStrategy:          "sectional_preserve_markdown",
			DurationMS:                   time.Since(started).Milliseconds(),
			Text:                         "섹션별 보존 조립 방식으로 장문 Markdown 리포트 artifact를 생성했습니다.",
			AgentUsage:                   frameResult.Usage,
			AgentUsageSurface:            "report_frame",
			AgentUsageDurationMS:         frameDurationMS,
			AgentResumed:                 frameResult.Resumed,
			Producer:                     app.Producer{Type: "agent_session", ID: fallbackSessionID(frameResult.SessionID, toolSessionID)},
		},
		Artifact:              artifact,
		PlanEventID:           planEvent.EventID,
		IncludePlanReview:     true,
		PlanReviewRequired:    false,
		PlanReviewState:       "auto_accepted",
		AssemblyStrategy:      "c4_normalized_section_headings",
		SectionCount:          len(sectionArtifactIDs),
		PartCount:             len(partArtifactIDs),
		SectionArtifactIDs:    sectionArtifactIDs,
		PartArtifactIDs:       partArtifactIDs,
		SectionWordCount:      sectionWordTotal,
		FinalWordCount:        finalWordCount,
		PreservationRatio:     preservationRatio,
		IncludeLongFormFields: true,
	}))
	if err != nil {
		return nil, err
	}
	if postReportHumanize == "disabled" {
		return map[string]any{"artifact": artifact, "event": event, "markdown": markdown}, nil
	}
	humanized, err := server.humanizeMarkdownReport(ctx, missionID, reportHumanizeInput{
		Title:             title,
		Markdown:          markdown,
		SourceArtifact:    artifact,
		ExecutorName:      executorName,
		AgentModel:        agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MCPMode:           mcpMode,
		PreviousSessionID: frameResult.SessionID,
		ReportMode:        reportModeLongForm,
		PendingEventID:    pendingEventID,
	}, executor)
	if err != nil {
		return nil, err
	}
	return map[string]any{"artifact": artifact, "event": event, "markdown": markdown, "humanized": humanized}, nil
}

func agentOneTakeMarkdownReportPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile, generationGuidanceProfile string) string {
	guidance := strings.TrimSpace(ReportGenerationGuidance(generationGuidanceProfile))
	if guidance != "" {
		guidance = "\n" + guidance + "\n"
	}
	return fmt.Sprintf(`You are writing a quick Plasma report as a Markdown artifact.

Write a useful Korean Markdown report or article in one pass. This is the fast path: do not create a separate plan artifact first, but still use Plasma MCP research tools when needed.

Mission ID: %s
Report title: %s
Plasma tool binding: use mission_id %s. If a tool requires session_id or producer, use session_id %s and producer {"type":"agent_session","id":"%s"}.

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s
%s

Rules:
- Use MCP/source read tools to inspect original materials when the report needs grounding. Do not assume source bodies are present in this prompt.
- Sources are original materials. Your report is a result artifact, not a source.
- Use prior investigation answers, normal conversation, and controller questions as working memory only. They may guide themes, gaps, structure, and practical implications, but they are not sources and must not be cited.
- Main conclusions must be grounded in original sources or clearly labeled as interpretation, hypothesis, practical implication, rumor, weak signal, or unresolved uncertainty according to the rigor level.
- Prefer a coherent article over a checklist. Include context, comparison, consequences, and tensions where the available material supports them.
- Do not create evidence, claims, confidence updates, source candidates, report blocks, report plans, or report AST JSON.
- Cite source titles, URLs, and human-readable locators when useful. Do not expose internal evidence, claim, or report block IDs as public citations.
- Do not mention this prompt, prompt variant names, experiment labels, tool session IDs, run identifiers, temporary paths, or working directories. Code/source file paths may be cited only when they are original source locators relevant to the user's topic.
- Return only the Markdown report body.`, missionID, title, missionID, toolSessionID, toolSessionID, rigor.level, rigor.label, rigor.description, rigor.instructions, guidance)
}

func agentMarkdownReportPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile, plan agentReportPlan, generationGuidanceProfile string) string {
	planJSON := agentReportPlanJSON(plan)
	guidance := strings.TrimSpace(ReportGenerationGuidance(generationGuidanceProfile))
	if guidance != "" {
		guidance = "\n" + guidance + "\n"
	}
	return fmt.Sprintf(`You are writing a Plasma report as a Markdown artifact.

Write a polished public-facing Korean Markdown report or article, not a thin stitched summary.

Mission ID: %s
Report title: %s
Plasma tool binding: use mission_id %s. If a tool requires session_id or producer, use session_id %s and producer {"type":"agent_session","id":"%s"}.

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s
%s

Rules:
- Use MCP/source read tools to inspect original materials. Do not assume source bodies are present in this prompt.
- Start with plasma.research.outline, then use plasma.research.list, plasma.research.read, plasma.research.grep, and plasma.research.references as needed.
- Distinguish snapshot_only pinned sources, PDF documents, and live_reference local_path sources. PDF reads return extracted text and metadata, not raw PDF bytes. Live local path reads produce source.observed events; when a report sentence depends on them, cite the human locator plus observation_event_id, observed_at, sha256, and git metadata when available.
- Sources are original materials. Your report is a result artifact, not a source.
- Use prior investigation answers, normal conversation, and controller questions as working memory only. They may guide themes, gaps, structure, and practical implications, but they are not sources and must not be cited.
- The visible generation plan below was created in the previous step. Follow it as the coverage contract for this draft. If additional reads show that a planned topic is unsupported or should be changed, reflect that in the report instead of silently dropping it.
- Before writing, re-read the source-backed clusters needed for the planned sections. Do not rely on the plan text alone as a source.
- Main conclusions must be grounded in original sources or clearly labeled as interpretation, hypothesis, practical implication, rumor, weak signal, or unresolved uncertainty according to the rigor level.
- Make the writing rich where the material supports it. Include context, comparison, consequences, and tensions, but do not invent facts.
- Do not create evidence, claims, confidence updates, source candidates, report blocks, or report AST JSON.
- Cite source titles, URLs, and human-readable locators when useful. Do not expose internal evidence, claim, or report block IDs as public citations.
- Do not mention this prompt, prompt variant names, experiment labels, tool session IDs, run identifiers, temporary paths, or working directories. Code/source file paths may be cited only when they are original source locators relevant to the user's topic.
- Return only the Markdown report body.

	Visible generation plan:
	%s`, missionID, title, missionID, toolSessionID, toolSessionID, rigor.level, rigor.label, rigor.description, rigor.instructions, guidance, planJSON)
}

func agentReportPatchPrompt(title string, missionID string, toolSessionID string, pendingEventID string, baseArtifactID string, instruction string, req reporting.PatchRequest) string {
	return AgentReportPatchPrompt(title, missionID, toolSessionID, pendingEventID, baseArtifactID, instruction, req)
}

func AgentReportPatchPrompt(title string, missionID string, toolSessionID string, pendingEventID string, baseArtifactID string, instruction string, req reporting.PatchRequest) string {
	return fmt.Sprintf(`You are patching an existing Plasma Markdown report artifact.

Do not rewrite the full report in your response. Use the report patch MCP tools to read and modify the report in bounded chunks, then finalize the patched report into a new artifact version.

Mission ID: %s
Base report artifact ID: %s
Patched report title: %s
Patch instruction: %s

Plasma tool binding:
- Use mission_id %s.
- Use session_id %s and producer {"type":"agent_session","id":"%s"} for all report patch tool calls.

Required MCP flow:
1. Call %s with base_artifact_id %s, title %s, and the patch instruction. Do not provide patch_id; use the patch_id returned by this call for later patch tool calls.
2. Use %s to inspect the relevant report ranges. Read more chunks when needed; do not assume the whole report is in the prompt.
3. Use %s with small replace, insert_after, or append operations. Prefer exact targeted edits over broad rewrites.
4. Call %s exactly once after edits are complete.

Finalize metadata is server-bound Plasma lineage. Do not infer it from the report text, previous pending events, or tool responses. When the finalize schema asks for these fields, use these exact values:
- pending_event_id: %s
- agent_executor: %s
- agent_model: %s
- agent_reasoning_effort: %s
- mcp_mode: %s
- agent_session_id: %s
- previous_agent_session_id: %s
- returned_agent_session_id: %s
- report_session_id: %s
- fork_source_agent_session_id: %s
- report_session_policy: %s
- report_session_policy_selection: %s
- session_chain_kind: %s

Rules:
- Keep the source and citation structure intact unless the user explicitly asked to change it.
- Preserve useful detail. Do not compress the report just because you are editing it.
- This patch session only exposes report patch tools. If the requested change requires source verification that cannot be done from the current artifact, stop and explain the blocker briefly instead of guessing.
- If you cannot make the requested change safely, do not finalize a fake artifact; explain the blocker briefly.
- After successful finalization, return only a short Korean summary of what changed and the new artifact ID if the tool returned one.`, missionID, baseArtifactID, strconv.Quote(title), strconv.Quote(instruction), missionID, toolSessionID, toolSessionID,
		plasmamcp.ToolReportPatchStart, baseArtifactID, strconv.Quote(title),
		plasmamcp.ToolReportPatchRead,
		plasmamcp.ToolReportPatchApply,
		plasmamcp.ToolReportPatchFinalize,
		pendingEventID,
		req.AgentExecutor,
		req.AgentModel,
		req.AgentReasoningEffort,
		req.MCPMode,
		req.ReportSessionID,
		req.PreviousAgentSessionID,
		req.ReportSessionID,
		req.ReportSessionID,
		req.ForkSourceAgentSessionID,
		req.ReportSessionPolicy,
		req.ReportSessionPolicySelection,
		req.SessionChainKind)
}

func agentSectionalReportPlanPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile) string {
	return fmt.Sprintf(`You are planning a section-first Korean long-form Plasma report.

Do not write the report yet. Return JSON only.
Use Plasma MCP research tools to inspect the mission before planning. Source bodies, evidence arrays, and mission recall JSON are not pasted into this prompt.

Mission ID: %s
Report title: %s
Plasma tool binding: use mission_id %s. If a tool requires session_id or producer, use session_id %s and producer {"type":"agent_session","id":"%s"}.

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s

Planning rules:
- First call plasma.research.outline for the mission overview.
- Use plasma.research.list, plasma.research.grep, plasma.research.read, and plasma.research.references to find the source-backed clusters the report should cover.
- Plan for long-form richness, not a short summary. Include concrete episodes, mechanisms, comparisons, tensions, caveats, weak signals, code/formulas/benchmarks when relevant.
- Group the report into Parts and Sections. A normal mission should usually have 2-5 Parts and 6-14 Sections total. Use fewer only when the mission material is genuinely small.
- Each Section must be specific enough to be drafted independently.
- Sources are original materials. Prior answers, controller questions, plans, generated notes, section drafts, and reports are working memory or results, not sources.
- target_refs should name the source snapshots, evidence records, or saved claims the Section should inspect when available.

Return exactly this JSON shape:
{
  "summary": "what this long-form report will produce",
  "parts": [
    {
      "title": "part title",
      "purpose": "why this part belongs",
      "sections": [
        {
          "title": "section title",
          "purpose": "what this section must explain",
          "target_refs": {"claim_ids": ["clm_..."], "evidence_ids": ["evd_..."], "snapshot_ids": ["src_..."]}
        }
      ]
    }
  ],
  "coverage_notes": ["source clusters and mission turns inspected"],
  "planned_omissions": ["known gaps or intentionally omitted areas"]
}`, missionID, title, missionID, toolSessionID, toolSessionID, rigor.level, rigor.label, rigor.description, rigor.instructions)
}

func agentSectionDraftPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile, plan agentSectionalReportPlan, part agentReportPart, section agentReportSection, partIndex int, sectionIndex int, generationGuidanceProfile string) string {
	guidance := strings.TrimSpace(ReportGenerationGuidance(generationGuidanceProfile))
	if guidance != "" {
		guidance = "\n" + guidance + "\n"
	}
	return fmt.Sprintf(`Draft one section of a Korean long-form Plasma report.

Report title: %s
Mission ID: %s
Part %d: %s
Section %d.%d: %s

Section purpose:
%s

Overall plan:
%s

Plasma tool binding: use mission_id %s. If a tool requires session_id or producer, use session_id %s and producer {"type":"agent_session","id":"%s"}.

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s
%s

Rules:
- Write only this section as Markdown. Do not write the whole report.
- Use MCP/source read tools to inspect original materials relevant to this Section. Do not assume source bodies are present in this prompt.
- Sources are original materials. Prior answers, controller questions, plans, generated notes, section drafts, and reports are working memory or results, not sources.
- Include concrete detail where the sources support it: events, mechanisms, examples, comparisons, tensions, caveats, weak signals, code, formulas, or benchmarks when relevant.
- Preserve uncertainty and competing interpretations instead of flattening them.
- Do not mention prompts, internal run labels, tool session IDs, or temporary implementation details.
- Return only the Markdown body for this section.`, title, missionID, partIndex+1, part.Title, partIndex+1, sectionIndex+1, section.Title, section.Purpose, agentReportAnyJSON(plan), missionID, toolSessionID, toolSessionID, rigor.level, rigor.label, rigor.description, rigor.instructions, guidance)
}

func agentPartAssemblyPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile, plan agentSectionalReportPlan, part agentReportPart, drafts []sectionalReportDraft, partIndex int, generationGuidanceProfile string) string {
	guidance := strings.TrimSpace(ReportGenerationGuidance(generationGuidanceProfile))
	if guidance != "" {
		guidance = "\n" + guidance + "\n"
	}
	return fmt.Sprintf(`Prepare connective tissue for one Part of a Korean long-form Plasma report.

Report title: %s
Mission ID: %s
Part %d: %s

This is not a rewrite task. The Section bodies are immutable and will be mechanically inserted by Plasma. You must not return rewritten Section bodies.

Section inventory:
%s

Overall plan:
%s

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s
%s

Return JSON only:
{
  "intro": "short Markdown introduction for this Part",
  "transitions": [
    {"after_section_index": 1, "markdown": "short transition after section 1"}
  ],
  "closing": "short Markdown closing for this Part"
}

Rules:
- Use Korean.
- Do not include the immutable Section bodies.
- Do not summarize the Section bodies into a replacement overview.
- Transitions are optional, but when useful they should connect adjacent Sections without compressing them.
- Do not mention prompts, experiments, internal run labels, tool session IDs, or temporary implementation details.`, title, missionID, partIndex+1, part.Title, sectionalDraftInventoryJSON(drafts), agentReportAnyJSON(plan), rigor.level, rigor.label, rigor.description, rigor.instructions, guidance)
}

func agentSectionalFramePrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile, plan agentSectionalReportPlan, parts []sectionalReportPartDraft, generationGuidanceProfile string) string {
	guidance := strings.TrimSpace(ReportGenerationGuidance(generationGuidanceProfile))
	if guidance != "" {
		guidance = "\n" + guidance + "\n"
	}
	return fmt.Sprintf(`Write front matter and closing for a Korean long-form Plasma report.

Report title: %s
Mission ID: %s

The Part manuscripts are already written and will be mechanically preserved by Plasma. Do not rewrite them.

Part inventory:
%s

Overall plan:
%s

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s
%s

Return JSON only:
{
  "front_matter": "Markdown title, introduction, reading guide, and compact table of contents",
  "closing": "Markdown conclusion that synthesizes tensions, supported conclusions, remaining uncertainty, and useful next checks"
}

Rules:
- Use Korean.
- Do not include or rewrite the Part manuscripts.
- Do not mention prompts, experiments, internal run labels, tool session IDs, or temporary implementation details.`, title, missionID, sectionalPartInventoryJSON(parts), agentReportAnyJSON(plan), rigor.level, rigor.label, rigor.description, rigor.instructions, guidance)
}

func normalizeReportMode(mode string) (string, error) {
	return reporting.NormalizeMode(mode)
}

func normalizeReportSessionPolicy(policy string) (string, error) {
	return reporting.NormalizeSessionPolicy(policy)
}

func (server *Server) selectReportSessionPolicy(ctx context.Context, missionID string, executorName string, reportMode string, requestedPolicy string, executor AgentExecutor) (string, string, error) {
	_, canFork := executor.(AgentSessionForker)
	_, canCheckFork := executor.(AgentSessionForkReadiness)
	preReportSessionID := ""
	forkReady := false
	if canFork {
		preReportSessionID = strings.TrimSpace(server.latestAgentSessionID(ctx, missionID, executorName))
		forkReady = canCheckFork && AgentSessionForkReady(ctx, executor, preReportSessionID)
	}
	return reporting.SelectSessionPolicy(reporting.SessionPolicySelectionInput{
		RequestedPolicy:             requestedPolicy,
		ReportMode:                  reportMode,
		CanForkSession:              canFork,
		HasPreReportResearchSession: preReportSessionID != "",
		ForkReady:                   forkReady,
	})
}

func (server *Server) validateReportSessionPolicy(ctx context.Context, missionID string, executorName string, reportMode string, policy string, executor AgentExecutor, requireReady bool) error {
	if executor == nil {
		return fmt.Errorf("%w: report generation requires an agent executor", app.ErrInvalidInput)
	}
	_, canFork := executor.(AgentSessionForker)
	_, canCheckFork := executor.(AgentSessionForkReadiness)
	preReportSessionID := strings.TrimSpace(server.latestAgentSessionID(ctx, missionID, executorName))
	return reporting.ValidateSessionPolicy(policy, reportMode, canFork, !requireReady || preReportSessionID != "", !requireReady || (canCheckFork && AgentSessionForkReady(ctx, executor, preReportSessionID)))
}

func AgentSessionForkReady(ctx context.Context, executor AgentExecutor, sourceSessionID string) bool {
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	if sourceSessionID == "" {
		return false
	}
	readiness, ok := executor.(AgentSessionForkReadiness)
	if !ok {
		return false
	}
	return readiness.CheckForkSession(ctx, sourceSessionID) == nil
}

func reportModeLabel(mode string) string {
	return reporting.ModeLabel(mode)
}

func normalizeReportRigorProfile(level string) (reportRigorProfile, error) {
	normalized := strings.TrimSpace(level)
	if normalized == "" {
		normalized = defaultReportRigorLevel
	}
	switch normalized {
	case "loose":
		normalized = "exploratory"
	case "normal":
		normalized = "balanced"
	case "rigorous":
		normalized = "strict"
	}
	profile, ok := reportRigorProfiles[normalized]
	if !ok {
		return reportRigorProfile{}, fmt.Errorf("%w: unsupported report rigor level", app.ErrInvalidInput)
	}
	return profile, nil
}

func agentReportPlanPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile) string {
	return fmt.Sprintf(`You are planning a Plasma report before writing it.

Create a user-visible Korean report generation plan for the current mission.
Do not write the article yet. Return JSON only.
Use Plasma MCP research tools to inspect the mission before planning. Source bodies, evidence arrays, and mission recall JSON are not pasted into this prompt. PDF reads return extracted text and metadata, not raw PDF bytes.
Live local_path sources are mutable origins. Use read tools to create source.observed events before relying on them, and plan to cite observation metadata rather than only source IDs.

Evidence rigor:
- Level: %s (%s)
- Meaning: %s
%s

Planning workflow:
- First call plasma.research.outline for the mission overview.
- Use plasma.research.list and plasma.research.grep to find relevant source snapshots, evidence records, saved claims, prior report blocks, prior user turns, agent responses, controller questions, and unresolved questions.
- Use plasma.research.read for the objects or source chunks you intend to rely on. If a read is truncated, continue with next_offset when that material matters.
- For PDF sources, rely on Plasma's extracted text reads and extraction metadata. Do not ask for raw PDF bytes in the prompt.
- For live_reference local_path sources, final report support should come from explicit read observations with observation_event_id, observed_at, relative_path, sha256, and git metadata when available.
- Use plasma.research.references when you need to understand source-evidence-claim-report links.
- General research may inspect proposed, pending, or rejected material as context, but the plan's target_refs should name only approved records you expect the final report to rely on.
- Treat repeated or explicit user questions as coverage signals. If the user steered the mission toward a person, event, comparison, dispute, or source cluster, include it in sections or planned_omissions after checking source support.
- Plan for richness. Include facts, interpretations, reactions, rumors, disputes, code, formulas, benchmarks, and open questions when the mission and rigor level allow them.
- The plan is visible to the user. Be concrete enough that the user can tell what the report will cover and what evidence clusters it will use.

Report title requested by the user interface:
%s

Plasma tool binding: use mission_id %s. If a tool requires session_id or producer, use session_id %s and producer {"type":"agent_session","id":"%s"}.

Return exactly this JSON shape:
{
  "summary": "what this report will try to produce",
  "sections": [
    {
      "title": "planned section title",
      "purpose": "why this section belongs in the report",
      "target_refs": {"claim_ids": ["clm_..."], "evidence_ids": ["evd_..."], "snapshot_ids": ["src_..."]}
    }
  ],
  "coverage_notes": ["what source or evidence clusters were inspected and will be used"],
  "planned_omissions": ["known gaps, weak areas, or items intentionally left out"]
}
`, rigor.level, rigor.label, rigor.description, rigor.instructions, strings.TrimSpace(title), strings.TrimSpace(missionID), toolSessionID, toolSessionID)
}

func agentReportPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile, plan agentReportPlan) string {
	planJSON := agentReportPlanJSON(plan)
	return fmt.Sprintf(`You are the Plasma report writer.

Write a polished Korean report or article for the current mission.
The canonical output must be a structured AST JSON object. Markdown and HTML will be rendered from this AST later.
Do not output Markdown fences, commentary, or prose outside the JSON object.
Do not invent source references. Use Plasma MCP research tools to inspect pinned sources, live local_path observations, evidence, saved claims, questions, and report blocks when needed. Source bodies, evidence arrays, and mission recall JSON are not pasted into this prompt.

Evidence rigor:
- Level: %s (%s)
- Meaning: %s
%s

General evidence handling:
- First call plasma.research.outline for the mission overview.
- Use plasma.research.list and plasma.research.grep to find candidate source snapshots, evidence records, claims, questions, and report blocks.
- Use plasma.research.read to confirm saved knowledge, evidence details, source chunks, and long payloads with offset/max_bytes.
- For PDF sources, rely on extracted text and extraction metadata returned by Plasma tools.
- For live_reference local_path sources, use read observations rather than source IDs alone. When a sentence depends on mutable local material, include the relevant human locator and observation_event_id/observed_at/sha256/git details in the text or refs context available to the renderer.
- Use plasma.research.references to verify source-evidence-claim-report links before relying on them.
- Treat grep matches as candidates only. A final report sentence that depends on mission material must be grounded in saved evidence, saved claims, or explicit source reads.
- Evidence can include facts, observations, interpretations, reactions, rumors, controversies, market signals, code, formulas, benchmarks, and open questions.
- Treat evidence_type and confidence as writing constraints, not as obstacles. The report should become richer without flattening weak signals into facts.
- If a sentence depends on a specific saved claim, evidence record, or source snapshot, include the relevant refs in that AST block.
- References are rendered as visible footnotes in Markdown and HTML exports. Include refs for every source-backed paragraph, list, and quote.
- You may inspect proposed, pending, or rejected material while researching, but final AST refs must only contain approved claim_ids and approved evidence_ids that are inside the report scope.
- If unapproved material is useful background, either replace it with approved refs that support the same point or describe it clearly as an unapproved candidate without using its claim_id or evidence_id as a final ref.
- Before returning the AST, check every refs/source_refs object. Any proposed, pending, rejected, missing, or out-of-scope claim_id/evidence_id will be rejected and you will need to repair the AST.

User-visible generation plan created in the previous step:
%s

Follow the plan unless your additional reads reveal that a section should be changed. If you change it, keep the final article coherent and evidence-grounded.

Report title requested by the user interface:
%s

Plasma tool binding: use mission_id %s. If a tool requires session_id or producer, use session_id %s and producer {"type":"agent_session","id":"%s"}.

Return exactly this JSON shape:
{
  "title": "short report title",
  "summary": "executive summary paragraph",
  "blocks": [
    {"type": "heading", "level": 2, "text": "section title"},
    {"type": "paragraph", "text": "article paragraph", "refs": {"claim_ids": ["clm_..."], "evidence_ids": ["evd_..."], "snapshot_ids": ["src_..."]}},
    {"type": "bullet_list", "items": ["item"], "refs": {"evidence_ids": ["evd_..."]}},
    {"type": "quote", "text": "short callout"}
  ]
}

Allowed block types are heading, paragraph, bullet_list, and quote.
Use refs only when a block depends on specific saved knowledge or evidence. Omit refs for narrative transitions.
Write a complete, readable article that covers the planned evidence clusters. Synthesize the material, but do not shrink away planned source-backed substance.
`, rigor.level, rigor.label, rigor.description, rigor.instructions, planJSON, strings.TrimSpace(title), strings.TrimSpace(missionID), toolSessionID, toolSessionID)
}

func parseAgentReportPlan(text string) (agentReportPlan, error) {
	raw, err := extractAgentJSONObject(text)
	if err != nil {
		return agentReportPlan{}, fmt.Errorf("%w: report planning agent did not return JSON", app.ErrInvalidInput)
	}
	var plan agentReportPlan
	decoder := json.NewDecoder(strings.NewReader(raw))
	if err := decoder.Decode(&plan); err != nil {
		return agentReportPlan{}, fmt.Errorf("%w: invalid report plan JSON: %v", app.ErrInvalidInput, err)
	}
	if strings.TrimSpace(plan.Summary) == "" && len(plan.Sections) == 0 {
		return agentReportPlan{}, fmt.Errorf("%w: report plan is empty", app.ErrInvalidInput)
	}
	return plan, nil
}

func parseAgentSectionalReportPlan(text string) (agentSectionalReportPlan, error) {
	raw, err := extractAgentJSONObject(text)
	if err != nil {
		return agentSectionalReportPlan{}, fmt.Errorf("%w: sectional report planning agent did not return JSON", app.ErrInvalidInput)
	}
	var plan agentSectionalReportPlan
	decoder := json.NewDecoder(strings.NewReader(raw))
	if err := decoder.Decode(&plan); err != nil {
		return agentSectionalReportPlan{}, fmt.Errorf("%w: invalid sectional report plan JSON: %v", app.ErrInvalidInput, err)
	}
	plan.Summary = strings.TrimSpace(plan.Summary)
	plan.CoverageNotes = limitNonEmptyStrings(plan.CoverageNotes, 24)
	plan.PlannedOmissions = limitNonEmptyStrings(plan.PlannedOmissions, 24)
	cleanParts := make([]agentReportPart, 0, len(plan.Parts))
	for _, part := range plan.Parts {
		part.Title = strings.TrimSpace(part.Title)
		part.Purpose = strings.TrimSpace(part.Purpose)
		cleanSections := make([]agentReportSection, 0, len(part.Sections))
		for _, section := range part.Sections {
			section.Title = strings.TrimSpace(section.Title)
			section.Purpose = strings.TrimSpace(section.Purpose)
			if section.Title == "" && section.Purpose == "" {
				continue
			}
			if section.Title == "" {
				section.Title = section.Purpose
			}
			cleanSections = append(cleanSections, section)
		}
		if part.Title == "" && len(cleanSections) == 0 {
			continue
		}
		if part.Title == "" {
			part.Title = fmt.Sprintf("Part %d", len(cleanParts)+1)
		}
		part.Sections = cleanSections
		if len(part.Sections) == 0 {
			part.Sections = []agentReportSection{{Title: part.Title, Purpose: firstNonEmpty(part.Purpose, part.Title)}}
		}
		cleanParts = append(cleanParts, part)
	}
	plan.Parts = cleanParts
	if strings.TrimSpace(plan.Summary) == "" && len(plan.Parts) == 0 {
		return agentSectionalReportPlan{}, fmt.Errorf("%w: sectional report plan is empty", app.ErrInvalidInput)
	}
	if len(plan.Parts) == 0 {
		plan.Parts = []agentReportPart{{
			Title: firstNonEmpty(plan.Summary, "장문 리포트"),
			Sections: []agentReportSection{{
				Title:   firstNonEmpty(plan.Summary, "핵심 내용"),
				Purpose: firstNonEmpty(plan.Summary, "미션 내용을 장문으로 정리한다."),
			}},
		}}
	}
	return plan, nil
}

func parseAgentPartAssembly(text string) (agentPartAssembly, error) {
	raw, err := extractAgentJSONObject(text)
	if err != nil {
		return agentPartAssembly{}, fmt.Errorf("%w: part assembly agent did not return JSON", app.ErrInvalidInput)
	}
	var assembly agentPartAssembly
	decoder := json.NewDecoder(strings.NewReader(raw))
	if err := decoder.Decode(&assembly); err != nil {
		return agentPartAssembly{}, fmt.Errorf("%w: invalid part assembly JSON: %v", app.ErrInvalidInput, err)
	}
	assembly.Intro = strings.TrimSpace(assembly.Intro)
	assembly.Closing = strings.TrimSpace(assembly.Closing)
	transitions := make([]agentPartTransition, 0, len(assembly.Transitions))
	for _, transition := range assembly.Transitions {
		transition.Markdown = strings.TrimSpace(transition.Markdown)
		if transition.AfterSectionIndex <= 0 || transition.Markdown == "" {
			continue
		}
		transitions = append(transitions, transition)
	}
	assembly.Transitions = transitions
	return assembly, nil
}

func parseAgentSectionalFrame(text string) (agentSectionalFrame, error) {
	raw, err := extractAgentJSONObject(text)
	if err != nil {
		return agentSectionalFrame{}, fmt.Errorf("%w: report frame agent did not return JSON", app.ErrInvalidInput)
	}
	var frame agentSectionalFrame
	decoder := json.NewDecoder(strings.NewReader(raw))
	if err := decoder.Decode(&frame); err != nil {
		return agentSectionalFrame{}, fmt.Errorf("%w: invalid report frame JSON: %v", app.ErrInvalidInput, err)
	}
	frame.FrontMatter = strings.TrimSpace(frame.FrontMatter)
	frame.Closing = strings.TrimSpace(frame.Closing)
	return frame, nil
}

func parseAgentReportAST(text string) (agentReportAST, error) {
	raw, err := extractAgentJSONObject(text)
	if err != nil {
		return agentReportAST{}, fmt.Errorf("%w: report agent did not return JSON AST", app.ErrInvalidInput)
	}
	var ast agentReportAST
	decoder := json.NewDecoder(strings.NewReader(raw))
	if err := decoder.Decode(&ast); err != nil {
		return agentReportAST{}, fmt.Errorf("%w: invalid report AST JSON: %v", app.ErrInvalidInput, err)
	}
	if strings.TrimSpace(ast.Title) == "" && strings.TrimSpace(ast.Summary) == "" && len(ast.Blocks) == 0 {
		return agentReportAST{}, fmt.Errorf("%w: report AST is empty", app.ErrInvalidInput)
	}
	return ast, nil
}

func extractAgentJSONObject(text string) (string, error) {
	raw := strings.TrimSpace(text)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "```"))
	if strings.HasPrefix(raw, "{") {
		return raw, nil
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return "", fmt.Errorf("%w: JSON object not found", app.ErrInvalidInput)
	}
	return raw[start : end+1], nil
}

func agentReportPlanJSON(plan agentReportPlan) string {
	encoded, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func agentReportASTJSON(ast agentReportAST) string {
	encoded, err := json.MarshalIndent(ast, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

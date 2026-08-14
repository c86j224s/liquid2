package web

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	htmlpkg "html"
	"log"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportpatch"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow"
)

func (server *Server) handleMissionReports(w http.ResponseWriter, r *http.Request, missionID string, rest []string) {
	if len(rest) == 1 && rest[0] == "cancel" {
		server.handleCancelMissionReport(w, r, missionID)
		return
	}
	if len(rest) == 1 && rest[0] == "retry" {
		server.handleRetryMissionReport(w, r, missionID)
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

func (server *Server) handleRetryMissionReport(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req reportRetryRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	unlock := server.reports.lock(missionID)
	defer unlock()
	pending, err := server.service.RequestReportRetry(r.Context(), app.ReportRetryRequest{
		EventID: newID("evt"), MissionID: missionID, FailedPendingEventID: req.FailedPendingEventID,
		Strategy: req.Strategy, RetryRequestID: req.RetryRequestID, Producer: app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	if err := server.resumeReportDraftWorker(r.Context(), missionID, pending); err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"pending_event": pending, "status": "pending"})
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
	if len(rest) >= 2 && rest[1] == "redpen" {
		server.handleReportRedpenRoute(w, r, missionID, rest)
		return
	}
	if len(rest) == 2 && rest[1] == "report_delete_preview" {
		server.handleReportDeletePreview(w, r, missionID, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "report" {
		server.handleReportDelete(w, r, missionID, rest[0])
		return
	}
	if len(rest) != 1 && !(len(rest) == 2 && (rest[1] == "download" || rest[1] == "preview" || rest[1] == "html_export" || rest[1] == "designed_html_export" || rest[1] == "humanized_markdown_export")) {
		http.NotFound(w, r)
		return
	}
	if len(rest) == 2 && rest[1] == "humanized_markdown_export" {
		// deprecated된 UI 시작 경로다. 과거 artifact/event와 직접 API 호환성을 위해
		// route만 유지한다.
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
	if ok, err := server.isReadableArtifact(r.Context(), missionID, artifact.ArtifactID); err != nil {
		writeAppError(w, err)
		return
	} else if !ok {
		writeError(w, http.StatusNotFound, "artifact not found")
		return
	}
	if len(rest) == 2 && rest[1] == "download" {
		writeRawArtifactDownload(w, artifact)
		return
	}
	if len(rest) == 2 && rest[1] == "preview" {
		writeRawArtifactHTMLPreview(w, artifact)
		return
	}
	writeRawArtifactFullPreview(w, artifact)
}

func (server *Server) handleReportDeletePreview(w http.ResponseWriter, r *http.Request, missionID string, artifactID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	activePendingID, _ := server.runningReports.PendingEventID(missionID)
	preview, err := server.service.PreviewReportDelete(r.Context(), app.ReportDeletePreviewRequest{
		MissionID: missionID, ArtifactID: artifactID, ActivePendingEventID: activePendingID,
	})
	if err != nil {
		writeReportArtifactRouteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (server *Server) handleReportDelete(w http.ResponseWriter, r *http.Request, missionID string, artifactID string) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req reportDeleteRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	unlock := server.reports.lock(missionID)
	defer unlock()
	activePendingID, _ := server.runningReports.PendingEventID(missionID)
	result, err := server.service.DeleteReport(r.Context(), app.ReportDeleteRequest{
		MissionID:            missionID,
		ArtifactID:           artifactID,
		ConfirmArtifactID:    req.ConfirmArtifactID,
		ExpectedRevision:     req.ExpectedRevision,
		DeleteFactsHash:      req.DeleteFactsHash,
		ActivePendingEventID: activePendingID,
		Producer:             app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeReportArtifactRouteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeReportArtifactRouteError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "artifact not found")
		return
	}
	writeAppError(w, err)
}

func (server *Server) handleReportArtifactHTMLExport(w http.ResponseWriter, r *http.Request, missionID string, artifactID string) {
	var req reportArtifactHTMLExportRequest
	if !decodeOptionalJSON(w, r, &req) {
		return
	}
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
	response := map[string]any{
		"artifact":        rawArtifactMetadata(result.Artifact),
		"source_artifact": rawArtifactMetadata(sourceArtifact),
		"event":           result.Event,
		"preview_url":     rawArtifactPreviewPath(missionID, result.Artifact.ArtifactID),
	}
	if req.IncludeContent == nil || *req.IncludeContent {
		response["content"] = string(result.Artifact.Content)
	}
	writeJSON(w, http.StatusOK, response)
}

type reportArtifactHTMLExportRequest struct {
	IncludeContent *bool `json:"include_content"`
}

func rawArtifactPreviewPath(missionID string, artifactID string) string {
	return "/api/missions/" + url.PathEscape(missionID) + "/artifacts/" + url.PathEscape(artifactID) + "/preview"
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

// ReportPatchSessionSelection는 patch 실행에 사용할 report session과 fork 출처 선택 결과다.
type ReportPatchSessionSelection = reportpatch.PatchSessionSelection

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

// SelectReportPatchSession는 patch 요청에 사용할 report session과 fork 출처를 선택한다.
func SelectReportPatchSession(ctx context.Context, executor AgentExecutor, sourceSessionID string, requestedPolicy string) (ReportPatchSessionSelection, error) {
	return reportpatch.SelectSession(ctx, executor, sourceSessionID, requestedPolicy)
}

func (server *Server) exportMarkdownArtifactAsHTML(ctx context.Context, missionID string, sourceArtifact app.RawArtifact) (app.ReportExportResult, error) {
	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	refetched, err := server.reportArtifact(ctx, missionID, sourceArtifact.ArtifactID)
	if err != nil {
		return app.ReportExportResult{}, err
	}
	sourceArtifact = refetched
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
	artifact, event, err := server.service.CreateRawArtifactWithEvent(ctx, app.CreateRawArtifactRequest{
		ArtifactID: newID("art"),
		MissionID:  missionID,
		MediaType:  "text/html; charset=utf-8",
		Filename:   markdownReportHTMLFilename(sourceArtifact),
		Producer:   app.Producer{Type: "plasma", ID: "html-export"},
		Content:    content,
	}, func(artifact app.RawArtifact) app.AppendEventRequest {
		return reportexecution.BuildSelfContainedHTMLExportAppendRequest(reportexecution.SelfContainedHTMLExportEventRequest{
			EventID:          newID("evt"),
			MissionID:        missionID,
			SourceArtifactID: sourceArtifact.ArtifactID,
			Artifact:         artifact,
			RendererVersion:  selfContainedReportRendererVersion,
			Producer:         app.Producer{Type: "plasma", ID: "html-export"},
		})
	})
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
			RendererVersion  string `json:"renderer_version"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.Kind) != reportexecution.ExportKindSelfContainedHTML ||
			strings.TrimSpace(payload.SourceArtifactID) != sourceArtifactID ||
			strings.TrimSpace(payload.Target) != reportexecution.ExportTargetSelfContainedHTML ||
			strings.TrimSpace(payload.RendererVersion) != selfContainedReportRendererVersion {
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
	sourceArtifact, err = server.reportArtifact(ctx, missionID, sourceArtifact.ArtifactID)
	if err != nil {
		return nil, false, err
	}
	if !isMarkdownMediaType(sourceArtifact.MediaType) {
		return nil, false, fmt.Errorf("%w: designed HTML export requires a Markdown report artifact", app.ErrInvalidInput)
	}
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
	pendingEvent, err := server.reportRunner().StartDesign(ctx, missionID, reportexecution.DesignRequest{
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
	content, err := server.renderDesignedReportHTML(sourceArtifact, model, images, notes)
	if err != nil {
		return app.ReportExportResult{}, err
	}
	producer := app.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)}
	_, artifact, event, closed, err := server.service.CreateDesignedReportHTMLExportIfOpen(ctx, missionID, pendingEventID, app.CreateRawArtifactRequest{
		ArtifactID: newID("art"),
		MissionID:  missionID,
		MediaType:  "application/json; charset=utf-8",
		Filename:   safeFilename(title+" content model", ".json"),
		Producer:   producer,
		Content:    modelJSON,
	}, app.CreateRawArtifactRequest{
		ArtifactID: newID("art"),
		MissionID:  missionID,
		MediaType:  "text/html; charset=utf-8",
		Filename:   safeFilename(title+" designed", ".html"),
		Producer:   producer,
		Content:    content,
	}, func(modelArtifact app.RawArtifact, artifact app.RawArtifact) app.AppendEventRequest {
		return reportexecution.BuildDesignedHTMLExportAppendRequest(reportexecution.DesignedHTMLExportEventRequest{
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
			Producer:               producer,
		})
	})
	if err != nil {
		return app.ReportExportResult{}, err
	}
	if !closed {
		return app.ReportExportResult{}, fmt.Errorf("%w: designed HTML report operation is already closed", app.ErrConflict)
	}
	return app.ReportExportResult{Artifact: artifact, Event: event}, nil
}

func (server *Server) reconcileStaleDesignedReportExports(ctx context.Context, missionID string) error {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return err
	}
	completed := reportexecution.CompletedPendingEventIDs(events)
	for _, event := range events {
		if event.EventType != "report.design.pending" {
			continue
		}
		if _, ok := completed[event.EventID]; ok || server.runningReports.Owns(missionID, event.EventID) {
			continue
		}
		return server.reportRunner().ResumeDesign(ctx, missionID, event)
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
		if strings.TrimSpace(payload.Kind) != reportexecution.ExportKindDesignedHTML ||
			strings.TrimSpace(payload.SourceArtifactID) != sourceArtifactID ||
			strings.TrimSpace(payload.Target) != reportexecution.ExportTargetDesignedHTML ||
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
	images, notes, err := server.inlineReportImages(ctx, missionID)
	if err != nil {
		return nil, err
	}
	title := reportArtifactTitle(sourceArtifact)
	mathHead, err := selfContainedMathHead()
	if err != nil {
		return nil, err
	}
	mermaidHead, err := selfContainedMermaidHead()
	if err != nil {
		return nil, err
	}
	mathScripts, err := selfContainedMarkdownScripts()
	if err != nil {
		return nil, err
	}
	wordCount := len(strings.Fields(string(sourceArtifact.Content)))
	markdownJSON, err := json.Marshal(string(sourceArtifact.Content))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString("<!doctype html>\n<html lang=\"ko\">\n<head>\n<meta charset=\"utf-8\">\n")
	out.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	out.WriteString("<title>" + htmlpkg.EscapeString(title) + "</title>\n")
	out.WriteString(selfContainedReportCSS())
	out.WriteString(mathHead)
	out.WriteString(mermaidHead)
	out.WriteString(selfContainedBasicMermaidCSS())
	out.WriteString("</head>\n<body>\n")
	out.WriteString("<header class=\"hero\"><div><p class=\"eyebrow\">Plasma Report</p><h1>" + htmlpkg.EscapeString(title) + "</h1><p class=\"sub\">Markdown report artifact에서 파생한 self-contained interactive HTML입니다. 이미지는 가능한 경우 원본 source artifact를 data URI로 포함했습니다.</p></div><button id=\"themeToggle\" type=\"button\" hidden aria-pressed=\"false\" aria-label=\"다크 모드 켜기\">다크 모드</button></header>\n")
	out.WriteString("<main class=\"layout\">\n")
	out.WriteString("<aside class=\"rail\"><div class=\"metric\"><span>본문 단어</span><strong>" + strconv.Itoa(wordCount) + "</strong></div><div class=\"metric\"><span>포함 이미지</span><strong>" + strconv.Itoa(len(images)) + "</strong></div><div class=\"metric\"><span>원본 artifact</span><code>" + htmlpkg.EscapeString(sourceArtifact.ArtifactID) + "</code></div><nav><a href=\"#report-body\">본문</a><a href=\"#media-gallery\">미디어</a><a href=\"#export-notes\">생성 노트</a></nav></aside>\n")
	out.WriteString("<article id=\"report-body\" class=\"report-body\">\n")
	out.WriteString("<pre class=\"report-markdown-raw\">" + htmlpkg.EscapeString(string(sourceArtifact.Content)) + "</pre>")
	out.WriteString("</article>\n")
	out.WriteString("<script id=\"report-markdown\" type=\"application/json\">" + string(markdownJSON) + "</script>\n")
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
	out.WriteString(selfContainedReportThemeScript())
	out.WriteString(mathScripts)
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

func (server *Server) isReportArtifact(ctx context.Context, missionID string, artifactID string) (bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return false, err
	}
	for _, event := range events {
		if event.EventType != "report.artifact.created" && event.EventType != "report.artifact.exported" && event.EventType != app.ReportRedpenSavedEvent {
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
		if strings.TrimSpace(payload.ArtifactID) == artifactID && (kind == "markdown_report_artifact" || kind == reportexecution.ExportKindSelfContainedHTML || kind == reportexecution.ExportKindDesignedHTML || kind == reportexecution.ExportKindHumanizedMarkdown || kind == app.ReportRedpenArtifactKind) {
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
	_, ok := reportexecution.CompletedPendingEventIDs(events)[strings.TrimSpace(pendingEventID)]
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
	executionStrategy, err := normalizeReportExecutionStrategy(req.ExecutionStrategy, reportMode)
	if err != nil {
		return nil, err
	}
	req.Title = title
	req.AgentExecutor = executorName
	req.MCPMode = mcpMode
	req.RigorLevel = rigor.level
	req.ReportMode = reportMode
	req.ExecutionStrategy = executionStrategy
	guidanceProfile, guidanceSHA, err := reportprompt.SelectReportGenerationGuidanceForMode(reportMode, req.GenerationGuidanceProfile)
	if err != nil {
		return nil, err
	}
	postReportHumanize := reportprompt.NormalizePostReportHumanize(req.PostReportHumanize)

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
	if server.hasOpenReportDraft(ctx, missionID) {
		return nil, errReportDraftRunning
	}
	if server.hasOpenAgentTurn(ctx, missionID) {
		return nil, fmt.Errorf("%w: agent turn is already running for this mission", app.ErrInvalidInput)
	}
	if active := server.activeWorkflowRun(ctx, missionID); active != nil {
		return nil, fmt.Errorf("%w: workflow %s is %s for this mission", app.ErrInvalidInput, active.WorkflowRunID, active.Status)
	}
	selection, err := server.resolveReportModelSelection(ctx, missionID, req)
	if err != nil {
		return nil, err
	}
	req.AgentModel = selection.Model
	req.AgentReasoningEffort = selection.ReasoningEffort
	req.AgentSelectionSource = selection.Source
	executor := server.agentExecutor(executorName)
	reportSessionPolicy, reportSessionPolicySelection, err := server.selectReportSessionPolicy(ctx, missionID, executorName, reportMode, strings.TrimSpace(req.ReportSessionPolicy), executor)
	if err != nil {
		return nil, err
	}
	req.ReportSessionPolicy = reportSessionPolicy
	req.ReportSessionPolicySelection = reportSessionPolicySelection
	pendingEvent, err := server.reportRunner().StartDraft(ctx, missionID, reportexecution.DraftRequest{
		Title:                        title,
		DirectionHint:                req.DirectionHint,
		ExecutionStrategy:            storedReportExecutionStrategy(req.ExecutionStrategy),
		AgentExecutor:                executorName,
		AgentModel:                   req.AgentModel,
		AgentReasoningEffort:         req.AgentReasoningEffort,
		AgentSelectionSource:         req.AgentSelectionSource,
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
	baseArtifact, err = server.reportArtifact(ctx, missionID, baseArtifactID)
	if err != nil {
		return nil, err
	}
	if !isMarkdownMediaType(baseArtifact.MediaType) {
		return nil, fmt.Errorf("%w: report patch requires a Markdown report artifact", app.ErrInvalidInput)
	}
	info, err = server.reportArtifactSessionInfo(ctx, missionID, baseArtifactID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.AgentExecutor) == "" {
		executorName = info.AgentExecutor
		executorName, err = normalizeAgentExecutorName(executorName)
		if err != nil {
			return nil, err
		}
	}
	if baseExecutor := strings.TrimSpace(info.AgentExecutor); baseExecutor != "" && baseExecutor != executorName {
		return nil, fmt.Errorf("%w: report patch must use the original report executor %q", app.ErrInvalidInput, baseExecutor)
	}
	executor = server.agentExecutor(executorName)
	if executor == nil {
		return nil, fmt.Errorf("%w: report patch requires an agent executor", app.ErrInvalidInput)
	}
	agentModel = strings.TrimSpace(req.AgentModel)
	if agentModel == "" {
		agentModel = strings.TrimSpace(info.AgentModel)
	}
	agentReasoningEffort = strings.TrimSpace(req.AgentReasoningEffort)
	if agentReasoningEffort == "" {
		agentReasoningEffort = strings.TrimSpace(info.AgentReasoningEffort)
	}
	agentModel, agentReasoningEffort, err = resolveAgentSettings(executorName, agentModel, agentReasoningEffort, strings.TrimSpace(info.ReportSessionID))
	if err != nil {
		return nil, err
	}
	title = strings.TrimSpace(req.Title)
	if title == "" {
		title = firstNonEmpty(info.Title+" 수정본", reportArtifactTitle(baseArtifact)+" 수정본", "Patched report")
	}
	if err := server.validateMissionAgentExecutor(ctx, missionID, executorName); err != nil {
		return nil, err
	}
	if err := server.reconcileStaleAgentTurn(ctx, missionID); err != nil {
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
	pendingEvent, err := server.reportRunner().StartPatch(ctx, missionID, reportexecution.PatchRequest{
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
	sourceArtifact, err = server.reportArtifact(ctx, missionID, sourceArtifact.ArtifactID)
	if err != nil {
		return nil, err
	}
	if !isMarkdownMediaType(sourceArtifact.MediaType) {
		return nil, fmt.Errorf("%w: H5 humanize requires a Markdown report artifact", app.ErrInvalidInput)
	}
	info, err = server.reportArtifactSessionInfo(ctx, missionID, sourceArtifact.ArtifactID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.AgentExecutor) == "" {
		executorName = info.AgentExecutor
		executorName, err = normalizeAgentExecutorName(executorName)
		if err != nil {
			return nil, err
		}
	}
	if baseExecutor := strings.TrimSpace(info.AgentExecutor); baseExecutor != "" && baseExecutor != executorName {
		return nil, fmt.Errorf("%w: H5 humanize must use the original report executor %q", app.ErrInvalidInput, baseExecutor)
	}
	executor = server.agentExecutor(executorName)
	if executor == nil {
		return nil, fmt.Errorf("%w: H5 humanize requires an agent executor", app.ErrInvalidInput)
	}
	reportSessionID = strings.TrimSpace(info.ReportSessionID)
	if reportSessionID == "" {
		return nil, fmt.Errorf("%w: H5 humanize requires a report session", app.ErrInvalidInput)
	}
	agentModel = strings.TrimSpace(req.AgentModel)
	if agentModel == "" {
		agentModel = firstNonEmpty(info.AgentModel, server.latestAgentSessionModel(ctx, missionID, executorName))
	}
	agentReasoningEffort = strings.TrimSpace(req.AgentReasoningEffort)
	if agentReasoningEffort == "" {
		agentReasoningEffort = firstNonEmpty(info.AgentReasoningEffort, server.latestAgentReasoningEffort(ctx, missionID, executorName))
	}
	agentModel, agentReasoningEffort, err = resolveAgentSettings(executorName, agentModel, agentReasoningEffort, reportSessionID)
	if err != nil {
		return nil, err
	}
	title = firstNonEmpty(req.Title, info.Title, reportArtifactTitle(sourceArtifact))
	reportMode = firstNonEmpty(info.ReportMode, defaultReportMode)
	if err := server.validateMissionAgentExecutor(ctx, missionID, executorName); err != nil {
		return nil, err
	}
	if err := server.reconcileStaleAgentTurn(ctx, missionID); err != nil {
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
	pendingEvent, err := server.reportRunner().StartHumanize(ctx, missionID, reportexecution.HumanizeRequest{
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

func (server *Server) reportRunner() reportexecution.Runner {
	return reportexecution.Runner{
		Service:  server.service,
		InFlight: &server.runningReports,
		NewID:    newID,
		GenerateDraft: func(ctx context.Context, missionID string, req reportexecution.DraftRequest, pendingEventID string) error {
			_, err := server.createReportDraft(ctx, missionID, reportDraftRequest{
				Title:                        req.Title,
				DirectionHint:                req.DirectionHint,
				ExecutionStrategy:            req.ExecutionStrategy,
				AgentExecutor:                req.AgentExecutor,
				AgentModel:                   req.AgentModel,
				AgentReasoningEffort:         req.AgentReasoningEffort,
				AgentSelectionSource:         req.AgentSelectionSource,
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
		GenerateDesign: func(ctx context.Context, missionID string, req reportexecution.DesignRequest, pendingEventID string) error {
			_, err := server.createDesignedReportHTMLExport(ctx, missionID, req.SourceArtifactID, reportDesignRequest{
				AgentExecutor:        req.AgentExecutor,
				AgentModel:           req.AgentModel,
				AgentReasoningEffort: req.AgentReasoningEffort,
			}, pendingEventID)
			return err
		},
		GenerateHumanize: func(ctx context.Context, missionID string, req reportexecution.HumanizeRequest, pendingEventID string) error {
			_, err := server.createReportHumanize(ctx, missionID, reportHumanizeRequest{
				Title:                req.Title,
				AgentExecutor:        req.AgentExecutor,
				AgentModel:           req.AgentModel,
				AgentReasoningEffort: req.AgentReasoningEffort,
				MCPMode:              req.MCPMode,
			}, pendingEventID, req)
			return err
		},
		GeneratePatch: func(ctx context.Context, missionID string, req reportexecution.PatchRequest, pendingEventID string) error {
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
	completed := reportexecution.CompletedPendingEventIDs(events)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if _, ok := completed[event.EventID]; ok {
			continue
		}
		switch event.EventType {
		case "report.draft.pending":
			if !server.runningReports.Owns(missionID, event.EventID) {
				return server.resumeReportDraftWorker(ctx, missionID, event)
			}
		case "report.humanize.pending":
			if server.runningReports.Owns(missionID, event.EventID) {
				continue
			}
			if recovered, err := server.recoverStaleReportHumanizeFinalizedPatch(ctx, missionID, event); err != nil {
				return err
			} else if recovered {
				return nil
			}
			return server.reportRunner().ResumeHumanize(ctx, missionID, event)
		case "report.patch.pending":
			if !server.runningReports.Owns(missionID, event.EventID) {
				return server.reportRunner().ResumePatch(ctx, missionID, event)
			}
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
	executionStrategy, err := normalizeReportExecutionStrategy(req.ExecutionStrategy, reportMode)
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
	if strings.TrimSpace(req.AgentSelectionSource) == "" {
		agentModel, agentReasoningEffort, err = resolveAgentSettings(executorName, agentModel, agentReasoningEffort, server.latestAgentSessionID(ctx, missionID, executorName))
		if err != nil {
			return nil, err
		}
	}
	if err := server.validateReportSessionPolicy(ctx, missionID, executorName, reportMode, reportSessionPolicy, executor, false); err != nil {
		return nil, err
	}
	postReportHumanize := reportprompt.NormalizePostReportHumanize(req.PostReportHumanize)
	guidanceProfile, guidanceSHA, err := reportprompt.SelectReportGenerationGuidanceForMode(reportMode, req.GenerationGuidanceProfile)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GenerationGuidanceSHA256) != "" {
		guidanceSHA = strings.TrimSpace(req.GenerationGuidanceSHA256)
	}
	switch reportMode {
	case reportModeLongForm:
		if executionStrategy == reportExecutionStrategySectionFanout {
			return server.createSectionFanoutLongFormReportDraft(ctx, missionID, title, req.DirectionHint, executorName, agentModel, agentReasoningEffort, req.AgentSelectionSource, mcpMode, rigor, reportSessionPolicy, req.ReportSessionPolicySelection, postReportHumanize, guidanceProfile, guidanceSHA, pendingEventID, executor)
		}
		return server.createSectionalLongFormReportDraft(ctx, missionID, title, req.DirectionHint, executorName, agentModel, agentReasoningEffort, req.AgentSelectionSource, mcpMode, rigor, reportSessionPolicy, req.ReportSessionPolicySelection, postReportHumanize, guidanceProfile, guidanceSHA, pendingEventID, executor)
	case reportModePlanned:
		return server.createPlannedReportDraft(ctx, missionID, title, req.DirectionHint, executorName, agentModel, agentReasoningEffort, req.AgentSelectionSource, mcpMode, rigor, reportSessionPolicy, req.ReportSessionPolicySelection, postReportHumanize, guidanceProfile, guidanceSHA, pendingEventID, executor)
	default:
		return server.createOneTakeReportDraft(ctx, missionID, title, req.DirectionHint, executorName, agentModel, agentReasoningEffort, req.AgentSelectionSource, mcpMode, rigor, reportSessionPolicy, req.ReportSessionPolicySelection, postReportHumanize, guidanceProfile, guidanceSHA, pendingEventID, executor)
	}
}

func (server *Server) createReportHumanize(ctx context.Context, missionID string, req reportHumanizeRequest, pendingEventID string, humanizeReq reportexecution.HumanizeRequest) (map[string]any, error) {
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

func (server *Server) createReportPatch(ctx context.Context, missionID string, req reportPatchRequest, pendingEventID string, patchReq reportexecution.PatchRequest) (map[string]any, error) {
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
	return reportpatch.MCPTools()
}

func (server *Server) createOneTakeReportDraft(ctx context.Context, missionID string, title string, directionHint string, executorName string, agentModel string, agentReasoningEffort string, agentSelectionSource string, mcpMode string, rigor reportRigorProfile, reportSessionPolicy string, reportSessionPolicySelection string, postReportHumanize string, generationGuidanceProfile string, generationGuidanceSHA256 string, pendingEventID string, executor AgentExecutor) (map[string]any, error) {
	return server.createReportWorkflowDraft(ctx, missionID, title, directionHint, executorName, agentModel, agentReasoningEffort, agentSelectionSource, mcpMode, rigor, reportSessionPolicy, reportSessionPolicySelection, postReportHumanize, generationGuidanceProfile, generationGuidanceSHA256, pendingEventID, reportModeOneTake, reportExecutionStrategySerial, executor)
}

func (server *Server) createPlannedReportDraft(ctx context.Context, missionID string, title string, directionHint string, executorName string, agentModel string, agentReasoningEffort string, agentSelectionSource string, mcpMode string, rigor reportRigorProfile, reportSessionPolicy string, reportSessionPolicySelection string, postReportHumanize string, generationGuidanceProfile string, generationGuidanceSHA256 string, pendingEventID string, executor AgentExecutor) (map[string]any, error) {
	return server.createReportWorkflowDraft(ctx, missionID, title, directionHint, executorName, agentModel, agentReasoningEffort, agentSelectionSource, mcpMode, rigor, reportSessionPolicy, reportSessionPolicySelection, postReportHumanize, generationGuidanceProfile, generationGuidanceSHA256, pendingEventID, reportModePlanned, reportExecutionStrategySerial, executor)
}

func (server *Server) createReportWorkflowDraft(ctx context.Context, missionID string, title string, directionHint string, executorName string, agentModel string, agentReasoningEffort string, agentSelectionSource string, mcpMode string, rigor reportRigorProfile, reportSessionPolicy string, reportSessionPolicySelection string, postReportHumanize string, generationGuidanceProfile string, generationGuidanceSHA256 string, pendingEventID string, reportMode string, executionStrategy string, executor AgentExecutor) (map[string]any, error) {
	runner := reportworkflow.NewRunner(reportworkflow.RunnerConfig{
		Service:         server.service,
		Lifecycle:       reporting.Runner(server.reportRunner()),
		Executor:        executor,
		NewID:           newID,
		LatestSessionID: server.latestAgentSessionID,
	})
	output, err := runner.RunDraft(ctx, reportworkflow.DraftInput{
		MissionID:                    missionID,
		PendingEventID:               pendingEventID,
		Title:                        title,
		DirectionHint:                directionHint,
		ExecutionStrategy:            executionStrategy,
		AgentExecutor:                executorName,
		AgentModel:                   agentModel,
		AgentReasoningEffort:         agentReasoningEffort,
		AgentSelectionSource:         agentSelectionSource,
		MCPMode:                      mcpMode,
		Rigor:                        reportWorkflowRigor(rigor),
		ReportMode:                   reportMode,
		ReportSessionPolicy:          reportSessionPolicy,
		ReportSessionPolicySelection: strings.TrimSpace(reportSessionPolicySelection),
		PostReportHumanize:           postReportHumanize,
		GenerationGuidanceProfile:    generationGuidanceProfile,
		GenerationGuidanceSHA256:     generationGuidanceSHA256,
	})
	if err != nil {
		return nil, err
	}
	result := map[string]any{"artifact": output.Artifact, "event": output.Event, "markdown": output.Markdown}
	if postReportHumanize == "disabled" {
		return result, nil
	}
	humanized, err := server.humanizeMarkdownReport(ctx, missionID, reportHumanizeInput{
		Title:             title,
		Markdown:          output.Markdown,
		SourceArtifact:    output.Artifact,
		ExecutorName:      executorName,
		AgentModel:        agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MCPMode:           mcpMode,
		PreviousSessionID: output.ReportSessionID,
		ReportMode:        reportMode,
		PendingEventID:    pendingEventID,
	}, executor)
	if err != nil {
		return nil, err
	}
	result["humanized"] = humanized
	return result, nil
}

func reportWorkflowRigor(rigor reportRigorProfile) reportprompt.RigorProfile {
	return reportprompt.RigorProfile{
		Level:        rigor.level,
		Label:        rigor.label,
		Description:  rigor.description,
		Instructions: rigor.instructions,
	}
}

func (server *Server) createSectionalLongFormReportDraft(ctx context.Context, missionID string, title string, directionHint string, executorName string, agentModel string, agentReasoningEffort string, agentSelectionSource string, mcpMode string, rigor reportRigorProfile, reportSessionPolicy string, reportSessionPolicySelection string, postReportHumanize string, generationGuidanceProfile string, generationGuidanceSHA256 string, pendingEventID string, executor AgentExecutor) (map[string]any, error) {
	return server.createLongFormPrefixWorkflowDraft(ctx, missionID, title, directionHint, executorName, agentModel, agentReasoningEffort, agentSelectionSource, mcpMode, rigor, reportSessionPolicy, reportSessionPolicySelection, postReportHumanize, generationGuidanceProfile, generationGuidanceSHA256, pendingEventID, reportExecutionStrategySerial, executor)
}

func (server *Server) createSectionalLongFormReportDraftLegacy(ctx context.Context, missionID string, title string, directionHint string, executorName string, agentModel string, agentReasoningEffort string, agentSelectionSource string, mcpMode string, rigor reportRigorProfile, reportSessionPolicy string, reportSessionPolicySelection string, postReportHumanize string, generationGuidanceProfile string, generationGuidanceSHA256 string, pendingEventID string, executor AgentExecutor) (map[string]any, error) {
	return server.createLongFormPrefixWorkflowDraft(ctx, missionID, title, directionHint, executorName, agentModel, agentReasoningEffort, agentSelectionSource, mcpMode, rigor, reportSessionPolicy, reportSessionPolicySelection, postReportHumanize, generationGuidanceProfile, generationGuidanceSHA256, pendingEventID, reportExecutionStrategySerial, executor)
}

func logLongFormFinalObservation(missionID, pendingEventID, planEventID string, attempt int, boundSessionID string, result AgentResult, durationMS int64) {
	returnedSessionID := strings.TrimSpace(result.SessionID)
	inputTokens, outputTokens, totalTokens := 0, 0, 0
	usageAvailable := result.Usage.ProviderUsage != nil
	if usageAvailable {
		inputTokens = result.Usage.ProviderUsage.InputTokens
		outputTokens = result.Usage.ProviderUsage.OutputTokens
		totalTokens = result.Usage.ProviderUsage.TotalTokens
		if totalTokens == 0 {
			totalTokens = inputTokens + outputTokens
		}
	}
	log.Printf("report_long_form_final_observed mission_id=%q pending_event_id=%q plan_event_id=%q attempt_count=%d returned_session_present=%t returned_session_matches_bound=%t usage_available=%t input_tokens=%d output_tokens=%d total_tokens=%d resumed=%t duration_ms=%d", missionID, pendingEventID, planEventID, attempt, returnedSessionID != "", returnedSessionID != "" && returnedSessionID == strings.TrimSpace(boundSessionID), usageAvailable, inputTokens, outputTokens, totalTokens, result.Resumed, durationMS)
}

func agentOneTakeMarkdownReportPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile, generationGuidanceProfile string) string {
	return reportprompt.OneTakeMarkdownReportPrompt(title, missionID, toolSessionID, reportWorkflowRigor(rigor), generationGuidanceProfile)
}

func agentMarkdownReportPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile, plan agentReportPlan, generationGuidanceProfile string) string {
	return reportprompt.PlannedMarkdownReportPrompt(title, missionID, toolSessionID, reportWorkflowRigor(rigor), plan, generationGuidanceProfile)
}

func agentReportPatchPrompt(title string, missionID string, toolSessionID string, pendingEventID string, baseArtifactID string, instruction string, req reportexecution.PatchRequest) string {
	return AgentReportPatchPrompt(title, missionID, toolSessionID, pendingEventID, baseArtifactID, instruction, req)
}

// AgentReportPatchPrompt는 patch agent에게 전달할 report 수정 지시문을 조립한다.
func AgentReportPatchPrompt(title string, missionID string, toolSessionID string, pendingEventID string, baseArtifactID string, instruction string, req reportexecution.PatchRequest) string {
	return reportpatch.Prompt(title, missionID, toolSessionID, pendingEventID, baseArtifactID, instruction, req)
}

func normalizeReportMode(mode string) (string, error) {
	return reportexecution.NormalizeMode(mode)
}

func normalizeReportExecutionStrategy(strategy string, reportMode string) (string, error) {
	strategy = strings.TrimSpace(strings.ToLower(strategy))
	if strategy == "" || strategy == reportExecutionStrategySerial {
		return reportExecutionStrategySerial, nil
	}
	if strategy != reportExecutionStrategySectionFanout {
		return "", fmt.Errorf("%w: unsupported report execution strategy", app.ErrInvalidInput)
	}
	if reportMode != reportModeLongForm {
		return "", fmt.Errorf("%w: section fanout is only supported for long-form reports", app.ErrInvalidInput)
	}
	return strategy, nil
}

func storedReportExecutionStrategy(strategy string) string {
	if strings.TrimSpace(strings.ToLower(strategy)) == reportExecutionStrategySerial {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(strategy))
}

func reportEventString(event app.LedgerEvent, key string) string {
	var payload map[string]any
	if json.Unmarshal(event.Payload, &payload) != nil {
		return ""
	}
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}

func normalizeReportSessionPolicy(policy string) (string, error) {
	return reportexecution.NormalizeSessionPolicy(policy)
}

func (server *Server) selectReportSessionPolicy(ctx context.Context, missionID string, executorName string, reportMode string, requestedPolicy string, executor AgentExecutor) (string, string, error) {
	requestedPolicy = strings.TrimSpace(requestedPolicy)
	if requestedPolicy == "" {
		return reportexecution.SelectSessionPolicy(reportexecution.SessionPolicySelectionInput{
			ReportMode: reportMode,
		})
	}
	requestedCanonical, err := reportexecution.NormalizeSessionPolicy(requestedPolicy)
	if err != nil {
		return "", "", err
	}
	if requestedCanonical != reportexecution.SessionPolicyIsolatedFork {
		return reportexecution.SelectSessionPolicy(reportexecution.SessionPolicySelectionInput{
			RequestedPolicy: requestedPolicy,
			ReportMode:      reportMode,
		})
	}
	_, canFork := executor.(AgentSessionForker)
	_, canCheckFork := executor.(AgentSessionForkReadiness)
	preReportSessionID := ""
	forkReady := false
	if canFork {
		preReportSessionID = strings.TrimSpace(server.latestAgentSessionID(ctx, missionID, executorName))
		forkReady = canCheckFork && AgentSessionForkReady(ctx, executor, preReportSessionID)
	}
	return reportexecution.SelectSessionPolicy(reportexecution.SessionPolicySelectionInput{
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
	return reportexecution.ValidateSessionPolicy(policy, reportMode, canFork, !requireReady || preReportSessionID != "", !requireReady || (canCheckFork && AgentSessionForkReady(ctx, executor, preReportSessionID)))
}

func reportModeLabel(mode string) string {
	return reportexecution.ModeLabel(mode)
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

func agentReportPlanPrompt(title string, missionID string, toolSessionID string, pendingEventID string, idempotencyKey string, rigor reportRigorProfile, generationGuidanceProfile string) string {
	return reportprompt.MarkdownReportPlanPrompt(title, missionID, toolSessionID, pendingEventID, idempotencyKey, reportWorkflowRigor(rigor), generationGuidanceProfile)
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

package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportpatch"
	"net/http"
	"net/url"
	"strings"
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
	pending, err := server.service.RequestReportRetry(r.Context(), reportexecution.ReportRetryRequest{
		EventID: newID("evt"), MissionID: missionID, FailedPendingEventID: req.FailedPendingEventID,
		Strategy: req.Strategy, RetryRequestID: req.RetryRequestID, Producer: ledger.Producer{Type: "user", ID: "plasma-ui"},
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

func (server *Server) cancelReportDraft(ctx context.Context, missionID string) (ledger.Event, bool, error) {
	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return ledger.Event{}, false, err
	}
	pending, ok := latestOpenReportDraftPendingEvent(events)
	if !ok {
		return ledger.Event{}, false, fmt.Errorf("%w: no report draft is running for this mission", app.ErrInvalidInput)
	}
	cancelInFlightPendingEventID := server.reportCancelInFlightPendingEventID(missionID, pending)
	canceledInFlight := false
	if pending.EventType != "report.humanize.pending" {
		canceledInFlight = server.runningReports.Cancel(missionID, cancelInFlightPendingEventID)
	} else {
		canceledInFlight = server.runningReports.Owns(missionID, cancelInFlightPendingEventID)
	}
	event, err := server.reportRunner().AppendCanceled(ctx, missionID, pending, canceledInFlight, ledger.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		return ledger.Event{}, canceledInFlight, err
	}
	if pending.EventType == "report.humanize.pending" && canceledInFlight {
		server.runningReports.Cancel(missionID, cancelInFlightPendingEventID)
	}
	return event, canceledInFlight, nil
}

func (server *Server) reportCancelInFlightPendingEventID(missionID string, pending ledger.Event) string {
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
		Producer:             ledger.Producer{Type: "user", ID: "plasma-ui"},
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

// Historical artifact URLs remain readable; the retired launch endpoint is explicit.
func (server *Server) handleReportArtifactHumanizedMarkdownExport(w http.ResponseWriter, r *http.Request, missionID string, artifactID string) {
	writeError(w, http.StatusGone, "Deprecated post-canonical H5 has been removed. Use long-form style editing instead.")
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

package web

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func (server *Server) handleMissions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		missions, err := server.service.ListMissions(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"missions": missions})
	case http.MethodPost:
		var req createMissionRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		response, err := server.createMission(r.Context(), req)
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, response)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (server *Server) handleMissionRoute(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/missions/"), "/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	missionID := parts[0]
	if len(parts) == 1 {
		if r.Method == http.MethodPatch {
			server.handleMissionMetadataUpdate(w, r, missionID)
			return
		}
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		server.writeMissionDetail(w, r, missionID)
		return
	}

	switch parts[1] {
	case "activity":
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		server.handleMissionActivity(w, r, missionID)
	case "events":
		server.handleMissionEvents(w, r, missionID)
	case "recall":
		server.handleMissionRecall(w, r, missionID)
	case "turns":
		server.handleMissionTurns(w, r, missionID, parts[2:])
	case "agent_sessions":
		server.handleAgentSessions(w, r, missionID, parts[2:])
	case "workflows":
		server.handleMissionWorkflows(w, r, missionID, parts[2:])
	case "artifacts":
		server.handleMissionArtifacts(w, r, missionID, parts[2:])
	case "sources":
		server.handleMissionSources(w, r, missionID, parts[2:])
	case "connector-access":
		server.handleMissionConnectorAccess(w, r, missionID, parts[2:])
	case "records":
		server.handleMissionRecords(w, r, missionID)
	case "claims":
		server.handleMissionClaims(w, r, missionID, parts[2:])
	case "candidates":
		server.handleMissionCandidates(w, r, missionID, parts[2:])
	case "proposals":
		server.handleMissionProposals(w, r, missionID, parts[2:])
	case "reports":
		server.handleMissionReports(w, r, missionID, parts[2:])
	default:
		http.NotFound(w, r)
	}
}

func (server *Server) handleMissionActivity(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	activity, err := server.service.MissionActivity(r.Context(), missionID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, missionActivityResponse{
		Activity: activity,
		Cursor:   server.missionActivityCursor(activity.LastSequence),
	})
}

func (server *Server) handleMissionEvents(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	events, err := server.service.ListEvents(r.Context(), missionID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (server *Server) handleMissionConnectorAccess(w http.ResponseWriter, r *http.Request, missionID string, rest []string) {
	if len(rest) != 1 || rest[0] != app.ConfluenceConnectorID {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		access, err := server.service.GetMissionConnectorAccess(r.Context(), missionID, app.ConfluenceConnectorID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"access": access})
	case http.MethodPut:
		var req connectorAccessRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		result, err := server.service.SetMissionConnectorAccess(r.Context(), app.SetConnectorAccessRequest{
			EventID:      newID("evt"),
			MissionID:    missionID,
			ConnectorID:  app.ConfluenceConnectorID,
			Enabled:      req.Enabled,
			ConnectionID: req.ConnectionID,
			CloudID:      req.CloudID,
			SpaceKey:     req.SpaceKey,
			Producer:     app.Producer{Type: "user", ID: "plasma-ui"},
		})
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	case http.MethodDelete:
		result, err := server.service.SetMissionConnectorAccess(r.Context(), app.SetConnectorAccessRequest{
			EventID:     newID("evt"),
			MissionID:   missionID,
			ConnectorID: app.ConfluenceConnectorID,
			Producer:    app.Producer{Type: "user", ID: "plasma-ui"},
		})
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (server *Server) handleMissionRecall(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	recall, err := server.buildRecall(r.Context(), missionID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, recall)
}

func (server *Server) writeMissionDetail(w http.ResponseWriter, r *http.Request, missionID string) {
	if err := server.reconcileMissionRecovery(r.Context(), missionID); err != nil {
		writeAppError(w, err)
		return
	}
	detail, err := server.missionDetail(r.Context(), missionID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (server *Server) createMission(ctx context.Context, req createMissionRequest) (missionDetailResponse, error) {
	title := strings.TrimSpace(req.Title)
	objective := strings.TrimSpace(req.Objective)
	if title == "" {
		return missionDetailResponse{}, fmt.Errorf("%w: title is required", app.ErrInvalidInput)
	}
	if objective == "" {
		objective = title
	}
	missionID := newID("mis")
	if _, err := server.service.CreateMission(ctx, app.CreateMissionRequest{
		MissionID: missionID,
		Title:     title,
	}); err != nil {
		return missionDetailResponse{}, err
	}
	if _, err := server.service.AppendEvent(ctx, app.BuildMissionCreatedAppendRequest(app.MissionCreatedEventRequest{
		EventID:   newID("evt"),
		MissionID: missionID,
		Title:     title,
		Objective: objective,
		Scope: app.MissionScope{
			Included: trimStrings(req.Scope.Included),
			Excluded: trimStrings(req.Scope.Excluded),
		},
		Producer: app.Producer{Type: "user", ID: "plasma-ui"},
	})); err != nil {
		return missionDetailResponse{}, err
	}
	if _, err := server.service.RebuildProjection(ctx, missionID); err != nil {
		return missionDetailResponse{}, err
	}
	return server.missionDetail(ctx, missionID)
}

func (server *Server) missionDetail(ctx context.Context, missionID string) (missionDetailResponse, error) {
	projection, err := server.service.GetProjection(ctx, missionID)
	if err != nil {
		return missionDetailResponse{}, err
	}
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return missionDetailResponse{}, err
	}
	sources, err := server.service.ListSourceSnapshots(ctx, missionID)
	if err != nil {
		return missionDetailResponse{}, err
	}
	records, err := server.collectRecords(ctx, missionID, events)
	if err != nil {
		return missionDetailResponse{}, err
	}
	reports, err := server.service.ListReports(ctx, missionID)
	if err != nil {
		return missionDetailResponse{}, err
	}
	versions, err := server.service.ListReportVersions(ctx, missionID)
	if err != nil {
		return missionDetailResponse{}, err
	}
	workflowRuns, err := server.service.ListWorkflowRuns(ctx, missionID)
	if err != nil {
		return missionDetailResponse{}, err
	}
	recall, err := server.buildRecall(ctx, missionID)
	if err != nil {
		return missionDetailResponse{}, err
	}
	return missionDetailResponse{
		Projection:          projection,
		ActivityCursor:      server.missionActivityCursor(lastMissionEventSequence(events)),
		Events:              events,
		Sources:             sources,
		Records:             records,
		Reports:             reports,
		ReportVersions:      versions,
		WorkflowRuns:        workflowRuns,
		Recall:              recall,
		AgentExecutors:      server.agentStatuses(),
		LockedAgentExecutor: app.LockedAgentExecutorFromEvents(events),
		ActiveWork:          app.ActiveWorkFromMissionState(events, workflowRuns),
		ReportProgress:      app.ReportProgressFromEvents(events),
	}, nil
}

func lastMissionEventSequence(events []app.LedgerEvent) int64 {
	if len(events) == 0 {
		return 0
	}
	return events[len(events)-1].Sequence
}

func (server *Server) collectRecords(ctx context.Context, missionID string, events []app.LedgerEvent) (recordsResponse, error) {
	evidence, err := server.service.ListEvidenceRecords(ctx, missionID)
	if err != nil {
		return recordsResponse{}, err
	}
	claims, err := server.service.ListClaimRecords(ctx, missionID)
	if err != nil {
		return recordsResponse{}, err
	}
	questions, err := server.service.ListQuestionRecords(ctx, missionID)
	if err != nil {
		return recordsResponse{}, err
	}
	options, err := server.service.ListOptionRecords(ctx, missionID)
	if err != nil {
		return recordsResponse{}, err
	}
	proposals, err := server.service.ListProposalBundles(ctx, missionID)
	if err != nil {
		return recordsResponse{}, err
	}
	if events == nil {
		events, err = server.service.ListEvents(ctx, missionID)
		if err != nil {
			return recordsResponse{}, err
		}
	}
	return recordsResponse{
		Evidence:                           evidence,
		Claims:                             claims,
		ClaimConfidence:                    claimConfidenceViews(claims, events),
		Questions:                          questions,
		Options:                            options,
		Proposals:                          proposals,
		approvedObjectIDsByDecisionEventID: approvedObjectIDsByDecisionEventID(events),
	}, nil
}

func claimConfidenceViews(claims []app.ClaimRecord, events []app.LedgerEvent) []claimConfidenceView {
	updatesByClaim := map[string][]app.ClaimConfidenceUpdate{}
	for _, update := range app.ClaimConfidenceUpdatesFromEvents(events) {
		updatesByClaim[update.ClaimID] = append(updatesByClaim[update.ClaimID], update)
	}
	views := make([]claimConfidenceView, 0, len(claims))
	for _, claim := range claims {
		initial := displayConfidence(claim.Confidence)
		updates := updatesByClaim[claim.ClaimID]
		current := initial
		previous := initial
		direction := "initial"
		currentEventID := ""
		updatedAt := ""
		history := make([]claimConfidenceHistoryEntry, 0, len(updates))
		for _, update := range updates {
			current = displayConfidence(update.Confidence)
			entryDirection := confidenceDirection(previous.Level, current.Level)
			history = append(history, claimConfidenceHistoryEntry{
				EventID:           update.EventID,
				Sequence:          update.Sequence,
				PreviousLevel:     previous.Level,
				Level:             current.Level,
				Direction:         entryDirection,
				Rationale:         current.Rationale,
				OpenRisks:         current.OpenRisks,
				NeedsVerification: current.NeedsVerification,
				BasisEvidenceIDs:  append([]string(nil), update.BasisEvidenceIDs...),
				Origin:            update.Origin,
				Producer:          update.Producer,
				CreatedAt:         formatTimeRFC3339(update.CreatedAt),
			})
			previous = current
			direction = entryDirection
			currentEventID = update.EventID
			updatedAt = formatTimeRFC3339(update.CreatedAt)
		}
		truncated := false
		if len(history) > 10 {
			history = history[len(history)-10:]
			truncated = true
		}
		views = append(views, claimConfidenceView{
			ClaimID:           claim.ClaimID,
			InitialConfidence: initial,
			CurrentConfidence: current,
			CurrentEventID:    currentEventID,
			Direction:         direction,
			UpdatedAt:         updatedAt,
			History:           history,
			HistoryTruncated:  truncated,
		})
	}
	return views
}

func displayConfidence(confidence app.Confidence) app.Confidence {
	level := strings.TrimSpace(confidence.Level)
	if level == "" {
		level = "unknown"
	}
	return app.Confidence{
		Level:             level,
		Rationale:         strings.TrimSpace(confidence.Rationale),
		OpenRisks:         trimStrings(confidence.OpenRisks),
		NeedsVerification: confidence.NeedsVerification,
	}
}

func confidenceDirection(previousLevel, currentLevel string) string {
	previous := confidenceRank(previousLevel)
	current := confidenceRank(currentLevel)
	switch {
	case current > previous:
		return "up"
	case current < previous:
		return "down"
	default:
		return "unchanged"
	}
}

func confidenceRank(level string) int {
	switch strings.TrimSpace(level) {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func formatTimeRFC3339(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func (server *Server) buildRecall(ctx context.Context, missionID string) (recallPreview, error) {
	projection, err := server.service.GetProjection(ctx, missionID)
	if err != nil {
		return recallPreview{}, err
	}
	sources, err := server.service.ListSourceSnapshots(ctx, missionID)
	if err != nil {
		return recallPreview{}, err
	}
	records, err := server.collectRecords(ctx, missionID, nil)
	if err != nil {
		return recallPreview{}, err
	}
	return recallPreview{
		SchemaVersion: "plasma.agent_recall_preview.v1",
		Mission: recallMission{
			MissionID: projection.MissionID,
			Title:     projection.Title,
			Objective: projection.Objective,
			Scope:     projection.Scope,
		},
		Sources:         sources,
		OpenQuestionIDs: projection.OpenQuestionIDs,
		SavedEvidence:   approvedEvidence(records),
		SavedClaims:     approvedClaimsByProposal(records),
		AllowedTools: []string{
			"plasma.research.outline",
			"plasma.research.list",
			"plasma.research.grep",
			"plasma.research.read",
			"plasma.research.references",
			"plasma.sources.read",
			"plasma.sources.tree",
			"plasma.sources.grep",
			"plasma.sources.search",
		},
		InvestigationAllowed: true,
		SourceSearchAllowed:  true,
	}, nil
}

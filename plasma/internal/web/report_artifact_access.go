package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"mime"
	"path/filepath"
	"strings"
)

func (server *Server) reportArtifact(ctx context.Context, missionID string, artifactID string) (artifactcontract.Raw, error) {
	artifact, err := server.service.GetRawArtifact(ctx, artifactID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return artifactcontract.Raw{}, fmt.Errorf("%w: artifact not found", app.ErrInvalidInput)
		}
		return artifactcontract.Raw{}, err
	}
	if artifact.MissionID != missionID {
		return artifactcontract.Raw{}, fmt.Errorf("%w: artifact not found", app.ErrInvalidInput)
	}
	if ok, err := server.isReportArtifact(ctx, missionID, artifact.ArtifactID); err != nil {
		return artifactcontract.Raw{}, err
	} else if !ok {
		return artifactcontract.Raw{}, fmt.Errorf("%w: artifact not found", app.ErrInvalidInput)
	}
	return artifact, nil
}

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

func isMarkdownMediaType(mediaType string) bool {
	base, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		base = mediaType
	}
	base = strings.ToLower(strings.TrimSpace(base))
	return base == "text/markdown" || base == "text/x-markdown"
}

func isImageMediaType(mediaType string) bool {
	return mediaKindForType(mediaType) == sourcecontract.MediaKindImage
}

func reportArtifactTitle(artifact artifactcontract.Raw) string {
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

func markdownReportHTMLFilename(artifact artifactcontract.Raw) string {
	title := reportArtifactTitle(artifact)
	return safeFilename(title, ".html")
}

func (server *Server) isArticleArtifact(ctx context.Context, missionID string, artifactID string) (bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return false, err
	}
	for _, event := range events {
		if event.EventType != "report.artifact.created" {
			continue
		}
		var payload struct {
			ArtifactID string `json:"artifact_id"`
			Kind       string `json:"kind"`
			OutputKind string `json:"output_kind"`
		}
		if json.Unmarshal(event.Payload, &payload) == nil && strings.TrimSpace(payload.ArtifactID) == artifactID && (strings.TrimSpace(payload.Kind) == "article_artifact" || strings.TrimSpace(payload.OutputKind) == reportexecution.OutputKindArticle) {
			return true, nil
		}
	}
	return false, nil
}

func (server *Server) isReportArtifact(ctx context.Context, missionID string, artifactID string) (bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return false, err
	}
	for _, event := range events {
		if event.EventType == "report.artifact.reprojected" {
			var repair struct {
				SourceEventID string `json:"source_event_id"`
			}
			if json.Unmarshal(event.Payload, &repair) != nil || strings.TrimSpace(repair.SourceEventID) == "" {
				continue
			}
			decoded, err := reportilcontract.DecodeTerminalPayload(event.Payload)
			if err != nil {
				continue
			}
			sourceFound := false
			for _, candidate := range events {
				if candidate.EventID == repair.SourceEventID && candidate.EventType == "report.artifact.created" {
					sourceFound = true
					break
				}
			}
			if !sourceFound {
				continue
			}
			for _, entry := range decoded.Bundle.Artifacts {
				if entry.ArtifactID == artifactID {
					return true, nil
				}
			}
			continue
		}
		if event.EventType != "report.artifact.created" && event.EventType != "report.artifact.exported" && event.EventType != app.ReportRedpenSavedEvent {
			continue
		}
		var payload struct {
			ArtifactID     string `json:"artifact_id"`
			Kind           string `json:"kind"`
			PendingEventID string `json:"pending_event_id"`
			PipelineFamily string `json:"pipeline_family"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		kind := strings.TrimSpace(payload.Kind)
		if strings.TrimSpace(payload.ArtifactID) == artifactID && (kind == reportexecution.ExportKindSelfContainedHTML || kind == reportexecution.ExportKindDesignedHTML || kind == reportexecution.ExportKindHumanizedMarkdown || kind == app.ReportRedpenArtifactKind) {
			return true, nil
		}
		if kind == "markdown_report_artifact" && payload.PipelineFamily == reportilcontract.PipelineFamily {
			var pending ledger.Event
			found := false
			for _, candidate := range events {
				if candidate.EventID == payload.PendingEventID {
					pending, found = candidate, true
					break
				}
			}
			if !found || pending.EventType != "report.draft.pending" {
				continue
			}
			var pendingPayload struct {
				PipelineFamily string `json:"pipeline_family"`
			}
			if json.Unmarshal(pending.Payload, &pendingPayload) != nil || pendingPayload.PipelineFamily != reportilcontract.PipelineFamily {
				continue
			}
			decoded, err := reportilcontract.DecodeTerminalPayload(event.Payload)
			if err != nil || decoded.PendingEventID != pending.EventID {
				continue
			}
			for _, entry := range decoded.Bundle.Artifacts {
				if entry.ArtifactID == artifactID {
					return true, nil
				}
			}
		}
		family := strings.TrimSpace(payload.PipelineFamily)
		if (kind == "markdown_report_artifact" || kind == "article_artifact") &&
			(family == "" || family == reportpipeline.Unverified) &&
			payload.ArtifactID == artifactID {
			return true, nil
		}
	}
	return false, nil
}

func (server *Server) reportArtifactEventForPending(ctx context.Context, missionID string, pendingEventID string) (ledger.Event, bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return ledger.Event{}, false, err
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
	return ledger.Event{}, false, nil
}

func (server *Server) reportPatchFinalizedEventForPending(ctx context.Context, missionID string, pendingEventID string) (ledger.Event, bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return ledger.Event{}, false, err
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
	return ledger.Event{}, false, nil
}

func (server *Server) promoteReportPatchFinalizedArtifact(ctx context.Context, missionID string, finalized ledger.Event) (ledger.Event, error) {
	var payload map[string]any
	if err := json.Unmarshal(finalized.Payload, &payload); err != nil {
		return ledger.Event{}, fmt.Errorf("%w: invalid report patch finalized payload", app.ErrInvalidInput)
	}
	pendingEventID, _ := payload["pending_event_id"].(string)
	pendingEventID = strings.TrimSpace(pendingEventID)
	if pendingEventID == "" {
		return ledger.Event{}, fmt.Errorf("%w: report patch finalized payload is missing pending_event_id", app.ErrInvalidInput)
	}
	if event, ok, err := server.reportArtifactEventForPending(ctx, missionID, pendingEventID); err != nil {
		return ledger.Event{}, err
	} else if ok {
		return event, nil
	}
	artifactID, _ := payload["artifact_id"].(string)
	artifact, err := server.service.GetRawArtifact(ctx, strings.TrimSpace(artifactID))
	if err != nil {
		return ledger.Event{}, err
	}
	if artifact.MissionID != missionID {
		return ledger.Event{}, fmt.Errorf("%w: finalized report artifact belongs to another mission", app.ErrInvalidInput)
	}
	producerID, _ := payload["report_session_id"].(string)
	producerID = firstNonEmpty(strings.TrimSpace(producerID), strings.TrimSpace(finalized.CorrelationID))
	return server.service.AppendEvent(ctx, reporting.BuildPromotedMarkdownReportArtifactAppendRequest(reporting.PromotedMarkdownReportArtifactEventRequest{
		EventID:             newID("evt"),
		MissionID:           missionID,
		PromotedFromEventID: finalized.EventID,
		Payload:             payload,
		Producer:            ledger.Producer{Type: "agent_session", ID: producerID},
	}))
}

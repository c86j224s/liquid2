package sourcecandidates

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func BuildSourceCandidateProposalEventRequest(req SourceCandidateProposalEventRequest) (app.AppendEventRequest, bool, error) {
	if len(req.Candidates) == 0 {
		return app.AppendEventRequest{}, false, nil
	}
	payload := map[string]any{
		"kind":            "source_candidate_proposed",
		"user_event_id":   strings.TrimSpace(req.UserEventID),
		"agent_event_id":  strings.TrimSpace(req.AgentEventID),
		"agent_executor":  strings.TrimSpace(req.ExecutorName),
		"mcp_mode":        strings.TrimSpace(req.MCPMode),
		"candidate_count": len(req.Candidates),
		"candidates":      req.Candidates,
	}
	if strings.TrimSpace(req.ToolSessionID) != "" {
		payload["tool_session_id"] = strings.TrimSpace(req.ToolSessionID)
	}
	if strings.TrimSpace(req.StrategyID) != "" {
		payload["strategy_id"] = strings.TrimSpace(req.StrategyID)
	}
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(req.MissionID),
		EventType: "source.candidate.proposed",
		Producer:  req.Producer,
		Payload:   mustMarshalJSON(payload),
	}, true, nil
}

func BuildSourceCandidateMCPProposalEventRequest(req SourceCandidateMCPProposalEventRequest) (app.AppendEventRequest, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	payload := map[string]any{
		"kind":             "source_candidate_proposed",
		"source":           "mcp",
		"tool_session_id":  sessionID,
		"agent_session_id": sessionID,
		"candidate_count":  len(req.Candidates),
		"candidates":       req.Candidates,
	}
	if strings.TrimSpace(req.CurrentUserEventID) != "" {
		payload["user_event_id"] = strings.TrimSpace(req.CurrentUserEventID)
	}
	if strings.TrimSpace(req.AgentExecutor) != "" {
		payload["agent_executor"] = strings.TrimSpace(req.AgentExecutor)
	}
	return app.AppendEventRequest{
		EventID:       strings.TrimSpace(req.EventID),
		MissionID:     strings.TrimSpace(req.MissionID),
		EventType:     "source.candidate.proposed",
		Producer:      req.Producer,
		CorrelationID: sessionID,
		Payload:       mustMarshalJSON(payload),
	}, nil
}

func BuildWorkflowSourceCandidateProposalEventRequest(req WorkflowSourceCandidateProposalEventRequest) (app.AppendEventRequest, bool, error) {
	if len(req.Candidates) == 0 {
		return app.AppendEventRequest{}, false, nil
	}
	payload := map[string]any{
		"kind":             "source_candidate_proposed",
		"workflow_run_id":  strings.TrimSpace(req.WorkflowRunID),
		"workflow_step_id": strings.TrimSpace(req.WorkflowStepID),
		"user_event_id":    strings.TrimSpace(req.UserEventID),
		"agent_event_id":   strings.TrimSpace(req.AgentEventID),
		"candidates":       req.Candidates,
	}
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(req.MissionID),
		EventType: "source.candidate.proposed",
		Producer:  req.Producer,
		Payload:   mustMarshalJSON(payload),
	}, true, nil
}

func sourceCandidateStagedEventRequest(job SourceCandidateStagingJob, eventID string, artifact app.RawArtifact, title string, fetched SourceCandidateFetched) app.AppendEventRequest {
	sessionID := strings.TrimSpace(job.SessionID)
	payload := map[string]any{
		"kind":               "source_candidate_staged",
		"candidate_kind":     sourceCandidateKind(firstNonEmpty(fetched.CandidateKind, job.CandidateKind)),
		"url":                job.Candidate.URL,
		"title":              title,
		"proposal_event_id":  strings.TrimSpace(job.ProposalEventID),
		"staging_event_id":   strings.TrimSpace(job.StartedEventID),
		"artifact_id":        artifact.ArtifactID,
		"media_type":         artifact.MediaType,
		"byte_size":          artifact.ByteSize,
		"sha256":             artifact.SHA256,
		"approval_state":     "unapproved_candidate",
		"not_report_default": true,
	}
	if fetched.ExternalVersion != "" {
		payload["external_version"] = fetched.ExternalVersion
	}
	if !fetched.ExternalUpdatedAt.IsZero() {
		payload["external_updated_at"] = fetched.ExternalUpdatedAt.Format(time.RFC3339Nano)
	}
	if fetched.ByteSize > 0 {
		payload["byte_size"] = fetched.ByteSize
	}
	if fetched.PageCount > 0 {
		payload["page_count"] = fetched.PageCount
	}
	if fetched.TextLength > 0 {
		payload["text_length"] = fetched.TextLength
	}
	payload["text_length_known"] = fetched.TextLengthKnown
	if sessionID != "" {
		payload["tool_session_id"] = sessionID
		payload["agent_session_id"] = sessionID
	}
	if job.EmitAgentExecutorInTerminalEvents && strings.TrimSpace(job.AgentExecutor) != "" {
		payload["agent_executor"] = strings.TrimSpace(job.AgentExecutor)
	}
	return app.AppendEventRequest{
		EventID:          strings.TrimSpace(eventID),
		MissionID:        strings.TrimSpace(job.MissionID),
		EventType:        "source.candidate.staged",
		Producer:         job.Producer,
		CausationEventID: strings.TrimSpace(job.StartedEventID),
		CorrelationID:    sessionID,
		Payload:          mustMarshalJSON(payload),
	}
}

func sourceCandidateStagingFailedEventRequest(job SourceCandidateStagingJob, eventID string, cause error) app.AppendEventRequest {
	sessionID := strings.TrimSpace(job.SessionID)
	payload := map[string]any{
		"kind":               "source_candidate_staging_failed",
		"candidate_kind":     sourceCandidateKind(job.CandidateKind),
		"url":                job.Candidate.URL,
		"title":              job.Candidate.Title,
		"proposal_event_id":  strings.TrimSpace(job.ProposalEventID),
		"staging_event_id":   strings.TrimSpace(job.StartedEventID),
		"message":            cause.Error(),
		"approval_state":     "unapproved_candidate",
		"not_report_default": true,
	}
	if sessionID != "" {
		payload["tool_session_id"] = sessionID
		payload["agent_session_id"] = sessionID
	}
	if job.EmitAgentExecutorInTerminalEvents && strings.TrimSpace(job.AgentExecutor) != "" {
		payload["agent_executor"] = strings.TrimSpace(job.AgentExecutor)
	}
	return app.AppendEventRequest{
		EventID:          strings.TrimSpace(eventID),
		MissionID:        strings.TrimSpace(job.MissionID),
		EventType:        "source.candidate.staging_failed",
		Producer:         job.Producer,
		CausationEventID: strings.TrimSpace(job.StartedEventID),
		CorrelationID:    sessionID,
		Payload:          mustMarshalJSON(payload),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func mustMarshalJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("marshal source candidate event payload: %v", err))
	}
	return data
}

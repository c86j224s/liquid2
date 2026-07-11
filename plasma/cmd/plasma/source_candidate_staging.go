package main

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidates"
	"github.com/c86j224s/liquid2/plasma/internal/sources/urlsource"
)

func cliSourceCandidateStager(svc *app.Service) func(context.Context, app.LedgerEvent) {
	return func(ctx context.Context, event app.LedgerEvent) {
		stageCLISourceCandidateProposalEvent(ctx, svc, event)
	}
}

func stageCLISourceCandidateProposalEvent(ctx context.Context, svc *app.Service, event app.LedgerEvent) {
	var payload struct {
		ToolSessionID string `json:"tool_session_id"`
		AgentExecutor string `json:"agent_executor"`
		Candidates    []struct {
			URL    string `json:"url"`
			Title  string `json:"title"`
			Reason string `json:"reason"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil || len(payload.Candidates) == 0 {
		return
	}
	toolSessionID := strings.TrimSpace(payload.ToolSessionID)
	for _, candidate := range payload.Candidates {
		proposal := sourcecandidates.SourceCandidateProposal{
			URL:    candidate.URL,
			Title:  strings.TrimSpace(candidate.Title),
			Reason: strings.TrimSpace(candidate.Reason),
			State:  "proposed",
		}
		started, err := sourcecandidates.StartStaging(ctx, svc, sourcecandidates.SourceCandidateStagingStartRequest{
			EventID:          cliNewID("evt"),
			MissionID:        event.MissionID,
			SessionID:        toolSessionID,
			ProposalEventID:  event.EventID,
			CausationEventID: event.EventID,
			Candidate:        proposal,
			Producer:         event.Producer,
			AgentExecutor:    strings.TrimSpace(payload.AgentExecutor),
		})
		if err != nil {
			continue
		}
		job := sourcecandidates.SourceCandidateStagingJob{
			MissionID:                         event.MissionID,
			SessionID:                         toolSessionID,
			ProposalEventID:                   event.EventID,
			Candidate:                         proposal,
			Producer:                          event.Producer,
			StartedEventID:                    started.Event.EventID,
			AgentExecutor:                     strings.TrimSpace(payload.AgentExecutor),
			EmitAgentExecutorInTerminalEvents: false,
		}
		_ = sourcecandidates.Stage(ctx, svc, sourcecandidates.SourceCandidateStageRequest{
			Job:              job,
			Fetcher:          cliSourceCandidateFetcher,
			NewArtifactID:    cliNewID,
			NewEventID:       cliNewID,
			FilenameFallback: "source-candidate",
		})
	}
}

func cliSourceCandidateFetcher(ctx context.Context, rawURL string) (sourcecandidates.SourceCandidateFetched, error) {
	fetched, err := urlsource.Fetch(ctx, rawURL)
	if err != nil {
		return sourcecandidates.SourceCandidateFetched{}, err
	}
	return sourcecandidates.SourceCandidateFetched{
		Content:           fetched.Content,
		MediaType:         fetched.MediaType,
		Title:             fetched.Title,
		ExternalVersion:   fetched.ExternalVersion,
		ExternalUpdatedAt: fetched.ExternalUpdatedAt,
		ByteSize:          fetched.ByteSize,
		PageCount:         fetched.PageCount,
		TextLength:        fetched.TextLength,
		TextLengthKnown:   fetched.TextLengthKnown,
	}, nil
}

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidates"
	"github.com/c86j224s/liquid2/plasma/internal/sources/pdftext"
	"github.com/c86j224s/liquid2/plasma/internal/sources/urlsource"
)

func (server *Server) callSourceCandidatesPropose(ctx context.Context, call ToolCall) ToolResult {
	var input sourceCandidatesProposeInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, producer, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundWriteSession(common); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	candidates, err := normalizeSourceCandidateProposals(input.Candidates)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	eventReq, err := sourcecandidates.BuildSourceCandidateMCPProposalEventRequest(sourcecandidates.SourceCandidateMCPProposalEventRequest{
		EventID:            newMCPID("evt"),
		MissionID:          common.MissionID,
		SessionID:          common.SessionID,
		CurrentUserEventID: server.binding.CurrentUserEventID,
		AgentExecutor:      server.binding.AgentExecutor,
		Producer:           producer,
		Candidates:         appSourceCandidateProposals(candidates),
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, nil)
	}
	event, err := server.service.AppendEvent(ctx, eventReq)
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, nil)
	}
	staging := make([]sourceCandidateStagingOutput, 0, len(candidates))
	createdEventIDs := []string{event.EventID}
	for _, candidate := range candidates {
		started, err := server.startSourceCandidateStaging(ctx, common.MissionID, common.SessionID, event.EventID, candidate, producer)
		if err != nil {
			return errorFromErr(call.Name, common.MissionID, err, []string{event.EventID})
		}
		if started.StagingEventID != "" {
			createdEventIDs = append(createdEventIDs, started.StagingEventID)
		}
		staging = append(staging, started)
	}
	return ToolResult{
		ToolName:             call.Name,
		MissionID:            common.MissionID,
		CreatedEventIDs:      createdEventIDs,
		RequiresUserApproval: true,
		Content: sourceCandidatesProposeOutput{
			EventID:    event.EventID,
			Candidates: candidates,
			Staging:    staging,
		},
	}
}

func (server *Server) startSourceCandidateStaging(ctx context.Context, missionID, sessionID, proposalEventID string, candidate sourceCandidateProposalEvent, producer app.Producer) (sourceCandidateStagingOutput, error) {
	started, err := sourcecandidates.StartStaging(ctx, server.service, sourcecandidates.SourceCandidateStagingStartRequest{
		EventID:         newMCPID("evt"),
		MissionID:       missionID,
		SessionID:       sessionID,
		ProposalEventID: proposalEventID,
		Candidate: sourcecandidates.SourceCandidateProposal{
			URL:    candidate.URL,
			Title:  candidate.Title,
			Reason: candidate.Reason,
			State:  candidate.State,
		},
		Producer:      producer,
		AgentExecutor: server.binding.AgentExecutor,
	})
	if err != nil {
		return sourceCandidateStagingOutput{}, err
	}
	output := sourceCandidateStagingOutput{
		URL:             started.Output.URL,
		ProposalEventID: started.Output.ProposalEventID,
		StagingEventID:  started.Output.StagingEventID,
		StagingState:    started.Output.StagingState,
		Message:         started.Output.Message,
	}
	go server.stageSourceCandidate(context.Background(), sourceCandidateStagingJob{
		MissionID:                         missionID,
		SessionID:                         sessionID,
		ProposalEventID:                   proposalEventID,
		Candidate:                         candidate,
		Producer:                          producer,
		StartedEventID:                    started.Event.EventID,
		AgentExecutor:                     server.binding.AgentExecutor,
		EmitAgentExecutorInTerminalEvents: true,
	})
	return output, nil
}

func normalizeSourceCandidateProposals(input []sourceCandidateProposalInput) ([]sourceCandidateProposalEvent, error) {
	appInput := make([]sourcecandidates.SourceCandidateProposalInput, 0, len(input))
	for _, candidate := range input {
		appInput = append(appInput, sourcecandidates.SourceCandidateProposalInput{
			URL:    candidate.URL,
			Title:  candidate.Title,
			Reason: candidate.Reason,
		})
	}
	normalized, err := sourcecandidates.NormalizeSourceCandidateProposals(appInput)
	if err != nil {
		return nil, err
	}
	candidates := make([]sourceCandidateProposalEvent, 0, len(normalized))
	for _, candidate := range normalized {
		candidates = append(candidates, sourceCandidateProposalEvent{
			URL:    candidate.URL,
			Title:  candidate.Title,
			Reason: candidate.Reason,
			State:  candidate.State,
		})
	}
	return candidates, nil
}

func appSourceCandidateProposals(candidates []sourceCandidateProposalEvent) []sourcecandidates.SourceCandidateProposal {
	appCandidates := make([]sourcecandidates.SourceCandidateProposal, 0, len(candidates))
	for _, candidate := range candidates {
		appCandidates = append(appCandidates, sourcecandidates.SourceCandidateProposal{
			URL:    candidate.URL,
			Title:  candidate.Title,
			Reason: candidate.Reason,
			State:  candidate.State,
		})
	}
	return appCandidates
}

func normalizeSourceCandidateURL(raw string) (string, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", fmt.Errorf("%w: source candidate URL is required", app.ErrInvalidInput)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", fmt.Errorf("%w: source candidate URL must be absolute", app.ErrInvalidInput)
	}
	if parsed.User != nil {
		return "", "", fmt.Errorf("%w: source candidate URL must not include credentials", app.ErrInvalidInput)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		parsed.Scheme = strings.ToLower(parsed.Scheme)
	default:
		return "", "", fmt.Errorf("%w: source candidate URL must use http or https", app.ErrInvalidInput)
	}
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""
	return parsed.String(), parsed.Hostname(), nil
}

type sourceCandidateStagingJob struct {
	MissionID                         string
	SessionID                         string
	ProposalEventID                   string
	Candidate                         sourceCandidateProposalEvent
	Producer                          app.Producer
	StartedEventID                    string
	AgentExecutor                     string
	EmitAgentExecutorInTerminalEvents bool
}

func (server *Server) stageSourceCandidate(ctx context.Context, job sourceCandidateStagingJob) {
	_ = sourcecandidates.Stage(ctx, server.service, sourcecandidates.SourceCandidateStageRequest{
		Job: sourcecandidates.SourceCandidateStagingJob{
			MissionID:       job.MissionID,
			SessionID:       job.SessionID,
			ProposalEventID: job.ProposalEventID,
			Candidate: sourcecandidates.SourceCandidateProposal{
				URL:    job.Candidate.URL,
				Title:  job.Candidate.Title,
				Reason: job.Candidate.Reason,
				State:  job.Candidate.State,
			},
			Producer:                          job.Producer,
			StartedEventID:                    job.StartedEventID,
			AgentExecutor:                     job.AgentExecutor,
			EmitAgentExecutorInTerminalEvents: job.EmitAgentExecutorInTerminalEvents,
		},
		Fetcher:          server.appSourceCandidateFetcher(),
		NewArtifactID:    newMCPID,
		NewEventID:       newMCPID,
		FilenameFallback: "source-candidate",
	})
}

func (server *Server) appSourceCandidateFetcher() sourcecandidates.SourceCandidateFetcher {
	fetcher := server.sourceCandidateFetcher
	if fetcher == nil {
		fetcher = urlsource.Fetch
	}
	return func(ctx context.Context, rawURL string) (sourcecandidates.SourceCandidateFetched, error) {
		fetched, err := fetcher(ctx, rawURL)
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
}

type sourceCandidateReadRecord struct {
	State           string
	URL             string
	ProposalEventID string
	StagingEventID  string
	ArtifactID      string
	Message         string
	FailureMessage  string
	Sequence        int64
}

func (server *Server) callSourceCandidatesRead(ctx context.Context, call ToolCall) ToolResult {
	var input sourceCandidatesReadInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	normalizedURL := ""
	if strings.TrimSpace(input.URL) != "" {
		var err error
		normalizedURL, _, err = normalizeSourceCandidateURL(input.URL)
		if err != nil {
			return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
		}
	}
	proposalEventID := strings.TrimSpace(input.ProposalEventID)
	if proposalEventID != "" {
		if err := validateID("evt_", proposalEventID); err != nil {
			return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
		}
	}
	stagingEventID := strings.TrimSpace(input.StagingEventID)
	if stagingEventID != "" {
		if err := validateID("evt_", stagingEventID); err != nil {
			return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
		}
	}
	artifactID := strings.TrimSpace(input.ArtifactID)
	if artifactID != "" {
		if err := validateID("art_", artifactID); err != nil {
			return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
		}
	}
	selectors := 0
	for _, value := range []string{normalizedURL, proposalEventID, stagingEventID, artifactID} {
		if value != "" {
			selectors++
		}
	}
	if selectors == 0 {
		return errorResult(call.Name, missionID, "validation", "url, proposal_event_id, staging_event_id, or artifact_id is required", false, nil)
	}
	record, ok, err := server.findSourceCandidateReadRecord(ctx, missionID, normalizedURL, proposalEventID, stagingEventID, artifactID)
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	if !ok {
		return errorResult(call.Name, missionID, "not_found", "matching staged source candidate was not found", false, nil)
	}
	output := sourceCandidatesReadOutput{
		ApprovalState:    "unapproved_candidate",
		NotReportDefault: true,
		CandidateURL:     record.URL,
		ProposalEventID:  record.ProposalEventID,
		StagingEventID:   record.StagingEventID,
		StagingState:     record.State,
		Message:          record.Message,
		FailureMessage:   record.FailureMessage,
	}
	if record.State != "staged" {
		if output.Message == "" && record.State == "fetching" {
			output.Message = "후보 원문을 아직 가져오는 중입니다. 잠시 뒤 다시 읽으세요."
		}
		return ToolResult{ToolName: call.Name, MissionID: missionID, Content: output}
	}
	artifact, err := server.service.GetRawArtifact(ctx, record.ArtifactID)
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	if artifact.MissionID != missionID {
		return errorResult(call.Name, missionID, "validation", "staged candidate artifact belongs to another mission", false, []string{record.ArtifactID})
	}
	if pdftext.IsPDFMediaType(artifact.MediaType) || pdftext.IsPDFBytes(artifact.Content) {
		chunk, err := pdftext.ExtractChunk(artifact.Content, input.Offset, input.MaxBytes)
		if err != nil {
			return errorResult(call.Name, missionID, "validation", "PDF text extraction failed: "+err.Error(), false, []string{artifact.ArtifactID})
		}
		artifactOutput := rawArtifactFromApp(artifact)
		output.Artifact = &artifactOutput
		output.Content = chunk.Text
		output.Offset = chunk.Offset
		output.NextOffset = chunk.NextOffset
		output.ContentLength = chunk.ContentLength
		output.ContentLengthKnown = chunk.ContentLengthKnown
		output.Truncated = chunk.Truncated
		output.Extraction = &sourceExtractionOutput{
			Type:               "pdf_text",
			PageCount:          chunk.PageCount,
			TextLength:         chunk.ContentLength,
			TextLengthKnown:    chunk.ContentLengthKnown,
			SuggestedReadBytes: pdftext.DefaultChunkMaxBytes,
			MaxReadBytes:       pdftext.MaxChunkBytes,
		}
		return ToolResult{ToolName: call.Name, MissionID: missionID, Content: output}
	}
	content, offset, nextOffset, truncated, err := boundedArtifactContent(artifact.Content, input.Offset, input.MaxBytes)
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{artifact.ArtifactID})
	}
	artifactOutput := rawArtifactFromApp(artifact)
	output.Artifact = &artifactOutput
	output.Content = content
	output.Offset = offset
	output.NextOffset = nextOffset
	output.ContentLength = len(artifact.Content)
	output.ContentLengthKnown = true
	output.Truncated = truncated
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: output}
}

func (server *Server) findSourceCandidateReadRecord(ctx context.Context, missionID, urlValue, proposalEventID, stagingEventID, artifactID string) (sourceCandidateReadRecord, bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return sourceCandidateReadRecord{}, false, err
	}
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Sequence < events[j].Sequence
	})
	var selected sourceCandidateReadRecord
	var found bool
	matchedURLs := map[string]struct{}{}
	matches := func(record sourceCandidateReadRecord) bool {
		if artifactID != "" && record.ArtifactID != artifactID {
			return false
		}
		if stagingEventID != "" && record.StagingEventID != stagingEventID {
			return false
		}
		if proposalEventID != "" && record.ProposalEventID != proposalEventID {
			return false
		}
		if urlValue != "" && record.URL != urlValue {
			return false
		}
		return true
	}
	for _, event := range events {
		record, ok := sourceCandidateReadRecordFromEvent(event)
		if !ok || !matches(record) {
			continue
		}
		if proposalEventID != "" && stagingEventID == "" && artifactID == "" && urlValue == "" && record.URL != "" {
			matchedURLs[record.URL] = struct{}{}
			if len(matchedURLs) > 1 {
				return sourceCandidateReadRecord{}, false, fmt.Errorf("%w: proposal_event_id matches multiple source candidates; provide url, staging_event_id, or artifact_id", app.ErrInvalidInput)
			}
		}
		selected = record
		found = true
	}
	return selected, found, nil
}

func sourceCandidateReadRecordFromEvent(event app.LedgerEvent) (sourceCandidateReadRecord, bool) {
	switch event.EventType {
	case "source.candidate.staging_started":
		var payload struct {
			URL             string `json:"url"`
			ProposalEventID string `json:"proposal_event_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return sourceCandidateReadRecord{}, false
		}
		return sourceCandidateReadRecord{
			State:           "fetching",
			URL:             strings.TrimSpace(payload.URL),
			ProposalEventID: strings.TrimSpace(payload.ProposalEventID),
			StagingEventID:  event.EventID,
			Message:         "후보 원문을 아직 가져오는 중입니다. 완료되면 같은 도구로 본문을 읽을 수 있습니다.",
			Sequence:        event.Sequence,
		}, true
	case "source.candidate.staged":
		var payload struct {
			URL             string `json:"url"`
			ProposalEventID string `json:"proposal_event_id"`
			StagingEventID  string `json:"staging_event_id"`
			ArtifactID      string `json:"artifact_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return sourceCandidateReadRecord{}, false
		}
		return sourceCandidateReadRecord{
			State:           "staged",
			URL:             strings.TrimSpace(payload.URL),
			ProposalEventID: strings.TrimSpace(payload.ProposalEventID),
			StagingEventID:  strings.TrimSpace(payload.StagingEventID),
			ArtifactID:      strings.TrimSpace(payload.ArtifactID),
			Message:         "이 본문은 미승인 소스 후보입니다. 대화와 조사에서는 참고할 수 있지만, 승인된 source snapshot이나 기본 report 입력은 아닙니다.",
			Sequence:        event.Sequence,
		}, true
	case "source.candidate.staging_failed":
		var payload struct {
			URL             string `json:"url"`
			ProposalEventID string `json:"proposal_event_id"`
			StagingEventID  string `json:"staging_event_id"`
			Message         string `json:"message"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return sourceCandidateReadRecord{}, false
		}
		return sourceCandidateReadRecord{
			State:           "staging_failed",
			URL:             strings.TrimSpace(payload.URL),
			ProposalEventID: strings.TrimSpace(payload.ProposalEventID),
			StagingEventID:  strings.TrimSpace(payload.StagingEventID),
			Message:         "후보 원문 가져오기가 실패했습니다. URL을 다시 확인하거나 다른 후보를 제안하세요.",
			FailureMessage:  strings.TrimSpace(payload.Message),
			Sequence:        event.Sequence,
		}, true
	default:
		return sourceCandidateReadRecord{}, false
	}
}

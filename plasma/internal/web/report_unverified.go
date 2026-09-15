package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentcapability"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
)

const unverifiedReportMaxBytes = 4 * 1024 * 1024

func (server *Server) createUnverifiedReportDraft(ctx context.Context, missionID string, req reportexecution.DraftRequest, pendingEventID string) error {
	req = reportexecution.NormalizeDraftRequest(req)
	if req.PipelineFamily != reportpipeline.Unverified {
		return fmt.Errorf("unsupported unverified report pipeline family")
	}
	if completed, err := server.completeExistingUnverifiedReport(
		ctx,
		missionID,
		pendingEventID,
	); err != nil || completed {
		return err
	}
	executor := server.agentExecutor("codex")
	if executor == nil {
		return fmt.Errorf("unverified reports require a Codex executor")
	}
	executor = agentExecutorWithCapabilityProfile(executor, agentcapability.ReportUnverified())
	mission, err := server.service.GetProjection(ctx, missionID)
	if err != nil {
		return err
	}
	toolSessionID := newID("ses")
	prompt := unverifiedReportPrompt(missionID, toolSessionID, req.Title, mission.Objective, req.DirectionHint)
	started := time.Now()
	result, err := executor.Run(ctx, AgentRequest{
		UserText:        "write unverified markdown report",
		Prompt:          prompt,
		Model:           "gpt-5.6-luna",
		ReasoningEffort: "xhigh",
		MissionID:       missionID,
		ToolSessionID:   toolSessionID,
		AgentExecutor:   "codex",
		MCPMode:         "source_read_only",
		ExtraMCPTools:   []string{mcptools.ToolSourcesList, mcptools.ToolSourcesRead},
		ReplaceMCPTools: true,
	})
	durationMS := time.Since(started).Milliseconds()
	if err != nil {
		return reportAgentFailure(err, result, "report_unverified", durationMS, "")
	}
	content := []byte(result.Text)
	if strings.TrimSpace(result.Text) == "" {
		return reportAgentFailure(fmt.Errorf("unverified report provider returned empty Markdown"), result, "report_unverified", durationMS, "")
	}
	if len(content) > unverifiedReportMaxBytes {
		return reportAgentFailure(fmt.Errorf("unverified report exceeded the artifact size limit"), result, "report_unverified", durationMS, "")
	}
	producerID := strings.TrimSpace(result.SessionID)
	if producerID == "" {
		producerID = toolSessionID
	}
	producer := ledger.Producer{Type: "agent_session", ID: producerID}
	artifact, terminal, created, err := server.service.CreateMarkdownReportArtifactIfOpen(ctx, missionID, pendingEventID, artifactcontract.CreateRequest{
		ArtifactID: unverifiedReportArtifactID(missionID, pendingEventID), MissionID: missionID,
		MediaType: "text/markdown; charset=utf-8", Filename: safeFilename(req.Title, ".md"),
		Producer: producer, Content: content,
	}, func(artifact artifactcontract.Raw) ledger.AppendRequest {
		request := reporting.BuildCLIMarkdownReportArtifactCreatedAppendRequest(reporting.CLIMarkdownReportArtifactCreatedEventRequest{
			EventID: newID("evt"), MissionID: missionID, PendingEventID: pendingEventID,
			Title: req.Title, Artifact: artifact, AgentExecutor: "codex",
			AgentModel: "gpt-5.6-luna", AgentReasoningEffort: "xhigh",
			AgentSelectionSource: "unverified_fixed", AgentSessionID: result.SessionID,
			ToolSessionID: toolSessionID, MCPMode: "source_read_only",
			ReportMode: reportexecution.ModePlanned, PipelineFamily: reportpipeline.Unverified,
			ReportSessionPolicy:          reportexecution.SessionPolicyFreshSession,
			ReportSessionPolicySelection: "unverified_fixed", PostReportHumanize: "disabled",
			HumanizeEnabled: false, SessionChainKind: "fresh_unverified_report",
			ReportSessionID: result.SessionID, CompositionStrategy: "unverified_single_call",
			DurationMS: durationMS, AgentUsage: result.Usage, AgentUsageSurface: "report_unverified",
			AgentUsageDurationMS: durationMS, AgentResumed: result.Resumed, Producer: producer,
		})
		return request
	})
	if err != nil {
		if server.hasReportDraftTerminalEvent(ctx, missionID, pendingEventID) {
			return nil
		}
		return err
	}
	if !created {
		if server.hasReportDraftTerminalEvent(ctx, missionID, pendingEventID) {
			return nil
		}
		return fmt.Errorf("unverified report artifact was already claimed")
	}
	if string(artifact.Content) != result.Text {
		return fmt.Errorf("stored unverified report differs from provider output")
	}
	_, err = reportrun.CompleteReportRun(ctx, server.service, reportrun.ReportCompletionRequest{MissionID: missionID, CanonicalEventID: terminal.EventID})
	return err
}

func (server *Server) completeExistingUnverifiedReport(
	ctx context.Context,
	missionID,
	pendingEventID string,
) (bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return false, err
	}
	for _, event := range events {
		if event.EventType != "report.artifact.created" {
			continue
		}
		var payload struct {
			PendingID      string `json:"pending_event_id"`
			PipelineFamily string `json:"pipeline_family"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil ||
			strings.TrimSpace(payload.PendingID) !=
				strings.TrimSpace(pendingEventID) ||
			strings.TrimSpace(payload.PipelineFamily) !=
				reportpipeline.Unverified {
			continue
		}
		_, err := reportrun.CompleteReportRun(
			ctx,
			server.service,
			reportrun.ReportCompletionRequest{
				MissionID:        missionID,
				CanonicalEventID: event.EventID,
			},
		)
		return true, err
	}
	pendingID := strings.TrimSpace(pendingEventID)
	if _, closed := reportexecution.CompletedPendingEventIDs(events)[pendingID]; closed {
		return true, nil
	}
	return false, nil
}

func unverifiedReportArtifactID(missionID, pendingEventID string) string {
	sum := sha256.Sum256([]byte(
		strings.TrimSpace(missionID) + "\x00" +
			strings.TrimSpace(pendingEventID),
	))
	return "art_unverified_" + hex.EncodeToString(sum[:16])
}

func unverifiedReportPrompt(missionID, toolSessionID, title, objective, direction string) string {
	return fmt.Sprintf(`Write one complete Markdown report now.

Use the topic and direction below. The connected mission materials are available through plasma.sources.list and plasma.sources.read. Decide for yourself what to read and how to write the report.

Return only the finished Markdown report. Do not return a plan, schema, validation result, authoring commentary, or a description of your process.

mission_id: %s
session_id: %s
producer: {"type":"agent_session","id":"%s"}
title: %s
objective: %s
direction: %s
`, strings.TrimSpace(missionID), strings.TrimSpace(toolSessionID), strings.TrimSpace(toolSessionID), strings.TrimSpace(title), strings.TrimSpace(objective), strings.TrimSpace(direction))
}

package web

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportilphase0"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func reportILRetryPendingLineage(events []ledger.Event, pendingID string) []string {
	parents := map[string]string{}
	for _, event := range events {
		if event.EventType != "report.draft.pending" {
			continue
		}
		var payload struct {
			RetryOf string `json:"retry_of_pending_event_id"`
		}
		_ = json.Unmarshal(event.Payload, &payload)
		parents[event.EventID] = strings.TrimSpace(payload.RetryOf)
	}
	lineage := []string{}
	seen := map[string]bool{}
	for current, depth := pendingID, 0; current != "" && depth < 64; depth++ {
		if seen[current] {
			return nil
		}
		seen[current] = true
		lineage = append(lineage, current)
		current = parents[current]
	}
	return lineage
}

func (server *Server) createExperimentalReportDraft(ctx context.Context, missionID string, req reportexecution.DraftRequest, pendingEventID string) error {
	req = reportexecution.NormalizeDraftRequest(req)
	if req.PipelineFamily != reportilcontract.PipelineFamily {
		return fmt.Errorf("unsupported report IL pipeline family")
	}
	req.AgentExecutor, req.AgentModel, req.AgentReasoningEffort, req.PostReportHumanize = "codex", "gpt-5.6-luna", "xhigh", "disabled"
	executor := server.agentExecutor("codex")
	if executor == nil {
		return fmt.Errorf("report IL requires a Codex executor")
	}
	progress := func(stage, status string) error {
		return server.service.AppendReportILProgress(ctx, missionID, pendingEventID, stage, status)
	}
	chromePath, err := server.resolveChromePath()
	if err != nil {
		return reportexecution.NewStageFailure("il_render", "", -1, -1, err)
	}
	mission, err := server.service.GetProjection(ctx, missionID)
	if err != nil {
		return reportexecution.NewStageFailure("source_packet", "", -1, -1, err)
	}
	imageFetch := func(ctx context.Context, rawURL string) (reportilphase0.ProductImage, error) {
		fetched, err := server.fetchMedia(ctx, rawURL)
		if err != nil {
			return reportilphase0.ProductImage{}, err
		}
		image := reportilphase0.ProductImage{MediaType: fetched.MediaType, Content: fetched.Content, Width: fetched.Width, Height: fetched.Height}
		if err := image.Validate(); err != nil {
			return reportilphase0.ProductImage{}, err
		}
		return image, nil
	}
	authoringMode := reportilphase0.AuthoringModeStandard
	if req.ReportMode == reportexecution.ModeLongForm {
		authoringMode = reportilphase0.AuthoringModeLongForm
	}
	longFormProgress := func(event reportilphase0.LongFormProgressEvent) error {
		return server.service.AppendReportILLongFormProgress(
			ctx, missionID, pendingEventID, event.Kind, event.Status,
			event.Plan, event.PartIndex, event.SectionIndex, event.Title,
		)
	}
	var resume *reportilcontract.ResumeCheckpoint
	if req.RetryStrategy == "resume_failed" {
		events, listErr := server.service.ListEvents(ctx, missionID)
		if listErr != nil {
			err = listErr
		} else {
			for _, attemptID := range reportILRetryPendingLineage(events, req.RetryOfPendingEventID) {
				resume, err = server.service.LoadReportILResumeCheckpoint(ctx, missionID, attemptID)
				if err == nil {
					break
				}
				if !strings.Contains(err.Error(), "no durable checkpoint") {
					break
				}
			}
			if resume == nil && err != nil && strings.Contains(err.Error(), "no durable checkpoint") {
				resume, err = reportilphase0.RecoverPartsCheckpoint(
					ctx, missionID, req.RetryOfPendingEventID, events,
					server.service.ReportILSourceReader(), server.service,
				)
				if err == nil {
					if writeErr := server.service.AppendReportILCheckpoint(ctx, missionID, resume.ProductCheckpoint); writeErr != nil {
						err = fmt.Errorf("append recovered Part checkpoint: %w", writeErr)
					}
				} else if strings.Contains(err.Error(), "not recoverable") {
					resume, err = reportilphase0.RecoverLegacyCheckpoint(
						ctx, missionID, req.RetryOfPendingEventID, events,
						server.service.ReportILSourceReader(), server.service,
					)
				}
			}
		}
		if err != nil {
			return reportexecution.NewStageFailure("source_packet", "", -1, -1, fmt.Errorf("load report IL checkpoint: %w", err))
		}
	}
	checkpoint := func(value reportilcontract.ProductCheckpoint) error {
		if err := server.service.AppendReportILCheckpoint(ctx, missionID, value); err != nil {
			return fmt.Errorf("append report IL checkpoint: %w", err)
		}
		return nil
	}
	bundle, err := reportilphase0.RunProduct(ctx, reportilphase0.ProductConfig{MissionID: missionID, MissionObjective: mission.Objective, Title: req.Title, Direction: req.DirectionHint, TargetLanguage: "ko", AuthoringMode: authoringMode, ValidationProfile: req.RigorLevel, ChromePath: chromePath, NewID: newID, Sources: server.service.ReportILSourceReader(), Provider: executor, Progress: progress, LongFormProgress: longFormProgress, PendingEventID: pendingEventID, VerifySourceRead: server.service, AuthorDocuments: server.service, ImageFetch: imageFetch, Resume: resume, Checkpoint: checkpoint})
	if err != nil {
		return err
	}
	if err := progress("il_store", "started"); err != nil {
		return reportexecution.NewStageFailure("il_store", "", -1, -1, err)
	}
	artifacts := make([]reportilcontract.ArtifactInput, 0, len(bundle.Artifacts))
	entries := make([]reportilcontract.TerminalArtifactEntry, 0, len(bundle.Artifacts))
	markdownID := ""
	for _, artifact := range bundle.Artifacts {
		if artifact.Kind == "markdown" {
			markdownID = artifact.ID
		}
		artifacts = append(artifacts, reportilcontract.ArtifactInput{ArtifactID: artifact.ID, MissionID: missionID, MediaType: artifact.MediaType, Filename: artifact.Filename, Producer: ledger.Producer{Type: "agent", ID: "codex"}, Content: artifact.Content, ExpectedSHA256: artifact.SHA256})
		entries = append(entries, reportilcontract.TerminalArtifactEntry{ArtifactID: artifact.ID, Kind: artifact.Kind, MediaType: artifact.MediaType, SHA256: artifact.SHA256, ByteSize: artifact.ByteSize, Role: artifact.Role, Filename: artifact.Filename, AssetID: artifact.AssetID})
	}
	if markdownID == "" {
		return reportexecution.NewStageFailure("il_store", "", -1, -1, fmt.Errorf("markdown artifact missing"))
	}
	lineage := reportilcontract.TerminalArtifactLineage{
		PipelineFamily:     reportilcontract.PipelineFamily,
		Artifacts:          entries,
		MarkdownArtifactID: markdownID,
		SourceSelection: &reportilcontract.TerminalSourceSelectionSummary{
			Applied:                 bundle.SourceSelection.Applied,
			AcceptedSources:         bundle.SourceSelection.AcceptedSources,
			UsableSources:           bundle.SourceSelection.UsableSources,
			SelectedSources:         bundle.SourceSelection.SelectedSources,
			SupplementalSources:     bundle.SourceSelection.SupplementalSources,
			ExcludedUnusableSources: bundle.SourceSelection.ExcludedUnusableSources,
			ExcludedBudgetSources:   bundle.SourceSelection.ExcludedBudgetSources,
		},
	}
	terminalPayload := map[string]any{"kind": "markdown_report_artifact", "pending_event_id": pendingEventID, "artifact_id": markdownID, "media_type": "text/markdown; charset=utf-8", "title": req.Title, "agent_executor": "codex", "agent_model": "gpt-5.6-luna", "agent_reasoning_effort": "xhigh", "report_mode": req.ReportMode, "pipeline_family": reportilcontract.PipelineFamily, "rigor_level": req.RigorLevel, "rigor_label": req.RigorLabel, "post_report_humanize": "disabled", "humanize_enabled": false, "artifact_bundle": lineage, "text": "IL 보고서가 생성되었습니다."}
	terminal := reportilcontract.EventInput{EventID: newID("evt"), MissionID: missionID, EventType: "report.artifact.created", CausationEventID: pendingEventID, CorrelationID: pendingEventID, Producer: ledger.Producer{Type: "agent", ID: "codex"}, Payload: mustJSON(terminalPayload)}
	storeCompleted := reportilcontract.EventInput{EventID: newID("evt"), MissionID: missionID, EventType: "report.il_store.completed", CausationEventID: pendingEventID, CorrelationID: pendingEventID, Producer: ledger.Producer{Type: "system", ID: "report-il"}, Payload: mustJSON(map[string]any{"kind": "report_il_stage_progress", "pending_event_id": pendingEventID, "pipeline_family": reportilcontract.PipelineFamily, "stage": "il_store", "status": "completed"})}
	result, err := server.service.CreateReportILBundleIfOpenContract(ctx, reportilcontract.BundleRequest{MissionID: missionID, PendingID: pendingEventID, Artifacts: artifacts, StoreCompleted: storeCompleted, Terminal: terminal})
	if err != nil {
		return reportexecution.NewStageFailure("il_store", "", -1, -1, fmt.Errorf("store report IL bundle: %w", err))
	}
	if !result.Created {
		if server.hasReportDraftTerminalEvent(ctx, missionID, pendingEventID) {
			return nil
		}
		return reportexecution.NewStageFailure("il_store", "", -1, -1, fmt.Errorf("report IL bundle was already claimed"))
	}
	_, err = reporting.CompleteReportRun(ctx, server.service, reporting.ReportCompletionRequest{MissionID: missionID, CanonicalEventID: result.Terminal.EventID})
	return err
}

func (server *Server) resolveChromePath() (string, error) {
	return resolveChromePath(server.reportILChromePath, os.Getenv("PLASMA_CHROME_PATH"), exec.LookPath, os.Stat, runtime.GOOS)
}

func resolveChromePath(explicit, env string, lookPath func(string) (string, error), stat func(string) (os.FileInfo, error), goos string) (string, error) {
	explicit = strings.TrimSpace(explicit)
	if explicit == "" {
		explicit = strings.TrimSpace(env)
	}
	if explicit != "" {
		if !isChromeExecutable(explicit, stat) {
			return "", fmt.Errorf("Chrome renderer is unavailable")
		}
		return explicit, nil
	}
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser"} {
		path, err := lookPath(name)
		if err != nil || !isChromeExecutable(path, stat) {
			continue
		}
		return path, nil
	}
	if goos == "darwin" {
		path := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
		if isChromeExecutable(path, stat) {
			return path, nil
		}
	}
	return "", fmt.Errorf("Chrome renderer is unavailable")
}

func isChromeExecutable(path string, stat func(string) (os.FileInfo, error)) bool {
	info, err := stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&0o111 != 0
}

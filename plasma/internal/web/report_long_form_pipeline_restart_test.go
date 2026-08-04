package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestReaderStyleGateRestartReusesOpenReaderBinding(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := openW4BRestartService(t, ctx, dbPath)
	req := seedW4BRestartFixture(t, ctx, svc, reporting.FinalEditHumanizeDisabled)
	readerBinding := w4BReaderBinding(req, "provider-reader-open")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_w4b_open_reader_start", readerBinding); err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	closeStore()

	executor := &w4BRestartExecutor{}
	reopened, closeReopened, _ := runW4BRestartPipeline(t, ctx, dbPath, req, executor)
	defer closeReopened()
	readerRequests := w4BStageRequests(executor.requests, reporting.FinalEditStageReader)
	if len(readerRequests) != 1 || readerRequests[0].FinalEditStage == nil || *readerRequests[0].FinalEditStage != readerBinding {
		t.Fatalf("open reader did not reuse binding: %#v", readerRequests)
	}
	events := w4BEvents(t, ctx, reopened, req.missionID)
	assertW4BEventCount(t, events, reporting.FinalEditReaderStartedEventType, 1)
	assertW4BEventCount(t, events, reporting.FinalEditReaderSubmittedEventType, 1)
	assertW4BEventCount(t, events, "report.artifact.created", 1)
}

func TestReaderStyleGateRestartSkipsSubmittedReader(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := openW4BRestartService(t, ctx, dbPath)
	req := seedW4BRestartFixture(t, ctx, svc, reporting.FinalEditHumanizeDisabled)
	reader := w4BSubmitReader(t, ctx, svc, req, "submitted_reader")
	closeStore()

	executor := &w4BRestartExecutor{}
	reopened, closeReopened, _ := runW4BRestartPipeline(t, ctx, dbPath, req, executor)
	defer closeReopened()
	if got := len(w4BStageRequests(executor.requests, reporting.FinalEditStageReader)); got != 0 {
		t.Fatalf("submitted reader reran provider %d time(s)", got)
	}
	gateRequests := w4BStageRequests(executor.requests, reporting.FinalEditStageGate)
	if got := len(gateRequests); got != 1 {
		t.Fatalf("gate provider calls=%d, want 1", got)
	}
	if gateRequests[0].FinalEditStage.SourceArtifactID != reader.Artifact.ArtifactID {
		t.Fatalf("gate source=%s, want submitted reader artifact %s", gateRequests[0].FinalEditStage.SourceArtifactID, reader.Artifact.ArtifactID)
	}
	events := w4BEvents(t, ctx, reopened, req.missionID)
	assertW4BEventCount(t, events, reporting.FinalEditReaderStartedEventType, 1)
	assertW4BEventCount(t, events, reporting.FinalEditReaderSubmittedEventType, 1)
	assertW4BEventCount(t, events, "report.artifact.created", 1)
}

func TestReaderStyleGateRestartSkipsSubmittedStyle(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := openW4BRestartService(t, ctx, dbPath)
	req := seedW4BRestartFixture(t, ctx, svc, reporting.FinalEditHumanizeEnabled)
	reader := w4BSubmitReader(t, ctx, svc, req, "submitted_style")
	style := w4BSubmitStyle(t, ctx, svc, req, reader, "submitted_style")
	closeStore()

	executor := &w4BRestartExecutor{}
	reopened, closeReopened, _ := runW4BRestartPipeline(t, ctx, dbPath, req, executor)
	defer closeReopened()
	if got := len(w4BStageRequests(executor.requests, reporting.FinalEditStageReader)); got != 0 {
		t.Fatalf("submitted reader reran provider %d time(s)", got)
	}
	if got := len(w4BStageRequests(executor.requests, reporting.FinalEditStageStyle)); got != 0 {
		t.Fatalf("submitted style reran provider %d time(s)", got)
	}
	gateRequests := w4BStageRequests(executor.requests, reporting.FinalEditStageGate)
	if got := len(gateRequests); got != 1 {
		t.Fatalf("gate provider calls=%d, want 1", got)
	}
	if gateRequests[0].FinalEditStage.SourceArtifactID != style.Artifact.ArtifactID {
		t.Fatalf("gate source=%s, want submitted style artifact %s", gateRequests[0].FinalEditStage.SourceArtifactID, style.Artifact.ArtifactID)
	}
	events := w4BEvents(t, ctx, reopened, req.missionID)
	assertW4BEventCount(t, events, reporting.FinalEditReaderSubmittedEventType, 1)
	assertW4BEventCount(t, events, reporting.FinalEditStyleSubmittedEventType, 1)
	assertW4BEventCount(t, events, "report.artifact.created", 1)
}

func TestReaderStyleGateRestartResumesSubmittedGateWithoutCanonical(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := openW4BRestartService(t, ctx, dbPath)
	req := seedW4BRestartFixture(t, ctx, svc, reporting.FinalEditHumanizeDisabled)
	reader := w4BSubmitReader(t, ctx, svc, req, "submitted_gate")
	gateBinding, _ := w4BGateBinding(req, reader.Artifact.ArtifactID, "provider-corrective-gate-submitted")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_w4b_submitted_gate_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	w4BAppendGateSubmission(t, ctx, svc, gateBinding, "evt_w4b_submitted_gate_submit")
	closeStore()

	executor := &w4BRestartExecutor{}
	reopened, closeReopened, result := runW4BRestartPipeline(t, ctx, dbPath, req, executor)
	defer closeReopened()
	if len(executor.requests) != 0 {
		t.Fatalf("submitted gate reran provider requests: %#v", executor.requests)
	}
	events := w4BEvents(t, ctx, reopened, req.missionID)
	assertW4BEventCount(t, events, reporting.FinalEditGateSubmittedEventType, 1)
	assertW4BEventCount(t, events, "report.artifact.created", 1)
	canonical := w4BCanonicalEvent(t, events)
	artifact := w4BResultArtifact(t, result)
	event := w4BResultEvent(t, result)
	payload := w4BPayload(t, canonical)
	if artifact.ArtifactID != reader.Artifact.ArtifactID || event.EventID != canonical.EventID ||
		payload["artifact_id"] != artifact.ArtifactID ||
		payload["planned_final_artifact_id"] != req.artifactID ||
		payload["final_edit_gate_event_id"] != "evt_w4b_submitted_gate_submit" ||
		payload["artifact_sha256"] != artifact.SHA256 {
		t.Fatalf("resumed canonical lineage mismatch artifact=%#v event=%#v payload=%#v", artifact, event, payload)
	}
}

func TestReaderStyleGateRestartReturnsExistingCanonicalWithoutProvider(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := openW4BRestartService(t, ctx, dbPath)
	req := seedW4BRestartFixture(t, ctx, svc, reporting.FinalEditHumanizeDisabled)
	reader := w4BSubmitReader(t, ctx, svc, req, "existing_canonical")
	gateBinding, finalBinding := w4BGateBinding(req, reader.Artifact.ArtifactID, "provider-corrective-gate-canonical")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_w4b_existing_gate_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	source, err := svc.GetRawArtifact(ctx, gateBinding.SourceArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	finalized, err := reporting.SubmitFinalEditGate(ctx, svc, reporting.FinalEditGateSubmitRequest{
		StageBinding:       gateBinding,
		FinalBinding:       finalBinding,
		StageEventID:       "evt_w4b_existing_gate_submit",
		CanonicalEventID:   "evt_w4b_existing_final",
		ManuscriptMarkdown: string(source.Content),
		OperationCount:     0,
	})
	if err != nil {
		t.Fatal(err)
	}
	beforeEvents := len(w4BEvents(t, ctx, svc, req.missionID))
	closeStore()

	executor := &w4BRestartExecutor{}
	reopened, closeReopened, result := runW4BRestartPipeline(t, ctx, dbPath, req, executor)
	defer closeReopened()
	if len(executor.requests) != 0 {
		t.Fatalf("existing canonical reran provider requests: %#v", executor.requests)
	}
	events := w4BEvents(t, ctx, reopened, req.missionID)
	if len(events) != beforeEvents {
		t.Fatalf("existing canonical replay changed event count %d -> %d", beforeEvents, len(events))
	}
	if artifact, event := w4BResultArtifact(t, result), w4BResultEvent(t, result); artifact.ArtifactID != finalized.Artifact.ArtifactID || event.EventID != finalized.Event.EventID {
		t.Fatalf("existing canonical replay identity differs artifact=%#v event=%#v want=%#v/%#v", artifact, event, finalized.Artifact, finalized.Event)
	}
	assertW4BEventCount(t, events, reporting.FinalEditGateSubmittedEventType, 1)
	assertW4BEventCount(t, events, "report.artifact.created", 1)
}

func TestReaderStyleGateRestartRejectsOpenGateAfterCanonicalWithoutProvider(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := openW4BRestartService(t, ctx, dbPath)
	req := seedW4BRestartFixture(t, ctx, svc, reporting.FinalEditHumanizeDisabled)
	reader := w4BSubmitReader(t, ctx, svc, req, "open_gate_terminal")
	gateBinding, _ := w4BGateBinding(req, reader.Artifact.ArtifactID, "provider-corrective-gate-open-terminal")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_w4b_open_gate_terminal_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	w4BAppendTerminalCanonical(t, ctx, svc, req, "evt_w4b_open_gate_terminal_final")
	closeStore()

	reopened, closeReopened := openW4BRestartService(t, ctx, dbPath)
	defer closeReopened()
	executor := &w4BRestartExecutor{service: reopened}
	server := NewServer(reopened, Options{}).(*Server)
	_, err := server.runLongFormReaderStyleGatePipeline(ctx, req, executor)
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("open gate terminal error=%v, want conflict", err)
	}
	if len(executor.requests) != 0 {
		t.Fatalf("open gate terminal exposed provider requests: %#v", executor.requests)
	}
}

func TestAssemblyWriterReaderStyleGateV2RestartAfterAssemblyRunsWriterReaderGate(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := openW4BRestartService(t, ctx, dbPath)
	req := seedW4BRestartFixtureWithPipeline(t, ctx, svc, reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2, reporting.FinalEditHumanizeDisabled)
	writerBinding := w4BWriterBinding(req, "provider-writer-assembly")
	assembly, created, err := reporting.EnsureFinalEditAssembly(ctx, svc, "evt_w4b_v2_assembly_created", writerBinding)
	if err != nil || !created {
		t.Fatalf("assembly created=%t result=%#v err=%v", created, assembly, err)
	}
	closeStore()

	executor := &w4BRestartExecutor{}
	reopened, closeReopened, _ := runW4BRestartPipeline(t, ctx, dbPath, req, executor)
	defer closeReopened()
	assertW4BStageRequestSequence(t, executor.requests, reporting.FinalEditStageWriter, reporting.FinalEditStageReader, reporting.FinalEditStageGate)
	writerRequest := executor.requests[0].FinalEditStage
	readerRequest := executor.requests[1].FinalEditStage
	gateRequest := executor.requests[2].FinalEditStage
	if writerRequest.SourceArtifactID != assembly.Artifact.ArtifactID ||
		writerRequest.FinalEditPipeline != reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2 ||
		writerRequest.PreviousProviderSessionID != req.reportPlanSessionID ||
		writerRequest.ForkSourceAgentSessionID != req.reportPlanSessionID {
		t.Fatalf("writer restart binding mismatch: %#v", writerRequest)
	}
	if readerRequest.SourceArtifactID != assembly.Artifact.ArtifactID ||
		readerRequest.PreviousProviderSessionID != req.reportPlanSessionID ||
		readerRequest.ForkSourceAgentSessionID != req.reportPlanSessionID {
		t.Fatalf("reader restart binding mismatch: %#v", readerRequest)
	}
	if gateRequest.SourceArtifactID != assembly.Artifact.ArtifactID ||
		gateRequest.PreviousProviderSessionID != req.reportPlanSessionID ||
		gateRequest.ForkSourceAgentSessionID != req.reportPlanSessionID {
		t.Fatalf("gate restart binding mismatch: %#v", gateRequest)
	}
	events := w4BEvents(t, ctx, reopened, req.missionID)
	assertW4BEventCount(t, events, reporting.FinalEditAssemblyCreatedEventType, 1)
	assertW4BEventCount(t, events, reporting.FinalEditWriterSubmittedEventType, 1)
	assertW4BEventCount(t, events, reporting.FinalEditReaderSubmittedEventType, 1)
	assertW4BEventCount(t, events, reporting.FinalEditGateSubmittedEventType, 1)
	assertW4BEventCount(t, events, "report.artifact.created", 1)
}

func TestAssemblyWriterReaderStyleGateV2RestartSkipsSubmittedWriter(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := openW4BRestartService(t, ctx, dbPath)
	req := seedW4BRestartFixtureWithPipeline(t, ctx, svc, reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2, reporting.FinalEditHumanizeDisabled)
	writer := w4BSubmitWriter(t, ctx, svc, req, "submitted_writer", true)
	closeStore()

	executor := &w4BRestartExecutor{}
	reopened, closeReopened, _ := runW4BRestartPipeline(t, ctx, dbPath, req, executor)
	defer closeReopened()
	if got := len(w4BStageRequests(executor.requests, reporting.FinalEditStageWriter)); got != 0 {
		t.Fatalf("submitted writer reran provider %d time(s)", got)
	}
	readerRequests := w4BStageRequests(executor.requests, reporting.FinalEditStageReader)
	gateRequests := w4BStageRequests(executor.requests, reporting.FinalEditStageGate)
	if len(readerRequests) != 1 || len(gateRequests) != 1 {
		t.Fatalf("expected reader+gate after submitted writer, requests=%#v", executor.requests)
	}
	if readerRequests[0].FinalEditStage.SourceArtifactID != writer.Artifact.ArtifactID {
		t.Fatalf("reader source=%s, want writer artifact %s", readerRequests[0].FinalEditStage.SourceArtifactID, writer.Artifact.ArtifactID)
	}
	events := w4BEvents(t, ctx, reopened, req.missionID)
	assertW4BEventCount(t, events, reporting.FinalEditAssemblyCreatedEventType, 1)
	assertW4BEventCount(t, events, reporting.FinalEditWriterSubmittedEventType, 1)
	assertW4BEventCount(t, events, "report.artifact.created", 1)
}

func TestAssemblyWriterReaderStyleGateV2RestartSkipsSubmittedReader(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := openW4BRestartService(t, ctx, dbPath)
	req := seedW4BRestartFixtureWithPipeline(t, ctx, svc, reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2, reporting.FinalEditHumanizeDisabled)
	writer := w4BSubmitWriter(t, ctx, svc, req, "submitted_reader", true)
	reader := w4BSubmitV2Reader(t, ctx, svc, req, writer, "submitted_reader", true)
	closeStore()

	executor := &w4BRestartExecutor{}
	reopened, closeReopened, _ := runW4BRestartPipeline(t, ctx, dbPath, req, executor)
	defer closeReopened()
	if got := len(w4BStageRequests(executor.requests, reporting.FinalEditStageWriter)); got != 0 {
		t.Fatalf("submitted writer reran provider %d time(s)", got)
	}
	if got := len(w4BStageRequests(executor.requests, reporting.FinalEditStageReader)); got != 0 {
		t.Fatalf("submitted reader reran provider %d time(s)", got)
	}
	gateRequests := w4BStageRequests(executor.requests, reporting.FinalEditStageGate)
	if len(gateRequests) != 1 || gateRequests[0].FinalEditStage.SourceArtifactID != reader.Artifact.ArtifactID {
		t.Fatalf("gate did not consume submitted reader artifact: requests=%#v reader=%#v", gateRequests, reader.Artifact)
	}
	events := w4BEvents(t, ctx, reopened, req.missionID)
	assertW4BEventCount(t, events, reporting.FinalEditWriterSubmittedEventType, 1)
	assertW4BEventCount(t, events, reporting.FinalEditReaderSubmittedEventType, 1)
	assertW4BEventCount(t, events, reporting.FinalEditGateSubmittedEventType, 1)
	assertW4BEventCount(t, events, "report.artifact.created", 1)
}

func TestAssemblyWriterReaderStyleGateV2RestartSkipsSubmittedStyle(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := openW4BRestartService(t, ctx, dbPath)
	req := seedW4BRestartFixtureWithPipeline(t, ctx, svc, reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2, reporting.FinalEditHumanizeEnabled)
	writer := w4BSubmitWriter(t, ctx, svc, req, "submitted_style", true)
	reader := w4BSubmitV2Reader(t, ctx, svc, req, writer, "submitted_style", true)
	style := w4BSubmitStyle(t, ctx, svc, req, reader, "submitted_style")
	closeStore()

	executor := &w4BRestartExecutor{}
	reopened, closeReopened, _ := runW4BRestartPipeline(t, ctx, dbPath, req, executor)
	defer closeReopened()
	if len(w4BStageRequests(executor.requests, reporting.FinalEditStageWriter)) != 0 ||
		len(w4BStageRequests(executor.requests, reporting.FinalEditStageReader)) != 0 ||
		len(w4BStageRequests(executor.requests, reporting.FinalEditStageStyle)) != 0 {
		t.Fatalf("submitted writer/reader/style reran provider requests: %#v", executor.requests)
	}
	gateRequests := w4BStageRequests(executor.requests, reporting.FinalEditStageGate)
	if len(gateRequests) != 1 || gateRequests[0].FinalEditStage.SourceArtifactID != style.Artifact.ArtifactID {
		t.Fatalf("gate did not consume submitted style artifact: requests=%#v style=%#v", gateRequests, style.Artifact)
	}
	if gateRequests[0].FinalEditStage.PreviousProviderSessionID != req.reportPlanSessionID ||
		gateRequests[0].FinalEditStage.ForkSourceAgentSessionID != req.reportPlanSessionID {
		t.Fatalf("v2 gate did not use plan-sibling ancestry: %#v", gateRequests[0].FinalEditStage)
	}
	events := w4BEvents(t, ctx, reopened, req.missionID)
	assertW4BEventCount(t, events, reporting.FinalEditStyleSubmittedEventType, 1)
	assertW4BEventCount(t, events, reporting.FinalEditGateSubmittedEventType, 1)
	assertW4BEventCount(t, events, "report.artifact.created", 1)
}

func TestAssemblyWriterReaderStyleGateV2RestartResumesSubmittedGateWithoutCanonical(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := openW4BRestartService(t, ctx, dbPath)
	req := seedW4BRestartFixtureWithPipeline(t, ctx, svc, reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2, reporting.FinalEditHumanizeDisabled)
	writer := w4BSubmitWriter(t, ctx, svc, req, "submitted_gate", true)
	reader := w4BSubmitV2Reader(t, ctx, svc, req, writer, "submitted_gate", true)
	gateBinding, _ := w4BV2GateBinding(req, reader.Artifact.ArtifactID, "provider-corrective-gate-v2-submitted")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_w4b_v2_submitted_gate_start", gateBinding); err != nil || !created {
		t.Fatalf("v2 gate start created=%t err=%v", created, err)
	}
	w4BAppendGateSubmission(t, ctx, svc, gateBinding, "evt_w4b_v2_submitted_gate_submit")
	closeStore()

	executor := &w4BRestartExecutor{}
	reopened, closeReopened, result := runW4BRestartPipeline(t, ctx, dbPath, req, executor)
	defer closeReopened()
	if len(executor.requests) != 0 {
		t.Fatalf("submitted v2 gate reran provider requests: %#v", executor.requests)
	}
	events := w4BEvents(t, ctx, reopened, req.missionID)
	assertW4BEventCount(t, events, reporting.FinalEditGateSubmittedEventType, 1)
	assertW4BEventCount(t, events, "report.artifact.created", 1)
	canonical := w4BCanonicalEvent(t, events)
	artifact := w4BResultArtifact(t, result)
	event := w4BResultEvent(t, result)
	payload := w4BPayload(t, canonical)
	if artifact.ArtifactID != reader.Artifact.ArtifactID || event.EventID != canonical.EventID ||
		payload["final_edit_pipeline"] != reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2 ||
		payload["artifact_id"] != artifact.ArtifactID ||
		payload["planned_final_artifact_id"] != req.artifactID ||
		payload["final_edit_gate_event_id"] != "evt_w4b_v2_submitted_gate_submit" ||
		payload["artifact_sha256"] != artifact.SHA256 {
		t.Fatalf("resumed v2 canonical lineage mismatch artifact=%#v event=%#v payload=%#v", artifact, event, payload)
	}
}

type w4BRestartExecutor struct {
	service  *app.Service
	requests []AgentRequest
	forks    int
}

func (executor *w4BRestartExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	executor.requests = append(executor.requests, req)
	if req.FinalEditStage == nil {
		return AgentResult{Text: "OK", SessionID: req.PreviousSessionID}, nil
	}
	binding := *req.FinalEditStage
	if _, _, err := reporting.StartFinalEditStage(ctx, executor.service, fmt.Sprintf("evt_w4b_restart_start_%d", len(executor.requests)), binding); err != nil {
		return AgentResult{}, err
	}
	source, err := executor.service.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return AgentResult{}, err
	}
	if binding.Stage == reporting.FinalEditStageGate {
		if req.LongFormFinalize == nil {
			return AgentResult{}, fmt.Errorf("gate run missing final binding")
		}
		markdown := string(source.Content)
		comparison, err := reporting.FinalEditSemanticComparison(ctx, executor.service, binding, markdown)
		if err != nil {
			return AgentResult{}, err
		}
		semanticAcceptance := make([]reporting.FinalEditSemanticAcceptance, 0, len(comparison))
		for _, item := range comparison {
			semanticAcceptance = append(semanticAcceptance, reporting.FinalEditSemanticAcceptance{
				ParagraphOrdinal:      item.ParagraphOrdinal,
				FinalParagraphOrdinal: item.ParagraphOrdinal,
				Verdict:               reporting.FinalEditSemanticAcceptedEquivalent,
			})
		}
		_, err = reporting.SubmitFinalEditGate(ctx, executor.service, reporting.FinalEditGateSubmitRequest{
			StageBinding:       binding,
			FinalBinding:       *req.LongFormFinalize,
			StageEventID:       fmt.Sprintf("evt_w4b_restart_submit_%d", len(executor.requests)),
			CanonicalEventID:   fmt.Sprintf("evt_w4b_restart_final_%d", len(executor.requests)),
			ManuscriptMarkdown: markdown,
			OperationCount:     0,
			SemanticAcceptance: semanticAcceptance,
		})
		if err != nil {
			return AgentResult{}, err
		}
		return AgentResult{Text: finalEditGateSubmittedSentinel, SessionID: binding.ProviderSessionID}, nil
	}
	if binding.Stage == reporting.FinalEditStageStyleSemanticValidation {
		comparison, err := reporting.FinalEditSemanticComparison(ctx, executor.service, binding, string(source.Content))
		if err != nil {
			return AgentResult{}, err
		}
		semanticAcceptance := make([]reporting.FinalEditSemanticAcceptance, 0, len(comparison))
		for _, item := range comparison {
			semanticAcceptance = append(semanticAcceptance, reporting.FinalEditSemanticAcceptance{
				ParagraphOrdinal: item.ParagraphOrdinal,
				Verdict:          reporting.FinalEditSemanticAcceptedEquivalent,
			})
		}
		if _, err := reporting.SubmitFinalEditStyleSemanticValidation(ctx, executor.service, binding, fmt.Sprintf("evt_w4b_restart_submit_%d", len(executor.requests)), semanticAcceptance); err != nil {
			return AgentResult{}, err
		}
		return AgentResult{Text: finalEditStageSubmittedSentinel, SessionID: binding.ProviderSessionID}, nil
	}
	if binding.Stage == reporting.FinalEditStageEvidenceGate {
		if req.LongFormFinalize == nil {
			return AgentResult{}, fmt.Errorf("evidence gate run missing final binding")
		}
		if _, err := reporting.SubmitFinalEditEvidenceGate(ctx, executor.service, reporting.FinalEditEvidenceGateSubmitRequest{
			StageBinding:     binding,
			FinalBinding:     *req.LongFormFinalize,
			StageEventID:     fmt.Sprintf("evt_w4b_restart_submit_%d", len(executor.requests)),
			CanonicalEventID: fmt.Sprintf("evt_w4b_restart_final_%d", len(executor.requests)),
			Findings:         nil,
		}); err != nil {
			return AgentResult{}, err
		}
		return AgentResult{Text: finalEditGateSubmittedSentinel, SessionID: binding.ProviderSessionID}, nil
	}
	if _, err := reporting.SubmitFinalEditStage(ctx, executor.service, binding, fmt.Sprintf("evt_w4b_restart_submit_%d", len(executor.requests)), string(source.Content), 0); err != nil {
		return AgentResult{}, err
	}
	return AgentResult{Text: finalEditStageSubmittedSentinel, SessionID: binding.ProviderSessionID}, nil
}

func (executor *w4BRestartExecutor) ForkSession(_ context.Context, sourceSessionID string) (AgentSessionForkResult, error) {
	executor.forks++
	sessionID := fmt.Sprintf("provider-w4b-fork-%d", executor.forks)
	return AgentSessionForkResult{SessionID: sessionID, SourceSessionID: strings.TrimSpace(sourceSessionID)}, nil
}

func (executor *w4BRestartExecutor) CheckForkSession(context.Context, string) error {
	return nil
}

func runW4BRestartPipeline(t *testing.T, ctx context.Context, dbPath string, req longFormReaderStyleGatePipelineRequest, executor *w4BRestartExecutor) (*app.Service, func(), map[string]any) {
	t.Helper()
	svc, closeStore := openW4BRestartService(t, ctx, dbPath)
	executor.service = svc
	server := NewServer(svc, Options{}).(*Server)
	result, err := server.runLongFormReaderStyleGatePipeline(ctx, req, executor)
	if err != nil {
		closeStore()
		t.Fatal(err)
	}
	if strings.TrimSpace(fmt.Sprint(result["markdown"])) == "" {
		closeStore()
		t.Fatalf("pipeline returned empty markdown: %#v", result)
	}
	return svc, closeStore, result
}

func openW4BRestartService(t *testing.T, ctx context.Context, dbPath string) (*app.Service, func()) {
	t.Helper()
	store, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	return app.NewService(store), func() { _ = store.Close() }
}

func seedW4BRestartFixture(t *testing.T, ctx context.Context, svc *app.Service, humanize string) longFormReaderStyleGatePipelineRequest {
	t.Helper()
	return seedW4BRestartFixtureWithPipeline(t, ctx, svc, reporting.FinalEditPipelineReaderStyleGateV1, humanize)
}

func seedW4BRestartFixtureWithPipeline(t *testing.T, ctx context.Context, svc *app.Service, pipeline string, humanize string) longFormReaderStyleGatePipelineRequest {
	t.Helper()
	missionID := "mis_w4b_restart"
	pendingID := "evt_w4b_pending"
	planID := "evt_w4b_plan"
	finalID := "art_w4b_final"
	partID := "art_w4b_part"
	sectionID := "art_w4b_section"
	title := "Restart Report"
	producer := app.Producer{Type: "agent_session", ID: "provider-plan"}
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: title}); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range []app.CreateRawArtifactRequest{
		{ArtifactID: partID, MissionID: missionID, MediaType: "text/markdown; charset=utf-8", Filename: "part.md", Producer: producer, Content: []byte("# Part 1\n\n이 작업은 수행되어야 한다.\n")},
		{ArtifactID: sectionID, MissionID: missionID, MediaType: "text/markdown; charset=utf-8", Filename: "section.md", Producer: producer, Content: []byte("# Section 1\n\n이 작업은 수행되어야 한다.\n")},
	} {
		if _, err := svc.CreateRawArtifact(ctx, artifact); err != nil {
			t.Fatal(err)
		}
	}
	events := []app.AppendEventRequest{
		{EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: w4BJSON(map[string]any{"report_mode": reportexecution.ModeLongForm})},
		{EventID: planID, MissionID: missionID, EventType: "report.plan.created", Producer: producer, Payload: w4BJSON(map[string]any{
			"pending_event_id": pendingID, "report_mode": reportexecution.ModeLongForm, "artifact_id": finalID,
			"final_edit_pipeline": strings.TrimSpace(pipeline), "post_report_humanize": humanize,
			"plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"section 1"}}}},
		})},
		{EventID: "evt_w4b_part", MissionID: missionID, EventType: "report.part.created", Producer: producer, Payload: w4BJSON(map[string]any{"pending_event_id": pendingID, "plan_event_id": planID, "artifact_id": partID, "part_index": 1})},
		{EventID: "evt_w4b_section", MissionID: missionID, EventType: "report.section.created", Producer: producer, Payload: w4BJSON(map[string]any{"pending_event_id": pendingID, "plan_event_id": planID, "artifact_id": sectionID, "part_index": 1, "section_index": 1})},
	}
	var planEvent app.LedgerEvent
	for _, event := range events {
		appended, err := svc.AppendEvent(ctx, event)
		if err != nil {
			t.Fatal(err)
		}
		if event.EventID == planID {
			planEvent = appended
		}
	}
	return longFormReaderStyleGatePipelineRequest{
		missionID: missionID, title: title, executorName: "codex", agentModel: "model", agentReasoningEffort: "high",
		agentSelectionSource: "request", mcpMode: "auto", rigor: reportRigorProfiles["balanced"],
		reportSessionPolicy: reportSessionPolicySameSession, reportSessionPolicySelection: "default",
		postReportHumanize: humanize, generationGuidanceProfile: reportprompt.ProfileNarrativeContract,
		generationGuidanceSHA256: "guidance-sha", pendingEventID: pendingID, artifactID: finalID, planEvent: planEvent,
		plan: agentSectionalReportPlan{Summary: "plan"}, partArtifactIDs: []string{partID}, sectionArtifactIDs: []string{sectionID},
		sectionWordTotal: 3, sessionChainKind: "same_session_report", preReportResearchSessionID: "provider-research",
		reportPlanSessionID: "provider-plan", forkSourceAgentSessionID: "provider-plan", started: time.Now().UTC(),
	}
}

func w4BSubmitWriter(t *testing.T, ctx context.Context, svc *app.Service, req longFormReaderStyleGatePipelineRequest, suffix string, changed bool) reporting.FinalEditStageResult {
	t.Helper()
	binding := w4BWriterBinding(req, "provider-writer-"+suffix)
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_w4b_writer_"+suffix+"_start", binding); err != nil || !created {
		t.Fatalf("writer start created=%t err=%v", created, err)
	}
	source, err := svc.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	markdown := string(source.Content)
	operationCount := 0
	if changed {
		markdown += "\n최종 작성 흐름을 보강했다. " + suffix + ".\n"
		operationCount = 1
	}
	result, err := reporting.SubmitFinalEditStage(ctx, svc, binding, "evt_w4b_writer_"+suffix+"_submit", markdown, operationCount)
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed != changed {
		t.Fatalf("writer changed=%t, want %t: %#v", result.Changed, changed, result)
	}
	return result
}

func w4BSubmitV2Reader(t *testing.T, ctx context.Context, svc *app.Service, req longFormReaderStyleGatePipelineRequest, writer reporting.FinalEditStageResult, suffix string, changed bool) reporting.FinalEditStageResult {
	t.Helper()
	binding := req.finalEditStageBinding(reporting.FinalEditStageReader, writer.Artifact.ArtifactID, "art_w4b_reader_v2_"+suffix, "ses_reader_v2_"+suffix, "provider-reader-"+suffix, req.reportPlanSessionID, req.reportPlanSessionID)
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_w4b_reader_v2_"+suffix+"_start", binding); err != nil || !created {
		t.Fatalf("v2 reader start created=%t err=%v", created, err)
	}
	markdown := string(writer.Artifact.Content)
	operationCount := 0
	if changed {
		markdown += "\n읽기 흐름을 다듬었다. " + suffix + ".\n"
		operationCount = 1
	}
	result, err := reporting.SubmitFinalEditStage(ctx, svc, binding, "evt_w4b_reader_v2_"+suffix+"_submit", markdown, operationCount)
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed != changed {
		t.Fatalf("v2 reader changed=%t, want %t: %#v", result.Changed, changed, result)
	}
	return result
}

func w4BSubmitReader(t *testing.T, ctx context.Context, svc *app.Service, req longFormReaderStyleGatePipelineRequest, suffix string) reporting.FinalEditStageResult {
	t.Helper()
	binding := w4BReaderBinding(req, "provider-reader-"+suffix)
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_w4b_reader_"+suffix+"_start", binding); err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	source, err := svc.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	markdown := string(source.Content) + "\n추가 검토가 필요할 수 있다. " + suffix + ".\n"
	result, err := reporting.SubmitFinalEditStage(ctx, svc, binding, "evt_w4b_reader_"+suffix+"_submit", markdown, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Artifact.ArtifactID != binding.EditedArtifactID || result.Artifact.SHA256 == source.SHA256 {
		t.Fatalf("reader fixture did not create changed artifact: %#v source=%#v", result, source)
	}
	return result
}

func w4BSubmitStyle(t *testing.T, ctx context.Context, svc *app.Service, req longFormReaderStyleGatePipelineRequest, reader reporting.FinalEditStageResult, suffix string) reporting.FinalEditStageResult {
	t.Helper()
	binding := req.finalEditStageBinding(reporting.FinalEditStageStyle, reader.Artifact.ArtifactID, "art_w4b_style_"+suffix, "ses_style_"+suffix, "provider-style-"+suffix, "provider-reader-"+suffix, "provider-reader-"+suffix)
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_w4b_style_"+suffix+"_start", binding); err != nil || !created {
		t.Fatalf("style start created=%t err=%v", created, err)
	}
	markdown := strings.Replace(string(reader.Artifact.Content), "이 작업은 수행되어야 한다.", "이 작업은 수행해야 한다.", 1)
	if markdown == string(reader.Artifact.Content) {
		t.Fatalf("style fixture could not produce conservative changed artifact")
	}
	result, err := reporting.SubmitFinalEditStyleStage(ctx, svc, binding, "evt_w4b_style_"+suffix+"_submit", markdown, 1, finalEditStyleDiagnosesForWebTest(1))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Artifact.ArtifactID != binding.EditedArtifactID || result.Artifact.SHA256 == reader.Artifact.SHA256 {
		t.Fatalf("style fixture did not create changed artifact: %#v reader=%#v", result, reader.Artifact)
	}
	return result
}

func w4BWriterBinding(req longFormReaderStyleGatePipelineRequest, providerSessionID string) reporting.FinalEditStageBinding {
	sourceID := reporting.FinalEditAssemblyArtifactID(req.planEvent.EventID, req.partArtifactIDs)
	suffix := strings.TrimPrefix(strings.TrimSpace(providerSessionID), "provider-writer-")
	if suffix == "" {
		suffix = "writer"
	}
	return req.finalEditStageBinding(reporting.FinalEditStageWriter, sourceID, "art_w4b_writer_"+suffix, "ses_writer_"+suffix, providerSessionID, req.reportPlanSessionID, req.reportPlanSessionID)
}

func w4BReaderBinding(req longFormReaderStyleGatePipelineRequest, providerSessionID string) reporting.FinalEditStageBinding {
	sourceID := reporting.FinalEditReaderSourceArtifactID(req.planEvent.EventID, req.partArtifactIDs)
	suffix := strings.TrimPrefix(strings.TrimSpace(providerSessionID), "provider-reader-")
	if suffix == "" {
		suffix = "reader"
	}
	return req.finalEditStageBinding(reporting.FinalEditStageReader, sourceID, "art_w4b_reader_"+suffix, "ses_reader_"+suffix, providerSessionID, providerSessionID, req.reportPlanSessionID)
}

func w4BGateBinding(req longFormReaderStyleGatePipelineRequest, sourceArtifactID string, providerSessionID string) (reporting.FinalEditStageBinding, reporting.LongFormFinalizeBinding) {
	final := req.longFormFinalBinding("ses_gate_"+strings.TrimPrefix(providerSessionID, "provider-corrective-gate-"), providerSessionID, providerSessionID, req.reportPlanSessionID)
	final.Producer = app.Producer{Type: "agent_session", ID: providerSessionID}
	gate := req.finalEditStageBinding(reporting.FinalEditStageGate, sourceArtifactID, req.artifactID, final.ToolSessionID, final.ProviderSessionID, final.PreviousProviderSessionID, final.ForkSourceAgentSessionID)
	return gate, final
}

func w4BV2GateBinding(req longFormReaderStyleGatePipelineRequest, sourceArtifactID string, providerSessionID string) (reporting.FinalEditStageBinding, reporting.LongFormFinalizeBinding) {
	final := req.longFormFinalBinding("ses_gate_"+strings.TrimPrefix(providerSessionID, "provider-corrective-gate-"), providerSessionID, req.reportPlanSessionID, req.reportPlanSessionID)
	final.Producer = app.Producer{Type: "agent_session", ID: providerSessionID}
	gate := req.finalEditStageBinding(reporting.FinalEditStageGate, sourceArtifactID, req.artifactID, final.ToolSessionID, final.ProviderSessionID, final.PreviousProviderSessionID, final.ForkSourceAgentSessionID)
	return gate, final
}

func w4BAppendGateSubmission(t *testing.T, ctx context.Context, svc *app.Service, binding reporting.FinalEditStageBinding, eventID string) {
	t.Helper()
	source, err := svc.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	pipeline := reporting.FinalEditPipelineReaderStyleGateV1
	if binding.FinalEditPipeline == reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2 {
		pipeline = reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2
	}
	payload := map[string]any{
		"kind": "long_form_final_edit_" + binding.Stage + "_submitted", "pending_event_id": binding.PendingEventID,
		"plan_event_id": binding.PlanEventID, "final_edit_pipeline": pipeline,
		"title": binding.Title, "stage": binding.Stage, "stage_id": "corrective-gate", "source_artifact_id": binding.SourceArtifactID,
		"artifact_id": source.ArtifactID, "edited_artifact_id": binding.EditedArtifactID, "filename": binding.Filename,
		"tool_session_id": binding.ToolSessionID, "provider_session_id": binding.ProviderSessionID,
		"previous_provider_session_id": binding.PreviousProviderSessionID, "idempotency_key": binding.IdempotencyKey,
		"agent_executor": binding.AgentExecutor, "agent_model": binding.AgentModel, "agent_reasoning_effort": binding.AgentReasoningEffort,
		"agent_selection_source": binding.AgentSelectionSource, "mcp_mode": binding.MCPMode, "rigor_level": binding.RigorLevel,
		"rigor_label": binding.RigorLabel, "report_session_policy": binding.ReportSessionPolicy,
		"report_session_policy_selection": binding.ReportSessionPolicySelection, "post_report_humanize": binding.PostReportHumanize,
		"generation_guidance_profile": binding.GenerationGuidanceProfile, "generation_guidance_sha256": binding.GenerationGuidanceSHA256,
		"session_chain_kind": binding.SessionChainKind, "pre_report_research_session_id": binding.PreReportResearchSessionID,
		"report_plan_session_id": binding.ReportPlanSessionID, "fork_source_agent_session_id": binding.ForkSourceAgentSessionID,
		"operation_count": 0, "source_word_count": len(strings.Fields(string(source.Content))),
		"edited_word_count": len(strings.Fields(string(source.Content))), "source_sha256": w4BSHA256(source.Content),
		"artifact_sha256": w4BSHA256(source.Content), "changed": false,
		"text": "장문 리포트 corrective_gate 단계를 durable artifact로 제출했습니다.",
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: strings.TrimSpace(eventID), MissionID: binding.MissionID, EventType: reporting.FinalEditGateSubmittedEventType,
		Producer: binding.Producer, CausationEventID: binding.PlanEventID, CorrelationID: binding.IdempotencyKey, Payload: w4BJSON(payload),
	}); err != nil {
		t.Fatal(err)
	}
}

func w4BAppendTerminalCanonical(t *testing.T, ctx context.Context, svc *app.Service, req longFormReaderStyleGatePipelineRequest, eventID string) {
	t.Helper()
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: strings.TrimSpace(eventID), MissionID: req.missionID, EventType: "report.artifact.created",
		Producer: app.Producer{Type: "agent_session", ID: "provider-terminal"}, CorrelationID: "terminal-" + req.pendingEventID,
		Payload: w4BJSON(map[string]any{
			"pending_event_id": req.pendingEventID,
			"plan_event_id":    req.planEvent.EventID,
			"artifact_id":      req.artifactID,
		}),
	}); err != nil {
		t.Fatal(err)
	}
}

func w4BStageRequests(requests []AgentRequest, stage string) []AgentRequest {
	out := []AgentRequest{}
	for _, req := range requests {
		if req.FinalEditStage != nil && req.FinalEditStage.Stage == stage {
			out = append(out, req)
		}
	}
	return out
}

func assertW4BStageRequestSequence(t *testing.T, requests []AgentRequest, stages ...string) {
	t.Helper()
	if len(requests) != len(stages) {
		t.Fatalf("stage request count=%d, want %d: %#v", len(requests), len(stages), requests)
	}
	for index, stage := range stages {
		if requests[index].FinalEditStage == nil || requests[index].FinalEditStage.Stage != stage {
			t.Fatalf("request %d stage=%#v, want %s: %#v", index, requests[index].FinalEditStage, stage, requests)
		}
	}
}

func w4BEvents(t *testing.T, ctx context.Context, svc *app.Service, missionID string) []app.LedgerEvent {
	t.Helper()
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	return events
}

func assertW4BEventCount(t *testing.T, events []app.LedgerEvent, eventType string, want int) {
	t.Helper()
	got := 0
	for _, event := range events {
		if event.EventType == eventType {
			got++
		}
	}
	if got != want {
		t.Fatalf("%s count=%d, want %d", eventType, got, want)
	}
}

func w4BCanonicalEvent(t *testing.T, events []app.LedgerEvent) app.LedgerEvent {
	t.Helper()
	var found app.LedgerEvent
	count := 0
	for _, event := range events {
		if event.EventType == "report.artifact.created" {
			found = event
			count++
		}
	}
	if count != 1 {
		t.Fatalf("canonical event count=%d, want 1", count)
	}
	return found
}

func w4BPayload(t *testing.T, event app.LedgerEvent) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func w4BResultArtifact(t *testing.T, result map[string]any) app.RawArtifact {
	t.Helper()
	artifact, ok := result["artifact"].(app.RawArtifact)
	if !ok {
		t.Fatalf("result artifact missing: %#v", result)
	}
	return artifact
}

func w4BResultEvent(t *testing.T, result map[string]any) app.LedgerEvent {
	t.Helper()
	event, ok := result["event"].(app.LedgerEvent)
	if !ok {
		t.Fatalf("result event missing: %#v", result)
	}
	return event
}

func w4BJSON(value any) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}

func w4BSHA256(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

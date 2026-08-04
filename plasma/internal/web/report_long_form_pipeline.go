package web

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const (
	finalEditStageSubmittedSentinel = "FINAL_EDIT_STAGE_SUBMITTED"
	finalEditGateSubmittedSentinel  = "REPORT_FINALIZED"
)

type longFormReaderStyleGatePipelineRequest struct {
	missionID                    string
	title                        string
	executorName                 string
	agentModel                   string
	agentReasoningEffort         string
	agentSelectionSource         string
	mcpMode                      string
	rigor                        reportRigorProfile
	reportSessionPolicy          string
	reportSessionPolicySelection string
	postReportHumanize           string
	generationGuidanceProfile    string
	generationGuidanceSHA256     string
	pendingEventID               string
	artifactID                   string
	planEvent                    app.LedgerEvent
	plan                         agentSectionalReportPlan
	requirementMap               reporting.ReportRequirementMap
	parts                        []sectionalReportPartDraft
	partArtifactIDs              []string
	sectionArtifactIDs           []string
	sectionWordTotal             int
	sessionChainKind             string
	preReportResearchSessionID   string
	reportPlanSessionID          string
	forkSourceAgentSessionID     string
	started                      time.Time
}

type longFormFinalEditStageRun struct {
	binding reporting.FinalEditStageBinding
	stage   reporting.FinalEditStageResult
	final   reporting.LongFormFinalizeResult
	agent   AgentResult
}

func longFormReaderStyleGateEnabled(profile string) bool {
	return longFormPartEditEnabled(profile)
}

func longFormFinalEditPipelineForPlan(profile string) string {
	if longFormReaderStyleGateEnabled(profile) {
		return reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	}
	return ""
}

func longFormReaderStyleGatePlanEventEnabled(event app.LedgerEvent) bool {
	state, ok, err := reporting.FinalEditPipelineFromPlanEvent(event)
	return err == nil && ok && longFormSupportedStagedFinalEditPipeline(state.Pipeline)
}

func longFormSupportedStagedFinalEditPipeline(pipeline string) bool {
	switch strings.TrimSpace(pipeline) {
	case reporting.FinalEditPipelineReaderStyleGateV1, reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2, reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3:
		return true
	default:
		return false
	}
}

func (server *Server) runLongFormReaderStyleGatePipeline(ctx context.Context, req longFormReaderStyleGatePipelineRequest, executor AgentExecutor) (map[string]any, error) {
	forker, ok := executor.(AgentSessionForker)
	if !ok {
		return nil, longFormFinalEditStageFailure("final_edit", req.planEvent.EventID, fmt.Errorf("%w: final edit pipeline requires an agent session forker", app.ErrInvalidInput))
	}
	planSessionID := firstNonEmpty(req.reportPlanSessionID, req.forkSourceAgentSessionID)
	if strings.TrimSpace(planSessionID) == "" {
		return nil, longFormFinalEditStageFailure("final_edit", req.planEvent.EventID, fmt.Errorf("%w: final edit pipeline requires a report plan session", app.ErrConflict))
	}
	switch req.finalEditPipeline() {
	case reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3:
		return server.runLongFormAssemblyWriterReaderStyleValidationEvidenceGatePipeline(ctx, req, executor, forker, planSessionID)
	case reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2:
		return server.runLongFormAssemblyWriterReaderStyleGatePipeline(ctx, req, executor, forker, planSessionID)
	case reporting.FinalEditPipelineReaderStyleGateV1:
	default:
		return nil, longFormFinalEditStageFailure("final_edit", req.planEvent.EventID, fmt.Errorf("%w: unsupported final edit pipeline", app.ErrConflict))
	}
	recoveryFinal := req.longFormFinalBinding(newID("ses"), planSessionID, planSessionID, firstNonEmpty(req.forkSourceAgentSessionID, planSessionID))
	readerProgress, err := server.finalEditStageProgress(ctx, recoveryFinal, req, reporting.FinalEditStageReader, reporting.FinalEditReaderSourceArtifactID(req.planEvent.EventID, req.partArtifactIDs), newID("art"), planSessionID, forker, "")
	if err != nil {
		return nil, err
	}
	reader, err := server.runLongFormFinalEditStage(ctx, req, readerProgress, nil, reportFinalEditReaderMCPTools(), finalEditStageSubmittedSentinel, agentLongFormReaderEditPrompt, executor)
	if err != nil {
		return nil, err
	}
	prior := reader
	if req.postReportHumanize == reporting.FinalEditHumanizeEnabled {
		styleProgress, err := server.finalEditStageProgress(ctx, recoveryFinal, req, reporting.FinalEditStageStyle, reader.stage.Artifact.ArtifactID, newID("art"), reader.binding.ProviderSessionID, forker, reader.binding.ProviderSessionID)
		if err != nil {
			return nil, err
		}
		style, err := server.runLongFormFinalEditStage(ctx, req, styleProgress, nil, reportFinalEditStyleMCPTools(), finalEditStageSubmittedSentinel, agentLongFormStyleEditPrompt, executor)
		if err != nil {
			return nil, err
		}
		prior = style
	}
	gateProgress, gateFinalBinding, err := server.finalEditGateProgress(ctx, req, recoveryFinal, prior.stage.Artifact.ArtifactID, prior.binding.ProviderSessionID, planSessionID, forker)
	if err != nil {
		return nil, err
	}
	gate, err := server.runLongFormFinalEditStage(ctx, req, gateProgress, &gateFinalBinding, reportFinalEditGateMCPToolsForHumanize(req.postReportHumanize), finalEditGateSubmittedSentinel, agentLongFormGatePromptForHumanize(req.postReportHumanize), executor)
	if err != nil {
		return nil, err
	}
	artifact, event := gate.final.Artifact, gate.final.Event
	return map[string]any{"artifact": artifact, "event": event, "markdown": string(artifact.Content)}, nil
}

func (server *Server) runLongFormAssemblyWriterReaderStyleValidationEvidenceGatePipeline(ctx context.Context, req longFormReaderStyleGatePipelineRequest, executor AgentExecutor, forker AgentSessionForker, planSessionID string) (map[string]any, error) {
	recoveryFinal := req.longFormFinalBinding(newID("ses"), planSessionID, planSessionID, firstNonEmpty(req.forkSourceAgentSessionID, planSessionID))
	assemblyArtifactID := reporting.FinalEditAssemblyArtifactID(req.planEvent.EventID, req.partArtifactIDs)
	writerProgress, err := server.finalEditStageProgress(ctx, recoveryFinal, req, reporting.FinalEditStageWriter, assemblyArtifactID, newID("art"), planSessionID, forker, planSessionID)
	if err != nil {
		return nil, err
	}
	if writerProgress.StartEvent.EventID == "" && writerProgress.Submission == nil {
		if _, _, err := reporting.EnsureFinalEditAssembly(ctx, server.service, newID("evt"), writerProgress.Binding); err != nil {
			return nil, longFormFinalEditStageFailure(reporting.FinalEditStageWriter, req.planEvent.EventID, err)
		}
	}
	writer, err := server.runLongFormFinalEditStage(ctx, req, writerProgress, nil, reportFinalEditWriterMCPTools(), finalEditStageSubmittedSentinel, agentLongFormWriterEditPrompt, executor)
	if err != nil {
		return nil, err
	}
	readerProgress, err := server.finalEditStageProgress(ctx, recoveryFinal, req, reporting.FinalEditStageReader, writer.stage.Artifact.ArtifactID, newID("art"), planSessionID, forker, planSessionID)
	if err != nil {
		return nil, err
	}
	reader, err := server.runLongFormFinalEditStage(ctx, req, readerProgress, nil, reportFinalEditReaderMCPTools(), finalEditStageSubmittedSentinel, agentLongFormReaderEditPrompt, executor)
	if err != nil {
		return nil, err
	}
	prior := reader
	if req.postReportHumanize == reporting.FinalEditHumanizeEnabled {
		styleProgress, err := server.finalEditStageProgress(ctx, recoveryFinal, req, reporting.FinalEditStageStyle, reader.stage.Artifact.ArtifactID, newID("art"), reader.binding.ProviderSessionID, forker, reader.binding.ProviderSessionID)
		if err != nil {
			return nil, err
		}
		style, err := server.runLongFormFinalEditStage(ctx, req, styleProgress, nil, reportFinalEditStyleMCPTools(), finalEditStageSubmittedSentinel, agentLongFormStyleEditPrompt, executor)
		if err != nil {
			return nil, err
		}
		semanticProgress, err := server.finalEditStageProgress(ctx, recoveryFinal, req, reporting.FinalEditStageStyleSemanticValidation, style.stage.Artifact.ArtifactID, newID("art"), planSessionID, forker, planSessionID)
		if err != nil {
			return nil, err
		}
		semantic, err := server.runLongFormFinalEditStage(ctx, req, semanticProgress, nil, reportFinalEditStyleSemanticValidationMCPTools(), finalEditStageSubmittedSentinel, agentLongFormStyleSemanticValidationPrompt, executor)
		if err != nil {
			return nil, err
		}
		prior = semantic
	}
	gateProgress, gateFinalBinding, err := server.finalEditEvidenceGateProgress(ctx, req, recoveryFinal, prior.stage.Artifact.ArtifactID, planSessionID, planSessionID, forker)
	if err != nil {
		return nil, err
	}
	gate, err := server.runLongFormFinalEditStage(ctx, req, gateProgress, &gateFinalBinding, reportFinalEditEvidenceGateMCPTools(), finalEditGateSubmittedSentinel, agentLongFormEvidenceGatePrompt, executor)
	if err != nil {
		return nil, err
	}
	artifact, event := gate.final.Artifact, gate.final.Event
	return map[string]any{"artifact": artifact, "event": event, "markdown": string(artifact.Content)}, nil
}

func (server *Server) runLongFormAssemblyWriterReaderStyleGatePipeline(ctx context.Context, req longFormReaderStyleGatePipelineRequest, executor AgentExecutor, forker AgentSessionForker, planSessionID string) (map[string]any, error) {
	recoveryFinal := req.longFormFinalBinding(newID("ses"), planSessionID, planSessionID, firstNonEmpty(req.forkSourceAgentSessionID, planSessionID))
	assemblyArtifactID := reporting.FinalEditAssemblyArtifactID(req.planEvent.EventID, req.partArtifactIDs)
	writerProgress, err := server.finalEditStageProgress(ctx, recoveryFinal, req, reporting.FinalEditStageWriter, assemblyArtifactID, newID("art"), planSessionID, forker, planSessionID)
	if err != nil {
		return nil, err
	}
	if writerProgress.StartEvent.EventID == "" && writerProgress.Submission == nil {
		if _, _, err := reporting.EnsureFinalEditAssembly(ctx, server.service, newID("evt"), writerProgress.Binding); err != nil {
			return nil, longFormFinalEditStageFailure(reporting.FinalEditStageWriter, req.planEvent.EventID, err)
		}
	}
	writer, err := server.runLongFormFinalEditStage(ctx, req, writerProgress, nil, reportFinalEditWriterMCPTools(), finalEditStageSubmittedSentinel, agentLongFormWriterEditPrompt, executor)
	if err != nil {
		return nil, err
	}
	readerProgress, err := server.finalEditStageProgress(ctx, recoveryFinal, req, reporting.FinalEditStageReader, writer.stage.Artifact.ArtifactID, newID("art"), planSessionID, forker, planSessionID)
	if err != nil {
		return nil, err
	}
	reader, err := server.runLongFormFinalEditStage(ctx, req, readerProgress, nil, reportFinalEditReaderMCPTools(), finalEditStageSubmittedSentinel, agentLongFormReaderEditPrompt, executor)
	if err != nil {
		return nil, err
	}
	prior := reader
	if req.postReportHumanize == reporting.FinalEditHumanizeEnabled {
		styleProgress, err := server.finalEditStageProgress(ctx, recoveryFinal, req, reporting.FinalEditStageStyle, reader.stage.Artifact.ArtifactID, newID("art"), reader.binding.ProviderSessionID, forker, reader.binding.ProviderSessionID)
		if err != nil {
			return nil, err
		}
		style, err := server.runLongFormFinalEditStage(ctx, req, styleProgress, nil, reportFinalEditStyleMCPTools(), finalEditStageSubmittedSentinel, agentLongFormStyleEditPrompt, executor)
		if err != nil {
			return nil, err
		}
		prior = style
	}
	gateProgress, gateFinalBinding, err := server.finalEditGateProgress(ctx, req, recoveryFinal, prior.stage.Artifact.ArtifactID, planSessionID, planSessionID, forker)
	if err != nil {
		return nil, err
	}
	gate, err := server.runLongFormFinalEditStage(ctx, req, gateProgress, &gateFinalBinding, reportFinalEditGateMCPToolsForHumanize(req.postReportHumanize), finalEditGateSubmittedSentinel, agentLongFormGatePromptForHumanize(req.postReportHumanize), executor)
	if err != nil {
		return nil, err
	}
	artifact, event := gate.final.Artifact, gate.final.Event
	return map[string]any{"artifact": artifact, "event": event, "markdown": string(artifact.Content)}, nil
}

func (req longFormReaderStyleGatePipelineRequest) finalEditPipeline() string {
	state, ok, err := reporting.FinalEditPipelineFromPlanEvent(req.planEvent)
	if err != nil || !ok {
		return ""
	}
	return strings.TrimSpace(state.Pipeline)
}

func (req longFormReaderStyleGatePipelineRequest) longFormFinalBinding(toolSessionID string, providerSessionID string, previousProviderSessionID string, forkSourceAgentSessionID string) reporting.LongFormFinalizeBinding {
	return reporting.LongFormFinalizeBinding{
		MissionID: req.missionID, PendingEventID: req.pendingEventID, PlanEventID: req.planEvent.EventID, ArtifactID: req.artifactID,
		Filename: safeFilename(req.title, ".md"), Title: req.title, ToolSessionID: toolSessionID,
		IdempotencyKey:    "report-long-form-finalize:" + req.pendingEventID + ":" + req.planEvent.EventID,
		ProviderSessionID: providerSessionID, PreviousProviderSessionID: previousProviderSessionID,
		PartArtifactIDs: req.partArtifactIDs, SectionArtifactIDs: req.sectionArtifactIDs, SectionWordCount: req.sectionWordTotal,
		CompositionStrategy: reporting.LongFormCompositionNarrativeEdit,
		AgentExecutor:       req.executorName, AgentModel: req.agentModel, AgentReasoningEffort: req.finalEditAgentReasoningEffort(), AgentSelectionSource: req.agentSelectionSource,
		MCPMode: req.mcpMode, RigorLevel: req.rigor.level, RigorLabel: req.rigor.label,
		ReportSessionPolicy: req.reportSessionPolicy, ReportSessionPolicySelection: req.reportSessionPolicySelection,
		PostReportHumanize: req.postReportHumanize, GenerationGuidanceProfile: req.generationGuidanceProfile, GenerationGuidanceSHA256: req.generationGuidanceSHA256,
		SessionChainKind: req.sessionChainKind, PreReportResearchSessionID: req.preReportResearchSessionID, ReportPlanSessionID: req.reportPlanSessionID,
		ForkSourceAgentSessionID: forkSourceAgentSessionID, PlanToolSessionID: reportEventString(req.planEvent, "tool_session_id"), StartedAt: req.started,
		Producer: app.Producer{Type: "agent_session", ID: providerSessionID},
	}
}

func (server *Server) finalEditStageProgress(ctx context.Context, final reporting.LongFormFinalizeBinding, req longFormReaderStyleGatePipelineRequest, stage string, sourceArtifactID string, editedArtifactID string, forkFromSessionID string, forker AgentSessionForker, previousProviderSessionID string) (reporting.FinalEditStageProgress, error) {
	if recovered, ok, err := reporting.LoadFinalEditStageProgress(ctx, server.service, reporting.FinalEditStageStartContract{FinalBinding: final, Stage: stage}); err != nil {
		return reporting.FinalEditStageProgress{}, longFormFinalEditStageFailure(stage, req.planEvent.EventID, err)
	} else if ok {
		return recovered, nil
	}
	providerSessionID, forkSourceID, err := forkLongFormFinalEditSession(ctx, forker, forkFromSessionID)
	if err != nil {
		return reporting.FinalEditStageProgress{}, longFormFinalEditStageFailure(stage, req.planEvent.EventID, err)
	}
	if strings.TrimSpace(previousProviderSessionID) == "" {
		previousProviderSessionID = providerSessionID
	}
	return reporting.FinalEditStageProgress{Binding: req.finalEditStageBinding(stage, sourceArtifactID, editedArtifactID, newID("ses"), providerSessionID, previousProviderSessionID, forkSourceID)}, nil
}

func (server *Server) finalEditGateProgress(ctx context.Context, req longFormReaderStyleGatePipelineRequest, finalBase reporting.LongFormFinalizeBinding, sourceArtifactID string, previousProviderSessionID string, planSessionID string, forker AgentSessionForker) (reporting.FinalEditStageProgress, reporting.LongFormFinalizeBinding, error) {
	if recovered, ok, err := reporting.LoadFinalEditStageProgress(ctx, server.service, reporting.FinalEditStageStartContract{FinalBinding: finalBase, Stage: reporting.FinalEditStageGate}); err != nil {
		return reporting.FinalEditStageProgress{}, reporting.LongFormFinalizeBinding{}, longFormFinalEditStageFailure(reporting.FinalEditStageGate, req.planEvent.EventID, err)
	} else if ok {
		final := req.longFormFinalBinding(recovered.Binding.ToolSessionID, recovered.Binding.ProviderSessionID, recovered.Binding.PreviousProviderSessionID, recovered.Binding.ForkSourceAgentSessionID)
		final.Producer = recovered.Binding.Producer
		if err := reporting.ValidateFinalEditGateBindingsCompatible(recovered.Binding, final); err != nil {
			return reporting.FinalEditStageProgress{}, reporting.LongFormFinalizeBinding{}, err
		}
		return recovered, final, nil
	}
	gateSessionID, gateForkSourceID, err := forkLongFormFinalEditSession(ctx, forker, planSessionID)
	if err != nil {
		return reporting.FinalEditStageProgress{}, reporting.LongFormFinalizeBinding{}, longFormFinalEditStageFailure(reporting.FinalEditStageGate, req.planEvent.EventID, err)
	}
	final := req.longFormFinalBinding(newID("ses"), gateSessionID, previousProviderSessionID, gateForkSourceID)
	gate := req.finalEditStageBinding(reporting.FinalEditStageGate, sourceArtifactID, req.artifactID, final.ToolSessionID, final.ProviderSessionID, final.PreviousProviderSessionID, final.ForkSourceAgentSessionID)
	if err := reporting.ValidateFinalEditGateBindingsCompatible(gate, final); err != nil {
		return reporting.FinalEditStageProgress{}, reporting.LongFormFinalizeBinding{}, err
	}
	return reporting.FinalEditStageProgress{Binding: gate}, final, nil
}

func (server *Server) finalEditEvidenceGateProgress(ctx context.Context, req longFormReaderStyleGatePipelineRequest, finalBase reporting.LongFormFinalizeBinding, sourceArtifactID string, previousProviderSessionID string, planSessionID string, forker AgentSessionForker) (reporting.FinalEditStageProgress, reporting.LongFormFinalizeBinding, error) {
	if recovered, ok, err := reporting.LoadFinalEditStageProgress(ctx, server.service, reporting.FinalEditStageStartContract{FinalBinding: finalBase, Stage: reporting.FinalEditStageEvidenceGate}); err != nil {
		return reporting.FinalEditStageProgress{}, reporting.LongFormFinalizeBinding{}, longFormFinalEditStageFailure(reporting.FinalEditStageEvidenceGate, req.planEvent.EventID, err)
	} else if ok {
		final := req.longFormFinalBinding(recovered.Binding.ToolSessionID, recovered.Binding.ProviderSessionID, recovered.Binding.PreviousProviderSessionID, recovered.Binding.ForkSourceAgentSessionID)
		final.Producer = recovered.Binding.Producer
		if err := reporting.ValidateFinalEditGateBindingsCompatible(recovered.Binding, final); err != nil {
			return reporting.FinalEditStageProgress{}, reporting.LongFormFinalizeBinding{}, err
		}
		return recovered, final, nil
	}
	gateSessionID, gateForkSourceID, err := forkLongFormFinalEditSession(ctx, forker, planSessionID)
	if err != nil {
		return reporting.FinalEditStageProgress{}, reporting.LongFormFinalizeBinding{}, longFormFinalEditStageFailure(reporting.FinalEditStageEvidenceGate, req.planEvent.EventID, err)
	}
	final := req.longFormFinalBinding(newID("ses"), gateSessionID, previousProviderSessionID, gateForkSourceID)
	gate := req.finalEditStageBinding(reporting.FinalEditStageEvidenceGate, sourceArtifactID, req.artifactID, final.ToolSessionID, final.ProviderSessionID, final.PreviousProviderSessionID, final.ForkSourceAgentSessionID)
	if err := reporting.ValidateFinalEditGateBindingsCompatible(gate, final); err != nil {
		return reporting.FinalEditStageProgress{}, reporting.LongFormFinalizeBinding{}, err
	}
	return reporting.FinalEditStageProgress{Binding: gate}, final, nil
}

func (req longFormReaderStyleGatePipelineRequest) finalEditStageBinding(stage string, sourceArtifactID string, editedArtifactID string, toolSessionID string, providerSessionID string, previousProviderSessionID string, forkSourceAgentSessionID string) reporting.FinalEditStageBinding {
	binding := reporting.FinalEditStageBinding{
		MissionID: req.missionID, PendingEventID: req.pendingEventID, PlanEventID: req.planEvent.EventID, Title: req.title,
		Stage: stage, SourceArtifactID: sourceArtifactID, EditedArtifactID: editedArtifactID, Filename: safeFilename(req.title, ".md"),
		ToolSessionID: toolSessionID, ProviderSessionID: providerSessionID, PreviousProviderSessionID: previousProviderSessionID,
		IdempotencyKey: reporting.FinalEditStageIdempotencyKey(stage, req.pendingEventID, req.planEvent.EventID),
		AgentExecutor:  req.executorName, AgentModel: req.agentModel, AgentReasoningEffort: req.finalEditAgentReasoningEffort(), AgentSelectionSource: req.agentSelectionSource,
		MCPMode: req.mcpMode, RigorLevel: req.rigor.level, RigorLabel: req.rigor.label,
		ReportSessionPolicy: req.reportSessionPolicy, ReportSessionPolicySelection: req.reportSessionPolicySelection,
		PostReportHumanize: req.postReportHumanize, GenerationGuidanceProfile: req.generationGuidanceProfile, GenerationGuidanceSHA256: req.generationGuidanceSHA256,
		SessionChainKind: req.sessionChainKind, PreReportResearchSessionID: req.preReportResearchSessionID, ReportPlanSessionID: req.reportPlanSessionID,
		ForkSourceAgentSessionID: forkSourceAgentSessionID, Producer: app.Producer{Type: "agent_session", ID: providerSessionID},
	}
	if pipeline := req.finalEditPipeline(); pipeline == reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2 || pipeline == reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		binding.FinalEditPipeline = pipeline
	}
	return binding
}

func (req longFormReaderStyleGatePipelineRequest) finalEditAgentReasoningEffort() string {
	return longFormFinalEditContractReasoningEffort(req.agentReasoningEffort)
}

func longFormFinalEditContractReasoningEffort(value string) string {
	return firstNonEmpty(value, "default")
}

func (server *Server) runLongFormFinalEditStage(ctx context.Context, req longFormReaderStyleGatePipelineRequest, progress reporting.FinalEditStageProgress, finalBinding *reporting.LongFormFinalizeBinding, tools []string, sentinel string, prompt func(longFormReaderStyleGatePipelineRequest, reporting.FinalEditStageBinding, string, int) string, executor AgentExecutor) (longFormFinalEditStageRun, error) {
	binding := progress.Binding
	if progress.Submission != nil {
		return server.replayLongFormFinalEditStage(ctx, req, binding, *progress.Submission, finalBinding)
	}
	var last AgentResult
	for attempt := 1; attempt <= 2; attempt++ {
		started := time.Now()
		result, runErr := executor.Run(ctx, AgentRequest{
			UserText: "run long-form " + binding.Stage,
			Prompt:   prompt(req, binding, newID("rfe"), attempt),
			Model:    req.agentModel, ReasoningEffort: req.agentReasoningEffort, MissionID: req.missionID, ToolSessionID: binding.ToolSessionID,
			PreviousSessionID: binding.ProviderSessionID, AgentExecutor: req.executorName, MCPMode: req.mcpMode,
			ExtraMCPTools: tools, ReplaceMCPTools: true, FinalEditStage: &binding, LongFormFinalize: finalBinding,
		})
		durationMS := time.Since(started).Milliseconds()
		if runErr == nil {
			result, runErr = validatedSameSessionResult(result, binding.ProviderSessionID)
		}
		if runErr == nil {
			last = result
		}
		stage, stageOK, stageErr := reporting.LoadFinalEditStageSubmission(context.WithoutCancel(ctx), server.service, binding)
		if stageErr != nil {
			return longFormFinalEditStageRun{}, longFormFinalEditStageFailure(binding.Stage, req.planEvent.EventID, stageErr)
		}
		if finalBinding != nil {
			final, finalOK, finalErr := reporting.LoadLongFormFinalization(context.WithoutCancel(ctx), server.service, *finalBinding)
			if finalErr != nil {
				return longFormFinalEditStageRun{}, longFormFinalEditStageFailure(binding.Stage, req.planEvent.EventID, finalErr)
			}
			if !finalOK && stageOK {
				final, finalErr = reporting.ResumeFinalEditGate(context.WithoutCancel(ctx), server.service, reporting.FinalEditGateResumeRequest{
					StageBinding: binding, FinalBinding: *finalBinding, CanonicalEventID: newID("evt"),
				})
				finalOK = finalErr == nil
				if finalErr != nil && runErr == nil {
					runErr = finalErr
				}
			}
			if finalOK && stageOK {
				return longFormFinalEditStageRun{binding: binding, stage: stage, final: final, agent: result}, nil
			}
		} else if stageOK {
			return longFormFinalEditStageRun{binding: binding, stage: stage, agent: result}, nil
		}
		if attempt == 1 {
			continue
		}
		cause := runErr
		if cause == nil {
			cause = fmt.Errorf("%w: final edit stage acknowledgement was not exact", app.ErrConflict)
		}
		return longFormFinalEditStageRun{}, longFormFinalEditStageFailure(binding.Stage, req.planEvent.EventID, reportAgentFailure(cause, result, "report_"+binding.Stage, durationMS, binding.ProviderSessionID))
	}
	return longFormFinalEditStageRun{binding: binding, agent: last}, nil
}

func (server *Server) replayLongFormFinalEditStage(ctx context.Context, req longFormReaderStyleGatePipelineRequest, binding reporting.FinalEditStageBinding, stage reporting.FinalEditStageResult, finalBinding *reporting.LongFormFinalizeBinding) (longFormFinalEditStageRun, error) {
	if finalBinding == nil {
		return longFormFinalEditStageRun{binding: binding, stage: stage}, nil
	}
	final, finalOK, finalErr := reporting.LoadLongFormFinalization(context.WithoutCancel(ctx), server.service, *finalBinding)
	if finalErr != nil {
		return longFormFinalEditStageRun{}, longFormFinalEditStageFailure(binding.Stage, req.planEvent.EventID, finalErr)
	}
	if !finalOK {
		resumed, err := reporting.ResumeFinalEditGate(context.WithoutCancel(ctx), server.service, reporting.FinalEditGateResumeRequest{
			StageBinding: binding, FinalBinding: *finalBinding, CanonicalEventID: newID("evt"),
		})
		if err != nil {
			return longFormFinalEditStageRun{}, longFormFinalEditStageFailure(binding.Stage, req.planEvent.EventID, err)
		}
		final = resumed
	}
	return longFormFinalEditStageRun{binding: binding, stage: stage, final: final}, nil
}

func longFormFinalEditStageFailure(stage, planID string, cause error) error {
	stage = strings.TrimSpace(stage)
	if stage != "" {
		cause = fmt.Errorf("%s: %w", stage, cause)
	}
	return longFormStageFailure("final", planID, 0, 0, cause)
}

func forkLongFormFinalEditSession(ctx context.Context, forker AgentSessionForker, sourceSessionID string) (string, string, error) {
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	if sourceSessionID == "" {
		return "", "", fmt.Errorf("%w: final edit requires a source provider session", app.ErrConflict)
	}
	fork, err := forker.ForkSession(ctx, sourceSessionID)
	if err != nil {
		return "", "", fmt.Errorf("final edit session fork failed: %w", err)
	}
	if strings.TrimSpace(fork.SessionID) == "" {
		return "", "", fmt.Errorf("%w: final edit session fork returned an empty session", app.ErrConflict)
	}
	return strings.TrimSpace(fork.SessionID), firstNonEmpty(strings.TrimSpace(fork.SourceSessionID), sourceSessionID), nil
}

func agentLongFormWriterEditPrompt(req longFormReaderStyleGatePipelineRequest, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	return fmt.Sprintf(`Write the final long-form manuscript from the deterministic assembly through the dedicated final-write MCP tools.

Report title: %s
Mission ID: %s
Bound stage metadata:
%s

Use exactly this workflow:
1. Call %s with draft_id %q and the bound identities.
2. Read the full deterministic assembly with %s until truncated is false.
3. Use %s for final writing edits that improve whole-report opening, conclusion, Part transitions, global logic, and cross-Part duplicate paragraphs without changing the report's evidence boundary.
4. Submit with %s using the same draft_id, pending_event_id, and plan_event_id.
5. Return exactly %s and nothing else.

Final-writer responsibilities:
- You may create or improve a whole-report opening, a conclusion, Part-to-Part transitions, and global connective logic.
- You may merge or move duplicate paragraphs across Parts when they repeat the same function, but preserve Part order and do not perform a full Part or Section reorder.
- Preserve every unique fact, number, condition, citation, uncertainty, owner requirement, caveat meaning, code identifier, technical identifier, and cited relationship.
- Do not add research, external facts, new sources, new citations, unsupported claims, or new policy.
- Keep source-boundary and uncertainty language where it changes claim scope; do not erase model/session-memory-supported connective reasoning merely because it is synthesis.
- Submit unchanged only after a full read finds no justified final-writing edit.

Do not call research or source tools. Do not expose IDs in the manuscript.%s`,
		req.title, req.missionID, agentReportAnyJSON(binding),
		mcptools.ToolReportLongFormFinalWriteStart, draftID,
		mcptools.ToolReportLongFormFinalWriteRead, mcptools.ToolReportLongFormFinalWritePatch,
		mcptools.ToolReportLongFormFinalWriteSubmit, finalEditStageSubmittedSentinel, finalEditRetryNote(attempt))
}

func agentLongFormReaderEditPrompt(req longFormReaderStyleGatePipelineRequest, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	if pipeline := req.finalEditPipeline(); pipeline == reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2 || pipeline == reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		return agentLongFormV2ReaderEditPrompt(req, binding, draftID, attempt)
	}
	return fmt.Sprintf(`Read and edit the durable long-form Part manuscript through the dedicated reader-edit MCP tools.

Report title: %s
Mission ID: %s
Bound stage metadata:
%s

Use exactly this workflow:
1. Call %s with draft_id %q and the bound identities.
2. Read the full manuscript with %s until truncated is false.
3. Use %s for reader-facing edits whenever the manuscript can be made clearer for a report-only reader without changing meaning: improve opening, transitions, ordering, repetition, conclusion, clarity, and direct explanation of the subject.
4. Submit with %s using the same draft_id, pending_event_id, and plan_event_id.
5. Return exactly %s and nothing else.

Reader-edit responsibilities:
- Explain the subject as the report's author to a reader who will only see this report. Digest the material and present the explanation instead of telling the reader how to interpret the sources. Use source-boundary language only where it changes claim scope or certainty.
- Do not optimize for brevity by itself. Keep or add explanation when it makes a supported concept, causal link, context, condition, example, or technical detail easier to understand; remove prose only when it does not advance the reader's understanding.
- Keep or create a brief report-level opening that states the subject, central question, and main answer or evidence boundary. Treat this orientation as useful content, not removable meta-signposting.
- Let later transitions follow the subject and the reader's next question. Remove repeated section roadmaps or writing-process narration, but keep transitions that add context, logic, or stakes. Clean obviously duplicated headings when their intended form is clear.
- Preserve every unique fact, citation, caveat meaning, number, code identifier, technical identifier, uncertainty boundary, and assigned requirement.
- Consolidate redundant caveats and source-process narration without losing unique information; keep the remaining limit near the claim it qualifies rather than repeating investigation-log phrasing.
- Judge repetition by function: keep a brief reminder when a long-form reader or a new context needs it; remove adjacent restatements and section-level duplication, keeping the strongest occurrence and merging unique detail into it.
- Submit unchanged only after a full read finds none of these responsibilities applicable.

Do not call research or source tools. Do not expose IDs in the manuscript.%s`,
		req.title, req.missionID, agentReportAnyJSON(binding),
		mcptools.ToolReportLongFormReaderEditStart, draftID,
		mcptools.ToolReportLongFormReaderEditRead, mcptools.ToolReportLongFormReaderEditPatch,
		mcptools.ToolReportLongFormReaderEditSubmit, finalEditStageSubmittedSentinel, finalEditRetryNote(attempt))
}

func agentLongFormV2ReaderEditPrompt(req longFormReaderStyleGatePipelineRequest, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	return fmt.Sprintf(`Read and edit the final-writer manuscript through the dedicated reader-edit MCP tools.

Report title: %s
Mission ID: %s
Bound stage metadata:
%s

Use exactly this workflow:
1. Call %s with draft_id %q and the bound identities.
2. Read the full manuscript with %s until truncated is false.
3. Use %s for reader-facing edits that improve direct explanation, paragraph flow, comprehension order, memory-supporting accumulation, and awkward wording without changing meaning.
4. Submit with %s using the same draft_id, pending_event_id, and plan_event_id.
5. Return exactly %s and nothing else.

Reader-edit responsibilities:
- Explain the subject as the report's author to a reader who will only see this report. Improve local clarity, sequence, and paragraph-level comprehension.
- Do not create a new opening, new conclusion, global redesign, full Part or Section reorder, or cross-Part restructure; those are final-writer responsibilities already completed.
- Do not optimize for brevity by itself. Keep or add explanation when it makes a supported concept, causal link, condition, example, or technical detail easier to understand.
- Preserve every unique fact, citation, caveat meaning, number, code identifier, technical identifier, uncertainty boundary, and assigned requirement.
- Consolidate adjacent repetition only when no unique information is lost; keep memory-supporting reminders when a long-form reader needs them.
- Submit unchanged only after a full read finds none of these responsibilities applicable.

Do not call research or source tools. Do not expose IDs in the manuscript.%s`,
		req.title, req.missionID, agentReportAnyJSON(binding),
		mcptools.ToolReportLongFormReaderEditStart, draftID,
		mcptools.ToolReportLongFormReaderEditRead, mcptools.ToolReportLongFormReaderEditPatch,
		mcptools.ToolReportLongFormReaderEditSubmit, finalEditStageSubmittedSentinel, finalEditRetryNote(attempt))
}

func agentLongFormStyleEditPrompt(req longFormReaderStyleGatePipelineRequest, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	return fmt.Sprintf(`Apply the pre-canonical Korean natural-voice style pass through the dedicated style-edit MCP tools.

Report title: %s
Mission ID: %s
Bound stage metadata:
%s

Use exactly this workflow:
1. Call %s with draft_id %q and the bound identities.
2. Read the full manuscript with %s until truncated is false.
3. Privately diagnose every paragraph before patching. Use only these categories and meanings:
   - opaque_or_strained_mapping = relationship between domains is not quickly recoverable, collocation feels invented/strained, or image adds interpretation cost without explanatory gain; do not prohibit metaphor and preserve conventional/clarifying metaphor.
   - unnatural_collocation = grammatical words that do not sound like normal Korean report prose in context.
   - vague_reference = unclear/cheap pointer instead of naming the referent.
   - nominalized_or_bureaucratic = noun/process-heavy phrase hiding a simple action.
   - compressed_abstraction = several ideas packed into an abstract phrase that costs effort to unpack.
   - report_process_meta = report/section narration where subject matter should be foregrounded.
   - formulaic_transition = stock movement announcement with no useful logic.
4. No edit quota/minimum.
5. Use %s only with exact replace operations. Never use insert_after, append, or replace_all. Empty replacement is allowed only for a diagnosed local deletion that leaves its Markdown block non-empty.
6. Each patch summary must use exactly this format: category: <one-known-token>; <concrete issue>. The category token must be one of the seven diagnosis categories above.
7. Preserve structure, claims, citations, and paragraph boundaries. Never change heading lines, table rows, list markers, blockquote lines, code fences or fenced code, or source/reference lines.
8. In every replacement, keep the exact ordered sequence of numbers, Latin technical tokens, inline-code spans, quoted spans, links, footnotes, and citation markers. If a repair overlaps one, copy the protected token or span verbatim and edit only the surrounding Korean; otherwise skip the repair.
9. For report_process_meta, skip the repair when the navigation phrase contains a number or Latin token. Forbidden examples: changing "1부에서" to "앞에서", changing "이 Part에서는" to "여기서는", or rewriting any table row.
10. Treat each successful replacement as the current draft. Before a later repair that overlaps earlier text, read the affected range again and use the current exact match.
11. Before submit, reread every changed range and verify that its protected sequences are unchanged. Repair any mismatch before submitting.
12. Submit unchanged if no safe local repair is justified.
13. Submit with %s. After submit succeeds, make no further tool calls and return exactly %s.

Do not summarize, add facts, call research/source tools, or expose IDs in the manuscript.%s`,
		req.title, req.missionID, agentReportAnyJSON(binding),
		mcptools.ToolReportLongFormStyleEditStart, draftID,
		mcptools.ToolReportLongFormStyleEditRead, mcptools.ToolReportLongFormStyleEditPatch,
		mcptools.ToolReportLongFormStyleEditSubmit, finalEditStageSubmittedSentinel, finalEditRetryNote(attempt))
}

func agentLongFormGatePrompt(req longFormReaderStyleGatePipelineRequest, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	return fmt.Sprintf(`Run the corrective provenance gate and canonicalize the long-form report through MCP.

Report title: %s
Mission ID: %s
Rigor: %s (%s)
Bound gate metadata:
%s

Global requirement preservation checks:
%s

Use exactly this workflow:
1. Call %s with draft_id %q and the bound identities.
2. Read the full manuscript with %s until truncated is false.
3. Use approved read tools to verify claims when support is unclear. Do not mutate sources or create new policy.
4. Apply required corrections with %s before submit.
5. Submit with %s and gate_findings. If there are no findings, pass an empty array. Use only these classifications: mission_source_grounded, session_grounded, derived_synthesis, rhetorical_construction, unverified_external_fact. Use only these repair actions: attach_approved_evidence, qualify_inference_or_uncertainty, retain_with_footnote, remove.
6. Return exactly %s and nothing else after submit succeeds.

Gate responsibilities:
- Read the complete manuscript before judging it.
- Enforce source/evidence boundaries and every owner-bound output requirement according to the rigor level.
- Order repairs before canonicalization; the gate is the only canonical producer.
- Do not include raw statement text anywhere except the transient gate_findings tool input.%s`,
		req.title, req.missionID, req.rigor.level, req.rigor.label, agentReportAnyJSON(binding),
		agentReportAnyJSON(reporting.ReportOwnerBoundRequirements(req.requirementMap)),
		mcptools.ToolReportLongFormEditStart, draftID,
		mcptools.ToolReportLongFormEditRead, mcptools.ToolReportLongFormEditPatch,
		mcptools.ToolReportLongFormEditSubmit, finalEditGateSubmittedSentinel, finalEditRetryNote(attempt))
}

func agentLongFormSemanticGatePrompt(req longFormReaderStyleGatePipelineRequest, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	return fmt.Sprintf(`Run the corrective provenance gate and canonicalize the long-form report through MCP.

Report title: %s
Mission ID: %s
Rigor: %s (%s)
Bound gate metadata:
%s

Global requirement preservation checks:
%s

Use exactly this workflow:
1. Call %s with draft_id %q and the bound identities.
2. Read the full manuscript with %s until truncated is false.
3. Use approved read tools to verify claims when support is unclear. Do not mutate sources or create new policy.
4. Read changed style paragraphs with %s until truncated is false. Follow next_offset exactly; the gate cannot submit until this bounded review read returns truncated=false. For each changed source paragraph, submit paragraph_ordinal, final_paragraph_ordinal, and exactly one semantic verdict: accepted_equivalent when style and final meaning are equivalent, reverted_to_reader when final text returns to reader meaning, or repaired_by_gate when you repaired unsafe style drift.
5. Apply required corrections with %s before submit. Semantic review is not an invitation to rewrite; fail closed on uncertainty by reverting the affected paragraph to reader meaning or repairing only the unsafe local drift.
6. Submit with %s, gate_findings, and semantic_acceptance. If there are no findings, pass an empty array. If there were no changed style paragraphs, omit semantic_acceptance or pass an empty array. Use only these classifications: mission_source_grounded, session_grounded, derived_synthesis, rhetorical_construction, unverified_external_fact. Use only these repair actions: attach_approved_evidence, qualify_inference_or_uncertainty, retain_with_footnote, remove.
7. Return exactly %s and nothing else after submit succeeds.

Gate responsibilities:
- Read the complete manuscript before judging it.
- Enforce source/evidence boundaries and every owner-bound output requirement according to the rigor level.
- Order repairs before canonicalization; the gate is the only canonical producer.
- Do not include raw statement text anywhere except the transient gate_findings tool input.%s`,
		req.title, req.missionID, req.rigor.level, req.rigor.label, agentReportAnyJSON(binding),
		agentReportAnyJSON(reporting.ReportOwnerBoundRequirements(req.requirementMap)),
		mcptools.ToolReportLongFormEditStart, draftID,
		mcptools.ToolReportLongFormEditRead, mcptools.ToolReportLongFormStyleReviewRead, mcptools.ToolReportLongFormEditPatch,
		mcptools.ToolReportLongFormEditSubmit, finalEditGateSubmittedSentinel, finalEditRetryNote(attempt))
}

func agentLongFormStyleSemanticValidationPrompt(req longFormReaderStyleGatePipelineRequest, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	return fmt.Sprintf(`Run read-only style semantic validation for the long-form report through MCP.

Report title: %s
Mission ID: %s
Bound stage metadata:
%s

Use exactly this workflow:
1. Read changed reader/style paragraph comparisons with %s until truncated is false.
2. Submit with %s using semantic_acceptance. For each changed paragraph submit paragraph_ordinal and exactly one verdict: accepted_equivalent or rejected_revert_to_reader.
3. Return exactly %s and nothing else after submit succeeds.

Validation responsibilities:
- Judge only whether the style paragraph preserves the reader paragraph's meaning.
- Do not submit prose, patches, final paragraph ordinals, repaired_by_gate, manuscript Markdown, or repair instructions.
- When uncertain, use rejected_revert_to_reader.%s`,
		req.title, req.missionID, agentReportAnyJSON(binding),
		mcptools.ToolReportLongFormStyleSemanticValidationRead,
		mcptools.ToolReportLongFormStyleSemanticValidationSubmit,
		finalEditStageSubmittedSentinel, finalEditRetryNote(attempt))
}

func agentLongFormEvidenceGatePrompt(req longFormReaderStyleGatePipelineRequest, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	return fmt.Sprintf(`Run read-only evidence connection judgment and canonicalize the long-form report through MCP.

Report title: %s
Mission ID: %s
Rigor: %s (%s)
Bound stage metadata:
%s

Use exactly this workflow:
1. Read the report passage packet with %s until truncated is false. Use only server-provided statement_sha256 values from that packet.
2. Use approved read tools to verify report-to-evidence connections when support is unclear.
3. Submit with %s and gate_findings. Each finding may contain only statement_sha256, classification, and approved evidence_ids. Use only these classifications: mission_source_grounded, session_grounded, derived_synthesis, rhetorical_construction, unverified_external_fact.
4. Return exactly %s and nothing else after submit succeeds.

Evidence gate responsibilities:
- Judge report-to-evidence connections only.
- Do not judge owner requirements, prose quality, style, or structure.
- Do not calculate statement hashes; copy statement_sha256 exactly from the read packet.
- Do not submit prose, patches, repair actions, manuscript Markdown, semantic acceptance, or operation counts.
- Evidence judgments do not trigger automatic repair; the server canonicalizes the exact bound source artifact with zero operations.%s`,
		req.title, req.missionID, req.rigor.level, req.rigor.label, agentReportAnyJSON(binding),
		mcptools.ToolReportLongFormEvidenceGateRead,
		mcptools.ToolReportLongFormEvidenceGateSubmit,
		finalEditGateSubmittedSentinel, finalEditRetryNote(attempt))
}

func agentLongFormGatePromptForHumanize(humanize string) func(longFormReaderStyleGatePipelineRequest, reporting.FinalEditStageBinding, string, int) string {
	if humanize == reporting.FinalEditHumanizeEnabled {
		return agentLongFormSemanticGatePrompt
	}
	return agentLongFormGatePrompt
}

func finalEditRetryNote(attempt int) string {
	if attempt <= 1 {
		return ""
	}
	return "\n\nThis is the one allowed technical retry. Reopen the durable stage and complete the same bound workflow without changing the contract."
}

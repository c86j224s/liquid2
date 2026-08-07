package web

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

type reportPartEditorRequest struct {
	title                        string
	missionID                    string
	pendingEventID               string
	planEventID                  string
	toolSessionID                string
	previousSessionID            string
	editedArtifactID             string
	filename                     string
	executorName                 string
	agentModel                   string
	agentReasoningEffort         string
	agentSelectionSource         string
	mcpMode                      string
	rigor                        reportRigorProfile
	plan                         agentSectionalReportPlan
	part                         agentReportPart
	partIndex                    int
	source                       sectionalReportPartDraft
	directionHint                string
	requirements                 []reporting.ReportRequirement
	requirementMapEvent          app.LedgerEvent
	requirementMap               reporting.ReportRequirementMap
	reportSessionPolicy          string
	reportSessionPolicySelection string
	generationGuidanceProfile    string
	generationGuidanceSHA256     string
	sessionChainKind             string
	reportPlanSessionID          string
	forkSourceAgentSessionID     string
}

func longFormPartEditEnabled(profile string) bool {
	return reportprompt.IsNarrativeContract(profile)
}

func (server *Server) runPartEditorAgent(ctx context.Context, req reportPartEditorRequest, executor AgentExecutor) (sectionalReportPartDraft, AgentResult, error) {
	binding, err := server.partEditBinding(ctx, req)
	if err != nil {
		return sectionalReportPartDraft{}, AgentResult{}, err
	}
	if _, _, err := reporting.StartPartEdit(ctx, server.service, newID("evt"), binding); err != nil {
		return sectionalReportPartDraft{}, AgentResult{}, err
	}
	draftID := newID("rpe")
	result, runErr := executor.Run(ctx, AgentRequest{
		UserText:          fmt.Sprintf("edit assembled part %d of the long-form report", req.partIndex+1),
		Prompt:            withLongFormDownstreamDirection(agentPartEditorPrompt(req, binding, draftID), req.directionHint),
		Model:             req.agentModel,
		ReasoningEffort:   req.agentReasoningEffort,
		MissionID:         req.missionID,
		ToolSessionID:     req.toolSessionID,
		PreviousSessionID: req.previousSessionID,
		AgentExecutor:     req.executorName,
		MCPMode:           req.mcpMode,
		ExtraMCPTools:     reportPartEditMCPTools(),
		ReplaceMCPTools:   true,
		PartEdit:          &binding,
	})
	if runErr == nil {
		result, runErr = validatedSameSessionResult(result, req.previousSessionID)
	}
	edited, exists, loadErr := reporting.LoadPartEdit(context.WithoutCancel(ctx), server.service, binding)
	if loadErr != nil {
		return sectionalReportPartDraft{}, result, loadErr
	}
	if !exists {
		if runErr != nil {
			return sectionalReportPartDraft{}, result, runErr
		}
		return sectionalReportPartDraft{}, result, fmt.Errorf("%w: Part editor did not submit a durable edit", app.ErrConflict)
	}
	if runErr == nil && strings.TrimSpace(result.Text) != reporting.PartEditSubmittedSentinel {
		return sectionalReportPartDraft{}, result, fmt.Errorf("%w: Part editor acknowledgement was not exact", app.ErrConflict)
	}
	markdown := strings.TrimSpace(string(edited.Artifact.Content))
	return sectionalReportPartDraft{
		Title: req.source.Title, Markdown: markdown, ArtifactID: edited.Artifact.ArtifactID,
		WordCount: reportWordCount(markdown),
	}, result, nil
}

func (server *Server) partEditBinding(ctx context.Context, req reportPartEditorRequest) (reporting.PartEditBinding, error) {
	sourcePartEventID, err := server.reportPartCreatedEventID(ctx, req.missionID, req.planEventID, req.partIndex+1, req.source.ArtifactID)
	if err != nil {
		return reporting.PartEditBinding{}, err
	}
	mapHash := ""
	if strings.TrimSpace(req.requirementMapEvent.EventID) != "" {
		mapHash, _, err = reporting.ReportRequirementMapHash(req.requirementMap)
		if err != nil {
			return reporting.PartEditBinding{}, err
		}
	}
	return reporting.PartEditBinding{
		MissionID: req.missionID, PendingEventID: req.pendingEventID, PlanEventID: req.planEventID,
		SourcePartEventID: sourcePartEventID, SourceArtifactID: req.source.ArtifactID,
		EditedArtifactID: req.editedArtifactID, Filename: req.filename,
		ToolSessionID: req.toolSessionID, ProviderSessionID: req.previousSessionID, PreviousProviderSessionID: req.previousSessionID,
		IdempotencyKey:        fmt.Sprintf("report-part-edit:%s:%s:%d", req.pendingEventID, req.planEventID, req.partIndex+1),
		PartIndex:             req.partIndex + 1,
		RequirementMapEventID: strings.TrimSpace(req.requirementMapEvent.EventID),
		RequirementMapHash:    mapHash,
		AgentExecutor:         req.executorName, AgentModel: req.agentModel, AgentReasoningEffort: req.agentReasoningEffort,
		AgentSelectionSource: req.agentSelectionSource, MCPMode: req.mcpMode,
		ReportSessionPolicy: req.reportSessionPolicy, ReportSessionPolicySelection: req.reportSessionPolicySelection,
		GenerationGuidanceProfile: req.generationGuidanceProfile, GenerationGuidanceSHA256: req.generationGuidanceSHA256,
		SessionChainKind: req.sessionChainKind, ReportPlanSessionID: req.reportPlanSessionID,
		ForkSourceAgentSessionID: req.forkSourceAgentSessionID,
	}, nil
}

func agentPartEditorPrompt(req reportPartEditorRequest, binding reporting.PartEditBinding, draftID string) string {
	adjacentBoundaryGuidance := strings.TrimSpace(reportprompt.PartAdjacentBoundaryEditGuidance(req.generationGuidanceProfile))
	if adjacentBoundaryGuidance != "" {
		adjacentBoundaryGuidance = "\n" + adjacentBoundaryGuidance
	}
	return fmt.Sprintf(`Edit one assembled Part of a Korean long-form Plasma report through its dedicated MCP tools.

Report title: %s
Mission ID: %s
Part %d: %s

This is a separate Part editor role. The source Part artifact is immutable. A real edit creates a separate artifact; an unchanged review records completion while reusing the source artifact.

Part-level requirements to preserve:
%s

Overall plan and writing contract:
%s

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s

Bound MCP Part edit metadata:
%s

Mutating call identity:
- Use mission_id %q and session_id %q on every start, patch, and submit call.
- Use producer {"type":"agent_session","id":%q} on every mutating call.
- Use idempotency_key %q for start, %q, %q, ... for successive patches, and %q for submit. Never reuse one call's key for another call.

Required tool sequence:
1. Call %s once with draft_id %q and the bound mission, session, pending, plan, Part, and source artifact values.
2. Read the entire Part with %s. Continue only from returned next_offset values until truncated is false.
3. Act as an editor, not as a researcher or a new Section author. Use %s only for exact edits that improve this Part.
4. Read each affected passage again. If no material edit is justified after the full read, leave the draft unchanged.
5. Call %s once with the same draft_id and bound pending and plan IDs.
6. After submission succeeds, return exactly %s and nothing else.

Editing responsibility:
- Fix repetition, abrupt Section order, weak transitions, and logical gaps inside this Part only.
- Preserve every concrete fact, number, example, code identifier, caveat, citation, uncertainty boundary, and assigned requirement.
- Prefer the smallest edit that improves a real reading problem. Do not rewrite merely to demonstrate activity.
- Do not add researched facts, use research or source tools, change other Parts, or pre-write the report opening or conclusion.
- Do not mention prompts, experiments, internal run labels, tool session IDs, or artifact IDs in the manuscript.%s`,
		req.title, req.missionID, req.partIndex+1, req.part.Title,
		agentReportAnyJSON(req.requirements), agentReportAnyJSON(req.plan),
		req.rigor.level, req.rigor.label, req.rigor.description, req.rigor.instructions,
		agentReportAnyJSON(binding), binding.MissionID, binding.ToolSessionID, binding.ToolSessionID,
		binding.IdempotencyKey+":start", binding.IdempotencyKey+":patch-1", binding.IdempotencyKey+":patch-2", binding.IdempotencyKey+":submit",
		mcptools.ToolReportPartEditStart, draftID,
		mcptools.ToolReportPartEditRead, mcptools.ToolReportPartEditPatch,
		mcptools.ToolReportPartEditSubmit, reporting.PartEditSubmittedSentinel,
		adjacentBoundaryGuidance,
	)
}

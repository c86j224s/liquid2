package web

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type reportPartAuthorRequest struct {
	editor            reportPartEditorRequest
	partPlanningBrief string
}

func (server *Server) authorSectionFanoutParts(ctx context.Context, req sectionFanoutLongFormRequest, state sectionFanoutPlanState, progress sectionalReportProgress, parts []sectionalReportPartDraft, executor AgentExecutor) ([]sectionalReportPartDraft, []string, error) {
	authoredParts := make([]sectionalReportPartDraft, 0, len(parts))
	authoredArtifactIDs := make([]string, 0, len(parts))
	for partIndex, source := range parts {
		if edited, ok := progress.editedParts[partIndex]; ok {
			authoredParts = append(authoredParts, edited)
			authoredArtifactIDs = append(authoredArtifactIDs, edited.ArtifactID)
			continue
		}
		partPlan, ok := state.partPlans[partIndex]
		if !ok || strings.TrimSpace(partPlan.providerSessionID) == "" || strings.TrimSpace(partPlan.brief) == "" {
			return nil, nil, longFormStageFailure("part_edit", state.planEvent.EventID, partIndex+1, 0, fmt.Errorf("%w: final Part author session is missing", app.ErrConflict))
		}
		previousSessionID := strings.TrimSpace(partPlan.providerSessionID)
		editorReq := reportPartEditorRequest{
			title: req.title, missionID: req.missionID, pendingEventID: req.pendingEventID,
			planEventID: state.planEvent.EventID, previousSessionID: previousSessionID,
			executorName: req.executorName, agentModel: req.agentModel, agentReasoningEffort: req.agentReasoningEffort,
			agentSelectionSource: req.agentSelectionSource, mcpMode: req.mcpMode, rigor: req.rigor,
			plan: state.plan, part: state.plan.Parts[partIndex], partIndex: partIndex, source: source,
			requirements:        reporting.ReportRequirementsForPart(state.requirementMap, partIndex+1),
			requirementMapEvent: state.requirementMapEvent, requirementMap: state.requirementMap,
			reportSessionPolicy: state.reportSessionPolicy, reportSessionPolicySelection: state.reportSessionPolicySelection,
			generationGuidanceProfile: req.generationGuidanceProfile, generationGuidanceSHA256: req.generationGuidanceSHA256,
			sessionChainKind: state.sessionChainKind, reportPlanSessionID: state.reportPlanSessionID,
			forkSourceAgentSessionID: state.reportPlanSessionID,
		}
		if recovered, ok, err := server.currentPartEditStart(ctx, editorReq, previousSessionID); err != nil {
			return nil, nil, longFormStageFailure("part_edit", state.planEvent.EventID, partIndex+1, 0, err)
		} else if ok {
			editorReq.toolSessionID = recovered.ToolSessionID
			editorReq.previousSessionID = recovered.ProviderSessionID
			editorReq.editedArtifactID = recovered.EditedArtifactID
			editorReq.filename = recovered.Filename
			editorReq.forkSourceAgentSessionID = recovered.ForkSourceAgentSessionID
		} else {
			editorReq.toolSessionID = newID("ses")
			editorReq.editedArtifactID = newID("art")
			editorReq.filename = safeFilename(fmt.Sprintf("%s part %02d edited", req.title, partIndex+1), ".md")
		}
		started := time.Now()
		edited, result, err := server.runPartAuthorAgent(ctx, reportPartAuthorRequest{
			editor:            editorReq,
			partPlanningBrief: partPlan.brief,
		}, executor)
		if err != nil {
			return nil, nil, longFormStageFailure("part_edit", state.planEvent.EventID, partIndex+1, 0,
				reportAgentFailure(err, result, "report_part_edit", time.Since(started).Milliseconds(), previousSessionID))
		}
		if edited.ArtifactID == "" {
			return nil, nil, longFormStageFailure("part_edit", state.planEvent.EventID, partIndex+1, 0, fmt.Errorf("%w: final Part author artifact is missing", app.ErrConflict))
		}
		authoredParts = append(authoredParts, edited)
		authoredArtifactIDs = append(authoredArtifactIDs, edited.ArtifactID)
	}
	return authoredParts, authoredArtifactIDs, nil
}

func (server *Server) runPartAuthorAgent(ctx context.Context, req reportPartAuthorRequest, executor AgentExecutor) (sectionalReportPartDraft, AgentResult, error) {
	binding, err := server.partEditBinding(ctx, req.editor)
	if err != nil {
		return sectionalReportPartDraft{}, AgentResult{}, err
	}
	if _, _, err := reporting.StartPartEdit(ctx, server.service, newID("evt"), binding); err != nil {
		return sectionalReportPartDraft{}, AgentResult{}, err
	}
	draftID := newID("rpe")
	result, runErr := executor.Run(ctx, AgentRequest{
		UserText:          fmt.Sprintf("write final part %d of the long-form report", req.editor.partIndex+1),
		Prompt:            agentPartAuthorPrompt(req, binding, draftID),
		Model:             req.editor.agentModel,
		ReasoningEffort:   req.editor.agentReasoningEffort,
		MissionID:         req.editor.missionID,
		ToolSessionID:     req.editor.toolSessionID,
		PreviousSessionID: req.editor.previousSessionID,
		AgentExecutor:     req.editor.executorName,
		MCPMode:           req.editor.mcpMode,
		ExtraMCPTools:     reportPartEditMCPTools(),
		ReplaceMCPTools:   true,
		PartEdit:          &binding,
	})
	if runErr == nil {
		result, runErr = validatedSameSessionResult(result, req.editor.previousSessionID)
	}
	edited, exists, loadErr := reporting.LoadPartEdit(context.WithoutCancel(ctx), server.service, binding)
	if loadErr != nil {
		return sectionalReportPartDraft{}, result, loadErr
	}
	if !exists {
		if runErr != nil {
			return sectionalReportPartDraft{}, result, runErr
		}
		return sectionalReportPartDraft{}, result, fmt.Errorf("%w: final Part author did not submit a durable Part", app.ErrConflict)
	}
	if runErr == nil && strings.TrimSpace(result.Text) != reporting.PartEditSubmittedSentinel {
		return sectionalReportPartDraft{}, result, fmt.Errorf("%w: final Part author acknowledgement was not exact", app.ErrConflict)
	}
	markdown := strings.TrimSpace(string(edited.Artifact.Content))
	return sectionalReportPartDraft{
		Title: req.editor.source.Title, Markdown: markdown, ArtifactID: edited.Artifact.ArtifactID,
		WordCount: reportWordCount(markdown),
	}, result, nil
}

func agentPartAuthorPrompt(req reportPartAuthorRequest, binding reporting.PartEditBinding, draftID string) string {
	editor := req.editor
	return fmt.Sprintf(`Write one final Part of a Korean long-form Plasma report through its dedicated MCP Part edit tools.

You are the final author of this Part. The assembled Sections are drafting material, not immutable manuscript prose.

Report title: %s
Mission ID: %s
Part %d: %s

Part-level requirements to preserve:
%s

Overall plan and writing contract:
%s

Part planning brief:
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
3. Use %s for exact edits. A purposeful whole-document exact replacement is allowed when it produces a more coherent final Part within the existing Part bounds.
4. Reread the affected passages, and reread the whole Part before submitting.
5. Call %s once with the same draft_id and bound pending and plan IDs.
6. After submission succeeds, return exactly %s and nothing else.

Authorship responsibility:
- Read all input before writing: the current Part, the Part brief, the overall plan, the writing contract, rigor, and assigned requirements.
- Use the Part brief to recover the intended reader movement.
- Write one coherent standalone Part, not a stitched inventory of Sections.
- Keep the Part title and planned Section headings/order.
- Treat Section prose as material that may be substantially rewritten, merged, shortened, or moved within the planned order.
- Preserve every fact, number, example, code identifier, caveat, citation, uncertainty boundary, and assigned requirement.
- Purposeful spaced recall is allowed only when it recontextualizes, applies, or decides something for the reader.
- Merge or remove restatement that gives the reader no new job.
- Explain the subject directly, with sources backstage as support for claims rather than recurring sentence subjects.
- Localize uncertainty beside the claim it qualifies.
- Add no researched facts, use no research or evidence tools, and do not change other Parts.
- Do not mention prompts, experiments, internal run labels, tool session IDs, or artifact IDs in the manuscript.`,
		editor.title, editor.missionID, editor.partIndex+1, editor.part.Title,
		agentReportAnyJSON(editor.requirements), agentReportAnyJSON(editor.plan), strings.TrimSpace(req.partPlanningBrief),
		editor.rigor.level, editor.rigor.label, editor.rigor.description, editor.rigor.instructions,
		agentReportAnyJSON(binding), binding.MissionID, binding.ToolSessionID, binding.ToolSessionID,
		binding.IdempotencyKey+":start", binding.IdempotencyKey+":patch-1", binding.IdempotencyKey+":patch-2", binding.IdempotencyKey+":submit",
		plasmamcp.ToolReportPartEditStart, draftID,
		plasmamcp.ToolReportPartEditRead, plasmamcp.ToolReportPartEditPatch,
		plasmamcp.ToolReportPartEditSubmit, reporting.PartEditSubmittedSentinel,
	)
}

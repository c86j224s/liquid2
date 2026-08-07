package web

import (
	"context"
	"fmt"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) editSectionFanoutParts(ctx context.Context, req sectionFanoutLongFormRequest, state sectionFanoutPlanState, progress sectionalReportProgress, parts []sectionalReportPartDraft, forker AgentSessionForker, executor AgentExecutor) ([]sectionalReportPartDraft, []string, error) {
	editedParts := make([]sectionalReportPartDraft, 0, len(parts))
	editedArtifactIDs := make([]string, 0, len(parts))
	for partIndex, source := range parts {
		if edited, ok := progress.editedParts[partIndex]; ok {
			editedParts = append(editedParts, edited)
			editedArtifactIDs = append(editedArtifactIDs, edited.ArtifactID)
			continue
		}
		editorReq := reportPartEditorRequest{
			title: req.title, missionID: req.missionID, pendingEventID: req.pendingEventID,
			planEventID:  state.planEvent.EventID,
			executorName: req.executorName, agentModel: req.agentModel, agentReasoningEffort: req.agentReasoningEffort,
			agentSelectionSource: req.agentSelectionSource, mcpMode: req.mcpMode, rigor: req.rigor,
			plan: state.plan, part: state.plan.Parts[partIndex], partIndex: partIndex, source: source,
			directionHint:       req.directionHint,
			requirements:        reporting.ReportRequirementsForPart(state.requirementMap, partIndex+1),
			requirementMapEvent: state.requirementMapEvent, requirementMap: state.requirementMap,
			reportSessionPolicy: state.reportSessionPolicy, reportSessionPolicySelection: state.reportSessionPolicySelection,
			generationGuidanceProfile: req.generationGuidanceProfile, generationGuidanceSHA256: req.generationGuidanceSHA256,
			sessionChainKind: state.sessionChainKind, reportPlanSessionID: state.reportPlanSessionID,
			forkSourceAgentSessionID: state.reportPlanSessionID,
		}
		if recovered, ok, err := server.currentPartEditStart(ctx, editorReq, ""); err != nil {
			return nil, nil, longFormStageFailure("part_edit", state.planEvent.EventID, partIndex+1, 0, err)
		} else if ok {
			editorReq.toolSessionID = recovered.ToolSessionID
			editorReq.previousSessionID = recovered.ProviderSessionID
			editorReq.editedArtifactID = recovered.EditedArtifactID
			editorReq.filename = recovered.Filename
			editorReq.forkSourceAgentSessionID = recovered.ForkSourceAgentSessionID
		} else {
			previousSessionID, forkSourceID, err := forkSectionFanoutSession(ctx, forker, state.reportPlanSessionID)
			if err != nil {
				return nil, nil, longFormStageFailure("part_edit", state.planEvent.EventID, partIndex+1, 0, err)
			}
			if forkSourceID == "" {
				forkSourceID = state.reportPlanSessionID
			}
			editorReq.toolSessionID = newID("ses")
			editorReq.previousSessionID = previousSessionID
			editorReq.editedArtifactID = newID("art")
			editorReq.filename = safeFilename(fmt.Sprintf("%s part %02d edited", req.title, partIndex+1), ".md")
			editorReq.forkSourceAgentSessionID = forkSourceID
		}
		started := time.Now()
		edited, result, err := server.runPartEditorAgent(ctx, editorReq, executor)
		if err != nil {
			return nil, nil, longFormStageFailure("part_edit", state.planEvent.EventID, partIndex+1, 0,
				reportAgentFailure(err, result, "report_part_edit", time.Since(started).Milliseconds(), editorReq.previousSessionID))
		}
		if edited.ArtifactID == "" {
			return nil, nil, longFormStageFailure("part_edit", state.planEvent.EventID, partIndex+1, 0, fmt.Errorf("%w: edited Part artifact is missing", app.ErrConflict))
		}
		editedParts = append(editedParts, edited)
		editedArtifactIDs = append(editedArtifactIDs, edited.ArtifactID)
	}
	return editedParts, editedArtifactIDs, nil
}

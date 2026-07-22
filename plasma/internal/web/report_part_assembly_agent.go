package web

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type reportPartAssemblyAgentRequest struct {
	title                        string
	missionID                    string
	toolSessionID                string
	previousSessionID            string
	pendingEventID               string
	planEventID                  string
	executorName                 string
	agentModel                   string
	agentReasoningEffort         string
	agentSelectionSource         string
	mcpMode                      string
	rigor                        reportRigorProfile
	plan                         agentSectionalReportPlan
	part                         agentReportPart
	drafts                       []sectionalReportDraft
	partIndex                    int
	reportSessionPolicy          string
	reportSessionPolicySelection string
	postReportHumanize           string
	generationGuidanceProfile    string
	generationGuidanceSHA256     string
	sessionChainKind             string
	preReportResearchSessionID   string
	reportPlanSessionID          string
	forkSourceAgentSessionID     string
}

func (server *Server) runPartAssemblyAgent(ctx context.Context, req reportPartAssemblyAgentRequest, executor AgentExecutor) (agentPartAssembly, AgentResult, string, error) {
	agentReq := AgentRequest{
		UserText:          fmt.Sprintf("assemble part %d for sectional long-form markdown report", req.partIndex+1),
		Prompt:            agentPartAssemblyPrompt(req.title, req.missionID, req.toolSessionID, req.rigor, req.plan, req.part, req.drafts, req.partIndex, req.generationGuidanceProfile),
		Model:             req.agentModel,
		ReasoningEffort:   req.agentReasoningEffort,
		MissionID:         req.missionID,
		ToolSessionID:     req.toolSessionID,
		PreviousSessionID: req.previousSessionID,
		AgentExecutor:     req.executorName,
		MCPMode:           req.mcpMode,
		ExtraMCPTools:     reportReadMCPTools(),
		ReplaceMCPTools:   true,
	}
	var binding reporting.PartAssemblyBinding
	if usePartAssemblyEditTools(req.generationGuidanceProfile) {
		binding = req.partAssemblyBinding()
		agentReq.Prompt = agentPartAssemblyEditToolsPrompt(req, binding, newID("rpa"))
		agentReq.ExtraMCPTools = reportPartAssemblyMCPTools(req.generationGuidanceProfile)
		agentReq.PartAssembly = &binding
	}
	result, err := executor.Run(ctx, agentReq)
	if err != nil {
		return agentPartAssembly{}, result, "", err
	}
	returnedSessionID := strings.TrimSpace(result.SessionID)
	result, err = validatedSameSessionResult(result, req.previousSessionID)
	if err != nil {
		return agentPartAssembly{}, result, returnedSessionID, err
	}
	if !usePartAssemblyEditTools(req.generationGuidanceProfile) {
		assembly, parseErr := parseAgentPartAssembly(result.Text)
		return assembly, result, returnedSessionID, parseErr
	}
	if strings.TrimSpace(result.Text) != reporting.PartAssemblySubmittedSentinel {
		return agentPartAssembly{}, result, returnedSessionID, fmt.Errorf("%w: part assembly agent did not confirm MCP submission", app.ErrInvalidInput)
	}
	submission, exists, err := reporting.LoadPartAssemblySubmission(context.WithoutCancel(ctx), server.service, binding)
	if err != nil {
		return agentPartAssembly{}, result, returnedSessionID, err
	}
	if !exists {
		return agentPartAssembly{}, result, returnedSessionID, fmt.Errorf("%w: part assembly MCP submission is missing", app.ErrConflict)
	}
	return submission.Assembly, result, returnedSessionID, nil
}

func (req reportPartAssemblyAgentRequest) partAssemblyBinding() reporting.PartAssemblyBinding {
	return reporting.PartAssemblyBinding{
		MissionID:                    req.missionID,
		PendingEventID:               req.pendingEventID,
		PlanEventID:                  req.planEventID,
		ToolSessionID:                req.toolSessionID,
		ProviderSessionID:            req.previousSessionID,
		PreviousProviderSessionID:    req.previousSessionID,
		PartIndex:                    req.partIndex + 1,
		SectionCount:                 len(req.drafts),
		SectionArtifactIDs:           partSectionArtifactIDs(req.drafts),
		AgentExecutor:                req.executorName,
		AgentModel:                   req.agentModel,
		AgentReasoningEffort:         req.agentReasoningEffort,
		AgentSelectionSource:         req.agentSelectionSource,
		MCPMode:                      req.mcpMode,
		ReportSessionPolicy:          req.reportSessionPolicy,
		ReportSessionPolicySelection: req.reportSessionPolicySelection,
		PostReportHumanize:           req.postReportHumanize,
		GenerationGuidanceProfile:    req.generationGuidanceProfile,
		GenerationGuidanceSHA256:     req.generationGuidanceSHA256,
		SessionChainKind:             req.sessionChainKind,
		PreReportResearchSessionID:   req.preReportResearchSessionID,
		ReportPlanSessionID:          req.reportPlanSessionID,
		ForkSourceAgentSessionID:     req.forkSourceAgentSessionID,
		Producer:                     app.Producer{Type: "agent_session", ID: req.toolSessionID},
	}
}

func partSectionArtifactIDs(drafts []sectionalReportDraft) []string {
	ids := make([]string, len(drafts))
	for index, draft := range drafts {
		ids[index] = strings.TrimSpace(draft.ArtifactID)
	}
	return ids
}

func usePartAssemblyEditTools(profile string) bool {
	return isReportGenerationGuidanceProfilePartAssemblyEditTools(profile) ||
		isReportGenerationGuidanceProfileVisualPlan(profile)
}

func agentPartAssemblyEditToolsPrompt(req reportPartAssemblyAgentRequest, binding reporting.PartAssemblyBinding, draftID string) string {
	guidance := strings.TrimSpace(LongFormReportGenerationGuidance(req.generationGuidanceProfile))
	if guidance != "" {
		guidance = "\n" + guidance + "\n"
	}
	sectionReading := ""
	sectionInventory := sectionalDraftInventoryJSON(req.drafts)
	promptBinding := binding
	if isReportGenerationGuidanceProfileNarrativeContract(req.generationGuidanceProfile) {
		sectionReading = fmt.Sprintf("\nRequired manuscript reading:\n- Call %s for every Section in this Part, following next_offset until truncated is false. Read the actual Section bodies before writing connective text.\n", plasmamcp.ToolReportPartSectionRead)
		sectionInventory = narrativePartSectionInventoryJSON(req.drafts)
		promptBinding.SectionArtifactIDs = nil
	}
	return fmt.Sprintf(`Prepare connective tissue for one Part of a Korean long-form Plasma report using MCP edit tools.

Report title: %s
Mission ID: %s
Part %d: %s

This is not a rewrite task. The Section bodies are immutable and will be mechanically inserted by Plasma. You must not submit rewritten Section bodies.

Section inventory:
%s

Overall plan:
%s

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s
%s

Bound MCP part assembly metadata:
%s

Required tool sequence:
1. Call %s once with draft_id %q, the bound mission/session/pending/plan IDs, part_index, section_count, producer, and a start idempotency_key.
%s
2. Use %s when you need to inspect the current connective draft.
3. Use %s to set only intro, transition, or closing. For a transition, after_section_index is the section number after which the transition appears; it must be before another section.
4. Call %s once with the same draft_id and bound pending/plan IDs.
5. After the submit tool succeeds, return exactly %s and nothing else.

Rules:
- Use Korean for the connective Markdown.
- Do not include immutable Section bodies in any patch.
- Do not summarize the Section bodies into a replacement overview.
- Transitions are optional, but when useful they should connect adjacent Sections without compressing them.
- Prefer one good intro and one good closing over many filler transitions.
- Do not mention prompts, experiments, internal run labels, tool session IDs, or temporary implementation details.`, req.title, req.missionID, req.partIndex+1, req.part.Title, sectionInventory, agentReportAnyJSON(req.plan), req.rigor.level, req.rigor.label, req.rigor.description, req.rigor.instructions, guidance, agentReportAnyJSON(promptBinding), plasmamcp.ToolReportPartAssemblyStart, draftID, sectionReading, plasmamcp.ToolReportPartAssemblyRead, plasmamcp.ToolReportPartAssemblyPatch, plasmamcp.ToolReportPartAssemblySubmit, reporting.PartAssemblySubmittedSentinel)
}

func narrativePartSectionInventoryJSON(drafts []sectionalReportDraft) string {
	items := make([]map[string]any, len(drafts))
	for index, draft := range drafts {
		items[index] = map[string]any{
			"section_index": index + 1,
			"title":         draft.Title,
			"word_count":    draft.WordCount,
		}
	}
	return agentReportAnyJSON(items)
}

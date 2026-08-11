package web

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/legacyfinalize"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestReportDirectionPromptAllowlist(t *testing.T) {
	hint := "DIRECTION_SENTINEL"
	allowed := withReportDirection("base prompt", hint)
	if !strings.Contains(allowed, reportexecution.DirectionAdvisory) || !strings.Contains(allowed, hint) {
		t.Fatalf("allowed prompt = %q", allowed)
	}
	for name, prompt := range map[string]string{
		"patch": AgentReportPatchPrompt("t", "mis_1", "ses_1", "evt_1", "art_1", "edit", reportexecution.PatchRequest{}),
		"part":  agentPartAssemblyPrompt("t", "mis_1", "ses_1", reportRigorProfiles["balanced"], agentSectionalReportPlan{}, agentReportPart{}, nil, 0, ""),
		"final": legacyfinalize.PromptWithRequirements(legacyfinalize.Input{
			MissionID: "mis_1", Title: "t", Rigor: reportWorkflowRigor(reportRigorProfiles["balanced"]),
			Plan: agentSectionalReportPlan{},
		}, reporting.LongFormFinalizeBinding{ToolSessionID: "ses_1", PendingEventID: "evt_1", PlanEventID: "evt_2", IdempotencyKey: "key"}, 1, false, reporting.LongFormFinalizationHint{}),
	} {
		if strings.Contains(prompt, hint) || strings.Contains(prompt, reportexecution.DirectionAdvisory) {
			t.Fatalf("%s leaked direction", name)
		}
	}
}

func TestLongFormDirectionPlanningAndDownstreamPromptContract(t *testing.T) {
	hint := "Focus on the operational trade-off, exclude vendor rankings, and use a compact comparison table."
	plan := longFormDirectionTestPlan()

	planning := withLongFormPlanningDirection(agentSectionalReportPlanPrompt("Long", "mis_1", "ses_1", "evt_1", "key_1", reportRigorProfiles["balanced"], ""), hint)
	assertLongFormPlanningDirectionPrompt(t, planning, hint)
	for _, forbidden := range []string{"Do not let its direction_hint change Part or Section structure", reportexecution.DirectionAdvisory} {
		if strings.Contains(planning, forbidden) {
			t.Fatalf("planning prompt retained forbidden direction rule %q:\n%s", forbidden, planning)
		}
	}

	section := withLongFormDownstreamDirection(agentSectionDraftPromptWithRequirements("Long", "mis_1", "ses_1", reportRigorProfiles["balanced"], plan, plan.Parts[0], plan.Parts[0].Sections[0], 0, 0, "", nil), hint)
	partPlanning := agentPartPlanningPrompt(sectionFanoutLongFormRequest{title: "Long", directionHint: hint}, sectionFanoutPlanState{plan: plan}, sectionFanoutPartPlanTask{partIndex: 0, part: plan.Parts[0]})
	partAssembly := withLongFormDownstreamDirection(agentPartAssemblyPrompt("Long", "mis_1", "ses_1", reportRigorProfiles["balanced"], plan, plan.Parts[0], nil, 0, ""), hint)
	partEditorReq := reportPartEditorRequest{
		title: "Long", missionID: "mis_1", pendingEventID: "evt_pending", planEventID: "evt_plan",
		toolSessionID: "ses_1", previousSessionID: "provider_1", rigor: reportRigorProfiles["balanced"],
		plan: plan, part: plan.Parts[0], partIndex: 0, directionHint: hint,
	}
	partEditor := withLongFormDownstreamDirection(agentPartEditorPrompt(partEditorReq, reporting.PartEditBinding{MissionID: "mis_1", ToolSessionID: "ses_1"}, "draft_1"), hint)
	partAuthor := withLongFormDownstreamDirection(agentPartAuthorPrompt(reportPartAuthorRequest{editor: partEditorReq, partPlanningBrief: "brief"}, reporting.PartEditBinding{MissionID: "mis_1", ToolSessionID: "ses_1"}, "draft_1"), hint)
	final := withLongFormDownstreamDirection(legacyfinalize.PromptWithRequirements(legacyfinalize.Input{
		MissionID: "mis_1", Title: "Long", Rigor: reportWorkflowRigor(reportRigorProfiles["balanced"]),
		Plan: plan, RequirementMap: reporting.ReportRequirementMap{},
	}, reporting.LongFormFinalizeBinding{ToolSessionID: "ses_final", PendingEventID: "evt_pending", PlanEventID: "evt_plan", IdempotencyKey: "key_final"}, 1, false, reporting.LongFormFinalizationHint{}), hint)
	for _, prompt := range map[string]string{"section": section, "part_planning": partPlanning, "part_assembly": partAssembly, "part_editor": partEditor, "part_author": partAuthor, "final": final} {
		assertLongFormDownstreamDirectionPrompt(t, prompt, hint)
		assertLongFormWritingContractPrompt(t, prompt)
	}
}

func TestLongFormDirectionCoverageDepthRuleDoesNotFixCounts(t *testing.T) {
	for _, expected := range []string{"adjust emphasis, interpretation, ordering, and presentation", "do not use it to reduce mission-relevant coverage or depth"} {
		if !strings.Contains(longFormDirectionCoverageDepthRule, expected) {
			t.Fatalf("coverage-depth rule lost contract phrase %q: %s", expected, longFormDirectionCoverageDepthRule)
		}
	}
	for _, forbidden := range []string{"3 Parts", "4 Parts", "9 Sections", "14 Sections", "16 Sections"} {
		if strings.Contains(longFormDirectionCoverageDepthRule, forbidden) {
			t.Fatalf("coverage-depth rule prescribed experiment-specific count %q: %s", forbidden, longFormDirectionCoverageDepthRule)
		}
	}
}

func TestLongFormDirectionPreservesMappedRequirementOwnership(t *testing.T) {
	for _, expected := range []string{"mapped report requirements", "which Section owns a concrete output", "do not duplicate a report-wide item"} {
		if !strings.Contains(longFormDownstreamDirectionGuidance, expected) {
			t.Fatalf("downstream direction guidance lost requirement-ownership phrase %q: %s", expected, longFormDownstreamDirectionGuidance)
		}
	}
}

func TestLongFormDirectionSerialAndFanoutCallSiteParity(t *testing.T) {
	hint := "Center the report on customer-impact sequencing while preserving the technical caveats."
	serial := runLongFormDirectionDraft(t, false, hint)
	fanout := runLongFormDirectionDraft(t, true, hint)

	for _, run := range []struct {
		name     string
		requests []AgentRequest
	}{
		{name: "serial", requests: serial},
		{name: "fanout", requests: fanout},
	} {
		planPrompt := findLongFormDirectionPrompt(t, run.requests, "plan ")
		assertLongFormPlanningDirectionPrompt(t, planPrompt, hint)

		for _, stage := range []string{"draft section", "assemble part", "finalize"} {
			prompt := findLongFormDirectionPrompt(t, run.requests, stage)
			assertLongFormDownstreamDirectionPrompt(t, prompt, hint)
			assertLongFormWritingContractPrompt(t, prompt)
		}
	}
}

func TestLongFormDirectionExcludedVerificationAndStylePrompts(t *testing.T) {
	hint := "DIRECTION_SENTINEL"
	plan := longFormDirectionTestPlan()
	for name, prompt := range map[string]string{
		"patch": AgentReportPatchPrompt("Long", "mis_1", "ses_1", "evt_patch", "art_1", "edit", reportexecution.PatchRequest{}),
		"canonical_final_edit": legacyfinalize.PromptWithRequirements(legacyfinalize.Input{
			MissionID: "mis_1", Title: "Long", Rigor: reportWorkflowRigor(reportRigorProfiles["balanced"]),
			Plan: plan, RequirementMap: reporting.ReportRequirementMap{}, GenerationGuidanceProfile: reportprompt.ProfileNarrativeContract,
		}, reporting.LongFormFinalizeBinding{}, 2, true, reporting.LongFormFinalizationHint{}),
	} {
		assertNoLongFormDirectionPrompt(t, name, prompt, hint)
	}
}

type longFormDirectionRouteExecutor struct {
	requests []AgentRequest
	forks    int
}

func (executor *longFormDirectionRouteExecutor) Run(_ context.Context, req AgentRequest) (AgentResult, error) {
	executor.requests = append(executor.requests, req)
	sessionID := strings.TrimSpace(req.PreviousSessionID)
	if sessionID == "" {
		sessionID = "plan-session"
	}
	switch {
	case req.ReportPlan != nil:
		return AgentResult{Text: agentReportAnyJSON(longFormDirectionTestPlan()), SessionID: sessionID}, nil
	case req.LongFormFinalize != nil:
		return AgentResult{Text: `{"front_matter":"# Long\n\nOpening.","closing":"Closing."}`, SessionID: sessionID}, nil
	case strings.Contains(req.UserText, "assemble part"):
		return AgentResult{Text: `{"intro":"Part intro.","transitions":[],"closing":"Part closing."}`, SessionID: sessionID}, nil
	default:
		return AgentResult{Text: "Section body.", SessionID: sessionID}, nil
	}
}

func (executor *longFormDirectionRouteExecutor) ForkSession(_ context.Context, sourceSessionID string) (AgentSessionForkResult, error) {
	executor.forks++
	sessionID := fmt.Sprintf("forked-session-%d", executor.forks)
	return AgentSessionForkResult{SessionID: sessionID, SourceSessionID: strings.TrimSpace(sourceSessionID)}, nil
}

func (executor *longFormDirectionRouteExecutor) CheckForkSession(_ context.Context, sourceSessionID string) error {
	if strings.TrimSpace(sourceSessionID) == "" {
		return fmt.Errorf("source session id is required")
	}
	return nil
}

func runLongFormDirectionDraft(t *testing.T, fanout bool, hint string) []AgentRequest {
	t.Helper()
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	missionID := "mis_direction"
	if _, err := service.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "Direction"}); err != nil {
		t.Fatal(err)
	}
	pendingEventID := "evt_pending_direction"
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: pendingEventID, MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"},
		Payload: mustJSON(map[string]any{"kind": "report_draft_pending", "title": "Long", "report_mode": reportModeLongForm, "direction_hint": hint, "agent_executor": "codex"}),
	}); err != nil {
		t.Fatal(err)
	}

	delegate := &longFormDirectionRouteExecutor{}
	executor := withReportPlanSubmissionFixture(service, delegate)
	server := &Server{service: service}
	rigor := reportRigorProfiles["balanced"]
	if fanout {
		_, err = server.createSectionFanoutLongFormReportDraft(ctx, missionID, "Long", hint, "codex", "gpt-test", "medium", "", "auto", rigor, reportSessionPolicySameSession, "", "disabled", "", "", pendingEventID, executor)
	} else {
		_, err = server.createSectionalLongFormReportDraft(ctx, missionID, "Long", hint, "codex", "gpt-test", "medium", "", "auto", rigor, reportSessionPolicySameSession, "", "disabled", "", "", pendingEventID, executor)
	}
	if err != nil {
		t.Fatalf("long-form direction draft failed: %v; chain: %s", err, errorChain(err))
	}
	return delegate.requests
}

func errorChain(err error) string {
	parts := []string{}
	for err != nil {
		parts = append(parts, err.Error())
		err = errors.Unwrap(err)
	}
	return strings.Join(parts, " | ")
}

func longFormDirectionTestPlan() agentSectionalReportPlan {
	return agentSectionalReportPlan{
		Summary: "Plan summary.",
		WritingContract: &reporting.ReportWritingContract{
			CentralQuestion: "direction-centered question",
			ReaderTakeaway:  "direction-centered takeaway",
			ReadingPath:     []string{"show the customer impact", "explain the caveat"},
			MustKeep:        []string{"contract sentinel detail"},
			VisualRole:      "compact comparison table",
			ToneAndShape:    "direct and bounded",
		},
		Parts: []agentReportPart{{
			Title: "Part", Purpose: "Explain the core sequence.",
			Sections: []agentReportSection{{Title: "Section", Purpose: "Explain the customer-impact sequence."}},
		}},
	}
}

func findLongFormDirectionPrompt(t *testing.T, requests []AgentRequest, userTextContains string) string {
	t.Helper()
	for _, req := range requests {
		if strings.Contains(req.UserText, userTextContains) {
			return req.Prompt
		}
	}
	t.Fatalf("missing request containing %q in %#v", userTextContains, requests)
	return ""
}

func assertLongFormPlanningDirectionPrompt(t *testing.T, prompt, hint string) {
	t.Helper()
	for _, expected := range []string{longFormPlanningDirectionGuidance, longFormDirectionCoverageDepthRule, longFormDirectionPriorityRule, longFormDirectionAdvisory, "<request_direction>", hint, "</request_direction>"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("planning prompt missing %q:\n%s", expected, prompt)
		}
	}
}

func assertLongFormDownstreamDirectionPrompt(t *testing.T, prompt, hint string) {
	t.Helper()
	for _, expected := range []string{longFormDownstreamDirectionGuidance, longFormDirectionCoverageDepthRule, longFormDirectionPriorityRule, longFormDirectionAdvisory, "<request_direction>", hint, "</request_direction>"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("downstream prompt missing %q:\n%s", expected, prompt)
		}
	}
}

func assertLongFormWritingContractPrompt(t *testing.T, prompt string) {
	t.Helper()
	for _, expected := range []string{`"writing_contract"`, "direction-centered question", "contract sentinel detail"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing writing contract %q:\n%s", expected, prompt)
		}
	}
}

func assertNoLongFormDirectionPrompt(t *testing.T, name, prompt, hint string) {
	t.Helper()
	for _, forbidden := range []string{hint, longFormPlanningDirectionGuidance, longFormDownstreamDirectionGuidance, longFormDirectionCoverageDepthRule, longFormDirectionAdvisory, "<request_direction>"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("%s prompt leaked long-form direction %q:\n%s", name, forbidden, prompt)
		}
	}
}

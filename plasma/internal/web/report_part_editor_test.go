package web

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestLongFormPartEditEnabledOnlyForAcceptedNarrativeContractProfile(t *testing.T) {
	for _, profile := range []string{
		reportprompt.ProfileNarrativeContract, "narrative_contract", "reader-first-editor", "reader_first_editor",
		reportprompt.ProfileReaderParagraphContract,
		reportprompt.ProfileCuriosityLedExplanation,
		reportprompt.ProfileCuriosityNaturalVoice,
		reportprompt.ProfileCuriosityTightVoice,
		reportprompt.ProfileEditedReadingVoice,
		reportprompt.ProfileSectionDirectReadingVoice,
		reportprompt.ProfilePartConnectiveEconomyVoice,
		reportprompt.ProfilePartConnectiveSubjectDirectSynthesis,
		reportprompt.ProfileSectionBriefNarrativeContract,
		reportprompt.ProfileSectionBriefClusterNarrativeContract,
	} {
		if !longFormPartEditEnabled(profile) {
			t.Fatalf("Part edit should be enabled for %q", profile)
		}
	}
	for _, profile := range []string{
		"", reportprompt.ProfileNone,
		reportprompt.ProfileVisualPlan,
		reportprompt.ProfileSectionBriefVisualPlan,
		reportprompt.ProfileSectionBriefClusterVisualPlan,
		reportprompt.ProfilePartAssemblyEditTools,
	} {
		if longFormPartEditEnabled(profile) {
			t.Fatalf("Part edit should stay disabled for %q", profile)
		}
	}
}

func TestRunPartEditorAgentUsesDedicatedPartEditTools(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	seedPartEditorFixture(t, ctx, svc)
	server := NewServer(svc, Options{}).(*Server)
	agent := &fakeAgentExecutor{
		responses: []AgentResult{{Text: reporting.PartEditSubmittedSentinel, SessionID: "provider-editor", Resumed: true}},
		onRun: func(runCtx context.Context, req AgentRequest) {
			if req.PartEdit == nil {
				t.Fatalf("expected Part edit binding on agent request: %#v", req)
			}
			assertReportMCPToolSurface(t, req,
				plasmamcp.ToolReportPartEditStart,
				plasmamcp.ToolReportPartEditRead,
				plasmamcp.ToolReportPartEditPatch,
				plasmamcp.ToolReportPartEditSubmit,
			)
			if !slices.Equal(req.ExtraMCPTools, reportPartEditMCPTools()) || slices.Contains(req.ExtraMCPTools, plasmamcp.ToolResearchRead) || slices.Contains(req.ExtraMCPTools, plasmamcp.ToolSourcesRead) {
				t.Fatalf("Part editor tool surface is not closed: %#v", req.ExtraMCPTools)
			}
			if !strings.Contains(req.Prompt, "PART_EDIT_SUBMITTED") || !strings.Contains(req.Prompt, "If no material edit is justified") || strings.Contains(req.Prompt, "plasma.research.read") || strings.Contains(req.Prompt, "plasma.sources.read") {
				t.Fatalf("Part editor prompt mismatch:\n%s", req.Prompt)
			}
			if _, err := reporting.FinalizePartEdit(runCtx, svc, *req.PartEdit, "evt_part_edit_done", "# Part 1\n\nEdited body.\n", 1); err != nil {
				t.Fatal(err)
			}
		},
	}

	edited, result, err := server.runPartEditorAgent(ctx, reportPartEditorRequest{
		title: "Reader Report", missionID: "mis_part_editor", pendingEventID: "evt_pending",
		planEventID: "evt_plan", toolSessionID: "ses_part_edit", previousSessionID: "provider-editor",
		editedArtifactID: "art_part_edit", filename: "reader-report-part-01-edited.md",
		executorName: "codex", agentModel: "gpt-5.5", agentReasoningEffort: "medium",
		agentSelectionSource: reporting.AgentSelectionSourceExplicitRequest, mcpMode: "auto",
		rigor: reportRigorProfiles["balanced"],
		plan:  agentSectionalReportPlan{Summary: "Plan", Parts: []agentReportPart{{Title: "Part", Sections: []agentReportSection{{Title: "Section"}}}}},
		part:  agentReportPart{Title: "Part", Sections: []agentReportSection{{Title: "Section"}}}, partIndex: 0,
		source: sectionalReportPartDraft{Title: "Part", Markdown: "# Part 1\n\nSource body.", ArtifactID: "art_part", WordCount: 4},
		requirements: []reporting.ReportRequirement{{
			RequirementID: "req_keep_thread", Instruction: "Preserve the assigned connective thread.",
			SourceEventIDs: []string{"evt_requirement"}, Owner: &reporting.ReportRequirementOwner{PartIndex: 1},
		}},
		reportSessionPolicy: reportSessionPolicyIsolatedFork, reportSessionPolicySelection: "default",
		generationGuidanceProfile: reportprompt.ProfileNarrativeContract,
		generationGuidanceSHA256:  "guidance-sha", sessionChainKind: "section_fanout_report",
		reportPlanSessionID: "provider-plan", forkSourceAgentSessionID: "provider-plan",
	}, agent)
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionID != "provider-editor" || edited.ArtifactID == "art_part" || edited.Markdown != "# Part 1\n\nEdited body." {
		t.Fatalf("unexpected Part editor result: edited=%#v result=%#v", edited, result)
	}
	events, err := svc.ListEvents(ctx, "mis_part_editor")
	if err != nil {
		t.Fatal(err)
	}
	if countLedgerEvents(events, reporting.PartEditStartedEventType) != 1 || countLedgerEvents(events, reporting.PartEditedEventType) != 1 {
		t.Fatalf("Part edit lifecycle events missing: %#v", events)
	}
}

func seedPartEditorFixture(t *testing.T, ctx context.Context, svc *app.Service) {
	t.Helper()
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: "mis_part_editor", Title: "Part editor"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_part", MissionID: "mis_part_editor", MediaType: "text/markdown; charset=utf-8",
		Filename: "part-1.md", Producer: app.Producer{Type: "agent_session", ID: "provider-part"},
		Content: []byte("# Part 1\n\nSource body.\n"),
	}); err != nil {
		t.Fatal(err)
	}
	for _, request := range []app.AppendEventRequest{
		{EventID: "evt_pending", MissionID: "mis_part_editor", EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{"report_mode": "long_form"})},
		{EventID: "evt_plan", MissionID: "mis_part_editor", EventType: "report.plan.created", Producer: app.Producer{Type: "agent_session", ID: "provider-plan"}, Payload: mustJSON(map[string]any{"pending_event_id": "evt_pending", "report_mode": "long_form", "artifact_id": "art_final", "part_edit_enabled": true})},
		{EventID: "evt_part", MissionID: "mis_part_editor", EventType: "report.part.created", Producer: app.Producer{Type: "agent_session", ID: "provider-part"}, Payload: mustJSON(map[string]any{"pending_event_id": "evt_pending", "plan_event_id": "evt_plan", "artifact_id": "art_part", "part_index": 1})},
	} {
		if _, err := svc.AppendEvent(ctx, request); err != nil {
			t.Fatal(err)
		}
	}
}

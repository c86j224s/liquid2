package web

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestSectionFanoutPartEditorRecoversCrashAfterCurrentStart(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	delegate := &partEditRecoveryExecutor{}
	executor := withReportPlanSubmissionFixture(svc, delegate)
	forker, ok := executor.(AgentSessionForker)
	if !ok {
		t.Fatal("fixture executor must fork")
	}
	server := NewServer(svc, Options{}).(*Server)

	const (
		missionID = "mis_part_editor_start_recovery"
		pendingID = "evt_part_editor_start_pending"
		planID    = "evt_part_editor_start_plan"
	)
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "Part editor start recovery"}); err != nil {
		t.Fatal(err)
	}
	plan := narrativeContractTestPlan()
	partArtifact := appendRetryPartEditAttempt(t, ctx, svc, missionID, pendingID, planID, "editor_start", plan, true)
	binding, err := server.partEditBinding(ctx, reportPartEditorRequest{
		title: "Reader Report", missionID: missionID, pendingEventID: pendingID, planEventID: planID,
		toolSessionID: "ses_stored_editor_start", previousSessionID: "stored-editor-session",
		editedArtifactID: "art_stored_editor_edit", filename: "stored-editor-edit.md",
		executorName: "codex", mcpMode: "auto", rigor: reportRigorProfiles["balanced"],
		plan: plan, part: plan.Parts[0], partIndex: 0,
		source: sectionalReportPartDraft{
			Title: plan.Parts[0].Title, Markdown: string(partArtifact.Content),
			ArtifactID: partArtifact.ArtifactID, WordCount: reportWordCount(string(partArtifact.Content)),
		},
		reportSessionPolicy: reportSessionPolicySameSession, generationGuidanceProfile: reportprompt.ProfilePartConnectiveEconomyVoice,
		sessionChainKind: "section_fanout_report", reportPlanSessionID: "editor_start-plan-session",
		forkSourceAgentSessionID: "editor_start-plan-session",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := reporting.StartPartEdit(ctx, svc, "evt_editor_start_open", binding); err != nil {
		t.Fatal(err)
	}
	progress, err := server.loadSectionalReportProgress(ctx, missionID, pendingID)
	if err != nil {
		t.Fatal(err)
	}
	editedParts, _, err := server.editSectionFanoutParts(ctx, sectionFanoutLongFormRequest{
		missionID: missionID, title: "Reader Report", executorName: "codex", mcpMode: "auto",
		rigor: reportRigorProfiles["balanced"], reportSessionPolicy: reportSessionPolicySameSession,
		generationGuidanceProfile: reportprompt.ProfilePartConnectiveEconomyVoice, pendingEventID: pendingID,
	}, sectionFanoutPlanState{
		artifactID: progress.artifactID, plan: plan, planEvent: progress.planEvent, reportPlanSessionID: "editor_start-plan-session",
		reportSessionPolicy: reportSessionPolicySameSession, sessionChainKind: "section_fanout_report", partEditEnabled: true,
	}, progress, []sectionalReportPartDraft{{
		Title: plan.Parts[0].Title, Markdown: string(partArtifact.Content),
		ArtifactID: partArtifact.ArtifactID, WordCount: reportWordCount(string(partArtifact.Content)),
	}}, forker, executor)
	if err != nil {
		t.Fatal(err)
	}
	requests := partEditAgentRequests(delegate.snapshotRequests())
	if len(editedParts) != 1 || len(requests) != 1 || requests[0].PartEdit == nil {
		t.Fatalf("Part editor did not run from recovered start: parts=%#v requests=%#v", editedParts, requests)
	}
	if len(delegate.forkSources) != 0 ||
		requests[0].PreviousSessionID != binding.ProviderSessionID ||
		requests[0].ToolSessionID != binding.ToolSessionID ||
		requests[0].PartEdit.EditedArtifactID != binding.EditedArtifactID ||
		requests[0].PartEdit.Filename != binding.Filename {
		t.Fatalf("Part editor minted identity instead of adopting start: forks=%#v request=%#v binding=%#v", delegate.forkSources, requests[0], binding)
	}
}

func TestSectionFanoutFinalPartAuthorRecoversCrashAfterCurrentStart(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	delegate := &partEditRecoveryExecutor{}
	executor := withReportPlanSubmissionFixture(svc, delegate)
	server := NewServer(svc, Options{}).(*Server)

	const (
		missionID = "mis_part_author_start_recovery"
		pendingID = "evt_part_author_start_pending"
		planID    = "evt_part_author_start_plan"
		label     = "author_start"
	)
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "Part author start recovery"}); err != nil {
		t.Fatal(err)
	}
	plan := narrativeContractTestPlan()
	partOwnerSessionID := appendRetryPartPlanningAttempt(t, ctx, svc, missionID, pendingID, planID, label, plan, true)
	partArtifact := createRetryMarkdownArtifact(t, ctx, svc, missionID, "art_"+label+"_part", label+"-part.md", "# Core Part\n\nAssembled Part body.\n")
	partEvent := reporting.BuildMarkdownReportPartCreatedAppendRequest(reporting.MarkdownReportPartCreatedEventRequest{
		MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
			EventID: "evt_" + label + "_part", MissionID: missionID, PendingEventID: pendingID, PlanEventID: planID,
			Title: plan.Parts[0].Title, Artifact: partArtifact, AgentExecutor: "codex",
			AgentSessionID: partOwnerSessionID, ReportMode: reportModeLongForm,
			ReportSessionPolicy: reportSessionPolicySameSession, ReportSessionPolicySelection: reportexecution.SessionPolicySelectionExplicitSameSession,
			GenerationGuidanceProfile: reportprompt.ProfilePartConnectiveEconomyVoice,
			SessionChainKind:          "section_fanout_report", ReportPlanSessionID: label + "-report-plan-session",
			ForkSourceAgentSessionID: label + "-report-plan-session",
			Producer:                 app.Producer{Type: "agent_session", ID: partOwnerSessionID},
		},
		PartIndex: 1, SectionCount: 1, WordCount: reportWordCount(string(partArtifact.Content)),
	})
	if _, err := svc.AppendEvent(ctx, partEvent); err != nil {
		t.Fatal(err)
	}
	planEvent := eventByID(t, ctx, svc, missionID, planID)
	source := sectionalReportPartDraft{
		Title: plan.Parts[0].Title, Markdown: string(partArtifact.Content),
		ArtifactID: partArtifact.ArtifactID, WordCount: reportWordCount(string(partArtifact.Content)),
	}
	binding, err := server.partEditBinding(ctx, reportPartEditorRequest{
		title: "Reader Report", missionID: missionID, pendingEventID: pendingID, planEventID: planID,
		toolSessionID: "ses_stored_author_start", previousSessionID: partOwnerSessionID,
		editedArtifactID: "art_stored_author_edit", filename: "stored-author-edit.md",
		executorName: "codex", mcpMode: "auto", rigor: reportRigorProfiles["balanced"],
		plan: plan, part: plan.Parts[0], partIndex: 0, source: source,
		reportSessionPolicy: reportSessionPolicySameSession, reportSessionPolicySelection: reportexecution.SessionPolicySelectionExplicitSameSession,
		generationGuidanceProfile: reportprompt.ProfilePartConnectiveEconomyVoice,
		sessionChainKind:          "section_fanout_report", reportPlanSessionID: label + "-report-plan-session",
		forkSourceAgentSessionID: label + "-report-plan-session",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := reporting.StartPartEdit(ctx, svc, "evt_author_start_open", binding); err != nil {
		t.Fatal(err)
	}
	authored, _, err := server.authorSectionFanoutParts(ctx, sectionFanoutLongFormRequest{
		missionID: missionID, title: "Reader Report", executorName: "codex", mcpMode: "auto",
		rigor: reportRigorProfiles["balanced"], reportSessionPolicy: reportSessionPolicySameSession,
		reportSessionPolicySelection: reportexecution.SessionPolicySelectionExplicitSameSession,
		generationGuidanceProfile:    reportprompt.ProfilePartConnectiveEconomyVoice,
		pendingEventID:               pendingID,
	}, sectionFanoutPlanState{
		plan: plan, planEvent: planEvent, reportPlanSessionID: label + "-report-plan-session",
		reportSessionPolicy: reportSessionPolicySameSession, reportSessionPolicySelection: reportexecution.SessionPolicySelectionExplicitSameSession,
		sessionChainKind: "section_fanout_report", partEditEnabled: true, partPlanningEnabled: true,
		partPlans: map[int]sectionFanoutPartPlan{0: {brief: label + " Part owner brief.", providerSessionID: partOwnerSessionID}},
	}, sectionalReportProgress{editedParts: map[int]sectionalReportPartDraft{}}, []sectionalReportPartDraft{source}, executor)
	if err != nil {
		t.Fatal(err)
	}
	requests := partEditAgentRequests(delegate.snapshotRequests())
	if len(authored) != 1 || len(requests) != 1 || requests[0].PartEdit == nil {
		t.Fatalf("Part author did not run from recovered start: authored=%#v requests=%#v", authored, requests)
	}
	if requests[0].PreviousSessionID != partOwnerSessionID ||
		requests[0].ToolSessionID != binding.ToolSessionID ||
		requests[0].PartEdit.EditedArtifactID != binding.EditedArtifactID ||
		requests[0].PartEdit.Filename != binding.Filename {
		t.Fatalf("Part author minted identity instead of adopting start: request=%#v binding=%#v", requests[0], binding)
	}
}

func eventByID(t *testing.T, ctx context.Context, svc *app.Service, missionID string, eventID string) app.LedgerEvent {
	t.Helper()
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.EventID == eventID {
			return event
		}
	}
	t.Fatalf("event %s is missing", eventID)
	return app.LedgerEvent{}
}

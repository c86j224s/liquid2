package reporting_test

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestValidateFinalEditGateBindingsCompatibleRequiresExactGateFinalContract(t *testing.T) {
	final := testGateCompatibleFinalBinding()
	stage := testGateCompatibleStageBinding(final)
	if err := reporting.ValidateFinalEditGateBindingsCompatible(stage, final); err != nil {
		t.Fatalf("compatible bindings failed: %v", err)
	}
	for name, mutate := range map[string]func(*reporting.FinalEditStageBinding){
		"stage":           func(stage *reporting.FinalEditStageBinding) { stage.Stage = reporting.FinalEditStageReader },
		"edited artifact": func(stage *reporting.FinalEditStageBinding) { stage.EditedArtifactID = "art_other" },
		"tool session":    func(stage *reporting.FinalEditStageBinding) { stage.ToolSessionID = "ses_other" },
		"provider session": func(stage *reporting.FinalEditStageBinding) {
			stage.ProviderSessionID = "provider-other"
			stage.Producer.ID = "provider-other"
		},
		"previous provider session": func(stage *reporting.FinalEditStageBinding) { stage.PreviousProviderSessionID = "provider-other" },
		"fork source":               func(stage *reporting.FinalEditStageBinding) { stage.ForkSourceAgentSessionID = "provider-other" },
		"agent model":               func(stage *reporting.FinalEditStageBinding) { stage.AgentModel = "other-model" },
		"post report humanize": func(stage *reporting.FinalEditStageBinding) {
			stage.PostReportHumanize = reporting.FinalEditHumanizeEnabled
		},
		"generation guidance sha":     func(stage *reporting.FinalEditStageBinding) { stage.GenerationGuidanceSHA256 = strings.Repeat("b", 64) },
		"report plan session":         func(stage *reporting.FinalEditStageBinding) { stage.ReportPlanSessionID = "provider-other" },
		"pre report research session": func(stage *reporting.FinalEditStageBinding) { stage.PreReportResearchSessionID = "provider-other" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := stage
			mutate(&changed)
			if err := reporting.ValidateFinalEditGateBindingsCompatible(changed, final); err == nil {
				t.Fatal("expected incompatible bindings to fail")
			}
		})
	}
}

func TestValidateFinalEditGateBindingsCompatibleRequiresNarrativeFinal(t *testing.T) {
	final := testGateCompatibleFinalBinding()
	final.CompositionStrategy = reporting.LongFormCompositionPreserveMarkdown
	stage := testGateCompatibleStageBinding(final)
	if err := reporting.ValidateFinalEditGateBindingsCompatible(stage, final); err == nil {
		t.Fatal("expected non-narrative final binding to fail")
	}
}

func testGateCompatibleFinalBinding() reporting.LongFormFinalizeBinding {
	return reporting.LongFormFinalizeBinding{
		MissionID: "mis_gate", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ArtifactID: "art_final", Filename: "report.md", Title: "Report",
		ToolSessionID: "ses_gate", IdempotencyKey: "final-key", ProviderSessionID: "provider-gate", PreviousProviderSessionID: "provider-reader",
		PartArtifactIDs: []string{"art_part"}, CompositionStrategy: reporting.LongFormCompositionNarrativeEdit,
		AgentExecutor: "codex", AgentModel: "gpt-5", AgentReasoningEffort: "medium", AgentSelectionSource: "system", MCPMode: "locked",
		RigorLevel: "standard", RigorLabel: "Standard", ReportSessionPolicy: "reuse", ReportSessionPolicySelection: "auto",
		PostReportHumanize: reporting.FinalEditHumanizeDisabled, GenerationGuidanceProfile: "reader-style-gate",
		GenerationGuidanceSHA256: strings.Repeat("a", 64), SessionChainKind: "report_final_edit",
		PreReportResearchSessionID: "provider-research", ReportPlanSessionID: "provider-plan", ForkSourceAgentSessionID: "provider-plan",
		Producer: app.Producer{Type: "agent_session", ID: "provider-gate"},
	}
}

func testGateCompatibleStageBinding(final reporting.LongFormFinalizeBinding) reporting.FinalEditStageBinding {
	return reporting.FinalEditStageBinding{
		MissionID: final.MissionID, PendingEventID: final.PendingEventID, PlanEventID: final.PlanEventID, Title: final.Title,
		Stage: reporting.FinalEditStageGate, SourceArtifactID: "art_source", EditedArtifactID: final.ArtifactID, Filename: final.Filename,
		ToolSessionID: final.ToolSessionID, ProviderSessionID: final.ProviderSessionID, PreviousProviderSessionID: final.PreviousProviderSessionID,
		IdempotencyKey: reporting.FinalEditStageIdempotencyKey(reporting.FinalEditStageGate, final.PendingEventID, final.PlanEventID),
		AgentExecutor:  final.AgentExecutor, AgentModel: final.AgentModel, AgentReasoningEffort: final.AgentReasoningEffort, AgentSelectionSource: final.AgentSelectionSource,
		MCPMode: final.MCPMode, RigorLevel: final.RigorLevel, RigorLabel: final.RigorLabel, ReportSessionPolicy: final.ReportSessionPolicy,
		ReportSessionPolicySelection: final.ReportSessionPolicySelection, PostReportHumanize: final.PostReportHumanize,
		GenerationGuidanceProfile: final.GenerationGuidanceProfile, GenerationGuidanceSHA256: final.GenerationGuidanceSHA256,
		SessionChainKind: final.SessionChainKind, PreReportResearchSessionID: final.PreReportResearchSessionID,
		ReportPlanSessionID: final.ReportPlanSessionID, ForkSourceAgentSessionID: final.ForkSourceAgentSessionID,
		Producer: final.Producer,
	}
}

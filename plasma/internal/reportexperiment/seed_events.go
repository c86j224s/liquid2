package reportexperiment

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func pendingAppendRequest(loaded LoadedFixture, missionID, pendingID, agentModel, reasoningEffort, guidanceProfile, guidanceSHA string) ledger.AppendRequest {
	payload := map[string]any{
		"kind":                              "markdown_report_artifact_pending",
		"origin_pending_event_id":           pendingID,
		"retry_strategy":                    "initial",
		"title":                             loaded.Spec.ReportTitle,
		"direction_hint":                    loaded.Spec.DirectionHint,
		"report_mode":                       reportexecution.ModeLongForm,
		"execution_strategy":                experimentExecutionStrategy,
		"rigor_level":                       loaded.Spec.Rigor.Level,
		"rigor_label":                       loaded.Spec.Rigor.Label,
		"agent_executor":                    executorCodex,
		"agent_model":                       agentModel,
		"agent_reasoning_effort":            reasoningEffort,
		"agent_selection_source":            experimentSelectionSource,
		"mcp_mode":                          experimentMCPMode,
		"report_session_policy":             reportexecution.SessionPolicySameSession,
		"report_session_policy_selection":   experimentPolicySelection,
		"generation_guidance_profile":       guidanceProfile,
		"generation_guidance_sha256":        guidanceSHA,
		"post_report_humanize":              loaded.Spec.PostReportHumanize,
		"humanize_enabled":                  loaded.Spec.PostReportHumanize == reporting.FinalEditHumanizeEnabled,
		"fixture_id":                        loaded.Spec.FixtureID,
		"fixture_sha256":                    loaded.SHA256,
		"fixture_source_provenance_id":      loaded.Spec.SourceProvenance.ProvenanceID,
		"fixture_source_product_commit":     loaded.Spec.SourceProvenance.ProductCommit,
		"fixed_reviewed_part_fixture_count": len(loaded.Parts),
	}
	return ledger.AppendRequest{
		EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending",
		Producer: ledger.Producer{Type: "user", ID: "reportexperiment"}, Payload: mustJSON(payload),
	}
}

func missionCreatedAppendRequest(eventID, missionID, title string) ledger.AppendRequest {
	return ledger.AppendRequest{
		EventID: eventID, MissionID: missionID, EventType: "mission.created",
		Producer: ledger.Producer{Type: "user", ID: "reportexperiment"},
		Payload: mustJSON(map[string]any{
			"title":     title,
			"objective": "Run fixed reviewed Part finalization through the product V3 final tail.",
			"scope": map[string]any{
				"included": []string{"archive-local fixture seed", "reportworkflow.FinalizeLongFormPrefix"},
				"excluded": []string{"product Web/HTTP/MCP/CLI changes", "prompt changes", "provider policy changes"},
			},
		}),
	}
}

type markdownEventBaseInput struct {
	EventID, MissionID, PendingEventID, Title string
	AgentModel, ReasoningEffort               string
	ToolSessionID, ProviderSessionID          string
	Rigor                                     FixtureRigor
	PostReportHumanize                        string
	GuidanceProfile, GuidanceSHA256           string
	PreReportSessionID, PlanSessionID         string
	ForkSourceSessionID                       string
	Text                                      string
	Producer                                  ledger.Producer
}

func markdownEventBase(input markdownEventBaseInput) reporting.MarkdownReportEventBase {
	return reporting.MarkdownReportEventBase{
		EventID: input.EventID, MissionID: input.MissionID, PendingEventID: input.PendingEventID, Title: input.Title,
		AgentExecutor: executorCodex, AgentModel: input.AgentModel, AgentReasoningEffort: input.ReasoningEffort, AgentSelectionSource: experimentSelectionSource,
		AgentSessionID: input.ProviderSessionID, ReturnedAgentSessionID: input.ProviderSessionID, ToolSessionID: input.ToolSessionID,
		MCPMode: experimentMCPMode, RigorLevel: input.Rigor.Level, RigorLabel: input.Rigor.Label,
		ReportMode: reportexecution.ModeLongForm, ReportModeLabel: reportexecution.ModeLabel(reportexecution.ModeLongForm),
		ReportSessionPolicy: reportexecution.SessionPolicySameSession, ReportSessionPolicySelection: experimentPolicySelection,
		PostReportHumanize: input.PostReportHumanize, HumanizeEnabled: input.PostReportHumanize == reporting.FinalEditHumanizeEnabled,
		GenerationGuidanceProfile: input.GuidanceProfile, GenerationGuidanceSHA256: input.GuidanceSHA256,
		SessionChainKind: experimentSessionChainKind, PreReportResearchSessionID: input.PreReportSessionID, ReportPlanSessionID: input.PlanSessionID,
		ForkSourceAgentSessionID: input.ForkSourceSessionID, CompositionStrategy: experimentCompositionStrategy,
		Text: input.Text, Producer: input.Producer,
	}
}

type markdownStageBaseInput struct {
	MissionID, PendingEventID, PlanEventID, Title string
	AgentModel, ReasoningEffort                   string
	ToolSessionID, ProviderSessionID              string
	Rigor                                         FixtureRigor
	PostReportHumanize                            string
	GuidanceProfile, GuidanceSHA256               string
	PreReportSessionID, PlanSessionID             string
	ForkSourceSessionID                           string
	Text                                          string
	Producer                                      ledger.Producer
}

func markdownStageBase(input markdownStageBaseInput) reporting.MarkdownReportStageEventBase {
	return reporting.MarkdownReportStageEventBase{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID, PlanEventID: input.PlanEventID, Title: input.Title,
		AgentExecutor: executorCodex, AgentModel: input.AgentModel, AgentReasoningEffort: input.ReasoningEffort, AgentSelectionSource: experimentSelectionSource,
		AgentSessionID: input.ProviderSessionID, ReturnedAgentSessionID: input.ProviderSessionID, ToolSessionID: input.ToolSessionID,
		ReportMode: reportexecution.ModeLongForm, ReportModeLabel: reportexecution.ModeLabel(reportexecution.ModeLongForm),
		ReportSessionPolicy: reportexecution.SessionPolicySameSession, ReportSessionPolicySelection: experimentPolicySelection,
		PostReportHumanize: input.PostReportHumanize, HumanizeEnabled: input.PostReportHumanize == reporting.FinalEditHumanizeEnabled,
		GenerationGuidanceProfile: input.GuidanceProfile, GenerationGuidanceSHA256: input.GuidanceSHA256,
		SessionChainKind: experimentSessionChainKind, PreReportResearchSessionID: input.PreReportSessionID, ReportPlanSessionID: input.PlanSessionID,
		ForkSourceAgentSessionID: input.ForkSourceSessionID, CompositionStrategy: experimentCompositionStrategy, AssemblyStrategy: reporting.LongFormCompositionPreserveMarkdown,
		Text: input.Text, Producer: input.Producer,
	}
}

func createSeedArtifact(ctx context.Context, svc Service, missionID, artifactID, filename string, producer ledger.Producer, content []byte) (artifact.Raw, error) {
	return svc.CreateRawArtifact(ctx, artifact.CreateRequest{
		ArtifactID: artifactID, MissionID: missionID, MediaType: "text/markdown; charset=utf-8",
		Filename: filepath.Base(filename), Producer: producer, Content: content,
	})
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

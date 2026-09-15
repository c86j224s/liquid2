package reportexecution

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
)

func TestNormalizePipelineFamilyFailsClosedWithoutChangingMode(t *testing.T) {
	if _, err := NormalizeMode("report_il_experimental"); err == nil {
		t.Fatal("experimental family was accepted as report mode")
	}
	if _, err := NormalizePipelineFamily("planned"); err == nil {
		t.Fatal("unknown pipeline family was accepted")
	}
}

func TestUnknownFamilyNeverCallsClassicGenerator(t *testing.T) {
	called := false
	runner := Runner{GenerateDraft: func(context.Context, string, DraftRequest, string) error { called = true; return nil }}
	if err := runner.generateDraft(context.Background(), "mis_1", DraftRequest{PipelineFamily: "unknown"}, "evt_1"); err == nil {
		t.Fatal("unknown family did not fail")
	}
	if called {
		t.Fatal("classic generator called for unknown family")
	}
}

func TestIndependentFamiliesDispatchBeforeClassic(t *testing.T) {
	for _, tc := range []struct {
		family string
		want   string
	}{
		{family: reportpipeline.ExperimentalIL, want: "experimental"},
		{family: reportpipeline.Unverified, want: "unverified"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			called := ""
			runner := Runner{
				GenerateDraft: func(context.Context, string, DraftRequest, string) error { called = "classic"; return nil },
				GenerateExperimental: func(context.Context, string, DraftRequest, string) error {
					called = "experimental"
					return nil
				},
				GenerateUnverified: func(context.Context, string, DraftRequest, string) error {
					called = "unverified"
					return nil
				},
			}
			if err := runner.generateDraft(context.Background(), "mis_1", DraftRequest{PipelineFamily: tc.family}, "evt_1"); err != nil {
				t.Fatal(err)
			}
			if called != tc.want {
				t.Fatalf("dispatch = %q, want %q", called, tc.want)
			}
		})
	}
}

func TestPendingRecoversCanonicalExperimentalValues(t *testing.T) {
	event := ledger.Event{EventID: "evt_pending", Payload: []byte(`{"title":"x","agent_executor":"claude","agent_model":"wrong","agent_reasoning_effort":"low","agent_selection_source":"mission","mcp_mode":"auto","rigor_level":"balanced","rigor_label":"균형형","report_mode":"long_form","pipeline_family":"report_il_experimental","execution_strategy":"section_fanout","report_session_policy":"same_session","report_session_policy_selection":"default","post_report_humanize":"enabled","humanize_enabled":true}`)}
	req, err := DraftRequestFromPendingEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	if req.PipelineFamily != "report_il_experimental" || req.PipelineGraph != reportpipeline.ExperimentalILValidationProfilesGraph || req.ReportMode != ModeLongForm || req.AgentExecutor != "codex" || req.AgentModel != "wrong" || req.AgentReasoningEffort != "low" || req.AgentSelectionSource != "mission" || req.MCPMode != "source_read_only" || req.RigorLevel != "strict" || req.RigorLabel != "검증형" || req.ReportSessionPolicy != SessionPolicyFreshSession || req.ReportSessionPolicySelection != "experimental_fixed" || req.PostReportHumanize != "disabled" || req.ExecutionStrategy != "" {
		t.Fatalf("experimental pending was not canonicalized: %#v", req)
	}
}

func TestPendingPreservesILValidationProfiles(t *testing.T) {
	for _, tc := range []struct {
		level string
		label string
	}{
		{level: "unverified", label: "무검증"},
		{level: "exploratory", label: "탐색형"},
		{level: "strict", label: "검증형"},
	} {
		t.Run(tc.level, func(t *testing.T) {
			event := ledger.Event{EventID: "evt_pending", Payload: []byte(`{"pipeline_family":"report_il_experimental","rigor_level":"` + tc.level + `"}`)}
			req, err := DraftRequestFromPendingEvent(event)
			if err != nil {
				t.Fatal(err)
			}
			if req.RigorLevel != tc.level || req.RigorLabel != tc.label || req.PipelineGraph != reportpipeline.ExperimentalILValidationProfilesGraph {
				t.Fatalf("IL validation profile = %#v", req)
			}
		})
	}
}

func TestPendingPreservesLegacyExperimentalGraphs(t *testing.T) {
	for _, graph := range []string{
		reportpipeline.ExperimentalILEditorialGraph,
		reportpipeline.ExperimentalILReaderGraph,
		reportpipeline.ExperimentalILFlowGraph,
	} {
		t.Run(graph, func(t *testing.T) {
			event := ledger.Event{EventID: "evt_pending", Payload: []byte(`{"pipeline_family":"report_il_experimental","pipeline_graph":"` + graph + `"}`)}
			req, err := DraftRequestFromPendingEvent(event)
			if err != nil {
				t.Fatal(err)
			}
			if req.PipelineGraph != graph {
				t.Fatalf("legacy experimental graph = %q, want %q", req.PipelineGraph, graph)
			}
		})
	}
}

func TestPendingRecoversCanonicalUnverifiedValues(t *testing.T) {
	event := ledger.Event{EventID: "evt_pending", Payload: []byte(`{"title":"x","agent_executor":"claude","agent_model":"wrong","agent_reasoning_effort":"low","agent_selection_source":"mission","mcp_mode":"auto","rigor_level":"strict","rigor_label":"검증형","report_mode":"long_form","pipeline_family":"report_unverified","execution_strategy":"section_fanout","report_session_policy":"same_session","report_session_policy_selection":"default","post_report_humanize":"enabled","generation_guidance_profile":"narrative-contract"}`)}
	req, err := DraftRequestFromPendingEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	if req.PipelineFamily != reportpipeline.Unverified || req.ReportMode != ModePlanned || req.AgentExecutor != "codex" || req.AgentModel != "gpt-5.6-luna" || req.AgentReasoningEffort != "xhigh" || req.AgentSelectionSource != "unverified_fixed" || req.MCPMode != "source_read_only" || req.RigorLevel != "unverified" || req.RigorLabel != "무검증형" || req.ReportSessionPolicy != SessionPolicyFreshSession || req.ReportSessionPolicySelection != "unverified_fixed" || req.PostReportHumanize != "disabled" || req.ExecutionStrategy != "" || req.GenerationGuidanceProfile != "" {
		t.Fatalf("unverified pending was not canonicalized: %#v", req)
	}
}

func TestArticlePendingPreservesReaderIntent(t *testing.T) {
	event := ledger.Event{EventID: "evt_pending", Payload: []byte(`{"title":"글","report_mode":"one_take","output_kind":"article","article_intent":{"audience":"개발자","reader_promise":"실행 순서를 이해한다","emphasis":"첫 실제 결과"}}`)}
	req, err := DraftRequestFromPendingEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	if req.OutputKind != OutputKindArticle || req.ArticleIntent.Audience != "개발자" || req.ArticleIntent.ReaderPromise != "실행 순서를 이해한다" || req.ArticleIntent.Emphasis != "첫 실제 결과" {
		t.Fatalf("article pending lost reader intent: %#v", req)
	}
}

func TestLongFormArticlePendingPreservesExistingILPath(t *testing.T) {
	event := ledger.Event{EventID: "evt_pending", Payload: []byte(`{"title":"책자","report_mode":"long_form","pipeline_family":"report_il_experimental","output_kind":"article","article_intent":{"audience":"개발자","reader_promise":"전체 흐름을 이해한다","emphasis":"실전 방법"},"rigor_level":"strict"}`)}
	req, err := DraftRequestFromPendingEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	if req.OutputKind != OutputKindArticle || req.ReportMode != ModeLongForm || req.PipelineFamily != reportpipeline.ExperimentalIL || req.ArticleIntent.Audience != "개발자" {
		t.Fatalf("long-form article pending lost existing IL binding: %#v", req)
	}
	if err := validateArticleRequest(req); err != nil {
		t.Fatal(err)
	}
}

func TestLongFormArticlePendingPreservesExecutionStrategy(t *testing.T) {
	for _, strategy := range []string{"serial", "section_fanout"} {
		event := ledger.Event{EventID: "evt_pending", Payload: []byte(`{"report_mode":"long_form","pipeline_family":"report_il_experimental","output_kind":"article","execution_strategy":"` + strategy + `","article_intent":{"audience":"Reader","reader_promise":"Promise"}}`)}
		req, err := DraftRequestFromPendingEvent(event)
		if err != nil {
			t.Fatal(err)
		}
		if req.ExecutionStrategy != strategy {
			t.Fatalf("strategy = %q, want %q", req.ExecutionStrategy, strategy)
		}
	}
}

func TestClassicPendingNormalizationRemainsUnchanged(t *testing.T) {
	event := ledger.Event{EventID: "evt_pending", Payload: []byte(`{"title":"x","agent_executor":"codex","agent_model":"custom","agent_reasoning_effort":"high","mcp_mode":"auto","rigor_level":"balanced","report_mode":"long_form","execution_strategy":"section_fanout","report_session_policy":"same_session","report_session_policy_selection":"explicit","post_report_humanize":"enabled"}`)}
	req, err := DraftRequestFromPendingEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	if req.PipelineFamily != "" || req.AgentModel != "custom" || req.AgentReasoningEffort != "high" || req.MCPMode != "auto" || req.RigorLevel != "balanced" || req.ReportMode != ModeLongForm || req.ExecutionStrategy != "section_fanout" || req.ReportSessionPolicy != SessionPolicySameSession || req.PostReportHumanize != "enabled" {
		t.Fatalf("classic pending changed: %#v", req)
	}
}

func TestStageFailureIsTypedAndSafe(t *testing.T) {
	failure := NewStageFailure("il_narrative", "", -1, -1, errors.New("provider raw response"))
	if failure.Kind != "il_narrative" || !strings.Contains(failure.Error(), "stage failed") {
		t.Fatalf("failure = %#v", failure)
	}
}

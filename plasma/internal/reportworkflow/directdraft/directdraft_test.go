package directdraft

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

func TestRunOneTakePreservesAgentRequestAndCandidate(t *testing.T) {
	ids := &idRecorder{}
	executor := &fakeExecutor{results: []agentexec.AgentResult{{Text: "# Quick\n\nBody.", SessionID: "research-session-1"}}}
	runner := Runner{Executor: executor, NewID: ids.next, LatestSessionID: func(context.Context, string, string) string { return "research-session-1" }}
	out, err := runner.RunOneTake(context.Background(), baseInput())
	if err != nil {
		t.Fatal(err)
	}
	req := executor.requests[0]
	if req.UserText != "generate quick markdown report artifact" || req.PreviousSessionID != "research-session-1" {
		t.Fatalf("unexpected one-take request: %#v", req)
	}
	if req.Model != "gpt-test" || req.ReasoningEffort != "medium" || req.AgentExecutor != "codex" ||
		req.MCPMode != "auto" || req.MissionID != "mis_1" || req.ToolSessionID != "ses_1" {
		t.Fatalf("one-take AgentRequest binding changed: %#v", req)
	}
	if !reflect.DeepEqual(req.ExtraMCPTools, ReadMCPTools()) || !req.ReplaceMCPTools {
		t.Fatalf("unexpected one-take tools: %#v replace=%t", req.ExtraMCPTools, req.ReplaceMCPTools)
	}
	if !reflect.DeepEqual(ids.order, []string{"art", "ses"}) {
		t.Fatalf("one_take id reservation changed: %#v", ids.order)
	}
	if out.ArtifactID != "art_1" || out.ToolSessionID != "ses_1" ||
		out.PreviousSessionID != "research-session-1" || out.Markdown != "# Quick\n\nBody." ||
		out.ReportSessionPolicy != reportexecution.SessionPolicyFreshSession {
		t.Fatalf("unexpected one_take candidate: %#v", out)
	}
}

func TestRunOneTakeArticleUsesReaderIntentWithExistingTools(t *testing.T) {
	input := baseInput()
	input.OutputKind = reportexecution.OutputKindArticle
	input.ArticleIntent = reportexecution.ArticleIntent{
		Audience:      "처음 에이전트를 제품에 적용하는 엔지니어",
		ReaderPromise: "작은 실제 결과부터 검증하는 순서를 이해한다",
		Emphasis:      "기반시설보다 사용자 산출물을 먼저 만든다",
	}
	executor := &fakeExecutor{results: []agentexec.AgentResult{{Text: "# Article\n\nBody.", SessionID: "research-session-1"}}}
	runner := Runner{Executor: executor, NewID: (&idRecorder{}).next, LatestSessionID: func(context.Context, string, string) string { return "research-session-1" }}
	if _, err := runner.RunOneTake(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	req := executor.requests[0]
	if req.UserText != "generate article artifact" || !strings.Contains(req.Prompt, "This output is an article, not a report.") ||
		!strings.Contains(req.Prompt, input.ArticleIntent.Audience) || !strings.Contains(req.Prompt, input.ArticleIntent.ReaderPromise) ||
		!reflect.DeepEqual(req.ExtraMCPTools, ReadMCPTools()) {
		t.Fatalf("article did not reuse the one-take source path: %#v", req)
	}
}

func TestRunPlannedPreservesPlanSessionAndCandidate(t *testing.T) {
	ids := &idRecorder{}
	executor := &fakeExecutor{results: []agentexec.AgentResult{{Text: "# Planned\n\nBody.", SessionID: "plan-session-1", Resumed: true}}}
	runner := Runner{Executor: executor, NewID: ids.next}
	started := time.Unix(1700000000, 0)
	input := PlannedInput{
		BaseInput:                  baseInput(),
		Plan:                       reporting.ReportPlan{Summary: "Plan", Sections: []reporting.ReportPlanSection{{Title: "Section"}}},
		PlanEventID:                "evt_plan",
		PlanToolSessionID:          "ses_plan",
		ArtifactID:                 "art_plan",
		ReportPlanSessionID:        "plan-session-1",
		SessionChainKind:           "fresh_session_report",
		PreReportResearchSessionID: "research-session-1",
		WorkflowStartedAt:          started,
	}
	out, err := runner.RunPlanned(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	req := executor.requests[0]
	if req.UserText != "generate markdown report artifact" || req.PreviousSessionID != "plan-session-1" {
		t.Fatalf("unexpected planned request: %#v", req)
	}
	if req.Model != "gpt-test" || req.ReasoningEffort != "medium" || req.AgentExecutor != "codex" ||
		req.MCPMode != "auto" || req.MissionID != "mis_1" || req.ToolSessionID != "ses_1" {
		t.Fatalf("planned AgentRequest binding changed: %#v", req)
	}
	if !reflect.DeepEqual(req.ExtraMCPTools, ReadMCPTools()) || !req.ReplaceMCPTools {
		t.Fatalf("unexpected planned tools: %#v replace=%t", req.ExtraMCPTools, req.ReplaceMCPTools)
	}
	if !reflect.DeepEqual(ids.order, []string{"ses"}) {
		t.Fatalf("planned must not reserve a replacement artifact id: %#v", ids.order)
	}
	if out.ArtifactID != "art_plan" || out.PlanEventID != "evt_plan" ||
		out.PlanToolSessionID != "ses_plan" || out.ReportPlanSessionID != "plan-session-1" ||
		out.WorkflowStartedAt != started || !out.AgentResumed {
		t.Fatalf("unexpected planned candidate: %#v", out)
	}
}

func baseInput() BaseInput {
	return BaseInput{
		MissionID: "mis_1", PendingEventID: "evt_pending", Title: "Report",
		AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "medium",
		MCPMode: "auto", Rigor: reportprompt.RigorProfile{Level: "balanced", Label: "균형형"},
		ReportSessionPolicy:          reportexecution.SessionPolicyFreshSession,
		ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoFreshSession,
		PostReportHumanize:           "disabled", GenerationGuidanceProfile: reportprompt.ProfileNarrativeContract,
	}
}

type fakeExecutor struct {
	requests []agentexec.AgentRequest
	results  []agentexec.AgentResult
}

func (fake *fakeExecutor) Run(_ context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
	fake.requests = append(fake.requests, req)
	result := fake.results[0]
	fake.results = fake.results[1:]
	return result, nil
}

type idRecorder struct {
	order  []string
	counts map[string]int
}

func (rec *idRecorder) next(prefix string) string {
	if rec.counts == nil {
		rec.counts = map[string]int{}
	}
	rec.order = append(rec.order, prefix)
	rec.counts[prefix]++
	return fmt.Sprintf("%s_%d", prefix, rec.counts[prefix])
}

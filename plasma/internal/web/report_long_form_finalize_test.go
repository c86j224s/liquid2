package web

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestLongFormFinalizationRetriesOnlyFinalStageWithNarrowHint(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: agentReportAnyJSON(agentSectionalReportPlan{Summary: "Plan", Parts: []agentReportPart{{Title: "Part", Sections: []agentReportSection{{Title: "Section"}}}}}), SessionID: "provider-session"},
		{Text: "Section body.", SessionID: "provider-session"},
		{Text: `{"intro":"Part intro","transitions":[],"closing":"Part close"}`, SessionID: "provider-session"},
		{Text: `{"front_matter":"# Recovered opening","closing":"## Recovered closing",}`, SessionID: "provider-session"},
		{Text: `{"front_matter":"# Recovered opening","closing":"## Recovered closing"}`, SessionID: "provider-session"},
	}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Final retry"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Report", "report_mode": "long_form", "generation_guidance_profile": reportGenerationGuidanceProfileVisualPlan,
	})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if countEvents(detail, "report.artifact.created") != 1 || countEvents(detail, "report.plan.created") != 1 || countEvents(detail, "report.section.created") != 1 || countEvents(detail, "report.part.created") != 1 {
		t.Fatalf("final-only retry duplicated durable stages: %#v", detail["events"])
	}
	if len(agent.requests) != 5 {
		t.Fatalf("request count=%d, want plan+section+part+two final", len(agent.requests))
	}
	first, second := agent.requests[3], agent.requests[4]
	if first.ToolSessionID != second.ToolSessionID || first.LongFormFinalize == nil || second.LongFormFinalize == nil || first.LongFormFinalize.IdempotencyKey != second.LongFormFinalize.IdempotencyKey {
		t.Fatalf("final retry changed logical binding: %#v %#v", first, second)
	}
	for _, expected := range []string{"# Recovered opening", "## Recovered closing", first.ToolSessionID, first.LongFormFinalize.PendingEventID, first.LongFormFinalize.PlanEventID, first.LongFormFinalize.IdempotencyKey, mcp.ToolReportLongFormFinalize, "REPORT_FINALIZED"} {
		if !strings.Contains(second.Prompt, expected) {
			t.Fatalf("retry prompt missing %q:\n%s", expected, second.Prompt)
		}
	}
}

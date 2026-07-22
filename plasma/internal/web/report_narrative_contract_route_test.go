package web

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestNarrativeContractSerialLongFormUsesProductEditorPath(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	finalMarkdown := "# Reader Report\n\n독자가 바로 이해할 수 있는 도입입니다.\n\n## Core Part\n\n근거를 소화해 직접 설명한 본문입니다.\n\n## Conclusion\n\n핵심 판단을 정리합니다.\n"
	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: agentReportAnyJSON(narrativeContractTestPlan()), SessionID: "report-session-1"},
		{Text: "근거를 소화해 직접 설명한 본문입니다.", SessionID: "report-session-1"},
		{Text: `{"intro":"파트의 질문을 먼저 설명합니다.","transitions":[],"closing":"파트 판단을 정리합니다."}`, SessionID: "report-session-1"},
		{Text: finalMarkdown, SessionID: "report-session-1"},
	}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Narrative serial", "objective": "Explain evidence to a report-only reader"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Reader Report", "report_mode": "long_form", "rigor_level": "balanced", "generation_guidance_profile": reportGenerationGuidanceProfileNarrativeContract,
	})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	assertNarrativeContractProductRequests(t, agent.requests)
	payload := lastEventPayload(t, detail, "report.artifact.created")
	if payload["composition_strategy"] != reporting.LongFormCompositionNarrativeEdit || payload["assembly_strategy"] != "narrative_contract_final_edit" {
		t.Fatalf("serial candidate metadata mismatch: %#v", payload)
	}
	artifact, err := svc.GetRawArtifact(ctx, payload["artifact_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if string(artifact.Content) != finalMarkdown {
		t.Fatalf("serial product path did not persist exact edited manuscript:\n%s", artifact.Content)
	}
}

func TestNarrativeContractSectionFanoutUsesSameProductEditorPath(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	finalMarkdown := "# Fanout Reader Report\n\n병렬로 작성된 내용을 하나의 설명으로 엽니다.\n\n## Core Part\n\n병렬 섹션의 근거를 독자에게 직접 설명합니다.\n\n## Conclusion\n\n전체 판단을 연결합니다.\n"
	agent := &fakeForkingAgentExecutor{
		fakeAgentExecutor: fakeAgentExecutor{responses: []AgentResult{
			{Text: "research context", SessionID: "research-session-1"},
			{Text: agentReportAnyJSON(narrativeContractTestPlan()), SessionID: "report-fork-1", Resumed: true},
			{Text: "병렬 섹션의 근거를 독자에게 직접 설명합니다.", SessionID: "report-fork-1", Resumed: true},
			{Text: `{"intro":"파트의 질문을 먼저 설명합니다.","transitions":[],"closing":"파트 판단을 정리합니다."}`, SessionID: "report-fork-1", Resumed: true},
			{Text: finalMarkdown, SessionID: "report-fork-1", Resumed: true},
		}},
		forkSessionID: "report-fork-1",
	}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Narrative fanout", "objective": "Explain parallel evidence as one report"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "Prepare report context."})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Fanout Reader Report", "report_mode": "long_form", "rigor_level": "balanced", "execution_strategy": reportExecutionStrategySectionFanout,
		"generation_guidance_profile": reportGenerationGuidanceProfileNarrativeContract,
	})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	assertNarrativeContractProductRequests(t, agent.requests[1:])
	payload := lastEventPayload(t, detail, "report.artifact.created")
	if payload["composition_strategy"] != reporting.LongFormCompositionNarrativeEdit || payload["assembly_strategy"] != "narrative_contract_final_edit" || payload["session_chain_kind"] != "section_fanout_report" {
		t.Fatalf("fanout candidate metadata mismatch: %#v", payload)
	}
	artifact, err := svc.GetRawArtifact(ctx, payload["artifact_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if string(artifact.Content) != finalMarkdown {
		t.Fatalf("fanout product path did not persist exact edited manuscript:\n%s", artifact.Content)
	}
}

func narrativeContractTestPlan() agentSectionalReportPlan {
	return agentSectionalReportPlan{
		Summary: "Explain one source-backed point without making the reader reconstruct the source.",
		WritingContract: &reporting.ReportWritingContract{
			CentralQuestion: "What should the reader understand?", ReaderTakeaway: "The source-backed mechanism and its limit.",
			ReadingPath: []string{"state the answer", "explain the mechanism", "close with the limit"}, MustKeep: []string{"the concrete mechanism", "the evidence limit"},
			VisualRole: "none needed", ToneAndShape: "direct, edited explanation",
		},
		Parts: []agentReportPart{{Title: "Core Part", Purpose: "Explain the answer.", Sections: []agentReportSection{{Title: "Core Section", Purpose: "Explain the mechanism and limit."}}}},
	}
}

func assertNarrativeContractProductRequests(t *testing.T, requests []AgentRequest) {
	t.Helper()
	if len(requests) != 4 {
		t.Fatalf("expected plan, section, Part editor, and final editor requests, got %d", len(requests))
	}
	if requests[0].ReportPlan == nil || !requests[0].ReportPlan.RequireWritingContract {
		t.Fatalf("candidate planner did not require writing contract: %#v", requests[0].ReportPlan)
	}
	if requests[2].PartAssembly == nil || !slices.Contains(requests[2].ExtraMCPTools, plasmamcp.ToolReportPartSectionRead) {
		t.Fatalf("candidate Part editor lost bound Section read: %#v", requests[2])
	}
	wantFinal := []string{
		plasmamcp.ToolReportLongFormEditStart, plasmamcp.ToolReportLongFormEditRead, plasmamcp.ToolReportLongFormEditPatch,
		plasmamcp.ToolReportLongFormEditSubmit, plasmamcp.ToolMermaidValidate,
	}
	if !slices.Equal(requests[3].ExtraMCPTools, wantFinal) || slices.Contains(requests[3].ExtraMCPTools, plasmamcp.ToolReportLongFormFinalize) {
		t.Fatalf("candidate final editor tool surface mismatch: %#v", requests[3].ExtraMCPTools)
	}
	if requests[3].LongFormFinalize == nil || requests[3].LongFormFinalize.CompositionStrategy != reporting.LongFormCompositionNarrativeEdit {
		t.Fatalf("candidate final binding mismatch: %#v", requests[3].LongFormFinalize)
	}
}

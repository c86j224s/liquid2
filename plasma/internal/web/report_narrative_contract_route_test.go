package web

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestFinalEditGatePromptAndToolsAreHumanizeAware(t *testing.T) {
	req := longFormReaderStyleGatePipelineRequest{title: "Report", missionID: "mis_gate", rigor: reportRigorProfiles["balanced"]}
	binding := reporting.FinalEditStageBinding{Stage: reporting.FinalEditStageGate, PostReportHumanize: reporting.FinalEditHumanizeDisabled}
	disabledPrompt := agentLongFormGatePromptForHumanize(reporting.FinalEditHumanizeDisabled)(req, binding, "rfe_gate", 1)
	if slices.Contains(reportFinalEditGateMCPToolsForHumanize(reporting.FinalEditHumanizeDisabled), plasmamcp.ToolReportLongFormStyleReviewRead) ||
		strings.Contains(disabledPrompt, "style_review") ||
		strings.Contains(disabledPrompt, "semantic_acceptance") ||
		!strings.Contains(disabledPrompt, "5. Submit with plasma.report.long_form.final_edit.submit and gate_findings") {
		t.Fatalf("disabled gate prompt/tools leaked semantic review:\n%s", disabledPrompt)
	}
	enabledPrompt := agentLongFormGatePromptForHumanize(reporting.FinalEditHumanizeEnabled)(req, binding, "rfe_gate", 1)
	if !slices.Contains(reportFinalEditGateMCPToolsForHumanize(reporting.FinalEditHumanizeEnabled), plasmamcp.ToolReportLongFormStyleReviewRead) ||
		!strings.Contains(enabledPrompt, "plasma.report.long_form.style_review.read") ||
		!strings.Contains(enabledPrompt, "semantic_acceptance") {
		t.Fatalf("enabled gate prompt/tools omitted semantic review:\n%s", enabledPrompt)
	}
}

func TestNarrativeContractSerialLongFormUsesProductEditorPath(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	writerMarkdown := "# Reader Report\n\n최종 작성자가 전체 도입과 결론을 정리했습니다.\n\n## Core Part\n\n근거를 소화해 직접 설명한 본문입니다.\n\n## Conclusion\n\n작성자가 핵심 판단을 연결합니다.\n"
	finalMarkdown := "# Reader Report\n\n독자가 바로 이해할 수 있는 도입입니다.\n\n## Core Part\n\n근거를 소화해 직접 설명한 본문입니다.\n\n## Conclusion\n\n핵심 판단을 정리합니다.\n"
	agent := &fakeForkingAgentExecutor{
		fakeAgentExecutor: fakeAgentExecutor{responses: []AgentResult{
			{Text: agentReportAnyJSON(narrativeContractTestPlan()), SessionID: "report-session-1"},
			{Text: "근거를 소화해 직접 설명한 본문입니다.", SessionID: "report-session-1"},
			{Text: `{"intro":"파트의 질문을 먼저 설명합니다.","transitions":[],"closing":"파트 판단을 정리합니다."}`, SessionID: "report-session-1"},
			{Text: "# Part 1. Core Part\n\n파트 편집자가 문단 연결을 확인했습니다.\n\n## 1.1 Core Section\n\n근거를 소화해 직접 설명한 본문입니다.\n\n파트 판단을 정리합니다.\n", SessionID: "part-editor-fork-serial"},
			{Text: writerMarkdown, SessionID: "writer-editor-fork-serial"},
			{Text: finalMarkdown, SessionID: "reader-editor-fork-serial"},
			{Text: finalEditGateSubmittedSentinel, SessionID: "gate-editor-fork-serial"},
		}},
		forkSessionIDs: []string{"part-editor-fork-serial", "writer-editor-fork-serial", "reader-editor-fork-serial", "gate-editor-fork-serial"},
	}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Narrative serial", "objective": "Explain evidence to a report-only reader"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Reader Report", "report_mode": "long_form", "rigor_level": "balanced", "generation_guidance_profile": reportprompt.ProfileNarrativeContract,
	})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	assertNarrativeContractProductRequests(t, agent.requests, true)
	assertNarrativeContractSeparateEditorForks(t, agent.requests, agent.forkSources, agent.forkSessions, "report-session-1", true, 0)
	planPayload := lastEventPayload(t, detail, "report.plan.created")
	if planPayload["final_edit_pipeline"] != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		t.Fatalf("serial plan did not activate v3 writer/reader/evidence pipeline: %#v", planPayload)
	}
	assertReaderStyleGateNoOpCanonicalPayload(t, detail, false)
	payload := lastEventPayload(t, detail, "report.artifact.created")
	if payload["composition_strategy"] != reporting.LongFormCompositionNarrativeEdit ||
		payload["assembly_strategy"] != "narrative_contract_final_edit" ||
		payload["final_edit_pipeline"] != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 ||
		payload["planned_final_artifact_id"] == "" ||
		payload["final_edit_gate_event_id"] == "" {
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

func TestSectionFanoutPlanActivationFlagsDecodeLifecyclePayload(t *testing.T) {
	event := app.LedgerEvent{Payload: mustJSON(map[string]any{
		"part_edit_enabled": true, "part_planning_enabled": true, "session_chain_kind": "section_fanout_report",
	})}
	partEdit, partPlanning, err := sectionFanoutPlanActivationFlags(event)
	if err != nil || !partEdit || !partPlanning {
		t.Fatalf("activation flags partEdit=%t partPlanning=%t err=%v", partEdit, partPlanning, err)
	}
	for _, tc := range []struct {
		name    string
		payload map[string]any
	}{
		{name: "planning without edit", payload: map[string]any{
			"part_edit_enabled": false, "part_planning_enabled": true, "session_chain_kind": "section_fanout_report",
		}},
		{name: "planning outside fanout", payload: map[string]any{
			"part_edit_enabled": true, "part_planning_enabled": true, "session_chain_kind": "same_session_report",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := sectionFanoutPlanActivationFlags(app.LedgerEvent{Payload: mustJSON(tc.payload)})
			if !errors.Is(err, app.ErrConflict) {
				t.Fatalf("error=%v, want conflict", err)
			}
		})
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
	writerMarkdown := "# Fanout Reader Report\n\n최종 작성자가 병렬 Part를 하나의 흐름으로 엽니다.\n\n## Core Part\n\n병렬 섹션의 근거를 독자에게 직접 설명합니다.\n\n## Conclusion\n\n작성자가 전체 판단을 연결합니다.\n"
	finalMarkdown := "# Fanout Reader Report\n\n병렬로 작성된 내용을 하나의 설명으로 엽니다.\n\n## Core Part\n\n병렬 섹션의 근거를 독자에게 직접 설명합니다.\n\n## Conclusion\n\n전체 판단을 연결합니다.\n"
	agent := &fakeForkingAgentExecutor{
		fakeAgentExecutor: fakeAgentExecutor{responses: []AgentResult{
			{Text: "research context", SessionID: "research-session-1"},
			{Text: agentReportAnyJSON(narrativeContractTestPlan()), SessionID: "report-fork-1", Resumed: true},
			{Text: "독자가 이해할 질문에서 근거와 한계로 자연스럽게 이동합니다.", SessionID: "part-owner-session", Resumed: true},
			{Text: "병렬 섹션의 근거를 독자에게 직접 설명합니다.", SessionID: "section-session", Resumed: true},
			{Text: `{"intro":"파트의 질문을 먼저 설명합니다.","transitions":[],"closing":"파트 판단을 정리합니다."}`, SessionID: "part-assembly-session", Resumed: true},
			{Text: "# Part 1. Core Part\n\n파트 최종 작성자가 병렬 섹션 연결을 확인했습니다.\n\n## 1.1 Core Section\n\n병렬 섹션의 근거를 독자에게 직접 설명합니다.\n\n파트 판단을 정리합니다.\n", SessionID: "part-owner-session", Resumed: true},
			{Text: writerMarkdown, SessionID: "writer-editor-session", Resumed: true},
			{Text: finalMarkdown, SessionID: "reader-editor-session", Resumed: true},
			{Text: finalEditGateSubmittedSentinel, SessionID: "gate-editor-session", Resumed: true},
		}},
		forkSessionIDs: []string{
			"report-fork-1",
			"part-owner-session",
			"section-session",
			"part-assembly-session",
			"writer-editor-session",
			"reader-editor-session",
			"gate-editor-session",
		},
	}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Narrative fanout", "objective": "Explain parallel evidence as one report"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "Prepare report context."})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Fanout Reader Report", "report_mode": "long_form", "rigor_level": "balanced", "execution_strategy": reportExecutionStrategySectionFanout,
		"generation_guidance_profile": reportprompt.ProfilePartConnectiveEconomyVoice,
	})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	assertNarrativeContractFanoutProductRequests(t, agent.requests[1:])
	planPayload := lastEventPayload(t, detail, "report.plan.created")
	if planPayload["final_edit_pipeline"] != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		t.Fatalf("fanout plan did not activate v3 writer/reader/evidence pipeline: %#v", planPayload)
	}
	if !slices.Equal(agent.forkSources, []string{"research-session-1", "report-fork-1", "part-owner-session", "part-owner-session", "report-fork-1", "report-fork-1", "report-fork-1"}) {
		t.Fatalf("fanout session ownership differs: %#v", agent.forkSources)
	}
	partPlanPayload := lastEventPayload(t, detail, reporting.PartPlanCreatedEventType)
	if partPlanPayload["agent_session_id"] != "part-owner-session" || partPlanPayload["brief"] != "독자가 이해할 질문에서 근거와 한계로 자연스럽게 이동합니다." {
		t.Fatalf("durable Part planning state differs: %#v", partPlanPayload)
	}
	progress, err := (&Server{service: svc}).loadSectionalReportProgress(ctx, missionID, payloadStringValue(t, partPlanPayload, "pending_event_id"))
	if err != nil {
		t.Fatal(err)
	}
	recovered, ok := progress.partPlans[0]
	if !progress.partPlanningEnabled || !ok || recovered.providerSessionID != "part-owner-session" || recovered.brief != "독자가 이해할 질문에서 근거와 한계로 자연스럽게 이동합니다." {
		t.Fatalf("Part planning state did not recover: %#v", progress)
	}
	assertReaderStyleGateNoOpCanonicalPayload(t, detail, false)
	payload := lastEventPayload(t, detail, "report.artifact.created")
	if payload["composition_strategy"] != reporting.LongFormCompositionNarrativeEdit ||
		payload["assembly_strategy"] != "narrative_contract_final_edit" ||
		payload["session_chain_kind"] != "section_fanout_report" ||
		payload["final_edit_pipeline"] != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 ||
		payload["planned_final_artifact_id"] == "" ||
		payload["final_edit_gate_event_id"] == "" {
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

func TestReaderStyleGateUsesHumanizeAsPreCanonicalStyleOnly(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	writerMarkdown := "# Styled Reader Report\n\n최종 작성자가 도입을 정리했습니다.\n\n## Core Part\n\n근거를 소화해 직접 설명한 본문입니다.\n"
	finalMarkdown := "# Styled Reader Report\n\n독자가 바로 이해할 수 있는 도입입니다.\n\n## Core Part\n\n근거를 소화해 직접 설명한 본문입니다.\n"
	agent := &fakeForkingAgentExecutor{
		fakeAgentExecutor: fakeAgentExecutor{responses: []AgentResult{
			{Text: agentReportAnyJSON(narrativeContractTestPlan()), SessionID: "report-session-style"},
			{Text: "근거를 소화해 직접 설명한 본문입니다.", SessionID: "report-session-style"},
			{Text: `{"intro":"파트 도입입니다.","transitions":[],"closing":"파트 마무리입니다."}`, SessionID: "report-session-style"},
			{Text: "# Part 1. Core Part\n\n근거를 소화해 직접 설명한 본문입니다.\n", SessionID: "part-editor-style"},
			{Text: writerMarkdown, SessionID: "writer-editor-style"},
			{Text: finalMarkdown, SessionID: "reader-editor-style"},
			{Text: finalEditStageSubmittedSentinel, SessionID: "style-editor-style"},
			{Text: finalEditStageSubmittedSentinel, SessionID: "style-semantic-editor-style"},
			{Text: finalEditGateSubmittedSentinel, SessionID: "gate-editor-style"},
		}},
		forkSessionIDs: []string{"part-editor-style", "writer-editor-style", "reader-editor-style", "style-editor-style", "style-semantic-editor-style", "gate-editor-style"},
	}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "style gate", "objective": "run style gate"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Styled Reader Report", "report_mode": "long_form", "rigor_level": "balanced",
		"generation_guidance_profile": reportprompt.ProfileNarrativeContract,
		"post_report_humanize":        "enabled",
	})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	time.Sleep(50 * time.Millisecond)
	detail = getJSON(t, server.URL+"/api/missions/"+missionID)
	if countEvents(detail, reporting.FinalEditStyleStartedEventType) != 1 ||
		countEvents(detail, reporting.FinalEditStyleSubmittedEventType) != 1 ||
		countEvents(detail, "report.humanize.pending") != 0 ||
		countEvents(detail, "report.artifact.exported") != 0 {
		t.Fatalf("enabled humanize must run as pre-canonical style only: %#v", detail["events"])
	}
	requests := agent.requests
	if len(requests) != 10 ||
		requests[5].FinalEditStage == nil ||
		requests[5].FinalEditStage.Stage != reporting.FinalEditStageWriter ||
		!slices.Equal(requests[5].ExtraMCPTools, reportFinalEditWriterMCPTools()) ||
		requests[6].FinalEditStage == nil ||
		requests[6].FinalEditStage.Stage != reporting.FinalEditStageReader ||
		!slices.Equal(requests[6].ExtraMCPTools, reportFinalEditReaderMCPTools()) ||
		requests[7].FinalEditStage == nil ||
		requests[7].FinalEditStage.Stage != reporting.FinalEditStageStyle ||
		!slices.Equal(requests[7].ExtraMCPTools, reportFinalEditStyleMCPTools()) ||
		requests[8].FinalEditStage == nil ||
		requests[8].FinalEditStage.Stage != reporting.FinalEditStageStyleSemanticValidation ||
		!slices.Equal(requests[8].ExtraMCPTools, reportFinalEditStyleSemanticValidationMCPTools()) ||
		!strings.Contains(requests[8].Prompt, "plasma.report.long_form.style_semantic_validation.read") ||
		requests[9].FinalEditStage == nil ||
		requests[9].FinalEditStage.Stage != reporting.FinalEditStageEvidenceGate ||
		!slices.Equal(requests[9].ExtraMCPTools, reportFinalEditEvidenceGateMCPTools()) ||
		strings.Contains(requests[9].Prompt, "semantic_acceptance") ||
		requests[6].FinalEditStage.SourceArtifactID != requests[5].FinalEditStage.EditedArtifactID ||
		requests[7].FinalEditStage.SourceArtifactID != requests[6].FinalEditStage.EditedArtifactID ||
		requests[8].FinalEditStage.SourceArtifactID != requests[7].FinalEditStage.SourceArtifactID ||
		requests[9].FinalEditStage.SourceArtifactID != requests[8].FinalEditStage.SourceArtifactID ||
		requests[5].FinalEditStage.ForkSourceAgentSessionID != "report-session-style" ||
		requests[6].FinalEditStage.ForkSourceAgentSessionID != "report-session-style" ||
		requests[7].FinalEditStage.ForkSourceAgentSessionID != "reader-editor-style" ||
		requests[8].FinalEditStage.ForkSourceAgentSessionID != "report-session-style" ||
		requests[9].FinalEditStage.ForkSourceAgentSessionID != "report-session-style" {
		t.Fatalf("style/gate request sequence mismatch: %#v", requests)
	}
	stylePayload := lastEventPayload(t, detail, reporting.FinalEditStyleSubmittedEventType)
	if stylePayload["changed"] != false || stylePayload["artifact_id"] != requests[7].FinalEditStage.SourceArtifactID {
		t.Fatalf("no-op style stage should leave gate source on reader artifact: %#v", stylePayload)
	}
	payload := lastEventPayload(t, detail, "report.artifact.created")
	artifact, err := svc.GetRawArtifact(ctx, payload["artifact_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if string(artifact.Content) != finalMarkdown {
		t.Fatalf("style pipeline did not preserve final manuscript: %q", string(artifact.Content))
	}
}

func TestReaderStyleGatePipelineAllowsStructurallySafeKoreanStylePolish(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	writerMarkdown := "# Styled Korean Report\n\n최종 작성자는 발행 위치와 증거 흐름을 한 원고로 연결합니다.\n\n독자는 문서 내용과 발행 위치에서 맥락을 확인하고, 이후 증거와 한계를 차례로 검토합니다.\n\n그 결과를 바탕으로 팀은 결론을 정리합니다.\n"
	readerMarkdown := "# Styled Korean Report\n\n독자는 문서 내용만이 아니라 발행 위치에서 먼저 맥락을 확인해야 하며, 이후 증거와 한계를 차례로 검토합니다.\n\n그 결과를 바탕으로 팀은 결론을 더 분명하게 정리합니다.\n"
	styleMarkdown := "# Styled Korean Report\n\n독자는 문서 내용만큼이나 발행 위치에서도 맥락을 먼저 확인하고, 이어서 증거와 한계를 차분히 검토합니다.\n\n그 판단을 바탕으로 팀은 결론을 한층 명확하게 정리합니다.\n"
	agent := &fakeForkingAgentExecutor{
		fakeAgentExecutor: fakeAgentExecutor{responses: []AgentResult{
			{Text: agentReportAnyJSON(narrativeContractTestPlan()), SessionID: "report-session-korean-style"},
			{Text: "근거를 소화해 직접 설명한 본문입니다.", SessionID: "report-session-korean-style"},
			{Text: `{"intro":"파트 도입입니다.","transitions":[],"closing":"파트 마무리입니다."}`, SessionID: "report-session-korean-style"},
			{Text: "# Part 1. Core Part\n\n근거를 소화해 직접 설명한 본문입니다.\n", SessionID: "part-editor-korean-style"},
			{Text: writerMarkdown, SessionID: "writer-editor-korean-style"},
			{Text: readerMarkdown, SessionID: "reader-editor-korean-style"},
			{Text: styleMarkdown, SessionID: "style-editor-korean-style"},
			{Text: finalEditStageSubmittedSentinel, SessionID: "style-semantic-editor-korean-style"},
			{Text: finalEditGateSubmittedSentinel, SessionID: "gate-editor-korean-style"},
		}},
		forkSessionIDs: []string{"part-editor-korean-style", "writer-editor-korean-style", "reader-editor-korean-style", "style-editor-korean-style", "style-semantic-editor-korean-style", "gate-editor-korean-style"},
	}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "korean style gate", "objective": "run structurally safe Korean style"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Styled Korean Report", "report_mode": "long_form", "rigor_level": "balanced",
		"generation_guidance_profile": reportprompt.ProfileNarrativeContract,
		"post_report_humanize":        "enabled",
	})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if countEvents(detail, reporting.FinalEditStyleSubmittedEventType) != 1 ||
		countEvents(detail, reporting.FinalEditStyleSemanticValidationSubmittedEventType) != 1 ||
		countEvents(detail, reporting.FinalEditEvidenceGateSubmittedEventType) != 1 ||
		countEvents(detail, "report.final.failed") != 0 {
		t.Fatalf("structurally safe style polish did not reach gate cleanly: %#v", detail["events"])
	}
	payload := lastEventPayload(t, detail, "report.artifact.created")
	artifact, err := svc.GetRawArtifact(ctx, payload["artifact_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if string(artifact.Content) != styleMarkdown {
		t.Fatalf("pipeline did not preserve structurally safe style result: %q", string(artifact.Content))
	}
}

func TestReaderStyleGateAdoptsDurableStageAfterAckMismatch(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	writerMarkdown := "# Reader Report\n\nWriter prepared the full manuscript.\n"
	finalMarkdown := "# Reader Report\n\nDurable reader result.\n"
	agent := &fakeForkingAgentExecutor{
		fakeAgentExecutor: fakeAgentExecutor{responses: []AgentResult{
			{Text: agentReportAnyJSON(narrativeContractTestPlan()), SessionID: "report-session-ack"},
			{Text: "Section body.", SessionID: "report-session-ack"},
			{Text: `{"intro":"Intro","transitions":[],"closing":"Close"}`, SessionID: "report-session-ack"},
			{Text: "# Part 1. Core Part\n\nEdited Part body.\n", SessionID: "part-editor-ack"},
			{Text: writerMarkdown, SessionID: "writer-editor-ack"},
			{Text: finalMarkdown, SessionID: "reader-editor-ack"},
			{Text: finalEditGateSubmittedSentinel, SessionID: "gate-editor-ack"},
		}},
		forkSessionIDs: []string{"part-editor-ack", "writer-editor-ack", "reader-editor-ack", "gate-editor-ack"},
	}
	executor := &finalEditStageAckMismatchExecutor{delegate: withReportPlanSubmissionFixture(svc, agent), stage: reporting.FinalEditStageReader}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: executor}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "ack mismatch"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Reader Report", "report_mode": "long_form", "generation_guidance_profile": reportprompt.ProfileNarrativeContract,
	})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if executor.calls != 1 ||
		countFinalEditStageRequests(agent.requests, reporting.FinalEditStageReader) != 1 ||
		countFinalEditStageRequests(agent.requests, reporting.FinalEditStageEvidenceGate) != 1 {
		t.Fatalf("reader ack mismatch should adopt durable submission without retry: calls=%d requests=%#v", executor.calls, agent.requests)
	}
	if countEvents(detail, reporting.FinalEditReaderSubmittedEventType) != 1 || countEvents(detail, "report.draft.failed") != 0 {
		t.Fatalf("durable reader submission was not recovered cleanly: %#v", detail["events"])
	}
	assertReaderStyleGateNoOpCanonicalPayload(t, detail, false)
	readerPayload := lastEventPayload(t, detail, reporting.FinalEditReaderSubmittedEventType)
	canonicalPayload := lastEventPayload(t, detail, "report.artifact.created")
	if canonicalPayload["artifact_id"] != readerPayload["artifact_id"] {
		t.Fatalf("canonical did not adopt durable reader artifact: reader=%#v canonical=%#v", readerPayload, canonicalPayload)
	}
	artifact, err := svc.GetRawArtifact(ctx, payloadStringValue(t, canonicalPayload, "artifact_id"))
	if err != nil {
		t.Fatal(err)
	}
	if string(artifact.Content) != finalMarkdown || artifact.SHA256 != payloadStringValue(t, canonicalPayload, "artifact_sha256") {
		t.Fatalf("canonical artifact did not preserve durable reader result: artifact=%#v payload=%#v", artifact, canonicalPayload)
	}
}

func TestReaderStyleGateRetriesOnceThenFailsWithoutCanonical(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	writerMarkdown := "# Reader Report\n\nWriter result before reader failure.\n"
	agent := &fakeForkingAgentExecutor{
		fakeAgentExecutor: fakeAgentExecutor{responses: []AgentResult{
			{Text: agentReportAnyJSON(narrativeContractTestPlan()), SessionID: "report-session-fail"},
			{Text: "Section body.", SessionID: "report-session-fail"},
			{Text: `{"intro":"Intro","transitions":[],"closing":"Close"}`, SessionID: "report-session-fail"},
			{Text: "# Part 1. Core Part\n\nEdited Part body.\n", SessionID: "part-editor-fail"},
			{Text: writerMarkdown, SessionID: "writer-editor-fail"},
		}},
		forkSessionIDs: []string{"part-editor-fail", "writer-editor-fail", "reader-editor-fail"},
	}
	executor := &finalEditStageFailureExecutor{delegate: withReportPlanSubmissionFixture(svc, agent), stage: reporting.FinalEditStageReader}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: executor}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "stage failure"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Reader Report", "report_mode": "long_form", "generation_guidance_profile": reportprompt.ProfileNarrativeContract,
	})
	detail := waitForEventType(t, server.URL, missionID, "report.draft.failed")
	time.Sleep(50 * time.Millisecond)
	detail = getJSON(t, server.URL+"/api/missions/"+missionID)
	if executor.calls != 2 || countFinalEditStageRequests(agent.requests, reporting.FinalEditStageReader) != 0 {
		t.Fatalf("reader technical failure should retry exactly once before terminal failure: calls=%d requests=%#v", executor.calls, agent.requests)
	}
	if countEvents(detail, "report.artifact.created") != 0 ||
		countEvents(detail, reporting.FinalEditReaderSubmittedEventType) != 0 ||
		countEvents(detail, "report.final.failed") != 1 ||
		countEvents(detail, "report.draft.failed") != 1 {
		t.Fatalf("second reader failure must block canonical completion: %#v", detail["events"])
	}
}

type finalEditStageAckMismatchExecutor struct {
	delegate AgentExecutor
	stage    string
	calls    int
}

func (executor *finalEditStageAckMismatchExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	result, err := executor.delegate.Run(ctx, req)
	if err == nil && req.FinalEditStage != nil && req.FinalEditStage.Stage == executor.stage {
		executor.calls++
		result.Text = "ACK_NOT_EXACT"
	}
	return result, err
}

func (executor *finalEditStageAckMismatchExecutor) ForkSession(ctx context.Context, sourceSessionID string) (AgentSessionForkResult, error) {
	forker, ok := executor.delegate.(AgentSessionForker)
	if !ok {
		return AgentSessionForkResult{}, errors.New("delegate cannot fork")
	}
	return forker.ForkSession(ctx, sourceSessionID)
}

func (executor *finalEditStageAckMismatchExecutor) CheckForkSession(ctx context.Context, sourceSessionID string) error {
	readiness, ok := executor.delegate.(AgentSessionForkReadiness)
	if !ok {
		return nil
	}
	return readiness.CheckForkSession(ctx, sourceSessionID)
}

type finalEditStageFailureExecutor struct {
	delegate AgentExecutor
	stage    string
	calls    int
}

func (executor *finalEditStageFailureExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	if req.FinalEditStage != nil && req.FinalEditStage.Stage == executor.stage {
		executor.calls++
		return AgentResult{SessionID: req.PreviousSessionID, Log: "stage failed"}, errors.New("final edit stage provider failed")
	}
	return executor.delegate.Run(ctx, req)
}

func (executor *finalEditStageFailureExecutor) ForkSession(ctx context.Context, sourceSessionID string) (AgentSessionForkResult, error) {
	forker, ok := executor.delegate.(AgentSessionForker)
	if !ok {
		return AgentSessionForkResult{}, errors.New("delegate cannot fork")
	}
	return forker.ForkSession(ctx, sourceSessionID)
}

func (executor *finalEditStageFailureExecutor) CheckForkSession(ctx context.Context, sourceSessionID string) error {
	readiness, ok := executor.delegate.(AgentSessionForkReadiness)
	if !ok {
		return nil
	}
	return readiness.CheckForkSession(ctx, sourceSessionID)
}

func payloadStringValue(t *testing.T, payload map[string]any, key string) string {
	t.Helper()
	value, ok := payload[key].(string)
	if !ok || value == "" {
		t.Fatalf("payload %s is missing: %#v", key, payload)
	}
	return value
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

func assertNarrativeContractFanoutProductRequests(t *testing.T, requests []AgentRequest) {
	t.Helper()
	if len(requests) != 9 {
		t.Fatalf("expected plan, requirements, Part plan, section, Part assembler, Part author, writer, reader, and gate requests, got %d", len(requests))
	}
	if requests[0].ReportPlan == nil || !requests[0].ReportPlan.RequireWritingContract {
		t.Fatalf("candidate planner did not require writing contract: %#v", requests[0].ReportPlan)
	}
	if requests[1].ReportRequirements == nil || !slices.Contains(requests[1].ExtraMCPTools, plasmamcp.ToolReportRequirementsSubmit) {
		t.Fatalf("candidate requirement mapper lost its bound submit tool: %#v", requests[1])
	}
	if requests[2].PreviousSessionID != "part-owner-session" || len(requests[2].ExtraMCPTools) != 0 || !requests[2].ReplaceMCPTools {
		t.Fatalf("Part planner did not use its isolated no-tool session: %#v", requests[2])
	}
	if requests[3].PreviousSessionID != "section-session" {
		t.Fatalf("Section did not fork from the Part owner: %#v", requests[3])
	}
	if requests[4].PartAssembly == nil || requests[4].PreviousSessionID != "part-assembly-session" || !slices.Contains(requests[4].ExtraMCPTools, plasmamcp.ToolReportPartSectionRead) {
		t.Fatalf("candidate Part assembler lost Part-owner lineage or bound Section read: %#v", requests[4])
	}
	wantPartEditor := []string{
		plasmamcp.ToolReportPartEditStart, plasmamcp.ToolReportPartEditRead,
		plasmamcp.ToolReportPartEditPatch, plasmamcp.ToolReportPartEditSubmit,
	}
	if requests[5].PartEdit == nil || requests[5].PreviousSessionID != "part-owner-session" || !slices.Equal(requests[5].ExtraMCPTools, wantPartEditor) {
		t.Fatalf("candidate Part author did not resume its planning session: %#v", requests[5])
	}
	if !strings.Contains(requests[5].Prompt, "You are the final author of this Part.") {
		t.Fatalf("planned fanout Part author did not use author prompt: %s", requests[5].Prompt)
	}
	if requests[6].PreviousSessionID != "writer-editor-session" ||
		!slices.Equal(requests[6].ExtraMCPTools, reportFinalEditWriterMCPTools()) ||
		requests[6].FinalEditStage == nil ||
		requests[6].FinalEditStage.Stage != reporting.FinalEditStageWriter ||
		requests[6].FinalEditStage.FinalEditPipeline != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 ||
		requests[6].LongFormFinalize != nil {
		t.Fatalf("candidate writer path differs: %#v", requests[6])
	}
	if slices.Contains(requests[6].ExtraMCPTools, plasmamcp.ToolResearchRead) || slices.Contains(requests[6].ExtraMCPTools, plasmamcp.ToolSourcesRead) {
		t.Fatalf("writer editor exposed research/source tools: %#v", requests[6].ExtraMCPTools)
	}
	if requests[7].PreviousSessionID != "reader-editor-session" ||
		!slices.Equal(requests[7].ExtraMCPTools, reportFinalEditReaderMCPTools()) ||
		requests[7].FinalEditStage == nil ||
		requests[7].FinalEditStage.Stage != reporting.FinalEditStageReader ||
		requests[7].FinalEditStage.FinalEditPipeline != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 ||
		requests[7].FinalEditStage.SourceArtifactID != requests[6].FinalEditStage.EditedArtifactID ||
		requests[7].LongFormFinalize != nil {
		t.Fatalf("candidate reader editor path differs: %#v", requests[7])
	}
	if slices.Contains(requests[7].ExtraMCPTools, plasmamcp.ToolResearchRead) || slices.Contains(requests[7].ExtraMCPTools, plasmamcp.ToolSourcesRead) {
		t.Fatalf("reader editor exposed research/source tools: %#v", requests[7].ExtraMCPTools)
	}
	if requests[8].PreviousSessionID != "gate-editor-session" ||
		!slices.Equal(requests[8].ExtraMCPTools, reportFinalEditEvidenceGateMCPTools()) ||
		slices.Contains(requests[8].ExtraMCPTools, plasmamcp.ToolReportLongFormStyleReviewRead) ||
		strings.Contains(requests[8].Prompt, "style_review") ||
		strings.Contains(requests[8].Prompt, "semantic_acceptance") ||
		requests[8].FinalEditStage == nil ||
		requests[8].FinalEditStage.Stage != reporting.FinalEditStageEvidenceGate ||
		requests[8].FinalEditStage.FinalEditPipeline != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 ||
		requests[8].FinalEditStage.SourceArtifactID != requests[7].FinalEditStage.EditedArtifactID ||
		requests[8].LongFormFinalize == nil {
		t.Fatalf("candidate evidence gate path differs: %#v", requests[8])
	}
}

func assertNarrativeContractProductRequests(t *testing.T, requests []AgentRequest, wantRequirements bool) {
	t.Helper()
	wantCount := 7
	if wantRequirements {
		wantCount = 8
	}
	if len(requests) != wantCount {
		t.Fatalf("expected %d narrative product requests, got %d", wantCount, len(requests))
	}
	if requests[0].ReportPlan == nil || !requests[0].ReportPlan.RequireWritingContract {
		t.Fatalf("candidate planner did not require writing contract: %#v", requests[0].ReportPlan)
	}
	sectionIndex, partIndex, partEditIndex, writerIndex, readerIndex, gateIndex := 1, 2, 3, 4, 5, 6
	if wantRequirements {
		if requests[1].ReportRequirements == nil || requests[1].ReportPlan != nil || !slices.Contains(requests[1].ExtraMCPTools, plasmamcp.ToolReportRequirementsSubmit) {
			t.Fatalf("candidate requirement mapper lost dedicated binding: %#v", requests[1])
		}
		sectionIndex, partIndex, partEditIndex, writerIndex, readerIndex, gateIndex = 2, 3, 4, 5, 6, 7
	}
	if requests[sectionIndex].ReportPlan != nil || requests[sectionIndex].ReportRequirements != nil || requests[sectionIndex].PartAssembly != nil || requests[sectionIndex].LongFormFinalize != nil || requests[sectionIndex].FinalEditStage != nil {
		t.Fatalf("candidate section request is out of order: %#v", requests[sectionIndex])
	}
	if requests[partIndex].PartAssembly == nil || !slices.Contains(requests[partIndex].ExtraMCPTools, plasmamcp.ToolReportPartSectionRead) {
		t.Fatalf("candidate Part editor lost bound Section read: %#v", requests[partIndex])
	}
	wantPartEdit := []string{
		plasmamcp.ToolReportPartEditStart, plasmamcp.ToolReportPartEditRead,
		plasmamcp.ToolReportPartEditPatch, plasmamcp.ToolReportPartEditSubmit,
	}
	if requests[partEditIndex].PartEdit == nil || !slices.Equal(requests[partEditIndex].ExtraMCPTools, wantPartEdit) {
		t.Fatalf("candidate Part edit surface mismatch: %#v", requests[partEditIndex])
	}
	if slices.Contains(requests[partEditIndex].ExtraMCPTools, plasmamcp.ToolResearchRead) || slices.Contains(requests[partEditIndex].ExtraMCPTools, plasmamcp.ToolSourcesRead) {
		t.Fatalf("candidate Part editor exposed research/source tools: %#v", requests[partEditIndex].ExtraMCPTools)
	}
	if !slices.Equal(requests[writerIndex].ExtraMCPTools, reportFinalEditWriterMCPTools()) ||
		slices.Contains(requests[writerIndex].ExtraMCPTools, plasmamcp.ToolResearchRead) ||
		slices.Contains(requests[writerIndex].ExtraMCPTools, plasmamcp.ToolSourcesRead) {
		t.Fatalf("candidate writer tool surface mismatch: %#v", requests[writerIndex].ExtraMCPTools)
	}
	if requests[writerIndex].FinalEditStage == nil ||
		requests[writerIndex].FinalEditStage.Stage != reporting.FinalEditStageWriter ||
		requests[writerIndex].FinalEditStage.FinalEditPipeline != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 ||
		requests[writerIndex].FinalEditStage.EditedArtifactID == requests[writerIndex].FinalEditStage.SourceArtifactID ||
		requests[writerIndex].LongFormFinalize != nil ||
		!strings.Contains(requests[writerIndex].Prompt, "plasma.report.long_form.final_write.start") ||
		!strings.Contains(requests[writerIndex].Prompt, "whole-report opening") {
		t.Fatalf("candidate writer binding/prompt mismatch: %#v", requests[writerIndex])
	}
	if !slices.Equal(requests[readerIndex].ExtraMCPTools, reportFinalEditReaderMCPTools()) ||
		slices.Contains(requests[readerIndex].ExtraMCPTools, plasmamcp.ToolResearchRead) ||
		slices.Contains(requests[readerIndex].ExtraMCPTools, plasmamcp.ToolSourcesRead) {
		t.Fatalf("candidate reader editor tool surface mismatch: %#v", requests[readerIndex].ExtraMCPTools)
	}
	if requests[readerIndex].FinalEditStage == nil ||
		requests[readerIndex].FinalEditStage.Stage != reporting.FinalEditStageReader ||
		requests[readerIndex].FinalEditStage.FinalEditPipeline != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 ||
		requests[readerIndex].FinalEditStage.SourceArtifactID != requests[writerIndex].FinalEditStage.EditedArtifactID ||
		requests[readerIndex].FinalEditStage.EditedArtifactID == requests[readerIndex].FinalEditStage.SourceArtifactID ||
		requests[readerIndex].LongFormFinalize != nil {
		t.Fatalf("candidate reader binding mismatch: %#v", requests[readerIndex])
	}
	if !slices.Equal(requests[gateIndex].ExtraMCPTools, reportFinalEditEvidenceGateMCPTools()) ||
		slices.Contains(requests[gateIndex].ExtraMCPTools, plasmamcp.ToolReportLongFormFinalize) {
		t.Fatalf("candidate evidence gate tool surface mismatch: %#v", requests[gateIndex].ExtraMCPTools)
	}
	if requests[gateIndex].FinalEditStage == nil ||
		requests[gateIndex].FinalEditStage.Stage != reporting.FinalEditStageEvidenceGate ||
		requests[gateIndex].FinalEditStage.FinalEditPipeline != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 ||
		requests[gateIndex].FinalEditStage.SourceArtifactID != requests[readerIndex].FinalEditStage.EditedArtifactID ||
		requests[gateIndex].LongFormFinalize == nil ||
		requests[gateIndex].LongFormFinalize.CompositionStrategy != reporting.LongFormCompositionNarrativeEdit {
		t.Fatalf("candidate evidence gate/final binding mismatch: %#v", requests[gateIndex])
	}
	if !strings.Contains(requests[gateIndex].Prompt, "mission_source_grounded") || !strings.Contains(requests[gateIndex].Prompt, "unverified_external_fact") {
		t.Fatalf("candidate evidence gate prompt lost exact classifications:\n%s", requests[gateIndex].Prompt)
	}
	if strings.Contains(requests[gateIndex].Prompt, "Global requirement preservation checks") {
		t.Fatalf("candidate evidence gate prompt retained requirement preservation checks:\n%s", requests[gateIndex].Prompt)
	}
}

func assertNarrativeContractSeparateEditorForks(t *testing.T, requests []AgentRequest, forkSources []string, forkSessions []string, planSessionID string, wantRequirements bool, firstPlanForks int) {
	t.Helper()
	_, partIndex, partEditIndex, writerIndex, readerIndex, gateIndex := narrativeContractRequestIndexes(wantRequirements)
	partSession := requests[partIndex].PreviousSessionID
	partEditSession := requests[partEditIndex].PreviousSessionID
	writerSession := requests[writerIndex].PreviousSessionID
	readerSession := requests[readerIndex].PreviousSessionID
	gateSession := requests[gateIndex].PreviousSessionID
	if partEditSession == "" || writerSession == "" || readerSession == "" || gateSession == "" ||
		partEditSession == writerSession || partEditSession == readerSession || partEditSession == gateSession ||
		writerSession == readerSession || writerSession == gateSession || readerSession == gateSession ||
		partEditSession == partSession || writerSession == partSession || readerSession == partSession || gateSession == partSession {
		t.Fatalf("editor sessions must be distinct forks from assembler/final paths: part=%q part_edit=%q writer=%q reader=%q gate=%q", partSession, partEditSession, writerSession, readerSession, gateSession)
	}
	if requests[partEditIndex].PartEdit.ReportPlanSessionID != planSessionID ||
		requests[partEditIndex].PartEdit.ForkSourceAgentSessionID != planSessionID ||
		requests[writerIndex].FinalEditStage.ReportPlanSessionID != planSessionID ||
		requests[writerIndex].FinalEditStage.ForkSourceAgentSessionID != planSessionID ||
		requests[readerIndex].FinalEditStage.ReportPlanSessionID != planSessionID ||
		requests[readerIndex].FinalEditStage.ForkSourceAgentSessionID != planSessionID ||
		requests[gateIndex].FinalEditStage.ReportPlanSessionID != planSessionID ||
		requests[gateIndex].FinalEditStage.ForkSourceAgentSessionID != planSessionID ||
		requests[gateIndex].LongFormFinalize.ReportPlanSessionID != planSessionID ||
		requests[gateIndex].LongFormFinalize.ForkSourceAgentSessionID != planSessionID {
		t.Fatalf("editor bindings must be forked from report plan session: part_edit=%#v writer=%#v reader=%#v gate=%#v final=%#v", requests[partEditIndex].PartEdit, requests[writerIndex].FinalEditStage, requests[readerIndex].FinalEditStage, requests[gateIndex].FinalEditStage, requests[gateIndex].LongFormFinalize)
	}
	if requests[readerIndex].FinalEditStage.PreviousProviderSessionID == writerSession ||
		requests[readerIndex].PreviousSessionID == writerSession {
		t.Fatalf("reader must consume writer artifact without inheriting writer session: writer=%#v reader=%#v", requests[writerIndex].FinalEditStage, requests[readerIndex].FinalEditStage)
	}
	if requests[readerIndex].FinalEditStage.SourceArtifactID != requests[writerIndex].FinalEditStage.EditedArtifactID ||
		requests[gateIndex].FinalEditStage.SourceArtifactID != requests[readerIndex].FinalEditStage.EditedArtifactID {
		t.Fatalf("final edit artifact chain mismatch: writer=%#v reader=%#v gate=%#v", requests[writerIndex].FinalEditStage, requests[readerIndex].FinalEditStage, requests[gateIndex].FinalEditStage)
	}
	if len(forkSources) < firstPlanForks+4 || len(forkSessions) < firstPlanForks+4 {
		t.Fatalf("missing recorded editor forks: sources=%#v sessions=%#v", forkSources, forkSessions)
	}
	partEditFork := firstPlanForks
	writerFork := firstPlanForks + 1
	readerFork := firstPlanForks + 2
	gateFork := firstPlanForks + 3
	if forkSources[partEditFork] != planSessionID || forkSources[writerFork] != planSessionID || forkSources[readerFork] != planSessionID || forkSources[gateFork] != planSessionID ||
		forkSessions[partEditFork] != partEditSession || forkSessions[writerFork] != writerSession || forkSessions[readerFork] != readerSession || forkSessions[gateFork] != gateSession {
		t.Fatalf("editor fork provenance mismatch: sources=%#v sessions=%#v part_edit=%q writer=%q reader=%q gate=%q", forkSources, forkSessions, partEditSession, writerSession, readerSession, gateSession)
	}
}

func narrativeContractRequestIndexes(wantRequirements bool) (sectionIndex int, partIndex int, partEditIndex int, writerIndex int, readerIndex int, gateIndex int) {
	sectionIndex, partIndex, partEditIndex, writerIndex, readerIndex, gateIndex = 1, 2, 3, 4, 5, 6
	if wantRequirements {
		sectionIndex, partIndex, partEditIndex, writerIndex, readerIndex, gateIndex = 2, 3, 4, 5, 6, 7
	}
	return sectionIndex, partIndex, partEditIndex, writerIndex, readerIndex, gateIndex
}

func countFinalEditStageRequests(requests []AgentRequest, stage string) int {
	count := 0
	for _, req := range requests {
		if req.FinalEditStage != nil && req.FinalEditStage.Stage == stage {
			count++
		}
	}
	return count
}

func assertReaderStyleGateNoOpCanonicalPayload(t *testing.T, detail map[string]any, wantStyle bool) {
	t.Helper()
	if countEvents(detail, reporting.FinalEditAssemblyCreatedEventType) != 1 ||
		countEvents(detail, reporting.FinalEditWriterStartedEventType) != 1 ||
		countEvents(detail, reporting.FinalEditWriterSubmittedEventType) != 1 ||
		countEvents(detail, reporting.FinalEditReaderStartedEventType) != 1 ||
		countEvents(detail, reporting.FinalEditReaderSubmittedEventType) != 1 ||
		countEvents(detail, reporting.FinalEditEvidenceGateStartedEventType) != 1 ||
		countEvents(detail, reporting.FinalEditEvidenceGateSubmittedEventType) != 1 ||
		countEvents(detail, "report.artifact.created") != 1 {
		t.Fatalf("v3 final edit durable event counts differ: %#v", detail["events"])
	}
	wantStyleCount := 0
	if wantStyle {
		wantStyleCount = 1
	}
	if countEvents(detail, reporting.FinalEditStyleStartedEventType) != wantStyleCount ||
		countEvents(detail, reporting.FinalEditStyleSubmittedEventType) != wantStyleCount ||
		countEvents(detail, reporting.FinalEditStyleSemanticValidationStartedEventType) != wantStyleCount ||
		countEvents(detail, reporting.FinalEditStyleSemanticValidationSubmittedEventType) != wantStyleCount {
		t.Fatalf("style durable event counts differ: %#v", detail["events"])
	}
	payload := lastEventPayload(t, detail, "report.artifact.created")
	artifactID := payloadStringValue(t, payload, "artifact_id")
	plannedID := payloadStringValue(t, payload, "planned_final_artifact_id")
	if payload["final_edit_pipeline"] != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 ||
		payload["final_edit_gate_changed"] != false ||
		payloadStringValue(t, payload, "final_edit_gate_event_id") == "" ||
		payloadStringValue(t, payload, "artifact_sha256") == "" ||
		artifactID == plannedID {
		t.Fatalf("reader/style/gate no-op canonical payload mismatch: %#v", payload)
	}
}

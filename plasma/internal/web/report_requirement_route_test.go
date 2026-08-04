package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestLongFormProductPathMapsRequirementToOnlyOwnedSection(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: agentReportAnyJSON(agentSectionalReportPlan{Summary: "Plan", Parts: []agentReportPart{{Title: "Part", Sections: []agentReportSection{{Title: "Comparison"}, {Title: "Limits"}}}}}), SessionID: "report-session"},
		{Text: "Comparison section.", SessionID: "report-session"},
		{Text: "Limits section.", SessionID: "report-session"},
		{Text: `{"intro":"Intro","transitions":[],"closing":"Close"}`, SessionID: "report-session"},
		{Text: `{"front_matter":"# Report","closing":"## Close"}`, SessionID: "report-session"},
	}}
	fixture := &reportRequirementFixtureExecutor{base: &reportPlanFixtureExecutor{delegate: agent, service: service}, service: service}
	fixture.requirementMapFixture = func(req AgentRequest) reporting.ReportRequirementMap {
		return reporting.ReportRequirementMap{
			ReviewedEventIDs: []string{req.ReportRequirements.PendingEventID},
			Requirements: []reporting.ReportRequirement{{
				RequirementID: "req_comparison_table", Instruction: "include a comparison table",
				SourceEventIDs: []string{req.ReportRequirements.PendingEventID},
				Owner:          &reporting.ReportRequirementOwner{PartIndex: 1, SectionIndex: 1},
			}},
		}
	}
	server := httptest.NewServer(NewServer(service, Options{AgentExecutor: fixture}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Requirement route"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Report", "report_mode": reportModeLongForm, "direction_hint": "include a comparison table",
		"generation_guidance_profile": reportprompt.ProfileVisualPlan,
	})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if countEvents(detail, reporting.ReportRequirementsMappedEventType) != 1 {
		t.Fatalf("requirement map was not durably recorded: %#v", detail["events"])
	}
	if len(agent.requests) != 6 {
		t.Fatalf("request count = %d, want plan+requirements+two sections+part+final", len(agent.requests))
	}
	if strings.Contains(agent.requests[0].Prompt, "include a comparison table") {
		t.Fatal("fixed-outline planner received the detailed output requirement")
	}
	if !strings.Contains(agent.requests[1].Prompt, "include a comparison table") || !requestHasMCPTool(agent.requests[1], "plasma.report.requirements.submit") || agent.requests[1].ReportRequirements == nil || agent.requests[1].ReportPlan != nil {
		t.Fatal("requirement mapper did not receive the current output requirement")
	}
	if !strings.Contains(agent.requests[2].Prompt, "req_comparison_table") || !strings.Contains(agent.requests[2].Prompt, "include a comparison table") {
		t.Fatal("owned Section did not receive its mapped requirement")
	}
	if strings.Contains(agent.requests[3].Prompt, "req_comparison_table") || strings.Contains(agent.requests[3].Prompt, "include a comparison table") {
		t.Fatal("mapped requirement leaked into an unowned Section")
	}
}

func TestLongFormRequirementReviewRunsWithoutDirectionHint(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	writerMarkdown := "# Report\n\nWriter prepared the full manuscript.\n"
	agent := &fakeForkingAgentExecutor{
		fakeAgentExecutor: fakeAgentExecutor{responses: []AgentResult{
			{Text: agentReportAnyJSON(agentSectionalReportPlan{Summary: "Plan", WritingContract: &reporting.ReportWritingContract{
				CentralQuestion: "What matters?", ReaderTakeaway: "The useful distinction and its limit.",
				ReadingPath: []string{"state the answer", "show the distinction", "close with the limit"}, MustKeep: []string{"the useful distinction"},
				VisualRole: "none", ToneAndShape: "direct",
			}, Parts: []agentReportPart{{Title: "Part", Sections: []agentReportSection{{Title: "Overview"}, {Title: "Limits"}}}}}), SessionID: "report-session"},
			{Text: "Overview section.", SessionID: "report-session"},
			{Text: "Limits section.", SessionID: "report-session"},
			{Text: `{"intro":"Intro","transitions":[],"closing":"Close"}`, SessionID: "report-session"},
			{Text: "# Part 1. Part\n\nReviewed Part body.\n", SessionID: "part-edit-session"},
			{Text: writerMarkdown, SessionID: "writer-edit-session"},
			{Text: finalEditStageSubmittedSentinel, SessionID: "reader-edit-session"},
			{Text: finalEditGateSubmittedSentinel, SessionID: "gate-edit-session"},
		}},
		forkSessionIDs: []string{"part-edit-session", "writer-edit-session", "reader-edit-session", "gate-edit-session"},
	}
	fixture := &reportRequirementFixtureExecutor{base: &reportPlanFixtureExecutor{delegate: agent, service: service}, service: service}
	fixture.requirementMapFixture = func(req AgentRequest) reporting.ReportRequirementMap {
		if !strings.Contains(req.Prompt, "evt_prior_user") {
			t.Fatalf("requirement mapper did not receive the prior user event id: %s", req.Prompt)
		}
		return reporting.ReportRequirementMap{
			ReviewedEventIDs: []string{"evt_prior_user", req.ReportRequirements.PendingEventID},
			Requirements: []reporting.ReportRequirement{{
				RequirementID:  "req_prior_limits",
				Instruction:    "include a constraints checklist in the Limits section",
				SourceEventIDs: []string{"evt_prior_user"},
				Owner:          &reporting.ReportRequirementOwner{PartIndex: 1, SectionIndex: 2},
			}},
		}
	}
	server := httptest.NewServer(NewServer(service, Options{AgentExecutor: fixture}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Requirement no direction"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_prior_user", MissionID: missionID, EventType: "turn.user",
		Producer: app.Producer{Type: "user", ID: "test"}, Payload: json.RawMessage(`{"text":"include a constraints checklist in the Limits section"}`),
	}); err != nil {
		t.Fatal(err)
	}
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Report", "report_mode": reportModeLongForm,
	})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if countEvents(detail, reporting.ReportRequirementsMappedEventType) != 1 {
		t.Fatalf("requirement map was skipped without direction hint: %#v", detail["events"])
	}
	if len(agent.requests) != 9 {
		t.Fatalf("request count = %d, want plan+requirements+two sections+part+part_edit+writer+reader+gate", len(agent.requests))
	}
	if agent.requests[1].ReportRequirements == nil || agent.requests[1].ReportPlan != nil || !requestHasMCPTool(agent.requests[1], "plasma.report.requirements.submit") {
		t.Fatalf("empty-direction mapper did not use the dedicated requirement binding: %#v", agent.requests[1])
	}
	requirementPayload := lastEventPayload(t, detail, reporting.ReportRequirementsMappedEventType)
	requirementMap := nestedMap(t, requirementPayload, "requirement_map")
	reviewed, _ := requirementMap["reviewed_event_ids"].([]any)
	if len(reviewed) != 2 || reviewed[0] != "evt_prior_user" {
		t.Fatalf("empty-direction requirement mapper did not record actual reviewed events: %#v", requirementMap)
	}
	requirements, _ := requirementMap["requirements"].([]any)
	if len(requirements) != 1 {
		t.Fatalf("empty-direction mapper did not persist one mapped prior requirement: %#v", requirementMap)
	}
	if strings.Contains(agent.requests[2].Prompt, "req_prior_limits") || strings.Contains(agent.requests[2].Prompt, "constraints checklist") {
		t.Fatal("prior requirement leaked into the unowned Section")
	}
	if !strings.Contains(agent.requests[3].Prompt, "req_prior_limits") || !strings.Contains(agent.requests[3].Prompt, "constraints checklist") {
		t.Fatal("owned Section did not receive the prior-user requirement")
	}
	partEdit := agent.requests[5]
	if partEdit.PartEdit == nil ||
		!requestHasMCPTool(partEdit, "plasma.report.part_edit.start") ||
		!requestHasMCPTool(partEdit, "plasma.report.part_edit.read") ||
		!requestHasMCPTool(partEdit, "plasma.report.part_edit.patch") ||
		!requestHasMCPTool(partEdit, "plasma.report.part_edit.submit") {
		t.Fatalf("Part edit did not use the dedicated binding: %#v", partEdit)
	}
	assertV2FinalEditRequestChain(t, agent.requests, 6, 7, 8, "report-session", true, false)
	if !slices.Equal(agent.forkSources, []string{"report-session", "report-session", "report-session", "report-session"}) {
		t.Fatalf("v2 no-direction final edit forks must be plan siblings after Part edit: %#v", agent.forkSources)
	}
}

func TestLongFormSectionFanoutMapsRequirementToOnlyOwnedSection(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	agent := &fanoutRequirementAgent{
		plan: agentSectionalReportPlan{
			Summary: "Plan",
			WritingContract: &reporting.ReportWritingContract{
				CentralQuestion: "What matters?", ReaderTakeaway: "The useful distinction and its limit.",
				ReadingPath: []string{"state the answer", "show the distinction", "close with the limit"}, MustKeep: []string{"the useful distinction"},
				VisualRole: "none", ToneAndShape: "direct",
			},
			Parts: []agentReportPart{{
				Title: "Part",
				Sections: []agentReportSection{
					{Title: "Overview", Purpose: "Explain the baseline."},
					{Title: "Constraints", Purpose: "Explain the limits."},
				},
			}},
		},
		finalMarkdown: "# Fanout Report\n\nEdited report.\n",
	}
	fixture := &reportRequirementFixtureExecutor{base: &reportPlanFixtureExecutor{delegate: agent, service: service}, service: service}
	fixture.requirementMapFixture = func(req AgentRequest) reporting.ReportRequirementMap {
		if !strings.Contains(req.Prompt, "include calibrated risk register") {
			t.Fatalf("fanout mapper did not receive direction: %s", req.Prompt)
		}
		return reporting.ReportRequirementMap{
			ReviewedEventIDs: []string{req.ReportRequirements.PendingEventID},
			Requirements: []reporting.ReportRequirement{
				{
					RequirementID:  "req_fanout_risk_register",
					Instruction:    "include calibrated risk register",
					SourceEventIDs: []string{req.ReportRequirements.PendingEventID},
					Owner:          &reporting.ReportRequirementOwner{PartIndex: 1, SectionIndex: 2},
				},
				{
					RequirementID:  "req_fanout_unmapped_appendix",
					Instruction:    "add unsupported appendix",
					SourceEventIDs: []string{req.ReportRequirements.PendingEventID},
					UnmappedReason: "no matching Section in the fixed outline",
				},
			},
		}
	}
	server := httptest.NewServer(NewServer(service, Options{AgentExecutor: fixture}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Fanout requirement route"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Fanout Report", "report_mode": reportModeLongForm, "execution_strategy": reportExecutionStrategySectionFanout,
		"direction_hint": "include calibrated risk register", "generation_guidance_profile": reportprompt.ProfileNarrativeContract,
	})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if countEvents(detail, reporting.ReportRequirementsMappedEventType) != 1 {
		t.Fatalf("fanout requirement map was not durably recorded once: %#v", detail["events"])
	}
	requirementPayload := lastEventPayload(t, detail, reporting.ReportRequirementsMappedEventType)
	requirementMap := nestedMap(t, requirementPayload, "requirement_map")
	requirements, _ := requirementMap["requirements"].([]any)
	if len(requirements) != 2 || !strings.Contains(fmt.Sprint(requirementMap), "req_fanout_unmapped_appendix") {
		t.Fatalf("fanout unmapped requirement was not durably retained: %#v", requirementMap)
	}
	requests := agent.snapshotRequests()
	if len(requests) != 9 {
		t.Fatalf("request count = %d, want plan+requirements+two sections+part+part_edit+writer+reader+gate", len(requests))
	}
	if strings.Contains(requests[0].Prompt, "include calibrated risk register") {
		t.Fatal("fanout fixed-outline planner received the detailed output requirement")
	}
	if requests[1].ReportRequirements == nil || requests[1].ReportPlan != nil || !requestHasMCPTool(requests[1], "plasma.report.requirements.submit") {
		t.Fatalf("fanout mapper did not use the dedicated requirement binding: %#v", requests[1])
	}
	sectionOne := requestByUserText(t, requests, "draft section 1.1")
	sectionTwo := requestByUserText(t, requests, "draft section 1.2")
	if strings.Contains(sectionOne.Prompt, "req_fanout_risk_register") || strings.Contains(sectionOne.Prompt, "calibrated risk register") {
		t.Fatal("fanout requirement leaked into the unowned Section")
	}
	if !strings.Contains(sectionTwo.Prompt, "req_fanout_risk_register") || !strings.Contains(sectionTwo.Prompt, "calibrated risk register") {
		t.Fatal("fanout owned Section did not receive its mapped requirement")
	}
	partEdit := requests[len(requests)-4]
	if partEdit.PartEdit == nil ||
		!requestHasMCPTool(partEdit, "plasma.report.part_edit.start") ||
		!requestHasMCPTool(partEdit, "plasma.report.part_edit.read") ||
		!requestHasMCPTool(partEdit, "plasma.report.part_edit.patch") ||
		!requestHasMCPTool(partEdit, "plasma.report.part_edit.submit") {
		t.Fatalf("fanout Part edit did not use the dedicated Part edit binding: %#v", partEdit)
	}
	if !strings.Contains(partEdit.Prompt, "req_fanout_risk_register") || strings.Contains(partEdit.Prompt, "req_fanout_unmapped_appendix") || requestHasMCPTool(partEdit, "plasma.research.read") || requestHasMCPTool(partEdit, "plasma.sources.read") {
		t.Fatalf("fanout Part edit requirement/tool surface mismatch:\n%s\n%#v", partEdit.Prompt, partEdit.ExtraMCPTools)
	}
	assertV2FinalEditRequestChain(t, requests, len(requests)-3, len(requests)-2, len(requests)-1, "report-session", false, false)
	final := requests[len(requests)-1]
	if final.FinalEditStage == nil || final.FinalEditStage.Stage != reporting.FinalEditStageEvidenceGate || final.LongFormFinalize == nil {
		t.Fatalf("fanout evidence gate did not receive the v3 evidence binding:\n%s", final.Prompt)
	}
	if strings.Contains(final.Prompt, "Global requirement preservation checks") || strings.Contains(final.Prompt, "req_fanout_risk_register") || strings.Contains(final.Prompt, "calibrated risk register") {
		t.Fatalf("fanout evidence gate retained requirement preservation checks:\n%s", final.Prompt)
	}
	if strings.Contains(final.Prompt, "req_fanout_unmapped_appendix") || strings.Contains(final.Prompt, "add unsupported appendix") {
		t.Fatalf("fanout gate editor received unmapped requirement preservation checks:\n%s", final.Prompt)
	}
}

func assertV2FinalEditRequestChain(t *testing.T, requests []AgentRequest, writerIndex int, readerIndex int, gateIndex int, planSessionID string, writerChanged bool, readerChanged bool) {
	t.Helper()
	writer := requests[writerIndex]
	reader := requests[readerIndex]
	gate := requests[gateIndex]
	if writer.FinalEditStage == nil ||
		writer.FinalEditStage.Stage != reporting.FinalEditStageWriter ||
		writer.FinalEditStage.FinalEditPipeline != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 ||
		writer.LongFormFinalize != nil ||
		!slices.Equal(writer.ExtraMCPTools, reportFinalEditWriterMCPTools()) {
		t.Fatalf("writer stage request mismatch: %#v", writer)
	}
	if reader.FinalEditStage == nil ||
		reader.FinalEditStage.Stage != reporting.FinalEditStageReader ||
		reader.FinalEditStage.FinalEditPipeline != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 ||
		reader.LongFormFinalize != nil ||
		!slices.Equal(reader.ExtraMCPTools, reportFinalEditReaderMCPTools()) {
		t.Fatalf("reader stage request mismatch: %#v", reader)
	}
	if gate.FinalEditStage == nil ||
		gate.FinalEditStage.Stage != reporting.FinalEditStageEvidenceGate ||
		gate.FinalEditStage.FinalEditPipeline != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 ||
		gate.LongFormFinalize == nil ||
		!slices.Equal(gate.ExtraMCPTools, reportFinalEditEvidenceGateMCPTools()) {
		t.Fatalf("evidence gate stage request mismatch: %#v", gate)
	}
	if writer.FinalEditStage.ForkSourceAgentSessionID != planSessionID ||
		reader.FinalEditStage.ForkSourceAgentSessionID != planSessionID ||
		gate.FinalEditStage.ForkSourceAgentSessionID != planSessionID ||
		writer.FinalEditStage.PreviousProviderSessionID != planSessionID ||
		reader.FinalEditStage.PreviousProviderSessionID != planSessionID ||
		gate.FinalEditStage.PreviousProviderSessionID != planSessionID {
		t.Fatalf("v3 final edit stages must be sibling forks from plan session: writer=%#v reader=%#v gate=%#v", writer.FinalEditStage, reader.FinalEditStage, gate.FinalEditStage)
	}
	writerArtifactID := writer.FinalEditStage.SourceArtifactID
	if writerChanged {
		writerArtifactID = writer.FinalEditStage.EditedArtifactID
	}
	readerArtifactID := reader.FinalEditStage.SourceArtifactID
	if readerChanged {
		readerArtifactID = reader.FinalEditStage.EditedArtifactID
	}
	if reader.PreviousSessionID == writer.PreviousSessionID ||
		reader.FinalEditStage.SourceArtifactID != writerArtifactID ||
		gate.FinalEditStage.SourceArtifactID != readerArtifactID {
		t.Fatalf("v3 final edit artifact/session chain mismatch: writer=%#v reader=%#v gate=%#v", writer.FinalEditStage, reader.FinalEditStage, gate.FinalEditStage)
	}
	if requestHasMCPTool(writer, "plasma.research.read") ||
		requestHasMCPTool(writer, "plasma.sources.read") ||
		requestHasMCPTool(reader, "plasma.research.read") ||
		requestHasMCPTool(reader, "plasma.sources.read") {
		t.Fatalf("writer/reader stage leaked research or source tools: writer=%#v reader=%#v", writer.ExtraMCPTools, reader.ExtraMCPTools)
	}
}

type reportRequirementFixtureExecutor struct {
	base                  *reportPlanFixtureExecutor
	service               *app.Service
	requirementMapFixture func(AgentRequest) reporting.ReportRequirementMap
}

func (executor *reportRequirementFixtureExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	if !requestHasMCPTool(req, "plasma.report.requirements.submit") {
		return executor.base.Run(ctx, req)
	}
	result, err := executor.base.delegate.Run(ctx, req)
	if err != nil {
		return result, err
	}
	if req.ReportRequirements == nil || executor.requirementMapFixture == nil {
		return result, fmt.Errorf("report requirement fixture is missing binding")
	}
	planEvent, plan, err := executor.requirementPlan(ctx, req)
	if err != nil {
		return result, err
	}
	requirementMap, err := reporting.NormalizeReportRequirementMap(executor.requirementMapFixture(req), plan)
	if err != nil {
		return result, err
	}
	hash, encoded, err := reporting.ReportRequirementMapHash(requirementMap)
	if err != nil {
		return result, err
	}
	_, err = executor.service.SubmitReportRequirementMap(ctx, app.ReportRequirementMapSubmissionRequest{
		EventID: fmt.Sprintf("evt_requirement_fixture_%d", executor.base.sequence.Add(1)), MissionID: req.MissionID,
		PendingEventID: req.ReportRequirements.PendingEventID, PlanEventID: planEvent.EventID, ToolSessionID: req.ReportRequirements.ToolSessionID,
		PreviousProviderSessionID: req.ReportRequirements.PreviousProviderSessionID, AgentExecutor: req.ReportRequirements.AgentExecutor,
		AgentModel: req.ReportRequirements.AgentModel, AgentReasoningEffort: req.ReportRequirements.AgentReasoningEffort,
		IdempotencyKey: req.ReportRequirements.IdempotencyKey, ArgumentsHash: "fixture-requirements", RequirementMapHash: hash,
		RequirementMap: encoded, ReviewedEventIDs: requirementMap.ReviewedEventIDs, Attempt: 1,
		ToolProducer: req.ReportRequirements.Producer,
	})
	if err != nil {
		return result, err
	}
	result.Text = reporting.ReportRequirementsMappedSentinel
	return result, nil
}

func (executor *reportRequirementFixtureExecutor) ForkSession(ctx context.Context, sessionID string) (AgentSessionForkResult, error) {
	forker, ok := executor.base.delegate.(AgentSessionForker)
	if !ok {
		return AgentSessionForkResult{}, fmt.Errorf("report requirement fixture delegate cannot fork")
	}
	return forker.ForkSession(ctx, sessionID)
}

func (executor *reportRequirementFixtureExecutor) CheckForkSession(ctx context.Context, sessionID string) error {
	readiness, ok := executor.base.delegate.(AgentSessionForkReadiness)
	if !ok {
		return nil
	}
	return readiness.CheckForkSession(ctx, sessionID)
}

func (executor *reportRequirementFixtureExecutor) requirementPlan(ctx context.Context, req AgentRequest) (app.LedgerEvent, reporting.SectionalReportPlan, error) {
	events, err := executor.service.ListEvents(ctx, req.MissionID)
	if err != nil {
		return app.LedgerEvent{}, reporting.SectionalReportPlan{}, err
	}
	for index := len(events) - 1; index >= 0; index-- {
		event := events[index]
		if event.EventType != "report.plan.created" {
			continue
		}
		var payload struct {
			PendingEventID string                        `json:"pending_event_id"`
			Plan           reporting.SectionalReportPlan `json:"plan"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil || payload.PendingEventID != req.ReportRequirements.PendingEventID || event.EventID != req.ReportRequirements.PlanEventID {
			continue
		}
		plan, err := reporting.NormalizeSectionalReportPlan(payload.Plan)
		return event, plan, err
	}
	return app.LedgerEvent{}, reporting.SectionalReportPlan{}, fmt.Errorf("report requirement fixture plan is missing")
}

type fanoutRequirementAgent struct {
	mu            sync.Mutex
	requests      []AgentRequest
	forkSources   []string
	forkSequence  int
	plan          agentSectionalReportPlan
	finalMarkdown string
}

func (agent *fanoutRequirementAgent) Run(_ context.Context, req AgentRequest) (AgentResult, error) {
	agent.mu.Lock()
	agent.requests = append(agent.requests, req)
	agent.mu.Unlock()
	sessionID := strings.TrimSpace(req.PreviousSessionID)
	if sessionID == "" {
		sessionID = "report-session"
	}
	switch {
	case req.ReportPlan != nil:
		return AgentResult{Text: agentReportAnyJSON(agent.plan), SessionID: "report-session"}, nil
	case req.ReportRequirements != nil:
		return AgentResult{Text: "requirements mapped", SessionID: sessionID}, nil
	case req.PartAssembly != nil:
		return AgentResult{Text: `{"intro":"Intro","transitions":[],"closing":"Close"}`, SessionID: sessionID}, nil
	case req.FinalEditStage != nil && (req.FinalEditStage.Stage == reporting.FinalEditStageGate || req.FinalEditStage.Stage == reporting.FinalEditStageEvidenceGate):
		return AgentResult{Text: finalEditGateSubmittedSentinel, SessionID: sessionID}, nil
	case req.FinalEditStage != nil:
		return AgentResult{Text: finalEditStageSubmittedSentinel, SessionID: sessionID}, nil
	case req.LongFormFinalize != nil:
		return AgentResult{Text: agent.finalMarkdown, SessionID: sessionID}, nil
	case strings.Contains(req.UserText, "draft section 1.1"):
		return AgentResult{Text: "Overview section.", SessionID: sessionID}, nil
	case strings.Contains(req.UserText, "draft section 1.2"):
		return AgentResult{Text: "Constraints section.", SessionID: sessionID}, nil
	default:
		return AgentResult{Text: "agent response", SessionID: sessionID}, nil
	}
}

func (agent *fanoutRequirementAgent) ForkSession(_ context.Context, sourceSessionID string) (AgentSessionForkResult, error) {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	agent.forkSources = append(agent.forkSources, sourceSessionID)
	agent.forkSequence++
	sessionID := fmt.Sprintf("fanout-session-%d", agent.forkSequence)
	return AgentSessionForkResult{SessionID: sessionID, SourceSessionID: sourceSessionID, SourceHash: "source-hash", CloneHash: "clone-hash", SourceSizeBytes: 100, CloneSizeBytes: 100}, nil
}

func (agent *fanoutRequirementAgent) CheckForkSession(_ context.Context, sourceSessionID string) error {
	if strings.TrimSpace(sourceSessionID) == "" {
		return fmt.Errorf("source session id is required")
	}
	return nil
}

func (agent *fanoutRequirementAgent) snapshotRequests() []AgentRequest {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	return append([]AgentRequest(nil), agent.requests...)
}

func requestByUserText(t *testing.T, requests []AgentRequest, text string) AgentRequest {
	t.Helper()
	for _, req := range requests {
		if strings.Contains(req.UserText, text) {
			return req
		}
	}
	t.Fatalf("request with user text %q was not recorded: %#v", text, requests)
	return AgentRequest{}
}

func requestHasMCPTool(req AgentRequest, tool string) bool {
	for _, candidate := range req.ExtraMCPTools {
		if candidate == tool {
			return true
		}
	}
	return false
}

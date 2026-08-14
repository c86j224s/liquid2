package web

import (
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func TestStaticMissionMetadataAndReportDirectionContracts(t *testing.T) {
	files := []string{"static/index.html", "static/app.js", "static/mission_metadata.js", "static/plasma/reports_direction.js", "static/plasma/reports_controls.js", "static/app.css"}
	var combined strings.Builder
	for _, name := range files {
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		combined.Write(content)
		combined.WriteByte('\n')
	}
	text := combined.String()
	for _, expected := range []string{"missionSettingsDetails", "missionMetadataForm", "missionMetadataIncluded", "missionMetadataExcluded", "missionMetadataLines", ".filter(Boolean)", `method: "PATCH"`, "reportDirectionHint", "direction_hint", "clearAcceptedReportDirectionHint", "catch (err)"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing static contract %q", expected)
		}
	}
	clearIndex := strings.Index(mustReadPlasmaReportScripts(t), "clearAcceptedReportDirectionHint")
	catchIndex := strings.Index(mustReadPlasmaReportScripts(t), "} catch (err) {")
	if clearIndex < 0 || catchIndex < 0 {
		t.Fatal("missing success-clear or failure branch")
	}
}

func TestPlasmaFoundationOwnerContracts(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	app := string(mustReadStatic(t, "static/app.js"))
	namespace := string(mustReadStatic(t, "static/plasma/namespace.js"))
	dom := string(mustReadStatic(t, "static/plasma/dom.js"))
	stateScript := string(mustReadStatic(t, "static/plasma/state.js"))
	ui := string(mustReadStatic(t, "static/plasma/ui.js"))
	uiFeedback := string(mustReadStatic(t, "static/plasma/ui_feedback.js"))
	uiDetail := string(mustReadStatic(t, "static/plasma/ui_detail.js"))
	mission := string(mustReadStatic(t, "static/plasma/mission.js"))
	transport := string(mustReadStatic(t, "static/plasma/transport.js"))
	polling := string(mustReadStatic(t, "static/plasma/polling.js"))
	conversation := string(mustReadStatic(t, "static/plasma/conversation.js"))
	conversationAgentState := string(mustReadStatic(t, "static/plasma/conversation_agent_state.js"))
	conversationAgentModels := string(mustReadStatic(t, "static/plasma/conversation_agent_models.js"))
	conversationAgentControls := string(mustReadStatic(t, "static/plasma/conversation_agent_controls.js"))
	conversationAgentSession := string(mustReadStatic(t, "static/plasma/conversation_agent_session.js"))
	conversationActiveWork := string(mustReadStatic(t, "static/plasma/conversation_active_work.js"))
	conversationTurnState := string(mustReadStatic(t, "static/plasma/conversation_turn_state.js"))
	conversationLiveTurn := string(mustReadStatic(t, "static/plasma/conversation_live_turn.js"))
	conversationTurnNav := string(mustReadStatic(t, "static/plasma/conversation_turn_nav.js"))
	conversationTurns := string(mustReadStatic(t, "static/plasma/conversation_turns.js"))
	workflow := string(mustReadStatic(t, "static/plasma/workflow.js"))
	workflowInput := string(mustReadStatic(t, "static/plasma/workflow_input.js"))
	workflowRendering := string(mustReadStatic(t, "static/plasma/workflow_rendering.js"))

	orderedScripts := []string{
		`/static/plasma/namespace.js`,
		`/static/plasma/dom.js`,
		`/static/plasma/state.js`,
		`/static/plasma/ui.js`,
		`/static/plasma/ui_feedback.js`,
		`/static/plasma/ui_detail.js`,
		`/static/plasma/mission.js`,
		`/static/plasma/transport.js`,
		`/static/plasma/polling.js`,
		`/static/plasma/conversation.js`,
		`/static/plasma/conversation_agent_state.js`,
		`/static/plasma/conversation_agent_models.js`,
		`/static/plasma/conversation_agent_controls.js`,
		`/static/plasma/conversation_agent_session.js`,
		`/static/plasma/conversation_active_work.js`,
		`/static/plasma/conversation_turn_state.js`,
		`/static/plasma/conversation_live_turn.js`,
		`/static/plasma/conversation_turn_nav.js`,
		`/static/plasma/conversation_turns.js`,
		`/static/plasma/workflow.js`,
		`/static/plasma/workflow_input.js`,
		`/static/plasma/workflow_rendering.js`,
		`/static/app.js`,
	}
	last := -1
	for _, script := range orderedScripts {
		at := strings.Index(index, script)
		if at < 0 || at <= last {
			t.Fatalf("Plasma foundation script %q missing or out of order", script)
		}
		last = at
	}

	if !strings.Contains(namespace, "root.Plasma = root.Plasma || {};") {
		t.Fatal("namespace.js must initialize window.Plasma")
	}
	for file, owner := range map[string]struct {
		content    string
		assignment string
	}{
		"static/plasma/dom.js":                         {content: dom, assignment: "Plasma.dom ="},
		"static/plasma/state.js":                       {content: stateScript, assignment: "Plasma.state ="},
		"static/plasma/ui.js":                          {content: ui, assignment: "Plasma.ui ="},
		"static/plasma/ui_feedback.js":                 {content: uiFeedback, assignment: "Object.assign(Plasma.ui"},
		"static/plasma/ui_detail.js":                   {content: uiDetail, assignment: "Object.assign(Plasma.ui"},
		"static/plasma/mission.js":                     {content: mission, assignment: "Plasma.mission ="},
		"static/plasma/transport.js":                   {content: transport, assignment: "Plasma.transport ="},
		"static/plasma/polling.js":                     {content: polling, assignment: "Plasma.polling ="},
		"static/plasma/conversation.js":                {content: conversation, assignment: "Plasma.conversation"},
		"static/plasma/conversation_agent_state.js":    {content: conversationAgentState, assignment: "Plasma.conversation"},
		"static/plasma/conversation_agent_models.js":   {content: conversationAgentModels, assignment: "Plasma.conversation"},
		"static/plasma/conversation_agent_controls.js": {content: conversationAgentControls, assignment: "Plasma.conversation"},
		"static/plasma/conversation_agent_session.js":  {content: conversationAgentSession, assignment: "Plasma.conversation"},
		"static/plasma/conversation_active_work.js":    {content: conversationActiveWork, assignment: "Plasma.conversation"},
		"static/plasma/conversation_turn_state.js":     {content: conversationTurnState, assignment: "Plasma.conversation"},
		"static/plasma/conversation_live_turn.js":      {content: conversationLiveTurn, assignment: "Plasma.conversation"},
		"static/plasma/conversation_turn_nav.js":       {content: conversationTurnNav, assignment: "Plasma.conversation"},
		"static/plasma/conversation_turns.js":          {content: conversationTurns, assignment: "Plasma.conversation"},
		"static/plasma/workflow.js":                    {content: workflow, assignment: "Plasma.workflow"},
		"static/plasma/workflow_input.js":              {content: workflowInput, assignment: "Plasma.workflow"},
		"static/plasma/workflow_rendering.js":          {content: workflowRendering, assignment: "Plasma.workflow"},
	} {
		if !strings.Contains(owner.content, owner.assignment) {
			t.Fatalf("%s must contain owner assignment %q", file, owner.assignment)
		}
	}
	for file, content := range map[string]string{
		"static/plasma/dom.js":                         dom,
		"static/plasma/state.js":                       stateScript,
		"static/plasma/ui.js":                          ui,
		"static/plasma/ui_feedback.js":                 uiFeedback,
		"static/plasma/ui_detail.js":                   uiDetail,
		"static/plasma/mission.js":                     mission,
		"static/plasma/transport.js":                   transport,
		"static/plasma/polling.js":                     polling,
		"static/plasma/conversation.js":                conversation,
		"static/plasma/conversation_agent_state.js":    conversationAgentState,
		"static/plasma/conversation_agent_models.js":   conversationAgentModels,
		"static/plasma/conversation_agent_controls.js": conversationAgentControls,
		"static/plasma/conversation_agent_session.js":  conversationAgentSession,
		"static/plasma/conversation_active_work.js":    conversationActiveWork,
		"static/plasma/conversation_turn_state.js":     conversationTurnState,
		"static/plasma/conversation_live_turn.js":      conversationLiveTurn,
		"static/plasma/conversation_turn_nav.js":       conversationTurnNav,
		"static/plasma/conversation_turns.js":          conversationTurns,
		"static/plasma/workflow.js":                    workflow,
		"static/plasma/workflow_input.js":              workflowInput,
		"static/plasma/workflow_rendering.js":          workflowRendering,
	} {
		if !strings.Contains(content, "})(window.Plasma);") {
			t.Fatalf("%s must attach through the existing Plasma root", file)
		}
		if !strings.HasPrefix(file, "static/plasma/ui") && file != "static/plasma/polling.js" && !strings.HasPrefix(file, "static/plasma/conversation") && strings.Contains(strings.ReplaceAll(content, "window.Plasma", ""), "window.") {
			t.Fatalf("%s must not introduce another window root", file)
		}
	}

	for _, expected := range []string{
		"missions: []",
		"missionId: \"\"",
		"selectionGeneration: 0",
		"detailGeneration: 0",
		"sourceCandidateBusy: new Set()",
		"selectedSourceCandidates: new Set()",
		"agentReasoningEffortExecutor: \"\"",
	} {
		if !strings.Contains(stateScript, expected) {
			t.Fatalf("state.js changed or omitted state default %q", expected)
		}
	}

	ownerDefinitions := map[string][]string{
		"static/plasma/dom.js": {
			"function $(",
			"function shortID(",
			"function timeShort(",
			"function escapeHTML(",
			"function escapeAttr(",
			"function formatBytes(",
		},
		"static/plasma/ui.js": {
			"function setSectionEmpty(",
			"function updateCountChip(",
			"function empty(",
			"function setElementDisabled(",
			"function setFormButtonsDisabled(",
			"function setButtonText(",
		},
		"static/plasma/ui_feedback.js": {
			"function showError(",
			"function hideError(",
			"async function copyError(",
			"async function copyText(",
		},
		"static/plasma/ui_detail.js": {
			"function showDetail(",
			"function openDetailModal(",
			"async function copyDetail(",
			"function hideDetail(",
			"function enableDetailScrollRatio(",
			"function disableDetailScrollRatio(",
			"function updateDetailScrollRatio(",
			"function detailScrollPosition(",
			"function onDetailModalClick(",
		},
		"static/plasma/mission.js": {
			"function captureMissionSelection(",
			"function ownsMissionSelection(",
			"function ownsDetailRequest(",
			"class StaleMissionOperationError",
			"function isStaleMissionOperation(",
			"function beginMissionSelection(",
			"function clearMissionSelection(",
			"function beforeSelectionChange(",
			"async function afterSelectionApplied(",
			"function applyMissionDetail(",
			"async function refreshSelectedMissionDetail(",
		},
		"static/plasma/transport.js": {
			"async function api(",
			"async function missionApi(",
			"async function missionFetch(",
		},
		"static/plasma/polling.js": {
			"function schedulePendingPoll(",
			"function clearPendingPoll(",
			"function scheduleMissionActivityPoll(",
			"async function refreshObservedMissionActivity(",
			"function missionActivityCursor(",
			"function detailMissionActivityCursor(",
			"function mergeMissionActivity(",
			"function recordDetailActivityCursor(",
			"async function refreshSelectedMissionActivity(",
		},
		"static/plasma/conversation.js": {
			"async function sendTurn(",
			"async function cancelTurn(",
			"function setTurnBusy(",
		},
		"static/plasma/conversation_agent_state.js": {
			"function agentExecutorStatus(",
			"function lockedAgentExecutor(",
			"function selectedAgentModel(",
			"function agentEventMatchesExecutor(",
		},
		"static/plasma/conversation_agent_models.js": {
			"function renderAgentModelOptions(",
			"function renderAgentReasoningEffortOptions(",
			"function currentAgentModel(",
			"function currentAgentReasoningEffort(",
		},
		"static/plasma/conversation_agent_controls.js": {
			"function renderAgentOptions(",
			"function onAgentExecutorChange(",
		},
		"static/plasma/conversation_agent_session.js": {
			"async function resetAgentSession(",
			"function renderAgentSessionStatus(",
			"function renderAgentControlsSummary(",
		},
		"static/plasma/conversation_active_work.js": {
			"function renderActiveWork(",
			"function displayActiveWorkItems(",
			"function activeWorkControlElementIDs(",
		},
		"static/plasma/conversation_turn_state.js": {
			"function hasOpenPendingTurn(",
			"function completedUserEventIDs(",
		},
		"static/plasma/conversation_live_turn.js": {
			"function startLiveTurn(",
			"function handleLiveTurnSnapshot(",
			"function syncLiveTurnSubscription(",
			"function liveTurnSnapshot(",
		},
		"static/plasma/conversation_turn_nav.js": {
			"function updateTurnNavVisibility(",
			"function onTurnNavClick(",
			"function turnNavScroll(",
		},
		"static/plasma/conversation_turns.js": {
			"function renderTurns(",
			"function renderSessionResetTurn(",
			"function renderStandaloneSteeringTurn(",
		},
		"static/plasma/workflow.js": {
			"async function startWorkflow(",
			"async function stopWorkflow(",
			"async function continueWorkflowRun(",
			"function workflowContinuationInstruction(",
		},
		"static/plasma/workflow_input.js": {
			"function workflowRawInputValue(",
			"async function draftWorkflowGoal(",
			"function setWorkflowBusy(",
		},
		"static/plasma/workflow_rendering.js": {
			"function renderWorkflowControls(",
			"function renderWorkflowRun(",
			"function workflowStatusLabel(",
			"function workflowDecisionLabel(",
		},
	}
	ownerContent := map[string]string{
		"static/plasma/dom.js":                         dom,
		"static/plasma/ui.js":                          ui,
		"static/plasma/ui_feedback.js":                 uiFeedback,
		"static/plasma/ui_detail.js":                   uiDetail,
		"static/plasma/mission.js":                     mission,
		"static/plasma/transport.js":                   transport,
		"static/plasma/polling.js":                     polling,
		"static/plasma/conversation.js":                conversation,
		"static/plasma/conversation_agent_state.js":    conversationAgentState,
		"static/plasma/conversation_agent_models.js":   conversationAgentModels,
		"static/plasma/conversation_agent_controls.js": conversationAgentControls,
		"static/plasma/conversation_agent_session.js":  conversationAgentSession,
		"static/plasma/conversation_active_work.js":    conversationActiveWork,
		"static/plasma/conversation_turn_state.js":     conversationTurnState,
		"static/plasma/conversation_live_turn.js":      conversationLiveTurn,
		"static/plasma/conversation_turn_nav.js":       conversationTurnNav,
		"static/plasma/conversation_turns.js":          conversationTurns,
		"static/plasma/workflow.js":                    workflow,
		"static/plasma/workflow_input.js":              workflowInput,
		"static/plasma/workflow_rendering.js":          workflowRendering,
	}
	for file, definitions := range ownerDefinitions {
		for _, definition := range definitions {
			if strings.Count(ownerContent[file], definition) != 1 {
				t.Fatalf("%s must contain exactly one owner definition %q", file, definition)
			}
			if strings.Contains(app, definition) {
				t.Fatalf("app.js must not retain moved owner definition %q", definition)
			}
		}
	}
	if strings.Contains(app, "const state = {") {
		t.Fatal("app.js must not retain the browser state object")
	}
	if strings.Contains(mission, "Plasma.polling") || strings.Contains(mission, "state.missionActivityCursors") {
		t.Fatal("Plasma.mission must not own activity cursor parsing/storage or directly depend on Plasma.polling")
	}
	missionSelection := string(mustReadStatic(t, "static/plasma/mission_selection.js"))
	for _, forbidden := range []string{"Plasma.reports", "Plasma.sources", "loadConfluence", "redpenController"} {
		if strings.Contains(missionSelection, forbidden) {
			t.Fatalf("mission_selection.js must use generic mission transition hooks, not cross-feature policy %q", forbidden)
		}
	}
	for _, expected := range []string{
		"beforeSelectionChange: (currentMissionId, nextMissionId)",
		"afterSelectionApplied: async (owner)",
		"Plasma.reports.redpenController?.beforeLeave()",
		"Plasma.sources.loadConfluenceConnections(\"\", owner)",
		"Plasma.mission.ownsDetailRequest(owner)",
		"Plasma.sources.loadConfluenceAccess(owner)",
	} {
		if !strings.Contains(app, expected) {
			t.Fatalf("app.js must own mission transition composition callback %q", expected)
		}
	}
	if strings.Contains(polling, "\"정상\"") || strings.Contains(polling, "\"재연결 중\"") || strings.Contains(polling, "healthBadge") {
		t.Fatal("Plasma.polling must not own user-facing health badge copy or DOM updates")
	}

	for _, alias := range []string{
		"Foundation transition",
		"const state = window.Plasma.state;",
		"const $ = window.Plasma.dom.$;",
		"var shortID = window.Plasma.dom.shortID;",
		"var timeShort = window.Plasma.dom.timeShort;",
		"var escapeHTML = window.Plasma.dom.escapeHTML;",
		"var escapeAttr = window.Plasma.dom.escapeAttr;",
		"var formatBytes = window.Plasma.dom.formatBytes;",
		"var captureMissionSelection = window.Plasma.mission.captureMissionSelection;",
		"var ownsMissionSelection = window.Plasma.mission.ownsMissionSelection;",
		"var ownsDetailRequest = window.Plasma.mission.ownsDetailRequest;",
		"const StaleMissionOperationError = window.Plasma.mission.StaleMissionOperationError;",
		"var isStaleMissionOperation = window.Plasma.mission.isStaleMissionOperation;",
		"var api = window.Plasma.transport.api;",
		"var missionApi = window.Plasma.transport.missionApi;",
		"var missionFetch = window.Plasma.transport.missionFetch;",
		"var beginMissionSelection = window.Plasma.mission.beginMissionSelection;",
		"var clearMissionSelection = window.Plasma.mission.clearMissionSelection;",
		"var applyMissionDetail = window.Plasma.mission.applyMissionDetail;",
		"var refreshSelectedMissionDetail = window.Plasma.mission.refreshSelectedMissionDetail;",
		"var clearPendingPoll = window.Plasma.polling.clearPendingPoll;",
		"var scheduleMissionActivityPoll = window.Plasma.polling.scheduleMissionActivityPoll;",
		"var refreshObservedMissionActivity = window.Plasma.polling.refreshObservedMissionActivity;",
		"var missionActivityCursor = window.Plasma.polling.missionActivityCursor;",
		"var detailMissionActivityCursor = window.Plasma.polling.detailMissionActivityCursor;",
		"var mergeMissionActivity = window.Plasma.polling.mergeMissionActivity;",
		"var refreshSelectedMissionActivity = window.Plasma.polling.refreshSelectedMissionActivity;",
	} {
		if strings.Contains(app, alias) {
			t.Fatalf("app.js retains transition alias %q", alias)
		}
	}
	if strings.Contains(app, "function empty(") || strings.Contains(dom, "function empty(") || !strings.Contains(ui, "function empty(") {
		t.Fatal("empty-state markup must be owned only by Plasma.ui")
	}

	for _, forbidden := range []string{"state.", "fetch(", "localStorage", "navigator.clipboard", "showDetail", "showError", "modal"} {
		if strings.Contains(dom, forbidden) {
			t.Fatalf("Plasma.dom must not own browser state, network, modal, or clipboard behavior: %q", forbidden)
		}
	}
	for _, expected := range []string{
		"Plasma.ui = {",
	} {
		if !strings.Contains(ui, expected) {
			t.Fatalf("Plasma.ui controls file must preserve shared UI behavior %q", expected)
		}
	}
	for _, expected := range []string{
		"detailHooks.beforeLeave",
		"detailHooks.copyContent",
		"`위치 ${Math.max(0, Math.min(100, percent))}%`",
		"event.target.closest(\"[data-detail-json]\")",
	} {
		if !strings.Contains(uiDetail, expected) {
			t.Fatalf("Plasma.ui detail file must preserve shared UI behavior %q", expected)
		}
	}
	for _, expected := range []string{
		"err.userMessage || err.message || String(err)",
		"state.lastError = err.stack",
		"JSON.stringify(err.details, null, 2)",
		"err.isNetworkError",
		"badge.textContent = \"연결 끊김\"",
		"navigator.clipboard?.writeText",
		"document.execCommand(\"copy\")",
	} {
		if !strings.Contains(uiFeedback, expected) {
			t.Fatalf("Plasma.ui feedback file must preserve shared UI behavior %q", expected)
		}
	}
	for _, forbidden := range []string{
		"renderMarkdown",
		"showClaimConfidenceDetail",
		"reportRedpenController",
		"state.reportPending",
		"state.workflowPending",
		"missionLifecycleWriteBlocked",
		"activeWorkBlocksControl",
		"fetch(",
		"localStorage",
	} {
		if strings.Contains(ui, forbidden) || strings.Contains(uiFeedback, forbidden) || strings.Contains(uiDetail, forbidden) {
			t.Fatalf("Plasma.ui files must not own feature markup, product policy, network, or persistence behavior: %q", forbidden)
		}
	}
	for _, expected := range []string{
		"Network request failed",
		"wrapped.userMessage",
		"response.text()",
		"JSON.parse(text)",
		"mission.ownsMissionSelection(owner)",
		"new mission.StaleMissionOperationError()",
	} {
		if !strings.Contains(transport, expected) {
			t.Fatalf("transport.js must preserve transport behavior %q", expected)
		}
	}
	for _, forbidden := range []string{
		"state.reportPending",
		"state.workflowPending",
		"state.workflowGoalDraftPending",
	} {
		for _, content := range []struct {
			name string
			body string
		}{
			{name: "conversation.js", body: conversation},
			{name: "conversation_agent_controls.js", body: conversationAgentControls},
			{name: "conversation_agent_session.js", body: conversationAgentSession},
			{name: "conversation_active_work.js", body: conversationActiveWork},
			{name: "conversation_turn_state.js", body: conversationTurnState},
			{name: "conversation_live_turn.js", body: conversationLiveTurn},
			{name: "conversation_turn_nav.js", body: conversationTurnNav},
			{name: "conversation_turns.js", body: conversationTurns},
		} {
			if strings.Contains(content.body, forbidden) {
				t.Fatalf("%s must receive cross-feature blocking through app.js callbacks, not %q", content.name, forbidden)
			}
		}
	}
	for _, forbidden := range []string{"state.reportPending", "state.turnPending"} {
		for _, content := range []struct {
			name string
			body string
		}{
			{name: "workflow.js", body: workflow},
			{name: "workflow_input.js", body: workflowInput},
			{name: "workflow_rendering.js", body: workflowRendering},
		} {
			if strings.Contains(content.body, forbidden) {
				t.Fatalf("%s must receive cross-feature blocking through app.js callbacks, not %q", content.name, forbidden)
			}
		}
	}
	for _, expected := range []string{"function turnControlsBlocked(", "function agentControlsBlocked(", "function workflowControlsBlocked(", "function workflowContinueBlocked("} {
		if !strings.Contains(app, expected) {
			t.Fatalf("app.js must own cross-feature blocking callback %q", expected)
		}
	}
}

func TestReportPipelineStaticGraphAndRetryContracts(t *testing.T) {
	script := string(mustReadStatic(t, "static/plasma/reports_pipeline_core.js")) + string(mustReadStatic(t, "static/plasma/reports_pipeline_graph.js")) + string(mustReadStatic(t, "static/plasma/reports_pipeline_render.js"))
	styles := string(mustReadStatic(t, "static/report_pipeline.css"))
	for _, expected := range []string{"<svg class=\"pipeline-graph", "--pipeline-width:", "--pipeline-height:", "pipeline-graph-fanout", "pipeline-visual-phase-fanout", "pathConnector", "pipelineLiveTimingTimer", "syncLiveTiming", "data-pipeline-live-timing", "data-pipeline-started-at", "data-pipeline-title-prefix", "최신 리포트 생성 파이프라인", "currentReportAttemptEvent(progress.attempt_id)", "<details class=\"pipeline-request-details\"", "생성 요청 상세", "<details class=\"pipeline-details\"", "role=\"img\"", "<ol class=\"pipeline-flow sr-only\"", "<li class=\"pipeline-node", "pipeline-phase", "요구 연결", "파트 계획", "섹션 작성", "파트 조립", "파트 편집", "파트 최종 작성", "최종 조립", "최종 작성", "독자 편집", "말투 편집", "근거·요구 교정", "finalEditClosingNodes(nodes)", "섹션 ${runningSections.length}개 병렬 작성", "파트 ${runningPartPlans.length}개 병렬 계획", "파트 ${runningPartAuthors.length}개 최종 작성", "phaseSummary(nodes)", "aria-current=\\\"step\\\"", "currentStage(graphNodes)", "hasPlannedContent(nodes)", "captureMissionSelection()", "isStaleMissionOperation(error)", "resume_failed", "restart", "started_at", "duration_ms", "visualNodeWidth(node)", "data-pipeline-node-width", "visualScrollLeft", "renderedVisual.scrollLeft"} {
		if !strings.Contains(script, expected) {
			t.Fatalf("missing report pipeline contract %q", expected)
		}
	}
	for _, expected := range []string{".pipeline-request-details", ".pipeline-details", ".report-generation-summary", ".report-generation-item", ".report-direction-line", "white-space: nowrap", ".pipeline-visual", "max-width: 100%", "overflow-x: auto", "min-width: 0", "width: max(100%, var(--pipeline-width))", "height: var(--pipeline-height, 136px)", ".pipeline-visual-dot", ".pipeline-visual-time", "font-variant-numeric: tabular-nums", "pipeline-node-pulse", "prefers-reduced-motion: reduce", "pipeline-graph-revealing"} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing report pipeline style contract %q", expected)
		}
	}
	if strings.Contains(styles, ".pipeline-attempt-meta") {
		t.Fatal("legacy pipeline attempt metadata grid should not be used")
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	fixture := `
const fs=require("fs"), vm=require("vm");
(async()=>{
  let reloads=0, current=true;
  const requests=[];
  const button=(strategy)=>({dataset:{reportRetry:strategy},disabled:false,addEventListener(_event,listener){this.listener=listener;}});
  const resume=button("resume_failed"), restart=button("restart");
  let pipelineVisual={scrollLeft:73}, pipelineDetails={open:false}, requestDetails={open:false};
  const host={_innerHTML:"",get innerHTML(){return this._innerHTML;},set innerHTML(value){this._innerHTML=value;pipelineVisual={scrollLeft:0};},querySelector(selector){return selector===".pipeline-visual"?pipelineVisual:null;},querySelectorAll(selector){
    if(selector==="[data-report-retry]") return [resume,restart];
    return [];
  }};
  const context={window:{},state:{detail:{events:[{EventID:"evt_failed",EventType:"report.draft.pending",Payload:{title:"<안전한 제목>",started_at:"2026-07-13T01:02:03Z"}}]}},document:{getElementById(id){return id==="reportPipeline"?host:null}},crypto:{randomUUID(){return "retry"}},setInterval(){return 99;},clearInterval(){},
    captureMissionSelection(){return {missionId:"mis_a"}},
    missionFetch(_owner,_path,options){requests.push(JSON.parse(options.body));return Promise.resolve({ok:true});},
    ownsMissionSelection(){return current;},reloadMission(){reloads++;},isStaleMissionOperation(){return false}};
  context.window.Plasma={reports:{},state:context.state,mission:{captureMissionSelection:context.captureMissionSelection,ownsMissionSelection:context.ownsMissionSelection,isStaleMissionOperation:context.isStaleMissionOperation},transport:{missionFetch:context.missionFetch},dom:{timeShort:(v)=>v}}; context.window.document=context.document; context.window.crypto=context.crypto; context.window.setInterval=context.setInterval; context.window.clearInterval=context.clearInterval; context.window.Plasma.reports.call=(name,...args)=>name==="reloadMission"?context.reloadMission(...args):undefined; vm.createContext(context); for (const file of ["static/plasma/reports_pipeline_core.js","static/plasma/reports_pipeline_graph.js","static/plasma/reports_pipeline_render.js"]) vm.runInContext(fs.readFileSync(file,"utf8"),context);
  host.querySelector=(selector)=>selector===".pipeline-visual"?pipelineVisual:selector===".pipeline-details"?pipelineDetails:selector===".pipeline-request-details"?requestDetails:null;
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_failed",attempt_number:1,state:"failed",nodes:[{id:"plan",kind:"plan",state:"completed",started_at:"2026-07-13T01:02:03Z",duration_ms:12000},{id:"section-1-1",kind:"section",part_index:1,section_index:1,state:"completed",started_at:"2026-07-13T01:02:03Z",duration_ms:12000},{id:"section-2-1",kind:"section",part_index:2,section_index:1,state:"running",started_at:"2026-07-13T01:02:15Z"},{id:"part-1",kind:"part",part_index:1,state:"pending"},{id:"part-2",kind:"part",part_index:2,state:"failed",error:"safe",started_at:"2026-07-13T01:02:03Z",duration_ms:360000000},{id:"final",kind:"final",state:"pending"},{id:"artifact",kind:"artifact",state:"pending"}],retry:{resume_failed:true,restart:true}},{mode:"<일반>",strategy:"",guidance:"문장 중심",rigor:"균형",model:"gpt-safe",effort:"low",direction:"<방향>",startedAt:"2026. 7. 13. 10:02:03",startedAtDateTime:"2026-07-13T01:02:03Z"});
  const html=host.innerHTML;
  if(!html.includes("<h3 id=\"reportPipelineTitle\">최신 리포트 생성 파이프라인</h3>")||!html.includes("&lt;안전한 제목&gt;")||!html.includes("<summary>생성 요청 상세</summary>")||!html.includes("class=\"report-generation-summary\" aria-label=\"생성 요청 설정\"")||!html.includes("class=\"report-generation-item\"><strong>방식</strong><span>&lt;일반&gt;</span>")||!html.includes("class=\"report-generation-item report-direction-line\"><strong>방향</strong><span>&lt;방향&gt;</span>")||!html.includes("글쓰기")||!html.includes("엄격도")||!html.includes("모델")||!html.includes("추론")||!html.includes("전체 생성 시작")||!html.includes("<time datetime=\"2026-07-13T01:02:03Z\">2026. 7. 13. 10:02:03</time>")||html.includes("pipeline-attempt-meta")||!html.includes("시도 1")||!html.includes("시작")||!html.includes("소요 12초")||!html.includes("경과")||!html.includes("<details class=\"pipeline-request-details\">")||!html.includes("<details class=\"pipeline-details\">")||html.includes("<details class=\"pipeline-details\" open")||html.includes("<details class=\"pipeline-request-details\" open")||!html.includes("<svg class=\"pipeline-graph\"")||!html.includes("--pipeline-width:")||!html.includes("<ol class=\"pipeline-flow sr-only\"")||!html.includes("<li class=\"pipeline-phase\"")||!html.includes("섹션 작성")||!html.includes("파트 조립"))process.exit(1);
  if(pipelineVisual.scrollLeft!==73)process.exit(10);
  requestDetails.open=true; pipelineDetails.open=false;
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_failed",attempt_number:1,state:"failed",nodes:[{id:"plan",kind:"plan",state:"running"}]},{mode:"일반",rigor:"균형",model:"gpt-safe",effort:"low",direction:"지정 없음",startedAt:"2026-07-13T01:02:03Z",startedAtDateTime:"2026-07-13T01:02:03Z"});
  if(!host.innerHTML.includes("<details class=\"pipeline-request-details\" open>")||host.innerHTML.includes("<details class=\"pipeline-details\" open>"))process.exit(25);
  requestDetails.open=false; pipelineDetails.open=true;
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_failed",attempt_number:1,state:"failed",nodes:[{id:"plan",kind:"plan",state:"running"}]},{mode:"일반",rigor:"균형",model:"gpt-safe",effort:"low",direction:"지정 없음",startedAt:"2026-07-13T01:02:03Z",startedAtDateTime:"2026-07-13T01:02:03Z"});
  if(host.innerHTML.includes("<details class=\"pipeline-request-details\" open>")||!host.innerHTML.includes("<details class=\"pipeline-details\" open>"))process.exit(26);
  const graphNodes=[...html.matchAll(/data-pipeline-node-width="(\d+)" transform="translate\((\d+) 62\)"/g)].map(([,width,x])=>({width:Number(width),x:Number(x)}));
  if(graphNodes.length!==7||graphNodes.some((node,index)=>index>0&&node.x-graphNodes[index-1].x<(node.width+graphNodes[index-1].width)/2+32))process.exit(11);
  if(!html.includes("role=\"img\"")||!html.includes("aria-current=\"step\"")||!html.includes("data-pipeline-live-timing=\"1\"")||!html.includes("data-pipeline-started-at=\"2026-07-13T01:02:15Z\"")||!html.includes("data-pipeline-title-prefix=\"섹션 2.1\"")||!html.includes("aria-label=\"파트 조립 2 실패, 시작")||!html.includes("safe\"")||!html.includes("tabindex=\"0\""))process.exit(2);
  if(!(html.indexOf("pipeline-plan") < html.indexOf("pipeline-section-1-1") && html.indexOf("pipeline-section-1-1") < html.indexOf("pipeline-section-2-1") && html.indexOf("pipeline-section-2-1") < html.indexOf("pipeline-part-1") && html.indexOf("pipeline-part-1") < html.indexOf("pipeline-part-2") && html.indexOf("pipeline-part-2") < html.indexOf("pipeline-final") && html.indexOf("pipeline-final") < html.indexOf("pipeline-artifact")))process.exit(7);
  if(typeof resume.listener!=="function"||typeof restart.listener!=="function")process.exit(3);
  await resume.listener();
  current=false;
  await restart.listener();
  if(requests.length!==2||requests[0].strategy!=="resume_failed"||requests[1].strategy!=="restart")process.exit(4);
  if(reloads!==1)process.exit(5);
  context.state.detail.events=[{EventID:"evt_planned",EventType:"report.draft.pending",Payload:{title:"일반 보고서",started_at:"2026-07-13T01:02:03Z",report_mode:"planned"}}];
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_planned",attempt_number:1,state:"running",nodes:[{id:"start",kind:"start",state:"completed"},{id:"final",kind:"final",state:"running"},{id:"artifact",kind:"artifact",state:"pending"}]});
  const plannedRunningHtml=host.innerHTML;
  if(!plannedRunningHtml.includes("pipeline-start")||!plannedRunningHtml.includes("pipeline-final")||!plannedRunningHtml.includes("pipeline-artifact")||!plannedRunningHtml.includes("pipeline-current-step\">최종 편집·확정<")||!plannedRunningHtml.includes("pipeline-current-status\">진행 중<"))process.exit(27);
  if(!(plannedRunningHtml.indexOf("pipeline-start") < plannedRunningHtml.indexOf("pipeline-final") && plannedRunningHtml.indexOf("pipeline-final") < plannedRunningHtml.indexOf("pipeline-artifact"))||(plannedRunningHtml.match(/data-pipeline-node-width=/g)||[]).length!==3||plannedRunningHtml.includes("pipeline-phase"))process.exit(28);
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_planned",attempt_number:1,state:"completed",nodes:[{id:"start",kind:"start",state:"completed"},{id:"final",kind:"final",state:"completed"},{id:"artifact",kind:"artifact",state:"completed"}]});
  const plannedCompletedHtml=host.innerHTML;
  if(!plannedCompletedHtml.includes("pipeline-final")||!plannedCompletedHtml.includes("pipeline-artifact")||!plannedCompletedHtml.includes("pipeline-current-step\">산출물 생성<")||!plannedCompletedHtml.includes("pipeline-current-status\">완료<")||plannedCompletedHtml.includes("pipeline-current-step\">계획 수립<"))process.exit(29);
  context.state.detail.events=[{EventID:"evt_humanize",EventType:"report.humanize.pending",Payload:{title:"다듬기",started_at:"2026-07-13T01:02:03Z"}}];
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_humanize",attempt_number:1,state:"running",nodes:[{id:"start",kind:"start",state:"completed"},{id:"final",kind:"final",state:"running"},{id:"artifact",kind:"artifact",state:"pending"}]});
  const compactOperationHtml=host.innerHTML;
  if(!compactOperationHtml.includes("pipeline-start")||compactOperationHtml.includes("pipeline-final")||compactOperationHtml.includes("pipeline-artifact"))process.exit(30);
  context.state.detail.events=[{EventID:"evt_part_edit",Payload:{title:"파트 편집 보고서",started_at:"2026-07-13T01:02:03Z",report_mode:"long_form"}},{EventID:"evt_staged",Payload:{title:"단계별 편집 보고서",started_at:"2026-07-13T01:02:03Z",report_mode:"long_form"}},{EventID:"evt_legacy",Payload:{title:"레거시 보고서",started_at:"2026-07-13T01:02:03Z",report_mode:"long_form"}}];
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_part_edit",attempt_number:1,state:"running",nodes:[{id:"plan",kind:"plan",state:"completed"},{id:"section-1-1",kind:"section",part_index:1,section_index:1,state:"completed"},{id:"part-1",kind:"part",part_index:1,state:"completed"},{id:"part-edit-1",kind:"part_edit",part_index:1,state:"running",started_at:"2026-07-13T01:02:20Z"},{id:"final",kind:"final",state:"pending"},{id:"artifact",kind:"artifact",state:"pending"}]});
  const partEditHtml=host.innerHTML;
  if(!partEditHtml.includes("파트 편집")||!partEditHtml.includes("파트 편집 1")||!partEditHtml.includes("pipeline-part-edit-1")||!partEditHtml.includes("data-pipeline-title-prefix=\"파트 편집 1\""))process.exit(14);
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_staged",attempt_number:1,state:"running",nodes:[{id:"plan",kind:"plan",state:"completed"},{id:"section-1-1",kind:"section",part_index:1,section_index:1,state:"completed"},{id:"part-1",kind:"part",part_index:1,state:"completed"},{id:"part-author-1",kind:"part_author",part_index:1,state:"completed"},{id:"reader-edit",kind:"reader_edit",state:"completed"},{id:"style-edit",kind:"style_edit",state:"running",started_at:"2026-07-13T01:02:20Z"},{id:"corrective-gate",kind:"corrective_gate",state:"pending"},{id:"final",kind:"final",state:"pending"},{id:"artifact",kind:"artifact",state:"pending"}]});
  const stagedHtml=host.innerHTML;
  if(!stagedHtml.includes("독자 편집")||!stagedHtml.includes("말투 편집")||!stagedHtml.includes("근거·요구 교정")||!stagedHtml.includes("pipeline-reader-edit")||!stagedHtml.includes("pipeline-style-edit")||!stagedHtml.includes("pipeline-corrective-gate")||stagedHtml.includes("pipeline-final"))process.exit(18);
  if(!(stagedHtml.indexOf("pipeline-part-author-1") < stagedHtml.indexOf("pipeline-reader-edit") && stagedHtml.indexOf("pipeline-reader-edit") < stagedHtml.indexOf("pipeline-style-edit") && stagedHtml.indexOf("pipeline-style-edit") < stagedHtml.indexOf("pipeline-corrective-gate") && stagedHtml.indexOf("pipeline-corrective-gate") < stagedHtml.indexOf("pipeline-artifact")))process.exit(19);
  if(!stagedHtml.includes("data-pipeline-title-prefix=\"말투 편집\"")||!stagedHtml.includes("pipeline-current-step\">말투 편집<"))process.exit(20);
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_staged",attempt_number:1,state:"running",nodes:[{id:"plan",kind:"plan",state:"completed"},{id:"section-1-1",kind:"section",part_index:1,section_index:1,state:"completed"},{id:"part-1",kind:"part",part_index:1,state:"completed"},{id:"reader-edit",kind:"reader_edit",state:"completed"},{id:"corrective-gate",kind:"corrective_gate",state:"running"},{id:"final",kind:"final",state:"pending"},{id:"artifact",kind:"artifact",state:"pending"}]});
  const noStyleHtml=host.innerHTML;
  if(noStyleHtml.includes("말투 편집")||noStyleHtml.includes("pipeline-style-edit")||noStyleHtml.includes("pipeline-final")||!noStyleHtml.includes("pipeline-reader-edit")||!noStyleHtml.includes("pipeline-corrective-gate"))process.exit(21);
  context.state.detail.events=[{EventID:"evt_v2",Payload:{title:"v2 보고서",started_at:"2026-07-13T01:02:03Z",report_mode:"long_form"}}];
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_v2",attempt_number:1,state:"running",nodes:[{id:"plan",kind:"plan",state:"completed"},{id:"section-1-1",kind:"section",part_index:1,section_index:1,state:"completed"},{id:"part-1",kind:"part",part_index:1,state:"completed"},{id:"part-author-1",kind:"part_author",part_index:1,state:"completed"},{id:"final-assembly",kind:"final_assembly",state:"completed"},{id:"final-write",kind:"final_write",state:"running",started_at:"2026-07-13T01:02:20Z"},{id:"reader-edit",kind:"reader_edit",state:"pending"},{id:"corrective-gate",kind:"corrective_gate",state:"pending"},{id:"final",kind:"final",state:"pending"},{id:"artifact",kind:"artifact",state:"pending"}]});
  const v2Html=host.innerHTML;
  if(!v2Html.includes("최종 조립")||!v2Html.includes("최종 작성")||!v2Html.includes("pipeline-final-assembly")||!v2Html.includes("pipeline-final-write")||v2Html.includes("id=\"pipeline-final\""))process.exit(22);
  if(!(v2Html.indexOf("pipeline-part-author-1") < v2Html.indexOf("pipeline-final-assembly") && v2Html.indexOf("pipeline-final-assembly") < v2Html.indexOf("pipeline-final-write") && v2Html.indexOf("pipeline-final-write") < v2Html.indexOf("pipeline-reader-edit") && v2Html.indexOf("pipeline-reader-edit") < v2Html.indexOf("pipeline-corrective-gate") && v2Html.indexOf("pipeline-corrective-gate") < v2Html.indexOf("pipeline-artifact")))process.exit(23);
  if(!v2Html.includes("data-pipeline-title-prefix=\"최종 작성\"")||!v2Html.includes("pipeline-current-step\">최종 작성<"))process.exit(24);
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_legacy",attempt_number:1,state:"running",nodes:[{id:"plan",kind:"plan",state:"completed"},{id:"section-1-1",kind:"section",part_index:1,section_index:1,state:"completed"},{id:"part-1",kind:"part",part_index:1,state:"running"},{id:"final",kind:"final",state:"pending"},{id:"artifact",kind:"artifact",state:"pending"}]});
  const legacyHtml=host.innerHTML;
  if(legacyHtml.includes("파트 편집")||legacyHtml.includes("pipeline-part-edit-1")||!legacyHtml.includes("pipeline-final")||!legacyHtml.includes("최종 편집·확정"))process.exit(15);
  context.state.detail.events=[{EventID:"evt_missing",Payload:{}}];
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_missing",state:"running",nodes:[]});
  if(!host.innerHTML.includes("제목 없는 리포트")||!host.innerHTML.includes("생성 시작 시각 알 수 없음")||!host.innerHTML.includes("시도 번호 알 수 없음")||!host.innerHTML.includes("계획 수립")||!host.innerHTML.includes("진행 중")||!host.innerHTML.includes("pipeline-plan")||host.innerHTML.includes("pipeline-final")||host.innerHTML.includes("pipeline-artifact")||host.innerHTML.includes("pipeline-phase"))process.exit(8);
  host.dataset={};
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_missing",state:"running",nodes:[]});
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_missing",state:"running",nodes:[{id:"plan",kind:"plan",state:"completed"},{id:"part-1",kind:"part",part_index:1,state:"running"}]});
  if(!host.innerHTML.includes("pipeline-graph-revealing")||!host.innerHTML.includes("파트 1 조립")||!host.innerHTML.includes("진행 중")||pipelineVisual.scrollLeft!==73)process.exit(9);
  context.state.detail.events=[{EventID:"evt_fanout",Payload:{title:"병렬 보고서",started_at:"2026-07-13T01:02:03Z",report_mode:"long_form",execution_strategy:"section_fanout"}}];
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_fanout",attempt_number:1,state:"running",nodes:[{id:"plan",kind:"plan",state:"completed"},{id:"section-1-1",kind:"section",part_index:1,section_index:1,state:"running"},{id:"section-2-1",kind:"section",part_index:2,section_index:1,state:"running"},{id:"part-1",kind:"part",part_index:1,state:"pending"},{id:"part-2",kind:"part",part_index:2,state:"pending"},{id:"final",kind:"final",state:"pending"},{id:"artifact",kind:"artifact",state:"pending"}]});
  const fanoutHtml=host.innerHTML;
  if(!fanoutHtml.includes("장문 · 빠른 병렬")||!fanoutHtml.includes("섹션 2개 병렬 작성")||!fanoutHtml.includes("진행 2")||!fanoutHtml.includes("pipeline-graph pipeline-graph-fanout")||!fanoutHtml.includes("계획에서 여러 섹션 작성으로 갈라지고")||!fanoutHtml.includes("pipeline-visual-phase-fanout"))process.exit(12);
  const fanoutRows=new Set([...fanoutHtml.matchAll(/transform="translate\((?:[\d.]+) ([\d.]+)\)"/g)].map(([,y])=>Number(y)));
  if(!fanoutRows.has(62)||!fanoutRows.has(146)||fanoutRows.size<2)process.exit(13);
  context.window.Plasma.reports.pipeline.render({attempt_id:"evt_fanout",attempt_number:1,state:"running",nodes:[{id:"plan",kind:"plan",state:"completed"},{id:"requirements",kind:"requirements",state:"completed"},{id:"part-plan-1",kind:"part_plan",part_index:1,state:"completed"},{id:"part-plan-2",kind:"part_plan",part_index:2,state:"running"},{id:"section-1-1",kind:"section",part_index:1,section_index:1,state:"completed"},{id:"section-2-1",kind:"section",part_index:2,section_index:1,state:"pending"},{id:"part-1",kind:"part",part_index:1,state:"completed"},{id:"part-2",kind:"part",part_index:2,state:"pending"},{id:"part-author-1",kind:"part_author",part_index:1,state:"completed"},{id:"part-author-2",kind:"part_author",part_index:2,state:"pending"},{id:"final",kind:"final",state:"pending"},{id:"artifact",kind:"artifact",state:"pending"}]});
  const w4FanoutHtml=host.innerHTML;
  if(!w4FanoutHtml.includes("파트 계획")||!w4FanoutHtml.includes("파트 최종 작성")||!w4FanoutHtml.includes("파트 2 읽기 흐름 계획")||!w4FanoutHtml.includes("pipeline-part-plan-1")||!w4FanoutHtml.includes("pipeline-part-author-1")||!w4FanoutHtml.includes("파트 계획과 여러 섹션 작성"))process.exit(16);
  if(!(w4FanoutHtml.indexOf("pipeline-requirements") < w4FanoutHtml.indexOf("pipeline-part-plan-1") && w4FanoutHtml.indexOf("pipeline-part-plan-1") < w4FanoutHtml.indexOf("pipeline-section-1-1") && w4FanoutHtml.indexOf("pipeline-part-1") < w4FanoutHtml.indexOf("pipeline-part-author-1") && w4FanoutHtml.indexOf("pipeline-part-author-1") < w4FanoutHtml.indexOf("pipeline-final")))process.exit(17);
})().catch((error)=>{console.error(error);process.exit(6);});`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("pipeline DOM fixture: %v: %s", err, out)
	}
}

func TestStaticAppCSSCompositionContracts(t *testing.T) {
	appCSS := string(mustReadStatic(t, "static/app.css"))
	var expected strings.Builder
	for _, href := range appCSSImportManifest {
		expected.WriteString(`@import url("`)
		expected.WriteString(href)
		expected.WriteString(`");`)
		expected.WriteByte('\n')
	}
	if appCSS != expected.String() {
		t.Fatalf("static/app.css must be the exact import-only composition entry")
	}

	imports := mustAppCSSImportPaths(t)
	if strings.Join(imports, "\n") != strings.Join(appCSSImportManifest, "\n") {
		t.Fatalf("static/app.css import manifest changed: got %v", imports)
	}
	for _, href := range imports {
		content := mustReadStatic(t, strings.TrimPrefix(href, "/"))
		if len(content) == 0 {
			t.Fatalf("static/app.css import is empty: %s", href)
		}
	}
	if strings.Contains(mustReadAppCSSComposed(t), `@import url("/static/plasma/`) {
		t.Fatal("composed app CSS must resolve imports before contract checks")
	}
}

func TestStaticAppCSSImportedFilesEndAtSyntaxBoundaries(t *testing.T) {
	for _, href := range mustAppCSSImportPaths(t) {
		validateCSSSegmentBoundary(t, href, string(mustReadStatic(t, strings.TrimPrefix(href, "/"))))
	}
}

func TestStaticIndexKeepsAppCSSAsStableCompositionEntry(t *testing.T) {
	html := string(mustReadStatic(t, "static/index.html"))
	matches := regexp.MustCompile(`<link rel="stylesheet" href="([^"]+)">`).FindAllStringSubmatch(html, -1)
	stylesheets := make([]string, 0, len(matches))
	for _, match := range matches {
		stylesheets = append(stylesheets, match[1])
	}
	expected := []string{
		"/static/app.css",
		"/static/image_viewer.css",
		"/static/report_pipeline.css",
		"/static/vendor/katex/katex.min.css",
		"/static/report_math.css",
		"/static/report_mermaid.css",
		"/static/report_redpen.css",
	}
	if strings.Join(stylesheets, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("index.html stylesheet order changed: got %v", stylesheets)
	}
}

func TestStaticMissionScopedActiveWorkContracts(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	script := mustReadPlasmaReportScripts(t) + mustReadPlasmaConversationScripts(t)
	styles := mustReadAppCSSComposed(t)
	combined := index + script + styles
	for _, expected := range []string{
		"active_work", "resetMissionTransientState", "ownsMissionSelection",
		"conversationActiveWork", "reportActiveWork", "report_generation_running",
		"workflow_running", "agent_turn_running", "data-active-work-action",
		".active-work-notice", "flex-wrap: wrap", "displayActiveWorkItems",
		`kind: "agent_turn"`, `reason_code: "agent_turn_running"`, `action: "cancel_turn"`,
		"리포트 생성 중입니다.", "자율 진행 중입니다.", "에이전트가 응답 중입니다.",
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("missing mission-scoped active-work contract %q", expected)
		}
	}
	for _, removed := range []string{`id="turnStatus"`, `id="cancelTurnButton"`, "$(\"cancelTurnButton\")"} {
		if strings.Contains(index+script, removed) {
			t.Fatalf("conversation header pending-turn UI contract should be absent: %q", removed)
		}
	}
	for _, expected := range []string{
		"width: fit-content;", "max-width: 100%;", "margin: 0 0 8px;", "padding: 5px 6px 5px 8px;",
		"gap: 6px;", "border-radius: 8px;", "font-size: 12px;", "line-height: 1.3;",
		"flex: 0 1 auto;", "margin-left: 2px;", "min-height: 28px;", "padding: 4px 8px;",
		"@media (max-width: 560px)", ".active-work-notice { width: 100%; }", ".active-work-item { flex: 1 1 100%; }",
		".active-work-notice button { margin-left: 0; }",
	} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing compact active-work CSS contract %q", expected)
		}
	}
}

func TestRenderActiveWorkSynthesizesLocalPendingTurnOnce(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for the active-work DOM fixture")
	}
	script := string(mustReadStatic(t, "static/plasma/conversation_active_work.js"))
	domScript := string(mustReadStatic(t, "static/plasma/dom.js"))
	fixture := `
const nodes = {
  conversationActiveWork: {classList:{hidden:true,toggle(_name, value){this.hidden = value;}}, innerHTML:""},
  reportActiveWork: {classList:{hidden:true,toggle(_name, value){this.hidden = value;}}, innerHTML:""}
};
const $ = (id) => nodes[id] || null;
const state = {turnPending:true};
const escapeHTML = ` + jsFunctionSource(t, domScript, "escapeHTML") + `;
const escapeAttr = ` + jsFunctionSource(t, domScript, "escapeAttr") + `;
let appliedActiveWork = null;
function applyActiveWorkDescriptions(activeWork) { appliedActiveWork = activeWork; }
` + jsFunctionSource(t, script, "displayActiveWorkItems") + `
` + jsFunctionSource(t, script, "activeWorkMessage") + `
` + jsFunctionSource(t, script, "activeWorkActionHTML") + `
` + jsFunctionSource(t, script, "renderActiveWork") + `
const backend = {items:[], blocked_controls:[]};
renderActiveWork(backend);
const localHTML = nodes.conversationActiveWork.innerHTML;
if (nodes.conversationActiveWork.classList.hidden) throw new Error("local pending active work is hidden");
if ((localHTML.match(/에이전트가 응답 중입니다\./g) || []).length !== 1) throw new Error("local pending message did not appear exactly once: " + localHTML);
if ((localHTML.match(/data-active-work-action="cancel_turn"/g) || []).length !== 1) throw new Error("local pending cancel action did not appear exactly once: " + localHTML);
if (backend.items.length !== 0 || appliedActiveWork !== backend) throw new Error("synthetic active work mutated backend detail");
renderActiveWork({items:[{kind:"agent_turn", reason_code:"agent_turn_running", action:"cancel_turn"}], blocked_controls:[]});
const backendHTML = nodes.conversationActiveWork.innerHTML;
if ((backendHTML.match(/에이전트가 응답 중입니다\./g) || []).length !== 1) throw new Error("backend active work was duplicated: " + backendHTML);
if ((backendHTML.match(/data-active-work-action="cancel_turn"/g) || []).length !== 1) throw new Error("backend cancel action was duplicated: " + backendHTML);
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("active-work local pending fixture failed: %v: %s", err, out)
	}
}

func TestPlasmaConversationPublicOwnerMethods(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	fixture := `
const calls = [];
function createClassList() {
  return {values:new Set(), toggle(name, force){ if (force) this.values.add(name); else this.values.delete(name); }, add(...names){ names.forEach((name)=>this.values.add(name)); }, remove(...names){ names.forEach((name)=>this.values.delete(name)); }, contains(name){ return this.values.has(name); }};
}
function createNode(id, value = "") {
  let html = "";
  const node = {
    id, value, disabled:false, textContent:"", title:"", attributes:{}, classList:createClassList(),
    scrollHeight:200, scrollTop:0, clientHeight:100, dataset:{},
    setAttribute(name, value){ this.attributes[name] = String(value); },
    removeAttribute(name){ delete this.attributes[name]; },
    insertAdjacentHTML(_where, markup){ this.innerHTML += markup; },
    scrollTo(opts){ this.scrollTop = typeof opts === "number" ? opts : opts.top; },
    addEventListener(){}, removeEventListener(){},
    getBoundingClientRect(){ return {top:0}; },
    querySelectorAll(selector){ return selector === ".turn" ? [{getBoundingClientRect(){return {top:0};}}, {getBoundingClientRect(){return {top:80};}}] : []; }
  };
  Object.defineProperty(node, "innerHTML", {
    get(){ return html; },
    set(value){
      html = String(value);
      if (id === "agentExecutor") {
        this.options = [...html.matchAll(/<option value="([^"]*)"([^>]*)>/g)].map((match) => ({value:match[1], disabled:match[2].includes("disabled")}));
      }
    }
  });
  node.innerHTML = "";
  return node;
}
const nodes = {
  turnText:createNode("turnText", "질문"),
  agentExecutor:createNode("agentExecutor", "codex"),
  agentModel:createNode("agentModel", ""),
  agentReasoningEffort:createNode("agentReasoningEffort", "medium"),
  mcpMode:createNode("mcpMode", "auto"),
  controllerStrategy:createNode("controllerStrategy", "auto"),
  resetAgentSessionButton:createNode("resetAgentSessionButton"),
  sendTurnButton:createNode("sendTurnButton"),
  conversationActiveWork:createNode("conversationActiveWork"),
  reportActiveWork:createNode("reportActiveWork"),
  turnLog:createNode("turnLog"),
  turnNav:createNode("turnNav"),
  agentSessionStatus:createNode("agentSessionStatus"),
  agentControlsSummaryText:createNode("agentControlsSummaryText")
};
const state = {
  missionId:"mis_1", selectionGeneration:1, detailGeneration:1,
  detail:{
    active_work:{items:[], blocked_controls:[]},
    locked_agent_executor:"",
    agent_executors:[
      {name:"codex", label:"Codex", configured:true, default_model:"gpt-default", default_model_label:"GPT Default", default_reasoning_effort:"medium", models:[{name:"gpt-default", label:"GPT Default", reasoning_efforts:["low","medium"]}]},
      {name:"claude", label:"Claude", configured:true, reasoning_effort_supported:false}
    ],
    events:[{EventType:"turn.agent.response", Payload:{kind:"agent_response", user_event_id:"user-1", agent_executor:"codex", agent_session_id:"sess_1", agent_model:"gpt-default", agent_reasoning_effort:"medium"}}]
  },
  turnPending:false, pendingTurn:null, turnScrollMission:"mis_1",
  agentModelTouched:false, agentReasoningEffortTouched:false
};
const window = {Plasma:{
  state,
  dom:{
    $: (id) => nodes[id] || null,
    escapeHTML: (value) => String(value).replaceAll("&","&amp;").replaceAll("<","&lt;").replaceAll(">","&gt;").replaceAll('"',"&quot;"),
    escapeAttr: (value) => String(value).replaceAll("&","&amp;").replaceAll('"',"&quot;").replaceAll("<","&lt;").replaceAll(">","&gt;"),
    shortID: (value) => String(value).slice(0, 8),
    timeShort: () => "12:00"
  },
  ui:{
    setElementDisabled(id, disabled){ if (nodes[id]) nodes[id].disabled = Boolean(disabled); },
    setButtonText(id, text){ if (nodes[id]) nodes[id].textContent = text; },
    empty: (value) => "<p>" + value + "</p>"
  },
  mission:{
    captureMissionSelection: () => ({missionId:state.missionId, selectionGeneration:state.selectionGeneration}),
    ownsMissionSelection: (owner) => owner.missionId === state.missionId && owner.selectionGeneration === state.selectionGeneration,
    refreshSelectedMissionDetail: async () => { calls.push(["reload"]); },
    isStaleMissionOperation: () => false
  },
  transport:{
    missionApi: async (_owner, path, options) => {
      calls.push(["missionApi", path, options]);
      if (path === "/turns") throw new Error("send failed");
      return {};
    }
  }
}};
window.confirm = () => true;
window.renderPlasmaMath = () => {};
window.renderPlasmaMermaid = () => {};
window.enhancePlasmaImageViewing = () => {};
` + mustReadPlasmaConversationScripts(t) + `
const conversation = window.Plasma.conversation;
let errors = 0;
conversation.configure({
  requireMission: () => true,
  reloadMission: async (missionId) => { calls.push(["reloadMission", missionId]); },
  showError: () => { errors += 1; },
  syncReportControls: () => { calls.push(["sync"]); },
  turnControlsBlocked: (busy) => Boolean(busy),
  agentControlsBlocked: () => state.turnPending || state.workflowPending || state.workflowGoalDraftPending || state.reportPending || !state.detail,
  renderReportModelSelection: () => { calls.push(["reportModelSelection"]); }
});
conversation.configureRendering({renderMarkdown:(value)=>String(value), empty:(value)=>"<p>" + value + "</p>"});
await conversation.sendTurn({preventDefault(){}});
if (errors !== 1 || state.turnPending || state.pendingTurn || nodes.turnText.value !== "질문") throw new Error("send failure did not recover");
await conversation.cancelTurn();
if (!calls.some((call) => call[0] === "missionApi" && call[1] === "/turns/cancel")) throw new Error("cancel request path changed");
conversation.renderAgentOptions(state.detail.agent_executors);
conversation.renderAgentModelOptions(state.detail.events);
conversation.renderAgentReasoningEffortOptions(state.detail.events);
conversation.onAgentExecutorChange();
state.workflowPending = true;
conversation.onAgentExecutorChange();
if (!nodes.agentReasoningEffort.disabled) throw new Error("workflow pending must block agent effort selection");
state.workflowPending = false;
state.reportPending = true;
conversation.onAgentExecutorChange();
if (!nodes.agentReasoningEffort.disabled) throw new Error("report pending must block agent effort selection");
state.reportPending = false;
conversation.onAgentExecutorChange();
conversation.renderAgentSessionStatus(state.detail.events);
conversation.renderAgentControlsSummary();
if (!nodes.agentControlsSummaryText.textContent.includes("codex") || !calls.some((call) => call[0] === "reportModelSelection")) throw new Error("agent public controls did not render through callbacks");
await conversation.resetAgentSession();
const reset = calls.find((call) => call[0] === "missionApi" && call[1] === "/agent_sessions/reset");
if (!reset || reset[2].body.agent_executor !== "codex") throw new Error("reset session request body changed");
conversation.renderActiveWork({items:[{kind:"agent_turn", reason_code:"agent_turn_running", action:"cancel_turn"}], blocked_controls:[{control:"turn_submit"}]});
if (!nodes.conversationActiveWork.innerHTML.includes("에이전트가 응답 중입니다.") || !nodes.turnText.attributes["aria-describedby"]) throw new Error("active-work public render changed");
conversation.renderTurns([
  {EventID:"user-2", EventType:"turn.user", CreatedAt:"now", Payload:{text:"사용자"}},
  {EventType:"turn.agent.response", CreatedAt:"now", Payload:{kind:"agent_response", text:"응답", agent_executor:"codex", agent_session_id:"sess_2"}}
]);
if (!nodes.turnLog.innerHTML.includes("사용자") || !nodes.turnLog.innerHTML.includes("응답")) throw new Error("turn render public method changed");
conversation.turnNavScroll("bottom");
if (nodes.turnLog.scrollTop !== nodes.turnLog.scrollHeight) throw new Error("turn navigation public method changed");
if (!conversation.completedUserEventIDs([{EventType:"turn.agent.response", Payload:{user_event_id:"u1"}}]).has("u1")) throw new Error("turn-state completed set missing");
if (!conversation.hasOpenPendingTurn([{EventType:"turn.agent.pending", Payload:{user_event_id:"u2"}}])) throw new Error("turn-state pending helper missing");
`
	if out, err := exec.Command("node", "--input-type=module", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("conversation public owner fixture failed: %v: %s", err, out)
	}
}

func TestPlasmaWorkflowPublicOwnerMethods(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	fixture := `
const calls = [];
function createClassList() { return {values:new Set(), toggle(name, force){ if (force) this.values.add(name); else this.values.delete(name); }}; }
function createNode(id, value = "") {
  return {
    id, value, disabled:false, textContent:"", innerHTML:"", title:"", classList:createClassList(), dataset:{},
    setAttribute(){}, removeAttribute(){},
    insertAdjacentHTML(_where, markup){ this.innerHTML += markup; }
  };
}
const nodes = {
  workflowInstruction:createNode("workflowInstruction", ""),
  workflowStepInstructionMode:createNode("workflowStepInstructionMode"),
  draftWorkflowGoalButton:createNode("draftWorkflowGoalButton"),
  workflowRunGoal:createNode("workflowRunGoal", ""),
  workflowStepInstruction:createNode("workflowStepInstruction", ""),
  workflowLayeredFields:createNode("workflowLayeredFields"),
  startWorkflowButton:createNode("startWorkflowButton"),
  stopWorkflowButton:createNode("stopWorkflowButton"),
  workflowStatusBadge:createNode("workflowStatusBadge"),
  workflowRunList:createNode("workflowRunList"),
  workflowRunCount:createNode("workflowRunCount"),
  agentExecutor:createNode("agentExecutor", "codex"),
  mcpMode:createNode("mcpMode", "auto"),
  turnText:createNode("turnText", "fallback turn")
};
const state = {
  missionId:"mis_1", selectionGeneration:1, detailGeneration:1,
  detail:{workflow_runs:[{workflow_run_id:"run_1", status:"paused", continuation_instruction:"continue", steps:[{status:"completed", decision:"continue"}]}]},
  workflowPending:false, workflowGoalDraftPending:false, workflowGoalDraftRaw:"", reportPending:false
};
const window = {Plasma:{
  state,
  dom:{
    $: (id) => nodes[id] || null,
    escapeHTML: (value) => String(value).replaceAll("&","&amp;").replaceAll("<","&lt;").replaceAll(">","&gt;").replaceAll('"',"&quot;"),
    escapeAttr: (value) => String(value).replaceAll("&","&amp;").replaceAll('"',"&quot;").replaceAll("<","&lt;").replaceAll(">","&gt;"),
    shortID: (value) => String(value).slice(0, 8)
  },
  ui:{
    setElementDisabled(id, disabled){ if (nodes[id]) nodes[id].disabled = Boolean(disabled); },
    setButtonText(id, text){ if (nodes[id]) nodes[id].textContent = text; }
  },
  mission:{
    captureMissionSelection: () => ({missionId:state.missionId, selectionGeneration:state.selectionGeneration}),
    ownsMissionSelection: (owner) => owner.missionId === state.missionId && owner.selectionGeneration === state.selectionGeneration,
    refreshSelectedMissionDetail: async () => {}
  },
  transport:{
    missionApi: async (_owner, path, options) => {
      calls.push(["missionApi", path, options]);
      if (path === "/workflows/goal_draft") return {workflow_goal_draft:{user_instruction_raw:options.body.user_instruction_raw, run_goal:"draft goal", step_instruction:"draft step"}};
      return {};
    }
  }
}};
` + mustReadPlasmaWorkflowScripts(t) + `
const workflow = window.Plasma.workflow;
let blockedContinue = false;
let errors = 0;
workflow.configure({
  requireMission: () => true,
  reloadMission: async (missionId) => { calls.push(["reloadMission", missionId]); },
  showError: () => { errors += 1; },
  syncReportControls: () => { calls.push(["sync"]); },
  workflowControlsBlocked: () => false,
  workflowContinueBlocked: () => blockedContinue,
  setFormsEnabled: () => { calls.push(["setFormsEnabled"]); }
});
if (workflow.workflowRawInputValue() !== "") throw new Error("conversation input leaked into workflow input");
await workflow.draftWorkflowGoal();
await workflow.startWorkflow();
if (errors !== 2) throw new Error("empty workflow input was not rejected");
if (calls.some((call) => call[0] === "missionApi" && (call[1] === "/workflows/goal_draft" || call[1] === "/workflows"))) throw new Error("conversation input started workflow work");
nodes.workflowInstruction.value = "raw";
await workflow.draftWorkflowGoal();
if (nodes.workflowRunGoal.value !== "draft goal" || nodes.workflowStepInstruction.value !== "draft step") throw new Error("goal draft public method changed");
nodes.workflowRunGoal.value = "goal";
nodes.workflowStepInstruction.value = "step";
state.workflowGoalDraftRaw = "";
await workflow.startWorkflow();
const start = calls.find((call) => call[0] === "missionApi" && call[1] === "/workflows" && !call[2].body.continue_from_workflow_run_id);
if (!start || start[2].body.user_instruction_raw !== "raw" || start[2].body.run_goal !== "goal" || start[2].body.instruction !== "step") throw new Error("workflow start body changed");
state.workflowPending = true;
state.detail.workflow_runs = [{workflow_run_id:"run_active", status:"running", steps:[]}];
workflow.renderWorkflowControls([{workflow_run_id:"run_active", status:"running", steps:[]}]);
await workflow.stopWorkflow();
if (!calls.some((call) => call[0] === "missionApi" && call[1] === "/workflows/run_active/stop")) throw new Error("workflow stop request changed");
state.workflowPending = false;
state.detail.workflow_runs = [{workflow_run_id:"run_1", status:"paused", continuation_instruction:"continue", steps:[{status:"completed", decision:"continue"}]}];
blockedContinue = true;
await workflow.continueWorkflowRun("run_1");
if (calls.some((call) => call[0] === "missionApi" && call[2]?.body?.continue_from_workflow_run_id === "run_1")) throw new Error("continue ignored cross-feature callback block");
workflow.renderWorkflowControls([{workflow_run_id:"run_1", status:"paused", continuation_instruction:"continue", steps:[]}]);
if (!nodes.workflowRunList.innerHTML.includes("data-continue-workflow-id=\"run_1\" disabled")) throw new Error("continue button did not use blocking callback");
blockedContinue = false;
await workflow.continueWorkflowRun("run_1");
const continued = calls.find((call) => call[0] === "missionApi" && call[2]?.body?.continue_from_workflow_run_id === "run_1");
if (!continued || continued[2].body.instruction !== "continue") throw new Error("workflow continue body changed");
`
	if out, err := exec.Command("node", "--input-type=module", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("workflow public owner fixture failed: %v: %s", err, out)
	}
}

func TestStaticMissionLifecycleContracts(t *testing.T) {
	combined := string(mustReadStatic(t, "static/index.html")) + mustReadPlasmaReportScripts(t) + mustReadAppCSSComposed(t)
	for _, expected := range []string{
		"includeArchivedMissions",
		"include_archived=true",
		"missionArchiveButton",
		"missionRestoreButton",
		"missionHardDeleteButton",
		"missionHardDeleteSettings",
		"hard_delete_preview",
		"hardDeleteMission",
		"confirmMissionHardDelete",
		"selectMissionAfterHardDelete",
		"missionLifecycleWriteBlocked",
		"missionLifecycleSettingsText",
		"changeMissionLifecycle",
		"confirmMissionLifecycleChange",
		"selectMissionAfterArchive",
		"missionLifecycleNotice",
		"보관된 미션 보기",
		"보관된 미션입니다",
		"mission-archived",
		`mission-archive-toggle input[type="checkbox"]`,
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("missing mission lifecycle contract %q", expected)
		}
	}
	banner := htmlSection(t, string(mustReadStatic(t, "static/index.html")), `class="panel mission-banner"`, `id="tabBar"`)
	for _, forbidden := range []string{"missionMetadataForm", "missionMetadataEdit", "missionArchiveButton", "missionRestoreButton"} {
		if strings.Contains(banner, forbidden) {
			t.Fatalf("mission banner must not expose mission management control %q", forbidden)
		}
	}
}

func TestStaticDesktopMissionRailCollapseContracts(t *testing.T) {
	html := string(mustReadStatic(t, "static/index.html"))
	script := mustReadPlasmaReportScripts(t)
	styles := mustReadAppCSSComposed(t)
	combined := html + script + styles
	for _, expected := range []string{
		"missionRailToggle",
		"mission-rail-toggle",
		"MISSION_RAIL_COLLAPSED_STORAGE_KEY",
		"plasma.missionRailCollapsed.v1",
		"initMissionRailToggle",
		"setMissionRailCollapsed",
		"readMissionRailCollapsed",
		"mission-rail-collapsed",
		"grid-template-columns: 18px minmax(360px, 1fr)",
		"right: -12px",
		"미션 목록 접기",
		"미션 목록 펼치기",
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("missing desktop mission rail collapse contract %q", expected)
		}
	}
	railButton := htmlSection(t, html, `id="missionRailToggle"`, `</button>`)
	for _, expected := range []string{`type="button"`, `aria-pressed="false"`, `aria-label="미션 목록 접기"`} {
		if !strings.Contains(railButton, expected) {
			t.Fatalf("mission rail toggle must be an accessible button; missing %q in %s", expected, railButton)
		}
	}
	mobileBlock := regexp.MustCompile(`(?s)@media \(max-width: 760px\)\s*\{(.+)$`).FindStringSubmatch(styles)
	if len(mobileBlock) != 2 {
		t.Fatal("missing mobile responsive block")
	}
	for _, expected := range []string{
		"body.mission-rail-collapsed .workspace",
		"grid-template-columns: 1fr",
		".mission-rail-toggle",
		"display: none",
		"body.mission-picker-open .rail",
	} {
		if !strings.Contains(mobileBlock[1], expected) {
			t.Fatalf("mobile block must preserve picker flow while hiding desktop rail toggle; missing %q", expected)
		}
	}
}

func TestStaticMissionListTitlesCanGrowItems(t *testing.T) {
	styles := mustReadAppCSSComposed(t)
	railMatch := regexp.MustCompile(`(?s)\.rail > \.panel\s*\{([^}]*)\}`).FindStringSubmatch(styles)
	if len(railMatch) != 2 {
		t.Fatal("missing mission rail panel grid rule")
	}
	if !strings.Contains(railMatch[1], "grid-template-rows: auto auto auto minmax(0, 1fr)") {
		t.Fatalf("mission rail must reserve a row for the archived toggle: %s", railMatch[1])
	}
	listMatch := regexp.MustCompile(`(?s)#missionList\s*\{([^}]*)\}`).FindStringSubmatch(styles)
	if len(listMatch) != 2 {
		t.Fatal("missing mission list grid rule")
	}
	listBlock := listMatch[1]
	for _, expected := range []string{
		"align-content: start",
		"align-items: start",
		"grid-auto-rows: max-content",
	} {
		if !strings.Contains(listBlock, expected) {
			t.Fatalf("mission list grid must size items from content; missing %q in %s", expected, listBlock)
		}
	}
	match := regexp.MustCompile(`(?s)#missionList \.item-title\s*\{([^}]*)\}`).FindStringSubmatch(styles)
	if len(match) != 2 {
		t.Fatal("missing mission list title override")
	}
	block := match[1]
	for _, expected := range []string{
		"display: block",
		"max-height: none",
		"overflow: visible",
		"overflow-wrap: anywhere",
		"-webkit-line-clamp: unset",
	} {
		if !strings.Contains(block, expected) {
			t.Fatalf("mission list titles must wrap and grow their item box; missing %q in %s", expected, block)
		}
	}
	if strings.Contains(block, "overflow: hidden") || strings.Contains(block, "max-height: 2.8em") {
		t.Fatalf("mission list title override must not clamp long titles: %s", block)
	}
	toggleMatch := regexp.MustCompile(`(?s)\.mission-archive-toggle\s*\{([^}]*)\}`).FindStringSubmatch(styles)
	if len(toggleMatch) != 2 {
		t.Fatal("missing mission archive toggle spacing rule")
	}
	toggleBlock := toggleMatch[1]
	for _, expected := range []string{
		"min-height: 24px",
		"margin: 10px 0 10px",
		"padding: 2px",
		"align-self: start",
	} {
		if !strings.Contains(toggleBlock, expected) {
			t.Fatalf("mission archive toggle must keep visible space between new mission and list; missing %q in %s", expected, toggleBlock)
		}
	}
}

func TestStaticImageViewerContracts(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	script := mustReadPlasmaReportScripts(t) + mustReadPlasmaConversationScripts(t)
	viewerScript := string(mustReadStatic(t, "static/plasma/reports_image_viewer.js")) +
		string(mustReadStatic(t, "static/plasma/reports_image_enhance.js")) +
		string(mustReadStatic(t, "static/plasma/reports_image_frame.js"))
	styles := string(mustReadStatic(t, "static/image_viewer.css"))
	for _, expected := range []string{
		`href="/static/image_viewer.css"`,
		`src="/static/plasma/reports_image_viewer.js"`,
	} {
		if !strings.Contains(index, expected) {
			t.Fatalf("missing image viewer asset include %q", expected)
		}
	}
	for _, expected := range []string{
		"window.Plasma?.reports?.enhanceImages?.(log)",
		"prepareHTMLPreview",
		"plasma-html-preview-frame",
		"reports.enhanceImages?.($(\"detailBody\"))",
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("missing app image viewer integration %q", expected)
		}
	}
	for _, expected := range []string{
		`const MESSAGE_TYPE = "plasma:image-viewer:open"`,
		"root.querySelectorAll(\"img\")",
		"mermaidSVGSelector()",
		"svg.outerHTML",
		"kind: \"svg\"",
		"legendFromMermaidSVG(svg)",
		"renderImageViewerLegend(details?.legend)",
		"legend: legendFromMermaidSVG(svg)",
		"legend:legend(svg)",
		"data-image-viewer-action=\"zoom-in\"",
		"data-image-viewer-action=\"zoom-out\"",
		"data-image-viewer-action=\"fit\"",
		"data-image-viewer-action=\"actual\"",
		"parent.postMessage(details(img),\"*\")",
		"frame.contentWindow === event.source",
		"prepareHTMLPreview: preparePlasmaHTMLPreview",
	} {
		if !strings.Contains(viewerScript, expected) {
			t.Fatalf("missing image viewer behavior contract %q", expected)
		}
	}
	for _, expected := range []string{
		".plasma-image-viewer-open",
		".plasma-image-viewer-target--svg",
		".image-viewer-modal",
		".image-viewer-stage",
		".image-viewer-legend",
		"overflow: auto",
		"max-width: none",
		"z-index: 45",
		"@media (hover: none)",
	} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing image viewer style contract %q", expected)
		}
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	fixture := `
const fs = require("fs"), vm = require("vm");
	const context = { window: { addEventListener() {} } };
	vm.createContext(context);
	vm.runInContext(fs.readFileSync("static/plasma/namespace.js", "utf8"), context);
	vm.runInContext(fs.readFileSync("static/plasma/reports.js", "utf8"), context);
	vm.runInContext(fs.readFileSync("static/plasma/reports_image_frame.js", "utf8"), context);
	const html = context.window.Plasma.reports.prepareHTMLPreview("<html><body><main><img src='chart.png' alt='chart'></main></body></html>");
	if (!html.includes("plasma-image-viewer-open") || !html.includes("parent.postMessage") || !html.includes("</script></body>")) process.exit(1);
	const fragment = context.window.Plasma.reports.prepareHTMLPreview("<img src='chart.png'>");
	if (!fragment.includes("plasma:image-viewer:open") || !fragment.includes("querySelectorAll(\"img\")")) process.exit(2);
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("image viewer fixture: %v: %s", err, out)
	}
	mermaidScript := string(mustReadStatic(t, "static/plasma/reports_mermaid.js"))
	if !strings.Contains(mermaidScript, "global.Plasma.reports.enhanceImages?.(output)") {
		t.Fatal("Mermaid renderer must enhance rendered SVGs after async render")
	}
}

func TestStaticBasicHTMLReportUsesArtifactPreviewURL(t *testing.T) {
	script := mustReadPlasmaReportScripts(t)
	htmlExportFn := jsSourceRange(t, script, "async function exportReportArtifactHTML(", "\nasync function exportReportArtifactDesignedHTML(")
	for _, expected := range []string{
		"missionArtifactPreviewURL",
		"openReportHTMLPreviewWindow",
		"navigateReportHTMLPreviewWindow",
		"body: { include_content: Boolean(options.download) }",
		"result.preview_url",
	} {
		if !strings.Contains(htmlExportFn, expected) {
			t.Fatalf("expected basic HTML export flow to contain %q", expected)
		}
	}
	for _, forbidden := range []string{
		"setReportPreviewLoading(key)",
		"applyReportPreview(key, \"html\"",
	} {
		if strings.Contains(htmlExportFn, forbidden) {
			t.Fatalf("basic HTML export should open an artifact preview URL, found %q", forbidden)
		}
	}
}

func TestMissionHardDeleteRequiresPreviewAndConfirmation(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	domScript := string(mustReadStatic(t, "static/plasma/dom.js"))
	hardDelete := jsSourceRange(t, script, "async function hardDeleteMission", "function confirmMissionHardDelete")
	confirm := jsFunctionSource(t, script, "confirmMissionHardDelete")
	impactLines := jsFunctionSource(t, script, "missionHardDeleteImpactLines")
	afterDelete := strings.Replace(jsFunctionSource(t, script, "selectMissionAfterHardDelete"), "function selectMissionAfterHardDelete", "async function selectMissionAfterHardDelete", 1)
	formatBytes := jsFunctionSource(t, domScript, "formatBytes")
	fixture := `
const state = {missionId:"mis_a",selectionGeneration:1,missionHardDeletePending:false,detail:{projection:{title:"삭제 대상",mission_id:"mis_a"}},missions:[]};
const requireMission = () => true;
const captureMissionSelection = () => ({missionId:state.missionId,selectionGeneration:state.selectionGeneration});
const ownsMissionSelection = (owner) => owner && owner.missionId === state.missionId && owner.selectionGeneration === state.selectionGeneration;
let calls = [], confirms = [], deleteBody = null;
const renderMissionLifecycleControls = () => calls.push("render:" + state.missionHardDeletePending);
const setFormsEnabled = () => calls.push("forms");
const showError = (err) => { throw err; };
const missionApi = async (owner, suffix, options = {}) => {
  calls.push((options.method || "GET") + ":" + suffix);
  if (suffix === "/hard_delete_preview") return {eligible:true,mission_id:"mis_a",title:"삭제 대상",impact:{raw_artifacts:1,raw_artifact_bytes:14,source_snapshots:1}};
  if (options.method === "DELETE") { deleteBody = options.body; return {deleted:true}; }
  throw new Error("unexpected missionApi call");
};
const refreshMissionList = async () => { calls.push("refresh"); state.missions = [{MissionID:"mis_a"},{MissionID:"mis_b"}]; };
const selectMission = async (missionID) => { calls.push("select:" + missionID); state.missionId = missionID; state.selectionGeneration += 1; };
const clearMissionSelection = () => { calls.push("clear"); state.missionId = ""; };
const mission = {call(name, arg){ if (name==="requireMission") return requireMission(); if (name==="showError") return showError(arg); if (name==="setFormsEnabled") return setFormsEnabled(arg); throw new Error(name); }, captureMissionSelection, ownsMissionSelection, renderMissionLifecycleControls, refreshMissionList, selectMission, clearMissionSelection, StaleMissionOperationError: class StaleMissionOperationError extends Error {}};
const window = {confirm(message) { confirms.push(message); return false; }};
` + formatBytes + `
` + impactLines + `
` + confirm + `
` + afterDelete + `
` + hardDelete + `
(async () => {
  await hardDeleteMission();
  if (calls.includes("DELETE:")) throw new Error("cancelled hard delete sent DELETE");
  if (!confirms[0].includes("복구할 수 없습니다") || !confirms[0].includes("원문 데이터 용량 14 B")) throw new Error("confirmation did not include irreversible impact");
  if (state.missionHardDeletePending) throw new Error("cancelled hard delete left pending state");
  calls = [];
  window.confirm = (message) => { confirms.push(message); return true; };
  await hardDeleteMission();
  if (calls.join("|") !== "render:true|GET:/hard_delete_preview|DELETE:|refresh|select:mis_b") throw new Error("unexpected hard delete flow: " + calls.join("|"));
  if (!deleteBody || deleteBody.confirm_mission_id !== "mis_a") throw new Error("DELETE confirmation body missing mission id");
  if (state.missionId !== "mis_b") throw new Error("hard delete did not select next mission");
})().catch((err) => { console.error(err); process.exit(1); });
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("mission hard delete fixture: %v: %s", err, out)
	}
}

func TestMissionLifecycleRequiresConfirmationBeforeRequest(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	change := strings.Replace(jsFunctionSource(t, script, "changeMissionLifecycle"), "function changeMissionLifecycle", "async function changeMissionLifecycle", 1)
	confirm := jsFunctionSource(t, script, "confirmMissionLifecycleChange")
	fixture := `
const state = {missionId:"mis_a",selectionGeneration:1,missionLifecyclePending:false,detail:{projection:{title:"테스트 미션",mission_id:"mis_a"}}};
const requireMission = () => true;
let confirms = [], apiCalls = 0, renders = 0;
const window = {confirm(message) { confirms.push(message); return false; }};
const captureMissionSelection = () => ({missionId:state.missionId,selectionGeneration:state.selectionGeneration});
const ownsMissionSelection = () => true;
const renderMissionLifecycleControls = () => { renders++; };
const missionApi = async () => { apiCalls++; };
const selectMission = async () => {};
const selectMissionAfterArchive = async () => {};
const setFormsEnabled = () => {};
const showError = (err) => { throw err; };
const mission = {call(name, arg){ if (name==="requireMission") return requireMission(); if (name==="showError") return showError(arg); if (name==="setFormsEnabled") return setFormsEnabled(arg); throw new Error(name); }, captureMissionSelection, ownsMissionSelection, renderMissionLifecycleControls, selectMission, StaleMissionOperationError: class StaleMissionOperationError extends Error {}};
` + confirm + `
` + change + `
(async () => {
  await changeMissionLifecycle("archive");
  if (apiCalls || renders || state.missionLifecyclePending) throw new Error("cancelled archive touched lifecycle state");
  if (confirms.length !== 1 || !confirms[0].includes("테스트 미션") || !confirms[0].includes("보관할까요")) throw new Error("archive confirmation message is missing context");
  window.confirm = (message) => { confirms.push(message); return false; };
  await changeMissionLifecycle("restore");
  if (apiCalls || renders || state.missionLifecyclePending) throw new Error("cancelled restore touched lifecycle state");
  if (!confirms[1].includes("복원할까요")) throw new Error("restore confirmation message is missing context");
})().catch((err) => { console.error(err); process.exit(1); });
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("mission lifecycle confirmation fixture: %v: %s", err, out)
	}
}

func TestArchiveMissionSelectsNextMission(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	change := strings.Replace(jsFunctionSource(t, script, "changeMissionLifecycle"), "function changeMissionLifecycle", "async function changeMissionLifecycle", 1)
	afterArchive := strings.Replace(jsFunctionSource(t, script, "selectMissionAfterArchive"), "function selectMissionAfterArchive", "async function selectMissionAfterArchive", 1)
	fixture := `
const state = {missionId:"mis_a",selectionGeneration:1,missionLifecyclePending:false,missions:[]};
const requireMission = () => true;
const confirmMissionLifecycleChange = () => true;
const captureMissionSelection = () => ({missionId:state.missionId,selectionGeneration:state.selectionGeneration});
const ownsMissionSelection = (owner) => owner && owner.missionId === state.missionId && owner.selectionGeneration === state.selectionGeneration;
let calls = [];
const renderMissionLifecycleControls = () => calls.push("render");
const missionApi = async (owner, suffix) => { calls.push(suffix); };
const refreshMissionList = async () => { state.missions = [{MissionID:"mis_a"},{MissionID:"mis_b"},{MissionID:"mis_c"}]; calls.push("refresh"); };
const selectMission = async (missionID) => { calls.push("select:"+missionID); state.missionId = missionID; state.selectionGeneration += 1; };
const clearMissionSelection = () => { throw new Error("unexpected clear"); };
const setFormsEnabled = () => calls.push("forms");
const showError = (err) => { throw err; };
const mission = {call(name, arg){ if (name==="requireMission") return requireMission(); if (name==="showError") return showError(arg); if (name==="setFormsEnabled") return setFormsEnabled(arg); throw new Error(name); }, captureMissionSelection, ownsMissionSelection, renderMissionLifecycleControls, refreshMissionList, selectMission, clearMissionSelection, StaleMissionOperationError: class StaleMissionOperationError extends Error {}};
` + afterArchive + `
` + change + `
(async () => {
  await changeMissionLifecycle("archive");
  if (calls.join("|") !== "render|/archive|refresh|select:mis_b") throw new Error("unexpected archive flow: " + calls.join("|"));
  if (state.missionLifecyclePending) throw new Error("archive left lifecycle pending");
})().catch((err) => { console.error(err); process.exit(1); });
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("archive next-mission fixture: %v: %s", err, out)
	}
}

func TestArchiveMissionClearsSelectionWhenNoOtherMissionExists(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	change := strings.Replace(jsFunctionSource(t, script, "changeMissionLifecycle"), "function changeMissionLifecycle", "async function changeMissionLifecycle", 1)
	afterArchive := strings.Replace(jsFunctionSource(t, script, "selectMissionAfterArchive"), "function selectMissionAfterArchive", "async function selectMissionAfterArchive", 1)
	fixture := `
const state = {missionId:"mis_a",selectionGeneration:1,missionLifecyclePending:false,missions:[]};
const requireMission = () => true;
const confirmMissionLifecycleChange = () => true;
const captureMissionSelection = () => ({missionId:state.missionId,selectionGeneration:state.selectionGeneration});
const ownsMissionSelection = (owner) => owner && owner.missionId === state.missionId && owner.selectionGeneration === state.selectionGeneration;
let calls = [];
const renderMissionLifecycleControls = () => calls.push("render");
const missionApi = async (owner, suffix) => { calls.push(suffix); };
const refreshMissionList = async () => { state.missions = [{MissionID:"mis_a"}]; calls.push("refresh"); };
const selectMission = async (missionID) => { calls.push("select:"+missionID); };
const clearMissionSelection = () => { calls.push("clear"); state.missionId = ""; state.selectionGeneration += 1; };
const setFormsEnabled = () => calls.push("forms");
const showError = (err) => { throw err; };
const mission = {call(name, arg){ if (name==="requireMission") return requireMission(); if (name==="showError") return showError(arg); if (name==="setFormsEnabled") return setFormsEnabled(arg); throw new Error(name); }, captureMissionSelection, ownsMissionSelection, renderMissionLifecycleControls, refreshMissionList, selectMission, clearMissionSelection, StaleMissionOperationError: class StaleMissionOperationError extends Error {}};
` + afterArchive + `
` + change + `
(async () => {
  await changeMissionLifecycle("archive");
  if (calls.join("|") !== "render|/archive|refresh|clear") throw new Error("unexpected archive clear flow: " + calls.join("|"));
  if (state.missionLifecyclePending || state.missionId) throw new Error("archive did not clear selection cleanly");
})().catch((err) => { console.error(err); process.exit(1); });
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("archive clear-selection fixture: %v: %s", err, out)
	}
}

func TestConversationExportStaticContracts(t *testing.T) {
	script := mustReadPlasmaReportScripts(t)
	for _, expected := range []string{
		"createConversationExport",
		"viewConversationExport",
		"conversationExportPayloads",
		`"/conversation_exports"`,
		`"conversation.exported"`,
		`"conversation_export_markdown"`,
		"data-conversation-export-create",
		"data-conversation-export-id",
		"대화내역 export",
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("missing conversation export static contract %q", expected)
		}
	}
}

func TestMissionActivityWatermarkHandlesMalformedStorageAndMarksAfterSelection(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	functions := []string{
		jsFunctionSource(t, script, "renderMissionActivity"),
		jsFunctionSource(t, script, "missionActivityIndicator"),
		jsFunctionSource(t, script, "missionActivitySeenSequence"),
		jsFunctionSource(t, script, "missionActivitySeenWatermarks"),
		jsFunctionSource(t, script, "markMissionActivitySeen"),
		jsFunctionSource(t, script, "pruneMissionActivitySeenWatermarks"),
	}
	fixture := `
const MISSION_ACTIVITY_SEEN_STORAGE_KEY = "plasma.missionActivitySeen.v1";
let stored = "malformed";
let quotaBlocked = false;
const localStorage = { getItem: () => stored, setItem: (_, value) => { if (quotaBlocked) throw new Error("quota"); stored = value; } };
` + strings.Join(functions, "\n") + `
const Plasma = {polling:{missionActivitySeenWatermarks, missionActivitySeenSequence}, dom:{escapeHTML:(value)=>String(value ?? "")}};
const mission = {MissionID:"mis_1", activity:{active_work:{items:[]}, latest_terminal_activity:{sequence:7,outcome:"failed"}}};
const failedIndicator = renderMissionActivity(mission);
if (!failedIndicator.includes("mission-activity-failed")) throw new Error("unseen failure is missing");
if (!failedIndicator.includes("sr-only") || failedIndicator.includes("role=\"status\"")) throw new Error("activity indicator accessibility semantics are wrong");
markMissionActivitySeen("mis_1", 7);
if (renderMissionActivity(mission) !== "") throw new Error("seen failure remains visible");
if (JSON.parse(stored).mis_1 !== 7) throw new Error("watermark was not saved");
if (!renderMissionActivity({...mission, activity:{active_work:{items:[{}]}}}).includes("mission-activity-running")) throw new Error("running state is missing");
if (!renderMissionActivity({MissionID:"mis_2", activity:{active_work:{items:[]}, latest_terminal_activity:{sequence:4,outcome:"completed"}}}).includes("mission-activity-completed")) throw new Error("unseen completion is missing");
stored = JSON.stringify({mis_1:7,mis_deleted:4});
pruneMissionActivitySeenWatermarks([{MissionID:"mis_1"}]);
if (JSON.stringify(JSON.parse(stored)) !== JSON.stringify({mis_1:7})) throw new Error("deleted mission watermark was retained");
quotaBlocked = true;
markMissionActivitySeen("mis_1", 8);
if (renderMissionActivity({...mission, activity:{latest_terminal_activity:{sequence:8,outcome:"failed"}}}) === "") throw new Error("quota failure hid the unseen result");
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("mission activity watermark fixture: %v: %s", err, out)
	}
}

func TestSelectMissionMarksActivitySeenOnlyAfterDetailLoad(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	missionScript := string(mustReadStatic(t, "static/plasma/mission.js"))
	pollingScript := string(mustReadStatic(t, "static/plasma/polling.js"))
	remember := jsFunctionSource(t, script, "rememberMissionID")
	source := strings.Replace(jsFunctionSource(t, script, "selectMission"), "function selectMission", "async function selectMission", 1)
	fixture := `
const MISSION_STORAGE_KEY = "plasma.activeMissionId";
const state = { missionId:"", selectionGeneration:0, detailGeneration:0, detail: null, missionActivityCursors: {} };
let shouldFail = false; let marked = []; let renderedFailure = 0;
let reportRedpenController = null;
const api = async () => { if (shouldFail) throw new Error("failed"); return {projection:{last_sequence:9},activity_cursor:{schema:"mission-activity/v1",sequence:9,server_id:"server-a"}}; };
const localStorage = { setItem() { throw new Error("quota"); } };
const markMissionActivitySeen = (...args) => marked.push(args);
const renderDetail = () => {}; const renderMissions = () => {};
const refreshMissionList = async () => {}; const loadConfluenceConnections = async () => {};
const loadConfluenceAccess = async () => {}; const renderMissionLoadFailed = () => { renderedFailure++; };
` + remember + `
const window = {Plasma:{state,transport:{api},sources:{loadConfluenceConnections,loadConfluenceAccess,resetConfluenceMissionUI(){},renderConfluenceControls(){},renderConfluenceResults(){}}}};
` + missionScript + `
` + pollingScript + `
const beginMissionSelection = window.Plasma.mission.beginMissionSelection;
const ownsDetailRequest = window.Plasma.mission.ownsDetailRequest;
const applyMissionDetail = window.Plasma.mission.applyMissionDetail;
	window.Plasma.mission.configure({api, rememberMissionID, markMissionActivitySeen, recordDetailActivityCursor:window.Plasma.polling.recordDetailActivityCursor, renderDetail, renderMissions, afterSelectionApplied: async (owner) => {
	  await loadConfluenceConnections("", owner);
	  if (!window.Plasma.mission.ownsDetailRequest(owner)) return;
	  await loadConfluenceAccess(owner);
	}});
window.Plasma.mission.refreshMissionList = refreshMissionList;
window.Plasma.mission.renderMissionLoadFailed = renderMissionLoadFailed;
const Plasma = window.Plasma;
const mission = window.Plasma.mission;
` + source + `
(async () => {
  await selectMission("mis_1");
  if (marked.length !== 1 || marked[0][0] !== "mis_1" || marked[0][1] !== 9) throw new Error("successful load was not marked");
  shouldFail = true;
  await selectMission("mis_2");
  if (marked.length !== 1 || renderedFailure !== 1) throw new Error("failed load changed watermark");
})().catch((err) => { console.error(err); process.exit(1); });
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("selection activity watermark fixture: %v: %s", err, out)
	}
}

func TestReloadMissionUsesFullSelectionRefresh(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	missionScript := string(mustReadStatic(t, "static/plasma/mission.js"))
	pollingScript := string(mustReadStatic(t, "static/plasma/polling.js"))
	asyncSource := func(name string) string {
		return strings.Replace(jsFunctionSource(t, script, name), "function "+name, "async function "+name, 1)
	}
	fixture := `
const MISSION_STORAGE_KEY = "plasma.activeMissionId";
const state = {missionId:"mis_1",selectionGeneration:1,detailGeneration:0,detail:null,missionActivityCursors:{}};
const requests=[]; let listRefreshes=0, connectionRefreshes=0, accessRefreshes=0;
const api=async(path)=>{requests.push(path); return {projection:{last_sequence:4},activity_cursor:{schema:"mission-activity/v1",sequence:4,server_id:"server-a"}};};
const beginMissionSelection=(missionId)=>({missionId,selectionGeneration:state.selectionGeneration,detailGeneration:++state.detailGeneration});
const ownsMissionSelection=(owner)=>owner.missionId===state.missionId && owner.selectionGeneration===state.selectionGeneration;
const ownsDetailRequest=(owner)=>ownsMissionSelection(owner) && owner.detailGeneration===state.detailGeneration;
const localStorage={setItem(){}}; const rememberMissionID=()=>{}; const markMissionActivitySeen=()=>{};
const renderDetail=()=>{}; const renderMissions=()=>{}; const renderMissionLoadFailed=()=>{throw new Error("detail load failed");};
const refreshMissionList=async()=>{listRefreshes++;}; const loadConfluenceConnections=async()=>{connectionRefreshes++;}; const loadConfluenceAccess=async()=>{accessRefreshes++;};
const window = {Plasma:{state,transport:{api},sources:{loadConfluenceConnections,loadConfluenceAccess,resetConfluenceMissionUI(){},renderConfluenceControls(){},renderConfluenceResults(){}}}};
` + missionScript + `
` + pollingScript + `
const applyMissionDetail = window.Plasma.mission.applyMissionDetail;
	window.Plasma.mission.configure({api, rememberMissionID, markMissionActivitySeen, recordDetailActivityCursor:window.Plasma.polling.recordDetailActivityCursor, renderDetail, renderMissions, afterSelectionApplied: async (owner) => {
	  await loadConfluenceConnections("", owner);
	  if (!window.Plasma.mission.ownsDetailRequest(owner)) return;
	  await loadConfluenceAccess(owner);
	}});
window.Plasma.mission.refreshMissionList = refreshMissionList;
window.Plasma.mission.renderMissionLoadFailed = renderMissionLoadFailed;
const Plasma = window.Plasma;
const mission = window.Plasma.mission;
` + asyncSource("selectMission") + `
` + asyncSource("reloadMission") + `
(async()=>{
  await reloadMission(); await Promise.resolve();
  if(requests.join()!=="/api/missions/mis_1" || listRefreshes!==1 || connectionRefreshes!==1 || accessRefreshes!==1) throw new Error("reload did not retain full selection refresh");
})().catch((err)=>{console.error(err);process.exit(1);});`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("reload selection refresh fixture: %v: %s", err, out)
	}
}

func TestMissionActivityPollSleepsWithoutObservedActiveWork(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	missionScript := string(mustReadStatic(t, "static/plasma/mission.js"))
	pollingScript := string(mustReadStatic(t, "static/plasma/polling.js"))
	fixture := `
let scheduled = 0; const callbacks = [];
const state = {missions:[],missionActivityPollTimer:0,missionActivityPollInFlight:false};
const window = {Plasma:{state,transport:{}},clearTimeout(){},setTimeout(callback){ scheduled++; callbacks.push(callback); return scheduled; }};
const document = {hidden:false};
` + missionScript + `
` + pollingScript + `
const scheduleMissionActivityPoll = window.Plasma.polling.scheduleMissionActivityPoll;
window.Plasma.polling.configure({renderMissions:()=>{}});
scheduleMissionActivityPoll();
if (scheduled !== 0 || state.missionActivityPollTimer !== 0) throw new Error("idle list scheduled a global activity poll");
state.missions = [{activity:{active_work:{items:[{}]}}}];
scheduleMissionActivityPoll();
if (scheduled !== 1 || state.missionActivityPollTimer !== 1) throw new Error("active work did not schedule a refresh");
const requests = [];
state.missions = [
  {MissionID:"mis_active",activity:{last_sequence:4,active_work:{items:[{}]}}},
  {MissionID:"mis_idle",activity:{last_sequence:8,active_work:{items:[]}}}
];
const api = async (path) => { requests.push(path); return {activity:{last_sequence:5,active_work:{items:[]},latest_terminal_activity:{sequence:5,outcome:"completed"}}}; };
let renders = 0; const renderMissions = () => { renders++; };
window.Plasma.polling.configure({api, renderMissions});
(async () => {
  await callbacks.shift()();
  if (requests.join() !== "/api/missions/mis_active/activity") throw new Error("poll refreshed the full list or an idle mission");
  if (state.missions[0].activity.last_sequence !== 5 || state.missions[1].activity.last_sequence !== 8 || renders !== 1) throw new Error("targeted activity refresh did not merge summaries correctly");
  if (scheduled !== 1 || state.missionActivityPollTimer !== 0) throw new Error("terminal activity scheduled another poll");
})().catch((err) => { console.error(err); process.exit(1); });
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("mission activity polling fixture: %v: %s", err, out)
	}
}

func TestWorkStartsRefreshMissionActivityList(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	missionScript := string(mustReadStatic(t, "static/plasma/mission.js"))
	pollingScript := string(mustReadStatic(t, "static/plasma/polling.js"))
	conversationScript := mustReadPlasmaConversationScripts(t)
	workflowScript := mustReadPlasmaWorkflowScripts(t)
	asyncSource := func(name string) string {
		return strings.Replace(jsFunctionSource(t, script, name), "function "+name, "async function "+name, 1)
	}
	fixture := `
const WORKFLOW_DEFAULT_MAX_STEPS = 20;
const WORKFLOW_DEFAULT_MAX_DURATION_MS = 0;
const MISSION_STORAGE_KEY = "plasma.activeMissionId";
const state = {missionId:"mis_1",selectionGeneration:1,detailGeneration:0,detail:{projection:{title:"Mission"}},missions:[],missionActivityCursors:{},turnPending:false,workflowPending:false,workflowGoalDraftPending:false,workflowGoalDraftRaw:"",reportPending:false,pendingTurn:null,missionActivityPollTimer:0,missionActivityPollInFlight:false};
const classList = {toggle(){}};
const nodes = {
  turnText:{value:"Question"}, agentExecutor:{value:"codex"}, mcpMode:{value:"auto"}, controllerStrategy:{value:"auto"},
  workflowInstruction:{value:"Research"}, workflowRunGoal:{value:""}, workflowStepInstruction:{value:""},
  reportAgentModel:{value:""}, reportAgentReasoningEffort:{value:""}, reportRigor:{value:"strict"},
  conversationActiveWork:{classList, innerHTML:""}, reportActiveWork:{classList, innerHTML:""},
  turnLog:{scrollHeight:0,scrollTop:0,clientHeight:0,innerHTML:""},
  turnNav:{classList}
};
Object.values(nodes).forEach((node) => { node.setAttribute = node.setAttribute || (()=>{}); node.removeAttribute = node.removeAttribute || (()=>{}); });
const $ = (id) => nodes[id] || {value:"", setAttribute(){}, removeAttribute(){}};
let detailLoads = 0, listLoads = 0, scheduled = 0; const started = [];
const api = async (path) => {
  if (path === "/api/missions") { listLoads++; return {missions:[{MissionID:"mis_1",activity:{last_sequence:listLoads,active_work:{items:[{}]}}}]}; }
  if (path === "/api/missions/mis_1") { detailLoads++; return {projection:{last_sequence:detailLoads,title:"Mission"},activity_cursor:{schema:"mission-activity/v1",sequence:detailLoads,server_id:"server-a"}}; }
  throw new Error("unexpected api path " + path);
};
const missionApi = async (_, path) => { started.push(path); return {pending_event:{}}; };
const beginMissionSelection = (missionId) => ({missionId,selectionGeneration:state.selectionGeneration,detailGeneration:++state.detailGeneration});
const captureMissionSelection = () => ({missionId:state.missionId,selectionGeneration:state.selectionGeneration});
const ownsMissionSelection = (owner) => owner.missionId === state.missionId && owner.selectionGeneration === state.selectionGeneration;
const ownsDetailRequest = () => true;
const requireMission = () => true;
const localStorage = {setItem(){}}; const rememberMissionID = () => {}; const markMissionActivitySeen = () => {};
const pruneMissionActivitySeenWatermarks = () => {};
const renderDetail = () => {}; const renderMissions = () => {}; const renderMissionLoadFailed = () => { throw new Error("detail load failed"); };
const loadConfluenceConnections = async () => {}; const loadConfluenceAccess = async () => {};
const setTurnBusy = () => {}; const syncReportControls = () => {}; const renderTurns = () => {}; const showError = (err) => { throw err; };
const workflowStepInstructionMode = () => "layered"; const workflowRawInputValue = () => "Research"; const setWorkflowBusy = () => {};
	const setReportBusy = (busy) => { state.reportPending = busy; }; const setReportNotice = () => {}; const reportPendingMessage = () => "pending";
	const selectedReportGenerationGuidance = () => "narrative-contract";
	const reports = {modelSelection:{payload:() => ({agent_model:"",agent_reasoning_effort:""})}, direction:{current:()=>"", clear(){}}, selectedReportGenerationGuidance, setReportBusy, setReportNotice, reportPendingMessage};
const currentReportDirectionHint = () => ""; const clearAcceptedReportDirectionHint = () => {};
const missionListPath = () => "/api/missions";
const document = {hidden:false}; const window = {Plasma:{state,transport:{api},sources:{loadConfluenceConnections,loadConfluenceAccess,resetConfluenceMissionUI(){},renderConfluenceControls(){},renderConfluenceResults(){}}},clearTimeout(){},setTimeout(){ scheduled++; return scheduled; }};
window.Plasma.dom = {$, escapeHTML:(v)=>String(v ?? ""), escapeAttr:(v)=>String(v ?? ""), shortID:(v)=>String(v ?? ""), timeShort:()=>""};
window.Plasma.ui = {setElementDisabled(){}, setButtonText(){}, empty:(value)=>String(value)};
` + missionScript + `
` + pollingScript + `
window.Plasma.transport.missionApi = missionApi;
` + conversationScript + `
` + workflowScript + `
const applyMissionDetail = window.Plasma.mission.applyMissionDetail;
const refreshSelectedMissionDetail = window.Plasma.mission.refreshSelectedMissionDetail;
const scheduleMissionActivityPoll = window.Plasma.polling.scheduleMissionActivityPoll;
window.Plasma.mission.configure({api, rememberMissionID, markMissionActivitySeen, recordDetailActivityCursor:window.Plasma.polling.recordDetailActivityCursor, renderDetail, renderMissions});
window.Plasma.polling.configure({api, refreshSelectedMissionDetail, renderMissions});
window.Plasma.conversation.configure({requireMission, reloadMission, showError, syncReportControls});
window.Plasma.conversation.configureRendering({renderMarkdown:(value)=>String(value), empty:(value)=>String(value)});
window.Plasma.workflow.configure({requireMission, reloadMission, showError, syncReportControls, setFormsEnabled:()=>{}});
const sendTurn = window.Plasma.conversation.sendTurn;
const startWorkflow = window.Plasma.workflow.startWorkflow;
window.Plasma.mission.refreshMissionList = refreshMissionList;
window.Plasma.mission.renderMissionLoadFailed = renderMissionLoadFailed;
const Plasma = window.Plasma;
const mission = window.Plasma.mission;
` + asyncSource("refreshMissionList") + `
` + asyncSource("selectMission") + `
` + asyncSource("reloadMission") + `
	` + asyncSource("draftReport") + `
async function assertWorkStart(name, run, expectedPath) {
  detailLoads = 0; listLoads = 0; scheduled = 0; started.length = 0; state.missions = []; state.missionActivityPollTimer = 0;
  state.turnPending = false; state.workflowPending = false; state.workflowGoalDraftPending = false; state.reportPending = false; state.pendingTurn = null;
  await run();
  await Promise.resolve(); await Promise.resolve();
  if (started.join() !== expectedPath) throw new Error(name + " did not start expected work: " + started.join());
  if (detailLoads !== 1) throw new Error(name + " did not reload mission detail");
  if (listLoads !== 1) throw new Error(name + " did not refresh mission list activity");
  if (scheduled !== 0 || state.missionActivityPollTimer) throw new Error(name + " scheduled redundant global activity polling");
}
(async () => {
  await assertWorkStart("sendTurn", () => sendTurn({preventDefault(){}}), "/turns");
  await assertWorkStart("startWorkflow", startWorkflow, "/workflows");
  await assertWorkStart("draftReport", () => draftReport("planned"), "/reports");
})().catch((err) => { console.error(err); process.exit(1); });
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("work-start mission activity fixture: %v: %s", err, out)
	}
}

func TestResetMissionTransientStateClearsPreviousMissionWork(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for the mission-switch state fixture")
	}
	script := mustReadPlasmaReportScripts(t)
	source := jsFunctionSource(t, script, "resetMissionTransientState")
	fixture := `
let pollCleared = false;
let notice = "stale report";
const state = {
  detail:{}, turnPending:true, reportPending:true, workflowPending:true,
  workflowGoalDraftPending:true, workflowGoalDraftRaw:"old", pendingTurn:{},
  sourceCandidateBusy:new Set(["source"]), selectedSourceCandidates:new Set(["candidate"]),
  selectedProposals:new Set(["proposal"]), selectedReportKey:"report", reportPreview:{}
};
const clearPendingPoll = () => { pollCleared = true; };
const setReportNotice = (value) => { notice = value; };
const renderActiveWork = () => {};
const setFormsEnabled = () => {};
const renderMissionLifecycleControls = () => {};
const renderMissionLoading = () => {};
const resetConfluenceMissionUI = () => {};
const hideDetail = () => {};
const empty = () => "";
const $ = () => null;
		const window = {Plasma:{reports:{setReportNotice}, ui:{hideDetail}, sources:{resetConfluenceMissionUI}}};
const Plasma = window.Plasma;
Plasma.polling = {clearPendingPoll};
Plasma.conversation = {renderActiveWork};
Plasma.mission = {call(name, arg){ if (name==="setFormsEnabled") return setFormsEnabled(arg); throw new Error(name); }, renderMissionLifecycleControls, renderMissionLoading};
Plasma.ui.empty = empty;
function hideBulkBars() {}
function clearDetailLists() {}
	` + source + `
resetMissionTransientState();
if (!pollCleared || notice || state.detail || state.turnPending || state.reportPending || state.workflowPending || state.workflowGoalDraftPending || state.workflowGoalDraftRaw || state.pendingTurn || state.sourceCandidateBusy.size || state.selectedSourceCandidates.size || state.selectedProposals.size || state.selectedReportKey || state.reportPreview) process.exit(1);
`
	command := exec.Command("node", "-e", fixture)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("mission-switch state fixture failed: %v: %s", err, output)
	}
}

func TestResetMissionTransientStateHidesBulkBarsImmediately(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	source := jsFunctionSource(t, mustReadPlasmaReportScripts(t), "resetMissionTransientState")
	fixture := `
const state={detail:{},turnPending:true,reportPending:true,workflowPending:true,workflowGoalDraftPending:false,workflowGoalDraftRaw:"",pendingTurn:null,sourceCandidateBusy:new Set(),selectedSourceCandidates:new Set(["a"]),selectedProposals:new Set(["p"]),selectedReportKey:"",reportPreview:null,confluenceSearchResults:[],confluenceSearchContext:null,confluenceSpaces:[],confluencePages:[],confluenceBrowseContext:null,confluencePreview:null,confluenceUpdatePreview:null,confluenceAccess:null,confluenceBusy:true,confluenceOAuthURL:""};
const nodes={sourceCandidateBulk:{classList:{hidden:false,add(v){this.hidden=v}}},proposalBulk:{classList:{hidden:false,add(v){this.hidden=v}}},sourceCandidateBulkCount:{textContent:"7"},proposalBulkCount:{textContent:"8"}};
const $=(id)=>nodes[id]||null; const clearPendingPoll=()=>{}; const setReportNotice=()=>{}; const renderActiveWork=()=>{}; const setFormsEnabled=()=>{}; const renderMissionLifecycleControls=()=>{}; const hideDetail=()=>{}; const empty=()=>""; const renderMissionLoading=()=>{}; const resetConfluenceMissionUI=()=>{};
		const window = {Plasma:{reports:{setReportNotice}, ui:{hideDetail}, sources:{resetConfluenceMissionUI}}};
const Plasma = window.Plasma;
Plasma.polling = {clearPendingPoll};
Plasma.conversation = {renderActiveWork};
Plasma.mission = {call(name, arg){ if (name==="setFormsEnabled") return setFormsEnabled(arg); throw new Error(name); }, renderMissionLifecycleControls, renderMissionLoading};
Plasma.ui.empty = empty;
function hideBulkBars() {
  for (const id of ["sourceCandidateBulk", "proposalBulk"]) { const el = $(id); if (el) el.classList.add("hidden"); }
  for (const id of ["sourceCandidateBulkCount", "proposalBulkCount"]) { const el = $(id); if (el) el.textContent = "0"; }
}
function clearDetailLists() {}
	` + source + `
resetMissionTransientState(); if(!nodes.sourceCandidateBulk.classList.hidden||!nodes.proposalBulk.classList.hidden||nodes.sourceCandidateBulkCount.textContent!=="0"||nodes.proposalBulkCount.textContent!=="0")process.exit(1);
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("bulk reset fixture: %v: %s", err, out)
	}
}

func TestCreateMissionHonorsSelectionOwnership(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	create := strings.Replace(jsFunctionSource(t, script, "createMission"), "function createMission", "async function createMission", 1)
	refresh := strings.Replace(jsFunctionSource(t, script, "refreshMissionList"), "function refreshMissionList", "async function refreshMissionList", 1)
	fixture := `
const state={missionId:"mis_a",selectionGeneration:1,missions:[]};
const nodes={missionTitle:{value:"New"},missionObjective:{value:"Goal"}}; const $=(id)=>nodes[id];
let createResolve,listResolve; let selected=[]; let renders=0; let errors=0;
const api=(path)=>path==="/api/missions"?(createResolve?new Promise(r=>{listResolve=r}):new Promise(r=>{createResolve=r})):Promise.reject(new Error("unexpected"));
const captureMissionSelection=()=>({missionId:state.missionId,selectionGeneration:state.selectionGeneration});
const ownsMissionSelection=(o)=>o.missionId===state.missionId&&o.selectionGeneration===state.selectionGeneration;
class StaleMissionOperationError extends Error{}; const renderMissions=()=>{renders++}; const pruneMissionActivitySeenWatermarks=()=>{}; const scheduleMissionActivityPoll=()=>{}; const selectMission=async(id)=>{selected.push(id)}; const showError=()=>{errors++}; const missionListPath=()=>"/api/missions";
const mission = {captureMissionSelection, ownsMissionSelection, StaleMissionOperationError, renderMissions, selectMission};
const Plasma = {polling:{pruneMissionActivitySeenWatermarks, scheduleMissionActivityPoll}};
` + create + `
` + refresh + `
(async()=>{
  const run=createMission({preventDefault(){}}); createResolve({projection:{mission_id:"mis_new"}}); await Promise.resolve(); await Promise.resolve(); listResolve({missions:[{MissionID:"mis_new"}]}); await run;
  if(selected.join()!=="mis_new"||state.missions.length!==1||renders!==1||errors!==0)throw new Error("owned create did not select and refresh");
  state.missionId="mis_a"; state.selectionGeneration=3; nodes.missionTitle.value="Late"; selected=[]; let lateResolve; createResolve=undefined; listResolve=undefined;
  const late=createMission({preventDefault(){}}); lateResolve=createResolve; state.missionId="mis_c"; state.selectionGeneration=4; lateResolve({projection:{mission_id:"mis_late"}}); await late;
  if(selected.length||nodes.missionTitle.value!=="Late"||errors)throw new Error("stale create mutated current selection");
})().catch(e=>{console.error(e);process.exit(1)});
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("create mission ownership fixture: %v: %s", err, out)
	}
}

func TestReportVersionResponsesCannotCrossMissionSelection(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	exportFn := jsSourceRange(t, script, "async function exportReport(", "\nasync function viewReportArtifact(")
	astFn := jsSourceRange(t, script, "async function viewReportAST(", "\n\n  Object.assign(reports")
	fixture := `
const state={missionId:"mis_a",selectionGeneration:1,reportPreview:null};
const captureMissionSelection=()=>({missionId:state.missionId,selectionGeneration:state.selectionGeneration}); const ownsMissionSelection=(o)=>o.missionId===state.missionId&&o.selectionGeneration===state.selectionGeneration;
let pending=[]; const api=()=>new Promise((resolve,reject)=>pending.push({resolve,reject})); let mutations=[];
	const setReportPreviewLoading=()=>{}; const assertReportExportMatches=()=>{}; const downloadText=()=>mutations.push("download"); const applyReportPreview=()=>mutations.push("preview"); const reloadMission=()=>mutations.push("reload"); const clearReportPreview=()=>mutations.push("clear"); const showError=()=>mutations.push("error");
	const reportExportPreviewHeader=()=>"";
	const reports={setReportPreviewLoading,assertReportExportMatches,downloadReportExport:()=>mutations.push("download"),applyReportPreview,reportExportPreviewHeader,clearReportPreview};
` + exportFn + `
` + astFn + `
(async()=>{
  let run=exportReport("ver_1","markdown",{}); state.missionId="mis_b"; state.selectionGeneration=2; pending.shift().resolve({content:"A"}); await run;
  state.missionId="mis_a"; state.selectionGeneration=3; run=exportReport("ver_1","markdown",{}); state.missionId="mis_b"; state.selectionGeneration=4; pending.shift().reject(new Error("A failed")); await run;
  state.missionId="mis_a"; state.selectionGeneration=5; run=viewReportAST("ver_1"); state.missionId="mis_b"; state.selectionGeneration=6; pending.shift().resolve({old:true}); await run;
  state.missionId="mis_a"; state.selectionGeneration=7; run=viewReportAST("ver_1"); state.missionId="mis_b"; state.selectionGeneration=8; pending.shift().reject(new Error("A failed")); await run;
  if(mutations.length)throw new Error("stale report response mutated current mission: "+mutations.join(","));
})().catch(e=>{console.error(e);process.exit(1)});
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("report response ownership fixture: %v: %s", err, out)
	}
}

func TestPendingPollOwnershipSurvivesMissionSwitch(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	missionScript := string(mustReadStatic(t, "static/plasma/mission.js"))
	pollingScript := string(mustReadStatic(t, "static/plasma/polling.js"))
	fixture := `
const state={missionId:"mis_a",selectionGeneration:1,detailGeneration:1,detail:{activity_cursor:{schema:"mission-activity/v1",sequence:1,server_id:"server-a"}},missions:[],missionActivityCursors:{mis_a:{schema:"mission-activity/v1",sequence:1,serverID:"server-a"},mis_b:{schema:"mission-activity/v1",sequence:1,serverID:"server-a"}},turnPending:true,reportPending:false,workflowPending:false,pollTimer:0,pollInFlight:false,pollOwner:null};
let callbacks=[]; const window={Plasma:{state,transport:{}},setTimeout:(fn)=>{callbacks.push(fn);return callbacks.length},clearTimeout(){}}; const console={warn(){},error(){}};
let waits={}; const api=(path)=>new Promise(r=>{waits[path.includes("mis_b")?"mis_b":"mis_a"]=()=>r({activity:{last_sequence:1},cursor:{schema:"mission-activity/v1",sequence:1,server_id:"server-a"}})});
` + missionScript + `
` + pollingScript + `
const schedulePendingPoll = window.Plasma.polling.schedulePendingPoll;
window.Plasma.polling.configure({api, refreshSelectedMissionDetail:async()=>{}, renderMissions:()=>{}, onPendingPollFailure:()=>{}, onPendingPollSuccess:()=>{}});
(async()=>{
  schedulePendingPoll(); const runA=callbacks.shift()(); await Promise.resolve();
  state.missionId="mis_b"; state.selectionGeneration=2; state.detailGeneration=2; schedulePendingPoll(); const runB=callbacks.shift()(); await Promise.resolve();
  if(state.pollOwner.missionId!=="mis_b")throw new Error("B did not own poll");
  waits.mis_a(); await runA; if(!state.pollInFlight||state.pollOwner.missionId!=="mis_b")throw new Error("stale A cleared B poll");
  waits.mis_b(); await runB; if(state.pollInFlight||state.pollOwner!==null||callbacks.length!==1)throw new Error("B poll did not reschedule cleanly");
})().catch(e=>{console.error(e);process.exit(1)});
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("pending poll ownership fixture: %v: %s", err, out)
	}
}

func TestPendingPollFallbackFailureClearsAndReschedulesCurrentSelection(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	missionScript := string(mustReadStatic(t, "static/plasma/mission.js"))
	pollingScript := string(mustReadStatic(t, "static/plasma/polling.js"))
	fixture := `
const state={missionId:"mis_a",selectionGeneration:1,detailGeneration:1,detail:{activity_cursor:{schema:"mission-activity/v1",sequence:1,server_id:"server-a"}},missions:[],missionActivityCursors:{mis_a:{schema:"mission-activity/v1",sequence:1,serverID:"server-a"}},turnPending:true,reportPending:false,workflowPending:false,pollTimer:0,pollInFlight:false,pollOwner:null};
const callbacks=[]; const window={setTimeout:(fn)=>{callbacks.push(fn);return callbacks.length},clearTimeout(){}};
window.Plasma={state,transport:{}}; const nodes={healthBadge:{textContent:""}}; const console={warn(){},error(){}};
const api=async()=>({activity:{last_sequence:1},cursor:null});
const refreshSelectedMissionDetail=async()=>{state.detailGeneration++; throw new Error("fallback detail failed");};
` + missionScript + `
` + pollingScript + `
const schedulePendingPoll = window.Plasma.polling.schedulePendingPoll;
window.Plasma.polling.configure({api, refreshSelectedMissionDetail, renderMissions:()=>{}, onPendingPollFailure:()=>{nodes.healthBadge.textContent="재연결 중";}, onPendingPollSuccess:()=>{nodes.healthBadge.textContent="정상";}});
(async()=>{
  schedulePendingPoll();
  await callbacks.shift()();
  if(state.pollInFlight||state.pollOwner!==null) throw new Error("failed fallback left poll ownership stuck");
  if(nodes.healthBadge.textContent!=="재연결 중") throw new Error("failed current fallback did not show reconnecting");
  if(callbacks.length!==1||state.pollTimer!==1) throw new Error("failed fallback did not schedule exactly one retry");
})().catch((err)=>{console.error(err);process.exit(1);});`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("pending poll fallback failure fixture: %v: %s", err, out)
	}
}

func TestPendingPollOlderSameMissionCannotOverwriteNewerHealth(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	missionScript := string(mustReadStatic(t, "static/plasma/mission.js"))
	pollingScript := string(mustReadStatic(t, "static/plasma/polling.js"))
	fixture := `
const state={missionId:"mis_a",selectionGeneration:1,detailGeneration:1,detail:{activity_cursor:{schema:"mission-activity/v1",sequence:1,server_id:"server-a"}},missions:[],missionActivityCursors:{mis_a:{schema:"mission-activity/v1",sequence:1,serverID:"server-a"}},turnPending:true,reportPending:false,workflowPending:false,pollTimer:0,pollInFlight:false,pollOwner:null};
const callbacks=[]; const window={setTimeout:(fn)=>{callbacks.push(fn);return callbacks.length},clearTimeout(){}};
window.Plasma={state,transport:{}}; const nodes={healthBadge:{textContent:""}}; const console={warn(){},error(){}};
let resolveA; const api=(path)=>state.detailGeneration===1 ? new Promise(resolve=>{resolveA=()=>resolve({activity:{last_sequence:1},cursor:{schema:"mission-activity/v1",sequence:1,server_id:"server-a"}});}) : Promise.reject(new Error("B failed"));
` + missionScript + `
` + pollingScript + `
const schedulePendingPoll = window.Plasma.polling.schedulePendingPoll;
window.Plasma.polling.configure({api, refreshSelectedMissionDetail:async()=>{}, renderMissions:()=>{}, onPendingPollFailure:()=>{nodes.healthBadge.textContent="재연결 중";}, onPendingPollSuccess:()=>{nodes.healthBadge.textContent="정상";}});
(async()=>{
  schedulePendingPoll(); const runA=callbacks.shift()(); await Promise.resolve();
  // An ordinary same-mission reload completed while poll A was in flight.
  state.detailGeneration=2;
  schedulePendingPoll(); const runB=callbacks.shift()(); await runB;
  if(nodes.healthBadge.textContent!=="재연결 중"||state.pollInFlight||state.pollOwner!==null||callbacks.length!==1) throw new Error("B failure did not retain reconnecting state and one retry");
  resolveA(); await runA;
  if(nodes.healthBadge.textContent!=="재연결 중") throw new Error("stale A overwrote B health state");
  if(state.pollInFlight||state.pollOwner!==null||callbacks.length!==1) throw new Error("stale A overwrote B poll ownership");
})().catch((err)=>{console.error(err);process.exit(1);});`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("same-mission pending poll health fixture: %v: %s", err, out)
	}
}

func TestMissionSelectionGenerationPreservesSameMissionReloadAndRejectsStaleResponses(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for the selection generation fixture")
	}
	missionScript := string(mustReadStatic(t, "static/plasma/mission.js"))
	fixture := `
const state = {missionId:"", selectionGeneration:0, detailGeneration:0};
let resets = 0;
let loadingRenders = 0;
const resetMissionTransientState = () => { resets += 1; loadingRenders += 1; };
const window = {Plasma:{state}};
` + missionScript + `
const {beginMissionSelection, captureMissionSelection, ownsMissionSelection} = window.Plasma.mission;
window.Plasma.mission.configure({resetTransientState: resetMissionTransientState});
const a = beginMissionSelection("mis_a");
const b = beginMissionSelection("mis_b");
const refreshB = beginMissionSelection("mis_b");
if (resets !== 2 || loadingRenders !== 2 || !ownsMissionSelection(b) || !ownsMissionSelection(refreshB) || b.detailGeneration === refreshB.detailGeneration) process.exit(1);
const c = beginMissionSelection("mis_c");
if (resets !== 3 || loadingRenders !== 3 || ownsMissionSelection(refreshB) || !ownsMissionSelection(c) || captureMissionSelection().missionId !== "mis_c") process.exit(1);
`
	command := exec.Command("node", "-e", fixture)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("late success/error selection fixture failed: %v: %s", err, output)
	}
}

func TestStaticConfluenceLoadsUseMissionSelectionGuards(t *testing.T) {
	for _, name := range []string{"static/plasma/sources_confluence_core.js", "static/plasma/sources_confluence_access.js", "static/plasma/sources_confluence_browse.js"} {
		content := string(mustReadStatic(t, name))
		if !strings.Contains(content, "ownsMissionSelection(owner)") || !strings.Contains(content, "captureMissionSelection()") {
			t.Fatalf("%s must guard stale mission responses", name)
		}
	}
}

func TestBatchAMissionRoutesUseCapturedTransport(t *testing.T) {
	appScript := mustReadPlasmaReportScripts(t) + mustReadPlasmaSourceScripts(t)
	conversationSessionScript := string(mustReadStatic(t, "static/plasma/conversation_agent_session.js"))
	for _, name := range []string{"resetAgentSession", "addTextSource", "addUploadSource", "addMediaURLSource", "addPDFURLSource", "browseLocalPathTree", "attachLocalPathSource", "refreshSourcesOnly", "removeSource", "restoreSource", "readSource", "addURLSource", "viewReportArtifact", "downloadReportArtifact"} {
		source := appScript
		if name == "resetAgentSession" {
			source = conversationSessionScript
		}
		body := jsFunctionBody(t, source, name)
		if strings.Contains(body, "/api/missions/${state.missionId}") || !strings.Contains(body, "missionApi") && !strings.Contains(body, "missionFetch") && !strings.Contains(body, "fetchSourceChunk") {
			t.Fatalf("%s must not build a mutable mission URL", name)
		}
	}
	metadata := string(mustReadStatic(t, "static/mission_metadata.js"))
	if strings.Contains(metadata, "/api/missions/${encodeURIComponent(state.missionId)}") || !strings.Contains(metadata, "missionApi(owner") {
		t.Fatal("mission metadata must use captured mission transport")
	}
}

func TestStaticTreeHasNoMutableMissionRouteInterpolation(t *testing.T) {
	entries, err := os.ReadDir("static")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}
		content := string(mustReadStatic(t, "static/"+entry.Name()))
		if strings.Contains(content, "/api/missions/${state.missionId}") || strings.Contains(content, "/api/missions/${encodeURIComponent(state.missionId)}") {
			t.Fatalf("mutable mission route interpolation in %s", entry.Name())
		}
	}
}

func TestBulkSourceAcceptKeepsCapturedMissionOwner(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t) + mustReadPlasmaSourceScripts(t)
	addURL := strings.Replace(jsFunctionSource(t, script, "addURLSource"), "function addURLSource", "async function addURLSource", 1)
	bulk := strings.Replace(jsFunctionSource(t, script, "bulkSourceCandidateAction"), "function bulkSourceCandidateAction", "async function bulkSourceCandidateAction", 1)
	runSequential := strings.Replace(jsFunctionSource(t, script, "runBulkSequential"), "function runBulkSequential", "async function runBulkSequential", 1)
	source := addURL + "\n" + runSequential + "\n" + bulk
	fixture := `
const state = {missionId:"mis_a",selectionGeneration:1,sourceCandidateBusy:new Set(),selectedSourceCandidates:new Set(["https://a.example","https://b.example"])};
let requests = []; let resolveFirst;
const requireMission=()=>true; const captureMissionSelection=()=>({missionId:state.missionId,selectionGeneration:state.selectionGeneration});
const ownsMissionSelection=(o)=>o.missionId===state.missionId&&o.selectionGeneration===state.selectionGeneration;
class StaleMissionOperationError extends Error {}; const isStaleMissionOperation=(e)=>e instanceof StaleMissionOperationError;
const missionApi=(owner,path)=>{ requests.push(owner.missionId+path); return new Promise((resolve,reject)=>{ resolveFirst=()=>ownsMissionSelection(owner)?resolve({}):reject(new StaleMissionOperationError()); }); };
const normalizeSourceURL=(v)=>v; const refreshSourceCandidates=()=>{}; const sourceRouteForURL=()=>"url"; const looksLikePDFSourceError=()=>false; const sourceCandidateTitleForURL=()=>"";
const reloadMission=async()=>{}; const showError=()=>{ throw new Error("stale error shown") }; const window={prompt:()=>"",Plasma:{sources:{}}};
` + source + `
(async()=>{ const run=bulkSourceCandidateAction("approve"); await Promise.resolve(); state.missionId="mis_b"; state.selectionGeneration=2; resolveFirst(); await run; if(requests.length!==1||requests[0].indexOf("mis_a")<0||state.selectedSourceCandidates.size!==2) throw new Error("batch crossed mission boundary"); })().catch(e=>{console.error(e);process.exit(1)});
`
	cmd := exec.Command("node", "-e", fixture)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bulk owner fixture failed: %v: %s", err, output)
	}
}

func mustReadStatic(t *testing.T, name string) []byte {
	t.Helper()
	content, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

var appCSSImportManifest = []string{
	"/static/plasma/design_tokens_base.css",
	"/static/plasma/form_controls_buttons.css",
	"/static/plasma/shell_layout_topbar.css",
	"/static/plasma/panels_mission_tabs.css",
	"/static/plasma/conversation_workflow.css",
	"/static/plasma/sources_settings.css",
	"/static/plasma/lists_reports_badges.css",
	"/static/plasma/detail_copy_previews.css",
	"/static/plasma/report_toc.css",
	"/static/plasma/ledger_modal_utilities.css",
	"/static/plasma/responsive_overrides.css",
}

func mustReadAppCSSComposed(t *testing.T) string {
	t.Helper()
	imports := mustAppCSSImportPaths(t)
	var combined strings.Builder
	for _, href := range imports {
		if !strings.HasPrefix(href, "/static/plasma/") {
			t.Fatalf("app.css import must stay under /static/plasma: %s", href)
		}
		combined.Write(mustReadStatic(t, strings.TrimPrefix(href, "/")))
	}
	return combined.String()
}

func mustAppCSSImportPaths(t *testing.T) []string {
	t.Helper()
	appCSS := string(mustReadStatic(t, "static/app.css"))
	matches := regexp.MustCompile(`@import url\("([^"]+)"\);\n`).FindAllStringSubmatch(appCSS, -1)
	if len(matches) == 0 {
		t.Fatal("static/app.css must declare CSS imports")
	}
	imports := make([]string, 0, len(matches))
	for _, match := range matches {
		imports = append(imports, match[1])
	}
	return imports
}

func validateCSSSegmentBoundary(t *testing.T, href string, content string) {
	t.Helper()
	depth := 0
	inComment := false
	var quote byte
	escaped := false
	pendingPrelude := false
	for i := 0; i < len(content); i++ {
		ch := content[i]
		next := byte(0)
		if i+1 < len(content) {
			next = content[i+1]
		}
		if inComment {
			if ch == '*' && next == '/' {
				inComment = false
				i++
			}
			continue
		}
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '/' && next == '*' {
			inComment = true
			i++
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		switch ch {
		case '{':
			if depth == 0 && !pendingPrelude {
				t.Fatalf("%s opens a CSS block without a selector or at-rule prelude", href)
			}
			depth++
			pendingPrelude = false
		case '}':
			depth--
			if depth < 0 {
				t.Fatalf("%s closes a CSS block that was opened in another file", href)
			}
			if depth == 0 {
				pendingPrelude = false
			}
		case ';':
			if depth == 0 {
				pendingPrelude = false
			}
		default:
			if depth == 0 && !isCSSWhitespace(ch) {
				pendingPrelude = true
			}
		}
	}
	if inComment {
		t.Fatalf("%s ends inside a CSS comment", href)
	}
	if quote != 0 {
		t.Fatalf("%s ends inside a CSS string", href)
	}
	if depth != 0 {
		t.Fatalf("%s ends with CSS block depth %d", href, depth)
	}
	if pendingPrelude {
		t.Fatalf("%s ends after a selector or at-rule prelude before its block or semicolon", href)
	}
}

func isCSSWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t' || ch == '\f'
}

func mustIndexClassicScriptPaths(t *testing.T) []string {
	t.Helper()
	html := string(mustReadStatic(t, "static/index.html"))
	matches := regexp.MustCompile(`<script src="/static/([^"]+\.js)"></script>`).FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		t.Fatal("index.html does not declare classic scripts")
	}
	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		path := "static/" + match[1]
		if strings.Contains(path, "/vendor/") {
			continue
		}
		paths = append(paths, path)
	}
	if len(paths) == 0 || paths[len(paths)-1] != "static/app.js" {
		t.Fatalf("classic scripts must end with app.js, got %v", paths)
	}
	return paths
}

func mustReadPlasmaReportScripts(t *testing.T) string {
	t.Helper()
	files := []string{
		"static/plasma/namespace.js", "static/plasma/dom.js", "static/plasma/state.js",
		"static/plasma/ui.js", "static/plasma/ui_feedback.js", "static/plasma/ui_detail.js", "static/plasma/ui_tabs.js",
		"static/plasma/mission.js", "static/plasma/transport.js", "static/plasma/polling.js",
		"static/plasma/polling_activity.js", "static/plasma/mission_storage.js", "static/plasma/mission_rendering.js",
		"static/plasma/mission_state.js", "static/plasma/mission_actions.js", "static/plasma/mission_selection.js",
		"static/plasma/reports.js", "static/plasma/reports_constants.js", "static/plasma/reports_math.js",
		"static/plasma/reports_mermaid_legend.js", "static/plasma/reports_image_viewer.js",
		"static/plasma/reports_image_enhance.js", "static/plasma/reports_image_frame.js",
		"static/plasma/reports_mermaid.js", "static/plasma/reports_redpen_markdown.js",
		"static/plasma/reports_redpen.js", "static/plasma/reports_toc.js", "static/plasma/reports_redpen_init.js", "static/plasma/reports_markdown.js",
		"static/plasma/reports_model_selection.js", "static/plasma/reports_direction.js",
		"static/plasma/reports_state.js", "static/plasma/reports_view_state.js", "static/plasma/reports_trace.js",
		"static/plasma/reports_notice.js", "static/plasma/reports_modal.js", "static/plasma/reports_rendering.js",
		"static/plasma/reports_cards_conversation.js", "static/plasma/reports_cards_artifacts.js",
		"static/plasma/reports_cards_legacy.js", "static/plasma/reports_exports_core.js",
		"static/plasma/reports_exports_html.js", "static/plasma/reports_downloads.js",
		"static/plasma/reports_controls.js", "static/plasma/reports_pipeline.js",
		"static/plasma/reports_pipeline_core.js", "static/plasma/reports_pipeline_graph.js",
		"static/plasma/reports_pipeline_render.js", "static/plasma/reports_events.js",
		"static/plasma/knowledge.js", "static/plasma/knowledge_rendering.js", "static/plasma/knowledge_detail.js",
		"static/plasma/proposals.js", "static/plasma/proposals_selection.js", "static/plasma/proposals_rendering.js",
		"static/plasma/proposals_actions.js", "static/plasma/ledger.js", "static/mission_metadata.js",
		"static/plasma/bootstrap_modules.js", "static/plasma/bootstrap_extras.js", "static/plasma/bootstrap.js",
		"static/plasma/bootstrap_startup.js",
		"static/app.js",
	}
	var combined strings.Builder
	for _, file := range files {
		combined.WriteString("\n")
		combined.Write(mustReadStatic(t, file))
	}
	return combined.String()
}

func mustReadPlasmaUIScripts(t *testing.T) string {
	t.Helper()
	return string(mustReadStatic(t, "static/plasma/ui.js")) +
		"\n" + string(mustReadStatic(t, "static/plasma/ui_feedback.js")) +
		"\n" + string(mustReadStatic(t, "static/plasma/ui_detail.js")) +
		"\n" + string(mustReadStatic(t, "static/plasma/ui_tabs.js"))
}

func mustReadPlasmaConversationScripts(t *testing.T) string {
	t.Helper()
	return string(mustReadStatic(t, "static/plasma/conversation.js")) +
		"\n" + string(mustReadStatic(t, "static/plasma/conversation_agent_state.js")) +
		"\n" + string(mustReadStatic(t, "static/plasma/conversation_agent_models.js")) +
		"\n" + string(mustReadStatic(t, "static/plasma/conversation_agent_controls.js")) +
		"\n" + string(mustReadStatic(t, "static/plasma/conversation_agent_session.js")) +
		"\n" + string(mustReadStatic(t, "static/plasma/conversation_active_work.js")) +
		"\n" + string(mustReadStatic(t, "static/plasma/conversation_turn_state.js")) +
		"\n" + string(mustReadStatic(t, "static/plasma/conversation_live_turn.js")) +
		"\n" + string(mustReadStatic(t, "static/plasma/conversation_turn_nav.js")) +
		"\n" + string(mustReadStatic(t, "static/plasma/conversation_turns.js"))
}

func mustReadPlasmaWorkflowScripts(t *testing.T) string {
	t.Helper()
	return string(mustReadStatic(t, "static/plasma/workflow.js")) +
		"\n" + string(mustReadStatic(t, "static/plasma/workflow_input.js")) +
		"\n" + string(mustReadStatic(t, "static/plasma/workflow_rendering.js"))
}

func mustReadPlasmaSourceScripts(t *testing.T) string {
	t.Helper()
	entries, err := os.ReadDir("static/plasma")
	if err != nil {
		t.Fatal(err)
	}
	var combined strings.Builder
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "sources") || !strings.HasSuffix(name, ".js") {
			continue
		}
		combined.WriteString("\n")
		combined.Write(mustReadStatic(t, "static/plasma/"+name))
	}
	return combined.String()
}

func mustReadPlasmaSettingsScripts(t *testing.T) string {
	t.Helper()
	entries, err := os.ReadDir("static/plasma")
	if err != nil {
		t.Fatal(err)
	}
	var combined strings.Builder
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "settings") || !strings.HasSuffix(name, ".js") {
			continue
		}
		combined.WriteString("\n")
		combined.Write(mustReadStatic(t, "static/plasma/"+name))
	}
	return combined.String()
}

func TestStaticAppReportPlanDisplaysLongFormHierarchy(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for report plan display regression")
	}
	nodeScript := `
const fs = require("fs");
const vm = require("vm");
function makeNode(id) {
  const classes = new Set(["hidden"]);
  return {
    id,
    textContent: "",
    innerHTML: "",
    classList: {
      add(value){ classes.add(value); },
      remove(value){ classes.delete(value); },
      contains(value){ return classes.has(value); }
    }
  };
}
const nodes = {
  detailTitle: makeNode("detailTitle"),
  detailBody: makeNode("detailBody"),
  detailModal: makeNode("detailModal")
};
const context = {
  console, Date, Math, JSON, RegExp, Error, TypeError, Set, Map, Array, Object, String, Number, Boolean,
  document: { getElementById(id){ return nodes[id] || (nodes[id] = makeNode(id)); } }
};
context.window = context;
context.globalThis = context;
vm.createContext(context);
for (const file of [
  "static/plasma/namespace.js",
  "static/plasma/dom.js",
  "static/plasma/state.js",
  "static/plasma/reports.js",
  "static/plasma/reports_constants.js",
  "static/plasma/reports_state.js",
  "static/plasma/reports_view_state.js",
  "static/plasma/reports_trace.js",
  "static/plasma/reports_cards_artifacts.js"
]) {
  vm.runInContext(fs.readFileSync(file, "utf8"), context, {filename: file});
}
const Plasma = context.Plasma;
const reports = Plasma.reports;
function assert(condition, message) {
  if (!condition) throw new Error(message);
}
reports.reportGenerationSummaryHTML = () => "";
reports.reportSourceContextHTML = () => "";
reports.reportPreviewInlineHTML = () => "";
reports.reportActionMenu = (_label, body) => body;
Plasma.state.detail = {
  reports: [{report_id: "r_single", title: "Single"}, {report_id: "r_single_no_summary", title: "Single no summary"}, {report_id: "r_long", title: "Long"}, {report_id: "r_long_no_summary", title: "Long no summary"}],
  events: [
    {
      EventID: "evt_single_plan",
      EventType: "report.plan.created",
      Payload: {
        report_version_id: "ver_single",
        pending_event_id: "pending_single",
        plan: {
          summary: "Single summary",
          sections: [{title: "Single <Title>", purpose: "Purpose & one", target_refs: {claim_ids: ["clm_<one>"]}}]
        }
      }
    },
    {
      EventID: "evt_single_no_summary",
      EventType: "report.plan.created",
      Payload: {
        report_version_id: "ver_single_no_summary",
        pending_event_id: "pending_single_no_summary",
        artifact_id: "art_single_no_summary",
        plan: {
          sections: [
            {title: "Single no summary one", purpose: "First purpose", target_refs: {claim_ids: ["clm_1"]}},
            {title: "Single no summary two", purpose: "Second purpose", target_refs: {evidence_ids: ["evd_2"]}}
          ]
        }
      }
    },
    {
      EventID: "evt_long_plan",
      EventType: "report.plan.created",
      Payload: {
        report_version_id: "ver_long",
        pending_event_id: "pending_long",
        artifact_id: "art_long",
        plan: {
          summary: "Long summary",
          sections: [{title: "Wrong fallback section"}],
          parts: [
            {
              title: "Unsafe <script>Part</script>",
              purpose: "Part purpose & one",
              sections: [
                {title: "Section <One>", purpose: "First purpose", target_refs: {claim_ids: ["clm_<one>"]}},
                {title: "Section Two", purpose: "Second purpose", target_refs: {evidence_ids: ["evd_2"]}}
              ]
            },
            {
              title: "Part Two",
              purpose: "Part purpose two",
              sections: [{title: "Section Three", purpose: "Third purpose", target_refs: {question_ids: ["qst_3"]}}]
            },
            {
              title: "Empty <Part>",
              purpose: "No sections",
              sections: []
            }
          ]
        }
      }
    },
    {
      EventID: "evt_long_no_summary",
      EventType: "report.plan.created",
      Payload: {
        report_version_id: "ver_long_no_summary",
        pending_event_id: "pending_long_no_summary",
        artifact_id: "art_long_no_summary",
        plan: {
          parts: [
            {
              title: "No summary part one",
              purpose: "No summary purpose one",
              sections: [{title: "No summary section one", purpose: "No summary section purpose one", target_refs: {snapshot_ids: ["snap_1"]}}]
            },
            {
              title: "No summary part two",
              purpose: "No summary purpose two",
              sections: [{title: "No summary section two", purpose: "No summary section purpose two", target_refs: {option_ids: ["opt_2"]}}]
            }
          ]
        }
      }
    }
  ]
};
assert(reports.reportPlanLabel({sections: [{title: "One"}]}) === "1개 섹션", "summary-less single outline label changed");
assert(reports.reportPlanLabel({}) === "", "empty plan label should stay empty");
const singleVM = reports.reportViewModel({report_version_id: "ver_single", report_id: "r_single", state: "draft"}, 0);
const singleNoSummaryVM = reports.reportViewModel({report_version_id: "ver_single_no_summary", report_id: "r_single_no_summary", state: "draft"}, 0);
const longVM = reports.reportViewModel({report_version_id: "ver_long", report_id: "r_long", state: "draft"}, 0);
const longNoSummaryVM = reports.reportViewModel({report_version_id: "ver_long_no_summary", report_id: "r_long_no_summary", state: "draft"}, 0);
assert(singleVM.planLabel === "1개 섹션 / Single summary", "single plan card count changed: " + singleVM.planLabel);
assert(singleNoSummaryVM.planLabel === "2개 섹션", "summary-less single legacy label should contain only section count: " + singleNoSummaryVM.planLabel);
assert(longVM.planLabel === "3개 섹션 / Long summary", "long-form legacy card count did not use parts: " + longVM.planLabel);
assert(longNoSummaryVM.planLabel === "2개 섹션", "summary-less long-form legacy label should contain only section count: " + longNoSummaryVM.planLabel);
const artifactHTML = reports.renderArtifactReportSection([{key: "artifact:art_long", isLatest: true, payload: {artifact_id: "art_long", plan_event_id: "evt_long_plan", pending_event_id: "pending_long", report_mode: "long_form", title: "Artifact"}}], "");
assert(artifactHTML.includes("3개 섹션 / Long summary"), "artifact card count did not use parts");
const artifactSingleNoSummaryHTML = reports.renderArtifactReportSection([{key: "artifact:art_single_no_summary", isLatest: false, payload: {artifact_id: "art_single_no_summary", plan_event_id: "evt_single_no_summary", pending_event_id: "pending_single_no_summary", report_mode: "planned", title: "Single artifact no summary"}}], "");
assert(artifactSingleNoSummaryHTML.includes("<span>2개 섹션</span>") && !artifactSingleNoSummaryHTML.includes("2개 섹션 /") && !artifactSingleNoSummaryHTML.includes("기록된 생성 계획 없음"), "summary-less single artifact label should contain only section count");
const artifactNoSummaryHTML = reports.renderArtifactReportSection([{key: "artifact:art_long_no_summary", isLatest: true, payload: {artifact_id: "art_long_no_summary", plan_event_id: "evt_long_no_summary", pending_event_id: "pending_long_no_summary", report_mode: "long_form", title: "Artifact no summary"}}], "");
assert(artifactNoSummaryHTML.includes("<span>2개 섹션</span>") && !artifactNoSummaryHTML.includes("2개 섹션 /") && !artifactNoSummaryHTML.includes("기록된 생성 계획 없음"), "summary-less artifact label should contain only section count");
const oneTakeHTML = reports.renderArtifactReportSection([{key: "artifact:art_one_take", isLatest: false, payload: {artifact_id: "art_one_take", report_mode: "one_take", title: "One take"}}], "");
assert(oneTakeHTML.includes("원테이크 생성: 별도 계획 없음"), "one_take artifact fallback changed");
reports.showReportPlanEvent("evt_single_plan");
const singleHTML = nodes.detailBody.innerHTML;
assert(singleHTML.includes("Single &lt;Title&gt;") && singleHTML.includes("Purpose &amp; one"), "single plan detail lost escaped section content");
assert(!singleHTML.includes("Part 1") && !singleHTML.includes("Section 1.1"), "single plan detail should keep existing non-hierarchical section list");
reports.showReportPlanEvent("evt_long_plan");
const longHTML = nodes.detailBody.innerHTML;
for (const expected of ["Part 1", "Part 2", "Part 3", "Section 1.1", "Section 1.2", "Section 2.1", "0개 Section", "Section 계획 없음", "clm_&lt;one&gt;", "qst_3"]) {
  assert(longHTML.includes(expected), "long-form hierarchy missing " + expected);
}
assert(longHTML.indexOf("Part 1") < longHTML.indexOf("Section 1.1") && longHTML.indexOf("Section 1.2") < longHTML.indexOf("Part 2"), "long-form hierarchy order changed");
assert(longHTML.includes("Unsafe &lt;script&gt;Part&lt;/script&gt;") && longHTML.includes("Section &lt;One&gt;") && longHTML.includes("Empty &lt;Part&gt;"), "plan-derived titles were not escaped");
assert(!longHTML.includes("<script>") && !longHTML.includes("Wrong fallback section"), "long-form detail rendered unsafe or fallback-only content");
assert(longHTML.includes("<ol class=\"plan-section-list\" role=\"list\">") && longHTML.includes("<li class=\"plan-section-item\">"), "long-form sections are not rendered as semantic lists");
reports.showReportPlanEvent("evt_long_no_summary");
const noSummaryHTML = nodes.detailBody.innerHTML;
assert(noSummaryHTML.includes("<ol class=\"plan-section-list\" role=\"list\">"), "summary-less long-form list role missing");
assert(!noSummaryHTML.includes("연결된 근거 없음"), "Part-level missing-evidence noise was rendered even though all Sections have refs");
reports.showReportPlanPayload({});
assert(nodes.detailBody.innerHTML.includes("저장된 생성 계획이 없습니다."), "empty plan state changed");
`
	command := exec.Command("node", "-e", nodeScript)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("report plan display regression failed: %v\n%s", err, output)
	}
}

func TestPlasmaT6ProductionOrderRuntimeContracts(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for production-order runtime fixture")
	}
	scripts := mustIndexClassicScriptPaths(t)
	scriptJSON, err := json.Marshal(scripts)
	if err != nil {
		t.Fatal(err)
	}
	var code strings.Builder
	code.WriteString(`
const fs = require("fs");
const vm = require("vm");
const domReady = [];
const windowListeners = {};
const fetchLog = [];
const calls = [];
let currentFetchHandler = async (url, init = {}) => {
  const path = String(url);
  if (path.endsWith("/api/health")) return {Status:"ok"};
  if (path.endsWith("/api/runtime")) return {};
  if (path.endsWith("/api/missions")) return {missions:[]};
  if (path.endsWith("/api/local_paths/roots")) return {roots:[]};
  if (path.endsWith("/api/settings/model-defaults")) return {};
  if (path.endsWith("/api/settings/connectors/confluence/connections")) return {connections:[], oauth_configured:false};
  return {};
};
function classList() {
  const values = new Set();
  return {
    values,
    add(...names){ names.forEach((name) => values.add(name)); },
    remove(...names){ names.forEach((name) => values.delete(name)); },
    toggle(name, force){ const enabled = force === undefined ? !values.has(name) : Boolean(force); enabled ? values.add(name) : values.delete(name); return enabled; },
    contains(name){ return values.has(name); }
  };
}
function makeElement(id) {
  let html = "";
  const listeners = {};
  const node = {
    id, tagName:"DIV", nodeType:1, value:"", checked:false, disabled:false, open:false, hidden:false, files:[],
    dataset:{}, style:{}, attributes:{}, textContent:"", title:"", className:"",
    parentElement:null, children:[], selectedOptions:[{dataset:{contentId:"plain_text", start:"0", end:"0"}}],
    classList: classList(),
    setAttribute(name, value){ this.attributes[name] = String(value); if (name.startsWith("data-")) this.dataset[name.slice(5).replace(/-([a-z])/g, (_, c) => c.toUpperCase())] = String(value); },
    getAttribute(name){ return this.attributes[name] || ""; },
    removeAttribute(name){ delete this.attributes[name]; },
    append(...items){ items.forEach((item) => this.appendChild(item)); },
    appendChild(child){ child.parentElement = this; this.children.push(child); return child; },
    replaceChildren(...items){ this.children = []; this.append(...items); },
    replaceWith(next){ if (this.parentElement) this.parentElement.children = this.parentElement.children.map((item) => item === this ? next : item); next.parentElement = this.parentElement; },
    remove(){ if (this.parentElement) this.parentElement.children = this.parentElement.children.filter((item) => item !== this); },
    click(){ (listeners.click || []).forEach((fn) => fn({target:this, preventDefault(){}})); },
    focus(){}, select(){}, setSelectionRange(){}, scrollIntoView(){},
    contains(target){ return target === this || this.children.some((child) => child.contains?.(target)); },
    closest(selector){ return matches(this, selector) ? this : this.parentElement?.closest?.(selector) || null; },
    querySelector(selector){ return this.querySelectorAll(selector)[0] || null; },
    querySelectorAll(selector){ return this.children.flatMap((child) => [child, ...child.querySelectorAll(selector)]).filter((child) => matches(child, selector)); },
    addEventListener(event, listener){ if (typeof listener !== "function") throw new Error(id + " " + event + " listener is " + typeof listener); (listeners[event] ||= []).push(listener); },
    removeEventListener(event, listener){ listeners[event] = (listeners[event] || []).filter((fn) => fn !== listener); },
    dispatchEvent(event){ (listeners[event.type] || []).forEach((fn) => fn(event)); },
    getBoundingClientRect(){ return {top:0,left:0,width:100,height:20}; }
  };
  Object.defineProperty(node, "innerHTML", {
    get(){ return html; },
    set(value){
      html = String(value);
      if (html.includes("data-redpen-start-line")) {
        const block = makeElement(id + ":redpen-block");
        block.tagName = "P";
        block.dataset.redpenStartLine = "0";
        block.dataset.redpenEndLine = "1";
        block.dataset.redpenBlockKind = "paragraph";
        node.children = [block];
        block.parentElement = node;
      }
    }
  });
  return node;
}
function matches(node, selector) {
  if (!node) return false;
  if (selector === "a" || selector === "summary") return node.tagName === selector.toUpperCase();
  if (selector.startsWith("#")) return node.id === selector.slice(1);
  if (selector.startsWith(".")) return node.className.split(/\s+/).includes(selector.slice(1)) || node.classList.contains(selector.slice(1));
  const data = selector.match(/^\[data-([^\]=]+)(?:=\"([^\"]*)\")?\]$/);
  if (data) {
    const key = data[1].replace(/-([a-z])/g, (_, c) => c.toUpperCase());
    return Object.prototype.hasOwnProperty.call(node.dataset, key) && (data[2] === undefined || node.dataset[key] === data[2]);
  }
  return false;
}
const elements = new Map();
const getElement = (id) => {
  if (!elements.has(id)) elements.set(id, makeElement(id));
  return elements.get(id);
};
function response(payload, status = 200) {
  return {
    ok: status < 400, status, statusText: status < 400 ? "OK" : "ERR",
    headers: { get(name){ return name.toLowerCase() === "content-type" ? "application/json" : ""; } },
    json: async () => payload,
    text: async () => JSON.stringify(payload || {}),
    blob: async () => ({payload})
  };
}
let timeoutCalls = 0;
const context = {
  console, setTimeout(fn){ timeoutCalls += 1; if (typeof fn === "function") Promise.resolve().then(fn); return timeoutCalls; }, clearTimeout(){},
  setInterval(){ return 1; }, clearInterval(){}, requestAnimationFrame(fn){ if (typeof fn === "function") fn(); },
  queueMicrotask, Promise, Date, Math, JSON, RegExp, Error, TypeError, Set, Map, Array, Object, String, Number, Boolean,
  URL: { createObjectURL(){ return "blob:preview"; }, revokeObjectURL(){} },
  Blob: function(parts, options){ this.parts = parts; this.type = options?.type || ""; },
  FormData: function(){}, navigator:{clipboard:{writeText:async()=>{}}},
  localStorage:{getItem(){return null;}, setItem(){}, removeItem(){}},
  CSS:{escape:(value)=>String(value)}, location:{hash:"", origin:"http://127.0.0.1:65534"},
  crypto:{randomUUID(){ return "uuid-1"; }},
  confirm(){ return true; }, prompt(){ return "패치 지시"; },
  open(){ return {closed:false, document:{title:"", body:makeElement("preview-body")}, location:{href:""}, close(){this.closed=true;}}; },
  addEventListener(event, listener){ if (typeof listener !== "function") throw new Error("window " + event + " listener is " + typeof listener); (windowListeners[event] ||= []).push(listener); },
  removeEventListener(){}, matchMedia:() => ({matches:false, addEventListener(){}}),
  getSelection:() => ({isCollapsed:true, removeAllRanges(){}})
};
context.document = {
  documentElement: makeElement("documentElement"),
  body: makeElement("body"),
  createElement(tag){ const node = makeElement(tag); node.tagName = String(tag).toUpperCase(); return node; },
  getElementById: getElement,
  querySelector(selector){ return selector.startsWith("[data-tab=") ? makeElement("tab") : null; },
  querySelectorAll(){ return []; },
  addEventListener(event, listener){ if (typeof listener !== "function") throw new Error("document " + event + " listener is " + typeof listener); if (event === "DOMContentLoaded") domReady.push(listener); },
  removeEventListener(){}
};
context.window = context;
context.globalThis = context;
context.self = context;
context.markdownit = () => ({
  parse(){
    const token = {type:"paragraph_open", level:0, map:[0,1], attrs:[], attrJoin(name, value){ this.attrs.push([name, value]); }, attrSet(name, value){ this.attrs.push([name, value]); }};
    return [token, {type:"inline", content:"문단"}, {type:"paragraph_close"}];
  },
  renderer:{rules:{}, render(tokens){ const attrs = (tokens[0].attrs || []).map(([name, value]) => name + "=\"" + value + "\"").join(" "); return "<p " + attrs + ">문단</p>"; }},
  options:{},
  render(value){return String(value);}
});
context.DOMPurify = {sanitize(value){ return String(value); }};
context.katex = {renderToString(value){ return "<span>" + value + "</span>"; }};
context.mermaid = {initialize(){}, render: async () => ({svg:"<svg></svg>"})};
context.fetch = async (url, init = {}) => {
  fetchLog.push({url:String(url), init});
  const payload = await currentFetchHandler(url, init);
  return response(payload, payload?.__status || 200);
};
`)
	code.WriteString("const scriptPaths = ")
	code.Write(scriptJSON)
	code.WriteString(`;
vm.createContext(context);
for (const file of scriptPaths) vm.runInContext(fs.readFileSync(file, "utf8"), context, {filename:file});
if (context.window !== context.globalThis) throw new Error("fixture global must match browser classic-script global");
const Plasma = context.Plasma;
function bodyOf(init){ return init?.body ? JSON.parse(init.body) : {}; }
function routedTarget(route) {
  return { closest(selector) { return route[selector] || null; } };
}
function clickRoute(route) { Plasma.reports.onReportListClick({target:routedTarget(route)}); }
(async () => {
  for (const listener of domReady) await listener();
  await Promise.resolve(); await Promise.resolve();
  const plasmaState = Plasma.state;
  Object.assign(plasmaState, {
    missionId:"mis_1", selectionGeneration:1, detailGeneration:1,
    detail:{projection:{title:"Mission", mission_id:"mis_1"}, agent_executors:[], events:[
      {EventID:"evt_plan", EventType:"report.plan.completed", Payload:{plan:{summary:"plan", sections:[]}}},
      {EventID:"evt_draft", EventType:"report.draft.completed", Payload:{report_version_id:"ver_1", generation:{tool_session_id:"sess_1"}}},
      {EventID:"evt_tool", EventType:"mcp.tool.called", Payload:{tool_name:"mcp__report__read", success:true, duration_ms:1}}
    ], report_versions:[{report_version_id:"ver_1", title:"Report"}], report_progress:{attempt_id:"evt_plan", state:"failed", retry:{resume_failed:true}, nodes:[{id:"plan", kind:"plan", state:"failed"}]}, records:{proposals:[], evidence:[], claims:[], claim_confidence:[]}, sources:[], active_work:{items:[]}, workflow_runs:[]},
    selectedReportKey:"", reportPending:false, turnPending:false, workflowPending:false, agentModelTouched:true, agentReasoningEffortTouched:true,
    selectedProposals:new Set(["prop_1"])
  });
  getElement("agentExecutor").value = "codex";
  getElement("mcpMode").value = "auto";
  getElement("agentModel").value = "gpt-agent";
  getElement("agentReasoningEffort").value = "medium";
  getElement("reportAgentModel").value = "gpt-report";
  getElement("reportAgentReasoningEffort").value = "high";
  getElement("reportRigor").value = "balanced";
  getElement("reportLongFormExecutionStrategy").value = "serial";
  getElement("reportGenerationGuidance").value = "narrative-contract";
  getElement("candidateSource").value = "src_1|art_src";
  getElement("candidateEvidenceType").value = "observation";
  getElement("candidateSummary").value = "candidate";
  currentFetchHandler = async (url, init = {}) => {
    const path = String(url);
    if (path.endsWith("/reports/patch")) return {pending_event:{Payload:bodyOf(init)}};
    if (path.endsWith("/artifacts/art_md/html_export")) return {content:"<html></html>", artifact:{artifact_id:"html_1", filename:"basic.html"}};
    if (path.endsWith("/artifacts/art_md/redpen")) return (init.method || "GET") === "POST" ? {changed:true, workcopy:{revision:2, artifact:{artifact_id:"red_2"}}} : {exists:true, content:"문단", workcopy:{revision:1, artifact:{artifact_id:"red_1"}}};
    if (path.endsWith("/reports/retry")) return {};
    if (path.endsWith("/candidates/evidence")) return {};
    if (path.endsWith("/proposals/prop_1/approve")) return {};
    if (path.endsWith("/api/missions/mis_1")) return plasmaState.detail;
    if (path.endsWith("/api/missions")) return {missions:[{MissionID:"mis_1"}]};
    return currentFetchHandler.__base(url, init);
  };
  currentFetchHandler.__base = async (url, init) => ({});
  await Plasma.reports.patchReportArtifact("art_md", "Report");
  const patch = fetchLog.find((item) => item.url.endsWith("/reports/patch"));
  if (!patch) throw new Error("report patch did not reach mission API");
  const patchBody = bodyOf(patch.init);
  if (patchBody.agent_model !== "gpt-agent" || patchBody.agent_reasoning_effort !== "medium" || patchBody.mcp_mode !== "auto") throw new Error("patch model/effort payload changed");
  await Plasma.reports.exportReportArtifactHTML("art_md");
  if (!fetchLog.some((item) => item.url.endsWith("/artifacts/art_md/html_export")) || !Plasma.mission.missionArtifactPreviewURL("mis_1", "html_1").includes("/artifacts/html_1/preview")) throw new Error("basic HTML preview fallback failed");
  Plasma.reports.redpenController.open({sourceArtifactID:"art_md", content:"문단"});
  await Plasma.reports.redpenController.beforeLeave();
  Plasma.reports.redpenController.open({sourceArtifactID:"art_md", content:"문단"});
  getElement("reportRedpenStart").click();
  for (let i = 0; i < 8; i += 1) await Promise.resolve();
  const block = getElement("detailBody").children[0];
  getElement("detailBody").dispatchEvent({type:"click", target:block, preventDefault(){}});
  getElement("reportRedpenStart").click();
  if (!getElement("detailBody").children.some((child) => child.className === "report-redpen-inline-editor")) throw new Error("redpen pointer edit did not create inline editor: " + JSON.stringify({html:getElement("detailBody").innerHTML, children:getElement("detailBody").children.map((child) => ({id:child.id, className:child.className, dataset:child.dataset}))}));
  const reportRoutes = [];
  for (const name of ["viewReportArtifact","downloadReportArtifact","patchReportArtifact","exportReportArtifactHumanizedMarkdown","exportReportArtifactDesignedHTML","showReportPlan","showMCPTrace","exportReport","selectReport","createConversationExport","viewConversationExport","viewReportRedpenWorkcopy"]) {
    const original = Plasma.reports[name];
    Plasma.reports[name] = (...args) => { reportRoutes.push([name, ...args]); return original?.name === name ? undefined : Promise.resolve(); };
  }
  clickRoute({"[data-detail-json]":{dataset:{detailJson:"{\"ok\":true}", detailTitle:"상세"}}});
  clickRoute({"[data-report-artifact-id][data-action]":{dataset:{reportArtifactId:"art_md", action:"view"}}});
  clickRoute({"[data-report-artifact-id][data-action]":{dataset:{reportArtifactId:"art_md", action:"download-artifact"}}});
  clickRoute({"[data-report-artifact-id][data-action]":{dataset:{reportArtifactId:"art_md", action:"patch-artifact", reportTitle:"Report"}}});
  clickRoute({"[data-report-artifact-id][data-action]":{dataset:{reportArtifactId:"art_md", action:"start-humanized-markdown-artifact"}}});
  clickRoute({"[data-report-artifact-id][data-action]":{dataset:{reportArtifactId:"art_md", action:"view-designed-html-artifact"}}});
  clickRoute({"[data-report-artifact-id][data-action]":{dataset:{reportArtifactId:"art_md", action:"view-redpen-artifact"}}});
  clickRoute({"[data-report-version-id][data-action]":{dataset:{reportVersionId:"ver_1", action:"plan"}}});
  clickRoute({"[data-report-version-id][data-action]":{dataset:{reportVersionId:"ver_1", action:"mcp-trace"}}});
  clickRoute({"[data-report-version-id][data-action]":{dataset:{reportVersionId:"ver_1", action:"download-markdown"}}});
  clickRoute({"[data-report-key]":{dataset:{reportKey:"version:ver_1"}}});
  clickRoute({"[data-conversation-export-create]":{dataset:{}}});
  clickRoute({"[data-conversation-export-id][data-action]":{dataset:{conversationExportId:"conv_1", action:"view"}}});
  for (const expected of ["viewReportArtifact","downloadReportArtifact","patchReportArtifact","exportReportArtifactHumanizedMarkdown","exportReportArtifactDesignedHTML","showReportPlan","showMCPTrace","exportReport","selectReport","createConversationExport","viewConversationExport","viewReportRedpenWorkcopy"]) {
    if (!reportRoutes.some((call) => call[0] === expected)) throw new Error("report list route missing " + expected);
  }
  const resume = {dataset:{reportRetry:"resume_failed"}, disabled:false, addEventListener(_event, listener){ this.listener = listener; }};
  getElement("reportPipeline").querySelectorAll = (selector) => selector === "[data-report-retry]" ? [resume] : [];
  Plasma.reports.pipeline.render(plasmaState.detail.report_progress, Plasma.reports.reportPipelineRequestSummary(plasmaState.detail.report_progress));
  await resume.listener?.();
  if (!fetchLog.some((item) => item.url.endsWith("/reports/retry"))) throw new Error("pipeline retry did not reach mission API");
  const idleTimeoutCalls = timeoutCalls;
  Plasma.polling.schedulePendingPoll();
  if (timeoutCalls !== idleTimeoutCalls) throw new Error("selectedPending scheduled while idle");
  plasmaState.reportPending = true;
  Plasma.polling.schedulePendingPoll();
  if (timeoutCalls !== idleTimeoutCalls + 1) throw new Error("selectedPending did not schedule while report was pending");
  plasmaState.reportPending = false;
  Plasma.polling.clearPendingPoll();
  const beforeAllowed = Plasma.mission.beforeSelectionChange("mis_1", "mis_2");
  if (beforeAllowed !== true) throw new Error("mission beforeSelectionChange hook did not delegate");
  let afterOrder = [];
  Plasma.sources.loadConfluenceConnections = async () => { afterOrder.push("connections"); plasmaState.detailGeneration += 1; };
  Plasma.sources.loadConfluenceAccess = async () => { afterOrder.push("access"); };
  await Plasma.mission.afterSelectionApplied({missionId:"mis_1", selectionGeneration:1, detailGeneration:1});
  if (afterOrder.join(",") !== "connections") throw new Error("mission afterSelectionApplied stale check changed");
  await Plasma.proposals.proposeEvidence({preventDefault(){}});
  await Plasma.proposals.decideProposal("prop_1", "approve");
  await Plasma.proposals.bulkProposalAction("approve");
  Plasma.knowledge.showClaimConfidenceDetail("claim_1");
  Plasma.ledger.renderLedger([{EventID:"evt_1", EventType:"mcp.tool.called", Payload:{tool_name:"x", success:true}, Sequence:1, CreatedAt:"2026-07-30T00:00:00Z"}]);
})().catch((err) => { console.error(err); process.exit(1); });
`)
	command := exec.Command("node", "-e", code.String())
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("production-order runtime fixture failed: %v\n%s", err, output)
	}
}

func TestPlasmaT5ModulesDoNotUseDeletedGlobalFallbacks(t *testing.T) {
	app := mustReadPlasmaReportScripts(t)
	for _, forbidden := range []string{
		"typeof loadConfluenceConnections",
		"typeof loadConfluenceAccess",
		"typeof resetConfluenceMissionUI",
		"typeof confluenceUpdateFailureText",
		"globalThis.Plasma?.sources",
		"globalThis.loadConfluence",
		"globalThis.resetConfluence",
	} {
		if strings.Contains(app, forbidden) {
			t.Fatalf("app.js retains deleted-global compatibility fallback %q", forbidden)
		}
	}
	sourcesSettings := mustReadPlasmaSourceScripts(t) + mustReadPlasmaSettingsScripts(t)
	for _, forbidden := range []string{
		"typeof setConfluenceSettingsStatus",
		"typeof renderConfluenceSettingsControls",
		"typeof renderConfluenceAccessControls",
		"typeof confluenceAPITokenConnections",
		"typeof looksLikeConfluenceURL",
		"typeof sourceCandidateTitleForURL",
		"Object.assign(sources, {\n\n  })",
		"Object.assign(settings, {\n\n  })",
	} {
		if strings.Contains(sourcesSettings, forbidden) {
			t.Fatalf("T5 owner module retains deleted-global fallback or empty export %q", forbidden)
		}
	}
}

func TestStaticReportModelSelectionContract(t *testing.T) {
	combined := string(mustReadStatic(t, "static/index.html")) + mustReadPlasmaReportScripts(t) + mustReadPlasmaUIScripts(t) + string(mustReadStatic(t, "static/plasma/reports_model_selection.js"))
	for _, expected := range []string{`id="reportAgentModel"`, `id="reportAgentReasoningEffort"`, `/static/plasma/reports_model_selection.js`, "agent_model", "agent_reasoning_effort", "미션 설정 상속", "refreshEfforts", "setElementDisabled", "segmented-select-label"} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("missing report model selection contract %q", expected)
		}
	}
	if _, err := exec.LookPath("node"); err == nil {
		command := exec.Command("node", "-e", `globalThis.window=globalThis; globalThis.Plasma={reports:{}}; require('./static/plasma/reports_model_selection.js'); const p=globalThis.Plasma.reports.modelSelection.payload; if(JSON.stringify(p('',''))!==JSON.stringify({agent_model:'',agent_reasoning_effort:''})||p('gpt-5.5','').agent_model!=='gpt-5.5'||p('gpt-5.5','high').agent_reasoning_effort!=='high') process.exit(1)`)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("node payload fixture failed: %v: %s", err, output)
		}
	}
}

func TestStaticReportControlsIntegrateLabelsInsideSelects(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	styles := mustReadAppCSSComposed(t)
	for _, expected := range []string{
		`class="inline-control segmented-select-control report-select-rigor"`,
		`class="inline-control segmented-select-control report-select-model"`,
		`class="inline-control segmented-select-control report-select-effort"`,
		`class="inline-control segmented-select-control report-select-execution"`,
		`<span class="segmented-select-label">엄격도</span>`,
		`<span class="segmented-select-label">모델</span>`,
		`<span class="segmented-select-label">추론</span>`,
		`<span class="segmented-select-label">장문 작성</span>`,
	} {
		if !strings.Contains(index, expected) {
			t.Fatalf("missing integrated report control label %q", expected)
		}
	}
	for _, expected := range []string{
		".segmented-select-label",
		"pointer-events: none",
		"border-radius: 999px",
		"background: var(--accent-from)",
		"border-radius: 0",
		"text-overflow: ellipsis",
		".segmented-select-control:focus-within",
		"grid-template-columns: repeat(auto-fit, minmax(220px, 1fr))",
	} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing integrated report control style %q", expected)
		}
	}
}

func TestStaticReportGenerationGuidanceLongFormOptions(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	if strings.Contains(index, `id="reportGenerationGuidance"`) {
		t.Fatalf("report writing selector remained in the Web UI")
	}
	if strings.Contains(index, `<option value="balanced"`) ||
		!strings.Contains(index, `<option value="strict" selected>검증형</option>`) {
		t.Fatalf("report rigor UI must default to strict and omit balanced")
	}
	if strings.Contains(index, `reportPostHumanize`) ||
		strings.Contains(index, `report-post-humanize-control`) ||
		strings.Contains(index, `말투 보정`) {
		t.Fatalf("report humanize checkbox must not be exposed in the Web UI")
	}
	for _, legacy := range []string{
		`<option value="part-connective-economy-voice" selected>기본: 시각자료 계획</option>`,
		`<option value="section-brief-narrative-contract">섹션 중심</option>`,
		`<option value="section-brief-cluster-memory-narrative-contract">섹션 중심 + 풍부하게</option>`,
		`<option value="part-assembly-edit-tools">파트 조립 다듬기</option>`,
		`<option value="g2">기본 글쓰기</option>`,
		`<option value="section-brief">섹션 중심</option>`,
		`<option value="section-brief-cluster-memory">섹션 중심 + 풍부하게</option>`,
		`<option value="visual-plan">시각자료 계획</option>`,
		`<option value="section-brief-visual-plan">섹션 중심</option>`,
		`<option value="section-brief-cluster-memory-visual-plan">섹션 중심 + 풍부하게</option>`,
	} {
		if strings.Contains(index, legacy) {
			t.Fatalf("legacy long-form option remained in the Web UI: %q", legacy)
		}
	}

	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	fixture := `
	` + jsSourceRange(t, script, "const REPORT_RIGOR_LABELS", "\n\n  Object.assign(reports") + `
if (selectedReportGenerationGuidance("long_form") !== "section-brief-cluster-memory-narrative-contract") throw new Error("rich long-form default did not apply");
if (selectedReportGenerationGuidance("planned") !== "narrative-contract") throw new Error("planned reports did not keep narrative default");
if (reportGenerationGuidanceLabel("narrative-contract") !== "시각자료 계획") throw new Error("default choice label mismatch");
if (reportGenerationGuidanceLabel("part-connective-economy-voice") !== "시각자료 계획") throw new Error("legacy long-form voice label mismatch");
if (reportGenerationGuidanceLabel("g2") !== "기본 글쓰기") throw new Error("legacy g2 label not retained");
if (reportGenerationGuidanceLabel("section-brief") !== "섹션 중심 (이전)") throw new Error("legacy section label not distinguished");
if (reportGenerationGuidanceLabel("part-assembly-edit-tools") !== "파트 조립 다듬기") throw new Error("hidden part assembly label not retained");
if (reportGenerationGuidanceLabel("visual-plan") !== "시각자료 계획 (이전)") throw new Error("legacy visual label mismatch");
if (reportGenerationGuidanceLabel("section-brief-narrative-contract") !== "섹션 중심") throw new Error("section choice label mismatch");
if (reportGenerationGuidanceLabel("section-brief-cluster-memory-narrative-contract") !== "섹션 중심 + 풍부하게") throw new Error("cluster choice label mismatch");
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("report generation guidance fixture failed: %v: %s", err, out)
	}
}

func TestStaticSegmentedSelectDesignCoversEveryLabeledCompactControl(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	ids := []string{
		"agentExecutor",
		"agentModel",
		"agentReasoningEffort",
		"mcpMode",
		"controllerStrategy",
		"confluenceConnectionSelect",
		"confluenceSiteSelect",
		"confluenceRangeSelect",
		"confluenceUpdateRangeSelect",
		"reportRigor",
		"reportAgentModel",
		"reportAgentReasoningEffort",
		"reportLongFormExecutionStrategy",
		"workflowGoalDefaultModel",
		"workflowGoalDefaultReasoningEffort",
	}
	for _, id := range ids {
		selectIndex := strings.Index(index, `<select id="`+id+`"`)
		if selectIndex < 0 {
			t.Fatalf("missing select %q", id)
		}
		labelIndex := strings.LastIndex(index[:selectIndex], "<label")
		if labelIndex < 0 {
			t.Fatalf("select %q is not wrapped by a label", id)
		}
		labelOpenEnd := strings.Index(index[labelIndex:selectIndex], ">")
		if labelOpenEnd < 0 {
			t.Fatalf("select %q has a malformed label", id)
		}
		labelOpenTag := index[labelIndex : labelIndex+labelOpenEnd+1]
		if !strings.Contains(labelOpenTag, "segmented-select-control") {
			t.Fatalf("select %q does not use the segmented select design: %s", id, labelOpenTag)
		}
	}
	if got := strings.Count(index, "segmented-select-control"); got != len(ids) {
		t.Fatalf("segmented select coverage changed: got %d controls, want %d", got, len(ids))
	}
}

func TestStaticButtonDesignSystemDefinesSharedRoles(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	styles := mustReadAppCSSComposed(t)
	for _, expected := range []string{
		"--control-height: 34px",
		"--control-height-mini: 24px",
		"--button-shadow-soft:",
		"--button-shadow-hover:",
		"min-height: var(--control-height)",
		"button.button-secondary",
		"button.button-quiet",
		"button.button-danger",
		`button[aria-pressed="true"]`,
		`button[aria-busy="true"]`,
		"button.button-sm",
		"@media (max-width: 760px)",
		"--control-height: 40px",
		"agent-control-meta",
	} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing shared button role %q", expected)
		}
	}
	for _, expected := range []string{
		`id="focusToggle" class="focus-mode-handle"`,
		`id="themeToggle" class="icon-button quiet"`,
		`id="refreshMissions" class="icon-button quiet"`,
		`id="missionArchiveButton" class="mission-lifecycle-button hidden"`,
		`id="closeDetail" class="icon-button quiet"`,
	} {
		if !strings.Contains(index, expected) {
			t.Fatalf("utility button is not assigned to quiet role: %q", expected)
		}
	}
}

func TestStaticFocusModeHandleContracts(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	script := mustReadPlasmaReportScripts(t)
	styles := mustReadAppCSSComposed(t)
	focusSource := jsSourceRange(t, script, "// ── Focus mode:", "// ── Wave 6b:")
	topbar := htmlSection(t, index, `<header class="topbar">`, `</header>`)
	runtime := htmlSection(t, topbar, `<div class="runtime">`, `</div>`)
	if strings.Contains(topbar, `id="focusToggle"`) {
		t.Fatal("focusToggle must not remain inside header.topbar")
	}
	if strings.Contains(runtime, `id="focusToggle"`) {
		t.Fatal("focusToggle must not remain inside .runtime")
	}
	mainColumn := htmlSection(t, index, `<section class="main-column">`, `<nav id="tabBar"`)
	if !strings.HasPrefix(strings.TrimSpace(mainColumn), `<section class="main-column">
          <div class="mission-banner-shell">`) {
		t.Fatal("mission-banner-shell must be the first direct child of section.main-column")
	}
	shell := htmlSection(t, mainColumn, `<div class="mission-banner-shell">`, `</div>

          `)
	for _, expected := range []string{
		`<section class="panel mission-banner">`,
		`<button id="focusToggle" class="focus-mode-handle" type="button" aria-pressed="false" aria-label="상단 정보 접기" title="상단 정보 접기"><span aria-hidden="true">⌃</span></button>`,
	} {
		if !strings.Contains(shell, expected) {
			t.Fatalf("missing mission banner shell markup contract %q", expected)
		}
	}
	if strings.Index(shell, `<section class="panel mission-banner">`) > strings.Index(shell, `<button id="focusToggle"`) {
		t.Fatal("mission-banner-shell must contain mission-banner before focusToggle")
	}
	if strings.TrimSpace(shell[strings.LastIndex(shell, `<button id="focusToggle"`):]) != `<button id="focusToggle" class="focus-mode-handle" type="button" aria-pressed="false" aria-label="상단 정보 접기" title="상단 정보 접기"><span aria-hidden="true">⌃</span></button>` {
		t.Fatal("focusToggle must be the final direct child of mission-banner-shell")
	}
	for _, expected := range []string{
		`const STORAGE_KEY = "plasma.chatFocus";`,
		`btn.querySelector("span")`,
		`const label = on ? "상단 정보 펼치기" : "상단 정보 접기";`,
		`btn.setAttribute("aria-pressed", on ? "true" : "false");`,
		`btn.setAttribute("aria-label", label);`,
		`btn.setAttribute("title", label);`,
		`glyph.textContent = on ? "⌄" : "⌃";`,
		`localStorage.setItem(STORAGE_KEY, on ? "1" : "0");`,
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("missing focus toggle script contract %q", expected)
		}
	}
	if strings.Contains(focusSource, "btn.textContent") {
		t.Fatal("focus toggle must preserve its child span instead of replacing button textContent")
	}
	if regexp.MustCompile(`(?s)body\.chat-focus \.mission-banner\s*,\s*body\.chat-focus #agentControlsDetails`).MatchString(styles) {
		t.Fatal("mission banner must not share the immediate display:none focus rule")
	}
	if regexp.MustCompile(`(?s)body\.chat-focus \.mission-banner\s*\{[^}]*display:\s*none`).MatchString(styles) {
		t.Fatal("mission banner must fold instead of using display:none")
	}
	if strings.Contains(styles, "#focusToggle.active") {
		t.Fatal("focus handle pressed state must not use a separate active fill rule")
	}
	focusHandleBlock := regexp.MustCompile(`(?s)\.focus-mode-handle\s*\{(.+?)\n\}`).FindStringSubmatch(styles)
	if len(focusHandleBlock) != 2 {
		t.Fatal("missing focus handle CSS block")
	}
	if strings.Contains(focusHandleBlock[1], "border-top") {
		t.Fatal("focus handle must not remove border-top or use a flat-topped shape")
	}
	for _, expected := range []string{
		"body.chat-focus #agentControlsDetails,\nbody.chat-focus #workflowControlDetails {\n  display: none;",
		".mission-banner-shell {\n  position: relative;\n  flex: 0 0 auto;\n  min-height: 0;\n  display: grid;\n  grid-template-rows: minmax(0, 1fr);\n  transition: grid-template-rows 0.2s ease;",
		".mission-banner-shell > .mission-banner {\n  min-height: 0;\n  overflow: hidden;\n  opacity: 1;\n  pointer-events: auto;\n  transform: translateY(0) scale(1);\n  transition:\n    opacity 0.18s ease,\n    transform 0.18s ease;",
		"body.chat-focus .mission-banner-shell {\n  grid-template-rows: minmax(0, 0fr);",
		"body.chat-focus .mission-banner {\n  opacity: 0;\n  pointer-events: none;\n  transform: translateY(-18px) scale(0.98);",
		".focus-mode-handle {\n  position: absolute;\n  left: 50%;\n  bottom: -12px;\n  z-index: 12;",
		"width: 68px;\n  height: 24px;\n  min-height: 24px;\n  padding: 0;",
		"border-radius: 999px;",
		"border-color: color-mix(in srgb, var(--line2) 82%, transparent);",
		"linear-gradient(180deg, color-mix(in srgb, var(--surface2) 94%, var(--amber) 6%), var(--surface));",
		"color: var(--muted);",
		"box-shadow: 0 8px 22px rgba(0, 0, 0, 0.24);",
		"transform: translateX(-50%);",
		"background 0.16s ease,\n    border-color 0.16s ease,\n    color 0.16s ease,\n    transform 0.16s ease;",
		".focus-mode-handle:hover:not(:disabled),\n.focus-mode-handle:focus-visible {\n  border-color: color-mix(in srgb, var(--amber) 45%, var(--line2));",
		"linear-gradient(180deg, color-mix(in srgb, var(--surface2) 86%, var(--amber) 14%), var(--surface));",
		"color: var(--text);",
		".focus-mode-handle span {\n  display: block;\n  font-size: 20px;\n  line-height: 1;",
		".mission-banner-shell,\n  .mission-banner-shell > .mission-banner,\n  .focus-mode-handle {\n    transition: none;",
	} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing focus handle CSS contract %q", expected)
		}
	}
	mobile760Start := strings.LastIndex(styles, "@media (max-width: 760px)")
	mobile480Start := strings.LastIndex(styles, "@media (max-width: 480px)")
	if mobile760Start < 0 || mobile480Start < 0 || mobile480Start <= mobile760Start {
		t.Fatal("missing max-width 760px block")
	}
	mobile760 := styles[mobile760Start:mobile480Start]
	for _, expected := range []string{
		"padding: 10px;",
		"body.chat-focus .workspace {\n    padding-top: 20px;",
		".focus-mode-handle {\n    bottom: -16px;\n    width: 76px;\n    height: 32px;\n    min-height: 32px;",
	} {
		if !strings.Contains(mobile760, expected) {
			t.Fatalf("missing mobile focus handle contract %q", expected)
		}
	}
	mobile480 := regexp.MustCompile(`(?s)@media \(max-width: 480px\)\s*\{(.+?)\n\}`).FindStringSubmatch(styles)
	if len(mobile480) != 2 {
		t.Fatal("missing max-width 480px block")
	}
	if !strings.Contains(mobile480[1], "padding: 8px;") {
		t.Fatal("max-width 480px workspace padding must restore normal mobile padding")
	}
}

func TestFocusModeHandleNodeBehavior(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for the focus mode DOM fixture")
	}
	script := mustReadPlasmaReportScripts(t)
	source := jsSourceRange(t, script, "// ── Focus mode:", "// ── Wave 6b:")
	fixture := `
const attrs = {};
let active = false;
let chatFocus = false;
let listener = null;
const glyph = {textContent:"⌃"};
const btn = {
  classList:{toggle(name,on){ if(name==="active") active=on; }},
  setAttribute(name,value){ attrs[name]=value; },
  querySelector(selector){ return selector==="span" ? glyph : null; },
  addEventListener(event,fn){ if(event==="click") listener=fn; },
};
const body = {classList:{
  toggle(name,on){ if(name==="chat-focus") chatFocus=on; },
  contains(name){ return name==="chat-focus" && chatFocus; },
}};
const store = {"plasma.chatFocus":"0"};
const localStorage = {
  getItem(key){ return Object.prototype.hasOwnProperty.call(store,key) ? store[key] : null; },
  setItem(key,value){ store[key]=value; },
};
const document = {body};
const $ = (id) => id==="focusToggle" ? btn : null;
` + source + `
if (attrs["aria-pressed"] !== "false" || attrs["aria-label"] !== "상단 정보 접기" || attrs.title !== "상단 정보 접기" || glyph.textContent !== "⌃" || active || chatFocus) throw new Error("initial collapsed state mismatch");
if (typeof listener !== "function") throw new Error("missing click listener");
listener();
if (store["plasma.chatFocus"] !== "1" || attrs["aria-pressed"] !== "true" || attrs["aria-label"] !== "상단 정보 펼치기" || attrs.title !== "상단 정보 펼치기" || glyph.textContent !== "⌄" || !active || !chatFocus) throw new Error("focused state mismatch");
listener();
if (store["plasma.chatFocus"] !== "0" || attrs["aria-pressed"] !== "false" || attrs["aria-label"] !== "상단 정보 접기" || attrs.title !== "상단 정보 접기" || glyph.textContent !== "⌃" || active || chatFocus) throw new Error("restored state mismatch");
`
	if output, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("focus handle DOM fixture failed: %v: %s", err, output)
	}
}

func TestStaticTabControlsKeepTheirFlatOriginalTreatment(t *testing.T) {
	styles := mustReadAppCSSComposed(t)
	for _, expected := range []string{
		".tab {",
		"min-height: 38px",
		".source-tab {",
		"min-height: 36px",
		"box-shadow: none",
	} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing original flat tab treatment %q", expected)
		}
	}
}

func TestStaticReportDirectionIsOptionalAndPrecedesGenerationAction(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	directionDetails := strings.Index(index, `class="report-direction-details"`)
	directionInput := strings.Index(index, `id="reportDirectionHint"`)
	settings := strings.Index(index, `class="report-generation-settings"`)
	generate := strings.Index(index, `id="draftQuickReport"`)
	if directionDetails < 0 || directionInput < 0 || settings < 0 || generate < 0 {
		t.Fatal("missing optional direction or report generation controls")
	}
	if !(settings < directionDetails && directionDetails < directionInput && directionInput < generate) {
		t.Fatalf("unexpected report control order: details=%d input=%d settings=%d generate=%d", directionDetails, directionInput, settings, generate)
	}
	for _, expected := range []string{"방향 추가", "선택", "이번 요청에만 적용할 약한 편집 방향"} {
		if !strings.Contains(index, expected) {
			t.Fatalf("missing optional direction wording %q", expected)
		}
	}
}

func TestStaticReportGenerationContextIsVisibleWhilePendingAndOnArtifacts(t *testing.T) {
	script := mustReadPlasmaReportScripts(t)
	for _, expected := range []string{
		"reportGenerationContext",
		"reportGenerationSummaryHTML",
		"reportPipelineRequestSummary",
		"timeShort(startedAt)",
		"shouldHideDraftPendingNotice",
		"reports.pipeline.render(state.detail?.report_progress, reportPipelineRequestSummary(state.detail?.report_progress))",
		"report-generation-summary",
		"pending_event_id",
		"rigor_label",
		"agent_model",
		"agent_reasoning_effort",
		"direction_hint",
		"미션 설정 상속",
		"지정 없음",
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("missing report generation context contract %q", expected)
		}
	}
}

func TestStaticReportDraftPendingNoticeMergesWithMatchingPipelineOnly(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	fixture := `
const REPORT_RIGOR_LABELS = {balanced:"균형"};
const REPORT_MODE_LABELS = {planned:"일반",long_form:"장문"};
const REPORT_EXECUTION_STRATEGY_LABELS = {serial:"순차",section_fanout:"빠른 병렬"};
const state = {detail:{report_progress:{attempt_id:"evt_pending",state:"running"},events:[{EventID:"evt_pending",EventType:"report.draft.pending",Payload:{title:"<제목>",started_at:"2026-07-13T01:02:03Z",report_mode:"long_form",execution_strategy:"section_fanout",generation_guidance_profile:"g2",rigor_level:"balanced",agent_model:"gpt-safe",agent_reasoning_effort:"medium",direction_hint:"<방향>"}}]}};
let notices=[], busy=[];
const setReportBusy = (value) => busy.push(value);
const setReportNotice = (text, kind) => notices.push({text, kind});
const reportPendingMessage = (event) => "pending:" + (event && event.EventID || "");
const reportTimingDetails = () => "";
const reportGenerationGuidanceLabel = (value) => value === "g2" ? "G2" : value;
const timeShort = (value) => "LOCAL:" + value;
const reports = {REPORT_MODE_LABELS,REPORT_EXECUTION_STRATEGY_LABELS,REPORT_RIGOR_LABELS,reportGenerationGuidanceLabel,setReportBusy,setReportNotice,shouldHideDraftPendingNotice,reportPendingMessage,reportTimingDetails:()=>""};
` + jsFunctionSource(t, script, "eventByID") + `
` + jsSourceRange(t, script, "function reportGenerationContext", "function reportSourceContext") + `
` + jsFunctionSource(t, script, "renderReportDraftStatus") + `
const summary = reportPipelineRequestSummary(state.detail.report_progress);
if(summary.mode!=="장문"||summary.strategy!=="빠른 병렬"||summary.guidance!=="G2"||summary.rigor!=="균형"||summary.model!=="gpt-safe"||summary.effort!=="medium"||summary.direction!=="<방향>"||summary.startedAt!=="LOCAL:2026-07-13T01:02:03Z"||summary.startedAtDateTime!=="2026-07-13T01:02:03Z")throw new Error("request summary mismatch");
renderReportDraftStatus({state:"pending",event:{EventID:"evt_pending",EventType:"report.draft.pending",Payload:{}}},false);
if(notices.length!==1||notices[0].text!=="")throw new Error("matching draft pending notice was not hidden");
state.detail.report_progress = null;
renderReportDraftStatus({state:"pending",event:{EventID:"evt_pending",EventType:"report.draft.pending",Payload:{}}},false);
if(notices.at(-1).text!=="pending:evt_pending")throw new Error("missing progress should keep pending notice");
state.detail.report_progress = {attempt_id:"evt_other",state:"running"};
renderReportDraftStatus({state:"pending",event:{EventID:"evt_pending",EventType:"report.draft.pending",Payload:{}}},false);
if(notices.at(-1).text!=="pending:evt_pending")throw new Error("mismatched attempt should keep pending notice");
state.detail.report_progress = {attempt_id:"evt_pending",state:"running"};
renderReportDraftStatus({state:"pending",event:{EventID:"evt_design",EventType:"report.design.pending",Payload:{}}},false);
if(notices.at(-1).text!=="pending:evt_design")throw new Error("design pending should keep existing notice");
renderReportDraftStatus({state:"failed",event:{EventType:"report.draft.failed",Payload:{error:"실패"}}},true);
if(!notices.at(-1).text.includes("리포트 초안 생성 실패")||notices.at(-1).kind!=="error")throw new Error("failure notice changed");
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("report draft pending merge fixture: %v: %s", err, out)
	}
}

func TestStaticReportModelSelectionFollowsExecutorAndActiveGuards(t *testing.T) {
	conversationControlsScript := string(mustReadStatic(t, "static/plasma/conversation_agent_controls.js"))
	executorBody := jsFunctionBody(t, conversationControlsScript, "onAgentExecutorChange")
	for _, expected := range []string{"ReportModelSelection.render", `$("agentExecutor").value`} {
		if !strings.Contains(executorBody, strings.Replace(expected, "ReportModelSelection.render", "callbacks.renderReportModelSelection", 1)) {
			t.Fatalf("executor switch missing %q: %s", expected, executorBody)
		}
	}
	appScript := mustReadPlasmaReportScripts(t)
	formsBody := jsFunctionBody(t, appScript, "setFormsEnabled")
	for _, expected := range []string{"reportAgentModel", "reportAgentReasoningEffort", "state.turnPending", "state.workflowPending", "state.reportPending", "draftQuickReport", "draftLongReport"} {
		if !strings.Contains(formsBody, expected) {
			t.Fatalf("report guard missing %q", expected)
		}
	}
	module := string(mustReadStatic(t, "static/plasma/reports_model_selection.js"))
	for _, expected := range []string{"status?.models", "model?.reasoning_efforts", `effortSelect.innerHTML`, `value=""`} {
		if !strings.Contains(module, expected) {
			t.Fatalf("selection semantics missing %q", expected)
		}
	}
}

func TestStaticSettingsExposeModelDefaultsCard(t *testing.T) {
	html := string(mustReadStatic(t, "static/index.html"))
	appScript := mustReadPlasmaReportScripts(t)
	modelSettingsScript := mustReadPlasmaSettingsScripts(t)
	combined := html + "\n" + appScript + "\n" + modelSettingsScript
	for _, expected := range []string{
		`id="modelDefaultsDetails"`,
		`id="modelDefaultsForm"`,
		`id="workflowGoalDefaultModel"`,
		`id="workflowGoalDefaultReasoningEffort"`,
		`/static/plasma/settings_model_defaults.js`,
		`/api/settings/model-defaults`,
		`saveModelDefaults`,
		`loadModelDefaults`,
		`renderModelDefaultEfforts`,
		`자율진행 조향 모델`,
		`자율진행을 조향하는 모델`,
		`현재는 시작 시점의 3층 지시 초안 생성에만 사용`,
		`새 에이전트 세션`,
		`보고서 생성`,
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("expected model defaults settings surface %q", expected)
		}
	}
	settingsPanel := htmlSection(t, html, `data-tab-panel="settings"`, `id="errorToast"`)
	if strings.Index(settingsPanel, `id="missionSettingsDetails"`) < 0 || strings.Index(settingsPanel, `id="modelDefaultsDetails"`) < 0 || strings.Index(settingsPanel, `id="confluenceSettingsDetails"`) < 0 ||
		strings.Index(settingsPanel, `id="missionSettingsDetails"`) > strings.Index(settingsPanel, `id="modelDefaultsDetails"`) ||
		strings.Index(settingsPanel, `id="modelDefaultsDetails"`) > strings.Index(settingsPanel, `id="confluenceSettingsDetails"`) {
		t.Fatalf("mission management, model defaults, and Confluence settings must stay ordered in the Settings panel")
	}
	setFormsBody := jsFunctionBody(t, appScript, "setFormsEnabled")
	for _, forbidden := range []string{"modelDefaultsForm", "workflowGoalDefaultModel", "workflowGoalDefaultReasoningEffort"} {
		if strings.Contains(setFormsBody, forbidden) {
			t.Fatalf("global model default setting %q must not be disabled by mission-bound form state", forbidden)
		}
	}
}

func TestModelSettingsScriptUsesCodexCatalogAndReasoningEfforts(t *testing.T) {
	script := mustReadPlasmaSettingsScripts(t)
	for _, expected := range []string{
		`status.models`,
		`reasoning_efforts`,
		`default_reasoning_effort`,
		`workflow_goal_model`,
		`workflow_goal_reasoning_effort`,
		`method: "PATCH"`,
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("model settings script missing %q", expected)
		}
	}
	saveBody := jsFunctionBody(t, script, "saveModelDefaults")
	if strings.Contains(saveBody, "JSON.stringify") {
		t.Fatalf("model settings save must pass a JSON object to api(); api() owns JSON encoding: %s", saveBody)
	}
	payloadIndex := strings.Index(saveBody, "const payload = {")
	busyIndex := strings.Index(saveBody, "state.modelDefaultsBusy = true")
	if payloadIndex < 0 || busyIndex < 0 || payloadIndex > busyIndex {
		t.Fatalf("model settings save must capture form values before busy render resets controls: %s", saveBody)
	}
	if !strings.Contains(saveBody, "body: payload") {
		t.Fatalf("model settings save must submit the captured payload: %s", saveBody)
	}
}

func TestSetReportBusyPreservesEveryActiveWorkGuard(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for the report control state-transition fixture")
	}
	script := mustReadPlasmaReportScripts(t)
	source := jsFunctionSource(t, script, "activeWorkBlocksControl") + "\n" + jsFunctionSource(t, script, "syncReportControls") + "\n" + jsFunctionSource(t, script, "setReportBusy")
	fixture := `
const elements = {};
for (const id of ["reportStatus","reportRigor","reportAgentModel","reportAgentReasoningEffort","reportLongFormExecutionStrategy","draftQuickReport","draftLongReport","cancelReportButton"]) {
  elements[id] = {disabled:false,textContent:"",classList:{toggle(){}}};
}
const $ = (id) => elements[id];
const state = {detail:{active_work:{blocked_controls:[]}},turnPending:false,workflowPending:false,workflowGoalDraftPending:false,reportPending:false};
const missionLifecycleWriteBlocked = () => false;
const window = {Plasma:{ui:{
  setElementDisabled(id, disabled) { elements[id].disabled = Boolean(disabled); },
  setButtonText(id, text) { elements[id].textContent = text; }
}}};
` + source + `
const controls = ["reportRigor","reportAgentModel","reportAgentReasoningEffort","reportLongFormExecutionStrategy","draftQuickReport","draftLongReport"];
function assertDisabled(label) {
  if (!controls.every((id) => elements[id].disabled)) throw new Error(label + " re-enabled a report control");
}
for (const guard of ["agent_turn_running","workflow_running","report_generation_running"]) {
  state.detail.active_work.blocked_controls = [{control:"report_start",reason_codes:[guard]}];
  state.turnPending = state.workflowPending = state.workflowGoalDraftPending = state.reportPending = false;
  setReportBusy(false);
  assertDisabled(guard);
}
state.detail.active_work.blocked_controls = [];
for (const guard of ["turnPending","workflowPending","workflowGoalDraftPending"]) {
  state.turnPending = state.workflowPending = state.workflowGoalDraftPending = state.reportPending = false;
  state[guard] = true;
  setReportBusy(false);
  assertDisabled(guard);
}
state.turnPending = state.workflowPending = state.workflowGoalDraftPending = false;
setReportBusy(true);
assertDisabled("reportPending");
setReportBusy(false);
if (controls.some((id) => elements[id].disabled)) throw new Error("controls did not re-enable after every guard cleared");
`
	command := exec.Command("node", "-e", fixture)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("report control state-transition fixture failed: %v: %s", err, output)
	}
}

func TestDraftReportRejectsEveryActiveWorkStateBeforeAPI(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for the report start race fixture")
	}
	source := jsFunctionSource(t, mustReadPlasmaReportScripts(t), "draftReport")
	source = strings.Replace(source, "function draftReport", "async function draftReport", 1)
	fixture := `
let apiCalls = 0;
const state = {detail:{projection:{title:"Mission"}},missionId:"mis_test",turnPending:false,workflowPending:false,workflowGoalDraftPending:false,reportPending:false};
const requireMission = () => true;
const api = async () => { apiCalls++; return {}; };
const $ = () => { throw new Error("draftReport touched controls before rejecting active work"); };
` + source + `
(async () => {
  for (const guard of ["turnPending","workflowPending","workflowGoalDraftPending","reportPending"]) {
    state.turnPending = state.workflowPending = state.workflowGoalDraftPending = state.reportPending = false;
    state[guard] = true;
    await draftReport("planned");
    if (apiCalls !== 0) throw new Error(guard + " allowed report API call");
  }
})().catch((error) => { console.error(error); process.exit(1); });
`
	command := exec.Command("node", "-e", fixture)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("draftReport active-work fixture failed: %v: %s", err, output)
	}
}

func TestDraftReportPostHumanizePayloadIsLongFormOnly(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for the report humanize payload fixture")
	}
	source := jsFunctionSource(t, mustReadPlasmaReportScripts(t), "draftReport")
	source = strings.Replace(source, "function draftReport", "async function draftReport", 1)
	fixture := `
const calls = [];
const state = {detail:{projection:{title:"Mission"}},turnPending:false,workflowPending:false,workflowGoalDraftPending:false,reportPending:false};
const nodes = {
  reportAgentModel:{value:""}, reportAgentReasoningEffort:{value:""}, reportLongFormExecutionStrategy:{value:"serial"},
  reportRigor:{value:"strict"}, agentExecutor:{value:"codex"}, mcpMode:{value:"auto"}
};
const $ = (id) => nodes[id] || {value:""};
const requireMission = () => true;
const captureMissionSelection = () => ({missionId:"mis_1"});
const ownsMissionSelection = () => true;
const missionApi = async (_owner, path, init) => {
  calls.push({path, body:init.body});
  return {pending_event:{Payload:init.body}};
};
const reports = {
  modelSelection:{payload:() => ({agent_model:"",agent_reasoning_effort:""})},
  selectedReportGenerationGuidance:(mode) => mode === "long_form" ? "section-brief-cluster-memory-narrative-contract" : "narrative-contract",
  direction:{current:()=>"", clear(){}},
  setReportBusy(busy){ state.reportPending = busy; }, setReportNotice(){}, reportPendingMessage(){ return "pending"; }
};
const reloadMission = async () => {};
const schedulePendingPoll = () => {};
const showError = (err) => { throw err; };
` + source + `
(async () => {
  await draftReport("long_form");
  state.reportPending = false;
  await draftReport("planned");
  state.reportPending = false;
  await draftReport("one_take");
  state.reportPending = false;
  await draftReport("long_form");
  const got = calls.map((call) => call.body.post_report_humanize);
  if (got.join(",") !== "enabled,disabled,disabled,enabled") throw new Error("unexpected humanize payloads " + got.join(","));
  if (calls[0].body.generation_guidance_profile !== "section-brief-cluster-memory-narrative-contract") throw new Error("long-form rich guidance not sent");
  if (calls[1].body.generation_guidance_profile !== "narrative-contract") throw new Error("planned narrative guidance not sent");
})().catch((error) => { console.error(error); process.exit(1); });
`
	command := exec.Command("node", "-e", fixture)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("draftReport humanize payload fixture failed: %v: %s", err, output)
	}
}

func TestStaticAppLabelsPendingEvidenceSignalType(t *testing.T) {
	script := []byte(mustReadPlasmaReportScripts(t))
	content := string(script) + mustReadPlasmaSourceScripts(t)
	for _, expected := range []string{
		"EVIDENCE_TYPE_LABELS",
		"근거 신호:",
		"evidenceTypeLabel(record.evidence_type)",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected static app to preserve pending evidence signal label %q", expected)
		}
	}
}

func TestStaticAppExposesControllerStrategySelector(t *testing.T) {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script := []byte(mustReadPlasmaReportScripts(t))
	conversationScript := mustReadPlasmaConversationScripts(t)
	for _, expected := range []string{
		`id="controllerStrategy"`,
		`value="v2"`,
		`value="v3"`,
		"조향 전략",
	} {
		if !strings.Contains(string(html), expected) {
			t.Fatalf("expected static app HTML to expose controller strategy selector %q", expected)
		}
	}
	if !strings.Contains(string(script)+conversationScript, "controller_strategy") ||
		!strings.Contains(string(script)+conversationScript, "controllerStrategy") {
		t.Fatalf("expected static app script to submit controller strategy")
	}
}

func TestStaticAppExposesEnvironmentBadge(t *testing.T) {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script := []byte(mustReadPlasmaReportScripts(t))
	style := mustReadAppCSSComposed(t)
	combined := string(html) + "\n" + string(script) + "\n" + mustReadPlasmaUIScripts(t) + "\n" + mustReadPlasmaWorkflowScripts(t) + "\n" + style
	for _, expected := range []string{
		`id="environmentBadge"`,
		"/api/runtime",
		"environment_label",
		"environment-badge",
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("expected static app to expose environment badge %q", expected)
		}
	}
}

func TestStaticReportMarkdownPreviewWrapsAndMarksHeadings(t *testing.T) {
	content := mustReadAppCSSComposed(t)
	for _, expected := range []string{
		".report-modal-body.turn-markdown",
		"overflow-wrap: anywhere",
		"white-space: pre-wrap",
		".report-modal-body.turn-markdown h1::before { content: \"#\"; }",
		".report-modal-body.turn-markdown h2::before { content: \"##\"; }",
		".report-modal-body.turn-markdown h3::before { content: \"###\"; }",
		".report-modal-body.turn-markdown h4::before { content: \"####\"; }",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected report markdown preview CSS to include %q", expected)
		}
	}
	for _, forbidden := range []string{
		`content: "Part`,
		`content: "Section`,
	} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("report markdown preview CSS should not synthesize report heading text %q", forbidden)
		}
	}
}

func TestStaticReportControlsShareMobileWidthWithoutLabelColumns(t *testing.T) {
	content := mustReadAppCSSComposed(t)
	for _, expected := range []string{
		`.report-request-actions`,
		`.report-generation-settings`,
		`.report-generation-settings > .inline-control`,
		`display: flex`,
		`.report-generation-settings .inline-control select`,
		`.report-mode-actions`,
		`justify-content: flex-end`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected mobile report control alignment CSS to include %q", expected)
		}
	}
	if strings.Contains(content, `grid-template-columns: 52px minmax(0, 1fr)`) {
		t.Fatal("mobile report controls should not reserve a separate label column")
	}
}

func TestStaticDetailModalKeepsTitleBarVisibleWhileBodyScrolls(t *testing.T) {
	content := mustReadAppCSSComposed(t)
	for _, expected := range []string{
		".modal-card > .panel-head",
		"position: sticky",
		"overflow: hidden",
		".detail-body",
		"display: block",
		"flex: 1 1 auto",
		"overflow: auto",
		"overscroll-behavior: contain",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected detail modal CSS to keep the title bar visible while body scrolls: %q", expected)
		}
	}
}

func TestStaticReportPreviewShowsVerticalPositionRatio(t *testing.T) {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script := []byte(mustReadPlasmaReportScripts(t))
	style := mustReadAppCSSComposed(t)
	combined := string(html) + "\n" + string(script) + "\n" + mustReadPlasmaUIScripts(t) + "\n" + mustReadPlasmaWorkflowScripts(t) + "\n" + style
	for _, expected := range []string{
		`id="detailPositionRatio"`,
		"detail-scroll-ratio",
		"detailScrollRatioEnabled",
		"enableDetailScrollRatio",
		"disableDetailScrollRatio",
		"updateDetailScrollRatio",
		"detailScrollPosition",
		"scrollTop / maxScroll",
		"`위치 ${Math.max(0, Math.min(100, percent))}%`",
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("expected report preview vertical position contract %q", expected)
		}
	}
	for _, forbidden := range []string{
		"instrumentHTMLPreview",
		"window.parent.postMessage",
		`plasma:detail-scroll-ratio`,
		`allow-same-origin`,
	} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("HTML preview scroll ratio support should stay disabled; found %q", forbidden)
		}
	}
}

func TestPlasmaUIDetailModalWidthContract(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for the detail modal width fixture")
	}
	ui := string(mustReadStatic(t, "static/plasma/ui.js")) + "\n" + string(mustReadStatic(t, "static/plasma/ui_feedback.js")) + "\n" + string(mustReadStatic(t, "static/plasma/ui_detail.js"))
	fixture := `
const nodes = {};
function classList(initial = []) {
  const values = new Set(initial);
  return {
    add(name) { values.add(name); },
    remove(name) { values.delete(name); },
    contains(name) { return values.has(name); },
    toggle(name, force) {
      if (force) values.add(name); else values.delete(name);
    }
  };
}
nodes.detailTitle = {textContent:""};
nodes.detailBody = {innerHTML:"", textContent:"", scrollTop:0, scrollHeight:0, clientHeight:0};
const card = {classList: classList(["modal-card", "modal-card--wide"])};
nodes.detailModal = {classList: classList(["hidden"]), querySelector: () => card};
nodes.detailPositionRatio = {classList: classList(["hidden"]), textContent:""};
const document = {getElementById: (id) => nodes[id] || null};
const window = {Plasma:{state:{detailText:"", detailScrollRatioEnabled:false}, dom:{
  $: (id) => document.getElementById(id),
  escapeHTML: (value) => String(value ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;")
}}};
const navigator = {};
const requestAnimationFrame = () => {};
` + ui + `
window.Plasma.ui.showDetail("일반", {a:1});
if (!card.classList.contains("modal-card--wide")) throw new Error("showDetail must preserve existing wide class when width is omitted");
if (nodes.detailModal.classList.contains("hidden")) throw new Error("showDetail must still open the modal");
if (nodes.detailTitle.textContent !== "일반" || window.Plasma.state.detailText !== JSON.stringify({a:1}, null, 2)) throw new Error("showDetail title/text behavior changed");
if (!nodes.detailBody.innerHTML.startsWith("<pre>")) throw new Error("showDetail must render preformatted detail text");
window.Plasma.ui.openDetailModal(false);
if (card.classList.contains("modal-card--wide")) throw new Error("explicit false must remove wide class");
window.Plasma.ui.openDetailModal(true);
if (!card.classList.contains("modal-card--wide")) throw new Error("explicit true must add wide class");
window.Plasma.ui.openDetailModal();
if (!card.classList.contains("modal-card--wide")) throw new Error("omitted width must preserve current wide class");
`
	if output, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("detail modal width fixture failed: %v: %s", err, output)
	}
	for _, forbidden := range []string{"options = {}", "options.html", "options.scrollRatio", "Boolean(options.wide)"} {
		if strings.Contains(ui, forbidden) {
			t.Fatalf("showDetail must not retain the removed options API %q", forbidden)
		}
	}
}

func TestStaticAppExposesWorkflowControlsWithoutTerminalUI(t *testing.T) {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script := []byte(mustReadPlasmaReportScripts(t))
	combined := string(html) + "\n" + string(script) + "\n" + mustReadPlasmaUIScripts(t) + "\n" + mustReadPlasmaWorkflowScripts(t)
	for _, expected := range []string{
		`id="workflowInstruction"`,
		`<label class="field-label hidden" for="workflowStepInstructionMode">스텝 지시 방식</label>`,
		`id="workflowStepInstructionMode" class="hidden" aria-hidden="true" tabindex="-1"`,
		`<option value="layered" selected>3층 지시</option>`,
		`id="workflowLayeredFields" class="workflow-layered-fields"`,
		`id="draftWorkflowGoalButton"`,
		`id="workflowRunGoal"`,
		`id="workflowStepInstruction"`,
		`id="startWorkflowButton"`,
		`id="stopWorkflowButton"`,
		"/workflows/goal_draft",
		"workflowRawInputValue",
		"state.workflowGoalDraftPending &&",
		"/workflows",
		"workflow_runs",
		"step_instruction_mode",
		"workflowStepInstructionMode",
		"updateWorkflowStepInstructionMode();",
		"user_instruction_raw",
		"run_goal",
		"const WORKFLOW_DEFAULT_MAX_STEPS = 20",
		"const WORKFLOW_DEFAULT_MAX_DURATION_MS = 0",
		"max_steps: WORKFLOW_DEFAULT_MAX_STEPS",
		"max_duration_ms: WORKFLOW_DEFAULT_MAX_DURATION_MS",
		"max_steps: Number(run.max_steps ?? WORKFLOW_DEFAULT_MAX_STEPS)",
		"max_duration_ms: Number(run.max_duration_ms ?? WORKFLOW_DEFAULT_MAX_DURATION_MS)",
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("expected static app to expose workflow control %q", expected)
		}
	}
	for _, forbidden := range []string{
		"rawInputFallback",
		`$("turnText").addEventListener("input", workflow.onWorkflowRawInput)`,
		"PTY",
		`<option value="current"`,
		`id="workflowStepInstructionMode">`,
		`id="workflowLayeredFields" class="workflow-layered-fields hidden"`,
		"3층 지시 실험",
		"3층 지시 선택 필요",
	} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("workflow controls should not expose terminal UI term %q", forbidden)
		}
	}
	if strings.Contains(string(html), "terminal") || strings.Contains(string(html), "터미널") {
		t.Fatal("workflow controls should not expose a terminal UI term")
	}
}

func TestStaticAppExposesSourceCandidateIndicators(t *testing.T) {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script := []byte(mustReadPlasmaReportScripts(t))
	combined := string(html) + "\n" + string(script) + "\n" + mustReadPlasmaUIScripts(t) + "\n" + mustReadPlasmaWorkflowScripts(t) + "\n" + mustReadPlasmaSourceScripts(t)
	for _, expected := range []string{
		`id="sourceTabCandidateCount"`,
		`id="sourceCandidateNotice"`,
		`id="openSourceCandidatesButton"`,
		"plasma.activeMissionId",
		"updateSourceCandidateIndicators",
		"openSourceCandidatesTab",
		`classList.toggle("hidden", isEmpty)`,
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("expected static app to expose source candidate indicator %q", expected)
		}
	}
}

func TestStaticAppBulkSourceCandidateApprovalUsesURLRouter(t *testing.T) {
	script := []byte(mustReadPlasmaReportScripts(t))
	content := string(script) + mustReadPlasmaSourceScripts(t)
	for _, expected := range []string{
		"function sourceCandidateTitleForURL(url)",
		"await addURLSource(url, sourceCandidateTitleForURL(url), owner)",
		"sourceRouteForURL(url)",
		`if (looksLikeConfluenceURL(value)) return "confluence/url"`,
		"looksLikePDFSourceError(err)",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected bulk source candidate approval to reuse routed URL source addition %q", expected)
		}
	}
	bulkBody := jsFunctionBody(t, content, "bulkSourceCandidateAction")
	if strings.Contains(bulkBody, "/sources/url`") {
		t.Fatalf("bulk source candidate approval must not post every candidate to the generic URL source route")
	}
}

func TestStaticAppSourceCandidateFilterUsesConfluenceLocator(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for semantic static app JS fixture test")
	}
	script := []byte(mustReadPlasmaReportScripts(t))
	content := string(script) + mustReadPlasmaSourceScripts(t)
	nodeScript := jsFunctionSource(t, content, "normalizeSourceURL") + "\n" +
		jsFunctionSource(t, content, "acceptedSourceCandidateKeys") + "\n" +
		jsFunctionSource(t, content, "sourceCandidateAccepted") + "\n" +
		jsFunctionSource(t, content, "sourceLocators") + "\n" +
		jsFunctionSource(t, content, "confluenceCandidateKeyFromURL") + "\n" +
		jsFunctionSource(t, content, "confluenceSourceKey") + `
const sources = [{
  Connector: {
    ExternalSourceID: "site_docs.atlassian.net:123",
    ExternalURI: "confluence://cloud/site_docs.atlassian.net/pages/123"
  },
  Locators: JSON.stringify([{
    site_url: "https://docs.atlassian.net/wiki",
    page_id: "123"
  }])
}];
const existing = acceptedSourceCandidateKeys(sources);
const accepted = sourceCandidateAccepted(existing, normalizeSourceURL("https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap"));
const other = sourceCandidateAccepted(existing, normalizeSourceURL("https://docs.atlassian.net/wiki/spaces/ENG/pages/456/Roadmap"));
process.stdout.write(JSON.stringify({ accepted, other }));
`
	output, err := exec.Command("node", "-e", nodeScript).CombinedOutput()
	if err != nil {
		t.Fatalf("execute Confluence source candidate filter fixture: %v\n%s", err, string(output))
	}
	var got struct {
		Accepted bool `json:"accepted"`
		Other    bool `json:"other"`
	}
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("decode Confluence source candidate filter fixture: %v\n%s", err, string(output))
	}
	if !got.Accepted || got.Other {
		t.Fatalf("expected only the accepted Confluence page candidate to be hidden, got %#v", got)
	}
}

func TestStaticAppSourceRefreshUsesExistingDetailRenderer(t *testing.T) {
	script := []byte(mustReadPlasmaReportScripts(t))
	content := string(script)
	if !strings.Contains(content, "function renderDetail()") {
		t.Fatalf("expected static app to define renderDetail")
	}
	if strings.Contains(content, "renderMissionDetail(") {
		t.Fatalf("static app should not call missing renderMissionDetail")
	}
}

func TestStaticAppHidesReportHumanizeCreateRetry(t *testing.T) {
	content := string(mustReadStatic(t, "static/plasma/reports_cards_artifacts.js")) + "\n" + string(mustReadStatic(t, "static/index.html"))
	for _, removed := range []string{
		"H5 말투 보정 다시 생성",
		"start-humanized-markdown-artifact",
	} {
		if strings.Contains(content, removed) {
			t.Fatalf("static app still exposes report humanize create/retry UI %q", removed)
		}
	}
	for _, historicalAccess := range []string{"보정 Markdown 보기", "보정 MD 받기"} {
		if !strings.Contains(content, historicalAccess) {
			t.Fatalf("historical humanized artifact access was removed %q", historicalAccess)
		}
	}
	script := string(mustReadPlasmaReportScripts(t))
	for _, retained := range []string{"exportReportArtifactHumanizedMarkdown", "/humanized_markdown_export"} {
		if !strings.Contains(script, retained) {
			t.Fatalf("expected static app to retain report humanize compatibility boundary %q", retained)
		}
	}
}

func TestStaticAppTreatsHumanizeSkippedAsTerminalState(t *testing.T) {
	script := []byte(mustReadPlasmaReportScripts(t))
	content := string(script)
	for _, expected := range []string{
		`if (event.EventType === "report.humanize.skipped")`,
		`return { state: "skipped", event };`,
		`if (status.state === "skipped" && wasPending)`,
		`H5 말투 보정 결과가 원본과 같아 별도 artifact를 만들지 않았습니다.`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected static app to treat H5 skipped as a terminal non-error state %q", expected)
		}
	}
}

func TestStaticAppExposesConfluenceSourceWorkflow(t *testing.T) {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	appScript := []byte(mustReadPlasmaReportScripts(t))
	combined := string(html) + "\n" + string(appScript) + "\n" + mustReadPlasmaSourceScripts(t) + "\n" + mustReadPlasmaSettingsScripts(t)
	for _, expected := range []string{
		`id="confluenceSourceDetails"`,
		`data-tab="settings"`,
		`id="confluenceSettingsAPIForm"`,
		`id="confluenceSettingsConnections"`,
		`data-conn-action="rename"`,
		`id="confluenceAccessEnable"`,
		`id="confluenceAccessDisable"`,
		`id="confluenceOneClickStart"`,
		`id="confluenceFlowStatus"`,
		`id="confluenceURLForm"`,
		`id="confluencePageURL"`,
		`id="confluenceAddURLButton"`,
		`https://id.atlassian.com/manage-profile/security/api-tokens`,
		`id="confluenceLoadSpaces"`,
		`id="confluenceLoadMoreSpaces"`,
		`id="confluenceLoadMorePages"`,
		`id="confluenceSpaces"`,
		`id="confluencePages"`,
		`id="confluencePreviewPanel"`,
		`id="confluenceRangeSelect"`,
		`id="confluenceUpdatePanel"`,
		`id="confluenceSearchForm"`,
		`id="confluenceResults"`,
		`/static/plasma/sources_confluence_core.js`,
		`/static/plasma/sources_confluence_errors.js`,
		`/static/plasma/settings_confluence_actions.js`,
		`/static/plasma/sources_confluence_access.js`,
		`/static/plasma/sources_confluence_workflow.js`,
		`/static/plasma/sources_confluence_browse.js`,
		`/static/plasma/sources_confluence_review.js`,
		`/static/plasma/sources_confluence_update.js`,
		`/api/settings/connectors/confluence/connections`,
		`/connector-access/confluence`,
		`/sources/confluence/spaces`,
		`/sources/confluence/space-pages`,
		`/sources/confluence/children`,
		`/sources/confluence/search`,
		`/sources/confluence/url`,
		`/sources/confluence/preview`,
		`/sources/confluence/snapshot`,
		`/sources/confluence/check-update`,
		`/sources/confluence/update-preview`,
		`/sources/confluence/update`,
		`data-confluence-candidate-index`,
		`data-confluence-page-index`,
		`data-confluence-source-update`,
		`confluence_page_range`,
		`clearConfluenceSearchResults`,
		`confluenceSearchContext`,
		`confluenceBrowseContext`,
		`loadMoreConfluenceSpaces`,
		`loadMoreConfluencePages`,
		`spaces_cursor: context.spaces_cursor || ""`,
		"renderConfluenceSpaces(state.confluenceSpaces);\n      renderConfluencePages([]);",
		`previewConfluenceCandidate`,
		`approveConfluenceSnapshot`,
		`preview.full_body_too_large || preview.FullBodyTooLarge`,
		`rangeRequired && !ranges.length`,
		`runConfluenceOneClickFlow`,
		`addConfluenceURLSource`,
		`sourceCandidateTitleForURL(url)`,
		`connection_id: connectionID`,
		`cloud_id: cloudID`,
		`API token 연결 추가`,
		`confluenceSettingsAPIToken").value = ""`,
		`confluenceCandidateDetailPayload(candidate)`,
		`업데이트 검토`,
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("expected static app to expose Confluence workflow %q", expected)
		}
	}
	if strings.Contains(combined, `id="confluenceAPICloudID"`) || strings.Contains(combined, "cloud id가 필요") {
		t.Fatalf("Confluence API token fallback must not ask users for cloud id")
	}
	sourceDetails := htmlSection(t, string(html), `id="confluenceSourceDetails"`, `id="liquid2Form"`)
	for _, forbidden := range []string{`id="confluenceSettingsOAuthForm"`, `id="confluenceSettingsAPIForm"`, `id="confluenceSettingsConnectionDisplayName"`, "Atlassian API token"} {
		if strings.Contains(sourceDetails, forbidden) {
			t.Fatalf("mission Sources must not contain Settings-only Confluence control %q", forbidden)
		}
	}
	if strings.Contains(combined, `id="confluenceSettingsOAuthForm"`) ||
		strings.Contains(combined, `/api/settings/connectors/confluence/oauth/start`) ||
		strings.Contains(combined, `window.open("about:blank", "plasmaConfluenceOAuth")`) {
		t.Fatalf("Confluence OAuth UI must not be exposed in Plasma 0.0")
	}
	for _, forbidden := range []string{`/api/missions/${state.missionId}/sources/confluence/oauth/start`, `/api/missions/${state.missionId}/sources/confluence/connections`, `/api/missions/${state.missionId}/sources/confluence/sites`} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("static UI must not call legacy mission lifecycle route %q", forbidden)
		}
	}
	setFormsBody := jsFunctionBody(t, string(appScript), "setFormsEnabled")
	for _, forbidden := range []string{"confluenceSettingsOAuthForm", "confluenceSettingsAPIForm", "confluenceSettingsAPIToken", "confluenceSettingsOAuthClientSecret"} {
		if strings.Contains(setFormsBody, forbidden) {
			t.Fatalf("global Confluence Settings control %q must not be disabled by mission-bound form state", forbidden)
		}
	}
	if strings.Contains(combined, "cloud ${info.cloud_id}") {
		t.Fatalf("Confluence source metadata must not display the internal cloud id")
	}
	if strings.Contains(combined, "if (info.external_uri) parts.push(info.external_uri)") ||
		!strings.Contains(combined, "confluenceDisplayableExternalURI(info.external_uri)") {
		t.Fatalf("Confluence source metadata must not render raw internal external_uri values")
	}
	if strings.Contains(combined, `data-detail-title="소스 상세" data-detail-json="${escapeAttr(JSON.stringify(source))}"`) ||
		!strings.Contains(combined, "sourceDetailPayload(source, confluence)") {
		t.Fatalf("Confluence source detail modal must use a sanitized user-facing payload")
	}
	confluenceResultsScript := string(mustReadStatic(t, "static/plasma/sources_confluence_results_rendering.js"))
	if strings.Contains(confluenceResultsScript, `data-detail-title="Confluence 검색 결과" data-detail-json="${escapeAttr(JSON.stringify(candidate))}"`) ||
		strings.Contains(confluenceResultsScript, "connector.ExternalURI") ||
		strings.Contains(confluenceResultsScript, "connector.external_uri") {
		t.Fatalf("Confluence search candidate detail must not expose the raw connector payload")
	}
}

func TestConfluenceErrorMessagesAreActionableAndSafe(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for Confluence error mapping fixture test")
	}
	script, err := os.ReadFile("static/plasma/sources_confluence_errors.js")
	if err != nil {
		t.Fatal(err)
	}
	nodeScript := jsFunctionSource(t, string(script), "confluenceErrorDetails") + "\n" +
		jsFunctionSource(t, string(script), "confluenceRetryMessage") + "\n" +
		jsFunctionSource(t, string(script), "confluenceErrorMessage") + `
const cases = [
  ["wrong credentials", { details: { error: { status: 401, message: "backend secret" } } }, "사이트 URL, Atlassian 계정 이메일, API token"],
  ["expired", { details: { error: { code: "confluence_token_expired", status: 401 } } }, "인증이 만료"],
  ["revoked", { details: { error: { code: "confluence_connection_revoked", status: 401 } } }, "연결이 해제"],
  ["forbidden", { details: { error: { category: "confluence_permission" } } }, "접근 권한"],
  ["not found", { details: { error: { status: 404 } } }, "사이트와 페이지 주소"],
  ["rate limited", { details: { error: { code: "confluence_rate_limited", retry_after: "30" } } }, "약 30초 후"],
  ["version drift", { details: { error: { code: "confluence_version_changed" } } }, "새 스냅샷"],
  ["site mismatch", { details: { error: { code: "confluence_cloud_mismatch" } } }, "사이트를 선택"],
  ["page mismatch", { details: { error: { code: "confluence_page_mismatch" } } }, "사이트를 선택"],
  ["too large", { details: { error: { code: "confluence_page_too_large" } } }, "범위를 선택"],
  ["upstream", { details: { error: { category: "confluence_upstream" } } }, "잠시 후"],
  ["network", { isNetworkError: true }, "네트워크 연결"],
  ["generic", { details: { error: { message: "backend secret" } } }, "연결, 사이트, 페이지"]
];
const results = cases.map(([name, err, expected]) => {
  const message = confluenceErrorMessage(err);
  if (!message.includes(expected) || message.includes("backend secret")) {
    throw new Error(name + ": " + message);
  }
  return name;
});
process.stdout.write(JSON.stringify(results));
`
	output, err := exec.Command("node", "-e", nodeScript).CombinedOutput()
	if err != nil {
		t.Fatalf("execute Confluence error mapping fixture: %v\n%s", err, string(output))
	}
	var got []string
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("decode Confluence error mapping fixture: %v\n%s", err, string(output))
	}
	if len(got) != 13 {
		t.Fatalf("expected 13 Confluence error mappings, got %#v", got)
	}
}

func TestConfluenceAsyncFailuresUseActionAwareErrorHelper(t *testing.T) {
	directShowError := regexp.MustCompile(`\bshowError\s*\(`)
	localValidationError := regexp.MustCompile(`\bshowError\s*\(\s*new\s+Error\s*\(\s*"([^"\\]|\\.)*"\s*\)\s*\)`)
	if localValidationError.MatchString(`showError(new Error(error.message))`) {
		t.Fatal("dynamic error content must not qualify as a local validation message")
	}
	files := []string{
		"static/plasma/sources_confluence_flow.js",
		"static/plasma/settings_confluence_actions.js",
		"static/plasma/sources_confluence_access.js",
		"static/plasma/sources_confluence_browse.js",
		"static/plasma/sources_confluence_review.js",
		"static/plasma/sources_confluence_update.js",
	}
	for _, file := range files {
		content := string(mustReadStatic(t, file))
		if got, allowed := len(directShowError.FindAllStringIndex(content, -1)), len(localValidationError.FindAllStringIndex(content, -1)); got != allowed {
			t.Fatalf("Confluence scripts may call showError only with explicit local new Error validation in %s: calls=%d local=%d", file, got, allowed)
		}
		catchCount := strings.Count(content, "} catch (err) {")
		if catchCount == 0 {
			t.Fatalf("expected Confluence async catch path in %s", file)
		}
		if got := strings.Count(content, "showConfluenceError(err)"); got != catchCount {
			t.Fatalf("expected every Confluence catch path in %s to use action-aware helper: catches=%d helpers=%d", file, catchCount, got)
		}
	}
}

func TestConfluenceAPITokenConnectionValidatesCredentialsBeforeRequest(t *testing.T) {
	content := string(mustReadStatic(t, "static/plasma/settings_confluence_actions.js"))
	body := jsFunctionBody(t, content, "connectConfluenceAPIToken")
	for _, expected := range []string{
		`const siteURL = $("confluenceSettingsAPISiteURL").value.trim();`,
		`const accountName = $("confluenceSettingsAPIEmail").value.trim();`,
		`const apiToken = $("confluenceSettingsAPIToken").value.trim();`,
		"Confluence 사이트 URL이 필요합니다.",
		"Atlassian 계정 이메일이 필요합니다.",
		"Atlassian API token이 필요합니다.",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected API token connection validation %q", expected)
		}
	}
	busyIndex := strings.Index(body, "setConfluenceBusy(true)")
	for _, validation := range []string{"if (!siteURL)", "if (!accountName)", "if (!apiToken)"} {
		if index := strings.Index(body, validation); index < 0 || index > busyIndex {
			t.Fatalf("expected %s before request busy state", validation)
		}
	}
}

func TestConfluenceSourceDetailPayloadIsSanitized(t *testing.T) {
	script := []byte(mustReadPlasmaReportScripts(t))
	body := jsFunctionBody(t, string(script)+mustReadPlasmaSourceScripts(t), "sourceDetailPayload")
	for _, expected := range []string{
		`type: "confluence_source"`,
		"snapshot_id:",
		"title:",
		"connector_id:",
		"connector_version:",
		"site_url:",
		"page_id:",
		"version:",
		"retrieval_policy:",
		"state:",
		"confluenceDisplayableExternalURI(confluence.external_uri)",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected Confluence detail payload to include user-facing field %q", expected)
		}
	}
	for _, forbidden := range []string{
		"cloud_id",
		"CloudID",
		"ExternalSourceID",
		"external_source_id",
		"Locators",
		"locators",
		"confluence://",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("Confluence detail payload must not include internal identity field %q", forbidden)
		}
	}
}

func TestConfluenceUpdateStateTextDoesNotClaimDeletion(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for Confluence update state fixture test")
	}
	script := mustReadPlasmaReportScripts(t) + mustReadPlasmaSourceScripts(t)
	domScript := string(mustReadStatic(t, "static/plasma/dom.js"))
	nodeScript := jsFunctionSource(t, domScript, "timeShort") + "\n" +
		jsFunctionSource(t, script, "confluenceUpdateFailureText") + "\n" +
		jsFunctionSource(t, script, "confluenceUpdateText") + `
const values = [
  confluenceUpdateText({ status: "current", checked_at: "", current_version: 7, latest_version: 7 }),
  confluenceUpdateText({ status: "update_available", checked_at: "", current_version: 7, latest_version: 8 }),
  confluenceUpdateText({ status: "check_failed", checked_at: "", error_category: "confluence_not_found" })
];
process.stdout.write(JSON.stringify(values));
`
	output, err := exec.Command("node", "-e", nodeScript).CombinedOutput()
	if err != nil {
		t.Fatalf("execute Confluence update state fixture: %v\n%s", err, string(output))
	}
	var values []string
	if err := json.Unmarshal(output, &values); err != nil {
		t.Fatalf("decode Confluence update state fixture: %v\n%s", err, string(output))
	}
	joined := strings.Join(values, "\n")
	for _, expected := range []string{"v7 최신", "v8 사용 가능", "원본을 찾거나 접근할 수 없음"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected update state text %q in %q", expected, joined)
		}
	}
	if strings.Contains(joined, "삭제") {
		t.Fatalf("not-found observation must not claim source deletion: %q", joined)
	}
}

func TestReportSourceContextRendersOutsideBodyWithoutUsageClaim(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for report source context fixture test")
	}
	script := mustReadPlasmaReportScripts(t) + mustReadPlasmaSourceScripts(t)
	domScript := string(mustReadStatic(t, "static/plasma/dom.js"))
	nodeScript := jsFunctionSource(t, domScript, "escapeHTML") + "\n" +
		jsFunctionSource(t, domScript, "timeShort") + "\n" +
		jsFunctionSource(t, script, "confluenceUpdateFailureText") + "\n" +
		"const window={Plasma:{sources:{confluenceUpdateFailureText}}};\n" +
		jsSourceRange(t, script, "function reportGenerationContext", "function renderReports") + `
const context = {
  captured_at: "2026-07-14T01:02:03Z",
  confluence_sources: [
    { title: "Roadmap", snapshot_version: "7", snapshot_captured_at: "2026-07-13T01:00:00Z", external_updated_at: "2026-07-12T01:00:00Z", last_check: { status: "update_available", checked_at: "2026-07-14T00:00:00Z", latest_version: 8 } },
    { title: "Restricted", snapshot_version: "2", last_check: { status: "check_failed", error_category: "confluence_not_found" } }
  ]
};
process.stdout.write(JSON.stringify({
  rendered: reportSourceContextHTML({ source_context: context }),
  empty: reportSourceContextHTML({ source_context: { captured_at: context.captured_at, confluence_sources: [] } }),
  legacy: reportSourceContextHTML({})
}));
`
	output, err := exec.Command("node", "-e", nodeScript).CombinedOutput()
	if err != nil {
		t.Fatalf("execute report source context fixture: %v\n%s", err, string(output))
	}
	var result struct {
		Rendered string `json:"rendered"`
		Empty    string `json:"empty"`
		Legacy   string `json:"legacy"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode report source context fixture: %v\n%s", err, string(output))
	}
	for _, expected := range []string{"생성 시점의 소스 정보", "Roadmap", "저장 v7", "v8 사용 가능", "원본을 찾거나 접근할 수 없음"} {
		if !strings.Contains(result.Rendered, expected) {
			t.Fatalf("missing report source context text %q: %s", expected, result.Rendered)
		}
	}
	for _, forbidden := range []string{"사용한 소스", "인용 근거", "삭제"} {
		if strings.Contains(result.Rendered, forbidden) {
			t.Fatalf("report source context made forbidden claim %q: %s", forbidden, result.Rendered)
		}
	}
	if !strings.Contains(result.Empty, "사용 가능한 Confluence 소스가 없었습니다") || result.Legacy != "" {
		t.Fatalf("empty or legacy context behavior changed: empty=%q legacy=%q", result.Empty, result.Legacy)
	}
}

func TestConfluenceSourceDetailPayloadFixtureIsSanitized(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for semantic static app JS fixture test")
	}
	script := []byte(mustReadPlasmaReportScripts(t))
	nodeScript := jsFunctionSource(t, string(script)+mustReadPlasmaSourceScripts(t), "confluenceDisplayableExternalURI") + "\n" +
		jsFunctionSource(t, string(script)+mustReadPlasmaSourceScripts(t), "sourceDetailPayload") + `
const source = {
  SnapshotID: "src_1",
  Title: "Roadmap",
  Connector: {
    ConnectorID: "confluence",
    ConnectorVersion: "v1",
    ExternalSourceID: "site_docs.atlassian.net:123",
    ExternalURI: "confluence://cloud/site_docs.atlassian.net/pages/123"
  },
  Locators: JSON.stringify([{
    cloud_id: "site_docs.atlassian.net",
    site_url: "https://docs.atlassian.net/wiki",
    page_id: "123"
  }]),
  Access: { RetrievalPolicy: "snapshot_only" },
  State: { State: "active" }
};
const confluence = {
  site_url: "https://docs.atlassian.net/wiki",
  page_id: "123",
  version: "7",
  external_uri: "confluence://cloud/site_docs.atlassian.net/pages/123"
};
process.stdout.write(JSON.stringify(sourceDetailPayload(source, confluence)));
`
	output, err := exec.Command("node", "-e", nodeScript).CombinedOutput()
	if err != nil {
		t.Fatalf("execute sourceDetailPayload fixture: %v\n%s", err, string(output))
	}
	var detail map[string]any
	if err := json.Unmarshal(output, &detail); err != nil {
		t.Fatalf("decode sourceDetailPayload fixture result: %v\n%s", err, string(output))
	}
	for key, expected := range map[string]string{
		"type":             "confluence_source",
		"snapshot_id":      "src_1",
		"title":            "Roadmap",
		"connector_id":     "confluence",
		"site_url":         "https://docs.atlassian.net/wiki",
		"page_id":          "123",
		"version":          "7",
		"retrieval_policy": "snapshot_only",
		"state":            "active",
	} {
		if got, _ := detail[key].(string); got != expected {
			t.Fatalf("expected sanitized detail field %s=%q, got %#v in %#v", key, expected, detail[key], detail)
		}
	}
	raw := string(output)
	for _, forbidden := range []string{
		"cloud_id",
		"CloudID",
		"ExternalSourceID",
		"external_source_id",
		"ExternalURI",
		"external_uri",
		"Locators",
		"locators",
		"confluence://",
		"site_docs.atlassian.net:123",
	} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("sanitized detail payload leaked internal field/value %q: %s", forbidden, raw)
		}
	}
}

func TestSourceReadingChunksAndConfluenceScopeFixture(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for source reading fixture test")
	}
	fixture := `
const fs = require("fs");
const vm = require("vm");
function assert(ok, message) { if (!ok) throw new Error(message); }
function node(id) {
  const n = {
    id, textContent:"", innerHTML:"", disabled:false, classList:{add(){},remove(){},contains(){return false;}},
    querySelector(selector) {
      if (selector === "[data-source-read-more]" && this.innerHTML.includes("data-source-read-more")) {
        return {disabled:this.innerHTML.includes("disabled"), addEventListener(type, fn){ n.readMoreHandler = fn; }};
      }
      return null;
    }
  };
  return n;
}
const nodes = {detailTitle:node("detailTitle"), detailBody:node("detailBody"), detailModal:node("detailModal"), sourceList:node("sourceList")};
nodes.detailModal.classList = {remove(name){ nodes.detailModal.hiddenRemoved = name; }, add(){}, contains(){return false;}};
nodes.detailModal.querySelector = () => ({classList:{toggle(){}, remove(){}}});
const document = {getElementById:(id)=>nodes[id]||null, querySelector:()=>node("q")};
const requests = [];
let responses = [];
const state = {missionId:"mis_a", selectionGeneration:1, detail:{sources:[]}};
const $ = (id) => nodes[id] || null;
const escapeHTML = (value) => String(value).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
const escapeAttr = escapeHTML;
const missionApi = (owner, path) => {
  requests.push({owner:{...owner}, path});
  const next = responses.shift();
  if (!next) return Promise.reject(new Error("missing response"));
  if (next.defer) return new Promise((resolve, reject) => { next.resolve = resolve; next.reject = reject; });
  if (next.reject) return Promise.reject(next.reject);
  return Promise.resolve(next.value);
};
const captureMissionSelection = () => ({missionId:state.missionId, selectionGeneration:state.selectionGeneration});
const ownsMissionSelection = (owner) => owner.missionId === state.missionId && owner.selectionGeneration === state.selectionGeneration;
class StaleMissionOperationError extends Error {}
const isStaleMissionOperation = (err) => err instanceof StaleMissionOperationError;
let errors = 0;
const Plasma = {
  state,
  dom: {$, escapeHTML, escapeAttr, timeShort:()=>"", formatBytes:()=>""},
  ui: {showDetail(title, value){ state.genericDetail = {title, value}; }, openDetailModal(){ nodes.detailModal.opened = true; }, empty:()=>"", updateCountChip(){}, setSectionEmpty(){}, showError(){ errors++; }},
  transport: {missionApi},
  mission: {captureMissionSelection, ownsMissionSelection, isStaleMissionOperation}
};
const window = {Plasma};
globalThis.window = window;
globalThis.document = document;
for (const script of [
  "static/plasma/sources.js",
  "static/plasma/sources_confluence_locators.js",
  "static/plasma/sources_reading.js"
]) {
  vm.runInThisContext(fs.readFileSync(script, "utf8"), {filename:script});
}
Object.assign(window.Plasma.sources, {
  localPathLocator:()=>null,
  mediaLocator:()=>null,
  pdfLocator:()=>null,
  documentLocator:()=>null,
  mediaSourceLabel:()=>"",
  documentSourceText:()=>"",
  pdfSourceText:()=>"",
  mediaSourceText:()=>"",
  sourceDetailPayload:(source)=>source,
  confluenceUpdateState:()=>null,
  confluenceUpdateText:()=>"",
  dependency(name) {
    if (name === "requireMission") return () => true;
    if (name === "reloadMission") return async () => {};
    if (name === "showError") return () => { errors++; };
    return () => {};
  },
  sourceLocatorType(locator) { return locator.locator_type || locator.LocatorType || locator.kind || locator.Kind || ""; }
});
vm.runInThisContext(fs.readFileSync("static/plasma/sources_rendering.js", "utf8"), {filename:"static/plasma/sources_rendering.js"});
(async () => {
  responses = [{value:{content:"AAA", next_offset:5, truncated:true}}];
  await window.Plasma.sources.readSource("snap_1");
  assert(requests.at(-1).path === "/sources/snap_1/read?offset=0&max_bytes=20000", "first read must use explicit offset and max_bytes");
  assert(state.detailText === "AAA", "first chunk detail text changed");
  assert(nodes.detailBody.innerHTML.includes("저장된 내용이 더 있습니다.") && nodes.detailBody.innerHTML.includes("더 보기"), "truncated status or button missing");

  const appendResponse = {defer:true};
  responses = [appendResponse];
  const firstAppend = window.Plasma.sources.loadMoreSourceReading();
  const duplicateAppend = window.Plasma.sources.loadMoreSourceReading();
  assert(requests.at(-1).path === "/sources/snap_1/read?offset=5&max_bytes=20000", "append must use returned next_offset");
  assert(requests.filter((request) => request.path.includes("offset=5")).length === 1, "duplicate click was not suppressed");
  assert(nodes.detailBody.innerHTML.includes("불러오는 중…") && nodes.detailBody.innerHTML.includes("disabled"), "loading button state missing");
  appendResponse.resolve({content:"BBB", next_offset:8, truncated:true});
  await firstAppend; await duplicateAppend;
  assert(state.detailText === "AAABBB", "append did not accumulate content");

  responses = [{reject:new Error("later failure")}];
  await window.Plasma.sources.loadMoreSourceReading();
  assert(state.detailText === "AAABBB" && nodes.detailBody.innerHTML.includes("저장된 내용이 더 있습니다.") && errors === 1, "failure did not retain prior content");

  const staleResponse = {defer:true};
  responses = [staleResponse];
  const staleRead = window.Plasma.sources.readSource("stale_snap");
  state.missionId = "mis_b"; state.selectionGeneration = 2;
  staleResponse.resolve({content:"STALE", next_offset:9, truncated:true});
  await staleRead;
  assert(state.detailText === "AAABBB", "stale mission response mutated detail text");

  state.missionId = "mis_a"; state.selectionGeneration = 1;
  state.sourceReading.loading = false;
  responses = [{value:{content:"CCC", next_offset:11, truncated:false}}];
  await window.Plasma.sources.loadMoreSourceReading();
  assert(state.detailText === "AAABBBCCC" && nodes.detailBody.innerHTML.includes("저장된 내용을 모두 불러왔습니다.") && !nodes.detailBody.innerHTML.includes("더 보기"), "final untruncated state changed");

  const full = {Connector:{ConnectorID:"confluence", ConnectorType:"confluence_cloud", ExternalVersion:"7"}, Locators:JSON.stringify([{locator_type:"confluence_page_body", site_url:"https://docs.example/wiki", page_id:"123", start:0, end:500}])};
  const range = {Title:"Roadmap (Confluence range 10-30)", Connector:{ConnectorID:"confluence", ConnectorType:"confluence_cloud"}, Locators:JSON.stringify([{locator_type:"confluence_page_range", site_url:"https://docs.example/wiki", page_id:"123", start:10, end:30, partial:true}])};
  assert(window.Plasma.sources.confluenceSourceText(window.Plasma.sources.confluenceSourceInfo(full)).includes("전체 페이지"), "full Confluence label missing");
  const rangeInfo = window.Plasma.sources.confluenceSourceInfo(range);
  assert(window.Plasma.sources.confluenceSourceText(rangeInfo).includes("선택 범위 / 원문 문자 11–30"), "range Confluence label missing");
  window.Plasma.sources.renderSources([range]);
  assert(nodes.sourceList.innerHTML.includes("Roadmap") && nodes.sourceList.innerHTML.includes("선택 범위"), "generated title cleanup or range badge missing");
  assert(!nodes.sourceList.innerHTML.includes("Roadmap (Confluence range 10-30)\n          <span"), "generated title suffix was not cleaned from visible title");
})().catch((err) => { console.error(err); process.exit(1); });
`
	if output, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("execute source reading fixture: %v\n%s", err, string(output))
	}
}

func TestPDFLocatorRecognizesUploadedPDFSource(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for semantic static app JS fixture test")
	}
	script := []byte(mustReadPlasmaReportScripts(t))
	nodeScript := jsFunctionSource(t, string(script)+mustReadPlasmaSourceScripts(t), "sourceLocatorType") + "\n" +
		jsFunctionSource(t, string(script)+mustReadPlasmaSourceScripts(t), "pdfLocator") + `
const canonical = {
  Locators: JSON.stringify([{
    locator_type: "pdf_document",
    original_filename: "Paper Final.pdf",
    sanitized_filename: "Paper-Final.pdf",
    mime_type: "application/pdf",
    byte_size: 2048,
    content_kind: "pdf",
    extraction_support: "pdf_text"
  }])
};
const legacy = {
  Locators: JSON.stringify([{
    kind: "file_upload",
    original_filename: "Legacy Paper.pdf",
    sanitized_filename: "Legacy-Paper.pdf",
    media_type: "application/pdf",
    byte_size: 1024,
    content_kind: "pdf"
  }])
};
process.stdout.write(JSON.stringify({
  canonical: pdfLocator(canonical),
  legacy: pdfLocator(legacy)
}));
`
	output, err := exec.Command("node", "-e", nodeScript).CombinedOutput()
	if err != nil {
		t.Fatalf("execute pdfLocator fixture: %v\n%s", err, string(output))
	}
	var result map[string]map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode pdfLocator fixture result: %v\n%s", err, string(output))
	}
	if got, _ := result["canonical"]["filename"].(string); got != "Paper-Final.pdf" {
		t.Fatalf("expected canonical uploaded PDF filename, got %#v in %#v", result["canonical"]["filename"], result)
	}
	if got, _ := result["canonical"]["extraction_support"].(string); got != "pdf_text" {
		t.Fatalf("expected canonical uploaded PDF extraction support, got %#v in %#v", result["canonical"]["extraction_support"], result)
	}
	if got, _ := result["legacy"]["filename"].(string); got != "Legacy-Paper.pdf" {
		t.Fatalf("expected legacy uploaded PDF filename, got %#v in %#v", result["legacy"]["filename"], result)
	}
	if got, _ := result["legacy"]["mime_type"].(string); got != "application/pdf" {
		t.Fatalf("expected legacy uploaded PDF MIME type, got %#v in %#v", result["legacy"]["mime_type"], result)
	}
}

func TestUploadedFileLegacyLocatorsRenderAsMediaOrDocument(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for semantic static app JS fixture test")
	}
	script := []byte(mustReadPlasmaReportScripts(t))
	nodeScript := jsFunctionSource(t, string(script)+mustReadPlasmaSourceScripts(t), "sourceLocatorType") + "\n" +
		jsFunctionSource(t, string(script)+mustReadPlasmaSourceScripts(t), "sourceConnectorType") + "\n" +
		jsFunctionSource(t, string(script)+mustReadPlasmaSourceScripts(t), "uploadedFileContentKind") + "\n" +
		jsFunctionSource(t, string(script)+mustReadPlasmaSourceScripts(t), "uploadedFileMediaType") + "\n" +
		jsFunctionSource(t, string(script)+mustReadPlasmaSourceScripts(t), "uploadedFileFilename") + "\n" +
		jsFunctionSource(t, string(script)+mustReadPlasmaSourceScripts(t), "mediaLocator") + "\n" +
		jsFunctionSource(t, string(script)+mustReadPlasmaSourceScripts(t), "documentLocator") + `
const legacyImage = {
  Connector: { ConnectorType: "file_upload" },
  Locators: JSON.stringify([{
    kind: "file_upload",
    original_filename: "Legacy Pixel.png",
    sanitized_filename: "Legacy-Pixel.png",
    media_type: "image/png",
    byte_size: 256,
    content_kind: "image"
  }])
};
const legacyText = {
  Connector: { ConnectorType: "file_upload" },
  Locators: JSON.stringify([{
    kind: "file_upload",
    original_filename: "Legacy Notes.md",
    sanitized_filename: "Legacy-Notes.md",
    media_type: "text/markdown",
    byte_size: 128,
    content_kind: "text"
  }])
};
process.stdout.write(JSON.stringify({
  image: mediaLocator(legacyImage),
  text: documentLocator(legacyText)
}));
`
	output, err := exec.Command("node", "-e", nodeScript).CombinedOutput()
	if err != nil {
		t.Fatalf("execute uploaded locator fixture: %v\n%s", err, string(output))
	}
	var result map[string]map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode uploaded locator fixture result: %v\n%s", err, string(output))
	}
	if got, _ := result["image"]["media_kind"].(string); got != "image" {
		t.Fatalf("expected legacy uploaded image media kind, got %#v in %#v", result["image"]["media_kind"], result)
	}
	if got, _ := result["image"]["filename"].(string); got != "Legacy-Pixel.png" {
		t.Fatalf("expected legacy uploaded image filename, got %#v in %#v", result["image"]["filename"], result)
	}
	if got, _ := result["text"]["filename"].(string); got != "Legacy-Notes.md" {
		t.Fatalf("expected legacy uploaded text filename, got %#v in %#v", result["text"]["filename"], result)
	}
	if got, _ := result["text"]["mime_type"].(string); got != "text/markdown" {
		t.Fatalf("expected legacy uploaded text MIME type, got %#v in %#v", result["text"]["mime_type"], result)
	}
}

func TestConfluenceCandidateDetailPayloadIsSanitized(t *testing.T) {
	script := []byte(mustReadPlasmaSourceScripts(t))
	body := jsFunctionBody(t, string(script)+mustReadPlasmaSourceScripts(t), "confluenceCandidateDetailPayload")
	for _, expected := range []string{
		`type: "confluence_candidate"`,
		"title",
		"site_url",
		"site_host",
		"page_id",
		"space_key",
		"version",
		"updated_at",
		"can_snapshot",
		"confluenceDisplayableExternalURI",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected Confluence candidate detail payload to include user-facing field %q", expected)
		}
	}
	for _, forbidden := range []string{
		"cloud_id",
		"CloudID",
		"Connector",
		"connector",
		"ExternalSourceID",
		"external_source_id",
		"confluence://",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("Confluence candidate detail payload must not include internal identity field %q", forbidden)
		}
	}
}

func TestConfluenceCandidateDetailPayloadFixtureIsSanitized(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for semantic static app JS fixture test")
	}
	appScript := []byte(mustReadPlasmaReportScripts(t))
	confluenceScript := []byte(mustReadPlasmaSourceScripts(t))
	browseScript, err := os.ReadFile("static/plasma/sources_confluence_browse.js")
	if err != nil {
		t.Fatal(err)
	}
	nodeScript := jsFunctionSource(t, string(appScript)+mustReadPlasmaSourceScripts(t), "confluenceDisplayableExternalURI") + "\n" +
		jsFunctionSource(t, string(appScript)+mustReadPlasmaSourceScripts(t), "confluenceExternalURIHost") + "\n" +
		jsFunctionSource(t, string(browseScript)+mustReadPlasmaSourceScripts(t), "confluenceCandidatePageID") + "\n" +
		jsFunctionSource(t, string(confluenceScript), "confluenceCandidateDetailPayload") + `
const candidate = {
  CloudID: "site_docs.atlassian.net",
  SiteURL: "https://docs.atlassian.net/wiki",
  SpaceKey: "ENG",
  Title: "Roadmap",
  SourceURI: "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap",
  Version: 7,
  UpdatedAt: "2026-07-06T01:02:03Z",
  CanSnapshot: true,
  Connector: {
    ExternalSourceID: "site_docs.atlassian.net:123",
    ExternalURI: "confluence://cloud/site_docs.atlassian.net/pages/123"
  }
};
process.stdout.write(JSON.stringify(confluenceCandidateDetailPayload(candidate)));
`
	output, err := exec.Command("node", "-e", nodeScript).CombinedOutput()
	if err != nil {
		t.Fatalf("execute confluenceCandidateDetailPayload fixture: %v\n%s", err, string(output))
	}
	var detail map[string]any
	if err := json.Unmarshal(output, &detail); err != nil {
		t.Fatalf("decode confluenceCandidateDetailPayload fixture result: %v\n%s", err, string(output))
	}
	for key, expected := range map[string]string{
		"type":       "confluence_candidate",
		"title":      "Roadmap",
		"site_url":   "https://docs.atlassian.net/wiki",
		"site_host":  "docs.atlassian.net",
		"page_id":    "123",
		"space_key":  "ENG",
		"updated_at": "2026-07-06T01:02:03Z",
		"source_uri": "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap",
	} {
		if got, _ := detail[key].(string); got != expected {
			t.Fatalf("expected sanitized candidate detail field %s=%q, got %#v in %#v", key, expected, detail[key], detail)
		}
	}
	if got, _ := detail["version"].(float64); got != 7 {
		t.Fatalf("expected sanitized candidate version 7, got %#v", detail["version"])
	}
	if got, _ := detail["can_snapshot"].(bool); !got {
		t.Fatalf("expected sanitized candidate can_snapshot true, got %#v", detail["can_snapshot"])
	}
	raw := string(output)
	for _, forbidden := range []string{
		"cloud_id",
		"CloudID",
		"Connector",
		"connector",
		"ExternalSourceID",
		"external_source_id",
		"ExternalURI",
		"external_uri",
		"confluence://",
		"site_docs.atlassian.net",
	} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("sanitized candidate detail payload leaked internal field/value %q: %s", forbidden, raw)
		}
	}
}

func TestConfluenceDeleteConnectionSendsJSONBody(t *testing.T) {
	script, err := os.ReadFile("static/plasma/settings_confluence_actions.js")
	if err != nil {
		t.Fatal(err)
	}
	content := string(script)
	functionStart := strings.Index(content, `async function deleteConfluenceSettingsConnection(connectionID)`)
	if functionStart < 0 {
		t.Fatalf("expected delete connection function in Confluence settings script")
	}
	deletePath := `/api/settings/connectors/confluence/connections/${encodeURIComponent(connectionID)}`
	start := strings.Index(content[functionStart:], deletePath)
	if start < 0 {
		t.Fatalf("expected delete connection path in Confluence settings script")
	}
	start += functionStart
	end := strings.Index(content[start:], `clearConfluenceDiscovery();`)
	if end < 0 {
		t.Fatalf("expected delete connection call before discovery clear")
	}
	deleteCall := content[start : start+end]
	for _, expected := range []string{
		`method: "DELETE"`,
		`body: {}`,
	} {
		if !strings.Contains(deleteCall, expected) {
			t.Fatalf("expected delete connection call to include %q, got:\n%s", expected, deleteCall)
		}
	}
}

func TestStaticAppDistinguishesAgentTerminalTurns(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := string(mustReadStatic(t, "static/plasma/conversation_turns.js"))
	fixture := `
const state={pendingTurn:null,turnPending:false,missionId:"mis_1",turnScrollMission:"mis_1"};
const log={scrollHeight:0,scrollTop:0,clientHeight:0,innerHTML:""};
const window={renderPlasmaMath(){}};
const $=()=>log;
const completedUserEventIDs=()=>new Set();
const conversation={completedUserEventIDs,updateTurnNavVisibility(){}};
const escapeHTML=(value)=>String(value);
const escapeAttr=(value)=>String(value);
const timeShort=()=>"12:00";
const shortID=(value)=>String(value);
const renderMarkdown=(value)=>String(value);
const empty=(value)=>String(value);
const callbacks={renderMarkdown, empty};
` + jsFunctionSource(t, script, "renderSessionBadge") + `
` + jsFunctionSource(t, script, "renderSessionResetTurn") + `
` + jsFunctionSource(t, script, "renderStandaloneSteeringTurn") + `
` + jsFunctionSource(t, script, "renderConversationTurn") + `
` + jsFunctionSource(t, script, "renderTurns") + `
renderTurns([
  {EventType:"turn.agent.response",CreatedAt:"now",Payload:{kind:"agent_error",text:"실패했습니다",agent_executor:"codex"}},
  {EventType:"turn.agent.response",CreatedAt:"now",Payload:{kind:"agent_canceled",text:"취소했습니다",agent_executor:"codex"}},
  {EventType:"turn.agent.response",CreatedAt:"now",Payload:{kind:"agent_response",text:"완료했습니다",agent_executor:"codex"}},
]);
if((log.innerHTML.match(/응답 실패/g)||[]).length!==1)throw new Error("failure badge is missing or duplicated");
if((log.innerHTML.match(/응답 취소/g)||[]).length!==1)throw new Error("canceled badge is missing or duplicated");
if((log.innerHTML.match(/badge danger/g)||[]).length!==1)throw new Error("failure badge style is missing");
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("agent terminal turn fixture: %v: %s", err, out)
	}
}

func TestStaticAppRendersAgentCompactionTurn(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := string(mustReadStatic(t, "static/plasma/conversation_turns.js"))
	fixture := `
const state={pendingTurn:null,turnPending:false,missionId:"mis_1",turnScrollMission:"mis_1"};
const log={scrollHeight:0,scrollTop:0,clientHeight:0,innerHTML:""};
const window={Plasma:{reports:{}},renderPlasmaMath(){}};
const $=()=>log;
const completedUserEventIDs=()=>new Set();
const conversation={completedUserEventIDs,updateTurnNavVisibility(){}};
const escapeHTML=(value)=>String(value);
const escapeAttr=(value)=>String(value);
const timeShort=()=>"12:00";
const shortID=(value)=>String(value);
const callbacks={renderMarkdown:(value)=>String(value),empty:(value)=>String(value)};
` + jsFunctionSource(t, script, "contextWindowPercent") + `
` + jsFunctionSource(t, script, "renderContextCompactionTurn") + `
` + jsFunctionSource(t, script, "renderSessionBadge") + `
` + jsFunctionSource(t, script, "renderSessionResetTurn") + `
` + jsFunctionSource(t, script, "renderStandaloneSteeringTurn") + `
` + jsFunctionSource(t, script, "renderConversationTurn") + `
` + jsFunctionSource(t, script, "renderTurns") + `
renderTurns([
  {EventType:"turn.agent.compacted",CreatedAt:"now",Payload:{
    kind:"agent_session_compacted",manual:false,context_used_tokens:222425,context_window_tokens:258400,
    agent_usage:{context_window:{used_tokens:76769,window_tokens:258400}},summary:"provider-only detail"
  }},
  {EventType:"turn.agent.compacted",CreatedAt:"now",Payload:{kind:"agent_session_compacted",manual:true}}
]);
const html=log.innerHTML;
if((html.match(/compaction-event/g)||[]).length!==2)throw new Error("compaction events are missing: "+html);
if(!html.includes("자동 압축")||!html.includes("압축 완료"))throw new Error("compaction labels are missing: "+html);
if(!html.includes("86.1%")||!html.includes("29.7%"))throw new Error("compaction range is missing: "+html);
if(!html.includes("같은 세션에서 작업을 이어갑니다."))throw new Error("continuity message is missing: "+html);
if(html.includes("provider-only detail")||html.includes("turn-copy"))throw new Error("internal detail or copy action leaked: "+html);
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("agent compaction turn fixture: %v: %s", err, out)
	}
}

func TestStaticAppRenderTurnsGroupsMatchedSteeringMetadata(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := string(mustReadStatic(t, "static/plasma/conversation_turns.js"))
	domScript := string(mustReadStatic(t, "static/plasma/dom.js"))
	fixture := `
const state={pendingTurn:null,turnPending:false,missionId:"mis_1",turnScrollMission:"mis_1"};
const log={scrollHeight:0,scrollTop:0,clientHeight:0,innerHTML:""};
const window={renderPlasmaMath(){},renderPlasmaMermaid(){},enhancePlasmaImageViewing(){}};
const $=()=>log;
const completedUserEventIDs=()=>new Set();
const conversation={completedUserEventIDs,updateTurnNavVisibility(){}};
const timeShort=()=>"12:00";
const shortID=(value)=>String(value);
const renderMarkdown=(value)=>String(value);
const empty=(value)=>String(value);
const callbacks={renderMarkdown, empty};
` + jsFunctionSource(t, domScript, "escapeHTML") + `
` + jsFunctionSource(t, domScript, "escapeAttr") + `
` + jsFunctionSource(t, script, "renderSessionBadge") + `
` + jsFunctionSource(t, script, "renderSessionResetTurn") + `
` + jsFunctionSource(t, script, "renderStandaloneSteeringTurn") + `
` + jsFunctionSource(t, script, "renderConversationTurn") + `
` + jsFunctionSource(t, script, "renderTurns") + `
renderTurns([
  {EventID:"user-1",EventType:"turn.user",CreatedAt:"now",Payload:{text:"첫 요청"}},
  {EventID:"steer-1",EventType:"controller.strategy.selected",CreatedAt:"now",Payload:{user_event_id:"user-1",strategy_label:"V2 conservative",strategy_id:"v2_conservative",reason:"English selection reason"}},
  {EventID:"steer-2",EventType:"controller.strategy.selected",CreatedAt:"now",Payload:{user_event_id:"missing-user",strategy_label:"단독 표시",strategy_id:"standalone_v3",reason:"연결 없음"}},
  {EventID:"steer-3",EventType:"controller.strategy.selected",CreatedAt:"now",Payload:{user_event_id:"user-1",strategy_label:"V3 broadening",strategy_id:"v3_broadening",reason:"Do not show this"}},
  {EventID:"steer-4",EventType:"controller.strategy.selected",CreatedAt:"now",Payload:{user_event_id:"user-1",strategy_id:"sid\"<x>",reason:"두 번째 & <이유>"}},
  {EventID:"agent-1",EventType:"turn.agent.response",CreatedAt:"now",Payload:{kind:"agent_response",text:"응답",agent_executor:"codex"}},
]);
const html=log.innerHTML;
const count=(needle)=>(html.match(new RegExp(needle.replace(/[.*+?^${}()|[\]\\]/g,"\\$&"),"g"))||[]).length;
if(count("turn-steering-meta")!==3)throw new Error("matched steering metadata count is wrong: "+html);
if(count("class=\"turn controller\"")!==1)throw new Error("standalone controller count is wrong: "+html);
const userStart=html.indexOf("class=\"turn user");
const userEnd=html.indexOf("class=\"turn controller\"");
if(userStart<0||userEnd<0||html.indexOf("turn-steering-meta",userStart)>userEnd)throw new Error("matched steering did not render inside owning user card: "+html);
const userHTML=html.slice(userStart,userEnd);
if(count("자동 조향")!==3)throw new Error("matched steering labels were dropped or duplicated: "+html);
if(!userHTML.includes("V2 conservative")||!userHTML.includes("V3 broadening"))throw new Error("matched steering labels are missing: "+html);
if(userHTML.includes("v2_conservative")||userHTML.includes("v3_broadening")||userHTML.includes("English selection reason")||userHTML.includes("Do not show this"))throw new Error("matched steering rendered hidden ID or reason: "+html);
if(!html.includes("standalone_v3")||!html.includes("연결 없음"))throw new Error("unmatched steering lost standalone content: "+html);
if(!userHTML.includes("sid&quot;&lt;x&gt;"))throw new Error("matched fallback strategy ID was not rendered escaped: "+html);
if(userHTML.includes("sid\"<x>")||userHTML.includes("두 번째 & <이유>")||userHTML.includes("두 번째 &amp; &lt;이유&gt;"))throw new Error("matched fallback leaked raw string or reason: "+html);
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("steering metadata fixture: %v: %s", err, out)
	}
}

func TestStaticAppSteeringMetadataCSSContract(t *testing.T) {
	styles := mustReadAppCSSComposed(t)
	for _, expected := range []string{
		".turn-steering-meta",
		"display: flex;",
		"align-items: baseline;",
		"gap: 6px;",
		"flex-wrap: nowrap;",
		"min-width: 0;",
		"margin-top: 8px;",
		"padding-top: 7px;",
		"border-top: 1px solid color-mix(in srgb, var(--amber) 24%, transparent);",
		"font-size: 11.5px;",
		"line-height: 1.45;",
		"color: var(--muted);",
		".turn-steering-label",
		"color: var(--amber);",
		"font-weight: 700;",
		".turn-steering-text strong",
		"color: var(--ink2);",
		"font-weight: 600;",
		".turn-steering-text",
		"overflow: hidden;",
		"text-overflow: ellipsis;",
		"white-space: nowrap;",
	} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing steering metadata CSS contract %q", expected)
		}
	}
}

func TestStaticAppReportsPreservedMarkdownAfterPatchFailure(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	fixture := `
let notice={};
const setReportBusy=()=>{};
const setReportNotice=(text,kind)=>{notice={text,kind};};
const reportTimingDetails=()=>"";
const reports = {setReportBusy,setReportNotice,reportTimingDetails};
` + jsFunctionSource(t, script, "renderReportDraftStatus") + `
renderReportDraftStatus({state:"failed",event:{EventType:"report.patch.failed",Payload:{error:"패치 실패"}}},true);
if(!notice.text.includes("패치 실패")||!notice.text.includes("원본 Markdown 리포트는 유지되었습니다."))throw new Error("patch preservation notice is missing");
if(notice.kind!=="error")throw new Error("patch failure lost its error state");
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("report patch failure fixture: %v: %s", err, out)
	}
}

func jsFunctionSource(t *testing.T, content string, name string) string {
	t.Helper()
	start, end := jsFunctionBounds(t, content, name)
	return content[start:end]
}

func jsSourceRange(t *testing.T, content, startMarker, endMarker string) string {
	t.Helper()
	start := strings.Index(content, startMarker)
	if start < 0 {
		t.Fatalf("expected JavaScript marker %q", startMarker)
	}
	end := strings.Index(content[start:], endMarker)
	if end < 0 {
		t.Fatalf("expected JavaScript marker %q after %q", endMarker, startMarker)
	}
	return content[start : start+end]
}

func htmlSection(t *testing.T, content string, startMarker string, endMarker string) string {
	t.Helper()
	start := strings.Index(content, startMarker)
	if start < 0 {
		t.Fatalf("expected HTML marker %q", startMarker)
	}
	end := strings.Index(content[start:], endMarker)
	if end < 0 {
		t.Fatalf("expected HTML marker %q after %q", endMarker, startMarker)
	}
	return content[start : start+end]
}

func jsFunctionBody(t *testing.T, content string, name string) string {
	t.Helper()
	_, end := jsFunctionBounds(t, content, name)
	start := strings.Index(content, "function "+name+"(")
	if start < 0 {
		t.Fatalf("expected function %s in static JavaScript content", name)
	}
	brace := strings.Index(content[start:], "{")
	if brace < 0 {
		t.Fatalf("expected function %s body", name)
	}
	bodyStart := start + brace
	depth := 0
	for i := bodyStart; i < end; i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return content[bodyStart+1 : i]
			}
		}
	}
	t.Fatalf("function %s body did not terminate", name)
	return ""
}

func jsFunctionBounds(t *testing.T, content string, name string) (int, int) {
	t.Helper()
	start := strings.Index(content, "function "+name+"(")
	if start < 0 {
		t.Fatalf("expected function %s in static JavaScript content", name)
	}
	brace := strings.Index(content[start:], "{")
	if brace < 0 {
		t.Fatalf("expected function %s body", name)
	}
	bodyStart := start + brace
	depth := 0
	for i := bodyStart; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return start, i + 1
			}
		}
	}
	t.Fatalf("function %s body did not terminate", name)
	return 0, 0
}

func TestConfluenceCommonRendererDoesNotOwnPreviewApprovalButtons(t *testing.T) {
	files := []string{
		"static/app.js",
		"static/plasma/sources_confluence_rendering.js",
	}
	for _, file := range files {
		script, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		content := string(script)
		start := strings.Index(content, `for (const id of [`)
		if start < 0 {
			t.Fatalf("expected common control loop in %s", file)
		}
		end := strings.Index(content[start:], `]) {`)
		if end < 0 {
			t.Fatalf("expected end of common control loop in %s", file)
		}
		loop := content[start : start+end]
		for _, forbidden := range []string{
			"confluenceApproveFullSnapshot",
			"confluenceApproveRangeSnapshot",
			"confluenceUpdatePreviewButton",
			"confluenceApproveUpdate",
		} {
			if strings.Contains(loop, forbidden) {
				t.Fatalf("common Confluence renderer in %s must not own %s disabled state", file, forbidden)
			}
		}
	}
}

func TestConfluenceBusyStateProtectsApprovalActions(t *testing.T) {
	common := []byte(mustReadPlasmaSourceScripts(t))
	review, err := os.ReadFile("static/plasma/sources_confluence_review.js")
	if err != nil {
		t.Fatal(err)
	}
	update, err := os.ReadFile("static/plasma/sources_confluence_update.js")
	if err != nil {
		t.Fatal(err)
	}
	commonContent := string(common)
	for _, expected := range []string{
		"renderConfluencePreview(state.confluencePreview)",
		"renderConfluenceUpdatePanel(state.confluenceUpdatePreview)",
	} {
		if !strings.Contains(commonContent, expected) {
			t.Fatalf("expected Confluence busy setter to refresh approval panels with %q", expected)
		}
	}
	reviewContent := string(review)
	for _, expected := range []string{
		"if (!requireMission() || state.confluenceBusy) return;",
		"if (!requireMission() || !page || state.confluenceBusy) return;",
		"if (state.confluenceBusy) return;\n    const preview = state.confluencePreview;",
	} {
		if !strings.Contains(reviewContent, expected) {
			t.Fatalf("expected Confluence review action guard %q", expected)
		}
	}
	updateContent := string(update)
	for _, expected := range []string{
		"if (!requireMission() || state.confluenceBusy) return;",
		"state.confluenceBusy || (!preview.new_page && !preview.NewPage)",
		"async function previewConfluenceUpdate() {\n    if (state.confluenceBusy) return;",
		"async function approveConfluenceUpdate() {\n    if (state.confluenceBusy) return;",
	} {
		if !strings.Contains(updateContent, expected) {
			t.Fatalf("expected Confluence update busy guard %q", expected)
		}
	}
}

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
	files := []string{"static/index.html", "static/app.js", "static/mission_metadata.js", "static/report_direction.js", "static/app.css"}
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
	for _, expected := range []string{"missionMetadataForm", "missionMetadataIncluded", "missionMetadataExcluded", "missionMetadataLines", ".filter(Boolean)", "method: 'PATCH'", "reportDirectionHint", "direction_hint", "clearAcceptedReportDirectionHint", "catch (err)"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing static contract %q", expected)
		}
	}
	clearIndex := strings.Index(string(mustReadStatic(t, "static/app.js")), "clearAcceptedReportDirectionHint")
	catchIndex := strings.Index(string(mustReadStatic(t, "static/app.js")), "} catch (err) {")
	if clearIndex < 0 || catchIndex < 0 {
		t.Fatal("missing success-clear or failure branch")
	}
}

func TestReportPipelineStaticGraphAndRetryContracts(t *testing.T) {
	script := string(mustReadStatic(t, "static/report_pipeline.js"))
	styles := string(mustReadStatic(t, "static/report_pipeline.css"))
	for _, expected := range []string{"<svg class=\"pipeline-graph", "--pipeline-width:", "--pipeline-height:", "pipeline-graph-fanout", "pipeline-visual-phase-fanout", "pathConnector", "pipelineLiveTimingTimer", "syncLiveTiming", "data-pipeline-live-timing", "data-pipeline-started-at", "data-pipeline-title-prefix", "최신 리포트 생성 파이프라인", "currentReportAttemptEvent(progress.attempt_id)", "<details class=\"pipeline-details\"", "role=\"img\"", "<ol class=\"pipeline-flow sr-only\"", "<li class=\"pipeline-node", "pipeline-phase", "섹션 작성", "파트 조립", "섹션 ${runningSections.length}개 병렬 작성", "phaseSummary(nodes)", "aria-current=\\\"step\\\"", "currentStage(graphNodes)", "hasPlannedContent(nodes)", "captureMissionSelection()", "isStaleMissionOperation(error)", "resume_failed", "restart", "started_at", "duration_ms", "visualNodeWidth(node)", "data-pipeline-node-width", "visualScrollLeft", "renderedVisual.scrollLeft"} {
		if !strings.Contains(script, expected) {
			t.Fatalf("missing report pipeline contract %q", expected)
		}
	}
	for _, expected := range []string{".pipeline-details", ".pipeline-attempt-meta", ".pipeline-visual", "max-width: 100%", "overflow-x: auto", "min-width: 0", "width: max(100%, var(--pipeline-width))", "height: var(--pipeline-height, 136px)", ".pipeline-visual-dot", ".pipeline-visual-time", "font-variant-numeric: tabular-nums", "pipeline-node-pulse", "prefers-reduced-motion: reduce", "pipeline-graph-revealing"} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing report pipeline style contract %q", expected)
		}
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
  let pipelineVisual={scrollLeft:73};
  const host={_innerHTML:"",get innerHTML(){return this._innerHTML;},set innerHTML(value){this._innerHTML=value;pipelineVisual={scrollLeft:0};},querySelector(selector){return selector===".pipeline-visual"?pipelineVisual:null;},querySelectorAll(selector){
    if(selector==="[data-report-retry]") return [resume,restart];
    return [];
  }};
  const context={window:{},state:{detail:{events:[{EventID:"evt_failed",Payload:{title:"<안전한 제목>",started_at:"2026-07-13T01:02:03Z"}}]}},document:{getElementById(id){return id==="reportPipeline"?host:null}},crypto:{randomUUID(){return "retry"}},setInterval(){return 99;},clearInterval(){},
    captureMissionSelection(){return {missionId:"mis_a"}},
    missionFetch(_owner,_path,options){requests.push(JSON.parse(options.body));return Promise.resolve({ok:true});},
    ownsMissionSelection(){return current;},reloadMission(){reloads++;},isStaleMissionOperation(){return false}};
  vm.createContext(context);vm.runInContext(fs.readFileSync("static/report_pipeline.js","utf8"),context);
  context.window.renderReportPipeline({attempt_id:"evt_failed",attempt_number:1,state:"failed",nodes:[{id:"plan",kind:"plan",state:"completed",started_at:"2026-07-13T01:02:03Z",duration_ms:12000},{id:"section-1-1",kind:"section",part_index:1,section_index:1,state:"completed",started_at:"2026-07-13T01:02:03Z",duration_ms:12000},{id:"section-2-1",kind:"section",part_index:2,section_index:1,state:"running",started_at:"2026-07-13T01:02:15Z"},{id:"part-1",kind:"part",part_index:1,state:"pending"},{id:"part-2",kind:"part",part_index:2,state:"failed",error:"safe",started_at:"2026-07-13T01:02:03Z",duration_ms:360000000},{id:"final",kind:"final",state:"pending"},{id:"artifact",kind:"artifact",state:"pending"}],retry:{resume_failed:true,restart:true}});
  const html=host.innerHTML;
  if(!html.includes("<h3 id=\"reportPipelineTitle\">최신 리포트 생성 파이프라인</h3>")||!html.includes("&lt;안전한 제목&gt;")||!html.includes("전체 생성 시작")||!html.includes("<time datetime=\"2026-07-13T01:02:03Z\">")||!html.includes("시도 1")||!html.includes("시작")||!html.includes("소요 12초")||!html.includes("경과")||!html.includes("<details class=\"pipeline-details\">")||html.includes("<details class=\"pipeline-details\" open")||!html.includes("<svg class=\"pipeline-graph\"")||!html.includes("--pipeline-width:")||!html.includes("<ol class=\"pipeline-flow sr-only\"")||!html.includes("<li class=\"pipeline-phase\"")||!html.includes("섹션 작성")||!html.includes("파트 조립"))process.exit(1);
  if(pipelineVisual.scrollLeft!==73)process.exit(10);
  const graphNodes=[...html.matchAll(/data-pipeline-node-width="(\d+)" transform="translate\((\d+) 62\)"/g)].map(([,width,x])=>({width:Number(width),x:Number(x)}));
  if(graphNodes.length!==7||graphNodes.some((node,index)=>index>0&&node.x-graphNodes[index-1].x<(node.width+graphNodes[index-1].width)/2+32))process.exit(11);
  if(!html.includes("role=\"img\"")||!html.includes("aria-current=\"step\"")||!html.includes("data-pipeline-live-timing=\"1\"")||!html.includes("data-pipeline-started-at=\"2026-07-13T01:02:15Z\"")||!html.includes("data-pipeline-title-prefix=\"section 2.1\"")||!html.includes("aria-label=\"part 2 실패, 시작")||!html.includes("safe\"")||!html.includes("tabindex=\"0\""))process.exit(2);
  if(!(html.indexOf("pipeline-plan") < html.indexOf("pipeline-section-1-1") && html.indexOf("pipeline-section-1-1") < html.indexOf("pipeline-section-2-1") && html.indexOf("pipeline-section-2-1") < html.indexOf("pipeline-part-1") && html.indexOf("pipeline-part-1") < html.indexOf("pipeline-part-2") && html.indexOf("pipeline-part-2") < html.indexOf("pipeline-final") && html.indexOf("pipeline-final") < html.indexOf("pipeline-artifact")))process.exit(7);
  if(typeof resume.listener!=="function"||typeof restart.listener!=="function")process.exit(3);
  await resume.listener();
  current=false;
  await restart.listener();
  if(requests.length!==2||requests[0].strategy!=="resume_failed"||requests[1].strategy!=="restart")process.exit(4);
  if(reloads!==1)process.exit(5);
  context.state.detail.events=[{EventID:"evt_missing",Payload:{}}];
  context.window.renderReportPipeline({attempt_id:"evt_missing",state:"running",nodes:[]});
  if(!host.innerHTML.includes("제목 없는 리포트")||!host.innerHTML.includes("생성 시작 시각 알 수 없음")||!host.innerHTML.includes("시도 번호 알 수 없음")||!host.innerHTML.includes("계획 수립")||!host.innerHTML.includes("진행 중")||!host.innerHTML.includes("pipeline-plan")||host.innerHTML.includes("pipeline-final")||host.innerHTML.includes("pipeline-artifact")||host.innerHTML.includes("pipeline-phase"))process.exit(8);
  host.dataset={};
  context.window.renderReportPipeline({attempt_id:"evt_missing",state:"running",nodes:[]});
  context.window.renderReportPipeline({attempt_id:"evt_missing",state:"running",nodes:[{id:"plan",kind:"plan",state:"completed"},{id:"part-1",kind:"part",part_index:1,state:"running"}]});
  if(!host.innerHTML.includes("pipeline-graph-revealing")||!host.innerHTML.includes("파트 1 작성")||!host.innerHTML.includes("진행 중")||pipelineVisual.scrollLeft!==73)process.exit(9);
  context.state.detail.events=[{EventID:"evt_fanout",Payload:{title:"병렬 보고서",started_at:"2026-07-13T01:02:03Z",report_mode:"long_form",execution_strategy:"section_fanout"}}];
  context.window.renderReportPipeline({attempt_id:"evt_fanout",attempt_number:1,state:"running",nodes:[{id:"plan",kind:"plan",state:"completed"},{id:"section-1-1",kind:"section",part_index:1,section_index:1,state:"running"},{id:"section-2-1",kind:"section",part_index:2,section_index:1,state:"running"},{id:"part-1",kind:"part",part_index:1,state:"pending"},{id:"part-2",kind:"part",part_index:2,state:"pending"},{id:"final",kind:"final",state:"pending"},{id:"artifact",kind:"artifact",state:"pending"}]});
  const fanoutHtml=host.innerHTML;
  if(!fanoutHtml.includes("장문 · 빠른 병렬")||!fanoutHtml.includes("섹션 2개 병렬 작성")||!fanoutHtml.includes("진행 2")||!fanoutHtml.includes("pipeline-graph pipeline-graph-fanout")||!fanoutHtml.includes("계획에서 여러 섹션 작성으로 갈라지고")||!fanoutHtml.includes("pipeline-visual-phase-fanout"))process.exit(12);
  const fanoutRows=new Set([...fanoutHtml.matchAll(/transform="translate\((?:[\d.]+) ([\d.]+)\)"/g)].map(([,y])=>Number(y)));
  if(!fanoutRows.has(62)||!fanoutRows.has(146)||fanoutRows.size<2)process.exit(13);
})().catch((error)=>{console.error(error);process.exit(6);});`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("pipeline DOM fixture: %v: %s", err, out)
	}
}

func TestStaticMissionScopedActiveWorkContracts(t *testing.T) {
	combined := string(mustReadStatic(t, "static/index.html")) + string(mustReadStatic(t, "static/app.js")) + string(mustReadStatic(t, "static/app.css"))
	for _, expected := range []string{
		"active_work", "resetMissionTransientState", "ownsMissionSelection",
		"conversationActiveWork", "reportActiveWork", "report_generation_running",
		"workflow_running", "agent_turn_running", "data-active-work-action",
		".active-work-notice", "flex-wrap: wrap",
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("missing mission-scoped active-work contract %q", expected)
		}
	}
}

func TestConversationExportStaticContracts(t *testing.T) {
	script := string(mustReadStatic(t, "static/app.js"))
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
	script := string(mustReadStatic(t, "static/app.js"))
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
	script := string(mustReadStatic(t, "static/app.js"))
	remember := jsFunctionSource(t, script, "rememberMissionID")
	cursor := jsFunctionSource(t, script, "missionActivityCursor")
	detailCursor := jsFunctionSource(t, script, "detailMissionActivityCursor")
	apply := jsFunctionSource(t, script, "applyMissionDetail")
	source := strings.Replace(jsFunctionSource(t, script, "selectMission"), "function selectMission", "async function selectMission", 1)
	fixture := `
const MISSION_STORAGE_KEY = "plasma.activeMissionId";
const state = { detail: null, missionActivityCursors: {} };
let shouldFail = false; let marked = []; let renderedFailure = 0;
const beginMissionSelection = (missionId) => ({ missionId });
const ownsDetailRequest = () => true;
const api = async () => { if (shouldFail) throw new Error("failed"); return {projection:{last_sequence:9},activity_cursor:{schema:"mission-activity/v1",sequence:9,server_id:"server-a"}}; };
const localStorage = { setItem() { throw new Error("quota"); } };
const markMissionActivitySeen = (...args) => marked.push(args);
const renderDetail = () => {}; const renderMissions = () => {};
const refreshMissionList = async () => {}; const loadConfluenceConnections = async () => {};
const loadConfluenceAccess = async () => {}; const renderMissionLoadFailed = () => { renderedFailure++; };
` + remember + `
` + cursor + `
` + detailCursor + `
` + apply + `
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
	script := string(mustReadStatic(t, "static/app.js"))
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
` + jsFunctionSource(t, script, "missionActivityCursor") + `
` + jsFunctionSource(t, script, "detailMissionActivityCursor") + `
` + jsFunctionSource(t, script, "applyMissionDetail") + `
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
	script := string(mustReadStatic(t, "static/app.js"))
	source := jsFunctionSource(t, script, "scheduleMissionActivityPoll")
	refresh := strings.Replace(jsFunctionSource(t, script, "refreshObservedMissionActivity"), "function refreshObservedMissionActivity", "async function refreshObservedMissionActivity", 1)
	fixture := `
let scheduled = 0; const callbacks = [];
const state = {missions:[],missionActivityPollTimer:0,missionActivityPollInFlight:false};
const window = {clearTimeout(){},setTimeout(callback){ scheduled++; callbacks.push(callback); return scheduled; }};
const document = {hidden:false};
const captureMissionSelection = () => ({missionId:""});
` + source + `
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
` + refresh + `
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
	script := string(mustReadStatic(t, "static/app.js"))
	asyncSource := func(name string) string {
		return strings.Replace(jsFunctionSource(t, script, name), "function "+name, "async function "+name, 1)
	}
	fixture := `
const WORKFLOW_DEFAULT_MAX_STEPS = 20;
const WORKFLOW_DEFAULT_MAX_DURATION_MS = 0;
const MISSION_STORAGE_KEY = "plasma.activeMissionId";
const state = {missionId:"mis_1",selectionGeneration:1,detailGeneration:0,detail:{projection:{title:"Mission"}},missions:[],missionActivityCursors:{},turnPending:false,workflowPending:false,workflowGoalDraftPending:false,workflowGoalDraftRaw:"",reportPending:false,pendingTurn:null,missionActivityPollTimer:0,missionActivityPollInFlight:false};
const nodes = {
  turnText:{value:"Question"}, agentExecutor:{value:"codex"}, mcpMode:{value:"auto"}, controllerStrategy:{value:"auto"},
  workflowInstruction:{value:"Research"}, workflowRunGoal:{value:""}, workflowStepInstruction:{value:""},
  reportAgentModel:{value:""}, reportAgentReasoningEffort:{value:""}, reportRigor:{value:"balanced"}
};
const $ = (id) => nodes[id] || {value:""};
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
const ReportModelSelection = {payload:() => ({agent_model:"",agent_reasoning_effort:""})};
const currentReportDirectionHint = () => ""; const clearAcceptedReportDirectionHint = () => {};
const document = {hidden:false}; const window = {clearTimeout(){},setTimeout(){ scheduled++; return scheduled; }};
` + asyncSource("refreshMissionList") + `
` + jsFunctionSource(t, script, "missionActivityCursor") + `
` + jsFunctionSource(t, script, "detailMissionActivityCursor") + `
` + jsFunctionSource(t, script, "applyMissionDetail") + `
` + asyncSource("refreshSelectedMissionDetail") + `
` + asyncSource("selectMission") + `
` + asyncSource("reloadMission") + `
` + jsFunctionSource(t, script, "scheduleMissionActivityPoll") + `
` + asyncSource("sendTurn") + `
` + asyncSource("startWorkflow") + `
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
	script := string(mustReadStatic(t, "static/app.js"))
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
const renderMissionLoading = () => {};
const resetConfluenceMissionUI = () => {};
const hideDetail = () => {};
const empty = () => "";
const $ = () => null;
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
	source := jsFunctionSource(t, string(mustReadStatic(t, "static/app.js")), "resetMissionTransientState")
	fixture := `
const state={detail:{},turnPending:true,reportPending:true,workflowPending:true,workflowGoalDraftPending:false,workflowGoalDraftRaw:"",pendingTurn:null,sourceCandidateBusy:new Set(),selectedSourceCandidates:new Set(["a"]),selectedProposals:new Set(["p"]),selectedReportKey:"",reportPreview:null,confluenceSearchResults:[],confluenceSearchContext:null,confluenceSpaces:[],confluencePages:[],confluenceBrowseContext:null,confluencePreview:null,confluenceUpdatePreview:null,confluenceAccess:null,confluenceBusy:true,confluenceOAuthURL:""};
const nodes={sourceCandidateBulk:{classList:{hidden:false,add(v){this.hidden=v}}},proposalBulk:{classList:{hidden:false,add(v){this.hidden=v}}},sourceCandidateBulkCount:{textContent:"7"},proposalBulkCount:{textContent:"8"}};
const $=(id)=>nodes[id]||null; const clearPendingPoll=()=>{}; const setReportNotice=()=>{}; const renderActiveWork=()=>{}; const setFormsEnabled=()=>{}; const hideDetail=()=>{}; const empty=()=>""; const renderMissionLoading=()=>{}; const resetConfluenceMissionUI=()=>{};
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
	script := string(mustReadStatic(t, "static/app.js"))
	create := strings.Replace(jsFunctionSource(t, script, "createMission"), "function createMission", "async function createMission", 1)
	refresh := strings.Replace(jsFunctionSource(t, script, "refreshMissionList"), "function refreshMissionList", "async function refreshMissionList", 1)
	fixture := `
const state={missionId:"mis_a",selectionGeneration:1,missions:[]};
const nodes={missionTitle:{value:"New"},missionObjective:{value:"Goal"}}; const $=(id)=>nodes[id];
let createResolve,listResolve; let selected=[]; let renders=0; let errors=0;
const api=(path)=>path==="/api/missions"?(createResolve?new Promise(r=>{listResolve=r}):new Promise(r=>{createResolve=r})):Promise.reject(new Error("unexpected"));
const captureMissionSelection=()=>({missionId:state.missionId,selectionGeneration:state.selectionGeneration});
const ownsMissionSelection=(o)=>o.missionId===state.missionId&&o.selectionGeneration===state.selectionGeneration;
class StaleMissionOperationError extends Error{}; const renderMissions=()=>{renders++}; const pruneMissionActivitySeenWatermarks=()=>{}; const scheduleMissionActivityPoll=()=>{}; const selectMission=async(id)=>{selected.push(id)}; const showError=()=>{errors++};
` + create + `
` + refresh + `
(async()=>{
  const run=createMission({preventDefault(){}}); createResolve({projection:{mission_id:"mis_new"}}); await Promise.resolve(); listResolve({missions:[{MissionID:"mis_new"}]}); await run;
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
	script := string(mustReadStatic(t, "static/app.js"))
	exportFn := jsSourceRange(t, script, "async function exportReport(", "\nasync function viewReportArtifact(")
	astFn := jsSourceRange(t, script, "async function viewReportAST(", "\nfunction setSectionEmpty(")
	fixture := `
const state={missionId:"mis_a",selectionGeneration:1,reportPreview:null};
const captureMissionSelection=()=>({missionId:state.missionId,selectionGeneration:state.selectionGeneration}); const ownsMissionSelection=(o)=>o.missionId===state.missionId&&o.selectionGeneration===state.selectionGeneration;
let pending=[]; const api=()=>new Promise((resolve,reject)=>pending.push({resolve,reject})); let mutations=[];
const setReportPreviewLoading=()=>{}; const assertReportExportMatches=()=>{}; const downloadText=()=>mutations.push("download"); const applyReportPreview=()=>mutations.push("preview"); const reloadMission=()=>mutations.push("reload"); const clearReportPreview=()=>mutations.push("clear"); const showError=()=>mutations.push("error");
const reportExportPreviewHeader=()=>"";
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
	script := string(mustReadStatic(t, "static/app.js"))
	schedule := jsFunctionSource(t, script, "schedulePendingPoll")
	fixture := `
const state={missionId:"mis_a",selectionGeneration:1,detailGeneration:1,turnPending:true,reportPending:false,workflowPending:false,pollTimer:0,pollInFlight:false,pollOwner:null};
const captureMissionSelection=()=>({missionId:state.missionId,selectionGeneration:state.selectionGeneration}); const ownsMissionSelection=(o)=>o.missionId===state.missionId&&o.selectionGeneration===state.selectionGeneration;
const ownsDetailRequest=(o)=>ownsMissionSelection(o)&&o.detailGeneration===state.detailGeneration;
let callbacks=[]; const window={setTimeout:(fn)=>{callbacks.push(fn);return callbacks.length}}; const clearPendingPoll=()=>{state.pollTimer=0}; const nodes={healthBadge:{textContent:""}}; const $=(id)=>nodes[id];
let waits={}; const refreshSelectedMissionActivity=(owner)=>new Promise(r=>{waits[owner.missionId]=r}); const console={warn(){},error(){}};
` + schedule + `
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
	script := string(mustReadStatic(t, "static/app.js"))
	schedule := jsFunctionSource(t, script, "schedulePendingPoll")
	fixture := `
const state={missionId:"mis_a",selectionGeneration:1,detailGeneration:1,turnPending:true,reportPending:false,workflowPending:false,pollTimer:0,pollInFlight:false,pollOwner:null};
const captureMissionSelection=()=>({missionId:state.missionId,selectionGeneration:state.selectionGeneration});
const ownsMissionSelection=(owner)=>owner.missionId===state.missionId&&owner.selectionGeneration===state.selectionGeneration;
const ownsDetailRequest=(owner)=>ownsMissionSelection(owner)&&owner.detailGeneration===state.detailGeneration;
const callbacks=[]; const window={setTimeout:(fn)=>{callbacks.push(fn);return callbacks.length},clearTimeout(){}};
const clearPendingPoll=()=>{state.pollTimer=0}; const nodes={healthBadge:{textContent:""}}; const $=(id)=>nodes[id]; const console={warn(){},error(){}};
const refreshSelectedMissionActivity=async()=>{state.detailGeneration++; throw new Error("fallback detail failed");};
` + schedule + `
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
	script := string(mustReadStatic(t, "static/app.js"))
	schedule := jsFunctionSource(t, script, "schedulePendingPoll")
	fixture := `
const state={missionId:"mis_a",selectionGeneration:1,detailGeneration:1,turnPending:true,reportPending:false,workflowPending:false,pollTimer:0,pollInFlight:false,pollOwner:null};
const captureMissionSelection=()=>({missionId:state.missionId,selectionGeneration:state.selectionGeneration});
const ownsMissionSelection=(owner)=>owner.missionId===state.missionId&&owner.selectionGeneration===state.selectionGeneration;
const ownsDetailRequest=(owner)=>ownsMissionSelection(owner)&&owner.detailGeneration===state.detailGeneration;
const callbacks=[]; const window={setTimeout:(fn)=>{callbacks.push(fn);return callbacks.length},clearTimeout(){}};
const clearPendingPoll=()=>{state.pollTimer=0}; const nodes={healthBadge:{textContent:""}}; const $=(id)=>nodes[id]; const console={warn(){},error(){}};
let resolveA; const refreshSelectedMissionActivity=(owner)=>owner.detailGeneration===1 ? new Promise(resolve=>{resolveA=resolve}) : Promise.reject(new Error("B failed"));
` + schedule + `
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
	script := string(mustReadStatic(t, "static/app.js"))
	source := jsFunctionSource(t, script, "beginMissionSelection") + "\n" + jsFunctionSource(t, script, "captureMissionSelection") + "\n" + jsFunctionSource(t, script, "ownsMissionSelection")
	fixture := `
const state = {missionId:"", selectionGeneration:0, detailGeneration:0};
let resets = 0;
let loadingRenders = 0;
const resetMissionTransientState = () => { resets += 1; loadingRenders += 1; };
` + source + `
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
	for _, name := range []string{"static/confluence.js", "static/confluence_access.js", "static/confluence_browse.js"} {
		content := string(mustReadStatic(t, name))
		if !strings.Contains(content, "ownsMissionSelection(owner)") || !strings.Contains(content, "captureMissionSelection()") {
			t.Fatalf("%s must guard stale mission responses", name)
		}
	}
}

func TestBatchAMissionRoutesUseCapturedTransport(t *testing.T) {
	script := string(mustReadStatic(t, "static/app.js"))
	for _, name := range []string{"resetAgentSession", "addTextSource", "addUploadSource", "addMediaURLSource", "addPDFURLSource", "browseLocalPathTree", "attachLocalPathSource", "refreshSourcesOnly", "removeSource", "restoreSource", "readSource", "addURLSource", "viewReportArtifact", "downloadReportArtifact"} {
		body := jsFunctionBody(t, script, name)
		if strings.Contains(body, "/api/missions/${state.missionId}") || !strings.Contains(body, "missionApi") && !strings.Contains(body, "missionFetch") {
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
	script := string(mustReadStatic(t, "static/app.js"))
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
const reloadMission=async()=>{}; const showError=()=>{ throw new Error("stale error shown") }; const window={prompt:()=>""};
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

func TestStaticReportModelSelectionContract(t *testing.T) {
	combined := string(mustReadStatic(t, "static/index.html")) + string(mustReadStatic(t, "static/app.js")) + string(mustReadStatic(t, "static/report_model_selection.js"))
	for _, expected := range []string{`id="reportAgentModel"`, `id="reportAgentReasoningEffort"`, `/static/report_model_selection.js`, "agent_model", "agent_reasoning_effort", "미션 설정 상속", "refreshEfforts", ".disabled = busy", "segmented-select-label"} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("missing report model selection contract %q", expected)
		}
	}
	if _, err := exec.LookPath("node"); err == nil {
		command := exec.Command("node", "-e", `require('./static/report_model_selection.js'); const p=globalThis.ReportModelSelection.payload; if(JSON.stringify(p('',''))!==JSON.stringify({agent_model:'',agent_reasoning_effort:''})||p('gpt-5.5','').agent_model!=='gpt-5.5'||p('gpt-5.5','high').agent_reasoning_effort!=='high') process.exit(1)`)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("node payload fixture failed: %v: %s", err, output)
		}
	}
}

func TestStaticReportControlsIntegrateLabelsInsideSelects(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	styles := string(mustReadStatic(t, "static/app.css"))
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
		"reportGenerationGuidance",
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
	styles := string(mustReadStatic(t, "static/app.css"))
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
		`id="focusToggle" class="quiet"`,
		`id="themeToggle" class="icon-button quiet"`,
		`id="refreshMissions" class="icon-button quiet"`,
		`id="missionMetadataEdit" class="quiet mission-recall-button"`,
		`id="closeDetail" class="icon-button quiet"`,
	} {
		if !strings.Contains(index, expected) {
			t.Fatalf("utility button is not assigned to quiet role: %q", expected)
		}
	}
}

func TestStaticTabControlsKeepTheirFlatOriginalTreatment(t *testing.T) {
	styles := string(mustReadStatic(t, "static/app.css"))
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
	script := string(mustReadStatic(t, "static/app.js"))
	for _, expected := range []string{
		"reportGenerationContext",
		"reportGenerationSummaryHTML",
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

func TestStaticReportModelSelectionFollowsExecutorAndActiveGuards(t *testing.T) {
	script := string(mustReadStatic(t, "static/app.js"))
	executorBody := jsFunctionBody(t, script, "onAgentExecutorChange")
	for _, expected := range []string{"ReportModelSelection.render", `$("agentExecutor").value`} {
		if !strings.Contains(executorBody, expected) {
			t.Fatalf("executor switch missing %q: %s", expected, executorBody)
		}
	}
	formsBody := jsFunctionBody(t, script, "setFormsEnabled")
	for _, expected := range []string{"reportAgentModel", "reportAgentReasoningEffort", "state.turnPending", "state.workflowPending", "state.reportPending", "draftQuickReport", "draftLongReport"} {
		if !strings.Contains(formsBody, expected) {
			t.Fatalf("report guard missing %q", expected)
		}
	}
	module := string(mustReadStatic(t, "static/report_model_selection.js"))
	for _, expected := range []string{"status?.models", "model?.reasoning_efforts", `effortSelect.innerHTML`, `value=""`} {
		if !strings.Contains(module, expected) {
			t.Fatalf("selection semantics missing %q", expected)
		}
	}
}

func TestStaticSettingsExposeModelDefaultsCard(t *testing.T) {
	html := string(mustReadStatic(t, "static/index.html"))
	appScript := string(mustReadStatic(t, "static/app.js"))
	modelSettingsScript := string(mustReadStatic(t, "static/model_settings.js"))
	combined := html + "\n" + appScript + "\n" + modelSettingsScript
	for _, expected := range []string{
		`id="modelDefaultsDetails"`,
		`id="modelDefaultsForm"`,
		`id="workflowGoalDefaultModel"`,
		`id="workflowGoalDefaultReasoningEffort"`,
		`/static/model_settings.js`,
		`/api/settings/model-defaults`,
		`saveModelDefaults`,
		`loadModelDefaults`,
		`renderModelDefaultEfforts`,
		`자율진행 조향 모델`,
		`workflow directing model`,
		`현재는 시작 시점의 3층 지시 초안 생성에만 사용`,
		`새 에이전트 세션`,
		`보고서 생성`,
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("expected model defaults settings surface %q", expected)
		}
	}
	settingsPanel := htmlSection(t, html, `data-tab-panel="settings"`, `id="errorToast"`)
	if strings.Index(settingsPanel, `id="modelDefaultsDetails"`) < 0 || strings.Index(settingsPanel, `id="confluenceSettingsDetails"`) < 0 ||
		strings.Index(settingsPanel, `id="modelDefaultsDetails"`) > strings.Index(settingsPanel, `id="confluenceSettingsDetails"`) {
		t.Fatalf("model defaults card must be the first Settings fold card")
	}
	setFormsBody := jsFunctionBody(t, appScript, "setFormsEnabled")
	for _, forbidden := range []string{"modelDefaultsForm", "workflowGoalDefaultModel", "workflowGoalDefaultReasoningEffort"} {
		if strings.Contains(setFormsBody, forbidden) {
			t.Fatalf("global model default setting %q must not be disabled by mission-bound form state", forbidden)
		}
	}
}

func TestModelSettingsScriptUsesCodexCatalogAndReasoningEfforts(t *testing.T) {
	script := string(mustReadStatic(t, "static/model_settings.js"))
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
	script := string(mustReadStatic(t, "static/app.js"))
	source := jsFunctionSource(t, script, "activeWorkBlocksControl") + "\n" + jsFunctionSource(t, script, "syncReportControls") + "\n" + jsFunctionSource(t, script, "setReportBusy")
	fixture := `
const elements = {};
for (const id of ["reportStatus","reportRigor","reportAgentModel","reportAgentReasoningEffort","reportLongFormExecutionStrategy","reportGenerationGuidance","draftQuickReport","draftLongReport","cancelReportButton"]) {
  elements[id] = {disabled:false,textContent:"",classList:{toggle(){}}};
}
const $ = (id) => elements[id];
const state = {detail:{active_work:{blocked_controls:[]}},turnPending:false,workflowPending:false,workflowGoalDraftPending:false,reportPending:false};
` + source + `
const controls = ["reportRigor","reportAgentModel","reportAgentReasoningEffort","reportLongFormExecutionStrategy","reportGenerationGuidance","draftQuickReport","draftLongReport"];
function assertDisabled(label) {
  if (!controls.every((id) => elements[id].disabled)) throw new Error(label + " re-enabled a report control");
}
for (const guard of ["agent_turn_running","workflow_running","report_generation_running"]) {
  state.detail.active_work.blocked_controls = [{control:"report_start",reason_codes:[guard]}];
  state.workflowGoalDraftPending = false;
  setReportBusy(false);
  assertDisabled(guard);
}
state.detail.active_work.blocked_controls = [];
state.workflowGoalDraftPending = false;
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
	source := jsFunctionSource(t, string(mustReadStatic(t, "static/app.js")), "draftReport")
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

func TestStaticAppLabelsPendingEvidenceSignalType(t *testing.T) {
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	content := string(script)
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
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
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
	if !strings.Contains(string(script), "controller_strategy") ||
		!strings.Contains(string(script), "controllerStrategy") {
		t.Fatalf("expected static app script to submit controller strategy")
	}
}

func TestStaticAppExposesEnvironmentBadge(t *testing.T) {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	style, err := os.ReadFile("static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(html) + "\n" + string(script) + "\n" + string(style)
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
	style, err := os.ReadFile("static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	content := string(style)
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
	style, err := os.ReadFile("static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	content := string(style)
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
	style, err := os.ReadFile("static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	content := string(style)
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
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	style, err := os.ReadFile("static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(html) + "\n" + string(script) + "\n" + string(style)
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

func TestStaticAppExposesWorkflowControlsWithoutTerminalUI(t *testing.T) {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(html) + "\n" + string(script)
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
		`$("turnText").addEventListener("input", onWorkflowRawInput)`,
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
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(html) + "\n" + string(script)
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
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	content := string(script)
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
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	content := string(script)
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
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	content := string(script)
	if !strings.Contains(content, "function renderDetail()") {
		t.Fatalf("expected static app to define renderDetail")
	}
	if strings.Contains(content, "renderMissionDetail(") {
		t.Fatalf("static app should not call missing renderMissionDetail")
	}
}

func TestStaticAppExposesReportHumanizeRetry(t *testing.T) {
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	content := string(script)
	for _, expected := range []string{
		"H5 말투 보정 다시 생성",
		"start-humanized-markdown-artifact",
		"exportReportArtifactHumanizedMarkdown",
		"/humanized_markdown_export",
		"H5 말투 보정 시작 실패",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected static app to expose report humanize retry %q", expected)
		}
	}
}

func TestStaticAppTreatsHumanizeSkippedAsTerminalState(t *testing.T) {
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
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
	appScript, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceErrorsScript, err := os.ReadFile("static/confluence_errors.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceScript, err := os.ReadFile("static/confluence.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceWorkflowScript, err := os.ReadFile("static/confluence_workflow.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceSettingsScript, err := os.ReadFile("static/confluence_settings.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceAccessScript, err := os.ReadFile("static/confluence_access.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceBrowseScript, err := os.ReadFile("static/confluence_browse.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceReviewScript, err := os.ReadFile("static/confluence_review.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceUpdateScript, err := os.ReadFile("static/confluence_update.js")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(html) + "\n" + string(appScript) + "\n" + string(confluenceErrorsScript) + "\n" + string(confluenceScript) + "\n" + string(confluenceSettingsScript) + "\n" + string(confluenceAccessScript) + "\n" + string(confluenceWorkflowScript) + "\n" + string(confluenceBrowseScript) + "\n" + string(confluenceReviewScript) + "\n" + string(confluenceUpdateScript)
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
		`/static/confluence.js`,
		`/static/confluence_errors.js`,
		`/static/confluence_settings.js`,
		`/static/confluence_access.js`,
		`/static/confluence_workflow.js`,
		`/static/confluence_browse.js`,
		`/static/confluence_review.js`,
		`/static/confluence_update.js`,
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
		"renderConfluenceSpaces(state.confluenceSpaces);\n    renderConfluencePages([]);",
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
	if strings.Contains(string(confluenceScript), `data-detail-title="Confluence 검색 결과" data-detail-json="${escapeAttr(JSON.stringify(candidate))}"`) ||
		strings.Contains(string(confluenceScript), "connector.ExternalURI") ||
		strings.Contains(string(confluenceScript), "connector.external_uri") {
		t.Fatalf("Confluence search candidate detail must not expose the raw connector payload")
	}
}

func TestConfluenceErrorMessagesAreActionableAndSafe(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for Confluence error mapping fixture test")
	}
	script, err := os.ReadFile("static/confluence_errors.js")
	if err != nil {
		t.Fatal(err)
	}
	nodeScript := string(script) + `
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
		"static/confluence.js",
		"static/confluence_settings.js",
		"static/confluence_access.js",
		"static/confluence_browse.js",
		"static/confluence_review.js",
		"static/confluence_update.js",
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
	content := string(mustReadStatic(t, "static/confluence_settings.js"))
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
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	body := jsFunctionBody(t, string(script), "sourceDetailPayload")
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
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	nodeScript := jsFunctionSource(t, string(script), "timeShort") + "\n" +
		jsFunctionSource(t, string(script), "confluenceUpdateFailureText") + "\n" +
		jsFunctionSource(t, string(script), "confluenceUpdateText") + `
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
	script := string(mustReadStatic(t, "static/app.js"))
	nodeScript := jsFunctionSource(t, script, "escapeHTML") + "\n" +
		jsFunctionSource(t, script, "timeShort") + "\n" +
		jsFunctionSource(t, script, "confluenceUpdateFailureText") + "\n" +
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
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	nodeScript := jsFunctionSource(t, string(script), "confluenceDisplayableExternalURI") + "\n" +
		jsFunctionSource(t, string(script), "sourceDetailPayload") + `
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

func TestPDFLocatorRecognizesUploadedPDFSource(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for semantic static app JS fixture test")
	}
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	nodeScript := jsFunctionSource(t, string(script), "sourceLocatorType") + "\n" +
		jsFunctionSource(t, string(script), "pdfLocator") + `
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
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	nodeScript := jsFunctionSource(t, string(script), "sourceLocatorType") + "\n" +
		jsFunctionSource(t, string(script), "sourceConnectorType") + "\n" +
		jsFunctionSource(t, string(script), "uploadedFileContentKind") + "\n" +
		jsFunctionSource(t, string(script), "uploadedFileMediaType") + "\n" +
		jsFunctionSource(t, string(script), "uploadedFileFilename") + "\n" +
		jsFunctionSource(t, string(script), "mediaLocator") + "\n" +
		jsFunctionSource(t, string(script), "documentLocator") + `
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
	script, err := os.ReadFile("static/confluence.js")
	if err != nil {
		t.Fatal(err)
	}
	body := jsFunctionBody(t, string(script), "confluenceCandidateDetailPayload")
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
	appScript, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceScript, err := os.ReadFile("static/confluence.js")
	if err != nil {
		t.Fatal(err)
	}
	browseScript, err := os.ReadFile("static/confluence_browse.js")
	if err != nil {
		t.Fatal(err)
	}
	nodeScript := jsFunctionSource(t, string(appScript), "confluenceDisplayableExternalURI") + "\n" +
		jsFunctionSource(t, string(appScript), "confluenceExternalURIHost") + "\n" +
		jsFunctionSource(t, string(browseScript), "confluenceCandidatePageID") + "\n" +
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
	script, err := os.ReadFile("static/confluence_settings.js")
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
	script := string(mustReadStatic(t, "static/app.js"))
	fixture := `
const state={pendingTurn:null,turnPending:false,missionId:"mis_1",turnScrollMission:"mis_1"};
const log={scrollHeight:0,scrollTop:0,clientHeight:0,innerHTML:""};
const window={renderPlasmaMath(){}};
const $=()=>log;
const completedUserEventIDs=()=>new Set();
const escapeHTML=(value)=>String(value);
const escapeAttr=(value)=>String(value);
const timeShort=()=>"12:00";
const shortID=(value)=>String(value);
const renderMarkdown=(value)=>String(value);
const empty=(value)=>String(value);
const updateTurnNavVisibility=()=>{};
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

func TestStaticAppReportsPreservedMarkdownAfterPatchFailure(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := string(mustReadStatic(t, "static/app.js"))
	fixture := `
let notice={};
const setReportBusy=()=>{};
const setReportNotice=(text,kind)=>{notice={text,kind};};
const reportTimingDetails=()=>"";
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
		t.Fatalf("expected function %s in static app", name)
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
		t.Fatalf("expected function %s in static app", name)
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
		"static/confluence.js",
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
	common, err := os.ReadFile("static/confluence.js")
	if err != nil {
		t.Fatal(err)
	}
	review, err := os.ReadFile("static/confluence_review.js")
	if err != nil {
		t.Fatal(err)
	}
	update, err := os.ReadFile("static/confluence_update.js")
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
		"if (state.confluenceBusy) return;\n  const preview = state.confluencePreview;",
	} {
		if !strings.Contains(reviewContent, expected) {
			t.Fatalf("expected Confluence review action guard %q", expected)
		}
	}
	updateContent := string(update)
	for _, expected := range []string{
		"if (!requireMission() || state.confluenceBusy) return;",
		"state.confluenceBusy || (!preview.new_page && !preview.NewPage)",
		"async function previewConfluenceUpdate() {\n  if (state.confluenceBusy) return;",
		"async function approveConfluenceUpdate() {\n  if (state.confluenceBusy) return;",
	} {
		if !strings.Contains(updateContent, expected) {
			t.Fatalf("expected Confluence update busy guard %q", expected)
		}
	}
}

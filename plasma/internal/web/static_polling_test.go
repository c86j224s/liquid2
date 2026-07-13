package web

import (
	"os/exec"
	"strings"
	"testing"
)

func TestSelectedMissionActivityPollUsesCursorBeforeDetailFallback(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := string(mustReadStatic(t, "static/app.js"))
	functions := []string{
		jsFunctionSource(t, script, "missionActivityCursor"),
		jsFunctionSource(t, script, "detailMissionActivityCursor"),
		jsFunctionSource(t, script, "mergeMissionActivity"),
		jsFunctionSource(t, script, "applyMissionDetail"),
		strings.Replace(jsFunctionSource(t, script, "refreshSelectedMissionDetail"), "function refreshSelectedMissionDetail", "async function refreshSelectedMissionDetail", 1),
		strings.Replace(jsFunctionSource(t, script, "refreshSelectedMissionActivity"), "function refreshSelectedMissionActivity", "async function refreshSelectedMissionActivity", 1),
	}
	fixture := `
const state = {
  missionId:"mis_1", selectionGeneration:1, detailGeneration:1,
  detail:null, missions:[{MissionID:"mis_1",activity:{last_sequence:5,active_work:{items:[{}]}}}],
  missionActivityCursors:{}
};
const requests=[];
let nextCursor={schema:"mission-activity/v1",sequence:5,server_id:"server-a"};
const captureMissionSelection=()=>({missionId:"mis_1",selectionGeneration:1});
const ownsMissionSelection=(owner)=>owner.missionId===state.missionId && owner.selectionGeneration===state.selectionGeneration;
const ownsDetailRequest=(owner)=>ownsMissionSelection(owner) && owner.detailGeneration===state.detailGeneration;
const api=async(path)=>{requests.push(path); const sequence=nextCursor?.sequence ?? 0; if(path.endsWith("/activity")) return {activity:{last_sequence:sequence,active_work:{items:[]}},cursor:nextCursor}; return {projection:{last_sequence:sequence},activity_cursor:nextCursor};};
const rememberMissionID=()=>{}; const markMissionActivitySeen=()=>{}; const renderDetail=()=>{}; const renderMissions=()=>{};
` + strings.Join(functions, "\n") + `
(async()=>{
  applyMissionDetail({missionId:"mis_1",selectionGeneration:1,detailGeneration:1}, {projection:{last_sequence:1},activity_cursor:nextCursor});
  if(!state.missionActivityCursors.mis_1 || state.missionActivityCursors.mis_1.sequence!==5) throw new Error("initial detail did not seed activity cursor");
  requests.length=0;
  await refreshSelectedMissionActivity();
  if(requests.join()!=="/api/missions/mis_1/activity") throw new Error("unchanged cursor fetched detail");
  requests.length=0; nextCursor={schema:"mission-activity/v1",sequence:6,server_id:"server-a"};
  await refreshSelectedMissionActivity();
  if(requests.join()!=="/api/missions/mis_1/activity,/api/missions/mis_1") throw new Error("advanced cursor did not refresh only selected detail");
  requests.length=0; nextCursor={schema:"mission-activity/v1",sequence:8,server_id:"server-a"};
  await refreshSelectedMissionActivity();
  if(requests.join()!=="/api/missions/mis_1/activity,/api/missions/mis_1") throw new Error("cursor gap did not use bounded detail fallback");
  requests.length=0; nextCursor={schema:"mission-activity/v1",sequence:8,server_id:"server-b"};
  await refreshSelectedMissionActivity();
  if(requests.join()!=="/api/missions/mis_1/activity,/api/missions/mis_1") throw new Error("server restart did not use bounded detail fallback");
  for (const [label, cursor] of [
    ["regression", {schema:"mission-activity/v1",sequence:7,server_id:"server-b"}],
    ["missing", null],
    ["invalid", {schema:"mission-activity/v1",sequence:7}],
    ["schema", {schema:"mission-activity/v2",sequence:7,server_id:"server-b"}]
  ]) {
    requests.length=0; nextCursor=cursor;
    await refreshSelectedMissionActivity();
    if(requests.join()!=="/api/missions/mis_1/activity,/api/missions/mis_1") throw new Error(label + " cursor did not use bounded detail fallback");
  }
})().catch((err)=>{console.error(err);process.exit(1);});`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("selected mission activity poll fixture: %v: %s", err, out)
	}
}

func TestOlderActivityPollCannotReplaceNewerOrdinaryReload(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := string(mustReadStatic(t, "static/app.js"))
	asyncSource := func(name string) string {
		return strings.Replace(jsFunctionSource(t, script, name), "function "+name, "async function "+name, 1)
	}
	functions := []string{
		jsFunctionSource(t, script, "missionActivityCursor"),
		jsFunctionSource(t, script, "detailMissionActivityCursor"),
		jsFunctionSource(t, script, "mergeMissionActivity"),
		jsFunctionSource(t, script, "applyMissionDetail"),
		asyncSource("refreshSelectedMissionDetail"),
		asyncSource("refreshSelectedMissionActivity"),
		asyncSource("selectMission"),
		asyncSource("reloadMission"),
	}
	fixture := `
const MISSION_STORAGE_KEY="plasma.activeMissionId";
const state={missionId:"mis_1",selectionGeneration:1,detailGeneration:1,detail:{projection:{last_sequence:5,title:"old"},activity_cursor:{schema:"mission-activity/v1",sequence:5,server_id:"server-a"}},missions:[{MissionID:"mis_1",activity:{last_sequence:5,active_work:{items:[{}]}}}],missionActivityCursors:{mis_1:{schema:"mission-activity/v1",sequence:5,serverID:"server-a"}}};
let activityResolve; const detailResolves=[]; let listRefreshes=0, connectionRefreshes=0, accessRefreshes=0; const requests=[];
const api=(path)=>{requests.push(path); if(path.endsWith("/activity")) return new Promise(resolve=>{activityResolve=resolve;}); return new Promise(resolve=>detailResolves.push(resolve));};
const captureMissionSelection=()=>({missionId:state.missionId,selectionGeneration:state.selectionGeneration});
const ownsMissionSelection=(owner)=>owner.missionId===state.missionId&&owner.selectionGeneration===state.selectionGeneration;
const ownsDetailRequest=(owner)=>ownsMissionSelection(owner)&&owner.detailGeneration===state.detailGeneration;
const beginMissionSelection=(missionId)=>{state.detailGeneration++; return {missionId,selectionGeneration:state.selectionGeneration,detailGeneration:state.detailGeneration};};
const localStorage={setItem(){}}; const rememberMissionID=()=>{}; const markMissionActivitySeen=()=>{}; const renderDetail=()=>{}; const renderMissions=()=>{}; const renderMissionLoadFailed=()=>{throw new Error("reload failed");};
const refreshMissionList=async()=>{listRefreshes++;}; const loadConfluenceConnections=async()=>{connectionRefreshes++;}; const loadConfluenceAccess=async()=>{accessRefreshes++;};
` + strings.Join(functions, "\n") + `
(async()=>{
  const poll=refreshSelectedMissionActivity({missionId:"mis_1",selectionGeneration:1,detailGeneration:1});
  await Promise.resolve();
  const reload=reloadMission();
  await Promise.resolve();
  detailResolves.shift()({projection:{last_sequence:10,title:"new"},activity_cursor:{schema:"mission-activity/v1",sequence:10,server_id:"server-a"}});
  await reload;
  activityResolve({activity:{last_sequence:6,active_work:{items:[]}},cursor:{schema:"mission-activity/v1",sequence:6,server_id:"server-a"}});
  await poll;
  if(state.detail.projection.title!=="new" || detailResolves.length!==0) throw new Error("late poll replaced newer reload detail");
  if(listRefreshes!==1 || connectionRefreshes!==1 || accessRefreshes!==1) throw new Error("newer reload lost full selection refresh");
  if(requests.join()!=="/api/missions/mis_1/activity,/api/missions/mis_1") throw new Error("late poll issued fallback detail");
})().catch((err)=>{console.error(err);process.exit(1);});`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("activity poll/reload ownership fixture: %v: %s", err, out)
	}
}

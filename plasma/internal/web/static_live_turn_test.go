package web

import (
	"os/exec"
	"strings"
	"testing"
)

func TestStaticLiveTurnRendersStatusThenAnswerPreview(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	turnStateScript := string(mustReadStatic(t, "static/plasma/conversation_turn_state.js"))
	liveScript := string(mustReadStatic(t, "static/plasma/conversation_live_turn.js"))
	turnsScript := string(mustReadStatic(t, "static/plasma/conversation_turns.js"))
	fixture := `
const log = {innerHTML:"", scrollHeight:0, scrollTop:0, clientHeight:100};
const state = {missionId:"mis_1", liveTurn:null, pendingTurn:null, turnPending:true, detail:null, turnScrollMission:""};
const escapeHTML = (value) => String(value).replace(/[&<>"]/g, (ch) => ({"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;"}[ch]));
const sources = [];
const window = {Plasma:{
  state,
  dom:{ $:(id) => id === "turnLog" ? log : null, escapeHTML, escapeAttr:escapeHTML, shortID:(v)=>v, timeShort:()=>"" },
  ui:{ empty:(value)=>value },
  mission:{ captureMissionSelection:()=>({missionId:"mis_1"}), ownsMissionSelection:()=>true },
  conversation:{
    updateTurnNavVisibility:()=>{},
    _callbacks:{ reloadMission:async()=>{} }
  }
}};
var EventSource = function(url){ this.url = url; this.listeners = {}; this.closed = false; sources.push(this); this.addEventListener = (name, cb) => { this.listeners[name] = cb; }; this.close = () => { this.closed = true; }; };
function countNeedle(value, needle){ return (value.match(new RegExp(needle, "g")) || []).length; }
` + turnStateScript + `
` + liveScript + `
` + turnsScript + `
const conversation = window.Plasma.conversation;
conversation.configureRendering({renderMarkdown:(value)=>"<p>"+escapeHTML(value)+"</p>", empty:(value)=>value});
const pending = {EventType:"turn.agent.pending", EventID:"evt_pending", CreatedAt:"", Payload:{user_event_id:"evt_user", agent_executor:"codex"}};
const workflowPending = {EventType:"turn.agent.pending", EventID:"evt_workflow_pending", CreatedAt:"", Payload:{user_event_id:"evt_workflow_user", workflow_run_id:"wfr_1", agent_executor:"codex"}};
conversation.syncLiveTurnSubscription([workflowPending]);
if (state.liveTurn?.userEventID !== "evt_workflow_user" || !sources[sources.length - 1].url.includes("evt_workflow_user")) throw new Error("workflow pending turn did not subscribe to live activity");
conversation.handleLiveTurnSnapshot(JSON.stringify({schema:"plasma-live-turn/v1",mission_id:"mis_1",user_event_id:"evt_workflow_user",state:"activity",status:"웹에서 조사하는 중...",sequence:1}), "mis_1", "evt_workflow_user");
conversation.renderTurns([workflowPending]);
if (!log.innerHTML.includes("워크플로우") || !log.innerHTML.includes("웹에서 조사하는 중...") || log.innerHTML.includes("에이전트 응답을 기다리는 중...")) throw new Error("workflow activity did not replace the fixed fallback: " + log.innerHTML);
conversation.clearLiveTurn();
conversation.startLiveTurn("mis_1", "evt_user");
conversation.handleLiveTurnSnapshot(JSON.stringify({schema:"plasma-live-turn/v1",mission_id:"mis_1",user_event_id:"other",state:"answer",preview:"wrong",sequence:1}), "mis_1", "other");
conversation.handleLiveTurnSnapshot(JSON.stringify({schema:"plasma-live-turn/v1",mission_id:"mis_1",user_event_id:"evt_user",state:"activity",status:"미션 자료를 살펴보는 중...",sequence:1}), "mis_1", "evt_user");
conversation.renderTurns([pending]);
if (!log.innerHTML.includes("미션 자료를 살펴보는 중...") || log.innerHTML.includes("wrong")) throw new Error("activity status did not render safely: " + log.innerHTML);
conversation.handleLiveTurnSnapshot(JSON.stringify({schema:"plasma-live-turn/v1",mission_id:"mis_1",user_event_id:"evt_user",state:"answer",preview:"첫 답변입니다.",sequence:2}), "mis_1", "evt_user");
conversation.renderTurns([pending]);
if (!log.innerHTML.includes("첫 답변입니다.") || !log.innerHTML.includes("응답 작성 중...") || !log.innerHTML.includes("spinner") || !log.innerHTML.includes("live-turn-status-text") || !log.innerHTML.includes('role="status"') || !log.innerHTML.includes('aria-live="polite"') || !log.innerHTML.includes('aria-atomic="true"') || log.innerHTML.includes("미션 자료를 살펴보는 중...")) throw new Error("answer preview did not keep pending progress affordance: " + log.innerHTML);
conversation.handleLiveTurnSnapshot(JSON.stringify({schema:"plasma-live-turn/v1",mission_id:"mis_1",user_event_id:"evt_user",state:"answer",preview:"첫 답변입니다.",status:"웹에서 조사하는 중...",sequence:3}), "mis_1", "evt_user");
conversation.renderTurns([pending]);
if (!log.innerHTML.includes("첫 답변입니다.") || !log.innerHTML.includes("웹에서 조사하는 중...") || log.innerHTML.includes("응답 작성 중...")) throw new Error("interleaved tool status did not display with preview: " + log.innerHTML);
conversation.handleLiveTurnSnapshot(JSON.stringify({schema:"plasma-live-turn/v1",mission_id:"mis_1",user_event_id:"evt_user",state:"answer",preview:"둘째 답변입니다.",sequence:4}), "mis_1", "evt_user");
conversation.renderTurns([pending]);
if (!log.innerHTML.includes("둘째 답변입니다.") || !log.innerHTML.includes("응답 작성 중...") || log.innerHTML.includes("웹에서 조사하는 중...")) throw new Error("subsequent answer did not restore fallback status: " + log.innerHTML);

const openedBeforeFailure = sources.length;
sources[sources.length - 1].onerror();
conversation.renderTurns([pending]);
if (sources.length !== openedBeforeFailure) throw new Error("SSE failure reopened EventSource immediately");
if (!sources[sources.length - 1].closed) throw new Error("failed EventSource was not closed");
if (!log.innerHTML.includes("에이전트 응답을 기다리는 중...")) throw new Error("SSE failure did not fall back to fixed spinner: " + log.innerHTML);

state.liveTurn = null;
state.pendingTurn = {missionId:"mis_1", text:"조사해줘", createdAt:"", userEventID:"evt_user"};
conversation.startLiveTurn("mis_1", "evt_user");
conversation.handleLiveTurnSnapshot(JSON.stringify({schema:"plasma-live-turn/v1",mission_id:"mis_1",user_event_id:"evt_user",state:"activity",status:"자료를 읽는 중입니다.",sequence:1}), "mis_1", "evt_user");
conversation.renderTurns([]);
if (!log.innerHTML.includes("turn user pending") || !log.innerHTML.includes("조사해줘")) throw new Error("optimistic pre-reload row missing: " + log.innerHTML);

const durableUser = {EventType:"turn.user", EventID:"evt_user", CreatedAt:"", Payload:{text:"조사해줘"}};
conversation.renderTurns([durableUser, pending]);
if (state.pendingTurn !== null) throw new Error("optimistic pending turn was not cleared after durable events");
if (log.innerHTML.includes("turn user pending") || countNeedle(log.innerHTML, 'class="turn user') !== 1) throw new Error("optimistic row duplicated durable user: " + log.innerHTML);

const response = {EventType:"turn.agent.response", EventID:"evt_response", CreatedAt:"", Payload:{user_event_id:"evt_user", kind:"agent_response", text:"최종 답변"}};
conversation.renderTurns([durableUser, pending, response]);
if (state.liveTurn !== null) throw new Error("live state was not cleared after durable response");
if (!log.innerHTML.includes("최종 답변") || log.innerHTML.includes("미션 자료를 살펴보는 중...") || log.innerHTML.includes("응답 작성 중...") || log.innerHTML.includes("live-turn-preview") || log.innerHTML.includes("live-turn-status-text") || log.innerHTML.includes('role="status"')) throw new Error("durable response did not replace live state: " + log.innerHTML);
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("static live turn fixture: %v: %s", err, out)
	}
}

func TestStaticLiveTurnStatusTextShineCSS(t *testing.T) {
	css := string(mustReadStatic(t, "static/plasma/conversation_workflow.css"))
	start := strings.Index(css, ".live-turn-status-text")
	if start < 0 {
		t.Fatalf("missing live status CSS block")
	}
	end := strings.Index(css[start:], ".turn-markdown")
	if end < 0 {
		t.Fatalf("missing CSS block boundary")
	}
	shineBlock := css[start : start+end]

	for _, expected := range []string{
		"linear-gradient(",
		"100deg,",
		"var(--ink2) 32%",
		"var(--muted) 42%",
		"var(--ink) 50%",
		"var(--muted) 58%",
		"var(--ink2) 68%",
		"animation: live-turn-text-shine 2.2s linear infinite;",
		"background-size: 240% 100%;",
		"from { background-position: 120% 0; }",
		"to { background-position: -120% 0; }",
		"@media (prefers-reduced-motion: reduce)",
		"animation: none;",
		"-webkit-text-fill-color: var(--ink2);",
		"color: var(--ink2);",
	} {
		if !strings.Contains(shineBlock, expected) {
			t.Fatalf("missing CSS literal %q", expected)
		}
	}
	if strings.Contains(shineBlock, "var(--accent-from)") {
		t.Fatalf("status shine must not use --accent-from")
	}
}

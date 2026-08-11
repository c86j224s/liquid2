package web

import (
	"os/exec"
	"strings"
	"testing"
)

func TestStaticReportCardExposesDeleteActionInToolMenu(t *testing.T) {
	cards := string(mustReadStatic(t, "static/plasma/reports_cards_artifacts.js"))
	controls := string(mustReadStatic(t, "static/plasma/reports_controls.js"))
	events := string(mustReadStatic(t, "static/plasma/reports_events.js"))
	state := string(mustReadStatic(t, "static/plasma/state.js"))
	for _, expected := range []string{
		`reportActionMenu("도구 ▾"`,
		`data-action="delete-report-artifact">보고서 삭제</button>`,
		`class="danger"`,
	} {
		if !strings.Contains(cards, expected) {
			t.Fatalf("missing report card delete contract %q", expected)
		}
	}
	if !strings.Contains(events, `reports.deleteReportArtifact(artifactID)`) {
		t.Fatal("report event dispatcher must route delete-report-artifact")
	}
	for _, expected := range []string{
		"reportDeletePreview: null",
		"async function deleteReportArtifact",
		"/report_delete_preview",
		"confirm_artifact_id: artifactID",
		"expected_revision: preview.revision",
		"delete_facts_hash: preview.delete_facts_hash",
		"보고서를 삭제했습니다.",
		"새로고침은 실패했습니다.",
	} {
		if !strings.Contains(state+"\n"+controls, expected) {
			t.Fatalf("missing report delete control contract %q", expected)
		}
	}
	missionState := string(mustReadStatic(t, "static/plasma/mission_state.js"))
	modal := string(mustReadStatic(t, "static/plasma/reports_modal.js"))
	if !strings.Contains(missionState, "state.reportDeletePreview = null;") ||
		!strings.Contains(modal, "state.reportDeletePreview = null;") {
		t.Fatal("report delete preview must be reset on mission transient reset and report preview cleanup")
	}
}

func TestStaticReportDeletePreviewConfirmAndDeleteFlow(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	domScript := string(mustReadStatic(t, "static/plasma/dom.js"))
	formatBytes := jsFunctionSource(t, domScript, "formatBytes")
	confirmFn := jsFunctionSource(t, script, "confirmReportDelete")
	deleteFn := "async " + jsFunctionSource(t, script, "deleteReportArtifact")
	fixture := `
const state = {missionId:"mis_1",selectionGeneration:1,detail:{},reportDeletePreview:null,reportPreview:{},selectedReportKey:"key"};
const requireMission = () => true;
const captureMissionSelection = () => ({missionId:state.missionId,selectionGeneration:state.selectionGeneration});
const ownsMissionSelection = (owner) => owner && owner.missionId === state.missionId && owner.selectionGeneration === state.selectionGeneration;
let calls = [], deleteBody = null, confirms = [], notice = "";
const missionApi = async (_owner, suffix, options = {}) => {
  calls.push((options.method || "GET") + ":" + suffix);
  if (suffix === "/artifacts/art_1/report_delete_preview") {
    return {eligible:true,run_id:"evt_pending",revision:5,delete_facts_hash:"abc123",deletable_event_count:2,deletable_artifact_count:1,deletable_artifact_bytes:2048,shared_artifact_count:1,blockers:[]};
  }
  if (suffix === "/artifacts/art_1/report" && options.method === "DELETE") {
    deleteBody = options.body;
    return {deleted:true};
  }
  throw new Error("unexpected missionApi call: " + suffix);
};
const reloadMission = async (missionID) => calls.push("reload:" + missionID);
const showError = (err) => { throw err; };
const reports = { setReportNotice(message) { notice = message; } };
const window = { confirm(message) { confirms.push(message); return true; } };
` + formatBytes + `
` + confirmFn + `
` + deleteFn + `
(async () => {
  await deleteReportArtifact("art_1");
  if (calls.join("|") !== "GET:/artifacts/art_1/report_delete_preview|DELETE:/artifacts/art_1/report|reload:mis_1") throw new Error("unexpected calls: " + calls.join("|"));
  if (!deleteBody || deleteBody.confirm_artifact_id !== "art_1" || deleteBody.expected_revision !== 5 || deleteBody.delete_facts_hash !== "abc123") throw new Error("bad delete body: " + JSON.stringify(deleteBody));
  if (!confirms[0].includes("장부 이벤트 2개") || !confirms[0].includes("artifact 1개 / 2.0 KiB") || !confirms[0].includes("복구할 수 없습니다")) throw new Error("bad confirm message: " + confirms[0]);
  if (state.reportDeletePreview !== null || state.reportPreview !== null || state.selectedReportKey !== "") throw new Error("report state was not cleared");
  if (notice !== "보고서를 삭제했습니다.") throw new Error("success notice missing");
})().catch((err) => { console.error(err); process.exit(1); });
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("report delete static fixture: %v: %s", err, out)
	}
}

func TestStaticReportDeleteReloadFailureIsNotDeletionFailure(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := mustReadPlasmaReportScripts(t)
	domScript := string(mustReadStatic(t, "static/plasma/dom.js"))
	formatBytes := jsFunctionSource(t, domScript, "formatBytes")
	confirmFn := jsFunctionSource(t, script, "confirmReportDelete")
	deleteFn := "async " + jsFunctionSource(t, script, "deleteReportArtifact")
	fixture := `
const state = {missionId:"mis_1",selectionGeneration:1,detail:{},reportDeletePreview:null,reportPreview:{},selectedReportKey:"key"};
const requireMission = () => true;
const captureMissionSelection = () => ({missionId:state.missionId,selectionGeneration:state.selectionGeneration});
const ownsMissionSelection = (owner) => owner && owner.missionId === state.missionId && owner.selectionGeneration === state.selectionGeneration;
let notices = [], shown = 0;
const missionApi = async (_owner, suffix, options = {}) => {
  if (suffix === "/artifacts/art_1/report_delete_preview") return {eligible:true,run_id:"evt_pending",revision:5,delete_facts_hash:"abc123",deletable_event_count:2,deletable_artifact_count:1,deletable_artifact_bytes:512,shared_artifact_count:0,blockers:[]};
  if (suffix === "/artifacts/art_1/report" && options.method === "DELETE") return {deleted:true};
  throw new Error("unexpected missionApi call: " + suffix);
};
const reloadMission = async () => { throw new Error("reload down"); };
const showError = () => { shown++; };
const reports = { setReportNotice(message, tone) { notices.push({message, tone}); } };
const window = { confirm() { return true; } };
` + formatBytes + `
` + confirmFn + `
` + deleteFn + `
(async () => {
  await deleteReportArtifact("art_1");
  if (shown !== 0) throw new Error("reload failure was shown as delete failure");
  if (!notices.some((item) => item.message === "보고서를 삭제했습니다.")) throw new Error("delete success notice missing");
  const reloadNotice = notices.find((item) => item.message.includes("새로고침은 실패했습니다."));
  if (!reloadNotice || reloadNotice.tone !== "error" || reloadNotice.message.includes("보고서 삭제 실패")) throw new Error("bad reload failure notice: " + JSON.stringify(notices));
})().catch((err) => { console.error(err); process.exit(1); });
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("report delete reload static fixture: %v: %s", err, out)
	}
}

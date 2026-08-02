(function reportsControls(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const state = root.Plasma.state;
	  const $ = root.Plasma.dom.$;
	  const captureMissionSelection = root.Plasma.mission.captureMissionSelection;
	  const ownsMissionSelection = root.Plasma.mission.ownsMissionSelection;
	  const missionApi = root.Plasma.transport.missionApi;
  const schedulePendingPoll = root.Plasma.polling.schedulePendingPoll;
  const requireMission = () => reports.call("requireMission");
  const reloadMission = (...args) => reports.call("reloadMission", ...args);
  const showError = (...args) => reports.call("showError", ...args);
  const activeWorkBlocksControl = (...args) => reports.call("activeWorkBlocksControl", ...args);
  const missionLifecycleWriteBlocked = () => reports.call("missionLifecycleWriteBlocked");
  const selectedAgentModel = () => reports.call("selectedAgentModel");
  const selectedAgentReasoningEffort = () => reports.call("selectedAgentReasoningEffort");
async function draftReport(reportMode = "one_take") {
  if (!requireMission()) return;
	if (state.turnPending || state.workflowPending || state.workflowGoalDraftPending || state.reportPending) return;
  const owner = captureMissionSelection();
  const missionId = owner.missionId;
  const title = `${state.detail?.projection?.title || "미션"} 리포트`;
  const reportSelection = reports.modelSelection.payload($("reportAgentModel").value, $("reportAgentReasoningEffort").value);
  const executionStrategy = reportMode === "long_form"
    ? ($("reportLongFormExecutionStrategy")?.value || "serial")
    : "serial";
  const generationGuidanceProfile = reports.selectedReportGenerationGuidance(reportMode);
  const postReportHumanize = reportMode === "long_form" ? "enabled" : "disabled";
  const pendingPayload = {
    title,
    report_mode: reportMode,
    execution_strategy: executionStrategy,
    generation_guidance_profile: generationGuidanceProfile,
    post_report_humanize: postReportHumanize,
    rigor_level: $("reportRigor").value || "strict",
    agent_model: reportSelection.agent_model,
    agent_reasoning_effort: reportSelection.agent_reasoning_effort,
    direction_hint: typeof reports.direction.current === "function" ? reports.direction.current() : ""
  };
  reports.setReportBusy(true);
  reports.setReportNotice(reports.reportPendingMessage({ Payload: pendingPayload }));
  let result;
  try {
    result = await missionApi(owner, "/reports", {
      method: "POST",
      body: {
        ...pendingPayload,
        agent_executor: $("agentExecutor").value,
        mcp_mode: $("mcpMode").value
      }
    });
  } catch (err) {
    if (!ownsMissionSelection(owner)) return;
    reports.setReportNotice(`리포트 초안 생성 실패\n\n${err.userMessage || err.message || String(err)}`, "error");
    reports.setReportBusy(false);
    showError(err);
    return;
  }
	if (!ownsMissionSelection(owner)) return;
	if (typeof reports.direction.clear === "function") reports.direction.clear();
  reports.setReportNotice(result.pending_event
    ? reports.reportPendingMessage(result.pending_event)
    : reports.reportPendingMessage({ Payload: pendingPayload }));
  try {
    await reloadMission(missionId);
  } catch (err) {
    showError(err);
    schedulePendingPoll();
  }
}

async function patchReportArtifact(artifactID, currentTitle = "") {
  if (!requireMission()) return;
  if (state.reportPending) return;
  artifactID = (artifactID || "").trim();
  if (!artifactID) return;
  const instruction = window.prompt("이 리포트를 어떻게 수정할까요? 보고서 세션에서 MCP 패치로 새 버전을 만듭니다.", "");
  if (!instruction || !instruction.trim()) return;
  const owner = captureMissionSelection();
  const titleBase = (currentTitle || state.detail?.projection?.title || "리포트").trim();
  const title = `${titleBase} 수정본`;
  const selectedModel = selectedAgentModel();
    const reasoningEffort = selectedAgentReasoningEffort();
  const agentModel = state.agentModelTouched ? selectedModel : "";
    const agentReasoningEffort = state.agentModelTouched || state.agentReasoningEffortTouched ? reasoningEffort : "";
  reports.setReportBusy(true);
  reports.setReportNotice(reports.reportPendingMessage({
    EventType: "report.patch.pending",
    Payload: { title, base_artifact_id: artifactID, instruction: instruction.trim() }
  }));
  let result;
  try {
    result = await missionApi(owner, "/reports/patch", {
      method: "POST",
      body: {
        base_artifact_id: artifactID,
        instruction: instruction.trim(),
        title,
        agent_executor: $("agentExecutor").value,
        agent_model: agentModel,
        agent_reasoning_effort: agentReasoningEffort,
        mcp_mode: $("mcpMode").value
      }
    });
  } catch (err) {
    if (!ownsMissionSelection(owner)) return;
    reports.setReportNotice(`리포트 MCP 패치 시작 실패\n\n${err.userMessage || err.message || String(err)}`, "error");
    reports.setReportBusy(false);
    showError(err);
    return;
  }
  if (!ownsMissionSelection(owner)) return;
  reports.setReportNotice(result.pending_event
    ? reports.reportPendingMessage(result.pending_event)
    : reports.reportPendingMessage({ EventType: "report.patch.pending", Payload: { title, base_artifact_id: artifactID } }));
  try {
    await reloadMission(owner.missionId);
  } catch (err) {
    if (ownsMissionSelection(owner)) { showError(err); schedulePendingPoll(); }
  }
}

async function cancelReport() {
  if (!requireMission()) return;
  if (!state.reportPending) return;
  const owner = captureMissionSelection();
  try {
    await missionApi(owner, "/reports/cancel", {
      method: "POST",
      body: {}
    });
    if (!ownsMissionSelection(owner)) return;
    reports.setReportNotice("리포트 생성 취소를 요청했습니다. 장부에 취소 이벤트가 기록되면 다시 생성할 수 있습니다.");
    await reloadMission(owner.missionId);
  } catch (err) {
    if (ownsMissionSelection(owner)) showError(err);
  }
}

function setReportBusy(busy) {
  state.reportPending = busy;
	syncReportControls();
  $("reportStatus").classList.toggle("hidden", !busy);
  window.Plasma.ui.setElementDisabled("cancelReportButton", !busy || !state.detail);
  $("cancelReportButton").classList.toggle("hidden", !busy);
  window.Plasma.ui.setButtonText("draftQuickReport", busy ? "생성 중" : "보고서");
  window.Plasma.ui.setButtonText("draftLongReport", busy ? "생성 중" : "장문 보고서");
}

function syncReportControls() {
	const blocked = activeWorkBlocksControl("report_start") || state.turnPending || state.workflowPending || state.workflowGoalDraftPending || state.reportPending || missionLifecycleWriteBlocked() || !state.detail;
	window.Plasma.ui.setElementDisabled("reportRigor", blocked);
	window.Plasma.ui.setElementDisabled("reportAgentModel", blocked);
	window.Plasma.ui.setElementDisabled("reportAgentReasoningEffort", blocked);
	window.Plasma.ui.setElementDisabled("reportLongFormExecutionStrategy", blocked);
	window.Plasma.ui.setElementDisabled("draftQuickReport", blocked);
	window.Plasma.ui.setElementDisabled("draftLongReport", blocked);
}
  Object.assign(reports, { draftReport, patchReportArtifact, cancelReport, setReportBusy, syncReportControls });
})(window);

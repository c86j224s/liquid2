(function reportsControls(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const state = root.Plasma.state;
	  const $ = root.Plasma.dom.$;
	  const captureMissionSelection = root.Plasma.mission.captureMissionSelection;
	  const ownsMissionSelection = root.Plasma.mission.ownsMissionSelection;
	  const missionApi = root.Plasma.transport.missionApi;
  const formatBytes = root.Plasma.dom.formatBytes;
  const schedulePendingPoll = root.Plasma.polling.schedulePendingPoll;
  const requireMission = () => reports.call("requireMission");
  const reloadMission = (...args) => reports.call("reloadMission", ...args);
  const showError = (...args) => reports.call("showError", ...args);
  const activeWorkBlocksControl = (...args) => reports.call("activeWorkBlocksControl", ...args);
  const missionLifecycleWriteBlocked = () => reports.call("missionLifecycleWriteBlocked");
  const selectedAgentModel = () => reports.call("selectedAgentModel");
  const selectedAgentReasoningEffort = () => reports.call("selectedAgentReasoningEffort");

function activateCreationTab(panelID) {
  const tabs = Array.from(document.querySelectorAll("[data-report-creation-tab]"));
  for (const tab of tabs) {
    const selected = tab.dataset.reportCreationTab === panelID;
    tab.classList.toggle("active", selected);
    tab.setAttribute("aria-selected", selected ? "true" : "false");
    const panel = document.getElementById(tab.dataset.reportCreationTab);
    if (panel) panel.hidden = !selected;
  }
}

async function draftReport(reportMode = "one_take", options) {
  options = options || {};
  if (!requireMission()) return;
	if (state.turnPending || state.workflowPending || state.workflowGoalDraftPending || state.reportPending) return;
  const owner = captureMissionSelection();
  const missionId = owner.missionId;
  const requestedFamily = String(options.pipelineFamily || "").trim();
  const article = options.outputKind === "article";
  const rigorControl = article ? $("articleRigor") : $("reportRigor");
  const validation = String(rigorControl.value || "strict").trim() || "strict";
  const ilReport = requestedFamily === reports.REPORT_IL_PIPELINE_FAMILY;
  const longArticle = article && reportMode === "long_form";
  const pipelineFamily = longArticle
    ? reports.REPORT_IL_PIPELINE_FAMILY
    : article
      ? ""
      : ilReport
        ? reports.REPORT_IL_PIPELINE_FAMILY
        : validation === "unverified"
          ? reports.REPORT_UNVERIFIED_PIPELINE_FAMILY
          : "";
  const unverified = pipelineFamily === reports.REPORT_UNVERIFIED_PIPELINE_FAMILY;
  const independent = ilReport || longArticle || unverified;
  const longFormIL = ilReport && reportMode === "long_form";
  const familyTitle = longFormIL ? " 장문 IL" : ilReport ? " IL" : unverified ? " 무검증" : "";
  const title = longArticle ? `${state.detail?.projection?.title || "미션"} 장문 글` : article ? `${state.detail?.projection?.title || "미션"} 글` : `${state.detail?.projection?.title || "미션"}${familyTitle} 리포트`;
  const modelControl = article ? $("articleAgentModel") : $("reportAgentModel");
  const effortControl = article ? $("articleAgentReasoningEffort") : $("reportAgentReasoningEffort");
  const reportSelection = unverified
    ? { agent_model: "gpt-5.6-luna", agent_reasoning_effort: "xhigh" }
    : reports.modelSelection.payload(modelControl.value, effortControl.value);
  const executionStrategy = longArticle
    ? ($("articleLongFormExecutionStrategy")?.value || "serial")
    : independent ? "" : reportMode === "long_form"
      ? ($("reportLongFormExecutionStrategy")?.value || "serial")
      : "serial";
  const generationGuidanceProfile = independent ? "" : reports.selectedReportGenerationGuidance(reportMode);
  const postReportHumanize = independent ? "disabled" : reportMode === "long_form" ? "enabled" : "disabled";
  const pendingPayload = {
    title,
    report_mode: reportMode,
    ...(independent ? { pipeline_family: pipelineFamily } : {}),
    execution_strategy: executionStrategy,
    generation_guidance_profile: generationGuidanceProfile,
    post_report_humanize: postReportHumanize,
    rigor_level: unverified ? "unverified" : validation,
    agent_model: reportSelection.agent_model,
    agent_reasoning_effort: reportSelection.agent_reasoning_effort,
    direction_hint: typeof reports.direction.current === "function" ? reports.direction.current(article ? "article" : "report") : "",
    ...(article ? { output_kind: "article", article_intent: options.articleIntent } : {})
  };
  reports.setReportBusy(true);
  reports.setReportNotice(reports.reportPendingMessage({ Payload: pendingPayload }));
  let result;
  try {
    result = await missionApi(owner, "/reports", {
      method: "POST",
      body: {
        ...pendingPayload,
        agent_executor: independent ? "codex" : $("agentExecutor").value,
        mcp_mode: independent ? "source_read_only" : $("mcpMode").value
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
	if (typeof reports.direction.clear === "function") reports.direction.clear(article ? "article" : "report");
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

async function draftArticle(reportMode = "one_take") {
  const audience = String($("articleAudience").value || "").trim();
  const readerPromise = String($("articleReaderPromise").value || "").trim();
  const emphasis = String($("articleEmphasis").value || "").trim();
  if (!audience || !readerPromise) {
    const err = new Error("글을 읽을 사람과 읽고 나서 얻어갈 것을 입력하세요.");
    err.userMessage = err.message;
    showError(err);
    return;
  }
  await draftReport(reportMode, {
    outputKind: "article",
    articleIntent: { audience, reader_promise: readerPromise, ...(emphasis ? { emphasis } : {}) }
  });
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

async function deleteReportArtifact(artifactID) {
  if (!requireMission()) return;
  artifactID = (artifactID || "").trim();
  if (!artifactID) return;
  const owner = captureMissionSelection();
  let preview;
  try {
    preview = await missionApi(owner, `/artifacts/${encodeURIComponent(artifactID)}/report_delete_preview`);
  } catch (err) {
    if (ownsMissionSelection(owner)) showError(err);
    return;
  }
  if (!ownsMissionSelection(owner)) return;
  state.reportDeletePreview = preview;
  if (!preview.eligible) {
    const message = (preview.blockers || []).map((item) => item.message).filter(Boolean).join("\n") || "이 보고서는 삭제할 수 없습니다.";
    const err = new Error(message);
    err.userMessage = message;
    showError(err);
    return;
  }
  if (!confirmReportDelete(preview)) return;
  try {
    await missionApi(owner, `/artifacts/${encodeURIComponent(artifactID)}/report`, {
      method: "DELETE",
      body: { confirm_artifact_id: artifactID, expected_revision: preview.revision, delete_facts_hash: preview.delete_facts_hash }
    });
    if (!ownsMissionSelection(owner)) return;
    state.reportDeletePreview = null;
    state.reportPreview = null;
    state.selectedReportKey = "";
    reports.setReportNotice("보고서를 삭제했습니다.");
  } catch (err) {
    if (!ownsMissionSelection(owner)) return;
    reports.setReportNotice(`보고서 삭제 실패\n\n${err.userMessage || err.message || String(err)}`, "error");
    showError(err);
    return;
  }
  try {
    await reloadMission(owner.missionId);
  } catch (err) {
    if (ownsMissionSelection(owner)) reports.setReportNotice(`보고서를 삭제했습니다. 새로고침은 실패했습니다.\n\n${err.userMessage || err.message || String(err)}`, "error");
  }
}

function confirmReportDelete(preview) {
  const bytes = formatBytes(preview.deletable_artifact_bytes || 0) || "0 B";
  const shared = Number(preview.shared_artifact_count || 0);
  const lines = [
    `장부 이벤트 ${Number(preview.deletable_event_count || 0)}개`,
    `artifact ${Number(preview.deletable_artifact_count || 0)}개 / ${bytes}`,
    `보존되는 공유 artifact ${shared}개`
  ];
  return window.confirm(`보고서를 삭제할까요?\n\n${lines.join("\n")}\n\n삭제한 보고서는 복구할 수 없습니다.`);
}

function setReportBusy(busy) {
  state.reportPending = busy;
	syncReportControls();
  $("reportStatus").classList.toggle("hidden", !busy);
  window.Plasma.ui.setElementDisabled("cancelReportButton", !busy || !state.detail);
  $("cancelReportButton").classList.toggle("hidden", !busy);
  window.Plasma.ui.setButtonText("draftQuickReport", busy ? "생성 중" : "보고서");
  window.Plasma.ui.setButtonText("draftLongReport", busy ? "생성 중" : "장문 보고서");
  window.Plasma.ui.setButtonText("draftExperimentalReport", busy ? "생성 중" : "IL 보고서");
  window.Plasma.ui.setButtonText("draftLongExperimentalReport", busy ? "생성 중" : "장문 IL 보고서");
  window.Plasma.ui.setButtonText("draftArticle", busy ? "생성 중" : "일반 글");
  window.Plasma.ui.setButtonText("draftLongArticle", busy ? "생성 중" : "장문 글");
}

function syncReportControls() {
	const blocked = activeWorkBlocksControl("report_start") || state.turnPending || state.workflowPending || state.workflowGoalDraftPending || state.reportPending || missionLifecycleWriteBlocked() || !state.detail;
	window.Plasma.ui.setElementDisabled("reportRigor", blocked);
	window.Plasma.ui.setElementDisabled("reportAgentModel", blocked);
	window.Plasma.ui.setElementDisabled("reportAgentReasoningEffort", blocked);
	window.Plasma.ui.setElementDisabled("reportLongFormExecutionStrategy", blocked);
	window.Plasma.ui.setElementDisabled("articleRigor", blocked);
	window.Plasma.ui.setElementDisabled("articleAgentModel", blocked);
	window.Plasma.ui.setElementDisabled("articleAgentReasoningEffort", blocked);
	window.Plasma.ui.setElementDisabled("articleLongFormExecutionStrategy", blocked);
	window.Plasma.ui.setElementDisabled("articleDirectionHint", blocked);
	window.Plasma.ui.setElementDisabled("draftQuickReport", blocked);
	window.Plasma.ui.setElementDisabled("draftLongReport", blocked);
	window.Plasma.ui.setElementDisabled("draftExperimentalReport", blocked);
	window.Plasma.ui.setElementDisabled("draftLongExperimentalReport", blocked);
	window.Plasma.ui.setElementDisabled("draftArticle", blocked);
	window.Plasma.ui.setElementDisabled("draftLongArticle", blocked);
}
  Object.assign(reports, { activateCreationTab, draftReport, draftArticle, patchReportArtifact, deleteReportArtifact, cancelReport, setReportBusy, syncReportControls });
})(window);

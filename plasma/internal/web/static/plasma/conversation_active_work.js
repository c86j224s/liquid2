(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const { $, escapeHTML, escapeAttr } = Plasma.dom;
  const conversation = Plasma.conversation = Plasma.conversation || {};

  function renderActiveWork(activeWork) {
    const items = displayActiveWorkItems(activeWork);
    const message = items
      .map((item) => ({ item, text: activeWorkMessage(item) }))
      .filter((entry) => entry.text);
    for (const id of ["conversationActiveWork", "reportActiveWork"]) {
      const el = $(id);
      if (!el) continue;
      el.classList.toggle("hidden", message.length === 0);
      el.innerHTML = message.map((entry) => `<div class="active-work-item">${escapeHTML(entry.text)}${activeWorkActionHTML(entry.item)}</div>`).join("");
    }
    applyActiveWorkDescriptions(activeWork);
  }

  function displayActiveWorkItems(activeWork) {
    const items = activeWork.items || [];
    const hasAgentTurn = items.some((item) => item?.kind === "agent_turn" || item?.reason_code === "agent_turn_running");
    if (!state.turnPending || hasAgentTurn) return items;
    return [
      ...items,
      { kind: "agent_turn", reason_code: "agent_turn_running", action: "cancel_turn" }
    ];
  }

  function activeWorkMessage(work) {
    switch (work?.reason_code) {
      case "report_generation_running": return "리포트 생성 중입니다.";
      case "workflow_running": return "자율 진행 중입니다.";
      case "agent_turn_running": return "에이전트가 응답 중입니다.";
      default: return "";
    }
  }

  function activeWorkActionHTML(work) {
    if (!work?.action) return "";
    const labels = { cancel_turn: "응답 취소", cancel_report: "생성 취소", view_workflow: "자율 진행 보기" };
    return `<button type="button" class="secondary" data-active-work-action="${escapeAttr(work.action)}">${escapeHTML(labels[work.action] || "진행 상태 보기")}</button>`;
  }

  function applyActiveWorkDescriptions(activeWork) {
    const descriptions = {
      turn_submit: "conversationActiveWork",
      workflow_start: "conversationActiveWork",
      report_start: "reportActiveWork"
    };
    for (const [control, descriptionID] of Object.entries(descriptions)) {
      const blocked = (activeWork.blocked_controls || []).some((item) => item.control === control);
      for (const id of activeWorkControlElementIDs(control)) {
        const el = $(id);
        if (!el) continue;
        if (blocked) el.setAttribute("aria-describedby", descriptionID);
        else el.removeAttribute("aria-describedby");
      }
    }
  }

  function activeWorkControlElementIDs(control) {
    if (control === "turn_submit") return ["turnText", "sendTurnButton"];
    if (control === "workflow_start") return ["workflowInstruction", "draftWorkflowGoalButton", "workflowRunGoal", "workflowStepInstruction", "startWorkflowButton"];
    if (control === "report_start") return ["reportRigor", "reportAgentModel", "reportAgentReasoningEffort", "reportLongFormExecutionStrategy", "draftQuickReport", "draftLongReport"];
    return [];
  }

  Object.assign(conversation, {
    renderActiveWork,
    displayActiveWorkItems,
    activeWorkMessage,
    activeWorkActionHTML,
    applyActiveWorkDescriptions,
    activeWorkControlElementIDs
  });
})(window.Plasma);

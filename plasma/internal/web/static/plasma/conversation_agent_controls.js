(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const { $, escapeHTML, escapeAttr } = Plasma.dom;
  const conversation = Plasma.conversation = Plasma.conversation || {};

  function renderAgentOptions(statuses) {
    const select = $("agentExecutor");
    if (!select) return;
    const current = select.value || "codex";
    const locked = conversation.lockedAgentExecutor();
    const known = statuses.length ? statuses : [
      { name: "codex", label: "Codex", configured: true },
      { name: "claude", label: "Claude", configured: false }
    ];
    select.innerHTML = known.map((status) => {
      const lockedOut = locked && status.name !== locked;
      const disabled = status.configured && !lockedOut ? "" : "disabled";
      const lockLabel = locked && status.name === locked ? " · 이 미션에서 사용 중" : "";
      const label = `${status.label || status.name}${status.configured ? "" : " 준비 중"}${lockLabel}`;
      return `<option value="${escapeAttr(status.name)}" ${disabled}>${escapeHTML(label)}</option>`;
    }).join("");
    if (locked && [...select.options].some((option) => option.value === locked)) {
      select.value = locked;
      return;
    }
    if ([...select.options].some((option) => option.value === current && !option.disabled)) {
      select.value = current;
      return;
    }
    const firstAvailable = [...select.options].find((option) => !option.disabled);
    if (firstAvailable) select.value = firstAvailable.value;
  }

  function onAgentExecutorChange() {
    state.agentModelTouched = false;
    state.agentReasoningEffortTouched = false;
    conversation.renderAgentModelOptions(state.detail?.events || []);
    conversation.renderAgentReasoningEffortOptions(state.detail?.events || []);
    conversation._callbacks.renderReportModelSelection(state.detail?.agent_executors || [], $("agentExecutor").value);
    const effortSelect = $("agentReasoningEffort");
    if (effortSelect) {
      const blocked = conversation._callbacks.agentControlsBlocked();
      effortSelect.disabled = conversation.agentReasoningEffortSelectionDisabled(blocked);
    }
    conversation.renderAgentSessionStatus(state.detail?.events || []);
    conversation.renderAgentControlsSummary();
  }

  function onAgentModelChange() {
    state.agentModelTouched = true;
    conversation.renderAgentReasoningEffortOptions(state.detail?.events || [], true);
    conversation.renderAgentControlsSummary();
  }

  function onAgentReasoningEffortChange() {
    state.agentReasoningEffortTouched = true;
    conversation.renderAgentControlsSummary();
  }

  Object.assign(conversation, {
    renderAgentOptions,
    onAgentExecutorChange,
    onAgentModelChange,
    onAgentReasoningEffortChange
  });
})(window.Plasma);

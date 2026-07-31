(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const conversation = Plasma.conversation = Plasma.conversation || {};

  function agentExecutorStatus(executor) {
    const statuses = state.detail?.agent_executors || [];
    return statuses.find((status) => status.name === executor) || null;
  }

  function lockedAgentExecutor() {
    return (state.detail?.locked_agent_executor || "").trim();
  }

  function agentExecutorSelectionDisabled(baseDisabled) {
    return baseDisabled || Boolean(lockedAgentExecutor());
  }

  function agentReasoningEffortSelectionDisabled(baseDisabled) {
    const executor = $("agentExecutor")?.value || "codex";
    return baseDisabled || !agentReasoningEffortSupported(executor, agentExecutorStatus(executor));
  }

  function agentReasoningEffortSupported(executor, status) {
    if (status && typeof status.reasoning_effort_supported === "boolean") {
      return status.reasoning_effort_supported;
    }
    return executor === "codex";
  }

  function selectedAgentModel() {
    return ($("agentModel")?.value || "").trim();
  }

  function selectedAgentReasoningEffort() {
    const executor = $("agentExecutor")?.value || "codex";
    const status = agentExecutorStatus(executor);
    if (!agentReasoningEffortSupported(executor, status)) return "";
    return ($("agentReasoningEffort")?.value || status?.default_reasoning_effort || "medium").trim() || "medium";
  }

  function agentEventMatchesExecutor(eventExecutor, executor) {
    const eventName = String(eventExecutor || "").trim();
    const executorName = String(executor || "codex").trim();
    if (!eventName) return executorName === "codex";
    return eventName === executorName;
  }

  Object.assign(conversation, {
    agentExecutorStatus,
    lockedAgentExecutor,
    agentExecutorSelectionDisabled,
    agentReasoningEffortSelectionDisabled,
    agentReasoningEffortSupported,
    selectedAgentModel,
    selectedAgentReasoningEffort,
    agentEventMatchesExecutor
  });
})(window.Plasma);

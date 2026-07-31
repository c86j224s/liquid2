(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const { $, escapeHTML, escapeAttr } = Plasma.dom;
  const conversation = Plasma.conversation = Plasma.conversation || {};

  const AGENT_MODEL_OPTIONS = {
    claude: [
      { value: "", label: "기본값" },
      { value: "haiku", label: "Claude Haiku (haiku)" },
      { value: "sonnet", label: "Claude Sonnet (sonnet)" },
      { value: "opus", label: "Claude Opus (opus)" }
    ]
  };

  const AGENT_REASONING_EFFORT_OPTIONS = {
    claude: [
      { value: "", label: "지정 불가" }
    ]
  };

  function renderAgentModelOptions(events) {
    const select = $("agentModel");
    if (!select) return;
    const executor = $("agentExecutor")?.value || "codex";
    if (state.agentModelExecutor !== executor) {
      state.agentModelTouched = false;
      state.agentModelExecutor = executor;
    }
    const status = conversation.agentExecutorStatus(executor);
    const options = agentModelOptions(executor, status);
    const saved = currentAgentModel(events || [], executor);
    const preferred = state.agentModelTouched ? select.value : saved;
    select.innerHTML = options.map((option) =>
      `<option value="${escapeAttr(option.value)}">${escapeHTML(agentModelOptionLabel(option, status))}</option>`
    ).join("");
    if (preferred && !options.some((option) => option.value === preferred)) {
      select.insertAdjacentHTML("beforeend", `<option value="${escapeAttr(preferred)}">${escapeHTML(`저장된 모델: ${preferred}`)}</option>`);
    }
    select.value = preferred || "";
  }

  function agentModelOptions(executor, status) {
    if (executor === "codex") {
      const catalog = Array.isArray(status?.models) ? status.models : [];
      return [{ value: "", label: "기본값" }, ...catalog.map((model) => ({
        value: String(model.name || "").trim(),
        label: String(model.label || model.name || "").trim()
      })).filter((model) => model.value)];
    }
    return AGENT_MODEL_OPTIONS[executor] || [{ value: "", label: "기본값" }];
  }

  function agentModelOptionLabel(option, status) {
    if (option.value !== "") return option.label;
    const defaultText = agentDefaultModelText(status);
    return defaultText ? `기본값 (${defaultText})` : option.label;
  }

  function agentDefaultModelText(status) {
    if (!status) return "";
    const label = String(status.default_model_label || "").trim();
    const version = String(status.default_model_version || status.default_model || "").trim();
    if (label && version && label !== version) return `${label}, ${version}`;
    return label || version;
  }

  function renderAgentReasoningEffortOptions(events, modelChanged = false) {
    const select = $("agentReasoningEffort");
    if (!select) return;
    const executor = $("agentExecutor")?.value || "codex";
    if (state.agentReasoningEffortExecutor !== executor) {
      state.agentReasoningEffortTouched = false;
      state.agentReasoningEffortExecutor = executor;
    }
    const status = conversation.agentExecutorStatus(executor);
    const options = agentReasoningEffortOptions(executor, status);
    const saved = currentAgentReasoningEffort(events || [], executor);
    const supported = conversation.agentReasoningEffortSupported(executor, status);
    const defaultEffort = String(status?.default_reasoning_effort || "medium").trim();
    const defaultSelection = options.some((option) => option.value === defaultEffort) ? defaultEffort : (options[0]?.value || "");
    const preferred = supported
      ? (modelChanged ? defaultSelection : (state.agentReasoningEffortTouched ? select.value : (saved || defaultSelection)))
      : "";
    select.innerHTML = options.map((option) =>
      `<option value="${escapeAttr(option.value)}">${escapeHTML(option.label)}</option>`
    ).join("");
    if (!modelChanged && preferred && !options.some((option) => option.value === preferred)) {
      select.insertAdjacentHTML("beforeend", `<option value="${escapeAttr(preferred)}">${escapeHTML(`저장된 추론 강도: ${preferred}`)}</option>`);
    }
    select.value = preferred;
    select.title = supported ? "" : (status?.reasoning_effort_note || "이 에이전트는 추론 강도 지정을 지원하지 않습니다.");
  }

  function agentReasoningEffortOptions(executor, status) {
    if (executor === "codex") {
      const model = conversation.selectedAgentModel() || String(status?.default_model || "").trim();
      const catalog = Array.isArray(status?.models) ? status.models : [];
      const selected = catalog.find((item) => String(item.name || "").trim() === model);
      const efforts = Array.isArray(selected?.reasoning_efforts) ? selected.reasoning_efforts : ["low", "medium", "high", "xhigh"];
      return efforts.map((effort) => ({ value: String(effort), label: String(effort).replace(/^./, (letter) => letter.toUpperCase()) }));
    }
    return AGENT_REASONING_EFFORT_OPTIONS[executor] || [{ value: "", label: "지정 불가" }];
  }

  function currentAgentModel(events, executor) {
    for (let index = events.length - 1; index >= 0; index--) {
      const event = events[index];
      const payload = event.Payload || {};
      if (!conversation.agentEventMatchesExecutor(payload.agent_executor || "", executor)) continue;
      if (event.EventType === "agent.session.reset") return (payload.agent_model || "").trim();
      if (event.EventType === "turn.agent.response" && payload.kind === "agent_response" && payload.agent_model) {
        return String(payload.agent_model || "").trim();
      }
    }
    return "";
  }

  function currentAgentReasoningEffort(events, executor) {
    for (let index = events.length - 1; index >= 0; index--) {
      const event = events[index];
      const payload = event.Payload || {};
      if (!conversation.agentEventMatchesExecutor(payload.agent_executor || "", executor)) continue;
      if (event.EventType === "agent.session.reset") return (payload.agent_reasoning_effort || "").trim();
      if (event.EventType === "turn.agent.response" && payload.kind === "agent_response" && payload.agent_reasoning_effort) {
        return String(payload.agent_reasoning_effort || "").trim();
      }
    }
    return "";
  }

  Object.assign(conversation, {
    renderAgentModelOptions,
    renderAgentReasoningEffortOptions,
    currentAgentModel,
    currentAgentReasoningEffort
  });
})(window.Plasma);

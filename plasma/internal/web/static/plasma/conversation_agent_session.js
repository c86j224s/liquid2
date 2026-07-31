(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const { $, shortID } = Plasma.dom;
  const mission = Plasma.mission;
  const transport = Plasma.transport;
  const conversation = Plasma.conversation = Plasma.conversation || {};

  async function resetAgentSession() {
    if (!conversation._callbacks.requireMission()) return;
    const executor = $("agentExecutor").value;
    const selectedModel = conversation.selectedAgentModel();
    const selectedReasoningEffort = conversation.selectedAgentReasoningEffort();
    const model = state.agentModelTouched ? selectedModel : "";
    const reasoningEffort = state.agentModelTouched || state.agentReasoningEffortTouched ? selectedReasoningEffort : "";
    const modelText = selectedModel ? ` 모델 ${selectedModel}로` : "";
    const effortText = selectedReasoningEffort ? `, 추론 강도 ${selectedReasoningEffort}` : "";
    if (!window.confirm(`${executor}${modelText}${effortText} 세션을 새로 시작할까요? Plasma 미션과 저장된 소스는 유지됩니다.`)) return;
    const owner = mission.captureMissionSelection();
    try {
      await transport.missionApi(owner, "/agent_sessions/reset", {
        method: "POST",
        body: { agent_executor: executor, agent_model: model, agent_reasoning_effort: reasoningEffort }
      });
      if (!mission.ownsMissionSelection(owner)) return;
      state.agentModelTouched = false;
      state.agentReasoningEffortTouched = false;
      await conversation._callbacks.reloadMission(owner.missionId);
    } catch (err) {
      if (!mission.isStaleMissionOperation(err) && mission.ownsMissionSelection(owner)) conversation._callbacks.showError(err);
    }
  }

  function renderAgentSessionStatus(events) {
    const el = $("agentSessionStatus");
    if (!el) return;
    const executor = $("agentExecutor").value || "codex";
    const model = conversation.currentAgentModel(events, executor);
    const modelText = model ? ` · 모델 ${model}` : "";
    const reasoningEffort = conversation.currentAgentReasoningEffort(events, executor);
    const effortText = reasoningEffort ? ` · 추론 ${reasoningEffort}` : "";
    el.classList.remove("ready", "live");
    el.removeAttribute("title");
    for (let index = events.length - 1; index >= 0; index--) {
      const event = events[index];
      const payload = event.Payload || {};
      if (!conversation.agentEventMatchesExecutor(payload.agent_executor || "", executor)) continue;
      if (event.EventType === "agent.session.reset") {
        const previousID = payload.previous_agent_session_id || "";
        el.textContent = previousID
          ? `${executor} 새 세션 준비됨${modelText}${effortText} · 이전 ${shortID(previousID)}`
          : `${executor} 새 세션 준비됨${modelText}${effortText}`;
        if (previousID || model || reasoningEffort) el.title = [
          model ? `모델: ${model}` : "",
          reasoningEffort ? `추론 강도: ${reasoningEffort}` : "",
          previousID ? `이전 세션: ${previousID}` : ""
        ].filter(Boolean).join("\n");
        el.classList.add("ready");
        renderAgentControlsSummary();
        return;
      }
      if (event.EventType === "turn.agent.response" && payload.kind === "agent_response" && payload.agent_session_id) {
        const isNew = payload.resumed === false;
        el.textContent = `${executor} ${isNew ? "새 세션" : "현재 세션"}${modelText}${effortText} · ${shortID(payload.agent_session_id)}`;
        el.title = [
          model ? `모델: ${model}` : "",
          reasoningEffort ? `추론 강도: ${reasoningEffort}` : "",
          `현재 세션: ${payload.agent_session_id}`
        ].filter(Boolean).join("\n");
        el.classList.add("live");
        renderAgentControlsSummary();
        return;
      }
    }
    el.textContent = `${executor} 세션 없음`;
    renderAgentControlsSummary();
  }

  function renderAgentControlsSummary() {
    const summaryText = $("agentControlsSummaryText");
    if (!summaryText) return;
    const executor = $("agentExecutor")?.value || "codex";
    const strategy = $("controllerStrategy")?.value || "auto";
    const locked = conversation.lockedAgentExecutor();
    const statusEl = $("agentSessionStatus");
    const statusText = statusEl ? statusEl.textContent.trim() : "";
    const selectedModel = conversation.selectedAgentModel();
    const model = state.agentModelTouched ? selectedModel : (selectedModel || conversation.currentAgentModel(state.detail?.events || [], executor));
    const selectedReasoningEffort = conversation.selectedAgentReasoningEffort();
    const status = conversation.agentExecutorStatus(executor);
    const reasoningEffort = state.agentReasoningEffortTouched
      ? selectedReasoningEffort
      : (selectedReasoningEffort || conversation.currentAgentReasoningEffort(state.detail?.events || [], executor) || status?.default_reasoning_effort || "");
    const strategyText = strategy === "auto" ? "자동 조향" : strategy.toUpperCase();
    const lockText = locked ? "미션 에이전트 고정" : "에이전트 선택 가능";
    const modelText = model ? `모델 ${model}` : "기본 모델";
    const effortText = status?.reasoning_effort_supported === false
      ? "추론 지정 불가"
      : `추론 ${reasoningEffort || "medium"}`;
    summaryText.textContent = statusText
      ? `${executor} · ${modelText} · ${effortText} · ${strategyText} · ${lockText} · ${statusText}`
      : `${executor} · ${modelText} · ${effortText} · ${strategyText} · ${lockText}`;
  }

  Object.assign(conversation, {
    resetAgentSession,
    renderAgentSessionStatus,
    renderAgentControlsSummary
  });
})(window.Plasma);

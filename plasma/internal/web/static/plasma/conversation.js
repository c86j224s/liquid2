(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const mission = Plasma.mission;
  const transport = Plasma.transport;
  const ui = Plasma.ui;
  const conversation = Plasma.conversation = Plasma.conversation || {};
  const callbacks = conversation._callbacks || {
    requireMission: () => Boolean(state.missionId),
    reloadMission: (missionId) => mission.refreshSelectedMissionDetail({ ...mission.captureMissionSelection(), missionId: missionId || state.missionId }),
    showError: (err) => console.error(err),
    syncReportControls: () => {},
    turnControlsBlocked: (busy) => Boolean(busy),
    agentControlsBlocked: () => false,
    renderReportModelSelection: () => {}
  };

  function configure(options = {}) {
    Object.assign(callbacks, options);
    conversation._callbacks = callbacks;
  }

  async function sendTurn(event) {
    event.preventDefault();
    if (!callbacks.requireMission()) return;
    const text = $("turnText").value.trim();
    if (!text) return;
    const owner = mission.captureMissionSelection();
    const missionId = owner.missionId;
    state.turnPending = true;
    state.pendingTurn = {
      missionId,
      text,
      agentExecutor: $("agentExecutor").value,
      mcpMode: $("mcpMode").value,
      controllerStrategy: $("controllerStrategy").value,
      createdAt: new Date().toISOString()
    };
    setTurnBusy(true);
    callbacks.syncReportControls();
    $("turnText").value = "";
    if (state.detail) conversation.renderTurns(state.detail.events || []);
    try {
      const response = await transport.missionApi(owner, "/turns", {
        method: "POST",
        body: {
          text,
          agent_executor: $("agentExecutor").value,
          mcp_mode: $("mcpMode").value,
          controller_strategy: $("controllerStrategy").value
        }
      });
      if (!mission.ownsMissionSelection(owner)) return;
      const userEventID = response?.user_event?.EventID || "";
      if (userEventID) {
        state.pendingTurn = { ...(state.pendingTurn || {}), userEventID };
        conversation.startLiveTurn?.(missionId, userEventID);
        if (state.detail) conversation.renderTurns(state.detail.events || []);
      }
      await callbacks.reloadMission(missionId);
    } catch (err) {
      if (!mission.ownsMissionSelection(owner)) return;
      callbacks.showError(err);
      state.turnPending = false;
      state.pendingTurn = null;
      setTurnBusy(false);
      callbacks.syncReportControls();
      if (state.missionId === missionId) {
        $("turnText").value = text;
        if (state.detail) conversation.renderTurns(state.detail.events || []);
      }
    }
  }

  async function cancelTurn() {
    if (!callbacks.requireMission()) return;
    const owner = mission.captureMissionSelection();
    try {
      await transport.missionApi(owner, "/turns/cancel", {
        method: "POST",
        body: {}
      });
      if (!mission.ownsMissionSelection(owner)) return;
      conversation.clearLiveTurnForMission?.(owner.missionId);
      state.pendingTurn = null;
      await callbacks.reloadMission(owner.missionId);
    } catch (err) {
      if (mission.ownsMissionSelection(owner)) callbacks.showError(err);
    }
  }

  function setTurnBusy(busy) {
    conversation.renderActiveWork(state.detail?.active_work || {});
    const blocked = callbacks.turnControlsBlocked(busy);
    ui.setElementDisabled("turnText", blocked);
    ui.setElementDisabled("agentExecutor", conversation.agentExecutorSelectionDisabled(blocked));
    ui.setElementDisabled("agentModel", blocked);
    ui.setElementDisabled("agentReasoningEffort", conversation.agentReasoningEffortSelectionDisabled(blocked));
    ui.setElementDisabled("mcpMode", blocked);
    ui.setElementDisabled("controllerStrategy", blocked);
    ui.setElementDisabled("resetAgentSessionButton", blocked);
    ui.setElementDisabled("sendTurnButton", blocked);
    ui.setButtonText("sendTurnButton", busy ? "대기 중" : "보내기");
  }

  Object.assign(conversation, {
    _callbacks: callbacks,
    configure,
    sendTurn,
    cancelTurn,
    setTurnBusy
  });
})(window.Plasma);

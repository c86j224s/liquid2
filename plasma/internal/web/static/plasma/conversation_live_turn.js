(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const mission = Plasma.mission;
  const conversation = Plasma.conversation = Plasma.conversation || {};

  function startLiveTurn(missionId, userEventID) {
    missionId = String(missionId || "");
    userEventID = String(userEventID || "");
    if (!missionId || !userEventID) return;
    if (state.liveTurn?.missionId === missionId && state.liveTurn?.userEventID === userEventID) return;
    clearLiveTurn();
    state.liveTurn = { missionId, userEventID, snapshot: null, source: null, terminal: false, failed: false };
    if (typeof EventSource !== "function") return;
    const source = new EventSource(`/api/missions/${encodeURIComponent(missionId)}/turns/${encodeURIComponent(userEventID)}/live`);
    state.liveTurn.source = source;
    source.addEventListener("snapshot", (event) => handleLiveTurnSnapshot(event.data, missionId, userEventID));
    source.onerror = () => {
      if (!state.liveTurn || state.liveTurn.terminal || state.liveTurn.missionId !== missionId || state.liveTurn.userEventID !== userEventID) return;
      state.liveTurn.failed = true;
      closeLiveTurnSource();
      if (state.detail) conversation.renderTurns(state.detail.events || []);
    };
  }

  function handleLiveTurnSnapshot(raw, missionId, userEventID) {
    let snapshot;
    try {
      snapshot = JSON.parse(raw);
    } catch {
      return;
    }
    if (!validLiveTurnSnapshot(snapshot, missionId, userEventID)) return;
    if (!state.liveTurn || state.liveTurn.missionId !== missionId || state.liveTurn.userEventID !== userEventID) return;
    if (state.missionId !== missionId) return;
    state.liveTurn.snapshot = snapshot;
    if (snapshot.terminal) {
      state.liveTurn.terminal = true;
      closeLiveTurnSource();
      conversation._callbacks?.reloadMission?.(missionId);
    }
    if (state.detail) conversation.renderTurns(state.detail.events || []);
  }

  function validLiveTurnSnapshot(snapshot, missionId, userEventID) {
    if (!snapshot || snapshot.schema !== "plasma-live-turn/v1") return false;
    if (snapshot.mission_id !== missionId || snapshot.user_event_id !== userEventID) return false;
    if (!Number.isSafeInteger(snapshot.sequence) || snapshot.sequence < 1) return false;
    if (!["activity", "answer", "completed", "error", "canceled"].includes(snapshot.state)) return false;
    return true;
  }

  function syncLiveTurnSubscription(events) {
    const pending = latestPendingTurn(events || []);
    if (pending) {
      startLiveTurn(state.missionId, pending);
      return;
    }
    if (state.pendingTurn?.missionId === state.missionId && state.pendingTurn.userEventID) return;
    clearLiveTurnForMission(state.missionId);
  }

  function latestPendingTurn(events) {
    const completed = conversation.completedUserEventIDs(events);
    for (let i = events.length - 1; i >= 0; i--) {
      const event = events[i];
      if (event.EventType !== "turn.agent.pending") continue;
      const payload = event.Payload || {};
      const userEventID = payload.user_event_id ? String(payload.user_event_id) : "";
      if (!userEventID || completed.has(userEventID)) continue;
      return userEventID;
    }
    return "";
  }

  function reconcileLiveTurnForDurableEvents(events) {
    events = events || [];
    const completed = conversation.completedUserEventIDs(events);
    if (state.liveTurn?.missionId === state.missionId && completed.has(state.liveTurn.userEventID)) {
      clearLiveTurn();
    }
    const pending = state.pendingTurn;
    if (!pending || pending.missionId !== state.missionId || !pending.userEventID) return;
    const userEventID = String(pending.userEventID);
    if (completed.has(userEventID) || hasDurableUserAndPending(events, userEventID)) {
      state.pendingTurn = null;
    }
  }

  function hasDurableUserAndPending(events, userEventID) {
    let hasUser = false;
    let hasPending = false;
    for (const event of events) {
      if (event.EventType === "turn.user" && String(event.EventID || "") === userEventID) hasUser = true;
      if (event.EventType === "turn.agent.pending" && String(event.Payload?.user_event_id || "") === userEventID) hasPending = true;
      if (hasUser && hasPending) return true;
    }
    return false;
  }

  function liveTurnSnapshot(userEventID) {
    userEventID = String(userEventID || "");
    if (!state.liveTurn || state.liveTurn.missionId !== state.missionId || state.liveTurn.userEventID !== userEventID) return null;
    if (state.liveTurn.failed) return null;
    const snapshot = state.liveTurn.snapshot;
    if (!snapshot || snapshot.terminal) return null;
    return snapshot;
  }

  function clearLiveTurnForMission(missionId) {
    if (!state.liveTurn || state.liveTurn.missionId !== missionId) return;
    clearLiveTurn();
  }

  function clearLiveTurn() {
    closeLiveTurnSource();
    state.liveTurn = null;
  }

  function closeLiveTurnSource() {
    const source = state.liveTurn?.source;
    if (source && typeof source.close === "function") source.close();
    if (state.liveTurn) state.liveTurn.source = null;
  }

  Object.assign(conversation, {
    startLiveTurn,
    handleLiveTurnSnapshot,
    validLiveTurnSnapshot,
    reconcileLiveTurnForDurableEvents,
    syncLiveTurnSubscription,
    latestPendingTurn,
    liveTurnSnapshot,
    clearLiveTurnForMission,
    clearLiveTurn
  });
})(window.Plasma);

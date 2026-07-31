(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const mission = Plasma.mission;
  const callbacks = {
    api: (path) => Plasma.transport.api(path),
    refreshSelectedMissionDetail: (owner) => mission.refreshSelectedMissionDetail(owner),
    renderMissions: () => {},
    selectedPending: () => state.turnPending || state.reportPending || state.workflowPending,
    onPendingPollFailure: () => {},
    onPendingPollSuccess: () => {}
  };

  function configure(options = {}) {
    Object.assign(callbacks, options);
  }

  function schedulePendingPoll() {
    clearPendingPoll();
    if (!callbacks.selectedPending() || !state.missionId) return;
    const owner = { ...mission.captureMissionSelection(), detailGeneration: state.detailGeneration };
    state.pollTimer = window.setTimeout(async () => {
      if (!mission.ownsDetailRequest(owner) || (state.pollInFlight && state.pollOwner && mission.ownsDetailRequest(state.pollOwner)) || !callbacks.selectedPending()) return;
      state.pollInFlight = true;
      state.pollOwner = owner;
      try {
        await refreshSelectedMissionActivity(owner);
        if (state.pollOwner === owner && mission.ownsMissionSelection(owner)) callbacks.onPendingPollSuccess();
      } catch (err) {
        if (state.pollOwner === owner && mission.ownsMissionSelection(owner)) {
          console.warn("pending poll failed", err);
          callbacks.onPendingPollFailure();
        }
      } finally {
        if (state.pollOwner !== owner) return;
        state.pollInFlight = false;
        state.pollOwner = null;
        if (mission.ownsMissionSelection(owner) && callbacks.selectedPending()) schedulePendingPoll();
      }
    }, 2000);
  }

  function clearPendingPoll() {
    if (!state.pollTimer) return;
    window.clearTimeout(state.pollTimer);
    state.pollTimer = 0;
  }

  function scheduleMissionActivityPoll() {
    if (state.missionActivityPollTimer) {
      window.clearTimeout(state.missionActivityPollTimer);
      state.missionActivityPollTimer = 0;
    }
    const selectedPending = callbacks.selectedPending();
    if (!state.missions.some((item) => (item.activity?.active_work?.items || []).length > 0 && (item.MissionID !== state.missionId || !selectedPending))) return;
    state.missionActivityPollTimer = window.setTimeout(async () => {
      state.missionActivityPollTimer = 0;
      if (!document.hidden && !state.missionActivityPollInFlight) {
        state.missionActivityPollInFlight = true;
        try {
          await refreshObservedMissionActivity();
        } catch (err) {
          console.warn("mission activity poll failed", err);
        } finally {
          state.missionActivityPollInFlight = false;
        }
      }
      scheduleMissionActivityPoll();
    }, 3000);
  }

  async function refreshObservedMissionActivity() {
    const missionIDs = state.missions
      .filter((item) => (item.activity?.active_work?.items || []).length > 0)
      .map((item) => item.MissionID)
      .filter(Boolean);
    const selectedOwner = { ...mission.captureMissionSelection(), detailGeneration: state.detailGeneration };
    const selectedMissionID = selectedOwner.missionId;
    const selectedPending = callbacks.selectedPending();
    if (!selectedPending && missionIDs.includes(selectedMissionID)) await refreshSelectedMissionActivity(selectedOwner);
    const responses = await Promise.all(missionIDs.filter((missionID) => missionID !== selectedMissionID).map(async (missionID) => [
      missionID,
      await callbacks.api(`/api/missions/${encodeURIComponent(missionID)}/activity`)
    ]));
    const activityByMissionID = new Map(responses.map(([missionID, response]) => [missionID, response.activity || {}]));
    state.missions = state.missions.map((item) => {
      const activity = activityByMissionID.get(item.MissionID);
      if (!activity || (item.activity?.last_sequence || 0) > (activity.last_sequence || 0)) return item;
      return { ...item, activity };
    });
    callbacks.renderMissions();
  }

  function missionActivityCursor(response) {
    const cursor = response?.cursor;
    if (!cursor || cursor.schema !== "mission-activity/v1" || !Number.isSafeInteger(cursor.sequence) || cursor.sequence < 0 || typeof cursor.server_id !== "string" || !cursor.server_id) return null;
    return { schema: cursor.schema, sequence: cursor.sequence, serverID: cursor.server_id };
  }

  function detailMissionActivityCursor(detail) {
    return missionActivityCursor({ cursor: detail?.activity_cursor });
  }

  function recordDetailActivityCursor(missionID, detail) {
    const cursor = detailMissionActivityCursor(detail);
    if (cursor) state.missionActivityCursors[missionID] = cursor;
    return cursor;
  }

  function mergeMissionActivity(missionID, activity) {
    state.missions = state.missions.map((item) => {
      if (item.MissionID !== missionID || !activity || (item.activity?.last_sequence || 0) > (activity.last_sequence || 0)) return item;
      return { ...item, activity };
    });
    callbacks.renderMissions();
  }

  // Polling trusts only a typed cursor from this server instance. A missing,
  // regressed, or skipped cursor performs one selected-detail recovery; it never
  // reloads connector settings or the global mission list.
  async function refreshSelectedMissionActivity(owner = mission.captureMissionSelection()) {
    if (owner && owner.detailGeneration === undefined) owner = { ...owner, detailGeneration: state.detailGeneration };
    if (!owner.missionId || !mission.ownsDetailRequest(owner)) return "stale";
    const response = await callbacks.api(`/api/missions/${encodeURIComponent(owner.missionId)}/activity`);
    if (!mission.ownsDetailRequest(owner)) return "stale";
    const cursor = missionActivityCursor(response);
    const previous = state.missionActivityCursors[owner.missionId];
    const detailCursor = detailMissionActivityCursor(state.detail);
    const detailSequence = detailCursor?.sequence;
    mergeMissionActivity(owner.missionId, response.activity || {});
    if (!cursor || !previous || !detailCursor || previous.serverID !== cursor.serverID || cursor.sequence < detailSequence || cursor.sequence > detailSequence + 1) {
      if (cursor) state.missionActivityCursors[owner.missionId] = cursor;
      await callbacks.refreshSelectedMissionDetail(owner);
      return "fallback";
    }
    state.missionActivityCursors[owner.missionId] = cursor;
    if (cursor.sequence > detailSequence) {
      await callbacks.refreshSelectedMissionDetail(owner);
      return "advanced";
    }
    return "unchanged";
  }

  Plasma.polling = {
    clearPendingPoll,
    configure,
    detailMissionActivityCursor,
    mergeMissionActivity,
    missionActivityCursor,
    recordDetailActivityCursor,
    refreshObservedMissionActivity,
    refreshSelectedMissionActivity,
    scheduleMissionActivityPoll,
    schedulePendingPoll
  };
})(window.Plasma);

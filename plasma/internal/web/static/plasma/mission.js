(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const callbacks = {
    resetTransientState: () => {},
    forgetStoredMissionID: () => {},
    rememberMissionID: () => {},
	    markMissionActivitySeen: () => {},
	    recordDetailActivityCursor: () => null,
	    renderDetail: () => {},
	    renderMissions: () => {},
	    renderSelectionCleared: () => {},
	    beforeSelectionChange: () => true,
	    afterSelectionApplied: async () => {},
	    api: (path) => Plasma.transport.api(path)
	  };

  function configure(options = {}) {
    Object.assign(callbacks, options);
  }

  function call(name, ...args) {
    const fn = callbacks[name];
    if (typeof fn !== "function") throw new Error(`Plasma.mission missing dependency: ${name}`);
    return fn(...args);
  }

  function captureMissionSelection() {
    return { missionId: state.missionId, selectionGeneration: state.selectionGeneration };
  }

  function ownsMissionSelection(owner) {
    return Boolean(owner) && state.missionId === owner.missionId && state.selectionGeneration === owner.selectionGeneration;
  }

  function ownsDetailRequest(owner) {
    return ownsMissionSelection(owner) && state.detailGeneration === owner.detailGeneration;
  }

  class StaleMissionOperationError extends Error {
    constructor() {
      super("mission selection changed");
      this.name = "StaleMissionOperationError";
    }
  }

  function isStaleMissionOperation(err) {
    return err instanceof StaleMissionOperationError;
  }

  function beginMissionSelection(missionId) {
    const changed = state.missionId !== missionId;
    if (changed) state.selectionGeneration += 1;
    state.detailGeneration += 1;
    state.missionId = missionId;
    if (changed) callbacks.resetTransientState();
    return { missionId, selectionGeneration: state.selectionGeneration, detailGeneration: state.detailGeneration };
  }

	  function clearMissionSelection() {
	    state.selectionGeneration += 1;
    state.detailGeneration += 1;
    state.missionId = "";
    callbacks.forgetStoredMissionID();
    callbacks.resetTransientState();
    callbacks.renderSelectionCleared();
	    callbacks.renderMissions();
	  }

	  function beforeSelectionChange(currentMissionId, nextMissionId) {
	    return callbacks.beforeSelectionChange(currentMissionId, nextMissionId);
	  }

	  async function afterSelectionApplied(owner) {
	    await callbacks.afterSelectionApplied(owner);
	  }

  function applyMissionDetail(owner, detail) {
    if (!ownsDetailRequest(owner)) return false;
    state.detail = detail;
    const cursor = callbacks.recordDetailActivityCursor(owner.missionId, detail);
    callbacks.rememberMissionID(owner.missionId);
    callbacks.markMissionActivitySeen(owner.missionId, cursor?.sequence ?? detail.projection?.last_sequence);
    callbacks.renderDetail();
    callbacks.renderMissions();
    return true;
  }

  async function refreshSelectedMissionDetail(owner = captureMissionSelection()) {
    if (!owner.missionId || !ownsMissionSelection(owner)) return false;
    const detailOwner = { ...owner, detailGeneration: ++state.detailGeneration };
    const detail = await callbacks.api(`/api/missions/${encodeURIComponent(owner.missionId)}`);
    return applyMissionDetail(detailOwner, detail);
  }

  Plasma.mission = {
	    StaleMissionOperationError,
	    afterSelectionApplied,
	    applyMissionDetail,
	    beginMissionSelection,
	    beforeSelectionChange,
	    captureMissionSelection,
    call,
    clearMissionSelection,
    configure,
    ownsMissionSelection,
    ownsDetailRequest,
    refreshSelectedMissionDetail,
    isStaleMissionOperation
  };
})(window.Plasma);

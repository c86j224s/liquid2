(function (Plasma) {
  "use strict";

  const MISSION_STORAGE_KEY = "plasma.activeMissionId";

  function storedMissionID() {
    try {
      return localStorage.getItem(MISSION_STORAGE_KEY) || "";
    } catch (_) {
      return "";
    }
  }

  function rememberMissionID(missionID) {
    try {
      localStorage.setItem(MISSION_STORAGE_KEY, missionID);
    } catch (_) {
      // Restoring the last selection is optional and must not fail a loaded mission.
    }
  }

  function forgetStoredMissionID() {
    try {
      localStorage.removeItem(MISSION_STORAGE_KEY);
    } catch (_) {
      // Clearing the last selection is optional and must not block UI recovery.
    }
  }

  Object.assign(Plasma.mission, { forgetStoredMissionID, rememberMissionID, storedMissionID });
})(window.Plasma);

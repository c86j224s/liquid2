(function (Plasma) {
  "use strict";

  const MISSION_ACTIVITY_SEEN_STORAGE_KEY = "plasma.missionActivitySeen.v1";

  function missionActivitySeenWatermarks() {
    try {
      const parsed = JSON.parse(localStorage.getItem(MISSION_ACTIVITY_SEEN_STORAGE_KEY) || "{}");
      if (!parsed || Array.isArray(parsed) || typeof parsed !== "object") return {};
      return Object.fromEntries(Object.entries(parsed).filter(([missionID, sequence]) =>
        typeof missionID === "string" && missionID.startsWith("mis_") && Number.isSafeInteger(sequence) && sequence >= 0
      ));
    } catch (_) {
      return {};
    }
  }

  function missionActivitySeenSequence(missionID, watermarks = missionActivitySeenWatermarks()) {
    const value = watermarks[missionID];
    return Number.isSafeInteger(value) && value >= 0 ? value : 0;
  }

  function markMissionActivitySeen(missionID, sequence) {
    if (typeof missionID !== "string" || !missionID.startsWith("mis_") || !Number.isSafeInteger(sequence) || sequence < 0) return;
    try {
      const watermarks = missionActivitySeenWatermarks();
      watermarks[missionID] = Math.max(missionActivitySeenSequence(missionID, watermarks), sequence);
      localStorage.setItem(MISSION_ACTIVITY_SEEN_STORAGE_KEY, JSON.stringify(watermarks));
    } catch (_) {
      // Browser storage is optional; activity remains server-derived without it.
    }
  }

  function pruneMissionActivitySeenWatermarks(missions) {
    const missionIDs = new Set((missions || []).map((mission) => mission.MissionID).filter((missionID) => typeof missionID === "string"));
    try {
      const watermarks = missionActivitySeenWatermarks();
      const retained = Object.fromEntries(Object.entries(watermarks).filter(([missionID]) => missionIDs.has(missionID)));
      if (Object.keys(retained).length !== Object.keys(watermarks).length) {
        localStorage.setItem(MISSION_ACTIVITY_SEEN_STORAGE_KEY, JSON.stringify(retained));
      }
    } catch (_) {
      // Browser storage is optional; stale local read state must not block the list.
    }
  }

  Object.assign(Plasma.polling, {
    markMissionActivitySeen,
    missionActivitySeenSequence,
    missionActivitySeenWatermarks,
    pruneMissionActivitySeenWatermarks
  });
})(window.Plasma);

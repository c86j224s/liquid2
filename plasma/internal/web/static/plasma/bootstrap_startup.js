(function bootstrapStartupModule(root) {
  "use strict";

  const Plasma = root.Plasma;
  const $ = Plasma.dom.$;
  const api = Plasma.transport.api;
  const MISSION_RAIL_COLLAPSED_STORAGE_KEY = "plasma.missionRailCollapsed.v1";

  async function boot() {
    try {
      const health = await api("/api/health");
      $("healthBadge").textContent = health.Status || "정상";
      await Plasma.mission.loadRuntimeInfo();
    } catch (err) {
      Plasma.ui.showError(err);
      $("healthBadge").textContent = "오프라인";
    }
    await Plasma.sources.loadLocalPathRoots();
    await Plasma.mission.loadMissions();
    Plasma.polling.scheduleMissionActivityPoll();
  }

  function initMissionRailToggle() {
    const button = $("missionRailToggle");
    if (!button) return;
    setMissionRailCollapsed(readMissionRailCollapsed(), false);
    button.addEventListener("click", () => {
      const collapsed = !document.body.classList.contains("mission-rail-collapsed");
      setMissionRailCollapsed(collapsed, true);
    });
  }

  function readMissionRailCollapsed() {
    try {
      return localStorage.getItem(MISSION_RAIL_COLLAPSED_STORAGE_KEY) === "true";
    } catch {
      return false;
    }
  }

  function setMissionRailCollapsed(collapsed, persist) {
    document.body.classList.toggle("mission-rail-collapsed", collapsed);
    const button = $("missionRailToggle");
    if (button) {
      button.setAttribute("aria-pressed", collapsed ? "true" : "false");
      button.setAttribute("aria-label", collapsed ? "미션 목록 펼치기" : "미션 목록 접기");
      button.title = collapsed ? "미션 목록 펼치기" : "미션 목록 접기";
      const glyph = button.querySelector("[aria-hidden='true']");
      if (glyph) glyph.textContent = collapsed ? "›" : "‹";
    }
    if (!persist) return;
    try {
      localStorage.setItem(MISSION_RAIL_COLLAPSED_STORAGE_KEY, collapsed ? "true" : "false");
    } catch {
      // The rail preference is local UI state. Losing it must not block the app.
    }
  }

  Object.assign(Plasma.bootstrap, { boot, initMissionRailToggle, readMissionRailCollapsed, setMissionRailCollapsed });
})(window);

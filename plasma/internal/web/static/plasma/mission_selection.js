(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const api = Plasma.transport.api;
  const mission = Plasma.mission;

	  async function selectMission(missionId) {
	    if (!missionId) return;
	    if (!mission.beforeSelectionChange(state.missionId, missionId)) return;
	    const owner = mission.beginMissionSelection(missionId);
	    try {
	      const detail = await api(`/api/missions/${encodeURIComponent(missionId)}`);
	      if (!mission.ownsDetailRequest(owner)) return;
	      mission.applyMissionDetail(owner, detail);
	      mission.refreshMissionList(owner).catch((err) => console.warn("mission activity refresh failed", err));
	      await mission.afterSelectionApplied(owner);
	    } catch (err) {
	      if (mission.ownsDetailRequest(owner)) mission.renderMissionLoadFailed();
	    }
  }

  async function reloadMission(missionId = state.missionId) {
    if (!missionId || missionId !== state.missionId) return;
    await selectMission(missionId);
  }

  function onMissionListClick(event) {
    const button = event.target.closest("[data-mission-id]");
    if (button) selectMission(button.dataset.missionId);
  }

  function showMissionRecall() {
    if (!state.detail) {
      mission.call("showError", new Error("먼저 미션을 선택하거나 만들어야 합니다."));
      return;
    }
    Plasma.ui.showDetail("현재 미션 상태", state.detail.recall || {});
  }

  Object.assign(mission, { onMissionListClick, reloadMission, selectMission, showMissionRecall });
})(window.Plasma);

(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const missionApi = Plasma.transport.missionApi;
  const captureMissionSelection = Plasma.mission.captureMissionSelection;
  const ownsMissionSelection = Plasma.mission.ownsMissionSelection;
  const isStaleMissionOperation = Plasma.mission.isStaleMissionOperation;
  const requireMission = () => sources.dependency("requireMission")();
  const reloadMission = (...args) => sources.dependency("reloadMission")(...args);
  const showError = (...args) => sources.dependency("showError")(...args);
  const renderDetail = (...args) => sources.dependency("renderDetail")(...args);

  async function toggleRemovedSources(event) {
    state.showRemovedSources = Boolean(event.target.checked);
    await refreshSourcesOnly();
  }

  async function refreshSourcesOnly() {
    if (!state.missionId || !state.detail) return;
    const owner = captureMissionSelection();
    try {
      const query = state.showRemovedSources ? "?include_removed=true" : "";
      const result = await missionApi(owner, `/sources${query}`);
      if (!ownsMissionSelection(owner)) return;
      state.detail.sources = result.sources || result.Sources || [];
      renderDetail();
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
    }
  }

  async function removeSource(snapshotID) {
    if (!requireMission()) return;
    if (!window.confirm("이 소스를 active 사용과 리포트에서 제외할까요? 저장된 artifact나 로컬 파일은 삭제하지 않습니다.")) return;
    const owner = captureMissionSelection();
    try {
      await missionApi(owner, `/sources/${encodeURIComponent(snapshotID)}/remove`, {
        method: "POST",
        body: { reason: "Removed from Plasma UI" }
      });
      await reloadMission(owner.missionId);
      if (state.showRemovedSources) await refreshSourcesOnly();
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
    }
  }

  async function restoreSource(snapshotID) {
    if (!requireMission()) return;
    const owner = captureMissionSelection();
    try {
      await missionApi(owner, `/sources/${encodeURIComponent(snapshotID)}/restore`, {
        method: "POST",
        body: {}
      });
      await reloadMission(owner.missionId);
      if (state.showRemovedSources) await refreshSourcesOnly();
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
    }
  }

  async function readSource(snapshotID) {
    if (!requireMission()) return;
    const owner = captureMissionSelection();
    try {
      const result = await missionApi(owner, `/sources/${encodeURIComponent(snapshotID)}/read?max_bytes=20000`);
      if (!ownsMissionSelection(owner)) return;
      showDetail("소스 읽기", result);
      await reloadMission(owner.missionId);
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
    }
  }

  Object.assign(sources, {
    toggleRemovedSources,
    refreshSourcesOnly,
    removeSource,
    restoreSource,
    readSource
  });
})(window.Plasma);

(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const formatBytes = Plasma.dom.formatBytes;
  const api = Plasma.transport.api;
  const missionApi = Plasma.transport.missionApi;
  const mission = Plasma.mission;

  async function loadRuntimeInfo() {
    const runtime = await api("/api/runtime");
    const label = (runtime.environment_label || "").trim();
    const badge = $("environmentBadge");
    if (!badge) return;
    if (!label) {
      badge.classList.add("hidden");
      badge.textContent = "";
      return;
    }
    badge.textContent = label;
    badge.classList.remove("hidden");
  }

  async function loadMissions() {
    try {
      const data = await api(missionListPath());
      state.missions = data.missions || [];
      if (state.showArchivedMissions) Plasma.polling.pruneMissionActivitySeenWatermarks(state.missions);
      mission.renderMissions();
      const missionIDs = new Set(state.missions.map((item) => item.MissionID).filter(Boolean));
      const savedMissionID = mission.storedMissionID();
      const nextMissionID = missionIDs.has(state.missionId) ? state.missionId :
        (missionIDs.has(savedMissionID) ? savedMissionID : state.missions[0]?.MissionID);
      if (nextMissionID) await mission.selectMission(nextMissionID);
      Plasma.polling.scheduleMissionActivityPoll();
    } catch (err) {
      mission.call("showError", err);
    }
  }

  async function createMission(event) {
    event.preventDefault();
    const selectionGeneration = state.selectionGeneration;
    const body = { title: $("missionTitle").value, objective: $("missionObjective").value, scope: { included: [], excluded: [] } };
    try {
      const detail = await api("/api/missions", { method: "POST", body });
      if (state.selectionGeneration !== selectionGeneration) return;
      $("missionTitle").value = "";
      $("missionObjective").value = "";
      const owner = mission.captureMissionSelection();
      await refreshMissionList(owner);
      if (!mission.ownsMissionSelection(owner)) return;
      await mission.selectMission(detail.projection.mission_id);
    } catch (err) {
      if (state.selectionGeneration === selectionGeneration) mission.call("showError", err);
    }
  }

  async function refreshMissionList(owner = mission.captureMissionSelection()) {
    const data = await api(missionListPath());
    if (!mission.ownsMissionSelection(owner)) throw new mission.StaleMissionOperationError();
    state.missions = data.missions || [];
    if (state.showArchivedMissions) Plasma.polling.pruneMissionActivitySeenWatermarks(state.missions);
    mission.renderMissions();
    Plasma.polling.scheduleMissionActivityPoll();
  }

  function missionListPath() {
    return state.showArchivedMissions ? "/api/missions?include_archived=true" : "/api/missions";
  }

  function onIncludeArchivedMissionsChange(event) {
    state.showArchivedMissions = Boolean(event.target.checked);
    loadMissions();
  }

  async function changeMissionLifecycle(action) {
    if (!mission.call("requireMission") || state.missionLifecyclePending || !confirmMissionLifecycleChange(action)) return;
    const owner = mission.captureMissionSelection();
    state.missionLifecyclePending = true;
    mission.renderMissionLifecycleControls();
    try {
      await missionApi(owner, `/${action}`, { method: "POST", body: { reason: "" } });
      if (!mission.ownsMissionSelection(owner)) return;
      if (action === "archive") {
        state.missionLifecyclePending = false;
        await selectMissionAfterArchive(owner);
        return;
      }
      await mission.selectMission(owner.missionId);
    } catch (err) {
      if (mission.ownsMissionSelection(owner)) mission.call("showError", err);
    } finally {
      if (mission.ownsMissionSelection(owner)) {
        state.missionLifecyclePending = false;
        mission.renderMissionLifecycleControls();
        mission.call("setFormsEnabled", Boolean(state.detail));
      }
    }
  }

  function confirmMissionLifecycleChange(action) {
    const projection = state.detail?.projection || {};
    const missionTitle = projection.title || projection.mission_id || state.missionId || "현재 미션";
    if (action === "archive") return window.confirm(`${missionTitle}을(를) 보관할까요?\n\n보관하면 기본 미션 목록에서 숨겨지고, 보관된 미션 보기에서 다시 찾을 수 있습니다.`);
    if (action === "restore") return window.confirm(`${missionTitle}을(를) 복원할까요?\n\n복원하면 기본 미션 목록에 다시 표시되고 새 작업을 이어갈 수 있습니다.`);
    return false;
  }

  async function selectMissionAfterArchive(owner) {
    await refreshMissionList(owner);
    const nextMissionID = state.missions.find((item) => item.MissionID && item.MissionID !== owner.missionId)?.MissionID;
    if (nextMissionID) await mission.selectMission(nextMissionID);
    else mission.clearMissionSelection();
  }

  async function hardDeleteMission() {
    if (!mission.call("requireMission") || state.missionHardDeletePending) return;
    const owner = mission.captureMissionSelection();
    state.missionHardDeletePending = true;
    mission.renderMissionLifecycleControls();
    try {
      const preview = await missionApi(owner, "/hard_delete_preview");
      if (!mission.ownsMissionSelection(owner)) return;
      if (!preview.eligible) return showHardDeleteBlocked(preview);
      if (!confirmMissionHardDelete(preview)) return;
      await missionApi(owner, "", { method: "DELETE", body: { confirm_mission_id: owner.missionId } });
      if (!mission.ownsMissionSelection(owner)) return;
      state.missionHardDeletePending = false;
      await selectMissionAfterHardDelete(owner);
    } catch (err) {
      if (mission.ownsMissionSelection(owner)) mission.call("showError", err);
    } finally {
      if (mission.ownsMissionSelection(owner)) {
        state.missionHardDeletePending = false;
        mission.renderMissionLifecycleControls();
        mission.call("setFormsEnabled", Boolean(state.detail));
      }
    }
  }

  function showHardDeleteBlocked(preview) {
    const reasons = (preview.blocking_reasons || []).map((reason) => reason.message).filter(Boolean).join("\n");
    const err = new Error(reasons || "이 미션은 완전 삭제할 수 없습니다.");
    err.userMessage = err.message;
    mission.call("showError", err);
  }

  function confirmMissionHardDelete(preview) {
    const title = preview.title || preview.mission_id || state.missionId || "현재 미션";
    const lines = missionHardDeleteImpactLines(preview.impact || {});
    const summary = lines.length ? lines.join("\n") : "삭제할 연결 데이터가 거의 없습니다.";
    return window.confirm(`${title}을(를) 완전 삭제할까요?\n\n이 작업은 복구할 수 없습니다.\n\n삭제 영향:\n${summary}\n\n저장공간 회수와 민감정보 흔적 제거는 별도 작업입니다.`);
  }

  function missionHardDeleteImpactLines(impact) {
    const rows = [["source_snapshots", "소스"], ["raw_artifacts", "원문 데이터"], ["ledger_events", "장부 이벤트"], ["reports", "보고서"], ["report_versions", "보고서 버전"], ["report_blocks", "보고서 블록"], ["evidence_records", "근거"], ["claim_records", "주장"], ["question_records", "질문"], ["option_records", "선택지"], ["proposal_bundles", "제안"]];
    const lines = rows.map(([key, label]) => [Number(impact[key] || 0), label]).filter(([count]) => count > 0).map(([count, label]) => `- ${label} ${count}개`);
    const bytes = formatBytes(impact.raw_artifact_bytes);
    if (bytes) lines.push(`- 원문 데이터 용량 ${bytes}`);
    return lines;
  }

  async function selectMissionAfterHardDelete(owner) {
    await refreshMissionList(owner);
    const nextMissionID = state.missions.find((item) => item.MissionID && item.MissionID !== owner.missionId)?.MissionID;
    if (nextMissionID) await mission.selectMission(nextMissionID);
    else mission.clearMissionSelection();
  }

  Object.assign(mission, { changeMissionLifecycle, createMission, hardDeleteMission, loadMissions, loadRuntimeInfo, onIncludeArchivedMissionsChange, refreshMissionList });
})(window.Plasma);

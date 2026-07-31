(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const escapeAttr = Plasma.dom.escapeAttr;
  const shortID = Plasma.dom.shortID;
  const empty = Plasma.ui.empty;
  const updateCountChip = Plasma.ui.updateCountChip;

  function renderMissions() {
    const list = $("missionList");
    const n = state.missions.length;
    const watermarks = Plasma.polling.missionActivitySeenWatermarks();
    updateCountChip("missionListCount", n);
    if ($("includeArchivedMissions")) $("includeArchivedMissions").checked = state.showArchivedMissions;
    if (n === 0) {
      list.innerHTML = empty(state.showArchivedMissions ? "표시할 미션 없음" : "미션 없음");
      return;
    }
    list.innerHTML = state.missions.map((mission) => {
      const archived = missionLifecycleState(mission) === "archived";
      return `
    <button class="item secondary ${mission.MissionID === state.missionId ? "active" : ""} ${archived ? "mission-archived" : ""}"
      type="button" data-mission-id="${escapeAttr(mission.MissionID)}" title="${escapeAttr(mission.Title || mission.MissionID)}">
      <div class="item-title-row">
        <div class="item-title">${escapeHTML(mission.Title || mission.MissionID)}</div>
        ${renderMissionActivity(mission, watermarks)}
      </div>
      ${archived ? `<div class="item-meta mission-item-meta"><span class="mission-lifecycle-pill">보관됨</span></div>` : ""}
    </button>
  `;
    }).join("");
  }

  function missionLifecycleState(value) {
    const raw = value?.lifecycle_state ?? value?.LifecycleState ?? value?.projection?.lifecycle_state ?? "";
    return String(raw).trim() === "archived" ? "archived" : "active";
  }

  function currentMissionArchived() {
    return missionLifecycleState(state.detail?.projection) === "archived";
  }

  function missionLifecycleWriteBlocked() {
    return currentMissionArchived() || state.missionLifecyclePending;
  }

  function renderMissionLifecycleControls() {
    const hasDetail = Boolean(state.detail);
    const archived = currentMissionArchived();
    const activeWork = state.detail?.active_work?.items || [];
    const blocked = !hasDetail || state.missionLifecyclePending || state.missionHardDeletePending || activeWork.length > 0;
    const text = $("missionLifecycleSettingsText");
    const hardDeleteSettings = $("missionHardDeleteSettings");
    const hardDeleteText = $("missionHardDeleteText");
    $("missionLifecycleBadge").classList.toggle("hidden", !archived);
    $("missionLifecycleNotice").classList.toggle("hidden", !archived);
    $("missionArchiveButton").classList.toggle("hidden", !hasDetail || archived);
    $("missionRestoreButton").classList.toggle("hidden", !hasDetail || !archived);
    if (hardDeleteSettings) hardDeleteSettings.classList.toggle("hidden", !hasDetail || !archived);
    $("missionArchiveButton").disabled = blocked || archived;
    $("missionRestoreButton").disabled = blocked || !archived;
    if ($("missionHardDeleteButton")) $("missionHardDeleteButton").disabled = blocked || !archived;
    if (!text) return;
    if (!hasDetail) text.textContent = "미션을 선택하면 보관 또는 복원할 수 있습니다.";
    else if (activeWork.length > 0) text.textContent = "진행 중인 작업이 있어 보관 또는 복원을 잠시 막았습니다.";
    else if (archived) text.textContent = "이 미션은 보관되어 있습니다. 복원하면 기본 미션 목록에 다시 표시됩니다.";
    else text.textContent = "보관하면 기본 미션 목록에서 숨겨지고, 보관된 미션 보기에서 다시 찾을 수 있습니다.";
    if (!hardDeleteText) return;
    if (state.missionHardDeletePending) hardDeleteText.textContent = "완전 삭제 영향을 확인하는 중입니다.";
    else if (activeWork.length > 0) hardDeleteText.textContent = "진행 중인 작업이 있어 완전 삭제할 수 없습니다.";
    else hardDeleteText.textContent = "이 보관된 미션의 앱 기록을 복구할 수 없게 삭제합니다. 저장공간 회수와 민감정보 흔적 제거는 별도 작업입니다.";
  }

  function renderMissionActivity(mission, watermarks = Plasma.polling.missionActivitySeenWatermarks()) {
    const activity = mission.activity || {};
    if ((activity.active_work?.items || []).length > 0) return missionActivityIndicator("running", "미션 작업 진행 중");
    const latest = activity.latest_terminal_activity;
    if (!latest || !Number.isSafeInteger(latest.sequence) || latest.sequence <= Plasma.polling.missionActivitySeenSequence(mission.MissionID, watermarks)) return "";
    if (latest.outcome === "failed") return missionActivityIndicator("failed", "확인하지 않은 미션 작업 실패");
    if (latest.outcome === "completed") return missionActivityIndicator("completed", "확인하지 않은 미션 작업 완료");
    return "";
  }

  function missionActivityIndicator(kind, label) {
    return `<span class="mission-activity mission-activity-${kind}" title="${label}"><span class="mission-activity-mark" aria-hidden="true"></span><span class="sr-only">${label}</span></span>`;
  }

  function renderMissionLoading() {
    $("missionName").textContent = "미션 불러오는 중";
    $("missionObjectiveText").textContent = "선택한 미션의 현재 작업을 불러오고 있습니다.";
    $("turnLog").innerHTML = empty("미션 불러오는 중");
  }

  function renderMissionLoadFailed() {
    $("missionName").textContent = "미션을 불러오지 못했습니다.";
    $("missionObjectiveText").textContent = "네트워크 또는 서버 상태를 확인한 뒤 다시 시도하세요.";
    $("turnLog").innerHTML = `<div class="empty-state">미션 상세를 불러오지 못했습니다. <button type="button" class="secondary" data-retry-mission-load>다시 시도</button></div>`;
    Plasma.mission.call("setFormsEnabled", false);
    renderMissionLifecycleControls();
  }

  function missionArtifactPreviewURL(missionID, artifactID) {
    return `/api/missions/${encodeURIComponent(missionID)}/artifacts/${encodeURIComponent(artifactID)}/preview`;
  }

  Object.assign(Plasma.mission, {
    currentMissionArchived,
    missionArtifactPreviewURL,
    missionLifecycleState,
    missionLifecycleWriteBlocked,
    renderMissionLifecycleControls,
    renderMissionLoadFailed,
    renderMissionLoading,
    renderMissions
  });
})(window.Plasma);

function missionMetadataLines(value) {
  return String(value || "").split(/\r?\n/).map((item) => item.trim()).filter(Boolean);
}

function missionMetadataFieldIDs() {
  return ["missionMetadataTitle", "missionMetadataObjective", "missionMetadataIncluded", "missionMetadataExcluded", "missionMetadataSave", "missionMetadataCancel"];
}

function setMissionMetadataDisabled(disabled) {
  for (const id of missionMetadataFieldIDs()) {
    const el = $(id);
    if (el) el.disabled = disabled;
  }
}

function renderMissionMetadataEditor(projection, force = false) {
  const form = $("missionMetadataForm");
  if (!form) return;
  if (!projection?.mission_id) {
    form.dataset.missionId = "";
    form.dataset.dirty = "false";
    $("missionMetadataTitle").value = "";
    $("missionMetadataObjective").value = "";
    $("missionMetadataIncluded").value = "";
    $("missionMetadataExcluded").value = "";
    $("missionMetadataError").textContent = "";
    $("missionSettingsStatus").textContent = "미션을 선택하면 제목, 목표, 범위와 보관 상태를 관리할 수 있습니다.";
    setMissionMetadataDisabled(true);
    return;
  }
  if (!force && form.dataset.missionId === projection.mission_id && form.dataset.dirty === "true") return;
  $("missionMetadataTitle").value = projection.title || "";
  $("missionMetadataObjective").value = projection.objective || "";
  $("missionMetadataIncluded").value = (projection.scope?.included || []).join("\n");
  $("missionMetadataExcluded").value = (projection.scope?.excluded || []).join("\n");
  $("missionMetadataError").textContent = "";
  $("missionSettingsStatus").textContent = "이 미션의 제목, 목표, 포함 범위, 제외 범위를 편집합니다.";
  form.dataset.missionId = projection.mission_id;
  form.dataset.dirty = "false";
  setMissionMetadataDisabled(false);
}

document.addEventListener("DOMContentLoaded", () => {
  const form = $("missionMetadataForm");
  if (!form) return;
  for (const id of ["missionMetadataTitle", "missionMetadataObjective", "missionMetadataIncluded", "missionMetadataExcluded"]) {
    $(id).addEventListener("input", () => {
      form.dataset.dirty = "true";
    });
  }
  $("missionMetadataCancel").addEventListener("click", () => {
    $("missionMetadataError").textContent = "";
    renderMissionMetadataEditor(state.detail?.projection, true);
  });
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!state.missionId) return;
    const owner = captureMissionSelection();
    $("missionMetadataError").textContent = "";
    try {
      await missionApi(owner, "", { method: "PATCH", body: {
        title: $("missionMetadataTitle").value,
        objective: $("missionMetadataObjective").value,
        scope: { included: missionMetadataLines($("missionMetadataIncluded").value), excluded: missionMetadataLines($("missionMetadataExcluded").value) }
      } });
      if (!ownsMissionSelection(owner)) return;
      form.dataset.dirty = "false";
      await loadMissions();
      await reloadMission();
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) $("missionMetadataError").textContent = err.userMessage || err.message;
    }
  });
  renderMissionMetadataEditor(null, true);
});

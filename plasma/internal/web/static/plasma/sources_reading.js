(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const missionApi = Plasma.transport.missionApi;
  const captureMissionSelection = Plasma.mission.captureMissionSelection;
  const ownsMissionSelection = Plasma.mission.ownsMissionSelection;
  const isStaleMissionOperation = Plasma.mission.isStaleMissionOperation;
  const requireMission = () => sources.dependency("requireMission")();
  const reloadMission = (...args) => sources.dependency("reloadMission")(...args);
  const showError = (...args) => sources.dependency("showError")(...args);
  const showDetail = Plasma.ui.showDetail;
  const openDetailModal = Plasma.ui.openDetailModal;

  const SOURCE_READ_MAX_BYTES = 20000;

  async function readSource(snapshotID) {
    if (!requireMission()) return;
    const owner = captureMissionSelection();
    try {
      const result = await fetchSourceChunk(owner, snapshotID, 0);
      if (!ownsMissionSelection(owner)) return;
      if (typeof result.content !== "string") {
        showDetail("소스 읽기", result);
        if (ownsMissionSelection(owner)) await reloadMission(owner.missionId);
        return;
      }
      state.sourceReading = sourceReadingState(snapshotID, owner, result, "");
      renderSourceReading();
      if (ownsMissionSelection(owner)) await reloadMission(owner.missionId);
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
    }
  }

  async function loadMoreSourceReading() {
    const reading = state.sourceReading;
    if (!reading || reading.loading || !canLoadMoreSourceReading(reading)) return;
    const owner = reading.owner;
    if (!ownsMissionSelection(owner)) return;
    reading.loading = true;
    renderSourceReading();
    try {
      const result = await fetchSourceChunk(owner, reading.snapshotID, reading.nextOffset);
      if (!ownsMissionSelection(owner) || state.sourceReading !== reading) return;
      if (typeof result.content !== "string") {
        if (sourceReadingIsVisible()) showDetail("소스 읽기", result);
        return;
      }
      const visible = sourceReadingIsVisible();
      state.sourceReading = sourceReadingState(reading.snapshotID, owner, result, reading.content);
      if (visible) renderSourceReading();
      if (ownsMissionSelection(owner)) await reloadMission(owner.missionId);
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner) && state.sourceReading === reading) {
        reading.loading = false;
        if (sourceReadingIsVisible()) {
          renderSourceReading();
          showError(err);
        }
      }
    }
  }

  function fetchSourceChunk(owner, snapshotID, offset) {
    const query = `offset=${encodeURIComponent(offset)}&max_bytes=${SOURCE_READ_MAX_BYTES}`;
    return missionApi(owner, `/sources/${encodeURIComponent(snapshotID)}/read?${query}`);
  }

  function sourceReadingState(snapshotID, owner, result, previousContent) {
    const content = previousContent + result.content;
    return {
      snapshotID,
      owner,
      content,
      nextOffset: result.next_offset ?? result.NextOffset ?? null,
      truncated: Boolean(result.truncated ?? result.Truncated),
      loading: false
    };
  }

  function renderSourceReading() {
    const reading = state.sourceReading;
    if (!reading) return;
    state.detailText = reading.content;
    $("detailTitle").textContent = "소스 읽기";
    $("detailBody").innerHTML = `
      <pre>${escapeHTML(reading.content)}</pre>
      <div class="item-meta">${escapeHTML(sourceReadingStatus(reading))}</div>
      ${canLoadMoreSourceReading(reading) ? `<div class="inline-actions"><button type="button" data-source-read-more ${reading.loading ? "disabled" : ""}>${reading.loading ? "불러오는 중…" : "더 보기"}</button></div>` : ""}
    `;
    $("detailBody").querySelector("[data-source-read-more]")?.addEventListener("click", loadMoreSourceReading);
    openDetailModal();
  }

  function sourceReadingStatus(reading) {
    if (reading.loading) return "불러오는 중…";
    return reading.truncated ? "저장된 내용이 더 있습니다." : "저장된 내용을 모두 불러왔습니다.";
  }

  function canLoadMoreSourceReading(reading) {
    return Boolean(reading?.truncated && reading.nextOffset !== null && reading.nextOffset !== undefined);
  }

  function sourceReadingIsVisible() {
    const modal = $("detailModal");
    return Boolean(modal && !modal.classList.contains("hidden") && $("detailTitle")?.textContent === "소스 읽기");
  }

  Object.assign(sources, {
    readSource,
    loadMoreSourceReading,
    sourceReadingStatus,
    canLoadMoreSourceReading
  });
})(window.Plasma);

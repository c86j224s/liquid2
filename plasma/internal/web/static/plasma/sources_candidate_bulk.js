(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const missionApi = Plasma.transport.missionApi;
  const captureMissionSelection = Plasma.mission.captureMissionSelection;
  const ownsMissionSelection = Plasma.mission.ownsMissionSelection;
  const StaleMissionOperationError = Plasma.mission.StaleMissionOperationError;
  const isStaleMissionOperation = Plasma.mission.isStaleMissionOperation;
  const requireMission = () => sources.dependency("requireMission")();
  const reloadMission = (...args) => sources.dependency("reloadMission")(...args);
  const showError = (...args) => sources.dependency("showError")(...args);
  const runBulkSequential = (...args) => sources.dependency("runBulkSequential")(...args);
  const normalizeSourceURL = sources.normalizeSourceURL;
  const refreshSourceCandidates = (...args) => sources.refreshSourceCandidates(...args);
  const addURLSource = (...args) => sources.addURLSource(...args);
  const sourceCandidateTitleForURL = (...args) => sources.sourceCandidateTitleForURL(...args);
  const onDetailButtonClick = (...args) => sources.dependency("onDetailButtonClick")(...args);

  async function rejectSourceCandidate(url, reason = null) {
    if (!requireMission()) return;
    const key = normalizeSourceURL(url) || url;
    const owner = captureMissionSelection();
    if (state.sourceCandidateBusy.has(key)) return;
    const rejectionReason = reason === null
    ? window.prompt("기각 사유를 입력하세요. 비워두면 기본 사유로 기록됩니다.", "")
    : reason;
    if (rejectionReason === null) return;
    state.sourceCandidateBusy.add(key);
    refreshSourceCandidates();
    try {
      await missionApi(owner, "/candidates/sources/reject", {
        method: "POST",
        body: { url, reason: rejectionReason.trim() }
      });
      await reloadMission(owner.missionId);
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
    } finally {
      if (ownsMissionSelection(owner)) { state.sourceCandidateBusy.delete(key); refreshSourceCandidates(); }
    }
  }

  async function restoreSourceCandidate(url) {
    if (!requireMission()) return;
    const key = normalizeSourceURL(url) || url;
    const owner = captureMissionSelection();
    if (state.sourceCandidateBusy.has(key)) return;
    state.sourceCandidateBusy.add(key);
    refreshSourceCandidates();
    try {
      await missionApi(owner, "/candidates/sources/restore", {
        method: "POST",
        body: { url }
      });
      await reloadMission(owner.missionId);
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
    } finally {
      if (ownsMissionSelection(owner)) { state.sourceCandidateBusy.delete(key); refreshSourceCandidates(); }
    }
  }

  function pruneSelectedSourceCandidates(candidates) {
    const valid = new Set();
    for (const c of candidates) {
      const k = normalizeSourceURL(c.url);
      if (k) valid.add(k);
    }
    for (const url of [...state.selectedSourceCandidates]) {
      if (!valid.has(url)) state.selectedSourceCandidates.delete(url);
    }
  }

  function updateSourceCandidateBulkBar() {
    const bar = $("sourceCandidateBulk");
    if (!bar) return;
    const n = state.selectedSourceCandidates.size;
    const countEl = $("sourceCandidateBulkCount");
    if (countEl) countEl.textContent = String(n);
    bar.classList.toggle("hidden", n === 0);
  }

  function toggleSourceCandidateSelection(checkbox) {
    const url = checkbox?.dataset?.selectSourceUrl;
    if (!url) return;
    if (checkbox.checked) state.selectedSourceCandidates.add(url);
    else state.selectedSourceCandidates.delete(url);
    if (checkbox.parentElement) {
      checkbox.parentElement.classList.toggle("selected", checkbox.checked);
    }
    updateSourceCandidateBulkBar();
  }

  function selectAllSourceCandidates() {
    $("sourceCandidateList")
    .querySelectorAll("input.item-select[data-select-source-url]")
    .forEach((cb) => {
      cb.checked = true;
      state.selectedSourceCandidates.add(cb.dataset.selectSourceUrl);
      if (cb.parentElement) cb.parentElement.classList.add("selected");
    });
    updateSourceCandidateBulkBar();
  }

  function clearSourceCandidateSelection() {
    state.selectedSourceCandidates.clear();
    $("sourceCandidateList")
    .querySelectorAll("input.item-select:checked")
    .forEach((cb) => {
      cb.checked = false;
      if (cb.parentElement) cb.parentElement.classList.remove("selected");
    });
    updateSourceCandidateBulkBar();
  }

  async function bulkSourceCandidateAction(action) {
    if (!requireMission()) return;
    const owner = captureMissionSelection();
    const urls = [...state.selectedSourceCandidates];
    if (urls.length === 0) return;
    const rejectionReason = action === "reject"
    ? window.prompt("선택한 후보에 남길 기각 사유를 입력하세요. 비워두면 기본 사유로 기록됩니다.", "")
    : "";
    if (rejectionReason === null) return;
    const errors = action === "approve"
    ? await runBulkSequential(urls, async (url) => {
      if (!ownsMissionSelection(owner)) throw new StaleMissionOperationError();
      const added = await addURLSource(url, sourceCandidateTitleForURL(url), owner);
      if (!added) throw new Error(`소스 추가 실패: ${url}`);
    })
    : await runBulkSequential(urls, (url) => {
      if (!ownsMissionSelection(owner)) throw new StaleMissionOperationError();
      return missionApi(owner, "/candidates/sources/reject", {
        method: "POST",
        body: { url, reason: rejectionReason.trim() }
      });
    });
    if (!ownsMissionSelection(owner)) return;
    state.selectedSourceCandidates.clear();
    await reloadMission(owner.missionId);
    if (errors.length > 0) {
      const sample = errors.slice(0, 3).map((e) => e?.message || String(e)).join("; ");
      showError(new Error(`소스 후보 ${urls.length}개 중 ${errors.length}개 처리 실패: ${sample}`));
    }
  }

  async function onSourceCandidateListClick(event) {
    if (onDetailButtonClick(event)) return;
    const addButton = event.target.closest("[data-add-source-url]");
    if (addButton) {
      await addURLSource(addButton.dataset.addSourceUrl, addButton.dataset.sourceCandidateTitle || "");
      return;
    }
    const rejectButton = event.target.closest("[data-reject-source-url]");
    if (rejectButton) {
      await rejectSourceCandidate(rejectButton.dataset.rejectSourceUrl);
    }
  }

  async function onRejectedSourceCandidateListClick(event) {
    if (onDetailButtonClick(event)) return;
    const restoreButton = event.target.closest("[data-restore-source-url]");
    if (restoreButton) {
      await restoreSourceCandidate(restoreButton.dataset.restoreSourceUrl);
    }
  }

  Object.assign(sources, {
    rejectSourceCandidate,
    restoreSourceCandidate,
    pruneSelectedSourceCandidates,
    updateSourceCandidateBulkBar,
    toggleSourceCandidateSelection,
    selectAllSourceCandidates,
    clearSourceCandidateSelection,
    bulkSourceCandidateAction,
    onSourceCandidateListClick,
    onRejectedSourceCandidateListClick
  });
})(window.Plasma);

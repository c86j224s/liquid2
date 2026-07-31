(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const missionApi = Plasma.transport.missionApi;
  const captureMissionSelection = Plasma.mission.captureMissionSelection;
  const ownsMissionSelection = Plasma.mission.ownsMissionSelection;
  const isStaleMissionOperation = Plasma.mission.isStaleMissionOperation;
  const requireMission = () => sources.dependency("requireMission")();
  const reloadMission = (...args) => sources.dependency("reloadMission")(...args);
  const showError = (...args) => sources.dependency("showError")(...args);
  const confluenceSelectedConnectionID = (...args) => sources.confluenceSelectedConnectionID(...args);
  const selectedConfluenceSite = (...args) => sources.selectedConfluenceSite(...args);
  const confluenceSiteCloudID = (...args) => sources.confluenceSiteCloudID(...args);
  const setConfluenceFlowStatus = (...args) => sources.setConfluenceFlowStatus(...args);
  const setConfluenceBusy = (...args) => sources.setConfluenceBusy(...args);
  const showConfluenceError = (...args) => sources.showConfluenceError(...args);
  const previewConfluenceCandidate = (...args) => sources.previewConfluenceCandidate(...args);
  const renderConfluenceResults = (...args) => sources.renderConfluenceResults(...args);
  const looksLikeConfluenceURL = (...args) => sources.looksLikeConfluenceURL(...args);
  const sourceCandidateTitleForURL = (...args) => sources.sourceCandidateTitleForURL(...args);

  async function searchConfluence(event) {
    event.preventDefault();
    await searchConfluenceResults({ previewSingle: false });
  }

  async function addConfluenceURLSource(event) {
    event.preventDefault();
    if (!requireMission()) return;
    const url = $("confluencePageURL").value.trim();
    const connectionID = confluenceSelectedConnectionID();
    const site = selectedConfluenceSite();
    const cloudID = confluenceSiteCloudID(site);
    if (!url) {
      showError(new Error("Confluence 페이지 URL이 필요합니다."));
      return;
    }
    if (!connectionID || !cloudID) {
      showError(new Error("Confluence 연결과 사이트를 선택한 뒤 URL을 추가하세요."));
      return;
    }
    if (!looksLikeConfluenceURL(url)) {
      showError(new Error("Confluence 페이지 URL만 이 영역에서 추가할 수 있습니다."));
      return;
    }
    const owner = captureMissionSelection();
    setConfluenceBusy(true);
    try {
      const title = sourceCandidateTitleForURL(url);
      await missionApi(owner, "/sources/confluence/url", {
        method: "POST",
        body: {
          url,
          title,
          connection_id: connectionID,
          cloud_id: cloudID
        }
      });
      if (!ownsMissionSelection(owner)) return;
      $("confluencePageURL").value = "";
      setConfluenceFlowStatus("Confluence URL을 소스로 추가했습니다.");
      await reloadMission(owner.missionId);
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showConfluenceError(err);
    } finally {
      if (ownsMissionSelection(owner)) setConfluenceBusy(false);
    }
  }

  async function searchConfluenceResults({ previewSingle = false } = {}) {
    if (!requireMission()) return;
    const owner = captureMissionSelection();
    const connectionID = confluenceSelectedConnectionID();
    const site = selectedConfluenceSite();
    const cloudID = confluenceSiteCloudID(site);
    const query = $("confluenceQuery").value.trim();
    if (!connectionID || !cloudID || !query) {
      showError(new Error("Confluence 연결, 사이트, 검색어가 필요합니다."));
      return;
    }
    setConfluenceBusy(true);
    try {
      const result = await missionApi(owner, "/sources/confluence/search", {
        method: "POST",
        body: {
          connection_id: connectionID,
          cloud_id: cloudID,
          query,
          space_key: $("confluenceSpaceKey").value.trim(),
          limit: Number($("confluenceLimit").value || 10)
        }
      });
      if (!ownsMissionSelection(owner)) return;
      state.confluenceSearchResults = result.Candidates || result.candidates || [];
      state.confluenceSearchContext = { connection_id: connectionID, cloud_id: cloudID };
      renderConfluenceResults(state.confluenceSearchResults);
      setConfluenceFlowStatus(state.confluenceSearchResults.length ? `검색 결과 ${state.confluenceSearchResults.length}개를 찾았습니다. 후보를 검토한 뒤 소스로 승인하세요.` : "검색 결과가 없습니다. 검색어를 바꿔 다시 시도하세요.", state.confluenceSearchResults.length ? "" : "warn");
    } catch (err) {
      if (ownsMissionSelection(owner)) showConfluenceError(err);
    } finally {
      if (ownsMissionSelection(owner)) setConfluenceBusy(false);
    }
    if (ownsMissionSelection(owner) && previewSingle && state.confluenceSearchResults.length === 1) {
      await previewConfluenceCandidate(0);
    }
  }

  function clearConfluenceSearchResults() {
    state.confluenceSearchResults = [];
    state.confluenceSearchContext = null;
    renderConfluenceResults([]);
  }

  Object.assign(sources, {
    searchConfluence,
    addConfluenceURLSource,
    searchConfluenceResults,
    clearConfluenceSearchResults
  });
})(window.Plasma);

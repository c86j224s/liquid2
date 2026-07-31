(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const requireMission = () => sources.dependency("requireMission")();
  const openConfluenceSourceDetails = (...args) => sources.openConfluenceSourceDetails(...args);
  const loadConfluenceConnections = (...args) => sources.loadConfluenceConnections(...args);
  const selectedConfluenceConnection = (...args) => sources.selectedConfluenceConnection(...args);
  const selectedConfluenceSite = (...args) => sources.selectedConfluenceSite(...args);
  const setConfluenceFlowStatus = (...args) => sources.setConfluenceFlowStatus(...args);
  const openSettingsTab = (...args) => Plasma.settings.openSettingsTab(...args);
  const confluenceSelectedConnectionID = (...args) => sources.confluenceSelectedConnectionID(...args);
  const confluenceSiteCloudID = (...args) => sources.confluenceSiteCloudID(...args);
  const loadConfluenceSpaces = (...args) => sources.loadConfluenceSpaces(...args);
  const searchConfluenceResults = (...args) => sources.searchConfluenceResults(...args);
  const loadConfluenceSpacePages = (...args) => sources.loadConfluenceSpacePages(...args);
  const previewConfluencePage = (...args) => sources.previewConfluencePage(...args);

  async function runConfluenceOneClickFlow({ fromOAuth = false } = {}) {
    if (!requireMission() || state.confluenceBusy) return;
    openConfluenceSourceDetails();
    if (!state.confluenceConnections.length) {
      await loadConfluenceConnections();
    }
    let connection = selectedConfluenceConnection();
    if (!connection) {
      setConfluenceFlowStatus("저장된 연결이 없습니다. 설정에서 Confluence 연결을 만든 뒤 다시 시도하세요.", "warn");
      openSettingsTab();
      return;
    }
    let site = selectedConfluenceSite();
    if (!site) {
      setConfluenceFlowStatus("선택할 Confluence 사이트가 없습니다. 설정에서 사이트를 새로고침하거나 연결 권한을 확인하세요.", "warn");
      return;
    }
    const query = $("confluenceQuery")?.value.trim() || "";
    if (query) {
      setConfluenceFlowStatus("검색하고 후보를 검토할 준비를 하고 있습니다.");
      await searchConfluenceResults({ previewSingle: true });
      return;
    }
    const currentContext = state.confluenceBrowseContext || {};
    const sameSite = currentContext.connection_id === confluenceSelectedConnectionID() && currentContext.cloud_id === confluenceSiteCloudID(site);
    if (!sameSite || !state.confluenceSpaces.length) {
      setConfluenceFlowStatus(fromOAuth ? "연결이 완료되어 공간 목록을 불러오고 있습니다." : "공간 목록을 불러오고 있습니다.");
      await loadConfluenceSpaces();
    }
    if (state.confluenceSpaces.length === 1 && !state.confluencePages.length) {
      const space = state.confluenceSpaces[0];
      await loadConfluenceSpacePages(space.space_id || space.SpaceID || "", space.name || space.Name || "");
    }
    if (state.confluencePages.length === 1) {
      setConfluenceFlowStatus("페이지가 하나라 후보 검토 화면까지 열었습니다. 내용을 확인한 뒤 소스로 승인하세요.");
      await previewConfluencePage(state.confluencePages[0]);
      return;
    }
    if (state.confluencePages.length > 1) {
      setConfluenceFlowStatus(`페이지 ${state.confluencePages.length}개를 찾았습니다. 필요한 페이지를 후보 검토하세요.`);
      return;
    }
    if (state.confluenceSpaces.length > 1) {
      setConfluenceFlowStatus(`공간 ${state.confluenceSpaces.length}개를 찾았습니다. 공간을 선택하면 페이지 목록으로 이어집니다.`);
      return;
    }
    setConfluenceFlowStatus("탐색 가능한 공간이나 페이지를 찾지 못했습니다.", "warn");
  }

  Object.assign(sources, {
    runConfluenceOneClickFlow
  });
})(window.Plasma);

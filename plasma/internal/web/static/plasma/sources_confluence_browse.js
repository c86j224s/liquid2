(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const missionApi = Plasma.transport.missionApi;
  const captureMissionSelection = Plasma.mission.captureMissionSelection;
  const ownsMissionSelection = Plasma.mission.ownsMissionSelection;
  const requireMission = () => sources.dependency("requireMission")();
  const showError = (...args) => sources.dependency("showError")(...args);
  const confluenceSelectedConnectionID = (...args) => sources.confluenceSelectedConnectionID(...args);
  const selectedConfluenceSite = (...args) => sources.selectedConfluenceSite(...args);
  const confluenceSiteCloudID = (...args) => sources.confluenceSiteCloudID(...args);
  const setConfluenceBusy = (...args) => sources.setConfluenceBusy(...args);
  const renderConfluenceSpaces = (...args) => sources.renderConfluenceSpaces(...args);
  const renderConfluencePages = (...args) => sources.renderConfluencePages(...args);
  const showConfluenceError = (...args) => sources.showConfluenceError(...args);

  async function loadConfluenceSpaces(cursor = "") {
    if (!requireMission()) return;
    const owner = captureMissionSelection();
    const connectionID = confluenceSelectedConnectionID();
    const site = selectedConfluenceSite();
    const cloudID = confluenceSiteCloudID(site);
    if (!connectionID || !cloudID) {
      showError(new Error("공간을 탐색하려면 Confluence 연결과 사이트가 필요합니다."));
      return;
    }
    setConfluenceBusy(true);
    try {
      const result = await missionApi(owner, "/sources/confluence/spaces", {
        method: "POST",
        body: { connection_id: connectionID, cloud_id: cloudID, limit: 20, cursor }
      });
      if (!ownsMissionSelection(owner)) return;
      const spaces = result.Spaces || result.spaces || [];
      state.confluenceSpaces = cursor ? state.confluenceSpaces.concat(spaces) : spaces;
      state.confluenceBrowseContext = {
        connection_id: connectionID,
        cloud_id: cloudID,
        spaces_cursor: result.NextCursor || result.next_cursor || ""
      };
      renderConfluenceSpaces(state.confluenceSpaces);
      renderConfluencePages([]);
    } catch (err) {
      if (ownsMissionSelection(owner)) showConfluenceError(err);
    } finally {
      if (ownsMissionSelection(owner)) setConfluenceBusy(false);
    }
  }

  async function loadConfluenceSpacePages(spaceID, spaceName = "", cursor = "") {
    const owner = captureMissionSelection();
    const context = state.confluenceBrowseContext || {};
    const connectionID = context.connection_id || confluenceSelectedConnectionID();
    const cloudID = context.cloud_id || confluenceSiteCloudID(selectedConfluenceSite());
    if (!connectionID || !cloudID || !spaceID) return;
    setConfluenceBusy(true);
    try {
      const result = await missionApi(owner, "/sources/confluence/space-pages", {
        method: "POST",
        body: { connection_id: connectionID, cloud_id: cloudID, space_id: spaceID, limit: 20, cursor }
      });
      if (!ownsMissionSelection(owner)) return;
      const pages = result.Pages || result.pages || [];
      state.confluencePages = cursor ? state.confluencePages.concat(pages) : pages;
      state.confluenceBrowseContext = {
        connection_id: connectionID,
        cloud_id: cloudID,
        space_id: spaceID,
        space_name: spaceName,
        spaces_cursor: context.spaces_cursor || "",
        pages_cursor: result.NextCursor || result.next_cursor || ""
      };
      renderConfluencePages(state.confluencePages);
    } catch (err) {
      if (ownsMissionSelection(owner)) showConfluenceError(err);
    } finally {
      if (ownsMissionSelection(owner)) setConfluenceBusy(false);
    }
  }

  async function loadConfluencePageChildren(pageID, title = "", cursor = "") {
    const owner = captureMissionSelection();
    const context = state.confluenceBrowseContext || {};
    const connectionID = context.connection_id || confluenceSelectedConnectionID();
    const cloudID = context.cloud_id || confluenceSiteCloudID(selectedConfluenceSite());
    if (!connectionID || !cloudID || !pageID) return;
    setConfluenceBusy(true);
    try {
      const result = await missionApi(owner, "/sources/confluence/children", {
        method: "POST",
        body: { connection_id: connectionID, cloud_id: cloudID, page_id: pageID, limit: 20, cursor }
      });
      if (!ownsMissionSelection(owner)) return;
      const pages = result.Pages || result.pages || [];
      state.confluencePages = cursor ? state.confluencePages.concat(pages) : pages;
      state.confluenceBrowseContext = { ...context, parent_page_id: pageID, parent_title: title, pages_cursor: result.NextCursor || result.next_cursor || "" };
      renderConfluencePages(state.confluencePages);
    } catch (err) {
      if (ownsMissionSelection(owner)) showConfluenceError(err);
    } finally {
      if (ownsMissionSelection(owner)) setConfluenceBusy(false);
    }
  }

  function loadMoreConfluenceSpaces() {
    const cursor = state.confluenceBrowseContext?.spaces_cursor || "";
    if (!cursor) return;
    loadConfluenceSpaces(cursor);
  }

  function loadMoreConfluencePages() {
    const context = state.confluenceBrowseContext || {};
    const cursor = context.pages_cursor || "";
    if (!cursor) return;
    if (context.parent_page_id) {
      loadConfluencePageChildren(context.parent_page_id, context.parent_title || "", cursor);
      return;
    }
    if (context.space_id) {
      loadConfluenceSpacePages(context.space_id, context.space_name || "", cursor);
    }
  }

  Object.assign(sources, {
    loadConfluenceSpaces,
    loadConfluenceSpacePages,
    loadConfluencePageChildren,
    loadMoreConfluenceSpaces,
    loadMoreConfluencePages
  });
})(window.Plasma);

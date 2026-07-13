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

function renderConfluenceSpaces(spaces) {
  const container = $("confluenceSpaces");
  if (!container) return;
  const context = state.confluenceBrowseContext || {};
  $("confluenceBrowseBreadcrumb").textContent = spaces.length ? "공간을 선택하면 페이지 목록을 불러옵니다." : "사이트를 선택하고 공간을 불러오세요.";
  container.innerHTML = spaces.length ? spaces.map((space) => {
    const id = space.space_id || space.SpaceID || "";
    const key = space.space_key || space.SpaceKey || "";
    const name = space.name || space.Name || id;
    return `
      <div class="item">
        <div class="item-title">${escapeHTML(name)} ${key ? `<span class="badge muted">${escapeHTML(key)}</span>` : ""}</div>
        <div class="item-meta">${escapeHTML(id)}</div>
        <div class="item-actions">
          <button type="button" data-confluence-space-id="${escapeAttr(id)}" data-confluence-space-name="${escapeAttr(name)}">페이지 보기</button>
        </div>
      </div>
    `;
  }).join("") : empty("Confluence 공간 없음");
  $("confluenceLoadMoreSpaces").classList.toggle("hidden", !context.spaces_cursor);
  $("confluenceLoadMoreSpaces").disabled = !context.spaces_cursor || state.confluenceBusy;
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

function renderConfluencePages(pages) {
  const container = $("confluencePages");
  if (!container) return;
  const context = state.confluenceBrowseContext || {};
  const trail = [context.space_name, context.parent_title].filter(Boolean).join(" / ");
  if (trail) $("confluenceBrowseBreadcrumb").textContent = trail;
  container.innerHTML = pages.length ? pages.map((page, index) => {
    const pageID = page.page_id || page.PageID || "";
    const title = page.title || page.Title || pageID;
    const version = page.version || page.Version || 0;
    const webURL = page.web_url || page.WebURL || "";
    return `
      <div class="item">
        <div class="item-title">${escapeHTML(title)} <span class="badge muted">v${escapeHTML(version || "?")}</span></div>
        <div class="item-meta">${escapeHTML(pageID)}</div>
        <div class="item-actions">
          ${webURL ? `<a class="button-link secondary" href="${escapeAttr(webURL)}" target="_blank" rel="noopener noreferrer">원문 열기</a>` : ""}
          <button type="button" class="secondary" data-confluence-children-page-id="${escapeAttr(pageID)}" data-confluence-page-title="${escapeAttr(title)}">하위 페이지</button>
          <button type="button" data-confluence-page-index="${escapeAttr(index)}">후보 검토</button>
        </div>
      </div>
    `;
  }).join("") : empty("Confluence 페이지 없음");
  $("confluenceLoadMorePages").classList.toggle("hidden", !context.pages_cursor);
  $("confluenceLoadMorePages").disabled = !context.pages_cursor || state.confluenceBusy;
}

function confluenceCandidatePageID(candidate) {
  const connector = candidate?.Connector || candidate?.connector || {};
  const direct = candidate?.PageID || candidate?.page_id || connector.PageID || connector.page_id || "";
  if (direct) return String(direct);
  const externalID = connector.ExternalSourceID || connector.external_source_id || "";
  const parts = String(externalID).split(":");
  return parts.length >= 2 ? parts.slice(1).join(":") : externalID;
}

function onConfluenceSpacesClick(event) {
  const button = event.target.closest("[data-confluence-space-id]");
  if (!button) return;
  loadConfluenceSpacePages(button.dataset.confluenceSpaceId, button.dataset.confluenceSpaceName || "");
}

function onConfluencePagesClick(event) {
  const children = event.target.closest("[data-confluence-children-page-id]");
  if (children) {
    loadConfluencePageChildren(children.dataset.confluenceChildrenPageId, children.dataset.confluencePageTitle || "");
    return;
  }
  const review = event.target.closest("[data-confluence-page-index]");
  if (review) {
    previewConfluencePage(state.confluencePages[Number(review.dataset.confluencePageIndex)]);
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

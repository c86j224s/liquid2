(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const escapeAttr = Plasma.dom.escapeAttr;
  const empty = Plasma.ui.empty;
  const loadConfluenceSpacePages = (...args) => sources.loadConfluenceSpacePages(...args);
  const loadConfluencePageChildren = (...args) => sources.loadConfluencePageChildren(...args);
  const previewConfluencePage = (...args) => sources.previewConfluencePage(...args);

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

  Object.assign(sources, {
    renderConfluenceSpaces,
    renderConfluencePages,
    confluenceCandidatePageID,
    onConfluenceSpacesClick,
    onConfluencePagesClick
  });
})(window.Plasma);

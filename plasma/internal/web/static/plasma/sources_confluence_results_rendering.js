(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const escapeAttr = Plasma.dom.escapeAttr;
  const timeShort = Plasma.dom.timeShort;
  const empty = Plasma.ui.empty;
  const confluenceCandidatePageID = (...args) => sources.confluenceCandidatePageID(...args);
  const confluenceDisplayableExternalURI = (...args) => sources.confluenceDisplayableExternalURI(...args);
  const confluenceExternalURIHost = (...args) => sources.confluenceExternalURIHost(...args);

  function confluenceCandidateDetailPayload(candidate) {
    const title = candidate.Title || candidate.title || confluenceCandidatePageID(candidate) || "Confluence 페이지";
    const sourceURI = confluenceDisplayableExternalURI(candidate.SourceURI || candidate.source_uri || "");
    const siteURL = confluenceDisplayableExternalURI(candidate.SiteURL || candidate.site_url || "");
    const pageID = confluenceCandidatePageID(candidate);
    const detail = {
      type: "confluence_candidate",
      title,
      site_url: siteURL,
      site_host: confluenceExternalURIHost(siteURL || sourceURI),
      page_id: pageID,
      space_key: candidate.SpaceKey || candidate.space_key || "",
      version: candidate.Version || candidate.version || "",
      updated_at: candidate.UpdatedAt || candidate.updated_at || "",
      can_snapshot: Boolean(candidate.CanSnapshot ?? candidate.can_snapshot)
    };
    if (sourceURI) detail.source_uri = sourceURI;
    return detail;
  }

  function renderConfluenceResults(candidates) {
    const container = $("confluenceResults");
    if (!container) return;
    container.innerHTML = candidates.length ? candidates.map((candidate, index) => {
      const title = candidate.Title || candidate.title || confluenceCandidatePageID(candidate) || "Confluence 페이지";
      const sourceURI = confluenceDisplayableExternalURI(candidate.SourceURI || candidate.source_uri || "");
      const space = candidate.SpaceKey || candidate.space_key || "";
      const version = candidate.Version || candidate.version || 0;
      const updated = candidate.UpdatedAt || candidate.updated_at || "";
      const detailPayload = confluenceCandidateDetailPayload(candidate);
      return `
      <div class="item">
        <div class="item-title">${escapeHTML(title)} <span class="badge muted">v${escapeHTML(version || "?")}</span></div>
        <div class="item-meta">${space ? `공간 ${escapeHTML(space)} / ` : ""}${escapeHTML(sourceURI || confluenceCandidatePageID(candidate))}</div>
        ${updated ? `<div class="item-meta">수정: ${escapeHTML(timeShort(updated))}</div>` : ""}
        <div class="item-actions">
          ${sourceURI ? `<a class="button-link secondary" href="${escapeAttr(sourceURI)}" target="_blank" rel="noopener noreferrer">원문 열기</a>` : ""}
          <button type="button" class="secondary" data-detail-title="Confluence 검색 결과" data-detail-json="${escapeAttr(JSON.stringify(detailPayload))}">자세히</button>
          <button type="button" data-confluence-candidate-index="${escapeAttr(index)}">후보 검토</button>
        </div>
      </div>
    `;
    }).join("") : empty("Confluence 검색 결과 없음");
  }

  function onConfluenceResultsClick(event) {
    if (sources.dependency("onDetailButtonClick")(event)) return;
    const button = event.target.closest("[data-confluence-candidate-index]");
    if (button) sources.previewConfluenceCandidate(button.dataset.confluenceCandidateIndex);
  }

  Object.assign(sources, {
    confluenceCandidateDetailPayload,
    renderConfluenceResults,
    onConfluenceResultsClick
  });
})(window.Plasma);

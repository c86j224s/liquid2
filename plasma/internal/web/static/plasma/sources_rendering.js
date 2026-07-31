(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const escapeAttr = Plasma.dom.escapeAttr;
  const empty = Plasma.ui.empty;
  const updateCountChip = Plasma.ui.updateCountChip;
  const setSectionEmpty = Plasma.ui.setSectionEmpty;
  const localPathLocator = (...args) => sources.localPathLocator(...args);
  const mediaLocator = (...args) => sources.mediaLocator(...args);
  const pdfLocator = (...args) => sources.pdfLocator(...args);
  const documentLocator = (...args) => sources.documentLocator(...args);
  const confluenceSourceInfo = (...args) => sources.confluenceSourceInfo(...args);
  const confluenceUpdateState = (...args) => sources.confluenceUpdateState(...args);
  const confluenceUpdateText = (...args) => sources.confluenceUpdateText(...args);
  const mediaSourceLabel = (...args) => sources.mediaSourceLabel(...args);
  const documentSourceText = (...args) => sources.documentSourceText(...args);
  const pdfSourceText = (...args) => sources.pdfSourceText(...args);
  const mediaSourceText = (...args) => sources.mediaSourceText(...args);
  const confluenceSourceText = (...args) => sources.confluenceSourceText(...args);
  const sourceDetailPayload = (...args) => sources.sourceDetailPayload(...args);

  function renderSources(sources) {
    const n = sources.length;
    updateCountChip("sourceListCount", n);
    const savedHeader = document.querySelector(".saved-source-header");
    const savedDivider = document.querySelector(".saved-source-divider");
    setSectionEmpty(savedHeader, n === 0);
    setSectionEmpty(savedDivider, n === 0);
    $("sourceList").innerHTML = n ? sources.map((source) => {
      const snapshotID = source.SnapshotID || source.snapshot_id || "";
      const connector = source.Connector || source.connector || {};
      const access = source.Access || source.access || {};
      const sourceState = source.State || source.state || {};
      const removed = Boolean(sourceState.removed || sourceState.Removed || sourceState.state === "removed" || sourceState.State === "removed");
      const retrievalPolicy = source.Access?.RetrievalPolicy || access.retrieval_policy || source.retrieval_policy || "snapshot_only";
      const modeLabel = retrievalPolicy === "live_reference" ? "라이브 참조" : "스냅샷";
      const locator = localPathLocator(source);
      const media = mediaLocator(source);
      const pdf = pdfLocator(source);
      const document = documentLocator(source);
      const confluence = confluenceSourceInfo(source);
      const confluenceUpdate = confluence ? confluenceUpdateState(source) : null;
      const confluenceUpdateLabel = confluenceUpdateText(confluenceUpdate);
      const mediaLabel = media ? mediaSourceLabel(media) : "";
      const pdfLabel = pdf ? "PDF" : "";
      const documentLabel = document && !pdf && !media ? "문서" : "";
      const confluenceLabel = confluence ? "Confluence" : "";
      let locatorText = connector.ExternalURI || connector.ExternalSourceID || "";
      if (locator) locatorText = `${locator.root_id}/${locator.relative_path || "."}`;
      if (document && !pdf && !media) locatorText = documentSourceText(document);
      if (pdf) locatorText = pdfSourceText(pdf);
      if (media) locatorText = mediaSourceText(media);
      if (confluence) locatorText = confluenceSourceText(confluence);
      const detailPayload = sourceDetailPayload(source, confluence);
      return `
      <div class="item ${removed ? "source-removed" : ""}">
        <div class="item-title">
          ${escapeHTML(source.Title || source.title || snapshotID)}
          <span class="badge muted">${escapeHTML(modeLabel)}</span>
          ${mediaLabel ? `<span class="badge">${escapeHTML(mediaLabel)}</span>` : ""}
          ${pdfLabel ? `<span class="badge">${escapeHTML(pdfLabel)}</span>` : ""}
          ${documentLabel ? `<span class="badge">${escapeHTML(documentLabel)}</span>` : ""}
          ${confluenceLabel ? `<span class="badge">${escapeHTML(confluenceLabel)}</span>` : ""}
          ${confluence?.version ? `<span class="badge muted">v${escapeHTML(confluence.version)}</span>` : ""}
          ${removed ? `<span class="badge warn">제거됨</span>` : ""}
        </div>
        <div class="item-meta">${escapeHTML(snapshotID)} / ${escapeHTML(connector.ConnectorID || connector.connector_id || "source")}</div>
        <div class="item-meta">${escapeHTML(locatorText)}</div>
        ${confluenceUpdateLabel ? `<div class="item-meta">${escapeHTML(confluenceUpdateLabel)}</div>` : ""}
        <div class="item-actions">
          <button type="button" class="secondary" data-detail-title="소스 상세" data-detail-json="${escapeAttr(JSON.stringify(detailPayload))}">자세히</button>
          ${removed ? `
      <button type="button" data-source-restore="${escapeAttr(snapshotID)}">복원</button>
      ` : `
      <button type="button" class="secondary" data-source-read="${escapeAttr(snapshotID)}">읽기</button>
      ${confluence ? `<button type="button" class="secondary" data-confluence-source-update="${escapeAttr(snapshotID)}">업데이트 확인</button>` : ""}
      <button type="button" class="danger" data-source-remove="${escapeAttr(snapshotID)}">제거</button>
      `}
        </div>
      </div>
    `;
    }).join("") : empty("저장된 소스 없음");
    setSectionEmpty($("sourceList"), n === 0);
  }

  Object.assign(sources, {
    renderSources
  });
})(window.Plasma);

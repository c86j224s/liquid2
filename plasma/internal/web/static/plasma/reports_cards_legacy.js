(function reportsLegacyCards(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const escapeHTML = root.Plasma.dom.escapeHTML;
  const escapeAttr = root.Plasma.dom.escapeAttr;

  function renderLegacyReportSection(legacyCards, selectedKey) {
    return `<div class="list-section-label">Legacy AST report</div>${legacyCards.map(({ key, report }) => `
      <div class="item report-card ${report.isLatest ? "active" : ""} ${key === selectedKey ? "selected" : ""}" data-report-key="${escapeAttr(key)}" data-report-card-version-id="${escapeAttr(report.versionID)}">
        <div class="item-title report-title-line report-card-toggle"><span>${escapeHTML(report.title)}</span><span class="chip-row report-chip-row">${report.isLatest ? `<span class="badge session-new">최신</span>` : `<span class="badge muted">이전</span>`}${report.rigorLabel ? `<span class="badge">${escapeHTML(report.rigorLabel)}</span>` : ""}<span class="badge muted">${escapeHTML(report.stateLabel)}</span></span></div>
        <div class="report-card-body"><div class="item-meta clamp-line" title="${escapeAttr(report.versionID)}">${escapeHTML(report.createdLabel)} / ${escapeHTML(report.versionID)}</div><div class="item-meta">${escapeHTML(report.exportLabel)}</div>
          <div class="report-plan-line"><span class="badge muted">생성 계획</span><span>${escapeHTML(report.planLabel)}</span></div>
          <div class="report-trace"><div class="report-trace-head"><span class="badge muted">MCP 추적</span><span>${escapeHTML(report.traceLabel)}</span></div>${reports.renderTraceBars(report.trace)}</div>
          <div class="item-actions"><button type="button" data-report-version-id="${escapeAttr(report.versionID)}" data-action="markdown">Markdown 보기</button><button type="button" data-report-version-id="${escapeAttr(report.versionID)}" data-action="html">HTML 보기</button><button type="button" data-report-version-id="${escapeAttr(report.versionID)}" data-action="json_ast">JSON 보기</button>${reports.reportActionMenu("도구 ▾", `<button type="button" class="secondary" data-detail-title="리포트 버전 상세" data-detail-json="${escapeAttr(JSON.stringify(report.version))}">자세히</button><button type="button" class="secondary" data-report-version-id="${escapeAttr(report.versionID)}" data-action="plan">생성 계획</button><button type="button" class="secondary" data-report-version-id="${escapeAttr(report.versionID)}" data-action="mcp-trace">MCP 추적</button><button type="button" class="secondary" data-report-version-id="${escapeAttr(report.versionID)}" data-action="ast">AST 보기</button>`)}${reports.reportActionMenu("받기 ▾", `<button type="button" class="secondary" data-report-version-id="${escapeAttr(report.versionID)}" data-action="download-markdown">MD 받기</button><button type="button" class="secondary" data-report-version-id="${escapeAttr(report.versionID)}" data-action="download-html">HTML 받기</button><button type="button" class="secondary" data-report-version-id="${escapeAttr(report.versionID)}" data-action="download-json_ast">JSON 받기</button>`)}</div>${reports.reportPreviewInlineHTML(key)}</div>
      </div>`).join("")}`;
  }
  reports.renderLegacyReportSection = renderLegacyReportSection;
})(window);

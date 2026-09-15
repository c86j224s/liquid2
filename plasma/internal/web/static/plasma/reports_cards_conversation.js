(function reportsConversationCards(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const escapeHTML = root.Plasma.dom.escapeHTML;
  const escapeAttr = root.Plasma.dom.escapeAttr;

  function renderConversationExportCard(key, payload, selectedKey, isLatest = false) {
    return `<div class="item report-card ${isLatest ? "active" : ""} ${key === selectedKey ? "selected" : ""}" data-report-key="${escapeAttr(key)}">
      <div class="item-title report-title-line report-card-toggle">
        <span>${escapeHTML(payload.title || "대화내역 export")}</span>
        <span class="chip-row report-chip-row">
          ${isLatest ? `<span class="badge session-new">최신</span>` : `<span class="badge muted">이전</span>`}
          <span class="badge">대화내역</span><span class="badge muted">Markdown artifact</span>
        </span>
      </div>
      <div class="report-card-body">
        <div class="item-meta clamp-line" title="${escapeAttr(payload.artifact_id || "")}">${escapeHTML(payload.artifact_id || "")}</div>
        <div class="item-meta">${escapeHTML(payload.text || "대화내역 export artifact가 생성되었습니다.")}</div>
        <div class="report-plan-line"><span class="badge muted">포함 항목</span><span>${escapeHTML(`${payload.entry_count || 0}개 요청/응답`)}</span></div>
        <div class="item-actions">
          <button type="button" data-conversation-export-id="${escapeAttr(payload.artifact_id || "")}" data-action="view">Markdown 보기</button>
          <button type="button" class="secondary" data-conversation-export-id="${escapeAttr(payload.artifact_id || "")}" data-action="download">MD 받기</button>
          ${reports.reportActionMenu("도구 ▾", `<button type="button" class="secondary" data-detail-title="대화내역 export 상세" data-detail-json="${escapeAttr(JSON.stringify(payload))}">자세히</button>`)}
        </div>${reports.reportPreviewInlineHTML(key)}
      </div>
    </div>`;
  }
  reports.renderConversationExportCard = renderConversationExportCard;
})(window);

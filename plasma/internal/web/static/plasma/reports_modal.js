(function reportsModal(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const state = root.Plasma.state;
  const $ = root.Plasma.dom.$;
  const escapeHTML = root.Plasma.dom.escapeHTML;
  const escapeAttr = root.Plasma.dom.escapeAttr;
  const ui = root.Plasma.ui;
function openReportModal(header, kind, content, options = {}) {
  const tableOfContents = kind === "markdown" && options.tableOfContents === true;
  reports.tocController?.prepare(tableOfContents);
  $("detailTitle").textContent = header || "리포트 보기";
  state.detailText = String(content ?? "");
  let body;
  let redpenManaged = false;
  if (kind === "markdown") {
    if (options.sourceArtifactID && reports.redpenController) {
      $("detailBody").innerHTML = `<div class="report-modal-body turn-markdown"></div>`;
      reports.redpenController.open({
        sourceArtifactID: options.sourceArtifactID,
        content: String(content ?? "")
      });
      redpenManaged = true;
    } else {
      reports.redpenController?.reset();
      body = `<div class="report-modal-body turn-markdown">${reports.renderMarkdown(content)}</div>`;
    }
  } else if (kind === "html") {
    reports.redpenController?.reset();
    const previewContent = reports.prepareHTMLPreview ? reports.prepareHTMLPreview(content) : content;
    body = `<div class="report-modal-frame"><iframe class="plasma-html-preview-frame" title="HTML 리포트" sandbox="allow-scripts" srcdoc="${escapeAttr(previewContent)}"></iframe></div>`;
  } else {
    reports.redpenController?.reset();
    body = `<pre class="report-modal-pre">${escapeHTML(content)}</pre>`;
  }
  if (!redpenManaged) {
    $("detailBody").innerHTML = body;
    reports.renderPlasmaMath?.($("detailBody"));
    if (kind === "markdown") reports.renderPlasmaMermaid?.($("detailBody"));
    if (kind === "markdown") reports.enhanceImages?.($("detailBody"));
    if (tableOfContents) reports.tocController?.refresh();
  }
  ui.openDetailModal(true);
  ui.enableDetailScrollRatio();
}

function openReportModalLoading(header) {
  reports.tocController?.reset();
  reports.redpenController?.reset();
  $("detailTitle").textContent = header || "리포트 불러오는 중";
  $("detailBody").innerHTML = `<div class="report-preview-loading"><span class="spinner"></span>불러오는 중…</div>`;
  ui.openDetailModal(true);
  ui.disableDetailScrollRatio();
}

function setReportPreviewLoading(key) {
  state.selectedReportKey = key;
  reports.renderReportsFromState();
  openReportModalLoading("리포트 불러오는 중");
}

function applyReportPreview(key, kind, header, content, options = {}) {
  state.selectedReportKey = key;
  reports.renderReportsFromState();
  openReportModal(header, kind, content, options);
}

function clearReportPreview() {
  state.reportDeletePreview = null;
  ui.hideDetail();
}

function selectReport(key) {
  if (!key) return;
  // Accordion select only — the 보기 buttons open the content modal explicitly.
  state.selectedReportKey = key;
  reports.renderReportsFromState();
}


  Object.assign(reports, { openReportModal, openReportModalLoading, setReportPreviewLoading, applyReportPreview, clearReportPreview, selectReport });
})(window);

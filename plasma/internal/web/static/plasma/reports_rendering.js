(function reportsRendering(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const state = root.Plasma.state;
  const $ = root.Plasma.dom.$;
  const escapeHTML = root.Plasma.dom.escapeHTML;
  const escapeAttr = root.Plasma.dom.escapeAttr;
  const timeShort = root.Plasma.dom.timeShort;
  const empty = root.Plasma.ui.empty;
  const updateCountChip = root.Plasma.ui.updateCountChip;
  const eventByID = (...args) => reports.eventByID(...args);
  const REPORT_MODE_LABELS = reports.REPORT_MODE_LABELS;
  const REPORT_EXECUTION_STRATEGY_LABELS = reports.REPORT_EXECUTION_STRATEGY_LABELS;
  const REPORT_RIGOR_LABELS = reports.REPORT_RIGOR_LABELS;
  const reportGenerationGuidanceLabel = (...args) => reports.reportGenerationGuidanceLabel(...args);

function reportActionMenu(label, itemsHTML) {
  if (!itemsHTML || !itemsHTML.trim()) return "";
  return `<details class="report-menu"><summary>${escapeHTML(label)}</summary><div class="report-menu-items">${itemsHTML}</div></details>`;
}

function reportGenerationContext(payload = {}) {
  const pendingID = payload.pending_event_id || payload.generation?.pending_event_id || "";
  const pendingPayload = pendingID ? (eventByID(pendingID)?.Payload || {}) : {};
  return { ...pendingPayload, ...payload };
}

function reportGenerationSummary(payload = {}) {
  const context = reportGenerationContext(payload);
  const mode = context.report_mode || "planned";
  const strategy = String(context.execution_strategy || "serial").trim() || "serial";
  const model = String(context.agent_model || "").trim();
  const effort = String(context.agent_reasoning_effort || "").trim();
  const guidance = String(context.generation_guidance_profile || "g2").trim() || "g2";
  return {
    mode: context.report_mode_label || REPORT_MODE_LABELS[mode] || "보고서",
    strategy: mode === "long_form" ? (REPORT_EXECUTION_STRATEGY_LABELS[strategy] || strategy) : "",
    guidance: reportGenerationGuidanceLabel(guidance),
    rigor: context.rigor_label || REPORT_RIGOR_LABELS[context.rigor_level] || "미지정",
    model: model || "미션 설정 상속",
    effort: effort || (model ? "모델 기본값" : "미션 설정 상속"),
    direction: String(context.direction_hint || "").trim() || "지정 없음"
  };
}

function reportGenerationSummaryHTML(payload = {}) {
  const summary = reportGenerationSummary(payload);
  return `
    <div class="report-generation-summary" aria-label="리포트 생성 설정">
      <span class="report-generation-item"><strong>방식</strong><span>${escapeHTML(summary.mode)}</span></span>
      ${summary.strategy ? `<span class="report-generation-item"><strong>장문 작성</strong><span>${escapeHTML(summary.strategy)}</span></span>` : ""}
      ${summary.guidance ? `<span class="report-generation-item"><strong>글쓰기</strong><span>${escapeHTML(summary.guidance)}</span></span>` : ""}
      <span class="report-generation-item"><strong>엄격도</strong><span>${escapeHTML(summary.rigor)}</span></span>
      <span class="report-generation-item"><strong>모델</strong><span>${escapeHTML(summary.model)}</span></span>
      <span class="report-generation-item"><strong>추론</strong><span>${escapeHTML(summary.effort)}</span></span>
      <span class="report-generation-item report-direction-line"><strong>방향</strong><span>${escapeHTML(summary.direction)}</span></span>
    </div>`;
}

function reportPipelineRequestSummary(progress = {}) {
  if (!progress || !progress.attempt_id) return null;
  const event = eventByID(progress.attempt_id);
  const payload = event?.Payload || {};
  const summary = reportGenerationSummary(payload);
  const startedAt = String(payload.started_at || "").trim();
  return {
    ...summary,
    startedAt: startedAt ? timeShort(startedAt) : "생성 시작 시각 알 수 없음",
    startedAtDateTime: startedAt && !Number.isNaN(new Date(startedAt).getTime()) ? startedAt : ""
  };
}

function shouldHideDraftPendingNotice(status) {
  if (status.state !== "pending" || status.event?.EventType !== "report.draft.pending") return false;
  const progress = state.detail?.report_progress;
  if (!progress || progress.state === "unknown") return false;
  return String(progress.attempt_id || "") === String(status.event?.EventID || "");
}

function reportSourceContext(payload = {}) {
  return reportGenerationContext(payload).source_context || null;
}

function reportSourceContextHTML(payload = {}) {
  const context = reportSourceContext(payload);
  if (!context || !Array.isArray(context.confluence_sources)) return "";
  const capturedAt = timeShort(context.captured_at || "");
  const sources = context.confluence_sources;
  const rows = sources.length
    ? sources.map((source) => {
        const title = String(source.title || "Confluence source").trim();
        const version = String(source.snapshot_version || "").trim();
        const snapshotAt = timeShort(source.snapshot_captured_at || "");
        const updatedAt = timeShort(source.external_updated_at || "");
        const metadata = [
          version ? `저장 v${version}` : "저장 버전 미상",
          snapshotAt ? `snapshot ${snapshotAt}` : "",
          updatedAt ? `원본 수정 ${updatedAt}` : ""
        ].filter(Boolean).join(" · ");
        return `<div class="report-plan-line"><span>${escapeHTML(title)}</span><span>${escapeHTML(metadata)} · ${escapeHTML(reportSourceCheckText(source.last_check || {}))}</span></div>`;
      }).join("")
    : `<div class="report-plan-line"><span>생성 시점에 사용 가능한 Confluence 소스가 없었습니다.</span></div>`;
  return `
    <div class="report-trace report-source-context" aria-label="생성 시점의 Confluence 소스 정보">
      <div class="report-trace-head">
        <span class="badge muted">생성 시점의 소스 정보</span>
        <span>${escapeHTML(capturedAt ? `${capturedAt} 기준` : "캡처 시각 기록됨")}</span>
      </div>
      ${rows}
    </div>`;
}

function reportSourceCheckText(check = {}) {
  const status = String(check.status || "not_checked").trim();
  const checkedAt = timeShort(check.checked_at || "");
  const latestVersion = check.latest_version || 0;
  let result = "원본 확인 기록 없음";
  if (status === "current") {
    result = `마지막 확인 당시 v${latestVersion || "?"} 최신`;
  } else if (status === "update_available") {
    result = `마지막 확인 당시 v${latestVersion || "?"} 사용 가능`;
  } else if (status === "check_failed") {
    result = window.Plasma.sources.confluenceUpdateFailureText(check.error_category || "");
  }
  return checkedAt ? `${checkedAt} · ${result}` : result;
}


function renderReports(versions) {
  reports.pipeline.render(state.detail?.report_progress, reportPipelineRequestSummary(state.detail?.report_progress));
  const conversationExports = reports.conversationExportPayloads();
  const artifactReports = reports.reportArtifactPayloads();
  const legacyReports = versions.map((version, index) => reports.reportViewModel(version, index));
  const total = conversationExports.length + artifactReports.length + legacyReports.length;
  updateCountChip("reportListCount", total);
  updateCountChip("reportTabCount", total);
  const conversationCards = conversationExports.map((payload, index) => ({ key: `conversation:${payload.artifact_id || `idx${index}`}`, isLatest: index === 0, payload }));
  const artifactCards = artifactReports.map((payload, index) => ({ key: `artifact:${payload.artifact_id || `idx${index}`}`, isLatest: index === 0, payload }));
  const legacyCards = legacyReports.map((report) => ({ key: `version:${report.versionID}`, report }));
  const allKeys = [...conversationCards.map((c) => c.key), ...artifactCards.map((c) => c.key), ...legacyCards.map((c) => c.key)];
  if (!state.selectedReportKey || !allKeys.includes(state.selectedReportKey)) state.selectedReportKey = allKeys[0] || "";
  if (state.reportPreview && !allKeys.includes(state.reportPreview.key)) state.reportPreview = null;
  const sections = [reports.renderConversationExportSection(conversationCards, state.selectedReportKey)];
  if (artifactCards.length) sections.push(reports.renderArtifactReportSection(artifactCards, state.selectedReportKey));
  if (legacyCards.length) sections.push(reports.renderLegacyReportSection(legacyCards, state.selectedReportKey));
  $("reportList").innerHTML = sections.length ? sections.join("") : empty("리포트 artifact 없음");
}

function renderReportsFromState() { renderReports(state.detail?.report_versions || []); }

function reportPreviewInlineHTML(key) {
  if (key === state.selectedReportKey) return `<div class="report-card-preview report-card-preview-hint">‘Markdown 보기’ 등을 누르면 전체 화면 팝업으로 열립니다.</div>`;
  return "";
}

Object.assign(reports, { reportActionMenu, reportGenerationContext, reportGenerationSummary, reportGenerationSummaryHTML, reportPipelineRequestSummary, shouldHideDraftPendingNotice, reportSourceContext, reportSourceContextHTML, reportSourceCheckText, renderReports, renderReportsFromState, reportPreviewInlineHTML });
})(window);

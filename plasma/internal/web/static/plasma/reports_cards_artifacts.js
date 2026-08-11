(function reportsArtifactCards(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const state = root.Plasma.state;
  const escapeHTML = root.Plasma.dom.escapeHTML;
  const escapeAttr = root.Plasma.dom.escapeAttr;

  function humanizedArtifacts(payload) {
    // Manual H5 creation is deprecated; retain historical status and artifact access.
    const humanized = reports.reportArtifactHumanizedExportState(payload.artifact_id || "");
    const redpen = humanized.state === "completed" ? reports.reportArtifactRedpenState(humanized.payload.artifact_id || "") : { state: "idle", payload: {} };
    const label = humanized.state === "completed" ? "생성 완료" : humanized.state === "failed" ? "생성 실패" : humanized.state === "pending" ? "생성 중" : humanized.state === "skipped" ? "변경 없음" : "생성 없음";
    const actions = humanized.state === "completed" ? `
      <button type="button" data-report-artifact-id="${escapeAttr(humanized.payload.artifact_id || "")}" data-action="view-artifact">보정 Markdown 보기</button>
      <button type="button" class="secondary" data-report-artifact-id="${escapeAttr(humanized.payload.artifact_id || "")}" data-action="download-artifact">보정 MD 받기</button>
      ${redpen.state === "completed" ? `<button type="button" data-report-artifact-id="${escapeAttr(humanized.payload.artifact_id || "")}" data-action="view-redpen-artifact">보정본 빨간펜 보기</button>` : ""}`
      : humanized.state === "pending" ? `<button type="button" class="secondary" disabled>말투 보정 중</button>`
      : "";
    const failureLine = humanized.state === "failed" ? `<div class="report-plan-line report-error-line"><span class="badge warn">실패 사유</span><span>${escapeHTML(humanized.payload.error || humanized.payload.text || "실패 사유 없음")}</span></div>` : "";
    const redpenLine = redpen.state === "completed" ? `<div class="report-plan-line"><span class="badge muted">보정본 빨간펜</span><span>${escapeHTML(`작업본 v${redpen.payload.revision || "?"} 저장됨`)}</span></div>` : "";
    const redpenDownload = redpen.state === "completed" ? `<button type="button" class="secondary" data-report-artifact-id="${escapeAttr(humanized.payload.artifact_id || "")}" data-action="download-redpen-artifact">보정본 빨간펜 받기</button>` : "";
    return { humanized, redpen, label, actions, failureLine, redpenLine, redpenDownload };
  }

  function designedArtifacts(payload) {
    const designed = reports.reportArtifactDesignedExportState(payload.artifact_id || "");
    const label = designed.state === "completed" ? "생성 완료" : designed.state === "pending" ? "생성 중" : designed.state === "failed" ? "생성 실패" : "아직 없음";
    const actions = designed.state === "completed" ? `
      <button type="button" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="view-designed-html-artifact">디자인 HTML 보기</button>
      <button type="button" class="secondary" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="download-designed-html-artifact">디자인 HTML 받기</button>`
      : designed.state === "pending" ? `<button type="button" class="secondary" disabled>디자인 HTML 생성 중</button>`
      : `<button type="button" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="start-designed-html-artifact">${designed.state === "failed" ? "디자인 HTML 다시 생성" : "디자인 HTML 생성"}</button>`;
    return { designed, label, actions };
  }

  function renderArtifactCard(key, isLatest, payload, selectedKey) {
    const plan = reports.reportArtifactPlanPayload(payload);
    const planData = plan.plan || {};
    const modeLabel = payload.report_mode_label || reports.REPORT_MODE_LABELS[payload.report_mode] || "보고서";
    const planLabel = reports.reportPlanLabel(planData) || (payload.report_mode === "one_take" ? "원테이크 생성: 별도 계획 없음" : "기록된 생성 계획 없음");
    const planButton = plan.event_id ? `<button type="button" class="secondary" data-report-plan-event-id="${escapeAttr(plan.event_id)}" data-action="plan">생성 계획</button>` : "";
    const trace = reports.mcpTraceSummary(payload.tool_session_id || payload.plan_tool_session_id || "");
    const designed = designedArtifacts(payload);
    const humanized = humanizedArtifacts(payload);
    const redpen = reports.reportArtifactRedpenState(payload.artifact_id || "");
    const redpenLabel = redpen.state === "completed" ? `작업본 v${redpen.payload.revision || "?"} 저장됨` : "아직 없음";
    const redpenView = redpen.state === "completed" ? `<button type="button" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="view-redpen-artifact">빨간펜 작업본 보기</button>` : "";
    const redpenDownload = redpen.state === "completed" ? `<button type="button" class="secondary" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="download-redpen-artifact">빨간펜 MD 받기</button>` : "";
    return `<div class="item report-card ${isLatest ? "active" : ""} ${key === selectedKey ? "selected" : ""}" data-report-key="${escapeAttr(key)}">
      <div class="item-title report-title-line report-card-toggle"><span>${escapeHTML(payload.title || "Markdown report")}</span><span class="chip-row report-chip-row">${isLatest ? `<span class="badge session-new">최신</span>` : `<span class="badge muted">이전</span>`}<span class="badge">${escapeHTML(modeLabel)}</span><span class="badge muted">Markdown artifact</span></span></div>
      <div class="report-card-body"><div class="item-meta clamp-line" title="${escapeAttr(payload.artifact_id || "")}">${escapeHTML(payload.artifact_id || "")}</div><div class="item-meta">${escapeHTML(payload.text || "리포트 artifact가 생성되었습니다.")}</div>
        ${reports.reportGenerationSummaryHTML(payload)}
        <div class="report-plan-line"><span class="badge muted">생성 계획</span><span>${escapeHTML(planLabel)}</span></div>
        <div class="report-plan-line"><span class="badge muted">디자인 HTML</span><span>${escapeHTML(designed.label)}</span></div>
        <div class="report-plan-line"><span class="badge muted">H5 말투 보정</span><span>${escapeHTML(humanized.label)}</span></div>
        <div class="report-plan-line"><span class="badge muted">빨간펜 작업본</span><span>${escapeHTML(redpenLabel)}</span></div>
        ${humanized.failureLine}${humanized.redpenLine}
        <div class="report-trace"><div class="report-trace-head"><span class="badge muted">MCP 추적</span><span>${escapeHTML(trace.total ? "도구 호출 기록 있음" : "기록된 MCP 호출 없음")}</span></div>${reports.renderTraceBars(trace)}</div>
        ${reports.reportSourceContextHTML(payload)}
        <div class="item-actions"><button type="button" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="view-artifact">Markdown 보기</button>${redpenView}<button type="button" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="view-html-artifact">기본 HTML 보기</button><button type="button" class="secondary" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-report-title="${escapeAttr(payload.title || "Markdown report")}" data-action="patch-artifact" ${state.reportPending ? "disabled" : ""}>MCP 패치</button>${humanized.actions}${designed.actions}${reports.reportActionMenu("도구 ▾", `<button type="button" class="secondary" data-detail-title="리포트 artifact 상세" data-detail-json="${escapeAttr(JSON.stringify(payload))}">자세히</button>${planButton}<button type="button" class="danger" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="delete-report-artifact">보고서 삭제</button>`)}${reports.reportActionMenu("받기 ▾", `<button type="button" class="secondary" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="download-artifact">MD 받기</button>${redpenDownload}${humanized.redpenDownload}<button type="button" class="secondary" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="download-html-artifact">기본 HTML 받기</button>`)}</div>
        ${reports.reportPreviewInlineHTML(key)}</div></div>`;
  }

  function renderArtifactReportSection(artifactCards, selectedKey) {
    return `<div class="list-section-label">Markdown artifact</div>${artifactCards.map(({ key, isLatest, payload }) => renderArtifactCard(key, isLatest, payload, selectedKey)).join("")}`;
  }
  Object.assign(reports, { renderArtifactReportSection });
})(window);

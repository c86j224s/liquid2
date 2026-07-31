(function reportsTrace(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const state = root.Plasma.state;
  const $ = root.Plasma.dom.$;
  const escapeHTML = root.Plasma.dom.escapeHTML;
  const escapeAttr = root.Plasma.dom.escapeAttr;
  const timeShort = root.Plasma.dom.timeShort;
function showReportPlan(versionID) {
  const plan = reports.reportPlanPayload(versionID);
  showReportPlanPayload(plan);
}

function showReportPlanEvent(eventID) {
  const plan = reports.reportPlanPayloadByEventID(eventID);
  showReportPlanPayload(plan);
}

function showReportPlanPayload(plan) {
  const data = plan.plan || {};
  state.detailText = JSON.stringify(plan || {}, null, 2);
  $("detailTitle").textContent = "리포트 생성 계획";
  if (!Object.keys(plan).length) {
    $("detailBody").innerHTML = `<p class="detail-meta">이 리포트 버전에는 저장된 생성 계획이 없습니다.</p>`;
    $("detailModal").classList.remove("hidden");
    return;
  }
  const sections = Array.isArray(data.sections) ? data.sections : [];
  $("detailBody").innerHTML = `
    <section class="detail-section">
      <h3>계획 요약</h3>
      <p>${escapeHTML(data.summary || "요약 없음")}</p>
      <div class="detail-meta">${escapeHTML(plan.agent_executor || "")} / ${escapeHTML(plan.tool_session_id || "")} / ${escapeHTML(plan.duration_ms || 0)}ms</div>
    </section>
    <section class="detail-section">
      <h3>섹션 계획</h3>
      ${sections.length ? sections.map((section) => `
        <div class="trace-entry">
          <div class="trace-entry-head">
            <strong>${escapeHTML(section.title || "제목 없음")}</strong>
          </div>
          <p>${escapeHTML(section.purpose || "목적 없음")}</p>
          ${detailChips(refValues(section.target_refs || {}))}
        </div>
      `).join("") : `<p class="detail-meta">섹션 계획 없음</p>`}
    </section>
    <section class="detail-grid">
      <div class="detail-box">
        <h3>활용 메모</h3>
        ${detailList(data.coverage_notes || [], "기록된 활용 메모 없음")}
      </div>
      <div class="detail-box">
        <h3>누락/한계</h3>
        ${detailList(data.planned_omissions || [], "기록된 누락 사항 없음")}
      </div>
    </section>
  `;
  $("detailModal").classList.remove("hidden");
}

function showMCPTrace(versionID) {
  const drafted = reports.reportDraftedPayload(versionID);
  const plan = reports.reportPlanPayload(versionID);
  const toolSessionID = drafted.generation?.tool_session_id || plan.tool_session_id || "";
  const trace = reports.mcpTraceSummary(toolSessionID);
  state.detailText = JSON.stringify(trace.events, null, 2);
  $("detailTitle").textContent = "MCP 호출 추적";
  if (!trace.total) {
    $("detailBody").innerHTML = `<p class="detail-meta">이 리포트 버전에 연결된 MCP 호출 기록이 없습니다.</p>`;
    $("detailModal").classList.remove("hidden");
    return;
  }
  $("detailBody").innerHTML = `
    <section class="detail-section">
      <h3>요약</h3>
      <div class="chip-row">
        <span class="badge">총 ${trace.total}회</span>
        <span class="badge ${trace.errors ? "warn" : "muted"}">오류 ${trace.errors}회</span>
        <span class="badge muted">${Math.round(trace.totalDuration)}ms</span>
        <span class="badge muted">${escapeHTML(toolSessionID)}</span>
      </div>
      ${reports.renderTraceBars(trace)}
    </section>
    <section class="detail-section">
      <h3>호출 목록</h3>
      <div class="trace-list">
        ${trace.events.map(renderTraceEvent).join("")}
      </div>
    </section>
  `;
  $("detailModal").classList.remove("hidden");
}

function renderTraceEvent(event) {
  const payload = event.Payload || {};
  const result = payload.result || {};
  const args = payload.arguments || {};
  const ok = payload.success !== false;
  return `
    <div class="trace-entry ${ok ? "" : "failed"}">
      <div class="trace-entry-head">
        <strong>${escapeHTML(reports.toolShortName(payload.tool_name || "unknown"))}</strong>
        <span class="badge ${ok ? "muted" : "warn"}">${ok ? "성공" : "실패"}</span>
      </div>
      <div class="detail-meta">#${escapeHTML(event.Sequence || "")} / ${escapeHTML(timeShort(event.CreatedAt))} / ${escapeHTML(payload.duration_ms || 0)}ms</div>
      <div class="trace-args">${escapeHTML(traceArgSummary(args))}</div>
      ${result.error ? `<div class="trace-error">${escapeHTML(result.error.message || "오류 메시지 없음")}</div>` : ""}
    </div>
  `;
}

function traceArgSummary(args) {
  const parts = [];
  for (const key of ["object_kind", "object_id", "query", "snapshot_id", "artifact_id", "claim_id", "evidence_id", "limit", "offset", "max_bytes"]) {
    if (args[key] === undefined || args[key] === null || args[key] === "") continue;
    parts.push(`${key}=${JSON.stringify(args[key])}`);
  }
  return parts.length ? parts.join(" / ") : "핵심 인자 없음";
}

function refValues(refs) {
  const values = [];
  for (const key of ["claim_ids", "evidence_ids", "snapshot_ids", "question_ids", "option_ids"]) {
    for (const value of refs[key] || []) {
      if (value && !values.includes(value)) values.push(value);
    }
  }
  return values;
}

function detailList(items, emptyText) {
  if (!Array.isArray(items) || !items.length) return `<p class="detail-meta">${escapeHTML(emptyText)}</p>`;
  return `<ul>${items.map((item) => `<li>${escapeHTML(item)}</li>`).join("")}</ul>`;
}


  Object.assign(reports, { showReportPlan, showReportPlanEvent, showReportPlanPayload, showMCPTrace, renderTraceEvent, traceArgSummary, refValues, detailList });
})(window);

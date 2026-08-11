(function reportsViewState(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const state = root.Plasma.state;
  const escapeHTML = root.Plasma.dom.escapeHTML;
  const escapeAttr = root.Plasma.dom.escapeAttr;
  const timeShort = root.Plasma.dom.timeShort;
  const REPORT_RIGOR_LABELS = reports.REPORT_RIGOR_LABELS;
function reportPlanSectionCount(planData) {
  const parts = Array.isArray(planData?.parts) ? planData.parts : [];
  if (parts.length) {
    return parts.reduce((count, part) => count + (Array.isArray(part?.sections) ? part.sections.length : 0), 0);
  }
  return Array.isArray(planData?.sections) ? planData.sections.length : 0;
}

function reportPlanLabel(planData) {
  const summary = planData?.summary || "";
  if (summary) return `${reportPlanSectionCount(planData)}개 섹션 / ${summary}`;
  if (Array.isArray(planData?.sections) || (Array.isArray(planData?.parts) && planData.parts.length)) return `${reportPlanSectionCount(planData)}개 섹션`;
  return "";
}

function reportViewModel(version, index) {
  const versionID = version.report_version_id || "";
  const drafted = reportDraftedPayload(versionID);
  const generation = drafted.generation || {};
  const plan = reportPlanPayload(versionID);
  const trace = mcpTraceSummary(generation.tool_session_id || plan.tool_session_id || "");
  const exports = reportExportPayloads(versionID);
  const title = reportTitle(version) || "리포트";
  const createdLabel = version.created_at ? timeShort(version.created_at) : "생성 시각 없음";
  const exportTargets = exports.map((item) => exportTargetLabel(item.target)).filter(Boolean);
  const planData = plan.plan || {};
  return {
    version,
    versionID,
    title,
    isLatest: index === 0,
    createdLabel,
    stateLabel: reportStateLabel(version.state),
    rigorLabel: generation.rigor_label || REPORT_RIGOR_LABELS[generation.rigor_level] || "",
    exportLabel: exportTargets.length ? `내보냄: ${exportTargets.join(", ")}` : "아직 내보낸 파일 없음",
    plan,
    trace,
    planLabel: reportPlanLabel(planData) || "기록된 생성 계획 없음",
    traceLabel: trace.total ? `총 ${trace.total}회, 오류 ${trace.errors}회` : "기록된 MCP 호출 없음"
  };
}

function reportDraftedPayload(versionID) {
  const events = state.detail?.events || [];
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i];
    if (event.EventType !== "report.drafted") continue;
    const payload = event.Payload || {};
    if (payload.report_version_id === versionID) return payload;
  }
  return {};
}

function reportPlanPayload(versionID) {
  const events = state.detail?.events || [];
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i];
    if (event.EventType !== "report.plan.created") continue;
    const payload = event.Payload || {};
    if (payload.report_version_id === versionID) return payload;
  }
  return {};
}

function reportPlanPayloadByEventID(eventID) {
  const events = state.detail?.events || [];
  for (const event of events) {
    if (event.EventType !== "report.plan.created" || event.EventID !== eventID) continue;
    return { ...(event.Payload || {}), event_id: event.EventID, created_at: event.CreatedAt };
  }
  return {};
}

function mcpTraceEvents(toolSessionID) {
  if (!toolSessionID) return [];
  const events = state.detail?.events || [];
  return events.filter((event) => {
    if (event.EventType !== "mcp.tool.called") return false;
    const payload = event.Payload || {};
    return payload.tool_session_id === toolSessionID || payload.agent_session_id === toolSessionID;
  });
}

function mcpTraceSummary(toolSessionID) {
  const events = mcpTraceEvents(toolSessionID);
  const tools = new Map();
  let errors = 0;
  let totalDuration = 0;
  for (const event of events) {
    const payload = event.Payload || {};
    const name = payload.tool_name || "unknown";
    tools.set(name, (tools.get(name) || 0) + 1);
    if (payload.success === false) errors += 1;
    totalDuration += Number(payload.duration_ms || 0);
  }
  const toolCounts = [...tools.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name));
  return { toolSessionID, events, total: events.length, errors, totalDuration, toolCounts };
}

function renderTraceBars(trace) {
  if (!trace.total) return `<div class="trace-empty">추적 데이터 없음</div>`;
  const max = Math.max(...trace.toolCounts.map((item) => item.count), 1);
  return `
    <div class="trace-bars">
      ${trace.toolCounts.slice(0, 5).map((item) => `
        <div class="trace-bar" title="${escapeAttr(item.name)} ${item.count}회">
          <span>${escapeHTML(toolShortName(item.name))}</span>
          <i><b style="width:${Math.max(8, Math.round((item.count / max) * 100))}%"></b></i>
          <em>${item.count}</em>
        </div>
      `).join("")}
    </div>
  `;
}

function reportExportPayloads(versionID) {
  const events = state.detail?.events || [];
  return events
    .filter((event) => event.EventType === "report.exported")
    .map((event) => event.Payload || {})
    .filter((payload) => payload.report_version_id === versionID);
}

function reportTitle(version) {
  const report = (state.detail?.reports || []).find((item) => item.report_id === version.report_id);
  return report?.title || version.report_id || version.report_version_id || "";
}

function reportStateLabel(stateValue) {
  switch (stateValue) {
    case "draft":
      return "초안";
    case "export_candidate":
      return "내보내기 가능";
    default:
      return stateValue || "상태 미정";
  }
}

function exportTargetLabel(target) {
  switch (target) {
    case "markdown":
      return "MD";
    case "html":
      return "HTML";
    case "json_ast":
      return "JSON";
    default:
      return target || "";
  }
}

function toolShortName(name) {
  return String(name || "unknown")
    .replace(/^plasma\./, "")
    .replace(/^research\./, "research ")
    .replace(/^sources\./, "sources ")
    .replace(/\./g, " ");
}



  Object.assign(reports, { reportViewModel, reportPlanSectionCount, reportPlanLabel, reportDraftedPayload, reportPlanPayload, reportPlanPayloadByEventID, mcpTraceEvents, mcpTraceSummary, renderTraceBars, reportExportPayloads, reportTitle, reportStateLabel, exportTargetLabel, toolShortName });
})(window);

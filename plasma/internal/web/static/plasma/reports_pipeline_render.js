(function reportsPipelineRender(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const state = root.Plasma.state;
  const core = reports.pipelineCore;
  const captureMissionSelection = root.Plasma.mission.captureMissionSelection;
  const ownsMissionSelection = root.Plasma.mission.ownsMissionSelection;
  const isStaleMissionOperation = root.Plasma.mission.isStaleMissionOperation;
  const missionFetch = root.Plasma.transport.missionFetch;
  const timeShort = root.Plasma.dom.timeShort;
  let pipelineLiveTimingTimer = 0;
  function renderActions(progress, independent, unverified) {
    if (progress.state !== "failed") return "";
    const retry = progress.retry || {};
    if (independent && unverified) {
      const fallback = "무검증 보고서 실패는 교정이나 자동 재시도 없이 종료됩니다.";
      return `<div class="pipeline-actions report-il-retry-boundary"><span class="badge warn">재시도 없음</span><span>${core.escapeHTML(fallback)}</span></div>`;
    }
    const disabled = (allowed) => allowed ? "" : "disabled aria-describedby=\"pipelineRetryReason\"";
    const reason = retry.reason ? `<span id="pipelineRetryReason">${core.escapeHTML(retry.reason)}</span>` : "";
    return `<div class="pipeline-actions">
      <button type="button" data-report-retry="resume_failed" ${disabled(retry.resume_failed)}>실패 지점부터 재시도</button>
      <button type="button" class="secondary" data-report-retry="restart" ${disabled(retry.restart)}>처음부터 다시 생성</button>
      ${reason}
    </div>`;
  }

  function currentReportAttemptEvent(attemptID) {
    const events = typeof state !== "undefined" && Array.isArray(state.detail?.events) ? state.detail.events : [];
    return events.find((event) => (event.EventID || event.event_id) === attemptID);
  }

  function reportAttemptDetails(progress) {
    const event = currentReportAttemptEvent(progress.attempt_id);
    const payload = event && typeof (event.Payload || event.payload) === "object" ? (event.Payload || event.payload) : {};
    const eventType = event ? (event.EventType || event.event_type || "") : "";
    const title = typeof payload.title === "string" && payload.title.trim() ? payload.title.trim() : "제목 없는 리포트";
    const startedAt = typeof payload.started_at === "string" && payload.started_at.trim() ? payload.started_at.trim() : "생성 시작 시각 알 수 없음";
    const attempt = Number.isInteger(progress.attempt_number) && progress.attempt_number > 0 ? `시도 ${progress.attempt_number}` : "시도 번호 알 수 없음";
    const fanout = payload.report_mode === "long_form" && payload.execution_strategy === "section_fanout";
    const experimental = Boolean(reports.REPORT_IL_PIPELINE_FAMILY) && payload.pipeline_family === reports.REPORT_IL_PIPELINE_FAMILY;
    const unverified = Boolean(reports.REPORT_UNVERIFIED_PIPELINE_FAMILY) && payload.pipeline_family === reports.REPORT_UNVERIFIED_PIPELINE_FAMILY;
    const independent = experimental || unverified;
    const article = payload.output_kind === "article";
    const longFormIL = experimental && payload.report_mode === "long_form";
    const strategy = article ? (payload.report_mode === "long_form" ? "장문 글" : "글") : experimental ? (longFormIL ? "장문 IL 보고서" : "IL 보고서") : unverified ? "무검증 보고서" : fanout ? "장문 · 빠른 병렬" :
      payload.report_mode === "long_form" ? "장문 · 순차" : "일반";
    return { title, startedAt, attempt, strategy, fanout, experimental, longFormIL, unverified, independent, article, draft: eventType === "report.draft.pending" };
  }

  function fallbackRequestSummary(attempt) {
    const startedAt = attempt.startedAt === "생성 시작 시각 알 수 없음"
      ? ""
      : attempt.startedAt;
    const startedAtLabel = startedAt && typeof timeShort === "function" ? timeShort(startedAt) : attempt.startedAt;
    return {
      mode: attempt.strategy,
      strategy: "",
      guidance: "",
      rigor: "미지정",
      model: "미션 설정 상속",
      effort: "미션 설정 상속",
      direction: "지정 없음",
      startedAt: startedAtLabel,
      startedAtDateTime: startedAt
    };
  }

  function renderRequestDetails(summary, open) {
    const request = summary || {};
    const item = (label, value, className = "") => {
      const display = value || "미지정";
      const extraClass = className ? ` ${className}` : "";
      return `<span class="report-generation-item${extraClass}"><strong>${core.escapeHTML(label)}</strong><span>${core.escapeHTML(display)}</span></span>`;
    };
    const startedAt = request.startedAt || "미지정";
    const startedAtTime = `<span class="report-generation-item"><strong>전체 생성 시작</strong><span><time datetime="${core.escapeHTML(request.startedAtDateTime || "")}">${core.escapeHTML(startedAt)}</time></span></span>`;
    return `<details class="pipeline-request-details"${open ? " open" : ""}><summary>생성 요청 상세</summary>
      <div class="report-generation-summary" aria-label="생성 요청 설정">
        ${item("방식", request.mode)}
        ${request.strategy ? item("장문 작성", request.strategy) : ""}
        ${request.guidance ? item("글쓰기", request.guidance) : ""}
        ${item("엄격도", request.rigor)}
        ${item("모델", request.model)}
        ${item("추론", request.effort)}
        ${request.session ? item("세션", request.session) : ""}
        ${request.tools ? item("능력", request.tools) : ""}
        ${request.humanize ? item("말투 보정", request.humanize) : ""}
        ${item("방향", request.direction, "report-direction-line")}
        ${startedAtTime}
      </div>
    </details>`;
  }

  function renderExperimentalStages(nodes) {
    const stageNodes = nodes.filter((node) => (reports.REPORT_IL_STAGE_LABELS || {})[node.kind]);
    const detailedNodes = nodes.filter((node) => node.kind === "section" || node.kind === "part_edit");
    const stageList = `<ol class="report-il-stage-list" aria-label="IL 보고서 생성 단계">${stageNodes.map((node) => {
      const status = core.status[node.state] || core.status.unknown;
      const timing = core.nodeTiming(node);
      return `<li class="report-il-stage state-${core.escapeHTML(node.state)}"${node.state === "running" ? " aria-current=\"step\"" : ""}>
        <span class="pipeline-icon" aria-hidden="true">${status.icon}</span>
        <span><strong>${core.escapeHTML(core.nodeLabel(node))}</strong><small>${core.escapeHTML(status.text)}${timing ? ` · ${core.escapeHTML(timing)}` : ""}</small>${node.error ? `<em>${core.escapeHTML(node.error)}</em>` : ""}</span>
      </li>`;
    }).join("")}</ol>`;
    if (!detailedNodes.length) return stageList;
    return `${stageList}<ol class="pipeline-flow sr-only" aria-label="장문 IL 섹션과 파트 진행 상태">${detailedNodes.map(core.renderAccessibleNode).join("")}</ol>`;
  }

  async function requestRetry(button, progress) {
    const owner = captureMissionSelection();
    const requestID = button.dataset.retryRequestId || core.retryRequestID();
    button.dataset.retryRequestId = requestID;
    button.disabled = true;
    try {
      const response = await missionFetch(owner, "/reports/retry", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ failed_pending_event_id: progress.attempt_id, strategy: button.dataset.reportRetry, retry_request_id: requestID }) });
      if (!response.ok) throw new Error("재시도 요청을 처리할 수 없습니다.");
      if (ownsMissionSelection(owner)) await reports.call("reloadMission", owner.missionId);
    } catch (error) {
      if (isStaleMissionOperation(error) || !ownsMissionSelection(owner)) return;
      button.disabled = false;
      const reason = document.getElementById("pipelineRetryReason");
      if (reason) reason.textContent = error.message;
    }
  }

  function stopLiveTiming() {
    if (!pipelineLiveTimingTimer) return;
    clearInterval(pipelineLiveTimingTimer);
    pipelineLiveTimingTimer = 0;
  }

  function liveTimingText(startedAt) {
    const clock = core.formatClock(startedAt);
    if (!clock) return "";
    const duration = core.formatDuration(Date.now() - new Date(startedAt).getTime());
    return `시작 ${clock}${duration ? `, 경과 ${duration}` : ""}`;
  }

  function updateLiveTiming(root) {
    if (!root || typeof root.querySelectorAll !== "function") return;
    root.querySelectorAll("[data-pipeline-live-timing]").forEach((node) => {
      const timing = liveTimingText(node.dataset.pipelineStartedAt);
      if (!timing) return;
      node.textContent = node.dataset.pipelineTitlePrefix ? `${node.dataset.pipelineTitlePrefix} ${timing}` : timing;
    });
  }

  function syncLiveTiming(root) {
    stopLiveTiming();
    if (!root || typeof root.querySelector !== "function" || !root.querySelector("[data-pipeline-live-timing]")) return;
    updateLiveTiming(root);
    if (typeof setInterval !== "function") return;
    pipelineLiveTimingTimer = setInterval(() => {
      const host = document.getElementById("reportPipeline");
      if (!host || !host.querySelector("[data-pipeline-live-timing]")) { stopLiveTiming(); return; }
      updateLiveTiming(host);
    }, 1000);
  }

  function render(progress, requestSummary) {
    const host = document.getElementById("reportPipeline");
    if (!host) return;
    if (!progress || progress.state === "unknown") { stopLiveTiming(); host.innerHTML = ""; return; }
    const nodes = Array.isArray(progress.nodes) ? progress.nodes : [];
    const detailed = core.hasPlannedContent(nodes);
    const attempt = reportAttemptDetails(progress);
    const plan = core.planNode(nodes, progress);
    const phases = detailed ? core.reportPhases(nodes) : [];
    const closing = detailed || attempt.draft ? core.finalEditClosingNodes(nodes) : [];
    const graphNodes = attempt.experimental ? nodes : [plan, ...phases.flatMap((phase) => phase.nodes), ...closing];
    const stage = attempt.unverified
      ? { name: "Markdown 단일 작성", state: (core.status[progress.state] || core.status.unknown).text }
      : core.currentStage(graphNodes);
    const details = typeof host.querySelector === "function" ? host.querySelector(".pipeline-details") : null;
    const requestDetails = typeof host.querySelector === "function" ? host.querySelector(".pipeline-request-details") : null;
    const visual = typeof host.querySelector === "function" ? host.querySelector(".pipeline-visual") : null;
    const detailsOpen = Boolean(details && details.open);
    const requestDetailsOpen = Boolean(requestDetails && requestDetails.open);
    const visualScrollLeft = visual && Number.isFinite(visual.scrollLeft) ? visual.scrollLeft : 0;
    const revealing = detailed && host.dataset && host.dataset.pipelinePhase === "planning";
    if (host.dataset) host.dataset.pipelinePhase = detailed ? "detailed" : "planning";
    const accessibleGraph = [plan].map(core.renderAccessibleNode).join("") + phases.map(core.renderAccessiblePhase).join("") + closing.map(core.renderAccessibleNode).join("");
    const retry = progress.retry || {};
    const reason = !attempt.independent && retry.reason ? `<p id="pipelineRetryReason" class="pipeline-reason">${core.escapeHTML(retry.reason)}</p>` : "";
    const request = requestSummary || fallbackRequestSummary(attempt);
    const pipelineTitle = attempt.article ? "최신 글 생성" : attempt.experimental ? "IL 보고서 생성 파이프라인" : attempt.unverified ? "무검증 보고서 생성" : "최신 리포트 생성 파이프라인";
    const experimentalGraph = attempt.longFormIL
      ? reports.pipelineGraph.longFormILProgressGraph(nodes)
      : reports.pipelineGraph.stageProgressGraph(nodes);
    const pipelineBody = attempt.experimental
      ? `<details class="pipeline-details"${detailsOpen ? " open" : ""}><summary>생성 파이프라인 펼치기</summary>
          <div class="pipeline-visual">${experimentalGraph}</div>
          ${renderExperimentalStages(nodes)}
        </details>${renderActions(progress, true, false)}`
      : attempt.unverified
        ? `<div class="pipeline-actions report-il-retry-boundary"><span class="badge">단일 호출</span><span>자료를 읽고 반환한 Markdown을 내용 수정 없이 저장합니다.</span></div>${renderActions(progress, true, true)}`
        : `<details class="pipeline-details"${detailsOpen ? " open" : ""}><summary>생성 파이프라인 펼치기</summary>
          <div class="pipeline-visual">${reports.pipelineGraph.progressGraph(plan, phases, closing, revealing, attempt.fanout)}</div>
          <ol class="pipeline-flow sr-only" aria-label="리포트 생성 단계의 상태">${accessibleGraph}</ol>${reason}${renderActions(progress, false, false)}
        </details>`;
    host.innerHTML = `<section class="report-pipeline${attempt.independent ? " report-il-pipeline" : ""}" aria-labelledby="reportPipelineTitle">
      <header class="pipeline-header"><div><h3 id="reportPipelineTitle">${pipelineTitle}</h3><p class="pipeline-report-title">${core.escapeHTML(attempt.title)}</p></div><p class="pipeline-current" aria-live="polite"><strong>${core.escapeHTML(attempt.attempt)}</strong><span class="pipeline-current-step">${core.escapeHTML(stage.name)}</span><span class="pipeline-current-status">${core.escapeHTML(stage.state)}</span></p></header>
      ${renderRequestDetails(request, requestDetailsOpen)}
      ${pipelineBody}
    </section>`;
    host.querySelectorAll("[data-report-retry]").forEach((button) => button.addEventListener("click", () => requestRetry(button, progress)));
    host.querySelectorAll(".pipeline-phase details").forEach((details) => details.addEventListener("toggle", () => {
      const summary = details.querySelector("summary");
      if (summary) summary.setAttribute("aria-expanded", String(details.open));
    }));
    const renderedVisual = typeof host.querySelector === "function" ? host.querySelector(".pipeline-visual") : null;
    if (renderedVisual) renderedVisual.scrollLeft = visualScrollLeft;
    syncLiveTiming(host);
    if (revealing && typeof requestAnimationFrame === "function") requestAnimationFrame(() => {
      const graph = typeof host.querySelector === "function" ? host.querySelector(".pipeline-graph-revealing") : null;
      if (graph && graph.classList) graph.classList.remove("pipeline-graph-revealing");
    });
  };


  reports.pipeline = { render };
})(window);

(function reportPipelineModule() {
  const status = {
    pending: { icon: "o", text: "대기" }, running: { icon: "~", text: "진행 중" },
    completed: { icon: "+", text: "완료" }, failed: { icon: "!", text: "실패" },
    skipped: { icon: "-", text: "건너뜀" }, unknown: { icon: "?", text: "알 수 없음" }
  };

  function escapeHTML(value) {
    return String(value || "").replace(/[&<>"']/g, (char) => ({
      "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;"
    }[char]));
  }

  function retryRequestID() {
    return crypto.randomUUID ? crypto.randomUUID() : `retry-${Date.now()}-${Math.random().toString(16).slice(2)}`;
  }

  function nodeLabel(node) {
    const coordinate = node.part_index ? ` ${node.part_index}${node.section_index ? `.${node.section_index}` : ""}` : "";
    return `${node.kind}${coordinate}`;
  }

  function nodeDescription(node) {
    const state = status[node.state] || status.unknown;
    const timing = nodeTiming(node);
    return `${nodeLabel(node)} ${state.text}${timing ? `, ${timing}` : ""}${node.error ? `: ${node.error}` : ""}`;
  }

  function formatClock(value) {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return "";
    return date.toLocaleTimeString("ko-KR", { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
  }

  function formatDuration(value) {
    const milliseconds = Number(value);
    if (!Number.isFinite(milliseconds) || milliseconds < 0) return "";
    const seconds = Math.floor(milliseconds / 1000);
    if (seconds < 60) return `${seconds}초`;
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}분 ${seconds % 60}초`;
    return `${Math.floor(minutes / 60)}시간 ${minutes % 60}분`;
  }

  function nodeTiming(node) {
    const startedAt = typeof node.started_at === "string" ? node.started_at : "";
    const clock = formatClock(startedAt);
    if (!clock) return "";
    const duration = node.state === "running" ? formatDuration(Date.now() - new Date(startedAt).getTime()) : formatDuration(node.duration_ms);
    return `시작 ${clock}${duration ? `, ${node.state === "running" ? "경과" : "소요"} ${duration}` : ""}`;
  }

  function planNode(nodes, progress) {
    return nodes.find((node) => node.id === "plan" || node.id === "start" || node.kind === "plan") || {
      id: "plan", kind: "plan", state: progress.state || "pending"
    };
  }

  function hasPlannedContent(nodes) {
    return nodes.some((node) => node.kind === "part" || node.kind === "section");
  }

  function stageName(node) {
    if (node.kind === "section") return `파트 ${node.part_index} 섹션 ${node.section_index} 작성`;
    if (node.kind === "part") return `파트 ${node.part_index} 작성`;
    if (node.kind === "final") return "최종화";
    if (node.kind === "artifact") return "산출물 생성";
    return "계획 수립";
  }

  function currentStage(nodes) {
    const current = nodes.find((node) => node.state === "running") ||
      nodes.find((node) => node.state === "failed") ||
      nodes.find((node) => node.state === "pending") ||
      nodes[nodes.length - 1];
    const state = status[current && current.state] || status.unknown;
    return { name: stageName(current || {}), state: state.text };
  }

  function renderAccessibleNode(node) {
    const state = status[node.state] || status.unknown;
    const current = node.state === "running" ? " aria-current=\"step\"" : "";
    return `<li class="pipeline-node state-${escapeHTML(node.state)}" id="pipeline-${escapeHTML(node.id)}" role="listitem" tabindex="0" aria-label="${escapeHTML(nodeDescription(node))}"${current}>
      <span class="pipeline-icon" aria-hidden="true">${state.icon}</span>
      <span class="pipeline-label">${escapeHTML(nodeLabel(node))}</span>
      <span class="pipeline-status">${state.text}</span>
      ${nodeTiming(node) ? `<span class="pipeline-timing">${escapeHTML(nodeTiming(node))}</span>` : ""}
      ${node.error ? `<span class="pipeline-error">${escapeHTML(node.error)}</span>` : ""}
    </li>`;
  }

  function reportPhases(nodes) {
    return [
      { label: "섹션 작성", nodes: nodes.filter((node) => node.kind === "section") },
      { label: "파트 조립", nodes: nodes.filter((node) => node.kind === "part") }
    ].filter((phase) => phase.nodes.length > 0);
  }

  function renderAccessiblePhase(phase) {
    const nodes = phase.nodes;
    const complete = nodes.filter((node) => node.state === "completed").length;
    const visible = nodes.some((node) => node.state === "running" || node.state === "failed");
    return `<li class="pipeline-phase"><details ${visible ? "open" : ""}>
      <summary aria-label="${escapeHTML(phase.label)} 단계 펼치기" aria-expanded="${visible}">${escapeHTML(phase.label)} <span>${complete}/${nodes.length}</span></summary>
      <ul>${nodes.map(renderAccessibleNode).join("")}</ul>
    </details></li>`;
  }

  function visualNode(node, x, y, width, fixed) {
    const label = escapeHTML(nodeLabel(node));
    const state = escapeHTML(node.state || "unknown");
    const timing = escapeHTML(nodeTiming(node));
    const fixedClass = fixed ? " pipeline-visual-node-plan" : "";
    return `<g class="pipeline-visual-node${fixedClass} state-${state}" data-pipeline-node-width="${width}" transform="translate(${x} ${y})">
      ${timing ? `<title>${escapeHTML(`${nodeLabel(node)} ${timing}`)}</title>` : ""}
      <circle class="pipeline-visual-dot" r="5"></circle>
      <text class="pipeline-visual-label" y="27" text-anchor="middle">${label}</text>
      ${timing ? `<text class="pipeline-visual-time" y="44" text-anchor="middle">${timing}</text>` : ""}
    </g>`;
  }

  function connector(x1, x2) {
    return `<path class="pipeline-connector" d="M ${x1 + 7} 62 H ${x2 - 9}" marker-end="url(#pipeline-arrow)"></path>`;
  }

  function visualNodeWidth(node) {
    const labelWidth = nodeLabel(node).length * 12 + 24;
    const timing = nodeTiming(node);
    const timingWidth = timing ? timing.length * 10 + 24 : 0;
    return Math.max(144, labelWidth, timingWidth);
  }

  function progressGraph(plan, phases, closing, revealing) {
    const output = [];
    const nodeGap = 32;
    const graphPadding = 32;
    let nextX = graphPadding;
    let previous;
    const addNode = (node) => {
      const width = visualNodeWidth(node);
      const x = previous ? previous.x + previous.width / 2 + nodeGap + width / 2 : nextX + width / 2;
      if (previous) output.push(connector(previous.x, x));
      output.push(visualNode(node, x, 62, width, !previous));
      previous = { x, width };
      nextX = x + width / 2 + graphPadding;
      return previous;
    };
    if (plan) addNode(plan);
    phases.forEach((phase) => {
      let first;
      phase.nodes.forEach((node) => {
        const layout = addNode(node);
        if (!first) first = layout;
      });
      const phaseStart = first.x - first.width / 2 - 14;
      const phaseWidth = previous.x + previous.width / 2 + 14 - phaseStart;
      output.unshift(`<g class="pipeline-visual-phase"><rect x="${phaseStart}" y="16" width="${phaseWidth}" height="82" rx="4"></rect><text class="pipeline-phase-label" x="${phaseStart + 10}" y="34">${escapeHTML(phase.label)}</text></g>`);
    });
    closing.forEach(addNode);
    const width = Math.max(760, nextX);
    const transition = revealing ? " pipeline-graph-revealing" : "";
    return `<svg class="pipeline-graph${transition}" style="--pipeline-width: ${width}px" viewBox="0 0 ${width} 136" role="img" aria-label="계획, 섹션 작성, 파트 조립, 최종화, 산출물 순서의 리포트 생성 진행 상황"><defs><marker id="pipeline-arrow" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M 0 0 L 8 4 L 0 8 z"></path></marker></defs>${output.join("")}</svg>`;
  }

  function renderActions(progress) {
    if (progress.state !== "failed") return "";
    const retry = progress.retry || {};
    const disabled = (allowed) => allowed ? "" : "disabled aria-describedby=\"pipelineRetryReason\"";
    return `<div class="pipeline-actions">
      <button type="button" data-report-retry="resume_failed" ${disabled(retry.resume_failed)}>실패 지점부터 재시도</button>
      <button type="button" class="secondary" data-report-retry="restart" ${disabled(retry.restart)}>처음부터 다시 생성</button>
    </div>`;
  }

  function currentReportAttemptEvent(attemptID) {
    const events = typeof state !== "undefined" && Array.isArray(state.detail?.events) ? state.detail.events : [];
    return events.find((event) => (event.EventID || event.event_id) === attemptID);
  }

  function reportAttemptDetails(progress) {
    const event = currentReportAttemptEvent(progress.attempt_id);
    const payload = event && typeof (event.Payload || event.payload) === "object" ? (event.Payload || event.payload) : {};
    const title = typeof payload.title === "string" && payload.title.trim() ? payload.title.trim() : "제목 없는 리포트";
    const startedAt = typeof payload.started_at === "string" && payload.started_at.trim() ? payload.started_at.trim() : "생성 시작 시각 알 수 없음";
    const attempt = Number.isInteger(progress.attempt_number) && progress.attempt_number > 0 ? `시도 ${progress.attempt_number}` : "시도 번호 알 수 없음";
    return { title, startedAt, attempt };
  }

  async function requestRetry(button, progress) {
    const owner = captureMissionSelection();
    const requestID = button.dataset.retryRequestId || retryRequestID();
    button.dataset.retryRequestId = requestID;
    button.disabled = true;
    try {
      const response = await missionFetch(owner, "/reports/retry", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ failed_pending_event_id: progress.attempt_id, strategy: button.dataset.reportRetry, retry_request_id: requestID }) });
      if (!response.ok) throw new Error("재시도 요청을 처리할 수 없습니다.");
      if (ownsMissionSelection(owner)) await reloadMission(owner.missionId);
    } catch (error) {
      if (isStaleMissionOperation(error) || !ownsMissionSelection(owner)) return;
      button.disabled = false;
      const reason = document.getElementById("pipelineRetryReason");
      if (reason) reason.textContent = error.message;
    }
  }

  window.renderReportPipeline = function renderReportPipeline(progress) {
    const host = document.getElementById("reportPipeline");
    if (!host) return;
    if (!progress || progress.state === "unknown") { host.innerHTML = ""; return; }
    const nodes = Array.isArray(progress.nodes) ? progress.nodes : [];
    const detailed = hasPlannedContent(nodes);
    const plan = planNode(nodes, progress);
    const phases = detailed ? reportPhases(nodes) : [];
    const closing = detailed ? nodes.filter((node) => node.id === "final" || node.id === "artifact" || node.kind === "final" || node.kind === "artifact") : [];
    const graphNodes = [plan, ...phases.flatMap((phase) => phase.nodes), ...closing];
    const stage = currentStage(graphNodes);
    const details = typeof host.querySelector === "function" ? host.querySelector(".pipeline-details") : null;
    const visual = typeof host.querySelector === "function" ? host.querySelector(".pipeline-visual") : null;
    const detailsOpen = Boolean(details && details.open);
    const visualScrollLeft = visual && Number.isFinite(visual.scrollLeft) ? visual.scrollLeft : 0;
    const revealing = detailed && host.dataset && host.dataset.pipelinePhase === "planning";
    if (host.dataset) host.dataset.pipelinePhase = detailed ? "detailed" : "planning";
    const accessibleGraph = [plan].map(renderAccessibleNode).join("") + phases.map(renderAccessiblePhase).join("") + closing.map(renderAccessibleNode).join("");
    const retry = progress.retry || {};
    const reason = retry.reason ? `<p id="pipelineRetryReason" class="pipeline-reason">${escapeHTML(retry.reason)}</p>` : "";
    const attempt = reportAttemptDetails(progress);
    host.innerHTML = `<section class="report-pipeline" aria-labelledby="reportPipelineTitle">
      <header class="pipeline-header"><div><h3 id="reportPipelineTitle">최신 리포트 생성 파이프라인</h3><p class="pipeline-report-title">${escapeHTML(attempt.title)}</p></div><p class="pipeline-current" aria-live="polite"><strong>${escapeHTML(attempt.attempt)}</strong><span class="pipeline-current-step">${escapeHTML(stage.name)}</span><span class="pipeline-current-status">${escapeHTML(stage.state)}</span></p></header>
      <dl class="pipeline-attempt-meta"><div><dt>전체 생성 시작</dt><dd><time datetime="${escapeHTML(attempt.startedAt === "생성 시작 시각 알 수 없음" ? "" : attempt.startedAt)}">${escapeHTML(attempt.startedAt)}</time></dd></div></dl>
      <details class="pipeline-details"${detailsOpen ? " open" : ""}><summary>생성 파이프라인 펼치기</summary>
        <div class="pipeline-visual">${progressGraph(plan, phases, closing, revealing)}</div>
        <ol class="pipeline-flow sr-only" aria-label="리포트 생성 단계의 상태">${accessibleGraph}</ol>${reason}${renderActions(progress)}
      </details>
    </section>`;
    host.querySelectorAll("[data-report-retry]").forEach((button) => button.addEventListener("click", () => requestRetry(button, progress)));
    host.querySelectorAll(".pipeline-phase details").forEach((details) => details.addEventListener("toggle", () => {
      const summary = details.querySelector("summary");
      if (summary) summary.setAttribute("aria-expanded", String(details.open));
    }));
    const renderedVisual = typeof host.querySelector === "function" ? host.querySelector(".pipeline-visual") : null;
    if (renderedVisual) renderedVisual.scrollLeft = visualScrollLeft;
    if (revealing && typeof requestAnimationFrame === "function") requestAnimationFrame(() => {
      const graph = typeof host.querySelector === "function" ? host.querySelector(".pipeline-graph-revealing") : null;
      if (graph && graph.classList) graph.classList.remove("pipeline-graph-revealing");
    });
  };
})();

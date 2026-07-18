(function reportPipelineModule() {
  const status = {
    pending: { icon: "o", text: "대기" }, running: { icon: "~", text: "진행 중" },
    completed: { icon: "+", text: "완료" }, failed: { icon: "!", text: "실패" },
    skipped: { icon: "-", text: "건너뜀" }, unknown: { icon: "?", text: "알 수 없음" }
  };
  let pipelineLiveTimingTimer = 0;

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

  function liveTimingAttributes(node, titlePrefix) {
    if (node.state !== "running" || !formatClock(node.started_at)) return "";
    const title = titlePrefix ? ` data-pipeline-title-prefix="${escapeHTML(titlePrefix)}"` : "";
    return ` data-pipeline-live-timing="1" data-pipeline-started-at="${escapeHTML(node.started_at)}"${title}`;
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
    const runningSections = nodes.filter((node) => node.kind === "section" && node.state === "running");
    if (runningSections.length > 1) return { name: `섹션 ${runningSections.length}개 병렬 작성`, state: status.running.text };
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
    const timing = nodeTiming(node);
    return `<li class="pipeline-node state-${escapeHTML(node.state)}" id="pipeline-${escapeHTML(node.id)}" role="listitem" tabindex="0" aria-label="${escapeHTML(nodeDescription(node))}"${current}>
      <span class="pipeline-icon" aria-hidden="true">${state.icon}</span>
      <span class="pipeline-label">${escapeHTML(nodeLabel(node))}</span>
      <span class="pipeline-status">${state.text}</span>
      ${timing ? `<span class="pipeline-timing"${liveTimingAttributes(node)}>${escapeHTML(timing)}</span>` : ""}
      ${node.error ? `<span class="pipeline-error">${escapeHTML(node.error)}</span>` : ""}
    </li>`;
  }

  function reportPhases(nodes) {
    return [
      { label: "섹션 작성", nodes: nodes.filter((node) => node.kind === "section") },
      { label: "파트 조립", nodes: nodes.filter((node) => node.kind === "part") }
    ].filter((phase) => phase.nodes.length > 0);
  }

  function phaseSummary(nodes) {
    const complete = nodes.filter((node) => node.state === "completed").length;
    const running = nodes.filter((node) => node.state === "running").length;
    const failed = nodes.filter((node) => node.state === "failed").length;
    const suffix = running ? ` · 진행 ${running}` : failed ? ` · 실패 ${failed}` : "";
    return `${complete}/${nodes.length}${suffix}`;
  }

  function renderAccessiblePhase(phase) {
    const nodes = phase.nodes;
    const visible = nodes.some((node) => node.state === "running" || node.state === "failed");
    return `<li class="pipeline-phase"><details ${visible ? "open" : ""}>
      <summary aria-label="${escapeHTML(phase.label)} 단계 펼치기" aria-expanded="${visible}">${escapeHTML(phase.label)} <span>${escapeHTML(phaseSummary(nodes))}</span></summary>
      <ul>${nodes.map(renderAccessibleNode).join("")}</ul>
    </details></li>`;
  }

  function visualNode(node, x, y, width, fixed) {
    const label = escapeHTML(nodeLabel(node));
    const state = escapeHTML(node.state || "unknown");
    const timing = escapeHTML(nodeTiming(node));
    const fixedClass = fixed ? " pipeline-visual-node-plan" : "";
    return `<g class="pipeline-visual-node${fixedClass} state-${state}" data-pipeline-node-width="${width}" transform="translate(${x} ${y})">
      ${timing ? `<title${liveTimingAttributes(node, nodeLabel(node))}>${escapeHTML(`${nodeLabel(node)} ${timing}`)}</title>` : ""}
      <circle class="pipeline-visual-dot" r="5"></circle>
      <text class="pipeline-visual-label" y="27" text-anchor="middle">${label}</text>
      ${timing ? `<text class="pipeline-visual-time" y="44" text-anchor="middle"${liveTimingAttributes(node)}>${timing}</text>` : ""}
    </g>`;
  }

  function connector(x1, x2) {
    return `<path class="pipeline-connector" d="M ${x1 + 7} 62 H ${x2 - 9}" marker-end="url(#pipeline-arrow)"></path>`;
  }

  function pathConnector(x1, y1, x2, y2) {
    const mid = Math.round((x1 + x2) / 2);
    return `<path class="pipeline-connector" d="M ${x1 + 7} ${y1} C ${mid} ${y1}, ${mid} ${y2}, ${x2 - 9} ${y2}" marker-end="url(#pipeline-arrow)"></path>`;
  }

  function visualNodeWidth(node) {
    const labelWidth = nodeLabel(node).length * 12 + 24;
    const timing = nodeTiming(node);
    const timingWidth = timing ? timing.length * 10 + 24 : 0;
    return Math.max(144, labelWidth, timingWidth);
  }

  function progressGraph(plan, phases, closing, revealing, fanout) {
    if (fanout) return fanoutProgressGraph(plan, phases, closing, revealing);
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
    return `<svg class="pipeline-graph${transition}" style="--pipeline-width: ${width}px; --pipeline-height: 136px" viewBox="0 0 ${width} 136" role="img" aria-label="계획, 섹션 작성, 파트 조립, 최종화, 산출물 순서의 리포트 생성 진행 상황"><defs>${arrowMarker()}</defs>${output.join("")}</svg>`;
  }

  function arrowMarker() {
    return `<marker id="pipeline-arrow" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M 0 0 L 8 4 L 0 8 z"></path></marker>`;
  }

  function groupByPart(nodes) {
    const groups = new Map();
    nodes.forEach((node) => {
      const part = Number.isInteger(node.part_index) && node.part_index > 0 ? node.part_index : 1;
      if (!groups.has(part)) groups.set(part, []);
      groups.get(part).push(node);
    });
    groups.forEach((items) => items.sort((a, b) => (a.section_index || 0) - (b.section_index || 0)));
    return [...groups.entries()].sort(([a], [b]) => a - b);
  }

  function maxNodeWidth(nodes) {
    return nodes.reduce((width, node) => Math.max(width, visualNodeWidth(node)), 144);
  }

  function fanoutProgressGraph(plan, phases, closing, revealing) {
    const sections = phases.flatMap((phase) => phase.nodes).filter((node) => node.kind === "section");
    const parts = phases.flatMap((phase) => phase.nodes).filter((node) => node.kind === "part");
    if (!sections.length || !parts.length) return progressGraph(plan, phases, closing, revealing, false);
    const output = [];
    const padding = 36;
    const nodeGap = 42;
    const branchGap = 76;
    const rowGap = 84;
    const rows = groupByPart(sections);
    const rowCount = Math.max(1, rows.length);
    const lastRowY = 62 + (rowCount - 1) * rowGap;
    const centerY = 62 + (lastRowY - 62) / 2;
    const height = Math.max(136, Math.round(lastRowY + 74));
    const maxSectionsPerPart = Math.max(...rows.map(([, items]) => items.length));
    const sectionWidth = maxNodeWidth(sections);
    const partWidth = maxNodeWidth(parts);
    const planWidth = plan ? visualNodeWidth(plan) : 144;
    const planX = padding + planWidth / 2;
    const firstSectionX = planX + planWidth / 2 + branchGap + sectionWidth / 2;
    const sectionStep = sectionWidth + nodeGap;
    const partX = firstSectionX + Math.max(0, maxSectionsPerPart - 1) * sectionStep + sectionWidth / 2 + branchGap + partWidth / 2;
    const closingNodes = closing.map((node) => ({ node, width: visualNodeWidth(node) }));
    let closingX = partX + partWidth / 2 + branchGap;
    const partByIndex = new Map(parts.map((node) => [node.part_index || 1, node]));

    const sectionPhaseStart = firstSectionX - sectionWidth / 2 - 16;
    const sectionPhaseEnd = firstSectionX + Math.max(0, maxSectionsPerPart - 1) * sectionStep + sectionWidth / 2 + 16;
    output.push(`<g class="pipeline-visual-phase pipeline-visual-phase-fanout"><rect x="${sectionPhaseStart}" y="16" width="${sectionPhaseEnd - sectionPhaseStart}" height="${height - 38}" rx="4"></rect><text class="pipeline-phase-label" x="${sectionPhaseStart + 10}" y="34">섹션 작성</text></g>`);
    output.push(`<g class="pipeline-visual-phase pipeline-visual-phase-fanout"><rect x="${partX - partWidth / 2 - 16}" y="16" width="${partWidth + 32}" height="${height - 38}" rx="4"></rect><text class="pipeline-phase-label" x="${partX - partWidth / 2 - 6}" y="34">파트 조립</text></g>`);
    if (plan) output.push(visualNode(plan, planX, centerY, planWidth, true));

    const partLayouts = [];
    rows.forEach(([partIndex, items], rowIndex) => {
      const y = 62 + rowIndex * rowGap;
      items.forEach((node, index) => {
        const x = firstSectionX + index * sectionStep;
        if (plan) output.push(pathConnector(planX, centerY, x, y));
        output.push(visualNode(node, x, y, sectionWidth, false));
      });
      const partNode = partByIndex.get(partIndex);
      if (!partNode || !items.length) return;
      items.forEach((_node, index) => output.push(pathConnector(firstSectionX + index * sectionStep, y, partX, y)));
      output.push(visualNode(partNode, partX, y, partWidth, false));
      partLayouts.push({ x: partX, y, node: partNode });
    });

    let previousClosing = null;
    closingNodes.forEach(({ node, width }, index) => {
      const x = closingX + width / 2;
      const y = centerY;
      if (index === 0) {
        partLayouts.forEach((layout) => output.push(pathConnector(layout.x, layout.y, x, y)));
      } else if (previousClosing) {
        output.push(pathConnector(previousClosing.x, previousClosing.y, x, y));
      }
      output.push(visualNode(node, x, y, width, false));
      previousClosing = { x, y, width };
      closingX = x + width / 2 + nodeGap;
    });
    const width = Math.max(760, Math.round((previousClosing ? previousClosing.x + previousClosing.width / 2 : partX + partWidth / 2) + padding));
    const transition = revealing ? " pipeline-graph-revealing" : "";
    return `<svg class="pipeline-graph pipeline-graph-fanout${transition}" style="--pipeline-width: ${width}px; --pipeline-height: ${height}px" viewBox="0 0 ${width} ${height}" role="img" aria-label="계획에서 여러 섹션 작성으로 갈라지고 파트 조립으로 합쳐지는 병렬 리포트 생성 진행 상황"><defs>${arrowMarker()}</defs>${output.join("")}</svg>`;
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
    const fanout = payload.report_mode === "long_form" && payload.execution_strategy === "section_fanout";
    const strategy = fanout ? "장문 · 빠른 병렬" :
      payload.report_mode === "long_form" ? "장문 · 순차" : "일반";
    return { title, startedAt, attempt, strategy, fanout };
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

  function stopLiveTiming() {
    if (!pipelineLiveTimingTimer) return;
    clearInterval(pipelineLiveTimingTimer);
    pipelineLiveTimingTimer = 0;
  }

  function liveTimingText(startedAt) {
    const clock = formatClock(startedAt);
    if (!clock) return "";
    const duration = formatDuration(Date.now() - new Date(startedAt).getTime());
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

  window.renderReportPipeline = function renderReportPipeline(progress) {
    const host = document.getElementById("reportPipeline");
    if (!host) return;
    if (!progress || progress.state === "unknown") { stopLiveTiming(); host.innerHTML = ""; return; }
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
      <dl class="pipeline-attempt-meta"><div><dt>작성 방식</dt><dd>${escapeHTML(attempt.strategy)}</dd></div><div><dt>전체 생성 시작</dt><dd><time datetime="${escapeHTML(attempt.startedAt === "생성 시작 시각 알 수 없음" ? "" : attempt.startedAt)}">${escapeHTML(attempt.startedAt)}</time></dd></div></dl>
      <details class="pipeline-details"${detailsOpen ? " open" : ""}><summary>생성 파이프라인 펼치기</summary>
        <div class="pipeline-visual">${progressGraph(plan, phases, closing, revealing, attempt.fanout)}</div>
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
    syncLiveTiming(host);
    if (revealing && typeof requestAnimationFrame === "function") requestAnimationFrame(() => {
      const graph = typeof host.querySelector === "function" ? host.querySelector(".pipeline-graph-revealing") : null;
      if (graph && graph.classList) graph.classList.remove("pipeline-graph-revealing");
    });
  };
})();

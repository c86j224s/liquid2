(function reportsPipelineCore(root) {
  "use strict";
  const reports = root.Plasma.reports;
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
    if (node.kind === "requirements") return "요구 연결";
    if (node.kind === "part_plan") return `파트 계획${coordinate}`;
    if (node.kind === "section") return `섹션${coordinate}`;
    if (node.kind === "part") return `파트 조립${coordinate}`;
    if (node.kind === "part_edit") return `파트 편집${coordinate}`;
    if (node.kind === "part_author") return `파트 최종 작성${coordinate}`;
    if (node.kind === "final_assembly") return "최종 조립";
    if (node.kind === "final_write") return "최종 작성";
    if (node.kind === "reader_edit") return "독자 편집";
    if (node.kind === "style_edit") return "말투 편집";
    if (node.kind === "corrective_gate") return "근거·요구 교정";
    if (node.kind === "final") return "최종 편집·확정";
    if (node.kind === "artifact") return "산출물";
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
    if (node.kind === "requirements") return "사용자 요구 연결";
    if (node.kind === "part_plan") return `파트 ${node.part_index} 읽기 흐름 계획`;
    if (node.kind === "section") return `파트 ${node.part_index} 섹션 ${node.section_index} 작성`;
    if (node.kind === "part") return `파트 ${node.part_index} 조립`;
    if (node.kind === "part_edit") return `파트 ${node.part_index} 편집`;
    if (node.kind === "part_author") return `파트 ${node.part_index} 최종 작성`;
    if (node.kind === "final_assembly") return "최종 조립";
    if (node.kind === "final_write") return "최종 작성";
    if (node.kind === "reader_edit") return "독자 편집";
    if (node.kind === "style_edit") return "말투 편집";
    if (node.kind === "corrective_gate") return "근거·요구 교정";
    if (node.kind === "final") return "최종 편집·확정";
    if (node.kind === "artifact") return "산출물 생성";
    return "계획 수립";
  }

  function currentStage(nodes) {
    const runningSections = nodes.filter((node) => node.kind === "section" && node.state === "running");
    if (runningSections.length > 1) return { name: `섹션 ${runningSections.length}개 병렬 작성`, state: status.running.text };
    const runningPartPlans = nodes.filter((node) => node.kind === "part_plan" && node.state === "running");
    if (runningPartPlans.length > 1) return { name: `파트 ${runningPartPlans.length}개 병렬 계획`, state: status.running.text };
    const runningPartEdits = nodes.filter((node) => node.kind === "part_edit" && node.state === "running");
    if (runningPartEdits.length > 1) return { name: `파트 ${runningPartEdits.length}개 병렬 편집`, state: status.running.text };
    const runningPartAuthors = nodes.filter((node) => node.kind === "part_author" && node.state === "running");
    if (runningPartAuthors.length > 1) return { name: `파트 ${runningPartAuthors.length}개 최종 작성`, state: status.running.text };
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
      { label: "요구 연결", nodes: nodes.filter((node) => node.kind === "requirements") },
      { label: "파트 계획", nodes: nodes.filter((node) => node.kind === "part_plan") },
      { label: "섹션 작성", nodes: nodes.filter((node) => node.kind === "section") },
      { label: "파트 조립", nodes: nodes.filter((node) => node.kind === "part") },
      { label: "파트 편집", nodes: nodes.filter((node) => node.kind === "part_edit") },
      { label: "파트 최종 작성", nodes: nodes.filter((node) => node.kind === "part_author") }
    ].filter((phase) => phase.nodes.length > 0);
  }

  function finalEditClosingNodes(nodes) {
    const staged = nodes.filter((node) => node.kind === "final_assembly" || node.kind === "final_write" || node.kind === "reader_edit" || node.kind === "style_edit" || node.kind === "corrective_gate");
    const artifacts = nodes.filter((node) => node.id === "artifact" || node.kind === "artifact");
    if (staged.length) return [...staged, ...artifacts];
    return nodes.filter((node) => node.id === "final" || node.id === "artifact" || node.kind === "final" || node.kind === "artifact");
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


  reports.pipelineCore = { status, escapeHTML, retryRequestID, nodeLabel, nodeDescription, formatClock, formatDuration, nodeTiming, liveTimingAttributes, planNode, hasPlannedContent, stageName, currentStage, renderAccessibleNode, reportPhases, finalEditClosingNodes, phaseSummary, renderAccessiblePhase };
})(window);

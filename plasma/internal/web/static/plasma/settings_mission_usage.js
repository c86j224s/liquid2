(function (Plasma) {
  "use strict";

  const settings = Plasma.settings;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const api = Plasma.transport.api;

  const numberFormat = new Intl.NumberFormat("ko-KR");

  function formatTokens(value) {
    return numberFormat.format(Number(value) || 0);
  }

  function setUsageBadge(text, tone = "muted") {
    const badge = $("missionUsageStatusBadge");
    if (!badge) return;
    badge.textContent = text;
    badge.className = `badge ${tone}`.trim();
    badge.classList.toggle("hidden", !text);
  }

  function renderMissionUsage() {
    const total = $("missionUsageTotal");
    const content = $("missionUsageContent");
    const refresh = $("missionUsageRefresh");
    if (!total || !content || !refresh) return;
    refresh.disabled = state.missionUsageLoading || !state.missionId;

    if (!state.missionId) {
      total.textContent = "—";
      setUsageBadge("미션 선택 필요");
      content.innerHTML = `<p class="source-helper">미션을 선택하면 사용량을 확인할 수 있습니다.</p>`;
      return;
    }
    if (state.missionUsageLoading || state.missionUsageMissionId !== state.missionId) {
      total.textContent = "—";
      setUsageBadge("집계 중");
      content.innerHTML = `<p class="source-helper">장부의 토큰 기록을 교정 집계하고 있습니다.</p>`;
      return;
    }
    if (state.missionUsageError) {
      total.textContent = "—";
      setUsageBadge("불러오기 실패", "warn");
      content.innerHTML = `<p class="source-helper warn">${escapeHTML(state.missionUsageError)}</p>`;
      return;
    }

    const usage = state.missionUsage || {};
    total.textContent = formatTokens(usage.total_tokens);
    if (!Number(usage.usage_record_count)) {
      setUsageBadge("사용량 없음");
      content.innerHTML = `<p class="source-helper">이 미션에는 아직 토큰 사용량 기록이 없습니다.</p>`;
      return;
    }
    setUsageBadge(usage.usage_partial ? "부분 집계" : "", usage.usage_partial ? "warn" : "");
    content.innerHTML = usageHTML(usage);
  }

  function usageHTML(usage) {
    const available = Number(usage.usage_available_count) || 0;
    const records = Number(usage.usage_record_count) || 0;
    const unavailable = Number(usage.usage_unavailable_count) || 0;
    const input = Number(usage.input_tokens) || 0;
    const cached = Number(usage.cached_input_tokens) || 0;
    const cacheRate = input > 0 ? `${((cached / input) * 100).toFixed(1)}%` : "0.0%";
    const categories = Array.isArray(usage.categories) ? usage.categories : [];
    const workflowRuns = Array.isArray(usage.workflow_runs) ? usage.workflow_runs : [];
    const maxCategory = Math.max(1, ...categories.map((item) => Number(item.total_tokens) || 0));
    const partialNote = usage.usage_partial
      ? `<p class="mission-usage-warning">정확히 합산할 수 없는 기록 ${formatTokens(unavailable)}건과 카운터 재시작 ${formatTokens(usage.counter_reset_count)}건을 제외하거나 새 기준점으로 반영했습니다.</p>`
      : "";
    return `
      ${partialNote}
      <div class="mission-usage-metrics">
        ${metricHTML("입력", usage.input_tokens)}
        ${metricHTML("캐시 입력", usage.cached_input_tokens, cacheRate)}
        ${metricHTML("비캐시 입력", usage.uncached_input_tokens)}
        ${metricHTML("출력", usage.output_tokens)}
        ${metricHTML("추론 출력", usage.reasoning_output_tokens)}
        ${metricHTML("실패 호출", usage.failed_total_tokens, `${formatTokens(usage.failed_call_count)}회`)}
      </div>
      <div class="mission-usage-meta">
        <span>기록 ${formatTokens(available)}/${formatTokens(records)}</span>
        <span>세션 ${formatTokens(usage.session_count)}</span>
        <span>교정 기준 ${escapeHTML(usage.aggregation_version || "mission_usage.v2")}</span>
      </div>
      <div class="mission-usage-percentiles" aria-label="호출당 토큰 분포">
        ${metricHTML("호출당 p50", usage.per_call?.p50)}
        ${metricHTML("호출당 p90", usage.per_call?.p90)}
        ${metricHTML("호출당 최대", usage.per_call?.max)}
      </div>
      <div class="mission-usage-breakdown">
        <h3>기능별 사용량</h3>
        ${categories.length ? categories.map((item) => categoryHTML(item, maxCategory)).join("") : `<p class="source-helper">분류할 수 있는 사용량이 없습니다.</p>`}
      </div>
      ${workflowRunsHTML(workflowRuns)}`;
  }

  function metricHTML(label, value, note = "토큰") {
    return `<div class="mission-usage-metric"><span>${escapeHTML(label)}</span><strong>${formatTokens(value)}</strong><small>${escapeHTML(note)}</small></div>`;
  }

  function categoryHTML(item, maxCategory) {
    const tokens = Number(item.total_tokens) || 0;
    const width = Math.max(2, Math.round((tokens / maxCategory) * 100));
    return `<div class="mission-usage-category">
      <div class="mission-usage-category-copy">
        <span>${escapeHTML(item.label || item.key || "기타")}</span>
        <span>${formatTokens(item.call_count)}회 · ${formatTokens(tokens)}</span>
      </div>
      <div class="mission-usage-bar" aria-hidden="true"><span style="width:${width}%"></span></div>
    </div>`;
  }

  function workflowRunsHTML(runs) {
    if (!runs.length) return "";
    return `<details class="mission-usage-runs">
      <summary>
        <span>자율 조사 실행별</span>
        <span>${formatTokens(runs.length)}건</span>
      </summary>
      <div class="mission-usage-run-list">
        ${runs.map((run, index) => workflowRunHTML(run, index)).join("")}
      </div>
    </details>`;
  }

  function workflowRunHTML(run, index) {
    const calls = Number(run.call_count) || 0;
    const average = calls > 0 ? Math.round((Number(run.total_tokens) || 0) / calls) : 0;
    return `<section class="mission-usage-run">
      <div class="mission-usage-run-head">
        <strong>실행 ${formatTokens(index + 1)}</strong>
        <span>${formatTokens(calls)}회 · ${formatTokens(run.total_tokens)} 토큰</span>
      </div>
      <div class="mission-usage-meta">
        <span>모델 ${escapeHTML(descriptorLabel(run.agent_model))}</span>
        <span>추론 ${escapeHTML(descriptorLabel(run.reasoning_effort))}</span>
        <span>세션 ${formatTokens(run.session_count)}</span>
        <span>재개 호출 ${formatTokens(run.resumed_call_count)}</span>
      </div>
      <div class="mission-usage-run-metrics">
        ${metricHTML("호출당 평균", average)}
        ${metricHTML("입력", run.input_tokens)}
        ${metricHTML("캐시 입력", run.cached_input_tokens)}
        ${metricHTML("비캐시 입력", run.uncached_input_tokens)}
        ${metricHTML("출력", run.output_tokens)}
        ${metricHTML("추론 출력", run.reasoning_output_tokens)}
      </div>
    </section>`;
  }

  function descriptorLabel(value) {
    const descriptor = String(value || "").trim();
    if (!descriptor) return "미기록";
    return descriptor === "mixed" ? "혼합" : descriptor;
  }

  async function loadMissionUsage(owner = Plasma.mission.captureMissionSelection()) {
    if (!owner.missionId) {
      state.missionUsage = null;
      state.missionUsageMissionId = "";
      state.missionUsageLoading = false;
      state.missionUsageError = "";
      renderMissionUsage();
      return;
    }
    state.missionUsageMissionId = owner.missionId;
    state.missionUsageLoading = true;
    state.missionUsageError = "";
    renderMissionUsage();
    try {
      const response = await api(`/api/missions/${encodeURIComponent(owner.missionId)}/usage`);
      if (!Plasma.mission.ownsMissionSelection(owner)) return;
      state.missionUsage = response.usage || null;
    } catch (err) {
      if (!Plasma.mission.ownsMissionSelection(owner)) return;
      state.missionUsage = null;
      state.missionUsageError = err?.message || "토큰 사용량을 불러오지 못했습니다.";
    } finally {
      if (Plasma.mission.ownsMissionSelection(owner)) {
        state.missionUsageLoading = false;
        renderMissionUsage();
      }
    }
  }

  Object.assign(settings, { formatTokens, loadMissionUsage, renderMissionUsage });
})(window.Plasma);

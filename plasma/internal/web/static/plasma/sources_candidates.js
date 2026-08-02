(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const escapeAttr = Plasma.dom.escapeAttr;
  const empty = Plasma.ui.empty;
  const updateCountChip = Plasma.ui.updateCountChip;
  const setSectionEmpty = Plasma.ui.setSectionEmpty;
  const renderTabs = (...args) => sources.dependency("renderTabs")(...args);
  const normalizeSourceURL = sources.normalizeSourceURL;
  const acceptedSourceCandidateKeys = (...args) => sources.acceptedSourceCandidateKeys(...args);
  const sourceCandidateAccepted = (...args) => sources.sourceCandidateAccepted(...args);
  const sourceCandidateDecisions = (...args) => sources.sourceCandidateDecisions(...args);
  const sourceCandidatesFromEvents = (...args) => sources.sourceCandidatesFromEvents(...args);
  const sourceCandidateStagingLabel = (...args) => sources.sourceCandidateStagingLabel(...args);
  const pruneSelectedSourceCandidates = (...args) => sources.pruneSelectedSourceCandidates(...args);
  const updateSourceCandidateBulkBar = (...args) => sources.updateSourceCandidateBulkBar(...args);

  function renderSourceCandidates(events, sources) {
    const existing = acceptedSourceCandidateKeys(sources);
    const decisions = sourceCandidateDecisions(events);
    const candidates = sourceCandidatesFromEvents(events).filter((candidate) => {
      const normalized = normalizeSourceURL(candidate.url);
      return normalized && !sourceCandidateAccepted(existing, normalized) && decisions.get(normalized)?.state !== "rejected";
    });
    updateCountChip("sourceCandidateCount", candidates.length);
    updateSourceCandidateIndicators(candidates.length);
    const candidateHeader = document.querySelector(".source-candidate-header");
    const candidateDivider = document.querySelector(".source-candidate-divider");
    setSectionEmpty(candidateHeader, candidates.length === 0);
    setSectionEmpty(candidateDivider, candidates.length === 0);
    setSectionEmpty($("sourceCandidateList"), candidates.length === 0);
    $("sourceCandidateList").innerHTML = candidates.length ? candidates.map((candidate) => {
      const normalized = normalizeSourceURL(candidate.url);
      const busy = state.sourceCandidateBusy.has(normalized);
      const selected = state.selectedSourceCandidates.has(normalized);
      const browserRender = candidate.staging?.browserRenderCandidate?.candidate === true ? candidate.staging.browserRenderCandidate : null;
      return `
      <div class="item${selected ? " selected" : ""}">
        <input type="checkbox" class="item-select" data-select-source-url="${escapeAttr(normalized)}" data-source-candidate-title="${escapeAttr(candidate.title || "")}" aria-label="후보 선택" ${selected ? "checked" : ""}>
        <div class="item-title">${escapeHTML(candidate.title || candidate.url)}</div>
        <div class="item-meta"><a href="${escapeAttr(candidate.url)}" target="_blank" rel="noopener noreferrer">${escapeHTML(candidate.url)}</a></div>
        ${candidate.staging ? `<div class="item-meta">${sourceCandidateStagingLabel(candidate.staging)}</div>` : ""}
        ${browserRender ? `<div class="item-meta source-candidate-diagnostic-note"><strong>진단</strong> ${escapeHTML(browserRender.reason || "브라우저 렌더링 검증 후보입니다.")}</div>` : ""}
        <div class="item-meta source-candidate-reason"><strong>채택 의견</strong> ${escapeHTML(candidate.reason)}</div>
        <div class="item-actions">
          <button type="button" class="secondary" data-detail-title="소스 후보 상세" data-detail-json="${escapeAttr(JSON.stringify(candidate))}">자세히</button>
          <button type="button" data-add-source-url="${escapeAttr(candidate.url)}" data-source-candidate-title="${escapeAttr(candidate.title || "")}" ${busy ? "disabled" : ""}>${busy ? "처리 중" : "소스로 추가"}</button>
          <button type="button" class="danger" data-reject-source-url="${escapeAttr(candidate.url)}" ${busy ? "disabled" : ""}>기각</button>
        </div>
      </div>
    `;
    }).join("") : empty("채택 의견이 있는 소스 후보 없음");
    pruneSelectedSourceCandidates(candidates);
    updateSourceCandidateBulkBar();
    const rejected = sourceCandidatesFromEvents(events).filter((candidate) => {
      const normalized = normalizeSourceURL(candidate.url);
      return normalized && !sourceCandidateAccepted(existing, normalized) && decisions.get(normalized)?.state === "rejected";
    });
    updateCountChip("rejectedSourceCandidateCount", rejected.length);
    setSectionEmpty($("rejectedSourceCandidateDetails"), rejected.length === 0);
    $("rejectedSourceCandidateList").innerHTML = rejected.length ? rejected.map((candidate) => {
      const decision = decisions.get(normalizeSourceURL(candidate.url)) || {};
      const busy = state.sourceCandidateBusy.has(normalizeSourceURL(candidate.url));
      return `
      <div class="item">
        <div class="item-title">${escapeHTML(candidate.title || candidate.url)}</div>
        <div class="item-meta"><a href="${escapeAttr(candidate.url)}" target="_blank" rel="noopener noreferrer">${escapeHTML(candidate.url)}</a></div>
        ${candidate.staging ? `<div class="item-meta">${sourceCandidateStagingLabel(candidate.staging)}</div>` : ""}
        <div class="item-meta">${escapeHTML(decision.reason || "사용자가 기각했습니다.")}</div>
        <div class="item-actions">
          <button type="button" class="secondary" data-detail-title="기각한 소스 후보 상세" data-detail-json="${escapeAttr(JSON.stringify({ candidate, decision }))}">자세히</button>
          <button type="button" data-restore-source-url="${escapeAttr(candidate.url)}" ${busy ? "disabled" : ""}>다시 검토</button>
        </div>
      </div>
    `;
    }).join("") : empty("기각한 후보 없음");
  }

  function updateSourceCandidateIndicators(count) {
    updateCountChip("sourceTabCandidateCount", count);
    const metric = $("candidateCount");
    if (metric) {
      metric.textContent = `소스 후보 ${count}개`;
      metric.classList.toggle("collapsed-empty", count === 0);
      metric.classList.remove("hidden");
    }
    const notice = $("sourceCandidateNotice");
    if (notice) notice.classList.toggle("hidden", count === 0);
    const noticeCount = $("sourceCandidateNoticeCount");
    if (noticeCount) noticeCount.textContent = String(count);
  }

  function openSourceCandidatesTab() {
    state.activeTab = "sources";
    location.hash = "sources";
    renderTabs();
    $("sourceCandidateList")?.scrollIntoView({ block: "start", behavior: "smooth" });
  }

  function refreshSourceCandidates() {
    if (!state.detail) return;
    renderSourceCandidates(state.detail.events || [], state.detail.sources || []);
  }

  Object.assign(sources, {
    renderSourceCandidates,
    updateSourceCandidateIndicators,
    openSourceCandidatesTab,
    refreshSourceCandidates
  });
})(window.Plasma);

(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const escapeAttr = Plasma.dom.escapeAttr;
  const timeShort = Plasma.dom.timeShort;
  const knowledge = Plasma.knowledge;

  function showClaimConfidenceDetail(claimID) {
    const records = state.detail?.records || {};
    const claim = (records.claims || []).find((record) => record.claim_id === claimID);
    const view = (records.claim_confidence || []).find((item) => item.claim_id === claimID) || knowledge.initialConfidenceView(claim || { claim_id: claimID });
    const current = view.current_confidence || {};
    const initial = view.initial_confidence || {};
    const risks = current.open_risks || [];
    const history = view.history || [];
    const latestHistory = history.length ? history[history.length - 1] : null;
    const data = { claim, confidence: view };
    state.detailText = JSON.stringify(data, null, 2);
    $("detailTitle").textContent = "주장 신뢰도 상세";
    $("detailBody").innerHTML = `
    <section class="detail-section">
      <h3>주장</h3>
      <p>${escapeHTML(claim?.text || claimID)}</p>
      <div class="detail-meta">${escapeHTML(claimID)}</div>
    </section>
    <section class="detail-grid">
      <div class="detail-box">
        <h3>현재 신뢰도</h3>
        ${knowledge.confidenceBadge(view)}
        <p>${escapeHTML(current.rationale || "현재 신뢰도 판단 사유가 없습니다.")}</p>
      </div>
      <div class="detail-box">
        <h3>초기 신뢰도</h3>
        <span class="badge confidence ${escapeAttr(initial.level || "unknown")}">${escapeHTML(knowledge.confidenceLabel(initial.level || "unknown"))}</span>
        <p>${escapeHTML(initial.rationale || "초기 신뢰도 판단 사유가 없습니다.")}</p>
      </div>
    </section>
    <section class="detail-section">
      <h3>판단 근거</h3>
      ${detailChips(latestHistory?.basis_evidence_ids || [])}
    </section>
    <section class="detail-section">
      <h3>열린 위험</h3>
      ${risks.length ? `<ul>${risks.map((risk) => `<li>${escapeHTML(risk)}</li>`).join("")}</ul>` : `<p class="detail-meta">열린 위험 없음</p>`}
    </section>
    <section class="detail-section">
      <h3>변경 이력</h3>
      ${history.length ? history.map(renderConfidenceHistoryEntry).join("") : `<p class="detail-meta">아직 변경 이력이 없습니다.</p>`}
    </section>
  `;
    Plasma.ui.openDetailModal();
  }

  function renderConfidenceHistoryEntry(entry) {
    return `
    <div class="history-entry">
      <div class="confidence-line">
        <span class="badge confidence ${escapeAttr(entry.level || "unknown")}">${escapeHTML(knowledge.confidenceLabel(entry.level || "unknown"))}${escapeHTML(knowledge.directionGlyph(entry.direction))}</span>
        <span class="item-meta">${escapeHTML(timeShort(entry.created_at))} / ${escapeHTML(entry.origin || "unknown")} / ${escapeHTML(entry.event_id || "")}</span>
      </div>
      <div>${escapeHTML(entry.rationale || "변경 사유 없음")}</div>
      ${detailChips(entry.basis_evidence_ids || [])}
    </div>
  `;
  }

  function detailChips(values) {
    if (!values.length) return `<p class="detail-meta">연결된 근거 없음</p>`;
    return `<div class="chip-row">${values.map((value) => `<span class="badge muted">${escapeHTML(value)}</span>`).join("")}</div>`;
  }

  function onDetailButtonClick(event) {
    const confidenceButton = event.target.closest("[data-confidence-claim-id]");
    if (!confidenceButton) return Plasma.ui.onDetailButtonClick(event);
    showClaimConfidenceDetail(confidenceButton.dataset.confidenceClaimId);
    return true;
  }

  Object.assign(knowledge, { detailChips, onDetailButtonClick, renderConfidenceHistoryEntry, showClaimConfidenceDetail });
})(window.Plasma);

(function (Plasma) {
  "use strict";

  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const escapeAttr = Plasma.dom.escapeAttr;
  const empty = Plasma.ui.empty;
  const updateCountChip = Plasma.ui.updateCountChip;
  const setSectionEmpty = Plasma.ui.setSectionEmpty;
  const knowledge = Plasma.knowledge;

  function renderSavedEvidence(saved) {
    const n = saved.length;
    updateCountChip("savedCountSummary", n);
    setSectionEmpty($("savedListDetails"), n === 0);
    $("savedList").innerHTML = n ? saved.map((record) => `
    <div class="item">
      <div>${escapeHTML(record.summary)}</div>
      <div class="item-meta">${escapeHTML(knowledge.evidenceTypeLabel(record.evidence_type))} / ${escapeHTML(record.evidence_id)}</div>
      <div class="item-actions">
        <button type="button" class="secondary" data-detail-title="승인된 근거 상세" data-detail-json="${escapeAttr(JSON.stringify(record))}">자세히</button>
      </div>
    </div>
  `).join("") : empty("승인된 근거 없음");
  }

  function renderClaimConfidenceChanges(confidenceViews, claims) {
    const claimByID = new Map((claims || []).map((record) => [record.claim_id, record]));
    const changed = (confidenceViews || []).filter((view) => (view.history || []).length > 0);
    updateCountChip("claimConfidenceCount", changed.length);
    setSectionEmpty($("claimConfidenceDetails"), changed.length === 0);
    $("claimConfidenceList").innerHTML = changed.length ? changed.map((view) => {
      const claim = claimByID.get(view.claim_id) || {};
      const current = view.current_confidence || {};
      return `
      <div class="item confidence-item">
        <div class="confidence-line">
          ${knowledge.confidenceBadge(view)}
          <span class="item-title">${escapeHTML(claim.text || view.claim_id)}</span>
        </div>
        <div class="item-meta clamp-line">${escapeHTML(current.rationale || "신뢰도 변경 사유 없음")}</div>
        <div class="item-actions">
          <button type="button" class="secondary" data-confidence-claim-id="${escapeAttr(view.claim_id)}">자세히</button>
        </div>
      </div>
    `;
    }).join("") : empty("신뢰도 변화 없음");
  }

  function renderSavedClaims(saved, confidenceViews = []) {
    const n = saved.length;
    updateCountChip("savedClaimListCount", n);
    setSectionEmpty($("savedClaimListDetails"), n === 0);
    const confidenceByClaim = new Map((confidenceViews || []).map((view) => [view.claim_id, view]));
    $("savedClaimList").innerHTML = n ? saved.map((record) => {
      const confidence = confidenceByClaim.get(record.claim_id) || knowledge.initialConfidenceView(record);
      const rationale = confidence.current_confidence?.rationale || record.confidence?.rationale || "";
      return `
    <div class="item">
      <div class="item-title">${escapeHTML(record.text)}</div>
      <div class="confidence-line">
        ${knowledge.confidenceBadge(confidence)}
        <span class="item-meta">${escapeHTML(record.claim_type || "claim")} / ${escapeHTML(record.claim_id)}</span>
      </div>
      ${rationale ? `<div class="item-meta clamp-line">${escapeHTML(rationale)}</div>` : ""}
      <div class="item-meta">근거 ${escapeHTML((record.supporting_evidence_ids || []).join(", ") || "없음")}</div>
      <div class="item-actions">
        <button type="button" class="secondary" data-detail-title="승인된 주장 상세" data-detail-json="${escapeAttr(JSON.stringify(record))}">자세히</button>
        <button type="button" class="secondary" data-confidence-claim-id="${escapeAttr(record.claim_id)}">신뢰도</button>
      </div>
    </div>
  `;
    }).join("") : empty("승인된 주장 없음");
  }

  Object.assign(knowledge, { renderClaimConfidenceChanges, renderSavedClaims, renderSavedEvidence });
})(window.Plasma);

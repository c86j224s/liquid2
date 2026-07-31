(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const escapeAttr = Plasma.dom.escapeAttr;
  const empty = Plasma.ui.empty;
  const updateCountChip = Plasma.ui.updateCountChip;
  const proposals = Plasma.proposals;

  function renderProposalExtractionStatus(status) {
    if (!status || typeof status !== "object") return "";
    if (status.error) return `<div class="turn-note warn">후보 생성 실패: ${escapeHTML(status.error)}</div>`;
    if (status.created_proposals) return `<div class="turn-note">소스 기반 후보가 검토 대기 목록에 추가되었습니다.</div>`;
    if (status.attempted) return `<div class="turn-note muted">소스 기반으로 저장할 후보를 찾지 못했습니다.</div>`;
    switch (status.reason) {
      case "no_sources": return `<div class="turn-note muted">저장된 소스가 없어 근거 후보를 만들지 않았습니다.</div>`;
      case "explicit_mode": return `<div class="turn-note muted">명시 요청 모드라 자동 후보 생성을 건너뛰었습니다.</div>`;
      case "no_agent_session": return `<div class="turn-note warn">에이전트 세션 ID가 없어 후보 생성을 건너뛰었습니다.</div>`;
      default: return "";
    }
  }

  function renderProposals(items, records) {
    const evidenceByID = new Map((records.evidence || []).map((record) => [record.evidence_id, record]));
    const claimByID = new Map((records.claims || []).map((record) => [record.claim_id, record]));
    const questionByID = new Map((records.questions || []).map((record) => [record.question_id, record]));
    const optionByID = new Map((records.options || []).map((record) => [record.option_id, record]));
    const pending = items.filter((proposal) => proposal.state === "pending_review");
    updateCountChip("proposalListCount", pending.length);
    $("proposalList").innerHTML = pending.length ? pending.map((proposal) => proposalCard(proposal, evidenceByID, claimByID, questionByID, optionByID)).join("") : empty("지금은 검토 대기 후보가 없습니다.");
    proposals.pruneSelectedProposals(pending);
    proposals.updateProposalBulkBar();
  }

  function proposalCard(proposal, evidenceByID, claimByID, questionByID, optionByID) {
    const refs = proposal.object_refs || [];
    const objects = refs.map((ref) => proposalObjectDetail(ref, evidenceByID, claimByID, questionByID, optionByID));
    const text = objects.map((object) => object.label).join("\n");
    const title = proposal.title || proposalKindLabel(refs) || proposal.proposal_id;
    const objectMeta = objects.map((object) => object.meta).filter(Boolean).join(" · ");
    const detail = { proposal, objects };
    const selected = state.selectedProposals.has(proposal.proposal_id);
    return `
      <div class="item${selected ? " selected" : ""}">
        <input type="checkbox" class="item-select" data-select-proposal-id="${escapeAttr(proposal.proposal_id)}" aria-label="후보 선택" ${selected ? "checked" : ""}>
        <div class="item-title">${escapeHTML(title)}</div>
        <div class="item-meta">${escapeHTML(proposalKindLabel(refs) || "저장 후보")} / ${escapeHTML(proposal.proposal_id)}${objectMeta ? ` / ${escapeHTML(objectMeta)}` : ""}</div>
        <div>${escapeHTML(text)}</div>
        <div class="item-actions">
          <button type="button" class="secondary" data-detail-title="후보 상세" data-detail-json="${escapeAttr(JSON.stringify(detail))}">자세히</button>
          <button type="button" data-proposal-id="${escapeAttr(proposal.proposal_id)}" data-action="approve">승인</button>
          <button type="button" class="danger" data-proposal-id="${escapeAttr(proposal.proposal_id)}" data-action="reject">기각</button>
        </div>
      </div>
    `;
  }

  function proposalObjectDetail(ref, evidenceByID, claimByID, questionByID, optionByID) {
    if (ref.object_kind === "evidence_record" && evidenceByID.has(ref.object_id)) {
      const record = evidenceByID.get(ref.object_id);
      return { ref, record, label: `근거: ${record.summary || ref.object_id}`, meta: `근거 신호: ${Plasma.knowledge.evidenceTypeLabel(record.evidence_type)}` };
    }
    if (ref.object_kind === "claim_record" && claimByID.has(ref.object_id)) return objectDetail(ref, claimByID.get(ref.object_id), "주장", "text");
    if (ref.object_kind === "question_record" && questionByID.has(ref.object_id)) return objectDetail(ref, questionByID.get(ref.object_id), "질문", "text");
    if (ref.object_kind === "option_record" && optionByID.has(ref.object_id)) return objectDetail(ref, optionByID.get(ref.object_id), "선택지", "title");
    return { ref, record: null, label: `${ref.object_kind}: ${ref.object_id}`, meta: "" };
  }

  function objectDetail(ref, record, label, field) {
    return { ref, record, label: `${label}: ${record[field] || ref.object_id}`, meta: "" };
  }

  function proposalKindLabel(refs) {
    const kinds = new Set((refs || []).map((ref) => ref.object_kind));
    if (kinds.size > 1) return "복합 후보";
    if (kinds.has("evidence_record")) return "근거 후보";
    if (kinds.has("claim_record")) return "주장 후보";
    if (kinds.has("question_record")) return "질문 후보";
    if (kinds.has("option_record")) return "선택지 후보";
    return "저장 후보";
  }

  Object.assign(proposals, { proposalKindLabel, proposalObjectDetail, renderProposalExtractionStatus, renderProposals });
})(window.Plasma);

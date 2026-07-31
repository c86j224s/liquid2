(function knowledgeNamespace(root) {
  "use strict";

  const Plasma = root.Plasma;
  const knowledge = Plasma.knowledge || (Plasma.knowledge = {});
  const EVIDENCE_TYPE_LABELS = {
    fact: "사실",
    observation: "관찰",
    quote: "인용",
    statistic: "통계",
    table_row: "표 항목",
    interpretation: "해석/평가",
    reaction: "반응",
    rumor: "루머/미확정설",
    controversy: "논쟁 축",
    market_signal: "시장 신호",
    code: "코드",
    formula: "수식",
    benchmark: "벤치마크",
    open_question: "열린 질문",
    user_assertion: "사용자 진술"
  };

  function evidenceTypeLabel(type) {
    return EVIDENCE_TYPE_LABELS[type] || type || "근거";
  }

  function approvedEvidence(proposals, evidence) {
    const ids = new Set();
    for (const proposal of proposals) {
      if (proposal.state !== "approved" && proposal.state !== "partially_approved") continue;
      for (const ref of proposal.object_refs || []) {
        if (ref.object_kind === "evidence_record") ids.add(ref.object_id);
      }
    }
    return evidence.filter((record) => record.state === "approved" || ids.has(record.evidence_id));
  }

  function approvedClaims(proposals, claims) {
    const ids = new Set();
    for (const proposal of proposals) {
      if (proposal.state !== "approved" && proposal.state !== "partially_approved") continue;
      for (const ref of proposal.object_refs || []) {
        if (ref.object_kind === "claim_record") ids.add(ref.object_id);
      }
    }
    return claims.filter((record) => record.state === "approved" || ids.has(record.claim_id));
  }

  function initialConfidenceView(record) {
    return {
      claim_id: record.claim_id,
      initial_confidence: record.confidence || { level: "unknown" },
      current_confidence: record.confidence || { level: "unknown" },
      direction: "initial",
      history: []
    };
  }

  function confidenceLabel(level) {
    switch (level) {
      case "high": return "신뢰 높음";
      case "medium": return "신뢰 보통";
      case "low": return "신뢰 낮음";
      default: return "신뢰 미정";
    }
  }

  function directionGlyph(direction) {
    switch (direction) {
      case "up": return " ↑";
      case "down": return " ↓";
      case "unchanged": return " ·";
      default: return "";
    }
  }

  function confidenceBadge(view) {
    const current = view.current_confidence || {};
    const level = current.level || "unknown";
    const needsVerification = current.needs_verification ? " · 확인 필요" : "";
    return `<span class="badge confidence ${Plasma.dom.escapeAttr(level)}">${Plasma.dom.escapeHTML(confidenceLabel(level))}${Plasma.dom.escapeHTML(directionGlyph(view.direction || "initial"))}${Plasma.dom.escapeHTML(needsVerification)}</span>`;
  }

  Object.assign(knowledge, {
    EVIDENCE_TYPE_LABELS,
    approvedClaims,
    approvedEvidence,
    confidenceBadge,
    confidenceLabel,
    directionGlyph,
    evidenceTypeLabel,
    initialConfidenceView
  });
})(window);

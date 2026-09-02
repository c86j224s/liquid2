(function reportsILArtifactCards(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const escapeHTML = root.Plasma.dom.escapeHTML;
  const escapeAttr = root.Plasma.dom.escapeAttr;
  const formatBytes = root.Plasma.dom.formatBytes;

  const artifactOrder = ["markdown", "html", "pdf", "narrative", "semantic_il", "manifest"];
  const legacyArtifactOrder = ["markdown", "html", "pdf", "narrative", "semantic_il", "flow_attestation", "manifest"];
  const maxImageArtifacts = 3;
  const imageMediaTypes = new Set(["image/png", "image/jpeg", "image/gif"]);

  function validImageFilename(filename, mediaType) {
    const value = String(filename || "").trim();
    if (!value || value.includes("/") || value.includes("\\")) return false;
    const lower = value.toLowerCase();
    if (mediaType === "image/png") return lower.endsWith(".png");
    if (mediaType === "image/jpeg") return lower.endsWith(".jpg") || lower.endsWith(".jpeg");
    if (mediaType === "image/gif") return lower.endsWith(".gif");
    return false;
  }

  function reportILSourceSelection(bundle) {
    if (!Object.prototype.hasOwnProperty.call(bundle, "source_selection")) return undefined;
    const summary = bundle.source_selection;
    const countKeys = [
      "accepted_sources",
      "usable_sources",
      "selected_sources",
      "excluded_unusable_sources",
      "excluded_budget_sources",
    ];
    const hasSupplement = Object.prototype.hasOwnProperty.call(summary || {}, "supplemental_sources");
    const expectedKeys = ["applied", ...countKeys, ...(hasSupplement ? ["supplemental_sources"] : [])];
    const supplementalSources = hasSupplement ? summary.supplemental_sources : 0;
    if (
      !summary ||
      Array.isArray(summary) ||
      typeof summary !== "object" ||
      typeof summary.applied !== "boolean" ||
      Object.keys(summary).length !== expectedKeys.length ||
      expectedKeys.some((key) => !Object.prototype.hasOwnProperty.call(summary, key)) ||
      countKeys.some((key) => !Number.isSafeInteger(summary[key])) ||
      !Number.isSafeInteger(supplementalSources) ||
      summary.accepted_sources < 1 ||
      summary.usable_sources < 1 ||
      summary.selected_sources < 1 ||
      supplementalSources < 0 ||
      summary.excluded_unusable_sources < 0 ||
      summary.excluded_budget_sources < 0 ||
      summary.accepted_sources !== summary.usable_sources + summary.excluded_unusable_sources ||
      summary.usable_sources !== summary.selected_sources + supplementalSources + summary.excluded_budget_sources ||
      summary.applied !== (supplementalSources > 0 || summary.excluded_unusable_sources + summary.excluded_budget_sources > 0)
    ) {
      return null;
    }
    return { ...summary, supplemental_sources: supplementalSources };
  }

  function reportILBundle(payload = {}) {
    const bundle = payload.artifact_bundle || {};
    const entries = Array.isArray(bundle.artifacts) ? bundle.artifacts : [];
    const coreEntries = entries.filter((entry) => entry.kind !== "image");
    const imageEntries = entries.filter((entry) => entry.kind === "image");
    const byKind = new Map(coreEntries.map((entry) => [entry.kind, entry]));
    const markdown = byKind.get("markdown") || {};
    const sourceSelection = reportILSourceSelection(bundle);
    const order = byKind.has("flow_attestation") ? legacyArtifactOrder : artifactOrder;
    const imageAssetIDs = new Set(imageEntries.map((entry) => entry.asset_id));
    if (
      bundle.pipeline_family !== reports.REPORT_IL_PIPELINE_FAMILY ||
      coreEntries.length !== order.length ||
      byKind.size !== order.length ||
      order.some((kind) => !byKind.has(kind)) ||
      imageEntries.length > maxImageArtifacts ||
      imageAssetIDs.size !== imageEntries.length ||
      imageEntries.some((entry) => !entry.asset_id || entry.role !== "asset" || !imageMediaTypes.has(entry.media_type) || !validImageFilename(entry.filename, entry.media_type)) ||
      !payload.artifact_id ||
      payload.artifact_id !== bundle.markdown_artifact_id ||
      payload.artifact_id !== markdown.artifact_id ||
      sourceSelection === null
    ) {
      return null;
    }
    return { ...bundle, sourceSelection, entries: [...order.map((kind) => byKind.get(kind)), ...imageEntries] };
  }

  function reportILArtifactAction(entry) {
    const artifactID = escapeAttr(entry.artifact_id || "");
    const label = escapeHTML(reports.REPORT_IL_ARTIFACT_LABELS[entry.kind] || entry.kind || "artifact");
    const view = entry.kind === "markdown"
      ? `<button type="button" data-report-artifact-id="${artifactID}" data-action="view-artifact">${label} 보기</button>`
      : entry.kind === "html"
        ? `<button type="button" data-report-artifact-id="${artifactID}" data-action="view-stored-html-artifact">${label} 보기</button>`
        : entry.kind === "pdf"
          ? ""
          : `<button type="button" data-report-artifact-id="${artifactID}" data-action="view-text-artifact" data-report-artifact-label="${escapeAttr(label)}">${label} 보기</button>`;
    return `${view}<button type="button" class="secondary" data-report-artifact-id="${artifactID}" data-action="download-artifact">${label} 받기</button>`;
  }

  function reportILLineageRows(bundle) {
    return bundle.entries.map((entry) => {
      const bytes = formatBytes(Number(entry.byte_size || 0)) || `${Number(entry.byte_size || 0)} B`;
      return `<div class="report-il-lineage-item">
        <div><strong>${escapeHTML(reports.REPORT_IL_ARTIFACT_LABELS[entry.kind] || entry.kind)}</strong><span>${escapeHTML(entry.filename || "")}</span></div>
        <span class="badge muted">${escapeHTML(entry.role || "")}</span>
        <span>${escapeHTML(bytes)}</span>
        <code title="SHA-256 ${escapeAttr(entry.sha256 || "")}">${escapeHTML(String(entry.sha256 || "").slice(0, 12))}</code>
      </div>`;
    }).join("");
  }

  function reportILSourceSelectionHTML(summary) {
    if (!summary) return "";
    const excluded = [];
    if (summary.excluded_unusable_sources) excluded.push(`읽기 불가 제외 ${summary.excluded_unusable_sources}`);
    if (summary.excluded_budget_sources) excluded.push(`예산 제외 ${summary.excluded_budget_sources}`);
    return `<div class="report-il-source-selection" aria-label="IL 보고서 소스 사용 범위">
      <div><strong>소스 사용 범위</strong><span>승인 ${summary.accepted_sources} · 사용 가능 ${summary.usable_sources} · 본문 ${summary.selected_sources}${summary.supplemental_sources ? ` · 장문 보충 ${summary.supplemental_sources}` : ""}</span></div>
      ${excluded.length ? `<span>${excluded.join(" · ")}</span>` : `<span>제외 없음 · 승인 소스 전체 사용</span>`}
    </div>`;
  }

  function renderILArtifactCard(key, isLatest, payload, selectedKey) {
    const bundle = reportILBundle(payload);
    if (!bundle) return "";
    const markdown = bundle.entries.find((entry) => entry.kind === "markdown") || {};
    const title = payload.title || "IL 보고서";
    const actions = bundle.entries.map(reportILArtifactAction).join("");
    const artifactCount = bundle.entries.length;
    const modeLabel = payload.report_mode === "long_form" ? "장문 IL 보고서" : "IL 보고서";
    return `<div class="item report-card report-il-card ${isLatest ? "active" : ""} ${key === selectedKey ? "selected" : ""}" data-report-key="${escapeAttr(key)}" data-report-il-bundle-key="${escapeAttr(key)}">
      <div class="item-title report-title-line report-card-toggle"><span>${escapeHTML(title)}</span><span class="chip-row report-chip-row">${isLatest ? `<span class="badge session-new">최신</span>` : `<span class="badge muted">이전</span>`}<span class="badge">${modeLabel}</span><span class="badge muted">출력 ${artifactCount}개</span></span></div>
      <div class="report-card-body">
        <div class="item-meta clamp-line" title="${escapeAttr(markdown.artifact_id || "")}">${escapeHTML(markdown.artifact_id || "")}</div>
        <div class="item-meta">${escapeHTML(payload.text || "IL 보고서가 생성되었습니다.")}</div>
        ${reports.reportGenerationSummaryHTML(payload)}
        ${reportILSourceSelectionHTML(bundle.sourceSelection)}
        <div class="report-il-card-note">같은 원고를 Markdown·HTML·PDF로 만들었으며, 포함된 이미지는 관련 소스에서 가져왔습니다.</div>
        <div class="report-il-output-actions" aria-label="IL 보고서 출력과 생성 기록">${actions}</div>
        <details class="report-il-lineage-details">
          <summary>생성 기록 ${artifactCount}개 보기</summary>
          <div class="report-il-lineage-list">${reportILLineageRows(bundle)}</div>
        </details>
        <div class="item-actions">${reports.reportActionMenu("도구 ▾", `<button type="button" class="secondary" data-detail-title="IL 보고서 생성 상세" data-detail-json="${escapeAttr(JSON.stringify(payload))}">자세히</button><button type="button" class="danger" data-report-artifact-id="${escapeAttr(markdown.artifact_id || "")}" data-action="delete-report-artifact">보고서 묶음 삭제</button>`)}</div>
        ${reports.reportPreviewInlineHTML(key)}
      </div>
    </div>`;
  }

  function renderILArtifactReportSection(artifactCards, selectedKey) {
    const cards = artifactCards.map(({ key, isLatest, payload }) => renderILArtifactCard(key, isLatest, payload, selectedKey)).filter(Boolean);
    return cards.length ? `<div class="list-section-label">IL 보고서 · Markdown / HTML / PDF</div>${cards.join("")}` : "";
  }

  Object.assign(reports, { reportILBundle, renderILArtifactCard, renderILArtifactReportSection });
})(window);

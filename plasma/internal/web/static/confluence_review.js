async function previewConfluenceCandidate(index) {
  if (!requireMission() || state.confluenceBusy) return;
  const candidate = state.confluenceSearchResults[Number(index)];
  if (!candidate) return;
  const searchContext = state.confluenceSearchContext || {};
  const connectionID = searchContext.connection_id || confluenceSelectedConnectionID();
  const cloudID = candidate.CloudID || candidate.cloud_id || searchContext.cloud_id || $("confluenceSiteSelect").value;
  const pageID = confluenceCandidatePageID(candidate);
  const version = Number(candidate.Version || candidate.version || 0);
  if (!connectionID || !cloudID || !pageID) {
    showError(new Error("Confluence 소스를 저장하려면 연결, 사이트, 페이지 정보가 필요합니다."));
    return;
  }
  await previewConfluencePage({ ...candidate, page_id: pageID, cloud_id: cloudID, version }, connectionID, cloudID);
}

async function previewConfluencePage(page, forcedConnectionID = "", forcedCloudID = "") {
  if (!requireMission() || !page || state.confluenceBusy) return;
  const owner = captureMissionSelection();
  const context = state.confluenceBrowseContext || {};
  const connectionID = forcedConnectionID || context.connection_id || confluenceSelectedConnectionID();
  const cloudID = forcedCloudID || page.cloud_id || page.CloudID || context.cloud_id || confluenceSiteCloudID(selectedConfluenceSite());
  const pageID = page.page_id || page.PageID || confluenceCandidatePageID(page);
  const version = Number(page.version || page.Version || 0);
  if (!connectionID || !cloudID || !pageID) {
    showError(new Error("Confluence 후보를 검토하려면 연결, 사이트, 페이지 정보가 필요합니다."));
    return;
  }
  setConfluenceBusy(true);
  try {
    const result = await missionApi(owner, "/sources/confluence/preview", {
      method: "POST",
      body: { connection_id: connectionID, cloud_id: cloudID, page_id: pageID, expected_version: version }
    });
    if (!ownsMissionSelection(owner)) return;
    state.confluencePreview = { result, connection_id: connectionID, cloud_id: cloudID };
    renderConfluencePreview(state.confluencePreview);
    setConfluenceFlowStatus("후보 미리보기를 열었습니다. 내용을 확인한 뒤 소스로 승인하세요.");
    $("confluencePreviewPanel")?.scrollIntoView({ block: "nearest" });
  } catch (err) {
    if (ownsMissionSelection(owner)) showConfluenceError(err);
  } finally {
    if (ownsMissionSelection(owner)) setConfluenceBusy(false);
  }
}

function renderConfluencePreview(preview) {
  const panel = $("confluencePreviewPanel");
  if (!panel) return;
  panel.classList.toggle("hidden", !preview);
  if (!preview) return;
  const result = preview.result || {};
  const page = result.page || result.Page || {};
  const ranges = result.range_options || result.RangeOptions || [];
  const tooLarge = Boolean(result.full_body_too_large || result.FullBodyTooLarge);
  $("confluencePreviewMeta").innerHTML = `
    <div>${escapeHTML(page.title || page.Title || page.page_id || page.PageID || "")}</div>
    ${confluencePageMetaHTML(page)}
    <div class="item-meta">${tooLarge ? "전체 페이지가 커서 범위 선택 필요" : "전체 승인 가능"}</div>
    <div class="item-meta">미리보기는 결과이며, 승인 전까지 소스가 아닙니다.</div>
  `;
  $("confluencePreviewText").textContent = result.preview_text || result.PreviewText || "";
  const select = $("confluenceRangeSelect");
  select.innerHTML = ranges.length ? ranges.map((range, index) => {
    const start = range.start ?? range.Start ?? 0;
    const end = range.end ?? range.End ?? 0;
    const contentID = range.content_id || range.ContentID || "plain_text";
    const label = range.label || range.Label || `${start}-${end}`;
    return `<option value="${escapeAttr(index)}" data-content-id="${escapeAttr(contentID)}" data-start="${escapeAttr(start)}" data-end="${escapeAttr(end)}">${escapeHTML(label)}</option>`;
  }).join("") : `<option value="">범위 없음</option>`;
  $("confluenceApproveFullSnapshot").disabled = tooLarge || state.confluenceBusy;
  $("confluenceApproveRangeSnapshot").disabled = !ranges.length || state.confluenceBusy;
}

function confluencePageMetaHTML(page) {
  const siteURL = page.site_url || page.SiteURL || "";
  const webURL = page.web_url || page.WebURL || "";
  const space = page.space_key || page.SpaceKey || page.space_id || page.SpaceID || "";
  const pageID = page.page_id || page.PageID || "";
  const version = page.version || page.Version || "";
  const updated = page.updated_at || page.UpdatedAt || "";
  const parts = [];
  if (siteURL) parts.push(`사이트 ${siteURL}`);
  if (space) parts.push(`공간 ${space}`);
  if (pageID) parts.push(`페이지 ${pageID}`);
  if (version) parts.push(`v${version}`);
  if (updated) parts.push(`수정 ${timeShort(updated)}`);
  return `
    ${parts.length ? `<div class="item-meta">${escapeHTML(parts.join(" / "))}</div>` : ""}
    ${webURL ? `<div class="item-meta"><a href="${escapeAttr(webURL)}" target="_blank" rel="noopener noreferrer">${escapeHTML(webURL)}</a></div>` : ""}
  `;
}

async function approveConfluenceSnapshot(useRange) {
  if (state.confluenceBusy) return;
  const preview = state.confluencePreview;
  if (!preview) return;
  const result = preview.result || {};
  const page = result.page || result.Page || {};
  const body = {
    connection_id: preview.connection_id,
    cloud_id: preview.cloud_id,
    page_id: page.page_id || page.PageID,
    expected_version: Number(page.version || page.Version || 0),
    reason: useRange ? "Plasma 작업공간에서 Confluence 페이지 범위를 승인함" : "Plasma 작업공간에서 Confluence 페이지를 승인함"
  };
  if (useRange) {
    const option = $("confluenceRangeSelect").selectedOptions[0];
    if (!option) return;
    body.range_content_id = option.dataset.contentId || "plain_text";
    body.range_start = Number(option.dataset.start || 0);
    body.range_end = Number(option.dataset.end || 0);
  }
  const owner = captureMissionSelection();
  setConfluenceBusy(true);
  try {
    await missionApi(owner, "/sources/confluence/snapshot", { method: "POST", body });
    if (!ownsMissionSelection(owner)) return;
    state.confluencePreview = null;
    renderConfluencePreview(null);
    setConfluenceFlowStatus("Confluence 페이지를 소스로 저장했습니다. 같은 결과에서 다른 후보도 계속 검토할 수 있습니다.");
    await reloadMission(owner.missionId);
  } catch (err) {
    if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showConfluenceError(err);
  } finally {
    if (ownsMissionSelection(owner)) setConfluenceBusy(false);
  }
}

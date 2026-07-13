async function checkConfluenceSourceUpdate(snapshotID) {
  if (!requireMission() || state.confluenceBusy) return;
  const owner = captureMissionSelection();
  const connectionID = confluenceSelectedConnectionID();
  if (!connectionID) {
    showError(new Error("업데이트 확인에 사용할 Confluence 연결을 먼저 선택하세요."));
    return;
  }
  setConfluenceBusy(true);
  try {
    const result = await missionApi(owner, "/sources/confluence/check-update", {
      method: "POST",
      body: { connection_id: connectionID, snapshot_id: snapshotID }
    });
    if (!ownsMissionSelection(owner)) return;
    state.confluenceUpdatePreview = { check: result, connection_id: connectionID, snapshot_id: snapshotID };
    openConfluenceSourceDetails();
    renderConfluenceUpdatePanel(state.confluenceUpdatePreview);
    setConfluenceFlowStatus("Confluence 업데이트 확인 결과를 열었습니다. 미리보기 후 새 스냅샷으로 승인하세요.");
    $("confluenceUpdatePanel")?.scrollIntoView({ block: "nearest" });
    await reloadMission();
  } catch (err) {
    if (ownsMissionSelection(owner)) showConfluenceError(err);
  } finally {
    if (ownsMissionSelection(owner)) setConfluenceBusy(false);
  }
}

function renderConfluenceUpdatePanel(update) {
  const panel = $("confluenceUpdatePanel");
  if (!panel) return;
  panel.classList.toggle("hidden", !update);
  if (!update) return;
  const check = update.check || {};
  const preview = update.preview || {};
  const latest = check.LatestVersion || check.latest_version || preview?.new_page?.version || preview?.NewPage?.Version || 0;
  const current = check.CurrentVersion || check.current_version || preview?.old_page?.version || preview?.OldPage?.Version || 0;
  const available = Boolean(check.UpdateAvailable || check.update_available || preview.update_available || preview.UpdateAvailable);
  const requiresRange = Boolean(preview.requires_range_reselect || preview.RequiresRangeReselect);
  const fullBodyTooLarge = Boolean(preview.full_body_too_large || preview.FullBodyTooLarge);
  const rangeRequired = requiresRange || fullBodyTooLarge;
  const oldPage = preview.old_page || preview.OldPage || {};
  const newPage = preview.new_page || preview.NewPage || {};
  $("confluenceUpdateMeta").innerHTML = `
    <div>현재 v${escapeHTML(current || "?")} / 최신 v${escapeHTML(latest || "?")} / ${available ? "업데이트 있음" : "업데이트 없음"}</div>
    ${oldPage.page_id || oldPage.PageID ? `<div class="item-meta"><strong>현재</strong> ${escapeHTML(oldPage.title || oldPage.Title || "")}</div>${confluencePageMetaHTML(oldPage)}` : ""}
    ${newPage.page_id || newPage.PageID ? `<div class="item-meta"><strong>새 버전</strong> ${escapeHTML(newPage.title || newPage.Title || "")}</div>${confluencePageMetaHTML(newPage)}` : ""}
    <div class="item-meta">${rangeRequired ? "이 업데이트는 새 범위를 선택해야 승인할 수 있습니다." : "미리보기 후 승인하면 새 스냅샷이 생성됩니다."}</div>
    <div class="item-meta">업데이트 미리보기는 결과이며, 승인 전까지 소스가 아닙니다.</div>
  `;
  $("confluenceUpdatePreviewText").textContent = preview.preview_text || preview.PreviewText || "";
  const ranges = preview.range_options || preview.RangeOptions || [];
  const select = $("confluenceUpdateRangeSelect");
  select.innerHTML = ranges.length ? ranges.map((range, index) => {
    const start = range.start ?? range.Start ?? 0;
    const end = range.end ?? range.End ?? 0;
    const contentID = range.content_id || range.ContentID || "plain_text";
    const label = range.label || range.Label || `${start}-${end}`;
    return `<option value="${escapeAttr(index)}" data-content-id="${escapeAttr(contentID)}" data-start="${escapeAttr(start)}" data-end="${escapeAttr(end)}">${escapeHTML(label)}</option>`;
  }).join("") : `<option value="">범위 없음</option>`;
  $("confluenceUpdatePreviewButton").disabled = !available || state.confluenceBusy;
  $("confluenceApproveUpdate").disabled = state.confluenceBusy || (!preview.new_page && !preview.NewPage) || (rangeRequired && !ranges.length);
}

async function previewConfluenceUpdate() {
  if (state.confluenceBusy) return;
  const update = state.confluenceUpdatePreview;
  if (!update) return;
  const owner = captureMissionSelection();
  const latest = update.check?.LatestVersion || update.check?.latest_version || 0;
  setConfluenceBusy(true);
  try {
    const preview = await missionApi(owner, "/sources/confluence/update-preview", {
      method: "POST",
      body: { connection_id: update.connection_id, snapshot_id: update.snapshot_id, expected_version: latest }
    });
    if (!ownsMissionSelection(owner)) return;
    state.confluenceUpdatePreview = { ...update, preview };
    renderConfluenceUpdatePanel(state.confluenceUpdatePreview);
  } catch (err) {
    if (ownsMissionSelection(owner)) showConfluenceError(err);
  } finally {
    if (ownsMissionSelection(owner)) setConfluenceBusy(false);
  }
}

async function approveConfluenceUpdate() {
  if (state.confluenceBusy) return;
  const update = state.confluenceUpdatePreview;
  if (!update?.preview) return;
  const owner = captureMissionSelection();
  const preview = update.preview;
  const newPage = preview.new_page || preview.NewPage || {};
  const body = {
    connection_id: update.connection_id,
    snapshot_id: update.snapshot_id,
    expected_version: Number(newPage.version || newPage.Version || 0),
    reason: "Plasma 작업공간에서 Confluence update를 검토하고 승인함"
  };
  if (preview.requires_range_reselect || preview.RequiresRangeReselect || preview.full_body_too_large || preview.FullBodyTooLarge) {
    const option = $("confluenceUpdateRangeSelect").selectedOptions[0];
    if (!option || !option.dataset.start) {
      showError(new Error("이 Confluence 업데이트는 새 범위를 선택해야 승인할 수 있습니다."));
      return;
    }
    body.range_content_id = option.dataset.contentId || "plain_text";
    body.range_start = Number(option.dataset.start || 0);
    body.range_end = Number(option.dataset.end || 0);
  }
  setConfluenceBusy(true);
  try {
    await missionApi(owner, "/sources/confluence/update", { method: "POST", body });
    if (!ownsMissionSelection(owner)) return;
    state.confluenceUpdatePreview = null;
    renderConfluenceUpdatePanel(null);
    await reloadMission(owner.missionId);
  } catch (err) {
    if (ownsMissionSelection(owner)) showConfluenceError(err);
  } finally {
    if (ownsMissionSelection(owner)) setConfluenceBusy(false);
  }
}

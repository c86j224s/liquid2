function confluenceCallbackURL() {
  return `${window.location.origin}/api/settings/connectors/confluence/oauth/callback`;
}

function confluenceConnectionID(connection) {
  return connection?.connection_id || connection?.ConnectionID || "";
}

function confluenceConnectionName(connection) {
  return connection?.display_name || connection?.DisplayName || confluenceConnectionID(connection);
}

function confluenceConnectionAuthType(connection) {
  return connection?.auth_type || connection?.AuthType || "";
}

function confluenceConnectionIsAPIToken(connection) {
  return confluenceConnectionAuthType(connection) === "api_token";
}

function confluenceAPITokenConnections() {
  return (state.confluenceConnections || []).filter(confluenceConnectionIsAPIToken);
}

function confluenceAuthLabel(connection) {
  const auth = confluenceConnectionAuthType(connection);
  if (auth === "api_token") return "API token";
  if (auth === "oauth") return "OAuth · 0.0 미지원";
  return auth || "auth";
}

function confluenceConnectionSites(connection) {
  return connection?.sites || connection?.Sites || [];
}

function selectedConfluenceConnection() {
  const id = $("confluenceConnectionSelect")?.value || "";
  return confluenceAPITokenConnections().find((connection) => confluenceConnectionID(connection) === id) || null;
}

function selectedConfluenceSite() {
  const cloudID = $("confluenceSiteSelect")?.value || "";
  const connection = selectedConfluenceConnection();
  return confluenceConnectionSites(connection).find((site) => (site.cloud_id || site.CloudID || site.id || site.ID || "") === cloudID) || null;
}

function confluenceSelectedConnectionID() {
  const connection = selectedConfluenceConnection();
  return confluenceConnectionID(connection);
}

function confluenceSiteCloudID(site) {
  return site?.cloud_id || site?.CloudID || site?.id || site?.ID || "";
}

function confluenceSiteName(site) {
  return site?.name || site?.Name || confluenceSiteCloudID(site);
}

function confluenceSiteURL(site) {
  return site?.url || site?.URL || "";
}

function setConfluenceFlowStatus(message, tone = "") {
  const el = $("confluenceFlowStatus");
  if (!el) return;
  el.textContent = message || "";
  el.classList.toggle("warn", tone === "warn");
}

function openConfluenceSourceDetails() {
  const details = $("confluenceSourceDetails");
  if (details) details.open = true;
}

function setConfluenceBusy(busy) {
  state.confluenceBusy = busy;
  renderConfluenceControls();
  if (typeof renderConfluenceSettingsControls === "function") renderConfluenceSettingsControls();
  if (typeof renderConfluenceAccessControls === "function") renderConfluenceAccessControls();
  renderConfluencePreview(state.confluencePreview);
  renderConfluenceUpdatePanel(state.confluenceUpdatePreview);
}

function resetConfluenceMissionUI() {
  renderConfluenceSpaces([]);
  renderConfluencePages([]);
  renderConfluencePreview(null);
  renderConfluenceUpdatePanel(null);
  renderConfluenceAccessControls();
  renderConfluenceControls();
}

async function loadConfluenceConnections(preferredConnectionID = "", owner = captureMissionSelection()) {
  try {
    const current = preferredConnectionID || $("confluenceConnectionSelect")?.value || "";
    const result = await api(`/api/settings/connectors/confluence/connections`);
    if (!ownsMissionSelection(owner)) return;
    state.confluenceConnections = result.connections || result.Connections || [];
    state.confluenceOAuthConfigured = Boolean(result.oauth_configured ?? result.OAuthConfigured);
    renderConfluenceControls(current);
    if (typeof renderConfluenceSettingsControls === "function") renderConfluenceSettingsControls(preferredConnectionID);
    if (typeof renderConfluenceAccessControls === "function") renderConfluenceAccessControls();
  } catch (err) {
    if (!ownsMissionSelection(owner)) return;
    showConfluenceError(err);
    renderConfluenceControls();
    if (typeof renderConfluenceSettingsControls === "function") renderConfluenceSettingsControls();
    if (typeof renderConfluenceAccessControls === "function") renderConfluenceAccessControls();
  }
}

function renderConfluenceControls(preferredConnectionID = "") {
  const select = $("confluenceConnectionSelect");
  if (!select) return;
  const enabled = Boolean(state.missionId);
  const connections = confluenceAPITokenConnections();
  const current = preferredConnectionID || select.value || confluenceConnectionID(connections[0] || {});
  select.innerHTML = connections.length
    ? connections.map((connection) => {
        const id = confluenceConnectionID(connection);
        return `<option value="${escapeAttr(id)}">${escapeHTML(confluenceConnectionName(connection))} · ${escapeHTML(confluenceAuthLabel(connection))}</option>`;
      }).join("")
    : `<option value="">API token 연결 없음</option>`;
  if (connections.some((connection) => confluenceConnectionID(connection) === current)) {
    select.value = current;
  }
  select.disabled = !enabled || state.confluenceBusy || connections.length === 0;
  const connection = selectedConfluenceConnection();
  renderConfluenceConnectionSummary(connection);
  renderConfluenceSiteOptions(connection);
  // Step ①: when no connection, make the button a prominent "연결하기"; once
  // connected it becomes a quiet "관리" link.
  const settingsBtn = $("openConfluenceSettings");
  if (settingsBtn) {
    settingsBtn.textContent = connection ? "연결 관리 (설정)" : "＋ Confluence 연결하기";
    settingsBtn.classList.toggle("secondary", Boolean(connection));
  }
  for (const id of [
    "confluenceRefreshConnections", "confluencePageURL", "confluenceAddURLButton", "confluenceQuery", "confluenceSpaceKey",
    "confluenceLimit", "confluenceOneClickStart", "confluenceLoadSpaces",
    "confluenceLoadMoreSpaces", "confluenceLoadMorePages", "confluenceRangeSelect",
    "confluenceUpdateRangeSelect"
  ]) {
    const el = $(id);
    if (el) el.disabled = !enabled || state.confluenceBusy;
  }
  if ($("confluenceLoadSpaces")) $("confluenceLoadSpaces").disabled = !enabled || state.confluenceBusy || !connection || !selectedConfluenceSite();
  if ($("confluenceAddURLButton")) $("confluenceAddURLButton").disabled = !enabled || state.confluenceBusy || !connection || !selectedConfluenceSite();
  if ($("confluenceOneClickStart")) $("confluenceOneClickStart").disabled = !enabled || state.confluenceBusy;
  $("confluenceSearchForm")?.querySelectorAll("button").forEach((button) => {
    button.disabled = !enabled || state.confluenceBusy || !connection || !selectedConfluenceSite();
  });
}

function renderConfluenceConnectionSummary(connection) {
  const el = $("confluenceConnectionSummary");
  if (!el) return;
  if (!state.missionId) {
    el.textContent = "먼저 미션을 선택하세요.";
    return;
  }
  if (!connection) {
    const hasOAuthOnly = (state.confluenceConnections || []).some((item) => confluenceConnectionAuthType(item) === "oauth");
    el.textContent = hasOAuthOnly
      ? "API token Confluence 연결이 없습니다. 기존 OAuth 연결은 0.0에서 사용하지 않습니다. 설정에서 API token 연결을 추가하세요."
      : "Confluence 연결이 없습니다. 설정에서 API token 연결을 만든 뒤 이 미션에서 페이지를 소스로 승인하세요.";
    return;
  }
  const sites = confluenceConnectionSites(connection);
  const auth = confluenceConnectionAuthType(connection);
  const expires = connection.token_expires_at || connection.TokenExpiresAt || "";
  const updated = connection.updated_at || connection.UpdatedAt || "";
  const revoked = Boolean(connection.revoked || connection.Revoked);
  const scopes = connection.scopes || connection.Scopes || [];
  el.innerHTML = `
    <span class="badge">${escapeHTML(confluenceAuthLabel(connection))}</span>
    ${revoked ? `<span class="badge warn">해제됨</span>` : ""}
    <span>${escapeHTML(confluenceConnectionID(connection))}</span>
    <span class="muted-inline">사이트 ${sites.length}개</span>
    ${expires ? `<span class="muted-inline">만료 ${escapeHTML(timeShort(expires))}</span>` : ""}
    ${updated ? `<span class="muted-inline">수정 ${escapeHTML(timeShort(updated))}</span>` : ""}
    ${scopes.length ? `<span class="muted-inline">scope ${escapeHTML(scopes.join(", "))}</span>` : ""}
  `;
}

function renderConfluenceSiteOptions(connection) {
  const select = $("confluenceSiteSelect");
  if (!select) return;
  const sites = confluenceConnectionSites(connection);
  const current = select.value || confluenceSiteCloudID(sites[0] || {});
  select.innerHTML = sites.length
    ? sites.map((site) => {
        const cloudID = confluenceSiteCloudID(site);
        const label = confluenceSiteURL(site)
          ? `${confluenceSiteName(site)} · ${confluenceSiteURL(site)}`
          : confluenceSiteName(site);
        return `<option value="${escapeAttr(cloudID)}">${escapeHTML(label)}</option>`;
      }).join("")
    : `<option value="">사이트 없음</option>`;
  if (sites.some((site) => confluenceSiteCloudID(site) === current)) {
    select.value = current;
  }
  select.disabled = !state.missionId || state.confluenceBusy || sites.length === 0;
}

async function searchConfluence(event) {
  event.preventDefault();
  await searchConfluenceResults({ previewSingle: false });
}

async function addConfluenceURLSource(event) {
  event.preventDefault();
  if (!requireMission()) return;
  const url = $("confluencePageURL").value.trim();
  const connectionID = confluenceSelectedConnectionID();
  const site = selectedConfluenceSite();
  const cloudID = confluenceSiteCloudID(site);
  if (!url) {
    showError(new Error("Confluence 페이지 URL이 필요합니다."));
    return;
  }
  if (!connectionID || !cloudID) {
    showError(new Error("Confluence 연결과 사이트를 선택한 뒤 URL을 추가하세요."));
    return;
  }
  if (typeof looksLikeConfluenceURL === "function" && !looksLikeConfluenceURL(url)) {
    showError(new Error("Confluence 페이지 URL만 이 영역에서 추가할 수 있습니다."));
    return;
  }
  const owner = captureMissionSelection();
  setConfluenceBusy(true);
  try {
    const title = typeof sourceCandidateTitleForURL === "function" ? sourceCandidateTitleForURL(url) : "";
    await missionApi(owner, "/sources/confluence/url", {
      method: "POST",
      body: {
        url,
        title,
        connection_id: connectionID,
        cloud_id: cloudID
      }
    });
    if (!ownsMissionSelection(owner)) return;
    $("confluencePageURL").value = "";
    setConfluenceFlowStatus("Confluence URL을 소스로 추가했습니다.");
    await reloadMission(owner.missionId);
  } catch (err) {
    if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showConfluenceError(err);
  } finally {
    if (ownsMissionSelection(owner)) setConfluenceBusy(false);
  }
}

async function searchConfluenceResults({ previewSingle = false } = {}) {
  if (!requireMission()) return;
  const owner = captureMissionSelection();
  const connectionID = confluenceSelectedConnectionID();
  const site = selectedConfluenceSite();
  const cloudID = confluenceSiteCloudID(site);
  const query = $("confluenceQuery").value.trim();
  if (!connectionID || !cloudID || !query) {
    showError(new Error("Confluence 연결, 사이트, 검색어가 필요합니다."));
    return;
  }
  setConfluenceBusy(true);
  try {
    const result = await missionApi(owner, "/sources/confluence/search", {
      method: "POST",
      body: {
        connection_id: connectionID,
        cloud_id: cloudID,
        query,
        space_key: $("confluenceSpaceKey").value.trim(),
        limit: Number($("confluenceLimit").value || 10)
      }
    });
    if (!ownsMissionSelection(owner)) return;
    state.confluenceSearchResults = result.Candidates || result.candidates || [];
    state.confluenceSearchContext = { connection_id: connectionID, cloud_id: cloudID };
    renderConfluenceResults(state.confluenceSearchResults);
    setConfluenceFlowStatus(state.confluenceSearchResults.length ? `검색 결과 ${state.confluenceSearchResults.length}개를 찾았습니다. 후보를 검토한 뒤 소스로 승인하세요.` : "검색 결과가 없습니다. 검색어를 바꿔 다시 시도하세요.", state.confluenceSearchResults.length ? "" : "warn");
  } catch (err) {
    if (ownsMissionSelection(owner)) showConfluenceError(err);
  } finally {
    if (ownsMissionSelection(owner)) setConfluenceBusy(false);
  }
  if (ownsMissionSelection(owner) && previewSingle && state.confluenceSearchResults.length === 1) {
    await previewConfluenceCandidate(0);
  }
}

function clearConfluenceSearchResults() {
  state.confluenceSearchResults = [];
  state.confluenceSearchContext = null;
  renderConfluenceResults([]);
}

function confluenceCandidateDetailPayload(candidate) {
  const title = candidate.Title || candidate.title || confluenceCandidatePageID(candidate) || "Confluence 페이지";
  const sourceURI = confluenceDisplayableExternalURI(candidate.SourceURI || candidate.source_uri || "");
  const siteURL = confluenceDisplayableExternalURI(candidate.SiteURL || candidate.site_url || "");
  const pageID = confluenceCandidatePageID(candidate);
  const detail = {
    type: "confluence_candidate",
    title,
    site_url: siteURL,
    site_host: confluenceExternalURIHost(siteURL || sourceURI),
    page_id: pageID,
    space_key: candidate.SpaceKey || candidate.space_key || "",
    version: candidate.Version || candidate.version || "",
    updated_at: candidate.UpdatedAt || candidate.updated_at || "",
    can_snapshot: Boolean(candidate.CanSnapshot ?? candidate.can_snapshot)
  };
  if (sourceURI) detail.source_uri = sourceURI;
  return detail;
}

function renderConfluenceResults(candidates) {
  const container = $("confluenceResults");
  if (!container) return;
  container.innerHTML = candidates.length ? candidates.map((candidate, index) => {
    const title = candidate.Title || candidate.title || confluenceCandidatePageID(candidate) || "Confluence 페이지";
    const sourceURI = confluenceDisplayableExternalURI(candidate.SourceURI || candidate.source_uri || "");
    const space = candidate.SpaceKey || candidate.space_key || "";
    const version = candidate.Version || candidate.version || 0;
    const updated = candidate.UpdatedAt || candidate.updated_at || "";
    const detailPayload = confluenceCandidateDetailPayload(candidate);
    return `
      <div class="item">
        <div class="item-title">${escapeHTML(title)} <span class="badge muted">v${escapeHTML(version || "?")}</span></div>
        <div class="item-meta">${space ? `공간 ${escapeHTML(space)} / ` : ""}${escapeHTML(sourceURI || confluenceCandidatePageID(candidate))}</div>
        ${updated ? `<div class="item-meta">수정: ${escapeHTML(timeShort(updated))}</div>` : ""}
        <div class="item-actions">
          ${sourceURI ? `<a class="button-link secondary" href="${escapeAttr(sourceURI)}" target="_blank" rel="noopener noreferrer">원문 열기</a>` : ""}
          <button type="button" class="secondary" data-detail-title="Confluence 검색 결과" data-detail-json="${escapeAttr(JSON.stringify(detailPayload))}">자세히</button>
          <button type="button" data-confluence-candidate-index="${escapeAttr(index)}">후보 검토</button>
        </div>
      </div>
    `;
  }).join("") : empty("Confluence 검색 결과 없음");
}

async function runConfluenceOneClickFlow({ fromOAuth = false } = {}) {
  if (!requireMission() || state.confluenceBusy) return;
  openConfluenceSourceDetails();
  if (!state.confluenceConnections.length) {
    await loadConfluenceConnections();
  }
  let connection = selectedConfluenceConnection();
  if (!connection) {
    setConfluenceFlowStatus("저장된 연결이 없습니다. 설정에서 Confluence 연결을 만든 뒤 다시 시도하세요.", "warn");
    openSettingsTab();
    return;
  }
  let site = selectedConfluenceSite();
  if (!site) {
    setConfluenceFlowStatus("선택할 Confluence 사이트가 없습니다. 설정에서 사이트를 새로고침하거나 연결 권한을 확인하세요.", "warn");
    return;
  }
  const query = $("confluenceQuery")?.value.trim() || "";
  if (query) {
    setConfluenceFlowStatus("검색하고 후보를 검토할 준비를 하고 있습니다.");
    await searchConfluenceResults({ previewSingle: true });
    return;
  }
  const currentContext = state.confluenceBrowseContext || {};
  const sameSite = currentContext.connection_id === confluenceSelectedConnectionID() && currentContext.cloud_id === confluenceSiteCloudID(site);
  if (!sameSite || !state.confluenceSpaces.length) {
    setConfluenceFlowStatus(fromOAuth ? "연결이 완료되어 공간 목록을 불러오고 있습니다." : "공간 목록을 불러오고 있습니다.");
    await loadConfluenceSpaces();
  }
  if (state.confluenceSpaces.length === 1 && !state.confluencePages.length) {
    const space = state.confluenceSpaces[0];
    await loadConfluenceSpacePages(space.space_id || space.SpaceID || "", space.name || space.Name || "");
  }
  if (state.confluencePages.length === 1) {
    setConfluenceFlowStatus("페이지가 하나라 후보 검토 화면까지 열었습니다. 내용을 확인한 뒤 소스로 승인하세요.");
    await previewConfluencePage(state.confluencePages[0]);
    return;
  }
  if (state.confluencePages.length > 1) {
    setConfluenceFlowStatus(`페이지 ${state.confluencePages.length}개를 찾았습니다. 필요한 페이지를 후보 검토하세요.`);
    return;
  }
  if (state.confluenceSpaces.length > 1) {
    setConfluenceFlowStatus(`공간 ${state.confluenceSpaces.length}개를 찾았습니다. 공간을 선택하면 페이지 목록으로 이어집니다.`);
    return;
  }
  setConfluenceFlowStatus("탐색 가능한 공간이나 페이지를 찾지 못했습니다.", "warn");
}

function initConfluenceOAuthListener() {
  if (state.confluenceOAuthListenerReady) return;
  state.confluenceOAuthListenerReady = true;
  const onOAuthMessage = async (payload) => {
    if (!payload || (payload.type !== "plasma.confluence.settings.oauth" && payload.type !== "plasma.confluence.oauth")) return;
    if (payload.mission_id && payload.mission_id !== state.missionId) return;
    if (payload.ok) {
      openSettingsTab();
      await loadConfluenceConnections(payload.connection_id || "");
      if (typeof setConfluenceSettingsStatus === "function") setConfluenceSettingsStatus("Confluence 연결이 완료되었습니다. 미션 소스에서 연결과 사이트를 선택해 페이지를 소스로 승인할 수 있습니다.");
      return;
    }
    if (typeof setConfluenceSettingsStatus === "function") setConfluenceSettingsStatus(payload.message || "Confluence 연결이 완료되지 않았습니다.", "warn");
  };
  window.addEventListener("message", (event) => {
    if (event.origin !== window.location.origin) return;
    onOAuthMessage(event.data);
  });
  if ("BroadcastChannel" in window) {
    const channel = new BroadcastChannel("plasma.confluence.oauth");
    channel.addEventListener("message", (event) => onOAuthMessage(event.data));
    state.confluenceOAuthChannel = channel;
    const settingsChannel = new BroadcastChannel("plasma.confluence.settings.oauth");
    settingsChannel.addEventListener("message", (event) => onOAuthMessage(event.data));
    state.confluenceSettingsOAuthChannel = settingsChannel;
  }
}

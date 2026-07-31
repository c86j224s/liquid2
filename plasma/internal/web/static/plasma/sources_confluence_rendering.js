(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const escapeAttr = Plasma.dom.escapeAttr;
  const timeShort = Plasma.dom.timeShort;
  const confluenceConnectionID = (...args) => sources.confluenceConnectionID(...args);
  const confluenceConnectionName = (...args) => sources.confluenceConnectionName(...args);
  const confluenceConnectionAuthType = (...args) => sources.confluenceConnectionAuthType(...args);
  const confluenceConnectionSites = (...args) => sources.confluenceConnectionSites(...args);
  const confluenceAuthLabel = (...args) => sources.confluenceAuthLabel(...args);
  const confluenceAPITokenConnections = (...args) => sources.confluenceAPITokenConnections(...args);
  const selectedConfluenceConnection = (...args) => sources.selectedConfluenceConnection(...args);
  const selectedConfluenceSite = (...args) => sources.selectedConfluenceSite(...args);
  const confluenceSiteCloudID = (...args) => sources.confluenceSiteCloudID(...args);
  const confluenceSiteName = (...args) => sources.confluenceSiteName(...args);
  const confluenceSiteURL = (...args) => sources.confluenceSiteURL(...args);

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

  Object.assign(sources, {
    renderConfluenceControls,
    renderConfluenceConnectionSummary,
    renderConfluenceSiteOptions
  });
})(window.Plasma);

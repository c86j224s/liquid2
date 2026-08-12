(function (Plasma) {
  "use strict";

  const settings = Plasma.settings;
  const sources = Plasma.sources;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const escapeAttr = Plasma.dom.escapeAttr;
  const empty = Plasma.ui.empty;
  const renderTabs = (...args) => settings.dependency("renderTabs")(...args);
  const loadModelDefaults = (...args) => settings.loadModelDefaults(...args);
  const loadMissionUsage = (...args) => settings.loadMissionUsage(...args);
  const loadConfluenceConnections = (...args) => sources.loadConfluenceConnections(...args);
  const confluenceConnectionSites = (...args) => sources.confluenceConnectionSites(...args);
  const confluenceSiteName = (...args) => sources.confluenceSiteName(...args);
  const confluenceSiteCloudID = (...args) => sources.confluenceSiteCloudID(...args);
  const confluenceSiteURL = (...args) => sources.confluenceSiteURL(...args);
  const confluenceConnectionID = (...args) => sources.confluenceConnectionID(...args);
  const confluenceConnectionName = (...args) => sources.confluenceConnectionName(...args);
  const confluenceConnectionAuthType = (...args) => sources.confluenceConnectionAuthType(...args);
  const confluenceAuthLabel = (...args) => sources.confluenceAuthLabel(...args);

  function openSettingsTab() {
    state.activeTab = "settings";
    renderTabs();
    loadMissionUsage();
    loadModelDefaults();
    loadConfluenceConnections();
  }

  function setConfluenceSettingsStatus(message, tone = "") {
    const el = $("confluenceSettingsConnectionSummary");
    if (!el) return;
    el.textContent = message || "";
    el.classList.toggle("warn", tone === "warn");
  }

  function renderConfluenceSettingsControls() {
    const container = $("confluenceSettingsConnections");
    if (!container) return;
    const connections = state.confluenceConnections || [];
    renderConfluenceSettingsSummary(connections);
    renderConfluenceSettingsConnections(connections);
    // The add-token form inputs / refresh button follow the busy state.
    for (const id of [
    "confluenceSettingsRefreshConnections",
    "confluenceSettingsAPIDisplayName", "confluenceSettingsAPIEmail",
    "confluenceSettingsAPIToken", "confluenceSettingsAPISiteURL", "confluenceSettingsAPISiteName"
    ]) {
      const el = $(id);
      if (el) el.disabled = state.confluenceBusy;
    }
  }

  function renderConfluenceSettingsSummary(connections) {
    const el = $("confluenceSettingsConnectionSummary");
    if (!el) return;
    el.classList.remove("warn");
    el.textContent = connections.length
    ? `저장된 Confluence 연결 ${connections.length}개. 카드를 펼쳐 사이트 확인·이름 변경·삭제할 수 있습니다.`
    : "저장된 Confluence 연결이 없습니다. 아래 ‘＋ 새 API token 연결 추가’로 연결하세요.";
  }

  function confluenceSettingsSitesHTML(connection) {
    const sites = confluenceConnectionSites(connection);
    return sites.length ? sites.map((site) => `
    <div class="item">
      <div class="item-title">${escapeHTML(confluenceSiteName(site))}</div>
      <div class="item-meta">${escapeHTML(confluenceSiteCloudID(site))}${confluenceSiteURL(site) ? ` / ${escapeHTML(confluenceSiteURL(site))}` : ""}</div>
    </div>
  `).join("") : empty("저장된 Confluence 사이트 없음");
  }

  function renderConfluenceSettingsConnections(connections) {
    const container = $("confluenceSettingsConnections");
    if (!container) return;
    if (!connections.length) {
      container.innerHTML = empty("저장된 연결 없음");
      return;
    }
    // Preserve which cards are expanded across a re-render.
    const openIDs = new Set(
    Array.from(container.querySelectorAll("details.confluence-conn-card[open]"))
    .map((card) => card.dataset.connectionId)
    );
    const disabled = state.confluenceBusy ? "disabled" : "";
    container.innerHTML = connections.map((connection) => {
      const id = confluenceConnectionID(connection);
      const name = confluenceConnectionName(connection);
      const auth = confluenceConnectionAuthType(connection);
      const revoked = Boolean(connection.revoked || connection.Revoked);
      const sites = confluenceConnectionSites(connection);
      const scopes = connection.scopes || connection.Scopes || [];
      const isOAuth = auth === "oauth";
      const open = openIDs.has(id) ? " open" : "";
      return `
      <details class="confluence-conn-card" data-connection-id="${escapeAttr(id)}"${open}>
        <summary>
          <span class="conn-card-name">${escapeHTML(name)}</span>
          <span class="badge">${escapeHTML(confluenceAuthLabel(connection))}</span>
          ${revoked ? `<span class="badge warn">해제됨</span>` : ""}
          ${isOAuth ? `<span class="badge warn">0.0 사용 불가</span>` : ""}
          <span class="muted-inline conn-card-sites-count">사이트 ${sites.length}개</span>
        </summary>
        <div class="conn-card-body stack">
          <div class="item-meta">${escapeHTML(id)}${scopes.length ? ` · scope ${escapeHTML(scopes.join(", "))}` : ""}</div>
          <div class="list compact conn-card-sites">${confluenceSettingsSitesHTML(connection)}</div>
          <div class="inline-form conn-card-rename">
            <input class="conn-rename-input" value="${escapeAttr(name)}" placeholder="연결 표시 이름" ${disabled}>
            <button type="button" class="secondary" data-conn-action="rename" ${disabled}>이름 변경</button>
          </div>
          <div class="inline-actions conn-card-actions">
            <button type="button" class="secondary" data-conn-action="refresh-sites" ${disabled}>사이트 새로고침</button>
            <button type="button" class="secondary" data-conn-action="revoke" ${disabled}>로컬 해제</button>
            <button type="button" class="danger" data-conn-action="delete" ${disabled}>연결 삭제</button>
          </div>
        </div>
      </details>
    `;
    }).join("");
  }

  Object.assign(settings, {
    openSettingsTab,
    setConfluenceSettingsStatus,
    renderConfluenceSettingsControls,
    renderConfluenceSettingsSummary,
    confluenceSettingsSitesHTML,
    renderConfluenceSettingsConnections
  });
})(window.Plasma);

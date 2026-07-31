(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const settings = Plasma.settings;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const api = Plasma.transport.api;
  const captureMissionSelection = Plasma.mission.captureMissionSelection;
  const ownsMissionSelection = Plasma.mission.ownsMissionSelection;
  const renderConfluenceSettingsControls = (...args) => settings.renderConfluenceSettingsControls(...args);
  const renderConfluenceControls = (...args) => sources.renderConfluenceControls(...args);
  const renderConfluenceSpaces = (...args) => sources.renderConfluenceSpaces(...args);
  const renderConfluencePages = (...args) => sources.renderConfluencePages(...args);
  const renderConfluencePreview = (...args) => sources.renderConfluencePreview(...args);
  const renderConfluenceUpdatePanel = (...args) => sources.renderConfluenceUpdatePanel(...args);
  const renderConfluenceAccessControls = (...args) => sources.renderConfluenceAccessControls(...args);
  const showConfluenceError = (...args) => sources.showConfluenceError(...args);

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
    renderConfluenceSettingsControls();
    renderConfluenceAccessControls();
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
      renderConfluenceSettingsControls(preferredConnectionID);
      renderConfluenceAccessControls();
    } catch (err) {
      if (!ownsMissionSelection(owner)) return;
      showConfluenceError(err);
      renderConfluenceControls();
      renderConfluenceSettingsControls();
      renderConfluenceAccessControls();
    }
  }

  Object.assign(sources, {
    confluenceCallbackURL,
    confluenceConnectionID,
    confluenceConnectionName,
    confluenceConnectionAuthType,
    confluenceConnectionIsAPIToken,
    confluenceAPITokenConnections,
    confluenceAuthLabel,
    confluenceConnectionSites,
    selectedConfluenceConnection,
    selectedConfluenceSite,
    confluenceSelectedConnectionID,
    confluenceSiteCloudID,
    confluenceSiteName,
    confluenceSiteURL,
    setConfluenceFlowStatus,
    openConfluenceSourceDetails,
    setConfluenceBusy,
    resetConfluenceMissionUI,
    loadConfluenceConnections
  });
})(window.Plasma);

(function (Plasma) {
  "use strict";

  const settings = Plasma.settings;
  const sources = Plasma.sources;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const api = Plasma.transport.api;
  const showError = Plasma.ui.showError;
  const setConfluenceBusy = (...args) => sources.setConfluenceBusy(...args);
  const loadConfluenceConnections = (...args) => sources.loadConfluenceConnections(...args);
  const clearConfluenceDiscovery = (...args) => sources.clearConfluenceDiscovery(...args);
  const showConfluenceError = (...args) => sources.showConfluenceError(...args);
  const setConfluenceSettingsStatus = (...args) => settings.setConfluenceSettingsStatus(...args);
  const confluenceConnectionID = (...args) => sources.confluenceConnectionID(...args);
  const confluenceConnectionAuthType = (...args) => sources.confluenceConnectionAuthType(...args);

  function onConfluenceSettingsCardClick(event) {
    const btn = event.target.closest("[data-conn-action]");
    if (!btn) return;
    const card = btn.closest("[data-connection-id]");
    const connectionID = card?.dataset.connectionId || "";
    if (!connectionID) return;
    const action = btn.dataset.connAction;
    if (action === "rename") {
      const input = card.querySelector(".conn-rename-input");
      renameConfluenceSettingsConnection(connectionID, input?.value.trim() || "");
    } else if (action === "revoke") {
      revokeConfluenceSettingsConnection(connectionID);
    } else if (action === "delete") {
      deleteConfluenceSettingsConnection(connectionID);
    } else if (action === "refresh-sites") {
      refreshConfluenceSettingsSites(connectionID);
    }
  }

  async function connectConfluenceAPIToken(event) {
    event.preventDefault();
    const siteURL = $("confluenceSettingsAPISiteURL").value.trim();
    const accountName = $("confluenceSettingsAPIEmail").value.trim();
    const apiToken = $("confluenceSettingsAPIToken").value.trim();
    if (!siteURL) {
      showError(new Error("API token 수동 연결에는 Confluence 사이트 URL이 필요합니다."));
      return;
    }
    if (!accountName) {
      showError(new Error("API token 연결에는 Atlassian 계정 이메일이 필요합니다."));
      return;
    }
    if (!apiToken) {
      showError(new Error("API token 연결에는 Atlassian API token이 필요합니다. 필요하면 새 token을 만드세요."));
      return;
    }
    setConfluenceBusy(true);
    let connectedStatus = "";
    try {
      await api(`/api/settings/connectors/confluence/connections`, {
        method: "POST",
        body: {
          display_name: $("confluenceSettingsAPIDisplayName").value.trim() || "Confluence",
          auth_type: "api_token",
          account_name: accountName,
          api_token: apiToken,
          sites: [{
            name: $("confluenceSettingsAPISiteName").value.trim(),
            url: siteURL
          }]
        }
      });
      $("confluenceSettingsAPIToken").value = "";
      $("confluenceSettingsAddCard")?.removeAttribute("open");
      await loadConfluenceConnections();
      connectedStatus = "API token 연결을 추가했습니다. 미션 소스에서 연결과 사이트를 선택해 페이지를 소스로 승인할 수 있습니다.";
      setConfluenceSettingsStatus(connectedStatus);
    } catch (err) {
      showConfluenceError(err);
    } finally {
      setConfluenceBusy(false);
    }
    if (connectedStatus) setConfluenceSettingsStatus(connectedStatus);
  }

  async function renameConfluenceSettingsConnection(connectionID, displayName) {
    if (!connectionID || !displayName) {
      showError(new Error("이름을 변경할 Confluence 연결과 새 표시 이름이 필요합니다."));
      return;
    }
    setConfluenceBusy(true);
    try {
      await api(`/api/settings/connectors/confluence/connections/${encodeURIComponent(connectionID)}`, {
        method: "PATCH",
        body: { display_name: displayName }
      });
      await loadConfluenceConnections(connectionID);
    } catch (err) {
      showConfluenceError(err);
    } finally {
      setConfluenceBusy(false);
    }
  }

  async function revokeConfluenceSettingsConnection(connectionID) {
    if (!connectionID) return;
    setConfluenceBusy(true);
    try {
      await api(`/api/settings/connectors/confluence/connections/${encodeURIComponent(connectionID)}/revoke`, {
        method: "POST",
        body: {}
      });
      clearConfluenceDiscovery();
      await loadConfluenceConnections(connectionID);
    } catch (err) {
      showConfluenceError(err);
    } finally {
      setConfluenceBusy(false);
    }
  }

  async function deleteConfluenceSettingsConnection(connectionID) {
    if (!connectionID) return;
    setConfluenceBusy(true);
    try {
      await api(`/api/settings/connectors/confluence/connections/${encodeURIComponent(connectionID)}`, {
        method: "DELETE",
        body: {}
      });
      clearConfluenceDiscovery();
      await loadConfluenceConnections();
    } catch (err) {
      showConfluenceError(err);
    } finally {
      setConfluenceBusy(false);
    }
  }

  async function refreshConfluenceSettingsSites(connectionID) {
    const connection = (state.confluenceConnections || []).find(
    (item) => confluenceConnectionID(item) === connectionID
    );
    if (!connectionID || !connection) {
      showError(new Error("먼저 Confluence 연결을 선택하세요."));
      return;
    }
    if (confluenceConnectionAuthType(connection) === "api_token") {
      showError(new Error("API token 연결은 등록할 때 저장한 사이트 정보를 사용합니다. 사이트를 바꾸려면 연결을 다시 추가하세요."));
      return;
    }
    if (confluenceConnectionAuthType(connection) === "oauth") {
      showError(new Error("OAuth 연결은 0.0에서 사용하지 않습니다. API token 연결을 새로 추가하세요."));
      return;
    }
    setConfluenceBusy(true);
    try {
      await api(`/api/settings/connectors/confluence/connections/${encodeURIComponent(connectionID)}/sites/refresh`, {
        method: "POST",
        body: {}
      });
      await loadConfluenceConnections(connectionID);
    } catch (err) {
      showConfluenceError(err);
    } finally {
      setConfluenceBusy(false);
    }
  }

  Object.assign(settings, {
    onConfluenceSettingsCardClick,
    connectConfluenceAPIToken,
    renameConfluenceSettingsConnection,
    revokeConfluenceSettingsConnection,
    deleteConfluenceSettingsConnection,
    refreshConfluenceSettingsSites
  });
})(window.Plasma);

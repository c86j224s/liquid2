function openSettingsTab() {
  state.activeTab = "settings";
  renderTabs();
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
    setConfluenceSettingsStatus("API token 연결을 추가했습니다. 미션 소스에서 연결과 사이트를 선택해 페이지를 소스로 승인할 수 있습니다.");
  } catch (err) {
    showConfluenceError(err);
  } finally {
    setConfluenceBusy(false);
  }
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

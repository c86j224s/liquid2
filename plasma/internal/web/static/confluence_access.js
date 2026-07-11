async function loadConfluenceAccess() {
  if (!state.missionId) {
    state.confluenceAccess = null;
    renderConfluenceAccessControls();
    return;
  }
  try {
    const result = await api(`/api/missions/${state.missionId}/connector-access/confluence`);
    state.confluenceAccess = result.access || result.Access || null;
    renderConfluenceAccessControls();
  } catch (err) {
    showError(err);
    renderConfluenceAccessControls();
  }
}

function renderConfluenceAccessControls() {
  const connectionSelect = $("confluenceAccessConnectionSelect");
  if (!connectionSelect) return;
  const access = state.confluenceAccess || {};
  const connections = typeof confluenceAPITokenConnections === "function" ? confluenceAPITokenConnections() : [];
  const selectedConnectionID = connectionSelect.value || access.connection_id || access.ConnectionID || confluenceConnectionID(connections[0] || {});
  connectionSelect.innerHTML = connections.length ? connections.map((connection) => {
    const id = confluenceConnectionID(connection);
    return `<option value="${escapeAttr(id)}">${escapeHTML(confluenceConnectionName(connection))} · API token</option>`;
  }).join("") : `<option value="">API token 연결 없음</option>`;
  if (connections.some((connection) => confluenceConnectionID(connection) === selectedConnectionID)) {
    connectionSelect.value = selectedConnectionID;
  }
  const connection = connections.find((item) => confluenceConnectionID(item) === connectionSelect.value) || null;
  const sites = confluenceConnectionSites(connection);
  const siteSelect = $("confluenceAccessSiteSelect");
  const selectedCloudID = siteSelect.value || access.cloud_id || access.CloudID || confluenceSiteCloudID(sites[0] || {});
  siteSelect.innerHTML = sites.length ? sites.map((site) => {
    const cloudID = confluenceSiteCloudID(site);
    const label = confluenceSiteURL(site) ? `${confluenceSiteName(site)} · ${confluenceSiteURL(site)}` : confluenceSiteName(site);
    return `<option value="${escapeAttr(cloudID)}">${escapeHTML(label)}</option>`;
  }).join("") : `<option value="">site 없음</option>`;
  if (sites.some((site) => confluenceSiteCloudID(site) === selectedCloudID)) {
    siteSelect.value = selectedCloudID;
  }
  const enabled = Boolean(state.missionId);
  const grantEnabled = Boolean(access.enabled || access.Enabled);
  const status = access.status || access.Status || "disabled";
  const invalidReason = access.invalid_reason || access.InvalidReason || "";
  $("confluenceAccessSpaceKey").value = access.space_key || access.SpaceKey || $("confluenceAccessSpaceKey").value || "";
  $("confluenceAccessBadge").textContent = grantEnabled && status === "enabled" ? "on" : (status === "invalid" ? "invalid" : "off");
  $("confluenceAccessBadge").classList.toggle("warn", status === "invalid");
  $("confluenceAccessStatus").textContent = confluenceAccessStatusText(access, invalidReason);
  connectionSelect.disabled = !enabled || state.confluenceBusy || connections.length === 0;
  siteSelect.disabled = !enabled || state.confluenceBusy || sites.length === 0;
  $("confluenceAccessSpaceKey").disabled = !enabled || state.confluenceBusy;
  $("confluenceAccessEnable").disabled = !enabled || state.confluenceBusy || !connection || !siteSelect.value;
  $("confluenceAccessDisable").disabled = !enabled || state.confluenceBusy || !grantEnabled;
}

function confluenceAccessStatusText(access, invalidReason) {
  if (!state.missionId) return "먼저 미션을 선택하세요.";
  if (!access || !(access.enabled || access.Enabled)) {
    return "Confluence agent search는 꺼져 있습니다. API token 연결이 있는 미션에서만 후보 검색을 켤 수 있습니다.";
  }
  const status = access.status || access.Status || "";
  if (status === "invalid") {
    return `Confluence agent search grant가 더 이상 유효하지 않습니다: ${invalidReason || "connection unavailable"}. 기존 source snapshots는 계속 읽을 수 있습니다.`;
  }
  const connectionID = access.connection_id || access.ConnectionID || "";
  const cloudID = access.cloud_id || access.CloudID || "";
  const spaceKey = access.space_key || access.SpaceKey || "";
  return `Confluence agent search가 이 미션에 켜져 있습니다. connection ${connectionID}, site ${cloudID}${spaceKey ? `, space ${spaceKey}` : ""} 안에서 후보 검색만 허용합니다.`;
}

async function enableConfluenceAccess() {
  if (!requireMission()) return;
  const connectionID = $("confluenceAccessConnectionSelect").value;
  const cloudID = $("confluenceAccessSiteSelect").value;
  if (!connectionID || !cloudID) {
    showError(new Error("Confluence agent search를 켜려면 Settings 연결과 site를 선택해야 합니다."));
    return;
  }
  setConfluenceBusy(true);
  try {
    const result = await api(`/api/missions/${state.missionId}/connector-access/confluence`, {
      method: "PUT",
      body: {
        enabled: true,
        connection_id: connectionID,
        cloud_id: cloudID,
        space_key: $("confluenceAccessSpaceKey").value.trim()
      }
    });
    state.confluenceAccess = result.access || result.Access || null;
    renderConfluenceAccessControls();
  } catch (err) {
    showError(err);
  } finally {
    setConfluenceBusy(false);
  }
}

async function disableConfluenceAccess() {
  if (!requireMission()) return;
  setConfluenceBusy(true);
  try {
    const result = await api(`/api/missions/${state.missionId}/connector-access/confluence`, {
      method: "DELETE",
      body: {}
    });
    state.confluenceAccess = result.access || result.Access || null;
    renderConfluenceAccessControls();
  } catch (err) {
    showError(err);
  } finally {
    setConfluenceBusy(false);
  }
}

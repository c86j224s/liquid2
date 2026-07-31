(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const timeShort = Plasma.dom.timeShort;
  const sourceLocatorType = (...args) => sources.sourceLocatorType(...args);

  function confluenceSourceInfo(source) {
    const connector = source.Connector || source.connector || {};
    const connectorID = connector.ConnectorID || connector.connector_id || "";
    const connectorType = connector.ConnectorType || connector.connector_type || "";
    if (connectorID !== "confluence" && connectorType !== "confluence_cloud") return null;
    const externalID = connector.ExternalSourceID || connector.external_source_id || "";
    const externalURI = connector.ExternalURI || connector.external_uri || "";
    const parts = String(externalID || "").split(":");
    const raw = source.Locators || source.locators;
    let locator = null;
    try {
      const parsed = raw ? (typeof raw === "string" ? JSON.parse(raw) : raw) : [];
      const locators = Array.isArray(parsed) ? parsed : [parsed];
      locator = locators.find((item) => {
        const locatorType = sourceLocatorType(item);
        return locatorType === "confluence_page_body" || locatorType === "confluence_page_range" || item.partial || item.Partial;
      }) || null;
    } catch (err) {
      locator = null;
    }
    return {
      cloud_id: locator?.cloud_id || locator?.CloudID || (parts.length >= 2 ? parts[0] : ""),
      site_url: locator?.site_url || locator?.SiteURL || "",
      page_id: locator?.page_id || locator?.PageID || (parts.length >= 2 ? parts.slice(1).join(":") : externalID),
      external_uri: externalURI,
      version: connector.ExternalVersion || connector.external_version || ""
    };
  }

  function confluenceUpdateState(source) {
    const sourceState = source.State || source.state || {};
    return sourceState.ConfluenceUpdate || sourceState.confluence_update || null;
  }

  function confluenceUpdateText(update) {
    if (!update) return "";
    const status = update.status || update.Status || "";
    const checkedAt = timeShort(update.checked_at || update.CheckedAt || "");
    const currentVersion = update.current_version || update.CurrentVersion || 0;
    const latestVersion = update.latest_version || update.LatestVersion || 0;
    const category = update.error_category || update.ErrorCategory || "";
    let result = "";
    if (status === "current") {
      result = `확인 당시 v${latestVersion || currentVersion || "?"} 최신`;
    } else if (status === "update_available") {
      result = `확인 당시 v${latestVersion || "?"} 사용 가능`;
    } else if (status === "check_failed") {
      result = confluenceUpdateFailureText(category);
    }
    if (!result) return "";
    return `${checkedAt ? checkedAt + " · " : ""}${result}`;
  }

  function confluenceUpdateFailureText(category) {
    switch (category) {
      case "confluence_auth": return "마지막 확인 실패 · 인증 필요";
      case "confluence_permission": return "마지막 확인 실패 · 접근 권한 확인 필요";
      case "confluence_not_found": return "마지막 확인 실패 · 원본을 찾거나 접근할 수 없음";
      case "confluence_rate_limited": return "마지막 확인 실패 · 요청 제한";
      default: return "마지막 확인 실패 · Confluence에 연결할 수 없음";
    }
  }

  function confluenceSourceText(info) {
    const parts = [];
    const externalURI = confluenceDisplayableExternalURI(info.external_uri);
    const siteHost = confluenceExternalURIHost(info.site_url || externalURI);
    if (siteHost) parts.push(`site ${siteHost}`);
    if (info.page_id) parts.push(`page ${info.page_id}`);
    if (info.version) parts.push(`v${info.version}`);
    if (externalURI) parts.push(externalURI);
    return parts.join(" / ");
  }

  function confluenceDisplayableExternalURI(uri) {
    try {
      const parsed = new URL(uri);
      return parsed.protocol === "https:" || parsed.protocol === "http:" ? uri : "";
    } catch (err) {
      return "";
    }
  }

  function confluenceExternalURIHost(uri) {
    try {
      return new URL(uri).host;
    } catch (err) {
      return "";
    }
  }

  Object.assign(sources, {
    confluenceSourceInfo,
    confluenceUpdateState,
    confluenceUpdateText,
    confluenceUpdateFailureText,
    confluenceSourceText,
    confluenceDisplayableExternalURI,
    confluenceExternalURIHost
  });
})(window.Plasma);

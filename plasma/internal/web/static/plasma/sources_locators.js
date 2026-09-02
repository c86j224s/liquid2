(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const formatBytes = Plasma.dom.formatBytes;
  const confluenceDisplayableExternalURI = (...args) => sources.confluenceDisplayableExternalURI(...args);
  const safeExternalSourceURL = (...args) => sources.safeExternalSourceURL(...args);

  function sourceDetailPayload(source, confluence) {
    if (!confluence) return source;
    const connector = source.Connector || source.connector || {};
    const access = source.Access || source.access || {};
    const sourceState = source.State || source.state || {};
    const displayURI = confluenceDisplayableExternalURI(confluence.external_uri);
    const detail = {
      type: "confluence_source",
      snapshot_id: source.SnapshotID || source.snapshot_id || "",
      title: source.Title || source.title || "",
      connector_id: connector.ConnectorID || connector.connector_id || "",
      connector_version: connector.ConnectorVersion || connector.connector_version || "",
      site_url: confluence.site_url || "",
      page_id: confluence.page_id || "",
      version: confluence.version || "",
      retrieval_policy: access.RetrievalPolicy || access.retrieval_policy || source.retrieval_policy || "",
      state: sourceState.state || sourceState.State || (sourceState.removed || sourceState.Removed ? "removed" : "active")
    };
    if (displayURI) detail.external_uri = displayURI;
    return detail;
  }

  function localPathLocator(source) {
    const raw = source.Locators || source.locators;
    if (!raw) return null;
    try {
      const parsed = typeof raw === "string" ? JSON.parse(raw) : raw;
      const locators = Array.isArray(parsed) ? parsed : [parsed];
      return locators.find((locator) => sourceLocatorType(locator) === "local_path") || null;
    } catch (err) {
      return null;
    }
  }

  function mediaLocator(source) {
    const raw = source.Locators || source.locators;
    if (!raw) return null;
    try {
      const parsed = typeof raw === "string" ? JSON.parse(raw) : raw;
      const locators = Array.isArray(parsed) ? parsed : [parsed];
      const locator = locators.find((item) => {
        if (sourceLocatorType(item) === "media") return true;
        if (sourceConnectorType(source) !== "file_upload" || sourceLocatorType(item) !== "file_upload") return false;
        return uploadedFileContentKind(item) === "image" || uploadedFileMediaType(item).startsWith("image/");
      });
      if (!locator) return null;
      return {
        media_kind: locator.media_kind || locator.MediaKind || (uploadedFileContentKind(locator) === "image" ? "image" : ""),
        filename: uploadedFileFilename(locator),
        mime_type: locator.mime_type || locator.MIMEType || locator.media_type || locator.MediaType || "",
        byte_size: locator.byte_size || locator.ByteSize || 0,
        width: locator.width || locator.Width || 0,
        height: locator.height || locator.Height || 0,
        canonical_url: locator.canonical_url || locator.CanonicalURL || "",
        direct_media_url: locator.direct_media_url || locator.DirectMediaURL || "",
        license: locator.license || locator.License || "",
        attribution: locator.attribution || locator.Attribution || "",
        inspection_support: locator.inspection_support || locator.InspectionSupport || ""
      };
    } catch (err) {
      return null;
    }
  }

  function sourceLocatorType(locator) {
    return locator.locator_type || locator.LocatorType || locator.kind || locator.Kind || "";
  }

  function sourceConnectorType(source) {
    const connector = source.Connector || source.connector || {};
    return connector.ConnectorType || connector.connector_type || "";
  }

  function uploadedFileContentKind(locator) {
    return locator.content_kind || locator.ContentKind || "";
  }

  function uploadedFileMediaType(locator) {
    return locator.mime_type || locator.MIMEType || locator.media_type || locator.MediaType || "";
  }

  function uploadedFileFilename(locator) {
    return locator.sanitized_filename || locator.SanitizedFilename || locator.original_filename || locator.OriginalFilename || locator.filename || locator.Filename || "";
  }

  function mediaSourceLabel(locator) {
    switch (locator.media_kind) {
      case "image": return "이미지";
      case "audio": return "오디오";
      case "video": return "영상";
      default: return "미디어";
    }
  }

  function mediaSourceText(locator) {
    const parts = [];
    if (locator.mime_type) parts.push(locator.mime_type);
    if (locator.width && locator.height) parts.push(`${locator.width}×${locator.height}`);
    if (locator.byte_size) parts.push(formatBytes(locator.byte_size));
    if (locator.inspection_support === "inspect_unsupported") parts.push("inspect 미지원");
    if (locator.inspection_support === "metadata_only_until_vision_engine_configured") parts.push("이미지 분석 미설정");
    const url = locator.canonical_url || locator.direct_media_url || locator.filename || "";
    return `${parts.join(" · ")}${parts.length && url ? " / " : ""}${url}`;
  }

  function sourceOriginalLink(source, confluence) {
    const connector = source.Connector || source.connector || {};
    const connectorID = String(connector.ConnectorID || connector.connector_id || "").toLowerCase();
    const connectorType = String(connector.ConnectorType || connector.connector_type || "").toLowerCase();
    if (confluence) {
      const url = safeExternalSourceURL(confluence.web_url);
      const siteURL = safeExternalSourceURL(confluence.site_url);
      if (!url || !siteURL || new URL(url).hostname !== new URL(siteURL).hostname) return null;
      return { url, label: "Confluence에서 열기" };
    }
    if (connectorID === "liquid2" || connectorType === "liquid2") {
      for (const locator of sourceLocatorList(source)) {
        const url = safeExternalSourceURL(locator.source_uri || locator.SourceURI || "");
        if (url) return { url, label: "원문 열기" };
      }
      return null;
    }
    const url = safeExternalSourceURL(connector.ExternalURI || connector.external_uri || "");
    return url ? { url, label: "원문 열기" } : null;
  }

  function sourceDownloadExcluded(source) {
    const connector = source.Connector || source.connector || {};
    const connectorID = String(connector.ConnectorID || connector.connector_id || "").toLowerCase();
    const connectorType = String(connector.ConnectorType || connector.connector_type || "").toLowerCase();
    return connectorID === "confluence" || connectorType === "confluence_cloud" || connectorID === "liquid2" || connectorType === "liquid2";
  }

  function sourceRetrievalLabel(source) {
    for (const locator of sourceLocatorList(source)) {
      if (String(locator.retrieval_method || locator.RetrievalMethod || "").toLowerCase() === "browser_render") {
        return "브라우저 렌더링 저장본";
      }
    }
    return "";
  }

  function sourceLocatorList(source) {
    const raw = source.Locators || source.locators;
    if (!raw) return [];
    try {
      const parsed = typeof raw === "string" ? JSON.parse(raw) : raw;
      return Array.isArray(parsed) ? parsed : [parsed];
    } catch (err) {
      return [];
    }
  }

  Object.assign(sources, {
    sourceDetailPayload,
    localPathLocator,
    mediaLocator,
    sourceLocatorType,
    sourceConnectorType,
    uploadedFileContentKind,
    uploadedFileMediaType,
    uploadedFileFilename,
    mediaSourceLabel,
    mediaSourceText,
    sourceOriginalLink,
    sourceDownloadExcluded,
    sourceRetrievalLabel,
    sourceLocatorList
  });
})(window.Plasma);

(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const normalizeSourceURL = sources.normalizeSourceURL;

  function acceptedSourceCandidateKeys(sources) {
    const keys = new Set();
    for (const source of sources || []) {
      const connector = source.Connector || source.connector || {};
      for (const value of [connector.ExternalURI, connector.external_uri, connector.ExternalSourceID, connector.external_source_id]) {
        const normalized = normalizeSourceURL(value);
        if (normalized) keys.add(normalized);
      }
      for (const locator of sourceLocators(source)) {
        const key = confluenceSourceKey(locator.site_url || locator.SiteURL || "", locator.page_id || locator.PageID || "");
        if (key) keys.add(key);
      }
    }
    return keys;
  }

  function sourceCandidateAccepted(existingKeys, normalizedURL) {
    if (!normalizedURL) return false;
    return existingKeys.has(normalizedURL) || existingKeys.has(confluenceCandidateKeyFromURL(normalizedURL));
  }

  function sourceLocators(source) {
    const raw = source?.Locators ?? source?.locators;
    if (!raw) return [];
    if (Array.isArray(raw)) return raw;
    if (typeof raw === "string") {
      try {
        const parsed = JSON.parse(raw);
        return Array.isArray(parsed) ? parsed : [];
      } catch (err) {
        return [];
      }
    }
    return [];
  }

  function confluenceCandidateKeyFromURL(rawURL) {
    try {
      const url = new URL(rawURL);
      const queryID = url.searchParams.get("pageId");
      if (/^\d+$/.test(queryID || "")) return confluenceSourceKey(`${url.protocol}//${url.host}`, queryID);
      const segments = url.pathname.split("/").filter(Boolean);
      for (let index = 0; index + 1 < segments.length; index++) {
        const marker = decodeURIComponent(segments[index]).toLowerCase();
        if (marker !== "pages" && marker !== "edit-v2") continue;
        const pageID = decodeURIComponent(segments[index + 1]);
        if (/^\d+$/.test(pageID)) return confluenceSourceKey(`${url.protocol}//${url.host}`, pageID);
      }
      return "";
    } catch (err) {
      return "";
    }
  }

  function confluenceSourceKey(siteURL, pageID) {
    try {
      const site = new URL(siteURL);
      const id = String(pageID || "").trim();
      if (!id) return "";
      return `confluence:${site.hostname.toLowerCase()}:${id}`;
    } catch (err) {
      return "";
    }
  }

  function sourceCandidatesFromEvents(events) {
    const byURL = new Map();
    const staging = sourceCandidateStagingByProposal(events);
    for (const event of events) {
      if (event.EventType !== "source.candidate.proposed") continue;
      const payload = event.Payload || {};
      for (const candidate of payload.candidates || []) {
        const url = normalizeSourceURL(candidate.url || candidate.URL || "");
        const reason = candidate.reason || candidate.Reason || "";
        if (!url || !String(reason).trim()) continue;
        const sequence = Number(event.Sequence || 0);
        const existing = byURL.get(url);
        if (existing && existing.sequence > sequence) continue;
        byURL.set(url, {
          url,
          title: candidate.title || candidate.Title || "",
          reason,
          eventID: event.EventID,
          sequence,
          userEventID: payload.user_event_id || "",
          agentEventID: payload.agent_event_id || "",
          staging: staging.get(url) || null
        });
      }
    }
    return [...byURL.values()];
  }

  function sourceCandidateStagingByProposal(events) {
    const byURL = new Map();
    const latestStarts = new Map();
    for (const event of events) {
      if (event.EventType !== "source.candidate.staging_started") continue;
      const payload = event.Payload || {};
      const url = normalizeSourceURL(payload.url || payload.URL || "");
      if (!url) continue;
      const existing = latestStarts.get(url);
      if (!existing || Number(existing.sequence || 0) <= Number(event.Sequence || 0)) {
        latestStarts.set(url, { eventID: event.EventID, sequence: Number(event.Sequence || 0) });
      }
    }
    for (const event of events) {
      if (!["source.candidate.staging_started", "source.candidate.staged", "source.candidate.staging_failed"].includes(event.EventType)) continue;
      const payload = event.Payload || {};
      const proposalEventID = payload.proposal_event_id || "";
      const url = normalizeSourceURL(payload.url || payload.URL || "");
      if (!url) continue;
      const sequence = Number(event.Sequence || 0);
      const latestStart = latestStarts.get(url);
      const attemptID = String(payload.staging_event_id || "").trim();
      if (event.EventType !== "source.candidate.staging_started" && latestStart &&
          (!attemptID || attemptID !== latestStart.eventID)) continue;
      const state = event.EventType === "source.candidate.staged" ? "staged" : event.EventType === "source.candidate.staging_failed" ? "failed" : "fetching";
      const record = { state, eventID: event.EventID, sequence, artifactID: payload.artifact_id || "", proposalEventID, url, stagedTerminalEventID: event.EventID, message: payload.message || "", browserRenderCandidate: payload.browser_render_candidate || payload.browserRenderCandidate || null };
      const existing = byURL.get(url);
      if (!existing || Number(existing.sequence || 0) <= sequence) byURL.set(url, record);
    }
    return byURL;
  }

  function sourceCandidateStagingLabel(staging) {
    if (!staging) return "";
    const browserRender = sourceCandidateBrowserRenderLabel(staging.browserRenderCandidate);
    if (staging.state === "staged") {
      return `<strong>본문 상태</strong> 미승인 후보 본문 준비됨${browserRender}`;
    }
    if (staging.state === "failed") {
      return `<strong>본문 상태</strong> 가져오기 실패${staging.message ? ` · ${escapeHTML(staging.message)}` : ""}`;
    }
    return `<strong>본문 상태</strong> 가져오는 중`;
  }

  function sourceCandidateBrowserRenderLabel(diagnosis) {
    if (!diagnosis || diagnosis.candidate !== true) return "";
    const visible = Number(diagnosis.visible_text_length || 0);
    const suffix = visible > 0 ? ` · 본문 ${visible}자` : "";
    return ` · <span class="source-candidate-diagnostic">브라우저 렌더링 후보${suffix}</span>`;
  }

  function confluenceCandidateKeyFromLegacyConnector(connector) {
    const type = String(connector?.connector_type || connector?.ConnectorType || "").trim().toLowerCase();
    const id = String(connector?.connector_id || connector?.ConnectorID || "").trim().toLowerCase();
    if (type !== "confluence_cloud" && id !== "confluence") return "";
    const raw = String(connector?.external_source_id || connector?.ExternalSourceID || "").trim();
    if (!raw.toLowerCase().startsWith("site_")) return "";
    const separator = raw.indexOf(":", 5);
    if (separator <= 5 || separator + 1 >= raw.length) return "";
    const host = raw.slice(5, separator).toLowerCase();
    const pageID = raw.slice(separator + 1).trim();
    if (!host || !pageID || host.includes("/") || host.includes("\\")) return "";
    return `confluence:${host}:${pageID}`;
  }

  function sourceCandidateSnapshotConsumption(events) {
    const consumedURLs = new Set();
    const consumedConfluenceKeys = new Set();

    for (const event of events || []) {
      if (event.EventType !== "source.snapshotted") continue;
      const payload = event.Payload || {};
      if (typeof payload !== "object" || Array.isArray(payload)) continue;
      const connector = payload.connector || payload.Connector || {};
      if (typeof connector !== "object" || Array.isArray(connector)) continue;
      const legacyKey = confluenceCandidateKeyFromLegacyConnector(connector);
      if (legacyKey) consumedConfluenceKeys.add(legacyKey);
      for (const value of [payload.url, payload.URL, connector.external_uri, connector.ExternalURI, connector.external_source_id, connector.ExternalSourceID]) {
        const normalized = normalizeSourceURL(value);
        if (normalized) consumedURLs.add(normalized);
      }
    }
    return { consumedURLs, consumedConfluenceKeys };
  }

  function sourceCandidateConsumedBySnapshot(consumption, candidate) {
    const normalized = normalizeSourceURL(candidate.url);
    return (normalized && consumption.consumedURLs.has(normalized)) || consumption.consumedConfluenceKeys.has(confluenceCandidateKeyFromURL(normalized));
  }

  function sourceCandidateDownloadExcluded(candidate) {
    const normalized = normalizeSourceURL(candidate?.url || "");
    if (!normalized) return true;
    try {
      const host = new URL(normalized).hostname.toLowerCase();
      return (host === "atlassian.net" || host.endsWith(".atlassian.net")) && Boolean(confluenceCandidateKeyFromURL(normalized));
    } catch (err) {
      return true;
    }
  }

  function sourceCandidateDecisions(events) {
    const decisions = new Map();
    for (const event of events) {
      if (event.EventType !== "source.candidate.rejected" && event.EventType !== "source.candidate.restored") continue;
      const url = normalizeSourceURL(event.Payload?.url || event.Payload?.URL || "");
      if (!url) continue;
      decisions.set(url, {
        state: event.EventType === "source.candidate.rejected" ? "rejected" : "restored",
        reason: event.Payload?.reason || "",
        eventID: event.EventID,
        sequence: event.Sequence
      });
    }
    return decisions;
  }

  function sourceCandidateTitleForURL(url) {
    const normalized = normalizeSourceURL(url);
    if (!normalized) return "";
    const candidates = sourceCandidatesFromEvents(state.detail?.events || []);
    const match = candidates.find((candidate) => normalizeSourceURL(candidate.url) === normalized);
    return match?.title || "";
  }

  Object.assign(sources, {
    acceptedSourceCandidateKeys,
    sourceCandidateAccepted,
    sourceLocators,
    confluenceCandidateKeyFromURL,
    confluenceSourceKey,
    sourceCandidatesFromEvents,
    sourceCandidateStagingByProposal,
    sourceCandidateStagingLabel,
    sourceCandidateBrowserRenderLabel,
    sourceCandidateDecisions,
    sourceCandidateSnapshotConsumption,
    sourceCandidateConsumedBySnapshot,
    sourceCandidateDownloadExcluded,
    sourceCandidateTitleForURL
  });
})(window.Plasma);

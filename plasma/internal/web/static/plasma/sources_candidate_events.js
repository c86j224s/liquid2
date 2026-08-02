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
      const segments = url.pathname.split("/").filter(Boolean);
      const index = segments.findIndex((segment) => segment.toLowerCase() === "pages");
      if (index < 0 || index + 1 >= segments.length) return "";
      return confluenceSourceKey(`${url.protocol}//${url.host}`, decodeURIComponent(segments[index + 1]));
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
          staging: staging.get(url) || staging.get(event.EventID) || null
        });
      }
    }
    return [...byURL.values()];
  }

  function sourceCandidateStagingByProposal(events) {
    const byKey = new Map();
    for (const event of events) {
      if (!["source.candidate.staging_started", "source.candidate.staged", "source.candidate.staging_failed"].includes(event.EventType)) continue;
      const payload = event.Payload || {};
      const proposalEventID = payload.proposal_event_id || "";
      const url = normalizeSourceURL(payload.url || payload.URL || "");
      const sequence = Number(event.Sequence || 0);
      const state = event.EventType === "source.candidate.staged"
      ? "staged"
      : event.EventType === "source.candidate.staging_failed"
      ? "failed"
      : "fetching";
      const record = {
        state,
        eventID: event.EventID,
        sequence,
        artifactID: payload.artifact_id || "",
        message: payload.message || "",
        browserRenderCandidate: payload.browser_render_candidate || payload.browserRenderCandidate || null
      };
      for (const key of [proposalEventID, url].filter(Boolean)) {
        const existing = byKey.get(key);
        if (!existing || Number(existing.sequence || 0) <= sequence) {
          byKey.set(key, record);
        }
      }
    }
    return byKey;
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
    sourceCandidateTitleForURL
  });
})(window.Plasma);

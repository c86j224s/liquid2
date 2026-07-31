(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const missionApi = Plasma.transport.missionApi;
  const captureMissionSelection = Plasma.mission.captureMissionSelection;
  const ownsMissionSelection = Plasma.mission.ownsMissionSelection;
  const StaleMissionOperationError = Plasma.mission.StaleMissionOperationError;
  const isStaleMissionOperation = Plasma.mission.isStaleMissionOperation;
  const normalizeSourceURL = sources.normalizeSourceURL;
  const refreshSourceCandidates = (...args) => sources.refreshSourceCandidates(...args);
  const requireMission = () => sources.dependency("requireMission")();
  const reloadMission = (...args) => sources.dependency("reloadMission")(...args);
  const showError = (...args) => sources.dependency("showError")(...args);

  async function addURLSource(url, title = "", owner = captureMissionSelection()) {
    if (!requireMission()) return;
    const key = normalizeSourceURL(url) || url;
    if (!ownsMissionSelection(owner)) return false;
    if (state.sourceCandidateBusy.has(key)) return false;
    state.sourceCandidateBusy.add(key);
    refreshSourceCandidates();
    try {
      const route = sourceRouteForURL(url);
      try {
        if (!ownsMissionSelection(owner)) throw new StaleMissionOperationError();
        await missionApi(owner, `/sources/${route}`, {
          method: "POST",
          body: { url, title }
        });
      } catch (err) {
        if (isStaleMissionOperation(err)) throw err;
        if (route !== "url" || !looksLikePDFSourceError(err)) throw err;
        if (!ownsMissionSelection(owner)) throw new StaleMissionOperationError();
        await missionApi(owner, "/sources/pdf_url", {
          method: "POST",
          body: { url, title }
        });
      }
      if (!ownsMissionSelection(owner)) return false;
      await reloadMission(owner.missionId);
      return true;
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
      return false;
    } finally {
      if (ownsMissionSelection(owner)) {
        state.sourceCandidateBusy.delete(key);
        refreshSourceCandidates();
      }
    }
  }

  function looksLikePDFSourceError(err) {
    const message = `${err?.userMessage || ""} ${err?.message || ""}`.toLowerCase();
    return message.includes("application/pdf") || message.includes("pdf source") || message.includes("pdf");
  }

  function sourceRouteForURL(value) {
    if (looksLikeConfluenceURL(value)) return "confluence/url";
    if (looksLikePDFURL(value)) return "pdf_url";
    if (looksLikeMediaURL(value)) return "media_url";
    return "url";
  }

  function looksLikeConfluenceURL(value) {
    try {
      const url = new URL(value);
      const host = url.hostname.toLowerCase();
      if (host !== "atlassian.net" && !host.endsWith(".atlassian.net")) return false;
      const path = url.pathname.replace(/\/+$/, "");
      return path === "" || path === "/wiki" || path.startsWith("/wiki/");
    } catch (err) {
      return false;
    }
  }

  function looksLikePDFURL(value) {
    try {
      const url = new URL(value);
      return /\.pdf$/i.test(url.pathname);
    } catch (err) {
      return false;
    }
  }

  function looksLikeMediaURL(value) {
    try {
      const url = new URL(value);
      const path = url.pathname.toLowerCase();
      return /\.(png|jpe?g|gif|mp3|m4a|wav|ogg|mp4|mov|webm|m4v)$/.test(path);
    } catch (err) {
      return false;
    }
  }

  Object.assign(sources, {
    addURLSource,
    looksLikePDFSourceError,
    sourceRouteForURL,
    looksLikeConfluenceURL,
    looksLikePDFURL,
    looksLikeMediaURL
  });
})(window.Plasma);

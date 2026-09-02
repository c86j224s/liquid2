(function (Plasma) {
  "use strict";

  const dependencies = {};

  function configure(values) {
    Object.assign(dependencies, values || {});
  }

  function dependency(name) {
    const value = dependencies[name];
    if (!value) throw new Error("Plasma.sources dependency missing: " + name);
    return value;
  }

  function normalizeSourceURL(value) {
    const raw = String(value || "");
    if (/[\x00-\x1f\x7f]/.test(raw)) return "";
    const text = raw.trim();
    if (!text) return "";
    try {
      const url = new URL(text);
      if (
        (url.protocol !== "http:" && url.protocol !== "https:") ||
        url.username ||
        url.password
      ) return "";
      url.protocol = url.protocol.toLowerCase();
      url.hostname = url.hostname.toLowerCase();
      url.hash = "";
      return url.toString();
    } catch (err) {
      return "";
    }
  }

  function safeExternalSourceURL(value) {
    const raw = String(value || "");
    if (/[\x00-\x1f\x7f]/.test(raw)) return "";
    const text = raw.trim();
    if (!text) return "";
    try {
      const url = new URL(text);
      if (
        (url.protocol !== "http:" && url.protocol !== "https:") ||
        url.username ||
        url.password
      ) return "";
      url.protocol = url.protocol.toLowerCase();
      url.hostname = url.hostname.toLowerCase();
      return url.toString();
    } catch (err) {
      return "";
    }
  }

  Plasma.sources = { configure, dependency, normalizeSourceURL, safeExternalSourceURL };
})(window.Plasma);

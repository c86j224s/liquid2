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
    const text = String(value || "").trim();
    if (!text) return "";
    try {
      const url = new URL(text);
      if (url.protocol !== "http:" && url.protocol !== "https:") return "";
      url.protocol = url.protocol.toLowerCase();
      url.hostname = url.hostname.toLowerCase();
      url.hash = "";
      return url.toString();
    } catch (err) {
      return "";
    }
  }

  Plasma.sources = { configure, dependency, normalizeSourceURL };
})(window.Plasma);

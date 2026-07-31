(function (Plasma) {
  "use strict";

  function $(id) {
    return document.getElementById(id);
  }

  function shortID(value) {
    const text = String(value || "");
    if (text.length <= 12) return text;
    return `${text.slice(0, 8)}…${text.slice(-4)}`;
  }

  function timeShort(value) {
    if (!value) return "";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return date.toLocaleString();
  }

  function escapeHTML(value) {
    return String(value ?? "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#039;");
  }

  function escapeAttr(value) {
    return escapeHTML(value).replaceAll("`", "&#096;");
  }

  function formatBytes(value) {
    const bytes = Number(value || 0);
    if (!Number.isFinite(bytes) || bytes <= 0) return "";
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`;
    return `${(bytes / 1024 / 1024).toFixed(1)} MiB`;
  }

  Plasma.dom = {
    $,
    shortID,
    timeShort,
    escapeHTML,
    escapeAttr,
    formatBytes
  };
})(window.Plasma);

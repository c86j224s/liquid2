(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const $ = Plasma.dom.$;

  function showError(err) {
    state.lastError = err.userMessage || err.message || String(err);
    if (!err.userMessage && err.stack) {
      state.lastError = err.stack;
    }
    if (err.details && !err.userMessage) {
      state.lastError += "\n\n" + JSON.stringify(err.details, null, 2);
    }
    $("errorText").textContent = state.lastError;
    // Transient connection failures (e.g. on browser wake / session recovery, or
    // a brief server restart) are surfaced by the health badge, not a toast.
    if (err && err.isNetworkError) {
      const badge = $("healthBadge");
      if (badge) badge.textContent = "연결 끊김";
      return;
    }
    $("errorToast").classList.remove("hidden");
  }

  function hideError() {
    $("errorToast").classList.add("hidden");
  }

  async function copyError() {
    try {
      await copyText(state.lastError || $("errorText").textContent);
    } catch (err) {
      $("errorText").textContent += "\n\nclipboard copy failed: " + err.message;
    }
  }

  async function copyText(value) {
    const text = String(value ?? "");
    // Only use the async Clipboard API in a secure context; over plain HTTP
    // (e.g. a non-loopback dev server) it is unavailable or rejects, so fall back
    // to the execCommand path instead of failing.
    if (window.isSecureContext && navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(text);
        return;
      } catch {
        /* fall through to the textarea fallback */
      }
    }
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.setAttribute("readonly", "");
    textarea.style.position = "fixed";
    textarea.style.left = "-9999px";
    document.body.appendChild(textarea);
    textarea.select();
    const copied = document.execCommand("copy");
    document.body.removeChild(textarea);
    if (!copied) {
      throw new Error("clipboard API is not available");
    }
  }

  Object.assign(Plasma.ui, {
    showError,
    hideError,
    copyError,
    copyText
  });

  window.showError = showError;
  window.hideError = hideError;
  window.copyError = copyError;
  window.copyText = copyText;
})(window.Plasma);

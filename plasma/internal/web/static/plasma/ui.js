(function (Plasma) {
  "use strict";

  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;

  function setSectionEmpty(el, isEmpty) {
    if (!el) return;
    el.classList.toggle("collapsed-empty", isEmpty);
    el.classList.toggle("hidden", isEmpty);
  }

  function updateCountChip(id, n) {
    const el = $(id);
    if (!el) return;
    el.textContent = n > 0 ? String(n) : "";
  }

  function empty(text) {
    return `<div class="item"><div class="item-meta">${escapeHTML(text)}</div></div>`;
  }

  function setElementDisabled(id, disabled) {
    const el = $(id);
    if (el) el.disabled = Boolean(disabled);
    return el;
  }

  function setFormButtonsDisabled(formId, disabledForButton) {
    const form = $(formId);
    if (!form) return;
    for (const button of form.querySelectorAll("button")) {
      button.disabled = Boolean(disabledForButton(button));
    }
  }

  function setButtonText(id, text) {
    const el = $(id);
    if (el) el.textContent = text;
  }

  Plasma.ui = {
    setSectionEmpty,
    updateCountChip,
    empty,
    setElementDisabled,
    setFormButtonsDisabled,
    setButtonText
  };

  window.setSectionEmpty = setSectionEmpty;
  window.updateCountChip = updateCountChip;
  window.empty = empty;
})(window.Plasma);

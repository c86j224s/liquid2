(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const copyText = Plasma.ui.copyText;
  const showError = Plasma.ui.showError;

  const detailHooks = {
    beforeLeave: null,
    copyContent: null
  };

  function configureDetailHooks(hooks = {}) {
    detailHooks.beforeLeave = typeof hooks.beforeLeave === "function" ? hooks.beforeLeave : null;
    detailHooks.copyContent = typeof hooks.copyContent === "function" ? hooks.copyContent : null;
  }

  function showDetail(title, value) {
    $("detailTitle").textContent = title;
    const text = typeof value === "string" ? value : JSON.stringify(value, null, 2);
    state.detailText = text;
    $("detailBody").innerHTML = `<pre>${escapeHTML(text)}</pre>`;
    openDetailModal();
  }

  function openDetailModal(wide) {
    const card = $("detailModal").querySelector(".modal-card");
    if (card && wide !== undefined) card.classList.toggle("modal-card--wide", Boolean(wide));
    $("detailModal").classList.remove("hidden");
  }

  async function copyDetail() {
    try {
      await copyText(detailHooks.copyContent?.() || state.detailText || $("detailBody").textContent);
    } catch (err) {
      showError(err);
    }
  }

  function hideDetail() {
    if (detailHooks.beforeLeave && !detailHooks.beforeLeave()) return false;
    $("detailModal").classList.add("hidden");
    const card = $("detailModal").querySelector(".modal-card");
    if (card) card.classList.remove("modal-card--wide");
    disableDetailScrollRatio();
    return true;
  }

  function enableDetailScrollRatio() {
    state.detailScrollRatioEnabled = true;
    $("detailBody").scrollTop = 0;
    requestAnimationFrame(updateDetailScrollRatio);
  }

  function disableDetailScrollRatio() {
    state.detailScrollRatioEnabled = false;
    const ratio = $("detailPositionRatio");
    if (!ratio) return;
    ratio.classList.add("hidden");
    ratio.textContent = "";
  }

  function updateDetailScrollRatio() {
    const ratio = $("detailPositionRatio");
    const body = $("detailBody");
    if (!ratio || !body || !state.detailScrollRatioEnabled || $("detailModal").classList.contains("hidden")) {
      if (ratio) ratio.classList.add("hidden");
      return;
    }
    const scroll = detailScrollPosition();
    const maxScroll = scroll.maxScroll;
    if (maxScroll <= 1) {
      ratio.classList.add("hidden");
      ratio.textContent = "";
      return;
    }
    const percent = Math.round((scroll.scrollTop / maxScroll) * 100);
    ratio.textContent = `위치 ${Math.max(0, Math.min(100, percent))}%`;
    ratio.classList.remove("hidden");
  }

  function detailScrollPosition() {
    const body = $("detailBody");
    return {
      scrollTop: body?.scrollTop || 0,
      maxScroll: Math.max(0, (body?.scrollHeight || 0) - (body?.clientHeight || 0))
    };
  }

  function onDetailModalClick(event) {
    if (event.target.id === "detailModal") hideDetail();
  }

  function onDetailButtonClick(event) {
    const button = event.target.closest("[data-detail-json]");
    if (!button) return false;
    const title = button.dataset.detailTitle || "상세 보기";
    try {
      showDetail(title, JSON.parse(button.dataset.detailJson));
    } catch (err) {
      showDetail(title, button.dataset.detailJson || "");
    }
    return true;
  }

  Object.assign(Plasma.ui, {
    configureDetailHooks,
    showDetail,
    openDetailModal,
    copyDetail,
    hideDetail,
    enableDetailScrollRatio,
    disableDetailScrollRatio,
    updateDetailScrollRatio,
    detailScrollPosition,
    onDetailModalClick,
    onDetailButtonClick
  });

  window.showDetail = showDetail;
  window.openDetailModal = openDetailModal;
  window.copyDetail = copyDetail;
  window.hideDetail = hideDetail;
  window.enableDetailScrollRatio = enableDetailScrollRatio;
  window.disableDetailScrollRatio = disableDetailScrollRatio;
  window.updateDetailScrollRatio = updateDetailScrollRatio;
  window.detailScrollPosition = detailScrollPosition;
  window.onDetailModalClick = onDetailModalClick;
})(window.Plasma);

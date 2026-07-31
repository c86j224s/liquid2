	(function imageViewerModule() {
	  const MIN_ZOOM = 0.1;
  const MAX_ZOOM = 8;
  const ZOOM_STEP = 1.25;
  let modal;
  let stage;
  let image;
  let title;
  let zoomLabel;
  let legend;
  let zoom = 1;
  let naturalWidth = 0;
  let naturalHeight = 0;

  function clampZoom(value) {
    return Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, value));
  }

  function imageLabel(details) {
    return String(details.alt || details.title || (details.kind === "svg" ? "Mermaid 그래프" : "이미지")).trim() || "이미지";
  }

  function ensureModal() {
    if (modal) return modal;
    modal = document.createElement("div");
    modal.id = "imageViewerModal";
    modal.className = "image-viewer-modal hidden";
    modal.setAttribute("role", "dialog");
    modal.setAttribute("aria-modal", "true");
    modal.setAttribute("aria-labelledby", "imageViewerTitle");
    modal.innerHTML = `
      <div class="image-viewer-card">
        <div class="image-viewer-head">
          <div id="imageViewerTitle" class="image-viewer-title">이미지 보기</div>
          <div class="image-viewer-tools" role="group" aria-label="이미지 확대 도구">
            <button type="button" class="icon-button quiet" data-image-viewer-action="zoom-out" title="축소" aria-label="축소">−</button>
            <button type="button" class="icon-button quiet" data-image-viewer-action="fit" title="화면에 맞춤" aria-label="화면에 맞춤">⛶</button>
            <button type="button" class="icon-button quiet" data-image-viewer-action="actual" title="원래 크기" aria-label="원래 크기">1:1</button>
            <button type="button" class="icon-button quiet" data-image-viewer-action="zoom-in" title="확대" aria-label="확대">+</button>
            <span class="image-viewer-zoom" aria-live="polite">100%</span>
            <button type="button" class="icon-button quiet" data-image-viewer-action="close" title="닫기" aria-label="닫기">×</button>
          </div>
        </div>
        <div class="image-viewer-stage">
          <img id="imageViewerImage" alt="">
        </div>
        <div class="image-viewer-legend" hidden role="group" aria-label="Mermaid 라인 차트 범례"></div>
      </div>
    `;
    document.body.appendChild(modal);
    stage = modal.querySelector(".image-viewer-stage");
    image = modal.querySelector("#imageViewerImage");
    title = modal.querySelector("#imageViewerTitle");
    zoomLabel = modal.querySelector(".image-viewer-zoom");
    legend = modal.querySelector(".image-viewer-legend");
    modal.addEventListener("click", onModalClick);
    modal.addEventListener("click", onToolClick);
    image.addEventListener("load", fitImageToStage);
    return modal;
  }

  function onModalClick(event) {
    if (event.target === modal) closeImageViewer();
  }

  function onToolClick(event) {
    const button = event.target.closest("[data-image-viewer-action]");
    if (!button) return;
    const action = button.dataset.imageViewerAction;
    if (action === "close") closeImageViewer();
    if (action === "zoom-in") setZoom(zoom * ZOOM_STEP, true);
    if (action === "zoom-out") setZoom(zoom / ZOOM_STEP, true);
    if (action === "actual") setZoom(1, true);
    if (action === "fit") fitImageToStage();
  }

  function onKeyDown(event) {
    if (!modal || modal.classList.contains("hidden")) return;
    if (event.key === "Escape") {
      event.preventDefault();
      closeImageViewer();
    } else if (event.key === "+" || event.key === "=") {
      event.preventDefault();
      setZoom(zoom * ZOOM_STEP, true);
    } else if (event.key === "-") {
      event.preventDefault();
      setZoom(zoom / ZOOM_STEP, true);
    } else if (event.key === "0") {
      event.preventDefault();
      setZoom(1, true);
    } else if (event.key.toLowerCase() === "f") {
      event.preventDefault();
      fitImageToStage();
    }
  }

  function setZoom(nextZoom, keepCenter) {
    if (!image || !stage) return;
    const previousZoom = zoom;
    zoom = clampZoom(nextZoom);
    if (!naturalWidth || !naturalHeight) {
      naturalWidth = image.naturalWidth || image.width || 1;
      naturalHeight = image.naturalHeight || image.height || 1;
    }
    const centerX = stage.scrollLeft + stage.clientWidth / 2;
    const centerY = stage.scrollTop + stage.clientHeight / 2;
    image.style.width = `${Math.max(1, Math.round(naturalWidth * zoom))}px`;
    image.style.height = `${Math.max(1, Math.round(naturalHeight * zoom))}px`;
    zoomLabel.textContent = `${Math.round(zoom * 100)}%`;
    if (keepCenter && previousZoom > 0) {
      const ratio = zoom / previousZoom;
      stage.scrollLeft = Math.max(0, centerX * ratio - stage.clientWidth / 2);
      stage.scrollTop = Math.max(0, centerY * ratio - stage.clientHeight / 2);
    }
  }

  function fitImageToStage() {
    if (!image || !stage) return;
    naturalWidth = image.naturalWidth || image.width || naturalWidth || 1;
    naturalHeight = image.naturalHeight || image.height || naturalHeight || 1;
    const availableWidth = Math.max(1, stage.clientWidth - 36);
    const availableHeight = Math.max(1, stage.clientHeight - 36);
    const fit = Math.min(1, availableWidth / naturalWidth, availableHeight / naturalHeight);
    setZoom(fit, false);
    stage.scrollLeft = 0;
    stage.scrollTop = 0;
  }

  function openImageViewer(details) {
    const svg = String(details?.svg || "").trim();
    const src = svg ? svgDataURL(svg) : String(details?.src || "").trim();
    if (!src) return;
    ensureModal();
    naturalWidth = 0;
    naturalHeight = 0;
    zoom = 1;
    title.textContent = imageLabel(details);
    image.alt = String(details.alt || details.title || "");
    image.removeAttribute("width");
    image.removeAttribute("height");
    image.style.width = "";
    image.style.height = "";
    renderImageViewerLegend(details?.legend);
    image.src = src;
    modal.classList.remove("hidden");
    document.addEventListener("keydown", onKeyDown);
    requestAnimationFrame(() => fitImageToStage());
  }

  function closeImageViewer() {
    if (!modal) return;
    modal.classList.add("hidden");
    renderImageViewerLegend([]);
    image.removeAttribute("src");
    document.removeEventListener("keydown", onKeyDown);
  }

  function renderImageViewerLegend(items) {
    if (!legend) return;
    legend.replaceChildren();
    const entries = Array.isArray(items) ? items.filter((item) => String(item?.label || "").trim()) : [];
    legend.hidden = entries.length === 0;
    entries.forEach((item) => {
      const entry = document.createElement("span");
      entry.className = "image-viewer-legend-item";
      const marker = document.createElement("span");
      marker.className = "image-viewer-legend-marker";
      marker.setAttribute("aria-hidden", "true");
      if (item.color) marker.style.backgroundColor = String(item.color);
      const text = document.createElement("span");
      text.className = "image-viewer-legend-text";
      text.textContent = String(item.label || "").trim();
      entry.append(marker, text);
      legend.appendChild(entry);
    });
  }

  Object.assign(window.Plasma.reports, { openImageViewer });
}());

(function imageViewerModule() {
  const MESSAGE_TYPE = "plasma:image-viewer:open";
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

  function imageDetailsFromElement(img) {
    return {
      src: img.currentSrc || img.src || img.getAttribute("src") || "",
      alt: img.getAttribute("alt") || "",
      title: img.getAttribute("title") || ""
    };
  }

  function imageTarget(img) {
    return img.closest("a") || img.closest("picture") || img;
  }

  function svgDataURL(svg) {
    const text = String(svg || "").trim();
    if (!text) return "";
    return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(text)}`;
  }

  function svgTitle(svg) {
    return svg.getAttribute("aria-label") || svg.querySelector("title")?.textContent || "Mermaid 그래프";
  }

  function legendFromMermaidSVG(svg) {
    const figure = svg.closest(".plasma-mermaid-card");
    const legend = figure?.querySelector(".plasma-mermaid-line-legend");
    if (!legend) return [];
    return Array.from(legend.querySelectorAll(".plasma-mermaid-line-legend-item"))
      .map((item) => {
        const marker = item.querySelector(".plasma-mermaid-line-legend-marker");
        const label = item.querySelector(".plasma-mermaid-line-legend-text")?.textContent || item.textContent || "";
        const color = marker?.style.backgroundColor || (marker ? getComputedStyle(marker).backgroundColor : "");
        return { label: label.replace(/\s+/g, " ").trim(), color };
      })
      .filter((item) => item.label);
  }

  function svgDetailsFromElement(svg) {
    return {
      kind: "svg",
      svg: svg.outerHTML || "",
      alt: svgTitle(svg),
      title: svgTitle(svg),
      legend: legendFromMermaidSVG(svg)
    };
  }

  function enhanceImage(img) {
    if (!img || img.dataset.plasmaImageViewerBound === "true") return;
    const src = img.currentSrc || img.src || img.getAttribute("src") || "";
    if (!src) return;
    const target = imageTarget(img);
    if (!target || !target.parentNode || target.dataset.plasmaImageViewerTarget === "true") return;
    const wrapper = document.createElement("span");
    wrapper.className = "plasma-image-viewer-target";
    target.parentNode.insertBefore(wrapper, target);
    wrapper.appendChild(target);
    target.dataset.plasmaImageViewerTarget = "true";
    img.dataset.plasmaImageViewerBound = "true";
    const button = document.createElement("button");
    button.type = "button";
    button.className = "plasma-image-viewer-open";
    button.title = "이미지 크게 보기";
    button.setAttribute("aria-label", "이미지 크게 보기");
    button.textContent = "⌕";
    button.addEventListener("click", (event) => {
      event.preventDefault();
      event.stopPropagation();
      openImageViewer(imageDetailsFromElement(img));
    });
    wrapper.appendChild(button);
  }

  function mermaidSVGSelector() {
    return [
      ".plasma-mermaid-diagram > svg",
      ".mermaid svg",
      "svg[id^='plasma-mermaid-']",
      "svg[id^='mermaid-']",
      "svg[aria-label='Mermaid 그래프']"
    ].join(",");
  }

  function enhanceMermaidSVG(svg) {
    if (!svg || svg.dataset.plasmaImageViewerBound === "true") return;
    if (!svg.parentNode || svg.dataset.plasmaImageViewerTarget === "true") return;
    const wrapper = document.createElement("span");
    wrapper.className = "plasma-image-viewer-target plasma-image-viewer-target--svg";
    svg.parentNode.insertBefore(wrapper, svg);
    wrapper.appendChild(svg);
    svg.dataset.plasmaImageViewerTarget = "true";
    svg.dataset.plasmaImageViewerBound = "true";
    const button = document.createElement("button");
    button.type = "button";
    button.className = "plasma-image-viewer-open";
    button.title = "Mermaid 그래프 크게 보기";
    button.setAttribute("aria-label", "Mermaid 그래프 크게 보기");
    button.textContent = "⌕";
    button.addEventListener("click", (event) => {
      event.preventDefault();
      event.stopPropagation();
      openImageViewer(svgDetailsFromElement(svg));
    });
    wrapper.appendChild(button);
  }

  function enhancePlasmaImageViewing(root) {
    if (!root) return;
    root.querySelectorAll("img").forEach(enhanceImage);
    root.querySelectorAll(mermaidSVGSelector()).forEach(enhanceMermaidSVG);
  }

  function frameScript() {
    return `
<style>
.plasma-image-viewer-target{position:relative;display:inline-block;max-width:100%;vertical-align:top}
.plasma-image-viewer-target img{display:block;max-width:100%;height:auto}
.plasma-image-viewer-target>svg{display:block;max-width:100%;height:auto}
.plasma-image-viewer-open{position:absolute;top:8px;right:8px;z-index:2147483647;display:inline-flex;align-items:center;justify-content:center;width:30px;height:30px;border:1px solid rgba(255,255,255,.42);border-radius:999px;background:rgba(17,19,24,.78);color:#fff;box-shadow:0 8px 22px rgba(0,0,0,.28);font:700 17px/1 system-ui,sans-serif;opacity:0;transform:translateY(-2px);transition:opacity .12s ease,transform .12s ease}
.plasma-image-viewer-target:hover .plasma-image-viewer-open,.plasma-image-viewer-target:focus-within .plasma-image-viewer-open,.plasma-image-viewer-open:focus-visible{opacity:1;transform:translateY(0)}
@media (hover:none){.plasma-image-viewer-open{opacity:1;transform:translateY(0)}}
</style>
<script>
(function(){
  var type=${JSON.stringify(MESSAGE_TYPE)};
  function targetFor(img){return img.closest("a")||img.closest("picture")||img;}
  function details(img){return{type:type,src:img.currentSrc||img.src||img.getAttribute("src")||"",alt:img.getAttribute("alt")||"",title:img.getAttribute("title")||""};}
  function svgSelector(){return [".plasma-mermaid-diagram > svg",".mermaid svg","svg[id^='plasma-mermaid-']","svg[id^='mermaid-']","svg[aria-label='Mermaid 그래프']"].join(",");}
  function svgTitle(svg){return svg.getAttribute("aria-label")||(svg.querySelector("title")&&svg.querySelector("title").textContent)||"Mermaid 그래프";}
  function legend(svg){
    var figure=svg.closest(".plasma-mermaid-card"),legendNode=figure&&figure.querySelector(".plasma-mermaid-line-legend");
    if(!legendNode)return[];
    return Array.prototype.map.call(legendNode.querySelectorAll(".plasma-mermaid-line-legend-item"),function(item){
      var marker=item.querySelector(".plasma-mermaid-line-legend-marker");
      var label=((item.querySelector(".plasma-mermaid-line-legend-text")||item).textContent||"").replace(/\\s+/g," ").trim();
      var color=marker?(marker.style.backgroundColor||getComputedStyle(marker).backgroundColor):"";
      return{label:label,color:color};
    }).filter(function(item){return item.label;});
  }
  function svgDetails(svg){var title=svgTitle(svg);return{type:type,kind:"svg",svg:svg.outerHTML||"",alt:title,title:title,legend:legend(svg)};}
  function addButton(wrapper,label,handler){
      var button=document.createElement("button");
      button.type="button";
      button.className="plasma-image-viewer-open";
      button.title=label;
      button.setAttribute("aria-label",label);
      button.textContent="⌕";
      button.addEventListener("click",handler);
      wrapper.appendChild(button);
  }
  function enhance(){
    Array.prototype.forEach.call(document.querySelectorAll("img"),function(img){
      if(img.dataset.plasmaImageViewerBound==="true")return;
      var src=img.currentSrc||img.src||img.getAttribute("src")||"";
      if(!src)return;
      var target=targetFor(img);
      if(!target||target.dataset.plasmaImageViewerTarget==="true")return;
      var wrapper=document.createElement("span");
      wrapper.className="plasma-image-viewer-target";
      target.parentNode.insertBefore(wrapper,target);
      wrapper.appendChild(target);
      target.dataset.plasmaImageViewerTarget="true";
      img.dataset.plasmaImageViewerBound="true";
      addButton(wrapper,"이미지 크게 보기",function(event){event.preventDefault();event.stopPropagation();parent.postMessage(details(img),"*");});
    });
    Array.prototype.forEach.call(document.querySelectorAll(svgSelector()),function(svg){
      if(svg.dataset.plasmaImageViewerBound==="true"||!svg.parentNode||svg.dataset.plasmaImageViewerTarget==="true")return;
      var wrapper=document.createElement("span");
      wrapper.className="plasma-image-viewer-target plasma-image-viewer-target--svg";
      svg.parentNode.insertBefore(wrapper,svg);
      wrapper.appendChild(svg);
      svg.dataset.plasmaImageViewerTarget="true";
      svg.dataset.plasmaImageViewerBound="true";
      addButton(wrapper,"Mermaid 그래프 크게 보기",function(event){event.preventDefault();event.stopPropagation();parent.postMessage(svgDetails(svg),"*");});
    });
  }
  if(document.readyState==="loading")document.addEventListener("DOMContentLoaded",enhance);else enhance();
}());
</script>`;
  }

  function preparePlasmaHTMLPreview(content) {
    const html = String(content || "");
    const injection = frameScript();
    if (/<\/body\s*>/i.test(html)) {
      return html.replace(/<\/body\s*>/i, `${injection}</body>`);
    }
    return `${html}${injection}`;
  }

  function onFrameMessage(event) {
    const data = event.data || {};
    if (!data || data.type !== MESSAGE_TYPE) return;
    const frames = Array.from(document.querySelectorAll("iframe.plasma-html-preview-frame"));
    const sourceFrame = frames.find((frame) => frame.contentWindow === event.source);
    if (!sourceFrame) return;
    openImageViewer(data);
  }

  window.addEventListener("message", onFrameMessage);
  window.enhancePlasmaImageViewing = enhancePlasmaImageViewing;
  window.preparePlasmaHTMLPreview = preparePlasmaHTMLPreview;
  window.openPlasmaImageViewer = openImageViewer;
}());

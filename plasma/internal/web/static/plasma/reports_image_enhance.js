(function reportsImageEnhance() {
  "use strict";
  const reports = window.Plasma.reports;
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
      reports.openImageViewer(imageDetailsFromElement(img));
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
      reports.openImageViewer(svgDetailsFromElement(svg));
    });
    wrapper.appendChild(button);
  }

  function enhancePlasmaImageViewing(root) {
    if (!root) return;
    root.querySelectorAll("img").forEach(enhanceImage);
    root.querySelectorAll(mermaidSVGSelector()).forEach(enhanceMermaidSVG);
  }


  reports.enhanceImages = enhancePlasmaImageViewing;
}());

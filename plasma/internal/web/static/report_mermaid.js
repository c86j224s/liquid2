(function (global) {
  "use strict";

  const mermaidScriptSrc = "/static/vendor/mermaid.min.js";
  const maxMermaidSourceLength = 50000;
  let mermaidLoadPromise = null;
  let mermaidRenderSequence = 0;

  function loadMermaidRuntime() {
    if (global.mermaid) return Promise.resolve(global.mermaid);
    if (mermaidLoadPromise) return mermaidLoadPromise;
    mermaidLoadPromise = new Promise((resolve, reject) => {
      const script = global.document.createElement("script");
      script.src = mermaidScriptSrc;
      script.async = true;
      script.onload = () => {
        if (global.mermaid) resolve(global.mermaid);
        else reject(new Error("Mermaid runtime did not expose window.mermaid"));
      };
      script.onerror = () => reject(new Error("Mermaid runtime failed to load"));
      global.document.head.appendChild(script);
    });
    return mermaidLoadPromise;
  }

  function configureMermaid(mermaid) {
    if (!mermaid || mermaid.__plasmaConfigured) return;
    mermaid.initialize({
      startOnLoad: false,
      securityLevel: "strict",
      theme: "default"
    });
    mermaid.__plasmaConfigured = true;
  }

  function mermaidSanitizeOptions() {
    return {
      USE_PROFILES: { html: true, svg: true, svgFilters: true },
      ADD_ATTR: ["aria-hidden", "aria-label", "aria-labelledby", "dominant-baseline", "dy", "role", "target", "text-anchor", "x", "y", "xmlns"]
    };
  }

  function sanitizeMermaidSVG(svg) {
    if (!global.DOMPurify) return "";
    return global.DOMPurify.sanitize(normalizeMermaidSVGLabels(String(svg || "")), mermaidSanitizeOptions());
  }

  function normalizeMermaidSVGLabels(svg) {
    if (!svg || !global.document) return svg;
    const template = global.document.createElement("template");
    template.innerHTML = svg;
    template.content.querySelectorAll("foreignObject").forEach((foreignObject) => {
      const lines = mermaidForeignObjectTextLines(foreignObject);
      if (!lines.length) {
        foreignObject.remove();
        return;
      }
      const width = Math.max(1, Number.parseFloat(foreignObject.getAttribute("width") || "0"));
      const height = Math.max(1, Number.parseFloat(foreignObject.getAttribute("height") || "0"));
      const text = global.document.createElementNS("http://www.w3.org/2000/svg", "text");
      text.setAttribute("class", "plasma-mermaid-label");
      text.setAttribute("x", String(width / 2));
      text.setAttribute("y", String(height / 2 - Math.max(0, lines.length - 1) * 8));
      text.setAttribute("text-anchor", "middle");
      text.setAttribute("dominant-baseline", "middle");
      lines.forEach((line, index) => {
        const tspan = global.document.createElementNS("http://www.w3.org/2000/svg", "tspan");
        tspan.setAttribute("x", String(width / 2));
        if (index > 0) tspan.setAttribute("dy", "1.15em");
        tspan.textContent = line;
        text.appendChild(tspan);
      });
      foreignObject.replaceWith(text);
    });
    return template.innerHTML;
  }

  function mermaidForeignObjectTextLines(node) {
    const lines = [];
    const current = [];
    const nodeTypes = global.Node || { TEXT_NODE: 3, ELEMENT_NODE: 1 };
    const flush = () => {
      const line = current.join("").replace(/\s+/g, " ").trim();
      if (line) lines.push(line);
      current.length = 0;
    };
    const visit = (item) => {
      if (item.nodeType === nodeTypes.TEXT_NODE) {
        current.push(item.nodeValue || "");
        return;
      }
      if (item.nodeType !== nodeTypes.ELEMENT_NODE) return;
      const tag = String(item.tagName || "").toUpperCase();
      if (tag === "BR") {
        flush();
        return;
      }
      Array.from(item.childNodes || []).forEach(visit);
      if (tag === "P" || tag === "DIV") flush();
    };
    Array.from(node.childNodes || []).forEach(visit);
    flush();
    return lines;
  }

  function buildMermaidFigure(source) {
    const figure = global.document.createElement("figure");
    figure.className = "plasma-mermaid-card";

    const output = global.document.createElement("div");
    output.className = "plasma-mermaid-diagram";
    output.setAttribute("aria-live", "polite");
    output.textContent = "Mermaid 그래프를 렌더링하는 중입니다...";
    figure.appendChild(output);

    const details = global.document.createElement("details");
    details.className = "plasma-mermaid-source";
    const summary = global.document.createElement("summary");
    summary.textContent = "Mermaid 코드";
    const pre = global.document.createElement("pre");
    const code = global.document.createElement("code");
    code.className = "plasma-mermaid-raw";
    code.textContent = source;
    pre.appendChild(code);
    details.append(summary, pre);
    figure.appendChild(details);

    return { figure, output, details };
  }

  function failMermaidFigure(figure, output, details, message) {
    figure.classList.add("plasma-mermaid-card--failed");
    output.textContent = message;
    details.open = true;
  }

  async function renderMermaidBlock(code, mermaid) {
    if (!code || !code.isConnected || code.dataset.plasmaMermaidState === "rendered") return;
    const pre = code.parentElement;
    if (!pre) return;
    const source = String(code.textContent || "").trim();
    if (!source) {
      code.dataset.plasmaMermaidState = "rendered";
      pre.classList.remove("plasma-mermaid-source-loading");
      return;
    }
    const { figure, output, details } = buildMermaidFigure(source);
    pre.replaceWith(figure);

    if (source.length > maxMermaidSourceLength) {
      failMermaidFigure(figure, output, details, "Mermaid 그래프가 너무 커서 안전하게 렌더링하지 않았습니다.");
      return;
    }
    try {
      const renderID = `plasma-mermaid-${Date.now()}-${++mermaidRenderSequence}`;
      const result = await mermaid.render(renderID, source);
      const sanitized = sanitizeMermaidSVG(result && result.svg);
      if (!sanitized) throw new Error("Mermaid SVG sanitizer returned empty output");
      output.innerHTML = sanitized;
      const svg = output.querySelector("svg");
      if (svg) {
        svg.setAttribute("role", "img");
        if (!svg.getAttribute("aria-label") && !svg.getAttribute("aria-labelledby")) {
          svg.setAttribute("aria-label", "Mermaid 그래프");
        }
      }
      if (typeof result.bindFunctions === "function") result.bindFunctions(output);
      global.applyPlasmaMermaidLineLegend?.(figure, source);
      global.enhancePlasmaImageViewing?.(output);
      code.dataset.plasmaMermaidState = "rendered";
    } catch (_error) {
      failMermaidFigure(figure, output, details, "Mermaid 그래프를 렌더링하지 못했습니다.");
    }
  }

  function renderPlasmaMermaid(root) {
    if (!root || !root.querySelectorAll || !global.document) return;
    const blocks = Array.from(root.querySelectorAll("pre > code.language-mermaid, pre > code.lang-mermaid"))
      .filter((code) => !code.closest(".plasma-mermaid-card") && code.dataset.plasmaMermaidState !== "loading" && code.dataset.plasmaMermaidState !== "rendered");
    if (!blocks.length) return;
    blocks.forEach((code) => {
      code.dataset.plasmaMermaidState = "loading";
      code.parentElement?.classList.add("plasma-mermaid-source-loading");
    });
    loadMermaidRuntime()
      .then((mermaid) => {
        configureMermaid(mermaid);
        blocks.forEach((code) => renderMermaidBlock(code, mermaid));
      })
      .catch(() => {
        blocks.forEach((code) => {
          code.dataset.plasmaMermaidState = "unavailable";
          code.parentElement?.classList.remove("plasma-mermaid-source-loading");
          code.parentElement?.classList.add("plasma-mermaid-source-unavailable");
        });
      });
  }

  global.renderPlasmaMermaid = renderPlasmaMermaid;
  global.plasmaMermaidSanitizeOptions = mermaidSanitizeOptions;
})(window);

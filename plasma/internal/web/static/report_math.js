(function (global) {
  "use strict";

  const mathOptions = Object.freeze({ throwOnError: true, trust: false, output: "htmlAndMathml", maxSize: 20, maxExpand: 1000 });

  function escaped(value) {
    return String(value).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/\"/g, "&quot;");
  }

  function placeholder(tex, display) {
    const raw = (display ? "\\[" : "\\(") + tex + (display ? "\\]" : "\\)");
    return `<span class="plasma-math plasma-math-${display ? "display" : "inline"}" data-tex="${escaped(tex)}" data-display="${display}">${escaped(raw)}</span>`;
  }

  function installMarkdownItMath(md) {
    if (!md || md.__plasmaMathInstalled || typeof global.texmath !== "function") return md;
    md.__plasmaMathInstalled = true;
    global.texmath(md, { delimiters: "brackets", engine: { renderToString(tex, options) { return placeholder(tex, options.displayMode === true); } } });
    return md;
  }

  function plasmaMathSanitizeOptions() {
    return { USE_PROFILES: { html: true, mathMl: true, svg: true }, ADD_ATTR: ["aria-hidden", "encoding", "xmlns"] };
  }

  function renderPlasmaMath(root, config) {
    if (!root || !root.querySelectorAll) return;
    root.querySelectorAll(".plasma-math[data-tex]").forEach((node) => {
      const raw = node.textContent;
      const trustedPlaceholders = config && config.trustedPlaceholders === true;
      if (!global.katex || (!global.DOMPurify && !trustedPlaceholders)) { node.textContent = raw; return; }
      try {
        const options = Object.assign({}, mathOptions, { displayMode: node.dataset.display === "true" });
        const rendered = global.katex.renderToString(node.dataset.tex || "", options);
        node.innerHTML = global.DOMPurify ? global.DOMPurify.sanitize(rendered, plasmaMathSanitizeOptions()) : rendered;
        node.removeAttribute("data-tex");
      } catch (_error) {
        node.textContent = raw;
        node.classList.add("plasma-math-error");
        node.removeAttribute("data-tex");
      }
    });
  }

  function renderPlasmaMarkdown(root, markdown) {
    if (!root || !global.markdownit || !global.DOMPurify) return;
    const md = installMarkdownItMath(global.markdownit({ html: false, linkify: true, breaks: true }));
    root.innerHTML = global.DOMPurify.sanitize(md.render(String(markdown || "")), { USE_PROFILES: { html: true }, ADD_ATTR: ["target", "rel", "data-tex", "data-display"] });
    renderPlasmaMath(root);
    bindReportHeadingInteractions(root);
    if (typeof global.renderPlasmaMermaid === "function") global.renderPlasmaMermaid(root);
  }

  function bindReportHeadingInteractions(root) {
    if (!root || !root.querySelectorAll) return;
    root.querySelectorAll("h2,h3").forEach((heading) => {
      if (heading.dataset.plasmaHeadingBound === "true") return;
      heading.dataset.plasmaHeadingBound = "true";
      heading.tabIndex = 0;
      heading.addEventListener("click", () => heading.classList.toggle("marked"));
    });
  }

  function excludedTextNode(node, root) {
    for (let parent = node.parentElement; parent && parent !== root; parent = parent.parentElement) {
      if (/^(A|CODE|PRE|SCRIPT|STYLE|SVG)$/.test(String(parent.tagName).toUpperCase())) return true;
    }
    return false;
  }

  function renderDesignedTextMath(root) {
    if (!root || !global.texmath || !global.document) return;
    const inlineRule = global.texmath.rules.brackets.inline[0];
    const displayRule = global.texmath.rules.brackets.block[1];
    const walker = global.document.createTreeWalker(root, global.NodeFilter.SHOW_TEXT);
    const nodes = [];
    while (walker.nextNode()) if (!excludedTextNode(walker.currentNode, root)) nodes.push(walker.currentNode);
    for (const node of nodes) {
      const source = node.nodeValue;
      const matchers = [{ rule: displayRule, display: true }, { rule: inlineRule, display: false }];
      let cursor = 0;
      let fragment = null;
      while (cursor < source.length) {
        let best = null;
        for (const candidate of matchers) {
          const match = findRuleMatch(source, candidate.rule, cursor);
          if (match && (!best || match.index < best.match.index)) best = { match, display: candidate.display };
        }
        if (!best) break;
        if (!fragment) fragment = global.document.createDocumentFragment();
        if (best.match.index > cursor) fragment.append(source.slice(cursor, best.match.index));
        const holder = global.document.createElement("span");
        holder.className = `plasma-math plasma-math-${best.display ? "display" : "inline"}`;
        holder.dataset.tex = best.match[1];
        holder.dataset.display = String(best.display);
        holder.textContent = best.match[0];
        fragment.append(holder);
        cursor = best.match.index + best.match[0].length;
      }
      if (fragment) {
        if (cursor < source.length) fragment.append(source.slice(cursor));
        node.replaceWith(fragment);
      }
    }
    renderPlasmaMath(root, { trustedPlaceholders: true });
  }

  function findRuleMatch(source, rule, from) {
    const cursor = source.indexOf(rule.tag, from);
    if (cursor < 0) return null;
    rule.rex.lastIndex = cursor;
    return rule.rex.exec(source);
  }

  global.plasmaMathOptions = mathOptions;
  global.installMarkdownItMath = installMarkdownItMath;
  global.renderPlasmaMath = renderPlasmaMath;
  global.renderPlasmaMarkdown = renderPlasmaMarkdown;
  global.bindReportHeadingInteractions = bindReportHeadingInteractions;
  global.renderDesignedTextMath = renderDesignedTextMath;
  global.plasmaMathSanitizeOptions = plasmaMathSanitizeOptions;
})(typeof window === "undefined" ? globalThis : window);

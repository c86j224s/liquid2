(function reportsMarkdown(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const escapeHTML = root.Plasma.dom.escapeHTML;
const markdownRenderer = window.markdownit ? window.markdownit({
  html: false,
  linkify: true,
  breaks: true,
  typographer: false
}) : null;

reports.installMarkdownItMath?.(markdownRenderer);

if (markdownRenderer) {
  const defaultLinkOpen = markdownRenderer.renderer.rules.link_open ||
    ((tokens, idx, options, env, self) => self.renderToken(tokens, idx, options));
  markdownRenderer.renderer.rules.link_open = (tokens, idx, options, env, self) => {
    const token = tokens[idx];
    const targetIndex = token.attrIndex("target");
    if (targetIndex < 0) {
      token.attrPush(["target", "_blank"]);
    } else {
      token.attrs[targetIndex][1] = "_blank";
    }
    const relIndex = token.attrIndex("rel");
    if (relIndex < 0) {
      token.attrPush(["rel", "noopener noreferrer"]);
    } else {
      token.attrs[relIndex][1] = "noopener noreferrer";
    }
    return defaultLinkOpen(tokens, idx, options, env, self);
  };
}


  function renderMarkdown(value, options = {}) {
    const text = String(value ?? "");
    if (!markdownRenderer || !root.DOMPurify) return escapeHTML(text);
    const rendered = options.redpenBlocks && reports.redpenMarkdown
      ? reports.redpenMarkdown.render(markdownRenderer, text)
      : markdownRenderer.render(text);
    return root.DOMPurify.sanitize(rendered, {
      USE_PROFILES: { html: true },
      ADD_ATTR: ["target", "rel", "data-tex", "data-display", "data-redpen-start-line", "data-redpen-end-line", "data-redpen-block-kind"]
    });
  }
  reports.renderMarkdown = renderMarkdown;
})(window);

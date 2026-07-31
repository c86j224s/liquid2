(function (global) {
  "use strict";

  const forbiddenContainerTokens = new Set([
    "fence", "code_block", "html_block", "table_open",
    "bullet_list_open", "ordered_list_open", "blockquote_open"
  ]);

  function render(renderer, markdown) {
    const source = String(markdown ?? "");
    const env = {};
    const tokens = renderer.parse(source, env);
    tokens.forEach((token, index) => {
      if (!supportedBlock(tokens, index)) return;
      token.attrJoin("class", "report-redpen-block");
      token.attrSet("data-redpen-start-line", String(token.map[0]));
      token.attrSet("data-redpen-end-line", String(token.map[1]));
      token.attrSet("data-redpen-block-kind", token.type.replace(/_open$/, ""));
    });
    return renderer.renderer.render(tokens, renderer.options, env);
  }

  function supportedBlock(tokens, index) {
    const token = tokens[index];
    if (!Array.isArray(token.map) || token.map.length !== 2 || token.map[1] <= token.map[0]) return false;
    if ((token.type === "heading_open" || token.type === "paragraph_open") && token.level === 0) return true;
    if (token.type === "list_item_open") return simpleContainer(tokens, index, "list_item_close");
    if (token.type === "blockquote_open" && token.level === 0) return simpleContainer(tokens, index, "blockquote_close");
    return false;
  }

  function simpleContainer(tokens, start, closeType) {
    for (let index = start + 1; index < tokens.length; index += 1) {
      const token = tokens[index];
      if (token.type === closeType) return true;
      if (forbiddenContainerTokens.has(token.type)) return false;
    }
    return false;
  }

  function selectionBlock(root, selection) {
    if (!root || !selection || selection.isCollapsed || !selection.anchorNode || !selection.focusNode) return null;
    const anchor = elementForNode(selection.anchorNode)?.closest("[data-redpen-start-line]");
    const focus = elementForNode(selection.focusNode)?.closest("[data-redpen-start-line]");
    if (!anchor || anchor !== focus) return null;
    return mappedBlock(root, anchor);
  }

  function blockForTarget(root, target) {
    return mappedBlock(root, elementForNode(target)?.closest?.("[data-redpen-start-line]"));
  }

  function mappedBlock(root, element) {
    if (!root || !element || !root.contains(element)) return null;
    const startLine = Number(element.dataset.redpenStartLine);
    const endLine = Number(element.dataset.redpenEndLine);
    if (!Number.isInteger(startLine) || !Number.isInteger(endLine) || endLine <= startLine) return null;
    return { element, startLine, endLine };
  }

  function elementForNode(node) {
    if (!node) return null;
    return node.nodeType === 1 ? node : node.parentElement;
  }

  function rawBlock(markdown, startLine, endLine) {
    const range = lineRange(String(markdown ?? ""), startLine, endLine);
    return range.text.slice(range.start, range.end);
  }

  function replaceBlock(markdown, startLine, endLine, replacement) {
    const text = String(markdown ?? "");
    const range = lineRange(text, startLine, endLine);
    let next = String(replacement ?? "");
    const original = text.slice(range.start, range.end);
    if (range.end < text.length && /\r?\n$/.test(original) && !/\r?\n$/.test(next)) {
      next += original.endsWith("\r\n") ? "\r\n" : "\n";
    }
    return text.slice(0, range.start) + next + text.slice(range.end);
  }

  function lineRange(text, startLine, endLine) {
    const starts = [0];
    for (let index = 0; index < text.length; index += 1) {
      if (text[index] === "\n") starts.push(index + 1);
    }
    return {
      text,
      start: starts[startLine] ?? text.length,
      end: starts[endLine] ?? text.length
    };
  }

  function createInlineEditor(block, raw, handlers) {
    const tag = block.tagName === "LI" || block.tagName === "BLOCKQUOTE" ? block.tagName.toLowerCase() : "div";
    const editor = document.createElement(tag);
    editor.className = "report-redpen-inline-editor";
    const textarea = document.createElement("textarea");
    textarea.className = "report-redpen-textarea";
    textarea.value = String(raw ?? "").replace(/\r?\n$/, "");
    textarea.rows = Math.max(4, Math.min(16, textarea.value.split(/\r?\n/).length + 2));
    textarea.setAttribute("aria-label", "선택한 Markdown 블록 편집");
    const actions = document.createElement("div");
    actions.className = "report-redpen-inline-actions";
    actions.append(iconButton("✓", "블록 수정 반영", () => handlers.apply(textarea.value)));
    actions.append(iconButton("×", "블록 수정 취소", handlers.cancel));
    editor.append(textarea, actions);
    block.replaceWith(editor);
    editor.scrollIntoView({ block: "center", inline: "nearest" });
    textarea.focus();
    textarea.setSelectionRange(textarea.value.length, textarea.value.length);
    return { editor, textarea };
  }

  function iconButton(symbol, label, action) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "icon-button quiet";
    button.textContent = symbol;
    button.title = label;
    button.setAttribute("aria-label", label);
    button.addEventListener("click", action);
    return button;
  }

  global.Plasma.reports.redpenMarkdown = { render, selectionBlock, blockForTarget, rawBlock, replaceBlock, createInlineEditor };
})(window);

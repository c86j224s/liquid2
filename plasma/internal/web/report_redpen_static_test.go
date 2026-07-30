package web

import (
	"os/exec"
	"strings"
	"testing"
)

func TestReportRedpenStaticContracts(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	appScript := string(mustReadStatic(t, "static/app.js"))
	controller := string(mustReadStatic(t, "static/report_redpen.js"))
	markdown := string(mustReadStatic(t, "static/report_redpen_markdown.js"))
	styles := string(mustReadStatic(t, "static/report_redpen.css"))
	combined := index + appScript + controller + markdown + styles
	for _, expected := range []string{
		`id="reportRedpenStart"`, `id="reportRedpenSave"`, `id="reportRedpenCancel"`,
		`src="/static/report_redpen_markdown.js"`, `src="/static/report_redpen.js"`,
		`href="/static/report_redpen.css"`, "createReportRedpenController",
		"expected_current_artifact_id", "/redpen/download", "report.redpen.saved",
		"data-redpen-start-line", "report-redpen-inline-editor", "beforeunload",
		"저장하지 않은 빨간펜 수정", "report-redpen-active", "report-redpen-mode",
		"position: sticky", "var(--code-fg)",
		`setAttribute("aria-pressed", String(active))`, `#reportRedpenStart[aria-pressed="true"]`,
		"빨간펜 켜짐",
		`scrollIntoView({ block: "center", inline: "nearest" })`,
		"textarea.focus()", "selection.removeAllRanges()",
		`body.addEventListener("click", selectBlockOnClick)`, "blockForTarget",
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("missing report redpen static contract %q", expected)
		}
	}
	for _, forbidden := range []string{"bottom-sheet", "redpen-fullscreen-editor"} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("report redpen must remain in-place, found %q", forbidden)
		}
	}
}

func TestReportRedpenStartsInlineEditWithinClickGesture(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	fixture := `
const fs = require("fs");
(async () => {
let eventPhase = "";
class ClassList {
  add() {}
  remove() {}
  toggle() {}
}
class Target {
  constructor() {
    this.listeners = {};
    this.classList = new ClassList();
    this.attributes = {};
  }
  addEventListener(type, listener) {
    (this.listeners[type] ||= []).push(listener);
  }
  setAttribute(name, value) { this.attributes[name] = value; }
  async dispatch(type, init = {}) {
    let prevented = false;
    const event = {button: 0, preventDefault() { prevented = true; }, ...init};
    eventPhase = type;
    for (const listener of this.listeners[type] || []) await listener(event);
    eventPhase = "";
    return prevented;
  }
}
global.window = global;
global.addEventListener = () => {};
global.document = new Target();
const body = new Target();
body.contains = () => true;
const container = new Target();
const status = new Target();
const startButton = new Target();
const saveButton = new Target();
const cancelButton = new Target();
const blockElement = new Target();
blockElement.contains = () => true;
const block = {element: blockElement, startLine: 0, endLine: 1};
let createCount = 0;
let createPhases = [];
let selectionClearCount = 0;
let editorHandlers;
global.getSelection = () => ({
  isCollapsed: false,
  anchorNode: {},
  focusNode: {},
  removeAllRanges() { selectionClearCount += 1; }
});
global.ReportRedpenMarkdown = {
  selectionBlock() { return block; },
  blockForTarget(_root, target) { return target === blockElement ? block : null; },
  rawBlock() { return "원문\n"; },
  createInlineEditor(_element, _raw, handlers) {
    createCount += 1;
    createPhases.push(eventPhase);
    editorHandlers = handlers;
    return {textarea: {}};
  }
};
eval(fs.readFileSync("static/report_redpen.js", "utf8"));
const controller = createReportRedpenController({
  body, container, status, startButton, saveButton, cancelButton,
  render(value) { return value; },
  async load() { return {exists: false, workcopy: {}}; },
  async save() { throw new Error("save should not run"); },
  error(error) { throw error; },
  confirm() { return true; }
});
controller.open({sourceArtifactID: "artifact-1", content: "원문\n"});
await startButton.dispatch("click");
if (startButton.attributes["aria-pressed"] !== "true" || status.textContent !== "빨간펜 켜짐") throw new Error("redpen mode did not expose its active state");
const tapPrevented = await body.dispatch("click", {target: blockElement});
if (!tapPrevented) throw new Error("tapped editable block did not suppress its default action");
if (selectionClearCount !== 1) throw new Error("native selection was not released after tap capture");
await startButton.dispatch("pointerdown");
if (createCount !== 0) throw new Error("pointerdown created the inline editor before click");
await startButton.dispatch("click");
if (createCount !== 1 || createPhases[0] !== "click") throw new Error("inline editor did not start in click");
await startButton.dispatch("click");
if (createCount !== 1) throw new Error("follow-up click created a duplicate editor");
editorHandlers.cancel();
await global.document.dispatch("selectionchange");
if (selectionClearCount !== 2) throw new Error("second native selection was not released");
await startButton.dispatch("click");
if (createCount !== 2 || createPhases[1] !== "click") throw new Error("keyboard click fallback did not edit");
editorHandlers.cancel();
await cancelButton.dispatch("click");
if (startButton.attributes["aria-pressed"] !== "false") throw new Error("redpen mode did not clear its active state");
})().catch(error => {
  console.error(error);
  process.exit(1);
});
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("report redpen pointer gesture fixture: %v: %s", err, out)
	}
}

func TestReportRedpenMarkdownBlockMapping(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	fixture := `
const fs = require("fs");
global.window = global;
eval(fs.readFileSync("static/report_redpen_markdown.js", "utf8"));
const md = require("./static/vendor/markdown-it.min.js")({html:false,breaks:true});
const fence = String.fromCharCode(96).repeat(3);
const source = "# 제목\n\n첫 문단입니다.\n\n- 단순 항목\n\n" + fence + "js\nconst unsafe = true;\n" + fence + "\n\n| 열 | 값 |\n|---|---|\n| a | b |\n";
const html = ReportRedpenMarkdown.render(md, source);
const marked = (html.match(/data-redpen-start-line=/g) || []).length;
if (marked !== 3) throw new Error("expected heading, paragraph, and simple list item only: " + html);
if (/<pre[^>]*data-redpen/.test(html) || /<table[^>]*data-redpen/.test(html)) throw new Error("complex block became editable: " + html);
const block = {dataset:{redpenStartLine:"2",redpenEndLine:"3"}};
const parent = {closest(){return block;}};
const node = {nodeType:3,parentElement:parent};
const root = {contains(value){return value === block;}};
const selection = {isCollapsed:false,anchorNode:node,focusNode:node};
const selected = ReportRedpenMarkdown.selectionBlock(root, selection);
if (!selected || selected.startLine !== 2 || selected.endLine !== 3) throw new Error("same-block selection was not mapped");
const tapped = ReportRedpenMarkdown.blockForTarget(root, node);
if (!tapped || tapped.element !== block || tapped.startLine !== 2 || tapped.endLine !== 3) throw new Error("tapped block was not mapped");
if (ReportRedpenMarkdown.blockForTarget(root, {nodeType:1,closest(){return null;}})) throw new Error("unsupported tap target was mapped");
const other = {nodeType:3,parentElement:{closest(){return {dataset:{redpenStartLine:"4",redpenEndLine:"5"}};}}};
if (ReportRedpenMarkdown.selectionBlock(root, {isCollapsed:false,anchorNode:node,focusNode:other})) throw new Error("multi-block selection was accepted");
const crlf = "# 제목\r\n\r\n기존 문장\r\n\r\n다음 문장\r\n";
if (ReportRedpenMarkdown.rawBlock(crlf, 2, 3) !== "기존 문장\r\n") throw new Error("raw block range changed");
const replaced = ReportRedpenMarkdown.replaceBlock(crlf, 2, 3, "고친 문장");
if (replaced !== "# 제목\r\n\r\n고친 문장\r\n\r\n다음 문장\r\n") throw new Error("surrounding Markdown changed: " + JSON.stringify(replaced));
let focused = 0;
let scrolled = 0;
const interactionOrder = [];
function fakeElement(tag) {
  return {
    tagName: tag.toUpperCase(),
    append() {},
    replaceWith() {},
    setAttribute() {},
    addEventListener() {},
    scrollIntoView(options) {
      if (options.block !== "center" || options.inline !== "nearest") throw new Error("unexpected inline editor scroll options");
      scrolled += 1;
      interactionOrder.push("scroll");
    },
    focus() { focused += 1; interactionOrder.push("focus"); document.activeElement = this; },
    setSelectionRange() {}
  };
}
const textarea = fakeElement("textarea");
global.document = {
  activeElement: null,
  createElement(tag) { return tag === "textarea" ? textarea : fakeElement(tag); }
};
ReportRedpenMarkdown.createInlineEditor(fakeElement("p"), "본문", {apply() {}, cancel() {}});
if (scrolled !== 1 || interactionOrder.join(",") !== "scroll,focus") throw new Error("inline editor was not revealed before focus");
if (focused !== 1 || document.activeElement !== textarea) throw new Error("inline editor was not focused");
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("report redpen Markdown fixture: %v: %s", err, out)
	}
}

package web

import (
	"os/exec"
	"strings"
	"testing"
)

func TestReportTOCStaticWiring(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	appCSS := string(mustReadStatic(t, "static/app.css"))
	styles := string(mustReadStatic(t, "static/plasma/report_toc.css"))
	modal := string(mustReadStatic(t, "static/plasma/reports_modal.js"))
	exports := string(mustReadStatic(t, "static/plasma/reports_exports_core.js"))
	redpen := string(mustReadStatic(t, "static/plasma/reports_redpen_init.js"))

	for _, expected := range []string{
		`id="reportTOCToggle"`, `aria-controls="reportTOCPanel"`, `aria-expanded="false"`,
		`id="reportTOCPanel"`, `id="reportTOCList"`, `/static/plasma/reports_toc.js`,
	} {
		if !strings.Contains(index, expected) {
			t.Fatalf("report TOC markup is missing %q", expected)
		}
	}
	if !strings.Contains(appCSS, `@import url("/static/plasma/report_toc.css");`) {
		t.Fatal("app CSS must load the report TOC styles")
	}
	for _, expected := range []string{
		".report-toc-panel", ".report-toc-link", `[data-level="4"]`, "@media (max-width: 560px)",
	} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("report TOC styles are missing %q", expected)
		}
	}
	if !strings.Contains(styles, "flex: 0 0 auto") || strings.Contains(styles, "flex: 0 1 auto") {
		t.Fatal("opened report TOC must keep its content height instead of shrinking against the report body")
	}
	for _, expected := range []string{
		`options.tableOfContents === true`, `tocController?.prepare(tableOfContents)`, `tocController?.refresh()`,
	} {
		if !strings.Contains(modal, expected) {
			t.Fatalf("report modal TOC lifecycle is missing %q", expected)
		}
	}
	if strings.Count(exports, "tableOfContents:") != 3 || strings.Count(exports, "tableOfContents: true") != 2 || !strings.Contains(exports, `tableOfContents: target === "markdown"`) {
		t.Fatal("report and redpen previews must opt into the TOC while markdown exports do so conditionally")
	}
	if !strings.Contains(redpen, "tocController?.refresh()") || !strings.Contains(redpen, "tocController?.reset()") {
		t.Fatal("redpen rerenders and modal exit must keep the report TOC lifecycle in sync")
	}
}

func TestReportTOCBehavior(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for the report TOC fixture")
	}
	script := string(mustReadStatic(t, "static/plasma/reports_toc.js"))
	fixture := `
function element(tag = "div") {
  const listeners = {}, attrs = {};
  const node = {
    tagName: tag.toUpperCase(), children: [], dataset: {}, hidden: false, textContent: "", className: "",
    ownerDocument: null,
    addEventListener(type, callback) { listeners[type] = callback; },
    dispatch(type, event = {}) { listeners[type]?.({...event, target: event.target || node}); },
    append(child) { node.children.push(child); },
    replaceChildren(...children) { node.children = children; },
    contains(target) { return target === node || node.children.some((child) => child.contains?.(target)); },
    setAttribute(name, value) { attrs[name] = String(value); },
    getAttribute(name) { return attrs[name]; },
    hasAttribute(name) { return Object.hasOwn(attrs, name); },
    removeAttribute(name) { delete attrs[name]; },
    closest(selector) { return selector === "[data-report-toc-index]" && node.dataset.reportTocIndex !== undefined ? node : null; }
  };
  return node;
}
const document = { createElement(tag) { const node = element(tag); node.ownerDocument = document; return node; } };
function heading(tag, text, top) {
  const node = element(tag);
  node.textContent = text;
  node.getBoundingClientRect = () => ({top});
  node.focus = (options) => { node.focused = options; };
  return node;
}
let headings = [heading("h1", "개요", 90), heading("h2", "같은 제목", 210), heading("h2", "같은 제목", 330), heading("h4", "세부", 450)];
const reportBody = { querySelectorAll: () => headings };
const body = element();
body.scrollTop = 20;
body.querySelector = () => reportBody;
body.getBoundingClientRect = () => ({top: 10});
body.scrollTo = (options) => { body.lastScroll = options; };
const panel = element("nav");
panel.hidden = true;
const list = element("ol");
list.ownerDocument = document;
const toggleButton = element("button");
const window = {document, Plasma:{reports:{}}};
` + script + `
const controller = window.Plasma.reports.createReportTOCController({body, panel, list, toggleButton});
controller.prepare(true);
controller.refresh();
if (toggleButton.hidden || !panel.hidden || list.children.length !== 4) throw new Error("TOC did not start collapsed with rendered headings");
if (list.children.map((item) => item.dataset.level).join(",") !== "1,2,2,4") throw new Error("heading hierarchy was not preserved");
if (list.children[1].children[0].textContent !== list.children[2].children[0].textContent) throw new Error("duplicate headings were not preserved");
toggleButton.dispatch("click");
if (panel.hidden || toggleButton.getAttribute("aria-expanded") !== "true") throw new Error("TOC did not expand");
const secondDuplicate = list.children[2].children[0];
list.dispatch("click", {target: secondDuplicate});
if (body.lastScroll.top !== 328 || body.lastScroll.behavior !== "smooth") throw new Error("TOC navigated outside the expected scroll position: " + JSON.stringify(body.lastScroll));
if (!headings[2].focused?.preventScroll || headings[2].getAttribute("tabindex") !== "-1") throw new Error("TOC navigation did not move accessible focus");
if (!panel.hidden || toggleButton.getAttribute("aria-expanded") !== "false") throw new Error("TOC must collapse after navigation");
controller.reset();
if (!toggleButton.hidden || list.children.length || headings[2].hasAttribute("tabindex")) throw new Error("TOC reset leaked preview state");
headings = [];
controller.prepare(true);
controller.refresh();
if (!toggleButton.hidden || !panel.hidden) throw new Error("heading-free reports must not expose an empty TOC");
`
	if output, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("report TOC fixture failed: %v: %s", err, output)
	}
}

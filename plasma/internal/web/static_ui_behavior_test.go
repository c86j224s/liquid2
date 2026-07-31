package web

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestPlasmaUIRuntimeBehaviorContracts(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for Plasma.ui runtime fixture")
	}
	fixture := `
const fs = require("fs");
const vm = require("vm");

function assert(ok, message) {
  if (!ok) throw new Error(message);
}

function classList(initial = []) {
  const values = new Set(initial);
  return {
    add(name) { values.add(name); },
    remove(name) { values.delete(name); },
    contains(name) { return values.has(name); },
    toggle(name, force) {
      const enabled = force === undefined ? !values.has(name) : Boolean(force);
      if (enabled) values.add(name); else values.delete(name);
      return enabled;
    }
  };
}

function node(id, options = {}) {
  return {
    id,
    textContent: options.textContent || "",
    innerHTML: options.innerHTML || "",
    value: options.value || "",
    disabled: false,
    scrollTop: options.scrollTop || 0,
    scrollHeight: options.scrollHeight || 0,
    clientHeight: options.clientHeight || 0,
    dataset: options.dataset || {},
    style: {},
    children: options.children || [],
    classList: classList(options.classes || []),
    querySelector(selector) {
      if (selector === ".modal-card") return this.children.find((child) => child.matches?.(selector)) || null;
      return null;
    },
    querySelectorAll(selector) {
      if (selector === "button") return this.children.filter((child) => child.tagName === "BUTTON");
      return [];
    },
    setAttribute(name, value) { this[name] = value; },
    select() { this.selected = true; },
    matches(selector) { return selector === ".modal-card" && this.classList.contains("modal-card"); },
    closest(selector) {
      if (selector === "[data-detail-json]" && Object.prototype.hasOwnProperty.call(this.dataset, "detailJson")) return this;
      return null;
    }
  };
}

const nodes = {
  countChip: node("countChip"),
  section: node("section"),
  control: node("control"),
  actionButton: node("actionButton"),
  errorText: node("errorText"),
  errorToast: node("errorToast", {classes:["hidden"]}),
  healthBadge: node("healthBadge"),
  detailTitle: node("detailTitle"),
  detailBody: node("detailBody", {scrollHeight:300, clientHeight:100}),
  detailPositionRatio: node("detailPositionRatio", {classes:["hidden"]})
};
const modalCard = node("modalCard", {classes:["modal-card", "modal-card--wide"]});
nodes.detailModal = node("detailModal", {classes:["hidden"], children:[modalCard]});
const formButtons = [node("formButtonA"), node("formButtonB")];
formButtons[0].tagName = "BUTTON";
formButtons[1].tagName = "BUTTON";
nodes.form = node("form", {children: formButtons});
const appended = [];
const copied = [];
let execCopyResult = true;

const document = {
  getElementById(id) { return nodes[id] || null; },
  createElement(tag) { return node(tag); },
  body: {
    appendChild(el) { appended.push(el); },
    removeChild(el) { assert(appended.pop() === el, "clipboard fallback removed the wrong node"); }
  },
  execCommand(command) {
    assert(command === "copy", "unexpected execCommand");
    copied.push(appended[appended.length - 1]?.value || "");
    return execCopyResult;
  }
};
const navigator = {clipboard: {writeText: async (text) => { copied.push("secure:" + text); }}};
globalThis.document = document;
globalThis.navigator = navigator;
globalThis.requestAnimationFrame = (callback) => { callback(); };
globalThis.window = globalThis;
globalThis.self = globalThis;
globalThis.isSecureContext = true;

const html = fs.readFileSync("static/index.html", "utf8");
const scripts = [...html.matchAll(/<script src="\/static\/([^"]+)"><\/script>/g)].map((match) => "static/" + match[1]);
const wanted = [
  "static/plasma/namespace.js",
  "static/plasma/dom.js",
  "static/plasma/state.js",
  "static/plasma/ui.js",
  "static/plasma/ui_feedback.js",
  "static/plasma/ui_detail.js"
];
const ordered = scripts.filter((script) => wanted.includes(script));
assert(JSON.stringify(ordered) === JSON.stringify(wanted), "Plasma.ui scripts are not in expected index order: " + JSON.stringify(ordered));
for (const script of ordered) {
  vm.runInThisContext(fs.readFileSync(script, "utf8"), {filename: script});
}

const ui = window.Plasma.ui;
assert(window === globalThis, "VM window and globalThis must be the same classic browser global");
for (const name of [
  "setSectionEmpty", "updateCountChip", "empty", "setElementDisabled", "setFormButtonsDisabled", "setButtonText",
  "showError", "hideError", "copyError", "copyText", "configureDetailHooks", "showDetail", "openDetailModal",
  "copyDetail", "hideDetail", "enableDetailScrollRatio", "disableDetailScrollRatio", "updateDetailScrollRatio",
  "detailScrollPosition", "onDetailModalClick", "onDetailButtonClick"
]) {
  assert(typeof ui[name] === "function", "Plasma.ui missing " + name);
}
for (const name of [
  "setSectionEmpty", "updateCountChip", "empty", "showError", "hideError", "copyError", "copyText", "showDetail",
  "openDetailModal", "copyDetail", "hideDetail", "enableDetailScrollRatio", "disableDetailScrollRatio",
  "updateDetailScrollRatio", "detailScrollPosition", "onDetailModalClick"
]) {
  assert(window[name] === ui[name], "classic global alias mismatch for " + name);
}

ui.updateCountChip("countChip", 3);
assert(nodes.countChip.textContent === "3", "count chip positive text changed");
ui.updateCountChip("countChip", 0);
assert(nodes.countChip.textContent === "", "count chip zero text changed");
assert(ui.empty("<x>") === '<div class="item"><div class="item-meta">&lt;x&gt;</div></div>', "empty markup changed");
ui.setSectionEmpty(nodes.section, true);
assert(nodes.section.classList.contains("hidden") && nodes.section.classList.contains("collapsed-empty"), "section empty true changed");
ui.setSectionEmpty(nodes.section, false);
assert(!nodes.section.classList.contains("hidden") && !nodes.section.classList.contains("collapsed-empty"), "section empty false changed");
ui.setElementDisabled("control", "yes");
assert(nodes.control.disabled === true, "element disabled true changed");
ui.setElementDisabled("control", 0);
assert(nodes.control.disabled === false, "element disabled false changed");
ui.setFormButtonsDisabled("form", (button) => button.id === "formButtonB");
assert(formButtons[0].disabled === false && formButtons[1].disabled === true, "form button disabling changed");
ui.setButtonText("actionButton", "저장");
assert(nodes.actionButton.textContent === "저장", "button text changed");

showError({userMessage:"사용자", message:"메시지", stack:"스택", details:{secret:true}});
assert(window.Plasma.state.lastError === "사용자", "userMessage precedence changed");
assert(nodes.errorText.textContent === "사용자", "normal error text changed for userMessage");
assert(!nodes.errorToast.classList.contains("hidden"), "normal error toast should show");
hideError();
assert(nodes.errorToast.classList.contains("hidden"), "hideError did not hide toast");
showError({message:"메시지", stack:"스택", details:{code:"x"}});
assert(window.Plasma.state.lastError.includes("스택") && window.Plasma.state.lastError.includes('"code": "x"'), "stack/details precedence changed");
assert(nodes.errorText.textContent.includes("스택") && nodes.errorText.textContent.includes('"code": "x"'), "stack/details error text changed");
hideError();
showError({isNetworkError:true, message:"네트워크"});
assert(nodes.healthBadge.textContent === "연결 끊김", "network health badge changed");
assert(nodes.errorToast.classList.contains("hidden"), "network error should not show toast");

let completed = false;
const watchdog = setTimeout(() => {
  console.error(new Error("Plasma.ui runtime fixture did not complete"));
  process.exit(1);
}, 5000);

(async () => {
  await copyText("secure-ok");
  assert(copied.at(-1) === "secure:secure-ok", "secure clipboard success changed");
  navigator.clipboard.writeText = async () => { throw new Error("reject"); };
  await copyText("secure-fallback");
  assert(copied.at(-1) === "secure-fallback", "secure clipboard fallback changed");
  window.isSecureContext = false;
  await copyText("plain-http");
  assert(copied.at(-1) === "plain-http", "plain HTTP execCommand fallback changed");
  execCopyResult = false;
  let failed = false;
  try { await copyText("fail"); } catch (err) { failed = err.message === "clipboard API is not available"; }
  assert(failed, "clipboard failure error changed");
  execCopyResult = true;

  openDetailModal();
  assert(modalCard.classList.contains("modal-card--wide"), "omitted width should preserve wide");
  openDetailModal(false);
  assert(!modalCard.classList.contains("modal-card--wide"), "explicit false width changed");
  openDetailModal(true);
  assert(modalCard.classList.contains("modal-card--wide"), "explicit true width changed");
  showDetail("제목", {a:1});
  assert(nodes.detailTitle.textContent === "제목", "detail title changed");
  assert(window.Plasma.state.detailText === JSON.stringify({a:1}, null, 2), "detail text changed");
  assert(nodes.detailBody.innerHTML === '<pre>{\n  &quot;a&quot;: 1\n}</pre>', "detail pre body changed");
  onDetailModalClick({target:{id:"other"}});
  assert(!nodes.detailModal.classList.contains("hidden"), "non-backdrop click closed modal");
  onDetailModalClick({target:nodes.detailModal});
  assert(nodes.detailModal.classList.contains("hidden"), "backdrop close changed");

  openDetailModal();
  nodes.detailBody.scrollTop = 50;
  enableDetailScrollRatio();
  nodes.detailBody.scrollTop = 50;
  updateDetailScrollRatio();
  assert(nodes.detailPositionRatio.textContent === "위치 25%", "scroll ratio changed");
  assert(!nodes.detailPositionRatio.classList.contains("hidden"), "scroll ratio visibility changed");
  disableDetailScrollRatio();
  assert(nodes.detailPositionRatio.classList.contains("hidden") && nodes.detailPositionRatio.textContent === "", "disable scroll ratio changed");

  let leaveAllowed = false;
  let beforeLeaveCalls = 0;
  let copyContentCalls = 0;
		const reports = {};
		reports.redpenController = {
	    beforeLeave() { beforeLeaveCalls++; return leaveAllowed; },
	    copyContent() { copyContentCalls++; return "edited-copy"; }
	  };
	  ui.configureDetailHooks({
	    beforeLeave: () => reports.redpenController?.beforeLeave() ?? true,
	    copyContent: () => reports.redpenController?.copyContent()
	  });
	  openDetailModal(true);
  assert(hideDetail() === false && !nodes.detailModal.classList.contains("hidden"), "app.js beforeLeave wiring changed");
  assert(beforeLeaveCalls === 1, "app.js beforeLeave hook was not called");
  leaveAllowed = true;
  await copyDetail();
  assert(copied.at(-1) === "edited-copy", "app.js edited copyContent wiring changed");
  assert(copyContentCalls === 1, "app.js copyContent hook was not called");
  assert(hideDetail() === true && nodes.detailModal.classList.contains("hidden"), "allowed hide detail changed");

  const detailButton = node("detailButton", {dataset:{detailTitle:"JSON", detailJson:'{"ok":true}'}});
  assert(ui.onDetailButtonClick({target:detailButton}) === true, "generic detail json was not handled");
  assert(window.Plasma.state.detailText === JSON.stringify({ok:true}, null, 2), "generic detail json content changed");
})().then(() => {
  completed = true;
  clearTimeout(watchdog);
}).catch((err) => {
  clearTimeout(watchdog);
  console.error(err);
  process.exit(1);
});
process.on("beforeExit", () => {
  if (!completed) process.exitCode = 1;
});
`
	output, err := exec.Command("node", "-e", fixture).CombinedOutput()
	if err != nil {
		t.Fatalf("Plasma.ui runtime behavior fixture failed: %v\n%s", err, output)
	}
}

func TestPlasmaUIPhysicalBoundaryContracts(t *testing.T) {
	controls := string(mustReadStaticUIBehavior(t, "static/plasma/ui.js"))
	feedback := string(mustReadStaticUIBehavior(t, "static/plasma/ui_feedback.js"))
	detail := string(mustReadStaticUIBehavior(t, "static/plasma/ui_detail.js"))
	app := string(mustReadStaticUIBehavior(t, "static/plasma/bootstrap_modules.js"))

	if strings.Count(controls+feedback+detail, "Plasma.ui =") != 1 {
		t.Fatal("Plasma.ui must have exactly one public namespace initialization")
	}
	for _, script := range []string{feedback, detail} {
		if !strings.Contains(script, "Object.assign(Plasma.ui") {
			t.Fatal("later Plasma.ui files must extend the existing public API")
		}
		if strings.Contains(script, "Plasma.ui =") {
			t.Fatal("later Plasma.ui files must not replace the public API object")
		}
	}
	for _, forbidden := range []string{
		"function showError(", "function hideError(", "async function copyError(", "async function copyText(",
		"function showDetail(", "function openDetailModal(", "async function copyDetail(", "function hideDetail(",
		"function enableDetailScrollRatio(", "function disableDetailScrollRatio(", "function updateDetailScrollRatio(",
		"function detailScrollPosition(", "function onDetailModalClick(",
	} {
		if strings.Contains(controls, forbidden) {
			t.Fatalf("controls file must not own feedback/detail behavior %q", forbidden)
		}
	}
	for _, forbidden := range []string{
		"function setSectionEmpty(", "function updateCountChip(", "function empty(",
		"function setElementDisabled(", "function setFormButtonsDisabled(", "function setButtonText(",
	} {
		if strings.Contains(feedback, forbidden) || strings.Contains(detail, forbidden) {
			t.Fatalf("feedback/detail files must not own controls behavior %q", forbidden)
		}
	}
	for _, forbidden := range []string{"function setSectionEmpty(", "function showError(", "function showDetail("} {
		if strings.Contains(app, forbidden) {
			t.Fatalf("app.js must not retain moved Plasma.ui implementation %q", forbidden)
		}
	}
}

func mustReadStaticUIBehavior(t *testing.T, name string) []byte {
	t.Helper()
	content, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

package web

import (
	"fmt"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
)

func TestSafeStoredArtifactFilename(t *testing.T) {
	cases := []struct{ name, raw, artifactID, want string }{
		{"path separators", "a/b\\c.txt", "art_1", "a_b_c.txt"},
		{"quote", "a\"b.txt", "art_1", "a_b.txt"},
		{"controls", "a\r\nb\x00c.txt", "art_1", "a__b_c.txt"},
		{"unicode format controls", "safe‮fdp⁦.txt", "art_1", "safe_fdp_.txt"},
		{"whitespace", "a b.txt", "art_1", "a_b.txt"},
		{"empty", "", "art_1", "art_1"},
		{"dot", "..", "art_1", "art_1"},
		{"unsafe artifact fallback", "\x00", "art_1/../../x", "art_1_.._.._x"},
		{"empty artifact fallback", "\x00", "\r\n", "download"},
		{"long unicode", strings.Repeat("가", 200), "art_1", strings.Repeat("가", 120)},
		{"long extension", strings.Repeat("a", 200) + ".html", "art_1", strings.Repeat("a", 114) + "_.html"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := safeStoredArtifactFilename(tc.raw, tc.artifactID); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestWriteStoredArtifactDownloadExactBytesAndHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	content := []byte("<html>stored</html>")
	writeStoredArtifactDownload(rec, artifactcontract.Raw{ArtifactID: "art_1", MediaType: "text/html; charset=utf-8", Filename: "dir\\page\".html", Content: content})
	if rec.Code != 200 || string(rec.Body.Bytes()) != string(content) {
		t.Fatalf("unexpected response %d %q", rec.Code, rec.Body.Bytes())
	}
	if got := rec.Header().Get("Content-Length"); got != fmt.Sprint(len(content)) {
		t.Fatalf("content length %q", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content type %q", got)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.HasPrefix(got, "attachment;") {
		t.Fatalf("disposition %q", got)
	}
	if got := rec.Header().Get("Content-Disposition"); strings.ContainsAny(got, "\r\n") {
		t.Fatalf("header injection %q", got)
	}
}

func TestWriteStoredArtifactDownloadFallsBackForInvalidMediaTypes(t *testing.T) {
	for _, mediaType := range []string{`text/html; charset="unterminated`, "text/html\x00evil"} {
		t.Run(fmt.Sprintf("%q", mediaType), func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeStoredArtifactDownload(rec, artifactcontract.Raw{
				ArtifactID: "art_1",
				MediaType:  mediaType,
				Filename:   "stored.bin",
				Content:    []byte("stored"),
			})
			if got := rec.Header().Get("Content-Type"); got != "application/octet-stream" {
				t.Fatalf("content type = %q", got)
			}
		})
	}
}

func TestSourceDownloadRenderStateMatrixAndEventIsolation(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for source download runtime fixture")
	}
	fixture := `
const fs = require("fs");
const vm = require("vm");
function assert(ok, message) { if (!ok) throw new Error(message); }
function classList() { return {add(){}, remove(){}, toggle(){}, contains(){return false;}}; }
function node(id) { return {id, innerHTML:"", textContent:"", classList:classList(), setAttribute(){}, querySelector(){return null;}}; }
const nodes = {sourceCandidateList:node("sourceCandidateList"), rejectedSourceCandidateDetails:node("rejectedSourceCandidateDetails"), rejectedSourceCandidateList:node("rejectedSourceCandidateList"), sourceList:node("sourceList"), confluenceResults:node("confluenceResults")};
const document = {querySelector(){return node("section");}, getElementById(id){return nodes[id] || null;}};
globalThis.window = globalThis; globalThis.self = globalThis; globalThis.document = document;
function load(name) { vm.runInThisContext(fs.readFileSync(name, "utf8"), {filename:name}); }
load("static/plasma/namespace.js");
load("static/plasma/dom.js");
load("static/plasma/state.js");
load("static/plasma/sources.js");
load("static/plasma/sources_locators.js");
load("static/plasma/sources_document_locators.js");
load("static/plasma/sources_confluence_locators.js");
Plasma.ui = {empty(){return ""}, updateCountChip(){}, setSectionEmpty(){}};
load("static/plasma/sources_rendering.js");
load("static/plasma/sources_candidate_events.js");
load("static/plasma/sources_candidates.js");
Plasma.sources.confluenceCandidatePageID = (candidate) => String(candidate.PageID || candidate.page_id || "");
load("static/plasma/sources_confluence_results_rendering.js");
Plasma.transport = {missionApi(){actionCalls++;}};
Plasma.mission = {
  captureMissionSelection(){return {missionId:Plasma.state.missionId, selectionGeneration:0};},
  ownsMissionSelection(){return true;},
  StaleMissionOperationError: class extends Error {},
  isStaleMissionOperation(){return false;}
};
Plasma.state.missionId = "mis / réservé?";
let detailCalls = 0;
let actionCalls = 0;
Plasma.sources.configure({
  renderTabs(){}, updateCountChip(){}, setSectionEmpty(){}, empty(){return ""},
  onDetailButtonClick(){detailCalls++; return false},
  sourceCandidateStagingLabel(){return ""}, pruneSelectedSourceCandidates(){}, updateSourceCandidateBulkBar(){},
  addURLSource(){actionCalls++}, rejectSourceCandidate(){actionCalls++}, restoreSourceCandidate(){actionCalls++},
  checkConfluenceSourceUpdate(){actionCalls++}, readSource(){actionCalls++}, removeSource(){actionCalls++}, restoreSource(){actionCalls++},
  showError(){actionCalls++}, missionApi(){actionCalls++}, reloadMission(){actionCalls++},
  localPathLocator(){return null}, mediaLocator(){return null}, pdfLocator(){return null}, documentLocator(){return null},
  confluenceSourceInfo(){return null}, sourceDetailPayload(source){return source}, confluenceUpdateState(){return null},
  confluenceUpdateText(){return ""}, confluenceScopeLabel(){return ""}, confluenceDisplayTitle(title){return title},
  mediaSourceLabel(){return ""}, documentSourceText(){return ""}, pdfSourceText(){return ""}, mediaSourceText(){return ""}, confluenceSourceText(){return ""}
});
load("static/plasma/sources_candidate_bulk.js");
load("static/plasma/sources_list_actions.js");
function candidateEvent(state, eventID, artifactID) { return {EventID:"evt_proposed", Sequence:1, EventType:"source.candidate.proposed", Payload:{candidates:[{url:"https://example.com/doc", title:"Doc", reason:"reason"}]},}; }
function staged(state, eventID, artifactID) {
  const terminal = state === "staged" ? "source.candidate.staged" : state === "fetching" ? "source.candidate.staging_started" : "source.candidate.staging_failed";
  return [candidateEvent(), {EventID:eventID, Sequence:2, EventType:terminal, Payload:{url:"https://example.com/doc", artifact_id:artifactID, proposal_event_id:"evt_proposed"}}];
}
for (const [state, events, existingSources] of [
  ["staged", staged("staged", "evt / staged?", "art / staged"), []],
  ["fetching", staged("fetching", "evt / fetching?", "art / fetching"), []],
  ["failed", staged("failed", "evt / failed?", "art / failed"), []],
  ["rejected", [...staged("staged", "evt / rejected?", "art / rejected"), {EventID:"evt_reject", Sequence:3, EventType:"source.candidate.rejected", Payload:{url:"https://example.com/doc"}}], []],
  ["accepted", staged("staged", "evt / accepted?", "art / accepted"), [{Connector:{ExternalURI:"https://example.com/doc"}}]],
  ["consumed", [...staged("staged", "evt / consumed?", "art / consumed"), {EventID:"evt_snapshot", Sequence:3, EventType:"source.snapshotted", Payload:{source_candidate_proposal_event_id:"evt_proposed", url:"https://example.com/doc"}}], []],
  ["same-url-old-proposal", [...staged("staged", "evt / same-url?", "art / same-url"), {EventID:"evt_snapshot", Sequence:3, EventType:"source.snapshotted", Payload:{source_candidate_proposal_event_id:"evt_old", url:"https://example.com/doc"}}], []],
  ["unrelated historical source", [...staged("staged", "evt / unrelated?", "art / unrelated"), {EventID:"evt_snapshot", Sequence:3, EventType:"source.snapshotted", Payload:{source_candidate_proposal_event_id:"evt_other", url:"https://example.com/other"}}], []],
  ["confluence proposal consumed", [...staged("staged", "evt / confluence?", "art / confluence"), {EventID:"evt_snapshot", Sequence:3, EventType:"source.snapshotted", Payload:{source_candidate_proposal_event_id:"evt_proposed", url:"https://example.atlassian.net/wiki/spaces/ENG/pages/123/title"}}], []],
  ["legacy Confluence API-token source", [{EventID:"evt_proposed", Sequence:1, EventType:"source.candidate.proposed", Payload:{candidates:[{url:"https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap", title:"Roadmap", reason:"reason"}]}}, {EventID:"evt_staged", Sequence:2, EventType:"source.candidate.staged", Payload:{url:"https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap", artifact_id:"art_legacy", proposal_event_id:"evt_proposed"}}, {EventID:"evt_snapshot", Sequence:3, EventType:"source.snapshotted", Payload:{connector:{connector_id:"confluence", connector_type:"confluence_cloud", external_source_id:"site_docs.atlassian.net:123", external_uri:"confluence://cloud/123"}}}], []],
  ["legacy different Confluence page", [{EventID:"evt_proposed", Sequence:1, EventType:"source.candidate.proposed", Payload:{candidates:[{url:"https://docs.atlassian.net/wiki/spaces/ENG/pages/124/Roadmap", title:"Roadmap", reason:"reason"}]}}, {EventID:"evt_staged", Sequence:2, EventType:"source.candidate.staged", Payload:{url:"https://docs.atlassian.net/wiki/spaces/ENG/pages/124/Roadmap", artifact_id:"art_legacy_other", proposal_event_id:"evt_proposed"}}, {EventID:"evt_snapshot", Sequence:3, EventType:"source.snapshotted", Payload:{connector:{connector_id:"confluence", connector_type:"confluence_cloud", external_source_id:"site_docs.atlassian.net:123", external_uri:"confluence://cloud/123"}}}], []]
]) {
  Plasma.sources.renderSourceCandidates(events, existingSources);
  const count = (nodes.sourceCandidateList.innerHTML.match(/source-saved-download/g) || []).length;
  assert(count === (["staged", "unrelated historical source", "confluence proposal consumed"].includes(state) ? 1 : 0), state + " candidate link count=" + count);
  if (state === "staged") {
    assert(nodes.sourceCandidateList.innerHTML.includes(encodeURIComponent("evt / staged?")), "staged event id not encoded");
    assert(nodes.sourceCandidateList.innerHTML.includes(encodeURIComponent("art / staged")), "staged artifact id not encoded");
    assert(nodes.sourceCandidateList.innerHTML.includes(encodeURIComponent(Plasma.state.missionId)), "mission id not encoded");
  }
}
const siblingEvents = [
  {EventID:"evt_pair", Sequence:1, EventType:"source.candidate.proposed", Payload:{candidates:[
    {url:"https://example.com/a", title:"A", reason:"reason A"},
    {url:"https://example.com/b", title:"B", reason:"reason B"}
  ]}},
  {EventID:"evt_stage_a", Sequence:2, EventType:"source.candidate.staged", Payload:{url:"https://example.com/a", artifact_id:"art_a", proposal_event_id:"evt_pair"}},
  {EventID:"evt_stage_b", Sequence:3, EventType:"source.candidate.staged", Payload:{url:"https://example.com/b", artifact_id:"art_b", proposal_event_id:"evt_pair"}}
];
Plasma.sources.renderSourceCandidates([...siblingEvents,
  {EventID:"evt_snapshot_a", Sequence:4, EventType:"source.snapshotted", Payload:{source_candidate_proposal_event_id:"evt_pair", url:"https://example.com/a"}}
], []);
assert(!nodes.sourceCandidateList.innerHTML.includes(encodeURIComponent("art_a")), "approved A remained visible");
assert(nodes.sourceCandidateList.innerHTML.includes(encodeURIComponent("art_b")), "same-proposal B lost its download link");
Plasma.sources.renderSourceCandidates([...siblingEvents,
  {EventID:"evt_snapshot_proposal_only", Sequence:4, EventType:"source.snapshotted", Payload:{source_candidate_proposal_event_id:"evt_pair"}}
], []);
assert(nodes.sourceCandidateList.innerHTML.includes(encodeURIComponent("art_a")), "proposal-only snapshot hid A");
assert(nodes.sourceCandidateList.innerHTML.includes(encodeURIComponent("art_b")), "proposal-only snapshot hid B");
const attemptEvents = [
  {EventID:"evt_attempt_proposed", Sequence:1, EventType:"source.candidate.proposed", Payload:{candidates:[{url:"https://example.com/attempt", title:"Attempt", reason:"reason"}]}},
  {EventID:"evt_start_a", Sequence:2, EventType:"source.candidate.staging_started", Payload:{url:"https://example.com/attempt", proposal_event_id:"evt_attempt_proposed", staging_event_id:"evt_start_a"}},
  {EventID:"evt_start_b", Sequence:3, EventType:"source.candidate.staging_started", Payload:{url:"https://example.com/attempt", proposal_event_id:"evt_attempt_proposed", staging_event_id:"evt_start_b"}},
  {EventID:"evt_stage_b", Sequence:4, EventType:"source.candidate.staged", Payload:{url:"https://example.com/attempt", proposal_event_id:"evt_attempt_proposed", staging_event_id:"evt_start_b", artifact_id:"art_attempt_b"}},
  {EventID:"evt_stage_a_late", Sequence:5, EventType:"source.candidate.staged", Payload:{url:"https://example.com/attempt", proposal_event_id:"evt_attempt_proposed", staging_event_id:"evt_start_a", artifact_id:"art_attempt_a"}}
];
Plasma.sources.renderSourceCandidates(attemptEvents, []);
assert(nodes.sourceCandidateList.innerHTML.includes(encodeURIComponent("evt_stage_b")), "latest attempt terminal event missing");
assert(nodes.sourceCandidateList.innerHTML.includes(encodeURIComponent("art_attempt_b")), "latest attempt artifact missing");
assert(!nodes.sourceCandidateList.innerHTML.includes(encodeURIComponent("art_attempt_a")), "late old attempt replaced current artifact");
const latestStart = {EventID:"evt_start_c", Sequence:6, EventType:"source.candidate.staging_started", Payload:{url:"https://example.com/attempt", proposal_event_id:"evt_attempt_proposed", staging_event_id:"evt_start_c"}};
Plasma.sources.renderSourceCandidates([...attemptEvents, latestStart], []);
assert(!nodes.sourceCandidateList.innerHTML.includes("source-saved-download"), "fetching latest attempt exposed a download");
const latestFailure = {EventID:"evt_fail_c", Sequence:7, EventType:"source.candidate.staging_failed", Payload:{url:"https://example.com/attempt", proposal_event_id:"evt_attempt_proposed", staging_event_id:"evt_start_c"}};
Plasma.sources.renderSourceCandidates([...attemptEvents, latestStart, latestFailure], []);
assert(!nodes.sourceCandidateList.innerHTML.includes("source-saved-download"), "failed latest attempt exposed a download");
const uncorrelatedTerminal = {EventID:"evt_stage_uncorrelated", Sequence:7, EventType:"source.candidate.staged", Payload:{url:"https://example.com/attempt", proposal_event_id:"evt_attempt_proposed", artifact_id:"art_uncorrelated"}};
Plasma.sources.renderSourceCandidates([...attemptEvents, latestStart, uncorrelatedTerminal], []);
assert(!nodes.sourceCandidateList.innerHTML.includes("source-saved-download"), "uncorrelated modern terminal exposed a download");
for (const raw of ["https://docs.atlassian.net/wiki/pages/123/title", "https://docs.atlassian.net/wiki/edit-v2/123", "https://docs.atlassian.net/wiki/%70ages/123/title", "https://docs.atlassian.net/wiki/%65dit-v2/123", "https://docs.atlassian.net/wiki?spaceKey=ENG&pageId=123"]) {
  assert(Plasma.sources.confluenceCandidateKeyFromURL(raw) === "confluence:docs.atlassian.net:123", "Confluence identity mismatch: " + raw);
}
const confluenceCandidate = [
  {EventID:"evt_confluence_proposed", Sequence:1, EventType:"source.candidate.proposed", Payload:{candidates:[{url:"https://docs.atlassian.net/wiki/pages/123/title", title:"Confluence", reason:"reason"}]}},
  {EventID:"evt_confluence_staged", Sequence:2, EventType:"source.candidate.staged", Payload:{url:"https://docs.atlassian.net/wiki/pages/123/title", artifact_id:"art_confluence", proposal_event_id:"evt_confluence_proposed"}}
];
Plasma.sources.renderSourceCandidates(confluenceCandidate, []);
assert(nodes.sourceCandidateList.innerHTML.includes("source-original-link"), "Confluence candidate lost original link");
assert(!nodes.sourceCandidateList.innerHTML.includes("source-saved-download"), "Confluence candidate exposed snapshot download");
const browserCandidate = [
  {EventID:"evt_browser_proposed", Sequence:1, EventType:"source.candidate.proposed", Payload:{candidates:[{url:"https://example.com/app", title:"App", reason:"reason"}]}},
  {EventID:"evt_browser_staged", Sequence:2, EventType:"source.candidate.staged", Payload:{url:"https://example.com/app", artifact_id:"art_raw", proposal_event_id:"evt_browser_proposed", browser_render_candidate:{candidate:true}}}
];
Plasma.sources.renderSourceCandidates(browserCandidate, []);
assert(nodes.sourceCandidateList.innerHTML.includes("초기 저장본 다운로드"), "browser candidate raw artifact label missing");
const snapshot = (state, policy, artifacts) => ({SnapshotID:"src / ?", State:state ? {state} : {}, Access:{RetrievalPolicy:policy}, ArtifactIDs:artifacts, Connector:{ConnectorID:"url", ConnectorType:"url", ExternalURI:"https://example.com/doc"}});
const sourceCases = {
  active:{source:snapshot("active","snapshot_only",["art / 1"]), downloads:1, originals:1},
  browser:{source:{...snapshot("active","snapshot_only",["art / 1"]), Locators:[{locator_type:"full_document", retrieval_method:"browser_render"}]}, downloads:1, originals:1, badge:true},
  removed:{source:snapshot("removed","snapshot_only",["art / 1"]), downloads:0, originals:0},
  superseded:{source:{...snapshot("active","snapshot_only",["art / 1"]), State:{state:"active", superseded:true}}, downloads:0, originals:0},
  live_reference:{source:snapshot("active","live_reference",["art / 1"]), downloads:0, originals:1},
  zero:{source:snapshot("active","snapshot_only",[]), downloads:0, originals:1},
  multi:{source:snapshot("active","snapshot_only",["art / 1","art / 2"]), downloads:0, originals:1},
  confluence:{source:{...snapshot("active","snapshot_only",["art / 1"]), Connector:{ConnectorID:"confluence",ConnectorType:"confluence_cloud",ExternalURI:"confluence://cloud/site_docs.atlassian.net/pages/123"}, Locators:[{locator_type:"confluence_page_body",site_url:"https://docs.atlassian.net/wiki",web_url:"https://docs.atlassian.net/wiki/pages/123/title",page_id:"123"}]}, downloads:0, originals:1, label:"Confluence에서 열기"},
  confluence_other_host:{source:{...snapshot("active","snapshot_only",["art / 1"]), Connector:{ConnectorID:"confluence",ConnectorType:"confluence_cloud",ExternalURI:"confluence://cloud/site_docs.atlassian.net/pages/123"}, Locators:[{locator_type:"confluence_page_body",site_url:"https://docs.atlassian.net/wiki",web_url:"https://attacker.example/page",page_id:"123"}]}, downloads:0, originals:0},
  confluence_credentials:{source:{...snapshot("active","snapshot_only",["art / 1"]), Connector:{ConnectorID:"confluence",ConnectorType:"confluence_cloud",ExternalURI:"confluence://cloud/site_docs.atlassian.net/pages/123"}, Locators:[{locator_type:"confluence_page_body",site_url:"https://docs.atlassian.net/wiki",web_url:"https://person:secret@docs.atlassian.net/wiki/pages/123",page_id:"123"}]}, downloads:0, originals:0},
  liquid2:{source:{...snapshot("active","snapshot_only",["art / 1"]), Connector:{ConnectorID:"liquid2",ConnectorType:"liquid2",ExternalURI:"liquid2://documents/doc_1"}, Locators:[{locator_type:"liquid2_content_range",source_uri:"https://example.com/source"}]}, downloads:0, originals:1},
  liquid2_internal_only:{source:{...snapshot("active","snapshot_only",["art / 1"]), Connector:{ConnectorID:"liquid2",ConnectorType:"liquid2",ExternalURI:"liquid2://documents/doc_1"}, Locators:[{locator_type:"liquid2_content_range"}]}, downloads:0, originals:0},
  upload:{source:{...snapshot("active","snapshot_only",["art / 1"]), Connector:{ConnectorID:"file_upload",ConnectorType:"file_upload",ExternalURI:"file-upload://sha"}}, downloads:1, originals:0}
};
for (const [name, tc] of Object.entries(sourceCases)) {
  Plasma.sources.renderSources([tc.source]);
  const downloads = (nodes.sourceList.innerHTML.match(/source-saved-download/g) || []).length;
  const originals = (nodes.sourceList.innerHTML.match(/source-original-link/g) || []).length;
  assert(downloads === tc.downloads, name + " download count=" + downloads);
  assert(originals === tc.originals, name + " original count=" + originals);
  if (tc.badge) assert(nodes.sourceList.innerHTML.includes("브라우저 렌더링 저장본"), name + " retrieval badge missing");
  if (tc.label) assert(nodes.sourceList.innerHTML.includes(tc.label), name + " original label missing");
  if (name === "active") {
    assert(nodes.sourceList.innerHTML.includes(encodeURIComponent("mis / réservé?")), "source mission id not encoded");
    assert(nodes.sourceList.innerHTML.includes(encodeURIComponent("src / ?")), "snapshot id not encoded");
    assert(nodes.sourceList.innerHTML.includes(encodeURIComponent("art / 1")), "source artifact id not encoded");
  }
}
assert(Plasma.sources.safeExternalSourceURL("javascript:alert(1)") === "", "javascript URL accepted");
assert(Plasma.sources.safeExternalSourceURL("https://user:secret@example.com/doc") === "", "credential URL accepted");
assert(Plasma.sources.safeExternalSourceURL("https://example.com/doc\nheader") === "", "control URL accepted");
Plasma.sources.renderConfluenceResults([
  {Title:"Valid",PageID:"1",SiteURL:"https://docs.atlassian.net/wiki",SourceURI:"https://docs.atlassian.net/wiki/pages/1"},
  {Title:"Cross tenant",PageID:"2",SiteURL:"https://docs.atlassian.net/wiki",SourceURI:"https://attacker.example/page"},
  {Title:"Credentials",PageID:"3",SiteURL:"https://docs.atlassian.net/wiki",SourceURI:"https://person:secret@docs.atlassian.net/wiki/pages/3"}
]);
assert(nodes.confluenceResults.innerHTML.includes("https://docs.atlassian.net/wiki/pages/1"), "valid Confluence result link missing");
assert(!nodes.confluenceResults.innerHTML.includes("attacker.example"), "cross-tenant Confluence result link rendered");
assert(!nodes.confluenceResults.innerHTML.includes("person:secret"), "credentialed Confluence result link rendered");
let prevented = 0;
for (const anchorClass of ["source-saved-download", "source-original-link"]) {
  const nativeAnchor = {tagName:"A", href:"/target", closest(selector){ return selector.includes("a." + anchorClass) ? this : null; }};
  const click = {target:nativeAnchor, preventDefault(){ prevented++; }};
  Plasma.sources.onSourceCandidateListClick(click);
  Plasma.sources.onSourceListClick(click);
}
setTimeout(() => {
  assert(detailCalls === 0, "native source link invoked detail handler");
  assert(actionCalls === 0, "native source link invoked source action handler");
  assert(prevented === 0, "native source link was intercepted");
}, 0);
`
	cmd := exec.Command("node", "-e", fixture)
	cmd.Dir = "../web"
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("source download runtime fixture failed: %v\n%s", err, output)
	}
}

func TestSourceDownloadLabelsAndEncodedRoutesStatic(t *testing.T) {
	candidates := string(mustReadStatic(t, "static/plasma/sources_candidates.js"))
	sources := string(mustReadStatic(t, "static/plasma/sources_rendering.js"))
	if strings.Count(candidates, "초기 저장본 다운로드") != 1 ||
		strings.Count(candidates, "저장본 다운로드") != 2 ||
		strings.Count(sources, "저장본 다운로드") != 1 {
		t.Fatal("expected candidate initial/saved and source saved download labels")
	}
	for _, content := range []string{candidates, sources} {
		if !strings.Contains(content, `class="button-link secondary source-saved-download"`) {
			t.Fatal("saved download must use the existing button-link control style")
		}
		if !strings.Contains(content, "encodeURIComponent(state.missionId)") || !strings.Contains(content, "encodeURIComponent(") {
			t.Fatal("download route must encode identifiers")
		}
	}
}

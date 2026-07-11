package web

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestStaticAppLabelsPendingEvidenceSignalType(t *testing.T) {
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	content := string(script)
	for _, expected := range []string{
		"EVIDENCE_TYPE_LABELS",
		"근거 신호:",
		"evidenceTypeLabel(record.evidence_type)",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected static app to preserve pending evidence signal label %q", expected)
		}
	}
}

func TestStaticAppExposesControllerStrategySelector(t *testing.T) {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`id="controllerStrategy"`,
		`value="v2"`,
		`value="v3"`,
		"조향 전략",
	} {
		if !strings.Contains(string(html), expected) {
			t.Fatalf("expected static app HTML to expose controller strategy selector %q", expected)
		}
	}
	if !strings.Contains(string(script), "controller_strategy") ||
		!strings.Contains(string(script), "controllerStrategy") {
		t.Fatalf("expected static app script to submit controller strategy")
	}
}

func TestStaticAppExposesEnvironmentBadge(t *testing.T) {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	style, err := os.ReadFile("static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(html) + "\n" + string(script) + "\n" + string(style)
	for _, expected := range []string{
		`id="environmentBadge"`,
		"/api/runtime",
		"environment_label",
		"environment-badge",
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("expected static app to expose environment badge %q", expected)
		}
	}
}

func TestStaticReportMarkdownPreviewWrapsAndMarksHeadings(t *testing.T) {
	style, err := os.ReadFile("static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	content := string(style)
	for _, expected := range []string{
		".report-modal-body.turn-markdown",
		"overflow-wrap: anywhere",
		"white-space: pre-wrap",
		".report-modal-body.turn-markdown h1::before { content: \"#\"; }",
		".report-modal-body.turn-markdown h2::before { content: \"##\"; }",
		".report-modal-body.turn-markdown h3::before { content: \"###\"; }",
		".report-modal-body.turn-markdown h4::before { content: \"####\"; }",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected report markdown preview CSS to include %q", expected)
		}
	}
	for _, forbidden := range []string{
		`content: "Part`,
		`content: "Section`,
	} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("report markdown preview CSS should not synthesize report heading text %q", forbidden)
		}
	}
}

func TestStaticDetailModalKeepsTitleBarVisibleWhileBodyScrolls(t *testing.T) {
	style, err := os.ReadFile("static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	content := string(style)
	for _, expected := range []string{
		".modal-card > .panel-head",
		"position: sticky",
		"overflow: hidden",
		".detail-body",
		"display: block",
		"flex: 1 1 auto",
		"overflow: auto",
		"overscroll-behavior: contain",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected detail modal CSS to keep the title bar visible while body scrolls: %q", expected)
		}
	}
}

func TestStaticAppExposesWorkflowControlsWithoutTerminalUI(t *testing.T) {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(html) + "\n" + string(script)
	for _, expected := range []string{
		`id="workflowInstruction"`,
		`<label class="field-label hidden" for="workflowStepInstructionMode">스텝 지시 방식</label>`,
		`id="workflowStepInstructionMode" class="hidden" aria-hidden="true" tabindex="-1"`,
		`<option value="layered" selected>3층 지시</option>`,
		`id="workflowLayeredFields" class="workflow-layered-fields"`,
		`id="draftWorkflowGoalButton"`,
		`id="workflowRunGoal"`,
		`id="workflowStepInstruction"`,
		`id="startWorkflowButton"`,
		`id="stopWorkflowButton"`,
		"/workflows/goal_draft",
		"workflowRawInputValue",
		`$("turnText").addEventListener("input", onWorkflowRawInput)`,
		"state.workflowGoalDraftPending &&",
		"/workflows",
		"workflow_runs",
		"step_instruction_mode",
		"workflowStepInstructionMode",
		"updateWorkflowStepInstructionMode();",
		"user_instruction_raw",
		"run_goal",
		"max_steps: 10",
		"max_duration_ms: 1500000",
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("expected static app to expose workflow control %q", expected)
		}
	}
	for _, forbidden := range []string{
		"PTY",
		"terminal",
		"터미널",
		`<option value="current"`,
		`id="workflowStepInstructionMode">`,
		`id="workflowLayeredFields" class="workflow-layered-fields hidden"`,
		"3층 지시 실험",
		"3층 지시 선택 필요",
	} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("workflow controls should not expose terminal UI term %q", forbidden)
		}
	}
}

func TestStaticAppExposesSourceCandidateIndicators(t *testing.T) {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(html) + "\n" + string(script)
	for _, expected := range []string{
		`id="sourceTabCandidateCount"`,
		`id="sourceCandidateNotice"`,
		`id="openSourceCandidatesButton"`,
		"plasma.activeMissionId",
		"updateSourceCandidateIndicators",
		"openSourceCandidatesTab",
		`classList.toggle("hidden", isEmpty)`,
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("expected static app to expose source candidate indicator %q", expected)
		}
	}
}

func TestStaticAppBulkSourceCandidateApprovalUsesURLRouter(t *testing.T) {
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	content := string(script)
	for _, expected := range []string{
		"function sourceCandidateTitleForURL(url)",
		"await addURLSource(url, sourceCandidateTitleForURL(url))",
		"sourceRouteForURL(url)",
		`if (looksLikeConfluenceURL(value)) return "confluence/url"`,
		"looksLikePDFSourceError(err)",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected bulk source candidate approval to reuse routed URL source addition %q", expected)
		}
	}
	bulkBody := jsFunctionBody(t, content, "bulkSourceCandidateAction")
	if strings.Contains(bulkBody, "/sources/url`") {
		t.Fatalf("bulk source candidate approval must not post every candidate to the generic URL source route")
	}
}

func TestStaticAppSourceCandidateFilterUsesConfluenceLocator(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for semantic static app JS fixture test")
	}
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	content := string(script)
	nodeScript := jsFunctionSource(t, content, "normalizeSourceURL") + "\n" +
		jsFunctionSource(t, content, "acceptedSourceCandidateKeys") + "\n" +
		jsFunctionSource(t, content, "sourceCandidateAccepted") + "\n" +
		jsFunctionSource(t, content, "sourceLocators") + "\n" +
		jsFunctionSource(t, content, "confluenceCandidateKeyFromURL") + "\n" +
		jsFunctionSource(t, content, "confluenceSourceKey") + `
const sources = [{
  Connector: {
    ExternalSourceID: "site_docs.atlassian.net:123",
    ExternalURI: "confluence://cloud/site_docs.atlassian.net/pages/123"
  },
  Locators: JSON.stringify([{
    site_url: "https://docs.atlassian.net/wiki",
    page_id: "123"
  }])
}];
const existing = acceptedSourceCandidateKeys(sources);
const accepted = sourceCandidateAccepted(existing, normalizeSourceURL("https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap"));
const other = sourceCandidateAccepted(existing, normalizeSourceURL("https://docs.atlassian.net/wiki/spaces/ENG/pages/456/Roadmap"));
process.stdout.write(JSON.stringify({ accepted, other }));
`
	output, err := exec.Command("node", "-e", nodeScript).CombinedOutput()
	if err != nil {
		t.Fatalf("execute Confluence source candidate filter fixture: %v\n%s", err, string(output))
	}
	var got struct {
		Accepted bool `json:"accepted"`
		Other    bool `json:"other"`
	}
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("decode Confluence source candidate filter fixture: %v\n%s", err, string(output))
	}
	if !got.Accepted || got.Other {
		t.Fatalf("expected only the accepted Confluence page candidate to be hidden, got %#v", got)
	}
}

func TestStaticAppSourceRefreshUsesExistingDetailRenderer(t *testing.T) {
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	content := string(script)
	if !strings.Contains(content, "function renderDetail()") {
		t.Fatalf("expected static app to define renderDetail")
	}
	if strings.Contains(content, "renderMissionDetail(") {
		t.Fatalf("static app should not call missing renderMissionDetail")
	}
}

func TestStaticAppExposesReportHumanizeRetry(t *testing.T) {
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	content := string(script)
	for _, expected := range []string{
		"H5 말투 보정 다시 생성",
		"start-humanized-markdown-artifact",
		"exportReportArtifactHumanizedMarkdown",
		"/humanized_markdown_export",
		"H5 말투 보정 시작 실패",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected static app to expose report humanize retry %q", expected)
		}
	}
}

func TestStaticAppTreatsHumanizeSkippedAsTerminalState(t *testing.T) {
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	content := string(script)
	for _, expected := range []string{
		`if (event.EventType === "report.humanize.skipped")`,
		`return { state: "skipped", event };`,
		`if (status.state === "skipped" && wasPending)`,
		`H5 말투 보정 결과가 원본과 같아 별도 artifact를 만들지 않았습니다.`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected static app to treat H5 skipped as a terminal non-error state %q", expected)
		}
	}
}

func TestStaticAppExposesConfluenceSourceWorkflow(t *testing.T) {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	appScript, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceScript, err := os.ReadFile("static/confluence.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceWorkflowScript, err := os.ReadFile("static/confluence_workflow.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceSettingsScript, err := os.ReadFile("static/confluence_settings.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceAccessScript, err := os.ReadFile("static/confluence_access.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceBrowseScript, err := os.ReadFile("static/confluence_browse.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceReviewScript, err := os.ReadFile("static/confluence_review.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceUpdateScript, err := os.ReadFile("static/confluence_update.js")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(html) + "\n" + string(appScript) + "\n" + string(confluenceScript) + "\n" + string(confluenceSettingsScript) + "\n" + string(confluenceAccessScript) + "\n" + string(confluenceWorkflowScript) + "\n" + string(confluenceBrowseScript) + "\n" + string(confluenceReviewScript) + "\n" + string(confluenceUpdateScript)
	for _, expected := range []string{
		`id="confluenceSourceDetails"`,
		`data-tab="settings"`,
		`id="confluenceSettingsAPIForm"`,
		`id="confluenceSettingsConnections"`,
		`data-conn-action="rename"`,
		`id="confluenceAccessEnable"`,
		`id="confluenceAccessDisable"`,
		`id="confluenceOneClickStart"`,
		`id="confluenceFlowStatus"`,
		`id="confluenceURLForm"`,
		`id="confluencePageURL"`,
		`id="confluenceAddURLButton"`,
		`https://id.atlassian.com/manage-profile/security/api-tokens`,
		`id="confluenceLoadSpaces"`,
		`id="confluenceLoadMoreSpaces"`,
		`id="confluenceLoadMorePages"`,
		`id="confluenceSpaces"`,
		`id="confluencePages"`,
		`id="confluencePreviewPanel"`,
		`id="confluenceRangeSelect"`,
		`id="confluenceUpdatePanel"`,
		`id="confluenceSearchForm"`,
		`id="confluenceResults"`,
		`/static/confluence.js`,
		`/static/confluence_settings.js`,
		`/static/confluence_access.js`,
		`/static/confluence_workflow.js`,
		`/static/confluence_browse.js`,
		`/static/confluence_review.js`,
		`/static/confluence_update.js`,
		`/api/settings/connectors/confluence/connections`,
		`/connector-access/confluence`,
		`/sources/confluence/spaces`,
		`/sources/confluence/space-pages`,
		`/sources/confluence/children`,
		`/sources/confluence/search`,
		`/sources/confluence/url`,
		`/sources/confluence/preview`,
		`/sources/confluence/snapshot`,
		`/sources/confluence/check-update`,
		`/sources/confluence/update-preview`,
		`/sources/confluence/update`,
		`data-confluence-candidate-index`,
		`data-confluence-page-index`,
		`data-confluence-source-update`,
		`confluence_page_range`,
		`clearConfluenceSearchResults`,
		`confluenceSearchContext`,
		`confluenceBrowseContext`,
		`loadMoreConfluenceSpaces`,
		`loadMoreConfluencePages`,
		`spaces_cursor: context.spaces_cursor || ""`,
		"renderConfluenceSpaces(state.confluenceSpaces);\n    renderConfluencePages([]);",
		`previewConfluenceCandidate`,
		`approveConfluenceSnapshot`,
		`preview.full_body_too_large || preview.FullBodyTooLarge`,
		`rangeRequired && !ranges.length`,
		`runConfluenceOneClickFlow`,
		`addConfluenceURLSource`,
		`sourceCandidateTitleForURL(url)`,
		`connection_id: connectionID`,
		`cloud_id: cloudID`,
		`API token 연결 추가`,
		`confluenceSettingsAPIToken").value = ""`,
		`confluenceCandidateDetailPayload(candidate)`,
		`업데이트 검토`,
	} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("expected static app to expose Confluence workflow %q", expected)
		}
	}
	if strings.Contains(combined, `id="confluenceAPICloudID"`) || strings.Contains(combined, "cloud id가 필요") {
		t.Fatalf("Confluence API token fallback must not ask users for cloud id")
	}
	sourceDetails := htmlSection(t, string(html), `id="confluenceSourceDetails"`, `id="liquid2Form"`)
	for _, forbidden := range []string{`id="confluenceSettingsOAuthForm"`, `id="confluenceSettingsAPIForm"`, `id="confluenceSettingsConnectionDisplayName"`, "Atlassian API token"} {
		if strings.Contains(sourceDetails, forbidden) {
			t.Fatalf("mission Sources must not contain Settings-only Confluence control %q", forbidden)
		}
	}
	if strings.Contains(combined, `id="confluenceSettingsOAuthForm"`) ||
		strings.Contains(combined, `/api/settings/connectors/confluence/oauth/start`) ||
		strings.Contains(combined, `window.open("about:blank", "plasmaConfluenceOAuth")`) {
		t.Fatalf("Confluence OAuth UI must not be exposed in Plasma 0.0")
	}
	for _, forbidden := range []string{`/api/missions/${state.missionId}/sources/confluence/oauth/start`, `/api/missions/${state.missionId}/sources/confluence/connections`, `/api/missions/${state.missionId}/sources/confluence/sites`} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("static UI must not call legacy mission lifecycle route %q", forbidden)
		}
	}
	setFormsBody := jsFunctionBody(t, string(appScript), "setFormsEnabled")
	for _, forbidden := range []string{"confluenceSettingsOAuthForm", "confluenceSettingsAPIForm", "confluenceSettingsAPIToken", "confluenceSettingsOAuthClientSecret"} {
		if strings.Contains(setFormsBody, forbidden) {
			t.Fatalf("global Confluence Settings control %q must not be disabled by mission-bound form state", forbidden)
		}
	}
	if strings.Contains(combined, "cloud ${info.cloud_id}") {
		t.Fatalf("Confluence source metadata must not display the internal cloud id")
	}
	if strings.Contains(combined, "if (info.external_uri) parts.push(info.external_uri)") ||
		!strings.Contains(combined, "confluenceDisplayableExternalURI(info.external_uri)") {
		t.Fatalf("Confluence source metadata must not render raw internal external_uri values")
	}
	if strings.Contains(combined, `data-detail-title="소스 상세" data-detail-json="${escapeAttr(JSON.stringify(source))}"`) ||
		!strings.Contains(combined, "sourceDetailPayload(source, confluence)") {
		t.Fatalf("Confluence source detail modal must use a sanitized user-facing payload")
	}
	if strings.Contains(string(confluenceScript), `data-detail-title="Confluence 검색 결과" data-detail-json="${escapeAttr(JSON.stringify(candidate))}"`) ||
		strings.Contains(string(confluenceScript), "connector.ExternalURI") ||
		strings.Contains(string(confluenceScript), "connector.external_uri") {
		t.Fatalf("Confluence search candidate detail must not expose the raw connector payload")
	}
}

func TestConfluenceSourceDetailPayloadIsSanitized(t *testing.T) {
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	body := jsFunctionBody(t, string(script), "sourceDetailPayload")
	for _, expected := range []string{
		`type: "confluence_source"`,
		"snapshot_id:",
		"title:",
		"connector_id:",
		"connector_version:",
		"site_url:",
		"page_id:",
		"version:",
		"retrieval_policy:",
		"state:",
		"confluenceDisplayableExternalURI(confluence.external_uri)",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected Confluence detail payload to include user-facing field %q", expected)
		}
	}
	for _, forbidden := range []string{
		"cloud_id",
		"CloudID",
		"ExternalSourceID",
		"external_source_id",
		"Locators",
		"locators",
		"confluence://",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("Confluence detail payload must not include internal identity field %q", forbidden)
		}
	}
}

func TestConfluenceSourceDetailPayloadFixtureIsSanitized(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for semantic static app JS fixture test")
	}
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	nodeScript := jsFunctionSource(t, string(script), "confluenceDisplayableExternalURI") + "\n" +
		jsFunctionSource(t, string(script), "sourceDetailPayload") + `
const source = {
  SnapshotID: "src_1",
  Title: "Roadmap",
  Connector: {
    ConnectorID: "confluence",
    ConnectorVersion: "v1",
    ExternalSourceID: "site_docs.atlassian.net:123",
    ExternalURI: "confluence://cloud/site_docs.atlassian.net/pages/123"
  },
  Locators: JSON.stringify([{
    cloud_id: "site_docs.atlassian.net",
    site_url: "https://docs.atlassian.net/wiki",
    page_id: "123"
  }]),
  Access: { RetrievalPolicy: "snapshot_only" },
  State: { State: "active" }
};
const confluence = {
  site_url: "https://docs.atlassian.net/wiki",
  page_id: "123",
  version: "7",
  external_uri: "confluence://cloud/site_docs.atlassian.net/pages/123"
};
process.stdout.write(JSON.stringify(sourceDetailPayload(source, confluence)));
`
	output, err := exec.Command("node", "-e", nodeScript).CombinedOutput()
	if err != nil {
		t.Fatalf("execute sourceDetailPayload fixture: %v\n%s", err, string(output))
	}
	var detail map[string]any
	if err := json.Unmarshal(output, &detail); err != nil {
		t.Fatalf("decode sourceDetailPayload fixture result: %v\n%s", err, string(output))
	}
	for key, expected := range map[string]string{
		"type":             "confluence_source",
		"snapshot_id":      "src_1",
		"title":            "Roadmap",
		"connector_id":     "confluence",
		"site_url":         "https://docs.atlassian.net/wiki",
		"page_id":          "123",
		"version":          "7",
		"retrieval_policy": "snapshot_only",
		"state":            "active",
	} {
		if got, _ := detail[key].(string); got != expected {
			t.Fatalf("expected sanitized detail field %s=%q, got %#v in %#v", key, expected, detail[key], detail)
		}
	}
	raw := string(output)
	for _, forbidden := range []string{
		"cloud_id",
		"CloudID",
		"ExternalSourceID",
		"external_source_id",
		"ExternalURI",
		"external_uri",
		"Locators",
		"locators",
		"confluence://",
		"site_docs.atlassian.net:123",
	} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("sanitized detail payload leaked internal field/value %q: %s", forbidden, raw)
		}
	}
}

func TestPDFLocatorRecognizesUploadedPDFSource(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for semantic static app JS fixture test")
	}
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	nodeScript := jsFunctionSource(t, string(script), "sourceLocatorType") + "\n" +
		jsFunctionSource(t, string(script), "pdfLocator") + `
const canonical = {
  Locators: JSON.stringify([{
    locator_type: "pdf_document",
    original_filename: "Paper Final.pdf",
    sanitized_filename: "Paper-Final.pdf",
    mime_type: "application/pdf",
    byte_size: 2048,
    content_kind: "pdf",
    extraction_support: "pdf_text"
  }])
};
const legacy = {
  Locators: JSON.stringify([{
    kind: "file_upload",
    original_filename: "Legacy Paper.pdf",
    sanitized_filename: "Legacy-Paper.pdf",
    media_type: "application/pdf",
    byte_size: 1024,
    content_kind: "pdf"
  }])
};
process.stdout.write(JSON.stringify({
  canonical: pdfLocator(canonical),
  legacy: pdfLocator(legacy)
}));
`
	output, err := exec.Command("node", "-e", nodeScript).CombinedOutput()
	if err != nil {
		t.Fatalf("execute pdfLocator fixture: %v\n%s", err, string(output))
	}
	var result map[string]map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode pdfLocator fixture result: %v\n%s", err, string(output))
	}
	if got, _ := result["canonical"]["filename"].(string); got != "Paper-Final.pdf" {
		t.Fatalf("expected canonical uploaded PDF filename, got %#v in %#v", result["canonical"]["filename"], result)
	}
	if got, _ := result["canonical"]["extraction_support"].(string); got != "pdf_text" {
		t.Fatalf("expected canonical uploaded PDF extraction support, got %#v in %#v", result["canonical"]["extraction_support"], result)
	}
	if got, _ := result["legacy"]["filename"].(string); got != "Legacy-Paper.pdf" {
		t.Fatalf("expected legacy uploaded PDF filename, got %#v in %#v", result["legacy"]["filename"], result)
	}
	if got, _ := result["legacy"]["mime_type"].(string); got != "application/pdf" {
		t.Fatalf("expected legacy uploaded PDF MIME type, got %#v in %#v", result["legacy"]["mime_type"], result)
	}
}

func TestUploadedFileLegacyLocatorsRenderAsMediaOrDocument(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for semantic static app JS fixture test")
	}
	script, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	nodeScript := jsFunctionSource(t, string(script), "sourceLocatorType") + "\n" +
		jsFunctionSource(t, string(script), "sourceConnectorType") + "\n" +
		jsFunctionSource(t, string(script), "uploadedFileContentKind") + "\n" +
		jsFunctionSource(t, string(script), "uploadedFileMediaType") + "\n" +
		jsFunctionSource(t, string(script), "uploadedFileFilename") + "\n" +
		jsFunctionSource(t, string(script), "mediaLocator") + "\n" +
		jsFunctionSource(t, string(script), "documentLocator") + `
const legacyImage = {
  Connector: { ConnectorType: "file_upload" },
  Locators: JSON.stringify([{
    kind: "file_upload",
    original_filename: "Legacy Pixel.png",
    sanitized_filename: "Legacy-Pixel.png",
    media_type: "image/png",
    byte_size: 256,
    content_kind: "image"
  }])
};
const legacyText = {
  Connector: { ConnectorType: "file_upload" },
  Locators: JSON.stringify([{
    kind: "file_upload",
    original_filename: "Legacy Notes.md",
    sanitized_filename: "Legacy-Notes.md",
    media_type: "text/markdown",
    byte_size: 128,
    content_kind: "text"
  }])
};
process.stdout.write(JSON.stringify({
  image: mediaLocator(legacyImage),
  text: documentLocator(legacyText)
}));
`
	output, err := exec.Command("node", "-e", nodeScript).CombinedOutput()
	if err != nil {
		t.Fatalf("execute uploaded locator fixture: %v\n%s", err, string(output))
	}
	var result map[string]map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode uploaded locator fixture result: %v\n%s", err, string(output))
	}
	if got, _ := result["image"]["media_kind"].(string); got != "image" {
		t.Fatalf("expected legacy uploaded image media kind, got %#v in %#v", result["image"]["media_kind"], result)
	}
	if got, _ := result["image"]["filename"].(string); got != "Legacy-Pixel.png" {
		t.Fatalf("expected legacy uploaded image filename, got %#v in %#v", result["image"]["filename"], result)
	}
	if got, _ := result["text"]["filename"].(string); got != "Legacy-Notes.md" {
		t.Fatalf("expected legacy uploaded text filename, got %#v in %#v", result["text"]["filename"], result)
	}
	if got, _ := result["text"]["mime_type"].(string); got != "text/markdown" {
		t.Fatalf("expected legacy uploaded text MIME type, got %#v in %#v", result["text"]["mime_type"], result)
	}
}

func TestConfluenceCandidateDetailPayloadIsSanitized(t *testing.T) {
	script, err := os.ReadFile("static/confluence.js")
	if err != nil {
		t.Fatal(err)
	}
	body := jsFunctionBody(t, string(script), "confluenceCandidateDetailPayload")
	for _, expected := range []string{
		`type: "confluence_candidate"`,
		"title",
		"site_url",
		"site_host",
		"page_id",
		"space_key",
		"version",
		"updated_at",
		"can_snapshot",
		"confluenceDisplayableExternalURI",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected Confluence candidate detail payload to include user-facing field %q", expected)
		}
	}
	for _, forbidden := range []string{
		"cloud_id",
		"CloudID",
		"Connector",
		"connector",
		"ExternalSourceID",
		"external_source_id",
		"confluence://",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("Confluence candidate detail payload must not include internal identity field %q", forbidden)
		}
	}
}

func TestConfluenceCandidateDetailPayloadFixtureIsSanitized(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for semantic static app JS fixture test")
	}
	appScript, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	confluenceScript, err := os.ReadFile("static/confluence.js")
	if err != nil {
		t.Fatal(err)
	}
	browseScript, err := os.ReadFile("static/confluence_browse.js")
	if err != nil {
		t.Fatal(err)
	}
	nodeScript := jsFunctionSource(t, string(appScript), "confluenceDisplayableExternalURI") + "\n" +
		jsFunctionSource(t, string(appScript), "confluenceExternalURIHost") + "\n" +
		jsFunctionSource(t, string(browseScript), "confluenceCandidatePageID") + "\n" +
		jsFunctionSource(t, string(confluenceScript), "confluenceCandidateDetailPayload") + `
const candidate = {
  CloudID: "site_docs.atlassian.net",
  SiteURL: "https://docs.atlassian.net/wiki",
  SpaceKey: "ENG",
  Title: "Roadmap",
  SourceURI: "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap",
  Version: 7,
  UpdatedAt: "2026-07-06T01:02:03Z",
  CanSnapshot: true,
  Connector: {
    ExternalSourceID: "site_docs.atlassian.net:123",
    ExternalURI: "confluence://cloud/site_docs.atlassian.net/pages/123"
  }
};
process.stdout.write(JSON.stringify(confluenceCandidateDetailPayload(candidate)));
`
	output, err := exec.Command("node", "-e", nodeScript).CombinedOutput()
	if err != nil {
		t.Fatalf("execute confluenceCandidateDetailPayload fixture: %v\n%s", err, string(output))
	}
	var detail map[string]any
	if err := json.Unmarshal(output, &detail); err != nil {
		t.Fatalf("decode confluenceCandidateDetailPayload fixture result: %v\n%s", err, string(output))
	}
	for key, expected := range map[string]string{
		"type":       "confluence_candidate",
		"title":      "Roadmap",
		"site_url":   "https://docs.atlassian.net/wiki",
		"site_host":  "docs.atlassian.net",
		"page_id":    "123",
		"space_key":  "ENG",
		"updated_at": "2026-07-06T01:02:03Z",
		"source_uri": "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap",
	} {
		if got, _ := detail[key].(string); got != expected {
			t.Fatalf("expected sanitized candidate detail field %s=%q, got %#v in %#v", key, expected, detail[key], detail)
		}
	}
	if got, _ := detail["version"].(float64); got != 7 {
		t.Fatalf("expected sanitized candidate version 7, got %#v", detail["version"])
	}
	if got, _ := detail["can_snapshot"].(bool); !got {
		t.Fatalf("expected sanitized candidate can_snapshot true, got %#v", detail["can_snapshot"])
	}
	raw := string(output)
	for _, forbidden := range []string{
		"cloud_id",
		"CloudID",
		"Connector",
		"connector",
		"ExternalSourceID",
		"external_source_id",
		"ExternalURI",
		"external_uri",
		"confluence://",
		"site_docs.atlassian.net",
	} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("sanitized candidate detail payload leaked internal field/value %q: %s", forbidden, raw)
		}
	}
}

func TestConfluenceDeleteConnectionSendsJSONBody(t *testing.T) {
	script, err := os.ReadFile("static/confluence_settings.js")
	if err != nil {
		t.Fatal(err)
	}
	content := string(script)
	functionStart := strings.Index(content, `async function deleteConfluenceSettingsConnection(connectionID)`)
	if functionStart < 0 {
		t.Fatalf("expected delete connection function in Confluence settings script")
	}
	deletePath := `/api/settings/connectors/confluence/connections/${encodeURIComponent(connectionID)}`
	start := strings.Index(content[functionStart:], deletePath)
	if start < 0 {
		t.Fatalf("expected delete connection path in Confluence settings script")
	}
	start += functionStart
	end := strings.Index(content[start:], `clearConfluenceDiscovery();`)
	if end < 0 {
		t.Fatalf("expected delete connection call before discovery clear")
	}
	deleteCall := content[start : start+end]
	for _, expected := range []string{
		`method: "DELETE"`,
		`body: {}`,
	} {
		if !strings.Contains(deleteCall, expected) {
			t.Fatalf("expected delete connection call to include %q, got:\n%s", expected, deleteCall)
		}
	}
}

func jsFunctionSource(t *testing.T, content string, name string) string {
	t.Helper()
	start, end := jsFunctionBounds(t, content, name)
	return content[start:end]
}

func htmlSection(t *testing.T, content string, startMarker string, endMarker string) string {
	t.Helper()
	start := strings.Index(content, startMarker)
	if start < 0 {
		t.Fatalf("expected HTML marker %q", startMarker)
	}
	end := strings.Index(content[start:], endMarker)
	if end < 0 {
		t.Fatalf("expected HTML marker %q after %q", endMarker, startMarker)
	}
	return content[start : start+end]
}

func jsFunctionBody(t *testing.T, content string, name string) string {
	t.Helper()
	_, end := jsFunctionBounds(t, content, name)
	start := strings.Index(content, "function "+name+"(")
	if start < 0 {
		t.Fatalf("expected function %s in static app", name)
	}
	brace := strings.Index(content[start:], "{")
	if brace < 0 {
		t.Fatalf("expected function %s body", name)
	}
	bodyStart := start + brace
	depth := 0
	for i := bodyStart; i < end; i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return content[bodyStart+1 : i]
			}
		}
	}
	t.Fatalf("function %s body did not terminate", name)
	return ""
}

func jsFunctionBounds(t *testing.T, content string, name string) (int, int) {
	t.Helper()
	start := strings.Index(content, "function "+name+"(")
	if start < 0 {
		t.Fatalf("expected function %s in static app", name)
	}
	brace := strings.Index(content[start:], "{")
	if brace < 0 {
		t.Fatalf("expected function %s body", name)
	}
	bodyStart := start + brace
	depth := 0
	for i := bodyStart; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return start, i + 1
			}
		}
	}
	t.Fatalf("function %s body did not terminate", name)
	return 0, 0
}

func TestConfluenceCommonRendererDoesNotOwnPreviewApprovalButtons(t *testing.T) {
	files := []string{
		"static/app.js",
		"static/confluence.js",
	}
	for _, file := range files {
		script, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		content := string(script)
		start := strings.Index(content, `for (const id of [`)
		if start < 0 {
			t.Fatalf("expected common control loop in %s", file)
		}
		end := strings.Index(content[start:], `]) {`)
		if end < 0 {
			t.Fatalf("expected end of common control loop in %s", file)
		}
		loop := content[start : start+end]
		for _, forbidden := range []string{
			"confluenceApproveFullSnapshot",
			"confluenceApproveRangeSnapshot",
			"confluenceUpdatePreviewButton",
			"confluenceApproveUpdate",
		} {
			if strings.Contains(loop, forbidden) {
				t.Fatalf("common Confluence renderer in %s must not own %s disabled state", file, forbidden)
			}
		}
	}
}

func TestConfluenceBusyStateProtectsApprovalActions(t *testing.T) {
	common, err := os.ReadFile("static/confluence.js")
	if err != nil {
		t.Fatal(err)
	}
	review, err := os.ReadFile("static/confluence_review.js")
	if err != nil {
		t.Fatal(err)
	}
	update, err := os.ReadFile("static/confluence_update.js")
	if err != nil {
		t.Fatal(err)
	}
	commonContent := string(common)
	for _, expected := range []string{
		"renderConfluencePreview(state.confluencePreview)",
		"renderConfluenceUpdatePanel(state.confluenceUpdatePreview)",
	} {
		if !strings.Contains(commonContent, expected) {
			t.Fatalf("expected Confluence busy setter to refresh approval panels with %q", expected)
		}
	}
	reviewContent := string(review)
	for _, expected := range []string{
		"if (!requireMission() || state.confluenceBusy) return;",
		"if (!requireMission() || !page || state.confluenceBusy) return;",
		"if (state.confluenceBusy) return;\n  const preview = state.confluencePreview;",
	} {
		if !strings.Contains(reviewContent, expected) {
			t.Fatalf("expected Confluence review action guard %q", expected)
		}
	}
	updateContent := string(update)
	for _, expected := range []string{
		"if (!requireMission() || state.confluenceBusy) return;",
		"state.confluenceBusy || (!preview.new_page && !preview.NewPage)",
		"async function previewConfluenceUpdate() {\n  if (state.confluenceBusy) return;",
		"async function approveConfluenceUpdate() {\n  if (state.confluenceBusy) return;",
	} {
		if !strings.Contains(updateContent, expected) {
			t.Fatalf("expected Confluence update busy guard %q", expected)
		}
	}
}

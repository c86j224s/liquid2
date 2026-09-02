package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportilphase0"
	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
	cdpbrowser "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

type reportILBrowserLayout struct {
	ViewportWidth         float64 `json:"viewport_width"`
	DocumentClientWidth   float64 `json:"document_client_width"`
	DocumentScrollWidth   float64 `json:"document_scroll_width"`
	ExperimentalCardWidth float64 `json:"experimental_card_width"`
	GenerateButtonWidth   float64 `json:"generate_button_width"`
	StageCount            int     `json:"stage_count"`
	StageFirstLeft        float64 `json:"stage_first_left"`
	StageSecondLeft       float64 `json:"stage_second_left"`
	StageFirstTop         float64 `json:"stage_first_top"`
	StageSecondTop        float64 `json:"stage_second_top"`
	LineageCount          int     `json:"lineage_count"`
	DownloadCount         int     `json:"download_count"`
	ViewCount             int     `json:"view_count"`
	ClassicActions        int     `json:"classic_actions"`
	RetryActions          int     `json:"retry_actions"`
	NoRetryVisible        bool    `json:"no_retry_visible"`
	PipelineGraphCount    int     `json:"pipeline_graph_count"`
	PipelineGraphWidth    float64 `json:"pipeline_graph_width"`
	FanoutGraphCount      int     `json:"fanout_graph_count"`
	SectionNodeCount      int     `json:"section_node_count"`
	PartEditNodeCount     int     `json:"part_edit_node_count"`
	ParallelSectionLabel  bool    `json:"parallel_section_label"`
	SequentialPartLabel   bool    `json:"sequential_part_label"`
}

type reportILBrowserReceipt struct {
	UserAgent          string                `json:"user_agent"`
	Desktop            reportILBrowserLayout `json:"desktop"`
	Mobile             reportILBrowserLayout `json:"mobile"`
	IntermediateTitle  string                `json:"intermediate_title"`
	IntermediateCard   string                `json:"intermediate_card"`
	StoredHTMLPath     string                `json:"stored_html_path"`
	StoredHTMLText     string                `json:"stored_html_text"`
	DownloadedFiles    []string              `json:"downloaded_files"`
	ScreenshotsWritten bool                  `json:"screenshots_written"`
}

type reportILUnverifiedBrowserReceipt struct {
	RequestFamily     string `json:"request_family"`
	RequestMode       string `json:"request_mode"`
	RequestProfile    string `json:"request_profile"`
	ManifestMode      string `json:"manifest_mode"`
	CardCount         int    `json:"card_count"`
	CardText          string `json:"card_text"`
	DownloadCount     int    `json:"download_count"`
	IntermediateCount int    `json:"intermediate_count"`
	ManifestProfile   string `json:"manifest_profile"`
}

func TestReportILExperimentalBrowserDogfood(t *testing.T) {
	chromePath := testChromePath()
	if chromePath == "" {
		t.Skip("Chrome or Chromium is required for the Experimental IL browser fixture")
	}

	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	missionID := "mis_il_browser"
	if _, err := service.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "Experimental IL browser dogfood"}); err != nil {
		t.Fatal(err)
	}
	bundle := seedILHTTPGateBundle(t, ctx, service, missionID)
	seedILBrowserFailure(t, ctx, service, missionID)

	handler := NewServer(service, Options{}).(*Server)
	handler.staticDir = filepath.Join("static")
	server := httptest.NewServer(handler)
	defer server.Close()

	profileDir := t.TempDir()
	downloadDir := t.TempDir()
	var browserOutput bytes.Buffer
	allocatorOptions := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
		chromedp.UserDataDir(profileDir),
		chromedp.DisableGPU,
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.CombinedOutput(&browserOutput),
	)
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(ctx, allocatorOptions...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx)
	defer cancelBrowser()
	browserCtx, cancelTimeout := context.WithTimeout(browserCtx, 60*time.Second)
	defer cancelTimeout()

	if err := chromedp.Run(browserCtx,
		cdpbrowser.SetDownloadBehavior(cdpbrowser.SetDownloadBehaviorBehaviorAllow).
			WithDownloadPath(downloadDir).
			WithEventsEnabled(true),
		chromedp.EmulateViewport(1440, 1000),
		chromedp.Navigate(server.URL+"/#reports"),
		chromedp.Poll(`document.querySelector('#missionList button.item[data-mission-id]') !== null`, nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Evaluate(`document.querySelector('#missionList button.item[data-mission-id]').click()`, nil),
		chromedp.Poll(`window.Plasma?.state?.missionId === 'mis_il_browser'`, nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Poll(`document.querySelector('.report-il-card') !== null && document.querySelectorAll('.report-il-stage').length === 9`, nil, chromedp.WithPollingTimeout(10*time.Second)),
	); err != nil {
		t.Fatalf("open Experimental IL browser fixture: %v\nChrome output:\n%s", err, browserOutput.String())
	}

	receipt := reportILBrowserReceipt{}
	if err := chromedp.Run(browserCtx,
		chromedp.Evaluate(`navigator.userAgent`, &receipt.UserAgent),
		chromedp.Evaluate(reportILBrowserLayoutExpression, &receipt.Desktop),
	); err != nil {
		t.Fatal(err)
	}
	assertReportILDesktopLayout(t, receipt.Desktop)
	writeReportILBrowserScreenshot(t, browserCtx, "desktop.png", &receipt)

	if err := chromedp.Run(browserCtx,
		chromedp.Click(`.report-il-card button[data-action="view-text-artifact"]`, chromedp.ByQuery),
		chromedp.Poll(`!document.querySelector('#detailModal').classList.contains('hidden') && document.querySelector('#detailTitle').textContent.includes('원고 설계')`, nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Text(`#detailTitle`, &receipt.IntermediateTitle, chromedp.ByID),
		chromedp.Evaluate(`window.Plasma.state.selectedReportKey`, &receipt.IntermediateCard),
	); err != nil {
		t.Fatalf("open intermediate artifact: %v", err)
	}
	if !strings.Contains(receipt.IntermediateTitle, "원고 설계") || !strings.Contains(receipt.IntermediateTitle, "art_il_http_gate_narrative") || receipt.IntermediateCard != "artifact:"+bundle.MarkdownID {
		t.Fatalf("intermediate preview lost bundle identity: title=%q card=%q", receipt.IntermediateTitle, receipt.IntermediateCard)
	}
	if err := chromedp.Run(browserCtx, chromedp.Click(`#closeDetail`, chromedp.ByID)); err != nil {
		t.Fatal(err)
	}

	popupTarget := chromedp.WaitNewTarget(browserCtx, func(info *target.Info) bool {
		return info.Type == "page" && info.OpenerID != ""
	})
	if err := chromedp.Run(browserCtx, chromedp.Click(`.report-il-card button[data-action="view-stored-html-artifact"]`, chromedp.ByQuery)); err != nil {
		t.Fatalf("open stored HTML artifact: %v", err)
	}
	var popupID target.ID
	select {
	case popupID = <-popupTarget:
	case <-time.After(10 * time.Second):
		t.Fatal("stored HTML preview did not open a browser target")
	}
	popupCtx, cancelPopup := chromedp.NewContext(browserCtx, chromedp.WithTargetID(popupID))
	defer cancelPopup()
	if err := chromedp.Run(popupCtx,
		chromedp.Poll(`location.pathname.includes('/artifacts/art_il_http_gate_html/preview')`, nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Evaluate(`location.pathname`, &receipt.StoredHTMLPath),
		chromedp.Text(`body`, &receipt.StoredHTMLText, chromedp.ByQuery),
	); err != nil {
		t.Fatalf("inspect stored HTML artifact: %v", err)
	}
	if !strings.Contains(receipt.StoredHTMLPath, "/artifacts/art_il_http_gate_html/preview") || !strings.Contains(receipt.StoredHTMLText, "IL route gate") {
		t.Fatalf("stored HTML preview mismatch: path=%q text=%q", receipt.StoredHTMLPath, receipt.StoredHTMLText)
	}
	_ = chromedp.Run(popupCtx, chromedp.Evaluate(`window.close()`, nil))

	if err := chromedp.Run(browserCtx, chromedp.Evaluate(`(() => {
	  const buttons = [...document.querySelectorAll('.report-il-card button[data-action="download-artifact"]')];
	  buttons.forEach((button, index) => setTimeout(() => button.click(), index * 80));
	  return buttons.length;
	})()`, nil)); err != nil {
		t.Fatalf("start artifact downloads: %v", err)
	}
	receipt.DownloadedFiles = waitForReportILDownloads(t, downloadDir, bundle.Artifacts)

	if err := chromedp.Run(browserCtx,
		chromedp.EmulateViewport(390, 844, chromedp.EmulateMobile, chromedp.EmulateTouch),
		chromedp.Evaluate(`document.querySelector('.report-il-lineage-details').open = true`, nil),
		chromedp.Evaluate(reportILBrowserLayoutExpression, &receipt.Mobile),
	); err != nil {
		t.Fatal(err)
	}
	assertReportILMobileLayout(t, receipt.Mobile)
	writeReportILBrowserScreenshot(t, browserCtx, "mobile.png", &receipt)
	writeReportILBrowserReceipt(t, receipt)
}

func TestReportILUnverifiedProfileBrowserGeneration(t *testing.T) {
	testReportILUnverifiedProfileBrowserGeneration(t, "draftExperimentalReport", "planned", reportilphase0.AuthoringModeStandard)
}

func TestLongFormReportILUnverifiedProfileBrowserGeneration(t *testing.T) {
	testReportILUnverifiedProfileBrowserGeneration(t, "draftLongExperimentalReport", "long_form", reportilphase0.AuthoringModeLongForm)
}

func testReportILUnverifiedProfileBrowserGeneration(t *testing.T, buttonID, reportMode, authoringMode string) {
	t.Helper()
	chromePath := testChromePath()
	if chromePath == "" {
		t.Skip("Chrome or Chromium is required for the unverified IL browser fixture")
	}

	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	observer := &reportILAcceptanceExecutor{delegate: &reportILSyntheticExecutor{service: service}}
	handler := NewServer(service, Options{AgentExecutor: observer, ReportILChromePath: chromePath}).(*Server)
	handler.staticDir = filepath.Join("static")
	server := httptest.NewServer(handler)
	defer server.Close()

	created, err := reportILAcceptancePostJSON(server.URL+"/api/missions", map[string]any{
		"title": "무검증 IL 브라우저 fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	missionID, err := acceptanceNestedString(created, "projection", "mission_id")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reportILAcceptancePostJSON(server.URL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title": "무검증 IL 브라우저 자료", "content": reportILProviderAcceptanceSource,
	}); err != nil {
		t.Fatal(err)
	}

	var browserOutput bytes.Buffer
	allocatorOptions := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath), chromedp.UserDataDir(t.TempDir()), chromedp.DisableGPU,
		chromedp.Flag("disable-background-networking", true), chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-extensions", true), chromedp.Flag("disable-sync", true),
		chromedp.Flag("no-first-run", true), chromedp.Flag("no-default-browser-check", true),
		chromedp.CombinedOutput(&browserOutput),
	)
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(ctx, allocatorOptions...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx)
	defer cancelBrowser()
	browserCtx, cancelTimeout := context.WithTimeout(browserCtx, 60*time.Second)
	defer cancelTimeout()

	if err := chromedp.Run(browserCtx,
		chromedp.EmulateViewport(1440, 1000),
		chromedp.Navigate(server.URL+"/#reports"),
		chromedp.Poll(fmt.Sprintf(`document.querySelector('#missionList button.item[data-mission-id="%s"]') !== null`, missionID), nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Evaluate(fmt.Sprintf(`document.querySelector('#missionList button.item[data-mission-id="%s"]').click()`, missionID), nil),
		chromedp.Poll(fmt.Sprintf(`window.Plasma?.state?.missionId === %q`, missionID), nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Evaluate(`document.querySelector('#reportRigor').value = 'unverified'`, nil),
		chromedp.Click(`#`+buttonID, chromedp.ByID),
		chromedp.Poll(`document.querySelector('.report-il-card') !== null && document.querySelector('.report-il-card')?.textContent.includes('Markdown') && document.querySelector('.report-il-card')?.textContent.includes('HTML') && document.querySelector('.report-il-card')?.textContent.includes('PDF')`, nil, chromedp.WithPollingTimeout(30*time.Second)),
	); err != nil {
		events, listErr := service.ListEvents(ctx, missionID)
		if listErr != nil {
			t.Fatalf("generate unverified IL report from browser: %v; list events: %v\nChrome output:\n%s", err, listErr, browserOutput.String())
		}
		types := make([]string, 0, len(events))
		payloads := make([]string, 0, len(events))
		for _, event := range events {
			types = append(types, event.EventType)
			if strings.Contains(event.EventType, "failed") {
				payloads = append(payloads, event.EventType+":"+string(event.Payload))
			}
		}
		t.Fatalf("generate unverified IL report from browser: %v\nevents=%#v\nfailures=%#v\ncalls=%#v\nChrome output:\n%s", err, types, payloads, observer.snapshot(), browserOutput.String())
	}

	receipt := reportILUnverifiedBrowserReceipt{}
	if err := chromedp.Run(browserCtx, chromedp.Evaluate(`(() => {
	  const card = document.querySelector('.report-il-card');
	  return {
	    card_count: document.querySelectorAll('.report-il-card').length,
	    card_text: card?.innerText || '',
	    download_count: card?.querySelectorAll('[data-action="download-artifact"]').length || 0,
	    intermediate_count: card?.querySelectorAll('[data-action="view-text-artifact"]').length || 0
	  };
	})()`, &receipt)); err != nil {
		t.Fatal(err)
	}
	if receipt.CardCount != 1 || receipt.DownloadCount != 6 || receipt.IntermediateCount != 3 ||
		!strings.Contains(receipt.CardText, "IL 보고서") || !strings.Contains(receipt.CardText, "출력 6개") {
		t.Fatalf("unverified IL browser card = %#v", receipt)
	}

	events, err := service.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	var terminal app.LedgerEvent
	for _, event := range events {
		if event.EventType == "report.draft.pending" {
			var payload map[string]any
			if err := json.Unmarshal(event.Payload, &payload); err == nil {
				receipt.RequestFamily, _ = payload["pipeline_family"].(string)
				receipt.RequestMode, _ = payload["report_mode"].(string)
				receipt.RequestProfile, _ = payload["rigor_level"].(string)
			}
		}
		if event.EventType == "report.artifact.created" {
			terminal = event
		}
	}
	if receipt.RequestFamily != reportilcontract.PipelineFamily || receipt.RequestMode != reportMode || receipt.RequestProfile != "unverified" {
		t.Fatalf("unverified IL browser request = %#v", receipt)
	}
	manifestArtifact, err := reportILAcceptanceArtifactByKind(ctx, service, terminal, "manifest")
	if err != nil {
		t.Fatal(err)
	}
	var manifest reportilphase0.ProductManifest
	if err := json.Unmarshal(manifestArtifact.Content, &manifest); err != nil {
		t.Fatal(err)
	}
	receipt.ManifestMode = manifest.AuthoringMode
	receipt.ManifestProfile = manifest.ValidationProfile
	calls := observer.snapshot()
	wantCalls := 1
	if authoringMode == reportilphase0.AuthoringModeLongForm {
		wantCalls = 10
	}
	if receipt.ManifestMode != authoringMode || receipt.ManifestProfile != "unverified" || len(calls) != wantCalls {
		t.Fatalf("unverified IL browser manifest/calls = %#v/%#v", receipt, calls)
	}
}

func TestLongFormReportILProgressFanoutBrowserLayout(t *testing.T) {
	chromePath := testChromePath()
	if chromePath == "" {
		t.Skip("Chrome or Chromium is required for the long-form IL progress fixture")
	}

	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	missionID := "mis_il_fanout_browser"
	if _, err := service.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "Long-form IL fan-out browser fixture"}); err != nil {
		t.Fatal(err)
	}
	seedLongFormILBrowserProgress(t, ctx, service, missionID)

	handler := NewServer(service, Options{}).(*Server)
	handler.staticDir = filepath.Join("static")
	server := httptest.NewServer(handler)
	defer server.Close()

	var browserOutput bytes.Buffer
	allocatorOptions := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath), chromedp.UserDataDir(t.TempDir()), chromedp.DisableGPU,
		chromedp.Flag("disable-background-networking", true), chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-extensions", true), chromedp.Flag("disable-sync", true),
		chromedp.Flag("no-first-run", true), chromedp.Flag("no-default-browser-check", true),
		chromedp.CombinedOutput(&browserOutput),
	)
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(ctx, allocatorOptions...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx)
	defer cancelBrowser()
	browserCtx, cancelTimeout := context.WithTimeout(browserCtx, 60*time.Second)
	defer cancelTimeout()

	if err := chromedp.Run(browserCtx,
		chromedp.EmulateViewport(1440, 1000),
		chromedp.Navigate(server.URL+"/#reports"),
		chromedp.Poll(fmt.Sprintf(`document.querySelector('#missionList button.item[data-mission-id="%s"]') !== null`, missionID), nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Evaluate(fmt.Sprintf(`document.querySelector('#missionList button.item[data-mission-id="%s"]').click()`, missionID), nil),
		chromedp.Poll(fmt.Sprintf(`window.Plasma?.state?.missionId === %q`, missionID), nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Poll(`document.querySelector('.report-il-pipeline .pipeline-details') !== null`, nil, chromedp.WithPollingTimeout(10*time.Second)),
	); err != nil {
		t.Fatalf("open long-form IL fan-out fixture: %v\nChrome output:\n%s", err, browserOutput.String())
	}

	var desktop reportILBrowserLayout
	if err := chromedp.Run(browserCtx, chromedp.Evaluate(reportILBrowserLayoutExpression, &desktop)); err != nil {
		t.Fatal(err)
	}
	writeLongFormILProgressScreenshot(t, browserCtx, "desktop.png", false)
	assertLongFormILFanoutLayout(t, desktop)
	if desktop.DocumentScrollWidth > desktop.DocumentClientWidth+1 {
		t.Fatalf("desktop long-form IL graph overflowed the document: %+v", desktop)
	}

	var mobile reportILBrowserLayout
	if err := chromedp.Run(browserCtx,
		chromedp.Evaluate(`document.querySelector('.workspace').style.gridTemplateColumns=''`, nil),
		chromedp.EmulateViewport(390, 844, chromedp.EmulateMobile, chromedp.EmulateTouch),
		chromedp.Evaluate(reportILBrowserLayoutExpression, &mobile),
	); err != nil {
		t.Fatal(err)
	}
	writeLongFormILProgressScreenshot(t, browserCtx, "mobile.png", true)
	assertLongFormILFanoutLayout(t, mobile)
	if mobile.DocumentScrollWidth > mobile.DocumentClientWidth+1 {
		t.Fatalf("mobile long-form IL graph overflowed the document: %+v", mobile)
	}
}

func seedLongFormILBrowserProgress(t *testing.T, ctx context.Context, service *app.Service, missionID string) {
	t.Helper()
	pendingID := "evt_il_fanout_pending"
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"},
		Payload: mustJSON(map[string]any{
			"title": "Long-form IL fan-out", "report_mode": "long_form",
			"pipeline_family": reportilcontract.PipelineFamily,
			"pipeline_graph":  reportpipeline.ExperimentalILValidationProfilesGraph,
			"started_at":      "2026-08-31T00:00:00Z", "rigor_level": "strict",
		}),
	}); err != nil {
		t.Fatal(err)
	}
	for _, stage := range []string{"source_packet", "il_editorial_memory", "il_narrative", "il_long_form_plan"} {
		for _, status := range []string{"started", "completed"} {
			if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
				EventID:   "evt_il_fanout_" + stage + "_" + status,
				MissionID: missionID, EventType: "report." + stage + "." + status,
				CausationEventID: pendingID, CorrelationID: pendingID,
				Producer: app.Producer{Type: "system", ID: "report-il"},
				Payload:  mustJSON(map[string]any{"pending_event_id": pendingID}),
			}); err != nil {
				t.Fatal(err)
			}
		}
	}
	plan := reportilcontract.LongFormPlan{Title: "Long-form IL fan-out"}
	for partIndex := 1; partIndex <= 2; partIndex++ {
		part := reportilcontract.LongFormPart{Title: fmt.Sprintf("Part %d", partIndex)}
		for sectionIndex := 1; sectionIndex <= 2; sectionIndex++ {
			part.Sections = append(part.Sections, reportilcontract.LongFormSection{Title: fmt.Sprintf("Section %d.%d", partIndex, sectionIndex)})
		}
		plan.Parts = append(plan.Parts, part)
	}
	if err := service.AppendReportILLongFormProgress(ctx, missionID, pendingID, "plan", "completed", &plan, 0, 0, plan.Title); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_il_fanout_sections_started", MissionID: missionID,
		EventType:        "report.il_long_form_sections.started",
		CausationEventID: pendingID, CorrelationID: pendingID,
		Producer: app.Producer{Type: "system", ID: "report-il"},
		Payload:  mustJSON(map[string]any{"pending_event_id": pendingID}),
	}); err != nil {
		t.Fatal(err)
	}
	for _, coordinate := range [][2]int{{1, 1}, {1, 2}, {2, 1}, {2, 2}} {
		if err := service.AppendReportILLongFormProgress(
			ctx, missionID, pendingID, "section", "started", nil,
			coordinate[0], coordinate[1], fmt.Sprintf("Section %d.%d", coordinate[0], coordinate[1]),
		); err != nil {
			t.Fatal(err)
		}
	}
}

func assertLongFormILFanoutLayout(t *testing.T, layout reportILBrowserLayout) {
	t.Helper()
	if layout.FanoutGraphCount != 1 || layout.PipelineGraphWidth <= 0 || layout.SectionNodeCount != 4 || layout.PartEditNodeCount != 2 || !layout.ParallelSectionLabel || !layout.SequentialPartLabel {
		t.Fatalf("long-form IL fan-out graph = %+v", layout)
	}
}

func writeLongFormILProgressScreenshot(t *testing.T, ctx context.Context, filename string, mobile bool) {
	t.Helper()
	directory := strings.TrimSpace(os.Getenv("PLASMA_IL_BROWSER_ARTIFACT_DIR"))
	if directory == "" {
		return
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if mobile {
		if err := chromedp.Run(ctx, chromedp.Evaluate(`document.querySelector('#reportPipeline').scrollIntoView({block:'start'})`, nil)); err != nil {
			t.Fatal(err)
		}
	} else if err := chromedp.Run(ctx, chromedp.Evaluate(`document.querySelector('.workspace').style.gridTemplateColumns='18px minmax(360px, 1fr)';document.querySelector('#reportPipeline').scrollIntoView({block:'start'})`, nil)); err != nil {
		t.Fatal(err)
	}
	var image []byte
	if err := chromedp.Run(ctx, chromedp.Screenshot(`#reportPipeline`, &image, chromedp.ByID)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, filename), image, 0o644); err != nil {
		t.Fatal(err)
	}
}

func seedILBrowserFailure(t *testing.T, ctx context.Context, service *app.Service, missionID string) {
	t.Helper()
	pendingID := "evt_il_browser_failed_pending"
	terminalID := "evt_il_browser_failed_terminal"
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"},
		Payload: mustJSON(map[string]any{
			"title": "Experimental IL typed failure", "report_mode": "planned",
			"pipeline_family": reportilcontract.PipelineFamily, "pipeline_graph": reportpipeline.ExperimentalILEditorialMemoryGraph,
			"started_at": "2026-08-21T00:00:00Z", "agent_model": "gpt-5.6-luna",
			"agent_reasoning_effort": "xhigh", "rigor_level": "strict",
		}),
	}); err != nil {
		t.Fatal(err)
	}
	for index, stage := range []string{"source_packet", "il_narrative"} {
		for _, status := range []string{"started", "completed"} {
			if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
				EventID:   fmt.Sprintf("evt_il_browser_%s_%s_%d", stage, status, index),
				MissionID: missionID, EventType: "report." + stage + "." + status,
				CausationEventID: pendingID, CorrelationID: pendingID,
				Producer: app.Producer{Type: "system", ID: "report-il"},
				Payload:  mustJSON(map[string]any{"pending_event_id": pendingID, "stage_kind": stage, "stage_id": stage}),
			}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_il_browser_reader_started", MissionID: missionID,
		EventType: "report.il_reader.started", CausationEventID: pendingID,
		CorrelationID: pendingID, Producer: app.Producer{Type: "system", ID: "report-il"},
		Payload: mustJSON(map[string]any{"pending_event_id": pendingID, "stage_kind": "il_reader", "stage_id": "il_reader"}),
	}); err != nil {
		t.Fatal(err)
	}
	stageFailureID := "evt_il_browser_reader_failed"
	appended, closed, err := service.AppendReportTerminalIfOpen(ctx, missionID, pendingID, []app.AppendEventRequest{
		{
			EventID: stageFailureID, MissionID: missionID, EventType: "report.il_reader.failed",
			CausationEventID: pendingID, CorrelationID: terminalID,
			Producer: app.Producer{Type: "agent", ID: "codex"},
			Payload: mustJSON(map[string]any{
				"pending_event_id": pendingID, "stage_kind": "il_reader", "stage_id": "il_reader",
				"safe_error_class": "report_stage_failed", "safe_error_message": "안전한 독자 편집 실패", "retryable": false,
				"terminal_event_id": terminalID,
			}),
		},
		{
			EventID: terminalID, MissionID: missionID, EventType: "report.draft.failed",
			CausationEventID: pendingID, CorrelationID: pendingID,
			Producer: app.Producer{Type: "agent", ID: "codex"},
			Payload: mustJSON(map[string]any{
				"kind": "report_draft_failed", "pending_event_id": pendingID,
				"failed_stage_kind": "il_reader", "failed_stage_id": "il_reader", "stage_failure_event_id": stageFailureID,
				"safe_error_class": "report_stage_failed", "safe_error_message": "안전한 독자 편집 실패",
			}),
		},
	})
	if err != nil || !closed || len(appended) != 2 {
		t.Fatalf("seed typed IL failure: closed=%v events=%d err=%v", closed, len(appended), err)
	}
}

const reportILBrowserLayoutExpression = `(() => {
  const experimental = document.querySelector('.report-il-experimental-card');
  const generate = document.querySelector('#draftExperimentalReport');
  const stages = [...document.querySelectorAll('.report-il-stage')];
  const pipelineDetails = document.querySelector('.report-il-pipeline .pipeline-details');
  if (pipelineDetails) pipelineDetails.open = true;
  const graph = document.querySelector('.report-il-pipeline .pipeline-graph');
  const lineage = document.querySelector('.report-il-lineage-details');
  if (lineage) lineage.open = true;
  const rect = (node) => node ? node.getBoundingClientRect() : {left:0,top:0,width:0};
  return {
    viewport_width: window.innerWidth,
    document_client_width: document.documentElement.clientWidth,
    document_scroll_width: document.documentElement.scrollWidth,
    experimental_card_width: rect(experimental).width,
    generate_button_width: rect(generate).width,
    stage_count: stages.length,
    stage_first_left: rect(stages[0]).left,
    stage_second_left: rect(stages[1]).left,
    stage_first_top: rect(stages[0]).top,
    stage_second_top: rect(stages[1]).top,
    lineage_count: document.querySelectorAll('.report-il-lineage-item').length,
    download_count: document.querySelectorAll('.report-il-card [data-action="download-artifact"]').length,
    view_count: document.querySelectorAll('.report-il-card [data-action^="view-"]').length,
    classic_actions: document.querySelectorAll('.report-il-card [data-action="patch-artifact"], .report-il-card [data-action="start-designed-html-artifact"], .report-il-card [data-action="start-humanized-markdown-artifact"]').length,
    retry_actions: document.querySelectorAll('.report-il-pipeline [data-report-retry]').length,
    no_retry_visible: Boolean(document.querySelector('.report-il-retry-boundary')),
    pipeline_graph_count: document.querySelectorAll('.report-il-pipeline .pipeline-graph').length,
    pipeline_graph_width: rect(graph).width,
    fanout_graph_count: document.querySelectorAll('.report-il-pipeline .pipeline-graph-il-long-form').length,
    section_node_count: document.querySelectorAll('.report-il-pipeline [data-pipeline-node-kind="section"]').length,
    part_edit_node_count: document.querySelectorAll('.report-il-pipeline [data-pipeline-node-kind="part_edit"]').length,
    parallel_section_label: [...document.querySelectorAll('.report-il-pipeline .pipeline-phase-label')].some((node) => node.textContent === '섹션 병렬 작성'),
    sequential_part_label: [...document.querySelectorAll('.report-il-pipeline .pipeline-phase-label')].some((node) => node.textContent === '파트 순차 편집'),
  };
})()`

func assertReportILDesktopLayout(t *testing.T, layout reportILBrowserLayout) {
	t.Helper()
	if layout.StageCount != 9 || layout.LineageCount != 8 || layout.DownloadCount != 8 || layout.ViewCount != 7 {
		t.Fatalf("desktop IL counts = %+v", layout)
	}
	if layout.ClassicActions != 0 || layout.RetryActions != 2 || layout.NoRetryVisible {
		t.Fatalf("desktop classic/retry boundary = %+v", layout)
	}
	if layout.PipelineGraphCount != 1 || layout.PipelineGraphWidth <= 0 {
		t.Fatalf("desktop IL progress graph = %+v", layout)
	}
	if layout.DocumentScrollWidth > layout.DocumentClientWidth+1 || layout.StageSecondLeft <= layout.StageFirstLeft || layout.StageSecondTop != layout.StageFirstTop {
		t.Fatalf("desktop layout overflow/grid mismatch = %+v", layout)
	}
}

func assertReportILMobileLayout(t *testing.T, layout reportILBrowserLayout) {
	t.Helper()
	if layout.StageCount != 9 || layout.LineageCount != 8 || layout.DownloadCount != 8 || layout.ViewCount != 7 {
		t.Fatalf("mobile IL counts = %+v", layout)
	}
	if layout.ClassicActions != 0 || layout.RetryActions != 2 || layout.NoRetryVisible {
		t.Fatalf("mobile classic/retry boundary = %+v", layout)
	}
	if layout.PipelineGraphCount != 1 || layout.PipelineGraphWidth <= 0 {
		t.Fatalf("mobile IL progress graph = %+v", layout)
	}
	if layout.DocumentScrollWidth > layout.DocumentClientWidth+1 || layout.StageSecondLeft != layout.StageFirstLeft || layout.StageSecondTop <= layout.StageFirstTop {
		t.Fatalf("mobile layout overflow/grid mismatch = %+v", layout)
	}
	if layout.GenerateButtonWidth < layout.ExperimentalCardWidth-30 {
		t.Fatalf("mobile Experimental button is not full width: %+v", layout)
	}
}

func waitForReportILDownloads(t *testing.T, directory string, artifacts []ilHTTPGateArtifact) []string {
	t.Helper()
	want := map[string][]byte{}
	for _, artifact := range artifacts {
		want[artifact.Filename] = artifact.Content
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) == len(want) {
			files := make([]string, 0, len(entries))
			matched := true
			for _, entry := range entries {
				if entry.IsDir() || strings.HasSuffix(entry.Name(), ".crdownload") {
					matched = false
					break
				}
				content, err := os.ReadFile(filepath.Join(directory, entry.Name()))
				if err != nil || !bytes.Equal(content, want[entry.Name()]) {
					matched = false
					break
				}
				files = append(files, entry.Name())
			}
			if matched {
				sort.Strings(files)
				return files
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	entries, _ := os.ReadDir(directory)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	t.Fatalf("artifact downloads did not complete: %v", names)
	return nil
}

func writeReportILBrowserScreenshot(t *testing.T, ctx context.Context, filename string, receipt *reportILBrowserReceipt) {
	t.Helper()
	directory := strings.TrimSpace(os.Getenv("PLASMA_IL_BROWSER_ARTIFACT_DIR"))
	if directory == "" {
		return
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	var image []byte
	if err := chromedp.Run(ctx, chromedp.FullScreenshot(&image, 100)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, filename), image, 0o644); err != nil {
		t.Fatal(err)
	}
	receipt.ScreenshotsWritten = true
}

func writeReportILBrowserReceipt(t *testing.T, receipt reportILBrowserReceipt) {
	t.Helper()
	directory := strings.TrimSpace(os.Getenv("PLASMA_IL_BROWSER_ARTIFACT_DIR"))
	if directory == "" {
		return
	}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(directory, "receipt.json"), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
}

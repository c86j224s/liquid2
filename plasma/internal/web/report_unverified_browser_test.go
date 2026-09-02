package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
	cdpbrowser "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

type unverifiedBrowserLayout struct {
	ViewportWidth       float64 `json:"viewport_width"`
	DocumentClientWidth float64 `json:"document_client_width"`
	DocumentScrollWidth float64 `json:"document_scroll_width"`
	ActionWidth         float64 `json:"action_width"`
	UnverifiedSelected  bool    `json:"unverified_selected"`
	ReportButtonCount   int     `json:"report_button_count"`
	ArtifactCardCount   int     `json:"artifact_card_count"`
	RetryActionCount    int     `json:"retry_action_count"`
	ClassicGraphCount   int     `json:"classic_graph_count"`
	PipelineText        string  `json:"pipeline_text"`
	ArtifactText        string  `json:"artifact_text"`
}

type unverifiedBrowserReceipt struct {
	UserAgent          string                  `json:"user_agent"`
	FailedDesktop      unverifiedBrowserLayout `json:"failed_desktop"`
	CompletedDesktop   unverifiedBrowserLayout `json:"completed_desktop"`
	CompletedMobile    unverifiedBrowserLayout `json:"completed_mobile"`
	ProviderCallCount  int                     `json:"provider_call_count"`
	PreviewTextExact   bool                    `json:"preview_text_exact"`
	DownloadBytesExact bool                    `json:"download_bytes_exact"`
	DownloadedFiles    []string                `json:"downloaded_files"`
	ScreenshotsWritten bool                    `json:"screenshots_written"`
}

func TestUnverifiedReportBrowserDogfood(t *testing.T) {
	chromePath := testChromePath()
	if chromePath == "" {
		t.Skip("Chrome or Chromium is required for the unverified report browser fixture")
	}

	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	const (
		missionID    = "mis_unverified_browser"
		failedID     = "evt_unverified_browser_failed_pending"
		providerText = "\n\n# 무검증형 브라우저 실증\n\n모델이 반환한 Markdown을 그대로 보여준다.  \n\n"
	)
	service := app.NewService(store)
	createUnverifiedMissionFixture(t, ctx, service, missionID, failedID, "브라우저 fixture 연결 자료")
	seedUnverifiedBrowserFailure(t, ctx, service, missionID, failedID)

	executor := &fakeAgentExecutor{responses: []AgentResult{{
		Text: providerText, SessionID: "ses_unverified_browser",
	}}}
	handler := NewServer(service, Options{AgentExecutor: executor}).(*Server)
	handler.staticDir = filepath.Join("static")
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	downloadDir := t.TempDir()
	var browserOutput bytes.Buffer
	allocatorOptions := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
		chromedp.UserDataDir(t.TempDir()),
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
		chromedp.Navigate(httpServer.URL+"/#reports"),
		chromedp.Poll(`document.querySelector('#missionList button.item[data-mission-id]') !== null`, nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Evaluate(`document.querySelector('#missionList button.item[data-mission-id]').click()`, nil),
		chromedp.Poll(`window.Plasma?.state?.missionId === 'mis_unverified_browser'`, nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Poll(`document.querySelector('#reportPipeline')?.textContent.includes('재시도 없음') && document.querySelector('#reportRigor option[value="unverified"]')`, nil, chromedp.WithPollingTimeout(10*time.Second)),
	); err != nil {
		t.Fatalf("open unverified browser fixture: %v\nChrome output:\n%s", err, browserOutput.String())
	}

	receipt := unverifiedBrowserReceipt{}
	if err := chromedp.Run(browserCtx,
		chromedp.Evaluate(`navigator.userAgent`, &receipt.UserAgent),
		chromedp.Evaluate(unverifiedBrowserLayoutExpression, &receipt.FailedDesktop),
	); err != nil {
		t.Fatal(err)
	}
	assertUnverifiedBrowserFailedLayout(t, receipt.FailedDesktop)
	writeUnverifiedBrowserScreenshot(t, browserCtx, "desktop-failed.png", &receipt)

	if err := chromedp.Run(browserCtx,
		chromedp.Evaluate(`document.querySelector('#reportDirectionHint').value = '자료에 맞춰 자유롭게 작성'; document.querySelector('#reportRigor').value = 'unverified'`, nil),
		chromedp.Click(`#draftQuickReport`, chromedp.ByID),
		chromedp.Poll(`document.querySelector('.report-card:not(.report-il-card)')?.textContent.includes('무검증 보고서')`, nil, chromedp.WithPollingTimeout(15*time.Second)),
		chromedp.Evaluate(unverifiedBrowserLayoutExpression, &receipt.CompletedDesktop),
	); err != nil {
		t.Fatalf("generate unverified report from browser: %v\nChrome output:\n%s", err, browserOutput.String())
	}
	assertUnverifiedBrowserCompletedLayout(t, receipt.CompletedDesktop, false)
	receipt.ProviderCallCount = len(executor.requests)
	if receipt.ProviderCallCount != 1 {
		t.Fatalf("browser provider call count = %d, want 1", receipt.ProviderCallCount)
	}
	assertUnverifiedBrowserLedger(t, ctx, service, missionID, providerText)
	writeUnverifiedBrowserScreenshot(t, browserCtx, "desktop-completed.png", &receipt)

	var previewText string
	if err := chromedp.Run(browserCtx,
		chromedp.Click(`.report-card:not(.report-il-card) [data-action="view-artifact"]`, chromedp.ByQuery),
		chromedp.Poll(`!document.querySelector('#detailModal').classList.contains('hidden') && document.querySelector('#detailBody').textContent.includes('무검증형 브라우저 실증')`, nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Evaluate(`window.Plasma.state.detailText`, &previewText),
	); err != nil {
		t.Fatalf("open unverified Markdown preview: %v", err)
	}
	receipt.PreviewTextExact = previewText == providerText
	if !receipt.PreviewTextExact {
		t.Fatalf("browser preview changed provider bytes: got %q want %q", previewText, providerText)
	}
	if err := chromedp.Run(browserCtx,
		chromedp.Click(`#closeDetail`, chromedp.ByID),
		chromedp.Evaluate(`document.querySelector('.report-card:not(.report-il-card) [data-action="download-artifact"]').click()`, nil),
	); err != nil {
		t.Fatalf("download unverified Markdown: %v", err)
	}
	receipt.DownloadedFiles, receipt.DownloadBytesExact = waitForUnverifiedBrowserDownload(t, downloadDir, []byte(providerText))

	if err := chromedp.Run(browserCtx,
		chromedp.EmulateViewport(390, 844, chromedp.EmulateMobile, chromedp.EmulateTouch),
		chromedp.Evaluate(unverifiedBrowserLayoutExpression, &receipt.CompletedMobile),
	); err != nil {
		t.Fatal(err)
	}
	assertUnverifiedBrowserCompletedLayout(t, receipt.CompletedMobile, true)
	writeUnverifiedBrowserScreenshot(t, browserCtx, "mobile-completed.png", &receipt)
	writeUnverifiedBrowserReceipt(t, receipt)
}

func seedUnverifiedBrowserFailure(t *testing.T, ctx context.Context, service *app.Service, missionID, pendingID string) {
	t.Helper()
	_, closed, err := service.AppendReportTerminalIfOpen(ctx, missionID, pendingID, []app.AppendEventRequest{{
		EventID: "evt_unverified_browser_failed_terminal", MissionID: missionID,
		EventType: "report.draft.failed", Producer: app.Producer{Type: "agent", ID: "codex"},
		Payload: mustJSON(map[string]any{
			"kind": "report_draft_failed", "pending_event_id": pendingID,
			"pipeline_family":  reportpipeline.Unverified,
			"safe_error_class": "agent_failed", "safe_error_message": "브라우저 fixture 실패",
		}),
	}})
	if err != nil || !closed {
		t.Fatalf("seed unverified browser failure: closed=%t err=%v", closed, err)
	}
}

const unverifiedBrowserLayoutExpression = `(() => {
  const actions = document.querySelector('.report-mode-actions');
  const artifact = document.querySelector('.report-card:not(.report-il-card)');
  const rect = (node) => node ? node.getBoundingClientRect() : {width: 0};
  return {
    viewport_width: window.innerWidth,
    document_client_width: document.documentElement.clientWidth,
    document_scroll_width: document.documentElement.scrollWidth,
    action_width: rect(actions).width,
    unverified_selected: document.querySelector('#reportRigor')?.value === 'unverified',
    report_button_count: document.querySelectorAll('#draftQuickReport, #draftLongReport, #draftExperimentalReport').length,
    artifact_card_count: document.querySelectorAll('.report-card:not(.report-il-card)').length,
    retry_action_count: document.querySelectorAll('#reportPipeline [data-report-retry]').length,
    classic_graph_count: document.querySelectorAll('#reportPipeline .pipeline-details, #reportPipeline .pipeline-visual').length,
    pipeline_text: document.querySelector('#reportPipeline')?.innerText || '',
    artifact_text: artifact?.innerText || ''
  };
})()`

func assertUnverifiedBrowserFailedLayout(t *testing.T, layout unverifiedBrowserLayout) {
	t.Helper()
	if layout.ReportButtonCount != 3 {
		t.Fatalf("integrated report controls are incomplete: %+v", layout)
	}
	if !strings.Contains(layout.PipelineText, "무검증 보고서 생성") ||
		!strings.Contains(layout.PipelineText, "Markdown 단일 작성") ||
		!strings.Contains(layout.PipelineText, "단일 호출") ||
		!strings.Contains(layout.PipelineText, "재시도 없음") ||
		!strings.Contains(layout.PipelineText, "교정이나 자동 재시도 없이 종료") ||
		strings.Contains(layout.PipelineText, "최종 편집·확정") ||
		strings.Contains(layout.PipelineText, "장문 보고서 실패") ||
		layout.RetryActionCount != 0 || layout.ClassicGraphCount != 0 {
		t.Fatalf("unverified failure exposed a classic or retry path: %+v", layout)
	}
	if layout.DocumentScrollWidth > layout.DocumentClientWidth+1 {
		t.Fatalf("desktop unverified layout overflowed: %+v", layout)
	}
}

func assertUnverifiedBrowserCompletedLayout(t *testing.T, layout unverifiedBrowserLayout, mobile bool) {
	t.Helper()
	if layout.ReportButtonCount != 3 || layout.ArtifactCardCount != 1 || !layout.UnverifiedSelected {
		t.Fatalf("completed integrated report controls or artifact card are incomplete: %+v", layout)
	}
	for _, text := range []string{"무검증 보고서", "최신", "무검증형", "Markdown artifact", "새 세션", "연결 자료 읽기 전용", "사용 안 함"} {
		if !strings.Contains(layout.ArtifactText, text) {
			t.Fatalf("completed unverified artifact is missing %q: %+v", text, layout)
		}
	}
	if layout.DocumentScrollWidth > layout.DocumentClientWidth+1 {
		t.Fatalf("completed unverified layout overflowed: %+v", layout)
	}
	if mobile && layout.ActionWidth < float64(layout.DocumentClientWidth)-48 {
		t.Fatalf("mobile report actions are not full width: %+v", layout)
	}
}

func assertUnverifiedBrowserLedger(t *testing.T, ctx context.Context, service *app.Service, missionID, providerText string) {
	t.Helper()
	events, err := service.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if countLedgerEvents(events, "report.artifact.created") != 1 ||
		countLedgerEvents(events, "report.run.completed") != 1 ||
		countLedgerEvents(events, "report.plan.created") != 0 ||
		countLedgerEvents(events, "report.requirements.mapped") != 0 ||
		countLedgerEvents(events, "report.final_edit.started") != 0 ||
		countLedgerEvents(events, "report.humanize.pending") != 0 {
		t.Fatalf("browser generation entered a classic report stage: %#v", events)
	}
	for _, event := range events {
		if event.EventType != "report.artifact.created" {
			continue
		}
		var payload struct {
			ArtifactID     string `json:"artifact_id"`
			PipelineFamily string `json:"pipeline_family"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		artifact, err := service.GetRawArtifact(ctx, payload.ArtifactID)
		if err != nil {
			t.Fatal(err)
		}
		if payload.PipelineFamily != reportpipeline.Unverified || string(artifact.Content) != providerText {
			t.Fatalf("browser artifact changed family or bytes: family=%q content=%q", payload.PipelineFamily, artifact.Content)
		}
		return
	}
	t.Fatal("browser generation did not store an unverified artifact")
}

func waitForUnverifiedBrowserDownload(t *testing.T, directory string, want []byte) ([]string, bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) == 1 && !entries[0].IsDir() && !strings.HasSuffix(entries[0].Name(), ".crdownload") {
			content, err := os.ReadFile(filepath.Join(directory, entries[0].Name()))
			if err == nil && bytes.Equal(content, want) {
				return []string{entries[0].Name()}, true
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	entries, _ := os.ReadDir(directory)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	t.Fatalf("unverified Markdown download did not preserve bytes: %v", names)
	return nil, false
}

func writeUnverifiedBrowserScreenshot(t *testing.T, ctx context.Context, filename string, receipt *unverifiedBrowserReceipt) {
	t.Helper()
	directory := strings.TrimSpace(os.Getenv("PLASMA_UNVERIFIED_BROWSER_ARTIFACT_DIR"))
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

func writeUnverifiedBrowserReceipt(t *testing.T, receipt unverifiedBrowserReceipt) {
	t.Helper()
	directory := strings.TrimSpace(os.Getenv("PLASMA_UNVERIFIED_BROWSER_ARTIFACT_DIR"))
	if directory == "" {
		return
	}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "receipt.json"), append(encoded, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

package web

import (
	"bytes"
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"math"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
	"github.com/chromedp/chromedp"
)

type mobileConversationLayout struct {
	ViewportHeight         float64 `json:"viewportHeight"`
	PanelClientHeight      float64 `json:"panelClientHeight"`
	PanelScrollHeight      float64 `json:"panelScrollHeight"`
	PanelBottom            float64 `json:"panelBottom"`
	ComposerBottom         float64 `json:"composerBottom"`
	ComposerVisible        bool    `json:"composerVisible"`
	ComposerInside         bool    `json:"composerInside"`
	TurnLogHeight          float64 `json:"turnLogHeight"`
	WorkflowClient         float64 `json:"workflowClient"`
	WorkflowFlexGrow       float64 `json:"workflowFlexGrow"`
	WorkflowScroll         float64 `json:"workflowScroll"`
	WorkflowScrollTop      float64 `json:"workflowScrollTop"`
	WorkflowOverflowY      string  `json:"workflowOverflowY"`
	WorkflowSummaryTop     float64 `json:"workflowSummaryTop"`
	WorkflowSummaryBottom  float64 `json:"workflowSummaryBottom"`
	WorkflowInstructionTop float64 `json:"workflowInstructionTop"`
	WorkflowBodyClient     float64 `json:"workflowBodyClient"`
	WorkflowBodyScroll     float64 `json:"workflowBodyScroll"`
	WorkflowBodyScrollTop  float64 `json:"workflowBodyScrollTop"`
	WorkflowBodyOverflowY  string  `json:"workflowBodyOverflowY"`
	AgentClient            float64 `json:"agentClient"`
	AgentScroll            float64 `json:"agentScroll"`
	AgentScrollTop         float64 `json:"agentScrollTop"`
	AgentOverflowY         string  `json:"agentOverflowY"`
	AgentSummaryTop        float64 `json:"agentSummaryTop"`
	AgentBodyClient        float64 `json:"agentBodyClient"`
	AgentBodyScroll        float64 `json:"agentBodyScroll"`
	AgentBodyScrollTop     float64 `json:"agentBodyScrollTop"`
	AgentBodyOverflowY     string  `json:"agentBodyOverflowY"`
	ResponsiveOverride     bool    `json:"responsiveOverride"`
	MissionBannerDisplayed bool    `json:"missionBannerDisplayed"`
}

func TestMobileConversationKeepsComposerVisibleWhenWorkflowExpands(t *testing.T) {
	chromePath := testChromePath()
	if chromePath == "" {
		t.Skip("Chrome or Chromium is required for the mobile layout browser fixture")
	}

	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	if _, err := service.CreateMission(ctx, mission.CreateRequest{
		MissionID: "mis_mobile_layout",
		Title:     "Mobile layout",
	}); err != nil {
		t.Fatal(err)
	}
	handler := NewServer(service, Options{}).(*Server)
	handler.staticDir = filepath.Join("static")
	server := httptest.NewServer(handler)
	defer server.Close()

	profileDir := t.TempDir()
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
	browserCtx, cancelTimeout := context.WithTimeout(browserCtx, 45*time.Second)
	defer cancelTimeout()

	if err := chromedp.Run(browserCtx,
		chromedp.Navigate(server.URL),
		chromedp.Poll(`document.querySelector('#missionList button.item[data-mission-id]') !== null`, nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Evaluate(`document.querySelector('#missionList button.item[data-mission-id]').click()`, nil),
		chromedp.Poll(`window.Plasma?.state?.missionId === 'mis_mobile_layout'`, nil, chromedp.WithPollingTimeout(10*time.Second)),
	); err != nil {
		t.Fatalf("open mobile layout fixture: %v\nChrome output:\n%s", err, browserOutput.String())
	}

	for _, viewport := range []struct {
		name          string
		width, height int64
	}{
		{name: "390x844", width: 390, height: 844},
		{name: "375x900", width: 375, height: 900},
		{name: "320x568", width: 320, height: 568},
	} {
		viewport := viewport
		t.Run(viewport.name, func(t *testing.T) {
			if err := chromedp.Run(browserCtx,
				chromedp.EmulateViewport(viewport.width, viewport.height, chromedp.EmulateMobile, chromedp.EmulateTouch),
				chromedp.Evaluate(`(() => {
				  const agent = document.querySelector('#agentControlsDetails');
				  const workflow = document.querySelector('#workflowControlDetails');
				  agent.open = false;
				  workflow.open = false;
				  agent.scrollTop = 0;
				  agent.querySelector(':scope > .agent-controls').scrollTop = 0;
				  workflow.scrollTop = 0;
				  workflow.querySelector(':scope > .workflow-control-body').scrollTop = 0;
				  document.querySelector('.conversation-panel').scrollTop = 0;
				})()`, nil),
			); err != nil {
				t.Fatal(err)
			}

			closed := measureMobileConversationLayout(t, browserCtx)
			assertMobileComposerContract(t, "closed", closed)
			if !closed.MissionBannerDisplayed {
				t.Fatalf("closed workflow must preserve the mobile mission banner: %+v", closed)
			}

			if err := chromedp.Run(browserCtx, chromedp.Evaluate(`document.querySelector('#workflowControlDetails').open = true`, nil)); err != nil {
				t.Fatal(err)
			}
			resetMobileSettingsScroll(t, browserCtx)
			workflow := measureMobileConversationLayout(t, browserCtx)
			assertMobileComposerContract(t, "workflow open", workflow)
			assertWorkflowGetsMoreMobileSpace(t, "workflow open", workflow, workflow.TurnLogHeight)
			assertWorkflowBodyScrollOwner(t, "workflow open", workflow)
			if workflow.WorkflowSummaryTop < 0 || workflow.WorkflowSummaryBottom > workflow.ViewportHeight+1 {
				t.Fatalf("workflow summary must stay visible when the mobile section opens: %+v", workflow)
			}
			if workflow.WorkflowInstructionTop <= workflow.WorkflowSummaryBottom {
				t.Fatalf("workflow scroll region must expose content below its summary: %+v", workflow)
			}
			if viewport.height <= 680 && workflow.MissionBannerDisplayed {
				t.Fatalf("short mobile screen must release the mission banner row while workflow is open: %+v", workflow)
			}
			scrollMobileSettingsToEnd(t, browserCtx)
			workflowScrolled := measureMobileConversationLayout(t, browserCtx)
			assertWorkflowSummaryFixed(t, workflow.WorkflowSummaryTop, workflowScrolled)
			resetMobileSettingsScroll(t, browserCtx)

			if err := chromedp.Run(browserCtx, chromedp.Evaluate(`document.querySelector('#agentControlsDetails').open = true`, nil)); err != nil {
				t.Fatal(err)
			}
			both := measureMobileConversationLayout(t, browserCtx)
			assertMobileComposerContract(t, "agent and workflow open", both)
			assertWorkflowGetsMoreMobileSpace(t, "agent and workflow open", both, both.AgentClient)
			assertWorkflowBodyScrollOwner(t, "agent and workflow open", both)
			assertAgentBodyScrollOwner(t, both)
			if both.WorkflowInstructionTop <= both.WorkflowSummaryBottom {
				t.Fatalf("workflow content must remain exposed when both mobile settings sections are open: %+v", both)
			}
			if viewport.height <= 680 && both.MissionBannerDisplayed {
				t.Fatalf("short mobile screen must keep the mission banner row released while both sections are open: %+v", both)
			}
			scrollMobileSettingsToEnd(t, browserCtx)
			bothScrolled := measureMobileConversationLayout(t, browserCtx)
			assertWorkflowSummaryFixed(t, both.WorkflowSummaryTop, bothScrolled)
			if bothScrolled.AgentScrollTop > 1 || bothScrolled.AgentBodyScrollTop <= 1 || math.Abs(bothScrolled.AgentSummaryTop-both.AgentSummaryTop) > 1 {
				t.Fatalf("agent summary must stay fixed while only its mobile body scrolls: before=%+v after=%+v", both, bothScrolled)
			}
		})
	}
}

func measureMobileConversationLayout(t *testing.T, ctx context.Context) mobileConversationLayout {
	t.Helper()
	var layout mobileConversationLayout
	const expression = `(() => {
	  const panel = document.querySelector('.conversation-panel');
	  const composer = document.querySelector('.composer');
	  const turnLog = document.querySelector('.turn-log');
	  const workflow = document.querySelector('#workflowControlDetails');
	  const workflowSummary = workflow.querySelector(':scope > summary');
	  const workflowBody = workflow.querySelector(':scope > .workflow-control-body');
	  const workflowInstruction = document.querySelector('#workflowInstruction');
	  const agent = document.querySelector('#agentControlsDetails');
	  const agentSummary = agent.querySelector(':scope > summary');
	  const agentBody = agent.querySelector(':scope > .agent-controls');
	  const missionBanner = document.querySelector('.mission-banner-shell');
	  const panelRect = panel.getBoundingClientRect();
	  const composerRect = composer.getBoundingClientRect();
	  const workflowSummaryRect = workflowSummary.getBoundingClientRect();
	  const workflowInstructionRect = workflowInstruction.getBoundingClientRect();
	  return {
	    viewportHeight: document.documentElement.clientHeight,
	    panelClientHeight: panel.clientHeight,
	    panelScrollHeight: panel.scrollHeight,
	    panelBottom: panelRect.bottom,
	    composerBottom: composerRect.bottom,
	    composerVisible: composerRect.bottom <= document.documentElement.clientHeight + 1,
	    composerInside: composerRect.bottom <= panelRect.bottom + 1,
	    turnLogHeight: turnLog.getBoundingClientRect().height,
	    workflowClient: workflow.clientHeight,
	    workflowFlexGrow: Number.parseFloat(getComputedStyle(workflow).flexGrow),
	    workflowScroll: workflow.scrollHeight,
	    workflowScrollTop: workflow.scrollTop,
	    workflowOverflowY: getComputedStyle(workflow).overflowY,
	    workflowSummaryTop: workflowSummaryRect.top,
	    workflowSummaryBottom: workflowSummaryRect.bottom,
	    workflowInstructionTop: workflowInstructionRect.top,
	    workflowBodyClient: workflowBody.clientHeight,
	    workflowBodyScroll: workflowBody.scrollHeight,
	    workflowBodyScrollTop: workflowBody.scrollTop,
	    workflowBodyOverflowY: getComputedStyle(workflowBody).overflowY,
	    agentClient: agent.clientHeight,
	    agentScroll: agent.scrollHeight,
	    agentScrollTop: agent.scrollTop,
	    agentOverflowY: getComputedStyle(agent).overflowY,
	    agentSummaryTop: agentSummary.getBoundingClientRect().top,
	    agentBodyClient: agentBody.clientHeight,
	    agentBodyScroll: agentBody.scrollHeight,
	    agentBodyScrollTop: agentBody.scrollTop,
	    agentBodyOverflowY: getComputedStyle(agentBody).overflowY,
	    responsiveOverride: matchMedia('(max-width: 760px)').matches,
	    missionBannerDisplayed: getComputedStyle(missionBanner).display !== 'none',
	  };
	})()`
	if err := chromedp.Run(ctx, chromedp.Evaluate(expression, &layout)); err != nil {
		t.Fatal(err)
	}
	return layout
}

func assertMobileComposerContract(t *testing.T, state string, layout mobileConversationLayout) {
	t.Helper()
	if !layout.ResponsiveOverride {
		t.Fatalf("%s did not activate the mobile layout: %+v", state, layout)
	}
	if !layout.ComposerVisible || !layout.ComposerInside {
		t.Fatalf("%s moved the composer outside the mobile viewport or conversation panel: %+v", state, layout)
	}
	if layout.PanelScrollHeight > layout.PanelClientHeight+1 {
		t.Fatalf("%s introduced outer conversation scrolling: %+v", state, layout)
	}
	if layout.TurnLogHeight < 39 {
		t.Fatalf("%s collapsed the turn log below its mobile reading floor: %+v", state, layout)
	}
}

func assertWorkflowGetsMoreMobileSpace(t *testing.T, state string, layout mobileConversationLayout, comparisonHeight float64) {
	t.Helper()
	if math.Abs(layout.WorkflowFlexGrow-1.25) > 0.01 {
		t.Fatalf("%s must apply the expanded mobile workflow share: %+v", state, layout)
	}
	if layout.WorkflowClient < comparisonHeight+8 {
		t.Fatalf("%s must make the workflow visibly taller than its peer: comparison=%v layout=%+v", state, comparisonHeight, layout)
	}
}

func assertWorkflowBodyScrollOwner(t *testing.T, state string, layout mobileConversationLayout) {
	t.Helper()
	if layout.WorkflowScroll > layout.WorkflowClient+1 || layout.WorkflowScrollTop > 1 || layout.WorkflowOverflowY != "hidden" {
		t.Fatalf("%s must keep the workflow summary outside the scroll owner: %+v", state, layout)
	}
	if layout.WorkflowBodyScroll <= layout.WorkflowBodyClient+1 || layout.WorkflowBodyOverflowY != "auto" {
		t.Fatalf("%s must make only the workflow body scrollable: %+v", state, layout)
	}
}

func assertAgentBodyScrollOwner(t *testing.T, layout mobileConversationLayout) {
	t.Helper()
	if layout.AgentScroll > layout.AgentClient+1 || layout.AgentScrollTop > 1 || layout.AgentOverflowY != "hidden" {
		t.Fatalf("agent and workflow open must keep the agent summary outside the scroll owner: %+v", layout)
	}
	if layout.AgentBodyScroll <= layout.AgentBodyClient+1 || layout.AgentBodyOverflowY != "auto" {
		t.Fatalf("agent and workflow open must make only the agent controls body scrollable: %+v", layout)
	}
}

func scrollMobileSettingsToEnd(t *testing.T, ctx context.Context) {
	t.Helper()
	const expression = `(() => {
	  const workflow = document.querySelector('#workflowControlDetails');
	  const workflowBody = workflow.querySelector(':scope > .workflow-control-body');
	  const agent = document.querySelector('#agentControlsDetails');
	  const agentBody = agent.querySelector(':scope > .agent-controls');
	  workflow.scrollTop = workflow.scrollHeight;
	  workflowBody.scrollTop = workflowBody.scrollHeight;
	  agent.scrollTop = agent.scrollHeight;
	  agentBody.scrollTop = agentBody.scrollHeight;
	})()`
	if err := chromedp.Run(ctx, chromedp.Evaluate(expression, nil)); err != nil {
		t.Fatal(err)
	}
}

func resetMobileSettingsScroll(t *testing.T, ctx context.Context) {
	t.Helper()
	const expression = `(() => {
	  document.querySelector('#workflowControlDetails').scrollTop = 0;
	  document.querySelector('#workflowControlDetails > .workflow-control-body').scrollTop = 0;
	  document.querySelector('#agentControlsDetails').scrollTop = 0;
	  document.querySelector('#agentControlsDetails > .agent-controls').scrollTop = 0;
	})()`
	if err := chromedp.Run(ctx, chromedp.Evaluate(expression, nil)); err != nil {
		t.Fatal(err)
	}
}

func assertWorkflowSummaryFixed(t *testing.T, beforeTop float64, after mobileConversationLayout) {
	t.Helper()
	if after.WorkflowScrollTop > 1 || after.WorkflowBodyScrollTop <= 1 || math.Abs(after.WorkflowSummaryTop-beforeTop) > 1 {
		t.Fatalf("workflow summary must stay fixed while only its mobile body scrolls: before_top=%v after=%+v", beforeTop, after)
	}
}

func testChromePath() string {
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	if runtime.GOOS == "darwin" {
		const path = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
	}
	return ""
}

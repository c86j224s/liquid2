package browserrender

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const defaultUserAgent = "Mozilla/5.0 (compatible; PlasmaBrowserRenderer/0.1)"

// Renderer는 headless browser 렌더링 실행자다. 인스턴스는 동시 실행 semaphore를
// 내부에서 지연 초기화하므로 여러 요청이 공유할 수 있지만, 각 Render 호출은 별도
// 임시 profile을 사용한다.
type Renderer struct {
	maxConcurrent int
	timeout       time.Duration
	settleDelay   time.Duration
	resolveTime   time.Duration
	userAgent     string

	once sync.Once
	sem  chan struct{}
}

// NewRenderer는 0 값 옵션을 기본값으로 보정해 Renderer를 만든다.
func NewRenderer(options Options) *Renderer {
	maxConcurrent := options.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = DefaultMaxConcurrent
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	settleDelay := options.SettleDelay
	if settleDelay <= 0 {
		settleDelay = DefaultSettleDelay
	}
	resolveTime := options.ResolveTime
	if resolveTime <= 0 {
		resolveTime = DefaultResolveTime
	}
	userAgent := strings.TrimSpace(options.UserAgent)
	if userAgent == "" {
		userAgent = defaultUserAgent
	}
	return &Renderer{
		maxConcurrent: maxConcurrent,
		timeout:       timeout,
		settleDelay:   settleDelay,
		resolveTime:   resolveTime,
		userAgent:     userAgent,
	}
}

// Render는 URL을 browser로 열고, 읽을 수 있는 HTML 문서만 Result로 반환한다.
// 렌더링 전 대상 URL과 렌더링 중 subresource URL을 모두 검사하며, 차단된 요청이
// 하나라도 있으면 결과를 저장하지 않는다.
func (renderer *Renderer) Render(ctx context.Context, rawURL string) (Result, error) {
	if err := renderer.allowURL(ctx, rawURL); err != nil {
		return Result{}, err
	}
	if err := renderer.acquire(ctx); err != nil {
		return Result{}, err
	}
	defer renderer.release()

	runCtx, cancel := context.WithTimeout(ctx, renderer.timeout)
	defer cancel()

	profileDir, err := os.MkdirTemp("", "plasma-browser-render-*")
	if err != nil {
		return Result{}, ErrRenderUnavailable
	}
	defer os.RemoveAll(profileDir)

	var stderr bytes.Buffer
	allocatorOptions := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(profileDir),
		chromedp.UserAgent(renderer.userAgent),
		chromedp.DisableGPU,
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.CombinedOutput(&stderr),
	)
	allocatorCtx, allocatorCancel := chromedp.NewExecAllocator(runCtx, allocatorOptions...)
	defer allocatorCancel()
	browserCtx, browserCancel := chromedp.NewContext(allocatorCtx)
	defer browserCancel()

	var blockedMu sync.Mutex
	blockedRequests := 0
	chromedp.ListenTarget(browserCtx, func(event any) {
		paused, ok := event.(*fetch.EventRequestPaused)
		if !ok {
			return
		}
		go func() {
			if err := renderer.allowURL(runCtx, paused.Request.URL); err != nil {
				blockedMu.Lock()
				blockedRequests++
				blockedMu.Unlock()
				_ = chromedp.Run(browserCtx, fetch.FailRequest(paused.RequestID, network.ErrorReasonBlockedByClient))
				return
			}
			_ = chromedp.Run(browserCtx, fetch.ContinueRequest(paused.RequestID))
		}()
	})

	var dom string
	var finalURL string
	err = chromedp.Run(browserCtx,
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers(map[string]any{
			"Accept-Language": "ko,en-US;q=0.9,en;q=0.8",
		})),
		fetch.Enable().WithPatterns([]*fetch.RequestPattern{{
			URLPattern:   "*",
			RequestStage: fetch.RequestStageRequest,
		}}),
		chromedp.Navigate(rawURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(renderer.settleDelay),
		chromedp.Location(&finalURL),
		chromedp.OuterHTML("html", &dom),
	)
	blockedMu.Lock()
	blocked := blockedRequests
	blockedMu.Unlock()
	if blocked > 0 {
		return Result{}, fmt.Errorf("%w: blocked subresource", ErrBlockedURL)
	}
	if err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return Result{}, ErrRenderTimedOut
		}
		return Result{}, ErrRenderFailed
	}
	if err := renderer.allowURL(ctx, finalURL); err != nil {
		return Result{}, err
	}
	return readableDocument(dom, finalURL, time.Now().UTC())
}

// acquire는 renderer 인스턴스 단위의 동시 실행 상한을 적용한다.
func (renderer *Renderer) acquire(ctx context.Context) error {
	renderer.once.Do(func() {
		renderer.sem = make(chan struct{}, renderer.maxConcurrent)
	})
	select {
	case renderer.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return ErrRenderTimedOut
		}
		return ctx.Err()
	}
}

// release는 acquire로 점유한 실행 슬롯을 반환한다.
func (renderer *Renderer) release() {
	<-renderer.sem
}

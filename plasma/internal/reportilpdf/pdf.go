package reportilpdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/reportilphase0"
	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// Chrome은 report IL HTML을 Chrome print backend로 변환하는 구체 어댑터다.
type Chrome struct {
	ChromePath string
}

// RenderPDF는 45초 제한, 임시 프로필과 0600 입력 파일, readiness 확인을 포함한
// 기존 Chrome PDF 실행 계약을 그대로 수행한다.
func (renderer Chrome) RenderPDF(ctx context.Context, htmlDocument []byte) (reportilphase0.PDFResult, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(htmlDocument), []byte("<!doctype html>")) {
		return reportilphase0.PDFResult{}, fmt.Errorf("PDF source is not a complete HTML document")
	}
	runCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	profileDir, err := os.MkdirTemp("", "plasma-report-il-pdf-*")
	if err != nil {
		return reportilphase0.PDFResult{}, err
	}
	defer os.RemoveAll(profileDir)
	htmlPath := profileDir + string(os.PathSeparator) + "report.html"
	if err := os.WriteFile(htmlPath, htmlDocument, 0o600); err != nil {
		return reportilphase0.PDFResult{}, fmt.Errorf("write Chrome PDF source: %w", err)
	}

	chromePath := strings.TrimSpace(renderer.ChromePath)
	if chromePath == "" {
		return reportilphase0.PDFResult{}, fmt.Errorf("Chrome executable path is required")
	}
	options := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(profileDir),
		chromedp.DisableGPU,
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.ExecPath(chromePath),
	)
	allocatorCtx, allocatorCancel := chromedp.NewExecAllocator(runCtx, options...)
	defer allocatorCancel()
	browserCtx, browserCancel := chromedp.NewContext(allocatorCtx)
	defer browserCancel()

	var result reportilphase0.PDFResult
	err = chromedp.Run(browserCtx,
		chromedp.Navigate("file://"+htmlPath),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Poll(`document.body?.dataset.mathRendered !== "false" && document.body?.dataset.mermaidRendered !== "false"`, nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var mathStatus string
			var mathError string
			if err := chromedp.Evaluate(`document.body?.dataset.mathRendered || ""`, &mathStatus).Do(ctx); err != nil {
				return err
			}
			if mathStatus != "true" {
				_ = chromedp.Evaluate(`document.body?.dataset.mathError || "unknown equation render error"`, &mathError).Do(ctx)
				return fmt.Errorf("equation render failed: %s", mathError)
			}
			var mermaidStatus string
			var mermaidError string
			if err := chromedp.Evaluate(`document.body?.dataset.mermaidRendered || ""`, &mermaidStatus).Do(ctx); err != nil {
				return err
			}
			if mermaidStatus != "true" {
				_ = chromedp.Evaluate(`document.body?.dataset.mermaidError || "unknown Mermaid render error"`, &mermaidError).Do(ctx)
				return fmt.Errorf("Mermaid render failed: %s", mermaidError)
			}
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, product, revision, _, _, err := browser.GetVersion().Do(ctx)
			if err != nil {
				return err
			}
			result.RendererProduct = product
			result.RendererRevision = revision
			return nil
		}),
		// Chrome tags role=img SVGs as atomic figures, hiding their text from
		// PDFKit extraction. Preserve diagram labels as children of a named
		// group in this isolated print document; browser HTML stays unchanged.
		chromedp.Evaluate(`document.querySelectorAll(".plasma-mermaid-diagram svg[role='img']").forEach(svg => svg.setAttribute("role", "group"))`, nil),
		chromedp.ActionFunc(func(ctx context.Context) error {
			content, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				WithGenerateTaggedPDF(true).
				WithGenerateDocumentOutline(true).
				Do(ctx)
			if err != nil {
				return err
			}
			result.Content = content
			return nil
		}),
	)
	if err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return reportilphase0.PDFResult{}, fmt.Errorf("Chrome PDF render timed out: %w", err)
		}
		return reportilphase0.PDFResult{}, fmt.Errorf("Chrome PDF render failed: %w", err)
	}
	if !bytes.HasPrefix(result.Content, []byte("%PDF-")) {
		return reportilphase0.PDFResult{}, fmt.Errorf("Chrome returned an invalid PDF artifact")
	}
	return result, nil
}

var _ reportilphase0.PDFRenderer = Chrome{}

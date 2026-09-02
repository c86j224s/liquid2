package reportilphase0

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// PDFResult is a Chrome-backed Phase 0 PDF and the pinned renderer identity.
type PDFResult struct {
	Content          []byte
	RendererProduct  string
	RendererRevision string
}

// RenderPDF prints fully self-contained HTML with headless Chrome. It does not
// fetch report subresources and returns no artifact when the renderer fails.
func RenderPDF(ctx context.Context, htmlDocument []byte, chromePath string) (PDFResult, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(htmlDocument), []byte("<!doctype html>")) {
		return PDFResult{}, fmt.Errorf("PDF source is not a complete HTML document")
	}
	runCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	profileDir, err := os.MkdirTemp("", "plasma-report-il-pdf-*")
	if err != nil {
		return PDFResult{}, err
	}
	defer os.RemoveAll(profileDir)
	htmlPath := profileDir + string(os.PathSeparator) + "report.html"
	if err := os.WriteFile(htmlPath, htmlDocument, 0o600); err != nil {
		return PDFResult{}, fmt.Errorf("write Chrome PDF source: %w", err)
	}

	if strings.TrimSpace(chromePath) == "" {
		return PDFResult{}, fmt.Errorf("Chrome executable path is required")
	}
	options := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(profileDir),
		chromedp.DisableGPU,
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("hide-scrollbars", true),
	)
	if strings.TrimSpace(chromePath) != "" {
		options = append(options, chromedp.ExecPath(strings.TrimSpace(chromePath)))
	}
	allocatorCtx, allocatorCancel := chromedp.NewExecAllocator(runCtx, options...)
	defer allocatorCancel()
	browserCtx, browserCancel := chromedp.NewContext(allocatorCtx)
	defer browserCancel()

	var result PDFResult
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
			return PDFResult{}, fmt.Errorf("Chrome PDF render timed out: %w", err)
		}
		return PDFResult{}, fmt.Errorf("Chrome PDF render failed: %w", err)
	}
	if !bytes.HasPrefix(result.Content, []byte("%PDF-")) {
		return PDFResult{}, fmt.Errorf("Chrome returned an invalid PDF artifact")
	}
	return result, nil
}

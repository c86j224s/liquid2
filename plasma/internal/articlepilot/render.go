package articlepilot

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportilphase0"
	"github.com/yuin/goldmark"
)

type Rendered struct {
	Markdown         []byte
	HTML             []byte
	PDF              []byte
	RendererProduct  string
	RendererRevision string
}

// Render deterministically turns accepted Markdown into a neutral reading shell
// and delegates PDF printing to the existing Report IL Chrome adapter.
func Render(ctx context.Context, markdown []byte, renderer reportilphase0.PDFRenderer) (Rendered, error) {
	if err := validateAcceptedMarkdown(markdown); err != nil {
		return Rendered{}, err
	}
	var body bytes.Buffer
	if err := goldmark.Convert(markdown, &body); err != nil {
		return Rendered{}, err
	}
	htmlDocument := []byte("<!doctype html>\n<html lang=\"ko\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><meta http-equiv=\"Content-Security-Policy\" content=\"default-src 'none'; style-src 'unsafe-inline'\"><style>body{margin:0;background:#faf9f6;color:#20201e;font-family:ui-serif,Georgia,serif;line-height:1.78}main{max-width:46rem;margin:0 auto;padding:clamp(2rem,6vw,5rem) 1.25rem}h1,h2,h3{line-height:1.2}p,li{font-size:1.08rem}code,pre{font-family:ui-monospace,monospace}pre{overflow:auto;padding:1rem;background:#f0eee8}@media(prefers-color-scheme:dark){body{background:#171716;color:#eceae4}pre{background:#242421}}</style><title>Article reading packet</title></head><body data-math-rendered=\"true\" data-mermaid-rendered=\"true\"><main>" + body.String() + "</main></body></html>\n")
	if renderer == nil {
		return Rendered{}, fmt.Errorf("PDF renderer is not configured")
	}
	pdf, err := renderer.RenderPDF(ctx, htmlDocument)
	if err != nil {
		return Rendered{}, err
	}
	return Rendered{Markdown: append([]byte(nil), markdown...), HTML: htmlDocument, PDF: pdf.Content, RendererProduct: pdf.RendererProduct, RendererRevision: pdf.RendererRevision}, nil
}

func validateAcceptedMarkdown(markdown []byte) error {
	trimmed := strings.TrimSpace(string(markdown))
	if trimmed == "" || !strings.HasPrefix(trimmed, "# ") {
		return fmt.Errorf("accepted Article Markdown requires one title")
	}
	lower := strings.ToLower(trimmed)
	for _, forbidden := range []string{"<script", "<iframe", "javascript:", "data:text/html"} {
		if strings.Contains(lower, forbidden) {
			return fmt.Errorf("accepted Article Markdown contains active content")
		}
	}
	return nil
}

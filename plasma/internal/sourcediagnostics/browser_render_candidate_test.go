package sourcediagnostics

import (
	"slices"
	"strings"
	"testing"
)

func TestDiagnoseBrowserRenderCandidateFlagsThinJSShell(t *testing.T) {
	diagnosis := DiagnoseBrowserRenderCandidate([]byte(browserRenderCandidateHTMLFixture()), "text/html; charset=utf-8")
	if !diagnosis.Candidate {
		t.Fatalf("expected browser render candidate, got %#v", diagnosis)
	}
	for _, signal := range []string{"short_visible_text", "script_heavy_html", "app_mount_node", "thin_visible_text_vs_html"} {
		if !slices.Contains(diagnosis.Signals, signal) {
			t.Fatalf("expected signal %q in %#v", signal, diagnosis.Signals)
		}
	}
	if diagnosis.VisibleTextLength <= 0 || diagnosis.VisibleTextLength > browserRenderCandidateMaxVisibleText {
		t.Fatalf("unexpected visible text length: %#v", diagnosis)
	}
}

func TestDiagnoseBrowserRenderCandidateIgnoresReadableStaticHTML(t *testing.T) {
	body := strings.Repeat("<p>Readable source paragraph with enough detail for the current fetch path.</p>", 80)
	diagnosis := DiagnoseBrowserRenderCandidate([]byte("<html><body><article>"+body+"</article></body></html>"), "text/html; charset=utf-8")
	if diagnosis.Candidate {
		t.Fatalf("readable static HTML must not be marked as browser render candidate: %#v", diagnosis)
	}
}

func TestDiagnoseBrowserRenderCandidateIgnoresBotVerificationWall(t *testing.T) {
	body := `<html><body><div id="app"></div><p>Checking your browser before accessing example.com.</p>` +
		`<script>window._cf_chl_opt={};</script>` + strings.Repeat(`<script src="/cdn-cgi/challenge-platform/challenge.js"></script>`, 5) +
		`<p>Cloudflare Turnstile CAPTCHA verification.</p></body></html>`
	diagnosis := DiagnoseBrowserRenderCandidate([]byte(body), "text/html; charset=utf-8")
	if diagnosis.Candidate {
		t.Fatalf("bot verification HTML must not be marked as browser render candidate: %#v", diagnosis)
	}
}

func browserRenderCandidateHTMLFixture() string {
	return `<html><head><title>Client App</title>` +
		strings.Repeat(`<script src="/assets/chunk.js"></script>`, 5) +
		`<script>window.__INITIAL_STATE__={};` + strings.Repeat("bundle();", 300) + `</script>` +
		`</head><body><div id="root" data-reactroot=""></div><noscript>Please enable JavaScript to view this page.</noscript></body></html>`
}

package reportilphase0_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/c86j224s/liquid2/plasma/internal/reportilpdf"
	phase0 "github.com/c86j224s/liquid2/plasma/internal/reportilphase0"
)

func installedChromePath(t *testing.T) string {
	t.Helper()
	chromePath := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	chromeInfo, err := os.Stat(chromePath)
	if err != nil || chromeInfo.IsDir() || chromeInfo.Mode()&0o111 == 0 {
		t.Skip("installed macOS Google Chrome is not executable")
	}
	return chromePath
}

func pdfKitPageTexts(t *testing.T, content []byte) []string {
	t.Helper()
	if _, err := exec.LookPath("swift"); err != nil {
		t.Skip("Swift is required for the macOS PDFKit pagination check")
	}
	directory := t.TempDir()
	pdfPath := filepath.Join(directory, "report.pdf")
	scriptPath := filepath.Join(directory, "page-text.swift")
	if err := os.WriteFile(pdfPath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	script := `import Foundation
import PDFKit
let document = PDFDocument(url: URL(fileURLWithPath: CommandLine.arguments[1]))!
let pages = (0..<document.pageCount).map { document.page(at: $0)?.string ?? "" }
let data = try! JSONSerialization.data(withJSONObject: pages)
FileHandle.standardOutput.write(data)
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("swift", scriptPath, pdfPath).CombinedOutput()
	if err != nil {
		t.Fatalf("PDFKit page text extraction failed: %v\n%s", err, output)
	}
	var texts []string
	if err := json.Unmarshal(output, &texts); err != nil {
		t.Fatalf("decode PDFKit page text: %v\n%s", err, output)
	}
	return texts
}

func TestReportFirstAuthoredContentSurvivesPDFProjection(t *testing.T) {
	chromePath := installedChromePath(t)
	document, authoredValues := phase0.CompileReportFirstAuthoredPDFFixture(t)
	html, _, err := phase0.RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	result, err := (reportilpdf.Chrome{ChromePath: chromePath}).RenderPDF(context.Background(), html)
	if err != nil {
		t.Fatal(err)
	}
	pdfText := strings.Join(pdfKitPageTexts(t, result.Content), "\n")
	compactPDFText := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, pdfText)
	for _, authoredValue := range authoredValues {
		compactAuthoredValue := strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, authoredValue)
		if !strings.Contains(compactPDFText, compactAuthoredValue) {
			t.Fatalf("PDF projection lost authored value %q in %q", authoredValue, pdfText)
		}
	}
}

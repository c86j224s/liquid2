package reportdocument

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExportPreservesEscapingAndFootnotes(t *testing.T) {
	blocks := []ReportBlock{{BlockType: "paragraph", Content: json.RawMessage(`{"text":"<tag> & text"}`), SourceRefs: ReportBlockSourceRefs{ClaimIDs: []string{"clm_one", "clm_one"}}}}
	md, _, _, err := RenderReportExport(ReportVersion{ReportVersionID: "rvn_one"}, blocks, ReportExportTargetMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(md), "<tag> & text [^1]") || strings.Count(string(md), "[^1]:") != 1 {
		t.Fatalf("unexpected markdown: %s", md)
	}
	html, _, _, err := RenderReportExport(ReportVersion{ReportVersionID: "rvn_one"}, blocks, ReportExportTargetHTML)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "&lt;tag&gt; &amp; text") || strings.Contains(string(html), "<tag>") {
		t.Fatalf("unexpected HTML: %s", html)
	}
}

func TestDraftInputRejectsDocumentBlock(t *testing.T) {
	_, err := BuildReportBlocksFromDraftInputs("rvn_one", "mis_one", ReportBlockAuthorship{}.Producer, []ReportBlockDraftInput{{BlockType: "document"}})
	if err == nil {
		t.Fatal("expected document block rejection")
	}
}

package reportilphase0

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestValidateBundleSeparatesT0T1T2Contracts(t *testing.T) {
	for _, arm := range []string{"T0", "T1", "T2"} {
		t.Run(arm, func(t *testing.T) {
			bundle := sampleBundle(t, arm)
			if err := ValidateBundle(bundle); err != nil {
				t.Fatalf("ValidateBundle(%s): %v", arm, err)
			}
		})
	}

	t.Run("T0 rejects narrative binding", func(t *testing.T) {
		bundle := sampleBundle(t, "T1")
		bundle.Arm = "T0"
		if err := ValidateBundle(bundle); err == nil || !strings.Contains(err.Error(), "T0 must not bind") {
			t.Fatalf("error = %v, want T0 binding rejection", err)
		}
	})

	t.Run("T2 requires whole manuscript attestation", func(t *testing.T) {
		bundle := sampleBundle(t, "T1")
		bundle.Arm = "T2"
		if err := ValidateBundle(bundle); err == nil || !strings.Contains(err.Error(), "T2 requires") {
			t.Fatalf("error = %v, want T2 attestation rejection", err)
		}
	})
}

func TestValidateBundleRejectsBrokenIdentityAssetAndFlowBindings(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Bundle)
		want string
	}{
		{name: "duplicate node", edit: func(bundle *Bundle) {
			bundle.Document.Blocks[1].NodeID = bundle.Document.Blocks[0].NodeID
		}, want: "duplicate node ID"},
		{name: "missing parent", edit: func(bundle *Bundle) {
			bundle.Document.Blocks[1].ParentNodeID = "section.missing"
		}, want: "invalid parent"},
		{name: "broken rhetorical edge", edit: func(bundle *Bundle) {
			bundle.Document.Blocks[1].Supports = []string{"node.missing"}
		}, want: "invalid rhetorical target"},
		{name: "asset hash mismatch", edit: func(bundle *Bundle) {
			bundle.Document.Assets[0].SHA256 = strings.Repeat("0", 64)
		}, want: "hash does not match"},
		{name: "active SVG", edit: func(bundle *Bundle) {
			active := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
			bundle.Document.Assets[0].DataBase64 = base64.StdEncoding.EncodeToString(active)
			bundle.Document.Assets[0].SHA256 = SHA256(active)
		}, want: "active or external content"},
		{name: "external SVG reference", edit: func(bundle *Bundle) {
			external := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><image href="https://example.com/a.png"/></svg>`)
			bundle.Document.Assets[0].DataBase64 = base64.StdEncoding.EncodeToString(external)
			bundle.Document.Assets[0].SHA256 = SHA256(external)
		}, want: "active or external content"},
		{name: "figure alt mismatch", edit: func(bundle *Bundle) {
			bundle.Document.Blocks[2].Figure.Alt = "다른 설명"
		}, want: "accessibility metadata does not match"},
		{name: "unused asset", edit: func(bundle *Bundle) {
			bundle.Document.Blocks[2].Figure = nil
			bundle.Document.Blocks[2].Kind = "prose"
			bundle.Document.Blocks[2].Prose = "figure removed"
		}, want: "not used by any figure"},
		{name: "invalid external reference", edit: func(bundle *Bundle) {
			bundle.Document.References[0].Target = "javascript:alert(1)"
		}, want: "unsupported target"},
		{name: "unused reference", edit: func(bundle *Bundle) {
			bundle.Document.Blocks[7].RefersTo = nil
		}, want: "not used by any block"},
		{name: "missing evidence reference", edit: func(bundle *Bundle) {
			bundle.Document.Blocks[1].EvidenceRefs = []string{"ref.missing"}
		}, want: "invalid evidence reference"},
		{name: "cross-reference used as evidence", edit: func(bundle *Bundle) {
			bundle.Document.References[0].Kind = "cross_reference"
			bundle.Document.References[0].Target = "prose.context"
			bundle.Document.Blocks[1].EvidenceRefs = []string{"ref.plan"}
		}, want: "invalid evidence reference"},
		{name: "missing requirement reference", edit: func(bundle *Bundle) {
			bundle.Document.Blocks[1].RequirementRefs = []string{"req.missing"}
		}, want: "invalid requirement reference"},
		{name: "duplicate coverage requirement", edit: func(bundle *Bundle) {
			bundle.Document.Coverage = append(bundle.Document.Coverage, bundle.Document.Coverage[0])
		}, want: "duplicate coverage requirement"},
		{name: "unregistered document extension", edit: func(bundle *Bundle) {
			bundle.Document.Extensions = map[string]json.RawMessage{"example": json.RawMessage(`{"value":true}`)}
		}, want: "registered Phase 0 extension profile"},
		{name: "incompatible block payload", edit: func(bundle *Bundle) {
			bundle.Document.Blocks[1].Items = []string{"silently lost"}
		}, want: "incompatible items payload"},
		{name: "unbound transition", edit: func(bundle *Bundle) {
			bundle.Narrative.TransitionObligations[0].ToSectionID = "section.missing"
		}, want: "invalid narrative transition"},
		{name: "dependency cycle", edit: func(bundle *Bundle) {
			bundle.Narrative.DependencyEdges = append(bundle.Narrative.DependencyEdges, DependencyEdge{FromSectionID: "section.end", ToSectionID: "section.context", Reason: "cycle"})
		}, want: "contains a cycle"},
		{name: "unresolved narrative loop", edit: func(bundle *Bundle) {
			bundle.Narrative.Callbacks = nil
		}, want: "has no callback"},
		{name: "callback resolving Section mismatch", edit: func(bundle *Bundle) {
			bundle.Narrative.Callbacks[0].ResolvedSectionID = "section.mechanism"
		}, want: "resolving Section differs"},
		{name: "stale flow attestation", edit: func(bundle *Bundle) {
			bundle.Document.Blocks[1].Prose += " 변경"
		}, want: "flow attestation is stale"},
		{name: "accepted unresolved flow", edit: func(bundle *Bundle) {
			bundle.Attestation.SectionHandoffsResolved = false
		}, want: "accepted flow attestation contains unresolved"},
		{name: "accepted repetition finding", edit: func(bundle *Bundle) {
			bundle.Attestation.AccidentalRepetitionFindings = []FlowFinding{{NodeID: "prose.end", Detail: "결론이 도입을 반복한다"}}
		}, want: "accepted flow attestation contains unresolved"},
		{name: "accepted abrupt transition finding", edit: func(bundle *Bundle) {
			bundle.Attestation.AbruptTransitionFindings = []FlowFinding{{NodeID: "section.end", Detail: "전환 근거가 없다"}}
		}, want: "accepted flow attestation contains unresolved"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bundle := sampleBundle(t, "T2")
			test.edit(&bundle)
			if err := ValidateBundle(bundle); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestValidateBundleRejectsUnsafeSVGDocuments(t *testing.T) {
	tests := []struct {
		name string
		svg  string
	}{
		{name: "second root", svg: `<svg xmlns="http://www.w3.org/2000/svg"><rect width="1" height="1"/></svg><svg xmlns="http://www.w3.org/2000/svg"/>`},
		{name: "nested root", svg: `<svg xmlns="http://www.w3.org/2000/svg"><svg/></svg>`},
		{name: "wrong namespace", svg: `<svg xmlns="https://example.com/not-svg"><rect width="1" height="1"/></svg>`},
		{name: "prefixed namespace", svg: `<svg:svg xmlns:svg="http://www.w3.org/2000/svg"><svg:rect width="1" height="1"/></svg:svg>`},
		{name: "root text after document", svg: `<svg xmlns="http://www.w3.org/2000/svg"><rect width="1" height="1"/></svg>trailing`},
		{name: "processing instruction", svg: `<?xml-stylesheet href="https://example.com/a.css"?><svg xmlns="http://www.w3.org/2000/svg"/>`},
		{name: "escaped CSS URL", svg: `<svg xmlns="http://www.w3.org/2000/svg"><rect fill="u\\72l(https://example.com/a.png)"/></svg>`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bundle := sampleBundle(t, "T2")
			data := []byte(test.svg)
			bundle.Document.Assets[0].DataBase64 = base64.StdEncoding.EncodeToString(data)
			bundle.Document.Assets[0].SHA256 = SHA256(data)
			if err := ValidateBundle(bundle); err == nil || !strings.Contains(err.Error(), "active or external content") {
				t.Fatalf("error = %v, want unsafe SVG rejection", err)
			}
		})
	}
}

func TestLoadBundleRejectsUnknownFields(t *testing.T) {
	bundle := sampleBundle(t, "T0")
	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"arm":"T0"`, `"arm":"T0","unexpected":true`, 1))
	if _, err := LoadBundle(raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error = %v, want unknown field rejection", err)
	}
}

func TestRenderTargetsAreDeterministicSelfContainedAndProsePreserving(t *testing.T) {
	bundle := sampleBundle(t, "T2")
	markdownA, markdownReceiptsA, err := RenderMarkdown(bundle.Document)
	if err != nil {
		t.Fatal(err)
	}
	markdownB, markdownReceiptsB, err := RenderMarkdown(bundle.Document)
	if err != nil {
		t.Fatal(err)
	}
	if string(markdownA) != string(markdownB) || !equalJSON(t, markdownReceiptsA, markdownReceiptsB) {
		t.Fatal("Markdown compilation is not deterministic")
	}
	for _, block := range bundle.Document.Blocks {
		if block.Prose != "" && !strings.Contains(string(markdownA), block.Prose) {
			t.Fatalf("Markdown changed authored prose for %s", block.NodeID)
		}
	}

	htmlA, htmlReceiptsA, err := RenderHTML(bundle.Document)
	if err != nil {
		t.Fatal(err)
	}
	htmlB, htmlReceiptsB, err := RenderHTML(bundle.Document)
	if err != nil {
		t.Fatal(err)
	}
	if string(htmlA) != string(htmlB) || !equalJSON(t, htmlReceiptsA, htmlReceiptsB) {
		t.Fatal("HTML compilation is not deterministic")
	}
	htmlText := string(htmlA)
	for _, forbidden := range []string{"<script", "src=\"http://", "src=\"https://", "url(http"} {
		if strings.Contains(htmlText, forbidden) {
			t.Fatalf("self-contained HTML contains forbidden dependency %q", forbidden)
		}
	}
	for _, required := range []string{"<!doctype html>", "Content-Security-Policy", "default-src 'none'", "<main>", "<figure>", "<figcaption>", "<table>", "data:image/svg+xml;base64,"} {
		if !strings.Contains(htmlText, required) {
			t.Fatalf("self-contained HTML missing %q", required)
		}
	}
	for _, required := range []string{`class="table-scroll"`, `role="region"`, `tabindex="0"`, `data-columns="2"`, `aria-label="Phase 0 책임 경계"`, `.table-scroll[data-columns="3"] table{min-width:42rem}`, `.table-scroll[data-columns] table{width:100%;min-width:0}`} {
		if !strings.Contains(htmlText, required) {
			t.Fatalf("responsive table HTML missing %q", required)
		}
	}
	if !strings.Contains(htmlText, "문제가 복잡해 보이는 이유는") {
		t.Fatal("HTML lost authored Korean prose")
	}

	parser := goldmark.DefaultParser()
	source := text.NewReader(markdownA)
	document := parser.Parse(source)
	if document.ChildCount() == 0 {
		t.Fatal("generated Markdown parsed to an empty document")
	}
}

func TestRenderHTMLResponsiveTablePreservesEveryCell(t *testing.T) {
	document := Document{
		Title: "Responsive table", Language: "ko",
		Blocks: []Block{{
			NodeID: "table.mobile", Kind: "table",
			Table: &Table{
				Caption: "모바일 근거 표",
				Columns: []string{"연구", "표본", "결과"},
				Rows:    [][]string{{"첫 번째 연구", "386명", "49.93 대 30.41"}, {"두 번째 연구", "1,309명", "4.65 대 4.48"}},
			},
		}},
	}

	rendered, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	htmlText := string(rendered)
	for _, required := range []string{`class="table-scroll"`, `aria-label="모바일 근거 표"`, `data-columns="3"`, `<caption>모바일 근거 표</caption>`, `<th scope="col">연구</th>`, `<td>1,309명</td>`, `<td>4.65 대 4.48</td>`} {
		if !strings.Contains(htmlText, required) {
			t.Fatalf("responsive table lost %q:\n%s", required, htmlText)
		}
	}
	if got := strings.Count(htmlText, "<td>"); got != 6 {
		t.Fatalf("responsive table cell count = %d, want 6", got)
	}
}

func TestRenderMarkdownUsesExplicitPortableFallbacks(t *testing.T) {
	bundle := sampleBundle(t, "T0")
	bundle.Document.References[0] = Reference{RefID: "ref.plan", Kind: "cross_reference", Target: "prose.context", VisibleLabel: "도입으로 이동"}
	bundle.Document.Blocks[7].RefersTo = []string{"ref.plan"}
	bundle.Document.Blocks = append(bundle.Document.Blocks, Block{NodeID: "raw.example", Kind: "raw", Extension: json.RawMessage("{\"literal\":\"```\"}")})

	markdown, receipts, err := RenderMarkdown(bundle.Document)
	if err != nil {
		t.Fatal(err)
	}
	markdownText := string(markdown)
	if strings.Contains(markdownText, "](#prose.context)") || !strings.Contains(markdownText, "document node `prose.context`") {
		t.Fatalf("Markdown cross-reference fallback differs:\n%s", markdownText)
	}
	if !strings.Contains(markdownText, "````json\n{\"literal\":\"```\"}\n````") {
		t.Fatalf("raw extension did not choose a safe fence:\n%s", markdownText)
	}
	if len(receipts) != 2 || receipts[0].Capability != "raw_extension" || receipts[1].Capability != "stable_cross_reference" {
		t.Fatalf("degradation receipts = %#v", receipts)
	}
}

func TestRunWritesArchiveArtifactsAndExplicitPDFBlocker(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	archive := filepath.Join(base, "archive")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(archive, 0o700); err != nil {
		t.Fatal(err)
	}
	bundle := sampleBundle(t, "T2")
	bundlePath := filepath.Join(archive, "fixture-t2.json")
	writeTestJSON(t, bundlePath, bundle)

	result, err := Run(context.Background(), RunConfig{
		ArchiveRoot: archive, RepositoryRoot: repo, BundlePath: bundlePath,
		RunID: "t2-blocker", ChromePath: filepath.Join(base, "missing-chrome"),
	})
	if err != nil {
		t.Fatalf("Run should preserve optional PDF blocker without failing: %v", err)
	}
	for _, path := range []string{result.ManifestPath, result.MarkdownPath, result.HTMLPath, result.BlockerPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected output %s: %v", path, err)
		}
		if pathInside(repo, path) {
			t.Fatalf("generated output entered repository: %s", path)
		}
	}
	if result.PDFPath != "" || result.Manifest.PDFBlocker == nil || result.Manifest.PDFBlocker.Code != "pdf_renderer_unavailable" {
		t.Fatalf("PDF blocker result differs: %#v", result)
	}
	if result.Manifest.CompilerVersion != CompilerVersion {
		t.Fatalf("compiler version = %q, want %q", result.Manifest.CompilerVersion, CompilerVersion)
	}
	if _, err := Run(context.Background(), RunConfig{
		ArchiveRoot: archive, RepositoryRoot: repo, BundlePath: bundlePath,
		RunID: "t2-blocker", ChromePath: filepath.Join(base, "missing-chrome"),
	}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("immutable run directory reuse error = %v", err)
	}
}

func TestRunRejectsRunsDirectorySymlinkEscape(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	archive := filepath.Join(base, "archive")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(archive, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(repo, filepath.Join(archive, "runs")); err != nil {
		t.Fatal(err)
	}
	bundlePath := filepath.Join(archive, "fixture-t0.json")
	writeTestJSON(t, bundlePath, sampleBundle(t, "T0"))

	_, err := Run(context.Background(), RunConfig{ArchiveRoot: archive, RepositoryRoot: repo, BundlePath: bundlePath, RunID: "symlink-escape"})
	if err == nil || !strings.Contains(err.Error(), "runs directory must remain") {
		t.Fatalf("error = %v, want runs directory escape rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(repo, "symlink-escape")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("run escaped through symlink: %v", statErr)
	}
}

func TestRunInvalidBundleCreatesNoRunDirectory(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	archive := filepath.Join(base, "archive")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(archive, 0o700); err != nil {
		t.Fatal(err)
	}
	bundle := sampleBundle(t, "T0")
	bundle.Document.Blocks[1].ParentNodeID = "missing"
	bundlePath := filepath.Join(archive, "invalid.json")
	writeTestJSON(t, bundlePath, bundle)
	_, err := Run(context.Background(), RunConfig{ArchiveRoot: archive, RepositoryRoot: repo, BundlePath: bundlePath, RunID: "invalid"})
	if err == nil {
		t.Fatal("Run accepted invalid bundle")
	}
	if _, statErr := os.Stat(filepath.Join(archive, "runs", "invalid")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("invalid input created a run directory: %v", statErr)
	}
}

func sampleBundle(t *testing.T, arm string) Bundle {
	t.Helper()
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="320" height="100" viewBox="0 0 320 100"><rect width="320" height="100" fill="#e8f2fc"/><path d="M40 50h220" stroke="#1d5da3" stroke-width="8"/><circle cx="270" cy="50" r="20" fill="#d9ad52"/></svg>`)
	document := Document{
		SchemaVersion:  DocumentSchemaVersion,
		PipelineFamily: PipelineFamily,
		DocumentID:     "doc.phase0",
		RevisionID:     "rev.t0",
		Title:          "구조와 흐름을 함께 다루는 보고서",
		Language:       "ko",
		Blocks: []Block{
			{NodeID: "section.context", Kind: "section", Level: 2, Title: "구조만으로는 부족한 이유", SemanticRole: "context"},
			{NodeID: "prose.context", Kind: "prose", ParentNodeID: "section.context", Prose: "문제가 복잡해 보이는 이유는 구조와 글의 흐름이 서로 다른 책임이기 때문입니다.", SemanticRole: "opening"},
			{NodeID: "figure.flow", Kind: "figure", ParentNodeID: "section.context", Figure: &Figure{AssetID: "asset.flow", Caption: "구조에서 독자 이해로 이어지는 흐름", Alt: "왼쪽에서 오른쪽으로 이어지는 선과 결론을 나타내는 원"}},
			{NodeID: "section.mechanism", Kind: "section", Level: 2, Title: "두 계약의 분리", SemanticRole: "mechanism"},
			{NodeID: "prose.mechanism", Kind: "prose", ParentNodeID: "section.mechanism", Prose: "Semantic IL은 의미와 자산을 보존하고, Narrative Contract는 독자가 따라갈 질문과 전환을 기록합니다.", SemanticRole: "mechanism", Supports: []string{"prose.context"}},
			{NodeID: "table.roles", Kind: "table", ParentNodeID: "section.mechanism", Table: &Table{Caption: "Phase 0 책임 경계", Columns: []string{"계층", "책임"}, Rows: [][]string{{"Narrative Contract", "독자 여정과 전환 의무"}, {"Semantic IL", "의미와 자산 보존"}, {"Compiler", "결정론적 출력"}}}},
			{NodeID: "section.end", Kind: "section", Level: 2, Title: "전체 원고에서 확인할 것", SemanticRole: "conclusion"},
			{NodeID: "prose.end", Kind: "prose", ParentNodeID: "section.end", Prose: "마지막 판단은 tree가 유효한지가 아니라, 전체 원고가 하나의 설명으로 읽히는지에 달려 있습니다.", SemanticRole: "conclusion", Elaborates: []string{"prose.mechanism"}, RefersTo: []string{"ref.plan"}},
		},
		References: []Reference{{RefID: "ref.plan", Kind: "citation", Target: "https://example.com/report-plan", VisibleLabel: "Phase 0 계획", Locator: "독립 실험 계약"}},
		Assets:     []Asset{{AssetID: "asset.flow", MediaType: "image/svg+xml", SHA256: SHA256(svg), DataBase64: base64.StdEncoding.EncodeToString(svg), Alt: "왼쪽에서 오른쪽으로 이어지는 선과 결론을 나타내는 원", LicenseStatus: "allowed"}},
		Coverage:   []Coverage{{RequirementID: "req.flow", NodeIDs: []string{"prose.context", "prose.mechanism", "prose.end"}, Status: "covered"}},
		Provenance: map[string]string{"fixture": "phase0-synthetic", "source": "user-approved architecture plan"},
	}
	bundle := Bundle{SchemaVersion: BundleSchemaVersion, Arm: arm, Document: document}
	if arm == "T0" {
		return bundle
	}
	bundle.Document.RevisionID = "rev." + strings.ToLower(arm)
	bundle.Document.NarrativeContractID = "narrative.phase0"
	bundle.Narrative = &Narrative{
		SchemaVersion: NarrativeSchemaVersion, ContractID: "narrative.phase0", DocumentID: document.DocumentID,
		CentralQuestion: "구조화가 글의 흐름을 어떻게 도울 수 있는가?", ReaderTakeaway: "구조와 서사 계약은 분리하되 전체 원고에서 다시 결합해야 합니다.",
		Throughline:   "구조 보존에서 출발해 독자 이해를 완성하는 책임 경계를 설명합니다.",
		ReaderJourney: []string{"구조와 흐름의 차이를 이해한다", "두 계약의 책임을 구분한다", "전체 원고 검토의 필요성을 판단한다"},
		ArgumentArc:   []string{"setup", "mechanism", "synthesis"},
		SectionRoles: []SectionRole{
			{SectionID: "section.context", Role: "setup", QuestionAnswered: "왜 AST만으로 부족한가?", MustEstablish: []string{"구조와 흐름은 다른 책임이다"}, OpenLoopsIntroduced: []string{"loop.whole-read"}, TransitionToNext: "문제 진단에서 책임 분리로 이동한다"},
			{SectionID: "section.mechanism", Role: "mechanism", QuestionAnswered: "각 계약은 무엇을 소유하는가?", PrerequisiteSections: []string{"section.context"}, TransitionFromPrevious: "구조의 한계를 두 계약으로 해결한다", TransitionToNext: "계약 정의를 실제 판정 방법으로 넘긴다"},
			{SectionID: "section.end", Role: "synthesis", QuestionAnswered: "어떻게 최종 품질을 판정하는가?", PrerequisiteSections: []string{"section.mechanism"}, CallbacksResolved: []string{"loop.whole-read"}, TransitionFromPrevious: "구조적 계약을 전체 원고의 독자 경험으로 검증한다"},
		},
		DependencyEdges:       []DependencyEdge{{FromSectionID: "section.context", ToSectionID: "section.mechanism", Reason: "진단 뒤 책임을 분리한다"}, {FromSectionID: "section.mechanism", ToSectionID: "section.end", Reason: "계약 뒤 전체 원고 판정으로 간다"}},
		TransitionObligations: []Transition{{FromSectionID: "section.context", ToSectionID: "section.mechanism", Obligation: "구조와 흐름의 차이를 두 계약의 필요성으로 연결한다"}, {FromSectionID: "section.mechanism", ToSectionID: "section.end", Obligation: "정의된 계약이 실제 글에서 검증돼야 함을 연결한다"}},
		ContinuityTerms:       []ContinuityTerm{{Canonical: "Semantic IL", Aliases: []string{"IL"}}, {Canonical: "Narrative Contract"}},
		OpenLoops:             []OpenLoop{{LoopID: "loop.whole-read", IntroducedSectionID: "section.context", Question: "그렇다면 흐름은 어디서 검증하는가?"}},
		Callbacks:             []Callback{{LoopID: "loop.whole-read", ResolvedSectionID: "section.end", Obligation: "전체 원고를 독자 순서로 읽어 판정한다고 답한다"}},
		RepetitionPolicies:    []RepetitionPolicy{{Concept: "구조와 흐름의 분리", Policy: "도입에서 설명하고 결론에서는 판정 기준으로만 상기한다"}},
		ConclusionObligations: []string{"AST validity와 whole-read quality를 구분한다"}, VoiceAndTone: "직접적이고 자연스러운 한국어 설명",
	}
	if arm == "T2" {
		projection, _, err := RenderMarkdown(bundle.Document)
		if err != nil {
			t.Fatal(err)
		}
		bundle.Attestation = &FlowAttestation{
			SchemaVersion: AttestationSchemaVersion, DocumentID: bundle.Document.DocumentID, RevisionID: bundle.Document.RevisionID,
			LinearProjectionSHA256: SHA256(projection), Reviewer: "phase0-human-reader", ReviewedFullManuscript: true,
			CentralThreadPreserved: true, SectionHandoffsResolved: true, OpenLoopsResolved: true, TerminologyContinuous: true, Verdict: "accept",
		}
	}
	return bundle
}

func equalJSON(t *testing.T, left, right any) bool {
	t.Helper()
	leftJSON, err := json.Marshal(left)
	if err != nil {
		t.Fatal(err)
	}
	rightJSON, err := json.Marshal(right)
	if err != nil {
		t.Fatal(err)
	}
	return string(leftJSON) == string(rightJSON)
}

func writeTestJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

package reportilphase0

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

func TestCompileReportFirstDraftProjectsAuthoredContentWithoutRewrite(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_report_first_projection")
	receipt := testSourceReadReceipt(catalog)
	draft := reportFirstAuthorDraft{
		Title:    "다케다성의 역사적 역할",
		Language: "ko",
		Sections: []reportFirstSectionDraft{
			{
				Title: "산성과 은광이 만나는 자리",
				Blocks: []reportFirstBlockDraft{
					{
						Kind:               "prose",
						Prose:              "다케다성은 산길의 방어 거점이자 이쿠노 은광을 둘러싼 지역 권력 관계를 읽게 하는 성곽이다.",
						EvidenceSourceKeys: []string{"source_001"},
					},
					{
						Kind:               "list",
						Items:              []string{"1433년과 1443년이라는 두 축성 전승이 남아 있다.", "1585년 이후 석벽 축조가 성의 모습을 바꾸었다."},
						EvidenceSourceKeys: []string{"source_001"},
					},
				},
			},
			{
				Title: "남는 판단",
				Blocks: []reportFirstBlockDraft{
					{
						Kind:               "prose",
						Prose:              "다케다성의 가치는 광산을 직접 통제했다는 단정이 아니라, 군사·교통·생산이 한 지역 권력 안에서 맞물린 흔적을 보여준다는 데 있다.",
						EvidenceSourceKeys: []string{"source_001"},
					},
				},
			},
		},
	}
	narrative, document, err := compileReportFirstAuthorDraft(
		draft,
		"narrative_1",
		"doc_1",
		"다케다성과 이쿠노 은광의 관계를 설명한다.",
		catalog,
		receipt,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(narrative.EvidencePackets) != 0 || len(narrative.ContinuityTerms) != 0 {
		t.Fatal("report-first compiler reintroduced proof metadata")
	}
	markdown, _, err := RenderMarkdown(document)
	if err != nil {
		t.Fatal(err)
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range draft.Sections {
		if !bytes.Contains(markdown, []byte(section.Title)) || !bytes.Contains(html, []byte(section.Title)) {
			t.Fatalf("projection lost section %q", section.Title)
		}
		for _, block := range section.Blocks {
			for _, value := range draftBlockReaderValues(block.documentDraft()) {
				if !bytes.Contains(markdown, []byte(value)) || !bytes.Contains(html, []byte(value)) {
					t.Fatalf("projection lost authored value %q", value)
				}
			}
		}
	}
}

func TestReportFirstDecodeRejectsLegacyBlockMetadata(t *testing.T) {
	_, err := strictDecode[reportFirstAuthorDraft]([]byte(`{
		"title":"Result",
		"language":"en",
		"sections":[
			{"title":"Answer","blocks":[{"kind":"prose","prose":"Answer.","evidence_source_keys":["source_001"],"semantic_role":"opening"}]},
			{"title":"Judgment","blocks":[{"kind":"prose","prose":"Judgment.","evidence_source_keys":["source_001"]}]}
		]
	}`))
	if err == nil {
		t.Fatal("report-first decoder accepted legacy semantic metadata")
	}
}

func TestCompileReportFirstDraftRejectsSourceKeysInHeadings(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_report_first_heading")
	base := reportFirstAuthorDraft{
		Title:    "Result",
		Language: "en",
		Sections: []reportFirstSectionDraft{
			{Title: "Answer", Blocks: []reportFirstBlockDraft{{Kind: "prose", Prose: "The direct answer.", EvidenceSourceKeys: []string{"source_001"}}}},
			{Title: "Judgment", Blocks: []reportFirstBlockDraft{{Kind: "prose", Prose: "The final judgment.", EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
	for name, mutate := range map[string]func(*reportFirstAuthorDraft){
		"title":   func(draft *reportFirstAuthorDraft) { draft.Title = "Internal source_001" },
		"section": func(draft *reportFirstAuthorDraft) { draft.Sections[0].Title = "Internal source_001" },
	} {
		t.Run(name, func(t *testing.T) {
			draft := base
			draft.Sections = append([]reportFirstSectionDraft(nil), base.Sections...)
			mutate(&draft)
			if _, _, err := compileReportFirstAuthorDraft(
				draft,
				"narrative_heading",
				"doc_heading",
				"Explain the result.",
				catalog,
				testSourceReadReceipt(catalog),
				nil,
			); err == nil {
				t.Fatal("report-first compiler accepted an internal source key in a heading")
			}
		})
	}
}

func TestCompileReportFirstDraftRejectsMissingBlockCitations(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_report_first_citations")
	for name, keys := range map[string][]string{
		"omitted": nil,
		"empty":   {},
	} {
		t.Run(name, func(t *testing.T) {
			draft := reportFirstAuthorDraft{
				Title:    "Result",
				Language: "en",
				Sections: []reportFirstSectionDraft{
					{Title: "Answer", Blocks: []reportFirstBlockDraft{{Kind: "prose", Prose: "The direct answer.", EvidenceSourceKeys: keys}}},
					{Title: "Judgment", Blocks: []reportFirstBlockDraft{{Kind: "prose", Prose: "The final judgment.", EvidenceSourceKeys: []string{"source_001"}}}},
				},
			}
			if _, _, err := compileReportFirstAuthorDraft(
				draft,
				"narrative_citations",
				"doc_citations",
				"Explain the result.",
				catalog,
				testSourceReadReceipt(catalog),
				nil,
			); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeSourceReadContract {
				t.Fatalf("missing citations error = %v", err)
			}
		})
	}
}

func TestCompileReportFirstDraftRequiresProseOpening(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_report_first_opening")
	draft := reportFirstAuthorDraft{
		Title:    "Result",
		Language: "en",
		Sections: []reportFirstSectionDraft{
			{Title: "Opening", Blocks: []reportFirstBlockDraft{{Kind: "list", Items: []string{"Not a direct prose answer"}, EvidenceSourceKeys: []string{"source_001"}}}},
			{Title: "Conclusion", Blocks: []reportFirstBlockDraft{{Kind: "prose", Prose: "The final judgment.", EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
	if _, _, err := compileReportFirstAuthorDraft(
		draft,
		"narrative_opening",
		"doc_opening",
		"Explain the result.",
		catalog,
		testSourceReadReceipt(catalog),
		nil,
	); err == nil {
		t.Fatal("report-first compiler accepted a non-prose opening")
	}
}

func TestReportFirstCompilerAppliesAuthoringModeSectionLimits(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_report_first_sections")
	section := reportFirstSectionDraft{
		Title:  "Substantial section",
		Blocks: []reportFirstBlockDraft{{Kind: "prose", Prose: "One connected factual passage.", EvidenceSourceKeys: []string{"source_001"}}},
	}
	draftWithSections := func(count int) reportFirstAuthorDraft {
		draft := reportFirstAuthorDraft{Title: "Result", Language: "en", Sections: make([]reportFirstSectionDraft, count)}
		for index := range draft.Sections {
			draft.Sections[index] = section
			draft.Sections[index].Title = fmt.Sprintf("Section %d", index+1)
		}
		return draft
	}
	for _, count := range []int{2, 8} {
		if _, _, err := compileReportFirstAuthorDraft(
			draftWithSections(count), fmt.Sprintf("narrative_sections_%d", count), fmt.Sprintf("doc_sections_%d", count),
			"Explain the result.", catalog, testSourceReadReceipt(catalog), nil,
		); err != nil {
			t.Fatalf("%d-section standard report was rejected: %v", count, err)
		}
	}
	if _, _, err := compileReportFirstAuthorDraft(
		draftWithSections(9), "narrative_sections_9", "doc_sections_9", "Explain the result.",
		catalog, testSourceReadReceipt(catalog), nil,
	); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeDocumentContract {
		t.Fatalf("nine-section standard report error = %v", err)
	}
	for _, count := range []int{4, 9, 16} {
		if _, _, err := compileReportFirstAuthorDraftForMode(
			draftWithSections(count), AuthoringModeLongForm,
			fmt.Sprintf("narrative_long_sections_%d", count), fmt.Sprintf("doc_long_sections_%d", count),
			"Explain the result.", catalog, testSourceReadReceipt(catalog), nil,
		); err != nil {
			t.Fatalf("%d-section long-form report was rejected: %v", count, err)
		}
	}
	for _, count := range []int{3, 17} {
		if _, _, err := compileReportFirstAuthorDraftForMode(
			draftWithSections(count), AuthoringModeLongForm,
			fmt.Sprintf("narrative_long_sections_%d", count), fmt.Sprintf("doc_long_sections_%d", count),
			"Explain the result.", catalog, testSourceReadReceipt(catalog), nil,
		); err == nil || providerValidationCode(err) != reportexecution.ProviderValidationCodeDocumentContract {
			t.Fatalf("%d-section long-form report error = %v", count, err)
		}
	}
}

func TestLongFormAuthorPromptsRequireDepthWithoutPipelineMachinery(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_report_first_long_prompt")
	config := ProductConfig{
		MissionObjective: "Explain the subject in depth.", Title: "Subject",
		AuthoringMode: AuthoringModeLongForm,
	}
	for name, prompt := range map[string]string{
		"direct": directSourceAuthorPrompt(config, catalog, false),
		"memory": reportFirstAuthorPrompt(config, catalog, false),
	} {
		for _, required := range []string{"long-form report", "genuinely long-form report", "four to sixteen reader-facing sections", "Do not pad, repeat"} {
			if !strings.Contains(prompt, required) {
				t.Fatalf("%s long-form prompt lacks %q: %s", name, required, prompt)
			}
		}
		for _, forbidden := range []string{"section fan-out", "part assembly", "write each section in a separate session"} {
			if strings.Contains(prompt, forbidden) {
				t.Fatalf("%s long-form prompt exposes classic machinery %q: %s", name, forbidden, prompt)
			}
		}
	}
	memoryPrompt := editorialMemoryPrompt(config, catalog)
	if !strings.Contains(memoryPrompt, "The downstream manuscript is long-form") ||
		!strings.Contains(memoryPrompt, "supporting accounts that materially deepen chronology") ||
		!strings.Contains(memoryPrompt, "source-native explanatory material") ||
		!strings.Contains(memoryPrompt, "equations and variable meanings") ||
		!strings.Contains(memoryPrompt, "executable code, API calls, commands, and configuration") ||
		!strings.Contains(memoryPrompt, "multi-step API lifecycle") ||
		!strings.Contains(memoryPrompt, "stateful/stateless transport alternative") ||
		!strings.Contains(memoryPrompt, "command-line budget control") ||
		!strings.Contains(memoryPrompt, "retain its exact formula, syntax, dimensions") ||
		!strings.Contains(memoryPrompt, "incomplete, mixed-script, placeholder-like, or malformed personal name") ||
		!strings.Contains(memoryPrompt, "Never guess the missing identity") {
		t.Fatalf("long-form editorial memory prompt lacks depth, explanatory-payload, or name-integrity guidance: %s", memoryPrompt)
	}
}

func TestRunReportFirstAuthorStageRequiresFinalizedMCPDocument(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_report_first_workspace")
	provider := &recordingProvider{outputs: []string{"workspace finalized"}}
	reader := &fixedAuthorDocumentReader{err: errors.New("workspace was not finalized")}
	config := ProductConfig{
		MissionID: catalog.MissionID, MissionObjective: "Explain the result.", Title: "Result",
		Provider: provider, PendingEventID: "evt_pending",
		VerifySourceRead: &acceptingSourceReadVerifier{}, AuthorDocuments: reader,
		NewID: func(prefix string) string { return prefix + "_1" },
	}
	_, _, _, results, err := runReportFirstAuthorStage(
		context.Background(), config, catalog, testEditorialMemory(catalog),
		reportilcontract.EditorialMemoryReceipt{ArtifactID: "art_memory", SHA256: strings.Repeat("d", 64)}, nil, false,
	)
	if err == nil || len(results) != 1 || len(provider.requests) != 1 || reader.calls != 1 {
		t.Fatalf("workspace rejection calls=%d/%d/%d err=%v", len(results), len(provider.requests), reader.calls, err)
	}
}

func TestReportFirstAuthorPromptUsesMCPWorkspaceAndReaderStandards(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_report_first_reader_contract")
	prompt := reportFirstAuthorPrompt(ProductConfig{
		MissionObjective: "Explain the castle's relationship to a mine and route.",
		Title:            "Castle history",
	}, catalog, true)
	for _, required := range []string{
		"server-owned MCP document workspace",
		"plasma.report_il.document.start",
		"plasma.report_il.document.append once per content block",
		"plasma.report_il.document.read from offset 0",
		"Every replacement invalidates the prior full read",
		"plasma.report_il.document.finalize exactly once",
		"Give the subject and central answer in the first body paragraph",
		"Preserve the concrete dates, durations, people, roles, structures, relationships",
		"End with a sharper subject-level judgment",
		"The assembled MCP workspace is the only authoritative manuscript",
		"selected catalog is ordered strongest-first",
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("report-first author prompt lacks %q: %s", required, prompt)
		}
	}
	for _, forbidden := range []string{"material_source_accounts", "coverage map", "Return two to eight"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("report-first author prompt retains proof-oriented term %q: %s", forbidden, prompt)
		}
	}
	unranked := reportFirstAuthorPrompt(ProductConfig{MissionObjective: "Explain the result.", Title: "Result"}, catalog, false)
	if strings.Contains(unranked, "selected catalog is ordered strongest-first") {
		t.Fatal("report-first author prompt labels an unselected catalog as strongest-first")
	}
}

func TestRunReportFirstAuthorStageUsesOneMCPWorkspaceCall(t *testing.T) {
	catalog := testSourceCatalog(t, "mis_report_first_call")
	draft := reportFirstAuthorDraft{
		Title:    "Bounded result",
		Language: "en",
		Sections: []reportFirstSectionDraft{
			{Title: "Answer", Blocks: []reportFirstBlockDraft{{Kind: "prose", Prose: "One bounded fact defines the result.", EvidenceSourceKeys: []string{"source_001"}}}},
			{Title: "Judgment", Blocks: []reportFirstBlockDraft{{Kind: "prose", Prose: "That fact supports one bounded conclusion.", EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
	provider := &recordingProvider{outputs: []string{string(mustMarshal(draft))}}
	config := ProductConfig{
		MissionID:        catalog.MissionID,
		MissionObjective: "Explain the result.",
		Title:            "Result",
		Provider:         provider,
		PendingEventID:   "evt_pending",
		VerifySourceRead: &acceptingSourceReadVerifier{},
		AuthorDocuments:  &fixedAuthorDocumentReader{document: testAuthorDocument()},
		NewID: func(prefix string) string {
			return prefix + "_1"
		},
	}
	_, _, _, results, err := runReportFirstAuthorStage(
		context.Background(), config, catalog, testEditorialMemory(catalog),
		reportilcontract.EditorialMemoryReceipt{ArtifactID: "art_memory", SHA256: strings.Repeat("d", 64)}, nil, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || len(provider.requests) != 1 {
		t.Fatalf("report-first calls = %d/%d", len(results), len(provider.requests))
	}
	request := provider.requests[0]
	if request.UserText != "report IL il_narrative" || request.ReasoningEffort != "xhigh" ||
		!request.IgnoreUserConfig || !request.EphemeralSession || request.DisableTools ||
		strings.Contains(request.Prompt, "Generate only strict JSON") ||
		!reflect.DeepEqual(request.ExtraMCPTools, []string{
			reportilcontract.EditorialMemoryReadTool,
			reportilcontract.AuthorDocumentStartTool, reportilcontract.AuthorDocumentAppendTool,
			reportilcontract.AuthorDocumentReadTool, reportilcontract.AuthorDocumentReplaceTool,
			reportilcontract.AuthorDocumentFinalizeTool,
		}) || len(request.OutputJSONSchema) != 0 {
		t.Fatalf("report-first request = %#v", request)
	}
}

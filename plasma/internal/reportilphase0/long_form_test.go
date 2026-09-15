package reportilphase0

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

func TestCompileLongFormAuthorDocumentProjectsEquationToMarkdownAndHTML(t *testing.T) {
	catalog := testSourceCatalogWithCount(t, "mis_long_form_equation", 6)
	parts := make([]reportilcontract.AuthorPart, 2)
	for partIndex := range parts {
		partKey := fmt.Sprintf("part_%03d", partIndex+1)
		parts[partIndex] = reportilcontract.AuthorPart{PartKey: partKey, Title: "Part"}
		for sectionIndex := 0; sectionIndex < 3; sectionIndex++ {
			sectionKey := fmt.Sprintf("%s.section_%03d", partKey, sectionIndex+1)
			blocks := []reportilcontract.AuthorBlock{{
				BlockKey: sectionKey + ".block_001", Kind: "prose",
				Prose: "A substantial reader-facing passage defines the relationship.", EvidenceSourceKeys: []string{"source_001"},
			}}
			if partIndex == 0 && sectionIndex == 0 {
				blocks = append(blocks, reportilcontract.AuthorBlock{
					BlockKey: sectionKey + ".block_002", Kind: "equation",
					Equation:           &reportilcontract.AuthorEquation{Expression: `\[ C = n_i p_i + n_o p_o \]`, Notation: "latex"},
					EvidenceSourceKeys: []string{"source_001"},
				})
			}
			parts[partIndex].Sections = append(parts[partIndex].Sections, reportilcontract.AuthorSection{
				SectionKey: sectionKey, Title: "Section", Blocks: blocks,
			})
		}
	}
	authored := reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.LongFormAuthorDocumentSchemaVersion,
		Title:         "Long report", Language: "en", Parts: parts,
	}
	_, document, err := compileLongFormAuthorDocument(
		authored, "narrative.equation", "document.equation", "Explain it.", catalog,
		editorialMemorySourceReadReceipt(catalog), nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	markdown, _, err := RenderMarkdown(document)
	if err != nil {
		t.Fatal(err)
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "$$\nC = n_i p_i + n_o p_o\n$$") ||
		strings.Contains(string(markdown), `\[`) ||
		!strings.Contains(string(html), `class="equation"`) ||
		!strings.Contains(string(html), `data-tex="C = n_i p_i + n_o p_o"`) ||
		!strings.Contains(string(html), `output:"mathml"`) ||
		strings.Contains(string(html), `<mtext>\[`) {
		t.Fatalf("equation projections:\n%s\n%s", markdown, html)
	}
}

func TestCompileLongFormAuthorDocumentPreservesPartSectionHierarchy(t *testing.T) {
	catalog := testSourceCatalogWithCount(t, "mis_long_form", 6)
	parts := make([]reportilcontract.AuthorPart, 2)
	for partIndex := range parts {
		partKey := "part_00" + string(rune('1'+partIndex))
		parts[partIndex] = reportilcontract.AuthorPart{PartKey: partKey, Title: "Part"}
		for sectionIndex := 0; sectionIndex < 3; sectionIndex++ {
			sectionKey := partKey + ".section_00" + string(rune('1'+sectionIndex))
			parts[partIndex].Sections = append(parts[partIndex].Sections, reportilcontract.AuthorSection{
				SectionKey: sectionKey, Title: "Section",
				Blocks: []reportilcontract.AuthorBlock{{
					BlockKey: sectionKey + ".block_001", Kind: "prose",
					Prose: "A substantial reader-facing passage.", EvidenceSourceKeys: []string{"source_001"},
				}},
			})
		}
	}
	authored := reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.LongFormAuthorDocumentSchemaVersion,
		Title:         "Long report", Language: "en", Parts: parts,
	}
	narrative, document, err := compileLongFormAuthorDocument(
		authored, "narrative.test", "document.test", "Explain it.", catalog,
		editorialMemorySourceReadReceipt(catalog), nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(readerSections(document)) != 6 || len(narrative.SectionRoles) != 6 {
		t.Fatalf("leaf section inventory = %d / %d", len(readerSections(document)), len(narrative.SectionRoles))
	}
	partCount, nestedCount := 0, 0
	for _, block := range document.Blocks {
		if block.Kind != "section" {
			continue
		}
		if block.Level == 2 && block.ParentNodeID == "" {
			partCount++
		}
		if block.Level == 3 && block.ParentNodeID != "" {
			nestedCount++
		}
	}
	if partCount != 2 || nestedCount != 6 {
		t.Fatalf("hierarchy = parts %d, sections %d", partCount, nestedCount)
	}
	markdown, _, err := RenderMarkdown(document)
	if err != nil {
		t.Fatal(err)
	}
	if string(markdown) == "" {
		t.Fatal("long-form projection is empty")
	}
}

func TestLongFormAuthoringReceiptPreservesOrderedArtifactChain(t *testing.T) {
	plan := reportilcontract.LongFormPlanReceipt{
		ArtifactID: "art_plan", SHA256: strings.Repeat("a", 64), ByteSize: 101,
	}
	sections := []reportilcontract.AuthorWorkspaceReceipt{
		{ArtifactID: "art_section_001", SHA256: strings.Repeat("b", 64), ByteSize: 201, Stage: "il_long_form_section"},
		{ArtifactID: "art_section_002", SHA256: strings.Repeat("c", 64), ByteSize: 202, Stage: "il_long_form_section"},
	}
	parts := []reportilcontract.AuthorWorkspaceReceipt{
		{ArtifactID: "art_part_001", SHA256: strings.Repeat("d", 64), ByteSize: 301, Stage: "il_long_form_part"},
		{ArtifactID: "art_part_002", SHA256: strings.Repeat("e", 64), ByteSize: 302, Stage: "il_long_form_part"},
	}
	final := reportilcontract.AuthorWorkspaceReceipt{
		ArtifactID: "art_final", SHA256: strings.Repeat("f", 64), ByteSize: 401, Stage: "il_long_form_final",
	}

	receipt := LongFormAuthoringReceipt{
		Plan: longFormPlanArtifactReceipt(plan),
	}
	appendLongFormFinalization(&receipt, "il_long_form_final", final)
	for _, section := range sections {
		receipt.SectionArtifacts = append(receipt.SectionArtifacts, longFormArtifactReceipt(section))
	}
	for _, part := range parts {
		receipt.PartArtifacts = append(receipt.PartArtifacts, longFormArtifactReceipt(part))
	}

	if receipt.Plan != (LongFormArtifactReceipt{ArtifactID: plan.ArtifactID, SHA256: plan.SHA256, ByteSize: plan.ByteSize, Stage: "il_long_form_plan"}) ||
		receipt.Final != (LongFormArtifactReceipt{ArtifactID: final.ArtifactID, SHA256: final.SHA256, ByteSize: final.ByteSize, Stage: final.Stage}) ||
		len(receipt.Finalizations) != 1 || receipt.Finalizations[0] != (LongFormFinalizationReceipt{
		Stage: "il_long_form_final", Artifact: receipt.Final,
	}) {
		t.Fatalf("plan/final artifact lineage = %#v", receipt)
	}
	if got, want := receipt.SectionArtifacts, []LongFormArtifactReceipt{
		{ArtifactID: sections[0].ArtifactID, SHA256: sections[0].SHA256, ByteSize: sections[0].ByteSize, Stage: sections[0].Stage},
		{ArtifactID: sections[1].ArtifactID, SHA256: sections[1].SHA256, ByteSize: sections[1].ByteSize, Stage: sections[1].Stage},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("section artifact lineage = %#v, want %#v", got, want)
	}
	if got, want := receipt.PartArtifacts, []LongFormArtifactReceipt{
		{ArtifactID: parts[0].ArtifactID, SHA256: parts[0].SHA256, ByteSize: parts[0].ByteSize, Stage: parts[0].Stage},
		{ArtifactID: parts[1].ArtifactID, SHA256: parts[1].SHA256, ByteSize: parts[1].ByteSize, Stage: parts[1].Stage},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("part artifact lineage = %#v, want %#v", got, want)
	}
}

func TestLongFormFinalizationRecordsNoOpArtifactReuse(t *testing.T) {
	receipt := LongFormAuthoringReceipt{}
	final := reportilcontract.AuthorWorkspaceReceipt{
		ArtifactID: "art_final", SHA256: strings.Repeat("f", 64),
		ByteSize: 401, Stage: "il_long_form_final",
	}
	reader := final
	reader.Stage = "il_reader"
	continuity := final
	continuity.Stage = "il_continuity"

	appendLongFormFinalization(&receipt, "il_long_form_final", final)
	appendLongFormFinalization(&receipt, "il_reader", reader)
	appendLongFormFinalization(&receipt, "il_continuity", continuity)

	if len(receipt.Finalizations) != 3 ||
		receipt.Finalizations[0].Reused ||
		!receipt.Finalizations[1].Reused ||
		!receipt.Finalizations[2].Reused ||
		receipt.Finalizations[1].Stage != "il_reader" ||
		receipt.Finalizations[2].Stage != "il_continuity" ||
		receipt.Final.Stage != "il_long_form_final" ||
		receipt.Final.ArtifactID != final.ArtifactID {
		t.Fatalf("no-op finalization lineage = %#v", receipt)
	}
}

func validLongFormPlan() reportilcontract.LongFormPlan {
	plan := reportilcontract.LongFormPlan{
		SchemaVersion: reportilcontract.LongFormPlanSchemaVersion,
		Title:         "Report", Language: "en", Summary: "Answer the subject.",
	}
	for partIndex := 0; partIndex < 2; partIndex++ {
		part := reportilcontract.LongFormPart{
			PartKey: "part_00" + string(rune('1'+partIndex)),
			Title:   "Part", Purpose: "Develop the answer.",
		}
		for sectionIndex := 0; sectionIndex < 3; sectionIndex++ {
			role := reportilcontract.LongFormSectionRoleBody
			if partIndex == 1 && sectionIndex == 2 {
				role = reportilcontract.LongFormSectionRoleConclusion
			}
			part.Sections = append(part.Sections, reportilcontract.LongFormSection{
				SectionKey: part.PartKey + ".section_00" + string(rune('1'+sectionIndex)),
				Title:      "Section", Purpose: "Explain one question.", Role: role,
				Representations: []string{}, EvidenceSourceKeys: []string{"source_001"},
			})
		}
		plan.Parts = append(plan.Parts, part)
	}
	return plan
}

func TestValidateLongFormPlanRejectsNullRepresentations(t *testing.T) {
	catalog := testSourceCatalogWithCount(t, "mis_long_form_representations_array", 1)
	plan := validLongFormPlan()
	plan.Parts[len(plan.Parts)-1].Sections[len(plan.Parts[len(plan.Parts)-1].Sections)-1].Representations = nil
	if err := reportilcontract.ValidateLongFormPlan(plan, nil, catalog); err == nil || !strings.Contains(err.Error(), "representations must be an array") {
		t.Fatalf("null representations validation = %v", err)
	}
}

func TestLongFormPromptsBindLanguageConclusionAndFactOwnership(t *testing.T) {
	config := ProductConfig{
		MissionObjective: "다케다성의 역사와 구조를 설명한다.",
		TargetLanguage:   "ko",
	}
	catalog := testSourceCatalogWithCount(t, "mis_long_form_prompt", 6)
	planPrompt := longFormPlanPrompt(config, catalog, reportilcontract.EditorialMemory{}, false)
	for _, expected := range []string{
		`server-owned target language "ko"`,
		`The server assigns role "body"`,
		`role "conclusion" to the final Section`,
		"do not submit a role field",
		"exclusive explanatory ownership",
		"Every essential editorial account must belong to at least one Section",
		"reconcile the complete memory account inventory",
		"submit a representations array",
		`"table" for a genuine comparison`,
		`"code" for executable or illustrative code`,
		`"equation" for a meaningful mathematical relationship`,
		"general cost equation plus a concrete break-even calculation",
		"Prefer executable API, configuration, command, and lifecycle examples",
		"not a quota",
		"do not submit a language field",
	} {
		if !strings.Contains(planPrompt, expected) {
			t.Fatalf("long-form plan prompt missing %q", expected)
		}
	}
	section := reportilcontract.LongFormSection{
		SectionKey: "part_002.section_003", Title: "결론", Purpose: "역사적 판단을 내린다.",
		Role: reportilcontract.LongFormSectionRoleConclusion,
		Representations: []string{
			reportilcontract.LongFormRepresentationTable,
			reportilcontract.LongFormRepresentationEquation,
		},
	}
	sectionPrompt := longFormSectionPrompt(config, reportilcontract.LongFormPlan{Title: "다케다성", Language: "ko"}, reportilcontract.LongFormPart{Title: "의미"}, section, reportilcontract.EditorialMemory{})
	for _, expected := range []string{
		`target language "ko"`, "SECTION ROLE: conclusion", "standalone conclusion", "do not reintroduce facts",
		`PLANNED REPRESENTATIONS: "table, equation"`, "A pair of short prose blocks is not a completion target",
		"use an equation block with a complete LaTeX expression", "Do not add unplanned forms for visual variety",
		"Every bound editorial account must appear in at least one block", "Do not put Markdown emphasis markers",
	} {
		if !strings.Contains(sectionPrompt, expected) {
			t.Fatalf("long-form Section prompt missing %q", expected)
		}
	}
	partPrompt := longFormPartEditPrompt(config, reportilcontract.LongFormPart{Purpose: "이어지는 판단을 발전시킨다."})
	for _, expected := range []string{"Do not replace a transition with a closing paragraph", "introduces no recap", "Never flatten structured explanatory material into generic prose"} {
		if !strings.Contains(partPrompt, expected) {
			t.Fatalf("long-form Part prompt missing %q", expected)
		}
	}
	finalPrompt := longFormFinalEditPrompt(config)
	for _, expected := range []string{`target language "ko"`, "Remove repeated setup", "compare adjacent paragraphs", "shorter summary is still repetition", "final Section is the planned standalone conclusion", "Never flatten structured explanatory material into generic prose"} {
		if !strings.Contains(finalPrompt, expected) {
			t.Fatalf("long-form final prompt missing %q", expected)
		}
	}
}

func TestLongFormArticleContractReusesEveryWritingStage(t *testing.T) {
	config := ProductConfig{
		MissionObjective: "복잡한 기능을 작은 결과부터 만드는 방법을 설명한다.",
		TargetLanguage:   "ko",
		ArticleContract:  "Audience: 제품 엔지니어\nReader promise: 실행 순서를 적용한다\nTarget length: booklet-length",
	}
	catalog := testSourceCatalogWithCount(t, "mis_long_form_article_prompt", 6)
	plan := reportilcontract.LongFormPlan{Title: "작은 결과부터", Language: "ko"}
	part := reportilcontract.LongFormPart{Title: "전환", Purpose: "독자의 이해를 전환한다."}
	section := reportilcontract.LongFormSection{SectionKey: "part_001.section_001", Title: "첫 결과", Purpose: "첫 결과를 설명한다.", Role: reportilcontract.LongFormSectionRoleBody, Representations: []string{}}
	for name, prompt := range map[string]string{
		"plan":       longFormPlanPrompt(config, catalog, reportilcontract.EditorialMemory{}, false),
		"section":    longFormSectionPrompt(config, plan, part, section, reportilcontract.EditorialMemory{}),
		"part":       longFormPartEditPrompt(config, part),
		"final":      longFormFinalEditPrompt(config),
		"reader":     publicationReaderPrompt(config, catalog),
		"continuity": continuityEditorPrompt(config, catalog),
	} {
		for _, expected := range []string{"ARTICLE CONTRACT", "booklet-length", "one continuous reader journey", "not a report"} {
			if !strings.Contains(prompt, expected) {
				t.Fatalf("%s prompt missing Article contract %q", name, expected)
			}
		}
	}
}

func TestPublicationReaderPromptOwnsBrokenNamesAndAdjacentRecaps(t *testing.T) {
	config := ProductConfig{MissionObjective: "일본 다케다성", TargetLanguage: "ko"}
	catalog := testSourceCatalogWithCount(t, "mis_long_form_reader_prompt", 6)
	prompt := publicationReaderPrompt(config, catalog)
	for _, expected := range []string{
		"malformed or incomplete names",
		"omit the name and retain the supported role",
		"Compare adjacent paragraphs",
		"only shortens or restates the preceding paragraph's thesis",
		"checking Korean-script integrity, remaining roadmap language, and adjacent-paragraph repetition explicitly",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("publication reader prompt missing %q: %s", expected, prompt)
		}
	}
	continuityPrompt := continuityEditorPrompt(config, catalog)
	for _, expected := range []string{
		"Never restore an incomplete, mixed-script, placeholder-like, or malformed personal name",
		"preserve the supported role, action, and relationship without the name",
		"Do not reintroduce a reader-facing language defect",
	} {
		if !strings.Contains(continuityPrompt, expected) {
			t.Fatalf("continuity prompt missing %q: %s", expected, continuityPrompt)
		}
	}
}

func TestLongFormSectionAccountRealizationRequiresEveryBoundAccount(t *testing.T) {
	planned := reportilcontract.LongFormSection{
		EditorialAccountKeys: []string{"account_001", "account_002"},
	}
	authored := reportilcontract.AuthorSection{Blocks: []reportilcontract.AuthorBlock{
		{EditorialAccountKeys: []string{"account_001"}},
	}}
	if err := validateLongFormSectionAccountRealization(planned, authored); err == nil || !strings.Contains(err.Error(), "editorial account is missing") {
		t.Fatalf("missing account realization = %v", err)
	}
	authored.Blocks = append(authored.Blocks, reportilcontract.AuthorBlock{EditorialAccountKeys: []string{"account_002"}})
	if err := validateLongFormSectionAccountRealization(planned, authored); err != nil {
		t.Fatal(err)
	}
}

func TestLongFormPlanRequiresEveryEssentialEditorialAccount(t *testing.T) {
	catalog := testSourceCatalogWithCount(t, "mis_long_form_essential_accounts", 1)
	memory := reportilcontract.EditorialMemory{
		SchemaVersion: reportilcontract.EditorialMemorySchemaVersion,
		Language:      "en",
		Accounts: []reportilcontract.EditorialAccount{
			{AccountKey: "account_001", Importance: "essential", Account: "The primary mechanism.", SourceKeys: []string{"source_001"}},
			{AccountKey: "account_002", Importance: "essential", Account: "A distinct operational strategy.", SourceKeys: []string{"source_001"}},
			{AccountKey: "account_003", Importance: "supporting", Account: "Optional context.", SourceKeys: []string{"source_001"}},
		},
	}
	plan := validLongFormPlan()
	for partIndex := range plan.Parts {
		for sectionIndex := range plan.Parts[partIndex].Sections {
			plan.Parts[partIndex].Sections[sectionIndex].EditorialAccountKeys = []string{"account_001"}
		}
	}
	if err := reportilcontract.ValidateLongFormPlan(plan, &memory, catalog); err == nil || !strings.Contains(err.Error(), "omits an essential editorial account") {
		t.Fatalf("omitted essential account validation = %v", err)
	}
	plan.Parts[1].Sections[1].EditorialAccountKeys = []string{"account_002"}
	if err := reportilcontract.ValidateLongFormPlan(plan, &memory, catalog); err != nil {
		t.Fatal(err)
	}
}

func TestLongFormPlanRequiresFinalConclusionRole(t *testing.T) {
	catalog := testSourceCatalogWithCount(t, "mis_long_form_conclusion", 6)
	plan := validLongFormPlan()
	if err := reportilcontract.ValidateLongFormPlan(plan, nil, catalog); err != nil {
		t.Fatal(err)
	}
	missing := plan
	missing.Parts = append([]reportilcontract.LongFormPart(nil), plan.Parts...)
	missing.Parts[1].Sections = append([]reportilcontract.LongFormSection(nil), plan.Parts[1].Sections...)
	missing.Parts[1].Sections[2].Role = reportilcontract.LongFormSectionRoleBody
	if err := reportilcontract.ValidateLongFormPlan(missing, nil, catalog); err == nil {
		t.Fatal("long-form plan without a final conclusion role was accepted")
	}
	misplaced := plan
	misplaced.Parts = append([]reportilcontract.LongFormPart(nil), plan.Parts...)
	misplaced.Parts[0].Sections = append([]reportilcontract.LongFormSection(nil), plan.Parts[0].Sections...)
	misplaced.Parts[0].Sections[0].Role = reportilcontract.LongFormSectionRoleConclusion
	if err := reportilcontract.ValidateLongFormPlan(misplaced, nil, catalog); err == nil {
		t.Fatal("long-form plan with an early conclusion role was accepted")
	}
}

func TestLongFormPlanRejectsUnknownOrDuplicateRepresentations(t *testing.T) {
	catalog := testSourceCatalogWithCount(t, "mis_long_form_representations", 6)
	plan := validLongFormPlan()
	plan.Parts[0].Sections[0].Representations = []string{
		reportilcontract.LongFormRepresentationTable,
		reportilcontract.LongFormRepresentationCode,
		reportilcontract.LongFormRepresentationEquation,
	}
	if err := reportilcontract.ValidateLongFormPlan(plan, nil, catalog); err != nil {
		t.Fatal(err)
	}

	duplicate := validLongFormPlan()
	duplicate.Parts[0].Sections[0].Representations = []string{
		reportilcontract.LongFormRepresentationTable,
		reportilcontract.LongFormRepresentationTable,
	}
	if err := reportilcontract.ValidateLongFormPlan(duplicate, nil, catalog); err == nil {
		t.Fatal("long-form plan with duplicate representations was accepted")
	}

	unknown := validLongFormPlan()
	unknown.Parts[0].Sections[0].Representations = []string{"decorative_chart"}
	if err := reportilcontract.ValidateLongFormPlan(unknown, nil, catalog); err == nil {
		t.Fatal("long-form plan with an unknown representation was accepted")
	}
}

func TestLongFormPlanRequiresClassicInventory(t *testing.T) {
	catalog := testSourceCatalogWithCount(t, "mis_long_form", 6)
	plan := validLongFormPlan()
	if err := reportilcontract.ValidateLongFormPlan(plan, nil, catalog); err != nil {
		t.Fatal(err)
	}
	if len(plan.Parts) != 2 || reportilcontract.LongFormPlanSectionCount(plan) != 6 {
		t.Fatalf("plan inventory = %#v", plan)
	}
	tooShort := plan
	tooShort.Parts = append([]reportilcontract.LongFormPart(nil), plan.Parts...)
	tooShort.Parts[1].Sections = tooShort.Parts[1].Sections[:2]
	if err := reportilcontract.ValidateLongFormPlan(tooShort, nil, catalog); err == nil {
		t.Fatal("five-Section long-form plan was accepted")
	}
}

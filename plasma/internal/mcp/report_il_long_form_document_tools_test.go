package mcp

import (
	"context"
	"encoding/json"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportil"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"strings"
	"testing"
)

func TestReportILAssembleLongFormInputsMergesSectionsInPlanOrder(t *testing.T) {
	plan := testLongFormAssemblyPlan()
	binding := reportilcontract.SourceAccessBinding{
		Stage:           "il_long_form_part",
		LongFormPartKey: "part_001",
	}
	fragments := []reportilcontract.AuthorDocument{
		testLongFormAssemblyFragment("part_001", "part_001.section_001", "Section 1"),
		testLongFormAssemblyFragment("part_001", "part_001.section_002", "Section 2"),
		testLongFormAssemblyFragment("part_001", "part_001.section_003", "Section 3"),
	}
	parts, err := reportil.ReportILAssembleLongFormInputs(binding, plan, fragments)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1 || parts[0].PartKey != "part_001" || len(parts[0].Sections) != 3 {
		t.Fatalf("assembled Part = %#v", parts)
	}
	for index, section := range parts[0].Sections {
		if section.SectionKey != plan.Parts[0].Sections[index].SectionKey {
			t.Fatalf("Section %d = %q", index, section.SectionKey)
		}
	}
}

func TestReportILAssembleLongFormInputsRejectsPartSectionReordering(t *testing.T) {
	plan := testLongFormAssemblyPlan()
	binding := reportilcontract.SourceAccessBinding{
		Stage:           "il_long_form_part",
		LongFormPartKey: "part_001",
	}
	fragments := []reportilcontract.AuthorDocument{
		testLongFormAssemblyFragment("part_001", "part_001.section_002", "Section 2"),
		testLongFormAssemblyFragment("part_001", "part_001.section_001", "Section 1"),
		testLongFormAssemblyFragment("part_001", "part_001.section_003", "Section 3"),
	}
	if _, err := reportil.ReportILAssembleLongFormInputs(binding, plan, fragments); err == nil {
		t.Fatal("reordered Section artifacts were accepted")
	}
}

func TestReportILAssembleLongFormInputsRejectsFinalPartReordering(t *testing.T) {
	plan := testLongFormAssemblyPlan()
	binding := reportilcontract.SourceAccessBinding{Stage: "il_long_form_final"}
	fragments := []reportilcontract.AuthorDocument{
		testLongFormAssemblyPartFragment(plan.Parts[1]),
		testLongFormAssemblyPartFragment(plan.Parts[0]),
	}
	if _, err := reportil.ReportILAssembleLongFormInputs(binding, plan, fragments); err == nil {
		t.Fatal("reordered Part artifacts were accepted")
	}
}

func TestReportILLongFormDocumentStartRequiresCompletePlanRead(t *testing.T) {
	service, binding := testLongFormDocumentServerFixture(t, false)
	server := newLongFormDocumentTestServer(service, binding)
	beforeRead := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentStart,
		Arguments: mustArgs(t, map[string]any{
			"title": "Long report", "language": "en",
		}),
	})
	if beforeRead.Error == nil || !strings.Contains(beforeRead.Error.Message, "complete bound plan read") {
		t.Fatalf("long-form workspace started before plan read: %#v", beforeRead)
	}
	readLongFormPlanCompletely(t, server, 32)
	started := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentStart,
		Arguments: mustArgs(t, map[string]any{
			"title": "Long report", "language": "en",
		}),
	})
	if started.Error != nil {
		t.Fatalf("long-form workspace did not start after plan read: %#v (%s)", started, started.Error.Message)
	}
}

func TestReportILLongFormSectionFinalizeRejectsEmptySection(t *testing.T) {
	service, binding := testLongFormDocumentServerFixture(t, false)
	server := newLongFormDocumentTestServer(service, binding)
	readLongFormPlanCompletely(t, server, reportil.ReportILDocumentMaxReadBytes)
	started := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentStart,
		Arguments: mustArgs(t, map[string]any{
			"title": "Long report", "language": "en",
		}),
	})
	if started.Error != nil {
		t.Fatalf("start empty long-form Section: %#v", started)
	}
	state := started.Content.(reportILDocumentStateOutput)
	readLongFormDocumentCompletely(t, server, state.WorkspaceID)
	finalized := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILLongFormDocumentFinalize,
		Arguments: mustArgs(t, map[string]any{"workspace_id": state.WorkspaceID}),
	})
	if finalized.Error == nil || !strings.Contains(finalized.Error.Message, "empty") {
		t.Fatalf("empty long-form Section finalized: %#v", finalized)
	}
}

func TestReportILLongFormSectionAppendsEquationBlock(t *testing.T) {
	service, binding := testLongFormDocumentServerFixture(t, false)
	server := newLongFormDocumentTestServer(service, binding)
	readLongFormPlanCompletely(t, server, reportil.ReportILDocumentMaxReadBytes)
	started := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentStart,
		Arguments: mustArgs(t, map[string]any{
			"title": "Long report", "language": "en",
		}),
	})
	if started.Error != nil {
		t.Fatalf("start equation Section: %#v", started)
	}
	state := started.Content.(reportILDocumentStateOutput)
	for _, block := range []map[string]any{
		{
			"workspace_id": state.WorkspaceID, "section_key": binding.LongFormSectionKey,
			"kind": "prose", "prose": "The equation separates two priced token classes.",
			"items": []string{}, "code": "", "language": nil, "table": nil, "equation": nil,
			"editorial_account_keys": []string{}, "evidence_source_keys": []string{"source_001"},
		},
		{
			"workspace_id": state.WorkspaceID, "section_key": binding.LongFormSectionKey,
			"kind": "equation", "prose": "", "items": []string{}, "code": "", "language": nil, "table": nil,
			"equation":               map[string]any{"expression": `C = n_i p_i + n_o p_o`, "notation": "latex"},
			"editorial_account_keys": []string{}, "evidence_source_keys": []string{"source_001"},
		},
	} {
		appended := server.Call(context.Background(), ToolCall{
			Name: ToolReportILLongFormDocumentAppend, Arguments: mustArgs(t, block),
		})
		if appended.Error != nil {
			t.Fatalf("append equation Section block: %#v", appended)
		}
	}
	readLongFormDocumentCompletely(t, server, state.WorkspaceID)
	finalized := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILLongFormDocumentFinalize,
		Arguments: mustArgs(t, map[string]any{"workspace_id": state.WorkspaceID}),
	})
	if finalized.Error != nil {
		t.Fatalf("finalize equation Section: %#v", finalized)
	}
	finalState := finalized.Content.(reportILDocumentStateOutput)
	artifactValue, err := service.GetRawArtifact(context.Background(), finalState.ArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	var document reportilcontract.AuthorDocument
	if err := json.Unmarshal(artifactValue.Content, &document); err != nil {
		t.Fatal(err)
	}
	equation := document.Parts[0].Sections[0].Blocks[1].Equation
	if equation == nil || equation.Expression != `C = n_i p_i + n_o p_o` || equation.Notation != "latex" {
		t.Fatalf("stored equation = %#v", equation)
	}
}

func TestReportILLongFormSectionCorrectBlockDeletesScratchTableAndCompactsKeys(t *testing.T) {
	service, binding := testLongFormDocumentServerFixtureWithRepresentations(
		t,
		false,
		[]string{reportilcontract.LongFormRepresentationTable},
	)
	server := newLongFormDocumentTestServer(service, binding)
	readLongFormPlanCompletely(t, server, reportil.ReportILDocumentMaxReadBytes)
	started := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentStart,
		Arguments: mustArgs(t, map[string]any{
			"title": "Long report", "language": "en",
		}),
	})
	if started.Error != nil {
		t.Fatalf("start table Section: %#v", started)
	}
	state := started.Content.(reportILDocumentStateOutput)
	blocks := []map[string]any{
		{
			"workspace_id": state.WorkspaceID, "section_key": binding.LongFormSectionKey,
			"kind": "prose", "prose": "This passage explains the comparison before tabulation.",
			"items": []string{}, "code": "", "language": nil, "table": nil, "equation": nil,
			"editorial_account_keys": []string{}, "evidence_source_keys": []string{"source_001"},
		},
		{
			"workspace_id": state.WorkspaceID, "section_key": binding.LongFormSectionKey,
			"kind": "table", "prose": "", "items": []string{}, "code": "", "language": nil,
			"table": map[string]any{
				"caption": "table-test", "columns": []string{"A", "B"},
				"rows": []map[string]any{{"cells": []string{"x", "y"}}},
			},
			"equation": nil, "editorial_account_keys": []string{}, "evidence_source_keys": []string{"source_001"},
		},
		{
			"workspace_id": state.WorkspaceID, "section_key": binding.LongFormSectionKey,
			"kind": "table", "prose": "", "items": []string{}, "code": "", "language": nil,
			"table": map[string]any{
				"caption": "Useful comparison", "columns": []string{"Choice", "Effect"},
				"rows": []map[string]any{{"cells": []string{"Reuse", "Avoid repeated input"}}},
			},
			"equation": nil, "editorial_account_keys": []string{}, "evidence_source_keys": []string{"source_001"},
		},
	}
	for _, block := range blocks {
		appended := server.Call(context.Background(), ToolCall{
			Name: ToolReportILLongFormDocumentAppend, Arguments: mustArgs(t, block),
		})
		if appended.Error != nil {
			t.Fatalf("append table Section block: %#v", appended)
		}
	}
	beforeRead := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentCorrectBlock,
		Arguments: mustArgs(t, map[string]any{
			"workspace_id": state.WorkspaceID,
			"block_key":    binding.LongFormSectionKey + ".block_002",
			"operation":    "delete", "replacement": nil,
		}),
	})
	if beforeRead.Error == nil || !strings.Contains(beforeRead.Error.Message, "read completely") {
		t.Fatalf("scratch block deleted before review: %#v", beforeRead)
	}
	readLongFormDocumentCompletely(t, server, state.WorkspaceID)
	corrected := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentCorrectBlock,
		Arguments: mustArgs(t, map[string]any{
			"workspace_id": state.WorkspaceID,
			"block_key":    binding.LongFormSectionKey + ".block_002",
			"operation":    "delete", "replacement": nil,
		}),
	})
	if corrected.Error != nil || corrected.Content.(reportILDocumentStateOutput).Replacements != 1 {
		t.Fatalf("delete scratch block: %#v", corrected)
	}
	workspace := server.reportILState.Documents.Workspaces[state.WorkspaceID]
	got := workspace.Document.Parts[0].Sections[0].Blocks
	if len(got) != 2 || got[1].BlockKey != binding.LongFormSectionKey+".block_002" || got[1].Table == nil ||
		got[1].Table.Caption == nil || *got[1].Table.Caption != "Useful comparison" {
		t.Fatalf("corrected block inventory = %#v", got)
	}
	readLongFormDocumentCompletely(t, server, state.WorkspaceID)
	finalized := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILLongFormDocumentFinalize,
		Arguments: mustArgs(t, map[string]any{"workspace_id": state.WorkspaceID}),
	})
	if finalized.Error != nil {
		t.Fatalf("finalize corrected table Section: %#v", finalized)
	}
}

func TestReportILLongFormSectionCorrectBlockReplacesScratchTable(t *testing.T) {
	service, binding := testLongFormDocumentServerFixtureWithRepresentations(
		t,
		false,
		[]string{reportilcontract.LongFormRepresentationTable},
	)
	server := newLongFormDocumentTestServer(service, binding)
	readLongFormPlanCompletely(t, server, reportil.ReportILDocumentMaxReadBytes)
	started := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILLongFormDocumentStart,
		Arguments: mustArgs(t, map[string]any{"title": "Long report", "language": "en"}),
	})
	if started.Error != nil {
		t.Fatalf("start replacement Section: %#v", started)
	}
	state := started.Content.(reportILDocumentStateOutput)
	for _, block := range []map[string]any{
		{
			"workspace_id": state.WorkspaceID, "section_key": binding.LongFormSectionKey,
			"kind": "prose", "prose": "This passage explains the comparison.",
			"items": []string{}, "code": "", "language": nil, "table": nil, "equation": nil,
			"editorial_account_keys": []string{}, "evidence_source_keys": []string{"source_001"},
		},
		{
			"workspace_id": state.WorkspaceID, "section_key": binding.LongFormSectionKey,
			"kind": "table", "prose": "", "items": []string{}, "code": "", "language": nil,
			"table": map[string]any{
				"caption": "table-test", "columns": []string{"A", "B"},
				"rows": []map[string]any{{"cells": []string{"x", "y"}}},
			},
			"equation": nil, "editorial_account_keys": []string{}, "evidence_source_keys": []string{"source_001"},
		},
	} {
		if appended := server.Call(context.Background(), ToolCall{
			Name: ToolReportILLongFormDocumentAppend, Arguments: mustArgs(t, block),
		}); appended.Error != nil {
			t.Fatalf("append replacement fixture: %#v", appended)
		}
	}
	readLongFormDocumentCompletely(t, server, state.WorkspaceID)
	corrected := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentCorrectBlock,
		Arguments: mustArgs(t, map[string]any{
			"workspace_id": state.WorkspaceID,
			"block_key":    binding.LongFormSectionKey + ".block_002",
			"operation":    "replace",
			"replacement": map[string]any{
				"kind": "table", "prose": "", "items": []string{}, "code": "", "language": nil,
				"table": map[string]any{
					"caption": "Useful comparison", "columns": []string{"Choice", "Effect"},
					"rows": []map[string]any{{"cells": []string{"Reuse", "Avoid repeated input"}}},
				},
				"equation": nil, "editorial_account_keys": []string{},
				"evidence_source_keys": []string{"source_001"},
			},
		}),
	})
	if corrected.Error != nil {
		t.Fatalf("replace scratch table: %#v", corrected)
	}
	block := server.reportILState.Documents.Workspaces[state.WorkspaceID].Document.Parts[0].Sections[0].Blocks[1]
	if block.BlockKey != binding.LongFormSectionKey+".block_002" || block.Table == nil ||
		block.Table.Caption == nil || *block.Table.Caption != "Useful comparison" {
		t.Fatalf("replaced block = %#v", block)
	}
	readLongFormDocumentCompletely(t, server, state.WorkspaceID)
	finalized := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILLongFormDocumentFinalize,
		Arguments: mustArgs(t, map[string]any{"workspace_id": state.WorkspaceID}),
	})
	if finalized.Error != nil {
		t.Fatalf("finalize replaced table Section: %#v", finalized)
	}
}

func TestReportILLongFormSectionFinalizeRejectsMissingConcreteRepresentation(t *testing.T) {
	service, binding := testLongFormDocumentServerFixtureWithRepresentations(
		t,
		false,
		[]string{reportilcontract.LongFormRepresentationEquation},
	)
	server := newLongFormDocumentTestServer(service, binding)
	readLongFormPlanCompletely(t, server, reportil.ReportILDocumentMaxReadBytes)
	started := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentStart,
		Arguments: mustArgs(t, map[string]any{
			"title": "Long report", "language": "en",
		}),
	})
	if started.Error != nil {
		t.Fatalf("start planned equation Section: %#v", started)
	}
	state := started.Content.(reportILDocumentStateOutput)
	appended := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentAppend,
		Arguments: mustArgs(t, map[string]any{
			"workspace_id": state.WorkspaceID, "section_key": binding.LongFormSectionKey,
			"kind": "prose", "prose": "This prose names the planned formula without an equation block.",
			"items": []string{}, "code": "", "language": nil, "table": nil, "equation": nil,
			"editorial_account_keys": []string{}, "evidence_source_keys": []string{"source_001"},
		}),
	})
	if appended.Error != nil {
		t.Fatalf("append planned equation prose: %#v", appended)
	}
	readLongFormDocumentCompletely(t, server, state.WorkspaceID)
	finalized := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILLongFormDocumentFinalize,
		Arguments: mustArgs(t, map[string]any{"workspace_id": state.WorkspaceID}),
	})
	if finalized.Error == nil || !strings.Contains(finalized.Error.Message, "planned long-form equation representation is missing") {
		t.Fatalf("missing planned equation was finalized: %#v", finalized)
	}
}

func TestReportILLongFormSectionRejectsOffPlanMemoryAccount(t *testing.T) {
	service, binding := testLongFormDocumentServerFixture(t, true)
	server := newLongFormDocumentTestServer(service, binding)
	readBoundReportILEditorialMemory(t, server)
	readLongFormPlanCompletely(t, server, reportil.ReportILDocumentMaxReadBytes)
	started := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentStart,
		Arguments: mustArgs(t, map[string]any{
			"title": "Long report", "language": "en",
		}),
	})
	if started.Error != nil {
		t.Fatalf("start memory-backed long-form Section: %#v", started)
	}
	state := started.Content.(reportILDocumentStateOutput)
	appended := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentAppend,
		Arguments: mustArgs(t, map[string]any{
			"workspace_id": state.WorkspaceID,
			"section_key":  binding.LongFormSectionKey,
			"kind":         "prose", "prose": "A complete off-plan passage.",
			"editorial_account_keys": []string{"account_002"},
			"evidence_source_keys":   []string{"source_001"},
		}),
	})
	if appended.Error == nil || !strings.Contains(appended.Error.Message, "differs from the bound plan") {
		t.Fatalf("off-plan memory account was accepted: %#v", appended)
	}
}

func TestReportILLongFormFinalizeRejectsPlanTitleMutation(t *testing.T) {
	service, binding := testLongFormDocumentServerFixture(t, false)
	server := newLongFormDocumentTestServer(service, binding)
	readLongFormPlanCompletely(t, server, reportil.ReportILDocumentMaxReadBytes)
	started := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentStart,
		Arguments: mustArgs(t, map[string]any{
			"title": "Long report", "language": "en",
		}),
	})
	if started.Error != nil {
		t.Fatalf("start direct long-form Section: %#v", started)
	}
	state := started.Content.(reportILDocumentStateOutput)
	appended := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentAppend,
		Arguments: mustArgs(t, map[string]any{
			"workspace_id": state.WorkspaceID,
			"section_key":  binding.LongFormSectionKey,
			"kind":         "prose", "prose": "A complete reader-facing passage.",
			"evidence_source_keys": []string{"source_001"},
		}),
	})
	if appended.Error != nil {
		t.Fatalf("append direct long-form Section: %#v", appended)
	}
	replaced := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentReplace,
		Arguments: mustArgs(t, map[string]any{
			"workspace_id": state.WorkspaceID,
			"old_text":     "Long report", "new_text": "Renamed report",
		}),
	})
	if replaced.Error != nil {
		t.Fatalf("replace long-form title: %#v", replaced)
	}
	readLongFormDocumentCompletely(t, server, state.WorkspaceID)
	finalized := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILLongFormDocumentFinalize,
		Arguments: mustArgs(t, map[string]any{"workspace_id": state.WorkspaceID}),
	})
	if finalized.Error == nil || !strings.Contains(finalized.Error.Message, "identity differs from the bound plan") {
		t.Fatalf("renamed long-form title finalized: %#v", finalized)
	}
}

func TestReportILLongFormSectionRejectsUnboundDirectSourceRead(t *testing.T) {
	service, binding := testLongFormDocumentServerFixture(t, false)
	sourceTwo := binding.Catalog.Sources[0]
	sourceTwo.SourceKey = "source_002"
	sourceTwo.AcceptedOrdinal = 2
	sourceTwo.SnapshotID = "src_report_il_002"
	sourceTwo.SnapshotReceipt = reportilcontract.SourceSnapshotReceipt(sourceTwo.SnapshotID, sourceTwo.ContentHash)
	sourceTwo.Artifacts[0].ArtifactID = "art_report_il_002"
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: binding.Catalog.MissionID,
		Sources:   append(binding.Catalog.Sources, sourceTwo),
	})
	if err != nil {
		t.Fatal(err)
	}
	binding.Catalog = catalog
	server := newLongFormDocumentTestServer(service, binding)
	read := server.Call(context.Background(), ToolCall{
		Name: ToolReportILSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"source_key": "source_002", "offset": 0, "max_bytes": reportilcontract.DefaultSourceReadMaxBytes,
		}),
	})
	if read.Error == nil || !strings.Contains(read.Error.Message, "not available") {
		t.Fatalf("unbound direct long-form source was readable: %#v", read)
	}
}

func TestReportILLongFormPlanSubmitOwnsSectionRoles(t *testing.T) {
	service, binding := testLongFormDocumentServerFixture(t, true)
	binding.Stage = "il_long_form_plan"
	binding.LongFormPlanArtifactID = ""
	binding.LongFormPlanSHA256 = ""
	binding.LongFormPartKey = ""
	binding.LongFormSectionKey = ""
	binding.LongFormSectionAccountKeys = nil
	server := newLongFormDocumentTestServer(service, binding)
	readBoundReportILEditorialMemory(t, server)

	plan := testLongFormAssemblyPlan()
	parts := make([]map[string]any, 0, len(plan.Parts))
	for _, part := range plan.Parts {
		sections := make([]map[string]any, 0, len(part.Sections))
		for _, section := range part.Sections {
			sections = append(sections, map[string]any{
				"title": section.Title, "purpose": section.Purpose,
				"representations":        section.Representations,
				"editorial_account_keys": []string{"account_001"},
				"evidence_source_keys":   section.EvidenceSourceKeys,
			})
		}
		parts = append(parts, map[string]any{
			"title": part.Title, "purpose": part.Purpose, "sections": sections,
		})
	}
	submitted := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormPlanSubmit,
		Arguments: mustArgs(t, map[string]any{
			"title": plan.Title, "summary": plan.Summary, "parts": parts,
		}),
	})
	if submitted.Error != nil {
		t.Fatalf("server-owned long-form roles rejected: %#v", submitted)
	}
	state := submitted.Content.(reportILLongFormPlanStateOutput)
	artifactValue, err := service.GetRawArtifact(context.Background(), state.ArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	var stored reportilcontract.LongFormPlan
	if err := json.Unmarshal(artifactValue.Content, &stored); err != nil {
		t.Fatal(err)
	}
	for partIndex, part := range stored.Parts {
		for sectionIndex, section := range part.Sections {
			last := partIndex == len(stored.Parts)-1 && sectionIndex == len(part.Sections)-1
			want := reportilcontract.LongFormSectionRoleBody
			if last {
				want = reportilcontract.LongFormSectionRoleConclusion
			}
			if section.Role != want {
				t.Fatalf("Section %q role = %q, want %q", section.SectionKey, section.Role, want)
			}
			if section.Representations == nil {
				t.Fatalf("Section %q representations serialized as null", section.SectionKey)
			}
		}
	}
}

func TestReportILLongFormPlanSubmitRejectsUnboundCatalogSource(t *testing.T) {
	service, binding := testLongFormDocumentServerFixture(t, false)
	sourceTwo := binding.Catalog.Sources[0]
	sourceTwo.SourceKey = "source_002"
	sourceTwo.AcceptedOrdinal = 2
	sourceTwo.SnapshotID = "src_report_il_002"
	sourceTwo.SnapshotReceipt = reportilcontract.SourceSnapshotReceipt(sourceTwo.SnapshotID, sourceTwo.ContentHash)
	sourceTwo.Artifacts[0].ArtifactID = "art_report_il_002"
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: binding.Catalog.MissionID,
		Sources:   append(binding.Catalog.Sources, sourceTwo),
	})
	if err != nil {
		t.Fatal(err)
	}
	binding.Catalog = catalog
	binding.Stage = "il_long_form_plan"
	binding.LongFormPlanArtifactID = ""
	binding.LongFormPlanSHA256 = ""
	binding.LongFormPartKey = ""
	binding.LongFormSectionKey = ""
	binding.LongFormSourceKeys = []string{"source_001"}
	binding.MaxCallBytes = reportilcontract.DefaultSourceReadMaxBytes
	binding.MaxReadBytes = reportilcontract.DefaultSourceAttemptReadBytes
	server := newLongFormDocumentTestServer(service, binding)
	server.reportILState.Source.Complete["source_001"] = true
	plan := testLongFormAssemblyPlan()
	plan.Parts[0].Sections[0].EvidenceSourceKeys = []string{"source_002"}
	parts := make([]map[string]any, 0, len(plan.Parts))
	for _, part := range plan.Parts {
		sections := make([]map[string]any, 0, len(part.Sections))
		for _, section := range part.Sections {
			sections = append(sections, map[string]any{
				"title": section.Title, "purpose": section.Purpose,
				"editorial_account_keys": []string{},
				"evidence_source_keys":   section.EvidenceSourceKeys,
			})
		}
		parts = append(parts, map[string]any{
			"title": part.Title, "purpose": part.Purpose, "sections": sections,
		})
	}
	submitted := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormPlanSubmit,
		Arguments: mustArgs(t, map[string]any{
			"title": plan.Title, "summary": plan.Summary, "parts": parts,
		}),
	})
	if submitted.Error == nil || !strings.Contains(submitted.Error.Message, "outside the bound inventory") {
		t.Fatalf("unbound catalog source was accepted in direct plan: %#v", submitted)
	}
}

func TestReportILLongFormSectionRejectsPlanLanguageOutsideRequestBinding(t *testing.T) {
	service, binding := testLongFormDocumentServerFixture(t, false)
	binding.LongFormTargetLanguage = "ko"
	server := newLongFormDocumentTestServer(service, binding)
	read := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormPlanRead,
		Arguments: mustArgs(t, map[string]any{
			"offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
		}),
	})
	if read.Error == nil || !strings.Contains(read.Error.Message, "target language differs") {
		t.Fatalf("mismatched plan language was readable: %#v", read)
	}
}

func TestReportILLongFormFinalRejectsSectionArtifactSubstitution(t *testing.T) {
	service, sectionBinding := testLongFormDocumentServerFixture(t, false)
	sectionServer := newLongFormDocumentTestServer(service, sectionBinding)
	readLongFormPlanCompletely(t, sectionServer, reportil.ReportILDocumentMaxReadBytes)
	started := sectionServer.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentStart,
		Arguments: mustArgs(t, map[string]any{
			"title": "Long report", "language": "en",
		}),
	})
	if started.Error != nil {
		t.Fatalf("start Section artifact fixture: %#v", started)
	}
	state := started.Content.(reportILDocumentStateOutput)
	appended := sectionServer.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentAppend,
		Arguments: mustArgs(t, map[string]any{
			"workspace_id": state.WorkspaceID,
			"section_key":  sectionBinding.LongFormSectionKey,
			"kind":         "prose", "prose": "A complete reader-facing passage.",
			"evidence_source_keys": []string{"source_001"},
		}),
	})
	if appended.Error != nil {
		t.Fatalf("append Section artifact fixture: %#v", appended)
	}
	readLongFormDocumentCompletely(t, sectionServer, state.WorkspaceID)
	finalized := sectionServer.Call(context.Background(), ToolCall{
		Name:      ToolReportILLongFormDocumentFinalize,
		Arguments: mustArgs(t, map[string]any{"workspace_id": state.WorkspaceID}),
	})
	if finalized.Error != nil {
		t.Fatalf("finalize Section artifact fixture: %#v", finalized)
	}
	sectionArtifact := finalized.Content.(reportILDocumentStateOutput)

	binding := sectionBinding
	binding.Stage = "il_long_form_final"
	binding.LongFormPartKey = ""
	binding.LongFormSectionKey = ""
	binding.LongFormSourceKeys = nil
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	binding.LongFormInputArtifactIDs = []string{sectionArtifact.ArtifactID}
	binding.LongFormInputSHA256s = []string{sectionArtifact.SHA256}
	server := newLongFormDocumentTestServer(service, binding)
	readLongFormPlanCompletely(t, server, reportil.ReportILDocumentMaxReadBytes)
	opened := server.Call(context.Background(), ToolCall{
		Name: ToolReportILLongFormDocumentStart,
		Arguments: mustArgs(t, map[string]any{
			"title": "Long report", "language": "en",
		}),
	})
	if opened.Error == nil || !strings.Contains(opened.Error.Message, "stage lineage") {
		t.Fatalf("Section artifact was accepted as final Part input: %#v", opened)
	}
}

func testLongFormDocumentServerFixture(t *testing.T, withMemory bool) (*fakeMCPService, reportilcontract.SourceAccessBinding) {
	t.Helper()
	return testLongFormDocumentServerFixtureWithRepresentations(t, withMemory, nil)
}

func testLongFormDocumentServerFixtureWithRepresentations(
	t *testing.T,
	withMemory bool,
	representations []string,
) (*fakeMCPService, reportilcontract.SourceAccessBinding) {
	t.Helper()
	service, binding, sourceText := reportILSourceToolFixture(t, "x", reportilcontract.DefaultSourceReadMaxBytes, reportilcontract.DefaultSourceAttemptReadBytes)
	plan := testLongFormAssemblyPlan()
	plan.Parts[0].Sections[0].Representations = append([]string{}, representations...)
	binding.LongFormTargetLanguage = plan.Language
	if withMemory {
		for partIndex := range plan.Parts {
			for sectionIndex := range plan.Parts[partIndex].Sections {
				plan.Parts[partIndex].Sections[sectionIndex].EditorialAccountKeys = []string{"account_001"}
			}
		}
		memory := reportilcontract.EditorialMemory{
			SchemaVersion: reportilcontract.EditorialMemorySchemaVersion,
			Language:      "en",
			Accounts: []reportilcontract.EditorialAccount{
				{AccountKey: "account_001", Importance: "essential", Account: "Planned account.", SourceKeys: []string{"source_001"}},
				{AccountKey: "account_002", Importance: "supporting", Account: "Off-plan account.", SourceKeys: []string{"source_001"}},
			},
		}
		content := append(mustArgs(t, reportILEditorialMemoryArtifactFixture(memory)), '\n')
		artifact, err := service.CreateRawArtifact(context.Background(), artifactcontract.CreateRequest{
			ArtifactID: "art_editorial_memory", MissionID: binding.Catalog.MissionID,
			MediaType: reportilcontract.EditorialMemoryMediaType, Filename: "report-il-editorial-memory.json",
			Producer: ledger.Producer{Type: "test", ID: "fixture"}, Content: content,
		})
		if err != nil {
			t.Fatal(err)
		}
		binding.EditorialMemoryArtifactID = artifact.ArtifactID
		binding.EditorialMemorySHA256 = artifact.SHA256
		binding.LongFormSectionAccountKeys = []string{"account_001"}
		binding.MaxCallBytes = 0
		binding.MaxReadBytes = 0
	}
	planContent := append(mustArgs(t, plan), '\n')
	planArtifact, err := service.CreateRawArtifact(context.Background(), artifactcontract.CreateRequest{
		ArtifactID: "art_long_form_plan", MissionID: binding.Catalog.MissionID,
		MediaType: reportilcontract.LongFormPlanMediaType, Filename: "report-il-long-form-plan.json",
		Producer: ledger.Producer{Type: "mcp_tool", ID: ToolReportILLongFormPlanSubmit}, Content: planContent,
	})
	if err != nil {
		t.Fatal(err)
	}
	binding.Stage = "il_long_form_section"
	binding.LongFormPlanArtifactID = planArtifact.ArtifactID
	binding.LongFormPlanSHA256 = planArtifact.SHA256
	binding.LongFormPartKey = plan.Parts[0].PartKey
	binding.LongFormSectionKey = plan.Parts[0].Sections[0].SectionKey
	if !withMemory {
		binding.LongFormSourceKeys = []string{"source_001"}
		binding.MaxCallBytes = reportilcontract.DefaultSourceReadMaxBytes
		binding.MaxReadBytes = reportilcontract.DefaultSourceAttemptReadBytes
	}
	service.sources[0].ContentHash = sourcecontract.ContentHash{Algorithm: "sha256", Value: sha256Hex([]byte(sourceText))}
	if err := reportilcontract.ValidateSourceAccessBinding(binding); err != nil {
		t.Fatal(err)
	}
	return service, binding
}

func newLongFormDocumentTestServer(service *fakeMCPService, binding reportilcontract.SourceAccessBinding) *Server {
	server := NewServer(
		service,
		WithBinding(Binding{MissionID: binding.Catalog.MissionID, AgentSessionID: "ses_report_il", AgentExecutor: "codex"}),
		WithReportILSourceBinding(binding),
	)
	if binding.Stage == "il_long_form_section" && binding.MaxReadBytes > 0 {
		server.reportILState.Source.Complete["source_001"] = true
	}
	return server
}

func readLongFormPlanCompletely(t *testing.T, server *Server, maxBytes int) {
	t.Helper()
	offset := 0
	for {
		read := server.Call(context.Background(), ToolCall{
			Name:      ToolReportILLongFormPlanRead,
			Arguments: mustArgs(t, map[string]any{"offset": offset, "max_bytes": maxBytes}),
		})
		if read.Error != nil {
			t.Fatalf("read long-form plan: %#v", read)
		}
		output := read.Content.(reportILLongFormPlanReadOutput)
		if !output.Truncated {
			return
		}
		offset = output.NextOffset
	}
}

func readLongFormDocumentCompletely(t *testing.T, server *Server, workspaceID string) {
	t.Helper()
	offset := 0
	for {
		read := server.Call(context.Background(), ToolCall{
			Name: ToolReportILLongFormDocumentRead,
			Arguments: mustArgs(t, map[string]any{
				"workspace_id": workspaceID, "offset": offset, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
			}),
		})
		if read.Error != nil {
			t.Fatalf("read long-form document: %#v", read)
		}
		var output reportILDocumentReadOutput
		encoded, err := json.Marshal(read.Content)
		if err != nil || json.Unmarshal(encoded, &output) != nil {
			t.Fatalf("decode long-form document read: %v", err)
		}
		if !output.Truncated {
			return
		}
		offset = output.NextOffset
	}
}

func testLongFormAssemblyPlan() reportilcontract.LongFormPlan {
	plan := reportilcontract.LongFormPlan{
		SchemaVersion: reportilcontract.LongFormPlanSchemaVersion,
		Title:         "Long report",
		Language:      "en",
		Summary:       "Develop the answer in ordered Parts.",
	}
	for partIndex := 1; partIndex <= 2; partIndex++ {
		partKey := "part_00" + string(rune('0'+partIndex))
		part := reportilcontract.LongFormPart{
			PartKey: partKey,
			Title:   "Part " + string(rune('0'+partIndex)),
			Purpose: "Develop one stage of the answer.",
		}
		for sectionIndex := 1; sectionIndex <= 3; sectionIndex++ {
			role := reportilcontract.LongFormSectionRoleBody
			if partIndex == 2 && sectionIndex == 3 {
				role = reportilcontract.LongFormSectionRoleConclusion
			}
			part.Sections = append(part.Sections, reportilcontract.LongFormSection{
				SectionKey:         partKey + ".section_00" + string(rune('0'+sectionIndex)),
				Title:              "Section " + string(rune('0'+sectionIndex)),
				Purpose:            "Explain one material question.",
				Role:               role,
				Representations:    []string{},
				EvidenceSourceKeys: []string{"source_001"},
			})
		}
		plan.Parts = append(plan.Parts, part)
	}
	return plan
}

func testLongFormAssemblyFragment(partKey, sectionKey, sectionTitle string) reportilcontract.AuthorDocument {
	return reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.LongFormAuthorDocumentSchemaVersion,
		Title:         "Long report",
		Language:      "en",
		Parts: []reportilcontract.AuthorPart{{
			PartKey: partKey,
			Title:   "Part " + partKey[len(partKey)-1:],
			Sections: []reportilcontract.AuthorSection{{
				SectionKey: sectionKey,
				Title:      sectionTitle,
				Blocks: []reportilcontract.AuthorBlock{{
					BlockKey:           sectionKey + ".block_001",
					Kind:               "prose",
					Prose:              "A complete reader-facing passage.",
					EvidenceSourceKeys: []string{"source_001"},
				}},
			}},
		}},
	}
}

func testLongFormAssemblyPartFragment(part reportilcontract.LongFormPart) reportilcontract.AuthorDocument {
	document := reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.LongFormAuthorDocumentSchemaVersion,
		Title:         "Long report",
		Language:      "en",
		Parts: []reportilcontract.AuthorPart{{
			PartKey: part.PartKey,
			Title:   part.Title,
		}},
	}
	for _, section := range part.Sections {
		fragment := testLongFormAssemblyFragment(part.PartKey, section.SectionKey, section.Title)
		document.Parts[0].Sections = append(document.Parts[0].Sections, fragment.Parts[0].Sections[0])
	}
	return document
}

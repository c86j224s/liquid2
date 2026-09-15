package architecturecheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConversationBoundaryAllowsOnlyAgentExecAndProductPrimitives(t *testing.T) {
	root := moduleRoot(t)
	edges, err := scanGoImports(root)
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{
		moduleImportPath + "/internal/agentcapability": true,
		moduleImportPath + "/internal/agentexec":       true,
		moduleImportPath + "/internal/agentusage":      true,
		moduleImportPath + "/internal/ledger":          true,
		moduleImportPath + "/internal/producterror":    true,
	}
	for _, edge := range edges {
		if !pathWithin(edge.file, "internal/conversation") || !strings.HasPrefix(edge.importPath, moduleImportPath+"/internal/") {
			continue
		}
		if !allowed[edge.importPath] {
			t.Fatalf("conversation import %s from %s is outside its canonical allowlist", edge.importPath, edge.file)
		}
	}
}

func TestReportUsageBoundaryRejectsAppReportingTransportAndSQLite(t *testing.T) {
	for _, importPath := range []string{
		moduleImportPath + "/internal/app",
		moduleImportPath + "/internal/reporting",
		moduleImportPath + "/internal/web",
		moduleImportPath + "/internal/mcp",
		moduleImportPath + "/internal/storage/sqlite",
	} {
		rule, ok := classifyViolation(importEdge{file: "internal/reportusage/record.go", importPath: importPath})
		if !ok || rule != "reportusage-boundary" {
			t.Fatalf("classifyViolation(%q) = %q, %v; want reportusage-boundary", importPath, rule, ok)
		}
	}
}

func TestReportRunBoundaryRejectsAppReportingTransportAndStorage(t *testing.T) {
	for _, importPath := range []string{
		moduleImportPath + "/internal/app",
		moduleImportPath + "/internal/reporting",
		moduleImportPath + "/internal/web",
		moduleImportPath + "/internal/mcp",
		moduleImportPath + "/internal/storage/sqlite",
	} {
		for _, file := range []string{"internal/reportrun/completion.go", "internal/reportrun/completion_test.go"} {
			rule, ok := classifyViolation(importEdge{file: file, importPath: importPath})
			if !ok || rule != "reportrun-boundary" {
				t.Fatalf("classifyViolation(%q from %s) = %q, %v; want reportrun-boundary", importPath, file, rule, ok)
			}
		}
	}
}

func TestReportRunBoundaryAllowsCanonicalCompletionDependencies(t *testing.T) {
	root := moduleRoot(t)
	edges, err := scanGoImports(root)
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{
		moduleImportPath + "/internal/agentusage":       true,
		moduleImportPath + "/internal/ledger":           true,
		moduleImportPath + "/internal/mission":          true,
		moduleImportPath + "/internal/reportusage":      true,
		moduleImportPath + "/internal/reportilcontract": true,
	}
	for _, edge := range edges {
		if !pathWithin(edge.file, "internal/reportrun") || !strings.HasPrefix(edge.importPath, moduleImportPath+"/internal/") {
			continue
		}
		if !allowed[edge.importPath] {
			t.Fatalf("reportrun import %s from %s is outside its canonical allowlist", edge.importPath, edge.file)
		}
	}
}

func TestReportUsageBoundaryAllowsOnlyCanonicalInternalImports(t *testing.T) {
	root := moduleRoot(t)
	edges, err := scanGoImports(root)
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{
		moduleImportPath + "/internal/agentusage":   true,
		moduleImportPath + "/internal/ledger":       true,
		moduleImportPath + "/internal/producterror": true,
	}
	for _, edge := range edges {
		if !pathWithin(edge.file, "internal/reportusage") || !strings.HasPrefix(edge.importPath, moduleImportPath+"/internal/") {
			continue
		}
		if !allowed[edge.importPath] {
			t.Fatalf("reportusage import %s from %s is outside its canonical allowlist", edge.importPath, edge.file)
		}
	}
}

func TestConfluenceSourceBoundaryRejectsNonCanonicalImports(t *testing.T) {
	for _, importPath := range []string{
		moduleImportPath + "/internal/app",
		moduleImportPath + "/internal/source/hidden",
		moduleImportPath + "/internal/connectors/confluence",
		moduleImportPath + "/internal/web",
		moduleImportPath + "/internal/mcp",
	} {
		rule, ok := classifyViolation(importEdge{file: "internal/source/confluencesource/errors.go", importPath: importPath})
		if !ok || rule != "confluence-source-boundary" {
			t.Fatalf("classifyViolation(%q) = %q, %v; want confluence-source-boundary", importPath, rule, ok)
		}
	}
	for _, importPath := range []string{
		moduleImportPath + "/internal/source",
		moduleImportPath + "/internal/producterror",
	} {
		if rule, ok := classifyViolation(importEdge{file: "internal/source/confluencesource/errors.go", importPath: importPath}); ok || rule != "" {
			t.Fatalf("classifyViolation(%q) = %q, %v; want allowed", importPath, rule, ok)
		}
	}
}

func TestSourceParentRejectsConfluenceSourceChild(t *testing.T) {
	rule, ok := classifyViolation(importEdge{
		file:       "internal/source/models.go",
		importPath: moduleImportPath + "/internal/source/confluencesource",
	})
	if !ok || rule != "source-boundary" {
		t.Fatalf("classifyViolation(source child) = %q, %v; want source-boundary", rule, ok)
	}
}

func TestScanGoImportsReadsProductionFilesOnly(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "feature/main.go", "package feature\nimport \"example.com/production\"\n")
	writeTestFile(t, root, "feature/main_test.go", "package feature\nimport \"example.com/test\"\n")
	writeTestFile(t, root, "vendor/example.com/vendored/vendored.go", "package vendored\nimport \"example.com/vendor-dependency\"\n")

	edges, err := scanGoImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 {
		t.Fatalf("edges = %#v, want one production edge", edges)
	}
	if edges[0].file != "feature/main.go" || edges[0].importPath != "example.com/production" {
		t.Fatalf("edge = %#v", edges[0])
	}
}

func TestCompareLinesReportsResolvedAndAddedDebt(t *testing.T) {
	missing, added := compareLines(
		[]string{"kept", "resolved"},
		[]string{"added", "kept"},
	)
	if len(missing) != 1 || missing[0] != "resolved" {
		t.Fatalf("missing = %#v", missing)
	}
	if len(added) != 1 || added[0] != "added" {
		t.Fatalf("added = %#v", added)
	}
}

func TestMissionMCPBoundaryRejectsNonRootInboundAndNonCanonicalOutbound(t *testing.T) {
	for _, file := range []string{"internal/mcp/research/read.go", "cmd/plasma/research.go"} {
		rule, ok := classifyViolation(importEdge{file: file, importPath: moduleImportPath + "/internal/mcp/mission"})
		if !ok || rule != "mission-mcp-inbound" {
			t.Fatalf("classifyViolation(%s) = %q, %v; want mission-mcp-inbound", file, rule, ok)
		}
	}
	for _, importPath := range []string{
		moduleImportPath + "/internal/app",
		moduleImportPath + "/internal/mcp",
		moduleImportPath + "/internal/storage/sqlite",
		moduleImportPath + "/internal/ledger/hidden",
	} {
		rule, ok := classifyViolation(importEdge{file: "internal/mcp/mission/handler.go", importPath: importPath})
		if !ok || rule != "mission-mcp-boundary" {
			t.Fatalf("classifyViolation(%q) = %q, %v; want mission-mcp-boundary", importPath, rule, ok)
		}
	}
	for _, importPath := range []string{
		moduleImportPath + "/internal/mcp/wire",
		moduleImportPath + "/internal/ledger",
		moduleImportPath + "/internal/mission",
		moduleImportPath + "/internal/source",
		moduleImportPath + "/internal/researchrecords",
		moduleImportPath + "/internal/producterror",
		moduleImportPath + "/internal/mcptools",
	} {
		if rule, ok := classifyViolation(importEdge{file: "internal/mcp/mission/handler.go", importPath: importPath}); ok || rule != "" {
			t.Fatalf("classifyViolation(%q) = %q, %v; want allowed", importPath, rule, ok)
		}
	}
}

func TestResearchRecordsBoundaryRejectsInternalChildren(t *testing.T) {
	tests := []struct {
		name string
		path string
		rule string
		want bool
	}{
		{name: "exact app", path: moduleImportPath + "/internal/app", rule: "research-records-boundary", want: true},
		{name: "app child", path: moduleImportPath + "/internal/app/hidden", rule: "research-records-boundary", want: true},
		{name: "transport", path: moduleImportPath + "/internal/mcp", rule: "research-records-boundary", want: true},
		{name: "transport child", path: moduleImportPath + "/internal/mcp/research", rule: "research-records-boundary", want: true},
		{name: "sqlite", path: moduleImportPath + "/internal/storage/sqlite", rule: "research-records-boundary", want: true},
		{name: "sqlite child", path: moduleImportPath + "/internal/storage/sqlite/researchrepo", rule: "research-records-boundary", want: true},
		{name: "ledger exact", path: moduleImportPath + "/internal/ledger", want: false},
		{name: "product error exact", path: moduleImportPath + "/internal/producterror", want: false},
		{name: "source exact", path: moduleImportPath + "/internal/source", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rule, got := classifyViolation(importEdge{file: "internal/researchrecords/builder.go", importPath: test.path})
			if got != test.want || (test.want && rule != test.rule) {
				t.Fatalf("classifyViolation(%q) = (%q, %t), want (%q, %t)", test.path, rule, got, test.rule, test.want)
			}
		})
	}
}

func TestClassifyViolation(t *testing.T) {
	tests := []struct {
		name string
		edge importEdge
		rule string
		want bool
	}{
		{
			name: "article experiment uses product error",
			edge: importEdge{file: "internal/articleexperiment/run.go", importPath: moduleImportPath + "/internal/producterror"},
			want: false,
		},
		{
			name: "article experiment cannot import product error child",
			edge: importEdge{file: "internal/articleexperiment/run.go", importPath: moduleImportPath + "/internal/producterror/hidden"},
			rule: "article-experiment-boundary",
			want: true,
		},
		{
			name: "article experiment cannot import report policy",
			edge: importEdge{file: "internal/articleexperiment/run.go", importPath: moduleImportPath + "/internal/reportilphase0"},
			rule: "article-experiment-boundary",
			want: true,
		},
		{
			name: "article experiment cannot import adapter",
			edge: importEdge{file: "internal/articleexperiment/run.go", importPath: moduleImportPath + "/internal/storage/sqlite"},
			rule: "article-experiment-boundary",
			want: true,
		},
		{
			name: "app hub",
			edge: importEdge{file: "internal/reporting/runner.go", importPath: moduleImportPath + "/internal/app"},
			rule: "app-hub",
			want: true,
		},
		{
			name: "standalone command composition root uses app",
			edge: importEdge{file: "cmd/plasma-report-recover/main.go", importPath: moduleImportPath + "/internal/app"},
			want: false,
		},
		{
			name: "standalone command helper cannot use app",
			edge: importEdge{file: "cmd/plasma-report-recover/run.go", importPath: moduleImportPath + "/internal/app"},
			rule: "app-hub",
			want: true,
		},
		{
			name: "research mcp uses app port",
			edge: importEdge{file: "internal/mcp/research/ports.go", importPath: moduleImportPath + "/internal/app"},
			want: false,
		},
		{
			name: "research mcp cannot import root mcp",
			edge: importEdge{file: "internal/mcp/research/read.go", importPath: moduleImportPath + "/internal/mcp"},
			rule: "research-mcp-boundary",
			want: true,
		},
		{
			name: "research mcp can import mcp wire",
			edge: importEdge{file: "internal/mcp/research/read.go", importPath: moduleImportPath + "/internal/mcp/wire"},
			want: false,
		},
		{
			name: "research mcp cannot import catalog hidden package",
			edge: importEdge{file: "internal/mcp/research/read.go", importPath: moduleImportPath + "/internal/researchcatalog/hidden"},
			rule: "research-mcp-boundary",
			want: true,
		},
		{
			name: "root mcp can import research mcp",
			edge: importEdge{file: "internal/mcp/server.go", importPath: moduleImportPath + "/internal/mcp/research"},
			want: false,
		},
		{
			name: "cmd cannot import research mcp",
			edge: importEdge{file: "cmd/plasma/research.go", importPath: moduleImportPath + "/internal/mcp/research"},
			rule: "research-mcp-inbound",
			want: true,
		},
		{
			name: "other internal cannot import research mcp",
			edge: importEdge{file: "internal/agentexec/research.go", importPath: moduleImportPath + "/internal/mcp/research"},
			rule: "research-mcp-inbound",
			want: true,
		},
		{
			name: "nested mcp subpackage cannot import research mcp",
			edge: importEdge{file: "internal/mcp/wire/models.go", importPath: moduleImportPath + "/internal/mcp/research"},
			rule: "research-mcp-inbound",
			want: true,
		},
		{
			name: "research mcp can import inspection root",
			edge: importEdge{file: "internal/mcp/research/read.go", importPath: moduleImportPath + "/internal/researchinspection"},
			want: false,
		},
		{
			name: "research mcp cannot import inspection child",
			edge: importEdge{file: "internal/mcp/research/read.go", importPath: moduleImportPath + "/internal/researchinspection/hidden"},
			rule: "research-mcp-boundary",
			want: true,
		},
		{
			name: "research mcp can import mcptools",
			edge: importEdge{file: "internal/mcp/research/definitions.go", importPath: moduleImportPath + "/internal/mcptools"},
			want: false,
		},
		{
			name: "research mcp can import researchproposal",
			edge: importEdge{file: "internal/mcp/research/proposal.go", importPath: moduleImportPath + "/internal/researchproposal"},
			want: false,
		},
		{
			name: "research mcp cannot import reporting",
			edge: importEdge{file: "internal/mcp/research/read.go", importPath: moduleImportPath + "/internal/reporting"},
			rule: "research-mcp-boundary",
			want: true,
		},
		{
			name: "research mcp cannot import localpath",
			edge: importEdge{file: "internal/mcp/research/read.go", importPath: moduleImportPath + "/internal/sources/localpath"},
			rule: "research-mcp-boundary",
			want: true,
		},
		{
			name: "research mcp cannot import storage",
			edge: importEdge{file: "internal/mcp/research/read.go", importPath: moduleImportPath + "/internal/storage/sqlite"},
			rule: "research-mcp-boundary",
			want: true,
		},
		{
			name: "research mcp cannot import provider package",
			edge: importEdge{file: "internal/mcp/research/read.go", importPath: moduleImportPath + "/internal/agentexec"},
			rule: "research-mcp-boundary",
			want: true,
		},
		{
			name: "research mcp can import evidence records",
			edge: importEdge{file: "internal/mcp/research/read.go", importPath: moduleImportPath + "/internal/researchrecords"},
			want: false,
		},
		{
			name: "research mcp cannot import evidence records child",
			edge: importEdge{file: "internal/mcp/research/read.go", importPath: moduleImportPath + "/internal/researchrecords/hidden"},
			rule: "research-mcp-boundary",
			want: true,
		},
		{
			name: "mcp wire uses ledger producer contract",
			edge: importEdge{file: "internal/mcp/wire/mutating.go", importPath: moduleImportPath + "/internal/ledger"},
			want: false,
		},
		{
			name: "mcp wire cannot import ledger implementation children",
			edge: importEdge{file: "internal/mcp/wire/mutating.go", importPath: moduleImportPath + "/internal/ledger/hidden"},
			rule: "mcp-wire-boundary",
			want: true,
		},
		{
			name: "mcp wire uses app object refs",
			edge: importEdge{file: "internal/mcp/wire/models.go", importPath: moduleImportPath + "/internal/app"},
			want: false,
		},
		{
			name: "mcp wire cannot import app subpackage",
			edge: importEdge{file: "internal/mcp/wire/models.go", importPath: moduleImportPath + "/internal/app/hidden"},
			rule: "mcp-wire-boundary",
			want: true,
		},
		{
			name: "mcp wire cannot import catalog hidden package",
			edge: importEdge{file: "internal/mcp/wire/models.go", importPath: moduleImportPath + "/internal/researchcatalog/hidden"},
			rule: "mcp-wire-boundary",
			want: true,
		},
		{
			name: "mcp wire cannot import research",
			edge: importEdge{file: "internal/mcp/wire/models.go", importPath: moduleImportPath + "/internal/mcp/research"},
			rule: "research-mcp-inbound",
			want: true,
		},
		{
			name: "mcp wire cannot import mcptools",
			edge: importEdge{file: "internal/mcp/wire/models.go", importPath: moduleImportPath + "/internal/mcptools"},
			rule: "mcp-wire-boundary",
			want: true,
		},
		{
			name: "mcp wire cannot import web",
			edge: importEdge{file: "internal/mcp/wire/models.go", importPath: moduleImportPath + "/internal/web"},
			rule: "mcp-wire-boundary",
			want: true,
		},
		{
			name: "web to mcp",
			edge: importEdge{file: "internal/web/report.go", importPath: moduleImportPath + "/internal/mcp"},
			rule: "transport-sibling",
			want: true,
		},
		{
			name: "mcp to web",
			edge: importEdge{file: "internal/mcp/report.go", importPath: moduleImportPath + "/internal/web"},
			rule: "transport-sibling",
			want: true,
		},
		{
			name: "command reuses web",
			edge: importEdge{file: "cmd/plasma/report.go", importPath: moduleImportPath + "/internal/web"},
			rule: "cmd-web-reuse",
			want: true,
		},
		{
			name: "serve command composes web",
			edge: importEdge{file: "cmd/plasma/serve_command.go", importPath: moduleImportPath + "/internal/web"},
			want: false,
		},
		{
			name: "capability to transport",
			edge: importEdge{file: "internal/workflow/runner.go", importPath: moduleImportPath + "/internal/web"},
			rule: "capability-to-transport",
			want: true,
		},
		{
			name: "transport to adapter",
			edge: importEdge{file: "internal/mcp/source.go", importPath: moduleImportPath + "/internal/sources/urlsource"},
			rule: "transport-to-adapter",
			want: true,
		},
		{
			name: "capability to adapter",
			edge: importEdge{file: "internal/workflow/runner.go", importPath: moduleImportPath + "/internal/storage/sqlite"},
			rule: "capability-to-adapter",
			want: true,
		},
		{
			name: "mission cannot import app hub",
			edge: importEdge{file: "internal/mission/change_contracts.go", importPath: moduleImportPath + "/internal/app"},
			rule: "app-hub",
			want: true,
		},
		{
			name: "mission cannot import web transport",
			edge: importEdge{file: "internal/mission/change_contracts.go", importPath: moduleImportPath + "/internal/web"},
			rule: "capability-to-transport",
			want: true,
		},
		{
			name: "mission cannot import mcp transport",
			edge: importEdge{file: "internal/mission/change_contracts.go", importPath: moduleImportPath + "/internal/mcp"},
			rule: "capability-to-transport",
			want: true,
		},
		{
			name: "mission cannot import sqlite adapter",
			edge: importEdge{file: "internal/mission/change_contracts.go", importPath: moduleImportPath + "/internal/storage/sqlite"},
			rule: "capability-to-adapter",
			want: true,
		},
		{
			name: "mission can import ledger contract",
			edge: importEdge{file: "internal/mission/change_contracts.go", importPath: moduleImportPath + "/internal/ledger"},
			want: false,
		},
		{
			name: "research catalog cannot import app",
			edge: importEdge{file: "internal/researchcatalog/models.go", importPath: moduleImportPath + "/internal/app"},
			rule: "research-catalog-boundary",
			want: true,
		},
		{
			name: "research catalog can import artifact contract",
			edge: importEdge{file: "internal/researchcatalog/summaries.go", importPath: moduleImportPath + "/internal/artifact"},
			want: false,
		},
		{
			name: "research catalog can import ledger contract",
			edge: importEdge{file: "internal/researchcatalog/summaries.go", importPath: moduleImportPath + "/internal/ledger"},
			want: false,
		},
		{
			name: "research catalog can import source contract",
			edge: importEdge{file: "internal/researchcatalog/summaries.go", importPath: moduleImportPath + "/internal/source"},
			want: false,
		},
		{
			name: "research catalog cannot import source adapter child",
			edge: importEdge{file: "internal/researchcatalog/summaries.go", importPath: moduleImportPath + "/internal/source/adapter"},
			rule: "research-catalog-boundary",
			want: true,
		},
		{
			name: "research catalog can import product errors",
			edge: importEdge{file: "internal/researchcatalog/algorithms.go", importPath: moduleImportPath + "/internal/producterror"},
			want: false,
		},
		{
			name: "research proposal can import ledger",
			edge: importEdge{file: "internal/researchproposal/policy.go", importPath: moduleImportPath + "/internal/ledger"},
			want: false,
		},
		{
			name: "research proposal can import catalog",
			edge: importEdge{file: "internal/researchproposal/policy.go", importPath: moduleImportPath + "/internal/researchcatalog"},
			want: false,
		},
		{
			name: "research proposal can import records",
			edge: importEdge{file: "internal/researchproposal/policy.go", importPath: moduleImportPath + "/internal/researchrecords"},
			want: false,
		},
		{
			name: "research proposal can import product errors",
			edge: importEdge{file: "internal/researchproposal/policy.go", importPath: moduleImportPath + "/internal/producterror"},
			want: false,
		},
		{
			name: "research proposal cannot import app",
			edge: importEdge{file: "internal/researchproposal/policy.go", importPath: moduleImportPath + "/internal/app"},
			rule: "research-proposal-boundary",
			want: true,
		},
		{
			name: "research proposal cannot import transport",
			edge: importEdge{file: "internal/researchproposal/policy.go", importPath: moduleImportPath + "/internal/mcp"},
			rule: "research-proposal-boundary",
			want: true,
		},
		{
			name: "research proposal cannot import sql",
			edge: importEdge{file: "internal/researchproposal/policy.go", importPath: moduleImportPath + "/internal/storage/sqlite"},
			rule: "research-proposal-boundary",
			want: true,
		},
		{
			name: "research records can import ledger",
			edge: importEdge{file: "internal/researchrecords/builder.go", importPath: moduleImportPath + "/internal/ledger"},
			want: false,
		},
		{
			name: "research records can import product errors",
			edge: importEdge{file: "internal/researchrecords/builder.go", importPath: moduleImportPath + "/internal/producterror"},
			want: false,
		},
		{
			name: "research records can import source",
			edge: importEdge{file: "internal/researchrecords/models.go", importPath: moduleImportPath + "/internal/source"},
			want: false,
		},
		{
			name: "research records cannot import app",
			edge: importEdge{file: "internal/researchrecords/builder.go", importPath: moduleImportPath + "/internal/app"},
			rule: "research-records-boundary",
			want: true,
		},
		{
			name: "research records cannot import app child",
			edge: importEdge{file: "internal/researchrecords/builder.go", importPath: moduleImportPath + "/internal/app/hidden"},
			rule: "research-records-boundary",
			want: true,
		},
		{
			name: "research records cannot import transport",
			edge: importEdge{file: "internal/researchrecords/builder.go", importPath: moduleImportPath + "/internal/mcp"},
			rule: "research-records-boundary",
			want: true,
		},
		{
			name: "research records cannot import sqlite",
			edge: importEdge{file: "internal/researchrecords/builder.go", importPath: moduleImportPath + "/internal/storage/sqlite"},
			rule: "research-records-boundary",
			want: true,
		},
		{
			name: "mission can import product errors",
			edge: importEdge{file: "internal/mission/change_contracts.go", importPath: moduleImportPath + "/internal/producterror"},
			want: false,
		},
		{
			name: "source creation cannot import app hub",
			edge: importEdge{file: "internal/source/creation_builder.go", importPath: moduleImportPath + "/internal/app"},
			rule: "app-hub",
			want: true,
		},
		{
			name: "source creation cannot import web transport",
			edge: importEdge{file: "internal/source/creation_builder.go", importPath: moduleImportPath + "/internal/web"},
			rule: "capability-to-transport",
			want: true,
		},
		{
			name: "source creation cannot import mcp transport",
			edge: importEdge{file: "internal/source/creation_builder.go", importPath: moduleImportPath + "/internal/mcp"},
			rule: "capability-to-transport",
			want: true,
		},
		{
			name: "source creation cannot import sqlite adapter",
			edge: importEdge{file: "internal/source/creation_builder.go", importPath: moduleImportPath + "/internal/storage/sqlite"},
			rule: "capability-to-adapter",
			want: true,
		},
		{
			name: "source creation cannot import local path adapter",
			edge: importEdge{file: "internal/source/creation_builder.go", importPath: moduleImportPath + "/internal/sources/localpath"},
			rule: "capability-to-adapter",
			want: true,
		},
		{
			name: "source creation can import artifact contract",
			edge: importEdge{file: "internal/source/creation_builder.go", importPath: moduleImportPath + "/internal/artifact"},
			want: false,
		},
		{
			name: "source creation can import ledger contract",
			edge: importEdge{file: "internal/source/creation_contracts.go", importPath: moduleImportPath + "/internal/ledger"},
			want: false,
		},
		{
			name: "source creation can import product errors",
			edge: importEdge{file: "internal/source/creation_policy.go", importPath: moduleImportPath + "/internal/producterror"},
			want: false,
		},
		{
			name: "artifact cannot import app hub",
			edge: importEdge{file: "internal/artifact/build.go", importPath: moduleImportPath + "/internal/app"},
			rule: "app-hub",
			want: true,
		},
		{
			name: "artifact cannot import web transport",
			edge: importEdge{file: "internal/artifact/build.go", importPath: moduleImportPath + "/internal/web"},
			rule: "capability-to-transport",
			want: true,
		},
		{
			name: "artifact cannot import mcp transport",
			edge: importEdge{file: "internal/artifact/build.go", importPath: moduleImportPath + "/internal/mcp"},
			rule: "capability-to-transport",
			want: true,
		},
		{
			name: "artifact cannot import sqlite adapter",
			edge: importEdge{file: "internal/artifact/build.go", importPath: moduleImportPath + "/internal/storage/sqlite"},
			rule: "capability-to-adapter",
			want: true,
		},
		{
			name: "artifact can import ledger contract",
			edge: importEdge{file: "internal/artifact/build.go", importPath: moduleImportPath + "/internal/ledger"},
			want: false,
		},
		{
			name: "artifact can import product errors",
			edge: importEdge{file: "internal/artifact/build.go", importPath: moduleImportPath + "/internal/producterror"},
			want: false,
		},
		{
			name: "adapter owns its implementation",
			edge: importEdge{file: "internal/sources/urlsource/fetch.go", importPath: "net/http"},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rule, got := classifyViolation(test.edge)
			if got != test.want || rule != test.rule {
				t.Fatalf("classifyViolation(%#v) = (%q, %t), want (%q, %t)", test.edge, rule, got, test.rule, test.want)
			}
		})
	}
}

func writeTestFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

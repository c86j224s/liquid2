package architecturecheck

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestClassifyViolation(t *testing.T) {
	tests := []struct {
		name string
		edge importEdge
		rule string
		want bool
	}{
		{
			name: "app hub",
			edge: importEdge{file: "internal/reporting/runner.go", importPath: moduleImportPath + "/internal/app"},
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

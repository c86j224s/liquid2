package architecturecheck

import (
	"flag"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

const moduleImportPath = "github.com/c86j224s/liquid2/plasma"

var updateImportDebt = flag.Bool("update", false, "rewrite the known import-debt baseline")

func TestPackageBoundaries(t *testing.T) {
	root := moduleRoot(t)
	edges, err := scanGoImports(root)
	if err != nil {
		t.Fatal(err)
	}

	actual := make([]string, 0)
	for _, edge := range edges {
		if rule, ok := classifyViolation(edge); ok {
			actual = append(actual, strings.Join([]string{rule, edge.file, edge.importPath}, "\t"))
		}
	}
	sort.Strings(actual)

	baselinePath := filepath.Join(root, "internal", "architecturecheck", "testdata", "known-import-debt.txt")
	if *updateImportDebt {
		if err := writeBaseline(baselinePath, actual); err != nil {
			t.Fatal(err)
		}
	}
	expected, err := readBaseline(baselinePath)
	if err != nil {
		t.Fatal(err)
	}
	missing, added := compareLines(expected, actual)
	if len(missing) == 0 && len(added) == 0 {
		return
	}

	var message strings.Builder
	message.WriteString("package dependency boundary changed\n")
	if len(added) > 0 {
		message.WriteString("new forbidden imports:\n")
		for _, line := range added {
			fmt.Fprintf(&message, "  + %s\n", line)
		}
	}
	if len(missing) > 0 {
		message.WriteString("resolved imports still present in the debt baseline:\n")
		for _, line := range missing {
			fmt.Fprintf(&message, "  - %s\n", line)
		}
	}
	message.WriteString("after an intentional boundary change, run: go test ./internal/architecturecheck -args -update")
	t.Fatal(message.String())
}

func classifyViolation(edge importEdge) (string, bool) {
	appImport := moduleImportPath + "/internal/app"
	webImport := moduleImportPath + "/internal/web"
	mcpImport := moduleImportPath + "/internal/mcp"
	researchMCPImport := moduleImportPath + "/internal/mcp/research"
	wireMCPImport := moduleImportPath + "/internal/mcp/wire"
	internalImport := moduleImportPath + "/internal"
	researchMCPAllowedImports := []string{
		appImport,
		wireMCPImport,
		moduleImportPath + "/internal/mcptools",
		moduleImportPath + "/internal/researchproposal",
	}
	adapterImports := []string{
		moduleImportPath + "/internal/connectors",
		moduleImportPath + "/internal/sources",
		moduleImportPath + "/internal/storage/sqlite",
	}
	sqliteChildRepoImports := []string{
		moduleImportPath + "/internal/storage/sqlite/artifactrepo",
		moduleImportPath + "/internal/storage/sqlite/confluencerepo",
		moduleImportPath + "/internal/storage/sqlite/missionrepo",
		moduleImportPath + "/internal/storage/sqlite/modeldefaultsrepo",
		moduleImportPath + "/internal/storage/sqlite/reportrepo",
		moduleImportPath + "/internal/storage/sqlite/researchrepo",
	}

	switch {
	case importMatchesAny(edge.importPath, sqliteChildRepoImports) && filepath.Dir(edge.file) != "internal/storage/sqlite":
		return "sqlite-child-repo-boundary", true
	case importMatches(edge.importPath, researchMCPImport) && filepath.Dir(edge.file) != "internal/mcp":
		return "research-mcp-inbound", true
	case pathWithin(edge.file, "internal/mcp/research") &&
		importMatches(edge.importPath, internalImport) &&
		!importMatchesAny(edge.importPath, researchMCPAllowedImports):
		return "research-mcp-boundary", true
	case pathWithin(edge.file, "internal/mcp/wire") &&
		importMatches(edge.importPath, internalImport) &&
		edge.importPath != appImport:
		return "mcp-wire-boundary", true
	case importMatches(edge.importPath, appImport) && !pathWithin(edge.file, "internal/app") &&
		!pathWithin(edge.file, "internal/storage/sqlite") &&
		!pathWithinAny(edge.file, []string{"internal/mcp/research", "internal/mcp/wire"}):
		return "app-hub", true
	case pathWithin(edge.file, "internal/web") && importMatches(edge.importPath, mcpImport):
		return "transport-sibling", true
	case pathWithin(edge.file, "internal/mcp") && importMatches(edge.importPath, webImport):
		return "transport-sibling", true
	case pathWithin(edge.file, "cmd/plasma") && importMatches(edge.importPath, webImport) && edge.file != "cmd/plasma/serve_command.go":
		return "cmd-web-reuse", true
	case pathWithin(edge.file, "internal") && !pathWithin(edge.file, "internal/web") && !pathWithin(edge.file, "internal/mcp") &&
		(importMatches(edge.importPath, webImport) || importMatches(edge.importPath, mcpImport)):
		return "capability-to-transport", true
	case (pathWithin(edge.file, "internal/web") || pathWithin(edge.file, "internal/mcp")) && importMatchesAny(edge.importPath, adapterImports):
		return "transport-to-adapter", true
	case pathWithin(edge.file, "internal") && !pathWithinAny(edge.file, []string{
		"internal/architecturecheck",
		"internal/connectors",
		"internal/mcp",
		"internal/sources",
		"internal/storage",
		"internal/web",
	}) && importMatchesAny(edge.importPath, adapterImports):
		return "capability-to-adapter", true
	default:
		return "", false
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate architecture test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func pathWithin(file, dir string) bool {
	return file == dir || strings.HasPrefix(file, strings.TrimSuffix(dir, "/")+"/")
}

func pathWithinAny(file string, dirs []string) bool {
	for _, dir := range dirs {
		if pathWithin(file, dir) {
			return true
		}
	}
	return false
}

func importMatches(importPath, prefix string) bool {
	return importPath == prefix || strings.HasPrefix(importPath, strings.TrimSuffix(prefix, "/")+"/")
}

func importMatchesAny(importPath string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if importMatches(importPath, prefix) {
			return true
		}
	}
	return false
}

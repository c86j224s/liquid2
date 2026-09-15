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
	missionMCPImport := moduleImportPath + "/internal/mcp/mission"
	workflowMCPImport := moduleImportPath + "/internal/mcp/workflow"
	workflowStateImport := moduleImportPath + "/internal/workflowstate"
	wireMCPImport := moduleImportPath + "/internal/mcp/wire"
	researchCatalogImport := moduleImportPath + "/internal/researchcatalog"
	researchRecordsImport := moduleImportPath + "/internal/researchrecords"
	researchInspectionImport := moduleImportPath + "/internal/researchinspection"
	internalImport := moduleImportPath + "/internal"
	articleExperimentProductErrorImport := moduleImportPath + "/internal/producterror"
	productErrorImport := moduleImportPath + "/internal/producterror"
	confluenceSourceImport := moduleImportPath + "/internal/source/confluencesource"
	reportrunAllowedImports := []string{
		moduleImportPath + "/internal/agentusage",
		moduleImportPath + "/internal/ledger",
		moduleImportPath + "/internal/mission",
		moduleImportPath + "/internal/reportusage",
		moduleImportPath + "/internal/reportilcontract",
	}
	reportUsageAllowedImports := []string{
		moduleImportPath + "/internal/agentusage",
		moduleImportPath + "/internal/ledger",
		productErrorImport,
	}
	researchCatalogAllowedImports := []string{
		moduleImportPath + "/internal/artifact",
		moduleImportPath + "/internal/ledger",
		moduleImportPath + "/internal/source",
		moduleImportPath + "/internal/mission",
		moduleImportPath + "/internal/researchrecords",
		productErrorImport,
	}
	researchMCPAllowedImports := []string{
		appImport,
		moduleImportPath + "/internal/source/confluencesource",
		wireMCPImport,
		researchInspectionImport,
		researchCatalogImport,
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
	case pathWithin(edge.file, "internal/reportilphase0") && (importMatches(edge.importPath, moduleImportPath+"/internal/reportilpdf") || importMatches(edge.importPath, moduleImportPath+"/internal/web") || importMatches(edge.importPath, moduleImportPath+"/internal/app") || importMatches(edge.importPath, moduleImportPath+"/internal/mcp") || strings.Contains(edge.importPath, "/chromedp")):
		return "reportilphase0-boundary", true
	case pathWithin(edge.file, "internal/reportilpdf") && (importMatches(edge.importPath, moduleImportPath+"/internal/app") || importMatches(edge.importPath, moduleImportPath+"/internal/web") || importMatches(edge.importPath, moduleImportPath+"/internal/reporting")):
		return "reportilpdf-boundary", true
	case pathWithin(edge.file, "internal/articleexperiment") && importMatches(edge.importPath, internalImport) && edge.importPath != articleExperimentProductErrorImport:
		return "article-experiment-boundary", true
	case pathWithin(edge.file, "internal/reportusage") && importMatches(edge.importPath, internalImport) && !containsExact(reportUsageAllowedImports, edge.importPath):
		return "reportusage-boundary", true
	case pathWithin(edge.file, "internal/reportrun") && importMatches(edge.importPath, internalImport) && !containsExact(reportrunAllowedImports, edge.importPath):
		return "reportrun-boundary", true
	case pathWithin(edge.file, "internal/researchrecords") &&
		importMatches(edge.importPath, internalImport) &&
		!containsExact([]string{
			moduleImportPath + "/internal/ledger",
			moduleImportPath + "/internal/producterror",
			moduleImportPath + "/internal/source",
		}, edge.importPath):
		return "research-records-boundary", true
	case importMatchesAny(edge.importPath, sqliteChildRepoImports) && filepath.Dir(edge.file) != "internal/storage/sqlite":
		return "sqlite-child-repo-boundary", true
	case importMatches(edge.importPath, researchMCPImport) && filepath.Dir(edge.file) != "internal/mcp":
		return "research-mcp-inbound", true
	case importMatches(edge.importPath, missionMCPImport) && filepath.Dir(edge.file) != "internal/mcp":
		return "mission-mcp-inbound", true
	case importMatches(edge.importPath, workflowMCPImport) && filepath.Dir(edge.file) != "internal/mcp":
		return "workflow-mcp-inbound", true
	case pathWithin(edge.file, "internal/mcp/workflow") && importMatches(edge.importPath, internalImport) && !containsExact([]string{
		wireMCPImport, moduleImportPath + "/internal/mcptools", productErrorImport, workflowStateImport,
	}, edge.importPath):
		return "workflow-mcp-boundary", true
	case pathWithin(edge.file, "internal/mcp/mission") && importMatches(edge.importPath, internalImport) && !containsExact([]string{
		wireMCPImport, moduleImportPath + "/internal/ledger", moduleImportPath + "/internal/mission",
		moduleImportPath + "/internal/source", researchRecordsImport, productErrorImport,
		moduleImportPath + "/internal/mcptools",
	}, edge.importPath):
		return "mission-mcp-boundary", true
	case pathWithin(edge.file, "internal/mcp/research") && strings.HasPrefix(edge.importPath, researchCatalogImport+"/"):
		return "research-mcp-boundary", true
	case pathWithin(edge.file, "internal/mcp/research") && strings.HasPrefix(edge.importPath, researchRecordsImport+"/"):
		return "research-mcp-boundary", true
	case pathWithin(edge.file, "internal/mcp/research") && strings.HasPrefix(edge.importPath, researchInspectionImport+"/"):
		return "research-mcp-boundary", true
	case pathWithin(edge.file, "internal/mcp/research") &&
		importMatches(edge.importPath, internalImport) &&
		edge.importPath != moduleImportPath+"/internal/ledger" &&
		edge.importPath != researchRecordsImport &&
		!importMatchesAny(edge.importPath, researchMCPAllowedImports):
		return "research-mcp-boundary", true
	case pathWithin(edge.file, "internal/mcp/wire") && strings.HasPrefix(edge.importPath, researchCatalogImport+"/"):
		return "mcp-wire-boundary", true
	case pathWithin(edge.file, "internal/mcp/wire") &&
		importMatches(edge.importPath, internalImport) &&
		edge.importPath != appImport && edge.importPath != researchCatalogImport &&
		edge.importPath != moduleImportPath+"/internal/ledger":
		return "mcp-wire-boundary", true
	case pathWithin(edge.file, "internal/researchcatalog") &&
		importMatches(edge.importPath, internalImport) && !containsExact(researchCatalogAllowedImports, edge.importPath):
		return "research-catalog-boundary", true
	case pathWithin(edge.file, "internal/researchproposal") &&
		importMatches(edge.importPath, internalImport) && !containsExact([]string{
		moduleImportPath + "/internal/ledger",
		moduleImportPath + "/internal/producterror",
		researchCatalogImport,
		researchRecordsImport,
	}, edge.importPath):
		return "research-proposal-boundary", true
	case pathWithin(edge.file, "internal/source/confluencesource") && importMatches(edge.importPath, internalImport):
		if edge.importPath == productErrorImport || edge.importPath == moduleImportPath+"/internal/source" {
			return "", false
		}
		return "confluence-source-boundary", true
	case pathWithin(edge.file, "internal/source") && importMatches(edge.importPath, confluenceSourceImport):
		return "source-boundary", true
	case importMatches(edge.importPath, appImport) && !pathWithin(edge.file, "internal/app") &&
		!pathWithin(edge.file, "internal/storage/sqlite") &&
		!pathWithinAny(edge.file, []string{"internal/mcp/research", "internal/mcp/wire"}) &&
		!isCommandCompositionRoot(edge.file):
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

func TestReportExecutionRecoveryLineageBoundary(t *testing.T) {
	for _, importPath := range []string{
		moduleImportPath + "/internal/ledger",
		moduleImportPath + "/internal/producterror",
	} {
		if rule, ok := classifyViolation(importEdge{file: "internal/reportexecution/recovery_lineage.go", importPath: importPath}); ok {
			t.Fatalf("classifyViolation(%q) = %q, %v; want allowed", importPath, rule, ok)
		}
	}
	for _, tc := range []struct {
		importPath string
		wantRule   string
	}{
		{importPath: moduleImportPath + "/internal/web", wantRule: "capability-to-transport"},
		{importPath: moduleImportPath + "/internal/app", wantRule: "app-hub"},
	} {
		rule, ok := classifyViolation(importEdge{file: "internal/reportexecution/recovery_lineage.go", importPath: tc.importPath})
		if !ok || rule != tc.wantRule {
			t.Fatalf("classifyViolation(%q) = %q, %v; want %s", tc.importPath, rule, ok, tc.wantRule)
		}
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

func isCommandCompositionRoot(file string) bool {
	return pathWithin(file, "cmd") && filepath.Base(file) == "main.go"
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

func isStandardLibraryImport(importPath string) bool {
	return !strings.Contains(importPath, ".")
}

func containsExact(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

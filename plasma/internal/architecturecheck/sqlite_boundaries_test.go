package architecturecheck

import "testing"

func TestSQLiteChildRepositoriesAreRootOnly(t *testing.T) {
	for _, edge := range []importEdge{
		{
			file:       "internal/web/report_routes.go",
			importPath: moduleImportPath + "/internal/storage/sqlite/reportrepo",
		},
		{
			file:       "internal/storage/sqlite/researchrepo/evidence.go",
			importPath: moduleImportPath + "/internal/storage/sqlite/reportrepo",
		},
	} {
		rule, ok := classifyViolation(edge)
		if !ok || rule != "sqlite-child-repo-boundary" {
			t.Fatalf("classifyViolation(%+v) = %q, %v; want sqlite-child-repo-boundary", edge, rule, ok)
		}
	}
}

func TestSQLiteRootMayImportChildRepositoriesAndAppModels(t *testing.T) {
	for _, edge := range []importEdge{
		{
			file:       "internal/storage/sqlite/store.go",
			importPath: moduleImportPath + "/internal/storage/sqlite/researchrepo",
		},
		{
			file:       "internal/storage/sqlite/atomic_write.go",
			importPath: moduleImportPath + "/internal/storage/sqlite/reportrepo",
		},
		{
			file:       "internal/storage/sqlite/research.go",
			importPath: moduleImportPath + "/internal/app",
		},
		{
			file:       "internal/storage/sqlite/reportrepo/report.go",
			importPath: moduleImportPath + "/internal/app",
		},
	} {
		if rule, ok := classifyViolation(edge); ok {
			t.Fatalf("classifyViolation(%+v) = %q, %v; want allowed", edge, rule, ok)
		}
	}
}

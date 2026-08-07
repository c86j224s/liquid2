package mcpdocs

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var canonicalDocumentFilenames = []string{
	"tools.md",
	"errors.md",
	"reporting.md",
	"sources.md",
	"mermaid.md",
}

func TestListReturnsStableStaticDocumentationResources(t *testing.T) {
	resources := List()
	expected := []string{URITools, URIErrors, URIReporting, URISources, URIMermaid}
	if len(resources) != len(expected) {
		t.Fatalf("unexpected resource count: got %d want %d", len(resources), len(expected))
	}
	for index, resource := range resources {
		if resource.URI != expected[index] {
			t.Fatalf("resource %d URI = %q, want %q", index, resource.URI, expected[index])
		}
		if resource.Name == "" || resource.Description == "" || resource.MIMEType != MIMETypeMarkdown {
			t.Fatalf("resource %q has incomplete metadata: %#v", resource.URI, resource)
		}
	}
}

func TestEmbeddedRuntimeDocumentsAreEnglishCanonicalOnly(t *testing.T) {
	entries, err := resourceFS.ReadDir(".")
	if err != nil {
		t.Fatalf("read embedded runtime documents: %v", err)
	}
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			t.Fatalf("embedded runtime documents must not include directories: %s", entry.Name())
		}
		got = append(got, entry.Name())
	}
	want := append([]string(nil), canonicalDocumentFilenames...)
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("embedded runtime documents changed:\ngot:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestReadReturnsMarkdownForEveryAdvertisedResource(t *testing.T) {
	for _, resource := range List() {
		document, ok := Read(" " + resource.URI + " ")
		if !ok {
			t.Fatalf("Read(%q) failed", resource.URI)
		}
		if document.URI != resource.URI || document.MIMEType != MIMETypeMarkdown {
			t.Fatalf("document metadata changed: %#v", document.Resource)
		}
		if !strings.Contains(document.Text, resource.URI) || !strings.HasPrefix(document.Text, "# ") {
			t.Fatalf("document %q is not recognizable Markdown: %.80q", resource.URI, document.Text)
		}
		assertNoPublicHygieneMarkers(t, resource.URI, document.Text)
	}
}

func TestEmbeddedEnglishCanonicalDocsMatchHumanDocs(t *testing.T) {
	for _, filename := range canonicalDocumentFilenames {
		embedded, err := resourceFS.ReadFile(filename)
		if err != nil {
			t.Fatalf("read embedded runtime document %s: %v", filename, err)
		}
		humanPath := humanDocsPath(t, filename)
		human, err := os.ReadFile(humanPath)
		if err != nil {
			t.Fatalf("read human documentation copy %s: %v", repoRelativeHumanDocsPath(filename), err)
		}
		if !bytes.Equal(embedded, human) {
			t.Fatalf("embedded runtime document %s drifted from human documentation copy %s; update both copies together", filename, repoRelativeHumanDocsPath(filename))
		}
	}
}

func TestReadRejectsUnknownResourceURI(t *testing.T) {
	if _, ok := Read("plasma://docs/mcp/unknown"); ok {
		t.Fatal("unknown URI unexpectedly resolved")
	}
}

func TestKoreanCounterpartsExistOutsideResourceCatalog(t *testing.T) {
	for _, filename := range canonicalDocumentFilenames {
		name := strings.TrimSuffix(filename, ".md")
		koreanFilename := name + ".ko.md"
		content, err := os.ReadFile(humanDocsPath(t, koreanFilename))
		if err != nil {
			t.Fatalf("missing Korean counterpart for %s at %s: %v", filename, repoRelativeHumanDocsPath(koreanFilename), err)
		}
		if !strings.HasPrefix(string(content), "# ") {
			t.Fatalf("Korean counterpart %s is not recognizable Markdown", repoRelativeHumanDocsPath(koreanFilename))
		}
		assertNoPublicHygieneMarkers(t, repoRelativeHumanDocsPath(koreanFilename), string(content))
		if _, err := resourceFS.ReadFile(koreanFilename); err == nil {
			t.Fatalf("Korean counterpart %s was embedded in runtime MCP docs", koreanFilename)
		}
	}
	for _, resource := range List() {
		if strings.Contains(resource.URI, ".ko") {
			t.Fatalf("Korean counterpart leaked into MCP resource catalog: %#v", resource)
		}
	}
}

func humanDocsPath(t *testing.T, filename string) string {
	t.Helper()
	return filepath.Join(humanDocsDirPath(t), filename)
}

func humanDocsDirPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(plasmaRepositoryRoot(t), "docs", "mcp")
}

func plasmaRepositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test working directory: %v", err)
	}
	for {
		if isRegularFile(filepath.Join(dir, "go.mod")) && isDirectory(filepath.Join(dir, "docs", "mcp")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate Plasma repository root from test working directory; expected an ancestor containing go.mod and docs/mcp")
		}
		dir = parent
	}
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func repoRelativeHumanDocsPath(filename string) string {
	return filepath.ToSlash(filepath.Join("plasma", "docs", "mcp", filename))
}

func assertNoPublicHygieneMarkers(t *testing.T, uri string, text string) {
	t.Helper()
	markers := []string{
		"/Users/",
		"local" + "host",
		"127.0.0.1",
		"BEGIN PRIVATE" + " KEY",
		"Author" + "ization:",
		"Cook" + "ie:",
		"sk" + "-",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			t.Fatalf("document %q contains public hygiene marker %q", uri, marker)
		}
	}
}

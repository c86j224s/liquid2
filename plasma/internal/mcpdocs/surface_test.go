package mcpdocs

import (
	"os"
	"sort"
	"strings"
	"testing"
)

var expectedHumanMarkdownFiles = []string{
	"README.md",
	"README.ko.md",
	"tools.md",
	"tools.ko.md",
	"errors.md",
	"errors.ko.md",
	"reporting.md",
	"reporting.ko.md",
	"sources.md",
	"sources.ko.md",
	"mermaid.md",
	"mermaid.ko.md",
}

func TestHumanDocumentationSurfaceHasExpectedMarkdownFiles(t *testing.T) {
	for _, filename := range expectedHumanMarkdownFiles {
		content, err := os.ReadFile(humanDocsPath(t, filename))
		if err != nil {
			t.Fatalf("missing human documentation file %s: %v", repoRelativeHumanDocsPath(filename), err)
		}
		if !strings.HasPrefix(string(content), "# ") {
			t.Fatalf("human documentation file %s is not recognizable Markdown", repoRelativeHumanDocsPath(filename))
		}
		assertNoPublicHygieneMarkers(t, repoRelativeHumanDocsPath(filename), string(content))
	}
}

func TestHumanDocumentationSurfaceHasExactExpectedMarkdownFiles(t *testing.T) {
	entries, err := os.ReadDir(humanDocsDirPath(t))
	if err != nil {
		t.Fatalf("read human documentation directory: %v", err)
	}
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		filename := entry.Name()
		if entry.IsDir() {
			t.Fatalf("human documentation directory must contain files only, found directory %s", repoRelativeHumanDocsPath(filename))
		}
		if !strings.HasSuffix(filename, ".md") {
			t.Fatalf("human documentation directory must contain Markdown files only, found %s", repoRelativeHumanDocsPath(filename))
		}
		got = append(got, filename)
	}
	sort.Strings(got)
	want := append([]string(nil), expectedHumanMarkdownFiles...)
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("human documentation Markdown file set changed:\ngot:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

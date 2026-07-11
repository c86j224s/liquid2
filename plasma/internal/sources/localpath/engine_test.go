package localpath

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestEngineRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "safe.txt"), "safe text")
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "secret.txt"), "secret")
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "escape.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	engine := newTestEngine(t, root)
	for _, rel := range []string{"/etc/passwd", "../safe.txt", `C:\temp\secret.txt`, `\\server\share\secret.txt`, "escape.txt"} {
		if _, err := engine.ReadFile(context.Background(), ReadRequest{RootID: "docs", RelativePath: rel}); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected %q to be rejected, got %v", rel, err)
		}
	}
}

func TestEngineRejectsSymlinkInsideRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "target.txt"), "target text")
	if err := os.Symlink(filepath.Join(root, "target.txt"), filepath.Join(root, "inside.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	engine := newTestEngine(t, root)
	if _, err := engine.ReadFile(context.Background(), ReadRequest{RootID: "docs", RelativePath: "inside.txt"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func TestEngineRejectsSpecialFiles(t *testing.T) {
	root := t.TempDir()
	fifo := filepath.Join(root, "pipe")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	engine := newTestEngine(t, root)
	if _, err := engine.ReadFile(context.Background(), ReadRequest{RootID: "docs", RelativePath: "pipe"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected FIFO rejection, got %v", err)
	}
}

func TestEngineReadTreeGrepAndDenyPatternsDoNotLeakAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "guide.txt"), "alpha\nneedle beta\n")
	writeFile(t, filepath.Join(root, ".env"), "SECRET=1")
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "sub", "note.txt"), "needle in subdir")
	engine := newTestEngine(t, root)
	read, err := engine.ReadFile(context.Background(), ReadRequest{RootID: "docs", RelativePath: "guide.txt", MaxBytes: 8})
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if read.Content != "alpha\nne" || !read.Metadata.Truncated || read.Metadata.SHA256 == "" {
		t.Fatalf("unexpected read result: %#v", read)
	}
	if _, err := engine.ReadFile(context.Background(), ReadRequest{RootID: "docs", RelativePath: ".env"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected deny pattern rejection, got %v", err)
	}
	tree, err := engine.Tree(context.Background(), TreeRequest{RootID: "docs", RelativePath: ".", Depth: 2, Limit: 10})
	if err != nil {
		t.Fatalf("Tree returned error: %v", err)
	}
	if !hasDeniedEntry(tree.Entries, ".env") || !hasTreeEntry(tree.Entries, "sub/note.txt") {
		t.Fatalf("expected denied .env and recursive note entry, got %#v", tree.Entries)
	}
	grep, err := engine.Grep(context.Background(), GrepRequest{RootID: "docs", RelativePath: ".", Query: "needle", MaxSnippets: 2})
	if err != nil {
		t.Fatalf("Grep returned error: %v", err)
	}
	if len(grep.Matches) != 2 {
		t.Fatalf("expected two grep matches, got %#v", grep)
	}
	assertNoAbsolutePath(t, root, read, tree, grep)
}

func TestEngineSubpathScopesOperationsUnderRelativePath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "docs", "nested", "note.txt"), "needle in nested")
	writeFile(t, filepath.Join(root, "docs", ".env"), "SECRET=1")
	engine := newTestEngine(t, root)

	read, err := engine.ReadFile(context.Background(), ReadRequest{RootID: "docs", RelativePath: "docs", Subpath: "nested/note.txt"})
	if err != nil {
		t.Fatalf("ReadFile with subpath returned error: %v", err)
	}
	if read.Content != "needle in nested" || read.Metadata.RelativePath != "docs/nested/note.txt" || read.Metadata.Subpath != "nested/note.txt" {
		t.Fatalf("unexpected subpath read result: %#v", read)
	}
	tree, err := engine.Tree(context.Background(), TreeRequest{RootID: "docs", RelativePath: "docs", Subpath: "nested", Depth: 1, Limit: 10})
	if err != nil {
		t.Fatalf("Tree with subpath returned error: %v", err)
	}
	if tree.Metadata.Subpath != "nested" || len(tree.Entries) != 1 || tree.Entries[0].RelativePath != "docs/nested/note.txt" {
		t.Fatalf("unexpected subpath tree result: %#v", tree)
	}
	grep, err := engine.Grep(context.Background(), GrepRequest{RootID: "docs", RelativePath: "docs", Subpath: "nested", Query: "needle", MaxSnippets: 5})
	if err != nil {
		t.Fatalf("Grep with subpath returned error: %v", err)
	}
	if grep.Metadata.Subpath != "nested" || len(grep.Matches) != 1 || grep.Matches[0].RelativePath != "docs/nested/note.txt" {
		t.Fatalf("unexpected subpath grep result: %#v", grep)
	}
	if _, err := engine.ReadFile(context.Background(), ReadRequest{RootID: "docs", RelativePath: "docs", Subpath: "../guide.txt"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected traversal subpath rejection, got %v", err)
	}
	if _, err := engine.ReadFile(context.Background(), ReadRequest{RootID: "docs", RelativePath: "docs", Subpath: "nested/../note.txt"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected normalized traversal subpath rejection, got %v", err)
	}
	if _, err := engine.ReadFile(context.Background(), ReadRequest{RootID: "docs", RelativePath: "docs", Subpath: "nested/.."}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected parent self subpath rejection, got %v", err)
	}
	if _, err := engine.ReadFile(context.Background(), ReadRequest{RootID: "docs", RelativePath: "docs", Subpath: ".env"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected denied subpath rejection, got %v", err)
	}
	assertNoAbsolutePath(t, root, read, tree, grep)
}

func TestEngineBinaryReadReturnsMetadataWithoutContent(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "data.bin"), []byte{0, 1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	engine := newTestEngine(t, root)
	result, err := engine.ReadFile(context.Background(), ReadRequest{RootID: "docs", RelativePath: "data.bin"})
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !result.Metadata.Binary || result.Content != "" {
		t.Fatalf("expected binary metadata without content, got %#v", result)
	}
	assertNoAbsolutePath(t, root, result)
}

func TestEngineReadPDFTextReturnsExtractedContent(t *testing.T) {
	root := t.TempDir()
	pdfBytes := testPDFBytes(t, []string{"Local PDF Source", "Alpha code is 79."})
	if err := os.WriteFile(filepath.Join(root, "source.pdf"), pdfBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	engine := newTestEngine(t, root)
	result, err := engine.ReadPDFText(context.Background(), ReadRequest{RootID: "docs", RelativePath: "source.pdf", MaxBytes: 20})
	if err != nil {
		t.Fatalf("ReadPDFText returned error: %v", err)
	}
	if !strings.Contains(result.Content, "Local PDF") || strings.Contains(result.Content, "%PDF-") {
		t.Fatalf("expected extracted PDF text, got %#v", result)
	}
	if result.Metadata.Extraction != "pdf_text" || result.Metadata.PageCount != 1 || !result.Metadata.Truncated || result.Metadata.NextOffset == 0 {
		t.Fatalf("unexpected PDF metadata: %#v", result.Metadata)
	}
	if result.Metadata.TextLengthKnown {
		t.Fatalf("expected truncated PDF read to mark text length unknown: %#v", result.Metadata)
	}
	next, err := engine.ReadPDFText(context.Background(), ReadRequest{RootID: "docs", RelativePath: "source.pdf", Offset: result.Metadata.NextOffset, MaxBytes: 50})
	if err != nil {
		t.Fatalf("ReadPDFText continuation returned error: %v", err)
	}
	if next.Metadata.Offset != result.Metadata.NextOffset || strings.TrimSpace(next.Content) == "" {
		t.Fatalf("expected continuation read, got %#v", next)
	}
	assertNoAbsolutePath(t, root, result)
}

func TestEngineDetectsExtensionlessPDFByHeader(t *testing.T) {
	root := t.TempDir()
	pdfBytes := testPDFBytes(t, []string{"Extensionless PDF", "Header detection works."})
	if err := os.WriteFile(filepath.Join(root, "source"), pdfBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	engine := newTestEngine(t, root)
	isPDF, err := engine.IsPDF(context.Background(), "docs", "source")
	if err != nil {
		t.Fatalf("IsPDF returned error: %v", err)
	}
	if !isPDF {
		t.Fatalf("expected extensionless PDF to be detected")
	}
}

func TestEngineIsPDFTreatsEmptyFileAsNonPDF(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "empty"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	engine := newTestEngine(t, root)
	isPDF, err := engine.IsPDF(context.Background(), "docs", "empty")
	if err != nil {
		t.Fatalf("IsPDF returned error for empty file: %v", err)
	}
	if isPDF {
		t.Fatalf("expected empty file not to be detected as PDF")
	}
}

func TestEngineTreeCaps(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		writeFile(t, filepath.Join(root, "file"+string(rune('a'+i))+".txt"), "content")
	}
	engine := newTestEngine(t, root)
	tree, err := engine.Tree(context.Background(), TreeRequest{RootID: "docs", RelativePath: ".", Limit: 2})
	if err != nil {
		t.Fatalf("Tree returned error: %v", err)
	}
	if len(tree.Entries) != 2 || !tree.Truncated {
		t.Fatalf("expected capped tree, got %#v", tree)
	}
}

func testPDFBytes(t *testing.T, lines []string) []byte {
	t.Helper()
	var stream bytes.Buffer
	stream.WriteString("BT\n/F1 12 Tf\n72 720 Td\n")
	for i, line := range lines {
		if i > 0 {
			stream.WriteString("0 -18 Td\n")
		}
		fmt.Fprintf(&stream, "(%s) Tj\n", escapeTestPDFString(line))
	}
	stream.WriteString("ET\n")
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(stream.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d /Filter /FlateDecode >>\nstream\n%s\nendstream", compressed.Len(), compressed.String()),
	}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return out.Bytes()
}

func escapeTestPDFString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `(`, `\(`)
	value = strings.ReplaceAll(value, `)`, `\)`)
	return value
}

func newTestEngine(t *testing.T, root string) *Engine {
	t.Helper()
	engine, err := New(Config{Roots: []RootConfig{{RootID: "docs", Path: root, Alias: "docs"}}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	return engine
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasDeniedEntry(entries []TreeEntry, rel string) bool {
	for _, entry := range entries {
		if entry.RelativePath == rel && entry.Denied {
			return true
		}
	}
	return false
}

func hasTreeEntry(entries []TreeEntry, rel string) bool {
	for _, entry := range entries {
		if entry.RelativePath == rel && !entry.Denied {
			return true
		}
	}
	return false
}

func assertNoAbsolutePath(t *testing.T, root string, values ...any) {
	t.Helper()
	encoded, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if strings.Contains(text, root) || strings.Contains(text, filepath.ToSlash(root)) {
		t.Fatalf("public response leaked absolute root %q: %s", root, text)
	}
}

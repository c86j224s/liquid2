package pdftext

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"strings"
	"testing"
)

func TestExtractCompressedPDFText(t *testing.T) {
	content := testPDF(t, []string{
		"Plasma PDF Source Test",
		"Alpha code is 41.",
		"PDF strategy is source first extraction second.",
	})
	extracted, err := Extract(content)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if extracted.PageCount != 1 {
		t.Fatalf("expected one page, got %d", extracted.PageCount)
	}
	for _, expected := range []string{"Plasma PDF Source Test", "Alpha code is 41", "source first extraction second"} {
		if !strings.Contains(extracted.Text, expected) {
			t.Fatalf("extracted text missing %q: %q", expected, extracted.Text)
		}
	}
	chunk, err := ExtractChunk(content, 0, 32)
	if err != nil {
		t.Fatalf("ExtractChunk returned error: %v", err)
	}
	if !chunk.Truncated || chunk.NextOffset == 0 || chunk.ContentLength < len(chunk.Text) {
		t.Fatalf("expected truncated chunk, got %#v", chunk)
	}
	if chunk.ContentLengthKnown {
		t.Fatalf("expected truncated chunk to have unknown full text length: %#v", chunk)
	}
	next, err := ExtractChunk(content, chunk.NextOffset, 64)
	if err != nil {
		t.Fatalf("ExtractChunk continuation returned error: %v", err)
	}
	if next.Offset != chunk.NextOffset || strings.TrimSpace(next.Text) == "" || strings.Contains(next.Text, "Plasma PDF Source Test") {
		t.Fatalf("expected continuation chunk, got %#v", next)
	}
}

func TestInspectCompressedPDFWithoutExtractingText(t *testing.T) {
	content := testPDF(t, []string{"Metadata only", "No read needed."})
	info, err := Inspect(content)
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if info.PageCount != 1 {
		t.Fatalf("expected one page, got %#v", info)
	}
}

func TestExtractChunkCapsLargeSinglePage(t *testing.T) {
	content := testPDF(t, []string{strings.Repeat("A", MaxChunkBytes*4)})
	chunk, err := ExtractChunk(content, 0, 1024)
	if err != nil {
		t.Fatalf("ExtractChunk returned error: %v", err)
	}
	if len(chunk.Text) > 1024 {
		t.Fatalf("expected chunk cap to be enforced, got %d bytes", len(chunk.Text))
	}
	if !chunk.Truncated || chunk.NextOffset != len(chunk.Text) || chunk.ContentLengthKnown {
		t.Fatalf("unexpected chunk metadata for capped page: %#v", chunk)
	}
}

func TestExtractPreservesQuoteOperatorLineAdvance(t *testing.T) {
	content := testPDFStream(t, "BT\n/F1 12 Tf\n72 720 Td\n(First) Tj\n(Second) '\n0 0 (Third) \"\nET\n")
	extracted, err := Extract(content)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	for _, expected := range []string{"First\nSecond", "Second\nThird"} {
		if !strings.Contains(extracted.Text, expected) {
			t.Fatalf("expected line advance %q in extracted text: %q", expected, extracted.Text)
		}
	}
	for _, unexpected := range []string{"FirstSecond", "SecondThird"} {
		if strings.Contains(extracted.Text, unexpected) {
			t.Fatalf("unexpected joined text %q in extracted text: %q", unexpected, extracted.Text)
		}
	}
}

func testPDF(t *testing.T, lines []string) []byte {
	t.Helper()
	var stream bytes.Buffer
	stream.WriteString("BT\n/F1 12 Tf\n72 720 Td\n")
	for i, line := range lines {
		if i > 0 {
			stream.WriteString("0 -18 Td\n")
		}
		fmt.Fprintf(&stream, "(%s) Tj\n", escapePDFString(line))
	}
	stream.WriteString("ET\n")
	return testPDFStream(t, stream.String())
}

func testPDFStream(t *testing.T, streamContent string) []byte {
	t.Helper()
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write([]byte(streamContent)); err != nil {
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

func escapePDFString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `(`, `\(`)
	value = strings.ReplaceAll(value, `)`, `\)`)
	return value
}

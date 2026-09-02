package sourceretrieval

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchWithClientAcceptsPDFSource(t *testing.T) {
	pdfBytes := testPDFBytes(t, []string{
		"Staged PDF Candidate",
		"Alpha code is 91.",
	})
	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdfBytes)
	}))
	defer sourceServer.Close()

	fetched, err := FetchWithClient(context.Background(), sourceServer.URL+"/candidate.pdf", sourceServer.Client())
	if err != nil {
		t.Fatalf("FetchWithClient returned error: %v", err)
	}
	if fetched.MediaType != "application/pdf" || fetched.PageCount != 1 || fetched.ByteSize != int64(len(pdfBytes)) {
		t.Fatalf("expected inspected PDF metadata, got %#v", fetched)
	}
	if !strings.HasSuffix(fetched.Title, "candidate.pdf") {
		t.Fatalf("expected PDF title from URL, got %q", fetched.Title)
	}
}

func TestFetchWithClientSniffsOctetStreamImage(t *testing.T) {
	imageBytes := testPNGBytes()
	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(imageBytes)
	}))
	defer sourceServer.Close()

	fetched, err := FetchWithClient(context.Background(), sourceServer.URL+"/viewimage.php?id=1", sourceServer.Client())
	if err != nil {
		t.Fatalf("FetchWithClient returned error: %v", err)
	}
	if fetched.MediaType != "image/png" || fetched.MediaKind != MediaKindImage || fetched.Width != 1 || fetched.Height != 1 || fetched.ByteSize != int64(len(imageBytes)) {
		t.Fatalf("expected sniffed image metadata, got %#v", fetched)
	}
}

type testRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn testRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestFetchWithClientLimitsOctetStreamBeforeReadingImageBody(t *testing.T) {
	called := false
	client := &http.Client{Transport: testRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		called = true
		header := make(http.Header)
		header.Set("Content-Type", "application/octet-stream")
		return &http.Response{
			StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader("not read")),
			ContentLength: MaxImageSourceBytes + 1, Request: req,
		}, nil
	})}
	_, err := FetchWithClient(context.Background(), "https://example.com/viewimage.php?id=1", client)
	if !called || err == nil || !strings.Contains(err.Error(), "larger than 10 MiB") {
		t.Fatalf("octet-stream size guard: called=%v err=%v", called, err)
	}
}

func testPNGBytes() []byte {
	return []byte{
		0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n',
		0, 0, 0, 13, 'I', 'H', 'D', 'R',
		0, 0, 0, 1, 0, 0, 0, 1, 8, 2, 0, 0, 0,
		0x90, 0x77, 0x53, 0xde,
		0, 0, 0, 12, 'I', 'D', 'A', 'T',
		8, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0, 0, 3, 1, 1, 0,
		0x18, 0xdd, 0x8d, 0xb0,
		0, 0, 0, 0, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82,
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

package pdfdocument

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"strings"
	"unicode"
	"unicode/utf8"

	pdf "github.com/ledongthuc/pdf"
)

const (
	// MediaType은 PDF source를 식별할 때 쓰는 canonical media type이다.
	MediaType             = "application/pdf"
	DefaultChunkMaxBytes  = 20000
	MaxChunkBytes         = 50000
	MaxExtractedTextBytes = 100 << 20
)

// Extracted는 PDF 전체 텍스트 추출 결과다.
//
// Truncated가 true면 Text는 앞부분 일부이고 TextLengthKnown은 false다. 이 경우
// caller는 chunk API로 필요한 범위만 다시 읽어야 한다.
type Extracted struct {
	Text            string
	PageCount       int
	Truncated       bool
	TextLengthKnown bool
}

// Chunk는 PDF 추출 텍스트의 bounded slice다.
//
// NextOffset이 0이면 더 읽을 chunk가 없다는 뜻이다.
type Chunk struct {
	Text               string
	Offset             int
	NextOffset         int
	ContentLength      int
	ContentLengthKnown bool
	Truncated          bool
	PageCount          int
}

// DocumentInfo는 PDF를 열어 확인한 가벼운 metadata다.
type DocumentInfo struct {
	PageCount int
}

// IsPDFMediaType은 media type 문자열이 PDF를 나타내는지 판정한다.
func IsPDFMediaType(mediaType string) bool {
	base, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		base = mediaType
	}
	return strings.EqualFold(strings.TrimSpace(base), MediaType)
}

// IsPDFBytes는 content prefix로 PDF 파일 여부를 빠르게 판정한다.
func IsPDFBytes(content []byte) bool {
	return bytes.HasPrefix(bytes.TrimLeft(content, "\xef\xbb\xbf\t\r\n "), []byte("%PDF-"))
}

// Extract는 메모리의 PDF 바이트 전체에서 정규화된 텍스트를 추출한다.
func Extract(content []byte) (Extracted, error) {
	return ExtractReaderAt(bytes.NewReader(content), int64(len(content)))
}

// ExtractReaderAt은 ReaderAt 기반 PDF source에서 정규화된 텍스트를 추출한다.
func ExtractReaderAt(source io.ReaderAt, size int64) (Extracted, error) {
	reader, err := openReaderAt(source, size)
	if err != nil {
		return Extracted{}, err
	}
	pageCount := reader.NumPage()
	collector := newNormalizedCollector(0, MaxExtractedTextBytes)
	if err := collectPDFText(reader, collector); err != nil {
		return Extracted{}, err
	}
	return Extracted{
		Text:            collector.String(),
		PageCount:       pageCount,
		Truncated:       collector.Truncated(),
		TextLengthKnown: !collector.Truncated(),
	}, nil
}

// Inspect는 메모리의 PDF 바이트에서 page count 같은 가벼운 정보만 읽는다.
func Inspect(content []byte) (DocumentInfo, error) {
	return InspectReaderAt(bytes.NewReader(content), int64(len(content)))
}

// InspectReaderAt은 ReaderAt 기반 PDF source에서 가벼운 문서 정보만 읽는다.
func InspectReaderAt(source io.ReaderAt, size int64) (DocumentInfo, error) {
	reader, err := openReaderAt(source, size)
	if err != nil {
		return DocumentInfo{}, err
	}
	return DocumentInfo{PageCount: reader.NumPage()}, nil
}

// ExtractChunk는 메모리의 PDF 바이트에서 offset/maxBytes 범위의 텍스트 chunk를 읽는다.
func ExtractChunk(content []byte, offset int, maxBytes int) (Chunk, error) {
	return ExtractChunkFromReaderAt(bytes.NewReader(content), int64(len(content)), offset, maxBytes)
}

// ExtractChunkFromReaderAt은 PDF 전체를 prompt에 싣지 않도록 bounded chunk만 추출한다.
func ExtractChunkFromReaderAt(source io.ReaderAt, size int64, offset int, maxBytes int) (Chunk, error) {
	if offset < 0 {
		return Chunk{}, fmt.Errorf("offset must be non-negative")
	}
	reader, err := openReaderAt(source, size)
	if err != nil {
		return Chunk{}, err
	}
	limit := normalizedMaxBytes(maxBytes)
	pageCount := reader.NumPage()
	collector := newNormalizedCollector(offset, limit)
	if err := collectPDFText(reader, collector); err != nil {
		return Chunk{}, err
	}
	if collector.SkipRemaining() > 0 {
		return Chunk{}, fmt.Errorf("offset is beyond content length")
	}
	text := collector.String()
	nextOffset := 0
	contentLengthKnown := !collector.Truncated()
	contentLength := collector.Total()
	if collector.Truncated() {
		nextOffset = offset + len(text)
		contentLength = offset + len(text)
	}
	return Chunk{
		Text:               text,
		Offset:             offset,
		NextOffset:         nextOffset,
		ContentLength:      contentLength,
		ContentLengthKnown: contentLengthKnown,
		Truncated:          collector.Truncated(),
		PageCount:          pageCount,
	}, nil
}

func openReaderAt(source io.ReaderAt, size int64) (*pdf.Reader, error) {
	if size <= 0 {
		return nil, fmt.Errorf("pdf content is empty")
	}
	if !isPDFReaderAt(source) {
		return nil, fmt.Errorf("content is not a PDF")
	}
	return pdf.NewReader(source, size)
}

func isPDFReaderAt(source io.ReaderAt) bool {
	var header [512]byte
	n, err := source.ReadAt(header[:], 0)
	if err != nil && n == 0 {
		return false
	}
	return IsPDFBytes(header[:n])
}

func collectPDFText(reader *pdf.Reader, collector *normalizedCollector) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if _, ok := recovered.(extractionStop); ok {
				err = collector.Err()
				return
			}
			err = fmt.Errorf("%v", recovered)
		}
	}()
	fonts := map[string]*pdf.Font{}
	for pageNumber := 1; pageNumber <= reader.NumPage(); pageNumber++ {
		collector.WriteRaw("\n")
		if err := collectPageText(reader, pageNumber, fonts, collector); err != nil {
			return err
		}
	}
	return collector.Err()
}

func collectPageText(reader *pdf.Reader, pageNumber int, fonts map[string]*pdf.Font, collector *normalizedCollector) error {
	page := reader.Page(pageNumber)
	if page.V.IsNull() || page.V.Key("Contents").Kind() == pdf.Null {
		return nil
	}
	for _, name := range page.Fonts() {
		if _, ok := fonts[name]; ok {
			continue
		}
		font := page.Font(name)
		fonts[name] = &font
	}
	var enc pdf.TextEncoding = rawTextEncoder{}
	pdf.Interpret(page.V.Key("Contents"), func(stk *pdf.Stack, op string) {
		n := stk.Len()
		args := make([]pdf.Value, n)
		for i := n - 1; i >= 0; i-- {
			args[i] = stk.Pop()
		}
		switch op {
		case "BT":
			collector.WriteRaw("\n")
		case "T*":
			collector.WriteRaw("\n")
		case "Tf":
			if len(args) != 2 {
				panic("bad Tf operator")
			}
			if font, ok := fonts[args[0].Name()]; ok {
				enc = font.Encoder()
			} else {
				enc = rawTextEncoder{}
			}
		case "\"":
			if len(args) != 3 {
				panic(`bad " operator`)
			}
			collector.WriteRaw("\n")
			collector.WriteRaw(enc.Decode(args[2].RawString()))
		case "'":
			if len(args) != 1 {
				panic("bad ' operator")
			}
			collector.WriteRaw("\n")
			collector.WriteRaw(enc.Decode(args[0].RawString()))
		case "Tj":
			if len(args) != 1 {
				panic("bad Tj operator")
			}
			collector.WriteRaw(enc.Decode(args[0].RawString()))
		case "TJ":
			if len(args) != 1 {
				panic("bad TJ operator")
			}
			value := args[0]
			for i := 0; i < value.Len(); i++ {
				item := value.Index(i)
				if item.Kind() == pdf.String {
					collector.WriteRaw(enc.Decode(item.RawString()))
				}
			}
		}
	})
	return collector.Err()
}

type rawTextEncoder struct{}

// Decode는 PDF string의 raw text를 변환 없이 반환한다.
func (rawTextEncoder) Decode(raw string) string {
	return raw
}

type extractionStop struct{}

type normalizedCollector struct {
	skipRemaining int
	limit         int
	total         int
	out           strings.Builder
	truncated     bool
	err           error
	pendingLine   bool
	lineHasText   bool
	pendingSpace  bool
}

func newNormalizedCollector(offset int, limit int) *normalizedCollector {
	return &normalizedCollector{skipRemaining: offset, limit: limit}
}

// String은 값의 사람이 읽을 수 있는 표현을 반환하며, 저장 식별자나 민감한 원문을 새로 만들지 않는다.
func (collector *normalizedCollector) String() string {
	return collector.out.String()
}

// Total은 offset 적용 전까지 포함한 전체 추출 문자 수를 반환한다.
func (collector *normalizedCollector) Total() int {
	return collector.total
}

// Truncated는 limit 때문에 출력이 잘렸는지 반환한다.
func (collector *normalizedCollector) Truncated() bool {
	return collector.truncated
}

// SkipRemaining은 아직 건너뛰어야 하는 offset 문자 수를 반환한다.
func (collector *normalizedCollector) SkipRemaining() int {
	return collector.skipRemaining
}

// Err는 추출 중단을 유발한 오류를 반환한다.
func (collector *normalizedCollector) Err() error {
	return collector.err
}

// WriteRaw는 PDF string fragment를 공백과 줄바꿈이 정규화된 텍스트로 누적한다.
func (collector *normalizedCollector) WriteRaw(text string) {
	for _, r := range text {
		switch {
		case r == '\r' || r == '\n':
			if collector.lineHasText {
				collector.pendingLine = true
			}
			collector.lineHasText = false
			collector.pendingSpace = false
		case unicode.IsSpace(r):
			if collector.lineHasText {
				collector.pendingSpace = true
			}
		default:
			if collector.pendingLine {
				collector.emit("\n")
				collector.pendingLine = false
			}
			if collector.pendingSpace && collector.lineHasText {
				collector.emit(" ")
				collector.pendingSpace = false
			}
			collector.emit(string(r))
			collector.lineHasText = true
		}
		if collector.err != nil || collector.truncated {
			panic(extractionStop{})
		}
	}
}

func (collector *normalizedCollector) emit(text string) {
	if collector.skipRemaining > 0 {
		if collector.skipRemaining >= len(text) {
			collector.skipRemaining -= len(text)
			collector.total += len(text)
			return
		}
		collector.err = fmt.Errorf("offset must align to UTF-8 boundary")
		panic(extractionStop{})
	}
	if collector.limit <= 0 {
		collector.truncated = true
		panic(extractionStop{})
	}
	if collector.out.Len()+len(text) > collector.limit {
		collector.truncated = true
		panic(extractionStop{})
	}
	collector.out.WriteString(text)
	collector.total += len(text)
}

func normalizeExtractedText(text string) string {
	var builder strings.Builder
	for len(text) > 0 {
		line, rest := nextLine(text)
		normalized := strings.Join(strings.Fields(line), " ")
		if normalized != "" {
			if builder.Len() > 0 {
				builder.WriteByte('\n')
			}
			builder.WriteString(normalized)
		}
		text = rest
	}
	return builder.String()
}

func nextLine(text string) (string, string) {
	for i, r := range text {
		if r == '\n' {
			line := strings.TrimSuffix(text[:i], "\r")
			return line, text[i+1:]
		}
		if r == '\r' {
			rest := text[i+1:]
			if strings.HasPrefix(rest, "\n") {
				rest = rest[1:]
			}
			return text[:i], rest
		}
	}
	return text, ""
}

func normalizedMaxBytes(maxBytes int) int {
	if maxBytes <= 0 {
		return DefaultChunkMaxBytes
	}
	if maxBytes > MaxChunkBytes {
		return MaxChunkBytes
	}
	return maxBytes
}

func validUTF8Prefix(text string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if maxBytes >= len(text) {
		return text
	}
	cut := maxBytes
	for cut > 0 && !utf8.ValidString(text[:cut]) {
		cut--
	}
	return text[:cut]
}

func boundedUTF8(text string, offset int, maxBytes int) (string, int, int, bool, error) {
	if offset < 0 {
		return "", 0, 0, false, fmt.Errorf("offset must be non-negative")
	}
	if offset > len(text) {
		return "", 0, 0, false, fmt.Errorf("offset is beyond content length")
	}
	if offset < len(text) && !utf8.RuneStart(text[offset]) {
		return "", 0, 0, false, fmt.Errorf("offset must align to UTF-8 boundary")
	}
	limit := maxBytes
	if limit <= 0 {
		limit = DefaultChunkMaxBytes
	} else if limit > MaxChunkBytes {
		limit = MaxChunkBytes
	}
	remaining := text[offset:]
	if len(remaining) <= limit {
		return remaining, offset, 0, false, nil
	}
	cut := offset + limit
	for cut > offset && !utf8.ValidString(text[offset:cut]) {
		cut--
	}
	if cut == offset {
		return "", 0, 0, false, fmt.Errorf("content could not be sliced as UTF-8")
	}
	return text[offset:cut], offset, cut, true, nil
}

package reportilphase0

import "context"

// PDFResult는 소비자가 보존하는 PDF 바이트와 렌더러의 실행 identity다.
type PDFResult struct {
	Content          []byte
	RendererProduct  string
	RendererRevision string
}

// PDFRenderer는 완성된 self-contained HTML을 PDF로 변환하는 소비자 측 포트다.
type PDFRenderer interface {
	RenderPDF(context.Context, []byte) (PDFResult, error)
}

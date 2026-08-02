package web

import (
	"context"
	"errors"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/sources/browserrender"
)

const sourceRetrievalMethodBrowserRender = "browser_render"

var defaultBrowserURLRenderer = browserrender.NewRenderer(browserrender.Options{})

func renderBrowserURLSource(ctx context.Context, rawURL string) (fetchedURLSource, error) {
	result, err := defaultBrowserURLRenderer.Render(ctx, rawURL)
	if err != nil {
		return fetchedURLSource{}, browserRenderSourceError(err)
	}
	return fetchedURLSource{
		Content:         result.Content,
		MediaType:       result.MediaType,
		Title:           result.Title,
		ByteSize:        int64(len(result.Content)),
		TextLength:      result.TextLength,
		TextLengthKnown: true,
		RetrievalMethod: sourceRetrievalMethodBrowserRender,
		FinalURL:        result.FinalURL,
		RenderedAt:      result.RenderedAt,
	}, nil
}

func browserRenderSourceError(err error) error {
	switch {
	case errors.Is(err, browserrender.ErrBlockedURL):
		return fmt.Errorf("%w: 브라우저 렌더링 중 차단된 주소 요청이 감지되어 URL 소스를 만들지 않았습니다.", app.ErrInvalidInput)
	case errors.Is(err, browserrender.ErrNoReadableBody):
		return fmt.Errorf("%w: 브라우저 렌더링으로도 읽을 수 있는 본문을 확인하지 못했습니다.", app.ErrInvalidInput)
	case errors.Is(err, browserrender.ErrRenderTimedOut):
		return fmt.Errorf("%w: 브라우저 렌더링이 제한 시간 내 완료되지 않았습니다.", app.ErrInvalidInput)
	case errors.Is(err, browserrender.ErrRenderUnavailable):
		return fmt.Errorf("%w: 브라우저 렌더링 실행 환경을 사용할 수 없습니다.", app.ErrInvalidInput)
	default:
		return fmt.Errorf("%w: 브라우저 렌더링으로 URL 본문을 가져오지 못했습니다.", app.ErrInvalidInput)
	}
}

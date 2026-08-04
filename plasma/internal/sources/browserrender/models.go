package browserrender

import (
	"errors"
	"time"
)

const (
	// DefaultMaxConcurrent는 headless browser 작업의 기본 동시 실행 상한이다.
	DefaultMaxConcurrent = 2
	// DefaultTimeout은 단일 browser render 작업 전체에 적용되는 기본 제한 시간이다.
	DefaultTimeout = 30 * time.Second
	// DefaultSettleDelay는 페이지 로드 후 DOM을 수집하기 전 기다리는 기본 시간이다.
	DefaultSettleDelay = 2 * time.Second
	// DefaultResolveTime은 렌더링 대상과 subresource URL의 DNS 검증 제한 시간이다.
	DefaultResolveTime = 3 * time.Second
	// MediaTypeHTML은 렌더링 결과 artifact가 사용하는 HTML media type이다.
	MediaTypeHTML = "text/html; charset=utf-8"
)

var (
	// ErrBlockedURL은 렌더링 대상이나 subresource가 허용되지 않는 URL일 때 반환된다.
	ErrBlockedURL = errors.New("browser render blocked URL")
	// ErrNoReadableBody는 렌더링 후에도 저장할 만큼의 읽을 수 있는 본문이 없을 때 반환된다.
	ErrNoReadableBody = errors.New("browser render produced no readable body")
	// ErrRenderFailed는 browser 실행은 가능했지만 렌더링 작업이 실패했을 때 반환된다.
	ErrRenderFailed = errors.New("browser render failed")
	// ErrRenderTimedOut은 browser render 제한 시간을 초과했을 때 반환된다.
	ErrRenderTimedOut = errors.New("browser render timed out")
	// ErrRenderUnavailable은 browser 실행 환경을 준비할 수 없을 때 반환된다.
	ErrRenderUnavailable = errors.New("browser render unavailable")
)

// Options는 Renderer의 실행 경계다. 0 값은 안전한 기본값으로 대체되며,
// UserAgent가 비어 있으면 Plasma 전용 기본 user agent를 사용한다.
type Options struct {
	MaxConcurrent int
	Timeout       time.Duration
	SettleDelay   time.Duration
	ResolveTime   time.Duration
	UserAgent     string
}

// Result는 browser render가 읽을 수 있는 문서로 정제한 결과다. Content는 저장
// 가능한 self-contained HTML fragment이고, FinalURL은 redirect 후 실제 문서 URL이다.
type Result struct {
	Content    []byte
	MediaType  string
	Title      string
	FinalURL   string
	RenderedAt time.Time
	TextLength int
}

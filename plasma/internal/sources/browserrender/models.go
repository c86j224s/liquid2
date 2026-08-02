package browserrender

import (
	"errors"
	"time"
)

const (
	DefaultMaxConcurrent = 2
	DefaultTimeout       = 30 * time.Second
	DefaultSettleDelay   = 2 * time.Second
	DefaultResolveTime   = 3 * time.Second
	MediaTypeHTML        = "text/html; charset=utf-8"
)

var (
	ErrBlockedURL        = errors.New("browser render blocked URL")
	ErrNoReadableBody    = errors.New("browser render produced no readable body")
	ErrRenderFailed      = errors.New("browser render failed")
	ErrRenderTimedOut    = errors.New("browser render timed out")
	ErrRenderUnavailable = errors.New("browser render unavailable")
)

type Options struct {
	MaxConcurrent int
	Timeout       time.Duration
	SettleDelay   time.Duration
	ResolveTime   time.Duration
	UserAgent     string
}

type Result struct {
	Content    []byte
	MediaType  string
	Title      string
	FinalURL   string
	RenderedAt time.Time
	TextLength int
}

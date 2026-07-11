package jobs

import "errors"

const (
	KindScrapeURL         = "scrape_url"
	KindTranslateDocument = "translate_document"
	KindPollFeed          = "poll_feed"
	KindExtractUploadText = "extract_upload_text"
	StatusQueued          = "queued"
	StatusRunning         = "running"
	StatusCompleted       = "completed"
	StatusFailed          = "failed"
)

var (
	ErrInvalidJob        = errors.New("invalid job")
	ErrJobConflict       = errors.New("job conflict")
	ErrJobNotFound       = errors.New("job not found")
	ErrInvalidTransition = errors.New("invalid job transition")
	ErrTooManyClaimKinds = errors.New("too many job claim kinds")
)

type Job struct {
	ID          string
	Kind        string
	Status      string
	PayloadJSON string
	Error       *string
	Attempts    int64
	CreatedAt   int64
	UpdatedAt   int64
	StartedAt   *int64
	FinishedAt  *int64
}

type EnqueueRequest struct {
	ID          string
	Kind        string
	PayloadJSON string
}

type Filters struct {
	Status string
	Kind   string
	Limit  int
}

func (request EnqueueRequest) validate() error {
	if request.Kind == "" || request.PayloadJSON == "" {
		return ErrInvalidJob
	}
	if !validKind(request.Kind) {
		return ErrInvalidJob
	}
	return nil
}

func validKind(kind string) bool {
	switch kind {
	case KindScrapeURL, KindTranslateDocument, KindPollFeed, KindExtractUploadText:
		return true
	default:
		return false
	}
}

func jobLimit(limit int) int64 {
	if limit <= 0 || limit > 100 {
		return 50
	}
	return int64(limit)
}

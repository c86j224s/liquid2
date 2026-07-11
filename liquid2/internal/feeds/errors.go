package feeds

import (
	"errors"
	"fmt"
)

var (
	ErrFetchFailed          = errors.New("feed fetch failed")
	ErrParseFailed          = errors.New("feed parse failed")
	ErrRefreshAlreadyQueued = errors.New("feed refresh already queued")
	ErrRefreshUnavailable   = errors.New("feed refresh unavailable")
	ErrInvalidJobPayload    = errors.New("invalid feed job payload")
)

func fetchFailed(message string, causes ...error) error {
	return classifiedError(ErrFetchFailed, message, causes...)
}

func parseFailed(message string, causes ...error) error {
	return classifiedError(ErrParseFailed, message, causes...)
}

func invalidPayload(message string, causes ...error) error {
	return classifiedError(ErrInvalidJobPayload, message, causes...)
}

func classifiedError(kind error, message string, causes ...error) error {
	if len(causes) > 0 && causes[0] != nil {
		return fmt.Errorf("%w: %s: %w", kind, message, causes[0])
	}
	return fmt.Errorf("%w: %s", kind, message)
}

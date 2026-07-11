package ingest

import (
	"errors"
	"fmt"
)

var (
	ErrBadRequest       = errors.New("bad request")
	ErrFetchFailed      = errors.New("fetch failed")
	ErrPayloadTooLarge  = errors.New("payload too large")
	ErrUnsafeURL        = errors.New("unsafe url")
	ErrUnsupportedMedia = errors.New("unsupported media type")
)

func badRequest(message string, causes ...error) error {
	return classifiedError(ErrBadRequest, message, causes...)
}

func fetchFailed(message string, causes ...error) error {
	return classifiedError(ErrFetchFailed, message, causes...)
}

func payloadTooLarge(message string, causes ...error) error {
	return classifiedError(ErrPayloadTooLarge, message, causes...)
}

func unsafeURL(message string, causes ...error) error {
	return classifiedError(ErrUnsafeURL, message, causes...)
}

func unsupportedMedia(message string, causes ...error) error {
	return classifiedError(ErrUnsupportedMedia, message, causes...)
}

func classifiedError(kind error, message string, causes ...error) error {
	if len(causes) > 0 && causes[0] != nil {
		return fmt.Errorf("%w: %s: %w", kind, message, causes[0])
	}
	return fmt.Errorf("%w: %s", kind, message)
}

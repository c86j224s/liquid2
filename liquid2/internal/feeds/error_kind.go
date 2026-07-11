package feeds

import (
	"context"
	"errors"

	"github.com/c86j224s/liquid2/internal/app"
)

func errorKind(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrFetchFailed):
		return "fetch_failed"
	case errors.Is(err, ErrParseFailed):
		return "parse_failed"
	case errors.Is(err, ErrInvalidJobPayload):
		return "invalid_payload"
	case errors.Is(err, app.ErrNotFound):
		return "not_found"
	case errors.Is(err, app.ErrConflict):
		return "conflict"
	case errors.Is(err, app.ErrValidation):
		return "validation"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	default:
		return "unknown"
	}
}

func safeStageError(err error) error {
	switch {
	case errors.Is(err, ErrFetchFailed):
		return ErrFetchFailed
	case errors.Is(err, ErrParseFailed):
		return ErrParseFailed
	case errors.Is(err, ErrInvalidJobPayload):
		return ErrInvalidJobPayload
	default:
		return err
	}
}

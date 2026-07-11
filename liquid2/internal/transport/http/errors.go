package httptransport

import (
	"errors"

	"github.com/c86j224s/liquid2/internal/app"
	feedrefresh "github.com/c86j224s/liquid2/internal/feeds"
	"github.com/c86j224s/liquid2/internal/ingest"
	"github.com/c86j224s/liquid2/internal/translation"
	"github.com/danielgtaylor/huma/v2"
)

func mapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ingest.ErrBadRequest):
		return huma.Error400BadRequest("bad request")
	case errors.Is(err, ingest.ErrPayloadTooLarge):
		return huma.Error413RequestEntityTooLarge("payload too large")
	case errors.Is(err, ingest.ErrUnsupportedMedia):
		return huma.Error415UnsupportedMediaType("unsupported media type")
	case errors.Is(err, ingest.ErrUnsafeURL):
		return huma.Error400BadRequest("unsafe url")
	case errors.Is(err, ingest.ErrFetchFailed):
		return huma.Error400BadRequest("fetch failed")
	case errors.Is(err, feedrefresh.ErrRefreshUnavailable):
		return huma.Error503ServiceUnavailable("feed refresh unavailable")
	case errors.Is(err, feedrefresh.ErrRefreshAlreadyQueued):
		return huma.Error409Conflict("feed refresh already queued")
	case errors.Is(err, translation.ErrTranslationUnavailable):
		return huma.Error503ServiceUnavailable("translation unavailable")
	case errors.Is(err, translation.ErrTranslationAlreadyQueued):
		return huma.Error409Conflict("translation already queued")
	case errors.Is(err, app.ErrNotFound):
		return huma.Error404NotFound(err.Error())
	case errors.Is(err, app.ErrConflict):
		return huma.Error409Conflict(err.Error())
	case errors.Is(err, app.ErrValidation):
		return huma.Error400BadRequest(err.Error())
	default:
		return huma.Error500InternalServerError("internal error")
	}
}

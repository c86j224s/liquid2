package ingest

import (
	"context"
	"log/slog"

	"github.com/c86j224s/liquid2/internal/app"
)

func (service *Service) Rescrape(ctx context.Context, documentID string) (app.DocumentDetail, error) {
	target, err := service.documents.PrepareRescrape(ctx, documentID)
	if err != nil {
		service.logger.WarnContext(ctx, "re-scrape target rejected",
			slog.String("operation", "ingest_rescrape"),
			slog.String("document_id", documentID),
			slog.Any("error", err),
		)
		return app.DocumentDetail{}, err
	}
	page, err := service.fetcher.Fetch(ctx, target.URL)
	if err != nil {
		service.logger.WarnContext(ctx, "re-scrape fetch failed",
			slog.String("operation", "ingest_rescrape"),
			slog.String("document_id", documentID),
			slog.String("error_kind", ingestErrorKind(err)),
		)
		return app.DocumentDetail{}, err
	}
	detail, err := service.documents.ReplaceRescrapedContent(ctx, documentID, app.RescrapedContentInput{
		URL: page.URL, SourceURL: target.URL, Content: page.Content, Format: page.Format,
	})
	if err != nil {
		service.logger.WarnContext(ctx, "re-scrape update failed",
			slog.String("operation", "ingest_rescrape"),
			slog.String("document_id", documentID),
			slog.Any("error", err),
		)
		return app.DocumentDetail{}, err
	}
	service.logger.DebugContext(ctx, "document re-scraped",
		slog.String("operation", "ingest_rescrape"),
		slog.String("document_id", detail.Document.ID),
	)
	return detail, nil
}

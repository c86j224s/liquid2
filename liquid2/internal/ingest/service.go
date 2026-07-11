package ingest

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"

	"github.com/c86j224s/liquid2/internal/app"
)

type Service struct {
	documents *app.Service
	guard     URLGuard
	fetcher   Fetcher
	logger    *slog.Logger
}

type ServiceOption func(*Service)

func NewService(documents *app.Service, options ...ServiceOption) *Service {
	service := &Service{
		documents: documents,
		guard:     NewURLGuard(),
		logger:    slog.Default().With("component", "ingest"),
	}
	for _, option := range options {
		option(service)
	}
	if service.fetcher == nil {
		service.fetcher = NewHTTPFetcher(WithURLGuard(service.guard))
	}
	return service
}

func WithFetcher(fetcher Fetcher) ServiceOption {
	return func(service *Service) {
		if fetcher != nil {
			service.fetcher = fetcher
		}
	}
}

func WithGuard(guard URLGuard) ServiceOption {
	return func(service *Service) {
		service.guard = guard
	}
}

func WithLogger(logger *slog.Logger) ServiceOption {
	return func(service *Service) {
		if logger != nil {
			service.logger = logger.With("component", "ingest")
		}
	}
}

type BookmarkInput struct {
	URL      string
	Title    string
	FolderID string
	TagIDs   []string
}

type ScrapeInput struct {
	URL      string
	FolderID string
	TagIDs   []string
}

type UploadDocumentInput struct {
	Title       string
	Filename    string
	ContentType string
	Data        []byte
	FolderID    string
	TagIDs      []string
}

func (service *Service) Bookmark(ctx context.Context, input BookmarkInput) (app.DocumentDetail, error) {
	normalized, err := service.guard.Normalize(ctx, input.URL)
	if err != nil {
		service.logger.WarnContext(ctx, "bookmark url rejected", slog.String("operation", "ingest_bookmark"))
		return app.DocumentDetail{}, err
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = titleFromURL(normalized)
	}
	detail, err := service.documents.CreateBookmarkDocument(ctx, app.BookmarkDocumentInput{
		URL: normalized, SourceURL: input.URL, Title: title,
		FolderID: input.FolderID, TagIDs: input.TagIDs,
	})
	if err != nil {
		service.logger.WarnContext(ctx, "bookmark create failed", slog.String("operation", "ingest_bookmark"), slog.Any("error", err))
		return app.DocumentDetail{}, err
	}
	service.logger.DebugContext(ctx, "bookmark created", slog.String("operation", "ingest_bookmark"), slog.String("document_id", detail.Document.ID))
	return detail, nil
}

func (service *Service) Scrape(ctx context.Context, input ScrapeInput) (app.DocumentDetail, error) {
	page, err := service.fetcher.Fetch(ctx, input.URL)
	if err != nil {
		service.logger.WarnContext(ctx, "scrape fetch failed",
			slog.String("operation", "ingest_scrape"),
			slog.String("error_kind", ingestErrorKind(err)),
		)
		return app.DocumentDetail{}, err
	}
	title := page.Title
	if strings.TrimSpace(title) == "" {
		title = titleFromURL(page.URL)
	}
	detail, err := service.documents.CreateScrapedDocument(ctx, app.ScrapedDocumentInput{
		URL: page.URL, SourceURL: input.URL, Title: title, Content: page.Content,
		Format: page.Format, FolderID: input.FolderID, TagIDs: input.TagIDs,
	})
	if err != nil {
		service.logger.WarnContext(ctx, "scrape create failed", slog.String("operation", "ingest_scrape"), slog.Any("error", err))
		return app.DocumentDetail{}, err
	}
	service.logger.DebugContext(ctx, "scrape created", slog.String("operation", "ingest_scrape"), slog.String("document_id", detail.Document.ID))
	return detail, nil
}

func (service *Service) Upload(ctx context.Context, input UploadDocumentInput) (app.DocumentDetail, error) {
	upload, err := PrepareUpload(UploadInput{
		Title: input.Title, Filename: input.Filename,
		ContentType: input.ContentType, Data: input.Data,
	})
	if err != nil {
		service.logger.WarnContext(ctx, "upload rejected", slog.String("operation", "ingest_upload"), slog.Any("error", err))
		return app.DocumentDetail{}, err
	}
	detail, err := service.documents.CreateUploadedDocument(ctx, app.UploadedDocumentInput{
		Title: upload.Title, Filename: upload.Filename, MimeType: upload.MimeType,
		Data: upload.Data, Content: upload.Content, Format: upload.Format,
		FolderID: input.FolderID, TagIDs: input.TagIDs,
	})
	if err != nil {
		service.logger.WarnContext(ctx, "upload create failed", slog.String("operation", "ingest_upload"), slog.Any("error", err))
		return app.DocumentDetail{}, err
	}
	service.logger.DebugContext(ctx, "upload created", slog.String("operation", "ingest_upload"), slog.String("document_id", detail.Document.ID))
	return detail, nil
}

func ingestErrorKind(err error) string {
	switch {
	case errors.Is(err, ErrUnsafeURL):
		return "unsafe_url"
	case errors.Is(err, ErrUnsupportedMedia):
		return "unsupported_media"
	case errors.Is(err, ErrPayloadTooLarge):
		return "payload_too_large"
	case errors.Is(err, ErrFetchFailed):
		return "fetch_failed"
	default:
		return "unknown"
	}
}

func titleFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return "Untitled document"
	}
	path := strings.Trim(strings.TrimSpace(parsed.Path), "/")
	if path == "" {
		return parsed.Hostname()
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

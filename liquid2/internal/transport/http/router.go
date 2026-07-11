package httptransport

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/ingest"
	"github.com/c86j224s/liquid2/internal/translation"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// Option customizes router construction.
type Option func(*routerConfig)

// routerConfig contains construction-time router options.
type routerConfig struct {
	// logger records HTTP request and routing events.
	logger *slog.Logger
	// ingestion coordinates URL and file ingestion workflows.
	ingestion *ingest.Service
	// feedRefresher enqueues manual feed refresh work.
	feedRefresher FeedRefresher
	// documentTranslator enqueues document translation work.
	documentTranslator DocumentTranslator
	// backupRunner creates server-side backup artifacts.
	backupRunner BackupRunner
	// exportRunner creates server-side markdown export artifacts.
	exportRunner ExportRunner
	// corsOrigins contains exact origins allowed to call the API from browsers.
	corsOrigins []string
}

type FeedRefresher interface {
	RefreshFeed(ctx context.Context, feedID string) (app.Job, error)
}

type DocumentTranslator interface {
	TranslateDocument(ctx context.Context, input translation.EnqueueDocumentInput) (app.Job, error)
}

type BackupRunner interface {
	Backup(ctx context.Context) (BackupArtifact, error)
}

type ExportRunner interface {
	Export(ctx context.Context, request ExportRequest) (ExportArtifact, error)
	GetExport(ctx context.Context, id string) (ExportArtifact, error)
}

func NewRouter(service *app.Service, options ...Option) http.Handler {
	config := routerConfig{
		logger: slog.Default().With("component", "http"),
	}
	for _, option := range options {
		option(&config)
	}

	router := chi.NewRouter()
	router.Use(requestLogger(config.logger))
	router.Use(corsMiddleware(config.corsOrigins))
	registerAPI(router, service, config)
	return router
}

func OpenAPISpec(service *app.Service, options ...Option) *huma.OpenAPI {
	config := routerConfig{
		logger: slog.Default().With("component", "http"),
	}
	for _, option := range options {
		option(&config)
	}
	router := chi.NewRouter()
	api := registerAPI(router, service, config)
	return api.OpenAPI()
}

func WithLogger(logger *slog.Logger) Option {
	return func(config *routerConfig) {
		if logger != nil {
			config.logger = logger.With("component", "http")
		}
	}
}

func WithIngestion(ingestion *ingest.Service) Option {
	return func(config *routerConfig) {
		if ingestion != nil {
			config.ingestion = ingestion
		}
	}
}

func WithFeedRefresher(refresher FeedRefresher) Option {
	return func(config *routerConfig) {
		if refresher != nil {
			config.feedRefresher = refresher
		}
	}
}

func WithDocumentTranslator(translator DocumentTranslator) Option {
	return func(config *routerConfig) {
		if translator != nil {
			config.documentTranslator = translator
		}
	}
}

func WithBackupRunner(runner BackupRunner) Option {
	return func(config *routerConfig) {
		if runner != nil {
			config.backupRunner = runner
		}
	}
}

func WithExportRunner(runner ExportRunner) Option {
	return func(config *routerConfig) {
		if runner != nil {
			config.exportRunner = runner
		}
	}
}

func WithCORSOrigins(origins []string) Option {
	return func(config *routerConfig) {
		config.corsOrigins = append([]string(nil), origins...)
	}
}

func registerAPI(router chi.Router, service *app.Service, config routerConfig) huma.API {
	apiConfig := huma.DefaultConfig("Liquid2 API", "0.2.0")
	apiConfig.Info.Description = "Personal document repository API."
	apiConfig.Servers = []*huma.Server{{URL: "/"}}
	apiConfig.RejectUnknownQueryParameters = true
	apiConfig.Transformers = nil
	apiConfig.CreateHooks = nil

	api := humachi.New(router, apiConfig)
	registerHealth(api, service)
	registerDocumentRoutes(api, service, config.documentTranslator)
	registerIngestionRoutes(api, ingestionService(service, config), config.documentTranslator)
	registerNoteRoutes(api, service)
	registerFolderRoutes(api, service)
	registerTagRoutes(api, service)
	registerFeedRoutes(api, service, config.feedRefresher)
	registerJobRoutes(api, service)
	registerSettingsRoutes(api, service)
	registerBackupRoutes(api, config.backupRunner, config.logger)
	registerExportRoutes(api, config.exportRunner, config.logger)
	config.logger.Debug("api routes registered", slog.String("operation", "api_register"))
	return api
}

func ingestionService(service *app.Service, config routerConfig) *ingest.Service {
	if config.ingestion != nil {
		return config.ingestion
	}
	return ingest.NewService(service, ingest.WithLogger(config.logger))
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/c86j224s/liquid2/internal/app"
	liquidconfig "github.com/c86j224s/liquid2/internal/config"
	feedrefresh "github.com/c86j224s/liquid2/internal/feeds"
	jobruntime "github.com/c86j224s/liquid2/internal/jobs"
	"github.com/c86j224s/liquid2/internal/logging"
	sqlitestore "github.com/c86j224s/liquid2/internal/storage/sqlite"
	"github.com/c86j224s/liquid2/internal/translation"
	httptransport "github.com/c86j224s/liquid2/internal/transport/http"
)

func main() {
	os.Exit(runWithArgs(os.Args[1:], os.Stdout, os.Stderr))
}

var activeConfig liquidconfig.Config
var hasActiveConfig bool

func run() int {
	return runWithArgs(nil, os.Stdout, os.Stderr)
}

func runWithArgs(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "serve":
			return runServe(stdout, stderr)
		case "status":
			return runStatus(args[1:], stdout, stderr)
		default:
			fmt.Fprintf(stderr, "unknown command %q\n", args[0])
			fmt.Fprintln(stderr, "usage: liquid2-api [serve|status]")
			return 2
		}
	}
	return runServe(stdout, stderr)
}

func runServe(stdout, stderr io.Writer) int {
	cfg, err := loadRuntimeConfig(nil)
	if err != nil {
		fmt.Fprintf(stderr, "configure: %v\n", err)
		return 2
	}
	activeConfig = cfg
	hasActiveConfig = true

	ctx := context.Background()
	addr := getenv("LIQUID2_ADDR", ":8080")
	logger, err := logging.New(stdout, logging.Config{
		Level:     getenv("LIQUID2_LOG_LEVEL", "info"),
		Format:    getenv("LIQUID2_LOG_FORMAT", logging.FormatJSON),
		AddSource: getenv("LIQUID2_LOG_SOURCE", "") == "1",
	})
	if err != nil {
		fallback := slog.New(slog.NewTextHandler(stderr, nil))
		fallback.Error("configure logger failed", slog.String("component", "api"), slog.Any("error", err))
		return 1
	}
	slog.SetDefault(logger)

	service, queue, runtimeStatus, backupRunner, exportRunner, cleanup, err := newAppService(ctx, logger)
	if err != nil {
		return 1
	}
	defer cleanup()
	if err := seedDemo(ctx, logger, service); err != nil {
		return 1
	}
	return serveAPI(addr, logger, service, queue, runtimeStatus, backupRunner, exportRunner)
}

func runStatus(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	field := fs.String("field", "", "print one resolved field")
	jsonOut := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadRuntimeConfig(nil)
	if err != nil {
		fmt.Fprintf(stderr, "configure: %v\n", err)
		return 2
	}
	status := resolvedStatus(cfg)
	if strings.TrimSpace(*field) != "" {
		value, ok := statusField(status, strings.TrimSpace(*field))
		if !ok {
			fmt.Fprintf(stderr, "unknown status field %q\n", *field)
			return 2
		}
		fmt.Fprintln(stdout, value)
		return 0
	}
	if *jsonOut {
		if err := json.NewEncoder(stdout).Encode(status); err != nil {
			fmt.Fprintf(stderr, "status json: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "Liquid2 %s\n", status.Mode)
	fmt.Fprintf(stdout, "  API     %s\n", status.APIURL)
	fmt.Fprintf(stdout, "  Web     %s\n", status.WebURL)
	fmt.Fprintf(stdout, "  DB      %s\n", status.DBPath)
	fmt.Fprintf(stdout, "  Mode    %s\n", status.Mode)
	return 0
}

func loadRuntimeConfig(args liquidconfig.Args) (liquidconfig.Config, error) {
	cfg, err := liquidconfig.Load(args)
	if err != nil {
		return liquidconfig.Config{}, err
	}
	if err := applyRuntimeDefaults(cfg); err != nil {
		return liquidconfig.Config{}, err
	}
	return cfg, nil
}

func applyRuntimeDefaults(cfg liquidconfig.Config) error {
	mode, err := liquidconfig.RuntimeMode()
	if err != nil {
		return err
	}
	cfg.ApplyDefaults(runtimeDefaults(mode))
	status := resolvedStatus(cfg)
	cfg.ApplyDefaults(liquidconfig.Args{
		liquidconfig.KeyCORSOrigins: defaultCORSOrigins(mode, status.WebURL, status.WebPort),
	})
	return nil
}

func runtimeDefaults(mode string) liquidconfig.Args {
	home, _ := os.UserHomeDir()
	defaults := liquidconfig.Args{
		liquidconfig.KeyLogFormat:   logging.FormatText,
		liquidconfig.KeyJobsEnabled: "1",
		liquidconfig.KeySeedDemo:    "0",
	}
	switch mode {
	case liquidconfig.RuntimeModeDev:
		defaults[liquidconfig.KeyAddr] = "127.0.0.1:6011"
		defaults[liquidconfig.KeyWebAddr] = "127.0.0.1"
		defaults[liquidconfig.KeyWebPort] = "6001"
		if home != "" {
			defaults[liquidconfig.KeyDBPath] = filepath.Join(home, "research-artifacts", "liquid2", "liquid2", "runtime", "dev-6011", "liquid2-dev.db")
			defaults[liquidconfig.KeyExportDir] = filepath.Join(home, "research-artifacts", "liquid2", "liquid2", "runtime", "dev-6011", "exports")
			defaults[liquidconfig.KeyBackupDir] = filepath.Join(home, "research-artifacts", "liquid2", "liquid2", "runtime", "dev-6011", "backups")
		}
	default:
		defaults[liquidconfig.KeyAddr] = "127.0.0.1:3011"
		defaults[liquidconfig.KeyWebAddr] = "127.0.0.1"
		defaults[liquidconfig.KeyWebPort] = "3001"
		if home != "" {
			defaults[liquidconfig.KeyDBPath] = filepath.Join(home, "Library", "Application Support", "Liquid2", "liquid2.db")
			defaults[liquidconfig.KeyExportDir] = filepath.Join(home, "Library", "Application Support", "Liquid2", "exports")
			defaults[liquidconfig.KeyBackupDir] = filepath.Join(home, "Library", "Application Support", "Liquid2", "backups")
		}
	}
	return defaults
}

type runtimeStatus struct {
	Mode             string `json:"mode"`
	APIAddr          string `json:"api_addr"`
	APIURL           string `json:"api_url"`
	WebHost          string `json:"web_host"`
	WebPort          string `json:"web_port"`
	WebURL           string `json:"web_url"`
	DBPath           string `json:"db_path"`
	ExportDir        string `json:"export_dir"`
	BackupDir        string `json:"backup_dir"`
	CORSOrigins      string `json:"cors_origins"`
	EnvironmentLabel string `json:"environment_label"`
}

func resolvedStatus(cfg liquidconfig.Config) runtimeStatus {
	mode, err := liquidconfig.RuntimeMode()
	if err != nil {
		mode = liquidconfig.RuntimeModeRelease
	}
	apiAddr := cfg.Value(liquidconfig.KeyAddr, "127.0.0.1:3011")
	webHost := resolvedWebHost(cfg, apiAddr)
	webPort := cfg.Value(liquidconfig.KeyWebPort, "3001")
	return runtimeStatus{
		Mode:             mode,
		APIAddr:          apiAddr,
		APIURL:           urlForAddr(apiAddr),
		WebHost:          webHost,
		WebPort:          webPort,
		WebURL:           urlForHostPort(webHost, webPort),
		DBPath:           cfg.Value(liquidconfig.KeyDBPath, ""),
		ExportDir:        cfg.Value(liquidconfig.KeyExportDir, ""),
		BackupDir:        cfg.Value(liquidconfig.KeyBackupDir, ""),
		CORSOrigins:      cfg.Value(liquidconfig.KeyCORSOrigins, ""),
		EnvironmentLabel: cfg.Value(liquidconfig.KeyEnvironmentLabel, ""),
	}
}

func statusField(status runtimeStatus, field string) (string, bool) {
	switch field {
	case "mode":
		return status.Mode, true
	case "api_addr":
		return status.APIAddr, true
	case "api_url":
		return status.APIURL, true
	case "web_host":
		return status.WebHost, true
	case "web_port":
		return status.WebPort, true
	case "web_url":
		return status.WebURL, true
	case "db_path":
		return status.DBPath, true
	case "export_dir":
		return status.ExportDir, true
	case "backup_dir":
		return status.BackupDir, true
	case "cors_origins":
		return status.CORSOrigins, true
	case "environment_label":
		return status.EnvironmentLabel, true
	default:
		return "", false
	}
}

func defaultCORSOrigins(mode string, webURL string, webPort string) string {
	plasmaPort := "3002"
	if mode == liquidconfig.RuntimeModeDev {
		plasmaPort = "6002"
	}
	values := []string{
		webURL,
		"http://127.0.0.1:" + webPort,
		"http://localhost:" + webPort,
		"http://127.0.0.1:" + plasmaPort,
		"http://localhost:" + plasmaPort,
	}
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return strings.Join(unique, ",")
}

func resolvedWebHost(cfg liquidconfig.Config, apiAddr string) string {
	value := strings.TrimSpace(cfg.Value(liquidconfig.KeyWebAddr, ""))
	if value == "" {
		return hostForAddr(apiAddr)
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		if strings.TrimSpace(host) == "" {
			return "127.0.0.1"
		}
		return host
	}
	return value
}

func urlForAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if host, port, err := net.SplitHostPort(addr); err == nil {
		if strings.TrimSpace(host) == "" {
			host = "127.0.0.1"
		}
		return "http://" + net.JoinHostPort(host, port)
	}
	return "http://" + addr
}

func urlForHostPort(host string, port string) string {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		return "http://" + host
	}
	return "http://" + net.JoinHostPort(host, port)
}

func hostForAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if host, _, err := net.SplitHostPort(addr); err == nil {
		if strings.TrimSpace(host) == "" {
			return "127.0.0.1"
		}
		return host
	}
	if strings.TrimSpace(addr) == "" || strings.HasPrefix(addr, ":") {
		return "127.0.0.1"
	}
	if index := strings.LastIndex(addr, ":"); index > 0 {
		return addr[:index]
	}
	return addr
}

func newAppService(
	ctx context.Context,
	logger *slog.Logger,
) (*app.Service, jobruntime.Queue, jobRuntimeStatus, httptransport.BackupRunner, httptransport.ExportRunner, func(), error) {
	serviceOptions := []app.Option{app.WithLogger(logger)}
	store, err := openSQLiteStore(ctx, logger)
	if err != nil {
		return nil, nil, jobRuntimeStatus{}, nil, nil, nil, err
	}
	var queue jobruntime.Queue
	if store != nil {
		serviceOptions = append(serviceOptions, app.WithRepository(app.NewSQLiteRepository(
			store,
			app.WithSQLiteRepositoryLogger(logger),
		)))
		queue = jobruntime.NewSQLiteQueue(store, jobruntime.WithSQLiteLogger(logger))
	}
	backupRunner, err := newBackupRunner(logger, store)
	if err != nil {
		if store != nil {
			_ = store.Close()
		}
		return nil, nil, jobRuntimeStatus{}, nil, nil, nil, err
	}
	service := app.NewService(serviceOptions...)
	exportRunner := newExportRunner(logger, service, store)
	runtimeStatus, jobsCleanup, err := startJobRuntime(ctx, logger, service, queue)
	if err != nil {
		_ = service.Close()
		if store != nil {
			_ = store.Close()
		}
		return nil, nil, jobRuntimeStatus{}, nil, nil, nil, err
	}
	cleanup := func() {
		jobsCleanup()
		if err := service.Close(); err != nil {
			logger.Error("close app service failed", slog.String("component", "api"), slog.Any("error", err))
		}
		if store != nil {
			if err := store.Close(); err != nil {
				logger.Error("close sqlite store failed", slog.String("component", "api"), slog.Any("error", err))
			}
		}
	}
	return service, queue, runtimeStatus, backupRunner, exportRunner, cleanup, nil
}

func openSQLiteStore(ctx context.Context, logger *slog.Logger) (_ *sqlitestore.Store, err error) {
	dbPath := getenv("LIQUID2_DB_PATH", "")
	if dbPath == "" {
		return nil, nil
	}
	if err := ensureDBParent(dbPath); err != nil {
		logger.Error("create sqlite parent directory failed", slog.String("component", "api"), slog.String("db_path", dbPath), slog.Any("error", err))
		return nil, err
	}
	store, err := sqlitestore.Open(ctx, dbPath, sqlitestore.WithLogger(logger))
	if err != nil {
		logger.Error("open sqlite store failed", slog.String("component", "api"), slog.String("db_path", dbPath), slog.Any("error", err))
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = store.Close()
		}
	}()
	if err = store.Migrate(ctx); err != nil {
		logger.Error("migrate sqlite store failed", slog.String("component", "api"), slog.String("db_path", dbPath), slog.Any("error", err))
		return nil, err
	}
	logger.Info("sqlite persistence enabled", slog.String("component", "api"), slog.String("operation", "sqlite_open"), slog.String("db_path", dbPath))
	return store, nil
}

func ensureDBParent(path string) error {
	path = strings.TrimSpace(path)
	if path == "" || path == ":memory:" || strings.HasPrefix(path, "file:") {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o700)
}

func seedDemo(ctx context.Context, logger *slog.Logger, service *app.Service) error {
	if getenv("LIQUID2_SEED_DEMO", "") == "1" {
		if err := service.SeedDemo(ctx); err != nil {
			logger.Error("demo seed failed", slog.String("component", "api"), slog.String("operation", "demo_seed"), slog.Any("error", err))
			return err
		}
		logger.Info("demo seed enabled", slog.String("component", "api"), slog.String("operation", "demo_seed"))
	}
	return nil
}

func serveAPI(
	addr string,
	logger *slog.Logger,
	service *app.Service,
	queue jobruntime.Queue,
	runtimeStatus jobRuntimeStatus,
	backupRunner httptransport.BackupRunner,
	exportRunner httptransport.ExportRunner,
) int {
	options := []httptransport.Option{httptransport.WithLogger(logger)}
	if origins := corsOriginsFromEnv(); len(origins) > 0 {
		logger.Info("cors origins enabled",
			slog.String("component", "api"),
			slog.String("operation", "cors_config"),
			slog.Any("origins", origins),
		)
		options = append(options, httptransport.WithCORSOrigins(origins))
	}
	if queue != nil && runtimeStatus.jobsEnabled {
		options = append(options, httptransport.WithFeedRefresher(
			feedrefresh.NewRefresher(service, queue, feedrefresh.WithRefresherLogger(logger)),
		))
	}
	if queue != nil && runtimeStatus.translationEnabled {
		options = append(options, httptransport.WithDocumentTranslator(
			translation.NewEnqueuer(service, queue, translation.WithEnqueuerLogger(logger)),
		))
	}
	if backupRunner != nil {
		options = append(options, httptransport.WithBackupRunner(backupRunner))
	}
	if exportRunner != nil {
		options = append(options, httptransport.WithExportRunner(exportRunner))
	}
	router := httptransport.NewRouter(service, options...)
	server := &http.Server{
		Addr:     addr,
		Handler:  router,
		ErrorLog: slog.NewLogLogger(logger.With("component", "http_server").Handler(), slog.LevelError),
	}

	logger.Info("api server starting", slog.String("component", "api"), slog.String("addr", addr))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("api server stopped", slog.String("component", "api"), slog.String("addr", addr), slog.Any("error", err))
		return 1
	}
	return 0
}

func getenv(key string, fallback string) string {
	if hasActiveConfig {
		return activeConfig.EnvValue(key, fallback)
	}
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

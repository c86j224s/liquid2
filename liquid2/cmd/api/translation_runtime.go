package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/internal/app"
	jobruntime "github.com/c86j224s/liquid2/internal/jobs"
	"github.com/c86j224s/liquid2/internal/translation"
)

const passthroughTranslationProvider = "passthrough"
const codexTranslationProvider = "codex"
const defaultCodexTimeoutSeconds = 300

func newTranslationHandler(
	logger *slog.Logger,
	service *app.Service,
	providerName string,
) (jobruntime.Handler, bool, error) {
	providerName = normalizeTranslationProviderName(providerName)
	if providerName == "" {
		return nil, false, nil
	}
	provider, err := newTranslationProvider(providerName)
	if err != nil {
		return nil, false, err
	}
	pipeline := translation.NewPipeline(service, provider, translation.WithLogger(logger))
	return pipeline.Handle, true, nil
}

func normalizeTranslationProviderName(providerName string) string {
	return strings.ToLower(strings.TrimSpace(providerName))
}

func newTranslationProvider(providerName string) (translation.Provider, error) {
	switch providerName {
	case passthroughTranslationProvider:
		return translation.PassthroughProvider{}, nil
	case codexTranslationProvider:
		timeout, err := codexProviderTimeout()
		if err != nil {
			return nil, err
		}
		return translation.NewCodexProvider(
			translation.WithCodexCommand(getenv("LIQUID2_CODEX_COMMAND", "codex")),
			translation.WithCodexModel(getenv("LIQUID2_CODEX_MODEL", "")),
			translation.WithCodexTimeout(timeout),
		), nil
	default:
		return nil, fmt.Errorf(
			"LIQUID2_TRANSLATION_PROVIDER must be %q or %q",
			passthroughTranslationProvider,
			codexTranslationProvider,
		)
	}
}

func codexProviderTimeout() (time.Duration, error) {
	raw := strings.TrimSpace(getenv("LIQUID2_CODEX_TIMEOUT_SECONDS", ""))
	if raw == "" {
		return defaultCodexTimeoutSeconds * time.Second, nil
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("LIQUID2_CODEX_TIMEOUT_SECONDS must be positive seconds")
	}
	return time.Duration(seconds) * time.Second, nil
}

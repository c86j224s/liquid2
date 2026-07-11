package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/config"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func openCLIService(ctx context.Context, dbPath string, localRootSpecs ...string) (*app.Service, func(), string, error) {
	cfg, err := config.Load(config.Args{DBPath: dbPath, LocalSourceRoots: localRootSpecs})
	if err != nil {
		return nil, nil, "", err
	}
	store, err := sqlite.Open(ctx, cfg.EffectiveDBPath())
	if err != nil {
		return nil, nil, "", err
	}
	svc, err := newCLIService(store, cfg, nil)
	if err != nil {
		_ = store.Close()
		return nil, nil, "", err
	}
	return svc, func() { _ = store.Close() }, cfg.DisplayDBPath(), nil
}

func loadAgentConfig(args config.Args) (config.Config, time.Duration, error) {
	cfg, err := config.Load(args)
	if err != nil {
		return config.Config{}, 0, err
	}
	timeout, err := parseConfigDuration(cfg.AgentTimeout)
	if err != nil {
		return config.Config{}, 0, err
	}
	return cfg, timeout, nil
}

func durationArg(value time.Duration) string {
	if value <= 0 {
		return ""
	}
	return value.String()
}

func parseConfigDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("agent_timeout must be a Go duration such as 30s or 5m: %w", err)
	}
	return duration, nil
}

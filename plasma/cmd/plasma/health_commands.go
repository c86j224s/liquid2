package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/config"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func runHealth(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.Load(config.Args{DBPath: *dbPath})
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 2
	}

	store, err := sqlite.Open(ctx, cfg.EffectiveDBPath())
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer store.Close()

	svc := app.NewService(store)
	health, err := svc.Health(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "health: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "plasma %s version=%s db=%s migrations=%d\n",
		health.Status, health.Version, cfg.DisplayDBPath(), len(health.Migrations))
	return 0
}

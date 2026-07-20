package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/config"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func runStorage(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: plasma storage <stats|compact> [options]")
		return 2
	}
	switch args[0] {
	case "stats":
		return runStorageStats(ctx, args[1:], stdout, stderr)
	case "compact":
		return runStorageCompact(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown storage command %q\n", args[0])
		return 2
	}
}

func runStorageStats(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("storage stats", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	jsonOut := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	path, err := storageDBPath(*dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 2
	}
	stats, err := sqlite.ReadStorageStats(ctx, path)
	if err != nil {
		fmt.Fprintf(stderr, "storage stats: %v\n", err)
		return storageErrorExitCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"storage": stats})
		return 0
	}
	writeStorageStats(stdout, stats)
	return 0
}

func runStorageCompact(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("storage compact", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	outputPath := fs.String("output", "", "write compacted SQLite database to this path")
	replace := fs.Bool("replace", false, "replace the source database after backing it up")
	dryRun := fs.Bool("dry-run", false, "measure exact compacted size without leaving an output database")
	jsonOut := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *dryRun && (strings.TrimSpace(*outputPath) != "" || *replace) {
		fmt.Fprintln(stderr, "--dry-run cannot be combined with --output or --replace")
		return 2
	}
	if !*dryRun && strings.TrimSpace(*outputPath) == "" && !*replace {
		fmt.Fprintln(stderr, "usage: plasma storage compact -db <path> (--dry-run | --output <path> | --replace)")
		return 2
	}
	if strings.TrimSpace(*outputPath) != "" && *replace {
		fmt.Fprintln(stderr, "--output cannot be combined with --replace")
		return 2
	}
	path, err := storageDBPath(*dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 2
	}
	var result sqlite.StorageCompactResult
	if *dryRun {
		result, err = sqlite.CompactStorageDryRun(ctx, path)
	} else if *replace {
		result, err = sqlite.CompactStorageReplace(ctx, path)
	} else {
		result, err = sqlite.CompactStorageTo(ctx, path, *outputPath)
	}
	if err != nil {
		fmt.Fprintf(stderr, "storage compact: %v\n", err)
		if errors.Is(err, sqlite.ErrStorageMaintenanceOfflineRequired) {
			fmt.Fprintln(stderr, "stop Plasma before running this command against the runtime database.")
		}
		return storageErrorExitCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"compact": result})
		return 0
	}
	writeStorageCompactResult(stdout, result)
	return 0
}

func storageDBPath(dbPath string) (string, error) {
	cfg, err := config.Load(config.Args{DBPath: dbPath})
	if err != nil {
		return "", err
	}
	applyServeDefaults(&cfg)
	return cfg.EffectiveDBPath(), nil
}

func writeStorageStats(w io.Writer, stats sqlite.StorageStats) {
	fmt.Fprintln(w, "Plasma storage")
	fmt.Fprintf(w, "  DB                 %s\n", stats.DBPath)
	fmt.Fprintf(w, "  DB bytes           %d\n", stats.DBBytes)
	fmt.Fprintf(w, "  WAL bytes          %d\n", stats.WALBytes)
	fmt.Fprintf(w, "  SHM bytes          %d\n", stats.SHMBytes)
	fmt.Fprintf(w, "  page size          %d\n", stats.PageSize)
	fmt.Fprintf(w, "  page count         %d\n", stats.PageCount)
	fmt.Fprintf(w, "  freelist count     %d\n", stats.FreelistCount)
	fmt.Fprintf(w, "  reclaimable bytes  %d\n", stats.ReclaimableBytes)
	fmt.Fprintf(w, "  journal mode       %s\n", stats.JournalMode)
	fmt.Fprintf(w, "  auto vacuum        %d\n", stats.AutoVacuum)
}

func writeStorageCompactResult(w io.Writer, result sqlite.StorageCompactResult) {
	if result.DryRun {
		fmt.Fprintln(w, "compacted storage dry run")
	} else {
		fmt.Fprintln(w, "compacted storage")
	}
	fmt.Fprintf(w, "  DB                 %s\n", result.DBPath)
	if result.OutputPath != "" {
		fmt.Fprintf(w, "  output             %s\n", result.OutputPath)
	}
	fmt.Fprintf(w, "  replaced           %t\n", result.Replaced)
	fmt.Fprintf(w, "  dry run            %t\n", result.DryRun)
	fmt.Fprintf(w, "  original bytes     %d\n", result.Original.TotalBytes())
	fmt.Fprintf(w, "  compacted bytes    %d\n", result.Compacted.TotalBytes())
	fmt.Fprintf(w, "  saved bytes        %d\n", result.SavedBytes)
	fmt.Fprintf(w, "  integrity check    %s\n", result.IntegrityCheck)
	for _, backup := range result.BackupPaths {
		fmt.Fprintf(w, "  backup             %s\n", backup)
	}
}

func storageErrorExitCode(err error) int {
	if errors.Is(err, sqlite.ErrStorageMaintenanceFileBackedRequired) ||
		errors.Is(err, sqlite.ErrStorageMaintenanceDestinationExists) {
		return 2
	}
	return 1
}

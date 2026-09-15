package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/config"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/sources/localpath"
	"io"
	"strings"
)

func runSourcesRoots(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources roots", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	jsonOut := fs.Bool("json", false, "write JSON")
	localRoots := repeatedStringFlag{}
	fs.Var(&localRoots, "local-source-root", "allowlisted local source root root_id=path; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath, []string(localRoots)...)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	roots, err := svc.ListLocalPathRoots(ctx)
	if err != nil {
		writeSourceCommandError(stderr, "sources roots", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"roots": roots})
		return 0
	}
	for _, root := range roots {
		fmt.Fprintf(stdout, "%s\talias=%s\n", root.RootID, root.Alias)
	}
	return 0
}

func runSourcesTree(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources tree", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	rootID := fs.String("root", "", "local source root id")
	relativePath := fs.String("path", ".", "root-relative path")
	depth := fs.Int("depth", 1, "tree depth")
	limit := fs.Int("limit", 0, "maximum entries")
	jsonOut := fs.Bool("json", false, "write JSON")
	localRoots := repeatedStringFlag{}
	fs.Var(&localRoots, "local-source-root", "allowlisted local source root root_id=path; repeatable")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources tree <mission_id> --root <root_id> --path <relative_path>")
		return 2
	}
	if err := validateCLIRelativePath(*relativePath); err != nil {
		writeSourceCommandError(stderr, "sources tree", err)
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath, []string(localRoots)...)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	tree, err := svc.BrowseLocalPathRoot(ctx, app.BrowseLocalPathRootRequest{
		RootID:       *rootID,
		RelativePath: *relativePath,
		Depth:        *depth,
		Limit:        *limit,
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources tree", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"mission_id": positionals[0], "tree": tree})
		return 0
	}
	fmt.Fprintf(stdout, "root=%s alias=%s path=%s truncated=%v\n", tree.RootID, tree.RootAlias, tree.RelativePath, tree.Truncated)
	for _, entry := range tree.Entries {
		status := entry.PathKind
		if entry.Denied {
			status = "denied:" + entry.Reason
		}
		fmt.Fprintf(stdout, "%s\t%s\t%s\n", entry.RelativePath, entry.Name, status)
	}
	return 0
}

func runSourcesAttachLocal(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources attach-local", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	rootID := fs.String("root", "", "local source root id")
	relativePath := fs.String("path", "", "root-relative path")
	title := fs.String("title", "", "source title")
	restore := fs.Bool("restore", false, "restore an exact removed local path source")
	jsonOut := fs.Bool("json", false, "write JSON")
	localRoots := repeatedStringFlag{}
	fs.Var(&localRoots, "local-source-root", "allowlisted local source root root_id=path; repeatable")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources attach-local <mission_id> --root <root_id> --path <relative_path>")
		return 2
	}
	if err := validateCLIRelativePath(*relativePath); err != nil {
		writeSourceCommandError(stderr, "sources attach-local", err)
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath, []string(localRoots)...)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	result, err := svc.AttachLocalPathSource(ctx, app.AttachLocalPathSourceRequest{
		MissionID:    positionals[0],
		RootID:       *rootID,
		RelativePath: *relativePath,
		Title:        *title,
		Restore:      *restore,
		Producer:     ledger.Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources attach-local", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{
			"snapshot":         result.Snapshot,
			"event":            result.Event,
			"event_id":         cliLedgerEventID(result.Event),
			"existing":         result.Existing,
			"restored":         result.Restored,
			"restore_required": result.RestoreRequired,
		})
		return 0
	}
	status := "attached"
	if result.Restored {
		status = "restored"
	} else if result.Existing {
		status = "existing"
	}
	fmt.Fprintf(stdout, "%s source %s %s root=%s path=%s event=%s\n",
		status, result.Snapshot.SnapshotID, result.Snapshot.Access.RetrievalPolicy, *rootID, cliSourceRelativePath(result.Snapshot), cliLedgerEventID(result.Event))
	return 0
}

func runSourcesList(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	includeRemoved := fs.Bool("include-removed", false, "include soft-removed sources")
	includeSuperseded := fs.Bool("include-superseded", false, "include superseded source snapshots")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources list <mission_id> [--include-removed] [--include-superseded]")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	sources, err := svc.ListSourceSnapshotsWithState(ctx, sourcecontract.ListRequest{
		MissionID:         positionals[0],
		IncludeRemoved:    *includeRemoved,
		IncludeSuperseded: *includeSuperseded,
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources list", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"sources": sources})
		return 0
	}
	for _, source := range sources {
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n",
			source.SnapshotID, cliSourceState(source), source.Access.RetrievalPolicy, source.Connector.ConnectorType, cliSourceLocatorSummary(source))
	}
	return 0
}

func runSourcesShow(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	includeRemoved := fs.Bool("include-removed", false, "show soft-removed sources")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources show <mission_id> <source_id> [--include-removed]")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	source, err := svc.GetSourceSnapshot(ctx, positionals[1])
	if err != nil {
		writeSourceCommandError(stderr, "sources show", err)
		return cliErrorCode(err)
	}
	if source.MissionID != positionals[0] {
		writeSourceCommandError(stderr, "sources show", fmt.Errorf("%w: source belongs to another mission", app.ErrInvalidInput))
		return 2
	}
	if source.State.Removed && !*includeRemoved {
		writeSourceCommandError(stderr, "sources show", fmt.Errorf("%w: source is removed; pass --include-removed for audit visibility", app.ErrInvalidInput))
		return 2
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"source": source})
		return 0
	}
	fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n",
		source.SnapshotID, cliSourceState(source), source.Access.RetrievalPolicy, source.Connector.ConnectorType, cliSourceLocatorSummary(source))
	return 0
}

func runSourcesRemove(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	reason := fs.String("reason", "", "source removal reason")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources remove <mission_id> <source_id> --reason <reason>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	result, err := svc.RemoveSource(ctx, app.RemoveSourceRequest{
		MissionID:  positionals[0],
		SnapshotID: positionals[1],
		Reason:     *reason,
		Producer:   ledger.Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources remove", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"snapshot": result.Snapshot, "event": result.Event, "event_id": cliLedgerEventID(result.Event), "idempotent": result.Idempotent})
		return 0
	}
	fmt.Fprintf(stdout, "removed source %s event=%s idempotent=%v\n", result.Snapshot.SnapshotID, cliLedgerEventID(result.Event), result.Idempotent)
	return 0
}

func runSourcesRestore(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources restore", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources restore <mission_id> <source_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	result, err := svc.RestoreSource(ctx, app.RestoreSourceRequest{
		MissionID:  positionals[0],
		SnapshotID: positionals[1],
		Producer:   ledger.Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources restore", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"snapshot": result.Snapshot, "event": result.Event, "event_id": cliLedgerEventID(result.Event), "idempotent": result.Idempotent})
		return 0
	}
	fmt.Fprintf(stdout, "restored source %s event=%s idempotent=%v\n", result.Snapshot.SnapshotID, cliLedgerEventID(result.Event), result.Idempotent)
	return 0
}

func newCLIService(store app.Store, cfg config.Config, extraRootSpecs []string) (*app.Service, error) {
	engine, err := localPathEngineFromSpecs(effectiveLocalSourceRootSpecs(cfg, extraRootSpecs))
	if err != nil {
		return nil, err
	}
	if engine == nil {
		return app.NewService(store), nil
	}
	return app.NewServiceWithLocalPathEngine(store, engine), nil
}

func effectiveLocalSourceRootSpecs(cfg config.Config, extraRootSpecs []string) []string {
	specs := make([]string, 0, len(cfg.LocalSourceRoots)+len(extraRootSpecs))
	seen := map[string]struct{}{}
	appendSpec := func(spec string) {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			return
		}
		if _, exists := seen[spec]; exists {
			return
		}
		seen[spec] = struct{}{}
		specs = append(specs, spec)
	}
	for _, spec := range cfg.LocalSourceRoots {
		appendSpec(spec)
	}
	for _, spec := range extraRootSpecs {
		appendSpec(spec)
	}
	return specs
}

func localPathEngineFromSpecs(specs []string) (sourcecontract.LocalPathReader, error) {
	roots := make([]localpath.RootConfig, 0, len(specs))
	for _, spec := range specs {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}
		parts := strings.SplitN(spec, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("%w: local source root must be root_id=path", app.ErrInvalidInput)
		}
		rootID := strings.TrimSpace(parts[0])
		roots = append(roots, localpath.RootConfig{RootID: rootID, Alias: rootID, Path: strings.TrimSpace(parts[1])})
	}
	if len(roots) == 0 {
		return nil, nil
	}
	engine, err := localpath.New(localpath.Config{Roots: roots})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", app.ErrInvalidInput, err)
	}
	return engine, nil
}

func validateCLIRelativePath(relativePath string) error {
	trimmed := strings.TrimSpace(relativePath)
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, `\`) || strings.HasPrefix(trimmed, "~") {
		return fmt.Errorf("%w: relative_path must be root-relative", app.ErrInvalidInput)
	}
	firstSegment := trimmed
	if slash := strings.IndexAny(firstSegment, `/\`); slash >= 0 {
		firstSegment = firstSegment[:slash]
	}
	if strings.Contains(firstSegment, ":") {
		return fmt.Errorf("%w: relative_path must be root-relative", app.ErrInvalidInput)
	}
	if _, err := sourcecontract.NormalizeLocalRelativePath(trimmed); err != nil {
		return fmt.Errorf("%w: %v", app.ErrInvalidInput, err)
	}
	return nil
}

func cliLocalPathLocator(snapshot sourcecontract.Snapshot) (sourcecontract.LocalPathLocator, error) {
	var locators []sourcecontract.LocalPathLocator
	if err := json.Unmarshal(snapshot.Locators, &locators); err != nil {
		return sourcecontract.LocalPathLocator{}, fmt.Errorf("%w: invalid local path locator", app.ErrInvalidInput)
	}
	for _, locator := range locators {
		if cliLocatorType(locator.LocatorType, locator.Kind) == sourcecontract.LocatorTypeLocalPath {
			locator.LocatorType = sourcecontract.LocatorTypeLocalPath
			locator.Kind = ""
			return locator, nil
		}
	}
	return sourcecontract.LocalPathLocator{}, fmt.Errorf("%w: local path locator is required", app.ErrInvalidInput)
}

func cliLocatorType(locatorType, legacyKind string) string {
	if strings.TrimSpace(locatorType) != "" {
		return strings.TrimSpace(locatorType)
	}
	return strings.TrimSpace(legacyKind)
}

func cliSourceRelativePath(snapshot sourcecontract.Snapshot) string {
	locator, err := cliLocalPathLocator(snapshot)
	if err == nil {
		return locator.RelativePath
	}
	return ""
}

func cliSourceLocatorSummary(snapshot sourcecontract.Snapshot) string {
	if snapshot.Connector.ConnectorType == sourcecontract.ConnectorTypeLocalPath {
		locator, err := cliLocalPathLocator(snapshot)
		if err == nil {
			return fmt.Sprintf("root=%s path=%s kind=%s", locator.RootID, locator.RelativePath, locator.PathKind)
		}
	}
	if strings.TrimSpace(snapshot.Connector.ExternalURI) != "" {
		return snapshot.Connector.ExternalURI
	}
	return snapshot.Connector.ExternalSourceID
}

func cliSourceState(snapshot sourcecontract.Snapshot) string {
	if snapshot.State.Removed || snapshot.State.State == sourcecontract.StateRemoved {
		return sourcecontract.StateRemoved
	}
	if snapshot.State.Superseded {
		return "superseded"
	}
	if strings.TrimSpace(snapshot.State.State) != "" {
		return strings.TrimSpace(snapshot.State.State)
	}
	return sourcecontract.StateActive
}

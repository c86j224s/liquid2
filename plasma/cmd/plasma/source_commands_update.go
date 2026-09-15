package main

import "github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"

import (
	"context"
	"flag"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"io"
)

func runSourcesConfluenceCheckUpdate(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence check-update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	apiBaseURL := fs.String("api-base-url", "", "override Confluence API base URL")
	siteURL := fs.String("site-url", "", "override Confluence site URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence check-update <mission_id> <source_id> --connection <id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	cloudID, err := cliConfluenceCloudID(ctx, svc, positionals[1])
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence check-update", err)
		return cliErrorCode(err)
	}
	connector, err := cliConfluenceClient(ctx, svc, *connectionID, cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence check-update", err)
		return cliErrorCode(err)
	}
	result, err := svc.CheckConfluenceSourceUpdateWithEvent(ctx, connector, app.CheckConfluenceSourceUpdateRequest{
		MissionID:  positionals[0],
		SnapshotID: positionals[1],
		EventID:    cliNewID("evt"),
		Producer:   ledger.Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence check-update", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	fmt.Fprintf(stdout, "confluence update_available=%v source=%s old_version=%d new_version=%d event=%s\n",
		result.UpdateAvailable, result.Snapshot.SnapshotID, result.CurrentVersion, result.LatestVersion, cliLedgerEventID(&result.Event))
	return 0
}

func runSourcesConfluenceUpdatePreview(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence update-preview", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	version := fs.Int("version", 0, "expected new Confluence page version")
	maxBodyBytes := fs.Int64("max-body-bytes", app.DefaultConfluenceMaxBodyBytes, "maximum storage body bytes")
	apiBaseURL := fs.String("api-base-url", "", "override Confluence API base URL")
	siteURL := fs.String("site-url", "", "override Confluence site URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence update-preview <mission_id> <source_id> --connection <id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	cloudID, err := cliConfluenceCloudID(ctx, svc, positionals[1])
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence update-preview", err)
		return cliErrorCode(err)
	}
	connector, err := cliConfluenceClient(ctx, svc, *connectionID, cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence update-preview", err)
		return cliErrorCode(err)
	}
	result, err := svc.PreviewConfluenceSourceUpdate(ctx, connector, app.ConfluenceUpdatePreviewRequest{
		MissionID:       positionals[0],
		SnapshotID:      positionals[1],
		ExpectedVersion: *version,
		MaxBodyBytes:    *maxBodyBytes,
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence update-preview", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	fmt.Fprintf(stdout, "confluence update_preview source=%s old_version=%d new_version=%d available=%v requires_range_reselect=%v\n",
		result.Snapshot.SnapshotID, result.OldPage.Version, result.NewPage.Version, result.UpdateAvailable, result.RequiresRangeReselect)
	for _, option := range result.RangeOptions {
		fmt.Fprintf(stdout, "range\t%s\t%d\t%d\n", option.ContentID, option.Start, option.End)
	}
	return 0
}

func runSourcesConfluenceUpdate(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	version := fs.Int("version", 0, "expected new Confluence page version")
	reason := fs.String("reason", "Confluence source update", "update reason")
	maxBodyBytes := fs.Int64("max-body-bytes", app.DefaultConfluenceMaxBodyBytes, "maximum storage body bytes")
	rangeContentID := fs.String("range-content-id", "", "range content id, normally plain_text")
	rangeStart := fs.Int("range-start", 0, "range start rune offset")
	rangeEnd := fs.Int("range-end", 0, "range end rune offset")
	apiBaseURL := fs.String("api-base-url", "", "override Confluence API base URL")
	siteURL := fs.String("site-url", "", "override Confluence site URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence update <mission_id> <source_id> --connection <id> --version <new_version>")
		return 2
	}
	if *version <= 0 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence update <mission_id> <source_id> --connection <id> --version <new_version>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	cloudID, err := cliConfluenceCloudID(ctx, svc, positionals[1])
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence update", err)
		return cliErrorCode(err)
	}
	connector, err := cliConfluenceClient(ctx, svc, *connectionID, cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence update", err)
		return cliErrorCode(err)
	}
	result, err := svc.UpdateConfluenceSourceWithEvent(ctx, connector, app.UpdateConfluenceSourceRequest{
		MissionID:          positionals[0],
		PreviousSnapshotID: positionals[1],
		ArtifactID:         cliNewID("art"),
		SnapshotID:         cliNewID("src"),
		ExpectedVersion:    *version,
		MaxBodyBytes:       *maxBodyBytes,
		Range: confluencesource.ConfluenceRangeSelection{
			ContentID: *rangeContentID,
			Start:     *rangeStart,
			End:       *rangeEnd,
		},
		Reason:          *reason,
		SnapshotEventID: cliNewID("evt"),
		UpdateEventID:   cliNewID("evt"),
		Producer:        ledger.Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence update", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	fmt.Fprintf(stdout, "updated confluence source old=%s new=%s event=%s\n",
		result.PreviousSnapshot.SnapshotID, result.Snapshot.SnapshotID, cliLedgerEventID(&result.UpdateEvent))
	return 0
}

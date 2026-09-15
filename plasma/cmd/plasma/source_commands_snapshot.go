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

func runSourcesConfluencePreview(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence preview", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	cloudID := fs.String("cloud-id", "", "Atlassian cloud id")
	pageID := fs.String("page-id", "", "Confluence page id")
	version := fs.Int("version", 0, "expected Confluence page version")
	maxBodyBytes := fs.Int64("max-body-bytes", app.DefaultConfluenceMaxBodyBytes, "maximum storage body bytes")
	apiBaseURL := fs.String("api-base-url", "", "override Confluence API base URL")
	siteURL := fs.String("site-url", "", "override Confluence site URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence preview <mission_id> --connection <id> --cloud-id <cloud_id> --page-id <page_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connector, err := cliConfluenceClient(ctx, svc, *connectionID, *cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence preview", err)
		return cliErrorCode(err)
	}
	result, err := svc.PreviewConfluenceSource(ctx, connector, app.ConfluenceSourcePreviewRequest{
		MissionID:       positionals[0],
		CloudID:         *cloudID,
		PageID:          *pageID,
		ExpectedVersion: *version,
		MaxBodyBytes:    *maxBodyBytes,
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence preview", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	fmt.Fprintf(stdout, "candidate confluence page=%s version=%d too_large=%v bytes=%d/%d title=%q\n",
		result.Page.PageID, result.Page.Version, result.FullBodyTooLarge, result.BodyBytes, result.MaxBodyBytes, result.Page.Title)
	for _, option := range result.RangeOptions {
		fmt.Fprintf(stdout, "range\t%s\t%d\t%d\n", option.ContentID, option.Start, option.End)
	}
	return 0
}

func runSourcesConfluenceSnapshot(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence snapshot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	cloudID := fs.String("cloud-id", "", "Atlassian cloud id")
	pageID := fs.String("page-id", "", "Confluence page id")
	version := fs.Int("version", 0, "expected Confluence page version")
	reason := fs.String("reason", "", "snapshot reason")
	maxBodyBytes := fs.Int64("max-body-bytes", app.DefaultConfluenceMaxBodyBytes, "maximum storage body bytes")
	rangeContentID := fs.String("range-content-id", "", "range content id, normally plain_text")
	rangeStart := fs.Int("range-start", 0, "range start rune offset")
	rangeEnd := fs.Int("range-end", 0, "range end rune offset")
	apiBaseURL := fs.String("api-base-url", "", "override Confluence API base URL")
	siteURL := fs.String("site-url", "", "override Confluence site URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence snapshot <mission_id> --connection <id> --cloud-id <cloud_id> --page-id <page_id> --version <version>")
		return 2
	}
	if *version <= 0 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence snapshot <mission_id> --connection <id> --cloud-id <cloud_id> --page-id <page_id> --version <version>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connector, err := cliConfluenceClient(ctx, svc, *connectionID, *cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence snapshot", err)
		return cliErrorCode(err)
	}
	result, err := svc.SnapshotConfluenceSourceWithEvent(ctx, connector, app.SnapshotConfluenceSourceWithEventRequest{
		Snapshot: app.SnapshotConfluenceSourceRequest{
			MissionID:       positionals[0],
			ArtifactID:      cliNewID("art"),
			SnapshotID:      cliNewID("src"),
			CloudID:         *cloudID,
			PageID:          *pageID,
			ExpectedVersion: *version,
			MaxBodyBytes:    *maxBodyBytes,
			Range: confluencesource.ConfluenceRangeSelection{
				ContentID: *rangeContentID,
				Start:     *rangeStart,
				End:       *rangeEnd,
			},
			Reason: *reason,
		},
		EventID:  cliNewID("evt"),
		Producer: ledger.Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence snapshot", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	fmt.Fprintf(stdout, "snapshotted confluence source %s artifact=%s event=%s\n",
		result.Snapshot.SnapshotID, result.Artifact.ArtifactID, cliLedgerEventID(&result.Event))
	return 0
}

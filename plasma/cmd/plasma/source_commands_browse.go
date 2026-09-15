package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"io"
)

func runSourcesConfluenceSpaces(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence spaces", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	cloudID := fs.String("cloud-id", "", "Atlassian cloud id")
	limit := fs.Int("limit", 10, "maximum spaces")
	cursor := fs.String("cursor", "", "Confluence cursor")
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
		fmt.Fprintln(stderr, "usage: plasma sources confluence spaces <mission_id> --connection <id> --cloud-id <cloud_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connector, err := cliConfluenceBrowserConnector(ctx, svc, *connectionID, *cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence spaces", err)
		return cliErrorCode(err)
	}
	result, err := svc.ListConfluenceSpaces(ctx, connector, confluencesource.ConfluenceSpaceListRequest{MissionID: positionals[0], CloudID: *cloudID, Limit: *limit, Cursor: *cursor})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence spaces", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	for _, space := range result.Spaces {
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", space.SpaceID, space.SpaceKey, space.Name, space.WebURL)
	}
	if result.NextCursor != "" {
		fmt.Fprintf(stdout, "next_cursor=%s\n", result.NextCursor)
	}
	return 0
}

func runSourcesConfluencePages(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence pages", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	cloudID := fs.String("cloud-id", "", "Atlassian cloud id")
	spaceID := fs.String("space-id", "", "Confluence space id")
	limit := fs.Int("limit", 10, "maximum pages")
	cursor := fs.String("cursor", "", "Confluence cursor")
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
		fmt.Fprintln(stderr, "usage: plasma sources confluence pages <mission_id> --connection <id> --cloud-id <cloud_id> --space-id <space_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connector, err := cliConfluenceBrowserConnector(ctx, svc, *connectionID, *cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence pages", err)
		return cliErrorCode(err)
	}
	result, err := svc.ListConfluenceSpacePages(ctx, connector, confluencesource.ConfluenceSpacePagesRequest{MissionID: positionals[0], CloudID: *cloudID, SpaceID: *spaceID, Limit: *limit, Cursor: *cursor})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence pages", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	writeConfluencePages(stdout, result)
	return 0
}

func runSourcesConfluenceChildren(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence children", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	cloudID := fs.String("cloud-id", "", "Atlassian cloud id")
	pageID := fs.String("page-id", "", "Confluence parent page id")
	limit := fs.Int("limit", 10, "maximum pages")
	cursor := fs.String("cursor", "", "Confluence cursor")
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
		fmt.Fprintln(stderr, "usage: plasma sources confluence children <mission_id> --connection <id> --cloud-id <cloud_id> --page-id <page_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connector, err := cliConfluenceBrowserConnector(ctx, svc, *connectionID, *cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence children", err)
		return cliErrorCode(err)
	}
	result, err := svc.ListConfluencePageChildren(ctx, connector, confluencesource.ConfluencePageChildrenRequest{MissionID: positionals[0], CloudID: *cloudID, PageID: *pageID, Limit: *limit, Cursor: *cursor})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence children", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	writeConfluencePages(stdout, result)
	return 0
}

func writeConfluencePages(stdout io.Writer, result confluencesource.ConfluencePageListResult) {
	for _, page := range result.Pages {
		fmt.Fprintf(stdout, "%s\tv%d\t%s\t%s\n", page.PageID, page.Version, page.Title, page.WebURL)
	}
	if result.NextCursor != "" {
		fmt.Fprintf(stdout, "next_cursor=%s\n", result.NextCursor)
	}
}

func runSourcesConfluenceSearch(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence search", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	cloudID := fs.String("cloud-id", "", "Atlassian cloud id")
	query := fs.String("query", "", "Confluence search text")
	spaceKey := fs.String("space-key", "", "Confluence space key")
	limit := fs.Int("limit", 10, "maximum candidates")
	cursor := fs.String("cursor", "", "Confluence cursor")
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
		fmt.Fprintln(stderr, "usage: plasma sources confluence search <mission_id> --connection <id> --cloud-id <cloud_id> --query <text>")
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
		writeSourceCommandError(stderr, "sources confluence search", err)
		return cliErrorCode(err)
	}
	result, err := svc.SearchConfluenceSources(ctx, connector, confluencesource.ConfluenceSourceSearchRequest{
		MissionID: positionals[0],
		CloudID:   *cloudID,
		Query:     *query,
		Limit:     *limit,
		Cursor:    *cursor,
		SpaceKey:  *spaceKey,
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence search", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	for _, candidate := range result.Candidates {
		fmt.Fprintf(stdout, "%s\tv%d\t%s\t%s\n", candidate.Connector.ExternalSourceID, candidate.Version, candidate.Title, candidate.SourceURI)
	}
	if result.NextCursor != "" {
		fmt.Fprintf(stdout, "next_cursor=%s\n", result.NextCursor)
	}
	return 0
}

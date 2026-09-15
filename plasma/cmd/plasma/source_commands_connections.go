package main

import (
	"context"
	"flag"
	"fmt"
	"io"
)

func runSourcesConfluenceConnections(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence connections", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	jsonOut := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connections, err := svc.ListConfluenceConnections(ctx)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence connections", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"connections": connections})
		return 0
	}
	for _, connection := range connections {
		fmt.Fprintf(stdout, "%s\t%s\t%s\tsites=%d\trevoked=%v\n",
			connection.ConnectionID, connection.AuthType, connection.DisplayName, len(connection.Sites), connection.Revoked)
	}
	return 0
}

func runSourcesConfluenceRenameConnection(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence rename-connection", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	name := fs.String("name", "", "new display name")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence rename-connection <connection_id> --name <display_name>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connection, err := svc.RenameConfluenceConnection(ctx, positionals[0], *name)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence rename-connection", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"connection": connection})
		return 0
	}
	fmt.Fprintf(stdout, "renamed confluence connection %s name=%q\n", connection.ConnectionID, connection.DisplayName)
	return 0
}

func runSourcesConfluenceRevokeConnection(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence revoke-connection", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence revoke-connection <connection_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connection, err := svc.RevokeConfluenceConnection(ctx, positionals[0])
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence revoke-connection", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"connection": connection, "revoked": true})
		return 0
	}
	fmt.Fprintf(stdout, "revoked confluence connection %s\n", connection.ConnectionID)
	return 0
}

func runSourcesConfluenceDeleteConnection(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence delete-connection", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence delete-connection <connection_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	if err := svc.DeleteConfluenceConnection(ctx, positionals[0]); err != nil {
		writeSourceCommandError(stderr, "sources confluence delete-connection", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"connection_id": positionals[0], "deleted": true})
		return 0
	}
	fmt.Fprintf(stdout, "deleted confluence connection %s\n", positionals[0])
	return 0
}

func runSourcesConfluenceSites(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence sites", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	refresh := fs.Bool("refresh", false, "refresh sites from Atlassian accessible-resources")
	discoveryURL := fs.String("discovery-url", "", "override Atlassian discovery base URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connection, err := svc.GetConfluenceConnection(ctx, *connectionID)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence sites", err)
		return cliErrorCode(err)
	}
	if *refresh {
		lister, err := cliConfluenceDiscoveryClient(connection, *discoveryURL, *allowOAuthOverrides)
		if err != nil {
			writeSourceCommandError(stderr, "sources confluence sites", err)
			return cliErrorCode(err)
		}
		connection, err = svc.RefreshConfluenceConnectionSites(ctx, connection.ConnectionID, lister)
		if err != nil {
			writeSourceCommandError(stderr, "sources confluence sites", err)
			return cliErrorCode(err)
		}
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"connection": connection, "sites": connection.Sites})
		return 0
	}
	for _, site := range connection.Sites {
		fmt.Fprintf(stdout, "%s\t%s\t%s\n", site.CloudID, site.Name, site.URL)
	}
	return 0
}

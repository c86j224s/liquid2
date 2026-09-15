package main

import (
	"context"
	"fmt"
	"io"
)

func runSourcesConfluence(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printSourcesConfluenceUsage(stderr)
		return 2
	}
	switch args[0] {
	case "oauth-url":
		return runSourcesConfluenceOAuthURL(ctx, args[1:], stdout, stderr)
	case "oauth-exchange":
		return runSourcesConfluenceOAuthExchange(ctx, args[1:], stdout, stderr)
	case "connect-token", "connect-oauth-token":
		return runSourcesConfluenceConnectToken(ctx, args[1:], stdout, stderr)
	case "connections":
		return runSourcesConfluenceConnections(ctx, args[1:], stdout, stderr)
	case "rename-connection":
		return runSourcesConfluenceRenameConnection(ctx, args[1:], stdout, stderr)
	case "revoke-connection":
		return runSourcesConfluenceRevokeConnection(ctx, args[1:], stdout, stderr)
	case "delete-connection":
		return runSourcesConfluenceDeleteConnection(ctx, args[1:], stdout, stderr)
	case "sites":
		return runSourcesConfluenceSites(ctx, args[1:], stdout, stderr)
	case "spaces":
		return runSourcesConfluenceSpaces(ctx, args[1:], stdout, stderr)
	case "pages":
		return runSourcesConfluencePages(ctx, args[1:], stdout, stderr)
	case "children":
		return runSourcesConfluenceChildren(ctx, args[1:], stdout, stderr)
	case "search":
		return runSourcesConfluenceSearch(ctx, args[1:], stdout, stderr)
	case "preview":
		return runSourcesConfluencePreview(ctx, args[1:], stdout, stderr)
	case "snapshot":
		return runSourcesConfluenceSnapshot(ctx, args[1:], stdout, stderr)
	case "check-update":
		return runSourcesConfluenceCheckUpdate(ctx, args[1:], stdout, stderr)
	case "update-preview":
		return runSourcesConfluenceUpdatePreview(ctx, args[1:], stdout, stderr)
	case "update":
		return runSourcesConfluenceUpdate(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown sources confluence command %q\n", args[0])
		printSourcesConfluenceUsage(stderr)
		return 2
	}
}

func printSourcesConfluenceUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: plasma sources confluence <connect-token|connections|rename-connection|revoke-connection|delete-connection|sites|spaces|pages|children|search|preview|snapshot|check-update|update-preview|update> [options]")
	fmt.Fprintln(w, "note: Confluence OAuth commands are disabled in Plasma 0.0; use API token connections.")
}

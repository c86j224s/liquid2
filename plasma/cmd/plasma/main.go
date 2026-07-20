package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/c86j224s/liquid2/plasma/internal/version"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "version":
		fmt.Fprintln(stdout, version.Version)
		return 0
	case "health":
		return runHealth(ctx, args[1:], stdout, stderr)
	case "missions":
		return runMissions(ctx, args[1:], stdout, stderr)
	case "turns":
		return runTurns(ctx, args[1:], stdout, stderr)
	case "sources":
		return runSources(ctx, args[1:], stdout, stderr)
	case "workflow":
		return runWorkflow(ctx, args[1:], stdout, stderr)
	case "reports":
		return runReports(ctx, args[1:], stdout, stderr)
	case "storage":
		return runStorage(ctx, args[1:], stdout, stderr)
	case "mcp":
		return runMCP(ctx, args[1:], os.Stdin, stdout, stderr)
	case "status":
		return runStatus(ctx, args[1:], stdout, stderr)
	case "serve":
		return runServe(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: plasma <version|health|missions|turns|sources|workflow|reports|storage|mcp|status|serve> [options]")
}

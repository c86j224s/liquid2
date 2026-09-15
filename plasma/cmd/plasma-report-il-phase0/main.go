package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportilpdf"
	"github.com/c86j224s/liquid2/plasma/internal/reportilphase0"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("plasma-report-il-phase0", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var archiveRoot string
	var bundlePath string
	var runID string
	var chromePath string
	var requirePDF bool
	flags.StringVar(&archiveRoot, "archive-root", "", "durable Phase 0 archive root outside the repository")
	flags.StringVar(&bundlePath, "bundle", "", "T0, T1, or T2 input bundle under the archive root")
	flags.StringVar(&runID, "run-id", "", "new immutable run directory ID")
	flags.StringVar(&chromePath, "chrome-path", "", "optional Chrome or Chromium executable")
	flags.BoolVar(&requirePDF, "require-pdf", false, "fail the command after preserving a PDF blocker packet")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(archiveRoot) == "" || strings.TrimSpace(bundlePath) == "" || strings.TrimSpace(runID) == "" {
		fmt.Fprintln(stderr, "archive-root, bundle, and run-id are required")
		return 2
	}
	repoRoot, err := findRepositoryRoot()
	if err != nil {
		fmt.Fprintf(stderr, "repository root: %v\n", err)
		return 1
	}
	result, err := reportilphase0.Run(ctx, reportilphase0.RunConfig{
		ArchiveRoot: archiveRoot, RepositoryRoot: repoRoot, BundlePath: bundlePath,
		RunID: runID, PDFRenderer: reportilpdf.Chrome{ChromePath: chromePath}, RequirePDF: requirePDF,
	})
	if result.RunDirectory != "" {
		fmt.Fprintf(stdout, "run_dir=%s\n", result.RunDirectory)
	}
	if result.ManifestPath != "" {
		fmt.Fprintf(stdout, "manifest=%s\n", result.ManifestPath)
	}
	if result.MarkdownPath != "" {
		fmt.Fprintf(stdout, "markdown=%s\nhtml=%s\n", result.MarkdownPath, result.HTMLPath)
	}
	if result.PDFPath != "" {
		fmt.Fprintf(stdout, "pdf=%s\n", result.PDFPath)
	}
	if result.BlockerPath != "" {
		fmt.Fprintf(stdout, "pdf_blocker=%s\n", result.BlockerPath)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func findRepositoryRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root")
		}
		dir = parent
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportexperiment"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

type commandConfig struct {
	archiveRoot     string
	fixturePath     string
	runID           string
	plasmaMCPBinary string
	codexCommand    string
	codexModel      string
	codexEffort     string
	codexTimeout    time.Duration
}

var runExperiment = reportexperiment.Run

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	config, err := parseFlags(args, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	repoRoot, err := findRepositoryRoot()
	if err != nil {
		fmt.Fprintf(stderr, "repository root: %v\n", err)
		return 1
	}
	experimentBinary, err := reportexperiment.CurrentBinaryMetadata("")
	if err != nil {
		fmt.Fprintf(stderr, "experiment binary metadata: %v\n", err)
		return 1
	}
	plasmaMCPBinary, err := reportexperiment.ReadExecutableMetadata(config.plasmaMCPBinary)
	if err != nil {
		fmt.Fprintf(stderr, "plasma MCP binary metadata: %v\n", err)
		return 1
	}
	config.plasmaMCPBinary = plasmaMCPBinary.Path
	codexBinary, err := reportexperiment.ReadExecutableMetadata(agentexec.ResolveAgentCommand(config.codexCommand))
	if err != nil {
		fmt.Fprintf(stderr, "Codex command metadata: %v\n", err)
		return 1
	}
	config.codexCommand = codexBinary.Path
	binaries := reportexperiment.BinaryPair{Experiment: experimentBinary, PlasmaMCP: plasmaMCPBinary, Codex: codexBinary}
	if err := reportexperiment.ValidateBinaryPair(binaries); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	result, err := runExperiment(ctx, reportexperiment.Config{
		ArchiveRoot: config.archiveRoot, FixturePath: config.fixturePath, RunID: config.runID, RepositoryRoot: repoRoot,
		AgentModel: config.codexModel, AgentReasoningEffort: config.codexEffort, BinaryPair: binaries,
		ServiceFactory: openService,
		ExecutorFactory: func(_ context.Context, input reportexperiment.ExecutorContext) (agentexec.AgentExecutor, error) {
			return codexExecutor(config, input)
		},
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "run_dir=%s\nreport=%s\nledger=%s\nmanifest=%s\n", result.RunDir, result.ReportPath, result.LedgerPath, result.ManifestPath)
	return 0
}

func openService(ctx context.Context, dbPath string) (reportexperiment.ServiceHandle, error) {
	store, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		return reportexperiment.ServiceHandle{}, err
	}
	return reportexperiment.ServiceHandle{Service: app.NewService(store), Close: store.Close}, nil
}

func parseFlags(args []string, stderr io.Writer) (commandConfig, error) {
	var config commandConfig
	flags := flag.NewFlagSet("plasma-report-experiment", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&config.archiveRoot, "archive-root", "", "archive root outside the repository")
	flags.StringVar(&config.fixturePath, "fixture", "", "fixed reviewed Part fixture JSON")
	flags.StringVar(&config.runID, "run-id", "", "unique run ID")
	flags.StringVar(&config.plasmaMCPBinary, "plasma-mcp-binary", "", "Plasma MCP server binary")
	flags.StringVar(&config.codexCommand, "codex-command", "codex", "Codex CLI command")
	flags.StringVar(&config.codexModel, "codex-model", "", "Codex model")
	flags.StringVar(&config.codexEffort, "codex-effort", "", "Codex reasoning effort")
	flags.DurationVar(&config.codexTimeout, "codex-timeout", 0, "per Codex stage timeout")
	if err := flags.Parse(args); err != nil {
		return commandConfig{}, err
	}
	if strings.TrimSpace(config.archiveRoot) == "" ||
		strings.TrimSpace(config.fixturePath) == "" ||
		strings.TrimSpace(config.runID) == "" ||
		strings.TrimSpace(config.plasmaMCPBinary) == "" ||
		strings.TrimSpace(config.codexCommand) == "" ||
		strings.TrimSpace(config.codexModel) == "" ||
		strings.TrimSpace(config.codexEffort) == "" ||
		config.codexTimeout <= 0 {
		return commandConfig{}, fmt.Errorf("archive-root, fixture, run-id, plasma-mcp-binary, codex-command, codex-model, codex-effort, and positive codex-timeout are required")
	}
	return config, nil
}

func codexExecutor(config commandConfig, input reportexperiment.ExecutorContext) (agentexec.CodexExecutor, error) {
	workDir := filepath.Join(input.RunDir, "provider", "work")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return agentexec.CodexExecutor{}, err
	}
	return agentexec.CodexExecutor{
		Command: strings.TrimSpace(config.codexCommand),
		WorkDir: workDir,
		Timeout: config.codexTimeout,
		MCPServer: agentexec.CodexMCPServer{
			Name: "plasma", Command: strings.TrimSpace(config.plasmaMCPBinary), Args: []string{"mcp", "-db", input.DBPath},
			Required: true, StartupTimeoutSec: 30, ToolTimeoutSec: int(config.codexTimeout.Seconds()),
		},
	}, nil
}

func findRepositoryRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return findRepositoryRootFrom(dir)
}

func findRepositoryRootFrom(dir string) (string, error) {
	dir = filepath.Clean(dir)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root from %s", dir)
		}
		dir = parent
	}
}

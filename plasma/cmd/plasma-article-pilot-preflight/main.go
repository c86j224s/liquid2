package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/articleexperiment"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("plasma-article-pilot-preflight", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var archiveRoot, repositoryRoot, protocolPath, protocolSHA, matrixSHA, failRun string
	var synthetic, verify bool
	flags.StringVar(&archiveRoot, "archive-root", "", "dedicated Issue 461 preflight archive root")
	flags.StringVar(&repositoryRoot, "repository-root", "", "repository root excluded from archive inputs and outputs")
	flags.StringVar(&protocolPath, "protocol", "", "frozen protocol JSON under the archive root")
	flags.StringVar(&protocolSHA, "protocol-sha256", "", "exact lowercase SHA-256 of the frozen protocol JSON")
	flags.StringVar(&matrixSHA, "matrix-sha256", "", "exact lowercase SHA-256 of an existing matrix terminal")
	flags.StringVar(&failRun, "fail-run", "", "preassigned run ID that must retain a synthetic failure")
	flags.BoolVar(&synthetic, "synthetic", false, "run the deterministic no-provider preflight executor")
	flags.BoolVar(&verify, "verify", false, "verify an existing matrix without executing cells")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	commonReady := strings.TrimSpace(archiveRoot) != "" && strings.TrimSpace(repositoryRoot) != "" && strings.TrimSpace(protocolPath) != "" && strings.TrimSpace(protocolSHA) != ""
	syntheticReady := synthetic && !verify && strings.TrimSpace(failRun) != "" && strings.TrimSpace(matrixSHA) == ""
	verifyReady := verify && !synthetic && strings.TrimSpace(matrixSHA) != "" && strings.TrimSpace(failRun) == ""
	if !commonReady || flags.NArg() != 0 || syntheticReady == verifyReady {
		fmt.Fprintln(stderr, "choose exactly one of synthetic or verify and provide only its required digest/run options")
		return 2
	}
	if verify {
		manifest, err := articleexperiment.ValidateMatrix(archiveRoot, repositoryRoot, protocolPath, strings.TrimSpace(protocolSHA), strings.TrimSpace(matrixSHA))
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "verified=true\ncompleted=%d\nfailed=%d\n", manifest.Completed, manifest.Failed)
		return 0
	}
	executor := syntheticExecutor{failRun: strings.TrimSpace(failRun)}
	result, err := articleexperiment.RunMatrix(ctx, articleexperiment.MatrixConfig{
		ArchiveRoot: archiveRoot, RepositoryRoot: repositoryRoot,
		ProtocolPath: protocolPath, ProtocolSHA256: strings.TrimSpace(protocolSHA),
		ExpectedFailedRun: strings.TrimSpace(failRun), Executor: executor,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "matrix=%s\nmatrix_sha256=%s\ncompleted=%d\nfailed=%d\n", result.ManifestPath, result.ManifestSHA256, result.Completed, result.Failed)
	return 0
}

type syntheticExecutor struct {
	failRun string
}

func (executor syntheticExecutor) Execute(_ context.Context, input articleexperiment.ExecutionInput) (articleexperiment.ExecutionOutput, error) {
	attempts := []articleexperiment.AttemptReceipt{{Attempt: 1, Kind: "semantic", Outcome: "completed"}}
	if input.Cell.RunID == executor.failRun {
		attempts[0].Outcome = "failed"
		return articleexperiment.ExecutionOutput{Attempts: attempts}, fmt.Errorf("forced synthetic failure")
	}
	body := []byte(fmt.Sprintf("# Synthetic %s/%s\n\n%s\n", input.Fixture.FixtureID, input.Arm.ArmID, input.Fixture.ReaderPromise))
	return articleexperiment.ExecutionOutput{
		Attempts: attempts,
		Artifacts: []articleexperiment.OutputArtifact{{
			Kind: "synthetic_markdown", MediaType: "text/markdown; charset=utf-8",
			Filename: "article.md", Content: body,
		}},
	}, nil
}

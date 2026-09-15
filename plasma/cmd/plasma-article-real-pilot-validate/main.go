package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/articleexperiment"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("plasma-article-real-pilot-validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var archiveRoot, repositoryRoot, protocolPath, protocolSHA string
	flags.StringVar(&archiveRoot, "archive-root", "", "dedicated Issue 461 real-pilot archive root")
	flags.StringVar(&repositoryRoot, "repository-root", "", "repository root excluded from pilot inputs and outputs")
	flags.StringVar(&protocolPath, "protocol", "", "frozen real-pilot protocol JSON")
	flags.StringVar(&protocolSHA, "protocol-sha256", "", "exact lowercase SHA-256 of the protocol JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || strings.TrimSpace(archiveRoot) == "" || strings.TrimSpace(repositoryRoot) == "" || strings.TrimSpace(protocolPath) == "" || strings.TrimSpace(protocolSHA) == "" {
		fmt.Fprintln(stderr, "provide archive-root, repository-root, protocol, and protocol-sha256 with no positional arguments")
		return 2
	}
	loaded, err := articleexperiment.LoadRealProtocol(archiveRoot, repositoryRoot, protocolPath, strings.TrimSpace(protocolSHA))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "bundle_validated=true\nissue_lock_verified=false\nprovider_ready=false\nprotocol_id=%s\nprotocol_sha256=%s\nfixtures=%d\narms=%d\nruns=%d\n", loaded.Protocol.ProtocolID, loaded.ProtocolSHA256, loaded.FixtureCount(), loaded.ArmCount(), len(loaded.Protocol.Runs))
	return 0
}

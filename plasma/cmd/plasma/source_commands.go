package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"os"
	"path/filepath"
	"strings"
)

const (
	cliDefaultReadBytes = int64(20 * 1024)
	cliMaxReadBytes     = int64(256 * 1024)
)

type repeatedStringFlag []string

func (flagValue *repeatedStringFlag) Set(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		*flagValue = append(*flagValue, trimmed)
	}
	return nil
}

func (flagValue repeatedStringFlag) String() string {
	return strings.Join(flagValue, ",")
}

func runSources(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printSourcesUsage(stderr)
		return 2
	}
	switch args[0] {
	case "roots":
		return runSourcesRoots(ctx, args[1:], stdout, stderr)
	case "tree":
		return runSourcesTree(ctx, args[1:], stdout, stderr)
	case "attach-local":
		return runSourcesAttachLocal(ctx, args[1:], stdout, stderr)
	case "upload":
		return runSourcesUpload(ctx, args[1:], stdout, stderr)
	case "list":
		return runSourcesList(ctx, args[1:], stdout, stderr)
	case "show":
		return runSourcesShow(ctx, args[1:], stdout, stderr)
	case "read":
		return runSourcesRead(ctx, args[1:], stdout, stderr)
	case "grep":
		return runSourcesGrep(ctx, args[1:], stdout, stderr)
	case "remove":
		return runSourcesRemove(ctx, args[1:], stdout, stderr)
	case "restore":
		return runSourcesRestore(ctx, args[1:], stdout, stderr)
	case "confluence":
		return runSourcesConfluence(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown sources command %q\n", args[0])
		printSourcesUsage(stderr)
		return 2
	}
}

func runSourcesUpload(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources upload", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	title := fs.String("title", "", "source title")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources upload <mission_id> <path> [--title title] [--json]")
		return 2
	}
	path := strings.TrimSpace(positionals[1])
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(stderr, "stat upload file: %v\n", err)
		return 1
	}
	if info.IsDir() {
		fmt.Fprintln(stderr, "upload path must be a file")
		return 2
	}
	if info.Size() > app.UploadedFileMaxBytes {
		fmt.Fprintln(stderr, "upload file exceeds 100 MiB limit")
		return 2
	}
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "read upload file: %v\n", err)
		return 1
	}
	if int64(len(content)) > app.UploadedFileMaxBytes {
		fmt.Fprintln(stderr, "upload file exceeds 100 MiB limit")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	result, err := svc.CreateUploadedFileSourceWithEvent(ctx, app.CreateUploadedFileSourceRequest{
		MissionID:        positionals[0],
		ArtifactID:       cliNewID("art"),
		SnapshotID:       cliNewID("src"),
		EventID:          cliNewID("evt"),
		Title:            *title,
		OriginalFilename: filepath.Base(path),
		Content:          content,
		Producer:         ledger.Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources upload", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{
			"artifact": cliRawArtifactResponse(result.Artifact),
			"snapshot": result.Snapshot,
			"event":    result.Event,
			"existing": result.Existing,
		})
		return 0
	}
	status := "uploaded"
	if result.Existing {
		status = "existing"
	}
	fmt.Fprintf(stdout, "%s file source %s artifact=%s sha256=%s event=%s\n",
		status, result.Snapshot.SnapshotID, result.Artifact.ArtifactID, result.Artifact.SHA256, cliLedgerEventID(&result.Event))
	return 0
}

func printSourcesUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: plasma sources <roots|tree|attach-local|upload|list|show|read|grep|remove|restore|confluence> [options]")
}

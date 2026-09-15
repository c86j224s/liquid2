package main

import uploadsource "github.com/c86j224s/liquid2/plasma/internal/source"

import (
	"context"
	"flag"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/pdfdocument"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"io"
	"strings"
	"unicode/utf8"
)

func runSourcesRead(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources read", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	artifactID := fs.String("artifact", "", "artifact id for snapshot sources")
	offset := fs.Int64("offset", 0, "read offset")
	maxBytes := fs.Int64("max-bytes", 0, "maximum bytes")
	depth := fs.Int("depth", 1, "directory tree depth for live directory sources")
	limit := fs.Int("limit", 0, "directory tree entry limit for live directory sources")
	jsonOut := fs.Bool("json", false, "write JSON")
	localRoots := repeatedStringFlag{}
	fs.Var(&localRoots, "local-source-root", "allowlisted local source root root_id=path; repeatable")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources read <mission_id> <source_id> [--offset N --max-bytes N]")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath, []string(localRoots)...)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	snapshot, err := svc.GetSourceSnapshot(ctx, positionals[1])
	if err != nil {
		writeSourceCommandError(stderr, "sources read", err)
		return cliErrorCode(err)
	}
	if snapshot.MissionID != positionals[0] {
		writeSourceCommandError(stderr, "sources read", fmt.Errorf("%w: source belongs to another mission", app.ErrInvalidInput))
		return 2
	}
	if snapshot.State.Removed {
		writeSourceCommandError(stderr, "sources read", fmt.Errorf("%w: source is removed", app.ErrInvalidInput))
		return 2
	}
	if snapshot.Access.RetrievalPolicy == sourcecontract.RetrievalPolicyLiveReference && snapshot.Connector.ConnectorType == sourcecontract.ConnectorTypeLocalPath {
		return runSourcesReadLive(ctx, svc, positionals[0], snapshot, *offset, *maxBytes, *depth, *limit, *jsonOut, stdout, stderr)
	}
	return runSourcesReadSnapshot(ctx, svc, positionals[0], snapshot, *artifactID, *offset, *maxBytes, *jsonOut, stdout, stderr)
}

func runSourcesReadLive(ctx context.Context, svc *app.Service, missionID string, snapshot sourcecontract.Snapshot, offset int64, maxBytes int64, depth int, limit int, jsonOut bool, stdout, stderr io.Writer) int {
	locator, err := cliLocalPathLocator(snapshot)
	if err != nil {
		writeSourceCommandError(stderr, "sources read", err)
		return cliErrorCode(err)
	}
	if locator.PathKind == "directory" {
		result, err := svc.TreeLocalPathSource(ctx, app.TreeLocalPathSourceRequest{
			MissionID:     missionID,
			SnapshotID:    snapshot.SnapshotID,
			Depth:         depth,
			Limit:         limit,
			Producer:      ledger.Producer{Type: "user", ID: "plasma-cli"},
			ToolSessionID: "plasma-cli",
		})
		if err != nil {
			writeSourceCommandError(stderr, "sources read", err)
			return cliErrorCode(err)
		}
		if jsonOut {
			writeCLIJSON(stdout, map[string]any{
				"snapshot":             result.Snapshot,
				"tree":                 result.Tree,
				"observation_metadata": result.Tree.Metadata,
				"observation_event":    result.ObservationEvent,
				"observation_event_id": cliLedgerEventID(result.ObservationEvent),
			})
			return 0
		}
		fmt.Fprintf(stdout, "observation_event=%s root=%s path=%s truncated=%v\n",
			cliLedgerEventID(result.ObservationEvent), result.Tree.RootID, result.Tree.RelativePath, result.Tree.Truncated)
		for _, entry := range result.Tree.Entries {
			fmt.Fprintf(stdout, "%s\t%s\n", entry.RelativePath, entry.PathKind)
		}
		return 0
	}
	result, err := svc.ReadLocalPathSource(ctx, app.ReadLocalPathSourceRequest{
		MissionID:     missionID,
		SnapshotID:    snapshot.SnapshotID,
		Offset:        offset,
		MaxBytes:      maxBytes,
		Producer:      ledger.Producer{Type: "user", ID: "plasma-cli"},
		ToolSessionID: "plasma-cli",
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources read", err)
		return cliErrorCode(err)
	}
	if jsonOut {
		writeCLIJSON(stdout, map[string]any{
			"snapshot":             result.Snapshot,
			"content":              result.Read.Content,
			"observation_metadata": result.Read.Metadata,
			"observation_event":    result.ObservationEvent,
			"observation_event_id": cliLedgerEventID(result.ObservationEvent),
		})
		return 0
	}
	fmt.Fprint(stdout, result.Read.Content)
	if result.Read.Metadata.Truncated {
		fmt.Fprintf(stdout, "\n[next_offset=%d]\n", result.Read.Metadata.NextOffset)
	}
	fmt.Fprintf(stdout, "\nobservation_event=%s\n", cliLedgerEventID(result.ObservationEvent))
	return 0
}

func runSourcesReadSnapshot(ctx context.Context, svc *app.Service, missionID string, snapshot sourcecontract.Snapshot, artifactID string, offset int64, maxBytes int64, jsonOut bool, stdout, stderr io.Writer) int {
	artifactID = strings.TrimSpace(artifactID)
	if artifactID == "" && len(snapshot.ArtifactIDs) == 1 {
		artifactID = snapshot.ArtifactIDs[0]
	}
	if artifactID == "" {
		writeSourceCommandError(stderr, "sources read", fmt.Errorf("%w: artifact id is required for snapshot sources", app.ErrInvalidInput))
		return 2
	}
	artifact, err := svc.GetRawArtifact(ctx, artifactID)
	if err != nil {
		writeSourceCommandError(stderr, "sources read", err)
		return cliErrorCode(err)
	}
	if artifact.MissionID != missionID {
		writeSourceCommandError(stderr, "sources read", fmt.Errorf("%w: artifact belongs to another mission", app.ErrInvalidInput))
		return 2
	}
	read, err := cliReadSourceArtifact(artifact, offset, maxBytes)
	if err != nil {
		writeSourceCommandError(stderr, "sources read", err)
		return 2
	}
	if jsonOut {
		response := map[string]any{
			"snapshot":    snapshot,
			"artifact":    cliRawArtifactMetadata(artifact),
			"content":     read.Content,
			"offset":      read.Offset,
			"next_offset": read.NextOffset,
			"truncated":   read.Truncated,
		}
		if read.ExtractionType != "" {
			response["content_length"] = read.ContentLength
			response["extraction"] = map[string]any{"type": read.ExtractionType, "page_count": read.PageCount}
		}
		if read.MetadataOnly {
			response["content_length"] = artifact.ByteSize
			response["metadata_only"] = true
		}
		writeCLIJSON(stdout, response)
		return 0
	}
	if read.MetadataOnly {
		fmt.Fprintf(stdout, "metadata-only source artifact %s media_type=%s byte_size=%d sha256=%s\n",
			artifact.ArtifactID, artifact.MediaType, artifact.ByteSize, artifact.SHA256)
		return 0
	}
	fmt.Fprint(stdout, read.Content)
	if read.Truncated {
		fmt.Fprintf(stdout, "\n[next_offset=%d]\n", read.NextOffset)
	}
	return 0
}

type cliArtifactRead struct {
	Content        string
	Offset         int64
	NextOffset     int64
	ContentLength  int
	Truncated      bool
	ExtractionType string
	PageCount      int
	MetadataOnly   bool
}

func cliReadSourceArtifact(artifact artifactcontract.Raw, offset int64, maxBytes int64) (cliArtifactRead, error) {
	if pdfdocument.IsPDFMediaType(artifact.MediaType) || pdfdocument.IsPDFBytes(artifact.Content) {
		chunk, err := pdfdocument.ExtractChunk(artifact.Content, int(offset), int(maxBytes))
		if err != nil {
			return cliArtifactRead{}, fmt.Errorf("%w: PDF text extraction failed: %v", app.ErrInvalidInput, err)
		}
		return cliArtifactRead{
			Content:        chunk.Text,
			Offset:         int64(chunk.Offset),
			NextOffset:     int64(chunk.NextOffset),
			ContentLength:  chunk.ContentLength,
			Truncated:      chunk.Truncated,
			ExtractionType: "pdf_text",
			PageCount:      chunk.PageCount,
		}, nil
	}
	if uploadsource.UploadedArtifactReadKind(artifact) == "metadata" {
		return cliArtifactRead{ContentLength: int(artifact.ByteSize), MetadataOnly: true}, nil
	}
	return cliReadArtifactContent(artifact.Content, offset, maxBytes)
}

func cliReadArtifactContent(content []byte, offset int64, maxBytes int64) (cliArtifactRead, error) {
	if !utf8.Valid(content) {
		return cliArtifactRead{}, fmt.Errorf("%w: source artifact is not UTF-8 text", app.ErrInvalidInput)
	}
	if offset < 0 {
		return cliArtifactRead{}, fmt.Errorf("%w: offset must be non-negative", app.ErrInvalidInput)
	}
	if maxBytes < 0 {
		return cliArtifactRead{}, fmt.Errorf("%w: max-bytes must be non-negative", app.ErrInvalidInput)
	}
	if maxBytes == 0 {
		maxBytes = cliDefaultReadBytes
	}
	if maxBytes > cliMaxReadBytes {
		maxBytes = cliMaxReadBytes
	}
	if offset > int64(len(content)) {
		offset = int64(len(content))
	}
	end := offset + maxBytes
	if end > int64(len(content)) {
		end = int64(len(content))
	}
	truncated := end < int64(len(content))
	nextOffset := int64(0)
	if truncated {
		nextOffset = end
	}
	return cliArtifactRead{
		Content:    string(content[offset:end]),
		Offset:     offset,
		NextOffset: nextOffset,
		Truncated:  truncated,
	}, nil
}

func runSourcesGrep(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources grep", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	query := fs.String("query", "", "grep query")
	maxSnippets := fs.Int("max-snippets", 0, "maximum snippets")
	jsonOut := fs.Bool("json", false, "write JSON")
	localRoots := repeatedStringFlag{}
	fs.Var(&localRoots, "local-source-root", "allowlisted local source root root_id=path; repeatable")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 || strings.TrimSpace(*query) == "" {
		fmt.Fprintln(stderr, "usage: plasma sources grep <mission_id> <source_id> --query <text>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath, []string(localRoots)...)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	result, err := svc.GrepLocalPathSource(ctx, app.GrepLocalPathSourceRequest{
		MissionID:     positionals[0],
		SnapshotID:    positionals[1],
		Query:         *query,
		MaxSnippets:   *maxSnippets,
		Producer:      ledger.Producer{Type: "user", ID: "plasma-cli"},
		ToolSessionID: "plasma-cli",
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources grep", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{
			"snapshot":             result.Snapshot,
			"grep":                 result.Grep,
			"observation_metadata": result.Grep.Metadata,
			"observation_event":    result.ObservationEvent,
			"observation_event_id": cliLedgerEventID(result.ObservationEvent),
		})
		return 0
	}
	for _, match := range result.Grep.Matches {
		fmt.Fprintf(stdout, "%s:%d:%d\t%s\n", match.RelativePath, match.Line, match.Column, match.Snippet)
	}
	fmt.Fprintf(stdout, "observation_event=%s truncated=%v\n", cliLedgerEventID(result.ObservationEvent), result.Grep.Truncated)
	return 0
}

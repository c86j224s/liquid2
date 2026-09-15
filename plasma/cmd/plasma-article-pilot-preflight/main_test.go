package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/articleexperiment"
)

func TestRunSyntheticMatrixAndVerify(t *testing.T) {
	archive := t.TempDir()
	if err := os.Chmod(archive, 0o700); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	protocolPath, protocolSHA := writeSyntheticProtocol(t, archive)
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{
		"--synthetic", "--archive-root", archive, "--repository-root", repo,
		"--protocol", protocolPath, "--protocol-sha256", protocolSHA,
		"--fail-run", "run-M2-A",
	}, &stdout, &stderr)
	fields := outputFields(t, stdout.String(), []string{"matrix", "matrix_sha256", "completed", "failed"})
	if code != 0 || stderr.Len() != 0 || fields["matrix"] == "" || fields["completed"] != "8" || fields["failed"] != "1" {
		t.Fatalf("code=%d fields=%#v stderr=%q", code, fields, stderr.String())
	}
	stdout.Reset()
	code = run(context.Background(), []string{
		"--verify", "--archive-root", archive, "--repository-root", repo,
		"--protocol", protocolPath, "--protocol-sha256", protocolSHA,
		"--matrix-sha256", fields["matrix_sha256"],
	}, &stdout, &stderr)
	verified := outputFields(t, stdout.String(), []string{"verified", "completed", "failed"})
	if code != 0 || verified["verified"] != "true" || verified["completed"] != "8" || verified["failed"] != "1" {
		t.Fatalf("verify code=%d fields=%#v stderr=%q", code, verified, stderr.String())
	}
}

func TestRunRequiresSyntheticAndExactProtocol(t *testing.T) {
	t.Run("missing mode", func(t *testing.T) {
		archive, repo := privateTempDir(t), t.TempDir()
		protocolPath, protocolSHA := writeSyntheticProtocol(t, archive)
		var stdout, stderr bytes.Buffer
		args := []string{"--archive-root", archive, "--repository-root", repo, "--protocol", protocolPath, "--protocol-sha256", protocolSHA}
		if got := run(context.Background(), args, &stdout, &stderr); got != 2 {
			t.Fatalf("run()=%d want=2", got)
		}
	})
	t.Run("rejects trailing arguments", func(t *testing.T) {
		archive, repo := privateTempDir(t), t.TempDir()
		protocolPath, protocolSHA := writeSyntheticProtocol(t, archive)
		var stdout, stderr bytes.Buffer
		args := []string{"--synthetic", "--archive-root", archive, "--repository-root", repo, "--protocol", protocolPath, "--protocol-sha256", protocolSHA, "--fail-run", "run-M2-A", "extra"}
		if got := run(context.Background(), args, &stdout, &stderr); got != 2 {
			t.Fatalf("run()=%d want=2", got)
		}
	})
	t.Run("rejects mode-specific extras", func(t *testing.T) {
		archive, repo := privateTempDir(t), t.TempDir()
		protocolPath, protocolSHA := writeSyntheticProtocol(t, archive)
		common := []string{"--archive-root", archive, "--repository-root", repo, "--protocol", protocolPath, "--protocol-sha256", protocolSHA}
		for _, args := range [][]string{
			append(append([]string{}, common...), "--synthetic", "--fail-run", "run-M2-A", "--matrix-sha256", strings.Repeat("0", 64)),
			append(append([]string{}, common...), "--verify", "--matrix-sha256", strings.Repeat("0", 64), "--fail-run", "run-M2-A"),
			append(append([]string{}, common...), "--synthetic", "--verify", "--fail-run", "run-M2-A", "--matrix-sha256", strings.Repeat("0", 64)),
		} {
			var stdout, stderr bytes.Buffer
			if got := run(context.Background(), args, &stdout, &stderr); got != 2 {
				t.Fatalf("run(%v)=%d want=2", args, got)
			}
		}
	})
	t.Run("repository root must be a directory", func(t *testing.T) {
		archive := privateTempDir(t)
		protocolPath, protocolSHA := writeSyntheticProtocol(t, archive)
		repoFile := filepath.Join(t.TempDir(), "repo-file")
		if err := os.WriteFile(repoFile, []byte("not a directory"), 0o600); err != nil {
			t.Fatal(err)
		}
		var stdout, stderr bytes.Buffer
		args := []string{"--synthetic", "--archive-root", archive, "--repository-root", repoFile, "--protocol", protocolPath, "--protocol-sha256", protocolSHA, "--fail-run", "run-M2-A"}
		if got := run(context.Background(), args, &stdout, &stderr); got != 1 {
			t.Fatalf("run()=%d want=1 stderr=%q", got, stderr.String())
		}
	})
	t.Run("wrong protocol hash", func(t *testing.T) {
		archive, repo := privateTempDir(t), t.TempDir()
		protocolPath, _ := writeSyntheticProtocol(t, archive)
		var stdout, stderr bytes.Buffer
		args := []string{"--synthetic", "--archive-root", archive, "--repository-root", repo, "--protocol", protocolPath, "--protocol-sha256", strings.Repeat("0", 64), "--fail-run", "run-M2-A"}
		if got := run(context.Background(), args, &stdout, &stderr); got != 1 {
			t.Fatalf("run()=%d want=1 stderr=%q", got, stderr.String())
		}
	})
	t.Run("unassigned fail run", func(t *testing.T) {
		archive, repo := privateTempDir(t), t.TempDir()
		protocolPath, protocolSHA := writeSyntheticProtocol(t, archive)
		var stdout, stderr bytes.Buffer
		args := []string{"--synthetic", "--archive-root", archive, "--repository-root", repo, "--protocol", protocolPath, "--protocol-sha256", protocolSHA, "--fail-run", "run-other"}
		if got := run(context.Background(), args, &stdout, &stderr); got != 1 {
			t.Fatalf("run()=%d want=1 stderr=%q", got, stderr.String())
		}
		if _, err := os.Stat(filepath.Join(archive, "runs")); !os.IsNotExist(err) {
			t.Fatalf("unassigned failure created runs: %v", err)
		}
	})
}

func privateTempDir(t *testing.T) string {
	t.Helper()
	path := t.TempDir()
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func outputFields(t *testing.T, output string, expected []string) map[string]string {
	t.Helper()
	allowed := map[string]bool{}
	for _, key := range expected {
		allowed[key] = true
	}
	fields := map[string]string{}
	for _, line := range strings.Split(strings.TrimSuffix(output, "\n"), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || !allowed[key] || value == "" || fields[key] != "" {
			t.Fatalf("invalid output line %q in %q", line, output)
		}
		fields[key] = value
	}
	if len(fields) != len(expected) {
		t.Fatalf("output fields = %#v, want %v", fields, expected)
	}
	return fields
}

func writeSyntheticProtocol(t *testing.T, archive string) (string, string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(archive, "fixtures"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(archive, "arms"), 0o700); err != nil {
		t.Fatal(err)
	}
	fixtureRefs := []articleexperiment.FileRef{}
	for _, fixtureID := range []string{"M1", "M2", "M3"} {
		input := []byte("source/" + fixtureID + "\n")
		inputPath := filepath.Join(archive, "fixtures", fixtureID+".txt")
		if err := os.WriteFile(inputPath, input, 0o600); err != nil {
			t.Fatal(err)
		}
		fixture := articleexperiment.Fixture{
			SchemaVersion: articleexperiment.FixtureSchemaVersion, FixtureID: fixtureID,
			ValueType: map[string]string{"M1": "method_how_to", "M2": "mechanism_insight", "M3": "discovery_context"}[fixtureID],
			Audience:  "독자", ReaderPromise: "새로운 이해", Language: "ko",
			InputFiles: []articleexperiment.FileRef{{ID: "source", Path: fixtureID + ".txt", SHA256: digest(input)}},
		}
		path := filepath.Join(archive, "fixtures", fixtureID+".json")
		raw := writeJSON(t, path, fixture)
		fixtureRefs = append(fixtureRefs, articleexperiment.FileRef{ID: fixtureID, Path: filepath.ToSlash(filepath.Join("fixtures", fixtureID+".json")), SHA256: digest(raw)})
	}
	armRefs := []articleexperiment.FileRef{}
	for _, armID := range []string{"R", "E", "A"} {
		contract := []byte("contract/" + armID + "\n")
		if err := os.WriteFile(filepath.Join(archive, "arms", armID+".txt"), contract, 0o600); err != nil {
			t.Fatal(err)
		}
		arm := articleexperiment.Arm{
			SchemaVersion: articleexperiment.ArmSchemaVersion, ArmID: armID,
			Role:         map[string]string{"R": "contextual_report_il", "E": "matched_control", "A": "article_narrative"}[armID],
			ContractPath: armID + ".txt", ContractSHA256: digest(contract), ContextualOnly: armID == "R",
		}
		path := filepath.Join(archive, "arms", armID+".json")
		raw := writeJSON(t, path, arm)
		armRefs = append(armRefs, articleexperiment.FileRef{ID: armID, Path: filepath.ToSlash(filepath.Join("arms", armID+".json")), SHA256: digest(raw)})
	}
	cells := []articleexperiment.RunCell{}
	for _, fixtureID := range []string{"M1", "M2", "M3"} {
		for _, armID := range []string{"R", "E", "A"} {
			cells = append(cells, articleexperiment.RunCell{FixtureID: fixtureID, ArmID: armID, RunID: "run-" + fixtureID + "-" + armID})
		}
	}
	protocol := articleexperiment.Protocol{SchemaVersion: articleexperiment.ProtocolSchemaVersion, ProtocolID: "synthetic-v1", CreatedAt: "2026-09-02T00:00:00Z", Fixtures: fixtureRefs, Arms: armRefs, Runs: cells}
	protocolPath := filepath.Join(archive, "protocol.json")
	raw := writeJSON(t, protocolPath, protocol)
	return protocolPath, digest(raw)
}

func writeJSON(t *testing.T, path string, value any) []byte {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return raw
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

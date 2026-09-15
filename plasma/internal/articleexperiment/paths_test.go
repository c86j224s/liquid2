package articleexperiment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestLoadJSONRejectsDuplicateKeysAndInvalidUTF8(t *testing.T) {
	for _, test := range []struct {
		name string
		raw  []byte
	}{
		{name: "duplicate key", raw: []byte(`{"schema_version":"one","schema_version":"two"}`)},
		{name: "invalid UTF-8", raw: []byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.json")
			if err := os.WriteFile(path, test.raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := loadJSONFile[Protocol](path); !errors.Is(err, producterror.ErrInvalidInput) {
				t.Fatalf("error = %v, want invalid input", err)
			}
		})
	}
}

func TestRunRejectsChangedAndSymlinkedInputs(t *testing.T) {
	t.Run("reference traversal", func(t *testing.T) {
		archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
		protocol, _, err := loadJSONFile[Protocol](protocolPath)
		if err != nil {
			t.Fatal(err)
		}
		protocol.Fixtures[0].Path = "fixtures/../fixtures/M1.json"
		writeTestJSON(t, protocolPath, protocol)
		_, err = Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-R", Executor: &fakeExecutor{}})
		if !errors.Is(err, producterror.ErrInvalidInput) {
			t.Fatalf("error = %v, want invalid input", err)
		}
	})
	t.Run("input count ceiling", func(t *testing.T) {
		archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
		fixturePath := filepath.Join(archive, "fixtures", "M1.json")
		fixture, _, err := loadJSONFile[Fixture](fixturePath)
		if err != nil {
			t.Fatal(err)
		}
		fixture.InputFiles = make([]FileRef, maxFixtureInputCount+1)
		for index := range fixture.InputFiles {
			fixture.InputFiles[index] = FileRef{ID: fmt.Sprintf("source-%d", index), Path: "M1-input.txt", SHA256: testFileSHA256(t, filepath.Join(archive, "fixtures", "M1-input.txt"))}
		}
		writeTestJSON(t, fixturePath, fixture)
		protocol, _, err := loadJSONFile[Protocol](protocolPath)
		if err != nil {
			t.Fatal(err)
		}
		protocol.Fixtures[0].SHA256 = testFileSHA256(t, fixturePath)
		writeTestJSON(t, protocolPath, protocol)
		_, err = Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-R", Executor: &fakeExecutor{}})
		if !errors.Is(err, producterror.ErrInvalidInput) {
			t.Fatalf("error = %v, want invalid input", err)
		}
	})

	t.Run("input SHA", func(t *testing.T) {
		archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
		if err := os.WriteFile(filepath.Join(archive, "fixtures", "M1-input.txt"), []byte("changed\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-R", Executor: &fakeExecutor{}})
		if !errors.Is(err, producterror.ErrConflict) {
			t.Fatalf("error = %v, want conflict", err)
		}
	})
	t.Run("symlink fixture", func(t *testing.T) {
		archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{symlinkFixture: true})
		_, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-R", Executor: &fakeExecutor{}})
		if !errors.Is(err, producterror.ErrInvalidInput) {
			t.Fatalf("error = %v, want invalid input", err)
		}
	})
	t.Run("duplicate input file identity", func(t *testing.T) {
		archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
		fixturePath := filepath.Join(archive, "fixtures", "M1.json")
		fixture, _, err := loadJSONFile[Fixture](fixturePath)
		if err != nil {
			t.Fatal(err)
		}
		fixture.InputFiles = append(fixture.InputFiles, FileRef{ID: "source-copy", Path: fixture.InputFiles[0].Path, SHA256: fixture.InputFiles[0].SHA256})
		writeTestJSON(t, fixturePath, fixture)
		protocol, _, err := loadJSONFile[Protocol](protocolPath)
		if err != nil {
			t.Fatal(err)
		}
		protocol.Fixtures[0].SHA256 = testFileSHA256(t, fixturePath)
		writeTestJSON(t, protocolPath, protocol)
		_, err = Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-R", Executor: &fakeExecutor{}})
		if !errors.Is(err, producterror.ErrInvalidInput) {
			t.Fatalf("error = %v, want invalid input", err)
		}
	})
	t.Run("archive resolves into repo", func(t *testing.T) {
		parent := t.TempDir()
		repo := filepath.Join(parent, "repo")
		if err := os.Mkdir(repo, 0o700); err != nil {
			t.Fatal(err)
		}
		archiveLink := filepath.Join(parent, "archive")
		if err := os.Symlink(repo, archiveLink); err != nil {
			t.Fatal(err)
		}
		_, _, err := prepareArchiveRoot(archiveLink, repo)
		if !errors.Is(err, producterror.ErrInvalidInput) {
			t.Fatalf("error = %v, want invalid input", err)
		}
	})
}

func TestValidateTerminalRejectsReservedArtifactAndEmptyIdentity(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*TerminalManifest, string)
	}{
		{name: "reserved artifact", edit: func(terminal *TerminalManifest, runDir string) {
			raw, err := os.ReadFile(filepath.Join(runDir, "run.pending.json"))
			if err != nil {
				t.Fatal(err)
			}
			terminal.Artifacts = []ArtifactReceipt{{Kind: "fake", MediaType: "application/json", Filename: "run.pending.json", SHA256: bytesSHA256(raw), ByteSize: len(raw)}}
			_ = os.Remove(filepath.Join(runDir, "article.md"))
		}},
		{name: "empty fixture identity", edit: func(terminal *TerminalManifest, _ string) {
			terminal.Pending.FixtureID = ""
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
			result, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-A", Executor: &fakeExecutor{}})
			if err != nil {
				t.Fatal(err)
			}
			terminal := result.Terminal
			test.edit(&terminal, result.RunDir)
			writeTestJSON(t, filepath.Join(result.RunDir, "run.pending.json"), terminal.Pending)
			writeTestJSON(t, result.TerminalPath, terminal)
			if _, err := ValidateTerminal(result.RunDir); !errors.Is(err, producterror.ErrInvalidInput) {
				t.Fatalf("error = %v, want invalid input", err)
			}
		})
	}
}

func TestValidateTerminalBindsExactPendingBytes(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	result, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-A", Executor: &fakeExecutor{}})
	if err != nil {
		t.Fatal(err)
	}
	pendingRaw, err := os.ReadFile(result.PendingPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result.PendingPath, append([]byte(" \n"), pendingRaw...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateTerminal(result.RunDir); !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
}

func TestValidateTerminalAcceptsLinkedAtomicTempResidue(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	result, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-A", Executor: &fakeExecutor{}})
	if err != nil {
		t.Fatal(err)
	}
	temporary := filepath.Join(result.RunDir, ".run.terminal.json.tmp")
	if err := os.Link(result.TerminalPath, temporary); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateTerminal(result.RunDir); err != nil {
		t.Fatalf("linked atomic temp residue was rejected: %v", err)
	}
	if err := os.Remove(temporary); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(temporary, []byte("different"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateTerminal(result.RunDir); !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("unlinked temp residue error = %v, want conflict", err)
	}
}

func TestValidateTerminalRejectsCaseFoldedDuplicateArtifacts(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	result, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-A", Executor: &fakeExecutor{}})
	if err != nil {
		t.Fatal(err)
	}
	terminal := result.Terminal
	terminal.Artifacts = append(terminal.Artifacts, terminal.Artifacts[0])
	terminal.Artifacts[1].Filename = strings.ToUpper(terminal.Artifacts[1].Filename)
	writeTestJSON(t, result.TerminalPath, terminal)
	if _, err := ValidateTerminal(result.RunDir); !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("error = %v, want invalid input", err)
	}
}

func TestValidateTerminalRejectsUnsafeFailureMessage(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	result, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-A", Executor: &fakeExecutor{failAt: "run-M1-A"}})
	if err == nil {
		t.Fatal("forced failure unexpectedly succeeded")
	}
	terminal := result.Terminal
	terminal.Failure.Message = "private provider detail"
	writeTestJSON(t, result.TerminalPath, terminal)
	if _, err := ValidateTerminal(result.RunDir); !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("error = %v, want invalid input", err)
	}
}

func TestValidateTerminalRejectsTamperAndIncompleteRun(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	result, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-A", Executor: &fakeExecutor{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(result.RunDir, "article.md"), []byte("tampered\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateTerminal(result.RunDir); !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("tamper error = %v, want conflict", err)
	}
	partial := filepath.Join(archive, "runs", "partial")
	if err := os.Mkdir(partial, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONExclusive(filepath.Join(partial, "run.pending.json"), PendingManifest{SchemaVersion: PendingSchemaVersion}); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateTerminal(partial); err == nil {
		t.Fatal("partial run without terminal manifest was accepted")
	}
}

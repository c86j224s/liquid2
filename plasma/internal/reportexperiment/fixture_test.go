package reportexperiment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestLoadFixtureValidatesReceiptJSONAndCanonicalPaths(t *testing.T) {
	t.Run("part sha mismatch", func(t *testing.T) {
		archive := t.TempDir()
		repo := t.TempDir()
		fixturePath := writeTestFixture(t, archive, "bad-sha")
		rewriteFixture(t, fixturePath, func(fixture *Fixture) {
			fixture.Parts[0].SHA256 = strings.Repeat("0", 64)
		})
		if _, err := loadFixture(archive, fixturePath, repo); !errors.Is(err, app.ErrInvalidInput) {
			t.Fatalf("loadFixture error = %v, want ErrInvalidInput", err)
		}
	})
	t.Run("fixture in repository", func(t *testing.T) {
		repo := t.TempDir()
		archive := filepath.Join(repo, "archive")
		if err := os.MkdirAll(archive, 0o700); err != nil {
			t.Fatal(err)
		}
		fixturePath := writeTestFixture(t, archive, "inside-repo")
		if _, err := loadFixture(archive, fixturePath, repo); !errors.Is(err, app.ErrInvalidInput) {
			t.Fatalf("loadFixture error = %v, want ErrInvalidInput", err)
		}
	})
	t.Run("fixture symlink points into repository", func(t *testing.T) {
		archive := t.TempDir()
		repo := t.TempDir()
		repoFixture := writeTestFixture(t, repo, "fixture-symlink-source")
		fixtureLinkDir := filepath.Join(archive, "fixtures", "fixture-symlink")
		if err := os.MkdirAll(fixtureLinkDir, 0o700); err != nil {
			t.Fatal(err)
		}
		fixtureLink := filepath.Join(fixtureLinkDir, "fixture.json")
		if err := os.Symlink(repoFixture, fixtureLink); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		archiveCanonical, err := prepareArchiveRoot(archive, repo)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := loadFixture(archiveCanonical, fixtureLink, repo); !errors.Is(err, app.ErrInvalidInput) {
			t.Fatalf("loadFixture error = %v, want ErrInvalidInput", err)
		}
	})
	t.Run("part symlink points into repository", func(t *testing.T) {
		archive := t.TempDir()
		repo := t.TempDir()
		fixturePath := writeTestFixture(t, archive, "part-symlink")
		repoPart := filepath.Join(repo, "repo-part.md")
		repoBytes := []byte("# repo part\n\n이 파일은 repository 내부 symlink escape 대상입니다.\n")
		if err := os.WriteFile(repoPart, repoBytes, 0o600); err != nil {
			t.Fatal(err)
		}
		linkPath := filepath.Join(filepath.Dir(fixturePath), "repo-part-link.md")
		if err := os.Symlink(repoPart, linkPath); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		rewriteFixture(t, fixturePath, func(fixture *Fixture) {
			fixture.Parts[0].Path = "repo-part-link.md"
			fixture.Parts[0].SHA256 = bytesSHA256(repoBytes)
		})
		archiveCanonical, err := prepareArchiveRoot(archive, repo)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := loadFixture(archiveCanonical, fixturePath, repo); !errors.Is(err, app.ErrInvalidInput) {
			t.Fatalf("loadFixture error = %v, want ErrInvalidInput", err)
		}
	})
	t.Run("trailing JSON is rejected", func(t *testing.T) {
		archive := t.TempDir()
		repo := t.TempDir()
		fixturePath := writeTestFixture(t, archive, "trailing-json")
		raw, err := os.ReadFile(fixturePath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fixturePath, append(raw, []byte(`{"extra":true}`)...), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadFixture(archive, fixturePath, repo); !errors.Is(err, app.ErrInvalidInput) {
			t.Fatalf("loadFixture error = %v, want ErrInvalidInput", err)
		}
	})
	t.Run("post report humanize must be canonical", func(t *testing.T) {
		archive := t.TempDir()
		repo := t.TempDir()
		fixturePath := writeTestFixture(t, archive, "humanize-space")
		rewriteFixture(t, fixturePath, func(fixture *Fixture) {
			fixture.PostReportHumanize = " enabled "
		})
		if _, err := loadFixture(archive, fixturePath, repo); !errors.Is(err, app.ErrInvalidInput) {
			t.Fatalf("loadFixture error = %v, want ErrInvalidInput", err)
		}
	})
	t.Run("empty part content is rejected", func(t *testing.T) {
		archive := t.TempDir()
		repo := t.TempDir()
		fixturePath := writeTestFixture(t, archive, "empty-part")
		partPath := filepath.Join(filepath.Dir(fixturePath), "part-01.md")
		if err := os.WriteFile(partPath, []byte(" \n\t"), 0o600); err != nil {
			t.Fatal(err)
		}
		rewriteFixture(t, fixturePath, func(fixture *Fixture) {
			fixture.Parts[0].SHA256 = bytesSHA256([]byte(" \n\t"))
		})
		if _, err := loadFixture(archive, fixturePath, repo); !errors.Is(err, app.ErrInvalidInput) {
			t.Fatalf("loadFixture error = %v, want ErrInvalidInput", err)
		}
	})
	t.Run("empty direction hint is allowed", func(t *testing.T) {
		archive := t.TempDir()
		repo := t.TempDir()
		fixturePath := writeTestFixture(t, archive, "empty-direction")
		rewriteFixture(t, fixturePath, func(fixture *Fixture) {
			fixture.DirectionHint = ""
		})
		if _, err := loadFixture(archive, fixturePath, repo); err != nil {
			t.Fatalf("loadFixture returned error for empty direction hint: %v", err)
		}
	})
}

func TestPrepareRunDirRejectsExistingEmptyDirectory(t *testing.T) {
	archive := t.TempDir()
	repo := t.TempDir()
	runDir := filepath.Join(archive, "runs", "duplicate-empty")
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		t.Fatal(err)
	}
	archiveCanonical, err := prepareArchiveRoot(archive, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prepareRunDir(archiveCanonical, "duplicate-empty", repo); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("prepareRunDir error = %v, want ErrConflict", err)
	}
}

func TestPrepareArchiveRootRejectsMissingPathThroughRepositorySymlink(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "archive-link")
	if err := os.Symlink(repo, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	archiveRoot := filepath.Join(link, "new-archive")
	if _, err := prepareArchiveRoot(archiveRoot, repo); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("prepareArchiveRoot error = %v, want ErrInvalidInput", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "new-archive")); !os.IsNotExist(err) {
		t.Fatalf("archive directory was created inside repo before rejection: %v", err)
	}
}

func TestRunPreflightRejectsNonV3ProfileBeforeRunDirectory(t *testing.T) {
	ctx := context.Background()
	archive := t.TempDir()
	repo := t.TempDir()
	fixturePath := writeTestFixture(t, archive, "non-v3-profile")
	rewriteFixture(t, fixturePath, func(fixture *Fixture) {
		fixture.GenerationGuidanceProfile = "none"
	})
	called := false
	_, err := Run(ctx, Config{
		ArchiveRoot: archive, FixturePath: fixturePath, RunID: "non-v3-profile", RepositoryRoot: repo,
		AgentModel: "gpt-test", AgentReasoningEffort: "medium",
		ServiceFactory: sqliteServiceFactory,
		BinaryPair: BinaryPair{
			Experiment: BinaryMetadata{Path: "/tmp/plasma-report-experiment", SHA256: strings.Repeat("a", 64)},
			PlasmaMCP:  BinaryMetadata{Path: "/tmp/plasma", SHA256: strings.Repeat("b", 64)},
			Codex:      BinaryMetadata{Path: "/tmp/codex", SHA256: strings.Repeat("c", 64)},
		},
		ExecutorFactory: func(context.Context, ExecutorContext) (agentexec.AgentExecutor, error) {
			called = true
			return &fakeV3Executor{}, nil
		},
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("Run error = %v, want ErrConflict", err)
	}
	if called {
		t.Fatal("executor factory was called after pure preflight failure")
	}
	if _, statErr := os.Stat(filepath.Join(archive, "runs", "non-v3-profile")); !os.IsNotExist(statErr) {
		t.Fatalf("run directory should not exist after pure preflight failure: %v", statErr)
	}
}

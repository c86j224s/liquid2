package articleexperiment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRevisionFixtureSupportsGitLayouts(t *testing.T) {
	const revision = "0123456789abcdef0123456789abcdef01234567"
	for _, layout := range []string{"checkout", "packed", "detached", "linked"} {
		t.Run(layout, func(t *testing.T) {
			root := t.TempDir()
			gitDir := filepath.Join(root, ".git")
			common := gitDir
			write := func(path, content string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if layout == "linked" {
				common = filepath.Join(root, "metadata")
				gitDir = filepath.Join(common, "worktrees", "qa")
				write(filepath.Join(root, ".git"), "gitdir: metadata/worktrees/qa\n")
				write(filepath.Join(gitDir, "commondir"), "../..\n")
			}
			if layout == "detached" {
				write(filepath.Join(gitDir, "HEAD"), revision+"\n")
			} else {
				write(filepath.Join(gitDir, "HEAD"), "ref: refs/heads/main\n")
				if layout == "packed" {
					write(filepath.Join(common, "packed-refs"), "# pack-refs\n"+revision+" refs/heads/main\n")
				} else {
					write(filepath.Join(common, "refs", "heads", "main"), revision+"\n")
				}
			}
			if got := testGitRevision(t, root); got != revision {
				t.Fatalf("got %s", got)
			}
		})
	}
}

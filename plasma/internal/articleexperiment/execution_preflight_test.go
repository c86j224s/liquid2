package articleexperiment

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestVerifyExecutionRevisionBindsRepositoryAndBinary(t *testing.T) {
	worktree := testWorktreeRoot(t)
	archive, _, protocolPath, protocolSHA := writeRealProtocolFixture(t, func(protocol *RealProtocol, _ map[string]*RealArm, _ map[string]*RealFixture, _ *BlindContract) {
		protocol.CodeRevision = testGitRevision(t, worktree)
	})
	loaded, err := LoadRealProtocol(archive, worktree, protocolPath, protocolSHA)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyExecutionRevision(worktree, loaded, loaded.Protocol.CodeRevision, false); err != nil {
		t.Fatal(err)
	}
	if err := VerifyExecutionRevision(worktree, loaded, loaded.Protocol.CodeRevision, true); !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("modified binary error = %v, want conflict", err)
	}
}

func testGitRevision(t *testing.T, repositoryRoot string) string {
	t.Helper()
	gitFile := filepath.Join(repositoryRoot, ".git")
	info, err := os.Stat(gitFile)
	if err != nil {
		t.Fatal(err)
	}
	gitDir := gitFile
	if !info.IsDir() {
		raw, err := os.ReadFile(gitFile)
		if err != nil {
			t.Fatal(err)
		}
		gitDir = strings.TrimSpace(strings.TrimPrefix(string(raw), "gitdir: "))
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(repositoryRoot, gitDir)
		}
		gitDir = filepath.Clean(gitDir)
	}
	headRaw, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	head := strings.TrimSpace(string(headRaw))
	if strings.HasPrefix(head, "ref: ") {
		commonDir := gitDir
		commonRaw, err := os.ReadFile(filepath.Join(gitDir, "commondir"))
		if err == nil {
			commonDir = strings.TrimSpace(string(commonRaw))
			if !filepath.IsAbs(commonDir) {
				commonDir = filepath.Join(gitDir, commonDir)
			}
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		refName := strings.TrimPrefix(head, "ref: ")
		refRaw, err := os.ReadFile(filepath.Join(commonDir, filepath.FromSlash(refName)))
		if os.IsNotExist(err) {
			packed, readErr := os.ReadFile(filepath.Join(commonDir, "packed-refs"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			head = ""
			for _, line := range strings.Split(string(packed), "\n") {
				fields := strings.Fields(line)
				if len(fields) == 2 && fields[1] == refName {
					head = fields[0]
					break
				}
			}
			if head == "" {
				t.Fatalf("missing packed ref %q", refName)
			}
		} else if err != nil {
			t.Fatal(err)
		} else {
			head = strings.TrimSpace(string(refRaw))
		}
	}
	return head
}

func testWorktreeRoot(t *testing.T) string {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(workingDirectory, "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Fatalf("test worktree root: %v", err)
	}
	return root
}

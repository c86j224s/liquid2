package articleexperiment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// VerifyExecutionRevision binds the frozen code revision to both the checked-out
// repository and the executable build metadata supplied by the command adapter.
func VerifyExecutionRevision(repositoryRoot string, loaded LoadedRealProtocol, executableRevision string, executableModified bool) error {
	repositoryRoot, err := canonicalExistingDirectory(repositoryRoot)
	if err != nil {
		return err
	}
	gitMarker := filepath.Join(repositoryRoot, ".git")
	if _, err := os.Stat(gitMarker); err != nil {
		return fmt.Errorf("%w: repository Git metadata is unavailable", producterror.ErrInvalidInput)
	}
	headPath := filepath.Join(gitMarker, "HEAD")
	if info, err := os.Stat(gitMarker); err == nil && !info.IsDir() {
		raw, readErr := os.ReadFile(gitMarker)
		if readErr != nil {
			return readErr
		}
		prefix := "gitdir: "
		value := strings.TrimSpace(string(raw))
		if !strings.HasPrefix(value, prefix) {
			return fmt.Errorf("%w: repository worktree metadata is invalid", producterror.ErrInvalidInput)
		}
		gitDir := strings.TrimSpace(strings.TrimPrefix(value, prefix))
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(repositoryRoot, gitDir)
		}
		headPath = filepath.Join(gitDir, "HEAD")
	}
	headRaw, err := os.ReadFile(headPath)
	if err != nil {
		return err
	}
	head := strings.TrimSpace(string(headRaw))
	if strings.HasPrefix(head, "ref: ") {
		refName := filepath.FromSlash(strings.TrimPrefix(head, "ref: "))
		gitDir := filepath.Dir(headPath)
		commonDir := gitDir
		if commonRaw, readErr := os.ReadFile(filepath.Join(gitDir, "commondir")); readErr == nil {
			commonDir = strings.TrimSpace(string(commonRaw))
			if !filepath.IsAbs(commonDir) {
				commonDir = filepath.Join(gitDir, commonDir)
			}
		} else if !os.IsNotExist(readErr) {
			return readErr
		}
		refPath := filepath.Join(commonDir, refName)
		refRaw, readErr := os.ReadFile(refPath)
		if os.IsNotExist(readErr) {
			head, readErr = readPackedRef(filepath.Join(commonDir, "packed-refs"), filepath.ToSlash(refName))
		} else if readErr == nil {
			head = strings.TrimSpace(string(refRaw))
		}
		if readErr != nil {
			return readErr
		}
	}
	if head != loaded.Protocol.CodeRevision || executableRevision != loaded.Protocol.CodeRevision || executableModified {
		return fmt.Errorf("%w: execution revision differs from frozen protocol", producterror.ErrConflict)
	}
	return nil
}

func readPackedRef(path, refName string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == refName && validGitRevision(fields[0]) {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("%w: repository HEAD ref is unavailable", producterror.ErrConflict)
}

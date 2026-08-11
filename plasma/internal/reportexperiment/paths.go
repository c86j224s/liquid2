package reportexperiment

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

var runIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$`)

func validateRunID(runID string) error {
	if !runIDPattern.MatchString(strings.TrimSpace(runID)) || strings.Contains(runID, "..") {
		return fmt.Errorf("%w: run ID must be a safe slug", producterror.ErrInvalidInput)
	}
	return nil
}

func safeIDStem(value string) string {
	var builder strings.Builder
	for _, r := range strings.TrimSpace(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}
	stem := strings.Trim(builder.String(), "_")
	if stem == "" {
		return "run"
	}
	return stem
}

func absolutePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%w: path is required", producterror.ErrInvalidInput)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}

func canonicalExistingPath(path string) (string, error) {
	absolute, err := absolutePath(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

func canonicalOptionalExistingPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	return canonicalExistingPath(path)
}

// prepareArchiveRoot는 만들기 전후 모두 symlink를 해석해 archive 저장 경계를 확정한다.
func prepareArchiveRoot(archiveRoot, repoRoot string) (string, error) {
	absolute, err := absolutePath(archiveRoot)
	if err != nil {
		return "", err
	}
	createPath, err := canonicalPathForCreate(absolute)
	if err != nil {
		return "", err
	}
	if err := rejectInsideRepo(repoRoot, createPath); err != nil {
		return "", err
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return "", err
	}
	canonical, err := canonicalExistingPath(absolute)
	if err != nil {
		return "", err
	}
	if err := rejectInsideRepo(repoRoot, canonical); err != nil {
		return "", err
	}
	return canonical, nil
}

// canonicalPathForCreate는 아직 없는 경로를 만들기 전 기존 상위 symlink를 반영한 후보 위치를 계산한다.
func canonicalPathForCreate(path string) (string, error) {
	absolute, err := absolutePath(path)
	if err != nil {
		return "", err
	}
	ancestor := absolute
	missing := []string{}
	for {
		info, err := os.Stat(ancestor)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("%w: archive ancestor is not a directory: %s", producterror.ErrInvalidInput, ancestor)
			}
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return "", err
		}
		missing = append([]string{filepath.Base(ancestor)}, missing...)
		ancestor = parent
	}
	canonicalAncestor, err := canonicalExistingPath(ancestor)
	if err != nil {
		return "", err
	}
	parts := append([]string{canonicalAncestor}, missing...)
	return filepath.Join(parts...), nil
}

// pathInside는 이미 canonicalize된 경로끼리 archive/repository 포함 관계를 검사한다.
func pathInside(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

// rejectInsideRepo는 symlink 우회를 포함해 fixture와 run output이 repo 안에 놓이는 것을 막는다.
func rejectInsideRepo(repoRoot, path string) error {
	repo, err := canonicalOptionalExistingPath(repoRoot)
	if err != nil {
		return err
	}
	if repo == "" {
		return nil
	}
	candidate, err := absolutePath(path)
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(candidate); err == nil {
		candidate = filepath.Clean(resolved)
	}
	if pathInside(repo, candidate) {
		return fmt.Errorf("%w: experiment fixtures and runs must stay outside the repository", producterror.ErrInvalidInput)
	}
	return nil
}

// prepareRunDir는 매 실행마다 새 run directory만 만들고 기존 빈 directory도 재사용하지 않는다.
func prepareRunDir(archiveRoot, runID, repoRoot string) (string, error) {
	if err := validateRunID(runID); err != nil {
		return "", err
	}
	runsRoot := filepath.Join(archiveRoot, "runs")
	if err := os.MkdirAll(runsRoot, 0o700); err != nil {
		return "", err
	}
	canonicalRunsRoot, err := canonicalExistingPath(runsRoot)
	if err != nil {
		return "", err
	}
	if !pathInside(archiveRoot, canonicalRunsRoot) {
		return "", fmt.Errorf("%w: run directory root must stay under archive root", producterror.ErrInvalidInput)
	}
	if err := rejectInsideRepo(repoRoot, canonicalRunsRoot); err != nil {
		return "", err
	}
	runDir := filepath.Join(canonicalRunsRoot, runID)
	if err := os.Mkdir(runDir, 0o700); err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("%w: run directory already exists: %s", producterror.ErrConflict, runDir)
		}
		return "", err
	}
	canonicalRunDir, err := canonicalExistingPath(runDir)
	if err != nil {
		return "", err
	}
	if !pathInside(archiveRoot, canonicalRunDir) {
		return "", fmt.Errorf("%w: run directory must stay under archive root", producterror.ErrInvalidInput)
	}
	if err := rejectInsideRepo(repoRoot, canonicalRunDir); err != nil {
		return "", err
	}
	entries, err := os.ReadDir(canonicalRunDir)
	if err != nil {
		return "", err
	}
	if len(entries) != 0 {
		return "", fmt.Errorf("%w: run directory is not empty: %s", producterror.ErrConflict, canonicalRunDir)
	}
	return canonicalRunDir, nil
}

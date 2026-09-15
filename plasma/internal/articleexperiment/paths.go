package articleexperiment

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

var safeSlugPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$`)

func prepareArchiveRoot(archiveRoot, repositoryRoot string) (string, string, error) {
	repo, err := canonicalExistingDirectory(repositoryRoot)
	if err != nil {
		return "", "", err
	}
	archiveCandidate, err := canonicalForCreate(archiveRoot)
	if err != nil {
		return "", "", err
	}
	if pathInside(repo, archiveCandidate) {
		return "", "", fmt.Errorf("%w: archive root must stay outside the repository", producterror.ErrInvalidInput)
	}
	if err := os.MkdirAll(archiveCandidate, 0o700); err != nil {
		return "", "", err
	}
	archive, err := canonicalExistingDirectory(archiveCandidate)
	if err != nil {
		return "", "", err
	}
	archiveInfo, err := os.Stat(archive)
	if err != nil {
		return "", "", err
	}
	if archiveInfo.Mode().Perm()&0o077 != 0 {
		return "", "", fmt.Errorf("%w: archive root must not grant group or other permissions", producterror.ErrInvalidInput)
	}
	if pathInside(repo, archive) {
		return "", "", fmt.Errorf("%w: archive root resolved inside the repository", producterror.ErrInvalidInput)
	}
	return archive, repo, nil
}

func prepareRunDir(archiveRoot, repositoryRoot, runID string) (string, error) {
	if !safeSlugPattern.MatchString(runID) || strings.Contains(runID, "..") {
		return "", fmt.Errorf("%w: run ID must be a safe slug", producterror.ErrInvalidInput)
	}
	runsRoot := filepath.Join(archiveRoot, "runs")
	if err := os.MkdirAll(runsRoot, 0o700); err != nil {
		return "", err
	}
	resolvedRuns, err := canonicalExisting(runsRoot)
	if err != nil {
		return "", err
	}
	if !pathInside(archiveRoot, resolvedRuns) || pathInside(repositoryRoot, resolvedRuns) {
		return "", fmt.Errorf("%w: runs directory escaped archive boundary", producterror.ErrInvalidInput)
	}
	runDir := filepath.Join(resolvedRuns, runID)
	if err := os.Mkdir(runDir, 0o700); err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("%w: run directory already exists", producterror.ErrConflict)
		}
		return "", err
	}
	resolvedRun, err := canonicalExisting(runDir)
	if err != nil {
		return "", err
	}
	if !pathInside(archiveRoot, resolvedRun) || pathInside(repositoryRoot, resolvedRun) {
		return "", fmt.Errorf("%w: run directory escaped archive boundary", producterror.ErrInvalidInput)
	}
	return resolvedRun, nil
}

func resolveInput(archiveRoot, repositoryRoot, baseDir, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%w: input path is required", producterror.ErrInvalidInput)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	return resolveInputPath(archiveRoot, repositoryRoot, path)
}

func resolveReference(archiveRoot, repositoryRoot, baseDir, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || filepath.IsAbs(path) {
		return "", fmt.Errorf("%w: reference path must be relative", producterror.ErrInvalidInput)
	}
	for _, segment := range strings.FieldsFunc(filepath.ToSlash(path), func(r rune) bool { return r == '/' }) {
		if segment == "." || segment == ".." {
			return "", fmt.Errorf("%w: reference path contains traversal", producterror.ErrInvalidInput)
		}
	}
	return resolveInputPath(archiveRoot, repositoryRoot, filepath.Join(baseDir, path))
}

func resolveInputPath(archiveRoot, repositoryRoot, path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: input must be a non-symlink regular file", producterror.ErrInvalidInput)
	}
	resolved, err := canonicalExisting(path)
	if err != nil {
		return "", err
	}
	if !pathInside(archiveRoot, resolved) || pathInside(repositoryRoot, resolved) {
		return "", fmt.Errorf("%w: input escaped archive boundary", producterror.ErrInvalidInput)
	}
	return resolved, nil
}

func canonicalExistingDirectory(path string) (string, error) {
	resolved, err := canonicalExisting(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: path must be a directory", producterror.ErrInvalidInput)
	}
	return resolved, nil
}

func canonicalExisting(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%w: path is required", producterror.ErrInvalidInput)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}

func canonicalForCreate(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%w: path is required", producterror.ErrInvalidInput)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	ancestor := absolute
	missing := []string{}
	for {
		info, statErr := os.Stat(ancestor)
		if statErr == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("%w: path ancestor is not a directory", producterror.ErrInvalidInput)
			}
			break
		}
		if !os.IsNotExist(statErr) {
			return "", statErr
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return "", statErr
		}
		missing = append([]string{filepath.Base(ancestor)}, missing...)
		ancestor = parent
	}
	resolved, err := canonicalExisting(ancestor)
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{resolved}, missing...)...), nil
}

func pathInside(root, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))))
}

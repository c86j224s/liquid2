package reportexperiment

import (
	buildinfo "debug/buildinfo"
	"fmt"
	"os"
	"os/exec"
	runtimedebug "runtime/debug"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// BinaryMetadata는 실행에 관여한 Go binary의 공개 가능한 build receipt다.
type BinaryMetadata struct {
	Path             string `json:"path"`
	SHA256           string `json:"sha256"`
	VCSRevision      string `json:"vcs_revision,omitempty"`
	VCSRevisionKnown bool   `json:"vcs_revision_known"`
	VCSModified      bool   `json:"vcs_modified,omitempty"`
	VCSModifiedKnown bool   `json:"vcs_modified_known"`
}

// BinaryPair는 현재 experiment binary와 연결된 Plasma MCP binary의 receipt다.
type BinaryPair struct {
	Experiment BinaryMetadata `json:"experiment"`
	PlasmaMCP  BinaryMetadata `json:"plasma_mcp"`
	Codex      BinaryMetadata `json:"codex,omitempty"`
}

// CurrentBinaryMetadata는 현재 process binary의 SHA-256과 runtime build info를 읽는다.
func CurrentBinaryMetadata(path string) (BinaryMetadata, error) {
	if strings.TrimSpace(path) == "" {
		executable, err := os.Executable()
		if err != nil {
			return BinaryMetadata{}, err
		}
		path = executable
	}
	metadata, err := binaryMetadata(path)
	if err != nil {
		return BinaryMetadata{}, err
	}
	if info, ok := runtimedebug.ReadBuildInfo(); ok && info != nil {
		applyBuildInfo(&metadata, info)
	}
	return metadata, nil
}

// ReadBinaryMetadata는 외부 Go binary 파일의 SHA-256과 embedded build info를 읽는다.
func ReadBinaryMetadata(path string) (BinaryMetadata, error) {
	metadata, err := binaryMetadata(path)
	if err != nil {
		return BinaryMetadata{}, err
	}
	info, err := buildinfo.ReadFile(metadata.Path)
	if err == nil && info != nil {
		applyBuildInfo(&metadata, info)
	}
	return metadata, nil
}

// ReadExecutableMetadata는 실행 전에 사용할 provider/MCP command를 canonical executable receipt로 고정한다.
func ReadExecutableMetadata(path string) (BinaryMetadata, error) {
	executable, err := canonicalExecutablePath(path)
	if err != nil {
		return BinaryMetadata{}, err
	}
	return ReadBinaryMetadata(executable)
}

// ValidateBinaryPair는 양쪽 VCS revision이 모두 알려진 경우 같은 commit인지 검사한다.
func ValidateBinaryPair(pair BinaryPair) error {
	left := strings.TrimSpace(pair.Experiment.VCSRevision)
	right := strings.TrimSpace(pair.PlasmaMCP.VCSRevision)
	if pair.Experiment.VCSRevisionKnown && pair.PlasmaMCP.VCSRevisionKnown && left != "" && right != "" && left != right {
		return fmt.Errorf("%w: experiment binary revision %s differs from Plasma MCP binary revision %s", producterror.ErrConflict, left, right)
	}
	return nil
}

func binaryMetadata(path string) (BinaryMetadata, error) {
	absolute, err := canonicalExistingPath(path)
	if err != nil {
		return BinaryMetadata{}, err
	}
	sha, err := fileSHA256(absolute)
	if err != nil {
		return BinaryMetadata{}, err
	}
	return BinaryMetadata{Path: absolute, SHA256: sha}, nil
}

func canonicalExecutablePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%w: executable path is required", producterror.ErrInvalidInput)
	}
	candidate := path
	if !strings.ContainsRune(candidate, os.PathSeparator) {
		resolved, err := exec.LookPath(candidate)
		if err != nil {
			return "", fmt.Errorf("%w: executable %q was not found", producterror.ErrInvalidInput, path)
		}
		candidate = resolved
	}
	canonical, err := canonicalExistingPath(candidate)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: executable must be a regular file: %s", producterror.ErrInvalidInput, canonical)
	}
	if info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("%w: executable is not marked executable: %s", producterror.ErrInvalidInput, canonical)
	}
	return canonical, nil
}

func applyBuildInfo(metadata *BinaryMetadata, info *runtimedebug.BuildInfo) {
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			metadata.VCSRevision = strings.TrimSpace(setting.Value)
			metadata.VCSRevisionKnown = metadata.VCSRevision != ""
		case "vcs.modified":
			switch strings.TrimSpace(setting.Value) {
			case "true":
				metadata.VCSModified = true
				metadata.VCSModifiedKnown = true
			case "false":
				metadata.VCSModified = false
				metadata.VCSModifiedKnown = true
			}
		}
	}
}

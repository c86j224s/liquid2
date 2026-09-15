package articleexperiment

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const (
	maxContractBytes       = 1 << 20
	maxInputBytes          = 16 << 20
	maxFixtureInputBytes   = 64 << 20
	maxProtocolInputBytes  = 64 << 20
	maxFixtureInputCount   = 16
	maxOutputBytes         = 16 << 20
	maxOutputTotalBytes    = 64 << 20
	maxOutputArtifactCount = 16
)

func loadJSONFile[T any](path string) (T, []byte, error) {
	var value T
	raw, err := readRegularFile(path, maxContractBytes)
	if err != nil {
		return value, nil, err
	}
	return decodeJSONBytes[T](filepath.Base(path), raw)
}

func decodeJSONBytes[T any](name string, raw []byte) (T, []byte, error) {
	var value T
	if !utf8.Valid(raw) {
		return value, nil, fmt.Errorf("%w: %s is not valid UTF-8", producterror.ErrInvalidInput, name)
	}
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return value, nil, fmt.Errorf("%w: %s: %v", producterror.ErrInvalidInput, name, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, nil, fmt.Errorf("%w: decode %s: %v", producterror.ErrInvalidInput, name, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("trailing JSON value")
		}
		return value, nil, fmt.Errorf("%w: %s: %v", producterror.ErrInvalidInput, name, err)
	}
	return value, raw, nil
}

func rejectDuplicateJSONKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok || seen[key] {
					return fmt.Errorf("duplicate or invalid object key")
				}
				seen[key] = true
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return fmt.Errorf("unexpected JSON delimiter")
		}
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return err
	}
	return nil
}

func readRegularFile(path string, maxBytes int64) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: input must be a regular file", producterror.ErrInvalidInput)
	}
	if before.Size() < 0 || before.Size() > maxBytes {
		return nil, fmt.Errorf("%w: input exceeds byte ceiling", producterror.ErrInvalidInput)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("%w: input changed while opening", producterror.ErrConflict)
	}
	limited := io.LimitReader(file, maxBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("%w: input exceeds byte ceiling", producterror.ErrInvalidInput)
	}
	return raw, nil
}

func writeExclusive(path string, content []byte) (err error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(path)
		}
	}()
	if _, err = file.Write(content); err != nil {
		return err
	}
	return file.Sync()
}

func writeJSONExclusive(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := writeExclusive(path, append(raw, '\n')); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return err
	}
	return directory.Close()
}

func writeTerminalAtomic(runDir string, terminal TerminalManifest) (string, error) {
	return writeJSONAtomic(runDir, ".run.terminal.json.tmp", "run.terminal.json", terminal)
}

func writeJSONAtomic(dir, temporaryName, finalName string, value any) (string, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	raw = append(raw, '\n')
	temporary := filepath.Join(dir, temporaryName)
	final := filepath.Join(dir, finalName)
	if err := writeExclusive(temporary, raw); err != nil {
		return "", err
	}
	if err := os.Link(temporary, final); err != nil {
		_ = os.Remove(temporary)
		return "", err
	}
	if err := os.Remove(temporary); err != nil {
		_ = os.Remove(final)
		return "", err
	}
	directory, err := os.Open(dir)
	if err != nil {
		_ = os.Remove(final)
		return "", err
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		_ = os.Remove(final)
		if cleanupDir, openErr := os.Open(dir); openErr == nil {
			_ = cleanupDir.Sync()
			_ = cleanupDir.Close()
		}
		return "", err
	}
	if err := directory.Close(); err != nil {
		_ = os.Remove(final)
		return "", err
	}
	return final, nil
}

func bytesSHA256(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}

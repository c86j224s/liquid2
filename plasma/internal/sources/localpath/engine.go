package localpath

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/sources/pdftext"
	"golang.org/x/sys/unix"
)

type RootConfig struct {
	RootID string
	Path   string
	Alias  string
}

type Config struct {
	Roots        []RootConfig
	Limits       Limits
	DenyPatterns []string
}

type Limits struct {
	MaxFileReadBytes    int64
	MaxGrepBytesPerFile int64
	MaxGrepTotalBytes   int64
	MaxDirectoryEntries int
	MaxRecursiveFiles   int
	MaxRecursiveBytes   int64
	MaxPDFReadBytes     int64
	MaxTreeDepth        int
	MaxReturnedSnippets int
	MaxSingleLineBytes  int
}

type Engine struct {
	roots        map[string]root
	limits       Limits
	denyPatterns []string
}

type root struct {
	id            string
	alias         string
	canonicalPath string
}

type RootView struct {
	RootID string `json:"root_id"`
	Alias  string `json:"alias,omitempty"`
}

type PathMetadata struct {
	ObservedAt      string       `json:"observed_at"`
	RootID          string       `json:"root_id"`
	RootAlias       string       `json:"root_alias,omitempty"`
	RelativePath    string       `json:"relative_path"`
	Subpath         string       `json:"subpath,omitempty"`
	PathKind        string       `json:"path_kind"`
	Size            int64        `json:"size,omitempty"`
	MTime           string       `json:"mtime,omitempty"`
	SHA256          string       `json:"sha256,omitempty"`
	Offset          int64        `json:"offset,omitempty"`
	MaxBytes        int64        `json:"max_bytes,omitempty"`
	NextOffset      int64        `json:"next_offset,omitempty"`
	Truncated       bool         `json:"truncated,omitempty"`
	Binary          bool         `json:"binary,omitempty"`
	Extraction      string       `json:"extraction,omitempty"`
	PageCount       int          `json:"page_count,omitempty"`
	TextLength      int64        `json:"text_length,omitempty"`
	TextLengthKnown bool         `json:"text_length_known"`
	Denied          []string     `json:"denied,omitempty"`
	Cap             string       `json:"cap,omitempty"`
	Git             *GitMetadata `json:"git,omitempty"`
}

type GitMetadata struct {
	Branch               string `json:"branch,omitempty"`
	Head                 string `json:"head,omitempty"`
	Dirty                bool   `json:"dirty"`
	WorktreeRelativePath string `json:"worktree_relative_path,omitempty"`
}

type ReadRequest struct {
	RootID       string
	RelativePath string
	Subpath      string
	Offset       int64
	MaxBytes     int64
}

type ReadResult struct {
	Content  string       `json:"content"`
	Metadata PathMetadata `json:"metadata"`
}

type TreeRequest struct {
	RootID       string
	RelativePath string
	Subpath      string
	Depth        int
	Limit        int
}

type TreeEntry struct {
	Name         string `json:"name"`
	RelativePath string `json:"relative_path"`
	PathKind     string `json:"path_kind"`
	Size         int64  `json:"size,omitempty"`
	MTime        string `json:"mtime,omitempty"`
	Denied       bool   `json:"denied,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

type TreeResult struct {
	RootID       string       `json:"root_id"`
	RootAlias    string       `json:"root_alias,omitempty"`
	RelativePath string       `json:"relative_path"`
	Entries      []TreeEntry  `json:"entries"`
	Truncated    bool         `json:"truncated"`
	Metadata     PathMetadata `json:"metadata"`
}

type GrepRequest struct {
	RootID       string
	RelativePath string
	Subpath      string
	Query        string
	MaxSnippets  int
}

type GrepMatch struct {
	RelativePath string `json:"relative_path"`
	Line         int    `json:"line"`
	Column       int    `json:"column"`
	Snippet      string `json:"snippet"`
	SHA256       string `json:"sha256,omitempty"`
}

type GrepResult struct {
	RootID       string       `json:"root_id"`
	RootAlias    string       `json:"root_alias,omitempty"`
	RelativePath string       `json:"relative_path"`
	Query        string       `json:"query"`
	Matches      []GrepMatch  `json:"matches"`
	Truncated    bool         `json:"truncated"`
	Metadata     PathMetadata `json:"metadata"`
}

var ErrInvalidInput = errors.New("invalid local path input")

func New(config Config) (*Engine, error) {
	limits := defaultLimits(config.Limits)
	denyPatterns := config.DenyPatterns
	if len(denyPatterns) == 0 {
		denyPatterns = DefaultDenyPatterns()
	}
	engine := &Engine{roots: map[string]root{}, limits: limits, denyPatterns: denyPatterns}
	for _, configured := range config.Roots {
		rootID := strings.TrimSpace(configured.RootID)
		if !validRootID(rootID) {
			return nil, fmt.Errorf("%w: invalid root id", ErrInvalidInput)
		}
		if _, exists := engine.roots[rootID]; exists {
			return nil, fmt.Errorf("%w: duplicate root id", ErrInvalidInput)
		}
		configuredPath := strings.TrimSpace(configured.Path)
		if configuredPath == "" {
			return nil, fmt.Errorf("%w: root path is required", ErrInvalidInput)
		}
		abs, err := filepath.Abs(configuredPath)
		if err != nil {
			return nil, fmt.Errorf("%w: canonicalize root", ErrInvalidInput)
		}
		if evaluated, err := filepath.EvalSymlinks(abs); err == nil {
			abs = evaluated
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("%w: root does not exist", ErrInvalidInput)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%w: root must be a directory", ErrInvalidInput)
		}
		alias := strings.TrimSpace(configured.Alias)
		if alias == "" {
			alias = rootID
		}
		engine.roots[rootID] = root{id: rootID, alias: alias, canonicalPath: filepath.Clean(abs)}
	}
	return engine, nil
}

func DefaultDenyPatterns() []string {
	return []string{
		".git", ".hg", ".svn", "node_modules", "vendor", "dist", "build", ".next",
		"target", "__pycache__", ".venv", "venv", ".env", ".env.*", "*.pem",
		"*.key", "id_rsa", "id_ed25519", "*.p12", "*.pfx", "*.lock", "*.cache", "*.log",
	}
}

func (engine *Engine) Roots() []RootView {
	roots := make([]RootView, 0, len(engine.roots))
	for _, root := range engine.roots {
		roots = append(roots, RootView{RootID: root.id, Alias: root.alias})
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].RootID < roots[j].RootID })
	return roots
}

func (engine *Engine) Inspect(ctx context.Context, rootID string, relativePath string) (PathMetadata, error) {
	resolved, err := engine.resolve(rootID, relativePath, true)
	if err != nil {
		return PathMetadata{}, err
	}
	defer resolved.close()
	return engine.metadata(ctx, resolved, ""), nil
}

func (engine *Engine) IsPDF(ctx context.Context, rootID string, relativePath string) (bool, error) {
	resolved, err := engine.resolve(rootID, relativePath, false)
	if err != nil {
		return false, err
	}
	defer resolved.close()
	if resolved.info.IsDir() || !resolved.info.Mode().IsRegular() {
		return false, nil
	}
	var header [512]byte
	n, err := resolved.file.ReadAt(header[:], 0)
	if err != nil && n == 0 {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, sanitizeError(err)
	}
	return pdftext.IsPDFBytes(header[:n]), nil
}

func (engine *Engine) ReadFile(ctx context.Context, req ReadRequest) (ReadResult, error) {
	target, subpath, err := TargetRelativePath(req.RelativePath, req.Subpath)
	if err != nil {
		return ReadResult{}, err
	}
	resolved, err := engine.resolve(req.RootID, target, false)
	if err != nil {
		return ReadResult{}, err
	}
	if resolved.info.IsDir() {
		return ReadResult{}, fmt.Errorf("%w: read target is a directory", ErrInvalidInput)
	}
	if !resolved.info.Mode().IsRegular() {
		return ReadResult{}, fmt.Errorf("%w: read target is not a regular file", ErrInvalidInput)
	}
	defer resolved.close()
	offset := req.Offset
	if offset < 0 {
		return ReadResult{}, fmt.Errorf("%w: offset must be non-negative", ErrInvalidInput)
	}
	maxBytes := req.MaxBytes
	if maxBytes <= 0 || maxBytes > engine.limits.MaxFileReadBytes {
		maxBytes = engine.limits.MaxFileReadBytes
	}
	if offset > 0 {
		if _, err := resolved.file.Seek(offset, io.SeekStart); err != nil {
			return ReadResult{}, sanitizeError(err)
		}
	}
	limited := io.LimitReader(resolved.file, maxBytes+1)
	bytesRead, err := io.ReadAll(limited)
	if err != nil {
		return ReadResult{}, sanitizeError(err)
	}
	truncated := int64(len(bytesRead)) > maxBytes
	if truncated {
		bytesRead = bytesRead[:maxBytes]
	}
	metadata := engine.metadata(ctx, resolved, subpath)
	metadata.Offset = offset
	metadata.MaxBytes = maxBytes
	metadata.Truncated = truncated
	if truncated {
		metadata.NextOffset = offset + int64(len(bytesRead))
	}
	metadata.SHA256 = sha256Hex(bytesRead)
	if pdftext.IsPDFBytes(bytesRead) {
		metadata.Binary = true
		metadata.Cap = "pdf_text"
		return ReadResult{Metadata: metadata}, nil
	}
	if likelyBinary(bytesRead) {
		metadata.Binary = true
		return ReadResult{Metadata: metadata}, nil
	}
	return ReadResult{Content: string(bytesRead), Metadata: metadata}, nil
}

func (engine *Engine) ReadPDFText(ctx context.Context, req ReadRequest) (ReadResult, error) {
	target, subpath, err := TargetRelativePath(req.RelativePath, req.Subpath)
	if err != nil {
		return ReadResult{}, err
	}
	resolved, err := engine.resolve(req.RootID, target, false)
	if err != nil {
		return ReadResult{}, err
	}
	if resolved.info.IsDir() {
		return ReadResult{}, fmt.Errorf("%w: read target is a directory", ErrInvalidInput)
	}
	if !resolved.info.Mode().IsRegular() {
		return ReadResult{}, fmt.Errorf("%w: read target is not a regular file", ErrInvalidInput)
	}
	defer resolved.close()
	if resolved.info.Size() > engine.limits.MaxPDFReadBytes {
		return ReadResult{}, fmt.Errorf("%w: PDF source is larger than the configured read limit", ErrInvalidInput)
	}
	chunk, err := pdftext.ExtractChunkFromReaderAt(resolved.file, resolved.info.Size(), int(req.Offset), int(req.MaxBytes))
	if err != nil {
		return ReadResult{}, fmt.Errorf("%w: PDF text extraction failed: %v", ErrInvalidInput, err)
	}
	metadata := engine.metadata(ctx, resolved, subpath)
	metadata.Offset = int64(chunk.Offset)
	metadata.MaxBytes = int64(normalizedPDFMaxBytes(req.MaxBytes))
	metadata.NextOffset = int64(chunk.NextOffset)
	metadata.Truncated = chunk.Truncated
	metadata.Extraction = "pdf_text"
	metadata.PageCount = chunk.PageCount
	metadata.TextLength = int64(chunk.ContentLength)
	metadata.TextLengthKnown = chunk.ContentLengthKnown
	metadata.Cap = "pdf_text"
	return ReadResult{Content: chunk.Text, Metadata: metadata}, nil
}

func (engine *Engine) Tree(ctx context.Context, req TreeRequest) (TreeResult, error) {
	target, subpath, err := TargetRelativePath(req.RelativePath, req.Subpath)
	if err != nil {
		return TreeResult{}, err
	}
	resolved, err := engine.resolve(req.RootID, target, true)
	if err != nil {
		return TreeResult{}, err
	}
	if !resolved.info.IsDir() {
		resolved.close()
		return TreeResult{}, fmt.Errorf("%w: tree target is not a directory", ErrInvalidInput)
	}
	defer resolved.close()
	depth := req.Depth
	if depth <= 0 || depth > engine.limits.MaxTreeDepth {
		depth = engine.limits.MaxTreeDepth
	}
	limit := req.Limit
	if limit <= 0 || limit > engine.limits.MaxDirectoryEntries {
		limit = engine.limits.MaxDirectoryEntries
	}
	result := TreeResult{
		RootID:       resolved.root.id,
		RootAlias:    resolved.root.alias,
		RelativePath: resolved.relativePath,
		Metadata:     engine.metadata(ctx, resolved, subpath),
	}
	err = engine.walkTree(resolved, depth, limit, &result)
	if err != nil {
		return TreeResult{}, err
	}
	return result, nil
}

func (engine *Engine) Grep(ctx context.Context, req GrepRequest) (GrepResult, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return GrepResult{}, fmt.Errorf("%w: grep query is required", ErrInvalidInput)
	}
	target, subpath, err := TargetRelativePath(req.RelativePath, req.Subpath)
	if err != nil {
		return GrepResult{}, err
	}
	resolved, err := engine.resolve(req.RootID, target, true)
	if err != nil {
		return GrepResult{}, err
	}
	defer resolved.close()
	limit := req.MaxSnippets
	if limit <= 0 || limit > engine.limits.MaxReturnedSnippets {
		limit = engine.limits.MaxReturnedSnippets
	}
	result := GrepResult{
		RootID:       resolved.root.id,
		RootAlias:    resolved.root.alias,
		RelativePath: resolved.relativePath,
		Query:        query,
		Metadata:     engine.metadata(ctx, resolved, subpath),
	}
	var totalBytes int64
	filesSeen := 0
	visit := func(file resolvedPath) error {
		if filesSeen >= engine.limits.MaxRecursiveFiles || totalBytes >= engine.limits.MaxGrepTotalBytes || len(result.Matches) >= limit {
			result.Truncated = true
			return nil
		}
		if !file.info.Mode().IsRegular() {
			return nil
		}
		filesSeen++
		readBytes := file.info.Size()
		if readBytes > engine.limits.MaxGrepBytesPerFile {
			readBytes = engine.limits.MaxGrepBytesPerFile
			result.Truncated = true
		}
		if totalBytes+readBytes > engine.limits.MaxGrepTotalBytes {
			readBytes = engine.limits.MaxGrepTotalBytes - totalBytes
			result.Truncated = true
		}
		if readBytes <= 0 {
			return nil
		}
		matches, consumed, truncated, err := grepFile(file, query, readBytes, engine.limits.MaxSingleLineBytes, limit-len(result.Matches))
		if err != nil {
			return err
		}
		totalBytes += consumed
		if truncated {
			result.Truncated = true
		}
		result.Matches = append(result.Matches, matches...)
		if len(result.Matches) >= limit {
			result.Truncated = true
		}
		return nil
	}
	if resolved.info.IsDir() {
		err = engine.walkFiles(ctx, resolved, visit)
	} else {
		err = visit(resolved)
	}
	if err != nil {
		return GrepResult{}, err
	}
	result.Metadata.Truncated = result.Truncated
	if len(result.Matches) > 0 {
		result.Metadata.SHA256 = result.Matches[0].SHA256
	}
	return result, nil
}

type resolvedPath struct {
	root         root
	relativePath string
	absPath      string
	info         fs.FileInfo
	file         *os.File
}

func (resolved resolvedPath) close() {
	if resolved.file != nil {
		_ = resolved.file.Close()
	}
}

func (engine *Engine) resolve(rootID string, relativePath string, allowDirectory bool) (resolvedPath, error) {
	rootID = strings.TrimSpace(rootID)
	root, ok := engine.roots[rootID]
	if !ok {
		return resolvedPath{}, fmt.Errorf("%w: unknown root id", ErrInvalidInput)
	}
	relative, err := NormalizeRelativePath(relativePath)
	if err != nil {
		return resolvedPath{}, err
	}
	if denied, pattern := engine.denied(relative); denied {
		return resolvedPath{}, fmt.Errorf("%w: path denied by policy %q", ErrInvalidInput, pattern)
	}
	return openRelativePath(root, relative, allowDirectory)
}

func openRelativePath(root root, relative string, allowDirectory bool) (resolvedPath, error) {
	rootFile, err := openRootDirectory(root.canonicalPath)
	if err != nil {
		return resolvedPath{}, err
	}
	if relative == "." {
		resolved, err := resolvedFromOpened(root, relative, root.canonicalPath, rootFile, allowDirectory)
		if err != nil {
			_ = rootFile.Close()
			return resolvedPath{}, err
		}
		return resolved, nil
	}
	current := rootFile
	parts := strings.Split(relative, "/")
	for index, part := range parts {
		final := index == len(parts)-1
		next, err := openChildNoFollow(current, part, !final)
		_ = current.Close()
		if err != nil {
			return resolvedPath{}, err
		}
		current = next
		if !final {
			continue
		}
		absPath := filepath.Join(root.canonicalPath, filepath.FromSlash(relative))
		resolved, err := resolvedFromOpened(root, relative, absPath, current, allowDirectory)
		if err != nil {
			_ = current.Close()
			return resolvedPath{}, err
		}
		return resolved, nil
	}
	_ = current.Close()
	return resolvedPath{}, fmt.Errorf("%w: path is required", ErrInvalidInput)
}

func openRootDirectory(rootPath string) (*os.File, error) {
	fd, err := unix.Open(rootPath, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, sanitizeError(err)
	}
	return os.NewFile(uintptr(fd), rootPath), nil
}

func openChildNoFollow(parent *os.File, name string, requireDirectory bool) (*os.File, error) {
	flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK
	if requireDirectory {
		flags |= unix.O_DIRECTORY
	}
	fd, err := unix.Openat(int(parent.Fd()), name, flags, 0)
	if err != nil {
		return nil, sanitizeError(err)
	}
	return os.NewFile(uintptr(fd), name), nil
}

func resolvedFromOpened(root root, relative string, absPath string, file *os.File, allowDirectory bool) (resolvedPath, error) {
	info, err := file.Stat()
	if err != nil {
		return resolvedPath{}, sanitizeError(err)
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return resolvedPath{}, fmt.Errorf("%w: special files are not supported", ErrInvalidInput)
	}
	if info.IsDir() && !allowDirectory {
		return resolvedPath{}, fmt.Errorf("%w: directory is not valid for this operation", ErrInvalidInput)
	}
	return resolvedPath{root: root, relativePath: relative, absPath: filepath.Clean(absPath), info: info, file: file}, nil
}

func NormalizeRelativePath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		value = "."
	}
	if strings.ContainsRune(value, 0) || containsControl(value) {
		return "", fmt.Errorf("%w: relative path contains control characters", ErrInvalidInput)
	}
	if filepath.IsAbs(value) || strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\`) || looksLikeWindowsAbs(value) {
		return "", fmt.Errorf("%w: absolute paths are not accepted", ErrInvalidInput)
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return "", fmt.Errorf("%w: path traversal is not accepted", ErrInvalidInput)
		}
	}
	cleaned := path.Clean(value)
	if cleaned == "/" {
		cleaned = "."
	}
	for _, part := range strings.Split(cleaned, "/") {
		if part == ".." {
			return "", fmt.Errorf("%w: path traversal is not accepted", ErrInvalidInput)
		}
	}
	return cleaned, nil
}

func TargetRelativePath(relativePath string, subpath string) (string, string, error) {
	relative, err := NormalizeRelativePath(relativePath)
	if err != nil {
		return "", "", err
	}
	cleanSubpath, err := NormalizeRelativePath(subpath)
	if err != nil {
		return "", "", err
	}
	if cleanSubpath == "." {
		return relative, "", nil
	}
	return joinRelative(relative, cleanSubpath), cleanSubpath, nil
}

func (engine *Engine) walkTree(resolved resolvedPath, depth int, limit int, result *TreeResult) error {
	if len(result.Entries) >= limit {
		result.Truncated = true
		return nil
	}
	entries, err := resolved.file.ReadDir(-1)
	if err != nil {
		return sanitizeError(err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if len(result.Entries) >= limit {
			result.Truncated = true
			return nil
		}
		childRel := joinRelative(resolved.relativePath, entry.Name())
		if denied, pattern := engine.denied(childRel); denied {
			result.Entries = append(result.Entries, TreeEntry{Name: entry.Name(), RelativePath: childRel, Denied: true, Reason: "denied by " + pattern})
			continue
		}
		child, err := openChildResolved(resolved, childRel, entry.Name(), true)
		if err != nil {
			result.Entries = append(result.Entries, TreeEntry{Name: entry.Name(), RelativePath: childRel, Denied: true, Reason: "unreadable or escapes root"})
			continue
		}
		info := child.info
		kind := pathKind(info)
		result.Entries = append(result.Entries, TreeEntry{Name: entry.Name(), RelativePath: childRel, PathKind: kind, Size: info.Size(), MTime: formatTime(info.ModTime())})
		if depth > 1 && info.IsDir() {
			if err := engine.walkTree(child, depth-1, limit, result); err != nil {
				child.close()
				return err
			}
		}
		child.close()
	}
	return nil
}

func (engine *Engine) walkFiles(ctx context.Context, resolved resolvedPath, visit func(resolvedPath) error) error {
	files := 0
	var bytesSeen int64
	var walk func(resolvedPath) error
	walk = func(current resolvedPath) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		entries, err := current.file.ReadDir(-1)
		if err != nil {
			return sanitizeError(err)
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			childRel := joinRelative(current.relativePath, entry.Name())
			if denied, _ := engine.denied(childRel); denied {
				continue
			}
			child, err := openChildResolved(current, childRel, entry.Name(), true)
			if err != nil {
				continue
			}
			if child.info.IsDir() {
				err = walk(child)
				child.close()
				if err != nil {
					return err
				}
				continue
			}
			if !child.info.Mode().IsRegular() {
				child.close()
				continue
			}
			files++
			bytesSeen += child.info.Size()
			if files > engine.limits.MaxRecursiveFiles || bytesSeen > engine.limits.MaxRecursiveBytes {
				child.close()
				return nil
			}
			if err := visit(child); err != nil {
				child.close()
				return err
			}
			child.close()
		}
		return nil
	}
	return walk(resolved)
}

func (engine *Engine) metadata(ctx context.Context, resolved resolvedPath, subpath string) PathMetadata {
	metadata := PathMetadata{
		ObservedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		RootID:       resolved.root.id,
		RootAlias:    resolved.root.alias,
		RelativePath: resolved.relativePath,
		Subpath:      subpath,
		PathKind:     pathKind(resolved.info),
		Size:         resolved.info.Size(),
		MTime:        formatTime(resolved.info.ModTime()),
	}
	if git := engine.gitMetadata(ctx, resolved); git != nil {
		metadata.Git = git
	}
	return metadata
}

func (engine *Engine) gitMetadata(ctx context.Context, resolved resolvedPath) *GitMetadata {
	ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	dir := resolved.absPath
	if !resolved.info.IsDir() {
		dir = filepath.Dir(resolved.absPath)
	}
	top, err := gitOutput(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil
	}
	top = filepath.Clean(top)
	if !insideRoot(resolved.root.canonicalPath, top) {
		return nil
	}
	branch, _ := gitOutput(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
	head, _ := gitOutput(ctx, dir, "rev-parse", "HEAD")
	status, _ := gitOutput(ctx, dir, "status", "--porcelain")
	rel, _ := filepath.Rel(resolved.root.canonicalPath, top)
	rel = filepath.ToSlash(rel)
	if rel == "" {
		rel = "."
	}
	return &GitMetadata{Branch: branch, Head: head, Dirty: strings.TrimSpace(status) != "", WorktreeRelativePath: rel}
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func grepFile(resolved resolvedPath, query string, maxBytes int64, maxLineBytes int, limit int) ([]GrepMatch, int64, bool, error) {
	if _, err := resolved.file.Seek(0, io.SeekStart); err != nil {
		return nil, 0, false, sanitizeError(err)
	}
	limited := io.LimitReader(resolved.file, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, 0, false, sanitizeError(err)
	}
	truncated := int64(len(data)) > maxBytes
	if truncated {
		data = data[:maxBytes]
	}
	if likelyBinary(data) {
		return nil, int64(len(data)), truncated, nil
	}
	sum := sha256Hex(data)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	lowerQuery := strings.ToLower(query)
	var matches []GrepMatch
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		pos := strings.Index(strings.ToLower(line), lowerQuery)
		if pos < 0 {
			continue
		}
		matches = append(matches, GrepMatch{RelativePath: resolved.relativePath, Line: lineNo, Column: pos + 1, Snippet: snippet(line, pos, len(query)), SHA256: sum})
		if len(matches) >= limit {
			truncated = true
			break
		}
	}
	if err := scanner.Err(); err != nil {
		truncated = true
	}
	return matches, int64(len(data)), truncated, nil
}

func openChildResolved(parent resolvedPath, relative string, name string, allowDirectory bool) (resolvedPath, error) {
	childFile, err := openChildNoFollow(parent.file, name, false)
	if err != nil {
		return resolvedPath{}, err
	}
	absPath := filepath.Join(parent.root.canonicalPath, filepath.FromSlash(relative))
	resolved, err := resolvedFromOpened(parent.root, relative, absPath, childFile, allowDirectory)
	if err != nil {
		_ = childFile.Close()
		return resolvedPath{}, err
	}
	return resolved, nil
}

func (engine *Engine) denied(relative string) (bool, string) {
	if relative == "." {
		return false, ""
	}
	for _, part := range strings.Split(relative, "/") {
		for _, pattern := range engine.denyPatterns {
			if pattern == "" {
				continue
			}
			if part == pattern {
				return true, pattern
			}
			if ok, _ := path.Match(pattern, part); ok {
				return true, pattern
			}
		}
	}
	return false, ""
}

func defaultLimits(limits Limits) Limits {
	if limits.MaxFileReadBytes <= 0 {
		limits.MaxFileReadBytes = 64 * 1024
	}
	if limits.MaxGrepBytesPerFile <= 0 {
		limits.MaxGrepBytesPerFile = 256 * 1024
	}
	if limits.MaxGrepTotalBytes <= 0 {
		limits.MaxGrepTotalBytes = 2 * 1024 * 1024
	}
	if limits.MaxDirectoryEntries <= 0 {
		limits.MaxDirectoryEntries = 200
	}
	if limits.MaxRecursiveFiles <= 0 {
		limits.MaxRecursiveFiles = 1000
	}
	if limits.MaxRecursiveBytes <= 0 {
		limits.MaxRecursiveBytes = 16 * 1024 * 1024
	}
	if limits.MaxPDFReadBytes <= 0 {
		limits.MaxPDFReadBytes = 100 * 1024 * 1024
	}
	if limits.MaxTreeDepth <= 0 {
		limits.MaxTreeDepth = 3
	}
	if limits.MaxReturnedSnippets <= 0 {
		limits.MaxReturnedSnippets = 20
	}
	if limits.MaxSingleLineBytes <= 0 {
		limits.MaxSingleLineBytes = 64 * 1024
	}
	return limits
}

func normalizedPDFMaxBytes(maxBytes int64) int {
	if maxBytes <= 0 {
		return pdftext.DefaultChunkMaxBytes
	}
	if maxBytes > pdftext.MaxChunkBytes {
		return pdftext.MaxChunkBytes
	}
	return int(maxBytes)
}

func validRootID(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func insideRoot(rootPath string, candidate string) bool {
	rootPath = filepath.Clean(rootPath)
	candidate = filepath.Clean(candidate)
	if candidate == rootPath {
		return true
	}
	rel, err := filepath.Rel(rootPath, candidate)
	return err == nil && rel != "." && rel != "" && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)
}

func pathKind(info fs.FileInfo) string {
	if info.IsDir() {
		return "directory"
	}
	if info.Mode().IsRegular() {
		return "file"
	}
	return "special"
}

func joinRelative(base string, name string) string {
	if base == "." || base == "" {
		return name
	}
	return path.Clean(base + "/" + name)
}

func likelyBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return true
	}
	sample := data
	if len(sample) > 512 {
		sample = sample[:512]
	}
	contentType := http.DetectContentType(sample)
	return !strings.HasPrefix(contentType, "text/") && !utf8.Valid(sample)
}

func sanitizeError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: path not found", ErrInvalidInput)
	}
	if errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("%w: path permission denied", ErrInvalidInput)
	}
	return fmt.Errorf("%w: local path operation failed", ErrInvalidInput)
}

func containsControl(value string) bool {
	for _, r := range value {
		if r >= 0 && r < 0x20 {
			return true
		}
	}
	return false
}

func looksLikeWindowsAbs(value string) bool {
	if len(value) >= 3 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':' && (value[2] == '/' || value[2] == '\\') {
		return true
	}
	return strings.HasPrefix(value, "//")
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func snippet(line string, pos int, length int) string {
	start := pos - 48
	if start < 0 {
		start = 0
	}
	end := pos + length + 48
	if end > len(line) {
		end = len(line)
	}
	return strings.TrimSpace(line[start:end])
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

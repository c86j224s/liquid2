package source

import "context"

// LocalPathReader is the source-owned port for observing an allowlisted local path.
type LocalPathReader interface {
	Roots() []LocalPathRoot
	Inspect(context.Context, string, string) (LocalPathMetadata, error)
	IsPDF(context.Context, string, string) (bool, error)
	ReadFile(context.Context, LocalPathReadRequest) (LocalPathReadResult, error)
	ReadPDFText(context.Context, LocalPathReadRequest) (LocalPathReadResult, error)
	Tree(context.Context, LocalPathTreeRequest) (LocalPathTreeResult, error)
	Grep(context.Context, LocalPathGrepRequest) (LocalPathGrepResult, error)
}

// LocalPathRoot is the public identity of an allowlisted local path root.
type LocalPathRoot struct {
	RootID string `json:"root_id"`
	Alias  string `json:"alias,omitempty"`
}

// LocalPathMetadata describes a local path observation without exposing an absolute path.
type LocalPathMetadata struct {
	ObservedAt      string                `json:"observed_at"`
	RootID          string                `json:"root_id"`
	RootAlias       string                `json:"root_alias,omitempty"`
	RelativePath    string                `json:"relative_path"`
	Subpath         string                `json:"subpath,omitempty"`
	PathKind        string                `json:"path_kind"`
	Size            int64                 `json:"size,omitempty"`
	MTime           string                `json:"mtime,omitempty"`
	SHA256          string                `json:"sha256,omitempty"`
	Offset          int64                 `json:"offset,omitempty"`
	MaxBytes        int64                 `json:"max_bytes,omitempty"`
	NextOffset      int64                 `json:"next_offset,omitempty"`
	Truncated       bool                  `json:"truncated,omitempty"`
	Binary          bool                  `json:"binary,omitempty"`
	Extraction      string                `json:"extraction,omitempty"`
	PageCount       int                   `json:"page_count,omitempty"`
	TextLength      int64                 `json:"text_length,omitempty"`
	TextLengthKnown bool                  `json:"text_length_known"`
	Denied          []string              `json:"denied,omitempty"`
	Cap             string                `json:"cap,omitempty"`
	Git             *LocalPathGitMetadata `json:"git,omitempty"`
}

// LocalPathGitMetadata is optional provenance for a local path in a git worktree.
type LocalPathGitMetadata struct {
	Branch               string `json:"branch,omitempty"`
	Head                 string `json:"head,omitempty"`
	Dirty                bool   `json:"dirty"`
	WorktreeRelativePath string `json:"worktree_relative_path,omitempty"`
}

// LocalPathReadRequest identifies a bounded file observation.
type LocalPathReadRequest struct {
	RootID       string
	RelativePath string
	Subpath      string
	Offset       int64
	MaxBytes     int64
}

// LocalPathReadResult contains bounded content and its observation metadata.
type LocalPathReadResult struct {
	Content  string            `json:"content"`
	Metadata LocalPathMetadata `json:"metadata"`
}

// LocalPathTreeRequest identifies a bounded directory tree observation.
type LocalPathTreeRequest struct {
	RootID       string
	RelativePath string
	Subpath      string
	Depth        int
	Limit        int
}

// LocalPathTreeEntry is one item in a bounded directory tree result.
type LocalPathTreeEntry struct {
	Name         string `json:"name"`
	RelativePath string `json:"relative_path"`
	PathKind     string `json:"path_kind"`
	Size         int64  `json:"size,omitempty"`
	MTime        string `json:"mtime,omitempty"`
	Denied       bool   `json:"denied,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// LocalPathTreeResult contains a bounded directory tree and its metadata.
type LocalPathTreeResult struct {
	RootID       string               `json:"root_id"`
	RootAlias    string               `json:"root_alias,omitempty"`
	RelativePath string               `json:"relative_path"`
	Entries      []LocalPathTreeEntry `json:"entries"`
	Truncated    bool                 `json:"truncated"`
	Metadata     LocalPathMetadata    `json:"metadata"`
}

// LocalPathGrepRequest identifies a bounded content search.
type LocalPathGrepRequest struct {
	RootID       string
	RelativePath string
	Subpath      string
	Query        string
	MaxSnippets  int
}

// LocalPathGrepMatch is one bounded search match.
type LocalPathGrepMatch struct {
	RelativePath string `json:"relative_path"`
	Line         int    `json:"line"`
	Column       int    `json:"column"`
	Snippet      string `json:"snippet"`
	SHA256       string `json:"sha256,omitempty"`
}

// LocalPathGrepResult contains bounded search matches and metadata.
type LocalPathGrepResult struct {
	RootID       string               `json:"root_id"`
	RootAlias    string               `json:"root_alias,omitempty"`
	RelativePath string               `json:"relative_path"`
	Query        string               `json:"query"`
	Matches      []LocalPathGrepMatch `json:"matches"`
	Truncated    bool                 `json:"truncated"`
	Metadata     LocalPathMetadata    `json:"metadata"`
}

package source

import (
	"errors"
	"testing"
)

func TestNormalizeLocalRelativePathParity(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty becomes dot", input: "", want: "."},
		{name: "trim and slash", input: `  docs\\nested/./note.txt  `, want: "docs/nested/note.txt"},
		{name: "absolute slash rejected", input: "////", wantErr: true},
		{name: "parent segment rejected", input: "docs/../note.txt", wantErr: true},
		{name: "absolute rejected", input: "/etc/passwd", wantErr: true},
		{name: "windows absolute rejected", input: `C:\\repo\\note.txt`, wantErr: true},
		{name: "control rejected", input: "docs/\x00note.txt", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeLocalRelativePath(tt.input)
			if tt.wantErr {
				if err == nil || !errors.Is(err, ErrLocalPathInput) {
					t.Fatalf("NormalizeLocalRelativePath(%q) error = %v, want ErrLocalPathInput", tt.input, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeLocalRelativePath(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeLocalRelativePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLocalPathTargetRelativePathParity(t *testing.T) {
	tests := []struct {
		name       string
		relative   string
		subpath    string
		wantTarget string
		wantSub    string
	}{
		{name: "no subpath preserves base", relative: "docs/guide.txt", subpath: ".", wantTarget: "docs/guide.txt"},
		{name: "directory subpath joins", relative: "docs", subpath: `nested\\notes`, wantTarget: "docs/nested/notes", wantSub: "nested/notes"},
		{name: "dot base joins", relative: ".", subpath: "nested", wantTarget: "nested", wantSub: "nested"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTarget, gotSub, err := LocalPathTargetRelativePath(tt.relative, tt.subpath)
			if err != nil {
				t.Fatalf("LocalPathTargetRelativePath returned error: %v", err)
			}
			if gotTarget != tt.wantTarget || gotSub != tt.wantSub {
				t.Fatalf("LocalPathTargetRelativePath(%q, %q) = (%q, %q), want (%q, %q)", tt.relative, tt.subpath, gotTarget, gotSub, tt.wantTarget, tt.wantSub)
			}
		})
	}

	if _, _, err := LocalPathTargetRelativePath("docs", "../outside"); err == nil || !errors.Is(err, ErrLocalPathInput) {
		t.Fatalf("traversal target error = %v, want ErrLocalPathInput", err)
	}
}

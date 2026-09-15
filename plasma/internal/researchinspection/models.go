package researchinspection

import "github.com/c86j224s/liquid2/plasma/internal/researchcatalog"

// ReadRequest bounds a research object body read and optional report children page.
type ReadRequest struct {
	MissionID  string
	ObjectKind string
	ObjectID   string
	Offset     int
	MaxBytes   int
	Cursor     string
	Limit      int
	Legacy     bool
}

// ObjectRead contains a bounded object body and, for report versions, its children page.
type ObjectRead struct {
	ObjectKind string                      `json:"object_kind"`
	ObjectID   string                      `json:"object_id"`
	MissionID  string                      `json:"mission_id"`
	Summary    string                      `json:"summary"`
	Refs       []researchcatalog.ObjectRef `json:"refs,omitempty"`
	Data       string                      `json:"data"`
	Truncated  bool                        `json:"truncated"`
	NextOffset int                         `json:"next_offset,omitempty"`
	Children   *researchcatalog.Page       `json:"children,omitempty"`
}

// GrepMatch is one literal mission-object search match and its bounded snippet.
type GrepMatch struct {
	ObjectKind string                      `json:"object_kind"`
	ObjectID   string                      `json:"object_id"`
	MissionID  string                      `json:"mission_id"`
	Snippet    string                      `json:"snippet"`
	Position   int                         `json:"position"`
	Refs       []researchcatalog.ObjectRef `json:"refs,omitempty"`
}

// GrepResult is one paginated literal mission-object search result.
type GrepResult struct {
	MissionID  string      `json:"mission_id"`
	Query      string      `json:"query"`
	Matches    []GrepMatch `json:"matches"`
	NextCursor string      `json:"next_cursor,omitempty"`
	Limit      int         `json:"limit"`
	Truncated  bool        `json:"truncated"`
}

// GrepCandidate is the app-owned materialization value passed to inspection.
type GrepCandidate struct {
	Summary researchcatalog.ObjectSummary
	Text    string
}

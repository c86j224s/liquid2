package researchinspection

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/researchcatalog"
)

const (
	DefaultBytes = 4096
	MaxBytes     = 32768
)

// Dependencies are the app-owned materialization callbacks used by Service.
// Callbacks are required on executed paths and are intentionally not validated eagerly.
type Dependencies struct {
	ReadChunked           func(context.Context, string, string, string, int, int) (ObjectRead, bool, error)
	ReadPayload           func(context.Context, string, string, string, bool) (researchcatalog.ObjectSummary, []byte, error)
	ReportVersionChildren func(context.Context, string, string, int, string) (researchcatalog.Page, error)
	GrepCandidates        func(context.Context, string, string, bool) ([]GrepCandidate, error)
}

// Service validates and sequences bounded research object reads without owning
// source I/O, PDF/local-path observation, security, visibility, or materialization.
type Service struct {
	deps Dependencies
}

// NewService constructs a bounded research object read service.
func NewService(deps Dependencies) *Service {
	return &Service{deps: deps}
}

// Read performs validation, chunked handling, payload materialization, UTF-8
// chunking, and report-version children paging in that exact order.
func (s *Service) Read(ctx context.Context, req ReadRequest) (ObjectRead, error) {
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return ObjectRead{}, err
	}
	objectKind := researchcatalog.NormalizeObjectKind(req.ObjectKind)
	objectID := strings.TrimSpace(req.ObjectID)
	if objectID == "" {
		return ObjectRead{}, fmt.Errorf("%w: object id is required", producterror.ErrInvalidInput)
	}
	maxBytes := ClampBytes(req.MaxBytes)
	offset := req.Offset
	if offset < 0 {
		return ObjectRead{}, fmt.Errorf("%w: offset must be non-negative", producterror.ErrInvalidInput)
	}
	if read, handled, err := s.deps.ReadChunked(ctx, missionID, objectKind, objectID, offset, maxBytes); handled || err != nil {
		return read, err
	}
	summary, data, err := s.deps.ReadPayload(ctx, missionID, objectKind, objectID, req.Legacy)
	if err != nil {
		return ObjectRead{}, err
	}
	chunk, truncated, nextOffset, err := ChunkBytes(data, offset, maxBytes)
	if err != nil {
		return ObjectRead{}, err
	}
	read := ObjectRead{ObjectKind: objectKind, ObjectID: objectID, MissionID: missionID, Summary: summary.Summary, Refs: summary.Refs, Data: string(chunk), Truncated: truncated, NextOffset: nextOffset}
	if objectKind == researchcatalog.ObjectReportVersion {
		children, err := s.deps.ReportVersionChildren(ctx, missionID, objectID, req.Limit, req.Cursor)
		if err != nil {
			return ObjectRead{}, err
		}
		read.Children = &children
	}
	return read, nil
}

// ClampBytes applies the established default and maximum byte limits.
func ClampBytes(maxBytes int) int {
	if maxBytes <= 0 {
		return DefaultBytes
	}
	if maxBytes > MaxBytes {
		return MaxBytes
	}
	return maxBytes
}

// ChunkBytes returns a UTF-8-safe range and continuation offset.
func ChunkBytes(data []byte, offset, maxBytes int) ([]byte, bool, int, error) {
	if !utf8.Valid(data) {
		return nil, false, 0, fmt.Errorf("%w: object payload is not UTF-8 text", producterror.ErrInvalidInput)
	}
	if offset > len(data) {
		return nil, false, 0, fmt.Errorf("%w: object payload offset is beyond content length", producterror.ErrInvalidInput)
	}
	if offset < len(data) && !utf8.RuneStart(data[offset]) {
		return nil, false, 0, fmt.Errorf("%w: object payload offset must align to UTF-8 boundary", producterror.ErrInvalidInput)
	}
	if offset == len(data) {
		return []byte{}, false, 0, nil
	}
	end := offset + maxBytes
	if end > len(data) {
		end = len(data)
	}
	for end > offset && !utf8.Valid(data[offset:end]) {
		end--
	}
	if end == offset {
		return nil, false, 0, fmt.Errorf("%w: object payload could not be sliced as UTF-8", producterror.ErrInvalidInput)
	}
	chunk := append([]byte(nil), data[offset:end]...)
	if end < len(data) {
		return chunk, true, end, nil
	}
	return chunk, false, 0, nil
}

func validateID(prefix, id string) error {
	if !strings.HasPrefix(id, prefix) || len(id) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", producterror.ErrInvalidInput, prefix)
	}
	return nil
}

package researchinspection

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/researchcatalog"
)

const snippetContext = 48

// Grep validates and sequences a bounded, case-insensitive literal search over
// app-materialized candidates. Matching positions remain byte offsets, as in
// the established application behavior.
func (s *Service) Grep(ctx context.Context, missionID, query string, limit int, cursor string, legacy bool) (GrepResult, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateID("mis_", missionID); err != nil {
		return GrepResult{}, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return GrepResult{}, fmt.Errorf("%w: grep query is required", producterror.ErrInvalidInput)
	}
	limit = researchcatalog.ClampLimit(limit)
	offset, err := researchcatalog.ParseCursor(cursor)
	if err != nil {
		return GrepResult{}, err
	}
	candidates, err := s.deps.GrepCandidates(ctx, missionID, query, legacy)
	if err != nil {
		return GrepResult{}, err
	}
	var matches []GrepMatch
	lowerQuery := strings.ToLower(query)
	for _, candidate := range candidates {
		lowerText := strings.ToLower(candidate.Text)
		for searchStart := 0; searchStart < len(lowerText); {
			pos := strings.Index(lowerText[searchStart:], lowerQuery)
			if pos < 0 {
				break
			}
			pos += searchStart
			matches = append(matches, GrepMatch{
				ObjectKind: candidate.Summary.ObjectKind,
				ObjectID:   candidate.Summary.ObjectID,
				MissionID:  missionID,
				Snippet:    snippet(candidate.Text, pos, len(query)),
				Position:   pos,
				Refs:       candidate.Summary.Refs,
			})
			searchStart = pos + len(lowerQuery)
		}
	}
	page, next, truncated := paginateMatches(matches, offset, limit)
	return GrepResult{MissionID: missionID, Query: query, Matches: page, NextCursor: next, Limit: limit, Truncated: truncated}, nil
}

func paginateMatches(items []GrepMatch, offset, limit int) ([]GrepMatch, string, bool) {
	if offset >= len(items) {
		return nil, "", false
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[offset:end], next, next != ""
}

func snippet(text string, pos, queryLen int) string {
	start := pos - snippetContext
	if start < 0 {
		start = 0
	}
	end := pos + queryLen + snippetContext
	if end > len(text) {
		end = len(text)
	}
	return strings.TrimSpace(text[start:end])
}

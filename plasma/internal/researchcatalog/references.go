package researchcatalog

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

// References preserves visibility-before-read and forward/backward pagination order.
func (s *Service) References(ctx context.Context, missionID, objectKind, objectID string, limit int, cursor string, legacy bool) (References, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateMissionID(missionID); err != nil {
		return References{}, err
	}
	limit = ClampLimit(limit)
	offset, err := ParseCursor(cursor)
	if err != nil {
		return References{}, err
	}
	objectKind = NormalizeObjectKind(objectKind)
	objectID = strings.TrimSpace(objectID)
	if !ObjectKindAllowed(objectKind, legacy) {
		return References{}, fmt.Errorf("%w: unsupported object kind", producterror.ErrInvalidInput)
	}
	reportArtifacts, err := s.deps.ReportArtifactIDsHiddenFromResearchDiscovery(ctx, missionID, legacy)
	if err != nil {
		return References{}, err
	}
	if err := s.deps.EnsureReferenceTargetVisible(ctx, missionID, objectKind, objectID, legacy, reportArtifacts); err != nil {
		return References{}, err
	}
	summary, err := s.deps.ReadReferenceSummary(ctx, missionID, objectKind, objectID, legacy)
	if err != nil {
		return References{}, err
	}
	summary.Refs = FilterReportArtifactRefs(summary.Refs, reportArtifacts)
	target := ObjectRef{ObjectKind: objectKind, ObjectID: objectID}
	all, err := s.AllObjectSummaries(ctx, missionID, "", legacy)
	if err != nil {
		return References{}, err
	}
	backward := BackwardReferences(all, target)
	forward, backward, next, truncated := PaginateReferenceSets(summary.Refs, backward, offset, limit)
	return References{
		MissionID:  missionID,
		ObjectKind: objectKind,
		ObjectID:   objectID,
		Forward:    forward,
		Backward:   backward,
		NextCursor: next,
		Limit:      limit,
		Truncated:  truncated,
	}, nil
}

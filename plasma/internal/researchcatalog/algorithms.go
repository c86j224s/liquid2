package researchcatalog

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// NormalizeObjectKind는 object kind를 identity를 바꾸지 않고 trim한다.
func NormalizeObjectKind(kind string) string {
	return strings.TrimSpace(kind)
}

// ObjectKindAllowed는 선택한 surface에서 kind를 사용할 수 있는지 반환한다.
func ObjectKindAllowed(kind string, legacy bool) bool {
	if legacy {
		return knownObjectKind(kind)
	}
	return defaultObjectKind(kind)
}

func knownObjectKind(kind string) bool {
	switch kind {
	case ObjectSourceSnapshot, ObjectRawArtifact, ObjectEvidenceRecord, ObjectClaimRecord, ObjectQuestionRecord, ObjectOptionRecord, ObjectProposalBundle, ObjectReport, ObjectReportVersion, ObjectReportBlock, ObjectLedgerEvent:
		return true
	default:
		return false
	}
}

func defaultObjectKind(kind string) bool {
	switch kind {
	case ObjectSourceSnapshot, ObjectRawArtifact, ObjectLedgerEvent:
		return true
	default:
		return false
	}
}

// ClampLimit는 기존 research list limit의 기본값과 상한을 적용한다.
func ClampLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

// ParseCursor는 기존 decimal offset cursor contract를 해석한다.
func ParseCursor(cursor string) (int, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(cursor)
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("%w: invalid cursor", producterror.ErrInvalidInput)
	}
	return offset, nil
}

// PaginateSummaries는 slice 순서를 보존하며 cursor page 하나를 반환한다.
func PaginateSummaries(items []ObjectSummary, offset, limit int) ([]ObjectSummary, string, bool) {
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

// PaginateReferenceSets는 forward 뒤에 backward를 둔 하나의 stream으로 page를 만든다.
func PaginateReferenceSets(forward []ObjectRef, backward []ObjectRef, offset, limit int) ([]ObjectRef, []ObjectRef, string, bool) {
	type referencedItem struct {
		direction string
		ref       ObjectRef
	}
	combined := make([]referencedItem, 0, len(forward)+len(backward))
	for _, ref := range forward {
		combined = append(combined, referencedItem{direction: "forward", ref: ref})
	}
	for _, ref := range backward {
		combined = append(combined, referencedItem{direction: "backward", ref: ref})
	}
	if offset >= len(combined) {
		return nil, nil, "", false
	}
	end := offset + limit
	if end > len(combined) {
		end = len(combined)
	}
	var pageForward []ObjectRef
	var pageBackward []ObjectRef
	for _, item := range combined[offset:end] {
		if item.direction == "forward" {
			pageForward = append(pageForward, item.ref)
		} else {
			pageBackward = append(pageBackward, item.ref)
		}
	}
	next := ""
	if end < len(combined) {
		next = strconv.Itoa(end)
	}
	return pageForward, pageBackward, next, next != ""
}

// ContainsRef는 refs에 target이 포함되는지 반환한다.
func ContainsRef(refs []ObjectRef, target ObjectRef) bool {
	for _, ref := range refs {
		if ref == target {
			return true
		}
	}
	return false
}

// BackwardReferences는 target 자신을 제외하고 입력 순서대로 referencing object를 수집한다.
func BackwardReferences(items []ObjectSummary, target ObjectRef) []ObjectRef {
	var backward []ObjectRef
	for _, item := range items {
		if item.ObjectKind == target.ObjectKind && item.ObjectID == target.ObjectID {
			continue
		}
		if ContainsRef(item.Refs, target) {
			backward = append(backward, ObjectRef{ObjectKind: item.ObjectKind, ObjectID: item.ObjectID})
		}
	}
	return backward
}

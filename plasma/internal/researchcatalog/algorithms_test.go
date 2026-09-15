package researchcatalog

import (
	"errors"
	"reflect"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestParseCursor(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
		err  bool
	}{
		{name: "empty", in: "", want: 0},
		{name: "trimmed", in: " 12 ", want: 12},
		{name: "malformed", in: "abc", err: true},
		{name: "negative", in: "-1", err: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCursor(tt.in)
			if tt.err {
				if err == nil || !errors.Is(err, producterror.ErrInvalidInput) || err.Error() != "invalid input: invalid cursor" {
					t.Fatalf("ParseCursor(%q) error = %v", tt.in, err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("ParseCursor(%q) = %d, %v; want %d", tt.in, got, err, tt.want)
			}
		})
	}
}

func TestClampLimit(t *testing.T) {
	for _, tt := range []struct{ in, want int }{
		{0, 20}, {-1, 20}, {1, 1}, {100, 100}, {101, 100},
	} {
		if got := ClampLimit(tt.in); got != tt.want {
			t.Fatalf("ClampLimit(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestPaginateSummaries(t *testing.T) {
	items := []ObjectSummary{{ObjectID: "one"}, {ObjectID: "two"}, {ObjectID: "three"}}
	for _, tt := range []struct {
		name          string
		offset, limit int
		items         []ObjectSummary
		next          string
		truncated     bool
	}{
		{name: "first", offset: 0, limit: 2, items: items[:2], next: "2", truncated: true},
		{name: "last", offset: 2, limit: 2, items: items[2:]},
		{name: "out of range", offset: 3, limit: 2},
		{name: "empty nil", offset: 0, limit: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, next, truncated := PaginateSummaries(func() []ObjectSummary {
				if tt.name == "empty nil" {
					return nil
				}
				return items
			}(), tt.offset, tt.limit)
			if !reflect.DeepEqual(got, tt.items) || next != tt.next || truncated != tt.truncated {
				t.Fatalf("PaginateSummaries = %#v, %q, %v; want %#v, %q, %v", got, next, truncated, tt.items, tt.next, tt.truncated)
			}
		})
	}
}

func TestPaginateReferenceSetsSharedCursorBoundary(t *testing.T) {
	forward := []ObjectRef{{ObjectID: "f1"}, {ObjectID: "f2"}}
	backward := []ObjectRef{{ObjectID: "b1"}, {ObjectID: "b2"}}
	first, firstBack, next, truncated := PaginateReferenceSets(forward, backward, 1, 2)
	if !reflect.DeepEqual(first, []ObjectRef{{ObjectID: "f2"}}) || !reflect.DeepEqual(firstBack, []ObjectRef{{ObjectID: "b1"}}) || next != "3" || !truncated {
		t.Fatalf("first shared page = %#v, %#v, %q, %v", first, firstBack, next, truncated)
	}
	last, lastBack, next, truncated := PaginateReferenceSets(forward, backward, 3, 2)
	if len(last) != 0 || len(lastBack) != 1 || lastBack[0].ObjectID != "b2" || next != "" || truncated {
		t.Fatalf("last shared page = %#v, %#v, %q, %v", last, lastBack, next, truncated)
	}
	empty, emptyBack, next, truncated := PaginateReferenceSets(nil, nil, 0, 2)
	if empty != nil || emptyBack != nil || next != "" || truncated {
		t.Fatalf("empty shared page = %#v, %#v, %q, %v", empty, emptyBack, next, truncated)
	}
}

func TestBackwardReferencesPreservesOrderDuplicatesAndExcludesSelf(t *testing.T) {
	target := ObjectRef{ObjectKind: ObjectClaimRecord, ObjectID: "claim"}
	items := []ObjectSummary{
		{ObjectKind: ObjectClaimRecord, ObjectID: "claim", Refs: []ObjectRef{target}},
		{ObjectKind: ObjectEvidenceRecord, ObjectID: "e1", Refs: []ObjectRef{target}},
		{ObjectKind: ObjectEvidenceRecord, ObjectID: "e2", Refs: []ObjectRef{target, target}},
		{ObjectKind: ObjectEvidenceRecord, ObjectID: "e3"},
	}
	want := []ObjectRef{{ObjectKind: ObjectEvidenceRecord, ObjectID: "e1"}, {ObjectKind: ObjectEvidenceRecord, ObjectID: "e2"}}
	if got := BackwardReferences(items, target); !reflect.DeepEqual(got, want) {
		t.Fatalf("BackwardReferences = %#v, want %#v", got, want)
	}
	if got := BackwardReferences(nil, target); got != nil {
		t.Fatalf("BackwardReferences(nil) = %#v, want nil", got)
	}
}

func TestObjectKindAllowedLegacyGate(t *testing.T) {
	if !ObjectKindAllowed(ObjectClaimRecord, true) || ObjectKindAllowed(ObjectClaimRecord, false) {
		t.Fatal("legacy object kind gate changed")
	}
	if !ObjectKindAllowed(ObjectSourceSnapshot, false) || ObjectKindAllowed("unknown", true) {
		t.Fatal("object kind allowlist changed")
	}
}

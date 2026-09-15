package researchcatalog

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestReferencesVisibilityFailurePrecedesPayloadRead(t *testing.T) {
	sentinel := errors.New("hidden target")
	var calls []string
	svc := NewService(Dependencies{
		ReportArtifactIDsHiddenFromResearchDiscovery: func(context.Context, string, bool) (map[string]struct{}, error) {
			calls = append(calls, "hidden")
			return nil, nil
		},
		EnsureReferenceTargetVisible: func(context.Context, string, string, string, bool, map[string]struct{}) error {
			calls = append(calls, "gate")
			return sentinel
		},
		ReadReferenceSummary: func(context.Context, string, string, string, bool) (ObjectSummary, error) {
			t.Fatal("read after rejected gate")
			return ObjectSummary{}, nil
		},
	})
	_, err := svc.References(context.Background(), "mis_one", ObjectRawArtifact, "art_one", 10, "", false)
	if !errors.Is(err, sentinel) || !reflect.DeepEqual(calls, []string{"hidden", "gate"}) {
		t.Fatalf("calls=%v err=%v", calls, err)
	}
}

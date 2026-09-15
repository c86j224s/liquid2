package researchcatalog

import (
	"context"
	"errors"
	"reflect"
	"testing"

	artifact "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

type serviceCallLog struct {
	calls []string
}

func (l *serviceCallLog) add(name string) {
	l.calls = append(l.calls, name)
}

func (l *serviceCallLog) reset() {
	l.calls = nil
}

func emptyServiceDependencies(log *serviceCallLog) Dependencies {
	return Dependencies{
		GetMissionProjection: func(context.Context, string) (mission.Projection, error) {
			log.add("projection")
			return mission.Projection{}, nil
		},
		ListSourceSnapshots: func(context.Context, string) ([]source.Snapshot, error) {
			log.add("snapshots")
			return nil, nil
		},
		ListVisibleRawArtifacts: func(context.Context, string, bool) ([]artifact.Raw, error) {
			log.add("raw_artifacts")
			return nil, nil
		},
		ListEvents: func(context.Context, string) ([]ledger.Event, error) {
			log.add("events")
			return nil, nil
		},
		ListVisibleLedgerEvents: func(context.Context, string, bool) ([]ledger.Event, error) {
			log.add("visible_events")
			return nil, nil
		},
		ReportArtifactIDsHiddenFromResearchDiscovery: func(context.Context, string, bool) (map[string]struct{}, error) {
			log.add("hidden_artifacts")
			return nil, nil
		},
		ListEvidenceRecords: func(context.Context, string) ([]researchrecords.EvidenceRecord, error) {
			log.add("evidence")
			return nil, nil
		},
		ListClaimRecords: func(context.Context, string) ([]researchrecords.ClaimRecord, error) {
			log.add("claims")
			return nil, nil
		},
		ListQuestionRecords: func(context.Context, string) ([]researchrecords.QuestionRecord, error) {
			log.add("questions")
			return nil, nil
		},
		ListOptionRecords: func(context.Context, string) ([]researchrecords.OptionRecord, error) {
			log.add("options")
			return nil, nil
		},
		ListProposalSummaries: func(context.Context, string) ([]ObjectSummary, error) {
			log.add("proposals")
			return nil, nil
		},
		ListReportSummaries: func(context.Context, string) ([]ObjectSummary, error) {
			log.add("reports")
			return nil, nil
		},
		ListReportVersionSummaries: func(context.Context, string) ([]ObjectSummary, error) {
			log.add("versions")
			return nil, nil
		},
		ListReportBlockSummaries: func(context.Context, string) ([]ObjectSummary, error) {
			log.add("blocks")
			return nil, nil
		},
	}
}

func reportTestSummaries() ([]ObjectSummary, []ObjectSummary) {
	reports := []ObjectSummary{{ObjectKind: ObjectReport, ObjectID: "r1", MissionID: "mis_1", Summary: "report"}}
	versions := []ObjectSummary{
		{ObjectKind: ObjectReportVersion, ObjectID: "v1", MissionID: "mis_1", Summary: "version 1"},
		{ObjectKind: ObjectReportVersion, ObjectID: "v2", MissionID: "mis_1", Summary: "version 2"},
	}
	return reports, versions
}

func TestServiceAddReportsReadsBlocksForEveryLegacyReportRequest(t *testing.T) {
	ctx := context.Background()
	for _, objectKind := range []string{"", ObjectReport, ObjectReportVersion, ObjectReportBlock} {
		t.Run("kind="+objectKind, func(t *testing.T) {
			log := &serviceCallLog{}
			deps := emptyServiceDependencies(log)
			reports, versions := reportTestSummaries()
			deps.ListReportSummaries = func(context.Context, string) ([]ObjectSummary, error) {
				log.add("reports")
				return reports, nil
			}
			deps.ListReportVersionSummaries = func(context.Context, string) ([]ObjectSummary, error) {
				log.add("versions")
				return versions, nil
			}
			deps.ListReportBlockSummaries = func(_ context.Context, versionID string) ([]ObjectSummary, error) {
				log.add("blocks:" + versionID)
				return nil, nil
			}

			page, err := NewService(deps).List(ctx, "mis_1", objectKind, 20, "", true)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := log.calls, expectedReportCalls(objectKind); !reflect.DeepEqual(got, want) {
				t.Fatalf("callback calls = %#v, want %#v", got, want)
			}
			if objectKind == ObjectReport && len(page.Items) != 1 {
				t.Fatalf("report items = %#v, want one report", page.Items)
			}
			if objectKind == ObjectReportVersion && len(page.Items) != 2 {
				t.Fatalf("version items = %#v, want two versions", page.Items)
			}
			if objectKind == ObjectReportBlock && len(page.Items) != 0 {
				t.Fatalf("block items = %#v, want empty block result", page.Items)
			}
		})
	}
}

func expectedReportCalls(objectKind string) []string {
	if objectKind == "" {
		return []string{"hidden_artifacts", "snapshots", "raw_artifacts", "evidence", "claims", "questions", "options", "proposals", "reports", "versions", "blocks:v1", "blocks:v2", "visible_events"}
	}
	return []string{"hidden_artifacts", "reports", "versions", "blocks:v1", "blocks:v2"}
}

func TestServiceAddReportsPreservesErrorPrecedenceAndStopsAfterFailure(t *testing.T) {
	ctx := context.Background()
	reportErr := errors.New("reports failed")
	versionErr := errors.New("versions failed")
	blockErr := errors.New("blocks failed")

	t.Run("report error", func(t *testing.T) {
		log := &serviceCallLog{}
		deps := emptyServiceDependencies(log)
		deps.ListReportSummaries = func(context.Context, string) ([]ObjectSummary, error) {
			log.add("reports")
			return nil, reportErr
		}
		_, err := NewService(deps).List(ctx, "mis_1", ObjectReport, 20, "", true)
		if !errors.Is(err, reportErr) {
			t.Fatalf("error = %v, want %v", err, reportErr)
		}
		if got, want := log.calls, []string{"hidden_artifacts", "reports"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("callback calls = %#v, want %#v", got, want)
		}
	})

	t.Run("version error", func(t *testing.T) {
		log := &serviceCallLog{}
		deps := emptyServiceDependencies(log)
		deps.ListReportSummaries = func(context.Context, string) ([]ObjectSummary, error) {
			log.add("reports")
			return nil, nil
		}
		deps.ListReportVersionSummaries = func(context.Context, string) ([]ObjectSummary, error) {
			log.add("versions")
			return nil, versionErr
		}
		_, err := NewService(deps).List(ctx, "mis_1", ObjectReportVersion, 20, "", true)
		if !errors.Is(err, versionErr) {
			t.Fatalf("error = %v, want %v", err, versionErr)
		}
		if got, want := log.calls, []string{"hidden_artifacts", "reports", "versions"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("callback calls = %#v, want %#v", got, want)
		}
	})

	for _, objectKind := range []string{ObjectReport, ObjectReportVersion, ObjectReportBlock} {
		t.Run("block error stops later versions kind="+objectKind, func(t *testing.T) {
			log := &serviceCallLog{}
			deps := emptyServiceDependencies(log)
			reports, versions := reportTestSummaries()
			deps.ListReportSummaries = func(context.Context, string) ([]ObjectSummary, error) {
				log.add("reports")
				return reports, nil
			}
			deps.ListReportVersionSummaries = func(context.Context, string) ([]ObjectSummary, error) {
				log.add("versions")
				return versions, nil
			}
			deps.ListReportBlockSummaries = func(_ context.Context, versionID string) ([]ObjectSummary, error) {
				log.add("blocks:" + versionID)
				if versionID == "v1" {
					return nil, blockErr
				}
				return nil, nil
			}
			_, err := NewService(deps).List(ctx, "mis_1", objectKind, 20, "", true)
			if !errors.Is(err, blockErr) {
				t.Fatalf("error = %v, want %v", err, blockErr)
			}
			if got, want := log.calls, []string{"hidden_artifacts", "reports", "versions", "blocks:v1"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("callback calls = %#v, want %#v", got, want)
			}
		})
	}
}

func TestServiceOutlineLegacyAndCurrentPreserveOutputShape(t *testing.T) {
	ctx := context.Background()

	t.Run("legacy empty counts are explicit", func(t *testing.T) {
		log := &serviceCallLog{}
		deps := emptyServiceDependencies(log)
		outline, err := NewService(deps).Outline(ctx, " mis_1 ", true)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]int{
			ObjectSourceSnapshot: 0,
			ObjectRawArtifact:    0,
			ObjectLedgerEvent:    0,
			ObjectEvidenceRecord: 0,
			ObjectClaimRecord:    0,
			ObjectQuestionRecord: 0,
			ObjectOptionRecord:   0,
			ObjectProposalBundle: 0,
			ObjectReport:         0,
			ObjectReportVersion:  0,
		}
		if !reflect.DeepEqual(outline.Counts, want) {
			t.Fatalf("counts = %#v, want exact map %#v", outline.Counts, want)
		}
		for _, kind := range []string{ObjectReportBlock, ObjectEvidenceRecord + ".draft", ObjectClaimRecord + ".draft", ObjectQuestionRecord + ".draft", ObjectOptionRecord + ".draft"} {
			if _, ok := outline.Counts[kind]; ok {
				t.Fatalf("unexpected count key %q", kind)
			}
		}
	})

	t.Run("current excludes legacy count keys and legacy IO", func(t *testing.T) {
		log := &serviceCallLog{}
		deps := Dependencies{
			GetMissionProjection: func(context.Context, string) (mission.Projection, error) {
				log.add("projection")
				return mission.Projection{}, nil
			},
			ListSourceSnapshots: func(context.Context, string) ([]source.Snapshot, error) {
				log.add("snapshots")
				return nil, nil
			},
			ListVisibleRawArtifacts: func(context.Context, string, bool) ([]artifact.Raw, error) {
				log.add("raw_artifacts")
				return nil, nil
			},
			ListEvents: func(context.Context, string) ([]ledger.Event, error) {
				log.add("events")
				return nil, nil
			},
		}
		outline, err := NewService(deps).Outline(ctx, "mis_1", false)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := log.calls, []string{"projection", "snapshots", "raw_artifacts", "events"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("callback calls = %#v, want %#v", got, want)
		}
		want := map[string]int{ObjectSourceSnapshot: 0, ObjectRawArtifact: 0, ObjectLedgerEvent: 0}
		if !reflect.DeepEqual(outline.Counts, want) {
			t.Fatalf("counts = %#v, want exact map %#v", outline.Counts, want)
		}
		for _, kind := range []string{ObjectReportBlock, ObjectEvidenceRecord + ".draft", ObjectClaimRecord + ".draft", ObjectQuestionRecord + ".draft", ObjectOptionRecord + ".draft"} {
			if _, ok := outline.Counts[kind]; ok {
				t.Fatalf("unexpected current count key %q", kind)
			}
		}
	})
}

func TestServiceListInvalidCursorPerformsZeroIO(t *testing.T) {
	calls := 0
	deps := Dependencies{
		ReportArtifactIDsHiddenFromResearchDiscovery: func(context.Context, string, bool) (map[string]struct{}, error) {
			calls++
			return nil, nil
		},
	}
	_, err := NewService(deps).List(context.Background(), "mis_1", ObjectReport, 20, "not-a-cursor", true)
	if err == nil {
		t.Fatal("expected invalid cursor error")
	}
	if calls != 0 {
		t.Fatalf("callback calls = %d, want zero", calls)
	}
}

func TestServiceListVisibilityIsGlobalNotPerReference(t *testing.T) {
	log := &serviceCallLog{}
	deps := emptyServiceDependencies(log)
	deps.ListReportSummaries = func(context.Context, string) ([]ObjectSummary, error) {
		log.add("reports")
		return []ObjectSummary{{ObjectKind: ObjectReport, ObjectID: "r1", MissionID: "mis_1", Refs: []ObjectRef{{ObjectKind: ObjectRawArtifact, ObjectID: "a1"}, {ObjectKind: ObjectRawArtifact, ObjectID: "a2"}}}}, nil
	}
	deps.ListReportVersionSummaries = func(context.Context, string) ([]ObjectSummary, error) {
		log.add("versions")
		return nil, nil
	}
	filtered := 0
	deps.FilterReportArtifactSummaryRefs = func(items []ObjectSummary, _ map[string]struct{}) []ObjectSummary {
		filtered++
		return items
	}
	_, err := NewService(deps).List(context.Background(), "mis_1", ObjectReport, 20, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := log.calls, []string{"hidden_artifacts", "reports", "versions"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("callback calls = %#v, want %#v", got, want)
	}
	if filtered != 1 {
		t.Fatalf("summary visibility filtering calls = %d, want one global filtering call", filtered)
	}
}

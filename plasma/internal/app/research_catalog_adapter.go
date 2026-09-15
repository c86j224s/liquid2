package app

import uploadsource "github.com/c86j224s/liquid2/plasma/internal/source"

import (
	"context"
	"fmt"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/researchcatalog"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

func (s *Service) catalogService() *researchcatalog.Service {
	return researchcatalog.NewService(researchcatalog.Dependencies{
		EnsureReferenceTargetVisible: s.ensureResearchReferenceTargetVisible,
		ReadReferenceSummary: func(ctx context.Context, missionID, objectKind, objectID string, legacy bool) (researchcatalog.ObjectSummary, error) {
			summary, _, err := s.readObjectPayload(ctx, missionID, objectKind, objectID, legacy)
			return summary, err
		},
		GetMissionProjection: s.store.GetMissionProjection,
		ListSourceSnapshots: func(ctx context.Context, missionID string) ([]source.Snapshot, error) {
			return s.listSourceSnapshots(ctx, missionID)
		},
		ListVisibleRawArtifacts: func(ctx context.Context, missionID string, legacy bool) ([]artifactcontract.Raw, error) {
			return s.listVisibleRawArtifacts(ctx, missionID, legacy)
		},
		RawArtifactReadKind: uploadsource.UploadedArtifactReadKind,
		ListEvents:          s.store.ListLedgerEvents,
		ListVisibleLedgerEvents: func(ctx context.Context, missionID string, legacy bool) ([]ledger.Event, error) {
			return s.listVisibleLedgerEvents(ctx, missionID, legacy)
		},
		ReportArtifactIDsHiddenFromResearchDiscovery: s.reportArtifactIDsHiddenFromResearchDiscovery,
		FilterReportArtifactRefs:                     researchcatalog.FilterReportArtifactRefs,
		FilterReportArtifactSummaryRefs:              researchcatalog.FilterReportArtifactSummaryRefs,
		ListEvidenceRecords: func(ctx context.Context, missionID string) ([]researchrecords.EvidenceRecord, error) {
			store, ok := s.store.(ResearchRecordListStore)
			if !ok {
				return nil, fmt.Errorf("%w: research record list store is required", ErrInvalidInput)
			}
			return store.ListEvidenceRecords(ctx, missionID)
		},
		ListClaimRecords: func(ctx context.Context, missionID string) ([]researchrecords.ClaimRecord, error) {
			store, ok := s.store.(ResearchRecordListStore)
			if !ok {
				return nil, fmt.Errorf("%w: research record list store is required", ErrInvalidInput)
			}
			return store.ListClaimRecords(ctx, missionID)
		},
		ListQuestionRecords: func(ctx context.Context, missionID string) ([]researchrecords.QuestionRecord, error) {
			store, ok := s.store.(ResearchRecordListStore)
			if !ok {
				return nil, fmt.Errorf("%w: research record list store is required", ErrInvalidInput)
			}
			return store.ListQuestionRecords(ctx, missionID)
		},
		ListOptionRecords: func(ctx context.Context, missionID string) ([]researchrecords.OptionRecord, error) {
			store, ok := s.store.(ResearchRecordListStore)
			if !ok {
				return nil, fmt.Errorf("%w: research record list store is required", ErrInvalidInput)
			}
			return store.ListOptionRecords(ctx, missionID)
		},
		ListProposalSummaries: func(ctx context.Context, missionID string) ([]researchcatalog.ObjectSummary, error) {
			store, ok := s.store.(ResearchRecordListStore)
			if !ok {
				return nil, fmt.Errorf("%w: research record list store is required", ErrInvalidInput)
			}
			values, err := store.ListProposalBundles(ctx, missionID)
			if err != nil {
				return nil, err
			}
			out := make([]researchcatalog.ObjectSummary, 0, len(values))
			for _, value := range values {
				out = append(out, summarizeProposal(value))
			}
			return out, nil
		},
		ListReportSummaries: func(ctx context.Context, missionID string) ([]researchcatalog.ObjectSummary, error) {
			store, ok := s.store.(ReportListStore)
			if !ok {
				return nil, fmt.Errorf("%w: report list store is required", ErrInvalidInput)
			}
			values, err := store.ListReports(ctx, missionID)
			if err != nil {
				return nil, err
			}
			out := make([]researchcatalog.ObjectSummary, 0, len(values))
			for _, value := range values {
				out = append(out, summarizeReport(value))
			}
			return out, nil
		},
		ListReportVersionSummaries: func(ctx context.Context, missionID string) ([]researchcatalog.ObjectSummary, error) {
			store, ok := s.store.(ReportListStore)
			if !ok {
				return nil, fmt.Errorf("%w: report list store is required", ErrInvalidInput)
			}
			values, err := store.ListReportVersions(ctx, missionID)
			if err != nil {
				return nil, err
			}
			out := make([]researchcatalog.ObjectSummary, 0, len(values))
			for _, value := range values {
				out = append(out, summarizeReportVersion(value))
			}
			return out, nil
		},
		ListReportBlockSummaries: func(ctx context.Context, versionID string) ([]researchcatalog.ObjectSummary, error) {
			values, err := s.store.ListReportBlocks(ctx, versionID)
			if err != nil {
				return nil, err
			}
			out := make([]researchcatalog.ObjectSummary, 0, len(values))
			for _, value := range values {
				out = append(out, summarizeReportBlock(value))
			}
			return out, nil
		},
	})
}

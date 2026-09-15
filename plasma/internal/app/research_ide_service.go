package app

import uploadsource "github.com/c86j224s/liquid2/plasma/internal/source"

import "github.com/c86j224s/liquid2/plasma/internal/reporting/reportdocument"

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/researchcatalog"
	"github.com/c86j224s/liquid2/plasma/internal/researchinspection"
	"github.com/c86j224s/liquid2/plasma/internal/researchproposal"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"github.com/c86j224s/liquid2/plasma/internal/source"
	"strings"
	"unicode/utf8"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/pdfdocument"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidateevents"
)

const (
	researchIDEDefaultLimit   = 20
	researchIDEMaxLimit       = 100
	researchIDEMaxSuggestions = 6
)

// ResearchIDEReader는 Research IDE 화면이 필요한 읽기 기능만 노출하는 조회 포트다.
type ResearchIDEReader interface {
	OutlineMission(context.Context, string) (ResearchIDEOutline, error)
	ListMissionChanges(context.Context, ResearchIDEChangesRequest) (ResearchIDEChanges, error)
	ListMissionObjects(context.Context, string, string, int, string) (researchcatalog.Page, error)
	ReadMissionObject(context.Context, researchinspection.ReadRequest) (researchinspection.ObjectRead, error)
	GrepMissionObjects(context.Context, string, string, int, string) (researchinspection.GrepResult, error)
	ListObjectReferences(context.Context, string, string, string, int, string) (researchcatalog.References, error)
}

// RawArtifactListStore는 artifact 목록 조회를 화면 조립에서 분리하는 저장소 포트다.
type RawArtifactListStore interface {
	ListRawArtifacts(context.Context, string) ([]artifactcontract.Raw, error)
}

// OutlineMission는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) OutlineMission(ctx context.Context, missionID string) (ResearchIDEOutline, error) {
	return s.outlineMission(ctx, missionID, false)
}

// OutlineMissionLegacy는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) OutlineMissionLegacy(ctx context.Context, missionID string) (ResearchIDEOutline, error) {
	return s.outlineMission(ctx, missionID, true)
}

func (s *Service) outlineMission(ctx context.Context, missionID string, legacy bool) (ResearchIDEOutline, error) {
	outline, err := s.catalogService().Outline(ctx, missionID, legacy)
	if err != nil {
		return ResearchIDEOutline{}, err
	}
	return ResearchIDEOutline{MissionID: outline.MissionID, LastSequence: outline.LastSequence, Title: outline.Title, Objective: outline.Objective, Scope: outline.Scope, Counts: outline.Counts, ActiveReportVersionID: outline.ActiveReportVersionID, RecentLedgerEvents: outline.RecentLedgerEvents, NextSuggestedObjectRefs: outline.NextSuggestedObjectRefs}, nil
}

// ListMissionObjects는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListMissionObjects(ctx context.Context, missionID, objectKind string, limit int, cursor string) (researchcatalog.Page, error) {
	return s.listMissionObjects(ctx, missionID, objectKind, limit, cursor, false)
}

// ListMissionObjectsLegacy는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListMissionObjectsLegacy(ctx context.Context, missionID, objectKind string, limit int, cursor string) (researchcatalog.Page, error) {
	return s.listMissionObjects(ctx, missionID, objectKind, limit, cursor, true)
}

func (s *Service) listMissionObjects(ctx context.Context, missionID, objectKind string, limit int, cursor string, legacy bool) (researchcatalog.Page, error) {
	return s.catalogService().List(ctx, missionID, objectKind, limit, cursor, legacy)
}

// ReadMissionObject delegates bounded read sequencing to researchinspection while
// retaining app-owned materialization, observation, security, visibility, and report policy.
func (s *Service) ReadMissionObject(ctx context.Context, req researchinspection.ReadRequest) (researchinspection.ObjectRead, error) {
	return researchinspection.NewService(researchinspection.Dependencies{
		ReadChunked:           s.readChunkedMissionObject,
		ReadPayload:           s.readObjectPayload,
		ReportVersionChildren: s.reportVersionBlockPage,
	}).Read(ctx, req)
}

func (s *Service) readChunkedMissionObject(ctx context.Context, missionID string, objectKind string, objectID string, offset int, maxBytes int) (researchinspection.ObjectRead, bool, error) {
	switch objectKind {
	case researchcatalog.ObjectSourceSnapshot:
		record, err := s.store.GetSourceSnapshot(ctx, objectID)
		if err != nil {
			return researchinspection.ObjectRead{}, true, err
		}
		if record.MissionID != missionID {
			return researchinspection.ObjectRead{}, true, fmt.Errorf("%w: source_snapshot %s belongs to another mission", ErrInvalidInput, objectID)
		}
		record.State, _ = s.sourceState(ctx, missionID, record.SnapshotID)
		if record.State.Removed {
			return researchinspection.ObjectRead{}, true, fmt.Errorf("%w: source_snapshot %s is removed", ErrInvalidInput, objectID)
		}
		if record.Access.RetrievalPolicy != sourcecontract.RetrievalPolicyLiveReference || record.Connector.ConnectorType != sourcecontract.ConnectorTypeLocalPath {
			payload, handled, err := s.readSourceSnapshotPDFPayload(ctx, missionID, record, offset, maxBytes)
			if err != nil {
				return researchinspection.ObjectRead{}, true, err
			}
			if handled {
				return researchinspection.ObjectRead{
					ObjectKind: objectKind,
					ObjectID:   objectID,
					MissionID:  missionID,
					Summary:    payload.summary.Summary,
					Refs:       payload.summary.Refs,
					Data:       string(payload.data),
					Truncated:  payload.truncated,
					NextOffset: payload.nextOffset,
				}, true, nil
			}
			return researchinspection.ObjectRead{}, false, nil
		}
		locator, err := source.ParseLocalPathLocator(record.Locators)
		if err != nil {
			return researchinspection.ObjectRead{}, true, err
		}
		if locator.PathKind == "directory" {
			return researchinspection.ObjectRead{}, false, nil
		}
		result, err := s.ReadLocalPathSource(ctx, ReadLocalPathSourceRequest{
			MissionID:  missionID,
			SnapshotID: objectID,
			Offset:     int64(offset),
			MaxBytes:   int64(maxBytes),
			Producer:   ledger.Producer{Type: "research_ide", ID: "plasma"},
		})
		if err != nil {
			return researchinspection.ObjectRead{}, true, err
		}
		data := mustJSON(map[string]any{
			"snapshot":             result.Snapshot,
			"content":              result.Read.Content,
			"observation_metadata": result.Read.Metadata,
			"observation_event_id": observationEventID(result.ObservationEvent),
		})
		return researchinspection.ObjectRead{
			ObjectKind: objectKind,
			ObjectID:   objectID,
			MissionID:  missionID,
			Summary:    researchcatalog.SummarizeSourceSnapshot(result.Snapshot).Summary,
			Refs:       researchcatalog.SummarizeSourceSnapshot(result.Snapshot).Refs,
			Data:       string(data),
			Truncated:  result.Read.Metadata.Truncated,
			NextOffset: int(result.Read.Metadata.NextOffset),
		}, true, nil
	case researchcatalog.ObjectRawArtifact:
		record, err := s.store.GetRawArtifact(ctx, objectID)
		if err != nil {
			return researchinspection.ObjectRead{}, true, err
		}
		if record.MissionID != missionID {
			return researchinspection.ObjectRead{}, true, fmt.Errorf("%w: raw_artifact %s belongs to another mission", ErrInvalidInput, objectID)
		}
		summary := researchcatalog.SummarizeRawArtifact(record, uploadsource.UploadedArtifactReadKind(record))
		if uploadsource.UploadedArtifactReadKind(record) == "metadata" {
			metadata := uploadsource.UploadedArtifactMetadata(record)
			metadata["metadata_only"] = true
			metadata["note"] = "binary artifact content is not returned through research.read"
			return researchinspection.ObjectRead{ObjectKind: objectKind, ObjectID: objectID, MissionID: missionID, Summary: summary.Summary, Refs: summary.Refs, Data: string(mustJSON(metadata))}, true, nil
		}
		if !pdfdocument.IsPDFMediaType(record.MediaType) && !pdfdocument.IsPDFBytes(record.Content) {
			return researchinspection.ObjectRead{}, false, nil
		}
		chunk, err := pdfdocument.ExtractChunk(record.Content, offset, maxBytes)
		if err != nil {
			data := mustJSON(map[string]any{
				"artifact_id": record.ArtifactID,
				"mission_id":  record.MissionID,
				"media_type":  record.MediaType,
				"byte_size":   record.ByteSize,
				"sha256":      record.SHA256,
				"storage_uri": record.StorageURI,
				"filename":    record.Filename,
				"note":        "PDF text extraction failed",
				"error":       err.Error(),
			})
			return researchinspection.ObjectRead{ObjectKind: objectKind, ObjectID: objectID, MissionID: missionID, Summary: summary.Summary, Refs: summary.Refs, Data: string(data)}, true, nil
		}
		data := mustJSON(map[string]any{
			"artifact_id":          record.ArtifactID,
			"mission_id":           record.MissionID,
			"media_type":           record.MediaType,
			"byte_size":            record.ByteSize,
			"sha256":               record.SHA256,
			"storage_uri":          record.StorageURI,
			"filename":             record.Filename,
			"content":              chunk.Text,
			"content_offset":       chunk.Offset,
			"content_length":       chunk.ContentLength,
			"content_length_known": chunk.ContentLengthKnown,
			"content_truncated":    chunk.Truncated,
			"next_offset":          chunk.NextOffset,
			"extraction_type":      "pdf_text",
			"page_count":           chunk.PageCount,
			"suggested_read_bytes": pdfdocument.DefaultChunkMaxBytes,
			"max_read_bytes":       pdfdocument.MaxChunkBytes,
		})
		return researchinspection.ObjectRead{
			ObjectKind: objectKind,
			ObjectID:   objectID,
			MissionID:  missionID,
			Summary:    summary.Summary,
			Refs:       summary.Refs,
			Data:       string(data),
			Truncated:  chunk.Truncated,
			NextOffset: chunk.NextOffset,
		}, true, nil
	default:
		return researchinspection.ObjectRead{}, false, nil
	}
}

type researchIDEReadPayload struct {
	summary    researchcatalog.ObjectSummary
	data       []byte
	truncated  bool
	nextOffset int
}

func (s *Service) readSourceSnapshotPDFPayload(ctx context.Context, missionID string, snapshot sourcecontract.Snapshot, offset int, maxBytes int) (researchIDEReadPayload, bool, error) {
	if snapshot.Access.RetrievalPolicy == sourcecontract.RetrievalPolicyLiveReference {
		return researchIDEReadPayload{}, false, nil
	}
	if len(snapshot.ArtifactIDs) != 1 {
		return researchIDEReadPayload{}, false, nil
	}
	artifactID := strings.TrimSpace(snapshot.ArtifactIDs[0])
	if artifactID == "" {
		return researchIDEReadPayload{}, false, nil
	}
	artifact, err := s.store.GetRawArtifact(ctx, artifactID)
	if err != nil {
		return researchIDEReadPayload{}, true, err
	}
	if artifact.MissionID != missionID {
		return researchIDEReadPayload{}, true, fmt.Errorf("%w: source artifact %s belongs to another mission", ErrInvalidInput, artifactID)
	}
	if !pdfdocument.IsPDFMediaType(artifact.MediaType) && !pdfdocument.IsPDFBytes(artifact.Content) {
		return researchIDEReadPayload{}, false, nil
	}
	summary := researchcatalog.SummarizeSourceSnapshot(snapshot)
	artifactMetadata := uploadsource.UploadedArtifactMetadata(artifact)
	sourceMetadata := sourceSnapshotReadMetadata(snapshot, summary)
	chunk, err := pdfdocument.ExtractChunk(artifact.Content, offset, maxBytes)
	if err != nil {
		data := mustJSON(map[string]any{
			"source":    sourceMetadata,
			"artifact":  artifactMetadata,
			"note":      "PDF text extraction failed",
			"error":     err.Error(),
			"read_kind": "source_pdf_text",
		})
		return researchIDEReadPayload{summary: summary, data: data}, true, nil
	}
	data := mustJSON(map[string]any{
		"source":               sourceMetadata,
		"artifact":             artifactMetadata,
		"content":              chunk.Text,
		"content_offset":       chunk.Offset,
		"content_length":       chunk.ContentLength,
		"content_length_known": chunk.ContentLengthKnown,
		"content_truncated":    chunk.Truncated,
		"next_offset":          chunk.NextOffset,
		"extraction_type":      "pdf_text",
		"page_count":           chunk.PageCount,
		"suggested_read_bytes": pdfdocument.DefaultChunkMaxBytes,
		"max_read_bytes":       researchinspection.MaxBytes,
		"read_kind":            "source_pdf_text",
	})
	return researchIDEReadPayload{
		summary:    summary,
		data:       data,
		truncated:  chunk.Truncated,
		nextOffset: chunk.NextOffset,
	}, true, nil
}

func sourceSnapshotReadMetadata(snapshot sourcecontract.Snapshot, summary researchcatalog.ObjectSummary) map[string]any {
	metadata := map[string]any{
		"snapshot_id":      snapshot.SnapshotID,
		"mission_id":       snapshot.MissionID,
		"title":            strings.TrimSpace(snapshot.Title),
		"connector_type":   snapshot.Connector.ConnectorType,
		"retrieval_policy": snapshot.Access.RetrievalPolicy,
		"state":            firstNonEmpty(snapshot.State.State, sourcecontract.StateActive),
		"refs":             summary.Refs,
	}
	if externalURI := strings.TrimSpace(snapshot.Connector.ExternalURI); externalURI != "" {
		metadata["external_uri"] = externalURI
	}
	if len(summary.Metadata) > 0 {
		metadata["metadata"] = summary.Metadata
	}
	return metadata
}

// GrepMissionObjects delegates bounded grep sequencing to researchinspection while
// retaining app-owned materialization, observation, security, and visibility.
func (s *Service) GrepMissionObjects(ctx context.Context, missionID, query string, limit int, cursor string) (researchinspection.GrepResult, error) {
	return researchinspection.NewService(researchinspection.Dependencies{GrepCandidates: s.grepCandidates}).Grep(ctx, missionID, query, limit, cursor, false)
}

// GrepMissionObjectsLegacy delegates bounded legacy grep sequencing to researchinspection.
func (s *Service) GrepMissionObjectsLegacy(ctx context.Context, missionID, query string, limit int, cursor string) (researchinspection.GrepResult, error) {
	return researchinspection.NewService(researchinspection.Dependencies{GrepCandidates: s.grepCandidates}).Grep(ctx, missionID, query, limit, cursor, true)
}

// ListObjectReferences는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListObjectReferences(ctx context.Context, missionID, objectKind, objectID string, limit int, cursor string) (researchcatalog.References, error) {
	return s.listObjectReferences(ctx, missionID, objectKind, objectID, limit, cursor, false)
}

// ListObjectReferencesLegacy는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListObjectReferencesLegacy(ctx context.Context, missionID, objectKind, objectID string, limit int, cursor string) (researchcatalog.References, error) {
	return s.listObjectReferences(ctx, missionID, objectKind, objectID, limit, cursor, true)
}

func (s *Service) listObjectReferences(ctx context.Context, missionID, objectKind, objectID string, limit int, cursor string, legacy bool) (researchcatalog.References, error) {
	return s.catalogService().References(ctx, missionID, objectKind, objectID, limit, cursor, legacy)
}

// ensureResearchReferenceTargetVisible keeps references as a discovery surface:
// direct reads may still address hidden report objects, but non-legacy
// reference traversal must fail closed before returning their links.
func (s *Service) ensureResearchReferenceTargetVisible(ctx context.Context, missionID string, objectKind string, objectID string, legacy bool, reportArtifactIDs map[string]struct{}) error {
	if legacy {
		return nil
	}
	switch objectKind {
	case researchcatalog.ObjectRawArtifact:
		if _, ok := reportArtifactIDs[objectID]; ok {
			return fmt.Errorf("%w: raw_artifact %s is hidden from research references", ErrInvalidInput, objectID)
		}
	case researchcatalog.ObjectLedgerEvent:
		event, err := s.findLedgerEvent(ctx, missionID, objectID)
		if err != nil {
			return err
		}
		if researchcatalog.ReportLedgerEvent(event) {
			return fmt.Errorf("%w: ledger_event %s is hidden from research references", ErrInvalidInput, objectID)
		}
	}
	return nil
}

func (s *Service) listResearchObjects(ctx context.Context, missionID string) ([]researchrecords.EvidenceRecord, []researchrecords.ClaimRecord, []researchrecords.QuestionRecord, []researchrecords.OptionRecord, []researchproposal.ProposalBundle, error) {
	store, ok := s.store.(ResearchRecordListStore)
	if !ok {
		return nil, nil, nil, nil, nil, fmt.Errorf("%w: research record list store is required", ErrInvalidInput)
	}
	evidence, err := store.ListEvidenceRecords(ctx, missionID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	claims, err := store.ListClaimRecords(ctx, missionID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	questions, err := store.ListQuestionRecords(ctx, missionID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	options, err := store.ListOptionRecords(ctx, missionID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	proposals, err := store.ListProposalBundles(ctx, missionID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return evidence, claims, questions, options, proposals, nil
}

func (s *Service) listSourceSnapshots(ctx context.Context, missionID string) ([]sourcecontract.Snapshot, error) {
	return s.ListSourceSnapshotsWithState(ctx, sourcecontract.ListRequest{MissionID: missionID})
}

func (s *Service) listRawArtifacts(ctx context.Context, missionID string) ([]artifactcontract.Raw, error) {
	store, ok := s.store.(RawArtifactListStore)
	if !ok {
		return nil, fmt.Errorf("%w: raw artifact list store is required", ErrInvalidInput)
	}
	return store.ListRawArtifacts(ctx, missionID)
}

// ListRawArtifacts는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListRawArtifacts(ctx context.Context, missionID string) ([]artifactcontract.Raw, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	return s.listRawArtifacts(ctx, missionID)
}

func (s *Service) listVisibleRawArtifacts(ctx context.Context, missionID string, legacy bool) ([]artifactcontract.Raw, error) {
	artifacts, err := s.listRawArtifacts(ctx, missionID)
	if err != nil {
		return nil, err
	}
	reportArtifacts, err := s.reportArtifactIDsHiddenFromResearchDiscovery(ctx, missionID, legacy)
	if err != nil {
		return nil, err
	}
	rejected, err := s.rejectedReportPatchArtifactIDs(ctx, missionID)
	if err != nil {
		return nil, err
	}
	stagedCandidates, err := s.stagedSourceCandidateArtifactIDs(ctx, missionID)
	if err != nil {
		return nil, err
	}
	if len(reportArtifacts) == 0 && len(rejected) == 0 && len(stagedCandidates) == 0 {
		return artifacts, nil
	}
	visible := artifacts[:0]
	for _, artifact := range artifacts {
		if _, ok := reportArtifacts[artifact.ArtifactID]; ok {
			continue
		}
		if _, ok := rejected[artifact.ArtifactID]; ok {
			continue
		}
		if _, ok := stagedCandidates[artifact.ArtifactID]; ok {
			continue
		}
		visible = append(visible, artifact)
	}
	return visible, nil
}

func (s *Service) listVisibleLedgerEvents(ctx context.Context, missionID string, legacy bool) ([]ledger.Event, error) {
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return nil, err
	}
	return researchcatalog.VisibleLedgerEvents(events, legacy), nil
}

func (s *Service) reportArtifactIDsHiddenFromResearchDiscovery(ctx context.Context, missionID string, legacy bool) (map[string]struct{}, error) {
	if legacy {
		return nil, nil
	}
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return nil, err
	}
	return researchcatalog.ReportArtifactIDs(events), nil
}

func (s *Service) isRejectedReportPatchArtifact(ctx context.Context, missionID string, artifactID string) (bool, error) {
	rejected, err := s.rejectedReportPatchArtifactIDs(ctx, missionID)
	if err != nil {
		return false, err
	}
	_, ok := rejected[strings.TrimSpace(artifactID)]
	return ok, nil
}

func (s *Service) rejectedReportPatchArtifactIDs(ctx context.Context, missionID string) (map[string]struct{}, error) {
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return nil, err
	}
	rejected := map[string]struct{}{}
	for _, event := range events {
		if event.EventType != "report.patch.rejected" {
			continue
		}
		var payload struct {
			ArtifactID string `json:"artifact_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if artifactID := strings.TrimSpace(payload.ArtifactID); artifactID != "" {
			rejected[artifactID] = struct{}{}
		}
	}
	return rejected, nil
}

func (s *Service) isStagedSourceCandidateArtifact(ctx context.Context, missionID string, artifactID string) (bool, error) {
	staged, err := s.stagedSourceCandidateArtifactIDs(ctx, missionID)
	if err != nil {
		return false, err
	}
	_, ok := staged[strings.TrimSpace(artifactID)]
	return ok, nil
}

func (s *Service) stagedSourceCandidateArtifactIDs(ctx context.Context, missionID string) (map[string]struct{}, error) {
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return nil, err
	}
	snapshots, err := s.listSourceSnapshots(ctx, missionID)
	if err != nil {
		return nil, err
	}
	return sourcecandidateevents.OpenStagedArtifactIDs(sourceCandidateEventsFromApp(events), sourceCandidateSnapshotsFromApp(snapshots)), nil
}

func (s *Service) listReportObjects(ctx context.Context, missionID string) ([]reportdocument.Report, []reportdocument.ReportVersion, error) {
	store, ok := s.store.(ReportListStore)
	if !ok {
		return nil, nil, fmt.Errorf("%w: report list store is required", ErrInvalidInput)
	}
	reports, err := store.ListReports(ctx, missionID)
	if err != nil {
		return nil, nil, err
	}
	versions, err := store.ListReportVersions(ctx, missionID)
	if err != nil {
		return nil, nil, err
	}
	return reports, versions, nil
}

func (s *Service) allObjectSummaries(ctx context.Context, missionID, objectKind string, legacy bool) ([]researchcatalog.ObjectSummary, error) {
	return s.catalogService().AllObjectSummaries(ctx, missionID, objectKind, legacy)
}

func (s *Service) readObjectPayload(ctx context.Context, missionID, objectKind, objectID string, legacy bool) (researchcatalog.ObjectSummary, []byte, error) {
	if !researchcatalog.ObjectKindAllowed(objectKind, legacy) {
		return researchcatalog.ObjectSummary{}, nil, fmt.Errorf("%w: unsupported object kind", ErrInvalidInput)
	}
	switch objectKind {
	case researchcatalog.ObjectSourceSnapshot:
		record, err := s.store.GetSourceSnapshot(ctx, objectID)
		if err != nil {
			return researchcatalog.ObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return researchcatalog.ObjectSummary{}, nil, fmt.Errorf("%w: source_snapshot %s belongs to another mission", ErrInvalidInput, objectID)
		}
		record.State, _ = s.sourceState(ctx, missionID, record.SnapshotID)
		if record.State.Removed {
			return researchcatalog.ObjectSummary{}, nil, fmt.Errorf("%w: source_snapshot %s is removed", ErrInvalidInput, objectID)
		}
		if record.Access.RetrievalPolicy == sourcecontract.RetrievalPolicyLiveReference && record.Connector.ConnectorType == sourcecontract.ConnectorTypeLocalPath {
			locator, err := source.ParseLocalPathLocator(record.Locators)
			if err != nil {
				return researchcatalog.ObjectSummary{}, nil, err
			}
			if locator.PathKind == "directory" {
				tree, err := s.TreeLocalPathSource(ctx, TreeLocalPathSourceRequest{
					MissionID:  missionID,
					SnapshotID: objectID,
					Depth:      1,
					Limit:      researchIDEDefaultLimit,
					Producer:   ledger.Producer{Type: "research_ide", ID: "plasma"},
				})
				if err != nil {
					return researchcatalog.ObjectSummary{}, nil, err
				}
				return researchcatalog.SummarizeSourceSnapshot(tree.Snapshot), mustJSON(map[string]any{
					"snapshot":             tree.Snapshot,
					"tree":                 tree.Tree,
					"observation_metadata": tree.Tree.Metadata,
					"observation_event_id": observationEventID(tree.ObservationEvent),
				}), nil
			}
			read, err := s.ReadLocalPathSource(ctx, ReadLocalPathSourceRequest{
				MissionID:  missionID,
				SnapshotID: objectID,
				MaxBytes:   int64(researchinspection.DefaultBytes),
				Producer:   ledger.Producer{Type: "research_ide", ID: "plasma"},
			})
			if err != nil {
				return researchcatalog.ObjectSummary{}, nil, err
			}
			return researchcatalog.SummarizeSourceSnapshot(read.Snapshot), mustJSON(map[string]any{
				"snapshot":             read.Snapshot,
				"content":              read.Read.Content,
				"observation_metadata": read.Read.Metadata,
				"observation_event_id": observationEventID(read.ObservationEvent),
			}), nil
		}
		if payload, handled, err := s.readSourceSnapshotPDFPayload(ctx, missionID, record, 0, researchinspection.DefaultBytes); handled || err != nil {
			return payload.summary, payload.data, err
		}
		return researchcatalog.SummarizeSourceSnapshot(record), mustJSON(record), nil
	case researchcatalog.ObjectRawArtifact:
		record, err := s.store.GetRawArtifact(ctx, objectID)
		if err != nil {
			return researchcatalog.ObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return researchcatalog.ObjectSummary{}, nil, fmt.Errorf("%w: raw_artifact %s belongs to another mission", ErrInvalidInput, objectID)
		}
		rejected, err := s.isRejectedReportPatchArtifact(ctx, missionID, record.ArtifactID)
		if err != nil {
			return researchcatalog.ObjectSummary{}, nil, err
		}
		if rejected {
			return researchcatalog.ObjectSummary{}, nil, fmt.Errorf("%w: raw_artifact %s is a rejected report patch artifact", ErrInvalidInput, objectID)
		}
		stagedCandidate, err := s.isStagedSourceCandidateArtifact(ctx, missionID, record.ArtifactID)
		if err != nil {
			return researchcatalog.ObjectSummary{}, nil, err
		}
		if stagedCandidate {
			return researchcatalog.ObjectSummary{}, nil, fmt.Errorf("%w: raw_artifact %s is an unapproved source candidate artifact; use plasma.sources.candidates.read", ErrInvalidInput, objectID)
		}
		if uploadsource.UploadedArtifactReadKind(record) == "metadata" {
			metadata := uploadsource.UploadedArtifactMetadata(record)
			metadata["metadata_only"] = true
			metadata["note"] = "binary artifact content is not returned through research.read"
			return researchcatalog.SummarizeRawArtifact(record, uploadsource.UploadedArtifactReadKind(record)), mustJSON(metadata), nil
		}
		if pdfdocument.IsPDFMediaType(record.MediaType) || pdfdocument.IsPDFBytes(record.Content) {
			chunk, err := pdfdocument.ExtractChunk(record.Content, 0, researchinspection.DefaultBytes)
			if err != nil {
				return researchcatalog.SummarizeRawArtifact(record, uploadsource.UploadedArtifactReadKind(record)), mustJSON(map[string]any{
					"artifact_id": record.ArtifactID,
					"mission_id":  record.MissionID,
					"media_type":  record.MediaType,
					"byte_size":   record.ByteSize,
					"sha256":      record.SHA256,
					"storage_uri": record.StorageURI,
					"filename":    record.Filename,
					"note":        "PDF text extraction failed",
					"error":       err.Error(),
				}), nil
			}
			return researchcatalog.SummarizeRawArtifact(record, uploadsource.UploadedArtifactReadKind(record)), mustJSON(map[string]any{
				"artifact_id":          record.ArtifactID,
				"mission_id":           record.MissionID,
				"media_type":           record.MediaType,
				"byte_size":            record.ByteSize,
				"sha256":               record.SHA256,
				"storage_uri":          record.StorageURI,
				"filename":             record.Filename,
				"content":              chunk.Text,
				"content_length":       chunk.ContentLength,
				"content_length_known": chunk.ContentLengthKnown,
				"truncated":            chunk.Truncated,
				"next_offset":          chunk.NextOffset,
				"extraction_type":      "pdf_text",
				"page_count":           chunk.PageCount,
				"suggested_read_bytes": pdfdocument.DefaultChunkMaxBytes,
				"max_read_bytes":       pdfdocument.MaxChunkBytes,
			}), nil
		}
		if !utf8.Valid(record.Content) {
			return researchcatalog.SummarizeRawArtifact(record, uploadsource.UploadedArtifactReadKind(record)), mustJSON(map[string]any{
				"artifact_id": record.ArtifactID,
				"mission_id":  record.MissionID,
				"media_type":  record.MediaType,
				"byte_size":   record.ByteSize,
				"sha256":      record.SHA256,
				"storage_uri": record.StorageURI,
				"filename":    record.Filename,
				"note":        "binary artifact content is not returned through research.read",
			}), nil
		}
		return researchcatalog.SummarizeRawArtifact(record, uploadsource.UploadedArtifactReadKind(record)), record.Content, nil
	case researchcatalog.ObjectEvidenceRecord:
		record, err := s.store.GetEvidenceRecord(ctx, objectID)
		if err != nil {
			return researchcatalog.ObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return researchcatalog.ObjectSummary{}, nil, fmt.Errorf("%w: evidence_record %s belongs to another mission", ErrInvalidInput, objectID)
		}
		return researchcatalog.SummarizeEvidence(record), mustJSON(record), nil
	case researchcatalog.ObjectClaimRecord:
		record, err := s.store.GetClaimRecord(ctx, objectID)
		if err != nil {
			return researchcatalog.ObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return researchcatalog.ObjectSummary{}, nil, fmt.Errorf("%w: claim_record %s belongs to another mission", ErrInvalidInput, objectID)
		}
		return researchcatalog.SummarizeClaim(record), mustJSON(record), nil
	case researchcatalog.ObjectQuestionRecord:
		record, err := s.store.GetQuestionRecord(ctx, objectID)
		if err != nil {
			return researchcatalog.ObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return researchcatalog.ObjectSummary{}, nil, fmt.Errorf("%w: question_record %s belongs to another mission", ErrInvalidInput, objectID)
		}
		return researchcatalog.SummarizeQuestion(record), mustJSON(record), nil
	case researchcatalog.ObjectOptionRecord:
		record, err := s.store.GetOptionRecord(ctx, objectID)
		if err != nil {
			return researchcatalog.ObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return researchcatalog.ObjectSummary{}, nil, fmt.Errorf("%w: option_record %s belongs to another mission", ErrInvalidInput, objectID)
		}
		return researchcatalog.SummarizeOption(record), mustJSON(record), nil
	case researchcatalog.ObjectProposalBundle:
		record, err := s.store.GetProposalBundle(ctx, objectID)
		if err != nil {
			return researchcatalog.ObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return researchcatalog.ObjectSummary{}, nil, fmt.Errorf("%w: proposal_bundle %s belongs to another mission", ErrInvalidInput, objectID)
		}
		return summarizeProposal(record), mustJSON(record), nil
	case researchcatalog.ObjectReport:
		record, err := s.store.GetReport(ctx, objectID)
		if err != nil {
			return researchcatalog.ObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return researchcatalog.ObjectSummary{}, nil, fmt.Errorf("%w: report %s belongs to another mission", ErrInvalidInput, objectID)
		}
		return summarizeReport(record), mustJSON(record), nil
	case researchcatalog.ObjectReportVersion:
		record, err := s.store.GetReportVersion(ctx, objectID)
		if err != nil {
			return researchcatalog.ObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return researchcatalog.ObjectSummary{}, nil, fmt.Errorf("%w: report_version %s belongs to another mission", ErrInvalidInput, objectID)
		}
		return summarizeReportVersion(record), mustJSON(record), nil
	case researchcatalog.ObjectReportBlock:
		block, err := s.findReportBlock(ctx, missionID, objectID)
		if err != nil {
			return researchcatalog.ObjectSummary{}, nil, err
		}
		return summarizeReportBlock(block), mustJSON(block), nil
	case researchcatalog.ObjectLedgerEvent:
		event, err := s.findLedgerEvent(ctx, missionID, objectID)
		if err != nil {
			return researchcatalog.ObjectSummary{}, nil, err
		}
		return researchcatalog.SummarizeLedgerEvent(event), mustJSON(event), nil
	default:
		return researchcatalog.ObjectSummary{}, nil, fmt.Errorf("%w: unsupported object kind", ErrInvalidInput)
	}
}

func (s *Service) grepCandidates(ctx context.Context, missionID string, query string, legacy bool) ([]researchinspection.GrepCandidate, error) {
	items, err := s.allObjectSummaries(ctx, missionID, "", legacy)
	if err != nil {
		return nil, err
	}
	var candidates []researchinspection.GrepCandidate
	for _, item := range items {
		if item.ObjectKind == researchcatalog.ObjectSourceSnapshot {
			source, err := s.GetSourceSnapshot(ctx, item.ObjectID)
			if err == nil {
				source.State, _ = s.sourceState(ctx, missionID, source.SnapshotID)
			}
			if err == nil && source.State.Removed {
				continue
			}
			if err == nil && source.Access.RetrievalPolicy == sourcecontract.RetrievalPolicyLiveReference && source.Connector.ConnectorType == sourcecontract.ConnectorTypeLocalPath {
				grep, err := s.GrepLocalPathSource(ctx, GrepLocalPathSourceRequest{
					MissionID:  missionID,
					SnapshotID: item.ObjectID,
					Query:      query,
					Producer:   ledger.Producer{Type: "research_ide", ID: "plasma"},
				})
				if err == nil {
					for _, match := range grep.Grep.Matches {
						candidates = append(candidates, researchinspection.GrepCandidate{Summary: item, Text: match.Snippet})
					}
					continue
				}
			}
		}
		_, data, err := s.readObjectPayload(ctx, missionID, item.ObjectKind, item.ObjectID, legacy)
		if err != nil {
			return nil, err
		}
		text := item.Summary + "\n" + string(data)
		candidates = append(candidates, researchinspection.GrepCandidate{Summary: item, Text: text})
	}
	return candidates, nil
}

func (s *Service) findReportBlock(ctx context.Context, missionID, blockID string) (reportdocument.ReportBlock, error) {
	_, versions, err := s.listReportObjects(ctx, missionID)
	if err != nil {
		return reportdocument.ReportBlock{}, err
	}
	for _, version := range versions {
		blocks, err := s.store.ListReportBlocks(ctx, version.ReportVersionID)
		if err != nil {
			return reportdocument.ReportBlock{}, err
		}
		for _, block := range blocks {
			if block.BlockID == blockID {
				if block.MissionID != missionID {
					return reportdocument.ReportBlock{}, fmt.Errorf("%w: report_block %s belongs to another mission", ErrInvalidInput, blockID)
				}
				return block, nil
			}
		}
	}
	return reportdocument.ReportBlock{}, fmt.Errorf("%w: report_block %s not found", ErrInvalidInput, blockID)
}

func (s *Service) reportVersionBlockPage(ctx context.Context, missionID, versionID string, limit int, cursor string) (researchcatalog.Page, error) {
	limit = researchcatalog.ClampLimit(limit)
	offset, err := researchcatalog.ParseCursor(cursor)
	if err != nil {
		return researchcatalog.Page{}, err
	}
	version, err := s.store.GetReportVersion(ctx, versionID)
	if err != nil {
		return researchcatalog.Page{}, err
	}
	if version.MissionID != missionID {
		return researchcatalog.Page{}, fmt.Errorf("%w: report_version %s belongs to another mission", ErrInvalidInput, versionID)
	}
	blocks, err := s.store.ListReportBlocks(ctx, versionID)
	if err != nil {
		return researchcatalog.Page{}, err
	}
	summaries := make([]researchcatalog.ObjectSummary, 0, len(blocks))
	for _, block := range blocks {
		if block.MissionID != missionID {
			return researchcatalog.Page{}, fmt.Errorf("%w: report_block %s belongs to another mission", ErrInvalidInput, block.BlockID)
		}
		summaries = append(summaries, summarizeReportBlock(block))
	}
	pageItems, next, truncated := researchcatalog.PaginateSummaries(summaries, offset, limit)
	return researchcatalog.Page{
		MissionID:  missionID,
		ObjectKind: researchcatalog.ObjectReportBlock,
		Items:      pageItems,
		NextCursor: next,
		Limit:      limit,
		Truncated:  truncated,
	}, nil
}

func (s *Service) findLedgerEvent(ctx context.Context, missionID, eventID string) (ledger.Event, error) {
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return ledger.Event{}, err
	}
	for _, event := range events {
		if event.EventID == eventID {
			return event, nil
		}
	}
	return ledger.Event{}, fmt.Errorf("%w: ledger_event %s not found", ErrInvalidInput, eventID)
}

func summarizeProposal(record researchproposal.ProposalBundle) researchcatalog.ObjectSummary {
	var refs []researchcatalog.ObjectRef
	for _, ref := range record.ObjectRefs {
		refs = append(refs, researchcatalog.ObjectRef{ObjectKind: researchcatalog.NormalizeObjectKind(ref.ObjectKind), ObjectID: ref.ObjectID})
	}
	return researchcatalog.ObjectSummary{ObjectKind: researchcatalog.ObjectProposalBundle, ObjectID: record.ProposalID, MissionID: record.MissionID, Summary: firstNonEmpty(record.Title, record.RequestedDecision, record.ProposalID), Refs: researchcatalog.DedupeRefs(refs), Metadata: map[string]any{"state": record.State}}
}

func summarizeReport(record reportdocument.Report) researchcatalog.ObjectSummary {
	var refs []researchcatalog.ObjectRef
	if record.ActiveVersionID != "" {
		refs = append(refs, researchcatalog.ObjectRef{ObjectKind: researchcatalog.ObjectReportVersion, ObjectID: record.ActiveVersionID})
	}
	return researchcatalog.ObjectSummary{ObjectKind: researchcatalog.ObjectReport, ObjectID: record.ReportID, MissionID: record.MissionID, Summary: record.Title, Refs: refs, Metadata: map[string]any{"state": record.State}}
}

func summarizeReportVersion(record reportdocument.ReportVersion) researchcatalog.ObjectSummary {
	refs := []researchcatalog.ObjectRef{{ObjectKind: researchcatalog.ObjectReport, ObjectID: record.ReportID}}
	for _, id := range record.IncludedEvidenceScope.ClaimIDs {
		refs = append(refs, researchcatalog.ObjectRef{ObjectKind: researchcatalog.ObjectClaimRecord, ObjectID: id})
	}
	for _, id := range record.IncludedEvidenceScope.EvidenceIDs {
		refs = append(refs, researchcatalog.ObjectRef{ObjectKind: researchcatalog.ObjectEvidenceRecord, ObjectID: id})
	}
	for _, id := range record.IncludedEvidenceScope.QuestionIDs {
		refs = append(refs, researchcatalog.ObjectRef{ObjectKind: researchcatalog.ObjectQuestionRecord, ObjectID: id})
	}
	return researchcatalog.ObjectSummary{ObjectKind: researchcatalog.ObjectReportVersion, ObjectID: record.ReportVersionID, MissionID: record.MissionID, Summary: record.State + " report version", Refs: researchcatalog.DedupeRefs(refs), Metadata: map[string]any{"report_id": record.ReportID, "state": record.State}}
}

func summarizeReportBlock(block reportdocument.ReportBlock) researchcatalog.ObjectSummary {
	var refs []researchcatalog.ObjectRef
	refs = append(refs, researchcatalog.ObjectRef{ObjectKind: researchcatalog.ObjectReportVersion, ObjectID: block.ReportVersionID})
	for _, id := range block.SourceRefs.ClaimIDs {
		refs = append(refs, researchcatalog.ObjectRef{ObjectKind: researchcatalog.ObjectClaimRecord, ObjectID: id})
	}
	for _, id := range block.SourceRefs.EvidenceIDs {
		refs = append(refs, researchcatalog.ObjectRef{ObjectKind: researchcatalog.ObjectEvidenceRecord, ObjectID: id})
	}
	for _, id := range block.SourceRefs.SnapshotIDs {
		refs = append(refs, researchcatalog.ObjectRef{ObjectKind: researchcatalog.ObjectSourceSnapshot, ObjectID: id})
	}
	for _, id := range block.SourceRefs.QuestionIDs {
		refs = append(refs, researchcatalog.ObjectRef{ObjectKind: researchcatalog.ObjectQuestionRecord, ObjectID: id})
	}
	return researchcatalog.ObjectSummary{ObjectKind: researchcatalog.ObjectReportBlock, ObjectID: block.BlockID, MissionID: block.MissionID, Summary: block.BlockType, Refs: researchcatalog.DedupeRefs(refs), Metadata: map[string]any{"report_version_id": block.ReportVersionID, "block_type": block.BlockType}}
}

func mustJSON(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		return []byte(`null`)
	}
	return encoded
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isResearchRecordKind(kind string) bool {
	switch kind {
	case researchcatalog.ObjectEvidenceRecord, researchcatalog.ObjectClaimRecord, researchcatalog.ObjectQuestionRecord, researchcatalog.ObjectOptionRecord, researchcatalog.ObjectProposalBundle:
		return true
	default:
		return false
	}
}

func filterSummaries(items []researchcatalog.ObjectSummary, kind string) []researchcatalog.ObjectSummary {
	var filtered []researchcatalog.ObjectSummary
	for _, item := range items {
		if item.ObjectKind == kind {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

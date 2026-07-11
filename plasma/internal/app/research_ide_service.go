package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidateevents"
	"github.com/c86j224s/liquid2/plasma/internal/sources/pdftext"
)

const (
	researchIDEDefaultLimit   = 20
	researchIDEMaxLimit       = 100
	researchIDEDefaultBytes   = 4096
	researchIDEMaxBytes       = 32768
	researchIDESnippetContext = 48
	researchIDEMaxSuggestions = 6
)

type ResearchIDEReader interface {
	OutlineMission(context.Context, string) (ResearchIDEOutline, error)
	ListMissionObjects(context.Context, string, string, int, string) (ResearchIDEPage, error)
	ReadMissionObject(context.Context, ResearchIDEReadRequest) (ResearchIDEObjectRead, error)
	GrepMissionObjects(context.Context, string, string, int, string) (ResearchIDEGrepResult, error)
	ListObjectReferences(context.Context, string, string, string, int, string) (ResearchIDEReferences, error)
}

type RawArtifactListStore interface {
	ListRawArtifacts(context.Context, string) ([]RawArtifact, error)
}

func (s *Service) OutlineMission(ctx context.Context, missionID string) (ResearchIDEOutline, error) {
	return s.outlineMission(ctx, missionID, false)
}

func (s *Service) OutlineMissionLegacy(ctx context.Context, missionID string) (ResearchIDEOutline, error) {
	return s.outlineMission(ctx, missionID, true)
}

func (s *Service) outlineMission(ctx context.Context, missionID string, legacy bool) (ResearchIDEOutline, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateID("mis_", missionID); err != nil {
		return ResearchIDEOutline{}, err
	}
	projection, err := s.store.GetMissionProjection(ctx, missionID)
	if err != nil {
		return ResearchIDEOutline{}, err
	}
	snapshots, err := s.listSourceSnapshots(ctx, missionID)
	if err != nil {
		return ResearchIDEOutline{}, err
	}
	artifacts, err := s.listVisibleRawArtifacts(ctx, missionID)
	if err != nil {
		return ResearchIDEOutline{}, err
	}
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return ResearchIDEOutline{}, err
	}
	counts := map[string]int{
		ResearchIDEObjectSourceSnapshot: len(snapshots),
		ResearchIDEObjectRawArtifact:    len(artifacts),
		ResearchIDEObjectLedgerEvent:    len(events),
	}
	if legacy {
		evidence, claims, questions, options, proposals, err := s.listResearchObjects(ctx, missionID)
		if err != nil {
			return ResearchIDEOutline{}, err
		}
		reports, versions, err := s.listReportObjects(ctx, missionID)
		if err != nil {
			return ResearchIDEOutline{}, err
		}
		counts[ResearchIDEObjectEvidenceRecord] = len(evidence)
		counts[ResearchIDEObjectClaimRecord] = len(claims)
		counts[ResearchIDEObjectQuestionRecord] = len(questions)
		counts[ResearchIDEObjectOptionRecord] = len(options)
		counts[ResearchIDEObjectProposalBundle] = len(proposals)
		counts[ResearchIDEObjectReport] = len(reports)
		counts[ResearchIDEObjectReportVersion] = len(versions)
		for _, record := range evidence {
			counts["evidence_record."+record.State]++
		}
		for _, record := range claims {
			counts["claim_record."+record.State]++
		}
		for _, record := range questions {
			counts["question_record."+record.State]++
		}
	}
	recent := make([]ResearchIDEObjectSummary, 0, 5)
	for i := len(events) - 1; i >= 0 && len(recent) < 5; i-- {
		if events[i].EventType == "mcp.tool.called" {
			continue
		}
		recent = append(recent, summarizeLedgerEvent(events[i]))
	}
	next := make([]ResearchIDEObjectRef, 0, researchIDEMaxSuggestions)
	appendNext := func(ref ResearchIDEObjectRef) {
		if len(next) < researchIDEMaxSuggestions {
			next = append(next, ref)
		}
	}
	if legacy {
		for _, id := range projection.OpenQuestionIDs {
			appendNext(ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectQuestionRecord, ObjectID: id})
		}
		if projection.ActiveReportVersionID != "" {
			appendNext(ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectReportVersion, ObjectID: projection.ActiveReportVersionID})
		}
	}
	for _, snapshot := range snapshots {
		if len(next) >= researchIDEMaxSuggestions {
			break
		}
		appendNext(ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectSourceSnapshot, ObjectID: snapshot.SnapshotID})
	}
	activeReportVersionID := ""
	if legacy {
		activeReportVersionID = projection.ActiveReportVersionID
	}
	return ResearchIDEOutline{
		MissionID:               missionID,
		Title:                   projection.Title,
		Objective:               projection.Objective,
		Scope:                   projection.Scope,
		Counts:                  counts,
		ActiveReportVersionID:   activeReportVersionID,
		RecentLedgerEvents:      recent,
		NextSuggestedObjectRefs: next,
	}, nil
}

func (s *Service) ListMissionObjects(ctx context.Context, missionID, objectKind string, limit int, cursor string) (ResearchIDEPage, error) {
	return s.listMissionObjects(ctx, missionID, objectKind, limit, cursor, false)
}

func (s *Service) ListMissionObjectsLegacy(ctx context.Context, missionID, objectKind string, limit int, cursor string) (ResearchIDEPage, error) {
	return s.listMissionObjects(ctx, missionID, objectKind, limit, cursor, true)
}

func (s *Service) listMissionObjects(ctx context.Context, missionID, objectKind string, limit int, cursor string, legacy bool) (ResearchIDEPage, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateID("mis_", missionID); err != nil {
		return ResearchIDEPage{}, err
	}
	objectKind = normalizeResearchIDEObjectKind(objectKind)
	limit = clampResearchIDELimit(limit)
	offset, err := parseResearchIDECursor(cursor)
	if err != nil {
		return ResearchIDEPage{}, err
	}
	items, err := s.allObjectSummaries(ctx, missionID, objectKind, legacy)
	if err != nil {
		return ResearchIDEPage{}, err
	}
	pageItems, next, truncated := paginateSummaries(items, offset, limit)
	return ResearchIDEPage{
		MissionID:  missionID,
		ObjectKind: objectKind,
		Items:      pageItems,
		NextCursor: next,
		Limit:      limit,
		Truncated:  truncated,
	}, nil
}

func (s *Service) ReadMissionObject(ctx context.Context, req ResearchIDEReadRequest) (ResearchIDEObjectRead, error) {
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return ResearchIDEObjectRead{}, err
	}
	objectKind := normalizeResearchIDEObjectKind(req.ObjectKind)
	objectID := strings.TrimSpace(req.ObjectID)
	if objectID == "" {
		return ResearchIDEObjectRead{}, fmt.Errorf("%w: object id is required", ErrInvalidInput)
	}
	maxBytes := clampResearchIDEBytes(req.MaxBytes)
	offset := req.Offset
	if offset < 0 {
		return ResearchIDEObjectRead{}, fmt.Errorf("%w: offset must be non-negative", ErrInvalidInput)
	}
	if read, handled, err := s.readChunkedMissionObject(ctx, missionID, objectKind, objectID, offset, maxBytes); handled || err != nil {
		return read, err
	}
	summary, data, err := s.readObjectPayload(ctx, missionID, objectKind, objectID, req.Legacy)
	if err != nil {
		return ResearchIDEObjectRead{}, err
	}
	chunk, truncated, nextOffset, err := chunkBytes(data, offset, maxBytes)
	if err != nil {
		return ResearchIDEObjectRead{}, err
	}
	read := ResearchIDEObjectRead{
		ObjectKind: objectKind,
		ObjectID:   objectID,
		MissionID:  missionID,
		Summary:    summary.Summary,
		Refs:       summary.Refs,
		Data:       string(chunk),
		Truncated:  truncated,
		NextOffset: nextOffset,
	}
	if objectKind == ResearchIDEObjectReportVersion {
		children, err := s.reportVersionBlockPage(ctx, missionID, objectID, req.Limit, req.Cursor)
		if err != nil {
			return ResearchIDEObjectRead{}, err
		}
		read.Children = &children
	}
	return read, nil
}

func (s *Service) readChunkedMissionObject(ctx context.Context, missionID string, objectKind string, objectID string, offset int, maxBytes int) (ResearchIDEObjectRead, bool, error) {
	switch objectKind {
	case ResearchIDEObjectSourceSnapshot:
		record, err := s.store.GetSourceSnapshot(ctx, objectID)
		if err != nil {
			return ResearchIDEObjectRead{}, true, err
		}
		if record.MissionID != missionID {
			return ResearchIDEObjectRead{}, true, fmt.Errorf("%w: source_snapshot %s belongs to another mission", ErrInvalidInput, objectID)
		}
		record.State, _ = s.sourceState(ctx, missionID, record.SnapshotID)
		if record.State.Removed {
			return ResearchIDEObjectRead{}, true, fmt.Errorf("%w: source_snapshot %s is removed", ErrInvalidInput, objectID)
		}
		if record.Access.RetrievalPolicy != SourceRetrievalPolicyLiveReference || record.Connector.ConnectorType != SourceConnectorTypeLocalPath {
			payload, handled, err := s.readSourceSnapshotPDFPayload(ctx, missionID, record, offset, maxBytes)
			if err != nil {
				return ResearchIDEObjectRead{}, true, err
			}
			if handled {
				return ResearchIDEObjectRead{
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
			return ResearchIDEObjectRead{}, false, nil
		}
		locator, err := parseLocalPathLocator(record.Locators)
		if err != nil {
			return ResearchIDEObjectRead{}, true, err
		}
		if locator.PathKind == "directory" {
			return ResearchIDEObjectRead{}, false, nil
		}
		result, err := s.ReadLocalPathSource(ctx, ReadLocalPathSourceRequest{
			MissionID:  missionID,
			SnapshotID: objectID,
			Offset:     int64(offset),
			MaxBytes:   int64(maxBytes),
			Producer:   Producer{Type: "research_ide", ID: "plasma"},
		})
		if err != nil {
			return ResearchIDEObjectRead{}, true, err
		}
		data := mustJSON(map[string]any{
			"snapshot":             result.Snapshot,
			"content":              result.Read.Content,
			"observation_metadata": result.Read.Metadata,
			"observation_event_id": observationEventID(result.ObservationEvent),
		})
		return ResearchIDEObjectRead{
			ObjectKind: objectKind,
			ObjectID:   objectID,
			MissionID:  missionID,
			Summary:    summarizeSourceSnapshot(result.Snapshot).Summary,
			Refs:       summarizeSourceSnapshot(result.Snapshot).Refs,
			Data:       string(data),
			Truncated:  result.Read.Metadata.Truncated,
			NextOffset: int(result.Read.Metadata.NextOffset),
		}, true, nil
	case ResearchIDEObjectRawArtifact:
		record, err := s.store.GetRawArtifact(ctx, objectID)
		if err != nil {
			return ResearchIDEObjectRead{}, true, err
		}
		if record.MissionID != missionID {
			return ResearchIDEObjectRead{}, true, fmt.Errorf("%w: raw_artifact %s belongs to another mission", ErrInvalidInput, objectID)
		}
		summary := summarizeRawArtifact(record)
		if UploadedArtifactReadKind(record) == "metadata" {
			metadata := UploadedArtifactMetadata(record)
			metadata["metadata_only"] = true
			metadata["note"] = "binary artifact content is not returned through research.read"
			return ResearchIDEObjectRead{ObjectKind: objectKind, ObjectID: objectID, MissionID: missionID, Summary: summary.Summary, Refs: summary.Refs, Data: string(mustJSON(metadata))}, true, nil
		}
		if !pdftext.IsPDFMediaType(record.MediaType) && !pdftext.IsPDFBytes(record.Content) {
			return ResearchIDEObjectRead{}, false, nil
		}
		chunk, err := pdftext.ExtractChunk(record.Content, offset, maxBytes)
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
			return ResearchIDEObjectRead{ObjectKind: objectKind, ObjectID: objectID, MissionID: missionID, Summary: summary.Summary, Refs: summary.Refs, Data: string(data)}, true, nil
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
			"suggested_read_bytes": pdftext.DefaultChunkMaxBytes,
			"max_read_bytes":       pdftext.MaxChunkBytes,
		})
		return ResearchIDEObjectRead{
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
		return ResearchIDEObjectRead{}, false, nil
	}
}

type researchIDEReadPayload struct {
	summary    ResearchIDEObjectSummary
	data       []byte
	truncated  bool
	nextOffset int
}

func (s *Service) readSourceSnapshotPDFPayload(ctx context.Context, missionID string, snapshot SourceSnapshot, offset int, maxBytes int) (researchIDEReadPayload, bool, error) {
	if snapshot.Access.RetrievalPolicy == SourceRetrievalPolicyLiveReference {
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
	if !pdftext.IsPDFMediaType(artifact.MediaType) && !pdftext.IsPDFBytes(artifact.Content) {
		return researchIDEReadPayload{}, false, nil
	}
	summary := summarizeSourceSnapshot(snapshot)
	artifactMetadata := UploadedArtifactMetadata(artifact)
	sourceMetadata := sourceSnapshotReadMetadata(snapshot, summary)
	chunk, err := pdftext.ExtractChunk(artifact.Content, offset, maxBytes)
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
		"suggested_read_bytes": pdftext.DefaultChunkMaxBytes,
		"max_read_bytes":       researchIDEMaxBytes,
		"read_kind":            "source_pdf_text",
	})
	return researchIDEReadPayload{
		summary:    summary,
		data:       data,
		truncated:  chunk.Truncated,
		nextOffset: chunk.NextOffset,
	}, true, nil
}

func sourceSnapshotReadMetadata(snapshot SourceSnapshot, summary ResearchIDEObjectSummary) map[string]any {
	metadata := map[string]any{
		"snapshot_id":      snapshot.SnapshotID,
		"mission_id":       snapshot.MissionID,
		"title":            strings.TrimSpace(snapshot.Title),
		"connector_type":   snapshot.Connector.ConnectorType,
		"retrieval_policy": snapshot.Access.RetrievalPolicy,
		"state":            firstNonEmpty(snapshot.State.State, SourceStateActive),
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

func (s *Service) GrepMissionObjects(ctx context.Context, missionID, query string, limit int, cursor string) (ResearchIDEGrepResult, error) {
	return s.grepMissionObjects(ctx, missionID, query, limit, cursor, false)
}

func (s *Service) GrepMissionObjectsLegacy(ctx context.Context, missionID, query string, limit int, cursor string) (ResearchIDEGrepResult, error) {
	return s.grepMissionObjects(ctx, missionID, query, limit, cursor, true)
}

func (s *Service) grepMissionObjects(ctx context.Context, missionID, query string, limit int, cursor string, legacy bool) (ResearchIDEGrepResult, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateID("mis_", missionID); err != nil {
		return ResearchIDEGrepResult{}, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return ResearchIDEGrepResult{}, fmt.Errorf("%w: grep query is required", ErrInvalidInput)
	}
	limit = clampResearchIDELimit(limit)
	offset, err := parseResearchIDECursor(cursor)
	if err != nil {
		return ResearchIDEGrepResult{}, err
	}
	candidates, err := s.grepCandidates(ctx, missionID, query, legacy)
	if err != nil {
		return ResearchIDEGrepResult{}, err
	}
	var matches []ResearchIDEGrepMatch
	lowerQuery := strings.ToLower(query)
	for _, candidate := range candidates {
		pos := strings.Index(strings.ToLower(candidate.text), lowerQuery)
		if pos < 0 {
			continue
		}
		matches = append(matches, ResearchIDEGrepMatch{
			ObjectKind: candidate.summary.ObjectKind,
			ObjectID:   candidate.summary.ObjectID,
			MissionID:  missionID,
			Snippet:    snippet(candidate.text, pos, len(query)),
			Position:   pos,
			Refs:       candidate.summary.Refs,
		})
	}
	page, next, truncated := paginateMatches(matches, offset, limit)
	return ResearchIDEGrepResult{MissionID: missionID, Query: query, Matches: page, NextCursor: next, Limit: limit, Truncated: truncated}, nil
}

func (s *Service) ListObjectReferences(ctx context.Context, missionID, objectKind, objectID string, limit int, cursor string) (ResearchIDEReferences, error) {
	return s.listObjectReferences(ctx, missionID, objectKind, objectID, limit, cursor, false)
}

func (s *Service) ListObjectReferencesLegacy(ctx context.Context, missionID, objectKind, objectID string, limit int, cursor string) (ResearchIDEReferences, error) {
	return s.listObjectReferences(ctx, missionID, objectKind, objectID, limit, cursor, true)
}

func (s *Service) listObjectReferences(ctx context.Context, missionID, objectKind, objectID string, limit int, cursor string, legacy bool) (ResearchIDEReferences, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateID("mis_", missionID); err != nil {
		return ResearchIDEReferences{}, err
	}
	limit = clampResearchIDELimit(limit)
	offset, err := parseResearchIDECursor(cursor)
	if err != nil {
		return ResearchIDEReferences{}, err
	}
	objectKind = normalizeResearchIDEObjectKind(objectKind)
	objectID = strings.TrimSpace(objectID)
	summary, _, err := s.readObjectPayload(ctx, missionID, objectKind, objectID, legacy)
	if err != nil {
		return ResearchIDEReferences{}, err
	}
	target := ResearchIDEObjectRef{ObjectKind: objectKind, ObjectID: objectID}
	all, err := s.allObjectSummaries(ctx, missionID, "", legacy)
	if err != nil {
		return ResearchIDEReferences{}, err
	}
	var backward []ResearchIDEObjectRef
	for _, item := range all {
		if item.ObjectKind == objectKind && item.ObjectID == objectID {
			continue
		}
		if containsResearchIDERef(item.Refs, target) {
			backward = append(backward, ResearchIDEObjectRef{ObjectKind: item.ObjectKind, ObjectID: item.ObjectID})
		}
	}
	forward, backward, next, truncated := paginateReferenceSets(summary.Refs, backward, offset, limit)
	return ResearchIDEReferences{
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

func (s *Service) listResearchObjects(ctx context.Context, missionID string) ([]EvidenceRecord, []ClaimRecord, []QuestionRecord, []OptionRecord, []ProposalBundle, error) {
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

func (s *Service) listSourceSnapshots(ctx context.Context, missionID string) ([]SourceSnapshot, error) {
	return s.ListSourceSnapshotsWithState(ctx, ListSourceSnapshotsRequest{MissionID: missionID})
}

func (s *Service) listRawArtifacts(ctx context.Context, missionID string) ([]RawArtifact, error) {
	store, ok := s.store.(RawArtifactListStore)
	if !ok {
		return nil, fmt.Errorf("%w: raw artifact list store is required", ErrInvalidInput)
	}
	return store.ListRawArtifacts(ctx, missionID)
}

func (s *Service) ListRawArtifacts(ctx context.Context, missionID string) ([]RawArtifact, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	return s.listRawArtifacts(ctx, missionID)
}

func (s *Service) listVisibleRawArtifacts(ctx context.Context, missionID string) ([]RawArtifact, error) {
	artifacts, err := s.listRawArtifacts(ctx, missionID)
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
	if len(rejected) == 0 && len(stagedCandidates) == 0 {
		return artifacts, nil
	}
	visible := artifacts[:0]
	for _, artifact := range artifacts {
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

func (s *Service) listReportObjects(ctx context.Context, missionID string) ([]Report, []ReportVersion, error) {
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

func (s *Service) allObjectSummaries(ctx context.Context, missionID, objectKind string, legacy bool) ([]ResearchIDEObjectSummary, error) {
	var items []ResearchIDEObjectSummary
	add := func(kind string, summaries []ResearchIDEObjectSummary) {
		if objectKind == "" || objectKind == kind {
			items = append(items, summaries...)
		}
	}
	if objectKind == "" || objectKind == ResearchIDEObjectSourceSnapshot {
		snapshots, err := s.listSourceSnapshots(ctx, missionID)
		if err != nil {
			return nil, err
		}
		var summaries []ResearchIDEObjectSummary
		for _, snapshot := range snapshots {
			summaries = append(summaries, summarizeSourceSnapshot(snapshot))
		}
		add(ResearchIDEObjectSourceSnapshot, summaries)
	}
	if objectKind == "" || objectKind == ResearchIDEObjectRawArtifact {
		artifacts, err := s.listVisibleRawArtifacts(ctx, missionID)
		if err != nil {
			return nil, err
		}
		var summaries []ResearchIDEObjectSummary
		for _, artifact := range artifacts {
			summaries = append(summaries, summarizeRawArtifact(artifact))
		}
		add(ResearchIDEObjectRawArtifact, summaries)
	}
	if legacy && (objectKind == "" || isResearchRecordKind(objectKind)) {
		evidence, claims, questions, options, proposals, err := s.listResearchObjects(ctx, missionID)
		if err != nil {
			return nil, err
		}
		var summaries []ResearchIDEObjectSummary
		for _, record := range evidence {
			summaries = append(summaries, summarizeEvidence(record))
		}
		add(ResearchIDEObjectEvidenceRecord, filterSummaries(summaries, ResearchIDEObjectEvidenceRecord))
		summaries = summaries[:0]
		for _, record := range claims {
			summaries = append(summaries, summarizeClaim(record))
		}
		add(ResearchIDEObjectClaimRecord, summaries)
		summaries = summaries[:0]
		for _, record := range questions {
			summaries = append(summaries, summarizeQuestion(record))
		}
		add(ResearchIDEObjectQuestionRecord, summaries)
		summaries = summaries[:0]
		for _, record := range options {
			summaries = append(summaries, summarizeOption(record))
		}
		add(ResearchIDEObjectOptionRecord, summaries)
		summaries = summaries[:0]
		for _, record := range proposals {
			summaries = append(summaries, summarizeProposal(record))
		}
		add(ResearchIDEObjectProposalBundle, summaries)
	}
	if legacy && (objectKind == "" || objectKind == ResearchIDEObjectReport || objectKind == ResearchIDEObjectReportVersion || objectKind == ResearchIDEObjectReportBlock) {
		reports, versions, err := s.listReportObjects(ctx, missionID)
		if err != nil {
			return nil, err
		}
		var reportSummaries []ResearchIDEObjectSummary
		for _, report := range reports {
			reportSummaries = append(reportSummaries, summarizeReport(report))
		}
		add(ResearchIDEObjectReport, reportSummaries)
		var versionSummaries []ResearchIDEObjectSummary
		var blockSummaries []ResearchIDEObjectSummary
		for _, version := range versions {
			versionSummaries = append(versionSummaries, summarizeReportVersion(version))
			blocks, err := s.store.ListReportBlocks(ctx, version.ReportVersionID)
			if err != nil {
				return nil, err
			}
			for _, block := range blocks {
				blockSummaries = append(blockSummaries, summarizeReportBlock(block))
			}
		}
		add(ResearchIDEObjectReportVersion, versionSummaries)
		add(ResearchIDEObjectReportBlock, blockSummaries)
	}
	if objectKind == "" || objectKind == ResearchIDEObjectLedgerEvent {
		events, err := s.store.ListLedgerEvents(ctx, missionID)
		if err != nil {
			return nil, err
		}
		var summaries []ResearchIDEObjectSummary
		for _, event := range events {
			summaries = append(summaries, summarizeLedgerEvent(event))
		}
		add(ResearchIDEObjectLedgerEvent, summaries)
	}
	if objectKind != "" && !legacyResearchIDEObjectKindAllowed(objectKind, legacy) {
		return nil, fmt.Errorf("%w: unsupported object kind", ErrInvalidInput)
	}
	return items, nil
}

func (s *Service) readObjectPayload(ctx context.Context, missionID, objectKind, objectID string, legacy bool) (ResearchIDEObjectSummary, []byte, error) {
	if !legacyResearchIDEObjectKindAllowed(objectKind, legacy) {
		return ResearchIDEObjectSummary{}, nil, fmt.Errorf("%w: unsupported object kind", ErrInvalidInput)
	}
	switch objectKind {
	case ResearchIDEObjectSourceSnapshot:
		record, err := s.store.GetSourceSnapshot(ctx, objectID)
		if err != nil {
			return ResearchIDEObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return ResearchIDEObjectSummary{}, nil, fmt.Errorf("%w: source_snapshot %s belongs to another mission", ErrInvalidInput, objectID)
		}
		record.State, _ = s.sourceState(ctx, missionID, record.SnapshotID)
		if record.State.Removed {
			return ResearchIDEObjectSummary{}, nil, fmt.Errorf("%w: source_snapshot %s is removed", ErrInvalidInput, objectID)
		}
		if record.Access.RetrievalPolicy == SourceRetrievalPolicyLiveReference && record.Connector.ConnectorType == SourceConnectorTypeLocalPath {
			locator, err := parseLocalPathLocator(record.Locators)
			if err != nil {
				return ResearchIDEObjectSummary{}, nil, err
			}
			if locator.PathKind == "directory" {
				tree, err := s.TreeLocalPathSource(ctx, TreeLocalPathSourceRequest{
					MissionID:  missionID,
					SnapshotID: objectID,
					Depth:      1,
					Limit:      researchIDEDefaultLimit,
					Producer:   Producer{Type: "research_ide", ID: "plasma"},
				})
				if err != nil {
					return ResearchIDEObjectSummary{}, nil, err
				}
				return summarizeSourceSnapshot(tree.Snapshot), mustJSON(map[string]any{
					"snapshot":             tree.Snapshot,
					"tree":                 tree.Tree,
					"observation_metadata": tree.Tree.Metadata,
					"observation_event_id": observationEventID(tree.ObservationEvent),
				}), nil
			}
			read, err := s.ReadLocalPathSource(ctx, ReadLocalPathSourceRequest{
				MissionID:  missionID,
				SnapshotID: objectID,
				MaxBytes:   int64(researchIDEDefaultBytes),
				Producer:   Producer{Type: "research_ide", ID: "plasma"},
			})
			if err != nil {
				return ResearchIDEObjectSummary{}, nil, err
			}
			return summarizeSourceSnapshot(read.Snapshot), mustJSON(map[string]any{
				"snapshot":             read.Snapshot,
				"content":              read.Read.Content,
				"observation_metadata": read.Read.Metadata,
				"observation_event_id": observationEventID(read.ObservationEvent),
			}), nil
		}
		if payload, handled, err := s.readSourceSnapshotPDFPayload(ctx, missionID, record, 0, researchIDEDefaultBytes); handled || err != nil {
			return payload.summary, payload.data, err
		}
		return summarizeSourceSnapshot(record), mustJSON(record), nil
	case ResearchIDEObjectRawArtifact:
		record, err := s.store.GetRawArtifact(ctx, objectID)
		if err != nil {
			return ResearchIDEObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return ResearchIDEObjectSummary{}, nil, fmt.Errorf("%w: raw_artifact %s belongs to another mission", ErrInvalidInput, objectID)
		}
		rejected, err := s.isRejectedReportPatchArtifact(ctx, missionID, record.ArtifactID)
		if err != nil {
			return ResearchIDEObjectSummary{}, nil, err
		}
		if rejected {
			return ResearchIDEObjectSummary{}, nil, fmt.Errorf("%w: raw_artifact %s is a rejected report patch artifact", ErrInvalidInput, objectID)
		}
		stagedCandidate, err := s.isStagedSourceCandidateArtifact(ctx, missionID, record.ArtifactID)
		if err != nil {
			return ResearchIDEObjectSummary{}, nil, err
		}
		if stagedCandidate {
			return ResearchIDEObjectSummary{}, nil, fmt.Errorf("%w: raw_artifact %s is an unapproved source candidate artifact; use plasma.sources.candidates.read", ErrInvalidInput, objectID)
		}
		if UploadedArtifactReadKind(record) == "metadata" {
			metadata := UploadedArtifactMetadata(record)
			metadata["metadata_only"] = true
			metadata["note"] = "binary artifact content is not returned through research.read"
			return summarizeRawArtifact(record), mustJSON(metadata), nil
		}
		if pdftext.IsPDFMediaType(record.MediaType) || pdftext.IsPDFBytes(record.Content) {
			chunk, err := pdftext.ExtractChunk(record.Content, 0, researchIDEDefaultBytes)
			if err != nil {
				return summarizeRawArtifact(record), mustJSON(map[string]any{
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
			return summarizeRawArtifact(record), mustJSON(map[string]any{
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
				"suggested_read_bytes": pdftext.DefaultChunkMaxBytes,
				"max_read_bytes":       pdftext.MaxChunkBytes,
			}), nil
		}
		if !utf8.Valid(record.Content) {
			return summarizeRawArtifact(record), mustJSON(map[string]any{
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
		return summarizeRawArtifact(record), record.Content, nil
	case ResearchIDEObjectEvidenceRecord:
		record, err := s.store.GetEvidenceRecord(ctx, objectID)
		if err != nil {
			return ResearchIDEObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return ResearchIDEObjectSummary{}, nil, fmt.Errorf("%w: evidence_record %s belongs to another mission", ErrInvalidInput, objectID)
		}
		return summarizeEvidence(record), mustJSON(record), nil
	case ResearchIDEObjectClaimRecord:
		record, err := s.store.GetClaimRecord(ctx, objectID)
		if err != nil {
			return ResearchIDEObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return ResearchIDEObjectSummary{}, nil, fmt.Errorf("%w: claim_record %s belongs to another mission", ErrInvalidInput, objectID)
		}
		return summarizeClaim(record), mustJSON(record), nil
	case ResearchIDEObjectQuestionRecord:
		record, err := s.store.GetQuestionRecord(ctx, objectID)
		if err != nil {
			return ResearchIDEObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return ResearchIDEObjectSummary{}, nil, fmt.Errorf("%w: question_record %s belongs to another mission", ErrInvalidInput, objectID)
		}
		return summarizeQuestion(record), mustJSON(record), nil
	case ResearchIDEObjectOptionRecord:
		record, err := s.store.GetOptionRecord(ctx, objectID)
		if err != nil {
			return ResearchIDEObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return ResearchIDEObjectSummary{}, nil, fmt.Errorf("%w: option_record %s belongs to another mission", ErrInvalidInput, objectID)
		}
		return summarizeOption(record), mustJSON(record), nil
	case ResearchIDEObjectProposalBundle:
		record, err := s.store.GetProposalBundle(ctx, objectID)
		if err != nil {
			return ResearchIDEObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return ResearchIDEObjectSummary{}, nil, fmt.Errorf("%w: proposal_bundle %s belongs to another mission", ErrInvalidInput, objectID)
		}
		return summarizeProposal(record), mustJSON(record), nil
	case ResearchIDEObjectReport:
		record, err := s.store.GetReport(ctx, objectID)
		if err != nil {
			return ResearchIDEObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return ResearchIDEObjectSummary{}, nil, fmt.Errorf("%w: report %s belongs to another mission", ErrInvalidInput, objectID)
		}
		return summarizeReport(record), mustJSON(record), nil
	case ResearchIDEObjectReportVersion:
		record, err := s.store.GetReportVersion(ctx, objectID)
		if err != nil {
			return ResearchIDEObjectSummary{}, nil, err
		}
		if record.MissionID != missionID {
			return ResearchIDEObjectSummary{}, nil, fmt.Errorf("%w: report_version %s belongs to another mission", ErrInvalidInput, objectID)
		}
		return summarizeReportVersion(record), mustJSON(record), nil
	case ResearchIDEObjectReportBlock:
		block, err := s.findReportBlock(ctx, missionID, objectID)
		if err != nil {
			return ResearchIDEObjectSummary{}, nil, err
		}
		return summarizeReportBlock(block), mustJSON(block), nil
	case ResearchIDEObjectLedgerEvent:
		event, err := s.findLedgerEvent(ctx, missionID, objectID)
		if err != nil {
			return ResearchIDEObjectSummary{}, nil, err
		}
		return summarizeLedgerEvent(event), mustJSON(event), nil
	default:
		return ResearchIDEObjectSummary{}, nil, fmt.Errorf("%w: unsupported object kind", ErrInvalidInput)
	}
}

type grepCandidate struct {
	summary ResearchIDEObjectSummary
	text    string
}

func (s *Service) grepCandidates(ctx context.Context, missionID string, query string, legacy bool) ([]grepCandidate, error) {
	items, err := s.allObjectSummaries(ctx, missionID, "", legacy)
	if err != nil {
		return nil, err
	}
	var candidates []grepCandidate
	for _, item := range items {
		if item.ObjectKind == ResearchIDEObjectSourceSnapshot {
			source, err := s.GetSourceSnapshot(ctx, item.ObjectID)
			if err == nil {
				source.State, _ = s.sourceState(ctx, missionID, source.SnapshotID)
			}
			if err == nil && source.State.Removed {
				continue
			}
			if err == nil && source.Access.RetrievalPolicy == SourceRetrievalPolicyLiveReference && source.Connector.ConnectorType == SourceConnectorTypeLocalPath {
				grep, err := s.GrepLocalPathSource(ctx, GrepLocalPathSourceRequest{
					MissionID:  missionID,
					SnapshotID: item.ObjectID,
					Query:      query,
					Producer:   Producer{Type: "research_ide", ID: "plasma"},
				})
				if err == nil {
					for _, match := range grep.Grep.Matches {
						candidates = append(candidates, grepCandidate{summary: item, text: match.Snippet})
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
		candidates = append(candidates, grepCandidate{summary: item, text: text})
	}
	return candidates, nil
}

func (s *Service) findReportBlock(ctx context.Context, missionID, blockID string) (ReportBlock, error) {
	_, versions, err := s.listReportObjects(ctx, missionID)
	if err != nil {
		return ReportBlock{}, err
	}
	for _, version := range versions {
		blocks, err := s.store.ListReportBlocks(ctx, version.ReportVersionID)
		if err != nil {
			return ReportBlock{}, err
		}
		for _, block := range blocks {
			if block.BlockID == blockID {
				if block.MissionID != missionID {
					return ReportBlock{}, fmt.Errorf("%w: report_block %s belongs to another mission", ErrInvalidInput, blockID)
				}
				return block, nil
			}
		}
	}
	return ReportBlock{}, fmt.Errorf("%w: report_block %s not found", ErrInvalidInput, blockID)
}

func (s *Service) reportVersionBlockPage(ctx context.Context, missionID, versionID string, limit int, cursor string) (ResearchIDEPage, error) {
	limit = clampResearchIDELimit(limit)
	offset, err := parseResearchIDECursor(cursor)
	if err != nil {
		return ResearchIDEPage{}, err
	}
	version, err := s.store.GetReportVersion(ctx, versionID)
	if err != nil {
		return ResearchIDEPage{}, err
	}
	if version.MissionID != missionID {
		return ResearchIDEPage{}, fmt.Errorf("%w: report_version %s belongs to another mission", ErrInvalidInput, versionID)
	}
	blocks, err := s.store.ListReportBlocks(ctx, versionID)
	if err != nil {
		return ResearchIDEPage{}, err
	}
	summaries := make([]ResearchIDEObjectSummary, 0, len(blocks))
	for _, block := range blocks {
		if block.MissionID != missionID {
			return ResearchIDEPage{}, fmt.Errorf("%w: report_block %s belongs to another mission", ErrInvalidInput, block.BlockID)
		}
		summaries = append(summaries, summarizeReportBlock(block))
	}
	pageItems, next, truncated := paginateSummaries(summaries, offset, limit)
	return ResearchIDEPage{
		MissionID:  missionID,
		ObjectKind: ResearchIDEObjectReportBlock,
		Items:      pageItems,
		NextCursor: next,
		Limit:      limit,
		Truncated:  truncated,
	}, nil
}

func (s *Service) findLedgerEvent(ctx context.Context, missionID, eventID string) (LedgerEvent, error) {
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return LedgerEvent{}, err
	}
	for _, event := range events {
		if event.EventID == eventID {
			return event, nil
		}
	}
	return LedgerEvent{}, fmt.Errorf("%w: ledger_event %s not found", ErrInvalidInput, eventID)
}

func summarizeSourceSnapshot(snapshot SourceSnapshot) ResearchIDEObjectSummary {
	refs := make([]ResearchIDEObjectRef, 0, len(snapshot.ArtifactIDs))
	for _, id := range snapshot.ArtifactIDs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectRawArtifact, ObjectID: id})
	}
	metadata := map[string]any{
		"connector_type":   snapshot.Connector.ConnectorType,
		"retrieval_policy": snapshot.Access.RetrievalPolicy,
		"state":            firstNonEmpty(snapshot.State.State, SourceStateActive),
		"removed":          snapshot.State.Removed,
	}
	if snapshot.Connector.ConnectorType == SourceConnectorTypeLocalPath {
		if locator, err := parseLocalPathLocator(snapshot.Locators); err == nil {
			metadata["root_id"] = locator.RootID
			metadata["relative_path"] = locator.RelativePath
			metadata["path_kind"] = locator.PathKind
		}
	}
	if snapshot.Connector.ConnectorType == SourceConnectorTypeMediaURL {
		if locator, err := parseMediaLocator(snapshot.Locators); err == nil {
			metadata["media_kind"] = locator.MediaKind
			metadata["mime_type"] = locator.MIMEType
			metadata["byte_size"] = locator.ByteSize
			metadata["width"] = locator.Width
			metadata["height"] = locator.Height
			metadata["canonical_url"] = locator.CanonicalURL
			metadata["source_page_url"] = locator.SourcePageURL
			metadata["direct_media_url"] = locator.DirectMediaURL
			metadata["license"] = locator.License
			metadata["attribution"] = locator.Attribution
			metadata["inspection_support"] = locator.InspectionSupport
		}
	}
	if snapshot.Connector.ConnectorType == SourceConnectorTypeFileUpload {
		for key, value := range uploadedFileLocatorMetadata(snapshot.Locators) {
			metadata[key] = value
		}
	}
	if snapshot.Connector.ConnectorType == SourceConnectorTypePDFURL || snapshot.Connector.ConnectorType == SourceConnectorTypeFileUpload {
		for key, value := range pdfLocatorMetadata(snapshot.Locators) {
			metadata[key] = value
		}
	}
	return ResearchIDEObjectSummary{ObjectKind: ResearchIDEObjectSourceSnapshot, ObjectID: snapshot.SnapshotID, MissionID: snapshot.MissionID, Summary: firstNonEmpty(snapshot.Title, snapshot.Connector.ExternalURI, snapshot.SnapshotID), Refs: refs, Metadata: metadata}
}

func pdfLocatorMetadata(raw json.RawMessage) map[string]any {
	metadata := map[string]any{}
	if len(raw) == 0 {
		return metadata
	}
	var locators []map[string]any
	if err := json.Unmarshal(raw, &locators); err != nil {
		var locator map[string]any
		if err := json.Unmarshal(raw, &locator); err != nil {
			return metadata
		}
		locators = []map[string]any{locator}
	}
	for _, locator := range locators {
		if !isPDFLocatorMap(locator) {
			continue
		}
		for _, key := range []string{"url", "filename", "original_filename", "sanitized_filename", "mime_type", "media_type", "byte_size", "sha256", "page_count", "text_length", "text_length_known", "extraction_support"} {
			if value, ok := locator[key]; ok {
				metadata[key] = value
			}
		}
		return metadata
	}
	return metadata
}

func uploadedFileLocatorMetadata(raw json.RawMessage) map[string]any {
	metadata := map[string]any{}
	if len(raw) == 0 {
		return metadata
	}
	var locators []map[string]any
	if err := json.Unmarshal(raw, &locators); err != nil {
		var locator map[string]any
		if err := json.Unmarshal(raw, &locator); err != nil {
			return metadata
		}
		locators = []map[string]any{locator}
	}
	for _, locator := range locators {
		locatorType := uploadedFileLocatorMapType(locator)
		if locatorType == "" {
			continue
		}
		metadata["locator_type"] = locatorType
		for _, key := range []string{"original_filename", "sanitized_filename", "filename", "media_kind", "content_kind", "byte_size", "sha256", "uploaded_at"} {
			if value, ok := locator[key]; ok {
				metadata[key] = value
			}
		}
		if filename := firstLocatorMapString(locator, "sanitized_filename", "filename", "original_filename"); filename != "" {
			metadata["filename"] = filename
		}
		if mimeType := firstLocatorMapString(locator, "mime_type", "media_type"); mimeType != "" {
			metadata["mime_type"] = mimeType
		}
		return metadata
	}
	return metadata
}

func uploadedFileLocatorMapType(locator map[string]any) string {
	discriminator := locatorMapDiscriminator(locator)
	switch discriminator {
	case SourceLocatorTypeFullDocument, SourceLocatorTypePDFDocument, SourceLocatorTypeMedia:
		return discriminator
	case SourceConnectorTypeFileUpload:
	default:
		return ""
	}
	contentKind := strings.TrimSpace(fmt.Sprint(locator["content_kind"]))
	mediaType := firstLocatorMapString(locator, "mime_type", "media_type")
	switch {
	case contentKind == UploadedContentKindPDF || mediaType == "application/pdf":
		return SourceLocatorTypePDFDocument
	case contentKind == UploadedContentKindImage || strings.HasPrefix(mediaType, "image/"):
		return SourceLocatorTypeMedia
	default:
		return SourceLocatorTypeFullDocument
	}
}

func isPDFLocatorMap(locator map[string]any) bool {
	discriminator := locatorMapDiscriminator(locator)
	if discriminator == SourceLocatorTypePDFDocument {
		return true
	}
	if discriminator != SourceConnectorTypeFileUpload {
		return false
	}
	contentKind := strings.TrimSpace(fmt.Sprint(locator["content_kind"]))
	mediaType := firstLocatorMapString(locator, "mime_type", "media_type")
	return contentKind == UploadedContentKindPDF || mediaType == "application/pdf"
}

func firstLocatorMapString(locator map[string]any, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(fmt.Sprint(locator[key]))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func locatorMapDiscriminator(locator map[string]any) string {
	if value := strings.TrimSpace(fmt.Sprint(locator["locator_type"])); value != "" && value != "<nil>" {
		return value
	}
	value := strings.TrimSpace(fmt.Sprint(locator["kind"]))
	if value == "<nil>" {
		return ""
	}
	return value
}

func summarizeRawArtifact(artifact RawArtifact) ResearchIDEObjectSummary {
	return ResearchIDEObjectSummary{ObjectKind: ResearchIDEObjectRawArtifact, ObjectID: artifact.ArtifactID, MissionID: artifact.MissionID, Summary: firstNonEmpty(artifact.Filename, artifact.MediaType, artifact.ArtifactID), Metadata: map[string]any{"byte_size": artifact.ByteSize, "media_type": artifact.MediaType, "read_kind": UploadedArtifactReadKind(artifact)}}
}

func summarizeEvidence(record EvidenceRecord) ResearchIDEObjectSummary {
	refs := make([]ResearchIDEObjectRef, 0, len(record.SnapshotRefs)*2)
	for _, ref := range record.SnapshotRefs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectSourceSnapshot, ObjectID: ref.SnapshotID})
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectRawArtifact, ObjectID: ref.ArtifactID})
	}
	return ResearchIDEObjectSummary{ObjectKind: ResearchIDEObjectEvidenceRecord, ObjectID: record.EvidenceID, MissionID: record.MissionID, Summary: record.Summary, Refs: dedupeResearchIDERefs(refs), Metadata: map[string]any{"state": record.State, "evidence_type": record.EvidenceType}}
}

func summarizeClaim(record ClaimRecord) ResearchIDEObjectSummary {
	var refs []ResearchIDEObjectRef
	for _, id := range record.SupportingEvidenceIDs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectEvidenceRecord, ObjectID: id})
	}
	for _, id := range record.OpposingEvidenceIDs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectEvidenceRecord, ObjectID: id})
	}
	for _, id := range record.DependsOnQuestionIDs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectQuestionRecord, ObjectID: id})
	}
	return ResearchIDEObjectSummary{ObjectKind: ResearchIDEObjectClaimRecord, ObjectID: record.ClaimID, MissionID: record.MissionID, Summary: record.Text, Refs: dedupeResearchIDERefs(refs), Metadata: map[string]any{"state": record.State, "claim_type": record.ClaimType}}
}

func summarizeQuestion(record QuestionRecord) ResearchIDEObjectSummary {
	var refs []ResearchIDEObjectRef
	for _, id := range record.RelatedEvidenceIDs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectEvidenceRecord, ObjectID: id})
	}
	for _, id := range record.RelatedClaimIDs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectClaimRecord, ObjectID: id})
	}
	return ResearchIDEObjectSummary{ObjectKind: ResearchIDEObjectQuestionRecord, ObjectID: record.QuestionID, MissionID: record.MissionID, Summary: record.Text, Refs: dedupeResearchIDERefs(refs), Metadata: map[string]any{"state": record.State, "priority": record.Priority}}
}

func summarizeOption(record OptionRecord) ResearchIDEObjectSummary {
	var refs []ResearchIDEObjectRef
	for _, id := range record.SupportingClaimIDs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectClaimRecord, ObjectID: id})
	}
	return ResearchIDEObjectSummary{ObjectKind: ResearchIDEObjectOptionRecord, ObjectID: record.OptionID, MissionID: record.MissionID, Summary: firstNonEmpty(record.Title, record.Description, record.OptionID), Refs: refs, Metadata: map[string]any{"state": record.State, "risk_level": record.RiskLevel}}
}

func summarizeProposal(record ProposalBundle) ResearchIDEObjectSummary {
	var refs []ResearchIDEObjectRef
	for _, ref := range record.ObjectRefs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: normalizeResearchIDEObjectKind(ref.ObjectKind), ObjectID: ref.ObjectID})
	}
	return ResearchIDEObjectSummary{ObjectKind: ResearchIDEObjectProposalBundle, ObjectID: record.ProposalID, MissionID: record.MissionID, Summary: firstNonEmpty(record.Title, record.RequestedDecision, record.ProposalID), Refs: dedupeResearchIDERefs(refs), Metadata: map[string]any{"state": record.State}}
}

func summarizeReport(record Report) ResearchIDEObjectSummary {
	var refs []ResearchIDEObjectRef
	if record.ActiveVersionID != "" {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectReportVersion, ObjectID: record.ActiveVersionID})
	}
	return ResearchIDEObjectSummary{ObjectKind: ResearchIDEObjectReport, ObjectID: record.ReportID, MissionID: record.MissionID, Summary: record.Title, Refs: refs, Metadata: map[string]any{"state": record.State}}
}

func summarizeReportVersion(record ReportVersion) ResearchIDEObjectSummary {
	refs := []ResearchIDEObjectRef{{ObjectKind: ResearchIDEObjectReport, ObjectID: record.ReportID}}
	for _, id := range record.IncludedEvidenceScope.ClaimIDs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectClaimRecord, ObjectID: id})
	}
	for _, id := range record.IncludedEvidenceScope.EvidenceIDs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectEvidenceRecord, ObjectID: id})
	}
	for _, id := range record.IncludedEvidenceScope.QuestionIDs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectQuestionRecord, ObjectID: id})
	}
	return ResearchIDEObjectSummary{ObjectKind: ResearchIDEObjectReportVersion, ObjectID: record.ReportVersionID, MissionID: record.MissionID, Summary: record.State + " report version", Refs: dedupeResearchIDERefs(refs), Metadata: map[string]any{"report_id": record.ReportID, "state": record.State}}
}

func summarizeReportBlock(block ReportBlock) ResearchIDEObjectSummary {
	var refs []ResearchIDEObjectRef
	refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectReportVersion, ObjectID: block.ReportVersionID})
	for _, id := range block.SourceRefs.ClaimIDs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectClaimRecord, ObjectID: id})
	}
	for _, id := range block.SourceRefs.EvidenceIDs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectEvidenceRecord, ObjectID: id})
	}
	for _, id := range block.SourceRefs.SnapshotIDs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectSourceSnapshot, ObjectID: id})
	}
	for _, id := range block.SourceRefs.QuestionIDs {
		refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectQuestionRecord, ObjectID: id})
	}
	return ResearchIDEObjectSummary{ObjectKind: ResearchIDEObjectReportBlock, ObjectID: block.BlockID, MissionID: block.MissionID, Summary: block.BlockType, Refs: dedupeResearchIDERefs(refs), Metadata: map[string]any{"report_version_id": block.ReportVersionID, "block_type": block.BlockType}}
}

func summarizeLedgerEvent(event LedgerEvent) ResearchIDEObjectSummary {
	refs := []ResearchIDEObjectRef{}
	var payload struct {
		SnapshotID  string   `json:"snapshot_id"`
		ArtifactID  string   `json:"artifact_id"`
		ArtifactIDs []string `json:"artifact_ids"`
	}
	if json.Unmarshal(event.Payload, &payload) == nil {
		if strings.TrimSpace(payload.SnapshotID) != "" {
			refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectSourceSnapshot, ObjectID: strings.TrimSpace(payload.SnapshotID)})
		}
		if strings.TrimSpace(payload.ArtifactID) != "" {
			refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectRawArtifact, ObjectID: strings.TrimSpace(payload.ArtifactID)})
		}
		for _, artifactID := range payload.ArtifactIDs {
			artifactID = strings.TrimSpace(artifactID)
			if artifactID != "" {
				refs = append(refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectRawArtifact, ObjectID: artifactID})
			}
		}
	}
	return ResearchIDEObjectSummary{ObjectKind: ResearchIDEObjectLedgerEvent, ObjectID: event.EventID, MissionID: event.MissionID, Summary: fmt.Sprintf("#%d %s", event.Sequence, event.EventType), Refs: dedupeResearchIDERefs(refs), Metadata: map[string]any{"event_type": event.EventType, "sequence": event.Sequence}}
}

func normalizeResearchIDEObjectKind(kind string) string {
	return strings.TrimSpace(kind)
}

func knownResearchIDEObjectKind(kind string) bool {
	switch kind {
	case ResearchIDEObjectSourceSnapshot, ResearchIDEObjectRawArtifact, ResearchIDEObjectEvidenceRecord, ResearchIDEObjectClaimRecord, ResearchIDEObjectQuestionRecord, ResearchIDEObjectOptionRecord, ResearchIDEObjectProposalBundle, ResearchIDEObjectReport, ResearchIDEObjectReportVersion, ResearchIDEObjectReportBlock, ResearchIDEObjectLedgerEvent:
		return true
	default:
		return false
	}
}

func defaultResearchIDEObjectKind(kind string) bool {
	switch kind {
	case ResearchIDEObjectSourceSnapshot, ResearchIDEObjectRawArtifact, ResearchIDEObjectLedgerEvent:
		return true
	default:
		return false
	}
}

func legacyResearchIDEObjectKindAllowed(kind string, legacy bool) bool {
	if legacy {
		return knownResearchIDEObjectKind(kind)
	}
	return defaultResearchIDEObjectKind(kind)
}

func isResearchRecordKind(kind string) bool {
	switch kind {
	case ResearchIDEObjectEvidenceRecord, ResearchIDEObjectClaimRecord, ResearchIDEObjectQuestionRecord, ResearchIDEObjectOptionRecord, ResearchIDEObjectProposalBundle:
		return true
	default:
		return false
	}
}

func clampResearchIDELimit(limit int) int {
	if limit <= 0 {
		return researchIDEDefaultLimit
	}
	if limit > researchIDEMaxLimit {
		return researchIDEMaxLimit
	}
	return limit
}

func clampResearchIDEBytes(maxBytes int) int {
	if maxBytes <= 0 {
		return researchIDEDefaultBytes
	}
	if maxBytes > researchIDEMaxBytes {
		return researchIDEMaxBytes
	}
	return maxBytes
}

func parseResearchIDECursor(cursor string) (int, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(cursor)
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("%w: invalid cursor", ErrInvalidInput)
	}
	return offset, nil
}

func paginateSummaries(items []ResearchIDEObjectSummary, offset, limit int) ([]ResearchIDEObjectSummary, string, bool) {
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

func paginateMatches(items []ResearchIDEGrepMatch, offset, limit int) ([]ResearchIDEGrepMatch, string, bool) {
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

func paginateReferenceSets(forward []ResearchIDEObjectRef, backward []ResearchIDEObjectRef, offset, limit int) ([]ResearchIDEObjectRef, []ResearchIDEObjectRef, string, bool) {
	type referencedItem struct {
		direction string
		ref       ResearchIDEObjectRef
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
	var pageForward []ResearchIDEObjectRef
	var pageBackward []ResearchIDEObjectRef
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

func chunkBytes(data []byte, offset, maxBytes int) ([]byte, bool, int, error) {
	if !utf8.Valid(data) {
		return nil, false, 0, fmt.Errorf("%w: object payload is not UTF-8 text", ErrInvalidInput)
	}
	if offset > len(data) {
		return nil, false, 0, fmt.Errorf("%w: object payload offset is beyond content length", ErrInvalidInput)
	}
	if offset < len(data) && !utf8.RuneStart(data[offset]) {
		return nil, false, 0, fmt.Errorf("%w: object payload offset must align to UTF-8 boundary", ErrInvalidInput)
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
		return nil, false, 0, fmt.Errorf("%w: object payload could not be sliced as UTF-8", ErrInvalidInput)
	}
	chunk := append([]byte(nil), data[offset:end]...)
	if end < len(data) {
		return chunk, true, end, nil
	}
	return chunk, false, 0, nil
}

func snippet(text string, pos, queryLen int) string {
	start := pos - researchIDESnippetContext
	if start < 0 {
		start = 0
	}
	end := pos + queryLen + researchIDESnippetContext
	if end > len(text) {
		end = len(text)
	}
	return strings.TrimSpace(text[start:end])
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

func dedupeResearchIDERefs(refs []ResearchIDEObjectRef) []ResearchIDEObjectRef {
	seen := map[ResearchIDEObjectRef]bool{}
	out := make([]ResearchIDEObjectRef, 0, len(refs))
	for _, ref := range refs {
		if ref.ObjectKind == "" || ref.ObjectID == "" || seen[ref] {
			continue
		}
		seen[ref] = true
		out = append(out, ref)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ObjectKind == out[j].ObjectKind {
			return out[i].ObjectID < out[j].ObjectID
		}
		return out[i].ObjectKind < out[j].ObjectKind
	})
	return out
}

func containsResearchIDERef(refs []ResearchIDEObjectRef, target ResearchIDEObjectRef) bool {
	for _, ref := range refs {
		if ref == target {
			return true
		}
	}
	return false
}

func filterSummaries(items []ResearchIDEObjectSummary, kind string) []ResearchIDEObjectSummary {
	var filtered []ResearchIDEObjectSummary
	for _, item := range items {
		if item.ObjectKind == kind {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

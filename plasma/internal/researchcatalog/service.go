package researchcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// Service orchestrates research discovery without depending on the application layer.
type Service struct{ deps Dependencies }

func NewService(deps Dependencies) *Service { return &Service{deps: deps} }

func (s *Service) Outline(ctx context.Context, missionID string, legacy bool) (Outline, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateMissionID(missionID); err != nil {
		return Outline{}, err
	}
	projection, err := s.deps.GetMissionProjection(ctx, missionID)
	if err != nil {
		return Outline{}, err
	}
	snapshots, err := s.deps.ListSourceSnapshots(ctx, missionID)
	if err != nil {
		return Outline{}, err
	}
	artifacts, err := s.deps.ListVisibleRawArtifacts(ctx, missionID, legacy)
	if err != nil {
		return Outline{}, err
	}
	ledgerEvents, err := s.deps.ListEvents(ctx, missionID)
	if err != nil {
		return Outline{}, err
	}
	events := VisibleLedgerEvents(ledgerEvents, legacy)
	var reportArtifacts map[string]struct{}
	if !legacy {
		reportArtifacts = ReportArtifactIDs(ledgerEvents)
	}
	counts := map[string]int{ObjectSourceSnapshot: len(snapshots), ObjectRawArtifact: len(artifacts), ObjectLedgerEvent: len(events)}
	if legacy {
		for _, kind := range []string{ObjectEvidenceRecord, ObjectClaimRecord, ObjectQuestionRecord, ObjectOptionRecord, ObjectProposalBundle, ObjectReport, ObjectReportVersion} {
			counts[kind] = 0
		}
	}
	var evidence []recordCountEvidence
	if legacy {
		evidence, err = s.loadRecordCounts(ctx, missionID)
		if err != nil {
			return Outline{}, err
		}
		reports, versions, err := s.loadReportCounts(ctx, missionID)
		if err != nil {
			return Outline{}, err
		}
		counts[ObjectReport], counts[ObjectReportVersion] = len(reports), len(versions)
		for _, item := range evidence {
			for state, n := range item.states {
				if state == "__count__" {
					counts[item.kind] = n
					continue
				}
				counts[item.kind] += n
				if item.kind == ObjectEvidenceRecord || item.kind == ObjectClaimRecord || item.kind == ObjectQuestionRecord {
					counts[item.kind+"."+state] += n
				}
			}
		}
	}
	recent := make([]ObjectSummary, 0, 5)
	for i := len(events) - 1; i >= 0 && len(recent) < 5; i-- {
		if events[i].EventType == "mcp.tool.called" {
			continue
		}
		summary := SummarizeLedgerEvent(events[i])
		summary.Refs = s.filterRefs(summary.Refs, reportArtifacts)
		recent = append(recent, summary)
	}
	next := make([]ObjectRef, 0, 6)
	appendNext := func(ref ObjectRef) {
		if len(next) < 6 {
			next = append(next, ref)
		}
	}
	if legacy {
		for _, id := range projection.OpenQuestionIDs {
			appendNext(ObjectRef{ObjectKind: ObjectQuestionRecord, ObjectID: id})
		}
		if projection.ActiveReportVersionID != "" {
			appendNext(ObjectRef{ObjectKind: ObjectReportVersion, ObjectID: projection.ActiveReportVersionID})
		}
	}
	for _, snapshot := range snapshots {
		if len(next) >= 6 {
			break
		}
		appendNext(ObjectRef{ObjectKind: ObjectSourceSnapshot, ObjectID: snapshot.SnapshotID})
	}
	active := ""
	if legacy {
		active = projection.ActiveReportVersionID
	}
	return Outline{MissionID: missionID, LastSequence: lastSequence(ledgerEvents), Title: projection.Title, Objective: projection.Objective, Scope: projection.Scope, Counts: counts, ActiveReportVersionID: active, RecentLedgerEvents: recent, NextSuggestedObjectRefs: next}, nil
}

type recordCountEvidence struct {
	kind   string
	states map[string]int
}

func (s *Service) loadRecordCounts(ctx context.Context, missionID string) ([]recordCountEvidence, error) {
	evidence, err := s.deps.ListEvidenceRecords(ctx, missionID)
	if err != nil {
		return nil, err
	}
	claims, err := s.deps.ListClaimRecords(ctx, missionID)
	if err != nil {
		return nil, err
	}
	questions, err := s.deps.ListQuestionRecords(ctx, missionID)
	if err != nil {
		return nil, err
	}
	options, err := s.deps.ListOptionRecords(ctx, missionID)
	if err != nil {
		return nil, err
	}
	proposals, err := s.deps.ListProposalSummaries(ctx, missionID)
	if err != nil {
		return nil, err
	}
	states := func(values []string) map[string]int {
		out := map[string]int{}
		for _, value := range values {
			out[value]++
		}
		return out
	}
	claimStates := make([]string, len(claims))
	for i := range claims {
		claimStates[i] = claims[i].State
	}
	questionStates := make([]string, len(questions))
	for i := range questions {
		questionStates[i] = questions[i].State
	}
	return []recordCountEvidence{{ObjectEvidenceRecord, states(func() []string {
		out := make([]string, len(evidence))
		for i := range evidence {
			out[i] = evidence[i].State
		}
		return out
	}())}, {ObjectClaimRecord, states(claimStates)}, {ObjectQuestionRecord, states(questionStates)}, {ObjectOptionRecord, states(func() []string {
		out := make([]string, len(options))
		for i := range options {
			out[i] = options[i].State
		}
		return out
	}())}, {ObjectProposalBundle, map[string]int{"__count__": len(proposals)}}}, nil
}

func (s *Service) loadReportCounts(ctx context.Context, missionID string) ([]ObjectSummary, []ObjectSummary, error) {
	reports, err := s.deps.ListReportSummaries(ctx, missionID)
	if err != nil {
		return nil, nil, err
	}
	versions, err := s.deps.ListReportVersionSummaries(ctx, missionID)
	if err != nil {
		return nil, nil, err
	}
	return reports, versions, nil
}

func (s *Service) List(ctx context.Context, missionID, objectKind string, limit int, cursor string, legacy bool) (Page, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateMissionID(missionID); err != nil {
		return Page{}, err
	}
	objectKind = NormalizeObjectKind(objectKind)
	limit = ClampLimit(limit)
	offset, err := ParseCursor(cursor)
	if err != nil {
		return Page{}, err
	}
	items, err := s.AllObjectSummaries(ctx, missionID, objectKind, legacy)
	if err != nil {
		return Page{}, err
	}
	pageItems, next, truncated := PaginateSummaries(items, offset, limit)
	return Page{MissionID: missionID, ObjectKind: objectKind, Items: pageItems, NextCursor: next, Limit: limit, Truncated: truncated}, nil
}

func (s *Service) AllObjectSummaries(ctx context.Context, missionID, objectKind string, legacy bool) ([]ObjectSummary, error) {
	if objectKind != "" && !ObjectKindAllowed(objectKind, legacy) {
		return nil, fmt.Errorf("%w: unsupported object kind", producterror.ErrInvalidInput)
	}
	var items []ObjectSummary
	reportArtifacts, err := s.deps.ReportArtifactIDsHiddenFromResearchDiscovery(ctx, missionID, legacy)
	if err != nil {
		return nil, err
	}
	add := func(kind string, summaries []ObjectSummary) {
		if objectKind == "" || objectKind == kind {
			items = append(items, s.filterSummaries(summaries, reportArtifacts)...)
		}
	}
	if objectKind == "" || objectKind == ObjectSourceSnapshot {
		snapshots, err := s.deps.ListSourceSnapshots(ctx, missionID)
		if err != nil {
			return nil, err
		}
		var summaries []ObjectSummary
		for _, snapshot := range snapshots {
			summaries = append(summaries, SummarizeSourceSnapshot(snapshot))
		}
		add(ObjectSourceSnapshot, summaries)
	}
	if objectKind == "" || objectKind == ObjectRawArtifact {
		artifacts, err := s.deps.ListVisibleRawArtifacts(ctx, missionID, legacy)
		if err != nil {
			return nil, err
		}
		var summaries []ObjectSummary
		for _, artifact := range artifacts {
			readKind := ""
			if s.deps.RawArtifactReadKind != nil {
				readKind = s.deps.RawArtifactReadKind(artifact)
			}
			summaries = append(summaries, SummarizeRawArtifact(artifact, readKind))
		}
		add(ObjectRawArtifact, summaries)
	}
	if legacy && (objectKind == "" || isRecordKind(objectKind)) {
		if err := s.addRecords(ctx, missionID, objectKind, add); err != nil {
			return nil, err
		}
	}
	if legacy && (objectKind == "" || objectKind == ObjectReport || objectKind == ObjectReportVersion || objectKind == ObjectReportBlock) {
		if err := s.addReports(ctx, missionID, objectKind, add); err != nil {
			return nil, err
		}
	}
	if objectKind == "" || objectKind == ObjectLedgerEvent {
		events, err := s.deps.ListVisibleLedgerEvents(ctx, missionID, legacy)
		if err != nil {
			return nil, err
		}
		var summaries []ObjectSummary
		for _, event := range events {
			summaries = append(summaries, SummarizeLedgerEvent(event))
		}
		add(ObjectLedgerEvent, summaries)
	}
	return items, nil
}

func (s *Service) addRecords(ctx context.Context, missionID, objectKind string, add func(string, []ObjectSummary)) error {
	evidence, err := s.deps.ListEvidenceRecords(ctx, missionID)
	if err != nil {
		return err
	}
	claims, err := s.deps.ListClaimRecords(ctx, missionID)
	if err != nil {
		return err
	}
	questions, err := s.deps.ListQuestionRecords(ctx, missionID)
	if err != nil {
		return err
	}
	options, err := s.deps.ListOptionRecords(ctx, missionID)
	if err != nil {
		return err
	}
	proposals, err := s.deps.ListProposalSummaries(ctx, missionID)
	if err != nil {
		return err
	}
	var summaries []ObjectSummary
	for _, record := range evidence {
		summaries = append(summaries, SummarizeEvidence(record))
	}
	add(ObjectEvidenceRecord, summaries)
	summaries = nil
	for _, record := range claims {
		summaries = append(summaries, SummarizeClaim(record))
	}
	add(ObjectClaimRecord, summaries)
	summaries = nil
	for _, record := range questions {
		summaries = append(summaries, SummarizeQuestion(record))
	}
	add(ObjectQuestionRecord, summaries)
	summaries = nil
	for _, record := range options {
		summaries = append(summaries, SummarizeOption(record))
	}
	add(ObjectOptionRecord, summaries)
	add(ObjectProposalBundle, proposals)
	return nil
}

func (s *Service) addReports(ctx context.Context, missionID, objectKind string, add func(string, []ObjectSummary)) error {
	reports, err := s.deps.ListReportSummaries(ctx, missionID)
	if err != nil {
		return err
	}
	versions, err := s.deps.ListReportVersionSummaries(ctx, missionID)
	if err != nil {
		return err
	}
	add(ObjectReport, reports)
	add(ObjectReportVersion, versions)
	var blocks []ObjectSummary
	for _, version := range versions {
		versionID := version.ObjectID
		loaded, err := s.deps.ListReportBlockSummaries(ctx, versionID)
		if err != nil {
			return err
		}
		blocks = append(blocks, loaded...)
	}
	add(ObjectReportBlock, blocks)
	return nil
}

func (s *Service) filterRefs(refs []ObjectRef, hidden map[string]struct{}) []ObjectRef {
	if s.deps.FilterReportArtifactRefs == nil {
		return refs
	}
	return s.deps.FilterReportArtifactRefs(refs, hidden)
}
func (s *Service) filterSummaries(summaries []ObjectSummary, hidden map[string]struct{}) []ObjectSummary {
	if s.deps.FilterReportArtifactSummaryRefs == nil {
		return summaries
	}
	return s.deps.FilterReportArtifactSummaryRefs(summaries, hidden)
}
func validateMissionID(id string) error {
	if !strings.HasPrefix(id, "mis_") || len(id) <= 4 {
		return fmt.Errorf("%w: id must start with mis_", producterror.ErrInvalidInput)
	}
	return nil
}
func VisibleLedgerEvents(events []ledger.Event, legacy bool) []ledger.Event {
	if legacy {
		return events
	}
	out := make([]ledger.Event, 0, len(events))
	for _, event := range events {
		if ReportLedgerEvent(event) {
			continue
		}
		out = append(out, event)
	}
	return out
}
func ReportArtifactIDs(events []ledger.Event) map[string]struct{} {
	out := map[string]struct{}{}
	for _, event := range events {
		if !ReportLedgerEvent(event) {
			continue
		}
		var p struct {
			ArtifactID string `json:"artifact_id"`
		}
		if json.Unmarshal(event.Payload, &p) == nil && strings.TrimSpace(p.ArtifactID) != "" {
			out[strings.TrimSpace(p.ArtifactID)] = struct{}{}
		}
	}
	return out
}
func lastSequence(events []ledger.Event) int64 {
	var last int64
	for _, event := range events {
		if event.Sequence > last {
			last = event.Sequence
		}
	}
	return last
}
func isRecordKind(kind string) bool {
	return kind == ObjectEvidenceRecord || kind == ObjectClaimRecord || kind == ObjectQuestionRecord || kind == ObjectOptionRecord || kind == ObjectProposalBundle
}

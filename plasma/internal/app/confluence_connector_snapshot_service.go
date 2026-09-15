package app

import (
	"context"
	"encoding/json"
	"fmt"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/source"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
	"strings"
)

// SnapshotConfluenceSource는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) SnapshotConfluenceSource(
	ctx context.Context,
	connector confluencesource.ConfluenceSourceConnector,
	req SnapshotConfluenceSourceRequest,
) (ConfluenceSnapshotResult, error) {
	artifact, snapshot, err := s.buildConfluenceSnapshot(ctx, connector, req)
	if err != nil {
		return ConfluenceSnapshotResult{}, err
	}
	if err := s.store.CreateRawArtifact(ctx, artifact); err != nil {
		return ConfluenceSnapshotResult{}, err
	}
	if err := s.store.CreateSourceSnapshot(ctx, snapshot); err != nil {
		return ConfluenceSnapshotResult{}, err
	}
	return ConfluenceSnapshotResult{Artifact: artifact, Snapshot: snapshot}, nil
}

// SnapshotConfluenceSourceWithEvent는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) SnapshotConfluenceSourceWithEvent(
	ctx context.Context,
	connector confluencesource.ConfluenceSourceConnector,
	req SnapshotConfluenceSourceWithEventRequest,
) (ConfluenceSnapshotWithEventResult, error) {
	artifact, snapshot, err := s.buildConfluenceSnapshot(ctx, connector, req.Snapshot)
	if err != nil {
		return ConfluenceSnapshotWithEventResult{}, err
	}
	payload := sourceevents.BuildConnectorSourceSnapshottedPayload(sourceevents.ConnectorSourceSnapshottedPayloadRequest{
		SnapshotID:  snapshot.SnapshotID,
		ArtifactIDs: snapshot.ArtifactIDs,
		Connector:   sourceEventConnectorRef(snapshot.Connector),
		Reason:      req.Snapshot.Reason,
	})
	var payloadFields map[string]any
	if err := json.Unmarshal(payload, &payloadFields); err != nil {
		return ConfluenceSnapshotWithEventResult{}, err
	}
	if proposalID := strings.TrimSpace(req.SourceCandidateProposalEventID); proposalID != "" {
		payloadFields["source_candidate_proposal_event_id"] = proposalID
	}
	if candidateURL := strings.TrimSpace(req.SourceCandidateURL); candidateURL != "" {
		payloadFields["url"] = candidateURL
	}
	payload, err = json.Marshal(payloadFields)
	if err != nil {
		return ConfluenceSnapshotWithEventResult{}, err
	}
	event, err := buildLedgerEvent(ledger.AppendRequest{
		EventID:   req.EventID,
		MissionID: snapshot.MissionID,
		EventType: sourceevents.SourceSnapshottedEventType,
		Producer:  req.Producer,
		Payload:   payload,
	})
	if err != nil {
		return ConfluenceSnapshotWithEventResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []ledger.Event{event},
		RawArtifacts:    []artifactcontract.Raw{artifact},
		SourceSnapshots: []sourcecontract.Snapshot{snapshot},
	})
	if err != nil {
		return ConfluenceSnapshotWithEventResult{}, err
	}
	return ConfluenceSnapshotWithEventResult{Artifact: artifact, Snapshot: snapshot, Event: committed.Events[0]}, nil
}

func (s *Service) buildConfluenceSnapshot(
	ctx context.Context,
	connector confluencesource.ConfluenceSourceConnector,
	req SnapshotConfluenceSourceRequest,
) (artifactcontract.Raw, sourcecontract.Snapshot, error) {
	if connector == nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, fmt.Errorf("%w: confluence connector is required", ErrInvalidInput)
	}
	missionID := strings.TrimSpace(req.MissionID)
	artifactID := strings.TrimSpace(req.ArtifactID)
	snapshotID := strings.TrimSpace(req.SnapshotID)
	cloudID := strings.TrimSpace(req.CloudID)
	pageID := strings.TrimSpace(req.PageID)
	if err := validateID("mis_", missionID); err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	if err := validateID("art_", artifactID); err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	if err := validateID("src_", snapshotID); err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	if cloudID == "" || pageID == "" {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, fmt.Errorf("%w: confluence cloud id and page id are required", ErrInvalidInput)
	}
	if err := validateConfluenceExpectedVersion(req.ExpectedVersion); err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}

	page, err := connector.ReadConfluenceSource(ctx, confluencesource.ConfluenceSourceReadRequest{CloudID: cloudID, PageID: pageID})
	if err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	if page.Title = strings.TrimSpace(page.Title); page.Title == "" {
		page.Title = strings.TrimSpace(req.Title)
	}
	page, err = confluencesource.NormalizePage(page, cloudID, pageID, req.ExpectedVersion)
	if err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	if confluencesource.RangeSelected(req.Range) {
		if err := confluencesource.ValidateSelectedRange(page, req.Range, req.MaxBodyBytes); err != nil {
			return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
		}
	} else {
		if err := confluencesource.ValidateSnapshotBodySize(page, req.MaxBodyBytes); err != nil {
			return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
		}
	}
	content, locators, err := confluencesource.BuildSnapshotPayload(page, artifactID, strings.TrimSpace(req.Reason), req.Range)
	if err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	producer, err := confluenceSnapshotProducer(req.Producer)
	if err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	artifact, err := artifactcontract.Build(artifactcontract.CreateRequest{
		ArtifactID: artifactID,
		MissionID:  missionID,
		MediaType:  confluencesource.ConfluenceSnapshotMediaType,
		Filename:   confluenceSnapshotFilename(page.Connector.ExternalSourceID),
		Producer:   producer,
		Content:    content,
	})
	if err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	title := page.Title
	if confluencesource.RangeSelected(req.Range) {
		title = fmt.Sprintf("%s (Confluence range %d-%d)", page.Title, req.Range.Start, req.Range.End)
	}
	snapshot, err := source.BuildSnapshot(ctx, s.store, source.CreateRequest{
		SnapshotID:        snapshotID,
		MissionID:         missionID,
		Connector:         page.Connector,
		Title:             title,
		ExternalUpdatedAt: page.UpdatedAt,
		ArtifactIDs:       []string{artifact.ArtifactID},
		ContentHash:       req.ExpectedContentHash,
		Locators:          locators,
	}, []artifactcontract.Raw{artifact})
	if err != nil {
		return artifactcontract.Raw{}, sourcecontract.Snapshot{}, err
	}
	return artifact, snapshot, nil
}

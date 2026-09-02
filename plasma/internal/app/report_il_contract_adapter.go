package app

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

func (s *Service) CreateReportILBundleIfOpenContract(ctx context.Context, req reportilcontract.BundleRequest) (reportilcontract.BundleResult, error) {
	artifacts := make([]CreateRawArtifactRequest, 0, len(req.Artifacts))
	for _, item := range req.Artifacts {
		artifacts = append(artifacts, CreateRawArtifactRequest{ArtifactID: item.ArtifactID, MissionID: item.MissionID, MediaType: item.MediaType, Filename: item.Filename, Producer: item.Producer, Content: item.Content, ExpectedSHA256: item.ExpectedSHA256})
	}
	store := AppendEventRequest{EventID: req.StoreCompleted.EventID, MissionID: req.StoreCompleted.MissionID, EventType: req.StoreCompleted.EventType, CausationEventID: req.StoreCompleted.CausationEventID, CorrelationID: req.StoreCompleted.CorrelationID, Producer: req.StoreCompleted.Producer, Payload: req.StoreCompleted.Payload}
	terminal := AppendEventRequest{EventID: req.Terminal.EventID, MissionID: req.Terminal.MissionID, EventType: req.Terminal.EventType, CausationEventID: req.Terminal.CausationEventID, CorrelationID: req.Terminal.CorrelationID, Producer: req.Terminal.Producer, Payload: req.Terminal.Payload}
	items, event, created, err := s.CreateReportILBundleIfOpen(ctx, ReportILBundleRequest{MissionID: req.MissionID, PendingID: req.PendingID, Artifacts: artifacts, StoreCompleted: store, Terminal: terminal})
	if err != nil {
		return reportilcontract.BundleResult{}, err
	}
	out := make([]reportilcontract.Artifact, 0, len(items))
	for _, item := range items {
		out = append(out, reportilcontract.Artifact{ArtifactID: item.ArtifactID, MissionID: item.MissionID, MediaType: item.MediaType, ByteSize: item.ByteSize, SHA256: item.SHA256, Content: item.Content})
	}
	return reportilcontract.BundleResult{Artifacts: out, Terminal: event, Created: created}, nil
}

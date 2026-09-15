package app

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/source"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

type reportILSourceAdapter struct{ service *Service }

func (s *Service) ReportILSourceReader() reportilcontract.SourceReader {
	return reportILSourceAdapter{service: s}
}

func (r reportILSourceAdapter) ListSourceSnapshots(ctx context.Context, missionID string) ([]reportilcontract.SourceSnapshot, error) {
	items, err := r.service.ListSourceSnapshotsWithState(ctx, source.ListRequest{MissionID: missionID})
	if err != nil {
		return nil, err
	}
	out := make([]reportilcontract.SourceSnapshot, 0, len(items))
	for _, item := range items {
		out = append(out, reportilcontract.SourceSnapshot{SnapshotID: item.SnapshotID, MissionID: item.MissionID, Title: item.Title, ArtifactIDs: append([]string(nil), item.ArtifactIDs...), ContentHash: item.ContentHash.Value, RetrievalPolicy: item.Access.RetrievalPolicy, ConnectorType: item.Connector.ConnectorType, ExternalURI: item.Connector.ExternalURI, Locators: append([]byte(nil), item.Locators...), Active: !item.State.Removed && !item.State.Superseded && (item.State.State == "" || item.State.State == source.StateActive)})
	}
	return out, nil
}

func (r reportILSourceAdapter) GetArtifact(ctx context.Context, artifactID string) (reportilcontract.Artifact, error) {
	item, err := r.service.GetRawArtifact(ctx, artifactID)
	if err != nil {
		return reportilcontract.Artifact{}, err
	}
	return reportilcontract.Artifact{ArtifactID: item.ArtifactID, MissionID: item.MissionID, MediaType: item.MediaType, Filename: item.Filename, ByteSize: item.ByteSize, SHA256: item.SHA256, Content: append([]byte(nil), item.Content...)}, nil
}

func (r reportILSourceAdapter) ReadLive(ctx context.Context, missionID, snapshotID string, maxBytes int64) (reportilcontract.LocalRead, error) {
	item, err := r.service.ReadLocalPathSource(ctx, ReadLocalPathSourceRequest{MissionID: missionID, SnapshotID: snapshotID, MaxBytes: maxBytes, Producer: ledger.Producer{Type: "report_il", ID: "source-catalog"}})
	if err != nil {
		return reportilcontract.LocalRead{}, err
	}
	receipt := ""
	if item.ObservationEvent != nil {
		receipt = item.ObservationEvent.EventID
	}
	return reportilcontract.LocalRead{Content: item.Read.Content, Binary: item.Read.Metadata.Binary, Truncated: item.Read.Metadata.Truncated, Size: item.Read.Metadata.Size, MTime: item.Read.Metadata.MTime, SHA256: item.Read.Metadata.SHA256, ObservationReceipt: receipt, Extraction: item.Read.Metadata.Extraction}, nil
}

func (s *Service) AppendReportILProgress(ctx context.Context, missionID, pendingID, stage, status string) error {
	_, err := s.AppendEvent(ctx, ledger.AppendRequest{EventID: newAppID("evt"), MissionID: missionID, EventType: "report." + stage + "." + status, Producer: ledger.Producer{Type: "system", ID: "report-il"}, Payload: mustMarshalJSON(map[string]any{"kind": "report_il_stage_progress", "pending_event_id": pendingID, "pipeline_family": reportilcontract.PipelineFamily, "stage": stage, "status": status})})
	return err
}

func (s *Service) AppendReportILLongFormProgress(
	ctx context.Context,
	missionID,
	pendingID,
	kind,
	status string,
	plan *reportilcontract.LongFormPlan,
	partIndex,
	sectionIndex int,
	title string,
) error {
	kind = strings.TrimSpace(kind)
	status = strings.TrimSpace(status)
	eventType := ""
	switch kind {
	case "plan":
		if status != "completed" || plan == nil {
			return fmt.Errorf("report IL long-form plan progress is invalid")
		}
		eventType = "report.il_long_form_plan.created"
	case "section":
		if (status != "started" && status != "completed" && status != "failed") || partIndex < 1 || sectionIndex < 1 {
			return fmt.Errorf("report IL long-form Section progress is invalid")
		}
		eventType = "report.il_long_form_section." + status
	case "part_edit":
		if (status != "started" && status != "completed" && status != "failed") || partIndex < 1 || sectionIndex != 0 {
			return fmt.Errorf("report IL long-form Part progress is invalid")
		}
		eventType = "report.il_long_form_part." + status
	default:
		return fmt.Errorf("report IL long-form progress kind is invalid")
	}
	payload := map[string]any{
		"kind":             "report_il_long_form_progress",
		"pending_event_id": pendingID,
		"pipeline_family":  reportilcontract.PipelineFamily,
		"stage_kind":       kind,
		"status":           status,
	}
	if title = strings.TrimSpace(title); title != "" {
		payload["title"] = title
	}
	if partIndex > 0 {
		payload["part_index"] = partIndex
	}
	if sectionIndex > 0 {
		payload["section_index"] = sectionIndex
	}
	if plan != nil {
		parts := make([]map[string]any, 0, len(plan.Parts))
		for _, part := range plan.Parts {
			sections := make([]map[string]any, 0, len(part.Sections))
			for _, section := range part.Sections {
				sections = append(sections, map[string]any{"title": section.Title})
			}
			parts = append(parts, map[string]any{"title": part.Title, "sections": sections})
		}
		payload["plan"] = map[string]any{"parts": parts}
	}
	_, err := s.AppendEvent(ctx, ledger.AppendRequest{
		EventID: newAppID("evt"), MissionID: missionID, EventType: eventType,
		Producer:         ledger.Producer{Type: "system", ID: "report-il"},
		CausationEventID: pendingID, CorrelationID: pendingID,
		Payload: mustMarshalJSON(payload),
	})
	return err
}

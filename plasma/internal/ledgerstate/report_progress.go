package ledgerstate

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
)

// ReportProgress는 장부에서 보수적으로 도출한 report work view다.
// Pending event는 지속적인 at-least-once visibility 경계다. terminal write가 실패하면
// pending 상태가 남고 projection도 보수적으로 계산한다. terminal-write-pending
// outbox는 의도적으로 후속 작업이며, 이 projection의 책임이 아니다.
type ReportProgress struct {
	AttemptID string                `json:"attempt_id,omitempty"`
	OriginID  string                `json:"origin_pending_event_id,omitempty"`
	Attempt   int                   `json:"attempt_number"`
	State     string                `json:"state"`
	Nodes     []ReportProgressNode  `json:"nodes"`
	Retry     ReportRetryCapability `json:"retry"`
}

// ReportProgressNode는 report pipeline 그래프의 단일 stage 표시 상태다.
type ReportProgressNode struct {
	ID         string     `json:"id"`
	Kind       string     `json:"kind"`
	Part       int        `json:"part_index,omitempty"`
	Section    int        `json:"section_index,omitempty"`
	State      string     `json:"state"`
	AttemptID  string     `json:"provenance_attempt_id,omitempty"`
	Error      string     `json:"error,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	DurationMS *int64     `json:"duration_ms,omitempty"`
}

// ReportRetryCapability는 실패한 report 작업에 대해 사용자가 선택할 수 있는 재시도 범위다.
type ReportRetryCapability struct {
	ResumeFailed bool   `json:"resume_failed"`
	Restart      bool   `json:"restart"`
	ReasonCode   string `json:"reason_code,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

type reportPayload struct {
	PendingID          string `json:"pending_event_id"`
	OriginID           string `json:"origin_pending_event_id"`
	RetryOf            string `json:"retry_of_pending_event_id"`
	RetryStrategy      string `json:"retry_strategy"`
	Attempt            int    `json:"attempt_number"`
	ReportMode         string `json:"report_mode"`
	PipelineFamily     string `json:"pipeline_family"`
	PipelineGraph      string `json:"pipeline_graph"`
	RigorLevel         string `json:"rigor_level"`
	Part               int    `json:"part_index"`
	Section            int    `json:"section_index"`
	FailedStage        string `json:"failed_stage_kind"`
	FailedStageID      string `json:"failed_stage_id"`
	Error              string `json:"safe_error_message"`
	Retryable          bool   `json:"retryable"`
	Kind               string `json:"kind"`
	PartEdit           bool   `json:"part_edit_enabled"`
	PartPlanning       bool   `json:"part_planning_enabled"`
	FinalEditPipeline  string `json:"final_edit_pipeline"`
	PostReportHumanize string `json:"post_report_humanize"`
	Plan               struct {
		Parts []struct {
			Sections []json.RawMessage `json:"sections"`
		} `json:"parts"`
	} `json:"plan"`
}

const (
	finalEditPipelineReaderStyleGateV1                                 = "reader_style_gate_v1"
	finalEditPipelineAssemblyWriterReaderStyleGateV2                   = "assembly_writer_reader_style_gate_v2"
	finalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 = "assembly_writer_reader_style_validation_evidence_gate_v3"
)

// ProjectReportProgress는 legacy report event를 정규화하되, 대응하는 장부 이벤트 없이
// 완료된 stage를 추론하지 않는다.
func ProjectReportProgress(events []Event) ReportProgress {
	pending := map[string]Event{}
	pendingTypes := map[string]string{}
	payloads := map[string]reportPayload{}
	terminal := map[string]reportPayload{}
	terminalEvents := map[string]Event{}
	terminalCount := map[string]int{}
	failedTerminal := map[string]bool{}
	canceledTerminal := map[string]bool{}
	for _, e := range events {
		var p reportPayload
		_ = json.Unmarshal(e.Payload, &p)
		if e.EventType == "report.drafted" && p.PendingID == "" {
			var legacy struct {
				Generation struct {
					PendingID string `json:"pending_event_id"`
				} `json:"generation"`
			}
			_ = json.Unmarshal(e.Payload, &legacy)
			p.PendingID = legacy.Generation.PendingID
		}
		switch e.EventType {
		case "report.draft.pending", "report.design.pending", "report.humanize.pending", "report.patch.pending":
			pending[e.EventID], payloads[e.EventID] = e, p
			pendingTypes[e.EventID] = e.EventType
		case "report.source_packet.started", "report.source_packet.completed", "report.il_source_selection.started", "report.il_source_selection.completed", "report.il_editorial_memory.started", "report.il_editorial_memory.completed", "report.il_narrative.started", "report.il_narrative.completed", "report.il_long_form_plan.started", "report.il_long_form_plan.completed", "report.il_long_form_sections.started", "report.il_long_form_sections.completed", "report.il_long_form_parts.started", "report.il_long_form_parts.completed", "report.il_long_form_final.started", "report.il_long_form_final.completed", "report.il_continuity.started", "report.il_continuity.completed", "report.il_reader.started", "report.il_reader.completed", "report.il_images.started", "report.il_images.completed", "report.il_document.started", "report.il_document.completed", "report.il_flow.started", "report.il_flow.completed", "report.il_render.started", "report.il_render.completed", "report.il_store.started", "report.il_store.completed", "report.il_long_form_plan.created", "report.il_long_form_section.started", "report.il_long_form_section.completed", "report.il_long_form_part.started", "report.il_long_form_part.completed":
			// Progress is telemetry. Preserve the pending metadata and do not replace it with sparse stage payloads.
		case "report.source_packet.failed", "report.il_source_selection.failed", "report.il_editorial_memory.failed", "report.il_narrative.failed", "report.il_long_form_plan.failed", "report.il_long_form_sections.failed", "report.il_long_form_parts.failed", "report.il_long_form_final.failed", "report.il_continuity.failed", "report.il_reader.failed", "report.il_images.failed", "report.il_document.failed", "report.il_flow.failed", "report.il_render.failed", "report.il_store.failed", "report.il_long_form_section.failed", "report.il_long_form_part.failed":
			// Typed IL/source failures are stage telemetry; report.draft.failed closes the attempt.
		case "report.draft.failed", "report.design.failed", "report.humanize.failed", "report.patch.failed":
			if p.PendingID != "" {
				terminal[p.PendingID] = p
				terminalEvents[p.PendingID] = e
				terminalCount[p.PendingID]++
				if strings.HasSuffix(p.Kind, "_canceled") {
					canceledTerminal[p.PendingID] = true
				} else {
					failedTerminal[p.PendingID] = true
				}
			}
		case "report.humanize.skipped":
			if p.PendingID != "" {
				terminal[p.PendingID] = p
				terminalEvents[p.PendingID] = e
				terminalCount[p.PendingID]++
				canceledTerminal[p.PendingID] = true
			}
		case "report.drafted", "report.artifact.created", "report.artifact.exported":
			if p.PendingID != "" {
				terminal[p.PendingID] = p
				terminalEvents[p.PendingID] = e
				terminalCount[p.PendingID]++
			}
		}
	}
	var selected string
	for _, e := range events {
		if _, ok := pending[e.EventID]; ok {
			selected = e.EventID
		}
	}
	if selected == "" {
		return ReportProgress{State: "unknown", Retry: ReportRetryCapability{ReasonCode: "no_report_attempt", Reason: "리포트 시도가 없습니다."}}
	}
	p := payloads[selected]
	if terminalCount[selected] > 1 {
		return unknownReportProgress()
	}
	origin := strings.TrimSpace(p.OriginID)
	if origin == "" {
		origin = selected
	}
	attempt := p.Attempt
	if attempt < 1 {
		attempt = 1
	}
	result := ReportProgress{AttemptID: selected, OriginID: origin, Attempt: attempt, State: "running"}
	if _, done := terminal[selected]; done && !failedTerminal[selected] {
		result.State = "completed"
	}
	if canceledTerminal[selected] {
		result.State = "skipped"
	}
	// plan에서 실제 graph shape를 만든 다음, 실제 이벤트만 적용한다.
	nodes := []ReportProgressNode{{ID: "plan", Kind: "plan", State: "pending"}}
	if p.PipelineFamily == "report_il_experimental" && pendingTypes[selected] == "report.draft.pending" {
		stageKinds := []string{"source_packet"}
		if reportILSelectionStageExists(events, selected, terminal[selected]) {
			stageKinds = append(stageKinds, "il_source_selection")
		}
		editorialMemoryStageExists := reportILStageExists(events, selected, terminal[selected], "il_editorial_memory")
		if p.PipelineGraph == reportpipeline.ExperimentalILValidationProfilesGraph {
			if p.RigorLevel != "unverified" || editorialMemoryStageExists {
				stageKinds = append(stageKinds, "il_editorial_memory")
			}
		} else if p.PipelineGraph == reportpipeline.ExperimentalILEditorialMemoryGraph || editorialMemoryStageExists {
			stageKinds = append(stageKinds, "il_editorial_memory")
		}
		if p.ReportMode == "long_form" {
			stageKinds = append(stageKinds,
				"il_narrative", "il_long_form_plan", "il_long_form_sections",
				"il_long_form_parts", "il_long_form_final",
			)
		} else {
			stageKinds = append(stageKinds, "il_narrative")
		}
		continuityStageExists := reportILStageExists(events, selected, terminal[selected], "il_continuity")
		readerStageExists := reportILStageExists(events, selected, terminal[selected], "il_reader")
		flowStageExists := reportILStageExists(events, selected, terminal[selected], "il_flow")
		switch {
		case p.PipelineGraph == reportpipeline.ExperimentalILValidationProfilesGraph:
			if p.RigorLevel != "unverified" || readerStageExists {
				stageKinds = append(stageKinds, "il_reader")
			}
			if p.RigorLevel == "strict" || continuityStageExists {
				stageKinds = append(stageKinds, "il_continuity")
			}
			stageKinds = append(stageKinds, "il_images", "il_document")
		case p.PipelineGraph == reportpipeline.ExperimentalILEditorialMemoryGraph || p.PipelineGraph == reportpipeline.ExperimentalILEditorialGraph || continuityStageExists:
			stageKinds = append(stageKinds, "il_reader", "il_continuity", "il_images", "il_document")
		case p.PipelineGraph == reportpipeline.ExperimentalILReaderGraph || readerStageExists:
			stageKinds = append(stageKinds, "il_reader", "il_document")
		case p.PipelineGraph == "report_il_flow_v1" || flowStageExists || strings.TrimSpace(p.PipelineGraph) == "":
			stageKinds = append(stageKinds, "il_document", "il_flow")
		default:
			stageKinds = append(stageKinds, "il_document")
		}
		stageKinds = append(stageKinds, "il_render", "il_store")
		for _, kind := range stageKinds {
			nodes = append(nodes, ReportProgressNode{ID: kind, Kind: kind, State: "pending"})
		}
	}
	lineage := map[string]bool{}
	lineageValid := false
	for current, depth := selected, 0; current != "" && depth < 64; depth++ {
		if lineage[current] {
			return unknownReportProgress()
		}
		lineage[current] = true
		item, ok := payloads[current]
		if !ok {
			return unknownReportProgress()
		}
		if item.OriginID != "" && item.OriginID != origin {
			return unknownReportProgress()
		}
		if item.RetryStrategy == "restart" {
			// Restart는 직접 실패한 parent를 갖지만 의도적으로 재사용하지 않는다.
			if item.RetryOf == "" {
				return unknownReportProgress()
			}
			parent, ok := payloads[item.RetryOf]
			if !ok || parent.OriginID != "" && parent.OriginID != origin {
				return unknownReportProgress()
			}
			lineageValid = true
			break
		}
		if item.RetryOf == "" {
			if current != origin {
				return unknownReportProgress()
			}
			lineageValid = true
			break
		}
		current = strings.TrimSpace(item.RetryOf)
	}
	if !lineageValid {
		return unknownReportProgress()
	}
	partPlanningEnabled := false
	for _, event := range events {
		var payload reportPayload
		_ = json.Unmarshal(event.Payload, &payload)
		if lineage[payload.PendingID] && event.EventType == "report.plan.created" && payload.PartPlanning {
			partPlanningEnabled = true
			break
		}
	}
	partCount := 0
	ilLongFormPlanSeen := false
	for _, e := range events {
		var q reportPayload
		_ = json.Unmarshal(e.Payload, &q)
		if !lineage[q.PendingID] {
			continue
		}
		if e.EventType == "report.il_long_form_plan.created" {
			if p.PipelineFamily != reportpipeline.ExperimentalIL || p.ReportMode != "long_form" || ilLongFormPlanSeen || len(q.Plan.Parts) == 0 {
				return unknownReportProgress()
			}
			ilLongFormPlanSeen = true
			for i, part := range q.Plan.Parts {
				if len(part.Sections) == 0 {
					return unknownReportProgress()
				}
				for j := range part.Sections {
					nodes = append(nodes, ReportProgressNode{ID: stageID("section", i+1, j+1), Kind: "section", Part: i + 1, Section: j + 1, State: "pending"})
				}
			}
			for i := range q.Plan.Parts {
				nodes = append(nodes, ReportProgressNode{ID: stageID("part_edit", i+1, 0), Kind: "part_edit", Part: i + 1, State: "pending"})
			}
			continue
		}
		if e.EventType == "report.plan.created" {
			nodes[0].State = "completed"
			nodes[0].AttemptID = q.PendingID
			partCount = len(q.Plan.Parts)
			if partPlanningEnabled {
				for i := range q.Plan.Parts {
					nodes = append(nodes, ReportProgressNode{ID: stageID("part_plan", i+1, 0), Kind: "part_plan", Part: i + 1, State: "pending"})
				}
			}
			for i, part := range q.Plan.Parts {
				for j := range part.Sections {
					nodes = append(nodes, ReportProgressNode{ID: stageID("section", i+1, j+1), Kind: "section", Part: i + 1, Section: j + 1, State: "pending"})
				}
			}
			for i := range q.Plan.Parts {
				nodes = append(nodes, ReportProgressNode{ID: stageID("part", i+1, 0), Kind: "part", Part: i + 1, State: "pending"})
			}
			if q.PartEdit {
				for i := range q.Plan.Parts {
					kind := "part_edit"
					if partPlanningEnabled {
						kind = "part_author"
					}
					nodes = append(nodes, ReportProgressNode{ID: stageID(kind, i+1, 0), Kind: kind, Part: i + 1, State: "pending"})
				}
			}
			switch q.FinalEditPipeline {
			case finalEditPipelineReaderStyleGateV1:
				nodes = append(nodes, ReportProgressNode{ID: stageID("reader_edit", 0, 0), Kind: "reader_edit", State: "pending"})
				if strings.TrimSpace(q.PostReportHumanize) == "enabled" {
					nodes = append(nodes, ReportProgressNode{ID: stageID("style_edit", 0, 0), Kind: "style_edit", State: "pending"})
				}
				nodes = append(nodes, ReportProgressNode{ID: stageID("corrective_gate", 0, 0), Kind: "corrective_gate", State: "pending"})
			case finalEditPipelineAssemblyWriterReaderStyleGateV2:
				nodes = append(nodes, ReportProgressNode{ID: stageID("final_assembly", 0, 0), Kind: "final_assembly", State: "pending"})
				nodes = append(nodes, ReportProgressNode{ID: stageID("final_write", 0, 0), Kind: "final_write", State: "pending"})
				nodes = append(nodes, ReportProgressNode{ID: stageID("reader_edit", 0, 0), Kind: "reader_edit", State: "pending"})
				if strings.TrimSpace(q.PostReportHumanize) == "enabled" {
					nodes = append(nodes, ReportProgressNode{ID: stageID("style_edit", 0, 0), Kind: "style_edit", State: "pending"})
				}
				nodes = append(nodes, ReportProgressNode{ID: stageID("corrective_gate", 0, 0), Kind: "corrective_gate", State: "pending"})
			case finalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3:
				nodes = append(nodes, ReportProgressNode{ID: stageID("final_assembly", 0, 0), Kind: "final_assembly", State: "pending"})
				nodes = append(nodes, ReportProgressNode{ID: stageID("final_write", 0, 0), Kind: "final_write", State: "pending"})
				nodes = append(nodes, ReportProgressNode{ID: stageID("reader_edit", 0, 0), Kind: "reader_edit", State: "pending"})
				if strings.TrimSpace(q.PostReportHumanize) == "enabled" {
					nodes = append(nodes, ReportProgressNode{ID: stageID("style_edit", 0, 0), Kind: "style_edit", State: "pending"})
					nodes = append(nodes, ReportProgressNode{ID: stageID("style_semantic_validation", 0, 0), Kind: "style_semantic_validation", State: "pending"})
				}
				nodes = append(nodes, ReportProgressNode{ID: stageID("evidence_gate", 0, 0), Kind: "evidence_gate", State: "pending"})
			}
		}
	}
	if pendingTypes[selected] != "report.draft.pending" || p.ReportMode != "long_form" ||
		p.PipelineFamily == reportpipeline.ExperimentalIL {
		// non-sectional 작업은 preparation 경계 하나 뒤에 finalization/artifact가 이어진다.
		nodes[0].ID, nodes[0].Kind = "start", "start"
		nodes[0].State = "completed"
	} else if partCount == 0 { // legacy / malformed plans: still provide final nodes safely.
		nodes[0].State = "unknown"
	}
	nodes = append(nodes, ReportProgressNode{ID: "final", Kind: "final", State: "pending"}, ReportProgressNode{ID: "artifact", Kind: "artifact", State: "pending"})
	index := map[string]int{}
	for i := range nodes {
		index[nodes[i].ID] = i
	}
	for _, e := range events {
		var q reportPayload
		_ = json.Unmarshal(e.Payload, &q)
		if !lineage[q.PendingID] {
			continue
		}
		id := ""
		switch e.EventType {
		case "report.part_plan.created":
			id = stageID("part_plan", q.Part, 0)
		case "report.il_long_form_plan.created":
			continue
		case "report.il_long_form_section.started":
			id = stageID("section", q.Part, q.Section)
			if i, ok := index[id]; ok && nodes[i].State != "completed" && nodes[i].State != "failed" {
				nodes[i].AttemptID = q.PendingID
				nodes[i].State = "running"
			} else {
				return unknownReportProgress()
			}
			continue
		case "report.il_long_form_section.completed":
			id = stageID("section", q.Part, q.Section)
			if _, ok := index[id]; !ok {
				return unknownReportProgress()
			}
		case "report.il_long_form_part.started":
			id = stageID("part_edit", q.Part, 0)
			if i, ok := index[id]; ok && nodes[i].State != "completed" && nodes[i].State != "failed" {
				nodes[i].AttemptID = q.PendingID
				nodes[i].State = "running"
			} else {
				return unknownReportProgress()
			}
			continue
		case "report.il_long_form_part.completed":
			id = stageID("part_edit", q.Part, 0)
			if _, ok := index[id]; !ok {
				return unknownReportProgress()
			}
		case "report.section.started":
			id = stageID("section", q.Part, q.Section)
			if i, ok := index[id]; ok && nodes[i].State != "completed" && nodes[i].AttemptID != q.PendingID {
				nodes[i].AttemptID = q.PendingID
				nodes[i].State = "running"
			}
			continue
		case "report.section.evidence_gap":
			id = stageID("section", q.Part, q.Section)
			if i, ok := index[id]; ok && nodes[i].State != "completed" && nodes[i].State != "failed" {
				nodes[i].AttemptID = q.PendingID
				nodes[i].State = "running"
			}
			continue
		case "report.section.created":
			id = stageID("section", q.Part, q.Section)
		case "report.part.created":
			id = stageID("part", q.Part, 0)
		case "report.part_edit.started":
			id = stageID(reportPartEditProgressKind(partPlanningEnabled), q.Part, 0)
			if i, ok := index[id]; ok && nodes[i].State == "pending" {
				nodes[i].AttemptID = q.PendingID
				nodes[i].State = "running"
			}
			continue
		case "report.part.edited":
			id = stageID(reportPartEditProgressKind(partPlanningEnabled), q.Part, 0)
		case "report.final_assembly.created":
			id = stageID("final_assembly", 0, 0)
		case "report.final_edit.writer.started":
			id = stageID("final_write", 0, 0)
			if i, ok := index[id]; ok && nodes[i].State == "pending" {
				nodes[i].AttemptID = q.PendingID
				nodes[i].State = "running"
			}
			continue
		case "report.final_edit.writer.submitted":
			id = stageID("final_write", 0, 0)
		case "report.final_edit.reader.started":
			id = stageID("reader_edit", 0, 0)
			if i, ok := index[id]; ok && nodes[i].State == "pending" {
				nodes[i].AttemptID = q.PendingID
				nodes[i].State = "running"
			}
			continue
		case "report.final_edit.reader.submitted":
			id = stageID("reader_edit", 0, 0)
		case "report.final_edit.style.started":
			id = stageID("style_edit", 0, 0)
			if i, ok := index[id]; ok && nodes[i].State == "pending" {
				nodes[i].AttemptID = q.PendingID
				nodes[i].State = "running"
			}
			continue
		case "report.final_edit.style.submitted":
			id = stageID("style_edit", 0, 0)
		case "report.final_edit.gate.started":
			id = stageID("corrective_gate", 0, 0)
			if i, ok := index[id]; ok && nodes[i].State == "pending" {
				nodes[i].AttemptID = q.PendingID
				nodes[i].State = "running"
			}
			continue
		case "report.final_edit.gate.submitted":
			id = stageID("corrective_gate", 0, 0)
		case "report.final_edit.style_semantic_validation.started":
			id = stageID("style_semantic_validation", 0, 0)
			if i, ok := index[id]; ok && nodes[i].State == "pending" {
				nodes[i].AttemptID = q.PendingID
				nodes[i].State = "running"
			}
			continue
		case "report.final_edit.style_semantic_validation.submitted":
			id = stageID("style_semantic_validation", 0, 0)
		case "report.final_edit.evidence_gate.started":
			id = stageID("evidence_gate", 0, 0)
			if i, ok := index[id]; ok && nodes[i].State == "pending" {
				nodes[i].AttemptID = q.PendingID
				nodes[i].State = "running"
			}
			continue
		case "report.final_edit.evidence_gate.submitted":
			id = stageID("evidence_gate", 0, 0)
		case "report.source_packet.started", "report.il_source_selection.started", "report.il_editorial_memory.started", "report.il_narrative.started", "report.il_long_form_plan.started", "report.il_long_form_sections.started", "report.il_long_form_parts.started", "report.il_long_form_final.started", "report.il_continuity.started", "report.il_reader.started", "report.il_images.started", "report.il_document.started", "report.il_flow.started", "report.il_render.started", "report.il_store.started":
			id = ilProgressStageID(e.EventType)
			if i, ok := index[id]; ok && nodes[i].State == "pending" {
				nodes[i].State = "running"
				nodes[i].AttemptID = q.PendingID
			}
			continue
		case "report.il_source_selection.reused", "report.il_editorial_memory.reused", "report.il_narrative.reused", "report.il_long_form_plan.reused", "report.il_long_form_sections.reused", "report.il_long_form_parts.reused", "report.il_long_form_final.reused", "report.il_reader.reused", "report.il_continuity.reused":
			id = ilProgressStageID(e.EventType)
		case "report.source_packet.completed", "report.il_source_selection.completed", "report.il_editorial_memory.completed", "report.il_narrative.completed", "report.il_long_form_plan.completed", "report.il_long_form_sections.completed", "report.il_long_form_parts.completed", "report.il_long_form_final.completed", "report.il_continuity.completed", "report.il_reader.completed", "report.il_images.completed", "report.il_document.completed", "report.il_flow.completed", "report.il_render.completed", "report.il_store.completed":
			id = ilProgressStageID(e.EventType)
		case "report.source_packet.failed", "report.il_source_selection.failed", "report.il_editorial_memory.failed", "report.il_narrative.failed", "report.il_long_form_plan.failed", "report.il_long_form_sections.failed", "report.il_long_form_parts.failed", "report.il_long_form_final.failed", "report.il_continuity.failed", "report.il_reader.failed", "report.il_images.failed", "report.il_document.failed", "report.il_flow.failed", "report.il_render.failed", "report.il_store.failed":
			id = ilProgressStageID(e.EventType)
		case "report.artifact.created", "report.artifact.exported":
			if q.PendingID != selected {
				continue
			}
			if result.State == "failed" || result.State == "skipped" {
				continue
			}
			for _, n := range []string{"final", "artifact"} {
				if i, ok := index[n]; ok {
					nodes[i].State = "completed"
					nodes[i].AttemptID = q.PendingID
				}
			}
		case "report.section.failed":
			id = stageID("section", q.Part, q.Section)
		case "report.il_long_form_section.failed":
			id = stageID("section", q.Part, q.Section)
			if _, ok := index[id]; !ok {
				return unknownReportProgress()
			}
		case "report.il_long_form_part.failed":
			id = stageID("part_edit", q.Part, 0)
			if _, ok := index[id]; !ok {
				return unknownReportProgress()
			}
		case "report.part.failed":
			id = stageID("part", q.Part, 0)
		case "report.part_edit.failed":
			id = stageID(reportPartEditProgressKind(partPlanningEnabled), q.Part, 0)
		case "report.part_plan.failed":
			id = stageID("part_plan", q.Part, 0)
		case "report.final.failed":
			if mapped, ok := finalEditProgressStageID(q.FailedStage); ok {
				id = mapped
			} else {
				id = "final"
			}
		}
		if i, ok := index[id]; ok {
			nodes[i].AttemptID = q.PendingID
			if strings.HasSuffix(e.EventType, ".failed") {
				nodes[i].State = "failed"
				nodes[i].Error = safeText(q.Error)
			} else {
				nodes[i].State = "completed"
			}
		}
	}
	if failure, failed := terminal[selected]; failed && result.State != "completed" && result.State != "skipped" {
		result.State = "failed"
		id := failure.FailedStageID
		if failure.FailedStage == "part_plan" {
			id = stageID("part_plan", failure.Part, 0)
		}
		if failure.FailedStage == "part_edit" && partPlanningEnabled {
			id = stageID("part_author", failure.Part, 0)
		}
		if mapped, ok := finalEditProgressStageID(failure.FailedStage); ok {
			id = mapped
		}
		if id == "" {
			id = failure.FailedStage
		}
		if id == "" {
			id = "final"
		}
		if i, ok := index[id]; ok {
			nodes[i].State = "failed"
			nodes[i].Error = safeText(failure.Error)
		}
	}
	// 열린 attempt에서는 첫 번째 미완료 stage만 running node로 본다.
	if result.State == "running" && !hasRunningReportNode(nodes) {
		for i := range nodes {
			if nodes[i].State == "pending" || nodes[i].State == "unknown" {
				nodes[i].State = "running"
				break
			}
		}
	}
	applyReportNodeTiming(nodes, pending[selected].CreatedAt, terminalEvents[selected], events, lineage)
	result.Nodes = nodes
	if result.State == "failed" && pendingTypes[selected] == "report.draft.pending" {
		if strings.TrimSpace(p.PipelineFamily) == "report_il_experimental" {
			if p.ReportMode == "long_form" && reportILProgressLineageHasCheckpoint(events, selected) {
				result.Retry = ReportRetryCapability{ResumeFailed: true, Restart: true}
			} else if p.ReportMode == "long_form" && reportILPartsCheckpointEligible(events, selected, terminal[selected]) {
				result.Retry = ReportRetryCapability{ResumeFailed: true, Restart: true, ReasonCode: "parts_checkpoint_recoverable", Reason: "완료된 Section과 Part를 검증한 뒤 최종 원고 단계부터 이어서 생성합니다."}
			} else if p.ReportMode == "long_form" && reportILLegacyCheckpointEligible(events, selected, terminal[selected]) {
				result.Retry = ReportRetryCapability{ResumeFailed: true, Restart: true, ReasonCode: "legacy_checkpoint_recoverable", Reason: "기존 완료 원고를 검증한 뒤 실패 지점부터 이어서 생성합니다."}
			} else if p.ReportMode == "long_form" {
				result.Retry = ReportRetryCapability{Restart: true, ReasonCode: "resume_checkpoint_missing", Reason: "이어갈 수 있는 검증된 체크포인트가 없어 처음부터 다시 생성할 수 있습니다."}
			} else {
				result.Retry = ReportRetryCapability{ReasonCode: "retry_requires_long_form", Reason: "다시 생성은 장문 보고서 실패에만 사용할 수 있습니다."}
			}
		} else if p.ReportMode == "long_form" {
			result.Retry = ReportRetryCapability{ResumeFailed: true, Restart: true}
		} else {
			result.Retry = ReportRetryCapability{ReasonCode: "retry_requires_long_form", Reason: "다시 생성은 장문 보고서 실패에만 사용할 수 있습니다."}
		}
	} else if result.State == "failed" {
		result.Retry = ReportRetryCapability{ReasonCode: "retry_not_supported", Reason: "이 리포트 작업은 다시 생성하지 않습니다."}
	} else if result.State == "skipped" {
		result.Retry = ReportRetryCapability{ReasonCode: "attempt_canceled", Reason: "취소된 리포트 시도는 실패 지점 재시도를 지원하지 않습니다."}
	} else {
		result.Retry = ReportRetryCapability{ReasonCode: "attempt_not_failed", Reason: "실패한 리포트 시도만 다시 생성할 수 있습니다."}
	}
	return result
}

func reportILPartsCheckpointEligible(events []Event, pendingID string, failure reportPayload) bool {
	if failure.FailedStage != "il_long_form_final" && failure.FailedStageID != "il_long_form_final" {
		return false
	}
	partsCompleted := false
	planFinalized, memoryFinalized := false, false
	partArtifacts, sectionArtifacts := 0, 0
	for _, event := range events {
		var payload struct {
			PendingID string `json:"pending_event_id"`
			ToolName  string `json:"tool_name"`
			Success   bool   `json:"success"`
			IOMetrics struct {
				Stage      string `json:"report_il_stage"`
				ArtifactID string `json:"artifact_id"`
				Finalized  bool   `json:"finalized"`
			} `json:"io_metrics"`
		}
		_ = json.Unmarshal(event.Payload, &payload)
		if payload.PendingID == pendingID && event.EventType == "report.il_long_form_parts.completed" {
			partsCompleted = true
		}
		if event.EventType != "mcp.tool.called" || !payload.Success || payload.IOMetrics.ArtifactID == "" {
			continue
		}
		switch {
		case payload.ToolName == reportilcontract.EditorialMemoryFinalizeTool && payload.IOMetrics.Stage == "il_editorial_memory" && payload.IOMetrics.Finalized:
			memoryFinalized = true
		case payload.ToolName == reportilcontract.LongFormPlanSubmitTool && payload.IOMetrics.Stage == "il_long_form_plan":
			planFinalized = true
		case payload.ToolName == reportilcontract.LongFormDocumentFinalizeTool && payload.IOMetrics.Stage == "il_long_form_section" && payload.IOMetrics.Finalized:
			sectionArtifacts++
		case payload.ToolName == reportilcontract.LongFormDocumentFinalizeTool && payload.IOMetrics.Stage == "il_long_form_part" && payload.IOMetrics.Finalized:
			partArtifacts++
		}
	}
	return partsCompleted && planFinalized && memoryFinalized && partArtifacts >= reportilcontract.MinLongFormParts && sectionArtifacts >= reportilcontract.MinLongFormSections
}

func reportILLegacyCheckpointEligible(events []Event, pendingID string, failure reportPayload) bool {
	if failure.FailedStage != "il_reader" && failure.FailedStageID != "il_reader" {
		return false
	}
	finalCompleted := false
	memoryArtifact := false
	finalArtifact := false
	for _, event := range events {
		var payload struct {
			PendingID string `json:"pending_event_id"`
			ToolName  string `json:"tool_name"`
			Success   bool   `json:"success"`
			IOMetrics struct {
				Stage      string `json:"report_il_stage"`
				ArtifactID string `json:"artifact_id"`
				SHA256     string `json:"sha256"`
				ByteSize   int    `json:"byte_size"`
				Revision   int    `json:"revision"`
				Accounts   int    `json:"accounts"`
				Finalized  bool   `json:"finalized"`
			} `json:"io_metrics"`
		}
		_ = json.Unmarshal(event.Payload, &payload)
		if payload.PendingID == pendingID && event.EventType == "report.il_long_form_final.completed" {
			finalCompleted = true
		}
		if event.EventType != "mcp.tool.called" || !payload.Success || !payload.IOMetrics.Finalized ||
			payload.IOMetrics.ArtifactID == "" || len(payload.IOMetrics.SHA256) != 64 ||
			payload.IOMetrics.ByteSize < 1 || payload.IOMetrics.Revision < 1 {
			continue
		}
		switch {
		case payload.ToolName == reportilcontract.EditorialMemoryFinalizeTool &&
			payload.IOMetrics.Stage == "il_editorial_memory" && payload.IOMetrics.Accounts > 0:
			memoryArtifact = true
		case payload.ToolName == reportilcontract.LongFormDocumentFinalizeTool &&
			payload.IOMetrics.Stage == "il_long_form_final":
			finalArtifact = true
		}
	}
	return finalCompleted && memoryArtifact && finalArtifact
}

func reportILProgressLineageHasCheckpoint(events []Event, pendingID string) bool {
	lineage := reportProgressPendingLineage(events, pendingID)
	for attemptID := range lineage {
		if reportILProgressHasCheckpoint(events, attemptID) {
			return true
		}
	}
	return false
}

func reportILProgressHasCheckpoint(events []Event, pendingID string) bool {
	for _, event := range events {
		if event.EventType != "report.il.checkpoint.created" {
			continue
		}
		var payload struct {
			PendingID  string `json:"pending_event_id"`
			Checkpoint struct {
				Stage string `json:"stage"`
			} `json:"checkpoint"`
		}
		if json.Unmarshal(event.Payload, &payload) == nil && payload.PendingID == pendingID {
			switch payload.Checkpoint.Stage {
			case "il_long_form_parts", "il_long_form_final", "il_reader", "il_continuity":
				return true
			}
		}
	}
	return false
}

// applyReportNodeTiming은 저장된 장부 timestamp만으로 stage boundary를 계산한다.
// CreatedAt이 없는 이벤트는 timing을 비워 두어 legacy projection이 지어낸 값을 보여 주지 않게 한다.
func applyReportNodeTiming(nodes []ReportProgressNode, attemptStartedAt time.Time, attemptTerminal Event, events []Event, lineage map[string]bool) {
	starts := map[string]time.Time{}
	terminals := map[string]time.Time{}
	partPlanningEnabled := reportLineageHasPartPlanning(events, lineage)
	for _, event := range events {
		var payload reportPayload
		_ = json.Unmarshal(event.Payload, &payload)
		if !lineage[payload.PendingID] || event.CreatedAt.IsZero() {
			continue
		}
		switch event.EventType {
		case "report.section.started", "report.il_long_form_section.started":
			starts[stageID("section", payload.Part, payload.Section)] = event.CreatedAt
		case "report.il_long_form_plan.created":
			terminals["il_long_form_plan"] = event.CreatedAt
		case "report.il_long_form_part.started":
			starts[stageID("part_edit", payload.Part, 0)] = event.CreatedAt
		case "report.plan.created":
			terminals["plan"] = event.CreatedAt
		case "report.part_plan.created", "report.part_plan.failed":
			terminals[stageID("part_plan", payload.Part, 0)] = event.CreatedAt
		case "report.section.created", "report.section.failed", "report.il_long_form_section.completed", "report.il_long_form_section.failed":
			terminals[stageID("section", payload.Part, payload.Section)] = event.CreatedAt
		case "report.il_long_form_part.completed", "report.il_long_form_part.failed":
			terminals[stageID("part_edit", payload.Part, 0)] = event.CreatedAt
		case "report.part.created", "report.part.failed":
			terminals[stageID("part", payload.Part, 0)] = event.CreatedAt
		case "report.part_edit.started":
			starts[stageID(reportPartEditProgressKind(partPlanningEnabled), payload.Part, 0)] = event.CreatedAt
		case "report.part.edited", "report.part_edit.failed":
			terminals[stageID(reportPartEditProgressKind(partPlanningEnabled), payload.Part, 0)] = event.CreatedAt
		case "report.final_assembly.created":
			terminals[stageID("final_assembly", 0, 0)] = event.CreatedAt
		case "report.final_edit.writer.started":
			starts[stageID("final_write", 0, 0)] = event.CreatedAt
		case "report.final_edit.writer.submitted":
			terminals[stageID("final_write", 0, 0)] = event.CreatedAt
		case "report.final_edit.reader.started":
			starts[stageID("reader_edit", 0, 0)] = event.CreatedAt
		case "report.final_edit.reader.submitted":
			terminals[stageID("reader_edit", 0, 0)] = event.CreatedAt
		case "report.final_edit.style.started":
			starts[stageID("style_edit", 0, 0)] = event.CreatedAt
		case "report.final_edit.style.submitted":
			terminals[stageID("style_edit", 0, 0)] = event.CreatedAt
		case "report.final_edit.gate.started":
			starts[stageID("corrective_gate", 0, 0)] = event.CreatedAt
		case "report.final_edit.gate.submitted":
			terminals[stageID("corrective_gate", 0, 0)] = event.CreatedAt
		case "report.final_edit.style_semantic_validation.started":
			starts[stageID("style_semantic_validation", 0, 0)] = event.CreatedAt
		case "report.final_edit.style_semantic_validation.submitted":
			terminals[stageID("style_semantic_validation", 0, 0)] = event.CreatedAt
		case "report.final_edit.evidence_gate.started":
			starts[stageID("evidence_gate", 0, 0)] = event.CreatedAt
		case "report.final_edit.evidence_gate.submitted":
			terminals[stageID("evidence_gate", 0, 0)] = event.CreatedAt
		case "report.final.failed":
			if mapped, ok := finalEditProgressStageID(payload.FailedStage); ok {
				terminals[mapped] = event.CreatedAt
			} else {
				terminals["final"] = event.CreatedAt
			}
		case "report.source_packet.started", "report.il_source_selection.started", "report.il_editorial_memory.started", "report.il_narrative.started", "report.il_long_form_plan.started", "report.il_long_form_sections.started", "report.il_long_form_parts.started", "report.il_long_form_final.started", "report.il_continuity.started", "report.il_reader.started", "report.il_images.started", "report.il_document.started", "report.il_flow.started", "report.il_render.started", "report.il_store.started":
			starts[ilProgressStageID(event.EventType)] = event.CreatedAt
		case "report.source_packet.completed", "report.il_source_selection.completed", "report.il_editorial_memory.completed", "report.il_narrative.completed", "report.il_long_form_plan.completed", "report.il_long_form_sections.completed", "report.il_long_form_parts.completed", "report.il_long_form_final.completed", "report.il_continuity.completed", "report.il_reader.completed", "report.il_images.completed", "report.il_document.completed", "report.il_flow.completed", "report.il_render.completed", "report.il_store.completed", "report.source_packet.failed", "report.il_source_selection.failed", "report.il_editorial_memory.failed", "report.il_narrative.failed", "report.il_long_form_plan.failed", "report.il_long_form_sections.failed", "report.il_long_form_parts.failed", "report.il_long_form_final.failed", "report.il_continuity.failed", "report.il_reader.failed", "report.il_images.failed", "report.il_document.failed", "report.il_flow.failed", "report.il_render.failed", "report.il_store.failed":
			terminals[ilProgressStageID(event.EventType)] = event.CreatedAt
		case "report.artifact.created", "report.artifact.exported":
			terminals["final"] = event.CreatedAt
			terminals["artifact"] = event.CreatedAt
		}
	}
	if !attemptTerminal.CreatedAt.IsZero() {
		var payload reportPayload
		_ = json.Unmarshal(attemptTerminal.Payload, &payload)
		if strings.HasSuffix(attemptTerminal.EventType, ".failed") {
			id := payload.FailedStageID
			if mapped, ok := finalEditProgressStageID(payload.FailedStage); ok {
				id = mapped
			}
			if id == "" {
				id = payload.FailedStage
			}
			if id == "" {
				id = "final"
			}
			terminals[id] = attemptTerminal.CreatedAt
		}
	}

	previous := attemptStartedAt
	for i := range nodes {
		terminalAt, completed := terminals[nodes[i].ID]
		if !completed && nodes[i].State != "running" {
			continue
		}
		startedAt := starts[nodes[i].ID]
		if startedAt.IsZero() {
			startedAt = previous
		}
		if !startedAt.IsZero() {
			nodes[i].StartedAt = &startedAt
		}
		if !completed {
			continue
		}
		if !startedAt.IsZero() && !terminalAt.Before(startedAt) {
			durationMS := terminalAt.Sub(startedAt).Milliseconds()
			nodes[i].DurationMS = &durationMS
		}
		previous = terminalAt
	}
}

func hasRunningReportNode(nodes []ReportProgressNode) bool {
	for _, node := range nodes {
		if node.State == "running" {
			return true
		}
	}
	return false
}

func unknownReportProgress() ReportProgress {
	return ReportProgress{State: "unknown", Retry: ReportRetryCapability{ReasonCode: "invalid_lineage", Reason: "리포트 계보를 안전하게 확인할 수 없습니다."}}
}

func reportILSelectionStageExists(events []Event, pendingID string, terminal reportPayload) bool {
	return reportILStageExists(events, pendingID, terminal, "il_source_selection")
}

func reportILStageExists(events []Event, pendingID string, terminal reportPayload, stage string) bool {
	if terminal.FailedStage == stage || terminal.FailedStageID == stage {
		return true
	}
	lineage := reportProgressPendingLineage(events, pendingID)
	for _, event := range events {
		var payload reportPayload
		_ = json.Unmarshal(event.Payload, &payload)
		if lineage[payload.PendingID] && ilProgressStageID(event.EventType) == stage {
			return true
		}
	}
	return false
}

func reportProgressPendingLineage(events []Event, pendingID string) map[string]bool {
	parents := map[string]string{}
	for _, event := range events {
		if event.EventType != "report.draft.pending" {
			continue
		}
		var payload reportPayload
		_ = json.Unmarshal(event.Payload, &payload)
		parents[event.EventID] = strings.TrimSpace(payload.RetryOf)
	}
	lineage := map[string]bool{}
	for current, depth := pendingID, 0; current != "" && depth < 64; depth++ {
		if lineage[current] {
			return map[string]bool{}
		}
		lineage[current] = true
		current = parents[current]
	}
	return lineage
}

func ilProgressStageID(eventType string) string {
	parts := strings.Split(eventType, ".")
	if len(parts) < 3 {
		return ""
	}
	name := strings.TrimPrefix(eventType, "report.")
	for _, suffix := range []string{".started", ".completed", ".reused", ".failed"} {
		name = strings.TrimSuffix(name, suffix)
	}
	return name
}

func stageID(kind string, part, section int) string {
	if kind == "section" {
		return "section-" + itoa(part) + "-" + itoa(section)
	}
	if kind == "part_plan" {
		return "part-plan-" + itoa(part)
	}
	if kind == "part_edit" {
		return "part-edit-" + itoa(part)
	}
	if kind == "part_author" {
		return "part-author-" + itoa(part)
	}
	if kind == "final_assembly" {
		return "final-assembly"
	}
	if kind == "final_write" {
		return "final-write"
	}
	if kind == "reader_edit" {
		return "reader-edit"
	}
	if kind == "style_edit" {
		return "style-edit"
	}
	if kind == "style_semantic_validation" {
		return "style-semantic-validation"
	}
	if kind == "evidence_gate" {
		return "evidence-gate"
	}
	if kind == "corrective_gate" {
		return "corrective-gate"
	}
	return "part-" + itoa(part)
}

func reportPartEditProgressKind(partPlanningEnabled bool) string {
	if partPlanningEnabled {
		return "part_author"
	}
	return "part_edit"
}

func finalEditProgressStageID(kind string) (string, bool) {
	switch strings.TrimSpace(kind) {
	case "final_assembly":
		return stageID("final_assembly", 0, 0), true
	case "final_write":
		return stageID("final_write", 0, 0), true
	case "reader_edit":
		return stageID("reader_edit", 0, 0), true
	case "style_edit":
		return stageID("style_edit", 0, 0), true
	case "style_semantic_validation":
		return stageID("style_semantic_validation", 0, 0), true
	case "evidence_gate":
		return stageID("evidence_gate", 0, 0), true
	case "corrective_gate":
		return stageID("corrective_gate", 0, 0), true
	default:
		return "", false
	}
}

func reportLineageHasPartPlanning(events []Event, lineage map[string]bool) bool {
	for _, event := range events {
		var payload reportPayload
		_ = json.Unmarshal(event.Payload, &payload)
		if lineage[payload.PendingID] && event.EventType == "report.plan.created" && payload.PartPlanning {
			return true
		}
	}
	return false
}
func itoa(v int) string { b, _ := json.Marshal(v); return string(b) }
func safeText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "안전한 오류 정보가 없습니다."
	}
	if len(value) > 240 {
		return value[:240]
	}
	return value
}

// sort import가 안정적인 확장 지점을 컴파일 시점에 계속 확인하게 한다.
var _ = sort.Strings

package ledgerstate

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// ReportProgress is a conservative, ledger-derived view of report work.
// Pending events are the durable at-least-once visibility boundary: a failed
// terminal write remains pending and is projected conservatively. A durable
// terminal-write-pending outbox is intentionally follow-up work, not part of
// this projection.
type ReportProgress struct {
	AttemptID string                `json:"attempt_id,omitempty"`
	OriginID  string                `json:"origin_pending_event_id,omitempty"`
	Attempt   int                   `json:"attempt_number"`
	State     string                `json:"state"`
	Nodes     []ReportProgressNode  `json:"nodes"`
	Retry     ReportRetryCapability `json:"retry"`
}

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

type ReportRetryCapability struct {
	ResumeFailed bool   `json:"resume_failed"`
	Restart      bool   `json:"restart"`
	ReasonCode   string `json:"reason_code,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

type reportPayload struct {
	PendingID     string `json:"pending_event_id"`
	OriginID      string `json:"origin_pending_event_id"`
	RetryOf       string `json:"retry_of_pending_event_id"`
	RetryStrategy string `json:"retry_strategy"`
	Attempt       int    `json:"attempt_number"`
	ReportMode    string `json:"report_mode"`
	Part          int    `json:"part_index"`
	Section       int    `json:"section_index"`
	FailedStage   string `json:"failed_stage_kind"`
	FailedStageID string `json:"failed_stage_id"`
	Error         string `json:"safe_error_message"`
	Retryable     bool   `json:"retryable"`
	Kind          string `json:"kind"`
	Plan          struct {
		Parts []struct {
			Sections []json.RawMessage `json:"sections"`
		} `json:"parts"`
	} `json:"plan"`
}

// ProjectReportProgress normalizes legacy report events and never infers a
// completed stage without its corresponding ledger event.
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
	// Build the real graph shape from the plan, then apply only actual events.
	nodes := []ReportProgressNode{{ID: "plan", Kind: "plan", State: "pending"}}
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
			// Restart has a direct failed parent but deliberately does not reuse it.
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
	partCount := 0
	for _, e := range events {
		var q reportPayload
		_ = json.Unmarshal(e.Payload, &q)
		if !lineage[q.PendingID] {
			continue
		}
		if e.EventType == "report.plan.created" {
			nodes[0].State = "completed"
			nodes[0].AttemptID = q.PendingID
			partCount = len(q.Plan.Parts)
			for i, part := range q.Plan.Parts {
				for j := range part.Sections {
					nodes = append(nodes, ReportProgressNode{ID: stageID("section", i+1, j+1), Kind: "section", Part: i + 1, Section: j + 1, State: "pending"})
				}
			}
			for i := range q.Plan.Parts {
				nodes = append(nodes, ReportProgressNode{ID: stageID("part", i+1, 0), Kind: "part", Part: i + 1, State: "pending"})
			}
		}
	}
	if pendingTypes[selected] != "report.draft.pending" || p.ReportMode != "long_form" {
		// Non-sectional operations have one preparation boundary followed by finalization/artifact.
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
		case "report.section.started":
			id = stageID("section", q.Part, q.Section)
			if i, ok := index[id]; ok && nodes[i].State == "pending" {
				nodes[i].AttemptID = q.PendingID
				nodes[i].State = "running"
			}
			continue
		case "report.section.created":
			id = stageID("section", q.Part, q.Section)
		case "report.part.created":
			id = stageID("part", q.Part, 0)
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
		case "report.part.failed":
			id = stageID("part", q.Part, 0)
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
	// The first incomplete stage is the only running node in an open attempt.
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
		if p.ReportMode == "long_form" {
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

// applyReportNodeTiming derives stage boundaries solely from durable ledger
// timestamps. Events without CreatedAt intentionally leave timing absent so
// legacy projections do not present invented values.
func applyReportNodeTiming(nodes []ReportProgressNode, attemptStartedAt time.Time, attemptTerminal Event, events []Event, lineage map[string]bool) {
	starts := map[string]time.Time{}
	terminals := map[string]time.Time{}
	for _, event := range events {
		var payload reportPayload
		_ = json.Unmarshal(event.Payload, &payload)
		if !lineage[payload.PendingID] || event.CreatedAt.IsZero() {
			continue
		}
		switch event.EventType {
		case "report.section.started":
			starts[stageID("section", payload.Part, payload.Section)] = event.CreatedAt
		case "report.plan.created":
			terminals["plan"] = event.CreatedAt
		case "report.section.created", "report.section.failed":
			terminals[stageID("section", payload.Part, payload.Section)] = event.CreatedAt
		case "report.part.created", "report.part.failed":
			terminals[stageID("part", payload.Part, 0)] = event.CreatedAt
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

func stageID(kind string, part, section int) string {
	if kind == "section" {
		return "section-" + itoa(part) + "-" + itoa(section)
	}
	return "part-" + itoa(part)
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

// Keep sort imported as a compile-time guard for stable future expansion.
var _ = sort.Strings

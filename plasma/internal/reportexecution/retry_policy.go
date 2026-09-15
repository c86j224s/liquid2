package reportexecution

import (
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"strings"
)

const reportRetryLineageLimit = 64

// ReportRetryRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type ReportRetryRequest struct {
	EventID, MissionID, FailedPendingEventID, Strategy, RetryRequestID string
	Producer                                                           ledger.Producer
}

type ReportAttemptPayload struct {
	OriginID       string `json:"origin_pending_event_id"`
	RetryOf        string `json:"retry_of_pending_event_id"`
	RetryStrategy  string `json:"retry_strategy"`
	RetryRequestID string `json:"retry_request_id"`
	ReportMode     string `json:"report_mode"`
	PipelineFamily string `json:"pipeline_family"`
	Attempt        int    `json:"attempt_number"`
}

type ReportTerminalPayload struct {
	PendingID string `json:"pending_event_id"`
	Kind      string `json:"kind"`
}

type ReportAttempt struct {
	ledger.Event
	ReportAttemptPayload
	Payload map[string]any
}

func ReportAttempts(events []ledger.Event) (map[string]ReportAttempt, map[string]string, error) {
	attempts := map[string]ReportAttempt{}
	terminals := map[string]string{}
	for _, event := range events {
		if event.EventType == "report.draft.pending" {
			var payload map[string]any
			if json.Unmarshal(event.Payload, &payload) != nil {
				return nil, nil, fmt.Errorf("%w: report pending payload is invalid", producterror.ErrInvalidInput)
			}
			var p ReportAttemptPayload
			_ = json.Unmarshal(event.Payload, &p)
			if p.OriginID == "" {
				p.OriginID = event.EventID
			}
			if p.Attempt < 1 {
				p.Attempt = 1
			}
			attempts[event.EventID] = ReportAttempt{Event: event, ReportAttemptPayload: p, Payload: payload}
		}
		if event.EventType == "report.draft.failed" {
			var p ReportTerminalPayload
			_ = json.Unmarshal(event.Payload, &p)
			if p.PendingID != "" {
				if _, exists := terminals[p.PendingID]; exists {
					return nil, nil, fmt.Errorf("%w: conflicting report terminal outcomes", producterror.ErrInvalidInput)
				}
				if p.Kind == "report_draft_canceled" {
					terminals[p.PendingID] = "canceled"
				} else {
					terminals[p.PendingID] = "failed"
				}
			}
		}
		if event.EventType == "report.artifact.created" || event.EventType == "report.drafted" {
			var p ReportTerminalPayload
			_ = json.Unmarshal(event.Payload, &p)
			if p.PendingID != "" {
				if _, exists := terminals[p.PendingID]; exists {
					return nil, nil, fmt.Errorf("%w: conflicting report terminal outcomes", producterror.ErrInvalidInput)
				}
				terminals[p.PendingID] = "completed"
			}
		}
	}
	return attempts, terminals, nil
}
func RetryIdempotencyMatch(attempts map[string]ReportAttempt, req ReportRetryRequest) (ledger.Event, bool) {
	for _, a := range attempts {
		if a.RetryRequestID == req.RetryRequestID && a.RetryOf == req.FailedPendingEventID && a.RetryStrategy == req.Strategy {
			return a.Event, true
		}
	}
	return ledger.Event{}, false
}
func RetryRequestIDTaken(attempts map[string]ReportAttempt, id string) bool {
	for _, a := range attempts {
		if a.RetryRequestID == id {
			return true
		}
	}
	return false
}
func ValidateRetryLeafAndLineage(attempts map[string]ReportAttempt, target ReportAttempt) error {
	for _, child := range attempts {
		if child.RetryOf == target.EventID {
			return fmt.Errorf("%w: retry target has a newer attempt", producterror.ErrConflict)
		}
	}
	seen := map[string]bool{}
	current := target
	for depth := 0; depth < reportRetryLineageLimit; depth++ {
		if seen[current.EventID] {
			return fmt.Errorf("%w: report retry lineage cycle", producterror.ErrInvalidInput)
		}
		seen[current.EventID] = true
		if current.OriginID == "" {
			return fmt.Errorf("%w: report retry origin missing", producterror.ErrInvalidInput)
		}
		if current.RetryOf == "" {
			if current.OriginID != current.EventID {
				return fmt.Errorf("%w: report retry origin mismatch", producterror.ErrInvalidInput)
			}
			return nil
		}
		parent, ok := attempts[current.RetryOf]
		if !ok {
			return fmt.Errorf("%w: report retry ancestor missing", producterror.ErrInvalidInput)
		}
		if parent.OriginID != current.OriginID {
			return fmt.Errorf("%w: report retry lineage origin mismatch", producterror.ErrInvalidInput)
		}
		current = parent
	}
	return fmt.Errorf("%w: report retry lineage too deep", producterror.ErrInvalidInput)
}
func ValidateReportRetryRequest(req ReportRetryRequest) error {
	if err := validateID("mis_", req.MissionID); err != nil {
		return err
	}
	if err := validateID("evt_", req.EventID); err != nil {
		return err
	}
	if req.Strategy != "resume_failed" && req.Strategy != "restart" {
		return fmt.Errorf("%w: unsupported retry strategy", producterror.ErrInvalidInput)
	}
	if strings.TrimSpace(req.RetryRequestID) == "" {
		return fmt.Errorf("%w: retry_request_id is required", producterror.ErrInvalidInput)
	}
	return nil
}
func CopyRetryPayload(payload map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range payload {
		out[k] = v
	}
	return out
}
func ReportILLegacyCheckpointAvailable(events []ledger.Event, pendingID string) bool {
	failedAtReader := false
	finalCompleted := false
	memoryArtifact := false
	finalArtifact := false
	for _, event := range events {
		var payload struct {
			PendingID string `json:"pending_event_id"`
			Failed    string `json:"failed_stage_kind"`
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
		if payload.PendingID == pendingID {
			switch event.EventType {
			case "report.draft.failed":
				failedAtReader = payload.Failed == "il_reader"
			case "report.il_long_form_final.completed":
				finalCompleted = true
			}
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
	return failedAtReader && finalCompleted && memoryArtifact && finalArtifact
}

func ReportILLineageCheckpointAvailable(events []ledger.Event, attempts map[string]ReportAttempt, target ReportAttempt) bool {
	current := target
	for depth := 0; depth < reportRetryLineageLimit; depth++ {
		if reportILCheckpointAvailable(events, current.EventID) {
			return true
		}
		if current.RetryOf == "" {
			return false
		}
		parent, ok := attempts[current.RetryOf]
		if !ok {
			return false
		}
		current = parent
	}
	return false
}

func ReportILSourceSelectionCheckpointAvailable(events []ledger.Event, pendingID string) bool {
	failedAtMemory := false
	selectionCompleted := false
	readComplete := false
	catalogSHA := ""
	seen := map[string]bool{}
	for _, event := range reportILAttemptEventRange(events, pendingID) {
		var payload struct {
			PendingID string `json:"pending_event_id"`
			Failed    string `json:"failed_stage_kind"`
			ToolName  string `json:"tool_name"`
			Success   bool   `json:"success"`
			IOMetrics struct {
				Stage         string `json:"report_il_stage"`
				CatalogSHA256 string `json:"catalog_sha256"`
				Remaining     int    `json:"remaining_sources"`
				SourceReads   []struct {
					SourceKey string `json:"source_key"`
				} `json:"source_reads"`
			} `json:"io_metrics"`
		}
		_ = json.Unmarshal(event.Payload, &payload)
		if payload.PendingID == pendingID {
			if event.EventType == "report.draft.failed" && payload.Failed == "il_editorial_memory" {
				failedAtMemory = true
			}
			if event.EventType == "report.il_source_selection.completed" {
				selectionCompleted = true
			}
		}
		if event.EventType == "mcp.tool.called" && payload.Success && payload.ToolName == reportilcontract.SourceReadTool && payload.IOMetrics.Stage == "il_editorial_memory" {
			if catalogSHA != "" && !strings.EqualFold(catalogSHA, payload.IOMetrics.CatalogSHA256) {
				return false
			}
			catalogSHA = payload.IOMetrics.CatalogSHA256
			for _, read := range payload.IOMetrics.SourceReads {
				seen[read.SourceKey] = true
			}
			if payload.IOMetrics.Remaining == 0 {
				readComplete = true
			}
		}
	}
	return failedAtMemory && selectionCompleted && readComplete && len(catalogSHA) == 64 && len(seen) > 0
}

func reportILAttemptEventRange(events []ledger.Event, pendingID string) []ledger.Event {
	start, end := -1, len(events)
	for index, event := range events {
		if event.EventID == pendingID && event.EventType == "report.draft.pending" {
			start = index
			break
		}
	}
	if start < 0 {
		return nil
	}
	for index := start + 1; index < len(events); index++ {
		if events[index].EventType != "report.draft.failed" {
			continue
		}
		var payload ReportTerminalPayload
		if json.Unmarshal(events[index].Payload, &payload) == nil && payload.PendingID == pendingID {
			end = index + 1
			break
		}
	}
	return events[start:end]
}

func reportILCheckpointAvailable(events []ledger.Event, pendingID string) bool {
	for _, event := range events {
		if event.EventType != reportILCheckpointEventType || event.CausationEventID != pendingID {
			continue
		}
		var payload struct {
			PendingID  string `json:"pending_event_id"`
			Checkpoint struct {
				Stage string `json:"stage"`
			} `json:"checkpoint"`
		}
		if json.Unmarshal(event.Payload, &payload) == nil &&
			payload.PendingID == pendingID && reportILResumeStageOrder(payload.Checkpoint.Stage) > 0 {
			return true
		}
	}
	return false
}

func RetryResumeStage(events []ledger.Event, pendingID, strategy string) string {
	if strategy == "restart" {
		return "plan"
	}
	for _, e := range events {
		var p struct {
			PendingID     string `json:"pending_event_id"`
			FailedStage   string `json:"failed_stage_kind"`
			FailedStageID string `json:"failed_stage_id"`
		}
		_ = json.Unmarshal(e.Payload, &p)
		if e.EventType == "report.draft.failed" && p.PendingID == pendingID {
			if p.FailedStageID != "" {
				return p.FailedStageID
			}
			if p.FailedStage != "" {
				return p.FailedStage
			}
		}
	}
	return "plan"
}

const reportILCheckpointEventType = "report.il.checkpoint.created"

func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", producterror.ErrInvalidInput, prefix)
	}
	return nil
}
func reportILResumeStageOrder(stage string) int {
	switch stage {
	case "il_source_selection":
		return 1
	case "il_long_form_parts":
		return 2
	case "il_long_form_final":
		return 3
	case "il_reader":
		return 4
	case "il_continuity":
		return 5
	default:
		return 0
	}
}

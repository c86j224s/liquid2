package mcp

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/mcptrace"
)

func (server *Server) recordToolCall(ctx context.Context, call ToolCall, result ToolResult, started time.Time) (string, error) {
	binding := server.binding
	if binding.MissionID == "" && binding.AgentSessionID == "" {
		return "", nil
	}
	if binding.AgentSessionID != "" {
		if argumentSessionID := sessionIDFromArguments(call.Arguments); argumentSessionID != "" && argumentSessionID != binding.AgentSessionID {
			return "", nil
		}
	}
	missionID := strings.TrimSpace(result.MissionID)
	if missionID == "" {
		missionID = missionIDFromArguments(call.Arguments)
	}
	if missionID == "" {
		missionID = binding.MissionID
	}
	if err := validateID("mis_", missionID); err != nil {
		return "", nil
	}
	finished := time.Now().UTC()
	producer := app.Producer{Type: "mcp_server", ID: "plasma"}
	if binding.AgentSessionID != "" {
		producer = app.Producer{Type: "agent_session", ID: binding.AgentSessionID}
	}
	argumentSummary := summarizeToolArguments(call.Arguments)
	resultSummary := summarizeToolResult(result)
	eventID := newTraceEventID()
	_, err := server.service.AppendEvent(ctx, mcptrace.BuildToolCalledAppendRequest(mcptrace.ToolCalledAppendRequest{
		EventID:        eventID,
		MissionID:      missionID,
		ToolName:       call.Name,
		AgentSessionID: binding.AgentSessionID,
		StartedAt:      started,
		FinishedAt:     finished,
		Success:        result.Error == nil,
		Arguments:      argumentSummary,
		Result:         resultSummary,
		IOMetrics:      toolIOMetrics(call.Arguments, result, argumentSummary, resultSummary),
		Producer:       producer,
	}))
	if err != nil {
		return "", err
	}
	return eventID, nil
}

func missionIDFromArguments(args json.RawMessage) string {
	var input struct {
		MissionID string `json:"mission_id"`
	}
	if len(args) == 0 {
		return ""
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return ""
	}
	return strings.TrimSpace(input.MissionID)
}

func sessionIDFromArguments(args json.RawMessage) string {
	var input struct {
		SessionID string `json:"session_id"`
	}
	if len(args) == 0 {
		return ""
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return ""
	}
	return strings.TrimSpace(input.SessionID)
}

func summarizeToolArguments(args json.RawMessage) map[string]any {
	summary := map[string]any{}
	var decoded map[string]any
	if len(args) == 0 || json.Unmarshal(args, &decoded) != nil {
		return summary
	}
	allowed := map[string]struct{}{
		"mission_id":         {},
		"workflow_run_id":    {},
		"object_kind":        {},
		"object_id":          {},
		"snapshot_id":        {},
		"artifact_id":        {},
		"claim_id":           {},
		"evidence_id":        {},
		"question_id":        {},
		"proposal_id":        {},
		"query":              {},
		"target":             {},
		"limit":              {},
		"depth":              {},
		"cursor":             {},
		"offset":             {},
		"max_bytes":          {},
		"include_removed":    {},
		"include_superseded": {},
		"root_id":            {},
		"relative_path":      {},
		"restore":            {},
		"reason":             {},
		"connectors":         {},
		"candidates":         {},
		"session_id":         {},
		"idempotency_key":    {},
		"source_snapshot":    {},
		"source_artifact":    {},
		"created_record":     {},
		"requested_action":   {},
	}
	keys := make([]string, 0, len(decoded))
	for key, value := range decoded {
		keys = append(keys, key)
		if _, ok := allowed[key]; !ok {
			continue
		}
		summary[key] = summarizeTraceValue(value)
	}
	summary["argument_keys"] = keys
	return summary
}

func summarizeToolResult(result ToolResult) map[string]any {
	summary := map[string]any{
		"mission_id": strings.TrimSpace(result.MissionID),
		"success":    result.Error == nil,
	}
	if result.Error != nil {
		summary["error"] = map[string]any{
			"error_kind": result.Error.ErrorKind,
			"message":    truncateTraceString(result.Error.Message, 512),
			"retryable":  result.Error.Retryable,
		}
		return summary
	}
	if len(result.CreatedEventIDs) > 0 {
		summary["created_event_ids"] = result.CreatedEventIDs
	}
	if result.ProposalID != "" {
		summary["proposal_id"] = result.ProposalID
	}
	if len(result.CreatedRecords) > 0 {
		summary["created_records"] = result.CreatedRecords
	}
	summary["content"] = summarizeToolContent(result.Content)
	return summary
}

func toolIOMetrics(args json.RawMessage, result ToolResult, argumentSummary map[string]any, resultSummary map[string]any) map[string]any {
	metrics := map[string]any{
		"argument_raw_bytes":     len(args),
		"argument_summary_bytes": jsonByteLen(argumentSummary),
		"result_raw_bytes":       jsonByteLen(result),
		"result_summary_bytes":   jsonByteLen(resultSummary),
		"content_raw_bytes":      jsonByteLen(result.Content),
	}
	addReadIOMetrics(metrics, args, result)
	if containsTraceTruncation(argumentSummary) || containsTraceTruncation(resultSummary) {
		metrics["truncated"] = true
	}
	if result.Error != nil {
		metrics["error_kind"] = result.Error.ErrorKind
	}
	return metrics
}

func addReadIOMetrics(metrics map[string]any, args json.RawMessage, result ToolResult) {
	switch result.ToolName {
	case ToolSourcesRead:
		addSourceReadIOMetrics(metrics, args, result)
	case ToolResearchRead:
		addResearchReadIOMetrics(metrics, args, result)
	}
}

func addSourceReadIOMetrics(metrics map[string]any, args json.RawMessage, result ToolResult) {
	var input sourcesReadInput
	if err := json.Unmarshal(args, &input); err == nil {
		metrics["read_kind"] = "source"
		metrics["source_snapshot_id"] = strings.TrimSpace(input.SnapshotID)
		if strings.TrimSpace(input.ArtifactID) != "" {
			metrics["source_artifact_id"] = strings.TrimSpace(input.ArtifactID)
		}
		metrics["requested_offset"] = input.Offset
		metrics["requested_max_bytes"] = input.MaxBytes
	}
	switch content := result.Content.(type) {
	case sourcesReadOutput:
		metrics["read_kind"] = "source_text"
		metrics["source_snapshot_id"] = content.Snapshot.SnapshotID
		if content.Artifact.ArtifactID != "" {
			metrics["source_artifact_id"] = content.Artifact.ArtifactID
			metrics["source_media_type"] = content.Artifact.MediaType
		}
		metrics["returned_offset"] = content.Offset
		metrics["returned_content_bytes"] = len([]byte(content.Content))
		metrics["content_length"] = content.ContentLength
		metrics["content_length_known"] = content.ContentLengthKnown
		metrics["response_truncated"] = content.Truncated
		if content.NextOffset > 0 {
			metrics["next_offset"] = content.NextOffset
		}
		if content.Extraction != nil {
			metrics["extraction_type"] = content.Extraction.Type
			metrics["extraction_text_length"] = content.Extraction.TextLength
			metrics["extraction_text_length_known"] = content.Extraction.TextLengthKnown
			metrics["suggested_read_bytes"] = content.Extraction.SuggestedReadBytes
			metrics["max_read_bytes"] = content.Extraction.MaxReadBytes
		}
		if content.ObservationEventID != "" {
			metrics["read_kind"] = "source_live_reference"
			metrics["observation_event_id"] = content.ObservationEventID
		}
		if content.ObservationMetadata != nil {
			metrics["relative_path"] = content.ObservationMetadata.RelativePath
			if content.ObservationMetadata.Subpath != "" {
				metrics["subpath"] = content.ObservationMetadata.Subpath
			}
		}
	case mediaSourceReadOutput:
		metrics["read_kind"] = "source_media_metadata"
		metrics["source_snapshot_id"] = content.Snapshot.SnapshotID
		if content.Artifact.ArtifactID != "" {
			metrics["source_artifact_id"] = content.Artifact.ArtifactID
			metrics["source_media_type"] = content.Artifact.MediaType
		}
		metrics["media_kind"] = content.Media.MediaKind
	}
}

func addResearchReadIOMetrics(metrics map[string]any, args json.RawMessage, result ToolResult) {
	var input researchReadInput
	if err := json.Unmarshal(args, &input); err == nil {
		metrics["read_kind"] = "research_object"
		metrics["object_kind"] = strings.TrimSpace(input.ObjectKind)
		metrics["object_id"] = strings.TrimSpace(input.ObjectID)
		metrics["requested_offset"] = input.Offset
		metrics["requested_max_bytes"] = input.MaxBytes
		metrics["requested_limit"] = input.Limit
	}
	content, ok := result.Content.(app.ResearchIDEObjectRead)
	if !ok {
		return
	}
	metrics["read_kind"] = "research_object"
	metrics["object_kind"] = content.ObjectKind
	metrics["object_id"] = content.ObjectID
	metrics["returned_content_bytes"] = len([]byte(content.Data))
	metrics["response_truncated"] = content.Truncated
	if content.NextOffset > 0 {
		metrics["next_offset"] = content.NextOffset
	}
	if content.Children != nil {
		metrics["child_count"] = len(content.Children.Items)
		metrics["child_limit"] = content.Children.Limit
		metrics["child_truncated"] = content.Children.Truncated
		if content.Children.NextCursor != "" {
			metrics["child_next_cursor"] = content.Children.NextCursor
		}
	}
}

func jsonByteLen(value any) int {
	if value == nil {
		return 0
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	return len(encoded)
}

func containsTraceTruncation(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		if truncated, ok := typed["truncated"].(bool); ok && truncated {
			return true
		}
		for _, nested := range typed {
			if containsTraceTruncation(nested) {
				return true
			}
		}
	case []any:
		for _, nested := range typed {
			if containsTraceTruncation(nested) {
				return true
			}
		}
	}
	return false
}

func summarizeToolContent(content any) any {
	if content == nil {
		return nil
	}
	encoded, err := json.Marshal(content)
	if err != nil {
		return map[string]any{"type": fmt.Sprintf("%T", content)}
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return map[string]any{"type": fmt.Sprintf("%T", content), "bytes": len(encoded)}
	}
	return summarizeTraceValue(decoded)
}

func summarizeTraceValue(value any) any {
	switch typed := value.(type) {
	case string:
		return truncateTraceString(typed, 512)
	case []any:
		values := make([]any, 0, min(len(typed), 8))
		for i := 0; i < len(typed) && i < 8; i++ {
			values = append(values, summarizeTraceValue(typed[i]))
		}
		output := map[string]any{"count": len(typed), "items": values}
		if len(typed) > len(values) {
			output["truncated"] = true
		}
		return output
	case map[string]any:
		output := map[string]any{"keys": sortedTraceKeys(typed)}
		for _, key := range []string{
			"object_kind", "object_id", "mission_id", "workflow_run_id", "snapshot_id", "artifact_id", "claim_id", "evidence_id",
			"proposal_id", "title", "summary", "state", "removed", "retrieval_policy", "connector_type", "root_id", "relative_path",
			"observation_event_id", "entry_count", "match_count", "limit", "depth", "next_cursor", "truncated", "next_offset",
			"query", "matches", "entries", "items", "counts", "active_report_version_id",
		} {
			if nested, ok := typed[key]; ok {
				output[key] = summarizeTraceValue(nested)
			}
		}
		return output
	default:
		return typed
	}
}

func sortedTraceKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func truncateTraceString(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}

func truncateRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return strings.TrimSpace(string(runes[:limit])) + "..."
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func newTraceEventID() string {
	return newMCPID("evt")
}

func newMCPID(prefix string) string {
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return strings.TrimSuffix(prefix, "_") + "_" + time.Now().UTC().Format("20060102150405")
	}
	return strings.TrimSuffix(prefix, "_") + "_" + time.Now().UTC().Format("20060102150405") + "_" + hex.EncodeToString(raw[:])
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func idempotencyKey(call ToolCall) (string, string, error) {
	var input commonMutatingInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return "", input.MissionID, err
	}
	missionID := strings.TrimSpace(input.MissionID)
	sessionID := strings.TrimSpace(input.SessionID)
	key := strings.TrimSpace(input.IdempotencyKey)
	if err := validateID("mis_", missionID); err != nil {
		return "", missionID, err
	}
	if err := validateID("ses_", sessionID); err != nil {
		return "", missionID, err
	}
	if key == "" {
		return "", missionID, fmt.Errorf("%w: idempotency_key is required", app.ErrInvalidInput)
	}
	return call.Name + "\x00" + missionID + "\x00" + sessionID + "\x00" + key, missionID, nil
}

func canonicalArgumentsHash(args json.RawMessage) (string, error) {
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	decoder := json.NewDecoder(bytes.NewReader(args))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", fmt.Errorf("%w: decode idempotency arguments: %v", app.ErrInvalidInput, err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("%w: encode idempotency arguments: %v", app.ErrInvalidInput, err)
	}
	sum := sha256.Sum256(encoded)
	return fmt.Sprintf("%x", sum[:]), nil
}

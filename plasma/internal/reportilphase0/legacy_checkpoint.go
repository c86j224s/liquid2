package reportilphase0

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

type legacySourceQuote struct {
	Offset   int
	ByteSize int
	SHA256   string
}

type legacySourceBinding struct {
	SourceKey     string
	ContentLength int
	Quotes        []legacySourceQuote
}

// RecoverPartsCheckpoint verifies a failed final-author attempt's completed
// plan, Section, and Part artifacts before promoting them to a checkpoint.
func RecoverPartsCheckpoint(
	ctx context.Context,
	missionID,
	pendingID string,
	events []ledger.Event,
	sources SourceReader,
	documents interface {
		ReadReportILEditorialMemory(context.Context, string, string, reportilcontract.SourceCatalog) (reportilcontract.EditorialMemory, reportilcontract.EditorialMemoryReceipt, error)
		ReadReportILLongFormPlan(context.Context, string, string, reportilcontract.SourceCatalog) (reportilcontract.LongFormPlan, reportilcontract.LongFormPlanReceipt, error)
		ReadReportILLongFormStageDocument(context.Context, string, string, string, reportilcontract.SourceCatalog) (reportilcontract.AuthorDocument, reportilcontract.AuthorWorkspaceReceipt, error)
	},
) (*reportilcontract.ResumeCheckpoint, error) {
	attemptEvents := reportILAttemptEvents(events, pendingID)
	if !legacyStageCompleted(attemptEvents, pendingID, "report.il_long_form_parts.completed") || !legacyStageFailed(attemptEvents, pendingID, "il_long_form_final") {
		return nil, fmt.Errorf("report IL Part checkpoint boundary is not recoverable")
	}
	candidateBuild, err := BuildSourceCatalogForSelection(ctx, sources, missionID)
	if err != nil {
		return nil, err
	}
	authorSHA, memorySession, bindings := legacyAuthorCatalogTrace(attemptEvents)
	candidateSHA, candidateCount := legacyCandidateCatalogTrace(attemptEvents)
	if authorSHA == "" || memorySession == "" || len(bindings) == 0 || candidateSHA == "" ||
		!strings.EqualFold(candidateSHA, candidateBuild.Catalog.SHA256) || candidateCount != len(candidateBuild.Catalog.Sources) {
		return nil, fmt.Errorf("report IL Part checkpoint source trace is incomplete")
	}
	authorCatalog, err := reconstructLegacyAuthorCatalog(candidateBuild, bindings)
	if err != nil || !strings.EqualFold(authorCatalog.SHA256, authorSHA) {
		return nil, fmt.Errorf("report IL Part checkpoint author catalog changed")
	}
	memory, memoryReceipt, err := documents.ReadReportILEditorialMemory(ctx, missionID, memorySession, authorCatalog)
	if err != nil {
		return nil, fmt.Errorf("report IL Part checkpoint editorial memory verification failed: %w", err)
	}
	planSessions := legacyFinalizedStageSessions(attemptEvents, "il_long_form_plan", reportilcontract.LongFormPlanSubmitTool)
	if len(planSessions) != 1 {
		return nil, fmt.Errorf("report IL Part checkpoint plan trace is incomplete")
	}
	plan, planReceipt, err := documents.ReadReportILLongFormPlan(ctx, missionID, planSessions[0], authorCatalog)
	if err != nil || reportilcontract.ValidateLongFormPlan(plan, &memory, authorCatalog) != nil {
		return nil, fmt.Errorf("report IL Part checkpoint plan verification failed")
	}
	sections := map[string]reportilcontract.AuthorWorkspaceReceipt{}
	for _, sessionID := range legacyFinalizedStageSessions(attemptEvents, "il_long_form_section", reportilcontract.LongFormDocumentFinalizeTool) {
		document, receipt, readErr := documents.ReadReportILLongFormStageDocument(ctx, missionID, sessionID, "il_long_form_section", authorCatalog)
		if readErr != nil || len(document.Parts) != 1 || len(document.Parts[0].Sections) != 1 {
			return nil, fmt.Errorf("report IL Part checkpoint Section verification failed")
		}
		key := document.Parts[0].Sections[0].SectionKey
		if _, exists := sections[key]; exists {
			return nil, fmt.Errorf("report IL Part checkpoint Section is duplicated")
		}
		sections[key] = receipt
	}
	parts := map[string]reportilcontract.AuthorWorkspaceReceipt{}
	for _, sessionID := range legacyFinalizedStageSessions(attemptEvents, "il_long_form_part", reportilcontract.LongFormDocumentFinalizeTool) {
		document, receipt, readErr := documents.ReadReportILLongFormStageDocument(ctx, missionID, sessionID, "il_long_form_part", authorCatalog)
		if readErr != nil || len(document.Parts) != 1 {
			return nil, fmt.Errorf("report IL Part checkpoint Part verification failed")
		}
		key := document.Parts[0].PartKey
		if _, exists := parts[key]; exists {
			return nil, fmt.Errorf("report IL Part checkpoint Part is duplicated")
		}
		parts[key] = receipt
	}
	sectionReceipts := make([]reportilcontract.AuthorWorkspaceReceipt, 0, reportilcontract.LongFormPlanSectionCount(plan))
	partReceipts := make([]reportilcontract.AuthorWorkspaceReceipt, 0, len(plan.Parts))
	for _, part := range plan.Parts {
		partReceipt, ok := parts[part.PartKey]
		if !ok {
			return nil, fmt.Errorf("report IL Part checkpoint Part inventory is incomplete")
		}
		for _, section := range part.Sections {
			sectionReceipt, ok := sections[section.SectionKey]
			if !ok {
				return nil, fmt.Errorf("report IL Part checkpoint Section inventory is incomplete")
			}
			sectionReceipts = append(sectionReceipts, sectionReceipt)
		}
		partReceipts = append(partReceipts, partReceipt)
	}
	excludedUnusable := len(candidateBuild.Dispositions) - len(candidateBuild.Catalog.Sources)
	excludedBudget := len(candidateBuild.Catalog.Sources) - len(authorCatalog.Sources)
	if excludedUnusable < 0 || excludedBudget < 0 {
		return nil, fmt.Errorf("report IL Part checkpoint source counts are invalid")
	}
	selection := SourceSelectionReceipt{Applied: excludedUnusable+excludedBudget > 0, AcceptedSources: len(candidateBuild.Dispositions), UsableSources: len(candidateBuild.Catalog.Sources), SelectedSources: len(authorCatalog.Sources), ExcludedUnusableSources: excludedUnusable, ExcludedBudgetSources: excludedBudget, CandidateCatalogSHA256: candidateBuild.Catalog.SHA256, SelectedCatalogSHA256: authorCatalog.SHA256, AuthorCatalogSHA256: authorCatalog.SHA256}
	checkpoint := NewPartsCheckpoint(ProductConfig{PendingEventID: pendingID}, candidateBuild.Catalog.SHA256, authorCatalog, candidateBuild.ImageCatalog, selection, memoryReceipt, planReceipt, sectionReceipts, partReceipts, len(plan.Parts), reportilcontract.LongFormPlanSectionCount(plan))
	if err := reportilcontract.ValidateProductCheckpoint(checkpoint, missionID); err != nil {
		return nil, err
	}
	return &reportilcontract.ResumeCheckpoint{ProductCheckpoint: checkpoint, EditorialMemory: memory, LongFormPlan: plan}, nil
}

func reportILAttemptEvents(events []ledger.Event, pendingID string) []ledger.Event {
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
		var payload struct {
			PendingID string `json:"pending_event_id"`
		}
		if json.Unmarshal(events[index].Payload, &payload) == nil && payload.PendingID == pendingID {
			end = index + 1
			break
		}
	}
	return append([]ledger.Event(nil), events[start:end]...)
}

func legacyStageFailed(events []ledger.Event, pendingID, stage string) bool {
	for _, event := range events {
		if event.EventType != "report.draft.failed" {
			continue
		}
		var payload struct {
			PendingID string `json:"pending_event_id"`
			Stage     string `json:"failed_stage_kind"`
		}
		if json.Unmarshal(event.Payload, &payload) == nil && payload.PendingID == pendingID && payload.Stage == stage {
			return true
		}
	}
	return false
}

func legacyFinalizedStageSessions(events []ledger.Event, stage, finalizeTool string) []string {
	sessions := []string{}
	seen := map[string]bool{}
	for _, event := range events {
		if event.EventType != "mcp.tool.called" {
			continue
		}
		var payload struct {
			ToolName      string `json:"tool_name"`
			ToolSessionID string `json:"tool_session_id"`
			Success       bool   `json:"success"`
			IOMetrics     struct {
				Stage      string `json:"report_il_stage"`
				Finalized  bool   `json:"finalized"`
				ArtifactID string `json:"artifact_id"`
			} `json:"io_metrics"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil || !payload.Success || payload.ToolName != finalizeTool || payload.IOMetrics.Stage != stage || payload.IOMetrics.ArtifactID == "" {
			continue
		}
		if finalizeTool == reportilcontract.LongFormDocumentFinalizeTool && !payload.IOMetrics.Finalized {
			continue
		}
		if !seen[payload.ToolSessionID] {
			seen[payload.ToolSessionID] = true
			sessions = append(sessions, payload.ToolSessionID)
		}
	}
	return sessions
}

// RecoverLegacyCheckpoint verifies pre-v67 tool traces and durable artifacts,
// then reconstructs the same source-bound final-author checkpoint used by new runs.
func RecoverLegacyCheckpoint(
	ctx context.Context,
	missionID,
	pendingID string,
	events []ledger.Event,
	sources SourceReader,
	documents interface {
		ReadReportILEditorialMemory(context.Context, string, string, reportilcontract.SourceCatalog) (reportilcontract.EditorialMemory, reportilcontract.EditorialMemoryReceipt, error)
		ReadReportILLongFormStageDocument(context.Context, string, string, string, reportilcontract.SourceCatalog) (reportilcontract.AuthorDocument, reportilcontract.AuthorWorkspaceReceipt, error)
	},
) (*reportilcontract.ResumeCheckpoint, error) {
	candidateBuild, err := BuildSourceCatalogForSelection(ctx, sources, missionID)
	if err != nil {
		return nil, err
	}
	authorSHA, memorySession, bindings := legacyAuthorCatalogTrace(events)
	if authorSHA == "" || memorySession == "" || len(bindings) == 0 {
		return nil, fmt.Errorf("legacy report IL author catalog trace is missing")
	}
	candidateSHA, candidateCount := legacyCandidateCatalogTrace(events)
	if candidateSHA == "" {
		candidateSHA = authorSHA
		candidateCount = len(candidateBuild.Catalog.Sources)
	}
	if !strings.EqualFold(candidateSHA, candidateBuild.Catalog.SHA256) ||
		candidateCount != len(candidateBuild.Catalog.Sources) {
		return nil, fmt.Errorf("legacy report IL candidate catalog changed or is incomplete")
	}
	authorCatalog, err := reconstructLegacyAuthorCatalog(candidateBuild, bindings)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(authorCatalog.SHA256, authorSHA) {
		return nil, fmt.Errorf("legacy report IL author catalog changed")
	}
	memory, memoryReceipt, err := documents.ReadReportILEditorialMemory(ctx, missionID, memorySession, authorCatalog)
	if err != nil {
		return nil, fmt.Errorf("legacy report IL editorial memory verification failed: %w", err)
	}
	finalSession := legacyFinalizedStageSession(events, "il_long_form_final", reportilcontract.LongFormDocumentFinalizeTool)
	if finalSession == "" {
		return nil, fmt.Errorf("legacy report IL final manuscript trace is missing")
	}
	authored, workspace, err := documents.ReadReportILLongFormStageDocument(ctx, missionID, finalSession, "il_long_form_final", authorCatalog)
	if err != nil {
		return nil, fmt.Errorf("legacy report IL final manuscript verification failed: %w", err)
	}
	if !legacyStageCompleted(events, pendingID, "report.il_long_form_final.completed") || !legacyReaderFailed(events, pendingID) {
		return nil, fmt.Errorf("legacy report IL failure boundary is not recoverable")
	}
	parts, sections := len(authored.Parts), 0
	for _, part := range authored.Parts {
		sections += len(part.Sections)
	}
	artifact := reportilcontract.CheckpointArtifact{
		ArtifactID: workspace.ArtifactID, SHA256: workspace.SHA256,
		ByteSize: workspace.ByteSize, Stage: workspace.Stage,
	}
	excludedUnusable := len(candidateBuild.Dispositions) - len(candidateBuild.Catalog.Sources)
	excludedBudget := len(candidateBuild.Catalog.Sources) - len(authorCatalog.Sources)
	if excludedUnusable < 0 || excludedBudget < 0 {
		return nil, fmt.Errorf("legacy report IL source selection counts are invalid")
	}
	checkpoint := reportilcontract.ProductCheckpoint{
		SchemaVersion:  reportilcontract.ProductCheckpointSchemaVersion,
		PendingEventID: pendingID, Stage: "il_long_form_final", ArtifactID: workspace.ArtifactID,
		CandidateCatalogSHA256: candidateBuild.Catalog.SHA256, AuthorCatalog: authorCatalog,
		ImageCatalog: candidateBuild.ImageCatalog,
		SourceSelection: reportilcontract.CheckpointSourceSelection{
			Applied:         excludedUnusable+excludedBudget > 0,
			AcceptedSources: len(candidateBuild.Dispositions), UsableSources: len(candidateBuild.Catalog.Sources),
			SelectedSources: len(authorCatalog.Sources), ExcludedUnusableSources: excludedUnusable, ExcludedBudgetSources: excludedBudget,
			CandidateCatalogSHA256: candidateBuild.Catalog.SHA256,
			SelectedCatalogSHA256:  authorCatalog.SHA256, AuthorCatalogSHA256: authorCatalog.SHA256,
		},
		EditorialMemory: reportilcontract.CheckpointEditorialMemory{
			ArtifactID: memoryReceipt.ArtifactID, SHA256: memoryReceipt.SHA256, ByteSize: memoryReceipt.ByteSize,
			Revision: memoryReceipt.Revision, Accounts: memoryReceipt.Accounts,
		},
		AuthorWorkspace: workspace,
		LongFormAuthoring: reportilcontract.CheckpointLongFormAuthoring{
			Final: artifact, Finalizations: []reportilcontract.CheckpointFinalization{{Stage: "il_long_form_final", Artifact: artifact}},
			Parts: parts, Sections: sections, SectionAuthors: sections, PartEditors: parts, FinalEdits: workspace.Replacements,
		},
	}
	if err := reportilcontract.ValidateProductCheckpoint(checkpoint, missionID); err != nil {
		return nil, err
	}
	return &reportilcontract.ResumeCheckpoint{ProductCheckpoint: checkpoint, AuthorDocument: authored, EditorialMemory: memory}, nil
}

func legacyCandidateCatalogTrace(events []ledger.Event) (string, int) {
	catalogSHA := ""
	seen := map[string]bool{}
	for _, event := range events {
		if event.EventType != "mcp.tool.called" {
			continue
		}
		var payload struct {
			ToolName  string `json:"tool_name"`
			Success   bool   `json:"success"`
			IOMetrics struct {
				Stage         string `json:"report_il_stage"`
				CatalogSHA256 string `json:"catalog_sha256"`
				SourceReads   []struct {
					SourceKey string `json:"source_key"`
				} `json:"source_reads"`
			} `json:"io_metrics"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil || !payload.Success ||
			payload.ToolName != reportilcontract.SourceReadTool || payload.IOMetrics.Stage != "il_source_selection" {
			continue
		}
		if catalogSHA != "" && !strings.EqualFold(catalogSHA, payload.IOMetrics.CatalogSHA256) {
			return "", 0
		}
		catalogSHA = payload.IOMetrics.CatalogSHA256
		for _, read := range payload.IOMetrics.SourceReads {
			seen[read.SourceKey] = true
		}
	}
	return catalogSHA, len(seen)
}

func legacyAuthorCatalogTrace(events []ledger.Event) (string, string, []legacySourceBinding) {
	catalogSHA, memorySession := "", ""
	bindings := []legacySourceBinding{}
	byKey := map[string]int{}
	finalizeCount := 0
	for _, event := range events {
		if event.EventType != "mcp.tool.called" {
			continue
		}
		var payload struct {
			ToolName      string `json:"tool_name"`
			ToolSessionID string `json:"tool_session_id"`
			Success       bool   `json:"success"`
			IOMetrics     struct {
				Stage          string `json:"report_il_stage"`
				CatalogSHA256  string `json:"catalog_sha256"`
				Finalized      bool   `json:"finalized"`
				SourceKey      string `json:"source_key"`
				SourceOffset   int    `json:"source_offset"`
				SourceByteSize int    `json:"source_byte_size"`
				SourceSHA256   string `json:"source_sha256"`
				SourceReads    []struct {
					SourceKey     string `json:"source_key"`
					ContentLength int    `json:"content_length"`
				} `json:"source_reads"`
			} `json:"io_metrics"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil || !payload.Success || payload.IOMetrics.Stage != "il_editorial_memory" {
			continue
		}
		if payload.IOMetrics.CatalogSHA256 != "" {
			if catalogSHA != "" && !strings.EqualFold(catalogSHA, payload.IOMetrics.CatalogSHA256) {
				return "", "", nil
			}
			catalogSHA = payload.IOMetrics.CatalogSHA256
		}
		if payload.ToolName == reportilcontract.SourceReadTool {
			for _, read := range payload.IOMetrics.SourceReads {
				if _, exists := byKey[read.SourceKey]; !exists {
					byKey[read.SourceKey] = len(bindings)
					bindings = append(bindings, legacySourceBinding{SourceKey: read.SourceKey, ContentLength: read.ContentLength})
				}
			}
		}
		if payload.ToolName == reportilcontract.SourceQuoteRegisterTool {
			index, exists := byKey[payload.IOMetrics.SourceKey]
			if !exists {
				return "", "", nil
			}
			bindings[index].Quotes = append(bindings[index].Quotes, legacySourceQuote{
				Offset: payload.IOMetrics.SourceOffset, ByteSize: payload.IOMetrics.SourceByteSize, SHA256: payload.IOMetrics.SourceSHA256,
			})
		}
		if payload.ToolName == reportilcontract.EditorialMemoryFinalizeTool && payload.IOMetrics.Finalized {
			finalizeCount++
			memorySession = payload.ToolSessionID
		}
	}
	if finalizeCount != 1 {
		return "", "", nil
	}
	return catalogSHA, memorySession, bindings
}

func reconstructLegacyAuthorCatalog(build SourceCatalogBuild, bindings []legacySourceBinding) (reportilcontract.SourceCatalog, error) {
	entries := make([]reportilcontract.SourceCatalogEntry, 0, len(bindings))
	used := map[int]bool{}
	for _, binding := range bindings {
		matches := []reportilcontract.SourceCatalogEntry{}
		for _, candidate := range build.Catalog.Sources {
			if used[candidate.AcceptedOrdinal] || candidate.ReadableBytes != binding.ContentLength {
				continue
			}
			readable := []byte(build.ReadableByAcceptedOrdinal[candidate.AcceptedOrdinal])
			matched := len(readable) == binding.ContentLength
			for _, quote := range binding.Quotes {
				if quote.Offset < 0 || quote.ByteSize < 1 || quote.Offset > len(readable)-quote.ByteSize ||
					!strings.EqualFold(SHA256(readable[quote.Offset:quote.Offset+quote.ByteSize]), quote.SHA256) {
					matched = false
					break
				}
			}
			if matched {
				matches = append(matches, candidate)
			}
		}
		if len(matches) != 1 {
			return reportilcontract.SourceCatalog{}, fmt.Errorf("legacy report IL source binding %s is ambiguous", binding.SourceKey)
		}
		entry := matches[0]
		used[entry.AcceptedOrdinal] = true
		entry.SourceKey = fmt.Sprintf("source_%03d", len(entries)+1)
		entries = append(entries, entry)
	}
	return reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{MissionID: build.Catalog.MissionID, Sources: entries})
}

func legacyFinalizedStageSession(events []ledger.Event, stage, finalizeTool string) string {
	count, sessionID := 0, ""
	for _, event := range events {
		if event.EventType != "mcp.tool.called" {
			continue
		}
		var payload struct {
			ToolName      string `json:"tool_name"`
			ToolSessionID string `json:"tool_session_id"`
			Success       bool   `json:"success"`
			IOMetrics     struct {
				Stage     string `json:"report_il_stage"`
				Finalized bool   `json:"finalized"`
			} `json:"io_metrics"`
		}
		if json.Unmarshal(event.Payload, &payload) == nil && payload.Success && payload.IOMetrics.Finalized &&
			payload.ToolName == finalizeTool && payload.IOMetrics.Stage == stage {
			count++
			sessionID = payload.ToolSessionID
		}
	}
	if count != 1 {
		return ""
	}
	return sessionID
}

func legacyStageCompleted(events []ledger.Event, pendingID, eventType string) bool {
	for _, event := range events {
		if event.EventType != eventType {
			continue
		}
		var payload struct {
			PendingID string `json:"pending_event_id"`
		}
		if json.Unmarshal(event.Payload, &payload) == nil && payload.PendingID == pendingID {
			return true
		}
	}
	return false
}

func legacyReaderFailed(events []ledger.Event, pendingID string) bool {
	for _, event := range events {
		if event.EventType != "report.draft.failed" {
			continue
		}
		var payload struct {
			PendingID string `json:"pending_event_id"`
			Stage     string `json:"failed_stage_kind"`
		}
		if json.Unmarshal(event.Payload, &payload) == nil && payload.PendingID == pendingID && payload.Stage == "il_reader" {
			return true
		}
	}
	return false
}

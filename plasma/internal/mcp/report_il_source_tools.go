package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportilsource"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

func (server *Server) callReportILSourcesList(call ToolCall) ToolResult {
	var input reportILSourcesListInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, server.binding.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.reportILSourceAccessBinding()
	if err != nil {
		return errorResult(call.Name, server.binding.MissionID, "binding", err.Error(), false, nil)
	}
	items := make([]reportILSourceListItem, 0, len(binding.Catalog.Sources))
	for _, entry := range binding.Catalog.Sources {
		items = append(items, reportILSourceListItem{
			SourceKey:     entry.SourceKey,
			ReadableBytes: entry.ReadableBytes,
			Extraction:    entry.Extraction,
		})
	}
	return ToolResult{
		ToolName:  call.Name,
		MissionID: binding.Catalog.MissionID,
		Content: reportILSourcesListOutput{
			CatalogSHA256: binding.Catalog.SHA256,
			Stage:         binding.Stage,
			Attempt:       binding.Attempt,
			Sources:       items,
		},
	}
}

func (server *Server) callReportILSourceQuote(ctx context.Context, call ToolCall) ToolResult {
	binding, err := server.reportILSourceAccessBinding()
	if err != nil {
		return errorResult(call.Name, server.binding.MissionID, "binding", err.Error(), false, nil)
	}
	if binding.Stage != "il_narrative" && binding.Stage != "il_editorial_memory" {
		return errorResult(call.Name, server.binding.MissionID, "validation", "source quote registration is unavailable for this report IL stage", false, nil)
	}
	var input reportILSourceQuoteInput
	if !utf8.Valid(call.Arguments) || decodeArgs(call.Arguments, &input) != nil {
		return errorResult(call.Name, server.binding.MissionID, "validation", "source quote registration input is invalid", false, nil)
	}
	entry, ok := binding.Catalog.Entry(input.SourceKey)
	quoteBytes := []byte(input.Quote)
	if !ok || strings.TrimSpace(input.Quote) == "" || len(quoteBytes) > reportilcontract.MaxSourceQuoteBytes ||
		!utf8.Valid(quoteBytes) {
		return errorResult(call.Name, server.binding.MissionID, "validation", "source quote registration input is invalid", false, nil)
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	for _, source := range binding.Catalog.Sources {
		if !server.reportILSourceComplete[source.SourceKey] {
			return errorResult(call.Name, server.binding.MissionID, "validation", "source quote registration requires the complete frozen catalog read", false, nil)
		}
	}
	if server.reportILSourceQuoteCount >= reportilcontract.MaxSourceReadSpans {
		return errorResult(call.Name, server.binding.MissionID, "validation", "source quote registration exceeds the attempt ceiling", false, nil)
	}
	readable, err := server.reportILReadableSource(ctx, binding, entry)
	if err != nil {
		return errorResult(call.Name, server.binding.MissionID, "validation", "frozen report IL source validation failed", false, nil)
	}
	offset := strings.Index(readable.Text, input.Quote)
	if offset < 0 {
		return errorResult(call.Name, server.binding.MissionID, "validation", "source quote is not an exact frozen-source excerpt", false, nil)
	}
	server.reportILSourceQuoteCount++
	quoteSHA := sha256Hex(quoteBytes)
	receipt := reportilcontract.SourceQuoteReceiptID(
		server.binding.AgentSessionID, binding.Catalog.SHA256, binding.Stage,
		server.reportILSourceQuoteCount, entry.SourceKey, offset, len(quoteBytes), quoteSHA,
	)
	registered := reportilcontract.SourceQuoteReceipt{
		Receipt: receipt, SourceKey: entry.SourceKey, Offset: offset,
		ByteSize: len(quoteBytes), SHA256: quoteSHA,
	}
	server.reportILSourceQuotes[receipt] = registered
	return ToolResult{
		ToolName: call.Name, MissionID: binding.Catalog.MissionID,
		Content: reportILSourceQuoteOutput{
			SourceReceipt: receipt,
			sourceKey:     entry.SourceKey, offset: offset,
			byteSize: len(quoteBytes), sha256: quoteSHA,
			catalogSHA256: binding.Catalog.SHA256, stage: binding.Stage,
		},
	}
}

func (server *Server) callReportILSourcesRead(ctx context.Context, call ToolCall) ToolResult {
	binding, err := server.reportILSourceAccessBinding()
	if err != nil {
		return errorResult(call.Name, server.binding.MissionID, "binding", err.Error(), false, nil)
	}
	if reportILServerOrderedReadStage(binding.Stage) {
		var input map[string]json.RawMessage
		if err := decodeArgs(call.Arguments, &input); err != nil {
			return errorResult(call.Name, server.binding.MissionID, "validation", err.Error(), false, nil)
		}
		if input == nil || len(input) != 0 {
			return errorResult(call.Name, server.binding.MissionID, "validation", "server-ordered batch read accepts only empty object arguments", false, nil)
		}
		if binding.Stage == "il_flow" && len(binding.ReadSpans) > 0 {
			return server.callReportILFlowSpanRead(ctx, call, binding)
		}
		return server.callReportILServerOrderedBatchRead(ctx, call, binding)
	}
	var input reportILSourcesReadInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, server.binding.MissionID, "validation", err.Error(), false, nil)
	}
	entry, ok := binding.Catalog.Entry(input.SourceKey)
	if !ok || binding.Stage == "il_long_form_section" && !reportILStringInventoryContains(binding.LongFormSourceKeys, input.SourceKey) {
		return errorResult(call.Name, server.binding.MissionID, "validation", "source key is not available to this report IL attempt", false, nil)
	}
	if input.Offset < 0 || input.Offset > entry.ReadableBytes {
		return errorResult(call.Name, server.binding.MissionID, "validation", "source offset is outside the frozen readable content", false, nil)
	}
	if input.MaxBytes < 1 || input.MaxBytes > binding.MaxCallBytes {
		return errorResult(call.Name, server.binding.MissionID, "validation", "source read exceeds the per-call byte ceiling", false, nil)
	}
	if binding.MaxSourceReadBytes > 0 && input.MaxBytes != binding.MaxSourceReadBytes {
		return errorResult(call.Name, server.binding.MissionID, "validation", "source selection read must use the exact per-source sample ceiling", false, nil)
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.reportILSourceComplete[input.SourceKey] {
		return errorResult(call.Name, server.binding.MissionID, "validation", "source is already completely read in this attempt", false, nil)
	}
	expectedOffset := server.reportILSourceNextOffsets[input.SourceKey]
	if input.Offset != expectedOffset {
		return errorResult(call.Name, server.binding.MissionID, "validation", fmt.Sprintf("source reads must start at offset 0 and continue at exact next_offset %d", expectedOffset), false, nil)
	}
	remainingAttemptBytes := binding.MaxReadBytes - server.reportILSourceReadBytes
	if remainingAttemptBytes < 1 {
		return errorResult(call.Name, server.binding.MissionID, "validation", "source read exceeds the attempt byte ceiling", false, nil)
	}
	remainingSourceBytes := entry.ReadableBytes - server.reportILSourceReadBySource[input.SourceKey]
	if binding.MaxSourceReadBytes > 0 {
		remainingSourceBytes = binding.MaxSourceReadBytes - server.reportILSourceReadBySource[input.SourceKey]
	}
	if remainingSourceBytes < 1 {
		return errorResult(call.Name, server.binding.MissionID, "validation", "source read exceeds the per-source byte ceiling", false, nil)
	}
	readable, err := server.reportILReadableSource(ctx, binding, entry)
	if err != nil {
		return errorResult(call.Name, server.binding.MissionID, "validation", "frozen report IL source validation failed", false, nil)
	}
	callLimit := binding.MaxCallBytes
	if remainingAttemptBytes < callLimit {
		callLimit = remainingAttemptBytes
	}
	if remainingSourceBytes < callLimit {
		callLimit = remainingSourceBytes
	}
	content, offset, nextOffset, truncated, err := boundedArtifactContentWithLimit(
		[]byte(readable.Text),
		input.Offset,
		input.MaxBytes,
		callLimit,
	)
	if err != nil {
		return errorResult(call.Name, server.binding.MissionID, "validation", err.Error(), false, nil)
	}
	returnedBytes := len([]byte(content))
	if server.reportILSourceReadBytes+returnedBytes > binding.MaxReadBytes {
		return errorResult(call.Name, server.binding.MissionID, "validation", "source read exceeds the attempt byte ceiling", false, nil)
	}
	server.reportILSourceReadBytes += returnedBytes
	server.reportILSourceReadBySource[input.SourceKey] += returnedBytes
	if binding.MaxSourceReadBytes > 0 {
		server.reportILSourceComplete[input.SourceKey] = true
		delete(server.reportILSourceNextOffsets, input.SourceKey)
	} else if truncated {
		server.reportILSourceNextOffsets[input.SourceKey] = nextOffset
	} else {
		server.reportILSourceComplete[input.SourceKey] = true
		delete(server.reportILSourceNextOffsets, input.SourceKey)
	}
	return ToolResult{
		ToolName:  call.Name,
		MissionID: binding.Catalog.MissionID,
		Content: reportILSourcesReadOutput{
			SourceKey:        entry.SourceKey,
			CatalogSHA256:    binding.Catalog.SHA256,
			Stage:            binding.Stage,
			Content:          content,
			Offset:           offset,
			NextOffset:       nextOffset,
			ContentLength:    readable.ByteSize,
			Truncated:        truncated,
			Extraction:       readable.Extraction,
			AttemptReadBytes: server.reportILSourceReadBytes,
			AttemptMaxBytes:  binding.MaxReadBytes,
		},
	}
}

const reportILSourceBatchItems = 32

func reportILServerOrderedReadStage(stage string) bool {
	return stage == "il_source_selection" || stage == "il_editorial_memory" || stage == "il_flow"
}

func (server *Server) callReportILFlowSpanRead(
	ctx context.Context,
	call ToolCall,
	binding reportilcontract.SourceAccessBinding,
) ToolResult {
	server.mu.Lock()
	defer server.mu.Unlock()
	items := make([]reportILSourceBatchItem, 0)
	returnedBytes := 0
	for index, span := range binding.ReadSpans {
		spanKey := fmt.Sprintf("span_%03d", index+1)
		if server.reportILSourceComplete[spanKey] {
			continue
		}
		if len(items) >= reportILSourceBatchItems || returnedBytes+span.ByteSize > binding.MaxCallBytes {
			break
		}
		entry, ok := binding.Catalog.Entry(span.SourceKey)
		if !ok {
			return errorResult(call.Name, server.binding.MissionID, "validation", "flow source span is unavailable", false, nil)
		}
		readable, err := server.reportILReadableSource(ctx, binding, entry)
		if err != nil {
			return errorResult(call.Name, server.binding.MissionID, "validation", "frozen report IL source validation failed", false, nil)
		}
		content, offset, nextOffset, truncated, err := boundedArtifactContentWithLimit(
			[]byte(readable.Text), span.Offset, span.ByteSize, span.ByteSize,
		)
		if err != nil || offset != span.Offset || len([]byte(content)) != span.ByteSize ||
			sha256Hex([]byte(content)) != span.SHA256 {
			return errorResult(call.Name, server.binding.MissionID, "validation", "flow source span validation failed", false, nil)
		}
		items = append(items, reportILSourceBatchItem{
			SourceKey: entry.SourceKey, Content: content, Offset: offset,
			NextOffset: nextOffset, ContentLength: readable.ByteSize,
			Truncated: truncated, Extraction: readable.Extraction,
		})
		returnedBytes += span.ByteSize
		server.reportILSourceComplete[spanKey] = true
		server.reportILSourceReadBySource[entry.SourceKey] += span.ByteSize
	}
	if len(items) == 0 {
		return errorResult(call.Name, server.binding.MissionID, "validation", "all server-ordered source content was already read", false, nil)
	}
	server.reportILSourceReadBytes += returnedBytes
	remaining := 0
	for index := range binding.ReadSpans {
		if !server.reportILSourceComplete[fmt.Sprintf("span_%03d", index+1)] {
			remaining++
		}
	}
	return ToolResult{
		ToolName: call.Name, MissionID: binding.Catalog.MissionID,
		Content: reportILSourcesBatchReadOutput{
			CatalogSHA256: binding.Catalog.SHA256, Stage: binding.Stage,
			Sources: items, RemainingSources: remaining,
			AttemptReadBytes: server.reportILSourceReadBytes,
			AttemptMaxBytes:  binding.MaxReadBytes,
		},
	}
}

func reportILSourceRuneFits(text string, offset, maximumBytes int) bool {
	if maximumBytes < 1 || offset < 0 || offset >= len(text) {
		return false
	}
	_, runeBytes := utf8.DecodeRuneInString(text[offset:])
	return runeBytes <= maximumBytes
}

func (server *Server) callReportILServerOrderedBatchRead(
	ctx context.Context,
	call ToolCall,
	binding reportilcontract.SourceAccessBinding,
) ToolResult {
	server.mu.Lock()
	defer server.mu.Unlock()

	items := make([]reportILSourceBatchItem, 0)
	returnedBytes := 0
	for _, entry := range binding.Catalog.Sources {
		if server.reportILSourceComplete[entry.SourceKey] {
			continue
		}
		if len(items) >= reportILSourceBatchItems {
			break
		}
		offset := server.reportILSourceNextOffsets[entry.SourceKey]
		maximumItemBytes := entry.ReadableBytes - offset
		if binding.MaxSourceReadBytes > 0 && maximumItemBytes > binding.MaxSourceReadBytes {
			maximumItemBytes = binding.MaxSourceReadBytes
		}
		remainingCallBytes := binding.MaxCallBytes - returnedBytes
		if binding.MaxSourceReadBytes > 0 && maximumItemBytes > remainingCallBytes && len(items) > 0 {
			break
		}
		if maximumItemBytes > remainingCallBytes {
			maximumItemBytes = remainingCallBytes
		}
		remainingAttemptBytes := binding.MaxReadBytes - server.reportILSourceReadBytes - returnedBytes
		if maximumItemBytes > remainingAttemptBytes {
			maximumItemBytes = remainingAttemptBytes
		}
		if maximumItemBytes < 1 {
			break
		}
		readable, err := server.reportILReadableSource(ctx, binding, entry)
		if err != nil {
			return errorResult(call.Name, server.binding.MissionID, "validation", "frozen report IL source validation failed", false, nil)
		}
		if len(items) > 0 && !reportILSourceRuneFits(readable.Text, offset, maximumItemBytes) {
			break
		}
		content, returnedOffset, nextOffset, truncated, err := boundedArtifactContentWithLimit(
			[]byte(readable.Text),
			offset,
			maximumItemBytes,
			maximumItemBytes,
		)
		if err != nil {
			return errorResult(call.Name, server.binding.MissionID, "validation", err.Error(), false, nil)
		}
		contentBytes := len([]byte(content))
		if contentBytes < 1 || returnedBytes+contentBytes > binding.MaxCallBytes ||
			server.reportILSourceReadBytes+returnedBytes+contentBytes > binding.MaxReadBytes {
			return errorResult(call.Name, server.binding.MissionID, "validation", "server-ordered source batch exceeds the frozen read budget", false, nil)
		}
		items = append(items, reportILSourceBatchItem{
			SourceKey:     entry.SourceKey,
			Content:       content,
			Offset:        returnedOffset,
			NextOffset:    nextOffset,
			ContentLength: readable.ByteSize,
			Truncated:     truncated,
			Extraction:    readable.Extraction,
		})
		returnedBytes += contentBytes
		if binding.MaxSourceReadBytes == 0 && truncated {
			break
		}
		if returnedBytes >= binding.MaxCallBytes {
			break
		}
	}
	if len(items) == 0 {
		return errorResult(call.Name, server.binding.MissionID, "validation", "all server-ordered source content was already read", false, nil)
	}
	for _, item := range items {
		contentBytes := len([]byte(item.Content))
		server.reportILSourceReadBySource[item.SourceKey] += contentBytes
		if binding.MaxSourceReadBytes > 0 || !item.Truncated {
			server.reportILSourceComplete[item.SourceKey] = true
			delete(server.reportILSourceNextOffsets, item.SourceKey)
		} else {
			server.reportILSourceNextOffsets[item.SourceKey] = item.NextOffset
		}
	}
	server.reportILSourceReadBytes += returnedBytes
	remaining := 0
	for _, entry := range binding.Catalog.Sources {
		if !server.reportILSourceComplete[entry.SourceKey] {
			remaining++
		}
	}
	return ToolResult{
		ToolName:  call.Name,
		MissionID: binding.Catalog.MissionID,
		Content: reportILSourcesBatchReadOutput{
			CatalogSHA256:    binding.Catalog.SHA256,
			Stage:            binding.Stage,
			Sources:          items,
			RemainingSources: remaining,
			AttemptReadBytes: server.reportILSourceReadBytes,
			AttemptMaxBytes:  binding.MaxReadBytes,
		},
	}
}

func (server *Server) reportILSourceAccessBinding() (reportilcontract.SourceAccessBinding, error) {
	if !server.reportILSourceBindingSet {
		return reportilcontract.SourceAccessBinding{}, fmt.Errorf("report IL source binding is unavailable")
	}
	binding := server.reportILSourceBinding
	if err := reportilcontract.ValidateSourceAccessBinding(binding); err != nil {
		return reportilcontract.SourceAccessBinding{}, err
	}
	if strings.TrimSpace(server.binding.MissionID) != binding.Catalog.MissionID {
		return reportilcontract.SourceAccessBinding{}, fmt.Errorf("report IL source binding conflicts with mission binding")
	}
	if !strings.HasPrefix(strings.TrimSpace(server.binding.AgentSessionID), "ses_") || server.binding.AgentExecutor != "codex" {
		return reportilcontract.SourceAccessBinding{}, fmt.Errorf("report IL source binding requires a Codex attempt session")
	}
	return binding, nil
}

func (server *Server) reportILReadableSource(ctx context.Context, binding reportilcontract.SourceAccessBinding, entry reportilcontract.SourceCatalogEntry) (reportilsource.Readable, error) {
	snapshot, err := server.service.GetSourceSnapshot(ctx, entry.SnapshotID)
	if err != nil {
		return reportilsource.Readable{}, err
	}
	if snapshot.MissionID != binding.Catalog.MissionID || sourceState(snapshot).Removed || sourceState(snapshot).Superseded {
		return reportilsource.Readable{}, fmt.Errorf("frozen report IL source is no longer active")
	}
	retrievalPolicy := strings.TrimSpace(snapshot.Access.RetrievalPolicy)
	if retrievalPolicy == "" {
		retrievalPolicy = source.RetrievalPolicySnapshotOnly
	}
	if retrievalPolicy != entry.RetrievalPolicy {
		return reportilsource.Readable{}, fmt.Errorf("frozen report IL source retrieval policy changed")
	}
	if sourceReceiptFromApp(snapshot) != entry.SnapshotReceipt || !strings.EqualFold(snapshot.ContentHash.Value, entry.ContentHash) {
		return reportilsource.Readable{}, fmt.Errorf("frozen report IL source identity changed")
	}
	var readable reportilsource.Readable
	if entry.RetrievalPolicy == source.RetrievalPolicyLiveReference {
		result, err := server.service.ReadLocalPathSource(ctx, newReportILLiveSourceReadRequest(
			binding.Catalog.MissionID,
			entry.SnapshotID,
			server.binding.AgentSessionID,
			reportilsource.MaxReadableBytes,
		))
		if err != nil {
			return reportilsource.Readable{}, err
		}
		if result.Read.Metadata.Binary || result.Read.Metadata.Truncated {
			return reportilsource.Readable{}, fmt.Errorf("frozen live report IL source cannot be read as complete text")
		}
		readable, err = reportilsource.Extract([]byte(result.Read.Content), "text/plain")
		if err != nil {
			return reportilsource.Readable{}, err
		}
		if strings.TrimSpace(result.Read.Metadata.Extraction) != "" {
			readable.Extraction = result.Read.Metadata.Extraction
		} else {
			readable.Extraction = "live_text"
		}
	} else {
		var combined strings.Builder
		if len(entry.Artifacts) == 0 || len(entry.Artifacts) != len(snapshot.ArtifactIDs) {
			return reportilsource.Readable{}, fmt.Errorf("frozen report IL artifact set changed")
		}
		for index, receipt := range entry.Artifacts {
			if snapshot.ArtifactIDs[index] != receipt.ArtifactID {
				return reportilsource.Readable{}, fmt.Errorf("frozen report IL artifact order changed")
			}
			artifact, err := server.service.GetRawArtifact(ctx, receipt.ArtifactID)
			if err != nil {
				return reportilsource.Readable{}, err
			}
			if artifact.MissionID != binding.Catalog.MissionID || !strings.EqualFold(artifact.SHA256, receipt.SHA256) || !strings.EqualFold(sha256Hex(artifact.Content), receipt.SHA256) || artifact.ByteSize != receipt.ByteSize || int64(len(artifact.Content)) != receipt.ByteSize || artifact.MediaType != receipt.MediaType {
				return reportilsource.Readable{}, fmt.Errorf("frozen report IL artifact receipt changed")
			}
			part, err := reportilsource.Extract(artifact.Content, artifact.MediaType)
			if err != nil {
				return reportilsource.Readable{}, err
			}
			if combined.Len() > 0 {
				combined.WriteString("\n\n")
			}
			combined.WriteString(part.Text)
			if len(entry.Artifacts) == 1 {
				readable.Extraction = part.Extraction
			}
		}
		readable, err = reportilsource.Extract([]byte(combined.String()), "text/plain")
		if err != nil {
			return reportilsource.Readable{}, err
		}
		if len(entry.Artifacts) == 1 {
			part, partErr := server.service.GetRawArtifact(ctx, entry.Artifacts[0].ArtifactID)
			if partErr != nil {
				return reportilsource.Readable{}, partErr
			}
			partReadable, partErr := reportilsource.Extract(part.Content, part.MediaType)
			if partErr != nil {
				return reportilsource.Readable{}, partErr
			}
			readable.Extraction = partReadable.Extraction
		} else {
			readable.Extraction = "joined_artifact_text"
		}
	}
	if readable.SHA256 != entry.ReadableSHA256 || readable.ByteSize != entry.ReadableBytes || readable.Extraction != entry.Extraction {
		return reportilsource.Readable{}, fmt.Errorf("frozen report IL readable content changed")
	}
	return readable, nil
}

func sourceReceiptFromApp(snapshot source.Snapshot) string {
	return reportilcontract.SourceSnapshotReceipt(snapshot.SnapshotID, snapshot.ContentHash.Value)
}

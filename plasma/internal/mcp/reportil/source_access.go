package reportil

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

func (server *SourceHandler) CallReportILSourcesList(call wire.ToolCall) wire.ToolResult {
	var input ReportILSourcesListInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", err.Error(), false, nil)
	}
	binding, err := server.AccessBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	items := make([]ReportILSourceListItem, 0, len(binding.Catalog.Sources))
	for _, entry := range binding.Catalog.Sources {
		items = append(items, ReportILSourceListItem{
			SourceKey:     entry.SourceKey,
			ReadableBytes: entry.ReadableBytes,
			Extraction:    entry.Extraction,
		})
	}
	return wire.ToolResult{
		ToolName:  call.Name,
		MissionID: binding.Catalog.MissionID,
		Content: ReportILSourcesListOutput{
			CatalogSHA256: binding.Catalog.SHA256,
			Stage:         binding.Stage,
			Attempt:       binding.Attempt,
			Sources:       items,
		},
	}
}

func (server *SourceHandler) CallReportILSourceQuote(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	binding, err := server.AccessBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	if binding.Stage != "il_narrative" && binding.Stage != "il_editorial_memory" {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "source quote registration is unavailable for this report IL stage", false, nil)
	}
	var input ReportILSourceQuoteInput
	if !utf8.Valid(call.Arguments) || server.Decode(call.Arguments, &input) != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "source quote registration input is invalid", false, nil)
	}
	entry, ok := binding.Catalog.Entry(input.SourceKey)
	quoteBytes := []byte(input.Quote)
	if !ok || strings.TrimSpace(input.Quote) == "" || len(quoteBytes) > reportilcontract.MaxSourceQuoteBytes ||
		!utf8.Valid(quoteBytes) {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "source quote registration input is invalid", false, nil)
	}

	server.Mu.Lock()
	defer server.Mu.Unlock()
	for _, source := range binding.Catalog.Sources {
		if !server.State.Complete[source.SourceKey] {
			return server.ErrorResult(call.Name, server.MissionID(), "validation", "source quote registration requires the complete frozen catalog read", false, nil)
		}
	}
	if server.State.QuoteCount >= reportilcontract.MaxSourceReadSpans {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "source quote registration exceeds the attempt ceiling", false, nil)
	}
	readable, err := server.ReadableSource(ctx, binding, entry)
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "frozen report IL source validation failed", false, nil)
	}
	offset := strings.Index(readable.Text, input.Quote)
	if offset < 0 {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "source quote is not an exact frozen-source excerpt", false, nil)
	}
	server.State.QuoteCount++
	quoteSHA := server.SHA256(quoteBytes)
	receipt := reportilcontract.SourceQuoteReceiptID(
		server.SessionID(), binding.Catalog.SHA256, binding.Stage,
		server.State.QuoteCount, entry.SourceKey, offset, len(quoteBytes), quoteSHA,
	)
	registered := reportilcontract.SourceQuoteReceipt{
		Receipt: receipt, SourceKey: entry.SourceKey, Offset: offset,
		ByteSize: len(quoteBytes), SHA256: quoteSHA,
	}
	server.State.Quotes[receipt] = registered
	return wire.ToolResult{
		ToolName: call.Name, MissionID: binding.Catalog.MissionID,
		Content: ReportILSourceQuoteOutput{
			SourceReceipt: receipt,
			SourceKey:     entry.SourceKey, Offset: offset,
			ByteSize: len(quoteBytes), Sha256: quoteSHA,
			CatalogSHA256: binding.Catalog.SHA256, Stage: binding.Stage,
		},
	}
}

func (server *SourceHandler) CallReportILSourcesRead(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	binding, err := server.AccessBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	if reportILServerOrderedReadStage(binding.Stage) {
		var input map[string]json.RawMessage
		if err := server.Decode(call.Arguments, &input); err != nil {
			return server.ErrorResult(call.Name, server.MissionID(), "validation", err.Error(), false, nil)
		}
		if input == nil || len(input) != 0 {
			return server.ErrorResult(call.Name, server.MissionID(), "validation", "server-ordered batch read accepts only empty object arguments", false, nil)
		}
		if binding.Stage == "il_flow" && len(binding.ReadSpans) > 0 {
			return server.CallReportILFlowSpanRead(ctx, call, binding)
		}
		return server.CallReportILServerOrderedBatchRead(ctx, call, binding)
	}
	var input ReportILSourcesReadInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", err.Error(), false, nil)
	}
	entry, ok := binding.Catalog.Entry(input.SourceKey)
	if !ok || binding.Stage == "il_long_form_section" && !reportILStringInventoryContains(binding.LongFormSourceKeys, input.SourceKey) {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "source key is not available to this report IL attempt", false, nil)
	}
	if input.Offset < 0 || input.Offset > entry.ReadableBytes {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "source offset is outside the frozen readable content", false, nil)
	}
	if input.MaxBytes < 1 || input.MaxBytes > binding.MaxCallBytes {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "source read exceeds the per-call byte ceiling", false, nil)
	}
	if binding.MaxSourceReadBytes > 0 && input.MaxBytes != binding.MaxSourceReadBytes {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "source selection read must use the exact per-source sample ceiling", false, nil)
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	if server.State.Complete[input.SourceKey] {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "source is already completely read in this attempt", false, nil)
	}
	expectedOffset := server.State.NextOffsets[input.SourceKey]
	if input.Offset != expectedOffset {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", fmt.Sprintf("source reads must start at offset 0 and continue at exact next_offset %d", expectedOffset), false, nil)
	}
	remainingAttemptBytes := binding.MaxReadBytes - server.State.ReadBytes
	if remainingAttemptBytes < 1 {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "source read exceeds the attempt byte ceiling", false, nil)
	}
	remainingSourceBytes := entry.ReadableBytes - server.State.ReadBySource[input.SourceKey]
	if binding.MaxSourceReadBytes > 0 {
		remainingSourceBytes = binding.MaxSourceReadBytes - server.State.ReadBySource[input.SourceKey]
	}
	if remainingSourceBytes < 1 {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "source read exceeds the per-source byte ceiling", false, nil)
	}
	readable, err := server.ReadableSource(ctx, binding, entry)
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "frozen report IL source validation failed", false, nil)
	}
	callLimit := binding.MaxCallBytes
	if remainingAttemptBytes < callLimit {
		callLimit = remainingAttemptBytes
	}
	if remainingSourceBytes < callLimit {
		callLimit = remainingSourceBytes
	}
	content, offset, nextOffset, truncated, err := server.BoundedRead(
		[]byte(readable.Text),
		input.Offset,
		input.MaxBytes,
		callLimit,
	)
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", err.Error(), false, nil)
	}
	returnedBytes := len([]byte(content))
	if server.State.ReadBytes+returnedBytes > binding.MaxReadBytes {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "source read exceeds the attempt byte ceiling", false, nil)
	}
	server.State.ReadBytes += returnedBytes
	server.State.ReadBySource[input.SourceKey] += returnedBytes
	if binding.MaxSourceReadBytes > 0 {
		server.State.Complete[input.SourceKey] = true
		delete(server.State.NextOffsets, input.SourceKey)
	} else if truncated {
		server.State.NextOffsets[input.SourceKey] = nextOffset
	} else {
		server.State.Complete[input.SourceKey] = true
		delete(server.State.NextOffsets, input.SourceKey)
	}
	return wire.ToolResult{
		ToolName:  call.Name,
		MissionID: binding.Catalog.MissionID,
		Content: ReportILSourcesReadOutput{
			SourceKey:        entry.SourceKey,
			CatalogSHA256:    binding.Catalog.SHA256,
			Stage:            binding.Stage,
			Content:          content,
			Offset:           offset,
			NextOffset:       nextOffset,
			ContentLength:    readable.ByteSize,
			Truncated:        truncated,
			Extraction:       readable.Extraction,
			AttemptReadBytes: server.State.ReadBytes,
			AttemptMaxBytes:  binding.MaxReadBytes,
		},
	}
}

const reportILSourceBatchItems = 32

func reportILServerOrderedReadStage(stage string) bool {
	return stage == "il_source_selection" || stage == "il_editorial_memory" || stage == "il_flow"
}

func (server *SourceHandler) CallReportILFlowSpanRead(
	ctx context.Context,
	call wire.ToolCall,
	binding reportilcontract.SourceAccessBinding,
) wire.ToolResult {
	server.Mu.Lock()
	defer server.Mu.Unlock()
	items := make([]ReportILSourceBatchItem, 0)
	returnedBytes := 0
	for index, span := range binding.ReadSpans {
		spanKey := fmt.Sprintf("span_%03d", index+1)
		if server.State.Complete[spanKey] {
			continue
		}
		if len(items) >= reportILSourceBatchItems || returnedBytes+span.ByteSize > binding.MaxCallBytes {
			break
		}
		entry, ok := binding.Catalog.Entry(span.SourceKey)
		if !ok {
			return server.ErrorResult(call.Name, server.MissionID(), "validation", "flow source span is unavailable", false, nil)
		}
		readable, err := server.ReadableSource(ctx, binding, entry)
		if err != nil {
			return server.ErrorResult(call.Name, server.MissionID(), "validation", "frozen report IL source validation failed", false, nil)
		}
		content, offset, nextOffset, truncated, err := server.BoundedRead(
			[]byte(readable.Text), span.Offset, span.ByteSize, span.ByteSize,
		)
		if err != nil || offset != span.Offset || len([]byte(content)) != span.ByteSize ||
			server.SHA256([]byte(content)) != span.SHA256 {
			return server.ErrorResult(call.Name, server.MissionID(), "validation", "flow source span validation failed", false, nil)
		}
		items = append(items, ReportILSourceBatchItem{
			SourceKey: entry.SourceKey, Content: content, Offset: offset,
			NextOffset: nextOffset, ContentLength: readable.ByteSize,
			Truncated: truncated, Extraction: readable.Extraction,
		})
		returnedBytes += span.ByteSize
		server.State.Complete[spanKey] = true
		server.State.ReadBySource[entry.SourceKey] += span.ByteSize
	}
	if len(items) == 0 {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "all server-ordered source content was already read", false, nil)
	}
	server.State.ReadBytes += returnedBytes
	remaining := 0
	for index := range binding.ReadSpans {
		if !server.State.Complete[fmt.Sprintf("span_%03d", index+1)] {
			remaining++
		}
	}
	return wire.ToolResult{
		ToolName: call.Name, MissionID: binding.Catalog.MissionID,
		Content: ReportILSourcesBatchReadOutput{
			CatalogSHA256: binding.Catalog.SHA256, Stage: binding.Stage,
			Sources: items, RemainingSources: remaining,
			AttemptReadBytes: server.State.ReadBytes,
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

func (server *SourceHandler) CallReportILServerOrderedBatchRead(
	ctx context.Context,
	call wire.ToolCall,
	binding reportilcontract.SourceAccessBinding,
) wire.ToolResult {
	server.Mu.Lock()
	defer server.Mu.Unlock()

	items := make([]ReportILSourceBatchItem, 0)
	returnedBytes := 0
	for _, entry := range binding.Catalog.Sources {
		if server.State.Complete[entry.SourceKey] {
			continue
		}
		if len(items) >= reportILSourceBatchItems {
			break
		}
		offset := server.State.NextOffsets[entry.SourceKey]
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
		remainingAttemptBytes := binding.MaxReadBytes - server.State.ReadBytes - returnedBytes
		if maximumItemBytes > remainingAttemptBytes {
			maximumItemBytes = remainingAttemptBytes
		}
		if maximumItemBytes < 1 {
			break
		}
		readable, err := server.ReadableSource(ctx, binding, entry)
		if err != nil {
			return server.ErrorResult(call.Name, server.MissionID(), "validation", "frozen report IL source validation failed", false, nil)
		}
		if len(items) > 0 && !reportILSourceRuneFits(readable.Text, offset, maximumItemBytes) {
			break
		}
		content, returnedOffset, nextOffset, truncated, err := server.BoundedRead(
			[]byte(readable.Text),
			offset,
			maximumItemBytes,
			maximumItemBytes,
		)
		if err != nil {
			return server.ErrorResult(call.Name, server.MissionID(), "validation", err.Error(), false, nil)
		}
		contentBytes := len([]byte(content))
		if contentBytes < 1 || returnedBytes+contentBytes > binding.MaxCallBytes ||
			server.State.ReadBytes+returnedBytes+contentBytes > binding.MaxReadBytes {
			return server.ErrorResult(call.Name, server.MissionID(), "validation", "server-ordered source batch exceeds the frozen read budget", false, nil)
		}
		items = append(items, ReportILSourceBatchItem{
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
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "all server-ordered source content was already read", false, nil)
	}
	for _, item := range items {
		contentBytes := len([]byte(item.Content))
		server.State.ReadBySource[item.SourceKey] += contentBytes
		if binding.MaxSourceReadBytes > 0 || !item.Truncated {
			server.State.Complete[item.SourceKey] = true
			delete(server.State.NextOffsets, item.SourceKey)
		} else {
			server.State.NextOffsets[item.SourceKey] = item.NextOffset
		}
	}
	server.State.ReadBytes += returnedBytes
	remaining := 0
	for _, entry := range binding.Catalog.Sources {
		if !server.State.Complete[entry.SourceKey] {
			remaining++
		}
	}
	return wire.ToolResult{
		ToolName:  call.Name,
		MissionID: binding.Catalog.MissionID,
		Content: ReportILSourcesBatchReadOutput{
			CatalogSHA256:    binding.Catalog.SHA256,
			Stage:            binding.Stage,
			Sources:          items,
			RemainingSources: remaining,
			AttemptReadBytes: server.State.ReadBytes,
			AttemptMaxBytes:  binding.MaxReadBytes,
		},
	}
}

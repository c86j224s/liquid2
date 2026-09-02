package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

// VerifyReportILSourceRead confirms the exact stage and forward-only content
// reads made by one request-bound provider attempt. The trace contains only
// source keys and byte metrics, never source content.
func (s *Service) VerifyReportILSourceRead(ctx context.Context, missionID, toolSessionID, expectedStage string, catalog reportilcontract.SourceCatalog) (reportilcontract.SourceReadReceipt, error) {
	missionID = strings.TrimSpace(missionID)
	toolSessionID = strings.TrimSpace(toolSessionID)
	expectedStage = strings.TrimSpace(expectedStage)
	if !strings.HasPrefix(missionID, "mis_") ||
		!strings.HasPrefix(toolSessionID, "ses_") ||
		catalog.MissionID != missionID ||
		!validReportILSourceReadStage(expectedStage) {
		return reportilcontract.SourceReadReceipt{}, fmt.Errorf("report IL source read verification binding is invalid")
	}
	if err := reportilcontract.ValidateSourceCatalog(catalog); err != nil {
		return reportilcontract.SourceReadReceipt{}, err
	}
	entries := make(map[string]reportilcontract.SourceCatalogEntry, len(catalog.Sources))
	catalogIndexes := make(map[string]int, len(catalog.Sources))
	for index, entry := range catalog.Sources {
		entries[entry.SourceKey] = entry
		catalogIndexes[entry.SourceKey] = index
	}
	events, err := s.ListEvents(ctx, missionID)
	if err != nil {
		return reportilcontract.SourceReadReceipt{}, err
	}
	type sourceReadMetrics struct {
		SourceKey            string `json:"source_key"`
		ReturnedOffset       int    `json:"returned_offset"`
		ReturnedContentBytes int    `json:"returned_content_bytes"`
		ContentLength        int    `json:"content_length"`
		ResponseTruncated    *bool  `json:"response_truncated"`
		NextOffset           int    `json:"next_offset"`
	}
	type sourceReadPayload struct {
		ToolName      string `json:"tool_name"`
		ToolSessionID string `json:"tool_session_id"`
		Success       bool   `json:"success"`
		IOMetrics     struct {
			SourceKey            string              `json:"source_key"`
			CatalogSHA256        string              `json:"catalog_sha256"`
			ReportILStage        string              `json:"report_il_stage"`
			ReturnedOffset       int                 `json:"returned_offset"`
			ReturnedContentBytes int                 `json:"returned_content_bytes"`
			ContentLength        int                 `json:"content_length"`
			ResponseTruncated    *bool               `json:"response_truncated"`
			NextOffset           int                 `json:"next_offset"`
			SourceReads          []sourceReadMetrics `json:"source_reads"`
			SourceReceipt        string              `json:"source_receipt"`
			SourceOffset         int                 `json:"source_offset"`
			SourceByteSize       int                 `json:"source_byte_size"`
			SourceSHA256         string              `json:"source_sha256"`
		} `json:"io_metrics"`
	}

	serverOrdered := expectedStage == "il_source_selection" ||
		expectedStage == "il_editorial_memory" ||
		expectedStage == "il_continuity" ||
		expectedStage == "il_reader" ||
		expectedStage == "il_flow"
	nextOffsets := make(map[string]int, len(catalog.Sources))
	completed := make(map[string]bool, len(catalog.Sources))
	seen := make(map[string]bool, len(catalog.Sources))
	nextCatalogIndex := 0
	receipt := reportilcontract.SourceReadReceipt{
		ReadBytesBySource: map[string]int{},
		SourceQuotes:      map[string]reportilcontract.SourceQuoteReceipt{},
	}
	for _, event := range events {
		if event.EventType != "mcp.tool.called" || event.CorrelationID != toolSessionID {
			continue
		}
		var payload sourceReadPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return reportilcontract.SourceReadReceipt{}, fmt.Errorf("report IL source read trace is malformed")
		}
		if payload.ToolName != reportilcontract.SourceReadTool &&
			payload.ToolName != reportilcontract.SourceQuoteRegisterTool {
			continue
		}
		if payload.ToolSessionID != toolSessionID {
			return reportilcontract.SourceReadReceipt{}, fmt.Errorf("report IL source read trace session is invalid")
		}
		if !payload.Success {
			continue
		}
		if payload.IOMetrics.CatalogSHA256 != catalog.SHA256 ||
			payload.IOMetrics.ReportILStage != expectedStage {
			return reportilcontract.SourceReadReceipt{}, fmt.Errorf("report IL source read trace stage binding is invalid")
		}
		if payload.ToolName == reportilcontract.SourceQuoteRegisterTool {
			entry, ok := entries[payload.IOMetrics.SourceKey]
			quote := reportilcontract.SourceQuoteReceipt{
				Receipt: payload.IOMetrics.SourceReceipt, SourceKey: payload.IOMetrics.SourceKey,
				Offset: payload.IOMetrics.SourceOffset, ByteSize: payload.IOMetrics.SourceByteSize,
				SHA256: payload.IOMetrics.SourceSHA256,
			}
			wantReceipt := reportilcontract.SourceQuoteReceiptID(
				toolSessionID, catalog.SHA256, expectedStage,
				len(receipt.SourceQuotes)+1, quote.SourceKey, quote.Offset,
				quote.ByteSize, quote.SHA256,
			)
			allSourcesRead := len(completed) == len(catalog.Sources)
			if (expectedStage != "il_narrative" && expectedStage != "il_editorial_memory") || !ok || !allSourcesRead ||
				quote.Offset < 0 || quote.ByteSize < 1 || quote.ByteSize > reportilcontract.MaxSourceQuoteBytes ||
				quote.Offset > entry.ReadableBytes-quote.ByteSize || quote.Receipt != wantReceipt ||
				receipt.SourceQuotes[quote.Receipt].Receipt != "" {
				return reportilcontract.SourceReadReceipt{}, fmt.Errorf("report IL source quote trace is invalid")
			}
			receipt.SourceQuotes[quote.Receipt] = quote
			continue
		}
		batch := len(payload.IOMetrics.SourceReads) > 0
		if batch != serverOrdered {
			return reportilcontract.SourceReadReceipt{}, fmt.Errorf("report IL source read trace shape is invalid for the stage")
		}
		reads := payload.IOMetrics.SourceReads
		if !batch {
			reads = []sourceReadMetrics{{
				SourceKey:            payload.IOMetrics.SourceKey,
				ReturnedOffset:       payload.IOMetrics.ReturnedOffset,
				ReturnedContentBytes: payload.IOMetrics.ReturnedContentBytes,
				ContentLength:        payload.IOMetrics.ContentLength,
				ResponseTruncated:    payload.IOMetrics.ResponseTruncated,
				NextOffset:           payload.IOMetrics.NextOffset,
			}}
		}
		seenEventKeys := make(map[string]bool, len(reads))
		batchBytes := 0
		for _, read := range reads {
			entry, sourceBound := entries[read.SourceKey]
			end := read.ReturnedOffset + read.ReturnedContentBytes
			if (expectedStage != "il_flow" && seenEventKeys[read.SourceKey]) ||
				!sourceBound || entry.ReadableBytes < 1 ||
				read.ContentLength != entry.ReadableBytes || read.ResponseTruncated == nil ||
				read.ReturnedOffset < 0 || read.ReturnedOffset >= entry.ReadableBytes ||
				read.ReturnedContentBytes < 1 ||
				read.ReturnedContentBytes > entry.ReadableBytes-read.ReturnedOffset ||
				(*read.ResponseTruncated && (read.NextOffset != end || end >= entry.ReadableBytes)) ||
				(!*read.ResponseTruncated && (read.NextOffset != 0 || end != entry.ReadableBytes)) {
				return reportilcontract.SourceReadReceipt{}, fmt.Errorf("report IL source read trace continuation is invalid")
			}
			seenEventKeys[read.SourceKey] = true
			batchBytes += read.ReturnedContentBytes

			sourceIndex := catalogIndexes[read.SourceKey]
			switch expectedStage {
			case "il_source_selection":
				if read.ReturnedOffset != 0 || completed[read.SourceKey] || sourceIndex != nextCatalogIndex {
					return reportilcontract.SourceReadReceipt{}, fmt.Errorf("report IL source selection trace order is invalid")
				}
				completed[read.SourceKey] = true
				nextCatalogIndex++
			case "il_editorial_memory", "il_continuity", "il_reader":
				if completed[read.SourceKey] || sourceIndex != nextCatalogIndex || read.ReturnedOffset != nextOffsets[read.SourceKey] {
					return reportilcontract.SourceReadReceipt{}, fmt.Errorf("report IL complete source trace order is invalid")
				}
				if *read.ResponseTruncated {
					nextOffsets[read.SourceKey] = read.NextOffset
				} else {
					completed[read.SourceKey] = true
					delete(nextOffsets, read.SourceKey)
					nextCatalogIndex++
				}
			case "il_narrative":
				if completed[read.SourceKey] || read.ReturnedOffset != nextOffsets[read.SourceKey] {
					return reportilcontract.SourceReadReceipt{}, fmt.Errorf("report IL direct author source trace order is invalid")
				}
				if *read.ResponseTruncated {
					nextOffsets[read.SourceKey] = read.NextOffset
				} else {
					completed[read.SourceKey] = true
					delete(nextOffsets, read.SourceKey)
				}
			case "il_flow":
				// Flow reads server-bound compact spans that may start at nonzero offsets
				// and may revisit one source through several non-overlapping excerpts.
				// The product compiler compares the exact returned ranges with its
				// request-local evidence packets after this content-free trace is built.
			default:
				if completed[read.SourceKey] || read.ReturnedOffset != nextOffsets[read.SourceKey] {
					return reportilcontract.SourceReadReceipt{}, fmt.Errorf("report IL source trace order is invalid")
				}
				if *read.ResponseTruncated {
					nextOffsets[read.SourceKey] = read.NextOffset
				} else {
					completed[read.SourceKey] = true
					delete(nextOffsets, read.SourceKey)
				}
			}
			if !seen[read.SourceKey] {
				seen[read.SourceKey] = true
				receipt.SourceKeys = append(receipt.SourceKeys, read.SourceKey)
			}
			receipt.ReadBytesBySource[read.SourceKey] += read.ReturnedContentBytes
			receipt.ReturnedContentBytes += read.ReturnedContentBytes
			receipt.ReadRanges = append(receipt.ReadRanges, reportilcontract.SourceReadRange{
				SourceKey: read.SourceKey, Offset: read.ReturnedOffset,
				ByteSize: read.ReturnedContentBytes,
			})
		}
		if batch && (batchBytes != payload.IOMetrics.ReturnedContentBytes ||
			batchBytes > reportilcontract.DefaultSourceReadMaxBytes ||
			len(reads) > reportilcontract.MaxSourceReadSpans) {
			return reportilcontract.SourceReadReceipt{}, fmt.Errorf("report IL source batch trace aggregate is invalid")
		}
	}
	for _, entry := range catalog.Sources {
		if receipt.ReadBytesBySource[entry.SourceKey] == entry.ReadableBytes {
			receipt.FullyReadSourceKeys = append(receipt.FullyReadSourceKeys, entry.SourceKey)
		}
	}
	if err := receipt.Validate(catalog); err != nil {
		return reportilcontract.SourceReadReceipt{}, err
	}
	return receipt, nil
}

func validReportILSourceReadStage(stage string) bool {
	switch stage {
	case "il_source_selection", "il_editorial_memory", "il_narrative", "il_continuity", "il_reader", "il_document", "il_flow":
		return true
	default:
		return false
	}
}

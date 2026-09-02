package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

type reportILSourceTraceStore struct {
	fakeStore
	events []LedgerEvent
}

func (store reportILSourceTraceStore) ListLedgerEvents(context.Context, string) ([]LedgerEvent, error) {
	return append([]LedgerEvent(nil), store.events...), nil
}

func TestVerifyReportILSourceReadAcceptsCompleteFlowBatches(t *testing.T) {
	catalog := reportILSourceVerifierCatalog(t, 17, 5)
	service := NewService(reportILSourceTraceStore{events: []LedgerEvent{
		reportILSourceVerifierBatchEvent(
			"evt_flow_1", "ses_flow", "il_flow", catalog.SHA256,
			[]map[string]any{
				reportILSourceVerifierRead("source_001", 0, 8, 17, true, 8),
			}, true,
		),
		reportILSourceVerifierBatchEvent(
			"evt_flow_2", "ses_flow", "il_flow", catalog.SHA256,
			[]map[string]any{
				reportILSourceVerifierRead("source_001", 8, 9, 17, false, 0),
				reportILSourceVerifierRead("source_002", 0, 5, 5, false, 0),
			}, true,
		),
	}})
	receipt, err := service.VerifyReportILSourceRead(
		context.Background(), catalog.MissionID, "ses_flow", "il_flow", catalog,
	)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ReturnedContentBytes != 22 ||
		!receipt.IncludesEntireSource("source_001") ||
		!receipt.IncludesEntireSource("source_002") {
		t.Fatalf("flow source read receipt = %#v", receipt)
	}
}

func TestVerifyReportILSourceReadRecordsCompactFlowRanges(t *testing.T) {
	catalog := reportILSourceVerifierCatalog(t, 40)
	service := NewService(reportILSourceTraceStore{events: []LedgerEvent{
		reportILSourceVerifierBatchEvent(
			"evt_flow_span_1", "ses_flow_span", "il_flow", catalog.SHA256,
			[]map[string]any{
				reportILSourceVerifierRead("source_001", 5, 7, 40, true, 12),
				reportILSourceVerifierRead("source_001", 20, 8, 40, true, 28),
			}, true,
		),
	}})
	receipt, err := service.VerifyReportILSourceRead(
		context.Background(), catalog.MissionID, "ses_flow_span", "il_flow", catalog,
	)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ReturnedContentBytes != 15 || receipt.IncludesEntireSource("source_001") ||
		len(receipt.ReadRanges) != 2 || receipt.ReadRanges[0].Offset != 5 ||
		receipt.ReadRanges[0].ByteSize != 7 || receipt.ReadRanges[1].Offset != 20 ||
		receipt.ReadRanges[1].ByteSize != 8 {
		t.Fatalf("compact flow read ranges = %#v", receipt)
	}
}

func TestVerifyReportILSourceReadAcceptsSelectionBatches(t *testing.T) {
	catalog := reportILSourceVerifierCatalog(t, 17, 19)
	service := NewService(reportILSourceTraceStore{events: []LedgerEvent{
		reportILSourceVerifierBatchEvent(
			"evt_selection",
			"ses_selection",
			"il_source_selection",
			catalog.SHA256,
			[]map[string]any{
				reportILSourceVerifierRead("source_001", 0, 8, 17, true, 8),
				reportILSourceVerifierRead("source_002", 0, 8, 19, true, 8),
			},
			true,
		),
	}})
	receipt, err := service.VerifyReportILSourceRead(
		context.Background(),
		catalog.MissionID,
		"ses_selection",
		"il_source_selection",
		catalog,
	)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ReturnedContentBytes != 16 ||
		receipt.ReadBytesBySource["source_001"] != 8 ||
		receipt.ReadBytesBySource["source_002"] != 8 ||
		len(receipt.FullyReadSourceKeys) != 0 {
		t.Fatalf("selection source read receipt = %#v", receipt)
	}
}

func TestVerifyReportILSourceReadAcceptsCompleteBatchContinuation(t *testing.T) {
	for _, stage := range []string{"il_continuity", "il_reader"} {
		t.Run(stage, func(t *testing.T) {
			catalog := reportILSourceVerifierCatalog(t, 17, 5)
			sessionID := "ses_" + strings.TrimPrefix(stage, "il_")
			service := NewService(reportILSourceTraceStore{events: []LedgerEvent{
				reportILSourceVerifierBatchEvent(
					"evt_"+stage+"_1", sessionID, stage, catalog.SHA256,
					[]map[string]any{
						reportILSourceVerifierRead("source_001", 0, 8, 17, true, 8),
					}, true,
				),
				reportILSourceVerifierBatchEvent(
					"evt_"+stage+"_2", sessionID, stage, catalog.SHA256,
					[]map[string]any{
						reportILSourceVerifierRead("source_001", 8, 9, 17, false, 0),
						reportILSourceVerifierRead("source_002", 0, 5, 5, false, 0),
					}, true,
				),
			}})
			receipt, err := service.VerifyReportILSourceRead(
				context.Background(), catalog.MissionID, sessionID, stage, catalog,
			)
			if err != nil {
				t.Fatal(err)
			}
			if receipt.ReturnedContentBytes != 22 ||
				!receipt.IncludesEntireSource("source_001") ||
				!receipt.IncludesEntireSource("source_002") {
				t.Fatalf("%s continuation receipt = %#v", stage, receipt)
			}
		})
	}
}

func TestVerifyReportILSourceReadAcceptsBoundQuoteReceipts(t *testing.T) {
	for _, stage := range []string{"il_narrative", "il_editorial_memory"} {
		t.Run(stage, func(t *testing.T) {
			catalog := reportILSourceVerifierCatalog(t, 17)
			sessionID := "ses_quote_" + stage
			quoteSHA := strings.Repeat("a", 64)
			quoteReceipt := reportilcontract.SourceQuoteReceiptID(
				sessionID, catalog.SHA256, stage, 1, "source_001", 4, 5, quoteSHA,
			)
			read := reportILSourceVerifierBatchEvent(
				"evt_quote_read", sessionID, stage, catalog.SHA256,
				[]map[string]any{reportILSourceVerifierRead("source_001", 0, 17, 17, false, 0)}, true,
			)
			if stage == "il_narrative" {
				read = reportILSourceVerifierEvent(
					"evt_quote_read", sessionID, reportilcontract.SourceReadTool,
					stage, "source_001", catalog.SHA256, 0, 17, 17, false, 0, true,
				)
			}
			quote := reportILSourceVerifierQuoteEvent(
				"evt_quote", sessionID, stage, catalog.SHA256,
				quoteReceipt, "source_001", 4, 5, quoteSHA,
			)
			service := NewService(reportILSourceTraceStore{events: []LedgerEvent{read, quote}})
			receipt, err := service.VerifyReportILSourceRead(
				context.Background(), catalog.MissionID, sessionID, stage, catalog,
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(receipt.SourceQuotes) != 1 || receipt.SourceQuotes[quoteReceipt].SHA256 != quoteSHA {
				t.Fatalf("source quote receipt = %#v", receipt.SourceQuotes)
			}

			quoteBeforeReadService := NewService(reportILSourceTraceStore{events: []LedgerEvent{quote, read}})
			if _, err := quoteBeforeReadService.VerifyReportILSourceRead(
				context.Background(), catalog.MissionID, sessionID, stage, catalog,
			); err == nil {
				t.Fatal("source quote trace before complete catalog read was accepted")
			}

			for name, mutate := range map[string]func(*LedgerEvent){
				"forged receipt": func(event *LedgerEvent) {
					var payload map[string]any
					_ = json.Unmarshal(event.Payload, &payload)
					payload["io_metrics"].(map[string]any)["source_receipt"] = "quote_" + strings.Repeat("f", 64)
					event.Payload, _ = json.Marshal(payload)
				},
				"wrong stage": func(event *LedgerEvent) {
					var payload map[string]any
					_ = json.Unmarshal(event.Payload, &payload)
					payload["io_metrics"].(map[string]any)["report_il_stage"] = "il_flow"
					event.Payload, _ = json.Marshal(payload)
				},
				"wrong source": func(event *LedgerEvent) {
					var payload map[string]any
					_ = json.Unmarshal(event.Payload, &payload)
					payload["io_metrics"].(map[string]any)["source_key"] = "source_999"
					event.Payload, _ = json.Marshal(payload)
				},
			} {
				t.Run(name, func(t *testing.T) {
					invalidQuote := quote
					mutate(&invalidQuote)
					invalidService := NewService(reportILSourceTraceStore{events: []LedgerEvent{read, invalidQuote}})
					if _, err := invalidService.VerifyReportILSourceRead(
						context.Background(), catalog.MissionID, sessionID, stage, catalog,
					); err == nil {
						t.Fatal("invalid source quote trace was accepted")
					}
				})
			}
		})
	}
}

func TestVerifyReportILSourceReadIgnoresFailedExtraRead(t *testing.T) {
	catalog := reportILSourceVerifierCatalog(t, 17)
	valid := reportILSourceVerifierBatchEvent(
		"evt_valid", "ses_exact", "il_flow", catalog.SHA256,
		[]map[string]any{reportILSourceVerifierRead("source_001", 0, 17, 17, false, 0)}, true,
	)
	failed := reportILSourceVerifierEvent(
		"evt_failed", "ses_exact", reportilcontract.SourceReadTool, "",
		"source_001", "", 17, 0, 17, false, 0, false,
	)
	service := NewService(reportILSourceTraceStore{events: []LedgerEvent{valid, failed}})
	receipt, err := service.VerifyReportILSourceRead(
		context.Background(), catalog.MissionID, "ses_exact", "il_flow", catalog,
	)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ReturnedContentBytes != 17 || !receipt.IncludesEntireSource("source_001") {
		t.Fatalf("source read receipt = %#v", receipt)
	}
}

func TestVerifyReportILSourceReadRejectsMalformedStageAndContinuation(t *testing.T) {
	catalog := reportILSourceVerifierCatalog(t, 17, 5)
	validNarrative := func(eventID, sourceKey string, offset, returnedBytes, contentLength int, truncated bool, nextOffset int) LedgerEvent {
		return reportILSourceVerifierEvent(
			eventID, "ses_trace", reportilcontract.SourceReadTool, "il_narrative",
			sourceKey, catalog.SHA256, offset, returnedBytes, contentLength, truncated,
			nextOffset, true,
		)
	}
	cases := []struct {
		name          string
		expectedStage string
		events        []LedgerEvent
	}{
		{
			name: "missing stage", expectedStage: "il_narrative",
			events: []LedgerEvent{reportILSourceVerifierBatchEvent(
				"evt_trace", "ses_trace", "", catalog.SHA256,
				[]map[string]any{reportILSourceVerifierRead("source_001", 0, 17, 17, false, 0)}, true,
			)},
		},
		{
			name: "wrong stage", expectedStage: "il_narrative",
			events: []LedgerEvent{reportILSourceVerifierBatchEvent(
				"evt_trace", "ses_trace", "il_source_selection", catalog.SHA256,
				[]map[string]any{reportILSourceVerifierRead("source_001", 0, 17, 17, false, 0)}, true,
			)},
		},
		{
			name: "missing truncation", expectedStage: "il_narrative",
			events: []LedgerEvent{reportILSourceVerifierBatchEvent(
				"evt_trace", "ses_trace", "il_narrative", catalog.SHA256,
				[]map[string]any{{
					"source_key": "source_001", "returned_offset": 0,
					"returned_content_bytes": 17, "content_length": 17,
				}}, true,
			)},
		},
		{
			name: "truncated next offset mismatch", expectedStage: "il_narrative",
			events: []LedgerEvent{validNarrative("evt_trace", "source_001", 0, 8, 17, true, 7)},
		},
		{
			name: "truncated at eof", expectedStage: "il_narrative",
			events: []LedgerEvent{validNarrative("evt_trace", "source_001", 0, 17, 17, true, 17)},
		},
		{
			name: "false eof followed by continuation", expectedStage: "il_narrative",
			events: []LedgerEvent{
				validNarrative("evt_trace_1", "source_001", 0, 8, 17, false, 0),
				validNarrative("evt_trace_2", "source_001", 8, 9, 17, false, 0),
			},
		},
		{
			name: "continuation gap", expectedStage: "il_narrative",
			events: []LedgerEvent{
				validNarrative("evt_trace_1", "source_001", 0, 8, 17, true, 8),
				validNarrative("evt_trace_2", "source_001", 9, 8, 17, false, 0),
			},
		},
		{
			name: "batch read for keyed stage", expectedStage: "il_narrative",
			events: []LedgerEvent{reportILSourceVerifierBatchEvent(
				"evt_trace", "ses_trace", "il_narrative", catalog.SHA256,
				[]map[string]any{reportILSourceVerifierRead("source_001", 0, 17, 17, false, 0)}, true,
			)},
		},
		{
			name: "single read for flow batch stage", expectedStage: "il_flow",
			events: []LedgerEvent{reportILSourceVerifierEvent(
				"evt_trace", "ses_trace", reportilcontract.SourceReadTool, "il_flow",
				"source_001", catalog.SHA256, 0, 17, 17, false, 0, true,
			)},
		},
		{
			name: "invalid expected stage", expectedStage: "il_unknown",
			events: []LedgerEvent{validNarrative("evt_trace", "source_001", 0, 17, 17, false, 0)},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService(reportILSourceTraceStore{events: tc.events})
			if _, err := service.VerifyReportILSourceRead(
				context.Background(), catalog.MissionID, "ses_trace", tc.expectedStage, catalog,
			); err == nil {
				t.Fatal("invalid source read trace was accepted")
			}
		})
	}
}

func TestVerifyReportILSourceReadRejectsMissingOrMismatchedTrace(t *testing.T) {
	catalog := reportILSourceVerifierCatalog(t, 17)
	valid := func(
		sessionID,
		toolName,
		sourceKey,
		catalogSHA string,
		offset,
		returnedBytes,
		contentLength int,
		success bool,
	) []LedgerEvent {
		if toolName == reportilcontract.SourceReadTool {
			return []LedgerEvent{reportILSourceVerifierBatchEvent(
				"evt_trace", sessionID, "il_flow", catalogSHA,
				[]map[string]any{reportILSourceVerifierRead(
					sourceKey, offset, returnedBytes, contentLength, false, 0,
				)}, success,
			)}
		}
		return []LedgerEvent{reportILSourceVerifierEvent(
			"evt_trace", sessionID, toolName, "il_flow", sourceKey, catalogSHA,
			offset, returnedBytes, contentLength, false, 0, success,
		)}
	}
	cases := []struct {
		name   string
		events []LedgerEvent
	}{
		{name: "missing"},
		{name: "wrong session", events: valid("ses_other", reportilcontract.SourceReadTool, "source_001", catalog.SHA256, 0, 17, 17, true)},
		{name: "wrong tool", events: valid("ses_exact", reportilcontract.SourceListTool, "source_001", catalog.SHA256, 0, 17, 17, true)},
		{name: "wrong source", events: valid("ses_exact", reportilcontract.SourceReadTool, "source_999", catalog.SHA256, 0, 17, 17, true)},
		{name: "wrong catalog", events: valid("ses_exact", reportilcontract.SourceReadTool, "source_001", strings.Repeat("f", 64), 0, 17, 17, true)},
		{name: "wrong content length", events: valid("ses_exact", reportilcontract.SourceReadTool, "source_001", catalog.SHA256, 0, 17, 18, true)},
		{name: "negative offset", events: valid("ses_exact", reportilcontract.SourceReadTool, "source_001", catalog.SHA256, -1, 17, 17, true)},
		{name: "out of range span", events: valid("ses_exact", reportilcontract.SourceReadTool, "source_001", catalog.SHA256, 1, 17, 17, true)},
		{name: "zero bytes", events: valid("ses_exact", reportilcontract.SourceReadTool, "source_001", catalog.SHA256, 0, 0, 17, true)},
		{name: "failed", events: valid("ses_exact", reportilcontract.SourceReadTool, "source_001", catalog.SHA256, 0, 17, 17, false)},
		{name: "malformed", events: []LedgerEvent{{EventID: "evt_trace", MissionID: catalog.MissionID, EventType: "mcp.tool.called", CorrelationID: "ses_exact", Payload: json.RawMessage(`{"broken":`)}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService(reportILSourceTraceStore{events: tc.events})
			if _, err := service.VerifyReportILSourceRead(
				context.Background(), catalog.MissionID, "ses_exact", "il_flow", catalog,
			); err == nil {
				t.Fatal("invalid source read trace was accepted")
			}
		})
	}
}

func reportILSourceVerifierCatalog(t *testing.T, sizes ...int) reportilcontract.SourceCatalog {
	t.Helper()
	entries := make([]reportilcontract.SourceCatalogEntry, 0, len(sizes))
	for index, size := range sizes {
		sourceKey := fmt.Sprintf("source_%03d", index+1)
		snapshotID := fmt.Sprintf("src_trace_%03d", index+1)
		artifactID := fmt.Sprintf("art_trace_%03d", index+1)
		contentHash := strings.Repeat(string(rune('a'+index)), 64)
		entries = append(entries, reportilcontract.SourceCatalogEntry{
			SourceKey:       sourceKey,
			SnapshotID:      snapshotID,
			SnapshotReceipt: reportilcontract.SourceSnapshotReceipt(snapshotID, contentHash),
			ContentHash:     contentHash,
			RetrievalPolicy: SourceRetrievalPolicySnapshotOnly,
			Artifacts: []reportilcontract.SourceCatalogArtifact{{
				ArtifactID: artifactID, SHA256: contentHash, ByteSize: int64(size), MediaType: "text/plain",
			}},
			ReadableSHA256: strings.Repeat(string(rune('f'-index)), 64),
			ReadableBytes:  size,
			Extraction:     "stored_text",
		})
	}
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: "mis_trace",
		Sources:   entries,
	})
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func reportILSourceVerifierRead(sourceKey string, offset, returnedBytes, contentLength int, truncated bool, nextOffset int) map[string]any {
	return map[string]any{
		"source_key":             sourceKey,
		"returned_offset":        offset,
		"returned_content_bytes": returnedBytes,
		"content_length":         contentLength,
		"response_truncated":     truncated,
		"next_offset":            nextOffset,
	}
}

func reportILSourceVerifierBatchEvent(eventID, sessionID, stage, catalogSHA string, reads []map[string]any, success bool) LedgerEvent {
	returnedBytes := 0
	for _, read := range reads {
		returnedBytes += read["returned_content_bytes"].(int)
	}
	payload, _ := json.Marshal(map[string]any{
		"tool_name": reportilcontract.SourceReadTool, "tool_session_id": sessionID, "success": success,
		"io_metrics": map[string]any{
			"catalog_sha256": catalogSHA, "report_il_stage": stage,
			"returned_content_bytes": returnedBytes, "source_reads": reads,
		},
	})
	return LedgerEvent{
		EventID: eventID, MissionID: "mis_trace", EventType: "mcp.tool.called",
		CorrelationID: sessionID, Payload: payload,
	}
}

func reportILSourceVerifierQuoteEvent(
	eventID,
	sessionID,
	stage,
	catalogSHA,
	sourceReceipt,
	sourceKey string,
	offset,
	byteSize int,
	quoteSHA string,
) LedgerEvent {
	payload, _ := json.Marshal(map[string]any{
		"tool_name":       reportilcontract.SourceQuoteRegisterTool,
		"tool_session_id": sessionID,
		"success":         true,
		"io_metrics": map[string]any{
			"source_receipt":   sourceReceipt,
			"source_key":       sourceKey,
			"source_offset":    offset,
			"source_byte_size": byteSize,
			"source_sha256":    quoteSHA,
			"catalog_sha256":   catalogSHA,
			"report_il_stage":  stage,
		},
	})
	return LedgerEvent{
		EventID: eventID, MissionID: "mis_trace", EventType: "mcp.tool.called",
		CorrelationID: sessionID, Payload: payload,
	}
}

func reportILSourceVerifierEvent(
	eventID,
	sessionID,
	toolName,
	stage,
	sourceKey,
	catalogSHA string,
	returnedOffset,
	returnedBytes,
	contentLength int,
	truncated bool,
	nextOffset int,
	success bool,
) LedgerEvent {
	payload, _ := json.Marshal(map[string]any{
		"tool_name": toolName, "tool_session_id": sessionID, "success": success,
		"io_metrics": map[string]any{
			"source_key":             sourceKey,
			"catalog_sha256":         catalogSHA,
			"report_il_stage":        stage,
			"returned_offset":        returnedOffset,
			"returned_content_bytes": returnedBytes,
			"content_length":         contentLength,
			"response_truncated":     truncated,
			"next_offset":            nextOffset,
		},
	})
	return LedgerEvent{
		EventID: eventID, MissionID: "mis_trace", EventType: "mcp.tool.called",
		CorrelationID: sessionID, Payload: payload,
	}
}
